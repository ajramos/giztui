package tui

import (
	"fmt"
	"time"

	"github.com/derailed/tview"
)

// showStatusMessage displays a transient message in the status bar
func (a *App) showStatusMessage(msg string) {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(fmt.Sprintf("Gmail TUI | %s | Press ? for help | Press q to quit", msg))
		go func() {
			time.Sleep(3 * time.Second)
			a.QueueUpdateDraw(func() {
				if status, ok := a.views["status"].(*tview.TextView); ok {
					status.SetText("Gmail TUI | Press ? for help | Press q to quit")
				}
			})
		}()
	}
}

// setStatusPersistent sets the status bar text without auto-clearing
func (a *App) setStatusPersistent(msg string) {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(fmt.Sprintf("Gmail TUI | %s | Press ? for help | Press q to quit", msg))
	}
}

// showError shows an error message via status helpers
func (a *App) showError(msg string) {
	a.showStatusMessage(msg)
}

// showInfo shows an info message via status helpers
func (a *App) showInfo(msg string) {
	a.showStatusMessage(fmt.Sprintf("ℹ️ %s", msg))
}
