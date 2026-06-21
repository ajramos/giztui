package tui

import "testing"

func TestUILifecycle_Defaults(t *testing.T) {
	var lc uiLifecycle

	if lc.ready.Load() {
		t.Errorf("expected ready to default to false, got true")
	}
	if lc.welcomeAnimating.Load() {
		t.Errorf("expected welcomeAnimating to default to false, got true")
	}

	lc.ready.Store(true)
	if !lc.ready.Load() {
		t.Errorf("expected ready to be true after Store(true), got false")
	}
}
