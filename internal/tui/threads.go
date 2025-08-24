package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/ajramos/gmail-tui/internal/services"
	gmailapi "google.golang.org/api/gmail/v1"
)

// ThreadViewMode represents the current threading display mode
type ThreadViewMode string

const (
	ThreadViewFlat   ThreadViewMode = "flat"
	ThreadViewThread ThreadViewMode = "thread"
)

// ThreadDisplayInfo holds UI-specific thread information
type ThreadDisplayInfo struct {
	*services.ThreadInfo
	IsExpanded bool
	Level      int  // Nesting level for replies (0 = root)
	IsVisible  bool // Whether this item should be shown in the current view
}

// Threading-related methods for App

// GetCurrentThreadViewMode returns the current threading view mode
func (a *App) GetCurrentThreadViewMode() ThreadViewMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if a.Config != nil && a.Config.Threading.Enabled {
		if a.currentView == "thread" {
			return ThreadViewThread
		}
	}
	
	return ThreadViewFlat
}

// SetCurrentThreadViewMode sets the current threading view mode
func (a *App) SetCurrentThreadViewMode(mode ThreadViewMode) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if mode == ThreadViewThread {
		a.currentView = "thread"
	} else {
		a.currentView = "flat"
	}
}

// ToggleThreadingMode toggles between flat and threaded view modes
func (a *App) ToggleThreadingMode() error {
	if !a.Config.Threading.Enabled {
		a.GetErrorHandler().ShowError(a.ctx, "Threading is disabled in configuration")
		return fmt.Errorf("threading disabled")
	}

	currentMode := a.GetCurrentThreadViewMode()
	
	if currentMode == ThreadViewFlat {
		a.SetCurrentThreadViewMode(ThreadViewThread)
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "Switched to threaded view")
		}()
		
		// Refresh the view to show threads
		go a.refreshThreadView()
	} else {
		a.SetCurrentThreadViewMode(ThreadViewFlat)
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "📄 Switched to flat view")
		}()
		
		// Refresh the view to show flat messages
		go a.refreshFlatView()
	}

	return nil
}

// refreshThreadView refreshes the display to show threaded conversations
func (a *App) refreshThreadView() {
	// Get thread service
	threadService := a.getThreadService()
	if threadService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Thread service not available")
		}()
		if a.logger != nil {
			a.logger.Printf("refreshThreadView: thread service is nil")
		}
		return
	}

	if a.logger != nil {
		a.logger.Printf("refreshThreadView: starting thread view refresh")
	}

	// Show progress with a slight delay to ensure visibility
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, "📧 Loading conversations...")
		time.Sleep(100 * time.Millisecond) // Brief pause to ensure message is visible
	}()

	// Get threads from Gmail
	opts := services.ThreadQueryOptions{
		MaxResults: 50,
		LabelIDs:   []string{"INBOX"}, // Start with inbox threads
	}

	if a.logger != nil {
		a.logger.Printf("refreshThreadView: calling GetThreads with opts: %+v", opts)
	}

	threadPage, err := threadService.GetThreads(a.ctx, opts)
	if err != nil {
		go func() {
			a.GetErrorHandler().ClearProgress()
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load threads: %v", err))
		}()
		if a.logger != nil {
			a.logger.Printf("refreshThreadView: GetThreads failed: %v", err)
		}
		return
	}

	if a.logger != nil {
		a.logger.Printf("refreshThreadView: GetThreads succeeded, got %d threads", len(threadPage.Threads))
	}

	// Show progress before processing threads
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Processing %d conversations...", len(threadPage.Threads)))
	}()
	
	// Update the UI with thread data
	if a.logger != nil {
		a.logger.Printf("refreshThreadView: calling displayThreads")
	}
	a.displayThreads(threadPage.Threads)
	
	// Clear progress and show success after processing with longer delay
	go func() {
		time.Sleep(1000 * time.Millisecond) // Allow 1 second for progress message to be visible
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Loaded %d conversations", len(threadPage.Threads)))
	}()
}

// refreshFlatView refreshes the display to show flat message list
func (a *App) refreshFlatView() {
	// Use existing message loading logic
	go a.reloadMessages()
}

// displayThreads updates the message list to show threads
func (a *App) displayThreads(threads []*services.ThreadInfo) {
	if a.logger != nil {
		a.logger.Printf("displayThreads: called with %d threads", len(threads))
	}
	
	a.QueueUpdateDraw(func() {
		table, ok := a.views["list"].(*tview.Table)
		if !ok {
			if a.logger != nil {
				a.logger.Printf("displayThreads: views[\"list\"] is not a *tview.Table")
			}
			return
		}

		if a.logger != nil {
			a.logger.Printf("displayThreads: clearing table and populating with threads")
		}

		// Clear existing content
		table.Clear()

		// Track threads for state management
		a.mu.Lock()
		threadIDs := make([]string, 0, len(threads)*5) // Allocate more space for individual messages
		threadMeta := make([]*services.ThreadInfo, 0, len(threads))
		allRowMeta := make([]interface{}, 0, len(threads)*5) // Track both threads and messages
		
		rowIndex := 0
		for i, thread := range threads {
			if thread == nil {
				continue
			}
			
			threadIDs = append(threadIDs, thread.ThreadID)
			threadMeta = append(threadMeta, thread)
			allRowMeta = append(allRowMeta, thread) // Store thread info
			
			// Format thread header for display
			threadText := a.formatThreadForList(thread, i)
			
			// Create thread header cell with appropriate styling
			cell := tview.NewTableCell(threadText).
				SetExpansion(1).
				SetAlign(tview.AlignLeft)
			
			// Apply thread-specific styling
			if thread.UnreadCount > 0 {
				cell.SetTextColor(a.currentTheme.UI.InfoColor.Color())
			} else {
				cell.SetTextColor(a.currentTheme.UI.FooterColor.Color())
			}
			
			table.SetCell(rowIndex, 0, cell)
			rowIndex++
			
			// Check if thread is expanded and add individual messages
			threadService := a.getThreadService()
			if threadService != nil && thread.MessageCount > 1 {
				accountEmail, _ := a.Client.ActiveAccountEmail(a.ctx)
				if accountEmail != "" {
					isExpanded, _ := threadService.IsThreadExpanded(a.ctx, accountEmail, thread.ThreadID)
					if isExpanded {
						// Fetch and display individual messages
						messages, err := a.fetchThreadMessages(a.ctx, thread.ThreadID)
						if err != nil {
							if a.logger != nil {
								a.logger.Printf("displayThreads: failed to fetch messages for thread %s: %v", thread.ThreadID, err)
							}
							// Add error message row
							errorText := "    ⚠️  Failed to load thread messages"
							errorCell := tview.NewTableCell(errorText).
								SetExpansion(1).
								SetAlign(tview.AlignLeft).
								SetTextColor(tcell.ColorOrange) // Use a warning color
							table.SetCell(rowIndex, 0, errorCell)
							threadIDs = append(threadIDs, "") // Placeholder ID
							allRowMeta = append(allRowMeta, nil) // Error marker
							rowIndex++
						} else {
							// Add individual message rows
							for msgIndex, message := range messages {
								messageText := a.formatThreadMessageForList(message, msgIndex, len(messages))
								
								messageCell := tview.NewTableCell(messageText).
									SetExpansion(1).
									SetAlign(tview.AlignLeft)
								
								// Style individual messages differently (slightly dimmer)
								messageCell.SetTextColor(a.currentTheme.UI.FooterColor.Color())
								
								table.SetCell(rowIndex, 0, messageCell)
								
								// Store message ID and metadata
								threadIDs = append(threadIDs, message.Id)
								allRowMeta = append(allRowMeta, message) // Store message info
								rowIndex++
							}
						}
					}
				}
			}
		}
		
		// Update app state (supporting both threads and individual messages)
		a.ids = threadIDs
		
		// Store metadata for all rows (threads and messages)
		a.messagesMeta = make([]*gmailapi.Message, len(allRowMeta))
		for i, rowData := range allRowMeta {
			if rowData == nil {
				// Error row - create placeholder
				a.messagesMeta[i] = &gmailapi.Message{
					Id:      "",
					Snippet: "Error loading thread messages",
				}
			} else if thread, isThread := rowData.(*services.ThreadInfo); isThread {
				// Thread header - create fake message structure for compatibility
				var fromField string
				if len(thread.Participants) > 0 {
					fromField = thread.Participants[0]
				} else {
					fromField = "Thread Participants"
				}
				
				fakeMsg := &gmailapi.Message{
					Id:       thread.ThreadID,
					ThreadId: thread.ThreadID,
					Snippet:  thread.Subject,
					Payload: &gmailapi.MessagePart{
						Headers: []*gmailapi.MessagePartHeader{
							{Name: "Subject", Value: thread.Subject},
							{Name: "From", Value: fromField},
							{Name: "Date", Value: thread.LatestDate.Format("Mon, 02 Jan 2006 15:04:05 -0700")},
						},
					},
					LabelIds: thread.Labels,
				}
				a.messagesMeta[i] = fakeMsg
			} else if message, isMessage := rowData.(*gmailapi.Message); isMessage {
				// Individual message - store directly
				a.messagesMeta[i] = message
			}
		}
		
		a.mu.Unlock()

		// Auto-select first thread if available
		if len(threads) > 0 {
			table.Select(0, 0)
			a.SetCurrentMessageID(threads[0].ThreadID)
		}
	})
}

// formatThreadForList formats a thread for display in the message list
func (a *App) formatThreadForList(thread *services.ThreadInfo, index int) string {
	var builder strings.Builder
	
	// Add message number if enabled
	if a.showMessageNumbers {
		builder.WriteString(fmt.Sprintf("%3d ", index+1))
	}
	
	// Add unified expansion indicator for all messages/threads
	var isExpanded bool
	threadService := a.getThreadService()
	if threadService != nil && thread.MessageCount > 1 {
		accountEmail, _ := a.Client.ActiveAccountEmail(a.ctx)
		if accountEmail != "" {
			var err error
			isExpanded, err = threadService.IsThreadExpanded(a.ctx, accountEmail, thread.ThreadID)
			if a.logger != nil {
				a.logger.Printf("formatThreadForList: thread %s, accountEmail=%s, isExpanded=%v, err=%v", thread.ThreadID, accountEmail, isExpanded, err)
			}
		}
	}
	
	// Emoji markers: 📧 for single messages, ▶️/▼️ for threads
	if thread.MessageCount > 1 {
		// Multi-message thread - use expansion icons
		if isExpanded {
			builder.WriteString("▼️ ")
			if a.logger != nil {
				a.logger.Printf("formatThreadForList: showing ▼️ for expanded thread %s", thread.ThreadID)
			}
		} else {
			builder.WriteString("▶️ ")
			if a.logger != nil {
				a.logger.Printf("formatThreadForList: showing ▶️ for collapsed thread %s", thread.ThreadID)
			}
		}
	} else {
		// Single message - use email icon
		builder.WriteString("📧 ")
		if a.logger != nil {
			a.logger.Printf("formatThreadForList: showing 📧 for single message %s", thread.ThreadID)
		}
	}
	
	// Add unread indicator with proper spacing
	if thread.UnreadCount > 0 {
		builder.WriteString("● ")
	} else {
		builder.WriteString("○ ")
	}
	
	// Get subject and participant info
	subject := thread.Subject
	if subject == "" {
		subject = "(No Subject)"
	}
	
	// Get primary participant (exclude self)
	var primaryParticipant string
	if len(thread.Participants) > 0 {
		primaryParticipant = thread.Participants[0]
	}
	
	// Add thread count in brackets after sender for multi-message threads
	var countSuffix string
	if thread.MessageCount > 1 {
		if thread.MessageCount >= 100 {
			countSuffix = " [99+]"
		} else {
			countSuffix = fmt.Sprintf(" [%d]", thread.MessageCount)
		}
	}

	// Format sender with thread count
	var senderWithCount string
	if primaryParticipant != "" {
		senderWithCount = primaryParticipant + countSuffix
	} else {
		senderWithCount = "(No sender)" + countSuffix
	}
	
	// Build attachment indicator
	var attachmentIcon string
	if thread.HasAttachment {
		attachmentIcon = "📎"
	}
	
	// Format date
	var dateStr string
	now := time.Now()
	threadTime := thread.LatestDate
	
	if threadTime.After(now.Add(-24 * time.Hour)) {
		dateStr = threadTime.Format("3:04 PM")
	} else if threadTime.After(now.Add(-7 * 24 * time.Hour)) {
		dateStr = threadTime.Format("Mon 3:04 PM")
	} else if threadTime.Year() == now.Year() {
		dateStr = threadTime.Format("Jan 02")
	} else {
		dateStr = threadTime.Format("2006")
	}
	
	// Simple format: [marker] [status] [sender] [subject] [attachment] [date]
	builder.WriteString(fmt.Sprintf("%s %s %s | %s", senderWithCount, subject, attachmentIcon, dateStr))
	
	return builder.String()
}


// fetchThreadMessages retrieves individual messages for a thread
func (a *App) fetchThreadMessages(ctx context.Context, threadID string) ([]*gmailapi.Message, error) {
	threadService := a.getThreadService()
	if threadService == nil {
		return nil, fmt.Errorf("thread service not available")
	}
	
	messages, err := threadService.GetThreadMessages(ctx, threadID, services.MessageQueryOptions{
		Format:    "metadata", // Get metadata for list display
		SortOrder: "asc",      // Chronological order
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch thread messages: %w", err)
	}
	
	return messages, nil
}

// formatThreadMessageForList formats an individual thread message for display in the list
func (a *App) formatThreadMessageForList(message *gmailapi.Message, messageIndex, totalMessages int) string {
	var builder strings.Builder
	
	// Add message number if enabled
	if a.showMessageNumbers {
		builder.WriteString(fmt.Sprintf("%3s ", "")) // Empty space to align with thread numbers
	}
	
	// Add tree-like indentation structure
	if messageIndex == totalMessages-1 {
		builder.WriteString("    └─ ") // Last message
	} else {
		builder.WriteString("    ├─ ") // Intermediate message
	}
	
	// Add message icon
	builder.WriteString("📧 ")
	
	// Add read/unread indicator
	isUnread := true // Default assumption
	for _, labelID := range message.LabelIds {
		if labelID == "UNREAD" {
			isUnread = true
			break
		}
		if labelID == "INBOX" {
			isUnread = false // Assume read if only INBOX label
		}
	}
	
	if isUnread {
		builder.WriteString("● ")
	} else {
		builder.WriteString("○ ")
	}
	
	// Add attachment indicator
	hasAttachment := false
	if message.Payload != nil {
		// Check for attachments in parts
		var checkForAttachments func(*gmailapi.MessagePart) bool
		checkForAttachments = func(part *gmailapi.MessagePart) bool {
			if part.Filename != "" && part.Filename != "." {
				return true
			}
			for _, subpart := range part.Parts {
				if checkForAttachments(subpart) {
					return true
				}
			}
			return false
		}
		hasAttachment = checkForAttachments(message.Payload)
	}
	if hasAttachment {
		builder.WriteString("📎 ")
	}
	
	// Get sender and subject
	var fromField, subjectField string
	if message.Payload != nil {
		for _, header := range message.Payload.Headers {
			switch header.Name {
			case "From":
				fromField = header.Value
			case "Subject":
				subjectField = header.Value
			}
		}
	}
	
	// Format sender (shorter for indented display)
	if fromField != "" {
		// Extract email or name
		if strings.Contains(fromField, "<") {
			// Format: "Name <email@domain.com>"
			if start := strings.Index(fromField, "\""); start != -1 {
				if end := strings.Index(fromField[start+1:], "\""); end != -1 {
					fromField = fromField[start+1 : start+1+end]
				}
			} else {
				if end := strings.Index(fromField, " <"); end != -1 {
					fromField = fromField[:end]
				}
			}
		}
		// Truncate for display
		if len(fromField) > 20 {
			fromField = fromField[:17] + "..."
		}
	} else {
		fromField = "Unknown"
	}
	
	// Format subject (remove "Re: " prefixes for cleaner display)
	if subjectField != "" {
		subjectField = strings.TrimPrefix(subjectField, "Re: ")
		subjectField = strings.TrimPrefix(subjectField, "RE: ")
		subjectField = strings.TrimPrefix(subjectField, "Fwd: ")
		subjectField = strings.TrimPrefix(subjectField, "FWD: ")
		
		// Truncate subject for indented display
		maxSubjectLen := 40
		if len(subjectField) > maxSubjectLen {
			subjectField = subjectField[:maxSubjectLen-3] + "..."
		}
	} else {
		subjectField = "(No Subject)"
	}
	
	// Add formatted content
	builder.WriteString(fmt.Sprintf("%-20s %s", fromField, subjectField))
	
	// Add improved date formatting for thread messages
	if message.InternalDate > 0 {
		timestamp := message.InternalDate / 1000 // Convert from milliseconds
		messageTime := time.Unix(timestamp, 0)
		now := time.Now()
		
		var dateStr string
		if messageTime.After(now.Add(-24 * time.Hour)) {
			// Today - show time only
			dateStr = messageTime.Format("3:04 PM")
		} else if messageTime.After(now.Add(-7*24*time.Hour)) {
			// This week - show day only (shorter for indented display)
			dateStr = messageTime.Format("Mon")
		} else if messageTime.Year() == now.Year() {
			// This year - show month and day
			dateStr = messageTime.Format("Jan 02")
		} else {
			// Older - show year
			dateStr = messageTime.Format("2006")
		}
		
		// Right-align date with less padding for indented messages
		currentLen := len(builder.String())
		targetWidth := a.screenWidth - 25 // More margin for indented content
		if currentLen < targetWidth {
			padding := targetWidth - currentLen - len(dateStr)
			if padding > 0 {
				builder.WriteString(strings.Repeat(" ", padding))
			}
		}
		builder.WriteString(dateStr)
	}
	
	return builder.String()
}

// expandThreadAsync handles thread expansion without full UI refresh to preserve cursor position
func (a *App) expandThreadAsync(threadID string, isExpanded bool) {
	if a.logger != nil {
		a.logger.Printf("expandThreadAsync: threadID=%s, isExpanded=%v", threadID, isExpanded)
	}

	table, ok := a.views["list"].(*tview.Table)
	if !ok {
		if a.logger != nil {
			a.logger.Printf("expandThreadAsync: list view is not a table")
		}
		return
	}

	// Find the thread row in the current display
	threadRowIndex := -1
	a.mu.Lock()
	for i, id := range a.ids {
		if id == threadID {
			threadRowIndex = i
			break
		}
	}
	a.mu.Unlock()

	if threadRowIndex == -1 {
		if a.logger != nil {
			a.logger.Printf("expandThreadAsync: thread %s not found in current display", threadID)
		}
		return
	}

	if isExpanded {
		// Add loading placeholder immediately
		a.QueueUpdateDraw(func() {
			a.insertThreadLoadingPlaceholder(table, threadRowIndex+1, threadID)
		})

		// Fetch messages asynchronously
		go func() {
			messages, err := a.fetchThreadMessages(a.ctx, threadID)
			if err != nil {
				if a.logger != nil {
					a.logger.Printf("expandThreadAsync: failed to fetch messages: %v", err)
				}
				// Replace loading with error
				a.QueueUpdateDraw(func() {
					a.replaceLoadingWithError(table, threadRowIndex+1, threadID)
				})
				return
			}

			// Replace loading with actual messages
			a.QueueUpdateDraw(func() {
				a.replaceLoadingWithMessages(table, threadRowIndex+1, threadID, messages)
			})
			
			// Clear progress status
			go func() {
				a.GetErrorHandler().ClearProgress()
			}()
		}()
	} else {
		// Collapse: remove all child messages immediately
		a.QueueUpdateDraw(func() {
			a.collapseThreadMessages(table, threadRowIndex, threadID)
		})
	}
}

// insertThreadLoadingPlaceholder adds a loading indicator below the thread
func (a *App) insertThreadLoadingPlaceholder(table *tview.Table, insertIndex int, threadID string) {
	if a.logger != nil {
		a.logger.Printf("insertThreadLoadingPlaceholder: inserting at index %d for thread %s", insertIndex, threadID)
	}

	// Shift existing rows down
	rowCount := table.GetRowCount()
	for i := rowCount; i > insertIndex; i-- {
		if i-1 >= 0 {
			cell := table.GetCell(i-1, 0)
			if cell != nil {
				table.SetCell(i, 0, cell)
			}
		}
	}

	// Insert loading placeholder
	loadingText := "    ⏳ Loading thread messages..."
	loadingCell := tview.NewTableCell(loadingText).
		SetExpansion(1).
		SetAlign(tview.AlignLeft).
		SetTextColor(a.currentTheme.UI.InfoColor.Color())
	
	table.SetCell(insertIndex, 0, loadingCell)

	// Update app state
	a.mu.Lock()
	// Insert placeholder in IDs and metadata
	a.ids = append(a.ids[:insertIndex], append([]string{""}, a.ids[insertIndex:]...)...)
	placeholderMsg := &gmailapi.Message{Id: "", Snippet: "Loading..."}
	a.messagesMeta = append(a.messagesMeta[:insertIndex], append([]*gmailapi.Message{placeholderMsg}, a.messagesMeta[insertIndex:]...)...)
	a.mu.Unlock()

	// Show progress message
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, "📧 Loading thread messages...")
	}()
}

// replaceLoadingWithMessages replaces the loading placeholder with actual thread messages
func (a *App) replaceLoadingWithMessages(table *tview.Table, loadingIndex int, threadID string, messages []*gmailapi.Message) {
	if a.logger != nil {
		a.logger.Printf("replaceLoadingWithMessages: replacing loading at index %d with %d messages", loadingIndex, len(messages))
	}

	// Remove the loading placeholder first
	a.removeTableRow(table, loadingIndex)

	// Insert actual message rows
	for i, message := range messages {
		insertIndex := loadingIndex + i
		messageText := a.formatThreadMessageForList(message, i, len(messages))
		
		messageCell := tview.NewTableCell(messageText).
			SetExpansion(1).
			SetAlign(tview.AlignLeft).
			SetTextColor(a.currentTheme.UI.FooterColor.Color())

		a.insertTableRow(table, insertIndex, messageCell, message.Id, message)
	}
}

// replaceLoadingWithError replaces loading placeholder with error message
func (a *App) replaceLoadingWithError(table *tview.Table, loadingIndex int, threadID string) {
	errorText := "    ⚠️  Failed to load thread messages"
	errorCell := tview.NewTableCell(errorText).
		SetExpansion(1).
		SetAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorOrange)

	table.SetCell(loadingIndex, 0, errorCell)

	// Update metadata
	a.mu.Lock()
	if loadingIndex < len(a.messagesMeta) {
		a.messagesMeta[loadingIndex] = &gmailapi.Message{Id: "", Snippet: "Error loading thread messages"}
	}
	a.mu.Unlock()
}

// collapseThreadMessages removes all child messages of a thread
func (a *App) collapseThreadMessages(table *tview.Table, threadRowIndex int, threadID string) {
	if a.logger != nil {
		a.logger.Printf("collapseThreadMessages: collapsing thread at index %d for threadID %s", threadRowIndex, threadID)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Find all expanded message rows that belong to this thread
	// They are consecutive rows after the thread header with empty IDs (individual messages)
	rowsToRemove := 0
	for i := threadRowIndex + 1; i < len(a.ids) && i < len(a.messagesMeta); i++ {
		// Stop when we hit another thread (non-empty ID that matches a thread pattern)
		if a.ids[i] != "" {
			// This is another thread header, stop here
			break
		}
		
		// This is an expanded message row (empty ID), remove it
		rowsToRemove++
		if a.logger != nil {
			a.logger.Printf("collapseThreadMessages: marking row %d for removal", i)
		}
		
		// Safety check: don't remove more than 50 rows (prevent infinite loop)
		if rowsToRemove > 50 {
			if a.logger != nil {
				a.logger.Printf("collapseThreadMessages: safety break, too many rows to remove")
			}
			break
		}
	}

	if a.logger != nil {
		a.logger.Printf("collapseThreadMessages: removing %d rows", rowsToRemove)
	}

	// Remove rows in reverse order to maintain indices
	for i := threadRowIndex + rowsToRemove; i > threadRowIndex; i-- {
		if a.logger != nil {
			a.logger.Printf("collapseThreadMessages: removing row %d", i)
		}
		a.removeTableRow(table, i)
	}
}

// Helper functions for table manipulation
func (a *App) insertTableRow(table *tview.Table, index int, cell *tview.TableCell, id string, meta *gmailapi.Message) {
	// Shift existing rows down
	rowCount := table.GetRowCount()
	for i := rowCount; i > index; i-- {
		if i-1 >= 0 {
			existingCell := table.GetCell(i-1, 0)
			if existingCell != nil {
				table.SetCell(i, 0, existingCell)
			}
		}
	}

	// Insert new row
	table.SetCell(index, 0, cell)

	// Update app state
	a.mu.Lock()
	a.ids = append(a.ids[:index], append([]string{id}, a.ids[index:]...)...)
	a.messagesMeta = append(a.messagesMeta[:index], append([]*gmailapi.Message{meta}, a.messagesMeta[index:]...)...)
	a.mu.Unlock()
}

func (a *App) removeTableRow(table *tview.Table, index int) {
	// Shift rows up
	rowCount := table.GetRowCount()
	for i := index; i < rowCount-1; i++ {
		cell := table.GetCell(i+1, 0)
		if cell != nil {
			table.SetCell(i, 0, cell)
		}
	}
	
	// Remove last row
	table.RemoveRow(rowCount - 1)

	// Update app state
	a.mu.Lock()
	if index < len(a.ids) {
		a.ids = append(a.ids[:index], a.ids[index+1:]...)
	}
	if index < len(a.messagesMeta) {
		a.messagesMeta = append(a.messagesMeta[:index], a.messagesMeta[index+1:]...)
	}
	a.mu.Unlock()
}

// ExpandThread expands a thread to show its messages
func (a *App) ExpandThread() error {
	threadID := a.GetCurrentMessageID() // In thread mode, this is actually a thread ID
	if threadID == "" {
		return fmt.Errorf("no thread selected")
	}

	// Get thread service
	threadService := a.getThreadService()
	if threadService == nil {
		return fmt.Errorf("thread service not available")
	}

	// Get account email
	accountEmail, err := a.Client.ActiveAccountEmail(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to get account email: %w", err)
	}

	// Toggle expansion state
	isExpanded, err := threadService.IsThreadExpanded(a.ctx, accountEmail, threadID)
	if err != nil {
		return fmt.Errorf("failed to check thread expansion state: %w", err)
	}
	
	if a.logger != nil {
		a.logger.Printf("ExpandThread: threadID=%s, currentExpanded=%v", threadID, isExpanded)
	}

	newExpandedState := !isExpanded
	err = threadService.SetThreadExpanded(a.ctx, accountEmail, threadID, newExpandedState)
	if err != nil {
		return fmt.Errorf("failed to set thread expansion: %w", err)
	}
	
	if a.logger != nil {
		a.logger.Printf("ExpandThread: set threadID=%s to expanded=%v", threadID, newExpandedState)
	}

	// Instead of full refresh, expand thread asynchronously to preserve cursor position
	go a.expandThreadAsync(threadID, newExpandedState)
	
	// Show feedback
	if newExpandedState {
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "Thread expanded")
		}()
	} else {
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "Thread collapsed")
		}()
	}

	return nil
}

// ExpandAllThreads expands all visible threads
func (a *App) ExpandAllThreads() error {
	if a.GetCurrentThreadViewMode() != ThreadViewThread {
		return fmt.Errorf("not in thread view mode")
	}

	// Get thread service
	threadService := a.getThreadService()
	if threadService == nil {
		return fmt.Errorf("thread service not available")
	}

	// Get account email
	accountEmail, err := a.Client.ActiveAccountEmail(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to get account email: %w", err)
	}

	// Expand all threads
	err = threadService.ExpandAllThreads(a.ctx, accountEmail)
	if err != nil {
		return fmt.Errorf("failed to expand all threads: %w", err)
	}

	// Refresh view
	go a.refreshThreadView()
	
	// Show feedback
	go func() {
		a.GetErrorHandler().ShowSuccess(a.ctx, "📤 All threads expanded")
	}()

	return nil
}

// CollapseAllThreads collapses all visible threads
func (a *App) CollapseAllThreads() error {
	if a.GetCurrentThreadViewMode() != ThreadViewThread {
		return fmt.Errorf("not in thread view mode")
	}

	// Get thread service
	threadService := a.getThreadService()
	if threadService == nil {
		return fmt.Errorf("thread service not available")
	}

	// Get account email
	accountEmail, err := a.Client.ActiveAccountEmail(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to get account email: %w", err)
	}

	// Collapse all threads
	err = threadService.CollapseAllThreads(a.ctx, accountEmail)
	if err != nil {
		return fmt.Errorf("failed to collapse all threads: %w", err)
	}

	// Refresh view
	go a.refreshThreadView()
	
	// Show feedback
	go func() {
		a.GetErrorHandler().ShowSuccess(a.ctx, "📥 All threads collapsed")
	}()

	return nil
}

// GenerateThreadSummary generates an AI summary for the selected thread
func (a *App) GenerateThreadSummary() error {
	threadID := a.GetCurrentMessageID() // In thread mode, this is actually a thread ID
	if threadID == "" {
		return fmt.Errorf("no thread selected")
	}

	// Get thread service
	threadService := a.getThreadService()
	if threadService == nil {
		return fmt.Errorf("thread service not available")
	}

	// Get account email
	accountEmail, err := a.Client.ActiveAccountEmail(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to get account email: %w", err)
	}

	// Show progress
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, "🧠 Generating thread summary...")
	}()

	// Prepare summary options
	summaryOptions := services.ThreadSummaryOptions{
		MaxLength:       500,
		Language:        "en",
		StreamEnabled:   true,
		UseCache:        true,
		ForceRegenerate: false,
		AccountEmail:    accountEmail,
		SummaryType:     "conversation",
	}

	// Generate summary with streaming
	var summaryResult *services.ThreadSummaryResult

	if summaryOptions.StreamEnabled {
		// Use streaming summary generation
		summaryResult, err = threadService.GenerateThreadSummaryStream(a.ctx, threadID, summaryOptions, func(token string) {
			// Update AI panel with streaming tokens
			a.QueueUpdateDraw(func() {
				if a.aiSummaryView != nil {
					currentText := a.aiSummaryView.GetText(false)
					a.aiSummaryView.SetText(currentText + token)
					a.aiSummaryView.ScrollToEnd()
				}
			})
		})
	} else {
		// Use non-streaming summary generation
		summaryResult, err = threadService.GenerateThreadSummary(a.ctx, threadID, summaryOptions)
	}

	// Clear progress and handle result
	go func() {
		a.GetErrorHandler().ClearProgress()
		
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to generate thread summary: %v", err))
			return
		}

		if summaryResult.FromCache {
			a.GetErrorHandler().ShowInfo(a.ctx, "🧠 Thread summary loaded from cache")
		} else {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("🧠 Thread summary generated (%d messages)", summaryResult.MessageCount))
		}
	}()

	// Show the AI panel with the summary
	if err == nil {
		a.QueueUpdateDraw(func() {
			if a.aiSummaryView != nil {
				if !summaryOptions.StreamEnabled {
					a.aiSummaryView.SetText(summaryResult.Summary)
				}
				a.showAIPanel()
			}
		})
	}

	return err
}

// showAIPanel displays the AI summary panel
func (a *App) showAIPanel() {
	if a.aiSummaryView == nil {
		return
	}

	// Show the AI summary panel (reuse existing logic)
	a.aiSummaryVisible = true
	a.aiPanelInPromptMode = false

	// Update layout to show the AI panel
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.aiSummaryView, 0, 1)
	}
}

// IsThreadingEnabled returns whether threading functionality is enabled
func (a *App) IsThreadingEnabled() bool {
	return a.Config != nil && a.Config.Threading.Enabled
}

// GetThreadingConfig returns the current threading configuration
func (a *App) GetThreadingConfig() services.ThreadingConfig {
	if a.Config == nil {
		return services.ThreadingConfig{}
	}
	
	// Convert config.ThreadingConfig to services.ThreadingConfig
	return services.ThreadingConfig{
		Enabled:               a.Config.Threading.Enabled,
		DefaultView:           a.Config.Threading.DefaultView,
		AutoExpandUnread:      a.Config.Threading.AutoExpandUnread,
		ShowThreadCount:       a.Config.Threading.ShowThreadCount,
		IndentReplies:         a.Config.Threading.IndentReplies,
		MaxThreadDepth:        a.Config.Threading.MaxThreadDepth,
		ThreadSummaryEnabled:  a.Config.Threading.ThreadSummaryEnabled,
		PreserveThreadState:   a.Config.Threading.PreserveThreadState,
	}
}

// updateThreadDisplay updates the UI to show thread expansion without reloading from Gmail
func (a *App) updateThreadDisplay(threadID string, isExpanded bool) {
	if a.logger != nil {
		a.logger.Printf("updateThreadDisplay: called with threadID=%s, isExpanded=%v", threadID, isExpanded)
	}
	
	// Get thread service
	threadService := a.getThreadService()
	if threadService == nil {
		if a.logger != nil {
			a.logger.Printf("updateThreadDisplay: thread service is nil")
		}
		return
	}

	if isExpanded {
		// For expanded threads, show additional detail in the same row
		a.QueueUpdateDraw(func() {
			if a.logger != nil {
				a.logger.Printf("updateThreadDisplay: inside QueueUpdateDraw for expansion")
			}
			
			table, ok := a.views["list"].(*tview.Table)
			if !ok {
				if a.logger != nil {
					a.logger.Printf("updateThreadDisplay: views[list] is not a table")
				}
				return
			}

			if a.logger != nil {
				a.logger.Printf("updateThreadDisplay: searching for threadID=%s in %d ids", threadID, len(a.ids))
			}

			// Find the thread row
			threadRowIndex := -1
			for i, id := range a.ids {
				if a.logger != nil {
					a.logger.Printf("updateThreadDisplay: checking id[%d]=%s", i, id)
				}
				if id == threadID {
					threadRowIndex = i
					if a.logger != nil {
						a.logger.Printf("updateThreadDisplay: found thread at row %d", i)
					}
					break
				}
			}

			if threadRowIndex == -1 {
				if a.logger != nil {
					a.logger.Printf("updateThreadDisplay: thread not found in ids list")
				}
				return
			}

			// Update the thread row to show expanded state with more detail
			cell := table.GetCell(threadRowIndex, 0)
			if cell != nil {
				currentText := cell.Text
				if a.logger != nil {
					a.logger.Printf("updateThreadDisplay: current cell text: '%s'", currentText)
				}
				if strings.Contains(currentText, "▶️") {
					// Change ▶️ to ▼️ and add expansion details
					expandedText := strings.Replace(currentText, "▶️", "▼️", 1)
					expandedText += " [EXPANDED - Press Enter to collapse]"
					if a.logger != nil {
						a.logger.Printf("updateThreadDisplay: setting new text: '%s'", expandedText)
					}
					cell.SetText(expandedText)
					// Make expanded threads more visually distinct
					cell.SetTextColor(a.currentTheme.UI.InfoColor.Color())
					if a.logger != nil {
						a.logger.Printf("updateThreadDisplay: cell text updated successfully")
					}
				} else {
					if a.logger != nil {
						a.logger.Printf("updateThreadDisplay: no ▶️ found in current text")
					}
				}
			} else {
				if a.logger != nil {
					a.logger.Printf("updateThreadDisplay: cell is nil at row %d", threadRowIndex)
				}
			}
		})
		// Queue another update to force refresh
		a.QueueUpdate(func() {
			if a.logger != nil {
				a.logger.Printf("updateThreadDisplay: QueueUpdate called for refresh")
			}
		})
		// After QueueUpdateDraw, force a draw to ensure immediate visibility
		a.ForceDraw()
		if a.logger != nil {
			a.logger.Printf("updateThreadDisplay: ForceDraw() called outside queue")
		}
	} else {
		// Just update the expansion indicator to collapsed
		a.QueueUpdateDraw(func() {
			table, ok := a.views["list"].(*tview.Table)
			if !ok {
				return
			}

			// Find the thread row and update its display
			for i, id := range a.ids {
				if id == threadID {
					cell := table.GetCell(i, 0)
					if cell != nil {
						currentText := cell.Text
						if strings.Contains(currentText, "▼️") {
							// Remove expansion details and change ▼️ back to ▶️
							collapsedText := strings.Replace(currentText, "▼️", "▶️", 1)
							// Remove the expansion detail text
							collapsedText = strings.Replace(collapsedText, " [EXPANDED - Press Enter to collapse]", "", 1)
							cell.SetText(collapsedText)
							// Reset color to default
							cell.SetTextColor(a.currentTheme.UI.FooterColor.Color())
							// Force table redraw to show the changes
							table.SetTitle(table.GetTitle()) // Trigger table refresh
						}
					}
					break
				}
			}
		})
		// After QueueUpdateDraw, force a draw to ensure immediate visibility
		a.ForceDraw()
		if a.logger != nil {
			a.logger.Printf("updateThreadDisplay: ForceDraw() called for collapse")
		}
	}
}

// Helper function to get the thread service easily
func (a *App) getThreadService() services.ThreadService {
	if a.logger != nil {
		a.logger.Printf("getThreadService: threadService=%v, dbStore=%v, aiService=%v, Client=%v", 
			a.threadService != nil, a.dbStore != nil, a.aiService != nil, a.Client != nil)
	}
	return a.threadService
}