# Action Plan analyzer — richer context (email body, not just snippet)

**Date:** 2026-06-11
**Status:** Approved (design)

## Problem

The inbox Action Plan analyzer classifies each unread email using only its **subject, sender,
and Gmail snippet** (~100 chars). That is too little context, so the LLM's categorization is
poor. The user wants the analyzer to also see (at least) the first ~1000 characters of each
email's body.

## Goal

Feed the analyzer the email **body** (plain text, truncated) in addition to subject/sender, so
classification quality improves — without making the analyzer prohibitively slow or expensive
for large inboxes or low-context local models.

## Decisions (confirmed with user)

- **Opt-in + configurable** (not always-on, not heuristic): a config flag plus a character
  limit. Default **on**, default limit **1000**. The user can lower the limit or disable it
  entirely for slow local models / huge inboxes.
- Bodies are fetched **only for the messages actually analyzed** (`BatchSize × MaxBatches`),
  concurrently, with progress feedback.
- Failure to fetch a body degrades gracefully to that message's snippet (never aborts).
- The analyzer service stays decoupled from Gmail (depends only on `aiService`); body fetching
  lives in `EmailService` and the TUI orchestration.

## Architecture

### Config — `internal/config/config.go`

`InboxAnalyzerConfig` gains two fields:

```go
IncludeBody   bool `json:"include_body"`    // include plain-text body in analyzer context
BodyCharLimit int  `json:"body_char_limit"` // max body chars per email
```

`DefaultInboxAnalyzerConfig()` sets `IncludeBody: true`, `BodyCharLimit: 1000`.
`include_body=false` reproduces today's behavior (snippet only, zero extra cost). A
non-positive `BodyCharLimit` falls back to 1000 at use-site.

### Data model — `internal/services/interfaces.go`

`AnalyzerMessage` gains `Body string`. Subject/From/Snippet are still populated from in-memory
metadata (no Gmail calls). `Body` is populated only when `IncludeBody` is on.

### New service method — `EmailService` (`interfaces.go` + `email_service.go`)

```go
// GetMessagePlainTexts fetches plain-text bodies for the given message IDs concurrently.
// Returns id -> plain text; IDs that fail to fetch are simply absent from the map.
GetMessagePlainTexts(ctx context.Context, ids []string, maxWorkers int) (map[string]string, error)
```

Implementation: `s.gmailClient.GetMessagesParallel(ids, maxWorkers)` then
`gmail.ExtractPlainText(msg)` per non-nil result, keyed by `msg.Id`. A nil/failed message is
skipped (absent from the map). `maxWorkers <= 0` lets the client pick its conservative default.

### Payload rendering — `internal/services/inbox_analyzer_service.go`

New pure helper:

```go
// truncateForAnalyzer collapses runs of whitespace (incl. newlines) to single spaces and
// cuts to at most limit runes on a rune boundary. limit <= 0 returns the collapsed text.
func truncateForAnalyzer(text string, limit int) string
```

`buildBatchPayload` renders each message using its `Body` (run through `truncateForAnalyzer`
with the configured limit) when `Body` is non-empty, otherwise the existing `Snippet`. Format:

```
1. Subject: <subject> | From: <from>
   <up to limit chars of plain-text body, single-spaced>
```

When `Body` is empty the line keeps today's compact single-line form
(`1. Subject: … | From: … | <snippet>`).

The character limit reaches `buildBatchPayload` via a new field on `InboxAnalyzerOptions`
(`BodyCharLimit int`); `0` means "bodies already truncated upstream / no extra trim".

### Orchestration — `internal/tui/action_plan.go`

In the analyze flow (where `buildAnalyzerMessages` / `buildAnalyzerMessagesForSelection` feed
the analyzer), when `cfg.InboxAnalyzer.IncludeBody`:

1. Cap the `[]AnalyzerMessage` slice to `BatchSize × MaxBatches` (bounds the fetch to what is
   actually analyzed — the service would cap batches anyway).
2. Collect those IDs, call `emailService.GetMessagePlainTexts(ctx, ids, 0)` on the existing
   analysis goroutine, reporting progress via `a.GetErrorHandler().ShowProgress(...)`
   ("Fetching email bodies…") dispatched off the UI thread.
3. For each message, set `Body` = its fetched plain text (the final trim to `BodyCharLimit`
   happens in `buildBatchPayload` via `truncateForAnalyzer`). Messages whose body did not come
   back keep `Body == ""` and fall back to `Snippet`.

Pass `BodyCharLimit` through `InboxAnalyzerOptions`.

## Error handling / threading

- Body fetch runs on the existing analysis goroutine (off the UI thread); all ErrorHandler
  progress calls follow the codebase rule (never inside `QueueUpdateDraw`).
- Partial fetch failure is non-fatal: missing bodies fall back to snippet.
- `IncludeBody=false` short-circuits the whole fetch (no behavior change vs today).

## Testing

- `truncateForAnalyzer` (table): whitespace/newline collapse; cut on rune boundary (multi-byte);
  `limit <= 0` returns collapsed-but-untrimmed; empty input.
- `buildBatchPayload`: uses `Body` (truncated) when present; falls back to `Snippet` when `Body`
  is empty; respects `BodyCharLimit`.
- `GetMessagePlainTexts`: with a stubbed Gmail client, returns `id -> text` for successful
  fetches and omits failed IDs. (Follows existing email_service test patterns.)
- `DefaultInboxAnalyzerConfig` includes `IncludeBody=true`, `BodyCharLimit=1000`.

## Out of scope

- Caching fetched bodies (the analyzer is a one-shot; reuse across runs is a separate idea).
- Changing batch sizing automatically based on body length (the user tunes `batch_size`
  manually for local models; documented).
- The effective-prompt viewer (separate backlog item #3) — though this work makes that viewer
  more useful.

## Docs

- Update `docs/` analyzer/config references and the `--setup` flow / example config to mention
  `include_body` and `body_char_limit`.
