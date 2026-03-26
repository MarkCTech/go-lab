# Platform API - consumer brief

Contract-first integration guide for TaskStack, Marble, and other clients.

## Canonical contract

- OpenAPI: [openapi.yaml](openapi.yaml)
- Planning status: [MASTER_PLAN.md](MASTER_PLAN.md)
- Ownership boundaries: [data-ownership.md](data-ownership.md)

If a route or schema is not in OpenAPI, treat it as out of contract.

## Invariants clients should assume

- Product API routes are versioned under `/api/v1`.
- Responses use envelope shape (`data` on success, `error` on failure, plus `meta.request_id`).
- `/api/*` routes outside `/api/v1` are legacy and return `410`.
- `/healthz` and `/readyz` are root-level endpoints.

## Auth modes

- Session cookie + CSRF: browser flows.
- HS256 bearer: `POST /api/v1/auth/token` for machine usage, desktop bearer from desktop exchange flow.
- OIDC bearer: enabled only when both `OIDC_ISSUER_URL` and `OIDC_AUDIENCE` are configured.

Human-only guard remains on `PUT`/`DELETE /api/v1/users/{id}`: machine (`client:*`) subjects are rejected.

## Desktop and game handoff

- Desktop login handoff uses:
  1. `POST /api/v1/auth/desktop/start`
  2. `POST /api/v1/auth/desktop/exchange`
  3. `POST /api/v1/auth/join-token`
- Marble validates join tokens game-side (`token_use=join`) per contract.

Details: [desktop-auth-bridge.md](desktop-auth-bridge.md).

## Integration boundaries

- TaskStack should call platform from server-side paths for sensitive operations.
- Do not ship `PLATFORM_CLIENT_SECRET` or equivalent long-lived secrets in browsers or client binaries.
- Platform API does not execute physical backup/restore actions; it governs approval workflow only.

## Related

- [platform-control-plane.md](platform-control-plane.md)
- [platform-admin-ui.md](platform-admin-ui.md)
- [migrations.md](migrations.md)
