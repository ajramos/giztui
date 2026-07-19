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
	if path, err := desktop.SetupFileLogging(); err == nil {
		log.Printf("desktop: logging to %s", path)
	}
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

// ThemesEnabled reports whether theming is available.
func (a *App) ThemesEnabled() bool {
	return a.session != nil && a.session.API.ThemesEnabled()
}

// ListThemes returns the available theme names.
func (a *App) ListThemes() ([]string, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListThemes(a.ctx)
}

// GetThemeColors returns a theme's palette (empty name = current theme).
func (a *App) GetThemeColors(name string) (*desktop.ThemeColors, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.GetThemeColors(a.ctx, name)
}

// ApplyBulkPromptStream applies a prompt across many messages, streaming tokens
// via the "prompt:token" event.
func (a *App) ApplyBulkPromptStream(ids []string, promptID int) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ApplyBulkPromptStream(a.ctx, ids, promptID, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, promptTokenEvent, tok)
	})
}

// ActionPlanEnabled reports whether the AI inbox action plan is available.
func (a *App) ActionPlanEnabled() bool {
	return a.session != nil && a.session.API.ActionPlanEnabled()
}

// AnalyzeInbox runs the AI inbox analyzer and returns an action plan.
func (a *App) AnalyzeInbox(inputs []desktop.AnalyzerInput) (*desktop.ActionPlanResult, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.AnalyzeInbox(a.ctx, inputs)
}

// BulkApplyLabelByName applies a label by name to many messages.
func (a *App) BulkApplyLabelByName(ids []string, name string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkApplyLabelByName(a.ctx, ids, name)
}

// AnalyzerRulesEnabled reports whether analyzer preference rules are available.
func (a *App) AnalyzerRulesEnabled() bool {
	return a.session != nil && a.session.API.AnalyzerRulesEnabled()
}

// ListAnalyzerRules returns the stored analyzer preference rules.
func (a *App) ListAnalyzerRules() ([]desktop.AnalyzerRule, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListAnalyzerRules(a.ctx)
}

// SaveAnalyzerRule persists a new analyzer preference rule.
func (a *App) SaveAnalyzerRule(text string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.SaveAnalyzerRule(a.ctx, text)
}

// DeleteAnalyzerRule removes a stored analyzer preference rule.
func (a *App) DeleteAnalyzerRule(id int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.DeleteAnalyzerRule(a.ctx, id)
}

// ViewAnalyzerPrompt returns the effective analyzer prompt for inspection.
func (a *App) ViewAnalyzerPrompt() (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ViewAnalyzerPrompt(a.ctx)
}

// SavedQueriesEnabled reports whether saved searches are available.
func (a *App) SavedQueriesEnabled() bool {
	return a.session != nil && a.session.API.SavedQueriesEnabled()
}

// ListSavedQueries returns the saved searches.
func (a *App) ListSavedQueries() ([]desktop.SavedQuery, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListSavedQueries(a.ctx)
}

// SaveQuery persists a named Gmail search.
func (a *App) SaveQuery(name, query string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.SaveQuery(a.ctx, name, query)
}

// DeleteSavedQuery removes a saved search.
func (a *App) DeleteSavedQuery(id int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.DeleteSavedQuery(a.ctx, id)
}

// RecordQueryUse bumps a saved query's usage counter.
func (a *App) RecordQueryUse(id int64) {
	if api, err := a.api(); err == nil {
		api.RecordQueryUse(a.ctx, id)
	}
}

// ThreadingEnabled reports whether conversation features are available.
func (a *App) ThreadingEnabled() bool {
	return a.session != nil && a.session.API.ThreadingEnabled()
}

// GetThread returns all messages in a thread for the conversation view.
func (a *App) GetThread(threadID string) ([]desktop.MessageDetail, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.GetThread(a.ctx, threadID)
}

// ThreadSummaryStream streams an AI summary of a conversation via the
// "summary:token" runtime event.
func (a *App) ThreadSummaryStream(threadID string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ThreadSummaryStream(a.ctx, threadID, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, summaryTokenEvent, tok)
	})
}

// ObsidianEnabled reports whether the Obsidian integration is available.
func (a *App) ObsidianEnabled() bool {
	return a.session != nil && a.session.API.ObsidianEnabled()
}

// SendToObsidian ingests a message into the Obsidian vault.
func (a *App) SendToObsidian(id string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.SendToObsidian(a.ctx, id)
}

// SlackEnabled reports whether the Slack integration is available.
func (a *App) SlackEnabled() bool {
	return a.session != nil && a.session.API.SlackEnabled()
}

// ForwardToSlack forwards a message to the default Slack channel.
func (a *App) ForwardToSlack(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.ForwardToSlack(a.ctx, id)
}

// SuggestLabels returns AI-suggested labels for a message.
func (a *App) SuggestLabels(id string) ([]string, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.SuggestLabels(a.ctx, id)
}

// ApplyLabelByName applies a label by name, creating it if needed.
func (a *App) ApplyLabelByName(messageID, name string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.ApplyLabelByName(a.ctx, messageID, name)
}

// MoveToLabel moves a message to a label (apply label + archive).
func (a *App) MoveToLabel(messageID, name string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.MoveToLabel(a.ctx, messageID, name)
}

// Unarchive puts a message back in the inbox (undo of Archive).
func (a *App) Unarchive(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.Unarchive(a.ctx, id)
}

// Untrash restores a message from the trash (undo of Trash).
func (a *App) Untrash(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.Untrash(a.ctx, id)
}

// BulkUnarchive re-applies INBOX to many messages (undo of a bulk archive).
func (a *App) BulkUnarchive(ids []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkUnarchive(a.ctx, ids)
}

// BulkUntrash restores many messages from the trash (undo of a bulk trash).
func (a *App) BulkUntrash(ids []string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkUntrash(a.ctx, ids)
}

// ListLinks returns the links found in a message body.
func (a *App) ListLinks(id string) ([]desktop.Link, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListLinks(a.ctx, id)
}

// OpenURL opens an arbitrary URL in the system browser.
func (a *App) OpenURL(url string) {
	if url != "" {
		wailsruntime.BrowserOpenURL(a.ctx, url)
	}
}

// SaveMessage saves a message to a text file and returns the path.
func (a *App) SaveMessage(id string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.SaveMessage(a.ctx, id)
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

// KeyMap returns the user's configured keyboard shortcuts (or defaults).
func (a *App) KeyMap() desktop.KeyMap {
	if a.session == nil {
		return desktop.DefaultKeyMap()
	}
	return a.session.KeyMap()
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
// Wails runtime event; it returns the complete summary when done. When force is
// true it bypasses the cache and regenerates the summary.
func (a *App) SummarizeStream(id string, force bool) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.SummarizeStream(a.ctx, id, force, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, summaryTokenEvent, tok)
	})
}

// GenerateReply drafts an AI reply to a message and returns the draft body.
func (a *App) GenerateReply(id string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.GenerateReply(a.ctx, id)
}

// TouchUp reformats a message's body with the AI for readability.
func (a *App) TouchUp(id string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.TouchUp(a.ctx, id)
}

// SaveRawMessage writes the full raw message (.eml) to disk and returns the path.
func (a *App) SaveRawMessage(id string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.SaveRawMessage(a.ctx, id)
}

// RSVPEnabled reports whether calendar RSVP is available.
func (a *App) RSVPEnabled() bool {
	return a.session != nil && a.session.API.RSVPEnabled()
}

// UsageStats returns AI prompt usage statistics.
func (a *App) UsageStats() (*desktop.UsageStats, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.UsageStats(a.ctx)
}

// ClearCaches clears the AI summary and prompt caches for the active account.
func (a *App) ClearCaches() error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.ClearCaches(a.ctx)
}

// ConfigInfo returns a read-only snapshot of the effective configuration.
func (a *App) ConfigInfo() desktop.ConfigInfo {
	if a.session == nil || a.session.Config == nil {
		return desktop.ConfigInfo{}
	}
	cfg := a.session.Config
	info := desktop.ConfigInfo{
		ConfigPath:  a.session.ConfigPath(),
		LogPath:     desktop.DefaultLogPath(),
		LLMProvider: cfg.LLM.Provider,
		LLMModel:    cfg.LLM.Model,
		Theme:       cfg.Theme.Current,
		SlackOn:     cfg.Slack.Enabled,
		AutoRefresh: cfg.AutoRefresh.Enabled,
	}
	if cfg.Obsidian != nil {
		info.ObsidianOn = cfg.Obsidian.Enabled
	}
	if email, err := a.session.AccountEmail(a.ctx); err == nil {
		info.Account = email
	}
	if api, err := a.api(); err == nil {
		info.DownloadPath = api.DownloadDir()
	}
	return info
}

// InviteInfo returns calendar-invite details for a message (IsInvite=false when
// the message isn't an invitation).
func (a *App) InviteInfo(id string) (*desktop.Invite, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.InviteInfo(a.ctx, id)
}

// RespondInvite sets the account's attendance (accepted/declined/tentative).
func (a *App) RespondInvite(id, status string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.RespondInvite(a.ctx, id, status)
}

// AutoRefreshSettings returns the configured inbox auto-refresh preference.
func (a *App) AutoRefreshSettings() desktop.AutoRefreshSettings {
	if a.session == nil || a.session.Config == nil {
		return desktop.AutoRefreshSettings{Enabled: false, IntervalSeconds: 300}
	}
	cfg := a.session.Config.AutoRefresh
	return desktop.AutoRefreshSettings{
		Enabled:         cfg.Enabled,
		IntervalSeconds: int(cfg.ResolvedInterval().Seconds()),
	}
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

// GetPrompt returns a prompt template including its editable text.
func (a *App) GetPrompt(id int) (*desktop.PromptDetail, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.GetPrompt(a.ctx, id)
}

// CreatePrompt saves a new prompt template and returns its id.
func (a *App) CreatePrompt(name, description, text, category string) (int, error) {
	api, err := a.api()
	if err != nil {
		return 0, err
	}
	return api.CreatePrompt(a.ctx, name, description, text, category)
}

// UpdatePrompt edits an existing prompt template.
func (a *App) UpdatePrompt(id int, name, description, text, category string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.UpdatePrompt(a.ctx, id, name, description, text, category)
}

// DeletePrompt removes a prompt template.
func (a *App) DeletePrompt(id int) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.DeletePrompt(a.ctx, id)
}

// RefinePromptText asks the AI to improve a prompt's text.
func (a *App) RefinePromptText(text string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.RefinePromptText(a.ctx, text)
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
