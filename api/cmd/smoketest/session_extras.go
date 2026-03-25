package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// runAuthSessionExtras exercises refresh, join-token, desktop PKCE exchange, and change-password
// for an http.Client that already completed register+login (session + CSRF cookies set).
func runAuthSessionExtras(client *http.Client, jar *cookiejar.Jar, base string, apiURL *url.URL, email, oldPassword, newPassword string) {
	csrfTok := cookieValue(jar, apiURL, "gl_csrf")
	must(csrfTok != "", "extras: expected gl_csrf after login")

	code, raw, err := do(client, http.MethodPost, join(base, "/api/v1/auth/refresh"), nil,
		hdr("X-CSRF-Token", csrfTok))
	mustOK(err)
	must(code == http.StatusOK, "POST /auth/refresh should return 200")
	var refreshEnv struct {
		Data struct {
			Refreshed bool `json:"refreshed"`
			UserID    int  `json:"user_id"`
		} `json:"data"`
	}
	must(json.Unmarshal(raw, &refreshEnv) == nil, "refresh JSON")
	must(refreshEnv.Data.Refreshed, "refresh should set refreshed=true")
	must(refreshEnv.Data.UserID > 0, "refresh should echo user_id")
	csrfTok = cookieValue(jar, apiURL, "gl_csrf")
	must(csrfTok != "", "extras: gl_csrf after refresh")

	joinBody := map[string]string{"session_id": "smoke-join-session"}
	jb, _ := json.Marshal(joinBody)
	code, jraw, err := do(client, http.MethodPost, join(base, "/api/v1/auth/join-token"), bytes.NewReader(jb),
		mergeHdr(hdr("Content-Type", "application/json"), hdr("X-CSRF-Token", csrfTok)))
	mustOK(err)
	must(code == http.StatusOK, "join-token should return 200")
	var joinEnv struct {
		Data struct {
			JoinToken string `json:"join_token"`
			TokenType string `json:"token_type"`
			ExpiresIn int    `json:"expires_in"`
			SessionID string `json:"session_id"`
		} `json:"data"`
	}
	must(json.Unmarshal(jraw, &joinEnv) == nil, "join-token JSON")
	must(joinEnv.Data.JoinToken != "", "join_token should be non-empty")
	must(joinEnv.Data.TokenType == "Bearer", "join token_type should be Bearer")
	must(joinEnv.Data.SessionID == "smoke-join-session", "join-token should echo session_id")
	must(joinEnv.Data.ExpiresIn >= 0, "join-token expires_in should be present")

	verifier, challenge, err := randomPKCEPair()
	mustOK(err)
	startBody := map[string]string{
		"session_id":     "smoke-desktop-sess",
		"code_challenge": challenge,
		"callback_uri":   "http://127.0.0.1:8765/callback",
	}
	sb, _ := json.Marshal(startBody)
	code, sraw, err := do(client, http.MethodPost, join(base, "/api/v1/auth/desktop/start"), bytes.NewReader(sb),
		mergeHdr(hdr("Content-Type", "application/json"), hdr("X-CSRF-Token", csrfTok)))
	mustOK(err)
	must(code == http.StatusOK, "desktop/start should return 200")
	var startEnv struct {
		Data struct {
			ExchangeCode string `json:"exchange_code"`
			SessionID    string `json:"session_id"`
		} `json:"data"`
	}
	must(json.Unmarshal(sraw, &startEnv) == nil, "desktop/start JSON")
	must(len(startEnv.Data.ExchangeCode) >= 64, "exchange_code should be present")
	must(startEnv.Data.SessionID == "smoke-desktop-sess", "desktop/start should echo session_id")

	// Exchange is unauthenticated; use a fresh client (no session jar).
	exClient := &http.Client{Timeout: client.Timeout}
	exBody := map[string]string{
		"exchange_code": startEnv.Data.ExchangeCode,
		"code_verifier": verifier,
	}
	eb, _ := json.Marshal(exBody)
	code, eraw, err := do(exClient, http.MethodPost, join(base, "/api/v1/auth/desktop/exchange"), bytes.NewReader(eb),
		hdr("Content-Type", "application/json"))
	mustOK(err)
	must(code == http.StatusOK, "desktop/exchange should return 200")
	var exEnv struct {
		Data struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			TokenUse    string `json:"token_use"`
		} `json:"data"`
	}
	must(json.Unmarshal(eraw, &exEnv) == nil, "desktop/exchange JSON")
	must(exEnv.Data.AccessToken != "", "desktop access_token should be present")
	must(exEnv.Data.TokenType == "Bearer", "desktop token_type should be Bearer")
	must(exEnv.Data.TokenUse == "desktop_access", "desktop token_use should be desktop_access")
	desktopBearer := hdr("Authorization", "Bearer "+exEnv.Data.AccessToken)
	code, _, err = do(exClient, http.MethodGet, join(base, "/api/v1/users"), nil, desktopBearer)
	mustOK(err)
	must(code == http.StatusOK, "GET /users with desktop bearer should succeed")

	cpBody := map[string]string{"current_password": oldPassword, "new_password": newPassword}
	cpb, _ := json.Marshal(cpBody)
	code, _, err = do(client, http.MethodPost, join(base, "/api/v1/auth/change-password"), bytes.NewReader(cpb),
		mergeHdr(hdr("Content-Type", "application/json"), hdr("X-CSRF-Token", csrfTok)))
	mustOK(err)
	must(code == http.StatusOK, "change-password should return 200")

	code, _, err = do(client, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusUnauthorized, "after change-password, old session cookie should be invalid (401)")

	lb := map[string]string{"email": email, "password": newPassword}
	lbb, _ := json.Marshal(lb)
	code, _, err = do(client, http.MethodPost, join(base, "/api/v1/auth/login"), bytes.NewReader(lbb),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusOK, "login with new password should return 200")

	code, _, err = do(client, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusOK, "after re-login, GET /users with cookie should succeed")
}

func randomPKCEPair() (verifier string, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("rand: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	if len(verifier) != 43 {
		return "", "", fmt.Errorf("unexpected verifier length %d", len(verifier))
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	if len(challenge) != 43 {
		return "", "", fmt.Errorf("unexpected challenge length %d", len(challenge))
	}
	return verifier, challenge, nil
}
