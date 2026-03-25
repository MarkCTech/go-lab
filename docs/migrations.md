# Database Migrations

Go-Lab uses SQL migrations in [`migrations/`](../migrations/) with `golang-migrate`. Other guides: [README.md](README.md).

## What to run

From repo root:

```powershell
docker compose run --rm migrate
```

Or:

```powershell
./scripts/migrate.ps1
```

## Rules

- API never creates/alters schema at runtime.
- Only migration files change schema.
- Run migrations before smoke tests or app rollout.

## Current migration set

- `000001_*`: base `users` table (`id`, `name`, `pennies`).
- `000002_*`: auth/session schema — `users` email/password/timestamps, `auth_sessions`, `auth_refresh_tokens`, `auth_audit_events`.
- `000003_*`: `user_identities` — maps `(issuer, subject)` → local `users.id` for OIDC / external IdPs.
- `000004_*`: `auth_desktop_exchange_codes` — one-time desktop exchange bridge (`desktop/start` -> `desktop/exchange`).
- `000005_*`: `platform_roles`, `user_platform_roles`, `admin_audit_events` — Phase A operator RBAC + immutable control-plane audit. **Grant access:** [platform-operator-roles.md](platform-operator-roles.md). **Boundaries + matrix:** [platform-control-plane.md](platform-control-plane.md).

## Readiness check (`/readyz`)

`/readyz` **always** checks that the database responds (ping).

**Migration version / dirty flag:** checked **only when** `MIGRATION_EXPECTED_VERSION` is set to a **positive** integer in `.env` (see [`api/config/config.go`](../api/config/config.go)). Then `/readyz` requires `schema_migrations` **not dirty** and `version >= MIGRATION_EXPECTED_VERSION`. If unset or zero, `/readyz` does **not** read `schema_migrations`.

Set `MIGRATION_EXPECTED_VERSION` to the integer prefix of the newest applied migration (e.g. `5` after `000005_*`).

If your DB version drifts from this branch’s migration chain, align it (backup, `migrate` up/down, or restore) before rollout.

## Schema golden (CI drift check)

After `migrate up`, the **application** schema (tables only — not `schema_migrations`) is compared to a normalized mysqldump checked in as [`schema_golden.sql`](../migrations/schema_golden.sql). The check is implemented in Go: [`api/cmd/schemagolden`](../api/cmd/schemagolden/main.go). CI runs `go run -C api ./cmd/schemagolden`; [`scripts/check-schema-golden.sh`](../scripts/check-schema-golden.sh) is a thin wrapper for convenience.

When you add or change migrations intentionally, refresh the golden file from a clean migrated DB (Linux/macOS/Git Bash):

```bash
docker compose exec -T mysql mysqldump -uroot -p"$DB_PASS" --no-data --skip-comments --single-transaction \
  --ignore-table="${DB_NAME}.schema_migrations" "${DB_NAME}" 2>/dev/null \
  | sed -E 's/ AUTO_INCREMENT=[0-9]+//g' > migrations/schema_golden.sql
```

Pin the MySQL image tag in Compose when upgrading majors (dump layout can change).
