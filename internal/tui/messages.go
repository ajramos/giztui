package tui

import (
	"fmt"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// reformatListItems recalculates list item strings for current screen width
func (a *App) reformatListItems() {
	list, ok := a.views["list"].(*tview.List)
	if !ok || len(a.ids) == 0 {
		return
	}
	for i := range a.ids {
		if i >= len(a.messagesMeta) || a.messagesMeta[i] == nil {
			continue
		}
		msg := a.messagesMeta[i]
		text, _ := a.emailRenderer.FormatEmailList(msg, a.screenWidth)
		unread := false
		for _, l := range msg.LabelIds {
			if l == "UNREAD" {
				unread = true
				break
			}
		}
		// Prefixes: unread marker always, selection checkbox only in bulk mode
		if a.bulkMode {
			sel := "☐ "
			if a.selected != nil && a.selected[a.ids[i]] {
				sel = "☑ "
			}
			if unread {
				text = sel + "● " + text
			} else {
				text = sel + "○ " + text
			}
		} else {
			if unread {
				text = "● " + text
			} else {
				text = "○ " + text
			}
		}
		list.SetItemText(i, text, "")
	}
}

// reloadMessages loads messages from the inbox
func (a *App) reloadMessages() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.draftMode = false
	if list, ok := a.views["list"].(*tview.List); ok {
		list.Clear()
	}
	a.ids = []string{}
	a.messagesMeta = []*gmailapi.Message{}

	// Show loading message
	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetTitle(" 🔄 Loading messages... ")
	}
	a.Draw()

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
			if list, ok := a.views["list"].(*tview.List); ok {
				list.SetTitle(" 📧 No messages found ")
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
			if list, ok := a.views["list"].(*tview.List); ok {
				list.AddItem(fmt.Sprintf("⚠️  Error loading message %d", i+1), "Failed to load", 0, nil)
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

		if list, ok := a.views["list"].(*tview.List); ok {
			// Add item with color using the standard method
			list.AddItem(formattedText, "", 0, nil)
		}

		// cache meta for resize re-rendering
		a.messagesMeta = append(a.messagesMeta, message)

		// Update title periodically
		if (i+1)%10 == 0 {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.SetTitle(fmt.Sprintf(" 🔄 Loading... (%d/%d) ", i+1, len(messages)))
			}
			a.Draw()
		}
	}

	a.QueueUpdateDraw(func() {
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
			// Also update status with current selection
			idx := list.GetCurrentItem()
			if idx >= 0 && len(a.ids) > 0 {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", idx+1, len(a.ids)))
			}
			// Ensure a sane initial selection (0) so dependent features work on first run
			if list.GetItemCount() > 0 && idx < 0 {
				list.SetCurrentItem(0)
			}
		}
	})

	// Do not steal focus if user moved to another pane (e.g., labels/summary/text)
	if pageName, _ := a.Pages.GetFrontPage(); pageName == "main" {
		if a.currentFocus == "" || a.currentFocus == "list" {
			a.SetFocus(a.views["list"])
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
		}
	}
}

// loadMoreMessages fetches the next page of inbox and appends to list
func (a *App) loadMoreMessages() {
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
		text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
		unread := false
		for _, l := range meta.LabelIds {
			if l == "UNREAD" {
				unread = true
				break
			}
		}
		if unread {
			text = "● " + text
		} else {
			text = "○ " + text
		}
		if list, ok := a.views["list"].(*tview.List); ok {
			list.AddItem(text, "", 0, nil)
		}
	}
	a.nextPageToken = next
	a.QueueUpdateDraw(func() {
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
			idx := list.GetCurrentItem()
			if idx >= 0 && len(a.ids) > 0 {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", idx+1, len(a.ids)))
			}
		}
	})
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
	if a.currentFocus == "list" {
		if list, ok := a.views["list"].(*tview.List); ok {
			selectedIndex := list.GetCurrentItem()
			if selectedIndex >= 0 && selectedIndex < len(a.ids) {
				return a.ids[selectedIndex]
			}
		}
	} else if a.currentFocus == "text" {
		if list, ok := a.views["list"].(*tview.List); ok {
			selectedIndex := list.GetCurrentItem()
			if selectedIndex >= 0 && selectedIndex < len(a.ids) {
				return a.ids[selectedIndex]
			}
		}
	}
	return ""
}

// getListWidth returns current inner width of the list view or a sensible fallback
func (a *App) getListWidth() int {
	if list, ok := a.views["list"].(*tview.List); ok {
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
	if list, ok := a.views["list"].(*tview.List); ok {
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
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex = list.GetCurrentItem()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex = list.GetCurrentItem()
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
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			return
		}
		count := list.GetItemCount()
		if count == 0 {
			return
		}

		// Determine index to remove; prefer current selection
		removeIndex := list.GetCurrentItem()
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
			list.SetCurrentItem(pre)
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
			if removeIndex >= 0 && removeIndex < list.GetItemCount() {
				list.RemoveItem(removeIndex)
			}
			// next already set to pre; clamp to new count
			if next >= 0 && next < list.GetItemCount() {
				list.SetCurrentItem(next)
			}
		}

		// Update title and content
		list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		if text, ok := a.views["text"].(*tview.TextView); ok {
			if next >= 0 && next < len(a.ids) {
				go a.showMessageWithoutFocus(a.ids[next])
			} else {
				text.SetText("No messages")
				text.ScrollToBeginning()
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
			if list, ok := a.views["list"].(*tview.List); ok {
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
						if i < list.GetItemCount() {
							list.RemoveItem(i)
						}
						continue
					}
					i++
				}
				list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
				// Adjust selection and content
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
	var messageID string
	var selectedIndex int = -1
	if a.currentFocus == "list" {
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex = list.GetCurrentItem()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex = list.GetCurrentItem()
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
	isUnread := false
	for _, l := range message.LabelIds {
		if l == "UNREAD" {
			isUnread = true
			break
		}
	}
	if isUnread {
		if err := a.Client.MarkAsRead(messageID); err != nil {
			a.showError(fmt.Sprintf("❌ Error marking as read: %v", err))
		} else {
			a.showStatusMessage("✅ Message marked as read")
		}
	} else {
		if err := a.Client.MarkAsUnread(messageID); err != nil {
			a.showError(fmt.Sprintf("❌ Error marking as unread: %v", err))
		} else {
			a.showStatusMessage("✅ Message marked as unread")
		}
	}
}

// listUnreadMessages placeholder
func (a *App) listUnreadMessages() { a.showInfo("Unread messages functionality not yet implemented") }

// loadDrafts placeholder
func (a *App) loadDrafts() { a.showInfo("Drafts functionality not yet implemented") }

// composeMessage placeholder
func (a *App) composeMessage(draft bool) {
	a.showInfo("Compose message functionality not yet implemented")
}
