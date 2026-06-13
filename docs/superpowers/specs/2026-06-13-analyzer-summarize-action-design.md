# Analyzer "summarize" action — combined AI digest (idea B)

**Date:** 2026-06-13
**Status:** Approved (design)

## Problem

The Action Plan only offers archive / trash / mark-read / label / none. The user wants a category
for emails they'd like a **quick AI digest** of (not read fully, not discarded) — e.g. newsletters
or threads worth skimming.

## Goal

Add a `summarize` action the analyzer can assign to a category; dispatching it generates a single
combined AI digest of that category's emails, shown in an in-place panel inside the Action Plan
(Markdown-rendered, streaming), with Esc returning to the tree.

## Decisions (confirmed with user)

- **Output:** one **combined digest** (single AI call over all the category's emails), not per-email.
- **View:** an **in-place body-swap panel** in the Action Plan (like the `v` prompt viewer / move
  chooser), Markdown-rendered, Esc returns.
- Reuse existing infra: body-swap pattern, `GetMessagePlainTexts` (bodies), `renderPromptResult`
  (Markdown), `AIService.GenerateSummaryStream` (streaming).

## Architecture

### A) Prompt — `internal/services/inbox_analyzer_prompt.txt`

Add to the "exact set" of actions:
```
  "summarize" — worth a quick AI digest; not full reading, not discarding
```
(The JSON schema's `action` field already accepts any string; the analyzer service's
`normalizePriority`/merge are action-agnostic, so no parser change is needed.)

### B) Action metadata — `internal/tui/action_plan.go`

- `actionVerbLabel("summarize")` → `"summarize"`.
- `actionRuleVerbShort("summarize")` → `"digest"` (footer short form).
- `actionKeyHint("summarize")` → `a.Keys.Summarize` (the `y` key) so the footer shows
  `y to digest (N)` for a summarize category.

### C) Dispatch trigger — `internal/tui/action_plan.go` (`actionPlanInputCapture`)

In the action-key `switch key { ... }`, add:
```go
	case a.Keys.Summarize:
		a.dispatchActionPlanSummarize(state)
		return nil
```
This summarizes the currently-focused category's checked emails (works on any category; it's the
action the analyzer suggests for ones worth skimming). Gated behind the existing
`if state.analyzing.Load() { return nil }` like the other quick actions.

### D) `dispatchActionPlanSummarize` — new `internal/tui/action_plan_summarize.go`

```go
func (a *App) dispatchActionPlanSummarize(state *actionPlanState)
```
- Resolve the focused category (`currentActionPlanCategory`); `checkedIDs` for its messages. If
  none checked → `go ...ShowWarning("nothing to summarize")` and return.
- Body-swap the tree for a read-only, scrollable `*tview.TextView` showing "⏳ Summarizing N
  emails…"; `a.currentFocus = "action_plan_summary"`; Esc input-capture → restore (mirrors
  `showActionPlanMoveChooser`'s restore).
- In a goroutine:
  - Fetch bodies via `emailService.GetMessagePlainTexts(ctx, ids, 0)` (degrade to snippet from
    `state.metaByID` when a body is missing).
  - Build the digest input with a pure helper `buildSummarizeInput(ids, bodies, metaByID, limit)`
    → a numbered "Subject / From / body(≤limit)" blob (reusing `truncateForAnalyzer`-style trim;
    `limit` = `a.Config.InboxAnalyzer.BodyCharLimit`).
  - Call `aiService.GenerateSummaryStream(ctx, blob, options, onToken)` where `onToken` appends
    **directly** to the TextView (NEVER `QueueUpdateDraw` — streaming-callback rule), guarded by
    `ctx.Err()`/state-still-active checks.
  - On completion, render the final text through `renderPromptResult` (Markdown) and set it.
- Esc restores the tree (`renderActionPlanPanel(state)`), `currentFocus = "action_plan"`.

The AI digest framing (a short "what's new across these emails" instruction) is passed via the
summary content/options; reuse `SummaryOptions` defaults. Errors → `go ...ShowError` + restore.

### E) keys.go

Add `"action_plan_summary"` to the global pass-through sentinel line (next to
`action_plan_prompt` etc.) so the TextView owns its keys.

## Error handling / threading

- Body fetch + AI call run on a worker goroutine (off the UI thread). `GetMessagePlainTexts` and
  the final `ShowError`/`ShowWarning` are safe there.
- Streaming tokens update the TextView via **direct `SetText`/append**, never `QueueUpdateDraw`
  (matches the AI-summary streaming pattern and avoids the ESC-deadlock class).
- Esc restore is synchronous (no `QueueUpdateDraw`).
- No checked emails → warning, no panel.

## Testing

- `TestActionVerb_Summarize`: `actionVerbLabel("summarize")=="summarize"`,
  `actionRuleVerbShort("summarize")=="digest"`, and `actionKeyHint("summarize")` returns the
  configured Summarize key.
- `TestBuildSummarizeInput`: builds a numbered blob from IDs + bodies (+ snippet fallback when a
  body is absent), truncated to the limit.
- `TestActionPlanSummarizeSwap` (mirror `TestActionPlanMoveInlineSwap`): `dispatchActionPlanSummarize`
  with a stub AIService body-swaps the tree for a TextView and sets `currentFocus ==
  "action_plan_summary"`; Esc restores the tree (`currentFocus == "action_plan"`). (Use a stub
  AIService whose `GenerateSummaryStream` returns immediately so the test is deterministic.)

## Out of scope

- Per-email summaries (combined digest only — decided).
- Persisting/caching the digest (one-shot view).
- A bulk Gmail action on summarize categories (summarize only produces a digest; the user can then
  archive/etc. via the normal keys).
