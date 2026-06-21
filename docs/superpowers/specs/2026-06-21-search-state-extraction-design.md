# Search/Filter State Extraction — Design (App god-object refactor, step 3)

**Date:** 2026-06-21
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `refactor/search-state`
**Issue:** #49 (graphify — `App` god object)

## Goal

Extract the search/filter state out of the `App` god object into a `searchState` type, **delete the
dead `searchHistory` field**, and keep zero user-visible behavior change. Third step of the
incremental App decomposition (after `vimState` v1.17.0 and `commandState`).

## Why this subsystem & what's different

The search/filter state is the next cohesive slice (8 declared fields). Two honest caveats found
during design:
- **`searchHistory` is dead** — declared (app.go:112) and initialized (app.go:362) but read/written
  nowhere. It is deleted, not extracted.
- **Less pure logic than VIM/commands.** There is no real history state machine; the value is
  grouping the state + encapsulating the local-filter base snapshot (save/restore/maintain) + the
  dead-field cleanup.

More entangled than the prior two steps:
- **Threading:** the next-page preload goroutines read `currentQuery` and `searchMode`
  (keys.go:1514-1516, 2653-2655). Those two cross a goroutine boundary, so they are guarded by a
  `sync.RWMutex`. The rest (localFilter + the base snapshot) is event-loop-only.
- **Coupling to the list model:** the base snapshot copies `a.ids` / `a.messagesMeta` /
  `a.nextPageToken`. The type stores its own copies; the App helpers pass data in and apply data out,
  keeping `searchState` independent of `App`'s list fields.

## The 8 declared fields (app.go ~108-116)

```go
searchMode    string // "" | "remote" | "local"   → searchState.mode (mutex)
currentQuery  string                              → searchState.query (mutex)
localFilter   string                              → searchState.localFilter
searchHistory []string                            → DELETED (dead)
baseIDs           []string                        → searchState.baseIDs
baseMessagesMeta  []*gmailapi.Message             → searchState.baseMessagesMeta
baseNextPageToken string                          → searchState.baseNextPageToken
baseSelectionID   string                          → searchState.baseSelectionID
```

Usage counts (non-test): searchMode 27, currentQuery 20, localFilter 4, baseIDs/baseMessagesMeta 13
each, baseNextPageToken/baseSelectionID 2 each, searchHistory 0.

## Current behavior (verified)

- `searchMode` transitions: `"remote"` (app.go:3019, on a remote search), `"local"` (messages.go:2125,
  entering local filter), `""` (messages.go:119/224/338 + on restore).
- Base snapshot lifecycle (messages.go): `captureLocalBaseSnapshot` (88-105) records the current
  selection id from the table, then copies `a.ids`/`a.messagesMeta`/`a.nextPageToken` into the base
  fields. `restoreLocalBaseSnapshot` (107+) copies them back (inside `QueueUpdateDraw`), resets
  mode/query/localFilter to `""`, restores `nextPageToken`/ids/meta, and re-renders the list.
  `baseRemoveByID` (47-61) and `baseRemoveByIDs` (63-86) keep the snapshot consistent when messages
  are deleted while a local filter is active.
- `currentQuery`/`searchMode` are read by the preload goroutines to rebuild the paginated query.

## Architecture

New file `internal/tui/search_state.go`:

```go
type searchState struct {
	mu          sync.RWMutex        // guards mode + query (read by preload goroutines)
	mode        string              // "" | "remote" | "local"
	query       string              // current query
	localFilter string              // event-loop only
	// Local-filter base snapshot (event-loop only; saved/restored around a local filter).
	baseIDs           []string
	baseMessagesMeta  []*gmailapi.Message
	baseNextPageToken string
	baseSelectionID   string
}
```

`App` composes `search searchState` in place of the 8 flat fields (the `searchHistory` initializer
goes away too).

**Synchronized accessors (mode + query only — they cross goroutines):**

```go
func (s *searchState) Mode() string        // RLock
func (s *searchState) SetMode(m string)    // Lock
func (s *searchState) Query() string       // RLock
func (s *searchState) SetQuery(q string)   // Lock
```

**Snapshot methods (event-loop only, no lock):**

```go
func (s *searchState) captureSnapshot(ids []string, meta []*gmailapi.Message, token, selID string)
func (s *searchState) snapshot() (ids []string, meta []*gmailapi.Message, token, selID string) // returns copies
func (s *searchState) removeFromSnapshotByID(id string)
func (s *searchState) removeFromSnapshotByIDs(ids []string)
func (s *searchState) clear() // mode="", query="", localFilter="" (under mu for mode/query)
```

`localFilter` is event-loop-only; it is accessed directly as `a.search.localFilter`. The App helpers
`captureLocalBaseSnapshot` / `restoreLocalBaseSnapshot` / `baseRemoveByID` / `baseRemoveByIDs` keep
their names and UI effects (table read for selID, `QueueUpdateDraw`, re-render) but delegate the
state work to the `searchState` methods (e.g. `restoreLocalBaseSnapshot` calls `a.search.snapshot()`
then `a.search.clear()` then applies ids/meta to the list and re-renders).

## Behavior preservation (non-negotiable)

Identical behavior: `/` remote search, local filter (enter → snapshot taken; type → filters; Esc →
snapshot restored, list + selection back), delete-while-filtering keeps the snapshot consistent,
next-page preload uses the current query/mode, and `searchMode` drives the same gates everywhere.

## Testing

New `internal/tui/search_state_test.go` (pure methods; snapshot is the testable core):
- `captureSnapshot` then `snapshot()` returns equal-but-independent copies (mutating the returned
  slices does not change the stored snapshot, and vice-versa — verifies the copy-on-capture/return).
- `removeFromSnapshotByID` drops the id + its aligned meta; a missing id is a no-op.
- `removeFromSnapshotByIDs` drops a set, keeping order and id↔meta alignment.
- `clear` resets mode/query/localFilter to `""`.
- `SetMode`/`Mode`, `SetQuery`/`Query` round-trip.
- `go test -race` exercises concurrent `Query()`/`SetQuery()` (a tiny goroutine loop) to prove the
  mutex closes the preload race.

Existing behavior verified by `make test` (TUI suite) + a manual search/filter smoke test.

## Out of scope (YAGNI)

- Adding a real search history (the dead field is removed, not revived).
- The list model itself (`ids`, `messagesMeta`, `nextPageToken`) stays on `App`; only the base
  *snapshot* of it moves.
- Other App subsystems (AI summary pane, bulk selection) — future steps.

## Definition of Done

- [ ] `searchState` type + accessors + snapshot methods in `internal/tui/search_state.go`.
- [ ] `App` composes `search searchState`; the 8 flat fields + `searchHistory` initializer removed.
- [ ] `searchMode`/`currentQuery` rewired to `a.search.Mode()/SetMode()` and `Query()/SetQuery()`
      (preload goroutine reads go through the accessors); `localFilter` + base snapshot rewired to
      `a.search.*`; the four base-snapshot App helpers delegate to the type.
- [ ] `searchHistory` deleted (field + initializer).
- [ ] `search_state_test.go` covering the snapshot + accessors.
- [ ] No stray `a.searchMode` / `a.currentQuery` / `a.localFilter` / `a.searchHistory` / `a.base*`
      references remain.
- [ ] `make pre-commit-check` green; `go test -race ./internal/tui/...` green.
- [ ] No user-visible behavior change (manual search/filter smoke test).
