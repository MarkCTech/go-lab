# Bootstrap auth sunset (`POST /api/v1/auth/bootstrap`)

## Goal

Remove the temporary bridge that mints JWTs for SPAs from an allowed `Origin` without user credentials, once every TaskStack (or other) deployment uses **cookie-based login** only.

## Suggested milestone

Pick one of:

- **Release tag:** e.g. disable bootstrap starting at `go-lab v0.3.0` (adjust to your versioning).
- **Date:** e.g. “no sooner than YYYY-MM-DD after SPA ships cookie login in production.”

Document the chosen milestone in your internal release notes.

## Operator checklist

1. **SPA:** Uses `POST /api/v1/auth/login` with `credentials: 'include'`, reads CSRF cookie and sends `CSRF_HEADER_NAME` on mutating requests (see [auth-session.md](auth-session.md)).
2. **CORS:** `CORS_ALLOWED_ORIGINS` lists every SPA origin that will send credentials.
3. **Cookies:** Behind HTTPS, set `SESSION_COOKIE_SECURE=true` (and `SameSite` appropriate for your topology).
4. **Config:** Set `AUTH_BOOTSTRAP_ENABLED=false` (or any non-empty value other than `1`/`true`/`yes`).
5. **Verify:** No client depends on `data.access_token` from bootstrap; smoke and integration tests pass without calling bootstrap.

## Rollback

If a legacy client still needs bootstrap briefly, set `AUTH_BOOTSTRAP_ENABLED=true` again, fix the client, then re-disable.
