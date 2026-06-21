package tui

import (
	"fmt"
	"testing"
)

func TestRenderCache(t *testing.T) {
	a := &App{caches: newAppCaches()}
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

func TestRenderCacheIsBounded(t *testing.T) {
	a := &App{caches: newAppCaches()}
	for i := 0; i < renderCacheMaxEntries*2; i++ {
		a.setRenderCache(fmt.Sprintf("id%d", i), true, 100, "x")
	}
	if got := a.caches.renderLen(); got > renderCacheMaxEntries {
		t.Errorf("cache unbounded: %d entries, cap %d", got, renderCacheMaxEntries)
	}
}
