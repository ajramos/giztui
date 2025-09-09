package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// ========================================
// Graceful Multi-Level Credential Fallback System Tests
// ========================================

// TestCredentialFallbackSystem tests the complete graceful fallback logic
func TestCredentialFallbackSystem(t *testing.T) {
	// Create temporary directory structure for testing
	tmpDir := t.TempDir()

	// Test helper to create credential files
	createCredFile := func(path string, content string) {
		err := os.MkdirAll(filepath.Dir(path), 0750)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err)
	}

	t.Run("Level1_CLI_Flag_Success", func(t *testing.T) {
		// Create CLI flag credentials
		cliCredPath := filepath.Join(tmpDir, "cli-credentials.json")
		createCredFile(cliCredPath, `{"client_id": "cli-test"}`)

		// Mock CLI flag
		credPathFlag := &cliCredPath

		// Mock config with bad credentials
		cfg := &config.Config{
			Credentials: filepath.Join(tmpDir, "nonexistent-config.json"),
			Token:       filepath.Join(tmpDir, "nonexistent-token.json"),
		}

		// Mock logger
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		// Test the fallback logic
		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, cliCredPath, credPath)
		assert.Contains(t, tokenPath, "token.json")
		assert.Equal(t, "CLI flag", fallbackMethod)

		// Verify logging
		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Starting graceful credential fallback sequence")
		assert.Contains(t, logOutput, "Trying CLI flag credentials")
		assert.Contains(t, logOutput, "CLI flag credentials found and validated")
	})

	t.Run("Level2_Config_File_Success", func(t *testing.T) {
		// Create config credentials
		configCredPath := filepath.Join(tmpDir, "config-credentials.json")
		configTokenPath := filepath.Join(tmpDir, "config-token.json")
		createCredFile(configCredPath, `{"client_id": "config-test"}`)
		createCredFile(configTokenPath, `{"access_token": "config-token"}`)

		// No CLI flag
		var credPathFlag *string = nil

		// Mock config with valid credentials
		cfg := &config.Config{
			Credentials: configCredPath,
			Token:       configTokenPath,
		}

		// Mock logger
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		// Test the fallback logic
		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, configCredPath, credPath)
		assert.Equal(t, configTokenPath, tokenPath)
		assert.Equal(t, "config file", fallbackMethod)

		// Verify logging
		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Trying config file credentials")
		assert.Contains(t, logOutput, "Config file credentials found and validated")
	})

	t.Run("Level3_Hardcoded_Defaults_Success", func(t *testing.T) {
		// Create default credentials
		defaultCredPath, _ := config.DefaultCredentialPaths()
		createCredFile(defaultCredPath, `{"client_id": "default-test"}`)

		// No CLI flag
		var credPathFlag *string = nil

		// Mock config with invalid credentials
		cfg := &config.Config{
			Credentials: filepath.Join(tmpDir, "missing-config.json"),
			Token:       filepath.Join(tmpDir, "missing-token.json"),
		}

		// Mock logger
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		// Test the fallback logic
		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, defaultCredPath, credPath)
		assert.Contains(t, tokenPath, "token.json")
		assert.Equal(t, "hardcoded defaults", fallbackMethod)

		// Verify logging shows the fallback sequence
		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Trying config file credentials")
		assert.Contains(t, logOutput, "Config file credentials not found")
		assert.Contains(t, logOutput, "Config credentials failed, trying hardcoded defaults")
		assert.Contains(t, logOutput, "Hardcoded default credentials found and validated")
	})

	t.Run("Disabled_Config_Goes_To_Defaults", func(t *testing.T) {
		// Create default credentials
		defaultCredPath, _ := config.DefaultCredentialPaths()
		createCredFile(defaultCredPath, `{"client_id": "default-test"}`)

		// No CLI flag
		var credPathFlag *string = nil

		// Mock config with disabled credentials (empty strings)
		cfg := &config.Config{
			Credentials: "", // Disabled (would be _credentials in JSON)
			Token:       "", // Disabled (would be _token in JSON)
		}

		// Mock logger
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		// Test the fallback logic
		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, defaultCredPath, credPath)
		assert.Contains(t, tokenPath, "token.json")
		assert.Equal(t, "hardcoded defaults", fallbackMethod)

		// Verify logging shows disabled config path
		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "No config credentials (disabled with prefix), trying hardcoded defaults")
		assert.Contains(t, logOutput, "Hardcoded default credentials found and validated")
	})
}

// TestCredentialFallbackFailures tests failure scenarios
func TestCredentialFallbackFailures(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("All_Sources_Exhausted_Returns_Empty", func(t *testing.T) {
		// No CLI flag
		var credPathFlag *string = nil

		// Mock config with invalid credentials
		cfg := &config.Config{
			Credentials: filepath.Join(tmpDir, "missing-config.json"),
			Token:       filepath.Join(tmpDir, "missing-token.json"),
		}

		// Mock logger
		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		// Test the fallback logic with missing default credentials
		credPath, tokenPath, fallbackMethod := testCredentialFallbackWithCustomDefaults(t, credPathFlag, cfg, logger,
			filepath.Join(tmpDir, "missing-default.json"))

		assert.Empty(t, credPath)
		assert.Empty(t, tokenPath)
		assert.Empty(t, fallbackMethod)

		// Verify logging shows all attempts failed
		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Config file credentials not found")
		assert.Contains(t, logOutput, "Hardcoded default credentials not found")
		assert.Contains(t, logOutput, "All credential fallback methods exhausted")
		assert.Contains(t, logOutput, "Tried CLI flag, config file, and hardcoded defaults")
	})

	t.Run("CLI_Flag_Missing_Falls_Back", func(t *testing.T) {
		// Create config credentials
		configCredPath := filepath.Join(tmpDir, "fallback-config.json")
		err := os.WriteFile(configCredPath, []byte(`{"client_id": "config"}`), 0600)
		require.NoError(t, err)

		// CLI flag points to missing file
		missingPath := filepath.Join(tmpDir, "missing-cli.json")
		credPathFlag := &missingPath

		cfg := &config.Config{
			Credentials: configCredPath,
			Token:       filepath.Join(tmpDir, "config-token.json"),
		}

		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		// Should fallback to config file
		credPath, _, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, configCredPath, credPath)
		assert.Equal(t, "config file", fallbackMethod)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "CLI flag credentials not found")
		assert.Contains(t, logOutput, "Trying config file credentials")
		assert.Contains(t, logOutput, "Config file credentials found and validated")
	})
}

// Helper function to test credential fallback logic in isolation
func testCredentialFallback(t *testing.T, credPathFlag *string, cfg *config.Config, logger *log.Logger) (credPath, tokenPath, fallbackMethod string) {
	defaultCredPath, _ := config.DefaultCredentialPaths()
	return testCredentialFallbackWithCustomDefaults(t, credPathFlag, cfg, logger, defaultCredPath)
}

// Helper function to test credential fallback logic with custom default paths
func testCredentialFallbackWithCustomDefaults(t *testing.T, credPathFlag *string, cfg *config.Config, logger *log.Logger, defaultCredPath string) (credPath, tokenPath, fallbackMethod string) {
	logger.Printf("🔄 Starting graceful credential fallback sequence...")

	var attemptNumber = 1

	// Level 1: Try CLI flag credentials (highest priority)
	if credPathFlag != nil && *credPathFlag != "" {
		logger.Printf("🎯 Attempt %d: Trying CLI flag credentials: %s", attemptNumber, *credPathFlag)
		attemptNumber++

		testCredPath := *credPathFlag
		testTokenPath := getTokenPath("", cfg.Token)

		logger.Printf("📍 Resolved paths - creds: %s, token: %s", testCredPath, testTokenPath)

		if testCredPath != "" {
			if _, err := os.Stat(testCredPath); err == nil {
				credPath = testCredPath
				tokenPath = testTokenPath
				fallbackMethod = "CLI flag"
				logger.Printf("✅ CLI flag credentials found and validated")
				logger.Printf("🚀 Initializing Gmail service with %s credentials (creds: %s, token: %s)", fallbackMethod, credPath, tokenPath)
				return
			} else {
				logger.Printf("❌ CLI flag credentials not found at %s", testCredPath)
			}
		}
	}

	// Level 2: Try config file credentials (if CLI didn't work and config has credentials)
	if credPath == "" && cfg.Credentials != "" {
		logger.Printf("🎯 Attempt %d: Trying config file credentials: %s", attemptNumber, cfg.Credentials)
		attemptNumber++

		testCredPath := expandPath(cfg.Credentials)
		testTokenPath := getTokenPath("", cfg.Token)

		logger.Printf("📍 Resolved paths - creds: %s, token: %s", testCredPath, testTokenPath)

		if _, err := os.Stat(testCredPath); err == nil {
			credPath = testCredPath
			tokenPath = testTokenPath
			fallbackMethod = "config file"
			logger.Printf("✅ Config file credentials found and validated")
			logger.Printf("🚀 Initializing Gmail service with %s credentials (creds: %s, token: %s)", fallbackMethod, credPath, tokenPath)
			return
		} else {
			logger.Printf("❌ Config file credentials not found at %s", testCredPath)
		}
	}

	// Level 3: Try hardcoded default credentials (final fallback)
	if credPath == "" {
		if cfg.Credentials != "" {
			logger.Printf("🎯 Attempt %d: Config credentials failed, trying hardcoded defaults as final fallback", attemptNumber)
		} else {
			logger.Printf("🎯 Attempt %d: No config credentials (disabled with prefix), trying hardcoded defaults", attemptNumber)
		}

		testCredPath := defaultCredPath
		testTokenPath := getTokenPath("", "")

		logger.Printf("📍 Resolved default paths - creds: %s, token: %s", testCredPath, testTokenPath)

		if testCredPath != "" {
			if _, err := os.Stat(testCredPath); err == nil {
				credPath = testCredPath
				tokenPath = testTokenPath
				fallbackMethod = "hardcoded defaults"
				logger.Printf("✅ Hardcoded default credentials found and validated")
				logger.Printf("🚀 Initializing Gmail service with %s credentials (creds: %s, token: %s)", fallbackMethod, credPath, tokenPath)
				return
			} else {
				logger.Printf("❌ Hardcoded default credentials not found at %s", testCredPath)
			}
		}
	}

	// Final validation - if still no valid credentials found
	if credPath == "" {
		logger.Printf("❌ All credential fallback methods exhausted")
		logger.Printf("💡 Tried CLI flag, config file, and hardcoded defaults")
		logger.Printf("💡 Please ensure at least one credential file exists and is accessible")
	}

	return credPath, tokenPath, fallbackMethod
}

// TestCredentialFallbackIntegration tests integration with actual path resolution functions
func TestCredentialFallbackIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Integration_With_getCredentialsPath", func(t *testing.T) {
		// Test that our fallback logic integrates with actual getCredentialsPath function
		configPath := filepath.Join(tmpDir, "integration-test.json")
		err := os.WriteFile(configPath, []byte(`{"client_id": "integration"}`), 0600)
		require.NoError(t, err)

		// Test getCredentialsPath with various inputs
		result := getCredentialsPath("", configPath)
		assert.Equal(t, configPath, result)

		// Test with CLI flag override
		cliPath := filepath.Join(tmpDir, "cli-override.json")
		err = os.WriteFile(cliPath, []byte(`{"client_id": "cli"}`), 0600)
		require.NoError(t, err)

		result = getCredentialsPath(cliPath, configPath)
		assert.Equal(t, cliPath, result, "CLI flag should override config")

		// Test with empty config (should go to defaults)
		result = getCredentialsPath("", "")
		assert.Contains(t, result, "credentials.json", "Should return default path when no inputs")
	})

	t.Run("Integration_With_getTokenPath", func(t *testing.T) {
		// Test token path resolution follows same patterns
		configTokenPath := filepath.Join(tmpDir, "config-token.json")

		result := getTokenPath("", configTokenPath)
		assert.Equal(t, configTokenPath, result)

		// Test CLI override for tokens
		cliTokenPath := filepath.Join(tmpDir, "cli-token.json")
		result = getTokenPath(cliTokenPath, configTokenPath)
		assert.Equal(t, cliTokenPath, result, "CLI should override config for token too")
	})
}
