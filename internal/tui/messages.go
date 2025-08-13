package tui

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/render"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/mattn/go-runewidth"
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

	// Process messages using the email renderer (progressive paint + color)
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

		// Update renderer context (labels map + system visibility policy)
		if labels, err := a.Client.ListLabels(); err == nil {
			m := make(map[string]string, len(labels))
			for _, l := range labels {
				m[l.Id] = l.Name
			}
			a.emailRenderer.SetLabelMap(m)
			a.emailRenderer.SetShowSystemLabelsInList(a.searchMode == "remote")
		}
		// Use the email renderer to format the message (with 📎/🗓️ and label chips)
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

		// cache meta for resize re-rendering
		a.messagesMeta = append(a.messagesMeta, message)

		// Paint row and apply colors immediately
		a.QueueUpdateDraw(func() {
			if table, ok := a.views["list"].(*tview.Table); ok {
				cell := tview.NewTableCell(formattedText).
					SetExpansion(1).
					SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
				table.SetCell(i, 0, cell)
			}
			a.reformatListItems()
		})

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
				// Auto-load content for the first message
				if len(a.ids) > 0 {
					firstID := a.ids[0]
					a.currentMessageID = firstID
					go a.showMessageWithoutFocus(firstID)
					if a.aiSummaryVisible {
						go a.generateOrShowSummary(firstID)
					}
				}
			}
		}
		// Final pass (in case of resize between frames)
		a.reformatListItems()
		// If advanced search is visible, keep focus on it
		if sp, ok := a.views["searchPanel"].(*tview.Flex); ok && sp.GetTitle() == "🔎 Advanced Search" {
			if f, fok := a.views["advFrom"].(*tview.InputField); fok {
				a.currentFocus = "search"
				a.updateFocusIndicators("search")
				a.SetFocus(f)
			}
		}
		// Stop spinner if running
		if spinnerStop != nil {
			close(spinnerStop)
		}
		// Force focus to list unless advanced search is visible
		if spt, ok := a.views["searchPanel"].(*tview.Flex); !(ok && spt.GetTitle() == "🔎 Advanced Search") {
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
			a.SetFocus(a.views["list"])
		}
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
			if labels, err := a.Client.ListLabels(); err == nil {
				m := make(map[string]string, len(labels))
				for _, l := range labels {
					m[l.Id] = l.Name
				}
				a.emailRenderer.SetLabelMap(m)
				a.emailRenderer.SetShowSystemLabelsInList(a.searchMode == "remote")
			}
			if labels, err := a.Client.ListLabels(); err == nil {
				m := make(map[string]string, len(labels))
				for _, l := range labels {
					m[l.Id] = l.Name
				}
				a.emailRenderer.SetLabelMap(m)
				a.emailRenderer.SetShowSystemLabelsInList(a.searchMode == "remote")
			}
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
		SetLabel("🔍 ").
		SetLabelColor(tcell.ColorYellow).
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
			query := strings.TrimSpace(input.GetText())
			if query == "" && curMode == "remote" {
				a.showStatusMessage("🔎 Enter a search query or press ESC to cancel")
				return
			}
			// If there was an LLM suggestion running, inform user it was cancelled
			if a.LLM != nil {
				for k := range a.aiInFlight {
					if a.aiInFlight[k] {
						a.aiInFlight[k] = false
						a.showStatusMessage("🔕 Sugerencia cancelada al abrir la búsqueda")
						break
					}
				}
			}
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
	fromField := tview.NewInputField().SetLabel("👤 From").SetPlaceholder("user@example.com")
	// Expose for focus restoration while background loads complete
	a.views["advFrom"] = fromField
	toField := tview.NewInputField().SetLabel("📩 To").SetPlaceholder("person@example.com")
	subjectField := tview.NewInputField().SetLabel("🧾 Subject").SetPlaceholder("exact words or phrase")
	hasField := tview.NewInputField().SetLabel("🔎 Has the words").SetPlaceholder("words here")
	notField := tview.NewInputField().SetLabel("🚫 Doesn't have").SetPlaceholder("exclude words")
	form.AddFormItem(fromField)
	form.AddFormItem(toField)
	form.AddFormItem(subjectField)
	form.AddFormItem(hasField)
	form.AddFormItem(notField)
	// Size single expression, e.g. "<2MB" or ">500KB"
	sizeExprField := tview.NewInputField().SetLabel("📦 Size").SetPlaceholder("e.g., <2MB or >500KB")
	form.AddFormItem(sizeExprField)
	// Date within single token, e.g. "2d", "3w", "1m", "4h", "6y"
	dateWithinField := tview.NewInputField().SetLabel("⏱️  Date within").SetPlaceholder("e.g., 2d, 3w, 1m, 4h, 6y")
	form.AddFormItem(dateWithinField)
	// Scope
	baseScopes := []string{"All Mail", "Inbox", "Sent", "Drafts", "Spam", "Trash", "Starred", "Important"}
	scopes := append([]string{}, baseScopes...)
	scopeVal := "All Mail"
	if a.logger != nil {
		a.logger.Println("advsearch: building form")
	}
	scopeField := tview.NewInputField().
		SetLabel("📂 Search").
		SetText(scopeVal).
		SetPlaceholder("Press Enter to pick scope/label")
	form.AddFormItem(scopeField)
	// Expose fields for global navigation handling by storing the form itself
	a.views["advForm"] = form
	// Enable arrow-key navigation between fields
	focusClamp := func(i int) int {
		c := form.GetFormItemCount()
		if c == 0 {
			return 0
		}
		if i < 0 {
			return 0
		}
		if i >= c {
			return c - 1
		}
		return i
	}
	setNav := func(f *tview.InputField, idx int) {
		f.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			if a.logger != nil {
				a.logger.Printf("advsearch: field[%d] key=%v rune=%q focus=%T", idx, ev.Key(), ev.Rune(), a.GetFocus())
			}
			switch ev.Key() {
			case tcell.KeyUp:
				next := focusClamp(idx - 1)
				if a.logger != nil {
					a.logger.Printf("advsearch: move Up %d -> %d", idx, next)
				}
				form.SetFocus(next)
				return nil
			case tcell.KeyDown:
				next := focusClamp(idx + 1)
				if a.logger != nil {
					a.logger.Printf("advsearch: move Down %d -> %d", idx, next)
				}
				form.SetFocus(next)
				return nil
			case tcell.KeyTab:
				next := focusClamp(idx + 1)
				if a.logger != nil {
					a.logger.Printf("advsearch: Tab %d -> %d", idx, next)
				}
				form.SetFocus(next)
				return nil
			case tcell.KeyBacktab:
				next := focusClamp(idx - 1)
				if a.logger != nil {
					a.logger.Printf("advsearch: Backtab %d -> %d", idx, next)
				}
				form.SetFocus(next)
				return nil
			}
			return ev
		})
	}
	// Indices match the order added above
	setNav(fromField, 0)
	setNav(toField, 1)
	setNav(subjectField, 2)
	setNav(hasField, 3)
	setNav(notField, 4)
	setNav(sizeExprField, 5)
	setNav(dateWithinField, 6)
	// Attachment
	var hasAttachment bool
	form.AddCheckbox("📎 Has attachment", false, func(label string, checked bool) { hasAttachment = checked })

	// Load labels asynchronously to build picker options
	go func() {
		if a.logger != nil {
			a.logger.Println("advsearch: loading labels...")
		}
		labels, err := a.Client.ListLabels()
		if err != nil || labels == nil {
			if a.logger != nil {
				a.logger.Printf("advsearch: ListLabels error=%v", err)
			}
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
			if a.logger != nil {
				a.logger.Printf("advsearch: injecting %d user labels", len(names))
			}
			sort.Strings(names)
			scopes = append(baseScopes, names...)
		})
	}()

	// Track right panel visibility for toggle behavior
	rightVisible := false

	// Will hold a function to restore the default layout after closing advanced search
	var restoreLayout func()

	// Right-side picker inside search panel (removed: replaced by unified filter in options)

	// Right-side options panel (categories with icons)
	// Right-side options panel (categories). Keep as local to be callable
	// Helper to hide right column
	hideRight := func() {
		if right, ok := a.views["searchRight"].(*tview.Flex); ok {
			right.Clear()
			if twoCol, ok := a.views["searchTwoCol"].(*tview.Flex); ok {
				twoCol.ResizeItem(right, 0, 0)
				// expand form back to full width in twoCol
				if form != nil {
					twoCol.ResizeItem(form, 0, 2)
				}
			}
		}
		rightVisible = false
	}

	renderRightOptions := func(setFocus bool) {
		right, ok := a.views["searchRight"].(*tview.Flex)
		if !ok {
			return
		}
		if twoCol, ok := a.views["searchTwoCol"].(*tview.Flex); ok {
			// Ensure right column is visible when options are rendered
			twoCol.ResizeItem(right, 0, 1)
			// shrink form to half width
			if form != nil {
				twoCol.ResizeItem(form, 0, 1)
			}
		}
		rightVisible = true
		right.Clear()
		// Helper to pad emoji to width 2 for alignment across fonts
		padIcon := func(icon string) string {
			if runewidth.StringWidth(icon) < 2 {
				return icon + " "
			}
			return icon
		}

		type optionItem struct {
			display string
			action  func()
		}
		options := make([]optionItem, 0, 256)
		// Folders
		options = append(options,
			optionItem{padIcon("📁") + "All Mail", func() { scopeVal = "All Mail"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{padIcon("📥") + "Inbox", func() { scopeVal = "Inbox"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{padIcon("📤") + "Sent Mail", func() { scopeVal = "Sent"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{padIcon("📝") + "Drafts", func() { scopeVal = "Drafts"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{padIcon("🚫") + "Spam", func() { scopeVal = "Spam"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{padIcon("🗑️") + "Trash", func() { scopeVal = "Trash"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
		)
		// Anywhere
		options = append(options, optionItem{padIcon("📬") + "Mail & Spam & Trash", func() {
			scopeVal = "Mail & Spam & Trash"
			scopeField.SetText(scopeVal)
			hideRight()
			a.SetFocus(scopeField)
		}})
		// State
		options = append(options,
			optionItem{padIcon("✅") + "Read Mail", func() { scopeField.SetText("is:read"); hideRight(); a.SetFocus(scopeField) }},
			optionItem{padIcon("✉️") + "Unread Mail", func() { scopeField.SetText("is:unread"); hideRight(); a.SetFocus(scopeField) }},
		)
		// Categories
		for _, c := range []string{"social", "updates", "forums", "promotions"} {
			cc := c
			disp := strings.Title(cc)
			options = append(options, optionItem{padIcon("🗂️") + disp, func() { scopeField.SetText("category:" + cc); hideRight(); a.SetFocus(scopeField) }})
		}
		// Labels (all user labels in 'scopes' beyond base)
		baseSet := map[string]struct{}{"All Mail": {}, "Inbox": {}, "Sent": {}, "Drafts": {}, "Spam": {}, "Trash": {}, "Starred": {}, "Important": {}}
		for _, s := range scopes {
			if _, okb := baseSet[s]; okb {
				continue
			}
			name := s
			options = append(options, optionItem{padIcon("🔖") + name, func() { scopeField.SetText("label:\"" + name + "\""); hideRight(); a.SetFocus(scopeField) }})
		}

		filter := tview.NewInputField().SetLabel("🔎 ")
		filter.SetPlaceholder("filter options…")
		filter.SetFieldWidth(30)
		list := tview.NewList().ShowSecondaryText(false)
		list.SetBorder(false)
		// Container con borde para incluir picker + lista
		box := tview.NewFlex().SetDirection(tview.FlexRow)
		box.SetBorder(true).SetTitle(" 📂 Search options ")
		box.SetBorderColor(tcell.ColorYellow)
		acts := make([]func(), 0, 256)
		apply := func(q string) {
			ql := strings.ToLower(strings.TrimSpace(q))
			list.Clear()
			acts = acts[:0]
			for _, it := range options {
				if ql == "" || strings.Contains(strings.ToLower(it.display), ql) {
					act := it.action
					list.AddItem(it.display, "", 0, func() { act() })
					acts = append(acts, act)
				}
			}
			if list.GetItemCount() > 0 {
				list.SetCurrentItem(0)
			}
		}
		filter.SetChangedFunc(func(s string) { apply(s) })
		filter.SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEnter:
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(acts) {
					acts[idx]()
					hideRight()
					a.SetFocus(scopeField)
				}
			case tcell.KeyTab:
				a.SetFocus(list)
			}
		})
		box.AddItem(filter, 1, 0, true)
		box.AddItem(list, 0, 1, true)
		right.AddItem(box, 0, 1, true)
		apply("")
		box.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			if e.Key() == tcell.KeyEscape {
				hideRight()
				a.SetFocus(scopeField)
				return nil
			}
			if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp {
				if a.GetFocus() == filter {
					a.SetFocus(list)
					return nil
				}
			}
			return e
		})
		list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
				a.SetFocus(filter)
				return nil
			}
			return e
		})
		if setFocus {
			a.SetFocus(filter)
		}
	}
	scopeField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			if a.logger != nil {
				a.logger.Println("advsearch: scopeField Enter -> open options panel")
			}
			if rightVisible {
				hideRight()
			} else {
				renderRightOptions(true)
			}
		}
	})

	// Wire input capture after openScopePicker is defined
	scopeField.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter, tcell.KeyTab:
			if a.logger != nil {
				a.logger.Printf("advsearch: scopeField key=%v -> open options panel", e.Key())
			}
			if rightVisible {
				hideRight()
			} else {
				renderRightOptions(true)
			}
			return nil
		case tcell.KeyRune:
			if a.logger != nil {
				a.logger.Printf("advsearch: scopeField rune '%c' -> open options panel", e.Rune())
			}
			if rightVisible {
				hideRight()
			} else {
				renderRightOptions(true)
			}
			return nil
		}
		return e
	})

	// Build submit function (triggered by the Search button)
	submit := func() {
		if a.logger != nil {
			a.logger.Println("advsearch: submit invoked")
		}
		// Hide options panel if visible
		hideRight()
		from := fromField.GetText()
		to := toField.GetText()
		subject := subjectField.GetText()
		hasWords := hasField.GetText()
		notWords := notField.GetText()
		sizeExpr := sizeExprField.GetText()
		dateWithinExpr := dateWithinField.GetText()

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
		// Size (parse <NMB or >NKB) with validation
		if expr := strings.TrimSpace(sizeExpr); expr != "" {
			valid := false
			if len(expr) >= 2 && (expr[0] == '>' || expr[0] == '<') {
				op := expr[0]
				rest := strings.TrimSpace(expr[1:])
				// Extract integer number and optional unit
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
				if num != "" {
					u := strings.ToLower(unit)
					suffix := ""
					unitValid := false
					if u == "" { // assume bytes if no unit
						suffix = ""
						unitValid = true
					} else if strings.HasPrefix(u, "mb") || u == "m" {
						suffix = "m"
						unitValid = true
					} else if strings.HasPrefix(u, "kb") || u == "k" {
						suffix = "k"
						unitValid = true
					} else if u == "b" || u == "bytes" {
						suffix = ""
						unitValid = true
					}
					if unitValid {
						valid = true
						if op == '>' {
							if suffix == "" {
								parts = append(parts, fmt.Sprintf("larger:%s", num))
							} else {
								parts = append(parts, fmt.Sprintf("larger:%s%s", num, suffix))
							}
						} else {
							if suffix == "" {
								parts = append(parts, fmt.Sprintf("smaller:%s", num))
							} else {
								parts = append(parts, fmt.Sprintf("smaller:%s%s", num, suffix))
							}
						}
					}
				}
			}
			if !valid {
				a.showStatusMessage("📦 Size must be like >500KB or <2MB")
				return
			}
		}
		// Date within -> compute symmetric window around today using after:/before:
		// Accepts Nd, Nw, Nm, Ny (e.g., 3d, 1w, 2m, 1y)
		if tok := strings.TrimSpace(dateWithinExpr); tok != "" {
			numStr := ""
			unit := ""
			for i := 0; i < len(tok); i++ {
				if tok[i] >= '0' && tok[i] <= '9' {
					numStr += string(tok[i])
				} else {
					unit = strings.ToLower(strings.TrimSpace(tok[i:]))
					break
				}
			}
			valid := false
			if numStr != "" && unit != "" {
				// Parse amount
				amount := 0
				for i := 0; i < len(numStr); i++ {
					amount = amount*10 + int(numStr[i]-'0')
				}
				if amount > 0 {
					now := time.Now()
					// Anchor at local date boundaries
					anchor := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
					var start time.Time
					var endExclusive time.Time
					switch unit[0] {
					case 'd':
						start = anchor.AddDate(0, 0, -amount)
						endExclusive = anchor.AddDate(0, 0, amount+1) // +1 day for exclusive before:
						valid = true
					case 'w':
						days := amount * 7
						start = anchor.AddDate(0, 0, -days)
						endExclusive = anchor.AddDate(0, 0, days+1)
						valid = true
					case 'm':
						start = anchor.AddDate(0, -amount, 0)
						// end = anchor + amount months, then +1 day for exclusive before
						endExclusive = anchor.AddDate(0, amount, 1)
						valid = true
					case 'y':
						start = anchor.AddDate(-amount, 0, 0)
						endExclusive = anchor.AddDate(amount, 0, 1)
						valid = true
					}
					if valid {
						// Format as YYYY/M/D without zero padding to match Gmail tolerance
						format := "2006/1/2"
						parts = append(parts, fmt.Sprintf("after:%s", start.Format(format)))
						parts = append(parts, fmt.Sprintf("before:%s", endExclusive.Format(format)))
					}
				}
			}
			if !valid {
				a.showStatusMessage("⏱️ Date must be like 3d, 1w, 2m or 1y")
				return
			}
		}
		// Scope and extra operators from the Search field
		scopeText := strings.TrimSpace(scopeField.GetText())
		if scopeText == "" {
			scopeText = scopeVal // fallback to last selected scope option
		}
		switch scopeText {
		case "", "All Mail":
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
		case "Mail & Spam & Trash":
			parts = append(parts, "in:anywhere")
		default:
			// If already a valid operator, pass-through; else treat as a label token
			if strings.HasPrefix(scopeText, "in:") || strings.HasPrefix(scopeText, "is:") || strings.HasPrefix(scopeText, "category:") || strings.HasPrefix(scopeText, "label:") {
				parts = append(parts, scopeText)
			} else {
				parts = append(parts, fmt.Sprintf("label:%q", scopeText))
			}
		}
		if hasAttachment {
			parts = append(parts, "has:attachment")
		}

		q := strings.Join(parts, " ")
		if a.logger != nil {
			a.logger.Printf("advsearch: built query='%s'", q)
		}
		// If empty, keep the advanced search open and show a hint (align with simple search)
		if strings.TrimSpace(q) == "" {
			a.showStatusMessage("🔎 Search query cannot be empty")
			return
		}

		// Restore main layout (list+content) and hide advanced search panel
		if restoreLayout != nil {
			restoreLayout()
		}
		// Ensure searchPanel is hidden and cleared in listContainer
		if spv, ok := a.views["searchPanel"].(*tview.Flex); ok {
			spv.Clear()
			spv.SetBorder(false).SetTitle("")
		}
		if lc, ok := a.views["listContainer"].(*tview.Flex); ok {
			lc.Clear()
			lc.AddItem(a.views["searchPanel"], 0, 0, false)
			lc.AddItem(a.views["list"], 0, 1, true)
		}
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])
		if a.logger != nil {
			a.logger.Println("advsearch: calling performSearch")
		}
		go a.performSearch(q)
	}
	form.SetButtonsAlign(tview.AlignRight)
	form.AddButton("🔎 Search", func() {
		if a.logger != nil {
			a.logger.Println("advsearch: Search button pressed")
		}
		submit()
	})
	form.SetBorder(false) // inner form without its own title; container shows the title
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if a.logger != nil {
			idx, _ := form.GetFocusedItemIndex()
			title := ""
			if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
				title = sp.GetTitle()
			}
			a.logger.Printf("advsearch: form key=%v rune=%q focusIndex=%d title=%q", ev.Key(), ev.Rune(), idx, title)
		}
		// When a dropdown is open, intercept keys (ESC/tab/enter)
		idx, _ := form.GetFocusedItemIndex()
		if idx >= 0 {
			if _, ok := form.GetFormItem(idx).(*tview.DropDown); ok {
				if ev.Key() == tcell.KeyEscape {
					// In advanced search: first ESC closes the right options panel if open
					if rightVisible {
						hideRight()
						a.SetFocus(scopeField)
						return nil
					}
					// Otherwise fall back to exiting advanced search to simple overlay
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
		// Arrow/Tab navigation between fields at form level
		if ev.Key() == tcell.KeyDown || ev.Key() == tcell.KeyUp || ev.Key() == tcell.KeyTab || ev.Key() == tcell.KeyBacktab {
			cur, btn := form.GetFocusedItemIndex()
			items := form.GetFormItemCount()
			unified := cur
			if unified < 0 && btn >= 0 {
				unified = items + btn
			}
			next := unified
			switch ev.Key() {
			case tcell.KeyDown, tcell.KeyTab:
				next = unified + 1
			case tcell.KeyUp, tcell.KeyBacktab:
				next = unified - 1
			}
			if next < 0 {
				next = 0
			}
			// Allow navigating into buttons: treat total = fields + buttons
			total := items + form.GetButtonCount()
			if total > 0 && next >= total {
				next = total - 1
			}
			if a.logger != nil {
				a.logger.Printf("advsearch: nav %v from %d to %d (items=%d buttons=%d)", ev.Key(), unified, next, items, form.GetButtonCount())
			}
			// If focusing a button index, map to button and force focus
			if next >= items {
				btnIdx := next - items
				if btnIdx < form.GetButtonCount() {
					if btn := form.GetButton(btnIdx); btn != nil {
						a.SetFocus(btn)
						a.currentFocus = "search"
						a.updateFocusIndicators("search")
						if a.logger != nil {
							a.logger.Printf("advsearch: focusing button index=%d (next=%d items=%d)", btnIdx, next, items)
						}
						return nil
					}
				}
			}
			if item := form.GetFormItem(next); item != nil {
				if p, ok := item.(tview.Primitive); ok {
					a.SetFocus(p)
					a.currentFocus = "search"
					a.updateFocusIndicators("search")
				} else {
					form.SetFocus(next)
				}
			} else {
				form.SetFocus(next)
			}
			return nil
		}
		// Do NOT submit on Enter; solo al pulsar el botón "🔎 Search".
		if ev.Key() == tcell.KeyEscape {
			// In advanced search: first ESC closes the right options panel if open
			if rightVisible {
				hideRight()
				a.SetFocus(scopeField)
				return nil
			}
			// Otherwise, exit to simple search overlay
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

	// Mount as two vertical panes: left = advanced search, right = message content
	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		sp.Clear()
		sp.SetBorder(true).
			SetBorderColor(tcell.ColorYellow).
			SetTitle("🔎 Advanced Search").
			SetTitleColor(tcell.ColorYellow).
			SetTitleAlign(tview.AlignCenter)
		twoCol := tview.NewFlex().SetDirection(tview.FlexColumn)
		a.views["searchTwoCol"] = twoCol
		twoCol.AddItem(form, 0, 2, true)
		right := tview.NewFlex().SetDirection(tview.FlexRow)
		a.views["searchRight"] = right
		twoCol.AddItem(right, 0, 0, false) // hidden until toggle
		sp.AddItem(twoCol, 0, 1, true)

		// Helper to restore the default main layout when exiting advanced search
		restoreLayout = func() {
			if cs, ok := a.views["contentSplit"].(*tview.Flex); ok {
				cs.Clear()
				cs.SetDirection(tview.FlexColumn)
				if tc, ok2 := a.views["textContainer"].(*tview.Flex); ok2 {
					cs.AddItem(tc, 0, 1, false)
				}
				cs.AddItem(a.aiSummaryView, 0, 0, false)
				cs.AddItem(a.labelsView, 0, 0, false)
			}
			if mf, ok := a.views["mainFlex"].(*tview.Flex); ok {
				if lc, ok2 := a.views["listContainer"].(*tview.Flex); ok2 {
					mf.ResizeItem(lc, 0, 40) // restore list area
				}
			}
		}

		// Hide the message list row; show two columns instead inside contentSplit
		if mf, ok := a.views["mainFlex"].(*tview.Flex); ok {
			if lc, ok2 := a.views["listContainer"].(*tview.Flex); ok2 {
				mf.ResizeItem(lc, 0, 0)
			}
		}
		if cs, ok := a.views["contentSplit"].(*tview.Flex); ok {
			cs.Clear()
			cs.SetDirection(tview.FlexRow)
			// 50/50 split (same weights)
			cs.AddItem(sp, 0, 1, true) // top: advanced search
			if tc, ok2 := a.views["textContainer"].(*tview.Flex); ok2 {
				cs.AddItem(tc, 0, 1, false) // bottom: message content
			}
		}

		// ESC in the left pane: close options first; otherwise exit to simple overlay
		sp.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
			if ev.Key() == tcell.KeyEscape {
				if rightVisible {
					hideRight()
					a.SetFocus(scopeField)
					return nil
				}
				restoreLayout()
				a.openSearchOverlay("remote")
				return nil
			}
			return ev
		})

		a.currentFocus = "search"
		a.updateFocusIndicators("search")
		a.SetFocus(fromField)
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
	labelTokens := make([]string, 0)
	textTokens := make([]string, 0)
	for _, t := range tokens {
		if strings.HasPrefix(t, "label:") {
			v := strings.TrimSpace(strings.TrimPrefix(t, "label:"))
			if v != "" {
				labelTokens = append(labelTokens, v)
			}
		} else {
			textTokens = append(textTokens, t)
		}
	}
	filteredIDs := make([]string, 0, len(a.ids))
	filteredMeta := make([]*gmailapi.Message, 0, len(a.messagesMeta))
	rows := make([]string, 0, len(a.messagesMeta))

	// Build label ID -> name map once (best-effort)
	idToName := map[string]string{}
	if a.Client != nil {
		if labels, err := a.Client.ListLabels(); err == nil {
			for _, l := range labels {
				idToName[l.Id] = l.Name
			}
		}
	}

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
		// Collect label display names (normalize CATEGORY_* → friendly name)
		labelNames := make([]string, 0, len(m.LabelIds))
		for _, lid := range m.LabelIds {
			name := idToName[lid]
			if name == "" {
				name = lid
			}
			up := strings.ToUpper(name)
			if strings.HasPrefix(up, "CATEGORY_") {
				name = strings.TrimPrefix(name, "CATEGORY_")
			}
			labelNames = append(labelNames, strings.ToLower(name))
		}
		labelsJoined := strings.Join(labelNames, " ")
		content := strings.ToLower(subject + " " + from + " " + to + " " + m.Snippet + " " + labelsJoined)
		match := true
		// General text tokens
		for _, t := range textTokens {
			if !strings.Contains(content, t) {
				match = false
				break
			}
		}
		// label: tokens (each must match at least one label name)
		if match && len(labelTokens) > 0 {
			for _, lt := range labelTokens {
				found := false
				for _, ln := range labelNames {
					if strings.Contains(ln, lt) {
						found = true
						break
					}
				}
				if !found {
					match = false
					break
				}
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
		// Detect calendar invite parts (best-effort)
		if inv, ok := a.detectCalendarInvite(message.Message); ok {
			a.inviteCache[id] = inv
		}

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
			// If invite detected, show hint in status bar
			if _, ok := a.inviteCache[id]; ok {
				a.showStatusMessage("📅 Calendar invite detected — press V to RSVP")
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

// saveCurrentMessageRawEML saves the raw RFC 5322 message as received from Gmail API (.eml)
func (a *App) saveCurrentMessageRawEML() {
	id := a.getCurrentMessageID()
	if id == "" {
		id = a.currentMessageID
	}
	if id == "" {
		a.showError("❌ No message selected")
		return
	}
	a.setStatusPersistent("💾 Saving raw .eml…")
	go func(mid string) {
		// Fetch raw via Gmail API (format=raw)
		if a.Client == nil || a.Client.Service == nil {
			a.QueueUpdateDraw(func() { a.showError("❌ Gmail client not initialized") })
			return
		}
		user := "me"
		msg, err := a.Client.Service.Users.Messages.Get(user, mid).Format("raw").Do()
		if err != nil || msg == nil || msg.Raw == "" {
			a.QueueUpdateDraw(func() { a.showError("❌ Could not fetch raw message") })
			return
		}
		// Decode base64url raw -> bytes
		data, err := base64.URLEncoding.DecodeString(msg.Raw)
		if err != nil {
			a.QueueUpdateDraw(func() { a.showError("❌ Could not decode raw message") })
			return
		}
		// Build filename
		subj := "message"
		if m, e := a.Client.GetMessage(mid); e == nil && m != nil {
			for _, h := range m.Payload.Headers {
				if strings.EqualFold(h.Name, "Subject") && strings.TrimSpace(h.Value) != "" {
					subj = h.Value
					break
				}
			}
		}
		home, _ := os.UserHomeDir()
		base := filepath.Join(home, ".config", "gmail-tui", "saved")
		if err := os.MkdirAll(base, 0o755); err != nil {
			a.QueueUpdateDraw(func() { a.showError("❌ Could not create saved folder") })
			return
		}
		name := sanitizeFilename(subj)
		ts := time.Now().Format("20060102-150405")
		file := filepath.Join(base, ts+"-"+name+".eml")
		if err := os.WriteFile(file, data, 0o644); err != nil {
			a.QueueUpdateDraw(func() { a.showError("❌ Could not write file") })
			return
		}
		a.QueueUpdateDraw(func() { a.showStatusMessage("💾 Saved raw: " + file) })
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

		// Detect calendar invite (same as showMessage) and cache result
		if inv, ok := a.detectCalendarInvite(message.Message); ok {
			a.inviteCache[id] = inv
		}

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
			// If invite detected in preview, show the same hint
			if _, ok := a.inviteCache[id]; ok {
				a.showStatusMessage("📅 Calendar invite detected — press V to RSVP")
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

// Invite holds parsed fields from a calendar invite
type Invite struct {
	UID       string
	Summary   string
	Organizer string
	DtStart   string
	DtEnd     string
	YesURL    string
	NoURL     string
	MaybeURL  string
}

// detectCalendarInvite parses a gmailapi.Message to find text/calendar REQUEST
func (a *App) detectCalendarInvite(msg *gmailapi.Message) (Invite, bool) {
	if msg == nil || msg.Payload == nil {
		return Invite{}, false
	}
	var out Invite
	var found bool
	var walk func(p *gmailapi.MessagePart)
	walk = func(p *gmailapi.MessagePart) {
		if p == nil || found {
			return
		}
		mt := strings.ToLower(p.MimeType)
		// Support inline calendar, application/ics, octet-stream .ics attachments
		isICS := strings.Contains(mt, "text/calendar") || strings.Contains(mt, "application/ics") ||
			(p.Filename != "" && strings.HasSuffix(strings.ToLower(p.Filename), ".ics")) ||
			(strings.Contains(mt, "application/octet-stream") && strings.HasSuffix(strings.ToLower(p.Filename), ".ics"))
		if isICS {
			// Heuristic: look for METHOD:REQUEST in headers or body
			methodReq := false
			for _, h := range p.Headers {
				if strings.EqualFold(h.Name, "Content-Type") && strings.Contains(strings.ToLower(h.Value), "method=request") {
					methodReq = true
					break
				}
			}
			// Load inline or attachment data
			var raw []byte
			if p.Body != nil {
				if p.Body.Data != "" {
					if data, err := base64.URLEncoding.DecodeString(p.Body.Data); err == nil {
						raw = data
					}
				} else if p.Body.AttachmentId != "" && a.Client != nil {
					if data, _, err := a.Client.GetAttachment(msg.Id, p.Body.AttachmentId); err == nil {
						raw = data
					}
				}
			}
			if len(raw) > 0 {
				s := strings.ToUpper(string(raw))
				if strings.Contains(s, "METHOD:REQUEST") {
					methodReq = true
				}
				out.UID = scanICSField(s, "UID:")
				out.Summary = scanICSField(s, "SUMMARY:")
				out.Organizer = scanICSField(s, "ORGANIZER:")
				out.DtStart = scanICSField(s, "DTSTART")
				out.DtEnd = scanICSField(s, "DTEND")
			}
			if methodReq {
				found = true
				return
			}
		}
		// Also try to extract RSVP links from HTML part
		if strings.Contains(mt, "text/html") && p.Body != nil && p.Body.Data != "" {
			if data, err := base64.URLEncoding.DecodeString(p.Body.Data); err == nil {
				y, n, m := extractRSVPLinksFromHTML(string(data))
				if out.YesURL == "" {
					out.YesURL = y
				}
				if out.NoURL == "" {
					out.NoURL = n
				}
				if out.MaybeURL == "" {
					out.MaybeURL = m
				}
			}
		}
		for _, c := range p.Parts {
			walk(c)
			if found {
				return
			}
		}
	}
	walk(msg.Payload)
	return out, found
}

// extractRSVPLinksFromHTML finds Google Calendar RESPOND links and maps rst codes to Yes/No/Maybe
func extractRSVPLinksFromHTML(htmlStr string) (yes, no, maybe string) {
	// Very small, robust search; not a full HTML parse to keep it cheap here
	s := strings.ToLower(htmlStr)
	// Find all occurrences of calendar google respond URLs
	// We will just scan for "https://calendar.google.com/calendar/event" slices
	const marker = "https://calendar.google.com/calendar/event"
	idx := 0
	for {
		i := strings.Index(s[idx:], marker)
		if i < 0 {
			break
		}
		i += idx
		// capture until quote or whitespace
		j := i
		for j < len(s) && s[j] != '"' && s[j] != '\'' && s[j] != '>' && !unicode.IsSpace(rune(s[j])) {
			j++
		}
		u := htmlStr[i:j]
		// Check action and rst
		if strings.Contains(u, "action=respond") {
			// Map rst
			if strings.Contains(u, "rst=1") && yes == "" {
				yes = u
			} else if strings.Contains(u, "rst=2") && no == "" {
				no = u
			} else if strings.Contains(u, "rst=3") && maybe == "" {
				maybe = u
			}
		}
		idx = j
	}
	return
}

func scanICSField(s, key string) string {
	idx := strings.Index(s, key)
	if idx < 0 {
		return ""
	}
	line := s[idx:]
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, key))
	line = strings.TrimPrefix(line, ";VALUE=DATE:")
	line = strings.TrimPrefix(line, ":")
	return strings.TrimSpace(line)
}

// openRSVPModal shows a simple modal to RSVP to a detected calendar invite
func (a *App) openRSVPModal() {
	mid := a.getCurrentMessageID()
	if mid == "" {
		mid = a.currentMessageID
	}
	if mid == "" {
		a.showError("❌ No message selected")
		return
	}
	inv, ok := a.inviteCache[mid]
	if !ok {
		// Best-effort: re-detect from cache message
		if m, ok2 := a.messageCache[mid]; ok2 && m != nil {
			if parsed, ok3 := a.detectCalendarInvite(m.Message); ok3 {
				inv = parsed
				a.inviteCache[mid] = inv
			}
		}
	}
	if inv.UID == "" {
		a.showError("❌ No calendar invite found in this message")
		return
	}

	// Build side panel like labels
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	list.AddItem("✅ Accept", "Enter to send", 0, nil)
	list.AddItem("🤔 Tentative", "Enter to send", 0, nil)
	list.AddItem("❌ Decline", "Enter to send", 0, nil)
	if list.GetItemCount() > 0 {
		list.SetCurrentItem(0)
	}
	comment := tview.NewInputField().SetLabel("Comment: ").SetFieldWidth(0)
	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter=send  |  Tab=switch  |  Esc=close ")
	footer.SetTextColor(tcell.ColorGray)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true).SetTitle(" 📅 RSVP ").SetTitleColor(tcell.ColorYellow)
	container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	container.AddItem(list, 0, 1, true)
	container.AddItem(comment, 3, 0, false)
	container.AddItem(footer, 1, 0, false)

	// Key handling
	sendSelected := func() {
		idx := list.GetCurrentItem()
		choice := ""
		switch idx {
		case 0:
			choice = "ACCEPTED"
		case 1:
			choice = "TENTATIVE"
		case 2:
			choice = "DECLINED"
		}
		if choice == "" {
			return
		}
		go a.sendRSVP(choice, comment.GetText())
		// Close panel
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.labelsVisible = false
		a.rsvpVisible = false
		a.restoreFocusAfterModal()
	}
	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyTab:
			a.SetFocus(comment)
			return nil
		case tcell.KeyEnter:
			sendSelected()
			return nil
		case tcell.KeyEscape:
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.labelsView, 0, 0)
			}
			a.labelsVisible = false
			a.rsvpVisible = false
			a.restoreFocusAfterModal()
			return nil
		}
		return e
	})
	comment.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.labelsView, 0, 0)
			}
			a.labelsVisible = false
			a.rsvpVisible = false
			a.restoreFocusAfterModal()
			return
		}
		if key == tcell.KeyEnter {
			sendSelected()
			return
		}
		if key == tcell.KeyTab {
			a.SetFocus(list)
			return
		}
	})

	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			// Mount into labels slot
			split.RemoveItem(a.labelsView)
			a.labelsView = container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.labelsVisible = true
		a.rsvpVisible = true
		a.currentFocus = "labels"
		a.updateFocusIndicators("labels")
		a.SetFocus(list)
	})
}

// sendRSVP builds an iCalendar REPLY and sends it to the organizer via email
func (a *App) sendRSVP(partstat, comment string) {
	mid := a.getCurrentMessageID()
	if mid == "" {
		mid = a.currentMessageID
	}
	inv, ok := a.inviteCache[mid]
	if !ok || inv.UID == "" {
		a.showError("❌ No invite to reply to")
		return
	}
	// Resolve organizer email
	orgEmail := extractEmailFromOrganizer(inv.Organizer)
	if orgEmail == "" {
		a.showError("❌ Could not determine organizer email")
		return
	}
	// Resolve our account email
	fromEmail, _ := a.Client.ActiveAccountEmail(a.ctx)
	if strings.TrimSpace(fromEmail) == "" {
		fromEmail = "me"
	}
	// Subject
	subject := inv.Summary
	if subject == "" {
		subject = "Meeting"
	}
	subjPrefix := map[string]string{"ACCEPTED": "Accepted: ", "TENTATIVE": "Tentative: ", "DECLINED": "Declined: "}
	if p, ok := subjPrefix[partstat]; ok {
		subject = p + subject
	}

	// Plain text body
	plain := "RSVP: " + strings.ToLower(partstat)
	if strings.TrimSpace(comment) != "" {
		plain += "\n\n" + comment
	}

	// ICS reply content (add SEQUENCE, optional DTSTART/DTEND; RSVP=TRUE)
	now := time.Now().UTC().Format("20060102T150405Z")
	attendee := fmt.Sprintf("ATTENDEE;CUTYPE=INDIVIDUAL;CN=%s;ROLE=REQ-PARTICIPANT;PARTSTAT=%s;RSVP=TRUE:mailto:%s",
		fromEmail, partstat, fromEmail)
	// Try to include CN for organizer as some servers require it
	organizerField := sanitizeOrganizer(inv.Organizer)
	organizerCN := extractEmailFromOrganizer(inv.Organizer)
	if organizerCN == "" {
		organizerCN = organizerField
	}
	organizerLine := fmt.Sprintf("ORGANIZER;CN=%s:%s", organizerCN, organizerField)
	vevent := []string{
		"BEGIN:VEVENT",
		"UID:" + inv.UID,
		"SEQUENCE:0",
		"DTSTAMP:" + now,
		organizerLine,
		attendee,
	}
	if strings.TrimSpace(inv.DtStart) != "" {
		vevent = append(vevent, "DTSTART:"+inv.DtStart)
	}
	if strings.TrimSpace(inv.DtEnd) != "" {
		vevent = append(vevent, "DTEND:"+inv.DtEnd)
	}
	if strings.TrimSpace(inv.Summary) != "" {
		vevent = append(vevent, "SUMMARY:"+inv.Summary)
	}
	vevent = append(vevent, "END:VEVENT")
	ics := strings.Join(append([]string{
		"BEGIN:VCALENDAR",
		"METHOD:REPLY",
		"PRODID:-//gmail-tui//RSVP 1.0//EN",
		"VERSION:2.0",
		"CALSCALE:GREGORIAN",
	}, append(vevent, "END:VCALENDAR", "")...), "\r\n")

	boundary := fmt.Sprintf("mime-boundary-%d", time.Now().UnixNano())
	raw := strings.Builder{}
	// Threading headers can help Gmail correlate the reply with the invite
	var inReplyTo, references string
	if m, ok := a.messageCache[mid]; ok && m != nil && m.Message != nil {
		for _, h := range m.Message.Payload.Headers {
			if strings.EqualFold(h.Name, "Message-Id") || strings.EqualFold(h.Name, "Message-ID") {
				inReplyTo = strings.TrimSpace(h.Value)
			}
		}
		if inReplyTo != "" {
			references = inReplyTo
		}
	}
	raw.WriteString(fmt.Sprintf("From: %s\r\n", fromEmail))
	raw.WriteString(fmt.Sprintf("To: %s\r\n", orgEmail))
	// Help Gmail route the reply
	raw.WriteString(fmt.Sprintf("Cc: %s\r\n", "calendar-notification@google.com"))
	raw.WriteString(fmt.Sprintf("Reply-To: %s\r\n", orgEmail))
	if inReplyTo != "" {
		raw.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", inReplyTo))
	}
	if references != "" {
		raw.WriteString(fmt.Sprintf("References: %s\r\n", references))
	}
	raw.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	// Calendar specific headers (some servers look at these)
	raw.WriteString("Content-Class: urn:content-classes:calendarmessage\r\n")
	raw.WriteString("MIME-Version: 1.0\r\n")
	raw.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary))
	// part 1: text/plain
	raw.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	raw.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	raw.WriteString(plain + "\r\n")
	// part 2: text/calendar REPLY
	raw.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	raw.WriteString("Content-Type: text/calendar; charset=UTF-8; method=REPLY\r\n")
	raw.WriteString("Content-Class: urn:content-classes:calendarmessage\r\n")
	raw.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	raw.WriteString(ics)
	raw.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	a.setStatusPersistent("📤 Sending RSVP…")
	go func() {
		if _, err := a.Client.SendRawMIME(raw.String()); err != nil {
			a.QueueUpdateDraw(func() { a.showError(fmt.Sprintf("❌ RSVP failed: %v", err)) })
			return
		}
		// Fallback: hit Google Calendar RESPOND link to force state update
		if inv.YesURL != "" || inv.NoURL != "" || inv.MaybeURL != "" {
			target := inv.YesURL
			switch partstat {
			case "DECLINED":
				target = inv.NoURL
			case "TENTATIVE":
				target = inv.MaybeURL
			}
			if strings.TrimSpace(target) != "" {
				// Best-effort GET (no cookies required for RSVP links received to this account)
				_ = doHTTPGetNoFollow(target)
			}
		}
		a.QueueUpdateDraw(func() { a.showStatusMessage("✅ RSVP sent") })
	}()
}

// doHTTPGetNoFollow performs a simple GET without following redirects; ignore errors
func doHTTPGetNoFollow(url string) error {
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	// Minimal UA
	req.Header.Set("User-Agent", "gmail-tui/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

func extractEmailFromOrganizer(org string) string {
	s := strings.ToLower(strings.TrimSpace(org))
	// Patterns like ORGANIZER:mailto:foo@bar or ORGANIZER;CN=..:mailto:foo@bar
	if i := strings.LastIndex(s, "mailto:"); i >= 0 {
		addr := strings.TrimSpace(s[i+7:])
		// cut params after email
		for _, sep := range []string{";", "\\r", "\\n"} {
			if j := strings.Index(addr, sep); j >= 0 {
				addr = addr[:j]
			}
		}
		return addr
	}
	// Fallback: extract last token with @
	toks := strings.Fields(s)
	for _, t := range toks {
		if strings.Contains(t, "@") {
			return strings.Trim(t, ":<>")
		}
	}
	return ""
}

func sanitizeOrganizer(org string) string {
	if strings.Contains(strings.ToUpper(org), "ORGANIZER:") {
		return strings.TrimSpace(strings.SplitN(org, ":", 2)[1])
	}
	return org
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
