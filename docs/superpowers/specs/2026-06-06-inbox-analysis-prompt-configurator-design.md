# Inbox Action Plan & Prompt Configurator — Design Spec

| Field | Value |
|---|---|
| Date | 2026-06-06 |
| Author | ajramos (with Claude assistance via brainstorming skill) |
| Status | Draft — awaiting review |
| Target version | v1.3.0 |
| Scope | Two related TUI features sharing common infrastructure |

## 1. Summary

This spec defines two new features for GizTUI aimed at reducing the time a user spends triaging unread email:

- **Inbox Action Plan (Feature 1)**: an AI-assisted analyzer that scans the unread messages in the current view, groups them into actionable categories, and presents a panel with quick-action keys to dispatch each group with a single keystroke.
- **Prompt Configurator (Feature 2)**: a UI for generating, refining, and saving custom AI prompts from natural-language intent — eliminating the friction of writing prompt templates from scratch.

The two features are designed as separate UIs but share common service infrastructure (prompt storage, bulk-apply engine, AI service, error handling). Feature 1 can optionally be driven by a custom prompt produced by Feature 2.

## 2. Problem statement

Today the user can:
- Open one email and apply a saved prompt to it (`p` key, existing flow).
- Enter bulk mode and apply a bulk prompt to N selected emails (`v` + `p`, existing flow we discovered already wired up in the codebase).

The user reports two unresolved pain points:

1. **No way to get a system-driven action plan over the unread inbox**. To know what to do next, the user has to read each subject one by one.
2. **Writing good prompts by hand is tedious**. The pre-seeded bulk prompts (e.g., "Business Intelligence Report") are narrow and rarely match the user's actual need. Composing a new prompt from scratch is friction that prevents the user from leveraging bulk operations in practice.

## 3. Goals

1. Let the user trigger an automated triage of the unread inbox in seconds and act on the results without leaving the panel.
2. Let the user describe an analytical intent in natural language and obtain a working prompt template they can iterate on, apply, and save for reuse.
3. Reuse the existing GizTUI infrastructure (bulk prompt service, prompt storage, error handling, theming, ESC patterns) to keep net-new code small and maintainable.
4. Degrade gracefully when the LLM produces unexpected output — the user never gets a dead-end screen.
5. Conform to all mandatory patterns in `CLAUDE.md` (service-first, ErrorHandler-only, ActivePicker enum, configurable keys, command parity, theming).

## 4. Non-goals (out of scope for this spec)

- Multi-account aggregation (the analyzer operates on the active account only).
- Persistent action plans across sessions (the plan is in-memory; per-batch LLM results are cached as today).
- Prompt sharing across accounts or export/import beyond what `PromptService` already supports.
- Real-time inbox monitoring or background analysis on a schedule.
- A general-purpose chat UI — the configurator is single-purpose (intent → prompt).

## 5. Decisions taken during brainstorming

These were the seven decisions confirmed by the user before this spec was written.

| # | Topic | Decision |
|---|---|---|
| 1 | Volume / batching | Batches of 50–100 messages, configurable. Default 50. |
| 2 | Action plan output format | Categorized report with quick-actions per category. Tabular, not a wall of text. |
| 3 | Escape hatch on categories | `:` opens command palette pre-scoped to that category's messages. `p` opens the configurator scoped to the same. |
| 4 | Categories | Default fixed set of 5 buckets, with override via custom prompt from the user's library. |
| 5 | Configurator iteration model | Hybrid: a directly-editable prompt box plus a "refine" input line for LLM-assisted regeneration. |
| 6 | Save semantics | Explicit save (default `Ctrl+S`, configurable) to the existing shared prompt library. Ephemeral if not saved. |
| 7 | Configurator invocation | The existing prompts picker (`p` key) is enriched with a "✨ Create new with AI" item at the top of the list. |

## 6. Architecture overview

The two features sit on top of the existing service layer with minimal additions.

```
┌────────────────────────────────────────────────────────────┐
│ UI Layer (internal/tui/)                                   │
│                                                            │
│  prompts.go              (modified) — picker enrichment    │
│  prompt_configurator.go  (NEW)      — Feature 2 panel      │
│  action_plan.go          (NEW)      — Feature 1 panel      │
│  keys.go                 (modified) — new key handlers     │
│  commands.go             (modified) — :action-plan family  │
│  app.go                  (modified) — wiring + enums       │
└────────────────────────────────────────────────────────────┘
                          │ via GetServices()
                          ▼
┌────────────────────────────────────────────────────────────┐
│ Service Layer (internal/services/)                         │
│                                                            │
│  PromptService           (extended) — generation methods   │
│  prompt_generator_service.go (NEW) — NL → prompt via LLM   │
│  inbox_analyzer_service.go   (NEW) — Feature 1 orchestrator│
│  BulkPromptServiceImpl   (reused)   — bulk LLM apply       │
│  AIService               (reused)   — LLM calls            │
│  EmailService            (reused)   — bulk actions         │
└────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌────────────────────────────────────────────────────────────┐
│ Storage / External                                         │
│  Gmail API   │   LLM provider   │   SQLite (prompts + cache)│
└────────────────────────────────────────────────────────────┘
```

### Scope of new code

| Type | Count | Details |
|---|---|---|
| New services | 2 | `prompt_generator_service`, `inbox_analyzer_service` |
| New UI files | 2 | `prompt_configurator.go`, `action_plan.go` |
| Modified files | ~5 | `prompts.go`, `keys.go`, `commands.go`, `app.go`, `interfaces.go` |
| New ActivePicker enums | 2 | `PickerPromptConfigurator`, `PickerActionPlan` |
| DB schema changes | 0 (optional v6 migration for `generated_by` metadata) | |

### Reused as-is (no changes)

- `BulkPromptServiceImpl` — streaming, caching, cancellation, content cleaning.
- `PromptService` storage and CRUD operations.
- `ErrorHandler` for all status/error messaging.
- Theming via `GetComponentColors`.
- ESC handler pattern with `streamingCancel`.
- The `selected` map for bulk selections.

## 7. Functional components

### 7.1 — Enriched prompts picker (Piece A, modified existing)

The picker that opens with `p` today. Two changes:

- **First list item** becomes "✨ Create new with AI…".
- **Category filter is relaxed**: instead of hard-filtering to `Category == "bulk_analysis"`, the picker shows all prompts compatible with the current context. Single message context → prompts with `{{body}}` placeholder. Bulk context → prompts with `{{messages}}` placeholder. Both placeholders supported uniformly by the service layer (already done).

When the user selects "Create new" → close picker, open the configurator preserving the current context (single message ID, bulk selection, or action-plan category).

When the user selects an existing prompt → existing flow, unchanged.

### 7.2 — Prompt Configurator (Piece B, new panel)

A side panel similar in structure to today's prompts picker. Layout:

```
┌─ ✨ Prompt Configurator (12 msgs scoped) ─────────────┐
│ Intent: identify urgent reply candidates_             │
│ ───────────────────────────────────────               │
│ Editable prompt:                                      │
│ ┌───────────────────────────────────────────────────┐ │
│ │ You are an email triage assistant. Analyze       │ │
│ │ the following emails {{messages}} and identify   │ │
│ │ which require urgent reply...                    │ │
│ └───────────────────────────────────────────────────┘ │
│                                                       │
│ Refine: > make output JSON_                           │
│                                                       │
│ [Enter] apply  [Ctrl+R] refine  [Ctrl+S] save         │
│ [Ctrl+T] test on 1 msg (stretch)  [Esc] cancel        │
└───────────────────────────────────────────────────────┘
```

User flow:
1. Type intent in the top box → Enter triggers LLM generation.
2. Generated prompt appears in the middle editable box (streamed).
3. User edits manually (small adjustments) and/or types refinement instructions in the bottom line.
4. Apply runs the prompt against the scoped context using the existing single or bulk apply path.
5. Optionally, Ctrl+S opens a save dialog with name/description pre-filled by the LLM.

All shortcut keys read from `a.Keys.*` and have sensible defaults; configurable in `~/.config/giztui/config.json`.

### 7.3 — Action Plan panel (Piece C, new panel)

Opens when `:action-plan` is executed or the configurable shortcut is pressed (default `A`).

```
┌─ 📋 Action Plan (52 msgs • batch 2/2 • cached) ──────┐
│                                                      │
│▸[a] Archive 18 newsletters              ◀ HIGH       │
│      Marketing, no engagement >30d                   │
│                                                      │
│ [r] Mark 8 informational as read        ◀ MED        │
│      Receipts, confirmations                         │
│                                                      │
│ [l] Label "needs-reply" on 12 messages  ◀ HIGH       │
│      Questions, deadlines                            │
│                                                      │
│ ─── Read manually (14) ──────────                    │
│   1. "Q3 budget review" — CFO — urgent               │
│   2. "Contract renewal" — Acme — Friday              │
│                                                      │
│ [↑↓] navigate  [Enter] suggested action              │
│ [:] command palette  [p] configurator  [Esc] close   │
└──────────────────────────────────────────────────────┘
```

Header reports total analyzed, current batch progress (when streaming), and cache status.

Each category shows: priority indicator, count, brief description from the LLM, and the suggested action with its mapped key.

"Read manually" bucket lists individual messages the LLM declined to categorize.

### 7.4 — Inbox Analyzer engine (Piece D, invisible to the user)

A service that:

1. Reads the unread messages currently visible in the active list (already loaded by the `MessagePreloader`, no extra Gmail calls — see §8.1).
2. Splits them into batches of configurable size (default 50). Batching is dictated by LLM context window, not Gmail pagination.
3. For each batch, calls the existing `BulkPromptService.ApplyBulkPromptStream` with either the default analyzer prompt or the user-selected override prompt.
4. Parses the LLM output into structured categories. The default prompt is engineered to return JSON; an override prompt's output is parsed best-effort.
5. Merges categories with matching names across batches.
6. Reports progress after each batch via a callback, allowing the UI to render progressively.

The engine never calls the LLM directly — it composes the existing `BulkPromptService`. Any improvement to that service (caching, retry, parallelism) benefits the analyzer automatically.

## 8. Data flow

### 8.1 — Triggering an Action Plan

```
User presses A (or :action-plan)
        │
        ▼
App reads current context:
  · current filter / search / labels on the list view
  · choice of prompt: default or user override (custom prompt)
        │
        ▼
App takes the unread message IDs from the loaded list
  → already in memory, zero extra Gmail calls (fast mode)
        │
        ▼
Split into batches of configured size
        │
        ▼
For each batch:
  · Build batch payload from subject + from + snippet
  · Call BulkPromptService.ApplyBulkPromptStream
  · Cancellable with ESC
        │
        ▼
Parse and merge categories progressively
        │
        ▼
Action Plan panel re-renders after each batch
```

**Fast mode (default for v1.3.0)**: the analyzer uses only data already in memory — subject, sender, date, labels, and snippet (~120 chars per message). No additional Gmail API calls are made. This keeps the feature cheap (typical batch fits comfortably in a 30k-token context window) and fast (no network round trips for fetching).

Deeper analysis using full message bodies is considered a future enhancement (see §13).

### 8.2 — Executing a suggested quick action

```
User is on category "[a] Archive 18 newsletters"
        │
        ▼
User presses a
        │
        ▼
App resolves: "bulk archive over these 18 IDs"
        │
        ▼
Calls EmailService.BulkArchive
        │
        ▼
Category disappears (or shows strikethrough)
Status bar confirms with success message
```

The action does not invent any new behavior — it uses the exact same bulk methods already used by the manual `v + a` flow.

### 8.3 — Escape hatch: command palette scoped to a category

```
User is on a category with N messages
        │
        ▼
User presses :
        │
        ▼
App opens the command palette
  with those N IDs already marked as selected (virtual bulk)
        │
        ▼
User types any existing command:
  :label work-archive
  :move folder/X
  :trash
        │
        ▼
Command executes scoped to those N
```

No new commands needed. Reuses the existing palette.

### 8.4 — Escape hatch: configurator scoped to a category

```
User is on a category with N messages
        │
        ▼
User presses p
        │
        ▼
Prompts picker opens (Piece A)
  with those N as bulk context
        │
        ▼
User picks:
  · saved prompt → applies directly
  · "✨ Create new with AI" → opens configurator
        │
        ▼
Result rendered in the existing AI panel
```

### 8.5 — Creating a new prompt via the configurator

```
User is in the configurator (entered from picker or category)
        │
        ▼
User types intent in natural language
        │
        ▼
LLM generates prompt draft (streamed)
        │
        ▼
Editable box filled with draft
        │
        ├─ Edit manually for small tweaks
        │
        └─ Type "refine" instruction → LLM regenerates
                       (loop until satisfied)
        │
        ▼
User presses Apply
        │
        ├─ Single message context → existing ApplyPrompt path
        ├─ Bulk selection context → existing ApplyBulkPrompt path
        └─ Action plan category → bulk apply scoped to those IDs
        │
        ▼
Result rendered in AI panel (streaming)
        │
        ▼
(Optional) Ctrl+S → Save dialog
  · Name (LLM-suggested, editable)
  · Description (LLM-suggested, editable)
  · Category (user picks existing or creates new)
        │
        ▼
Saved to existing PromptService library
```

### 8.6 — Override the analyzer with a saved prompt

```
User has a saved prompt: "group by sprint project, output JSON"
        │
        ▼
User runs: :action-plan with-prompt triage-sprint
   (or selects from a prompt picker in the action plan panel)
        │
        ▼
Inbox Analyzer uses that prompt instead of the default
        │
        ▼
Categories rendered reflect what the user's prompt produces
  (not the 5 canonical buckets of the default)
        │
        ▼
Quick-actions assigned heuristically:
  · If the prompt outputs metadata (e.g., "action: archive"
    per category), the analyzer maps it to keys.
  · If not, categories show no quick-action — only escape
    hatches : and p remain available.
```

### 8.7 — Where data lives

| Data | Lives in | Cleared when |
|---|---|---|
| Gmail messages in the list view | `MessagePreloader` cache (existing) | TTL-based, existing behavior |
| In-progress action plan categories | Action plan panel memory only | Panel closes or cancellation |
| Per-batch LLM result | `BulkPromptService` cache (SQLite, existing) | Cache TTL |
| User-saved prompts | `PromptService` storage (SQLite, existing) | Only when user deletes |
| Intent + draft prompt in configurator | Configurator panel memory only | Panel closes without save |

### 8.8 — ESC behavior summary

| Situation | ESC does | What is preserved |
|---|---|---|
| Action plan processing | Cancel current batch | Categories rendered so far stay visible |
| Action plan finished | Close panel | Cached batch results survive in DB |
| Configurator while generating | Cancel LLM call | Previous prompt text |
| Configurator while refining | Cancel LLM call | Prompt prior to refinement |
| Configurator with unsaved changes | Prompt "discard?" | Yes → close, No → stay |
| Quick action in progress | Cancel remaining operations | Already-executed actions stay applied |

All ESC handlers must follow the synchronous pattern documented in `CLAUDE.md` (no `QueueUpdateDraw` inside ESC handlers).

## 9. Error handling

Aligned with the mandatory `ErrorHandler` pattern. All user-facing messages flow through `ErrorHandler` with severity-appropriate colors.

### 9.1 — LLM unavailable

- **First batch of action plan fails** → panel does not open. Red status: "⚠ LLM unavailable. Try again later."
- **Intermediate batch fails** → panel keeps already-processed categories. Header changes to "📋 Action Plan (52 of 120 analyzed — interrupted)". Footer offers "[r] Retry remaining" to re-attempt only the failed batches.
- **Configurator generation fails** → editable box empty with: "⚠ Failed to generate. Edit manually or retry [Ctrl+R]."
- **Configurator refinement fails** → previous prompt preserved. Refine line shows the error.

### 9.2 — LLM output is unparseable

- **Default analyzer prompt** designed to return structured JSON. If output is malformed:
  - One retry with a stricter "repair" prompt.
  - If that also fails: categories are rendered as best-effort from free text, **without quick-action keys**. Only escape hatches `:` and `p` remain. Status: "ℹ Action plan rendered with limited actions — LLM output was malformed."
- **Override (custom) prompts**: graceful degradation by default. Free-text output is split into blocks treated as categories. Quick-actions only appear when the custom prompt emits recognizable metadata. Otherwise the user still has `:` and `p`.

### 9.3 — Empty context

- **Action plan, no unread in current view** → panel does not open. Status: "ℹ No unread messages in current view. Try `:search is:unread` or change filter."
- **Configurator with no message context** → opens in "draft only" mode. Generation and save work, but Apply is disabled with a tooltip.

### 9.4 — Partial bulk action failure

Example: pressing `[a]` to archive 18 newsletters, 3 fail (concurrent move, deletion, etc.).

- Status bar: "✓ Archived 15 of 18 messages. 3 failed."
- Category remains with the 3 failed messages visible plus a red indicator.
- User can retry (`a` again) or use an escape hatch.

### 9.5 — Storage errors

- Save fails (DB lock, disk full) → red status with reason. Prompt remains active in memory; user can retry Ctrl+S.
- Duplicate name on save → dialog "Prompt '<name>' already exists. Overwrite? [y/n]".
- Override prompt ID not found → yellow status "⚠ Prompt ID <n> not found. Using default analyzer prompt." Analyzer falls back to default instead of aborting.

### 9.6 — Universal guarantees

1. No raw output to console (`fmt.Printf`, `log.Printf`) anywhere — all messages go through `ErrorHandler`.
2. No silent error swallowing — every failure is surfaced to the user.
3. Cancellation always available while async work is in flight.
4. Partial state is always preserved when work is interrupted.

## 10. Configurable keybindings

All keys read from `a.Keys.*` with the defaults below. User overrides go in `~/.config/giztui/config.json`.

| Action | Default | Config key |
|---|---|---|
| Open Action Plan | `A` | `Keys.ActionPlan` |
| Cancel Action Plan / configurator | `Esc` | (universal, existing) |
| Execute suggested action in category | Maps to existing per-action keys (`Keys.Archive`, `Keys.MarkAsRead`, `Keys.Labels`, etc.) | — (reuses existing keys; LLM action verb → existing key) |
| Open command palette scoped to category | `:` | (existing) |
| Open configurator scoped to category | `p` | `Keys.Prompt` (existing) |
| Configurator: regenerate prompt | `Ctrl+R` | `Keys.PromptRegenerate` |
| Configurator: apply | `Enter` (in editable box) | `Keys.PromptApply` |
| Configurator: save prompt | `Ctrl+S` | `Keys.SavePrompt` |
| Configurator: test on 1 message (stretch) | `Ctrl+T` | `Keys.PromptTest` |

## 11. Configurable settings

Added to `config.json` under a new `ai_assist` section (or extending existing AI config).

| Setting | Default | Description |
|---|---|---|
| `inbox_analyzer.batch_size` | 50 | Messages per batch sent to the LLM |
| `inbox_analyzer.max_batches` | 10 | Safety cap on total batches per invocation |
| `inbox_analyzer.scope` | `current_view` | Either `current_view` or `literal_unread` |
| `inbox_analyzer.default_prompt_id` | (built-in) | Override the built-in default analyzer prompt |
| `prompt_configurator.streaming` | true | Stream tokens during generation/refinement |

## 12. Commands (parity with shortcuts)

Following the mandatory shortcut/command parity rule from `CLAUDE.md`.

| Command | Aliases | Description |
|---|---|---|
| `:action-plan` | `:plan`, `:ap` | Launch inbox analysis with default prompt |
| `:action-plan with-prompt <name-or-id>` | — | Launch analysis with a custom prompt |
| `:prompt-new` | `:pn` | Open configurator from scratch (no context) |
| `:prompt-refine` | `:pr` | Refine the active prompt in the configurator |
| `:prompt-save` | `:ps` | Save the active prompt in the configurator |

## 13. Testing strategy

### 13.1 — Service-level unit tests

Inbox Analyzer:
- Correctly divides N messages into batches per `BatchSize`.
- Respects `MaxBatches` as a safety cap.
- Merges categories with the same name across batches without duplicating IDs.
- Emits progress in correct order.
- Cleanly cancels when context is canceled (no goroutine leaks).
- Uses override prompt when `CustomPromptID` is set; falls back to default otherwise.
- Fails with a clear error when override prompt is not found.

Prompt Generator:
- Generates a valid prompt for a simple intent.
- Refinement preserves prior content while applying the change.
- Streaming emits tokens in order.
- Returns suggested name and description for the save dialog.

All tests use `mockery`-generated mocks for `AIService` (already present in `internal/services/mocks/`). Zero real LLM calls in CI.

### 13.2 — TUI tests

Action plan panel:
- Renders categories when fed an `ActionPlan` structure.
- Arrow keys navigate between categories.
- Suggested key executes the correct action on the correct IDs.
- `:` and `p` escape hatches open the right tools with pre-scoped selection.
- ESC cancels cleanly and releases focus.
- Header reflects batch progress in real time.

Configurator panel:
- Intent box accepts input and triggers generation on Enter.
- Editable prompt box accepts direct modification.
- Refine line triggers regeneration without losing prior text.
- Apply uses the correct context.
- Ctrl+S opens save dialog with LLM-suggested defaults pre-filled.
- ESC honors the "unsaved changes → confirm" pattern.

Enriched picker:
- "✨ Create new with AI" appears as first item.
- Relaxed filter shows context-compatible prompts.
- Selecting "Create new" closes the picker and opens the configurator.

### 13.3 — Integration scenarios (end-to-end)

Run with `AIService` mocked and `MessageRepository` populated from fixtures:

1. **Action plan happy path**: invoke `:action-plan` → receive 3 progressive categories → press `[a]` → bulk archive executed → category disappears.
2. **Override path**: save a custom prompt → run `:action-plan with-prompt <id>` → verify the custom prompt is used → output rendered.
3. **Configurator full loop**: open picker → "Create new" → type intent → receive prompt → refine → edit manually → apply → see result → save → reopen picker → verify saved prompt is present.
4. **Graceful degradation**: LLM returns malformed output for action plan → categories appear without quick-actions → `:` and `p` still work.

### 13.4 — Explicit edge cases

- Empty inbox / no unread → empty state shown, panel does not open.
- LLM fails on first batch → clear message, no panel.
- LLM fails on intermediate batch → "Retry remaining" works.
- ESC during streaming → clean cancellation, no deadlock.
- Override prompt deleted between invocations → fallback to default.
- Duplicate name on save → "overwrite?" dialog.
- Partial bulk action failure → category shows fail indicator.

### 13.5 — Out of scope for testing

- LLM output parser against real LLMs (use fixtures — it would be brittle).
- Category quality from the LLM (responsibility of the prompt, not the code).
- Performance under real Gmail load (manual smoke test for MVP).

### 13.6 — Coverage targets

- New services: ≥85%.
- New UI: 60–70% (hot paths: render, navigate, action trigger, ESC).
- Integration: the four narrative scenarios are the minimum bar.

### 13.7 — Pre-commit gate

`make pre-commit-check` must pass before any commit (fmt + vet + golangci-lint + essential tests), per the existing `CLAUDE.md` rule.

## 14. Open items (resolved during implementation planning)

These are decisions deliberately deferred from the spec to the implementation plan:

1. **Schema migration for `generated_by` metadata** on saved prompts — opt-in v6 DB migration. Likely deferred to a follow-up if needed.
2. **Heuristic for assigning quick-action keys** to override prompts that emit metadata — exact format of metadata to recognize (e.g., `action: archive` annotation on each category).
3. **"Test on 1 message" (Ctrl+T)** in the configurator — stretch goal for v1, can ship without it.
4. **Default analyzer prompt text** — the exact prompt template needs prompt engineering iteration. Will be developed and tuned during implementation, kept in a dedicated `prompts/default_analyzer.txt` for clarity.

## 15. Future enhancements (explicitly deferred)

- **Deep mode** for the analyzer — fetch full message bodies when snippet+metadata are insufficient.
- **Hybrid on-demand mode** — fast mode by default with an option to deep-analyze a specific ambiguous category.
- **Persistent action plans** across sessions, with diff against current inbox state.
- **Per-account analyzer profiles** — different default prompts per Gmail account.
- **Action plan history** — see past plans and the actions executed.
- **Prompt versioning** — track edits to saved prompts over time.

## 16. References

- `CLAUDE.md` — project architectural rules (service-first, ErrorHandler, ActivePicker, theming, ESC patterns, command parity).
- `internal/services/interfaces.go` — existing service contracts including the discovered `BulkPromptService` interface.
- `internal/services/bulk_prompt_service.go` — existing implementation reused by the inbox analyzer.
- `internal/tui/bulk_prompts.go` — existing bulk prompt picker, reused as base for the enriched picker.
- `docs/ARCHITECTURE.md` — architectural conventions.
- `docs/THEMING.md` — component color guidelines.
- `docs/KEYBOARD_SHORTCUTS.md` — keyboard shortcut conventions and command parity examples.
