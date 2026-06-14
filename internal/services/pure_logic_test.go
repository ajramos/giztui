package services

import "testing"

func TestSenderDomain(t *testing.T) {
	cases := map[string]string{
		"Ángel Ramos <angel@doit.com>": "doit.com",
		"angel@DoiT.com":               "doit.com", // lowercased
		"<bot@noreply.uber.com>":       "noreply.uber.com",
		"no-at-sign":                   "no-at-sign",
		"  spaced@x.org  ":             "x.org",
	}
	for in, want := range cases {
		if got := senderDomain(in); got != want {
			t.Errorf("senderDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestActionRuleVerb(t *testing.T) {
	cases := map[string]string{
		"archive":   "archive",
		"mark_read": "mark as read",
		"trash":     "trash",
		"label":     "label",
		"summarize": "review", // default
		"":          "review",
	}
	for in, want := range cases {
		if got := actionRuleVerb(in); got != want {
			t.Errorf("actionRuleVerb(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaultPreloadConfig(t *testing.T) {
	c := DefaultPreloadConfig()
	if c == nil {
		t.Fatal("DefaultPreloadConfig returned nil")
	}
	if !c.Enabled || !c.NextPageEnabled || !c.AdjacentEnabled {
		t.Errorf("preload toggles should default on: %+v", c)
	}
	if c.NextPageThreshold <= 0 || c.AdjacentCount <= 0 || c.BackgroundWorkers <= 0 {
		t.Errorf("preload numbers should be positive: %+v", c)
	}
}
