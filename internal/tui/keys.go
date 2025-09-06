package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// handleConfigurableKey checks if a key event matches a configurable shortcut and executes the corresponding action
func (a *App) handleConfigurableKey(event *tcell.EventKey) bool {
	// Only handle single character keys for configurable shortcuts
	if event.Rune() == 0 {
		return false
	}

	key := string(event.Rune())

	// Check each configurable shortcut
	switch key {
	// Core email operations
	case a.Keys.Summarize:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> summarize", key)
		}
		a.toggleAISummary()
		return true
	}

	// Check for uppercase version of summarize key (force regenerate)
	// Only create uppercase mapping if the uppercase key is NOT explicitly configured for something else
	if a.Keys.Summarize != "" && len(a.Keys.Summarize) == 1 {
		upperKey := strings.ToUpper(a.Keys.Summarize)

		// Check if this key is explicitly configured for any other function
		// If so, the configured function takes precedence over the automatic uppercase mapping
		if len(upperKey) == 1 && !a.isKeyConfigured(rune(upperKey[0])) && key == upperKey {
			// Only handle uppercase force regenerate if the key is not configured for anything else
			if a.logger != nil {
				a.logger.Printf("Auto-generated shortcut: '%s' -> force_regenerate_summary (uppercase of '%s')", key, a.Keys.Summarize)
			}
			go a.forceRegenerateSummary()
			return true
		}
		// If the uppercase key IS configured, let the configurable shortcuts system handle it
		// (it will be handled in the switch statement below)
	}

	switch key {
	case a.Keys.ForceRegenerateSummary:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> force_regenerate_summary", key)
		}
		go a.forceRegenerateSummary()
		return true
	case a.Keys.GenerateReply:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> generate_reply", key)
		}
		go a.generateReply()
		return true
	case a.Keys.SuggestLabel:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> suggest_label", key)
		}
		go a.suggestLabel()
		return true
	case a.Keys.Reply:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> reply", key)
		}
		go a.replySelected()
		return true
	case a.Keys.ReplyAll:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> reply_all", key)
		}
		go a.replyAllSelected()
		return true
	case a.Keys.Forward:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> forward", key)
		}
		go a.forwardSelected()
		return true
	case a.Keys.Compose:
		// CRITICAL: Check if this is 'n' and we're in content search context
		if key == "n" && a.currentFocus == "text" && a.enhancedTextView != nil && a.enhancedTextView.HasActiveSearch() {
			if a.logger != nil {
				a.logger.Printf("Configurable shortcut: '%s' -> content search next (overriding compose)", key)
			}
			go func() {
			}()
			a.enhancedTextView.searchNext()
			return true
		}

		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> compose", key)
		}
		go a.composeMessage(false)
		return true
	case a.Keys.Refresh:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> refresh", key)
		}
		go a.reloadMessages()
		return true
	case a.Keys.Search:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> search", key)
		}
		// Only handle for email list search when focus is NOT on message content
		// When focus is on "text", let EnhancedTextView handle content search if using same key
		if a.currentFocus != "text" {
			a.openSearchOverlay("remote")
		}
		return true
	case a.Keys.Unread:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> unread", key)
		}
		go a.listUnreadMessages()
		return true
	case a.Keys.Archived:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> archived", key)
		}
		go a.listArchivedMessages()
		return true
	case a.Keys.SearchFrom:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> search_from", key)
		}
		go a.searchByFromCurrent()
		return true
	case a.Keys.SearchTo:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> search_to", key)
		}
		go a.searchByToCurrent()
		return true
	case a.Keys.SearchSubject:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> search_subject", key)
		}
		go a.searchBySubjectCurrent()
		return true
	case a.Keys.ToggleRead:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> toggle_read", key)
		}
		// CRITICAL: Check for bulk mode to ensure bulk operations work
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("Bulk mode active with %d selected messages, calling toggleMarkReadUnreadBulk()", len(a.selected))
			}
			go a.toggleMarkReadUnreadBulk()
		} else {
			go a.toggleMarkReadUnread()
		}
		return true
	case a.Keys.Trash:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> trash (bulkMode: %t, selected: %d)", key, a.bulkMode, len(a.selected))
		}
		go func() {
		}()

		// CRITICAL: Check for bulk mode to ensure bulk operations work
		if a.bulkMode && len(a.selected) > 0 {
			// OBLITERATED: empty logger branch eliminated! 💥
			go a.trashSelectedBulk()
		} else {
			// OBLITERATED: empty logger branch eliminated! 💥
			go a.trashSelected()
		}
		return true
	case a.Keys.Archive:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> archive", key)
		}
		// CRITICAL: Check for bulk mode to ensure bulk operations work
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("Bulk mode active with %d selected messages, calling archiveSelectedBulk()", len(a.selected))
			}
			go a.archiveSelectedBulk()
		} else {
			go a.archiveSelected()
		}
		return true
	case a.Keys.Drafts:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> drafts", key)
		}
		go a.loadDrafts()
		return true
	case a.Keys.Attachments:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> attachments", key)
		}
		go a.showAttachments()
		return true
	case a.Keys.Move:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> move", key)
		}
		// In bulk mode, prioritize bulk operations
		if a.bulkMode && len(a.selected) > 0 {
			a.openMovePanelBulk()
		} else {
			a.openMovePanel()
		}
		return true
	case a.Keys.ManageLabels:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> manage_labels (bulkMode: %t, selected: %d)", key, a.bulkMode, len(a.selected))
		}
		// CRITICAL: Check for bulk mode to ensure bulk label operations work
		if a.bulkMode && len(a.selected) > 0 {
			// OBLITERATED: empty logger branch eliminated! 💥
			a.manageLabelsBulk()
		} else {
			a.manageLabels()
		}
		return true
	case a.Keys.Quit:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> quit", key)
		}
		a.cancel()
		a.Stop()
		return true

	// Additional configurable shortcuts
	case a.Keys.Obsidian:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> obsidian", key)
		}
		go a.sendEmailToObsidian()
		return true
	case a.Keys.Slack:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> slack", key)
		}
		if a.bulkMode && len(a.selected) > 0 {
			go a.showSlackBulkForwardDialog()
		} else {
			go a.showSlackForwardDialog()
		}
		return true
	case a.Keys.Markdown:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> markdown", key)
		}
		a.toggleMarkdown()
		return true
	case a.Keys.SaveMessage:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> save_message", key)
		}
		go a.saveCurrentMessageToFile()
		return true
	case a.Keys.SaveRaw:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> save_raw", key)
		}
		go a.saveCurrentMessageRawEML()
		return true
	case a.Keys.RSVP:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> rsvp", key)
		}
		if a.currentActivePicker == PickerRSVP {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.labelsView, 0, 0) // Hide RSVP panel
			}
			a.setActivePicker(PickerNone)
			a.restoreFocusAfterModal()
		} else {
			go a.openRSVPModal()
		}
		return true
	case a.Keys.LinkPicker:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> link_picker", key)
		}
		go a.openLinkPicker()
		return true
	case a.Keys.ThemePicker:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> theme_picker", key)
		}
		go a.openThemePicker()
		return true
	case a.Keys.OpenGmail:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> open_gmail", key)
		}
		go a.openEmailInGmail()
		return true
	case a.Keys.BulkMode:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> bulk_mode", key)
		}
		if list, ok := a.views["list"].(*tview.Table); ok {
			if !a.bulkMode {
				a.bulkMode = true
				messageIndex := a.getCurrentSelectedMessageIndex()
				if messageIndex >= 0 {
					if a.selected == nil {
						a.selected = make(map[string]bool)
					}
					a.selected[a.ids[messageIndex]] = true
				}
				a.refreshTableDisplay()
				list.SetSelectedStyle(a.getSelectionStyle())
				go func() {
					a.GetErrorHandler().ShowInfo(a.ctx, "Bulk mode — space/v=select, *=all, a=archive, d=trash, m=move, p=prompt, K=slack, O=obsidian, ESC=exit")
				}()
			} else {
				a.bulkMode = false
				a.selected = make(map[string]bool)
				a.refreshTableDisplay()
				list.SetSelectedStyle(a.getSelectionStyle())
				go func() {
					a.GetErrorHandler().ClearProgress()
				}()
			}
		}
		return true
	case a.Keys.CommandMode:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> command_mode", key)
		}
		a.showCommandBar()
		return true
	case a.Keys.Help:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> help", key)
		}
		a.toggleHelp()
		return true
	case a.Keys.LoadMore:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> load_more", key)
		}
		// Only handle when focus is on list
		if a.currentFocus == "list" {
			go a.loadMoreMessages()
		}
		return true
	case a.Keys.ToggleHeaders:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> toggle_headers", key)
		}
		go a.toggleHeaderVisibility()
		return true

	// Threading shortcuts
	case a.Keys.ToggleThreading:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> toggle_threading", key)
		}
		go func() { _ = a.ToggleThreadingMode() }()
		return true
	case a.Keys.ExpandAllThreads:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> expand_all_threads", key)
		}
		go func() { _ = a.ExpandAllThreads() }()
		return true
	case a.Keys.CollapseAllThreads:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> collapse_all_threads", key)
		}
		go func() { _ = a.CollapseAllThreads() }()
		return true
	case a.Keys.BulkSelect:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> bulk_select", key)
		}
		return a.handleBulkSelect()

	// Saved queries
	case a.Keys.SaveQuery:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> save_query", key)
		}
		a.showSaveCurrentQueryDialog()
		return true
	case a.Keys.QueryBookmarks:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> query_bookmarks", key)
		}
		a.showSavedQueriesPicker()
		return true
	}

	return false
}

// isKeyConfigured checks if a key is already configured in the configurable shortcuts
func (a *App) isKeyConfigured(key rune) bool {
	if key == 0 {
		return false
	}

	keyStr := string(key)
	return keyStr == a.Keys.Summarize ||
		keyStr == a.Keys.GenerateReply ||
		keyStr == a.Keys.SuggestLabel ||
		keyStr == a.Keys.Reply ||
		keyStr == a.Keys.ReplyAll ||
		keyStr == a.Keys.Forward ||
		keyStr == a.Keys.Compose ||
		keyStr == a.Keys.Refresh ||
		keyStr == a.Keys.Search ||
		keyStr == a.Keys.Unread ||
		keyStr == a.Keys.ToggleRead ||
		keyStr == a.Keys.Trash ||
		keyStr == a.Keys.Archive ||
		keyStr == a.Keys.Drafts ||
		keyStr == a.Keys.Attachments ||
		keyStr == a.Keys.Move ||
		keyStr == a.Keys.ManageLabels ||
		keyStr == a.Keys.Quit ||
		keyStr == a.Keys.Obsidian ||
		keyStr == a.Keys.Slack ||
		keyStr == a.Keys.Markdown ||
		keyStr == a.Keys.SaveMessage ||
		keyStr == a.Keys.SaveRaw ||
		keyStr == a.Keys.RSVP ||
		keyStr == a.Keys.LinkPicker ||
		keyStr == a.Keys.ThemePicker ||
		keyStr == a.Keys.OpenGmail ||
		keyStr == a.Keys.BulkMode ||
		keyStr == a.Keys.BulkSelect ||
		keyStr == a.Keys.CommandMode ||
		keyStr == a.Keys.Help ||
		keyStr == a.Keys.LoadMore ||
		keyStr == a.Keys.ToggleHeaders ||
		keyStr == a.Keys.SaveQuery ||
		keyStr == a.Keys.QueryBookmarks
}

// bindKeys sets up keyboard shortcuts and routes actions to feature modules
func (a *App) bindKeys() {
	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Debug logging for all key presses
		if a.logger != nil && event.Rune() >= '0' && event.Rune() <= '9' {
			focusType := "nil"
			if focus := a.GetFocus(); focus != nil {
				focusType = fmt.Sprintf("%T", focus)
			}
			a.logger.Printf("=== DIGIT KEY PRESSED: '%c', focus=%s, currentFocus=%s ===", event.Rune(), focusType, a.currentFocus)
		}
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

		// CRITICAL: Check if composition panel is visible first - let it handle ALL input
		if a.compositionPanel != nil && a.compositionPanel.IsVisible() {
			if a.logger != nil {
				a.logger.Printf("=== COMPOSITION PANEL ACTIVE: Allowing event to pass through to composition panel ===")
			}
			// Let the composition panel handle ALL input events
			return event
		}

		// If focus is on form widgets (advanced/simple search), don't intercept
		switch focused := a.GetFocus().(type) {
		case *tview.InputField:
			if a.logger != nil && event.Rune() >= '0' && event.Rune() <= '9' {
				a.logger.Printf("DIGIT KEY: early return for InputField")
			}
			return event
		case *tview.DropDown:
			if a.logger != nil && event.Rune() >= '0' && event.Rune() <= '9' {
				a.logger.Printf("DIGIT KEY: early return for DropDown")
			}
			return event
		case *tview.Form:
			if a.logger != nil && event.Rune() >= '0' && event.Rune() <= '9' {
				a.logger.Printf("DIGIT KEY: early return for Form")
			}
			return event
		case *tview.List:
			if a.logger != nil && event.Rune() >= '0' && event.Rune() <= '9' {
				a.logger.Printf("DIGIT KEY: early return for List")
			}
			// When a modal/list picker is open, do not intercept global keys
			return event
		default:
			if a.logger != nil && event.Rune() >= '0' && event.Rune() <= '9' {
				a.logger.Printf("DIGIT KEY: no early return, focus type: %T", focused)
			}
		}

		// Only intercept specific keys, let navigation keys pass through
		// Ensure arrow keys navigate the currently focused pane, not the list always
		// tview handles arrow keys per focused primitive, so we avoid overriding them here.

		// Debug: log when digit keys reach the main switch
		if a.logger != nil && event.Rune() >= '0' && event.Rune() <= '9' {
			a.logger.Printf("DIGIT KEY: reached main switch statement, checking for VIM sequence")
		}

		// CRITICAL FIX: Check bulk mode operations BEFORE VIM sequences
		// This ensures bulk operations like 'p' work correctly in bulk mode
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("Bulk mode active with %d selected - checking for bulk operations first", len(a.selected))
			}
			// Handle bulk-specific operations before VIM processing
			switch event.Rune() {
			case 'p':
				if a.Keys.Prompt != "" && string(event.Rune()) == a.Keys.Prompt {
					if a.logger != nil {
						a.logger.Printf("Bulk mode: 'p' key intercepted for bulk prompt operation")
					}
					go a.openBulkPromptPicker()
					return nil
				}
			case 'm':
				if a.Keys.Move != "" && string(event.Rune()) == a.Keys.Move {
					if a.logger != nil {
						a.logger.Printf("Bulk mode: 'm' key intercepted for bulk move operation")
					}
					a.openMovePanelBulk()
					return nil
				}
			case 'K':
				if a.Keys.Slack != "" && string(event.Rune()) == a.Keys.Slack {
					if a.logger != nil {
						a.logger.Printf("Bulk mode: 'K' key intercepted for bulk Slack operation")
					}
					a.showSlackBulkForwardDialog()
					return nil
				}
			case 'O':
				if a.Keys.Obsidian != "" && string(event.Rune()) == a.Keys.Obsidian {
					if a.logger != nil {
						a.logger.Printf("Bulk mode: 'O' key intercepted for bulk Obsidian operation")
					}
					go a.openBulkObsidianPanel()
					return nil
				}
			}
		}

		// CRITICAL FIX: Check VIM sequences BEFORE configurable shortcuts
		// This allows f3f to work even when f is configured for toggle_read
		if a.handleVimSequence(event.Rune()) {
			return nil
		}

		// VIM navigation sequences (gg, G) - these don't conflict with main keys
		if a.handleVimNavigation(event.Rune()) {
			return nil
		}

		// Check configurable shortcuts after VIM sequences
		if a.handleConfigurableKey(event) {
			return nil
		}

		switch event.Rune() {
		case ' ':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured(' ') {
				a.handleBulkSelect()
				return nil
			}
		case 'v':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('v') {
				// Toggle bulk mode with 'v' (visual mode - like Vim)
				if list, ok := a.views["list"].(*tview.Table); ok {
					if !a.bulkMode {
						a.bulkMode = true
						messageIndex := a.getCurrentSelectedMessageIndex()
						if messageIndex >= 0 {
							if a.selected == nil {
								a.selected = make(map[string]bool)
							}
							a.selected[a.ids[messageIndex]] = true
						}
						a.refreshTableDisplay()
						// Keep focus highlight consistent (blue) even in Bulk mode
						list.SetSelectedStyle(a.getSelectionStyle())
						// Show status message asynchronously to avoid deadlock
						go func() {
							a.GetErrorHandler().ShowInfo(a.ctx, "Bulk mode — space/v=select, *=all, a=archive, d=trash, m=move, p=prompt, K=slack, O=obsidian, ESC=exit")
						}()
					} else {
						a.bulkMode = false
						a.selected = make(map[string]bool)
						a.refreshTableDisplay()
						list.SetSelectedStyle(a.getSelectionStyle())
						// Clear status message asynchronously to avoid deadlock
						go func() {
							a.GetErrorHandler().ClearProgress()
						}()
					}
					return nil
				}
			}
			// OBLITERATED: redundant break statement eliminated! 💥
		case 'b':
			// Toggle bulk mode with 'b' (alternative to 'v')
			if list, ok := a.views["list"].(*tview.Table); ok {
				if !a.bulkMode {
					a.bulkMode = true
					messageIndex := a.getCurrentSelectedMessageIndex()
					if messageIndex >= 0 {
						if a.selected == nil {
							a.selected = make(map[string]bool)
						}
						a.selected[a.ids[messageIndex]] = true
					}
					a.refreshTableDisplay()
					// Keep focus highlight consistent (blue) even in Bulk mode
					list.SetSelectedStyle(a.getSelectionStyle())
					// Show status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, "Bulk mode — space/v=select, *=all, a=archive, d=trash, m=move, p=prompt, K=slack, O=obsidian, ESC=exit")
					}()
				} else {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.refreshTableDisplay()
					list.SetSelectedStyle(a.getSelectionStyle())
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
					a.refreshTableDisplay()
					// Show status message asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Selected: %d", len(a.selected)))
					}()
				}
				return nil
			}
		case ':':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured(':') {
				a.showCommandBar()
				return nil
			}
			// OBLITERATED: redundant break statement eliminated! 💥
		case '?':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('?') {
				a.toggleHelp()
				return nil
			}
			// OBLITERATED: redundant break statement eliminated! 💥
		case 'q':
			a.cancel()
			a.Stop()
			return nil
		case 'r':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('r') {
				if a.draftMode {
					go a.loadDrafts()
				} else {
					go a.reloadMessages()
				}
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'n':
			// DEBUGGING: Log focus state for n key
			if a.logger != nil {
				focusType := "nil"
				if focus := a.GetFocus(); focus != nil {
					focusType = fmt.Sprintf("%T", focus)
				}
				a.logger.Printf("=== 'n' key pressed: currentFocus=%s, actualFocus=%s ===", a.currentFocus, focusType)
			}
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('n') {
				// Only handle 'n' for compose/load more when focus is on list
				// When focus is on text, let the EnhancedTextView handle it for search navigation
				if a.currentFocus == "list" {
					if (event.Modifiers() & tcell.ModShift) == 0 {
						if a.logger != nil {
							a.logger.Printf("=== 'n' executing loadMoreMessages ===")
						}
						go a.loadMoreMessages()
						return nil
					}
				} else if a.currentFocus != "text" {
					// Only compose message if not focused on text (let text view handle 'n' for search)
					if a.logger != nil {
						a.logger.Printf("=== 'n' executing composeMessage (currentFocus=%s) ===", a.currentFocus)
					}
					go a.composeMessage(false)
					return nil
				}
				// If focus is on text, check if we should handle content search navigation
				if a.currentFocus == "text" && a.enhancedTextView != nil {
					// Check if there's an active search in the EnhancedTextView
					if a.enhancedTextView.HasActiveSearch() {
						if a.logger != nil {
							a.logger.Printf("=== 'n' delegating to EnhancedTextView.searchNext() ===")
						}
						go func() {
						}()
						a.enhancedTextView.searchNext()
						return nil
					}
				}
				// If focus is on text but no active search, let the event pass through to EnhancedTextView
				if a.logger != nil {
					a.logger.Printf("=== 'n' letting event pass through to EnhancedTextView (no active search) ===")
				}
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 's':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('s') {
				a.openSearchOverlay("remote")
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case '/':
			// Only handle for email list search when focus is NOT on message content
			// When focus is on "text", let EnhancedTextView handle content search
			if a.currentFocus != "text" {
				a.openSearchOverlay("local")
				return nil
			}
		case 'u':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('u') {
				go a.listUnreadMessages()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 't':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('t') {
				if a.logger != nil {
					a.logger.Printf("=== MAIN KEY HANDLER: 't' pressed, bulkMode=%v, selected=%d ===", a.bulkMode, len(a.selected))
				}
				// In bulk mode, prioritize bulk operations over VIM sequences
				if a.bulkMode && len(a.selected) > 0 {
					if a.logger != nil {
						a.logger.Printf("Main handler: bulk mode active, calling toggleMarkReadUnreadBulk")
					}
					go a.toggleMarkReadUnreadBulk()
					return nil
				}
				// Check if this might be part of a VIM sequence
				if a.logger != nil {
					a.logger.Printf("Main handler: checking VIM sequence for 't'")
				}
				if a.handleVimSequence(event.Rune()) {
					if a.logger != nil {
						a.logger.Printf("Main handler: VIM sequence handled 't', returning")
					}
					return nil
				}
				if a.logger != nil {
					a.logger.Printf("Main handler: VIM sequence did not handle 't', calling single operation")
				}
				go a.toggleMarkReadUnread()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'd':
			// DEBUGGING: Log bulk mode state
			if a.logger != nil {
				a.logger.Printf("=== 'd' key pressed: bulkMode=%v, selected=%d, currentFocus=%s ===", a.bulkMode, len(a.selected), a.currentFocus)
			}
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('d') {
				// In bulk mode, prioritize bulk operations over VIM sequences
				if a.bulkMode && len(a.selected) > 0 {
					if a.logger != nil {
						a.logger.Printf("=== Executing bulk trash operation ===")
					}
					go func() {
					}()
					go a.trashSelectedBulk()
					return nil
				}
				// Check if this might be part of a VIM sequence
				if a.handleVimSequence(event.Rune()) {
					return nil
				}
				if a.logger != nil {
					a.logger.Printf("=== Executing single trash operation ===")
				}
				go func() {
				}()
				go a.trashSelected()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'a':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('a') {
				// In bulk mode, prioritize bulk operations over VIM sequences
				if a.bulkMode && len(a.selected) > 0 {
					go a.archiveSelectedBulk()
					return nil
				}
				// Check if this might be part of a VIM sequence
				if a.handleVimSequence(event.Rune()) {
					return nil
				}
				go a.archiveSelected()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'R':
			go a.replySelected()
			return nil
		case 'D':
			go a.loadDrafts()
			return nil
		case 'A':
			go a.showAttachments()
			return nil
		case 'B':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('B') {
				go a.listArchivedMessages()
				return nil
			}
		case 'F':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('F') {
				go a.searchByFromCurrent()
				return nil
			}
		case 'T':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('T') {
				go a.searchByToCurrent()
				return nil
			}
		case 'U':
			// Undo last action
			if a.undoService != nil && a.undoService.HasUndoableAction() {
				go a.performUndo()
			} else {
				go func() {
					a.GetErrorHandler().ShowInfo(a.ctx, "No action to undo")
				}()
			}
			return nil
		case 'S':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('S') {
				go a.searchBySubjectCurrent()
				return nil
			}
		case 'K':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('K') {
				// Forward to Slack
				if a.currentFocus == "search" {
					return nil
				}
				if a.bulkMode && len(a.selected) > 0 {
					go a.showSlackBulkForwardDialog()
				} else {
					go a.showSlackForwardDialog()
				}
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'l':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('l') {
				if a.currentFocus == "search" {
					return nil
				}
				// Check if this might be part of a VIM sequence first
				if a.handleVimSequence(event.Rune()) {
					return nil
				}
				// Toggle contextual labels panel
				a.manageLabels()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'p':
			if a.currentFocus == "search" {
				return nil
			}
			// If focus is on AI summary panel, toggle it off
			if a.currentFocus == "summary" && a.aiSummaryVisible {
				a.toggleAISummary()
				return nil
			}
			// Bulk mode is now handled above, before VIM sequences
			// Check if this might be part of a VIM sequence
			if a.handleVimSequence(event.Rune()) {
				return nil
			}
			// Otherwise, open prompt library picker for single message
			if a.logger != nil {
				a.logger.Printf("keys.go: 'p' pressed in single mode - calling openPromptPicker()")
			}
			go a.openPromptPicker()
			return nil
		case 'm':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('m') {
				if a.currentFocus == "search" {
					return nil
				}
				// Bulk mode is now handled above, before VIM sequences
				// Check if this might be part of a VIM sequence
				if a.handleVimSequence(event.Rune()) {
					return nil
				}
				a.openMovePanel()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'M':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('M') {
				a.toggleMarkdown()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'V':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('V') {
				if a.currentFocus == "search" {
					return nil
				}
				// Toggle RSVP side panel
				if a.currentActivePicker == PickerRSVP {
					if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
						split.ResizeItem(a.labelsView, 0, 0) // Hide RSVP panel
					}
					a.setActivePicker(PickerNone)
					a.restoreFocusAfterModal()
					return nil
				}
				go a.openRSVPModal()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'o':
			// Avoid opening suggestions while advanced search is active
			if a.currentFocus == "search" {
				a.showStatusMessage("🔕 Label suggestions disabled while searching")
				return nil
			}
			// Check if this might be part of a VIM sequence first
			if a.handleVimSequence(event.Rune()) {
				return nil
			}
			go a.suggestLabel()
			return nil
		case 'O': // Shift+O for Obsidian ingestion
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('O') {
				if a.currentFocus == "search" {
					return nil
				}
				// Allow Obsidian ingestion in both normal and bulk modes
				go a.sendEmailToObsidian()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'L': // Shift+L for link picker
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('L') {
				if a.currentFocus == "search" {
					return nil
				}
				// Open link picker for current message
				go a.openLinkPicker()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'w':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('w') {
				go a.saveCurrentMessageToFile()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		case 'W':
			// Only handle if not configured as a configurable shortcut
			if !a.isKeyConfigured('W') {
				go a.saveCurrentMessageRawEML()
				return nil
			}
			// OBLITERATED: redundant break eliminated! 💥
		}

		// ESC exits bulk mode, closes panels, or exits help screen
		if event.Key() == tcell.KeyEscape {
			if a.logger != nil {
				a.logger.Printf("keys: ESC pressed - bulkMode=%v, currentFocus=%s, aiSummaryVisible=%v, streaming=%v, showHelp=%v",
					a.bulkMode, a.currentFocus, a.aiSummaryVisible, a.streamingCancel != nil, a.showHelp)
			}

			// If help screen is showing, close it first
			if a.showHelp {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - closing help screen")
				}
				a.toggleHelp()
				return nil
			}
			// If preload status screen is showing, close it first
			if a.preloadStatusVisible {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - closing preload status screen")
				}
				a.hidePreloadStatus()
				return nil
			}
			// If prompt stats screen is showing, close it first
			if a.promptStatsVisible {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - closing prompt stats screen")
				}
				a.hidePromptStats()
				return nil
			}

			// FIRST: Cancel any active streaming operations (this fixes the hanging issue)
			if a.streamingCancel != nil {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - canceling active streaming operation")
				}
				a.streamingCancel()
				a.streamingCancel = nil

				// After canceling streaming, always hide AI panel if visible
				if a.aiSummaryVisible {
					if a.logger != nil {
						a.logger.Printf("keys: ESC - hiding AI panel after stream cancellation")
					}
					a.hideAIPanel()
					return nil
				}
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

			// If focus is on Slack panel, close it
			if a.currentFocus == "slack" && a.slackVisible {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - hiding Slack panel")
				}
				a.hideSlackPanel()
				return nil
			}

			// If focus is on prompts panel, close it
			if a.currentFocus == "prompts" && a.currentActivePicker != PickerNone {
				if a.logger != nil {
					a.logger.Printf("keys: ESC - closing prompts panel")
				}
				a.closePromptPicker()
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

		// Threading special key handlers
		// Handle Shift+T for thread summary
		if event.Key() == tcell.KeyRune && (event.Modifiers()&tcell.ModShift) != 0 && event.Rune() == 'T' {
			if a.logger != nil {
				a.logger.Printf("Special key: Shift+T -> thread_summary")
			}
			go func() { _ = a.GenerateThreadSummary() }()
			return nil
		}

		// Handle Ctrl+N for next thread
		if (event.Key() == tcell.KeyCtrlN) || ((event.Modifiers()&tcell.ModCtrl) != 0 && event.Rune() == 'n') {
			if a.logger != nil {
				a.logger.Printf("Special key: Ctrl+N -> next_thread")
			}
			// TODO: [THREAD] Implement next thread navigation in conversation view
			go func() {
				a.GetErrorHandler().ShowInfo(a.ctx, "Next thread navigation - not yet implemented")
			}()
			return nil
		}

		// Handle Ctrl+P for previous thread
		if (event.Key() == tcell.KeyCtrlP) || ((event.Modifiers()&tcell.ModCtrl) != 0 && event.Rune() == 'p') {
			if a.logger != nil {
				a.logger.Printf("Special key: Ctrl+P -> prev_thread")
			}
			// TODO: [THREAD] Implement previous thread navigation in conversation view
			go func() {
				a.GetErrorHandler().ShowInfo(a.ctx, "Previous thread navigation - not yet implemented")
			}()
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

		// Handle digit keys for VIM sequences
		if event.Rune() >= '0' && event.Rune() <= '9' {
			if a.logger != nil {
				a.logger.Printf("DIGIT KEY: checking VIM sequence for '%c'", event.Rune())
			}
			if a.handleVimSequence(event.Rune()) {
				if a.logger != nil {
					a.logger.Printf("DIGIT KEY: handled by VIM sequence")
				}
				return nil
			}
			if a.logger != nil {
				a.logger.Printf("DIGIT KEY: not handled by VIM sequence, passing through")
			}
		}

		return event
	})

	// Enter key behavior on list; keep UI-only here
	if table, ok := a.views["list"].(*tview.Table); ok {
		table.SetSelectedFunc(func(row, column int) {
			// Convert table row to message index (subtract 1 for header)
			messageIndex := row - 1
			if messageIndex >= 0 && messageIndex < len(a.ids) {
				// Check if we're in threading mode
				if a.GetCurrentThreadViewMode() == ThreadViewThread {
					// In thread mode, Enter expands/collapses threads
					go func() { _ = a.ExpandThread() }()
				} else {
					// In flat mode, Enter shows the message
					go a.showMessage(a.ids[messageIndex])
				}
			}
		})
		table.SetSelectionChangedFunc(func(row, column int) {
			// Convert table row to message index (subtract 1 for header)
			messageIndex := row - 1
			if messageIndex >= 0 && messageIndex < len(a.ids) {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", messageIndex+1, len(a.ids)))
				id := a.ids[messageIndex]
				go a.showMessageWithoutFocus(id)
				if a.isLabelsPickerActive() {
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
				a.refreshTableDisplay()

				// CRITICAL: Update focus indicators to show list has focus during arrow navigation
				// This was missing - causing visual focus loss even though actual focus stayed on list
				a.updateFocusIndicators("list")

				// Phase 2.4: Background preloading for performance optimization
				go func() {
					defer func() {
						if r := recover(); r != nil {
							// Log panic but don't crash the app
							if a.logger != nil {
								a.logger.Printf("Preloader panic recovered: %v", r)
							}
						}
					}()

					preloader := a.GetPreloaderService()
					if preloader == nil || !preloader.IsEnabled() {
						return
					}

					// 1. Next page preloading: Check if user is near end of current page
					totalMessages := len(a.ids)
					if totalMessages > 0 && preloader.IsNextPageEnabled() {
						threshold := preloader.GetStatus().Config.NextPageThreshold
						if float64(messageIndex+1)/float64(totalMessages) >= threshold {
							// User is at threshold, trigger next page preload
							query := a.currentQuery
							if query == "" && a.searchMode == "remote" {
								query = a.currentQuery
							}
							maxResults := int64(50) // Default page size

							if err := preloader.PreloadNextPage(a.ctx, a.nextPageToken, query, maxResults); err != nil {
								if a.logger != nil {
									a.logger.Printf("Next page preload failed: %v", err)
								}
							}
						}
					}

					// 2. Adjacent message preloading: Preload messages around current selection
					if totalMessages > 0 && preloader.IsAdjacentEnabled() {
						currentMessageID := a.ids[messageIndex]
						if err := preloader.PreloadAdjacentMessages(a.ctx, currentMessageID, a.ids); err != nil {
							if a.logger != nil {
								a.logger.Printf("Adjacent message preload failed: %v", err)
							}
						}
					}
				}()
			}
		})
	}
}

// handleBulkSelect handles bulk selection logic (entering bulk mode or toggling selection)
func (a *App) handleBulkSelect() bool {
	// OBLITERATED: empty logger branch eliminated! 💥
	go func() {
	}()

	if list, ok := a.views["list"].(*tview.Table); ok {
		if !a.bulkMode {
			// OBLITERATED: empty logger branch eliminated! 💥
			a.bulkMode = true
			messageIndex := a.getCurrentSelectedMessageIndex()
			if messageIndex >= 0 {
				if a.selected == nil {
					a.selected = make(map[string]bool)
				}
				a.selected[a.ids[messageIndex]] = true
				// OBLITERATED: empty logger branch eliminated! 💥
			}
			a.refreshTableDisplay()
			// Keep focus highlight consistent (blue) even in Bulk mode
			list.SetSelectedStyle(a.getSelectionStyle())
			// Show status message asynchronously to avoid deadlock
			go func() {
				a.GetErrorHandler().ShowInfo(a.ctx, "Bulk mode — space=select, *=all, a=archive, d=trash, m=move, p=prompt, K=slack, O=obsidian, ESC=exit")
			}()
			return true
		}
		// toggle selection
		messageIndex := a.getCurrentSelectedMessageIndex()
		if messageIndex >= 0 {
			mid := a.ids[messageIndex]
			if a.selected[mid] {
				delete(a.selected, mid)
			} else {
				a.selected[mid] = true
			}
			a.refreshTableDisplay()
			// Show status message asynchronously to avoid deadlock
			go func() {
				a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Selected: %d", len(a.selected)))
			}()
		}
		return true
	}
	return false
}

// toggleFocus switches focus between list and text view
func (a *App) toggleFocus() {
	currentFocus := a.GetFocus()

	// Build focus ring dynamically based on visible components
	ring := []tview.Primitive{}
	ringNames := []string{}
	// 1) Search (if visible)
	if sc, ok := a.views["searchContainer"].(*tview.Flex); ok {
		_, _, w, h := sc.GetRect()
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
	// 3) Text (always) - keep using regular text view for focus ring
	// Only use EnhancedTextView when explicitly needed for content navigation
	ring = append(ring, a.views["text"])
	ringNames = append(ringNames, "text")
	// 4) Labels/Prompts (if visible)
	if a.currentActivePicker != PickerNone {
		ring = append(ring, a.labelsView)
		// Use more specific name based on current focus state
		if a.currentFocus == "prompts" {
			ringNames = append(ringNames, "prompts")
		} else {
			ringNames = append(ringNames, "labels")
		}
	}
	// 5) Summary (if visible)
	if a.aiSummaryVisible {
		ring = append(ring, a.aiSummaryView)
		ringNames = append(ringNames, "summary")
	}
	// 6) Slack (if visible)
	if a.slackVisible {
		ring = append(ring, a.slackView)
		ringNames = append(ringNames, "slack")
	}

	// Find index of current focus
	idx := -1
	for i, p := range ring {
		if currentFocus == p {
			idx = i
			// OBLITERATED: redundant break eliminated! 💥
		}
	}
	// If current focus isn't recognized (e.g., nil), start at beginning
	next := 0
	if idx >= 0 {
		next = (idx + 1) % len(ring)
	}
	// Apply focus and indicators
	targetName := ringNames[next]

	// Special handling for prompt picker - focus will be handled by the container's default behavior
	// The container should automatically focus the first focusable child (input field)
	if targetName == "prompts" {
		a.SetFocus(ring[next])
		a.currentFocus = "prompts"
		a.updateFocusIndicators("prompts")
		return
	}

	a.SetFocus(ring[next])
	a.currentFocus = targetName
	a.updateFocusIndicators(targetName)
}

// restoreFocusAfterModal restores focus to the appropriate view after closing a modal
func (a *App) restoreFocusAfterModal() {
	// Check for special focus overrides (e.g., content search)
	if a.cmdFocusOverride != "" {
		override := a.cmdFocusOverride
		a.cmdFocusOverride = "" // Clear the override

		switch override {
		case "enhanced-text":
			// For content search, focus the EnhancedTextView directly
			if a.enhancedTextView != nil {
				a.SetFocus(a.enhancedTextView)
				a.currentFocus = "text"
				a.updateFocusIndicators("text")
				return
			}
		}
	}

	// Default behavior - restore to list
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// handleVimSequence handles VIM-style key sequences including navigation and range operations
func (a *App) handleVimSequence(key rune) bool {

	// Check if we're in a context where VIM sequences should work
	// Allow VIM sequences when focus is on list or text (for content navigation)
	if a.currentFocus != "" && a.currentFocus != "list" && a.currentFocus != "text" {
		if a.logger != nil {
			a.logger.Printf("handleVimSequence: wrong focus context '%s', returning false", a.currentFocus)
		}
		return false
	}

	now := time.Now()

	// Clear sequence if timeout exceeded (configurable for range operations)
	rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
	if rangeTimeoutMs <= 0 {
		rangeTimeoutMs = 2000 // Default fallback
	}
	if !a.vimTimeout.IsZero() && now.Sub(a.vimTimeout) > time.Duration(rangeTimeoutMs)*time.Millisecond {
		// OBLITERATED: empty logger branch eliminated! 💥
		a.vimSequence = ""
		a.vimOperationType = ""
		a.vimOperationCount = 0
		a.vimOriginalMessageID = ""
		// Clear any status message for cancelled sequence
		go func() {
			a.GetErrorHandler().ClearProgress()
		}()
	}

	// Handle navigation sequences (existing functionality)
	if key == 'g' || key == 'G' {
		return a.handleVimNavigation(key)
	}

	// Handle range operation sequences: {op}, {op}{digits}, {op}{digits}{op}
	result := a.handleVimRangeOperation(key)
	if a.logger != nil {
		a.logger.Printf("handleVimSequence: returning %v", result)
	}
	return result
}

// handleVimNavigation handles traditional VIM navigation (gg, G) - focus-aware
func (a *App) handleVimNavigation(key rune) bool {
	now := time.Now()

	switch key {
	case 'g':
		if a.vimSequence == "g" {
			// Double 'g' - context-dependent behavior
			a.vimSequence = ""
			a.vimTimeout = time.Time{}

			// CRITICAL: Check focus context for gg behavior
			if a.currentFocus == "text" && a.enhancedTextView != nil {
				// Content context: go to top of message content
				if a.logger != nil {
					a.logger.Printf("VIM NAVIGATION: 'gg' in text context - calling EnhancedTextView.gotoTop()")
				}
				a.enhancedTextView.GotoTop()
			} else {
				// List context: go to first message
				if a.logger != nil {
					a.logger.Printf("VIM NAVIGATION: 'gg' in list context - go to first message")
				}
				a.executeGoToFirst()
			}
			return true
		} else {
			// Start of sequence - wait for next key
			a.vimSequence = "g"

			// Use configurable navigation timeout for gg sequence
			navTimeoutMs := a.Keys.VimNavigationTimeoutMs
			if navTimeoutMs <= 0 {
				navTimeoutMs = 1000 // Default fallback
			}
			a.vimTimeout = now.Add(time.Duration(navTimeoutMs) * time.Millisecond)
			return true
		}
	case 'G':
		// Single 'G' - context-dependent behavior
		a.vimSequence = ""
		a.vimTimeout = time.Time{}

		// CRITICAL: Check focus context for G behavior
		if a.currentFocus == "text" && a.enhancedTextView != nil {
			// Content context: go to bottom of message content
			if a.logger != nil {
				a.logger.Printf("VIM NAVIGATION: 'G' in text context - calling EnhancedTextView.gotoBottom()")
			}
			a.enhancedTextView.GotoBottom()
		} else {
			// List context: go to last message
			if a.logger != nil {
				a.logger.Printf("VIM NAVIGATION: 'G' in list context - go to last message")
			}
			a.executeGoToCommand([]string{}) // Use working command function
		}
		return true
	}

	return false
}

// handleVimRangeOperation handles range operations like s5s, a3a, d7d, etc.
func (a *App) handleVimRangeOperation(key rune) bool {
	if a.logger != nil {
		a.logger.Printf("=== handleVimRangeOperation called with key='%c' ===", key)
	}
	now := time.Now()

	// VIM-only operation keys (always handled by VIM) - use dynamic mapping based on config
	vimOnlyOps := map[rune]bool{
		's': true, // select (no conflict)
	}
	// Add configured keys dynamically to prevent hardcoding
	if a.Keys.Archive != "" {
		vimOnlyOps[rune(a.Keys.Archive[0])] = true // archive
	}
	if a.Keys.Trash != "" {
		vimOnlyOps[rune(a.Keys.Trash[0])] = true // delete/trash
	}
	if a.Keys.ToggleRead != "" {
		vimOnlyOps[rune(a.Keys.ToggleRead[0])] = true // toggle read (user's configured key)
	}
	if a.Keys.Move != "" {
		vimOnlyOps[rune(a.Keys.Move[0])] = true // move
	}
	if a.Keys.ManageLabels != "" {
		vimOnlyOps[rune(a.Keys.ManageLabels[0])] = true // label
	}
	if a.Keys.Slack != "" {
		vimOnlyOps[rune(a.Keys.Slack[0])] = true // slack
	}
	if a.Keys.Prompt != "" {
		vimOnlyOps[rune(a.Keys.Prompt[0])] = true // prompt - allow VIM sequences
	}

	// Conflict operation keys (only handled by VIM when in sequence) - use dynamic mapping
	conflictOps := map[rune]bool{}
	// Add configured keys that might conflict with single-key operations
	if a.Keys.Obsidian != "" {
		conflictOps[rune(a.Keys.Obsidian[0])] = true // obsidian
	}

	// All valid operation keys
	validOps := make(map[rune]bool)
	for k, v := range vimOnlyOps {
		validOps[k] = v
	}
	for k, v := range conflictOps {
		validOps[k] = v
	}

	// Handle digits in sequence
	if key >= '0' && key <= '9' {
		if a.vimOperationType != "" {
			// We're building a count: s5 -> s56
			digit := int(key - '0')
			oldCount := a.vimOperationCount
			a.vimOperationCount = a.vimOperationCount*10 + digit
			a.vimSequence += string(key)
			// Use configurable range timeout for digit sequences
			rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
			if rangeTimeoutMs <= 0 {
				rangeTimeoutMs = 2000 // Default fallback
			}
			a.vimTimeout = now.Add(time.Duration(rangeTimeoutMs) * time.Millisecond)

			if a.logger != nil {
				a.logger.Printf("VIM digit pressed: %c, operation: %s, oldCount: %d, digit: %d, newCount: %d", key, a.vimOperationType, oldCount, digit, a.vimOperationCount)
			}

			// Show status
			go func() {
				a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("%s%d... (waiting for operation)", a.vimOperationType, a.vimOperationCount))
			}()
			return true
		}
		return false
	}

	// Handle operation keys
	if !validOps[key] {
		return false
	}

	// For conflict keys, only handle them if we're already in a VIM sequence
	// This allows 'p' and 'o' to work normally for prompts/obsidian when not in VIM mode
	if conflictOps[key] && a.vimOperationType == "" {
		return false
	}

	if a.vimOperationType == "" {
		// Starting new sequence: s, a, d, etc.
		a.vimOperationType = string(key)
		a.vimOperationCount = 0
		a.vimSequence = string(key)
		// Use configurable range timeout for VIM operation sequences
		rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
		if rangeTimeoutMs <= 0 {
			rangeTimeoutMs = 2000 // Default fallback
		}
		a.vimTimeout = now.Add(time.Duration(rangeTimeoutMs) * time.Millisecond)

		// CRITICAL FIX: Capture current message ID when sequence starts
		// This prevents issues where cursor moves during the timeout delay
		a.vimOriginalMessageID = a.GetCurrentMessageID()

		// VIM sequence started

		// Show status and consume the key to prevent single operation
		go func() {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("%s... (enter count then %s, or wait for timeout)", string(key), string(key)))
		}()

		// Start timeout goroutine to execute single operation if no sequence completed
		go func() {
			// OBLITERATED: empty logger branch eliminated! 💥

			// Use configurable range timeout for single operation fallback
			rangeTimeoutMs := a.Keys.VimRangeTimeoutMs
			if rangeTimeoutMs <= 0 {
				rangeTimeoutMs = 2000 // Default fallback
			}
			time.Sleep(time.Duration(rangeTimeoutMs) * time.Millisecond)
			// OBLITERATED: empty logger branch eliminated! 💥
			a.mu.Lock()

			// OBLITERATED: empty logger branch eliminated! 💥

			// Check if sequence is still pending (not completed or cleared)
			if a.vimOperationType == string(key) && a.vimOperationCount == 0 {
				// OBLITERATED: empty logger branch eliminated! 💥
				// Capture original message ID while holding mutex
				originalMessageID := a.vimOriginalMessageID

				// Clear sequence state while holding mutex
				a.vimSequence = ""
				a.vimOperationType = ""
				a.vimTimeout = time.Time{}
				a.vimOriginalMessageID = ""

				// OBLITERATED: empty logger branch eliminated! 💥

				// Release mutex BEFORE accessing UI elements or executing operations
				a.mu.Unlock()

				// OBLITERATED: empty logger branch eliminated! 💥

				go func() {
					a.GetErrorHandler().ClearProgress()
				}()

				// Execute single operation with original message ID (without mutex)
				// OBLITERATED: empty logger branch eliminated! 💥
				a.executeVimSingleOperationWithID(string(key), originalMessageID)
				// OBLITERATED: empty logger branch eliminated! 💥
			} else {
				// OBLITERATED: empty logger branch eliminated! 💥
				a.mu.Unlock()
			}
		}()

		return true // Consume the key to prevent immediate single operation
	} else if a.vimOperationType == string(key) {
		// Completing sequence: s5s, a3a, etc.
		count := a.vimOperationCount
		if count == 0 {
			count = 1 // Default to 1 if no count specified
		}

		if a.logger != nil {
			a.logger.Printf("VIM completing sequence: %s%d%s, passing count=%d to executeVimRangeOperation", string(key), count, string(key), count)
		}

		// Clear sequence state
		operation := a.vimOperationType
		a.vimSequence = ""
		a.vimOperationType = ""
		a.vimOperationCount = 0
		a.vimTimeout = time.Time{}
		a.vimOriginalMessageID = ""

		// Execute the range operation
		a.executeVimRangeOperation(operation, count)
		return true
	}

	if a.logger != nil {
		a.logger.Printf("handleVimRangeOperation: no pattern matched, returning false")
	}
	return false
}

// executeVimRangeOperation executes a VIM range operation
func (a *App) executeVimRangeOperation(operation string, count int) {
	if a.logger != nil {
		a.logger.Printf("executeVimRangeOperation: operation=%s, count=%d", operation, count)
	}

	// Get current message index
	messageIndex := a.getCurrentSelectedMessageIndex()
	if messageIndex < 0 {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Validate range doesn't exceed available messages
	maxCount := len(a.ids) - messageIndex
	if count > maxCount {
		count = maxCount
		go func() {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Range limited to %d available messages", count))
		}()
	}

	// Use dynamic operation mapping based on configured keys
	switch operation {
	case a.Keys.BulkSelect:
		a.selectRange(messageIndex, count)
	case a.Keys.Archive:
		a.archiveRange(messageIndex, count)
	case a.Keys.Trash:
		a.trashRange(messageIndex, count)
	case a.Keys.ToggleRead:
		a.toggleReadRange(messageIndex, count)
	case a.Keys.Move:
		a.moveRange(messageIndex, count)
	case a.Keys.ManageLabels:
		a.labelRange(messageIndex, count)
	case a.Keys.Slack:
		a.slackRange(messageIndex, count)
	case a.Keys.Obsidian:
		a.obsidianRange(messageIndex, count)
	case a.Keys.Prompt:
		a.promptRange(messageIndex, count)
	default:
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Unknown VIM operation: %s", operation))
	}
}

// executeVimSingleOperation executes a single operation when VIM timeout occurs
func (a *App) executeVimSingleOperation(operation string) {
	// Use dynamic operation mapping based on configured keys
	switch operation {
	case a.Keys.BulkSelect:
		// For single bulk select key, just highlight current message (no bulk mode)
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "Message selected")
		}()
	case a.Keys.Archive:
		// Archive current message
		go a.archiveSelected()
	case a.Keys.Trash:
		// Trash current message
		go a.trashSelected()
	case a.Keys.ToggleRead:
		// Toggle read status of current message
		go a.toggleMarkReadUnread()
	case a.Keys.Move:
		// Move current message
		a.openMovePanel()
	case a.Keys.ManageLabels:
		// CRITICAL: Check for bulk mode first - VIM 'l' should respect bulk selection
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("VIM BULK FIX: 'l' key with bulk mode - %d selected messages, calling manageLabelsBulk()", len(a.selected))
			}
			a.manageLabelsBulk()
		} else {
			// Show labels dialog for current message
			a.manageLabels()
		}
	case a.Keys.Slack:
		// Show Slack dialog for current message
		go a.showSlackForwardDialog()
	case a.Keys.Obsidian:
		// Send current message to Obsidian
		go a.sendEmailToObsidian()
	case a.Keys.Prompt:
		// Show prompts dialog for current message
		go a.openPromptPicker()
	}
}

// executeVimSingleOperationWithID executes a single operation with a specific message ID
// This is used when VIM timeout occurs to ensure operation applies to the original message
func (a *App) executeVimSingleOperationWithID(operation string, messageID string) {
	// OBLITERATED: empty logger branch eliminated! 💥

	if messageID == "" {
		// OBLITERATED: empty logger branch eliminated! 💥
		// Fallback to current message if no ID provided
		a.executeVimSingleOperation(operation)
		return
	}

	// OBLITERATED: empty logger branch eliminated! 💥

	switch operation {
	case a.Keys.BulkSelect:
		// For single bulk select key, just show a message (no actual operation needed)
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "Message selected")
		}()
	case a.Keys.Archive:
		// CRITICAL: Check for bulk mode first - VIM 'a' should respect bulk selection
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("VIM BULK FIX: 'a' key with bulk mode - %d selected messages, calling archiveSelectedBulk()", len(a.selected))
			}
			go a.archiveSelectedBulk()
		} else {
			// Archive specific message - temporarily set current ID
			go func() {
				a.SetCurrentMessageID(messageID)
				a.archiveSelected()
			}()
		}
	case a.Keys.Trash:
		// CRITICAL: Check for bulk mode first - VIM 'd' should respect bulk selection
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("VIM BULK FIX: 'd' key with bulk mode - %d selected messages, calling trashSelectedBulk()", len(a.selected))
			}
			go a.trashSelectedBulk()
		} else {
			// Trash specific message by ID (bypasses current selection) - single message only
			// OBLITERATED: empty logger branch eliminated! 💥
			a.trashSelectedByID(messageID)
			// OBLITERATED: empty logger branch eliminated! 💥
		}
	case a.Keys.ToggleRead:
		// CRITICAL: Check for bulk mode first - VIM 't' should respect bulk selection
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("VIM BULK FIX: 't' key with bulk mode - %d selected messages, calling toggleMarkReadUnreadBulk()", len(a.selected))
			}
			go a.toggleMarkReadUnreadBulk()
		} else {
			// Toggle read status of specific message - temporarily set current ID
			go func() {
				a.SetCurrentMessageID(messageID)
				a.toggleMarkReadUnread()
			}()
		}
	case a.Keys.Move:
		// Move specific message - temporarily set current ID
		// OBLITERATED: empty logger branch eliminated! 💥
		go func() {
			a.SetCurrentMessageID(messageID)
			// OBLITERATED: empty logger branch eliminated! 💥
			a.openMovePanel()
		}()
	case a.Keys.ManageLabels:
		// CRITICAL: Check for bulk mode first - VIM 'l' should respect bulk selection
		if a.bulkMode && len(a.selected) > 0 {
			if a.logger != nil {
				a.logger.Printf("VIM BULK FIX: 'l' key with bulk mode - %d selected messages, calling manageLabelsBulk()", len(a.selected))
			}
			a.manageLabelsBulk()
		} else {
			// Show labels dialog for specific message
			go func() {
				currentID := a.GetCurrentMessageID()
				a.SetCurrentMessageID(messageID)
				a.manageLabels()
				// Labels dialog is modal, so we can restore after
				a.SetCurrentMessageID(currentID)
			}()
		}
	case a.Keys.Slack:
		// Show Slack dialog for specific message
		go func() {
			currentID := a.GetCurrentMessageID()
			a.SetCurrentMessageID(messageID)
			a.showSlackForwardDialog()
			a.SetCurrentMessageID(currentID)
		}()
	case a.Keys.Obsidian:
		// Send specific message to Obsidian
		go func() {
			currentID := a.GetCurrentMessageID()
			a.SetCurrentMessageID(messageID)
			a.sendEmailToObsidian()
			a.SetCurrentMessageID(currentID)
		}()
	case a.Keys.Prompt:
		// Show prompts dialog for specific message
		go func() {
			currentID := a.GetCurrentMessageID()
			a.SetCurrentMessageID(messageID)
			a.openPromptPicker()
			a.SetCurrentMessageID(currentID)
		}()
	}
}

// Removed unused function: isVimNavigationActive

// VIM Range Operation Functions

// selectRange selects a range of messages starting from startIndex
func (a *App) selectRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Enter bulk mode if not already in it
	if !a.bulkMode {
		a.bulkMode = true
		if a.selected == nil {
			a.selected = make(map[string]bool)
		}
	}

	// Select the range of messages
	selected := 0
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageID := a.ids[startIndex+i]
		a.selected[messageID] = true
		selected++
	}

	// Update UI
	a.refreshTableDisplay()
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetSelectedStyle(a.getSelectionStyle())
	}

	// Show status
	go func() {
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Selected %d messages (s%ds)", selected, count))
	}()
}

// archiveRange archives a range of messages starting from startIndex
func (a *App) archiveRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	// Show progress
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Archiving %d messages (a%da)...", actualCount, count))
	}()

	// Archive in background using bulk service for proper undo recording
	go func() {
		// Use bulk archive service method for proper undo recording
		emailService, _, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
		err := emailService.BulkArchive(a.ctx, messageIDs)

		// Clear progress and show result
		a.GetErrorHandler().ClearProgress()
		if err == nil {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Archived %d messages (a%da)", actualCount, count))
		} else {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to archive some messages (a%da): %v", count, err))
		}

		// Remove archived messages from current view (no server reload needed)
		a.QueueUpdateDraw(func() {
			a.removeIDsFromCurrentList(messageIDs)
			a.refreshTableDisplay()
		})
	}()
}

// trashRange moves a range of messages to trash starting from startIndex
func (a *App) trashRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	// Show progress
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Moving %d messages to trash (d%dd)...", actualCount, count))
	}()

	// Trash in background using bulk service for proper undo recording
	go func() {
		// Use bulk trash service method for proper undo recording
		emailService, _, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
		err := emailService.BulkTrash(a.ctx, messageIDs)

		// Clear progress and show result
		a.GetErrorHandler().ClearProgress()
		if err == nil {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Moved %d messages to trash (d%dd)", actualCount, count))
		} else {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to trash some messages (d%dd): %v", count, err))
		}

		// Remove trashed messages from current view (no server reload needed)
		a.QueueUpdateDraw(func() {
			a.removeIDsFromCurrentList(messageIDs)
			a.refreshTableDisplay()
		})
	}()
}

// toggleReadRange toggles read status for a range of messages starting from startIndex
func (a *App) toggleReadRange(startIndex, count int) {
	if a.logger != nil {
		a.logger.Printf("toggleReadRange called: startIndex=%d, count=%d", startIndex, count)
	}

	if count <= 0 {
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	if a.logger != nil {
		a.logger.Printf("toggleReadRange: actualCount=%d, count=%d, will show (%s%d%s)", actualCount, count, a.Keys.ToggleRead, count, a.Keys.ToggleRead)
	}

	// Show progress with correct VIM sequence display
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Toggling read status for %d messages (%s%d%s)...", actualCount, a.Keys.ToggleRead, count, a.Keys.ToggleRead))
	}()

	// Toggle read status in background
	go func() {
		_, _, _, _, repository, _, _, _, _, _, _, _ := a.GetServices()
		undoService := a.GetUndoService()

		// Create a single comprehensive undo action for the entire range operation
		// We need to capture the state of all messages before performing any operations
		prevStates := make(map[string]services.ActionState)
		unreadIDs := make([]string, 0)
		// OBLITERATED: unused readIDs eliminated! 💥

		// Capture state and separate messages by current read status
		for i, messageID := range messageIDs {
			// Capture current state for undo
			if undoServiceImpl, ok := undoService.(*services.UndoServiceImpl); ok {
				if prevState, err := undoServiceImpl.CaptureMessageState(a.ctx, messageID); err == nil {
					prevStates[messageID] = prevState
				}
			}

			// Determine current read status by checking message meta
			isUnread := false
			messageIndex := startIndex + i
			if messageIndex < len(a.messagesMeta) && a.messagesMeta[messageIndex] != nil {
				for _, labelID := range a.messagesMeta[messageIndex].LabelIds {
					if labelID == "UNREAD" {
						isUnread = true
						// OBLITERATED: redundant break eliminated! 💥
					}
				}
			}

			if isUnread {
				unreadIDs = append(unreadIDs, messageID)
			}
			// OBLITERATED: unused readIDs append eliminated! 💥
		}

		failed := 0

		// Perform toggle operations using repository directly and record combined undo action
		for i, messageID := range messageIDs {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Toggling read %d/%d messages...", i+1, actualCount))

			// Determine if this message was unread or read
			isUnread := false
			for _, unreadID := range unreadIDs {
				if unreadID == messageID {
					isUnread = true
					// OBLITERATED: redundant break eliminated! 💥
				}
			}

			var err error
			if isUnread {
				// Mark as read (remove UNREAD label)
				updates := services.MessageUpdates{
					RemoveLabels: []string{"UNREAD"},
				}
				err = repository.UpdateMessage(a.ctx, messageID, updates)
				if err == nil {
					// Update local cache to remove UNREAD label
					a.updateCachedMessageLabels(messageID, "UNREAD", false)
				}
			} else {
				// Mark as unread (add UNREAD label)
				updates := services.MessageUpdates{
					AddLabels: []string{"UNREAD"},
				}
				err = repository.UpdateMessage(a.ctx, messageID, updates)
				if err == nil {
					// Update local cache to add UNREAD label
					a.updateCachedMessageLabels(messageID, "UNREAD", true)
				}
			}

			if err != nil {
				if a.logger != nil {
					a.logger.Printf("toggleReadRange: Failed to toggle message %s: %v", messageID, err)
				}
				failed++
			}
		}

		// Record a single undo action for the entire toggle range operation after all operations complete
		if len(prevStates) > 0 && failed < len(messageIDs) {
			// Create a custom undo action that will restore each message to its previous state
			// We'll use UndoActionMarkRead as the type, but with custom logic in the undo service
			action := &services.UndoableAction{
				Type:        services.UndoActionMarkRead, // We'll modify the undo logic to handle this case
				MessageIDs:  messageIDs,
				PrevState:   prevStates,
				Description: fmt.Sprintf("Toggle read status (%s%d%s)", a.Keys.ToggleRead, count, a.Keys.ToggleRead),
				IsBulk:      true,
				ExtraData: map[string]interface{}{
					"operation_type": "toggle_read", // Custom flag to indicate this is a toggle operation
				},
			}
			_ = undoService.RecordAction(a.ctx, action)
		}

		// Clear progress and show result
		a.GetErrorHandler().ClearProgress()
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Toggled read status for %d messages (%s%d%s)", actualCount, a.Keys.ToggleRead, count, a.Keys.ToggleRead))
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Toggled read status for %d messages, %d failed (%s%d%s)", actualCount-failed, failed, a.Keys.ToggleRead, count, a.Keys.ToggleRead))
		}

		// Update display to show updated read status (no server reload needed)
		a.QueueUpdateDraw(func() {
			a.refreshTableDisplay()
		})
	}()
}

// moveRange opens move panel for a range of messages starting from startIndex
func (a *App) moveRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	// Enter bulk mode and select the messages
	a.bulkMode = true
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}

	// Clear previous selection and select the range
	a.selected = make(map[string]bool)
	for _, id := range messageIDs {
		a.selected[id] = true
	}

	// Update UI to show selection
	a.refreshTableDisplay()
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetSelectedStyle(a.getSelectionStyle())
	}

	// Show status and open move panel
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("📁 Selected %d messages for move operation (%s%d%s)", actualCount, a.Keys.Move, count, a.Keys.Move))
	}()

	// Open bulk move panel
	a.openMovePanelBulk()
}

// labelRange opens label panel for a range of messages starting from startIndex
func (a *App) labelRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	// Enter bulk mode and select the messages
	a.bulkMode = true
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}

	// Clear previous selection and select the range
	a.selected = make(map[string]bool)
	for _, id := range messageIDs {
		a.selected[id] = true
	}

	// Update UI to show selection
	a.refreshTableDisplay()
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetSelectedStyle(a.getSelectionStyle())
	}

	// Show status and open labels panel
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("🔖 Selected %d messages for labeling (l%dl)", actualCount, count))
	}()

	// Open labels panel (which will work in bulk mode)
	go a.manageLabels()
}

// slackRange sends a range of messages to Slack starting from startIndex
func (a *App) slackRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Check if Slack is enabled
	if !a.Config.Slack.Enabled {
		a.GetErrorHandler().ShowError(a.ctx, "Slack integration is not enabled in configuration")
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	// Enter bulk mode and select the messages
	a.bulkMode = true
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}

	// Clear previous selection and select the range
	a.selected = make(map[string]bool)
	for _, id := range messageIDs {
		a.selected[id] = true
	}

	// Update UI to show selection
	a.refreshTableDisplay()
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetSelectedStyle(a.getSelectionStyle())
	}

	// Show status and open Slack bulk panel
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("💬 Selected %d messages for Slack forwarding (k%dk)", actualCount, count))
	}()

	// Open Slack bulk forwarding panel
	go a.showSlackBulkForwardDialog()
}

// obsidianRange sends a range of messages to Obsidian starting from startIndex
func (a *App) obsidianRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	// Enter bulk mode and select the messages
	a.bulkMode = true
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}

	// Clear previous selection and select the range
	a.selected = make(map[string]bool)
	for _, id := range messageIDs {
		a.selected[id] = true
	}

	// Update UI to show selection
	a.refreshTableDisplay()
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetSelectedStyle(a.getSelectionStyle())
	}

	// Show status and send to Obsidian
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("📝 Sending %d messages to Obsidian (o%do)", actualCount, count))
	}()

	// Send to Obsidian (which handles bulk mode automatically)
	go a.sendEmailToObsidian()
}

// promptRange applies AI prompts to a range of messages starting from startIndex
func (a *App) promptRange(startIndex, count int) {
	if count <= 0 {
		return
	}

	// Get message IDs for the range
	messageIDs := make([]string, 0, count)
	for i := 0; i < count && startIndex+i < len(a.ids); i++ {
		messageIDs = append(messageIDs, a.ids[startIndex+i])
	}

	actualCount := len(messageIDs)

	// Enter bulk mode and select the messages
	a.bulkMode = true
	if a.selected == nil {
		a.selected = make(map[string]bool)
	}

	// Clear previous selection and select the range
	a.selected = make(map[string]bool)
	for _, id := range messageIDs {
		a.selected[id] = true
	}

	// Update UI to show selection
	a.refreshTableDisplay()
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetSelectedStyle(a.getSelectionStyle())
	}

	// Show status and open bulk prompt picker
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("🤖 Selected %d messages for AI prompting (%s%d%s)", actualCount, a.Keys.Prompt, count, a.Keys.Prompt))
	}()

	// Open bulk prompt picker
	go a.openBulkPromptPicker()
}

// triggerPreloadingForMessage triggers background preloading for a specific message index
func (a *App) triggerPreloadingForMessage(messageIndex int) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Log panic but don't crash the app
				if a.logger != nil {
					a.logger.Printf("Preloader panic recovered: %v", r)
				}
			}
		}()

		preloader := a.GetPreloaderService()
		if preloader == nil || !preloader.IsEnabled() {
			return
		}

		// Validate message index
		if messageIndex < 0 || messageIndex >= len(a.ids) {
			return
		}

		// 1. Next page preloading: Check if user is near end of current page
		totalMessages := len(a.ids)
		if totalMessages > 0 && preloader.IsNextPageEnabled() {
			threshold := preloader.GetStatus().Config.NextPageThreshold
			currentPosition := float64(messageIndex+1) / float64(totalMessages)
			if a.logger != nil {
				a.logger.Printf("PRELOAD MANUAL: message %d/%d (%.2f%%) vs threshold %.2f%%, nextPageToken='%s'",
					messageIndex+1, totalMessages, currentPosition*100, threshold*100, a.nextPageToken)
			}
			if currentPosition >= threshold {
				// User is at threshold, trigger next page preload
				query := a.currentQuery
				if query == "" && a.searchMode == "remote" {
					query = a.currentQuery
				}
				maxResults := int64(50) // Default page size

				if err := preloader.PreloadNextPage(a.ctx, a.nextPageToken, query, maxResults); err != nil {
					if a.logger != nil {
						a.logger.Printf("PRELOAD MANUAL: Next page preload failed: %v", err)
					}
				} else {
					if a.logger != nil {
						a.logger.Printf("PRELOAD MANUAL: Next page preload initiated successfully")
					}
				}
			}
		}

		// 2. Adjacent message preloading: Preload messages around current selection
		if totalMessages > 0 && preloader.IsAdjacentEnabled() {
			currentMessageID := a.ids[messageIndex]
			if a.logger != nil {
				a.logger.Printf("PRELOAD ADJACENT: triggering for message %d (ID: %s), total messages: %d",
					messageIndex+1, currentMessageID, totalMessages)
			}
			if err := preloader.PreloadAdjacentMessages(a.ctx, currentMessageID, a.ids); err != nil {
				if a.logger != nil {
					a.logger.Printf("PRELOAD ADJACENT: failed: %v", err)
				}
			} else {
				if a.logger != nil {
					a.logger.Printf("PRELOAD ADJACENT: initiated successfully for message %d", messageIndex+1)
				}
			}
		}
	}()
}
