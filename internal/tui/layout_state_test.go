package tui

import "testing"

func TestLayoutState_SizeRoundTrip(t *testing.T) {
	var l layoutState

	if w, h := l.size(); w != 0 || h != 0 {
		t.Fatalf("expected zero size, got %d x %d", w, h)
	}

	l.setSize(120, 40)

	if w, h := l.size(); w != 120 || h != 40 {
		t.Fatalf("expected 120 x 40, got %d x %d", w, h)
	}
}
