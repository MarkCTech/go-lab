#!/usr/bin/env bash
# Thin wrapper: schema comparison is implemented in Go (api/cmd/schemagolden) for CI and Windows parity.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
exec go run -C api ./cmd/schemagolden
