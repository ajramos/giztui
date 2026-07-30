package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/render"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// chatPanelState holds the "chat with this email" side panel: a read-only
// transcript plus an input line, scoped to one message. History itself lives in
// the ChatService (keyed by message id); this only renders it.
type chatPanelState struct {
	container  *tview.Flex
	transcript *tview.TextView
	input      *tview.InputField
	messageID  string
	content    string // grounding email body (readable text), loaded lazily
	buf        strings.Builder
	streaming  bool
	cancel     context.CancelFunc
	userTag    string // dynamic-color tag for the user's turns
	aiTag      string // dynamic-color tag for the assistant's turns
}

// userLine / aiLine format one committed turn with its role color. The text is
// tview.Escape'd so markdown brackets (links, etc.) don't break color parsing.
func (st *chatPanelState) userLine(text string) string {
	return st.userTag + "You: " + tview.Escape(text) + chatColorReset + "\n"
}
func (st *chatPanelState) aiLine(text string) string {
	return st.aiTag + "AI: " + tview.Escape(text) + chatColorReset + "\n\n"
}

// chatColorReset resets fg/bg/flags. derailed/tview's colorPattern requires the
// full "[fg:bg:flags]" form (two colons) — a bare "[aqua]"/"[-]" does NOT match
// the regex and would print literally.
const chatColorReset = "[-:-:-]"

// chatColorTag builds a bold, two-colon tview color tag from a theme color so the
// chat's role colors follow the active theme. Falls back to a named color when
// the theme color has no RGB (default). Hex uses the "[#rrggbb::b]" form, which
// the colorPattern regex accepts.
func chatColorTag(c tcell.Color, fallback string) string {
	if h := c.Hex(); h >= 0 {
		return fmt.Sprintf("[#%06x::b]", h)
	}
	return "[" + fallback + "::b]"
}

// openChatPanel opens (or toggles closed) the chat panel for the open message.
// Runs everything on the UI goroutine (call via `go a.openChatPanel()`).
func (a *App) openChatPanel() {
	a.QueueUpdateDraw(func() {
		if a.isChatPanelActive() {
			a.closeChatPanel()
			return
		}
		if a.GetChatService() == nil {
			go a.GetErrorHandler().ShowWarning(a.ctx, "AI chat is not available (no LLM configured)")
			return
		}
		mid := a.GetCurrentMessageID()
		if mid == "" {
			go a.GetErrorHandler().ShowWarning(a.ctx, "Open a message first to chat about it")
			return
		}
		a.buildChatPanel(mid)
	})
}

// buildChatPanel constructs and mounts the chat panel. Must run on the UI goroutine.
func (a *App) buildChatPanel(messageID string) {
	colors := a.GetComponentColors("ai")
	bg := colors.Background.Color()

	transcript := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true)
	transcript.SetBackgroundColor(bg)
	transcript.SetTextColor(colors.Text.Color())
	transcript.SetText("Ask anything about this email. Enter to send, Esc to close.")

	input := tview.NewInputField().
		SetLabel(" › ").
		SetFieldBackgroundColor(bg)
	input.SetLabelColor(colors.Accent.Color())
	input.SetFieldTextColor(colors.Text.Color())
	input.SetBackgroundColor(bg)
	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			a.closeChatPanel()
		case tcell.KeyEnter:
			text := strings.TrimSpace(input.GetText())
			if text == "" {
				return
			}
			st := a.chatPanelState
			if st == nil || st.streaming {
				return // ignore while a reply is streaming
			}
			st.streaming = true
			input.SetText("")
			go a.sendChatMessage(text)
		}
	})

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(bg)
	container.SetBorder(true)
	container.SetBorderColor(colors.Border.Color())
	container.SetTitle(" 💬 Chat with this email ")
	container.SetTitleColor(colors.Title.Color())
	container.AddItem(transcript, 0, 1, false)
	container.AddItem(input, 1, 0, true)

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter send · Esc close ")
	footer.SetTextColor(a.GetComponentColors("general").Text.Color())
	footer.SetBackgroundColor(bg)
	container.AddItem(footer, 1, 0, false)

	st := &chatPanelState{
		container:  container,
		transcript: transcript,
		input:      input,
		messageID:  messageID,
		// Distinct, theme-derived role colors so you can tell your turns from the
		// assistant's (links accent for you, AI accent for the assistant).
		userTag: chatColorTag(a.GetComponentColors("links").Accent.Color(), "aqua"),
		aiTag:   chatColorTag(a.GetComponentColors("ai").Accent.Color(), "lime"),
	}
	a.chatPanelState = st

	// Restore any prior conversation for this message (the ChatService keeps the
	// history; without this the AI would "remember" but the panel would look empty).
	if svc := a.GetChatService(); svc != nil {
		for _, t := range svc.GetHistory(messageID) {
			if t.Role == "assistant" {
				st.buf.WriteString(st.aiLine(t.Text))
			} else {
				st.buf.WriteString(st.userLine(t.Text))
			}
		}
	}
	if st.buf.Len() > 0 {
		transcript.SetText(st.buf.String())
		transcript.ScrollToEnd()
	}

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.SetBackgroundColor(bg)
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.markFocus("labels")
	a.setActivePicker(PickerChat)
	a.SetFocus(input)
}

// closeChatPanel cancels any in-flight reply, collapses the panel and restores
// focus. Synchronous — NEVER QueueUpdateDraw in a close path (CLAUDE.md rule).
func (a *App) closeChatPanel() {
	if a.chatPanelState != nil && a.chatPanelState.cancel != nil {
		a.chatPanelState.cancel()
		a.chatPanelState.cancel = nil
	}
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok && a.labelsView != nil {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.setActivePicker(PickerNone)
	a.chatPanelState = nil
	a.restoreFocusAfterModal()
}

// setChatTranscript replaces the transcript text from a background goroutine.
// QueueUpdateDraw (safe off the event loop) forces the redraw that a bare
// SetText from a goroutine would not; guarded so a closed/replaced panel is a
// no-op.
func (a *App) setChatTranscript(st *chatPanelState, text string) {
	a.QueueUpdateDraw(func() {
		if a.chatPanelState == st && st.transcript != nil {
			st.transcript.SetText(text)
			st.transcript.ScrollToEnd()
		}
	})
}

// sendChatMessage streams a reply to `text`, run in its own goroutine (launched
// from the input's Enter handler). The streaming callback updates the transcript
// directly (NEVER QueueUpdateDraw in a streaming callback — deadlock risk); the
// final commit uses QueueUpdateDraw since it runs off the event loop.
func (a *App) sendChatMessage(text string) {
	st := a.chatPanelState
	if st == nil {
		return
	}
	defer func() { st.streaming = false }()

	chatSvc := a.GetChatService()
	if chatSvc == nil {
		go a.GetErrorHandler().ShowError(a.ctx, "Chat service not available")
		return
	}

	// Live view = the committed transcript so far + this pending exchange. The
	// user turn is only persisted to st.buf on success (mirroring the service,
	// which doesn't record failed turns), so an error leaves a clean history.
	liveBase := st.buf.String() + st.userLine(text)
	// pendingAI renders the in-progress assistant line (already-escaped body).
	pendingAI := func(body string) string {
		return liveBase + st.aiTag + "AI: " + body + chatColorReset
	}

	// Show the user's message + a "thinking" cue IMMEDIATELY — before the
	// (possibly slow) content load — so it never looks stuck. QueueUpdateDraw is
	// safe here (off the event loop, not a streaming callback) and forces the
	// redraw that a bare SetText from this goroutine would not trigger.
	a.setChatTranscript(st, pendingAI("…thinking…"))

	// Load the grounding content once (prefer rendered-visible HTML text so the
	// chat doesn't answer with hidden preheaders / "can't view" boilerplate).
	if st.content == "" {
		m, err := a.Client.GetMessageWithContent(st.messageID)
		if err != nil {
			a.setChatTranscript(st, pendingAI("⚠️ couldn't load the message"))
			go a.GetErrorHandler().ShowError(a.ctx, "Failed to load message for chat")
			return
		}
		st.content = tuiReadableBody(m)
	}
	if strings.TrimSpace(st.content) == "" {
		a.setChatTranscript(st, pendingAI("⚠️ no readable content in this message"))
		go a.GetErrorHandler().ShowError(a.ctx, "Message has no readable content to chat about")
		return
	}

	ctx, cancel := context.WithCancel(a.ctx)
	st.cancel = cancel
	defer func() {
		cancel()
		if st.cancel != nil {
			st.cancel = nil
		}
	}()

	var reply strings.Builder
	_, err := chatSvc.SendMessageStream(ctx, st.messageID, st.content, text, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		reply.WriteString(token)
		// Streaming callback: update directly + ForceDraw (NEVER QueueUpdateDraw
		// here — it can deadlock with the Esc/cancel handler).
		if ctx.Err() == nil && st.transcript != nil {
			st.transcript.SetText(pendingAI(tview.Escape(reply.String())))
			st.transcript.ScrollToEnd()
			a.ForceDraw()
		}
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			return
		}
		a.setChatTranscript(st, pendingAI("⚠️ "+tview.Escape(err.Error())))
		go a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Chat failed: %v", err))
		return
	}

	// Commit the completed exchange to the persistent transcript buffer.
	st.buf.WriteString(st.userLine(text) + st.aiLine(reply.String()))
	a.setChatTranscript(st, st.buf.String())
}

// tuiReadableBody returns readable text for a message, preferring the rendered
// HTML text over the (often hidden-preheader) plain-text part.
func tuiReadableBody(m *gmail.Message) string {
	if m == nil {
		return ""
	}
	if strings.TrimSpace(m.HTML) != "" {
		if t := render.HTMLToText(m.HTML); strings.TrimSpace(t) != "" {
			return t
		}
	}
	return strings.TrimSpace(m.PlainText)
}
