# Platform admin UI (Angular)

The SPA under [`client/`](../client/) is an operator console for platform-admin workflows in this repo.

## Scope

- Auth: login/session handling, invite acceptance, logout, refresh.
- Views: dashboard, users, players, characters, economy, cases, dataops, security, audit.
- Role-aware navigation and actions based on `GET /api/v1/security/me`.

It is not the full TaskStack control-plane product.

## Route usage

The SPA consumes `/api/v1` endpoints documented in [openapi.yaml](openapi.yaml), with privileged surfaces mapped in [platform-control-plane.md](platform-control-plane.md).

Primary high-risk areas:

- Cases workflow (`/api/v1/cases/*`)
- Backup/restore governance (`/api/v1/backups/*`)
- Support ack (`/api/v1/support/ack`)

## Session and security behavior

- Cookie mode is default and uses CSRF (`GET /api/v1/auth/csrf` bootstrap + mutating header).
- Refresh timer uses `POST /api/v1/auth/refresh`.
- On protected-request `401`, the app clears auth state and redirects to `/login`.
- Privileged mutations require `X-Platform-Action-Reason`; UI enforces minimum length before submit.

## Configuration pointers

- Repo setup and same-origin guidance: [README.md](../README.md)
- Auth/session details: [auth-session.md](auth-session.md)
- Environment variable catalog: [`.env.example`](../.env.example)
