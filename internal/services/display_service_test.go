package services

import "testing"

func TestDisplayServiceMarkdownRendering(t *testing.T) {
	d := NewDisplayService(true)
	if !d.IsMarkdownRendering() {
		t.Error("default should be markdown on")
	}
	if got := d.ToggleMarkdownRendering(); got != false {
		t.Errorf("toggle returned %v, want false", got)
	}
	if d.IsMarkdownRendering() {
		t.Error("toggle did not turn off")
	}
	d.SetMarkdownRendering(true)
	if !d.IsMarkdownRendering() {
		t.Error("SetMarkdownRendering(true) failed")
	}
}
