package desktop

import (
	"context"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/services"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// --- fakes -------------------------------------------------------------------

type fakeRepo struct {
	page     *services.MessagePage
	detail   *gmail.Message
	lastOpts services.QueryOptions
	lastQ    string
	err      error
}

func (f *fakeRepo) GetMessages(ctx context.Context, opts services.QueryOptions) (*services.MessagePage, error) {
	f.lastOpts = opts
	return f.page, f.err
}
func (f *fakeRepo) SearchMessages(ctx context.Context, query string, opts services.QueryOptions) (*services.MessagePage, error) {
	f.lastQ = query
	f.lastOpts = opts
	return f.page, f.err
}
func (f *fakeRepo) GetMessage(ctx context.Context, id string) (*gmail.Message, error) {
	return f.detail, f.err
}
func (f *fakeRepo) UpdateMessage(ctx context.Context, id string, updates services.MessageUpdates) error {
	return f.err
}
func (f *fakeRepo) GetDrafts(ctx context.Context, maxResults int64) ([]*gmail_v1.Draft, error) {
	return nil, f.err
}
func (f *fakeRepo) GetDraft(ctx context.Context, draftID string) (*gmail_v1.Draft, error) {
	return nil, f.err
}

type fakeMail struct {
	metas []*gmail_v1.Message
	err   error
}

func (f *fakeMail) GetMessagesMetadataParallel(ids []string, maxWorkers int) ([]*gmail_v1.Message, error) {
	return f.metas, f.err
}
func (f *fakeMail) ExtractHeader(msg *gmail_v1.Message, name string) string {
	if msg.Payload == nil {
		return ""
	}
	for _, h := range msg.Payload.Headers {
		if h.Name == name {
			return h.Value
		}
	}
	return ""
}
func (f *fakeMail) ExtractDate(msg *gmail_v1.Message) time.Time  { return time.Unix(0, 0) }
func (f *fakeMail) ExtractLabels(msg *gmail_v1.Message) []string { return msg.LabelIds }

// email service that only records the calls the API forwards.
type fakeEmail struct {
	services.EmailService
	archived, trashed, read, unread string
}

func (f *fakeEmail) ArchiveMessage(ctx context.Context, id string) error { f.archived = id; return nil }
func (f *fakeEmail) TrashMessage(ctx context.Context, id string) error   { f.trashed = id; return nil }
func (f *fakeEmail) MarkAsRead(ctx context.Context, id string) error     { f.read = id; return nil }
func (f *fakeEmail) MarkAsUnread(ctx context.Context, id string) error   { f.unread = id; return nil }

func rawMsg(id, subject, from string, labels []string) *gmail_v1.Message {
	return &gmail_v1.Message{
		Id:       id,
		ThreadId: "t-" + id,
		Snippet:  "snippet-" + id,
		LabelIds: labels,
		Payload: &gmail_v1.MessagePart{
			Headers: []*gmail_v1.MessagePartHeader{
				{Name: "Subject", Value: subject},
				{Name: "From", Value: from},
			},
		},
	}
}

// --- tests -------------------------------------------------------------------

func TestListInboxHydratesSummaries(t *testing.T) {
	repo := &fakeRepo{page: &services.MessagePage{
		Messages:      []*gmail_v1.Message{{Id: "1"}, {Id: "2"}},
		NextPageToken: "next",
	}}
	mail := &fakeMail{metas: []*gmail_v1.Message{
		rawMsg("1", "Hello", "a@x.com", []string{"INBOX", "UNREAD", "Work"}),
		rawMsg("2", "World", "b@x.com", []string{"INBOX"}),
	}}
	api := NewAPI(repo, &fakeEmail{}, nil, mail, nil)

	list, err := api.ListInbox(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("ListInbox: %v", err)
	}
	if list.NextPageToken != "next" {
		t.Errorf("NextPageToken = %q, want next", list.NextPageToken)
	}
	if got := len(list.Messages); got != 2 {
		t.Fatalf("got %d messages, want 2", got)
	}
	m0 := list.Messages[0]
	if m0.Subject != "Hello" || m0.From != "a@x.com" || m0.Snippet != "snippet-1" {
		t.Errorf("unexpected summary: %+v", m0)
	}
	if !m0.Unread {
		t.Errorf("message 1 should be unread")
	}
	if len(m0.Labels) != 1 || m0.Labels[0] != "Work" {
		t.Errorf("expected only user label Work, got %v", m0.Labels)
	}
	if list.Messages[1].Unread {
		t.Errorf("message 2 should be read")
	}
	if repo.lastOpts.MaxResults != defaultPageSize {
		t.Errorf("default page size not applied: %d", repo.lastOpts.MaxResults)
	}
}

func TestSearchPassesQuery(t *testing.T) {
	repo := &fakeRepo{page: &services.MessagePage{}}
	api := NewAPI(repo, &fakeEmail{}, nil, &fakeMail{}, nil)
	if _, err := api.Search(context.Background(), "from:x has:attachment", "tok", 10); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if repo.lastQ != "from:x has:attachment" {
		t.Errorf("query not forwarded: %q", repo.lastQ)
	}
	if repo.lastOpts.PageToken != "tok" || repo.lastOpts.MaxResults != 10 {
		t.Errorf("opts not forwarded: %+v", repo.lastOpts)
	}
}

func TestGetMessageDetail(t *testing.T) {
	repo := &fakeRepo{detail: &gmail.Message{
		Message: &gmail_v1.Message{Id: "9", ThreadId: "t9", LabelIds: []string{"UNREAD", "INBOX"}},
		Subject: "Subj", From: "f@x.com", To: "t@x.com", PlainText: "body",
		Labels: []string{"UNREAD", "Work"},
	}}
	api := NewAPI(repo, &fakeEmail{}, nil, &fakeMail{}, nil)
	d, err := api.GetMessage(context.Background(), "9")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if d.ID != "9" || d.Subject != "Subj" || d.PlainText != "body" {
		t.Errorf("unexpected detail: %+v", d)
	}
	if !d.Unread {
		t.Errorf("expected unread")
	}
	if len(d.Labels) != 1 || d.Labels[0] != "Work" {
		t.Errorf("expected user label Work, got %v", d.Labels)
	}
}

func TestActionsForward(t *testing.T) {
	email := &fakeEmail{}
	api := NewAPI(&fakeRepo{}, email, nil, &fakeMail{}, nil)
	ctx := context.Background()
	_ = api.Archive(ctx, "a")
	_ = api.Trash(ctx, "b")
	_ = api.MarkRead(ctx, "c")
	_ = api.MarkUnread(ctx, "d")
	if email.archived != "a" || email.trashed != "b" || email.read != "c" || email.unread != "d" {
		t.Errorf("actions not forwarded: %+v", email)
	}
}

func TestUserLabelsFilter(t *testing.T) {
	got := userLabels([]string{"INBOX", "UNREAD", "CATEGORY_PERSONAL", "Work", "STARRED", "Project/Sub"})
	want := []string{"Work", "Project/Sub"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
