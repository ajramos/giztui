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

	if _, _, ok := v.completeOperation("s"); ok {
		t.Fatal("completeOperation should fail with no pending op")
	}

	v.startOperation("s", now, d, "msg-1")
	if op, count, ok := v.completeOperation("s"); !ok || op != "s" || count != 1 {
		t.Fatalf("complete s s: op=%q count=%d ok=%v, want s/1/true", op, count, ok)
	}

	if _, _, ok := v.completeOperation("s"); ok {
		t.Fatal("state should be reset after completeOperation")
	}

	v.startOperation("s", now, d, "msg-1")
	v.appendDigit(5, now, d)
	if op, count, ok := v.completeOperation("s"); !ok || op != "s" || count != 5 {
		t.Fatalf("complete s5s: op=%q count=%d ok=%v, want s/5/true", op, count, ok)
	}

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

	// timeout is set to start+d (=start+2s); expiry is now.Sub(timeout) > d, i.e. now > start+2d
	// (=start+4s). This preserves the original handler's inherited 2x-timeout behavior.

	// now=start+3s: 3-2=1s, not > 2s → no reset.
	if v.clearIfExpired(start.Add(3*time.Second), d) {
		t.Fatal("should not expire before start+2d")
	}
	// now=start+5s: 5-2=3s > 2s → reset.
	if !v.clearIfExpired(start.Add(5*time.Second), d) {
		t.Fatal("should expire after start+2d")
	}
	if _, _, ok := v.completeOperation("s"); ok {
		t.Fatal("expired sequence should have been reset")
	}
	if v.clearIfExpired(start.Add(10*time.Second), d) {
		t.Fatal("idle state should not report expiry")
	}
}

func TestVimState_NavigationGG(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 1 * time.Second

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

// TestVimState_ClearSequencePreservesPendingOp guards the gg/G-mid-range-op corner case: pressing a
// navigation key while a range operation is pending must NOT discard the captured message ID, so the
// timeout-fired single op still targets the original message (matching the pre-refactor handlers).
func TestVimState_ClearSequencePreservesPendingOp(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 2 * time.Second

	// Start a range op (captures msg-42), then a navigation key clears the nav sequence.
	v.startOperation("s", now, d, "msg-42")
	v.clearSequence()

	// The pending single op must still resolve to the captured message.
	if id, ok := v.takePendingSingle("s"); !ok || id != "msg-42" {
		t.Fatalf("after clearSequence: takePendingSingle id=%q ok=%v, want msg-42/true", id, ok)
	}
}

func TestVimState_TakePendingSingle(t *testing.T) {
	var v vimState
	now := time.Unix(0, 0)
	d := 2 * time.Second

	if _, ok := v.takePendingSingle("s"); ok {
		t.Fatal("takePendingSingle should fail with no pending op")
	}

	v.startOperation("s", now, d, "msg-42")
	if id, ok := v.takePendingSingle("s"); !ok || id != "msg-42" {
		t.Fatalf("takePendingSingle: id=%q ok=%v, want msg-42/true", id, ok)
	}
	if _, ok := v.takePendingSingle("s"); ok {
		t.Fatal("takePendingSingle should fail after the sequence was taken")
	}

	v.startOperation("s", now, d, "msg-1")
	v.appendDigit(3, now, d)
	if _, ok := v.takePendingSingle("s"); ok {
		t.Fatal("takePendingSingle should fail when a count was entered")
	}
}

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
