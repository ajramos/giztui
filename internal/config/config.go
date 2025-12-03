package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/obsidian"
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

// ThemeConfig holds theme-related configuration
type ThemeConfig struct {
	Current   string `json:"current"`    // Active theme name (e.g., "gmail-dark")
	CustomDir string `json:"custom_dir"` // Custom themes directory (empty = default)
}

// AccountConfig holds configuration for a single Gmail account
type AccountConfig struct {
	ID          string `json:"id"`           // unique identifier (e.g., "personal", "work")
	DisplayName string `json:"display_name"` // human-readable name for the account
	Credentials string `json:"credentials"`  // path to credentials.json for this account
	Token       string `json:"token"`        // path to token.json for this account
	Active      bool   `json:"active"`       // whether this is the currently active account
}

// Config holds all configuration for the GizTUI application
type Config struct {
	// Multi-account support
	Accounts []AccountConfig `json:"accounts,omitempty"`

	// Legacy single-account support (for backward compatibility)
	Credentials string `json:"credentials,omitempty"`
	Token       string `json:"token,omitempty"`

	// LLM configuration (unified)
	LLM LLMConfig `json:"llm"`

	// Slack integration
	Slack SlackConfig `json:"slack"`

	// Layout configuration
	Layout LayoutConfig `json:"layout"`

	// Keyboard shortcuts
	Keys KeyBindings `json:"keys"`

	// Theme configuration
	Theme ThemeConfig `json:"theme"`

	// Logging
	LogFile string `json:"log_file"`

	// Obsidian integration
	Obsidian *obsidian.ObsidianConfig `json:"obsidian"`

	// Attachments configuration
	Attachments AttachmentsConfig `json:"attachments"`

	// Threading configuration
	Threading ThreadingConfig `json:"threading"`

	// Performance configuration
	Performance PerformanceConfig `json:"performance"`

	// Display configuration
	Display DisplayConfig `json:"display"`
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
	ShowBorders bool `json:"show_borders"`
	ShowTitles  bool `json:"show_titles"`
	CompactMode bool `json:"compact_mode"`

	// Header field display
	MaxRecipientLines int `json:"max_recipient_lines"`
}

// LayoutBreakpoint defines minimum dimensions for layout types
type LayoutBreakpoint struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// AttachmentsConfig defines attachment handling settings
type AttachmentsConfig struct {
	// DownloadPath specifies where to save downloaded attachments
	DownloadPath string `json:"download_path"`

	// AutoOpen automatically opens attachments after download
	AutoOpen bool `json:"auto_open"`

	// MaxDownloadSize limits the maximum size for automatic downloads (in MB)
	MaxDownloadSize int64 `json:"max_download_size"`
}

// ThreadingConfig defines message threading behavior and preferences
type ThreadingConfig struct {
	// Enabled controls whether threading functionality is available
	Enabled bool `json:"enabled"`

	// DefaultView specifies the default message view: "flat" or "thread"
	DefaultView string `json:"default_view"`

	// AutoExpandUnread automatically expands threads containing unread messages
	AutoExpandUnread bool `json:"auto_expand_unread"`

	// ShowThreadCount displays message count badges on threaded conversations (📧 5)
	ShowThreadCount bool `json:"show_thread_count"`

	// IndentReplies visually indents reply messages to show conversation hierarchy
	IndentReplies bool `json:"indent_replies"`

	// MaxThreadDepth limits the maximum nesting level for thread display
	MaxThreadDepth int `json:"max_thread_depth"`

	// ThreadSummaryEnabled enables AI-powered thread summaries
	ThreadSummaryEnabled bool `json:"thread_summary_enabled"`

	// PreserveThreadState remembers expanded/collapsed state between sessions
	PreserveThreadState bool `json:"preserve_thread_state"`
}

// KeyBindings defines keyboard shortcuts for the TUI
type KeyBindings struct {
	// Core email operations
	Summarize              string `json:"summarize"`
	ForceRegenerateSummary string `json:"force_regenerate_summary"` // Force regenerate AI summary (ignore cache)
	GenerateReply          string `json:"generate_reply"`
	SuggestLabel           string `json:"suggest_label"`
	Reply                  string `json:"reply"`
	ReplyAll               string `json:"reply_all"` // Reply to all recipients
	Forward                string `json:"forward"`   // Forward message
	Compose                string `json:"compose"`
	Refresh                string `json:"refresh"`
	Search                 string `json:"search"`
	Unread                 string `json:"unread"`
	Archived               string `json:"archived"`
	SearchFrom             string `json:"search_from"`    // Quick search: from current sender
	SearchTo               string `json:"search_to"`      // Quick search: to current sender
	SearchSubject          string `json:"search_subject"` // Quick search: by current subject
	ToggleRead             string `json:"toggle_read"`
	Trash                  string `json:"trash"`
	Archive                string `json:"archive"`
	Move                   string `json:"move"`
	Prompt                 string `json:"prompt"`
	Drafts                 string `json:"drafts"`
	Attachments            string `json:"attachments"`
	ManageLabels           string `json:"manage_labels"`
	Quit                   string `json:"quit"`

	// Additional configurable shortcuts
	Obsidian      string `json:"obsidian"`       // Send to Obsidian
	Slack         string `json:"slack"`          // Forward to Slack
	Markdown      string `json:"markdown"`       // Toggle markdown
	SaveMessage   string `json:"save_message"`   // Save message to file
	SaveRaw       string `json:"save_raw"`       // Save raw EML
	RSVP          string `json:"rsvp"`           // Toggle RSVP panel
	LinkPicker    string `json:"link_picker"`    // Open link picker
	ThemePicker   string `json:"theme_picker"`   // Open theme picker
	OpenGmail     string `json:"open_gmail"`     // Open message in Gmail web UI
	BulkMode      string `json:"bulk_mode"`      // Toggle bulk mode
	BulkSelect    string `json:"bulk_select"`    // Select/deselect message in bulk mode
	CommandMode   string `json:"command_mode"`   // Open command bar
	Help          string `json:"help"`           // Toggle help
	LoadMore      string `json:"load_more"`      // Load next 50 messages
	ToggleHeaders string `json:"toggle_headers"` // Toggle header visibility

	// Saved queries
	SaveQuery      string `json:"save_query"`      // Save current search as query
	QueryBookmarks string `json:"query_bookmarks"` // Browse saved queries

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

	// Threading shortcuts
	ToggleThreading    string `json:"toggle_threading"`     // Toggle between thread and flat view
	ExpandThread       string `json:"expand_thread"`        // Expand/collapse selected thread
	ExpandAllThreads   string `json:"expand_all_threads"`   // Expand all threads in current view
	CollapseAllThreads string `json:"collapse_all_threads"` // Collapse all threads
	ThreadSummary      string `json:"thread_summary"`       // Generate AI summary of thread
	NextThread         string `json:"next_thread"`          // Navigate to next thread
	PrevThread         string `json:"prev_thread"`          // Navigate to previous thread

	// Account management
	Accounts string `json:"accounts"` // Open account picker

	// Undo functionality
	Undo string `json:"undo"` // Undo last action

	// Validation settings
	ValidateShortcuts bool `json:"validate_shortcuts"` // Enable shortcut conflict validation (default: true)
}

// PerformanceConfig defines performance optimization settings
type PerformanceConfig struct {
	// Preloading controls background message preloading
	Preloading PreloadingConfig `json:"preloading"`
}

// PreloadingConfig defines background message preloading settings
type PreloadingConfig struct {
	// Enabled controls whether preloading functionality is active
	Enabled bool `json:"enabled"`

	// NextPage settings for preloading next page of messages
	NextPage NextPageConfig `json:"next_page"`

	// AdjacentMessages settings for preloading messages around current selection
	AdjacentMessages AdjacentMessagesConfig `json:"adjacent_messages"`

	// Limits define resource constraints for preloading
	Limits PreloadingLimitsConfig `json:"limits"`
}

// NextPageConfig defines next page preloading behavior
type NextPageConfig struct {
	// Enabled controls next page preloading
	Enabled bool `json:"enabled"`

	// Threshold defines when to start preloading (0.7 = start at 70% scroll)
	Threshold float64 `json:"threshold"`

	// MaxPages limits how many pages ahead to preload
	MaxPages int `json:"max_pages"`
}

// AdjacentMessagesConfig defines adjacent message preloading behavior
type AdjacentMessagesConfig struct {
	// Enabled controls adjacent message preloading
	Enabled bool `json:"enabled"`

	// Count defines how many messages around current selection to preload
	Count int `json:"count"`
}

// PreloadingLimitsConfig defines resource limits for preloading
type PreloadingLimitsConfig struct {
	// BackgroundWorkers limits concurrent background preloading tasks
	BackgroundWorkers int `json:"background_workers"`

	// CacheSizeMB limits memory usage for preloaded messages cache
	CacheSizeMB int `json:"cache_size_mb"`

	// APIQuotaReservePercent reserves % of API quota for user actions
	APIQuotaReservePercent int `json:"api_quota_reserve_percent"`
}

// DisplayConfig holds UI display preferences
type DisplayConfig struct {
	// ShowMessageNumbers enables message number column in list view
	ShowMessageNumbers bool `json:"show_message_numbers"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		LLM:         DefaultLLMConfig(),
		Slack:       DefaultSlackConfig(),
		Layout:      DefaultLayoutConfig(),
		Keys:        DefaultKeyBindings(),
		Theme:       DefaultThemeConfig(),
		Threading:   DefaultThreadingConfig(),
		Performance: DefaultPerformanceConfig(),
		Display:     DefaultDisplayConfig(),
		LogFile:     "",
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
		ReplyAll:      "E",
		Forward:       "f",
		Compose:       "c",
		Refresh:       "R",
		Search:        "s",
		Unread:        "u",
		Archived:      "B",
		SearchFrom:    "F",
		SearchTo:      "T",
		SearchSubject: "S",
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
		Obsidian:      "O",
		Slack:         "K",
		Markdown:      "M",
		SaveMessage:   "w",
		SaveRaw:       "W",
		RSVP:          "V",
		LinkPicker:    "L",
		ThemePicker:   "H",
		OpenGmail:     "O",
		BulkMode:      "v",
		BulkSelect:    "space", // Space key for bulk selection
		CommandMode:   ":",
		Help:          "?",
		LoadMore:      "N",      // Shift+N for load more (n is used for search next)
		ToggleHeaders: "h",      // Toggle header visibility
		Accounts:      "ctrl+a", // Open account picker

		// Saved queries
		SaveQuery:      "Z", // Save current search as query
		QueryBookmarks: "Q", // Browse saved queries

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

		// Threading shortcuts
		ToggleThreading:    "T",       // Toggle between thread and flat view
		ExpandThread:       "enter",   // Expand/collapse selected thread
		ExpandAllThreads:   "E",       // Expand all threads in current view
		CollapseAllThreads: "C",       // Collapse all threads
		ThreadSummary:      "shift+t", // Generate AI summary of thread
		NextThread:         "ctrl+n",  // Navigate to next thread
		PrevThread:         "ctrl+p",  // Navigate to previous thread

		// Undo functionality
		Undo: "U", // Undo last action

		// Validation settings (default: enabled for safety)
		ValidateShortcuts: true, // Enable shortcut conflict validation by default
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
		DefaultLayout:     "auto",
		ShowBorders:       true,
		ShowTitles:        true,
		CompactMode:       false,
		MaxRecipientLines: 3,
	}
}

// DefaultThemeConfig returns default theme configuration
func DefaultThemeConfig() ThemeConfig {
	return ThemeConfig{
		Current:   "gmail-dark",
		CustomDir: "",
	}
}

// DefaultThreadingConfig returns default threading configuration
func DefaultThreadingConfig() ThreadingConfig {
	return ThreadingConfig{
		Enabled:              true,
		DefaultView:          "flat",
		AutoExpandUnread:     true,
		ShowThreadCount:      true,
		IndentReplies:        true,
		MaxThreadDepth:       10,
		ThreadSummaryEnabled: true,
		PreserveThreadState:  true,
	}
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		Preloading: PreloadingConfig{
			Enabled: true, // Preloading ON by default as requested
			NextPage: NextPageConfig{
				Enabled:   true,
				Threshold: 0.7, // Start preloading at 70% scroll
				MaxPages:  2,   // Preload up to 2 pages ahead
			},
			AdjacentMessages: AdjacentMessagesConfig{
				Enabled: true,
				Count:   3, // Preload 3 messages around current selection
			},
			Limits: PreloadingLimitsConfig{
				BackgroundWorkers:      3,  // 3 background workers for preloading
				CacheSizeMB:            50, // 50MB cache limit
				APIQuotaReservePercent: 20, // Reserve 20% of API quota for user actions
			},
		},
	}
}

// DefaultDisplayConfig returns the default display configuration
func DefaultDisplayConfig() DisplayConfig {
	return DisplayConfig{
		ShowMessageNumbers: false, // Off by default - users enable via config or :numbers command
	}
}

// LoadConfig loads configuration from file and command line flags
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	if configPath != "" {
		// Validate path to prevent directory traversal
		cleanPath := filepath.Clean(configPath)
		if strings.Contains(cleanPath, "..") {
			return nil, fmt.Errorf("invalid config path: contains directory traversal")
		}
		if data, err := os.ReadFile(cleanPath); err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Validate configuration and show warnings for potential conflicts
	if warnings := ValidateKeyboardConfig(cfg.Keys); len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  Configuration warnings:\n")
		for _, warning := range warnings {
			fmt.Fprintf(os.Stderr, "   • %s\n", warning)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	return cfg, nil
}

// ValidateKeyboardConfig checks for potential configuration conflicts and returns warnings
func ValidateKeyboardConfig(keys KeyBindings) []string {
	// Check if validation is disabled
	if !keys.ValidateShortcuts {
		return []string{} // Return empty warnings if validation is disabled
	}

	var warnings []string

	// Define hardcoded shortcuts and their corresponding config alternatives
	// This maps hardcoded keys to the config parameter that can override them
	hardcodedShortcuts := map[string]string{
		// Hardcoded shortcuts WITH isKeyConfigured checks (can be overridden)
		" ": "bulk_select",    // Space key → bulk_select config
		"v": "bulk_mode",      // v key → bulk_mode config
		":": "command_mode",   // : key → command_mode config
		"?": "help",           // ? key → help config
		"r": "refresh",        // r key → refresh config (reload messages)
		"n": "load_more",      // n key → load_more config (or compose in some contexts)
		"s": "search",         // s key → search config
		"u": "unread",         // u key → unread config
		"t": "toggle_read",    // t key → toggle_read config
		"d": "trash",          // d key → trash config
		"a": "archive",        // a key → archive config
		"B": "archived",       // B key → archived config
		"F": "search_from",    // F key → search_from config
		"T": "search_to",      // T key → search_to config
		"S": "search_subject", // S key → search_subject config
		"K": "slack",          // K key → slack config
		"l": "manage_labels",  // l key → manage_labels config
		"m": "move",           // m key → move config
		"M": "markdown",       // M key → markdown config
		"V": "rsvp",           // V key → rsvp config
		"O": "obsidian",       // O key → obsidian config
		"L": "link_picker",    // L key → link_picker config
		"w": "save_message",   // w key → save_message config
		"W": "save_raw",       // W key → save_raw config

		// Hardcoded shortcuts WITHOUT isKeyConfigured checks (always active, but user can override)
		"b": "bulk_mode",      // b key → bulk_mode config (alternative to 'v')
		"q": "quit",           // q key → quit config (always hardcoded)
		"R": "reply",          // R key → reply config
		"D": "drafts",         // D key → drafts config
		"A": "attachments",    // A key → attachments config
		"U": "undo",           // U key → undo config
		"o": "suggest_label",  // o key → suggest_label config
		"p": "prompt",         // p key → prompt config (bulk or single mode)
		"g": "generate_reply", // g key → generate_reply config
		"y": "summarize",      // y key → summarize config
		"E": "reply_all",      // E key → reply_all config
		"c": "compose",        // c key → compose config
		"f": "forward",        // f key → forward config

		// Default configurable shortcuts that could conflict with user overrides
		// These have defaults but can be reconfigured, so we should warn about conflicts
		"Z": "save_query",      // Z key → save_query config (default)
		"Q": "query_bookmarks", // Q key → query_bookmarks config (default)
		"H": "theme_picker",    // H key → theme_picker config (default)
		"N": "load_more",       // N key → load_more config (default)
		"h": "toggle_headers",  // h key → toggle_headers config (default)
	}

	// Create a map of all configured keys to detect duplicates
	keyMap := make(map[string][]string)

	// Use reflection to check all keyboard config fields
	v := reflect.ValueOf(keys)
	t := reflect.TypeOf(keys)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip non-string fields and private fields
		if field.Kind() != reflect.String || !field.CanInterface() {
			continue
		}

		keyValue := field.String()
		if keyValue != "" {
			fieldName := strings.ToLower(fieldType.Tag.Get("json"))
			if fieldName == "" || fieldName == "-" {
				fieldName = fieldType.Name
			}
			keyMap[keyValue] = append(keyMap[keyValue], fieldName)
		}
	}

	// Check for duplicate key assignments
	for key, fields := range keyMap {
		if len(fields) > 1 {
			warnings = append(warnings, fmt.Sprintf("Key '%s' is assigned to multiple functions: %s", key, strings.Join(fields, ", ")))
		}
	}

	// Check for specific known conflict patterns
	if keys.Summarize != "" && len(keys.Summarize) == 1 {
		upperKey := strings.ToUpper(keys.Summarize)
		// Check if the uppercase version conflicts with any configured key
		conflictingFields := keyMap[upperKey]
		if len(conflictingFields) > 0 {
			// Only warn if force_regenerate_summary is NOT explicitly configured
			// If the user has explicitly configured force_regenerate_summary, there's no loss of functionality
			if keys.ForceRegenerateSummary == "" {
				warnings = append(warnings, fmt.Sprintf("Auto-generated force_regenerate_summary key '%s' (uppercase of summarize '%s') conflicts with configured: %s. Your configured shortcut will take precedence. Consider adding explicit 'force_regenerate_summary' configuration.", upperKey, keys.Summarize, strings.Join(conflictingFields, ", ")))
			}
			// If force_regenerate_summary IS configured, no warning needed - user has explicit control
		}
	}

	// Check for hardcoded shortcut conflicts - warn when user overrides hardcoded functionality without alternative
	for hardcodedKey, configParam := range hardcodedShortcuts {
		// Check if this hardcoded key is configured for a different function
		conflictingFields := keyMap[hardcodedKey]
		if len(conflictingFields) > 0 {
			// Check if the user has provided an explicit alternative for this functionality
			hasAlternative := false

			// Use reflection to check if the corresponding config parameter is set
			v := reflect.ValueOf(keys)
			t := reflect.TypeOf(keys)
			for i := 0; i < v.NumField(); i++ {
				field := v.Field(i)
				fieldType := t.Field(i)

				// Skip non-string fields
				if field.Kind() != reflect.String || !field.CanInterface() {
					continue
				}

				// Get the JSON tag name
				jsonTag := fieldType.Tag.Get("json")
				if jsonTag == "" {
					continue
				}

				// Remove options from tag (like omitempty)
				jsonName := strings.Split(jsonTag, ",")[0]

				// Check if this field matches the config parameter we're looking for
				if jsonName == configParam {
					keyValue := field.String()
					if keyValue != "" {
						hasAlternative = true
						break
					}
				}
			}

			// Only warn if no alternative is provided
			if !hasAlternative {
				warnings = append(warnings, fmt.Sprintf("Key '%s' is configured for '%s' but no '%s' alternative provided - %s functionality will be lost. Consider adding '%s' configuration.", hardcodedKey, strings.Join(conflictingFields, ", "), configParam, getFunctionName(configParam), configParam))
			}
		}
	}

	return warnings
}

// getFunctionName returns a user-friendly name for a config parameter
func getFunctionName(configParam string) string {
	functionNames := map[string]string{
		"bulk_select":     "bulk selection",
		"bulk_mode":       "bulk mode",
		"command_mode":    "command mode",
		"help":            "help",
		"refresh":         "refresh/reload messages",
		"load_more":       "load more messages",
		"search":          "search",
		"unread":          "unread messages",
		"toggle_read":     "toggle read/unread",
		"trash":           "delete/trash",
		"archive":         "archive",
		"archived":        "archived messages",
		"search_from":     "search from sender",
		"search_to":       "search to recipient",
		"search_subject":  "search by subject",
		"slack":           "Slack integration",
		"manage_labels":   "label management",
		"move":            "move messages",
		"markdown":        "markdown toggle",
		"rsvp":            "RSVP",
		"obsidian":        "Obsidian integration",
		"link_picker":     "link picker",
		"save_message":    "save message",
		"save_raw":        "save raw message",
		"quit":            "quit application",
		"reply":           "reply to message",
		"drafts":          "drafts",
		"attachments":     "attachments",
		"undo":            "undo last action",
		"suggest_label":   "AI label suggestions",
		"prompt":          "AI prompts",
		"generate_reply":  "AI reply generation",
		"summarize":       "AI summary",
		"reply_all":       "reply to all",
		"compose":         "compose message",
		"forward":         "forward message",
		"save_query":      "save search query",
		"query_bookmarks": "saved query bookmarks",
		"theme_picker":    "theme picker",
		"toggle_headers":  "toggle headers",
	}

	if name, exists := functionNames[configParam]; exists {
		return name
	}
	return configParam // fallback to parameter name
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
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
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

		// Validate path to prevent directory traversal
		cleanPath := filepath.Clean(fullPath)
		if strings.Contains(cleanPath, "..") {
			return fallbackPrompt
		}
		if content, err := os.ReadFile(cleanPath); err == nil {
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
