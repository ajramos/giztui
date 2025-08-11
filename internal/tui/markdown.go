package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/render"
	"github.com/derailed/tview"
)

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

// renderMessageContent builds body via deterministic formatter and optional LLM touch-up
func (a *App) renderMessageContent(m *gmail.Message) (string, bool) {
	// Update header TextView separately (tview markup)
	if hv, ok := a.views["header"].(*tview.TextView); ok {
		hv.SetDynamicColors(true)
		hv.SetText(a.emailRenderer.FormatHeaderPlain(m.Subject, m.From, m.Date, m.Labels))
	}

	width := a.getListWidth()
	useLLM := a.llmTouchUpEnabled

	// Optional LLM touch-up function
	var touch render.TouchUpFunc
	if useLLM && a.LLM != nil {
		touch = func(ctx context.Context, input string, wrapWidth int) (string, error) {
			// Strict instruction: whitespace-only changes
			prompt := "You are a formatting assistant. Do NOT paraphrase, translate, summarize, or remove any content. " +
				"Only adjust whitespace and line breaks to improve terminal readability within a wrap width of " +
				fmt.Sprintf("%d", wrapWidth) + ".\n" +
				"Preserve quotes (> ), code/pre/PGP blocks verbatim, lists, ASCII tables, and link references (text [n] + [LINKS]).\n" +
				"Preserve [ATTACHMENTS] and [IMAGES] sections unchanged. Output only the adjusted text.\n\n" + input
			return a.LLM.Generate(prompt)
		}
	}

	// Deterministic format
	text, err := render.FormatEmailForTerminal(a.ctx, m, render.FormatOptions{WrapWidth: width, UseLLM: useLLM}, touch)
	if err != nil || strings.TrimSpace(text) == "" {
		// Fallback to plain text
		body := sanitizeForTerminal(m.PlainText)
		if body == "" {
			body = "No text content available"
		}
		return tview.Escape(body), false
	}
	// Do not escape; we don't emit ANSI/tview markup here
	// Large cap to protect UI
	const maxLen = 200_000
	if len(text) > maxLen {
		text = text[:maxLen] + "\n(truncated)"
	}
	return text, false
}

// toggleMarkdown repurposed: toggle LLM touch-up on/off for current message
func (a *App) toggleMarkdown() {
	mid := a.getCurrentMessageID()
	if mid == "" {
		a.showError("❌ No message selected")
		return
	}
	a.currentMessageID = mid
	// Show immediate feedback while formatting
	if !a.llmTouchUpEnabled {
		a.setStatusPersistent("🧠 Enabling LLM touch-up…")
	} else {
		a.setStatusPersistent("🧾 Disabling LLM touch-up…")
	}
	a.llmTouchUpEnabled = !a.llmTouchUpEnabled
    if m, ok := a.messageCache[mid]; ok {
		go func(msg *gmail.Message) {
			rendered, _ := a.renderMessageContent(msg)
			a.QueueUpdateDraw(func() {
				if text, ok := a.views["text"].(*tview.TextView); ok {
					text.SetDynamicColors(true)
					text.Clear()
					text.SetText(rendered)
					text.ScrollToBeginning()
				}
				if a.llmTouchUpEnabled {
					a.showStatusMessage("✅ LLM touch-up enabled")
				} else {
					a.showStatusMessage("✅ Deterministic formatting only")
				}
			})
		}(m)
		return
	}
    // If not cached (e.g., after local search), fetch and then render
    go func(id string) {
        fetched, err := a.Client.GetMessageWithContent(id)
        if err != nil {
            a.showError("❌ Could not load message content")
            return
        }
        a.messageCache[id] = fetched
        rendered, _ := a.renderMessageContent(fetched)
        a.QueueUpdateDraw(func() {
            if text, ok := a.views["text"].(*tview.TextView); ok {
                text.SetDynamicColors(true)
                text.Clear()
                text.SetText(rendered)
                text.ScrollToBeginning()
            }
            if a.llmTouchUpEnabled {
                a.showStatusMessage("✅ LLM touch-up enabled")
            } else {
                a.showStatusMessage("✅ Deterministic formatting only")
            }
        })
    }(mid)
}
