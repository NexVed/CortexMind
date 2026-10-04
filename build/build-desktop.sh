#!/usr/bin/env bash
# ============================================================
# build-desktop.sh — native CortexMind desktop app for macOS / Linux (Wails v3)
#
#   1. builds the SolidJS UI              (ui/dist)
#   2. embeds it into internal/web/dist   (the daemon serves it locally)
#   3. compiles the Wails webview shell   -> build/dist/CortexMind
#
# The desktop app boots the same embedded SQLite daemon as cortexd and opens
# a native webview window at http://127.0.0.1:<port> instead of a browser.
#
# IMPORTANT: Wails uses the OS-native webview and CGO, so this MUST be run on the
# target OS — you cannot cross-compile the macOS/Linux desktop binaries from
# Windows. (cmd/cortexd, the headless daemon, remains cross-compilable.)
#
# Requirements:
#   - Go 1.25+, Node 18+, a C compiler (cc/clang/gcc)
#   - Linux:  webkit2gtk dev headers, e.g.
#               Debian/Ubuntu: sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
#               Fedora:        sudo dnf install gtk3-devel webkit2gtk4.1-devel
#   - macOS:  Xcode command line tools (xcode-select --install)
#
# Usage:   ./build/build-desktop.sh
#
# On Linux this also packages a Type-2 AppImage (and a .tar.gz) into build/dist/.
# From Windows, use build-linux.ps1 (Docker) to produce the same artifacts.
# ============================================================
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

step() { printf '\n==> %s\n' "$1"; }

step "Building the UI (vite)"
( cd ui && { [ -d node_modules/vite ] || npm install; } && npm run build )

step "Embedding UI into internal/web/dist"
dist_dst="internal/web/dist"
mkdir -p "$dist_dst"
find "$dist_dst" -mindepth 1 ! -name '.gitkeep' -exec rm -rf {} +
cp -R ui/dist/. "$dist_dst"/

step "Compiling the Wails desktop shell -> build/dist/CortexMind"
mkdir -p build/dist
# Linux uses GTK3 + webkit2gtk 4.1 (Wails -tags gtk3) so the AppImage runs on
# Ubuntu 22.04+ and equivalent distros. Default Wails v3 is GTK4/WebKitGTK 6.
tags=""
if [ "$(uname -s)" = "Linux" ]; then tags="-tags gtk3"; fi
CGO_ENABLED=1 go build $tags -ldflags "-s -w" -o build/dist/CortexMind ./cmd/cortexmind

if [ "$(uname -s)" = "Linux" ]; then
  step "Packaging Linux AppImage + portable tarball"
  "$root/build/linux/package-appimage.sh"
fi

step "Done. -> build/dist/CortexMind"
