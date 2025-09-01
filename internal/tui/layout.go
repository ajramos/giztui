package tui

import (
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// ForceFilledBorderFlex forces a Flex container to use a filled background by replacing
// its internal Box with a fresh Box (dontClear=false) and reapplying styling.
// This ensures borders appear solid like Table borders instead of hollow.
func ForceFilledBorderFlex(f *tview.Flex) {
	// Only process Flex containers that have borders enabled
	// We need to check the Box's border state by looking at the current styling
	// Since Flex embeds Box, we can access Box methods directly

	// Store current styling before replacing the Box
	backgroundColor := f.GetBackgroundColor()
	borderColor := f.GetBorderColor()
	borderAttributes := f.GetBorderAttributes()
	title := f.GetTitle()
	// For title color and alignment, we'll use the current theme values
	// since these are not directly accessible from the Box

	// Replace the internal Box with a fresh one that has dontClear=false
	// This ensures the Flex will clear/fill its background area like Table does
	f.Box = tview.NewBox()

	// Reapply all the styling to the new Box
	f.SetBackgroundColor(backgroundColor)
	f.SetBorder(true)
	f.SetBorderColor(borderColor)
	f.SetBorderAttributes(borderAttributes)
	f.SetTitle(title)
	// Note: Title color and alignment will be set by the calling code
	// since we can't retrieve them from the original Box
}

// initComponents initializes the main UI components
func (a *App) initComponents() {
	// Create main list component as Table to support per-row colors
	list := tview.NewTable().SetSelectable(true, false)
	list.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	list.SetBorder(true).
		SetBorderColor(a.GetComponentColors("general").Border.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📧 Messages ").
		SetTitleColor(a.GetComponentColors("general").Title.Color()).
		SetTitleAlign(tview.AlignCenter)
		// Search panel placeholder (hidden by default)
	searchPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	searchPanel.SetBorder(false)
	searchPanel.SetBackgroundColor(a.GetComponentColors("general").Background.Color())

	ForceFilledBorderFlex(searchPanel)

	// Container that holds search panel (top) and list (bottom)
	listContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	listContainer.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	// Start hidden: panel proportion 0; list takes all
	listContainer.AddItem(searchPanel, 0, 0, false)
	listContainer.AddItem(list, 0, 1, true)

	// Create header view (colored) and main text view inside a column container
	header := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	header.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	header.SetBorder(false)
	header.SetTextColor(a.getMessageHeaderColor()) // Use theme header color for email message headers

	enhancedText := NewEnhancedTextView(a)
	text := enhancedText.TextView
	text.SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	text.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	text.SetBorder(false)

	// Store the enhanced text view in the app
	a.enhancedTextView = enhancedText

	textContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	textContainer.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	textContainer.SetBorder(true).
		SetBorderColor(a.GetComponentColors("general").Border.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📄 Message Content ").
		SetTitleColor(a.GetComponentColors("general").Title.Color()).
		SetTitleAlign(tview.AlignCenter)
	// Force filled background for consistent border rendering
	ForceFilledBorderFlex(textContainer)
	// Reapply title styling since the helper can't preserve it
	textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color()).SetTitleAlign(tview.AlignCenter)

	// Fixed height for header (room for Subject, From, To, Cc, Date, Labels)
	textContainer.AddItem(header, 6, 0, false)
	textContainer.AddItem(text, 0, 1, false)

	// Create AI Summary view (hidden by default)
	ai := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	ai.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	ai.SetBorder(true).
		SetBorderColor(a.GetComponentColors("general").Background.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🤖 AI Summary ").
		SetTitleColor(a.GetComponentColors("ai").Title.Color()).
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
	labelsFlex.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	labelsFlex.SetBorder(true).
		SetBorderColor(a.GetComponentColors("general").Background.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🏷️ Labels ").
		SetTitleColor(a.GetComponentColors("labels").Title.Color()).
		SetTitleAlign(tview.AlignCenter)
	// Force filled background for consistent border rendering
	ForceFilledBorderFlex(labelsFlex)
	// Reapply title styling since the helper can't preserve it
	labelsFlex.SetTitleColor(a.GetComponentColors("labels").Title.Color()).SetTitleAlign(tview.AlignCenter)
	a.labelsView = labelsFlex

	// Slack contextual panel container (hidden by default)
	slackFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	slackFlex.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	slackFlex.SetBorder(true).
		SetBorderColor(a.GetComponentColors("general").Background.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 💬 Send to Slack channel ").
		SetTitleColor(a.GetComponentColors("slack").Title.Color()).
		SetTitleAlign(tview.AlignCenter)
	// Force filled background for consistent border rendering
	ForceFilledBorderFlex(slackFlex)
	// Reapply title styling since the helper can't preserve it
	slackFlex.SetTitleColor(a.GetComponentColors("slack").Title.Color()).SetTitleAlign(tview.AlignCenter)
	a.slackView = slackFlex

	// Composition panel (hidden by default)
	a.compositionPanel = NewCompositionPanel(a)

	// Command panel (hidden by default)
	cmdPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	cmdPanel.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	cmdPanel.SetBorder(true).
		SetBorderColor(a.GetComponentColors("general").Background.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🐶 Command ").
		SetTitleColor(a.GetComponentColors("general").Title.Color()).
		SetTitleAlign(tview.AlignCenter)
	// Force filled background for consistent border rendering
	ForceFilledBorderFlex(cmdPanel)
	// Reapply title styling since the helper can't preserve it
	cmdPanel.SetTitleColor(a.GetComponentColors("general").Title.Color()).SetTitleAlign(tview.AlignCenter)
	a.views["cmdPanel"] = cmdPanel

	// Search container (hidden by default)
	searchContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	// Use hierarchical theme system for search container
	searchColors := a.GetComponentColors("search")
	searchContainer.SetBackgroundColor(searchColors.Background.Color())
	searchContainer.SetBorder(true).
		SetBorderColor(searchColors.Border.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🔍 Search ").
		SetTitleColor(searchColors.Title.Color()).
		SetTitleAlign(tview.AlignCenter)
	// Force filled background for consistent border rendering
	ForceFilledBorderFlex(searchContainer)
	// Reapply title styling since the helper can't preserve it
	searchContainer.SetTitleColor(searchColors.Title.Color()).SetTitleAlign(tview.AlignCenter)
	a.views["searchContainer"] = searchContainer

	// Advanced search container (hidden by default)
	advancedSearchContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	// Use hierarchical theme system for advanced search container
	advancedSearchColors := a.GetComponentColors("search")
	advancedSearchContainer.SetBackgroundColor(advancedSearchColors.Background.Color())
	advancedSearchContainer.SetBorder(true).
		SetBorderColor(advancedSearchColors.Border.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🔍 Advanced Search ").
		SetTitleColor(advancedSearchColors.Title.Color()).
		SetTitleAlign(tview.AlignCenter)
	// Force filled background for consistent border rendering
	ForceFilledBorderFlex(advancedSearchContainer)
	// Reapply title styling since the helper can't preserve it
	advancedSearchContainer.SetTitleColor(advancedSearchColors.Title.Color()).SetTitleAlign(tview.AlignCenter)
	a.views["advancedSearchContainer"] = advancedSearchContainer
}

// initViews initializes the main views
func (a *App) initViews() {
	// Add a background page that paints the full-screen background color.
	// This works around tview containers not painting their own border areas.
	background := tview.NewBox().SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	a.Pages.AddPage("background", background, true, true)

	// Create main layout
	mainLayout := a.createMainLayout()

	// Add main view
	a.Pages.AddPage("main", mainLayout, true, true)

	// Help is now displayed in message content area, no separate page needed

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

	mainFlex.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	// Add flash notification at the top (hidden by default)
	mainFlex.AddItem(a.flash.textView, 0, 0, false)

	// Search containers mounted hidden (height 0). They will be resized when opened.
	if sc, ok := a.views["searchContainer"]; ok {
		mainFlex.AddItem(sc, 0, 0, false)
	}
	if asc, ok := a.views["advancedSearchContainer"]; ok {
		mainFlex.AddItem(asc, 0, 0, false)
	}

	// Add list+search container (takes 40% of available height)
	mainFlex.AddItem(a.views["listContainer"], 0, 40, true)

	// Message content row: split into content | AI summary (hidden initially)
	contentSplit := tview.NewFlex().SetDirection(tview.FlexColumn)
	contentSplit.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	contentSplit.AddItem(a.views["textContainer"], 0, 1, false)
	contentSplit.AddItem(a.aiSummaryView, 0, 0, false) // weight 0 = hidden
	contentSplit.AddItem(a.labelsView, 0, 0, false)    // hidden by default
	contentSplit.AddItem(a.slackView, 0, 0, false)     // hidden by default
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

// createCompositionLayoutWithStatus creates a composition layout that preserves the status bar
func (a *App) createCompositionLayoutWithStatus() tview.Primitive {
	// Create a vertical flex container for composition + status bar
	compositionLayout := tview.NewFlex().SetDirection(tview.FlexRow)
	compositionLayout.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
	
	// Add composition panel (takes most of the screen)
	if a.compositionPanel != nil {
		compositionLayout.AddItem(a.compositionPanel, 0, 1, true) // flexible, focusable
	}
	
	// Add status bar (fixed height at bottom)
	if statusBar, exists := a.views["status"]; exists {
		compositionLayout.AddItem(statusBar, 1, 0, false) // fixed height, not focusable
	}
	
	return compositionLayout
}

// updateFocusIndicators updates the visual indicators for the focused view
func (a *App) updateFocusIndicators(focusedView string) {
	// Reset all borders to theme's default border color
	unfocusedColor := a.GetComponentColors("general").Border.Color() // Use theme's border color
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetBorderColor(unfocusedColor)
	}
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetBorderColor(unfocusedColor)
	}
	if a.aiSummaryView != nil {
		a.aiSummaryView.SetBorderColor(unfocusedColor)
	}
	if a.labelsView != nil {
		a.labelsView.SetBorderColor(unfocusedColor)
	}
	if a.slackView != nil {
		a.slackView.SetBorderColor(unfocusedColor)
	}
	if tc, ok := a.views["textContainer"].(*tview.Flex); ok {
		// Subtle unfocused border to keep contour visible but not contrasty
		tc.SetBorderColor(unfocusedColor)
	}
	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		sp.SetBorderColor(unfocusedColor)
	}
	if cp, ok := a.views["cmdPanel"].(*tview.Flex); ok {
		cp.SetBorderColor(unfocusedColor)
	}
	if sc, ok := a.views["searchContainer"].(*tview.Flex); ok {
		sc.SetBorderColor(unfocusedColor)
	}

	// Set focused view border to theme's focus color
	focusedColor := a.GetComponentColors("general").Accent.Color() // Use theme's focus color
	switch focusedView {
	case "list":
		if list, ok := a.views["list"].(*tview.Table); ok {
			list.SetBorderColor(focusedColor)
		}
	case "text":
		if tc, ok := a.views["textContainer"].(*tview.Flex); ok {
			tc.SetBorderColor(focusedColor)
		}
	case "summary":
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetBorderColor(focusedColor)
		}
	case "labels":
		if a.labelsView != nil {
			a.labelsView.SetBorderColor(focusedColor)
		}
	case "slack":
		if a.slackView != nil {
			a.slackView.SetBorderColor(focusedColor)
		}
	case "prompts":
		if a.labelsView != nil {
			a.labelsView.SetBorderColor(focusedColor)
		}
	case "search":
		if sc, ok := a.views["searchContainer"].(*tview.Flex); ok {
			sc.SetBorderColor(focusedColor)
		}
		if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
			sp.SetBorderColor(focusedColor)
		}
	case "cmd":
		if cp, ok := a.views["cmdPanel"].(*tview.Flex); ok {
			cp.SetBorderColor(focusedColor)
		}
	case "obsidian":
		if a.labelsView != nil {
			a.labelsView.SetBorderColor(focusedColor)
		}
	case "drafts":
		if a.labelsView != nil {
			a.labelsView.SetBorderColor(focusedColor)
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

// Help is now displayed directly in message content area

// createSearchView creates the search view
func (a *App) createSearchView() tview.Primitive {
	searchInput := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(50).
		SetPlaceholder("Enter search terms (e.g., from:user@example.com, subject:meeting)")

	// Apply consistent theme colors
	a.ConfigureInputFieldTheme(searchInput, "simple")

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

// RefreshBordersForFilledFlexes refreshes the background and border colors
// for all Flex containers that have been forced to use filled backgrounds.
// This should be called when themes change to ensure consistent styling.
func (a *App) RefreshBordersForFilledFlexes() {
	// Update textContainer
	if tc, ok := a.views["textContainer"].(*tview.Flex); ok {
		tc.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
		tc.SetBorderColor(a.GetComponentColors("general").Background.Color())
	}

	// Update labelsFlex
	if a.labelsView != nil {
		a.labelsView.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
		a.labelsView.SetBorderColor(a.GetComponentColors("general").Background.Color())
	}

	// Update slackFlex
	if a.slackView != nil {
		a.slackView.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
		a.slackView.SetBorderColor(a.GetComponentColors("general").Background.Color())
	}

	// Update cmdPanel
	if cp, ok := a.views["cmdPanel"].(*tview.Flex); ok {
		cp.SetBackgroundColor(a.GetComponentColors("general").Background.Color())
		cp.SetBorderColor(a.GetComponentColors("general").Background.Color())
	}

	// Update searchContainer with hierarchical theme system
	if sc, ok := a.views["searchContainer"].(*tview.Flex); ok {
		searchColors := a.GetComponentColors("search")
		sc.SetBackgroundColor(searchColors.Background.Color())
		sc.SetBorderColor(searchColors.Border.Color())
		sc.SetTitleColor(searchColors.Title.Color())
	}

	// Update advancedSearchContainer with hierarchical theme system
	if asc, ok := a.views["advancedSearchContainer"].(*tview.Flex); ok {
		advancedSearchColors := a.GetComponentColors("search")
		asc.SetBackgroundColor(advancedSearchColors.Background.Color())
		asc.SetBorderColor(advancedSearchColors.Border.Color())
		asc.SetTitleColor(advancedSearchColors.Title.Color())
	}
}
