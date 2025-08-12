package tui

import (
	"fmt"

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
					a.setStatusPersistent("Bulk mode — space=select, *=all, a=archive, d=trash, m=move, ESC=exit")
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
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
					a.setStatusPersistent(fmt.Sprintf("Selected: %d", len(a.selected)))
				}
				return nil
			}
		case 'b':
			// Toggle bulk mode with 'b'
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
					a.setStatusPersistent("Bulk mode — space/b=select, *=all, a=archive, d=trash, m=move, ESC=exit")
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite))
				} else {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.reformatListItems()
					a.setStatusPersistent("")
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
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
					a.setStatusPersistent(fmt.Sprintf("Selected: %d", len(a.selected)))
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
			// Toggle contextual labels panel
			a.manageLabels()
			return nil
		case 'm':
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
			go a.suggestLabel()
			return nil
		case 'w':
			go a.saveCurrentMessageToFile()
			return nil
		}

		// ESC exits bulk mode
		if event.Key() == tcell.KeyEscape {
			if a.bulkMode {
				a.bulkMode = false
				a.selected = make(map[string]bool)
				a.reformatListItems()
				a.setStatusPersistent("")
				if list, ok := a.views["list"].(*tview.Table); ok {
					list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
				}
				return nil
			}
			// If a search is active and overlay is not focused, delegate to exitSearch
			if a.searchMode != "" {
				go a.exitSearch()
				return nil
			}
		}

		// Advanced search (Ctrl+F) global binding
		if (event.Key() == tcell.KeyCtrlF) || ((event.Modifiers()&tcell.ModCtrl) != 0 && event.Rune() == 'f') {
			a.openAdvancedSearchForm()
			return nil
		}

		// Focus toggle
		if event.Key() == tcell.KeyTab {
			a.toggleFocus()
			return nil
		}
		// If a picker/list or advanced search form field has focus, do not handle runes globally
		switch a.GetFocus().(type) {
		case *tview.InputField, *tview.List:
			return event
		}

		// LLM features
		if a.LLM != nil {
			switch event.Rune() {
			case 'y':
				a.toggleAISummary()
				return nil
			case 'g':
				a.showStatusMessage("💬 Generate reply: placeholder")
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
				if a.aiSummaryVisible {
					go a.generateOrShowSummary(id)
				}
				a.currentMessageID = id
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
