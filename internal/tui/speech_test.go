package tui

import (
	"testing"

	"github.com/derailed/tview"
)

func TestFocusedSpeakText(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	tv := tview.NewTextView().SetText("read me aloud")
	a.SetFocus(tv)
	if got := a.focusedSpeakText(); got != "read me aloud" {
		t.Fatalf("focused TextView text = %q, want 'read me aloud'", got)
	}

	a.SetFocus(tview.NewList())
	if got := a.focusedSpeakText(); got != "" {
		t.Fatalf("non-text focus should yield empty, got %q", got)
	}
}
