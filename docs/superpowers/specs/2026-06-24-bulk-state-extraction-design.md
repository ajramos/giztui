# Bulk Selection State Extraction — Design

**Date:** 2026-06-24
**Status:** Approved (refactor pattern) — pending implementation plan
**Branch:** `refactor/bulk-state`
**Issue:** #49 (App god object) — final large slice; refactor scorecard candidate "Bulk selection".

## Goal

Extract the two bulk-selection fields (`bulkMode`, `selected`) from the `App` god object into a
self-contained, mutex-guarded `bulkState` type, **fixing a latent concurrent-map race** in the
process. This is the last large field group on `App` (now 93 fields, down from 129).

## Why this slice / the bug

- `selected map[string]bool` (the set of bulk-selected message IDs) and `bulkMode bool` have **no
  synchronization**. `selected` is written on the event loop (Space toggles selection in `keys.go`)
  and **ranged/read inside operation goroutines** (`bulk_prompts.go:233` `for id := range a.selected`,
  plus obsidian/slack/messages_bulk bulk ops, and `auto_refresh.go`). Concurrent map read/write in Go
  can **panic** — the same class of latent bug fixed for the caches in v1.18.0.
- ~280 access sites across 12 files: `keys.go`, `labels.go`, `columns.go`, `messages.go`,
  `messages_bulk.go`, `bulk_prompts.go`, `obsidian.go`, `slack.go`, `action_plan.go`, `commands.go`,
  `auto_refresh.go`, `app.go`. Measured shapes: bulkMode read 66 / write 29; selected
  count(`len`) 89, range 19, index-read 20, index-write 7, delete 1, whole-map assign 35, nil-check 13.

After this, the picker state is already encapsulated (ActivePicker enum + `isXxxPickerActive`
accessors), so the god-object field-grouping refactor is effectively complete.

## Architecture

New file `internal/tui/bulk_state.go`:

```go
type bulkState struct {
	mu       sync.RWMutex
	mode     bool
	selected map[string]bool // set of selected message IDs (value always true)
}

func newBulkState() *bulkState { return &bulkState{selected: map[string]bool{}} }

func (b *bulkState) isMode() bool            // RLock
func (b *bulkState) setMode(v bool)          // Lock
func (b *bulkState) isSelected(id string) bool // RLock; nil-safe
func (b *bulkState) add(id string)           // Lock; selected[id]=true
func (b *bulkState) remove(id string)        // Lock; delete(selected,id)
func (b *bulkState) toggle(id string) bool   // Lock; flip membership, return new state
func (b *bulkState) count() int              // RLock; len(selected)
func (b *bulkState) ids() []string           // RLock; copy of the selected IDs (for ranging)
func (b *bulkState) clear()                  // Lock; replace with a fresh empty map
```

`App` composes `bulk *bulkState` (pointer, since the mutex makes it non-copyable — same pattern as
`caches *appCaches`), initialized with `newBulkState()` in `NewApp`, replacing the `selected` and
`bulkMode` fields.

## Rewire mapping (mechanical)

| Old | New |
|-----|-----|
| `a.bulkMode` (read) | `a.bulk.isMode()` |
| `a.bulkMode = v` | `a.bulk.setMode(v)` |
| `a.selected = make(map[string]bool)` | `a.bulk.clear()` |
| `len(a.selected)` | `a.bulk.count()` |
| `for id := range a.selected` | `for _, id := range a.bulk.ids()` |
| `a.selected[id]` (read/guard) | `a.bulk.isSelected(id)` |
| `a.selected != nil && a.selected[id]` | `a.bulk.isSelected(id)` |
| `a.selected[id] = true` | `a.bulk.add(id)` |
| `delete(a.selected, id)` | `a.bulk.remove(id)` |
| `a.selected != nil` (guard alone) | `a.bulk.count() > 0` (or drop where redundant) |
| toggle pattern `if a.selected[id] {delete} else {a.selected[id]=true}` | `a.bulk.toggle(id)` |

The constructor's `bulkMode: false` and any `selected:` initializer are removed (replaced by
`bulk: newBulkState()`).

## Behavior preservation (non-negotiable)

Identical behavior: entering/leaving bulk mode, Space to toggle one, select-all/range, the selection
count in the status bar, and every bulk action (archive/trash/read/move/label/prompt/slack/obsidian)
operate on the same set as before. The only difference is that the set is now accessed under a lock,
so a selection edit concurrent with a bulk op can no longer corrupt the map. `ids()` returns an
independent copy, so iterating it while the user toggles selection is safe (the op sees a snapshot).

## Edge cases

- **Compound select-all loops** (`for i ... { a.selected[a.ids[i]] = true }`) become a loop of
  `a.bulk.add(...)`. Each add takes the lock; fine (not hot).
- **toggle** must be a single method (atomic read-modify-write) to avoid a check-then-act race.
- **`ids()` snapshot semantics**: bulk ops that range and then act per-id already tolerate the set
  changing (they captured ids); returning a copy makes this explicit and safe.

## Testing

New `internal/tui/bulk_state_test.go`:
- mode round-trip; add/isSelected/remove; toggle returns new state and flips membership; count;
  `ids()` returns all added and is an independent copy (mutating the result doesn't affect state);
  clear empties it; isSelected on a fresh state is false (nil-safe).
- A `-race` test that hammers add/remove/ids/count from multiple goroutines passes the race detector.

Existing behavior verified by `make test` + `go test -race ./internal/tui/` + a live harness smoke
test (enter bulk mode, select a couple, run a bulk action, confirm it acts on them; no panic).

## Out of scope (YAGNI)

- Moving bulk *operations* into the service layer (a separate, larger design question).
- Picker state (already encapsulated).
- Any change to selection UX.

## Definition of Done

- [ ] `bulkState` type + methods + `newBulkState` in `internal/tui/bulk_state.go`.
- [ ] `App` composes `bulk *bulkState`; `bulkMode`/`selected` fields removed; constructor updated.
- [ ] All ~280 sites rewired per the mapping; no `a.bulkMode`/`a.selected` references remain.
- [ ] `bulk_state_test.go` (unit + race) green.
- [ ] `make pre-commit-check` + `go test -race ./internal/tui/` + full suite green.
- [ ] Harness smoke test: bulk select + a bulk action, no panic, acts on the selection.
- [ ] No user-visible behavior change.
