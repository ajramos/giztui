package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/gmail"
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
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
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
		promptText  string
		category    string
	}

	var all []promptItem
	var visible []promptItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]

		// Always include "Create new with AI" as the first option (not subject to filter).
		list.AddItem("✨ Create new with AI...", "Enter: open configurator", 0, func() {
			pctx := promptConfiguratorContext{
				mode:      "single",
				messageID: messageID,
			}
			a.closePromptPicker()
			a.openPromptConfigurator(pctx)
		})

		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Category icon
			var icon string
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

			list.AddItem(display, "Enter: apply", 0, func() {
				if a.logger != nil {
					a.logger.Printf("prompt picker: selected promptID=%d name=%s", promptID, promptName)
				}
				// Apply prompt (it will handle closing picker and setting focus)
				go a.applyPromptToMessage(messageID, promptID, promptName, message)
			})
		}

		// Keep the highlight in sync with what Enter will act on: "Create new"
		// (row 0) when unfiltered, otherwise the first match (row 1).
		if filter != "" && len(visible) > 0 {
			list.SetCurrentItem(1)
		} else {
			list.SetCurrentItem(0)
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

		// Convert to promptItem (all categories shown)
		all = make([]promptItem, 0, len(prompts))
		for _, p := range prompts {
			all = append(all, promptItem{
				id:          p.ID,
				name:        p.Name,
				description: p.Description,
				promptText:  p.PromptText,
				category:    p.Category,
			})
		}

		a.QueueUpdateDraw(func() {
			// Set up input field
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })

			// Forward-declared: input.SetInputCapture below closes over triggerPreview, but it
			// is assigned later in this same synchronous QueueUpdateDraw (after container/footer
			// exist). Captures can't fire until this closure returns, so the nil window is never observable.
			var triggerPreview func()

			// Allow navigation from input to list
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if a.matchesConfiguredKey(e, a.Keys.PromptPreview) {
					triggerPreview()
					return nil
				}
				if a.pickerTabCycle(e) {
					return nil
				}
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
					// Act on the list's highlighted item so Enter always matches
					// what's visually selected (row 0 = "✨ Create new with AI...").
					isCreateNew, vi := promptPickerSelection(list.GetCurrentItem(), len(visible))
					if isCreateNew {
						a.closePromptPicker()
						a.openPromptConfigurator(promptConfiguratorContext{
							mode:      "single",
							messageID: messageID,
						})
						return
					}
					v := visible[vi]
					if a.logger != nil {
						a.logger.Printf("prompt picker: pick via search promptID=%d name=%s", v.id, v.name)
					}
					// Apply prompt (it will handle closing picker and setting focus)
					go a.applyPromptToMessage(messageID, v.id, v.name, message)
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

			// triggerPreview opens an inline preview of the highlighted prompt inside
			// the picker container, replacing the list with a scrollable text view.
			triggerPreview = func() {
				isCreateNew, vi := promptPickerSelection(list.GetCurrentItem(), len(visible))
				var name, body string
				var onApply func()
				if isCreateNew {
					name, body = "Create new with AI", promptPreviewCreateNewHint
					onApply = func() {
						a.closePromptPicker()
						a.openPromptConfigurator(promptConfiguratorContext{mode: "single", messageID: messageID})
					}
				} else {
					v := visible[vi]
					name, body = v.name, promptPreviewText(v.description, v.promptText)
					vid, vname := v.id, v.name
					onApply = func() { go a.applyPromptToMessage(messageID, vid, vname, message) }
				}
				a.showPromptPreviewInline(container, input, list, footer, " Enter to apply | Esc to cancel ", name, body, onApply)
			}

			// Handle navigation between input and list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if a.matchesConfiguredKey(e, a.Keys.PromptPreview) {
					triggerPreview()
					return nil
				}
				if a.pickerTabCycle(e) {
					return nil
				}
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
			a.markFocus("prompts")
			a.setActivePicker(PickerPrompts) // Needed for proper visual state

			// Initial load
			reload("")
		})
	}()
}

// closePromptPicker closes the prompt picker and restores focus
func (a *App) closePromptPicker() {
	// Cancel any active streaming operations
	a.aiPanel.cancelStreaming()

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.setActivePicker(PickerNone)

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
			// OBLITERATED: empty else branch eliminated! 💥
			textContainer.ResizeItem(header, height, 0)
			a.originalHeaderHeight = 0 // Reset the stored height
		}
	}

	if text, ok := a.views["text"].(*tview.TextView); ok {
		a.SetFocus(text)
		a.markFocus("text")
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
		a.setActivePicker(PickerNone)
	})

	// Get services
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
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
		if !a.aiPanel.visible.Load() {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.aiSummaryView, 0, 1)
			}
			a.aiPanel.visible.Store(true)
		}

		if a.aiSummaryView != nil {
			// Mark panel as being in prompt mode
			a.aiPanel.inPromptMode = true

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
			a.markFocus("summary")

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
				// Set the cached result text (rendered through Markdown pipeline)
				a.aiSummaryView.SetText(a.renderPromptResult(cachedResult.ResultText))
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
		a.aiPanel.setStreamingCancel(cancel) // Store cancel function for Esc handler
		defer func() {
			cancel()
			a.aiPanel.clearStreamingCancel() // Clear when done
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

			// Final UI update to ensure result is shown (rendered through Markdown pipeline)
			a.QueueUpdateDraw(func() {
				if a.aiSummaryView != nil {
					a.aiSummaryView.SetText(a.renderPromptResult(result.ResultText))
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

	// Update AI panel with result (rendered through Markdown pipeline)
	a.QueueUpdateDraw(func() {
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetText(a.renderPromptResult(result.ResultText))
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

// openPromptPickerForManagement opens the Prompt Library Manager: a list-first CRUD
// picker (parity with the rules manager and saved-queries picker, and with the
// desktop PromptsPicker). Enter views the prompt, '/' filters by name, n creates a
// new prompt, e edits the selected one, d deletes it (two-press confirmation) and
// x exports it to a file.
func (a *App) openPromptPickerForManagement() {
	// Get prompt service
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available - check LLM and cache configuration")
		return
	}

	promptColors := a.GetComponentColors("prompts")
	input := tview.NewInputField().
		SetLabel("🔍 Filter: ").
		SetFieldWidth(30).
		SetLabelColor(promptColors.Title.Color()).
		SetFieldBackgroundColor(promptColors.Background.Color()).
		SetFieldTextColor(promptColors.Text.Color())
	input.SetPlaceholder("press / to filter")
	input.SetPlaceholderTextColor(a.getHintColor())
	list := tview.NewList().ShowSecondaryText(true)
	list.SetBorder(false)
	list.SetMainTextColor(promptColors.Text.Color())
	list.SetSecondaryTextColor(a.getHintColor())
	list.SetSelectedTextColor(promptColors.Background.Color())
	list.SetSelectedBackgroundColor(promptColors.Accent.Color())

	type promptItem struct {
		id          int
		name        string
		description string
		category    string
		usageCount  int
	}

	var all []promptItem
	var visible []promptItem
	currentFilter := ""
	// deletePendingID arms the two-press delete confirmation (-1 = none).
	deletePendingID := -1
	clearDeletePending := func() {
		if deletePendingID != -1 {
			deletePendingID = -1
			go a.GetErrorHandler().ClearPersistentMessage()
		}
	}

	// Reload function for filtering
	reload := func(filter string) {
		currentFilter = filter
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Category icon and usage count display
			var icon string
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

	selected := func() (promptItem, bool) {
		cur := list.GetCurrentItem()
		if cur >= 0 && cur < len(visible) {
			return visible[cur], true
		}
		return promptItem{}, false
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

			// Filter field (list-first: '/' focuses this, typing filters live).
			input.SetChangedFunc(func(text string) {
				currentFilter = strings.TrimSpace(text)
				reload(currentFilter)
			})

			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if a.pickerTabCycle(e) {
					return nil
				}
				switch e.Key() {
				case tcell.KeyDown, tcell.KeyPgDn:
					a.SetFocus(list)
					return nil
				case tcell.KeyEscape:
					input.SetText("")
					reload("")
					a.SetFocus(list)
					return nil
				}
				return e
			})

			// Enter in the filter views the first visible match.
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEnter && len(visible) > 0 {
					v := visible[0]
					a.showPromptDetails(v.id, v.name)
				}
			})

			// List-first CRUD input capture.
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if a.pickerTabCycle(e) {
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					if deletePendingID != -1 {
						clearDeletePending() // cancel the armed delete only; panel stays open
						return nil
					}
					a.closePromptManager()
					return nil
				}
				// '/' enters filter mode (k9s-style; the picker is list-first).
				if e.Rune() == '/' {
					clearDeletePending()
					a.SetFocus(input)
					return nil
				}
				// New prompt (configurable; default "n").
				if a.matchesConfiguredKey(e, a.Keys.PromptNew) {
					clearDeletePending()
					a.showPromptForm(0, "", "", "", "")
					return nil
				}
				// Edit the selected prompt (configurable; default "e").
				if a.matchesConfiguredKey(e, a.Keys.PromptEdit) {
					clearDeletePending()
					if it, ok := selected(); ok {
						a.editPromptByID(it.id)
					}
					return nil
				}
				// Export the selected prompt (configurable; default "x").
				if a.matchesConfiguredKey(e, a.Keys.PromptExport) {
					clearDeletePending()
					if it, ok := selected(); ok {
						go a.promptForExportPath(it.id, it.name)
					}
					return nil
				}
				// Delete the selected prompt (configurable; default "d") — two-press
				// status-bar confirmation, the shape used across the app.
				if a.matchesConfiguredKey(e, a.Keys.PromptDelete) {
					it, ok := selected()
					if !ok {
						return nil
					}
					if deletePendingID == it.id { // second press on the armed row → delete
						deletePendingID = -1
						a.performPromptDelete(it.id, it.name, func() {
							for i := range all {
								if all[i].id == it.id {
									all = append(all[:i], all[i+1:]...)
									break
								}
							}
							a.QueueUpdateDraw(func() { reload(currentFilter) })
						})
						return nil
					}
					deletePendingID = it.id
					go a.GetErrorHandler().ShowPersistentMessage(a.ctx,
						fmt.Sprintf("Delete prompt '%s'? Press '%s' again to confirm, Esc cancels", it.name, a.Keys.PromptDelete),
						LogLevelInfo)
					return nil
				}
				return e
			})
		})
	}()

	// Create container
	container := tview.NewFlex().SetDirection(tview.FlexRow)

	// Force background rendering for modal containers
	bgColor := promptColors.Background.Color()
	container.SetBackgroundColor(bgColor)
	container.SetBorder(true)
	container.SetBorderColor(promptColors.Border.Color())

	// Set background on child components as well
	input.SetBackgroundColor(bgColor)
	list.SetBackgroundColor(bgColor)

	container.SetTitle(" 📚 Prompt Library Manager ")
	container.SetTitleColor(promptColors.Title.Color())
	// List-first: the filter is 3 rows tall but the list holds focus ('/' filters).
	container.AddItem(input, 3, 0, false)
	container.AddItem(list, 0, 1, true)

	// Enhanced footer with management instructions
	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter view · / filter · n new · e edit · d delete · x export · Esc close ")
	footer.SetTextColor(a.GetComponentColors("general").Text.Color())
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
	a.SetFocus(list) // list-first
	a.markFocus("prompts")
	a.setActivePicker(PickerPrompts)
}

// editPromptByID loads a prompt's full template and opens the edit form.
func (a *App) editPromptByID(id int) {
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}
	go func() {
		p, err := promptService.GetPrompt(a.ctx, id)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load prompt: %v", err))
			return
		}
		a.QueueUpdateDraw(func() {
			a.showPromptForm(p.ID, p.Name, p.Description, p.PromptText, p.Category)
		})
	}()
}

// showPromptForm opens an inline form to create (id == 0) or edit a prompt,
// persisting via PromptService.CreatePrompt / UpdatePrompt. Reopens the manager
// (refreshed) on save or cancel. The Template field is single-line (tview has no
// multi-line editor); rich templates are better authored via ':prompt create <file>'.
func (a *App) showPromptForm(id int, name, description, promptText, category string) {
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}
	colors := a.GetComponentColors("prompts")

	form := tview.NewForm()
	form.SetBackgroundColor(colors.Background.Color())
	form.SetFieldBackgroundColor(colors.Background.Color())
	form.SetFieldTextColor(colors.Text.Color())
	form.SetLabelColor(colors.Title.Color())
	form.SetButtonBackgroundColor(colors.Background.Color())
	form.SetButtonTextColor(colors.Text.Color())
	form.AddInputField("Name", name, 44, nil, nil)
	form.AddInputField("Category", category, 44, nil, nil)
	form.AddInputField("Description", description, 44, nil, nil)
	form.AddInputField("Template", promptText, 0, nil, nil)

	field := func(label string) string {
		if fi, ok := form.GetFormItemByLabel(label).(*tview.InputField); ok {
			return strings.TrimSpace(fi.GetText())
		}
		return ""
	}
	reopen := func() {
		a.closePromptManager()
		a.openPromptPickerForManagement()
	}
	save := func() {
		n, cat, desc, tpl := field("Name"), field("Category"), field("Description"), field("Template")
		if n == "" || tpl == "" {
			go a.GetErrorHandler().ShowWarning(a.ctx, "Name and template are required")
			return
		}
		go func() {
			var err error
			if id == 0 {
				_, err = promptService.CreatePrompt(a.ctx, n, desc, tpl, cat)
			} else {
				err = promptService.UpdatePrompt(a.ctx, id, n, desc, tpl, cat)
			}
			switch {
			case err != nil:
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to save prompt: %v", err))
			case id == 0:
				a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Created prompt: %s", n))
			default:
				a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Updated prompt: %s", n))
			}
		}()
		reopen()
	}
	form.AddButton("Save", save)
	form.AddButton("Cancel", reopen)
	form.SetCancelFunc(reopen) // Esc

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(colors.Background.Color())
	container.SetBorder(true)
	container.SetBorderColor(colors.Border.Color())
	title := " ✏️  Edit Prompt "
	if id == 0 {
		title = " ➕ New Prompt "
	}
	container.SetTitle(title)
	container.SetTitleColor(colors.Title.Color())
	container.AddItem(form, 0, 1, true)

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Tab move · Enter on Save · Esc cancel ")
	footer.SetTextColor(a.GetComponentColors("general").Text.Color())
	footer.SetBackgroundColor(colors.Background.Color())
	container.AddItem(footer, 1, 0, false)

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.SetBackgroundColor(colors.Background.Color())
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.markFocus("prompts")
	a.setActivePicker(PickerPrompts)
	a.SetFocus(form)
}

// closePromptManager closes the prompt manager and restores the original view
func (a *App) closePromptManager() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.ResizeItem(a.labelsView, 0, 0)
		}
	}
	a.setActivePicker(PickerNone)

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
			// OBLITERATED: empty else branch eliminated! 💥
			textContainer.ResizeItem(header, height, 0)
			a.originalHeaderHeight = 0 // Reset the stored height
		}
	}

	a.SetFocus(a.views["list"])
	a.markFocus("list")
}

// showPromptDetails displays the full prompt in the text view
func (a *App) showPromptDetails(promptID int, promptName string) {
	// Get services
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
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
				a.markFocus("text")
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
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
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

// performPromptDelete deletes a prompt via PromptService and runs onDone (an
// in-place list refresh) on success. The two-press confirmation prompt is cleared
// first; the manager stays open so the refreshed list is shown.
func (a *App) performPromptDelete(promptID int, promptName string, onDone func()) {
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}
	go func() {
		a.GetErrorHandler().ClearPersistentMessage()
		if err := promptService.DeletePrompt(a.ctx, promptID); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to delete prompt: %v", err))
			return
		}
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Deleted prompt: %s", promptName))
		if onDone != nil {
			onDone()
		}
	}()
}
