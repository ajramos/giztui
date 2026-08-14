#!/usr/bin/env python3
"""Fail when source-file complexity exceeds the checked-in per-file baseline."""

import argparse
import csv
import sys
from pathlib import Path

try:
    import lizard
except ImportError:
    print("ERROR: lizard is required; install requirements-ci.txt", file=sys.stderr)
    raise SystemExit(2)


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_BASELINE = ROOT / "quality-baseline-per-file.csv"
SOURCE_SUFFIXES = {".go", ".js", ".jsx", ".ts", ".tsx"}
EXCLUDED_PARTS = {
    ".venv-ci",
    "build",
    "dist",
    "e2e",
    "graphify-out",
    "mocks",
    "node_modules",
    "test",
    "test-results",
    "wailsjs",
}
EXCLUDED_SUFFIXES = (
    "_test.go",
    ".d.ts",
    ".spec.ts",
    ".spec.tsx",
    ".test.ts",
    ".test.tsx",
)
FIELDS = ("max_nloc", "max_ccn", "max_func_len", "funcs_over_ccn10")


def is_source(path: Path) -> bool:
    rel = path.relative_to(ROOT)
    return (
        path.suffix in SOURCE_SUFFIXES
        and not any(part in EXCLUDED_PARTS for part in rel.parts)
        and not str(rel).endswith(EXCLUDED_SUFFIXES)
    )


def source_files() -> list[Path]:
    return sorted(path for path in ROOT.rglob("*") if path.is_file() and is_source(path))


def metrics(path: Path) -> dict[str, int]:
    try:
        result = lizard.analyze_file(str(path))
    except Exception as exc:
        raise RuntimeError(f"failed to analyze {path.relative_to(ROOT)}: {exc}") from exc

    functions = result.function_list
    return {
        "max_nloc": result.nloc,
        "max_ccn": max((fn.cyclomatic_complexity for fn in functions), default=0),
        "max_func_len": max((fn.end_line - fn.start_line + 1 for fn in functions), default=0),
        "funcs_over_ccn10": sum(fn.cyclomatic_complexity > 10 for fn in functions),
    }


def load_baseline(path: Path) -> dict[str, dict[str, int]]:
    with path.open(newline="", encoding="utf-8") as handle:
        rows = csv.DictReader(handle)
        return {
            row["file"]: {field: int(row[field]) for field in FIELDS}
            for row in rows
        }


def write_baseline(path: Path, current: dict[str, dict[str, int]]) -> None:
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(
            handle,
            fieldnames=("file", *FIELDS),
            lineterminator="\n",
        )
        writer.writeheader()
        for name, values in sorted(current.items()):
            writer.writerow({"file": name, **values})


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--baseline", type=Path, default=DEFAULT_BASELINE)
    parser.add_argument("--write", action="store_true")
    args = parser.parse_args()

    try:
        current = {
            str(path.relative_to(ROOT)): metrics(path)
            for path in source_files()
        }
    except RuntimeError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2

    if args.write:
        write_baseline(args.baseline, current)
        print(f"Wrote {len(current)} source rows to {args.baseline}")
        return 0

    if not args.baseline.exists():
        print(f"ERROR: missing quality baseline: {args.baseline}", file=sys.stderr)
        return 2

    baseline = load_baseline(args.baseline)
    violations: list[str] = []
    for name, values in current.items():
        allowed = baseline.get(name)
        if allowed is None:
            violations.append(f"{name}: new source file is not reviewed in the baseline")
            continue
        for field in FIELDS:
            if values[field] > allowed[field]:
                violations.append(
                    f"{name}: {field} increased {allowed[field]} -> {values[field]}"
                )

    if violations:
        print("Quality ratchet failed:", file=sys.stderr)
        for violation in violations:
            print(f"  - {violation}", file=sys.stderr)
        return 1

    print(f"Quality ratchet passed for {len(current)} source files.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
