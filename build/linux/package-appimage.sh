#!/usr/bin/env bash
# ============================================================
# package-appimage.sh — wrap build/dist/CortexMind as a Type-2 AppImage
#
# AppImage is the distro-agnostic Linux format (chmod +x and run). The Wails
# shell still needs WebKitGTK 4.1 on the host — same as other GTK webview apps.
#
# Also writes a portable .tar.gz next to the AppImage.
#
# Usage (after ./build/build-desktop.sh, or standalone if the binary exists):
#   ./build/linux/package-appimage.sh
# ============================================================
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

version="${CORTEXMIND_VERSION:-0.1.0}"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="x86_64" ;;
  aarch64|arm64) arch="aarch64" ;;
esac

bin="$root/build/dist/CortexMind"
out_dir="$root/build/dist"
appdir="$out_dir/CortexMind.AppDir"
cache="$root/build/linux/.cache"
icon_src="$root/ui/public/logo.png"
desktop_src="$root/build/linux/CortexMind.desktop"
apprun_src="$root/build/linux/AppRun"
tool="$cache/appimagetool-${arch}.AppImage"
appimage_name="CortexMind-${version}-${arch}.AppImage"
tarball_name="CortexMind-${version}-linux-${arch}.tar.gz"

step() { printf '\n==> %s\n' "$1"; }

if [ ! -x "$bin" ] && [ ! -f "$bin" ]; then
  echo "error: $bin not found. Build the Linux desktop binary first:" >&2
  echo "  ./build/build-desktop.sh" >&2
  exit 1
fi
chmod +x "$bin"

if [ ! -f "$icon_src" ]; then
  echo "error: app icon missing at $icon_src" >&2
  exit 1
fi

step "Assembling AppDir"
rm -rf "$appdir"
mkdir -p \
  "$appdir/usr/bin" \
  "$appdir/usr/share/applications" \
  "$appdir/usr/share/icons/hicolor/256x256/apps" \
  "$appdir/usr/share/metainfo"

cp "$bin" "$appdir/usr/bin/CortexMind"
chmod +x "$appdir/usr/bin/CortexMind"
cp "$desktop_src" "$appdir/CortexMind.desktop"
cp "$desktop_src" "$appdir/usr/share/applications/CortexMind.desktop"
cp "$icon_src" "$appdir/cortexmind.png"
cp "$icon_src" "$appdir/usr/share/icons/hicolor/256x256/apps/cortexmind.png"
cp "$apprun_src" "$appdir/AppRun"
chmod +x "$appdir/AppRun"

step "Writing portable tarball -> $out_dir/$tarball_name"
stage="$out_dir/CortexMind-linux-${arch}"
rm -rf "$stage"
mkdir -p "$stage"
cp "$bin" "$stage/CortexMind"
cp "$desktop_src" "$stage/CortexMind.desktop"
cp "$icon_src" "$stage/cortexmind.png"
cat > "$stage/README.txt" <<EOF
CortexMind ${version} for Linux (${arch})

Run:
  chmod +x CortexMind
  ./CortexMind

This binary uses the system WebKitGTK 4.1 webview (same as other Wails/GTK apps).
If the window does not open, install WebKitGTK 4.1:

  Debian / Ubuntu / Mint:  sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0
  Fedora:                  sudo dnf install webkit2gtk4.1 gtk3
  Arch:                    sudo pacman -S webkit2gtk-4.1 gtk3

Data lives in ~/.cortex/
EOF
tar -C "$out_dir" -czf "$out_dir/$tarball_name" "CortexMind-linux-${arch}"
rm -rf "$stage"

step "Fetching appimagetool (if needed)"
mkdir -p "$cache"
if [ ! -x "$tool" ]; then
  url="https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-${arch}.AppImage"
  echo "  downloading $url"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$tool" "$url"
  else
    wget -q -O "$tool" "$url"
  fi
  chmod +x "$tool"
fi

step "Building AppImage -> $out_dir/$appimage_name"
export ARCH="$arch"
export APPIMAGE_EXTRACT_AND_RUN=1
# appimagetool needs FUSE; in Docker / WSL without FUSE, extract-and-run is used.
if ! "$tool" "$appdir" "$out_dir/$appimage_name"; then
  echo "appimagetool failed; keeping AppDir and tarball." >&2
  echo "  AppDir:  $appdir" >&2
  echo "  tarball: $out_dir/$tarball_name" >&2
  exit 1
fi

rm -rf "$appdir"
chmod +x "$out_dir/$appimage_name"
ln -sf "$appimage_name" "$out_dir/CortexMind-${arch}.AppImage"

step "Done"
echo "  AppImage: $out_dir/$appimage_name"
echo "  Tarball:  $out_dir/$tarball_name"
echo "  Run:      chmod +x $out_dir/$appimage_name && $out_dir/$appimage_name"
