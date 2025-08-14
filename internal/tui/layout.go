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
	list.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	list.SetBorder(true).
		SetBorderColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📧 Messages ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)
		// Search panel placeholder (hidden by default)
	searchPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	searchPanel.SetBorder(false)
	searchPanel.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	// Container that holds search panel (top) and list (bottom)
	listContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	listContainer.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	// Start hidden: panel proportion 0; list takes all
	listContainer.AddItem(searchPanel, 0, 0, false)
	listContainer.AddItem(list, 0, 1, true)

	// Create header view (colored) and main text view inside a column container
	header := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	header.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	header.SetBorder(false)
	header.SetTextColor(tcell.ColorGreen)

	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	text.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	text.SetBorder(false)

	textContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	textContainer.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	textContainer.SetBorder(true).
		SetBorderColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📄 Message Content ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)

	// Fixed height for header (room for Subject, From, To, Cc, Date, Labels)
	textContainer.AddItem(header, 6, 0, false)
	textContainer.AddItem(text, 0, 1, false)

	// Create AI Summary view (hidden by default)
	ai := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	ai.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	ai.SetBorder(true).
		SetBorderColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🤖 AI Summary ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)

		// Store components
	a.views["list"] = list
	a.views["searchPanel"] = searchPanel
	a.views["listContainer"] = listContainer
	a.views["text"] = text
	a.views["header"] = header
	a.views["textContainer"] = textContainer
	a.aiSummaryView = ai

	// Labels contextual panel container (hidden by default)
	labelsFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	labelsFlex.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	labelsFlex.SetBorder(true).
		SetBorderColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🏷️ Labels ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)
	a.labelsView = labelsFlex

	// Command panel (hidden by default)
	cmdPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	cmdPanel.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	cmdPanel.SetBorder(true).
		SetBorderColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🐶 Command ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)
	a.views["cmdPanel"] = cmdPanel
}

// initViews initializes the main views
func (a *App) initViews() {
	// Add a background page that paints the full-screen background color.
	// This works around tview containers not painting their own border areas.
	background := tview.NewBox().SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	a.Pages.AddPage("background", background, true, true)

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
	// Allow scrolling the text view when focused
	if tv, ok := a.views["text"].(*tview.TextView); ok {
		tv.SetChangedFunc(func() {}) // ensure internal init
		tv.SetScrollable(true)
	}
}

// createMainLayout creates the main application layout
func (a *App) createMainLayout() tview.Primitive {
	// Create the main flex container (vertical layout - one below the other)
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	mainFlex.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	// Add flash notification at the top (hidden by default)
	mainFlex.AddItem(a.flash.textView, 0, 0, false)

	// Add list+search container (takes 40% of available height)
	mainFlex.AddItem(a.views["listContainer"], 0, 40, true)

	// Message content row: split into content | AI summary (hidden initially)
	contentSplit := tview.NewFlex().SetDirection(tview.FlexColumn)
	contentSplit.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	contentSplit.AddItem(a.views["textContainer"], 0, 1, false)
	contentSplit.AddItem(a.aiSummaryView, 0, 0, false) // weight 0 = hidden
	contentSplit.AddItem(a.labelsView, 0, 0, false)    // hidden by default
	a.views["contentSplit"] = contentSplit
	// Add message content (takes 40% of available height)
	mainFlex.AddItem(contentSplit, 0, 40, false)

	// Command panel mounted hidden (height 0). It will be resized when opened.
	if cp, ok := a.views["cmdPanel"]; ok {
		mainFlex.AddItem(cp, 0, 0, false)
	}

	// Add status bar at the bottom
	statusBar := a.createStatusBar()
	a.views["status"] = statusBar // Store status bar as a view
	mainFlex.AddItem(statusBar, 1, 0, false)

	// Store reference for dynamic resize (e.g., advanced search)
	a.views["mainFlex"] = mainFlex
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
	if tc, ok := a.views["textContainer"].(*tview.Flex); ok {
		// Subtle unfocused border to keep contour visible but not contrasty
		tc.SetBorderColor(tcell.ColorGray)
	}
	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		sp.SetBorderColor(tcell.ColorGray)
	}
	if cp, ok := a.views["cmdPanel"].(*tview.Flex); ok {
		cp.SetBorderColor(tcell.ColorGray)
	}

	// Set focused view border to bright color
	switch focusedView {
	case "list":
		if list, ok := a.views["list"].(*tview.Table); ok {
			list.SetBorderColor(tcell.ColorYellow)
		}
	case "text":
		if tc, ok := a.views["textContainer"].(*tview.Flex); ok {
			tc.SetBorderColor(tcell.ColorYellow)
		}
	case "summary":
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetBorderColor(tcell.ColorYellow)
		}
	case "labels":
		if a.labelsView != nil {
			a.labelsView.SetBorderColor(tcell.ColorYellow)
		}
	case "prompts":
		if a.labelsView != nil {
			a.labelsView.SetBorderColor(tcell.ColorYellow)
		}
	case "search":
		if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
			sp.SetBorderColor(tcell.ColorYellow)
		}
	case "cmd":
		if cp, ok := a.views["cmdPanel"].(*tview.Flex); ok {
			cp.SetBorderColor(tcell.ColorYellow)
		}
	}
}

// createStatusBar creates the status bar
func (a *App) createStatusBar() tview.Primitive {
	status := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(a.statusBaseline())

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
