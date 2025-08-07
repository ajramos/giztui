package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/derailed/tcell/v2"
	googleGmail "google.golang.org/api/gmail/v1"
)

// EmailColorer maneja los colores de emails
type EmailColorer struct {
	UnreadColor    tcell.Color
	ReadColor      tcell.Color
	ImportantColor tcell.Color
	SentColor      tcell.Color
	DraftColor     tcell.Color
}

// NewEmailColorer creates a new email colorer with default colors
func NewEmailColorer() *EmailColorer {
	return &EmailColorer{
		UnreadColor:    tcell.ColorOrange,
		ReadColor:      tcell.ColorGray,
		ImportantColor: tcell.ColorRed,
		SentColor:      tcell.ColorGreen,
		DraftColor:     tcell.ColorYellow,
	}
}

// ColorerFunc devuelve función de coloreo para emails
func (ec *EmailColorer) ColorerFunc() func(*googleGmail.Message, string) tcell.Color {
	return func(message *googleGmail.Message, column string) tcell.Color {
		switch strings.ToUpper(column) {
		case "STATUS":
			if ec.isUnread(message) {
				return ec.UnreadColor // 🔵 Azul para no leído
			}
			return ec.ReadColor // ⚪ Gris para leído

		case "FROM":
			if ec.isImportant(message) {
				return ec.ImportantColor // 🔴 Rojo para importante
			}
			if ec.isUnread(message) {
				return ec.UnreadColor // 🟠 Naranja para no leído
			}
			return tcell.ColorWhite

		case "SUBJECT":
			if ec.isDraft(message) {
				return ec.DraftColor // 🟡 Amarillo para borrador
			}
			if ec.isSent(message) {
				return ec.SentColor // 🟢 Verde para enviado
			}
			if ec.isUnread(message) {
				return tcell.ColorWhite // ⚪ Blanco brillante
			}
			return ec.ReadColor // ⚫ Gris para leído
		}
		return tcell.ColorWhite
	}
}

// UpdateFromStyles actualiza colores desde configuración
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

// EmailRenderer handles email rendering and formatting
type EmailRenderer struct {
	colorer *EmailColorer
}

// NewEmailRenderer creates a new email renderer
func NewEmailRenderer() *EmailRenderer {
	return &EmailRenderer{
		colorer: NewEmailColorer(),
	}
}

// FormatEmailList formats an email for list display
func (er *EmailRenderer) FormatEmailList(message *googleGmail.Message, maxWidth int) (string, tcell.Color) {
	colorer := er.colorer.ColorerFunc()

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

	// Truncate fields to fit
	senderWidth := 20
	subjectWidth := maxWidth - senderWidth - 15 // Leave space for date and separators

	senderText := er.truncateString(senderName, senderWidth)
	subjectText := er.truncateString(subject, subjectWidth)

	// Create formatted string
	formatted := fmt.Sprintf("%s | %s | %s", senderText, subjectText, date)

	// Determine color based on unread status
	textColor := colorer(message, "SUBJECT")

	return formatted, textColor
}

// FormatEmailHeader formats email header for display
func (er *EmailRenderer) FormatEmailHeader(message *googleGmail.Message) string {
	// Format header
	header := fmt.Sprintf("Subject: %s\nFrom: %s\nDate: %s\nLabels: %s",
		er.getHeader(message, "Subject"),
		er.getHeader(message, "From"),
		er.formatDate(er.getDate(message)),
		strings.Join(message.LabelIds, ", "))

	return header
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
	dateStr := er.getHeader(message, "Date")
	if dateStr == "" {
		return time.Now()
	}

	// Try to parse the date
	date, err := time.Parse(time.RFC1123Z, dateStr)
	if err != nil {
		// Try alternative formats
		date, err = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", dateStr)
		if err != nil {
			return time.Now()
		}
	}
	return date
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
