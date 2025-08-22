package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/obsidian"
)

// LLMConfig holds all LLM-related configuration
type LLMConfig struct {
	// Core LLM settings
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"` // ollama, openai, anthropic, bedrock, custom
	Model    string `json:"model"`
	Endpoint string `json:"endpoint"`
	Region   string `json:"region"` // For AWS Bedrock
	APIKey   string `json:"api_key"`
	Timeout  string `json:"timeout"`

	// Streaming configuration
	StreamEnabled bool `json:"stream_enabled"`
	StreamChunkMs int  `json:"stream_chunk_ms"`

	// Caching configuration
	CacheEnabled bool   `json:"cache_enabled"`
	CachePath    string `json:"cache_path"`

	// Template file paths (relative to config dir or absolute)
	SummarizeTemplate string `json:"summarize_template"`
	ReplyTemplate     string `json:"reply_template"`
	LabelTemplate     string `json:"label_template"`
	TouchUpTemplate   string `json:"touch_up_template"`

	// Inline prompt overrides (optional - takes precedence over files)
	SummarizePrompt string `json:"summarize_prompt,omitempty"`
	ReplyPrompt     string `json:"reply_prompt,omitempty"`
	LabelPrompt     string `json:"label_prompt,omitempty"`
	// Touch-up prompt for LLM whitespace/line-break adjustments (no semantic changes)
	TouchUpPrompt string `json:"touch_up_prompt,omitempty"`
}

// Config holds all configuration for the Gmail TUI application
type Config struct {
	Credentials string `json:"credentials"`
	Token       string `json:"token"`

	// LLM configuration (unified)
	LLM LLMConfig `json:"llm"`

	// Slack integration
	Slack SlackConfig `json:"slack"`

	// Layout configuration
	Layout LayoutConfig `json:"layout"`

	// Keyboard shortcuts
	Keys KeyBindings `json:"keys"`

	// Logging
	LogFile string `json:"log_file"`

	// Obsidian integration
	Obsidian *obsidian.ObsidianConfig `json:"obsidian"`
}

// SlackConfig contains all Slack integration settings
type SlackConfig struct {
	// Enabled controls whether Slack integration is available
	Enabled bool `json:"enabled"`

	// Channels defines the list of available Slack channels for forwarding
	Channels []SlackChannel `json:"channels"`

	// Defaults specifies default behavior for email forwarding
	Defaults SlackDefaults `json:"defaults"`

	// Template file path for summary prompt (relative to config dir or absolute)
	SummaryTemplate string `json:"summary_template"`

	// Inline prompt override (optional - takes precedence over file)
	// Available variables: {{body}}, {{subject}}, {{from}}, {{to}}, {{cc}}, {{bcc}},
	// {{date}}, {{reply-to}}, {{message-id}}, {{in-reply-to}}, {{references}}, {{max_words}}
	SummaryPrompt string `json:"summary_prompt,omitempty"`
}

// SlackChannel defines a Slack channel configuration
type SlackChannel struct {
	// ID is a unique internal identifier for the channel
	ID string `json:"id"`

	// Name is the display name shown in the UI (e.g., "team-updates", "personal-dm")
	Name string `json:"name"`

	// WebhookURL is the Slack webhook URL for posting messages to this channel
	WebhookURL string `json:"webhook_url"`

	// Default indicates if this channel should be pre-selected in the UI
	Default bool `json:"default"`

	// Description provides optional additional context for the channel
	Description string `json:"description"`
}

// SlackDefaults defines default Slack forwarding behavior
type SlackDefaults struct {
	// FormatStyle controls how emails are formatted: "summary" (AI-generated), "compact" (headers + preview), "full" (TUI processed), "raw" (minimal processing)
	FormatStyle string `json:"format_style"`
}

// LayoutConfig defines layout-specific configuration
type LayoutConfig struct {
	// Auto-resize settings
	AutoResize bool `json:"auto_resize"`

	// Layout breakpoints (minimum dimensions)
	WideBreakpoint   LayoutBreakpoint `json:"wide_breakpoint"`
	MediumBreakpoint LayoutBreakpoint `json:"medium_breakpoint"`
	NarrowBreakpoint LayoutBreakpoint `json:"narrow_breakpoint"`

	// Layout preferences
	DefaultLayout string `json:"default_layout"`

	// UI customization
	ShowBorders    bool   `json:"show_borders"`
	ShowTitles     bool   `json:"show_titles"`
	CompactMode    bool   `json:"compact_mode"`
	CurrentTheme   string `json:"current_theme"`      // Active theme name (e.g., "gmail-dark")
	CustomThemeDir string `json:"custom_theme_dir"`   // Custom themes directory (empty = default)
}

// LayoutBreakpoint defines minimum dimensions for layout types
type LayoutBreakpoint struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// KeyBindings defines keyboard shortcuts for the TUI
type KeyBindings struct {
	// Core email operations
	Summarize     string `json:"summarize"`
	GenerateReply string `json:"generate_reply"`
	SuggestLabel  string `json:"suggest_label"`
	Reply         string `json:"reply"`
	Compose       string `json:"compose"`
	Refresh       string `json:"refresh"`
	Search        string `json:"search"`
	Unread        string `json:"unread"`
	ToggleRead    string `json:"toggle_read"`
	Trash         string `json:"trash"`
	Archive       string `json:"archive"`
	Move          string `json:"move"`
	Prompt        string `json:"prompt"`
	Drafts        string `json:"drafts"`
	Attachments   string `json:"attachments"`
	ManageLabels  string `json:"manage_labels"`
	Quit          string `json:"quit"`

	// Additional configurable shortcuts
	Obsidian    string `json:"obsidian"`     // Send to Obsidian
	Slack       string `json:"slack"`        // Forward to Slack
	Markdown    string `json:"markdown"`     // Toggle markdown
	SaveMessage string `json:"save_message"` // Save message to file
	SaveRaw     string `json:"save_raw"`     // Save raw EML
	RSVP        string `json:"rsvp"`         // Toggle RSVP panel
	LinkPicker  string `json:"link_picker"`  // Open link picker
	ThemePicker string `json:"theme_picker"` // Open theme picker
	OpenGmail   string `json:"open_gmail"`   // Open message in Gmail web UI
	BulkMode    string `json:"bulk_mode"`    // Toggle bulk mode
	BulkSelect  string `json:"bulk_select"`  // Select/deselect message in bulk mode
	CommandMode string `json:"command_mode"` // Open command bar
	Help          string `json:"help"`           // Toggle help
	LoadMore      string `json:"load_more"`      // Load next 50 messages
	ToggleHeaders string `json:"toggle_headers"` // Toggle header visibility

	// VIM sequence timeouts (in milliseconds)
	VimNavigationTimeoutMs int `json:"vim_navigation_timeout_ms"` // Timeout for gg navigation (default: 1000ms)
	VimRangeTimeoutMs      int `json:"vim_range_timeout_ms"`      // Timeout for bulk operations like d3d (default: 2000ms)

	// Content navigation shortcuts (when focused on text view)
	ContentSearch string `json:"content_search"` // Start search within message content
	SearchNext    string `json:"search_next"`    // Jump to next search match
	SearchPrev    string `json:"search_prev"`    // Jump to previous search match
	FastUp        string `json:"fast_up"`        // Fast navigation up (paragraph jump)
	FastDown      string `json:"fast_down"`      // Fast navigation down (paragraph jump)
	WordLeft      string `json:"word_left"`      // Word-wise navigation left
	WordRight     string `json:"word_right"`     // Word-wise navigation right
	GotoTop       string `json:"goto_top"`       // Jump to top of content
	GotoBottom    string `json:"goto_bottom"`    // Jump to bottom of content
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		LLM:     DefaultLLMConfig(),
		Slack:   DefaultSlackConfig(),
		Layout:  DefaultLayoutConfig(),
		Keys:    DefaultKeyBindings(),
		LogFile: "",
	}
}

// DefaultLLMConfig returns default LLM configuration
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Enabled:           true,
		Provider:          "ollama",
		Model:             "llama3.2:latest",
		Endpoint:          "http://localhost:11434/api/generate",
		Timeout:           "20s",
		StreamEnabled:     true,
		StreamChunkMs:     60,
		CacheEnabled:      true,
		CachePath:         "",
		SummarizeTemplate: "templates/ai/summarize.md",
		ReplyTemplate:     "templates/ai/reply.md",
		LabelTemplate:     "templates/ai/label.md",
		TouchUpTemplate:   "templates/ai/touch_up.md",
		// No inline prompts in defaults - use template files
		SummarizePrompt: "",
		ReplyPrompt:     "",
		LabelPrompt:     "",
		TouchUpPrompt:   "",
	}
}

// DefaultSlackConfig returns default Slack configuration
func DefaultSlackConfig() SlackConfig {
	return SlackConfig{
		Enabled:         false,
		Channels:        []SlackChannel{},
		Defaults:        DefaultSlackDefaults(),
		SummaryTemplate: "templates/slack/summary.md",
		SummaryPrompt:   "You are a precise email summarizer. Extract only factual information from the email below. Do not add opinions, interpretations, or information not present in the original email.\n\nRequirements:\n- Maximum {{max_words}} words\n- Preserve exact names, dates, numbers, and technical terms\n- If forwarding urgent/important items, start with \"[URGENT]\" or \"[ACTION REQUIRED]\" only if explicitly stated\n- Do not infer emotions or intentions not explicitly stated\n- If email contains meeting details, preserve exact time/date/location\n- If email contains action items, list them exactly as written\n\nEmail to summarize:\n{{body}}\n\nProvide only the factual summary, nothing else.",
	}
}

// DefaultSlackDefaults returns default Slack formatting preferences
func DefaultSlackDefaults() SlackDefaults {
	return SlackDefaults{
		FormatStyle: "summary",
	}
}

// DefaultKeyBindings returns default keyboard shortcuts
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		// Core email operations
		Summarize:     "y",
		GenerateReply: "g",
		SuggestLabel:  "o",
		Reply:         "r",
		Compose:       "c",
		Refresh:       "R",
		Search:        "s",
		Unread:        "u",
		ToggleRead:    "t",
		Trash:         "d",
		Archive:       "a",
		Move:          "m",
		Prompt:        "p",
		Drafts:        "D",
		Attachments:   "A",
		ManageLabels:  "l",
		Quit:          "q",

		// Additional configurable shortcuts
		Obsidian:    "O",
		Slack:       "K",
		Markdown:    "M",
		SaveMessage: "w",
		SaveRaw:     "W",
		RSVP:        "V",
		LinkPicker:  "L",
		ThemePicker: "H",
		OpenGmail:   "O",
		BulkMode:      "v",
		BulkSelect:    "space", // Space key for bulk selection
		CommandMode:   ":",
		Help:          "?",
		LoadMore:      "N",  // Shift+N for load more (n is used for search next)
		ToggleHeaders: "h",  // Toggle header visibility

		// VIM sequence timeouts (in milliseconds)
		VimNavigationTimeoutMs: 1000, // 1 second for gg navigation
		VimRangeTimeoutMs:      2000, // 2 seconds for bulk operations like d3d

		// Content navigation shortcuts (vim-like for familiar UX)
		ContentSearch: "/",      // Standard vim search
		SearchNext:    "n",      // Standard vim next match
		SearchPrev:    "N",      // Standard vim previous match
		FastUp:        "ctrl+k", // Fast up navigation
		FastDown:      "ctrl+j", // Fast down navigation
		WordLeft:      "ctrl+h", // Word left navigation
		WordRight:     "ctrl+l", // Word right navigation
		GotoTop:       "gg",     // Vim-like go to top
		GotoBottom:    "G",      // Vim-like go to bottom
	}
}

// DefaultLayoutConfig returns default layout configuration
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		AutoResize: true,
		WideBreakpoint: LayoutBreakpoint{
			Width:  120,
			Height: 30,
		},
		MediumBreakpoint: LayoutBreakpoint{
			Width:  80,
			Height: 25,
		},
		NarrowBreakpoint: LayoutBreakpoint{
			Width:  60,
			Height: 20,
		},
		DefaultLayout:  "auto",
		ShowBorders:    true,
		ShowTitles:     true,
		CompactMode:    false,
		CurrentTheme:   "gmail-dark",
		CustomThemeDir: "",
	}
}

// LoadConfig loads configuration from file and command line flags
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	return cfg, nil
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "giztui", "config.json")
}

// DefaultCredentialPaths returns the default paths for credentials and token
func DefaultCredentialPaths() (string, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}

	configDir := filepath.Join(home, ".config", "giztui")
	credentialsPath := filepath.Join(configDir, "credentials.json")
	tokenPath := filepath.Join(configDir, "token.json")

	return credentialsPath, tokenPath
}

// DefaultCacheDir returns the default cache directory path
func DefaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "giztui", "cache")
}

// DefaultSavedDir returns the default saved files directory path
func DefaultSavedDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "giztui", "saved")
}

// DefaultLogDir returns the default log directory path
func DefaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "giztui")
}

// SaveConfig saves the configuration to a file
func (c *Config) SaveConfig(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetLLMTimeout returns parsed timeout for LLM
func (c *Config) GetLLMTimeout() time.Duration {
	if c.LLM.Timeout != "" {
		if d, err := time.ParseDuration(c.LLM.Timeout); err == nil {
			return d
		}
	}
	return 20 * time.Second
}

// LoadTemplate loads a template with proper priority: file first, then inline, then fallback
func LoadTemplate(templatePath, inlinePrompt, fallbackPrompt string) string {
	// First priority: Try to load from template file if path is specified
	if strings.TrimSpace(templatePath) != "" {
		// Make path relative to config directory if not absolute
		var fullPath string
		if filepath.IsAbs(templatePath) {
			fullPath = templatePath
		} else {
			configDir := filepath.Dir(DefaultConfigPath())
			fullPath = filepath.Join(configDir, templatePath)
		}

		if content, err := os.ReadFile(fullPath); err == nil {
			return strings.TrimSpace(string(content))
		}
	}

	// Second priority: Use inline prompt if provided
	if strings.TrimSpace(inlinePrompt) != "" {
		return inlinePrompt
	}

	// Final fallback: Use provided fallback prompt
	return fallbackPrompt
}

// GetSummarizePrompt returns the summarize prompt, loading from template file if needed
func (c *LLMConfig) GetSummarizePrompt() string {
	fallback := "Briefly summarize the following email. Keep it concise and factual.\n\n{{body}}"
	return LoadTemplate(c.SummarizeTemplate, c.SummarizePrompt, fallback)
}

// GetReplyPrompt returns the reply prompt, loading from template file if needed
func (c *LLMConfig) GetReplyPrompt() string {
	fallback := "Write a professional and friendly reply to the following email. Keep the same language as the input.\n\n{{body}}"
	return LoadTemplate(c.ReplyTemplate, c.ReplyPrompt, fallback)
}

// GetLabelPrompt returns the label prompt, loading from template file if needed
func (c *LLMConfig) GetLabelPrompt() string {
	fallback := "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: {{labels}}\n\nEmail:\n{{body}}"
	return LoadTemplate(c.LabelTemplate, c.LabelPrompt, fallback)
}

// GetTouchUpPrompt returns the touch-up prompt, loading from template file if needed
func (c *LLMConfig) GetTouchUpPrompt() string {
	fallback := "You are a formatting assistant. Do NOT paraphrase, translate, or summarize. Your goals: (1) Adjust whitespace and line breaks to improve terminal readability within a wrap width of {{wrap_width}}; (2) Remove strictly duplicated sections or paragraphs. Output only the adjusted text.\n\n{{body}}"
	return LoadTemplate(c.TouchUpTemplate, c.TouchUpPrompt, fallback)
}

// GetSummaryPrompt returns the Slack summary prompt, loading from template file if needed
func (c *SlackConfig) GetSummaryPrompt() string {
	fallback := "You are a precise email summarizer. Extract only factual information from the email below. Do not add opinions, interpretations, or information not present in the original email.\n\nRequirements:\n- Maximum {{max_words}} words\n- Preserve exact names, dates, numbers, and technical terms\n- If forwarding urgent/important items, start with \"[URGENT]\" or \"[ACTION REQUIRED]\" only if explicitly stated\n- Do not infer emotions or intentions not explicitly stated\n- If email contains meeting details, preserve exact time/date/location\n- If email contains action items, list them exactly as written\n\nEmail to summarize:\n{{body}}\n\nProvide only the factual summary, nothing else."
	return LoadTemplate(c.SummaryTemplate, c.SummaryPrompt, fallback)
}
