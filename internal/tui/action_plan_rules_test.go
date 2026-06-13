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

func TestRememberRuleInlineSwap(t *testing.T) {
	a := newRulesTestApp()

	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1"}},
		}},
		excluded: map[string]bool{},
		expanded: map[int]bool{},
		metaByID: nil,
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.showRememberRuleModal("always archive promos")
	if a.currentFocus != "action_plan_rule" {
		t.Fatalf("expected currentFocus=action_plan_rule, got %q", a.currentFocus)
	}
	// Tree swapped out for the input.
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out while the remember-rule input is shown")
	}
	input, ok := a.GetFocus().(*tview.InputField)
	if !ok {
		t.Fatalf("expected the remember-rule input focused, got %T", a.GetFocus())
	}
	if input.GetText() != "always archive promos" {
		t.Fatalf("input should be pre-seeded with the suggestion, got %q", input.GetText())
	}

	// Esc restores the tree.
	if done := input.GetInputCapture(); done != nil {
		done(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.currentFocus != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.currentFocus)
	}
	if a.actionPlanState.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc, the tree should be restored as the container body")
	}
}

func TestAnalyzerRulesPickerSetsFocusKeepOverride(t *testing.T) {
	a := newRulesTestApp()
	a.openAnalyzerRulesManager()
	if a.cmdFocusOverride != "keep" {
		t.Fatalf("expected cmdFocusOverride=keep so the command bar teardown won't steal focus, got %q", a.cmdFocusOverride)
	}
}

func TestRestoreFocusAfterModal_Keep(t *testing.T) {
	a := newRulesTestApp()
	a.currentFocus = "analyzer_rules"
	a.cmdFocusOverride = "keep"
	a.restoreFocusAfterModal()
	if a.currentFocus != "analyzer_rules" {
		t.Fatalf("keep override must not re-focus the list, got %q", a.currentFocus)
	}
	if a.cmdFocusOverride != "" {
		t.Fatalf("override should be consumed, got %q", a.cmdFocusOverride)
	}
}
