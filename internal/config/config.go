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

	// Prompt templates for LLM interactions
	SummarizePrompt string `json:"summarize_prompt"`
	ReplyPrompt     string `json:"reply_prompt"`
	LabelPrompt     string `json:"label_prompt"`

	// Layout configuration
	Layout LayoutConfig `json:"layout"`

	// Keyboard shortcuts
	Keys KeyBindings `json:"keys"`
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
		OllamaTimeout:   "30s",
		SummarizePrompt: "Resume brevemente el siguiente correo electrónico:\n\n{{body}}\n\nDevuelve el resumen en español en un párrafo.",
		ReplyPrompt:     "Redacta una respuesta profesional y amable al siguiente correo:\n\n{{body}}",
		LabelPrompt:     "Sugiere una etiqueta adecuada para el siguiente correo considerando las ya existentes: {{labels}}.\n\nCorreo:\n{{body}}",
		Layout:          DefaultLayoutConfig(),
		Keys:            DefaultKeyBindings(),
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
