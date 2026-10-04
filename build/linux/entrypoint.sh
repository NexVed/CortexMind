#!/usr/bin/env bash
# Runs inside the Ubuntu 22.04 builder image (see Dockerfile).
# Strips CR from scripts so a Windows checkout still runs under bash.
set -euo pipefail
cd /src

for f in \
  build/build-desktop.sh \
  build/linux/package-appimage.sh \
  build/linux/AppRun \
  build/linux/entrypoint.sh
do
  if [ -f "$f" ]; then
    sed -i 's/\r$//' "$f"
    chmod +x "$f"
  fi
done

exec ./build/build-desktop.sh
