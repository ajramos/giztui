# 🆘 Troubleshooting GizTUI

A practical guide to the problems people actually hit, with the fix for each.
If something here doesn't match what you see, grab your log (see below) and open
an issue — [How to report a bug](#-how-to-report-a-bug) tells you what to include.

> This is the **user-facing** guide. Developers tracking known internal defects
> should see [KNOWN_ISSUES.md](KNOWN_ISSUES.md) instead.

## 📂 Where things live

Everything lives under `~/.config/giztui/`:

| File | What it is |
|------|-----------|
| `config.json` | Your configuration (keys, theme, LLM, integrations) |
| `credentials.json` | Google OAuth **client** credentials (you download these) |
| `token.json` | The OAuth **token** minted after you authorize (auto-created) |
| `giztui.log` | The TUI log — the first thing to check when something misbehaves |
| `desktop.log` | The desktop app log (GizTUI Desktop only) |

Useful commands:

```bash
giztui --version          # exact version / commit (include this in bug reports)
giztui --setup            # re-run the interactive setup wizard
giztui --migrate-config   # add any new default config keys to your config.json
tail -f ~/.config/giztui/giztui.log   # watch the log live while reproducing
```

Inside the app: `?` opens help, `:config` shows the active configuration.

---

## 🔑 Credentials & authentication

### "Credentials not found"
GizTUI can't find your OAuth **client** file. Fix, in order:

1. Make sure the file is at `~/.config/giztui/credentials.json` (this is the file
   you download from Google Cloud — see
   [Gmail API Setup](GETTING_STARTED.md#gmail-api-setup)).
2. Or point at it explicitly: `giztui --credentials /path/to/credentials.json`.
3. Or set the environment variable `GMAIL_TUI_CREDENTIALS=/path/to/credentials.json`.
4. Run `giztui --setup` to be walked through it.

### "Access blocked: This app isn't verified"
Your OAuth consent screen is in *testing* mode and your account isn't a test
user. In the Google Cloud console → **APIs & Services → OAuth consent screen →
Test users**, add the Gmail address you're logging in with. Then retry.

### 403 `ACCESS_TOKEN_SCOPE_INSUFFICIENT` (or a feature says "insufficient scope")
Your saved token was minted with fewer scopes than the feature needs (this often
appears after upgrading, or when first using RSVP/calendar). Re-consent:

```bash
rm ~/.config/giztui/token.json
giztui --setup            # or just relaunch — it re-opens the Google auth flow
```

You'll be asked to grant access again; the new token carries the full scope set.

### 401 / "token expired or revoked" — auth loop on launch
Same fix as above: delete `token.json` and re-authorize. This also resolves the
case where you revoked GizTUI's access from your Google Account security page.

### RSVP / calendar actions don't work
Calendar access is a separate grant. Remove `~/.config/giztui/token.json` and
re-authorize (`giztui --setup`) so the token includes calendar scope.

---

## 📥 Nothing loads / empty inbox

- **Check the log** (`~/.config/giztui/giztui.log`) for an API error — it usually
  names the cause (scope, quota, network).
- **Wrong account?** If you use multiple accounts, confirm the active one
  (`Ctrl+A` opens the account picker).
- **Rate limit / quota (429):** wait a moment and press `R` to refresh.
- **Network:** GizTUI needs outbound HTTPS to `googleapis.com`. Behind a proxy,
  ensure `HTTPS_PROXY` is set in your shell.

---

## 🎨 Theme not loading / wrong colors

- Run `:theme` to open the picker and pick a theme live; if the picker shows the
  theme working, the issue is your config value.
- In `config.json`, the theme is referenced **by name** (e.g. `"gmail-dark"`),
  not by path. A typo falls back to the default theme.
- Custom themes must be valid JSON in the themes directory; an invalid file is
  skipped (check the log for a parse warning).
- Colors look flat/washed out? Your terminal needs **256-color** (or truecolor)
  support — see terminal rendering below.

---

## 🖥️ Terminal rendering (misaligned columns, broken emoji)

GizTUI is a TUI; a few issues are the terminal, not the app:

- **Columns look shifted / an emoji "eats" a character.** Some glyphs are
  *East-Asian-Wide* and render two cells wide in some terminals and one in
  others. Use a terminal with correct Unicode width handling (recent
  iTerm2/WezTerm/Kitty/Windows Terminal) and a font with good emoji coverage.
- **Boxes/borders look wrong.** Ensure `TERM` advertises 256 colors
  (`echo $TERM` → something like `xterm-256color`).
- **Everything is monochrome.** Your terminal or multiplexer is stripping colors;
  in tmux set `set -g default-terminal "tmux-256color"`.

---

## ✅ Bulk mode & selection

- Enter bulk mode with `v` (or `b`); the status bar shows the bulk hint.
- `Space` toggles the current row; `*` selects/clears **all**; `Esc` exits bulk.
- Actions in bulk apply to the whole selection: `a` archive, `d` trash, `m` move,
  `l` labels, `:star` star, `p` prompt.
- If the status bar looks "stuck" after a bulk action, press `Esc` to exit bulk
  mode and refresh.

---

## 🔄 Auto-refresh / new-mail not updating

- Auto-refresh is **opt-in**. Toggle it in-app or set it in `config.json`; the
  setting now persists across restarts.
- If it seems off despite the config saying on, relaunch — startup reads the
  config as the source of truth.

---

## 🧩 Missing options after an upgrade

New releases add config keys. If a new feature seems absent or a new shortcut
does nothing, your `config.json` predates it. Pull in the new defaults:

```bash
giztui --migrate-config       # or :config migrate inside the app
```

Your existing settings are preserved; only missing keys are added.

---

## 🍏 GizTUI Desktop (Wails app)

- **Remote images don't show.** By design, remote images are **off** until you
  load them (privacy); they're then fetched through the app's backend. Use the
  load-images action in the reader.
- **The window won't drag.** Drag from the top bar (it's the window drag region).
- **A feature errors with "account email not set".** Switch/select an account so
  account-scoped services (saved queries, rules, analyzer) initialize.
- **Logs:** `~/.config/giztui/desktop.log`.

---

## 🐛 How to report a bug

Open an issue at <https://github.com/ajramos/giztui/issues> with:

1. **Version:** output of `giztui --version`.
2. **OS + terminal** (e.g. macOS 14 + WezTerm; or "GizTUI Desktop").
3. **What you did** and **what happened** vs. what you expected.
4. **A log snippet** from `~/.config/giztui/giztui.log` (or `desktop.log`) around
   the moment it failed — redact any addresses/subjects you'd rather not share.

The log is local-only and never uploaded by GizTUI; you choose what to include.
