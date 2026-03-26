# Continuous integration

Workflow file: [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)

## What CI enforces

- Go tests: `go test ./...` (api module)
- Vulnerability scan: `govulncheck`
- OpenAPI structure validation: `api/cmd/openapivalidate`
- Compose smoke: boot stack, run migrations, wait for `/readyz`, run schema golden + smoketest

## Jobs

| Job | Purpose |
|-----|---------|
| `dependency-freshness` | Reports outdated npm/go dependencies |
| `backend-tests` | `go test`, `govulncheck`, OpenAPI validation |
| `compose-smoke` | Compose startup + migrate + ready check + schema golden + smoketest |

## Local equivalents

- Fast checks: `./scripts/ci-local.ps1` or `bash scripts/ci-local.sh`
- Full CI-equivalent: `./scripts/ci-full.ps1` or `bash scripts/ci-full.sh`
- Ad hoc:
  - `go run -C api ./cmd/openapivalidate -spec docs/openapi.yaml`
  - `go run -C api ./cmd/schemagolden`
  - `go run -C api ./cmd/smoketest`

## Boundaries

- CI catches regressions, schema/contract drift, and startup-footgun failures.
- CI does not replace threat modeling, fuzzing, pentesting, or load testing.

## Related

- [security-posture.md](security-posture.md)
- [migrations.md](migrations.md)
- [scripts/README.md](../scripts/README.md)
