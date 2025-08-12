package tui

import (
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/render"
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

// baseRemoveByID removes a message from the local base snapshot if present
func (a *App) baseRemoveByID(messageID string) {
	if a.searchMode != "local" || a.baseIDs == nil {
		return
	}
	idx := -1
	for i, id := range a.baseIDs {
		if id == messageID {
			idx = i
			break
		}
	}
	if idx >= 0 {
		a.baseIDs = append(a.baseIDs[:idx], a.baseIDs[idx+1:]...)
		if idx < len(a.baseMessagesMeta) {
			a.baseMessagesMeta = append(a.baseMessagesMeta[:idx], a.baseMessagesMeta[idx+1:]...)
		}
	}
}

// baseRemoveByIDs removes multiple messages from the local base snapshot
func (a *App) baseRemoveByIDs(ids []string) {
	if a.searchMode != "local" || a.baseIDs == nil || len(ids) == 0 {
		return
	}
	rm := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		rm[id] = struct{}{}
	}
	// rebuild slices
	newIDs := a.baseIDs[:0]
	newMeta := a.baseMessagesMeta[:0]
	for i, id := range a.baseIDs {
		if _, ok := rm[id]; ok {
			continue
		}
		newIDs = append(newIDs, id)
		if i < len(a.baseMessagesMeta) {
			newMeta = append(newMeta, a.baseMessagesMeta[i])
		}
	}
	a.baseIDs = append([]string(nil), newIDs...)
	a.baseMessagesMeta = append([]*gmailapi.Message(nil), newMeta...)
}

// captureLocalBaseSnapshot stores the current inbox view as base for local filtering
func (a *App) captureLocalBaseSnapshot() {
	// Record current selection by message ID to restore later
	var selID string
	if table, ok := a.views["list"].(*tview.Table); ok {
		row, _ := table.GetSelection()
		if row >= 0 && row < len(a.ids) {
			selID = a.ids[row]
		}
	}
	// Copy slices to avoid aliasing
	a.baseIDs = append([]string(nil), a.ids...)
	a.baseMessagesMeta = append([]*gmailapi.Message(nil), a.messagesMeta...)
	a.baseNextPageToken = a.nextPageToken
	a.baseSelectionID = selID
}

// restoreLocalBaseSnapshot restores the base view after exiting local filter
func (a *App) restoreLocalBaseSnapshot() {
	ids := append([]string(nil), a.baseIDs...)
	metas := append([]*gmailapi.Message(nil), a.baseMessagesMeta...)
	next := a.baseNextPageToken
	selID := a.baseSelectionID

	a.QueueUpdateDraw(func() {
		a.searchMode = ""
		a.currentQuery = ""
		a.localFilter = ""
		a.nextPageToken = next
		a.ids = ids
		a.messagesMeta = metas
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.Clear()
			for i := range a.ids {
				if i >= len(a.messagesMeta) || a.messagesMeta[i] == nil {
					continue
				}
				msg := a.messagesMeta[i]
				line, _ := a.emailRenderer.FormatEmailList(msg, a.getFormatWidth())
				// Prefix unread state for consistency
				unread := false
				for _, l := range msg.LabelIds {
					if l == "UNREAD" {
						unread = true
						break
					}
				}
				prefix := "○ "
				if unread {
					prefix = "● "
				}
				table.SetCell(i, 0, tview.NewTableCell(prefix+line).SetExpansion(1))
			}
			// Try to restore selection by ID
			selectIdx := 0
			if selID != "" {
				for i, id := range a.ids {
					if id == selID {
						selectIdx = i
						break
					}
				}
			}
			if table.GetRowCount() > 0 {
				if selectIdx < 0 || selectIdx >= table.GetRowCount() {
					selectIdx = 0
				}
				table.Select(selectIdx, 0)
			}
			table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		}
		a.reformatListItems()
	})
}

// exitSearch handles ESC from search contexts
func (a *App) exitSearch() {
	if a.searchMode == "local" {
		a.restoreLocalBaseSnapshot()
		return
	}
	if a.searchMode == "remote" {
		// Remote search returns to inbox (fresh from server)
		a.searchMode = ""
		a.currentQuery = ""
		a.localFilter = ""
		a.nextPageToken = ""
		go a.reloadMessages()
		return
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

	// Initial title before we know page size
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

	// Spinner with progress once we know how many items are coming
	var spinnerStop chan struct{}
	loaded := 0
	total := len(messages)
	if _, ok := a.views["list"].(*tview.Table); ok {
		spinnerStop = make(chan struct{})
		go func() {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			ticker := time.NewTicker(150 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-spinnerStop:
					return
				case <-ticker.C:
					prog := loaded
					a.QueueUpdateDraw(func() {
						if tb, ok := a.views["list"].(*tview.Table); ok {
							tb.SetTitle(fmt.Sprintf(" %s Loading… (%d/%d) ", frames[i%len(frames)], prog, total))
						}
					})
					i++
				}
			}
		}()
	}

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
			cell := tview.NewTableCell(formattedText).
				SetExpansion(1).
				SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			table.SetCell(i, 0, cell)
		}

		// cache meta for resize re-rendering
		a.messagesMeta = append(a.messagesMeta, message)

		loaded = i + 1
	}

	a.QueueUpdateDraw(func() {
		// If command mode is active, close it to avoid stealing focus after load
		if a.cmdMode {
			a.hideCommandBar()
		}
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
		// Stop spinner if running
		if spinnerStop != nil {
			close(spinnerStop)
		}
		// Force focus to list at the end of initial load
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])
	})

	// Do not steal focus if user moved to another pane (e.g., labels/summary/text)
	// Keep currentFocus list during loading; focus is enforced above on completion
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
	// Append with lightweight progress in title
	var spinnerStop chan struct{}
	loaded := 0
	total := len(messages)
	if _, ok := a.views["list"].(*tview.Table); ok {
		spinnerStop = make(chan struct{})
		go func() {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			ticker := time.NewTicker(120 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-spinnerStop:
					return
				case <-ticker.C:
					prog := loaded
					a.QueueUpdateDraw(func() {
						if tb, ok := a.views["list"].(*tview.Table); ok {
							tb.SetTitle(fmt.Sprintf(" %s Loading more… (%d/%d) ", frames[i%len(frames)], prog, total))
						}
					})
					i++
				}
			}
		}()
	}
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
			cell := tview.NewTableCell(text).
				SetExpansion(1).
				SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			table.SetCell(row, 0, cell)
		}
		loaded++
	}
	a.nextPageToken = next
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		}
		a.reformatListItems()
		if spinnerStop != nil {
			close(spinnerStop)
		}
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
			cell := tview.NewTableCell(text).
				SetExpansion(1).
				SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			table.SetCell(row, 0, cell)
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

	ph := "e.g., from:user@domain.com subject:\"report\" is:unread label:work"
	if mode == "local" {
		ph = "Type words to match (space-separated)"
	}
	input := tview.NewInputField().
		SetLabel("").
		SetFieldWidth(0).
		SetPlaceholder(ph)
	// expose input so Tab from list can focus it
	a.views["searchInput"] = input
	help := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	help.SetTextColor(tcell.ColorGray)
	if mode == "remote" {
		help.SetText("Press Ctrl+F for advanced search | Enter=search, Ctrl-T=switch, ESC to back")
	} else {
		help.SetText("Type space-separated terms; all must match | Enter=apply, Ctrl-T=switch, ESC to back")
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
				// Before applying local filter, capture base snapshot
				a.captureLocalBaseSnapshot()
				a.localFilter = query
				go a.applyLocalFilter(query)
			}
			delete(a.views, "searchInput")
		}
		if key == tcell.KeyEscape {
			// If simple overlay is visible, hide it; else, restore list
			if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
				if sp, ok2 := a.views["searchPanel"].(*tview.Flex); ok2 {
					// Heuristic: if searchPanel currently has a title, consider it visible
					if sp.GetTitle() != "" {
						lc.Clear()
						sp.SetBorder(false)
						sp.SetTitle("")
						lc.AddItem(a.views["searchPanel"], 0, 0, false)
						lc.AddItem(a.views["list"], 0, 1, true)
						a.currentFocus = "list"
						a.updateFocusIndicators("list")
						a.SetFocus(a.views["list"])
						delete(a.views, "searchInput")
						// If exiting overlay from a local filter, restore base view immediately
						if a.searchMode == "local" {
							go a.exitSearch()
						}
						return
					}
				}
				lc.Clear()
				lc.AddItem(a.views["searchPanel"], 0, 0, false)
				lc.AddItem(a.views["list"], 0, 1, true)
			} else {
				a.Pages.RemovePage("searchOverlay")
			}
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
			a.SetFocus(a.views["list"])
			delete(a.views, "searchInput")
			// If leaving overlay and a local filter was active, restore base
			if a.searchMode == "local" {
				go a.exitSearch()
			}
		}
	})
	input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		// Ctrl+T: toggle remote/local (support both KeyCtrlT and modifier+rune)
		if ev.Key() == tcell.KeyCtrlT || ((ev.Modifiers()&tcell.ModCtrl) != 0 && ev.Rune() == 't') {
			if curMode == "remote" {
				curMode = "local"
				box.SetTitle("🔎 Local Filter")
				help.SetText("Type space-separated terms; all must match | Enter=apply, Ctrl-T=switch, ESC to back")
				input.SetPlaceholder("Type words to match (space-separated)")
				if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
					sp.SetTitle("🔎 Local Filter")
				}
			} else {
				curMode = "remote"
				box.SetTitle("🔍 Gmail Search")
				help.SetText("Press Ctrl+F for advanced search | Enter=search, Ctrl-T=switch, ESC to back")
				input.SetPlaceholder("e.g., from:user@domain.com subject:\"report\" is:unread label:work")
				if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
					sp.SetTitle("🔍 Gmail Search")
				}
			}
			return nil
		}
		// Ctrl+F: open advanced search form
		if ev.Key() == tcell.KeyCtrlF || ((ev.Modifiers()&tcell.ModCtrl) != 0 && ev.Rune() == 'f') {
			a.openAdvancedSearchForm()
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
		// Ensure container does not intercept ESC here; let input handle hiding
		sp.SetInputCapture(nil)
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
	// Build form fields similar to Gmail advanced search (with placeholders)
	form := tview.NewForm()
	form.AddFormItem(tview.NewInputField().SetLabel("From").SetPlaceholder("user@example.com"))
	form.AddFormItem(tview.NewInputField().SetLabel("To").SetPlaceholder("person@example.com"))
	form.AddFormItem(tview.NewInputField().SetLabel("Subject").SetPlaceholder("exact words or phrase"))
	form.AddFormItem(tview.NewInputField().SetLabel("Has the words").SetPlaceholder("words here"))
	form.AddFormItem(tview.NewInputField().SetLabel("Doesn't have").SetPlaceholder("exclude words"))
	// Size single expression, e.g. "<2MB" or ">500KB"
	sizeExprField := tview.NewInputField().SetLabel("Size").SetPlaceholder("e.g., <2MB or >500KB")
	form.AddFormItem(sizeExprField)
	// Date within single token, e.g. "2d", "3w", "1m", "4h", "6y"
	dateWithinField := tview.NewInputField().SetLabel("Date within").SetPlaceholder("e.g., 2d, 3w, 1m, 4h, 6y")
	form.AddFormItem(dateWithinField)
	// Scope
	baseScopes := []string{"All Mail", "Inbox", "Sent", "Drafts", "Spam", "Trash", "Starred", "Important"}
	scopes := append([]string{}, baseScopes...)
	scopeVal := "All Mail"
	scopeField := tview.NewInputField().
		SetLabel("Search").
		SetText(scopeVal).
		SetPlaceholder("Press Enter to pick scope/label")
	// Prevent manual typing; we use a picker for consistency with Browse all labels
	scopeField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool { return false })
	form.AddFormItem(scopeField)
	// Attachment
	var hasAttachment bool
	form.AddCheckbox("Has attachment", false, func(label string, checked bool) { hasAttachment = checked })

	// Load labels asynchronously to build picker options
	go func() {
		labels, err := a.Client.ListLabels()
		if err != nil || labels == nil {
			return
		}
		names := make([]string, 0, len(labels))
		for _, l := range labels {
			// Hide system categories we already map
			if l.Type == "system" {
				continue
			}
			names = append(names, l.Name)
		}
		if len(names) == 0 {
			return
		}
		a.QueueUpdateDraw(func() {
			sort.Strings(names)
			scopes = append(baseScopes, names...)
		})
	}()

	// Picker dentro del panel (mismo patrón que expandLabelsBrowseWithMode)
	openScopePicker := func() {
		sp, ok := a.views["searchPanel"].(*tview.Flex)
		if !ok {
			return
		}

		filter := tview.NewInputField().
			SetLabel("🔎 Filter: ").
			SetFieldWidth(30).
			SetPlaceholder("type to filter; Enter=select, ESC=back")
		list := tview.NewList().ShowSecondaryText(false)
		list.SetBorder(false)
		list.SetSelectedTextColor(tcell.ColorBlack)
		list.SetSelectedBackgroundColor(tcell.ColorWhite)

		// Cargar lista
		var update func()
		update = func() {
			txt := strings.ToLower(strings.TrimSpace(filter.GetText()))
			list.Clear()
			for _, s := range scopes {
				if txt == "" || strings.Contains(strings.ToLower(s), txt) {
					list.AddItem(s, "", 0, nil)
				}
			}
			if list.GetItemCount() > 0 {
				list.SetCurrentItem(0)
			}
		}
		filter.SetChangedFunc(func(_ string) { update() })
		update()

		// Selección
		list.SetSelectedFunc(func(index int, mainText, _ string, _ rune) {
			if mainText != "" {
				scopeVal = mainText
				scopeField.SetText(scopeVal)
			}
			sp.Clear()
			sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)
			sp.AddItem(form, 0, 1, true)
			a.SetFocus(scopeField)
		})
		// ESC desde filtro
		filter.SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEscape {
				sp.Clear()
				sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)
				sp.AddItem(form, 0, 1, true)
				a.SetFocus(scopeField)
			}
			if key == tcell.KeyEnter && list.GetItemCount() > 0 {
				main, _ := list.GetItemText(list.GetCurrentItem())
				scopeVal = main
				scopeField.SetText(scopeVal)
				sp.Clear()
				sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)
				sp.AddItem(form, 0, 1, true)
				a.SetFocus(scopeField)
			}
		})
		// Flechas desde filtro → lista
		filter.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			switch e.Key() {
			case tcell.KeyDown, tcell.KeyUp, tcell.KeyPgDn, tcell.KeyPgUp, tcell.KeyHome, tcell.KeyEnd:
				a.SetFocus(list)
				return e
			}
			return e
		})
		// ESC desde lista
		list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			if e.Key() == tcell.KeyEscape {
				sp.Clear()
				sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)
				sp.AddItem(form, 0, 1, true)
				a.SetFocus(scopeField)
				return nil
			}
			return e
		})

		// Pintar dentro del panel
		a.QueueUpdateDraw(func() {
			sp.Clear()
			sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle("🔎 Pick scope or label").SetTitleColor(tcell.ColorYellow)
			sp.AddItem(filter, 3, 0, true)
			sp.AddItem(list, 0, 1, true)
			a.SetFocus(filter)
			a.currentFocus = "search"
			a.updateFocusIndicators("search")
		})
	}
	scopeField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			openScopePicker()
		}
	})
	// Do not intercept Enter here; DoneFunc above will handle opening the picker
	scopeField.SetInputCapture(nil)

	// Build submit function shared by button and Ctrl+Enter
	submit := func() {
		from := form.GetFormItemByLabel("From").(*tview.InputField).GetText()
		to := form.GetFormItemByLabel("To").(*tview.InputField).GetText()
		subject := form.GetFormItemByLabel("Subject").(*tview.InputField).GetText()
		hasWords := form.GetFormItemByLabel("Has the words").(*tview.InputField).GetText()
		notWords := form.GetFormItemByLabel("Doesn't have").(*tview.InputField).GetText()
		sizeExpr := form.GetFormItemByLabel("Size").(*tview.InputField).GetText()
		dateWithinExpr := form.GetFormItemByLabel("Date within").(*tview.InputField).GetText()

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
		if hasWords != "" {
			parts = append(parts, hasWords)
		}
		if notWords != "" {
			parts = append(parts, fmt.Sprintf("-%s", notWords))
		}
		// Size (parse <NMB or >NKB)
		if expr := strings.TrimSpace(sizeExpr); expr != "" {
			op := expr[0]
			rest := strings.TrimSpace(expr[1:])
			// split number and unit
			num := ""
			unit := ""
			for i := 0; i < len(rest); i++ {
				if rest[i] >= '0' && rest[i] <= '9' {
					num += string(rest[i])
				} else {
					unit = strings.TrimSpace(rest[i:])
					break
				}
			}
			u := strings.ToLower(unit)
			suffix := ""
			if strings.HasPrefix(u, "mb") || u == "m" {
				suffix = "m"
			} else if strings.HasPrefix(u, "kb") || u == "k" {
				suffix = "k"
			}
			if num != "" && suffix != "" {
				if op == '>' {
					parts = append(parts, fmt.Sprintf("larger:%s%s", num, suffix))
				} else if op == '<' {
					parts = append(parts, fmt.Sprintf("smaller:%s%s", num, suffix))
				}
			}
		}
		// Date within -> newer_than:N(unit)
		if tok := strings.TrimSpace(dateWithinExpr); tok != "" {
			// token like 2d, 3w, 1m, 4h, 6y
			n := ""
			unit := ""
			for i := 0; i < len(tok); i++ {
				if tok[i] >= '0' && tok[i] <= '9' {
					n += string(tok[i])
				} else {
					unit = strings.ToLower(strings.TrimSpace(tok[i:]))
					break
				}
			}
			if n != "" && unit != "" {
				// Allow d,w,m,y,h (Gmail may ignore h/w in some cases)
				switch unit[0] {
				case 'd', 'w', 'm', 'y', 'h':
					parts = append(parts, fmt.Sprintf("newer_than:%s%s", n, string(unit[0])))
				}
			}
		}
		// Scope (union fixed folders + labels) using scopeVal
		if scopeVal != "" {
			sel := scopeVal
			switch sel {
			case "All Mail":
				// no-op
			case "Inbox":
				parts = append(parts, "in:inbox")
			case "Sent":
				parts = append(parts, "in:sent")
			case "Drafts":
				parts = append(parts, "in:draft")
			case "Spam":
				parts = append(parts, "in:spam")
			case "Trash":
				parts = append(parts, "in:trash")
			case "Starred":
				parts = append(parts, "is:starred")
			case "Important":
				parts = append(parts, "is:important")
			default:
				// Assume label name
				parts = append(parts, fmt.Sprintf("label:%q", sel))
			}
		}
		if hasAttachment {
			parts = append(parts, "has:attachment")
		}

		q := strings.Join(parts, " ")

		if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
			lc.Clear()
			// Hide search panel and restore list to full height
			lc.AddItem(a.views["searchPanel"], 0, 0, false)
			lc.AddItem(a.views["list"], 0, 1, true)
		}
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])
		go a.performSearch(q)
	}
	form.SetButtonsAlign(tview.AlignRight)
	form.AddButton("Search", submit)
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
	form.SetBorder(false) // inner form without its own title; container shows the title
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		// When a dropdown is open, intercept keys (ESC/tab/enter)
		idx, _ := form.GetFocusedItemIndex()
		if idx >= 0 {
			if _, ok := form.GetFormItem(idx).(*tview.DropDown); ok {
				if ev.Key() == tcell.KeyEscape {
					// Return to simple search overlay instead of main list
					if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
						sp.Clear()
					}
					if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
						lc.Clear()
						lc.AddItem(a.views["searchPanel"], 0, 1, true)
						lc.AddItem(a.views["list"], 0, 3, true)
					}
					a.openSearchOverlay("remote")
					return nil
				}
				// Let Enter select option; do not submit here
				if ev.Key() == tcell.KeyTab {
					return ev
				}
			}
		}
		// Submit only with Ctrl+Enter (or Ctrl+S)
		if (ev.Key() == tcell.KeyEnter && (ev.Modifiers()&tcell.ModCtrl) != 0) || (ev.Rune() == 's' && (ev.Modifiers()&tcell.ModCtrl) != 0) {
			submit()
			return nil
		}
		if ev.Key() == tcell.KeyEscape {
			// Return to simple search overlay instead of main list
			if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
				sp.Clear()
			}
			if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
				lc.Clear()
				// Restore simple search overlay at 25% and list below
				lc.AddItem(a.views["searchPanel"], 0, 1, true)
				lc.AddItem(a.views["list"], 0, 3, true)
			}
			// Reopen simple search in remote mode by default
			a.openSearchOverlay("remote")
			return nil
		}
		return ev
	})

	// Mount form into searchPanel area; expand to 50% and hide list for spacious layout
	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		sp.Clear()
		sp.SetBorder(true).SetBorderColor(tcell.ColorYellow).SetTitle("🔎 Advanced Search").SetTitleColor(tcell.ColorYellow)
		sp.AddItem(form, 0, 1, true)
		if lc, ok2 := a.views["listContainer"].(*tview.Flex); ok2 {
			lc.Clear()
			// Allocate 50% to form, hide list (weight 0)
			lc.AddItem(a.views["searchPanel"], 0, 1, true)
			lc.AddItem(a.views["list"], 0, 0, false)
		}
		// Allow ESC at container level to return to simple search overlay
		if sp2, ok3 := a.views["searchPanel"].(*tview.Flex); ok3 {
			sp2.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
				if ev.Key() == tcell.KeyEscape {
					a.openSearchOverlay("remote")
					return nil
				}
				return ev
			})
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
	// Compute matches off the UI thread
	tokens := strings.Fields(strings.ToLower(expr))
	filteredIDs := make([]string, 0, len(a.ids))
	filteredMeta := make([]*gmailapi.Message, 0, len(a.messagesMeta))
	rows := make([]string, 0, len(a.messagesMeta))

	for i, m := range a.messagesMeta {
		if m == nil {
			continue
		}
		// Build a rich searchable string: Subject, From, To, Snippet
		var subject, from, to string
		if m.Payload != nil {
			for _, h := range m.Payload.Headers {
				switch strings.ToLower(h.Name) {
				case "subject":
					subject = h.Value
				case "from":
					from = h.Value
				case "to":
					to = h.Value
				}
			}
		}
		content := strings.ToLower(subject + " " + from + " " + to + " " + m.Snippet)
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
		filteredIDs = append(filteredIDs, a.ids[i])
		filteredMeta = append(filteredMeta, m)
		// Render the row text for display using the renderer
		line, _ := a.emailRenderer.FormatEmailList(m, a.getFormatWidth())
		rows = append(rows, line)
	}

	// Apply results on UI thread
	a.QueueUpdateDraw(func() {
		a.searchMode = "local"
		a.localFilter = expr
		// Replace current view with filtered content BEFORE selecting rows to ensure
		// selection handlers reference the filtered ids/meta, not the previous inbox
		a.ids = filteredIDs
		a.messagesMeta = filteredMeta
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.Clear()
			for i, text := range rows {
				table.SetCell(i, 0, tview.NewTableCell(text).SetExpansion(1))
			}
			table.SetTitle(fmt.Sprintf(" 🔎 Filter (%d) — %s ", len(rows), expr))
			if table.GetRowCount() > 0 {
				table.Select(0, 0)
			}
		}
		a.reformatListItems()
	})
}

// showMessage displays a message in the text view
func (a *App) showMessage(id string) {
	// Show loading message immediately
	if text, ok := a.views["text"].(*tview.TextView); ok {
		if a.debug {
			a.logger.Printf("showMessage: id=%s", id)
		}
		if a.llmTouchUpEnabled {
			a.setStatusPersistent("🧠 Optimizing format with LLM…")
		} else {
			a.setStatusPersistent("🧾 Loading message…")
		}
		text.SetText("Loading message…")
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

// saveCurrentMessageToFile writes the currently focused message to disk under config dir
func (a *App) saveCurrentMessageToFile() {
	id := a.getCurrentMessageID()
	if id == "" {
		// Fallback to last opened message
		id = a.currentMessageID
	}
	if id == "" {
		a.showError("❌ No message selected")
		return
	}
	// Immediate feedback on UI thread
	a.setStatusPersistent("💾 Saving message…")
	go func(mid string) {
		// Try cache first
		var m *gmail.Message
		if cached, ok := a.messageCache[mid]; ok {
			m = cached
		} else {
			fetched, err := a.Client.GetMessageWithContent(mid)
			if err != nil {
				a.QueueUpdateDraw(func() { a.showError("❌ Could not load message") })
				return
			}
			m = fetched
		}
		// Build output using deterministic formatter without LLM
		width := a.getListWidth()
		txt, _ := render.FormatEmailForTerminal(a.ctx, m, render.FormatOptions{WrapWidth: width, UseLLM: false}, nil)
		// Compose full content with header
		header := a.emailRenderer.FormatHeaderPlain(m.Subject, m.From, m.To, m.Cc, m.Date, m.Labels)
		content := header + "\n\n" + txt

		// Resolve config dir and saved folder
		home, _ := os.UserHomeDir()
		base := filepath.Join(home, ".config", "gmail-tui", "saved")
		if err := os.MkdirAll(base, 0o755); err != nil {
			a.QueueUpdateDraw(func() { a.showError("❌ Could not create saved folder") })
			return
		}
		// Sanitize subject to filename
		name := m.Subject
		if strings.TrimSpace(name) == "" {
			name = mid
		}
		name = sanitizeFilename(name)
		// Ensure uniqueness with timestamp
		ts := time.Now().Format("20060102-150405")
		file := filepath.Join(base, ts+"-"+name+".txt")
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			a.QueueUpdateDraw(func() { a.showError("❌ Could not write file") })
			return
		}
		a.QueueUpdateDraw(func() { a.showStatusMessage("💾 Saved: " + file) })
	}(id)
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "|", "-")
	s = strings.ReplaceAll(s, "*", "-")
	s = strings.ReplaceAll(s, "?", "-")
	s = strings.ReplaceAll(s, "\"", "'")
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		s = s[:80]
	}
	if s == "" {
		s = "message"
	}
	return s
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

		// In preview (selection change), do not run LLM touch-up to avoid many calls
		prev := a.llmTouchUpEnabled
		a.llmTouchUpEnabled = false
		rendered, isANSI := a.renderMessageContent(message)
		a.llmTouchUpEnabled = prev

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

// extractHeaderValue returns the value of a header (case-insensitive) from a Gmail message metadata
func extractHeaderValue(m *gmailapi.Message, headerName string) string {
	if m == nil || m.Payload == nil {
		return ""
	}
	hn := strings.ToLower(headerName)
	for _, h := range m.Payload.Headers {
		if strings.ToLower(h.Name) == hn {
			return h.Value
		}
	}
	return ""
}

// parseEmailAddress parses a raw RFC5322 address string and returns the email and domain
func parseEmailAddress(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if addr, err := mail.ParseAddress(raw); err == nil && addr != nil {
		a := strings.TrimSpace(strings.ToLower(addr.Address))
		if i := strings.LastIndexByte(a, '@'); i > 0 && i < len(a)-1 {
			return a, a[i+1:]
		}
		return a, ""
	}
	// Fallback: try to extract between < > or use raw token
	if i := strings.IndexByte(raw, '<'); i >= 0 {
		if j := strings.IndexByte(raw[i:], '>'); j > 0 {
			token := strings.TrimSpace(raw[i+1 : i+j])
			token = strings.ToLower(token)
			if k := strings.LastIndexByte(token, '@'); k > 0 && k < len(token)-1 {
				return token, token[k+1:]
			}
			return token, ""
		}
	}
	low := strings.ToLower(raw)
	if k := strings.LastIndexByte(low, '@'); k > 0 && k < len(low)-1 {
		return low, low[k+1:]
	}
	return low, ""
}

// searchByFromCurrent searches messages in Inbox from the sender of the currently selected message
func (a *App) searchByFromCurrent() {
	id := a.getCurrentMessageID()
	if id == "" {
		a.showError("❌ No message selected")
		return
	}
	var meta *gmailapi.Message
	// Prefer cached metadata slice
	if table, ok := a.views["list"].(*tview.Table); ok {
		row, _ := table.GetSelection()
		if row >= 0 && row < len(a.messagesMeta) {
			meta = a.messagesMeta[row]
		}
	}
	if meta == nil {
		m, err := a.Client.GetMessage(id)
		if err != nil {
			a.showError("❌ Could not load message metadata")
			return
		}
		meta = m
	}
	from := extractHeaderValue(meta, "From")
	email, _ := parseEmailAddress(from)
	if strings.TrimSpace(email) == "" {
		a.showError("❌ Could not determine sender")
		return
	}
	q := fmt.Sprintf("from:%s", email)
	go a.performSearch(q)
}

// searchByToCurrent searches messages anywhere addressed to the sender of the selected message
// Includes Sent to cover your messages to that person, and excludes spam/trash
func (a *App) searchByToCurrent() {
	id := a.getCurrentMessageID()
	if id == "" {
		a.showError("❌ No message selected")
		return
	}
	var meta *gmailapi.Message
	if table, ok := a.views["list"].(*tview.Table); ok {
		row, _ := table.GetSelection()
		if row >= 0 && row < len(a.messagesMeta) {
			meta = a.messagesMeta[row]
		}
	}
	if meta == nil {
		m, err := a.Client.GetMessage(id)
		if err != nil {
			a.showError("❌ Could not load message metadata")
			return
		}
		meta = m
	}
	from := extractHeaderValue(meta, "From")
	email, _ := parseEmailAddress(from)
	if strings.TrimSpace(email) == "" {
		a.showError("❌ Could not determine recipient")
		return
	}
	// Explicit in:anywhere prevents default inbox-only constraint in performSearch
	q := fmt.Sprintf("in:anywhere -in:spam -in:trash to:%s", email)
	go a.performSearch(q)
}

// searchBySubjectCurrent searches messages by the exact subject of the currently selected message
func (a *App) searchBySubjectCurrent() {
	id := a.getCurrentMessageID()
	if id == "" {
		a.showError("❌ No message selected")
		return
	}
	var meta *gmailapi.Message
	if table, ok := a.views["list"].(*tview.Table); ok {
		row, _ := table.GetSelection()
		if row >= 0 && row < len(a.messagesMeta) {
			meta = a.messagesMeta[row]
		}
	}
	if meta == nil {
		m, err := a.Client.GetMessage(id)
		if err != nil {
			a.showError("❌ Could not load message metadata")
			return
		}
		meta = m
	}
	subject := extractHeaderValue(meta, "Subject")
	subject = strings.TrimSpace(subject)
	if subject == "" {
		a.showError("❌ Subject not available")
		return
	}
	// Quote for exact match
	q := fmt.Sprintf("subject:%q", subject)
	go a.performSearch(q)
}

// searchByDomainCurrent searches messages from the sender's domain of the selected message
func (a *App) searchByDomainCurrent() {
	id := a.getCurrentMessageID()
	if id == "" {
		a.showError("❌ No message selected")
		return
	}
	var meta *gmailapi.Message
	if table, ok := a.views["list"].(*tview.Table); ok {
		row, _ := table.GetSelection()
		if row >= 0 && row < len(a.messagesMeta) {
			meta = a.messagesMeta[row]
		}
	}
	if meta == nil {
		m, err := a.Client.GetMessage(id)
		if err != nil {
			a.showError("❌ Could not load message metadata")
			return
		}
		meta = m
	}
	from := extractHeaderValue(meta, "From")
	_, domain := parseEmailAddress(from)
	domain = strings.TrimSpace(domain)
	if domain == "" {
		a.showError("❌ Could not determine domain")
		return
	}
	q := fmt.Sprintf("from:(@%s)", domain)
	go a.performSearch(q)
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

		// Preselect a different index to avoid removal-on-selected glitches
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
			// No selection remains
		} else {
			if removeIndex >= 0 && removeIndex < list.GetRowCount() {
				list.RemoveRow(removeIndex)
			}
			// Keep the same visual position when possible (select the row that shifted into removeIndex)
			desired := removeIndex
			newCount := list.GetRowCount()
			if desired >= newCount {
				desired = newCount - 1
			}
			if desired >= 0 && desired < newCount {
				list.Select(desired, 0)
			}
		}

		// Update title and content
		list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		if text, ok := a.views["text"].(*tview.TextView); ok {
			// Reflect the newly selected row (if any)
			cur, _ := list.GetSelection()
			if cur >= 0 && cur < len(a.ids) {
				go a.showMessageWithoutFocus(a.ids[cur])
				if a.aiSummaryVisible {
					go a.generateOrShowSummary(a.ids[cur])
				}
			} else {
				text.SetText("No messages")
				text.ScrollToBeginning()
				if a.aiSummaryVisible && a.aiSummaryView != nil {
					a.aiSummaryView.SetText("")
				}
			}
		}
		// Propagate removal to base snapshot if in local filter
		if messageID != "" {
			a.baseRemoveByID(messageID)
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
			// Propagate to base snapshot if in local filter
			a.baseRemoveByIDs(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
			}
			a.setStatusPersistent("")
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
			if list, ok := a.views["list"].(*tview.Table); ok {
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
			// Propagate to base snapshot if in local filter
			a.baseRemoveByIDs(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
			}
			a.setStatusPersistent("")
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
