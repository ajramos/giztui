package render

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ajramos/giztui/internal/config"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/mattn/go-runewidth"
	googleGmail "google.golang.org/api/gmail/v1"
)

// EmailColorer maneja los colores de emails
type EmailColorer struct {
	UnreadColor    tcell.Color
	ReadColor      tcell.Color
	ImportantColor tcell.Color
	SentColor      tcell.Color
	DraftColor     tcell.Color
	DefaultColor   tcell.Color
}

// NewEmailColorer creates a new email colorer with theme-aware colors
func NewEmailColorer(unreadColor, readColor, importantColor, sentColor, draftColor, defaultColor tcell.Color) *EmailColorer {
	return &EmailColorer{
		UnreadColor:    unreadColor,
		ReadColor:      readColor,
		ImportantColor: importantColor,
		SentColor:      sentColor,
		DraftColor:     draftColor,
		DefaultColor:   defaultColor,
	}
}

// NewEmailColorerDefault creates a new email colorer with default fallback colors
func NewEmailColorerDefault() *EmailColorer {
	return &EmailColorer{
		UnreadColor:    tcell.ColorOrange,
		ReadColor:      tcell.ColorGray,
		ImportantColor: tcell.ColorRed,
		SentColor:      tcell.ColorGreen,
		DraftColor:     tcell.ColorYellow,
		DefaultColor:   tcell.ColorWhite,
	}
}

// ColorerFunc returns a colorer function for emails
func (ec *EmailColorer) ColorerFunc() func(*googleGmail.Message, string) tcell.Color {
	return func(message *googleGmail.Message, column string) tcell.Color {
		switch strings.ToUpper(column) {
		case "STATUS":
			if ec.isUnread(message) {
				return ec.UnreadColor // blue for unread
			}
			return ec.ReadColor // gray for read

		case "FROM":
			if ec.isImportant(message) {
				return ec.ImportantColor // 🔴 Rojo para importante
			}
			if ec.isUnread(message) {
				return ec.UnreadColor // orange for unread
			}
			return ec.DefaultColor

		case "SUBJECT":
			if ec.isDraft(message) {
				return ec.DraftColor // 🟡 Amarillo para borrador
			}
			if ec.isSent(message) {
				return ec.SentColor // 🟢 Verde para enviado
			}
			if ec.isUnread(message) {
				return ec.DefaultColor // ⚪ Blanco brillante
			}
			return ec.ReadColor // gray for read
		}
		return ec.DefaultColor
	}
}

// UpdateFromStyles updates colors from configuration using v2.0 hierarchical themes
func (ec *EmailColorer) UpdateFromStyles(colors *config.ColorsConfig) {
	// Map email classification to hierarchical color system
	// Using semantic colors for meaningful message states
	ec.UnreadColor = colors.Semantic.Accent.Color()     // Cyan/blue for attention (unread)
	ec.ReadColor = colors.Foundation.Foreground.Color() // Default text color (read)
	ec.ImportantColor = colors.Semantic.Warning.Color() // Orange/yellow for importance
	ec.SentColor = colors.Semantic.Success.Color()      // Green for sent items
	ec.DraftColor = colors.Semantic.Secondary.Color()   // Gray for drafts
}

// Helper methods to determine email state
func (ec *EmailColorer) isUnread(message *googleGmail.Message) bool {
	// Check if message has UNREAD label
	for _, labelId := range message.LabelIds {
		if labelId == "UNREAD" {
			return true
		}
	}
	return false
}

func (ec *EmailColorer) isImportant(message *googleGmail.Message) bool {
	// Check for important labels
	importantLabels := []string{"IMPORTANT", "PRIORITY", "URGENT"}
	for _, labelId := range message.LabelIds {
		for _, important := range importantLabels {
			if strings.Contains(strings.ToUpper(labelId), important) {
				return true
			}
		}
	}
	return false
}

func (ec *EmailColorer) isDraft(message *googleGmail.Message) bool {
	for _, labelId := range message.LabelIds {
		if strings.ToUpper(labelId) == "DRAFT" {
			return true
		}
	}
	return false
}

func (ec *EmailColorer) isSent(message *googleGmail.Message) bool {
	for _, labelId := range message.LabelIds {
		if strings.ToUpper(labelId) == "SENT" {
			return true
		}
	}
	return false
}

// EmailRowType represents the type of row being displayed
type EmailRowType int

const (
	RowTypeFlatMessage EmailRowType = iota
	RowTypeThreadHeader
	RowTypeThreadMessage
	RowTypeHeader
)

// ColumnCell represents individual cell data within a table row
type ColumnCell struct {
	Content   string
	Alignment int // tview alignment constant
	MaxWidth  int // 0 = unlimited
	Expansion int // Weight for width distribution
}

// EmailColumnData represents structured data for any display mode
type EmailColumnData struct {
	Columns []ColumnCell
	RowType EmailRowType
	Color   tcell.Color
}

// ColumnConfig defines column behavior and styling
type ColumnConfig struct {
	Header    string
	Alignment int // tview alignment constant
	Expansion int // Weight for extra width distribution
	MaxWidth  int // 0 = unlimited
	MinWidth  int // Minimum guaranteed width
}

// DisplayMode represents different email list display modes
type DisplayMode int

const (
	ModeFlatList DisplayMode = iota
	ModeThreaded
)

// EmailRenderer handles email rendering and formatting
type EmailRenderer struct {
	colorer      *EmailColorer
	headerKeyTag string // e.g., "[#50fa7b]"
	// Optional label mapping and flags for list rendering enhancements
	labelIdToName          map[string]string
	showSystemLabelsInList bool
	config                 *config.Config
}

// NewEmailRenderer creates a new email renderer
func NewEmailRenderer(cfg *config.Config) *EmailRenderer {
	return &EmailRenderer{
		colorer:                NewEmailColorerDefault(),
		headerKeyTag:           "[yellow]",
		labelIdToName:          make(map[string]string),
		showSystemLabelsInList: false,
		config:                 cfg,
	}
}

// SetLabelMap sets a map of label ID -> label Name used when rendering list chips
func (er *EmailRenderer) SetLabelMap(m map[string]string) {
	if m == nil {
		er.labelIdToName = make(map[string]string)
		return
	}
	er.labelIdToName = m
}

// SetShowSystemLabelsInList toggles whether system labels (Inbox, Sent, Spam, etc.)
// should be rendered as chips in the list view.
func (er *EmailRenderer) SetShowSystemLabelsInList(v bool) { er.showSystemLabelsInList = v }

// UpdateColorer updates the email colorer with theme-aware colors
func (er *EmailRenderer) UpdateColorer(unreadColor, readColor, importantColor, sentColor, draftColor, defaultColor tcell.Color) {
	er.colorer = NewEmailColorer(unreadColor, readColor, importantColor, sentColor, draftColor, defaultColor)
}

// FormatEmailList formats an email for list display
func (er *EmailRenderer) FormatEmailList(message *googleGmail.Message, maxWidth int) (string, tcell.Color) {
	// colorer not used in the simple version

	// Extract sender name
	senderName := er.extractSenderName(er.getHeader(message, "From"))
	if senderName == "" {
		senderName = "(No sender)"
	}

	// Extract subject
	subject := er.getHeader(message, "Subject")
	if subject == "" {
		subject = "(No subject)"
	}

	// Format date
	date := er.formatRelativeTime(er.getDate(message))

	// Fixed widths with padding for alignment
	// Keep a minimum width for usability
	if maxWidth < 40 {
		maxWidth = 40
	}
	senderWidth := 22
	dateWidth := 8
	// Remaining for subject minus suffix width (icons + chips)
	suffix := er.buildIconsAndChips(message)
	suffixWidth := runewidth.StringWidth(suffix)
	// account for separators and spaces (" | ", " | ") = 6
	subjectWidth := maxWidth - senderWidth - dateWidth - 6 - suffixWidth
	if subjectWidth < 10 {
		subjectWidth = 10
	}

	senderText := er.fitWidth(senderName, senderWidth)
	subjectText := er.fitWidth(subject, subjectWidth)
	// Date at the end, left aligned
	dateText := er.fitWidth(date, dateWidth)

	// Create formatted string with fixed columns: Sender | Subject(+suffix) | Date
	formatted := fmt.Sprintf("%s | %s%s | %s", senderText, subjectText, suffix, dateText)

	// Devolvemos color neutro para simplificar (sin estilos)
	textColor := er.colorer.DefaultColor

	return formatted, textColor
}

// FormatFlatMessageColumns formats a message for flat list display using columns
func (er *EmailRenderer) FormatFlatMessageColumns(message *googleGmail.Message) EmailColumnData {
	if message == nil || message.Payload == nil {
		return EmailColumnData{
			RowType: RowTypeFlatMessage,
			Columns: []ColumnCell{
				{"○", tview.AlignCenter, 3, 0},
				{"(No sender)", tview.AlignLeft, 0, 1},
				{"(No subject)", tview.AlignLeft, 0, 3},
				{"", tview.AlignLeft, 16, 1},  // Empty labels column
				{"", tview.AlignCenter, 2, 0}, // Empty attachment column
				{"", tview.AlignCenter, 2, 0}, // Empty calendar column
				{"--", tview.AlignRight, 16, 0},
			},
			Color: er.colorer.DefaultColor,
		}
	}

	// Extract message flags (unread, important)
	flags := er.extractMessageFlags(message)

	// Extract and format sender
	senderName := er.extractSenderName(er.getHeader(message, "From"))
	if senderName == "" {
		senderName = "(No sender)"
	}

	// Extract subject
	subject := er.getHeader(message, "Subject")
	if subject == "" {
		subject = "(No subject)"
	}

	// Extract separate labels (no longer embedded in subject)
	labels := er.FormatLabelsForColumn(message, 16) // Default width, will be adjusted by responsive system

	// Extract separate attachment and calendar icons
	attachmentIcon := er.ExtractAttachmentIcon(message)
	calendarIcon := er.ExtractCalendarIcon(message)

	// Format date
	date := er.formatRelativeTime(er.getDate(message))

	// Determine row color
	color := er.getMessageColor(message)

	return EmailColumnData{
		RowType: RowTypeFlatMessage,
		Columns: []ColumnCell{
			{flags, tview.AlignCenter, 3, 0},
			{senderName, tview.AlignLeft, 0, 1},
			{subject, tview.AlignLeft, 0, 3}, // Clean subject without labels
			{labels, tview.AlignLeft, 16, 1}, // Dedicated labels column
			{attachmentIcon, tview.AlignCenter, 2, 0},
			{calendarIcon, tview.AlignCenter, 2, 0},
			{date, tview.AlignRight, 16, 0},
		},
		Color: color,
	}
}

// extractMessageFlags returns status flags for a message (●/○ for unread/read, ! for important)
func (er *EmailRenderer) extractMessageFlags(message *googleGmail.Message) string {
	var flags strings.Builder

	// Unread indicator
	if er.colorer.isUnread(message) {
		flags.WriteString("●")
	} else {
		flags.WriteString("○")
	}

	// Important indicator
	if er.colorer.isImportant(message) {
		flags.WriteString("!")
	}

	return flags.String()
}

// getMessageColor returns the appropriate color for a message based on its state
func (er *EmailRenderer) getMessageColor(message *googleGmail.Message) tcell.Color {
	if er.colorer.isImportant(message) {
		return er.colorer.ImportantColor
	}
	if er.colorer.isDraft(message) {
		return er.colorer.DraftColor
	}
	if er.colorer.isSent(message) {
		return er.colorer.SentColor
	}
	if er.colorer.isUnread(message) {
		return er.colorer.UnreadColor
	}
	return er.colorer.ReadColor
}

// IsUnread checks if a message is unread
func (er *EmailRenderer) IsUnread(message *googleGmail.Message) bool {
	return er.colorer.isUnread(message)
}

// IsImportant checks if a message is important
func (er *EmailRenderer) IsImportant(message *googleGmail.Message) bool {
	return er.colorer.isImportant(message)
}

// IsDraft checks if a message is a draft
func (er *EmailRenderer) IsDraft(message *googleGmail.Message) bool {
	return er.colorer.isDraft(message)
}

// IsSent checks if a message is sent
func (er *EmailRenderer) IsSent(message *googleGmail.Message) bool {
	return er.colorer.isSent(message)
}

// GetHeader extracts a header value from a message
func (er *EmailRenderer) GetHeader(message *googleGmail.Message, name string) string {
	return er.getHeader(message, name)
}

// ExtractSenderName extracts the sender name from a From header
func (er *EmailRenderer) ExtractSenderName(from string) string {
	return er.extractSenderName(from)
}

// GetDate extracts the date from a message
func (er *EmailRenderer) GetDate(message *googleGmail.Message) time.Time {
	return er.getDate(message)
}

// GetMessageColor returns the appropriate color for a message based on its state
func (er *EmailRenderer) GetMessageColor(message *googleGmail.Message) tcell.Color {
	return er.getMessageColor(message)
}

// GetColumnConfig returns column configuration for the specified display mode
func GetColumnConfig(mode DisplayMode) []ColumnConfig {
	switch mode {
	case ModeFlatList:
		return []ColumnConfig{
			{"", tview.AlignCenter, 0, 3, 2},       // Flags: ●○!
			{"From", tview.AlignLeft, 1, 0, 15},    // From: expand weight 1
			{"Subject", tview.AlignLeft, 3, 0, 20}, // Subject: expand weight 3
			{"", tview.AlignCenter, 0, 4, 2},       // Icons: 📎🗓️
			{"Date", tview.AlignRight, 0, 16, 8},   // Date: fixed max width
		}
	case ModeThreaded:
		return []ColumnConfig{
			{"", tview.AlignLeft, 0, 3, 3},         // Type: Thread/message icons only (▼️/▶️/📧) - increased to 3
			{"#", tview.AlignRight, 0, 6, 3},       // Thread Count: [4] or empty
			{"", tview.AlignCenter, 0, 3, 2},       // Status: Read/unread only (●/○)
			{"From", tview.AlignLeft, 1, 0, 15},    // From: expand weight 1
			{"Subject", tview.AlignLeft, 3, 0, 20}, // Subject: expand weight 3
			{"", tview.AlignCenter, 0, 2, 2},       // Attachment: 📎
			{"", tview.AlignCenter, 0, 2, 2},       // Calendar: 📅
			{"Date", tview.AlignRight, 0, 16, 8},   // Date: fixed max width
		}
	default:
		// Fallback to flat list config
		return GetColumnConfig(ModeFlatList)
	}
}

// ExtractAttachmentIcon returns attachment icon (📎) padded to 2 characters
func (er *EmailRenderer) ExtractAttachmentIcon(message *googleGmail.Message) string {
	if message == nil || message.Payload == nil {
		return "  " // 2 spaces
	}

	hasAttachment := false
	var walk func(p *googleGmail.MessagePart)
	walk = func(p *googleGmail.MessagePart) {
		if p == nil {
			return
		}
		if p.Body != nil && p.Body.AttachmentId != "" {
			hasAttachment = true
		}
		if p.Filename != "" {
			hasAttachment = true
		}
		for _, c := range p.Parts {
			walk(c)
		}
	}
	walk(message.Payload)

	if hasAttachment {
		return "📎" // Icon only for 2 total characters
	}
	return "  " // 2 spaces
}

// ExtractCalendarIcon returns calendar icon (📅) padded to 2 characters
func (er *EmailRenderer) ExtractCalendarIcon(message *googleGmail.Message) string {
	if message == nil || message.Payload == nil {
		return "  " // 2 spaces
	}

	hasCalendar := false
	var walk func(p *googleGmail.MessagePart)
	walk = func(p *googleGmail.MessagePart) {
		if p == nil {
			return
		}
		mt := strings.ToLower(p.MimeType)
		if p.Filename != "" {
			if strings.HasSuffix(strings.ToLower(p.Filename), ".ics") {
				hasCalendar = true
			}
		}
		if strings.Contains(mt, "text/calendar") || strings.Contains(mt, "application/ics") {
			hasCalendar = true
		}
		for _, c := range p.Parts {
			walk(c)
		}
	}
	walk(message.Payload)

	if hasCalendar {
		return "📅" // Icon only for 2 total characters
	}
	return "  " // 2 spaces
}

// OBLITERATED: buildLabelChips function eliminated! 💥

// FormatLabelsForColumn formats labels specifically for dedicated column display
// Returns labels formatted for the given available width with intelligent truncation
func (er *EmailRenderer) FormatLabelsForColumn(message *googleGmail.Message, maxWidth int) string {
	if message == nil || maxWidth <= 0 {
		return ""
	}

	// Extract label names using same logic as buildLabelChips
	names := make([]string, 0, len(message.LabelIds))
	for _, id := range message.LabelIds {
		name := id
		if n, ok := er.labelIdToName[id]; ok && strings.TrimSpace(n) != "" {
			name = n
		}
		upperID := strings.ToUpper(id)
		upperName := strings.ToUpper(name)
		// Skip state/importance labels (represented via colors)
		isStarVariant := strings.HasSuffix(upperID, "_STAR") || strings.HasSuffix(upperID, "_STARRED") || strings.HasSuffix(upperName, "_STAR") || strings.HasSuffix(upperName, "_STARRED")
		if upperID == "UNREAD" || upperID == "STARRED" || upperID == "IMPORTANT" || upperName == "UNREAD" || upperName == "STARRED" || upperName == "IMPORTANT" || isStarVariant {
			continue
		}
		// General system labels (Inbox/Sent/Trash/Spam/Draft/Category_*)
		isSystemGeneral := strings.HasPrefix(upperID, "CATEGORY_") || upperID == "INBOX" || upperID == "CHAT" || upperID == "SENT" || upperID == "TRASH" || upperID == "SPAM" || upperID == "DRAFT"
		if isSystemGeneral && !er.showSystemLabelsInList {
			continue
		}
		// Normalize display name (Category_* → friendly name; Title Case otherwise)
		names = append(names, normalizeLabelDisplay(name, id))
	}

	if len(names) == 0 {
		return ""
	}

	// Responsive formatting based on available width
	var b strings.Builder

	// Very narrow: skip labels entirely
	if maxWidth < 8 {
		return ""
	}

	// Narrow: compact format without spaces
	if maxWidth < 16 {
		// Format: [Lbl1][Lbl2][+N] - compact, no spaces
		totalUsed := 0
		labelsShown := 0
		for i, name := range names {
			// Truncate label name if needed
			labelName := name
			if len(labelName) > 4 {
				labelName = labelName[:4] // Truncate to 4 chars
			}
			labelText := "[" + labelName + "]"

			// Check if adding this label would exceed width
			overflowText := ""
			if i < len(names)-1 {
				remaining := len(names) - i - 1
				if remaining > 0 {
					overflowText = fmt.Sprintf("[+%d]", remaining)
				}
			}

			if totalUsed+len(labelText)+len(overflowText) > maxWidth {
				// Add overflow indicator if needed
				if labelsShown == 0 && totalUsed+5 <= maxWidth { // [+N] = at least 4 chars
					b.WriteString(fmt.Sprintf("[+%d]", len(names)))
				}
				break
			}

			b.WriteString(labelText)
			totalUsed += len(labelText)
			labelsShown++
		}
		return b.String()
	}

	// Medium/Wide: standard format with spaces
	// Format: [Label1] [Label2] [+N] - standard, with spaces
	totalUsed := 0
	labelsShown := 0
	for i, name := range names {
		labelText := "[" + name + "]"
		if i > 0 {
			labelText = " " + labelText // Add space before subsequent labels
		}

		// Check if adding this label would exceed width
		overflowText := ""
		if i < len(names)-1 {
			remaining := len(names) - i - 1
			if remaining > 0 {
				overflowText = fmt.Sprintf(" [+%d]", remaining)
			}
		}

		if totalUsed+len(labelText)+len(overflowText) > maxWidth {
			// Add overflow indicator if we have space
			if labelsShown > 0 {
				remaining := len(names) - labelsShown
				if remaining > 0 {
					overflowIndicator := fmt.Sprintf(" [+%d]", remaining)
					if totalUsed+len(overflowIndicator) <= maxWidth {
						b.WriteString(overflowIndicator)
					}
				}
			} else if totalUsed+5 <= maxWidth { // At least space for [+N]
				b.WriteString(fmt.Sprintf("[+%d]", len(names)))
			}
			break
		}

		b.WriteString(labelText)
		totalUsed += len(labelText)
		labelsShown++
	}

	return b.String()
}

// buildIconsAndChips returns a string like "  📎🗓️  [Aws] [Finance] [+2]" (legacy function, kept for compatibility)
func (er *EmailRenderer) buildIconsAndChips(message *googleGmail.Message) string {
	if message == nil || message.Payload == nil {
		return ""
	}
	// Detect attachments and calendar from MIME structure (metadata only)
	hasAttachment := false
	hasCalendar := false
	var walk func(p *googleGmail.MessagePart)
	walk = func(p *googleGmail.MessagePart) {
		if p == nil {
			return
		}
		mt := strings.ToLower(p.MimeType)
		if p.Body != nil && p.Body.AttachmentId != "" {
			// treat any attachment as a real attachment; filename strengthens signal but is optional
			hasAttachment = true
		}
		if p.Filename != "" {
			hasAttachment = true
			if strings.HasSuffix(strings.ToLower(p.Filename), ".ics") {
				hasCalendar = true
			}
		}
		if strings.Contains(mt, "text/calendar") || strings.Contains(mt, "application/ics") {
			hasCalendar = true
		}
		for _, c := range p.Parts {
			walk(c)
		}
	}
	walk(message.Payload)

	// Labels as chips (limit to 3 + +N). Use ID->Name map when available
	names := make([]string, 0, len(message.LabelIds))
	for _, id := range message.LabelIds {
		name := id
		if n, ok := er.labelIdToName[id]; ok && strings.TrimSpace(n) != "" {
			name = n
		}
		upperID := strings.ToUpper(id)
		upperName := strings.ToUpper(name)
		// Always skip state/importance labels (represented via colors)
		isStarVariant := strings.HasSuffix(upperID, "_STAR") || strings.HasSuffix(upperID, "_STARRED") || strings.HasSuffix(upperName, "_STAR") || strings.HasSuffix(upperName, "_STARRED")
		if upperID == "UNREAD" || upperID == "STARRED" || upperID == "IMPORTANT" || upperName == "UNREAD" || upperName == "STARRED" || upperName == "IMPORTANT" || isStarVariant {
			continue
		}
		// General system labels (Inbox/Sent/Trash/Spam/Draft/Category_*)
		isSystemGeneral := strings.HasPrefix(upperID, "CATEGORY_") || upperID == "INBOX" || upperID == "CHAT" || upperID == "SENT" || upperID == "TRASH" || upperID == "SPAM" || upperID == "DRAFT"
		if isSystemGeneral && !er.showSystemLabelsInList {
			continue
		}
		// Normalize display name (Category_* → friendly name; Title Case otherwise)
		names = append(names, normalizeLabelDisplay(name, id))
	}
	var b strings.Builder
	b.WriteString(" ")
	// First: labels
	if len(names) > 0 {
		if len(names) > 3 {
			for i := 0; i < 3; i++ {
				b.WriteString(" [")
				b.WriteString(names[i])
				b.WriteString("]")
			}
			b.WriteString(fmt.Sprintf(" [+%d]", len(names)-3))
		} else {
			for _, n := range names {
				b.WriteString(" [")
				b.WriteString(n)
				b.WriteString("]")
			}
		}
	}
	// Then: icons
	if hasAttachment {
		b.WriteString(" 📎")
	}
	if hasCalendar {
		b.WriteString(" 🗓️")
	}
	result := b.String()
	return result
}

// toTitleCase converts strings like "AWS", "spam", "aws-partners" to "Aws", "Spam", "Aws Partners"
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	// Replace common separators with spaces, lower the rest, then title-case tokens
	repl := strings.NewReplacer("_", " ", "-", " ", ".", " ")
	s = repl.Replace(s)
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	for i, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(strings.ToLower(p))
		r[0] = unicode.ToUpper(r[0])
		parts[i] = string(r)
	}
	return strings.Join(parts, " ")
}

// normalizeLabelDisplay maps system names to friendly Display, including Category_* → <name>
func normalizeLabelDisplay(name, id string) string {
	if name == "" && id != "" {
		name = id
	}
	upperID := strings.ToUpper(id)
	upperName := strings.ToUpper(name)
	// Category_*: show only the category name
	if strings.HasPrefix(upperID, "CATEGORY_") {
		n := strings.TrimPrefix(id, "CATEGORY_")
		return toTitleCase(n)
	}
	if strings.HasPrefix(upperName, "CATEGORY_") {
		n := strings.TrimPrefix(name, "CATEGORY_")
		return toTitleCase(n)
	}
	// Generic title case otherwise
	return toTitleCase(name)
}

// FormatEmailHeader formats email header for display
func (er *EmailRenderer) FormatEmailHeader(message *googleGmail.Message) string {
	// Backward-compatible simple header
	header := fmt.Sprintf("Subject: %s\nFrom: %s\nDate: %s\nLabels: %s",
		er.getHeader(message, "Subject"),
		er.getHeader(message, "From"),
		er.formatDate(er.getDate(message)),
		strings.Join(message.LabelIds, ", "))
	return header
}

// FormatHeaderStyled: simple header without colors/markup
func (er *EmailRenderer) FormatHeaderStyled(subject, from, to, cc string, date time.Time, labels []string) string {
	// Plain styled header (tview markup): everything in green, values escaped by caller if needed
	// Show To and Cc only if present
	lines := []string{
		fmt.Sprintf("Subject: %s", subject),
		fmt.Sprintf("From: %s", from),
	}
	if strings.TrimSpace(to) != "" {
		lines = append(lines, fmt.Sprintf("To: %s", to))
	}
	if strings.TrimSpace(cc) != "" {
		lines = append(lines, fmt.Sprintf("Cc: %s", cc))
	}
	lines = append(lines, fmt.Sprintf("Date: %s", er.formatDate(date)))
	lines = append(lines, fmt.Sprintf("Labels: %s", strings.Join(labels, ", ")))
	header := strings.Join(lines, "\n")
	return "[green]" + header + "[-]\n\n"
}

// FormatHeaderANSI returns the email header formatted using ANSI escape codes (green block)
func (er *EmailRenderer) FormatHeaderANSI(subject, from, to, cc string, date time.Time, labels []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Subject: %s\n", subject)
	fmt.Fprintf(&b, "From: %s\n", from)
	if strings.TrimSpace(to) != "" {
		fmt.Fprintf(&b, "To: %s\n", to)
	}
	if strings.TrimSpace(cc) != "" {
		fmt.Fprintf(&b, "Cc: %s\n", cc)
	}
	fmt.Fprintf(&b, "Date: %s\n", er.formatDate(date))
	fmt.Fprintf(&b, "Labels: %s", strings.Join(labels, ", "))
	// \x1b[32m → green; \x1b[0m → reset
	return "\x1b[32m" + b.String() + "\x1b[0m\n\n"
}

// FormatHeaderPlain returns a plain header without markup/tags
func (er *EmailRenderer) FormatHeaderPlain(subject, from, to, cc string, date time.Time, labels []string) string {
	return er.FormatHeaderPlainWithWidth(subject, from, to, cc, date, labels, 80)
}

// FormatHeaderPlainWithWidth returns a plain header with line wrapping for long fields
func (er *EmailRenderer) FormatHeaderPlainWithWidth(subject, from, to, cc string, date time.Time, labels []string, width int) string {
	var b strings.Builder

	// Format each header field with wrapping
	er.writeWrappedHeaderField(&b, "Subject", subject, width)
	er.writeWrappedHeaderField(&b, "From", from, width)

	if strings.TrimSpace(to) != "" {
		er.writeWrappedHeaderField(&b, "To", to, width)
	}
	if strings.TrimSpace(cc) != "" {
		er.writeWrappedHeaderField(&b, "Cc", cc, width)
	}

	er.writeWrappedHeaderField(&b, "Date", er.formatDate(date), width)
	er.writeWrappedHeaderField(&b, "Labels", strings.Join(labels, ", "), width)

	return strings.TrimRight(b.String(), "\n")
}

// TruncateRecipientField truncates recipient fields to fit within specified line limit
func (er *EmailRenderer) TruncateRecipientField(fieldName, value string, maxLines int, lineWidth int) string {
	if maxLines <= 0 {
		maxLines = 3 // Default to 3 lines for recipient fields
	}

	// Calculate available width for content (excluding field name)
	prefix := fieldName + ": "
	availableWidth := lineWidth - len(prefix)
	if availableWidth < 20 {
		availableWidth = 20 // Minimum reasonable width
	}

	// Split recipients by comma and trim whitespace
	recipients := make([]string, 0)
	for _, recipient := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(recipient); trimmed != "" {
			recipients = append(recipients, trimmed)
		}
	}

	if len(recipients) == 0 {
		return ""
	}

	// Calculate how many recipients fit within maxLines
	currentLine := 1
	currentLineLength := 0
	fittingRecipients := 0

	for i, recipient := range recipients {
		recipientLength := len(recipient)
		if i > 0 {
			recipientLength += 2 // Add ", " separator
		}

		// Check if this recipient fits on current line
		if currentLineLength+recipientLength <= availableWidth {
			currentLineLength += recipientLength
			fittingRecipients++
		} else {
			// Need new line
			if currentLine >= maxLines {
				break // Hit line limit
			}
			currentLine++
			currentLineLength = recipientLength
			fittingRecipients++
		}
	}

	// Build result string
	if fittingRecipients >= len(recipients) {
		// All recipients fit
		return strings.Join(recipients, ", ")
	}

	// Truncation needed - ensure we show at least one recipient if possible
	if fittingRecipients == 0 && len(recipients) > 0 {
		// Force include the first recipient even if it's long
		fittingRecipients = 1
	}

	truncatedRecipients := recipients[:fittingRecipients]
	remaining := len(recipients) - fittingRecipients

	result := strings.Join(truncatedRecipients, ", ")
	if remaining > 0 {
		suffix := fmt.Sprintf(" ... and %d more recipient", remaining)
		if remaining > 1 {
			suffix += "s"
		}

		// Calculate lines currently used by checking for newlines in result
		linesUsed := strings.Count(result, "\n") + 1

		// Ensure suffix fits on last line or new line
		if len(result)+len(suffix) <= availableWidth {
			result += suffix
		} else {
			// Put suffix on new line if we have space for another line
			if linesUsed < maxLines {
				result += "\n" + strings.Repeat(" ", len(prefix)) + suffix
			} else {
				// Try to replace last recipient with suffix if we have multiple recipients
				if len(truncatedRecipients) > 1 {
					newResult := strings.Join(truncatedRecipients[:len(truncatedRecipients)-1], ", ") + suffix
					if len(newResult) <= availableWidth {
						result = newResult
						// remaining++ // Commented out as not used
					}
					// OBLITERATED: empty else branch eliminated! 💥
				} else {
					// Only one recipient and suffix doesn't fit - show truncation on new line anyway
					// This is better than no indication at all
					result += "\n" + strings.Repeat(" ", len(prefix)) + suffix
				}
			}
		}
	}

	return result
}

// writeWrappedHeaderField writes a header field with proper line wrapping
func (er *EmailRenderer) writeWrappedHeaderField(b *strings.Builder, fieldName, value string, width int) {
	if strings.TrimSpace(value) == "" {
		return
	}

	// Special handling for recipient fields - truncate to configurable max lines
	if fieldName == "To" || fieldName == "Cc" {
		maxLines := 3 // Default fallback
		if er.config != nil && er.config.Layout.MaxRecipientLines > 0 {
			maxLines = er.config.Layout.MaxRecipientLines
		}
		truncatedValue := er.TruncateRecipientField(fieldName, value, maxLines, width)
		if truncatedValue == "" {
			return
		}
		value = truncatedValue
	}

	prefix := fieldName + ": "
	prefixLen := len(prefix)

	// If the entire line fits, write it as-is
	if prefixLen+len(value) <= width {
		fmt.Fprintf(b, "%s%s\n", prefix, value)
		return
	}

	// Line needs wrapping
	availableWidth := width - prefixLen
	if availableWidth < 20 { // Minimum reasonable wrap width
		availableWidth = 20
	}

	// Write first line with prefix
	words := strings.Fields(value)
	if len(words) == 0 {
		fmt.Fprintf(b, "%s%s\n", prefix, value)
		return
	}

	currentLine := words[0]
	wordIndex := 1

	// Add words to current line while they fit
	for wordIndex < len(words) {
		testLine := currentLine + " " + words[wordIndex]
		if len(testLine) <= availableWidth {
			currentLine = testLine
			wordIndex++
		} else {
			break
		}
	}

	// Write first line with prefix
	fmt.Fprintf(b, "%s%s\n", prefix, currentLine)

	// Write continuation lines with proper indentation
	indent := strings.Repeat(" ", prefixLen)
	for wordIndex < len(words) {
		currentLine = words[wordIndex]
		wordIndex++

		// Add more words to continuation line
		for wordIndex < len(words) {
			testLine := currentLine + " " + words[wordIndex]
			if len(testLine) <= availableWidth {
				currentLine = testLine
				wordIndex++
			} else {
				break
			}
		}

		// Write continuation line
		fmt.Fprintf(b, "%s%s\n", indent, currentLine)
	}
}

// Helper methods
func (er *EmailRenderer) getHeader(message *googleGmail.Message, name string) string {
	for _, header := range message.Payload.Headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}
	return ""
}

func (er *EmailRenderer) getDate(message *googleGmail.Message) time.Time {
	// Prefer Gmail internal epoch if presente
	if message.InternalDate > 0 {
		return time.UnixMilli(message.InternalDate)
	}
	dateStr := er.getHeader(message, "Date")
	if dateStr != "" {
		// Try multiple formats
		if t, err := time.Parse(time.RFC1123Z, dateStr); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC1123, dateStr); err == nil {
			return t
		}
		if t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", dateStr); err == nil {
			return t
		}
	}
	// Fallback: use serverReceived time now to avoid zeros
	return time.Now()
}

func (er *EmailRenderer) extractSenderName(from string) string {
	if from == "" {
		return ""
	}

	// Handle "Name <email@domain.com>" format
	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		parts := strings.Split(from, "<")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	return from
}

// OBLITERATED: truncateString function eliminated! 💥

// fitWidth truncates and pads on the right to fit a fixed width
func (er *EmailRenderer) fitWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	// Truncate by display width with ellipsis
	s = runewidth.Truncate(s, width, "...")
	// Pad on the right to exact width
	pad := width - runewidth.StringWidth(s)
	if pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// OBLITERATED: rightFit function eliminated! 💥

func (er *EmailRenderer) formatRelativeTime(date time.Time) string {
	now := time.Now()
	diff := now.Sub(date)

	if diff < time.Minute {
		return "now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd", int(diff.Hours()/24))
	} else {
		return date.Format("Jan 2")
	}
}

func (er *EmailRenderer) formatDate(date time.Time) string {
	// Defensive programming: handle zero time to prevent crash
	if date.IsZero() {
		return "Unknown Date"
	}
	return date.Format("Mon, 02 Jan 2006 15:04:05 -0700")
}

// UpdateFromConfig updates the renderer with new configuration
func (er *EmailRenderer) UpdateFromConfig(colors *config.ColorsConfig) {
	er.colorer.UpdateFromStyles(colors)
}
