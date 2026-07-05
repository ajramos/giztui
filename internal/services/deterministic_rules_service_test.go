package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/db"
	internalgmail "github.com/ajramos/giztui/internal/gmail"
	gmailapi "google.golang.org/api/gmail/v1"
)

// stubMessageRepo is a minimal MessageRepository for testing the rules service.
// Only SearchMessages is exercised; all other methods panic if called unexpectedly.
type stubMessageRepo struct {
	// searchFn is called for each SearchMessages invocation in order.
	calls []func(query string, opts QueryOptions) (*MessagePage, error)
	idx   int
}

func (r *stubMessageRepo) SearchMessages(_ context.Context, query string, opts QueryOptions) (*MessagePage, error) {
	if r.idx >= len(r.calls) {
		panic(fmt.Sprintf("unexpected SearchMessages call #%d: query=%q", r.idx+1, query))
	}
	fn := r.calls[r.idx]
	r.idx++
	return fn(query, opts)
}

func (r *stubMessageRepo) GetMessages(_ context.Context, _ QueryOptions) (*MessagePage, error) {
	panic("GetMessages not expected in rules service tests")
}
func (r *stubMessageRepo) GetMessage(_ context.Context, _ string) (*internalgmail.Message, error) {
	panic("GetMessage not expected in rules service tests")
}
func (r *stubMessageRepo) UpdateMessage(_ context.Context, _ string, _ MessageUpdates) error {
	panic("UpdateMessage not expected in rules service tests")
}
func (r *stubMessageRepo) GetDrafts(_ context.Context, _ int64) ([]*gmailapi.Draft, error) {
	panic("GetDrafts not expected in rules service tests")
}
func (r *stubMessageRepo) GetDraft(_ context.Context, _ string) (*gmailapi.Draft, error) {
	panic("GetDraft not expected in rules service tests")
}

// page builds a MessagePage of ids for the stub repository.
func page(next string, ids ...string) *MessagePage {
	p := &MessagePage{NextPageToken: next}
	for _, id := range ids {
		p.Messages = append(p.Messages, &gmailapi.Message{Id: id})
	}
	return p
}

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

func TestRulesServiceSaveValidatesQueryAgainstGmail(t *testing.T) {
	repo := &stubMessageRepo{
		calls: []func(string, QueryOptions) (*MessagePage, error){
			// Valid query: SearchMessages(MaxResults:1) succeeds → rule saved.
			func(query string, opts QueryOptions) (*MessagePage, error) {
				if query != "from:medium.com" || opts.MaxResults != 1 {
					t.Errorf("unexpected search call: query=%q opts=%+v", query, opts)
				}
				return page(""), nil
			},
			// Gmail rejects the query → save fails, nothing persisted.
			func(query string, opts QueryOptions) (*MessagePage, error) {
				if query != "from:(broken" || opts.MaxResults != 1 {
					t.Errorf("unexpected search call: query=%q opts=%+v", query, opts)
				}
				return nil, fmt.Errorf("400 invalid query")
			},
		},
	}
	svc := newTestRulesService(t, repo)
	ctx := context.Background()

	r, err := svc.SaveRule(ctx, "from:medium.com", "archive", "", 0)
	if err != nil || r == nil || r.ID <= 0 {
		t.Fatalf("save valid rule: %v %+v", err, r)
	}

	if _, err := svc.SaveRule(ctx, "from:(broken", "archive", "", 0); err == nil {
		t.Fatal("invalid query must fail")
	}
	rules, _ := svc.ListRules(ctx)
	if len(rules) != 1 {
		t.Fatalf("rejected rule must not persist, got %+v", rules)
	}
}

func TestRulesServiceFieldValidation(t *testing.T) {
	// None of these must reach Gmail (field validation happens first).
	repo := &stubMessageRepo{}
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
	if repo.idx != 0 {
		t.Fatalf("field validation must not reach Gmail, but SearchMessages was called %d time(s)", repo.idx)
	}
}

func TestRulesServiceNoAccount(t *testing.T) {
	repo := &stubMessageRepo{}
	svc := newTestRulesService(t, repo)
	svc.SetAccountEmail("")
	if _, err := svc.ListRules(context.Background()); err == nil {
		t.Fatal("no account must fail, not panic")
	}
}

// stubFailingFilter implements GmailFilterAPI; CreateFilter succeeds (returns a
// fixed filter ID) and DeleteFilter always returns an error.
type stubFailingFilter struct{}

func (f *stubFailingFilter) CreateFilter(_ string, _ *gmailapi.FilterAction) (string, error) {
	return "filter-abc", nil
}
func (f *stubFailingFilter) DeleteFilter(_ string) error {
	return fmt.Errorf("remote: permission denied")
}

// seedRule inserts a rule bypassing Gmail validation (direct store write) so Partition
// tests don't need SearchMessages expectations for the save path.
func seedRule(t *testing.T, svc *DeterministicRulesServiceImpl, query, action, label string, promptID int64) {
	t.Helper()
	if _, err := svc.store.SaveRule(context.Background(), "user@example.com", query, action, label, promptID); err != nil {
		t.Fatalf("seed rule: %v", err)
	}
}

func TestRulesServicePartitionFirstMatchWins(t *testing.T) {
	// SearchMessages expectations (in order):
	//   1. "in:inbox from:medium.com" {MaxResults:100} → page("", "m1", "m2")
	//   2. "in:inbox is:important"    {MaxResults:100} → page("", "m2", "m3")
	repo := &stubMessageRepo{
		calls: []func(string, QueryOptions) (*MessagePage, error){
			func(query string, opts QueryOptions) (*MessagePage, error) {
				if query != "in:inbox from:medium.com" || opts.MaxResults != 100 || opts.PageToken != "" {
					t.Errorf("call 1: unexpected query=%q opts=%+v", query, opts)
				}
				return page("", "m1", "m2"), nil
			},
			func(query string, opts QueryOptions) (*MessagePage, error) {
				if query != "in:inbox is:important" || opts.MaxResults != 100 || opts.PageToken != "" {
					t.Errorf("call 2: unexpected query=%q opts=%+v", query, opts)
				}
				return page("", "m2", "m3"), nil
			},
		},
	}
	svc := newTestRulesService(t, repo)
	ctx := context.Background()
	seedRule(t, svc, "from:medium.com", "archive", "", 0)
	seedRule(t, svc, "is:important", "mark_read", "", 0)

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
	// SearchMessages expectation:
	//   1. "in:inbox is:unread from:medium.com" {MaxResults:100} → page("", "m1", "m9")
	repo := &stubMessageRepo{
		calls: []func(string, QueryOptions) (*MessagePage, error){
			func(query string, opts QueryOptions) (*MessagePage, error) {
				if query != "in:inbox is:unread from:medium.com" || opts.MaxResults != 100 || opts.PageToken != "" {
					t.Errorf("unexpected query=%q opts=%+v", query, opts)
				}
				return page("", "m1", "m9"), nil
			},
		},
	}
	svc := newTestRulesService(t, repo)
	ctx := context.Background()
	seedRule(t, svc, "from:medium.com", "archive", "", 0)

	// Gmail returns m1,m9 but only m1,m2,m3 are candidates → match {m1}, remaining {m2,m3}.
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
	// SearchMessages expectations (in order):
	//   1. "in:inbox from:medium.com" {MaxResults:100, PageToken:""}    → page("tok2", "m1")
	//   2. "in:inbox from:medium.com" {MaxResults:100, PageToken:"tok2"} → page("", "m2")
	repo := &stubMessageRepo{
		calls: []func(string, QueryOptions) (*MessagePage, error){
			func(query string, opts QueryOptions) (*MessagePage, error) {
				if query != "in:inbox from:medium.com" || opts.MaxResults != 100 || opts.PageToken != "" {
					t.Errorf("call 1: unexpected query=%q opts=%+v", query, opts)
				}
				return page("tok2", "m1"), nil
			},
			func(query string, opts QueryOptions) (*MessagePage, error) {
				if query != "in:inbox from:medium.com" || opts.MaxResults != 100 || opts.PageToken != "tok2" {
					t.Errorf("call 2: unexpected query=%q opts=%+v", query, opts)
				}
				return page("", "m2"), nil
			},
		},
	}
	svc := newTestRulesService(t, repo)
	seedRule(t, svc, "from:medium.com", "archive", "", 0)

	matches, _, err := svc.Partition(context.Background(), "in:inbox", nil)
	if err != nil || len(matches) != 1 || len(matches[0].MessageIDs) != 2 {
		t.Fatalf("pagination broken: %v %+v", err, matches)
	}
}

// TestDeleteRuleBestEffort verifies that when the mirrored Gmail filter delete
// fails, DeleteRule still removes the local rule and returns a non-nil error
// mentioning the Gmail filter.
func TestDeleteRuleBestEffort(t *testing.T) {
	ctx := context.Background()

	// Build the DB store so we can call SetGmailFilterID directly.
	rawDB, err := db.Open(ctx, t.TempDir()+"/best_effort.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })
	rulesStore := db.NewDeterministicRulesStore(rawDB)

	repo := &stubMessageRepo{
		calls: []func(string, QueryOptions) (*MessagePage, error){
			// validateQuery probe for SaveRule.
			func(_ string, _ QueryOptions) (*MessagePage, error) {
				return page(""), nil
			},
		},
	}

	svc := NewDeterministicRulesService(rulesStore, repo, nil, &stubFailingFilter{})
	svc.SetAccountEmail("user@example.com")

	// Save a rule so we have something to delete.
	r, err := svc.SaveRule(ctx, "from:newsletter.com", "archive", "", 0)
	if err != nil || r == nil {
		t.Fatalf("save rule: %v %+v", err, r)
	}

	// Simulate the rule having been synced: stamp a gmail_filter_id directly.
	if err := rulesStore.SetGmailFilterID(ctx, "user@example.com", r.ID, "filter-abc"); err != nil {
		t.Fatalf("SetGmailFilterID: %v", err)
	}

	// DeleteRule should propagate the remote error but still remove locally.
	delErr := svc.DeleteRule(ctx, r.ID)
	if delErr == nil {
		t.Fatal("expected non-nil error when Gmail filter delete fails")
	}
	if !strings.Contains(delErr.Error(), "Gmail filter") {
		t.Errorf("error should mention Gmail filter, got: %v", delErr)
	}

	// Rule must be gone locally.
	rules, err := svc.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("rule should be deleted locally, got %d rule(s)", len(rules))
	}
}
