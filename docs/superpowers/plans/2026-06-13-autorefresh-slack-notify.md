# Auto-refresh Slack Notification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When auto-refresh detects new inbox mail (opt-in `auto_refresh.notify_slack`), post one Slack message to the default channel listing the new emails (count + subject · sender).

**Architecture:** A new `SlackService.SendNotification` posts plain text to the default channel's webhook (reusing `sendToSlack`). Auto-refresh's `performAutoRefreshTick` calls a gated `notifyNewMailSlack` that fetches the new IDs' metadata and sends a formatted message.

**Tech Stack:** Go, GizTUI Slack service + auto-refresh + config.

Spec: `docs/superpowers/specs/2026-06-13-autorefresh-slack-notify-design.md`

---

### Task 1: `SlackService.SendNotification` + `defaultSlackWebhook`

**Files:**
- Modify: `internal/services/interfaces.go` (`SlackService` interface)
- Modify: `internal/services/slack_service.go`
- Test: `internal/services/slack_service_test.go` (create if absent, else append)

- [ ] **Step 1: Write the failing test**

Append to (or create) `internal/services/slack_service_test.go`:

```go
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
```

(If the file exists with a `package services` header and config import, just append the func.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestDefaultSlackWebhook -v`
Expected: FAIL — `defaultSlackWebhook` undefined.

- [ ] **Step 3: Implement helper + method**

In `internal/services/slack_service.go`, add:

```go
// defaultSlackWebhook returns the webhook of the default channel (the one with Default==true,
// else the first configured channel). Errors when no channel is configured.
func defaultSlackWebhook(cfg *config.Config) (string, error) {
	if cfg == nil || len(cfg.Slack.Channels) == 0 {
		return "", fmt.Errorf("no Slack channel configured")
	}
	for _, ch := range cfg.Slack.Channels {
		if ch.Default && strings.TrimSpace(ch.WebhookURL) != "" {
			return ch.WebhookURL, nil
		}
	}
	if wh := strings.TrimSpace(cfg.Slack.Channels[0].WebhookURL); wh != "" {
		return wh, nil
	}
	return "", fmt.Errorf("no Slack webhook configured")
}

// SendNotification posts a plain-text message to the default Slack channel.
func (s *SlackServiceImpl) SendNotification(ctx context.Context, text string) error {
	webhook, err := defaultSlackWebhook(s.config)
	if err != nil {
		return err
	}
	return s.sendToSlack(ctx, SlackMessage{Text: text}, webhook)
}
```

Ensure `slack_service.go` imports `strings` and `fmt` (both already used in that file).

- [ ] **Step 4: Add to the interface**

In `internal/services/interfaces.go`, add to the `SlackService` interface:
```go
	SendNotification(ctx context.Context, text string) error
```

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/services/ -run TestDefaultSlackWebhook -v && go build ./...`
Expected: PASS, build success.

- [ ] **Step 6: Commit**

```bash
git add internal/services/interfaces.go internal/services/slack_service.go internal/services/slack_service_test.go
git commit -m "feat(services): SlackService.SendNotification (plain text to default channel)"
```

---

### Task 2: Config flag + notify wiring

**Files:**
- Modify: `internal/config/config.go` (`AutoRefreshConfig`)
- Modify: `internal/tui/auto_refresh.go`
- Test: `internal/tui/auto_refresh_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/auto_refresh_test.go`:

```go
func TestBuildNewMailSlackMessage(t *testing.T) {
	mk := func(subj, from string) *gmailapi.Message {
		return &gmailapi.Message{Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
			{Name: "Subject", Value: subj}, {Name: "From", Value: from},
		}}}
	}
	msg := buildNewMailSlackMessage([]*gmailapi.Message{mk("Hello", "a@x.com"), mk("World", "b@x.com")})
	if !strings.Contains(msg, "2 new email") {
		t.Fatalf("expected count, got:\n%s", msg)
	}
	if !strings.Contains(msg, "Hello — a@x.com") || !strings.Contains(msg, "World — b@x.com") {
		t.Fatalf("expected subject — from lines, got:\n%s", msg)
	}

	// Cap at 10 with overflow note.
	many := make([]*gmailapi.Message, 13)
	for i := range many {
		many[i] = mk("S", "f@x.com")
	}
	capped := buildNewMailSlackMessage(many)
	if !strings.Contains(capped, "and 3 more") {
		t.Fatalf("expected overflow note for 13, got:\n%s", capped)
	}
}
```

Ensure `auto_refresh_test.go` imports `"strings"` and `gmailapi "google.golang.org/api/gmail/v1"`
(add if missing).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestBuildNewMailSlackMessage -v`
Expected: FAIL — `buildNewMailSlackMessage` undefined.

- [ ] **Step 3: Add the config flag**

In `internal/config/config.go`, change `AutoRefreshConfig`:
```go
type AutoRefreshConfig struct {
	Enabled     bool   `json:"enabled"`
	Interval    string `json:"interval"`     // Go duration string, e.g. "5m"; clamped to a 1m minimum
	NotifySlack bool   `json:"notify_slack"` // also post a Slack notification when new mail is detected
}
```
(The `DefaultConfig` literal `AutoRefreshConfig{Enabled: false, Interval: "5m"}` leaves `NotifySlack`
at its zero value `false` — no change needed there.)

- [ ] **Step 4: Implement the helper + notify in `auto_refresh.go`**

In `internal/tui/auto_refresh.go`, add:

```go
// buildNewMailSlackMessage formats a Slack notification listing the new emails (capped at 10).
func buildNewMailSlackMessage(metas []*gmailapi.Message) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📬 %d new email(s):", len(metas))
	const maxList = 10
	for i, m := range metas {
		if m == nil {
			continue
		}
		if i >= maxList {
			fmt.Fprintf(&b, "\n…and %d more", len(metas)-maxList)
			break
		}
		subject := extractHeaderValue(m, "Subject")
		from := extractHeaderValue(m, "From")
		fmt.Fprintf(&b, "\n• %s — %s", subject, from)
	}
	return b.String()
}

// notifyNewMailSlack posts a Slack notification about newly-detected mail, when enabled.
func (a *App) notifyNewMailSlack(newIDs []string) {
	if !a.Config.AutoRefresh.NotifySlack || !a.Config.Slack.Enabled {
		return
	}
	svc := a.GetSlackService()
	if svc == nil || a.Client == nil || len(newIDs) == 0 {
		return
	}
	metas, err := a.Client.GetMessagesMetadataParallel(newIDs, 10)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("AUTO_REFRESH: slack metadata fetch error: %v", err)
		}
		return
	}
	msg := buildNewMailSlackMessage(metas)
	if err := svc.SendNotification(a.ctx, msg); err != nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Slack notify failed: "+err.Error())
	}
}
```

Ensure `auto_refresh.go` imports `gmailapi "google.golang.org/api/gmail/v1"` (it already imports
`gmailapi` for the prepend; if not, add it). `fmt`/`strings` are already imported.

- [ ] **Step 5: Wire it in `performAutoRefreshTick`**

In `internal/tui/auto_refresh.go`, in `performAutoRefreshTick`, right after the
`if len(newIDs) == 0 { return }` guard, add:
```go
	go a.notifyNewMailSlack(newIDs)
```

- [ ] **Step 6: Run tests + build**

Run: `go test ./internal/tui/ -run 'TestBuildNewMailSlackMessage|TestAutoRefresh' -v && go build ./...`
Expected: PASS, build success.

- [ ] **Step 7: Commit**

```bash
git add internal/config/config.go internal/tui/auto_refresh.go internal/tui/auto_refresh_test.go
git commit -m "feat(tui): auto-refresh posts a Slack notification on new mail (opt-in)"
```

---

### Task 3: Docs + full verification

**Files:**
- Modify: `docs/CONFIGURATION.md`

- [ ] **Step 1: Document `notify_slack`**

In `docs/CONFIGURATION.md`, in the Auto-Refresh section, add to the table / prose:
`notify_slack` (boolean, default `false`) — when auto-refresh detects new mail, also post a Slack
message (count + subject/sender) to your default Slack channel. Requires Slack configured
(`slack.enabled` + a channel). Run `:config migrate` to add the key.

- [ ] **Step 2: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass.

- [ ] **Step 3: Suite + leak check**

Run: `go test ./internal/services/ ./internal/config/ ./internal/tui/ ./test/helpers/ 2>&1 | tail -5`
Expected: all `ok`.

- [ ] **Step 4: Build + commit docs**

Run: `make build`
```bash
git add docs/CONFIGURATION.md
git commit -m "docs: document auto_refresh.notify_slack"
```

(Live E2E — enable auto_refresh + notify_slack with Slack configured; receive mail; a Slack message
lists the new emails — deferred to the user's E2E sweep.)

---

## Self-review notes

- **Spec coverage:** `SendNotification` + `defaultSlackWebhook` (Task 1), `NotifySlack` config +
  `buildNewMailSlackMessage` + `notifyNewMailSlack` + wiring in `performAutoRefreshTick` (Task 2),
  docs + verification (Task 3). All spec sections mapped.
- **Type consistency:** `defaultSlackWebhook(cfg *config.Config) (string, error)`,
  `SendNotification(ctx, text string) error`, `buildNewMailSlackMessage([]*gmailapi.Message) string`,
  `notifyNewMailSlack(newIDs []string)`, `AutoRefreshConfig.NotifySlack` — consistent across tasks.
- **No placeholders:** every code step shows full code; commands have expected output.
- **Threading:** `notifyNewMailSlack` runs in a goroutine off the tick; failures are non-fatal
  warnings; the gate short-circuits when disabled.
