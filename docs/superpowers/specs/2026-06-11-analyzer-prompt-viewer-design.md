# Action Plan — effective analyzer-prompt viewer (`v`)

**Date:** 2026-06-11
**Status:** Approved (design)

## Problem

The user cannot see what prompt the analyzer actually sends to the LLM. The prompt is assembled
from three pieces (saved rules → base prompt → per-batch `{{messages}}` payload), so it is hard
to reason about classification quality without seeing the assembled result.

## Goal

A key (`v`) in the Action Plan opens an in-place, read-only view of the **effective prompt** —
the saved-rules block plus the base prompt (default or the one the panel was opened with), with
`{{messages}}` shown literally and a note explaining how it is filled per batch.

## Decisions (confirmed with user)

- Trigger key: **`v`** (view), inside the Action Plan panel.
- Content: rules block + base prompt + literal `{{messages}}` + a one-line note. The static
  parts the user controls, shown exactly. No live per-email sample (bodies are not retained in
  panel state, so a sample would be approximate — out of scope).
- In-place body-swap (consistent with the move chooser / prompt preview), not a floating modal.

## Architecture

### A) Service — `internal/services/inbox_analyzer_service.go`

New pure method on the `InboxAnalyzerService` interface and `InboxAnalyzerServiceImpl`:

```go
// BuildPromptPreview returns the assembled analyzer prompt (user-rules block + base prompt),
// with {{messages}} left literal. It mirrors the assembly Analyze performs so the preview can
// never drift from what is actually sent. No AI call, no network.
BuildPromptPreview(opts InboxAnalyzerOptions) string
```

Implementation reuses the existing helpers exactly as `Analyze` does:
`base := opts.CustomPromptText; if blank → defaultAnalyzerPrompt`, then
`return prependUserRules(base, opts.UserRules)`. (It does NOT call `buildBatchPrompt` /
`buildBatchPayload`, so `{{messages}}` stays as a placeholder.)

### B) TUI — `internal/tui/action_plan_prompt.go` (new)

`func (a *App) showActionPlanPromptView(state *actionPlanState)`:

- Gather `UserRules` from `GetAnalyzerRulesService().ListRules` (empty if the service is nil),
  build `services.InboxAnalyzerOptions{CustomPromptText: state.customPromptText, UserRules: …,
  BodyCharLimit: a.Config.InboxAnalyzer.BodyCharLimit}`, and call
  `a.GetInboxAnalyzerService().BuildPromptPreview(opts)`.
- Compose the displayed text: a header note —
  "`{{messages}}` is replaced per batch with each email's subject / from / body (up to N chars)"
  (N = the configured `BodyCharLimit`, or "snippet only" when `include_body` is false) — followed
  by the assembled prompt (escaped via `tview.Escape`).
- Body-swap the panel container's tree ↔ a read-only, scrollable, word-wrapped `*tview.TextView`
  (the `showActionPlanMoveChooser` pattern: RemoveItem(tree)/RemoveItem(footer),
  AddItem(textview)/AddItem(footer); restore mirrors it). `a.currentFocus = "action_plan_prompt"`.
- Esc (TextView input capture) → restore the tree (`renderActionPlanPanel(state)`),
  `a.currentFocus = "action_plan"`.

### C) Integration — `internal/tui/action_plan.go` + `internal/tui/keys.go`

- In `actionPlanInputCapture`, after the analyzing-gate, add: `if ev.Rune() == 'v' { … return nil }`
  → `showActionPlanPromptView(state)`. (Blocked during analysis, like the quick actions.)
- Add a `v prompt` hint to the Action Plan footer text.
- Add `"action_plan_prompt"` to the global pass-through sentinel line in `keys.go`.

## Error handling / threading

- `ListRules` is a fast read; the body-swap and Esc restore are synchronous (no `QueueUpdateDraw`),
  matching the move chooser.
- If the analyzer service is nil, the `v` handler does nothing (the panel only exists when the
  analyzer is configured, so this is defensive).
- No rules / nil rules service → preview shows base prompt + note only.

## Testing

- `TestBuildPromptPreview`: with `UserRules` → output contains `## User preferences` and the rule
  text; with `CustomPromptText` set → output starts from that base (not the default); with both
  blank → output is the default analyzer prompt; output always contains `{{messages}}`.
- `TestActionPlanPromptViewSwap` (mirror `TestActionPlanMoveInlineSwap`): `v` swaps the tree for a
  TextView and sets `currentFocus == "action_plan_prompt"`; Esc restores the tree and
  `currentFocus == "action_plan"`.

## Out of scope

- A live per-email payload sample (bodies not retained in panel state).
- Editing the prompt from the viewer (read-only; editing is `:plan with-prompt` / the prompt
  configurator).
