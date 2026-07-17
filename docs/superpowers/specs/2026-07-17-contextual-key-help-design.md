# Contextual key-help cheat-sheet (design)

**Date:** 2026-07-17
**Status:** approved (design), pending spec review → implementation plan
**Related:** GitHub issue #36 (context-aware help filtered by current focus). This delivers the reusable mechanism and wires the first consumer (Action Plan); other panels are folded in later.

## Problem

Panels like the Action Plan have grown many keys (navigation, `space` exclude, action keys, `m` move, `i` prompt, `Ctrl+R` remember, `c` confirm, `g` assist, `.` accept, `Esc`). The one-line footer can only advertise a couple at a time, so users can't see everything available in the current context. `?` today shows the **global** help screen — everything, requiring scrolling — not the keys of the panel you're in.

## Goal

A reusable "key cheat-sheet" mechanism: pressing `?` inside a panel shows a focused, read-only list of THAT panel's keys, `Esc` returns exactly where you were. Wire it fully for the Action Plan now; leave the base ready for other panels (labels, prompts, …) to adopt later. `?` outside any panel keeps showing the global help unchanged.

## Decisions (from brainstorming)

| Question | Decision |
|---|---|
| Scope | Reusable mechanism, but wire ONLY the Action Plan now |
| Presentation | Cheat-sheet **inside the panel** via body-swap (same pattern as the `v` prompt viewer), Esc returns |
| Trigger key | `?` (`a.Keys.Help`) — context-sensitive: inside a panel shows the panel cheat-sheet; elsewhere unchanged (global help). Closes #36's proposal. No new key. |
| Key source | Single source of truth — the same configured bindings the footer already uses; footer teaser and cheat-sheet never drift |

## Architecture

Three small, isolated units:

### 1. `KeyHint` value + cheat-sheet model (reusable)
```go
// KeyHint is one row of a context cheat-sheet: a display key and what it does.
type KeyHint struct {
	Key  string // already display-formatted, e.g. "Ctrl+R", "Esc", "g"
	Desc string
}
```
A cheat-sheet is `(title string, hints []KeyHint)`. No interface ceremony required for one consumer; a panel simply builds `[]KeyHint` and calls the shared renderer. (If/when several panels adopt it, a `keyHelpProvider` interface can be introduced — deferred, YAGNI.)

### 2. Reusable presentation helper
A method that renders a cheat-sheet as a read-only, themed, focused view and handles Esc to restore the prior view. It reuses the Action Plan's existing **body-swap** mechanism (the `v` prompt viewer already swaps `state.tree` for a read-only `TextView` and restores it on Esc). The renderer:
- Formats `hints` into aligned `key  description` lines under `title`.
- Themes via `a.GetComponentColors("ai")` (the Action Plan's component), matching the panel.
- Is invoked by the panel and restored by the panel's existing Esc handling; no floating overlay, no new focus lifecycle (avoids focus/deadlock risk).

Location: a new focused file `internal/tui/key_help.go` for the `KeyHint` type + a pure formatter `formatKeyHelp(title string, hints []KeyHint) string` (testable without tview). The swap-in/out wiring lives with the panel that owns the view (Action Plan) because the swap uses that panel's container/state.

### 3. Action Plan cheat-sheet source (single source of truth)
The Action Plan has `actionPlanFooterKeys` (currently only `viewPrompt, remember, move, skip`) populated from `a.Keys.*` and feeding `actionPlanFooterText`. That struct does NOT yet carry the action keys (archive/trash/label/toggle-read), `c` (confirm), `g` (assist), `.` (accept) — those are read directly from `a.Keys` elsewhere. To make one honest source of truth:
1. **Extend `actionPlanFooterKeys`** to also hold the remaining advertised bindings (archive, trash, label, toggleRead, confirmPlan, assist, accept), populated in the same place the current 4 are.
2. Add a pure builder taking that (extended) struct:
```go
func actionPlanKeyHints(keys actionPlanFooterKeys) []KeyHint
```
returning the full ordered list (nav, space, action keys, m, i, Ctrl+R, c, g, ., Esc), each key rendered with `prettyKeyLabel`.
Both the footer teaser and the cheat-sheet then read the single extended `actionPlanFooterKeys` value, so rebinding any advertised key updates both. (Navigation/Enter/Tab/Esc are fixed tview keys, listed as literals in the builder.) The footer teaser and this builder read the SAME bindings, so rebinding a key updates both. Wiring: the panel's `SetInputCapture` intercepts `a.Keys.Help` (`?`), builds the hints, calls the renderer to body-swap the tree for the cheat-sheet, and returns nil (so `?` never reaches the global handler while the panel is focused).

## Data flow

1. User is in the Action Plan panel, presses `?`.
2. Panel input capture matches `a.Keys.Help` → `actionPlanKeyHints(...)` → `formatKeyHelp(title, hints)` → body-swap `state.tree` for a read-only cheat-sheet `TextView`, focus it.
3. `Esc` (existing panel Esc handling, extended for this view) swaps the tree back and restores focus/selection.
4. Outside any panel, `?` is untouched: hits `a.Keys.Help` in the global handler → `toggleHelp()` (global help screen).

## Edge cases & error handling

- **Esc precedence:** the cheat-sheet Esc must be handled BEFORE the panel's "Esc closes panel" and before any pending two-press confirm — i.e. Esc closes the cheat-sheet first, returning to the tree; a second Esc then behaves normally. Mirror how the `v` prompt viewer's Esc returns to the tree.
- **Re-entrancy / threading:** the swap is synchronous UI work on the event loop (no `QueueUpdateDraw`, no goroutine, no ErrorHandler) — consistent with the ESC/cleanup rules. Showing the cheat-sheet must not fire while a rename/confirm sub-mode owns the panel; gate on the same state the `v` viewer gates on.
- **Empty/degenerate:** hints list is always non-empty for the Action Plan (static set); the formatter handles an empty slice by rendering just the title (defensive, for future panels).
- **Long lists / small terminal:** the cheat-sheet `TextView` scrolls (tview default) if the panel is short; no truncation of entries.

## Testing

- **Pure unit (`key_help_test.go`):** `formatKeyHelp` — alignment, title line, empty-slice case.
- **Pure unit (Action Plan):** `actionPlanKeyHints` returns the expected ordered entries; **rebinding a key** (e.g. `remember_rule`) is reflected in the output — proving footer + cheat-sheet share one source.
- **TUI component:** pressing `?` in the panel swaps to the cheat-sheet and `Esc` restores the tree (selection preserved); `?` while NOT in the panel still toggles global help (no regression).

## Config / Definition of Done

- **No new key, no config migration** — reuses `a.Keys.Help`.
- Update the in-app `:help` screen to note that `?` inside a panel shows that panel's cheat-sheet.
- Document in `docs/KEYBOARD_SHORTCUTS.md` (Action Plan section + a general note).

## Out of scope

- Wiring labels / prompts / other pickers (base is ready; they adopt later, user-chosen timing).
- `??` / `:help all` global-from-anywhere shortcut (not selected).
- Any change to the global `?` behavior outside panels.
- A `keyHelpProvider` interface / registry (deferred until a 2nd/3rd consumer exists — YAGNI).
