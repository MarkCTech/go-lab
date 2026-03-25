package myhandlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/authstore"
	"github.com/codemarked/go-lab/api/config"
	"github.com/codemarked/go-lab/api/middleware"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

func testAuthConfig() *config.Config {
	return &config.Config{
		SessionCookieName:     "gl_session",
		SessionCookieSecure:   false,
		SessionSameSiteMode:   0, // Lax default unused in test without SetCookie assert
		SessionIdleTTL:        30 * time.Minute,
		SessionAbsoluteTTL:    24 * time.Hour,
		AuthBootstrapEnabled:  true,
		CSRFCookieName:        "gl_csrf",
		CSRFHeaderName:        "X-CSRF-Token",
	}
}

func TestRegisterUserSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	Database = &DBConn{Db: db}
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	mock.ExpectExec(`INSERT INTO users \(name, email, password_hash, pennies\) VALUES \(\?, \?, \?, 0\)`).
		WithArgs("Reg User", "reg@example.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(101, 1))
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_register_success", 101, sqlmock.AnyArg(), sqlmock.AnyArg(), "reg@example.com", nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/register", RegisterUser(cfg))

	body := bytes.NewBufferString(`{"email":"reg@example.com","password":"longenough","name":"Reg User"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}
	var env struct {
		Data struct {
			ID    int64  `json:"id"`
			Email string `json:"email"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.ID != 101 || env.Data.Email != "reg@example.com" {
		t.Fatalf("unexpected data: %+v", env.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginSetsSessionAndUsersAcceptsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	Database = &DBConn{Db: db}
	store := authstore.New(db, 30*time.Minute, 24*time.Hour)
	AuthStore = store
	cfg := testAuthConfig()

	hash, err := auth.HashPassword("login-pass-9")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email = \?`).
		WithArgs("login@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(7, hash))

	mock.ExpectExec(`INSERT INTO auth_sessions`).
		WithArgs(7, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_login_success", 7, sqlmock.AnyArg(), sqlmock.AnyArg(), "login@example.com", nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/login", LoginUser(cfg))

	loginBody := `{"email":"login@example.com","password":"login-pass-9"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status %d body=%s", rr.Code, rr.Body.String())
	}

	cookies := rr.Result().Cookies()
	var sessionRaw string
	for _, ck := range cookies {
		if ck.Name == "gl_session" {
			sessionRaw = ck.Value
			break
		}
	}
	if sessionRaw == "" {
		t.Fatal("expected gl_session cookie")
	}

	hashTok := authstore.HashOpaqueToken(sessionRaw)
	abs := time.Now().UTC().Add(24 * time.Hour)
	slide := time.Now().UTC().Add(29 * time.Minute)
	mock.ExpectQuery(`SELECT id, user_id, expires_at, absolute_expires_at, revoked_at FROM auth_sessions WHERE token_hash = \?`).
		WithArgs(hashTok).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "absolute_expires_at", "revoked_at"}).
			AddRow(int64(1), 7, slide, abs, nil))
	mock.ExpectExec(`UPDATE auth_sessions SET last_seen_at = UTC_TIMESTAMP\(\), expires_at = \? WHERE id = \? AND revoked_at IS NULL`).
		WithArgs(sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`SELECT id, name, pennies FROM users ORDER BY pennies DESC, id ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "pennies"}).AddRow(7, "U", 0))

	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts

	r2 := gin.New()
	r2.Use(requestid.Middleware())
	u := r2.Group("/api/v1/users")
	u.Use(middleware.BearerOrSession(ts, store, "gl_session", nil))
	u.GET("", GetUsers)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req2.Header.Set("Cookie", "gl_session="+sessionRaw)
	rr2 := httptest.NewRecorder()
	r2.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("users with cookie: status %d body=%s", rr2.Code, rr2.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	raw := "beefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeef"
	csrf := "cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe"
	mock.ExpectExec(`UPDATE auth_sessions SET revoked_at = UTC_TIMESTAMP\(\) WHERE token_hash = \? AND revoked_at IS NULL`).
		WithArgs(authstore.HashOpaqueToken(raw)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := gin.New()
	r.Use(middleware.CSRFCookieProtect(cfg))
	r.POST("/api/v1/auth/logout", LogoutUser(cfg))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Cookie", "gl_session="+raw+"; gl_csrf="+csrf)
	req.Header.Set("X-CSRF-Token", csrf)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCookiePOSTUsersWithoutCSRFIsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	cfg := testAuthConfig()
	store := authstore.New(db, 30*time.Minute, 24*time.Hour)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(middleware.CSRFCookieProtect(cfg))
	v1 := r.Group("/api/v1")
	u := v1.Group("/users")
	u.Use(middleware.BearerOrSession(ts, store, "gl_session", nil))
	u.POST("", CreateUser)

	sess := "abcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd"
	body := bytes.NewBufferString(`{"name":"X","pennies":0}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "gl_session="+sess)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 csrf, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterInvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	Database = &DBConn{Db: db}
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/register", RegisterUser(cfg))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestLoginUnknownUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	Database = &DBConn{Db: db}
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email = \?`).
		WithArgs("nope@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`INSERT INTO auth_audit_events \(event_type, user_id, ip, user_agent, subject_hint, meta_json\)`).
		WithArgs("auth_login_failure", nil, sqlmock.AnyArg(), sqlmock.AnyArg(), "nope@example.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/login", LoginUser(cfg))

	body := bytes.NewBufferString(`{"email":"nope@example.com","password":"whatever-8"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	Database = &DBConn{Db: db}
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	hash, err := auth.HashPassword("correct-pass-9")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email = \?`).
		WithArgs("u@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(1, hash))
	mock.ExpectExec(`INSERT INTO auth_audit_events \(event_type, user_id, ip, user_agent, subject_hint, meta_json\)`).
		WithArgs("auth_login_failure", 1, sqlmock.AnyArg(), sqlmock.AnyArg(), "u@example.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/login", LoginUser(cfg))

	body := bytes.NewBufferString(`{"email":"u@example.com","password":"wrong-pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshMissingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	r := gin.New()
	r.POST("/api/v1/auth/refresh", RefreshSession(cfg))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetAuthCSRFWithoutSessionUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := testAuthConfig()

	r := gin.New()
	r.GET("/api/v1/auth/csrf", GetAuthCSRF(cfg))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/csrf", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
