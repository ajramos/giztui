# Analyzer Faithful Labels — Design

**Date:** 2026-06-20
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `feat/analyzer-faithful-labels`

## Goal

Stop the Inbox Action Plan analyzer from inventing or duplicating labels. With a weak local model,
the `label` action proposes label names that don't match the user's real labels, and the dispatch
creates them — polluting the label list. In strict mode, the analyzer may use only labels that
already exist; an email whose suggested label has no match is sent to "review manually" instead of
getting a new/duplicate label created.

## Problem (diagnosed)

The mechanism to pass the user's labels to the model already exists (`userLabelNames()` →
`AvailableLabels` → the "## Existing labels" prompt block). Two weaknesses:

1. **Soft prompt.** The block says *"PREFER an exact name from this list when one fits. Only invent a
   NEW label when none fits"*. A weak local model (Ollama) ignores this and invents labels.
2. **Create-on-miss dispatch.** `resolveOrCreateLabelID` matches existing labels case-insensitively
   (`EqualFold`) and, on no match, **creates** the label — so any invented or variant name becomes a
   new label.

User confirmed (E2E, local model): it both invents new labels and creates near-duplicates.

## Decisions (from brainstorming)

- **Strict mode:** the analyzer uses only existing user labels. Never create new ones from analysis.
- **No match → "review manually":** an email whose suggested label matches no existing label is
  reclassified into the plan's read-manually group (not silently left, not force-fit).
- **Conservative matching:** tolerate case and surrounding whitespace only. NO fuzzy/nearest-match
  ("Trabajo Urgente" must NOT be forced into "Urgente" — that re-introduces wrong guesses).
- **Configurable, default ON.** Auto-creating labels from an analysis is surprising for anyone, so
  strict is the safer default; users who want the old create-on-miss behavior set it to `false`.
- **Scope:** only the Action Plan analyzer. Manual labeling (the user applies a label with `l`) is
  untouched — there, creating a new label is intentional.

## Configuration

New field on `config.InboxAnalyzerConfig`: `StrictLabels bool` (json `strict_labels`), default `true`
in `DefaultInboxAnalyzerConfig()`. Self-migration must surface the key to existing `config.json`
files on other machines.

## Architecture (service-first)

All logic lives in `InboxAnalyzerService` (which builds the plan) and reuses `AvailableLabels`
(already passed in via `InboxAnalyzerOptions`).

### 1. Pure helper — `resolveExistingLabel`

```go
// resolveExistingLabel returns the canonical existing label that matches `suggested` (case- and
// whitespace-insensitive), or ok=false when none matches. No fuzzy matching.
func resolveExistingLabel(suggested string, existing []string) (canonical string, ok bool)
```

Normalize by `strings.TrimSpace` + case-fold for comparison; return the existing label's original
casing as `canonical`.

### 2. New option — `InboxAnalyzerOptions.StrictLabels bool`

Fed from `config.InboxAnalyzer.StrictLabels` at the call sites in `internal/tui/action_plan.go`
(the analyze flow) and `internal/tui/action_plan_prompt.go` (the prompt preview), wherever
`AvailableLabels` is already set.

### 3. Label validation step in `Analyze`

After the categories are assembled (in `InboxAnalyzerServiceImpl.Analyze`, before returning the
final `*ActionPlan`), run a pure post-processing pass `enforceLabelPolicy(categories, messages,
availableLabels, strict)`:

For each category with `Action == "label"`:
- `resolveExistingLabel(cat.Label, availableLabels)`:
  - **match** → set `cat.Label` to the canonical existing name (fixes case/whitespace drift).
  - **no match** AND `strict` → move that category's `MessageIDs` into `plan.ReadManually` (using an
    id→`AnalyzerMessage` lookup built from `messages`) and drop the category.
  - **no match** AND not strict → leave the category unchanged (legacy create-on-dispatch behavior).

Return the adjusted categories + the augmented read-manually set. This runs once on the final plan,
so partial-batch reconciliation (the existing read_manually invariant logic) is unaffected.

### 4. Dispatch defense — `resolveOrCreateLabelID`

When strict mode is on, `resolveOrCreateLabelID` resolves existing labels (current `EqualFold`
scan, optionally widened to trim) but does NOT create on miss — it returns an error/skip instead.
With step 3 already removing invalid "label" categories, this is belt-and-suspenders, but it
guarantees no creation path remains reachable from the analyzer in strict mode. The method needs the
strict flag (pass it in, or read `a.Config.InboxAnalyzer.StrictLabels` at the call site).

### 5. Prompt reinforcement (complementary)

Change `availableLabelsBlock` from the soft "PREFER … when one fits" to an imperative instruction:
*"Use ONLY a label from this exact list for the `label` action. Do NOT invent new labels. If none of
them fits the email, put the email in read_manually instead."* This helps capable models; the code
(steps 3–4) is the real guardrail for weak ones. (If we want the wording to depend on strict mode,
pass the flag into the block builder; otherwise the imperative wording is fine in both modes since
non-strict simply still allows creation at dispatch.)

## Error handling

- Empty `AvailableLabels` (user has no labels, or label fetch failed): strict mode would send every
  "label" email to read-manually. Guard: when `availableLabels` is empty, **skip** the strict
  reclassification (treat as "can't enforce") so the analyzer degrades to current behavior rather
  than dumping everything into read-manually.
- All reclassification is pure/in-memory; no new Gmail calls.

## Testing

- `resolveExistingLabel`: exact, case-only, whitespace-only matches return the canonical name;
  genuinely different name → ok=false; empty inputs → ok=false.
- `enforceLabelPolicy` (pure): an invented label's messages land in ReadManually and the category is
  dropped; a case-variant label keeps its category with the canonical name; non-strict leaves
  categories untouched; empty availableLabels skips enforcement.
- Config default `StrictLabels == true`.
- Existing analyzer tests (`Analyze` happy path, reconciliation) stay green.

## Out of scope (YAGNI)

- Fuzzy/nearest-label matching.
- Touching manual labeling (`l` key) or any non-analyzer label creation.
- Letting the user pick a replacement label inline for a no-match category (future idea; for now it
  goes to read-manually and the user labels it manually).

## Definition of Done

- [ ] `config.InboxAnalyzer.StrictLabels` (default true) + self-migration of the key.
- [ ] `resolveExistingLabel` helper + `InboxAnalyzerOptions.StrictLabels` + `enforceLabelPolicy` in `Analyze`.
- [ ] `resolveOrCreateLabelID` honors strict (no create-on-miss).
- [ ] `availableLabelsBlock` imperative wording.
- [ ] Empty-labels guard (skip enforcement, no mass read-manually).
- [ ] In-app `:help` / config docs note `strict_labels`.
- [ ] Tests (helper + enforcement + config default); existing analyzer tests green.
- [ ] `make pre-commit-check` green; manual E2E on the user's Mac with the local model.
