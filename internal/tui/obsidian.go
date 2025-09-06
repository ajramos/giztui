package tui

import (
	"fmt"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/obsidian"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// sendEmailToObsidian initiates the process of sending an email to Obsidian
func (a *App) sendEmailToObsidian() {
	if a.logger != nil {
		a.logger.Printf("=== sendEmailToObsidian called: bulkMode=%t, selected=%d ===", a.bulkMode, len(a.selected))
	}

	// Check for bulk mode first - but don't open panel here since keys.go handles it directly
	if a.bulkMode && len(a.selected) > 0 {
		if a.logger != nil {
			a.logger.Printf("Bulk mode detected with %d selected messages, but keys.go should handle this directly", len(a.selected))
		}
		// Don't call openBulkObsidianPanel here to avoid double opening
		// The bulk mode is handled directly in keys.go
		return
	}

	// Single message logic - use cached message ID (for undo functionality) with sync fallback
	messageID := a.GetCurrentMessageID()

	// Ensure cache is synchronized with cursor position
	if a.logger != nil {
		cursorID := a.getCurrentSelectedMessageID()
		// If they don't match, sync the cached state
		if messageID != cursorID && cursorID != "" {
			messageID = cursorID
			a.SetCurrentMessageID(messageID)
		}
	}

	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Show loading state immediately (this should be instant)
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, "Opening Obsidian panel...")
	}()

	// Load message content in background
	go func() {
		message, err := a.Client.GetMessageWithContent(messageID)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, "Failed to load message content")
			return
		}

		// Clear loading and open panel with explicit UI update
		a.GetErrorHandler().ClearProgress()
		a.QueueUpdateDraw(func() {
			a.openObsidianIngestPanel(message)
		})
	}()
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
	bgColor := a.GetComponentColors("obsidian").Background.Color()
	container.SetBackgroundColor(bgColor)
	container.SetBorder(true)
	container.SetTitle(" 📥 Send to Obsidian ")
	container.SetTitleColor(a.GetComponentColors("obsidian").Title.Color())

	// Show the configurable template
	templateContent := a.getObsidianTemplate()
	templateView := tview.NewTextView().
		SetText(templateContent).
		SetScrollable(true).
		SetWordWrap(true).
		SetBorder(false)

	// Set background on child components as well
	templateView.SetBackgroundColor(bgColor)

	// Create form for input fields including repopack checkbox
	form := tview.NewForm()
	form.SetBackgroundColor(bgColor)
	form.SetBorder(false)
	form.SetFieldBackgroundColor(a.GetComponentColors("obsidian").Background.Color())
	form.SetFieldTextColor(a.GetComponentColors("obsidian").Text.Color())
	form.SetLabelColor(a.GetComponentColors("obsidian").Title.Color())
	form.SetButtonBackgroundColor(a.GetComponentColors("obsidian").Background.Color())
	form.SetButtonTextColor(a.GetComponentColors("obsidian").Text.Color())

	// Variables to capture form data
	var comment string
	var repopackMode bool

	// Add comment input field
	form.AddInputField("💬 Pre-message:", "", 50, nil, func(text string) {
		comment = text
	})

	// Note: Single message mode doesn't need repopack checkbox - only individual file ingestion makes sense

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignRight)
	instructions.SetText("Enter to ingest | Esc to cancel")
	instructions.SetTextColor(a.GetComponentColors("obsidian").Text.Color())
	instructions.SetBackgroundColor(bgColor)

	// Add items to container with proper proportions
	container.AddItem(templateView, 0, 1, false) // Template takes most space
	container.AddItem(form, 3, 0, false)         // Form with input only (reduced from 4)
	container.AddItem(instructions, 1, 0, false) // Instructions take minimal space

	// Add to content split like prompts
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)

		if a.logger != nil {
			a.logger.Printf("Obsidian panel added to contentSplit successfully")
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("ERROR: Failed to find contentSplit view for Obsidian panel")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Failed to find contentSplit view")
		return
	}

	// Configure form navigation and submission
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			a.closeObsidianPanel()
			return nil
		case tcell.KeyEnter:
			// Perform ingestion with form data (repopackMode will always be false for single message)
			go a.performObsidianIngest(message, accountEmail, "default", comment, repopackMode)
			return nil
		}
		return event
	})

	// Configure Tab navigation between template view and form
	templateView.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyTab {
			a.SetFocus(form)
			return nil
		}
		if e.Key() == tcell.KeyEscape {
			a.closeObsidianPanel()
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

	// CRITICAL FIX: Set focus FIRST, then focus state (like working bulk prompt picker)
	a.SetFocus(form)
	a.currentFocus = "obsidian"
	a.updateFocusIndicators("obsidian")
	a.setActivePicker(PickerObsidian)

	if a.logger != nil {
		a.logger.Printf("Obsidian panel setup complete, focus set to form")
	}
}

// previewObsidianContent removed - no longer needed

// OBLITERATED: unused formatEmailPreview function eliminated! 💥

// performObsidianIngest performs the actual ingestion to Obsidian
func (a *App) performObsidianIngest(message *gmail.Message, accountEmail string, templateName string, comment string, repopackMode bool) {
	// Close panel immediately (like prompts do)
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.setActivePicker(PickerNone)
		// Restore focus to message list
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	})

	// Show progress
	a.GetErrorHandler().ShowProgress(a.ctx, "📝 Saving email to your notes...")

	// Get Obsidian service
	_, _, _, _, _, _, _, obsidianService, _, _, _, _ := a.GetServices()
	if obsidianService == nil {
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, "Obsidian service not available")
		return
	}

	// Create options for ingestion
	options := obsidian.ObsidianOptions{
		AccountEmail: accountEmail,
		RepopackMode: repopackMode,
		CustomMetadata: map[string]interface{}{
			"comment": comment,
		},
	}

	// Perform actual ingestion (for single messages, repopack mode is always false)
	var err error
	if repopackMode {
		// This shouldn't happen for single messages, but handle gracefully
		_, err = obsidianService.IngestEmailsToSingleFile(a.ctx, []*gmail.Message{message}, accountEmail, options)
	} else {
		_, err = obsidianService.IngestEmailToObsidian(a.ctx, message, options)
	}
	if err != nil {
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to ingest email: %v", err))
		return
	}

	// Clear progress and show success
	a.GetErrorHandler().ClearProgress()

	// Show user-friendly success message
	successMsg := "📝 Email saved to your notes!"
	if comment != "" {
		successMsg += " (with your comment)" // OBLITERATED: unnecessary fmt.Sprintf eliminated! 💥
	}

	a.GetErrorHandler().ShowSuccess(a.ctx, successMsg)
}

// Removed unused function: getObsidianVaultPath

// extractHeader is already defined in prompts.go

// OBLITERATED: unused showObsidianHistory function eliminated! 💥

// Removed unused function: closeModal

// getObsidianTemplate returns the configurable template from config
func (a *App) getObsidianTemplate() string {
	// TODO: [CONFIG] Load Obsidian template from user configuration file
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
		_, _, _, _, _, _, _, obsidianService, _, _, _, _ := a.GetServices()
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
			_, err = obsidianService.IngestEmailToObsidian(a.ctx, message, options)
			if err != nil {
				failed++
				continue
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

// OBLITERATED: unused sendSelectedBulkToObsidian function eliminated! 💥

// openBulkObsidianPanel shows the panel for bulk Obsidian ingestion
func (a *App) openBulkObsidianPanel() {
	if a.logger != nil {
		a.logger.Printf("=== openBulkObsidianPanel START ===")
	}

	if !a.bulkMode || len(a.selected) == 0 {
		if a.logger != nil {
			a.logger.Printf("ERROR: Invalid bulk mode state - bulkMode=%t, selected=%d", a.bulkMode, len(a.selected))
		}
		a.GetErrorHandler().ShowWarning(a.ctx, "No messages selected for bulk Obsidian ingestion")
		return
	}

	messageCount := len(a.selected)
	if a.logger != nil {
		a.logger.Printf("Processing bulk obsidian for %d messages", messageCount)
	}
	// Re-enabled now that double opening issue is fixed
	a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Preparing to send %d messages to Obsidian", messageCount))

	// Get account email
	if a.logger != nil {
		a.logger.Printf("Getting active account email...")
	}
	accountEmail := a.getActiveAccountEmail()
	if accountEmail == "" {
		if a.logger != nil {
			a.logger.Printf("ERROR: No account email available")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Account email not available")
		return
	}
	if a.logger != nil {
		a.logger.Printf("Account email: %s", accountEmail)
	}

	// Create panel similar to single message but for bulk
	if a.logger != nil {
		a.logger.Printf("Creating bulk obsidian UI container...")
	}
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(a.GetComponentColors("obsidian").Background.Color())
	container.SetBorder(true)
	container.SetTitle(fmt.Sprintf(" 📥 Send %d Messages to Obsidian ", messageCount))
	container.SetTitleColor(a.GetComponentColors("obsidian").Title.Color())

	if a.logger != nil {
		a.logger.Printf("Getting bulk template content...")
	}
	// Show bulk template info
	templateContent := a.getBulkObsidianTemplate(messageCount)
	templateView := tview.NewTextView().
		SetText(templateContent).
		SetScrollable(true).
		SetWordWrap(true).
		SetBorder(false)

	// Set background color for templateView
	templateView.SetBackgroundColor(a.GetComponentColors("obsidian").Background.Color())

	if a.logger != nil {
		a.logger.Printf("Creating bulk form with input fields...")
	}
	// Create form for bulk input fields including repopack checkbox
	bgColor := a.GetComponentColors("obsidian").Background.Color()
	form := tview.NewForm()
	form.SetBackgroundColor(bgColor)
	form.SetBorder(false)

	// Enhanced form theming to ensure visibility
	obsidianColors := a.GetComponentColors("obsidian")
	form.SetFieldBackgroundColor(obsidianColors.Background.Color())
	form.SetFieldTextColor(obsidianColors.Text.Color())
	form.SetLabelColor(obsidianColors.Title.Color())
	form.SetButtonBackgroundColor(obsidianColors.Background.Color())
	form.SetButtonTextColor(obsidianColors.Text.Color())

	if a.logger != nil {
		a.logger.Printf("Form colors - bg: %v, text: %v, label: %v", obsidianColors.Background.Color(), obsidianColors.Text.Color(), obsidianColors.Title.Color())
	}

	// Variables to capture form data
	var comment string
	var repopackMode bool

	if a.logger != nil {
		a.logger.Printf("Adding form fields (input and checkbox)...")
	}
	// Add comment input field
	form.AddInputField("💬 Bulk pre-message:", "", 50, nil, func(text string) {
		comment = text
	})
	if a.logger != nil {
		a.logger.Printf("Added input field: 'Bulk comment'")
	}

	// Add repopack checkbox (enabled for bulk mode)
	form.AddCheckbox("📦 Combined file:", false, func(label string, checked bool) {
		repopackMode = checked
	})
	if a.logger != nil {
		a.logger.Printf("Added checkbox: 'Combine into one file'")
	}

	if a.logger != nil {
		a.logger.Printf("Creating instructions view...")
	}
	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignRight)
	instructions.SetText("Enter to ingest | Esc to cancel")
	instructions.SetTextColor(a.GetComponentColors("obsidian").Text.Color())
	instructions.SetBackgroundColor(bgColor)

	if a.logger != nil {
		a.logger.Printf("Adding items to container...")
	}
	// Add items to container with proper proportions
	container.AddItem(templateView, 0, 1, false)
	container.AddItem(form, 5, 0, true) // Form with input and checkbox - MAKE FOCUSABLE
	container.AddItem(instructions, 1, 0, false)

	if a.logger != nil {
		a.logger.Printf("Adding container to contentSplit...")
	}
	// Add to content split
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			if a.logger != nil {
				a.logger.Printf("Removing existing labelsView...")
			}
			split.RemoveItem(a.labelsView)
		}
		if a.logger != nil {
			a.logger.Printf("Setting labelsView to new container...")
		}
		a.labelsView = container
		if a.logger != nil {
			a.logger.Printf("Adding container to split...")
		}
		split.AddItem(a.labelsView, 0, 1, true)
		if a.logger != nil {
			a.logger.Printf("Resizing container in split...")
		}
		split.ResizeItem(a.labelsView, 0, 1)

		if a.logger != nil {
			a.logger.Printf("Bulk Obsidian panel added to contentSplit successfully")
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("ERROR: Failed to find contentSplit view for bulk Obsidian panel")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Failed to find contentSplit view")
		return
	}

	if a.logger != nil {
		a.logger.Printf("Configuring form-level input capture with complete Tab containment (following advanced search pattern)...")
	}
	// Configure form-level navigation control (following advanced search pattern exactly)
	// This provides complete Tab containment and prevents focus escape

	// Form-level input capture handles ALL navigation and special keys
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if a.logger != nil {
			cur, _ := form.GetFocusedItemIndex()
			items := form.GetFormItemCount()
			a.logger.Printf("FORM key event: key=%v rune=%q focusIndex=%d/%d", ev.Key(), ev.Rune(), cur, items)
		}

		// Handle Tab/Shift+Tab with containment (following advanced search pattern)
		if ev.Key() == tcell.KeyTab || ev.Key() == tcell.KeyBacktab {
			cur, _ := form.GetFocusedItemIndex()
			items := form.GetFormItemCount() // Should be 2 (comment + checkbox)
			if items <= 0 {
				return ev
			}

			next := cur
			switch ev.Key() {
			case tcell.KeyTab:
				next = cur + 1
				if next >= items {
					next = 0 // Wrap to first element (comment)
				}
				if a.logger != nil {
					a.logger.Printf("TAB navigation: %d -> %d (wrapped=%v)", cur, next, cur+1 >= items)
				}
			case tcell.KeyBacktab:
				next = cur - 1
				if next < 0 {
					next = items - 1 // Wrap to last element (checkbox)
				}
				if a.logger != nil {
					a.logger.Printf("SHIFT+TAB navigation: %d -> %d (wrapped=%v)", cur, next, cur-1 < 0)
				}
			}

			form.SetFocus(next)
			return nil // Consume Tab - prevent escape to message list
		}

		// Handle Space key for checkbox toggle (form level like advanced search)
		if ev.Key() == tcell.KeyRune && ev.Rune() == ' ' {
			cur, _ := form.GetFocusedItemIndex()
			if a.logger != nil {
				a.logger.Printf("SPACE key pressed at focusIndex=%d", cur)
			}

			if cur == 1 { // Checkbox is at index 1
				if form.GetFormItemCount() >= 2 {
					if checkbox, ok := form.GetFormItem(1).(*tview.Checkbox); ok {
						repopackMode = !repopackMode
						checkbox.SetChecked(repopackMode)
						if a.logger != nil {
							a.logger.Printf("SPACE toggled checkbox: repopackMode=%v", repopackMode)
						}
						return nil // Consume space key
					} else {
						if a.logger != nil {
							a.logger.Printf("SPACE: failed to get checkbox at index 1")
						}
					}
				}
			} else {
				if a.logger != nil {
					a.logger.Printf("SPACE: not on checkbox (focusIndex=%d), let comment field handle", cur)
				}
			}
			// Let form handle space normally for comment field or fallback
			return ev
		}

		// Handle ESC at form level (primary handler)
		if ev.Key() == tcell.KeyEscape {
			if a.logger != nil {
				a.logger.Printf("ESC key pressed at form level - closing panel")
			}
			a.closeObsidianPanel()
			return nil
		}

		// Handle Enter submission
		if ev.Key() == tcell.KeyEnter {
			if a.logger != nil {
				a.logger.Printf("ENTER key pressed - submitting form")
			}
			go a.performBulkObsidianIngest(accountEmail, comment, repopackMode)
			return nil
		}

		// Let form handle all other keys normally
		return ev
	})

	if a.logger != nil {
		a.logger.Printf("Adding container-level ESC fallback handler for comprehensive coverage...")
	}
	// Add container-level ESC handling as failsafe (works if focus somehow escapes form)
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			if a.logger != nil {
				a.logger.Printf("ESC key pressed in container (fallback) - closing panel")
			}
			a.closeObsidianPanel()
			return nil
		}
		// Let form handle all other input
		return event
	})

	// CRITICAL FIX: Add synchronization delay to prevent white highlight and ensure proper theme colors
	if a.logger != nil {
		a.logger.Printf("Adding focus synchronization delay to prevent theme color issues...")
	}

	// Use QueueUpdateDraw to ensure UI is fully rendered before setting focus
	a.QueueUpdateDraw(func() {
		if a.logger != nil {
			a.logger.Printf("Setting focus state and theme colors synchronously...")
		}

		// Set focus state first to ensure proper theme application
		a.currentFocus = "obsidian"
		a.updateFocusIndicators("obsidian")
		a.setActivePicker(PickerObsidian)

		// Then set the actual focus - this should now have correct theme colors
		a.SetFocus(form)

		if a.logger != nil {
			a.logger.Printf("=== openBulkObsidianPanel FOCUS SET WITH PROPER THEME ===")
		}
	})

	if a.logger != nil {
		a.logger.Printf("=== openBulkObsidianPanel COMPLETE ===")
	}
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

// performBulkObsidianIngest performs bulk ingestion with repopack mode support
func (a *App) performBulkObsidianIngest(accountEmail, comment string, repopackMode bool) {
	// Close panel immediately
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.setActivePicker(PickerNone)
		// Restore focus to message list
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	})

	// Call the appropriate bulk function based on repopack mode
	if repopackMode {
		a.sendSelectedBulkToObsidianAsRepopack(accountEmail, comment)
	} else {
		a.sendSelectedBulkToObsidianWithComment(comment)
	}
}

// sendSelectedBulkToObsidianAsRepopack sends all selected messages to Obsidian as a single repopack file
func (a *App) sendSelectedBulkToObsidianAsRepopack(accountEmail, comment string) {
	if len(a.selected) == 0 {
		a.GetErrorHandler().ShowError(a.ctx, "No messages selected")
		return
	}

	// Snapshot selection (following archiveSelectedBulk pattern)
	ids := make([]string, 0, len(a.selected))
	for id := range a.selected {
		ids = append(ids, id)
	}

	messageCount := len(ids)
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("📦 Creating repopack with %d emails…", messageCount))

	go func() {
		// Load all messages
		messages := make([]*gmail.Message, 0, len(ids))
		failedCount := 0

		for _, id := range ids {
			message, err := a.Client.GetMessageWithContent(id)
			if err != nil {
				failedCount++
				continue
			}
			messages = append(messages, message)
		}

		if len(messages) == 0 {
			a.GetErrorHandler().ShowError(a.ctx, "Failed to load any messages for repopack")
			return
		}

		// Show progress update
		actualCount := len(messages)
		if failedCount > 0 {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("📦 Creating repopack with %d emails (%d failed to load)…", actualCount, failedCount))
		} else {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("📦 Creating repopack with %d emails…", actualCount))
		}

		// Get Obsidian service
		_, _, _, _, _, _, _, obsidianService, _, _, _, _ := a.GetServices()
		if obsidianService == nil {
			a.GetErrorHandler().ShowError(a.ctx, "Obsidian service not available")
			return
		}

		// Create options for repopack ingestion
		options := obsidian.ObsidianOptions{
			AccountEmail: accountEmail,
			RepopackMode: true,
			CustomMetadata: map[string]interface{}{
				"comment":        comment,
				"bulk_operation": true,
				"repopack":       true,
				"message_count":  actualCount,
			},
		}

		// Perform repopack ingestion
		result, err := obsidianService.IngestEmailsToSingleFile(a.ctx, messages, accountEmail, options)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to create repopack: %v", err))
			return
		}

		if !result.Success {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Repopack creation failed: %s", result.ErrorMessage))
			return
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

		// Clear progress and show success
		a.GetErrorHandler().ClearProgress()
		successMsg := fmt.Sprintf("📦 Repopack created with %d messages!", actualCount)
		if comment != "" {
			successMsg += " (with your comment)"
		}
		if failedCount > 0 {
			successMsg += fmt.Sprintf(" (%d messages failed to load)", failedCount)
		}
		a.GetErrorHandler().ShowSuccess(a.ctx, successMsg)
	}()
}

// closeObsidianPanel closes the Obsidian ingestion panel
func (a *App) closeObsidianPanel() {
	if a.logger != nil {
		a.logger.Printf("=== closeObsidianPanel: Starting cleanup ===")
	}

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
		if a.logger != nil {
			a.logger.Printf("closeObsidianPanel: Resized labelsView to hide panel")
		}
	}

	a.setActivePicker(PickerNone)
	if a.logger != nil {
		a.logger.Printf("closeObsidianPanel: Set active picker to None")
	}

	// Enhanced focus restoration with proper theme colors
	if a.logger != nil {
		a.logger.Printf("closeObsidianPanel: Restoring focus to message list with proper theme")
	}
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
	a.SetFocus(a.views["list"])

	if a.logger != nil {
		a.logger.Printf("=== closeObsidianPanel: Cleanup complete ===")
	}
}

// openBulkObsidianPanelWithRepack opens the bulk Obsidian panel with repack mode pre-selected
func (a *App) openBulkObsidianPanelWithRepack() {
	if !a.bulkMode || len(a.selected) == 0 {
		a.GetErrorHandler().ShowWarning(a.ctx, "No messages selected for bulk Obsidian repack")
		return
	}

	messageCount := len(a.selected)
	a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Preparing to create repopack with %d messages", messageCount))

	// Get account email
	accountEmail := a.getActiveAccountEmail()
	if accountEmail == "" {
		a.GetErrorHandler().ShowError(a.ctx, "Account email not available")
		return
	}

	// Create panel similar to bulk mode
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(a.GetComponentColors("obsidian").Background.Color())
	container.SetBorder(true)
	container.SetTitle(fmt.Sprintf(" 📦 Create Repopack with %d Messages ", messageCount))
	container.SetTitleColor(a.GetComponentColors("obsidian").Title.Color())

	// Show repack template info
	templateContent := a.getRepackObsidianTemplate(messageCount)
	templateView := tview.NewTextView().
		SetText(templateContent).
		SetScrollable(true).
		SetWordWrap(true).
		SetBorder(false)

	// Create form for repack input fields
	bgColor := a.GetComponentColors("obsidian").Background.Color()
	form := tview.NewForm()
	form.SetBackgroundColor(bgColor)
	form.SetBorder(false)
	form.SetFieldBackgroundColor(a.GetComponentColors("obsidian").Background.Color())
	form.SetFieldTextColor(a.GetComponentColors("obsidian").Text.Color())
	form.SetLabelColor(a.GetComponentColors("obsidian").Title.Color())
	form.SetButtonBackgroundColor(a.GetComponentColors("obsidian").Background.Color())
	form.SetButtonTextColor(a.GetComponentColors("obsidian").Text.Color())

	// Variables to capture form data
	var comment string

	// Add comment input field
	form.AddInputField("💬 Repack comment:", "", 50, nil, func(text string) {
		comment = text
	})

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter to create repopack | Esc to cancel")
	instructions.SetTextColor(a.GetComponentColors("obsidian").Text.Color())
	instructions.SetBackgroundColor(bgColor)

	// Add items to container with proper proportions
	container.AddItem(templateView, 0, 1, false)
	container.AddItem(form, 3, 0, false) // Form with input
	container.AddItem(instructions, 1, 0, false)

	// Add to content split
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)

		if a.logger != nil {
			a.logger.Printf("Repack Obsidian panel added to contentSplit successfully")
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("ERROR: Failed to find contentSplit view for repack Obsidian panel")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Failed to find contentSplit view")
		return
	}

	// Configure form navigation and submission
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			a.closeObsidianPanel()
			return nil
		case tcell.KeyEnter:
			// Perform repack ingestion directly
			go a.performBulkObsidianIngest(accountEmail, comment, true) // true = repopack mode
			return nil
		}
		return event
	})

	// Container-level input capture for Escape
	container.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.closeObsidianPanel()
			return nil
		}
		return e
	})

	// CRITICAL FIX: Set focus FIRST, then focus state (like working bulk prompt picker)
	a.SetFocus(form)
	a.currentFocus = "obsidian"
	a.updateFocusIndicators("obsidian")
	a.setActivePicker(PickerObsidian)

	if a.logger != nil {
		a.logger.Printf("Repack Obsidian panel setup complete, focus set to form")
	}
}

// getRepackObsidianTemplate returns template info for repack operations
func (a *App) getRepackObsidianTemplate(messageCount int) string {
	return fmt.Sprintf(`📦 OBSIDIAN REPOPACK MODE

Selected: %d messages for repopack creation

All emails will be combined into a single Markdown file using the repopack template.

Repopack includes:
• Frontmatter with metadata
• Individual email sections with headers
• Subject, From, To, CC, Date for each email
• Complete email body content
• Your repack comment
• Compilation timestamp

File will be created in your Obsidian vault's 00-Inbox folder.

Press Enter to create repopack or Esc to cancel.`, messageCount)
}
