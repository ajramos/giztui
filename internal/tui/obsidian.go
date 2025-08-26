package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/obsidian"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// sendEmailToObsidian initiates the process of sending an email to Obsidian
func (a *App) sendEmailToObsidian() {
	// Check for bulk mode first (following the established pattern)
	if a.bulkMode && len(a.selected) > 0 {
		go a.openBulkObsidianPanel()
		return
	}

	// Single message logic
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Load message content directly from Gmail client
	message, err := a.Client.GetMessageWithContent(messageID)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Failed to load message content")
		return
	}

	// Show success message for debugging
	a.GetErrorHandler().ShowInfo(a.ctx, "Opening Obsidian panel...")

	// Show options panel
	go a.openObsidianIngestPanel(message)
}

// openObsidianIngestPanel shows the panel for Obsidian ingestion
func (a *App) openObsidianIngestPanel(message *gmail.Message) {
	// Get account email
	accountEmail := a.getActiveAccountEmail()
	if accountEmail == "" {
		a.GetErrorHandler().ShowError(a.ctx, "Account email not available")
		return
	}

	// Create panel similar to prompt picker
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	container.SetBorder(true)
	container.SetTitle(" 📥 Send to Obsidian ")
	container.SetTitleColor(a.getTitleColor())

	// Show the configurable template
	templateContent := a.getObsidianTemplate()
	templateView := tview.NewTextView().
		SetText(templateContent).
		SetScrollable(true).
		SetWordWrap(true).
		SetBorder(false)

	// Comment input field
	commentLabel := tview.NewTextView().SetText("💬 Pre-message:")
	commentLabel.SetTextColor(a.getTitleColor())

	commentInput := tview.NewInputField()
	commentInput.SetLabel("")
	commentInput.SetText("")
	commentInput.SetPlaceholder("Add a personal note about this email...")
	commentInput.SetFieldWidth(50)
	commentInput.SetBorder(false)                         // No border for cleaner look
	commentInput.SetFieldBackgroundColor(a.GetComponentColors("obsidian").Background.Color()) // Component background (not accent)
	commentInput.SetFieldTextColor(a.GetComponentColors("obsidian").Text.Color())  // Component text color
	commentInput.SetPlaceholderTextColor(a.getHintColor())                         // Consistent placeholder color

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter to ingest | Esc to cancel")
	instructions.SetTextColor(a.getFooterColor())

	// Create a horizontal flex for label and input alignment
	commentRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	commentRow.AddItem(commentLabel, 0, 1, false)
	commentRow.AddItem(commentInput, 0, 1, false)

	// Add items to container with proper proportions
	container.AddItem(templateView, 0, 1, false) // Template takes most space
	container.AddItem(commentRow, 2, 0, false)   // Label and input in same row
	container.AddItem(instructions, 1, 0, false) // Instructions take minimal space

	// Add to content split like prompts
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)

		// Debug: Show success message
		a.GetErrorHandler().ShowSuccess(a.ctx, "Obsidian panel opened successfully!")
	} else {
		// Debug: Show error if split not found
		a.GetErrorHandler().ShowError(a.ctx, "Failed to find contentSplit view")
	}

	// Set focus and state
	a.currentFocus = "obsidian"
	a.updateFocusIndicators("obsidian")
	a.labelsVisible = true

	// Configure Tab navigation between template view and comment input
	templateView.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyTab {
			a.SetFocus(commentInput)
			return nil
		}
		return e
	})

	commentInput.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyTab {
			a.SetFocus(templateView)
			return nil
		}
		if e.Key() == tcell.KeyEscape {
			a.closeObsidianPanel()
			return nil
		}
		if e.Key() == tcell.KeyEnter {
			// Get comment text
			comment := commentInput.GetText()
			// Perform ingestion with comment
			go a.performObsidianIngest(message, accountEmail, "default", comment)
			return nil
		}
		return e
	})

	// Container-level input capture for Escape
	container.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.closeObsidianPanel()
			return nil
		}
		return e
	})

	// Set focus immediately and force redraw
	a.currentFocus = "obsidian"
	a.updateFocusIndicators("obsidian")
	a.labelsVisible = true // Needed for proper visual state

	// Force focus with multiple attempts
	a.SetFocus(commentInput)
	a.QueueUpdateDraw(func() {
		a.SetFocus(commentInput)
	})

	// Additional focus attempt after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		a.QueueUpdateDraw(func() {
			a.SetFocus(commentInput)
		})
	}()
}

// previewObsidianContent removed - no longer needed

// formatEmailPreview formats email content for preview
func (a *App) formatEmailPreview(message *gmail.Message) string {
	// Simple template for preview
	template := `---
title: "{{subject}}"
date: {{date}}
from: {{from}}
type: email
status: inbox
---

# {{subject}}

**From:** {{from}}  
**Date:** {{date}}  
**Labels:** {{labels}}

---

{{body}}

---

*Ingested from Gmail on {{ingest_date}}*`

	// Extract message content
	body := message.PlainText
	if body == "" && message.Snippet != "" {
		body = message.Snippet
	}

	// Truncate if too long
	if len([]rune(body)) > 8000 {
		body = string([]rune(body)[:8000])
	}

	// Prepare variables for substitution
	variables := map[string]string{
		"subject":     message.Subject,
		"from":        a.extractHeader(message, "From"),
		"to":          a.extractHeader(message, "To"),
		"cc":          a.extractHeader(message, "Cc"),
		"date":        a.extractHeader(message, "Date"),
		"body":        body,
		"labels":      strings.Join(message.LabelIds, ", "),
		"message_id":  message.Id,
		"ingest_date": "2024-01-15 15:04:05", // Placeholder for preview
	}

	// Replace variables in template
	content := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content
}

// performObsidianIngest performs the actual ingestion to Obsidian
func (a *App) performObsidianIngest(message *gmail.Message, accountEmail string, templateName string, comment string) {
	// Close panel immediately (like prompts do)
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.labelsVisible = false
		// Restore focus to message list
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	})

	// Show progress
	a.GetErrorHandler().ShowProgress(a.ctx, "📝 Saving email to your notes...")

	// Add debug logging
	if a.logger != nil {
		a.logger.Printf("DEBUG: performObsidianIngest called with message ID: %s", message.Id)
		a.logger.Printf("DEBUG: Account email: %s", accountEmail)
		a.logger.Printf("DEBUG: Comment: %s", comment)
	}

	// Get Obsidian service
	_, _, _, _, _, _, obsidianService, _, _, _, _ := a.GetServices()
	if obsidianService == nil {
		if a.logger != nil {
			a.logger.Printf("DEBUG: Obsidian service is nil!")
		}
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, "Obsidian service not available")
		return
	}

	if a.logger != nil {
		a.logger.Printf("DEBUG: Obsidian service obtained successfully")
	}

	// Create options for ingestion
	options := obsidian.ObsidianOptions{
		AccountEmail: accountEmail,
		CustomMetadata: map[string]interface{}{
			"comment": comment,
		},
	}

	// Perform actual ingestion
	if a.logger != nil {
		a.logger.Printf("DEBUG: Calling obsidianService.IngestEmailToObsidian...")
	}
	result, err := obsidianService.IngestEmailToObsidian(a.ctx, message, options)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("DEBUG: Ingestion failed with error: %v", err)
		}
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to ingest email: %v", err))
		return
	}

	if a.logger != nil {
		a.logger.Printf("DEBUG: Ingestion successful, result: %+v", result)
	}

	// Clear progress and show success
	a.GetErrorHandler().ClearProgress()

	// Show user-friendly success message
	successMsg := "📝 Email saved to your notes!"
	if comment != "" {
		successMsg += fmt.Sprintf(" (with your comment)")
	}

	a.GetErrorHandler().ShowSuccess(a.ctx, successMsg)
}

// getObsidianVaultPath returns the configured Obsidian vault path
func (a *App) getObsidianVaultPath() string {
	// For now, return a default path
	// TODO: Get from configuration
	return "/Users/ajramos/Documents/ObsidianVault"
}

// extractHeader is already defined in prompts.go

// showObsidianHistory shows the history of Obsidian forwards
func (a *App) showObsidianHistory() {
	// Get Obsidian service
	_, _, _, _, _, _, obsidianService, _, _, _, _ := a.GetServices()
	if obsidianService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Obsidian service not available")
		return
	}

	// Get account email
	accountEmail := a.getActiveAccountEmail()
	if accountEmail == "" {
		a.GetErrorHandler().ShowError(a.ctx, "Account email not available")
		return
	}

	// TODO: Implement history display
	// This would show a list of recent forwards with options to re-ingest or view details
	a.GetErrorHandler().ShowInfo(a.ctx, "Obsidian history feature coming soon")
}

// closeModal closes the current modal
func (a *App) closeModal() {
	// TODO: Implement modal closing
	// This would restore the previous view and focus
}

// getObsidianTemplate returns the configurable template from config
func (a *App) getObsidianTemplate() string {
	// TODO: Load from config when available
	// Simple template without message preview
	return `📧 OBSIDIAN INGESTION

This email will be ingested to Obsidian using the template configured in your config file.

Template includes:
• Subject, From, To, CC, Date
• Labels and Message ID
• Email body content
• Your personal comment (if provided)

Press Enter to ingest or Esc to cancel.`
}

// sendSelectedBulkToObsidianWithComment sends all selected messages to Obsidian with a comment
func (a *App) sendSelectedBulkToObsidianWithComment(comment string) {
	if len(a.selected) == 0 {
		return
	}

	// Snapshot selection (following archiveSelectedBulk pattern)
	ids := make([]string, 0, len(a.selected))
	for id := range a.selected {
		ids = append(ids, id)
	}

	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("📝 Saving %d emails to your notes…", len(ids)))
	go func() {
		failed := 0
		total := len(ids)

		// Get account email
		accountEmail := a.getActiveAccountEmail()
		if accountEmail == "" {
			a.GetErrorHandler().ShowError(a.ctx, "Account email not available")
			return
		}

		// Get Obsidian service
		_, _, _, _, _, _, obsidianService, _, _, _, _ := a.GetServices()
		if obsidianService == nil {
			a.GetErrorHandler().ShowError(a.ctx, "Obsidian service not available")
			return
		}

		// Process each message individually with progress updates (following bulk pattern)
		for i, id := range ids {
			// Load message content
			message, err := a.Client.GetMessageWithContent(id)
			if err != nil {
				failed++
				continue
			}

			// Progress update on UI thread (following archiveSelectedBulk pattern)
			idx := i + 1
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("📝 Saving email %d/%d…", idx, total))

			// Create options for ingestion
			options := obsidian.ObsidianOptions{
				AccountEmail: accountEmail,
				CustomMetadata: map[string]interface{}{
					"comment":        comment,
					"bulk_operation": true,
					"batch_index":    idx,
					"batch_total":    total,
				},
			}

			// Perform ingestion for this message
			result, err := obsidianService.IngestEmailToObsidian(a.ctx, message, options)
			if err != nil {
				failed++
				// Log the error for debugging
				if a.logger != nil {
					a.logger.Printf("Bulk Obsidian ingestion failed for message %s: %v", id, err)
				}
				continue
			}

			// Log successful ingestion for debugging
			if a.logger != nil && result != nil {
				a.logger.Printf("Bulk Obsidian ingestion successful for message %s: %s", id, result.FilePath)
			}
		}

		// Final UI update (following archiveSelectedBulk pattern)
		a.QueueUpdateDraw(func() {
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.refreshTableDisplay()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(a.getSelectionStyle())
			}
		})

		// ErrorHandler calls outside QueueUpdateDraw to avoid deadlock
		a.GetErrorHandler().ClearProgress()

		// Show final result
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, "All emails saved to your notes!")
		} else {
			successCount := total - failed
			if successCount > 0 {
				a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("%d emails saved (%d failed)", successCount, failed))
			} else {
				a.GetErrorHandler().ShowError(a.ctx, "Failed to save emails to notes")
			}
		}
	}()
}

// sendSelectedBulkToObsidian sends all selected messages to Obsidian (backward compatibility)
func (a *App) sendSelectedBulkToObsidian() {
	a.sendSelectedBulkToObsidianWithComment("")
}

// openBulkObsidianPanel shows the panel for bulk Obsidian ingestion
func (a *App) openBulkObsidianPanel() {
	if !a.bulkMode || len(a.selected) == 0 {
		a.GetErrorHandler().ShowWarning(a.ctx, "No messages selected for bulk Obsidian ingestion")
		return
	}

	messageCount := len(a.selected)
	a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Preparing to send %d messages to Obsidian", messageCount))

	// Get account email
	accountEmail := a.getActiveAccountEmail()
	if accountEmail == "" {
		a.GetErrorHandler().ShowError(a.ctx, "Account email not available")
		return
	}

	// Create panel similar to single message but for bulk
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	container.SetBorder(true)
	container.SetTitle(fmt.Sprintf(" 📥 Send %d Messages to Obsidian ", messageCount))
	container.SetTitleColor(a.getTitleColor())

	// Show bulk template info
	templateContent := a.getBulkObsidianTemplate(messageCount)
	templateView := tview.NewTextView().
		SetText(templateContent).
		SetScrollable(true).
		SetWordWrap(true).
		SetBorder(false)

	// Comment input field for bulk operation
	commentLabel := tview.NewTextView().SetText("💬 Bulk comment:")
	commentLabel.SetTextColor(a.getTitleColor())

	commentInput := tview.NewInputField()
	commentInput.SetLabel("")
	commentInput.SetText("")
	commentInput.SetPlaceholder("Add a note for all emails in this batch...")
	commentInput.SetFieldWidth(50)
	commentInput.SetBorder(false)
	commentInput.SetFieldBackgroundColor(a.GetComponentColors("obsidian").Background.Color()) // Component background (not accent)
	commentInput.SetFieldTextColor(a.GetComponentColors("obsidian").Text.Color())
	commentInput.SetPlaceholderTextColor(a.getHintColor())                         // Consistent placeholder color

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter to ingest all | Esc to cancel")
	instructions.SetTextColor(a.getFooterColor())

	// Create a horizontal flex for label and input alignment
	commentRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	commentRow.AddItem(commentLabel, 0, 1, false)
	commentRow.AddItem(commentInput, 0, 1, false)

	// Add items to container with proper proportions
	container.AddItem(templateView, 0, 1, false)
	container.AddItem(commentRow, 2, 0, false)
	container.AddItem(instructions, 1, 0, false)

	// Add to content split
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}

	// Set focus and state
	a.currentFocus = "obsidian"
	a.updateFocusIndicators("obsidian")
	a.labelsVisible = true

	// Configure input handling
	commentInput.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.closeObsidianPanel()
			return nil
		}
		if e.Key() == tcell.KeyEnter {
			// Get comment text
			comment := commentInput.GetText()
			// Perform bulk ingestion
			go a.performBulkObsidianIngest(accountEmail, comment)
			return nil
		}
		return e
	})

	// Container-level input capture for Escape
	container.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.closeObsidianPanel()
			return nil
		}
		return e
	})

	// Set focus to input
	a.SetFocus(commentInput)
}

// getBulkObsidianTemplate returns template info for bulk operations
func (a *App) getBulkObsidianTemplate(messageCount int) string {
	return fmt.Sprintf(`📧 BULK OBSIDIAN INGESTION

Selected: %d messages for bulk ingestion to Obsidian

Each email will be processed using the template configured in your config file.

Template includes:
• Subject, From, To, CC, Date
• Labels and Message ID  
• Email body content
• Your bulk comment (if provided)
• Batch metadata (index, total)

Files will be created in your Obsidian vault's 00-Inbox folder.

Press Enter to process all messages or Esc to cancel.`, messageCount)
}

// performBulkObsidianIngest performs bulk ingestion (wraps the existing bulk function)
func (a *App) performBulkObsidianIngest(accountEmail, comment string) {
	// Close panel immediately
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.labelsVisible = false
		// Restore focus to message list
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	})

	// Call the bulk function with the comment
	a.sendSelectedBulkToObsidianWithComment(comment)
}

// closeObsidianPanel closes the Obsidian ingestion panel
func (a *App) closeObsidianPanel() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.labelsVisible = false
	a.restoreFocusAfterModal()
}
