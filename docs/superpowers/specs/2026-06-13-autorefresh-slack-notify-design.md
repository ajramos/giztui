# Auto-refresh → Slack notification (idea F)

**Date:** 2026-06-13
**Status:** Approved (design)

## Problem

Auto-refresh tells you about new inbox mail only inside the app (toast / `📬N` counter). The user
wants an optional Slack ping when new mail arrives, so they hear about it away from the TUI.

## Goal

When auto-refresh detects new mail and the feature is enabled, post one Slack message to the
default channel listing the new emails (count + subject · sender).

## Decisions (confirmed with user)

- **Message:** count + list — `📬 N new email(s):` then `• Subject — Sender` per email (capped).
- **Activation:** opt-in `auto_refresh.notify_slack` (default false); only when Slack is enabled and
  a channel is configured. Uses the **default** Slack channel.

## Architecture

### A) SlackService — `internal/services/slack_service.go` (+ interface)

Add `SendNotification(ctx context.Context, text string) error`:
- Resolve the default channel's webhook with a pure helper
  `defaultSlackWebhook(cfg *config.Config) (string, error)`: the first channel with `Default==true`,
  else the first channel; error if none.
- `sendToSlack(ctx, SlackMessage{Text: text}, webhook)` (reuse the existing poster).
- Add `SendNotification` to the `SlackService` interface.

### B) Config — `internal/config/config.go`

`AutoRefreshConfig` gains `NotifySlack bool` (`json:"notify_slack"`, default false). Surfaced via
`:config migrate`.

### C) TUI — `internal/tui/auto_refresh.go`

- Pure helper `buildNewMailSlackMessage(metas []*gmailapi.Message) string`:
  `📬 N new email(s):\n• Subject — From` per non-nil message, capped at 10 with a trailing
  `…and K more` when there are more.
- `notifyNewMailSlack(newIDs []string)`: gate on `a.Config.AutoRefresh.NotifySlack &&
  a.Config.Slack.Enabled && a.GetSlackService() != nil`; fetch metadata for `newIDs`
  (`a.Client.GetMessagesMetadataParallel(newIDs, 10)`); build the message; `go
  a.GetSlackService().SendNotification(a.ctx, msg)` (errors → `go ...ShowWarning`, non-fatal).
- Call `go a.notifyNewMailSlack(newIDs)` from `performAutoRefreshTick` right after `newIDs` is
  found non-empty (before the safe/unsafe branch), so it fires whether mail is prepended or just
  counted. (A small extra metadata fetch in the safe path — newIDs is small; acceptable.)

### D) Docs

`docs/CONFIGURATION.md` auto-refresh section: document `notify_slack` (requires Slack configured;
posts to the default channel). Note `:config migrate` adds it.

## Error handling / threading

- Notification runs in a goroutine off the tick; failures are non-fatal (warning, never blocks
  auto-refresh).
- No channel / Slack disabled / flag off → no-op (the gate short-circuits before any work).

## Testing

- `TestDefaultSlackWebhook`: a config with a `Default` channel returns its webhook; with no default
  returns the first; with no channels returns an error.
- `TestBuildNewMailSlackMessage`: formats `📬 N new email(s):` + `• Subject — From` lines from
  metadata; caps at 10 with `…and K more`; nil entries skipped.
- The HTTP POST is already covered by `sendToSlack` (existing).

## Out of scope

- Per-notification channel selection (uses the default channel — decided).
- Rich Slack blocks/attachments (plain text only).
- Throttling beyond "only fires when a tick finds new mail".
