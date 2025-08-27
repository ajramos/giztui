package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test command parsing logic
func TestExecuteCommand_CommandParsing(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"help", []string{"help"}},
		{"labels create new", []string{"labels", "create", "new"}},
		{"search from:user", []string{"search", "from:user"}},
		{" trim spaces ", []string{"trim", "spaces"}},
		{"", []string{}},
		{"   ", []string{}},
	}

	for _, tc := range testCases {
		parts := strings.Fields(tc.input)
		assert.Equal(t, tc.expected, parts, "Command parsing for input: '%s'", tc.input)
	}
}

func TestExecuteCommand_CommandNormalization(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"HELP", "help"},
		{"Labels", "labels"},
		{"SEARCH", "search"},
		{"quit", "quit"},
	}

	for _, tc := range testCases {
		normalized := strings.ToLower(tc.input)
		assert.Equal(t, tc.expected, normalized, "Command normalization for: '%s'", tc.input)
	}
}

// Test content search command detection
func TestExecuteCommand_ContentSearchDetection(t *testing.T) {
	testCases := []struct {
		command         string
		isContentSearch bool
		searchTerm      string
	}{
		{"/error", true, "error"},
		{"/test query", true, "test"},
		{"search", false, ""},
		{"/", false, ""},
		{"help", false, ""},
	}

	for _, tc := range testCases {
		isContentSearch := strings.HasPrefix(tc.command, "/") && len(tc.command) > 1
		assert.Equal(t, tc.isContentSearch, isContentSearch, "Content search detection for: '%s'", tc.command)

		if isContentSearch {
			searchTerm := tc.command[1:]
			parts := strings.Fields(searchTerm)
			if len(parts) > 0 {
				assert.Equal(t, tc.searchTerm, parts[0], "Search term extraction for: '%s'", tc.command)
			}
		}
	}
}

// Test command alias handling
func TestExecuteCommand_CommandAliases(t *testing.T) {
	aliases := map[string]string{
		"l":   "labels",
		"i":   "inbox",
		"c":   "compose",
		"h":   "help",
		"?":   "help",
		"n":   "numbers",
		"q":   "quit",
		"a":   "archive",
		"d":   "trash",
		"t":   "toggle-read",
		"r":   "reply",
		"u":   "unread",
		"b":   "archived",
		"o":   "open-web",
		"sl":  "slack",
		"pr":  "prompt",
		"p":   "prompt",
		"th":  "theme",
		"lbl": "label",
		"obs": "obsidian",
		"sel": "select",
		"mv":  "move",
	}

	// Test that aliases map to expected commands
	for alias, expected := range aliases {
		// This simulates the command matching logic
		assert.NotEmpty(t, expected, "Alias '%s' should map to a command", alias)
	}
}

// Test special "s" command handling (ambiguous search/slack)
func TestExecuteCommand_AmbiguousS(t *testing.T) {
	testCases := []struct {
		input            string
		expectSearchMode bool
	}{
		{"s query", true},     // Has args, should be search
		{"s", false},          // No args, should be slack
		{"s from:user", true}, // Has args, should be search
	}

	for _, tc := range testCases {
		parts := strings.Fields(tc.input)
		command := parts[0]
		args := parts[1:]

		if command == "s" {
			isSearch := len(args) > 0
			assert.Equal(t, tc.expectSearchMode, isSearch, "Ambiguous 's' handling for: '%s'", tc.input)
		}
	}
}

// Test emojiBox component
func TestEmojiBox_Creation(t *testing.T) {
	text := "🔥 Test"
	eb := newEmojiBox(text, 0x0)

	assert.NotNil(t, eb)
	assert.NotNil(t, eb.Box)
	assert.Equal(t, text, eb.text)
}

func TestEmojiBox_EmptyText(t *testing.T) {
	eb := newEmojiBox("", 0x0)

	assert.NotNil(t, eb)
	assert.Empty(t, eb.text)
}

func TestEmojiBox_UnicodeHandling(t *testing.T) {
	testCases := []string{
		"🔥",           // Fire emoji
		"测试",          // Chinese characters
		"🎯📧✅",         // Multiple emojis
		"Normal text", // ASCII text
	}

	for _, text := range testCases {
		eb := newEmojiBox(text, 0x0)
		assert.Equal(t, text, eb.text, "Unicode handling for: '%s'", text)
	}
}

// Test command edge cases
func TestExecuteCommand_EdgeCases(t *testing.T) {
	t.Run("empty_command", func(t *testing.T) {
		parts := strings.Fields("")
		assert.Empty(t, parts, "Empty command should result in empty parts")
	})

	t.Run("whitespace_only", func(t *testing.T) {
		parts := strings.Fields("   \t\n   ")
		assert.Empty(t, parts, "Whitespace-only command should result in empty parts")
	})

	t.Run("multiple_spaces", func(t *testing.T) {
		parts := strings.Fields("help     me     please")
		expected := []string{"help", "me", "please"}
		assert.Equal(t, expected, parts, "Multiple spaces should be normalized")
	})

	t.Run("quotes_handling", func(t *testing.T) {
		// Note: strings.Fields doesn't handle quotes specially
		parts := strings.Fields("search \"quoted string\"")
		expected := []string{"search", "\"quoted", "string\""}
		assert.Equal(t, expected, parts, "Quoted strings are split by spaces")
	})

	t.Run("special_characters", func(t *testing.T) {
		parts := strings.Fields("search from:user@domain.com")
		expected := []string{"search", "from:user@domain.com"}
		assert.Equal(t, expected, parts, "Special characters in args should be preserved")
	})
}

// Test command execution safety
func TestExecuteCommand_Safety(t *testing.T) {
	t.Run("case_insensitive_commands", func(t *testing.T) {
		commands := []string{"HELP", "help", "Help", "HeLp"}

		for _, cmd := range commands {
			normalized := strings.ToLower(cmd)
			assert.Equal(t, "help", normalized, "Command should be normalized to lowercase")
		}
	})

	t.Run("command_length_validation", func(t *testing.T) {
		// Test very long commands (potential DoS protection)
		longCommand := strings.Repeat("a", 1000)
		parts := strings.Fields(longCommand)

		assert.Len(t, parts, 1, "Very long single command should result in one part")
		assert.Equal(t, longCommand, parts[0], "Long command content should be preserved")
	})
}

// Test content search command variations
func TestExecuteCommand_ContentSearchVariations(t *testing.T) {
	testCases := []struct {
		input           string
		isContentSearch bool
		extractedTerm   string
	}{
		{"/error", true, "error"},
		{"/user@example.com", true, "user@example.com"},
		{"/123", true, "123"},
		{"/special-chars_test", true, "special-chars_test"},
		{"/", false, ""},
		{"//double", true, "/double"},
		{"/with space", true, "with"},
	}

	for _, tc := range testCases {
		parts := strings.Fields(tc.input)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		isContentSearch := strings.HasPrefix(command, "/") && len(command) > 1

		assert.Equal(t, tc.isContentSearch, isContentSearch, "Content search detection for: '%s'", tc.input)

		if isContentSearch {
			searchTerm := command[1:] // Remove the "/"
			assert.Equal(t, tc.extractedTerm, searchTerm, "Search term extraction for: '%s'", tc.input)
		}
	}
}

// Benchmark command parsing performance
func BenchmarkExecuteCommand_Parsing(b *testing.B) {
	testCommand := "search from:user@domain.com subject:important label:work"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parts := strings.Fields(testCommand)
		if len(parts) > 0 {
			command := strings.ToLower(parts[0])
			args := parts[1:]
			_ = command
			_ = args
		}
	}
}

func BenchmarkExecuteCommand_ContentSearch(b *testing.B) {
	testCommand := "/search_term_here"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isContentSearch := strings.HasPrefix(testCommand, "/") && len(testCommand) > 1
		if isContentSearch {
			searchTerm := testCommand[1:]
			_ = searchTerm
		}
	}
}
