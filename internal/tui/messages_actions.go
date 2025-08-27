package tui

import (
	"fmt"
)

// archiveSelected archives the selected message
func (a *App) archiveSelected() {
	// Use helper function to get correct message ID
	messageID := a.getCurrentSelectedMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}

	if a.currentFocus == "list" || a.currentFocus == "text" || a.currentFocus == "summary" {
		// messageID is already correctly determined above
	} else {
		a.showError("❌ Unknown focus state")
		return
	}
	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}

	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Archive message using EmailService for undo support
	emailService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
	if err := emailService.ArchiveMessage(a.ctx, messageID); err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Error archiving message: %v", err))
		return
	}
	go func() {
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("📥 Archived: %s", subject))
	}()

	// Safe UI removal (preselect another index before removing)
	a.QueueUpdateDraw(func() { a.safeRemoveCurrentSelection(messageID) })
}

// trashSelectedByID moves a specific message to trash by ID
func (a *App) trashSelectedByID(messageID string) {
	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: trashSelectedByID ENTRY - messageID: %s", messageID)
	}

	if messageID == "" {
		if a.logger != nil {
			a.logger.Printf("HANG DEBUG: messageID empty, calling showError")
		}
		a.showError("❌ Invalid message ID")
		if a.logger != nil {
			a.logger.Printf("HANG DEBUG: returned from showError")
		}
		return
	}

	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: About to call Client.GetMessage")
	}
	// Get the current message to show confirmation
	message, err := a.Client.GetMessage(messageID)
	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: Returned from Client.GetMessage, err: %v", err)
	}
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("HANG DEBUG: GetMessage error, calling showError")
		}
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		if a.logger != nil {
			a.logger.Printf("HANG DEBUG: returned from showError after GetMessage error")
		}
		return
	}

	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: About to extract subject from headers")
	}
	// Extract subject for confirmation
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: Extracted subject: %s", subject)
		a.logger.Printf("HANG DEBUG: About to call EmailService.TrashMessage")
	}
	// Move message to trash using EmailService for undo support
	emailService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
	err = emailService.TrashMessage(a.ctx, messageID)
	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: Returned from EmailService.TrashMessage, err: %v", err)
	}
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("HANG DEBUG: TrashMessage error, calling showError")
		}
		a.showError(fmt.Sprintf("❌ Error moving to trash: %v", err))
		if a.logger != nil {
			a.logger.Printf("HANG DEBUG: returned from showError after TrashMessage error")
		}
		return
	}

	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: TrashMessage successful, about to call showStatusMessage")
	}

	// Show success message
	go func() {
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("🗑️ Moved to trash: %s", subject))
	}()

	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: Returned from showStatusMessage, about to call QueueUpdateDraw")
	}

	// Remove the message from the list and adjust selection (UI thread)
	a.QueueUpdateDraw(func() { a.safeRemoveCurrentSelection(messageID) })

	if a.logger != nil {
		a.logger.Printf("HANG DEBUG: Returned from QueueUpdateDraw, trashSelectedByID COMPLETE")
	}
}

// trashSelected moves the selected message to trash
func (a *App) trashSelected() {
	// Use helper function to get correct message ID
	messageID := a.getCurrentSelectedMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}

	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}

	// Get the current message to show confirmation
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}

	// Extract subject for confirmation
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Move message to trash using EmailService for undo support
	emailService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
	err = emailService.TrashMessage(a.ctx, messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error moving to trash: %v", err))
		return
	}

	// Show success message
	go func() {
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("🗑️ Moved to trash: %s", subject))
	}()

	// Remove the message from the list and adjust selection (UI thread)
	a.QueueUpdateDraw(func() { a.safeRemoveCurrentSelection(messageID) })
}
