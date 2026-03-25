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

## Readiness check

Set `MIGRATION_EXPECTED_VERSION` in `.env` to the **latest applied** migration version number (integer prefix of the newest `NNNNNN_*.sql` file).

Example: `MIGRATION_EXPECTED_VERSION=4` after `000004_*` is applied.

`/readyz` requires:

- DB ping succeeds
- `schema_migrations` is not dirty
- `version >= MIGRATION_EXPECTED_VERSION`

If your local DB version is ahead or behind the migration files in this branch (e.g. after a branch switch or a removed migration in history), align the database with the repo’s migration chain (backup, `migrate` up/down as appropriate, or restore) before relying on automated `migrate up`.

## Schema golden (CI drift check)

After `migrate up`, the **application** schema (tables only — not `schema_migrations`) is compared to a normalized mysqldump checked in as [`schema_golden.sql`](../migrations/schema_golden.sql). The check is implemented in Go: [`api/cmd/schemagolden`](../api/cmd/schemagolden/main.go). CI runs `go run -C api ./cmd/schemagolden`; [`scripts/check-schema-golden.sh`](../scripts/check-schema-golden.sh) is a thin wrapper for convenience.

When you add or change migrations intentionally, refresh the golden file from a clean migrated DB (Linux/macOS/Git Bash):

```bash
docker compose exec -T mysql mysqldump -uroot -p"$DB_PASS" --no-data --skip-comments --single-transaction \
  --ignore-table="${DB_NAME}.schema_migrations" "${DB_NAME}" 2>/dev/null \
  | sed -E 's/ AUTO_INCREMENT=[0-9]+//g' > migrations/schema_golden.sql
```

Pin the MySQL image tag in Compose when upgrading majors (dump layout can change).
