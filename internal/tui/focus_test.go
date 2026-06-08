package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/derailed/tview"
)

// After a bulk action (archive/trash) completes, focus and the focus-indicator
// border must return to the message list so the active pane stays highlighted —
// without the user having to press Tab. focusList() centralizes that restoration.
func TestFocusListRestoresListFocus(t *testing.T) {
	app := &App{
		Application: tview.NewApplication(),
		Config:      &config.Config{},
		views: map[string]tview.Primitive{
			"list": tview.NewTable(),
		},
	}
	// Simulate the post-bulk state where focus is stale/elsewhere.
	app.currentFocus = "text"

	app.focusList()

	if app.currentFocus != "list" {
		t.Errorf("currentFocus = %q, want \"list\"", app.currentFocus)
	}
}
