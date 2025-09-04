package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.True(t, cfg.LLM.Enabled)
	assert.Equal(t, "ollama", cfg.LLM.Provider)
	assert.Equal(t, "llama3.2:latest", cfg.LLM.Model)
	assert.False(t, cfg.Slack.Enabled)
	assert.True(t, cfg.Layout.AutoResize)
	assert.NotEmpty(t, cfg.Keys.Summarize)
}

func TestDefaultLLMConfig(t *testing.T) {
	cfg := DefaultLLMConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "ollama", cfg.Provider)
	assert.Equal(t, "llama3.2:latest", cfg.Model)
	assert.Equal(t, "http://localhost:11434/api/generate", cfg.Endpoint)
	assert.Equal(t, "20s", cfg.Timeout)
	assert.True(t, cfg.StreamEnabled)
	assert.Equal(t, 60, cfg.StreamChunkMs)
	assert.True(t, cfg.CacheEnabled)
	assert.Equal(t, "", cfg.CachePath)
	assert.Equal(t, "templates/ai/summarize.md", cfg.SummarizeTemplate)
	assert.Equal(t, "templates/ai/reply.md", cfg.ReplyTemplate)
	assert.Equal(t, "templates/ai/label.md", cfg.LabelTemplate)
	assert.Equal(t, "templates/ai/touch_up.md", cfg.TouchUpTemplate)
	assert.Empty(t, cfg.SummarizePrompt)
	assert.Empty(t, cfg.ReplyPrompt)
	assert.Empty(t, cfg.LabelPrompt)
	assert.Empty(t, cfg.TouchUpPrompt)
}

func TestDefaultSlackConfig(t *testing.T) {
	cfg := DefaultSlackConfig()

	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.Channels)
	assert.Equal(t, "summary", cfg.Defaults.FormatStyle)
	assert.Equal(t, "templates/slack/summary.md", cfg.SummaryTemplate)
	assert.NotEmpty(t, cfg.SummaryPrompt) // Has default inline prompt
}

func TestDefaultKeyBindings(t *testing.T) {
	keys := DefaultKeyBindings()

	// Core operations
	assert.Equal(t, "y", keys.Summarize)
	assert.Equal(t, "g", keys.GenerateReply)
	assert.Equal(t, "o", keys.SuggestLabel)
	assert.Equal(t, "r", keys.Reply)
	assert.Equal(t, "c", keys.Compose)
	assert.Equal(t, "R", keys.Refresh)
	assert.Equal(t, "s", keys.Search)
	assert.Equal(t, "u", keys.Unread)
	assert.Equal(t, "t", keys.ToggleRead)
	assert.Equal(t, "d", keys.Trash)
	assert.Equal(t, "a", keys.Archive)
	assert.Equal(t, "q", keys.Quit)

	// VIM timeouts
	assert.Equal(t, 1000, keys.VimNavigationTimeoutMs)
	assert.Equal(t, 2000, keys.VimRangeTimeoutMs)

	// Content navigation
	assert.Equal(t, "/", keys.ContentSearch)
	assert.Equal(t, "n", keys.SearchNext)
	assert.Equal(t, "N", keys.SearchPrev)
	assert.Equal(t, "gg", keys.GotoTop)
	assert.Equal(t, "G", keys.GotoBottom)
}

func TestDefaultLayoutConfig(t *testing.T) {
	layout := DefaultLayoutConfig()

	assert.True(t, layout.AutoResize)
	assert.Equal(t, 120, layout.WideBreakpoint.Width)
	assert.Equal(t, 30, layout.WideBreakpoint.Height)
	assert.Equal(t, 80, layout.MediumBreakpoint.Width)
	assert.Equal(t, 25, layout.MediumBreakpoint.Height)
	assert.Equal(t, 60, layout.NarrowBreakpoint.Width)
	assert.Equal(t, 20, layout.NarrowBreakpoint.Height)
	assert.Equal(t, "auto", layout.DefaultLayout)
	assert.True(t, layout.ShowBorders)
	assert.True(t, layout.ShowTitles)
	assert.False(t, layout.CompactMode)
}

func TestGetLLMTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  string
		expected time.Duration
	}{
		{"valid_seconds", "30s", 30 * time.Second},
		{"valid_minutes", "2m", 2 * time.Minute},
		{"valid_milliseconds", "500ms", 500 * time.Millisecond},
		{"invalid_format", "invalid", 20 * time.Second}, // fallback
		{"empty_string", "", 20 * time.Second},          // fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{LLM: LLMConfig{Timeout: tt.timeout}}
			result := cfg.GetLLMTimeout()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadTemplate_FilePriority(t *testing.T) {
	// Create temporary directory and file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "test_template.md")
	templateContent := "Template from file"

	err := os.WriteFile(templateFile, []byte(templateContent), 0600)
	assert.NoError(t, err)

	// Test file priority (should use file content)
	result := LoadTemplate(templateFile, "Inline prompt", "Fallback prompt")
	assert.Equal(t, templateContent, result)
}

func TestLoadTemplate_InlinePriority(t *testing.T) {
	// Test with non-existent file - should use inline prompt
	inlinePrompt := "Inline prompt content"
	result := LoadTemplate("/nonexistent/file.md", inlinePrompt, "Fallback prompt")
	assert.Equal(t, inlinePrompt, result)
}

func TestLoadTemplate_FallbackPriority(t *testing.T) {
	// Test with empty paths - should use fallback
	fallback := "Fallback prompt content"
	result := LoadTemplate("", "", fallback)
	assert.Equal(t, fallback, result)
}

func TestLoadTemplate_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		inlinePrompt string
		fallback     string
		expected     string
	}{
		{"whitespace_template_path", "   ", "inline", "fallback", "inline"},
		{"whitespace_inline_prompt", "nonexistent", "   ", "fallback", "fallback"},
		{"empty_all", "", "", "fallback", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LoadTemplate(tt.templatePath, tt.inlinePrompt, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLLMConfig_GetPromptMethods(t *testing.T) {
	cfg := DefaultLLMConfig()

	// Test that prompts are loaded (may be from file or fallback)
	summarize := cfg.GetSummarizePrompt()
	assert.NotEmpty(t, summarize)
	assert.Contains(t, summarize, "{{body}}")

	reply := cfg.GetReplyPrompt()
	assert.NotEmpty(t, reply)
	assert.Contains(t, reply, "{{body}}")

	label := cfg.GetLabelPrompt()
	assert.NotEmpty(t, label)
	assert.Contains(t, label, "{{body}}")

	touchUp := cfg.GetTouchUpPrompt()
	assert.NotEmpty(t, touchUp)
	assert.Contains(t, touchUp, "{{body}}")
}

func TestLLMConfig_GetPromptMethods_WithInlineOverrides(t *testing.T) {
	cfg := LLMConfig{
		SummarizePrompt: "Custom summarize: {{body}}",
		ReplyPrompt:     "Custom reply: {{body}}",
		LabelPrompt:     "Custom label: {{labels}} {{body}}",
		TouchUpPrompt:   "Custom touchup: {{body}}",
	}

	assert.Equal(t, "Custom summarize: {{body}}", cfg.GetSummarizePrompt())
	assert.Equal(t, "Custom reply: {{body}}", cfg.GetReplyPrompt())
	assert.Equal(t, "Custom label: {{labels}} {{body}}", cfg.GetLabelPrompt())
	assert.Equal(t, "Custom touchup: {{body}}", cfg.GetTouchUpPrompt())
}

func TestSlackConfig_GetSummaryPrompt(t *testing.T) {
	cfg := DefaultSlackConfig()

	prompt := cfg.GetSummaryPrompt()
	assert.Contains(t, prompt, "summarizer")
	assert.Contains(t, prompt, "{{max_words}}")
	assert.Contains(t, prompt, "{{body}}")
	assert.Contains(t, prompt, "factual")
}

func TestSlackConfig_GetSummaryPrompt_WithOverride(t *testing.T) {
	cfg := SlackConfig{
		SummaryPrompt: "Custom slack summary: {{body}}",
	}

	prompt := cfg.GetSummaryPrompt()
	assert.Equal(t, "Custom slack summary: {{body}}", prompt)
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()

	// Should not be empty (unless no home directory)
	if path != "" {
		assert.Contains(t, path, ".config")
		assert.Contains(t, path, "giztui")
		assert.Contains(t, path, "config.json")
	}
}

func TestDefaultCredentialPaths(t *testing.T) {
	credPath, tokenPath := DefaultCredentialPaths()

	// Both should be empty or both should be valid
	if credPath != "" && tokenPath != "" {
		assert.Contains(t, credPath, ".config")
		assert.Contains(t, credPath, "giztui")
		assert.Contains(t, credPath, "credentials.json")

		assert.Contains(t, tokenPath, ".config")
		assert.Contains(t, tokenPath, "giztui")
		assert.Contains(t, tokenPath, "token.json")
	}
}

func TestDefaultCacheDir(t *testing.T) {
	path := DefaultCacheDir()

	if path != "" {
		assert.Contains(t, path, ".config")
		assert.Contains(t, path, "giztui")
		assert.Contains(t, path, "cache")
	}
}

func TestDefaultSavedDir(t *testing.T) {
	path := DefaultSavedDir()

	if path != "" {
		assert.Contains(t, path, ".config")
		assert.Contains(t, path, "giztui")
		assert.Contains(t, path, "saved")
	}
}

func TestDefaultLogDir(t *testing.T) {
	path := DefaultLogDir()

	if path != "" {
		assert.Contains(t, path, ".config")
		assert.Contains(t, path, "giztui")
	}
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	cfg, err := LoadConfig("")

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// Should return default config
	assert.Equal(t, DefaultConfig().LLM.Provider, cfg.LLM.Provider)
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/config.json")

	assert.NoError(t, err) // Should not error for missing file
	assert.NotNil(t, cfg)
	// Should return default config
	assert.Equal(t, DefaultConfig().LLM.Provider, cfg.LLM.Provider)
}

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	testConfig := &Config{
		Credentials: "test-creds",
		Token:       "test-token",
		LLM: LLMConfig{
			Enabled:  false,
			Provider: "custom",
			Model:    "test-model",
		},
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	assert.NoError(t, err)

	err = os.WriteFile(configFile, data, 0600)
	assert.NoError(t, err)

	// Load config
	cfg, err := LoadConfig(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have loaded values
	assert.Equal(t, "test-creds", cfg.Credentials)
	assert.Equal(t, "test-token", cfg.Token)
	assert.False(t, cfg.LLM.Enabled)
	assert.Equal(t, "custom", cfg.LLM.Provider)
	assert.Equal(t, "test-model", cfg.LLM.Model)
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(configFile, []byte("invalid json content"), 0600)
	assert.NoError(t, err)

	cfg, err := LoadConfig(configFile)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.json")

	cfg := DefaultConfig()
	cfg.Credentials = "test-save-creds"
	cfg.LLM.Provider = "test-provider"

	err := cfg.SaveConfig(configFile)
	assert.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configFile)

	// Verify content by loading it back
	loadedCfg, err := LoadConfig(configFile)
	assert.NoError(t, err)
	assert.Equal(t, "test-save-creds", loadedCfg.Credentials)
	assert.Equal(t, "test-provider", loadedCfg.LLM.Provider)
}

func TestSaveConfig_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "nested", "deep", "config.json")

	cfg := DefaultConfig()
	err := cfg.SaveConfig(configFile)
	assert.NoError(t, err)

	// Verify nested directories were created
	assert.FileExists(t, configFile)
}

// Test struct validation
func TestSlackChannel_Validation(t *testing.T) {
	channel := SlackChannel{
		ID:          "test-id",
		Name:        "Test Channel",
		WebhookURL:  "https://hooks.slack.com/test",
		Default:     true,
		Description: "Test description",
	}

	assert.Equal(t, "test-id", channel.ID)
	assert.Equal(t, "Test Channel", channel.Name)
	assert.Equal(t, "https://hooks.slack.com/test", channel.WebhookURL)
	assert.True(t, channel.Default)
	assert.Equal(t, "Test description", channel.Description)
}

func TestLayoutBreakpoint_Validation(t *testing.T) {
	bp := LayoutBreakpoint{
		Width:  100,
		Height: 50,
	}

	assert.Equal(t, 100, bp.Width)
	assert.Equal(t, 50, bp.Height)
}

func TestAttachmentsConfig_Validation(t *testing.T) {
	attachments := AttachmentsConfig{
		DownloadPath:    "/tmp/downloads",
		AutoOpen:        true,
		MaxDownloadSize: 100,
	}

	assert.Equal(t, "/tmp/downloads", attachments.DownloadPath)
	assert.True(t, attachments.AutoOpen)
	assert.Equal(t, int64(100), attachments.MaxDownloadSize)
}

// Benchmark tests for performance critical operations
func BenchmarkDefaultConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

func BenchmarkLoadTemplate_Fallback(b *testing.B) {
	fallback := "Fallback prompt content"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LoadTemplate("", "", fallback)
	}
}

func BenchmarkGetLLMTimeout_Valid(b *testing.B) {
	cfg := &Config{LLM: LLMConfig{Timeout: "30s"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetLLMTimeout()
	}
}

func BenchmarkGetLLMTimeout_Invalid(b *testing.B) {
	cfg := &Config{LLM: LLMConfig{Timeout: "invalid"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetLLMTimeout()
	}
}

// Test edge cases
func TestConfig_EdgeCases(t *testing.T) {
	// Test empty struct initialization
	emptyConfig := &Config{}
	timeout := emptyConfig.GetLLMTimeout()
	assert.Equal(t, 20*time.Second, timeout) // Should use default fallback
}

func TestLoadTemplate_AbsoluteVsRelativePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	templateFile := filepath.Join(tmpDir, "template.md")
	content := "Template content"
	err := os.WriteFile(templateFile, []byte(content), 0600)
	assert.NoError(t, err)

	// Test absolute path
	result := LoadTemplate(templateFile, "inline", "fallback")
	assert.Equal(t, content, result)

	// Test relative path (should fail to find file and use inline)
	result = LoadTemplate("relative/path/template.md", "inline prompt", "fallback")
	assert.Equal(t, "inline prompt", result)
}

// Test JSON marshaling/unmarshaling
func TestConfig_JSONSerialization(t *testing.T) {
	original := DefaultConfig()
	original.Credentials = "test-credentials"
	original.LLM.Provider = "test-provider"

	// Marshal to JSON
	data, err := json.Marshal(original)
	assert.NoError(t, err)

	// Unmarshal back
	var loaded Config
	err = json.Unmarshal(data, &loaded)
	assert.NoError(t, err)

	// Compare key fields
	assert.Equal(t, original.Credentials, loaded.Credentials)
	assert.Equal(t, original.LLM.Provider, loaded.LLM.Provider)
	assert.Equal(t, original.LLM.Enabled, loaded.LLM.Enabled)
}
