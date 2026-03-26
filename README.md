# go-lab

**Marble / TaskStack suite — platform service:** HTTP JSON API, Angular admin SPA, MySQL schema (migrations only), Docker Compose. Application code: [`api/`](api/) (Go module `github.com/codemarked/go-lab/api`), [`client/`](client/), [`migrations/`](migrations/). Planning and integrator docs: [`docs/README.md`](docs/README.md).

## Start here (install and play)

For a first local run, follow [`docs/install-and-play.md`](docs/install-and-play.md).

Fast path:

1. `Copy-Item .env.example .env`
2. `docker compose up -d --build`
3. `docker compose run --rm migrate`
4. `go run -C api ./cmd/smoketest`

**Defaults:** Local and example configuration use the logical MySQL database name `suite_platform` and HS256 JWT issuer/audience `suite-platform` / `suite-platform-api` (see [`.env.example`](.env.example)). Override in production as needed. If you have an existing database named `todosdb` or old JWT claims, set `DB_NAME` / `JWT_*` explicitly when migrating.

**Need:** Docker + Compose + Git. **Go 1.25.8+** (or auto toolchain from `api/go.mod`) for local `go test` / `go run` (CI and the backend image match `api/go.mod`).

## First-time setup

1. `cp .env.example .env` (PowerShell: `Copy-Item .env.example .env`) — set long `JWT_SECRET`, `PLATFORM_CLIENT_*`.
2. `docker compose up -d --build`
3. `docker compose run --rm migrate` (any fresh DB or new migration)
4. **UI** [localhost:4200](http://localhost:4200) · **API** [localhost:5000](http://localhost:5000) · **health** `/healthz` · **ready** `/readyz` · MySQL `localhost:3306`

Schema comes **only** from [`migrations/`](migrations/), not app startup.

### Admin SPA (Angular)

- **Cookie auth (default):** invite-based flow (`/auth/invite/accept`) then sign in. Mutations need **CSRF** header (`X-CSRF-Token` = `gl_csrf` cookie) and **`withCredentials`**. Periodic **`POST /api/v1/auth/refresh`**; **401** -> sign-in again.
- **Bootstrap JWT (dev-only bridge):** `useBootstrapAuth: true` calls `POST /api/v1/auth/bootstrap`. Endpoint is disabled by default and source IP is restricted by `AUTH_BOOTSTRAP_ALLOWED_CIDRS` (localhost by default) when enabled. Keep off for prod-style deploys — [docs/bootstrap-sunset.md](docs/bootstrap-sunset.md).
- **Same origin for cookies:** different ports = different sites, so use **`apiBaseUrl: ''`** and proxy `/api` on the SPA host ([`docker/frontend.nginx.conf`](docker/frontend.nginx.conf), [`client/proxy.conf.json`](client/proxy.conf.json)). Full URL to `:5000` only if you mean to use **Bearer**, not cookie session.

## Daily use

- Up: `docker compose up -d` (add `--build` when images change)
- Migrate: `docker compose run --rm migrate` (backend waits for migrate on full stack)
- **Auth onboarding failures:** verify latest migrations ran (current chain ends at `000010_*`) and `MIGRATION_EXPECTED_VERSION` matches ([`.env.example`](.env.example))
- **Schema out of sync** (`no change` but missing columns): compare `users` columns vs `schema_migrations`; repair with care — [docs/migrations.md](docs/migrations.md)
- Down: `docker compose down` · wipe DB: `docker compose down -v`
- Logs: `docker compose logs backend --tail 100`

## Configuration

Full list: [`.env.example`](.env.example). Common:

| Var | Note |
|-----|------|
| `DB_*` | DSN uses `parseTime=true` — required for sessions |
| `JWT_SECRET` | ≥32 chars prod; optional `JWT_SECRET_PREVIOUS` — [jwt-rotation.md](docs/jwt-rotation.md) |
| `JOIN_TOKEN_TTL_SECONDS` | TTL for end-user Marble join handoff tokens (`/api/v1/auth/join-token`) |
| `OPERATOR_INVITE_TTL_SECONDS` | TTL for operator invite tokens (`/api/v1/auth/invite/accept`) |
| `DESKTOP_EXCHANGE_CODE_TTL_SECONDS` | TTL for one-time desktop exchange codes (`/api/v1/auth/desktop/start`) |
| `DESKTOP_EXCHANGE_CALLBACK_HOSTS` | Comma-separated callback host allowlist for desktop exchange start validation |
| `SESSION_*` | HttpOnly session cookie; **`SESSION_COOKIE_SECURE=false`** on plain HTTP (see `.env.example` and Compose) or browsers drop cookies → 401 |
| `CSRF_*` | Double-submit for cookie mutations — [auth-session.md](docs/auth-session.md) |
| `PLATFORM_CLIENT_*` | `/auth/token` + bootstrap |
| `MIGRATION_EXPECTED_VERSION` | Optional; `/readyz` checks version |
| `OIDC_ISSUER_URL` + `OIDC_AUDIENCE` | Both or neither; RS256 Bearer — [oidc-auth0.md](docs/oidc-auth0.md) |
| `REDIS_URL` | Optional shared rate limits / lockout; `docker compose --profile redis` |
| `CORS_ALLOWED_ORIGINS` | No `*` |
| `TRUSTED_PROXIES` | Optional CIDRs/IPs trusted for `ClientIP`; empty = trust none |
| `AUTH_BOOTSTRAP_ALLOWED_CIDRS` | CIDRs allowed to call `/auth/bootstrap` when enabled (default localhost only) |

## API (`/api/v1`)

| Area | Endpoints |
|------|-----------|
| Health | `GET /healthz`, `GET /readyz` |
| Auth | `POST .../auth/register` (disabled; invite-only), `auth/invite/accept`, `login`, `logout`, `refresh` (cookie session), `change-password`, `join-token` (human user only), desktop bridge: `desktop/start` + `desktop/exchange` (PKCE challenge/verifier) |
| Service | `POST .../auth/token`, `.../auth/bootstrap` (temporary bridge) |
| Users | List/create/search + get by id — **session cookie or Bearer** (`client_credentials` OK). **`PUT`/`DELETE` by id** require **human** auth (`user:` subject); M2M Bearer → **403** — [openapi.yaml](docs/openapi.yaml) |

Responses: `{ data, meta }` or `{ error, meta }`. Old `/api/...` outside v1 → **410**. SPA config: [`client/src/environments/environment.ts`](client/src/environments/environment.ts).

**Docs:** [docs/README.md](docs/README.md) · **OpenAPI:** [docs/openapi.yaml](docs/openapi.yaml) · **Roadmap / suite:** [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md)

## Tests and CI

- Fast local checks (same as CI **job 1** — tests + OpenAPI): `./scripts/ci-local.ps1` or `bash scripts/ci-local.sh`
- Full local CI (**both** jobs, ends with `docker compose down -v`): `./scripts/ci-full.ps1` or `bash scripts/ci-full.sh` (`pwsh` required for smoke on Unix)
- Details: [docs/ci.md](docs/ci.md) · [scripts/README.md](scripts/README.md)
- Unit tests only (from repo root): `go test -C api ./...`
- Smoke: `go run -C api ./cmd/smoketest` or `./scripts/test.ps1` (Compose stack running) · `./scripts/migrate.ps1` or `bash scripts/migrate.sh`
- [`.github/workflows/ci.yml`](.github/workflows/ci.yml) — tests + OpenAPI + Compose smoke; cache via [`docker-compose.ci.yml`](docker-compose.ci.yml)

## Production

Strong secrets; TLS in front; **`SESSION_COOKIE_SECURE=true`**; private MySQL; migrate before new replicas; tight CORS; keep **`AUTH_BOOTSTRAP_ENABLED=false`** unless you are doing local migration work. Set **`TRUSTED_PROXIES`** explicitly when running behind reverse proxies. **Multi-replica:** set **`REDIS_URL`** (or edge limits) so rate limits / lockout aren’t per-process only — [auth-session.md](docs/auth-session.md).

## More

- Migrations: [docs/migrations.md](docs/migrations.md)
- Docker Desktop must be running on Windows
