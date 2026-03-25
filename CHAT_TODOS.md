# Session notes (CHAT_TODOS)

**Canonical plan:** [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md) §7–§9. Use this file for **short-lived session notes and in-flight PR reminders** only. Trim after merge; do not duplicate §9 here.

## Next focus

**Phase 5 (platform service) shipped:** `POST /api/v1/auth/join-token`, `POST /api/v1/auth/desktop/start`, `POST /api/v1/auth/desktop/exchange` (PKCE `code_challenge` / `code_verifier`, callback host allowlist, DB `000004_*` + failure audit buckets). Contract: [docs/openapi.yaml](docs/openapi.yaml) · flow: [docs/desktop-auth-bridge.md](docs/desktop-auth-bridge.md).

**Next (suite / game–owned or cross-repo):** wire Marble/TaskStack clients to the exchange → desktop Bearer → join-token path; implement game-side validation of `token_use=join` JWTs; heartbeat / split-host playbooks. Backlog: [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md) §9. Data model: [docs/data-ownership.md](docs/data-ownership.md).

**Fresh DB:** `docker compose down -v` → up → `migrate`; set `MIGRATION_EXPECTED_VERSION=4` in `.env`. Example `DB_NAME` / `JWT_*` defaults live in [`.env.example`](.env.example) (`suite_platform`, `suite-platform`); align an existing `.env` when upgrading.

## Rules

- Prune merged items. Git history is the audit trail.
- Security-first; minimal deps; migrations-only schema; Compose baseline.

**Handoff:** [docs/README.md](docs/README.md) · [docs/ci.md](docs/ci.md) · `./scripts/ci-local.ps1` (fast checks) · [docs/platform-api-consumer-brief.md](docs/platform-api-consumer-brief.md) (external integrators) · `/healthz`, `/readyz` · `./scripts/test.ps1` · bump `MIGRATION_EXPECTED_VERSION` when schema changes.
