# One-shot bootstrap + run for Windows PowerShell.
#
#   .\launch.ps1            # check prerequisites, install, start db + service + console
#   .\launch.ps1 -SetupOnly # check + install, don't start
#
# Everything real happens in scripts/setup.mjs (Node). This wrapper only exists so a fresh
# clone works without pnpm on PATH yet: corepack ships with Node and activates the pinned
# pnpm version for us.
param([switch]$SetupOnly)

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host 'Node.js >= 24 is required and was not found. Install it from https://nodejs.org (or `winget install OpenJS.NodeJS.LTS`), then rerun.' -ForegroundColor Red
    exit 1
}

corepack enable | Out-Null

if ($SetupOnly) { node scripts/setup.mjs } else { node scripts/setup.mjs --start }
exit $LASTEXITCODE
