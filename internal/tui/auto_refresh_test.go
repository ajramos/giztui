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
	a.searchMode = ""
	a.bulkMode = false
	if !a.isAutoRefreshSafeState() {
		t.Error("plain inbox with nothing open should be safe")
	}
	a.currentActivePicker = PickerLabels
	if a.isAutoRefreshSafeState() {
		t.Error("open picker must be unsafe")
	}
	a.currentActivePicker = PickerNone
	a.searchMode = "remote"
	if a.isAutoRefreshSafeState() {
		t.Error("search mode must be unsafe")
	}
	a.searchMode = ""
	a.bulkMode = true
	if a.isAutoRefreshSafeState() {
		t.Error("bulk mode must be unsafe")
	}
}

func TestAutoRefreshShouldPoll(t *testing.T) {
	a := &App{}
	a.searchMode = ""
	a.currentQuery = ""
	if !a.shouldAutoRefreshPoll() {
		t.Error("plain inbox should poll")
	}
	a.searchMode = "remote"
	if a.shouldAutoRefreshPoll() {
		t.Error("remote search must not poll")
	}
}
