package tui

import (
	"strings"

	"github.com/ajramos/giztui/internal/tts"
	"github.com/derailed/tview"
)

// focusedSpeakText returns the text of the focused panel when it is a TextView (reader, AI
// summary, action-plan digest), tags stripped; otherwise "".
func (a *App) focusedSpeakText() string {
	if tv, ok := a.GetFocus().(*tview.TextView); ok {
		return strings.TrimSpace(tv.GetText(true))
	}
	return ""
}

// toggleSpeak reads the focused panel aloud, or stops if already speaking.
func (a *App) toggleSpeak() {
	svc := a.GetSpeechService()
	if svc == nil {
		return
	}
	if svc.IsSpeaking() {
		svc.Stop()
		go a.GetErrorHandler().ShowInfo(a.ctx, "🔇 Stopped reading")
		return
	}
	if !svc.IsConfigured() {
		go a.GetErrorHandler().ShowWarning(a.ctx, "TTS not configured — set tts.piper_path and tts.model_path (see docs/TTS.md)")
		return
	}
	text := a.focusedSpeakText()
	if text == "" {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing to read here")
		return
	}
	go func() {
		if err := svc.Speak(a.ctx, text); err != nil && err != tts.ErrEmptyText {
			a.GetErrorHandler().ShowError(a.ctx, "TTS failed: "+err.Error())
		}
	}()
	go a.GetErrorHandler().ShowInfo(a.ctx, "🔊 Reading aloud…")
}
