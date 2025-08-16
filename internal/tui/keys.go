package tui

import (
	"fmt"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// bindKeys sets up keyboard shortcuts and routes actions to feature modules
func (a *App) bindKeys() {
	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If command panel is open but focus moved away, auto-hide to avoid stuck state
		if a.cmdMode {
			if inp, ok := a.views["cmdInput"].(*tview.InputField); ok {
				if a.GetFocus() != inp {
					a.hideCommandBar()
				}
			}
		}
		// Command mode routing
		if a.cmdMode {
			// If command input has focus, let it handle input natively
			if inp, ok := a.views["cmdInput"].(*tview.InputField); ok {
				if a.GetFocus() == inp {
					return event
				}
			}
			return a.handleCommandInput(event)
		}

		// If focus is on form widgets (advanced/simple search), don't intercept
		switch a.GetFocus().(type) {
		case *tview.InputField:
			return event
		case *tview.DropDown:
			return event
		case *tview.Form:
			return event
		case *tview.List:
			// When a modal/list picker is open, do not intercept global keys
			return event
		}

		// Only intercept specific keys, let navigation keys pass through
		// Ensure arrow keys navigate the currently focused pane, not the list always
		// tview handles arrow keys per focused primitive, so we avoid overriding them here.
		switch event.Rune() {
		case ' ':
			if list, ok := a.views["list"].(*tview.Table); ok {
				if !a.bulkMode {
					a.bulkMode = true
					r, _ := list.GetSelection()
					if r >= 0 && r < len(a.ids) {
						if a.selected == nil {
							a.selected = make(map[string]bool)
						}
						a.selected[a.ids[r]] = true
					}
					a.reformatListItems()
					// Keep focus highlight consistent (blue) even in Bulk mode
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
					// Show status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, "Bulk mode — space=select, *=all, a=archive, d=trash, m=move, p=prompt, O=obsidian, ESC=exit")
					}()
					return nil
				}
				// toggle selection
				r, _ := list.GetSelection()
				if r >= 0 && r < len(a.ids) {
					mid := a.ids[r]
					if a.selected[mid] {
						delete(a.selected, mid)
					} else {
						a.selected[mid] = true
					}
					a.reformatListItems()
					// Show status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Selected: %d", len(a.selected)))
					}()
				}
				return nil
			}
		case 'v':
			// Toggle bulk mode with 'v' (visual mode - like Vim)
			if list, ok := a.views["list"].(*tview.Table); ok {
				if !a.bulkMode {
					a.bulkMode = true
					r, _ := list.GetSelection()
					if r >= 0 && r < len(a.ids) {
						if a.selected == nil {
							a.selected = make(map[string]bool)
						}
						a.selected[a.ids[r]] = true
					}
					a.reformatListItems()
					// Keep focus highlight consistent (blue) even in Bulk mode
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
					// Show status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, "Bulk mode — space/v=select, *=all, a=archive, d=trash, m=move, p=prompt, O=obsidian, ESC=exit")
					}()
				} else {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.reformatListItems()
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
					// Clear status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ClearProgress()
					}()
				}
				return nil
			}
		case 'b':
			// Toggle bulk mode with 'b' (alternative to 'v')
			if list, ok := a.views["list"].(*tview.Table); ok {
				if !a.bulkMode {
					a.bulkMode = true
					r, _ := list.GetSelection()
					if r >= 0 && r < len(a.ids) {
						if a.selected == nil {
							a.selected = make(map[string]bool)
						}
						a.selected[a.ids[r]] = true
					}
					a.reformatListItems()
					// Keep focus highlight consistent (blue) even in Bulk mode
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
					// Show status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, "Bulk mode — space/v=select, *=all, a=archive, d=trash, m=move, p=prompt, O=obsidian, ESC=exit")
					}()
				} else {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.reformatListItems()
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
					// Clear status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ClearProgress()
					}()
				}
				return nil
			}
		case '*':
			if a.bulkMode {
				if list, ok := a.views["list"].(*tview.Table); ok {
					count := list.GetRowCount()
					if count == 0 {
						return nil
					}
					sel := 0
					for i := 0; i < count && i < len(a.ids); i++ {
						if a.selected[a.ids[i]] {
							sel++
						}
					}
					if sel*2 >= count {
						a.selected = make(map[string]bool)
					} else {
						for i := 0; i < count && i < len(a.ids); i++ {
							a.selected[a.ids[i]] = true
						}
					}
					a.reformatListItems()
					// Show status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Selected: %d", len(a.selected)))
					}()
				}
				return nil
			}
		case ':':
			a.showCommandBar()
			return nil
		case '?':
			a.toggleHelp()
			return nil
		case 'q':
			a.cancel()
			a.Stop()
			return nil
		case 'r':
			if a.draftMode {
				go a.loadDrafts()
			} else {
				go a.reloadMessages()
			}
			return nil
		case 'n':
			if a.currentFocus == "list" && (event.Modifiers()&tcell.ModShift) == 0 {
				go a.loadMoreMessages()
				return nil
			}
			go a.composeMessage(false)
			return nil
		case 's':
			a.openSearchOverlay("remote")
			return nil
		case '/':
			a.openSearchOverlay("local")
			return nil
		case 'u':
			go a.listUnreadMessages()
			return nil
		case 't':
			if a.bulkMode && len(a.selected) > 0 {
				go a.toggleMarkReadUnreadBulk()
				return nil
			}
			go a.toggleMarkReadUnread()
			return nil
		case 'd':
			if a.bulkMode && len(a.selected) > 0 {
				go a.trashSelectedBulk()
				return nil
			}
			go a.trashSelected()
			return nil
		case 'a':
			if a.bulkMode && len(a.selected) > 0 {
				go a.archiveSelectedBulk()
				return nil
			}
			go a.archiveSelected()
			return nil
		case 'R':
			go a.replySelected()
			return nil
		case 'D':
			go a.loadDrafts()
			return nil
		case 'A':
			go a.showAttachments()
			return nil
		case 'F':
			// Search by sender of current message (Inbox scope by default)
			go a.searchByFromCurrent()
			return nil
		case 'T':
			// Search messages addressed to this sender (include Sent)
			go a.searchByToCurrent()
			return nil
		case 'S':
			// Search by exact subject of current message
			go a.searchBySubjectCurrent()
			return nil
		case 'l':
			if a.currentFocus == "search" {
				return nil
			}
			// Toggle contextual labels panel
			a.manageLabels()
			return nil
		case 'p':
			if a.currentFocus == "search" {
				return nil
			}
			// If focus is on AI summary panel, toggle it off
			if a.currentFocus == "summary" && a.aiSummaryVisible {
				a.toggleAISummary()
				return nil
			}
			// If in bulk mode with selected messages, open bulk prompt picker
			if a.bulkMode && len(a.selected) > 0 {
				go a.openBulkPromptPicker()
				return nil
			}
			// Otherwise, open prompt library picker for single message
			go a.openPromptPicker()
			return nil
		case 'm':
			if a.currentFocus == "search" {
				return nil
			}
			if a.bulkMode && len(a.selected) > 0 {
				a.openMovePanelBulk()
			} else {
				a.openMovePanel()
			}
			return nil
		case 'M':
			a.toggleMarkdown()
			return nil
		case 'V':
			if a.currentFocus == "search" {
				return nil
			}
			// Toggle RSVP side panel
			if a.rsvpVisible {
				if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
					split.ResizeItem(a.labelsView, 0, 0)
				}
				a.labelsVisible = false
				a.rsvpVisible = false
				a.restoreFocusAfterModal()
				return nil
			}
			go a.openRSVPModal()
			return nil
		case 'o':
			// Avoid opening suggestions while advanced search is active
			if a.currentFocus == "search" {
				a.showStatusMessage("🔕 Label suggestions disabled while searching")
				return nil
			}
			go a.suggestLabel()
			return nil
		case 'O': // Shift+O for Obsidian ingestion
			if a.currentFocus == "search" {
				return nil
			}
			// Allow Obsidian ingestion in both normal and bulk modes
			go a.sendEmailToObsidian()
			return nil
		case 'w':
			go a.saveCurrentMessageToFile()
			return nil
		case 'W':
			go a.saveCurrentMessageRawEML()
			return nil
		}

		// ESC exits bulk mode or closes AI panel
		if event.Key() == tcell.KeyEscape {
			if a.logger != nil {
				a.logger.Printf("keys: ESC pressed - bulkMode=%v, currentFocus=%s, aiSummaryVisible=%v, streaming=%v",
					a.bulkMode, a.currentFocus, a.aiSummaryVisible, a.streamingCancel != nil)
			}

			// FIRST: Cancel any active streaming operations (this fixes the hanging issue)
			if a.streamingCancel != nil {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - canceling active streaming operation")
				}
				a.streamingCancel()
				a.streamingCancel = nil
				// Continue with other ESC actions after canceling streaming
			}

			// If we're in bulk mode, exit bulk mode
			if a.bulkMode {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - exiting bulk mode")
				}
				a.exitBulkMode()
				// If AI panel is visible, also hide it
				if a.aiSummaryVisible {
					if a.logger != nil {
						a.logger.Printf("keys: ESC - hiding AI panel after bulk mode exit")
					}
					a.hideAIPanel()
				}
				return nil
			}

			// If focus is on AI summary panel, close it
			if a.currentFocus == "summary" && a.aiSummaryVisible {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - hiding AI panel")
				}
				a.hideAIPanel()
				return nil
			}

			// If a search is active and overlay is not focused, delegate to exitSearch
			if a.searchMode != "" {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - exiting search")
				}
				go a.exitSearch()
				return nil
			}

			if a.logger != nil {
				a.logger.Printf("keys: ESC - no action taken")
			}
		}

		// Advanced search (Ctrl+F) global binding
		if (event.Key() == tcell.KeyCtrlF) || ((event.Modifiers()&tcell.ModCtrl) != 0 && event.Rune() == 'f') {
			a.openAdvancedSearchForm()
			return nil
		}

		// Focus toggle between panes; but when advanced search is active, Tab navigates fields
		if event.Key() == tcell.KeyTab {
			if sp, ok := a.views["searchPanel"].(*tview.Flex); ok && sp.GetTitle() == "🔎 Advanced Search" {
				if frm, ok2 := a.views["advForm"].(*tview.Form); ok2 {
					idx, _ := frm.GetFocusedItemIndex()
					items := frm.GetFormItemCount()
					buttons := frm.GetButtonCount()
					if idx < 0 {
						idx = items
					}
					next := idx + 1
					total := items + buttons
					if total > 0 && next >= total {
						next = total - 1
					}
					frm.SetFocus(next)
					if a.logger != nil {
						a.logger.Printf("keys: Tab advsearch idx=%d -> next=%d (items=%d buttons=%d)", idx, next, items, buttons)
					}
					a.currentFocus = "search"
					a.updateFocusIndicators("search")
					return nil
				}
			}
			a.toggleFocus()
			return nil
		}
		// If a picker/list or advanced search form field has focus, do not handle runes globally
		switch a.GetFocus().(type) {
		case *tview.InputField, *tview.List:
			return event
		}

		// VIM-style navigation shortcuts
		switch event.Rune() {
		case 'g':
			if a.handleVimNavigation('g') {
				return nil
			}
			// Fall through to LLM features if not VIM navigation
		case 'G':
			if a.handleVimNavigation('G') {
				return nil
			}
		}

		// LLM features
		if a.LLM != nil {
			switch event.Rune() {
			case 'y':
				a.toggleAISummary()
				return nil
			case 'Y':
				go a.forceRegenerateSummary()
				return nil
			case 'g':
				// Only show generate reply if not handled by VIM navigation
				if !a.isVimNavigationActive() {
					a.showStatusMessage("💬 Generate reply: placeholder")
				}
				return nil
			case 'o':
				go a.suggestLabel()
				return nil
			}
		}

		return event
	})

	// Enter key behavior on list; keep UI-only here
	if table, ok := a.views["list"].(*tview.Table); ok {
		table.SetSelectedFunc(func(row, column int) {
			if row < len(a.ids) {
				go a.showMessage(a.ids[row])
			}
		})
		table.SetSelectionChangedFunc(func(row, column int) {
			if row >= 0 && row < len(a.ids) {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", row+1, len(a.ids)))
				id := a.ids[row]
				go a.showMessageWithoutFocus(id)
				if a.labelsVisible {
					go a.populateLabelsQuickView(id)
				}
				// Close AI panel when changing messages to avoid conflicts and storm requests
				if a.aiSummaryVisible {
					if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
						split.ResizeItem(a.aiSummaryView, 0, 0)
					}
					a.aiSummaryVisible = false
					a.aiPanelInPromptMode = false
					// Don't change focus, just hide the panel
				}
				a.SetCurrentMessageID(id)
				// Re-render list items so bulk selection backgrounds update when focus moves
				a.reformatListItems()
			}
		})
	}
}

// toggleFocus switches focus between list and text view
func (a *App) toggleFocus() {
	currentFocus := a.GetFocus()

	// Build focus ring dynamically based on visible components
	ring := []tview.Primitive{}
	ringNames := []string{}
	// 1) Search (if visible)
	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		_, _, w, h := sp.GetRect()
		if w > 0 && h > 0 {
			if inp, ok2 := a.views["searchInput"].(*tview.InputField); ok2 {
				ring = append(ring, inp)
				ringNames = append(ringNames, "search")
			}
		}
	}
	// 2) List (always)
	ring = append(ring, a.views["list"])
	ringNames = append(ringNames, "list")
	// 3) Text (always)
	ring = append(ring, a.views["text"])
	ringNames = append(ringNames, "text")
	// 4) Labels (if visible)
	if a.labelsVisible {
		ring = append(ring, a.labelsView)
		ringNames = append(ringNames, "labels")
	}
	// 5) Summary (if visible)
	if a.aiSummaryVisible {
		ring = append(ring, a.aiSummaryView)
		ringNames = append(ringNames, "summary")
	}

	// Find index of current focus
	idx := -1
	for i, p := range ring {
		if currentFocus == p {
			idx = i
			break
		}
	}
	// If current focus isn't recognized (e.g., nil), start at beginning
	next := 0
	if idx >= 0 {
		next = (idx + 1) % len(ring)
	}
	// Apply focus and indicators
	a.SetFocus(ring[next])
	a.currentFocus = ringNames[next]
	a.updateFocusIndicators(ringNames[next])
}

// restoreFocusAfterModal restores focus to the appropriate view after closing a modal
func (a *App) restoreFocusAfterModal() {
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// handleVimNavigation handles VIM-style navigation key sequences
func (a *App) handleVimNavigation(key rune) bool {
	// Check if we're in a context where VIM navigation should work
	// Allow VIM navigation when focus is on list or when no specific focus is set
	if a.currentFocus != "" && a.currentFocus != "list" {
		return false
	}

	now := time.Now()

	// Clear sequence if timeout exceeded (1 second)
	if !a.vimTimeout.IsZero() && now.Sub(a.vimTimeout) > time.Second {
		a.vimSequence = ""
	}

	switch key {
	case 'g':
		if a.vimSequence == "g" {
			// Double 'g' - go to first message
			a.vimSequence = ""
			a.vimTimeout = time.Time{}
			a.executeGoToFirst()
			return true
		} else {
			// Start of sequence - wait for next key
			a.vimSequence = "g"
			a.vimTimeout = now.Add(time.Second)
			return true
		}
	case 'G':
		// Single 'G' - go to last message
		a.vimSequence = ""
		a.vimTimeout = time.Time{}
		a.executeGoToCommand([]string{}) // Use working command function
		return true
	}

	return false
}

// isVimNavigationActive checks if a VIM navigation sequence is currently active
func (a *App) isVimNavigationActive() bool {
	if a.vimSequence == "" {
		return false
	}
	// Check if sequence hasn't timed out
	if !a.vimTimeout.IsZero() && time.Now().Sub(a.vimTimeout) <= time.Second {
		return true
	}
	// Clear expired sequence
	a.vimSequence = ""
	a.vimTimeout = time.Time{}
	return false
}
