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
		{"archive mirrored", services.DeterministicRuleInfo{Query: "from:foo", Action: "archive", GmailFilterID: "flt1"}, "", "⚡ Archive: from:foo ☁️"},
		{"label", services.DeterministicRuleInfo{Query: "from:bank", Action: "label", Label: "Finance"}, "", "⚡ Label Finance: from:bank"},
		{"prompt named", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "Daily digest", "⚡ Prompt 'Daily digest': from:boss"},
		{"prompt unnamed", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "", "⚡ Prompt: from:boss"},
		// mark_read verb: actionVerbLabel returns "Mark read"
		{"mark_read", services.DeterministicRuleInfo{Query: "from:news", Action: "mark_read"}, "", "⚡ Mark read: from:news"},
		// label action with empty Label: falls back to plain "Label" from actionVerbLabel
		{"label empty label", services.DeterministicRuleInfo{Query: "from:x", Action: "label", Label: ""}, "", "⚡ Label: from:x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deterministicRuleListItem(c.rule, c.promptName); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestGmailOnlyListItem(t *testing.T) {
	f := services.GmailOnlyFilter{ID: "F1", Description: "from:(boss@x.com) → forward to me@else.com", Reason: "forwards mail"}
	want := "☁️ from:(boss@x.com) → forward to me@else.com  (Gmail only)"
	if got := gmailOnlyListItem(f); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRuleSyncOp(t *testing.T) {
	cases := []struct {
		name      string
		mirror    bool
		action    string
		hadFilter bool
		want      string
	}{
		{"prompt+hadFilter→unsync", true, "prompt", true, "unsync"},
		{"prompt+noFilter→none", true, "prompt", false, "none"},
		{"mirror+archive→sync", true, "archive", false, "sync"},
		{"mirror+archive+hadFilter→sync", true, "archive", true, "sync"},
		{"no mirror+archive+hadFilter→unsync", false, "archive", true, "unsync"},
		{"no mirror+archive+no filter→none", false, "archive", false, "none"},
		{"no mirror+prompt+hadFilter→unsync", false, "prompt", true, "unsync"},
		{"no mirror+prompt+no filter→none", false, "prompt", false, "none"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ruleSyncOp(c.mirror, c.action, c.hadFilter); got != c.want {
				t.Fatalf("ruleSyncOp(%v, %q, %v) = %q, want %q", c.mirror, c.action, c.hadFilter, got, c.want)
			}
		})
	}
}
