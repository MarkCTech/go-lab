package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/redisx"
	"github.com/codemarked/go-lab/api/respond"
	"github.com/gin-gonic/gin"
)

type ipWindow struct {
	windowStart time.Time
	count       int
}

// NewFixedWindowLimiter limits requests per client IP in a fixed 1-minute window.
// scope isolates Redis keys between routes. With redisx.Client set, counts are shared across replicas; otherwise in-memory only.
func NewFixedWindowLimiter(scope string, rpm int, message string) gin.HandlerFunc {
	if rpm <= 0 {
		rpm = 30
	}
	if message == "" {
		message = "too many requests"
	}
	if scope == "" {
		scope = "default"
	}
	var mu sync.Mutex
	byIP := make(map[string]*ipWindow)
	const win = time.Minute
	lastSweep := time.Now()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if rdb := redisx.Client; rdb != nil {
			ok, rerr := fixedWindowRedisAllow(c.Request.Context(), rdb, scope, ip, rpm)
			if rerr != nil {
				// Redis degradation should not disable protection entirely; fall back to in-process limiting.
				slog.Warn("redis_rate_limit_fallback_local", "scope", scope, "error", rerr.Error())
			} else {
				if !ok {
					respond.Error(c, http.StatusTooManyRequests, api.CodeRateLimited, message, nil)
					c.Abort()
					return
				}
				c.Next()
				return
			}
		}

		now := time.Now()
		mu.Lock()
		if now.Sub(lastSweep) >= win {
			for k, v := range byIP {
				if now.Sub(v.windowStart) >= 10*win {
					delete(byIP, k)
				}
			}
			lastSweep = now
		}
		w, ok := byIP[ip]
		if !ok {
			w = &ipWindow{windowStart: now, count: 0}
			byIP[ip] = w
		}
		if now.Sub(w.windowStart) >= win {
			w.windowStart = now
			w.count = 0
		}
		if w.count >= rpm {
			mu.Unlock()
			respond.Error(c, http.StatusTooManyRequests, api.CodeRateLimited, message, nil)
			c.Abort()
			return
		}
		w.count++
		mu.Unlock()
		c.Next()
	}
}

// NewTokenEndpointLimiter limits token endpoint calls per client IP (fixed 1-minute window).
func NewTokenEndpointLimiter(rpm int) gin.HandlerFunc {
	return NewFixedWindowLimiter("auth_token", rpm, "too many token requests")
}
