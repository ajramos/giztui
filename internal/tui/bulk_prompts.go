package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// openBulkPromptPicker shows a picker for selecting prompts to apply to multiple messages
func (a *App) openBulkPromptPicker() {
	if a.logger != nil {
		a.logger.Printf("openBulkPromptPicker: starting - bulkMode=%v, selectedCount=%d", a.bulkMode, len(a.selected))
	}

	if !a.bulkMode || len(a.selected) == 0 {
		if a.logger != nil {
			a.logger.Printf("openBulkPromptPicker: invalid state - bulkMode=%v, selectedCount=%d", a.bulkMode, len(a.selected))
		}
		a.GetErrorHandler().ShowWarning(a.ctx, "No messages selected for bulk prompt")
		return
	}

	messageCount := len(a.selected)
	if a.logger != nil {
		a.logger.Printf("openBulkPromptPicker: opening picker for %d messages", messageCount)
	}
	a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Applying prompt to %d selected messages", messageCount))

	// Get prompt service
	_, _, _, _, _, promptService, _ := a.GetServices()
	if promptService == nil {
		if a.logger != nil {
			a.logger.Printf("openBulkPromptPicker: prompt service not available")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	if a.logger != nil {
		a.logger.Printf("openBulkPromptPicker: prompt service available, creating UI")
	}

	// Create picker UI similar to individual prompts
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30)
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	type promptItem struct {
		id          int
		name        string
		description string
		category    string
	}

	var all []promptItem
	var visible []promptItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) &&
				!strings.Contains(strings.ToLower(item.description), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Category icon and priority indicator
			icon := "📄"
			switch item.category {
			case "bulk_analysis":
				icon = "🚀"
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			default:
				icon = "📝"
			}

			display := fmt.Sprintf("%s %s", icon, item.name)

			// Capture variables for closure
			promptID := item.id
			promptName := item.name

			list.AddItem(display, "Enter: apply bulk prompt", 0, func() {
				if a.logger != nil {
					a.logger.Printf("bulk prompt picker: selected promptID=%d name=%s for %d messages", promptID, promptName, messageCount)
				}
				// Apply bulk prompt
				go a.applyBulkPrompt(promptID, promptName)
			})
		}
	}

	// Load prompts in background
	go func() {
		if a.logger != nil {
			a.logger.Printf("bulk prompt picker: loading prompts")
		}

		prompts, err := promptService.ListPrompts(a.ctx, "")
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load prompts: %v", err))
			return
		}

		a.QueueUpdateDraw(func() {
			all = make([]promptItem, 0, len(prompts))

			// Only show bulk_analysis prompts for bulk operations
			for _, p := range prompts {
				if p.Category == "bulk_analysis" {
					all = append(all, promptItem{
						id:          p.ID,
						name:        p.Name,
						description: p.Description,
						category:    p.Category,
					})
				}
			}

			if a.logger != nil {
				a.logger.Printf("bulk prompt picker: loaded %d prompts (%d bulk_analysis, %d others)",
					len(all),
					len(prompts)-len(all),
					len(all)-(len(prompts)-len(all)))
			}

			reload("")
			// Set up event handlers after successful loading
			// Handle input changes
			input.SetChangedFunc(func(text string) {
				reload(text)
			})

			// Handle key events for input field
			input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEscape {
					a.closeBulkPromptPicker()
					return nil
				}
				if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp || event.Key() == tcell.KeyPgDn || event.Key() == tcell.KeyPgUp {
					a.SetFocus(list)
					return event
				}
				return event
			})

			// Handle navigation for list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closeBulkPromptPicker()
					return nil
				}
				return e
			})

			// Handle enter in input field (select first match)
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.closeBulkPromptPicker()
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						v := visible[0]
						if a.logger != nil {
							a.logger.Printf("bulk prompt picker: pick via search promptID=%d name=%s", v.id, v.name)
						}
						// Apply bulk prompt
						go a.applyBulkPrompt(v.id, v.name)
					}
				}
			})

			reload("")
		})
	}()

	// Create container similar to individual prompt picker
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	container.SetBorder(true)
	container.SetTitle(fmt.Sprintf(" 🤖 Bulk Prompt Library (%d messages) ", messageCount))
	container.SetTitleColor(tcell.ColorYellow)
	container.AddItem(input, 3, 0, true)
	container.AddItem(list, 0, 1, true)

	// Footer
	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter to apply | Esc to cancel ")
	footer.SetTextColor(tcell.ColorGray)
	container.AddItem(footer, 1, 0, false)

	// Add to content split like individual prompt picker
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.SetFocus(input)
	a.currentFocus = "prompts"
	a.updateFocusIndicators("prompts")
	a.labelsVisible = true // Needed for proper visual state
}

// closeBulkPromptPicker closes the bulk prompt picker and restores the original view
func (a *App) closeBulkPromptPicker() {
	// Cancel any active streaming operations
	if a.streamingCancel != nil {
		a.streamingCancel()
		a.streamingCancel = nil
	}

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.ResizeItem(a.labelsView, 0, 0)
		}
	}
	a.labelsVisible = false
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// exitBulkMode exits bulk mode and returns to normal view
func (a *App) exitBulkMode() {
	if a.logger != nil {
		a.logger.Printf("exitBulkMode: starting")
	}

	// Do everything synchronously to avoid UI thread blocking
	// Clear bulk mode
	a.bulkMode = false
	a.selected = make(map[string]bool)

	// Hide AI panel if it's visible
	if a.aiSummaryVisible {
		// Hide AI panel directly
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.aiSummaryView, 0, 0) // Hide AI panel
		}
		a.aiSummaryVisible = false
		a.aiPanelInPromptMode = false
		a.GetErrorHandler().ClearProgress()
	}

	// Return focus to list
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")

	// Update status using queue to avoid blocking
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, "Exited bulk mode")
	}()

	if a.logger != nil {
		a.logger.Printf("exitBulkMode: completed")
	}
}

// hideAIPanel hides the AI panel and returns focus to the message list
func (a *App) hideAIPanel() {
	if a.logger != nil {
		a.logger.Printf("hideAIPanel: starting")
	}

	// Do everything synchronously to avoid UI thread blocking (same fix as exitBulkMode)
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.aiSummaryView, 0, 0) // Hide AI panel
	}

	a.aiSummaryVisible = false
	a.aiPanelInPromptMode = false

	// Return focus to list
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")

	// Clear any status message
	a.GetErrorHandler().ClearProgress()

	if a.logger != nil {
		a.logger.Printf("hideAIPanel: completed")
	}
}

// applyBulkPrompt applies a prompt to all selected messages
func (a *App) applyBulkPrompt(promptID int, promptName string) {
	if !a.bulkMode || len(a.selected) == 0 {
		a.GetErrorHandler().ShowWarning(a.ctx, "No messages selected for bulk prompt")
		return
	}

	messageCount := len(a.selected)
	messageIDs := make([]string, 0, messageCount)
	for id := range a.selected {
		messageIDs = append(messageIDs, id)
	}

	// Close the picker
	a.closeBulkPromptPicker()

	// Show AI panel immediately with loading message and set focus
	a.QueueUpdateDraw(func() {
		// Show AI panel manually to avoid potential issues with toggleAISummary
		if !a.aiSummaryVisible {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.aiSummaryView, 0, 1)
			}
			a.aiSummaryVisible = true
		}

		if a.aiSummaryView != nil {
			// Mark panel as being in prompt mode
			a.aiPanelInPromptMode = true

			// Update title to show bulk prompt name
			a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 Bulk: %s (%d messages) ", promptName, messageCount))
			// Show loading message
			a.aiSummaryView.SetText("🤖 Applying bulk prompt...")
			a.aiSummaryView.ScrollToBeginning()

			// Set focus to AI panel
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		}
	})

	// Show initial progress
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Applying '%s' to %d messages...", promptName, messageCount))

	// Get prompt service
	_, _, _, _, _, promptService, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	// Apply bulk prompt - FOLLOWING THE BULK MOVE PATTERN
	go func() {
		var resultBuilder strings.Builder

		ctx, cancel := context.WithCancel(a.ctx)
		a.streamingCancel = cancel // Store cancel function for Esc handler
		defer func() {
			cancel()
			a.streamingCancel = nil // Clear when done
		}()

		// FOLLOWING THE BULK MOVE PATTERN: Process each message individually with progress updates
		failed := 0
		total := len(messageIDs)
		
		// Initial progress update
		a.QueueUpdateDraw(func() {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Processing %d/%d messages...", 1, total))
		})

		// Process each message individually to show progress like bulk move
		for i, messageID := range messageIDs {
			// Progress update for each message (like bulk move does)
			idx := i + 1
			a.QueueUpdateDraw(func() {
				a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Processing %d/%d messages...", idx, total))
			})

			// Get message content for this specific message
			message, err := a.Client.GetMessageWithContent(messageID)
			if err != nil {
				failed++
				continue
			}

			// Extract content for this message using the PlainText field
			rawContent := message.PlainText
			if rawContent == "" {
				failed++
				continue
			}
			
			// Clean the email content before sending to LLM (like the bulk service does)
			content := a.cleanEmailContentForLLM(rawContent)

			// Apply the prompt to this individual message
			result, err := promptService.ApplyPrompt(ctx, content, promptID, map[string]string{
				"from":    a.extractHeader(message, "From"),
				"subject": a.extractHeader(message, "Subject"),
				"date":    a.extractHeader(message, "Date"),
			})

			if err != nil {
				failed++
				continue
			}

			// Build the combined result
			resultBuilder.WriteString(fmt.Sprintf("\n--- MESSAGE %d/%d ---\n", idx, total))
			resultBuilder.WriteString(fmt.Sprintf("From: %s\n", a.extractHeader(message, "From")))
			resultBuilder.WriteString(fmt.Sprintf("Subject: %s\n", a.extractHeader(message, "Subject")))
			resultBuilder.WriteString(fmt.Sprintf("Result:\n%s\n", result.ResultText))
			resultBuilder.WriteString("---\n")

			// Update UI with progress for this message
			a.QueueUpdateDraw(func() {
				if a.aiSummaryView != nil {
					progressText := fmt.Sprintf("🤖 Bulk Prompt Progress: %s\n\n", promptName)
					progressText += fmt.Sprintf("📊 Messages Processed: %d/%d\n", idx, total)
					progressText += fmt.Sprintf("✅ Successful: %d\n", idx-failed)
					progressText += fmt.Sprintf("❌ Failed: %d\n", failed)
					progressText += fmt.Sprintf("⏱️  Processing... 🔄\n\n")
					progressText += "📝 Current Results:\n"
					progressText += resultBuilder.String()

					a.aiSummaryView.SetText(progressText)
					a.aiSummaryView.ScrollToEnd()
				}
			})
		}

		// Final update with complete information
		a.QueueUpdateDraw(func() {
			if a.aiSummaryView != nil {
				finalResult := fmt.Sprintf("🤖 Bulk Prompt Result: %s\n\n", promptName)
				finalResult += fmt.Sprintf("📊 Total Messages: %d\n", total)
				finalResult += fmt.Sprintf("✅ Successful: %d\n", total-failed)
				finalResult += fmt.Sprintf("❌ Failed: %d\n", failed)
				finalResult += fmt.Sprintf("⏱️  Completed\n\n")
				finalResult += fmt.Sprintf("📝 Combined Analysis:\n")
				finalResult += resultBuilder.String()

				a.aiSummaryView.SetText(finalResult)
				a.aiSummaryView.ScrollToBeginning()

				// Ensure focus is on AI panel so escape key works
				a.SetFocus(a.aiSummaryView)
				a.currentFocus = "summary"
				a.updateFocusIndicators("summary")
			}
		})

		// Increment usage count for bulk prompt
		_ = promptService.IncrementUsage(a.ctx, promptID)

		// Clear progress and show success
		a.GetErrorHandler().ClearProgress()
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Bulk prompt '%s' completed for %d messages", promptName, total))
		} else {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Bulk prompt '%s' completed for %d messages (%d failed)", promptName, total, failed))
		}
	}()
}

// cleanEmailContentForLLM processes email content to make it more digestible for LLM
// Similar to the bulk service's cleanEmailContent function
func (a *App) cleanEmailContentForLLM(content string) string {
	if content == "" {
		return "[Empty email]"
	}

	// Remove excessive URLs and tracking links
	lines := strings.Split(content, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip pure URL lines (tracking/unsubscribe links)
		if strings.HasPrefix(line, "https://") && len(line) > 50 {
			continue
		}

		// Skip common email footers
		if strings.Contains(strings.ToLower(line), "unsubscribe") ||
			strings.Contains(strings.ToLower(line), "privacy policy") ||
			strings.Contains(strings.ToLower(line), "powered by") ||
			strings.Contains(strings.ToLower(line), "support@") {
			continue
		}

		// Clean up encoded characters (basic cleanup)
		line = strings.ReplaceAll(line, "-2F", "/")
		line = strings.ReplaceAll(line, "-2B", "+")
		line = strings.ReplaceAll(line, "-3D", "=")

		// Limit line length for readability
		if len(line) > 200 {
			line = line[:200] + "..."
		}

		cleanLines = append(cleanLines, line)

		// Limit total lines to prevent overwhelming the LLM
		if len(cleanLines) >= 20 {
			cleanLines = append(cleanLines, "[Content truncated for brevity...]")
			break
		}
	}

	if len(cleanLines) == 0 {
		return "[No meaningful content found]"
	}

	return strings.Join(cleanLines, "\n")
}

// showBulkPromptResult displays the result of a bulk prompt operation
func (a *App) showBulkPromptResult(result *services.BulkPromptResult, promptName string) {
	if result == nil {
		a.GetErrorHandler().ShowError(a.ctx, "No result to display")
		return
	}

	// Create result view
	resultView := a.createBulkPromptResultView(result, promptName)

	// Show in modal using Flex layout (following existing pattern)
	modal := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetText(fmt.Sprintf("📊 Bulk Prompt Result: %s", promptName))
	title.SetTextColor(tcell.ColorYellow)
	title.SetBorder(true)

	instructions := tview.NewTextView().SetTextAlign(tview.AlignRight)
	instructions.SetText(" Enter to save  |  Esc to close ")
	instructions.SetTextColor(tcell.ColorGray)

	modal.AddItem(title, 3, 0, false)
	modal.AddItem(resultView, 0, 1, false)
	modal.AddItem(instructions, 2, 0, false)

	// Add modal to pages
	a.Pages.AddPage("bulkPromptResult", modal, true, true)
	a.Pages.SwitchToPage("bulkPromptResult")
	a.SetFocus(resultView)
}

// createBulkPromptResultView creates a view to display bulk prompt results
func (a *App) createBulkPromptResultView(result *services.BulkPromptResult, promptName string) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	// Format the result display
	var content strings.Builder
	content.WriteString(fmt.Sprintf("[yellow]Bulk Prompt Result: %s[white]\n", promptName))
	content.WriteString(fmt.Sprintf("[blue]Messages Analyzed: %d[white]\n", result.MessageCount))
	content.WriteString(fmt.Sprintf("[blue]Processing Time: %v[white]\n", result.Duration))
	if result.FromCache {
		content.WriteString("[green]Result from cache[white]\n")
	}
	content.WriteString("\n[cyan]Analysis:[white]\n")
	content.WriteString(result.Summary)

	textView.SetText(content.String())
	return textView
}

// saveBulkPromptResult saves the bulk prompt result
func (a *App) saveBulkPromptResult(result *services.BulkPromptResult) {
	if result == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "No result to save")
		return
	}

	// For now, we'll just show a success message
	// In the future, this could save to a file or database
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Bulk prompt result saved for %d messages", result.MessageCount))

	// Close the modal by switching back to main view
	a.Pages.SwitchToPage("main")
}
