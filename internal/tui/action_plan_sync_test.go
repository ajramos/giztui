package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tview"
)

// dropMessagesFromPlan must remove externally-acted-on messages (trashed/archived from the
// list or reader) from every category and from ReadManually, prune emptied categories, and
// report whether anything changed — so the open Action Plan can drop them and rebuild.
func TestDropMessagesFromPlan(t *testing.T) {
	plan := &services.ActionPlan{
		Categories: []services.ActionPlanCategory{
			{Name: "Newsletters", Action: "archive", MessageIDs: []string{"a", "b", "c"}},
			{Name: "Junk", Action: "trash", MessageIDs: []string{"d"}}, // will empty out
		},
		ReadManually: []services.AnalyzerMessage{{ID: "e"}, {ID: "f"}},
	}

	changed := dropMessagesFromPlan(plan, []string{"b", "d", "f"})
	if !changed {
		t.Fatal("expected changed=true")
	}
	// "b" removed from Newsletters; "a","c" remain.
	if got := plan.Categories[0].MessageIDs; len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("Newsletters after drop = %v, want [a c]", got)
	}
	// "Junk" emptied (only "d") → pruned away, so only 1 category remains.
	if len(plan.Categories) != 1 {
		t.Fatalf("categories after drop = %d, want 1 (Junk pruned)", len(plan.Categories))
	}
	// "f" removed from ReadManually; "e" remains.
	if len(plan.ReadManually) != 1 || plan.ReadManually[0].ID != "e" {
		t.Fatalf("ReadManually after drop = %+v, want only e", plan.ReadManually)
	}
}

// End-to-end: with the Action Plan panel active, removing a message from the list/reader
// (which calls syncActionPlanRemovedIDs) drops it from the plan and rebuilds the tree.
func TestSyncActionPlanRemovedIDs_ActivePanel(t *testing.T) {
	a := &App{}
	a.Keys.Archive, a.Keys.ToggleRead, a.Keys.Trash, a.Keys.ManageLabels = "a", "t", "d", "l"
	root := tview.NewTreeNode("")
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "News", Action: "archive", MessageIDs: []string{"m1", "m2"}},
		}},
		root:      root,
		tree:      tview.NewTreeView().SetRoot(root),
		container: tview.NewFlex().SetDirection(tview.FlexRow),
		excluded:  map[string]bool{},
		expanded:  map[string]bool{},
	}
	a.actionPlanState = state
	a.currentActivePicker = PickerActionPlan // isActionPlanActive() == true

	a.syncActionPlanRemovedIDs("m1")

	if got := state.plan.Categories[0].MessageIDs; len(got) != 1 || got[0] != "m2" {
		t.Fatalf("plan after external removal = %v, want [m2]", got)
	}
}

// When the panel is NOT active, syncing is a no-op (the plan snapshot is left alone).
func TestSyncActionPlanRemovedIDs_InactivePanelNoOp(t *testing.T) {
	a := &App{}
	state := &actionPlanState{plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "News", Action: "archive", MessageIDs: []string{"m1"}},
	}}}
	a.actionPlanState = state
	a.currentActivePicker = PickerNone // not active

	a.syncActionPlanRemovedIDs("m1")

	if len(state.plan.Categories[0].MessageIDs) != 1 {
		t.Fatal("inactive panel must not mutate the plan")
	}
}

func TestDropMessagesFromPlan_NoMatchNoChange(t *testing.T) {
	plan := &services.ActionPlan{
		Categories:   []services.ActionPlanCategory{{Name: "X", Action: "archive", MessageIDs: []string{"a"}}},
		ReadManually: []services.AnalyzerMessage{{ID: "b"}},
	}
	if dropMessagesFromPlan(plan, []string{"zzz"}) {
		t.Fatal("expected changed=false when no id matches")
	}
	if dropMessagesFromPlan(nil, []string{"a"}) {
		t.Fatal("nil plan must be a safe no-op")
	}
	if dropMessagesFromPlan(plan, nil) {
		t.Fatal("empty ids must be a safe no-op")
	}
}
