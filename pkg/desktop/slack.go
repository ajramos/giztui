package desktop

import (
	"context"
	"fmt"

	"github.com/ajramos/giztui/internal/services"
)

// SlackEnabled reports whether the Slack integration is available.
func (a *API) SlackEnabled() bool { return a.slack != nil }

// SlackChannels returns the configured Slack channels for the forward picker so
// the desktop can offer the same channel choice the TUI does (instead of always
// forwarding to the default).
func (a *API) SlackChannels(ctx context.Context) ([]SlackChannelInfo, error) {
	if a.slack == nil {
		return nil, fmt.Errorf("the Slack integration is not configured")
	}
	channels, err := a.slack.ListConfiguredChannels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SlackChannelInfo, 0, len(channels))
	for _, c := range channels {
		out = append(out, SlackChannelInfo{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			Default:     c.Default,
		})
	}
	return out, nil
}

// ForwardToSlack forwards a message to a Slack channel. channelID selects the
// target (empty → the default channel, else the first configured). formatStyle
// mirrors the TUI's slack.defaults.format_style ("summary"/"compact"/"full"/"raw");
// empty falls back to "summary" when AI is available, else "compact". userMessage
// is an optional pre-message prepended to the post.
//
// For the "full" style we populate ProcessedContent with the rendered-visible body
// (readableBody), because the SlackService's full formatter posts
// "⚠️ Processed content not available" when that field is empty — which is exactly
// what a desktop "full" forward produced before this.
func (a *API) ForwardToSlack(ctx context.Context, id, channelID, userMessage, formatStyle string) error {
	if a.slack == nil {
		return fmt.Errorf("the Slack integration is not configured")
	}
	channels, err := a.slack.ListConfiguredChannels(ctx)
	if err != nil {
		return err
	}
	if len(channels) == 0 {
		return fmt.Errorf("no Slack channels configured")
	}

	// Pick the requested channel; fall back to the default, then the first.
	ch := channels[0]
	for _, c := range channels {
		if c.Default {
			ch = c
		}
	}
	if channelID != "" {
		for _, c := range channels {
			if c.ID == channelID {
				ch = c
				break
			}
		}
	}

	if formatStyle == "" {
		formatStyle = "compact"
		if a.ai != nil {
			formatStyle = "summary"
		}
	}

	opts := services.SlackForwardOptions{
		ChannelID:   ch.ID,
		WebhookURL:  ch.WebhookURL,
		ChannelName: ch.Name,
		UserMessage: userMessage,
		FormatStyle: formatStyle,
	}

	// "full" renders the reader-visible body; without it the formatter emits a
	// "not available" placeholder. Prefer HTML→text so hidden preheaders don't leak.
	if formatStyle == "full" {
		if msg, gerr := a.repo.GetMessage(ctx, id); gerr == nil {
			opts.ProcessedContent = readableBody(msg.PlainText, msg.HTML)
		}
	}

	return a.slack.ForwardEmail(ctx, id, opts)
}
