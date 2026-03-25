package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthUserIDFromContext returns the numeric user id when auth_subject is "user:<id>".
func AuthUserIDFromContext(c *gin.Context) (int, bool) {
	sub := strings.TrimSpace(c.GetString("auth_subject"))
	const prefix = "user:"
	if !strings.HasPrefix(sub, prefix) {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimSpace(sub[len(prefix):]))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
