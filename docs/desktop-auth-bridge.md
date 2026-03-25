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

## Desktop app (future-friendly)

1. **Preferred:** OAuth2-style or device-code flow is out of scope here; until then, desktop may:
   - Open a **system browser** to the web login page and capture a **short-lived exchange code** redirected to a localhost callback (design TBD), or
   - Prompt for **user email/password over TLS** and call `POST /api/v1/auth/login` **without** relying on cookies: would require a **non-cookie** response shape (e.g. tokens in body) gated by `client_id` / `Accept` / separate endpoint—**not implemented** in this repo yet to avoid putting tokens in the default browser path.

2. **Interim:** Run a **local agent** that holds `PLATFORM_CLIENT_*` and mints JWTs via `POST /api/v1/auth/token` for the desktop process only (secrets never ship inside the game binary in clear text; use OS keychain where possible).

## JWT access

- Bearer tokens are **short-lived** (`JWT_ACCESS_TTL_SECONDS`). Desktop should refresh by re-authenticating or using a future refresh contract (`auth_refresh_tokens` table is reserved).
