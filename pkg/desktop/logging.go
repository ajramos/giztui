package desktop

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ajramos/giztui/internal/config"
)

// DefaultLogPath returns where the desktop client writes its log file — next to
// the config, so it's easy to find: ~/.config/giztui/desktop.log.
func DefaultLogPath() string {
	cfg := config.DefaultConfigPath()
	if cfg == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(cfg), "desktop.log")
}

// SetupFileLogging points the standard logger at the desktop log file (appending)
// while still writing to stderr for `wails dev`. It returns the resolved path so
// the caller can surface it. Best-effort: on any error it leaves logging as-is.
func SetupFileLogging() (string, error) {
	path := DefaultLogPath()
	if path == "" {
		return "", os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return "", err
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	return path, nil
}
