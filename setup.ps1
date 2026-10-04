$ErrorActionPreference = 'Stop'
$setupRoot = $PSScriptRoot
Push-Location $setupRoot
try {
    Write-Host 'Downloading Go dependencies...' -ForegroundColor Cyan
    go mod download
    if ($LASTEXITCODE -ne 0) { throw 'Go dependency download failed. Check Go and network access.' }
    Push-Location (Join-Path $setupRoot 'ui')
    try {
        Write-Host 'Installing locked UI dependencies...' -ForegroundColor Cyan
        npm ci
        if ($LASTEXITCODE -ne 0) { throw 'UI dependency installation failed. Check Node.js and network access.' }
    } finally { Pop-Location }
    Write-Host 'Setup complete. Build the desktop app with build/build-desktop.ps1.' -ForegroundColor Green
    Write-Host 'For Vite development, follow the two-terminal setup in README.md.'
} finally { Pop-Location }
