# Focus State Extraction — Design

**Date:** 2026-06-28
**Status:** Approved (refactor pattern) — pending plan
**Branch:** `refactor/focus-state`
**Issue:** #53 / #49 (App god object).

## Goal

Extract `currentFocus`/`currentView` from `App` into a small mutex-guarded `focusState`, AND remove the
dominant footgun: the focus name is duplicated at ~95 sites (`a.currentFocus = "X"` immediately
followed by `a.updateFocusIndicators("X")`). A single `markFocus(name)` helper collapses each pair,
killing the duplication. Also makes the field thread-safe (it is touched from goroutines today).

## Current state (measured)

- `currentFocus string` — 173 accesses / 25 files: 122 writes, 40 `==`/`!=` reads, 11 other reads.
  Touched inside goroutines (attachments, bulk_prompts, ai, accounts…) with no synchronization.
- **95 of the writes are the adjacent literal pair** `a.currentFocus = "X"` then
  `a.updateFocusIndicators("X")` (same literal). `updateFocusIndicators(name)` takes the name as a
  param and repaints borders — it does NOT read `currentFocus`, so the pair is purely "record focus +
  repaint".
- `a.SetFocus(view)` (tview) usually precedes the pair but with a varying view and not always adjacent
  — it is LEFT UNTOUCHED (no change to which primitive holds tview focus).
- `currentView string` — 5 uses: `GetCurrentView`/`SetCurrentView` accessors + 3 raw in threads.go.
- Existing accessors: `GetCurrentFocus()`, `GetCurrentView()`, `SetCurrentView()` (use `a.mu`).
  There is no `SetCurrentFocus` — writes are raw.

## Why this is worth it (unlike #52)

#52 was positional surgery on aligned slices (bad fit). #53 is a single string whose value is
*duplicated 95 times*. The win is de-duplication (a real drift footgun) + a clean intent helper, not
just field-grouping. "173 accesses" is misleading — ~95 collapse into one helper call.

## Architecture

New `internal/tui/focus_state.go`:

```go
type focusState struct {
	mu      sync.RWMutex
	current string // "list" | "text" | "labels" | "summary" | "slack" | "prompts" | ...
	view    string // "messages" | "thread" | "flat"
}

func (f *focusState) cur() string          // RLock
func (f *focusState) set(name string)       // Lock
func (f *focusState) is(name string) bool   // RLock; cur()==name
func (f *focusState) viewName() string      // RLock
func (f *focusState) setView(v string)      // Lock
```

`App` composes `focus focusState` (value field; pointer-receiver methods, never copied — like
`searchState`) in place of `currentFocus`/`currentView`.

App helper (the de-dup):
```go
// markFocus records which named view is focused and repaints the focus borders. Replaces the
// duplicated `a.currentFocus = name; a.updateFocusIndicators(name)` pair.
func (a *App) markFocus(name string) {
	a.focus.set(name)
	a.updateFocusIndicators(name)
}
```

## Rewire mapping

| Old | New |
|-----|-----|
| `a.currentFocus = "X"` + next line `a.updateFocusIndicators("X")` (95 pairs, + the 2 variable pairs) | `a.markFocus("X")` |
| `a.currentFocus = X` (standalone, no adjacent updateFocusIndicators) | `a.focus.set(X)` |
| `a.currentFocus == X` / `!= X` | `a.focus.is(X)` / `!a.focus.is(X)` |
| `a.currentFocus` (other read) | `a.focus.cur()` |
| `GetCurrentFocus()` body | `return a.focus.cur()` |
| `a.currentView` read / `== X` | `a.focus.viewName()` (or in the `GetCurrentView` body) |
| `a.currentView = X` | `a.focus.setView(X)` |
| `GetCurrentView`/`SetCurrentView` bodies | delegate to `focus.viewName()`/`setView()` |

The 2 variable-arg pairs (`a.currentFocus = currentFocusState; a.updateFocusIndicators(currentFocusState)`
at app.go ~398, and `target.name` at keys.go ~1664) collapse to `a.markFocus(<var>)`.

## Behavior preservation (non-negotiable)

Identical focus behavior: every `markFocus(name)` does exactly what the original two lines did, in the
same order (record name, then repaint borders). `a.SetFocus(view)` calls are untouched, so tview's
real focus is unchanged. Border repaint is identical (`updateFocusIndicators` unchanged).

## Testing

`focus_state_test.go`: cur/set/is round-trip; view round-trip; default empty; a `-race` test hammering
set/cur/is from goroutines. Existing behavior: `make test` + `-race` + harness smoke (open/close
labels picker, AI summary, slack, command bar; focus indicator + `currentFocus`-gated logic still
behave; no panic).

## Out of scope (YAGNI)

- A general focus-ring/transition API beyond `markFocus` (the focus ring already exists separately).
- Touching `a.SetFocus(view)` call sites.

## Definition of Done

- [ ] `focusState` type + `markFocus` helper + tests.
- [ ] `App` composes `focus focusState`; `currentFocus`/`currentView` removed; accessors delegate.
- [ ] 95 literal pairs (+2 variable) collapsed to `markFocus`; remaining sites routed through focus.
- [ ] No raw `a.currentFocus`/`a.currentView` remain.
- [ ] pre-commit + `-race` + full suite green; harness focus smoke; no user-visible change.
