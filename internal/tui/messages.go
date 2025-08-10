package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// reformatListItems recalculates list item strings for current screen width
func (a *App) reformatListItems() {
	table, ok := a.views["list"].(*tview.Table)
	if !ok || len(a.ids) == 0 {
		return
	}
	for i := range a.ids {
		if i >= len(a.messagesMeta) || a.messagesMeta[i] == nil {
			continue
		}
		msg := a.messagesMeta[i]
		text, _ := a.emailRenderer.FormatEmailList(msg, a.screenWidth)

		// Label flags
		unread := false
		starred := false
		yellowStar := false
		important := false
		for _, l := range msg.LabelIds {
			switch l {
			case "UNREAD":
				unread = true
			case "STARRED":
				starred = true
			case "YELLOW_STAR":
				yellowStar = true
			case "IMPORTANT":
				important = true
			}
		}

		// Determine base color by priority (as tcell.Color for Table)
		var textColor tcell.Color = tcell.ColorWhite
		if yellowStar {
			textColor = tcell.ColorYellow
		} else if starred {
			textColor = tcell.ColorGreen
		} else if important {
			textColor = tcell.ColorRed
		} else if !unread { // read
			textColor = tcell.ColorGray
		}

		// Build prefixes
		var prefix string
		if a.bulkMode {
			if a.selected != nil && a.selected[a.ids[i]] {
				prefix = "☑ "
			} else {
				prefix = "☐ "
			}
		}
		if unread {
			prefix += "● "
		} else {
			prefix += "○ "
		}

		// Build cell with explicit colors
		final := prefix + text
		cell := tview.NewTableCell(final).
			SetExpansion(1).
			SetAlign(tview.AlignLeft)
		if a.bulkMode && a.selected != nil && a.selected[a.ids[i]] {
			cell.SetTextColor(tcell.ColorBlack).SetBackgroundColor(tcell.ColorWhite)
		} else {
			cell.SetTextColor(textColor).SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		}
		table.SetCell(i, 0, cell)
	}
}

// reloadMessages loads messages from the inbox
func (a *App) reloadMessages() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.draftMode = false
	if table, ok := a.views["list"].(*tview.Table); ok {
		table.Clear()
	}
	a.ids = []string{}
	a.messagesMeta = []*gmailapi.Message{}

	// Show loading message
	if table, ok := a.views["list"].(*tview.Table); ok {
		table.SetTitle(" 🔄 Loading messages... ")
	}
	a.Draw()

	// If coming from remote search mode, clear it on full reload
	if a.searchMode == "remote" {
		a.searchMode = ""
		a.currentQuery = ""
		a.nextPageToken = ""
	}

	// Check if client is available
	if a.Client == nil {
		a.showError("❌ Gmail client not initialized")
		return
	}

	messages, next, err := a.Client.ListMessagesPage(50, "")
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading messages: %v", err))
		return
	}
	a.nextPageToken = next

	// Show success message if no messages
	if len(messages) == 0 {
		a.QueueUpdateDraw(func() {
			if table, ok := a.views["list"].(*tview.Table); ok {
				table.SetTitle(" 📧 No messages found ")
			}
		})
		a.showInfo("📧 No messages found in your inbox")
		return
	}

	// Usar ancho disponible actual del list (simple, sin watchers)
	screenWidth := a.getFormatWidth()

	// Process messages using the email renderer
	for i, msg := range messages {
		a.ids = append(a.ids, msg.Id)

		// Get only metadata, not full content
		message, err := a.Client.GetMessage(msg.Id)
		if err != nil {
			if table, ok := a.views["list"].(*tview.Table); ok {
				table.SetCell(i, 0, tview.NewTableCell(fmt.Sprintf("⚠️  Error loading message %d", i+1)))
			}
			continue
		}

		// Use the email renderer to format the message
		formattedText, _ := a.emailRenderer.FormatEmailList(message, screenWidth)

		// Add unread indicator
		unread := false
		for _, labelId := range message.LabelIds {
			if labelId == "UNREAD" {
				unread = true
				break
			}
		}

		if unread {
			formattedText = "● " + formattedText
		} else {
			formattedText = "○ " + formattedText
		}

		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetCell(i, 0, tview.NewTableCell(formattedText).SetExpansion(1))
		}

		// cache meta for resize re-rendering
		a.messagesMeta = append(a.messagesMeta, message)

		// Update title periodically
		if (i+1)%10 == 0 {
			if table, ok := a.views["list"].(*tview.Table); ok {
				table.SetTitle(fmt.Sprintf(" 🔄 Loading... (%d/%d) ", i+1, len(messages)))
			}
			a.Draw()
		}
	}

	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
			// Ensure a sane initial selection
			r, _ := table.GetSelection()
			if table.GetRowCount() > 0 && r < 0 {
				table.Select(0, 0)
			}
		}
		// Apply per-row colors after initial load
		a.reformatListItems()
	})

	// Do not steal focus if user moved to another pane (e.g., labels/summary/text)
	if pageName, _ := a.Pages.GetFrontPage(); pageName == "main" {
		if a.currentFocus == "" || a.currentFocus == "list" {
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
			a.SetFocus(a.views["list"])
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
		}
	}
}

// loadMoreMessages fetches the next page of inbox and appends to list
func (a *App) loadMoreMessages() {
	// If in remote search mode, paginate that query
	if a.searchMode == "remote" {
		if a.nextPageToken == "" {
			a.showStatusMessage("No more results")
			return
		}
		a.setStatusPersistent("Loading more results…")
		messages, next, err := a.Client.SearchMessagesPage(a.currentQuery, 50, a.nextPageToken)
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading more: %v", err))
			return
		}
		a.appendMessages(messages)
		a.nextPageToken = next
		return
	}

	if a.nextPageToken == "" {
		a.showStatusMessage("No more messages")
		return
	}
	a.setStatusPersistent("Loading next 50 messages…")
	messages, next, err := a.Client.ListMessagesPage(50, a.nextPageToken)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading more: %v", err))
		return
	}
	// Append
	screenWidth := a.getFormatWidth()
	for _, msg := range messages {
		a.ids = append(a.ids, msg.Id)
		meta, err := a.Client.GetMessage(msg.Id)
		if err != nil {
			continue
		}
		a.messagesMeta = append(a.messagesMeta, meta)
		// Set placeholder cell; colors will be applied by reformatListItems below
		if table, ok := a.views["list"].(*tview.Table); ok {
			row := table.GetRowCount()
			text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
			table.SetCell(row, 0, tview.NewTableCell(text).SetExpansion(1))
		}
	}
	a.nextPageToken = next
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		}
		a.reformatListItems()
	})
}

// appendMessages adds messages to current table from a slice of gmail.Message (IDs)
func (a *App) appendMessages(messages []*gmailapi.Message) {
	screenWidth := a.getFormatWidth()
	for _, msg := range messages {
		a.ids = append(a.ids, msg.Id)
		meta, err := a.Client.GetMessage(msg.Id)
		if err != nil {
			continue
		}
		a.messagesMeta = append(a.messagesMeta, meta)
		if table, ok := a.views["list"].(*tview.Table); ok {
			row := table.GetRowCount()
			text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
			table.SetCell(row, 0, tview.NewTableCell(text).SetExpansion(1))
		}
	}
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		}
		a.reformatListItems()
	})
}

// openSearchOverlay opens a transient overlay above the message list for remote/local search
func (a *App) openSearchOverlay(mode string) {
	if mode != "remote" && mode != "local" {
		mode = "remote"
	}
	title := "🔍 Gmail Search"
	if mode == "local" {
		title = "🔎 Local Filter"
	}

	input := tview.NewInputField().
		SetLabel("").
		SetFieldWidth(0).
		SetPlaceholder("e.g., from:user@domain.com subject:report is:unread or plain text for local")
	// expose input so Tab from list can focus it
	a.views["searchInput"] = input
	help := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	help.SetTextColor(tcell.ColorGray)
	if mode == "remote" {
		help.SetText("Pulsa Ctrl+F para búsqueda avanzada | Enter=buscar, Ctrl-T=cambiar, ESC=cancelar")
	} else {
		help.SetText("Pulsa Ctrl+F para búsqueda avanzada | Enter=aplicar, Ctrl-T=cambiar, ESC=limpiar")
	}

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).SetTitle(title).SetTitleColor(tcell.ColorYellow)
	// vertical center input; place help at bottom
	topSpacer := tview.NewBox()
	bottomSpacer := tview.NewBox()
	box.AddItem(topSpacer, 0, 1, false)
	box.AddItem(input, 1, 0, true)
	box.AddItem(bottomSpacer, 0, 2, false)
	box.AddItem(help, 1, 0, false)

	// Capture Enter/ESC and Ctrl-T to toggle modes
	curMode := mode
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			query := input.GetText()
			if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
				lc.Clear()
				if sp, ok2 := a.views["searchPanel"].(*tview.Flex); ok2 {
					sp.SetBorder(false)
					sp.SetTitle("")
				}
				lc.AddItem(a.views["searchPanel"], 0, 0, false)
				lc.AddItem(a.views["list"], 0, 1, true)
			} else {
				a.Pages.RemovePage("searchOverlay")
			}
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
			a.SetFocus(a.views["list"])
			if curMode == "remote" {
				go a.performSearch(query)
			} else {
				a.localFilter = query
				go a.applyLocalFilter(query)
			}
			delete(a.views, "searchInput")
		}
		if key == tcell.KeyEscape {
			if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
				lc.Clear()
				if sp, ok2 := a.views["searchPanel"].(*tview.Flex); ok2 {
					sp.SetBorder(false)
					sp.SetTitle("")
				}
				lc.AddItem(a.views["searchPanel"], 0, 0, false)
				lc.AddItem(a.views["list"], 0, 1, true)
			} else {
				a.Pages.RemovePage("searchOverlay")
			}
			// If we were in remote search, ESC resets to inbox
			if a.searchMode != "" {
				go a.reloadMessages()
				a.searchMode = ""
				a.currentQuery = ""
				a.localFilter = ""
				a.nextPageToken = ""
			}
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
			a.SetFocus(a.views["list"])
			delete(a.views, "searchInput")
		}
	})
	input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Modifiers() == tcell.ModCtrl && ev.Rune() == 't' {
			if curMode == "remote" {
				curMode = "local"
				box.SetTitle("🔎 Local Filter")
				help.SetText("Tokens: from: subject: label: is:unread|starred|important text… | Enter=apply, Ctrl-T=switch, ESC=clear")
				if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
					sp.SetTitle("🔎 Local Filter")
				}
			} else {
				curMode = "remote"
				box.SetTitle("🔍 Gmail Search")
				help.SetText("Operators: from: to: subject: label: has:attachment is:unread older_than:7d newer_than:1d | Enter=search, Ctrl-T=switch, ESC=cancel")
				if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
					sp.SetTitle("🔍 Gmail Search")
				}
			}
			return nil
		}
		if ev.Key() == tcell.KeyTab {
			// move focus back to list while keeping search open
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
			a.SetFocus(a.views["list"])
			return nil
		}
		if ev.Key() == tcell.KeyEscape {
			// mirror ESC handling here to ensure consistent behavior
			if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
				lc.Clear()
				if sp, ok2 := a.views["searchPanel"].(*tview.Flex); ok2 {
					sp.SetBorder(false)
					sp.SetTitle("")
				}
				lc.AddItem(a.views["searchPanel"], 0, 0, false)
				lc.AddItem(a.views["list"], 0, 1, true)
			} else {
				a.Pages.RemovePage("searchOverlay")
			}
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
			a.SetFocus(a.views["list"])
			delete(a.views, "searchInput")
			return nil
		}
		return ev
	})

	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		// vertical center input; place help at bottom
		topSpacer := tview.NewBox()
		bottomSpacer := tview.NewBox()
		sp.Clear()
		sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle(title).SetTitleColor(tcell.ColorYellow)
		sp.AddItem(topSpacer, 0, 1, false)
		sp.AddItem(input, 1, 0, true)
		sp.AddItem(bottomSpacer, 0, 2, false)
		sp.AddItem(help, 1, 0, false)
		if lc, ok2 := a.views["listContainer"].(*tview.Flex); ok2 {
			lc.Clear()
			lc.AddItem(a.views["searchPanel"], 0, 1, true)
			lc.AddItem(a.views["list"], 0, 3, true)
		}
		a.currentFocus = "search"
		a.updateFocusIndicators("search")
		a.SetFocus(input)
		return
	}
	a.Pages.AddPage("searchOverlay", box, true, true)
	a.SetFocus(input)
}

// openAdvancedSearchForm shows a guided form to compose a Gmail query, splitting the list area
func (a *App) openAdvancedSearchForm() {
	// Build form fields
	form := tview.NewForm().
		AddInputField("From", "", 0, nil, nil).
		AddInputField("To", "", 0, nil, nil).
		AddInputField("Subject", "", 0, nil, nil).
		AddInputField("Label", "", 0, nil, nil).
		AddCheckbox("Unread", false, nil).
		AddCheckbox("Starred", false, nil).
		AddCheckbox("Important", false, nil).
		AddInputField("Has: attachment (yes/no)", "", 0, nil, nil).
		AddInputField("Older than (e.g., 7d)", "", 0, nil, nil).
		AddInputField("Newer than (e.g., 1d)", "", 0, nil, nil)
	form.SetButtonsAlign(tview.AlignRight)
	form.AddButton("Search", func() {
		// Collect values
		from := form.GetFormItemByLabel("From").(*tview.InputField).GetText()
		to := form.GetFormItemByLabel("To").(*tview.InputField).GetText()
		subject := form.GetFormItemByLabel("Subject").(*tview.InputField).GetText()
		label := form.GetFormItemByLabel("Label").(*tview.InputField).GetText()
		unread := form.GetFormItemByLabel("Unread").(*tview.Checkbox).IsChecked()
		starred := form.GetFormItemByLabel("Starred").(*tview.Checkbox).IsChecked()
		important := form.GetFormItemByLabel("Important").(*tview.Checkbox).IsChecked()
		hasAttach := strings.TrimSpace(form.GetFormItemByLabel("Has: attachment (yes/no)").(*tview.InputField).GetText())
		older := strings.TrimSpace(form.GetFormItemByLabel("Older than (e.g., 7d)").(*tview.InputField).GetText())
		newer := strings.TrimSpace(form.GetFormItemByLabel("Newer than (e.g., 1d)").(*tview.InputField).GetText())

		// Build Gmail query
		parts := []string{}
		if from != "" {
			parts = append(parts, fmt.Sprintf("from:%s", from))
		}
		if to != "" {
			parts = append(parts, fmt.Sprintf("to:%s", to))
		}
		if subject != "" {
			parts = append(parts, fmt.Sprintf("subject:%q", subject))
		}
		if label != "" {
			parts = append(parts, fmt.Sprintf("label:%s", label))
		}
		if unread {
			parts = append(parts, "is:unread")
		}
		if starred {
			parts = append(parts, "is:starred")
		}
		if important {
			parts = append(parts, "is:important")
		}
		if strings.EqualFold(hasAttach, "yes") || strings.EqualFold(hasAttach, "y") || strings.EqualFold(hasAttach, "true") {
			parts = append(parts, "has:attachment")
		}
		if older != "" {
			parts = append(parts, fmt.Sprintf("older_than:%s", older))
		}
		if newer != "" {
			parts = append(parts, fmt.Sprintf("newer_than:%s", newer))
		}
		q := strings.Join(parts, " ")

		// Close panel and run search
		if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
			lc.Clear()
			lc.AddItem(a.views["searchPanel"], 0, 0, false)
			lc.AddItem(a.views["list"], 0, 1, true)
		}
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])
		go a.performSearch(q)
	})
	form.AddButton("Cancel", func() {
		if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
			lc.Clear()
			lc.AddItem(a.views["searchPanel"], 0, 0, false)
			lc.AddItem(a.views["list"], 0, 1, true)
		}
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])
	})
	form.SetBorder(true).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)

	// Mount form into searchPanel area splitting list (top form, bottom list)
	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		sp.Clear()
		sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)
		sp.AddItem(form, 0, 1, true)
		if lc, ok2 := a.views["listContainer"].(*tview.Flex); ok2 {
			lc.Clear()
			lc.AddItem(a.views["searchPanel"], 0, 1, true)
			lc.AddItem(a.views["list"], 0, 3, false)
		}
		a.currentFocus = "search"
		a.updateFocusIndicators("search")
		a.SetFocus(form)
		return
	}

	// Fallback modal if searchPanel not present
	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	modal.SetBorder(true).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)
	modal.AddItem(form, 0, 1, true)
	a.Pages.AddPage("advancedSearch", modal, true, true)
	a.SetFocus(form)
}

// applyLocalFilter filters current in-memory messages based on a simple expression
func (a *App) applyLocalFilter(expr string) {
	a.searchMode = "local"
	a.localFilter = expr
	tokens := strings.Fields(strings.ToLower(expr))
	if table, ok := a.views["list"].(*tview.Table); ok {
		table.Clear()
		count := 0
		for _, m := range a.messagesMeta {
			if m == nil {
				continue
			}
			text, _ := a.emailRenderer.FormatEmailList(m, a.getFormatWidth())
			content := strings.ToLower(text)
			match := true
			for _, t := range tokens {
				if !strings.Contains(content, t) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
			table.SetCell(count, 0, tview.NewTableCell(text).SetExpansion(1))
			count++
		}
		table.SetTitle(fmt.Sprintf(" 🔎 Filter (%d) — %s ", count, expr))
		a.reformatListItems()
	}
}

// showMessage displays a message in the text view
func (a *App) showMessage(id string) {
	// Show loading message immediately
	if text, ok := a.views["text"].(*tview.TextView); ok {
		if a.debug {
			a.logger.Printf("showMessage: id=%s", id)
		}
		text.SetText("Loading message...")
		text.ScrollToBeginning()
	}

	// Automatically switch focus to text view when viewing a message
	a.SetFocus(a.views["text"])
	a.currentFocus = "text"
	a.updateFocusIndicators("text")
	a.currentMessageID = id

	a.Draw()

	// Load message content in background
	go func() {
		if a.debug {
			a.logger.Printf("showMessage background: id=%s", id)
		}
		// Use cache if available; otherwise fetch and cache
		var message *gmail.Message
		if cached, ok := a.messageCache[id]; ok {
			if a.debug {
				a.logger.Printf("showMessage: cache hit id=%s", id)
			}
			message = cached
		} else {
			m, err := a.Client.GetMessageWithContent(id)
			if err != nil {
				a.showError(fmt.Sprintf("❌ Error loading message: %v", err))
				return
			}
			if a.debug {
				a.logger.Printf("showMessage: fetched id=%s", id)
			}
			a.messageCache[id] = m
			message = m
		}

		rendered, isANSI := a.renderMessageContent(message)

		// Update UI in main thread
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				if isANSI {
					// Convert ANSI → tview markup while writing
					fmt.Fprint(tview.ANSIWriter(text, "", ""), rendered)
				} else {
					text.SetText(rendered)
				}
				// Scroll to the top of the text
				text.ScrollToBeginning()
			}
			// If AI pane is visible, refresh summary for this message
			if a.aiSummaryVisible {
				a.generateOrShowSummary(id)
			}
		})
	}()
}

// showMessageWithoutFocus loads the message content but does not change focus
func (a *App) showMessageWithoutFocus(id string) {
	// Show loading message
	if text, ok := a.views["text"].(*tview.TextView); ok {
		if a.debug {
			a.logger.Printf("showMessageWithoutFocus: id=%s", id)
		}
		text.SetText("Loading message...")
		text.ScrollToBeginning()
	}
	a.currentMessageID = id

	go func() {
		if a.debug {
			a.logger.Printf("showMessageWithoutFocus background: id=%s", id)
		}
		// Use cache if available; otherwise fetch and cache
		var message *gmail.Message
		if cached, ok := a.messageCache[id]; ok {
			if a.debug {
				a.logger.Printf("showMessageWithoutFocus: cache hit id=%s", id)
			}
			message = cached
		} else {
			m, err := a.Client.GetMessageWithContent(id)
			if err != nil {
				a.showError(fmt.Sprintf("❌ Error loading message: %v", err))
				return
			}
			if a.debug {
				a.logger.Printf("showMessageWithoutFocus: fetched id=%s", id)
			}
			a.messageCache[id] = m
			message = m
		}

		rendered, isANSI := a.renderMessageContent(message)

		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				if isANSI {
					fmt.Fprint(tview.ANSIWriter(text, "", ""), rendered)
				} else {
					text.SetText(rendered)
				}
				text.ScrollToBeginning()
			}
		})
	}()
}

// refreshMessageContent reloads the message and updates the text view without changing focus
func (a *App) refreshMessageContent(id string) {
	if id == "" {
		return
	}
	go func() {
		if a.debug {
			a.logger.Printf("refreshMessageContent: id=%s", id)
		}
		// Prefer cached message to avoid re-fetching on toggles
		var m *gmail.Message
		if cached, ok := a.messageCache[id]; ok {
			if a.debug {
				a.logger.Printf("refreshMessageContent: cache hit id=%s", id)
			}
			m = cached
		} else {
			fetched, err := a.Client.GetMessageWithContent(id)
			if err != nil {
				return
			}
			if a.debug {
				a.logger.Printf("refreshMessageContent: fetched id=%s", id)
			}
			a.messageCache[id] = fetched
			m = fetched
		}
		rendered, isANSI := a.renderMessageContent(m)
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				if isANSI {
					fmt.Fprint(tview.ANSIWriter(text, "", ""), rendered)
				} else {
					text.SetText(rendered)
				}
				text.ScrollToBeginning()
			}
		})
	}()
}

// refreshMessageContentWithOverride reloads message and overrides labels shown with provided names
func (a *App) refreshMessageContentWithOverride(id string, labelsOverride []string) {
	if id == "" {
		return
	}
	go func() {
		m, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			return
		}
		// Merge override labels
		if len(labelsOverride) > 0 {
			seen := make(map[string]struct{}, len(m.Labels)+len(labelsOverride))
			merged := make([]string, 0, len(m.Labels)+len(labelsOverride))
			for _, l := range m.Labels {
				if _, ok := seen[l]; !ok {
					seen[l] = struct{}{}
					merged = append(merged, l)
				}
			}
			for _, l := range labelsOverride {
				if _, ok := seen[l]; !ok {
					seen[l] = struct{}{}
					merged = append(merged, l)
				}
			}
			m.Labels = merged
		}
		rendered, isANSI := a.renderMessageContent(m)
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				if isANSI {
					fmt.Fprint(tview.ANSIWriter(text, "", ""), rendered)
				} else {
					text.SetText(rendered)
				}
				text.ScrollToBeginning()
			}
		})
	}()
}

// getCurrentMessageID gets the ID of the currently selected message
func (a *App) getCurrentMessageID() string {
	if table, ok := a.views["list"].(*tview.Table); ok {
		row, _ := table.GetSelection()
		if row >= 0 && row < len(a.ids) {
			return a.ids[row]
		}
	}
	return ""
}

// getListWidth returns current inner width of the list view or a sensible fallback
func (a *App) getListWidth() int {
	if list, ok := a.views["list"].(*tview.Table); ok {
		_, _, w, _ := list.GetInnerRect()
		if w > 0 {
			return w
		}
	}
	if a.screenWidth > 0 {
		return a.screenWidth
	}
	return 80
}

// getFormatWidth devuelve el ancho disponible para el texto de las filas
func (a *App) getFormatWidth() int {
	if list, ok := a.views["list"].(*tview.Table); ok {
		_, _, w, _ := list.GetInnerRect()
		if w > 10 {
			return w - 2
		}
	}
	if a.screenWidth > 0 {
		return a.screenWidth - 2
	}
	return 78
}

// archiveSelected archives the selected message
func (a *App) archiveSelected() {
	var messageID string
	var selectedIndex int = -1
	if a.currentFocus == "list" {
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "summary" {
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else {
		a.showError("❌ Unknown focus state")
		return
	}
	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}

	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	if err := a.Client.ArchiveMessage(messageID); err != nil {
		a.showError(fmt.Sprintf("❌ Error archiving message: %v", err))
		return
	}
	a.showStatusMessage(fmt.Sprintf("📥 Archived: %s", subject))

	// Safe UI removal (preselect another index before removing)
	a.QueueUpdateDraw(func() {
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			return
		}
		count := list.GetRowCount()
		if count == 0 {
			return
		}

		// Determine index to remove; prefer current selection
		removeIndex, _ := list.GetSelection()
		if removeIndex < 0 || removeIndex >= count {
			removeIndex = 0
		}

		// Compute pre-selection different from the removed index
		var next = -1
		if count > 1 {
			pre := removeIndex - 1
			if removeIndex == 0 {
				pre = 1
			}
			if pre < 0 {
				pre = 0
			}
			if pre >= count {
				pre = count - 1
			}
			list.Select(pre, 0)
			next = pre
		}

		// Update caches using removeIndex
		if removeIndex >= 0 && removeIndex < len(a.ids) {
			a.ids = append(a.ids[:removeIndex], a.ids[removeIndex+1:]...)
		}
		if removeIndex >= 0 && removeIndex < len(a.messagesMeta) {
			a.messagesMeta = append(a.messagesMeta[:removeIndex], a.messagesMeta[removeIndex+1:]...)
		}

		// Visual removal
		if count == 1 {
			list.Clear()
			next = -1
		} else {
			if removeIndex >= 0 && removeIndex < list.GetRowCount() {
				list.RemoveRow(removeIndex)
			}
			// next already set to pre; clamp to new count
			if next >= 0 && next < list.GetRowCount() {
				list.Select(next, 0)
			}
		}

		// Update title and content
		list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		if text, ok := a.views["text"].(*tview.TextView); ok {
			if next >= 0 && next < len(a.ids) {
				go a.showMessageWithoutFocus(a.ids[next])
				if a.aiSummaryVisible {
					go a.generateOrShowSummary(a.ids[next])
				}
			} else {
				text.SetText("No messages")
				text.ScrollToBeginning()
				if a.aiSummaryVisible && a.aiSummaryView != nil {
					a.aiSummaryView.SetText("")
				}
			}
		}
	})
}

// archiveSelectedBulk archives all selected messages
func (a *App) archiveSelectedBulk() {
	if len(a.selected) == 0 {
		return
	}
	// Snapshot selection
	ids := make([]string, 0, len(a.selected))
	for id := range a.selected {
		ids = append(ids, id)
	}
	a.setStatusPersistent(fmt.Sprintf("Archiving %d message(s)…", len(ids)))
	go func() {
		failed := 0
		for _, id := range ids {
			if err := a.Client.ArchiveMessage(id); err != nil {
				failed++
				continue
			}
			// Remove from UI list on main thread after loop
		}
		a.QueueUpdateDraw(func() {
			// Remove all archived from current list
			if list, ok := a.views["list"].(*tview.Table); ok {
				// Build a set for quick lookup
				rm := make(map[string]struct{}, len(ids))
				for _, id := range ids {
					rm[id] = struct{}{}
				}
				// Walk ids and remove those that are in rm
				i := 0
				for i < len(a.ids) {
					if _, ok := rm[a.ids[i]]; ok {
						a.ids = append(a.ids[:i], a.ids[i+1:]...)
						if i < len(a.messagesMeta) {
							a.messagesMeta = append(a.messagesMeta[:i], a.messagesMeta[i+1:]...)
						}
						if i < list.GetRowCount() {
							list.RemoveRow(i)
						}
						continue
					}
					i++
				}
				list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
				// Adjust selection and content
				cur, _ := list.GetSelection()
				if cur >= list.GetRowCount() {
					cur = list.GetRowCount() - 1
				}
				if cur >= 0 {
					list.Select(cur, 0)
					if cur < len(a.ids) {
						go a.showMessageWithoutFocus(a.ids[cur])
						if a.aiSummaryVisible {
							go a.generateOrShowSummary(a.ids[cur])
						}
					}
				}
				if list.GetRowCount() == 0 {
					if tv, ok := a.views["text"].(*tview.TextView); ok {
						tv.SetText("No messages")
						tv.ScrollToBeginning()
					}
					if a.aiSummaryVisible && a.aiSummaryView != nil {
						a.aiSummaryView.SetText("")
					}
				}
			}
			a.selected = make(map[string]bool)
			a.bulkMode = false
			if failed == 0 {
				a.showStatusMessage("✅ Archived")
			} else {
				a.showStatusMessage(fmt.Sprintf("✅ Archived with %d failure(s)", failed))
			}
		})
	}()
}

// trashSelectedBulk moves all selected messages to trash
func (a *App) trashSelectedBulk() {
	if len(a.selected) == 0 {
		return
	}
	ids := make([]string, 0, len(a.selected))
	for id := range a.selected {
		ids = append(ids, id)
	}
	a.setStatusPersistent(fmt.Sprintf("Trashing %d message(s)…", len(ids)))
	go func() {
		failed := 0
		for _, id := range ids {
			if err := a.Client.TrashMessage(id); err != nil {
				failed++
			}
		}
		a.QueueUpdateDraw(func() {
			if list, ok := a.views["list"].(*tview.List); ok {
				rm := make(map[string]struct{}, len(ids))
				for _, id := range ids {
					rm[id] = struct{}{}
				}
				i := 0
				for i < len(a.ids) {
					if _, ok := rm[a.ids[i]]; ok {
						a.ids = append(a.ids[:i], a.ids[i+1:]...)
						if i < len(a.messagesMeta) {
							a.messagesMeta = append(a.messagesMeta[:i], a.messagesMeta[i+1:]...)
						}
						if i < list.GetItemCount() {
							list.RemoveItem(i)
						}
						continue
					}
					i++
				}
				list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
				cur := list.GetCurrentItem()
				if cur >= list.GetItemCount() {
					cur = list.GetItemCount() - 1
				}
				if cur >= 0 {
					list.SetCurrentItem(cur)
					if cur < len(a.ids) {
						go a.showMessageWithoutFocus(a.ids[cur])
					}
				}
				if list.GetItemCount() == 0 {
					if tv, ok := a.views["text"].(*tview.TextView); ok {
						tv.SetText("No messages")
						tv.ScrollToBeginning()
					}
				}
			}
			a.selected = make(map[string]bool)
			a.bulkMode = false
			if failed == 0 {
				a.showStatusMessage("✅ Trashed")
			} else {
				a.showStatusMessage(fmt.Sprintf("✅ Trashed with %d failure(s)", failed))
			}
		})
	}()
}

// replySelected replies to the selected message (placeholder)
func (a *App) replySelected() { a.showInfo("Reply functionality not yet implemented") }

// showAttachments shows attachments (placeholder)
func (a *App) showAttachments() { a.showInfo("Attachments functionality not yet implemented") }

// toggleMarkReadUnread toggles UNREAD label on selected message
func (a *App) toggleMarkReadUnread() {
	// Use current list selection regardless of focus
	list, ok := a.views["list"].(*tview.Table)
	if !ok {
		a.showError("❌ Could not access message list")
		return
	}
	idx, _ := list.GetSelection()
	if idx < 0 || idx >= len(a.ids) {
		a.showError("❌ No message selected")
		return
	}
	messageID := a.ids[idx]
	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}
	// Determine unread state from cache if possible to avoid extra roundtrip
	isUnread := false
	if idx < len(a.messagesMeta) && a.messagesMeta[idx] != nil {
		for _, l := range a.messagesMeta[idx].LabelIds {
			if l == "UNREAD" {
				isUnread = true
				break
			}
		}
	} else {
		// Fallback to fetching
		message, err := a.Client.GetMessage(messageID)
		if err == nil {
			for _, l := range message.LabelIds {
				if l == "UNREAD" {
					isUnread = true
					break
				}
			}
		}
	}
	go func(markUnread bool) {
		if markUnread {
			if err := a.Client.MarkAsUnread(messageID); err != nil {
				a.showError(fmt.Sprintf("❌ Error marking as unread: %v", err))
				return
			}
			a.showStatusMessage("✅ Message marked as unread")
			// Update caches/UI on main thread
			a.QueueUpdateDraw(func() {
				a.updateCachedMessageLabels(messageID, "UNREAD", true)
				a.reformatListItems()
			})
		} else {
			if err := a.Client.MarkAsRead(messageID); err != nil {
				a.showError(fmt.Sprintf("❌ Error marking as read: %v", err))
				return
			}
			a.showStatusMessage("✅ Message marked as read")
			a.QueueUpdateDraw(func() {
				a.updateCachedMessageLabels(messageID, "UNREAD", false)
				a.reformatListItems()
			})
		}
	}(!isUnread)
}

// listUnreadMessages placeholder
func (a *App) listUnreadMessages() { a.showInfo("Unread messages functionality not yet implemented") }

// loadDrafts placeholder
func (a *App) loadDrafts() { a.showInfo("Drafts functionality not yet implemented") }

// composeMessage placeholder
func (a *App) composeMessage(draft bool) {
	a.showInfo("Compose message functionality not yet implemented")
}
