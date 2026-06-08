# Inbox Action Plan Rework — Design

**Date:** 2026-06-08
**Status:** Approved (brainstorm), pending implementation plan
**Origin:** User testing of the Inbox Action Plan (shipped v1.3.0; `P` / `:action-plan`;
service `InboxAnalyzerService`; code `internal/tui/action_plan.go`). Four feedback
threads (A focus bug, B whole-inbox scope, C cryptic footer, D no email visibility),
recorded in memory `action-plan-feedback.md`.

## Summary

Rework the Inbox Action Plan from a **whole-inbox, scroll-only, blind** panel into a
**selection-scoped, expandable, trustable** one, and add a lightweight, user-controlled
**learning** layer.

- **Scope (B):** analyze the user's current bulk selection when present; fall back to the
  unread inbox when nothing is selected.
- **Widget (D + A):** replace the `tview.TextView` body with a `tview.TreeView` — categories
  expand to reveal their actual emails. Real selection/focus semantics fix the post-analysis
  focus/navigation bug (A) at the root rather than patching it.
- **Per-email control (D):** each email in a category can be toggled out (`space`); actions
  apply only to the still-checked emails.
- **Action trigger (C):** the configured action keys (`a`/`t`/`l`/`r`) act on the highlighted
  category's checked emails.
- **Footer (C):** a single context-aware line that never truncates; the old `:` palette and
  `p` configurator escape hatches are removed.
- **Learning (new):** free-text natural-language preference rules, persisted per account and
  injected into the analyzer prompt for the LLM to honor. Captured via `Ctrl+R` (context-seeded,
  editable) inside the panel and managed via `:action-plan rules`.

## Motivation

The shipped feature analyzes the entire unread inbox and proposes bulk actions over everything,
shows categories as one-line summaries with no way to see which emails they contain, has a
cryptic footer that truncated off the narrow side panel, and loses keyboard focus/highlight
after the streamed analysis completes. Together these make the feature feel untrustworthy:
the user is asked to trash/label N emails blind, can't correct a miscategorization, and can't
navigate the result. This rework addresses all four threads and folds the focus bug into the
widget change (a standalone focus patch would be thrown away by the redesign).

## Decisions (from brainstorm)

- **Scope: selection-first, inbox fallback** — one entry point (`P` / `:action-plan`), smart
  default. Keeps the zero-effort "triage my whole inbox" flow alive while making selection the
  primary, bulk-like mode.
- **Widget: expandable tree** (`tview.TreeView`) — categories are root nodes, emails are
  children; everything lives in one panel. Native selection/focus.
- **Granularity: per-email exclusion** — emails start checked; `space` toggles. Action applies
  to checked-only. Maximum trust/control.
- **Action trigger: suggested-action keys** (`a`/`t`/`l`/`r`) — reuses main-list muscle memory;
  no abstract generic "apply" verb.
- **Footer: context-aware single line** — shows only the gestures relevant to the highlighted
  node; never truncates.
- **Escape hatches dropped** — per-email exclusion + direct action keys make `:` and `p`
  redundant; removing them simplifies the footer and mental model.
- **Learning: lightweight, prompt-context, free-text** — rules are natural-language strings the
  LLM interprets, NOT deterministic sender-matching. Rules may be sender/domain-specific OR
  general. Store is just strings; interpretation lives in the LLM already in the loop.
- **Capture: explicit `Ctrl+R`** (context-seeded, editable) + `:action-plan rules` for general
  rules. No auto-inference (no surprises). `Ctrl+R` avoids the global `R` = reload-all collision
  and mirrors the `Ctrl+P` prompt-preview chord shipped in v1.5.0.

## Behavior

### Opening (scope)

1. `P` / `:action-plan` is pressed.
2. If `a.selected` is non-empty (bulk mode), build the analyzer input from **those message IDs**
   (no UNREAD filter — an explicit selection counts regardless of read state).
3. Otherwise, fall back to the current behavior: unread messages already in memory
   (`buildAnalyzerMessages` filtering `UNREAD`).
4. The header states the active mode: `Action Plan · 5 selected` vs `Action Plan · 23 unread (inbox)`.

### The tree

- **Root nodes = categories**, e.g. `▸ Archive · 7/8 · Newsletters · HIGH`
  - `7/8` = checked/total. `HIGH/MED/LOW` = priority. Label categories also show `→ label: X`.
- **Child nodes = emails**, e.g. `[x] Weekly digest — Medium` (`[ ]` when excluded).
- `↑↓` navigate (native), `Enter`/`→` expand a collapsed category, `←` collapse.
- Categories start collapsed; expanding lazily realizes child nodes from `cat.MessageIDs`
  (subject/from resolved from the same in-memory metadata the analyzer used).
- During streaming, batch-progress re-renders rebuild the tree while preserving the selected
  node and each category's expand/exclusion state. The final post-analysis render does the same —
  this is what fixes thread A: the `TreeView` owns focus, so a re-render can't silently drop it.

### Per-email exclusion

- Every email node starts **checked**. `space` on an email toggles its checked state.
- A category's checked count is reflected live in its root label (`7/8`).
- Excluding all emails in a category disables its action (the action keys no-op with a hint).
- Exclusion state is held in `actionPlanState` and is **ephemeral** (lost on close), by design.

### Acting

- With a category highlighted (or one of its emails), pressing the action's configured key
  (`a` archive, `t` trash, `l` label, `r` mark-read — per `a.Keys.*`) runs that bulk action on
  the category's **checked** message IDs only.
- On success the category is removed from the tree (existing `removeActionPlanCategory` pattern),
  and selection moves to the next category.
- Quick-actions remain blocked while `state.analyzing` is true (unchanged).

### Footer (context-aware)

- On a **category** node: `[a] archive 7  [Enter] expand  [^R] remember  [Esc]`
  (verb + count reflect the highlighted category's suggested action and checked count).
- On an **email** node: `[space] skip  [^R] remember sender  [←] collapse  [Esc]`.
- The footer is rebuilt on every selection change; it is always short enough not to truncate.

### Learning

- **Capture from context (`Ctrl+R`):** opens a small input modal pre-seeded with a suggestion
  derived from the highlighted node:
  - on an email: `Never trash emails from tldr.tech` (verb from the category's action, domain
    parsed from the email's `From`),
  - on a category: a category-level phrasing.
  The user can edit the text freely (including rewriting it as a fully general rule) before
  confirming. Confirm saves; `Esc` cancels.
- **Manage general rules (`:action-plan rules`):** a simple manager — list existing rules, add a
  free-text rule, delete a rule. For preferences not tied to a specific email
  (e.g. `Treat anything from my team as high priority`).
- **Injection:** on the next analysis, all of the account's rule texts are passed via the new
  `InboxAnalyzerOptions.UserRules` field. The service prepends a block to the prompt before
  batching:

  ```
  ## User preferences (respect these rules)
  - Never trash emails from tldr.tech
  - Treat anything from my team as high priority
  - Archive newsletters automatically
  ```

  The LLM honors these when categorizing. No deterministic matching is done in Go.

## Architecture

Service-first, per CLAUDE.md. All persistence and rule logic lives in `internal/db` +
`internal/services`; the TUI only captures gestures and renders.

### New: rules persistence (`internal/db/analyzer_rules_store.go`)

Mirrors `query_store.go`. Single table, account-scoped:

```sql
CREATE TABLE IF NOT EXISTS analyzer_rules (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    account_email TEXT    NOT NULL,
    rule_text     TEXT    NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Store methods: `SaveRule(accountEmail, ruleText)`, `ListRules(accountEmail) []AnalyzerRule`,
`DeleteRule(accountEmail, id)`.

### New: rules service (`internal/services/analyzer_rules_service.go` + interface)

```go
type AnalyzerRule struct {
    ID        int
    RuleText  string
    CreatedAt time.Time
}

type AnalyzerRulesService interface {
    SaveRule(ctx context.Context, ruleText string) error
    ListRules(ctx context.Context) ([]AnalyzerRule, error)
    DeleteRule(ctx context.Context, id int) error
    // SuggestRuleFromContext builds an editable default rule string from a message's From
    // and an action verb (e.g. "Never trash emails from tldr.tech"). Pure, testable.
    SuggestRuleFromContext(from, action string) string
}
```

Registered in `App` and returned from `GetServices()` (extending the current 12-value tuple —
all call sites updated). Domain extraction from `From` is service-side, pure, unit-tested.

### Changed: analyzer options + prompt (`internal/services/inbox_analyzer_service.go`)

- Add `UserRules []string` to `InboxAnalyzerOptions`.
- In `Analyze`, when `UserRules` is non-empty, prepend the `## User preferences` block to
  `promptText` before the existing `buildBatchPrompt` flow. Empty slice → unchanged behavior.

### Changed: TUI (`internal/tui/action_plan.go`)

- `openActionPlanWithText` chooses selection vs unread-fallback input; loads the account's rules
  via `AnalyzerRulesService` and passes them in `InboxAnalyzerOptions.UserRules`.
- Replace `state.body *tview.TextView` with `state.tree *tview.TreeView`; replace
  `renderActionPlanText` with tree (re)building that preserves selection + expand + exclusion.
- `actionPlanState` gains per-category exclusion state (`map[catIndex]map[msgID]bool` or a set of
  excluded IDs) and email subject/from lookup from the in-memory metadata.
- `actionPlanInputCapture` updated: `space` toggle, `Ctrl+R` remember, action keys act on
  checked-only, remove `:`/`p` handling. ESC stays synchronous (no `QueueUpdateDraw`).
- Footer becomes a function of the currently-selected node.
- `:action-plan rules` subcommand added to `commands.go` (+ command suggestions), reusing the
  rules service.

## Data flow

```
P / :action-plan
  └─ selection? → IDs from a.selected   ─┐
     else       → unread metas in memory ─┤→ []AnalyzerMessage
                                          │
  AnalyzerRulesService.ListRules() ───────┤→ UserRules []string
                                          ▼
  InboxAnalyzerService.Analyze(msgs, {…, UserRules}, onProgress)
     └─ prepends "## User preferences" block to prompt → batches → categories
                                          ▼
  TreeView: categories → (expand) emails, space-toggle exclusion
     └─ action key → BulkArchive/Trash/MarkRead / labelService on CHECKED ids
  Ctrl+R → SuggestRuleFromContext → editable modal → SaveRule
```

## Error handling

- Selection mode with zero resolvable messages → `ShowInfo` hint, do not open.
- `AnalyzerRulesService` unavailable (no account/DB) → analysis still runs with empty `UserRules`;
  `Ctrl+R` / `:action-plan rules` show a `ShowWarning` ("rules unavailable — check account/DB")
  instead of failing. (Mirrors the known legacy account DB-init gap.)
- Rule save/delete failures → `ShowError`, no panel disruption.
- All user feedback via `GetErrorHandler()`; no direct output. Streaming/ESC threading rules
  from CLAUDE.md preserved (no `QueueUpdateDraw` in ESC/cleanup; batch renders stay on the UI
  thread via the existing `QueueUpdateDraw` in the analysis goroutine).

## Testing

- **Store:** CRUD + account scoping (`analyzer_rules_store_test.go`), following `query_store_test.go`.
- **Service:** `SuggestRuleFromContext` domain/verb extraction; `ListRules` ordering; injection
  block formatting (assert the prepended prompt contains each rule and the header).
- **Analyzer:** `UserRules` empty → prompt unchanged; non-empty → block prepended once.
- **TUI (where testable):** scope selection (selected vs fallback), exclusion math (checked count,
  all-excluded disables action), footer text per node type, `Ctrl+R` seeding string.
- `make pre-commit-check` before claiming complete; E2E via `tmux` for the focus/nav fix (A) and
  the live tree interaction (catches what unit tests miss).

## Command parity

- `:action-plan` honors the same selection-first scope.
- `:action-plan rules` (alias e.g. `:ap rules`) manages rules outside the panel.
- No new global shortcut: `Ctrl+R` and `space` are in-panel gestures owned by the panel's
  input capture.

## Out of scope (explicit)

- Auto-inference of rules from repeated corrections (decided against; explicit-only).
- Persisting per-email exclusions across sessions (ephemeral by design).
- Deterministic rule matching in Go (LLM interprets rules).
- The separate auto-refresh-inbox feature (already specced/planned, queued after this).

## Risks / notes

- **`R` collision:** global `R` = reload-all. Mitigated by `Ctrl+R` in-panel + the panel input
  capture consuming the chord. Verify in E2E that reload never fires from within the panel.
- **Non-determinism:** the LLM may imperfectly honor a poorly-worded rule. Acceptable for soft
  triage preferences; the user sees the result in the tree and can still exclude per-email.
- **`GetServices()` arity change:** adding `AnalyzerRulesService` touches all call sites — a
  mechanical but repo-wide edit; the plan must enumerate them.
