package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/platformrbac"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

func TestRequirePlatformPermissionOperatorFromDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	store := authstore.New(db, time.Hour, time.Hour)
	mock.ExpectQuery(`SELECT DISTINCT r.name`).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("operator"))

	r := gin.New()
	r.Use(requestid.Middleware())
	r.GET("/p",
		func(c *gin.Context) {
			c.Set("auth_subject", "user:7")
			c.Next()
		},
		RequirePlatformPermission(store, platformrbac.PermAuditWrite),
		func(c *gin.Context) { c.Status(http.StatusTeapot) },
	)
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected 418, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRequirePlatformPermissionDeniedNoRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	store := authstore.New(db, time.Hour, time.Hour)
	mock.ExpectQuery(`SELECT DISTINCT r.name`).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows([]string{"name"}))

	r := gin.New()
	r.Use(requestid.Middleware())
	r.GET("/p",
		func(c *gin.Context) {
			c.Set("auth_subject", "user:3")
			c.Next()
		},
		RequirePlatformPermission(store, platformrbac.PermPlayersRead),
		func(c *gin.Context) { t.Fatal("handler should not run") },
	)
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRequirePlatformPermissionDBRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	store := authstore.New(db, time.Hour, time.Hour)
	mock.ExpectQuery(`SELECT DISTINCT r.name`).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("support"))

	r := gin.New()
	r.Use(requestid.Middleware())
	r.GET("/p",
		func(c *gin.Context) {
			c.Set("auth_subject", "user:2")
			c.Next()
		},
		RequirePlatformPermission(store, platformrbac.PermPlayersRead),
		func(c *gin.Context) { c.Status(http.StatusTeapot) },
	)
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected 418, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
