# Platform admin UI (Angular in go-lab)

## Role

The Angular app under [`client/`](../client/) is an optional **operator console**: login, dashboard, user directory CRUD, and **Phase A** nav (Players, Characters, DataOps, Security, Audit) calling gated `/api/v1/*` routes. It is **not** the full suite control-plane UX — positioning vs TaskStack/Marble: [MASTER_PLAN.md](MASTER_PLAN.md), [data-ownership.md](data-ownership.md), [platform-api-consumer-brief.md](platform-api-consumer-brief.md).

## Configuration

See repo [README](../README.md) § Platform admin: `useBootstrapAuth`, **`apiBaseUrl: ''`** + same-origin proxy to `/api`, CORS vs `CORS_ALLOWED_ORIGINS`.

## Phase A navigation (this SPA)

- **Players / Characters / DataOps:** read-only JSON views; `GET` via [`platform.service.ts`](../client/src/app/platform.service.ts) with permissions enforced server-side (see [platform-control-plane.md](platform-control-plane.md)).
- **Security:** `GET /api/v1/security/me`; support ack **`POST /api/v1/support/ack`** with header **`X-Platform-Action-Reason`** (min length enforced server-side; UI requires ≥ 10 chars before submit).
- **Audit:** `GET /api/v1/audit/admin-events`.

**Grant platform roles in SQL** — [platform-operator-roles.md](platform-operator-roles.md).

## Session behavior

- After login, or when **`GET /api/v1/auth/csrf`** succeeds on reload, the app timers **`POST /api/v1/auth/refresh`** on `sessionRefreshIntervalMs` (cookie mode; skipped when `useBootstrapAuth` is true). Keep the interval safely under **`SESSION_IDLE_TTL_SECONDS`**.
- **`UnauthorizedInterceptor`:** **401** on protected API calls clears auth and navigates to **`/login`** (with `session=expired` query param). **401** on login/register/bootstrap/token and on the initial **`GET /api/v1/auth/csrf`** probe is ignored.

## Related

- [platform-control-plane.md](platform-control-plane.md) — RBAC matrix, route ↔ permission.
- [openapi.yaml](openapi.yaml) — contract.
