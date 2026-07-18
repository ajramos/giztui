package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SaveMessage writes a message to a .txt file in the download directory and
// returns the path.
func (a *API) SaveMessage(ctx context.Context, id string) (string, error) {
	dir := a.downloadDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	name := "message-" + id
	if msg, err := a.repo.GetMessage(ctx, id); err == nil && msg != nil {
		if s := sanitizeFilename(msg.Subject); s != "" {
			name = s
		}
	}
	path := filepath.Join(dir, name+".txt")
	if err := a.email.SaveMessageToFile(ctx, id, path); err != nil {
		return "", err
	}
	return path, nil
}

// SaveRawMessage writes the full raw message (.eml) to the download directory
// and returns the path.
func (a *API) SaveRawMessage(ctx context.Context, id string) (string, error) {
	if a.draft == nil {
		return "", fmt.Errorf("gmail client not available")
	}
	raw, err := a.draft.GetMessageRaw(id)
	if err != nil {
		return "", err
	}
	dir := a.downloadDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	name := "message-" + id
	if msg, err := a.repo.GetMessage(ctx, id); err == nil && msg != nil {
		if s := sanitizeFilename(msg.Subject); s != "" {
			name = s
		}
	}
	path := filepath.Join(dir, name+".eml")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return path, nil
}

// downloadDir returns the configured attachment download directory (or a sane
// default) to reuse for saved messages.
func (a *API) downloadDir() string {
	if a.attach != nil {
		if p := a.attach.GetDefaultDownloadPath(); p != "" {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Downloads")
	}
	return "."
}

// sanitizeFilename strips characters unsafe for filenames and truncates.
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "", "?", "",
		"\"", "", "<", "", ">", "", "|", "", "\n", " ", "\r", " ")
	s = replacer.Replace(s)
	if len(s) > 80 {
		s = s[:80]
	}
	return strings.TrimSpace(s)
}
