# Slack Auto-Refresh Per-Email Summary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the auto-refresh Slack notification optionally include a short AI summary of each new email.

**Architecture:** Business logic lives in `SlackService` (service-first). A new `SendNewMailDigest` method fetches metadata for all new IDs, fetches bodies + AI-summarizes the first N (configurable, default 5, opt-in), and builds the Slack text via pure helpers. The TUI's `notifyNewMailSlack` only gathers IDs and calls the service. Config defaults flow automatically because `LoadConfig` unmarshals user JSON over `DefaultConfig()`.

**Tech Stack:** Go, `google.golang.org/api/gmail/v1` (aliased `gmailapi`/`gmail` in client), existing `AIService.ApplyCustomPrompt`, `testify` mocks (`internal/services/mocks`).

---

## Reference facts (verified in code)

- `internal/config/config.go:724` — `LoadConfig` does `cfg := DefaultConfig()` then `json.Unmarshal(data, cfg)`. Absent JSON keys keep their `DefaultConfig()` value → no migration code needed. The `<=0 → 5` clamp guards an explicit `0`.
- `internal/services/slack_service.go` — `SlackServiceImpl` has `client *gmail.Client`, `config *config.Config`, `aiService AIService`. It already has `extractEmailMetadata(*gmailapi.Message) map[string]string`, `extractEmailBody(*gmailapi.Message) string`, `sendToSlack(...)`, and `defaultSlackWebhook(cfg)`.
- `internal/services/interfaces.go:88` — `ApplyCustomPrompt(ctx, prompt string, variables map[string]string) (string, error)`.
- `internal/services/interfaces.go:105` — `SlackService` interface (currently ends with `SendNotification`). `SlackForwardOptions` at line 297.
- `internal/gmail/client.go` — `GetMessagesMetadataParallel(ids, maxWorkers) ([]*gmailapi.Message, error)` (headers only) and `GetMessagesParallel(ids, maxWorkers) ([]*gmailapi.Message, error)` (full payload). Both return results in input order, `nil` for failures.
- `internal/tui/auto_refresh.go:249` — `buildNewMailSlackMessage(metas, urlFor)` (cap `maxList = 10`). `gmailapi` is still used at line 225, so removing this helper does not orphan the import.
- `internal/tui/auto_refresh.go:276` — `notifyNewMailSlack(newIDs []string)`.
- `internal/tui/auto_refresh_test.go:92` — `TestBuildNewMailSlackMessage` (to be removed; its coverage moves to the service).
- No generated mock exists for `SlackService` (tested via concrete struct), so adding a method is safe.
- `config.Slack.GetSummaryPrompt()` returns the summary prompt with `{{body}}` and `{{max_words}}` placeholders.

---

## Task 1: Config fields + defaults

**Files:**
- Modify: `internal/config/config.go` (`AutoRefreshConfig` struct ~line 243; `DefaultConfig()` ~line 476)
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestAutoRefreshSummaryDefaults(t *testing.T) {
	c := DefaultConfig()
	if c.AutoRefresh.SlackSummary {
		t.Errorf("SlackSummary should default to false (opt-in)")
	}
	if c.AutoRefresh.SlackSummaryLimit != 5 {
		t.Errorf("SlackSummaryLimit default = %d, want 5", c.AutoRefresh.SlackSummaryLimit)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestAutoRefreshSummaryDefaults -v`
Expected: FAIL — `c.AutoRefresh.SlackSummary` / `SlackSummaryLimit` undefined (compile error).

- [ ] **Step 3: Add the struct fields**

In `internal/config/config.go`, change `AutoRefreshConfig` to:

```go
// AutoRefreshConfig controls opt-in background polling of the inbox for new mail.
type AutoRefreshConfig struct {
	Enabled           bool   `json:"enabled"`
	Interval          string `json:"interval"`             // Go duration string, e.g. "5m"; clamped to a 1m minimum
	NotifySlack       bool   `json:"notify_slack"`         // also post a Slack notification when new mail is detected
	SlackSummary      bool   `json:"slack_summary"`        // include a per-email AI summary in the Slack notification
	SlackSummaryLimit int    `json:"slack_summary_limit"`  // max emails summarized per refresh cycle (default 5)
}
```

- [ ] **Step 4: Set the defaults**

In `DefaultConfig()` (currently `AutoRefresh: AutoRefreshConfig{Enabled: false, Interval: "5m"}`), change to:

```go
		AutoRefresh:   AutoRefreshConfig{Enabled: false, Interval: "5m", SlackSummary: false, SlackSummaryLimit: 5},
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestAutoRefreshSummaryDefaults -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add auto_refresh slack_summary + slack_summary_limit"
```

---

## Task 2: `summaryCount` pure helper (clamp + cap)

**Files:**
- Modify: `internal/services/slack_service.go`
- Test: `internal/services/slack_service_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/services/slack_service_test.go`:

```go
func TestSummaryCount(t *testing.T) {
	cases := []struct {
		name              string
		summaries         bool
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestSummaryCount -v`
Expected: FAIL — `summaryCount` undefined.

- [ ] **Step 3: Implement the helper**

Add to `internal/services/slack_service.go` (near the top-level helpers, e.g. above `defaultSlackWebhook`):

```go
// digestMaxList caps how many emails are listed in the new-mail Slack digest.
const digestMaxList = 10

// summaryCount returns how many of `total` new emails should be AI-summarized,
// applying the opt-in flag, the <=0 → 5 clamp, the digestMaxList cap, and min(total).
func summaryCount(summaries bool, limit, total int) int {
	if !summaries {
		return 0
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > digestMaxList {
		limit = digestMaxList
	}
	if limit > total {
		return total
	}
	return limit
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestSummaryCount -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/slack_service.go internal/services/slack_service_test.go
git commit -m "feat(services): add summaryCount helper for slack digest cap"
```

---

## Task 3: `digestItem` + `buildNewMailDigest` pure formatter

**Files:**
- Modify: `internal/services/slack_service.go`
- Test: `internal/services/slack_service_test.go`

- [ ] **Step 1: Write the failing test** (ports + extends `TestBuildNewMailSlackMessage`)

Add to `internal/services/slack_service_test.go`:

```go
func TestBuildNewMailDigest(t *testing.T) {
	// Two items, no links, no summaries.
	got := buildNewMailDigest([]digestItem{
		{Subject: "Hello", From: "a@x.com"},
		{Subject: "World", From: "b@x.com"},
	})
	if !strings.Contains(got, "📬 2 new email(s):") {
		t.Errorf("missing header: %q", got)
	}
	if !strings.Contains(got, "• Hello — a@x.com") || !strings.Contains(got, "• World — b@x.com") {
		t.Errorf("missing plain lines: %q", got)
	}

	// Link + summary rendering.
	linked := buildNewMailDigest([]digestItem{
		{Subject: "Hi", From: "a@x.com", Link: "https://mail.google.com/x", Summary: "Short recap."},
	})
	if !strings.Contains(linked, "• <https://mail.google.com/x|Hi> — a@x.com") {
		t.Errorf("missing hyperlink line: %q", linked)
	}
	if !strings.Contains(linked, "\n   _Short recap._") {
		t.Errorf("missing italic summary line: %q", linked)
	}

	// Summary line absent when Summary == "".
	noSum := buildNewMailDigest([]digestItem{{Subject: "Hi", From: "a@x.com"}})
	if strings.Contains(noSum, "_") {
		t.Errorf("should not render summary line when empty: %q", noSum)
	}

	// Cap at 10 with "…and N more".
	many := make([]digestItem, 12)
	for i := range many {
		many[i] = digestItem{Subject: "S", From: "f@x.com"}
	}
	capped := buildNewMailDigest(many)
	if !strings.Contains(capped, "…and 2 more") {
		t.Errorf("missing overflow line: %q", capped)
	}

	// Empty input.
	if got := buildNewMailDigest(nil); !strings.Contains(got, "📬 0 new email(s):") {
		t.Errorf("empty digest wrong: %q", got)
	}
}
```

Note: `strings` is already imported in `slack_service_test.go`? If not, add `"strings"` to its imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestBuildNewMailDigest -v`
Expected: FAIL — `digestItem` / `buildNewMailDigest` undefined.

- [ ] **Step 3: Implement the type + formatter**

Add to `internal/services/slack_service.go`:

```go
// digestItem is one new-mail row for the Slack notification.
type digestItem struct {
	Subject string
	From    string
	Link    string // pre-resolved Gmail hyperlink URL, "" if none
	Summary string // AI summary, "" when not summarized
}

// buildNewMailDigest formats the new-mail Slack notification, capping listed rows at digestMaxList.
// A non-empty Summary renders as an indented italic line under the row.
func buildNewMailDigest(items []digestItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📬 %d new email(s):", len(items))
	for i, it := range items {
		if i >= digestMaxList {
			fmt.Fprintf(&b, "\n…and %d more", len(items)-digestMaxList)
			break
		}
		if it.Link != "" {
			fmt.Fprintf(&b, "\n• <%s|%s> — %s", it.Link, it.Subject, it.From)
		} else {
			fmt.Fprintf(&b, "\n• %s — %s", it.Subject, it.From)
		}
		if it.Summary != "" {
			fmt.Fprintf(&b, "\n   _%s_", it.Summary)
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestBuildNewMailDigest -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/slack_service.go internal/services/slack_service_test.go
git commit -m "feat(services): add buildNewMailDigest formatter for slack notification"
```

---

## Task 4: `summarizeForDigest` (AI call + fallback)

**Files:**
- Modify: `internal/services/slack_service.go`
- Test: `internal/services/slack_service_test.go`

- [ ] **Step 1: Write the failing test** (uses the generated `mocks.AIService`)

Add to `internal/services/slack_service_test.go` (imports needed: `context`, `errors`, `github.com/ajramos/giztui/internal/config`, `github.com/ajramos/giztui/internal/services/mocks`, `github.com/stretchr/testify/mock`):

```go
func TestSummarizeForDigest(t *testing.T) {
	cfg := &config.Config{}
	cfg.Slack = config.DefaultSlackConfig() // provides GetSummaryPrompt with {{body}}/{{max_words}}

	// Success path: returns trimmed AI output.
	ai := &mocks.AIService{}
	ai.On("ApplyCustomPrompt", mock.Anything, mock.Anything, mock.Anything).Return("  Recap here.  ", nil)
	s := &SlackServiceImpl{config: cfg, aiService: ai}
	if got := s.summarizeForDigest(context.Background(), "long body text"); got != "Recap here." {
		t.Errorf("summary = %q, want %q", got, "Recap here.")
	}

	// AI error → "" (caller keeps the plain line).
	aiErr := &mocks.AIService{}
	aiErr.On("ApplyCustomPrompt", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("boom"))
	sErr := &SlackServiceImpl{config: cfg, aiService: aiErr}
	if got := sErr.summarizeForDigest(context.Background(), "body"); got != "" {
		t.Errorf("on AI error want \"\", got %q", got)
	}

	// nil aiService or empty body → "".
	sNil := &SlackServiceImpl{config: cfg, aiService: nil}
	if got := sNil.summarizeForDigest(context.Background(), "body"); got != "" {
		t.Errorf("nil aiService want \"\", got %q", got)
	}
	if got := s.summarizeForDigest(context.Background(), "   "); got != "" {
		t.Errorf("empty body want \"\", got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestSummarizeForDigest -v`
Expected: FAIL — `summarizeForDigest` undefined.

- [ ] **Step 3: Implement the method**

Add to `internal/services/slack_service.go`:

```go
// summarizeForDigest returns a short AI summary of body, or "" when unavailable
// (nil AI service, empty body, or AI error) so the caller falls back to the plain row.
func (s *SlackServiceImpl) summarizeForDigest(ctx context.Context, body string) string {
	if s.aiService == nil || strings.TrimSpace(body) == "" {
		return ""
	}
	const maxWords = "50"
	prompt := s.config.Slack.GetSummaryPrompt()
	prompt = strings.ReplaceAll(prompt, "{{body}}", body)
	prompt = strings.ReplaceAll(prompt, "{{max_words}}", maxWords)
	variables := map[string]string{"body": body, "max_words": maxWords}
	out, err := s.aiService.ApplyCustomPrompt(ctx, prompt, variables)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestSummarizeForDigest -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/slack_service.go internal/services/slack_service_test.go
git commit -m "feat(services): add summarizeForDigest with graceful AI fallback"
```

---

## Task 5: `SendNewMailDigest` method + interface

**Files:**
- Modify: `internal/services/interfaces.go` (`SlackService` interface ~line 105; add `NewMailDigestOptions` near `SlackForwardOptions` ~line 297)
- Modify: `internal/services/slack_service.go`

- [ ] **Step 1: Add the options type + interface method**

In `internal/services/interfaces.go`, add near `SlackForwardOptions`:

```go
// NewMailDigestOptions configures the auto-refresh new-mail Slack notification.
type NewMailDigestOptions struct {
	Summaries    bool                          // generate per-email AI summaries
	SummaryLimit int                           // max emails summarized; <=0 clamps to 5
	LinkFor      func(messageID string) string // optional Gmail hyperlink builder; nil = no link
}
```

In the `SlackService` interface, add after `SendNotification(ctx context.Context, text string) error`:

```go
	// SendNewMailDigest posts an auto-refresh notification for the given new message IDs to the
	// default channel, optionally including a per-email AI summary (capped by opts.SummaryLimit).
	SendNewMailDigest(ctx context.Context, messageIDs []string, opts NewMailDigestOptions) error
```

- [ ] **Step 2: Implement the method**

Add to `internal/services/slack_service.go`:

```go
// SendNewMailDigest posts the new-mail notification to the default Slack channel. When
// opts.Summaries is set, the first summaryCount(...) emails get an AI summary line; AI/fetch
// failures degrade gracefully to a plain row.
func (s *SlackServiceImpl) SendNewMailDigest(ctx context.Context, messageIDs []string, opts NewMailDigestOptions) error {
	if len(messageIDs) == 0 {
		return nil
	}
	webhook, err := defaultSlackWebhook(s.config)
	if err != nil {
		return err
	}

	metas, err := s.client.GetMessagesMetadataParallel(messageIDs, 10)
	if err != nil {
		return fmt.Errorf("failed to fetch new mail metadata: %w", err)
	}

	// Summarize the first n emails (n already clamped + capped).
	n := summaryCount(opts.Summaries, opts.SummaryLimit, len(metas))
	summaries := make([]string, len(metas))
	if n > 0 {
		full, ferr := s.client.GetMessagesParallel(messageIDs[:n], n)
		if ferr == nil {
			for i := 0; i < n && i < len(full); i++ {
				if full[i] == nil {
					continue
				}
				summaries[i] = s.summarizeForDigest(ctx, s.extractEmailBody(full[i]))
			}
		}
	}

	items := make([]digestItem, 0, len(metas))
	for i, m := range metas {
		if m == nil {
			continue
		}
		hdr := s.extractEmailMetadata(m)
		link := ""
		if opts.LinkFor != nil && m.Id != "" {
			link = opts.LinkFor(m.Id)
		}
		summary := ""
		if i < len(summaries) {
			summary = summaries[i]
		}
		items = append(items, digestItem{
			Subject: hdr["subject"],
			From:    hdr["from"],
			Link:    link,
			Summary: summary,
		})
	}

	return s.sendToSlack(ctx, SlackMessage{Text: buildNewMailDigest(items)}, webhook)
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/services/...`
Expected: no output (success). The concrete `*SlackServiceImpl` now satisfies the extended `SlackService` interface.

- [ ] **Step 4: Run the services tests**

Run: `go test ./internal/services/ 2>&1 | tail -5`
Expected: `ok  github.com/ajramos/giztui/internal/services`

- [ ] **Step 5: Commit**

```bash
git add internal/services/interfaces.go internal/services/slack_service.go
git commit -m "feat(services): add SlackService.SendNewMailDigest with per-email summaries"
```

---

## Task 6: Rewire the TUI to call the service

**Files:**
- Modify: `internal/tui/auto_refresh.go` (remove `buildNewMailSlackMessage` ~line 249; rewrite `notifyNewMailSlack` ~line 276)
- Modify: `internal/tui/auto_refresh_test.go` (remove `TestBuildNewMailSlackMessage` ~line 92)

- [ ] **Step 1: Remove the moved test**

Delete `TestBuildNewMailSlackMessage` (and its local `mk` helper if used only there) from `internal/tui/auto_refresh_test.go`. Its coverage now lives in `TestBuildNewMailDigest`.

- [ ] **Step 2: Remove the old formatter**

Delete the entire `buildNewMailSlackMessage` function from `internal/tui/auto_refresh.go` (the block starting at the `// buildNewMailSlackMessage formats...` comment through its closing `}`).

- [ ] **Step 3: Rewrite `notifyNewMailSlack`**

Replace the body of `notifyNewMailSlack` in `internal/tui/auto_refresh.go` with:

```go
// notifyNewMailSlack posts a Slack notification about newly-detected mail, when enabled.
func (a *App) notifyNewMailSlack(newIDs []string) {
	if !a.Config.AutoRefresh.NotifySlack || !a.Config.Slack.Enabled {
		return
	}
	svc := a.GetSlackService()
	if svc == nil || len(newIDs) == 0 {
		return
	}
	var linkFor func(string) string
	if a.gmailWebService != nil {
		linkFor = a.gmailWebService.GenerateGmailWebURL
	}
	opts := services.NewMailDigestOptions{
		Summaries:    a.Config.AutoRefresh.SlackSummary,
		SummaryLimit: a.Config.AutoRefresh.SlackSummaryLimit,
		LinkFor:      linkFor,
	}
	if err := svc.SendNewMailDigest(a.ctx, newIDs, opts); err != nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Slack notify failed: "+err.Error())
	}
}
```

- [ ] **Step 4: Fix imports**

`notifyNewMailSlack` no longer fetches metadata, but `gmailapi` is still used at `auto_refresh.go:225` (`a.messagesMeta`), so keep it. Add `"github.com/ajramos/giztui/internal/services"` to `auto_refresh.go` imports if not already present (it is needed for `services.NewMailDigestOptions`). In `auto_refresh_test.go`, remove the `gmailapi` import only if `TestBuildNewMailSlackMessage` was its sole user — let the build tell you.

- [ ] **Step 5: Verify build + tests**

Run: `go build ./... && go test ./internal/tui/ ./internal/services/ 2>&1 | tail -8`
Expected: build succeeds; both packages `ok`. If `auto_refresh_test.go` reports `"gmailapi" imported and not used`, remove that import line and re-run.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/auto_refresh.go internal/tui/auto_refresh_test.go
git commit -m "refactor(tui): route new-mail Slack notification through SendNewMailDigest"
```

---

## Task 7: Update in-app `:help` and docs

**Files:**
- Modify: `internal/tui/app.go` (help text near the `:autorefresh` line ~2422)
- Modify: config docs (find with grep below)

- [ ] **Step 1: Find where auto-refresh config is documented**

Run: `grep -rln "notify_slack\|auto_refresh" docs/ README.md`
Expected: a docs file listing auto-refresh config keys (e.g. a config reference). Note the path for Step 3.

- [ ] **Step 2: Extend the `:help` line**

In `internal/tui/app.go`, just after the existing `:autorefresh` help line (~2422), add:

```go
	fmt.Fprintf(&help, "    %-18s     config: auto_refresh.slack_summary + slack_summary_limit (AI summary of new mail in Slack)\n", "")
```

(Match the surrounding `Fprintf` column formatting; the leading `%-18s` with an empty string keeps the description column aligned under `:autorefresh`.)

- [ ] **Step 3: Document the config keys**

In the docs file found in Step 1, add the two keys under the `auto_refresh` section:

```json
"auto_refresh": {
  "enabled": false,
  "interval": "5m",
  "notify_slack": false,
  "slack_summary": false,
  "slack_summary_limit": 5
}
```

With a sentence: `slack_summary` includes a short AI summary of each new email in the Slack notification (reuses the Slack summary prompt). `slack_summary_limit` caps how many emails are summarized per refresh cycle (default 5; values ≤0 are treated as 5).

- [ ] **Step 4: Verify build**

Run: `go build ./internal/tui/...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go docs/
git commit -m "docs: document auto_refresh slack_summary keys in help and config reference"
```

---

## Task 8: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Run the canonical pre-commit check**

Run: `make pre-commit-check`
Expected: `All pre-commit checks passed!` (fmt + vet + golangci-lint + essential tests).

- [ ] **Step 2: Run the full test suite**

Run: `make test 2>&1 | grep -E "^(ok|FAIL)" | grep -v "no test files"`
Expected: all `ok`, no `FAIL`.

- [ ] **Step 3: Build the binary**

Run: `make build`
Expected: `Built build/giztui v1.15.0`.

- [ ] **Step 4: Finish the branch**

Use the superpowers:finishing-a-development-branch skill to present merge/PR options. Do NOT push, tag, or release without explicit user confirmation (project rule: "commit" ≠ "publish").

---

## Self-Review

**Spec coverage:**
- Config fields + defaults → Task 1. ✓
- Self-migration → covered by LoadConfig-over-DefaultConfig (documented in Reference facts); no code needed. ✓
- Zero-value clamp (`<=0 → 5`) → `summaryCount`, Task 2. ✓
- `SendNewMailDigest` + `NewMailDigestOptions` + `buildNewMailDigest` → Tasks 3, 5. ✓
- Reuse `GetSummaryPrompt()` → `summarizeForDigest`, Task 4. ✓
- TUI rewire + old helper removed → Task 6. ✓
- Per-email AI/fetch failure degrades gracefully → `summarizeForDigest` returns ""; `SendNewMailDigest` ignores `GetMessagesParallel` error → Tasks 4, 5. ✓
- `:help` + docs → Task 7. ✓
- Tests (pure helpers + mocked AIService) → Tasks 2, 3, 4. ✓
- `make pre-commit-check` green → Task 8. ✓

**Type consistency:** `digestItem` fields (Subject/From/Link/Summary), `NewMailDigestOptions` fields (Summaries/SummaryLimit/LinkFor), `summaryCount(bool,int,int)`, `buildNewMailDigest([]digestItem)`, `summarizeForDigest(ctx,string)`, `SendNewMailDigest(ctx,[]string,NewMailDigestOptions)` — used consistently across Tasks 2–6. ✓

**Placeholder scan:** no TBD/TODO; every code step shows full code. ✓
