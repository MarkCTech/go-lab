package middleware

import (
	"net/http"
	"strings"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

// PlatformActionReasonHeader is required on privileged mutating control-plane routes (cookie or Bearer).
const PlatformActionReasonHeader = "X-Platform-Action-Reason"

// RequirePlatformActionReason rejects empty or too-short reasons for operator accountability.
func RequirePlatformActionReason(minLen int) gin.HandlerFunc {
	if minLen < 1 {
		minLen = 8
	}
	return func(c *gin.Context) {
		reason := strings.TrimSpace(c.GetHeader(PlatformActionReasonHeader))
		if len(reason) < minLen {
			respond.Error(c, http.StatusBadRequest, api.CodeValidation, "missing or too short privileged action reason", map[string]any{
				"header":  PlatformActionReasonHeader,
				"min_len": minLen,
			})
			c.Abort()
			return
		}
		c.Set("platform_action_reason", reason)
		c.Next()
	}
}
