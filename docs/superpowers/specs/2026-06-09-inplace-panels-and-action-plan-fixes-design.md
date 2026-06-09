# In-Place Panels + Action Plan Correctness Fixes — Design

Date: 2026-06-09
Branch: `feat/action-plan-rework`
Status: approved (pending written-spec review)

## Problem

Three user-reported issues, two of which share a root cause class (floating
overlay focused on a non-`List`/`InputField` primitive escapes the picker's own
key handling):

1. **Prompt preview won't close.** Pressing `p` opens the prompt picker; `Ctrl+P`
   opens a preview. Neither `Esc` nor `Ctrl+P` closes it — the app must be killed.
2. **Action plan "move" hangs.** Pressing `m` on an email opens a destination
   modal; selecting a destination and pressing `Enter` hangs the app (requires
   `Ctrl+C`).
3. **Action plan tree navigation desyncs.** Navigating the tree intermittently
   shows the wrong footer — the category header is treated as if an email were
   selected (and vice versa), so the bottom option menu is wrong.

The user also dislikes the floating-modal experience for both the preview and the
move chooser, and wants the selection to happen **inside the picker/panel**, as
"deeper navigation", in the same position.

## Root Causes (verified)

### #1 Prompt preview won't close
The preview is a floating overlay (`promptPreviewPage`) focused on a
`*tview.TextView`. In tview, `Application.SetInputCapture` (keys.go:527) runs
*before* the focused primitive's own input capture. The focus-type switch at
keys.go:599 only passes events straight through (`return event`) for
`*tview.InputField`, `*tview.List`, `*tview.Form`, `*tview.DropDown`. A
`TextView` falls into `default` with no early return, so:

- `Esc` is consumed by the global Esc handler (keys.go:1183), not the preview.
- `Ctrl+P` is consumed by the global "previous thread" binding (keys.go:1327), a
  no-op that `return nil`s.

The preview's own `SetInputCapture` (correct) is never reached. Opening works
only because the picker's focus is on the `input` (InputField) or `list` (List),
which *do* get the pass-through.

### #2 Action plan "move" hangs
`action_plan_move.go` calls `a.GetErrorHandler().ShowSuccess(...)` synchronously
inside the move list's `SetSelectedFunc` (which runs on the UI goroutine).
`ShowSuccess` internally calls `QueueUpdateDraw` (error_handler.go:321), which
queues work onto the UI goroutine that is currently blocked inside the callback
→ deadlock. The established pattern (links.go) always wraps ErrorHandler calls
made from select callbacks in `go func()`. Introduced in commit `cdcbf41`.

### #3 Action plan tree navigation desyncs
Native Up/Down movement fires `SetChangedFunc` (action_plan.go:205), which
correctly derives `selectedCategory`/`selectedMsgID` from the node reference and
refreshes the footer. But `rebuildActionPlanTree` restores the cursor with
`SetCurrentNode`, which does **not** fire `SetChangedFunc`. When the previously
selected email is no longer visible (its category collapsed after a rebuild), the
restore falls through to a category node, yet `state.selectedMsgID` keeps the
stale email id and the footer is never refreshed. Result: footer shows the
"email" context while the cursor sits on a category — intermittently, exactly when
a rebuild relocates the cursor.

## Design

### Unifying pattern: in-panel body-swap
Neither the preview nor the move chooser is a floating overlay. Each replaces the
body primitive (list/tree) **within the existing side-panel container**, keeping
the border, title, and footer. `Esc` returns to the previous body. This matches
the user's "deeper navigation" preference and structurally avoids the focus-escape
bug class (everything stays inside the picker's focus + its own captures, guarded
by a focus-specific pass-through in keys.go).

### A. Prompt preview in-place
Files: `prompt_preview.go`, `prompts.go`, `bulk_prompts.go`.

- `Ctrl+P` on a highlighted row swaps the `list` item out of the picker container
  and swaps in a scrollable preview `TextView` (the search `input` stays at top).
  Footer text becomes `Enter to apply | Esc/Ctrl+P to go back`.
- **`Enter` applies the prompt** (same path as selecting it in the list:
  close picker + `applyPromptToMessage`). When the previewed row is the
  "✨ Create new with AI…" entry, `Enter` instead opens the prompt configurator
  (same as selecting it in the list). **`Esc`/`Ctrl+P`** swaps the list back in,
  restores focus to the list, restores the footer. Other keys scroll.
- While the preview is shown, `a.currentFocus = "prompt_preview"`. keys.go gets an
  early pass-through: `if a.currentFocus == "prompt_preview" { return event }`, so
  the preview's own input capture receives `Esc`/`Ctrl+P`/`Enter`. On return,
  `currentFocus` is restored to `"prompts"`.
- The floating `promptPreviewPage` overlay (`AddPage`/`closePromptPreview`/
  `promptPreviewPrevFocus`) is removed. The `promptPreviewText()` body builder and
  `promptPreviewCreateNewHint` are retained.
- `bulk_prompts.go` mirrors the same in-place behavior for parity (it has the same
  preview affordance today).

### B. Action plan move in-place
Files: `action_plan_move.go`, `action_plan.go`, `keys.go`.

- `m` on an email swaps the `tree` out of `state.container` and swaps in the
  destinations `List` (standard actions + existing categories, as approved). Title
  becomes `➫ Move "<subject>" to`, footer `Enter to move | Esc to go back`.
- `Enter` (via `SetSelectedFunc`) applies `applyActionPlanMove`, swaps the tree
  back, re-renders, and shows the success toast. **`ShowSuccess` is wrapped in
  `go func()`** (deadlock fix). `Esc` swaps the tree back without moving.
- While the chooser is shown, `a.currentFocus = "action_plan_move"`. keys.go gets a
  branch: for that focus, `return event` (the List handles Enter/Esc/nav via its
  own capture; Esc → swap back). The action-plan focus block must not treat this as
  `"action_plan"` (so `Esc` doesn't close the whole panel).
- The floating `actionPlanMovePage` overlay and its `HasPage` guard are removed.
- `closeActionPlanPanel` no longer needs to `RemovePage(actionPlanMovePage)`.

### C. Tree navigation fix
File: `action_plan.go`.

- New helper `syncSelectionToNode(state, node)`:
  - `state.tree.SetCurrentNode(node)`
  - derive from `node.GetReference()`: `int` → `selectedCategory = ref`,
    `selectedMsgID = ""`; `emailRef` → `selectedCategory = ref.catIndex`,
    `selectedMsgID = ref.msgID`
  - `a.updateActionPlanFooter(state)`
- `rebuildActionPlanTree`'s three terminal cursor-placement sites (email restore,
  category restore, fallback to first child) call `syncSelectionToNode` instead of
  bare `SetCurrentNode`. The node becomes the single source of truth even when
  `SetChangedFunc` does not fire.
- `SetChangedFunc` keeps deriving state from its `node` param (already correct);
  optionally it can call the same helper, but no behavior change is required there.

## Components & Boundaries

- `prompt_preview.go` — owns the in-panel preview swap (show/hide), the preview
  TextView, its input capture, and `promptPreviewText`. No knowledge of action plan.
- `action_plan_move.go` — owns the in-panel destination chooser (show/hide), the
  destinations List, `applyActionPlanMove`, target building. No knowledge of prompts.
- `action_plan.go` — owns tree build/restore and `syncSelectionToNode` (selection
  source-of-truth).
- `keys.go` — owns global routing; gains two focus-specific pass-throughs
  (`"prompt_preview"`, `"action_plan_move"`).

## Error Handling

- All `ErrorHandler` calls made from tview select/done callbacks run in `go func()`
  (never synchronously on the UI goroutine).
- Body-swaps are synchronous UI mutations on the UI thread (no `QueueUpdateDraw`),
  consistent with ESC/cleanup rules; the swap-in/out functions are only ever called
  from key handlers already on the UI thread.

## Testing

- Unit: `syncSelectionToNode` derives correct state for an `int` ref (category and
  read-manually -1) and an `emailRef`; and the "selected email no longer visible →
  lands on category and clears `selectedMsgID`" case (footer reflects category).
- Unit: existing `TestApplyActionPlanMove`, footer, and title tests remain valid.
- Unit (if feasible without a live App): preview text builder unchanged
  (`promptPreviewText`).
- Manual E2E (user): preview opens/applies/returns with Enter and Esc/Ctrl+P; move
  opens in-panel, Enter moves without hanging, Esc returns; tree footer matches the
  highlighted node during and after analysis.

## Out of Scope

- No change to the analyzer prompt, rules, or dispatch logic.
- No new keybindings beyond the existing `Ctrl+P` (preview) and `m` (move).
- No redesign of the inbox or message reader.
