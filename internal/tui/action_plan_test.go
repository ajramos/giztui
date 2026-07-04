package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/stretchr/testify/assert"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestBuildAnalyzerMessages(t *testing.T) {
	mk := func(id, subj, from, snippet string, unread bool) *gmailapi.Message {
		labels := []string{}
		if unread {
			labels = append(labels, "UNREAD")
		}
		return &gmailapi.Message{
			Id:       id,
			Snippet:  snippet,
			LabelIds: labels,
			Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
				{Name: "Subject", Value: subj},
				{Name: "From", Value: from},
			}},
		}
	}

	metas := []*gmailapi.Message{
		mk("m1", "Hello", "a@x.com", "snip1", true),
		mk("m2", "Read one", "b@x.com", "snip2", false), // read → excluded
		mk("m3", "World", "c@x.com", "snip3", true),
		nil, // defensive: nil entries are skipped
	}

	got := buildAnalyzerMessages(metas)
	assert.Len(t, got, 2)
	assert.Equal(t, services.AnalyzerMessage{ID: "m1", Subject: "Hello", From: "a@x.com", Snippet: "snip1"}, got[0])
	assert.Equal(t, "m3", got[1].ID)
}

func msgWith(id, from, subj string, unread bool) *gmailapi.Message {
	m := &gmailapi.Message{Id: id, Snippet: "snip", Payload: &gmailapi.MessagePart{
		Headers: []*gmailapi.MessagePartHeader{
			{Name: "From", Value: from}, {Name: "Subject", Value: subj},
		},
	}}
	if unread {
		m.LabelIds = []string{"UNREAD"}
	}
	return m
}

func TestBuildAnalyzerMessagesForSelection(t *testing.T) {
	metas := []*gmailapi.Message{
		msgWith("1", "a@x.com", "S1", true),
		msgWith("2", "b@x.com", "S2", false), // read, but explicitly selected
		msgWith("3", "c@x.com", "S3", true),
	}
	selected := map[string]bool{"2": true, "3": true}

	got := buildAnalyzerMessagesForSelection(metas, selected)
	if len(got) != 2 {
		t.Fatalf("want 2 selected (incl. read), got %d", len(got))
	}
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids["2"] || !ids["3"] {
		t.Fatalf("expected ids 2 and 3, got %+v", ids)
	}
}

func TestActionKeyHint(t *testing.T) {
	a := &App{}
	a.Keys.Archive = "a"
	a.Keys.ToggleRead = "t"
	a.Keys.Trash = "d"
	a.Keys.ManageLabels = "l"
	assert.Equal(t, "a", a.actionKeyHint("archive"))
	assert.Equal(t, "t", a.actionKeyHint("mark_read"))
	assert.Equal(t, "d", a.actionKeyHint("trash"))
	assert.Equal(t, "l", a.actionKeyHint("label"))
	assert.Equal(t, "", a.actionKeyHint("none"))
}

func TestActionPlanFooterText(t *testing.T) {
	// Default-ish bindings: view_prompt "i", remember "ctrl+r", move "m", skip "space".
	keys := actionPlanFooterKeys{viewPrompt: "i", remember: "ctrl+r", move: "m", skip: "space"}
	onCat := actionPlanFooterText(true, "a", "archive", 7, keys)
	if !strings.Contains(onCat, "a to archive (7)") || !strings.Contains(onCat, "Enter to expand") || !strings.Contains(onCat, "Ctrl+R to remember") {
		t.Fatalf("category footer wrong: %q", onCat)
	}
	if strings.Contains(onCat, "^R") {
		t.Fatalf("footer should spell out Ctrl+R, not ^R: %q", onCat)
	}
	// No suggested action (e.g. read-manually node): only expand/remember/close.
	noAction := actionPlanFooterText(true, "", "none", 0, keys)
	if strings.Contains(noAction, " to ") && strings.Contains(noAction, "(0)") {
		t.Fatalf("no-action footer should not show an action verb: %q", noAction)
	}
	if !strings.Contains(noAction, "Enter to expand") {
		t.Fatalf("no-action footer missing expand: %q", noAction)
	}
	onEmail := actionPlanFooterText(false, "a", "archive", 7, keys)
	if !strings.Contains(onEmail, "Enter to open") || !strings.Contains(onEmail, "Space to skip") || !strings.Contains(onEmail, "Ctrl+R to remember sender") {
		t.Fatalf("email footer wrong: %q", onEmail)
	}
	// Footer must advertise the CONFIGURED view-prompt key ("i prompt"), not a hardcoded "v".
	if !strings.Contains(onCat, "i prompt") {
		t.Fatalf("category footer should advertise the configured prompt viewer key: %q", onCat)
	}
	if !strings.Contains(onEmail, "i prompt") {
		t.Fatalf("email footer should advertise the configured prompt viewer key: %q", onEmail)
	}
	// And it must reflect a custom binding too.
	custom := actionPlanFooterText(false, "a", "archive", 1, actionPlanFooterKeys{viewPrompt: "x", remember: "ctrl+r", move: "m", skip: "space"})
	if !strings.Contains(custom, "x prompt") {
		t.Fatalf("footer should reflect a custom view-prompt key: %q", custom)
	}
}

func TestCheckedIDs(t *testing.T) {
	all := []string{"a", "b", "c"}
	excluded := map[string]bool{"b": true}
	got := checkedIDs(all, excluded)
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("want [a c], got %v", got)
	}
	if len(checkedIDs(all, map[string]bool{"a": true, "b": true, "c": true})) != 0 {
		t.Fatal("all excluded should yield empty")
	}
}

func TestActionPlanTitleText(t *testing.T) {
	// Before the first batch (total==0): analyzing indicator, no batch counts.
	pre := actionPlanTitleText("5 selected", 0, 0, 0, true)
	if !strings.Contains(pre, "5 selected") || !strings.Contains(pre, "analyzing") {
		t.Fatalf("pre-batch title missing scope/indicator: %q", pre)
	}
	if strings.Contains(pre, "batch") {
		t.Fatalf("pre-batch title should not show batch counts: %q", pre)
	}
	// Mid-analysis: batch counts.
	mid := actionPlanTitleText("23 unread (inbox)", 1, 3, 0, true)
	if !strings.Contains(mid, "batch 1/3") {
		t.Fatalf("mid title wrong: %q", mid)
	}
	// Completed: group count + done (no batch counts).
	done := actionPlanTitleText("23 unread (inbox)", 3, 3, 4, false)
	if !strings.Contains(done, "4 groups") || !strings.Contains(done, "done") {
		t.Fatalf("done title wrong: %q", done)
	}
}

func TestSyncSelectionToNode(t *testing.T) {
	a := &App{}
	a.Keys.Archive = "a"
	a.Keys.BulkSelect = "space"
	a.Keys.Move = "m"
	a.Keys.ViewPrompt = "i"
	a.Keys.RememberRule = "ctrl+r"
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1"}},
		}},
		excluded: map[string]bool{},
		expanded: map[string]bool{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)

	catNode := tview.NewTreeNode("Promos")
	catNode.SetReference(0)
	emailNode := tview.NewTreeNode("m1")
	emailNode.SetReference(emailRef{catIndex: 0, msgID: "m1"})
	catNode.AddChild(emailNode)
	state.root.AddChild(catNode)

	// Landing on an email node selects that email.
	a.syncSelectionToNode(state, emailNode)
	if state.selectedMsgID != "m1" || state.selectedCategory != 0 {
		t.Fatalf("email node: got msgID=%q cat=%d", state.selectedMsgID, state.selectedCategory)
	}
	if !strings.Contains(state.footer.GetText(true), "Space to skip") {
		t.Fatalf("email footer not shown: %q", state.footer.GetText(true))
	}

	// Landing on a category node MUST clear selectedMsgID (the desync bug).
	a.syncSelectionToNode(state, catNode)
	if state.selectedMsgID != "" {
		t.Fatalf("category node must clear selectedMsgID, got %q", state.selectedMsgID)
	}
	if state.selectedCategory != 0 {
		t.Fatalf("category node: got cat=%d", state.selectedCategory)
	}
	if !strings.Contains(state.footer.GetText(true), "Enter to expand") {
		t.Fatalf("category footer not shown: %q", state.footer.GetText(true))
	}
}

func TestActionPlanMoveInlineSwap(t *testing.T) {
	a := &App{
		Application: tview.NewApplication(),
	}
	a.Pages = NewPages()
	a.Keys.Archive = "a"
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1", "m2"}},
			{Name: "Notifs", Action: "mark_read", MessageIDs: []string{"m3"}},
		}},
		excluded: map[string]bool{},
		expanded: map[string]bool{"promos": true},
		metaByID: map[string]*gmailapi.Message{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.showActionPlanMoveInline(state, 0, "m2")
	if a.focus.cur() != "action_plan_move" {
		t.Fatalf("expected currentFocus=action_plan_move, got %q", a.focus.cur())
	}
	// The tree must be swapped out; item[0] must be the list chooser, not the tree.
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out while the move chooser is shown")
	}
	// item[1] must still be the footer.
	if state.container.ItemAt(1) != state.footer {
		t.Fatal("footer should remain as container item[1]")
	}

	// Esc must restore the tree and reset focus (no wedge at "action_plan_move").
	if lst, ok := a.GetFocus().(*tview.List); ok {
		if cap := lst.GetInputCapture(); cap != nil {
			cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
		}
	} else {
		t.Fatalf("expected the move chooser list to be focused, got %T", a.GetFocus())
	}
	if a.focus.cur() != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.focus.cur())
	}
	if a.actionPlanState.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc, the tree should be restored as the container body")
	}
}

func TestApplyActionPlanMove(t *testing.T) {
	mk := func(name, action string, ids ...string) services.ActionPlanCategory {
		return services.ActionPlanCategory{Name: name, Action: action, MessageIDs: ids}
	}
	// Move m2 from "Promos" (archive) to existing category "Notifs" (mark_read).
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		mk("Promos", "archive", "m1", "m2"),
		mk("Notifs", "mark_read", "m3"),
	}}
	applyActionPlanMove(plan, nil, "m2", moveTarget{kind: "category", catName: "Notifs"})
	if len(plan.Categories) != 2 {
		t.Fatalf("want 2 cats, got %d", len(plan.Categories))
	}
	// Look up by name: moves keep the plan alphabetically sorted, so indices shift.
	if got := plan.Categories[categoryIndexByName(plan, "Promos")].MessageIDs; len(got) != 1 || got[0] != "m1" {
		t.Fatalf("Promos should hold [m1], got %v", got)
	}
	if got := plan.Categories[categoryIndexByName(plan, "Notifs")].MessageIDs; len(got) != 2 {
		t.Fatalf("Notifs should hold 2, got %v", got)
	}

	// Move m1 to a standard action (trash) with no existing trash group → new category.
	applyActionPlanMove(plan, nil, "m1", moveTarget{kind: "action", action: "trash"})
	idx := firstCategoryWithAction(plan, "trash")
	if idx < 0 {
		t.Fatal("expected a new trash category")
	}
	if plan.Categories[idx].MessageIDs[0] != "m1" {
		t.Fatalf("trash group should hold m1, got %v", plan.Categories[idx].MessageIDs)
	}
	// "Promos" is now empty and must be pruned.
	if categoryIndexByName(plan, "Promos") != -1 {
		t.Fatal("empty Promos category should have been pruned")
	}

	// Move m3 to keep → ReadManually. Notifs still holds m2 (moved in earlier),
	// so it must NOT be pruned yet.
	applyActionPlanMove(plan, nil, "m3", moveTarget{kind: "action", action: "keep"})
	if len(plan.ReadManually) != 1 || plan.ReadManually[0].ID != "m3" {
		t.Fatalf("m3 should be in ReadManually, got %+v", plan.ReadManually)
	}
	if ni := categoryIndexByName(plan, "Notifs"); ni == -1 {
		t.Fatal("Notifs should remain — it still holds m2")
	} else if got := plan.Categories[ni].MessageIDs; len(got) != 1 || got[0] != "m2" {
		t.Fatalf("Notifs should hold [m2], got %v", got)
	}

	// Move m2 to keep too → Notifs is now empty and must be pruned.
	applyActionPlanMove(plan, nil, "m2", moveTarget{kind: "action", action: "keep"})
	if categoryIndexByName(plan, "Notifs") != -1 {
		t.Fatal("empty Notifs category should have been pruned")
	}
	if len(plan.ReadManually) != 2 {
		t.Fatalf("ReadManually should hold m3 and m2, got %+v", plan.ReadManually)
	}
}

func TestApplyActionPlanBulkMove(t *testing.T) {
	mk := func(name, action string, ids ...string) services.ActionPlanCategory {
		return services.ActionPlanCategory{Name: name, Action: action, MessageIDs: ids}
	}

	// Category → another category: all IDs move, empty source pruned.
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		mk("Promos", "archive", "m1", "m2", "m3"),
		mk("Notifs", "mark_read", "m4"),
	}}
	n := applyActionPlanBulkMove(plan, nil, 0, moveTarget{kind: "category", catName: "Notifs"})
	if n != 3 {
		t.Fatalf("want 3 moved, got %d", n)
	}
	if categoryIndexByName(plan, "Promos") != -1 {
		t.Fatal("empty source Promos should be pruned")
	}
	if ni := categoryIndexByName(plan, "Notifs"); ni < 0 || len(plan.Categories[ni].MessageIDs) != 4 {
		t.Fatalf("Notifs should hold 4, got %+v", plan.Categories)
	}

	// Category → action (trash), no existing trash group → new category gets all IDs.
	plan = &services.ActionPlan{Categories: []services.ActionPlanCategory{
		mk("Promos", "archive", "a1", "a2"),
	}}
	n = applyActionPlanBulkMove(plan, nil, 0, moveTarget{kind: "action", action: "trash"})
	if n != 2 {
		t.Fatalf("want 2 moved, got %d", n)
	}
	idx := firstCategoryWithAction(plan, "trash")
	if idx < 0 || len(plan.Categories[idx].MessageIDs) != 2 {
		t.Fatalf("trash group should hold 2, got %+v", plan.Categories)
	}

	// Read manually (-1) → a category: all ReadManually IDs move, ReadManually emptied.
	plan = &services.ActionPlan{
		Categories:   []services.ActionPlanCategory{mk("Notifs", "mark_read", "k1")},
		ReadManually: []services.AnalyzerMessage{{ID: "r1"}, {ID: "r2"}},
	}
	n = applyActionPlanBulkMove(plan, nil, -1, moveTarget{kind: "category", catName: "Notifs"})
	if n != 2 {
		t.Fatalf("want 2 moved from ReadManually, got %d", n)
	}
	if len(plan.ReadManually) != 0 {
		t.Fatalf("ReadManually should be empty, got %+v", plan.ReadManually)
	}
	if ni := categoryIndexByName(plan, "Notifs"); ni < 0 || len(plan.Categories[ni].MessageIDs) != 3 {
		t.Fatalf("Notifs should hold 3, got %+v", plan.Categories)
	}

	// Out-of-range source: no-op, returns 0.
	plan = &services.ActionPlan{Categories: []services.ActionPlanCategory{mk("Notifs", "mark_read", "x1")}}
	if got := applyActionPlanBulkMove(plan, nil, 9, moveTarget{kind: "action", action: "trash"}); got != 0 {
		t.Fatalf("out-of-range source should move 0, got %d", got)
	}
	if len(plan.Categories) != 1 || len(plan.Categories[0].MessageIDs) != 1 {
		t.Fatalf("plan should be unchanged, got %+v", plan.Categories)
	}
}

type stubAnalyzerSvc struct{}

func (stubAnalyzerSvc) Analyze(ctx context.Context, messages []services.AnalyzerMessage, opts services.InboxAnalyzerOptions, onProgress func(*services.ActionPlan)) (*services.ActionPlan, error) {
	return nil, nil
}
func (stubAnalyzerSvc) BuildPromptPreview(opts services.InboxAnalyzerOptions) string {
	return "PREVIEW-BODY {{messages}}"
}

func TestActionPlanPromptViewSwap(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	a.Config = config.DefaultConfig()
	a.inboxAnalyzerService = stubAnalyzerSvc{}
	state := &actionPlanState{
		customPromptText: "",
		excluded:         map[string]bool{},
		expanded:         map[string]bool{},
		footer:           tview.NewTextView(),
		plan:             &services.ActionPlan{},
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.showActionPlanPromptView(state)
	if a.focus.cur() != "action_plan_prompt" {
		t.Fatalf("expected currentFocus=action_plan_prompt, got %q", a.focus.cur())
	}
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out while the prompt view is shown")
	}
	view, ok := a.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected the prompt TextView focused, got %T", a.GetFocus())
	}
	if !strings.Contains(view.GetText(true), "PREVIEW-BODY") {
		t.Fatalf("view should show the assembled prompt, got %q", view.GetText(true))
	}

	if cap := view.GetInputCapture(); cap != nil {
		cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.focus.cur() != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.focus.cur())
	}
	if a.actionPlanState.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc, the tree should be restored as the container body")
	}
}

func TestActionVerb_Summarize(t *testing.T) {
	if got := actionVerbLabel("summarize"); got != "summarize" {
		t.Fatalf("actionVerbLabel(summarize)=%q", got)
	}
	// Live feedback: "Review" read like a distinct action next to the "Read manually"
	// bucket. Action "none" must render as "No action".
	if got := actionVerbLabel("none"); got != "No action" {
		t.Fatalf("actionVerbLabel(none)=%q, want \"No action\"", got)
	}
	if got := actionRuleVerbShort("summarize"); got != "digest" {
		t.Fatalf("actionRuleVerbShort(summarize)=%q", got)
	}
	a := &App{}
	a.Keys.Summarize = "y"
	if got := a.actionKeyHint("summarize"); got != "y" {
		t.Fatalf("actionKeyHint(summarize)=%q, want y", got)
	}
}

func TestMessageRowInList(t *testing.T) {
	ids := []string{"a", "b", "c"}
	if row, ok := messageRowInList(ids, "a"); !ok || row != 1 {
		t.Errorf("'a' → row=%d ok=%v, want 1/true (header is row 0)", row, ok)
	}
	if row, ok := messageRowInList(ids, "c"); !ok || row != 3 {
		t.Errorf("'c' → row=%d ok=%v, want 3/true", row, ok)
	}
	if _, ok := messageRowInList(ids, "z"); ok {
		t.Error("absent id should return ok=false")
	}
	if _, ok := messageRowInList(nil, "a"); ok {
		t.Error("empty list should return ok=false")
	}
	if _, ok := messageRowInList(ids, ""); ok {
		t.Error("empty msgID should return ok=false")
	}
}

// Categories stay alphabetically sorted, so a mid-analysis merge can INSERT a category
// and shift every index after it. Expansion is keyed by name (catExpandKey), not index:
// the same category must stay expanded across the shift.
func TestActionPlanExpansionSurvivesResort(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Notifs", Action: "mark_read", MessageIDs: []string{"m1"}},
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m2"}},
		}},
		excluded: map[string]bool{},
		expanded: map[string]bool{"promos": true},
		metaByID: map[string]*gmailapi.Message{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)

	expandedOf := func(name string) bool {
		for _, n := range state.root.GetChildren() {
			if strings.Contains(n.GetText(), name) {
				return n.IsExpanded()
			}
		}
		t.Fatalf("category %q not found in tree", name)
		return false
	}

	a.rebuildActionPlanTree(state)
	if !expandedOf("Promos") || expandedOf("Notifs") {
		t.Fatal("initial expansion wrong")
	}

	// A new batch inserts "Alerts" first (alphabetical) — Promos' index shifts 1→2.
	state.plan.Categories = append([]services.ActionPlanCategory{
		{Name: "Alerts", Action: "trash", MessageIDs: []string{"m3"}},
	}, state.plan.Categories...)
	a.rebuildActionPlanTree(state)
	if !expandedOf("Promos") {
		t.Fatal("Promos must stay expanded after an index-shifting insert")
	}
	if expandedOf("Alerts") || expandedOf("Notifs") {
		t.Fatal("only Promos should be expanded")
	}
}

// Live feedback: the move chooser must number its destinations so "m then a digit"
// picks that target directly — no cursor navigation. Targets beyond 9 keep plain
// Enter selection.
func TestActionPlanMoveChooserDigitShortcut(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	a.Pages = NewPages()
	a.errorHandler = NewErrorHandler(nil, nil, nil, nil, nil)
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1", "m2"}},
			{Name: "Notifs", Action: "mark_read", MessageIDs: []string{"m3"}},
		}},
		excluded: map[string]bool{},
		expanded: map[string]bool{},
		metaByID: map[string]*gmailapi.Message{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.showActionPlanMoveInline(state, 0, "m2")
	lst, ok := a.GetFocus().(*tview.List)
	if !ok {
		t.Fatalf("expected the move chooser list to be focused, got %T", a.GetFocus())
	}

	// Destinations (alphabetical actions first): 1 Archive, 2 Keep, 3 Mark read,
	// 4 No action, 5 Trash, 6 Mark read · Notifs. Pressing '6' must select
	// target 6 immediately (List shortcut runes).
	lst.InputHandler()(tcell.NewEventKey(tcell.KeyRune, '6', tcell.ModNone), func(p tview.Primitive) {})

	ni := categoryIndexByName(state.plan, "Notifs")
	if ni == -1 {
		t.Fatal("Notifs category missing after move")
	}
	if got := state.plan.Categories[ni].MessageIDs; len(got) != 2 || got[1] != "m2" {
		t.Fatalf("digit shortcut should move m2 into Notifs, got %v", got)
	}
	if a.focus.cur() != "action_plan" {
		t.Fatalf("after a digit move, focus should return to action_plan, got %q", a.focus.cur())
	}
}

// Live feedback: after moving an email to an action with no existing group, the new
// category was appended at the END of the plan — the tree and the move chooser then
// showed categories out of order. Moves must keep the plan in display order
// (action first, then name — how the rows read left to right).
func TestApplyActionPlanMoveKeepsDisplayOrder(t *testing.T) {
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "Promos", Action: "archive", MessageIDs: []string{"m1", "m2"}},
		{Name: "Updates", Action: "trash", MessageIDs: []string{"m3"}},
	}}

	// No mark_read group exists → a new "Mark read" category is created. It must slot
	// into display order (archive:Promos < mark_read:Mark read < trash:Updates),
	// not land at the end.
	applyActionPlanMove(plan, nil, "m1", moveTarget{kind: "action", action: "mark_read"})

	var names []string
	for _, c := range plan.Categories {
		names = append(names, c.Name)
	}
	want := []string{"Promos", "Mark read", "Updates"}
	if len(names) != 3 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] {
		t.Fatalf("plan must stay in display order after a move, got %v", names)
	}
}

// Same duplication in the tree: the category header renders "verb · … · name", so a
// verb-named category reads "Mark read · 0/1 · Mark read · MEDIUM". Render the verb once.
func TestTopLevelNodeLabelNoDuplicateVerbName(t *testing.T) {
	a := &App{}
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Mark read", Action: "mark_read", Priority: "medium", MessageIDs: []string{"m1"}},
		}},
		expanded: map[string]bool{},
		excluded: map[string]bool{},
	}
	got := a.topLevelNodeLabel(state, 0)
	if strings.Count(got, "Mark read") != 1 {
		t.Fatalf("verb-named category header should show the verb once, got %q", got)
	}
}

// Categories auto-created by a move are named after their action verb (e.g. a "Mark read"
// category with action mark_read). The chooser's "verb · name" format then reads
// "Mark read · Mark read" (live feedback: "cosas raras que se repiten") — when the name
// IS the verb, render it once.
func TestActionPlanMoveTargetsNoDuplicateVerbLabel(t *testing.T) {
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "Mark read", Action: "mark_read", MessageIDs: []string{"m1"}},
		{Name: "Promos", Action: "archive", MessageIDs: []string{"m2"}},
	}}
	targets := actionPlanMoveTargets(plan, "Promos")
	last := targets[len(targets)-1]
	if last.label != "Mark read" || last.kind != "category" || last.catName != "Mark read" {
		t.Fatalf("verb-named category should render its label once, got %+v", last)
	}
}

// Live feedback: besides "Keep (read manually)" the move chooser must offer "No action" —
// it keeps the email in a themed group with nothing applied, distinct from read-manually.
func TestActionPlanMoveTargetsIncludeNoAction(t *testing.T) {
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "Promos", Action: "archive", MessageIDs: []string{"m1"}},
	}}
	targets := actionPlanMoveTargets(plan, "")
	// Fixed actions, alphabetical: 1 Archive, 2 Keep, 3 Mark read, 4 No action, 5 Trash.
	if targets[3].label != "No action" || targets[3].action != "none" {
		t.Fatalf("4th fixed target should be No action (action none), got %+v", targets[3])
	}
	applyActionPlanMove(plan, nil, "m1", targets[3])
	idx := firstCategoryWithAction(plan, "none")
	if idx < 0 || plan.Categories[idx].Name != "No action" || plan.Categories[idx].MessageIDs[0] != "m1" {
		t.Fatalf("moving to No action should create a none-category holding m1, got %+v", plan.Categories)
	}
}
