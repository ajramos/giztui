package main

// App bindings: threading, Obsidian/Slack, labels/move, links, save, drafts. Split out of app.go.

import (
	"fmt"

	"github.com/ajramos/giztui/pkg/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ThreadingEnabled() bool {
	return a.enabled((*desktop.API).ThreadingEnabled)
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
	return a.enabled((*desktop.API).ObsidianEnabled)
}

// SendToObsidian ingests a message into the Obsidian vault with an optional
// comment (rendered into the note as "> **Note:** …", TUI parity).
func (a *App) SendToObsidian(id, comment string) (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.SendToObsidian(a.ctx, id, comment)
}

// SlackEnabled reports whether the Slack integration is available.
func (a *App) SlackEnabled() bool {
	return a.enabled((*desktop.API).SlackEnabled)
}

// SlackChannels returns the configured Slack channels for the forward picker.
func (a *App) SlackChannels() ([]desktop.SlackChannelInfo, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.SlackChannels(a.ctx)
}

// SlackDefaultFormat returns the configured slack.defaults.format_style so the
// forward picker can preselect it.
func (a *App) SlackDefaultFormat() string {
	if s := a.session.Load(); s != nil && s.Config != nil {
		if f := s.Config.Slack.Defaults.FormatStyle; f != "" {
			return f
		}
	}
	return "summary"
}

// ForwardToSlack forwards a message to the chosen Slack channel with an optional
// pre-message. format selects the style ("summary"/"markdown"/"compact"/"full"/
// "raw"); an empty format falls back to the configured slack.defaults.format_style.
func (a *App) ForwardToSlack(id, channelID, userMessage, format string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	if format == "" {
		if s := a.session.Load(); s != nil && s.Config != nil {
			format = s.Config.Slack.Defaults.FormatStyle
		}
	}
	return api.ForwardToSlack(a.ctx, id, channelID, userMessage, format)
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

// BulkMoveToLabel moves every message in ids to a label (apply label + archive).
func (a *App) BulkMoveToLabel(ids []string, name string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkMoveToLabel(a.ctx, ids, name)
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
