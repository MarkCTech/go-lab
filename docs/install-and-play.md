# Install and play (quickstart)

This guide is for first-time local setup. Goal: run the stack, verify it works, and try the UI flow.

## 1) Prerequisites

- Docker Desktop (or Docker Engine + Compose)
- Git
- Optional: Go 1.25.8+ (only needed to run smoke checks outside containers)

## 2) Start the stack

From repo root:

```powershell
Copy-Item .env.example .env
docker compose up -d --build
docker compose run --rm migrate
```

What you should have:

- UI: `http://localhost:4200`
- API: `http://localhost:5000`
- Health: `http://localhost:5000/healthz`
- Ready: `http://localhost:5000/readyz`

## 3) Verify it actually works

Run smoke checks:

```powershell
go run -C api ./cmd/smoketest
```

If smoke passes, your install is good.

## 4) Try the UI quickly (dev-only shortcut)

The default auth path is invite-based and can require operator setup. For a local first-run UI preview, use the bootstrap bridge:

1. In `.env`, set:
   - `AUTH_BOOTSTRAP_ENABLED=true`
2. In `client/src/environments/environment.ts`, set:
   - `useBootstrapAuth: true`
3. Rebuild/restart:

```powershell
docker compose up -d --build
```

This is for local exploration only. Do not use bootstrap auth for production-style deployments.

## 5) Common fixes

- `401` in UI after login attempts:
  - Confirm cookies are allowed and `SESSION_COOKIE_SECURE=false` for plain local HTTP.
- Ready check fails:
  - Re-run `docker compose run --rm migrate`
  - Confirm `MIGRATION_EXPECTED_VERSION` matches current migration chain.
- API seems up but behavior is odd:
  - Run smoke again to pinpoint setup drift.

## Next docs

- Main setup/config: [../README.md](../README.md)
- API contract: [openapi.yaml](openapi.yaml)
- Auth/session behavior: [auth-session.md](auth-session.md)
- Migration/readiness details: [migrations.md](migrations.md)
