package myhandlers

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/codemarked/go-lab/api/auth"
	"github.com/codemarked/go-lab/api/middleware"
	"github.com/codemarked/go-lab/api/requestid"
	"github.com/gin-gonic/gin"
)

const testJWTSecret = "01234567890123456789012345678901"

func setupRouterWithMockDB(t *testing.T) (*gin.Engine, sqlmock.Sqlmock, string, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatalf("token service: %v", err)
	}
	TokenSvc = ts

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed creating sqlmock: %v", err)
	}

	Database = &DBConn{Db: db}

	router := gin.New()
	router.Use(requestid.Middleware())
	v1 := router.Group("/api/v1")
	u := v1.Group("/users")
	u.Use(middleware.BearerOrSession(ts, nil, "", nil))
	{
		u.GET("", GetUsers)
		u.GET("/search", SearchUsers)
		u.GET("/:id", GetUserByID)
		u.POST("", CreateUser)
		u.PUT("/:id", middleware.RequireHumanUser(), UpdateUser)
		u.DELETE("/:id", middleware.RequireHumanUser(), DeleteUser)
	}

	tok, _, err := ts.MintAccessToken("client:test")
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
	}

	return router, mock, tok, cleanup
}

func userBearerToken(t *testing.T, ts *auth.TokenService, userID int) string {
	t.Helper()
	tok, _, err := ts.MintAccessToken("user:" + strconv.Itoa(userID))
	if err != nil {
		t.Fatalf("mint user token: %v", err)
	}
	return tok
}

func authHeader(token string) http.Header {
	h := make(http.Header)
	h.Set("Authorization", "Bearer "+token)
	return h
}

func unwrapUsers(t *testing.T, body []byte) []User {
	t.Helper()
	var env struct {
		Data []User `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	return env.Data
}

func unwrapUser(t *testing.T, body []byte) User {
	t.Helper()
	var env struct {
		Data User `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	return env.Data
}

func TestGetUsersReturnsUsers(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "name", "pennies"}).
		AddRow(1, "Alice", 55).
		AddRow(2, "Bob", 10)

	mock.ExpectQuery("SELECT id, name, pennies FROM users ORDER BY pennies DESC, id ASC").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header = authHeader(tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	users := unwrapUsers(t, rr.Body.Bytes())
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "Alice" || users[1].Name != "Bob" {
		t.Fatalf("unexpected users payload: %+v", users)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestGetUsersUnauthorizedWithoutToken(t *testing.T) {
	router, mock, _, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestCreateUserValidationFailure(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"name":"   ","pennies":12}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header = authHeader(tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestCreateUserSuccess(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	mock.ExpectExec("INSERT INTO users \\(name, pennies\\) VALUES \\(\\?, \\?\\)").
		WithArgs("Alice", 42).
		WillReturnResult(sqlmock.NewResult(11, 1))

	body := bytes.NewBufferString(`{"name":"Alice","pennies":42}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header = authHeader(tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rr.Code, rr.Body.String())
	}

	user := unwrapUser(t, rr.Body.Bytes())
	if user.ID != 11 || user.Name != "Alice" || user.Pennies != 42 {
		t.Fatalf("unexpected response user: %+v", user)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, name, pennies FROM users WHERE id = \\?").
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/999", nil)
	req.Header = authHeader(tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestDeleteUserSuccess(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	userTok := userBearerToken(t, TokenSvc, 1)

	mock.ExpectExec("DELETE FROM users WHERE id = \\?").
		WithArgs(7).
		WillReturnResult(driver.RowsAffected(1))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/7", nil)
	req.Header = authHeader(userTok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
	_ = tok
}

func TestDeleteUserForbiddenForClientSubject(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/7", nil)
	req.Header = authHeader(tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestUpdateUserSuccess(t *testing.T) {
	router, mock, _, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	userTok := userBearerToken(t, TokenSvc, 1)

	mock.ExpectExec("UPDATE users SET name = \\?, pennies = \\? WHERE id = \\?").
		WithArgs("Bob", 5, 3).
		WillReturnResult(driver.RowsAffected(1))

	body := bytes.NewBufferString(`{"name":"Bob","pennies":5}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/3", body)
	req.Header = authHeader(userTok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	u := unwrapUser(t, rr.Body.Bytes())
	if u.ID != 3 || u.Name != "Bob" || u.Pennies != 5 {
		t.Fatalf("unexpected user: %+v", u)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestUpdateUserForbiddenForClientSubject(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"name":"Bob","pennies":5}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/3", body)
	req.Header = authHeader(tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestSearchUsersEmptyQuery(t *testing.T) {
	router, mock, tok, cleanup := setupRouterWithMockDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search?name=%20", nil)
	req.Header = authHeader(tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	users := unwrapUsers(t, rr.Body.Bytes())
	if len(users) != 0 {
		t.Fatalf("expected empty list, got %d", len(users))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestJWTWrongSecretRejected(t *testing.T) {
	ts, err := auth.NewTokenService(testJWTSecret, "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	other, err := auth.NewTokenService(strings.Repeat("y", 32), "", "suite-platform", "suite-platform-api", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := ts.MintAccessToken("sub")
	if err != nil {
		t.Fatal(err)
	}
	_, err = other.ParseAccessToken(tok)
	if err == nil {
		t.Fatal("expected parse failure with wrong secret")
	}
}
