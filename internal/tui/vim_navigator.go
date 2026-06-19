package tui

import (
	"strconv"
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
	operationCount int       // numeric count being built ("5" in "s5s")
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
// never expires. The expiry uses now.Sub(timeout) > d to preserve the original handler's behavior
// exactly (timeout is set to start+d, so a sequence effectively clears at start+2d).
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

// clearSequence clears the navigation sequence (used for gg/G completion). It touches only the
// navigation fields and deliberately leaves operationType AND originalMsgID untouched, matching the
// original handlers: pressing gg/G mid-range-op must NOT discard the captured message ID, or the
// pending single op would later fire on the wrong message.
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
	v.sequence += strconv.Itoa(digit)
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
