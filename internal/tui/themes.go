package tui

import (
	"fmt"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// openThemePicker shows a side panel picker similar to prompts for selecting themes
func (a *App) openThemePicker() {
	if a.logger != nil {
		a.logger.Printf("openThemePicker: *** ENTERING THEME PICKER ***")
	}

	// Get theme service
	themeService := a.GetThemeService()
	if themeService == nil {
		if a.logger != nil {
			a.logger.Printf("openThemePicker: theme service is nil")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Theme service not available")
		return
	}

	// Create picker UI similar to prompts
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30)
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	type themeItem struct {
		name        string
		description string
		current     bool
	}

	var all []themeItem
	var visible []themeItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Active indicator like labels picker (○ unselected, ✅ selected)
			var displayText string
			if item.current {
				displayText = fmt.Sprintf("✅ %s", item.name)
			} else {
				displayText = fmt.Sprintf("○ %s", item.name)
			}

			// Capture variables for closure
			themeName := item.name

			list.AddItem(displayText, "Enter: preview | Space: apply", 0, func() {
				if a.logger != nil {
					a.logger.Printf("theme picker: selected theme=%s", themeName)
				}
				// Show theme preview (like showPromptDetails)
				a.showThemePreview(themeName)
			})
		}
	}

	// Load themes in background
	go func() {
		if a.logger != nil {
			a.logger.Printf("openThemePicker: loading themes...")
		}
		themes, err := themeService.ListAvailableThemes(a.ctx)
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("openThemePicker: failed to load themes: %v", err)
			}
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load themes: %v", err))
			return
		}

		currentTheme, _ := themeService.GetCurrentTheme(a.ctx)
		if a.logger != nil {
			a.logger.Printf("openThemePicker: loaded %d themes, current: %s", len(themes), currentTheme)
		}

		a.QueueUpdateDraw(func() {
			// Convert themes to themeItems
			all = make([]themeItem, 0, len(themes))
			for _, themeName := range themes {
				// Get theme description from config
				description := "Theme configuration"
				if themeConfig, err := themeService.GetThemeConfig(a.ctx, themeName); err == nil && themeConfig != nil {
					description = themeConfig.Description
				}

				all = append(all, themeItem{
					name:        themeName,
					description: description,
					current:     themeName == currentTheme,
				})
			}

			reload("")

			// Set up input field
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })

			// Handle input events
			input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEscape:
					a.closeThemePicker()
					return nil
				case tcell.KeyDown, tcell.KeyUp, tcell.KeyPgDn, tcell.KeyPgUp:
					a.SetFocus(list)
					return event
				case tcell.KeyEnter:
					// Enter key picks first visible theme if filtering
					if len(visible) > 0 {
						themeName := visible[0].name
						if a.logger != nil {
							a.logger.Printf("theme picker: pick via search theme=%s", themeName)
						}
						a.showThemePreview(themeName)
					}
					return nil
				}
				return event
			})

			// Create container (same as prompt picker)
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			container.SetBorder(true)
			container.SetTitle(" 🎨 Theme Picker ")
			container.SetTitleColor(tcell.ColorYellow)
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)

			// Footer
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter: preview | Space: apply | Esc: cancel ")
			footer.SetTextColor(tcell.ColorGray)
			container.AddItem(footer, 1, 0, false)

			// Handle navigation between input and list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closeThemePicker()
					return nil
				}
				if e.Key() == tcell.KeyRune && e.Rune() == ' ' {
					// Space key applies theme directly
					currentIdx := list.GetCurrentItem()
					if currentIdx >= 0 && currentIdx < len(visible) {
						themeName := visible[currentIdx].name
						a.applyThemeFromPicker(themeName)
					}
					return nil
				}
				return e
			})

			// Add to content split like labels (same as prompt picker)
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}
			a.SetFocus(input)
			a.currentFocus = "prompts"  // Use same focus identifier as prompts for consistency
			a.updateFocusIndicators("prompts")
			a.labelsVisible = true // Needed for proper visual state

			// Show info message
			go func() {
				a.GetErrorHandler().ShowInfo(a.ctx, "Theme picker opened | Enter: preview | Space: apply")
			}()
		})
	}()
}

// showThemePreview shows theme details in the text view (similar to showPromptDetails)
func (a *App) showThemePreview(themeName string) {
	// Get theme service
	themeService := a.GetThemeService()
	if themeService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Theme service not available")
		return
	}

	go func() {
		themeConfig, err := themeService.GetThemeConfig(a.ctx, themeName)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to get theme details: %v", err))
			return
		}

		// Format theme details for display (simple approach first)
		details := fmt.Sprintf("🎨 Theme: %s\n", themeName)
		if themeConfig.Description != "" {
			details += fmt.Sprintf("📄 Description: %s\n", themeConfig.Description)
		}
		details += "\n📧 Email Colors:\n"
		details += fmt.Sprintf("  • Unread: %s\n", themeConfig.EmailColors.UnreadColor)
		details += fmt.Sprintf("  • Read: %s\n", themeConfig.EmailColors.ReadColor)
		details += fmt.Sprintf("  • Important: %s\n", themeConfig.EmailColors.ImportantColor)
		details += fmt.Sprintf("  • Sent: %s\n", themeConfig.EmailColors.SentColor)
		details += fmt.Sprintf("  • Draft: %s\n", themeConfig.EmailColors.DraftColor)

		details += "\n🎨 UI Colors:\n"
		details += fmt.Sprintf("  • Text: %s\n", themeConfig.UIColors.FgColor)
		details += fmt.Sprintf("  • Background: %s\n", themeConfig.UIColors.BgColor)
		details += fmt.Sprintf("  • Border: %s\n", themeConfig.UIColors.BorderColor)
		details += fmt.Sprintf("  • Focus: %s\n", themeConfig.UIColors.FocusColor)

		details += "\n💡 Press Space in theme picker to apply this theme"

		// TODO: Add visual color swatches - need to investigate tview color tag format
		details += "\n\n[yellow]Note: Color visualization coming soon...[-]"

		// Show in text view with improved UX (same pattern as showPromptDetails)
		a.QueueUpdateDraw(func() {
			// Update the text container title and hide headers
			if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
				textContainer.SetTitle(" 🎨 Theme Preview ")
				// Use standard yellow for consistency with other titles
				textContainer.SetTitleColor(tcell.ColorYellow)

				// Store the current header height before hiding it
				if header, ok := a.views["header"].(*tview.TextView); ok {
					// Calculate current header height based on its content
					headerContent := header.GetText(false)
					a.originalHeaderHeight = a.calculateHeaderHeight(headerContent)

					// Hide message headers by resizing header to 0 height
					textContainer.ResizeItem(header, 0, 0)
				}

				// Debug: Log that we're setting the title
				if a.logger != nil {
					a.logger.Printf("TITLE DEBUG: Set title to 'Theme Preview' and hid headers for theme: %s (original height: %d)", themeName, a.originalHeaderHeight)
				}
			}

			if textView, ok := a.views["text"].(*tview.TextView); ok {
				textView.SetText(details)
				textView.ScrollToBeginning()

				// Move focus to text view for scrolling (use EnhancedTextView if available)
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
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Previewing: %s | Space: apply | Tab: back to picker", themeName))
		}()
	}()
}

// applyThemeFromPicker applies the selected theme and shows success message
func (a *App) applyThemeFromPicker(themeName string) {
	// Get theme service
	themeService := a.GetThemeService()
	if themeService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Theme service not available")
		return
	}

	go func() {
		if err := themeService.ApplyTheme(a.ctx, themeName); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply theme '%s': %v", themeName, err))
			return
		}

		// Force immediate UI refresh after theme application
		a.QueueUpdateDraw(func() {
			// Force message list to refresh with new colors
			if list, ok := a.views["list"].(*tview.List); ok {
				// Trigger list refresh by clearing and reloading
				currentItem := list.GetCurrentItem()
				go func() {
					// Refresh messages to apply new theme colors
					a.reloadMessages()
					// Restore selection
					a.QueueUpdateDraw(func() {
						if currentItem >= 0 && currentItem < list.GetItemCount() {
							list.SetCurrentItem(currentItem)
						}
					})
				}()
			}

			// Close the theme picker
			a.closeThemePicker()
		})

		// Show success
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied theme: %s", themeName))
	}()
}

// closeThemePicker closes the theme picker and restores focus (like closePromptPicker)
func (a *App) closeThemePicker() {
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
		textContainer.SetTitleColor(tcell.ColorYellow)
		
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
	
	// Restore message content (if we were showing theme preview)
	currentMessageID := a.GetCurrentMessageID()
	if currentMessageID != "" {
		// Force immediate message content refresh to clear theme preview
		go func() {
			// First clear any theme preview content
			a.QueueUpdateDraw(func() {
				if textView, ok := a.views["text"].(*tview.TextView); ok {
					textView.SetText("Loading message...")
				}
				if a.enhancedTextView != nil {
					a.enhancedTextView.SetContent("Loading message...")
				}
			})
			// Then refresh with actual message content
			a.refreshMessageContent(currentMessageID)
		}()
	}
	
	// Restore focus properly to text view (Enhanced or regular)
	if a.enhancedTextView != nil {
		a.SetFocus(a.enhancedTextView)
	} else if text, ok := a.views["text"].(*tview.TextView); ok {
		a.SetFocus(text)
	}
	a.currentFocus = "text"
	a.updateFocusIndicators("text")

	if a.logger != nil {
		a.logger.Printf("closeThemePicker: theme picker closed")
	}
}

// parseColorFromTheme parses a color string from theme config
func (a *App) parseColorFromTheme(colorStr string) *tcell.Color {
	if colorStr == "" {
		return nil
	}
	
	color := tcell.GetColor(colorStr)
	return &color
}

// TODO: Investigate proper tview color tag format for hex colors
// Current attempt: [#ff5555]text[-] may not work as expected
// Alternative: Use color names like [red]text[-] or implement custom color display