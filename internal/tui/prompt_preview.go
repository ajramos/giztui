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

// promptPreviewPage is the Pages name for the preview overlay.
const promptPreviewPage = "promptPreview"

// promptPreviewText builds the modal body for a prompt preview: a Description
// block followed by the full Template (PromptText). Empty fields become explicit
// placeholders so the preview is never blank.
func promptPreviewText(description, promptText string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "(no description)"
	}
	tmpl := strings.TrimSpace(promptText)
	if tmpl == "" {
		tmpl = "(empty template)"
	}
	return fmt.Sprintf("Description:\n%s\n\nTemplate:\n%s", desc, tmpl)
}

// closePromptPreview removes the preview overlay and restores focus to whatever
// the picker had focused before it opened. Synchronous (no QueueUpdateDraw) so it
// is safe from Esc handlers.
func (a *App) closePromptPreview() {
	a.Pages.RemovePage(promptPreviewPage)
	if a.promptPreviewPrevFocus != nil {
		a.SetFocus(a.promptPreviewPrevFocus)
		a.promptPreviewPrevFocus = nil
	}
}

// showPromptPreview opens a centered, scrollable modal showing a prompt preview.
// Esc or Ctrl+P closes it (via closePromptPreview); other keys scroll the body.
func (a *App) showPromptPreview(name, body string) {
	colors := a.GetComponentColors("prompts")

	tv := tview.NewTextView().
		SetDynamicColors(false).
		SetWrap(true).
		SetText(body)
	tv.SetScrollable(true)
	tv.SetBackgroundColor(colors.Background.Color())
	tv.SetTextColor(colors.Text.Color())

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).
		SetTitle(fmt.Sprintf(" 👁 Preview: %s ", name)).
		SetTitleColor(colors.Title.Color()).
		SetBorderColor(colors.Border.Color()).
		SetBackgroundColor(colors.Background.Color())
	ForceFilledBorderFlex(box)
	box.SetTitleColor(colors.Title.Color())
	box.AddItem(tv, 0, 1, true)

	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Esc / Ctrl+P to close")
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	box.AddItem(footer, 1, 0, false)

	tv.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape || e.Key() == tcell.KeyCtrlP {
			a.closePromptPreview()
			return nil
		}
		return e
	})

	centered := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(box, 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 4, true).
		AddItem(nil, 0, 1, false)

	a.promptPreviewPrevFocus = a.GetFocus()
	a.Pages.AddPage(promptPreviewPage, centered, true, true)
	a.SetFocus(tv)
}
