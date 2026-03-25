#!/usr/bin/env bash
# Fast local checks: Go unit tests + OpenAPI (matches CI backend-tests minus govulncheck).
# Govulncheck runs only in CI (or manually — docs/ci.md). Full parity: ci-full.sh — see docs/ci.md
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== Go unit tests (api) =="
go test -C api ./...

echo "== OpenAPI 3 validation (Go) =="
go run -C api ./cmd/openapivalidate -spec "$ROOT/docs/openapi.yaml"

echo "== Local fast checks done. =="
