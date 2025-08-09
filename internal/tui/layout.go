package tui

import (
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// initComponents initializes the main UI components
func (a *App) initComponents() {
	// Create main list component as Table to support per-row colors
	list := tview.NewTable().SetSelectable(true, false)
	list.SetBorder(true).
		SetBorderColor(tcell.ColorBlue).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📧 Messages ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)

		// Create header view (colored) and main text view inside a column container
	header := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	header.SetBorder(false)
	header.SetTextColor(tcell.ColorGreen)

	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	text.SetBorder(false)

	textContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	textContainer.SetBorder(true).
		SetBorderColor(tcell.ColorYellow).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📄 Message Content ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)
	// Fixed height for header
	textContainer.AddItem(header, 4, 0, false)
	textContainer.AddItem(text, 0, 1, false)

	// Create AI Summary view (hidden by default)
	ai := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	ai.SetBorder(true).
		SetBorderColor(tcell.ColorYellow).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🤖 AI Summary ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)

	// Store components
	a.views["list"] = list
	a.views["text"] = text
	a.views["header"] = header
	a.views["textContainer"] = textContainer
	a.aiSummaryView = ai

	// Labels contextual panel container (hidden by default)
	labelsFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	labelsFlex.SetBorder(true).
		SetBorderColor(tcell.ColorGray).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🏷️ Labels ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)
	a.labelsView = labelsFlex
}

// initViews initializes the main views
func (a *App) initViews() {
	// Create main layout
	mainLayout := a.createMainLayout()

	// Add main view
	a.Pages.AddPage("main", mainLayout, true, true)

	// Add help view
	helpView := a.createHelpView()
	a.Pages.AddPage("help", helpView, true, false)

	// Add search view
	searchView := a.createSearchView()
	a.Pages.AddPage("search", searchView, true, false)

	// Initialize focus indicators
	a.updateFocusIndicators("list")
}

// createMainLayout creates the main application layout
func (a *App) createMainLayout() tview.Primitive {
	// Create the main flex container (vertical layout - one below the other)
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add flash notification at the top (hidden by default)
	mainFlex.AddItem(a.flash.textView, 0, 0, false)

	// Add list of messages (takes 40% of available height)
	mainFlex.AddItem(a.views["list"], 0, 40, true)

	// Message content row: split into content | AI summary (hidden initially)
	contentSplit := tview.NewFlex().SetDirection(tview.FlexColumn)
	contentSplit.AddItem(a.views["textContainer"], 0, 1, false)
	contentSplit.AddItem(a.aiSummaryView, 0, 0, false) // weight 0 = hidden
	contentSplit.AddItem(a.labelsView, 0, 0, false)    // hidden by default
	a.views["contentSplit"] = contentSplit
	// Add message content (takes 40% of available height)
	mainFlex.AddItem(contentSplit, 0, 40, false)

	// Add command bar (hidden by default, appears when : is pressed)
	cmdBar := a.createCommandBar()
	mainFlex.AddItem(cmdBar, 1, 0, false)

	// Add status bar at the bottom
	statusBar := a.createStatusBar()
	a.views["status"] = statusBar // Store status bar as a view
	mainFlex.AddItem(statusBar, 1, 0, false)

	return mainFlex
}

// updateFocusIndicators updates the visual indicators for the focused view
func (a *App) updateFocusIndicators(focusedView string) {
	// Reset all borders to default
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetBorderColor(tcell.ColorGray)
	}
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetBorderColor(tcell.ColorGray)
	}
	if a.aiSummaryView != nil {
		a.aiSummaryView.SetBorderColor(tcell.ColorGray)
	}
	if a.labelsView != nil {
		a.labelsView.SetBorderColor(tcell.ColorGray)
	}

	// Set focused view border to bright color
	switch focusedView {
	case "list":
		if list, ok := a.views["list"].(*tview.Table); ok {
			list.SetBorderColor(tcell.ColorYellow)
		}
	case "text":
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetBorderColor(tcell.ColorYellow)
		}
	case "summary":
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetBorderColor(tcell.ColorYellow)
		}
	case "labels":
		if a.labelsView != nil {
			a.labelsView.SetBorderColor(tcell.ColorYellow)
		}
	}
}

// createStatusBar creates the status bar
func (a *App) createStatusBar() tview.Primitive {
	status := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText("Gmail TUI | Press ? for help | Press q to quit")

	return status
}

// createHelpView creates the help view
func (a *App) createHelpView() tview.Primitive {
	help := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetScrollable(true)

	help.SetText(a.generateHelpText())
	return help
}

// createSearchView creates the search view
func (a *App) createSearchView() tview.Primitive {
	searchInput := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(50).
		SetPlaceholder("Enter search terms (e.g., from:user@example.com, subject:meeting)").
		SetPlaceholderTextColor(tcell.ColorGray)

	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			query := strings.TrimSpace(searchInput.GetText())
			if query != "" {
				go a.performSearch(query)
			}
			a.Pages.SwitchToPage("main")
			a.SetFocus(a.views["list"])
		} else if key == tcell.KeyEscape {
			a.Pages.SwitchToPage("main")
			a.SetFocus(a.views["list"])
		}
	})
	return searchInput
}
