# Overlay Backup Extraction — Design (App god-object refactor, step 4)

**Date:** 2026-06-21
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `refactor/overlay-backup`
**Issue:** #49 (graphify — `App` god object)

## Goal

Collapse the three identical reader-content backup mechanisms (help, preload status, prompt stats)
into a single reusable `overlayBackup` type, turning 9 flat `App` fields into 3 typed ones and
removing the triplicated save/restore pattern — with zero user-visible behavior change. Fourth step
of the incremental App decomposition (after vimState v1.17.0, commandState, searchState).

## Why this subsystem

The cleanest remaining slice:
- **Real duplication:** three overlays (`?` help, preload status, prompt stats) each back up the
  reader's text/header/title before showing, then restore + clear on close. The save/restore/clear
  code is the same shape three times.
- **9 cohesive fields, all in `app.go`** (lines ~78-86): `helpBackup{Text,Header,Title}`,
  `preloadBackup{Text,Header,Title}`, `promptStatsBackup{Text,Header,Title}`.
- **No threading:** all 33 accesses are on the event loop (overlay open/close from key handlers); no
  goroutine touches these fields. No mutex.
- Honest note: little *new testable logic* (it stores/returns three strings); the win is de-duplication
  + 6 fewer fields.

## The 9 fields → 3 (app.go ~78-86)

```go
helpBackupText/Header/Title         string  → helpBackup        overlayBackup
preloadBackupText/Header/Title      string  → preloadBackup     overlayBackup
promptStatsBackupText/Header/Title  string  → promptStatsBackup overlayBackup
```

No struct-literal initializers exist for these (zero value `""` is the default), so nothing to
remove there.

## Current behavior (verified)

For each overlay, the same three operations:
- **Save** (before showing the overlay): `a.<x>BackupText = text.GetText(false)`,
  `a.<x>BackupHeader = header.GetText(false)`, `a.<x>BackupTitle = textContainer.GetTitle()`.
  (help ~2564-2570, preload ~2625-2631, promptStats ~2743-2749)
- **Restore** (on close, guarded by `a.<x>BackupText != ""`): set `enhancedTextView.SetContent(...)`,
  `text.SetText(...)`, `header.SetText(...)`, `textContainer.SetTitle(...)` from the backup.
  (help ~2517-2546, preload ~2693-2722, promptStats ~2814-2843)
- **Clear** (after restore): set the three fields back to `""`.
  (help ~2551-2553, preload ~2727-2729, promptStats ~2848-2850)

## Architecture

New file `internal/tui/overlay_backup.go`:

```go
// overlayBackup snapshots the reader pane's content (text/header/title) before a full-pane overlay
// (help, preload status, prompt stats) replaces it, so the prior content can be restored on close.
// Event-loop only — no synchronization needed.
type overlayBackup struct {
	text   string
	header string
	title  string
}

func (b *overlayBackup) save(text, header, title string) {
	b.text = text
	b.header = header
	b.title = title
}

// active reports whether a backup is currently held (used as the restore guard).
func (b *overlayBackup) active() bool { return b.text != "" }

func (b *overlayBackup) clear() {
	b.text = ""
	b.header = ""
	b.title = ""
}
```

`App` composes three fields — `helpBackup`, `preloadBackup`, `promptStatsBackup overlayBackup` — in
place of the nine flat fields.

**Rewire pattern** (per overlay, e.g. help):
- Save block → `a.helpBackup.save(text.GetText(false), header.GetText(false), textContainer.GetTitle())`.
- Restore guard `if a.helpBackupText != ""` → `if a.helpBackup.active()`; reads `a.helpBackupText` /
  `a.helpBackupHeader` / `a.helpBackupTitle` → `a.helpBackup.text` / `.header` / `.title`.
- Clear block → `a.helpBackup.clear()`.

The UI effects (`SetContent`/`SetText`/`SetTitle`, the `enhancedTextView != nil` and widget casts)
stay in the App handlers — only the field storage moves into the type.

## Behavior preservation (non-negotiable)

Identical behavior: opening `?` help / preload status / prompt stats backs up the reader; closing
restores the exact prior text, header, and title; the empty-backup guard still skips restore when
nothing was saved.

## Testing

New `internal/tui/overlay_backup_test.go`:
- `save` then read `text`/`header`/`title` round-trips the three values.
- `active` is false on a zero value, true after `save` with non-empty text, false after `clear`.
- `clear` resets all three to `""`.

Existing behavior verified by `make test` (TUI suite) + a manual smoke test (open/close `?` help and
the prompt-stats / preload overlays; the reader content returns).

## Out of scope (YAGNI)

- Changing what the overlays show or how they open.
- Other App subsystems (AI summary pane, bulk selection) — future steps.

## Definition of Done

- [ ] `overlayBackup` type + methods in `internal/tui/overlay_backup.go`.
- [ ] `App` composes `helpBackup`/`preloadBackup`/`promptStatsBackup overlayBackup`; the 9 flat
      fields removed.
- [ ] All save/restore/clear sites in `app.go` rewired to `a.<x>Backup.save/active/clear` + field reads.
- [ ] `overlay_backup_test.go` covering save/active/clear.
- [ ] No stray `a.helpBackup(Text|Header|Title)` / `a.preloadBackup*` / `a.promptStatsBackup*`
      references remain.
- [ ] `make pre-commit-check` green; `go test -race ./internal/tui/...` green.
- [ ] No user-visible behavior change (manual overlay smoke test).
