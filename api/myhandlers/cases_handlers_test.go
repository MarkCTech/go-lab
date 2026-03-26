package myhandlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/gin-gonic/gin"
)

func setupCasesRouterWithMockDB(t *testing.T) (*gin.Engine, sqlmock.Sqlmock, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	Database = &DBConn{Db: db}
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("auth_subject", "user:1")
		c.Next()
	})
	r.POST("/api/v1/cases", PostOperatorCase)
	r.PATCH("/api/v1/cases/:id", PatchOperatorCase)

	cleanup := func() {
		_ = db.Close()
	}
	return r, mock, cleanup
}

func TestPostOperatorCaseInvalidPriorityReturnsValidation(t *testing.T) {
	r, mock, cleanup := setupCasesRouterWithMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id FROM users WHERE id = \? AND deleted_at IS NULL`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

	body := bytes.NewBufferString(`{"subject_platform_user_id":7,"title":"T","priority":"bad-priority"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid case priority") {
		t.Fatalf("expected invalid priority message, body=%s", rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPatchOperatorCaseInvalidStatusReturnsValidation(t *testing.T) {
	r, mock, cleanup := setupCasesRouterWithMockDB(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"status":"bad-status"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cases/55", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid case status") {
		t.Fatalf("expected invalid status message, body=%s", rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPatchOperatorCaseInvalidPriorityReturnsValidation(t *testing.T) {
	r, mock, cleanup := setupCasesRouterWithMockDB(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"priority":"bad-priority"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/cases/55", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "invalid case priority") {
		t.Fatalf("expected invalid priority message, body=%s", rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
