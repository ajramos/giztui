package tui

import "testing"

func TestAIPanelState_Visible(t *testing.T) {
	var s aiPanelState
	if s.visible.Load() {
		t.Fatal("zero aiPanelState must not be visible")
	}
	s.visible.Store(true)
	if !s.visible.Load() {
		t.Fatal("visible should be true after Store(true)")
	}
}

func TestAIPanelState_StreamingCancel(t *testing.T) {
	var s aiPanelState

	// No active stream: cancelStreaming is a no-op returning false.
	if s.isStreaming() {
		t.Fatal("zero state must not be streaming")
	}
	if s.cancelStreaming() {
		t.Fatal("cancelStreaming with no active op must return false")
	}

	calls := 0
	s.setStreamingCancel(func() { calls++ })
	if !s.isStreaming() {
		t.Fatal("isStreaming must be true after setStreamingCancel")
	}

	// First cancel calls the func exactly once and clears it.
	if !s.cancelStreaming() {
		t.Fatal("cancelStreaming must return true when an op is active")
	}
	if calls != 1 {
		t.Fatalf("cancel func should be called once, got %d", calls)
	}
	// Second cancel does nothing (already cleared).
	if s.cancelStreaming() {
		t.Fatal("second cancelStreaming must return false")
	}
	if calls != 1 {
		t.Fatalf("cancel func must not be called again, got %d", calls)
	}
	if s.isStreaming() {
		t.Fatal("must not be streaming after cancel")
	}

	// clearStreamingCancel drops the func WITHOUT calling it.
	cleared := 0
	s.setStreamingCancel(func() { cleared++ })
	s.clearStreamingCancel()
	if cleared != 0 {
		t.Fatalf("clearStreamingCancel must not call the func, got %d", cleared)
	}
	if s.isStreaming() {
		t.Fatal("must not be streaming after clear")
	}
}
