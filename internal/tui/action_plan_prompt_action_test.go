package tui

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// The prompt dispatcher must body-swap the tree for the result view and restore on Esc,
// exactly like the summarize dispatcher.
func TestDispatchActionPlanPromptSwapAndEsc(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	a.ctx = context.Background()
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "⚡ Prompt: from:boss", Action: "prompt", PromptID: 5, MessageIDs: []string{"m1"}},
		}},
		selectedCategory: 0,
		excluded:         map[string]bool{},
		expanded:         map[string]bool{},
		metaByID:         map[string]*gmailapi.Message{"m1": {Id: "m1", Snippet: "s"}},
		footer:           tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.dispatchActionPlanPrompt(state)
	if a.focus.cur() != "action_plan_prompt_run" {
		t.Fatalf("expected currentFocus=action_plan_prompt_run, got %q", a.focus.cur())
	}
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out for the result view")
	}
	view, ok := a.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected the result TextView focused, got %T", a.GetFocus())
	}
	if cap := view.GetInputCapture(); cap != nil {
		cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.focus.cur() != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.focus.cur())
	}
	if a.actionPlanState.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc the tree should be restored")
	}
}
