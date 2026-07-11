# Deterministic Rules (token-free Action Plan) — Design Spec

**Date:** 2026-07-04
**Status:** Approved by user (hybrid engine, Gmail-search rule format, AI-plan prefilter, opt-in Gmail filter sync, prompt actions)
**Depends on:** #54 (`feat/action-plan-confirm-all`) — the shared Action Plan panel gains the whole-plan confirm there; this branch should be rebased/merged after #54 lands on main.

## Summary

A deterministic, zero-token counterpart to the AI Inbox Action Plan. The user defines **rules** — a Gmail search query plus an action — that are:

1. Managed in a new `:rules` panel (list / add / edit / delete, per account).
2. Applied on demand to the current inbox via `:rules plan`, reusing the existing Action Plan panel (tree, exclusions, per-category apply, whole-plan `c` confirm) with one category per rule, built instantly with no LLM call.
3. Run automatically as a **prefilter** when the AI plan opens (`:plan`): messages matched by a rule become pre-resolved ⚡ categories and are NOT sent to the LLM — that is the token saving.
4. Optionally mirrored as **real Gmail filters** (per-rule opt-in) so future incoming mail is processed server-side 24/7, even with GizTUI closed.

## Rule model

New table + store, mirroring `internal/db/analyzer_rules_store.go` (`analyzer_rules`):

```
deterministic_rules
  id             INTEGER PK
  account_email  TEXT NOT NULL
  query          TEXT NOT NULL      -- Gmail search syntax, e.g. "from:amazon.com subject:pedido"
  action         TEXT NOT NULL      -- archive | mark_read | trash | label | prompt
  label          TEXT               -- when action == label
  prompt_id      INTEGER            -- when action == prompt (references prompt_templates.id)
  gmail_filter_id TEXT              -- non-empty when mirrored as a Gmail filter
  created_at     TIMESTAMP
```

- New file `internal/db/deterministic_rules_store.go` (`SaveRule` / `ListRules` / `UpdateRule` / `DeleteRule`, account-scoped like `AnalyzerRulesStore`). Table created via `CREATE TABLE IF NOT EXISTS` in the store schema (`internal/db/store.go`), additive — no destructive migration.
- Business logic in a new service `RulesService` (`internal/services/`, interface in `interfaces.go`): rule CRUD, query validation, matching, Gmail filter sync. UI stays presentation-only.
- Rules apply **in creation order; first match wins** (a message matched by rule 1 is not offered to rule 2). Order is visible in the manager.

### Query validation at save time

On save, `RulesService` runs the query through the existing search path (`SearchMessages(ctx, query, opts)` with `MaxResults: 1`, `internal/services/interfaces.go:18`). A Gmail-side syntax error surfaces immediately via `ShowError` — not later during a sweep.

## Rules manager: `:rules`

A side panel following the analyzer-rules-manager pattern (`openAnalyzerRulesManager`, `internal/tui/action_plan_rules.go`):

- Lists rules for the active account: `⚡ from:newsletter@medium.com → archive`, `📎 from:jira → prompt: resume ticket`, with a `☁` marker on rules mirrored in Gmail.
- Add / edit: input for the query, action selector (archive / label / trash / mark read / prompt), label name or prompt picker as applicable, and a "also in Gmail" toggle (disabled for `prompt` rules).
- Delete removes the rule and, if mirrored, deletes the Gmail filter too.
- New `ActivePicker` constant (e.g. `PickerRules`) — no shared boolean flags.
- Theming via `app.GetComponentColors(...)` (component: `labels`-style picker or a new `rules` entry following THEMING.md guidance).

## Deterministic plan: `:rules plan`

Builds a `services.ActionPlan` (same type the analyzer returns) with one category per rule and opens the **existing** Action Plan panel (`openActionPlanWithText` flow minus the LLM, `internal/tui/action_plan.go:188`):

- For each rule, run its query scoped to the inbox (`in:inbox <query>`) via `SearchMessages`; the returned IDs become the category's `MessageIDs` (deduped across rules, first match wins). Empty categories are dropped.
- Category name = rule description (query, truncated), `Action`/`Label` from the rule; title/footer make clear it is the deterministic plan (⚡, "no AI").
- Everything the panel already does works unchanged: Space exclusions, per-category action keys, move, Enter to open, and — after #54 — the two-press `c` whole-plan confirm and `:plan apply`.
- `prompt` categories: excluded from the whole-plan `c` confirm (same as `summarize` today — a prompt yields output to read, not a blind mailbox mutation). From the category the user can launch the saved prompt over its checked emails, reusing the bulk-prompt path (`dispatchActionPlanSummarize` pattern, `internal/tui/action_plan_summarize.go:45`, but with the rule's `prompt_id` template instead of the built-in summarize prompt); results render in the AI panel like bulk prompt results.

## AI-plan prefilter

When the AI plan opens (`openActionPlanWithText`, before launching the analyzer):

1. Build the candidate message set as today (`buildAnalyzerMessages` / `buildAnalyzerMessagesForSelection`, `internal/tui/action_plan.go:19,41`).
2. For each rule (in order), run its query scoped to the same source (`in:inbox` and `is:unread` when the plan source is unread) and **intersect** the returned IDs with the candidate set — server-side semantics, exact match with what Gmail itself would do.
3. Matched messages become pre-resolved ⚡ categories prepended to the plan; only the remainder goes to the LLM.
4. Status feedback: `⚡ 14 resolved by rules · 23 sent to AI` (via ErrorHandler).
5. Config toggle `inbox_analyzer.deterministic_prefilter` (default `true`) to let the AI see everything. Added to `DefaultConfig()` + surfaced by config self-migration.
6. If every candidate is matched by rules, skip the LLM entirely and show the plan (zero tokens).

## Gmail filter sync (opt-in per rule)

- New Gmail client methods in `internal/gmail/` wrapping `gmail.Users.Settings.Filters` Create/Delete (the API also supports List, used to detect orphans): criteria `{Query: rule.query}`, action mapped as:
  - archive → `RemoveLabelIds: ["INBOX"]`
  - mark_read → `RemoveLabelIds: ["UNREAD"]`
  - trash → `AddLabelIds: ["TRASH"]`
  - label → `AddLabelIds: [labelID]` (resolved/created via the existing label path)
  - prompt → **not syncable**; toggle disabled in the manager.
- The created filter's ID is stored in `gmail_filter_id`; deleting the rule deletes the filter; editing a mirrored rule recreates the filter (delete + create).
- **New OAuth scope required:** `https://www.googleapis.com/auth/gmail.settings.basic`, added in BOTH scope lists (`cmd/giztui/main.go:341` and `internal/services/account_service.go:~470`). Existing tokens lack the scope → the first sync attempt fails with a clear message telling the user to re-authorize (delete token / re-run auth); scope addition itself forces re-consent on next login. Document in README/config docs.
- Gmail filters only process **future incoming mail**; the local sweep (`:rules plan`) is the tool for what is already in the inbox. The manager/help text states this.
- Gmail rejects or silently ignores some operators as filter criteria (e.g. `is:unread`, `in:`, relative dates) even though they work in search. Sync is attempted as-is; on API rejection the rule stays local and the user gets a `ShowWarning` explaining Gmail did not accept the query as a filter.

## Command parity & keys

- `:rules` (alias `:ru`) — open the manager.
- `:rules plan` — open the deterministic plan.
- `:rules sync <n>` / `:rules unsync <n>` — optional explicit toggle by list position (manager toggle is the primary path).
- Arg completion (`command_completion.go` registry entry + `completeRulesArg`: first token → `plan|sync|unsync`).
- No default keyboard shortcut initially (command-first, like TTS `speak`); an optional `keys.rules` binding (default empty) may be added later — out of scope now.

## Error handling & threading

- All user feedback via `GetErrorHandler()`; `go`-wrapped when called from the UI goroutine, direct from workers (same discipline as the Action Plan).
- Rule sweeps and Gmail sync run in worker goroutines; panel/tree mutations only inside `QueueUpdateDraw` guarded by the state-identity check (`a.actionPlanState == state`), as in #54.
- ESC paths synchronous — no `QueueUpdateDraw`.
- Sequential per-rule searches (no parallel fan-out) to keep Gmail quota and progress messages coherent.

## Config & docs (Definition of Done)

1. `inbox_analyzer.deterministic_prefilter` (bool, default true) in `DefaultConfig()` + self-migration surfaces it in existing config files.
2. In-app `?` help: `:rules`, `:rules plan`, prefilter behavior line under the Action Plan section.
3. `docs/KEYBOARD_SHORTCUTS.md`: commands table rows; note on the ⚡ categories in the Action Plan section.
4. README / features docs: short section on deterministic rules incl. the re-auth requirement for Gmail sync.

## Testing

1. **Unit (services):** rule matching/partition logic (first-match-wins, dedupe, intersection with candidate set); Gmail filter action mapping; query validation errors. Mocked repository/client per TESTING.md.
2. **Unit (db):** deterministic_rules store CRUD, account scoping (mirror `analyzer_rules_store_test.go`).
3. **TUI:** manager panel open/close/ActivePicker; deterministic plan builds categories from stubbed search results; prompt categories excluded from whole-plan confirm.
4. `make pre-commit-check` before claiming done; full `make test` as a separate step before merge (leak detector).
5. Live smoke test (test Gmail account): create rule, `:rules plan`, apply, verify in Gmail; sync one rule and verify the filter exists in Gmail settings.

## Out of scope (own specs later)

- **Rule promotion (feature 2):** analyzing AI preference rules / past plans and proposing equivalent deterministic rules.
- **Second-round UX (feature 3):** better experience for `prompt` and "read manually" leftovers — the user explicitly wants to revisit this UX afterwards.
- Whole-plan undo; parallel rule execution; rule import from existing Gmail filters (nice-to-have, phase 2).
