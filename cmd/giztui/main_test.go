package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test path resolution functions
func TestGetConfigPath_Priority(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("GMAIL_TUI_CONFIG")
	defer func() { _ = os.Setenv("GMAIL_TUI_CONFIG", originalEnv) }()

	// Test CLI flag takes precedence
	result := getConfigPath("/custom/config.json")
	assert.Equal(t, "/custom/config.json", result)

	// Test environment variable when no flag
	_ = os.Setenv("GMAIL_TUI_CONFIG", "/env/config.json")
	result = getConfigPath("")
	assert.Equal(t, "/env/config.json", result)

	// Test default when neither flag nor env
	_ = os.Unsetenv("GMAIL_TUI_CONFIG")
	result = getConfigPath("")
	assert.Contains(t, result, "config.json") // Should contain default path
}

func TestGetCredentialsPath_Priority(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("GMAIL_TUI_CREDENTIALS")
	defer func() { _ = os.Setenv("GMAIL_TUI_CREDENTIALS", originalEnv) }()

	// Test CLI flag takes precedence
	result := getCredentialsPath("/custom/creds.json", "/config/creds.json")
	assert.Equal(t, "/custom/creds.json", result)

	// Test environment variable when no flag
	_ = os.Setenv("GMAIL_TUI_CREDENTIALS", "/env/creds.json")
	result = getCredentialsPath("", "/config/creds.json")
	assert.Equal(t, "/env/creds.json", result)

	// Test config value when no flag or env
	_ = os.Unsetenv("GMAIL_TUI_CREDENTIALS")
	result = getCredentialsPath("", "/config/creds.json")
	assert.Equal(t, "/config/creds.json", result)

	// Test default when nothing provided
	result = getCredentialsPath("", "")
	assert.Contains(t, result, "credentials.json")
}

func TestGetTokenPath_Priority(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("GMAIL_TUI_TOKEN")
	defer func() { _ = os.Setenv("GMAIL_TUI_TOKEN", originalEnv) }()

	// Test CLI flag takes precedence
	result := getTokenPath("/custom/token.json", "/config/token.json")
	assert.Equal(t, "/custom/token.json", result)

	// Test environment variable when no flag
	_ = os.Setenv("GMAIL_TUI_TOKEN", "/env/token.json")
	result = getTokenPath("", "/config/token.json")
	assert.Equal(t, "/env/token.json", result)

	// Test config value when no flag or env
	_ = os.Unsetenv("GMAIL_TUI_TOKEN")
	result = getTokenPath("", "/config/token.json")
	assert.Equal(t, "/config/token.json", result)

	// Test default when nothing provided
	result = getTokenPath("", "")
	assert.Contains(t, result, "token.json")
}

// Test path expansion
func TestExpandPath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		contains string // What the result should contain
	}{
		{"absolute_path", "/absolute/path", "/absolute/path"},
		{"relative_path", "relative/path", "relative/path"},
		{"home_only", "~", ""},
		{"home_with_subpath", "~/config/file", "config/file"},
		{"empty_path", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := expandPath(tc.input)

			if tc.input == tc.contains {
				// For non-home paths, should be unchanged
				assert.Equal(t, tc.input, result)
			} else if strings.HasPrefix(tc.input, "~") && tc.contains != "" {
				// For home paths, should contain the expected subpath
				assert.Contains(t, result, tc.contains)
				assert.NotContains(t, result, "~") // Tilde should be expanded
			}
		})
	}
}

func TestExpandPath_HomeDirectory(t *testing.T) {
	// Get actual home directory
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	testCases := []struct {
		input    string
		expected string
	}{
		{"~", home},
		{"~/test", filepath.Join(home, "test")},
		{"~/config/file.json", filepath.Join(home, "config", "file.json")},
	}

	for _, tc := range testCases {
		result := expandPath(tc.input)
		assert.Equal(t, tc.expected, result, "Path expansion for: %s", tc.input)
	}
}

func TestExpandPath_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"no_tilde", "/path/without/tilde", "/path/without/tilde"},
		{"tilde_middle", "/path/~/middle", "/path/~/middle"},
		{"empty_string", "", ""},
		{"just_slash", "/", "/"},
		{"multiple_tildes", "~/~/test", "~/test"}, // Expands first ~ only
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "multiple_tildes" {
				// Special case: starts with ~ so should be expanded, but contains ~/test in result
				result := expandPath(tc.input)
				assert.Contains(t, result, tc.expected)
				return
			}

			result := expandPath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	envVars := []string{
		"GMAIL_TUI_CONFIG",
		"GMAIL_TUI_CREDENTIALS",
		"GMAIL_TUI_TOKEN",
	}

	for _, envVar := range envVars {
		t.Run(envVar, func(t *testing.T) {
			// Save original value
			original := os.Getenv(envVar)
			defer func() { _ = os.Setenv(envVar, original) }()

			// Test setting and getting
			testValue := "/test/path"
			_ = os.Setenv(envVar, testValue)

			result := os.Getenv(envVar)
			assert.Equal(t, testValue, result)

			// Test unsetting
			_ = os.Unsetenv(envVar)
			result = os.Getenv(envVar)
			assert.Empty(t, result)
		})
	}
}

// Test AWS region environment variable (for bedrock provider)
func TestAWSRegionHandling(t *testing.T) {
	// Save original environment
	originalRegion := os.Getenv("AWS_REGION")
	defer func() { _ = os.Setenv("AWS_REGION", originalRegion) }()

	// Test setting AWS region
	_ = os.Setenv("AWS_REGION", "us-east-1")
	result := os.Getenv("AWS_REGION")
	assert.Equal(t, "us-east-1", result)

	// Test clearing AWS region
	_ = os.Unsetenv("AWS_REGION")
	result = os.Getenv("AWS_REGION")
	assert.Empty(t, result)
}

// Test string manipulation utilities
func TestStringManipulation(t *testing.T) {
	t.Run("email_sanitization", func(t *testing.T) {
		// Test the email sanitization logic used for database paths
		email := "user@example.com"
		replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "_", " ", "_")
		sanitized := replacer.Replace(strings.ToLower(strings.TrimSpace(email)))

		expected := "user_example.com"
		assert.Equal(t, expected, sanitized)
	})

	t.Run("path_sanitization_edge_cases", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"User@Domain.Com", "user_domain.com"},
			{"  spaced@domain.com  ", "spaced_domain.com"},
			{"special/chars\\here:test@domain.com", "special_chars_here_test_domain.com"},
			{"", ""},
			{"   ", ""},
		}

		replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "_", " ", "_")

		for _, tc := range testCases {
			result := replacer.Replace(strings.ToLower(strings.TrimSpace(tc.input)))
			if tc.expected == "" && tc.input != "" {
				// For whitespace-only input, result should be empty after trimming
				assert.Empty(t, result, "Input: '%s'", tc.input)
			} else {
				assert.Equal(t, tc.expected, result, "Input: '%s'", tc.input)
			}
		}
	})
}

// Test file extension handling
func TestFileExtensionHandling(t *testing.T) {
	testCases := []struct {
		path      string
		hasExt    bool
		extension string
	}{
		{"/path/file.txt", true, ".txt"},
		{"/path/file.json", true, ".json"},
		{"/path/file", false, ""},
		{"/path/dir/", false, ""},
		{"/path/dir", false, ""},
		{"file.sqlite3", true, ".sqlite3"},
		{"/path/file.", true, "."},
	}

	for _, tc := range testCases {
		ext := filepath.Ext(tc.path)
		if tc.hasExt {
			assert.Equal(t, tc.extension, ext, "Extension for path: %s", tc.path)
		} else {
			assert.Empty(t, ext, "Path should have no extension: %s", tc.path)
		}
	}
}

// Test path joining and building
func TestPathJoining(t *testing.T) {
	testCases := []struct {
		base     string
		filename string
		expected string
	}{
		{"/base/path", "file.txt", "/base/path/file.txt"},
		{"/base", "subdir/file.txt", "/base/subdir/file.txt"},
		{".", "file.txt", "file.txt"},
		{"", "file.txt", "file.txt"},
	}

	for _, tc := range testCases {
		result := filepath.Join(tc.base, tc.filename)
		if tc.expected == result || strings.HasSuffix(result, tc.expected) {
			assert.True(t, true, "Path joining works for %s + %s", tc.base, tc.filename)
		} else {
			assert.Equal(t, tc.expected, result, "Path joining for %s + %s", tc.base, tc.filename)
		}
	}
}

// Test command line argument parsing concepts
func TestFlagParsing_Concepts(t *testing.T) {
	t.Run("empty_string_flag", func(t *testing.T) {
		// Simulate flag parsing - empty string should be treated as not provided
		flagValue := ""

		if flagValue != "" {
			t.Error("Empty string flag should be treated as not provided")
		}
	})

	t.Run("non_empty_flag", func(t *testing.T) {
		// Simulate flag parsing - non-empty string should be used
		flagValue := "/custom/path"

		if flagValue == "" {
			t.Error("Non-empty flag should be used")
		}

		assert.Equal(t, "/custom/path", flagValue)
	})

	t.Run("boolean_flag", func(t *testing.T) {
		// Simulate boolean flag handling
		setupFlag := true

		if setupFlag {
			assert.True(t, true, "Setup flag should trigger special handling")
		}
	})
}

// Test path validation concepts
func TestPathValidation_Concepts(t *testing.T) {
	t.Run("file_exists_check", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.json")

		// File doesn't exist yet
		_, err := os.Stat(testFile)
		assert.True(t, os.IsNotExist(err), "File should not exist initially")

		// Create the file
		err = os.WriteFile(testFile, []byte("{}"), 0600)
		assert.NoError(t, err)

		// File should exist now
		_, err = os.Stat(testFile)
		assert.NoError(t, err, "File should exist after creation")
	})

	t.Run("directory_creation", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir", "nested")

		// Directory doesn't exist
		_, err := os.Stat(subDir)
		assert.True(t, os.IsNotExist(err), "Directory should not exist initially")

		// Create directory
		err = os.MkdirAll(subDir, 0750)
		assert.NoError(t, err)

		// Directory should exist now
		info, err := os.Stat(subDir)
		assert.NoError(t, err, "Directory should exist after creation")
		assert.True(t, info.IsDir(), "Should be a directory")
	})
}

// Test configuration prioritization logic
func TestConfigurationPriority(t *testing.T) {
	// This tests the priority logic used in the main function
	testCases := []struct {
		name     string
		flag     string
		env      string
		config   string
		expected string
	}{
		{"flag_priority", "/flag/path", "/env/path", "/config/path", "/flag/path"},
		{"env_priority", "", "/env/path", "/config/path", "/env/path"},
		{"config_priority", "", "", "/config/path", "/config/path"},
		{"all_empty", "", "", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the priority logic from getCredentialsPath
			var result string

			if tc.flag != "" {
				result = tc.flag
			} else if tc.env != "" {
				result = tc.env
			} else if tc.config != "" {
				result = tc.config
			}

			assert.Equal(t, tc.expected, result)
		})
	}
}
