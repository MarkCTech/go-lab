package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

// BearerAuth validates Authorization: Bearer <JWT> using the token service.
func BearerAuth(ts *auth.TokenService) gin.HandlerFunc {
	if ts == nil {
		return func(c *gin.Context) {
			writeAuthError(c, http.StatusInternalServerError, api.CodeInternal, "authentication not configured")
			c.Abort()
		}
	}
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" {
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "missing authorization header")
			c.Abort()
			return
		}
		const prefix = "Bearer "
		if len(raw) < len(prefix) || !strings.EqualFold(raw[:len(prefix)], prefix) {
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid authorization scheme")
			c.Abort()
			return
		}
		token := strings.TrimSpace(raw[len(prefix):])
		if token == "" {
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "empty bearer token")
			c.Abort()
			return
		}
		claims, err := ts.ParseAccessToken(token)
		if err != nil {
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}
		sub := strings.TrimSpace(claims.Subject)
		if sub == "" {
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid token subject")
			c.Abort()
			return
		}
		c.Set("auth_subject", sub)
		c.Next()
	}
}

func writeAuthError(c *gin.Context, status int, code, message string) {
	c.JSON(status, api.ErrorEnvelope{
		Error: api.ErrBody{Code: code, Message: message},
		Meta:  api.Meta{RequestID: requestid.FromContext(c)},
	})
}

// BearerOrSession accepts either Authorization: Bearer (JWT) or a valid HttpOnly session cookie (opaque token, server-side hash).
// If Authorization is present, cookie is ignored (avoids ambiguous dual-auth).
// Bearer path: HS256 platform JWT first; if that fails and oidc is non-nil, RS256 OIDC access tokens (e.g. Auth0) are tried.
func BearerOrSession(ts *auth.TokenService, store *authstore.Store, sessionCookieName string, oidc *auth.OIDC) gin.HandlerFunc {
	if ts == nil {
		return func(c *gin.Context) {
			writeAuthError(c, http.StatusInternalServerError, api.CodeInternal, "authentication not configured")
			c.Abort()
		}
	}
	return func(c *gin.Context) {
		rawAuth := strings.TrimSpace(c.GetHeader("Authorization"))
		if rawAuth != "" {
			const prefix = "Bearer "
			if len(rawAuth) < len(prefix) || !strings.EqualFold(rawAuth[:len(prefix)], prefix) {
				writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid authorization scheme")
				c.Abort()
				return
			}
			token := strings.TrimSpace(rawAuth[len(prefix):])
			if token == "" {
				writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "empty bearer token")
				c.Abort()
				return
			}
			claims, errHS := ts.ParseAccessToken(token)
			if errHS == nil {
				sub := strings.TrimSpace(claims.Subject)
				if sub == "" {
					writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid token subject")
					c.Abort()
					return
				}
				c.Set("auth_subject", sub)
				c.Next()
				return
			}
			if oidc != nil {
				subj, errOIDC := oidc.TryResolveAuthSubject(c.Request.Context(), token)
				if errOIDC == nil {
					c.Set("auth_subject", subj)
					c.Next()
					return
				}
				auth.LogReject(requestid.FromContext(c), token, errOIDC)
				writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid or expired token")
				c.Abort()
				return
			}
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		if store == nil || strings.TrimSpace(sessionCookieName) == "" {
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "missing authorization")
			c.Abort()
			return
		}
		rawCookie, err := c.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(rawCookie) == "" {
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "missing session")
			c.Abort()
			return
		}
		uid, err := store.ValidateSession(c.Request.Context(), rawCookie)
		if err != nil {
			slog.Warn("session_validate_failed",
				"request_id", requestid.FromContext(c),
				"path", c.Request.URL.Path,
				"reason", err.Error(),
			)
			writeAuthError(c, http.StatusUnauthorized, api.CodeUnauthorized, "invalid or expired session")
			c.Abort()
			return
		}
		c.Set("auth_subject", fmt.Sprintf("user:%d", uid))
		c.Next()
	}
}
