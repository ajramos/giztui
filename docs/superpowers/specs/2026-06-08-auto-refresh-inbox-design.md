# Auto-refresh Inbox Toggle — Design

**Date:** 2026-06-08
**Status:** Approved (brainstorm), pending implementation plan
**Backlog item:** #1 (auto-refresh inbox toggle)

## Summary

A configurable, opt-in toggle that periodically checks the inbox for new mail in
the background. The refresh is **non-disruptive by design**: it never wipes the
list or flashes a spinner. When new mail arrives and the UI is in a safe state,
new messages are inserted incrementally at the top of the list (cursor
preserved); otherwise the status bar shows a pending count (`📬 N`) until the
user manually refreshes with `R`.

## Motivation

Today the only way to see new mail is to press `R` (manual refresh). Users have
no signal that new mail exists, so they refresh blindly. This feature surfaces
"there is new mail" passively and, when safe, loads it for free — without the
hostile experience of a periodic full reload.

## Key Constraint That Shapes the Design

The existing `reloadMessages()` / `reloadMessagesFlat()`
(`internal/tui/messages.go:234`, `:313`) is **destructive**: it calls
`table.Clear()`, shows a "🔄 Loading messages…" spinner, wipes `messagesMeta`
and the ID cache, resets pagination to the first 50, and clears remote search
mode (`messages.go:336-341`). That is acceptable for an explicit `R` press but
hostile as a periodic background action.

**Therefore auto-refresh does NOT reuse the destructive reload for its happy
path.** Because detection already computes exactly which message IDs are new,
the safe-state path is an **incremental prepend**: fetch metadata for only the
new IDs, insert their rows at the top of the table, and shift the selected row
index by `+N` so the cursor stays on the same message. No `table.Clear()`, no
spinner, no impact on pagination or search. The destructive reload remains
exclusively for the manual `R`.

## Behavior (Hybrid Model)

A background ticker fires every `interval` (default **5m**, configurable).

Auto-refresh **only operates when the displayed view is the plain flat inbox**.
When the user is in a search, a different folder, or threading mode, the ticker
idles — it does not poll and does not notify. (Pressing `R` still works
normally.) This keeps "new" unambiguous: it is always relative to the inbox the
user is looking at.

On each tick (in a goroutine):

1. If auto-refresh is disabled, or the view is not the plain inbox, or a reload
   is already in progress (`SetMessagesLoading` true) → skip this tick.
2. Lightweight fetch of the first page of inbox message IDs; diff against the
   currently-known IDs (`GetMessageIDs()`) → `newIDs`.
3. If `newIDs` is empty → refresh the indicator only; done.
4. If `newIDs` is non-empty:
   - **Safe state** (plain inbox AND not composing AND no picker open
     (`ActivePicker == PickerNone`) AND not in bulk-select mode) → **incremental
     prepend**: fetch metadata for `newIDs`, insert rows at the top, update
     `a.ids` / `a.messagesMeta`, shift the selected row by `+len(newIDs)`. Clear
     the pending counter.
   - **Not-safe state** → do not touch the list; set the status pending count to
     `len(newIDs)` (`📬 N`) until the user presses `R`.

### Scope cuts (YAGNI for v1)

- v1 ignores messages **removed** elsewhere (archived/read/deleted in another
  client). The incremental prepend only adds new arrivals. Full reconciliation
  remains the job of the manual `R`.
- v1 operates on the **default inbox view only**. Background polling of the inbox
  while the user views a search/folder is intentionally out of scope (avoids an
  ambiguous "new relative to what?" counter).
- Threading mode is out of scope for the prepend path; while in threading view
  the ticker idles.

## State & Configuration

New top-level config block:

```json
"auto_refresh": {
  "enabled": false,
  "interval": "5m"
}
```

- `enabled` defaults to `false` (opt-in — no unexpected network polling).
- `interval` is a Go duration string parsed with `time.ParseDuration`. A sane
  minimum (**1m**) is enforced; values below the minimum are clamped (with a
  logged warning) to avoid hammering the Gmail API.

Runtime control mirrors the existing session-toggle pattern (`M` /
LLM touch-up): the keybinding and command flip **session state** and start/stop
the ticker; they do **not** rewrite config. Config sets only the startup
default.

- Keybinding: `Keys.AutoRefresh` (configurable; default proposed below).
- Command parity: `:autorefresh` / `:ar` toggles; `:autorefresh <duration>`
  (e.g. `:ar 2m`) also sets the interval at runtime.

**Open item for spec review:** the default key. `R` is taken (Refresh). The
command (`:autorefresh` / `:ar`) works regardless of the key, so a default key is
a convenience, not a requirement. Candidate defaults to confirm with the user
(any free, non-conflicting key works — final pick is the user's during spec
review):

- `Ctrl+R` — mnemonic ("Refresh"), pairs naturally with the existing `R`.
- A two-key chord under an existing prefix, if the app has one.
- Leave it **unbound by default** and rely on the command only (most
  conservative; zero risk of clobbering an existing binding).

Recommendation: leave unbound by default and ship the command; let users opt into
a key via config. Revisit if the user prefers a bound default.

## Architecture (Service-First)

Per CLAUDE.md, business logic lives in `internal/services/`; the TUI only
orchestrates UI and the ticker lifecycle.

**New `AutoRefreshService`** (interface in
`internal/services/interfaces.go`, impl in
`internal/services/auto_refresh_service.go`):

- Thread-safe state: `IsEnabled() bool`, `SetEnabled(bool)`,
  `Interval() time.Duration`, `SetInterval(time.Duration) error`.
- Detection: `CheckForNewMessages(ctx, query string, knownIDs []string)
  (newIDs []string, err error)` — wraps the Gmail list call and the diff. Pure,
  testable in isolation with a mocked client.

**`App` (TUI)** owns:

- The ticker goroutine lifecycle: `startAutoRefresh()` / `stopAutoRefresh()`,
  created/torn down when the toggle flips and stopped in `Shutdown()`.
- The per-tick orchestration: read service state, call detection, decide
  safe-vs-notify, and perform either the incremental prepend or the status
  indicator update.

Threading rules (CLAUDE.md): detection and metadata fetch run in goroutines; UI
updates follow existing patterns; **no `QueueUpdateDraw` in ESC/cleanup/stream
paths**. The ticker goroutine selects on a stop channel and the app context so
it exits cleanly on shutdown.

All user-facing messages go through `a.GetErrorHandler()` (`ShowInfo`, etc.).
Any new UI uses `GetComponentColors(...)` for theming (the status indicator
reuses the existing status baseline styling).

## Status Bar Indicator

Extends `statusBaseline()` (`internal/tui/status.go:82`), which already appends
persistent indicators (e.g. ` | 🧠` / ` | 🧾` for LLM touch-up):

- Append ` | ⟳` when auto-refresh is enabled (omitted when disabled).
- When there are `N` pending new messages not yet loaded (not-safe state),
  append ` 📬N`. It clears when the messages are loaded (prepend) or on manual
  `R`.

## Multi-Account

The ticker follows the active account. On account switch (`internal/tui/
accounts.go`), reset the known-ID baseline and the pending counter so the next
tick computes "new" relative to the newly active account's inbox.

## Error Handling

- Detection failures are logged (and optionally surfaced once via
  `ShowWarning`); a failed tick does not stop the ticker — the next tick retries.
- Network errors never crash the UI; the list is left untouched on a failed
  tick.

## Testing

- **Unit:** `CheckForNewMessages` diff logic (new IDs, no new IDs, all known);
  interval parsing and clamping to the minimum; the safe-vs-not-safe decision
  predicate given mocked UI state.
- **TUI:** toggle on/off updates ticker lifecycle and indicator; prepend inserts
  rows and shifts the cursor by `+N`; not-safe state sets the counter without
  touching the list.
- **E2E:** real-app run via `/usr/bin/tmux` (not the zsh alias) against a real
  account — enable auto-refresh, observe a prepend on new mail while on the
  inbox, and a `📬N` counter while in a picker or composing (while in a
  search/folder the ticker idles and shows neither). (Hard-won lesson:
  real-app E2E catches what unit tests miss.)

## Command Parity Checklist (CLAUDE.md)

- Keybinding `Keys.AutoRefresh` → command `:autorefresh` / `:ar`.
- Added to `executeCommand()` and `generateCommandSuggestion()` in
  `internal/tui/commands.go`.
- Documented in `docs/KEYBOARD_SHORTCUTS.md`.

## Files Touched (anticipated)

- `internal/config/config.go` — `AutoRefreshConfig` block, defaults,
  `Keys.AutoRefresh`.
- `internal/services/interfaces.go` — `AutoRefreshService` contract.
- `internal/services/auto_refresh_service.go` — implementation.
- `internal/tui/app.go` — service init, ticker lifecycle, `Shutdown()` teardown.
- `internal/tui/messages.go` — incremental prepend helper.
- `internal/tui/status.go` — indicator in `statusBaseline()`.
- `internal/tui/commands.go` — `:autorefresh` command + suggestion.
- `internal/tui/keys.go` — key handler.
- `internal/tui/accounts.go` — baseline reset on account switch.
- `docs/KEYBOARD_SHORTCUTS.md` — documentation.
- Tests under `internal/services/` and `internal/tui/`.
