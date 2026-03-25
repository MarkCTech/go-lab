# Continuous integration

**Workflow definition:** [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).

## Cross-platform tooling

| Concern | Approach |
|---------|----------|
| **Unit tests, OpenAPI, schema golden, HTTP smoke** | **Go** only (`go test`, `api/cmd/openapivalidate`, `api/cmd/schemagolden`, `api/cmd/smoketest`). Works the same on Windows, macOS, and Linux with **Go** installed. |
| **Bash scripts** (`ci-local.sh`, `ci-full.sh`, `check-schema-golden.sh`, `migrate.sh`) | Optional convenience on Unix; **not required** on Windows — use the `.ps1` equivalents or run the `go run -C api ./cmd/...` commands directly. `check-schema-golden.sh` is only a one-line wrapper around Go. |
| **PowerShell** | Optional: `scripts/test.ps1` and `migrate.ps1` are thin wrappers for Windows habits; smoke is implemented in Go. |
| **Docker Compose** | Required for `compose-smoke` / `ci-full`. Uses the same `docker compose` CLI everywhere. |
| **Angular / Node** | Only for building the `client/` image locally or in CI — not part of `ci-local`. |
| **Python** | Not used in CI. |

**Other pitfalls:** Line endings in `schema_golden.sql` are normalized in `schemagolden`. Paths with spaces in repo root are untested.

## OpenAPI (`docs/openapi.yaml`)

[OpenAPI](https://www.openapis.org/) is an **industry standard**: you keep `openapi.yaml` and run tools that validate or generate from it.

**Hand-maintaining the YAML is normal** when the contract is the product. This repo authors it next to the Go code. CI validates structure with [`api/cmd/openapivalidate`](../api/cmd/openapivalidate/main.go) ([kin-openapi](https://github.com/getkin/kin-openapi)).

```bash
go run -C api ./cmd/openapivalidate -spec docs/openapi.yaml   # from repository root
```

## Triggers

- **Push** to any branch
- **Pull requests** (any base)

## Jobs

### `backend-tests`

- **Runner:** `ubuntu-latest`
- **Steps:** Checkout → setup Go (`api/go.mod`) → `go test ./...` → **govulncheck** → `go run ./cmd/openapivalidate -spec ../docs/openapi.yaml`

### `compose-smoke`

- **Runner:** `ubuntu-latest`
- **Timeout:** 20 minutes
- **Compose files:** `docker-compose.yml` plus `docker-compose.ci.yml` when the workflow can use the actions cache; otherwise `docker-compose.yml` only
- **Flow:**
  1. Checkout, setup Go (`api/go.mod`), Docker Buildx
  2. `cp .env.example .env`
  3. `docker compose build --parallel`
  4. `docker compose up -d --no-build`
  5. `docker compose run --rm migrate`
  6. Wait until `http://localhost:5000/readyz` succeeds (up to ~60s)
  7. `go run -C api ./cmd/schemagolden`
  8. `go run -C api ./cmd/smoketest -- -base http://localhost:5000`
  9. On failure: dump logs
  10. **Always:** `docker compose down -v`

## Local checks

| What | Command |
|------|---------|
| **Fast** — `go test` + OpenAPI (what most people run before commit) | `./scripts/ci-local.ps1` or `bash scripts/ci-local.sh` |
| **Full CI** — `govulncheck` + OpenAPI + compose + golden + smoke (matches both CI jobs); ends with `down -v` | `./scripts/ci-full.ps1` or `bash scripts/ci-full.sh` |
| Smoke only (API already running) | `go run -C api ./cmd/smoketest` or `./scripts/test.ps1` |
| Golden only (Compose mysql up) | `go run -C api ./cmd/schemagolden` |

`ci-local` intentionally **does not** run `govulncheck` (extra network/time). **`ci-full`** and **CI** run it.

## Govulncheck

[`govulncheck`](https://go.dev/blog/govulncheck) reports known vulnerabilities in module dependencies. It runs in the **`backend-tests` CI job** and in **`scripts/ci-full`**. To run ad hoc:

```bash
cd api && go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

## What this testing does and does not do

**In scope:** Unit/handler tests, OpenAPI structure, govulncheck (known vulns in deps), Compose smoke (Go), schema golden (Go).

**Out of scope:** Penetration testing, fuzzing, load tests — see [security-posture.md](security-posture.md).

## Related

[security-posture.md](security-posture.md) · [MASTER_PLAN.md](MASTER_PLAN.md) · [migrations.md](migrations.md) · [README.md](../README.md) · [scripts/README.md](../scripts/README.md)
