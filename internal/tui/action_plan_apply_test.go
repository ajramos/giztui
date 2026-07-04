package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
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
