package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

func TestRequirePlatformActionReasonAllows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/x", RequirePlatformActionReason(4), func(c *gin.Context) {
		c.Status(http.StatusTeapot)
	})
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set(PlatformActionReasonHeader, "good reason here")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected 418, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestRequirePlatformActionReasonRejectsShort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/x", RequirePlatformActionReason(10), func(c *gin.Context) {
		t.Fatal("handler should not run")
	})
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set(PlatformActionReasonHeader, "short")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rr.Code, rr.Body.String())
	}
}
