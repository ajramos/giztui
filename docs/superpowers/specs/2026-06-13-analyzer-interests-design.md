# Analyzer user-interests layer (idea C)

**Date:** 2026-06-13
**Status:** Approved (design)

## Problem

The user wants the inbox analyzer to factor in their **interests** (e.g. "I'm very interested in
AI") so emails matching those interests aren't buried in bulk archive/trash and instead get
surfaced. Today the analyzer only consumes saved free-text rules as behavioral preferences; there
is no notion of "interest/relevance", and the rules UI ("Analyzer rules" / "New rule:") doesn't
hint that you can express interests there.

## Goal

Let the user state interests through the existing rules store and have the analyzer treat them as
relevance signals — surfacing matching emails (priority "high" + a note in the category
description) rather than burying them. Make this discoverable by relabeling the rules UI. Also fix
a focus bug where `:plan rules` opens the picker but leaves focus on the message list.

## Decisions (confirmed with user)

- **Reuse the existing rules store** (no new storage/UI for interests) — just reframe the prompt.
- **Effect:** interest-matching emails get priority "high" and are noted, staying in their action
  category (no new "of interest" category, no schema change).
- **Discoverability:** relabel the rules manager / inputs to mention interests.
- **Bonus fix in scope:** `:plan rules` focus bug (picker opens but focus stays on the list).

## Architecture

### A) Prompt reframe — `internal/services/inbox_analyzer_service.go` (`prependUserRules`)

Change the prepended block heading + add a relevance instruction. From:

```
## User preferences (respect these rules)
- <rule>
```

to:

```
## User preferences and interests
Treat the following as BOTH action rules to respect AND interest/relevance signals. When an email
matches a stated interest, do NOT bury it in a bulk archive/trash group: keep it visible, set that
category's priority to "high", and note the matched interest in the category description.
- <rule or interest>
```

Only emitted when rules exist (unchanged); empty rules → no block, prompt unchanged. Reuses the
existing `priority` / `description` schema fields (both already rendered:
`actionVerbLabel · N/M · Name · PRIORITY`). No schema or other code change. The `v` prompt viewer
(`BuildPromptPreview`) uses `prependUserRules`, so it reflects the new text automatically.

### B) UI relabel — `internal/tui/action_plan_rules.go`

- Rules manager container title: `" 🧠 Analyzer rules "` → `" 🧠 Analyzer rules & interests "`.
- Add-rule embedded input label: `" New rule: "` → `" Rule or interest: "`.
- `Ctrl+R` remember-rule panel title: `" 🧠 Remember preference "` →
  `" 🧠 Remember rule or interest "` (and its footer text stays "Enter save · Esc cancel").

### C) Focus fix — `:plan rules` should take focus

Root cause: `:plan rules` runs `executeActionPlanCommand` → `openAnalyzerRulesManager()`
synchronously during `executeCommand`, then `hideCommandBar()` → `restoreFocusAfterModal()` runs
afterward and re-focuses the message list (default branch), clobbering the picker's focus.

Fix (synchronous, mirrors the existing `cmdFocusOverride` precedent used by content search):
- In `openAnalyzerRulesManager`, after `a.SetFocus(list)` / `a.currentFocus = "analyzer_rules"`,
  set `a.cmdFocusOverride = "keep"`.
- In `restoreFocusAfterModal` (`internal/tui/...`), add `case "keep": return` so it leaves the
  focus the picker already grabbed (and the override is consumed/cleared as usual).

This is synchronous (no `QueueUpdateDraw` defer → no leaked goroutine, no test breakage) because
`openAnalyzerRulesManager` already runs on the UI thread during command execution.

### D) Docs / `:help` (Definition of Done)

- `:help` command list: note that `:plan rules` manages "rules or interests".
- `docs/CONFIGURATION.md` (analyzer section): mention you can write action rules **or** interests
  (e.g. "I'm interested in AI") in `:plan rules` / `Ctrl+R`, and that the analyzer raises the
  priority of emails matching your interests.

## Testing

- `TestPrependUserRules_Interests`: with rules → output contains "User preferences and interests",
  the relevance instruction substring, and the rule text; with empty rules → no block (unchanged).
- `TestAnalyzerRulesPickerSetsFocusOverride`: `openAnalyzerRulesManager()` sets
  `currentFocus == "analyzer_rules"` AND `cmdFocusOverride == "keep"` (synchronous — extends the
  existing `TestAnalyzerRulesPickerSwap` setup).
- `TestRestoreFocusAfterModal_Keep`: with `cmdFocusOverride == "keep"`, `restoreFocusAfterModal()`
  does NOT change `currentFocus` to "list" and clears the override.

## Out of scope

- Separate interests storage/UI (reuse rules — decided).
- A dedicated "of interest" category or per-email priority (schema is category-level — decided).
- Sorting categories by priority (possible later; not needed — priority is already displayed).
