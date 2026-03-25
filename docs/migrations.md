# Database Migrations

Go-Lab uses SQL migrations in [`migrations/`](../migrations/) with `golang-migrate`.

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

## Readiness check

Set `MIGRATION_EXPECTED_VERSION` in `.env` to latest version number.

Example: `MIGRATION_EXPECTED_VERSION=2` after `000002_*` is applied.

**Note:** If you previously applied a removed `000003_*` migration locally, your `schema_migrations.version` may be 3 while this repo only ships `000001`–`000002`. Fix by aligning the DB with the current files (e.g. backup, `migrate down` to match your history, or restore from snapshot) before relying on automated `migrate up`.

`/readyz` requires:

- DB ping succeeds
- `schema_migrations` is not dirty
- `version >= MIGRATION_EXPECTED_VERSION`
