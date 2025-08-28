package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// openPromptPicker shows a picker similar to labels for selecting prompts
func (a *App) openPromptPicker() {
	// Use cached message ID (for undo functionality) with sync fallback
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

	if a.logger != nil {
		a.logger.Printf("openPromptPicker: *** ENTERING SINGLE PROMPT PICKER *** for message: %s", messageID)
	}

	// Get message content for prompt processing
	message, err := a.Client.GetMessageWithContent(messageID)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Failed to load message content")
		return
	}

	// Get prompt service
	_, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		if a.logger != nil {
			a.logger.Printf("openPromptPicker: prompt service is nil")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available - check LLM and cache configuration")
		return
	}

	// Create picker UI similar to labels
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(a.GetComponentColors("prompts").Title.Color()).
		SetFieldBackgroundColor(a.GetComponentColors("prompts").Background.Color()).
		SetFieldTextColor(a.GetComponentColors("prompts").Text.Color())
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
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Category icon
			icon := "📄"
			switch item.category {
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

			list.AddItem(display, "Enter: apply", 0, func() {
				if a.logger != nil {
					a.logger.Printf("prompt picker: selected promptID=%d name=%s", promptID, promptName)
				}
				// Apply prompt (it will handle closing picker and setting focus)
				go a.applyPromptToMessage(messageID, promptID, promptName, message)
			})
		}
	}

	// Load prompts in background
	go func() {
		if a.logger != nil {
			a.logger.Printf("openPromptPicker: loading prompts...")
		}
		prompts, err := promptService.ListPrompts(a.ctx, "")
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("openPromptPicker: failed to load prompts: %v", err)
			}
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load prompts: %v", err))
			return
		}
		if a.logger != nil {
			a.logger.Printf("openPromptPicker: loaded %d prompts", len(prompts))
		}

		// Convert to promptItem, excluding bulk_analysis prompts
		all = make([]promptItem, 0, len(prompts))
		for _, p := range prompts {
			// Skip bulk_analysis prompts for single message picker
			if p.Category == "bulk_analysis" {
				continue
			}
			all = append(all, promptItem{
				id:          p.ID,
				name:        p.Name,
				description: p.Description,
				category:    p.Category,
			})
		}

		a.QueueUpdateDraw(func() {
			// Set up input field
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })

			// Allow navigation from input to list
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp || e.Key() == tcell.KeyPgDn || e.Key() == tcell.KeyPgUp {
					a.SetFocus(list)
					return e
				}
				return e
			})

			// Handle enter in input field (select first match)
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.closePromptPicker()
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						v := visible[0]
						if a.logger != nil {
							a.logger.Printf("prompt picker: pick via search promptID=%d name=%s", v.id, v.name)
						}
						// Apply prompt (it will handle closing picker and setting focus)
						go a.applyPromptToMessage(messageID, v.id, v.name, message)
					}
				}
			})

			// Create container
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			promptColors := a.GetComponentColors("prompts")
			// Force background rendering for modal containers
			bgColor := promptColors.Background.Color()
			container.SetBackgroundColor(bgColor)
			container.SetBorder(true)
			
			// Set background on child components as well
			input.SetBackgroundColor(bgColor)
			list.SetBackgroundColor(bgColor)
			
			container.SetTitle(" 🤖 Prompt Library ")
			container.SetTitleColor(a.GetComponentColors("prompts").Title.Color())
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)

			// Footer
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter to apply | Esc to cancel ")
			footer.SetTextColor(a.GetComponentColors("prompts").Text.Color())
			footer.SetBackgroundColor(bgColor)
			container.AddItem(footer, 1, 0, false)

			// Handle navigation between input and list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closePromptPicker()
					return nil
				}
				return e
			})

			// Add to content split like labels
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

			// Initial load
			reload("")
		})
	}()
}

// closePromptPicker closes the prompt picker and restores focus
func (a *App) closePromptPicker() {
	// Cancel any active streaming operations
	if a.streamingCancel != nil {
		a.streamingCancel()
		a.streamingCancel = nil
	}

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.labelsVisible = false

	// Restore original text container title and show headers
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitle(" 📄 Message Content ")
		textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())

		// Restore message headers by resizing header back to original height
		if header, ok := a.views["header"].(*tview.TextView); ok {
			// Use stored original height if available, otherwise fallback to default
			height := a.originalHeaderHeight
			if height == 0 {
				height = 6 // Fallback to default height
			}
			textContainer.ResizeItem(header, height, 0)
			a.originalHeaderHeight = 0 // Reset the stored height
		}
	}

	if text, ok := a.views["text"].(*tview.TextView); ok {
		a.SetFocus(text)
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	}
}

// applyPromptToMessage applies the selected prompt to the message and shows result in AI panel
func (a *App) applyPromptToMessage(messageID string, promptID int, promptName string, message *gmail.Message) {
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: starting messageID=%s promptID=%d name=%s", messageID, promptID, promptName)
	}

	// Close picker first
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.labelsVisible = false
	})

	// Get services
	_, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: prompt service not available")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: got services successfully")
	}

	// Extract message content using same pattern as AI summary
	content := message.PlainText
	if len([]rune(content)) > 8000 {
		content = string([]rune(content)[:8000])
	}

	if content == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No content found in message")
		return
	}

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

			// Update title to show prompt name
			a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 %s ", promptName))
			a.aiSummaryView.SetTitleColor(a.GetComponentColors("ai").Title.Color())
			// Show loading message
			a.aiSummaryView.SetText("🤖 Applying prompt...")
			a.aiSummaryView.ScrollToBeginning()

			// Remove direct ESC handler from AI panel to avoid conflicts
			// The main ESC handler in keys.go will handle all ESC events
			a.aiSummaryView.SetInputCapture(nil)

			// Set focus to AI panel
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")

			if a.logger != nil {
				a.logger.Printf("applyPromptToMessage: initial panel setup - title: %s, text: 🤖 Applying prompt...", promptName)
			}
		}
	})

	// Get account email for caching
	accountEmail := a.getActiveAccountEmail()

	// Check if we have a cached result first
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: checking cache for accountEmail=%s messageID=%s promptID=%d", accountEmail, messageID, promptID)
	}
	if cachedResult, err := promptService.GetCachedResult(a.ctx, accountEmail, messageID, promptID); err == nil && cachedResult != nil {
		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: found cached result")
		}
		// Show progress message for cached result
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Loading cached result: %s...", promptName))

		// Update AI panel with cached result
		a.QueueUpdateDraw(func() {
			if a.aiSummaryView != nil {
				// Set the cached result text
				a.aiSummaryView.SetText(cachedResult.ResultText)
				a.aiSummaryView.ScrollToBeginning()
			}
		})

		// Clear progress and show success
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("%s (cached)", promptName))
		return
	}

	// Show progress in status
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: no cached result, applying prompt")
	}
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Applying prompt: %s...", promptName))

	// Get the prompt template to build the full prompt text
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: getting prompt template for promptID=%d", promptID)
	}
	template, err := promptService.GetPrompt(a.ctx, promptID)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: failed to get prompt template: %v", err)
		}
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to get prompt template: %v", err))
		return
	}
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: got template name=%s", template.Name)
	}

	// Build the full prompt text with variable substitution
	promptText := template.PromptText
	variables := map[string]string{
		"from":    a.extractHeader(message, "From"),
		"subject": a.extractHeader(message, "Subject"),
		"date":    a.extractHeader(message, "Date"),
		"body":    content,
	}

	// Replace all variables in the prompt
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		promptText = strings.ReplaceAll(promptText, placeholder, value)
	}

	// Try streaming first
	if a.Config != nil && a.Config.LLM.StreamEnabled {
		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: attempting streaming via prompt service - StreamEnabled=%v", a.Config.LLM.StreamEnabled)
		}

		// Show streaming progress in status bar
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Streaming prompt: %s...", promptName))

		// Show loading message before starting streaming
		a.QueueUpdateDraw(func() {
			if a.aiSummaryView != nil {
				a.aiSummaryView.SetText("🤖 Processing prompt...")
			}
		})

		ctx, cancel := context.WithCancel(a.ctx)
		a.streamingCancel = cancel // Store cancel function for Esc handler
		defer func() {
			cancel()
			a.streamingCancel = nil // Clear when done
		}()

		// Throttling for visible streaming effect
		var lastUpdate time.Time
		var b strings.Builder
		chunkDelayMs := a.Config.LLM.StreamChunkMs
		if chunkDelayMs <= 0 {
			chunkDelayMs = 150 // Default 150ms for smooth streaming
		}
		chunkDelay := time.Duration(chunkDelayMs) * time.Millisecond

		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: using %dms chunk delay", chunkDelayMs)
		}

		result, err := promptService.ApplyPromptStream(ctx, content, promptID, map[string]string{
			"from":    a.extractHeader(message, "From"),
			"subject": a.extractHeader(message, "Subject"),
			"date":    a.extractHeader(message, "Date"),
		}, func(token string) {
			// Check if context is cancelled before processing
			select {
			case <-ctx.Done():
				if a.logger != nil {
					a.logger.Printf("STREAMING CALLBACK: Context cancelled, exiting early")
				}
				return // Exit early if cancelled
			default:
			}

			b.WriteString(token)

			// Throttle UI updates for visible streaming effect
			now := time.Now()
			if now.Sub(lastUpdate) >= chunkDelay || lastUpdate.IsZero() {
				lastUpdate = now
				currentText := sanitizeForTerminal(b.String())

				// CRITICAL: NEVER use QueueUpdateDraw in streaming callbacks
				// Direct UI update to prevent deadlock with ESC handler
				if ctx.Err() == nil && a.aiSummaryView != nil {
					a.aiSummaryView.SetText(currentText)
					a.aiSummaryView.ScrollToEnd()

					// Force tview to refresh the screen for visible streaming
					a.ForceDraw()
				}

				// Add small sleep to ensure UI updates are visible
				time.Sleep(time.Duration(chunkDelayMs/2) * time.Millisecond)
			}
		})

		if err != nil {
			if a.logger != nil {
				a.logger.Printf("applyPromptToMessage: streaming failed, falling back to non-streaming: %v", err)
			}
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Streaming failed: %v", err))
		} else {
			if a.logger != nil {
				a.logger.Printf("applyPromptToMessage: streaming completed successfully")
			}
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s", promptName))

			// Final UI update to ensure result is shown
			a.QueueUpdateDraw(func() {
				if a.aiSummaryView != nil {
					finalText := sanitizeForTerminal(result.ResultText)
					a.aiSummaryView.SetText(finalText)
					a.aiSummaryView.ScrollToBeginning()
				}
			})

			// Clear progress and show success
			a.GetErrorHandler().ClearProgress()
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s", promptName))
			return
		}
	}

	// Fallback to non-streaming
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: using non-streaming fallback")
	}
	a.GetErrorHandler().ShowWarning(a.ctx, "Using non-streaming fallback")

	result, err := promptService.ApplyPrompt(a.ctx, content, promptID, map[string]string{
		"from":    a.extractHeader(message, "From"),
		"subject": a.extractHeader(message, "Subject"),
		"date":    a.extractHeader(message, "Date"),
	})

	if err != nil {
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply prompt: %v", err))
		return
	}

	// Save result for history
	_ = promptService.SaveResult(a.ctx, accountEmail, messageID, promptID, result.ResultText)

	// Update AI panel with result
	a.QueueUpdateDraw(func() {
		if a.aiSummaryView != nil {
			// Set the result text
			a.aiSummaryView.SetText(result.ResultText)
			a.aiSummaryView.ScrollToBeginning()
		}
	})

	// Clear progress and show success
	a.GetErrorHandler().ClearProgress()
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s", promptName))
}

// extractHeader extracts a header value from a message
func (a *App) extractHeader(message *gmail.Message, headerName string) string {
	if message.Payload == nil || message.Payload.Headers == nil {
		return ""
	}

	for _, header := range message.Payload.Headers {
		if header.Name == headerName {
			return header.Value
		}
	}
	return ""
}

// openPromptPickerForManagement opens an enhanced prompt picker for management operations
func (a *App) openPromptPickerForManagement() {
	// Get prompt service
	_, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available - check LLM and cache configuration")
		return
	}

	// Create picker UI similar to regular prompt picker
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(a.GetComponentColors("prompts").Title.Color()).
		SetFieldBackgroundColor(a.GetComponentColors("prompts").Background.Color()).
		SetFieldTextColor(a.GetComponentColors("prompts").Text.Color())
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	type promptItem struct {
		id          int
		name        string
		description string
		category    string
		usageCount  int
	}

	var all []promptItem
	var visible []promptItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Category icon and usage count display
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
			secondary := fmt.Sprintf("Category: %s | Used: %d times", item.category, item.usageCount)

			// Capture variables for closure
			promptID := item.id
			promptName := item.name

			list.AddItem(display, secondary, 0, func() {
				// Show full prompt details in text view
				a.showPromptDetails(promptID, promptName)
			})
		}
	}

	// Load prompts in background
	go func() {
		prompts, err := promptService.ListPrompts(a.ctx, "")
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load prompts: %v", err))
			return
		}

		a.QueueUpdateDraw(func() {
			// Show ALL prompts (no category filtering for management)
			all = make([]promptItem, 0, len(prompts))
			for _, p := range prompts {
				all = append(all, promptItem{
					id:          p.ID,
					name:        p.Name,
					description: p.Description,
					category:    p.Category,
					usageCount:  p.UsageCount,
				})
			}

			reload("")

			// Set up input field
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })

			// Handle input events
			input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEscape:
					a.closePromptManager()
					return nil
				case tcell.KeyDown, tcell.KeyUp, tcell.KeyPgDn, tcell.KeyPgUp:
					a.SetFocus(list)
					return event
				}
				return event
			})

			// Enhanced list input capture with management keys
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closePromptManager()
					return nil
				}

				// Management key bindings
				switch e.Rune() {
				case 'e':
					// Export prompt - ask for file path
					if len(visible) > 0 {
						currentIndex := list.GetCurrentItem()
						if currentIndex >= 0 && currentIndex < len(visible) {
							item := visible[currentIndex]
							go a.promptForExportPath(item.id, item.name)
						}
					}
					return nil
				case 'd':
					// Delete prompt with confirmation
					if len(visible) > 0 {
						currentIndex := list.GetCurrentItem()
						if currentIndex >= 0 && currentIndex < len(visible) {
							item := visible[currentIndex]
							go a.confirmDeletePrompt(item.id, item.name)
						}
					}
					return nil
				}
				return e
			})

			// Handle enter in input field (select first match)
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.closePromptManager()
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						v := visible[0]
						a.showPromptDetails(v.id, v.name)
					}
				}
			})
		})
	}()

	// Create container
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	promptColors := a.GetComponentColors("prompts")
	
	// Force background rendering for modal containers
	bgColor := promptColors.Background.Color()
	container.SetBackgroundColor(bgColor)
	container.SetBorder(true)
	
	// Set background on child components as well
	input.SetBackgroundColor(bgColor)
	list.SetBackgroundColor(bgColor)
	
	container.SetTitle(" 📚 Prompt Library Manager ")
	container.SetTitleColor(a.GetComponentColors("prompts").Title.Color())
	container.AddItem(input, 3, 0, true)
	container.AddItem(list, 0, 1, true)

	// Enhanced footer with management instructions
	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter: view | e: export | d: delete | Esc: close ")
	footer.SetTextColor(a.GetComponentColors("prompts").Text.Color())
	footer.SetBackgroundColor(bgColor)
	container.AddItem(footer, 1, 0, false)

	// Add to content split
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
	a.labelsVisible = true
}

// closePromptManager closes the prompt manager and restores the original view
func (a *App) closePromptManager() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.ResizeItem(a.labelsView, 0, 0)
		}
	}
	a.labelsVisible = false

	// Restore original text container title and show headers
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitle(" 📄 Message Content ")
		textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())

		// Restore message headers by resizing header back to original height
		if header, ok := a.views["header"].(*tview.TextView); ok {
			// Use stored original height if available, otherwise fallback to default
			height := a.originalHeaderHeight
			if height == 0 {
				height = 6 // Fallback to default height
			}
			textContainer.ResizeItem(header, height, 0)
			a.originalHeaderHeight = 0 // Reset the stored height
		}
	}

	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// showPromptDetails displays the full prompt in the text view
func (a *App) showPromptDetails(promptID int, promptName string) {
	// Get services
	_, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	go func() {
		prompt, err := promptService.GetPrompt(a.ctx, promptID)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to get prompt details: %v", err))
			return
		}

		// Format prompt details for display
		details := fmt.Sprintf("📝 Prompt: %s\n", prompt.Name)
		details += fmt.Sprintf("📁 Category: %s\n", prompt.Category)
		details += fmt.Sprintf("📊 Usage Count: %d\n", prompt.UsageCount)
		if prompt.Description != "" {
			details += fmt.Sprintf("📄 Description: %s\n", prompt.Description)
		}
		details += fmt.Sprintf("🆔 ID: %d\n", prompt.ID)
		details += "\nTemplate:\n\n"
		details += prompt.PromptText

		// Show in text view with improved UX
		a.QueueUpdateDraw(func() {
			// Update the text container title and hide headers
			if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
				textContainer.SetTitle(" 📝 Prompt Details ")
				textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())

				// Store the current header height before hiding it
				if header, ok := a.views["header"].(*tview.TextView); ok {
					// Calculate current header height based on its content
					headerContent := header.GetText(false)
					a.originalHeaderHeight = a.calculateHeaderHeight(headerContent)

					// Hide message headers by resizing header to 0 height
					textContainer.ResizeItem(header, 0, 0)
				}

				// Debug: Log that we're setting the title
			}

			if textView, ok := a.views["text"].(*tview.TextView); ok {
				textView.SetText(details)
				textView.ScrollToBeginning()

				// Set focus to text view for scrolling (use EnhancedTextView if available)
				if a.enhancedTextView != nil {
					a.SetFocus(a.enhancedTextView)
				} else {
					a.SetFocus(textView)
				}
				a.currentFocus = "text"
				a.updateFocusIndicators("text")
			}
			// Also update enhanced text view if available
			if a.enhancedTextView != nil {
				a.enhancedTextView.SetContent(details)
			}
		})

		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Showing details for: %s | Tab: back to picker", promptName))
		}()
	}()
}

// promptForExportPath prompts user for export path via input dialog
func (a *App) promptForExportPath(promptID int, promptName string) {
	// For now, use a simple naming pattern - in a real implementation you might want a file picker
	defaultPath := fmt.Sprintf("~/prompt_%s.md", strings.ReplaceAll(strings.ToLower(promptName), " ", "_"))

	// Get services
	_, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	// Export to the default path
	err := promptService.ExportToFile(a.ctx, promptID, defaultPath)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to export prompt: %v", err))
		return
	}

	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Exported '%s' to %s", promptName, defaultPath))
}

// confirmDeletePrompt asks for confirmation before deleting a prompt
func (a *App) confirmDeletePrompt(promptID int, promptName string) {
	// For now, delete directly - in a real implementation you might want a confirmation dialog
	_, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	// Delete the prompt
	err := promptService.DeletePrompt(a.ctx, promptID)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to delete prompt: %v", err))
		return
	}

	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Deleted prompt: %s", promptName))

	// Refresh the prompt manager
	go a.openPromptPickerForManagement()
}
