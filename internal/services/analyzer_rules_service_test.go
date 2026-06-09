package services

import "testing"

func TestSuggestRuleFromContext(t *testing.T) {
	s := &AnalyzerRulesServiceImpl{}
	cases := []struct {
		name   string
		from   string
		action string
		negate bool
		want   string
	}{
		{"trash negate domain", `"TLDR" <news@tldr.tech>`, "trash", true, "Never trash emails from tldr.tech"},
		{"archive directive domain", "news@tldr.tech", "archive", false, "Always archive emails from tldr.tech"},
		{"bare email no angle", "boss@team.io", "label", false, "Always label emails from team.io"},
		{"no domain falls back to whole from", "weird-sender", "trash", true, "Never trash emails from weird-sender"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := s.SuggestRuleFromContext(c.from, c.action, c.negate)
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}
