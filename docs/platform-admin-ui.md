# Platform admin UI (Angular in go-lab)

## Role

The Angular app under [`client/`](../client/) is an optional **self-host operator console** and API demo: login, dashboard, user directory CRUD. It is **not** the full TaskStack product control plane.

## Boundaries

| Piece | Owns |
|-------|------|
| **go-lab Angular** | Cookie (or dev bootstrap) auth against this repo’s API; admin-style navigation; local operator tasks. |
| **TaskStack** (separate product) | End-user accounts, orchestration, billing, broader control plane when integrated. |
| **Marble** | Game client/server simulation. |

## Configuration

See repo [README](../README.md) § Platform admin: `useBootstrapAuth`, **empty `apiBaseUrl`** + same-origin `/api` proxy (cookie sessions), CORS alignment with `CORS_ALLOWED_ORIGINS`.

## Session behavior

- After login (or on reload when `GET /auth/csrf` succeeds), the app starts a timer that calls **`POST /api/v1/auth/refresh`** on an interval (`sessionRefreshIntervalMs` in environment files). Align that interval with **`SESSION_IDLE_TTL_SECONDS`** on the API (stay safely under idle so the cookie session slides before expiry).
- **`UnauthorizedInterceptor`:** responses **401** from protected routes clear local auth state and navigate to **`/login`**. Expected 401s on login/register/bootstrap/token and the initial **`GET /auth/csrf`** probe do not trigger that redirect.
