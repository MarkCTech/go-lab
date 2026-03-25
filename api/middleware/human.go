package middleware

import (
	"net/http"
	"strings"

	"github.com/codemarked/go-lab/api/api"
	"github.com/gin-gonic/gin"
)

// RequireHumanUser rejects authenticated subjects that are not end-user accounts (subject must be "user:…").
// Machine tokens from client_credentials use "client:…" and must not mutate or delete user rows.
func RequireHumanUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		sub := strings.TrimSpace(c.GetString("auth_subject"))
		if !strings.HasPrefix(sub, "user:") {
			writeAuthError(c, http.StatusForbidden, api.CodeForbidden, "this operation requires a signed-in user account")
			c.Abort()
			return
		}
		c.Next()
	}
}
