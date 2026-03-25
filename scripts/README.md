# Scripts

All **CI-equivalent checks** that are not `go test` live under **`api/cmd/`** (OpenAPI validate, schema golden, HTTP smoke) so they run on **Windows, macOS, and Linux** with only **Go** (+ Docker where noted).

| Script | Purpose |
|--------|---------|
| [`ci-local.ps1`](ci-local.ps1) / [`ci-local.sh`](ci-local.sh) | **Fast:** `go test` + OpenAPI only (no `govulncheck`). |
| [`ci-full.ps1`](ci-full.ps1) / [`ci-full.sh`](ci-full.sh) | **Full local CI:** `govulncheck` + OpenAPI + Docker compose + `schemagolden` + `smoketest`, then `down -v`. |
| [`test.ps1`](test.ps1) | Windows wrapper: `go run -C api ./cmd/smoketest` |
| [`check-schema-golden.sh`](check-schema-golden.sh) | `go run -C api ./cmd/schemagolden` |
| [`migrate.ps1`](migrate.ps1) / [`migrate.sh`](migrate.sh) | `docker compose run --rm migrate` |

Go commands from repo root:

```text
go run -C api ./cmd/smoketest [-- -base URL -client-id ID -client-secret SECRET]
go run -C api ./cmd/schemagolden
go run -C api ./cmd/openapivalidate -spec docs/openapi.yaml
```

Docs: [docs/ci.md](../docs/ci.md)
