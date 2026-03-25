package myhandlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/config"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
)

// AuthStore is set from main (session + user credential persistence).
var AuthStore *authstore.Store

func RegisterUser(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil || Database == nil || Database.Db == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		var body struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
			Name     string `json:"name" binding:"required,min=1,max=100"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid request body", map[string]any{"field": "body"})
			return
		}
		email, err := normalizeEmail(body.Email)
		if err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid email", map[string]any{"field": "email"})
			return
		}
		if err := auth.ValidatePasswordLength(body.Password); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "password does not meet policy", map[string]any{"field": "password", "min_len": 8})
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "name is required", map[string]any{"field": "name"})
			return
		}
		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "password does not meet policy", nil)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		id, err := AuthStore.CreateRegisteredUser(ctx, email, name, hash)
		if err != nil {
			var me *mysql.MySQLError
			if errors.As(err, &me) {
				slog.Error("auth_register_db_error",
					"request_id", requestid.FromContext(c),
					"mysql_errno", me.Number,
					"err", err.Error(),
				)
				if me.Number == 1062 {
					respond.Error(c, http.StatusConflict, api.CodeConflict, "email already registered", map[string]any{"field": "email"})
					return
				}
				// Unknown column / missing table: schema behind migrations (common if migrate never ran or /readyz check disabled).
				if me.Number == 1054 || me.Number == 1146 {
					respond.Error(c, http.StatusInternalServerError, api.CodeInternal,
						"database schema is missing auth columns or tables; run migrations and set MIGRATION_EXPECTED_VERSION (see README)", nil)
					return
				}
			} else {
				slog.Error("auth_register_failed", "request_id", requestid.FromContext(c), "err", err.Error())
			}
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to register user", nil)
			return
		}
		_ = AuthStore.InsertAudit(ctx, "auth_register_success", intPtr(int(id)), clientIP(c), c.Request.UserAgent(), email, nil)
		respond.JSONOK(c, http.StatusCreated, gin.H{
			"id":    id,
			"email": email,
			"name":  name,
		})
	}
}

func LoginUser(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil || Database == nil || Database.Db == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		var body struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid request body", map[string]any{"field": "body"})
			return
		}
		email, err := normalizeEmail(body.Email)
		if err != nil {
			authLoginFail(c, nil, email, "invalid_email")
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid email or password", nil)
			return
		}
		if locked, retry := loginEmailLocked(c.Request.Context(), email); locked {
			sec := int(retry.Seconds()) + 1
			if sec < 1 {
				sec = 1
			}
			respond.Error(c, http.StatusTooManyRequests, api.CodeRateLimited, "too many login attempts; try again later", map[string]any{"retry_after_seconds": sec})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		uid, hash, err := AuthStore.GetUserAuthByEmail(ctx, email)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "login failed", nil)
			return
		}
		if errors.Is(err, sql.ErrNoRows) || strings.TrimSpace(hash) == "" {
			authLoginFail(c, nil, email, "unknown_user_or_no_password")
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid email or password", nil)
			return
		}
		ok, err := auth.VerifyPassword(body.Password, hash)
		if err != nil || !ok {
			recordLoginFailureKnownUser(c.Request.Context(), email)
			authLoginFail(c, &uid, email, "bad_password")
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid email or password", nil)
			return
		}
		recordLoginSuccessClearThrottle(c.Request.Context(), email)
		raw, err := authstore.NewOpaqueToken()
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "login failed", nil)
			return
		}
		if err := AuthStore.CreateSession(ctx, uid, raw, clientIP(c), c.Request.UserAgent()); err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "login failed", nil)
			return
		}
		_ = AuthStore.InsertAudit(ctx, "auth_login_success", &uid, clientIP(c), c.Request.UserAgent(), email, nil)
		setSessionCookie(c, cfg, raw)
		csrfTok, err := authstore.NewOpaqueToken()
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "login failed", nil)
			return
		}
		setCSRFCookie(c, cfg, csrfTok)
		respond.JSONOK(c, http.StatusOK, gin.H{
			"user_id": uid,
			"email":   email,
		})
	}
}

func LogoutUser(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		raw, err := c.Cookie(cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(raw) == "" {
			clearSessionCookie(c, cfg)
			clearCSRFCookie(c, cfg)
			respond.JSONOK(c, http.StatusOK, gin.H{"logged_out": true})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		_ = AuthStore.RevokeSessionByRawToken(ctx, raw)
		_ = AuthStore.InsertAudit(ctx, "auth_logout", nil, clientIP(c), c.Request.UserAgent(), "", nil)
		clearSessionCookie(c, cfg)
		clearCSRFCookie(c, cfg)
		respond.JSONOK(c, http.StatusOK, gin.H{"logged_out": true})
	}
}

func RefreshSession(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		raw, err := c.Cookie(cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(raw) == "" {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "missing session", nil)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		uid, err := AuthStore.RefreshSession(ctx, raw)
		if err != nil {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid or expired session", nil)
			return
		}
		_ = AuthStore.InsertAudit(ctx, "auth_refresh", &uid, clientIP(c), c.Request.UserAgent(), "", nil)
		// Re-send cookie so Max-Age tracks remaining absolute window (browser hint only).
		setSessionCookie(c, cfg, raw)
		csrfTok, err := authstore.NewOpaqueToken()
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "refresh failed", nil)
			return
		}
		setCSRFCookie(c, cfg, csrfTok)
		respond.JSONOK(c, http.StatusOK, gin.H{"refreshed": true, "user_id": uid})
	}
}

func authLoginFail(c *gin.Context, uid *int, email, reason string) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if AuthStore != nil {
		meta, _ := json.Marshal(map[string]string{"reason": reason})
		_ = AuthStore.InsertAudit(ctx, "auth_login_failure", uid, clientIP(c), c.Request.UserAgent(), email, meta)
	}
	slog.Info("auth_login_failure",
		"request_id", requestid.FromContext(c),
		"reason", reason,
		"email_domain", emailDomainHint(email),
	)
}

func normalizeEmail(s string) (string, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "", errors.New("empty")
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", err
	}
	if addr.Name != "" || strings.TrimSpace(addr.Address) == "" {
		return "", errors.New("invalid")
	}
	return strings.ToLower(strings.TrimSpace(addr.Address)), nil
}

func emailDomainHint(email string) string {
	i := strings.LastIndex(email, "@")
	if i < 0 || i >= len(email)-1 {
		return ""
	}
	return email[i+1:]
}

func clientIP(c *gin.Context) string {
	return strings.TrimSpace(c.ClientIP())
}

func intPtr(v int) *int {
	return &v
}

func setSessionCookie(c *gin.Context, cfg *config.Config, raw string) {
	abs := time.Now().UTC().Add(cfg.SessionAbsoluteTTL)
	maxAge := int(time.Until(abs).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetSameSite(cfg.SessionSameSiteMode)
	c.SetCookie(cfg.SessionCookieName, raw, maxAge, "/", "", cfg.SessionCookieSecure, true)
}

func clearSessionCookie(c *gin.Context, cfg *config.Config) {
	c.SetSameSite(cfg.SessionSameSiteMode)
	c.SetCookie(cfg.SessionCookieName, "", -1, "/", "", cfg.SessionCookieSecure, true)
}

// setCSRFCookie sets the double-submit CSRF token (not HttpOnly; SPA reads cookie and echoes header).
func setCSRFCookie(c *gin.Context, cfg *config.Config, raw string) {
	abs := time.Now().UTC().Add(cfg.SessionAbsoluteTTL)
	maxAge := int(time.Until(abs).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetSameSite(cfg.SessionSameSiteMode)
	c.SetCookie(cfg.CSRFCookieName, raw, maxAge, "/", "", cfg.SessionCookieSecure, false)
}

func clearCSRFCookie(c *gin.Context, cfg *config.Config) {
	c.SetSameSite(cfg.SessionSameSiteMode)
	c.SetCookie(cfg.CSRFCookieName, "", -1, "/", "", cfg.SessionCookieSecure, false)
}

// GetAuthCSRF issues or rotates the CSRF cookie for an existing valid session (GET; no CSRF header required).
func GetAuthCSRF(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		raw, err := c.Cookie(cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(raw) == "" {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "missing session", nil)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		_, err = AuthStore.ValidateSession(ctx, raw)
		if err != nil {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid or expired session", nil)
			return
		}
		tok, err := authstore.NewOpaqueToken()
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to issue csrf", nil)
			return
		}
		setCSRFCookie(c, cfg, tok)
		respond.JSONOK(c, http.StatusOK, gin.H{"csrf_ready": true})
	}
}
