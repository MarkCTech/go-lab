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

func TestRequireHumanUserAllowsUserSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService("01234567890123456789012345678901", "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := ts.MintAccessToken("user:9")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(requestid.Middleware())
	r.GET("/x", BearerAuth(ts), RequireHumanUser(), func(c *gin.Context) {
		c.Status(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected 418, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestRequireHumanUserBlocksClientSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService("01234567890123456789012345678901", "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := ts.MintAccessToken("client:dev-platform")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(requestid.Middleware())
	r.GET("/x", BearerAuth(ts), RequireHumanUser(), func(c *gin.Context) {
		t.Fatal("handler should not run")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", rr.Code, rr.Body.String())
	}
}
