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
	container.SetTitleColor(tcell.ColorYellow)

	// Show the configurable template
	templateContent := a.getObsidianTemplate()
	templateView := tview.NewTextView().
		SetText(templateContent).
		SetScrollable(true).
		SetWordWrap(true).
		SetBorder(false)

	// Comment input field
	commentLabel := tview.NewTextView().SetText("💬 Pre-message:")
	commentLabel.SetTextColor(tcell.ColorYellow)

	commentInput := tview.NewInputField()
	commentInput.SetLabel("")
	commentInput.SetText("")
	commentInput.SetPlaceholder("Add a personal note about this email...")
	commentInput.SetFieldWidth(50)
	commentInput.SetBorder(false)                         // No border for cleaner look
	commentInput.SetFieldBackgroundColor(tcell.ColorBlue) // Blue background when focused
	commentInput.SetFieldTextColor(tcell.ColorDarkGreen)  // Dark green text like in the image

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter to ingest | Esc to cancel")
	instructions.SetTextColor(tcell.ColorGray)

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
	})

	// Show progress
	a.GetErrorHandler().ShowProgress(a.ctx, "Ingesting email to Obsidian...")

	// Add debug logging
	if a.logger != nil {
		a.logger.Printf("DEBUG: performObsidianIngest called with message ID: %s", message.Id)
		a.logger.Printf("DEBUG: Account email: %s", accountEmail)
		a.logger.Printf("DEBUG: Comment: %s", comment)
	}

	// Get Obsidian service
	_, _, _, _, _, _, obsidianService := a.GetServices()
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

	// Show detailed success message
	successMsg := fmt.Sprintf("Email ingested successfully!")
	if comment != "" {
		successMsg += fmt.Sprintf("\n💬 Comment: %s", comment)
	}
	if result != nil && result.FilePath != "" {
		successMsg += fmt.Sprintf("\n📁 File: %s", result.FilePath)
	}
	successMsg += "\n📁 Check your Obsidian vault"

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
	_, _, _, _, _, _, obsidianService := a.GetServices()
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

// closeObsidianPanel closes the Obsidian ingestion panel
func (a *App) closeObsidianPanel() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.labelsVisible = false
	a.restoreFocusAfterModal()
}
