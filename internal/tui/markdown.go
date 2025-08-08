package tui

import (
	"strings"
	"unicode"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/derailed/tview"
	html2text "github.com/jaytaylor/html2text"
)

// convertHTMLToText converts HTML to plain text optimized for terminal readability
func convertHTMLToText(html string, width int) string {
	if strings.TrimSpace(html) == "" {
		return ""
	}
	// Use library defaults for stable, legible plain text output
	txt, err := html2text.FromString(html)
	if err != nil {
		return html
	}
	return txt
}

// sanitizeForTerminal replaces or removes glyphs that often render as tofu (�) in terminals
func sanitizeForTerminal(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u00A0': // NBSP
			b.WriteRune(' ')
		case '\u200B', '\u200C', '\u200D', '\uFEFF':
			// zero-width and BOM → drop
		case '\u2013', '\u2014':
			b.WriteRune('-')
		case '\u2022', '\u2043', '\u25AA', '\u25CF', '\u25E6':
			b.WriteString("- ")
		case '\u2018', '\u2019':
			b.WriteRune('\'')
		case '\u201C', '\u201D':
			b.WriteRune('"')
		case '\u2026':
			b.WriteString("...")
		default:
			// Skip control chars except newline/tab
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				continue
			}
			// Many emoji and special symbols fall in So; drop them to avoid tofu
			if unicode.Is(unicode.So, r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	// Collapse excessive blank lines (simple pass)
	out := b.String()
	out = strings.ReplaceAll(out, "\r\n", "\n")
	out = strings.ReplaceAll(out, "\r", "\n")
	out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	return out
}

// renderMessageContent builds header + body (Markdown or plain text)
func (a *App) renderMessageContent(m *gmail.Message) (string, bool) {
	preferMD := a.markdownEnabled && a.markdownTogglePer[m.Id]
	var b strings.Builder
	rawHeader := a.emailRenderer.FormatHeaderStyled(m.Subject, m.From, m.Date, m.Labels)
	if preferMD && m.HTML != "" {
		// Escape header to avoid tview markup issues
		b.WriteString(tview.Escape(rawHeader))
		// Use HTML→text rendering for stability and readability
		width := a.screenWidth
		if width <= 0 {
			width = 100
		}
		txt := convertHTMLToText(m.HTML, width)
		txt = sanitizeForTerminal(txt)
		// Cap very large outputs to avoid sluggish UI
		const maxLen = 200_000
		if len(txt) > maxLen {
			txt = txt[:maxLen] + "\n(truncated)"
		}
		// Escape to ensure consistent rendering inside tview
		b.WriteString(tview.Escape("\n" + txt))
		return b.String(), false
	}
	// Plain text path: escape everything to avoid accidental markup parsing
	body := sanitizeForTerminal(m.PlainText)
	if body == "" {
		body = "No text content available"
	}
	return tview.Escape(rawHeader + body), false
}

// toggleMarkdown toggles markdown view for current selected message and re-renders from cache
func (a *App) toggleMarkdown() {
	mid := a.getCurrentMessageID()
	if mid == "" {
		a.showError("❌ No message selected")
		return
	}
	a.currentMessageID = mid
	a.markdownTogglePer[mid] = !a.markdownTogglePer[mid]
	delete(a.markdownCache, mid)
	if m, ok := a.messageCache[mid]; ok {
		if a.debug {
			a.logger.Printf("toggle M: re-render from cache id=%s preferMD=%v", mid, a.markdownTogglePer[mid])
		}
		prefer := a.markdownTogglePer[mid]
		go func(msg *gmail.Message) {
			rendered, _ := a.renderMessageContent(msg)
			a.QueueUpdateDraw(func() {
				if text, ok := a.views["text"].(*tview.TextView); ok {
					text.SetDynamicColors(true)
					text.Clear()
					text.SetText(rendered)
					text.ScrollToBeginning()
				}
				if prefer {
					a.showStatusMessage("📝 Markdown view enabled")
				} else {
					a.showStatusMessage("🧾 Plain text view enabled")
				}
			})
		}(m)
		return
	}
	if a.debug {
		a.logger.Printf("toggle M: no cache for id=%s", mid)
	}
	a.showStatusMessage("ℹ️ Open the message first to enable Markdown toggle")
}
