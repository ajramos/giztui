# Deterministic Rules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A zero-token counterpart to the AI Inbox Action Plan: user-defined rules (Gmail search query → action) managed in a `:rules` panel, runnable as an instant `:rules plan`, used as a prefilter before the AI plan, and optionally mirrored as real Gmail filters.

**Architecture:** New `deterministic_rules` table (SQLite migration v10) + `DeterministicRulesStore`, a `DeterministicRulesService` (CRUD, query validation via `SearchMessages`, first-match-wins `Partition`, Gmail filter sync), and TUI layers that reuse the existing Action Plan panel (tree/exclusions/apply/whole-plan confirm) by extracting its construction into shared helpers. Spec: `docs/superpowers/specs/2026-07-04-deterministic-rules-design.md`.

**Tech Stack:** Go, derailed/tview (NOT thread-safe — all UI mutations on the UI thread via `QueueUpdateDraw`), SQLite (`PRAGMA user_version` migrations), Gmail API (`Users.Settings.Filters`), mockery mocks + testify.

**Project rules that bind every task:**
- Commit messages: clean, conventional, **NO Claude signatures / Co-Authored-By lines**.
- User feedback ONLY via `a.GetErrorHandler().ShowError/ShowSuccess/ShowWarning/ShowInfo/ShowProgress` — `go`-wrapped when called from the UI goroutine, direct from worker goroutines. Never `fmt.Printf`.
- Never `QueueUpdateDraw` in ESC handlers; panel/tree mutations from workers only inside `QueueUpdateDraw` guarded by `a.actionPlanState == state`.
- Run `gofmt -w` on touched files before each commit (no need to ask).
- Run `make test` scoped packages, not `go test ./...`.
- After code changes in a task: `graphify update .` is run once at the end (Task 16), not per task.

---

### Task 1: DB migration v10 + DeterministicRulesStore

**Files:**
- Modify: `internal/db/store.go` (migration chain, after the v9 block at ~line 393-413, before the final `return nil` of `migrate`)
- Create: `internal/db/deterministic_rules_store.go`
- Test: `internal/db/deterministic_rules_store_test.go`

Mirror `internal/db/analyzer_rules_store.go` and its test. **Head of the migration chain is v9 → this adds v10.** Critical difference from the analyzer store: `ListRules` orders **ASC (creation order)** because rules apply first-match-wins in creation order — the analyzer store orders DESC, do NOT copy that line blindly.

- [ ] **Step 1: Write the failing store test**

Create `internal/db/deterministic_rules_store_test.go` (mirror the setup pattern of `analyzer_rules_store_test.go`: `Open(ctx, t.TempDir()+"/rules.db")`, `defer store.Close()`):

```go
package db

import (
	"context"
	"testing"
)

func TestDeterministicRulesStoreCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/rules.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()
	s := NewDeterministicRulesStore(store)
	const acct = "user@example.com"

	// Save two rules; List must return them in CREATION order (first-match-wins).
	r1, err := s.SaveRule(ctx, acct, "from:newsletter@medium.com", "archive", "", 0)
	if err != nil {
		t.Fatalf("save r1: %v", err)
	}
	r2, err := s.SaveRule(ctx, acct, "from:jira@corp.com", "prompt", "", 7)
	if err != nil {
		t.Fatalf("save r2: %v", err)
	}
	rules, err := s.ListRules(ctx, acct)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rules) != 2 || rules[0].ID != r1.ID || rules[1].ID != r2.ID {
		t.Fatalf("want creation order [r1 r2], got %+v", rules)
	}
	if rules[1].PromptID != 7 || rules[1].Action != "prompt" {
		t.Fatalf("prompt rule fields lost: %+v", rules[1])
	}

	// Update r1 to a label rule.
	if err := s.UpdateRule(ctx, acct, r1.ID, "from:amazon.com", "label", "Compras", 0); err != nil {
		t.Fatalf("update: %v", err)
	}
	rules, _ = s.ListRules(ctx, acct)
	if rules[0].Query != "from:amazon.com" || rules[0].Action != "label" || rules[0].Label != "Compras" {
		t.Fatalf("update not persisted: %+v", rules[0])
	}

	// Gmail filter ID round-trip.
	if err := s.SetGmailFilterID(ctx, acct, r1.ID, "FILTER123"); err != nil {
		t.Fatalf("set filter id: %v", err)
	}
	rules, _ = s.ListRules(ctx, acct)
	if rules[0].GmailFilterID != "FILTER123" {
		t.Fatalf("filter id not persisted: %+v", rules[0])
	}

	// Account scoping.
	other, err := s.ListRules(ctx, "other@example.com")
	if err != nil || len(other) != 0 {
		t.Fatalf("account scoping broken: %v %+v", err, other)
	}

	// Delete.
	if err := s.DeleteRule(ctx, acct, r1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.DeleteRule(ctx, acct, r1.ID); err == nil {
		t.Fatal("second delete should fail with rule not found")
	}
	rules, _ = s.ListRules(ctx, acct)
	if len(rules) != 1 || rules[0].ID != r2.ID {
		t.Fatalf("want only r2 left, got %+v", rules)
	}
}

func TestDeterministicRulesStoreValidation(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/rules.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()
	s := NewDeterministicRulesStore(store)

	if _, err := s.SaveRule(ctx, "", "from:x", "archive", "", 0); err == nil {
		t.Fatal("empty account must fail")
	}
	if _, err := s.SaveRule(ctx, "a@b.c", "  ", "archive", "", 0); err == nil {
		t.Fatal("empty query must fail")
	}
	if _, err := s.SaveRule(ctx, "a@b.c", "from:x", "explode", "", 0); err == nil {
		t.Fatal("unknown action must fail")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db/ -run TestDeterministicRulesStore -v`
Expected: FAIL (compile error: `NewDeterministicRulesStore` undefined)

- [ ] **Step 3: Add the v10 migration block**

In `internal/db/store.go`, inside `migrate`, insert after the v9 block (`ver = 9`) and before `return nil` — copy the exact shape of the v8 block:

```go
	// v10: deterministic rules (Gmail-query → action, optional Gmail filter mirror)
	if ver == 9 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS deterministic_rules (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  account_email   TEXT NOT NULL,
  query           TEXT NOT NULL,
  action          TEXT NOT NULL,
  label           TEXT NOT NULL DEFAULT '',
  prompt_id       INTEGER NOT NULL DEFAULT 0,
  gmail_filter_id TEXT NOT NULL DEFAULT '',
  created_at      INTEGER NOT NULL
);`)

		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=10;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v10: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 10
	}
```

(If golangci-lint flags the trailing `ver = 10` as ineffectual, append `_ = ver` after the last block — but first check how the existing v9 block gets away with it and match that.)

- [ ] **Step 4: Implement the store**

Create `internal/db/deterministic_rules_store.go`:

```go
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DeterministicRule is one Gmail-query → action rule, applied first-match-wins in
// creation order. label is set when action == "label"; prompt_id when action == "prompt";
// gmail_filter_id is non-empty when the rule is mirrored as a server-side Gmail filter.
type DeterministicRule struct {
	ID            int64  `json:"id"`
	AccountEmail  string `json:"account_email"`
	Query         string `json:"query"`
	Action        string `json:"action"`
	Label         string `json:"label"`
	PromptID      int64  `json:"prompt_id"`
	GmailFilterID string `json:"gmail_filter_id"`
	CreatedAt     int64  `json:"created_at"`
}

// validRuleActions is the closed set of rule actions the store accepts.
var validRuleActions = map[string]bool{
	"archive": true, "mark_read": true, "trash": true, "label": true, "prompt": true,
}

// DeterministicRulesStore handles persistence of deterministic rules.
type DeterministicRulesStore struct {
	db *sql.DB
}

// NewDeterministicRulesStore creates a new deterministic rules store.
func NewDeterministicRulesStore(store *Store) *DeterministicRulesStore {
	return &DeterministicRulesStore{db: store.DB()}
}

// SaveRule inserts a new rule for the account and returns it.
func (s *DeterministicRulesStore) SaveRule(ctx context.Context, accountEmail, query, action, label string, promptID int64) (*DeterministicRule, error) {
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("account_email and query cannot be empty")
	}
	if !validRuleActions[action] {
		return nil, fmt.Errorf("unknown action %q", action)
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO deterministic_rules (account_email, query, action, label, prompt_id, gmail_filter_id, created_at)
		VALUES (?, ?, ?, ?, ?, '', ?)`,
		accountEmail, strings.TrimSpace(query), action, strings.TrimSpace(label), promptID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to save rule: %w", err)
	}
	id, _ := res.LastInsertId()
	return &DeterministicRule{ID: id, AccountEmail: accountEmail, Query: strings.TrimSpace(query), Action: action, Label: strings.TrimSpace(label), PromptID: promptID, CreatedAt: now}, nil
}

// ListRules returns all rules for an account in CREATION order (first-match-wins order).
func (s *DeterministicRulesStore) ListRules(ctx context.Context, accountEmail string) ([]*DeterministicRule, error) {
	if strings.TrimSpace(accountEmail) == "" {
		return nil, fmt.Errorf("account_email cannot be empty")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_email, query, action, label, prompt_id, gmail_filter_id, created_at
		FROM deterministic_rules
		WHERE account_email = ?
		ORDER BY created_at ASC, id ASC`, accountEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*DeterministicRule
	for rows.Next() {
		r := &DeterministicRule{}
		if err := rows.Scan(&r.ID, &r.AccountEmail, &r.Query, &r.Action, &r.Label, &r.PromptID, &r.GmailFilterID, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return out, nil
}

// UpdateRule rewrites the query/action/label/prompt of an existing rule.
func (s *DeterministicRulesStore) UpdateRule(ctx context.Context, accountEmail string, id int64, query, action, label string, promptID int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 || strings.TrimSpace(query) == "" {
		return fmt.Errorf("account_email, id and query are required")
	}
	if !validRuleActions[action] {
		return fmt.Errorf("unknown action %q", action)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE deterministic_rules SET query = ?, action = ?, label = ?, prompt_id = ?
		WHERE account_email = ? AND id = ?`,
		strings.TrimSpace(query), action, strings.TrimSpace(label), promptID, accountEmail, id)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

// SetGmailFilterID records (or clears, with "") the mirrored Gmail filter's ID.
func (s *DeterministicRulesStore) SetGmailFilterID(ctx context.Context, accountEmail string, id int64, filterID string) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE deterministic_rules SET gmail_filter_id = ?
		WHERE account_email = ? AND id = ?`, filterID, accountEmail, id)
	if err != nil {
		return fmt.Errorf("failed to set gmail filter id: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

// DeleteRule removes a rule by id for the account.
func (s *DeterministicRulesStore) DeleteRule(ctx context.Context, accountEmail string, id int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM deterministic_rules WHERE account_email = ? AND id = ?`, accountEmail, id)
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

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/db/ -v`
Expected: new tests PASS. Note: `TestMigration_V6_SavedQueriesTable` has a known pre-existing failure on main — if it fails, confirm it also fails on `git stash`-clean state before worrying.

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/db/store.go internal/db/deterministic_rules_store.go internal/db/deterministic_rules_store_test.go
git add internal/db/store.go internal/db/deterministic_rules_store.go internal/db/deterministic_rules_store_test.go
git commit -m "feat(db): deterministic_rules table (v10) + store"
```

---

### Task 2: Service interface, types, CRUD + query validation

**Files:**
- Modify: `internal/services/interfaces.go` (append after the `AnalyzerRulesService` interface at ~line 1049-1068; also extend `ActionPlanCategory` at ~line 996-1034)
- Create: `internal/services/deterministic_rules_service.go`
- Test: `internal/services/deterministic_rules_service_test.go`

- [ ] **Step 1: Add types + interface to `interfaces.go`**

Append after `AnalyzerRulesService`/`AnalyzerRuleInfo`:

```go
// DeterministicRuleInfo describes one deterministic rule (Gmail query → action) for the UI.
type DeterministicRuleInfo struct {
	ID            int64
	Query         string
	Action        string // archive | mark_read | trash | label | prompt
	Label         string // when Action == "label"
	PromptID      int64  // when Action == "prompt" (prompt_templates.id)
	GmailFilterID string // non-empty when mirrored as a server-side Gmail filter
	CreatedAt     int64
}

// RuleMatch pairs a rule with the message IDs its query matched during a sweep.
type RuleMatch struct {
	Rule       DeterministicRuleInfo
	MessageIDs []string
}

// DeterministicRulesService manages deterministic rules: CRUD (with Gmail-side query
// validation at save time), first-match-wins partitioning of a message set, and optional
// mirroring of rules as real Gmail filters.
type DeterministicRulesService interface {
	SetAccountEmail(email string)
	SaveRule(ctx context.Context, query, action, label string, promptID int64) (*DeterministicRuleInfo, error)
	UpdateRule(ctx context.Context, id int64, query, action, label string, promptID int64) error
	ListRules(ctx context.Context) ([]DeterministicRuleInfo, error)
	DeleteRule(ctx context.Context, id int64) error
	// Partition runs each rule's query (creation order) prefixed by scopeQuery and assigns
	// every message to the FIRST rule that matches it. candidates == nil means "no
	// intersection" (take whatever Gmail returns, deduped across rules); otherwise matches
	// are intersected with candidates and remaining returns the unmatched candidates in
	// input order. Rules that match nothing produce no RuleMatch entry.
	Partition(ctx context.Context, scopeQuery string, candidates []string) (matches []RuleMatch, remaining []string, err error)
	// SyncRule mirrors the rule as a Gmail filter (recreating it if already mirrored);
	// UnsyncRule deletes the mirrored filter. Both persist gmail_filter_id.
	SyncRule(ctx context.Context, id int64) error
	UnsyncRule(ctx context.Context, id int64) error
}
```

In the same file, add a `PromptID` field to `ActionPlanCategory` (after `Label`). NOTE: this struct has NO json tags — keep the new field untagged, matching the existing style:

```go
	PromptID   int64    // saved prompt (prompt_templates.id), set only when Action == "prompt" (deterministic rules)
```

- [ ] **Step 2: Write the failing service test (CRUD + validation)**

Create `internal/services/deterministic_rules_service_test.go`. Uses the existing mockery mock `mocks.MessageRepository` (`internal/services/mocks/message_repository.go`, testify-style `_m.Called(ctx, query, opts)`) and an in-memory store via `db.Open(ctx, t.TempDir()+"/svc.db")`:

```go
package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/ajramos/giztui/internal/db"
	"github.com/ajramos/giztui/internal/services/mocks"
	"github.com/stretchr/testify/mock"
	gmailapi "google.golang.org/api/gmail/v1"
)

func newTestRulesService(t *testing.T, repo MessageRepository) *DeterministicRulesServiceImpl {
	t.Helper()
	ctx := context.Background()
	store, err := db.Open(ctx, t.TempDir()+"/svc.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := NewDeterministicRulesService(db.NewDeterministicRulesStore(store), repo, nil, nil)
	svc.SetAccountEmail("user@example.com")
	return svc
}

// page builds a MessagePage of ids for the mocked repository.
func page(next string, ids ...string) *MessagePage {
	p := &MessagePage{NextPageToken: next}
	for _, id := range ids {
		p.Messages = append(p.Messages, &gmailapi.Message{Id: id})
	}
	return p
}

func TestRulesServiceSaveValidatesQueryAgainstGmail(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	ctx := context.Background()

	// Valid query: SearchMessages(MaxResults:1) succeeds → rule saved.
	repo.On("SearchMessages", mock.Anything, "from:medium.com", QueryOptions{MaxResults: 1}).
		Return(page(""), nil).Once()
	r, err := svc.SaveRule(ctx, "from:medium.com", "archive", "", 0)
	if err != nil || r == nil || r.ID <= 0 {
		t.Fatalf("save valid rule: %v %+v", err, r)
	}

	// Gmail rejects the query → save fails, nothing persisted.
	repo.On("SearchMessages", mock.Anything, "from:(broken", QueryOptions{MaxResults: 1}).
		Return(nil, fmt.Errorf("400 invalid query")).Once()
	if _, err := svc.SaveRule(ctx, "from:(broken", "archive", "", 0); err == nil {
		t.Fatal("invalid query must fail")
	}
	rules, _ := svc.ListRules(ctx)
	if len(rules) != 1 {
		t.Fatalf("rejected rule must not persist, got %+v", rules)
	}
}

func TestRulesServiceFieldValidation(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	ctx := context.Background()

	if _, err := svc.SaveRule(ctx, "from:x", "label", "", 0); err == nil {
		t.Fatal("label action without label must fail")
	}
	if _, err := svc.SaveRule(ctx, "from:x", "prompt", "", 0); err == nil {
		t.Fatal("prompt action without prompt id must fail")
	}
	if _, err := svc.SaveRule(ctx, "from:x", "explode", "", 0); err == nil {
		t.Fatal("unknown action must fail")
	}
	// None of the above may reach Gmail (field validation happens first) — mockery
	// asserts no unexpected SearchMessages calls at cleanup.
}

func TestRulesServiceNoAccount(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	svc.SetAccountEmail("")
	if _, err := svc.ListRules(context.Background()); err == nil {
		t.Fatal("no account must fail, not panic")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestRulesService -v`
Expected: FAIL (compile error: `NewDeterministicRulesService` undefined)

- [ ] **Step 4: Implement the service (CRUD + validation; Partition/Sync stubs)**

Create `internal/services/deterministic_rules_service.go`:

```go
package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/db"
	gmailapi "google.golang.org/api/gmail/v1"
)

// GmailFilterAPI is the narrow Gmail settings surface the rules service needs for
// mirroring rules as server-side filters. *gmail.Client implements it (Task 4).
type GmailFilterAPI interface {
	CreateFilter(query string, action *gmailapi.FilterAction) (string, error)
	DeleteFilter(id string) error
}

// DeterministicRulesServiceImpl implements DeterministicRulesService.
// labels and filters may be nil (no Gmail client / no label service): CRUD and
// Partition still work; SyncRule/UnsyncRule fail with a clear error.
type DeterministicRulesServiceImpl struct {
	store   *db.DeterministicRulesStore
	repo    MessageRepository
	labels  LabelService
	filters GmailFilterAPI

	mu           sync.RWMutex
	accountEmail string
}

// NewDeterministicRulesService creates the deterministic rules service.
func NewDeterministicRulesService(store *db.DeterministicRulesStore, repo MessageRepository, labels LabelService, filters GmailFilterAPI) *DeterministicRulesServiceImpl {
	return &DeterministicRulesServiceImpl{store: store, repo: repo, labels: labels, filters: filters}
}

// SetAccountEmail scopes all operations to the given account.
func (s *DeterministicRulesServiceImpl) SetAccountEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountEmail = email
}

func (s *DeterministicRulesServiceImpl) account() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if strings.TrimSpace(s.accountEmail) == "" {
		return "", fmt.Errorf("no active account")
	}
	return s.accountEmail, nil
}

// validateRuleFields checks the action/label/prompt combination before touching Gmail.
func validateRuleFields(query, action, label string, promptID int64) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("query cannot be empty")
	}
	switch action {
	case "archive", "mark_read", "trash":
		return nil
	case "label":
		if strings.TrimSpace(label) == "" {
			return fmt.Errorf("label action needs a label name")
		}
		return nil
	case "prompt":
		if promptID <= 0 {
			return fmt.Errorf("prompt action needs a saved prompt")
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}

// validateQuery runs the query through Gmail search (MaxResults 1) so a syntax error
// surfaces at save time, not later during a sweep.
func (s *DeterministicRulesServiceImpl) validateQuery(ctx context.Context, query string) error {
	if s.repo == nil {
		return fmt.Errorf("message repository not available")
	}
	if _, err := s.repo.SearchMessages(ctx, query, QueryOptions{MaxResults: 1}); err != nil {
		return fmt.Errorf("Gmail rejected the query: %w", err)
	}
	return nil
}

func ruleInfoFromDB(r *db.DeterministicRule) DeterministicRuleInfo {
	return DeterministicRuleInfo{
		ID: r.ID, Query: r.Query, Action: r.Action, Label: r.Label,
		PromptID: r.PromptID, GmailFilterID: r.GmailFilterID, CreatedAt: r.CreatedAt,
	}
}

// SaveRule validates fields + Gmail query syntax, then persists the rule.
func (s *DeterministicRulesServiceImpl) SaveRule(ctx context.Context, query, action, label string, promptID int64) (*DeterministicRuleInfo, error) {
	acct, err := s.account()
	if err != nil {
		return nil, err
	}
	if err := validateRuleFields(query, action, label, promptID); err != nil {
		return nil, err
	}
	if err := s.validateQuery(ctx, query); err != nil {
		return nil, err
	}
	saved, err := s.store.SaveRule(ctx, acct, query, action, label, promptID)
	if err != nil {
		return nil, err
	}
	info := ruleInfoFromDB(saved)
	return &info, nil
}

// UpdateRule validates and rewrites an existing rule. Re-mirroring an already-synced
// rule is the caller's job (manager calls SyncRule afterwards).
func (s *DeterministicRulesServiceImpl) UpdateRule(ctx context.Context, id int64, query, action, label string, promptID int64) error {
	acct, err := s.account()
	if err != nil {
		return err
	}
	if err := validateRuleFields(query, action, label, promptID); err != nil {
		return err
	}
	if err := s.validateQuery(ctx, query); err != nil {
		return err
	}
	return s.store.UpdateRule(ctx, acct, id, query, action, label, promptID)
}

// ListRules returns the account's rules in creation (first-match-wins) order.
func (s *DeterministicRulesServiceImpl) ListRules(ctx context.Context) ([]DeterministicRuleInfo, error) {
	acct, err := s.account()
	if err != nil {
		return nil, err
	}
	rules, err := s.store.ListRules(ctx, acct)
	if err != nil {
		return nil, err
	}
	out := make([]DeterministicRuleInfo, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleInfoFromDB(r))
	}
	return out, nil
}

// DeleteRule removes the rule; if mirrored, it deletes the Gmail filter first (best
// effort — a failed remote delete still removes the local rule but returns the error).
func (s *DeterministicRulesServiceImpl) DeleteRule(ctx context.Context, id int64) error {
	acct, err := s.account()
	if err != nil {
		return err
	}
	rule, err := s.findRule(ctx, id)
	if err != nil {
		return err
	}
	var filterErr error
	if rule.GmailFilterID != "" && s.filters != nil {
		filterErr = s.filters.DeleteFilter(rule.GmailFilterID)
	}
	if err := s.store.DeleteRule(ctx, acct, id); err != nil {
		return err
	}
	if filterErr != nil {
		return fmt.Errorf("rule deleted locally, but the Gmail filter could not be removed: %w", filterErr)
	}
	return nil
}

func (s *DeterministicRulesServiceImpl) findRule(ctx context.Context, id int64) (*DeterministicRuleInfo, error) {
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i], nil
		}
	}
	return nil, fmt.Errorf("rule not found")
}
```

Also add TEMPORARY stubs so the interface is satisfied (implemented for real in Tasks 3 and 5 — replace, don't keep):

```go
// Partition is implemented in Task 3.
func (s *DeterministicRulesServiceImpl) Partition(ctx context.Context, scopeQuery string, candidates []string) ([]RuleMatch, []string, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

// SyncRule / UnsyncRule are implemented in Task 5.
func (s *DeterministicRulesServiceImpl) SyncRule(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}
func (s *DeterministicRulesServiceImpl) UnsyncRule(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}
```

- [ ] **Step 5: Compile-check interface conformance**

Add at the bottom of `deterministic_rules_service.go`:

```go
var _ DeterministicRulesService = (*DeterministicRulesServiceImpl)(nil)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestRulesService -v`
Expected: PASS

- [ ] **Step 7: Regenerate mocks (new interface → mockery)**

Run: `make test-mocks` then `go build ./...`
Expected: `internal/services/mocks/` gains a `DeterministicRulesService` mock; build green.

- [ ] **Step 8: Commit**

```bash
gofmt -w internal/services/interfaces.go internal/services/deterministic_rules_service.go internal/services/deterministic_rules_service_test.go
git add internal/services/ 
git commit -m "feat(services): DeterministicRulesService — CRUD with Gmail query validation"
```

---

### Task 3: Partition (first-match-wins sweep)

**Files:**
- Modify: `internal/services/deterministic_rules_service.go` (replace the Partition stub)
- Test: `internal/services/deterministic_rules_service_test.go` (append)

- [ ] **Step 1: Write the failing tests**

Append to `internal/services/deterministic_rules_service_test.go`:

```go
// seedRule inserts a rule bypassing Gmail validation (direct store write) so Partition
// tests don't need SearchMessages expectations for the save path.
func seedRule(t *testing.T, svc *DeterministicRulesServiceImpl, query, action, label string, promptID int64) {
	t.Helper()
	if _, err := svc.store.SaveRule(context.Background(), "user@example.com", query, action, label, promptID); err != nil {
		t.Fatalf("seed rule: %v", err)
	}
}

func TestRulesServicePartitionFirstMatchWins(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	ctx := context.Background()
	seedRule(t, svc, "from:medium.com", "archive", "", 0)
	seedRule(t, svc, "is:important", "mark_read", "", 0)

	// Rule queries run scoped, in creation order. m2 matches BOTH rules → belongs to rule 1 only.
	repo.On("SearchMessages", mock.Anything, "in:inbox from:medium.com", QueryOptions{MaxResults: 100}).
		Return(page("", "m1", "m2"), nil).Once()
	repo.On("SearchMessages", mock.Anything, "in:inbox is:important", QueryOptions{MaxResults: 100}).
		Return(page("", "m2", "m3"), nil).Once()

	matches, remaining, err := svc.Partition(ctx, "in:inbox", nil)
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("want 2 matches, got %+v", matches)
	}
	if got := matches[0].MessageIDs; len(got) != 2 || got[0] != "m1" || got[1] != "m2" {
		t.Fatalf("rule 1 should own m1,m2: %v", got)
	}
	if got := matches[1].MessageIDs; len(got) != 1 || got[0] != "m3" {
		t.Fatalf("rule 2 should own only m3 (first match wins): %v", got)
	}
	if remaining != nil {
		t.Fatalf("nil candidates → nil remaining, got %v", remaining)
	}
}

func TestRulesServicePartitionIntersectsCandidates(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	ctx := context.Background()
	seedRule(t, svc, "from:medium.com", "archive", "", 0)

	// Gmail returns m1,m9 but only m1,m2,m3 are candidates → match {m1}, remaining {m2,m3} in input order.
	repo.On("SearchMessages", mock.Anything, "in:inbox is:unread from:medium.com", QueryOptions{MaxResults: 100}).
		Return(page("", "m1", "m9"), nil).Once()

	matches, remaining, err := svc.Partition(ctx, "in:inbox is:unread", []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if len(matches) != 1 || len(matches[0].MessageIDs) != 1 || matches[0].MessageIDs[0] != "m1" {
		t.Fatalf("want match {m1}, got %+v", matches)
	}
	if len(remaining) != 2 || remaining[0] != "m2" || remaining[1] != "m3" {
		t.Fatalf("want remaining [m2 m3], got %v", remaining)
	}
}

func TestRulesServicePartitionPaginates(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	seedRule(t, svc, "from:medium.com", "archive", "", 0)

	repo.On("SearchMessages", mock.Anything, "in:inbox from:medium.com", QueryOptions{MaxResults: 100}).
		Return(page("tok2", "m1"), nil).Once()
	repo.On("SearchMessages", mock.Anything, "in:inbox from:medium.com", QueryOptions{MaxResults: 100, PageToken: "tok2"}).
		Return(page("", "m2"), nil).Once()

	matches, _, err := svc.Partition(context.Background(), "in:inbox", nil)
	if err != nil || len(matches) != 1 || len(matches[0].MessageIDs) != 2 {
		t.Fatalf("pagination broken: %v %+v", err, matches)
	}
}
```

Note: `seedRule` accesses `svc.store` — same package, fine.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestRulesServicePartition -v`
Expected: FAIL with "not implemented"

- [ ] **Step 3: Implement Partition (replace the stub)**

```go
// Partition search paging: pages of 100 IDs, hard cap of 500 per rule so a catch-all
// query cannot sweep an entire mailbox in one go.
const (
	partitionPageSize   = 100
	partitionMaxPerRule = 500
)

// Partition runs every rule's query (creation order, sequential — Gmail quota and
// coherent progress) and assigns each message to the first rule that matches it.
func (s *DeterministicRulesServiceImpl) Partition(ctx context.Context, scopeQuery string, candidates []string) ([]RuleMatch, []string, error) {
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, nil, err
	}
	var candidateSet map[string]bool
	if candidates != nil {
		candidateSet = make(map[string]bool, len(candidates))
		for _, id := range candidates {
			candidateSet[id] = true
		}
	}
	seen := make(map[string]bool)
	var matches []RuleMatch
	for _, r := range rules {
		query := strings.TrimSpace(strings.TrimSpace(scopeQuery) + " " + r.Query)
		ids, err := s.searchAllIDs(ctx, query)
		if err != nil {
			return nil, nil, fmt.Errorf("rule %q: %w", r.Query, err)
		}
		var mine []string
		for _, id := range ids {
			if seen[id] {
				continue // first match wins — an earlier rule already owns this message
			}
			if candidateSet != nil && !candidateSet[id] {
				continue
			}
			seen[id] = true
			mine = append(mine, id)
		}
		if len(mine) > 0 {
			matches = append(matches, RuleMatch{Rule: r, MessageIDs: mine})
		}
	}
	var remaining []string
	if candidates != nil {
		remaining = make([]string, 0, len(candidates))
		for _, id := range candidates {
			if !seen[id] {
				remaining = append(remaining, id)
			}
		}
	}
	return matches, remaining, nil
}

// searchAllIDs collects message IDs for query across pages, capped at partitionMaxPerRule.
func (s *DeterministicRulesServiceImpl) searchAllIDs(ctx context.Context, query string) ([]string, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("message repository not available")
	}
	var ids []string
	pageToken := ""
	for {
		res, err := s.repo.SearchMessages(ctx, query, QueryOptions{MaxResults: partitionPageSize, PageToken: pageToken})
		if err != nil {
			return nil, err
		}
		for _, m := range res.Messages {
			if m != nil {
				ids = append(ids, m.Id)
			}
		}
		if res.NextPageToken == "" || len(ids) >= partitionMaxPerRule {
			return ids, nil
		}
		pageToken = res.NextPageToken
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestRulesService -v`
Expected: PASS (all, including Task 2's)

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/services/deterministic_rules_service.go internal/services/deterministic_rules_service_test.go
git add internal/services/deterministic_rules_service.go internal/services/deterministic_rules_service_test.go
git commit -m "feat(services): first-match-wins rule partitioning with candidate intersection"
```

---

### Task 4: Gmail filter client methods

**Files:**
- Create: `internal/gmail/filters.go`

Thin API wrapper — no unit test (nothing to test without the network); Task 5 tests the logic above it through the `GmailFilterAPI` interface. Compile check + interface conformance is the gate.

- [ ] **Step 1: Implement the wrapper**

Create `internal/gmail/filters.go`. Note: the existing wrappers in `client.go` (e.g. `GetMessage` at :88) use a local `user := "me"` — match that style:

```go
package gmail

import (
	gmailapi "google.golang.org/api/gmail/v1"
)

// CreateFilter creates a server-side Gmail filter (Settings → Filters) that applies
// action to FUTURE incoming mail matching query. Returns the created filter's ID.
// Requires the gmail.settings.basic OAuth scope — older tokens without it fail here
// and the user must re-authorize.
func (c *Client) CreateFilter(query string, action *gmailapi.FilterAction) (string, error) {
	user := "me"
	f := &gmailapi.Filter{
		Criteria: &gmailapi.FilterCriteria{Query: query},
		Action:   action,
	}
	created, err := c.Service.Users.Settings.Filters.Create(user, f).Do()
	if err != nil {
		return "", err
	}
	return created.Id, nil
}

// DeleteFilter removes a server-side Gmail filter by ID.
func (c *Client) DeleteFilter(id string) error {
	user := "me"
	return c.Service.Users.Settings.Filters.Delete(user, id).Do()
}
```

Check the `gmailapi` import alias against the rest of the package first — if `client.go` imports the API package under a different name (e.g. plain `gmail`), use the same alias.

- [ ] **Step 2: Verify *gmail.Client satisfies services.GmailFilterAPI**

Add to `internal/services/deterministic_rules_service_test.go`:

```go
var _ GmailFilterAPI = (*gmailclient.Client)(nil)
```

with import `gmailclient "github.com/ajramos/giztui/internal/gmail"`.

Run: `go build ./... && go vet ./internal/gmail/ ./internal/services/`
Expected: clean

- [ ] **Step 3: Commit**

```bash
gofmt -w internal/gmail/filters.go internal/services/deterministic_rules_service_test.go
git add internal/gmail/filters.go internal/services/deterministic_rules_service_test.go
git commit -m "feat(gmail): Users.Settings.Filters create/delete wrappers"
```

---

### Task 5: Filter action mapping + SyncRule / UnsyncRule

**Files:**
- Modify: `internal/services/deterministic_rules_service.go` (add mapping; replace Sync/Unsync stubs)
- Test: `internal/services/deterministic_rules_service_test.go` (append)

- [ ] **Step 1: Write the failing tests**

Append:

```go
// fakeFilterAPI records filter calls for sync tests.
type fakeFilterAPI struct {
	created []string // queries passed to CreateFilter
	deleted []string // ids passed to DeleteFilter
	nextID  string
	fail    error
}

func (f *fakeFilterAPI) CreateFilter(query string, action *gmailapi.FilterAction) (string, error) {
	if f.fail != nil {
		return "", f.fail
	}
	f.created = append(f.created, query)
	return f.nextID, nil
}
func (f *fakeFilterAPI) DeleteFilter(id string) error {
	if f.fail != nil {
		return f.fail
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestFilterActionForRule(t *testing.T) {
	cases := []struct {
		action, labelID string
		wantAdd         []string
		wantRemove      []string
		wantErr         bool
	}{
		{action: "archive", wantRemove: []string{"INBOX"}},
		{action: "mark_read", wantRemove: []string{"UNREAD"}},
		{action: "trash", wantAdd: []string{"TRASH"}},
		{action: "label", labelID: "Label_42", wantAdd: []string{"Label_42"}},
		{action: "label", labelID: "", wantErr: true},
		{action: "prompt", wantErr: true},
		{action: "explode", wantErr: true},
	}
	for _, c := range cases {
		got, err := FilterActionForRule(c.action, c.labelID)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: want error", c.action)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.action, err)
			continue
		}
		if len(c.wantAdd) > 0 && (len(got.AddLabelIds) != 1 || got.AddLabelIds[0] != c.wantAdd[0]) {
			t.Errorf("%s: add %v, want %v", c.action, got.AddLabelIds, c.wantAdd)
		}
		if len(c.wantRemove) > 0 && (len(got.RemoveLabelIds) != 1 || got.RemoveLabelIds[0] != c.wantRemove[0]) {
			t.Errorf("%s: remove %v, want %v", c.action, got.RemoveLabelIds, c.wantRemove)
		}
	}
}

func TestRulesServiceSyncUnsync(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	ctx := context.Background()
	filters := &fakeFilterAPI{nextID: "F1"}
	svc.filters = filters
	seedRule(t, svc, "from:medium.com", "archive", "", 0)
	rules, _ := svc.ListRules(ctx)
	id := rules[0].ID

	if err := svc.SyncRule(ctx, id); err != nil {
		t.Fatalf("sync: %v", err)
	}
	rules, _ = svc.ListRules(ctx)
	if rules[0].GmailFilterID != "F1" {
		t.Fatalf("filter id not stored: %+v", rules[0])
	}
	if len(filters.created) != 1 || filters.created[0] != "from:medium.com" {
		t.Fatalf("create not called with rule query: %v", filters.created)
	}

	// Re-sync recreates: delete old + create new.
	filters.nextID = "F2"
	if err := svc.SyncRule(ctx, id); err != nil {
		t.Fatalf("resync: %v", err)
	}
	if len(filters.deleted) != 1 || filters.deleted[0] != "F1" {
		t.Fatalf("resync must delete the old filter: %v", filters.deleted)
	}
	rules, _ = svc.ListRules(ctx)
	if rules[0].GmailFilterID != "F2" {
		t.Fatalf("resync id not stored: %+v", rules[0])
	}

	// Unsync deletes and clears.
	if err := svc.UnsyncRule(ctx, id); err != nil {
		t.Fatalf("unsync: %v", err)
	}
	rules, _ = svc.ListRules(ctx)
	if rules[0].GmailFilterID != "" {
		t.Fatalf("unsync must clear the id: %+v", rules[0])
	}
}

func TestRulesServiceSyncPromptRuleRejected(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	svc.filters = &fakeFilterAPI{nextID: "F1"}
	seedRule(t, svc, "from:jira", "prompt", "", 7)
	rules, _ := svc.ListRules(context.Background())
	if err := svc.SyncRule(context.Background(), rules[0].ID); err == nil {
		t.Fatal("prompt rules cannot be mirrored — sync must fail")
	}
}

func TestRulesServiceDeleteMirroredRuleDeletesFilter(t *testing.T) {
	repo := mocks.NewMessageRepository(t)
	svc := newTestRulesService(t, repo)
	ctx := context.Background()
	filters := &fakeFilterAPI{nextID: "F1"}
	svc.filters = filters
	seedRule(t, svc, "from:medium.com", "archive", "", 0)
	rules, _ := svc.ListRules(ctx)
	if err := svc.SyncRule(ctx, rules[0].ID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := svc.DeleteRule(ctx, rules[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(filters.deleted) != 1 || filters.deleted[0] != "F1" {
		t.Fatalf("delete must remove the Gmail filter: %v", filters.deleted)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run 'TestFilterActionForRule|TestRulesServiceSync|TestRulesServiceDeleteMirrored' -v`
Expected: FAIL (`FilterActionForRule` undefined; sync stubs return "not implemented")

- [ ] **Step 3: Implement mapping + sync (replace the stubs)**

```go
// FilterActionForRule maps a rule action to the Gmail filter action applied to future
// incoming mail. labelID must be a resolved Gmail label ID when action == "label".
// "prompt" (and anything unknown) cannot run server-side.
func FilterActionForRule(action, labelID string) (*gmailapi.FilterAction, error) {
	switch action {
	case "archive":
		return &gmailapi.FilterAction{RemoveLabelIds: []string{"INBOX"}}, nil
	case "mark_read":
		return &gmailapi.FilterAction{RemoveLabelIds: []string{"UNREAD"}}, nil
	case "trash":
		return &gmailapi.FilterAction{AddLabelIds: []string{"TRASH"}}, nil
	case "label":
		if strings.TrimSpace(labelID) == "" {
			return nil, fmt.Errorf("label rule needs a label")
		}
		return &gmailapi.FilterAction{AddLabelIds: []string{labelID}}, nil
	default:
		return nil, fmt.Errorf("a %q rule cannot run as a Gmail filter", action)
	}
}

// resolveLabelID finds (case-insensitively) or creates the Gmail label and returns its ID.
func (s *DeterministicRulesServiceImpl) resolveLabelID(ctx context.Context, name string) (string, error) {
	if s.labels == nil {
		return "", fmt.Errorf("label service not available")
	}
	existing, err := s.labels.ListLabels(ctx)
	if err != nil {
		return "", err
	}
	for _, l := range existing {
		if l != nil && strings.EqualFold(l.Name, name) {
			return l.Id, nil
		}
	}
	created, err := s.labels.CreateLabel(ctx, name)
	if err != nil {
		return "", err
	}
	return created.Id, nil
}

// SyncRule mirrors the rule as a Gmail filter; if already mirrored, the old filter is
// deleted first (edit → recreate semantics).
func (s *DeterministicRulesServiceImpl) SyncRule(ctx context.Context, id int64) error {
	acct, err := s.account()
	if err != nil {
		return err
	}
	if s.filters == nil {
		return fmt.Errorf("Gmail client not available")
	}
	rule, err := s.findRule(ctx, id)
	if err != nil {
		return err
	}
	labelID := ""
	if rule.Action == "label" {
		if labelID, err = s.resolveLabelID(ctx, rule.Label); err != nil {
			return err
		}
	}
	action, err := FilterActionForRule(rule.Action, labelID)
	if err != nil {
		return err
	}
	if rule.GmailFilterID != "" {
		if err := s.filters.DeleteFilter(rule.GmailFilterID); err != nil {
			return fmt.Errorf("could not replace the existing Gmail filter: %w", err)
		}
	}
	filterID, err := s.filters.CreateFilter(rule.Query, action)
	if err != nil {
		return fmt.Errorf("Gmail did not accept the query as a filter: %w", err)
	}
	return s.store.SetGmailFilterID(ctx, acct, id, filterID)
}

// UnsyncRule deletes the mirrored Gmail filter and clears gmail_filter_id.
func (s *DeterministicRulesServiceImpl) UnsyncRule(ctx context.Context, id int64) error {
	acct, err := s.account()
	if err != nil {
		return err
	}
	rule, err := s.findRule(ctx, id)
	if err != nil {
		return err
	}
	if rule.GmailFilterID == "" {
		return nil // nothing mirrored — no-op
	}
	if s.filters == nil {
		return fmt.Errorf("Gmail client not available")
	}
	if err := s.filters.DeleteFilter(rule.GmailFilterID); err != nil {
		return fmt.Errorf("could not delete the Gmail filter: %w", err)
	}
	return s.store.SetGmailFilterID(ctx, acct, id, "")
}
```

Note `TestRulesServiceSyncUnsync` seeds a `label`-less archive rule, so `resolveLabelID` (which needs `s.labels`, nil in the harness) is never hit there. `mocks.LabelService` exists if a label-rule sync test is wanted later — YAGNI now.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run 'TestFilterActionForRule|TestRulesService' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/services/deterministic_rules_service.go internal/services/deterministic_rules_service_test.go
git add internal/services/deterministic_rules_service.go internal/services/deterministic_rules_service_test.go
git commit -m "feat(services): Gmail filter mirroring (sync/unsync) with action mapping"
```

---

### Task 6: OAuth scope gmail.settings.basic

**Files:**
- Modify: `cmd/giztui/main.go` (~line 341-347, the `auth.NewGmailService(...)` scope list)
- Modify: `internal/services/account_service.go` (~line 468-475, the `auth.NewGmailServiceWithAccount(...)` scope list)

- [ ] **Step 1: Add the scope to BOTH lists**

In each call, append after the existing five scopes (`gmail.readonly`, `gmail.send`, `gmail.modify`, `gmail.compose`, `calendar.events`):

```go
		"https://www.googleapis.com/auth/gmail.settings.basic",
```

Both lists must stay identical — a token minted by one path must work for the other.

- [ ] **Step 2: Verify**

Run: `grep -rn "gmail.settings.basic" cmd/ internal/ | wc -l`
Expected: `2`
Run: `go build ./...`
Expected: clean

Behavior note (documented in Task 15): existing tokens lack the scope; the first `SyncRule` fails with a Google `insufficient scopes` error, surfaced by the manager as a warning telling the user to re-authorize (remove the token file / re-run auth). New logins consent to the scope automatically.

- [ ] **Step 3: Commit**

```bash
git add cmd/giztui/main.go internal/services/account_service.go
git commit -m "feat(auth): request gmail.settings.basic for filter mirroring"
```

---

### Task 7: App wiring — service field, init, accessor, PickerRules

**Files:**
- Modify: `internal/tui/app.go`:
  - field next to `analyzerRulesService` (~line 171)
  - init block in `reinitializeServices` next to the analyzer-rules block (~lines 525-536)
  - account-switch propagation where `analyzerRulesService.SetAccountEmail` is called (~line 510)
  - accessor next to `GetAnalyzerRulesService()` (~line 1442)
  - `ActivePicker` enum (~lines 29-50)

- [ ] **Step 1: Add the field**

Next to `analyzerRulesService services.AnalyzerRulesService`:

```go
	deterministicRulesService services.DeterministicRulesService
```

- [ ] **Step 2: Add the init block**

Immediately after the existing analyzer-rules block in `reinitializeServices` (same nil-guard shape):

```go
	if a.dbStore != nil && a.deterministicRulesService == nil {
		rulesStore := db.NewDeterministicRulesStore(a.dbStore)
		_, _, labelService, _, repo, _, _, _, _, _, _, _ := a.GetServices()
		// a.Client may be nil in degraded startups — the service treats a nil filter API
		// as "sync unavailable" while CRUD and sweeps keep working.
		var filters services.GmailFilterAPI
		if a.Client != nil {
			filters = a.Client
		}
		svc := services.NewDeterministicRulesService(rulesStore, repo, labelService, filters)
		if email := a.getActiveAccountEmail(); email != "" {
			svc.SetAccountEmail(email)
		}
		a.deterministicRulesService = svc
	}
```

`GetServices()` tuple order (fixed, 12 values): EmailService, AIService, **LabelService(3rd)**, CacheService, **MessageRepository(5th)**, CompositionService, PromptService, ObsidianService, LinkService, GmailWebService, AttachmentService, DisplayService. If `reinitializeServices` builds these services locally instead of via `GetServices()` (check the surrounding code), source `repo`/`labelService` the same way the surrounding blocks do.

- [ ] **Step 3: Propagate account switches**

Find every call site of `a.analyzerRulesService.SetAccountEmail(...)` (or the `svc.SetAccountEmail(email)` pattern around app.go:510) and add the equivalent:

```go
	if a.deterministicRulesService != nil {
		a.deterministicRulesService.SetAccountEmail(email)
	}
```

- [ ] **Step 4: Accessor + picker constant**

Next to `GetAnalyzerRulesService`:

```go
// GetDeterministicRulesService returns the deterministic rules service (nil when the
// account has no DB — callers must nil-check, like GetAnalyzerRulesService).
func (a *App) GetDeterministicRulesService() services.DeterministicRulesService {
	return a.deterministicRulesService
}
```

In the `ActivePicker` enum block:

```go
	PickerRules ActivePicker = "rules"
```

- [ ] **Step 5: Build + run TUI tests**

Run: `go build ./... && go test ./internal/tui/ -count=1 | tail -5`
Expected: green

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/tui/app.go
git add internal/tui/app.go
git commit -m "feat(tui): wire DeterministicRulesService + PickerRules"
```

---

### Task 8: Config toggle `deterministic_prefilter`

**Files:**
- Modify: `internal/config/config.go` (struct ~:296-303, defaults ~:718-727)
- Test: `internal/config/config_test.go` (append)

The toggle gates the AI-plan prefilter (Task 11). Config self-migration is automatic in this codebase: a struct field + `DefaultConfig()` entry is enough (`defaultConfigMap` → `deepMergeMissing` surfaces new keys into existing `config.json` files on load). No migration code needed.

- [ ] **Step 1: Write the failing test**

Append to `internal/config/config_test.go`:

```go
func TestDefaultInboxAnalyzerConfigDeterministicPrefilter(t *testing.T) {
	if !DefaultInboxAnalyzerConfig().DeterministicPrefilter {
		t.Fatal("DeterministicPrefilter should default to true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestDefaultInboxAnalyzerConfigDeterministicPrefilter -v`
Expected: FAIL to compile — `DeterministicPrefilter` undefined

- [ ] **Step 3: Add the field and default**

In `internal/config/config.go`, `InboxAnalyzerConfig` struct — add after `StrictLabels`:

```go
	StrictLabels           bool `json:"strict_labels"`           // analyzer uses only existing labels; no creating new ones (default true)
	DeterministicPrefilter bool `json:"deterministic_prefilter"` // resolve deterministic rules before sending the rest to the LLM (default true)
```

In `DefaultInboxAnalyzerConfig()` — add after `StrictLabels: true,`:

```go
		StrictLabels:           true,
		DeterministicPrefilter: true,
```

(gofmt will realign the struct tags; run `gofmt -w internal/config/config.go`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -count=1`
Expected: PASS (whole package — existing default/migration tests must stay green)

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/config/config.go internal/config/config_test.go
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): deterministic_prefilter toggle (default true)"
```

---

### Task 9: Extract Action Plan panel helpers (refactor, no behavior change)

**Files:**
- Modify: `internal/tui/action_plan.go` (`openActionPlanWithText`, ~:227-313)

`:rules plan` (Task 10) reuses the Action Plan panel UI without the LLM. Extract two helpers from `openActionPlanWithText` so both entry points share panel construction and mounting. This is a pure refactor guarded by the existing action-plan tests — no new tests.

- [ ] **Step 1: Add the two helpers**

Add to `internal/tui/action_plan.go` (above `openActionPlanWithText`). The bodies are moved verbatim from `openActionPlanWithText` lines ~227-313, with two parameterizations: `analyzing` controls the initial title/placeholder, and the rest is unchanged:

```go
// buildActionPlanPanelState constructs the Action Plan panel widgets (tree, footer,
// container) and state, shared by the AI plan (openActionPlanWithText) and the
// deterministic rules plan (:rules plan). analyzing=true shows the "Analyzing…"
// placeholder + spinner title; false builds an idle panel the caller renders into.
func (a *App) buildActionPlanPanelState(customPromptText, scopeLabel string, metaByID map[string]*gmailapi.Message, analyzing bool) *actionPlanState {
	colors := a.GetComponentColors("ai")
	bg := colors.Background.Color()

	state := &actionPlanState{
		selectedCategory: 0,
		customPromptText: customPromptText,
		scopeLabel:       scopeLabel,
		excluded:         make(map[string]bool),
		expanded:         make(map[string]bool),
		metaByID:         metaByID,
	}
	state.analyzing.Store(analyzing)

	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root).SetCurrentNode(state.root)
	state.tree.SetTopLevel(1) // hide the empty root; categories are the visible top level
	state.tree.SetBackgroundColor(bg)
	state.tree.SetGraphics(true)
	state.tree.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		switch ref := node.GetReference().(type) {
		case int:
			state.selectedCategory = ref
			state.selectedMsgID = ""
		case emailRef:
			state.selectedCategory = ref.catIndex
			state.selectedMsgID = ref.msgID
		}
		a.updateActionPlanFooter(state)
		// tview postpones cursor movement to draw time (process()), and Flex defers the
		// FOCUSED item's Draw to last — so this callback runs AFTER the footer already
		// painted this frame, leaving it one keystroke behind the cursor. Force one more
		// repaint so the footer tracks the highlighted node live. (go avoids QueueUpdateDraw
		// deadlocking when invoked from the UI goroutine mid-draw.)
		go a.QueueUpdateDraw(func() {})
	})

	// Footer matches the other pickers: right-aligned, " X to Y | … " phrasing.
	state.footer = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight)
	state.footer.SetBackgroundColor(bg)
	state.footer.SetTextColor(colors.Text.Color())

	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.SetBackgroundColor(bg)
	state.container.SetBorder(true)
	// The status/summary lives in the border title (no separate header row).
	state.container.SetTitle(actionPlanTitleText(scopeLabel, 0, 0, 0, analyzing))
	state.container.SetTitleColor(colors.Title.Color())
	state.container.SetBorderColor(colors.Border.Color())
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)

	if analyzing {
		// Immediate "analyzing" feedback so the panel isn't blank before the first batch.
		state.root.AddChild(tview.NewTreeNode("⏳ Analyzing your messages…").
			SetSelectable(false).SetColor(colors.Text.Color()))
	}
	state.tree.SetInputCapture(a.actionPlanInputCapture(state))
	return state
}

// mountActionPlanPanel mounts the panel into the content split, focuses it and
// activates the picker. Must run on the UI thread; both entry points are invoked
// on background goroutines, so QueueUpdateDraw marshals AND forces a redraw
// (the same pattern openLinkPicker uses).
func (a *App) mountActionPlanPanel(state *actionPlanState) {
	a.QueueUpdateDraw(func() {
		a.actionPlanState = state
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.RemoveItem(a.labelsView)
			}
			a.labelsView = state.container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.SetFocus(state.tree)
		a.updateActionPlanFooter(state)
		a.markFocus("action_plan")
		a.setActivePicker(PickerActionPlan)
	})
}
```

- [ ] **Step 2: Replace the moved code in `openActionPlanWithText`**

Delete lines ~227-313 (from `colors := a.GetComponentColors("ai")` through the closing of the `a.QueueUpdateDraw(func() { ... })` mount block) and replace with:

```go
	// Build metaByID lookup for subject/from display in email child nodes.
	metaByID := make(map[string]*gmailapi.Message, len(metas))
	for _, m := range metas {
		if m != nil {
			metaByID[m.Id] = m
		}
	}

	state := a.buildActionPlanPanelState(customPromptText, scopeLabel, metaByID, true)
	a.mountActionPlanPanel(state)
```

(The `metaByID` loop stays where it is — only the widget construction and mount move into the helpers.)

- [ ] **Step 3: Build + run the action-plan tests**

Run: `go build ./... && go test ./internal/tui/ -run 'TestActionPlan|TestApplyActionPlan|TestMergeCategories' -count=1 -v | tail -20`
Expected: all PASS (pure refactor)

- [ ] **Step 4: Run the full TUI package**

Run: `go test ./internal/tui/ -count=1 | tail -3`
Expected: ok

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/action_plan.go
git add internal/tui/action_plan.go
git commit -m "refactor(tui): extract action plan panel build/mount helpers"
```

---

### Task 10: `:rules plan` — deterministic Action Plan panel (no LLM)

**Files:**
- Create: `internal/tui/rules_plan.go`
- Create: `internal/tui/rules_plan_test.go`
- Modify: `internal/tui/action_plan.go` (verb switches: `actionVerbLabel` ~:149, `actionKeyHint` ~:170, `actionRuleVerbShort` ~:657)

Reuses the panel helpers from Task 9. The plan is built purely from `Partition` matches — every category is a rule, marked `⚡`. Uses `services.SortCategories` (NOTE: renamed from `SortCategoriesByName` in the #54 merge — action-first, then name).

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/rules_plan_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestDeterministicRuleCategoryName(t *testing.T) {
	cases := []struct {
		name string
		rule services.DeterministicRuleInfo
		want string
	}{
		{"archive", services.DeterministicRuleInfo{Query: "from:foo@bar.com", Action: "archive"}, "⚡ Archive: from:foo@bar.com"},
		{"mark_read", services.DeterministicRuleInfo{Query: "list:news", Action: "mark_read"}, "⚡ Mark read: list:news"},
		{"trash", services.DeterministicRuleInfo{Query: "subject:spam", Action: "trash"}, "⚡ Trash: subject:spam"},
		{"label with name", services.DeterministicRuleInfo{Query: "from:bank", Action: "label", Label: "Finance"}, "⚡ Label Finance: from:bank"},
		{"prompt", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "⚡ Prompt: from:boss"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deterministicRuleCategoryName(c.rule); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestDeterministicRuleCategoryNameTruncatesLongQuery(t *testing.T) {
	long := strings.Repeat("á", 60) // runes, not bytes
	got := deterministicRuleCategoryName(services.DeterministicRuleInfo{Query: long, Action: "archive"})
	want := "⚡ Archive: " + strings.Repeat("á", 40) + "…"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildDeterministicPlan(t *testing.T) {
	matches := []services.RuleMatch{
		{Rule: services.DeterministicRuleInfo{Query: "from:z", Action: "trash"}, MessageIDs: []string{"m3"}},
		{Rule: services.DeterministicRuleInfo{Query: "from:a", Action: "archive"}, MessageIDs: []string{"m1", "m2"}},
		{Rule: services.DeterministicRuleInfo{Query: "from:none", Action: "archive"}, MessageIDs: nil}, // dropped
		{Rule: services.DeterministicRuleInfo{Query: "from:p", Action: "prompt", PromptID: 7}, MessageIDs: []string{"m4"}},
	}
	plan := buildDeterministicPlan(matches)
	if len(plan.Categories) != 3 {
		t.Fatalf("expected 3 categories (empty match dropped), got %d", len(plan.Categories))
	}
	for _, c := range plan.Categories {
		if c.Priority != "medium" {
			t.Fatalf("category %q priority = %q, want medium", c.Name, c.Priority)
		}
	}
	// SortCategories orders action-first, then name — just assert it ran (stable, deterministic).
	sorted := append([]services.ActionPlanCategory{}, plan.Categories...)
	services.SortCategories(sorted)
	for i := range sorted {
		if sorted[i].Name != plan.Categories[i].Name {
			t.Fatalf("categories not sorted: got %v", plan.Categories)
		}
	}
	// PromptID must survive into the category.
	found := false
	for _, c := range plan.Categories {
		if c.Action == "prompt" {
			found = true
			if c.PromptID != 7 {
				t.Fatalf("PromptID = %d, want 7", c.PromptID)
			}
		}
	}
	if !found {
		t.Fatal("prompt category missing")
	}
}

func TestActionVerbsForPrompt(t *testing.T) {
	if got := actionVerbLabel("prompt"); got != "Prompt" {
		t.Fatalf("actionVerbLabel(prompt) = %q, want Prompt", got)
	}
	if got := actionRuleVerbShort("prompt"); got != "prompt" {
		t.Fatalf("actionRuleVerbShort(prompt) = %q, want prompt", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run 'TestDeterministicRule|TestBuildDeterministicPlan|TestActionVerbsForPrompt' -v`
Expected: FAIL to compile — `deterministicRuleCategoryName` undefined

- [ ] **Step 3: Add the "prompt" verb cases in `action_plan.go`**

In `actionVerbLabel` (~:149), add before `default`:

```go
	case "prompt":
		return "Prompt"
```

In `actionKeyHint` (~:170), add before `default`:

```go
	case "prompt":
		return a.Keys.Summarize
```

In `actionRuleVerbShort` (~:657), add before `default`:

```go
	case "prompt":
		return "prompt"
```

- [ ] **Step 4: Create `internal/tui/rules_plan.go`**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	gmailapi "google.golang.org/api/gmail/v1"
)

// deterministicRuleCategoryName renders a rule as an Action Plan category header:
// "⚡ <verb>: <query>" with the query capped at 40 runes.
func deterministicRuleCategoryName(r services.DeterministicRuleInfo) string {
	verb := actionVerbLabel(r.Action)
	if r.Action == "label" && strings.TrimSpace(r.Label) != "" {
		verb = "Label " + r.Label
	}
	q := strings.TrimSpace(r.Query)
	if rq := []rune(q); len(rq) > 40 {
		q = string(rq[:40]) + "…"
	}
	return fmt.Sprintf("⚡ %s: %s", verb, q)
}

// buildDeterministicPlan converts rule matches into an ActionPlan without any LLM
// involvement. Rules with no matched messages are dropped; categories are sorted
// with the same criterion as AI plans (action first, then name).
func buildDeterministicPlan(matches []services.RuleMatch) *services.ActionPlan {
	cats := make([]services.ActionPlanCategory, 0, len(matches))
	for _, m := range matches {
		if len(m.MessageIDs) == 0 {
			continue
		}
		cats = append(cats, services.ActionPlanCategory{
			Name:       deterministicRuleCategoryName(m.Rule),
			Priority:   "medium",
			Action:     m.Rule.Action,
			Label:      m.Rule.Label,
			PromptID:   m.Rule.PromptID,
			MessageIDs: m.MessageIDs,
		})
	}
	services.SortCategories(cats)
	return &services.ActionPlan{Categories: cats}
}

// openDeterministicPlan opens the Action Plan panel populated purely by
// deterministic rules (":rules plan"). Runs on a background goroutine (network
// searches); mounting/rendering marshal onto the UI thread via the Task 9 helpers.
func (a *App) openDeterministicPlan() {
	svc := a.GetDeterministicRulesService()
	if svc == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}

	a.GetErrorHandler().ShowProgress(a.ctx, "Matching inbox against your rules…")
	matches, _, err := svc.Partition(a.ctx, "in:inbox", nil)
	a.GetErrorHandler().ClearPersistentMessage()
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Rules search failed — check connection")
		return
	}
	plan := buildDeterministicPlan(matches)
	if len(plan.Categories) == 0 {
		a.GetErrorHandler().ShowInfo(a.ctx, "No rules matched any inbox messages")
		return
	}

	total := 0
	for _, c := range plan.Categories {
		total += len(c.MessageIDs)
	}

	// metaByID from the in-memory list; rule searches can surface messages beyond the
	// loaded page, so fetch metadata for the gap (subjects/senders in the tree).
	a.mu.RLock()
	metaByID := make(map[string]*gmailapi.Message, len(a.messagesMeta))
	for _, m := range a.messagesMeta {
		if m != nil {
			metaByID[m.Id] = m
		}
	}
	a.mu.RUnlock()
	var missing []string
	for _, c := range plan.Categories {
		for _, id := range c.MessageIDs {
			if _, ok := metaByID[id]; !ok {
				missing = append(missing, id)
			}
		}
	}
	if len(missing) > 0 && a.Client != nil {
		if fetched, ferr := a.Client.GetMessagesMetadataParallel(missing, 5); ferr == nil {
			for _, m := range fetched {
				if m != nil {
					metaByID[m.Id] = m
				}
			}
		}
	}

	scopeLabel := fmt.Sprintf("⚡ %d by rules (no AI)", total)
	state := a.buildActionPlanPanelState("", scopeLabel, metaByID, false)
	state.plan = plan
	a.mountActionPlanPanel(state)
	a.QueueUpdateDraw(func() {
		if a.actionPlanState == state {
			a.renderActionPlanPanel(state)
		}
	})
}
```

Note: `openDeterministicPlan` is only ever invoked via `go a.openDeterministicPlan()` (Task 14), matching `go a.openActionPlanPanel()` — the direct `ShowProgress`/`ShowError` calls are safe off the key-handler path.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestDeterministicRule|TestBuildDeterministicPlan|TestActionVerbsForPrompt' -v`
Expected: PASS

- [ ] **Step 6: Full TUI package**

Run: `go test ./internal/tui/ -count=1 | tail -3`
Expected: ok

- [ ] **Step 7: Commit**

```bash
gofmt -w internal/tui/rules_plan.go internal/tui/rules_plan_test.go internal/tui/action_plan.go
git add internal/tui/rules_plan.go internal/tui/rules_plan_test.go internal/tui/action_plan.go
git commit -m "feat(tui): deterministic rules plan panel (:rules plan)"
```

---

### Task 11: Deterministic prefilter in the AI Action Plan

**Files:**
- Modify: `internal/tui/action_plan.go` (`openActionPlanWithText`, after the empty-messages check)
- Create: `internal/tui/action_plan_prefilter.go`
- Test: `internal/tui/action_plan_prefilter_test.go`

Config-gated (`inbox_analyzer.deterministic_prefilter`, Task 8). Rule-matched messages become `⚡` categories prepended to the AI plan; only the remainder goes to the LLM. If rules resolve everything, skip the LLM entirely.

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/action_plan_prefilter_test.go`:

```go
package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestApplyPrefilterToMessages(t *testing.T) {
	msgs := []services.AnalyzerMessage{{ID: "m1"}, {ID: "m2"}, {ID: "m3"}}
	out := applyPrefilterToMessages(msgs, []string{"m3", "m1"})
	if len(out) != 2 || out[0].ID != "m1" || out[1].ID != "m3" {
		t.Fatalf("expected [m1 m3] preserving input order, got %v", out)
	}
	if got := applyPrefilterToMessages(msgs, nil); len(got) != 0 {
		t.Fatalf("nil remaining should filter everything, got %v", got)
	}
}

func TestMergePreResolved(t *testing.T) {
	ai := &services.ActionPlan{Categories: []services.ActionPlanCategory{{Name: "AI cat"}}}
	pre := []services.ActionPlanCategory{{Name: "⚡ Archive: from:a"}}

	merged := mergePreResolved(ai, pre)
	if len(merged.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(merged.Categories))
	}
	if merged.Categories[0].Name != "⚡ Archive: from:a" {
		t.Fatalf("rule categories must come first, got %q", merged.Categories[0].Name)
	}
	// The AI snapshot must not be mutated (the analyzer reuses it across batches).
	if len(ai.Categories) != 1 {
		t.Fatalf("original plan mutated: %v", ai.Categories)
	}

	// No pre-resolved categories → same plan back, untouched.
	if got := mergePreResolved(ai, nil); got != ai {
		t.Fatal("empty preResolved should return the plan unchanged")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run 'TestApplyPrefilter|TestMergePreResolved' -v`
Expected: FAIL to compile — `applyPrefilterToMessages` undefined

- [ ] **Step 3: Create `internal/tui/action_plan_prefilter.go`**

```go
package tui

import "github.com/ajramos/giztui/internal/services"

// applyPrefilterToMessages keeps only the analyzer messages whose IDs are in
// remaining (the messages no deterministic rule claimed), preserving input order.
func applyPrefilterToMessages(messages []services.AnalyzerMessage, remaining []string) []services.AnalyzerMessage {
	keep := make(map[string]bool, len(remaining))
	for _, id := range remaining {
		keep[id] = true
	}
	out := make([]services.AnalyzerMessage, 0, len(remaining))
	for _, m := range messages {
		if keep[m.ID] {
			out = append(out, m)
		}
	}
	return out
}

// mergePreResolved prepends the deterministic rule categories to an AI plan
// snapshot without mutating it (the analyzer owns and reuses the snapshot
// across batch callbacks).
func mergePreResolved(p *services.ActionPlan, preResolved []services.ActionPlanCategory) *services.ActionPlan {
	if p == nil || len(preResolved) == 0 {
		return p
	}
	merged := *p
	merged.Categories = append(append([]services.ActionPlanCategory{}, preResolved...), p.Categories...)
	return &merged
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestApplyPrefilter|TestMergePreResolved' -v`
Expected: PASS

- [ ] **Step 5: Wire the prefilter into `openActionPlanWithText`**

In `internal/tui/action_plan.go`, immediately AFTER the `if len(messages) == 0 { ... return }` block and the `metaByID` build (metaByID must already exist so the all-matched shortcut can render subjects), insert:

```go
	// Deterministic prefilter: resolve rule-matched messages without the LLM,
	// send only the remainder to the analyzer. Config-gated; degrades to the
	// full AI path on any rules error.
	var preResolved []services.ActionPlanCategory
	if a.Config.InboxAnalyzer.DeterministicPrefilter {
		if svc := a.GetDeterministicRulesService(); svc != nil {
			scopeQuery := "in:inbox is:unread"
			if len(selected) > 0 {
				scopeQuery = "in:inbox" // selections can include read mail
			}
			candidates := make([]string, len(messages))
			for i := range messages {
				candidates[i] = messages[i].ID
			}
			matches, remaining, err := svc.Partition(a.ctx, scopeQuery, candidates)
			if err == nil && len(matches) > 0 {
				preResolved = buildDeterministicPlan(matches).Categories
				resolved := len(messages) - len(remaining)
				messages = applyPrefilterToMessages(messages, remaining)
				go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("⚡ %d resolved by rules · %d sent to AI", resolved, len(messages)))

				// Everything matched a rule: show the deterministic plan, skip the LLM.
				if len(messages) == 0 {
					plan := buildDeterministicPlan(matches)
					state := a.buildActionPlanPanelState(customPromptText, fmt.Sprintf("⚡ %d by rules (no AI)", resolved), metaByID, false)
					state.plan = plan
					a.mountActionPlanPanel(state)
					a.QueueUpdateDraw(func() {
						if a.actionPlanState == state {
							a.renderActionPlanPanel(state)
						}
					})
					return
				}
			}
		}
	}
```

NOTE for the implementer: after Task 9 the `metaByID` build sits between the empty-check and `state := a.buildActionPlanPanelState(...)`. This block goes between `metaByID` and that `state :=` line.

- [ ] **Step 6: Merge `preResolved` into every plan snapshot**

Still in `openActionPlanWithText`, in the per-batch progress callback, change:

```go
				a.QueueUpdateDraw(func() {
					if a.actionPlanState != state {
						return
					}
					state.plan = p
					a.renderActionPlanPanel(state)
				})
```

to:

```go
				a.QueueUpdateDraw(func() {
					if a.actionPlanState != state {
						return
					}
					state.plan = mergePreResolved(p, preResolved)
					a.renderActionPlanPanel(state)
				})
```

(The final render after `Analyze` returns re-renders `state.plan`, which already holds the merged snapshot — no second merge point needed.)

- [ ] **Step 7: Build + full TUI package**

Run: `go build ./... && go test ./internal/tui/ -count=1 | tail -3`
Expected: ok

- [ ] **Step 8: Commit**

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_prefilter.go internal/tui/action_plan_prefilter_test.go
git add internal/tui/action_plan.go internal/tui/action_plan_prefilter.go internal/tui/action_plan_prefilter_test.go
git commit -m "feat(tui): deterministic prefilter in AI action plan"
```

---

### Task 12: `:rules` manager panel (list + add/edit form + Gmail mirror toggle)

**Files:**
- Create: `internal/tui/rules_manager.go`
- Create: `internal/tui/rules_manager_test.go`

Mirrors `openAnalyzerRulesManager` (`internal/tui/action_plan_rules.go:76`) — in-place side panel mounted as `a.labelsView`, body-swap for add/edit, `a.cmd.focusOverride = "keep"`. The add/edit body-swap uses a `tview.Form` (Query, Action dropdown, Label, Prompt dropdown, "Also in Gmail" checkbox). Form values are captured via changed-callbacks into closure variables — do NOT use `GetFormItemByLabel` lookups.

**Spec deviation (deliberate):** the spec says the "Also in Gmail" toggle is *disabled* for prompt rules; dynamically disabling form items in the derailed/tview fork is version-dependent, so we implement the same safety as warn-and-ignore — a checked toggle on a prompt rule is ignored with a `ShowWarning` ("Prompt rules can't be mirrored to Gmail"). Same outcome: a prompt rule can never reach Gmail.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/rules_manager_test.go`:

```go
package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestDeterministicRuleListItem(t *testing.T) {
	cases := []struct {
		name       string
		rule       services.DeterministicRuleInfo
		promptName string
		want       string
	}{
		{"archive local", services.DeterministicRuleInfo{Query: "from:foo", Action: "archive"}, "", "⚡ Archive: from:foo"},
		{"archive mirrored", services.DeterministicRuleInfo{Query: "from:foo", Action: "archive", GmailFilterID: "flt1"}, "", "⚡ Archive: from:foo ☁"},
		{"label", services.DeterministicRuleInfo{Query: "from:bank", Action: "label", Label: "Finance"}, "", "⚡ Label Finance: from:bank"},
		{"prompt named", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "Daily digest", "⚡ Prompt 'Daily digest': from:boss"},
		{"prompt unnamed", services.DeterministicRuleInfo{Query: "from:boss", Action: "prompt", PromptID: 3}, "", "⚡ Prompt: from:boss"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deterministicRuleListItem(c.rule, c.promptName); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestDeterministicRuleListItem -v`
Expected: FAIL to compile — `deterministicRuleListItem` undefined

- [ ] **Step 3: Create `internal/tui/rules_manager.go`**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

const rulesManagerFooter = " a add · Enter edit · d delete · Esc close "

// deterministicRuleListItem renders one rule for the manager list:
// "⚡ <verb>: <query>" plus " ☁" when the rule is mirrored as a Gmail filter.
func deterministicRuleListItem(r services.DeterministicRuleInfo, promptName string) string {
	verb := actionVerbLabel(r.Action)
	switch {
	case r.Action == "label" && strings.TrimSpace(r.Label) != "":
		verb = "Label " + r.Label
	case r.Action == "prompt" && promptName != "":
		verb = "Prompt '" + promptName + "'"
	}
	item := fmt.Sprintf("⚡ %s: %s", verb, r.Query)
	if r.GmailFilterID != "" {
		item += " ☁"
	}
	return item
}

// openRulesManager shows the deterministic rules as an in-place side-panel picker
// (the openAnalyzerRulesManager pattern). 'a' adds, Enter edits, 'd' deletes, Esc
// closes. Add/edit body-swap the list for a form inside the same container.
func (a *App) openRulesManager() {
	svc := a.GetDeterministicRulesService()
	if svc == nil {
		go a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}
	colors := a.GetComponentColors("ai")

	// Prompt names for list display and the form dropdown ("" category = all).
	_, _, _, _, _, _, promptSvc, _, _, _, _, _ := a.GetServices()
	promptNameByID := map[int64]string{}
	promptNames := []string{"(none)"}
	promptIDs := []int64{0}
	if promptSvc != nil {
		if pts, err := promptSvc.ListPrompts(a.ctx, ""); err == nil {
			for _, p := range pts {
				promptNameByID[int64(p.ID)] = p.Name
				promptNames = append(promptNames, p.Name)
				promptIDs = append(promptIDs, int64(p.ID))
			}
		}
	}

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(colors.Background.Color())
	container.SetBorder(true)
	container.SetTitle(" ⚡ Deterministic rules ")
	container.SetTitleColor(colors.Title.Color())
	container.SetBorderColor(colors.Border.Color())

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	footer.SetText(rulesManagerFooter)

	var rules []services.DeterministicRuleInfo
	reload := func() {
		list.Clear()
		rs, err := svc.ListRules(a.ctx)
		if err != nil {
			go a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("List rules failed: %v", err))
			return
		}
		rules = rs
		if len(rs) == 0 {
			list.AddItem("(no rules yet — press 'a' to add)", "", 0, nil)
			return
		}
		for _, r := range rs {
			list.AddItem(deterministicRuleListItem(r, promptNameByID[r.PromptID]), "", 0, nil)
		}
	}
	reload()

	container.AddItem(list, 0, 1, true)
	container.AddItem(footer, 1, 0, false)

	closePicker := func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.ResizeItem(a.labelsView, 0, 0)
			}
		}
		a.setActivePicker(PickerNone)
		if l, ok := a.views["list"].(*tview.Table); ok {
			a.SetFocus(l)
		}
		a.markFocus("list")
	}

	// showRuleForm body-swaps the list for an add/edit form. existing == nil → new rule.
	showRuleForm := func(existing *services.DeterministicRuleInfo) {
		actionsDisplay := []string{"Archive", "Mark read", "Trash", "Label", "Prompt"}
		actionTokens := []string{"archive", "mark_read", "trash", "label", "prompt"}

		// Seed values from the rule under edit (defaults for a new rule).
		queryText, labelText := "", ""
		actionIdx, promptIdx := 0, 0
		mirrored := false
		if existing != nil {
			queryText, labelText = existing.Query, existing.Label
			for i, tok := range actionTokens {
				if tok == existing.Action {
					actionIdx = i
				}
			}
			for i, pid := range promptIDs {
				if pid != 0 && pid == existing.PromptID {
					promptIdx = i
				}
			}
			mirrored = existing.GmailFilterID != ""
		}

		// Live form values, captured via changed-callbacks (no form-item lookups).
		action := actionTokens[actionIdx]
		promptID := promptIDs[promptIdx]
		mirror := mirrored

		form := tview.NewForm()
		form.SetBackgroundColor(colors.Background.Color())
		form.SetFieldBackgroundColor(colors.Background.Color())
		form.SetFieldTextColor(colors.Text.Color())
		form.SetLabelColor(colors.Title.Color())
		form.SetButtonBackgroundColor(colors.Background.Color())
		form.SetButtonTextColor(colors.Text.Color())
		form.AddInputField("Query", queryText, 0, nil, func(text string) { queryText = text })
		form.AddDropDown("Action", actionsDisplay, actionIdx, func(_ string, idx int) {
			if idx >= 0 && idx < len(actionTokens) {
				action = actionTokens[idx]
			}
		})
		form.AddInputField("Label (for Label action)", labelText, 0, nil, func(text string) { labelText = text })
		form.AddDropDown("Prompt (for Prompt action)", promptNames, promptIdx, func(_ string, idx int) {
			if idx >= 0 && idx < len(promptIDs) {
				promptID = promptIDs[idx]
			}
		})
		form.AddCheckbox("Also in Gmail", mirrored, func(checked bool) { mirror = checked })

		restore := func() {
			container.RemoveItem(form)
			container.RemoveItem(footer)
			container.AddItem(list, 0, 1, true)
			container.AddItem(footer, 1, 0, false)
			container.SetTitle(" ⚡ Deterministic rules ")
			footer.SetText(rulesManagerFooter)
			a.focus.set("rules_manager")
			a.SetFocus(list)
		}

		save := func() {
			q := strings.TrimSpace(queryText)
			lbl := strings.TrimSpace(labelText)
			act := action
			pid := promptID
			if act != "prompt" {
				pid = 0
			}
			mir := mirror
			var existingID int64
			hadFilter := false
			if existing != nil {
				existingID = existing.ID
				hadFilter = existing.GmailFilterID != ""
			}
			restore()
			go func() {
				var (
					id  int64
					err error
				)
				if existing == nil {
					var saved *services.DeterministicRuleInfo
					saved, err = svc.SaveRule(a.ctx, q, act, lbl, pid)
					if saved != nil {
						id = saved.ID
					}
				} else {
					id = existingID
					err = svc.UpdateRule(a.ctx, id, q, act, lbl, pid)
				}
				if err != nil {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
					return
				}
				// Gmail mirroring. Prompt rules cannot exist as Gmail filters.
				warned := false
				switch {
				case mir && act == "prompt":
					a.GetErrorHandler().ShowWarning(a.ctx, "Prompt rules can't be mirrored to Gmail — saved locally only")
					warned = true
				case mir:
					if serr := svc.SyncRule(a.ctx, id); serr != nil {
						a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Rule saved locally, but Gmail did not accept it as a filter: %v", serr))
						warned = true
					}
				case hadFilter:
					if serr := svc.UnsyncRule(a.ctx, id); serr != nil {
						a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Rule saved, but the Gmail filter could not be removed: %v", serr))
						warned = true
					}
				}
				a.QueueUpdateDraw(reload)
				if !warned {
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved")
				}
			}()
		}

		form.AddButton("Save", save)
		form.AddButton("Cancel", restore)
		form.SetCancelFunc(restore) // Esc anywhere in the form

		container.RemoveItem(list)
		container.RemoveItem(footer)
		container.AddItem(form, 0, 1, true)
		container.AddItem(footer, 1, 0, false)
		if existing == nil {
			container.SetTitle(" ⚡ New rule ")
		} else {
			container.SetTitle(" ⚡ Edit rule ")
		}
		footer.SetText(" Tab fields · Enter/Save save · Esc cancel ")
		a.focus.set("rules_manager_form")
		a.SetFocus(form)
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx >= 0 && idx < len(rules) {
			r := rules[idx]
			showRuleForm(&r)
		}
	})
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEscape:
			closePicker()
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RuleAdd):
			showRuleForm(nil)
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RuleDelete):
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(rules) {
				id := rules[idx].ID
				go func() {
					// DeleteRule also removes the mirrored Gmail filter (service layer).
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

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.setActivePicker(PickerRules)
	a.focus.set("rules_manager")
	a.SetFocus(list)
	// :rules runs during command execution; hideCommandBar()'s restoreFocusAfterModal()
	// would otherwise re-focus the message list afterward. "keep" leaves our focus alone.
	a.cmd.focusOverride = "keep"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestDeterministicRuleListItem -v`
Expected: PASS

- [ ] **Step 5: Build + full TUI package**

Run: `go build ./... && go test ./internal/tui/ -count=1 | tail -3`
Expected: ok

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/tui/rules_manager.go internal/tui/rules_manager_test.go
git add internal/tui/rules_manager.go internal/tui/rules_manager_test.go
git commit -m "feat(tui): deterministic rules manager panel (:rules)"
```

---

### Task 13: "Prompt" action dispatch in the Action Plan panel

**Files:**
- Create: `internal/tui/action_plan_prompt_action.go`
- Test: `internal/tui/action_plan_prompt_action_test.go`
- Modify: `internal/tui/action_plan.go` (`Keys.Summarize` case in `actionPlanInputCapture`, ~:871)
- Test (regression): append to `internal/tui/action_plan_apply_test.go`

Prompt-rule categories run their saved prompt over the checked emails, in-place (the `dispatchActionPlanSummarize` body-swap pattern). Whole-plan confirm (#54) must keep SKIPPING prompt categories — `buildPlanApply`'s default case already does this; pin it with a regression test.

`PromptService.ApplyBulkPrompt(ctx, accountEmail string, messageIDs []string, promptID int, variables map[string]string) (*BulkPromptResult, error)` — note `promptID` is **int**, the category carries **int64**: cast with `int(cat.PromptID)`.

- [ ] **Step 1: Write the failing regression test (whole-plan confirm skips prompt)**

Append to `internal/tui/action_plan_apply_test.go`:

```go
func TestBuildPlanApplySkipsPromptCategories(t *testing.T) {
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "⚡ Prompt: from:boss", Action: "prompt", PromptID: 7, MessageIDs: []string{"p1", "p2"}},
		{Name: "⚡ Archive: from:news", Action: "archive", MessageIDs: []string{"a1", "a2"}},
	}}
	s := buildPlanApply(plan, nil)
	if s.total != 2 {
		t.Fatalf("prompt messages must not count toward whole-plan apply: total=%d want 2", s.total)
	}
	if len(s.items) != 1 {
		t.Fatalf("expected only the archive item, got %d items", len(s.items))
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/tui/ -run TestBuildPlanApplySkipsPromptCategories -v`
Expected: FAIL to compile until Task 2's `PromptID` field lands; once it compiles it should PASS immediately (the `default:` case in `buildPlanApply` already skips unknown actions). If it FAILS with wrong totals, `buildPlanApply` changed — fix by skipping `case "prompt"` explicitly.

- [ ] **Step 3: Write the failing dispatch test**

Create `internal/tui/action_plan_prompt_action_test.go` (modeled on `TestDispatchActionPlanSummarize`, `action_plan_summarize_test.go`):

```go
package tui

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// The prompt dispatcher must body-swap the tree for the result view and restore on Esc,
// exactly like the summarize dispatcher.
func TestDispatchActionPlanPromptSwapAndEsc(t *testing.T) {
	a := newTestAppForActionPlan(t) // reuse the same constructor the summarize test uses
	a.ctx = context.Background()
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "⚡ Prompt: from:boss", Action: "prompt", PromptID: 5, MessageIDs: []string{"m1"}},
		}},
		selectedCategory: 0,
		excluded:         map[string]bool{},
		expanded:         map[string]bool{},
		metaByID:         map[string]*gmailapi.Message{"m1": {Id: "m1", Snippet: "s"}},
		footer:           tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.dispatchActionPlanPrompt(state)
	if a.focus.cur() != "action_plan_prompt_run" {
		t.Fatalf("expected currentFocus=action_plan_prompt_run, got %q", a.focus.cur())
	}
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out for the result view")
	}
	view, ok := a.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected the result TextView focused, got %T", a.GetFocus())
	}
	if cap := view.GetInputCapture(); cap != nil {
		cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.focus.cur() != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.focus.cur())
	}
	if state.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc the tree should be restored")
	}
}
```

NOTE for the implementer: `newTestAppForActionPlan` is whatever helper `TestDispatchActionPlanSummarize` in `action_plan_summarize_test.go` uses to build its `*App` — copy that setup verbatim (it constructs the App with `views`, `focus`, etc.). If it's inline rather than a helper, inline the same lines here.

- [ ] **Step 4: Run it**

Run: `go test ./internal/tui/ -run TestDispatchActionPlanPromptSwapAndEsc -v`
Expected: FAIL to compile — `dispatchActionPlanPrompt` undefined

- [ ] **Step 5: Create `internal/tui/action_plan_prompt_action.go`**

```go
package tui

import (
	"fmt"

	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// dispatchActionPlanPrompt runs the category's saved prompt (rule action "prompt")
// over the checked emails and shows the result in-place — the same body-swap
// pattern as dispatchActionPlanSummarize. Esc returns to the tree.
func (a *App) dispatchActionPlanPrompt(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil || cat.Action != "prompt" {
		return
	}
	if cat.PromptID == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "This rule has no prompt attached — edit it in :rules")
		return
	}
	ids := checkedIDs(cat.MessageIDs, state.excluded)
	if len(ids) == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "All emails in this category are excluded — nothing to run")
		return
	}

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(true)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(fmt.Sprintf("⏳ Running prompt on %d email(s)…", len(ids)))

	restore := func() {
		state.container.RemoveItem(view)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.focus.set("action_plan")
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state)
	}
	view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(view, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(fmt.Sprintf(" 🧠 Prompt on %q ", cat.Name))
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.focus.set("action_plan_prompt_run")
	a.SetFocus(view)

	promptID := int(cat.PromptID) // ApplyBulkPrompt takes int; rules store int64
	_, _, _, _, _, _, promptSvc, _, _, _, _, _ := a.GetServices()
	go func() {
		if promptSvc == nil {
			a.QueueUpdateDraw(func() {
				if a.actionPlanState == state {
					view.SetText("⚠️ Prompt service not available")
				}
			})
			return
		}
		res, err := promptSvc.ApplyBulkPrompt(a.ctx, a.getActiveAccountEmail(), ids, promptID, map[string]string{})
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state || a.ctx.Err() != nil {
				return
			}
			if err != nil {
				view.SetText(fmt.Sprintf("⚠️ Prompt failed: %v", err))
				return
			}
			view.SetText(a.renderPromptResult(res.Summary))
		})
	}()
}
```

- [ ] **Step 6: Route the Summarize key by category action**

In `internal/tui/action_plan.go`, `actionPlanInputCapture`, change:

```go
		case a.Keys.Summarize:
			a.dispatchActionPlanSummarize(state)
			return nil
```

to:

```go
		case a.Keys.Summarize:
			if cat := a.currentActionPlanCategory(state); cat != nil && cat.Action == "prompt" {
				a.dispatchActionPlanPrompt(state)
				return nil
			}
			a.dispatchActionPlanSummarize(state)
			return nil
```

- [ ] **Step 7: Run the new tests + full TUI package**

Run: `go test ./internal/tui/ -run 'TestDispatchActionPlanPrompt|TestBuildPlanApplySkips' -v && go test ./internal/tui/ -count=1 | tail -3`
Expected: PASS / ok

- [ ] **Step 8: Commit**

```bash
gofmt -w internal/tui/action_plan_prompt_action.go internal/tui/action_plan_prompt_action_test.go internal/tui/action_plan.go internal/tui/action_plan_apply_test.go
git add internal/tui/action_plan_prompt_action.go internal/tui/action_plan_prompt_action_test.go internal/tui/action_plan.go internal/tui/action_plan_apply_test.go
git commit -m "feat(tui): run prompt rules from the action plan panel"
```

---

### Task 14: Command parity — `:rules` / `:ru` + in-app help

**Files:**
- Modify: `internal/tui/commands.go` (dispatch switch ~:412, new `executeRulesCommand`)
- Modify: `internal/tui/command_completion.go` (registry entry after `action-plan` ~:182, `completeRulesArg` after `completeActionPlanArg` ~:385)
- Modify: `internal/tui/app.go` (help text, after the `:plan apply` line ~:2332)
- Test: `internal/tui/help_text_test.go` (extend want-strings)

The feature is command-first (like `:accounts`): no default keyboard shortcut. `sync`/`unsync` take a **NUMBER** — the 1-based position in the `:rules` list (per the command-grammar convention: `:slack`/`:label`/`:move` also take numbers).

- [ ] **Step 1: Extend the failing help test**

In `internal/tui/help_text_test.go`, add to the `want` slice:

```go
		":rules",
		"Deterministic rules",
```

Run: `go test ./internal/tui/ -run TestGenerateHelpText -v`
Expected: FAIL — help is missing ":rules"

- [ ] **Step 2: Add the command dispatch**

In `internal/tui/commands.go`, after the `case "action-plan", "plan", "ap":` block (~:412):

```go
	case "rules", "ru":
		a.executeRulesCommand(args)
```

And add the handler (near `executeActionPlanCommand`, ~:2195):

```go
// executeRulesCommand handles :rules / :ru [plan|sync <n>|unsync <n>].
// <n> is the 1-based position in the :rules list (creation order).
func (a *App) executeRulesCommand(args []string) {
	if len(args) == 0 {
		a.openRulesManager()
		return
	}
	svc := a.GetDeterministicRulesService()
	if svc == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "plan":
		go a.openDeterministicPlan()
	case "sync", "unsync":
		if len(args) < 2 {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Usage: :rules %s <number> (position in the :rules list)", sub))
			return
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Usage: :rules %s <number> (position in the :rules list)", sub))
			return
		}
		go func() {
			rules, lerr := svc.ListRules(a.ctx)
			if lerr != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("List rules failed: %v", lerr))
				return
			}
			if n > len(rules) {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Rule %d not found — :rules lists %d rule(s)", n, len(rules)))
				return
			}
			r := rules[n-1]
			if sub == "sync" {
				if r.Action == "prompt" {
					a.GetErrorHandler().ShowWarning(a.ctx, "Prompt rules can't be mirrored to Gmail")
					return
				}
				if serr := svc.SyncRule(a.ctx, r.ID); serr != nil {
					a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Gmail did not accept the filter: %v", serr))
					return
				}
				a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule mirrored to Gmail")
				return
			}
			if serr := svc.UnsyncRule(a.ctx, r.ID); serr != nil {
				a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Could not remove the Gmail filter: %v", serr))
				return
			}
			a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Gmail filter removed — rule kept locally")
		}()
	default:
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Unknown rules option: %s", args[0]))
	}
}
```

(`strconv` may need adding to the imports of `commands.go` — check; other handlers already parse numbers, so it is likely present.)

- [ ] **Step 3: Register the command + completion**

In `internal/tui/command_completion.go`, after the `action-plan` registry entry (~:186):

```go
	{name: "rules", aliases: []string{"ru"}, completeArg: completeRulesArg, help: &cmdHelp{
		summary:  "Deterministic rules (no AI): manage, preview as a plan, mirror to Gmail.",
		syntax:   ":rules [plan|sync <n>|unsync <n>]",
		examples: []string{":rules", ":rules plan", ":rules sync 2", ":rules unsync 2"},
	}},
```

And after `completeActionPlanArg` (~:385):

```go
// completeRulesArg: ':rules <subcommand>'. First token → plan/sync/unsync.
func completeRulesArg(a *App, rest string) []string {
	head, prefix := splitLastToken(rest)
	if head != "" {
		return nil
	}
	return withHead("", filterByPrefix([]string{"plan", "sync", "unsync"}, prefix))
}
```

- [ ] **Step 4: Add the in-app help lines**

In `internal/tui/app.go`, right after the `:plan apply` help line (~:2332):

```go
	fmt.Fprintf(&help, "    %-18s ⚡  Deterministic rules manager (alias :ru; ☁ = also in Gmail)\n", ":rules")
	fmt.Fprintf(&help, "    %-18s ⚡  Preview what your rules match — no AI involved\n", ":rules plan")
	fmt.Fprintf(&help, "    %-18s ⚡  Mirror rule <n> to Gmail / remove the mirror\n", ":rules sync <n>")
	fmt.Fprintf(&help, "    %-18s ⚡  Rules pre-filter the AI :plan (config: inbox_analyzer.deterministic_prefilter)\n", "")
```

- [ ] **Step 5: Run the help test + full TUI package**

Run: `go test ./internal/tui/ -run TestGenerateHelpText -v && go test ./internal/tui/ -count=1 | tail -3`
Expected: PASS / ok

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/tui/commands.go internal/tui/command_completion.go internal/tui/app.go internal/tui/help_text_test.go
git add internal/tui/commands.go internal/tui/command_completion.go internal/tui/app.go internal/tui/help_text_test.go
git commit -m "feat(tui): :rules command family + in-app help"
```

---

### Task 15: Documentation

**Files:**
- Modify: `docs/KEYBOARD_SHORTCUTS.md` (commands table)
- Modify: `README.md` (features section)

- [ ] **Step 1: KEYBOARD_SHORTCUTS.md**

Find the commands table that documents `:action-plan` and add rows below it (match the table's exact column format — read the surrounding rows first):

```markdown
| `:rules` | `:ru` | Deterministic rules manager (⚡ rules run without AI; ☁ = also mirrored as a Gmail filter) |
| `:rules plan` | `:ru plan` | Preview what your rules match as an Action Plan — no AI involved |
| `:rules sync <n>` / `:rules unsync <n>` | — | Mirror rule *n* to Gmail / remove the mirror (*n* = position in `:rules`) |
```

Also, in the document's Action Plan section, add one sentence:

```markdown
Categories marked ⚡ were resolved by your deterministic rules (no AI); with `inbox_analyzer.deterministic_prefilter` enabled (default), the AI plan resolves rule matches first and only sends the remainder to the LLM.
```

- [ ] **Step 2: README.md**

In the features/AI section (near the Inbox Action Plan description), add:

```markdown
### ⚡ Deterministic rules (no AI)

Define Gmail-query rules (`from:newsletter@x.com` → archive / mark read / trash / label / run a saved prompt) that resolve mail *deterministically* — first match wins, in creation order:

- `:rules` — manage rules; the "Also in Gmail" toggle mirrors a rule as a real Gmail filter (☁) so it also applies server-side
- `:rules plan` — preview what your rules match, in the Action Plan panel, without any AI call
- With `inbox_analyzer.deterministic_prefilter` (default on), the AI Action Plan first resolves rule-matched mail (⚡ categories) and only sends the remainder to the LLM — fewer tokens, faster plans

> **Note:** mirroring rules to Gmail needs the `gmail.settings.basic` OAuth scope, added in this version. Existing installs must re-authenticate once (delete the token file, e.g. `~/.config/giztui/token.json`, and restart) before `:rules sync` works. Everything else keeps working with the old token.
```

- [ ] **Step 3: Commit**

```bash
git add docs/KEYBOARD_SHORTCUTS.md README.md
git commit -m "docs: deterministic rules (:rules) — shortcuts + README"
```

---

### Task 16: Final quality gates + live smoke test

**Files:** none (verification only)

- [ ] **Step 1: Format + CI-equivalent local check**

```bash
make pre-commit-check
```
Expected: fmt + vet + golangci-lint + essential tests all green.

- [ ] **Step 2: Full test suite (SEPARATE step — required before any merge)**

```bash
make test
```
Expected: green, including the `test/helpers` goroutine-leak detector. If the leak detector fires: check that no `go ShowInfo(...)`/`QueueUpdateDraw` was added to `initServices`/`reinitializeServices` (run-path UI actions belong in `App.Run()`).

- [ ] **Step 3: Update the knowledge graph**

```bash
graphify update .
```

- [ ] **Step 4: Live smoke test (test Gmail account on this laptop)**

Using the tatto2k@gmail.com test account and the pty driver (`/tmp/giztui_smoke.py`, needs TIOCSWINSZ):

1. `make build`, launch, wait for inbox load; `grep -i panic ~/.local/share/giztui/giztui.log` (or the configured log path) — none expected.
2. `:rules` → panel opens with "(no rules yet…)"; `a` → form; save an `archive` rule with query `from:notifications@github.com` (any sender that exists in the test inbox), "Also in Gmail" OFF.
3. `:rules plan` → panel shows a `⚡ Archive: …` category with matching emails; Esc closes.
4. `:plan` (AI) with `deterministic_prefilter` on → status shows `⚡ N resolved by rules · M sent to AI`; the ⚡ category appears above AI categories.
5. Edit the rule in `:rules` (Enter), toggle "Also in Gmail" ON → expect success; verify in Gmail web UI (Settings → Filters) that the filter exists; toggle OFF → filter disappears. NOTE: requires re-auth for the new `gmail.settings.basic` scope first (delete token, re-run OAuth flow) — expect a clear warning, not a crash, with the old token.
6. `d` deletes the rule; `:rules plan` reports "No rules matched any inbox messages".
7. Final `grep -i panic` on the log.

- [ ] **Step 5: Verify branch state**

```bash
git log --oneline main..feat/deterministic-rules
git status
```
Expected: one commit per task, clean tree. Do NOT merge or push — pushing for Mac smoke-testing and merging are separate, user-authorized steps.

---

## Out of scope (do NOT implement)

- Rule promotion from AI/analyzer preferences
- "Read manually" UX iteration (separate, deferred)
- Whole-plan undo
- Importing existing Gmail filters as rules
