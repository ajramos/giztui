package render

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ajramos/gmail-tui/internal/config"
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

// UpdateFromStyles updates colors from configuration
func (ec *EmailColorer) UpdateFromStyles(colors *config.ColorsConfig) {
	ec.UnreadColor = colors.Email.UnreadColor.Color()
	ec.ReadColor = colors.Email.ReadColor.Color()
	ec.ImportantColor = colors.Email.ImportantColor.Color()
	ec.SentColor = colors.Email.SentColor.Color()
	ec.DraftColor = colors.Email.DraftColor.Color()
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
	Header      string
	Alignment   int // tview alignment constant
	Expansion   int // Weight for extra width distribution
	MaxWidth    int // 0 = unlimited
	MinWidth    int // Minimum guaranteed width
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
}

// NewEmailRenderer creates a new email renderer
func NewEmailRenderer() *EmailRenderer {
	return &EmailRenderer{
		colorer:                NewEmailColorerDefault(),
		headerKeyTag:           "[yellow]",
		labelIdToName:          make(map[string]string),
		showSystemLabelsInList: false,
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

	// Extract separate attachment and calendar icons
	attachmentIcon := er.extractAttachmentIcon(message)
	calendarIcon := er.extractCalendarIcon(message)
	
	// Add labels as suffix to subject (but not icons, they go in separate column)
	labelChips := er.buildLabelChips(message)
	subjectWithSuffix := subject + labelChips

	// Format date
	date := er.formatRelativeTime(er.getDate(message))

	// Determine row color
	color := er.getMessageColor(message)

	return EmailColumnData{
		RowType: RowTypeFlatMessage,
		Columns: []ColumnCell{
			{flags, tview.AlignCenter, 3, 0},
			{senderName, tview.AlignLeft, 0, 1},
			{subjectWithSuffix, tview.AlignLeft, 0, 3},
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
			{"", tview.AlignCenter, 0, 3, 2},      // Flags: ●○!
			{"From", tview.AlignLeft, 1, 0, 15},   // From: expand weight 1
			{"Subject", tview.AlignLeft, 3, 0, 20}, // Subject: expand weight 3
			{"", tview.AlignCenter, 0, 4, 2},      // Icons: 📎🗓️
			{"Date", tview.AlignRight, 0, 16, 8},   // Date: fixed max width
		}
	case ModeThreaded:
		return []ColumnConfig{
			{"", tview.AlignLeft, 0, 8, 4},        // Thread icon + status
			{"#", tview.AlignRight, 0, 6, 3},      // Count [99]
			{"From", tview.AlignLeft, 1, 0, 15},   // From: expand weight 1
			{"Subject", tview.AlignLeft, 3, 0, 20}, // Subject: expand weight 3
			{"", tview.AlignCenter, 0, 4, 2},      // Icons: 📎🗓️
			{"Date", tview.AlignRight, 0, 16, 8},   // Date: fixed max width
		}
	default:
		// Fallback to flat list config
		return GetColumnConfig(ModeFlatList)
	}
}

// extractAttachmentIcon returns attachment icon (📎) padded to 2 characters
func (er *EmailRenderer) extractAttachmentIcon(message *googleGmail.Message) string {
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

// extractCalendarIcon returns calendar icon (📅) padded to 2 characters  
func (er *EmailRenderer) extractCalendarIcon(message *googleGmail.Message) string {
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

// buildLabelChips returns just the label chips like "  [Aws] [Finance] [+2]"  
func (er *EmailRenderer) buildLabelChips(message *googleGmail.Message) string {
	if message == nil {
		return ""
	}
	
	// Use the same label processing logic as the existing buildIconsAndChips function
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
	// Render labels as chips with overflow indicator
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
	return b.String()
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

// writeWrappedHeaderField writes a header field with proper line wrapping
func (er *EmailRenderer) writeWrappedHeaderField(b *strings.Builder, fieldName, value string, width int) {
	if strings.TrimSpace(value) == "" {
		return
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

func (er *EmailRenderer) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen-3]) + "..."
}

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

// rightFit truncates and right-aligns/pads to width
func (er *EmailRenderer) rightFit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	// Truncate from the left by display width
	s = runewidth.TruncateLeft(s, width, "")
	// Pad on the left
	pad := width - runewidth.StringWidth(s)
	if pad > 0 {
		s = strings.Repeat(" ", pad) + s
	}
	return s
}

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
	return date.Format("Mon, 02 Jan 2006 15:04:05 -0700")
}

// UpdateFromConfig updates the renderer with new configuration
func (er *EmailRenderer) UpdateFromConfig(colors *config.ColorsConfig) {
	er.colorer.UpdateFromStyles(colors)
}
