# Analyzer rules UI → in-place (fix the hanging modal)

**Date:** 2026-06-11
**Status:** Approved (design)

## Problem

The analyzer-rules manager (`:plan rules` → `openAnalyzerRulesManager`) opens as a floating
`Pages` overlay and **hangs / can't be closed** — the user has to kill the app. The quick
"remember rule" input (`Ctrl+R` → `showRememberRuleModal`) is the same floating-modal class.
The user wants both to behave like the rest of the pickers (in-place, reliably closeable).

## Root cause

Floating `Pages` overlays interact badly with the global `Application.SetInputCapture`, which
runs *before* the focused primitive's input capture. Without a `currentFocus` sentinel
pass-through in `keys.go`, the global handler swallows the overlay's `Esc`. This is the same
class of bug already fixed for the prompt preview and the action-plan move chooser by converting
floating modals to in-place panels with a `currentFocus` sentinel. (The existing
`keys.go` guard `if a.Pages.HasPage(actionPlanRulePage) || HasPage(analyzerRulesPage)` only
passes through while the Action Plan is active, so `:plan rules` invoked from the inbox is not
covered.)

## Decisions (confirmed with user)

- Convert **both** entry points to in-place, for a uniform experience.
- The rules **manager** becomes a side-panel picker (like the other pickers / the Action Plan).
- "Add rule" inside the manager is an **embedded input** (body-swap), not a sub-modal.
- The `Ctrl+R` quick-remember becomes an **embedded input inside the Action Plan panel**
  (body-swap of the tree), since it is only reachable from there.

## Architecture

### A) Rules manager (`:plan rules`) → side-panel picker

- New picker constant: `PickerAnalyzerRules ActivePicker = "analyzer_rules"` (in `app.go`).
- `openAnalyzerRulesManager` (in `internal/tui/action_plan_rules.go`) is rewritten to:
  - Guard `svc == nil` (as today).
  - If the Action Plan is active (`a.actionPlanState != nil`), close it first — the rules
    picker and the Action Plan both occupy `a.labelsView` in `contentSplit`.
  - Build a container `*tview.Flex` (bordered, themed via `GetComponentColors("ai")`) holding a
    `*tview.List` of rules plus a footer (`a add · d delete · Esc close`).
  - Mount it as `a.labelsView` in `a.views["contentSplit"].(*tview.Flex)` (visible weight),
    `setActivePicker(PickerAnalyzerRules)`, `a.currentFocus = "analyzer_rules"`, focus the list.
- Close (Esc): resize `a.labelsView` to 0, `setActivePicker(PickerNone)`, focus the inbox list,
  `a.currentFocus = "list"`, `updateFocusIndicators("list")` — mirroring the tail of
  `closeActionPlanPanel`.
- `a` (add): body-swap the list ↔ a `*tview.InputField` inside the same container (the
  prompt-preview / move-chooser pattern). `a.currentFocus = "analyzer_rules_add"` while editing.
  Enter → `svc.SaveRule` off-thread, then reload the list and swap back. Esc → swap back without
  saving.
- `d` (delete): unchanged behavior — delete the current rule off-thread, then reload.
- Remove the floating `a.Pages.AddPage(analyzerRulesPage, …)`.

### B) `Ctrl+R` quick-remember → embedded input in the Action Plan

- `showRememberRuleModal(suggestion string)` (same file) is rewritten to body-swap the Action
  Plan tree ↔ a pre-seeded `*tview.InputField` inside the panel container, mirroring
  `showActionPlanMoveChooser` (body-swap, restore closure, Esc capture, focus sentinel).
- `a.currentFocus = "action_plan_rule"` while editing. Enter → `svc.SaveRule` off-thread, then
  swap back to the tree (`renderActionPlanPanel`). Esc → swap back without saving.
- Remove the floating `a.Pages.AddPage(actionPlanRulePage, …)`.

### C) `keys.go` + cleanup

- Add `"analyzer_rules"`, `"analyzer_rules_add"`, and `"action_plan_rule"` to the existing
  global pass-through line that already lists `"prompt_preview"` / `"action_plan_move"`.
- Remove the now-dead guard `if a.Pages.HasPage(actionPlanRulePage) || HasPage(analyzerRulesPage)`
  in `keys.go`.
- Remove the `a.Pages.RemovePage(actionPlanRulePage)` / `RemovePage(analyzerRulesPage)` calls in
  `closeActionPlanPanel`.
- Remove the `actionPlanRulePage` / `analyzerRulesPage` constants if they become unused.

## Error handling / threading

- All `ErrorHandler.Show*` calls dispatched with `go` (they wrap `QueueUpdateDraw`; calling
  synchronously from key/SetSelectedFunc/SetDoneFunc handlers deadlocks — the recurring rule).
- Esc/close paths are synchronous UI swaps (no `QueueUpdateDraw`), matching the move chooser.
- `svc.SaveRule` / `DeleteRule` run off-thread; the list reload after them uses
  `QueueUpdateDraw` from that goroutine (as the current delete path already does).

## Testing

- `TestAnalyzerRulesPickerSwap` (mirror `TestActionPlanMoveInlineSwap`): open the manager →
  `currentFocus == "analyzer_rules"` and the container is mounted as `labelsView`; Esc →
  `currentFocus == "list"` and `activePicker == PickerNone`.
- `TestRememberRuleInlineSwap`: invoking the Ctrl+R inline input body-swaps the tree for an
  input and sets `currentFocus == "action_plan_rule"`; Esc restores the tree and
  `currentFocus == "action_plan"`.
- Existing `AnalyzerRulesService` tests (Save/List/Delete) remain the coverage for persistence.

## Out of scope

- No change to how rules are interpreted by the LLM or stored (persistence layer untouched).
- No new keybinding (reuses `:plan rules` and `Ctrl+R`).
