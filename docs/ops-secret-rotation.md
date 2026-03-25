# Operations: secret rotation checklist

Use this as an operator checklist when rotating credentials. Details for HS256 JWT rotation live in [jwt-rotation.md](jwt-rotation.md).

## HS256 platform JWT (`JWT_SECRET`)

1. Generate a new secret (≥32 characters).
2. Set `**JWT_SECRET_PREVIOUS**` to the current `**JWT_SECRET**`, then set `**JWT_SECRET**` to the new value.
3. Redeploy / restart all API instances so both secrets verify during overlap.
4. After `**JWT_ACCESS_TTL_SECONDS**` (plus a small buffer), remove `**JWT_SECRET_PREVIOUS**` so only the new secret signs tokens.
5. Confirm `**POST /api/v1/auth/token**` and existing Bearer clients re-fetch tokens as needed.

See [jwt-rotation.md](jwt-rotation.md).

## Platform client credentials (`PLATFORM_CLIENT_ID` / `PLATFORM_CLIENT_SECRET`)

Used for `**POST /api/v1/auth/token**` (client_credentials). Rotate `**PLATFORM_CLIENT_SECRET**` in config; update every caller that exchanges credentials. Prefer zero-downtime overlap only if you run duplicate client IDs (not typical)—usually a coordinated cutover is enough.

## OIDC / Auth0 (RS256 access tokens)

- **Signing keys:** Rotated on the IdP; go-oidc/JWKS picks up new keys automatically when discovery/JWKS is reachable.
- **Audience / issuer:** Changing `**OIDC_AUDIENCE`** or `**OIDC_ISSUER_URL**` is a **contract change**—update all issuers and consumers together.
- Optional `**OIDC_JWKS_URL`** override (see [MASTER_PLAN.md](MASTER_PLAN.md) open questions) for locked-down networks.

See [oidc-auth0.md](oidc-auth0.md).

## Session and bootstrap

- **Session invalidation:** Password change revokes all sessions for that user; global secret rotation does not invalidate opaque session tokens in DB (they remain valid until idle/absolute expiry or explicit logout).
- **Bootstrap:** Disable with `**AUTH_BOOTSTRAP_ENABLED=false`** when no longer needed ([bootstrap-sunset.md](bootstrap-sunset.md)).

## Audit events

Auth-related audit rows (`auth_audit_events`) support post-incident review. For taxonomy expansion, align new `event_type` strings with existing usage in `authstore` / handlers before deploying.

## Related

- [MASTER_PLAN.md](MASTER_PLAN.md) — backlog and decisions.
- [ci.md](ci.md) — automation.

