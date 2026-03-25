# Platform admin UI (Angular in go-lab)

## Role

The Angular app under [`client/`](../client/) is an optional **self-host operator console** and API demo: login, dashboard, user directory CRUD. It is **not** the full TaskStack product control plane.

## go-lab admin vs TaskStack (future consumer)

| Topic | go-lab Angular admin | TaskStack (suite) |
|-------|----------------------|-------------------|
| **Purpose** | Operate this deployment: try auth flows, manage `users` rows, smoke the platform API. | Product UX for accounts, workflows, and orchestration across suite services. |
| **Auth default** | Cookie session + CSRF against **`/api/v1`** on the same origin as the SPA (see README). | Expected to use the **same platform API** with user-centric flows (OIDC / session per product design); not limited to this Angular bundle. |
| **API contract** | Hand-written UI against JSON envelopes; formal contract is **[openapi.yaml](openapi.yaml)**. | Should generate clients or contract-test against that OpenAPI as the suite control plane matures. |
| **Machine vs human API use** | Logged-in operators use **`user:`** subjects (session). Dev **`client_credentials`** tokens can **list/create** users but **cannot** `PUT`/`DELETE` `/api/v1/users/{id}` (403) — those routes require a signed-in end-user. TaskStack backends that need service-level user lifecycle must either use a **human** delegated token or gain a **future** scoped service API if you add one explicitly. |

TaskStack remains a **separate repo/product**; this section only clarifies how the go-lab admin fits next to it so agents and operators do not conflate “admin SPA” with “full control plane.”

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

## Related

- [README.md](README.md) — documentation index.
