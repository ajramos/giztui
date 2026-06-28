package tui

import (
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

func TestPromptPreviewText(t *testing.T) {
	out := promptPreviewText("Short factual summary", "Summarize {{body}} in {{max_words}} words")
	if !strings.Contains(out, "[::b]Description[::-]") || !strings.Contains(out, "Short factual summary") {
		t.Errorf("description block missing: %q", out)
	}
	if !strings.Contains(out, "[::b]Template[::-]") || !strings.Contains(out, "{{body}}") {
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

	input := tview.NewInputField()
	list := tview.NewList()
	footerNormal := " Enter to apply | Esc to cancel "
	footer := tview.NewTextView()
	footer.SetText(footerNormal)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetTitle(" 🤖 Prompt Library ")
	container.AddItem(input, 3, 0, true)
	container.AddItem(list, 0, 1, true)
	container.AddItem(footer, 1, 0, false)

	applied := false
	app.showPromptPreviewInline(
		container, input, list, footer,
		footerNormal,
		"Quick Summary",
		"Description:\nx\n\nTemplate:\ny",
		func() { applied = true },
	)

	// --- SHOW assertions ---

	// After showPromptPreviewInline the title should contain the prompt name.
	if !strings.Contains(container.GetTitle(), "Quick Summary") {
		t.Errorf("container title should contain prompt name, got %q", container.GetTitle())
	}

	// The footer text should have changed to the preview hint.
	if !strings.Contains(footer.GetText(false), "Esc/Ctrl+P to go back") {
		t.Errorf("footer should show back hint, got %q", footer.GetText(false))
	}

	// currentFocus should be "prompt_preview".
	if app.focus.cur() != "prompt_preview" {
		t.Errorf("expected currentFocus=prompt_preview, got %q", app.focus.cur())
	}

	// The container's second item (index 1, after the input search field is absent here,
	// so index 0 is the preview TextView and index 1 is the footer) must NOT be the list.
	// Equivalently, verify the focused primitive is a *tview.TextView (the preview, not the list).
	tv, ok := app.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected focused primitive to be *tview.TextView after show, got %T", app.GetFocus())
	}

	// --- RESTORE path: drive Esc through the TextView's input capture ---

	capture := tv.GetInputCapture()
	if capture == nil {
		t.Fatal("preview TextView has no input capture installed")
	}

	escEvent := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	result := capture(escEvent)
	// A nil return means the event was consumed (restore ran).
	if result != nil {
		t.Errorf("Esc should be consumed by the preview capture (return nil), got %v", result)
	}

	// currentFocus must have been reset to "prompts".
	if app.focus.cur() != "prompts" {
		t.Errorf("expected currentFocus=prompts after Esc restore, got %q", app.focus.cur())
	}

	// The footer text must be back to the normal hint.
	// tview.TextView.GetText appends a trailing newline; trim before comparing.
	if strings.TrimRight(footer.GetText(false), "\n") != footerNormal {
		t.Errorf("expected footer restored to %q, got %q", footerNormal, footer.GetText(false))
	}

	// After restore the layout must be [input, list, footer].
	if container.ItemAt(0) != input {
		t.Errorf("expected input to be container item[0] after restore, got %T", container.ItemAt(0))
	}
	if container.ItemAt(1) != list {
		t.Errorf("expected list to be container item[1] after restore, got %T", container.ItemAt(1))
	}

	_ = applied // onApply is tested via the Enter path below
}

func TestShowPromptPreviewInlineEnterRestoresBeforeApply(t *testing.T) {
	// Verifies that pressing Enter calls restore() before onApply(), so currentFocus is
	// never left wedged at "prompt_preview" even when onApply early-returns.
	app := &App{
		Application: tview.NewApplication(),
		Config:      &config.Config{},
	}
	app.Pages = NewPages()

	input := tview.NewInputField()
	list := tview.NewList()
	footerNormal := " Enter to apply | Esc to cancel "
	footer := tview.NewTextView()
	footer.SetText(footerNormal)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetTitle(" 🤖 Prompt Library ")
	container.AddItem(input, 3, 0, true)
	container.AddItem(list, 0, 1, true)
	container.AddItem(footer, 1, 0, false)

	// onApply records whether it was called and what currentFocus was at call time.
	var applyCalled bool
	var focusAtApply string
	app.showPromptPreviewInline(
		container, input, list, footer,
		footerNormal,
		"Quick Summary",
		"Description:\nx\n\nTemplate:\ny",
		func() {
			applyCalled = true
			focusAtApply = app.focus.cur()
		},
	)

	// Verify preview is active.
	if app.focus.cur() != "prompt_preview" {
		t.Fatalf("pre-condition: expected prompt_preview, got %q", app.focus.cur())
	}

	tv, ok := app.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected *tview.TextView, got %T", app.GetFocus())
	}

	capture := tv.GetInputCapture()
	if capture == nil {
		t.Fatal("preview TextView has no input capture")
	}

	enterEvent := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	result := capture(enterEvent)
	if result != nil {
		t.Errorf("Enter should be consumed (return nil), got %v", result)
	}

	if !applyCalled {
		t.Error("onApply was not called on Enter")
	}

	// restore() must have run BEFORE onApply, so focusAtApply should be "prompts" (not "prompt_preview").
	if focusAtApply != "prompts" {
		t.Errorf("restore must run before onApply: expected focusAtApply=prompts, got %q", focusAtApply)
	}

	// After the whole Enter sequence, currentFocus remains "prompts".
	if app.focus.cur() != "prompts" {
		t.Errorf("expected currentFocus=prompts after Enter, got %q", app.focus.cur())
	}
}
