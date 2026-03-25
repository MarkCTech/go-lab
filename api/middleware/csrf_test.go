package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codemarked/go-lab/api/config"
	"github.com/gin-gonic/gin"
)

func TestCSRFSkipsWhenBearerPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SessionCookieName: "gl_session",
		CSRFCookieName:    "gl_csrf",
		CSRFHeaderName:    "X-CSRF-Token",
	}
	r := gin.New()
	r.Use(CSRFCookieProtect(cfg))
	r.POST("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	req.Header.Set("Cookie", "gl_session=should-be-ignored-for-csrf")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestCSRFAllowsRegisterExemptPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SessionCookieName: "gl_session",
		CSRFCookieName:    "gl_csrf",
		CSRFHeaderName:    "X-CSRF-Token",
	}
	r := gin.New()
	r.Use(CSRFCookieProtect(cfg))
	r.POST("/api/v1/auth/register", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestCSRFAllowsDesktopExchangeExemptPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		SessionCookieName: "gl_session",
		CSRFCookieName:    "gl_csrf",
		CSRFHeaderName:    "X-CSRF-Token",
	}
	r := gin.New()
	r.Use(CSRFCookieProtect(cfg))
	r.POST("/api/v1/auth/desktop/exchange", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/exchange", nil)
	req.Header.Set("Cookie", "gl_session=session-present-without-csrf")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
}
