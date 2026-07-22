package desktop

import (
	"reflect"
	"testing"
)

func TestConsolidateCategories_WordPrefixMerge(t *testing.T) {
	in := []PlanCategory{
		{Name: "Newsletters", Action: "label", Label: "", MessageIDs: []string{"a", "b"}},
		{Name: "Newsletters & Marketing", Action: "label", Label: "", MessageIDs: []string{"c"}},
		{Name: "Receipts", Action: "archive", MessageIDs: []string{"d"}},
	}
	got := consolidateCategories(in)
	if len(got) != 2 {
		t.Fatalf("want 2 categories, got %d: %+v", len(got), got)
	}
	if got[0].Name != "Newsletters" {
		t.Errorf("want merged name 'Newsletters' (shorter), got %q", got[0].Name)
	}
	if !reflect.DeepEqual(got[0].MessageIDs, []string{"a", "b", "c"}) {
		t.Errorf("want unioned ids [a b c], got %v", got[0].MessageIDs)
	}
}

func TestConsolidateCategories_SameLabelMerge(t *testing.T) {
	in := []PlanCategory{
		{Name: "Work stuff", Action: "label", Label: "Work", MessageIDs: []string{"a"}},
		{Name: "Job things", Action: "label", Label: "Work", MessageIDs: []string{"b"}},
	}
	got := consolidateCategories(in)
	if len(got) != 1 {
		t.Fatalf("want 1 category, got %d: %+v", len(got), got)
	}
	if !reflect.DeepEqual(got[0].MessageIDs, []string{"a", "b"}) {
		t.Errorf("want unioned ids [a b], got %v", got[0].MessageIDs)
	}
}

func TestConsolidateCategories_SkipsRuleAndReadManually(t *testing.T) {
	in := []PlanCategory{
		{Name: "Newsletters", Action: "label", MessageIDs: []string{"a"}, ByRule: true},
		{Name: "Newsletters & Marketing", Action: "label", MessageIDs: []string{"b"}},
		{Name: "Newsletters weekly", Action: "label", MessageIDs: []string{"c"}, ReadManually: true},
	}
	got := consolidateCategories(in)
	// The ByRule and ReadManually buckets must survive untouched; only the plain
	// AI one stands alone (nothing mergeable that isn't protected).
	if len(got) != 3 {
		t.Fatalf("want 3 categories (none merged across protected), got %d: %+v", len(got), got)
	}
}

func TestConsolidateCategories_NoFalsePositivePrefix(t *testing.T) {
	in := []PlanCategory{
		{Name: "New", Action: "archive", MessageIDs: []string{"a"}},
		{Name: "Newsletters", Action: "archive", MessageIDs: []string{"b"}},
	}
	got := consolidateCategories(in)
	if len(got) != 2 {
		t.Fatalf("want 2 categories ('New' is not a word-prefix of 'Newsletters'), got %d: %+v", len(got), got)
	}
}

func TestConsolidateCategories_DifferentActionsDoNotMerge(t *testing.T) {
	in := []PlanCategory{
		{Name: "Newsletters", Action: "archive", MessageIDs: []string{"a"}},
		{Name: "Newsletters & Marketing", Action: "label", MessageIDs: []string{"b"}},
	}
	got := consolidateCategories(in)
	if len(got) != 2 {
		t.Fatalf("want 2 categories (different actions), got %d: %+v", len(got), got)
	}
}

func TestIsWordPrefix(t *testing.T) {
	cases := []struct {
		short, long string
		want        bool
	}{
		{"Newsletters", "Newsletters & Marketing", true},
		{"Newsletters", "Newsletters/Promos", true},
		{"New", "Newsletters", false},
		{"Newsletters", "Newsletters", true},
		{"", "Newsletters", false},
		{"Work", "Work - urgent", true},
		{"Work", "Workspace", false},
	}
	for _, c := range cases {
		if got := isWordPrefix(c.short, c.long); got != c.want {
			t.Errorf("isWordPrefix(%q,%q)=%v want %v", c.short, c.long, got, c.want)
		}
	}
}
