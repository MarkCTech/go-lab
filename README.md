# go-lab

Go API, Angular SPA, MySQL 8.4, and golang-migrate, orchestrated with Docker Compose. Sources: [`go_CRUD_api/`](go_CRUD_api/), [`client/`](client/), [`migrations/`](migrations/).

**Prerequisites:** Docker with Compose and Git.

**Go on your machine is optional.** Image builds only run `go build` (not tests). CI runs `go test ./...` on GitHub. Install Go 1.23+ if you want local tests or `go run` without Docker.

## First-time setup

1. Copy [`.env.example`](.env.example) to `.env` and set secrets (e.g. long `JWT_SECRET`; `PLATFORM_CLIENT_ID` / `PLATFORM_CLIENT_SECRET` for auth).

   ```bash
   cp .env.example .env
   ```

   PowerShell: `Copy-Item .env.example .env`

2. Start the stack:

   ```bash
   docker compose up -d --build
   ```

3. Apply schema (every fresh DB, and after new migration files):

   ```bash
   docker compose run --rm migrate
   ```

4. **App** [http://localhost:4200](http://localhost:4200) · **API** [http://localhost:5000](http://localhost:5000) · **health** [http://localhost:5000/healthz](http://localhost:5000/healthz) · **readiness** [http://localhost:5000/readyz](http://localhost:5000/readyz) · MySQL `localhost:3306`

Tables are not created by the app at startup—only the `migrate` job applies [`migrations/`](migrations/).

## Daily use

- **Start (rebuild images):** `docker compose up -d --build`
- **Start (reuse images):** `docker compose up -d`
- **Migrations:** `docker compose run --rm migrate`
- **Stop:** `docker compose down` · **wipe DB volume:** `docker compose down -v`
- **Logs:** `docker compose logs backend --tail 100`

## Configuration

Everything is env-driven; the full list is in [`.env.example`](.env.example). Commonly touched:

- `DB_*` — MySQL
- `JWT_SECRET` — signing key (use **≥32** chars in production)
- `PLATFORM_CLIENT_ID` / `PLATFORM_CLIENT_SECRET` — `POST /api/v1/auth/token`
- `MIGRATION_EXPECTED_VERSION` — if set, `/readyz` enforces migration version
- `CORS_ALLOWED_ORIGINS` — comma-separated origins (no `*`)
- `GIN_MODE` — `release` (default) or `debug`

## API (`/api/v1`)

| Area | Endpoints |
|------|-----------|
| Health | `GET /healthz`, `GET /readyz` |
| Auth | `POST /api/v1/auth/token` (client credentials, rate-limited), `POST /api/v1/auth/bootstrap` |
| Users (Bearer) | `GET`/`POST /api/v1/users`, `GET /api/v1/users/search?name=`, `GET`/`PUT`/`DELETE /api/v1/users/:id` (`DELETE` → 204) |

JSON envelope: success `{ data, meta.request_id }`; errors `{ error: { code, message, details }, meta }`. Examples: `VALIDATION_ERROR`, `NOT_FOUND`, `UNAUTHORIZED`, `INTERNAL_ERROR`, `RATE_LIMITED`. Unversioned `/api/...` (outside `/api/v1`) → **410 Gone**. SPA base URL/config: [`client/src/environments/environment.ts`](client/src/environments/environment.ts).

## Tests and CI

- **Unit (needs Go):** `cd go_CRUD_api && go test ./...`
- **Smoke (stack + migrate running):** `./scripts/test.ps1` · helper: `./scripts/migrate.ps1`

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml) — unit tests and Compose smoke in parallel; BuildKit + [GHA cache](https://docs.docker.com/build/cache/backends/gha/) via [`docker-compose.ci.yml`](docker-compose.ci.yml) on push / same-repo PRs.

## Production

Strong `JWT_SECRET`; rotate `PLATFORM_CLIENT_*` if leaked; TLS in front; keep MySQL private; bump `MIGRATION_EXPECTED_VERSION` after deploys; run migrations before new app replicas; tight `CORS_ALLOWED_ORIGINS`; aggregate logs.

## Migrations and troubleshooting

- Guide: [`docs/migrations.md`](docs/migrations.md)
- Windows: if Docker errors on the engine socket, start Docker Desktop.
- `/readyz` stuck: run migrations; match `MIGRATION_EXPECTED_VERSION` to the DB; check `schema_migrations` for `dirty = 1`.
