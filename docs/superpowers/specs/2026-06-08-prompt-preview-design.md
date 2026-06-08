# Prompt Preview (On-Demand) — Design

**Date:** 2026-06-08
**Status:** Approved (brainstorm), pending implementation plan
**Origin:** User-reported item #5 from v1.4.1 testing — "I'd like to preview a prompt
before applying it because I don't always remember what it does."

## Summary

Add an on-demand preview to both prompt pickers (single-message and bulk). With a
prompt highlighted, **Ctrl+P** opens a modal popup showing that prompt's name,
description and full template text. **Esc** (or Ctrl+P again) closes it and returns
to the picker with the cursor and filter intact. The preview is presentation-only:
all data is already loaded by the picker, so there are no service or data changes.

## Motivation

The pickers show only a prompt's name (and a category icon). Users who don't recall
what a given prompt does have to either guess or apply it and inspect the result.
A lightweight, on-demand preview lets them confirm a prompt's behavior before
committing — without permanently consuming picker space (the preview only appears
when asked).

## Decisions (from brainstorm)

- **On-demand, not always-visible** — a key opens the preview; nothing shows by default.
- **Modal popup** (not an inline pane) — maximum room for long templates, zero
  permanent space cost.
- **Trigger: Ctrl+P** — works whether focus is on the search input or the list
  (Ctrl chord inserts no text, so it never conflicts with filter typing).
- **Content: name + description + full PromptText**, scrollable for long templates.
- **Both pickers** — single (`prompts.go`) and bulk (`bulk_prompts.go`).

## Behavior

1. In a prompt picker, the user highlights a prompt (by typing to filter and/or
   arrowing through the list) and presses **Ctrl+P**.
2. The highlighted item is resolved with the existing
   `promptPickerSelection(list.GetCurrentItem(), len(visible))` helper:
   - **Real prompt** (index ≥ 1) → open the preview modal for `visible[idx-1]`.
   - **"✨ Create new with AI…"** (index 0) → the modal shows a short hint
     ("Opens the AI prompt configurator to create a new prompt."), not a template.
3. The modal is a centered overlay with:
   - Title: `👁 Preview: <prompt name>`
   - Body: `Description:` block (or "(no description)" when empty), then a
     `Template:` block with the full `PromptText`.
   - A focusable, scrollable `TextView` so long templates can be read in full.
   - Footer hint: `Esc / Ctrl+P to close`.
4. **Esc** or **Ctrl+P** closes the modal and returns focus to the picker exactly
   where it was (same focused widget — input or list — same highlighted item, same
   filter). The picker state is untouched while the modal is open.

## Architecture

Presentation-only. No changes to `PromptService`, the prompt store, or
`PromptTemplate` (which already carries `Name`, `Description`, `PromptText`,
`Category`; the pickers already receive the full slice from `ListPrompts`).

**Picker change (both files):** the local `promptItem` struct currently stores
`id`, `name`, `description`, `category`. Add a `promptText` field so the preview
has the template without an extra lookup. Populate it in the `reload`/load step
from the `PromptTemplate.PromptText` already in hand.

**Shared helpers (new):**
- `promptPreviewText(name, description, promptText string) string` — pure formatter
  that builds the modal body (Description block + Template block, with empty-field
  handling). Unit-testable in isolation.
- `showPromptPreview(name, body string, onClose func())` — builds and shows the
  modal overlay (themed `TextView` inside a bordered, centered container), wires
  Esc/Ctrl+P to close and invoke `onClose` (which restores picker focus). Reused by
  both pickers.

**Trigger wiring (both pickers):** add a Ctrl+P branch to the picker's
`input.SetInputCapture` and `list.SetInputCapture` so it fires from either focus.
The branch resolves the highlighted item via `promptPickerSelection` and calls
`showPromptPreview` (or shows the create-new hint).

**Modal mechanics:** the app already roots its UI in `a.Pages` (`a.SetRoot(a.Pages,
true)`; `a.Pages.GetFrontPage()`). Show the preview as a centered overlay page:
`a.Pages.AddPage("promptPreview", centeredFlex, true, true)` and remove it on close
with `a.Pages.RemovePage("promptPreview")`. A page added on top is modal by nature
(it holds focus while present). Closing restores focus to the picker via `onClose`.
The preview does NOT use the `ActivePicker` enum (that tracks side-panel pickers in
`contentSplit`, not this transient page) and must not touch existing picker state.

**Theming:** all modal components use `app.GetComponentColors("prompts")` for
background/border/title/text, consistent with the pickers themselves.

**Threading:** display-only and synchronous on the UI thread (key handler →
build/show modal). No goroutines, no `QueueUpdateDraw` in the open/close path
(consistent with ESC-handler rules — closing must be synchronous to avoid deadlock).

## Error Handling

- Empty `PromptText` or `Description` render as explicit placeholders
  ("(no description)" / "(empty template)") rather than blank space.
- If the highlighted index resolves to nothing (stale/empty list), Ctrl+P is a no-op.

## Testing

- **Unit:** `promptPreviewText` — description+template layout; empty description;
  empty template; create-new hint text. Pure function, no tview.
- **Unit (reuse):** `promptPickerSelection` already covers the index→item mapping.
- **E2E (tmux, real account):** open the bulk picker, highlight a prompt, press
  Ctrl+P → modal shows the template; scroll a long one; Esc → back to the picker
  with cursor preserved; repeat from the single-message picker. Use `/usr/bin/tmux`
  directly.

## Command Parity

Not applicable: this is a picker-local interaction (like Esc/Enter/arrows within a
modal), not a global keyboard shortcut, so it needs no `:command` equivalent.

## Files Touched (anticipated)

- `internal/tui/prompts.go` — `promptItem.promptText`, Ctrl+P wiring, hint for row 0.
- `internal/tui/bulk_prompts.go` — same wiring for the bulk picker.
- `internal/tui/prompt_preview.go` — NEW: `promptPreviewText` + `showPromptPreview`.
- `internal/tui/prompt_preview_test.go` — NEW: unit tests for `promptPreviewText`.
- `docs/KEYBOARD_SHORTCUTS.md` — document Ctrl+P preview within the prompt pickers.

## Out of Scope (YAGNI for v1)

- Editing a prompt from the preview (the configurator already does this).
- Showing favorite/usage metadata in the preview.
- A config option to rebind the preview key or change modal size.
- An always-visible preview pane (explicitly rejected in favor of on-demand).
