package myhandlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/middleware"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

func TestChangePasswordWithBearerRevokesSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	Database = &DBConn{Db: db}
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	oldHash, err := auth.HashPassword("old-pass-9")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT password_hash FROM users WHERE id = \?`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(oldHash))
	mock.ExpectExec(`UPDATE users SET password_hash = \? WHERE id = \?`).
		WithArgs(sqlmock.AnyArg(), 42).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE auth_sessions SET revoked_at = UTC_TIMESTAMP\(\) WHERE user_id = \? AND revoked_at IS NULL`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := ts.MintAccessToken("user:42")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(middleware.CSRFCookieProtect(cfg))
	r.POST("/api/v1/auth/change-password",
		middleware.BearerOrSession(ts, AuthStore, "gl_session", nil),
		ChangePassword(cfg),
	)

	body := bytes.NewBufferString(`{"current_password":"old-pass-9","new_password":"new-pass-10"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
