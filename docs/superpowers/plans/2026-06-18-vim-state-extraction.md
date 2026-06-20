# VIM State Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the five VIM key-sequence fields from the `App` god object into a self-contained, mutex-protected `vimState` type, with unit tests and zero user-visible behavior change.

**Architecture:** New package-private `vimState` (in `internal/tui/vim_navigator.go`) owns the VIM state machine behind its own `sync.Mutex` and exposes decision-returning methods. `keys.go` keeps the effects (`executeVim*`, `GotoTop`, `ShowProgress`) and calls `a.vim.*` for state; the timeout goroutine stops taking `a.mu` for VIM fields. `App` composes one `vim vimState` field in place of the five flat fields.

**Tech Stack:** Go, tview/tcell event loop, `sync.Mutex`, standard `testing`.

---

## Reference facts (verified in code)

- Five fields on `App` (`internal/tui/app.go:172-178`): `vimSequence string`, `vimTimeout time.Time`, `vimOperationCount int`, `vimOperationType string`, `vimOriginalMessageID string`.
- All 45 accesses are in `internal/tui/keys.go`, in three handler methods: `handleVimSequence` (1720), `handleVimNavigation` (1764), `handleVimRangeOperation` (1827). The three `executeVim*` methods (2018, 2065, 2110) are pure effects — they take `operation`/`count`/`messageID` params and do NOT touch the vim fields.
- No `_test.go` references any vim field (safe to refactor).
- Config: `config.KeyBindings.VimNavigationTimeoutMs` (default 1000) and `VimRangeTimeoutMs` (default 2000), at `internal/config/config.go:328-329`. keys.go resolves them with a `<=0 → fallback` pattern.
- `keys.go` already imports `fmt`, `strings`, `time`, `services`, `tcell`, `tview`.
- The timeout goroutine (`keys.go:1939-1984`) currently does `a.mu.Lock()` → check `vimOperationType==key && vimOperationCount==0` → capture `vimOriginalMessageID` → reset fields → `a.mu.Unlock()` → run effect. The event-loop writes the same fields WITHOUT `a.mu` (1882, 1917-1919, 2000-2004) — the latent inconsistency this refactor fixes.
- `sequence` is read for decisions only in navigation (`== "g"`); in range-ops it is written but never read for a decision, so its exact contents don't affect behavior.

---

## Task 1: Create `vimState` type + unit tests

**Files:**
- Create: `internal/tui/vim_navigator.go`
- Test: `internal/tui/vim_navigator_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/vim_navigator_test.go`:

```go
package tui

import (
	"testing"
	"time"
)

func TestVimState_AppendDigit(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 2 * time.Second

	// No pending operation → not ok.
	if _, _, ok := v.appendDigit(5, now, d); ok {
		t.Fatal("appendDigit should not accept a digit with no pending operation")
	}

	// Start an operation, then accumulate digits: 5 then 6 → 56.
	v.startOperation("s", now, d, "msg-1")
	if op, count, ok := v.appendDigit(5, now, d); !ok || op != "s" || count != 5 {
		t.Fatalf("first digit: op=%q count=%d ok=%v, want s/5/true", op, count, ok)
	}
	if op, count, ok := v.appendDigit(6, now, d); !ok || op != "s" || count != 56 {
		t.Fatalf("second digit: op=%q count=%d ok=%v, want s/56/true", op, count, ok)
	}
}

func TestVimState_CompleteOperation(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 2 * time.Second

	// Complete with no pending op → not ok.
	if _, _, ok := v.completeOperation("s"); ok {
		t.Fatal("completeOperation should fail with no pending op")
	}

	// s then s (no digits) → count defaults to 1.
	v.startOperation("s", now, d, "msg-1")
	if op, count, ok := v.completeOperation("s"); !ok || op != "s" || count != 1 {
		t.Fatalf("complete s s: op=%q count=%d ok=%v, want s/1/true", op, count, ok)
	}

	// After completion the state is reset → second complete fails.
	if _, _, ok := v.completeOperation("s"); ok {
		t.Fatal("state should be reset after completeOperation")
	}

	// s 5 s → count 5.
	v.startOperation("s", now, d, "msg-1")
	v.appendDigit(5, now, d)
	if op, count, ok := v.completeOperation("s"); !ok || op != "s" || count != 5 {
		t.Fatalf("complete s5s: op=%q count=%d ok=%v, want s/5/true", op, count, ok)
	}

	// Mismatched completion key → not ok.
	v.startOperation("a", now, d, "msg-1")
	if _, _, ok := v.completeOperation("s"); ok {
		t.Fatal("completeOperation with a different key should fail")
	}
}

func TestVimState_ClearIfExpired(t *testing.T) {
	var v vimState
	start := time.Unix(100, 0)
	d := 2 * time.Second

	v.startOperation("s", start, d, "msg-1")

	// timeout was set to start+d (=start+2s); the original expires when now-timeout > d,
	// i.e. now > start+2d (=start+4s). This is the inherited 2x timeout behavior — preserved.

	// Not yet expired (now=start+3s; 3-2=1s, not > 2s) → no reset.
	if v.clearIfExpired(start.Add(3*time.Second), d) {
		t.Fatal("should not expire before start+2d")
	}
	// Past the timeout (now=start+5s; 5-2=3s > 2s) → reset.
	if !v.clearIfExpired(start.Add(5*time.Second), d) {
		t.Fatal("should expire after start+2d")
	}
	// After reset, completing fails (state cleared).
	if _, _, ok := v.completeOperation("s"); ok {
		t.Fatal("expired sequence should have been reset")
	}
	// Idle state (zero timeout) never expires.
	if v.clearIfExpired(start.Add(10*time.Second), d) {
		t.Fatal("idle state should not report expiry")
	}
}

func TestVimState_NavigationGG(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 1 * time.Second

	// Fresh state: 'g' not pending.
	if v.pendingG() {
		t.Fatal("pendingG should be false initially")
	}
	v.startG(now, d)
	if !v.pendingG() {
		t.Fatal("pendingG should be true after startG")
	}
	v.clearSequence()
	if v.pendingG() {
		t.Fatal("pendingG should be false after clearSequence")
	}
}

func TestVimState_TakePendingSingle(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 2 * time.Second

	// No pending op → not ok.
	if _, ok := v.takePendingSingle("s"); ok {
		t.Fatal("takePendingSingle should fail with no pending op")
	}

	// Pending single op (count 0) → returns captured msgID once.
	v.startOperation("s", now, d, "msg-42")
	if id, ok := v.takePendingSingle("s"); !ok || id != "msg-42" {
		t.Fatalf("takePendingSingle: id=%q ok=%v, want msg-42/true", id, ok)
	}
	// Second take fails (already taken/reset).
	if _, ok := v.takePendingSingle("s"); ok {
		t.Fatal("takePendingSingle should fail after the sequence was taken")
	}

	// A sequence with a count is NOT a pending single → not ok.
	v.startOperation("s", now, d, "msg-1")
	v.appendDigit(3, now, d)
	if _, ok := v.takePendingSingle("s"); ok {
		t.Fatal("takePendingSingle should fail when a count was entered")
	}
}
```

- [ ] **Step 2: Run the test, verify it FAILS to compile**

Run: `go test ./internal/tui/ -run TestVimState -v`
Expected: FAIL — `vimState` and its methods are undefined.

- [ ] **Step 3: Implement the type**

Create `internal/tui/vim_navigator.go`:

```go
package tui

import (
	"sync"
	"time"
)

// vimState owns the VIM key-sequence state machine. It is extracted from the App god object so the
// logic is cohesive and unit-testable, and so all access goes through one mutex (the fields were
// previously read under App.mu by a timeout goroutine but written without a lock by the event loop).
//
// Methods return decisions; the caller (keys.go) runs the effects OUTSIDE any lock. Time is injected
// (now + resolved duration) so the state machine is deterministic in tests.
type vimState struct {
	mu             sync.Mutex
	sequence       string    // accumulated key sequence ("g", "s", "s5", ...)
	operationType  string    // pending range-op key ("s", archive key, ...)
	operationCount int        // numeric count being built ("5" in "s5s")
	timeout        time.Time // when the current sequence expires
	originalMsgID  string    // message under the cursor when the sequence started
}

// reset clears the whole state machine. Caller must hold v.mu.
func (v *vimState) reset() {
	v.sequence = ""
	v.operationType = ""
	v.operationCount = 0
	v.timeout = time.Time{}
	v.originalMsgID = ""
}

// clearIfExpired resets the state if the current sequence's timeout has passed. Returns whether a
// reset happened (so the caller can clear any on-screen progress). An idle state (zero timeout)
// never expires.
func (v *vimState) clearIfExpired(now time.Time, d time.Duration) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.timeout.IsZero() && now.Sub(v.timeout) > d {
		v.reset()
		return true
	}
	return false
}

// pendingG reports whether a lone "g" has been typed (waiting for the second g of "gg").
func (v *vimState) pendingG() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.sequence == "g"
}

// startG begins a "g" navigation sequence with a fresh timeout.
func (v *vimState) startG(now time.Time, d time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.sequence = "g"
	v.timeout = now.Add(d)
}

// clearSequence clears the navigation sequence (used for gg/G completion). It does not touch a
// pending range operation, matching the original handlers.
func (v *vimState) clearSequence() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.sequence = ""
	v.timeout = time.Time{}
}

// appendDigit accumulates a digit into the pending operation's count (count = count*10 + digit) and
// refreshes the timeout. Returns the operation key, the new count, and ok=false when no operation is
// pending (so the caller passes the key through).
func (v *vimState) appendDigit(digit int, now time.Time, d time.Duration) (op string, count int, ok bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.operationType == "" {
		return "", 0, false
	}
	v.operationCount = v.operationCount*10 + digit
	v.sequence += string(rune('0' + digit))
	v.timeout = now.Add(d)
	return v.operationType, v.operationCount, true
}

// startOperation begins a new range-operation sequence (e.g. "s"), capturing the message under the
// cursor and setting the timeout.
func (v *vimState) startOperation(key string, now time.Time, d time.Duration, msgID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.operationType = key
	v.operationCount = 0
	v.sequence = key
	v.timeout = now.Add(d)
	v.originalMsgID = msgID
}

// completeOperation finishes a range op when the same key is pressed again ("s5s"). Returns the
// operation and the count (defaulting to 1 when no digits were entered) and resets the state.
// ok=false when the pending operation key does not match.
func (v *vimState) completeOperation(key string) (op string, count int, ok bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.operationType != key {
		return "", 0, false
	}
	count = v.operationCount
	if count == 0 {
		count = 1
	}
	op = v.operationType
	v.reset()
	return op, count, true
}

// takePendingSingle is the timeout-goroutine's atomic check-and-take: if the sequence is still a
// pending single op (same key, no count), it captures the original message ID, resets the state, and
// returns ok=true. Otherwise ok=false (the sequence was completed or cleared in the meantime). This
// closes the race window that the old check-then-act-under-App.mu code had.
func (v *vimState) takePendingSingle(key string) (msgID string, ok bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.operationType == key && v.operationCount == 0 {
		msgID = v.originalMsgID
		v.reset()
		return msgID, true
	}
	return "", false
}

// operationPending reports whether a range operation is currently in progress.
func (v *vimState) operationPending() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.operationType != ""
}
```

Add one more case to `TestVimState_CompleteOperation` (or as a standalone assertion) verifying `operationPending`: append this test function to `vim_navigator_test.go`:

```go
func TestVimState_OperationPending(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 2 * time.Second
	if v.operationPending() {
		t.Fatal("no operation should be pending initially")
	}
	v.startOperation("s", now, d, "m")
	if !v.operationPending() {
		t.Fatal("operation should be pending after startOperation")
	}
	v.completeOperation("s")
	if v.operationPending() {
		t.Fatal("operation should not be pending after completeOperation")
	}
}
```

- [ ] **Step 4: Run the test, verify it PASSES**

Run: `go test ./internal/tui/ -run TestVimState -v`
Expected: PASS (all five TestVimState_* tests).

- [ ] **Step 5: Run with the race detector**

Run: `go test -race ./internal/tui/ -run TestVimState`
Expected: PASS, no race warnings.

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/tui/vim_navigator.go internal/tui/vim_navigator_test.go
git add internal/tui/vim_navigator.go internal/tui/vim_navigator_test.go
git commit -m "feat(tui): add vimState type with unit-tested state machine"
```

---

## Task 2: Rewire `App` and `keys.go` to use `vimState`

**Files:**
- Modify: `internal/tui/app.go:172-178` (replace five fields with one)
- Modify: `internal/tui/keys.go` (handlers + timeout goroutine)

- [ ] **Step 1: Replace the five App fields with one**

In `internal/tui/app.go`, the current block (lines ~170-178) reads:

```go
	// VIM-style navigation
	vimSequence string    // Track VIM key sequences like "gg"
	vimTimeout  time.Time // Timeout for key sequences

	// VIM-style range operations
	vimOperationCount    int    // Track count in sequences (e.g., "5" in "s5s")
	vimOperationType     string // Track operation type (e.g., "s" in "s5s")
	vimOriginalMessageID string // Store message ID when VIM sequence started
```

Replace the whole block with:

```go
	// VIM-style navigation and range operations (state machine in vim_navigator.go)
	vim vimState
```

Note: `vimState` is a value field (not a pointer) and its zero value is ready to use — no constructor change needed. If `app.go` no longer uses `time.Time` elsewhere the import stays (it is used widely); do not remove imports unless the build complains.

- [ ] **Step 2: Rewire `handleVimSequence` (keys.go ~1731-1748)**

Replace the timeout-clearing block. The current code (after `now := time.Now()`) is:

```go
	// Clear sequence if timeout exceeded (configurable for range operations)
	rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
	if rangeTimeoutMs <= 0 {
		rangeTimeoutMs = 2000 // Default fallback
	}
	if !a.vimTimeout.IsZero() && now.Sub(a.vimTimeout) > time.Duration(rangeTimeoutMs)*time.Millisecond {
		// OBLITERATED: empty logger branch eliminated! 💥
		a.vimSequence = ""
		a.vimOperationType = ""
		a.vimOperationCount = 0
		a.vimOriginalMessageID = ""
		// Clear any status message for cancelled sequence
		go func() {
			a.GetErrorHandler().ClearProgress()
		}()
	}
```

Replace with:

```go
	// Clear sequence if timeout exceeded (configurable for range operations)
	rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
	if rangeTimeoutMs <= 0 {
		rangeTimeoutMs = 2000 // Default fallback
	}
	if a.vim.clearIfExpired(now, time.Duration(rangeTimeoutMs)*time.Millisecond) {
		// Clear any status message for cancelled sequence
		go func() {
			a.GetErrorHandler().ClearProgress()
		}()
	}
```

- [ ] **Step 3: Rewire `handleVimNavigation` (keys.go ~1764-1824)**

Replace the `'g'` case body. Current:

```go
	case 'g':
		if a.vimSequence == "g" {
			// Double 'g' - context-dependent behavior
			a.vimSequence = ""
			a.vimTimeout = time.Time{}

			// CRITICAL: Check focus context for gg behavior
			if a.currentFocus == "text" && a.enhancedTextView != nil {
```

becomes:

```go
	case 'g':
		if a.vim.pendingG() {
			// Double 'g' - context-dependent behavior
			a.vim.clearSequence()

			// CRITICAL: Check focus context for gg behavior
			if a.currentFocus == "text" && a.enhancedTextView != nil {
```

And the `else` branch that starts the sequence. Current:

```go
		} else {
			// Start of sequence - wait for next key
			a.vimSequence = "g"

			// Use configurable navigation timeout for gg sequence
			navTimeoutMs := a.Keys.VimNavigationTimeoutMs
			if navTimeoutMs <= 0 {
				navTimeoutMs = 1000 // Default fallback
			}
			a.vimTimeout = now.Add(time.Duration(navTimeoutMs) * time.Millisecond)
			return true
		}
```

becomes:

```go
		} else {
			// Start of sequence - wait for next key
			// Use configurable navigation timeout for gg sequence
			navTimeoutMs := a.Keys.VimNavigationTimeoutMs
			if navTimeoutMs <= 0 {
				navTimeoutMs = 1000 // Default fallback
			}
			a.vim.startG(now, time.Duration(navTimeoutMs)*time.Millisecond)
			return true
		}
```

And the `'G'` case. Current first two lines of the case body:

```go
	case 'G':
		// Single 'G' - context-dependent behavior
		a.vimSequence = ""
		a.vimTimeout = time.Time{}
```

becomes:

```go
	case 'G':
		// Single 'G' - context-dependent behavior
		a.vim.clearSequence()
```

(The focus-context effect calls below — `GotoTop`/`GotoBottom`/`executeGoToFirst`/`executeGoToCommand` — are unchanged.)

- [ ] **Step 4: Rewire the digit branch in `handleVimRangeOperation` (keys.go ~1877-1902)**

Current:

```go
	// Handle digits in sequence
	if key >= '0' && key <= '9' {
		if a.vimOperationType != "" {
			// We're building a count: s5 -> s56
			digit := int(key - '0')
			oldCount := a.vimOperationCount
			a.vimOperationCount = a.vimOperationCount*10 + digit
			a.vimSequence += string(key)
			// Use configurable range timeout for digit sequences
			rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
			if rangeTimeoutMs <= 0 {
				rangeTimeoutMs = 2000 // Default fallback
			}
			a.vimTimeout = now.Add(time.Duration(rangeTimeoutMs) * time.Millisecond)

			if a.logger != nil {
				a.logger.Printf("VIM digit pressed: %c, operation: %s, oldCount: %d, digit: %d, newCount: %d", key, a.vimOperationType, oldCount, digit, a.vimOperationCount)
			}

			// Show status
			go func() {
				a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("%s%d... (waiting for operation)", a.vimOperationType, a.vimOperationCount))
			}()
			return true
		}
		return false
	}
```

Replace with:

```go
	// Handle digits in sequence
	if key >= '0' && key <= '9' {
		rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
		if rangeTimeoutMs <= 0 {
			rangeTimeoutMs = 2000 // Default fallback
		}
		if op, count, ok := a.vim.appendDigit(int(key-'0'), now, time.Duration(rangeTimeoutMs)*time.Millisecond); ok {
			if a.logger != nil {
				a.logger.Printf("VIM digit pressed: %c, operation: %s, newCount: %d", key, op, count)
			}
			// Show status
			go func() {
				a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("%s%d... (waiting for operation)", op, count))
			}()
			return true
		}
		return false
	}
```

- [ ] **Step 5: Rewire the operation-start / complete branches (keys.go ~1915-2009)**

Current (the `if a.vimOperationType == ""` start branch through the `else if` complete branch):

```go
	if a.vimOperationType == "" {
		// Starting new sequence: s, a, d, etc.
		a.vimOperationType = string(key)
		a.vimOperationCount = 0
		a.vimSequence = string(key)
		// Use configurable range timeout for VIM operation sequences
		rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
		if rangeTimeoutMs <= 0 {
			rangeTimeoutMs = 2000 // Default fallback
		}
		a.vimTimeout = now.Add(time.Duration(rangeTimeoutMs) * time.Millisecond)

		// CRITICAL FIX: Capture current message ID when sequence starts
		// This prevents issues where cursor moves during the timeout delay
		a.vimOriginalMessageID = a.GetCurrentMessageID()

		// VIM sequence started

		// Show status and consume the key to prevent single operation
		go func() {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("%s... (enter count then %s, or wait for timeout)", string(key), string(key)))
		}()

		// Start timeout goroutine to execute single operation if no sequence completed
		go func() {
			// OBLITERATED: empty logger branch eliminated! 💥

			// Use configurable range timeout for single operation fallback
			rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
			if rangeTimeoutMs <= 0 {
				rangeTimeoutMs = 2000 // Default fallback
			}
			time.Sleep(time.Duration(rangeTimeoutMs) * time.Millisecond)
			// OBLITERATED: empty logger branch eliminated! 💥
			a.mu.Lock()

			// OBLITERATED: empty logger branch eliminated! 💥

			// Check if sequence is still pending (not completed or cleared)
			if a.vimOperationType == string(key) && a.vimOperationCount == 0 {
				// OBLITERATED: empty logger branch eliminated! 💥
				// Capture original message ID while holding mutex
				originalMessageID := a.vimOriginalMessageID

				// Clear sequence state while holding mutex
				a.vimSequence = ""
				a.vimOperationType = ""
				a.vimTimeout = time.Time{}
				a.vimOriginalMessageID = ""

				// OBLITERATED: empty logger branch eliminated! 💥

				// Release mutex BEFORE accessing UI elements or executing operations
				a.mu.Unlock()

				// OBLITERATED: empty logger branch eliminated! 💥

				go func() {
					a.GetErrorHandler().ClearProgress()
				}()

				// Execute single operation with original message ID (without mutex)
				// OBLITERATED: empty logger branch eliminated! 💥
				a.executeVimSingleOperationWithID(string(key), originalMessageID)
				// OBLITERATED: empty logger branch eliminated! 💥
			} else {
				// OBLITERATED: empty logger branch eliminated! 💥
				a.mu.Unlock()
			}
		}()

		return true // Consume the key to prevent immediate single operation
	} else if a.vimOperationType == string(key) {
		// Completing sequence: s5s, a3a, etc.
		count := a.vimOperationCount
		if count == 0 {
			count = 1 // Default to 1 if no count specified
		}

		if a.logger != nil {
			a.logger.Printf("VIM completing sequence: %s%d%s, passing count=%d to executeVimRangeOperation", string(key), count, string(key), count)
		}

		// Clear sequence state
		operation := a.vimOperationType
		a.vimSequence = ""
		a.vimOperationType = ""
		a.vimOperationCount = 0
		a.vimTimeout = time.Time{}
		a.vimOriginalMessageID = ""

		// Execute the range operation
		a.executeVimRangeOperation(operation, count)
		return true
	}
```

Replace the entire block above with:

```go
	if op, count, ok := a.vim.completeOperation(string(key)); ok {
		// Completing sequence: s5s, a3a, etc.
		if a.logger != nil {
			a.logger.Printf("VIM completing sequence: %s%d%s, passing count=%d to executeVimRangeOperation", op, count, op, count)
		}
		// Execute the range operation
		a.executeVimRangeOperation(op, count)
		return true
	}

	if !a.vim.operationPending() {
		// Starting new sequence: s, a, d, etc.
		rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
		if rangeTimeoutMs <= 0 {
			rangeTimeoutMs = 2000 // Default fallback
		}
		// CRITICAL FIX: Capture current message ID when sequence starts so a cursor move during
		// the timeout delay does not change the target.
		a.vim.startOperation(string(key), now, time.Duration(rangeTimeoutMs)*time.Millisecond, a.GetCurrentMessageID())

		// Show status and consume the key to prevent single operation
		go func() {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("%s... (enter count then %s, or wait for timeout)", string(key), string(key)))
		}()

		// Start timeout goroutine to execute single operation if no sequence completed
		go func() {
			time.Sleep(time.Duration(rangeTimeoutMs) * time.Millisecond)
			if originalMessageID, taken := a.vim.takePendingSingle(string(key)); taken {
				go func() {
					a.GetErrorHandler().ClearProgress()
				}()
				a.executeVimSingleOperationWithID(string(key), originalMessageID)
			}
		}()

		return true // Consume the key to prevent immediate single operation
	}
```

Note on ordering: the original checked "start (empty)" first, then "complete (same key)". They are mutually exclusive (a key cannot be both the first op AND a matching completion in the same call), so checking `completeOperation` first is equivalent — `completeOperation` returns ok=false unless an operation with that exact key is already pending, in which case the old code would have taken the `else if` branch anyway.

- [ ] **Step 6: Build and check for stragglers**

Run: `go build ./...`
Expected: success. If the compiler reports any remaining `a.vimSequence` / `a.vimTimeout` / `a.vimOperationCount` / `a.vimOperationType` / `a.vimOriginalMessageID` reference, that is a missed site — rewire it via the matching `vimState` method and rebuild.

Then confirm there are no stragglers:
Run: `grep -nE 'a\.vim(Sequence|Timeout|Operation(Count|Type)|OriginalMessageID)' internal/tui/*.go`
Expected: no output.

- [ ] **Step 7: Run the TUI test suite + race detector**

Run: `go test ./internal/tui/ 2>&1 | tail -3`
Expected: `ok  github.com/ajramos/giztui/internal/tui`

Run: `go test -race ./internal/tui/ -run 'TestVimState' 2>&1 | tail -3`
Expected: PASS, no race warnings.

- [ ] **Step 8: Commit**

```bash
gofmt -w internal/tui/app.go internal/tui/keys.go internal/tui/vim_navigator.go
git add internal/tui/app.go internal/tui/keys.go internal/tui/vim_navigator.go
git commit -m "refactor(tui): route VIM key handling through vimState (removes 5 App fields, fixes locking)"
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

Use the superpowers:finishing-a-development-branch skill. Note that user-visible VIM behavior must be smoke-tested manually on the user's Mac (gg/G/s5s and single-op-after-timeout) before merge. Do NOT push/merge without explicit user confirmation (project rule: "commit" ≠ "publish").

---

## Self-Review

**Spec coverage:**
- `vimState` type + methods in `vim_navigator.go` → Task 1. ✓
- App composes `vim vimState`; five fields removed → Task 2 Step 1. ✓
- keys.go rewired to `a.vim.*`; timeout goroutine no longer takes `a.mu` → Task 2 Steps 2-6. ✓
- `vim_navigator_test.go` covering the state machine → Task 1. ✓
- `make pre-commit-check` + `-race` green → Task 3. ✓
- No user-visible behavior change → preserved by mapping each field access to an equivalent method; manual smoke test in Task 3 Step 5. ✓
- Latent race fixed → `takePendingSingle` (atomic check-and-take), all access under `vimState.mu` → Task 1 + Task 2. ✓

**Type/signature consistency (used identically across tasks):**
`clearIfExpired(now, d) bool`, `pendingG() bool`, `startG(now, d)`, `clearSequence()`, `appendDigit(digit, now, d) (op, count, ok)`, `startOperation(key, now, d, msgID)`, `completeOperation(key) (op, count, ok)`, `takePendingSingle(key) (msgID, ok)`, `operationPending() bool`, private `reset()`. ✓

**Placeholder scan:** no TBD/TODO; every code step shows full before/after. ✓

**Behavioral equivalence notes (intentional, documented):** `clearIfExpired` also zeroes `timeout` (original left it set, but the observable result — no pending sequence — is identical); `completeOperation` applies the count-default-of-1 that keys.go used to apply inline. Both preserve user-visible behavior.
