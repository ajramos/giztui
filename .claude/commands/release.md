---
description: "Prepare, validate, or inspect a GizTUI release"
---

# Release Management: $ARGUMENTS

Use `docs/RELEASE_PROCEDURE.md` as the canonical release contract. Never create
a GitHub release or upload assets manually. The only publication path is the
protected, tag-triggered `.github/workflows/release.yml` workflow.

## Usage

- `/release status` - inspect versions, branch state, recent tags, PR checks, and
  the latest release workflow.
- `/release validate` - run local product and release-source validation without
  publishing.
- `/release prepare X.Y.Z[-prerelease]` - prepare all version-bearing files and
  changelog content, then stop for review.
- `/release publish X.Y.Z[-prerelease]` - verify the prepared commit is merged to
  protected `main`, create and locally validate the tag, preview the push, and
  request confirmation before pushing it.
- `/release hotfix X.Y.Z` - use the same preparation and validation gates for a
  patch release. Hotfixes have no reduced-validation path.

## Version Sources

Every preparation must update these together:

```text
VERSION                                      full version, without v
internal/version/version.go                  full version
CHANGELOG.md                                 ## [full version] - YYYY-MM-DD
packaging/homebrew/giztui-desktop.rb         full version
desktop/frontend/package.json                full version
desktop/frontend/package-lock.json           full version
desktop/wails.json info.productVersion       numeric MAJOR.MINOR.PATCH core
```

Run `go test ./internal/version` after editing. Do not use `X.Y.Z+1` as a patch
increment; `+...` is SemVer build metadata.

## Validate

```bash
make ci-tools
make ci
git diff --check
```

`make ci` is the canonical local product gate. GitHub additionally runs the OS
matrix, Trivy SARIF, dependency review, and the protected `required` aggregator.

Release-specific validation requires a local tag on the exact clean commit:

```bash
TAG=vX.Y.Z
git tag "$TAG"
./scripts/validate-release.sh "$TAG"
```

Delete only an unpublished local tag if validation fails. Never move or replace
a pushed or published release tag.

## Publish

Before proposing a tag push, verify:

- The prepared commit is on `origin/main`.
- The commit has a successful `required` check.
- The worktree is clean and local `main` is current.
- The tag passes `scripts/validate-release.sh` locally.
- Stable releases are still allowed by the current tracker state; otherwise use
  the planned `-rc.N` rehearsal version.

Pushing a protected `v*` tag is a sensitive action. Show the exact tag and commit
SHA and obtain confirmation before running `git push origin "$TAG"`.

After the push, monitor the run for that tag. Verify `SHA256SUMS`, each CycloneDX
SBOM, per-file GitHub attestations constrained to the release workflow/tag/SHA,
native package metadata, and Homebrew behavior as specified in
`docs/RELEASE_PROCEDURE.md`.

## Failure Recovery

- Never bypass the workflow because CI or GitHub infrastructure is unavailable.
- Rerun the original workflow for transient external failures.
- Fix source/build failures in a new version; do not move a published tag.
- If Homebrew promotion alone fails, rerun its idempotent job.
- Rollbacks are new patch releases that revert the defective change, not tag or
  asset replacement.
