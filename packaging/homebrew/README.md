# Homebrew distribution for GizTUI Desktop

GizTUI Desktop is a GUI app, so it ships as a **Homebrew Cask** (not a formula).
Casks live in a **tap** — a separate repo the maintainer owns.

## One-time setup

1. Create a public repo named **`homebrew-giztui`** under `ajramos`
   (the `homebrew-` prefix is required; the tap is then `ajramos/giztui`).
2. Add `giztui-desktop.rb` (from this folder) at `Casks/giztui-desktop.rb`.

Users then install with:

```bash
brew tap ajramos/giztui
brew install --cask giztui-desktop
```

## Per-release bump

After a release publishes `GizTUI-Desktop-<version>-universal.dmg`, update the
cask's `version` and `sha256` in the tap.

**Manual:**
```bash
VER=1.21.0
URL="https://github.com/ajramos/giztui/releases/download/v$VER/GizTUI-Desktop-$VER-universal.dmg"
SHA=$(curl -sSL "$URL" | shasum -a 256 | awk '{print $1}')
# edit Casks/giztui-desktop.rb: version "$VER", sha256 "$SHA"
```

**Automated:** from the release workflow (or a small job in the tap), use
[`dawidd6/action-homebrew-bump-cask`](https://github.com/dawidd6/action-homebrew-bump-cask)
with a `GH_TAP_TOKEN` secret that can push to the tap repo. Deferred to phase 2
in `docs/DESKTOP_DISTRIBUTION.md`.

## Signing note

The build is currently **unsigned**. `sha256 :no_check` in the cask lets it
install without a pinned checksum; switch to a real `sha256` (as above) once you
bump per release for a verified install. macOS notarization (phase 2) removes the
Gatekeeper warning entirely.
