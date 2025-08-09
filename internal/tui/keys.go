package tui

import (
	"fmt"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// bindKeys sets up keyboard shortcuts and routes actions to feature modules
func (a *App) bindKeys() {
	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Command mode routing
		if a.cmdMode {
			return a.handleCommandInput(event)
		}

		// If focus is on an input field (e.g., label search), don't intercept runes
		switch a.GetFocus().(type) {
		case *tview.InputField:
			return event
		}

		// Only intercept specific keys, let navigation keys pass through
		// Ensure arrow keys navigate the currently focused pane, not the list always
		// tview handles arrow keys per focused primitive, so we avoid overriding them here.
		switch event.Rune() {
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
			a.Pages.SwitchToPage("search")
			a.SetFocus(a.views["search"])
			return nil
		case 'u':
			go a.listUnreadMessages()
			return nil
		case 't':
			go a.toggleMarkReadUnread()
			return nil
		case 'd':
			go a.trashSelected()
			return nil
		case 'a':
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
		case 'l':
			// Toggle contextual labels panel
			a.manageLabels()
			return nil
		case 'm':
			// Open move panel directly in browse-all mode
			a.openMovePanel()
			return nil
		case 'M':
			a.toggleMarkdown()
			return nil
		case 'o':
			go a.suggestLabel()
			return nil
		}

		// Focus toggle
		if event.Key() == tcell.KeyTab {
			a.toggleFocus()
			return nil
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
	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			if index < len(a.ids) {
				go a.showMessage(a.ids[index])
			}
		})
		list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			if index >= 0 && index < len(a.ids) {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", index+1, len(a.ids)))
				id := a.ids[index]
				// Update message content without changing focus
				go a.showMessageWithoutFocus(id)
				// If labels pane is visible, refresh its content for the new message
				if a.labelsVisible {
					// No cambiar foco: solo refrescar contenido de labels
					go a.populateLabelsQuickView(id)
				}
				a.currentMessageID = id
			}
		})
	}
}

// toggleFocus switches focus between list and text view
func (a *App) toggleFocus() {
	currentFocus := a.GetFocus()

	if currentFocus == a.views["list"] {
		a.SetFocus(a.views["text"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	} else if currentFocus == a.views["text"] {
		// Cycle: text -> labels (if visible) -> summary (if visible) -> list
		if a.labelsVisible {
			a.SetFocus(a.labelsView)
			a.currentFocus = "labels"
			a.updateFocusIndicators("labels")
		} else if a.aiSummaryVisible {
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		} else {
			a.SetFocus(a.views["list"])
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
		}
	} else if a.labelsVisible && currentFocus == a.labelsView {
		if a.aiSummaryVisible {
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		} else {
			a.SetFocus(a.views["list"])
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
		}
	} else if a.aiSummaryVisible && currentFocus == a.aiSummaryView {
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	} else {
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	}
}

// restoreFocusAfterModal restores focus to the appropriate view after closing a modal
func (a *App) restoreFocusAfterModal() {
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}
