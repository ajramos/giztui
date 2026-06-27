package tui

import (
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/derailed/tview"
)

// generateHelpText drifted behind shipped features: "M" was mislabeled as
// "Export as Markdown" (it toggles Markdown rendering), and the touch-up,
// accounts, and inbox Action Plan features were undocumented. This pins the
// corrected content so the help can't silently fall behind again.
func TestGenerateHelpText_CoversRecentFeatures(t *testing.T) {
	app := &App{
		Application: tview.NewApplication(),
		Config:      &config.Config{},
		bulk:        newBulkState(),
		Keys: config.KeyBindings{
			Help:       "?",
			Markdown:   "M",
			Accounts:   "ctrl+a",
			ActionPlan: "P",
		},
	}

	help := app.generateHelpText()

	if strings.Contains(help, "Export as Markdown") {
		t.Error("help still mislabels M as 'Export as Markdown'")
	}
	for _, want := range []string{
		"Toggle Markdown rendering",
		":touch-up",
		":accounts",
		":action-plan",
		"Account picker",
		"Action Plan",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help is missing %q", want)
		}
	}
}
