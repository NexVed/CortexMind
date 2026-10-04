# ============================================================
# build-linux.ps1 — produce the Linux AppImage from Windows via Docker
#
# Builds CortexMind as a Type-2 AppImage (the distro-agnostic Linux format)
# inside Ubuntu 22.04 so the result runs on most current Linux distributions.
#
# Requirements: Docker Desktop running (Linux containers).
#
# Usage:   powershell -ExecutionPolicy Bypass -File build-linux.ps1
# Output:  build/dist/CortexMind-0.1.0-x86_64.AppImage
#          build/dist/CortexMind-0.1.0-linux-x86_64.tar.gz
# ============================================================
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }

Step "Checking Docker"
docker info 2>$null | Out-Null
if ($LASTEXITCODE) {
    throw "Docker is not running. Start Docker Desktop, then re-run this script."
}

$image = 'cortexmind-linux-builder'
Step "Building the Ubuntu 22.04 toolchain image ($image)"
docker build -t $image -f (Join-Path $root 'build\linux\Dockerfile') (Join-Path $root 'build\linux')
if ($LASTEXITCODE) { throw "docker build failed" }

Step "Compiling CortexMind and packaging the AppImage"
# Overlay Linux node_modules so the Windows ui/node_modules tree is not used.
$src = ($root -replace '\\', '/')
# Docker Desktop on Windows accepts the Windows path as a named bind.
docker run --rm `
    -e APPIMAGE_EXTRACT_AND_RUN=1 `
    -e CORTEXMIND_VERSION=0.1.0 `
    -v "${root}:/src" `
    -v cortexmind-ui-node-modules:/src/ui/node_modules `
    -v cortexmind-go-mod:/go/pkg/mod `
    -v cortexmind-go-cache:/root/.cache/go-build `
    -w /src `
    $image
if ($LASTEXITCODE) { throw "Linux AppImage build failed" }

$appimage = Join-Path $root 'build\dist\CortexMind-0.1.0-x86_64.AppImage'
$tarball  = Join-Path $root 'build\dist\CortexMind-0.1.0-linux-x86_64.tar.gz'
Write-Host "`nDone." -ForegroundColor Green
if (Test-Path $appimage) { Write-Host "  AppImage: $appimage" -ForegroundColor Green }
if (Test-Path $tarball)  { Write-Host "  Tarball:  $tarball" -ForegroundColor Green }
Write-Host "Copy the AppImage to Linux, then:  chmod +x CortexMind-0.1.0-x86_64.AppImage && ./CortexMind-0.1.0-x86_64.AppImage"
