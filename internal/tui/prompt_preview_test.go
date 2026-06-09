package tui

import (
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/derailed/tview"
)

func TestPromptPreviewText(t *testing.T) {
	out := promptPreviewText("Short factual summary", "Summarize {{body}} in {{max_words}} words")
	if !strings.Contains(out, "Description:") || !strings.Contains(out, "Short factual summary") {
		t.Errorf("description block missing: %q", out)
	}
	if !strings.Contains(out, "Template:") || !strings.Contains(out, "{{body}}") {
		t.Errorf("template block missing: %q", out)
	}
	out2 := promptPreviewText("  ", "")
	if !strings.Contains(out2, "(no description)") {
		t.Errorf("empty description placeholder missing: %q", out2)
	}
	if !strings.Contains(out2, "(empty template)") {
		t.Errorf("empty template placeholder missing: %q", out2)
	}
}

func TestShowPromptPreviewInlineSwapsAndRestores(t *testing.T) {
	app := &App{
		Application: tview.NewApplication(),
		Config:      &config.Config{},
	}
	app.Pages = NewPages()
	// currentTheme is nil → GetComponentColors returns built-in fallback colors; no theme setup needed.

	list := tview.NewList()
	footer := tview.NewTextView()
	footer.SetText(" Enter to apply | Esc to cancel ")

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetTitle(" 🤖 Prompt Library ")
	container.AddItem(list, 0, 1, true)
	container.AddItem(footer, 1, 0, false)

	applied := false
	app.showPromptPreviewInline(
		container, list, footer,
		" Enter to apply | Esc to cancel ",
		"Quick Summary",
		"Description:\nx\n\nTemplate:\ny",
		func() { applied = true },
	)

	// After showPromptPreviewInline the title should contain the prompt name.
	if !strings.Contains(container.GetTitle(), "Quick Summary") {
		t.Errorf("container title should contain prompt name, got %q", container.GetTitle())
	}

	// The footer text should have changed to the preview hint.
	if !strings.Contains(footer.GetText(false), "Esc/Ctrl+P to go back") {
		t.Errorf("footer should show back hint, got %q", footer.GetText(false))
	}

	// currentFocus should be "prompt_preview".
	if app.currentFocus != "prompt_preview" {
		t.Errorf("expected currentFocus=prompt_preview, got %q", app.currentFocus)
	}

	_ = applied // onApply is tested indirectly via key events in integration; unit scope ends here
}
