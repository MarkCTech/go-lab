# Auth and sessions (platform API)

## Dependencies

- **`golang.org/x/crypto`** (direct in `go_CRUD_api/go.mod`) — Argon2id password hashing. No extra password-hash library; parameters live in `auth/password.go`.

## Password policy

- Algorithm: **Argon2id** with fixed memory/time/thread parameters (see `auth.PasswordPolicySummary()` and comments in `auth/password.go`).
- Passwords are never logged or returned in APIs.
- Changing Argon2 parameters later: existing hashes with old parameters still verify until you implement a re-hash-on-login upgrade path (hashes encode their own `m,t,p` and mismatches are rejected).

## Browser sessions

- **Login:** `POST /api/v1/auth/login` with JSON `{ "email", "password" }` sets an **HttpOnly** session cookie (name from `SESSION_COOKIE_NAME`, default `gl_session`).
- **Cookie flags:** `HttpOnly` always; **`Secure`** from `SESSION_COOKIE_SECURE` (default **true** when unset — set **`false`** for plain HTTP dev/Compose as in `.env.example`). **`SameSite`** from `SESSION_SAMESITE` (`Lax` default).
- **Expiry:** Sliding idle window (`SESSION_IDLE_TTL_SECONDS`) capped by absolute lifetime (`SESSION_ABSOLUTE_TTL_SECONDS`). Enforced in `authstore.Store.ValidateSession` (also used by `POST /api/v1/auth/refresh`).
- **Logout:** `POST /api/v1/auth/logout` revokes the current session server-side and clears session + CSRF cookies.
- **Do not** put access tokens or refresh secrets in frontend bundles; use the cookie + `credentials: 'include'` (or equivalent). The SPA must be served **from the same site** as the API paths it calls (e.g. reverse-proxy `/api` on the SPA host): **cross-port** `localhost:4200` → `localhost:5000` XHR does not send `SameSite=Lax` cookies. See repo README § Platform admin (nginx + `proxy.conf.json`).
- **CORS:** `CORS_ALLOWED_ORIGINS` still applies when the browser sends an `Origin` (e.g. some POSTs); keep your SPA origin listed. Pure same-origin fetches do not rely on CORS.

## CSRF (cookie-authenticated mutating requests)

For **POST, PUT, PATCH, DELETE** when the client does **not** send `Authorization: Bearer`, the API enforces **double-submit** protection:

1. **Cookie:** `CSRF_COOKIE_NAME` (default `gl_csrf`) — **not** HttpOnly so the SPA can read it.
2. **Header:** must match the cookie value; header name from `CSRF_HEADER_NAME` (default `X-CSRF-Token`).

**Skipped when:**

- Request uses **Bearer** JWT (automation / service clients).
- Method is safe (GET, HEAD, OPTIONS, TRACE).
- Path is an exempt auth bootstrap: `POST /auth/register`, `/auth/login`, `/auth/token`, `/auth/bootstrap`.
- `POST /auth/logout` with **no** session cookie (no-op logout).

**Issuing / rotating the CSRF cookie**

- Set on successful **login** and **refresh**.
- Cleared on **logout** and **password change** (which revokes all sessions).
- **`GET /api/v1/auth/csrf`:** validates the session cookie and sets a fresh CSRF cookie (for SPAs that have a session but lost the CSRF pair).

CORS preflight must allow the CSRF header: `DynamicCORS` appends `CSRF_HEADER_NAME` to `Access-Control-Allow-Headers`. For cookie sessions the SPA uses credentialed XHR/fetch; the API sets **`Access-Control-Allow-Credentials: true`** together with a concrete **`Access-Control-Allow-Origin`** (never `*`) for allowlisted origins.

## Auth abuse and rate limits

Per-IP **fixed 1-minute windows** (separate limiter instances; defaults in code):

| Route group | Approx. limit / minute | Error message |
|-------------|------------------------|---------------|
| `POST /auth/register` | 15 | too many registration attempts |
| `POST /auth/login` | 30 | too many login attempts |
| `POST /auth/logout` | 60 | too many logout attempts |
| `POST /auth/refresh` | 120 | too many session refresh attempts |
| `GET /auth/csrf` | 60 | too many csrf requests |
| `POST /auth/change-password` | 10 | too many password change attempts |
| `POST /auth/token`, `/auth/bootstrap` | 30 | too many token requests |

**Per-email lockout (failed password):** after **5** failed attempts for a **known** email (user exists), that email is blocked for **15 minutes** (process-local memory only). Unknown emails are not lockout-keyed (reduces account-DoS). Tuned in `myhandlers/login_throttle.go`.

**Multi-instance:** All of the above are **in-memory per process**. For several API replicas behind a load balancer, enforce complementary limits at the **gateway** or use a shared store (e.g. Redis) in a future iteration.

## Password change

- **`POST /api/v1/auth/change-password`** — JSON `{ "current_password", "new_password" }`. Requires authenticated **user** subject (`user:<id>` from session or Bearer JWT with that subject). **Revokes all sessions** for the user and clears cookies; client must log in again.

## JWT / Bearer (service and legacy clients)

- `POST /api/v1/auth/token` (client credentials) mints short-lived JWTs.
- `GET/POST/... /api/v1/users` accepts **either** `Authorization: Bearer <jwt>` **or** a valid session cookie. If a Bearer header is present, the cookie is ignored for that request.
- **Rotation:** see [jwt-rotation.md](jwt-rotation.md) (`JWT_SECRET_PREVIOUS`).

## Bootstrap bridge (temporary)

- `POST /api/v1/auth/bootstrap` remains for SPAs that still need a JWT without embedding platform secrets. It requires a browser **`Origin`** header on the CORS allowlist.
- Responses include a `data.bootstrap` object marking **temporary / deprecated** migration toward cookie login.
- Sunset checklist: [bootstrap-sunset.md](bootstrap-sunset.md).

## Desktop / automation

- See [desktop-auth-bridge.md](desktop-auth-bridge.md).

## Optional metadata

- **`JWT_ACTIVE_KEY_ID`:** Logged at startup; see [jwt-rotation.md](jwt-rotation.md).

## Schema

- See migration `000002_*`: `users` auth columns, `auth_sessions`, `auth_refresh_tokens` (reserved for future refresh-token rotation), `auth_audit_events`.
