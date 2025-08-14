package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config holds all configuration for the Gmail TUI application
type Config struct {
	Credentials    string `json:"credentials"`
	Token          string `json:"token"`
	OllamaEndpoint string `json:"ollama_endpoint"`
	OllamaModel    string `json:"ollama_model"`
	OllamaTimeout  string `json:"ollama_timeout"`

	// Generic LLM configuration
	LLMEnabled  bool   `json:"llm_enabled"`
	LLMProvider string `json:"llm_provider"` // ollama, openai, anthropic, custom
	LLMModel    string `json:"llm_model"`
	LLMEndpoint string `json:"llm_endpoint"`
	// For providers that use regions (e.g., Bedrock), prefer LLMRegion over LLMEndpoint
	LLMRegion  string `json:"llm_region"`
	LLMAPIKey  string `json:"llm_api_key"`
	LLMTimeout string `json:"llm_timeout"`
	// Streaming
	LLMStreamEnabled bool `json:"llm_stream_enabled"`
	LLMStreamChunkMs int  `json:"llm_stream_chunk_ms"`

	// AI summary cache (SQLite)
	AISummaryCacheEnabled bool   `json:"ai_summary_cache_enabled"`
	AISummaryCachePath    string `json:"ai_summary_cache_path"`

	// Prompt templates for LLM interactions
	SummarizePrompt string `json:"summarize_prompt"`
	ReplyPrompt     string `json:"reply_prompt"`
	LabelPrompt     string `json:"label_prompt"`
	// Touch-up prompt for LLM whitespace/line-break adjustments (no semantic changes)
	TouchUpPrompt string `json:"touch_up_prompt"`

	// Layout configuration
	Layout LayoutConfig `json:"layout"`

	// Keyboard shortcuts
	Keys KeyBindings `json:"keys"`

	// Logging
	LogFile string `json:"log_file"`
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
		OllamaTimeout:         "30s",
		LLMEnabled:            true,
		LLMProvider:           "ollama",
		LLMModel:              "llama3.2:latest",
		LLMEndpoint:           "http://localhost:11434/api/generate",
		LLMTimeout:            "20s",
		LLMStreamEnabled:      true,
		LLMStreamChunkMs:      60,
		AISummaryCacheEnabled: true,
		AISummaryCachePath:    "",
		SummarizePrompt:       "Briefly summarize the following email. Keep it concise and factual.\n\n{{body}}",
		ReplyPrompt:           "Write a professional and friendly reply to the following email. Keep the same language as the input.\n\n{{body}}",
		LabelPrompt:           "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: {{labels}}\n\nEmail:\n{{body}}",
		TouchUpPrompt:         "You are a formatting assistant. Do NOT paraphrase, translate, or summarize. Your goals: (1) Adjust whitespace and line breaks to improve terminal readability within a wrap width of {{wrap_width}}; (2) Remove strictly duplicated sections or paragraphs. A section/paragraph counts as duplicate if its text is identical to a previous one except for whitespace or numeric link reference indices like [1], [23]. Do NOT remove unique content. Preserve quotes (> ), code/pre/PGP blocks verbatim, lists, ASCII tables, link references (text [n] + [LINKS]), and keep [ATTACHMENTS] and [IMAGES] unchanged. Output only the adjusted text.\n\n{{body}}",
		Layout:                DefaultLayoutConfig(),
		Keys:                  DefaultKeyBindings(),
		LogFile:               "",
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
	return filepath.Join(home, ".config", "gmail-tui", "config.json")
}

// DefaultCredentialPaths returns the default paths for credentials and token
func DefaultCredentialPaths() (string, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}

	configDir := filepath.Join(home, ".config", "gmail-tui")
	credentialsPath := filepath.Join(configDir, "credentials.json")
	tokenPath := filepath.Join(configDir, "token.json")

	return credentialsPath, tokenPath
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

// GetOllamaTimeout returns the parsed timeout duration
func (c *Config) GetOllamaTimeout() time.Duration {
	if c.OllamaTimeout == "" {
		return 30 * time.Second
	}

	if d, err := time.ParseDuration(c.OllamaTimeout); err == nil {
		return d
	}

	return 30 * time.Second
}

// GetLLMTimeout returns parsed timeout for generic LLM
func (c *Config) GetLLMTimeout() time.Duration {
	if c.LLMTimeout != "" {
		if d, err := time.ParseDuration(c.LLMTimeout); err == nil {
			return d
		}
	}
	return 20 * time.Second
}
