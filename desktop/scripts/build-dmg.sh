#!/usr/bin/env bash
# Build the GizTUI Desktop macOS app (universal) and package it into a .dmg.
#
# Runs on macOS only (needs Xcode command line tools + the Wails CLI). No extra
# tooling required: the DMG is made with hdiutil, which ships with macOS. If
# `create-dmg` is installed (brew install create-dmg) it is used instead for a
# nicer window layout.
#
# Usage:  desktop/scripts/build-dmg.sh
# Output: desktop/build/bin/GizTUI-Desktop-<version>-universal.dmg
set -euo pipefail

# --- locate the repo + read the version -------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$DESKTOP_DIR/.." && pwd)"
VERSION="$(tr -d '[:space:]' < "$REPO_ROOT/VERSION")"
APP_NAME="GizTUI Desktop"
APP="$DESKTOP_DIR/build/bin/$APP_NAME.app"
DMG="$DESKTOP_DIR/build/bin/GizTUI-Desktop-$VERSION-universal.dmg"

if [[ "$(uname)" != "Darwin" ]]; then
  echo "error: a macOS .dmg can only be built on macOS." >&2
  exit 1
fi
command -v wails >/dev/null 2>&1 || {
  echo "error: the Wails CLI is not installed. Run:" >&2
  echo "  go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2" >&2
  exit 1
}

# --- build the universal .app ----------------------------------------------
echo "==> Building $APP_NAME.app (universal) v$VERSION"
( cd "$DESKTOP_DIR" && wails build -clean -platform darwin/universal \
    -ldflags "-X 'github.com/ajramos/giztui/internal/version.Version=$VERSION'" )

[[ -d "$APP" ]] || { echo "error: build did not produce $APP" >&2; exit 1; }

# --- package the .dmg -------------------------------------------------------
rm -f "$DMG"
if command -v create-dmg >/dev/null 2>&1; then
  echo "==> Packaging with create-dmg"
  create-dmg \
    --volname "$APP_NAME" \
    --app-drop-link 480 170 \
    --icon "$APP_NAME.app" 160 170 \
    --window-size 640 360 \
    "$DMG" "$APP"
else
  echo "==> Packaging with hdiutil (install create-dmg for a nicer layout)"
  STAGING="$(mktemp -d)"
  cp -R "$APP" "$STAGING/"
  ln -s /Applications "$STAGING/Applications"
  hdiutil create -volname "$APP_NAME" -srcfolder "$STAGING" \
    -ov -format UDZO "$DMG"
  rm -rf "$STAGING"
fi

echo "==> Built $DMG"
