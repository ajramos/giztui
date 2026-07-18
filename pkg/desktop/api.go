// Package desktop exposes a thin, JSON-friendly API over GizTUI's existing
// service layer so alternative front-ends (e.g. a Wails desktop app or a local
// web server) can reuse all of the Gmail/LLM business logic without touching
// the TUI presentation layer.
//
// This package deliberately depends only on the service interfaces defined in
// internal/services, so it compiles and unit-tests on any platform (no Wails,
// no CGO, no webkit required). The Wails-specific glue lives in the nested
// module under ./desktop and simply binds an *API instance.
package desktop

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ajramos/giztui/internal/services"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// defaultPageSize is the number of messages fetched per inbox/search page.
const defaultPageSize = 50

// metadataWorkers bounds the concurrency used to hydrate list metadata.
const metadataWorkers = 10

// mailClient is the minimal subset of *gmail.Client the API needs to hydrate
// message summaries. Declaring it as an interface keeps the API unit-testable
// with a fake and avoids leaking the concrete Gmail client into list logic.
type mailClient interface {
	GetMessagesMetadataParallel(ids []string, maxWorkers int) ([]*gmail_v1.Message, error)
	ExtractHeader(msg *gmail_v1.Message, name string) string
	ExtractDate(msg *gmail_v1.Message) time.Time
	ExtractLabels(msg *gmail_v1.Message) []string
}

// Deps bundles everything an API needs. Using a struct keeps the constructor
// stable as the surface grows (AI, compose, …).
type Deps struct {
	Repo         services.MessageRepository
	Email        services.EmailService
	Labels       services.LabelService
	Mail         mailClient
	AI           services.AIService            // optional; nil when no LLM is configured
	Attach       services.AttachmentService    // optional
	Prompts      services.PromptService        // optional; nil without LLM+DB
	Web          services.GmailWebService      // optional
	Composition  services.CompositionService   // optional; used to load drafts
	Draft        draftClient                   // optional; draft CRUD (usually *gmail.Client)
	Link         services.LinkService          // optional
	Obsidian     services.ObsidianService      // optional
	Slack        services.SlackService         // optional
	Thread       services.ThreadService        // optional
	Query        services.QueryService         // optional
	Analyzer     services.InboxAnalyzerService // optional
	Rules        services.AnalyzerRulesService // optional; analyzer preference rules
	Theme        services.ThemeService         // optional
	Invite       inviteClient                  // optional; calendar invite detection (gmail client)
	Cal          calClient                     // optional; calendar RSVP responder
	Cache        services.CacheService         // optional; summary cache (for clearing)
	AccountEmail string                        // active account address, used as the "from" for sends
	Logger       *log.Logger
}

// API is the front-end-agnostic entry point. Every method returns
// JSON-serializable DTOs and plain errors, making it trivial to bind from Wails
// or serve over HTTP.
type API struct {
	repo         services.MessageRepository
	email        services.EmailService
	labels       services.LabelService
	mail         mailClient
	ai           services.AIService
	attach       services.AttachmentService
	prompts      services.PromptService
	web          services.GmailWebService
	composition  services.CompositionService
	draft        draftClient
	link         services.LinkService
	obsidian     services.ObsidianService
	slack        services.SlackService
	thread       services.ThreadService
	query        services.QueryService
	analyzer     services.InboxAnalyzerService
	rules        services.AnalyzerRulesService
	theme        services.ThemeService
	invite       inviteClient
	cal          calClient
	cache        services.CacheService
	accountEmail string
	logger       *log.Logger

	labelsOnce sync.Once
	labelNames map[string]string // Gmail label ID -> human name
}

// NewAPI wires an API from already-constructed services. Use NewSession when you
// want the whole stack built from config/credentials on disk.
func NewAPI(d Deps) *API {
	return &API{
		repo:         d.Repo,
		email:        d.Email,
		labels:       d.Labels,
		mail:         d.Mail,
		ai:           d.AI,
		attach:       d.Attach,
		prompts:      d.Prompts,
		web:          d.Web,
		composition:  d.Composition,
		draft:        d.Draft,
		link:         d.Link,
		obsidian:     d.Obsidian,
		slack:        d.Slack,
		thread:       d.Thread,
		query:        d.Query,
		analyzer:     d.Analyzer,
		rules:        d.Rules,
		theme:        d.Theme,
		invite:       d.Invite,
		cal:          d.Cal,
		cache:        d.Cache,
		accountEmail: d.AccountEmail,
		logger:       d.Logger,
	}
}

func (a *API) logf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Printf(format, args...)
	}
}

// ListInbox returns a page of inbox message summaries. Pass an empty pageToken
// for the first page; use the returned NextPageToken to paginate.
func (a *API) ListInbox(ctx context.Context, pageToken string, pageSize int64) (*MessageList, error) {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	page, err := a.repo.GetMessages(ctx, services.QueryOptions{
		MaxResults: pageSize,
		PageToken:  pageToken,
	})
	if err != nil {
		return nil, err
	}
	return a.hydrate(page)
}

// Search returns a page of message summaries matching a Gmail search query
// (supports the full Gmail operator syntax, e.g. "from:x has:attachment").
func (a *API) Search(ctx context.Context, query, pageToken string, pageSize int64) (*MessageList, error) {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	page, err := a.repo.SearchMessages(ctx, query, services.QueryOptions{
		MaxResults: pageSize,
		PageToken:  pageToken,
	})
	if err != nil {
		return nil, err
	}
	return a.hydrate(page)
}

// GetMessage returns the full detail (headers + body) of a single message.
func (a *API) GetMessage(ctx context.Context, id string) (*MessageDetail, error) {
	msg, err := a.repo.GetMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := &MessageDetail{
		Subject:   msg.Subject,
		From:      msg.From,
		To:        msg.To,
		Cc:        msg.Cc,
		Date:      msg.Date,
		PlainText: msg.PlainText,
		HTML:      msg.HTML,
		Labels:    userLabels(msg.Labels),
	}
	if msg.Message != nil {
		detail.ID = msg.Message.Id
		detail.ThreadID = msg.Message.ThreadId
		detail.Unread = containsLabel(msg.Message.LabelIds, "UNREAD")
	}
	return detail, nil
}

// Archive removes the INBOX label from a message.
func (a *API) Archive(ctx context.Context, id string) error {
	return a.email.ArchiveMessage(ctx, id)
}

// Trash moves a message to the Gmail trash.
func (a *API) Trash(ctx context.Context, id string) error {
	return a.email.TrashMessage(ctx, id)
}

// MarkRead marks a message as read.
func (a *API) MarkRead(ctx context.Context, id string) error {
	return a.email.MarkAsRead(ctx, id)
}

// MarkUnread marks a message as unread.
func (a *API) MarkUnread(ctx context.Context, id string) error {
	return a.email.MarkAsUnread(ctx, id)
}

// ListLabels returns all of the account's labels.
func (a *API) ListLabels(ctx context.Context) ([]Label, error) {
	labels, err := a.labels.ListLabels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Label, 0, len(labels))
	for _, l := range labels {
		if l == nil {
			continue
		}
		out = append(out, Label{ID: l.Id, Name: l.Name})
	}
	return out, nil
}

// hydrate turns a page of bare message IDs into fully-populated summaries by
// fetching metadata in parallel, preserving the original inbox order.
func (a *API) hydrate(page *services.MessagePage) (*MessageList, error) {
	ids := make([]string, 0, len(page.Messages))
	for _, m := range page.Messages {
		if m != nil {
			ids = append(ids, m.Id)
		}
	}

	result := &MessageList{
		Messages:      make([]MessageSummary, 0, len(ids)),
		NextPageToken: page.NextPageToken,
	}
	if len(ids) == 0 {
		return result, nil
	}

	metas, err := a.mail.GetMessagesMetadataParallel(ids, metadataWorkers)
	if err != nil {
		return nil, err
	}
	for _, m := range metas {
		if m == nil {
			// Metadata fetch failed for this ID; skip rather than fail the page.
			continue
		}
		rawLabels := a.mail.ExtractLabels(m)
		result.Messages = append(result.Messages, MessageSummary{
			ID:       m.Id,
			ThreadID: m.ThreadId,
			Subject:  a.mail.ExtractHeader(m, "Subject"),
			From:     a.mail.ExtractHeader(m, "From"),
			Snippet:  m.Snippet,
			Date:     a.mail.ExtractDate(m),
			Unread:   containsLabel(rawLabels, "UNREAD"),
			Labels:   a.resolveLabels(rawLabels),
		})
	}
	return result, nil
}

// systemLabels are Gmail-internal labels not shown as user-facing chips.
var systemLabels = map[string]struct{}{
	"INBOX": {}, "UNREAD": {}, "SENT": {}, "DRAFT": {}, "TRASH": {},
	"SPAM": {}, "CHAT": {}, "IMPORTANT": {}, "STARRED": {},
}

// resolveLabels maps raw Gmail label IDs to human names (lazily loading the
// account's label list once) and drops system/category labels. Used for list
// summaries, whose metadata only carries label IDs.
func (a *API) resolveLabels(ids []string) []string {
	a.labelsOnce.Do(func() {
		a.labelNames = map[string]string{}
		if a.labels != nil {
			if ls, err := a.labels.ListLabels(context.Background()); err == nil {
				for _, l := range ls {
					if l != nil {
						a.labelNames[l.Id] = l.Name
					}
				}
			}
		}
	})
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, sys := systemLabels[id]; sys {
			continue
		}
		if strings.HasPrefix(id, "CATEGORY_") {
			continue
		}
		if name, ok := a.labelNames[id]; ok && name != "" {
			out = append(out, name)
		} else {
			out = append(out, id)
		}
	}
	return out
}

// userLabels filters out system/category labels, leaving only labels a user
// would recognize (their own labels and named categories).
func userLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if _, sys := systemLabels[l]; sys {
			continue
		}
		if strings.HasPrefix(l, "CATEGORY_") {
			continue
		}
		out = append(out, l)
	}
	return out
}

func containsLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}
