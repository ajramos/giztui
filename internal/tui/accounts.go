package tui

import (
	"fmt"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// openAccountPicker shows a picker for selecting and managing accounts
func (a *App) openAccountPicker() {
	if a.logger != nil {
		a.logger.Printf("openAccountPicker: *** ENTERING ACCOUNT PICKER ***")
	}

	// Get account service
	accountService := a.GetAccountService()
	if accountService == nil {
		if a.logger != nil {
			a.logger.Printf("openAccountPicker: account service is nil")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Account service not available")
		return
	}

	// Create picker UI similar to prompts
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(a.GetComponentColors("accounts").Title.Color()).
		SetFieldBackgroundColor(a.GetComponentColors("accounts").Background.Color()).
		SetFieldTextColor(a.GetComponentColors("accounts").Text.Color())
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	type accountItem struct {
		id          string
		displayName string
		email       string
		status      string
		isActive    bool
	}

	var all []accountItem
	var visible []accountItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.displayName), strings.ToLower(filter)) &&
				!strings.Contains(strings.ToLower(item.email), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Status icon
			var statusIcon string
			switch item.status {
			case "connected":
				statusIcon = "✓"
			case "disconnected":
				statusIcon = "⚠"
			case "error":
				statusIcon = "❌"
			default:
				statusIcon = "?"
			}

			// Active indicator
			activeIndicator := ""
			if item.isActive {
				activeIndicator = "● "
			}

			display := fmt.Sprintf("%s%s %s %s", activeIndicator, statusIcon, item.displayName, item.email)

			// Capture variables for closure
			accountID := item.id
			accountName := item.displayName

			list.AddItem(display, "Enter: switch | v: validate", 0, func() {
				if a.logger != nil {
					a.logger.Printf("account picker: selected accountID=%s name=%s", accountID, accountName)
				}
				// Switch to account
				go a.switchToAccount(accountID, accountName)
			})
		}
	}

	// Load accounts in background
	go func() {
		if a.logger != nil {
			a.logger.Printf("openAccountPicker: loading accounts...")
		}
		accounts, err := accountService.ListAccounts(a.ctx)
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("openAccountPicker: failed to load accounts: %v", err)
			}
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load accounts: %v", err))
			return
		}
		if a.logger != nil {
			a.logger.Printf("openAccountPicker: loaded %d accounts", len(accounts))
		}

		// Convert to accountItem
		all = make([]accountItem, 0, len(accounts))
		for _, account := range accounts {
			all = append(all, accountItem{
				id:          account.ID,
				displayName: account.DisplayName,
				email:       account.Email,
				status:      string(account.Status),
				isActive:    account.IsActive,
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
					a.closeAccountPicker()
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						v := visible[0]
						if a.logger != nil {
							a.logger.Printf("account picker: pick via search accountID=%s name=%s", v.id, v.displayName)
						}
						// Switch to account
						go a.switchToAccount(v.id, v.displayName)
					}
				}
			})

			// Create container
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			accountColors := a.GetComponentColors("accounts")
			// Force background rendering for modal containers
			bgColor := accountColors.Background.Color()
			container.SetBackgroundColor(bgColor)
			container.SetBorder(true)

			// Set background on child components as well
			input.SetBackgroundColor(bgColor)
			list.SetBackgroundColor(bgColor)

			container.SetTitle(" 👤 Account Selector ")
			container.SetTitleColor(a.GetComponentColors("accounts").Title.Color())
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)

			// Footer
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter: switch | v: validate | a: add | r: remove | Esc: cancel ")
			footer.SetTextColor(a.GetComponentColors("accounts").Text.Color())
			footer.SetBackgroundColor(bgColor)
			container.AddItem(footer, 1, 0, false)

			// Handle navigation between input and list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closeAccountPicker()
					return nil
				}

				// Account management key bindings
				switch e.Rune() {
				case 'v':
					// Validate account
					if len(visible) > 0 {
						currentIndex := list.GetCurrentItem()
						if currentIndex >= 0 && currentIndex < len(visible) {
							item := visible[currentIndex]
							go a.validateAccount(item.id, item.displayName)
						}
					}
					return nil
				case 'a':
					// Add new account
					go a.addNewAccount()
					return nil
				case 'r':
					// Remove account
					if len(visible) > 0 {
						currentIndex := list.GetCurrentItem()
						if currentIndex >= 0 && currentIndex < len(visible) {
							item := visible[currentIndex]
							go a.removeAccount(item.id, item.displayName)
						}
					}
					return nil
				}
				return e
			})

			// Add to content split like prompts (reusing labels infrastructure)
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}
			a.SetFocus(input)
			a.currentFocus = "labels" // Reuse labels focus infrastructure
			a.updateFocusIndicators("labels")
			a.setActivePicker(PickerAccounts) // Use new picker enum

			// Initial load
			reload("")
		})
	}()
}

// closeAccountPicker closes the account picker and restores focus
func (a *App) closeAccountPicker() {
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

// switchToAccount switches to the selected account with proper cleanup
func (a *App) switchToAccount(accountID, accountName string) {
	if a.logger != nil {
		a.logger.Printf("switchToAccount: switching to accountID=%s name=%s", accountID, accountName)
	}

	// Close picker first
	a.QueueUpdateDraw(func() {
		a.closeAccountPicker()
	})

	// Get account service
	accountService := a.GetAccountService()
	if accountService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Account service not available")
		return
	}

	// Show progress
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Switching to %s...", accountName))

	// Switch account
	if err := accountService.SwitchAccount(a.ctx, accountID); err != nil {
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to switch account: %v", err))
		return
	}

	// TODO: Implement proper cleanup and service reinitialization
	// This will be part of Phase 4 when we update all services

	// Clear progress and show success
	a.GetErrorHandler().ClearProgress()
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Switched to %s", accountName))

	if a.logger != nil {
		a.logger.Printf("switchToAccount: successfully switched to %s", accountName)
	}
}

// validateAccount validates the selected account's connectivity
func (a *App) validateAccount(accountID, accountName string) {
	if a.logger != nil {
		a.logger.Printf("validateAccount: validating accountID=%s name=%s", accountID, accountName)
	}

	// Get account service
	accountService := a.GetAccountService()
	if accountService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Account service not available")
		return
	}

	// Show progress
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Validating %s...", accountName))

	// Validate account
	result, err := accountService.ValidateAccount(a.ctx, accountID)
	if err != nil {
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Validation failed: %v", err))
		return
	}

	// Clear progress and show result
	a.GetErrorHandler().ClearProgress()
	if result.IsValid {
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("%s is connected (%s)", accountName, result.Email))
	} else {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("%s: %s", accountName, result.ErrorMsg))
	}

	if a.logger != nil {
		a.logger.Printf("validateAccount: validation completed for %s, valid=%v", accountName, result.IsValid)
	}
}

// addNewAccount starts the account configuration wizard
func (a *App) addNewAccount() {
	if a.logger != nil {
		a.logger.Printf("addNewAccount: starting account setup wizard")
	}

	// Get account service
	accountService := a.GetAccountService()
	if accountService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Account service not available")
		return
	}

	// For Phase 2, just show a placeholder message
	// The actual wizard implementation will be in Phase 3
	a.GetErrorHandler().ShowWarning(a.ctx, "Account setup wizard not yet implemented - coming in Phase 3")

	if a.logger != nil {
		a.logger.Printf("addNewAccount: wizard not implemented yet")
	}
}

// removeAccount removes the selected account with confirmation
func (a *App) removeAccount(accountID, accountName string) {
	if a.logger != nil {
		a.logger.Printf("removeAccount: removing accountID=%s name=%s", accountID, accountName)
	}

	// Get account service
	accountService := a.GetAccountService()
	if accountService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Account service not available")
		return
	}

	// For now, remove directly - in a real implementation you might want a confirmation dialog
	// Show progress
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Removing %s...", accountName))

	// Remove account
	if err := accountService.RemoveAccount(a.ctx, accountID); err != nil {
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to remove account: %v", err))
		return
	}

	// Clear progress and show success
	a.GetErrorHandler().ClearProgress()
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Removed account: %s", accountName))

	// Refresh the account picker
	go a.openAccountPicker()

	if a.logger != nil {
		a.logger.Printf("removeAccount: successfully removed %s", accountName)
	}
}
