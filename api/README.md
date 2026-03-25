# api (platform service)

Go HTTP API: Gin, MySQL, JSON envelopes, JWT and HttpOnly cookie sessions.

```bash
go test -C api ./...   # from repository root
# From this directory:
go test ./...
```

Docker builds from the repository root — see [../README.md](../README.md).

**Also in this module:**
- `cmd/openapivalidate` — validates [docs/openapi.yaml](../docs/openapi.yaml) (OpenAPI 3 structural check).
- `cmd/schemagolden` — compares migrated MySQL schema to [migrations/schema_golden.sql](../migrations/schema_golden.sql) (requires Compose mysql up).
- `cmd/smoketest` — HTTP smoke tests against a running API (cookie session, CSRF, M2M, bootstrap metadata). Mutates DB; `-base` host must match **`SMOKETEST_ALLOW_HOSTS`** (default: loopback only) or `*` to disable — see `.env.example`.

From repository root:  
`go run -C api ./cmd/openapivalidate -spec docs/openapi.yaml` · `go run -C api ./cmd/schemagolden` · `go run -C api ./cmd/smoketest`

**Security-related behavior:** Argon2id password hashing (`auth/password.go`); session storage in `authstore` (migration `000002_*`); CSRF protection on cookie-authenticated mutations; per-IP and per-email rate limits; optional **`REDIS_URL`** for shared limits. Configuration: [`.env.example`](../.env.example). Documentation: [docs/README.md](../docs/README.md).
