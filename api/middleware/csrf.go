package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/config"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

// CSRFExemptPathPrefixes are mutating auth routes that run before a CSRF cookie exists.
var CSRFExemptPathPrefixes = []string{
	"/api/v1/auth/register",
	"/api/v1/auth/login",
	"/api/v1/auth/token",
	"/api/v1/auth/bootstrap",
	"/api/v1/auth/desktop/exchange",
}

// CSRFCookieProtect enforces double-submit cookie vs header for unsafe methods when the client
// uses cookie sessions (no Bearer). Bearer-authenticated requests skip CSRF.
func CSRFCookieProtect(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions || m == http.MethodTrace {
			c.Next()
			return
		}
		authz := strings.TrimSpace(c.GetHeader("Authorization"))
		if len(authz) >= 7 && strings.EqualFold(authz[:7], "Bearer ") {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		for _, p := range CSRFExemptPathPrefixes {
			if strings.HasPrefix(path, p) {
				c.Next()
				return
			}
		}
		sess, err := c.Cookie(cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(sess) == "" {
			// Logout without session is a no-op; no CSRF surface.
			if path == "/api/v1/auth/logout" {
				c.Next()
				return
			}
			c.Next()
			return
		}
		cookieTok, err := c.Cookie(cfg.CSRFCookieName)
		hdr := strings.TrimSpace(c.GetHeader(cfg.CSRFHeaderName))
		if err != nil || strings.TrimSpace(cookieTok) == "" || hdr == "" ||
			subtle.ConstantTimeCompare([]byte(cookieTok), []byte(hdr)) != 1 {
			respond.Error(c, http.StatusForbidden, api.CodeForbidden, "csrf token missing or invalid", map[string]any{
				"header": cfg.CSRFHeaderName,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
