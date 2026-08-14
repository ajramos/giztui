# Release Procedure

GizTUI releases are produced only by the tag-triggered GitHub Actions workflow in
`.github/workflows/release.yml`. The workflow validates the exact source commit,
builds every CLI and desktop target with read-only jobs, creates checksums and
CycloneDX SBOMs, attests the artifacts, and publishes once through the protected
`release` environment.

Do not create releases or upload release assets manually.

## Release Contract

A release tag must satisfy all of these conditions:

- The tag is strict SemVer: `vMAJOR.MINOR.PATCH`, optionally with a SemVer
  prerelease or build suffix.
- The tag resolves to the current protected `main` history.
- The tagged commit has a successful required CI check named `required`.
- The tag version without the leading `v` exactly matches `VERSION`,
  `internal/version/version.go`, the changelog heading, and the Homebrew cask
  template.
- `desktop/wails.json` contains the numeric `MAJOR.MINOR.PATCH` core because
  native package version fields cannot represent SemVer suffixes.
- Native file/build versions use the protected `main` commit count so release
  candidates and stable packages remain distinguishable.
- Every numeric version component is at most `65535`, the Windows package
  metadata limit.

The release workflow has no manual-dispatch path. This keeps GitHub provenance
bound to the tag event. Rerun the original workflow run after correcting an
external transient failure.

## Prepare A Release

1. Choose the next SemVer version. Use a prerelease such as `1.28.0-rc.1` for a
   release-candidate rehearsal.
2. Update all version sources in the feature branch:

```text
VERSION                                      full version, without v
internal/version/version.go                  full version
CHANGELOG.md                                 ## [full version] - YYYY-MM-DD
packaging/homebrew/giztui-desktop.rb         full version
desktop/frontend/package.json + lock          full version
desktop/wails.json info.productVersion       numeric version core
```

3. Add substantive release notes under the new changelog heading.
4. Run the same fail-closed gate used by CI:

```bash
make ci
git diff --check
```

5. Commit the release preparation, push the branch, merge it through a pull
   request, and wait for `CI / required` on the resulting `main` commit.
6. Update local `main` and verify the intended commit:

```bash
git switch main
git pull --ff-only origin main
git status --short
git log -1 --oneline
```

The working tree must be clean before tagging.

## Trigger The Workflow

Create the tag only after the prepared commit is on protected `main` and its
required check is successful:

```bash
VERSION=$(tr -d '\n' < VERSION)
git tag "v${VERSION}"
git push origin "v${VERSION}"
```

The protected tag starts the `Release` workflow. Monitor it with:

```bash
gh run list --workflow release.yml --limit 1
gh run watch RUN_ID
```

## Pipeline Behavior

The workflow performs these stages in order:

1. `Validate release source` checks SemVer, all version sources, exact tag/HEAD
   agreement, protected `main` ancestry, and the required CI check.
2. Read-only jobs build six CLI archives, native desktop packages for macOS,
   Windows, and Linux, and three reproducible CycloneDX SBOMs.
3. The publisher requires the complete expected artifact set, generates the
   release-specific Homebrew cask and central `SHA256SUMS`, and creates GitHub
   build-provenance attestations.
4. Assets are uploaded to a draft release. The release becomes public only
   after all mandatory builds, metadata assertions, checksums, SBOMs, and
   attestations succeed.
5. Stable releases run an idempotent post-publication Homebrew job. The job is
   blocking and requires `HOMEBREW_TAP_TOKEN`; prereleases attach a candidate
   cask but do not change the stable tap.

Build jobs check out the validated commit SHA rather than resolving the tag
again. Immediately before publication, the publisher also confirms that the
remote tag still resolves to that SHA.

## Expected Assets

Each release contains:

- Four Unix CLI tarballs: Linux and macOS, amd64 and arm64.
- Two Windows CLI zip files: amd64 and arm64.
- macOS universal DMG and zip desktop packages.
- Windows amd64 NSIS installer and portable zip.
- Linux amd64 AppImage and tarball.
- CLI Go, desktop Go, and desktop frontend CycloneDX JSON SBOMs.
- `giztui-desktop.rb`, generated with the universal DMG checksum.
- `SHA256SUMS` covering every preceding release asset.

Desktop binaries are currently unsigned. The workflow has explicit package
boundaries where Developer ID/notarization and Authenticode can be added when
commercial signing identities are available.

## Verify A Release Candidate

Stable releases remain paused until a complete `-rc.N` rehearsal has passed.
For the candidate:

```bash
TAG=vX.Y.Z-rc.1
mkdir -p /tmp/giztui-release-check
gh release download "$TAG" --dir /tmp/giztui-release-check
(cd /tmp/giztui-release-check && sha256sum --check SHA256SUMS)
gh attestation verify /tmp/giztui-release-check/* --repo ajramos/giztui
```

Also verify:

- The GitHub release is marked as a prerelease and is not a draft.
- Every expected asset is present and non-empty.
- Each SBOM parses as valid CycloneDX JSON.
- The macOS bundle identifier is `com.ajramos.giztui.desktop`, the short version
  is the numeric version core, and the minimum OS is macOS 12.
- The Windows installer reports the numeric product-version core.
- The Linux desktop file and AppImage launch correctly.
- CLI `--version` reports the full version and tagged source commit.
- The attached Homebrew cask has the candidate version and DMG checksum but the
  stable tap remains unchanged.

Record the rehearsal evidence in the release tracking issue before reopening
stable releases.

## Verify A Stable Release

```bash
TAG=vX.Y.Z
gh release view "$TAG" --json isDraft,isPrerelease,assets
gh run list --workflow release.yml --limit 1 --json status,conclusion,url
gh release download "$TAG" --dir /tmp/giztui-release-check
(cd /tmp/giztui-release-check && sha256sum --check SHA256SUMS)
gh attestation verify /tmp/giztui-release-check/* --repo ajramos/giztui
curl -fsSL https://raw.githubusercontent.com/ajramos/homebrew-giztui/main/Casks/giztui-desktop.rb
```

Test at least one applicable installation path and confirm `giztui --version`.
For Homebrew, confirm the tap version and DMG checksum match the release asset.

## Failure Recovery

- If a build, SBOM, or attestation fails, fix the source and create a new
  version. Never move a published tag.
- If an external service fails before publication, rerun the original workflow.
  Existing draft assets are replaced, while a complete published release is
  detected and verified without being overwritten.
- If stable Homebrew promotion fails after GitHub publication, rerun the failed
  `Update stable Homebrew cask` job. An already-correct cask is a successful
  no-op.
- If a published release has a product defect, prepare a patch release. Do not
  replace its tag or artifacts.
- If a tag was pushed before its commit reached `main` or before required CI
  succeeded, validation fails closed. Delete an unpublished erroneous tag only
  through the protected-tag maintainer process, correct the source, and create
  the intended version tag.

## Related Documentation

- [Release workflow](../.github/workflows/release.yml)
- [Reusable desktop builders](../.github/workflows/release-desktop.yml)
- [Desktop distribution](DESKTOP_DISTRIBUTION.md)
- [Homebrew distribution](../packaging/homebrew/README.md)
- [Testing](TESTING.md)
