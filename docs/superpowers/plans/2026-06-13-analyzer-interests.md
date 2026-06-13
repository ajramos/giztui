# Analyzer Interests Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the analyzer treat saved free-text rules as interest/relevance signals (surfacing matching emails as priority "high"), relabel the rules UI so interests are discoverable, and fix the `:plan rules` focus bug.

**Architecture:** A prompt reframe in `prependUserRules` (no schema/UI change). UI label tweaks in the rules manager. A synchronous focus fix using the existing `cmdFocusOverride` mechanism so the rules picker keeps focus after the command bar closes.

**Tech Stack:** Go, GizTUI analyzer prompt + TUI command/focus handling.

Spec: `docs/superpowers/specs/2026-06-13-analyzer-interests-design.md`

---

### Task 1: Prompt reframe — interests in `prependUserRules`

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go` (`prependUserRules`)
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/inbox_analyzer_service_test.go`:

```go
func TestPrependUserRules_Interests(t *testing.T) {
	got := prependUserRules("BASEPROMPT", []string{"interested in AI"})
	if !strings.Contains(got, "## User preferences and interests") {
		t.Fatalf("expected reframed heading, got:\n%s", got)
	}
	if !strings.Contains(got, "interest/relevance") {
		t.Fatalf("expected the relevance instruction, got:\n%s", got)
	}
	if !strings.Contains(got, "interested in AI") || !strings.Contains(got, "BASEPROMPT") {
		t.Fatalf("expected rule text + base prompt, got:\n%s", got)
	}
	// No rules → unchanged base prompt (no block).
	if prependUserRules("BASEPROMPT", nil) != "BASEPROMPT" {
		t.Fatal("empty rules must return the base prompt unchanged")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestPrependUserRules_Interests -v`
Expected: FAIL — heading is still "## User preferences (respect these rules)" / no "interest/relevance" string.

- [ ] **Step 3: Reframe `prependUserRules`**

In `internal/services/inbox_analyzer_service.go`, replace the final `return` of `prependUserRules`:

```go
	return "## User preferences (respect these rules)\n" + strings.Join(clean, "\n") + "\n\n" + promptText
```

with:

```go
	header := "## User preferences and interests\n" +
		"Treat the following as BOTH action rules to respect AND interest/relevance signals. " +
		"When an email matches a stated interest, do NOT bury it in a bulk archive/trash group: " +
		"keep it visible, set that category's priority to \"high\", and note the matched interest " +
		"in the category description.\n"
	return header + strings.Join(clean, "\n") + "\n\n" + promptText
```

- [ ] **Step 4: Run tests to verify they pass (and existing ones still pass)**

Run: `go test ./internal/services/ -run 'TestPrependUserRules|TestBuildPromptPreview|TestAnalyze' -v`
Expected: PASS. (Existing assertions use `Contains(..., "## User preferences")`, which still matches the new heading.)

- [ ] **Step 5: Commit**

```bash
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): treat saved rules as interest/relevance signals (prompt reframe)"
```

---

### Task 2: Fix `:plan rules` focus (cmdFocusOverride="keep")

**Files:**
- Modify: `internal/tui/keys.go` (`restoreFocusAfterModal`, ~line 1630)
- Modify: `internal/tui/action_plan_rules.go` (`openAnalyzerRulesManager`)
- Test: `internal/tui/action_plan_rules_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/tui/action_plan_rules_test.go`:

```go
func TestAnalyzerRulesPickerSetsFocusKeepOverride(t *testing.T) {
	a := newRulesTestApp()
	a.openAnalyzerRulesManager()
	if a.cmdFocusOverride != "keep" {
		t.Fatalf("expected cmdFocusOverride=keep so the command bar teardown won't steal focus, got %q", a.cmdFocusOverride)
	}
}

func TestRestoreFocusAfterModal_Keep(t *testing.T) {
	a := newRulesTestApp()
	a.currentFocus = "analyzer_rules"
	a.cmdFocusOverride = "keep"
	a.restoreFocusAfterModal()
	if a.currentFocus != "analyzer_rules" {
		t.Fatalf("keep override must not re-focus the list, got %q", a.currentFocus)
	}
	if a.cmdFocusOverride != "" {
		t.Fatalf("override should be consumed, got %q", a.cmdFocusOverride)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run 'TestAnalyzerRulesPickerSetsFocusKeepOverride|TestRestoreFocusAfterModal_Keep' -v`
Expected: FAIL — `openAnalyzerRulesManager` doesn't set the override yet; `restoreFocusAfterModal` has no "keep" case so it sets `currentFocus="list"`.

- [ ] **Step 3: Add the "keep" case in `restoreFocusAfterModal`**

In `internal/tui/keys.go`, inside `restoreFocusAfterModal`, the existing `switch override { ... }` (within the `if a.cmdFocusOverride != ""` block, which already clears the override) — add a case:

```go
		case "keep":
			// A picker/command already set focus; leave it as-is.
			return
```

- [ ] **Step 4: Set the override in `openAnalyzerRulesManager`**

In `internal/tui/action_plan_rules.go`, at the end of `openAnalyzerRulesManager`, right after `a.SetFocus(list)` / `a.currentFocus = "analyzer_rules"`, add:

```go
	// :plan rules runs during command execution; hideCommandBar()'s restoreFocusAfterModal()
	// would otherwise re-focus the message list afterward. "keep" tells it to leave our focus.
	a.cmdFocusOverride = "keep"
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestAnalyzerRulesPicker|TestRestoreFocusAfterModal_Keep' -v`
Expected: PASS (including the existing `TestAnalyzerRulesPickerSwap`).

- [ ] **Step 6: Commit**

```bash
git add internal/tui/keys.go internal/tui/action_plan_rules.go internal/tui/action_plan_rules_test.go
git commit -m "fix(tui): :plan rules keeps focus on the picker (cmdFocusOverride=keep)"
```

---

### Task 3: Relabel the rules UI for interests

**Files:**
- Modify: `internal/tui/action_plan_rules.go` (3 label strings)

No unit test (static label strings; build covers it).

- [ ] **Step 1: Update the three labels**

In `internal/tui/action_plan_rules.go`:

Manager title:
```go
	container.SetTitle(" 🧠 Analyzer rules ")
```
→
```go
	container.SetTitle(" 🧠 Analyzer rules & interests ")
```

Add-rule input label:
```go
		input := tview.NewInputField().SetLabel(" New rule: ").SetFieldWidth(0)
```
→
```go
		input := tview.NewInputField().SetLabel(" Rule or interest: ").SetFieldWidth(0)
```

Ctrl+R remember panel title:
```go
	state.container.SetTitle(" 🧠 Remember preference ")
```
→
```go
	state.container.SetTitle(" 🧠 Remember rule or interest ")
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/action_plan_rules.go
git commit -m "feat(tui): relabel rules UI to surface interests (discoverability)"
```

---

### Task 4: Docs — `:help` + CONFIGURATION.md (Definition of Done)

**Files:**
- Modify: `internal/tui/app.go` (`generateHelpText` — the `:action-plan` help line)
- Modify: `docs/CONFIGURATION.md`

- [ ] **Step 1: Update the in-app help**

In `internal/tui/app.go` `generateHelpText`, find the action-plan command line:
```go
	fmt.Fprintf(&help, "    %-18s 🧠  Open inbox Action Plan (alias :plan, :ap)\n", ":action-plan")
```
and add a line after it:
```go
	fmt.Fprintf(&help, "    %-18s 🧠  Manage analyzer rules/interests (e.g. 'interested in AI')\n", ":plan rules")
```

- [ ] **Step 2: Document in CONFIGURATION.md**

In `docs/CONFIGURATION.md`, in the analyzer section (near the `inbox_analyzer` table / the `:plan rules` mention), add:

```markdown
**Rules and interests:** with `:plan rules` (or `Ctrl+R` inside the Action Plan) you can save
free-text **action rules** ("archive everything from GitHub") *and* **interests** ("I'm interested
in AI"). The analyzer treats interests as relevance signals: emails matching them are surfaced
(priority "high" + a note in the category description) instead of being buried in a bulk action.
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go docs/CONFIGURATION.md
git commit -m "docs: document analyzer rules/interests (help + config)"
```

---

### Task 5: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass.

- [ ] **Step 2: Targeted + full suite**

Run: `go test ./internal/services/ ./internal/tui/ 2>&1 | tail -4`
Expected: all `ok`.
Run: `go test ./test/helpers/ 2>&1 | tail -2`
Expected: `ok` (no leaked-goroutine regression — Task 2's fix is synchronous, no QueueUpdateDraw).

- [ ] **Step 3: Build**

Run: `make build`
Expected: success.

(Live E2E — `:plan rules` now grabs focus; add an interest like "interested in AI"; re-run the Action Plan and check matching emails get HIGH priority + a note — deferred to the user's E2E sweep.)

---

## Self-review notes

- **Spec coverage:** prompt reframe (Task 1), focus fix via cmdFocusOverride="keep" (Task 2), UI
  relabel (Task 3), docs/`:help` (Task 4), verification incl. leak check (Task 5). All spec sections
  mapped. Existing `prependUserRules`/`BuildPromptPreview` tests stay green (Contains-prefix match).
- **Type consistency:** `cmdFocusOverride` string with value `"keep"` matches between
  `openAnalyzerRulesManager` (set) and `restoreFocusAfterModal` (case). `prependUserRules` signature
  unchanged.
- **No placeholders:** every code step shows full code; commands have expected output.
- **Leak-safety:** Task 2's fix is synchronous (no `QueueUpdateDraw`), so it cannot reintroduce the
  goroutine-leak class that the config-notice fix addressed.
