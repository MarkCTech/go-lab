# Desktop / non-browser clients (auth bridge)

## Principles

- **No platform client secrets in browser bundles** (already enforced for the SPA).
- Desktop and automation clients are **not** cookie-first; they use **Bearer JWTs** or future opaque API tokens.

## Current contract (v0.1)

| Flow | Mechanism |
|------|-----------|
| Browser (TaskStack) | HttpOnly session cookie + CSRF for mutating calls |
| Automation / service | `POST /api/v1/auth/token` with `grant_type=client_credentials` (server-side only) |
| Temporary SPA bridge | `POST /api/v1/auth/bootstrap` (deprecated; see [bootstrap-sunset.md](bootstrap-sunset.md)) |

## Desktop applications

1. **Primary flow (implemented):** browser-assisted exchange-code bridge:
   - Authenticated user (cookie or user Bearer) calls `POST /api/v1/auth/desktop/start` with `session_id`, **`code_challenge`** (PKCE S256), and optional loopback `callback_uri`.
   - API returns one-time `exchange_code`.
   - Desktop calls `POST /api/v1/auth/desktop/exchange` with `exchange_code` + **`code_verifier`** and receives a short-lived user Bearer (`token_use=desktop_access`).
   - Trust hardening: callback URI must be `http` loopback host from `DESKTOP_EXCHANGE_CALLBACK_HOSTS`, without userinfo/fragment.
   - Exchange failures are audit-bucketed (`code_not_found`, `code_already_used`, `code_expired`, `code_verifier_mismatch`, etc.) for incident triage.
2. **Service and automation:** trusted processes may use `PLATFORM_CLIENT_*` with `POST /api/v1/auth/token` for machine-to-machine scenarios. Client credentials must not ship in browser bundles.

## Join-token bridge (Phase 5)

- **Implemented endpoint:** `POST /api/v1/auth/join-token` (requires authenticated **human** subject: session cookie+CSRF or user Bearer).
- **Request body:** `{ "session_id": "..." }` (`3..128` chars).
- **Response:** short-lived signed token + `expires_in` and echoed `session_id`.
- **Token intent:** JWT contains `token_use=join` and `join_session_id`; Marble-side handshake validation remains game-owned.
- **TTL control:** `JOIN_TOKEN_TTL_SECONDS` (default `120`, bounded to `30..1800`).

## JWT access

- Bearer tokens are **short-lived** (`JWT_ACCESS_TTL_SECONDS`). Desktop re-authenticates via exchange-code when expired; future refresh contract remains optional (`auth_refresh_tokens` table is reserved).

## Related

- [auth-session.md](auth-session.md) — cookies, CSRF, service token endpoint.
- [oidc-auth0.md](oidc-auth0.md) — external IdP Bearer validation.
- [jwt-rotation.md](jwt-rotation.md) — HS256 platform JWT rotation.
