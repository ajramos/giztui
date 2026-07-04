# Action Plan: Confirm Whole Plan (`c`) — Design Spec

**Issue:** #54
**Date:** 2026-07-04
**Status:** Approved by user (key `c`, two-press confirmation)

## Summary

Add a shortcut in the Inbox Action Plan panel that applies the entire plan in one go: for every category, run its suggested `Action` on all of that category's non-excluded emails. Today the user must walk each category and press its action key by hand; this adds a single "accept the plan as-is" step.

## Behavior

### Trigger

- New configurable key **`confirm_plan`**, default `"c"`, handled inside `actionPlanInputCapture` (`internal/tui/action_plan.go:702`).
- Placed **after** the `state.analyzing.Load()` guard — blocked while analysis is running, like the other quick-actions.
- `"c"` is free in the panel today (taken keys: `a`/`t`/`r`/`l` per-category via `a.Keys.*`, `y` = Summarize, Space = exclude, `m` = move, `i` = view prompt, Ctrl+R = remember rule).

### What gets applied

For each category in `state.plan.Categories`, in plan order:

- Skip categories with `Action == "none"` (includes the "read manually" style categories).
- Within a category, apply only non-excluded messages: `checkedIDs(cat.MessageIDs, state.excluded)` — the same filter `executeActionPlanAction` uses.
- Skip categories whose filtered ID list is empty.
- Manual moves are already reflected in `state.plan.Categories` (the move feature mutates the plan), so they are respected for free.

If, after filtering, **nothing** remains to apply, show `ShowInfo`: "Nothing to apply — all emails are excluded or categories have no action" and do NOT enter the confirmation state.

### Two-press confirmation flow

State: add `confirmPending bool` to `actionPlanState` (no mutex needed — only touched on the UI goroutine inside the input capture and panel lifecycle).

1. **First press of `c`:** compute the summary (counts per action across applicable categories) and show a persistent status message via `SetPersistentStatus`:
   `Apply plan: 12 archive, 3 trash, 5 label — press 'c' again to confirm, Esc cancels`
   (only the non-zero action counts are listed; label count aggregates all label categories). Set `confirmPending = true`.
2. **Second press of `c`** while `confirmPending`: clear the persistent status, set `confirmPending = false`, execute (see below).
3. **Any other key** while `confirmPending` (navigation, Space, `m`, `a`/`t`/`r`/`l`, `y`, `i`, Ctrl+R): clear `confirmPending` + clear the persistent status, then let the key do its normal job.
4. **Esc** while `confirmPending`: cancel the confirmation ONLY (clear flag + status), do NOT close the panel. The next Esc closes the panel as usual. This is a small addition at the top of the input capture, before the existing Esc branch — synchronous, no `QueueUpdateDraw` (ESC rule).
5. Re-analysis or closing the panel resets `confirmPending` implicitly (fresh state / panel gone). `closeActionPlanPanel` already clears persistent status? — verify; if not, clear it there when `confirmPending`.

### Execution (non-blocking, sequential)

One goroutine (mirrors `executeActionPlanAction`'s shape, `action_plan.go:854-901`):

```
snapshot := buildPlanApply(state)   // computed BEFORE launching, on the UI goroutine
go func() {
    applied, failed := 0, 0
    for i, item := range snapshot.items {
        // progress prefix: "Applying plan (i+1/len)"
        err := <run the same bulk op as executeActionPlanAction for item.action>
        if err != nil { failed++; ShowError "Action failed on <cat>: err"; continue }  // continue with the rest
        applied++; appliedMessages += len(item.ids)
        a.QueueUpdateDraw(func() { if a.actionPlanState == state { a.removeActionPlanCategory(state, item.catName) } })
    }
    ClearPersistentMessage()
    if failed == 0 { ShowSuccess "✓ Plan applied: N categories, M messages" }
    else { ShowWarning "Plan applied: N categories, M messages (K failed)" }
}()
```

- **Sequential**, not parallel: one bulk op at a time, so `bulkProgress` output stays coherent and Gmail quota isn't hammered.
- Per-category ops are exactly the existing ones: `emailService.BulkArchive` / `BulkMarkAsRead` / `BulkTrash` / `applyActionPlanLabel` (which handles strict-labels via `resolveOrCreateLabelID`).
- A label category failing (e.g. missing label in strict mode) does NOT abort the rest — report and continue.
- Successfully applied categories are removed from the tree as they complete (same visual as per-category apply). Guard every tree mutation with `a.actionPlanState == state` inside `QueueUpdateDraw` (panel may have been closed mid-run).
- Refactor note: extract the `switch action { ... }` bulk-op dispatch from `executeActionPlanAction` into a helper `runActionPlanBulkOp(action, ids, label string) error` reused by both the per-category and whole-plan paths, so they cannot drift.

### Pure helper for testability

`buildPlanApply(plan *services.InboxActionPlan, excluded map[string]bool) planApplySummary` in a new file `internal/tui/action_plan_apply.go`:

```go
type planApplyItem struct {
    catName string
    action  string   // "archive" | "mark_read" | "trash" | "label"
    label   string   // when action == "label"
    ids     []string // non-excluded message IDs
}
type planApplySummary struct {
    items  []planApplyItem
    counts map[string]int // action -> message count (labels aggregated under "label")
    total  int            // total messages
}
func buildPlanApply(plan *services.InboxActionPlan, excluded map[string]bool) planApplySummary
func (s planApplySummary) statusLine(confirmKey string) string // "Apply plan: 12 archive, ..."
```

Pure function: no App, no services — unit-testable without the TUI harness.

## Command parity

`:action-plan apply` / `:plan apply` / `:ap apply` (extend `executeActionPlanCommand`, `internal/tui/commands.go:2195`):

- Panel open + analysis finished → same first-press behavior: compute summary, set `confirmPending`, show the persistent status ("press 'c' again to confirm"), and set `a.cmd.focusOverride = "keep"` so focus stays on the panel (known focus-steal bug class).
- Panel not open, or analysis still running → `ShowError`: "Open the action plan first (:plan)" / "Analysis still running".
- Update the `:action-plan` usage/error message and the command registry help card (`command_completion.go`) to mention `apply` (registry `help` + `completePlanArg`-style completer if one exists for `:plan` — verify; add `apply` to argument completion alongside `with-prompt` / `rules`).

## Config & docs (Definition of Done)

1. `ConfirmPlan string \`json:"confirm_plan"\`` in `KeysConfig` (`internal/config/config.go` ~line 401, next to `ViewPrompt`), comment: "Action plan: confirm & apply the whole plan".
2. Default `"c"` in `DefaultConfig()` (~line 654 block).
3. Config self-migration: `MissingDefaultKeys` (`internal/config/migrate.go`) diffs the user file against defaults generically — verify the new key surfaces automatically with the default; no bespoke migration code expected.
4. In-app `?` help: add `c  Confirm & apply whole plan` to the Action Plan section.
5. `docs/KEYBOARD_SHORTCUTS.md`: Action Plan table row + `:plan apply` command row.
6. Key-conflict validation: if `KeysConfig` validation warns on duplicate single-letter keys (see the `Summarize` uppercase logic ~line 912), make sure `confirm_plan` participates like any other key (no special casing expected).

## Undo

Each underlying bulk op keeps whatever undo support it already has (last-operation undo). Undoing the *whole plan* in one step is **out of scope** (per issue #54).

## Error handling

- All user feedback via `GetErrorHandler()` (Show* from goroutines directly; `go ...` when called from the UI goroutine key handler — same discipline as `executeActionPlanAction:863`).
- No `QueueUpdateDraw` in the Esc/cancel path (synchronous).
- Panel closed mid-execution: bulk ops finish against Gmail (they already run detached), tree updates are guarded by `a.actionPlanState == state`; final summary still shown in the status bar.

## Testing

1. **Unit (new `action_plan_apply_test.go`):** `buildPlanApply` — skips `none`, filters excluded, drops empty categories, aggregates label counts, empty-plan → zero total; `statusLine` formatting (single action, multiple actions, key name interpolation).
2. **TUI test:** two-press state machine — first `c` sets `confirmPending` + status, second `c` triggers execution path, interleaved navigation key clears pending, Esc while pending does not close the panel.
3. `make pre-commit-check` green before claiming done; full `make test` as a separate step before merge (leak detector).

## Out of scope

- Whole-plan single-step undo.
- Parallel category execution.
- Any change to per-category apply behavior.
