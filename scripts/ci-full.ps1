# Full local parity with GitHub Actions (both jobs): Go tests + OpenAPI, then Docker build/up/migrate + schema golden + HTTP smoke.
# Ends with `docker compose down -v` (wipes Compose named volumes — same as CI). Destructive to local compose data; do not use on a stack you need to keep.
# Requires: Docker, Go.
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Wait-Ready {
    for ($i = 0; $i -lt 30; $i++) {
        try {
            $r = Invoke-WebRequest -Uri "http://localhost:5000/readyz" -UseBasicParsing -TimeoutSec 2
            if ($r.StatusCode -eq 200) { return }
        } catch { }
        Start-Sleep -Seconds 2
    }
    throw "Backend readiness check timed out (http://localhost:5000/readyz)"
}

try {
    Write-Host "== Job 1: Go unit tests + govulncheck + OpenAPI =="
    go test -C api ./...
    Push-Location (Join-Path $Root "api")
    try {
        go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
    $spec = Join-Path $Root "docs/openapi.yaml"
    go run -C api ./cmd/openapivalidate -spec $spec

    Write-Host "== Job 2: Compose build, migrate, schema golden, smoke =="
    docker compose build --parallel
    docker compose up -d --no-build
    docker compose run --rm migrate
    Wait-Ready
    go run -C api ./cmd/schemagolden
    go run -C api ./cmd/smoketest -- -base http://localhost:5000
    Write-Host "== Full CI parity passed. =="
}
finally {
    Write-Host "== Teardown (docker compose down -v) =="
    docker compose down -v
}
