# ============================================================
# build-windows.ps1 — produce a self-contained CortexMind.exe
#
#   1. builds the SolidJS UI            (ui/dist)
#   2. copies it into the embed package (internal/web/dist)
#   3. generates the white-background app icon + version resource
#   4. compiles a single static Windows binary  -> CortexMind.exe
#
# Usage:   powershell -ExecutionPolicy Bypass -File build-windows.ps1
# ============================================================
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }

# ── 1. Build the UI ─────────────────────────────────────
Step "Building the UI (vite)"
Push-Location (Join-Path $root 'ui')
try {
    if (-not (Test-Path 'node_modules')) { npm install; if ($LASTEXITCODE) { throw "npm install failed" } }
    npm run build
    if ($LASTEXITCODE) { throw "npm run build failed" }
} finally { Pop-Location }

# ── 2. Copy dist into the Go embed package ──────────────
Step "Embedding UI into internal/web/dist"
$distSrc = Join-Path $root 'ui\dist'
$distDst = Join-Path $root 'internal\web\dist'
New-Item -ItemType Directory -Force -Path $distDst | Out-Null
Get-ChildItem $distDst -Force -Exclude '.gitkeep' | Remove-Item -Recurse -Force
Copy-Item (Join-Path $distSrc '*') $distDst -Recurse -Force

# ── 3. Icon + version resource ──────────────────────────
Step "Generating white-background app icon"
& (Join-Path $root 'build\windows\gen-icon.ps1')

Step "Embedding icon + version info (goversioninfo)"
$syso = Join-Path $root 'cmd\cortexd\resource.syso'
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest `
    -64 `
    -icon (Join-Path $root 'build\windows\icon.ico') `
    -o $syso `
    (Join-Path $root 'build\windows\versioninfo.json')
if ($LASTEXITCODE) { throw "goversioninfo failed" }

# ── 4. Compile the static Windows binary ────────────────
Step "Compiling CortexMind.exe"
$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go build -ldflags "-s -w" -o (Join-Path $root 'CortexMind.exe') ./cmd/cortexd
if ($LASTEXITCODE) { throw "go build failed" }

Write-Host "`nDone. -> $(Join-Path $root 'CortexMind.exe')" -ForegroundColor Green
Write-Host "Run it (double-click or from a terminal); it serves the app and opens your browser at http://127.0.0.1:8090" -ForegroundColor Green
