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
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

const testCodeVerifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~abcde"

func TestIssueTokenUnsupportedGrant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	cfg := &config.Config{
		PlatformClientID:     "dev-platform",
		PlatformClientSecret: "secret-sixteen-chars",
	}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/token", IssueToken(cfg))

	body := bytes.NewBufferString(`{"grant_type":"password","client_id":"dev-platform","client_secret":"secret-sixteen-chars"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestIssueTokenInvalidClientCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	cfg := &config.Config{
		PlatformClientID:     "dev-platform",
		PlatformClientSecret: "secret-sixteen-chars",
	}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/token", IssueToken(cfg))

	body := bytes.NewBufferString(`{"grant_type":"client_credentials","client_id":"dev-platform","client_secret":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestIssueJoinTokenSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	cfg := &config.Config{JoinTokenTTL: 2 * time.Minute}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(func(c *gin.Context) {
		c.Set("auth_subject", "user:7")
		c.Next()
	})
	r.POST("/api/v1/auth/join-token", IssueJoinToken(cfg))

	body := bytes.NewBufferString(`{"session_id":"room-alpha"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/join-token", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rr.Code, rr.Body.String())
	}
	var env struct {
		Data struct {
			JoinToken string `json:"join_token"`
			TokenType string `json:"token_type"`
			ExpiresIn int    `json:"expires_in"`
			SessionID string `json:"session_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.TokenType != "Bearer" || env.Data.SessionID != "room-alpha" || env.Data.ExpiresIn <= 0 || env.Data.JoinToken == "" {
		t.Fatalf("unexpected response: %+v", env.Data)
	}
	claims, err := ts.ParseAccessToken(env.Data.JoinToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user:7" || claims.TokenUse != "join" || claims.JoinSessionID != "room-alpha" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestIssueJoinTokenRequiresUserSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	cfg := &config.Config{JoinTokenTTL: 2 * time.Minute}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(func(c *gin.Context) {
		c.Set("auth_subject", "client:svc")
		c.Next()
	})
	r.POST("/api/v1/auth/join-token", IssueJoinToken(cfg))

	body := bytes.NewBufferString(`{"session_id":"room-alpha"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/join-token", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestIssueJoinTokenValidatesSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	cfg := &config.Config{JoinTokenTTL: 2 * time.Minute}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(func(c *gin.Context) {
		c.Set("auth_subject", "user:7")
		c.Next()
	})
	r.POST("/api/v1/auth/join-token", IssueJoinToken(cfg))

	body := bytes.NewBufferString(`{"session_id":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/join-token", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestStartDesktopExchangeCodeSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := &config.Config{
		DesktopExchangeCodeTTL:       2 * time.Minute,
		DesktopExchangeCallbackHosts: []string{"localhost", "127.0.0.1", "::1"},
	}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(func(c *gin.Context) {
		c.Set("auth_subject", "user:7")
		c.Next()
	})
	r.POST("/api/v1/auth/desktop/start", StartDesktopExchangeCode(cfg))

	mock.ExpectExec(`INSERT INTO auth_desktop_exchange_codes`).
		WithArgs(7, sqlmock.AnyArg(), pkceS256(testCodeVerifier), "room-alpha", "http://localhost:7777/callback", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_desktop_exchange_start", 7, sqlmock.AnyArg(), sqlmock.AnyArg(), "user:7", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"session_id":"room-alpha","code_challenge":"` + pkceS256(testCodeVerifier) + `","callback_uri":"http://localhost:7777/callback"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/start", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rr.Code, rr.Body.String())
	}
	var env struct {
		Data struct {
			ExchangeCode string `json:"exchange_code"`
			ExpiresIn    int    `json:"expires_in"`
			SessionID    string `json:"session_id"`
			PKCEMethod   string `json:"pkce_method"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.ExchangeCode == "" || env.Data.ExpiresIn != 120 || env.Data.SessionID != "room-alpha" || env.Data.PKCEMethod != "S256" {
		t.Fatalf("unexpected response: %+v", env.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStartDesktopExchangeCodeRejectsDisallowedCallbackHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := &config.Config{
		DesktopExchangeCodeTTL:       2 * time.Minute,
		DesktopExchangeCallbackHosts: []string{"localhost"},
	}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(func(c *gin.Context) {
		c.Set("auth_subject", "user:7")
		c.Next()
	})
	r.POST("/api/v1/auth/desktop/start", StartDesktopExchangeCode(cfg))

	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_desktop_exchange_start_failure", 7, sqlmock.AnyArg(), sqlmock.AnyArg(), "user:7", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"session_id":"room-alpha","code_challenge":"` + pkceS256(testCodeVerifier) + `","callback_uri":"http://evil.example/callback"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/start", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExchangeDesktopCodeSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	cfg := &config.Config{}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/desktop/exchange", ExchangeDesktopCode(cfg))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, user_id, code_challenge, session_id, expires_at, consumed_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code_challenge", "session_id", "expires_at", "consumed_at"}).
			AddRow(int64(11), 7, pkceS256(testCodeVerifier), "room-alpha", time.Now().UTC().Add(5*time.Minute), nil))
	mock.ExpectExec(`UPDATE auth_desktop_exchange_codes`).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_desktop_exchange_success", 7, sqlmock.AnyArg(), sqlmock.AnyArg(), "user:7", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"exchange_code":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","code_verifier":"` + testCodeVerifier + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/exchange", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rr.Code, rr.Body.String())
	}
	var env struct {
		Data struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			SessionID   string `json:"session_id"`
			TokenUse    string `json:"token_use"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.AccessToken == "" || env.Data.TokenType != "Bearer" || env.Data.SessionID != "room-alpha" || env.Data.TokenUse != "desktop_access" {
		t.Fatalf("unexpected response: %+v", env.Data)
	}
	claims, err := ts.ParseAccessToken(env.Data.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user:7" || claims.TokenUse != "desktop_access" || claims.SessionID != "room-alpha" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExchangeDesktopCodeNotFoundAudited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/desktop/exchange", ExchangeDesktopCode(&config.Config{}))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, user_id, code_challenge, session_id, expires_at, consumed_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_desktop_exchange_failure", nil, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"exchange_code":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","code_verifier":"` + testCodeVerifier + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/exchange", body)
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

func TestExchangeDesktopCodeVerifierMismatchAudited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/desktop/exchange", ExchangeDesktopCode(&config.Config{}))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, user_id, code_challenge, session_id, expires_at, consumed_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code_challenge", "session_id", "expires_at", "consumed_at"}).
			AddRow(int64(11), 7, pkceS256(testCodeVerifier), "room-alpha", time.Now().UTC().Add(5*time.Minute), nil))
	mock.ExpectExec(`UPDATE auth_desktop_exchange_codes`).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_desktop_exchange_failure", 7, sqlmock.AnyArg(), sqlmock.AnyArg(), "user:7", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"exchange_code":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","code_verifier":"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~zzzzz"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/exchange", body)
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

func TestExchangeDesktopCodeExpiredAudited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/desktop/exchange", ExchangeDesktopCode(&config.Config{}))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, user_id, code_challenge, session_id, expires_at, consumed_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code_challenge", "session_id", "expires_at", "consumed_at"}).
			AddRow(int64(11), 7, pkceS256(testCodeVerifier), "room-alpha", time.Now().UTC().Add(-5*time.Minute), nil))
	mock.ExpectRollback()
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_desktop_exchange_failure", nil, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"exchange_code":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","code_verifier":"` + testCodeVerifier + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/exchange", body)
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

func TestExchangeDesktopCodeAlreadyUsedAudited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	TokenSvc = ts
	r := gin.New()
	r.Use(requestid.Middleware())
	r.POST("/api/v1/auth/desktop/exchange", ExchangeDesktopCode(&config.Config{}))

	past := time.Now().UTC().Add(-1 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, user_id, code_challenge, session_id, expires_at, consumed_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "code_challenge", "session_id", "expires_at", "consumed_at"}).
			AddRow(int64(11), 7, pkceS256(testCodeVerifier), "room-alpha", time.Now().UTC().Add(5*time.Minute), sql.NullTime{Time: past, Valid: true}))
	mock.ExpectRollback()
	mock.ExpectExec(`INSERT INTO auth_audit_events`).
		WithArgs("auth_desktop_exchange_failure", nil, sqlmock.AnyArg(), sqlmock.AnyArg(), nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := bytes.NewBufferString(`{"exchange_code":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","code_verifier":"` + testCodeVerifier + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/exchange", body)
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

func TestStartDesktopExchangeRequiresUserSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	AuthStore = authstore.New(db, 30*time.Minute, 24*time.Hour)
	cfg := &config.Config{
		DesktopExchangeCodeTTL:       2 * time.Minute,
		DesktopExchangeCallbackHosts: []string{"localhost"},
	}
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(func(c *gin.Context) {
		c.Set("auth_subject", "client:dev-platform")
		c.Next()
	})
	r.POST("/api/v1/auth/desktop/start", StartDesktopExchangeCode(cfg))

	body := bytes.NewBufferString(`{"session_id":"room-alpha","code_challenge":"` + pkceS256(testCodeVerifier) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/desktop/start", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", rr.Code, rr.Body.String())
	}
}
