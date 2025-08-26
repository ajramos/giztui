package tui

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/ajramos/gmail-tui/internal/render"
	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// getCurrentDisplayMode determines the current display mode
func (a *App) getCurrentDisplayMode() render.DisplayMode {
	if a.IsThreadingEnabled() && a.GetCurrentThreadViewMode() == ThreadViewThread {
		return render.ModeThreaded
	}
	return render.ModeFlatList
}

// configureTableForMode sets up the table structure for the specified display mode
func (a *App) configureTableForMode(table *tview.Table, mode render.DisplayMode) {
	config := render.GetColumnConfig(mode)
	
	// Clear existing table structure
	table.Clear()
	
	// Set table properties
	table.SetBorders(false).
		SetSeparator('│').
		SetFixed(1, 0). // Fix header row
		SetSelectable(true, false) // Allow row selection only
	
	// Create and populate header row
	for col, columnConfig := range config {
		cell := tview.NewTableCell(columnConfig.Header).
			SetSelectable(false).
			SetAlign(columnConfig.Alignment)
		
		if columnConfig.Expansion > 0 {
			cell.SetExpansion(columnConfig.Expansion)
		}
		if columnConfig.MaxWidth > 0 {
			cell.SetMaxWidth(columnConfig.MaxWidth)
		}
		
		table.SetCell(0, col, cell)
	}
}

// populateTableRow populates a single table row with the provided column data
func (a *App) populateTableRow(table *tview.Table, row int, data render.EmailColumnData) {
	var config []render.ColumnConfig
	if data.RowType == render.RowTypeThreadHeader || data.RowType == render.RowTypeThreadMessage {
		config = render.GetColumnConfig(render.ModeThreaded)
	} else {
		config = render.GetColumnConfig(render.ModeFlatList)
	}
	
	for col, cellData := range data.Columns {
		if col >= len(config) {
			continue // Skip extra columns
		}
		
		cell := tview.NewTableCell(cellData.Content).
			SetAlign(cellData.Alignment).
			SetTextColor(data.Color)
		
		// Apply column-specific settings from config
		columnConfig := config[col]
		if columnConfig.Expansion > 0 {
			cell.SetExpansion(columnConfig.Expansion)
		}
		if columnConfig.MaxWidth > 0 {
			cell.SetMaxWidth(columnConfig.MaxWidth)
		}
		
		// Override with cell-specific settings if provided
		if cellData.MaxWidth > 0 {
			cell.SetMaxWidth(cellData.MaxWidth)
		}
		if cellData.Expansion > 0 {
			cell.SetExpansion(cellData.Expansion)
		}
		
		table.SetCell(row, col, cell)
	}
}

// applyBulkModeStyle applies bulk selection styling to the table if in bulk mode
func (a *App) applyBulkModeStyle(table *tview.Table) {
	if !a.bulkMode {
		return
	}
	
	// Apply bulk selection styling to selected rows
	for row := 1; row < table.GetRowCount(); row++ { // Skip header row
		messageID := a.getRowMessageID(row - 1) // Adjust for header
		if a.selected[messageID] {
			// Apply bulk selection style to entire row
			for col := 0; col < table.GetColumnCount(); col++ {
				if cell := table.GetCell(row, col); cell != nil {
					cell.SetBackgroundColor(a.getBulkSelectionColor())
				}
			}
		}
	}
}

// getRowMessageID returns the message ID for a specific table row (0-based, excluding header)
func (a *App) getRowMessageID(row int) string {
	if row >= 0 && row < len(a.ids) {
		return a.ids[row]
	}
	return ""
}

// getCurrentSelectedMessageIndex returns the current selected message index (0-based, excluding header)
// Returns -1 if no valid selection
func (a *App) getCurrentSelectedMessageIndex() int {
	table, ok := a.views["list"].(*tview.Table)
	if !ok {
		return -1
	}
	
	selectedRow, _ := table.GetSelection()
	if selectedRow <= 0 { // 0 is header row, so <= 0 means no valid message selected
		return -1
	}
	
	// Convert table row to message index (subtract 1 for header)
	messageIndex := selectedRow - 1
	if messageIndex >= len(a.ids) {
		return -1
	}
	
	return messageIndex
}

// getCurrentSelectedMessageID returns the current selected message ID
// Returns empty string if no valid selection
func (a *App) getCurrentSelectedMessageID() string {
	messageIndex := a.getCurrentSelectedMessageIndex()
	if messageIndex < 0 {
		return ""
	}
	return a.ids[messageIndex]
}

// getBulkSelectionColor returns the background color for bulk-selected rows
func (a *App) getBulkSelectionColor() tcell.Color {
	// Use a darker background for selected items in bulk mode
	return tcell.ColorDarkBlue
}

// refreshTableDisplay refreshes the entire table display based on current mode and data
func (a *App) refreshTableDisplay() {
	table, ok := a.views["list"].(*tview.Table)
	if !ok {
		return
	}
	
	mode := a.getCurrentDisplayMode()
	
	// Configure table structure for current mode
	a.configureTableForMode(table, mode)
	
	// Populate rows based on mode
	switch mode {
	case render.ModeFlatList:
		a.populateFlatRows(table)
	case render.ModeThreaded:
		a.populateThreadedRows(table)
	}
	
	// Apply bulk mode styling if active
	a.applyBulkModeStyle(table)
}

// populateFlatRows populates the table with flat message list data
func (a *App) populateFlatRows(table *tview.Table) {
	for i := 0; i < len(a.ids); i++ {
		if i >= len(a.messagesMeta) || a.messagesMeta[i] == nil {
			// Show loading placeholder
			loadingData := render.EmailColumnData{
				RowType: render.RowTypeFlatMessage,
				Columns: []render.ColumnCell{
					{"○", tview.AlignCenter, 3, 0},
					{"Loading...", tview.AlignLeft, 0, 1},
					{"Loading message content...", tview.AlignLeft, 0, 3},
					{"--", tview.AlignRight, 16, 0},
				},
				Color: a.currentTheme.UI.FooterColor.Color(),
			}
			a.populateTableRow(table, i+1, loadingData) // +1 for header row
			continue
		}
		
		msg := a.messagesMeta[i]
		columnData := a.emailRenderer.FormatFlatMessageColumns(msg)
		
		// Enhance flags column with bulk mode and message numbering
		flags := a.buildEnhancedFlags(msg, i)
		columnData.Columns[0].Content = flags
		
		// Apply bulk mode styling if this message is selected
		if a.bulkMode && a.selected != nil && a.selected[a.ids[i]] {
			// Apply bulk selection styling
			cur, _ := table.GetSelection()
			if cur == i+1 { // +1 for header
				// Keep normal colors for focused selection
				columnData.Color = columnData.Color
			} else {
				// Use bulk selection background
				columnData.Color = a.currentTheme.Body.BgColor.Color()
			}
		}
		
		a.populateTableRow(table, i+1, columnData) // +1 for header row
		
		// Apply bulk mode background styling if needed
		if a.bulkMode && a.selected != nil && a.selected[a.ids[i]] {
			cur, _ := table.GetSelection()
			if cur != i+1 { // Not currently focused
				for col := 0; col < table.GetColumnCount(); col++ {
					if cell := table.GetCell(i+1, col); cell != nil {
						cell.SetBackgroundColor(a.currentTheme.UI.SelectionFgColor.Color())
					}
				}
			}
		}
	}
}

// buildEnhancedFlags builds the flags column content including message numbers, bulk checkboxes, and status
func (a *App) buildEnhancedFlags(msg *gmailapi.Message, index int) string {
	var flags strings.Builder
	
	// Add message number if enabled (leftmost position)
	if a.showMessageNumbers {
		maxNumber := len(a.ids)
		width := len(fmt.Sprintf("%d", maxNumber))
		flags.WriteString(fmt.Sprintf("%*d ", width, index+1)) // Right-aligned numbering
	}
	
	// Add bulk mode checkbox
	if a.bulkMode {
		if a.selected != nil && a.selected[a.ids[index]] {
			flags.WriteString("☑")
		} else {
			flags.WriteString("☐")
		}
	} else {
		// Add read/unread indicator when not in bulk mode
		if a.emailRenderer.IsUnread(msg) {
			flags.WriteString("●")
		} else {
			flags.WriteString("○")
		}
		
		// Add important indicator
		if a.emailRenderer.IsImportant(msg) {
			flags.WriteString("!")
		}
	}
	
	return flags.String()
}

// FormatThreadHeaderColumns formats a thread header for column display
func (a *App) FormatThreadHeaderColumns(thread *services.ThreadInfo, index int, isExpanded bool) render.EmailColumnData {
	if thread == nil {
		return render.EmailColumnData{
			RowType: render.RowTypeThreadHeader,
			Columns: []render.ColumnCell{
				{"○", tview.AlignLeft, 8, 0},
				{"", tview.AlignRight, 6, 0},
				{"(No thread)", tview.AlignLeft, 0, 1},
				{"(No subject)", tview.AlignLeft, 0, 3},
				{"--", tview.AlignRight, 16, 0},
			},
			Color: a.currentTheme.UI.FooterColor.Color(),
		}
	}

	// Build thread icon and status
	var threadIcon strings.Builder
	
	// Add message number if enabled
	if a.showMessageNumbers {
		maxNumber := len(a.ids) // Approximate based on current view
		width := len(fmt.Sprintf("%d", maxNumber))
		threadIcon.WriteString(fmt.Sprintf("%*d ", width, index+1))
	}
	
	// Thread expansion indicator
	if thread.MessageCount > 1 {
		if isExpanded {
			threadIcon.WriteString("▼️ ")
		} else {
			threadIcon.WriteString("▶️ ")
		}
	} else {
		threadIcon.WriteString("📧 ")
	}
	
	// Unread indicator
	if thread.UnreadCount > 0 {
		threadIcon.WriteString("● ")
	} else {
		threadIcon.WriteString("○ ")
	}

	// Format thread count
	countText := fmt.Sprintf("[%d]", thread.MessageCount)

	// Get primary participant
	var senderName string
	if len(thread.Participants) > 0 {
		senderName = thread.Participants[0]
	} else {
		senderName = "(No sender)"
	}

	// Build subject with attachments
	subject := thread.Subject
	if subject == "" {
		subject = "(No subject)"
	}
	if thread.HasAttachment {
		subject += " 📎"
	}

	// Format date
	dateStr := a.formatThreadDate(thread.LatestDate)

	// Determine thread color based on unread count
	var color tcell.Color
	if thread.UnreadCount > 0 {
		color = a.currentTheme.UI.InfoColor.Color()
	} else {
		color = a.currentTheme.UI.FooterColor.Color()
	}

	return render.EmailColumnData{
		RowType: render.RowTypeThreadHeader,
		Columns: []render.ColumnCell{
			{threadIcon.String(), tview.AlignLeft, 8, 0},
			{countText, tview.AlignRight, 6, 0},
			{senderName, tview.AlignLeft, 0, 1},
			{subject, tview.AlignLeft, 0, 3},
			{dateStr, tview.AlignRight, 16, 0},
		},
		Color: color,
	}
}

// FormatThreadMessageColumns formats an individual thread message for column display
func (a *App) FormatThreadMessageColumns(message *gmailapi.Message, treePrefix string) render.EmailColumnData {
	if message == nil || message.Payload == nil {
		return render.EmailColumnData{
			RowType: render.RowTypeThreadMessage,
			Columns: []render.ColumnCell{
				{treePrefix, tview.AlignLeft, 8, 0},
				{"", tview.AlignRight, 6, 0},
				{"(No message)", tview.AlignLeft, 0, 1},
				{"(No subject)", tview.AlignLeft, 0, 3},
				{"--", tview.AlignRight, 16, 0},
			},
			Color: a.currentTheme.UI.FooterColor.Color(),
		}
	}

	// Build tree structure with message icon and status
	var treeIcon strings.Builder
	treeIcon.WriteString(treePrefix) // "    ├─ " or "    └─ "
	treeIcon.WriteString("📧 ")
	
	// Add unread indicator
	if a.emailRenderer.IsUnread(message) {
		treeIcon.WriteString("● ")
	} else {
		treeIcon.WriteString("○ ")
	}

	// Extract sender
	senderName := a.emailRenderer.ExtractSenderName(a.emailRenderer.GetHeader(message, "From"))
	if senderName == "" {
		senderName = "(No sender)"
	}

	// Extract subject
	subject := a.emailRenderer.GetHeader(message, "Subject")
	if subject == "" {
		subject = "(No subject)"
	}

	// Format date
	dateStr := a.formatThreadDate(a.emailRenderer.GetDate(message))

	// Determine message color
	color := a.emailRenderer.GetMessageColor(message)

	return render.EmailColumnData{
		RowType: render.RowTypeThreadMessage,
		Columns: []render.ColumnCell{
			{treeIcon.String(), tview.AlignLeft, 8, 0},
			{"", tview.AlignRight, 6, 0}, // Empty count cell for individual messages
			{senderName, tview.AlignLeft, 0, 1},
			{subject, tview.AlignLeft, 0, 3},
			{dateStr, tview.AlignRight, 16, 0},
		},
		Color: color,
	}
}

// formatThreadDate formats a date for thread display
func (a *App) formatThreadDate(date time.Time) string {
	now := time.Now()
	
	if date.After(now.Add(-24 * time.Hour)) {
		return date.Format("3:04 PM")
	} else if date.After(now.Add(-7 * 24 * time.Hour)) {
		return date.Format("Mon 3:04 PM")
	} else if date.Year() == now.Year() {
		return date.Format("Jan 02")
	} else {
		return date.Format("2006")
	}
}

// populateThreadedRows populates the table with threaded conversation data
func (a *App) populateThreadedRows(table *tview.Table) {
	// Get threads from current state
	// This assumes threads are stored in a similar way to flat messages
	// For now, this is a simplified version - the full implementation will be done
	// when we fully replace displayThreadsSync
	
	// First, we need to access the thread data. Since this is complex,
	// let's create a helper method to get current threads and fall back for now
	threads := a.getCurrentThreads()
	if threads == nil {
		// Fall back to flat mode if no thread data available
		a.populateFlatRows(table)
		return
	}
	
	rowIndex := 1 // Start after header row
	
	// Process each thread
	for i, thread := range threads {
		if thread == nil {
			continue
		}
		
		// Check if thread is expanded
		isExpanded := a.isThreadExpanded(thread.ThreadID)
		
		// Create and populate thread header row
		threadData := a.FormatThreadHeaderColumns(thread, i, isExpanded)
		a.populateTableRow(table, rowIndex, threadData)
		rowIndex++
		
		// If thread is expanded, add individual message rows
		if isExpanded && thread.MessageCount > 1 {
			messages, err := a.fetchThreadMessages(a.ctx, thread.ThreadID)
			if err != nil {
				// Add error row
				errorData := render.EmailColumnData{
					RowType: render.RowTypeThreadMessage,
					Columns: []render.ColumnCell{
						{"    ⚠️ ", tview.AlignLeft, 8, 0},
						{"", tview.AlignRight, 6, 0},
						{"Failed to load messages", tview.AlignLeft, 0, 1},
						{"", tview.AlignLeft, 0, 3},
						{"--", tview.AlignRight, 16, 0},
					},
					Color: tcell.ColorOrange,
				}
				a.populateTableRow(table, rowIndex, errorData)
				rowIndex++
			} else {
				// Add individual message rows with tree structure
				for msgIndex, message := range messages {
					// Determine tree prefix
					var treePrefix string
					if msgIndex == len(messages)-1 {
						treePrefix = "    └─ " // Last message
					} else {
						treePrefix = "    ├─ " // Intermediate message
					}
					
					messageData := a.FormatThreadMessageColumns(message, treePrefix)
					a.populateTableRow(table, rowIndex, messageData)
					rowIndex++
				}
			}
		}
	}
}

// getCurrentThreads gets the current thread list for display
func (a *App) getCurrentThreads() []*services.ThreadInfo {
	// Use the stored thread data from displayThreadsSync
	a.mu.RLock()
	threads := a.currentThreads
	a.mu.RUnlock()
	
	return threads
}

// isThreadExpanded checks if a thread is currently expanded
func (a *App) isThreadExpanded(threadID string) bool {
	threadService := a.getThreadService()
	if threadService == nil {
		return false
	}
	
	accountEmail, _ := a.Client.ActiveAccountEmail(a.ctx)
	if accountEmail == "" {
		return false
	}
	
	isExpanded, _ := threadService.IsThreadExpanded(a.ctx, accountEmail, threadID)
	return isExpanded
}