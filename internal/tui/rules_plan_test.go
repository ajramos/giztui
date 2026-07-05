package tui

import (
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestDeterministicRuleCategoryName(t *testing.T) {
	cases := []struct {
		name string
		rule services.DeterministicRuleInfo
		want string
	}{
		{"archive", services.DeterministicRuleInfo{Query: "from:foo@bar.com", Action: "archive"}, "⚡ Archive: from:foo@bar.com"},
		{"mark_read", services.DeterministicRuleInfo{Query: "list:news", Action: "mark_read"}, "⚡ Mark read: list:news"},
		{"trash", services.DeterministicRuleInfo{Query: "subject:spam", Action: "trash"}, "⚡ Trash: subject:spam"},
		{"label with name", services.DeterministicRuleInfo{Query: "from:bank", Action: "label", Label: "Finance"}, "⚡ Label Finance: from:bank"},
		{"prompt", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "⚡ Prompt: from:boss"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deterministicRuleCategoryName(c.rule); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestDeterministicRuleCategoryNameTruncatesLongQuery(t *testing.T) {
	long := strings.Repeat("á", 60) // runes, not bytes
	got := deterministicRuleCategoryName(services.DeterministicRuleInfo{Query: long, Action: "archive"})
	want := "⚡ Archive: " + strings.Repeat("á", 40) + "…"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildDeterministicPlan(t *testing.T) {
	matches := []services.RuleMatch{
		{Rule: services.DeterministicRuleInfo{Query: "from:z", Action: "trash"}, MessageIDs: []string{"m3"}},
		{Rule: services.DeterministicRuleInfo{Query: "from:a", Action: "archive"}, MessageIDs: []string{"m1", "m2"}},
		{Rule: services.DeterministicRuleInfo{Query: "from:none", Action: "archive"}, MessageIDs: nil}, // dropped
		{Rule: services.DeterministicRuleInfo{Query: "from:p", Action: "prompt", PromptID: 7}, MessageIDs: []string{"m4"}},
	}
	plan := buildDeterministicPlan(matches)
	if len(plan.Categories) != 3 {
		t.Fatalf("expected 3 categories (empty match dropped), got %d", len(plan.Categories))
	}
	for _, c := range plan.Categories {
		if c.Priority != "medium" {
			t.Fatalf("category %q priority = %q, want medium", c.Name, c.Priority)
		}
	}
	// SortCategories orders action-first, then name — just assert it ran (stable, deterministic).
	sorted := append([]services.ActionPlanCategory{}, plan.Categories...)
	services.SortCategories(sorted)
	for i := range sorted {
		if sorted[i].Name != plan.Categories[i].Name {
			t.Fatalf("categories not sorted: got %v", plan.Categories)
		}
	}
	// PromptID must survive into the category.
	found := false
	for _, c := range plan.Categories {
		if c.Action == "prompt" {
			found = true
			if c.PromptID != 7 {
				t.Fatalf("PromptID = %d, want 7", c.PromptID)
			}
		}
	}
	if !found {
		t.Fatal("prompt category missing")
	}
}

func TestActionVerbsForPrompt(t *testing.T) {
	if got := actionVerbLabel("prompt"); got != "Prompt" {
		t.Fatalf("actionVerbLabel(prompt) = %q, want Prompt", got)
	}
	if got := actionRuleVerbShort("prompt"); got != "prompt" {
		t.Fatalf("actionRuleVerbShort(prompt) = %q, want prompt", got)
	}
}

func TestBuildDeterministicPlanDisambiguatesDuplicateNames(t *testing.T) {
	matches := []services.RuleMatch{
		{Rule: services.DeterministicRuleInfo{Query: "from:a", Action: "archive"}, MessageIDs: []string{"m1"}},
		{Rule: services.DeterministicRuleInfo{Query: "from:a", Action: "archive"}, MessageIDs: []string{"m2"}},
	}
	plan := buildDeterministicPlan(matches)
	if len(plan.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(plan.Categories))
	}
	names := map[string]bool{}
	for _, c := range plan.Categories {
		if names[c.Name] {
			t.Fatalf("duplicate category name %q", c.Name)
		}
		names[c.Name] = true
	}
	if !names["⚡ Archive: from:a"] || !names["⚡ Archive: from:a (2)"] {
		t.Fatalf("unexpected names: %v", names)
	}
}
