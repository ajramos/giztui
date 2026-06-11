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
