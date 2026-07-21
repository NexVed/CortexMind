# ============================================================
# build-desktop.ps1 — native CortexMind desktop app + installer for Windows (Wails v3)
#
#   1. builds the SolidJS UI              (ui/dist)
#   2. embeds it into internal/web/dist   (the daemon serves it locally)
#   3. embeds the app icon + version info (goversioninfo)
#   4. compiles the Wails webview shell   -> build/dist/CortexMind.exe
#   5. builds a per-user installer         -> build/dist/CortexMind-Setup-<ver>.exe
#
# The desktop app boots the same embedded SQLite daemon as cortexd and opens
# a native WebView2 window at http://127.0.0.1:<port> instead of a browser.
#
# Requirements: Go 1.25+, Node 18+, WebView2 runtime (preinstalled on Win 11).
# Optional for the installer: NSIS (winget install NSIS.NSIS). If NSIS is not
# found the script still produces the standalone .exe and skips the installer.
#
# Usage:   powershell -ExecutionPolicy Bypass -File build/build-desktop.ps1
# ============================================================
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
$version = '0.1.0'

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }

Step "Building the UI (vite)"
Push-Location (Join-Path $root 'ui')
try {
    if (-not (Test-Path 'node_modules')) { npm install; if ($LASTEXITCODE) { throw "npm install failed" } }
    npm run build
    if ($LASTEXITCODE) { throw "npm run build failed" }
} finally { Pop-Location }

Step "Embedding UI into internal/web/dist"
$distSrc = Join-Path $root 'ui\dist'
$distDst = Join-Path $root 'internal\web\dist'
New-Item -ItemType Directory -Force -Path $distDst | Out-Null
Get-ChildItem $distDst -Force -Exclude '.gitkeep' | Remove-Item -Recurse -Force
Copy-Item (Join-Path $distSrc '*') $distDst -Recurse -Force

Step "Embedding app icon + version info (goversioninfo)"
# Suffixed name so this Windows resource is only linked for windows/amd64 and
# never breaks the daemon's cross-platform builds.
$syso = Join-Path $root 'cmd\cortexmind\resource_windows_amd64.syso'
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest `
    -64 `
    -icon (Join-Path $root 'build\windows\icon.ico') `
    -o $syso `
    (Join-Path $root 'build\windows\versioninfo.json')
if ($LASTEXITCODE) { throw "goversioninfo failed" }

Step "Compiling the Wails desktop shell -> build/dist/CortexMind.exe"
$out = Join-Path $root 'build\dist\CortexMind.exe'
New-Item -ItemType Directory -Force -Path (Split-Path $out) | Out-Null
$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
# -H windowsgui hides the console for a true GUI app.
go build -ldflags "-s -w -H windowsgui" -o $out ./cmd/cortexmind
if ($LASTEXITCODE) { throw "go build failed" }
Write-Host "  -> $out" -ForegroundColor Green

Step "Building the installer (NSIS)"
$makensis = $null
$cmd = Get-Command makensis -ErrorAction SilentlyContinue
if ($cmd) { $makensis = $cmd.Source }
if (-not $makensis) {
    $candidates = @(
        (Join-Path $env:ProgramFiles 'NSIS\makensis.exe'),
        (Join-Path ([Environment]::GetFolderPath('ProgramFilesX86')) 'NSIS\makensis.exe')
    )
    foreach ($p in $candidates) { if (Test-Path $p) { $makensis = $p; break } }
}
if ($makensis) {
    & $makensis "/DVERSION=$version" (Join-Path $root 'build\windows\installer.nsi')
    if ($LASTEXITCODE) { throw "makensis failed" }
    Write-Host ("  -> " + (Join-Path $root "build\dist\CortexMind-Setup-$version.exe")) -ForegroundColor Green
} else {
    Write-Host "  NSIS not found - skipping installer. Install with: winget install NSIS.NSIS" -ForegroundColor Yellow
}

Write-Host "`nDone." -ForegroundColor Green
