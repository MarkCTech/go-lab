package myhandlers

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/config"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

// TokenSvc is set from main after the auth service is constructed.
var TokenSvc *auth.TokenService

// IssueToken handles POST /api/v1/auth/token (client_credentials).
func IssueToken(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			GrantType    string `json:"grant_type" binding:"required"`
			ClientID     string `json:"client_id" binding:"required"`
			ClientSecret string `json:"client_secret" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid request body", map[string]any{"field": "body"})
			return
		}
		if strings.TrimSpace(body.GrantType) != "client_credentials" {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "unsupported grant_type", map[string]any{"grant_type": body.GrantType})
			return
		}
		if !secureStringEq(strings.TrimSpace(body.ClientID), cfg.PlatformClientID) ||
			!secureStringEq(body.ClientSecret, cfg.PlatformClientSecret) {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid client credentials", nil)
			return
		}
		if TokenSvc == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "token service not configured", nil)
			return
		}
		issueTokenResponse(c, cfg.PlatformClientID)
	}
}

// IssueBootstrapToken handles POST /api/v1/auth/bootstrap.
// It mints a token server-side without exposing platform credentials to the SPA.
//
// Deprecated: prefer POST /api/v1/auth/login with cookie sessions; remove this endpoint once all clients have migrated.
// Operators may set AUTH_BOOTSTRAP_ENABLED=false to allow only registered-user authentication.
func IssueBootstrapToken(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.AuthBootstrapEnabled {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "bootstrap auth disabled; use user login", nil)
			return
		}
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "origin required", nil)
			return
		}
		if !originAllowed(origin, cfg.CORSAllowedOrigins) {
			slog.Warn("auth_bootstrap_denied",
				"request_id", requestid.FromContext(c),
				"origin", origin,
				"reason", "origin_not_allowed",
			)
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "origin not allowed", nil)
			return
		}
		slog.Info("auth_bootstrap_used",
			"request_id", requestid.FromContext(c),
			"origin", origin,
			"path", c.Request.URL.Path,
			"client_ip", strings.TrimSpace(c.ClientIP()),
			"deprecation", "temporary_bridge",
			"replacement_flow", "POST /api/v1/auth/login",
		)
		if AuthStore != nil {
			ctx := c.Request.Context()
			_ = AuthStore.InsertAudit(ctx, "auth_bootstrap_used", nil, clientIP(c), c.Request.UserAgent(), origin, nil)
		}
		issueBootstrapTokenResponse(c, cfg.PlatformClientID)
	}
}

// IssueJoinToken handles POST /api/v1/auth/join-token for authenticated end-users.
func IssueJoinToken(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub := strings.TrimSpace(c.GetString("auth_subject"))
		if !strings.HasPrefix(sub, "user:") {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "join token requires an authenticated user", nil)
			return
		}
		userID, err := strconv.Atoi(strings.TrimPrefix(sub, "user:"))
		if err != nil || userID <= 0 {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid session subject", nil)
			return
		}

		var body struct {
			SessionID string `json:"session_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid request body", map[string]any{"field": "body"})
			return
		}
		sessionID := strings.TrimSpace(body.SessionID)
		if len(sessionID) < 3 || len(sessionID) > 128 {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "session_id must be between 3 and 128 characters", map[string]any{"field": "session_id"})
			return
		}

		if TokenSvc == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "token service not configured", nil)
			return
		}
		rawToken, exp, err := TokenSvc.MintJoinToken(sub, sessionID, cfg.JoinTokenTTL)
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to issue join token", nil)
			return
		}
		if AuthStore != nil {
			meta, _ := json.Marshal(map[string]any{"session_id": sessionID, "token_use": "join"})
			_ = AuthStore.InsertAudit(c.Request.Context(), "auth_join_token_issued", &userID, clientIP(c), c.Request.UserAgent(), sub, meta)
		}

		expiresIn := int(time.Until(exp).Seconds())
		if expiresIn < 0 {
			expiresIn = 0
		}
		respond.JSONOK(c, http.StatusOK, gin.H{
			"join_token": rawToken,
			"token_type": "Bearer",
			"expires_in": expiresIn,
			"session_id": sessionID,
		})
	}
}

// StartDesktopExchangeCode issues a short-lived one-time code for desktop token exchange.
func StartDesktopExchangeCode(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		sub := strings.TrimSpace(c.GetString("auth_subject"))
		if !strings.HasPrefix(sub, "user:") {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "desktop exchange requires an authenticated user", nil)
			return
		}
		userID, err := strconv.Atoi(strings.TrimPrefix(sub, "user:"))
		if err != nil || userID <= 0 {
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid session subject", nil)
			return
		}
		var body struct {
			SessionID     string `json:"session_id" binding:"required"`
			CodeChallenge string `json:"code_challenge" binding:"required"`
			CallbackURI   string `json:"callback_uri"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid request body", map[string]any{"field": "body"})
			return
		}
		sessionID := strings.TrimSpace(body.SessionID)
		if len(sessionID) < 3 || len(sessionID) > 128 {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "session_id must be between 3 and 128 characters", map[string]any{"field": "session_id"})
			return
		}
		codeChallenge := strings.TrimSpace(body.CodeChallenge)
		if !isValidCodeChallenge(codeChallenge) {
			recordDesktopExchangeStartFailure(c, &userID, sub, "invalid_code_challenge", nil)
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "code_challenge must be a valid PKCE S256 challenge", map[string]any{"field": "code_challenge"})
			return
		}
		callbackURI := strings.TrimSpace(body.CallbackURI)
		if callbackURI != "" && !isAllowedLoopbackCallbackURI(callbackURI, cfg.DesktopExchangeCallbackHosts) {
			recordDesktopExchangeStartFailure(c, &userID, sub, "invalid_callback_uri", map[string]any{"callback_uri": callbackURI})
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "callback_uri must be a loopback http URL", map[string]any{"field": "callback_uri"})
			return
		}
		rawCode, err := authstore.NewOpaqueToken()
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to create exchange code", nil)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		if err := AuthStore.CreateDesktopExchangeCode(ctx, userID, rawCode, codeChallenge, sessionID, callbackURI, cfg.DesktopExchangeCodeTTL); err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to save exchange code", nil)
			return
		}
		meta, _ := json.Marshal(map[string]any{"session_id": sessionID, "callback_uri": callbackURI, "pkce": "s256"})
		_ = AuthStore.InsertAudit(ctx, "auth_desktop_exchange_start", &userID, clientIP(c), c.Request.UserAgent(), sub, meta)
		respond.JSONOK(c, http.StatusOK, gin.H{
			"exchange_code": rawCode,
			"expires_in":    int(cfg.DesktopExchangeCodeTTL.Seconds()),
			"session_id":    sessionID,
			"callback_uri":  callbackURI,
			"pkce_method":   "S256",
		})
	}
}

// ExchangeDesktopCode redeems a one-time code for a user bearer token.
func ExchangeDesktopCode(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthStore == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "auth store not configured", nil)
			return
		}
		if TokenSvc == nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "token service not configured", nil)
			return
		}
		var body struct {
			ExchangeCode string `json:"exchange_code" binding:"required"`
			CodeVerifier string `json:"code_verifier" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			recordDesktopExchangeFailure(c, nil, "", "invalid_request_body", nil)
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "invalid request body", map[string]any{"field": "body"})
			return
		}
		rawCode := strings.TrimSpace(body.ExchangeCode)
		if !isValidOpaqueCode(rawCode) {
			recordDesktopExchangeFailure(c, nil, "", "invalid_exchange_code_format", nil)
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "exchange_code is invalid", map[string]any{"field": "exchange_code"})
			return
		}
		codeVerifier := strings.TrimSpace(body.CodeVerifier)
		if !isValidCodeVerifier(codeVerifier) {
			recordDesktopExchangeFailure(c, nil, "", "invalid_code_verifier", nil)
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "code_verifier is invalid", map[string]any{"field": "code_verifier"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		row, err := AuthStore.RedeemDesktopExchangeCode(ctx, rawCode)
		if err != nil {
			switch err {
			case authstore.ErrExchangeCodeNotFound:
				recordDesktopExchangeFailure(c, nil, "", "code_not_found", nil)
				respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "exchange code is invalid or expired", nil)
				return
			case authstore.ErrExchangeCodeUsed:
				recordDesktopExchangeFailure(c, nil, "", "code_already_used", nil)
				respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "exchange code is invalid or expired", nil)
				return
			case authstore.ErrExchangeCodeExpired:
				recordDesktopExchangeFailure(c, nil, "", "code_expired", nil)
				respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "exchange code is invalid or expired", nil)
				return
			default:
				recordDesktopExchangeFailure(c, nil, "", "exchange_internal_error", nil)
				respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to exchange code", nil)
				return
			}
		}
		expectedChallenge := strings.TrimSpace(row.CodeChallenge)
		actualChallenge := pkceS256(codeVerifier)
		if expectedChallenge == "" || subtle.ConstantTimeCompare([]byte(expectedChallenge), []byte(actualChallenge)) != 1 {
			recordDesktopExchangeFailure(c, &row.UserID, "user:"+strconv.Itoa(row.UserID), "code_verifier_mismatch", map[string]any{"session_id": row.SessionID})
			respond.Error(c, http.StatusUnauthorized, api.CodeUnauthorized, "exchange code is invalid or expired", nil)
			return
		}
		sub := "user:" + strconv.Itoa(row.UserID)
		rawToken, exp, err := TokenSvc.MintDesktopAccessToken(sub, row.SessionID)
		if err != nil {
			respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to issue desktop token", nil)
			return
		}
		meta, _ := json.Marshal(map[string]any{"session_id": row.SessionID, "token_use": "desktop_access"})
		_ = AuthStore.InsertAudit(ctx, "auth_desktop_exchange_success", &row.UserID, clientIP(c), c.Request.UserAgent(), sub, meta)

		expiresIn := int(time.Until(exp).Seconds())
		if expiresIn < 0 {
			expiresIn = 0
		}
		respond.JSONOK(c, http.StatusOK, gin.H{
			"access_token": rawToken,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
			"session_id":   row.SessionID,
			"token_use":    "desktop_access",
		})
	}
}

func issueTokenResponse(c *gin.Context, clientID string) {
	if TokenSvc == nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "token service not configured", nil)
		return
	}
	subject := "client:" + clientID
	rawToken, exp, err := TokenSvc.MintAccessToken(subject)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to issue token", nil)
		return
	}
	expiresIn := int(time.Until(exp).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}
	respond.JSONOK(c, http.StatusOK, gin.H{
		"access_token": rawToken,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
	})
}

func issueBootstrapTokenResponse(c *gin.Context, clientID string) {
	if TokenSvc == nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "token service not configured", nil)
		return
	}
	subject := "client:" + clientID
	rawToken, exp, err := TokenSvc.MintAccessToken(subject)
	if err != nil {
		respond.Error(c, http.StatusInternalServerError, api.CodeInternal, "failed to issue token", nil)
		return
	}
	expiresIn := int(time.Until(exp).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}
	respond.JSONOK(c, http.StatusOK, gin.H{
		"access_token": rawToken,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
		"bootstrap": gin.H{
			"temporary":      true,
			"deprecated":     true,
			"migrate_to":     "POST /api/v1/auth/login",
			"disable_env":    "AUTH_BOOTSTRAP_ENABLED=false",
			"session_cookie": "gl_session (see SESSION_COOKIE_NAME)",
		},
	})
}

func originAllowed(origin string, allowlist []string) bool {
	for _, allowed := range allowlist {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}
	return false
}

func secureStringEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func isAllowedLoopbackCallbackURI(raw string, allowedHosts []string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	if u.User != nil || strings.TrimSpace(u.Fragment) != "" {
		return false
	}
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return false
		}
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return false
	}
	for _, allowed := range allowedHosts {
		if host == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func recordDesktopExchangeStartFailure(c *gin.Context, userID *int, subject, reason string, extra map[string]any) {
	if AuthStore == nil {
		return
	}
	meta := map[string]any{"reason": reason}
	for k, v := range extra {
		meta[k] = v
	}
	metaJSON, _ := json.Marshal(meta)
	_ = AuthStore.InsertAudit(c.Request.Context(), "auth_desktop_exchange_start_failure", userID, clientIP(c), c.Request.UserAgent(), subject, metaJSON)
}

func recordDesktopExchangeFailure(c *gin.Context, userID *int, subject, reason string, extra map[string]any) {
	if AuthStore == nil {
		return
	}
	meta := map[string]any{"reason": reason}
	for k, v := range extra {
		meta[k] = v
	}
	metaJSON, _ := json.Marshal(meta)
	_ = AuthStore.InsertAudit(c.Request.Context(), "auth_desktop_exchange_failure", userID, clientIP(c), c.Request.UserAgent(), subject, metaJSON)
}

func isValidOpaqueCode(raw string) bool {
	if len(raw) != 64 {
		return false
	}
	for i := 0; i < len(raw); i++ {
		b := raw[i]
		isDigit := b >= '0' && b <= '9'
		isLowerHex := b >= 'a' && b <= 'f'
		if !isDigit && !isLowerHex {
			return false
		}
	}
	return true
}

func isValidCodeChallenge(v string) bool {
	// RFC 7636 S256 output is base64url without padding (43 chars for SHA-256).
	if len(v) != 43 {
		return false
	}
	for i := 0; i < len(v); i++ {
		b := v[i]
		isAlphaNum := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
		if !isAlphaNum && b != '-' && b != '_' {
			return false
		}
	}
	return true
}

func isValidCodeVerifier(v string) bool {
	// RFC 7636 verifier: 43..128 chars, unreserved URI characters.
	if len(v) < 43 || len(v) > 128 {
		return false
	}
	for i := 0; i < len(v); i++ {
		b := v[i]
		isAlphaNum := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
		if !isAlphaNum && b != '-' && b != '.' && b != '_' && b != '~' {
			return false
		}
	}
	return true
}

func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
