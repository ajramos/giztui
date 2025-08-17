package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	Region   string `json:"region"`  // For AWS Bedrock
	APIKey   string `json:"api_key"`
	Timeout  string `json:"timeout"`

	// Streaming configuration
	StreamEnabled bool `json:"stream_enabled"`
	StreamChunkMs int  `json:"stream_chunk_ms"`

	// Caching configuration
	CacheEnabled bool   `json:"cache_enabled"`
	CachePath    string `json:"cache_path"`

	// Prompt templates for LLM interactions
	SummarizePrompt string `json:"summarize_prompt"`
	ReplyPrompt     string `json:"reply_prompt"`
	LabelPrompt     string `json:"label_prompt"`
	// Touch-up prompt for LLM whitespace/line-break adjustments (no semantic changes)
	TouchUpPrompt string `json:"touch_up_prompt"`
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
	
	// SummaryPrompt is the AI prompt template for generating email summaries
	// Available variables: {{body}}, {{subject}}, {{from}}, {{to}}, {{cc}}, {{bcc}}, 
	// {{date}}, {{reply-to}}, {{message-id}}, {{in-reply-to}}, {{references}}, {{max_words}}
	SummaryPrompt string `json:"summary_prompt"`
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
	ShowBorders bool   `json:"show_borders"`
	ShowTitles  bool   `json:"show_titles"`
	CompactMode bool   `json:"compact_mode"`
	ColorScheme string `json:"color_scheme"`
}

// LayoutBreakpoint defines minimum dimensions for layout types
type LayoutBreakpoint struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// KeyBindings defines keyboard shortcuts for the TUI
type KeyBindings struct {
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
	Drafts        string `json:"drafts"`
	Attachments   string `json:"attachments"`
	ManageLabels  string `json:"manage_labels"`
	Quit          string `json:"quit"`
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
		Enabled:         true,
		Provider:        "ollama",
		Model:           "llama3.2:latest",
		Endpoint:        "http://localhost:11434/api/generate",
		Timeout:         "20s",
		StreamEnabled:   true,
		StreamChunkMs:   60,
		CacheEnabled:    true,
		CachePath:       "",
		SummarizePrompt: "Briefly summarize the following email. Keep it concise and factual.\n\n{{body}}",
		ReplyPrompt:     "Write a professional and friendly reply to the following email. Keep the same language as the input.\n\n{{body}}",
		LabelPrompt:     "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: {{labels}}\n\nEmail:\n{{body}}",
		TouchUpPrompt:   "You are a formatting assistant. Do NOT paraphrase, translate, or summarize. Your goals: (1) Adjust whitespace and line breaks to improve terminal readability within a wrap width of {{wrap_width}}; (2) Remove strictly duplicated sections or paragraphs. A section/paragraph counts as duplicate if its text is identical to a previous one except for whitespace or numeric link reference indices like [1], [23]. Do NOT remove unique content. Preserve quotes (> ), code/pre/PGP blocks verbatim, lists, ASCII tables, link references (text [n] + [LINKS]), and keep [ATTACHMENTS] and [IMAGES] unchanged. Output only the adjusted text.\n\n{{body}}",
	}
}

// DefaultSlackConfig returns default Slack configuration
func DefaultSlackConfig() SlackConfig {
	return SlackConfig{
		Enabled:  false,
		Channels: []SlackChannel{},
		Defaults: DefaultSlackDefaults(),
		SummaryPrompt: "You are a precise email summarizer. Extract only factual information from the email below. Do not add opinions, interpretations, or information not present in the original email.\n\nRequirements:\n- Maximum {{max_words}} words\n- Preserve exact names, dates, numbers, and technical terms\n- If forwarding urgent/important items, start with \"[URGENT]\" or \"[ACTION REQUIRED]\" only if explicitly stated\n- Do not infer emotions or intentions not explicitly stated\n- If email contains meeting details, preserve exact time/date/location\n- If email contains action items, list them exactly as written\n\nEmail to summarize:\n{{body}}\n\nProvide only the factual summary, nothing else.",
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
		Summarize:     "y",
		GenerateReply: "g",
		SuggestLabel:  "o",
		Reply:         "r",
		Compose:       "n",
		Refresh:       "R",
		Search:        "s",
		Unread:        "u",
		ToggleRead:    "t",
		Trash:         "d",
		Archive:       "a",
		Drafts:        "D",
		Attachments:   "A",
		ManageLabels:  "l",
		Quit:          "q",
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
		DefaultLayout: "auto",
		ShowBorders:   true,
		ShowTitles:    true,
		CompactMode:   false,
		ColorScheme:   "default",
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
