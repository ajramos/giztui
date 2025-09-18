package tui

import (
	"encoding/base64"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	calclient "github.com/ajramos/giztui/internal/calendar"
	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/render"
	"github.com/ajramos/giztui/internal/services"
	"github.com/ajramos/giztui/pkg/auth"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	cal "google.golang.org/api/calendar/v3"
	gmailapi "google.golang.org/api/gmail/v1"
)

// reformatListItems recalculates list item strings for current screen width
func (a *App) reformatListItems() {
	_, ok := a.views["list"].(*tview.Table)
	if !ok || len(a.ids) == 0 {
		return
	}

	// Skip reformatting in thread mode - threads have their own formatting
	if a.GetCurrentThreadViewMode() == ThreadViewThread {
		a.reformatThreadItems()
		return
	}

	// Use the new column-based system for flat mode
	a.refreshTableDisplay()
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
		// Account for header row (row 0 is header, messages start at row 1)
		messageIndex := row - 1
		if messageIndex >= 0 && messageIndex < len(a.ids) {
			selID = a.ids[messageIndex]
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

	if a.logger != nil {
		a.logger.Printf("🔍 ESC: Restoring base snapshot with %d messages (was searchMode=%q)", len(ids), a.searchMode)
	}

	a.QueueUpdateDraw(func() {
		a.searchMode = ""
		a.currentQuery = ""
		a.localFilter = ""
		a.nextPageToken = next
		a.SetMessageIDs(ids)
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
				var prefix string

				// Add message number if enabled (leftmost position)
				if a.showMessageNumbers {
					maxNumber := len(a.ids)
					width := len(fmt.Sprintf("%d", maxNumber))
					prefix = fmt.Sprintf("%*d ", width, i+1) // Right-aligned numbering
				}

				// Add read/unread indicator
				if unread {
					prefix += "● "
				} else {
					prefix += "○ "
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
		a.refreshTableDisplay()

		// Ensure focus is properly set to list after restore
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])

		if a.logger != nil {
			a.logger.Printf("🔍 ESC: Focus restored to list after base snapshot restoration")
		}
	})
}

// exitSearch handles ESC from search contexts
func (a *App) exitSearch() {
	if a.searchMode == "local" {
		if a.logger != nil {
			a.logger.Printf("🔍 ESC: exitSearch for local search - hiding container and restoring data")
		}
		// Hide search container first, then restore data
		if mainFlex, ok := a.views["mainFlex"].(*tview.Flex); ok {
			if searchContainer, ok := a.views["searchContainer"]; ok {
				mainFlex.ResizeItem(searchContainer, 0, 0) // Hide container
				if a.logger != nil {
					a.logger.Printf("🔍 ESC: Search container hidden via ResizeItem(0,0)")
				}
			}
		}
		delete(a.views, "searchInput") // Remove search input from views
		a.restoreLocalBaseSnapshot()
		return
	}
	if a.searchMode == "remote" {
		if a.logger != nil {
			a.logger.Printf("🔍 ESC: exitSearch for remote search - hiding container and reloading inbox")
		}
		// Hide search container first, then reload inbox from server
		if mainFlex, ok := a.views["mainFlex"].(*tview.Flex); ok {
			if searchContainer, ok := a.views["searchContainer"]; ok {
				mainFlex.ResizeItem(searchContainer, 0, 0) // Hide container
				if a.logger != nil {
					a.logger.Printf("🔍 ESC: Remote search container hidden via ResizeItem(0,0)")
				}
			}
		}
		delete(a.views, "searchInput") // Remove search input from views

		// Clear search state
		a.searchMode = ""
		a.currentQuery = ""
		a.localFilter = ""
		a.nextPageToken = ""
		go a.reloadMessages()
		return
	}
}

// reloadMessages loads messages from the inbox, respecting current threading mode
func (a *App) reloadMessages() {

	// Check if we're in threading mode and should reload threads instead
	if a.IsThreadingEnabled() && a.GetCurrentThreadViewMode() == ThreadViewThread {
		a.reloadThreadsWithSpinner()
		return
	}

	// Otherwise reload messages in flat mode
	a.reloadMessagesFlat()
}

// reloadThreadsWithSpinner reloads threads with animated spinner like flat messages
func (a *App) reloadThreadsWithSpinner() {
	// Set loading state like reloadMessagesFlat does
	a.SetMessagesLoading(true)

	// Clear list UI and prepare for animated spinner
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.Clear()
			table.SetTitle(" Loading conversations... ")
		}
	})

	// Clear cached data like reloadMessagesFlat does
	a.ClearMessageIDs()
	a.mu.Lock()
	a.messagesMeta = []*gmailapi.Message{}
	a.mu.Unlock()

	// Create animated spinner with progress tracking like in flat mode
	var spinnerStop chan struct{}
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
					frame := frames[i%len(frames)]
					// Show current progress based on loaded conversations
					currentCount := len(a.ids)
					a.QueueUpdateDraw(func() {
						if table, ok := a.views["list"].(*tview.Table); ok {
							if currentCount > 0 {
								table.SetTitle(fmt.Sprintf(" %s Loading conversations %d... ", frame, currentCount))
							} else {
								table.SetTitle(fmt.Sprintf(" %s Loading conversations... ", frame))
							}
						}
					})
					i++
				}
			}
		}()
	}

	// Call the actual thread refresh with spinner cleanup
	go func() {
		a.refreshThreadView()

		// Stop the spinner when done
		if spinnerStop != nil {
			close(spinnerStop)
		}

		// Mark loading as complete
		a.SetMessagesLoading(false)
	}()
}

// reloadMessagesFlat loads messages in flat mode (original implementation)
func (a *App) reloadMessagesFlat() {
	// Set loading state
	a.SetMessagesLoading(true)

	// Reset modes/state with minimal locking
	a.mu.Lock()
	a.draftMode = false
	a.mu.Unlock()

	// Clear list UI and state safely on UI thread
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.Clear()
			table.SetTitle(" 🔄 Loading messages... ")
		}
	})
	// Clear cached IDs using thread-safe method
	a.ClearMessageIDs()
	// Clear message metadata under lock
	a.mu.Lock()
	a.messagesMeta = []*gmailapi.Message{}
	a.mu.Unlock()

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

	// Use current query for reload to maintain current view context
	query := a.currentQuery

	// Fallback: if currentQuery is empty but we're clearly in a specific folder,
	// try to detect the folder from current message and construct appropriate query
	if query == "" {
		query = a.detectCurrentFolderQuery()
	}

	if a.logger != nil {
		a.logger.Printf("RELOAD_MSG: Loading messages with query: '%s'", query)
	}

	messages, next, err := a.Client.ListMessagesPage(50, query)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("RELOAD_MSG: Error loading messages: %v", err)
		}
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

	// Preload labels once for renderer context (avoid per-row API calls)
	if labels, err := a.Client.ListLabels(); err == nil {
		m := make(map[string]string, len(labels))
		for _, l := range labels {
			m[l.Id] = l.Name
		}
		a.emailRenderer.SetLabelMap(m)
		a.emailRenderer.SetShowSystemLabelsInList(a.searchMode == "remote")
	}

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

	// Collect message IDs for parallel fetching
	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.Id
		a.AppendMessageID(msg.Id)
	}

	// Fetch message metadata in parallel (optimized for list display - uses format=metadata)
	detailedMessages, err := a.Client.GetMessagesMetadataParallel(messageIDs, 10)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading messages: %v", err))
		return
	}

	// Process fetched messages and update UI
	for i, message := range detailedMessages {
		if message == nil {
			// Handle failed fetch
			a.QueueUpdateDraw(func() {
				if table, ok := a.views["list"].(*tview.Table); ok {
					table.SetCell(i, 0, tview.NewTableCell(fmt.Sprintf("⚠️  Error loading message %d", i+1)))
				}
			})
			continue
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

		// Build prefix with message number (if enabled) and read/unread indicator
		var prefix string
		if a.showMessageNumbers {
			maxNumber := len(a.ids)
			width := len(fmt.Sprintf("%d", maxNumber))
			prefix = fmt.Sprintf("%*d ", width, i+1) // Right-aligned numbering
		}
		if unread {
			prefix += "● "
		} else {
			prefix += "○ "
		}
		formattedText = prefix + formattedText

		// cache meta for resize re-rendering (protect shared slice)
		a.mu.Lock()
		a.messagesMeta = append(a.messagesMeta, message)
		a.mu.Unlock()

		// Paint row and apply colors immediately
		a.QueueUpdateDraw(func() {
			if table, ok := a.views["list"].(*tview.Table); ok {
				cell := tview.NewTableCell(formattedText).
					SetExpansion(1).
					SetBackgroundColor(a.GetComponentColors("general").Background.Color())
				table.SetCell(i, 0, cell)
			}
			a.refreshTableDisplay()
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

			// Always ensure the first message is selected when loading messages
			if table.GetRowCount() > 1 && len(a.ids) > 0 {
				firstID := a.ids[0] // Define firstID here so it's available for both conditions

				// Only auto-select first message if composition panel is not active
				if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
					if a.logger != nil {
						a.logger.Printf("📧 MESSAGE LOAD: Auto-selecting first message (composer not active)")
					}
					// Force selection of first message (row 1, since row 0 is header)
					table.Select(1, 0)

					// Set the current message ID to the first message
					a.SetCurrentMessageID(firstID)

					// Auto-load content for the first message
					go a.showMessageWithoutFocus(firstID)
				} else {
					if a.logger != nil {
						a.logger.Printf("📧 MESSAGE LOAD: Skipping auto-select - composer is active")
					}
				}

				// Generate AI summary if panel is visible
				if a.aiSummaryVisible {
					go a.generateOrShowSummary(firstID)
				}
			}
		}
		// Final pass (in case of resize between frames)
		a.refreshTableDisplay()
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
		// Force focus to list unless advanced search is visible OR composer is active
		if spt, ok := a.views["searchPanel"].(*tview.Flex); !ok || spt.GetTitle() != "🔎 Advanced Search" { // OBLITERATED: De Morgan's law applied! 💥
			// Only set focus if composition panel is not active
			if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
				if a.logger != nil {
					a.logger.Printf("📧 RELOAD: Setting focus to list (no advanced search, no composer)")
				}
				a.currentFocus = "list"
				a.updateFocusIndicators("list")
				a.SetFocus(a.views["list"])
			} else {
				if a.logger != nil {
					a.logger.Printf("📧 RELOAD: Skipping focus to list - composer is active")
				}
			}
		}
	})

	// Do not steal focus if user moved to another pane (e.g., labels/summary/text)
	// Keep currentFocus list during loading; focus is enforced above on completion

	// Mark loading as complete
	a.SetMessagesLoading(false)
}

// loadMoreMessages fetches the next page of inbox and appends to list, respecting current threading mode
func (a *App) loadMoreMessages() {
	// Check if we're in threading mode - threads don't support pagination yet
	if a.IsThreadingEnabled() && a.GetCurrentThreadViewMode() == ThreadViewThread {
		if a.logger != nil {
			a.logger.Printf("loadMoreMessages: currently in thread mode, pagination not supported")
		}
		go func() {
			a.GetErrorHandler().ShowWarning(a.ctx, "Load more not available in threading mode - use reload instead")
		}()
		return
	}
	// If in remote search mode, paginate that query
	if a.searchMode == "remote" {
		if a.nextPageToken == "" {
			a.showStatusMessage("No more results")
			return
		}

		a.setStatusPersistent("Loading more results…")

		// Try to use cached results first (with token preservation)
		if preloader := a.GetPreloaderService(); preloader != nil {
			// Use cache key with query to match how preloader stores search results
			cacheKey := a.currentQuery + ":" + a.nextPageToken
			if a.logger != nil {
				a.logger.Printf("CACHE SEARCH: Checking for cached search results with key='%s'", cacheKey)
			}
			startTime := time.Now()
			if cachedMessages, nextToken, found := preloader.GetCachedMessagesWithToken(a.ctx, cacheKey); found {
				loadTime := time.Since(startTime)
				if a.logger != nil {
					a.logger.Printf("🎯 CACHE HIT (SEARCH): Found %d preloaded messages in %v, nextToken='%s'", len(cachedMessages), loadTime, nextToken)
				}
				// Process cached messages directly (already have metadata)
				screenWidth := a.getFormatWidth()
				for _, meta := range cachedMessages {
					if meta == nil {
						continue // Skip failed cached messages
					}
					a.AppendMessageID(meta.Id) // Add to IDs list
					a.mu.Lock()
					a.messagesMeta = append(a.messagesMeta, meta)
					a.mu.Unlock()

					// Add to table
					if table, ok := a.views["list"].(*tview.Table); ok {
						row := table.GetRowCount()
						text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
						cell := tview.NewTableCell(text).
							SetExpansion(1).
							SetBackgroundColor(a.GetComponentColors("general").Background.Color())
						table.SetCell(row, 0, cell)
					}
				}
				a.nextPageToken = nextToken

				// Clear loading status and refresh UI
				go func() {
					a.GetErrorHandler().ClearPersistentMessage()
				}()

				a.QueueUpdateDraw(func() {
					if table, ok := a.views["list"].(*tview.Table); ok {
						table.SetTitle(fmt.Sprintf(" 📧 Search: %s (%d) ", a.currentQuery, len(a.ids)))
					}
					a.refreshTableDisplay()
					// FOCUS FIX: Restore focus to message list after loading cached search results
					a.SetFocus(a.views["list"])
					a.currentFocus = "list"
					a.updateFocusIndicators("list")
				})
				return
			}
		}

		// Cache miss - fetch from API
		if a.logger != nil {
			a.logger.Printf("❌ CACHE MISS (SEARCH): No cached results for key='%s', fetching from API", a.currentQuery+":"+a.nextPageToken)
		}
		apiStartTime := time.Now()
		messages, next, err := a.Client.SearchMessagesPage(a.currentQuery, 50, a.nextPageToken)
		apiLoadTime := time.Since(apiStartTime)
		if a.logger != nil && err == nil {
			a.logger.Printf("🌐 API FETCH (SEARCH): Loaded %d messages from API in %v", len(messages), apiLoadTime)
		}
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading more: %v", err))
			return
		}
		a.appendMessages(messages)
		a.nextPageToken = next

		// FOCUS FIX: Restore focus to message list after loading more search results
		a.QueueUpdateDraw(func() {
			a.SetFocus(a.views["list"])
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
		})
		return
	}

	if a.nextPageToken == "" {
		a.showStatusMessage("No more messages")
		return
	}

	a.setStatusPersistent("Loading next 50 messages…")

	// Try to use cached results first (with token preservation)
	if preloader := a.GetPreloaderService(); preloader != nil {
		if a.logger != nil {
			a.logger.Printf("CACHE INBOX: Checking for cached inbox results with token='%s'", a.nextPageToken)
		}
		startTime := time.Now()
		if cachedMessages, nextToken, found := preloader.GetCachedMessagesWithToken(a.ctx, a.nextPageToken); found {
			loadTime := time.Since(startTime)
			if a.logger != nil {
				a.logger.Printf("🎯 CACHE HIT (INBOX): Found %d preloaded messages in %v, nextToken='%s'", len(cachedMessages), loadTime, nextToken)
			}
			// Process cached messages directly (already have metadata)
			screenWidth := a.getFormatWidth()

			// Preload labels once for this page
			if labels, err := a.Client.ListLabels(); err == nil {
				m := make(map[string]string, len(labels))
				for _, l := range labels {
					m[l.Id] = l.Name
				}
				a.emailRenderer.SetLabelMap(m)
				a.emailRenderer.SetShowSystemLabelsInList(a.searchMode == "remote")
			}

			for _, meta := range cachedMessages {
				if meta == nil {
					continue // Skip failed cached messages
				}
				a.AppendMessageID(meta.Id) // Add to IDs list
				a.mu.Lock()
				a.messagesMeta = append(a.messagesMeta, meta)
				a.mu.Unlock()

				// Add to table
				if table, ok := a.views["list"].(*tview.Table); ok {
					row := table.GetRowCount()
					text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
					cell := tview.NewTableCell(text).
						SetExpansion(1).
						SetBackgroundColor(a.GetComponentColors("general").Background.Color())
					table.SetCell(row, 0, cell)
				}
			}
			a.nextPageToken = nextToken

			// Clear loading status and refresh UI
			go func() {
				a.GetErrorHandler().ClearPersistentMessage()
			}()

			a.QueueUpdateDraw(func() {
				if table, ok := a.views["list"].(*tview.Table); ok {
					table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
				}
				a.refreshTableDisplay()
				// FOCUS FIX: Restore focus to message list after loading cached messages
				a.SetFocus(a.views["list"])
				a.currentFocus = "list"
				a.updateFocusIndicators("list")
			})
			return
		}
	}

	// Cache miss - fetch from API
	if a.logger != nil {
		a.logger.Printf("❌ CACHE MISS (INBOX): No cached results for token='%s', fetching from API", a.nextPageToken)
	}
	apiStartTime := time.Now()
	messages, next, err := a.Client.ListMessagesPage(50, a.nextPageToken)
	apiLoadTime := time.Since(apiStartTime)
	if a.logger != nil && err == nil {
		a.logger.Printf("🌐 API FETCH (INBOX): Loaded %d messages from API in %v", len(messages), apiLoadTime)
	}
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
	// Preload labels once for this page
	if labels, err := a.Client.ListLabels(); err == nil {
		m := make(map[string]string, len(labels))
		for _, l := range labels {
			m[l.Id] = l.Name
		}
		a.emailRenderer.SetLabelMap(m)
		a.emailRenderer.SetShowSystemLabelsInList(a.searchMode == "remote")
	}
	// Collect message IDs for parallel fetching
	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.Id
		a.AppendMessageID(msg.Id)
	}

	// Fetch message metadata in parallel (optimized for list display)
	detailedMessages, err := a.Client.GetMessagesMetadataParallel(messageIDs, 10)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading more messages: %v", err))
		return
	}

	for _, meta := range detailedMessages {
		if meta == nil {
			continue // Skip failed fetches
		}

		a.mu.Lock()
		a.messagesMeta = append(a.messagesMeta, meta)
		a.mu.Unlock()

		// Set placeholder cell; colors will be applied by reformatListItems below
		if table, ok := a.views["list"].(*tview.Table); ok {
			row := table.GetRowCount()
			text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
			cell := tview.NewTableCell(text).
				SetExpansion(1).
				SetBackgroundColor(a.GetComponentColors("general").Background.Color())
			table.SetCell(row, 0, cell)
		}
		loaded++
	}
	a.nextPageToken = next

	// Clear the persistent loading status before updating UI
	go func() {
		a.GetErrorHandler().ClearPersistentMessage()
	}()

	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
		}
		a.refreshTableDisplay()
		if spinnerStop != nil {
			close(spinnerStop)
		}

		// FOCUS FIX: Restore focus to message list after loading more messages
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	})
}

// appendMessages adds messages to current table from a slice of gmail.Message (IDs)
func (a *App) appendMessages(messages []*gmailapi.Message) {
	// Collect message IDs for parallel fetching
	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.Id
		a.AppendMessageID(msg.Id)
	}

	// Fetch message metadata in parallel (optimized for list display)
	detailedMessages, err := a.Client.GetMessagesMetadataParallel(messageIDs, 10)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading message details: %v", err))
		return
	}

	screenWidth := a.getFormatWidth()
	for _, meta := range detailedMessages {
		if meta == nil {
			continue // Skip failed fetches
		}

		a.messagesMeta = append(a.messagesMeta, meta)
		if table, ok := a.views["list"].(*tview.Table); ok {
			row := table.GetRowCount()
			text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
			cell := tview.NewTableCell(text).
				SetExpansion(1).
				SetBackgroundColor(a.GetComponentColors("general").Background.Color())
			table.SetCell(row, 0, cell)
		}
	}
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))

			// Ensure there's always a valid selection when appending messages
			if table.GetRowCount() > 0 && len(a.ids) > 0 {
				currentRow, _ := table.GetSelection()
				// If no selection or selection is invalid, select the first message
				if currentRow < 1 || currentRow >= table.GetRowCount() {
					// Only auto-select if composition panel is not active
					if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
						if a.logger != nil {
							a.logger.Printf("📧 APPEND: Auto-selecting first message (invalid selection, composer not active)")
						}
						table.Select(1, 0) // Select first message (row 1, since row 0 is header)
						// Update current message ID if not set
						if a.GetCurrentMessageID() == "" && len(a.ids) > 0 {
							firstID := a.ids[0]
							a.SetCurrentMessageID(firstID)
							go a.showMessageWithoutFocus(firstID)
						}
					} else {
						if a.logger != nil {
							a.logger.Printf("📧 APPEND: Skipping auto-select - composer is active")
						}
					}
				}
			}
		}
		a.refreshTableDisplay()
	})
}

// openSearchOverlay populates the existing persistent search container (like Slack panel)
func (a *App) openSearchOverlay(mode string) {
	// Hide advanced search if it's visible (mutual exclusion)
	if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
		if sp.GetTitle() == "🔎 Advanced Search" {
			sp.Clear()
			sp.SetBorder(false).SetTitle("")
		}
	}

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

	// Get the existing persistent search container (created in initComponents like textContainer)
	searchContainer, ok := a.views["searchContainer"].(*tview.Flex)
	if !ok {
		return
	}

	// Get component colors using hierarchical theme system
	searchColors := a.GetComponentColors("search")

	// Clear and populate the persistent container (like Slack does)
	searchContainer.Clear()

	// Apply background color to main container
	bgColor := searchColors.Background.Color()
	searchContainer.SetBackgroundColor(bgColor)

	// Update container title dynamically (like Slack does)
	searchContainer.SetTitle(" " + title + " ").
		SetTitleColor(searchColors.Title.Color())

	input := tview.NewInputField().
		SetLabel("🔍 ").
		SetFieldWidth(0).
		SetPlaceholder(ph).
		SetFieldBackgroundColor(searchColors.Background.Color()).
		SetFieldTextColor(searchColors.Text.Color()).
		SetLabelColor(searchColors.Title.Color()).
		SetPlaceholderTextColor(a.getHintColor())
	input.SetBackgroundColor(bgColor)
	// expose input so Tab from list can focus it
	a.views["searchInput"] = input

	help := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	help.SetTextColor(searchColors.Text.Color()).SetBackgroundColor(searchColors.Background.Color())
	if mode == "remote" {
		help.SetText("Press Ctrl+F for advanced search | Enter=search, Ctrl-T=switch, ESC to back")
	} else {
		help.SetText("Type space-separated terms; all must match | Enter=apply, Ctrl-T=switch, ESC to back")
	}

	// Add content to the persistent container (like Slack panel)
	topSpacer := tview.NewBox().SetBackgroundColor(searchColors.Background.Color())
	bottomSpacer := tview.NewBox().SetBackgroundColor(searchColors.Background.Color())
	searchContainer.AddItem(topSpacer, 0, 1, false)
	searchContainer.AddItem(input, 1, 0, true)
	searchContainer.AddItem(bottomSpacer, 0, 2, false)
	searchContainer.AddItem(help, 1, 0, false)

	// Show the persistent container (like Slack panel does)
	if mainFlex, ok := a.views["mainFlex"].(*tview.Flex); ok {
		mainFlex.ResizeItem(searchContainer, 5, 0) // Show with fixed height
	}

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
						a.showStatusMessage("🔕 Suggestion cancelled when opening search")
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
				a.hideSearchContainer()
			}
			if curMode == "remote" {
				go a.performSearch(query)
			} else {
				// Before applying local filter, capture base snapshot only if not already in search mode
				// This preserves the original inbox during search refinement
				if a.searchMode != "local" {
					if a.logger != nil {
						a.logger.Printf("🔍 LOCAL SEARCH: Capturing base snapshot (searchMode=%q)", a.searchMode)
					}
					a.captureLocalBaseSnapshot()
				} else {
					if a.logger != nil {
						a.logger.Printf("🔍 LOCAL SEARCH: Preserving existing base snapshot during refinement (searchMode=%q)", a.searchMode)
					}
				}
				a.localFilter = query
				go a.applyLocalFilter(query)
			}
			// Keep searchInput in views map for Tab navigation - do NOT delete it
			// delete(a.views, "searchInput") // REMOVED: Allow Tab cycling back to search
			// Focus management will be handled in applyLocalFilter's QueueUpdateDraw callback
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
				a.hideSearchContainer()
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
				searchContainer.SetTitle("🔎 Local Filter")
				help.SetText("Type space-separated terms; all must match | Enter=apply, Ctrl-T=switch, ESC to back")
				input.SetPlaceholder("Type words to match (space-separated)")
				if sp, ok := a.views["searchPanel"].(*tview.Flex); ok {
					sp.SetTitle("🔎 Local Filter")
				}
			} else {
				curMode = "remote"
				searchContainer.SetTitle("🔍 Gmail Search")
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
			a.hideSearchContainer()
			return nil
		}
		return ev
	})

	// Old searchPanel path removed - using persistent container approach only

	// Focus the input field
	a.currentFocus = "search"
	a.updateFocusIndicators("search")
	a.SetFocus(input)
}

// hideSearchContainer hides the persistent search container (like Slack panel)
func (a *App) hideSearchContainer() {
	// For local search, let exitSearch handle everything (container hiding + data restoration)
	if a.searchMode == "local" {
		go a.exitSearch()
		return
	}

	// For remote search, also call exitSearch to handle data restoration
	if a.searchMode == "remote" {
		go a.exitSearch()
		return
	}

	// For non-search modes, just hide the container
	if mainFlex, ok := a.views["mainFlex"].(*tview.Flex); ok {
		if searchContainer, ok := a.views["searchContainer"]; ok {
			mainFlex.ResizeItem(searchContainer, 0, 0) // Hide container
		}
	}
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
	a.SetFocus(a.views["list"])
	delete(a.views, "searchInput")
}

// openAdvancedSearchForm shows a guided form to compose a Gmail query, splitting the list area
func (a *App) openAdvancedSearchForm() {
	// Hide simple search container if it's visible (mutual exclusion)
	a.hideSearchContainer()

	// Build form fields similar to Gmail advanced search (with placeholders)
	form := tview.NewForm()

	// Apply hierarchical theme system for consistent styling with search overlay
	formColors := a.GetComponentColors("search")
	form.SetBackgroundColor(formColors.Background.Color()).SetBorder(false)

	// Apply component colors to form elements for consistency
	form.SetButtonBackgroundColor(formColors.Background.Color())
	form.SetButtonTextColor(formColors.Text.Color())
	form.SetLabelColor(formColors.Title.Color())
	form.SetFieldBackgroundColor(formColors.Background.Color())
	form.SetFieldTextColor(formColors.Text.Color())
	fromField := tview.NewInputField().
		SetLabel("👤 From").
		SetPlaceholder("user@example.com").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(fromField, "advanced")
	// Expose for focus restoration while background loads complete
	a.views["advFrom"] = fromField
	toField := tview.NewInputField().
		SetLabel("📩 To").
		SetPlaceholder("person@example.com").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(toField, "advanced")

	subjectField := tview.NewInputField().
		SetLabel("🧾 Subject").
		SetPlaceholder("exact words or phrase").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(subjectField, "advanced")

	hasField := tview.NewInputField().
		SetLabel("🔎 Has the words").
		SetPlaceholder("words here").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(hasField, "advanced")

	notField := tview.NewInputField().
		SetLabel("🚫 Doesn't have").
		SetPlaceholder("exclude words").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(notField, "advanced")
	form.AddFormItem(fromField)
	form.AddFormItem(toField)
	form.AddFormItem(subjectField)
	form.AddFormItem(hasField)
	form.AddFormItem(notField)

	// Remove duplicate theme applications - already applied above

	// Size single expression, e.g. "<2MB" or ">500KB"
	sizeExprField := tview.NewInputField().
		SetLabel("📦 Size").
		SetPlaceholder("e.g., <2MB or >500KB").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(sizeExprField, "advanced")
	form.AddFormItem(sizeExprField)

	// Date within single token, e.g. "2d", "3w", "1m", "4h", "6y"
	dateWithinField := tview.NewInputField().
		SetLabel("📅 Date within").
		SetPlaceholder("e.g., 2d, 3w, 1m, 4h, 6y").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(dateWithinField, "advanced")
	form.AddFormItem(dateWithinField)
	// Scope
	baseScopes := []string{"All Mail", "Inbox", "Archive", "Sent", "Drafts", "Spam", "Trash", "Starred", "Important"}
	scopes := append([]string{}, baseScopes...)
	scopeVal := "All Mail"
	if a.logger != nil {
		a.logger.Println("advsearch: building form")
	}
	scopeField := tview.NewInputField().
		SetLabel("📂 Search").
		SetText(scopeVal).
		SetPlaceholder("Press Enter to pick scope/label").
		SetFieldWidth(50)
	a.ConfigureInputFieldTheme(scopeField, "advanced")
	form.AddFormItem(scopeField)

	// Remove duplicate theme applications - already applied above
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
			// FIX: Use fixed width to prevent viewport scrolling mode
			twoCol.ResizeItem(right, 35, 0) // Width: 35 columns to prevent wrapping
			// form takes remaining space
			if form != nil {
				twoCol.ResizeItem(form, 0, 1)
			}
		}
		rightVisible = true
		right.Clear()
		type optionItem struct {
			display string
			action  func()
		}
		options := make([]optionItem, 0, 256)
		// Folders (safe emoji alternatives)
		options = append(options,
			optionItem{"📧 All Mail", func() { scopeVal = "All Mail"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{"📥 Inbox", func() { scopeVal = "Inbox"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{"📦 Archive", func() { scopeVal = "Archive"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{"📤 Sent Mail", func() { scopeVal = "Sent"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{"📝 Drafts", func() { scopeVal = "Drafts"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{"🚫 Spam", func() { scopeVal = "Spam"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
			optionItem{"🗑 Trash", func() { scopeVal = "Trash"; scopeField.SetText(scopeVal); hideRight(); a.SetFocus(scopeField) }},
		)
		// Anywhere
		options = append(options, optionItem{"🔖 Mail & Spam & Trash", func() {
			scopeVal = "Mail & Spam & Trash"
			scopeField.SetText(scopeVal)
			hideRight()
			a.SetFocus(scopeField)
		}})
		// State
		options = append(options,
			optionItem{"✅ Read Mail", func() { scopeField.SetText("is:read"); hideRight(); a.SetFocus(scopeField) }},
			optionItem{"📬 Unread Mail", func() { scopeField.SetText("is:unread"); hideRight(); a.SetFocus(scopeField) }},
		)
		// Categories (testing safe emoji alternatives)
		categoryEmojis := map[string]string{
			"social":     "👥", // People emoji
			"updates":    "🔄", // Refresh/update emoji
			"forums":     "💬", // Speech balloon
			"promotions": "💰", // Money bag - represents promotions/deals
		}
		for _, c := range []string{"social", "updates", "forums", "promotions"} {
			cc := c
			disp := strings.ToUpper(cc[:1]) + cc[1:] // OBLITERATED: deprecated strings.Title eliminated! 💥
			emoji := categoryEmojis[cc]
			options = append(options, optionItem{emoji + " " + disp, func() { scopeField.SetText("category:" + cc); hideRight(); a.SetFocus(scopeField) }})
		}
		// Labels (all user labels in 'scopes' beyond base)
		baseSet := map[string]struct{}{"All Mail": {}, "Inbox": {}, "Archive": {}, "Sent": {}, "Drafts": {}, "Spam": {}, "Trash": {}, "Starred": {}, "Important": {}}
		for _, s := range scopes {
			if _, okb := baseSet[s]; okb {
				continue
			}
			name := s
			options = append(options, optionItem{"🔖 " + name, func() { scopeField.SetText("label:\"" + name + "\""); hideRight(); a.SetFocus(scopeField) }})
		}

		filter := tview.NewInputField().SetLabel("🔎 ")
		filter.SetPlaceholder("filter options…")
		filter.SetFieldWidth(31) // Adjusted width to match 35-column container
		// Apply consistent theme styling to filter field
		a.ConfigureInputFieldTheme(filter, "advanced")

		list := tview.NewList().ShowSecondaryText(false)
		list.SetBorder(false)
		// Apply hierarchical theme styling to list
		searchOptionsColors := a.GetComponentColors("search")
		list.SetBackgroundColor(searchOptionsColors.Background.Color())
		list.SetMainTextColor(searchOptionsColors.Text.Color())
		list.SetSelectedBackgroundColor(searchOptionsColors.Accent.Color())
		list.SetSelectedTextColor(searchOptionsColors.Background.Color())

		// FORCE REDRAW: Add selection change callback to force complete redraw
		list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			// Force the entire List widget to redraw when selection changes
			list.SetBorder(false) // Trigger internal redraw
		})

		// Container con borde para incluir picker + lista
		box := tview.NewFlex().SetDirection(tview.FlexRow)
		box.SetBorder(true).SetTitle(" 📂 Search options ")
		box.SetBorderColor(searchOptionsColors.Border.Color())
		box.SetTitleColor(searchOptionsColors.Title.Color())
		box.SetBackgroundColor(searchOptionsColors.Background.Color())
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
		right.AddItem(box, 0, 1, true) // Full height: occupy whole search panel
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
				a.showStatusMessage("⏰ Date must be like 3d, 1w, 2m or 1y")
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
		case "Archive":
			parts = append(parts, "in:archive")
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
		// Do NOT submit on Enter; only when pressing the "🔎 Search" button.
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
		// Apply hierarchical theme system for advanced search panel
		advancedSearchColors := a.GetComponentColors("search")
		sp.Clear()
		sp.SetBorder(true).
			SetBorderColor(advancedSearchColors.Border.Color()).
			SetTitle("🔎 Advanced Search").
			SetTitleColor(advancedSearchColors.Title.Color()).
			SetTitleAlign(tview.AlignCenter)
		sp.SetBackgroundColor(advancedSearchColors.Background.Color())
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
	advancedSearchColors := a.GetComponentColors("search")

	// Apply ForceFilledBorderFlex directly to bordered container (like textContainer)
	modal := tview.NewFlex().SetDirection(tview.FlexRow)
	generalColors := a.GetComponentColors("general")
	modal.SetBackgroundColor(generalColors.Background.Color()).
		SetBorder(true).
		SetBorderColor(generalColors.Border.Color()).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle("🔎 Advanced Search").
		SetTitleColor(advancedSearchColors.Title.Color()).
		SetTitleAlign(tview.AlignCenter)

	// Apply ForceFilledBorderFlex for consistent border rendering
	ForceFilledBorderFlex(modal)

	// Re-apply title styling after ForceFilledBorderFlex
	modal.SetTitleColor(advancedSearchColors.Title.Color())
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
		a.SetMessageIDs(filteredIDs)
		a.messagesMeta = filteredMeta
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.Clear()
			for i, text := range rows {
				// Build prefix for filtered results
				var prefix string
				if a.showMessageNumbers {
					maxNumber := len(filteredIDs)
					width := len(fmt.Sprintf("%d", maxNumber))
					prefix = fmt.Sprintf("%*d ", width, i+1) // Right-aligned numbering for filtered results
				}

				// Add read/unread indicator
				if i < len(filteredMeta) && filteredMeta[i] != nil {
					unread := false
					for _, labelId := range filteredMeta[i].LabelIds {
						if labelId == "UNREAD" {
							unread = true
							break
						}
					}
					if unread {
						prefix += "● "
					} else {
						prefix += "○ "
					}
				}

				table.SetCell(i, 0, tview.NewTableCell(prefix+text).SetExpansion(1))
			}
			table.SetTitle(fmt.Sprintf(" 🔎 Filter (%d) — %s ", len(rows), expr))
			if table.GetRowCount() > 1 {
				// Only auto-select if composition panel is not active
				if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
					table.Select(1, 0) // Select first message (row 1, since row 0 is header)
					// Set current message ID to the first filtered message
					if len(filteredIDs) > 0 {
						firstID := filteredIDs[0]
						a.SetCurrentMessageID(firstID)
						go a.showMessageWithoutFocus(firstID)
					}
				}
			}
		}
		a.refreshTableDisplay()

		// Set focus to list and update focus indicators after table is fully rebuilt
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])
	})
}

// showMessage displays a message in the text view
func (a *App) showMessage(id string) {
	// Restore text container title when viewing messages (but not when in help mode)
	if !a.showHelp {
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			textContainer.SetTitle(" 📄 Message Content ")
			textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
		}
	}

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
		a.enhancedTextView.SetContent("Loading message…")
		text.ScrollToBeginning()
	}

	// Automatically switch focus to text view when viewing a message
	a.SetFocus(a.views["text"])
	a.currentFocus = "text"
	a.updateFocusIndicators("text")
	a.SetCurrentMessageID(id)

	a.Draw()

	// Load message content in background
	go func() {
		if a.debug {
			a.logger.Printf("showMessage background: id=%s", id)
		}
		// Guard: capture requested ID to prevent stale updates if selection changes
		requestedID := id
		// Use cache if available; otherwise fetch and cache
		var message *gmail.Message
		if cached, ok := a.GetMessageFromCache(id); ok {
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
			a.SetMessageInCache(id, m)
			message = m
		}

		rendered, isANSI := a.renderMessageContent(message)
		// Detect calendar invite parts (best-effort)
		if inv, ok := a.detectCalendarInvite(message.Message); ok {
			a.inviteCache[id] = inv
		}

		// Update UI in main thread
		// Clear persistent status after content loads (outside QueueUpdateDraw to avoid deadlock)
		go func() {
			a.GetErrorHandler().ClearPersistentMessage()
		}()

		a.QueueUpdateDraw(func() {
			// Abort if selection changed while loading
			if a.GetCurrentMessageID() != requestedID {
				if a.debug {
					a.logger.Printf("showMessage: abort UI update, currentID changed (requested=%s, current=%s)", requestedID, a.GetCurrentMessageID())
				}
				return
			}
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				if isANSI {
					// Convert ANSI → tview markup while writing
					_, _ = fmt.Fprint(tview.ANSIWriter(text, "", ""), rendered)
				} else {
					a.enhancedTextView.SetContent(rendered)
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
		base := config.DefaultSavedDir()
		if err := os.MkdirAll(base, 0o750); err != nil {
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
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
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
		base := config.DefaultSavedDir()
		if err := os.MkdirAll(base, 0o750); err != nil {
			a.QueueUpdateDraw(func() { a.showError("❌ Could not create saved folder") })
			return
		}
		name := sanitizeFilename(subj)
		ts := time.Now().Format("20060102-150405")
		file := filepath.Join(base, ts+"-"+name+".eml")
		if err := os.WriteFile(file, data, 0o600); err != nil {
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
	// Restore text container title when viewing messages (but not when in help mode)
	if !a.showHelp {
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			textContainer.SetTitle(" 📄 Message Content ")
			textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
		}
	}

	// Show loading message
	if text, ok := a.views["text"].(*tview.TextView); ok {
		if a.debug {
			a.logger.Printf("showMessageWithoutFocus: id=%s", id)
		}
		a.enhancedTextView.SetContent("Loading message...")
		text.ScrollToBeginning()
	}
	// Do NOT set currentMessageID here; selection changes (and Enter) manage it.
	// Setting it here could race with selection updates after deletions and cause stale content.

	go func() {
		if a.debug {
			a.logger.Printf("showMessageWithoutFocus background: id=%s", id)
		}
		// Guard: capture requested ID to prevent stale updates if selection changes
		requestedID := id
		// Use cache if available; otherwise fetch and cache
		var message *gmail.Message

		// Phase 2.4: Check preloader cache first for adjacent message preloading
		if preloader := a.GetPreloaderService(); preloader != nil && preloader.IsEnabled() {
			if cachedMessage, found := preloader.GetCachedMessage(a.ctx, id); found {
				if a.debug {
					a.logger.Printf("showMessageWithoutFocus: PRELOADER CACHE HIT id=%s", id)
				}
				// Check if cached message has body content (preloader uses metadata only)
				hasContent := cachedMessage.Payload != nil &&
					len(cachedMessage.Payload.Parts) > 0 ||
					(cachedMessage.Payload.Body != nil && cachedMessage.Payload.Body.Data != "")

				if hasContent {
					// Convert gmail_v1.Message to gmail.Message without additional API calls
					message = a.Client.CreateMessageFromRaw(cachedMessage)
				} else {
					if a.debug {
						a.logger.Printf("showMessageWithoutFocus: Preloader cache has metadata only, need full content")
					}
					// Preloader cache only has metadata, fetch full content
					fullMessage, err := a.Client.GetMessageWithContent(id)
					if err == nil {
						message = fullMessage
						// Store in regular cache for future use
						a.SetMessageInCache(id, fullMessage)
					}
				}
			}
		}

		// Fallback to regular message cache
		if message == nil {
			if cached, ok := a.GetMessageFromCache(id); ok {
				if a.debug {
					a.logger.Printf("showMessageWithoutFocus: regular cache hit id=%s", id)
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
				a.SetMessageInCache(id, m)
				message = m
			}
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
			// Abort if selection changed while loading
			if a.GetCurrentMessageID() != requestedID {
				if a.debug {
					a.logger.Printf("showMessageWithoutFocus: abort UI update, currentID changed (requested=%s, current=%s)", requestedID, a.GetCurrentMessageID())
				}
				return
			}
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				if isANSI {
					_, _ = fmt.Fprint(tview.ANSIWriter(text, "", ""), rendered)
				} else {
					a.enhancedTextView.SetContent(rendered)
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
		if cached, ok := a.GetMessageFromCache(id); ok {
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
			a.SetMessageInCache(id, fetched)
			m = fetched
		}
		rendered, isANSI := a.renderMessageContent(m)
		a.QueueUpdateDraw(func() {
			// Don't update content when in help mode to preserve help content
			if !a.showHelp {
				if text, ok := a.views["text"].(*tview.TextView); ok {
					text.SetDynamicColors(true)
					text.Clear()
					if isANSI {
						_, _ = fmt.Fprint(tview.ANSIWriter(text, "", ""), rendered)
					} else {
						a.enhancedTextView.SetContent(rendered)
					}
					text.ScrollToBeginning()
				}
			}
		})
	}()
}

// Removed unused function: refreshMessageContentWithOverride

// getCurrentMessageID gets the ID of the currently selected message
func (a *App) getCurrentMessageID() string {
	// Safety check: ensure views map exists and is not nil
	if a.views == nil {
		return ""
	}

	// Safety check: ensure list view exists
	table, ok := a.views["list"]
	if !ok || table == nil {
		return ""
	}

	// Type assertion with safety check
	tableView, ok := table.(*tview.Table)
	if !ok {
		return ""
	}

	// Get selection safely
	row, _ := tableView.GetSelection()

	// Account for header row (row 0 is header, messages start at row 1)
	messageIndex := row - 1

	// Safety check: ensure ids slice exists and is not nil, and messageIndex is valid
	if a.ids == nil || messageIndex < 0 || messageIndex >= len(a.ids) {
		return ""
	}

	return a.ids[messageIndex]
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
		// Account for header row (row 0 is header, messages start at row 1)
		messageIndex := row - 1
		if messageIndex >= 0 && messageIndex < len(a.messagesMeta) {
			meta = a.messagesMeta[messageIndex]
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
		// Account for header row (row 0 is header, messages start at row 1)
		messageIndex := row - 1
		if messageIndex >= 0 && messageIndex < len(a.messagesMeta) {
			meta = a.messagesMeta[messageIndex]
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
		// Account for header row (row 0 is header, messages start at row 1)
		messageIndex := row - 1
		if messageIndex >= 0 && messageIndex < len(a.messagesMeta) {
			meta = a.messagesMeta[messageIndex]
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
		// Account for header row (row 0 is header, messages start at row 1)
		messageIndex := row - 1
		if messageIndex >= 0 && messageIndex < len(a.messagesMeta) {
			meta = a.messagesMeta[messageIndex]
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

// getHeaderWidth returns the available width for header display
func (a *App) getHeaderWidth() int {
	if header, ok := a.views["header"].(*tview.TextView); ok {
		_, _, w, _ := header.GetInnerRect()
		if w > 0 {
			return w
		}
	}
	// Fallback to list width or screen width
	return a.getListWidth()
}

// adjustHeaderHeight dynamically adjusts the header container height based on content
func (a *App) adjustHeaderHeight(headerContent string) {
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		if header, ok := a.views["header"].(*tview.TextView); ok {
			// If header content is empty (headers hidden), set height to 0
			if strings.TrimSpace(headerContent) == "" {
				textContainer.ResizeItem(header, 0, 0)
				return
			}

			// Count the number of lines in the header content
			lines := strings.Count(headerContent, "\n") + 1

			// Set minimum height of 4 and maximum of 12 to prevent extreme cases
			minHeight := 4
			maxHeight := 12

			if lines < minHeight {
				lines = minHeight
			}
			if lines > maxHeight {
				lines = maxHeight
			}

			// Resize the header item in the text container
			textContainer.ResizeItem(header, lines, 0)
		}
	}
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
				s := string(raw)
				if strings.Contains(strings.ToUpper(s), "METHOD:REQUEST") {
					methodReq = true
				}

				// Debug: log the raw iCalendar data

				out.UID = scanICSField(s, "UID")
				out.Summary = scanICSField(s, "SUMMARY")
				out.Organizer = scanICSField(s, "ORGANIZER")
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
	lines := strings.Split(s, "\n")

	// Find the VEVENT section first
	inVEvent := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Track when we're inside a VEVENT section
		if line == "BEGIN:VEVENT" {
			inVEvent = true
			continue
		}
		if line == "END:VEVENT" {
			inVEvent = false
			continue
		}

		// Only look for fields inside the VEVENT section
		if !inVEvent {
			continue
		}

		// Check if line starts with our key
		if strings.HasPrefix(line, key) {
			// Handle parameters like DTSTART;TZID=Europe/Madrid:20250818T163000
			colonIdx := strings.Index(line, ":")
			if colonIdx > 0 {
				// Extract everything after the colon, including timezone parameter
				fullValue := strings.TrimSpace(line[colonIdx+1:])

				// Handle multiline values (continuation lines start with space/tab)
				for j := i + 1; j < len(lines); j++ {
					nextLine := lines[j]
					if len(nextLine) > 0 && (nextLine[0] == ' ' || nextLine[0] == '\t') {
						// It's a continuation line
						fullValue += strings.TrimSpace(nextLine)
					} else {
						break // Not a continuation
					}
				}

				// For DTSTART/DTEND, we want to preserve the timezone parameter
				// Return the full line including parameters: ";TZID=Europe/Madrid:20250818T163000"
				if key == "DTSTART" || key == "DTEND" {
					// Extract the parameter part too
					paramStart := strings.Index(line, ";")
					if paramStart > 0 && paramStart < colonIdx {
						params := line[paramStart:colonIdx]
						return params + ":" + fullValue
					}
				}

				return fullValue
			}
		}
	}

	return ""
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
		// Fallback 1: Re-detect from message cache
		if m, ok2 := a.messageCache[mid]; ok2 && m != nil {
			if parsed, ok3 := a.detectCalendarInvite(m.Message); ok3 {
				inv = parsed
				a.inviteCache[mid] = inv
				ok = true
				if a.logger != nil {
					a.logger.Printf("RSVP: Re-detected calendar invite from message cache")
				}
			}
		}

		// Fallback 2: Search any invite in cache (handles cache key mismatches)
		if !ok && len(a.inviteCache) > 0 {
			for _, cachedInv := range a.inviteCache {
				if cachedInv.UID != "" {
					inv = cachedInv
					// Removed ineffectual assignment: ok = true (not used after break)
					if a.logger != nil {
						a.logger.Printf("RSVP: Using cached invite from alternate cache entry")
					}
					break
				}
			}
		}
	}
	if inv.UID == "" {
		a.showError("❌ No calendar invite found in this message")
		return
	}

	// Create meeting details sections - separate TextView for each line to control colors
	meetingContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	meetingContainer.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())

	// Meeting title
	titleView := tview.NewTextView().SetWordWrap(true)
	if inv.Summary != "" {
		titleView.SetText(fmt.Sprintf("📅 %s", inv.Summary))
	} else {
		titleView.SetText("📅 Meeting Invitation")
	}
	titleView.SetTextColor(a.GetComponentColors("rsvp").Title.Color())
	titleView.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())
	meetingContainer.AddItem(titleView, 1, 0, false)

	// Organizer
	if inv.Organizer != "" {
		organizerName := formatOrganizerName(inv.Organizer)
		if a.logger != nil {
			a.logger.Printf("RSVP Debug: Raw organizer='%s', formatted='%s'", inv.Organizer, organizerName)
		}
		if organizerName != "" {
			organizerView := tview.NewTextView().SetWordWrap(true)
			organizerView.SetText(fmt.Sprintf("👤 %s", organizerName))
			organizerView.SetTextColor(a.GetComponentColors("rsvp").Text.Color())
			organizerView.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())
			meetingContainer.AddItem(organizerView, 1, 0, false)
		}
	} else if a.logger != nil {
		a.logger.Printf("RSVP Debug: No organizer field found in invite")
	}

	// Date and time
	if inv.DtStart != "" {
		// Debug logging to see what we actually get
		timeRange := formatMeetingTimeRange(inv.DtStart, inv.DtEnd)
		if timeRange != "" {
			timeView := tview.NewTextView().SetWordWrap(true)
			timeView.SetText(fmt.Sprintf("🕐 %s", timeRange))
			timeView.SetTextColor(a.GetComponentColors("rsvp").Text.Color())
			timeView.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())
			meetingContainer.AddItem(timeView, 1, 0, false)
		}
	}

	// Build RSVP options list
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	list.SetMainTextColor(a.GetComponentColors("rsvp").Text.Color())
	list.SetSelectedTextColor(a.GetComponentColors("rsvp").Accent.Color())
	list.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())
	list.AddItem("✅ Accept", "I'll be there", 0, nil)
	list.AddItem("🤔 Tentative", "Maybe attending", 0, nil)
	list.AddItem("❌ Decline", "Cannot attend", 0, nil)
	if list.GetItemCount() > 0 {
		list.SetCurrentItem(0)
	}

	// Footer with instructions
	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter to respond | Esc to close ")
	footer.SetTextColor(a.GetComponentColors("rsvp").Text.Color())
	footer.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())

	// Create container with meeting info at top (standard picker pattern - no fixed width)
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true).SetTitle(" 📅 RSVP ").SetTitleColor(a.GetComponentColors("rsvp").Title.Color())
	container.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())

	// Add meeting info section (fixed height)
	container.AddItem(meetingContainer, 3, 0, false)

	// Add blank line between meeting details and RSVP options
	spacer := tview.NewTextView()
	spacer.SetBackgroundColor(a.GetComponentColors("rsvp").Background.Color())
	container.AddItem(spacer, 1, 0, false)

	// Add RSVP options list (flexible height)
	container.AddItem(list, 0, 1, true)

	// Add footer
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
		go a.sendRSVP(choice, "")
		// Close panel
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0) // Hide RSVP panel
		}
		a.setActivePicker(PickerNone)
		a.restoreFocusAfterModal()
	}
	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:
			sendSelected()
			return nil
		case tcell.KeyEscape:
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.labelsView, 0, 0) // Hide RSVP panel
			}
			a.setActivePicker(PickerNone)
			a.restoreFocusAfterModal()
			return nil
		}
		return e
	})

	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			// Keep main content visible and show RSVP panel on the right (like other pickers)
			split.RemoveItem(a.labelsView)
			a.labelsView = container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1) // Show RSVP panel
		}
		a.setActivePicker(PickerRSVP)
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

	// Use fallback strategies if invite not found
	if !ok || inv.UID == "" {
		// Fallback 1: Re-detect from message cache
		if m, ok2 := a.messageCache[mid]; ok2 && m != nil {
			if parsed, ok3 := a.detectCalendarInvite(m.Message); ok3 {
				inv = parsed
				a.inviteCache[mid] = inv
				ok = true
			}
		}

		// Fallback 2: Search any invite in cache (handles cache key mismatches)
		if (!ok || inv.UID == "") && len(a.inviteCache) > 0 {
			for _, cachedInv := range a.inviteCache {
				if cachedInv.UID != "" {
					inv = cachedInv
					ok = true
					if a.logger != nil {
						a.logger.Printf("RSVP: Using cached invite from alternate cache entry for sending response")
					}
					break
				}
			}
		}

		// Final check after all fallbacks
		if !ok || inv.UID == "" {
			a.showError("❌ No invite to reply to")
			return
		}
	}
	if a.Calendar == nil || a.Calendar.Service == nil {
		a.showError("❌ Calendar API not available. Please re-authorize with Calendar permissions.")
		return
	}
	// Resolve our account email
	attendeeEmail, _ := a.Client.ActiveAccountEmail(a.ctx)
	if strings.TrimSpace(attendeeEmail) == "" {
		a.showError("❌ Could not determine account email")
		return
	}
	a.setStatusPersistent("📤 Updating status in Calendar…")
	go func(uid string, status string) {
		attempt := func() (*cal.Event, error) {
			evt, err := a.Calendar.FindByICalUID(a.ctx, uid)
			if err != nil || evt == nil {
				return nil, err
			}
			if err := a.Calendar.RespondToInvite(a.ctx, evt.Id, attendeeEmail, status, true); err != nil {
				return nil, err
			}
			return evt, nil
		}
		if evt, err := attempt(); err != nil {
			if a.isInsufficientPermissions(err) {
				a.QueueUpdateDraw(func() { a.setStatusPersistent("🔐 Re-authorizing Calendar…") })
				if err2 := a.reauthorizeCalendar(); err2 == nil {
					if _, err3 := attempt(); err3 != nil {
						a.QueueUpdateDraw(func() { a.showError(fmt.Sprintf("❌ Error after re-authorize: %v", err3)) })
						return
					}
					a.QueueUpdateDraw(func() { a.showStatusMessage("✅ RSVP actualizado en Calendar") })
					return
				}
				a.QueueUpdateDraw(func() { a.showError("❌ Could not re-authorize Calendar. Check credentials and token.") })
				return
			}
			// If not found by iCalUID, and we extracted RSVP links, give a friendly hint
			if strings.Contains(strings.ToLower(err.Error()), "event not found") {
				a.QueueUpdateDraw(func() {
					a.showError("❌ Event not found in Calendar for this invite. It may not be added to your calendar yet.")
				})
				return
			}
			a.QueueUpdateDraw(func() { a.showError(fmt.Sprintf("❌ Error updating RSVP: %v", err)) })
			return
		} else {
			// Success
			_ = evt
			a.QueueUpdateDraw(func() { a.showStatusMessage("✅ RSVP updated in Calendar") })
		}
	}(inv.UID, partstat)
}

// isInsufficientPermissions checks common markers for missing Calendar scope
func (a *App) isInsufficientPermissions(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "insufficientpermissions") || strings.Contains(s, "403")
}

// reauthorizeCalendar runs OAuth flow with Calendar scopes and swaps the client
func (a *App) reauthorizeCalendar() error {
	cred, tok := config.DefaultCredentialPaths()
	if strings.TrimSpace(a.Config.Credentials) != "" {
		cred = a.Config.Credentials
	}
	if strings.TrimSpace(a.Config.Token) != "" {
		tok = a.Config.Token
	}
	svc, err := auth.NewCalendarService(a.ctx, cred, tok, "https://www.googleapis.com/auth/calendar.events")
	if err != nil {
		return err
	}
	a.Calendar = calclient.NewClient(svc)
	return nil
}

// formatICalDateTime parses iCalendar datetime format and returns human-readable string
func formatICalDateTime(dtStr string) string {
	if dtStr == "" {
		return ""
	}

	// Common iCalendar datetime formats:
	// DTSTART:20250115T100000Z (UTC)
	// DTSTART;VALUE=DATE:20250115 (date only)
	// DTSTART;TZID=America/New_York:20250115T100000
	// ;TZID=Europe/Madrid:20250818T170000 (from scanICSField)

	// Clean up the datetime string - handle timezone parameters properly
	cleanDt := dtStr

	// Handle format like ";TZID=Europe/Madrid:20250818T170000"
	if strings.HasPrefix(cleanDt, ";") && strings.Contains(cleanDt, ":") {
		// Extract the datetime part after the last colon
		if colonIdx := strings.LastIndex(cleanDt, ":"); colonIdx >= 0 {
			cleanDt = cleanDt[colonIdx+1:]
		}
	} else if strings.Contains(cleanDt, ";") && strings.Contains(cleanDt, ":") {
		// Handle format like "DTSTART;TZID=America/New_York:20250115T100000"
		if colonIdx := strings.LastIndex(cleanDt, ":"); colonIdx >= 0 {
			cleanDt = cleanDt[colonIdx+1:]
		}
	}

	// Validate that we have a reasonable date format
	if len(cleanDt) < 8 {
		return dtStr // Return original if too short
	}

	// Try different parsing formats
	formats := []string{
		"20060102T150405Z",     // UTC format: 20250115T100000Z
		"20060102T150405",      // Local format: 20250115T100000
		"20060102T1504Z",       // UTC without seconds: 20250115T1030Z
		"20060102T1504",        // Local without seconds: 20250115T1030
		"20060102",             // Date only: 20250115
		"2006-01-02T15:04:05Z", // RFC format with dashes
		"2006-01-02T15:04:05",  // RFC format without Z
		"2006-01-02T15:04Z",    // RFC format short
		"2006-01-02T15:04",     // RFC format short without Z
		"2006-01-02",           // Date with dashes
	}

	for _, format := range formats {
		if t, err := time.Parse(format, cleanDt); err == nil {
			// Sanity check: reject dates before 1990 or after 2050 (likely parsing errors)
			if t.Year() < 1990 || t.Year() > 2050 {
				continue
			}

			// For date-only format, don't show time
			if format == "20060102" {
				return t.Format("Mon, Jan 2 2006")
			}
			// For datetime formats, show both date and time
			return t.Format("Mon, Jan 2 2006, 3:04 PM")
		}
	}

	// If parsing fails, return a cleaned version of original
	return strings.TrimSpace(strings.ReplaceAll(dtStr, "DTSTART:", ""))
}

// formatMeetingTimeRange creates a time range string from start and end times
func formatMeetingTimeRange(dtStart, dtEnd string) string {
	startStr := formatICalDateTime(dtStart)
	endStr := formatICalDateTime(dtEnd)

	// Debug logging removed - was leaking to main content

	// If start parsing failed, try to show something meaningful
	// Check if startStr is empty or still contains unparsed format (like "DTSTART:" prefix)
	if startStr == "" || len(startStr) < 10 || strings.HasPrefix(startStr, "DTSTART") {
		// Return a generic message if we can't parse the dates
		return "Meeting time details not available"
	}

	// If no end time, just show start
	if endStr == "" || len(endStr) < 10 || strings.HasPrefix(endStr, "DTEND") {
		return startStr
	}

	// Try to parse both to see if they're on the same date
	startTime := parseICalDateTime(dtStart)
	endTime := parseICalDateTime(dtEnd)

	if !startTime.IsZero() && !endTime.IsZero() {
		// Same date - show "Mon, Jan 2 2006, 10:00 AM - 11:00 AM"
		if startTime.Format("20060102") == endTime.Format("20060102") {
			return startTime.Format("Mon, Jan 2 2006, 3:04 PM") + " - " + endTime.Format("3:04 PM")
		}
		// Different dates - show full date/time for both
		return startStr + " - " + endStr
	}

	// Fallback - just show start if we have it
	return startStr
}

// parseICalDateTime helper function to parse iCalendar datetime for comparison
func parseICalDateTime(dtStr string) time.Time {
	if dtStr == "" {
		return time.Time{}
	}

	// Clean up the datetime string - handle timezone parameters properly
	cleanDt := dtStr

	// Handle format like ";TZID=Europe/Madrid:20250818T170000"
	if strings.HasPrefix(cleanDt, ";") && strings.Contains(cleanDt, ":") {
		// Extract the datetime part after the last colon
		if colonIdx := strings.LastIndex(cleanDt, ":"); colonIdx >= 0 {
			cleanDt = cleanDt[colonIdx+1:]
		}
	} else if strings.Contains(cleanDt, ";") && strings.Contains(cleanDt, ":") {
		// Handle format like "DTSTART;TZID=America/New_York:20250115T100000"
		if colonIdx := strings.LastIndex(cleanDt, ":"); colonIdx >= 0 {
			cleanDt = cleanDt[colonIdx+1:]
		}
	}

	// Validate minimum length
	if len(cleanDt) < 8 {
		return time.Time{}
	}

	formats := []string{
		"20060102T150405Z",     // UTC format: 20250115T100000Z
		"20060102T150405",      // Local format: 20250115T100000
		"20060102T1504Z",       // UTC without seconds: 20250115T1030Z
		"20060102T1504",        // Local without seconds: 20250115T1030
		"20060102",             // Date only: 20250115
		"2006-01-02T15:04:05Z", // RFC format with dashes
		"2006-01-02T15:04:05",  // RFC format without Z
		"2006-01-02T15:04Z",    // RFC format short
		"2006-01-02T15:04",     // RFC format short without Z
		"2006-01-02",           // Date with dashes
	}

	for _, format := range formats {
		if t, err := time.Parse(format, cleanDt); err == nil {
			// Sanity check: reject dates before 1990 or after 2050
			if t.Year() < 1990 || t.Year() > 2050 {
				continue
			}
			return t
		}
	}

	return time.Time{}
}

// formatOrganizerName extracts and formats organizer name from iCalendar organizer field
func formatOrganizerName(organizer string) string {
	if organizer == "" {
		return ""
	}

	// iCalendar organizer format: "CN=John Doe:MAILTO:john@example.com" or just "MAILTO:john@example.com"

	// Try to extract Common Name (CN)
	if cnIdx := strings.Index(organizer, "CN="); cnIdx >= 0 {
		cnStart := cnIdx + 3
		cnEnd := strings.Index(organizer[cnStart:], ":")
		if cnEnd >= 0 {
			return strings.TrimSpace(organizer[cnStart : cnStart+cnEnd])
		}
	}

	// Try to extract email from MAILTO
	if mailtoIdx := strings.Index(strings.ToUpper(organizer), "MAILTO:"); mailtoIdx >= 0 {
		emailStart := mailtoIdx + 7
		email := organizer[emailStart:]
		// Clean up any trailing parameters
		if spaceIdx := strings.IndexAny(email, " \t\n\r;"); spaceIdx >= 0 {
			email = email[:spaceIdx]
		}
		return strings.TrimSpace(email)
	}

	// If it looks like just an email, return it
	if strings.Contains(organizer, "@") && !strings.Contains(organizer, " ") {
		return organizer
	}

	// Return cleaned up original
	return strings.TrimSpace(organizer)
}

// Removed unused functions: extractEmailFromOrganizer, sanitizeOrganizer

/* moved to messages_actions.go
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
	go func() {
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("📥 Archived: %s", subject))
	}()

    // Safe UI removal (preselect another index before removing)
    a.QueueUpdateDraw(func() {
        a.safeRemoveCurrentSelection(messageID)
    })
}

// moved to messages_bulk.go
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
		total := len(ids)
		for i, id := range ids {
			if err := a.Client.ArchiveMessage(id); err != nil {
				failed++
				continue
			}
			// Progress update on UI thread
			idx := i + 1
			a.QueueUpdateDraw(func() {
				a.setStatusPersistent(fmt.Sprintf("Archiving %d/%d…", idx, total))
			})
			// Remove from UI list on main thread after loop
		}
        a.QueueUpdateDraw(func() {
            a.removeIDsFromCurrentList(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.refreshTableDisplay()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(a.getSelectionStyle())
			}
			a.setStatusPersistent("")
			if failed == 0 {
				go func() {
					a.GetErrorHandler().ShowSuccess(a.ctx, "✅ Archived")
				}()
			} else {
				go func() {
					a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Archived with %d failure(s)", failed))
				}()
			}
		})
	}()
}

// moved to messages_bulk.go
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
		total := len(ids)
		for i, id := range ids {
			if err := a.Client.TrashMessage(id); err != nil {
				failed++
			}
			// Progress update on UI thread
			idx := i + 1
			a.QueueUpdateDraw(func() {
				a.setStatusPersistent(fmt.Sprintf("Trashing %d/%d…", idx, total))
			})
		}
        a.QueueUpdateDraw(func() {
            a.removeIDsFromCurrentList(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.refreshTableDisplay()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(a.getSelectionStyle())
			}
			a.setStatusPersistent("")
			if failed == 0 {
				go func() {
					a.GetErrorHandler().ShowSuccess(a.ctx, "✅ Trashed")
				}()
			} else {
				go func() {
					a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Trashed with %d failure(s)", failed))
				}()
			}
		})
	}()
}
*/

// replySelected replies to the selected message (placeholder)
func (a *App) replySelected() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		}()
		return
	}

	a.showCompositionWithStatusBar(services.CompositionTypeReply, messageID)
}

// replyAllSelected opens the composition panel for replying to all recipients
func (a *App) replyAllSelected() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		}()
		return
	}

	a.showCompositionWithStatusBar(services.CompositionTypeReplyAll, messageID)
}

// forwardSelected opens the composition panel for forwarding the current message
func (a *App) forwardSelected() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		}()
		return
	}

	a.showCompositionWithStatusBar(services.CompositionTypeForward, messageID)
}

// showAttachments opens the attachment picker for the current message
func (a *App) showAttachments() {
	go a.openAttachmentPicker()
}

// toggleMarkReadUnread toggles UNREAD label on selected message
func (a *App) toggleMarkReadUnread() {
	// Use helper function to get correct message index
	messageIndex := a.getCurrentSelectedMessageIndex()
	if messageIndex < 0 {
		a.showError("❌ No message selected")
		return
	}

	messageID := a.ids[messageIndex]
	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}

	// Determine unread state from cache if possible to avoid extra roundtrip
	isUnread := false
	if messageIndex < len(a.messagesMeta) && a.messagesMeta[messageIndex] != nil {
		for _, l := range a.messagesMeta[messageIndex].LabelIds {
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
		// Get EmailService to ensure undo actions are recorded
		emailService, _, _, _, _, _, _, _, _, _, _, _ := a.GetServices()

		if markUnread {
			if err := emailService.MarkAsUnread(a.ctx, messageID); err != nil {
				a.showError(fmt.Sprintf("❌ Error marking as unread: %v", err))
				return
			}
			a.showStatusMessage("✅ Message marked as unread")
			// Update caches/UI on main thread
			a.QueueUpdateDraw(func() {
				a.updateCachedMessageLabels(messageID, "UNREAD", true)
				a.refreshTableDisplay()
			})
		} else {
			if err := emailService.MarkAsRead(a.ctx, messageID); err != nil {
				a.showError(fmt.Sprintf("❌ Error marking as read: %v", err))
				return
			}
			a.showStatusMessage("✅ Message marked as read")
			a.QueueUpdateDraw(func() {
				a.updateCachedMessageLabels(messageID, "UNREAD", false)
				a.refreshTableDisplay()
			})
		}
	}(!isUnread)
}

// listUnreadMessages searches for all unread messages using is:unread query
func (a *App) listUnreadMessages() {
	a.performSearch("is:unread")
}

// listArchivedMessages searches for all archived messages using in:archive query
func (a *App) listArchivedMessages() {
	a.performSearch("in:archive")
}

// loadDrafts shows a draft picker in the side panel
func (a *App) loadDrafts() {
	if a.logger != nil {
		a.logger.Printf("loadDrafts: opening draft picker")
	}

	// Use side panel like labels
	go a.populateDraftsView()
}

// composeMessage starts composing a new email
func (a *App) composeMessage(draft bool) {
	if draft {
		// TODO: [FUTURE] Load draft composition using loadDrafts()
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "Draft loading functionality not yet implemented")
		}()
		return
	}

	a.showCompositionWithStatusBar(services.CompositionTypeNew, "")
}

// calculateHeaderHeight calculates the height needed for header content (same logic as adjustHeaderHeight)
func (a *App) calculateHeaderHeight(headerContent string) int {
	// Count the number of lines in the header content
	lines := strings.Count(headerContent, "\n") + 1
	// Set minimum height of 4 and maximum of 12 to prevent extreme cases
	minHeight := 4
	maxHeight := 12
	if lines < minHeight {
		lines = minHeight
	}
	if lines > maxHeight {
		lines = maxHeight
	}
	return lines
}

// toggleHeaderVisibility toggles the visibility of email headers and refreshes the current message display
func (a *App) toggleHeaderVisibility() {
	// Get DisplayService
	_, _, _, _, _, _, _, _, _, _, _, displayService := a.GetServices()
	if displayService == nil {
		if a.logger != nil {
			a.logger.Printf("toggleHeaderVisibility: displayService is nil")
		}
		return
	}

	// Toggle visibility and get new state
	newState := displayService.ToggleHeaderVisibility()

	// Refresh the current message to apply the change
	messageID := a.GetCurrentMessageID()
	if messageID != "" {
		a.refreshMessageContent(messageID)
	}

	// Show user feedback
	go func() {
		if newState {
			a.GetErrorHandler().ShowInfo(a.ctx, "📄 Headers visible")
		} else {
			a.GetErrorHandler().ShowInfo(a.ctx, "📄 Headers hidden - more space for content")
		}
	}()

	if a.logger != nil {
		a.logger.Printf("toggleHeaderVisibility: headers now %v", map[bool]string{true: "visible", false: "hidden"}[newState])
	}
}

// populateDraftsView loads and displays drafts in an enhanced picker with search functionality
func (a *App) populateDraftsView() {
	if a.logger != nil {
		a.logger.Printf("populateDraftsView: starting enhanced draft picker")
	}

	// Get services
	_, _, _, _, messageRepo, compositionService, _, _, _, _, _, _ := a.GetServices()

	// Create enhanced draft picker components
	input := tview.NewInputField()
	list := tview.NewList().ShowSecondaryText(true) // Enable secondary text for recipient display
	list.SetBorder(false)

	// Apply drafts theme colors
	draftsColors := a.GetComponentColors("drafts")
	input.SetFieldBackgroundColor(draftsColors.Background.Color())
	input.SetFieldTextColor(draftsColors.Text.Color())
	input.SetLabelColor(draftsColors.Title.Color())
	input.SetLabel("🔍 Search drafts: ")

	list.SetMainTextColor(draftsColors.Text.Color())
	list.SetSelectedTextColor(draftsColors.Background.Color())
	list.SetSelectedBackgroundColor(draftsColors.Accent.Color())
	list.SetBackgroundColor(draftsColors.Background.Color())

	// Draft data structures
	type draftItem struct {
		id      string
		subject string
		snippet string
		// date field removed as unused
		to string
	}

	var allDrafts []draftItem
	var visibleDrafts []draftItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visibleDrafts = visibleDrafts[:0]

		for _, draft := range allDrafts {
			if filter != "" {
				filterLower := strings.ToLower(filter)
				if !strings.Contains(strings.ToLower(draft.subject), filterLower) &&
					!strings.Contains(strings.ToLower(draft.snippet), filterLower) &&
					!strings.Contains(strings.ToLower(draft.to), filterLower) {
					// Check for special filters
					if strings.HasPrefix(filterLower, "to:") {
						toFilter := strings.TrimPrefix(filterLower, "to:")
						if !strings.Contains(strings.ToLower(draft.to), toFilter) {
							continue
						}
					} else if strings.HasPrefix(filterLower, "subject:") {
						subjectFilter := strings.TrimPrefix(filterLower, "subject:")
						if !strings.Contains(strings.ToLower(draft.subject), subjectFilter) {
							continue
						}
					} else {
						continue
					}
				}
			}

			visibleDrafts = append(visibleDrafts, draft)

			// Use visibleDrafts index for display
			visibleIndex := len(visibleDrafts) - 1

			// Create display text with subject on main line, recipient on secondary line
			displayText := fmt.Sprintf("📝 [%d] %s", visibleIndex+1, draft.subject)
			secondaryText := ""
			if draft.to != "" {
				secondaryText = fmt.Sprintf("       %s", draft.to) // Indent to align with subject
			}

			// Capture draft ID for closure - no shortcut rune to avoid yellow numbers
			draftID := draft.id
			list.AddItem(displayText, secondaryText, 0, func() {
				a.loadDraftForEditing(draftID, compositionService)
			})
		}

		// Add "Compose New Message" option if we have space
		if len(visibleDrafts) < 9 {
			list.AddItem("", "", 0, func() {}) // Separator
			list.AddItem("✏️ Compose New Message", "", 0, func() {
				a.hideDraftsPicker()
				a.composeMessage(false)
			})
		}

		// Update search input label with count
		if len(allDrafts) > 0 {
			input.SetLabel(fmt.Sprintf("🔍 Search drafts (%d/%d): ", len(visibleDrafts), len(allDrafts)))
		} else {
			input.SetLabel("🔍 Search drafts: ")
		}
	}

	// Load drafts in background
	go func() {
		drafts, err := messageRepo.GetDrafts(a.ctx, 50)
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("populateDraftsView: failed to load drafts: %v", err)
			}
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load drafts: %v", err))
			return
		}

		if len(drafts) == 0 {
			a.GetErrorHandler().ShowInfo(a.ctx, "No drafts found")
			return
		}

		// Convert to draftItem format
		allDrafts = make([]draftItem, 0, len(drafts))
		for _, draft := range drafts {
			subject := "No Subject"
			snippet := ""
			to := ""

			if draft.Message != nil && draft.Message.Payload != nil && draft.Message.Payload.Headers != nil {
				for _, header := range draft.Message.Payload.Headers {
					switch header.Name {
					case "Subject":
						subject = header.Value
					case "To":
						to = header.Value
						if len(to) > 30 {
							to = to[:30] + "..."
						}
					}
				}
				if draft.Message.Snippet != "" {
					snippet = draft.Message.Snippet
					if len(snippet) > 50 {
						snippet = snippet[:50] + "..."
					}
				}
			}

			allDrafts = append(allDrafts, draftItem{
				id:      draft.Id,
				subject: subject,
				snippet: snippet,
				to:      to,
			})
		}

		a.QueueUpdateDraw(func() {
			if a.logger != nil {
				a.logger.Printf("populateDraftsView: rendering enhanced picker with %d drafts", len(allDrafts))
			}

			// Set up search functionality
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })

			// Navigation from input to list
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyTab {
					if list.GetItemCount() > 0 {
						a.SetFocus(list)
						return nil
					}
				}
				if e.Key() == tcell.KeyEscape {
					a.hideDraftsPicker()
					return nil
				}
				if e.Key() == tcell.KeyEnter {
					// Select first visible draft if any
					if len(visibleDrafts) > 0 {
						a.loadDraftForEditing(visibleDrafts[0].id, compositionService)
						return nil
					}
				}
				// Handle number shortcuts
				if e.Rune() >= '1' && e.Rune() <= '9' {
					num := int(e.Rune() - '0')
					if num <= len(visibleDrafts) {
						a.loadDraftForEditing(visibleDrafts[num-1].id, compositionService)
						return nil
					}
				}
				if e.Key() == tcell.KeyCtrlN {
					a.hideDraftsPicker()
					a.composeMessage(false)
					return nil
				}
				return e
			})

			// Navigation from list back to input
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.hideDraftsPicker()
					return nil
				}
				// Handle number shortcuts in list too
				if e.Rune() >= '1' && e.Rune() <= '9' {
					num := int(e.Rune() - '0')
					if num <= len(visibleDrafts) {
						a.loadDraftForEditing(visibleDrafts[num-1].id, compositionService)
						return nil
					}
				}
				if e.Key() == tcell.KeyCtrlN {
					a.hideDraftsPicker()
					a.composeMessage(false)
					return nil
				}
				// Delete draft functionality
				if e.Rune() == 'D' || e.Key() == tcell.KeyDelete {
					currentIdx := list.GetCurrentItem()
					if currentIdx >= 0 && currentIdx < len(visibleDrafts) {
						draftToDelete := visibleDrafts[currentIdx]
						go func() {
							// Delete the draft
							if err := compositionService.DeleteComposition(a.ctx, draftToDelete.id); err != nil {
								a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to delete draft: %v", err))
								return
							}
							a.GetErrorHandler().ShowSuccess(a.ctx, "Draft deleted")
							// Reload the drafts list
							a.populateDraftsView()
						}()
					}
					return nil
				}
				return e
			})

			// Create container with proper layout
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBackgroundColor(draftsColors.Background.Color())
			container.SetBorder(true)
			container.SetTitle(" 📝 Drafts ")
			container.SetTitleColor(draftsColors.Title.Color())
			container.SetBorderColor(draftsColors.Border.Color())

			// Set background on child components
			input.SetBackgroundColor(draftsColors.Background.Color())
			list.SetBackgroundColor(draftsColors.Background.Color())

			// Add components to container
			container.AddItem(input, 3, 0, true) // Search input (3 lines, fixed)
			container.AddItem(list, 0, 1, true)  // Draft list (flexible)

			// Footer with instructions
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter/1-9 to edit | D to delete | Ctrl+N to compose new | Esc to cancel ")
			footer.SetTextColor(draftsColors.Text.Color())
			footer.SetBackgroundColor(draftsColors.Background.Color())
			container.AddItem(footer, 1, 0, false) // Instructions (1 line, fixed)

			// Show the enhanced picker
			a.showDraftsPicker(container)

			// Initial load and focus on input
			reload("")
			a.SetFocus(input)
		})
	}()
}

// showDraftsPicker shows the enhanced draft picker in the side panel
func (a *App) showDraftsPicker(container *tview.Flex) {
	// Following the pattern from other side panels (labels, attachments, etc.)
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		// Hide any existing panels first
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}

		// Replace a.labelsView with our enhanced container (same pattern as other pickers)
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true) // true = focusable for tab navigation
		split.ResizeItem(a.labelsView, 0, 1)

		// Update state
		a.setActivePicker(PickerDrafts) // Set specific drafts picker state
		a.currentFocus = "drafts"       // Set focus state to drafts
		a.updateFocusIndicators("drafts")

		if a.logger != nil {
			a.logger.Printf("showDraftsPicker: enhanced draft picker visible")
		}
	}
}

// hideDraftsPicker hides the draft picker and restores normal layout
func (a *App) hideDraftsPicker() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok && a.currentActivePicker == PickerDrafts {
		// Remove drafts list (using same slot as labels)
		split.ResizeItem(a.labelsView, 0, 0) // This hides the side panel

		// Update state
		a.setActivePicker(PickerNone)
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])

		if a.logger != nil {
			a.logger.Printf("hideDraftsPicker: draft picker hidden")
		}
	}
}

// hideDraftsPickerNoFocus hides the draft picker without setting focus to list (used when opening composition)
func (a *App) hideDraftsPickerNoFocus() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok && a.currentActivePicker == PickerDrafts {
		// Remove drafts list (using same slot as labels)
		split.ResizeItem(a.labelsView, 0, 0) // This hides the side panel

		// Update state but don't set focus
		a.setActivePicker(PickerNone)
	}
}

// loadDraftForEditing loads a draft and opens it in the composition panel
func (a *App) loadDraftForEditing(draftID string, compositionService services.CompositionService) {
	// Load the draft using the composition service
	go func() {
		composition, err := compositionService.LoadDraftComposition(a.ctx, draftID)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load draft: %v", err))
			return
		}

		// Hide the drafts picker and show composition panel
		a.hideDraftsPickerNoFocus()

		// Show the composition panel with the loaded draft
		if a.compositionPanel != nil {
			a.showCompositionWithDraft(composition)
		}
	}()
}

// detectCurrentFolderQuery attempts to detect what folder we're viewing based on current messages
func (a *App) detectCurrentFolderQuery() string {
	// Check if we have any messages in the current view
	messageIDs := a.GetMessageIDs()
	if len(messageIDs) == 0 {
		if a.logger != nil {
			a.logger.Printf("FOLDER_DETECT: No messages in current view, defaulting to inbox")
		}
		return "" // Default to inbox
	}

	// Sample a few messages to detect the common folder
	sampleSize := min(3, len(messageIDs))
	folderCounts := make(map[string]int)

	for i := 0; i < sampleSize; i++ {
		messageID := messageIDs[i]

		// Check message cache first
		if cached, ok := a.messageCache[messageID]; ok {
			folder := a.detectMessageFolder(cached.Labels)
			if folder != "" {
				folderCounts[folder]++
			}
		}
	}

	// Find the most common folder
	var detectedFolder string
	maxCount := 0
	for folder, count := range folderCounts {
		if count > maxCount {
			maxCount = count
			detectedFolder = folder
		}
	}

	if a.logger != nil {
		a.logger.Printf("FOLDER_DETECT: Detected folder '%s' based on %d messages", detectedFolder, maxCount)
	}

	return detectedFolder
}

// detectMessageFolder determines what folder a message belongs to based on its labels
func (a *App) detectMessageFolder(labels []string) string {
	labelSet := make(map[string]bool)
	for _, label := range labels {
		labelSet[label] = true
	}

	// Priority order for folder detection (most specific first)
	if labelSet["SPAM"] {
		return "in:spam"
	}
	if labelSet["TRASH"] {
		return "in:trash"
	}
	if labelSet["SENT"] {
		return "in:sent"
	}
	if labelSet["DRAFT"] {
		return "in:draft"
	}
	if !labelSet["INBOX"] {
		// If no INBOX label, likely archived
		return "in:archive"
	}

	// Default to inbox if INBOX label is present or no special labels detected
	return ""
}
