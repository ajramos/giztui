# Bulk State Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Extract `bulkMode`/`selected` from the `App` god object into a mutex-guarded `bulkState`, fixing the latent concurrent-map race on `selected`. No user-visible change.

**Architecture:** New `bulkState{mu, mode, selected}` (`internal/tui/bulk_state.go`) with leaf-lock accessor methods. `App` composes `bulk *bulkState`. ~280 sites rewired per a fixed mapping; ~20 compound sites fixed by hand first, the rest by ordered sed.

**Tech Stack:** Go, standard `testing` + `sync`.

---

## Task 1: bulkState type + tests

**Files:** Create `internal/tui/bulk_state.go`, `internal/tui/bulk_state_test.go`.

- [ ] **Step 1: test (write first)** — `internal/tui/bulk_state_test.go`:

```go
package tui

import (
	"sync"
	"testing"
)

func TestBulkState_Basics(t *testing.T) {
	b := newBulkState()
	if b.isMode() || b.count() != 0 || b.isSelected("x") {
		t.Fatal("fresh bulkState must be empty/off")
	}
	b.setMode(true)
	if !b.isMode() {
		t.Fatal("setMode(true)")
	}
	b.add("a")
	b.add("b")
	if b.count() != 2 || !b.isSelected("a") {
		t.Fatalf("after add: count=%d", b.count())
	}
	if got := b.toggle("a"); got || b.isSelected("a") {
		t.Fatal("toggle('a') should remove it and return false")
	}
	if got := b.toggle("c"); !got || !b.isSelected("c") {
		t.Fatal("toggle('c') should add it and return true")
	}
	b.remove("b")
	if b.isSelected("b") {
		t.Fatal("remove('b')")
	}
	ids := b.ids()
	if len(ids) != 1 || ids[0] != "c" {
		t.Fatalf("ids() = %v, want [c]", ids)
	}
	// ids() is an independent copy.
	ids[0] = "mutated"
	if !b.isSelected("c") {
		t.Fatal("mutating ids() result must not affect state")
	}
	b.clear()
	if b.count() != 0 {
		t.Fatal("clear()")
	}
}

func TestBulkState_Race(t *testing.T) {
	b := newBulkState()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := string(rune('a' + n))
			for j := 0; j < 1000; j++ {
				b.add(id)
				_ = b.count()
				_ = b.ids()
				_ = b.isSelected(id)
				b.remove(id)
			}
		}(i)
	}
	wg.Wait()
}
```

- [ ] **Step 2: run, verify FAIL** — `go test ./internal/tui/ -run TestBulkState -v` → FAIL (undefined).

- [ ] **Step 3: implement** — `internal/tui/bulk_state.go`:

```go
package tui

import "sync"

// bulkState holds bulk-selection state extracted from the App god object. `selected` is written on
// the event loop (Space toggles) and ranged from bulk-operation goroutines, so it is mutex-guarded —
// previously an unsynchronized map, a latent concurrent-map race. Leaf lock: methods never call back
// into App, so there is no lock-ordering risk with a.mu.
type bulkState struct {
	mu       sync.RWMutex
	mode     bool
	selected map[string]bool
}

func newBulkState() *bulkState { return &bulkState{selected: map[string]bool{}} }

func (b *bulkState) isMode() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.mode
}

func (b *bulkState) setMode(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mode = v
}

func (b *bulkState) isSelected(id string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.selected[id]
}

func (b *bulkState) add(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.selected[id] = true
}

func (b *bulkState) remove(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.selected, id)
}

// toggle flips membership and returns the new state (true = now selected).
func (b *bulkState) toggle(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.selected[id] {
		delete(b.selected, id)
		return false
	}
	b.selected[id] = true
	return true
}

func (b *bulkState) count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.selected)
}

// ids returns an independent copy of the selected IDs (safe to range while selection changes).
func (b *bulkState) ids() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.selected))
	for id := range b.selected {
		out = append(out, id)
	}
	return out
}

// clear empties the selection.
func (b *bulkState) clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.selected = map[string]bool{}
}
```

- [ ] **Step 4: run, verify PASS** — `go test ./internal/tui/ -run TestBulkState -v` and `go test -race ./internal/tui/ -run TestBulkState_Race` → PASS.

- [ ] **Step 5: commit** — `gofmt -w` both; `git commit -m "feat(tui): add bulkState type (mutex-guarded) with tests"`. NO Co-Authored-By.

---

## Task 2: App field swap + fix the compound sites by hand

**Files:** `internal/tui/app.go`, `internal/tui/keys.go`, `internal/tui/action_plan.go`, `internal/tui/columns.go`.

- [ ] **Step 1: struct + constructor (app.go)** — replace the two fields:
```go
	selected map[string]bool // messageID -> selected
	bulkMode bool
```
with:
```go
	bulk *bulkState // bulk-selection state (mode + selected set), see bulk_state.go
```
In `NewApp`, remove the `bulkMode: false,` initializer (and any `selected:` initializer) and add `bulk: newBulkState(),` in the struct literal.

- [ ] **Step 2: toggle site (keys.go ~1573)** — replace:
```go
			if a.selected[mid] {
				delete(a.selected, mid)
			} else {
				a.selected[mid] = true
			}
```
with `a.bulk.toggle(mid)`.

- [ ] **Step 3: nil-init guards (keys.go — 6 sites: ~393, 784, 817, 1552, 2156, 2422, 2463)** — each looks like:
```go
				if a.selected == nil {
					a.selected = make(map[string]bool)
				}
				a.selected[a.ids[messageIndex]] = true
```
Replace the WHOLE guard+assign with a single `a.bulk.add(a.ids[messageIndex])` (the index expression varies — keep it). Where the `== nil` guard precedes a different op (e.g. a range or a `len`), just DELETE the 3-line guard (bulkState's map is always initialized). After this step, grep `grep -n 'a\.selected == nil' internal/tui/keys.go` must be empty.

- [ ] **Step 4: value-range snapshot (action_plan.go ~203)** — replace:
```go
	selected := make(map[string]bool, len(a.selected))
	for id, ok := range a.selected {
		if ok {
			selected[id] = true
		}
	}
```
with:
```go
	selected := make(map[string]bool)
	for _, id := range a.bulk.ids() {
		selected[id] = true
	}
```

- [ ] **Step 5: nil-check-index guards (columns.go ~839, 854, 874)** — replace `a.selected != nil && a.selected[a.ids[i]]` with `a.bulk.isSelected(a.ids[i])` (and `a.bulkMode && a.bulk.isSelected(...)` becomes `a.bulk.isMode() && a.bulk.isSelected(...)` after Task 3's sed; for now just fix the `selected != nil && selected[...]` part).

- [ ] **Step 6: build** — `go build ./...` will still FAIL (remaining mechanical sites). That's expected; just confirm the compound sites compile in isolation by eye. Do NOT commit yet.

---

## Task 3: mechanical sed rewire of the remaining sites + build green

**Files:** all 12 in the union (keys.go, labels.go, columns.go, messages.go, messages_bulk.go, bulk_prompts.go, obsidian.go, slack.go, action_plan.go, commands.go, auto_refresh.go, app.go).

- [ ] **Step 1: apply ordered sed across `internal/tui/*.go` (NON-test)** — order matters; run exactly this sequence per file:

```bash
cd /home/ajramos/dev/giztui
grep -rl -E 'a\.(bulkMode|selected)\b' internal/tui/*.go | grep -v _test | while IFS= read -r f; do
  sed -i \
    -e 's/a\.bulkMode = \(true\|false\)/a.bulk.setMode(\1)/g' \
    -e 's/len(a\.selected)/a.bulk.count()/g' \
    -e 's/a\.selected != nil && a\.selected\(\[[^]]*\]\)/a.bulk.isSelected(REPL\1)/g' \
    -e 's/for \([a-z_]*\) := range a\.selected/for _, \1 := range a.bulk.ids()/g' \
    -e 's/a\.selected = make(map\[string\]bool)/a.bulk.clear()/g' \
    -e 's/delete(a\.selected, \([a-zA-Z0-9_.()\[\]]*\))/a.bulk.remove(\1)/g' \
    "$f"
done
```

(The `isSelected(REPL[...])` is fixed up next — the `[x]` index needs unwrapping to `(x)`.)

- [ ] **Step 2: fix the `isSelected(REPL[x])` form and remaining index reads/writes** — run:
```bash
cd /home/ajramos/dev/giztui
grep -rl -E 'a\.(bulkMode|selected)\b|isSelected\(REPL' internal/tui/*.go | grep -v _test | while IFS= read -r f; do
  sed -i \
    -e 's/a\.bulk\.isSelected(REPL\[\([^]]*\)\])/a.bulk.isSelected(\1)/g' \
    -e 's/a\.selected\[\([^]]*\)\] = true/a.bulk.add(\1)/g' \
    -e 's/a\.selected\[\([^]]*\)\]/a.bulk.isSelected(\1)/g' \
    -e 's/a\.bulkMode/a.bulk.isMode()/g' \
    "$f"
done
```

NOTE: the last two run AFTER all writes/sets are gone (Task 2 handled toggle/nil-init writes; Step 1 handled `= true`/`!= nil &&`). `a.bulkMode` is replaced last (only reads remain — writes were `a.bulkMode = ...` already converted in Step 1).

- [ ] **Step 3: straggler check** — these MUST all be empty:
```bash
grep -rnE 'a\.selected\b' internal/tui/*.go | grep -v _test
grep -rnE 'a\.bulkMode\b' internal/tui/*.go | grep -v _test
grep -rn 'REPL' internal/tui/*.go | grep -v _test
grep -rn 'isMode()()' internal/tui/*.go | grep -v _test   # double-call typo guard
```
If any are non-empty, fix by hand (the surrounding context tells you which method applies).

- [ ] **Step 4: build** — `go build ./...` → success. Fix any compile errors (commonly a stray `a.bulk.isMode()` used as an lvalue, or an index expression the sed mangled — inspect and correct).

- [ ] **Step 5: gofmt + tui tests** — `gofmt -w internal/tui/*.go`; `go test ./internal/tui/ 2>&1 | tail -3` → ok.

- [ ] **Step 6: commit** — `git add -A && git commit -m "refactor(tui): route bulk selection through bulkState (fixes selected-map race)"`. NO Co-Authored-By.

---

## Task 4: verification + finish

- [ ] **Step 1:** `make pre-commit-check` → passed.
- [ ] **Step 2:** `go test -race -count=1 ./internal/tui/ 2>&1 | tail -3` → ok, no races.
- [ ] **Step 3:** `make test 2>&1 | grep -E '^FAIL' || echo NO_FAILURES` → NO_FAILURES.
- [ ] **Step 4:** `make build` → built.
- [ ] **Step 5: harness smoke** — enter bulk mode + select + run a bulk action; confirm it acts on the selection and 0 panics. Suggested sequence (adapt keys to config): load, `b` (bulk mode), `space` ×2 (select two), `a` (bulk archive) or `:archive`, then check the log shows the bulk op ran on the selected count and `grep -aic panic` is 0.
- [ ] **Step 6:** superpowers:finishing-a-development-branch. Manual Mac smoke: bulk select a few, archive/label/prompt them, confirm correct set acted on. NO push/merge without explicit user OK. NO Co-Authored-By.

---

## Self-Review

**Spec coverage:** bulkState type+methods (T1); App composes `bulk`, fields removed (T2.1); toggle/nil-init/value-range/nil-check compound sites (T2.2-5); mechanical mapping for the rest (T3); tests incl. race (T1); gate+race+harness (T4). ✓

**Placeholder scan:** the sed commands are concrete; the only judgment step is fixing stragglers/compile errors (T3.3-4), which is bounded by the straggler greps that must reach empty. ✓

**Type consistency:** `bulkState`/`newBulkState`/`isMode`/`setMode`/`isSelected`/`add`/`remove`/`toggle`/`count`/`ids`/`clear` named identically in T1 and used identically in T2-3. ✓

**Risk note:** the sed in T3 is the dangerous part. The straggler greps (must be empty) + `go build` + `-race` + the harness smoke are the backstops. If sed mangles an index expression, the build catches it. Recommend INLINE execution (one driver, iterating with the greps) over parallel subagents for this mechanical sweep.
