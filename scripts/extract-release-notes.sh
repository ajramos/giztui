#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 VERSION OUTPUT" >&2
  exit 2
fi

version=$1
output=$2
temp_file=$(mktemp)
trap 'rm -f "$temp_file"' EXIT

awk -v version="$version" '
  /^##[[:space:]]+\[/ {
    line = $0
    sub(/^##[[:space:]]+\[/, "", line)
    closing = index(line, "]")
    if (found) exit
    if (closing > 1 && substr(line, 1, closing - 1) == version && substr(line, closing + 1, 1) ~ /[[:space:]]|^$/) { found = 1; next }
  }
  found { print }
  END { if (!found) exit 1 }
' CHANGELOG.md > "$temp_file"

if ! grep -q '[^[:space:]]' "$temp_file"; then
  echo "CHANGELOG.md entry for $version is empty" >&2
  exit 1
fi

{
  printf '## Release notes\n'
  cat "$temp_file"
  printf '\n## Supply-chain verification\n\n'
  printf -- '- Verify downloads with `SHA256SUMS`.\n'
  printf -- '- CycloneDX SBOMs cover the CLI module, desktop Go module, and frontend dependency tree.\n'
  printf -- '- GitHub artifact attestations bind every downloadable file to this workflow and source commit.\n'
} > "$output"
