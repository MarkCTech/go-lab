package myhandlers

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/codemarked/go-lab/go_CRUD_api/api"
	"github.com/codemarked/go-lab/go_CRUD_api/auth"
	"github.com/codemarked/go-lab/go_CRUD_api/config"
	"github.com/codemarked/go-lab/go_CRUD_api/respond"
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
func IssueBootstrapToken(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "origin required", nil)
			return
		}
		if !originAllowed(origin, cfg.CORSAllowedOrigins) {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "origin not allowed", nil)
			return
		}
		issueTokenResponse(c, cfg.PlatformClientID)
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
