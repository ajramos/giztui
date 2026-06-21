# Search/Filter State Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the 8 search/filter fields from the `App` god object into a `searchState` type (deleting the dead `searchHistory`), with zero user-visible behavior change.

**Architecture:** New `searchState` (in `internal/tui/search_state.go`) holds the search/filter state. `mode`+`query` are guarded by a `sync.RWMutex` (read by the next-page preload goroutines); `localFilter` + the base snapshot are event-loop-only. The base-snapshot logic (capture/restore/remove) moves to pure methods; the App helpers delegate to them and keep their UI effects.

**Tech Stack:** Go, `sync.RWMutex`, tview event loop, `google.golang.org/api/gmail/v1` (`gmailapi`), standard `testing`.

---

## Reference facts (verified in code)

- Eight declared fields (app.go ~109-116): `searchMode string`, `currentQuery string`, `localFilter string`, `searchHistory []string` (**DEAD** — only declared + initialized), `baseIDs []string`, `baseMessagesMeta []*gmailapi.Message`, `baseNextPageToken string`, `baseSelectionID string`. Initialized at app.go ~359-366.
- `searchMode` writes: app.go:3019 (`"remote"`), messages.go:2125 (`"local"`), messages.go:119/224/338 + restore (`""`). Reads: 27 total (== comparisons).
- `currentQuery`/`searchMode` are read by preload goroutines at keys.go:1514-1516 and keys.go:2653-2655 — off the event loop. These are the only cross-goroutine accesses.
- Base snapshot helpers (messages.go): `baseRemoveByID` (44-61), `baseRemoveByIDs` (63-86), `captureLocalBaseSnapshot` (88-105) — reads the table selection, copies `a.ids`/`a.messagesMeta`/`a.nextPageToken` into base fields, `restoreLocalBaseSnapshot` (107+) — copies back inside `QueueUpdateDraw`, resets mode/query/localFilter to `""`, restores the list, re-renders.
- Access counts (non-test): searchMode 27, currentQuery 20, localFilter 4, baseIDs/baseMessagesMeta 13 each, baseNextPageToken/baseSelectionID 2 each, searchHistory 0. Files: app.go, auto_refresh.go, keys.go, messages.go.
- `gmailapi` is `google.golang.org/api/gmail/v1` (already imported in messages.go and app.go).

---

## Task 1: Create `searchState` type + tests

**Files:**
- Create: `internal/tui/search_state.go`
- Test: `internal/tui/search_state_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/search_state_test.go`:

```go
package tui

import (
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func metas(ids ...string) []*gmailapi.Message {
	out := make([]*gmailapi.Message, len(ids))
	for i, id := range ids {
		out[i] = &gmailapi.Message{Id: id}
	}
	return out
}

func TestSearchState_SnapshotCopyIndependence(t *testing.T) {
	var s searchState
	ids := []string{"a", "b", "c"}
	s.captureSnapshot(ids, metas("a", "b", "c"), "tok", "b")

	// Mutating the source slice after capture must not change the stored snapshot.
	ids[0] = "MUT"
	gotIDs, gotMeta, tok, sel := s.snapshot()
	if gotIDs[0] != "a" || tok != "tok" || sel != "b" || len(gotMeta) != 3 {
		t.Fatalf("snapshot not isolated from source: %v tok=%q sel=%q", gotIDs, tok, sel)
	}
	// Mutating the returned slice must not change the stored snapshot either.
	gotIDs[1] = "MUT"
	again, _, _, _ := s.snapshot()
	if again[1] != "b" {
		t.Fatalf("snapshot not isolated from returned copy: %v", again)
	}
}

func TestSearchState_RemoveFromSnapshotByID(t *testing.T) {
	var s searchState
	s.captureSnapshot([]string{"a", "b", "c"}, metas("a", "b", "c"), "", "")
	s.removeFromSnapshotByID("b")
	ids, meta, _, _ := s.snapshot()
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "c" {
		t.Fatalf("ids = %v, want [a c]", ids)
	}
	if len(meta) != 2 || meta[0].Id != "a" || meta[1].Id != "c" {
		t.Fatalf("meta misaligned: %v", meta)
	}
	s.removeFromSnapshotByID("zzz") // missing → no-op
	if ids2, _, _, _ := s.snapshot(); len(ids2) != 2 {
		t.Fatalf("missing id should be a no-op, got %v", ids2)
	}
}

func TestSearchState_RemoveFromSnapshotByIDs(t *testing.T) {
	var s searchState
	s.captureSnapshot([]string{"a", "b", "c", "d"}, metas("a", "b", "c", "d"), "", "")
	s.removeFromSnapshotByIDs([]string{"b", "d"})
	ids, meta, _, _ := s.snapshot()
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "c" {
		t.Fatalf("ids = %v, want [a c]", ids)
	}
	if meta[0].Id != "a" || meta[1].Id != "c" {
		t.Fatalf("meta misaligned: %v", meta)
	}
}

func TestSearchState_Accessors(t *testing.T) {
	var s searchState
	s.SetMode("remote")
	s.SetQuery("is:unread")
	s.localFilter = "foo"
	if s.Mode() != "remote" || s.Query() != "is:unread" || s.localFilter != "foo" {
		t.Fatalf("accessors: mode=%q query=%q filter=%q", s.Mode(), s.Query(), s.localFilter)
	}
	s.clear()
	if s.Mode() != "" || s.Query() != "" || s.localFilter != "" {
		t.Fatalf("clear left state: mode=%q query=%q filter=%q", s.Mode(), s.Query(), s.localFilter)
	}
}
```

- [ ] **Step 2: Run it, verify FAIL (searchState undefined)**

Run: `go test ./internal/tui/ -run TestSearchState -v`
Expected: FAIL — `searchState` undefined.

- [ ] **Step 3: Implement the type**

Create `internal/tui/search_state.go`:

```go
package tui

import (
	"sync"

	gmailapi "google.golang.org/api/gmail/v1"
)

// searchState holds the search/filter state, extracted from the App god object. mode + query are
// read by the next-page preload goroutines, so they are guarded by mu; localFilter and the base
// snapshot are event-loop-only. The base snapshot is a copy of the inbox view taken when a local
// filter starts, so it can be restored on exit. sync.RWMutex makes searchState non-copyable: use it
// as a field accessed via a.search.* with pointer-receiver methods.
type searchState struct {
	mu          sync.RWMutex
	mode        string // "" | "remote" | "local"
	query       string // current query
	localFilter string // event-loop only

	// Local-filter base snapshot (event-loop only).
	baseIDs           []string
	baseMessagesMeta  []*gmailapi.Message
	baseNextPageToken string
	baseSelectionID   string
}

func (s *searchState) Mode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *searchState) SetMode(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = m
}

func (s *searchState) Query() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.query
}

func (s *searchState) SetQuery(q string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.query = q
}

// clear resets mode, query, and localFilter to empty (used when exiting a search/filter).
func (s *searchState) clear() {
	s.mu.Lock()
	s.mode = ""
	s.query = ""
	s.mu.Unlock()
	s.localFilter = ""
}

// captureSnapshot stores an independent copy of the current inbox view as the local-filter base.
func (s *searchState) captureSnapshot(ids []string, meta []*gmailapi.Message, token, selID string) {
	s.baseIDs = append([]string(nil), ids...)
	s.baseMessagesMeta = append([]*gmailapi.Message(nil), meta...)
	s.baseNextPageToken = token
	s.baseSelectionID = selID
}

// snapshot returns independent copies of the base ids/meta plus the token and selection id.
func (s *searchState) snapshot() (ids []string, meta []*gmailapi.Message, token, selID string) {
	ids = append([]string(nil), s.baseIDs...)
	meta = append([]*gmailapi.Message(nil), s.baseMessagesMeta...)
	return ids, meta, s.baseNextPageToken, s.baseSelectionID
}

// removeFromSnapshotByID drops one message (and its aligned meta) from the base snapshot.
func (s *searchState) removeFromSnapshotByID(id string) {
	idx := -1
	for i, x := range s.baseIDs {
		if x == id {
			idx = i
			break
		}
	}
	if idx >= 0 {
		s.baseIDs = append(s.baseIDs[:idx], s.baseIDs[idx+1:]...)
		if idx < len(s.baseMessagesMeta) {
			s.baseMessagesMeta = append(s.baseMessagesMeta[:idx], s.baseMessagesMeta[idx+1:]...)
		}
	}
}

// removeFromSnapshotByIDs drops a set of messages from the base snapshot, preserving order/alignment.
func (s *searchState) removeFromSnapshotByIDs(ids []string) {
	rm := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		rm[id] = struct{}{}
	}
	newIDs := s.baseIDs[:0]
	newMeta := s.baseMessagesMeta[:0]
	for i, id := range s.baseIDs {
		if _, ok := rm[id]; ok {
			continue
		}
		newIDs = append(newIDs, id)
		if i < len(s.baseMessagesMeta) {
			newMeta = append(newMeta, s.baseMessagesMeta[i])
		}
	}
	s.baseIDs = append([]string(nil), newIDs...)
	s.baseMessagesMeta = append([]*gmailapi.Message(nil), newMeta...)
}
```

- [ ] **Step 4: Run the test, verify PASS**

Run: `go test ./internal/tui/ -run TestSearchState -v`
Expected: PASS (all 4 TestSearchState_* tests).

- [ ] **Step 5: Race check**

Run: `go test -race ./internal/tui/ -run TestSearchState`
Expected: PASS, no races.

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/tui/search_state.go internal/tui/search_state_test.go
git add internal/tui/search_state.go internal/tui/search_state_test.go
git commit -m "feat(tui): add searchState type with snapshot logic + tests"
```

(Do NOT add a Co-Authored-By line — the project forbids it.)

---

## Task 2: Rewire `App` and call sites to `searchState`

**Files:**
- Modify: `internal/tui/app.go` (field block ~109-116; init ~359-366), `internal/tui/messages.go`, `internal/tui/keys.go`, `internal/tui/auto_refresh.go`

- [ ] **Step 1: Replace the eight App fields with one**

In `internal/tui/app.go`, replace the block (~109-116):

```go
	// Search/Filter state
	searchMode    string // "" | "remote" | "local"
	currentQuery  string
	localFilter   string
	searchHistory []string
	// Local filter base snapshot (used only while searchMode=="local")
	baseIDs           []string
	baseMessagesMeta  []*gmailapi.Message
	baseNextPageToken string
	baseSelectionID   string
```

with:

```go
	// Search/Filter state (state machine in search_state.go)
	search searchState
```

- [ ] **Step 2: Remove the obsolete initializers (app.go ~360-366)**

Delete these lines from the App struct literal:

```go
		searchMode:         "",
		currentQuery:       "",
		localFilter:        "",
		searchHistory:      make([]string, 0, 10),
		baseIDs:            nil,
		baseMessagesMeta:   nil,
		baseNextPageToken:  "",
		baseSelectionID:    "",
```

(Delete whichever of these keys are present — the `searchState` zero value covers them all. Leave `nextPageToken: "",` which is a different field.)

- [ ] **Step 3: Rewire `searchMode` and `currentQuery` via accessors**

These two cross goroutines, so all access goes through the methods. Apply across all four files:
- A **write** `a.searchMode = X` → `a.search.SetMode(X)`. A **read** `a.searchMode` (e.g. `a.searchMode == "remote"`) → `a.search.Mode()`.
- A **write** `a.currentQuery = X` → `a.search.SetQuery(X)`. A **read** `a.currentQuery` → `a.search.Query()`.

The preload goroutine reads at keys.go:1514-1516 and 2653-2655 become, e.g.:

```go
			query := a.search.Query()
			if query == "" && a.search.Mode() == "remote" {
				query = a.search.Query()
			}
```

- [ ] **Step 4: Rewire `localFilter` (direct field)**

`localFilter` is event-loop-only: `a.localFilter` → `a.search.localFilter` everywhere (read and write), e.g. `a.localFilter = ""` → `a.search.localFilter = ""`.

- [ ] **Step 5: Rewire the base-snapshot helpers to delegate**

In `internal/tui/messages.go`:

`baseRemoveByID` (44-61) becomes:

```go
func (a *App) baseRemoveByID(messageID string) {
	if a.search.Mode() != "local" || a.search.baseIDs == nil {
		return
	}
	a.search.removeFromSnapshotByID(messageID)
}
```

`baseRemoveByIDs` (63-86) becomes:

```go
func (a *App) baseRemoveByIDs(ids []string) {
	if a.search.Mode() != "local" || a.search.baseIDs == nil || len(ids) == 0 {
		return
	}
	a.search.removeFromSnapshotByIDs(ids)
}
```

`captureLocalBaseSnapshot` (88-105) — keep the table-selection read, then delegate the copy:

```go
func (a *App) captureLocalBaseSnapshot() {
	var selID string
	if table, ok := a.views["list"].(*tview.Table); ok {
		row, _ := table.GetSelection()
		messageIndex := row - 1
		if messageIndex >= 0 && messageIndex < len(a.ids) {
			selID = a.ids[messageIndex]
		}
	}
	a.search.captureSnapshot(a.ids, a.messagesMeta, a.nextPageToken, selID)
}
```

`restoreLocalBaseSnapshot` (107+) — replace the snapshot read + the three resets with the type's
methods; keep everything else (the `QueueUpdateDraw` body, list apply, re-render). Concretely, at the
top of the function:

```go
	ids, metas, next, selID := a.search.snapshot()
```
(replacing the four `append(...)` reads of `a.baseIDs`/`a.baseMessagesMeta` + `a.baseNextPageToken` + `a.baseSelectionID`), and inside the `QueueUpdateDraw` callback replace:
```go
		a.searchMode = ""
		a.currentQuery = ""
		a.localFilter = ""
```
with:
```go
		a.search.clear()
```
The rest of `restoreLocalBaseSnapshot` (setting `a.nextPageToken = next`, `a.SetMessageIDs(ids)`, `a.messagesMeta = metas`, the table re-render using `selID`) is unchanged.

- [ ] **Step 6: Build and check for stragglers**

Run: `go build ./...`
Expected: success. Fix any compile error by rewiring the named field.

Run: `grep -nE 'a\.(searchMode|currentQuery|localFilter|searchHistory|baseIDs|baseMessagesMeta|baseNextPageToken|baseSelectionID)\b' internal/tui/*.go | grep -v _test`
Expected: no output. (`a.search.baseIDs` etc. inside the helpers are fine — those are `a.search.base*`, which the regex `a\.baseIDs` does not match.)

- [ ] **Step 7: Tests + race**

Run: `go test ./internal/tui/ 2>&1 | tail -3` → `ok`.
Run: `go test -race ./internal/tui/ -run TestSearchState 2>&1 | tail -3` → PASS.
Run `gofmt -w internal/tui/app.go internal/tui/messages.go internal/tui/keys.go internal/tui/auto_refresh.go`.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/app.go internal/tui/messages.go internal/tui/keys.go internal/tui/auto_refresh.go
git commit -m "refactor(tui): route search/filter state through searchState (removes 8 App fields, drops dead searchHistory)"
```

(Do NOT add a Co-Authored-By line.)

---

## Task 3: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Canonical pre-commit check**

Run: `make pre-commit-check`
Expected: `All pre-commit checks passed!`

- [ ] **Step 2: Full test suite**

Run: `make test 2>&1 | grep -E "^(ok|FAIL)" | grep -v "no test files"`
Expected: all `ok`, no `FAIL`.

- [ ] **Step 3: Race detector on the whole TUI package**

Run: `go test -race ./internal/tui/ 2>&1 | tail -3`
Expected: `ok`, no race warnings.

- [ ] **Step 4: Build the binary**

Run: `make build`
Expected: `Built build/giztui ...`.

- [ ] **Step 5: Finish the branch**

Use the superpowers:finishing-a-development-branch skill. Manual search/filter smoke test on the
user's Mac before merge: `/` remote search returns results and paginates; enter a local filter, type
to narrow, Esc restores the full list + the prior selection; delete a message while filtering and the
restored list is consistent; next-page preload still works mid-search. Do NOT push/merge without
explicit user confirmation (project rule: "commit" ≠ "publish"). Do NOT add a Co-Authored-By line.

---

## Self-Review

**Spec coverage:**
- `searchState` type + accessors + snapshot methods → Task 1. ✓
- App composes `search searchState`; 8 fields + initializers removed → Task 2 Steps 1-2. ✓
- `searchMode`/`currentQuery` via accessors (preload goroutine reads through them) → Task 2 Step 3. ✓
- `localFilter` + base snapshot rewired; the four helpers delegate → Task 2 Steps 4-5. ✓
- `searchHistory` deleted (field + init) → Task 2 Steps 1-2. ✓
- `search_state_test.go` covering snapshot + accessors → Task 1. ✓
- `make pre-commit-check` + `-race` green → Task 3. ✓
- No user-visible behavior change → each field access mapped 1:1; manual smoke test Task 3 Step 5. ✓

**Type/signature consistency:** `searchState` fields (`mode`/`query`/`localFilter`/`baseIDs`/`baseMessagesMeta`/`baseNextPageToken`/`baseSelectionID`) and methods (`Mode()`,`SetMode(string)`,`Query()`,`SetQuery(string)`,`clear()`,`captureSnapshot(ids,meta,token,selID)`,`snapshot() (ids,meta,token,selID)`,`removeFromSnapshotByID(string)`,`removeFromSnapshotByIDs([]string)`) defined in Task 1 and used identically in Task 2. ✓

**Placeholder scan:** no TBD/TODO; every code step shows before/after. The Step-2 "delete whichever keys are present" and Step-6 straggler grep are bounded clean-up backed by the grep gate, not placeholders. ✓
