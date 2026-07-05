package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/db"
	internalgmail "github.com/ajramos/giztui/internal/gmail"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

// Compile-time check: *internalgmail.Client must satisfy GmailFilterAPI.
var _ GmailFilterAPI = (*internalgmail.Client)(nil)

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
func (f *stubFailingFilter) ListFilters() ([]*gmailapi.Filter, error) {
	return nil, nil
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

func TestRulesServicePartitionCapsPerRule(t *testing.T) {
	// 5 pages of 100 distinct IDs each, every page with a non-empty NextPageToken.
	// The cap (500) stops the sweep after exactly 5 calls; a 6th call panics (overcall).
	const pages = 5
	const idsPerPage = 100

	calls := make([]func(string, QueryOptions) (*MessagePage, error), pages)
	for p := 0; p < pages; p++ {
		p := p // capture
		calls[p] = func(_ string, opts QueryOptions) (*MessagePage, error) {
			ids := make([]string, idsPerPage)
			for i := 0; i < idsPerPage; i++ {
				ids[i] = fmt.Sprintf("msg-%d-%d", p, i)
			}
			return page("tok-next", ids...), nil
		}
	}

	repo := &stubMessageRepo{calls: calls}
	svc := newTestRulesService(t, repo)
	ctx := context.Background()
	seedRule(t, svc, "from:newsletter.com", "archive", "", 0)

	matches, _, err := svc.Partition(ctx, "", nil)
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	if repo.idx != pages {
		t.Fatalf("expected exactly %d SearchMessages calls, got %d", pages, repo.idx)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 RuleMatch, got %d", len(matches))
	}
	if got := len(matches[0].MessageIDs); got != partitionMaxPerRule {
		t.Fatalf("expected %d MessageIDs (cap), got %d", partitionMaxPerRule, got)
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

// fakeFilterAPI records filter calls for sync tests.
type fakeFilterAPI struct {
	created   []string                 // queries passed to CreateFilter
	actions   []*gmailapi.FilterAction // actions passed to CreateFilter
	deleted   []string                 // ids passed to DeleteFilter
	nextID    string
	fail      error // returned by CreateFilter (and DeleteFilter if deleteErr is nil)
	deleteErr error // returned by DeleteFilter when set (takes priority over fail)

	remote    []*gmailapi.Filter // returned by ListFilters
	remoteErr error              // returned by ListFilters when set
}

func (f *fakeFilterAPI) ListFilters() ([]*gmailapi.Filter, error) {
	if f.remoteErr != nil {
		return nil, f.remoteErr
	}
	return f.remote, nil
}

func (f *fakeFilterAPI) CreateFilter(query string, action *gmailapi.FilterAction) (string, error) {
	if f.fail != nil {
		return "", f.fail
	}
	f.created = append(f.created, query)
	f.actions = append(f.actions, action)
	return f.nextID, nil
}
func (f *fakeFilterAPI) DeleteFilter(id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
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
	svc := newTestRulesService(t, &stubMessageRepo{})
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
	// Archive action must map to RemoveLabelIds=["INBOX"] with no AddLabelIds.
	if len(filters.actions) != 1 {
		t.Fatalf("expected 1 recorded action, got %d", len(filters.actions))
	}
	gotAction := filters.actions[0]
	if len(gotAction.RemoveLabelIds) != 1 || gotAction.RemoveLabelIds[0] != "INBOX" {
		t.Fatalf("archive action: want RemoveLabelIds=[INBOX], got %v", gotAction.RemoveLabelIds)
	}
	if len(gotAction.AddLabelIds) != 0 {
		t.Fatalf("archive action: want empty AddLabelIds, got %v", gotAction.AddLabelIds)
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
	svc := newTestRulesService(t, &stubMessageRepo{})
	svc.filters = &fakeFilterAPI{nextID: "F1"}
	seedRule(t, svc, "from:jira", "prompt", "", 7)
	rules, _ := svc.ListRules(context.Background())
	if err := svc.SyncRule(context.Background(), rules[0].ID); err == nil {
		t.Fatal("prompt rules cannot be mirrored — sync must fail")
	}
}

func TestRulesServiceDeleteMirroredRuleDeletesFilter(t *testing.T) {
	svc := newTestRulesService(t, &stubMessageRepo{})
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

// TestRulesServiceSyncToleratesMissingRemoteFilter verifies that a 404 from Gmail
// on delete (stale filter ID) is treated as success so sync and unsync can proceed.
// Also asserts that a non-404 delete error still propagates.
func TestRulesServiceSyncToleratesMissingRemoteFilter(t *testing.T) {
	ctx := context.Background()

	// --- Part 1: 404 on SyncRule (re-sync of already-gone filter) ---
	svc := newTestRulesService(t, &stubMessageRepo{})
	filters := &fakeFilterAPI{nextID: "F1"}
	svc.filters = filters
	seedRule(t, svc, "from:newsletter.com", "archive", "", 0)
	rules, _ := svc.ListRules(ctx)
	id := rules[0].ID

	// Initial sync → stores "F1".
	if err := svc.SyncRule(ctx, id); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	rules, _ = svc.ListRules(ctx)
	if rules[0].GmailFilterID != "F1" {
		t.Fatalf("expected F1 stored, got %q", rules[0].GmailFilterID)
	}

	// Simulate filter already deleted on Gmail side → 404 on next delete.
	filters.deleteErr = &googleapi.Error{Code: 404}
	filters.nextID = "F2"

	// Re-sync must succeed and store "F2".
	if err := svc.SyncRule(ctx, id); err != nil {
		t.Fatalf("re-sync with 404 delete must succeed: %v", err)
	}
	rules, _ = svc.ListRules(ctx)
	if rules[0].GmailFilterID != "F2" {
		t.Fatalf("expected F2 stored after 404-tolerant re-sync, got %q", rules[0].GmailFilterID)
	}

	// --- Part 2: 404 on UnsyncRule ---
	// deleteErr is still 404; unsync must succeed and clear the ID.
	if err := svc.UnsyncRule(ctx, id); err != nil {
		t.Fatalf("unsync with 404 delete must succeed: %v", err)
	}
	rules, _ = svc.ListRules(ctx)
	if rules[0].GmailFilterID != "" {
		t.Fatalf("unsync must clear GmailFilterID, got %q", rules[0].GmailFilterID)
	}

	// --- Part 3: non-404 delete error must still fail SyncRule ---
	// Seed a second rule and sync it cleanly first.
	svc2 := newTestRulesService(t, &stubMessageRepo{})
	filters2 := &fakeFilterAPI{nextID: "F3"}
	svc2.filters = filters2
	seedRule(t, svc2, "from:spam.com", "archive", "", 0)
	rules2, _ := svc2.ListRules(ctx)
	id2 := rules2[0].ID

	if err := svc2.SyncRule(ctx, id2); err != nil {
		t.Fatalf("sync rule2: %v", err)
	}

	// Inject a 500 delete error → re-sync must fail.
	filters2.deleteErr = &googleapi.Error{Code: 500}
	filters2.nextID = "F4"
	if err := svc2.SyncRule(ctx, id2); err == nil {
		t.Fatal("re-sync with non-404 delete error must fail")
	}
}
