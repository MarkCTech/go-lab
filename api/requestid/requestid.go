package requestid

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	// Header is the canonical request correlation header.
	Header = "X-Request-ID"
	ctxKey = "request_id"
)

// Middleware ensures every request has a correlation id.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(Header)
		if rid == "" || len(rid) > 128 {
			rid = newID()
		}
		c.Set(ctxKey, rid)
		c.Writer.Header().Set(Header, rid)
		c.Next()
	}
}

// FromContext returns the request id (empty if missing).
func FromContext(c *gin.Context) string {
	if v, ok := c.Get(ctxKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
