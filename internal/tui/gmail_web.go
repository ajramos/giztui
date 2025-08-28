package tui

import (
	"fmt"
)

// openEmailInGmail opens the current email in Gmail web interface
func (a *App) openEmailInGmail() {
	// Use cached message ID (for undo functionality) with sync fallback
	messageID := a.GetCurrentMessageID()
	
	// CRITICAL DEBUG: Ensure cache is synchronized with cursor position
	if a.logger != nil {
		cursorID := a.getCurrentSelectedMessageID()
		a.logger.Printf("GMAIL WEB DEBUG: cached='%s', cursor='%s', match=%t", messageID, cursorID, messageID == cursorID)
		
		// If they don't match, sync the cached state
		if messageID != cursorID && cursorID != "" {
			a.logger.Printf("GMAIL WEB SYNC: Cached ID is stale, updating from cursor position")
			messageID = cursorID
			a.SetCurrentMessageID(messageID)
		}
	}
	
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
