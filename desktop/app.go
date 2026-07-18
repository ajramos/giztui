package main

import (
	"context"
	"log"

	"github.com/ajramos/giztui/pkg/desktop"
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
