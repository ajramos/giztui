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

		// Only intercept specific keys, let navigation keys pass through
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
			a.manageLabels()
			return nil
		case 'm':
			a.moveSelected()
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
