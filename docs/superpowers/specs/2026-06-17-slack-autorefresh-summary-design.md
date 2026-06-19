# Slack Auto-Refresh Per-Email Summary — Design

**Date:** 2026-06-17
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `feat/slack-autorefresh-summary`

## Goal

When auto-refresh detects new mail and posts a Slack notification, optionally include a short
AI-generated summary of each new email, so the recipient gets the gist without opening Gmail.

## Background

The auto-refresh Slack notification already exists. Today `App.notifyNewMailSlack`
(`internal/tui/auto_refresh.go`) fetches metadata for the new message IDs and builds a digest via
the pure helper `buildNewMailSlackMessage(metas, urlFor)`:

```
📬 N new email(s):
• <gmailLink|Subject> — From
…and M more   (when more than 10)
```

It then sends the text via `SlackService.SendNotification` to the default channel webhook.

The Slack service *also* already knows how to summarize an email with AI: `FormatStyle: "summary"`
in `formatEmailForSlack`/`formatSummaryMessage` calls `aiService.ApplyCustomPrompt` with
`config.Slack.GetSummaryPrompt()` (max ~50 words). That capability is reused here.

Two relevant facts:
- The current digest path fetches **metadata only** (`GetMessagesMetadataParallel`). Summaries need
  the **body**, so the new path must fetch message content for the emails it summarizes.
- The list is capped at `maxList = 10` lines. The summary cap (default 5) is a *separate*, smaller
  limit applied to the first emails.

## Decisions (from brainstorming)

- **Scope:** only the auto-refresh Slack notification (not the manual forward panel).
- **Shape:** a per-email summary — each summarized email gets its line plus an italic summary line
  underneath. (Not one combined summary; not summarize-only-when-single.)
- **Cap:** configurable, default 5. Emails beyond the cap appear as a normal line (no summary).
- **Prompt:** reuse `config.Slack.GetSummaryPrompt()`. No new prompt config.
- **Approach:** A — business logic lives in `SlackService` (service-first). The TUI only gathers IDs
  and calls the service.
- **Default:** opt-in (`slack_summary: false`), because summaries add AI cost per refresh cycle.

## Configuration

Two new fields on `AutoRefreshConfig` (`internal/config/config.go`):

```go
type AutoRefreshConfig struct {
    Enabled           bool   `json:"enabled"`
    Interval          string `json:"interval"`
    NotifySlack       bool   `json:"notify_slack"`
    SlackSummary      bool   `json:"slack_summary"`        // include per-email AI summary in the Slack notification
    SlackSummaryLimit int    `json:"slack_summary_limit"`  // max emails summarized per cycle (default 5)
}
```

`DefaultConfig()`:
- `SlackSummary: false`
- `SlackSummaryLimit: 5`

**Self-migration (Definition of Done):** existing `config.json` files on other machines must surface
the two new keys through the config self-migration path — a default in code is not enough.

**Zero-value guard:** an absent JSON `int` deserializes to `0`, which would mean "summarize 0
emails". At the use site, if `SlackSummary` is enabled and `SlackSummaryLimit <= 0`, treat it as 5.

## Architecture (Approach A — service-first)

### Service interface (`internal/services/interfaces.go`)

Add to `SlackService`:

```go
SendNewMailDigest(ctx context.Context, messageIDs []string, opts NewMailDigestOptions) error
```

New options type:

```go
type NewMailDigestOptions struct {
    Summaries    bool                          // generate AI summaries
    SummaryLimit int                           // max summaries; clamp <=0 → 5
    LinkFor      func(messageID string) string // optional Gmail hyperlink builder; nil = no link
}
```

### Service implementation (`internal/services/slack_service.go`)

`SendNewMailDigest`:
1. Return `nil` early when `messageIDs` is empty.
2. Resolve the default webhook via `defaultSlackWebhook(s.config)`; return its error if unconfigured.
3. Fetch metadata (subject/from) for all IDs.
4. If `opts.Summaries`: compute `n = min(clamp(opts.SummaryLimit), maxList)` and, for the first `n`
   emails, fetch the body and summarize via `s.aiService.ApplyCustomPrompt(ctx, prompt, vars)` where
   `prompt = s.config.Slack.GetSummaryPrompt()` with `{{body}}` and `{{max_words}}=50` substituted
   (mirroring `formatSummaryMessage`).
5. Build the message text with a pure helper `buildNewMailDigest(items, linkPresent)`.
6. Send via `s.sendToSlack(ctx, SlackMessage{Text: text}, webhook)`.

Pure helper (testable, ported from the TUI's `buildNewMailSlackMessage`):

```go
type digestItem struct {
    ID      string
    Subject string
    From    string
    Link    string // pre-resolved (via opts.LinkFor), "" if none
    Summary string // "" when not summarized
}

func buildNewMailDigest(items []digestItem) string
```

Format (cap `maxList = 10`):
```
📬 N new email(s):
• <link|Subject> — From
   _Summary text._          (only when item.Summary != "")
…and M more                 (when N > 10)
```

### TUI (`internal/tui/auto_refresh.go`)

`notifyNewMailSlack` simplifies to:

```go
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

`buildNewMailSlackMessage` and the in-TUI metadata fetch are removed (logic moves to the service).
The existing `buildNewMailSlackMessage` test moves to the service test for the new helper.

## Error handling

- **Per-email AI/body-fetch failure:** skip that email's summary (keep the normal line), log a
  warning. The notification must never fail because a summary could not be produced.
- **Total send failure:** surfaced via `ShowWarning` in the TUI (unchanged from today).
- **No Slack channel configured:** `defaultSlackWebhook` error returned; TUI shows the warning.

## Testing

- `buildNewMailDigest` (pure):
  - list cap of 10 + "…and M more"
  - hyperlink present vs absent
  - summary line rendered only for items with a summary, placed under the right line
  - empty input
- Orchestration with a mocked `AIService`:
  - respects `SummaryLimit` (only first N summarized)
  - clamp `<=0 → 5`
  - falls back to no-summary line when `ApplyCustomPrompt` returns an error
  - `Summaries: false` produces today's metadata-only digest

## Definition of Done

- [ ] Config fields + `DefaultConfig()` values + self-migration of the two keys
- [ ] Zero-value clamp (`SlackSummaryLimit <= 0 → 5`)
- [ ] `SlackService.SendNewMailDigest` + `NewMailDigestOptions` + pure `buildNewMailDigest`
- [ ] TUI `notifyNewMailSlack` rewired; old helper/fetch removed
- [ ] In-app `:help` updated with `slack_summary` / `slack_summary_limit`
- [ ] Config docs updated (auto-refresh section)
- [ ] Tests (pure helper + mocked AIService orchestration)
- [ ] `make pre-commit-check` green

## Out of scope (YAGNI)

- Summaries in the manual Slack forward panel (already available via `FormatStyle: "summary"`).
- Combined/single-summary mode and "summarize only when one email" mode.
- A separate auto-refresh summary prompt (reuse the Slack summary prompt).
- Threads/attachments for the full body alongside the summary.
