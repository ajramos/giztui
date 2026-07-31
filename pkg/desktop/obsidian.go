package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/obsidian"
)

// ObsidianEnabled reports whether the Obsidian integration is available.
func (a *API) ObsidianEnabled() bool { return a.obsidian != nil }

// SendToObsidian ingests a message into the configured Obsidian vault. comment is
// an optional pre-message rendered into the note as "> **Note:** <comment>" (TUI
// parity — the desktop previously dropped it).
func (a *API) SendToObsidian(ctx context.Context, id, comment string) (string, error) {
	if a.obsidian == nil {
		return "", fmt.Errorf("the Obsidian integration is not configured")
	}
	msg, err := a.repo.GetMessage(ctx, id)
	if err != nil {
		return "", err
	}
	opts := obsidian.ObsidianOptions{AccountEmail: a.accountEmail}
	if strings.TrimSpace(comment) != "" {
		opts.CustomMetadata = map[string]interface{}{"comment": comment}
	}
	res, err := a.obsidian.IngestEmailToObsidian(ctx, msg, opts)
	if err != nil {
		return "", err
	}
	if res != nil {
		return res.FilePath, nil
	}
	return "", nil
}
