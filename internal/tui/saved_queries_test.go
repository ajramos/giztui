package tui

import (
	"fmt"
	"testing"
)

func TestSavedQueryCategoryLabel(t *testing.T) {
	cases := map[string]string{
		"":       "Default",
		"   ":    "Default",
		"Work":   "Work",
		" Work ": "Work",
	}
	for in, want := range cases {
		if got := savedQueryCategoryLabel(in); got != want {
			t.Errorf("savedQueryCategoryLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatchesSavedQueryFilter(t *testing.T) {
	q := queryItem{name: "Unread from team", category: "Work"}
	uncategorised := queryItem{name: "Invoices", category: ""}

	tests := []struct {
		filter string
		item   queryItem
		want   bool
	}{
		{"", q, true},                     // empty matches everything
		{"team", q, true},                 // name substring
		{"TEAM", q, true},                 // case-insensitive
		{"missing", q, false},             // no name match
		{"@work", q, true},                // category match
		{"@WORK", q, true},                // category case-insensitive
		{"@wo", q, true},                  // category substring
		{"@finance", q, false},            // category mismatch
		{"@", q, true},                    // bare @ matches all
		{"@default", uncategorised, true}, // uncategorised → "Default"
		{"@work", uncategorised, false},   // uncategorised isn't Work
	}
	for _, tc := range tests {
		if got := matchesSavedQueryFilter(tc.item, tc.filter); got != tc.want {
			t.Errorf("matchesSavedQueryFilter(%q, filter=%q) = %v, want %v", tc.item.name, tc.filter, got, tc.want)
		}
	}
}

func TestSortSavedQueriesByCategory(t *testing.T) {
	items := []queryItem{
		{name: "b", category: ""}, // Default
		{name: "a", category: "Work"},
		{name: "z", category: "Finance"},
		{name: "a", category: ""},     // Default
		{name: "B", category: "Work"}, // case-insensitive name order
	}
	sortSavedQueriesByCategory(items)

	// Named categories alphabetically (Finance, Work), then Default last; names
	// sorted case-insensitively within a group.
	got := make([]string, len(items))
	for i, it := range items {
		got[i] = fmt.Sprintf("%s/%s", savedQueryCategoryLabel(it.category), it.name)
	}
	want := []string{"Finance/z", "Work/a", "Work/B", "Default/a", "Default/b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sort order[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}
