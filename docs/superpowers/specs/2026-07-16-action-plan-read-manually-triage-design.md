# Action Plan — "Read manually" assisted triage (design)

**Date:** 2026-07-16
**Status:** approved (design), pending spec review → implementation plan
**Scope decision (2026-07-04):** this iteration revisits ONLY the "Read manually" bucket of the Action Plan. The actionable ("No action") categories stay exactly as they are.

## Goal

Turn "Read manually" from a flat, blind list into a **sender-grouped, AI-assisted triage surface** so the user can empty it fast. Three user-approved objectives, unified:

1. **AI help per email** — on demand, a one-line hint + a suggested action, acceptable with one key.
2. **Grouping** — by sender (deterministic, no AI).
3. **Fast actions** — accept a suggestion (per email or per sender group) and/or apply one chosen action to a whole sender group.

Non-goal (explicitly not selected by the user): reducing *how many* emails land in "Read manually" (no second decisive classification pass, no auto-apply).

## Context (current behavior)

- `ActionPlan.ReadManually []AnalyzerMessage` (`internal/services/interfaces.go`) holds the messages the LLM declined to categorize (plus strict-label no-match and degraded batches — see `inbox_analyzer_service.go`).
- `AnalyzerMessage` carries `ID, Subject, From, Snippet, Body` — `From` is present, so sender grouping is feasible with no new data.
- In the panel (`internal/tui/action_plan.go`) it is a single pseudo-node `ref == -1` rendered `▸ Read manually · N` (line ~544), with a **flat** list of email children (~627–660). Per-email you can move (`m`), read (Enter), or "Keep (read manually)" via the move chooser (`action_plan_move.go`).

## Decisions (from brainstorming)

| Question | Decision |
|---|---|
| What AI gives per email | Hint + suggested action, one-key accept |
| When the AI pass runs | **On demand**, triggered by a key on the bucket/group (0 cost if untouched) |
| Grouping dimension | **By sender** (deterministic) |
| Group-level action | **Both**: "apply one chosen action to the whole group" AND "accept the group's AI suggestions" |
| Fit/approach | **A** — enrich the existing "Read manually" node in place; reuse the tree, move, and bulk infra |

## Architecture (Approach A)

Enrich the existing `ReadManually` subtree in place. Two layers of change:

- **Service (business logic):** a new on-demand method on `InboxAnalyzerService` that, given the read-manually messages, returns one suggestion per message. No auto-invocation.
- **TUI (presentation):** group `ReadManually` by normalized sender into an extra tree level; render AI annotations inline when present; wire two new keys and reuse the existing action chooser + bulk executor for the actions.

### Service

`internal/services/interfaces.go`:

```go
// ReadManuallySuggestion is the AI's per-email assist result for a read-manually message.
type ReadManuallySuggestion struct {
    ID     string // AnalyzerMessage.ID
    Hint   string // one short line: what it is / why it's here
    Action string // "archive" | "mark_read" | "trash" | "label" | "read"  ("read" = no action)
    Label  string // set only when Action == "label"
}

// AssistReadManually enriches read-manually messages on demand: one suggestion per input
// message, in input order. Honors opts.StrictLabels/AvailableLabels (a label suggestion that
// is not an existing label degrades to "read"). Never applies anything — suggestions only.
AssistReadManually(ctx context.Context, msgs []AnalyzerMessage, opts InboxAnalyzerOptions) ([]ReadManuallySuggestion, error)
```

- Implemented in `internal/services/inbox_analyzer_service.go` (or a sibling `inbox_analyzer_assist.go` if `inbox_analyzer_service.go` is already large — split by responsibility).
- Reuses `InboxAnalyzerOptions.{AvailableLabels, StrictLabels}` (already exist).
- Own prompt: `assistReadManuallyPrompt` embedded from a new `internal/services/inbox_analyzer_assist_prompt.txt`, parallel to `defaultAnalyzerPrompt`. Editable/overridable like the other analyzer prompts; batched like `Analyze` (respect `batch_size`/`max_batches`).
- Parsing mirrors the existing analyzer JSON parsing: reconcile missing IDs to `{Action:"read"}` (never drop a message); an unknown/empty action → `"read"`; `label` with a name not in `AvailableLabels` (when `StrictLabels`) → `"read"`.

### TUI

`internal/tui/action_plan.go` + a new focused file `internal/tui/action_plan_read_manually.go` for the grouping/rendering/actions specific to this subtree (keeps `action_plan.go` from growing further).

1. **Sender grouping (pure, testable):**
   ```go
   type readManuallyGroup struct {
       senderKey  string               // normalized address, e.g. "ana@x.com"
       senderDisp string               // display, e.g. "Ana García <ana@x.com>" (first seen)
       msgs       []services.AnalyzerMessage
   }
   func groupReadManuallyBySender(msgs []services.AnalyzerMessage) []readManuallyGroup
   ```
   - Normalize with `net/mail.ParseAddress(From)` → lowercase address; fall back to the raw trimmed/lowercased `From` if it doesn't parse.
   - Order groups by descending size, then by `senderDisp` for stable ties. Messages within a group keep plan order.

2. **Suggestions state:** `map[string]services.ReadManuallySuggestion` keyed by message ID, held on `actionPlanState` (nil until the assist key is pressed). Rendering reads it; empty = no annotations yet.

3. **Tree model:** extend the read-manually subtree to three levels: pseudo-node → sender-group nodes → email leaves. Top-level nodes keep their existing int index (`i`: categories `0..n-1`, read-manually `-1`) and the string-keyed `expanded` map (`catExpandKey` returns the category name or the `"\x00read-manually"` sentinel). Reuse that same string-keyed mechanism for the new level: a `senderExpandKey(senderKey)` returning `"\x00read-manually:" + senderKey`, so each sender group expands/collapses independently and category navigation is untouched. Email leaves need no expand key (they don't expand).

4. **Rendering:**
   - Sender header: `▸ <senderDisp> · <n>`.
   - Email leaf: `<Subject>` and, when a suggestion exists, `<Subject> — 💡 <Hint> · sugiere: <ActionLabel>` (`ActionLabel` via the existing `actionVerbLabel`; `"read"` → no "sugiere" suffix).

5. **Keys (two new, configurable):**
   - `keys.assist_read_manually` — trigger the on-demand AI pass for the whole `ReadManually` bucket (usable from the pseudo-node or any read-manually descendant). Progress via `ShowProgress`; non-blocking goroutine; fills the suggestions map; re-render.
   - `keys.accept_suggestion` — **context-sensitive accept**: on an email leaf, apply *that* email's suggested action; on a sender header, apply each email-in-group's suggested action (skip `"read"`). No-op with an info message if no suggestions yet.
   - Reuse the **existing action chooser** (the `m`/move chooser in `action_plan_move.go`, which already offers actions incl. Keep) for *"apply one chosen action to the whole sender group"* — invoked on a sender header. This satisfies the "both" group-level requirement without a third new key.

6. **Actions execution:** route every apply through the existing `runActionPlanBulkOp` + progress helpers; remove applied messages from `plan.ReadManually` via the existing `removeReadManuallyByID`; drop a sender group when it empties; drop the whole pseudo-node when `ReadManually` empties. Reuse `analyzerMessageFor`/metadata plumbing already in `action_plan_move.go`.

## Data flow

1. Action Plan analyze runs (unchanged) → `plan.ReadManually` populated.
2. User navigates to "Read manually" → shown grouped by sender immediately (deterministic, 0 tokens).
3. User presses `assist_read_manually` → `AssistReadManually(ctx, plan.ReadManually, opts)` → suggestions map → re-render with hints/suggested actions.
4. User accepts (email or group) or applies a chosen action to a group → `runActionPlanBulkOp` → messages removed from `ReadManually` → re-render.

## Edge cases & error handling

- **No LLM / assist error:** keep the sender-grouped list without annotations; show a warning (`ShowWarning`). Grouping never depends on the AI.
- **No clear action:** AI returns `"read"` → message stays in the bucket (correct: the bucket still means "actually read these").
- **strict_labels:** label suggestions restricted to existing labels; non-existent → `"read"`.
- **Confirmation:** group-level actions (especially trash / anything touching several messages) use the app-standard **two-press status-bar confirmation** (`ShowPersistentMessage` + pending flag, Esc cancels) — consistent with the rest of the app ([[feedback-confirmation-ux-pattern]]). Single-email accept is direct.
- **Re-running assist:** re-analyzes the *current* `ReadManually` (already-accepted/moved messages are gone). Overwrites the suggestions map.
- **Threading:** the assist goroutine must not call `QueueUpdateDraw`-blocking ErrorHandler methods synchronously on the event loop; follow the established goroutine pattern ([[errorhandler-eventloop-deadlock]]). Streaming is not used (single structured result), so no streaming-callback constraints.

## Testing

- **Service (unit):** `AssistReadManually` — JSON parsing into suggestions; missing-ID reconcile → `"read"`; strict-label non-match → `"read"`; degrade path returns an error cleanly. Hand-rolled fake LLM (existing analyzer test pattern).
- **TUI (pure unit):** `groupReadManuallyBySender` — sender normalization (`Name <addr>` and bare `addr` collapse together), ordering by size, stable ties, within-group order preserved. Accept-removes-from-bucket / empties-group logic (pure list ops over `ReadManually`).
- **TUI (component):** 3-level render + navigation; sender-header group action calls the correct bulk op; accept key is context-sensitive (email vs header).

## Config / Definition of Done

- **Two new configurable keys** (`assist_read_manually`, `accept_suggestion`) — added to `DefaultConfig()`/`KeyBindings`, seeded by `LoadConfig`, surfaced by config self-migration for existing users' `config.json`, and documented in the in-app `:help` screen and `docs/KEYBOARD_SHORTCUTS.md`.
- **Assist prompt** shipped as a default (`inbox_analyzer_assist_prompt.txt`) and overridable via config like the other analyzer prompts; migration surfaces the key.
- **No on/off toggle:** on-demand by nature — 0 cost if unused.
- Update `docs/` where the Action Plan / analyzer is described.

## Out of scope

- The "No action" / actionable categories (unchanged, per the 2026-07-04 scope decision).
- Reducing how many messages land in "Read manually" (a more decisive re-classification pass) — not selected.
- Auto-running the assist pass (kept strictly on-demand).
