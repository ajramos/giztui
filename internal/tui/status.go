package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/derailed/tview"
)

// showStatusMessage displays a transient message in the status bar
func (a *App) showStatusMessage(msg string) {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(fmt.Sprintf("%s | %s", a.statusBaseline(), msg))
		go func() {
			time.Sleep(3 * time.Second)
			a.QueueUpdateDraw(func() {
				if status, ok := a.views["status"].(*tview.TextView); ok {
					status.SetText(a.statusBaseline())
				}
			})
		}()
	}
}

// setStatusPersistent sets the status bar text without auto-clearing
func (a *App) setStatusPersistent(msg string) {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(fmt.Sprintf("%s | %s", a.statusBaseline(), msg))
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

// showSuccess shows a success message via status helpers
func (a *App) showSuccess(msg string) {
	a.showStatusMessage(fmt.Sprintf("✅ %s", msg))
}

// showLLMError logs the full error and shows a concise message in the status bar
func (a *App) showLLMError(operation string, err error) {
	if err == nil {
		return
	}
	// Log full detail for debugging
	if a.logger != nil {
		a.logger.Printf("LLM error during %s: %v", operation, err)
	}
	// Show concise error to the user
	a.showStatusMessage(fmt.Sprintf("⚠️ LLM error (%s): %s", operation, a.shortError(err, 180)))
}

// shortError returns a single-line, length-limited error string
func (a *App) shortError(err error, max int) string {
	if err == nil {
		return ""
	}
	s := strings.TrimSpace(err.Error())
	// Replace newlines and tabs to keep status bar clean
	s = strings.ReplaceAll(s, "\n", " | ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	if max <= 0 {
		max = 180
	}
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}

// statusBaseline returns the baseline status text including persistent indicators
func (a *App) statusBaseline() string {
	base := "GizTUI"
	// Append active account email if available (non-blocking; never call network here)
	if a != nil && strings.TrimSpace(a.welcomeEmail) != "" {
		base += " | " + a.welcomeEmail
	}
	if a != nil && a.llmTouchUpEnabled {
		base += " | 🧠"
	} else {
		base += " | 🧾"
	}
	
	// Check if composition panel is active and show context-appropriate message
	if a != nil && a.compositionPanel != nil {
		// Check if we're on the composition page
		if currentPage, _ := a.Pages.GetFrontPage(); currentPage == "compose_with_status" {
			return base + " | Press ? for help | Email composer view"
		}
		// Fallback to original visibility check
		if a.compositionPanel.IsVisible() {
			return base + " | Press ? for help | Email composer view"
		}
	}
	
	// Default baseline message
	return base + " | Press ? for help"
}
