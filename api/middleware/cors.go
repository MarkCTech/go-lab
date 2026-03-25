package middleware

import (
	"net/http"
	"strings"

	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

// DynamicCORS applies an explicit origin allowlist (no wildcard when credentials might be used).
// extraAllowHeaders are appended to Access-Control-Allow-Headers (e.g. CSRF header name).
func DynamicCORS(allowedOrigins []string, extraAllowHeaders ...string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allow[o] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := allow[origin]; !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", strings.Join([]string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions,
		}, ", "))
		baseHdr := "Authorization, Content-Type, Accept, " + requestid.Header
		for _, h := range extraAllowHeaders {
			h = strings.TrimSpace(h)
			if h != "" {
				baseHdr += ", " + h
			}
		}
		c.Header("Access-Control-Allow-Headers", baseHdr)
		c.Header("Access-Control-Expose-Headers", "Content-Length, "+requestid.Header)
		c.Header("Access-Control-Max-Age", "600")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
