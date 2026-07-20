package desktop

import "github.com/ajramos/giztui/internal/version"

// Version returns the build version string (injected at build time, or "dev").
// Exposed so the desktop client can surface it the way the TUI does.
func Version() string { return version.GetVersion() }
