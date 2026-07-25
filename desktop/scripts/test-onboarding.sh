#!/usr/bin/env bash
# Test the desktop first-run onboarding (missing credentials.json) end-to-end.
#
# It temporarily moves your credentials.json (and, with --with-token, your
# token.json) out of the way, launches `wails dev`, and restores everything on
# exit — even on Ctrl-C or error. The onboarding "Welcome" screen should appear;
# click "Choose credentials.json…" and pick the .bak file this script prints to
# verify the import + sign-in flow.
#
# Usage:
#   desktop/scripts/test-onboarding.sh              # hide credentials.json only
#   desktop/scripts/test-onboarding.sh --with-token # also hide token.json (forces full OAuth)
#
# Honors GMAIL_TUI_CREDENTIALS / GMAIL_TUI_TOKEN if you point the app elsewhere.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

CRED="${GMAIL_TUI_CREDENTIALS:-$HOME/.config/giztui/credentials.json}"
TOKEN="${GMAIL_TUI_TOKEN:-$HOME/.config/giztui/token.json}"

WITH_TOKEN=0
[[ "${1:-}" == "--with-token" ]] && WITH_TOKEN=1

STAMP="$(date +%s)"
CRED_BAK="$CRED.onboardtest.$STAMP.bak"
TOKEN_BAK="$TOKEN.onboardtest.$STAMP.bak"
moved_cred=0
moved_token=0

# restore is idempotent: safe to run from both the INT and EXIT traps.
restore() {
  echo
  echo "==> Restoring your credentials…"
  if [[ "$moved_cred" == 1 && -f "$CRED_BAK" ]]; then
    mv -f "$CRED_BAK" "$CRED"
    echo "    restored $CRED"
  fi
  if [[ "$moved_token" == 1 && -f "$TOKEN_BAK" ]]; then
    mv -f "$TOKEN_BAK" "$TOKEN"
    echo "    restored $TOKEN"
  fi
}
trap restore EXIT INT TERM

command -v wails >/dev/null 2>&1 || {
  echo "error: the Wails CLI is not installed. Run:" >&2
  echo "  (cd '$DESKTOP_DIR' && make deps)" >&2
  echo "  # or: go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2" >&2
  exit 1
}

if [[ -f "$CRED" ]]; then
  mv "$CRED" "$CRED_BAK"
  moved_cred=1
  echo "==> Hid credentials.json — in the app's import dialog, pick:"
  echo "    $CRED_BAK"
else
  echo "note: $CRED does not exist; you're already in the no-credentials state."
  echo "      Import any OAuth credentials.json from the app to test."
fi

if [[ "$WITH_TOKEN" == 1 && -f "$TOKEN" ]]; then
  mv "$TOKEN" "$TOKEN_BAK"
  moved_token=1
  echo "==> Hid token.json (full OAuth consent will run after import)"
fi

echo "==> Launching 'wails dev' (Ctrl-C to stop — your files restore automatically)"
cd "$DESKTOP_DIR"
wails dev
