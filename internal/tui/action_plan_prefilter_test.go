package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestApplyPrefilterToMessages(t *testing.T) {
	msgs := []services.AnalyzerMessage{{ID: "m1"}, {ID: "m2"}, {ID: "m3"}}
	out := applyPrefilterToMessages(msgs, []string{"m3", "m1"})
	if len(out) != 2 || out[0].ID != "m1" || out[1].ID != "m3" {
		t.Fatalf("expected [m1 m3] preserving input order, got %+v", out)
	}
	out = applyPrefilterToMessages(msgs, nil)
	if len(out) != 0 {
		t.Fatalf("nil remaining should filter everything, got %+v", out)
	}
}

func TestMergePreResolved(t *testing.T) {
	ai := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "Newsletters", Action: "archive", MessageIDs: []string{"a1"}},
	}}
	pre := []services.ActionPlanCategory{
		{Name: "⚡ Archive: from:foo", Action: "archive", MessageIDs: []string{"p1"}},
	}
	merged := mergePreResolved(ai, pre)
	if len(merged.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(merged.Categories))
	}
	if merged.Categories[0].Name != "⚡ Archive: from:foo" {
		t.Fatalf("rule categories must come first, got %q", merged.Categories[0].Name)
	}
	if len(ai.Categories) != 1 {
		t.Fatalf("original AI plan must not be mutated, got %d categories", len(ai.Categories))
	}
	if got := mergePreResolved(ai, nil); got != ai {
		t.Fatalf("nil preResolved must return the same plan pointer unchanged")
	}
}
