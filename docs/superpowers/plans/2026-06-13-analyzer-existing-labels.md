# Analyzer Existing-Labels Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Feed the analyzer the user's existing (user-created) labels so it prefers an exact match for the `label` action instead of inventing labels.

**Architecture:** A new `AvailableLabels` option + a `prependAvailableLabels` prompt block (mirroring `prependUserRules`), applied in `Analyze` and `BuildPromptPreview`. The TUI populates it from `ListLabels` filtered to user labels. Dispatch unchanged.

**Tech Stack:** Go, GizTUI analyzer prompt + Label service + Action Plan TUI.

Spec: `docs/superpowers/specs/2026-06-13-analyzer-existing-labels-design.md`

---

### Task 1: `AvailableLabels` option + `prependAvailableLabels` + prompt wiring

**Files:**
- Modify: `internal/services/interfaces.go` (`InboxAnalyzerOptions`)
- Modify: `internal/services/inbox_analyzer_service.go` (`prependAvailableLabels`, `Analyze`, `BuildPromptPreview`)
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/services/inbox_analyzer_service_test.go`:

```go
func TestPrependAvailableLabels(t *testing.T) {
	got := prependAvailableLabels("BASEPROMPT", []string{"work", "receipts"})
	if !strings.Contains(got, "## Existing labels") {
		t.Fatalf("expected labels heading, got:\n%s", got)
	}
	if !strings.Contains(got, "work") || !strings.Contains(got, "receipts") {
		t.Fatalf("expected label names, got:\n%s", got)
	}
	if !strings.Contains(got, "PREFER an exact name") {
		t.Fatalf("expected the prefer-existing instruction, got:\n%s", got)
	}
	if !strings.Contains(got, "BASEPROMPT") {
		t.Fatalf("expected base prompt retained, got:\n%s", got)
	}
	if prependAvailableLabels("BASEPROMPT", nil) != "BASEPROMPT" {
		t.Fatal("empty labels must return the base prompt unchanged")
	}
}

func TestBuildPromptPreview_Labels(t *testing.T) {
	s := NewInboxAnalyzerService(nil)
	out := s.BuildPromptPreview(InboxAnalyzerOptions{AvailableLabels: []string{"work"}})
	if !strings.Contains(out, "## Existing labels") || !strings.Contains(out, "work") {
		t.Fatalf("preview should include the labels block, got:\n%s", out)
	}
	out = s.BuildPromptPreview(InboxAnalyzerOptions{})
	if strings.Contains(out, "## Existing labels") {
		t.Fatalf("no labels → no labels block, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run 'TestPrependAvailableLabels|TestBuildPromptPreview_Labels' -v`
Expected: FAIL — `prependAvailableLabels` undefined / `AvailableLabels` field missing.

- [ ] **Step 3: Add the option field**

In `internal/services/interfaces.go`, add to `InboxAnalyzerOptions`:

```go
	AvailableLabels  []string // existing user-label names; analyzer prefers these for the "label" action
```

- [ ] **Step 4: Add `prependAvailableLabels`**

In `internal/services/inbox_analyzer_service.go`, add (near `prependUserRules`):

```go
// prependAvailableLabels prepends an "## Existing labels" block when labels are present, telling
// the model to prefer an exact existing label for the "label" action. Empty → prompt unchanged.
func prependAvailableLabels(promptText string, labels []string) string {
	clean := make([]string, 0, len(labels))
	for _, l := range labels {
		if strings.TrimSpace(l) != "" {
			clean = append(clean, strings.TrimSpace(l))
		}
	}
	if len(clean) == 0 {
		return promptText
	}
	header := "## Existing labels\n" +
		"These labels already exist in the mailbox: " + strings.Join(clean, ", ") + "\n" +
		"For the \"label\" action, PREFER an exact name from this list when one fits. Only invent a " +
		"NEW label when none fits; if you do, keep it short kebab-case and note \"(new)\" in the " +
		"category description.\n\n"
	return header + promptText
}
```

- [ ] **Step 5: Apply it in `Analyze` and `BuildPromptPreview`**

In `internal/services/inbox_analyzer_service.go`, in `Analyze`, after the existing
`promptText = prependUserRules(promptText, opts.UserRules)` line, add:

```go
	promptText = prependAvailableLabels(promptText, opts.AvailableLabels)
```

In `BuildPromptPreview`, change:
```go
	return prependUserRules(base, opts.UserRules)
```
to:
```go
	withRules := prependUserRules(base, opts.UserRules)
	return prependAvailableLabels(withRules, opts.AvailableLabels)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/services/ -run 'TestPrependAvailableLabels|TestBuildPromptPreview|TestPrependUserRules' -v`
Expected: PASS (existing prompt tests unaffected).

- [ ] **Step 7: Commit**

```bash
git add internal/services/interfaces.go internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): prefer existing user labels via prompt (AvailableLabels)"
```

---

### Task 2: TUI populates AvailableLabels (user labels only)

**Files:**
- Modify: `internal/tui/action_plan.go` (new `userLabelNames` helper + analyze-flow opts)
- Modify: `internal/tui/action_plan_prompt.go` (viewer opts)

No unit test for the wiring (covered by service tests + build); the filter helper is simple.

- [ ] **Step 1: Add the `userLabelNames` helper**

In `internal/tui/action_plan.go`, add:

```go
// userLabelNames returns the names of the user's own labels (excluding Gmail system labels),
// for feeding the analyzer so it prefers existing labels. Errors degrade to an empty slice.
func (a *App) userLabelNames() []string {
	_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
	if labelService == nil {
		return nil
	}
	labels, err := labelService.ListLabels(a.ctx)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if l != nil && l.Type == "user" {
			out = append(out, l.Name)
		}
	}
	return out
}
```

- [ ] **Step 2: Populate it in the analyze flow**

In `internal/tui/action_plan.go`, where the analyze goroutine builds the options (the line passing
`services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules, BodyCharLimit: bodyCharLimit}`), capture the labels before
the goroutine (alongside `userRules`):

```go
	availableLabels := a.userLabelNames()
```

and add `AvailableLabels: availableLabels` to that `InboxAnalyzerOptions` literal:

```go
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules, BodyCharLimit: bodyCharLimit, AvailableLabels: availableLabels},
```

(Compute `availableLabels` next to where `userRules` is gathered — outside the goroutine — so the
ListLabels read isn't racing the analysis.)

- [ ] **Step 3: Populate it in the prompt viewer**

In `internal/tui/action_plan_prompt.go`, in the `opts := services.InboxAnalyzerOptions{...}` literal,
add `AvailableLabels: a.userLabelNames()` so the `v` viewer matches what Analyze sends.

- [ ] **Step 4: Build + tests**

Run: `go build ./... && go test ./internal/services/ ./internal/tui/ 2>&1 | tail -3`
Expected: build success; all `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/action_plan_prompt.go
git commit -m "feat(action-plan): feed existing user labels to the analyzer + prompt viewer"
```

---

### Task 3: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass.

- [ ] **Step 2: Targeted + leak check**

Run: `go test ./internal/services/ ./internal/tui/ ./test/helpers/ 2>&1 | tail -5`
Expected: all `ok`.

- [ ] **Step 3: Build**

Run: `make build`
Expected: success.

(Live E2E — open the Action Plan, press `v`, confirm an "## Existing labels" block lists your real
labels; run analysis and check `label` categories reuse existing names — deferred to user E2E.)

---

## Self-review notes

- **Spec coverage:** `AvailableLabels` option (Task 1), `prependAvailableLabels` + Analyze +
  BuildPromptPreview (Task 1), TUI `userLabelNames` filtered to `Type=="user"` + analyze-flow +
  viewer population (Task 2), verification (Task 3). All spec sections mapped. Dispatch unchanged
  (per decision) — no task.
- **Type consistency:** `AvailableLabels []string` on `InboxAnalyzerOptions`;
  `prependAvailableLabels(promptText string, labels []string) string`; `userLabelNames() []string`
  — match across tasks and call sites. `GetServices()` 12-tuple: `labelService` is the 3rd return.
- **No placeholders:** every code step shows full code; commands have expected output.
