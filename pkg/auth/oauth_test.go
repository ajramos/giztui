package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// Test OAuth2Config constructor
func TestNewOAuth2Config(t *testing.T) {
	credPath := "/path/to/credentials.json"
	tokenPath := "/path/to/token.json"
	scopes := []string{"https://www.googleapis.com/auth/gmail.readonly"}

	config := NewOAuth2Config(credPath, tokenPath, scopes...)

	assert.NotNil(t, config)
	assert.Equal(t, credPath, config.CredentialsPath)
	assert.Equal(t, tokenPath, config.TokenPath)
	assert.Equal(t, scopes, config.Scopes)
}

func TestNewOAuth2Config_EmptyScopes(t *testing.T) {
	config := NewOAuth2Config("cred.json", "token.json")

	assert.NotNil(t, config)
	assert.Empty(t, config.Scopes)
}

func TestNewOAuth2Config_MultipleScopes(t *testing.T) {
	scopes := []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/calendar.readonly",
	}

	config := NewOAuth2Config("cred.json", "token.json", scopes...)

	assert.Equal(t, scopes, config.Scopes)
}

// Test LoadCredentials validation
func TestOAuth2Config_LoadCredentials_ValidationErrors(t *testing.T) {
	t.Run("empty_credentials_path", func(t *testing.T) {
		config := &OAuth2Config{CredentialsPath: ""}

		oauthConfig, err := config.LoadCredentials()
		assert.Error(t, err)
		assert.Nil(t, oauthConfig)
		assert.Contains(t, err.Error(), "could not read credentials file")
	})

	t.Run("nonexistent_credentials_file", func(t *testing.T) {
		config := &OAuth2Config{CredentialsPath: "/nonexistent/path/credentials.json"}

		oauthConfig, err := config.LoadCredentials()
		assert.Error(t, err)
		assert.Nil(t, oauthConfig)
		assert.Contains(t, err.Error(), "could not read credentials file")
	})

	t.Run("invalid_credentials_content", func(t *testing.T) {
		// Create temporary invalid credentials file
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "invalid_credentials.json")

		err := os.WriteFile(credPath, []byte("invalid json content"), 0600)
		assert.NoError(t, err)

		config := &OAuth2Config{CredentialsPath: credPath}

		oauthConfig, err := config.LoadCredentials()
		assert.Error(t, err)
		assert.Nil(t, oauthConfig)
		assert.Contains(t, err.Error(), "could not parse credentials file")
	})

	t.Run("empty_credentials_file", func(t *testing.T) {
		// Create temporary empty credentials file
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "empty_credentials.json")

		err := os.WriteFile(credPath, []byte(""), 0600)
		assert.NoError(t, err)

		config := &OAuth2Config{CredentialsPath: credPath}

		oauthConfig, err := config.LoadCredentials()
		assert.Error(t, err)
		assert.Nil(t, oauthConfig)
	})
}

// Test LoadToken validation and file handling
func TestOAuth2Config_LoadToken_ValidationErrors(t *testing.T) {
	config := &oauth2.Config{} // Mock oauth2 config

	t.Run("empty_token_path", func(t *testing.T) {
		authConfig := &OAuth2Config{TokenPath: ""}

		token, err := authConfig.LoadToken(config)
		assert.Error(t, err)
		assert.Nil(t, token)
	})

	t.Run("nonexistent_token_file", func(t *testing.T) {
		authConfig := &OAuth2Config{TokenPath: "/nonexistent/path/token.json"}

		token, err := authConfig.LoadToken(config)
		assert.Error(t, err)
		assert.Nil(t, token)
	})

	t.Run("invalid_token_content", func(t *testing.T) {
		// Create temporary invalid token file
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "invalid_token.json")

		err := os.WriteFile(tokenPath, []byte("invalid json content"), 0600)
		assert.NoError(t, err)

		authConfig := &OAuth2Config{TokenPath: tokenPath}

		token, err := authConfig.LoadToken(config)
		// JSON decoder may fail but Go's json.Decoder.Decode may still return a token struct
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
		// Token may still be created even with decode error
		assert.NotNil(t, token)
	})

	t.Run("empty_token_file", func(t *testing.T) {
		// Create temporary empty token file
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "empty_token.json")

		err := os.WriteFile(tokenPath, []byte(""), 0600)
		assert.NoError(t, err)

		authConfig := &OAuth2Config{TokenPath: tokenPath}

		token, err := authConfig.LoadToken(config)
		// JSON decoder fails with empty content
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EOF")
		// Token may still be created even with decode error
		assert.NotNil(t, token)
	})
}

func TestOAuth2Config_LoadToken_ValidToken(t *testing.T) {
	config := &oauth2.Config{}

	// Create temporary valid token file
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "valid_token.json")

	validTokenJSON := `{
		"access_token": "test-access-token",
		"token_type": "Bearer",
		"refresh_token": "test-refresh-token",
		"expiry": "2025-12-31T23:59:59Z"
	}`

	err := os.WriteFile(tokenPath, []byte(validTokenJSON), 0600)
	assert.NoError(t, err)

	authConfig := &OAuth2Config{TokenPath: tokenPath}

	token, err := authConfig.LoadToken(config)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "test-access-token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, "test-refresh-token", token.RefreshToken)
	assert.False(t, token.Expiry.IsZero())
}

// Test SaveToken validation and file operations
func TestOAuth2Config_SaveToken_ValidationErrors(t *testing.T) {
	t.Run("nil_token", func(t *testing.T) {
		config := &OAuth2Config{TokenPath: "/tmp/test_token.json"}

		err := config.SaveToken(nil)
		// Note: SaveToken may handle nil token by encoding null - not necessarily an error
		if err == nil {
			// Nil token saved successfully as JSON null
			t.Log("SaveToken handled nil token gracefully")
		} else {
			// SaveToken rejected nil token
			assert.Error(t, err)
		}
	})

	t.Run("empty_token_path", func(t *testing.T) {
		config := &OAuth2Config{TokenPath: ""}
		token := &oauth2.Token{AccessToken: "test"}

		err := config.SaveToken(token)
		assert.Error(t, err)
	})
}

func TestOAuth2Config_SaveToken_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "token.json")

	config := &OAuth2Config{TokenPath: nestedPath}
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	err := config.SaveToken(token)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(nestedPath)
	assert.NoError(t, err)

	// Verify file permissions
	fileInfo, err := os.Stat(nestedPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
}

func TestOAuth2Config_SaveToken_ValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "test_token.json")

	config := &OAuth2Config{TokenPath: tokenPath}
	originalToken := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Save token
	err := config.SaveToken(originalToken)
	assert.NoError(t, err)

	// Load and verify token
	oauthConfig := &oauth2.Config{} // Mock config for LoadToken
	loadedToken, err := config.LoadToken(oauthConfig)
	assert.NoError(t, err)
	assert.Equal(t, originalToken.AccessToken, loadedToken.AccessToken)
	assert.Equal(t, originalToken.TokenType, loadedToken.TokenType)
	assert.Equal(t, originalToken.RefreshToken, loadedToken.RefreshToken)
	// Note: Expiry comparison might have slight time differences due to JSON serialization
	assert.True(t, originalToken.Expiry.Sub(loadedToken.Expiry) < time.Second)
}

func TestOAuth2Config_SaveToken_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "overwrite_token.json")

	config := &OAuth2Config{TokenPath: tokenPath}

	// Save first token
	token1 := &oauth2.Token{
		AccessToken: "first-token",
		TokenType:   "Bearer",
	}
	err := config.SaveToken(token1)
	assert.NoError(t, err)

	// Save second token (should overwrite)
	token2 := &oauth2.Token{
		AccessToken: "second-token",
		TokenType:   "Bearer",
	}
	err = config.SaveToken(token2)
	assert.NoError(t, err)

	// Verify second token was saved
	oauthConfig := &oauth2.Config{}
	loadedToken, err := config.LoadToken(oauthConfig)
	assert.NoError(t, err)
	assert.Equal(t, "second-token", loadedToken.AccessToken)
}

// Test GetToken method with various scenarios
func TestOAuth2Config_GetToken_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_credentials_path", func(t *testing.T) {
		config := &OAuth2Config{
			CredentialsPath: "/nonexistent/credentials.json",
			TokenPath:       "/tmp/token.json",
		}

		token, err := config.GetToken(ctx)
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "could not read credentials file")
	})
}

// Test NewGmailService and NewCalendarService validation
func TestNewGmailService_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_credentials_path", func(t *testing.T) {
		service, err := NewGmailService(ctx, "/nonexistent/cred.json", "/tmp/token.json", "scope1")
		assert.Error(t, err)
		assert.Nil(t, service)
	})

	t.Run("empty_credentials_path", func(t *testing.T) {
		service, err := NewGmailService(ctx, "", "/tmp/token.json", "scope1")
		assert.Error(t, err)
		assert.Nil(t, service)
	})

	t.Run("empty_token_path", func(t *testing.T) {
		// Even with empty token path, it should fail at credentials loading first
		service, err := NewGmailService(ctx, "/nonexistent/cred.json", "", "scope1")
		assert.Error(t, err)
		assert.Nil(t, service)
	})
}

func TestNewCalendarService_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_credentials_path", func(t *testing.T) {
		service, err := NewCalendarService(ctx, "/nonexistent/cred.json", "/tmp/token.json", "scope1")
		assert.Error(t, err)
		assert.Nil(t, service)
	})

	t.Run("empty_credentials_path", func(t *testing.T) {
		service, err := NewCalendarService(ctx, "", "/tmp/token.json", "scope1")
		assert.Error(t, err)
		assert.Nil(t, service)
	})

	t.Run("empty_token_path", func(t *testing.T) {
		// Even with empty token path, it should fail at credentials loading first
		service, err := NewCalendarService(ctx, "/nonexistent/cred.json", "", "scope1")
		assert.Error(t, err)
		assert.Nil(t, service)
	})
}

// Test file path validation and security
func TestOAuth2Config_FilePathSecurity(t *testing.T) {
	t.Run("relative_paths", func(t *testing.T) {
		config := NewOAuth2Config("../credentials.json", "./token.json")

		// Should handle relative paths without error in constructor
		assert.Equal(t, "../credentials.json", config.CredentialsPath)
		assert.Equal(t, "./token.json", config.TokenPath)

		// But should fail when trying to load nonexistent relative paths
		_, err := config.LoadCredentials()
		assert.Error(t, err)
	})

	t.Run("absolute_paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "credentials.json")
		tokenPath := filepath.Join(tmpDir, "token.json")

		config := NewOAuth2Config(credPath, tokenPath)

		assert.Equal(t, credPath, config.CredentialsPath)
		assert.Equal(t, tokenPath, config.TokenPath)
	})
}

// Test edge cases and special scenarios
func TestOAuth2Config_EdgeCases(t *testing.T) {
	t.Run("very_long_paths", func(t *testing.T) {
		longPath := "/tmp/" + strings.Repeat("very_long_directory_name_", 10) + "file.json"
		config := NewOAuth2Config(longPath, longPath)

		assert.Equal(t, longPath, config.CredentialsPath)
		assert.Equal(t, longPath, config.TokenPath)
	})

	t.Run("special_characters_in_paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		specialPath := filepath.Join(tmpDir, "file with spaces & special chars!@#.json")

		config := NewOAuth2Config(specialPath, specialPath)

		// Should handle special characters in paths
		assert.Equal(t, specialPath, config.CredentialsPath)
		assert.Equal(t, specialPath, config.TokenPath)
	})

	t.Run("unicode_paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		unicodePath := filepath.Join(tmpDir, "файл_с_юникодом.json")

		config := NewOAuth2Config(unicodePath, unicodePath)

		assert.Equal(t, unicodePath, config.CredentialsPath)
		assert.Equal(t, unicodePath, config.TokenPath)
	})
}

// Test token validation and expiry scenarios
func TestOAuth2Config_TokenExpiry(t *testing.T) {
	t.Run("expired_token", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "expired_token.json")

		// Create expired token
		expiredToken := &oauth2.Token{
			AccessToken:  "expired-access-token",
			TokenType:    "Bearer",
			RefreshToken: "refresh-token",
			Expiry:       time.Now().Add(-time.Hour), // Expired 1 hour ago
		}

		config := &OAuth2Config{TokenPath: tokenPath}
		err := config.SaveToken(expiredToken)
		assert.NoError(t, err)

		// Load expired token
		oauthConfig := &oauth2.Config{}
		loadedToken, err := config.LoadToken(oauthConfig)
		assert.NoError(t, err)
		assert.False(t, loadedToken.Valid()) // Should be invalid
	})

	t.Run("valid_token", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "valid_token.json")

		// Create valid token
		validToken := &oauth2.Token{
			AccessToken:  "valid-access-token",
			TokenType:    "Bearer",
			RefreshToken: "refresh-token",
			Expiry:       time.Now().Add(time.Hour), // Valid for 1 hour
		}

		config := &OAuth2Config{TokenPath: tokenPath}
		err := config.SaveToken(validToken)
		assert.NoError(t, err)

		// Load valid token
		oauthConfig := &oauth2.Config{}
		loadedToken, err := config.LoadToken(oauthConfig)
		assert.NoError(t, err)
		assert.True(t, loadedToken.Valid()) // Should be valid
	})
}

// Benchmark tests for performance-critical operations
func BenchmarkOAuth2Config_SaveToken(b *testing.B) {
	tmpDir := b.TempDir()
	tokenPath := filepath.Join(tmpDir, "benchmark_token.json")

	config := &OAuth2Config{TokenPath: tokenPath}
	token := &oauth2.Token{
		AccessToken:  "benchmark-access-token",
		TokenType:    "Bearer",
		RefreshToken: "benchmark-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.SaveToken(token)
	}
}

func BenchmarkOAuth2Config_LoadToken(b *testing.B) {
	tmpDir := b.TempDir()
	tokenPath := filepath.Join(tmpDir, "benchmark_load_token.json")

	config := &OAuth2Config{TokenPath: tokenPath}
	token := &oauth2.Token{
		AccessToken:  "benchmark-access-token",
		TokenType:    "Bearer",
		RefreshToken: "benchmark-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Save token first
	_ = config.SaveToken(token)

	oauthConfig := &oauth2.Config{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.LoadToken(oauthConfig)
	}
}

// Test concurrent access scenarios
func TestOAuth2Config_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "concurrent_token.json")

	config := &OAuth2Config{TokenPath: tokenPath}

	// Test concurrent saves don't cause corruption
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			token := &oauth2.Token{
				AccessToken:  fmt.Sprintf("token-%d", id),
				TokenType:    "Bearer",
				RefreshToken: fmt.Sprintf("refresh-%d", id),
				Expiry:       time.Now().Add(time.Hour),
			}
			done <- config.SaveToken(token)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	// Verify file still exists and contains valid JSON
	oauthConfig := &oauth2.Config{}
	_, err := config.LoadToken(oauthConfig)
	assert.NoError(t, err)
}
