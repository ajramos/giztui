# Command State Extraction — Design (App god-object refactor, step 2)

**Date:** 2026-06-21
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `refactor/command-state`
**Issue:** #49 (graphify — `App` god object)

## Goal

Extract the command-bar state (the `:`-command prompt) out of the `App` god object into a
self-contained `commandState` type, with the command-history logic unit-tested and **zero change to
user-visible behavior**. This is the second step of the incremental App decomposition; the first
(VIM state → `vimState`) shipped in v1.17.0.

## Why this subsystem

After the VIM pilot, `App` has 125 fields. The command-bar state is the best next slice:
- **Cohesive:** 6 fields that together model the `:`-command prompt (mode, buffer, suggestion,
  focus-override, history, history cursor).
- **Has testable logic:** the command history (add with skip-empty/skip-consecutive-dup + cap-100,
  and Up/Down navigation with clamps) — today untested.
- **No threading complexity:** unlike VIM (which had a timeout goroutine touching its fields), the
  command fields are accessed only from the tview event loop. No mutex needed.
- 76 accesses across 4 files (`commands.go`, `keys.go`, `messages.go`, `action_plan_rules.go`); does
  not touch the central `app.go` logic beyond the field declarations + one initializer line.

## The 6 fields being extracted (today on `App`, app.go ~93-98)

```go
cmdMode          bool     // whether the command bar is open
cmdBuffer        string   // current command text
cmdHistory       []string // executed-command history
cmdHistoryIndex  int      // cursor into history (== len(history) means "new line")
cmdSuggestion    string   // current Tab/auto suggestion
cmdFocusOverride string   // overrides focus restoration after a special command
```

## Current behavior (verified)

- `addToHistory(cmd)` (commands.go:942): skips empty and consecutive-duplicate commands, appends,
  caps history at 100 (drops oldest), and resets the cursor to the end.
- Opening the bar sets `cmdHistoryIndex = len(cmdHistory)` (commands.go:68).
- `Up` (commands.go:97): if cursor > 0, decrement and show `cmdHistory[cursor]`.
- `Down` (commands.go:105): if cursor < len-1, increment and show `cmdHistory[cursor]`; otherwise set
  cursor to len and clear the input.
- `cmdBuffer` is kept in sync via the input's ChangedFunc; `cmdSuggestion` feeds the live hint.
- `cmdMode` gates several key handlers (keys.go:560/568, messages.go:500, commands.go:1022).
- `cmdFocusOverride` (set in commands.go:1015 + action_plan_rules.go:221) is read once and cleared in
  keys.go:1694-1696 to redirect focus after closing.
- All accesses are on the event-loop goroutine. No goroutine touches these fields.

## Architecture

New file `internal/tui/command_state.go` defining a package-private `commandState`:

```go
type commandState struct {
    mode          bool
    buffer        string
    suggestion    string
    focusOverride string
    history       []string
    historyIndex  int
}
```

`App` composes one field `cmd commandState` in place of the six flat fields. Zero value is ready to
use, so the `cmdMode: false` initializer line in app.go is removed (no constructor change needed).

**Separation of concerns:** `commandState` owns the **pure history state machine**; `keys.go` /
`commands.go` keep the UI effects (`input.SetText`, theming, focus) and call the methods. No mutex
(single-goroutine access, documented on the type).

### Method surface (pure history logic; decisions in, the input SetText stays in the handler)

```go
func (c *commandState) addToHistory(cmd string)              // skip empty/consecutive-dup, cap 100, reset cursor
func (c *commandState) resetHistoryCursor()                  // historyIndex = len(history); called when the bar opens
func (c *commandState) historyUp() (text string, ok bool)    // older entry; ok=false when already at the top
func (c *commandState) historyDown() (text string, ok bool)  // newer entry, or "" past the end; ok=true means "set input to text"
```

The four simple fields (`mode`, `buffer`, `suggestion`, `focusOverride`) carry no logic; handlers
access them directly as `a.cmd.mode`, `a.cmd.buffer`, etc. (safe — event-loop only).

`historyUp`/`historyDown` reproduce the exact clamp behavior: Up at the top is a no-op (`ok=false`,
handler does not touch the input); Down past the end resets the cursor and returns `("", true)` so the
handler clears the input.

## Behavior preservation (non-negotiable)

The command bar must behave identically: `:` opens it, typing updates buffer + live hint, Tab
completes, Up/Down walk history (with the same clamps and the clear-on-past-end), Enter executes and
records to history (skipping empties/consecutive dups, capped at 100), Esc closes, and the
focus-override redirect after special commands still fires once and clears.

## Testing

New `internal/tui/command_state_test.go` covering the history state machine (today: 0 coverage):
- `addToHistory`: appends; skips empty; skips a consecutive duplicate; caps at 100 (oldest dropped);
  resets the cursor to the end each time.
- `resetHistoryCursor`: sets the cursor to `len(history)`.
- `historyUp`: walks from newest to oldest; returns `ok=false` at the top (no further movement).
- `historyDown`: walks back toward newest; past the end returns `("", true)` and the cursor lands at
  `len(history)`.
- A combined up-then-down sequence returns to the empty new-line state.

Existing behavior verified by `make test` (TUI suite) + a manual command-bar smoke test.

## Out of scope (YAGNI)

- Changing any command, suggestion logic, or keybinding.
- Persisting command history across runs.
- Other App subsystems (search/filter, AI summary pane) — future steps, separate specs.

## Definition of Done

- [ ] `commandState` type + history methods in `internal/tui/command_state.go`.
- [ ] `App` composes `cmd commandState`; the six flat fields + the `cmdMode:false` initializer removed.
- [ ] `commands.go` / `keys.go` / `messages.go` / `action_plan_rules.go` rewired to `a.cmd.*`;
      `addToHistory` becomes a thin wrapper or is replaced by `a.cmd.addToHistory`.
- [ ] `command_state_test.go` covering the history state machine.
- [ ] No stray `a.cmdMode` / `a.cmdBuffer` / `a.cmdHistory` / `a.cmdHistoryIndex` / `a.cmdSuggestion`
      / `a.cmdFocusOverride` references remain.
- [ ] `make pre-commit-check` green; `go test -race ./internal/tui/...` green.
- [ ] No user-visible behavior change (manual command-bar smoke test).
