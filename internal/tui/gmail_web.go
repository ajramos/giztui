package tui

import (
	"fmt"
)

// openEmailInGmail opens the current email in Gmail web interface
func (a *App) openEmailInGmail() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Get Gmail web service
	_, _, _, _, _, _, _, _, gmailWebService, _, _ := a.GetServices()
	if gmailWebService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Gmail web service not available")
		return
	}

	if err := gmailWebService.OpenMessageInWeb(a.ctx, messageID); err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to open in Gmail: %v", err))
		return
	}

	a.GetErrorHandler().ShowSuccess(a.ctx, "Opening message in Gmail web UI")
}
