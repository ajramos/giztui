package tui

import (
	"context"
	"testing"
)

func TestAutoRefreshLifecycleIdempotent(t *testing.T) {
	a := &App{}
	a.ctx = context.Background()
	a.startAutoRefresh()
	if !a.isAutoRefreshRunning() {
		t.Fatal("expected running after start")
	}
	// Starting again must not panic or spawn a second ticker.
	a.startAutoRefresh()
	a.stopAutoRefresh()
	if a.isAutoRefreshRunning() {
		t.Fatal("expected stopped after stop")
	}
	// Stopping again must be a no-op.
	a.stopAutoRefresh()
}

func TestAutoRefreshSafeState(t *testing.T) {
	a := &App{}
	a.currentActivePicker = PickerNone
	a.search.SetMode("")
	a.bulkMode = false
	if !a.isAutoRefreshSafeState() {
		t.Error("plain inbox with nothing open should be safe")
	}
	a.currentActivePicker = PickerLabels
	if a.isAutoRefreshSafeState() {
		t.Error("open picker must be unsafe")
	}
	a.currentActivePicker = PickerNone
	a.search.SetMode("remote")
	if a.isAutoRefreshSafeState() {
		t.Error("search mode must be unsafe")
	}
	a.search.SetMode("")
	a.bulkMode = true
	if a.isAutoRefreshSafeState() {
		t.Error("bulk mode must be unsafe")
	}
}

func TestAutoRefreshShouldPoll(t *testing.T) {
	a := &App{}
	a.search.SetMode("")
	a.search.SetQuery("")
	if !a.shouldAutoRefreshPoll() {
		t.Error("plain inbox should poll")
	}
	a.search.SetMode("remote")
	if a.shouldAutoRefreshPoll() {
		t.Error("remote search must not poll")
	}
}

func TestPrependModelMath(t *testing.T) {
	// Existing model: cursor on "c" (index 2).
	ids := []string{"a", "b", "c"}
	selectedID := "c"
	newIDs := []string{"x", "y"} // newest-first

	gotIDs, gotIdx := prependIDsAndLocate(newIDs, ids, selectedID)
	wantIDs := []string{"x", "y", "a", "b", "c"}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("ids = %v, want %v", gotIDs, wantIDs)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("ids = %v, want %v", gotIDs, wantIDs)
		}
	}
	if gotIdx != 4 { // "c" moved from 2 to 4 (+len(newIDs))
		t.Errorf("selected index = %d, want 4", gotIdx)
	}

	// Selection not found => index 0 (top).
	_, idx := prependIDsAndLocate(newIDs, ids, "missing")
	if idx != 0 {
		t.Errorf("missing selection index = %d, want 0", idx)
	}
}
