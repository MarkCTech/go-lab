# Master plan — Product suite + go-lab (platform API)

**Single planning doc:** specs (summary), decisions, shipped matrix, phases, prioritized backlog. Edit when priorities change; deep detail stays in `docs/*.md`.

**Related:** [Documentation index](README.md) · [Repo README](../README.md)

---

## Table of contents

1. [Executive snapshot](#1-executive-snapshot)
2. [Product Suite Architecture Spec v0.1 (condensed)](#2-product-suite-architecture-spec-v01-condensed)
3. [Go-lab platform API (this repo)](#3-go-lab-platform-api-this-repo)
4. [Go-lab repo — implementation reality](#4-go-lab-repo--implementation-reality)
5. [Version stance (suite vs repo)](#5-version-stance-suite-vs-repo)
6. [Architecture decisions (consensus)](#6-architecture-decisions-consensus)
7. [Shipped vs still open](#7-shipped-vs-still-open)
8. [Roadmap phases](#8-roadmap-phases)
9. [Prioritized backlog](#9-prioritized-backlog)
10. [Open questions](#10-open-questions)
11. [Constraints & ground rules](#11-constraints--ground-rules)
12. [Agent handoff & priming](#12-agent-handoff--priming)
13. [Maintenance notes](#13-maintenance-notes)

---

## 1. Executive snapshot


| Item                      | State                                                                                                                                                                                                                                      |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Products**              | **Marble** (game sim), **TaskStack** (control-plane UX), **Go-lab** (this repo — platform API + admin SPA + migrations)                                                                                                                    |
| **Suite spec maturity**   | **v0.1** (foundation; not full prod hardening of every subsystem)                                                                                                                                                                          |
| **Auth / scale**          | Cookie + CSRF SPA; HS256 `/auth/token`; optional OIDC + `user_identities`; optional **Redis** for limits/lockout                                                                                                                           |
| **Gaps vs quality gates** | **Phase 5 suite consumption** (Marble/TaskStack use exchange + join-token; game validates join JWT); optional polish: OpenAPI↔route drift, OIDC leeway, trusted proxy / real client IP behind TLS, audit `event_type` taxonomy, scoped M2M |


**Login UX goals (current simplification):**

- **Browser web:** cookie session + CSRF is the default and preferred path.
- **Desktop Marble:** browser-assisted exchange-code -> desktop Bearer -> join-token.
- **Service/automation:** `client_credentials` only (`client:*`), scoped over time.
- **Singleplayer/local:** no required platform login; optional account-link for cloud features later.
- **Bootstrap bridge:** deprecated; disable with `AUTH_BOOTSTRAP_ENABLED=false` once legacy callers are migrated.

---

## 2. Product Suite Architecture Spec v0.1 (condensed)

**Marble** (game authority) · **TaskStack** (suite control-plane UX, consumes API) · **Go-lab** (this repo: API + admin SPA + migrations). **Principles:** additive platform, multi-repo contracts, Compose-first. **Schema:** migrations only; **API:** `/api/v1`, contract [openapi.yaml](openapi.yaml). **Non-goals (v0.1):** microservices split, enterprise SSO, global matchmaking, full streaming stack.

**Deep dive:** [data-ownership.md](data-ownership.md). **Phase A RBAC + routes:** [platform-control-plane.md](platform-control-plane.md). **Execution backlog:** §9.

---

## 3. Go-lab platform API (this repo)

Self-hostable **Go API**: users, auth/session, tokens, health/readiness. **Owns:** user data, sessions (evolving), future entitlements. **Does not own:** game sim, rendering, physics.

**Ops:** Compose; env-only config; **no runtime DDL**; `go test ./...` + smoke + CI.

---

## 4. Go-lab repo — implementation reality


| Area           | Location / notes                                                                                                                                                                              |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Backend**    | `api/` — Gin, `middleware/`, `auth/`, `authstore/`, `myhandlers/`                                                                                                                             |
| **Frontend**   | `client/` — Angular admin SPA                                                                                                                                                                 |
| **Migrations** | `migrations/` — full chain `000001`–`000005`: [migrations.md](migrations.md) |
| **Compose**    | `docker-compose.yml` — mysql, migrate, backend, frontend; optional **`redis`** profile                                                                                                        |
| **Docs**       | Topic guides under `docs/`; index [README.md](README.md); CI summary [ci.md](ci.md)                                                                                                           |
| **Key env**    | See `[.env.example](../.env.example)`: `JWT_*`, `JOIN_TOKEN_TTL_SECONDS`, `DESKTOP_EXCHANGE_*`, `SESSION_*`, `OIDC_*`, `REDIS_URL`, `MIGRATION_EXPECTED_VERSION`, CSRF, platform client creds |
| **Operator RBAC** | DB: [platform-operator-roles.md](platform-operator-roles.md). Boundaries + matrix: [platform-control-plane.md](platform-control-plane.md). |


---

## 5. Version stance (suite vs repo)

Suite spec **v0.1** until bumped. **Go-lab:** v0.1 + auth/OIDC + Phase 4 contracts + **Phase 5 platform auth bridge shipped** (join-token, desktop exchange + PKCE, `000004_*`). **Next focus:** suite repos consume those APIs; gameplay trust + data plane per [data-ownership.md](data-ownership.md).

---

## 6. Architecture decisions (consensus)


| Topic                          | Decision                                                                                                                                                                                                                            |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Identity source of truth**   | **Hybrid:** first-party email/password + opaque session cookie for admin SPA; local `users` + `user_identities` for `(issuer, sub)`. Optional OIDC access JWT as Bearer when `OIDC_*` set. Platform owns authz/tenancy/audit in DB. |
| **Auth0-only vs API-only**     | Prefer hybrid. Pure Auth0-only only if API is a thin BFF with almost no local user model.                                                                                                                                           |
| **TaskStack / Marble**         | TaskStack: **consumer** of platform APIs. Marble: gameplay authority; join/session trust **later** (Phase 5). Boundaries + performance/sync design: [data-ownership.md](data-ownership.md).                                         |
| **Canonical `aud` (OIDC)**     | One API identifier: **`OIDC_AUDIENCE`**. Do not conflate with **`JWT_AUDIENCE`** unless deliberately standardized and documented. See [oidc-auth0.md](oidc-auth0.md).                                                               |
| **Multiple `aud` / multi-API** | Prefer gateway as single `aud`, or separate Auth0 APIs/tokens, or multi-aud with explicit validation only.                                                                                                                          |
| **`sub`**                      | Identity = **`(issuer, sub)`** only; `user_identities` enforces.                                                                                                                                                                    |
| **Account linking**            | No naive email auto-link. Safe: verified email + explicit action, admin linking, or migration “password once to attach.” JIT new user on first OIDC `(iss, sub)` today.                                                             |
| **Refresh tokens**             | Auth0 refresh = client/Auth0/BFF; do not stuff into `auth_refresh_tokens` without naming flow ownership.                                                                                                                            |
| **Cookie vs Bearer**           | Shipped default: HttpOnly cookie + CSRF for admin. Bearer for HS256 and OIDC. Bearer-in-SPA = XSS tradeoff; BFF/cookie preferred for IdP-in-browser if avoiding tokens in JS.                                                       |
| **M2M**                        | `…@clients` → `client:<id>`; not in `users`. **`PUT`/`DELETE` `/api/v1/users/:id` require `user:`** subject (403 for `client:*`). Scoped service access for TaskStack jobs = future.                                                |
| **Redis**                      | Optional; fail **open** on errors for limits/lockout. Fail closed only for controls that need strong consistency.                                                                                                                   |
| **Gateway vs app limits**      | Edge: coarse IP/TLS/WAF. App: email lockout, route semantics. Avoid duplicate same numeric cap unless intentional.                                                                                                                  |
| **Clock / gameplay**           | JWT strictness affects **API** calls, not game loop. NTP on API hosts; optional leeway = polish.                                                                                                                                    |
| **Cross-platform play**        | Login UX can vary; matchmaking/connectivity = game + netcode + join tokens, orthogonal to OIDC skew.                                                                                                                                |
| **IdP portability**            | OIDC primitives in config/code; avoid vendor SDKs scattered through handlers.                                                                                                                                                       |
| **Env / config catalog**       | **Authoritative variable list:** `[.env.example](../.env.example)`. New or changed settings belong there and are cross-referenced from topic docs (`auth-session`, `oidc-auth0`, etc.).                                             |


Topic detail: [README.md](README.md).

---

## 7. Shipped vs still open


| Area                                                                             | Status                                                                                                                                                                                                                                   |
| -------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Browser register/login/logout/refresh, session cookie, `/users` Bearer or cookie | **Shipped**                                                                                                                                                                                                                              |
| `000002_*` users auth cols, sessions, refresh shell, audit                       | **Shipped**                                                                                                                                                                                                                              |
| Argon2id                                                                         | **Shipped**                                                                                                                                                                                                                              |
| Idle + absolute session, logout                                                  | **Shipped**                                                                                                                                                                                                                              |
| Bootstrap bridge + sunset docs                                                   | **Shipped** ([bootstrap-sunset.md](bootstrap-sunset.md))                                                                                                                                                                                 |
| JWT rotation                                                                     | **Incremental** — `JWT_SECRET_PREVIOUS` ([jwt-rotation.md](jwt-rotation.md))                                                                                                                                                             |
| Change-password + revoke all sessions                                            | **Shipped**                                                                                                                                                                                                                              |
| Desktop exchange + join-token API                                                | **Shipped** — `POST /auth/desktop/start` + `/auth/desktop/exchange` (PKCE, `000004_*`) + `POST /auth/join-token` ([desktop-auth-bridge.md](desktop-auth-bridge.md), [openapi.yaml](openapi.yaml)); game validates join JWT (suite-owned) |
| CSRF                                                                             | **Shipped**                                                                                                                                                                                                                              |
| Rate limits + email lockout                                                      | **Shipped** (memory default; [Redis optional](auth-session.md))                                                                                                                                                                          |
| Admin Angular UX (+ positioning vs TaskStack)                                    | **Shipped** ([platform-admin-ui.md](platform-admin-ui.md))                                                                                                                                                                               |
| OIDC + `user_identities`                                                         | **Shipped** when `OIDC_*` set ([oidc-auth0.md](oidc-auth0.md))                                                                                                                                                                           |
| OpenAPI 3 spec + CI validation                                                   | **Shipped** ([openapi.yaml](openapi.yaml), [ci.md](ci.md))                                                                                                                                                                               |
| Auth negative tests + human-only `PUT`/`DELETE` `/users`                         | **Shipped** (handler + middleware tests; [openapi.yaml](openapi.yaml) `x-requiresHumanSubject`)                                                                                                                                          |
| Migration schema golden + CI check                                               | **Shipped** ([migrations/schema_golden.sql](../migrations/schema_golden.sql), [scripts/check-schema-golden.sh](../scripts/check-schema-golden.sh), [migrations.md](migrations.md))                                                       |
| TLS reverse-proxy + cookie runbook                                               | **Shipped** ([tls-reverse-proxy.md](tls-reverse-proxy.md))                                                                                                                                                                               |
| Ops secret rotation checklist                                                    | **Shipped** ([ops-secret-rotation.md](ops-secret-rotation.md))                                                                                                                                                                           |
| Phase A control plane                                                            | **Shipped** — `000005_*`; RBAC on `/api/v1` routes in [platform-control-plane.md](platform-control-plane.md); [openapi.yaml](openapi.yaml) |


---

## 8. Roadmap phases


| Phase                                       | Summary                                                                                                                                                                                                                    |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **A** Platform control plane foundations    | **Shipped (v0):** domain boundaries + RBAC matrix + privileged ack path + read-only stubs + Angular sections; full IA/data parity → Phase B/C ([platform-control-plane.md](platform-control-plane.md)). |
| **B** Game operations surfaces              | Deliver player/character support workflows (sanctions, recovery, transfer), session/device trust operations, and economy ledger visibility for operator workflows.                                                         |
| **C** DataOps + suite integration hardening | Ship backup/restore approval flows, complete Marble/TaskStack consumer semantics, and add split-host operational hardening with contract/observability checks.                                                             |


---

## 9. Prioritized backlog

Order is **default execution priority** for platform work unless you reprioritize.

### P0 — Now / next session

1. ~~**Control plane v1 boundaries (doc):**~~ **Done** — summarized in [platform-control-plane.md](platform-control-plane.md); deep ownership remains [data-ownership.md](data-ownership.md). **Extend** concrete schemas/APIs in Phase B/C.
2. ~~**Admin IA + API alignment:**~~ **Done (v0):** Angular sections + `/api/v1` stubs documented in OpenAPI; iterate with Phase B data.
3. **Privileged action guardrails:** reason header + request correlation + `admin_audit_events` shipped for `POST /support/ack`; extend pattern to sanctions, character recovery, restore, and credential operations in Phase B/C.
4. **Suite trust follow-through:** wire Marble/TaskStack consumers to exchange → desktop Bearer → join-token path; complete game-side `token_use=join` validation + heartbeat semantics (suite-owned integration side).

### P1 — Soon after

1. **RBAC extensions:** add roles/permissions and routes as Phase B surfaces ship; keep [platform-control-plane.md](platform-control-plane.md) + [`api/platformrbac/permissions.go`](../api/platformrbac/permissions.go) in sync. Baseline matrix + human-only enforcement on Phase A routes are **shipped**.
2. **Character lifecycle workflows:** formalize restore/rename/transfer behavior, soft-delete window, and audit/event contract.
3. **Backup/restore flows:** define policy/run/request/approval model (including high-risk two-person approval path).
4. **Economy observability baseline:** expose immutable ledger query surfaces with support tooling hooks for dispute/anomaly review.

### P2 — Platform hardening (cross-cutting)

1. **OIDC hardening follow-ups:** decide JWT leeway policy; evaluate `OIDC_JWKS_URL` override for restricted networks.
2. **M2M least privilege:** define additional routes rejecting `client:*` and/or scoped M2M patterns for TaskStack jobs.
3. **Redis rollout stance:** keep memory-default local path; define criteria to require Redis in multi-replica deployments (and where fail-open vs fail-closed applies).
4. **Audit taxonomy:** standardize `event_type` vocabulary and actor/resource fields across auth + admin operations.
5. **Contract hygiene:** optional OpenAPI↔route drift check; TaskStack client generation or contract tests when that repo consumes the API.

### Explicit non-goals (unless you change §6)

- Rebuild shipped cookie admin without explicit ask.
- SPA Auth0 login UI before explicit ask (cookie path remains default for admin).

---

## 10. Open questions

- **Bootstrap disable milestone:** *TBD* — choose **release tag** or **calendar date** per [bootstrap-sunset.md](bootstrap-sunset.md); record the choice **here** when set (and in release notes / tags as appropriate).
- **Control plane v1 first slices:** **resolved** — Phase A shipped security-first (RBAC, audit, stubs, Angular IA). **Phase B** owns richer Players/Characters/DataOps data and workflows ([platform-control-plane.md](platform-control-plane.md) §Phase B/C).
- **Desktop user auth shape:** **decided** — exchange-code bridge (`/auth/desktop/start` + `/auth/desktop/exchange`) for user desktop login; avoid token-in-body on default browser login path.
- JWT clock **leeway**: implement vs strict + NTP only?
- Which **additional** routes **reject `client:*`** (beyond `PUT`/`DELETE` `/api/v1/users/:id`)? Scoped M2M for TaskStack server-side user sync?
- **`OIDC_JWKS_URL`** override for locked-down networks?
- **Redis policy:** at what deployment threshold do we require Redis instead of memory defaults for lockout/rate limit consistency?
- **Restore governance:** do we enforce two-person approval for all restores, or only production/user-impacting scopes?
- **Argon2 `m,t,p` changes:** if parameters change, require a **re-hash-on-login** (or equivalent) strategy — document in [auth-session.md](auth-session.md) or a short ADR when triggered.
- PM tooling: optional until multiple assignees need dates/queues; use this file + `docs/adr/*.md`.

---

## 11. Constraints & ground rules

Migration-only schema; **`/api/v1`** additive where possible; no secrets in frontend; Compose baseline; security-first; **new deps** — one-line justification. Bump **`MIGRATION_EXPECTED_VERSION`** ([migrations.md](migrations.md)). **CI:** [.github/workflows/ci.yml](../.github/workflows/ci.yml) + [ci.md](ci.md). **Security expectations vs CI:** [security-posture.md](security-posture.md).

**Planning:** prefer **append** to §9; do not remove shipped scope from §7 without a recorded decision (edit status, do not erase rows).

---

## 12. Agent handoff & priming

**Quick checks:** `/healthz`, `/readyz`; local: `scripts/test.ps1` with Docker up. **CI:** [ci.md](ci.md).

For AI/assistant context: read this file (§7/§9), then [README.md](README.md) index and topic guides as needed — do not treat long pasted prompts in-repo as required; keep answers aligned with OpenAPI + code.

---

## 13. Maintenance notes

Update **§1 + §9** each sprint; **§7** on ship; **§6** on decisions. **ADRs:** [adr-account-linking.md](adr-account-linking.md) pattern. Secrets only in `.env`.

---

*Last consolidated: 2026-03-23.*