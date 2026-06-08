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

func TestShowPromptPreviewAddsAndRemovesPage(t *testing.T) {
	app := &App{
		Application: tview.NewApplication(),
		Config:      &config.Config{},
	}
	app.Pages = NewPages()

	app.showPromptPreview("Quick Summary", "Description:\nx\n\nTemplate:\ny")
	if !app.Pages.HasPage("promptPreview") {
		t.Fatal("expected promptPreview page to be added")
	}
	app.closePromptPreview()
	if app.Pages.HasPage("promptPreview") {
		t.Fatal("expected promptPreview page to be removed")
	}
}
