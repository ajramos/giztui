# Action Plan — Bulk move a whole category

**Date:** 2026-06-10
**Status:** Approved (design)

## Problem

In the Action Plan panel, `m` moves/recategorizes a single email (when the cursor is on an
email node). There is no way to move an *entire* category's emails to another category or
action in one step — the user has to move them one at a time.

## Goal

Pressing `m` while the cursor is on a **category header** (or the **Read manually** group)
moves *all* of that group's emails to a chosen destination (another category, or one of the
standard actions: Archive / Trash / Mark read / Keep). The per-email `m` behavior is unchanged.

## Decisions (confirmed with user)

- **Trigger:** reuse `m`. Branch on the tree node's reference type — `emailRef` → single move
  (unchanged); `int` → whole-group move.
- **Scope:** move *all* emails in the group, **preserving each email's excluded/checked flag**.
  Moving is reorganization, not dispatching, so exclusions ride along and are still honored when
  the destination group is later dispatched.
- **Read manually:** the `-1` "Read manually" pseudo-node is also bulk-movable (re-filing the
  pile is a real workflow).
- **UI:** extract the shared body-swap destination chooser so single and bulk paths share one
  focus/restore implementation (the focus-wedge + deadlock-prone code we already fixed once).

## Architecture

### Core logic — `internal/tui/action_plan_move.go`

New pure function:

```go
// applyActionPlanBulkMove reassigns every message in the source group (a category by index,
// or ReadManually when srcCatIdx == -1) to target, returning the number moved. Excluded flags
// are NOT touched: msgIDs are unchanged so state.excluded keys still apply to the moved emails.
func applyActionPlanBulkMove(plan *services.ActionPlan, metaByID map[string]*gmailapi.Message, srcCatIdx int, target moveTarget) int
```

Implementation: collect the source group's message IDs up front, then call the existing
`applyActionPlanMove` once per ID. `applyActionPlanMove` resolves the target by name/action
(not index) and prunes empty categories, so looping over a pre-collected ID list is
index-shift-safe. Returns `len(ids)`.

Source IDs:
- `srcCatIdx == -1` → `plan.ReadManually[*].ID`
- `0 <= srcCatIdx < len(plan.Categories)` → `plan.Categories[srcCatIdx].MessageIDs`
- otherwise → empty (no-op)

### UI — `internal/tui/action_plan_move.go`

Extract the body-swap chooser from `showActionPlanMoveInline` into:

```go
func (a *App) showActionPlanMoveChooser(state *actionPlanState, srcCatName, title string, onChosen func(target moveTarget))
```

Owns: build the destinations `List` from `actionPlanMoveTargets(plan, srcCatName)`, the
`restore()` closure, Esc input-capture, `currentFocus = "action_plan_move"` sentinel,
swap-in/swap-out of tree↔list and footer, and calling `onChosen(target)` then `restore()` on
selection (with the existing stale-state guards: out-of-range index, `state.plan == nil`,
`a.actionPlanState != state`).

Two thin callers:

- `showActionPlanMoveInline(state, srcCatIdx, msgID)` — title `➫ Move "<subject>" to`;
  `onChosen` = `applyActionPlanMove(...)` + `delete(state.excluded, msgID)` + success toast.
  (Behavior identical to today.)
- `showActionPlanBulkMoveInline(state, srcCatIdx)` — title `➫ Move "<name>" (N) to`
  (name = category name, or "Read manually"; N = group size); `onChosen` =
  `applyActionPlanBulkMove(...)` (does **not** touch `state.excluded`) + success toast
  `Moved N emails to <target> — applies when you dispatch that group`.

`srcCatName` for the bulk chooser is the category's name (so it's excluded from its own
destination list); `""` for Read manually (nothing to exclude).

### Key routing — `internal/tui/action_plan.go`

In `actionPlanInputCapture`, replace the `emailRef`-only `m` handler with a type switch:

```go
if ev.Rune() == 'm' && cur != nil {
    switch ref := cur.GetReference().(type) {
    case emailRef:
        a.showActionPlanMoveInline(state, ref.catIndex, ref.msgID)
        return nil
    case int:
        a.showActionPlanBulkMoveInline(state, ref)
        return nil
    }
}
```

## Error handling / threading

- Success toasts dispatched with `go a.GetErrorHandler().ShowSuccess(...)` — the handler wraps
  `QueueUpdateDraw` and this runs on the UI goroutine from the key handler (synchronous call
  would deadlock; same pattern as the existing single move).
- Empty source group: `showActionPlanBulkMoveInline` early-returns with a
  `go ...ShowWarning("Category is empty — nothing to move")` rather than opening a chooser.

## Testing

- `TestApplyActionPlanBulkMove` (table-driven), asserting on the plan only (no tview):
  - category → another category: all IDs land in target, source pruned, `excluded` map untouched
    (an excluded ID stays excluded).
  - category → action ("archive"): IDs land in first category with that action (or a new one).
  - Read manually (`-1`) → category: all ReadManually IDs moved, `ReadManually` emptied.
  - empty / out-of-range source: returns 0, plan unchanged.
- Shared chooser swap remains covered by the existing `TestActionPlanMoveInlineSwap`
  (in `internal/tui/action_plan_test.go`); update it if the extraction changes the entry point.

## Out of scope

- No new keybinding or command alias (reuses `m`; command parity for `m`/move is a separate
  existing concern).
- No change to how/when actions are dispatched — move only reassigns within the plan.
