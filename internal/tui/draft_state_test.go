package tui

import "testing"

func TestDraftState_ModeRoundTrip(t *testing.T) {
	var d draftState

	if d.isMode() {
		t.Fatalf("zero value isMode() = true, want false")
	}

	d.setMode(true)
	if !d.isMode() {
		t.Fatalf("after setMode(true), isMode() = false, want true")
	}

	d.setMode(false)
	if d.isMode() {
		t.Fatalf("after setMode(false), isMode() = true, want false")
	}
}
