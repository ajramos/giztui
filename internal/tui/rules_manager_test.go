package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestDeterministicRuleListItem(t *testing.T) {
	cases := []struct {
		name       string
		rule       services.DeterministicRuleInfo
		promptName string
		want       string
	}{
		{"archive local", services.DeterministicRuleInfo{Query: "from:foo", Action: "archive"}, "", "⚡ Archive: from:foo"},
		{"archive mirrored", services.DeterministicRuleInfo{Query: "from:foo", Action: "archive", GmailFilterID: "flt1"}, "", "⚡ Archive: from:foo ☁"},
		{"label", services.DeterministicRuleInfo{Query: "from:bank", Action: "label", Label: "Finance"}, "", "⚡ Label Finance: from:bank"},
		{"prompt named", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "Daily digest", "⚡ Prompt 'Daily digest': from:boss"},
		{"prompt unnamed", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "", "⚡ Prompt: from:boss"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deterministicRuleListItem(c.rule, c.promptName); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}
