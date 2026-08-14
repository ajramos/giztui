#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT

mkdir -p "$fixture/desktop/frontend" "$fixture/packaging/homebrew"
git -C "$fixture" init --initial-branch=main --quiet
git -C "$fixture" config user.name release-test
git -C "$fixture" config user.email release-test@example.com
git -C "$fixture" remote add origin .

write_release_fixture() {
  local version=$1
  local product_version=$2
  printf '%s\n' "$version" > "$fixture/VERSION"
  printf '# Changelog\n\n## [%s] - 2026-08-14\n\nSubstantive release notes.\n' "$version" > "$fixture/CHANGELOG.md"
  printf '{"info":{"productVersion":"%s"}}\n' "$product_version" > "$fixture/desktop/wails.json"
  printf '{"name":"giztui-desktop-frontend","version":"%s"}\n' "$version" > "$fixture/desktop/frontend/package.json"
  printf '{"name":"giztui-desktop-frontend","version":"%s","lockfileVersion":3,"packages":{"":{"name":"giztui-desktop-frontend","version":"%s"}}}\n' "$version" "$version" > "$fixture/desktop/frontend/package-lock.json"
  printf 'cask "giztui-desktop" do\n  version "%s"\nend\n' "$version" > "$fixture/packaging/homebrew/giztui-desktop.rb"
  git -C "$fixture" add .
  git -C "$fixture" commit --quiet -m "prepare $version"
  git -C "$fixture" tag "v$version"
  git -C "$fixture" update-ref refs/remotes/origin/main HEAD
}

expect_validation_failure() {
  if (cd "$fixture" && "$repo_root/scripts/validate-release.sh" "$1" >/dev/null 2>&1); then
    echo "expected release validation to fail for $1" >&2
    exit 1
  fi
}

write_release_fixture 1.2.3 1.2.3
(cd "$fixture" && "$repo_root/scripts/validate-release.sh" v1.2.3 >/dev/null)
(cd "$fixture" && "$repo_root/scripts/extract-release-notes.sh" 1.2.3 "$fixture/notes.md")
grep -q 'Substantive release notes.' "$fixture/notes.md"
expect_validation_failure v01.2.3
expect_validation_failure v65536.0.0
expect_validation_failure v18446744073709551616.0.0

printf '1.2.\n3\n' > "$fixture/VERSION"
expect_validation_failure v1.2.3
printf '1.2.3\n' > "$fixture/VERSION"

printf '# Changelog\n\nprefix ## [1.2.3]\n\nSubstantive release notes.\n' > "$fixture/CHANGELOG.md"
expect_validation_failure v1.2.3
git -C "$fixture" restore CHANGELOG.md

write_release_fixture 1.3.0-rc.1 1.3.0
(cd "$fixture" && "$repo_root/scripts/validate-release.sh" v1.3.0-rc.1 >/dev/null)

write_release_fixture 1.4.0+build-1 1.4.0
(cd "$fixture" && "$repo_root/scripts/validate-release.sh" v1.4.0+build-1 >/dev/null)

printf '# Changelog\n\n## [1.4.0+build-1] - 2026-08-14\n\n  \n## [1.3.0-rc.1] - 2026-08-14\n' > "$fixture/CHANGELOG.md"
if (cd "$fixture" && "$repo_root/scripts/extract-release-notes.sh" 1.4.0+build-1 "$fixture/empty-notes.md" >/dev/null 2>&1); then
  echo "expected empty release notes extraction to fail" >&2
  exit 1
fi

echo "release script tests passed"
