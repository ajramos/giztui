#!/bin/bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASELINE="$ROOT/architecture-baseline.csv"
violations=0

if [ ! -f "$BASELINE" ]; then
    echo "ERROR: missing architecture baseline: $BASELINE"
    exit 1
fi

echo "Checking architecture ratchets..."

# The TUI still has legacy direct Gmail client access. Freeze the count per file
# so service-layer migration can happen incrementally without allowing new debt.
for file in "$ROOT"/internal/tui/*.go; do
    case "$file" in
        *_test.go) continue ;;
    esac

    rel="${file#"$ROOT"/}"
    count="$(grep -Eo '\b(a|app)\.Client\.' "$file" 2>/dev/null | wc -l | tr -d ' ')" || count=0
    allowed="$(awk -F, -v name="$rel" '$1 == "direct_client" && $2 == name { print $3 }' "$BASELINE")"
    allowed="${allowed:-0}"

    if [ "$count" -gt "$allowed" ]; then
        echo "ERROR: $rel has $count direct client accesses; baseline allows $allowed"
        violations=$((violations + 1))
    fi
done

if grep -r -E '(fmt\.Printf|fmt\.Print|log\.Printf)' "$ROOT/internal/tui" \
    --include='*.go' --exclude='*_test.go' | grep -v '^[[:space:]]*//' >/dev/null 2>&1; then
    echo "ERROR: direct user-facing output found in internal/tui; use ErrorHandler"
    violations=$((violations + 1))
fi

# Derive the expected interface from each concrete FooServiceImpl. Existing
# exceptions are explicit in the baseline; any new exception fails the gate.
while IFS= read -r service; do
    [ -n "$service" ] || continue
    if ! grep -q "^type $service interface" "$ROOT/internal/services/interfaces.go"; then
        allowed="$(awk -F, -v name="$service" '$1 == "missing_interface" && $2 == name { print $3 }' "$BASELINE")"
        if [ "${allowed:-0}" -lt 1 ]; then
            echo "ERROR: $service implementation has no canonical interface"
            violations=$((violations + 1))
        fi
    fi
done < <(
    grep -hE '^type [A-Za-z0-9]+ServiceImpl struct' "$ROOT"/internal/services/*_service.go \
        | sed -E 's/^type ([A-Za-z0-9]+Service)Impl struct.*$/\1/' \
        | sort -u
)

echo "INFO: direct App field access is not yet machine-enforced; no false pass is claimed."

if [ "$violations" -ne 0 ]; then
    echo "Architecture gate failed with $violations violation(s)."
    exit 1
fi

echo "Architecture ratchets passed."
