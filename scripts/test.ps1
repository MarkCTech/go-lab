# Delegates to Go smoketest (cross-platform; same as CI). Requires Go on PATH.
param(
    [string]$ApiBaseUrl = "http://localhost:5000",
    [string]$ClientId = "dev-platform",
    [string]$ClientSecret = "change-me-dev-secret-min-length-16"
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
go run -C api ./cmd/smoketest -- -base $ApiBaseUrl -client-id $ClientId -client-secret $ClientSecret
exit $LASTEXITCODE
