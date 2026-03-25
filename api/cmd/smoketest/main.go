// smoketest runs HTTP smoke checks against a running platform API (same scenarios as legacy scripts/test.ps1).
// Session flow also covers refresh, join-token, desktop PKCE exchange, and change-password (see session_extras.go).
//
// Each run uses a unique suffix (nanosecond) on user names and emails so repeated runs against a shared DB
// do not collide on fixed strings like "SmokeAlice", and so list/search assertions are not confused by leftovers.
// The machine-created user (M2M) is deleted during the flow; the single registered human account is deleted
// at the end after logout + re-login, so smoke does not leave standing registered users when the run succeeds.
//
// By default only loopback hosts are allowed for -base (see SMOKETEST_ALLOW_HOSTS in allowlist.go).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	smokePasswordInitial = "smoke-pass-8ch"
	smokePasswordChanged = "smoke-pass-9ch-xx"
)

func main() {
	baseFlag := flag.String("base", "http://localhost:5000", "API base URL (no trailing slash); host must match SMOKETEST_ALLOW_HOSTS (default loopback only)")
	clientID := flag.String("client-id", "dev-platform", "client_id for client_credentials")
	clientSecret := flag.String("client-secret", "change-me-dev-secret-min-length-16", "client_secret for client_credentials")
	flag.Parse()

	base := strings.TrimRight(strings.TrimSpace(*baseFlag), "/")
	apiURL, err := url.Parse(base)
	if err != nil || apiURL.Scheme == "" || apiURL.Host == "" {
		fmt.Fprintf(os.Stderr, "smoketest: invalid -base URL\n")
		os.Exit(1)
	}
	mustHostAllowed(apiURL.Hostname())

	runSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
	smokeEmail := fmt.Sprintf("smoke+%s@example.com", runSuffix)
	aliceName := fmt.Sprintf("SmokeAlice-%s", runSuffix)

	fmt.Printf("Running smoke tests against %s (run suffix %s)\n", base, runSuffix)

	httpClient := &http.Client{Timeout: 60 * time.Second}

	var h struct {
		Status string `json:"status"`
	}
	_, err = getJSON(httpClient, join(base, "/healthz"), &h)
	mustOK(err)
	must(h.Status == "ok", "healthz should return status=ok")

	var rdy struct {
		Status string `json:"status"`
	}
	_, err = getJSON(httpClient, join(base, "/readyz"), &rdy)
	mustOK(err)
	must(rdy.Status == "ready", "readyz should return status=ready")

	code, _, err := do(httpClient, http.MethodGet, join(base, "/api/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusGone, "legacy /api/users should return 410 Gone")

	code, _, err = do(httpClient, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusUnauthorized, "GET /api/v1/users without bearer should return 401")

	tokenBody := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     *clientID,
		"client_secret": *clientSecret,
	}
	tb, _ := json.Marshal(tokenBody)
	code, tokenRaw, err := do(httpClient, http.MethodPost, join(base, "/api/v1/auth/token"), bytes.NewReader(tb),
		hdr("Content-Type", "application/json"))
	mustOK(err)
	must(code == http.StatusOK, "auth/token should return 200")
	var tokenEnv struct {
		Data struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		} `json:"data"`
	}
	must(json.Unmarshal(tokenRaw, &tokenEnv) == nil, "token response JSON")
	must(len(tokenEnv.Data.AccessToken) > 10, "access_token should be present")
	must(tokenEnv.Data.TokenType == "Bearer", "token_type should be Bearer")
	bearer := hdr("Authorization", "Bearer "+tokenEnv.Data.AccessToken)

	newUser := map[string]any{"name": aliceName, "pennies": 77}
	nb, _ := json.Marshal(newUser)
	code, createRaw, err := do(httpClient, http.MethodPost, join(base, "/api/v1/users"), bytes.NewReader(nb),
		mergeHdr(bearer, hdr("Content-Type", "application/json")))
	mustOK(err)
	must(code == http.StatusCreated, "create user should return 201")
	var created struct {
		Data struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			Pennies int    `json:"pennies"`
		} `json:"data"`
	}
	must(json.Unmarshal(createRaw, &created) == nil, "create user JSON")
	must(created.Data.ID > 0, "created user should include numeric id")
	must(created.Data.Name == aliceName, "created user name mismatch")
	uid := created.Data.ID
	putURL := join(base, "/api/v1/users/"+strconv.FormatInt(uid, 10))

	var listEnv struct {
		Data []json.RawMessage `json:"data"`
	}
	code, listRaw, err := do(httpClient, http.MethodGet, join(base, "/api/v1/users"), nil, bearer)
	mustOK(err)
	must(code == http.StatusOK, "list users should return 200")
	must(json.Unmarshal(listRaw, &listEnv) == nil, "list users JSON")
	must(len(listEnv.Data) >= 1, "users list should have at least one user after create")

	var searchEnv struct {
		Data []json.RawMessage `json:"data"`
	}
	searchURL := join(base, "/api/v1/users/search?name="+url.QueryEscape(aliceName))
	code, searchRaw, err := do(httpClient, http.MethodGet, searchURL, nil, bearer)
	mustOK(err)
	must(code == http.StatusOK, "search should return 200")
	must(json.Unmarshal(searchRaw, &searchEnv) == nil, "search JSON")
	must(len(searchEnv.Data) >= 1, "search should return created user")

	aliceUpdatedName := aliceName + "Updated"
	update := map[string]any{"name": aliceUpdatedName, "pennies": 99}
	ub, _ := json.Marshal(update)
	code, _, err = do(httpClient, http.MethodPut, putURL, bytes.NewReader(ub),
		mergeHdr(bearer, hdr("Content-Type", "application/json")))
	mustOK(err)
	must(code == http.StatusForbidden, "PUT /users/{id} with client_credentials token should return 403")

	code, _, err = do(httpClient, http.MethodDelete, putURL, nil, bearer)
	mustOK(err)
	must(code == http.StatusForbidden, "DELETE /users/{id} with client_credentials token should return 403")

	humanJar, err := cookiejar.New(nil)
	mustOK(err)
	humanClient := &http.Client{Jar: humanJar, Timeout: 60 * time.Second}
	regBody := map[string]string{"email": smokeEmail, "password": smokePasswordInitial, "name": "SmokeRun"}
	rb1, _ := json.Marshal(regBody)
	code, regRaw, err := do(humanClient, http.MethodPost, join(base, "/api/v1/auth/register"), bytes.NewReader(rb1),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusCreated, "register should return 201")
	var regEnv struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	must(json.Unmarshal(regRaw, &regEnv) == nil, "register JSON")
	humanUserID := regEnv.Data.ID
	must(humanUserID > 0, "register should return user id")
	humanUserURL := join(base, "/api/v1/users/"+strconv.FormatInt(humanUserID, 10))

	login1 := map[string]string{"email": smokeEmail, "password": smokePasswordInitial}
	lb1, _ := json.Marshal(login1)
	code, _, err = do(humanClient, http.MethodPost, join(base, "/api/v1/auth/login"), bytes.NewReader(lb1),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusOK, "login should return 200")

	csrf1 := cookieValue(humanJar, apiURL, "gl_csrf")
	must(csrf1 != "", "login should set gl_csrf")

	code, putBody, err := do(humanClient, http.MethodPut, putURL, bytes.NewReader(ub),
		mergeHdr(hdr("Content-Type", "application/json"), hdr("X-CSRF-Token", csrf1)))
	mustOK(err)
	must(code == http.StatusOK, "PUT with session should return 200")
	var updated struct {
		Data struct {
			Name    string `json:"name"`
			Pennies int    `json:"pennies"`
		} `json:"data"`
	}
	must(json.Unmarshal(putBody, &updated) == nil, "PUT response JSON")
	must(updated.Data.Name == aliceUpdatedName, "update should change user name")
	must(updated.Data.Pennies == 99, "update should change pennies")

	code, _, err = do(humanClient, http.MethodDelete, putURL, nil, hdr("X-CSRF-Token", csrf1))
	mustOK(err)
	must(code == http.StatusNoContent, "DELETE should return 204 No Content")

	code, _, err = do(httpClient, http.MethodGet, putURL, nil, bearer)
	mustOK(err)
	must(code == http.StatusNotFound, "deleted user should return 404 on fetch")

	must(cookieValue(humanJar, apiURL, "gl_session") != "", "session cookie should exist before auth extras")
	must(cookieValue(humanJar, apiURL, "gl_csrf") != "", "csrf cookie should exist before auth extras")

	var csrfReady struct {
		Data struct {
			Ready bool `json:"csrf_ready"`
		} `json:"data"`
	}
	code, raw, err := do(humanClient, http.MethodGet, join(base, "/api/v1/auth/csrf"), nil, nil)
	mustOK(err)
	must(code == http.StatusOK, "GET /auth/csrf should return 200")
	must(json.Unmarshal(raw, &csrfReady) == nil, "csrf JSON")
	must(csrfReady.Data.Ready, "GET /auth/csrf should report csrf_ready")

	var usersCookie struct {
		Data []json.RawMessage `json:"data"`
	}
	code, ucRaw, err := do(humanClient, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusOK, "GET /users with session cookie should succeed")
	must(json.Unmarshal(ucRaw, &usersCookie) == nil, "cookie users JSON")
	must(len(usersCookie.Data) >= 1, "GET /users with session should return data")

	runAuthSessionExtras(humanClient, humanJar, base, apiURL, smokeEmail, smokePasswordInitial, smokePasswordChanged)

	csrfOut := cookieValue(humanJar, apiURL, "gl_csrf")
	must(csrfOut != "", "csrf cookie should exist before logout")
	code, _, err = do(humanClient, http.MethodPost, join(base, "/api/v1/auth/logout"), nil, hdr("X-CSRF-Token", csrfOut))
	mustOK(err)
	must(code == http.StatusOK, "logout should return 200")

	code, _, err = do(humanClient, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusUnauthorized, "after logout, GET /users with cookie should return 401")

	// Re-authenticate so we can delete the smoke account row (no orphan registered users on success).
	relogin := map[string]string{"email": smokeEmail, "password": smokePasswordChanged}
	reloginBody, _ := json.Marshal(relogin)
	code, _, err = do(humanClient, http.MethodPost, join(base, "/api/v1/auth/login"), bytes.NewReader(reloginBody),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusOK, "re-login cleanup should return 200")
	csrfDel := cookieValue(humanJar, apiURL, "gl_csrf")
	must(csrfDel != "", "csrf should exist after re-login for cleanup delete")
	code, _, err = do(humanClient, http.MethodDelete, humanUserURL, nil, hdr("X-CSRF-Token", csrfDel))
	mustOK(err)
	must(code == http.StatusNoContent, "cleanup DELETE smoke user should return 204")

	code, _, err = do(humanClient, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusUnauthorized, "after deleting user, GET /users with cookie should return 401")

	bootHdr := mergeHdr(hdr("Content-Type", "application/json"), hdr("Origin", "http://localhost:4200"))
	code, bootRaw, err := do(httpClient, http.MethodPost, join(base, "/api/v1/auth/bootstrap"), strings.NewReader("{}"), bootHdr)
	mustOK(err)
	must(code == http.StatusOK, "bootstrap should return 200")
	var boot struct {
		Data struct {
			Bootstrap *struct {
				Temporary bool `json:"temporary"`
			} `json:"bootstrap"`
		} `json:"data"`
	}
	must(json.Unmarshal(bootRaw, &boot) == nil, "bootstrap JSON")
	must(boot.Data.Bootstrap != nil, "bootstrap should include data.bootstrap deprecation object")
	must(boot.Data.Bootstrap.Temporary, "bootstrap.temporary should be true")

	fmt.Println("Smoke tests passed.")
}

func do(client *http.Client, method, urlStr string, body io.Reader, h http.Header) (int, []byte, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return 0, nil, err
	}
	if h != nil {
		req.Header = h.Clone()
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, b, err
}

func getJSON(client *http.Client, urlStr string, out any) (int, error) {
	code, raw, err := do(client, http.MethodGet, urlStr, nil, nil)
	if err != nil {
		return 0, err
	}
	if code != http.StatusOK {
		return code, fmt.Errorf("status %d", code)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return code, err
	}
	return code, nil
}

func join(base, path string) string {
	u, err := url.Parse(base)
	if err != nil {
		panic(err)
	}
	rel, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.ResolveReference(rel).String()
}

func hdr(k, v string) http.Header {
	h := http.Header{}
	h.Set(k, v)
	return h
}

func mergeHdr(a, b http.Header) http.Header {
	out := http.Header{}
	for k, vv := range a {
		out[k] = append([]string(nil), vv...)
	}
	for k, vv := range b {
		out[k] = append(out[k], vv...)
	}
	return out
}

func cookieValue(jar *cookiejar.Jar, api *url.URL, name string) string {
	for _, c := range jar.Cookies(api) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func must(cond bool, msg string) {
	if !cond {
		fmt.Fprintf(os.Stderr, "ASSERT FAILED: %s\n", msg)
		os.Exit(1)
	}
}

func mustOK(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "smoketest: %v\n", err)
		os.Exit(1)
	}
}
