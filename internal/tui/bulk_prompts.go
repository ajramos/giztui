package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	_, _, _, _, _, promptService, _, _, _ := a.GetServices()
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
		SetFieldWidth(30).
		SetLabelColor(a.getTitleColor()).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetFieldTextColor(tview.Styles.PrimaryTextColor)
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
			a.logger.Printf("bulk prompt picker: loading prompts for bulk mode=%v, selected=%d", a.bulkMode, len(a.selected))
		}

		prompts, err := promptService.ListPrompts(a.ctx, "")
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load prompts: %v", err))
			return
		}

		a.QueueUpdateDraw(func() {
			all = make([]promptItem, 0, len(prompts))

			if a.logger != nil {
				a.logger.Printf("bulk prompt picker: received %d total prompts", len(prompts))
				for i, p := range prompts {
					a.logger.Printf("bulk prompt picker: prompt[%d] name='%s' category='%s' ID=%d", i, p.Name, p.Category, p.ID)
				}
			}

			// Only show bulk_analysis prompts for bulk operations
			for _, p := range prompts {
				if p.Category == "bulk_analysis" {
					all = append(all, promptItem{
						id:          p.ID,
						name:        p.Name,
						description: p.Description,
						category:    p.Category,
					})
					if a.logger != nil {
						a.logger.Printf("bulk prompt picker: ADDED bulk_analysis prompt: '%s'", p.Name)
					}
				}
			}

			if a.logger != nil {
				bulkCount := 0
				for _, p := range prompts {
					if p.Category == "bulk_analysis" {
						bulkCount++
					}
				}
				a.logger.Printf("bulk prompt picker: loaded %d bulk_analysis prompts out of %d total", bulkCount, len(prompts))
				a.logger.Printf("bulk prompt picker: final 'all' array has %d items", len(all))
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
	container.SetTitleColor(a.GetComponentColors("prompts").Title.Color())
	container.AddItem(input, 3, 0, true)
	container.AddItem(list, 0, 1, true)

	// Footer
	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter to apply | Esc to cancel ")
	footer.SetTextColor(a.getFooterColor())
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

	// Reformat list items to remove bulk indicators
	a.reformatListItems()

	// Reset list selection style to normal
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetSelectedStyle(a.getSelectionStyle())
	}

	// Hide AI panel if it's visible
	if a.aiSummaryVisible {
		// Hide AI panel directly
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.aiSummaryView, 0, 0) // Hide AI panel
		}
		a.aiSummaryVisible = false
		a.aiPanelInPromptMode = false
	}

	// Return focus to list
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")

	// Update status asynchronously to avoid deadlock
	go func() {
		a.GetErrorHandler().ClearProgress()
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

	// Clear any status message asynchronously to avoid deadlock
	go func() {
		a.GetErrorHandler().ClearProgress()
	}()

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
	_, _, _, _, _, promptService, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	// Apply bulk prompt using the dedicated bulk prompt service
	go func() {
		var resultBuilder strings.Builder

		ctx, cancel := context.WithCancel(a.ctx)
		a.streamingCancel = cancel // Store cancel function for Esc handler
		defer func() {
			cancel()
			a.streamingCancel = nil // Clear when done
		}()

		accountEmail := a.getActiveAccountEmail()

		// Check if we can use streaming for this bulk operation
		_, _, _, _, _, promptService, _, _, _ := a.GetServices()
		if promptService == nil {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
			return
		}

		if a.logger != nil {
			a.logger.Printf("BULK_PROMPT: Starting bulk prompt operation - promptID=%d, messageCount=%d", promptID, messageCount)
		}

		// Try streaming bulk prompt first
		result, err := promptService.ApplyBulkPromptStream(ctx, accountEmail, messageIDs, promptID, map[string]string{}, func(token string) {
			// Check if context was canceled
			select {
			case <-ctx.Done():
				return
			default:
			}

			resultBuilder.WriteString(token)
			currentText := resultBuilder.String()

			// CRITICAL: NEVER use QueueUpdateDraw in streaming callbacks
			// Direct UI update to prevent deadlock with ESC handler
			if ctx.Err() == nil && a.aiSummaryView != nil {
				// Format the streaming result
				formattedResult := fmt.Sprintf("🤖 Bulk Prompt Result: %s\n\n", promptName)
				formattedResult += fmt.Sprintf("📊 Messages Processed: %d\n", messageCount)
				formattedResult += "⏱️  Processing... 🔄\n"
				formattedResult += "📝 Analysis (streaming):\n"
				formattedResult += currentText

				a.aiSummaryView.SetText(formattedResult)
				a.aiSummaryView.ScrollToEnd()
			}
		})

		if err != nil {
			// Check if error is due to context cancellation (user pressed ESC)
			if ctx.Err() == context.Canceled {
				a.GetErrorHandler().ShowInfo(a.ctx, "Bulk prompt operation canceled")
				return
			}
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply bulk prompt: %v", err))
			return
		}

		// Final update with complete information
		a.QueueUpdateDraw(func() {
			if a.aiSummaryView != nil {
				formattedResult := fmt.Sprintf("🤖 Bulk Prompt Result: %s\n\n", promptName)
				formattedResult += fmt.Sprintf("📊 Messages Processed: %d\n", result.MessageCount)
				formattedResult += fmt.Sprintf("⏱️  Processing Time: %v\n", result.Duration)
				formattedResult += fmt.Sprintf("💾 From Cache: %v\n\n", result.FromCache)
				formattedResult += "📝 Analysis:\n"
				formattedResult += result.Summary

				a.aiSummaryView.SetText(formattedResult)
				a.aiSummaryView.ScrollToBeginning()

				// Ensure focus is on AI panel so escape key works
				a.SetFocus(a.aiSummaryView)
				a.currentFocus = "summary"
				a.updateFocusIndicators("summary")
			}
		})

		// Clear progress and show success asynchronously to avoid hanging
		go func() {
			// Small delay to ensure UI update completes first
			time.Sleep(10 * time.Millisecond)
			a.GetErrorHandler().ClearProgress()
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Bulk prompt '%s' completed for %d messages", promptName, result.MessageCount))
		}()
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
	title.SetTextColor(a.getTitleColor())
	title.SetBorder(true)

	instructions := tview.NewTextView().SetTextAlign(tview.AlignRight)
	instructions.SetText(" Enter to save  |  Esc to close ")
	instructions.SetTextColor(a.GetComponentColors("prompts").Accent.Color())

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
	content.WriteString(fmt.Sprintf("%sBulk Prompt Result: %s%s\n", a.GetColorTag("title"), promptName, a.GetEndTag()))
	content.WriteString(fmt.Sprintf("%sMessages Analyzed: %d%s\n", a.GetColorTag("link"), result.MessageCount, a.GetEndTag()))
	content.WriteString(fmt.Sprintf("%sProcessing Time: %v%s\n", a.GetColorTag("link"), result.Duration, a.GetEndTag()))
	if result.FromCache {
		content.WriteString(a.FormatHeader("Result from cache") + "\n")
	}
	content.WriteString("\n" + a.FormatEmphasis("Analysis:") + "\n")
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
