// smoketest runs HTTP smoke checks against a running platform API (same scenarios as legacy scripts/test.ps1).
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

func main() {
	baseFlag := flag.String("base", "http://localhost:5000", "API base URL (no trailing slash)")
	clientID := flag.String("client-id", "dev-platform", "client_id for client_credentials")
	clientSecret := flag.String("client-secret", "change-me-dev-secret-min-length-16", "client_secret for client_credentials")
	flag.Parse()

	base := strings.TrimRight(strings.TrimSpace(*baseFlag), "/")
	apiURL, err := url.Parse(base)
	if err != nil || apiURL.Scheme == "" || apiURL.Host == "" {
		fmt.Fprintf(os.Stderr, "smoketest: invalid -base URL\n")
		os.Exit(1)
	}

	fmt.Printf("Running smoke tests against %s\n", base)

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

	newUser := map[string]any{"name": "SmokeAlice", "pennies": 77}
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
	must(created.Data.Name == "SmokeAlice", "created user name mismatch")
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
	searchURL := join(base, "/api/v1/users/search?name="+url.QueryEscape("SmokeAli"))
	code, searchRaw, err := do(httpClient, http.MethodGet, searchURL, nil, bearer)
	mustOK(err)
	must(code == http.StatusOK, "search should return 200")
	must(json.Unmarshal(searchRaw, &searchEnv) == nil, "search JSON")
	must(len(searchEnv.Data) >= 1, "search should return created user")

	update := map[string]any{"name": "SmokeAliceUpdated", "pennies": 99}
	ub, _ := json.Marshal(update)
	code, _, err = do(httpClient, http.MethodPut, putURL, bytes.NewReader(ub),
		mergeHdr(bearer, hdr("Content-Type", "application/json")))
	mustOK(err)
	must(code == http.StatusForbidden, "PUT /users/{id} with client_credentials token should return 403")

	code, _, err = do(httpClient, http.MethodDelete, putURL, nil, bearer)
	mustOK(err)
	must(code == http.StatusForbidden, "DELETE /users/{id} with client_credentials token should return 403")

	jar1, err := cookiejar.New(nil)
	mustOK(err)
	client1 := &http.Client{Jar: jar1, Timeout: 60 * time.Second}
	crudEmail := fmt.Sprintf("smokecrud+%d@example.com", time.Now().Unix())
	reg1 := map[string]string{"email": crudEmail, "password": "smoke-pass-8ch", "name": "SmokeCrud"}
	rb1, _ := json.Marshal(reg1)
	code, _, err = do(client1, http.MethodPost, join(base, "/api/v1/auth/register"), bytes.NewReader(rb1),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusCreated, "crud register should return 201")

	login1 := map[string]string{"email": crudEmail, "password": "smoke-pass-8ch"}
	lb1, _ := json.Marshal(login1)
	code, _, err = do(client1, http.MethodPost, join(base, "/api/v1/auth/login"), bytes.NewReader(lb1),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusOK, "crud login should return 200")

	csrf1 := cookieValue(jar1, apiURL, "gl_csrf")
	must(csrf1 != "", "crud login should set gl_csrf")

	code, putBody, err := do(client1, http.MethodPut, putURL, bytes.NewReader(ub),
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
	must(updated.Data.Name == "SmokeAliceUpdated", "update should change user name")
	must(updated.Data.Pennies == 99, "update should change pennies")

	code, _, err = do(client1, http.MethodDelete, putURL, nil, hdr("X-CSRF-Token", csrf1))
	mustOK(err)
	must(code == http.StatusNoContent, "DELETE should return 204 No Content")

	code, _, err = do(httpClient, http.MethodGet, putURL, nil, bearer)
	mustOK(err)
	must(code == http.StatusNotFound, "deleted user should return 404 on fetch")

	jar2, err := cookiejar.New(nil)
	mustOK(err)
	client2 := &http.Client{Jar: jar2, Timeout: 60 * time.Second}
	sessEmail := fmt.Sprintf("smoke+%d@example.com", time.Now().Unix())
	reg2 := map[string]string{"email": sessEmail, "password": "smoke-pass-8ch", "name": "SmokeSession"}
	rb2, _ := json.Marshal(reg2)
	code, _, err = do(client2, http.MethodPost, join(base, "/api/v1/auth/register"), bytes.NewReader(rb2),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusCreated, "register should return 201")

	login2 := map[string]string{"email": sessEmail, "password": "smoke-pass-8ch"}
	lb2, _ := json.Marshal(login2)
	code, _, err = do(client2, http.MethodPost, join(base, "/api/v1/auth/login"), bytes.NewReader(lb2),
		hdr("Content-Type", "application/json; charset=utf-8"))
	mustOK(err)
	must(code == http.StatusOK, "login should return 200")
	must(cookieValue(jar2, apiURL, "gl_session") != "", "login should set gl_session cookie")
	must(cookieValue(jar2, apiURL, "gl_csrf") != "", "login should set gl_csrf cookie")

	var csrfReady struct {
		Data struct {
			Ready bool `json:"csrf_ready"`
		} `json:"data"`
	}
	code, raw, err := do(client2, http.MethodGet, join(base, "/api/v1/auth/csrf"), nil, nil)
	mustOK(err)
	must(code == http.StatusOK, "GET /auth/csrf should return 200")
	must(json.Unmarshal(raw, &csrfReady) == nil, "csrf JSON")
	must(csrfReady.Data.Ready, "GET /auth/csrf should report csrf_ready")

	var usersCookie struct {
		Data []json.RawMessage `json:"data"`
	}
	code, ucRaw, err := do(client2, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusOK, "GET /users with session cookie should succeed")
	must(json.Unmarshal(ucRaw, &usersCookie) == nil, "cookie users JSON")
	must(len(usersCookie.Data) >= 1, "GET /users with session should return data")

	csrfOut := cookieValue(jar2, apiURL, "gl_csrf")
	must(csrfOut != "", "csrf cookie should exist before logout")
	code, _, err = do(client2, http.MethodPost, join(base, "/api/v1/auth/logout"), nil, hdr("X-CSRF-Token", csrfOut))
	mustOK(err)
	must(code == http.StatusOK, "logout should return 200")

	code, _, err = do(client2, http.MethodGet, join(base, "/api/v1/users"), nil, nil)
	mustOK(err)
	must(code == http.StatusUnauthorized, "after logout, GET /users with cookie should return 401")

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
