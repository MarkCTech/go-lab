# Platform API — integration overview

*Contract-first guide for TaskStack, Marble, and other HTTP clients. Treat the service as a documented HTTP API; the server implementation lives in this repository under the `api/` Go module.*

**Audience:** Engineers integrating TaskStack, Marble, or other suite components with this platform.

**Canonical contract:** [openapi.yaml](openapi.yaml) (validated in CI). This page summarizes behavior and links to topic guides.

**Planning and backlog:** [MASTER_PLAN.md](MASTER_PLAN.md) §7 (shipped), §9 (backlog). Short-form session notes: [CHAT_TODOS.md](../CHAT_TODOS.md).

---

## Base URL and versioning

- All product JSON routes are under **`/api/v1`**.
- Health checks: **`/healthz`** (liveness) and **`/readyz`** (database and optional migration version).
- Deployments supply the host (for example `https://api.example.com`). Clients should configure a **base URL** and append paths from this document or from OpenAPI.
- **Legacy paths:** requests under `/api/` that are not under `/api/v1` receive **410 Gone**. Clients must use `/api/v1` only.

---

## Response shape

- **Success:** `{ "data": <payload>, "meta": { "request_id": "..." } }`
- **Error:** `{ "error": { "code": "...", "message": "...", "details": ... }, "meta": { "request_id": "..." } }`

Per-operation schemas are defined in OpenAPI.

---

## Authentication modes

| Mode | When | Notes |
|------|------|--------|
| **Session cookie** | Browser flows (for example admin SPA or TaskStack web after login) | Issued by `POST /api/v1/auth/login`. Mutating requests with cookies require **CSRF**: the header (default `X-CSRF-Token`) must match the CSRF cookie. Use `GET /api/v1/auth/csrf` when needed. |
| **Bearer (HS256)** | API clients, desktop after exchange | Platform JWT from `POST /api/v1/auth/token` (`client_credentials`) or user access token from `POST /api/v1/auth/desktop/exchange`. |
| **Bearer (OIDC)** | When the deployment sets `OIDC_*` | RS256 access token; see [oidc-auth0.md](oidc-auth0.md). |

Bearer-authenticated **mutations** do **not** require CSRF (double-submit applies to browser session cookies only).

**Machine clients:** `POST /api/v1/auth/token` with `grant_type=client_credentials` yields a subject such as `client:<id>`. **`PUT` and `DELETE` on `/api/v1/users/{id}` require a human end-user subject** (`user:...`). Machine tokens receive **403** for those operations. See OpenAPI `x-requiresHumanSubject` where applicable.

**Bootstrap:** `POST /api/v1/auth/bootstrap` is a **development-oriented** bridge. Do **not** rely on it for production TaskStack or Marble flows — see [bootstrap-sunset.md](bootstrap-sunset.md).

---

## Endpoint index (OpenAPI `operationId`)

Use [openapi.yaml](openapi.yaml) for request and response bodies and security requirements.

| operationId | Method | Path | Typical consumer |
|-------------|--------|------|-------------------|
| healthz | GET | `/healthz` | Operations, probes |
| readyz | GET | `/readyz` | Orchestration, readiness gates |
| authRegister | POST | `/api/v1/auth/register` | TaskStack backend / signup |
| authLogin | POST | `/api/v1/auth/login` | Web login (cookie session) |
| authLogout | POST | `/api/v1/auth/logout` | Web |
| authRefresh | POST | `/api/v1/auth/refresh` | Web session renewal |
| authCsrf | GET | `/api/v1/auth/csrf` | Browser clients |
| authChangePassword | POST | `/api/v1/auth/change-password` | Authenticated human user |
| authJoinToken | POST | `/api/v1/auth/join-token` | Marble join handoff (human session or user Bearer) |
| authDesktopStart | POST | `/api/v1/auth/desktop/start` | Desktop login bridge (human user) |
| authDesktopExchange | POST | `/api/v1/auth/desktop/exchange` | Desktop application (PKCE verifier) |
| authToken | POST | `/api/v1/auth/token` | Machine-to-machine / `client_credentials` |
| authBootstrap | POST | `/api/v1/auth/bootstrap` | Development only |
| usersList | GET | `/api/v1/users` | Authenticated listing |
| usersCreate | POST | `/api/v1/users` | Server-side provisioning (often M2M); align with your authorization model |
| usersSearch | GET | `/api/v1/users/search` | Authenticated search |
| usersGetById | GET | `/api/v1/users/{id}` | Profile and identity linkage |
| usersUpdate | PUT | `/api/v1/users/{id}` | **Human user only** |
| usersDelete | DELETE | `/api/v1/users/{id}` | **Human user only** |

---

## Marble-oriented flow (Phase 5)

End-to-end sequence and security requirements: **[desktop-auth-bridge.md](desktop-auth-bridge.md)** (exchange codes, PKCE, callback host allowlist, join JWT).

**Game-side responsibility:** Verify **join** JWTs (`token_use=join` per contract) using platform keys and TTL. Validation logic is implemented in **Marble**; the API contract is defined in OpenAPI.

**Data boundaries:** [data-ownership.md](data-ownership.md).

---

## TaskStack-oriented notes

- Prefer **server-side** calls for signup and sensitive operations so **`PLATFORM_CLIENT_SECRET`** is never exposed to browsers.
- Same-origin cookie setups typically proxy `/api` to the API host — see [platform-admin-ui.md](platform-admin-ui.md) and the repository [README.md](../README.md).

---

## Configuration (integrators)

Clients need a **base URL** and deployment-specific secrets (for example OAuth client configuration for the desktop bridge, or M2M credentials for `auth/token`). Operators configure environment variables listed in [`.env.example`](../.env.example). Do not embed long-lived secrets in end-user client binaries.

---

## Prompt template (AI-assisted integration)

Optional: use the following when onboarding a coding assistant to implement a client without reading Go sources.

```text
You are implementing an HTTP client for the Marble / TaskStack platform API.

Contract (single source of truth for paths, methods, bodies, and errors):
  - OpenAPI 3 file at repository root: docs/openapi.yaml
  - Summary: docs/platform-api-consumer-brief.md

Rules:
  - All JSON API routes are under /api/v1. /healthz and /readyz are at the root.
  - Success: { data, meta }. Error: { error, meta }.
  - Browser session auth: cookie from POST /api/v1/auth/login; mutating requests need CSRF header matching CSRF cookie (see openapi info.description).
  - Bearer: HS256 from POST /api/v1/auth/token (client_credentials) or desktop exchange; OIDC RS256 when OIDC_* is configured.
  - PUT and DELETE /api/v1/users/{id} require a human user subject, not client:*.
  - Do not use POST /api/v1/auth/bootstrap in production integrations.
  - Desktop user login: docs/desktop-auth-bridge.md (PKCE, start, exchange, join-token as documented).
  - The game must verify token_use=join JWTs locally per contract; implementation belongs in the game repository.

Undocumented endpoints are out of scope; if it is not in openapi.yaml, it is not part of the public contract.
```
