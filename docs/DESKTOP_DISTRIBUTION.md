# 📦 GizTUI Desktop — Distribution Design

Design for building and distributing the **Wails desktop client** across macOS,
Windows and Linux, and publishing it through package managers (Homebrew first).

> Status: **design / proposal**. Nothing here is wired yet. The current
> `release.yml` builds only the **TUI CLI** (`./cmd/giztui`) on Ubuntu; the
> desktop app is not built anywhere.

## 🎯 Goals

- One tagged release (`vX.Y.Z`) produces installable desktop artifacts for the
  three OSes, attached to the GitHub Release.
- Keyboard-first parity aside, users can install with a one-liner where possible
  (`brew install --cask …`), or download a native installer.
- The TUI CLI release keeps working exactly as today (untouched).

## 🚫 Non-goals (deferred to phase 2)

- **Code signing & notarization** (macOS Developer ID, Windows Authenticode).
  Phase 1 ships **unsigned**; users get a Gatekeeper/SmartScreen warning and
  open via right-click → Open (macOS) / "More info → Run anyway" (Windows).
- winget / Scoop / AUR / Flatpak / Snap. (Homebrew cask is phase 1.)
- `.deb` / `.rpm` repositories. Phase 1 Linux ships an **AppImage** + tarball.
- Auto-update inside the app (Sparkle/electron-updater style).

## 🧱 Why a build matrix (no cross-compile)

Wails links against each platform's **native webview** (WKWebView / WebView2 /
WebKitGTK) through CGO, so a build must run **on its target OS**. There is no
reliable cross-compilation. Therefore distribution is a **GitHub Actions matrix**
with one job per OS.

| OS | Runner | `wails build` | Package | Release asset |
|----|--------|---------------|---------|----------------|
| macOS | `macos-latest` | `-platform darwin/universal` → `GizTUI Desktop.app` (Intel+ARM) | `.dmg` via `create-dmg` (fallback `hdiutil`) | `GizTUI-Desktop-<ver>-universal.dmg` |
| Windows | `windows-latest` | `-platform windows/amd64 -nsis` → installer + `.exe` | NSIS (built into Wails) | `GizTUI-Desktop-<ver>-setup.exe`, `…-windows-amd64.zip` (portable) |
| Linux | `ubuntu-22.04` | `-platform linux/amd64` → binary | AppImage (`linuxdeploy`) + tarball | `GizTUI-Desktop-<ver>-x86_64.AppImage`, `…-linux-amd64.tar.gz` |

Notes:
- **macOS universal**: `macos-latest` builds both arches; one `.app` for everyone.
- **Windows NSIS**: Wails generates the installer from `desktop/build/windows/`
  templates; the runner needs `makensis` on PATH (`choco install nsis`).
- **Linux deps**: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` (or `4.1`), plus
  `linuxdeploy` + `linuxdeploy-plugin-appimage` to make the AppImage.
- The desktop is a **nested Go module** (`desktop/`), so jobs `cd desktop` and
  install the Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`).

## 🔁 Workflow shape

New workflow **`.github/workflows/release-desktop.yml`**, triggered on the same
`v*` tags as the CLI release (and `workflow_dispatch` for manual runs). Kept
separate from `release.yml` so a desktop-packaging failure never blocks the CLI
release, and vice-versa.

```
on:
  push: { tags: ['v*'] }
  workflow_dispatch: { inputs: { version: {...} } }

jobs:
  macos:   { runs-on: macos-latest,   steps: [setup, wails build universal, create-dmg, upload] }
  windows: { runs-on: windows-latest, steps: [setup, choco nsis, wails build -nsis, zip portable, upload] }
  linux:   { runs-on: ubuntu-22.04,   steps: [apt webkit deps, wails build, appimage, tar, upload] }
  release: { needs: [macos, windows, linux], steps: [download artifacts, gh release upload] }
```

Each build job uploads its files as **workflow artifacts**; a final `release`
job attaches them to the GitHub Release created by the tag (using
`softprops/action-gh-release` or `gh release upload`, matching how `release.yml`
already publishes).

## 🏷️ Versioning

- Single source of truth stays the repo **`VERSION`** file + git tag.
- The desktop version is injected into the `.app`/installer via
  `wails build -ldflags "-X main.version=<ver>"` and the macOS `Info.plist`
  `CFBundleShortVersionString` (Wails reads `desktop/build/darwin/Info.plist`).
- Asset filenames embed `<ver>` from the tag (strip the leading `v`).

## 🍺 Homebrew (phase 1 distribution channel)

A **Cask** (not a Formula — the deliverable is a GUI `.app`, not a CLI) in a
dedicated tap repo the maintainer owns: **`ajramos/homebrew-giztui`**.

`Casks/giztui-desktop.rb`:
```ruby
cask "giztui-desktop" do
  version "1.21.0"
  sha256 "<dmg-sha256>"
  url "https://github.com/ajramos/giztui/releases/download/v#{version}/GizTUI-Desktop-#{version}-universal.dmg"
  name "GizTUI Desktop"
  desc "Visual Gmail client (Wails) sharing the GizTUI service layer"
  homepage "https://github.com/ajramos/giztui"
  app "GizTUI Desktop.app"
  # Unsigned build: needed until notarization lands, else Gatekeeper blocks it.
  # (Homebrew removes the quarantine attr on cask installs by default.)
end
```

Install: `brew tap ajramos/giztui && brew install --cask giztui-desktop`.

**Auto-bump**: after a successful release, a step opens a PR (or pushes) to the
tap updating `version` + `sha256` from the new DMG. Options: a small script with
a `GH_TAP_TOKEN` secret, or `dawidd6/action-homebrew-bump-cask`. Manual bump is
fine for the first releases.

## ✍️ Signing plan (phase 2)

Unsigned is acceptable for early adopters but is the main friction point. When we
invest:

- **macOS**: import a *Developer ID Application* cert into the keychain
  (`APPLE_CERT_P12` + password secrets), `codesign --deep --options runtime` the
  `.app`, then `xcrun notarytool submit` + `stapler staple` the `.dmg`
  (`APPLE_ID` / `APPLE_APP_PASSWORD` / `APPLE_TEAM_ID` secrets).
- **Windows**: Authenticode `signtool` with a cert (`WINDOWS_CERT` secrets), or
  Azure Trusted Signing.

Everything else in the pipeline stays; signing is inserted between build and
package. Homebrew cask then drops the quarantine workaround.

## 🚦 Rollout phases

1. **Phase 1 (this design):** matrix workflow → unsigned `.dmg` + NSIS `.exe` +
   AppImage/tarball attached to each release. Homebrew tap + cask for macOS,
   bumped manually the first time.
2. **Phase 2:** macOS notarization + Windows signing (secrets). Auto-bump the
   cask.
3. **Phase 3 (optional):** winget/Scoop manifests, `.deb`/`.rpm`, AUR.

## ❓ Open decisions (defaults chosen; override any)

1. **Platforms in phase 1** — default **all three**. If you only care about
   macOS right now, we ship Mac only and add Windows/Linux later (less CI to
   babysit).
2. **Same tag as the CLI release** — default **yes** (`v1.21.0` builds both CLI
   and desktop). Alternative: a `desktop-v*` tag to release the desktop on its
   own cadence.
3. **Homebrew tap name** — default `ajramos/homebrew-giztui` (⇒
   `brew tap ajramos/giztui`). The tap repo must be created by the maintainer;
   the cask file and bump step live here.
4. **Linux format** — default **AppImage + tarball**. `.deb`/`.rpm` deferred.
5. **Unsigned in phase 1** — default **yes**. If you have an Apple Developer
   account we can wire notarization from the start instead.

## ⚠️ What I can and can't verify from here

The workflow/scripts/manifests can be written and committed from any environment,
but the **macOS and Windows builds cannot be run or tested on the Linux dev
container** — they are validated on the first CI run (or on your Mac for the
local DMG script). Expect one or two iterations to shake out per-runner details
(webkit dev-package name, NSIS path, notarization entitlements).
