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
	sent                            *sendRecord
	replied                         *replyRecord
	bulkArchived, bulkTrashed       []string
	bulkRead, bulkUnread            []string
}

func (f *fakeEmail) ArchiveMessage(ctx context.Context, id string) error { f.archived = id; return nil }
func (f *fakeEmail) TrashMessage(ctx context.Context, id string) error   { f.trashed = id; return nil }
func (f *fakeEmail) MarkAsRead(ctx context.Context, id string) error     { f.read = id; return nil }
func (f *fakeEmail) MarkAsUnread(ctx context.Context, id string) error   { f.unread = id; return nil }

// records send/reply calls forwarded by the compose API.
type sendRecord struct {
	from, to, subject, body string
	cc, bcc                 []string
}
type replyRecord struct {
	id, body string
	send     bool
	cc       []string
}

func (f *fakeEmail) SendMessage(ctx context.Context, from, to, subject, body string, cc, bcc []string) error {
	f.sent = &sendRecord{from: from, to: to, subject: subject, body: body, cc: cc, bcc: bcc}
	return nil
}
func (f *fakeEmail) ReplyToMessage(ctx context.Context, originalID, replyBody string, send bool, cc []string) error {
	f.replied = &replyRecord{id: originalID, body: replyBody, send: send, cc: cc}
	return nil
}
func (f *fakeEmail) BulkArchive(ctx context.Context, ids []string, onProgress ...func(done, total int)) error {
	f.bulkArchived = ids
	return nil
}
func (f *fakeEmail) BulkTrash(ctx context.Context, ids []string, onProgress ...func(done, total int)) error {
	f.bulkTrashed = ids
	return nil
}
func (f *fakeEmail) BulkMarkAsRead(ctx context.Context, ids []string, onProgress ...func(done, total int)) error {
	f.bulkRead = ids
	return nil
}
func (f *fakeEmail) BulkMarkAsUnread(ctx context.Context, ids []string, onProgress ...func(done, total int)) error {
	f.bulkUnread = ids
	return nil
}

// fakeAI records the content it was asked to summarize.
type fakeAI struct {
	services.AIService
	gotContent string
	summary    string
}

func (f *fakeAI) GenerateSummary(ctx context.Context, content string, options services.SummaryOptions) (*services.SummaryResult, error) {
	f.gotContent = content
	return &services.SummaryResult{Summary: f.summary}, nil
}
func (f *fakeAI) GenerateSummaryStream(ctx context.Context, content string, options services.SummaryOptions, onToken func(string)) (*services.SummaryResult, error) {
	f.gotContent = content
	// Emit the summary in two chunks to exercise the streaming path.
	if len(f.summary) > 0 {
		mid := len(f.summary) / 2
		onToken(f.summary[:mid])
		onToken(f.summary[mid:])
	}
	return &services.SummaryResult{Summary: f.summary}, nil
}

type fakeLabels struct {
	services.LabelService
	messageLabels []string
	applied       [2]string
	removed       [2]string
	bulkApplied   [2]interface{}
	bulkRemoved   [2]interface{}
}

func (f *fakeLabels) GetMessageLabels(ctx context.Context, messageID string) ([]string, error) {
	return f.messageLabels, nil
}
func (f *fakeLabels) ApplyLabel(ctx context.Context, messageID, labelID string) error {
	f.applied = [2]string{messageID, labelID}
	return nil
}
func (f *fakeLabels) RemoveLabel(ctx context.Context, messageID, labelID string) error {
	f.removed = [2]string{messageID, labelID}
	return nil
}
func (f *fakeLabels) BulkApplyLabel(ctx context.Context, ids []string, labelID string, onProgress ...func(done, total int)) error {
	f.bulkApplied = [2]interface{}{ids, labelID}
	return nil
}
func (f *fakeLabels) BulkRemoveLabel(ctx context.Context, ids []string, labelID string) error {
	f.bulkRemoved = [2]interface{}{ids, labelID}
	return nil
}

type fakePrompts struct {
	services.PromptService
	templates  []*services.PromptTemplate
	gotContent string
	gotID      int
	result     string
}

func (f *fakePrompts) ListPrompts(ctx context.Context, category string) ([]*services.PromptTemplate, error) {
	return f.templates, nil
}
func (f *fakePrompts) ApplyPromptStream(ctx context.Context, messageContent string, promptID int, variables map[string]string, onToken func(string)) (*services.PromptResult, error) {
	f.gotContent = messageContent
	f.gotID = promptID
	if len(f.result) > 0 {
		onToken(f.result)
	}
	return &services.PromptResult{PromptID: promptID, ResultText: f.result}, nil
}

type fakeDraft struct {
	created   *sendRecord
	updatedID string
	deletedID string
	drafts    []*gmail_v1.Draft
}

func (f *fakeDraft) ListDrafts(maxResults int64) ([]*gmail_v1.Draft, error) {
	return f.drafts, nil
}
func (f *fakeDraft) CreateDraft(to, subject, body string, cc []string) (string, error) {
	f.created = &sendRecord{to: to, subject: subject, body: body, cc: cc}
	return "draft-new", nil
}
func (f *fakeDraft) UpdateDraft(draftID, to, subject, body string, cc []string) error {
	f.updatedID = draftID
	return nil
}
func (f *fakeDraft) DeleteDraft(draftID string) error {
	f.deletedID = draftID
	return nil
}
func (f *fakeDraft) UntrashMessage(messageID string) error { return nil }
func (f *fakeDraft) GetMessageRaw(messageID string) ([]byte, error) {
	return []byte("From: x@y.com\r\nSubject: raw\r\n\r\nbody"), nil
}

type fakeComposition struct {
	services.CompositionService
	comp *services.Composition
}

func (f *fakeComposition) LoadDraftComposition(ctx context.Context, draftID string) (*services.Composition, error) {
	return f.comp, nil
}

type fakeWeb struct {
	services.GmailWebService
}

func (f *fakeWeb) GenerateGmailWebURL(messageID string) string {
	return "https://mail.google.com/mail/u/0/#inbox/" + messageID
}

type fakeAttach struct {
	services.AttachmentService
	infos        []services.AttachmentInfo
	downloadArgs [3]string
}

func (f *fakeAttach) GetMessageAttachments(ctx context.Context, messageID string) ([]services.AttachmentInfo, error) {
	return f.infos, nil
}
func (f *fakeAttach) DownloadAttachmentWithFilename(ctx context.Context, messageID, attachmentID, savePath, suggestedFilename string) (string, error) {
	f.downloadArgs = [3]string{messageID, attachmentID, suggestedFilename}
	return "/downloads/" + suggestedFilename, nil
}

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
	api := NewAPI(Deps{Repo: repo, Email: &fakeEmail{}, Mail: mail})

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
	api := NewAPI(Deps{Repo: repo, Email: &fakeEmail{}, Mail: &fakeMail{}})
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
	api := NewAPI(Deps{Repo: repo, Email: &fakeEmail{}, Mail: &fakeMail{}})
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
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: email, Mail: &fakeMail{}})
	ctx := context.Background()
	_ = api.Archive(ctx, "a")
	_ = api.Trash(ctx, "b")
	_ = api.MarkRead(ctx, "c")
	_ = api.MarkUnread(ctx, "d")
	if email.archived != "a" || email.trashed != "b" || email.read != "c" || email.unread != "d" {
		t.Errorf("actions not forwarded: %+v", email)
	}
}

func TestSummarize(t *testing.T) {
	repo := &fakeRepo{detail: &gmail.Message{
		Message:   &gmail_v1.Message{Id: "1"},
		PlainText: "the full email body",
	}}
	ai := &fakeAI{summary: "a short summary"}
	api := NewAPI(Deps{Repo: repo, Email: &fakeEmail{}, Mail: &fakeMail{}, AI: ai, AccountEmail: "me@x.com"})

	if !api.AIEnabled() {
		t.Fatal("AIEnabled should be true when AI dep is set")
	}
	got, err := api.Summarize(context.Background(), "1")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if got != "a short summary" {
		t.Errorf("summary = %q", got)
	}
	if ai.gotContent != "the full email body" {
		t.Errorf("AI got wrong content: %q", ai.gotContent)
	}
}

func TestSummarizeWithoutAI(t *testing.T) {
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}})
	if api.AIEnabled() {
		t.Fatal("AIEnabled should be false without AI dep")
	}
	if _, err := api.Summarize(context.Background(), "1"); err == nil {
		t.Fatal("expected error when AI not configured")
	}
}

func TestSendMail(t *testing.T) {
	email := &fakeEmail{}
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: email, Mail: &fakeMail{}, AccountEmail: "me@x.com"})

	if err := api.SendMail(context.Background(), "", "s", "b", nil, nil); err == nil {
		t.Error("expected error for empty recipient")
	}
	if err := api.SendMail(context.Background(), "to@x.com", "Hi", "Body", []string{" ", "cc@x.com"}, nil); err != nil {
		t.Fatalf("SendMail: %v", err)
	}
	if email.sent == nil {
		t.Fatal("send not forwarded")
	}
	if email.sent.from != "me@x.com" || email.sent.to != "to@x.com" || email.sent.subject != "Hi" {
		t.Errorf("unexpected send: %+v", email.sent)
	}
	if len(email.sent.cc) != 1 || email.sent.cc[0] != "cc@x.com" {
		t.Errorf("cc not cleaned: %v", email.sent.cc)
	}
}

func TestReply(t *testing.T) {
	email := &fakeEmail{}
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: email, Mail: &fakeMail{}})

	if err := api.Reply(context.Background(), "", "body", nil); err == nil {
		t.Error("expected error for empty original id")
	}
	if err := api.Reply(context.Background(), "orig", "", nil); err == nil {
		t.Error("expected error for empty body")
	}
	if err := api.Reply(context.Background(), "orig", "my reply", nil); err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if email.replied == nil || email.replied.id != "orig" || email.replied.body != "my reply" || !email.replied.send {
		t.Errorf("unexpected reply: %+v", email.replied)
	}
}

func TestLabelsForward(t *testing.T) {
	labels := &fakeLabels{messageLabels: []string{"Work", "UNREAD"}}
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}, Labels: labels})
	ctx := context.Background()

	ids, err := api.MessageLabelIDs(ctx, "m1")
	if err != nil || len(ids) != 2 {
		t.Fatalf("MessageLabelIDs: %v %v", ids, err)
	}
	_ = api.ApplyLabel(ctx, "m1", "L1")
	_ = api.RemoveLabel(ctx, "m1", "L2")
	if labels.applied != [2]string{"m1", "L1"} {
		t.Errorf("apply not forwarded: %v", labels.applied)
	}
	if labels.removed != [2]string{"m1", "L2"} {
		t.Errorf("remove not forwarded: %v", labels.removed)
	}
}

func TestAttachments(t *testing.T) {
	attach := &fakeAttach{infos: []services.AttachmentInfo{
		{AttachmentID: "a1", Filename: "doc.pdf", MimeType: "application/pdf", Size: 1234, Type: "document"},
	}}
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}, Attach: attach})
	ctx := context.Background()

	list, err := api.ListAttachments(ctx, "m1")
	if err != nil || len(list) != 1 {
		t.Fatalf("ListAttachments: %v %v", list, err)
	}
	if list[0].Filename != "doc.pdf" || list[0].Size != 1234 {
		t.Errorf("unexpected attachment: %+v", list[0])
	}
	path, err := api.DownloadAttachment(ctx, "m1", "a1", "doc.pdf")
	if err != nil || path != "/downloads/doc.pdf" {
		t.Fatalf("Download: %q %v", path, err)
	}
	if attach.downloadArgs != [3]string{"m1", "a1", "doc.pdf"} {
		t.Errorf("download args: %v", attach.downloadArgs)
	}
}

func TestAttachmentsNilService(t *testing.T) {
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}})
	list, err := api.ListAttachments(context.Background(), "m1")
	if err != nil || len(list) != 0 {
		t.Errorf("expected empty list without attach service, got %v %v", list, err)
	}
}

func TestSummarizeStream(t *testing.T) {
	repo := &fakeRepo{detail: &gmail.Message{
		Message:   &gmail_v1.Message{Id: "1"},
		PlainText: "body text",
	}}
	ai := &fakeAI{summary: "streamed summary"}
	api := NewAPI(Deps{Repo: repo, Email: &fakeEmail{}, Mail: &fakeMail{}, AI: ai})

	var got string
	final, err := api.SummarizeStream(context.Background(), "1", false, func(tok string) {
		got += tok
	})
	if err != nil {
		t.Fatalf("SummarizeStream: %v", err)
	}
	if final != "streamed summary" {
		t.Errorf("final = %q", final)
	}
	if got != "streamed summary" {
		t.Errorf("streamed tokens reassembled to %q", got)
	}
}

func TestBulkForward(t *testing.T) {
	email := &fakeEmail{}
	labels := &fakeLabels{}
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: email, Mail: &fakeMail{}, Labels: labels})
	ctx := context.Background()
	ids := []string{"a", "b", "c"}

	_ = api.BulkArchive(ctx, ids)
	_ = api.BulkTrash(ctx, ids)
	_ = api.BulkMarkRead(ctx, ids)
	_ = api.BulkMarkUnread(ctx, ids)
	_ = api.BulkApplyLabel(ctx, ids, "L1")
	_ = api.BulkRemoveLabel(ctx, ids, "L2")

	if len(email.bulkArchived) != 3 || len(email.bulkTrashed) != 3 {
		t.Errorf("bulk archive/trash not forwarded")
	}
	if len(email.bulkRead) != 3 || len(email.bulkUnread) != 3 {
		t.Errorf("bulk read/unread not forwarded")
	}
	if labels.bulkApplied[1] != "L1" || labels.bulkRemoved[1] != "L2" {
		t.Errorf("bulk labels not forwarded: %v %v", labels.bulkApplied, labels.bulkRemoved)
	}

	// Empty ids are a no-op that never touches the service.
	empty := &fakeEmail{}
	api2 := NewAPI(Deps{Repo: &fakeRepo{}, Email: empty, Mail: &fakeMail{}})
	if err := api2.BulkArchive(ctx, nil); err != nil {
		t.Errorf("empty BulkArchive should be nil, got %v", err)
	}
	if empty.bulkArchived != nil {
		t.Errorf("empty BulkArchive should not call service")
	}
}

func TestPrompts(t *testing.T) {
	repo := &fakeRepo{detail: &gmail.Message{
		Message:   &gmail_v1.Message{Id: "1"},
		PlainText: "email content",
	}}
	prompts := &fakePrompts{
		templates: []*services.PromptTemplate{
			{ID: 1, Name: "Summarize", Description: "d", Category: "general"},
			{ID: 2, Name: "Extract actions", Category: "general"},
			nil,
		},
		result: "prompt output",
	}
	api := NewAPI(Deps{Repo: repo, Email: &fakeEmail{}, Mail: &fakeMail{}, Prompts: prompts})

	if !api.PromptsEnabled() {
		t.Fatal("PromptsEnabled should be true")
	}
	list, err := api.ListPrompts(context.Background())
	if err != nil || len(list) != 2 {
		t.Fatalf("ListPrompts: %v %v", list, err)
	}
	if list[0].Name != "Summarize" || list[1].Name != "Extract actions" {
		t.Errorf("unexpected prompts: %+v", list)
	}

	var streamed string
	out, err := api.ApplyPromptStream(context.Background(), "1", 2, func(tok string) {
		streamed += tok
	})
	if err != nil {
		t.Fatalf("ApplyPromptStream: %v", err)
	}
	if out != "prompt output" || streamed != "prompt output" {
		t.Errorf("out=%q streamed=%q", out, streamed)
	}
	if prompts.gotContent != "email content" || prompts.gotID != 2 {
		t.Errorf("prompt got content=%q id=%d", prompts.gotContent, prompts.gotID)
	}
}

func TestPromptsDisabled(t *testing.T) {
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}})
	if api.PromptsEnabled() {
		t.Fatal("PromptsEnabled should be false without prompt service")
	}
	if list, err := api.ListPrompts(context.Background()); err != nil || len(list) != 0 {
		t.Errorf("expected empty list, got %v %v", list, err)
	}
	if _, err := api.ApplyPromptStream(context.Background(), "1", 1, func(string) {}); err == nil {
		t.Error("expected error when prompts disabled")
	}
}

func TestGmailWebURL(t *testing.T) {
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}, Web: &fakeWeb{}})
	if url := api.GmailWebURL("abc"); url != "https://mail.google.com/mail/u/0/#inbox/abc" {
		t.Errorf("GmailWebURL = %q", url)
	}
	// Without a web service it returns empty.
	api2 := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}})
	if url := api2.GmailWebURL("abc"); url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
}

func TestDrafts(t *testing.T) {
	draft := &fakeDraft{drafts: []*gmail_v1.Draft{
		{Id: "d1", Message: rawMsg("m1", "Draft subject", "", nil)},
	}}
	// rawMsg sets a From header, not To; add a To header for the list test.
	draft.drafts[0].Message.Payload.Headers = append(
		draft.drafts[0].Message.Payload.Headers,
		&gmail_v1.MessagePartHeader{Name: "To", Value: "bob@x.com"},
	)
	comp := &fakeComposition{comp: &services.Composition{
		To:      []services.Recipient{{Email: "bob@x.com"}, {Email: "eve@x.com"}},
		Subject: "Draft subject",
		Body:    "draft body",
	}}
	api := NewAPI(Deps{
		Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{},
		Draft: draft, Composition: comp,
	})
	ctx := context.Background()

	list, err := api.ListDrafts(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListDrafts: %v %v", list, err)
	}
	if list[0].ID != "d1" || list[0].Subject != "Draft subject" || list[0].To != "bob@x.com" {
		t.Errorf("unexpected draft summary: %+v", list[0])
	}

	det, err := api.GetDraft(ctx, "d1")
	if err != nil {
		t.Fatalf("GetDraft: %v", err)
	}
	if det.To != "bob@x.com, eve@x.com" || det.Body != "draft body" {
		t.Errorf("unexpected draft detail: %+v", det)
	}

	id, err := api.SaveDraft(ctx, "to@x.com", "S", "B", []string{"cc@x.com"})
	if err != nil || id != "draft-new" {
		t.Fatalf("SaveDraft: %q %v", id, err)
	}
	if draft.created == nil || draft.created.to != "to@x.com" {
		t.Errorf("create not forwarded: %+v", draft.created)
	}
	if err := api.UpdateDraft(ctx, "d1", "to@x.com", "S", "B", nil); err != nil || draft.updatedID != "d1" {
		t.Errorf("UpdateDraft: %v %q", err, draft.updatedID)
	}
	if err := api.UpdateDraft(ctx, "", "to", "S", "B", nil); err == nil {
		t.Error("expected error for empty draft id")
	}
	if err := api.DeleteDraft(ctx, "d1"); err != nil || draft.deletedID != "d1" {
		t.Errorf("DeleteDraft: %v %q", err, draft.deletedID)
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
