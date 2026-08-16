#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]" >&2
  exit 2
fi

tag=$1
semver='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(\+([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?$'
if [[ ! $tag =~ $semver ]]; then
  echo "release tag is not strict SemVer: $tag" >&2
  exit 1
fi

version=${tag#v}
version_core=${version%%[-+]*}
IFS=. read -r major minor patch <<< "$version_core"
for component in "$major" "$minor" "$patch"; do
  if (( ${#component} > 5 )) || (( 10#$component > 65535 )); then
    echo "release version component exceeds desktop package limit 65535: $component" >&2
    exit 1
  fi
done

file_version=$(<VERSION)
file_version=${file_version%$'\n'}
if [[ $file_version != "$version" ]]; then
  echo "VERSION is $file_version, expected $version from $tag" >&2
  exit 1
fi

tag_commit=$(git rev-parse --verify "${tag}^{commit}")
head_commit=$(git rev-parse --verify HEAD)
if [[ $tag_commit != "$head_commit" ]]; then
  echo "checked-out commit $head_commit does not match $tag ($tag_commit)" >&2
  exit 1
fi

if ! git show-ref --verify --quiet refs/remotes/origin/main; then
  echo "origin/main is unavailable; fetch it before validating a release" >&2
  exit 1
fi
if ! git merge-base --is-ancestor "$tag_commit" origin/main; then
  echo "$tag does not point to a commit on protected origin/main" >&2
  exit 1
fi

if ! awk -v version="$version" '
  /^##[[:space:]]+\[/ {
    line = $0
    sub(/^##[[:space:]]+\[/, "", line)
    closing = index(line, "]")
    if (closing > 1 && substr(line, 1, closing - 1) == version && substr(line, closing + 1, 1) ~ /[[:space:]]|^$/) found = 1
  }
  END { exit !found }
' CHANGELOG.md; then
  echo "CHANGELOG.md has no entry for $version" >&2
  exit 1
fi

wails_version=$(node -p "require('./desktop/wails.json').info.productVersion")
if [[ $wails_version != "$version_core" ]]; then
  echo "desktop/wails.json productVersion is $wails_version, expected $version_core" >&2
  exit 1
fi

frontend_version=$(node -p "require('./desktop/frontend/package.json').version")
frontend_lock_version=$(node -p "require('./desktop/frontend/package-lock.json').version")
if [[ $frontend_version != "$version" || $frontend_lock_version != "$version" ]]; then
  echo "desktop frontend package versions are $frontend_version/$frontend_lock_version, expected $version" >&2
  exit 1
fi

cask_version=$(awk '/^[[:space:]]*version[[:space:]]+"/ { line = $0; sub(/^[^"]*"/, "", line); sub(/".*/, "", line); print line; exit }' packaging/homebrew/giztui-desktop.rb)
if [[ $cask_version != "$version" ]]; then
  echo "Homebrew cask template version is $cask_version, expected $version" >&2
  exit 1
fi

echo "release source validated: $tag at $tag_commit"
