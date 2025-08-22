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

// whitespaceOnlyEqual checks if two strings differ only by whitespace (spaces/newlines/tabs)
func whitespaceOnlyEqual(a, b string) bool {
	norm := func(s string) string {
		// collapse all runs of whitespace to single spaces
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		s = strings.TrimSpace(s)
		fields := strings.Fields(s)
		return strings.Join(fields, " ")
	}
	return norm(a) == norm(b)
}

// paragraphsOnlyDeleted returns true if out can be obtained from in by removing
// whole paragraphs (blocks separated by blank lines) and changing only whitespace.
func paragraphsOnlyDeleted(in, out string) bool {
	normalizeParagraphs := func(s string) []string {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		parts := strings.Split(s, "\n\n")
		res := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			// collapse internal whitespace within each paragraph
			res = append(res, strings.Join(strings.Fields(p), " "))
		}
		return res
	}
	ain := normalizeParagraphs(in)
	bout := normalizeParagraphs(out)
	// out must be a subsequence of in (only deletions allowed)
	i := 0
	for _, b := range bout {
		for i < len(ain) && ain[i] != b {
			i++
		}
		if i == len(ain) {
			return false
		}
		i++
	}
	return true
}

// renderMessageContent builds body via deterministic formatter and optional LLM touch-up
func (a *App) renderMessageContent(m *gmail.Message) (string, bool) {
	// Update header TextView separately (tview markup)
	if hv, ok := a.views["header"].(*tview.TextView); ok {
		hv.SetDynamicColors(true)
		
		// Check header visibility via DisplayService
		_, _, _, _, _, _, _, _, _, displayService := a.GetServices()
		if displayService != nil && displayService.IsHeaderVisible() {
			headerWidth := a.getHeaderWidth()
			headerContent := a.emailRenderer.FormatHeaderPlainWithWidth(m.Subject, m.From, m.To, m.Cc, m.Date, m.Labels, headerWidth)
			hv.SetText(headerContent)
			
			// Dynamically adjust header height based on content
			a.adjustHeaderHeight(headerContent)
		} else {
			// Headers hidden - clear content and set height to 0
			hv.SetText("")
			a.adjustHeaderHeight("") // This will set height to 0
		}
	}

	width := a.getListWidth()
	useLLM := a.llmTouchUpEnabled

	// Optional LLM touch-up function
	var touch render.TouchUpFunc
	if useLLM && a.LLM != nil {
		touch = func(ctx context.Context, input string, wrapWidth int) (string, error) {
			// Build from configurable template
			tmpl := strings.TrimSpace(a.Config.LLM.GetTouchUpPrompt())
			// Default strict prompt
			if tmpl == "" {
				tmpl = "You are a formatting assistant. Do NOT paraphrase, translate, summarize, or remove any content. " +
					"Only adjust whitespace and line breaks to improve terminal readability within a wrap width of {{wrap_width}}.\n" +
					"Preserve quotes (> ), code/pre/PGP blocks verbatim, lists, ASCII tables, and link references (text [n] + [LINKS]).\n" +
					"Preserve [ATTACHMENTS] and [IMAGES] sections unchanged. Output only the adjusted text.\n\n{{body}}"
			}
			prompt := strings.ReplaceAll(tmpl, "{{wrap_width}}", fmt.Sprintf("%d", wrapWidth))
			prompt = strings.ReplaceAll(prompt, "{{body}}", input)
			// Prefer ParamProvider with temperature 0 to avoid semantic drift
			type paramProv interface {
				GenerateWithParams(string, map[string]interface{}) (string, error)
			}
			if pp, ok := a.LLM.(paramProv); ok {
				out, err := pp.GenerateWithParams(prompt, map[string]interface{}{"temperature": 0.0})
				if err != nil {
					return "", err
				}
				if whitespaceOnlyEqual(input, out) || paragraphsOnlyDeleted(input, out) {
					return out, nil
				}
				return input, nil
			}
			out, err := a.LLM.Generate(prompt)
			if err != nil {
				return "", err
			}
			if whitespaceOnlyEqual(input, out) || paragraphsOnlyDeleted(input, out) {
				return out, nil
			}
			return input, nil
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
	a.SetCurrentMessageID(mid)
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
