package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

func TestBearerOrSessionInvalidScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService("01234567890123456789012345678901", "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(requestid.Middleware())
	r.GET("/x", BearerOrSession(ts, nil, "", nil), func(c *gin.Context) {
		t.Fatal("handler should not run")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Basic c3VwZXI6c2VjcmV0")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestBearerOrSessionMissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService("01234567890123456789012345678901", "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(requestid.Middleware())
	r.GET("/x", BearerOrSession(ts, nil, "gl_session", nil), func(c *gin.Context) {
		t.Fatal("handler should not run")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", rr.Code, rr.Body.String())
	}
}
