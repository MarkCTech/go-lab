#!/usr/bin/env bash
# Apply SQL migrations via the migrate service (same as migrate.ps1). Requires Docker Compose.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
exec docker compose run --rm migrate
