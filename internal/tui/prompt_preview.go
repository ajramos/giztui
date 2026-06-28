package tui

import (
	"fmt"
	"strings"

	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// promptPreviewCreateNewHint is shown when the highlighted row is the
// "✨ Create new with AI…" entry rather than a real prompt.
const promptPreviewCreateNewHint = "Opens the AI prompt configurator to create a new prompt."

// promptPreviewText builds the preview body: a Description block followed by the
// full Template. Empty fields become explicit placeholders so it is never blank.
func promptPreviewText(description, promptText string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "(no description)"
	}
	tmpl := strings.TrimSpace(promptText)
	if tmpl == "" {
		tmpl = "(empty template)"
	}
	// [::b] bold headings; content is tview.Escape()d so template brackets render literally.
	return fmt.Sprintf("[::b]Description[::-]\n%s\n\n[::b]Template[::-]\n%s",
		tview.Escape(desc), tview.Escape(tmpl))
}

// showPromptPreviewInline swaps the picker's list for a scrollable preview inside the
// SAME container (search input stays on top), so the preview reads as deeper navigation
// rather than a floating modal. Enter runs onApply (apply prompt / open configurator);
// Esc or Ctrl+P restores the list. While shown, currentFocus is "prompt_preview" so
// keys.go passes keys straight to the preview's own capture.
func (a *App) showPromptPreviewInline(container *tview.Flex, input *tview.InputField, list *tview.List, footer *tview.TextView, footerNormal, name, body string, onApply func()) {
	colors := a.GetComponentColors("prompts")

	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetText(body)
	tv.SetScrollable(true)
	tv.SetBackgroundColor(colors.Background.Color())
	tv.SetTextColor(colors.Text.Color())

	prevTitle := container.GetTitle()

	restore := func() {
		container.RemoveItem(tv)
		container.RemoveItem(footer)
		container.AddItem(input, 3, 0, false)
		container.AddItem(list, 0, 1, true)
		container.AddItem(footer, 1, 0, false)
		container.SetTitle(prevTitle)
		footer.SetText(footerNormal)
		a.focus.set("prompts")
		a.SetFocus(list)
	}

	tv.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch {
		case e.Key() == tcell.KeyEscape || a.matchesConfiguredKey(e, a.Keys.PromptPreview):
			restore()
			return nil
		case e.Key() == tcell.KeyEnter:
			restore() // reset currentFocus="prompts" + swap the list back BEFORE any async apply,
			onApply() // so an early-returning apply can never leave focus wedged at "prompt_preview"
			return nil
		}
		return e
	})

	container.RemoveItem(input)
	container.RemoveItem(list)
	container.RemoveItem(footer)
	container.AddItem(tv, 0, 1, true)
	container.AddItem(footer, 1, 0, false)
	container.SetTitle(fmt.Sprintf(" 👁 Preview: %s ", name))
	footer.SetText(" Enter to apply  |  Esc/Ctrl+P to go back ")
	a.focus.set("prompt_preview")
	a.SetFocus(tv)
}
