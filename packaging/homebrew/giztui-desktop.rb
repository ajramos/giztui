# Homebrew Cask for GizTUI Desktop.
#
# This file belongs in a *tap* repo (a repo named `homebrew-giztui` owned by the
# maintainer): copy it to `ajramos/homebrew-giztui` at `Casks/giztui-desktop.rb`.
# Then users install with:
#
#   brew tap ajramos/giztui
#   brew trust ajramos/giztui
#   brew install --cask giztui-desktop
#
# This is the source-of-truth template. On a tagged release the
# release.yml `homebrew` job promotes a generated cask with the new
# `version` and the real `sha256` of the universal .dmg (needs the
# HOMEBREW_TAP_TOKEN secret — see packaging/homebrew/README.md). Until macOS
# notarization lands the build is unsigned and macOS may require right-click,
# Open, and explicit confirmation on first launch.
cask "giztui-desktop" do
  version "1.27.0"
  sha256 :no_check # the tap copy is pinned to the DMG's sha256 by CI

  url "https://github.com/ajramos/giztui/releases/download/v#{version}/GizTUI-Desktop-#{version}-universal.dmg"
  name "GizTUI Desktop"
  desc "Visual Gmail client (Wails) sharing the GizTUI service layer"
  homepage "https://github.com/ajramos/giztui"

  depends_on macos: ">= :monterey"

  app "GizTUI Desktop.app"

  zap trash: [
    "~/Library/Application Support/giztui",
    "~/Library/Logs/giztui",
  ]
end
