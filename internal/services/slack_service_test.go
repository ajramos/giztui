package services

import (
	"testing"

	"github.com/ajramos/giztui/internal/config"
)

func TestDefaultSlackWebhook(t *testing.T) {
	cfg := &config.Config{Slack: config.SlackConfig{Channels: []config.SlackChannel{
		{Name: "first", WebhookURL: "https://hooks/first"},
		{Name: "main", WebhookURL: "https://hooks/main", Default: true},
	}}}
	if got, err := defaultSlackWebhook(cfg); err != nil || got != "https://hooks/main" {
		t.Fatalf("default channel webhook = %q, err=%v", got, err)
	}

	cfg2 := &config.Config{Slack: config.SlackConfig{Channels: []config.SlackChannel{
		{Name: "only", WebhookURL: "https://hooks/only"},
	}}}
	if got, err := defaultSlackWebhook(cfg2); err != nil || got != "https://hooks/only" {
		t.Fatalf("no default → first; got %q err=%v", got, err)
	}

	cfg3 := &config.Config{Slack: config.SlackConfig{}}
	if _, err := defaultSlackWebhook(cfg3); err == nil {
		t.Fatal("no channels should error")
	}
}

func TestSummaryCount(t *testing.T) {
	cases := []struct {
		name               string
		summaries          bool
		limit, total, want int
	}{
		{"disabled", false, 5, 10, 0},
		{"normal", true, 5, 10, 5},
		{"limit zero clamps to 5", true, 0, 10, 5},
		{"negative clamps to 5", true, -3, 10, 5},
		{"capped at 10", true, 20, 50, 10},
		{"fewer emails than limit", true, 5, 2, 2},
	}
	for _, c := range cases {
		if got := summaryCount(c.summaries, c.limit, c.total); got != c.want {
			t.Errorf("%s: summaryCount(%v,%d,%d) = %d, want %d", c.name, c.summaries, c.limit, c.total, got, c.want)
		}
	}
}
