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
# ============================================================
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

step() { printf '\n==> %s\n' "$1"; }

step "Building the UI (vite)"
( cd ui && { [ -d node_modules ] || npm install; } && npm run build )

step "Embedding UI into internal/web/dist"
dist_dst="internal/web/dist"
mkdir -p "$dist_dst"
find "$dist_dst" -mindepth 1 ! -name '.gitkeep' -exec rm -rf {} +
cp -R ui/dist/. "$dist_dst"/

step "Compiling the Wails desktop shell -> build/dist/CortexMind"
mkdir -p build/dist
# Linux uses webkit2gtk 4.1 headers; drop this tag if your distro ships 4.0.
tags=""
if [ "$(uname -s)" = "Linux" ]; then tags="-tags webkit2_41"; fi
CGO_ENABLED=1 go build $tags -ldflags "-s -w" -o build/dist/CortexMind ./cmd/cortexmind

step "Done. -> build/dist/CortexMind"
