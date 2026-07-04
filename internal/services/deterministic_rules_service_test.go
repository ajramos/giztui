package services

import (
	"context"
	"fmt"
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
