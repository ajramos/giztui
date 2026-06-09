# Inbox Action Plan Rework — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rework the Inbox Action Plan into a selection-scoped, expandable, per-email-controllable panel with a lightweight LLM-interpreted learning layer.

**Architecture:** Service-first per CLAUDE.md. New `analyzer_rules` SQLite table (migration v8) → `AnalyzerRulesStore` (db) → `AnalyzerRulesService` (business logic, account-scoped, pure rule-suggestion helper) → accessed via a dedicated `GetAnalyzerRulesService()` getter (mirroring `GetInboxAnalyzerService()`, NOT the `GetServices()` tuple). The analyzer gains a `UserRules []string` option whose texts are prepended to the prompt. The TUI replaces the scroll-only `TextView` with a `tview.TreeView` (categories → emails), which gives native focus/selection (fixing the focus bug) and a home for per-email exclusion + `Ctrl+R` rule capture.

**Tech Stack:** Go, `github.com/derailed/tview`, `github.com/derailed/tcell/v2`, SQLite (`internal/db`), existing `InboxAnalyzerService` + `EmailService`/`LabelService`.

**Reference patterns (read before starting):**
- Store: `internal/db/query_store.go`
- Migration: `internal/db/store.go:264-366` (v6/v7 blocks; add v8)
- Service: `internal/services/query_service.go:1-70`
- Service wiring: `internal/tui/app.go:557-575` (reinitializeServices), struct field near `app.go:213`
- Analyzer: `internal/services/inbox_analyzer_service.go:228-289`
- Modal overlay: `internal/tui/prompt_preview.go`
- Panel under rework: `internal/tui/action_plan.go` (entire file)
- Command dispatch: `internal/tui/commands.go:2593-2609`

---

## File Structure

**Create:**
- `internal/db/analyzer_rules_store.go` — CRUD for the `analyzer_rules` table (account-scoped).
- `internal/db/analyzer_rules_store_test.go` — store tests.
- `internal/services/analyzer_rules_service.go` — `AnalyzerRulesService` impl + pure `SuggestRuleFromContext`.
- `internal/services/analyzer_rules_service_test.go` — service tests.
- `internal/tui/action_plan_rules.go` — `Ctrl+R` capture modal + `:action-plan rules` manager.

**Modify:**
- `internal/db/store.go` — add migration v8.
- `internal/services/interfaces.go` — `AnalyzerRule` type, `AnalyzerRulesService` interface, `UserRules` field on `InboxAnalyzerOptions`.
- `internal/services/inbox_analyzer_service.go` — prepend user-rules block.
- `internal/services/inbox_analyzer_service_test.go` — rules-injection test.
- `internal/tui/app.go` — service field, wiring, `GetAnalyzerRulesService()` getter.
- `internal/tui/action_plan.go` — selection scope, TreeView, exclusion, footer, action-on-checked, load rules.
- `internal/tui/action_plan_test.go` — pure-helper tests (footer, checked-count, scope select).
- `internal/tui/commands.go` — `:action-plan rules` subcommand.
- `docs/KEYBOARD_SHORTCUTS.md` + `docs/ARCHITECTURE.md` (picker note) — document new gestures.

---

## Task 1: Migration v8 — `analyzer_rules` table

**Files:**
- Modify: `internal/db/store.go:363-365` (insert a v8 block before `return nil`)

- [ ] **Step 1: Add the v8 migration block**

In `internal/db/store.go`, immediately after the `ver = 7` block closes (the line `ver = 7` then `}`), before `return nil`:

```go
	// v8: analyzer preference rules (free-text, LLM-interpreted)
	if ver == 7 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS analyzer_rules (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  account_email TEXT NOT NULL,
  rule_text     TEXT NOT NULL,
  created_at    INTEGER NOT NULL
);`)

		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=8;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v8: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 8
	}
```

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./internal/db/...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/db/store.go
git commit -m "feat(db): add analyzer_rules table (migration v8)"
```

---

## Task 2: `AnalyzerRulesStore`

**Files:**
- Create: `internal/db/analyzer_rules_store.go`
- Test: `internal/db/analyzer_rules_store_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/db/analyzer_rules_store_test.go`. Mirror the test bootstrap used in `internal/db/store_test.go` (open a temp DB via `Open`). If `store_test.go` exposes a helper like `newTestStore(t)`, reuse it; otherwise open with `Open(ctx, filepath.Join(t.TempDir(), "test.db"))`.

```go
package db

import (
	"context"
	"testing"
)

func TestAnalyzerRulesStore_SaveListDelete(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/rules.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	rs := NewAnalyzerRulesStore(store)
	const acct = "user@example.com"

	if _, err := rs.SaveRule(ctx, acct, "Never trash emails from tldr.tech"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := rs.SaveRule(ctx, acct, "Archive newsletters automatically"); err != nil {
		t.Fatalf("save 2: %v", err)
	}

	rules, err := rs.ListRules(ctx, acct)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(rules))
	}

	// Account scoping: another account sees nothing.
	other, err := rs.ListRules(ctx, "someone@else.com")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("want 0 rules for other account, got %d", len(other))
	}

	if err := rs.DeleteRule(ctx, acct, rules[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rules, _ = rs.ListRules(ctx, acct)
	if len(rules) != 1 {
		t.Fatalf("want 1 rule after delete, got %d", len(rules))
	}
}

func TestAnalyzerRulesStore_RejectsEmpty(t *testing.T) {
	ctx := context.Background()
	store, _ := Open(ctx, t.TempDir()+"/rules.db")
	defer func() { _ = store.Close() }()
	rs := NewAnalyzerRulesStore(store)

	if _, err := rs.SaveRule(ctx, "", "x"); err == nil {
		t.Fatal("expected error for empty account")
	}
	if _, err := rs.SaveRule(ctx, "a@b.c", "   "); err == nil {
		t.Fatal("expected error for blank rule text")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/db/ -run TestAnalyzerRulesStore -v`
Expected: FAIL — `undefined: NewAnalyzerRulesStore`.

- [ ] **Step 3: Implement the store**

Create `internal/db/analyzer_rules_store.go`:

```go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AnalyzerRule is one free-text, LLM-interpreted preference rule for the inbox analyzer.
type AnalyzerRule struct {
	ID           int64  `json:"id"`
	AccountEmail string `json:"account_email"`
	RuleText     string `json:"rule_text"`
	CreatedAt    int64  `json:"created_at"`
}

// AnalyzerRulesStore handles persistence of analyzer preference rules.
type AnalyzerRulesStore struct {
	db *sql.DB
}

// NewAnalyzerRulesStore creates a new analyzer rules store.
func NewAnalyzerRulesStore(store *Store) *AnalyzerRulesStore {
	return &AnalyzerRulesStore{db: store.DB()}
}

// SaveRule inserts a new rule for the account and returns it.
func (s *AnalyzerRulesStore) SaveRule(ctx context.Context, accountEmail, ruleText string) (*AnalyzerRule, error) {
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(ruleText) == "" {
		return nil, fmt.Errorf("account_email and rule_text cannot be empty")
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO analyzer_rules (account_email, rule_text, created_at)
		VALUES (?, ?, ?)`,
		accountEmail, strings.TrimSpace(ruleText), now)
	if err != nil {
		return nil, fmt.Errorf("failed to save rule: %w", err)
	}
	id, _ := res.LastInsertId()
	return &AnalyzerRule{ID: id, AccountEmail: accountEmail, RuleText: strings.TrimSpace(ruleText), CreatedAt: now}, nil
}

// ListRules returns all rules for an account, newest first.
func (s *AnalyzerRulesStore) ListRules(ctx context.Context, accountEmail string) ([]*AnalyzerRule, error) {
	if strings.TrimSpace(accountEmail) == "" {
		return nil, fmt.Errorf("account_email cannot be empty")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_email, rule_text, created_at
		FROM analyzer_rules
		WHERE account_email = ?
		ORDER BY created_at DESC, id DESC`, accountEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*AnalyzerRule
	for rows.Next() {
		r := &AnalyzerRule{}
		if err := rows.Scan(&r.ID, &r.AccountEmail, &r.RuleText, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return out, nil
}

// DeleteRule removes a rule by id for the account.
func (s *AnalyzerRulesStore) DeleteRule(ctx context.Context, accountEmail string, id int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM analyzer_rules WHERE account_email = ? AND id = ?`, accountEmail, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/db/ -run TestAnalyzerRulesStore -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/db/analyzer_rules_store.go internal/db/analyzer_rules_store_test.go
git commit -m "feat(db): AnalyzerRulesStore CRUD with account scoping"
```

---

## Task 2 note: confirm `store_test.go` bootstrap

If `Open` has a different signature than `Open(ctx, path) (*Store, error)`, adapt Steps 1/3 of Task 2 to match `internal/db/store_test.go`. Check first:

Run: `grep -n "func Open\|func newTestStore\|Open(" internal/db/store.go internal/db/store_test.go | head`

Use whatever bootstrap the existing tests use. Do not invent a new one.

---

## Task 3: `AnalyzerRulesService` (interface + impl + pure suggestion helper)

**Files:**
- Modify: `internal/services/interfaces.go` (add type + interface near the `InboxAnalyzerService` block, ~line 1008)
- Create: `internal/services/analyzer_rules_service.go`
- Test: `internal/services/analyzer_rules_service_test.go`

- [ ] **Step 1: Add interface + type to `interfaces.go`**

After the `InboxAnalyzerService` interface (after line 1008), add:

```go
// AnalyzerRuleInfo is a free-text analyzer preference rule, surfaced to the TUI.
type AnalyzerRuleInfo struct {
	ID        int64
	RuleText  string
	CreatedAt int64
}

// AnalyzerRulesService persists and supplies the user's free-text analyzer
// preference rules. Rules are natural-language strings injected into the analyzer
// prompt (the LLM interprets them); no deterministic matching is done here.
type AnalyzerRulesService interface {
	SaveRule(ctx context.Context, ruleText string) error
	ListRules(ctx context.Context) ([]AnalyzerRuleInfo, error)
	DeleteRule(ctx context.Context, id int64) error
	// SuggestRuleFromContext builds an editable default rule string from a message's
	// From header and an action token. negate=true phrases it as a prohibition
	// (e.g. "Never trash emails from tldr.tech"); negate=false as a directive
	// (e.g. "Always archive emails from tldr.tech"). Pure — no I/O.
	SuggestRuleFromContext(from, action string, negate bool) string
}
```

- [ ] **Step 2: Write the failing test**

Create `internal/services/analyzer_rules_service_test.go`:

```go
package services

import "testing"

func TestSuggestRuleFromContext(t *testing.T) {
	s := &AnalyzerRulesServiceImpl{}
	cases := []struct {
		name   string
		from   string
		action string
		negate bool
		want   string
	}{
		{"trash negate domain", `"TLDR" <news@tldr.tech>`, "trash", true, "Never trash emails from tldr.tech"},
		{"archive directive domain", "news@tldr.tech", "archive", false, "Always archive emails from tldr.tech"},
		{"bare email no angle", "boss@team.io", "label", false, "Always label emails from team.io"},
		{"no domain falls back to whole from", "weird-sender", "trash", true, "Never trash emails from weird-sender"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := s.SuggestRuleFromContext(c.from, c.action, c.negate)
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestSuggestRuleFromContext -v`
Expected: FAIL — `undefined: AnalyzerRulesServiceImpl`.

- [ ] **Step 4: Implement the service**

Create `internal/services/analyzer_rules_service.go`:

```go
package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/db"
)

// AnalyzerRulesServiceImpl implements AnalyzerRulesService.
type AnalyzerRulesServiceImpl struct {
	store        *db.AnalyzerRulesStore
	accountEmail string
	mu           sync.RWMutex
}

// NewAnalyzerRulesService creates a new analyzer rules service.
func NewAnalyzerRulesService(store *db.AnalyzerRulesStore) *AnalyzerRulesServiceImpl {
	return &AnalyzerRulesServiceImpl{store: store}
}

// SetAccountEmail sets the active account for scoping.
func (s *AnalyzerRulesServiceImpl) SetAccountEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountEmail = email
}

func (s *AnalyzerRulesServiceImpl) account() (string, error) {
	s.mu.RLock()
	email := s.accountEmail
	s.mu.RUnlock()
	if strings.TrimSpace(email) == "" {
		return "", fmt.Errorf("account email not set")
	}
	return email, nil
}

func (s *AnalyzerRulesServiceImpl) SaveRule(ctx context.Context, ruleText string) error {
	if s.store == nil {
		return fmt.Errorf("analyzer rules store not available")
	}
	if strings.TrimSpace(ruleText) == "" {
		return fmt.Errorf("rule text cannot be empty")
	}
	email, err := s.account()
	if err != nil {
		return err
	}
	_, err = s.store.SaveRule(ctx, email, ruleText)
	return err
}

func (s *AnalyzerRulesServiceImpl) ListRules(ctx context.Context) ([]AnalyzerRuleInfo, error) {
	if s.store == nil {
		return nil, fmt.Errorf("analyzer rules store not available")
	}
	email, err := s.account()
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListRules(ctx, email)
	if err != nil {
		return nil, err
	}
	out := make([]AnalyzerRuleInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, AnalyzerRuleInfo{ID: r.ID, RuleText: r.RuleText, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func (s *AnalyzerRulesServiceImpl) DeleteRule(ctx context.Context, id int64) error {
	if s.store == nil {
		return fmt.Errorf("analyzer rules store not available")
	}
	email, err := s.account()
	if err != nil {
		return err
	}
	return s.store.DeleteRule(ctx, email, id)
}

// actionRuleVerb maps an action token to the verb used in a suggested rule.
func actionRuleVerb(action string) string {
	switch action {
	case "archive":
		return "archive"
	case "mark_read":
		return "mark as read"
	case "trash":
		return "trash"
	case "label":
		return "label"
	default:
		return "review"
	}
}

// senderDomain extracts the domain from a From header, falling back to the whole
// trimmed string if there is no parseable address.
func senderDomain(from string) string {
	f := strings.TrimSpace(from)
	// Prefer the bracketed address if present: "Name" <user@host>
	if i := strings.LastIndex(f, "<"); i >= 0 {
		if j := strings.Index(f[i:], ">"); j >= 0 {
			f = strings.TrimSpace(f[i+1 : i+j])
		}
	}
	if at := strings.LastIndex(f, "@"); at >= 0 && at+1 < len(f) {
		return strings.ToLower(strings.TrimSpace(f[at+1:]))
	}
	return f
}

func (s *AnalyzerRulesServiceImpl) SuggestRuleFromContext(from, action string, negate bool) string {
	target := senderDomain(from)
	verb := actionRuleVerb(action)
	if negate {
		return fmt.Sprintf("Never %s emails from %s", verb, target)
	}
	return fmt.Sprintf("Always %s emails from %s", verb, target)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestSuggestRuleFromContext -v`
Expected: PASS (all 4 sub-cases).

- [ ] **Step 6: Commit**

```bash
git add internal/services/interfaces.go internal/services/analyzer_rules_service.go internal/services/analyzer_rules_service_test.go
git commit -m "feat(services): AnalyzerRulesService with pure rule-suggestion helper"
```

---

## Task 4: Analyzer prompt injection (`UserRules`)

**Files:**
- Modify: `internal/services/interfaces.go:994-999` (`InboxAnalyzerOptions`)
- Modify: `internal/services/inbox_analyzer_service.go:235-246` (`Analyze`)
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Add the field to `InboxAnalyzerOptions`**

In `interfaces.go`, inside `InboxAnalyzerOptions`:

```go
	CustomPromptText string   // empty → use the built-in default analyzer prompt
	UserRules        []string // free-text preference rules prepended to the prompt; empty → none
```

- [ ] **Step 2: Add a pure helper + write its failing test**

Add to `inbox_analyzer_service_test.go`:

```go
func TestPrependUserRules(t *testing.T) {
	base := "Categorize these messages."
	got := prependUserRules(base, []string{
		"Never trash emails from tldr.tech",
		"Archive newsletters automatically",
	})
	if !strings.Contains(got, "## User preferences") {
		t.Fatalf("missing header in: %q", got)
	}
	if !strings.Contains(got, "- Never trash emails from tldr.tech") {
		t.Fatalf("missing rule 1 in: %q", got)
	}
	if !strings.Contains(got, "- Archive newsletters automatically") {
		t.Fatalf("missing rule 2 in: %q", got)
	}
	if !strings.HasSuffix(got, base) {
		t.Fatalf("base prompt must follow the rules block, got: %q", got)
	}

	// Empty rules → unchanged.
	if prependUserRules(base, nil) != base {
		t.Fatal("nil rules must return the base prompt unchanged")
	}
}
```

Ensure the test file imports `"strings"` (add if missing).

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestPrependUserRules -v`
Expected: FAIL — `undefined: prependUserRules`.

- [ ] **Step 4: Implement the helper and call it in `Analyze`**

In `inbox_analyzer_service.go`, add near `buildBatchPrompt`:

```go
// prependUserRules adds a "## User preferences" block before the analyzer prompt
// so the LLM honors the user's free-text rules. Empty rules → prompt unchanged.
func prependUserRules(promptText string, rules []string) string {
	clean := make([]string, 0, len(rules))
	for _, r := range rules {
		if strings.TrimSpace(r) != "" {
			clean = append(clean, "- "+strings.TrimSpace(r))
		}
	}
	if len(clean) == 0 {
		return promptText
	}
	return "## User preferences (respect these rules)\n" + strings.Join(clean, "\n") + "\n\n" + promptText
}
```

Then in `Analyze`, after the `promptText` default-resolution block (after line 246):

```go
	promptText = prependUserRules(promptText, opts.UserRules)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/services/ -run "TestPrependUserRules|TestSuggestRuleFromContext" -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/services/interfaces.go internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(services): inject user preference rules into analyzer prompt"
```

---

## Task 5: Wire `AnalyzerRulesService` into `App`

**Files:**
- Modify: `internal/tui/app.go` (struct field ~213; reinitializeServices ~557-575; add getter near `GetInboxAnalyzerService`)

- [ ] **Step 1: Add the struct field**

Near `app.go:213` (next to `queryService`):

```go
	analyzerRulesService    services.AnalyzerRulesService
```

- [ ] **Step 2: Wire it in `reinitializeServices`**

In `app.go`, in the same area as the query service wiring (~557), add:

```go
	if a.dbStore != nil && a.analyzerRulesService == nil {
		rulesStore := db.NewAnalyzerRulesStore(a.dbStore)
		svc := services.NewAnalyzerRulesService(rulesStore)
		if email := a.getActiveAccountEmail(); email != "" {
			svc.SetAccountEmail(email)
		}
		a.analyzerRulesService = svc
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: analyzer rules service initialized: %v", a.analyzerRulesService != nil)
		}
	}
```

If the account email can change after init, also call `SetAccountEmail` wherever the query service's account email is refreshed (search: `grep -n "SetAccountEmail" internal/tui/app.go` and add an equivalent line for `a.analyzerRulesService` guarded by a type assertion to `*services.AnalyzerRulesServiceImpl`).

- [ ] **Step 3: Add the getter**

Find `GetInboxAnalyzerService` (`grep -n "func (a \*App) GetInboxAnalyzerService" internal/tui/app.go`) and add beside it:

```go
// GetAnalyzerRulesService returns the analyzer rules service (may be nil if no DB/account).
func (a *App) GetAnalyzerRulesService() services.AnalyzerRulesService {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.analyzerRulesService
}
```

(If `GetInboxAnalyzerService` does NOT take the lock, match its exact style — do not add locking it doesn't use.)

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): wire AnalyzerRulesService into App with getter"
```

---

## Task 6: Selection-first scope

**Files:**
- Modify: `internal/tui/action_plan.go:19-46` (add selection builder), `:147-165` (scope choice), header text
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing test for the selection builder**

Add to `internal/tui/action_plan_test.go` (create the file if absent, `package tui`):

```go
package tui

import (
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func msgWith(id, from, subj string, unread bool) *gmailapi.Message {
	m := &gmailapi.Message{Id: id, Snippet: "snip", Payload: &gmailapi.MessagePart{
		Headers: []*gmailapi.MessagePartHeader{
			{Name: "From", Value: from}, {Name: "Subject", Value: subj},
		},
	}}
	if unread {
		m.LabelIds = []string{"UNREAD"}
	}
	return m
}

func TestBuildAnalyzerMessagesForSelection(t *testing.T) {
	metas := []*gmailapi.Message{
		msgWith("1", "a@x.com", "S1", true),
		msgWith("2", "b@x.com", "S2", false), // read, but explicitly selected
		msgWith("3", "c@x.com", "S3", true),
	}
	selected := map[string]bool{"2": true, "3": true}

	got := buildAnalyzerMessagesForSelection(metas, selected)
	if len(got) != 2 {
		t.Fatalf("want 2 selected (incl. read), got %d", len(got))
	}
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids["2"] || !ids["3"] {
		t.Fatalf("expected ids 2 and 3, got %+v", ids)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestBuildAnalyzerMessagesForSelection -v`
Expected: FAIL — `undefined: buildAnalyzerMessagesForSelection`.

- [ ] **Step 3: Implement the selection builder**

In `action_plan.go`, after `buildAnalyzerMessages`:

```go
// buildAnalyzerMessagesForSelection converts the explicitly-selected messages into
// AnalyzerMessages. Unlike buildAnalyzerMessages it does NOT filter by UNREAD — an
// explicit selection counts regardless of read state.
func buildAnalyzerMessagesForSelection(metas []*gmailapi.Message, selected map[string]bool) []services.AnalyzerMessage {
	out := make([]services.AnalyzerMessage, 0, len(selected))
	for _, m := range metas {
		if m == nil || !selected[m.Id] {
			continue
		}
		out = append(out, services.AnalyzerMessage{
			ID:      m.Id,
			Subject: extractHeaderValue(m, "Subject"),
			From:    extractHeaderValue(m, "From"),
			Snippet: m.Snippet,
		})
	}
	return out
}
```

- [ ] **Step 4: Use the selection in `openActionPlanWithText` + set header mode**

Replace the message-collection block (`action_plan.go:156-165`) with:

```go
	// Scope: selection-first (analyze the user's bulk selection if any), else fall
	// back to the unread inbox already in memory.
	a.mu.RLock()
	metas := make([]*gmailapi.Message, len(a.messagesMeta))
	copy(metas, a.messagesMeta)
	selected := make(map[string]bool, len(a.selected))
	for id, ok := range a.selected {
		if ok {
			selected[id] = true
		}
	}
	a.mu.RUnlock()

	var messages []services.AnalyzerMessage
	scopeLabel := ""
	if len(selected) > 0 {
		messages = buildAnalyzerMessagesForSelection(metas, selected)
		scopeLabel = fmt.Sprintf("%d selected", len(messages))
	} else {
		messages = buildAnalyzerMessages(metas)
		scopeLabel = fmt.Sprintf("%d unread (inbox)", len(messages))
	}
	if len(messages) == 0 {
		a.GetErrorHandler().ShowInfo(a.ctx, "No messages to analyze. Select messages (v/space) or try :search is:unread.")
		return
	}
```

Store `scopeLabel` on the state (add `scopeLabel string` to `actionPlanState`) and show it in the header. In `renderActionPlanPanel`, change the header `SetText` to include it:

```go
	state.header.SetText(fmt.Sprintf("[::b]Action Plan · %s • batch %d/%d • %s[::-]", state.scopeLabel, p.BatchesDone, p.BatchesTotal, status))
```

Set `state.scopeLabel = scopeLabel` where the state is constructed (line ~170).

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/tui/ -run TestBuildAnalyzerMessagesForSelection -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): action plan analyzes current selection, falls back to unread inbox"
```

---

## Task 7: Replace `TextView` body with `TreeView` (categories only)

**Files:**
- Modify: `internal/tui/action_plan.go` — `actionPlanState`, `openActionPlanWithText`, `renderActionPlanPanel`, remove `renderActionPlanText` usage for the body

This task switches the widget but keeps behavior (categories as root nodes, no expansion yet). Per-email children + exclusion come in Task 8.

- [ ] **Step 1: Change the state to hold a tree**

In `actionPlanState`, replace `body *tview.TextView` with:

```go
	tree     *tview.TreeView
	root     *tview.TreeNode
```

Add an import for nothing new (tview already imported).

- [ ] **Step 2: Build the tree in `openActionPlanWithText`**

Replace the body construction (`action_plan.go:177-180`) with:

```go
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root).SetCurrentNode(state.root)
	state.tree.SetTopLevel(1) // hide the empty root; categories are the visible top level
	state.tree.SetBackgroundColor(bg)
	state.tree.SetGraphics(true)
```

Update the container assembly (`action_plan.go:193-195`) to add `state.tree` instead of `state.body`:

```go
	state.container.AddItem(state.header, 1, 0, false)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
```

Update focus lines (`action_plan.go:209-210`) to focus the tree and attach the input capture to the tree:

```go
	state.tree.SetInputCapture(a.actionPlanInputCapture(state))
	a.SetFocus(state.tree)
```

- [ ] **Step 3: Implement a tree (re)builder that preserves selection**

Add a new function and call it from `renderActionPlanPanel` instead of `state.body.SetText(...)`:

```go
// rebuildActionPlanTree repopulates the tree from state.plan, preserving the
// selected category index and each category's expanded state. Categories are root
// nodes; per-email children are added lazily in Task 8.
func (a *App) rebuildActionPlanTree(state *actionPlanState) {
	if state.plan == nil || state.root == nil {
		return
	}
	colors := a.GetComponentColors("ai")
	state.root.ClearChildren()
	for i, c := range state.plan.Categories {
		label := fmt.Sprintf("%s · %d · %s · %s", actionVerbLabel(c.Action), len(c.MessageIDs), c.Name, strings.ToUpper(c.Priority))
		node := tview.NewTreeNode(label).SetSelectable(true).SetColor(colors.Text.Color())
		node.SetReference(i) // category index
		state.root.AddChild(node)
	}
	// Restore selection.
	children := state.root.GetChildren()
	if len(children) == 0 {
		state.tree.SetCurrentNode(state.root)
		return
	}
	if state.selectedCategory < 0 {
		state.selectedCategory = 0
	}
	if state.selectedCategory >= len(children) {
		state.selectedCategory = len(children) - 1
	}
	state.tree.SetCurrentNode(children[state.selectedCategory])
}
```

In `renderActionPlanPanel`, replace the body `SetText` line with `a.rebuildActionPlanTree(state)` (keep the header update).

- [ ] **Step 4: Update navigation + delete the manual highlight**

`moveActionPlanSelection` and the `tcell.KeyUp/KeyDown` cases in `actionPlanInputCapture` should no longer manually move selection — let the `TreeView` handle up/down natively by returning the event. Change the Up/Down cases to:

```go
		case tcell.KeyUp, tcell.KeyDown:
			return ev // let TreeView move the cursor natively
		}
```

Track the selected category from the tree instead. Add a `SetChangedFunc` after building the tree (in Step 2 area):

```go
	state.tree.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		if idx, ok := node.GetReference().(int); ok {
			state.selectedCategory = idx
		}
		a.updateActionPlanFooter(state) // implemented in Task 9
	})
```

For now, stub `updateActionPlanFooter` as a no-op (Task 9 implements it):

```go
func (a *App) updateActionPlanFooter(state *actionPlanState) {}
```

`currentActionPlanCategory` stays valid (reads `state.selectedCategory`). `renderActionPlanText` and the old `moveActionPlanSelection` body become dead — delete `renderActionPlanText` and `moveActionPlanSelection` and any now-unused helper (`actionKeyHint` is still used by Task 9 footer; keep it).

- [ ] **Step 5: Build + smoke test**

Run: `go build ./... && go test ./internal/tui/ -run TestBuildAnalyzerMessages -v`
Expected: clean build, existing tests pass.

- [ ] **Step 6: E2E smoke (manual, via tmux)**

Build and run; press `P` on a populated inbox; confirm categories render in a tree, ↑↓ moves the highlight natively, and the highlight persists after analysis completes (this is the thread-A fix). Esc closes.

Run: `make build` then drive with `/usr/bin/tmux` (see Notes). Confirm no focus loss after streaming.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/action_plan.go
git commit -m "feat(tui): action plan uses TreeView (fixes post-analysis focus/nav)"
```

---

## Task 8: Expandable emails + per-email exclusion

**Files:**
- Modify: `internal/tui/action_plan.go` — tree children, exclusion state, `space` toggle
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Add exclusion state + a pure checked-count helper, write its test**

Add to `actionPlanState`:

```go
	excluded map[string]bool // message IDs the user toggled OFF (skip on action)
```

Initialize it where the state is created: `excluded: make(map[string]bool)`.

Add a pure helper and test it. In `action_plan_test.go`:

```go
func TestCheckedIDs(t *testing.T) {
	all := []string{"a", "b", "c"}
	excluded := map[string]bool{"b": true}
	got := checkedIDs(all, excluded)
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("want [a c], got %v", got)
	}
	if len(checkedIDs(all, map[string]bool{"a": true, "b": true, "c": true})) != 0 {
		t.Fatal("all excluded should yield empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestCheckedIDs -v`
Expected: FAIL — `undefined: checkedIDs`.

- [ ] **Step 3: Implement `checkedIDs` + email child nodes**

```go
// checkedIDs returns the subset of ids not present in excluded, preserving order.
func checkedIDs(ids []string, excluded map[string]bool) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !excluded[id] {
			out = append(out, id)
		}
	}
	return out
}
```

Build a lookup of message metadata for subject/from. When constructing the state, capture the metas used for analysis into `state.metaByID map[string]*gmailapi.Message` (add the field). Populate it in `openActionPlanWithText` from the `metas`/`messages` you already collected (build from `metas` filtered to analyzed IDs).

Extend `rebuildActionPlanTree` so each category node gets email children:

```go
		for _, id := range c.MessageIDs {
			checked := !state.excluded[id]
			box := "[x]"
			if !checked {
				box = "[ ]"
			}
			subj, from := "(unknown)", ""
			if m := state.metaByID[id]; m != nil {
				subj = extractHeaderValue(m, "Subject")
				from = extractHeaderValue(m, "From")
			}
			child := tview.NewTreeNode(fmt.Sprintf("%s %s — %s", box, subj, from)).
				SetSelectable(true).
				SetColor(colors.Text.Color())
			child.SetReference(emailRef{catIndex: i, msgID: id})
			node.AddChild(child)
		}
		node.SetExpanded(state.expanded[i]) // expand state map; default collapsed
```

Add helper types/fields:

```go
type emailRef struct {
	catIndex int
	msgID    string
}
```

Add `expanded map[int]bool` to `actionPlanState` (init empty). Update the category label to show checked/total:

```go
		checked := len(checkedIDs(c.MessageIDs, state.excluded))
		label := fmt.Sprintf("%s · %d/%d · %s · %s", actionVerbLabel(c.Action), checked, len(c.MessageIDs), c.Name, strings.ToUpper(c.Priority))
```

- [ ] **Step 4: Wire expand/collapse + space toggle in the input capture**

In `actionPlanInputCapture`, before the Up/Down handling:

```go
		cur := state.tree.GetCurrentNode()
		switch ev.Key() {
		case tcell.KeyEnter, tcell.KeyRight:
			if cur != nil {
				if idx, ok := cur.GetReference().(int); ok { // category node
					state.expanded[idx] = !state.expanded[idx]
					cur.SetExpanded(state.expanded[idx])
					return nil
				}
			}
			return nil
		case tcell.KeyLeft:
			if cur != nil {
				if idx, ok := cur.GetReference().(int); ok {
					state.expanded[idx] = false
					cur.SetExpanded(false)
				}
			}
			return nil
		}
```

Add `space` handling (note: space arrives as `ev.Rune() == ' '`). In the rune section:

```go
		if ev.Rune() == ' ' {
			if cur != nil {
				if ref, ok := cur.GetReference().(emailRef); ok {
					state.excluded[ref.msgID] = !state.excluded[ref.msgID]
					a.renderActionPlanPanel(state) // re-render to update [x]/[ ] + counts
				}
			}
			return nil
		}
```

IMPORTANT: rebuilding the tree on toggle resets node objects; preserve the current selection by msgID. Enhance `rebuildActionPlanTree`'s selection-restore to also handle email nodes — track a `state.selectedMsgID string` updated in `SetChangedFunc` when the node is an `emailRef`, and re-select that child after rebuild if present. (Add the field; in `SetChangedFunc`, set `state.selectedMsgID` for email refs and clear it for category refs.)

- [ ] **Step 5: Remove the old `Enter == action` behavior**

The previous `tcell.KeyEnter` "execute action" case is replaced by expand/collapse (above). Action firing moves entirely to the action keys (Task 9). Delete the old Enter case.

- [ ] **Step 6: Build + test**

Run: `go test ./internal/tui/ -run "TestCheckedIDs|TestBuildAnalyzerMessages" -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 7: E2E (tmux)** — expand a category with Enter/→, see emails, `space` toggles `[x]`↔`[ ]`, category count updates, ← collapses. Selection stays put across toggles.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): expandable categories + per-email exclusion in action plan"
```

---

## Task 9: Action-on-checked + context-aware footer (drop `:`/`p`)

**Files:**
- Modify: `internal/tui/action_plan.go` — `executeActionPlanAction`, `updateActionPlanFooter`, input capture cleanup
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing footer test**

The footer text must be a pure function of node kind + category. Add:

```go
func TestActionPlanFooterText(t *testing.T) {
	cat := &servicesActionPlanCategoryStub("archive", 7)
	onCat := actionPlanFooterText(true, "a", "archive", 7)
	if !strings.Contains(onCat, "[a]") || !strings.Contains(onCat, "archive 7") || !strings.Contains(onCat, "[^R]") {
		t.Fatalf("category footer wrong: %q", onCat)
	}
	onEmail := actionPlanFooterText(false, "a", "archive", 7)
	if !strings.Contains(onEmail, "[space]") || !strings.Contains(onEmail, "[^R]") {
		t.Fatalf("email footer wrong: %q", onEmail)
	}
	_ = cat
}
```

Remove the `cat`/stub line if you prefer — the helper takes primitives. Simplify to:

```go
func TestActionPlanFooterText(t *testing.T) {
	onCat := actionPlanFooterText(true, "a", "archive", 7)
	if !strings.Contains(onCat, "[a]") || !strings.Contains(onCat, "archive 7") || !strings.Contains(onCat, "[^R]") {
		t.Fatalf("category footer wrong: %q", onCat)
	}
	onEmail := actionPlanFooterText(false, "a", "archive", 7)
	if !strings.Contains(onEmail, "[space]") || !strings.Contains(onEmail, "[^R]") {
		t.Fatalf("email footer wrong: %q", onEmail)
	}
}
```

Ensure `action_plan_test.go` imports `"strings"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestActionPlanFooterText -v`
Expected: FAIL — `undefined: actionPlanFooterText`.

- [ ] **Step 3: Implement footer text + `updateActionPlanFooter`**

Replace the Task-7 stub:

```go
// actionPlanFooterText builds the context-aware footer. onCategory=true means a
// category node is highlighted; key/verb/count describe its suggested action.
func actionPlanFooterText(onCategory bool, key, action string, checkedCount int) string {
	if onCategory {
		act := "[Enter] expand"
		if key != "" && action != "none" && action != "" {
			act = fmt.Sprintf("[%s] %s %d  [Enter] expand", key, actionRuleVerbShort(action), checkedCount)
		}
		return act + "  [^R] remember  [Esc]"
	}
	return "[space] skip  [^R] remember sender  [←] collapse  [Esc]"
}

// actionRuleVerbShort is the short imperative verb for the footer.
func actionRuleVerbShort(action string) string {
	switch action {
	case "archive":
		return "archive"
	case "mark_read":
		return "read"
	case "trash":
		return "trash"
	case "label":
		return "label"
	default:
		return "act"
	}
}

func (a *App) updateActionPlanFooter(state *actionPlanState) {
	if state == nil || state.footer == nil {
		return
	}
	cur := state.tree.GetCurrentNode()
	onCategory := true
	if cur != nil {
		if _, ok := cur.GetReference().(emailRef); ok {
			onCategory = false
		}
	}
	cat := a.currentActionPlanCategory(state)
	key, action, count := "", "none", 0
	if cat != nil {
		action = cat.Action
		key = a.actionKeyHint(cat.Action)
		count = len(checkedIDs(cat.MessageIDs, state.excluded))
	}
	state.footer.SetText(actionPlanFooterText(onCategory, key, action, count))
}
```

Set the initial footer text in `openActionPlanWithText` via `a.updateActionPlanFooter(state)` after the tree is built (replace the static `state.footer.SetText("[↑↓]...")`).

- [ ] **Step 4: Make actions operate on checked IDs only + drop `:`/`p`**

In `executeActionPlanAction`, replace the `ids := copy(cat.MessageIDs)` block with:

```go
	ids := checkedIDs(cat.MessageIDs, state.excluded)
	if len(ids) == 0 {
		a.GetErrorHandler().ShowWarning(a.ctx, "All emails in this category are excluded — nothing to do")
		return
	}
```

In `actionPlanInputCapture`, DELETE the `:` (CommandMode) and `p` (Prompt) escape-hatch branches and the now-unused `actionPlanOpenPalette`, `actionPlanOpenConfigurator`, `setVirtualBulkSelection` (verify `setVirtualBulkSelection` is not used elsewhere: `grep -rn "setVirtualBulkSelection" internal/tui/`; if used elsewhere, keep it).

- [ ] **Step 5: Build + test**

Run: `go test ./internal/tui/ -run "TestActionPlanFooterText|TestCheckedIDs" -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): action plan acts on checked-only + context-aware footer; drop :/p hatches"
```

---

## Task 10: `Ctrl+R` remember modal + `:action-plan rules` manager

**Files:**
- Create: `internal/tui/action_plan_rules.go`
- Modify: `internal/tui/action_plan.go` (Ctrl+R in input capture), `internal/tui/commands.go` (subcommand)

- [ ] **Step 1: Implement the capture modal**

Create `internal/tui/action_plan_rules.go` (mirror `prompt_preview.go` overlay):

```go
package tui

import (
	"fmt"

	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

const actionPlanRulePage = "actionPlanRule"

// showRememberRuleModal opens an editable input pre-seeded with a suggested rule.
// Enter saves via AnalyzerRulesService; Esc cancels. Synchronous open/close.
func (a *App) showRememberRuleModal(suggestion string) {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	colors := a.GetComponentColors("ai")

	input := tview.NewInputField().
		SetLabel(" Rule: ").
		SetText(suggestion).
		SetFieldWidth(0)
	input.SetBackgroundColor(colors.Background.Color())
	input.SetFieldBackgroundColor(colors.Background.Color())
	input.SetFieldTextColor(colors.Text.Color())

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).
		SetTitle(" 🧠 Remember preference ").
		SetTitleColor(colors.Title.Color()).
		SetBorderColor(colors.Border.Color()).
		SetBackgroundColor(colors.Background.Color())
	box.AddItem(input, 1, 0, true)
	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Enter save · Esc cancel")
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	box.AddItem(footer, 1, 0, false)

	prev := a.GetFocus()
	closeModal := func() {
		a.Pages.RemovePage(actionPlanRulePage)
		if prev != nil {
			a.SetFocus(prev)
		}
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := input.GetText()
			closeModal()
			go func() {
				if err := svc.SaveRule(a.ctx, text); err != nil {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
					return
				}
				a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved — applies on next analysis")
			}()
			return
		}
		closeModal() // Esc / Tab
	})

	centered := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(box, 0, 6, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	a.Pages.AddPage(actionPlanRulePage, centered, true, true)
	a.SetFocus(input)
}
```

- [ ] **Step 2: Wire `Ctrl+R` in the action plan input capture**

In `actionPlanInputCapture`, near the top (before action keys):

```go
		if ev.Key() == tcell.KeyCtrlR {
			cur := state.tree.GetCurrentNode()
			from, action, negate := "", "none", false
			cat := a.currentActionPlanCategory(state)
			if cat != nil {
				action = cat.Action
			}
			if ref, ok := cur.GetReference().(emailRef); ok {
				if m := state.metaByID[ref.msgID]; m != nil {
					from = extractHeaderValue(m, "From")
				}
				negate = true // on a specific email, default to a prohibition
			} else if cat != nil && len(cat.MessageIDs) > 0 {
				if m := state.metaByID[cat.MessageIDs[0]]; m != nil {
					from = extractHeaderValue(m, "From")
				}
			}
			suggestion := ""
			if svc := a.GetAnalyzerRulesService(); svc != nil {
				suggestion = svc.SuggestRuleFromContext(from, action, negate)
			}
			a.showRememberRuleModal(suggestion)
			return nil
		}
```

- [ ] **Step 3: Implement `:action-plan rules` manager**

In `commands.go` `executeActionPlanCommand`, before the final "Unknown option" error:

```go
	if strings.ToLower(args[0]) == "rules" {
		a.openAnalyzerRulesManager()
		return
	}
```

Add `openAnalyzerRulesManager` to `action_plan_rules.go` — a simple modal listing rules with add/delete. Minimal viable implementation: a `tview.List` of existing rules (each item `RuleText`), `a` to add (opens `showRememberRuleModal("")`), `d`/`Delete` to delete the highlighted rule via `svc.DeleteRule`, `Esc` to close. Pre-seed list via `svc.ListRules`. Use the same Pages overlay + `GetComponentColors("ai")` styling. (Full code follows the modal pattern above; build the List, populate from `ListRules`, set an input capture for `a`/`d`/Esc, `AddPage`/`SetFocus`.)

```go
const analyzerRulesPage = "analyzerRules"

func (a *App) openAnalyzerRulesManager() {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	colors := a.GetComponentColors("ai")
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())

	var rules []services.AnalyzerRuleInfo
	reload := func() {
		list.Clear()
		rs, err := svc.ListRules(a.ctx)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("List rules failed: %v", err))
			return
		}
		rules = rs
		if len(rs) == 0 {
			list.AddItem("(no rules yet — press 'a' to add)", "", 0, nil)
			return
		}
		for _, r := range rs {
			list.AddItem(r.RuleText, "", 0, nil)
		}
	}
	reload()

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).SetTitle(" 🧠 Analyzer rules ").
		SetTitleColor(colors.Title.Color()).SetBorderColor(colors.Border.Color()).
		SetBackgroundColor(colors.Background.Color())
	box.AddItem(list, 0, 1, true)
	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("a add · d delete · Esc close")
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	box.AddItem(footer, 1, 0, false)

	prev := a.GetFocus()
	closeModal := func() {
		a.Pages.RemovePage(analyzerRulesPage)
		if prev != nil {
			a.SetFocus(prev)
		}
	}
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEscape:
			closeModal()
			return nil
		case ev.Rune() == 'a':
			closeModal()
			a.showRememberRuleModal("")
			return nil
		case ev.Rune() == 'd':
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(rules) {
				id := rules[idx].ID
				go func() {
					if err := svc.DeleteRule(a.ctx, id); err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Delete failed: %v", err))
						return
					}
					a.QueueUpdateDraw(reload)
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule deleted")
				}()
			}
			return nil
		}
		return ev
	})

	centered := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(box, 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 3, true).
		AddItem(nil, 0, 1, false)
	a.Pages.AddPage(analyzerRulesPage, centered, true, true)
	a.SetFocus(list)
}
```

Add `services` + `tview` + `tcell` imports to `action_plan_rules.go` as needed.

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 5: E2E (tmux)** — In the panel, `Ctrl+R` on an email opens the modal pre-seeded with "Never trash emails from <domain>"; edit + Enter saves (toast). `:action-plan rules` lists it; `a` adds a free general rule; `d` deletes. Confirm global `R` (reload) never fires while the panel/modal is focused.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/action_plan_rules.go internal/tui/action_plan.go internal/tui/commands.go
git commit -m "feat(tui): Ctrl+R remember rule + :action-plan rules manager"
```

---

## Task 11: Load rules into the analysis

**Files:**
- Modify: `internal/tui/action_plan.go` — pass `UserRules` to `Analyze`

- [ ] **Step 1: Load rules before launching analysis**

In `openActionPlanWithText`, before the `go func()` that calls `Analyze`, load the rules (best-effort; failures must not block analysis):

```go
	var userRules []string
	if svc := a.GetAnalyzerRulesService(); svc != nil {
		if rs, err := svc.ListRules(a.ctx); err == nil {
			for _, r := range rs {
				userRules = append(userRules, r.RuleText)
			}
		}
	}
```

- [ ] **Step 2: Pass them into options**

Update the `Analyze` call's options:

```go
		services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules},
```

- [ ] **Step 3: Build + full test**

Run: `go build ./... && make test`
Expected: clean build, tests pass.

- [ ] **Step 4: E2E (tmux)** — Save a rule like "Never trash emails from tldr.tech", then re-open the panel and confirm the analyzer no longer proposes trashing those (LLM-dependent; verify the rule text reaches the prompt via a debug log if needed).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan.go
git commit -m "feat(tui): feed saved preference rules into inbox analysis"
```

---

## Task 12: Documentation

**Files:**
- Modify: `docs/KEYBOARD_SHORTCUTS.md`, `docs/ARCHITECTURE.md` (picker list), in-app help if it lists action plan keys (`grep -rn "action-plan\|Action Plan" internal/tui/*help*`)

- [ ] **Step 1: Document the new gestures**

Update KEYBOARD_SHORTCUTS.md Action Plan section: `P` opens (selection-first / unread fallback); inside: `↑↓` navigate, `Enter`/`→` expand, `←` collapse, `space` exclude an email, action keys (`a`/`t`/`l`/`r`) act on checked-only, `Ctrl+R` remember a rule, `Esc` close. Add `:action-plan rules` command.

- [ ] **Step 2: Verify docs match code**

Run: `grep -rn "palette\|configurator" docs/KEYBOARD_SHORTCUTS.md` — remove any references to the dropped `:`/`p` hatches in the Action Plan context.

- [ ] **Step 3: Commit**

```bash
git add docs/
git commit -m "docs: update Action Plan keyboard shortcuts for rework"
```

---

## Task 13: Final verification

- [ ] **Step 1: Pre-commit check**

Run: `make pre-commit-check`
Expected: fmt + vet + golangci-lint + essential tests all pass. (Note: there may be pre-existing lint issues unrelated to this work — compare against a clean `main` if anything looks unfamiliar.)

- [ ] **Step 2: Full E2E sweep (tmux)**

Verify all four threads end-to-end:
- B: selection vs fallback header correct.
- A: focus/highlight persists after analysis; ↑↓ work.
- D: expand shows emails; `space` excludes; counts update; actions skip excluded.
- C: footer is clear, context-aware, never truncated; `:`/`p` gone.
- Learning: `Ctrl+R` saves; `:action-plan rules` manages; rule text reaches the prompt.

- [ ] **Step 3: Update memory + handoff**

Update `action-plan-feedback.md` (mark threads A–D addressed) and `.remember/remember.md` (Action Plan rework done; auto-refresh now next).

---

## Self-Review Notes (author)

- **Spec coverage:** B→Task 6; D(preview)→Tasks 7-8; D(exclusion)→Task 8; A→Task 7 (TreeView); action trigger→Task 9; C(footer)→Task 9; C(drop hatches)→Task 9; learning store→Tasks 1-2; service→Task 3; injection→Task 4; wiring→Task 5; capture→Task 10; general rules cmd→Task 10; load into analysis→Task 11; docs→Task 12. All covered.
- **Deviation from spec (intentional):** uses a dedicated `GetAnalyzerRulesService()` getter instead of extending the `GetServices()` 12-tuple — matches the existing `GetInboxAnalyzerService()` pattern and avoids a repo-wide arity change.
- **Type consistency:** `emailRef{catIndex,msgID}`, `checkedIDs(ids,excluded)`, `actionPlanFooterText(onCategory,key,action,count)`, `prependUserRules`, `SuggestRuleFromContext(from,action,negate)`, `AnalyzerRuleInfo`, `AnalyzerRule` used consistently across tasks.
- **Risk:** tview `TreeView` re-render-on-toggle must preserve selection (Task 8 Step 4 tracks `selectedMsgID`/`selectedCategory`). Verify in E2E.
- **Threading:** all batch renders stay in the existing `QueueUpdateDraw` in the analysis goroutine; ESC/close stays synchronous; rule saves/deletes run in goroutines with ErrorHandler feedback (CLAUDE.md compliant).
