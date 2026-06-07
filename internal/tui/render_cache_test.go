package tui

import "testing"

func TestRenderCache(t *testing.T) {
	a := &App{}
	if _, ok := a.getRenderCache("id1", true, 100); ok {
		t.Error("expected miss on empty cache")
	}
	a.setRenderCache("id1", true, 100, "rendered")
	got, ok := a.getRenderCache("id1", true, 100)
	if !ok || got != "rendered" {
		t.Errorf("hit failed: got %q ok=%v", got, ok)
	}
	if _, ok := a.getRenderCache("id1", true, 120); ok {
		t.Error("different width must miss")
	}
	if _, ok := a.getRenderCache("id1", false, 100); ok {
		t.Error("different mode must miss")
	}
}
