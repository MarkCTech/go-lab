package respond

import (
	"net/http"

	"github.com/codemarked/go-lab/api/api"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

// OK sends a 200 JSON success envelope with data.
func OK(c *gin.Context, data any) {
	JSONOK(c, http.StatusOK, data)
}

// JSONOK sends a JSON success envelope with the given HTTP status.
func JSONOK(c *gin.Context, status int, data any) {
	c.JSON(status, api.OKEnvelope{
		Data: data,
		Meta: api.Meta{RequestID: requestid.FromContext(c)},
	})
}

// Error sends a JSON error envelope.
func Error(c *gin.Context, status int, code, message string, details map[string]any) {
	c.JSON(status, api.ErrorEnvelope{
		Error: api.ErrBody{Code: code, Message: message, Details: details},
		Meta:  api.Meta{RequestID: requestid.FromContext(c)},
	})
}

// NoContent sends 204 with no body (DELETE success).
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
