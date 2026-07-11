package services

import (
	"context"
	"fmt"
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

// stubLabelService answers ListLabels with a fixed set; everything else panics
// (the import path must not touch other label operations).
type stubLabelService struct {
	LabelService // panic on any method not overridden below (nil embedded interface)
	labels       []*gmailapi.Label
}

func (s *stubLabelService) ListLabels(_ context.Context) ([]*gmailapi.Label, error) {
	return s.labels, nil
}

func TestImportQueryFromCriteria(t *testing.T) {
	cases := []struct {
		name     string
		criteria *gmailapi.FilterCriteria
		want     string
		wantErr  bool
	}{
		{"from", &gmailapi.FilterCriteria{From: "a@b.com"}, "from:(a@b.com)", false},
		{"combo", &gmailapi.FilterCriteria{From: "a@b.com", Subject: "hi", HasAttachment: true},
			"from:(a@b.com) subject:(hi) has:attachment", false},
		{"query verbatim", &gmailapi.FilterCriteria{Query: "list:golang-nuts"}, "list:golang-nuts", false},
		{"negated", &gmailapi.FilterCriteria{To: "me@x.com", NegatedQuery: "unsubscribe"},
			"to:(me@x.com) -(unsubscribe)", false},
		{"size unsupported", &gmailapi.FilterCriteria{Size: 5, SizeComparison: "larger"}, "", true},
		{"exclude chats unsupported", &gmailapi.FilterCriteria{From: "a@b.com", ExcludeChats: true}, "", true},
		{"empty", &gmailapi.FilterCriteria{}, "", true},
		{"nil", nil, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := importQueryFromCriteria(tc.criteria)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("query = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestImportActionFromFilter(t *testing.T) {
	labels := map[string]string{"Label_7": "Newsletters"}
	mk := func(add, remove []string, forward string) *gmailapi.Filter {
		return &gmailapi.Filter{Action: &gmailapi.FilterAction{
			AddLabelIds: add, RemoveLabelIds: remove, Forward: forward,
		}}
	}
	cases := []struct {
		name       string
		filter     *gmailapi.Filter
		wantAction string
		wantLabel  string
		wantErr    bool
	}{
		{"archive", mk(nil, []string{"INBOX"}, ""), "archive", "", false},
		{"mark read", mk(nil, []string{"UNREAD"}, ""), "mark_read", "", false},
		{"trash", mk([]string{"TRASH"}, nil, ""), "trash", "", false},
		{"label", mk([]string{"Label_7"}, nil, ""), "label", "Newsletters", false},
		{"unknown label id", mk([]string{"Label_404"}, nil, ""), "", "", true},
		{"forward", mk(nil, nil, "x@y.com"), "", "", true},
		{"label plus archive combo", mk([]string{"Label_7"}, []string{"INBOX"}, ""), "", "", true},
		{"no action", mk(nil, nil, ""), "", "", true},
		{"nil action", &gmailapi.Filter{}, "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action, label, err := importActionFromFilter(tc.filter, labels)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if action != tc.wantAction || label != tc.wantLabel {
				t.Fatalf("got (%q, %q), want (%q, %q)", action, label, tc.wantAction, tc.wantLabel)
			}
		})
	}
}

// End-to-end import pass against a real (temp) store: one filter adopts an existing
// unmirrored rule, one becomes a new mirrored rule, an already-linked one is skipped,
// and a forwarding one surfaces as Gmail-only. A second pass must be a no-op.
func TestImportGmailFilters(t *testing.T) {
	ctx := context.Background()
	svc := newTestRulesService(t, &stubMessageRepo{})

	// Existing local state: an unmirrored "from:(foo)" archive rule (adoption target)
	// and a rule already linked to filter F-linked.
	seedRule(t, svc, "from:(foo)", "archive", "", 0)
	seedRule(t, svc, "from:(old)", "trash", "", 0)
	rules, err := svc.ListRules(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := svc.store.SetGmailFilterID(ctx, "user@example.com", rules[1].ID, "F-linked"); err != nil {
		t.Fatalf("link seed rule: %v", err)
	}

	svc.labels = &stubLabelService{labels: []*gmailapi.Label{{Id: "Label_7", Name: "Newsletters"}}}

	svc.filters = &fakeFilterAPI{remote: []*gmailapi.Filter{
		{Id: "F-adopt", Criteria: &gmailapi.FilterCriteria{From: "foo"},
			Action: &gmailapi.FilterAction{RemoveLabelIds: []string{"INBOX"}}},
		{Id: "F-new", Criteria: &gmailapi.FilterCriteria{From: "news@x.com"},
			Action: &gmailapi.FilterAction{AddLabelIds: []string{"Label_7"}}},
		{Id: "F-linked", Criteria: &gmailapi.FilterCriteria{From: "old"},
			Action: &gmailapi.FilterAction{AddLabelIds: []string{"TRASH"}}},
		{Id: "F-fwd", Criteria: &gmailapi.FilterCriteria{From: "boss@x.com"},
			Action: &gmailapi.FilterAction{Forward: "me@else.com"}},
	}}

	res, err := svc.ImportGmailFilters(ctx)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.Imported != 1 || res.Adopted != 1 || len(res.Unsupported) != 1 {
		t.Fatalf("got imported=%d adopted=%d unsupported=%d, want 1/1/1",
			res.Imported, res.Adopted, len(res.Unsupported))
	}
	if res.Unsupported[0].ID != "F-fwd" || res.Unsupported[0].Reason == "" {
		t.Fatalf("unsupported = %+v, want F-fwd with a reason", res.Unsupported[0])
	}

	after, err := svc.ListRules(ctx)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(after) != 3 {
		t.Fatalf("rules after import = %d, want 3 (2 seeded + 1 imported)", len(after))
	}
	byQuery := map[string]DeterministicRuleInfo{}
	for _, r := range after {
		byQuery[r.Query] = r
	}
	if r := byQuery["from:(foo)"]; r.GmailFilterID != "F-adopt" {
		t.Fatalf("adoption: from:(foo) filter id = %q, want F-adopt", r.GmailFilterID)
	}
	if r := byQuery["from:(news@x.com)"]; r.Action != "label" || r.Label != "Newsletters" || r.GmailFilterID != "F-new" {
		t.Fatalf("imported rule wrong: %+v", r)
	}

	// Second pass: everything is linked or unsupported already — no new rules.
	res2, err := svc.ImportGmailFilters(ctx)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if res2.Imported != 0 || res2.Adopted != 0 {
		t.Fatalf("second pass must be a no-op, got imported=%d adopted=%d", res2.Imported, res2.Adopted)
	}
	if again, _ := svc.ListRules(ctx); len(again) != 3 {
		t.Fatalf("second pass created rules: %d, want 3", len(again))
	}
}

// Reconcile follows Gmail: a mirrored rule whose filter vanished from Gmail is
// dropped, while a local-only rule (never mirrored) is left untouched even though
// it isn't in the Gmail list.
func TestReconcileFollowsGmailDeletions(t *testing.T) {
	ctx := context.Background()
	svc := newTestRulesService(t, &stubMessageRepo{})

	seedRule(t, svc, "from:(mirrored)", "archive", "", 0)
	seedRule(t, svc, "from:(localonly)", "trash", "", 0)
	rules, err := svc.ListRules(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := svc.store.SetGmailFilterID(ctx, "user@example.com", rules[0].ID, "F-gone"); err != nil {
		t.Fatalf("link: %v", err)
	}

	// Gmail no longer has F-gone (deleted or edited there — editing recreates a new ID).
	svc.filters = &fakeFilterAPI{remote: []*gmailapi.Filter{}}

	res, err := svc.ImportGmailFilters(ctx)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.Removed != 1 {
		t.Fatalf("Removed = %d, want 1", res.Removed)
	}

	after, err := svc.ListRules(ctx)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(after) != 1 || after[0].Query != "from:(localonly)" {
		t.Fatalf("after reconcile = %+v, want only the local-only rule kept", after)
	}
}

func TestImportGmailFiltersListError(t *testing.T) {
	svc := newTestRulesService(t, &stubMessageRepo{})
	svc.filters = &fakeFilterAPI{remoteErr: fmt.Errorf("403: insufficient scopes")}
	if _, err := svc.ImportGmailFilters(context.Background()); err == nil {
		t.Fatal("list failure must surface as an error")
	}
}
