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
	// Use dynamic configuration that accounts for numbers mode
	var config []render.ColumnConfig
	if mode == render.ModeThreaded {
		config = a.getColumnConfigForCurrentMode(render.RowTypeThreadHeader)
	} else {
		config = a.getColumnConfigForCurrentMode(render.RowTypeFlatMessage)
	}
	
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

// ResponsiveBreakpoint represents different screen size categories
type ResponsiveBreakpoint int

const (
	BreakpointVeryNarrow ResponsiveBreakpoint = iota // < 50 chars
	BreakpointNarrow                                 // 50-69 chars  
	BreakpointMedium                                 // 70-99 chars
	BreakpointWide                                   // 100+ chars
)

// getResponsiveBreakpoint determines the current responsive breakpoint based on available width
func (a *App) getResponsiveBreakpoint() ResponsiveBreakpoint {
	width := a.getListWidth()
	
	if width < 50 {
		return BreakpointVeryNarrow
	} else if width < 70 {
		return BreakpointNarrow
	} else if width < 100 {
		return BreakpointMedium
	}
	return BreakpointWide
}

// getColumnConfigForCurrentMode returns the appropriate responsive column configuration based on current display settings
func (a *App) getColumnConfigForCurrentMode(rowType render.EmailRowType) []render.ColumnConfig {
	breakpoint := a.getResponsiveBreakpoint()
	availableWidth := a.getListWidth()
	
	if rowType == render.RowTypeThreadHeader || rowType == render.RowTypeThreadMessage {
		return a.getResponsiveThreadedConfig(breakpoint, availableWidth)
	} else {
		return a.getResponsiveFlatConfig(breakpoint, availableWidth)
	}
}

// getResponsiveFlatConfig returns responsive column configuration for flat message lists
func (a *App) getResponsiveFlatConfig(breakpoint ResponsiveBreakpoint, availableWidth int) []render.ColumnConfig {
	config := make([]render.ColumnConfig, 0, 6) // Max possible columns with numbers
	
	// Column fixed and minimum widths
	flagsFixedWidth := 3      // Fixed width for flags column (●/○/!)
	fromMinWidth := 8
	subjectMinWidth := 15
	attachmentFixedWidth := 3 // Fixed width for attachment column (📎)
	calendarFixedWidth := 3   // Fixed width for calendar column (📅)
	dateMinWidth := 8
	numbersWidth := 0
	
	// If numbers are enabled, calculate numbers column width
	if a.showMessageNumbers {
		maxNumber := len(a.ids)
		numbersWidth = len(fmt.Sprintf("%d", maxNumber)) + 1 // +1 for spacing
		
		numbersColumn := render.ColumnConfig{
			Header:    "#",
			Alignment: tview.AlignRight,
			Expansion: 0,
			MaxWidth:  numbersWidth,
			MinWidth:  numbersWidth,
		}
		config = append(config, numbersColumn)
	}
	
	// Always include flags column (highest priority) - fixed width
	flagsColumn := render.ColumnConfig{
		Header:    "",
		Alignment: tview.AlignCenter,
		Expansion: 0,
		MaxWidth:  flagsFixedWidth,
		MinWidth:  flagsFixedWidth,
	}
	config = append(config, flagsColumn)
	
	// Calculate remaining width after fixed columns (numbers + flags)
	usedWidth := numbersWidth + flagsFixedWidth + 2 // +2 for separators
	remainingWidth := availableWidth - usedWidth
	
	// Responsive column inclusion based on breakpoint and available space
	switch breakpoint {
	case BreakpointVeryNarrow:
		// Minimal: Numbers (if enabled) + Flags + From (truncated) + Subject (truncated)
		if remainingWidth >= fromMinWidth + subjectMinWidth {
			fromWidth := fromMinWidth
			subjectWidth := remainingWidth - fromWidth - 2 // -2 for separator
			
			config = append(config, render.ColumnConfig{
				Header: "From", Alignment: tview.AlignLeft, Expansion: 0,
				MaxWidth: fromWidth, MinWidth: fromWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "Subject", Alignment: tview.AlignLeft, Expansion: 1,
				MaxWidth: subjectWidth, MinWidth: subjectMinWidth,
			})
		}
		
	case BreakpointNarrow:
		// Show: Numbers + Flags + From + Subject + Attachment + Calendar + Date (all columns, compact)
		totalIconsWidth := attachmentFixedWidth + calendarFixedWidth
		if remainingWidth >= fromMinWidth + subjectMinWidth + totalIconsWidth + dateMinWidth + 8 { // +8 for separators
			fromWidth := 12
			dateWidth := dateMinWidth
			subjectWidth := remainingWidth - fromWidth - totalIconsWidth - dateWidth - 8
			
			config = append(config, render.ColumnConfig{
				Header: "From", Alignment: tview.AlignLeft, Expansion: 0,
				MaxWidth: fromWidth, MinWidth: fromMinWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "Subject", Alignment: tview.AlignLeft, Expansion: 1,
				MaxWidth: subjectWidth, MinWidth: subjectMinWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "", Alignment: tview.AlignCenter, Expansion: 0,
				MaxWidth: attachmentFixedWidth, MinWidth: attachmentFixedWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "", Alignment: tview.AlignCenter, Expansion: 0,
				MaxWidth: calendarFixedWidth, MinWidth: calendarFixedWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "Date", Alignment: tview.AlignRight, Expansion: 0,
				MaxWidth: dateWidth, MinWidth: dateWidth,
			})
		}
		
	case BreakpointMedium:
		// Show: Numbers + Flags + From + Subject + Attachment + Calendar + Date (comfortable spacing)
		totalIconsWidth := attachmentFixedWidth + calendarFixedWidth
		if remainingWidth >= fromMinWidth + subjectMinWidth + totalIconsWidth + dateMinWidth + 8 { // +8 for separators
			fromWidth := 15
			dateWidth := 12
			subjectWidth := remainingWidth - fromWidth - totalIconsWidth - dateWidth - 8
			
			config = append(config, render.ColumnConfig{
				Header: "From", Alignment: tview.AlignLeft, Expansion: 0,
				MaxWidth: fromWidth, MinWidth: fromMinWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "Subject", Alignment: tview.AlignLeft, Expansion: 1,
				MaxWidth: subjectWidth, MinWidth: subjectMinWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "", Alignment: tview.AlignCenter, Expansion: 0,
				MaxWidth: attachmentFixedWidth, MinWidth: attachmentFixedWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "", Alignment: tview.AlignCenter, Expansion: 0,
				MaxWidth: calendarFixedWidth, MinWidth: calendarFixedWidth,
			})
			config = append(config, render.ColumnConfig{
				Header: "Date", Alignment: tview.AlignRight, Expansion: 0,
				MaxWidth: dateWidth, MinWidth: dateMinWidth,
			})
		}
		
	case BreakpointWide:
		// Show: Numbers + Flags + From + Subject + Attachment + Calendar + Date (generous spacing)
		// Use expansion weights for flexible columns, fixed width for icons
		config = append(config, render.ColumnConfig{
			Header: "From", Alignment: tview.AlignLeft, Expansion: 1,
			MaxWidth: 0, MinWidth: fromMinWidth,
		})
		config = append(config, render.ColumnConfig{
			Header: "Subject", Alignment: tview.AlignLeft, Expansion: 3,
			MaxWidth: 0, MinWidth: subjectMinWidth,
		})
		config = append(config, render.ColumnConfig{
			Header: "", Alignment: tview.AlignCenter, Expansion: 0,
			MaxWidth: attachmentFixedWidth, MinWidth: attachmentFixedWidth,
		})
		config = append(config, render.ColumnConfig{
			Header: "", Alignment: tview.AlignCenter, Expansion: 0,
			MaxWidth: calendarFixedWidth, MinWidth: calendarFixedWidth,
		})
		config = append(config, render.ColumnConfig{
			Header: "Date", Alignment: tview.AlignRight, Expansion: 0,
			MaxWidth: 16, MinWidth: dateMinWidth,
		})
	}
	
	return config
}

// getResponsiveThreadedConfig returns responsive column configuration for threaded view
func (a *App) getResponsiveThreadedConfig(breakpoint ResponsiveBreakpoint, availableWidth int) []render.ColumnConfig {
	// For now, use the base threaded configuration
	// TODO: Implement responsive threaded configuration similar to flat config
	baseConfig := render.GetColumnConfig(render.ModeThreaded)
	
	// Add numbers column if enabled
	if a.showMessageNumbers {
		maxNumber := len(a.ids)
		numberWidth := len(fmt.Sprintf("%d", maxNumber)) + 1
		
		numbersColumn := render.ColumnConfig{
			Header:    "#",
			Alignment: tview.AlignRight,
			Expansion: 0,
			MaxWidth:  numberWidth,
			MinWidth:  numberWidth,
		}
		
		// Prepend numbers column to the base configuration
		config := make([]render.ColumnConfig, 0, len(baseConfig)+1)
		config = append(config, numbersColumn)
		config = append(config, baseConfig...)
		return config
	}
	
	return baseConfig
}

// mapEmailDataToResponsiveColumns maps fixed email column data to responsive column configuration
func (a *App) mapEmailDataToResponsiveColumns(emailData render.EmailColumnData, config []render.ColumnConfig, rowIndex int) []render.ColumnCell {
	mappedColumns := make([]render.ColumnCell, len(config))
	
	// Source data indices (fixed structure from email renderer)
	const (
		SRC_FLAGS = 0
		SRC_FROM = 1  
		SRC_SUBJECT = 2
		SRC_ATTACHMENT = 3
		SRC_CALENDAR = 4
		SRC_DATE = 5
	)
	
	// Determine if numbers column is present in config (always first if present)
	configIndex := 0
	hasNumbers := len(config) > 0 && config[0].Header == "#"
	
	if hasNumbers {
		// Numbers column: create number content using the passed row index
		maxNumber := len(a.ids)
		width := len(fmt.Sprintf("%d", maxNumber))
		numberContent := fmt.Sprintf("%*d", width, rowIndex+1) // +1 to make it 1-based for display
		
		mappedColumns[configIndex] = render.ColumnCell{
			Content:   numberContent,
			Alignment: tview.AlignRight,
			MaxWidth:  width + 1,
			Expansion: 0,
		}
		configIndex++
	}
	
	// Track which empty-header columns we've seen (flags, then attachment, then calendar)
	flagsColumnSeen := false
	attachmentColumnSeen := false
	
	// Map remaining columns based on config headers and availability
	for configIndex < len(config) {
		configHeader := config[configIndex].Header
		
		switch configHeader {
		case "": // Either flags, attachment, or calendar column
			if config[configIndex].Alignment == tview.AlignCenter {
				if !flagsColumnSeen {
					// This is the first empty-header column - it's the flags column
					if len(emailData.Columns) > SRC_FLAGS {
						mappedColumns[configIndex] = emailData.Columns[SRC_FLAGS]
					}
					flagsColumnSeen = true
				} else if !attachmentColumnSeen {
					// This is the second empty-header column - it's the attachment column
					if len(emailData.Columns) > SRC_ATTACHMENT {
						mappedColumns[configIndex] = emailData.Columns[SRC_ATTACHMENT]
					}
					attachmentColumnSeen = true
				} else {
					// This is the third empty-header column - it's the calendar column
					if len(emailData.Columns) > SRC_CALENDAR {
						mappedColumns[configIndex] = emailData.Columns[SRC_CALENDAR]
					}
				}
			}
		case "From":
			if len(emailData.Columns) > SRC_FROM {
				mappedColumns[configIndex] = emailData.Columns[SRC_FROM]
			}
		case "Subject":
			if len(emailData.Columns) > SRC_SUBJECT {
				mappedColumns[configIndex] = emailData.Columns[SRC_SUBJECT]
			}
		case "Date":
			if len(emailData.Columns) > SRC_DATE {
				mappedColumns[configIndex] = emailData.Columns[SRC_DATE]
			}
		}
		
		// Apply responsive column configuration overrides
		if mappedColumns[configIndex].Content != "" {
			mappedColumns[configIndex].Alignment = config[configIndex].Alignment
			if config[configIndex].MaxWidth > 0 {
				mappedColumns[configIndex].MaxWidth = config[configIndex].MaxWidth
			}
			if config[configIndex].Expansion > 0 {
				mappedColumns[configIndex].Expansion = config[configIndex].Expansion
			}
		}
		
		configIndex++
	}
	
	return mappedColumns
}

// populateTableRow populates a single table row with the provided column data
func (a *App) populateTableRow(table *tview.Table, row int, data render.EmailColumnData) {
	config := a.getColumnConfigForCurrentMode(data.RowType)
	
	// Convert table row to message index (row - 1 for header)
	messageIndex := row - 1
	
	// Map email data to responsive column structure
	mappedColumns := a.mapEmailDataToResponsiveColumns(data, config, messageIndex)
	
	for col, cellData := range mappedColumns {
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
		// Note: tview.TableCell doesn't have SetMinWidth, width control is at table level
		
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
			// Show loading placeholder - responsive mapping will handle layout
			loadingData := render.EmailColumnData{
				RowType: render.RowTypeFlatMessage,
				Columns: []render.ColumnCell{
					{"○", tview.AlignCenter, 3, 0},           // Flags
					{"Loading...", tview.AlignLeft, 0, 1},    // From
					{"Loading message content...", tview.AlignLeft, 0, 3}, // Subject
					{"   ", tview.AlignCenter, 3, 0},         // Attachment (empty, 3 spaces)
					{"   ", tview.AlignCenter, 3, 0},         // Calendar (empty, 3 spaces)
					{"--", tview.AlignRight, 16, 0},          // Date
				},
				Color: a.currentTheme.UI.FooterColor.Color(),
			}
			a.populateTableRow(table, i+1, loadingData) // +1 for header row
			continue
		}
		
		msg := a.messagesMeta[i]
		columnData := a.emailRenderer.FormatFlatMessageColumns(msg)
		
		// Enhance flags column with bulk mode, preserving original status flags
		// The responsive mapping will handle numbers column and layout
		originalFlags := columnData.Columns[0].Content
		flags := a.buildEnhancedFlags(msg, i, originalFlags)
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

// buildEnhancedFlags builds the flags column content with bulk checkboxes and original status indicators
// Note: Numbers are now handled in a separate dedicated column, so this function no longer includes them
func (a *App) buildEnhancedFlags(msg *gmailapi.Message, index int, originalFlags string) string {
	var flags strings.Builder
	
	// Add bulk mode checkbox, but preserve original status flags
	if a.bulkMode {
		if a.selected != nil && a.selected[a.ids[index]] {
			flags.WriteString("☑")
		} else {
			flags.WriteString("☐")
		}
		// Add a space, then preserve original status flags (●/○/!)
		if originalFlags != "" {
			flags.WriteString(" ")
			flags.WriteString(originalFlags)
		}
	} else {
		// When not in bulk mode, just use the original status flags
		flags.WriteString(originalFlags)
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