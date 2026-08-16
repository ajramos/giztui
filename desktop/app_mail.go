package main

// App bindings: keymap, accounts, mailbox core (list/search/message/archive/trash/read/labels), AI. Split out of app.go.

import (
	"fmt"

	"github.com/ajramos/giztui/pkg/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) KeyMap() desktop.KeyMap {
	if s := a.session.Load(); s != nil {
		return s.KeyMap()
	}
	return desktop.DefaultKeyMap()
}

// JobsNotifyOnComplete reports whether the AI-jobs subsystem should toast when a
// background job finishes (config keys jobs.notify_on_complete, default true).
func (a *App) JobsNotifyOnComplete() bool {
	if s := a.session.Load(); s != nil && s.Config != nil {
		return s.Config.Jobs.NotifyOnComplete
	}
	return true
}

// ListAccounts returns all configured accounts for the account switcher.
func (a *App) ListAccounts() ([]desktop.AccountInfo, error) {
	s := a.session.Load()
	if s == nil {
		return nil, a.notReady()
	}
	return s.ListAccounts(a.ctx)
}

// SwitchAccount switches the active account and rebuilds the service stack.
func (a *App) SwitchAccount(id string) error {
	s := a.session.Load()
	if s == nil {
		return a.notReady()
	}
	return s.SwitchAccount(a.ctx, id)
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

// Star adds the STARRED label to a message.
func (a *App) Star(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.Star(a.ctx, id)
}

// Unstar removes the STARRED label from a message.
func (a *App) Unstar(id string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.Unstar(a.ctx, id)
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
	return a.enabled((*desktop.API).AIEnabled)
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

// ChatEnabled reports whether the "chat with this email" feature is available.
func (a *App) ChatEnabled() bool {
	api, err := a.api()
	if err != nil {
		return false
	}
	return api.ChatEnabled()
}

// ChatStream answers a user's message about message `id`, emitting each token as
// a "chat:token" Wails event and returning the full reply when done.
func (a *App) ChatStream(id string, message string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ChatStream(a.ctx, id, message, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, chatTokenEvent, tok)
	})
}

// ChatReset clears the chat history for message `id`.
func (a *App) ChatReset(id string) {
	if api, err := a.api(); err == nil {
		api.ChatReset(id)
	}
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
	return a.enabled((*desktop.API).RSVPEnabled)
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
	s := a.session.Load()
	if s == nil || s.Config == nil {
		return desktop.ConfigInfo{}
	}
	cfg := s.Config
	// Show the active account's effective engine, not the global config.
	eff := cfg.EffectiveLLM(s.CurrentAccountID())
	info := desktop.ConfigInfo{
		ConfigPath:  s.ConfigPath(),
		LogPath:     desktop.DefaultLogPath(),
		LLMProvider: eff.Provider,
		LLMModel:    eff.Model,
		Theme:       cfg.Theme.Current,
		SlackOn:     cfg.Slack.Enabled,
		AutoRefresh: cfg.AutoRefresh.Enabled,
	}
	if cfg.Obsidian != nil {
		info.ObsidianOn = cfg.Obsidian.Enabled
	}
	// Subscription providers (chatgpt) need an interactive login; surface the
	// state so the ConfigModal can offer Login/Logout.
	if eff.Enabled && eff.Provider == "chatgpt" {
		info.LLMNeedsLogin = true
		info.LLMLoggedIn = desktop.ChatGPTLoggedIn()
	}
	if email, err := s.AccountEmail(a.ctx); err == nil {
		info.Account = email
	}
	if api := s.API; api != nil {
		info.DownloadPath = api.DownloadDir()
	}
	return info
}

// MigrateConfig runs the config self-migration against this session's config
// file (adds missing default keys, prunes obsolete ones, writes a .bak first),
// mirroring the TUI's ":config migrate". Returns a human-readable summary the UI
// can toast.
func (a *App) MigrateConfig() (string, error) {
	s := a.session.Load()
	if s == nil {
		return "", a.notReady()
	}
	added, removed, backup, err := s.MigrateConfig()
	if err != nil {
		return "", err
	}
	if len(added) == 0 && len(removed) == 0 {
		return "Config is already up to date", nil
	}
	return fmt.Sprintf("Config updated: +%d added, -%d removed (backup: %s). Restart to apply.", len(added), len(removed), backup), nil
}

// LLMLogin runs the ChatGPT subscription OAuth (PKCE) flow, mirroring the TUI's
// ":llm login chatgpt". It copies the authorization URL to the clipboard (so the
// user can paste it into whatever browser/profile they want, instead of forcing
// the system-default browser) and then blocks until the login callback completes
// (or errors), persisting the machine-global token. The binding call returns only
// when the flow finishes, so the frontend can await it and refresh ConfigInfo
// afterward. Credentials are shared by every account that selects the "chatgpt"
// provider — one login per machine.
func (a *App) LLMLogin() error {
	model := ""
	if s := a.session.Load(); s != nil && s.Config != nil {
		model = s.Config.EffectiveLLM(s.CurrentAccountID()).Model
	}
	authURL, wait, err := desktop.ChatGPTStartLogin(a.ctx, model)
	if err != nil {
		return err
	}
	// Do both: open the default browser AND copy the URL to the clipboard, so the
	// user can use the browser that opened or paste it into a different one.
	_ = desktop.OpenLoginBrowser(authURL)
	_ = wailsruntime.ClipboardSetText(a.ctx, authURL)
	return wait()
}

// LLMLogout removes the stored ChatGPT subscription tokens (":llm logout chatgpt").
func (a *App) LLMLogout() error {
	return desktop.ChatGPTLogout()
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

// SetAutoRefreshEnabled persists the auto-refresh on/off choice to config.json so it survives a
// restart. The desktop never sends the Slack new-mail digest itself, but writing enabled:false here
// stops any TUI launched later from silently re-arming it from a stale enabled:true.
func (a *App) SetAutoRefreshEnabled(enabled bool) error {
	s := a.session.Load()
	if s == nil {
		return a.notReady()
	}
	return s.SetAutoRefreshEnabled(enabled)
}

// AutoRefreshSettings returns the configured inbox auto-refresh preference.
func (a *App) AutoRefreshSettings() desktop.AutoRefreshSettings {
	s := a.session.Load()
	if s == nil || s.Config == nil {
		return desktop.AutoRefreshSettings{Enabled: false, IntervalSeconds: 300}
	}
	cfg := s.Config.AutoRefresh
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

// FetchInlineImage returns an inline attachment as a data: URI so the reader can
// render cid: image references embedded in the HTML body.
func (a *App) FetchInlineImage(messageID, attachmentID string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.FetchInlineImage(a.ctx, messageID, attachmentID)
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
	return a.enabled((*desktop.API).PromptsEnabled)
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
func (a *App) ApplyPromptStream(messageID string, promptID int, force bool) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ApplyPromptStream(a.ctx, messageID, promptID, force, func(tok string) {
		wailsruntime.EventsEmit(a.ctx, promptTokenEvent, tok)
	})
}

// CachedPrompts returns persisted prompt results for a message so the reader can
// restore its AI panels across sessions.
func (a *App) CachedPrompts(messageID string) ([]desktop.CachedPromptResult, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.CachedPrompts(a.ctx, messageID)
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
