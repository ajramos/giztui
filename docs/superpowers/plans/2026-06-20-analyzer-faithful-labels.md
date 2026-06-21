# Analyzer Faithful Labels Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In strict mode (default on), the Inbox Action Plan analyzer uses only existing user labels; an email whose suggested label matches none is reclassified to "review manually" instead of creating a new/duplicate label.

**Architecture:** Service-first. A pure `resolveExistingLabel` helper + a pure `enforceLabelPolicy` pass run inside `InboxAnalyzerService.Analyze` (reusing the `AvailableLabels` already passed in). A new `StrictLabels` config flag flows through `InboxAnalyzerOptions`. The dispatch (`resolveOrCreateLabelID`) gains a no-create-on-miss guard as defense in depth. The "## Existing labels" prompt block is made imperative.

**Tech Stack:** Go, existing `InboxAnalyzerService`, `config.InboxAnalyzerConfig`.

---

## Reference facts (verified in code)

- `internal/services/interfaces.go:1005` — `ActionPlanCategory{ Name, Priority, Description, Action, Label, MessageIDs }`. `Action == "label"` uses `Label`.
- `internal/services/interfaces.go:1016` — `ActionPlan{ ..., Categories []ActionPlanCategory, ReadManually []AnalyzerMessage, Degraded bool }`.
- `internal/services/interfaces.go:1027` — `InboxAnalyzerOptions{ BatchSize, MaxBatches, CustomPromptText, UserRules, BodyCharLimit, AvailableLabels }`.
- `AnalyzerMessage{ ID, Subject, From, Snippet, Body }` (interfaces.go ~998).
- `internal/services/inbox_analyzer_service.go:416` — `Analyze` ends with `return plan, nil`; insert the enforcement pass just before it.
- `internal/services/inbox_analyzer_service.go:291` — `availableLabelsBlock(labels []string) string` (the "## Existing labels" text). Tested by `TestAvailableLabelsBlock` (inbox_analyzer_service_test.go:375), which asserts the literal `"PREFER an exact name"`.
- `internal/config/config.go:296` — `InboxAnalyzerConfig`; `DefaultInboxAnalyzerConfig()` at line 711.
- Call sites that build `InboxAnalyzerOptions`: `internal/tui/action_plan.go:386` (analyze flow) and `internal/tui/action_plan_prompt.go:28` (prompt preview). Both already set `AvailableLabels`.
- `internal/tui/action_plan.go:913` — `resolveOrCreateLabelID(labelService, name)`; only caller is `applyActionPlanLabel` (action_plan.go:905).
- `inbox_analyzer_service_test.go` exists with tests like `TestAvailableLabelsBlock`, `TestParseAnalyzerResponse`, `TestAnalyze_*` (use `stubAIService`-style mocks).

---

## Task 1: Config flag `StrictLabels` (default true)

**Files:**
- Modify: `internal/config/config.go` (`InboxAnalyzerConfig` line 296; `DefaultInboxAnalyzerConfig` line 711)
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go`:

```go
func TestDefaultInboxAnalyzer_StrictLabels(t *testing.T) {
	c := DefaultInboxAnalyzerConfig()
	if !c.StrictLabels {
		t.Errorf("StrictLabels should default to true")
	}
}
```

- [ ] **Step 2: Run it, verify FAIL (field undefined)**

Run: `go test ./internal/config/ -run TestDefaultInboxAnalyzer_StrictLabels -v`
Expected: FAIL — `StrictLabels` undefined.

- [ ] **Step 3: Add the field + default**

In `internal/config/config.go`, add to `InboxAnalyzerConfig` (after `BodyCharLimit`):

```go
	StrictLabels    bool   `json:"strict_labels"`     // analyzer uses only existing labels; no creating new ones (default true)
```

In `DefaultInboxAnalyzerConfig()`, add to the returned struct literal:

```go
		StrictLabels:    true,
```

- [ ] **Step 4: Run the test, verify PASS**

Run: `go test ./internal/config/ -run TestDefaultInboxAnalyzer_StrictLabels -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/config/config.go internal/config/config_test.go
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add inbox_analyzer.strict_labels (default true)"
```

---

## Task 2: Pure `resolveExistingLabel` helper

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go`
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/services/inbox_analyzer_service_test.go`:

```go
func TestResolveExistingLabel(t *testing.T) {
	existing := []string{"Work", "Receipts", "Newsletters"}
	// exact, case-insensitive, and whitespace-tolerant matches return the canonical existing name.
	for _, in := range []string{"Work", "work", "  WORK  "} {
		if got, ok := resolveExistingLabel(in, existing); !ok || got != "Work" {
			t.Errorf("resolveExistingLabel(%q) = %q,%v; want Work,true", in, got, ok)
		}
	}
	// genuinely different name → no match (no fuzzy).
	if _, ok := resolveExistingLabel("Work Urgent", existing); ok {
		t.Error("'Work Urgent' must NOT match 'Work' (no fuzzy)")
	}
	// empty inputs → no match.
	if _, ok := resolveExistingLabel("", existing); ok {
		t.Error("empty suggested → no match")
	}
	if _, ok := resolveExistingLabel("Work", nil); ok {
		t.Error("empty existing → no match")
	}
}
```

- [ ] **Step 2: Run it, verify FAIL**

Run: `go test ./internal/services/ -run TestResolveExistingLabel -v`
Expected: FAIL — `resolveExistingLabel` undefined.

- [ ] **Step 3: Implement the helper**

Add to `internal/services/inbox_analyzer_service.go` (near `availableLabelsBlock`):

```go
// resolveExistingLabel returns the canonical existing label matching `suggested` (case- and
// surrounding-whitespace-insensitive), or ok=false when none matches. No fuzzy matching.
func resolveExistingLabel(suggested string, existing []string) (string, bool) {
	s := strings.TrimSpace(suggested)
	if s == "" {
		return "", false
	}
	for _, e := range existing {
		if strings.EqualFold(strings.TrimSpace(e), s) {
			return e, true
		}
	}
	return "", false
}
```

(`strings` is already imported in this file.)

- [ ] **Step 4: Run the test, verify PASS**

Run: `go test ./internal/services/ -run TestResolveExistingLabel -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(services): add resolveExistingLabel (case/whitespace-tolerant, no fuzzy)"
```

---

## Task 3: `StrictLabels` option + `enforceLabelPolicy` pass

**Files:**
- Modify: `internal/services/interfaces.go` (`InboxAnalyzerOptions`)
- Modify: `internal/services/inbox_analyzer_service.go` (add `enforceLabelPolicy`; call it in `Analyze`)
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Add the option field**

In `internal/services/interfaces.go`, add to `InboxAnalyzerOptions` (after `AvailableLabels`):

```go
	StrictLabels     bool     // when true, the "label" action may only use an existing label; no-match emails go to read-manually
```

- [ ] **Step 2: Write the failing test for `enforceLabelPolicy`**

Add to `internal/services/inbox_analyzer_service_test.go`:

```go
func TestEnforceLabelPolicy(t *testing.T) {
	msgs := []AnalyzerMessage{{ID: "1"}, {ID: "2"}, {ID: "3"}}
	available := []string{"Work", "Receipts"}

	// strict: a matching (case-variant) label is kept + canonicalized; an invented label's
	// messages go to ReadManually and its category is dropped.
	plan := &ActionPlan{
		Categories: []ActionPlanCategory{
			{Name: "A", Action: "label", Label: "work", MessageIDs: []string{"1"}},      // case variant → keep as "Work"
			{Name: "B", Action: "label", Label: "Project X", MessageIDs: []string{"2"}}, // invented → read-manually
			{Name: "C", Action: "archive", MessageIDs: []string{"3"}},                   // non-label → untouched
		},
	}
	enforceLabelPolicy(plan, msgs, available, true)

	if len(plan.Categories) != 2 {
		t.Fatalf("invented label category should be dropped; got %d categories", len(plan.Categories))
	}
	if plan.Categories[0].Label != "Work" {
		t.Errorf("label should be canonicalized to 'Work', got %q", plan.Categories[0].Label)
	}
	if len(plan.ReadManually) != 1 || plan.ReadManually[0].ID != "2" {
		t.Errorf("invented label's message should be in ReadManually, got %+v", plan.ReadManually)
	}

	// non-strict: invented categories are left untouched (legacy create-on-dispatch).
	plan2 := &ActionPlan{Categories: []ActionPlanCategory{
		{Name: "B", Action: "label", Label: "Project X", MessageIDs: []string{"2"}},
	}}
	enforceLabelPolicy(plan2, msgs, available, false)
	if len(plan2.Categories) != 1 {
		t.Errorf("non-strict must keep invented category, got %d", len(plan2.Categories))
	}

	// empty available labels: enforcement is skipped even in strict mode (degrade to current behavior).
	plan3 := &ActionPlan{Categories: []ActionPlanCategory{
		{Name: "B", Action: "label", Label: "Project X", MessageIDs: []string{"2"}},
	}}
	enforceLabelPolicy(plan3, msgs, nil, true)
	if len(plan3.Categories) != 1 || len(plan3.ReadManually) != 0 {
		t.Errorf("empty available labels must skip enforcement, got %d cats / %d read-manually", len(plan3.Categories), len(plan3.ReadManually))
	}
}
```

- [ ] **Step 3: Run it, verify FAIL**

Run: `go test ./internal/services/ -run TestEnforceLabelPolicy -v`
Expected: FAIL — `enforceLabelPolicy` undefined.

- [ ] **Step 4: Implement `enforceLabelPolicy`**

Add to `internal/services/inbox_analyzer_service.go`:

```go
// enforceLabelPolicy resolves each "label" category against the existing user labels (in place).
// A match canonicalizes the category's Label. In strict mode (with a non-empty label set), a no-match
// category's messages are moved to ReadManually and the category is dropped — never creating a new
// label. With no available labels, enforcement is skipped so the analyzer degrades to prior behavior.
func enforceLabelPolicy(plan *ActionPlan, messages []AnalyzerMessage, availableLabels []string, strict bool) {
	if plan == nil {
		return
	}
	byID := make(map[string]AnalyzerMessage, len(messages))
	for _, m := range messages {
		byID[m.ID] = m
	}
	kept := plan.Categories[:0]
	for _, c := range plan.Categories {
		if c.Action != "label" {
			kept = append(kept, c)
			continue
		}
		if canonical, ok := resolveExistingLabel(c.Label, availableLabels); ok {
			c.Label = canonical
			kept = append(kept, c)
			continue
		}
		if strict && len(availableLabels) > 0 {
			for _, id := range c.MessageIDs {
				if m, found := byID[id]; found {
					plan.ReadManually = append(plan.ReadManually, m)
				}
			}
			continue // drop the invented-label category
		}
		kept = append(kept, c) // non-strict (or no label set): leave as-is
	}
	plan.Categories = kept
}
```

- [ ] **Step 5: Call it in `Analyze`**

In `internal/services/inbox_analyzer_service.go`, the success return at the end of `Analyze` is:

```go
	return plan, nil
}
```

Change it to:

```go
	enforceLabelPolicy(plan, messages, opts.AvailableLabels, opts.StrictLabels)
	return plan, nil
}
```

(There is exactly one such terminal `return plan, nil` at the end of the batch loop; the earlier `return plan, err` on intermediate-batch failure is left unchanged — that is an error path.)

- [ ] **Step 6: Run tests + build**

Run: `go test ./internal/services/ -run 'EnforceLabelPolicy|ResolveExistingLabel|Analyze' -v 2>&1 | tail -8`
Expected: PASS.
Run: `go build ./internal/...` → success.

- [ ] **Step 7: Commit**

```bash
gofmt -w internal/services/interfaces.go internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git add internal/services/interfaces.go internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(services): enforce strict label policy in Analyze (no-match → read-manually)"
```

---

## Task 4: Wire `StrictLabels` from config at the call sites

**Files:**
- Modify: `internal/tui/action_plan.go:386` (analyze flow)
- Modify: `internal/tui/action_plan_prompt.go:28` (prompt preview)

- [ ] **Step 1: Wire the analyze flow**

In `internal/tui/action_plan.go`, the options literal at line ~386 is:

```go
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules, BodyCharLimit: bodyCharLimit, AvailableLabels: availableLabels},
```

Change it to:

```go
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules, BodyCharLimit: bodyCharLimit, AvailableLabels: availableLabels, StrictLabels: a.Config.InboxAnalyzer.StrictLabels},
```

- [ ] **Step 2: Wire the prompt preview**

In `internal/tui/action_plan_prompt.go`, the options literal at line ~28 is:

```go
	opts := services.InboxAnalyzerOptions{
		CustomPromptText: state.customPromptText,
		UserRules:        userRules,
		BodyCharLimit:    a.Config.InboxAnalyzer.BodyCharLimit,
		AvailableLabels:  a.userLabelNames(),
	}
```

Change it to:

```go
	opts := services.InboxAnalyzerOptions{
		CustomPromptText: state.customPromptText,
		UserRules:        userRules,
		BodyCharLimit:    a.Config.InboxAnalyzer.BodyCharLimit,
		AvailableLabels:  a.userLabelNames(),
		StrictLabels:     a.Config.InboxAnalyzer.StrictLabels,
	}
```

- [ ] **Step 3: Build**

Run: `go build ./internal/tui/...` → success.

- [ ] **Step 4: Commit**

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_prompt.go
git add internal/tui/action_plan.go internal/tui/action_plan_prompt.go
git commit -m "feat(tui): pass strict_labels config into the analyzer"
```

---

## Task 5: Dispatch defense — no create-on-miss in strict mode

**Files:**
- Modify: `internal/tui/action_plan.go` (`resolveOrCreateLabelID` ~line 913)

- [ ] **Step 1: Guard creation behind strict mode**

In `internal/tui/action_plan.go`, the function is:

```go
// resolveOrCreateLabelID finds a label by name (case-insensitive) or creates it.
func (a *App) resolveOrCreateLabelID(labelService services.LabelService, name string) (string, error) {
	labels, err := labelService.ListLabels(a.ctx)
	if err != nil {
		return "", err
	}
	for _, l := range labels {
		if strings.EqualFold(l.Name, name) {
			return l.Id, nil
		}
	}
	created, err := labelService.CreateLabel(a.ctx, name)
	if err != nil {
		return "", err
	}
	return created.Id, nil
}
```

Replace it with:

```go
// resolveOrCreateLabelID finds a label by name (case-insensitive). It creates a missing label only
// when strict-labels mode is OFF; in strict mode a missing label is an error (the analyzer's
// enforceLabelPolicy already drops such categories, so this is defense in depth).
func (a *App) resolveOrCreateLabelID(labelService services.LabelService, name string) (string, error) {
	labels, err := labelService.ListLabels(a.ctx)
	if err != nil {
		return "", err
	}
	for _, l := range labels {
		if strings.EqualFold(l.Name, name) {
			return l.Id, nil
		}
	}
	if a.Config.InboxAnalyzer.StrictLabels {
		return "", fmt.Errorf("label %q does not exist (strict labels mode)", name)
	}
	created, err := labelService.CreateLabel(a.ctx, name)
	if err != nil {
		return "", err
	}
	return created.Id, nil
}
```

(`fmt` is already imported in action_plan.go.)

- [ ] **Step 2: Build + tests**

Run: `go build ./internal/tui/...` → success.
Run: `go test ./internal/tui/ -run ActionPlan 2>&1 | tail -3` → `ok`.

- [ ] **Step 3: Commit**

```bash
gofmt -w internal/tui/action_plan.go
git add internal/tui/action_plan.go
git commit -m "feat(tui): resolveOrCreateLabelID does not create in strict mode"
```

---

## Task 6: Imperative "## Existing labels" prompt wording

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go` (`availableLabelsBlock` ~line 291)
- Test: `internal/services/inbox_analyzer_service_test.go` (`TestAvailableLabelsBlock` ~line 375)

- [ ] **Step 1: Update the test to the new wording**

In `internal/services/inbox_analyzer_service_test.go`, replace the assertion (line ~383):

```go
	if !strings.Contains(got, "PREFER an exact name") {
		t.Fatalf("expected the prefer-existing instruction, got:\n%s", got)
	}
```

with:

```go
	if !strings.Contains(got, "Use ONLY a label from this exact list") {
		t.Fatalf("expected the imperative existing-labels instruction, got:\n%s", got)
	}
```

- [ ] **Step 2: Run it, verify FAIL**

Run: `go test ./internal/services/ -run TestAvailableLabelsBlock -v`
Expected: FAIL — block still has the old wording.

- [ ] **Step 3: Update `availableLabelsBlock`**

In `internal/services/inbox_analyzer_service.go`, the return is:

```go
	return "## Existing labels\n" +
		"These labels already exist in the mailbox: " + strings.Join(clean, ", ") + "\n" +
		"For the \"label\" action, PREFER an exact name from this list when one fits. Only invent a " +
		"NEW label when none fits; if you do, keep it short kebab-case and note \"(new)\" in the " +
		"category description."
```

Replace it with:

```go
	return "## Existing labels\n" +
		"These labels already exist in the mailbox: " + strings.Join(clean, ", ") + "\n" +
		"For the \"label\" action, Use ONLY a label from this exact list. Do NOT invent new labels. " +
		"If none of them fits the email, put the email in read_manually instead."
```

- [ ] **Step 4: Run the test, verify PASS**

Run: `go test ./internal/services/ -run TestAvailableLabelsBlock -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(services): imperative existing-labels prompt (use only this list, no invents)"
```

---

## Task 7: Docs / `:help` + final verification

**Files:**
- Modify: config docs (`docs/CONFIGURATION.md`) — `inbox_analyzer` section
- Modify: `internal/tui/app.go` only if `:help` documents inbox_analyzer config keys

- [ ] **Step 1: Document the config key**

Run: `grep -nE "inbox_analyzer|include_body|body_char_limit|batch_size" docs/CONFIGURATION.md | head`
In the `inbox_analyzer` section of `docs/CONFIGURATION.md`, add a row/line for `strict_labels`:
- `strict_labels` (boolean, default `true`): the Action Plan analyzer may only apply labels you already have. An email whose suggested label matches none of yours goes to "review manually" instead of creating a new label. Set `false` to allow the analyzer to create new labels (legacy behavior). If you have no labels at all, this is skipped.
Match the formatting of the surrounding keys (`batch_size`, `include_body`, `body_char_limit`).

- [ ] **Step 2: Check `:help`**

Run: `grep -niE "inbox_analyzer|strict|analyzer config" internal/tui/app.go | grep -i help | head`
- If `:help` enumerates `inbox_analyzer` config keys, add a brief mention of `strict_labels` in the same style. If it does not (config keys are documented only in CONFIGURATION.md), make no change and note that the docs are the source of truth.

- [ ] **Step 3: Canonical pre-commit check**

Run: `make pre-commit-check`
Expected: `All pre-commit checks passed!`

- [ ] **Step 4: Full test suite**

Run: `make test 2>&1 | grep -E "^(ok|FAIL)" | grep -v "no test files"`
Expected: all `ok`, no `FAIL`.

- [ ] **Step 5: Build**

Run: `make build`
Expected: `Built build/giztui ...`.

- [ ] **Step 6: Commit (if docs/help changed) and finish the branch**

```bash
git add docs/CONFIGURATION.md internal/tui/app.go
git commit -m "docs: document inbox_analyzer.strict_labels"
```

Then use the superpowers:finishing-a-development-branch skill. Manual E2E on the user's Mac with the local model: run the Action Plan; confirm the "label" categories only use existing labels, that an email the model wanted to label with a non-existent name lands in "review manually", and that no new/duplicate labels get created. Do NOT push/merge without explicit user confirmation (project rule: "commit" ≠ "publish").

---

## Self-Review

**Spec coverage:**
- Config `strict_labels` default true + self-migration → Task 1 (self-migration is automatic via LoadConfig-over-DefaultConfig, same as prior keys). ✓
- `resolveExistingLabel` (case/whitespace, no fuzzy) → Task 2. ✓
- `InboxAnalyzerOptions.StrictLabels` + `enforceLabelPolicy` in `Analyze` + reclassify-to-read-manually → Task 3. ✓
- Wire config → analyzer at both call sites → Task 4. ✓
- Dispatch no-create-on-miss → Task 5. ✓
- Imperative prompt wording → Task 6. ✓
- Empty-labels guard (skip enforcement) → Task 3 (`strict && len(availableLabels) > 0`) + tested. ✓
- Docs/help → Task 7. ✓
- Scope: only the analyzer; manual `l` labeling untouched (resolveOrCreateLabelID is only called by applyActionPlanLabel) → confirmed in Reference facts. ✓

**Type/signature consistency:** `resolveExistingLabel(suggested string, existing []string) (string, bool)` (Task 2) is used identically in `enforceLabelPolicy` (Task 3). `enforceLabelPolicy(plan *ActionPlan, messages []AnalyzerMessage, availableLabels []string, strict bool)` defined and called consistently (Task 3). `InboxAnalyzerOptions.StrictLabels` defined (Task 3 Step 1), set in Task 4, read in Task 3 Step 5. `config.InboxAnalyzer.StrictLabels` defined Task 1, read Tasks 4 & 5. ✓

**Placeholder scan:** no TBD/TODO; every code step shows full before/after. Task 7 Steps 1–2 are genuine conditionals (depends on what the docs/help currently enumerate) with both branches specified. ✓
