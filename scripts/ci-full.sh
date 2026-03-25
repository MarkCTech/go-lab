#!/usr/bin/env bash
# Full local parity with GitHub Actions (both jobs). Ends with `docker compose down -v` (wipes volumes). Destructive to local compose data.
# Requires: Docker, Go, bash (smoke is Go — no PowerShell).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

cleanup() {
  echo "== Teardown (docker compose down -v) =="
  docker compose down -v
}
trap cleanup EXIT

echo "== Job 1: Go unit tests + govulncheck + OpenAPI =="
go test -C api ./...
( cd "$ROOT/api" && go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./... )
go run -C api ./cmd/openapivalidate -spec "$ROOT/docs/openapi.yaml"

echo "== Job 2: Compose build, migrate, schema golden, smoke =="
docker compose build --parallel
docker compose up -d --no-build
docker compose run --rm migrate

for i in $(seq 1 30); do
  if curl -fsS "http://localhost:5000/readyz" > /dev/null; then
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "Backend readiness check timed out"
    exit 1
  fi
  sleep 2
done

go run -C api ./cmd/schemagolden
go run -C api ./cmd/smoketest -- -base http://localhost:5000

echo "== Full CI parity passed. =="
