#!/usr/bin/env bash
# Prove the SDLC baseline output is deterministic: run it twice over the same
# ref and require byte-identical JSON.
#
# Usage: verify-baseline-determinism.sh REF
#
# Uses --no-weekly to bound runtime in CI; the weekly lizard snapshots are
# deterministic too (sorted traversal, fixed cohort) but are exercised in the
# full regeneration instead of here.
set -euo pipefail

ref=${1:?usage: verify-baseline-determinism.sh REF}

# Prefer the pinned CI venv; fall back to any python3 with lizard available.
if [[ -x ".venv-ci/bin/python" ]]; then
  PY="${PY:-.venv-ci/bin/python}"
else
  PY="${PY:-python3}"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "baseline determinism check: ref=${ref} python=${PY}"
"$PY" baseline.py --ref "$ref" --no-weekly --out "$tmp/run1.json" --cache "$tmp/cache.json" >/dev/null
"$PY" baseline.py --ref "$ref" --no-weekly --out "$tmp/run2.json" --cache "$tmp/cache2.json" >/dev/null

if diff -q "$tmp/run1.json" "$tmp/run2.json" >/dev/null; then
  echo "baseline determinism OK (run1 == run2)"
else
  echo "baseline determinism FAILED: repeated runs over ${ref} differ" >&2
  exit 1
fi
