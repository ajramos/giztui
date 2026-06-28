package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

type stubAISummary struct{}

func (stubAISummary) GenerateSummary(ctx context.Context, content string, o services.SummaryOptions) (*services.SummaryResult, error) {
	return &services.SummaryResult{Summary: "DIGEST RESULT"}, nil
}
func (stubAISummary) GenerateSummaryStream(ctx context.Context, content string, o services.SummaryOptions, onToken func(string)) (*services.SummaryResult, error) {
	return &services.SummaryResult{Summary: "DIGEST RESULT"}, nil
}
func (stubAISummary) GenerateReply(ctx context.Context, content string, o services.ReplyOptions) (string, error) {
	return "", nil
}
func (stubAISummary) SuggestLabels(ctx context.Context, content string, l []string) ([]string, error) {
	return nil, nil
}
func (stubAISummary) FormatContent(ctx context.Context, content string, o services.FormatOptions) (string, error) {
	return content, nil
}
func (stubAISummary) ApplyCustomPrompt(ctx context.Context, p string, v map[string]string) (string, error) {
	return "", nil
}
func (stubAISummary) ApplyCustomPromptStream(ctx context.Context, p string, v map[string]string, onToken func(string)) (string, error) {
	return "", nil
}

func TestActionPlanSummarizeSwap(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	a.ctx = context.Background()
	a.aiService = stubAISummary{}
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "News", Action: "summarize", MessageIDs: []string{"m1"}},
		}},
		selectedCategory: 0,
		excluded:         map[string]bool{},
		expanded:         map[int]bool{},
		metaByID:         map[string]*gmailapi.Message{"m1": {Id: "m1", Snippet: "s"}},
		footer:           tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.dispatchActionPlanSummarize(state)
	if a.focus.cur() != "action_plan_summary" {
		t.Fatalf("expected currentFocus=action_plan_summary, got %q", a.focus.cur())
	}
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out for the summary view")
	}
	view, ok := a.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected the summary TextView focused, got %T", a.GetFocus())
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

func TestBuildSummarizeInput(t *testing.T) {
	meta := map[string]*gmailapi.Message{
		"m1": {Id: "m1", Snippet: "snip1", Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
			{Name: "Subject", Value: "Hello"}, {Name: "From", Value: "a@x.com"},
		}}},
		"m2": {Id: "m2", Snippet: "snip2", Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
			{Name: "Subject", Value: "World"}, {Name: "From", Value: "b@x.com"},
		}}},
	}
	bodies := map[string]string{"m1": "full body one"}

	out := buildSummarizeInput([]string{"m1", "m2"}, bodies, meta, 1000)

	if !strings.Contains(out, "Hello") || !strings.Contains(out, "a@x.com") {
		t.Fatalf("missing m1 subject/from:\n%s", out)
	}
	if !strings.Contains(out, "full body one") {
		t.Fatalf("m1 should use its body:\n%s", out)
	}
	if !strings.Contains(out, "snip2") {
		t.Fatalf("m2 has no body → should fall back to snippet:\n%s", out)
	}
}
