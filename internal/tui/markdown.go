package tui

import (
	"bytes"
	"fmt"
	"strings"

	htmmd "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/charmbracelet/glamour"
	"github.com/derailed/tview"
)

// convertHTMLToMarkdown converts HTML to Markdown with sane defaults
func convertHTMLToMarkdown(html string) string {
	if strings.TrimSpace(html) == "" {
		return ""
	}
	conv := htmmd.NewConverter("", true, nil)
	// Normalize <br> to newlines
	conv.AddRules(htmmd.Rule{
		Filter: []string{"br"},
		Replacement: func(content string, sele *goquery.Selection, opt *htmmd.Options) *string {
			s := "\n"
			return &s
		},
	})
	md, err := conv.ConvertString(html)
	if err != nil {
		return html
	}
	return md
}

// renderMarkdownToANSI renders Markdown to ANSI using Glamour
func renderMarkdownToANSI(md string) string {
	if strings.TrimSpace(md) == "" {
		return ""
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return out
}

// ansiToTview converts ANSI string to tview markup string off the UI thread
func ansiToTview(ansi string) string {
	if ansi == "" {
		return ""
	}
	var buf bytes.Buffer
	w := tview.ANSIWriter(&buf, "", "")
	fmt.Fprint(w, ansi)
	return buf.String()
}

// renderMessageContent builds header + body (Markdown or plain text)
func (a *App) renderMessageContent(m *gmail.Message) (string, bool) {
	preferMD := a.markdownEnabled && a.markdownTogglePer[m.Id]
	var b strings.Builder
	rawHeader := a.emailRenderer.FormatHeaderStyled(m.Subject, m.From, m.Date, m.Labels)
	if preferMD && m.HTML != "" {
		// Escape header to avoid tview markup issues, then append ANSI-rendered markdown
		b.WriteString(tview.Escape(rawHeader))
		md := convertHTMLToMarkdown(m.HTML)
		ansi := renderMarkdownToANSI(md)
		// Convert ANSI to tview markup off-UI thread for faster SetText
		markup := ansiToTview(ansi)
		// Cap very large outputs to avoid sluggish UI
		const maxLen = 200_000
		if len(markup) > maxLen {
			markup = markup[:maxLen] + "\n[grey](truncated)[-]"
		}
		b.WriteString(markup)
		return b.String(), false
	}
	// Plain text path: escape everything to avoid accidental markup parsing
	body := m.PlainText
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
			rendered, isANSI := a.renderMessageContent(msg)
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
