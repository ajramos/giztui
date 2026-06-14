package tui

import (
	"testing"

	"github.com/derailed/tview"
)

func ringNames(ring []focusRingEntry) []string {
	names := make([]string, len(ring))
	for i, e := range ring {
		names[i] = e.name
	}
	return names
}

func TestBuildFocusRing_Composition(t *testing.T) {
	a := &App{
		views: map[string]tview.Primitive{
			"list": tview.NewTable(),
			"text": tview.NewTextView(),
		},
		currentActivePicker: PickerLabels,
		labelsView:          tview.NewFlex(),
		aiSummaryVisible:    true,
		aiSummaryView:       tview.NewTextView(),
	}
	got := ringNames(a.buildFocusRing())
	want := []string{"list", "text", "labels", "summary"}
	if len(got) != len(want) {
		t.Fatalf("ring = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ring[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestBuildFocusRing_NoPickerNoSummary(t *testing.T) {
	a := &App{
		views: map[string]tview.Primitive{
			"list": tview.NewTable(),
			"text": tview.NewTextView(),
		},
		currentActivePicker: PickerNone,
	}
	got := ringNames(a.buildFocusRing())
	want := []string{"list", "text"}
	if len(got) != len(want) || got[0] != "list" || got[1] != "text" {
		t.Fatalf("ring = %v, want %v", got, want)
	}
}

// When the Action Plan is mounted, the ring must include its tree (so Tab reaches the reader and
// the panel), and must NOT also add the hidden labels slot — Action Plan reuses currentActivePicker.
func TestBuildFocusRing_ActionPlan(t *testing.T) {
	a := &App{
		views: map[string]tview.Primitive{
			"list": tview.NewTable(),
			"text": tview.NewTextView(),
		},
		currentActivePicker: PickerActionPlan,
		labelsView:          tview.NewFlex(),
		actionPlanState:     &actionPlanState{tree: tview.NewTreeView()},
	}
	got := ringNames(a.buildFocusRing())
	want := []string{"list", "text", "action_plan"}
	if len(got) != len(want) {
		t.Fatalf("ring = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ring[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
	for _, n := range got {
		if n == "labels" {
			t.Fatalf("Action Plan ring must not include the labels slot: %v", got)
		}
	}
}

// Regression for the "Tab only toggles list ↔ picker, never the reader" bug: cycling must include
// the message reader ("text") between the list and the picker.
func TestFocusCycle_IncludesReader(t *testing.T) {
	ring := []focusRingEntry{{name: "list"}, {name: "text"}, {name: "labels"}}

	step := func(from string, forward bool) string {
		return ring[stepFocusIndex(len(ring), focusRingIndex(ring, from), forward)].name
	}

	// Forward: list → text → labels → (wrap) list.
	if got := step("list", true); got != "text" {
		t.Fatalf("forward from list = %q, want text (reader must be reachable)", got)
	}
	if got := step("text", true); got != "labels" {
		t.Fatalf("forward from text = %q, want labels", got)
	}
	if got := step("labels", true); got != "list" {
		t.Fatalf("forward from labels = %q, want list (wrap)", got)
	}

	// Reverse (Shift+Tab): list → labels → text → list.
	if got := step("list", false); got != "labels" {
		t.Fatalf("reverse from list = %q, want labels", got)
	}
	if got := step("text", false); got != "list" {
		t.Fatalf("reverse from text = %q, want list", got)
	}

	// Unknown current focus → forward starts at the first pane, reverse at the last.
	if got := step("???", true); got != "list" {
		t.Fatalf("forward from unknown = %q, want list", got)
	}
	if got := step("???", false); got != "labels" {
		t.Fatalf("reverse from unknown = %q, want labels", got)
	}
}

func TestStepFocusIndex_EmptyRing(t *testing.T) {
	if got := stepFocusIndex(0, -1, true); got != 0 {
		t.Fatalf("empty ring step = %d, want 0", got)
	}
}
