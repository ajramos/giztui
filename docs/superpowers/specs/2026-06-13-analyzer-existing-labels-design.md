# Analyzer uses existing labels (idea A)

**Date:** 2026-06-13
**Status:** Approved (design)

## Problem

The analyzer's `label` action tells the LLM to "provide a short kebab-case label", so it invents
label names that usually don't exist in the user's mailbox — and dispatch (`resolveOrCreateLabelID`)
then creates them. The user wants the analyzer to reuse their existing labels when one fits, only
inventing a new label when none does.

## Goal

Give the analyzer the list of the user's existing (user-created) labels and instruct it to prefer
an exact match for the `label` action, inventing a new label only when none fits (marking it as
new). Dispatch behavior is unchanged (still creates a label if it doesn't exist).

## Decisions (confirmed with user)

- Pass the user's **existing user labels** (not system labels) into the prompt; the LLM prefers
  them, may still propose a new one when none fits (noting it's new).
- Dispatch unchanged: `resolveOrCreateLabelID` keeps creating a missing label (the constrained
  prompt makes invention rare).

## Architecture

### A) Options field — `internal/services/interfaces.go`

`InboxAnalyzerOptions` gains `AvailableLabels []string` (existing user-label names).

### B) Prompt injection — `internal/services/inbox_analyzer_service.go`

New helper, mirroring `prependUserRules`:
```go
// prependAvailableLabels prepends an "## Existing labels" block when labels are present, telling
// the model to prefer an exact existing label for the "label" action. Empty → prompt unchanged.
func prependAvailableLabels(promptText string, labels []string) string
```
Block content (only when `len(labels) > 0`):
```
## Existing labels
These labels already exist in the mailbox: <comma-joined names>
For the "label" action, PREFER an exact name from this list when one fits. Only invent a NEW label
when none fits; if you do, keep it short kebab-case and note "(new)" in the category description.
```
`Analyze` applies it to the base prompt (after `prependUserRules`, before batching), and
`BuildPromptPreview` applies it too (so the `v` viewer reflects it). Order: rules block, then
labels block, then base prompt — both are independent prepends.

### C) TUI population — `internal/tui/action_plan.go`

- A small helper `userLabelNames(labelService) []string`: `ListLabels` → keep only `Type == "user"`
  → return names. Errors → empty slice (degrade gracefully; analyzer just won't get the list).
- In `openActionPlanWithText`'s analyze flow, populate `opts.AvailableLabels = a.userLabelNames(...)`
  alongside the existing options.
- In `showActionPlanPromptView` (the `v` viewer), populate `AvailableLabels` the same way so the
  preview matches what Analyze sends.

### D) Dispatch — unchanged

`resolveOrCreateLabelID` keeps its find-or-create behavior. No change.

## Error handling

- `ListLabels` failure → `userLabelNames` returns empty → no labels block → analyzer behaves as
  before (may invent). Non-fatal.
- Off the UI thread (analyze runs in a goroutine); the viewer call is synchronous on the UI thread
  but `ListLabels` is a quick read (label list is small/cached).

## Testing

- `TestPrependAvailableLabels`: with labels → block contains "## Existing labels", the names, and
  the "PREFER an exact name" instruction; empty → returns the base prompt unchanged.
- `TestBuildPromptPreview_Labels`: `BuildPromptPreview` with `AvailableLabels` set includes the
  labels block; without it, no block.
- `userLabelNames` filtering is covered indirectly (system labels excluded) — if easily unit-testable
  with a stub LabelService, add `TestUserLabelNames_FiltersSystem`; otherwise rely on the prompt tests.

## Out of scope

- Changing dispatch to not auto-create (user chose to keep creating).
- A confirmation prompt before creating a new label (deferred; current auto-create stays).
- Mapping/normalizing near-miss label names (the LLM prefers exact existing names; no fuzzy match).
