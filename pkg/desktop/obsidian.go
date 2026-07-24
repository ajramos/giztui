package desktop

import (
	"context"
	"fmt"

	"github.com/ajramos/giztui/internal/obsidian"
)

// ObsidianEnabled reports whether the Obsidian integration is available.
func (a *API) ObsidianEnabled() bool { return a.obsidian != nil }

// SendToObsidian ingests a message into the configured Obsidian vault.
func (a *API) SendToObsidian(ctx context.Context, id string) (string, error) {
	if a.obsidian == nil {
		return "", fmt.Errorf("the Obsidian integration is not configured")
	}
	msg, err := a.repo.GetMessage(ctx, id)
	if err != nil {
		return "", err
	}
	res, err := a.obsidian.IngestEmailToObsidian(ctx, msg, obsidian.ObsidianOptions{
		AccountEmail: a.accountEmail,
	})
	if err != nil {
		return "", err
	}
	if res != nil {
		return res.FilePath, nil
	}
	return "", nil
}
