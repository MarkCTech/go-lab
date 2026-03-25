# OIDC / Auth0 and platform API (Bearer validation)

## Purpose

**Opt-in OIDC** (e.g. Auth0): RS256 **access** JWTs in `Authorization: Bearer`, alongside HS256 (`/auth/token`) and cookie sessions. Enable only when **`OIDC_ISSUER_URL` and `OIDC_AUDIENCE` are both set** (or leave both empty).

**Clock:** `exp` / `nbf` via **go-oidc** (strict). Use **NTP** on API hosts; add **leeway** later if you see boundary 401s.

## Canonical `aud`

Tokens for **this** API must have **`aud` = `OIDC_AUDIENCE`** (Auth0: API Identifier). Document the value in env + here.

**Several APIs?** Prefer **one gateway `aud`**, or **separate Auth0 APIs + tokens**, or multi-aud only if IdP + validator both support it and you check **every** allowed `aud`. M2M calling **this** API should normally use the **same** `aud` unless the client only talks to another service.

## Identity is `(issuer, sub)` — not `sub` alone

- **`iss`:** issuer URL (normalized; Auth0 tenant issuer, e.g. `https://YOUR_DOMAIN.auth0.com/`).
- **`sub`:** subject string **unique only within that issuer**.

**Storage:** [`user_identities`](../migrations/) maps `(issuer, subject)` → local `users.id`. Lookups and JIT provisioning always use **both** fields. Never merge identities across issuers using `sub` alone.

## JIT provisioning vs account linking

**Current behavior (v1):** On first valid OIDC token for a new `(issuer, sub)`, the API creates a **new** local `users` row (minimal profile) and links `user_identities`. It does **not** auto-attach to an existing **email/password** account by email alone (account-takeover risk if IdP email is not strongly verified).

**Future linking (documented policy options):**

- Same verified email + **explicit** user action (e.g. “Connect Auth0” after password re-entry or magic link).
- Admin-only linking.
- Migration window: one-time password sign-in to attach `issuer`+`sub`.

Prefer a dedicated flow or `user_identities` rows over duplicate `users` rows. Formal policy: [adr-account-linking.md](adr-account-linking.md).

## M2M vs human (`auth_subject`)

- **Human** OIDC users resolve to **`user:<local_id>`** for handlers that expect end-user subjects (e.g. change-password).
- **Auth0 client-credentials** access tokens typically use `sub` like **`{clientId}@clients`**. The API maps these to **`client:<clientId>`** so they align with the existing **`client:`** convention from HS256 `POST /auth/token` and are **not** inserted into `users`.

**Human-only routes:** **`PUT` and `DELETE` `/api/v1/users/:id`** require an authenticated **`user:`** subject (session or user-scoped Bearer). **`client:*`** (HS256 `POST /auth/token` or Auth0 client-credentials mapped to `client:<id>`) receives **403** on those operations — see [openapi.yaml](openapi.yaml) (`x-requiresHumanSubject`). Other routes may still accept any authenticated subject until you add **scopes** or additional guards for TaskStack/Marble.

## Refresh tokens

- **Auth0 refresh tokens** are owned by the **client** (SPA SDK, desktop, BFF) and Auth0 — not stored in `auth_refresh_tokens` unless you design a BFF/token-exchange server-side.
- **`auth_refresh_tokens`** in our schema remains for **first-party** session/refresh evolution — keep concepts separate.

## Gateway vs app vs Redis

**Edge:** coarse per-IP, TLS, WAF. **App (+ Redis):** email lockout, route limits, shared replica counters. Don’t mirror the **same** numeric cap in both layers unless you mean to. **Redis errors:** fail open + log — [auth-session.md](auth-session.md).

## Env

[`.env.example`](../.env.example): `OIDC_ISSUER_URL`, `OIDC_AUDIENCE`. Future: `OIDC_JWKS_URL` override.

## Related

[Index](README.md) · [adr-account-linking.md](adr-account-linking.md) · [auth-session.md](auth-session.md) · [desktop-auth-bridge.md](desktop-auth-bridge.md) · [jwt-rotation.md](jwt-rotation.md)
