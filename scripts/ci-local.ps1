# Fast local checks: Go unit tests + OpenAPI validation (matches CI backend-tests minus govulncheck).
# Govulncheck runs only in CI (or run manually — docs/ci.md). Full parity: ci-full.ps1 — see docs/ci.md
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

Write-Host "== Go unit tests (api) =="
go test -C api ./...

Write-Host "== OpenAPI 3 validation (Go) =="
$OpenAPIspec = Join-Path $Root "docs/openapi.yaml"
go run -C api ./cmd/openapivalidate -spec $OpenAPIspec
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "== Local fast checks done. =="
