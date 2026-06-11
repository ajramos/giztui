package tui

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// stubRulesService implements services.AnalyzerRulesService for picker swap/focus tests.
type stubRulesService struct{}

func (stubRulesService) SaveRule(ctx context.Context, ruleText string) error { return nil }
func (stubRulesService) ListRules(ctx context.Context) ([]services.AnalyzerRuleInfo, error) {
	return nil, nil
}
func (stubRulesService) DeleteRule(ctx context.Context, id int64) error { return nil }
func (stubRulesService) SuggestRuleFromContext(from, action string, negate bool) string {
	return ""
}

func newRulesTestApp() *App {
	a := &App{Application: tview.NewApplication()}
	a.ctx = context.Background()
	a.Pages = NewPages()
	a.errorHandler = NewErrorHandler(nil, nil, nil, nil, nil)
	a.analyzerRulesService = stubRulesService{}
	a.views = map[string]tview.Primitive{
		"contentSplit": tview.NewFlex(),
		"list":         tview.NewTable(),
	}
	return a
}

func TestAnalyzerRulesPickerSwap(t *testing.T) {
	a := newRulesTestApp()

	a.openAnalyzerRulesManager()
	if a.currentFocus != "analyzer_rules" {
		t.Fatalf("expected currentFocus=analyzer_rules, got %q", a.currentFocus)
	}
	if a.currentActivePicker != PickerAnalyzerRules {
		t.Fatalf("expected active picker=analyzer_rules, got %q", a.currentActivePicker)
	}
	// The picker list must be focused.
	list, ok := a.GetFocus().(*tview.List)
	if !ok {
		t.Fatalf("expected the rules list focused, got %T", a.GetFocus())
	}

	// Esc closes the picker and returns focus to the inbox.
	if cap := list.GetInputCapture(); cap != nil {
		cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.currentFocus != "list" {
		t.Fatalf("after Esc, currentFocus should be list, got %q", a.currentFocus)
	}
	if a.currentActivePicker != PickerNone {
		t.Fatalf("after Esc, active picker should be none, got %q", a.currentActivePicker)
	}
}
