package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/config"
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

	// Create enhanced picker UI with theme-aware colors
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(a.GetComponentColors("themes").Title.Color()).
		SetFieldBackgroundColor(a.GetComponentColors("themes").Background.Color()).
		SetFieldTextColor(a.GetComponentColors("themes").Text.Color())

	list := tview.NewList().ShowSecondaryText(false) // Keep clean list display
	list.SetBorder(false)

	// Apply theme colors to the list
	themesColors := a.GetComponentColors("themes")
	list.SetMainTextColor(themesColors.Text.Color())
	list.SetSelectedTextColor(themesColors.Background.Color())   // Inverse contrast for selection
	list.SetSelectedBackgroundColor(themesColors.Accent.Color()) // Accent color as highlight

	type themeItem struct {
		name        string
		description string
		version     string
		current     bool
		valid       bool // NEW: Theme validation status
		builtin     bool // NEW: Built-in vs user theme
	}

	var all []themeItem
	var visible []themeItem

	// Enhanced reload function with better formatting
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]

		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Enhanced display with validation and type indicators
			var statusIcon, typeIcon string
			if item.current {
				statusIcon = "✅"
			} else {
				statusIcon = "○"
			}

			if !item.valid {
				statusIcon = "❌" // Invalid theme
			}

			if item.builtin {
				typeIcon = "📦" // Built-in theme
			} else {
				typeIcon = "👤" // User theme
			}

			displayText := fmt.Sprintf("%s %s %s", statusIcon, typeIcon, item.name)
			secondaryText := fmt.Sprintf("%s | v%s", item.description, item.version)

			// Color-code based on status
			if !item.valid {
				secondaryText = "⚠️ Invalid theme file"
			}

			// Capture variables for closure
			themeName := item.name

			list.AddItem(displayText, secondaryText, 0, func() {
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
			// Convert themes to themeItems with validation
			all = make([]themeItem, 0, len(themes))
			for _, themeName := range themes {
				item := themeItem{
					name:    themeName,
					current: themeName == currentTheme,
					builtin: a.isBuiltinTheme(themeName),
				}

				// Validate theme and get metadata
				if themeConfig, err := themeService.GetThemeConfig(a.ctx, themeName); err == nil && themeConfig != nil {
					item.valid = true
					item.description = themeConfig.Description
					item.version = "1.0" // ThemeConfig doesn't have version field
					if item.description == "" {
						item.description = "Theme configuration"
					}
				} else {
					item.valid = false
					item.description = "Invalid theme"
					item.version = "?"
				}

				all = append(all, item)
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
			bgColor := a.GetComponentColors("themes").Background.Color()
			container.SetBackgroundColor(bgColor)
			container.SetBorder(true)

			// Set background on child components as well
			input.SetBackgroundColor(bgColor)
			list.SetBackgroundColor(bgColor)

			container.SetTitle(" 🎨 Theme Picker ")
			container.SetTitleColor(a.GetComponentColors("themes").Title.Color())
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)

			// Footer
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter: preview | Space: apply | Esc: cancel ")
			footer.SetTextColor(a.GetComponentColors("themes").Text.Color())
			footer.SetBackgroundColor(bgColor)
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
			a.currentFocus = "prompts" // Use same focus identifier as prompts for consistency
			a.updateFocusIndicators("prompts")
			a.setActivePicker(PickerThemes) // Needed for proper visual state

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

		// Enhanced preview with actual color samples
		details := fmt.Sprintf("🎨 Theme: %s\n", themeConfig.Name)
		details += fmt.Sprintf("📄 Description: %s\n", themeConfig.Description)
		details += fmt.Sprintf("🔖 Version: 1.0\n\n")

		// Color samples using actual theme colors from service ThemeConfig
		details += "📧 Email Colors:\n"
		details += a.formatColorSampleString("Unread", themeConfig.EmailColors.UnreadColor)
		details += a.formatColorSampleString("Read", themeConfig.EmailColors.ReadColor)
		details += a.formatColorSampleString("Important", themeConfig.EmailColors.ImportantColor)
		details += a.formatColorSampleString("Sent", themeConfig.EmailColors.SentColor)
		details += a.formatColorSampleString("Draft", themeConfig.EmailColors.DraftColor)

		details += "\n🎨 UI Colors:\n"
		details += a.formatColorSampleString("Background", themeConfig.UIColors.BgColor)
		details += a.formatColorSampleString("Text", themeConfig.UIColors.FgColor)
		details += a.formatColorSampleString("Borders", themeConfig.UIColors.BorderColor)
		details += a.formatColorSampleString("Focus", themeConfig.UIColors.FocusColor)

		details += "\n🔖 Status Colors:\n"
		details += a.formatColorSampleString("Error", themeConfig.UIColors.ErrorColor)
		details += a.formatColorSampleString("Success", themeConfig.UIColors.SuccessColor)
		details += a.formatColorSampleString("Warning", themeConfig.UIColors.WarningColor)

		details += "\n💡 Press Space to apply this theme | Tab to return to picker"

		// Show in text view with improved UX (same pattern as showPromptDetails)
		a.QueueUpdateDraw(func() {
			// Update the text container title and hide headers
			if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
				textContainer.SetTitle(" 🎨 Theme Preview ")
				// Use standard yellow for consistency with other titles
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
			} else {
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

// isBuiltinTheme checks if a theme is built-in
func (a *App) isBuiltinTheme(themeName string) bool {
	builtinThemes := []string{"gmail-dark", "gmail-light"}
	for _, builtin := range builtinThemes {
		if themeName == builtin {
			return true
		}
	}
	return false
}

// formatColorSample formats a color sample with actual colors (for config.Color)
func (a *App) formatColorSample(name string, colorValue config.Color) string {
	// Create color tag from hex value
	colorTag := fmt.Sprintf("[%s]", colorValue.String())
	return fmt.Sprintf("  %s● %s%s (%s)\n", colorTag, name, "[-]", colorValue.String())
}

// formatColorSampleString formats a color sample with actual colors (for string hex values)
func (a *App) formatColorSampleString(name string, colorValue string) string {
	// Use the closest named color that tview supports
	namedColor := a.hexToNamedColor(colorValue)

	if namedColor != "" {
		// Format: colored bullet point + colored name + hex value in parentheses
		return fmt.Sprintf("  [%s]●[-] [%s]%s[-] (%s)\n", namedColor, namedColor, name, colorValue)
	}

	// Fallback: no color formatting if we can't map to a named color
	return fmt.Sprintf("  ● %s (%s)\n", name, colorValue)
}

// hexToNamedColor converts hex colors to the closest tview named color
func (a *App) hexToNamedColor(hexColor string) string {
	// Map common hex colors to tview named colors
	colorMap := map[string]string{
		"#ff5555": "red",
		"#50fa7b": "green",
		"#f1fa8c": "yellow",
		"#8be9fd": "cyan",
		"#bd93f9": "purple",
		"#ffb86c": "orange",
		"#6272a4": "blue",
		"#44475a": "gray",
		"#f8f8f2": "white",
		"#282a36": "black",
		// Light theme colors
		"#e74c3c": "red",
		"#27ae60": "green",
		"#f39c12": "yellow",
		"#3498db": "blue",
		"#2c3e50": "black",
		"#ecf0f1": "white",
		"#7f8c8d": "gray",
		"#e67e22": "orange",
	}

	if named, exists := colorMap[hexColor]; exists {
		return named
	}

	// For unknown colors, try to guess based on hex values
	if len(hexColor) == 7 && hexColor[0] == '#' {
		// Simple heuristic based on RGB values
		hex := hexColor[1:]
		if len(hex) == 6 {
			// Parse RGB components (simplified)
			switch {
			case hex[0:2] > hex[2:4] && hex[0:2] > hex[4:6]: // Red dominant
				return "red"
			case hex[2:4] > hex[0:2] && hex[2:4] > hex[4:6]: // Green dominant
				return "green"
			case hex[4:6] > hex[0:2] && hex[4:6] > hex[2:4]: // Blue dominant
				return "blue"
			case hex[0:2] == hex[2:4] && hex[2:4] == hex[4:6]: // Grayscale
				if hex[0:2] > "80" {
					return "white"
				} else {
					return "gray"
				}
			default:
				return "white" // Default fallback
			}
		}
	}

	return "" // No suitable named color found
}

// parseColorFromTheme parses a color string from theme config
func (a *App) parseColorFromTheme(colorStr string) *tcell.Color {
	if colorStr == "" {
		return nil
	}

	color := tcell.GetColor(colorStr)
	return &color
}

// TODO: [THEMING] Investigate proper tview color tag format for hex colors in hierarchical theme system
// Current attempt: [#ff5555]text[-] may not work as expected
// Alternative: Use color names like [red]text[-] or implement custom color display
