package desktop

import (
	"context"
	"fmt"

	"github.com/ajramos/giztui/internal/services"
)

// SlackEnabled reports whether the Slack integration is available.
func (a *API) SlackEnabled() bool { return a.slack != nil }

// ForwardToSlack forwards a message to the default configured Slack channel.
func (a *API) ForwardToSlack(ctx context.Context, id string) error {
	if a.slack == nil {
		return fmt.Errorf("Slack is not configured")
	}
	channels, err := a.slack.ListConfiguredChannels(ctx)
	if err != nil {
		return err
	}
	if len(channels) == 0 {
		return fmt.Errorf("no Slack channels configured")
	}
	ch := channels[0]
	for _, c := range channels {
		if c.Default {
			ch = c
			break
		}
	}
	format := "compact"
	if a.ai != nil {
		format = "summary"
	}
	return a.slack.ForwardEmail(ctx, id, services.SlackForwardOptions{
		ChannelID:   ch.ID,
		WebhookURL:  ch.WebhookURL,
		ChannelName: ch.Name,
		FormatStyle: format,
	})
}
