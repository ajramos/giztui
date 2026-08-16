package config

import (
	"fmt"
	"strings"
	"time"
)

// ConfigIssue is a single problem found by Config.Validate. Fatal marks a
// misconfiguration that will break a feature (e.g. LLM enabled with no model);
// non-fatal issues are suspicious-but-usable and surface as warnings.
type ConfigIssue struct {
	Field   string // dotted config path, e.g. "llm.model"
	Message string
	Fatal   bool
}

// ConfigValidationError aggregates EVERY problem found in a single validation
// pass, so a user can fix their whole config in one go instead of rerunning to
// discover the next error. It implements error; Error() lists only fatal issues.
type ConfigValidationError struct {
	Issues []ConfigIssue
}

func (e *ConfigValidationError) Error() string {
	fatal := e.Fatal()
	if len(fatal) == 0 {
		return "invalid configuration"
	}
	parts := make([]string, len(fatal))
	for i, is := range fatal {
		parts[i] = fmt.Sprintf("%s: %s", is.Field, is.Message)
	}
	return "invalid configuration: " + strings.Join(parts, "; ")
}

// Fatal returns the issues that make the config broken.
func (e *ConfigValidationError) Fatal() []ConfigIssue { return e.filter(true) }

// Warnings returns the non-fatal, suspicious-but-usable issues.
func (e *ConfigValidationError) Warnings() []ConfigIssue { return e.filter(false) }

// HasFatal reports whether any issue is fatal.
func (e *ConfigValidationError) HasFatal() bool { return len(e.Fatal()) > 0 }

func (e *ConfigValidationError) filter(fatal bool) []ConfigIssue {
	var out []ConfigIssue
	for _, is := range e.Issues {
		if is.Fatal == fatal {
			out = append(out, is)
		}
	}
	return out
}

// Validate checks the whole configuration in one pass and returns a
// *ConfigValidationError listing every problem (fatal and warning), or nil when
// the config is clean. It never mutates the config. Keyboard-shortcut conflicts
// are folded in as warnings so all config problems come through one channel.
func (c *Config) Validate() error {
	if c == nil {
		return &ConfigValidationError{Issues: []ConfigIssue{{Field: "config", Message: "config is nil", Fatal: true}}}
	}

	var issues []ConfigIssue
	add := func(field, msg string, fatal bool) {
		issues = append(issues, ConfigIssue{Field: field, Message: msg, Fatal: fatal})
	}

	// --- LLM ---------------------------------------------------------------
	if c.LLM.IsEnabled() {
		switch c.LLM.Provider {
		case "", "ollama":
			if strings.TrimSpace(c.LLM.Endpoint) == "" {
				add("llm.endpoint", "ollama provider needs an endpoint (e.g. http://localhost:11434)", false)
			}
			if strings.TrimSpace(c.LLM.Model) == "" {
				add("llm.model", "LLM is enabled but no model is set", true)
			}
		case "bedrock":
			if strings.TrimSpace(c.LLM.Model) == "" {
				add("llm.model", "bedrock provider needs a model id", true)
			}
			if strings.TrimSpace(c.LLM.Region) == "" && strings.TrimSpace(c.LLM.Endpoint) == "" {
				add("llm.region", "bedrock provider needs a region (region or endpoint)", false)
			}
		default:
			add("llm.provider", fmt.Sprintf("unknown provider %q; it falls back to ollama (valid: ollama, bedrock)", c.LLM.Provider), false)
		}
		if strings.TrimSpace(c.LLM.Timeout) != "" {
			if _, err := time.ParseDuration(c.LLM.Timeout); err != nil {
				add("llm.timeout", fmt.Sprintf("invalid duration %q (expected e.g. \"20s\"): %v", c.LLM.Timeout, err), true)
			}
		}
		if c.LLM.ChatMaxBodyChars < 0 {
			add("llm.chat_max_body_chars", "must be >= 0 (0 = built-in default)", false)
		}
		if c.LLM.ChatMaxTurns < 0 {
			add("llm.chat_max_turns", "must be >= 0 (0 = built-in default)", false)
		}
	}

	// --- Obsidian: the pointer must be nil OR internally consistent --------
	// (mirrors the IsObsidianEnabled defensive pattern that fixed the #6 panic).
	if c.Obsidian != nil && c.Obsidian.Enabled {
		if strings.TrimSpace(c.Obsidian.VaultPath) == "" {
			add("obsidian.vault_path", "Obsidian is enabled but vault_path is empty", true)
		}
	}

	// --- Numeric ranges (0 means "use the built-in default"; negatives are wrong)
	if c.Attachments.MaxDownloadSize < 0 {
		add("attachments.max_download_size", "must be >= 0", false)
	}
	if c.Threading.MaxThreadDepth < 0 {
		add("threading.max_thread_depth", "must be >= 0", false)
	}
	if c.InboxAnalyzer.MaxBatches < 0 {
		add("inbox_analyzer.max_batches", "must be >= 0", false)
	}
	if c.Layout.MaxRecipientLines < 0 {
		add("layout.max_recipient_lines", "must be >= 0", false)
	}
	if c.Keys.VimNavigationTimeoutMs < 0 {
		add("keys.vim_navigation_timeout_ms", "must be >= 0", false)
	}
	if c.Keys.VimRangeTimeoutMs < 0 {
		add("keys.vim_range_timeout_ms", "must be >= 0", false)
	}

	// --- Keyboard shortcut conflicts (folded in as warnings) ---------------
	for _, w := range ValidateKeyboardConfig(c.Keys) {
		add("keys", w, false)
	}

	if len(issues) == 0 {
		return nil
	}
	return &ConfigValidationError{Issues: issues}
}
