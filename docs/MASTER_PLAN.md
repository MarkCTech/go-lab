# Master plan - suite + go-lab

Single planning document for platform scope in this repo.

## 1) Snapshot

- Products: Marble (game authority), TaskStack (control-plane UX), go-lab (platform API + admin SPA + migrations).
- API contract: `[openapi.yaml](openapi.yaml)`.
- Current auth posture: cookie + CSRF for browser admin; HS256 `/auth/token`; optional OIDC bearer.
- Current platform focus: stable control-plane surfaces and suite consumption of desktop exchange + join-token.

## 2) Current decisions

- Identity subject is `(issuer, sub)` for OIDC mapping; no naive email auto-link.
- Cookie session is the default browser path; bearer remains for desktop/service use.
- Human-only enforcement remains on privileged user mutation routes (`PUT`/`DELETE /api/v1/users/:id`).
- Schema changes are migration-only; no runtime DDL.
- Redis remains optional for shared lockout/rate-limit state (memory default stays valid for local/single-replica).

## 3) Shipped status


| Area                                                                     | Status                             |
| ------------------------------------------------------------------------ | ---------------------------------- |
| Register/login/logout/refresh + cookie sessions + CSRF                   | Shipped                            |
| Change-password + session revocation                                     | Shipped                            |
| OIDC validation + `user_identities`                                      | Shipped (when `OIDC_*` configured) |
| Desktop exchange (`/auth/desktop/start` + `/auth/desktop/exchange`)      | Shipped                            |
| Join-token endpoint (`/auth/join-token`)                                 | Shipped                            |
| Platform control-plane RBAC on `/api/v1` routes                          | Shipped                            |
| Economy read model + route                                               | Shipped                            |
| Backup/restore governance workflow                                       | Shipped                            |
| Operator case workflows + routes                                         | Shipped                            |
| Operator identity model (`operator_accounts`, invites, role assignments) | Shipped                            |
| CI checks (tests, OpenAPI validate, schema golden, compose smoke)        | Shipped                            |


## 4) Active backlog

### P0 (next)

1. Suite follow-through: Marble/TaskStack fully consume exchange -> desktop bearer -> join-token flow.
2. Game-side join token validation semantics are finalized and documented in Marble runbooks.
3. Extend privileged action-reason + audit taxonomy consistently to new high-risk mutations.

### P1 (soon)

1. RBAC extension as new routes land; keep docs and `api/platformrbac/permissions.go` aligned.
2. Character lifecycle and recovery workflow definition (policy + contract).
3. Backup/restore policy automation beyond current governance workflow.

### P2 (hardening)

1. OIDC policy hardening (leeway decision, optional JWKS override path).
2. M2M least-privilege expansion (`client:*` guardrails on additional routes where needed).
3. Optional OpenAPI-to-route drift guard in CI.
4. Redis requirement threshold guidance for multi-replica production.

## 5) Open questions

- Bootstrap disable milestone: release-tag based or date-based cutoff.
- Which additional routes should reject `client:*` by default.
- OIDC leeway policy: strict-only vs bounded skew tolerance.

## 6) Maintenance

- Update this file on every meaningful backlog or decision shift.
- Keep deep implementation detail in domain docs under `[docs/README.md](README.md)`.

