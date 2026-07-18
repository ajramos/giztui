package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ajramos/giztui/pkg/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Wails events the frontend subscribes to for streaming AI output.
const (
	summaryTokenEvent = "summary:token"
	promptTokenEvent  = "prompt:token"
)

// App is the Wails-bound backend. Its exported methods are callable from the
// frontend (window.go.main.App.*). It is intentionally a thin wrapper: all real
// logic lives in the shared pkg/desktop.API, which is reused verbatim from the
// TUI's service layer.
type App struct {
	ctx     context.Context
	session *desktop.Session
	initErr string
}

// NewApp creates the App in an unstarted state. Startup wiring happens in
// startup, once Wails has provided a context.
func NewApp() *App {
	return &App{}
}

// startup is invoked by Wails when the app launches. It builds the Gmail/service
// stack from the user's existing GizTUI config and OAuth token.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	sess, err := desktop.NewSession(ctx, desktop.Options{Logger: log.Default()})
	if err != nil {
		a.initErr = err.Error()
		log.Printf("desktop: session init failed: %v", err)
		return
	}
	a.session = sess
	if email, err := sess.AccountEmail(ctx); err == nil {
		log.Printf("desktop: signed in as %s", email)
	}
}

// shutdown is invoked by Wails on exit; it releases the local database.
func (a *App) shutdown(_ context.Context) {
	if a.session != nil {
		_ = a.session.Close()
	}
}

// api returns the shared API or an error describing why startup failed.
func (a *App) api() (*desktop.API, error) {
	if a.session == nil {
		if a.initErr != "" {
			return nil, &startupError{a.initErr}
		}
		return nil, &startupError{"session not initialized"}
	}
	return a.session.API, nil
}

type startupError struct{ msg string }

func (e *startupError) Error() string { return e.msg }

// --- bound methods (called from the frontend) --------------------------------

// InitError returns the startup error, if any, so the UI can surface config or
// auth problems instead of silently showing an empty inbox.
func (a *App) InitError() string { return a.initErr }

// AccountEmail returns the active account's email address.
func (a *App) AccountEmail() (string, error) {
	if a.session == nil {
		return "", &startupError{a.initErr}
	}
	return a.session.AccountEmail(a.ctx)
}

// OpenGmailWeb opens the message in Gmail's web interface in the system browser.
func (a *App) OpenGmailWeb(messageID string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	url := api.GmailWebURL(messageID)
	if url == "" {
		return fmt.Errorf("could not build a Gmail web URL")
	}
	wailsruntime.BrowserOpenURL(a.ctx, url)
	return nil
}

// ListDrafts returns the account's drafts.
func (a *App) ListDrafts() ([]desktop.DraftSummary, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListDrafts(a.ctx)
}

// GetDraft loads a draft for editing.
func (a *App) GetDraft(draftID string) (*desktop.DraftDetail, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.GetDraft(a.ctx, draftID)
}

// SaveDraft creates a new draft and returns its ID.
func (a *App) SaveDraft(to, subject, body string, cc []string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.SaveDraft(a.ctx, to, subject, body, cc)
}

// UpdateDraft overwrites an existing draft.
func (a *App) UpdateDraft(draftID, to, subject, body string, cc []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.UpdateDraft(a.ctx, draftID, to, subject, body, cc)
}

// DeleteDraft removes a draft.
func (a *App) DeleteDraft(draftID string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.DeleteDraft(a.ctx, draftID)
}

// ListAccounts returns all configured accounts for the account switcher.
func (a *App) ListAccounts() ([]desktop.AccountInfo, error) {
	if a.session == nil {
		return nil, &startupError{a.initErr}
	}
	return a.session.ListAccounts(a.ctx)
}

// SwitchAccount switches the active account and rebuilds the service stack.
func (a *App) SwitchAccount(id string) error {
	if a.session == nil {
		return &startupError{a.initErr}
	}
	return a.session.SwitchAccount(a.ctx, id)
}

// ListInbox returns a page of inbox summaries.
func (a *App) ListInbox(pageToken string, pageSize int) (*desktop.MessageList, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListInbox(a.ctx, pageToken, int64(pageSize))
}

// Search returns a page of message summaries matching a Gmail query.
func (a *App) Search(query, pageToken string, pageSize int) (*desktop.MessageList, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.Search(a.ctx, query, pageToken, int64(pageSize))
}

// GetMessage returns the full body/headers for a single message.
func (a *App) GetMessage(id string) (*desktop.MessageDetail, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.GetMessage(a.ctx, id)
}

// Archive removes a message from the inbox.
func (a *App) Archive(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.Archive(a.ctx, id)
}

// Trash moves a message to trash.
func (a *App) Trash(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.Trash(a.ctx, id)
}

// MarkRead marks a message as read.
func (a *App) MarkRead(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.MarkRead(a.ctx, id)
}

// MarkUnread marks a message as unread.
func (a *App) MarkUnread(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.MarkUnread(a.ctx, id)
}

// ListLabels returns the account's labels.
func (a *App) ListLabels() ([]desktop.Label, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListLabels(a.ctx)
}

// AIEnabled reports whether an LLM provider is configured.
func (a *App) AIEnabled() bool {
	if a.session == nil {
		return false
	}
	return a.session.API.AIEnabled()
}

// Summarize returns an AI-generated summary of a message.
func (a *App) Summarize(id string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.Summarize(a.ctx, id)
}

// SummarizeStream generates a summary and emits each token as a "summary:token"
// Wails runtime event; it returns the complete summary when done.
func (a *App) SummarizeStream(id string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.SummarizeStream(a.ctx, id, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, summaryTokenEvent, tok)
	})
}

// SendMail sends a new message from the active account.
func (a *App) SendMail(to, subject, body string, cc, bcc []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.SendMail(a.ctx, to, subject, body, cc, bcc)
}

// Reply sends a reply to an existing message.
func (a *App) Reply(originalID, body string, cc []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.Reply(a.ctx, originalID, body, cc)
}

// MessageLabelIDs returns the label IDs currently applied to a message.
func (a *App) MessageLabelIDs(id string) ([]string, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.MessageLabelIDs(a.ctx, id)
}

// ApplyLabel adds a label to a message.
func (a *App) ApplyLabel(messageID, labelID string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.ApplyLabel(a.ctx, messageID, labelID)
}

// RemoveLabel removes a label from a message.
func (a *App) RemoveLabel(messageID, labelID string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.RemoveLabel(a.ctx, messageID, labelID)
}

// ListAttachments returns the attachments of a message.
func (a *App) ListAttachments(id string) ([]desktop.Attachment, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListAttachments(a.ctx, id)
}

// DownloadAttachment saves an attachment to the download directory and returns
// the resulting path.
func (a *App) DownloadAttachment(messageID, attachmentID, filename string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.DownloadAttachment(a.ctx, messageID, attachmentID, filename)
}

// OpenAttachment opens a downloaded attachment with the system default app.
func (a *App) OpenAttachment(path string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.OpenAttachment(a.ctx, path)
}

// PromptsEnabled reports whether the AI prompt library is available.
func (a *App) PromptsEnabled() bool {
	if a.session == nil {
		return false
	}
	return a.session.API.PromptsEnabled()
}

// ListPrompts returns the saved AI prompt templates.
func (a *App) ListPrompts() ([]desktop.Prompt, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListPrompts(a.ctx)
}

// ApplyPromptStream applies a prompt to a message, emitting each token as a
// "prompt:token" runtime event, and returns the full result.
func (a *App) ApplyPromptStream(messageID string, promptID int) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ApplyPromptStream(a.ctx, messageID, promptID, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, promptTokenEvent, tok)
	})
}

// BulkArchive archives every message in ids.
func (a *App) BulkArchive(ids []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkArchive(a.ctx, ids)
}

// BulkTrash trashes every message in ids.
func (a *App) BulkTrash(ids []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkTrash(a.ctx, ids)
}

// BulkMarkRead marks every message in ids as read.
func (a *App) BulkMarkRead(ids []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkMarkRead(a.ctx, ids)
}

// BulkMarkUnread marks every message in ids as unread.
func (a *App) BulkMarkUnread(ids []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkMarkUnread(a.ctx, ids)
}

// BulkApplyLabel applies a label to every message in ids.
func (a *App) BulkApplyLabel(ids []string, labelID string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkApplyLabel(a.ctx, ids, labelID)
}

// BulkRemoveLabel removes a label from every message in ids.
func (a *App) BulkRemoveLabel(ids []string, labelID string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkRemoveLabel(a.ctx, ids, labelID)
}
