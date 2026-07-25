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
brew trust ajramos/giztui        # required once: third-party taps are untrusted by default
brew install --cask giztui-desktop
```

> **`brew trust` is required.** Recent Homebrew refuses to load a cask from a
> third-party tap until it is trusted, failing with *"Refusing to load cask …
> from untrusted tap ajramos/giztui"*. Running `brew trust ajramos/giztui` (or
> the per-cask `brew trust --cask ajramos/giztui/giztui-desktop`) clears it. This
> is expected for any tap outside `homebrew/core`; document it wherever you point
> users at the tap so the message doesn't scare them off.
>
> The same gate applies to `brew audit` — audit **by name** after trusting
> (`brew audit --cask giztui-desktop`), not by file path (`brew audit [path]` is
> disabled in current Homebrew).

## Per-release bump

**Automated (default).** The `homebrew` job in `.github/workflows/release-desktop.yml`
runs on every tagged release: it downloads the freshly built
`GizTUI-Desktop-<version>-universal.dmg`, computes its `sha256`, and pushes an
updated `Casks/giztui-desktop.rb` (pinned `version` + `sha256`) to the tap repo
`ajramos/homebrew-giztui`. So `brew upgrade --cask giztui-desktop` just works.

**One-time setup:** add a repository secret **`HOMEBREW_TAP_TOKEN`** to
`ajramos/giztui` — a fine-grained PAT (or classic token with `repo`) that has
**contents: write** on `ajramos/homebrew-giztui`. Without it the job logs a
warning and skips; the release itself never fails.

**Manual fallback** (e.g. the very first tap seed, or the secret isn't set yet):
```bash
VER=1.22.0
URL="https://github.com/ajramos/giztui/releases/download/v$VER/GizTUI-Desktop-$VER-universal.dmg"
SHA=$(curl -sSL "$URL" | shasum -a 256 | awk '{print $1}')
# edit Casks/giztui-desktop.rb: version "$VER", sha256 "$SHA"
```

## Signing note

The build is currently **unsigned**. `sha256 :no_check` in the cask lets it
install without a pinned checksum; switch to a real `sha256` (as above) once you
bump per release for a verified install. macOS notarization (phase 2) removes the
Gatekeeper warning entirely.
