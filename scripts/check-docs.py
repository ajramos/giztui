#!/usr/bin/env python3
"""Fail-closed documentation gate for GizTUI CI.

Validates, relative to the repository root:

1. Every JSON file parses (excluding generated/dependency trees).
2. Every YAML/YML file parses with the pinned PyYAML.
3. Markdown internal links in README.md and docs/**/*.md resolve to existing
   files. External URLs and pure anchors are skipped.

Exits non-zero on the first class of failure so CI cannot report success after
a skip. Run from the repository root.
"""

from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent

# Trees that are generated, vendored, or tool-installed; never validated.
EXCLUDED_DIRS = {
    ".git",
    ".venv",
    ".venv-ci",
    "node_modules",
    "vendor",
    "dist",
    "build",
    ".next",
    "coverage",
    "__pycache__",
    "out",
    "graphify-out",
}

EXCLUDED_PATTERNS = {
    "package-lock.json",
    "go.sum",
    "coverage.out",
    ".DS_Store",
}

MARKDOWN_LINK_RE = re.compile(r"\[[^\]]*\]\(([^)]+)\)")
FENCE_RE = re.compile(r"^```.*?^```", re.MULTILINE | re.DOTALL)
INLINE_CODE_RE = re.compile(r"`[^`\n]+`")


def should_skip(rel: str) -> bool:
    parts = Path(rel).parts
    if any(p in EXCLUDED_DIRS for p in parts):
        return True
    if Path(rel).name in EXCLUDED_PATTERNS:
        return True
    return False


def walk_files(suffixes: tuple[str, ...]):
    for root, dirs, files in os.walk(ROOT):
        dirs[:] = [d for d in dirs if d not in EXCLUDED_DIRS]
        for name in files:
            if name.endswith(suffixes):
                full = Path(root) / name
                rel = full.relative_to(ROOT)
                if not should_skip(str(rel)):
                    yield full, rel


def check_json() -> int:
    failures = 0
    for full, rel in walk_files((".json",)):
        try:
            json.loads(full.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError, OSError) as exc:
            print(f"JSON error: {rel}: {exc}")
            failures += 1
    return failures


def check_yaml() -> int:
    failures = 0
    for full, rel in walk_files((".yml", ".yaml")):
        try:
            # load_all so multi-document files are fully parsed
            list(yaml.safe_load_all(full.read_text(encoding="utf-8")))
        except (yaml.YAMLError, UnicodeDecodeError, OSError) as exc:
            print(f"YAML error: {rel}: {exc}")
            failures += 1
    return failures


def markdown_links() -> list[tuple[Path, str]]:
    links: list[tuple[Path, str]] = []
    for full, _ in walk_files((".md",)):
        if "node_modules" in str(full):
            continue
        text = full.read_text(encoding="utf-8", errors="replace")
        text = FENCE_RE.sub("", text)  # drop fenced code blocks
        text = INLINE_CODE_RE.sub("", text)  # drop inline code spans
        # Drop indented (tab/4-space) code blocks; they are not links.
        text = "\n".join(
            line
            for line in text.split("\n")
            if not line.startswith("\t") and not line.startswith("    ")
        )
        for target in MARKDOWN_LINK_RE.findall(text):
            links.append((full, target.strip()))
    return links


def check_markdown_links() -> int:
    failures = 0
    for src, target in markdown_links():
        if not target:
            continue
        if target.startswith(("http://", "https://", "mailto:", "#")):
            continue
        path_part = target.split("#", 1)[0]
        if not path_part:
            continue
        candidate = (src.parent / path_part).resolve() if not path_part.startswith("/") else (ROOT / path_part[1:]).resolve()
        if not candidate.exists():
            print(f"broken markdown link: {src.relative_to(ROOT)} -> {target}")
            failures += 1
    return failures


def main() -> int:
    total = 0
    total += check_json()
    total += check_yaml()
    total += check_markdown_links()
    if total:
        print(f"docs gate: {total} failure(s)")
        return 1
    print("docs gate: JSON, YAML, and markdown links OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
