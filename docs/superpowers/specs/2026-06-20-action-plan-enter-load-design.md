# Action Plan — Enter loads the email into the list + reader

**Date:** 2026-06-20
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `feat/action-plan-enter-load`
**Issue:** #51

## Goal

When the focus is on an **email node** in the Inbox Action Plan tree, pressing **Enter** (or `→`)
selects that email in the inbox message list and loads its content into the reader pane — **without
moving focus** — so the user can then **Tab** to the reader and read the body. Today Enter on an
email node is a no-op; you cannot inspect an email's body from the Action Plan.

## Current behavior (verified)

In `internal/tui/action_plan.go`, the tree's `SetInputCapture` handles `KeyEnter`/`KeyRight`
(~line 673): if the current node's reference is a category (`int`), it toggles expand/collapse; if
it is an `emailRef` (`{catIndex int, msgID string}`), the key falls through and is consumed — a
no-op. **Space** (`Keys.BulkSelect`) toggles an email's excluded state; `m` moves; `i` shows the
prompt viewer. None of these change.

The inbox list (`internal/tui/keys.go:1465`) already wires `SetSelectionChangedFunc`: when the table
selection changes, it calls `go a.showMessageWithoutFocus(id)` (loads content in the background,
using the cache/preloader, **without stealing focus** and without clobbering `currentMessageID` in a
racy way) and updates the `Message N/M` status. `table.Select(row, 0)` does not change application
focus — only the table cursor.

## Behavior

When Enter/`→` is pressed and the current node is an `emailRef`:

1. **Resolve the email's row in the list.** Find `msgID` in `a.ids`.
2. **If found** → `list.Select(rowForIndex, 0)` (table row = index + 1 because row 0 is the header).
   This moves the list cursor to the email and triggers the existing `SetSelectionChangedFunc`,
   which loads the body into the reader. As a safety net against tview not firing
   `SelectionChangedFunc` on a programmatic select, also call `go a.showMessageWithoutFocus(msgID)`
   directly — it is idempotent (guards against stale updates by captured ID; cache makes a repeat
   cheap).
3. **If not found** (the email was archived/moved out of the list from within the Action Plan) →
   `go a.showMessageWithoutFocus(msgID)` directly, so Enter still loads it for reading; the list
   cursor is left unchanged.
4. **Focus stays on the Action Plan tree** in all cases (no `SetFocus`). The user presses **Tab** to
   reach the reader (the reader is already in the focus ring when the Action Plan is open, per the
   v1.14.0 Tab-cycling rework).

Category nodes keep their current Enter behavior (expand/collapse). Email-node Enter is purely
additive.

## Architecture

This is a TUI-only change — no new service or fetch logic. It reuses two existing primitives:
`(*tview.Table).Select` and `App.showMessageWithoutFocus(id)`. The email→row lookup is a small linear
scan over `a.ids` (same data the list is built from). Add a tiny helper
`messageRowInList(msgID string) (row int, ok bool)` (or inline) to keep the handler readable.

The new handling slots into the existing `KeyEnter`/`KeyRight` branch in
`actionPlanInputCapture`, adding an `emailRef` case alongside the existing category case.

## Error handling / threading

- Content load runs in a goroutine (as everywhere else that calls `showMessageWithoutFocus`); no
  blocking, no `QueueUpdateDraw` in ESC/cleanup paths.
- Empty/zero state: if `a.ids` is empty or `msgID` is empty, fall back to the no-op (do nothing).
- No focus change, so no risk of the ESC-deadlock class tied to focus juggling.

## Testing

- Unit-test the pure lookup helper `messageRowInList`: returns the correct 0-based-plus-header row
  for an id present in `a.ids`; returns `ok=false` for an absent id and for an empty list.
- The Enter→load wiring itself depends on the live tview table + reader and is covered by the
  manual E2E smoke test (see Definition of Done), consistent with how other Action Plan key paths
  are verified.

## Out of scope (YAGNI)

- Changing Space, category-Enter, `m`, `i`, or any other Action Plan key.
- Moving focus to the reader on Enter (explicitly rejected — focus stays on the Action Plan).
- Any change to how the Action Plan is built or analyzed.
- Marking the email read / changing `currentMessageID` semantics beyond what the existing
  `SetSelectionChangedFunc` path already does.

## Definition of Done

- [ ] Enter/`→` on an email node selects it in the list (when present) and loads it in the reader.
- [ ] Email not in the list still loads in the reader by id; list cursor unchanged.
- [ ] Focus remains on the Action Plan tree; Tab reaches the reader.
- [ ] Category-node Enter unchanged; Space/`m`/`i` unchanged.
- [ ] `messageRowInList` helper unit-tested.
- [ ] In-app `:help` / Action Plan footer mentions Enter-to-open if other email-node keys are listed
      there (keep parity with how Space/`m`/`i` are surfaced).
- [ ] `make pre-commit-check` green; manual E2E smoke test on the user's Mac.
