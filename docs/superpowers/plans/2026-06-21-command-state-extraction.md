# Command State Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the 6 command-bar fields from the `App` god object into a self-contained `commandState` type (history logic unit-tested), with zero user-visible behavior change.

**Architecture:** New `commandState` (in `internal/tui/command_state.go`) holds the command-bar state. `cmdMode` becomes an `atomic.Bool` (it is read by the focus-backup goroutine in `executeContentSearch`); the other five fields are plain (event-loop-only). The command-history logic (add + Up/Down navigation) moves to pure methods. `App` composes one `cmd commandState` field; handlers call `a.cmd.*`.

**Tech Stack:** Go, `sync/atomic`, tview event loop, standard `testing`.

---

## Reference facts (verified in code)

- Six fields on `App` (`internal/tui/app.go:93-98`): `cmdMode bool`, `cmdBuffer string`, `cmdHistory []string`, `cmdHistoryIndex int`, `cmdSuggestion string`, `cmdFocusOverride string`. Initialized at app.go:350-353 (`cmdMode:false`, `cmdBuffer:""`, `cmdHistory:make([]string,0)`).
- `cmdMode` writes: commands.go:51 (`= true`), commands.go:158 (`= false`). Reads: keys.go:560, keys.go:568, messages.go:500 (event loop), commands.go:1022 (**inside the `go func()` focus-backup loop in `executeContentSearch`** — off the event loop).
- History logic: `addToHistory` (commands.go:942-951) skips empty + consecutive-dup, appends, caps at 100, resets cursor to `len`. Open sets cursor to `len` (commands.go:68). Up/Down navigation at commands.go:97-115.
- `cmdBuffer`: synced in the input ChangedFunc (commands.go:121), set on open (commands.go:52), cleared on close (commands.go:159), used in `completeCommand` (commands.go:654-655). `cmdSuggestion`: set/cleared at commands.go:53/160, read at commands.go:654. `cmdFocusOverride`: set at commands.go:1015 (`"enhanced-text"`) + action_plan_rules.go:221 (`"keep"`), read+cleared at keys.go:1694-1696.
- Access counts (non-test): cmdBuffer 14, cmdFocusOverride 5, cmdHistory 22, cmdHistoryIndex 24, cmdMode 6, cmdSuggestion 5 — across commands.go, keys.go, messages.go, action_plan_rules.go.

---

## Task 1: Create `commandState` type + history tests

**Files:**
- Create: `internal/tui/command_state.go`
- Test: `internal/tui/command_state_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/command_state_test.go`:

```go
package tui

import "testing"

func TestCommandState_AddToHistory(t *testing.T) {
	var c commandState
	c.addToHistory("a")
	c.addToHistory("b")
	c.addToHistory("b") // consecutive dup → skipped
	c.addToHistory("")  // empty → skipped
	if len(c.history) != 2 || c.history[0] != "a" || c.history[1] != "b" {
		t.Fatalf("history = %v, want [a b]", c.history)
	}
	// addToHistory resets the cursor to the end.
	if c.historyIndex != 2 {
		t.Errorf("historyIndex = %d, want 2 (end)", c.historyIndex)
	}
	// cap at 100: oldest dropped.
	var big commandState
	for i := 0; i < 150; i++ {
		big.addToHistory(string(rune('A'+i%26)) + string(rune('0'+i%10)) + string(rune(i)))
	}
	if len(big.history) != 100 {
		t.Errorf("history len = %d, want 100 (capped)", len(big.history))
	}
}

func TestCommandState_ResetHistoryCursor(t *testing.T) {
	c := commandState{history: []string{"x", "y", "z"}}
	c.historyIndex = 0
	c.resetHistoryCursor()
	if c.historyIndex != 3 {
		t.Errorf("historyIndex = %d, want 3 (len)", c.historyIndex)
	}
}

func TestCommandState_HistoryUpDown(t *testing.T) {
	c := commandState{history: []string{"first", "second", "third"}}
	c.resetHistoryCursor() // cursor at 3 (new line)

	// Up walks newest→oldest.
	if txt, ok := c.historyUp(); !ok || txt != "third" {
		t.Fatalf("up #1 = %q,%v, want third,true", txt, ok)
	}
	if txt, ok := c.historyUp(); !ok || txt != "second" {
		t.Fatalf("up #2 = %q,%v, want second,true", txt, ok)
	}
	if txt, ok := c.historyUp(); !ok || txt != "first" {
		t.Fatalf("up #3 = %q,%v, want first,true", txt, ok)
	}
	// Up at the top is a no-op.
	if _, ok := c.historyUp(); ok {
		t.Fatal("up at top should return ok=false")
	}

	// Down walks back toward newest.
	if txt, ok := c.historyDown(); !ok || txt != "second" {
		t.Fatalf("down #1 = %q,%v, want second,true", txt, ok)
	}
	if txt, ok := c.historyDown(); !ok || txt != "third" {
		t.Fatalf("down #2 = %q,%v, want third,true", txt, ok)
	}
	// Down past the end clears the input and parks the cursor at len.
	if txt, ok := c.historyDown(); !ok || txt != "" {
		t.Fatalf("down past end = %q,%v, want \"\",true", txt, ok)
	}
	if c.historyIndex != 3 {
		t.Errorf("cursor = %d, want 3 (parked at len)", c.historyIndex)
	}
}
```

- [ ] **Step 2: Run the test, verify it FAILS to compile**

Run: `go test ./internal/tui/ -run TestCommandState -v`
Expected: FAIL — `commandState` undefined.

- [ ] **Step 3: Implement the type**

Create `internal/tui/command_state.go`:

```go
package tui

import "sync/atomic"

// commandState holds the `:`-command bar state, extracted from the App god object so the command
// history logic is cohesive and unit-testable. Only `mode` crosses a goroutine boundary (the
// focus-backup loop in executeContentSearch reads it), so it is an atomic.Bool; the rest are
// event-loop-only plain fields. atomic.Bool is non-copyable: never copy a commandState; use it as
// a field accessed via a.cmd.* with pointer-receiver methods.
type commandState struct {
	mode          atomic.Bool // command bar open?
	buffer        string      // current command text
	suggestion    string      // current Tab/auto suggestion
	focusOverride string      // overrides focus restoration after a special command
	history       []string    // executed-command history (capped)
	historyIndex  int         // cursor into history; == len(history) means "new line"
}

// addToHistory records a command, skipping empties and a consecutive duplicate, capping the history
// at 100 (oldest dropped), and resetting the cursor to the end.
func (c *commandState) addToHistory(cmd string) {
	if cmd == "" || (len(c.history) > 0 && c.history[len(c.history)-1] == cmd) {
		return
	}
	c.history = append(c.history, cmd)
	if len(c.history) > 100 {
		c.history = c.history[1:]
	}
	c.historyIndex = len(c.history)
}

// resetHistoryCursor parks the cursor at the end (the empty "new line"); called when the bar opens.
func (c *commandState) resetHistoryCursor() {
	c.historyIndex = len(c.history)
}

// historyUp moves to an older entry and returns its text. ok=false when already at the top (the
// caller then leaves the input unchanged).
func (c *commandState) historyUp() (string, bool) {
	if c.historyIndex > 0 {
		c.historyIndex--
		if c.historyIndex >= 0 && c.historyIndex < len(c.history) {
			return c.history[c.historyIndex], true
		}
	}
	return "", false
}

// historyDown moves toward newer entries. Past the end it parks the cursor at len(history) and
// returns ("", true) so the caller clears the input. ok is true whenever the caller should set the
// input to the returned text.
func (c *commandState) historyDown() (string, bool) {
	if c.historyIndex < len(c.history)-1 {
		c.historyIndex++
		if c.historyIndex >= 0 && c.historyIndex < len(c.history) {
			return c.history[c.historyIndex], true
		}
		return "", false
	}
	c.historyIndex = len(c.history)
	return "", true
}
```

- [ ] **Step 4: Run the test, verify it PASSES**

Run: `go test ./internal/tui/ -run TestCommandState -v`
Expected: PASS (all three TestCommandState_* tests).

- [ ] **Step 5: Race check**

Run: `go test -race ./internal/tui/ -run TestCommandState`
Expected: PASS, no race warnings.

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/tui/command_state.go internal/tui/command_state_test.go
git add internal/tui/command_state.go internal/tui/command_state_test.go
git commit -m "feat(tui): add commandState type with unit-tested history logic"
```

---

## Task 2: Rewire `App` and call sites to `commandState`

**Files:**
- Modify: `internal/tui/app.go` (field block 93-98; init 350-353)
- Modify: `internal/tui/commands.go`, `internal/tui/keys.go`, `internal/tui/messages.go`, `internal/tui/action_plan_rules.go`

- [ ] **Step 1: Replace the six App fields with one**

In `internal/tui/app.go`, replace the field block (lines ~92-98):

```go
	// Command system (k9s style)
	cmdMode          bool     // Whether we're in command mode
	cmdBuffer        string   // Current command buffer
	cmdHistory       []string // Command history
	cmdHistoryIndex  int      // Current position in history
	cmdSuggestion    string   // Current command suggestion
	cmdFocusOverride string   // Override focus restoration for special commands
```

with:

```go
	// Command bar state (the `:` prompt) — state machine in command_state.go
	cmd commandState
```

- [ ] **Step 2: Remove the obsolete initializers**

In `internal/tui/app.go`, delete the three initializer lines (≈350-353):

```go
		cmdMode:            false,
		cmdBuffer:          "",
		cmdHistory:         make([]string, 0),
```

(The `commandState` zero value already gives `mode=false`, `buffer=""`, `history=nil`. If a later
line in the same literal initialized `cmdHistoryIndex`, remove that too — search the literal for any
`cmd*:` keys and delete them all.)

- [ ] **Step 3: Rewire `cmdMode` (atomic Load/Store)**

`cmdMode` is now `a.cmd.mode` (an `atomic.Bool`). Apply these exact edits:

- commands.go:51 `a.cmdMode = true` → `a.cmd.mode.Store(true)`
- commands.go:158 `a.cmdMode = false` → `a.cmd.mode.Store(false)`
- commands.go:1022 `if !a.cmdMode {` → `if !a.cmd.mode.Load() {`
- keys.go:560 `if a.cmdMode {` → `if a.cmd.mode.Load() {`
- keys.go:568 `if a.cmdMode {` → `if a.cmd.mode.Load() {`
- messages.go:500 `if a.cmdMode {` → `if a.cmd.mode.Load() {`

Then `grep -n 'a\.cmdMode' internal/tui/*.go | grep -v _test` must return nothing.

- [ ] **Step 4: Rewire the history navigation in `commands.go`**

Replace the open-cursor line (commands.go:68):

```go
	// Start at end of history
	a.cmdHistoryIndex = len(a.cmdHistory)
```

with:

```go
	// Start at end of history
	a.cmd.resetHistoryCursor()
```

Replace the Up/Down block (commands.go:97-115):

```go
		case tcell.KeyUp:
			if a.cmdHistoryIndex > 0 {
				a.cmdHistoryIndex--
				if a.cmdHistoryIndex >= 0 && a.cmdHistoryIndex < len(a.cmdHistory) {
					input.SetText(a.cmdHistory[a.cmdHistoryIndex])
				}
			}
			return nil
		case tcell.KeyDown:
			if a.cmdHistoryIndex < len(a.cmdHistory)-1 {
				a.cmdHistoryIndex++
				if a.cmdHistoryIndex >= 0 && a.cmdHistoryIndex < len(a.cmdHistory) {
					input.SetText(a.cmdHistory[a.cmdHistoryIndex])
				}
			} else {
				a.cmdHistoryIndex = len(a.cmdHistory)
				input.SetText("")
			}
			return nil
```

with:

```go
		case tcell.KeyUp:
			if txt, ok := a.cmd.historyUp(); ok {
				input.SetText(txt)
			}
			return nil
		case tcell.KeyDown:
			if txt, ok := a.cmd.historyDown(); ok {
				input.SetText(txt)
			}
			return nil
```

- [ ] **Step 5: Move `addToHistory` onto the type**

In `internal/tui/commands.go`, delete the `App.addToHistory` method (commands.go:941-951):

```go
// addToHistory adds a command to the history
func (a *App) addToHistory(cmd string) {
	if cmd == "" || (len(a.cmdHistory) > 0 && a.cmdHistory[len(a.cmdHistory)-1] == cmd) {
		return
	}
	a.cmdHistory = append(a.cmdHistory, cmd)
	if len(a.cmdHistory) > 100 {
		a.cmdHistory = a.cmdHistory[1:]
	}
	a.cmdHistoryIndex = len(a.cmdHistory)
}
```

Its only caller is commands.go:704 `a.addToHistory(cmd)` → change it to `a.cmd.addToHistory(cmd)`.
Also remove the stale comment block at app.go:3314-3315 (`// (moved to commands.go) addToHistory`).

- [ ] **Step 6: Rewire the plain fields (buffer / suggestion / focusOverride)**

These have no logic — mechanical field rename across the four files:

- `a.cmdBuffer` → `a.cmd.buffer`
- `a.cmdSuggestion` → `a.cmd.suggestion`
- `a.cmdFocusOverride` → `a.cmd.focusOverride`

And any remaining direct `a.cmdHistory` / `a.cmdHistoryIndex` reads (there should be none after Steps 4-5; if the grep in Step 7 finds any, rewire each through the matching method or `a.cmd.history` / `a.cmd.historyIndex`).

- [ ] **Step 7: Build and check for stragglers**

Run: `go build ./...`
Expected: success. Fix any compile error by rewiring the named field.

Run: `grep -nE 'a\.cmd(Mode|Buffer|History|HistoryIndex|Suggestion|FocusOverride)\b' internal/tui/*.go | grep -v _test`
Expected: no output.

- [ ] **Step 8: Tests + race**

Run: `go test ./internal/tui/ 2>&1 | tail -3` → `ok`.
Run: `go test -race ./internal/tui/ -run 'TestCommandState' 2>&1 | tail -3` → PASS, no races.
Run `gofmt -w internal/tui/app.go internal/tui/commands.go internal/tui/keys.go internal/tui/messages.go internal/tui/action_plan_rules.go`.

- [ ] **Step 9: Commit**

```bash
git add internal/tui/app.go internal/tui/commands.go internal/tui/keys.go internal/tui/messages.go internal/tui/action_plan_rules.go
git commit -m "refactor(tui): route command-bar state through commandState (removes 6 App fields)"
```

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

Use the superpowers:finishing-a-development-branch skill. Manual command-bar smoke test on the user's Mac before merge: `:` opens the bar; typing shows the live hint; Tab completes; Up/Down walk command history (and Down past the newest clears the line); Enter executes and the command is remembered (no empties/consecutive dups; old ones drop after 100); Esc closes; `:plan rules` and content-search still restore focus correctly. Do NOT push/merge without explicit user confirmation (project rule: "commit" ≠ "publish"). Do NOT add a `Co-Authored-By` line to any commit (project CLAUDE.md forbids it).

---

## Self-Review

**Spec coverage:**
- `commandState` type + history methods in `command_state.go` → Task 1. ✓
- App composes `cmd commandState`; six fields + initializers removed → Task 2 Steps 1-2. ✓
- `cmdMode` synchronized via `atomic.Bool` (the focus-backup goroutine reads it) → Task 1 (field) + Task 2 Step 3 (Load/Store). ✓
- History logic (add/up/down/reset) extracted + rewired → Task 1 + Task 2 Steps 4-5. ✓
- Plain fields rewired; no stray `a.cmd*` refs → Task 2 Steps 6-7. ✓
- `make pre-commit-check` + `-race` green → Task 3. ✓
- No user-visible behavior change → each field access maps 1:1; manual smoke test in Task 3 Step 5. ✓

**Type/signature consistency:** `commandState` fields (`mode atomic.Bool`, `buffer`, `suggestion`, `focusOverride`, `history`, `historyIndex`) and methods (`addToHistory(string)`, `resetHistoryCursor()`, `historyUp() (string,bool)`, `historyDown() (string,bool)`) are defined in Task 1 and used identically in Task 2. `a.cmd.mode.Load()/.Store()` used consistently. ✓

**Placeholder scan:** no TBD/TODO; every code step shows the before/after. The Step-2/6 "search the literal / grep finds any" notes are bounded clean-up instructions backed by the Step-7 straggler grep, not placeholders. ✓
