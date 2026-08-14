#!/bin/bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROFILE="${1:-$ROOT/coverage.out}"
BASELINE_FILE="$ROOT/coverage-baseline.txt"

if [ ! -f "$PROFILE" ]; then
    echo "ERROR: coverage profile not found: $PROFILE"
    exit 1
fi
if [ ! -f "$BASELINE_FILE" ]; then
    echo "ERROR: coverage baseline not found: $BASELINE_FILE"
    exit 1
fi

actual="$(go tool cover -func="$PROFILE" | awk '/^total:/ { gsub("%", "", $3); print $3 }')"
baseline="$(tr -d '[:space:]' < "$BASELINE_FILE")"

if [ -z "$actual" ] || [ -z "$baseline" ]; then
    echo "ERROR: could not parse coverage or baseline"
    exit 1
fi

if ! awk -v actual="$actual" -v baseline="$baseline" 'BEGIN { exit !(actual + 0 >= baseline + 0) }'; then
    echo "ERROR: statement coverage fell from ${baseline}% to ${actual}%"
    exit 1
fi

echo "Coverage ratchet passed: ${actual}% >= ${baseline}%."
