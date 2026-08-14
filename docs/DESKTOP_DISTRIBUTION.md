# 📦 GizTUI Desktop — Distribution Design

Design for building and distributing the **Wails desktop client** across macOS,
Windows and Linux, and publishing it through package managers (Homebrew first).

> Status: native builders are reusable jobs in
> `.github/workflows/release-desktop.yml`; `.github/workflows/release.yml` is the
> only tag trigger and publisher. A release candidate rehearsal on hosted
> macOS, Windows, and Linux runners is required before stable releases resume.

## 🎯 Goals

- One tagged release (`vX.Y.Z`) produces installable desktop artifacts for the
  three OSes, attached to the GitHub Release.
- Keyboard-first parity aside, users can install with a one-liner where possible
  (`brew install --cask …`), or download a native installer.
- A desktop failure blocks publication of the combined CLI and desktop release.

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
| macOS | `macos-15` | `-platform darwin/universal` -> `GizTUI Desktop.app` (Intel+ARM) | `.dmg` via `hdiutil` + zip | `GizTUI-Desktop-<ver>-universal.dmg`, `...-macOS-universal.zip` |
| Windows | `windows-2025` | `-platform windows/amd64 -nsis` -> installer + `.exe` | pinned NSIS | `...-windows-amd64-installer.exe`, `...-windows-amd64-portable.zip` |
| Linux | `ubuntu-22.04` | `-platform linux/amd64` -> binary | verified `linuxdeploy` AppImage + normalized tarball | `...-linux-amd64.AppImage`, `...-linux-amd64.tar.gz` |

Notes:
- **macOS universal**: `macos-latest` builds both arches; one `.app` for everyone.
- **Windows NSIS**: Wails generates the installer from `desktop/build/windows/`
  templates; the runner needs `makensis` on PATH (`choco install nsis`).
- **Linux deps**: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` (or `4.1`), plus
  `linuxdeploy` + `linuxdeploy-plugin-appimage` to make the AppImage.
- The desktop is a **nested Go module** (`desktop/`), so jobs `cd desktop` and
  install the Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`).

## 🔁 Workflow shape

`.github/workflows/release-desktop.yml` is callable only. The tag-triggered
`.github/workflows/release.yml` validates and exports an immutable source SHA,
then passes that SHA to all native builders. Build jobs have read-only
permissions and no persisted checkout credentials.

```
jobs:
  validate: { steps: [tag, version, ancestry, required-check validation] }
  macos:   { runs-on: macos-15,       steps: [setup, wails build universal, hdiutil, upload] }
  windows: { runs-on: windows-2025,   steps: [setup, pinned nsis, wails build -nsis, zip portable, upload] }
  linux:   { runs-on: ubuntu-22.04,   steps: [apt webkit deps, wails build, appimage, tar, upload] }
  publish: { needs: [cli, sbom, desktop], steps: [checksums, attest, draft, upload, publish] }
```

Each build job uploads mandatory workflow artifacts. One environment-gated
publisher validates the complete set, generates checksums and SBOM provenance,
uploads to a draft, and publishes only after all mandatory work succeeds.

## 🏷️ Versioning

- Single source of truth stays the repo **`VERSION`** file + git tag.
- The full version is injected into application code via linker flags. Native
  product versions use the numeric SemVer core, while file/build versions use
  the protected `main` commit count as a monotonic build number.
- The macOS bundle identifier is `com.ajramos.giztui.desktop` and the deployment
  target is macOS 12.
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

Install: `brew tap ajramos/giztui && brew trust ajramos/giztui && brew install --cask giztui-desktop`.
(`brew trust` is required once — recent Homebrew refuses casks from untrusted
third-party taps. Audit **by name** after trusting: `brew audit --cask giztui-desktop`.)

**Auto-bump**: after a successful stable release, the blocking `homebrew` job
promotes the release's generated and attested cask to the tap using
`HOMEBREW_TAP_TOKEN`. Prereleases leave the stable tap unchanged.

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

## ⚠️ What I can and can't verify from here

The workflow/scripts/manifests can be written and committed from any environment,
but the **macOS and Windows builds cannot be run or tested on the Linux dev
container** — they are validated on the first CI run (or on your Mac for the
local DMG script). Expect one or two iterations to shake out per-runner details
(webkit dev-package name, NSIS path, notarization entitlements).
