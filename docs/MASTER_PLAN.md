# Master plan — Product suite + go-lab (platform API)

**Single planning doc:** specs (summary), decisions, shipped matrix, phases, prioritized backlog. Edit when priorities change; deep detail stays in `docs/*.md`.

**Related:** [Documentation index](README.md) · [Repo README](../README.md)

---

## Table of contents

1. [Executive snapshot](#1-executive-snapshot)
2. [Product Suite Architecture Spec v0.1 (condensed)](#2-product-suite-architecture-spec-v01-condensed)
3. [Go-lab platform API — architecture spec v0.1](#3-go-lab-platform-api--architecture-spec-v01)
4. [Go-lab repo — implementation reality](#4-go-lab-repo--implementation-reality)
5. [Version stance (suite vs repo)](#5-version-stance-suite-vs-repo)
6. [Architecture decisions (consensus)](#6-architecture-decisions-consensus)
7. [Shipped vs still open](#7-shipped-vs-still-open)
8. [Roadmap phases (operator)](#8-roadmap-phases-operator)
9. [Prioritized backlog](#9-prioritized-backlog)
10. [Open questions](#10-open-questions)
11. [Constraints & ground rules](#11-constraints--ground-rules)
12. [Agent handoff & priming](#12-agent-handoff--priming)
13. [Maintenance notes](#13-maintenance-notes)

---

## 1. Executive snapshot

| Item | State |
|------|--------|
| **Products** | **Marble** (game sim), **TaskStack** (control-plane UX), **Go-lab** (this repo — platform API + admin SPA + migrations) |
| **Suite spec maturity** | **v0.1** (foundation; not full prod hardening of every subsystem) |
| **Auth / scale** | Cookie + CSRF SPA; HS256 `/auth/token`; optional OIDC + `user_identities`; optional **Redis** for limits/lockout |
| **Gaps vs quality gates** | **Phase 5 suite consumption** (Marble/TaskStack use exchange + join-token; game validates join JWT); optional polish: OpenAPI↔route drift, OIDC leeway, trusted proxy / real client IP behind TLS, audit `event_type` taxonomy, scoped M2M |

**Login UX goals (current simplification):**
- **Browser web:** cookie session + CSRF is the default and preferred path.
- **Desktop Marble:** browser-assisted exchange-code -> desktop Bearer -> join-token.
- **Service/automation:** `client_credentials` only (`client:*`), scoped over time.
- **Singleplayer/local:** no required platform login; optional account-link for cloud features later.
- **Bootstrap bridge:** deprecated; disable with `AUTH_BOOTSTRAP_ENABLED=false` once legacy callers are migrated.

---

## 2. Product Suite Architecture Spec v0.1 (condensed)

**Products:** **Marble** — simulation + netcode authority. **TaskStack** — accounts / orchestration UX (consumes platform API). **Go-lab** — this repo: platform API, admin SPA, migrations.

**Principles:** Gameplay can stay offline; platform is additive. **Multi-repo** with explicit contracts. **Self-host Docker Compose** first; split-host / cloud later without boundary rewrites.

**MySQL:** schema = **migrations only**. **API:** JSON `/api/v1`, additive changes preferred; **formal contract:** [openapi.yaml](openapi.yaml) (validated in CI). **Trust:** clients trust API; game server trusts platform tokens later; platform never trusts client gameplay state.

**Deploy:** (A) single-host Compose — primary. (B) split host. (C) managed cloud.

**Security / observability / quality:** env config, validation, CORS allowlist, envelopes, no secrets in repo; structured logs; health + smoke; CI green; release = migrations + upgrade path.

**Non-goals (v0.1):** microservices split, enterprise SSO, global matchmaking, full streaming stack.

**Suite roadmap (spec):** (1) **Stabilize core** — API conventions, auth, smoke, docs/runbook. (2) **Control plane** — TaskStack auth flows, entitlements sketch, OpenAPI, contract tests. (3) **Marble bridge** — handshake, token validation, heartbeat, split-host guidance.

**Versioning:** spec `vMAJOR.MINOR`; breaking boundaries → changelog + migration note + compatibility statement.

*Long-form suite prose may live outside the repo. **§9** is always the execution backlog.*

**Data ownership & gameplay persistence design (deep dive):** [data-ownership.md](data-ownership.md) — platform vs TaskStack vs Marble, TaskStack signup → platform user, authority/sync/anti-cheat, DB hot path vs async persistence.

---

## 3. Go-lab platform API (this repo)

Self-hostable **Go API**: users, auth/session, tokens, health/readiness. **Owns:** user data, sessions (evolving), future entitlements. **Does not own:** game sim, rendering, physics.

**Ops:** Compose; env-only config; **no runtime DDL**; `go test ./...` + smoke + CI.

---

## 4. Go-lab repo — implementation reality

| Area | Location / notes |
|------|------------------|
| **Backend** | `api/` — Gin, `middleware/`, `auth/`, `authstore/`, `myhandlers/` |
| **Frontend** | `client/` — Angular admin SPA |
| **Migrations** | `migrations/` — `000002_*` auth/session/users; `000003_*` `user_identities`; `000004_*` `auth_desktop_exchange_codes` |
| **Compose** | `docker-compose.yml` — mysql, migrate, backend, frontend; optional **`redis`** profile |
| **Docs** | Topic guides under `docs/`; index [README.md](README.md); CI summary [ci.md](ci.md) |
| **Key env** | See [`.env.example`](../.env.example): `JWT_*`, `JOIN_TOKEN_TTL_SECONDS`, `DESKTOP_EXCHANGE_*`, `SESSION_*`, `OIDC_*`, `REDIS_URL`, `MIGRATION_EXPECTED_VERSION`, CSRF, platform client creds |

---

## 5. Version stance (suite vs repo)

Suite spec **v0.1** until bumped. **Go-lab:** v0.1 + auth/OIDC + Phase 4 contracts + **Phase 5 platform auth bridge shipped** (join-token, desktop exchange + PKCE, `000004_*`). **Next focus:** suite repos consume those APIs; gameplay trust + data plane per [data-ownership.md](data-ownership.md).

---

## 6. Architecture decisions (consensus)

| Topic | Decision |
|-------|----------|
| **Identity source of truth** | **Hybrid:** first-party email/password + opaque session cookie for admin SPA; local `users` + `user_identities` for `(issuer, sub)`. Optional OIDC access JWT as Bearer when `OIDC_*` set. Platform owns authz/tenancy/audit in DB. |
| **Auth0-only vs API-only** | Prefer hybrid. Pure Auth0-only only if API is a thin BFF with almost no local user model. |
| **TaskStack / Marble** | TaskStack: **consumer** of platform APIs. Marble: gameplay authority; join/session trust **later** (Phase 5). Boundaries + performance/sync design: [data-ownership.md](data-ownership.md). |
| **Canonical `aud` (OIDC)** | One API identifier: **`OIDC_AUDIENCE`**. Do not conflate with **`JWT_AUDIENCE`** unless deliberately standardized and documented. See [oidc-auth0.md](oidc-auth0.md). |
| **Multiple `aud` / multi-API** | Prefer gateway as single `aud`, or separate Auth0 APIs/tokens, or multi-aud with explicit validation only. |
| **`sub`** | Identity = **`(issuer, sub)`** only; `user_identities` enforces. |
| **Account linking** | No naive email auto-link. Safe: verified email + explicit action, admin linking, or migration “password once to attach.” JIT new user on first OIDC `(iss, sub)` today. |
| **Refresh tokens** | Auth0 refresh = client/Auth0/BFF; do not stuff into `auth_refresh_tokens` without naming flow ownership. |
| **Cookie vs Bearer** | Shipped default: HttpOnly cookie + CSRF for admin. Bearer for HS256 and OIDC. Bearer-in-SPA = XSS tradeoff; BFF/cookie preferred for IdP-in-browser if avoiding tokens in JS. |
| **M2M** | `…@clients` → `client:<id>`; not in `users`. **`PUT`/`DELETE` `/api/v1/users/:id` require `user:`** subject (403 for `client:*`). Scoped service access for TaskStack jobs = future. |
| **Redis** | Optional; fail **open** on errors for limits/lockout. Fail closed only for controls that need strong consistency. |
| **Gateway vs app limits** | Edge: coarse IP/TLS/WAF. App: email lockout, route semantics. Avoid duplicate same numeric cap unless intentional. |
| **Clock / gameplay** | JWT strictness affects **API** calls, not game loop. NTP on API hosts; optional leeway = polish. |
| **Cross-platform play** | Login UX can vary; matchmaking/connectivity = game + netcode + join tokens, orthogonal to OIDC skew. |
| **IdP portability** | OIDC primitives in config/code; avoid vendor SDKs scattered through handlers. |
| **Env / config catalog** | **Authoritative variable list:** [`.env.example`](../.env.example). New or changed settings belong there and are cross-referenced from topic docs (`auth-session`, `oidc-auth0`, etc.). |

Topic detail: [README.md](README.md).

---

## 7. Shipped vs still open

| Area | Status |
|------|--------|
| Browser register/login/logout/refresh, session cookie, `/users` Bearer or cookie | **Shipped** |
| `000002_*` users auth cols, sessions, refresh shell, audit | **Shipped** |
| Argon2id | **Shipped** |
| Idle + absolute session, logout | **Shipped** |
| Bootstrap bridge + sunset docs | **Shipped** ([bootstrap-sunset.md](bootstrap-sunset.md)) |
| JWT rotation | **Incremental** — `JWT_SECRET_PREVIOUS` ([jwt-rotation.md](jwt-rotation.md)) |
| Change-password + revoke all sessions | **Shipped** |
| Desktop exchange + join-token API | **Shipped** — `POST /auth/desktop/start` + `/auth/desktop/exchange` (PKCE, `000004_*`) + `POST /auth/join-token` ([desktop-auth-bridge.md](desktop-auth-bridge.md), [openapi.yaml](openapi.yaml)); game validates join JWT (suite-owned) |
| CSRF | **Shipped** |
| Rate limits + email lockout | **Shipped** (memory default; [Redis optional](auth-session.md)) |
| Admin Angular UX | **Shipped** ([platform-admin-ui.md](platform-admin-ui.md)) |
| OIDC + `user_identities` | **Shipped** when `OIDC_*` set ([oidc-auth0.md](oidc-auth0.md)) |
| OpenAPI 3 spec + CI validation | **Shipped** ([openapi.yaml](openapi.yaml), [ci.md](ci.md)) |
| Auth negative tests + human-only `PUT`/`DELETE` `/users` | **Shipped** (handler + middleware tests; [openapi.yaml](openapi.yaml) `x-requiresHumanSubject`) |
| Migration schema golden + CI check | **Shipped** ([migrations/schema_golden.sql](../migrations/schema_golden.sql), [scripts/check-schema-golden.sh](../scripts/check-schema-golden.sh), [migrations.md](migrations.md)) |
| TLS reverse-proxy + cookie runbook | **Shipped** ([tls-reverse-proxy.md](tls-reverse-proxy.md)) |
| Ops secret rotation checklist | **Shipped** ([ops-secret-rotation.md](ops-secret-rotation.md)) |
| Admin vs TaskStack positioning | **Shipped** ([platform-admin-ui.md](platform-admin-ui.md)) |

---

## 8. Roadmap phases (go-lab–centric)

| Phase | State | Summary |
|-------|--------|---------|
| **1** Core auth | **Shipped** | Cookie session, CSRF, `000002_*`, HS256 + `JWT_SECRET_PREVIOUS`, bootstrap sunset docs — detail §7 |
| **2** OIDC | **Shipped** (env-gated) + follow-ups | `OIDC_*`, `000003_*`, M2M → `client:<id>`; follow-ups: `OIDC_JWKS_URL`, leeway, **additional** human-only or scoped-M2M routes |
| **3** Scale-out | **Mostly shipped** | Optional Redis; Compose profile |
| **4** Contracts | **Shipped** | OpenAPI, negative tests, schema golden CI, TLS + ops runbooks, admin vs TaskStack doc — detail §7 |
| **5** Integration | **Platform shipped; suite next** | go-lab: join-token + desktop exchange (PKCE) + `000004_*`; Marble/TaskStack wiring + join JWT validation + heartbeat **suite/game** |

---

## 9. Prioritized backlog

Order is **default execution priority** for platform work unless you reprioritize.

### P0 — Now / next session

1. **Phase 5 — trust follow-through:** harden desktop exchange bridge and game trust integration (join-token + desktop exchange implementation shipped; game handshake semantics remain suite-owned).
2. **Phase 2 follow-ups** (pick as needed): OIDC leeway; **additional** routes rejecting `client:*` or **scoped** M2M for TaskStack; account linking endpoint spec + implementation ([adr-account-linking.md](adr-account-linking.md)).

### P1 — Soon after

3. **Auth audit event taxonomy** — expand beyond ad hoc `event_type` strings; cross-ref [ops-secret-rotation.md](ops-secret-rotation.md).
4. **Marble / TaskStack data plane:** implement persistence and authority patterns per [data-ownership.md](data-ownership.md) (sim hot path vs DB batching, async pipelines, `platform_user_id` linkage).
5. **Contract hygiene:** optional OpenAPI↔route drift check; TaskStack client generation or contract tests when that repo consumes the API.

### P2 — Optional

6. If the suite “source of truth” doc is **outside** git, add a **link in §2** after the summary block.

### P3 — Phase 5 (suite integration)

**Go-lab–owned (shipped for v0.1 handoff):** join-token + desktop exchange **APIs**, **OpenAPI**, migrations `000004_*`, docs [desktop-auth-bridge.md](desktop-auth-bridge.md). Further platform surfaces → extend OpenAPI as needed.

**Suite / game–owned:** Marble join-token **consumer** semantics (verify HS256 JWT, bind to session), in-game heartbeat, split-host playbooks beyond API/env documentation.

### Explicit non-goals (unless you change §6)

- Rebuild shipped cookie admin without explicit ask.
- SPA Auth0 login UI before explicit ask (cookie path remains default for admin).

---

## 10. Open questions

- **Bootstrap disable milestone:** *TBD* — choose **release tag** or **calendar date** per [bootstrap-sunset.md](bootstrap-sunset.md); record the choice **here** when set (and in release notes / tags as appropriate).
- **Desktop user auth shape:** **decided** — exchange-code bridge (`/auth/desktop/start` + `/auth/desktop/exchange`) for user desktop login; avoid token-in-body on default browser login path.
- JWT clock **leeway**: implement vs strict + NTP only?
- Which **additional** routes **reject `client:*`** (beyond `PUT`/`DELETE` `/api/v1/users/:id`)? Scoped M2M for TaskStack server-side user sync?
- **`OIDC_JWKS_URL`** override for locked-down networks?
- **Argon2 `m,t,p` changes:** if parameters change, require a **re-hash-on-login** (or equivalent) strategy — document in [auth-session.md](auth-session.md) or a short ADR when triggered.
- PM tooling: optional until multiple assignees need dates/queues; use this file + `docs/adr/*.md`.

---

## 11. Constraints & ground rules

Migration-only schema; **`/api/v1`** additive where possible; no secrets in frontend; Compose baseline; security-first; **new deps** — one-line justification. Bump **`MIGRATION_EXPECTED_VERSION`** ([migrations.md](migrations.md)). **CI:** [.github/workflows/ci.yml](../.github/workflows/ci.yml) + [ci.md](ci.md). **Security expectations vs CI:** [security-posture.md](security-posture.md).

**Planning:** prefer **append** to §9; do not remove shipped scope from §7 without a recorded decision (edit status, do not erase rows).

---

## 12. Agent handoff & priming

**Smoke / health:** `/healthz`, `/readyz`; `scripts/test.ps1` when Docker is up. **CI:** [ci.md](ci.md).

**Architecture-only agent — paste as system or first message:**

```text
You are the architecture steward for the go-lab monorepo (platform API) within Marble + TaskStack (Product Suite Architecture Spec v0.1).

Sources of truth (read in order):
1) docs/MASTER_PLAN.md (this repo)
2) docs/README.md — full doc index; then topic files as needed (data-ownership, openapi, oidc-auth0, auth-session, desktop-auth-bridge, bootstrap-sunset, jwt-rotation, migrations, adr-account-linking, platform-admin-ui, tls-reverse-proxy, ops-secret-rotation, ci)
3) Any attached suite spec deltas

Your job: shipped vs planned; flag contradictions; propose next 3 milestones with risks; do not expand scope silently. Do not implement code unless asked.
```

---

## 13. Maintenance notes

Update **§1 + §9** each sprint; **§7** on ship; **§6** on decisions. **ADRs:** [adr-account-linking.md](adr-account-linking.md) pattern. Secrets only in `.env`.

---

*Last consolidated: 2026-03-22 — Phase 5 platform auth bridge (join-token + desktop exchange) shipped; backlog aimed at suite consumption.*
