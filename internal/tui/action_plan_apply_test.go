package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestBuildPlanApply(t *testing.T) {
	mk := func(name, action, label string, ids ...string) services.ActionPlanCategory {
		return services.ActionPlanCategory{Name: name, Action: action, Label: label, MessageIDs: ids}
	}
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		mk("Newsletters", "archive", "", "m1", "m2", "m3"),
		mk("Receipts", "label", "Finance", "m4", "m5"),
		mk("Spammy", "trash", "", "m6"),
		mk("FYI", "mark_read", "", "m7", "m8"),
		mk("Review", "none", "", "m9"),       // skipped: no action
		mk("Digest", "summarize", "", "m10"), // skipped: not bulk-appliable
		mk("NoLabel", "label", "", "m11"),    // skipped: label action without a label name
		mk("AllOff", "archive", "", "m12"),   // skipped: every email excluded
	}}
	excluded := map[string]bool{"m2": true, "m12": true}

	s := buildPlanApply(plan, excluded)

	if len(s.items) != 4 {
		t.Fatalf("want 4 applicable items, got %d: %+v", len(s.items), s.items)
	}
	// Plan order preserved; exclusions filtered out.
	if s.items[0].catName != "Newsletters" || len(s.items[0].ids) != 2 || s.items[0].ids[0] != "m1" || s.items[0].ids[1] != "m3" {
		t.Fatalf("item 0 wrong: %+v", s.items[0])
	}
	if s.items[1].action != "label" || s.items[1].label != "Finance" || len(s.items[1].ids) != 2 {
		t.Fatalf("item 1 wrong: %+v", s.items[1])
	}
	if s.items[2].catName != "Spammy" || s.items[3].catName != "FYI" {
		t.Fatalf("order wrong: %+v", s.items)
	}
	if s.counts["archive"] != 2 || s.counts["label"] != 2 || s.counts["trash"] != 1 || s.counts["mark_read"] != 2 {
		t.Fatalf("counts wrong: %v", s.counts)
	}
	if s.total != 7 {
		t.Fatalf("want total 7, got %d", s.total)
	}
}

func TestBuildPlanApplyEmpty(t *testing.T) {
	if s := buildPlanApply(nil, nil); s.total != 0 || len(s.items) != 0 {
		t.Fatalf("nil plan should be empty, got %+v", s)
	}
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "Review", Action: "none", MessageIDs: []string{"m1"}},
	}}
	if s := buildPlanApply(plan, nil); s.total != 0 || len(s.items) != 0 {
		t.Fatalf("none-only plan should be empty, got %+v", s)
	}
}

func TestPlanApplyStatusLine(t *testing.T) {
	// Fixed action order (archive, mark read, trash, label); only non-zero counts listed.
	s := planApplySummary{counts: map[string]int{"archive": 12, "trash": 3, "label": 5}, total: 20}
	want := "Apply plan: 12 archive, 3 trash, 5 label — press 'c' again to confirm, Esc cancels"
	if got := s.statusLine("c"); got != want {
		t.Fatalf("statusLine:\n got %q\nwant %q", got, want)
	}
	s2 := planApplySummary{counts: map[string]int{"mark_read": 4}, total: 4}
	want2 := "Apply plan: 4 mark read — press 'x' again to confirm, Esc cancels"
	if got := s2.statusLine("x"); got != want2 {
		t.Fatalf("statusLine single:\n got %q\nwant %q", got, want2)
	}
}

func newConfirmTestApp(t *testing.T) (*App, *actionPlanState, func(*tcell.EventKey) *tcell.EventKey) {
	t.Helper()
	a := &App{Application: tview.NewApplication()}
	a.Pages = NewPages()
	a.errorHandler = NewErrorHandler(nil, nil, nil, nil, nil)
	a.Keys.ConfirmPlan = "c"
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1", "m2"}},
		}},
		excluded: map[string]bool{},
		expanded: map[string]bool{},
		metaByID: map[string]*gmailapi.Message{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	a.actionPlanState = state
	return a, state, a.actionPlanInputCapture(state)
}

func TestActionPlanConfirmTwoPressStateMachine(t *testing.T) {
	a, state, capture := newConfirmTestApp(t)

	// First press of 'c' arms the confirmation and is consumed.
	if ev := capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone)); ev != nil {
		t.Fatal("first confirm press should be consumed")
	}
	if !state.confirmPending {
		t.Fatal("first confirm press should set confirmPending")
	}

	// Any other key clears the pending confirmation AND still does its normal job
	// (Down passes through to the TreeView).
	if ev := capture(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)); ev == nil {
		t.Fatal("navigation key should still pass through to the tree")
	}
	if state.confirmPending {
		t.Fatal("any other key must clear confirmPending")
	}

	// Esc while pending cancels the confirmation ONLY — panel stays open.
	capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if !state.confirmPending {
		t.Fatal("re-arming failed")
	}
	if ev := capture(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)); ev != nil {
		t.Fatal("Esc while pending should be consumed")
	}
	if state.confirmPending {
		t.Fatal("Esc while pending must clear confirmPending")
	}
	if a.actionPlanState != state {
		t.Fatal("Esc while pending must NOT close the panel")
	}
}

func TestActionPlanConfirmBlockedWhileAnalyzing(t *testing.T) {
	_, state, capture := newConfirmTestApp(t)
	state.analyzing.Store(true)
	capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if state.confirmPending {
		t.Fatal("confirm must be blocked while analysis is running")
	}
}

func TestStartActionPlanConfirmNothingToApply(t *testing.T) {
	_, state, capture := newConfirmTestApp(t)
	state.excluded["m1"] = true
	state.excluded["m2"] = true // everything excluded → nothing applicable
	capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if state.confirmPending {
		t.Fatal("empty apply set must not arm the confirmation")
	}
}

// Live feedback: Space must ALWAYS toggle an email's exclusion in the plan panel,
// regardless of what bulk_select is bound to (on the user's machine it wasn't "space",
// so Space silently did nothing and only the configured key worked).
func TestActionPlanSpaceAlwaysTogglesExclusion(t *testing.T) {
	a, state, capture := newConfirmTestApp(t)
	a.Keys.BulkSelect = "s" // simulate a non-space bulk_select binding
	state.container = tview.NewFlex()

	email := tview.NewTreeNode("mail").SetReference(emailRef{catIndex: 0, msgID: "m1"})
	state.root.AddChild(email)
	state.tree.SetCurrentNode(email)

	if ev := capture(tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)); ev != nil {
		t.Fatal("Space on an email node should be consumed")
	}
	if !state.excluded["m1"] {
		t.Fatal("Space must toggle exclusion even when bulk_select is not \"space\"")
	}
}

// Regression (live-found): the GLOBAL input capture (bindKeys, keys.go) runs before the
// tree's capture and used to close the panel on Esc even while a whole-plan confirmation
// was pending — the panel's own Esc-cancels-confirmation branch was never reached.
func TestActionPlanConfirmEscGlobalHandler(t *testing.T) {
	a, state, capture := newConfirmTestApp(t)
	a.currentActivePicker = PickerActionPlan
	a.focus.set("action_plan")
	a.bindKeys()
	global := a.GetInputCapture()

	// Arm via the panel capture, as the real tree would.
	capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if !state.confirmPending {
		t.Fatal("arming failed")
	}

	// Esc hits the global handler first: while pending it must cancel the
	// confirmation ONLY — panel stays open.
	if ev := global(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)); ev != nil {
		t.Fatal("global Esc should be consumed")
	}
	if state.confirmPending {
		t.Fatal("global Esc must clear confirmPending")
	}
	if a.actionPlanState != state || !a.isActionPlanActive() {
		t.Fatal("global Esc while pending must NOT close the panel")
	}

	// With nothing pending, Esc closes the panel as always.
	global(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if a.isActionPlanActive() || a.actionPlanState != nil {
		t.Fatal("plain Esc should close the panel")
	}
}

func TestExecuteActionPlanApplyCommand(t *testing.T) {
	a, state, _ := newConfirmTestApp(t)

	// Analysis still running → refused, not armed.
	state.analyzing.Store(true)
	a.executeActionPlanCommand([]string{"apply"})
	if state.confirmPending {
		t.Fatal(":plan apply must be refused while analysis is running")
	}
	state.analyzing.Store(false)

	// Panel open + finished → same first-press behavior as the key, and focusOverride
	// is set so hideCommandBar's teardown doesn't steal focus from the panel.
	a.executeActionPlanCommand([]string{"apply"})
	if !state.confirmPending {
		t.Fatal(":plan apply should arm the two-press confirmation")
	}
	if a.cmd.focusOverride != "keep" {
		t.Fatalf("expected cmd.focusOverride=keep, got %q", a.cmd.focusOverride)
	}

	// No panel open → error, no panic.
	a.actionPlanState = nil
	a.executeActionPlanCommand([]string{"apply"})
}

func TestBuildPlanApplySkipsPromptCategories(t *testing.T) {
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "⚡ Prompt: from:boss", Action: "prompt", PromptID: 7, MessageIDs: []string{"p1", "p2"}},
		{Name: "⚡ Archive: from:news", Action: "archive", MessageIDs: []string{"a1", "a2"}},
	}}
	s := buildPlanApply(plan, nil)
	if s.total != 2 {
		t.Fatalf("prompt messages must not count toward whole-plan apply: total=%d want 2", s.total)
	}
	if len(s.items) != 1 {
		t.Fatalf("expected only the archive item, got %d items", len(s.items))
	}
}
