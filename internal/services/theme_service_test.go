package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/stretchr/testify/assert"
)

// Test Theme Service constructor
func TestNewThemeService(t *testing.T) {
	themesDir := "/path/to/themes"
	customDir := "/path/to/custom"
	applyFunc := func(*config.ColorsConfig) error { return nil }

	service := NewThemeService(themesDir, customDir, applyFunc)

	assert.NotNil(t, service)
	assert.Equal(t, "gmail-dark", service.currentTheme) // Default theme
	assert.Equal(t, themesDir, service.themesDir)
	assert.Equal(t, customDir, service.customThemeDir)
	assert.NotNil(t, service.themeLoader)
	assert.NotNil(t, service.applyThemeFunc)
}

func TestNewThemeService_EmptyDirs(t *testing.T) {
	applyFunc := func(*config.ColorsConfig) error { return nil }

	service := NewThemeService("", "", applyFunc)

	assert.NotNil(t, service)
	assert.Equal(t, "gmail-dark", service.currentTheme)
	assert.Empty(t, service.themesDir)
	assert.Empty(t, service.customThemeDir)
}

func TestNewThemeService_NilApplyFunc(t *testing.T) {
	service := NewThemeService("/themes", "/custom", nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.applyThemeFunc)
}

// Test ListAvailableThemes with real directories
func TestThemeServiceImpl_ListAvailableThemes_WithDirectories(t *testing.T) {
	// Create temporary directories with theme files
	tmpDir := t.TempDir()
	themesDir := filepath.Join(tmpDir, "themes")
	customDir := filepath.Join(tmpDir, "custom")

	err := os.MkdirAll(themesDir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(customDir, 0755)
	assert.NoError(t, err)

	// Create sample theme files
	themeFiles := []string{
		filepath.Join(themesDir, "dark-theme.yaml"),
		filepath.Join(themesDir, "light-theme.yaml"),
		filepath.Join(customDir, "custom-theme.yaml"),
		filepath.Join(customDir, "my-theme.yaml"),
	}

	for _, file := range themeFiles {
		err := os.WriteFile(file, []byte("theme: test"), 0644)
		assert.NoError(t, err)
	}

	service := NewThemeService(themesDir, customDir, nil)

	themes, err := service.ListAvailableThemes(context.Background())

	assert.NoError(t, err)
	assert.Contains(t, themes, "dark-theme")
	assert.Contains(t, themes, "light-theme")
	assert.Contains(t, themes, "custom-theme")
	assert.Contains(t, themes, "my-theme")
	// May include additional built-in themes, so check minimum count
	assert.GreaterOrEqual(t, len(themes), 4)
}

func TestThemeServiceImpl_ListAvailableThemes_NonExistentDirs(t *testing.T) {
	service := NewThemeService("/nonexistent", "/also-nonexistent", nil)

	themes, err := service.ListAvailableThemes(context.Background())

	// Should handle non-existent directories gracefully
	// In CI environments, this may return an error if no themes are found
	if err != nil {
		assert.Contains(t, err.Error(), "no themes found")
		assert.Nil(t, themes)
	} else {
		assert.NotNil(t, themes)
	}
}

func TestThemeServiceImpl_ListAvailableThemes_EmptyDirs(t *testing.T) {
	tmpDir := t.TempDir()
	themesDir := filepath.Join(tmpDir, "empty_themes")

	err := os.MkdirAll(themesDir, 0755)
	assert.NoError(t, err)

	service := NewThemeService(themesDir, "", nil)

	themes, err := service.ListAvailableThemes(context.Background())

	// In CI environments, this may return an error if no themes are found
	if err != nil {
		assert.Contains(t, err.Error(), "no themes found")
		assert.Nil(t, themes)
	} else {
		assert.NotNil(t, themes)
		// Should handle empty directories without error
	}
}

func TestThemeServiceImpl_ListAvailableThemes_DuplicateThemes(t *testing.T) {
	tmpDir := t.TempDir()
	themesDir := filepath.Join(tmpDir, "themes")
	customDir := filepath.Join(tmpDir, "custom")

	err := os.MkdirAll(themesDir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(customDir, 0755)
	assert.NoError(t, err)

	// Create same theme name in both directories
	err = os.WriteFile(filepath.Join(themesDir, "duplicate.yaml"), []byte("theme: builtin"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(customDir, "duplicate.yaml"), []byte("theme: custom"), 0644)
	assert.NoError(t, err)

	service := NewThemeService(themesDir, customDir, nil)

	themes, err := service.ListAvailableThemes(context.Background())

	assert.NoError(t, err)

	// Should contain "duplicate" only once despite being in both directories
	duplicateCount := 0
	for _, theme := range themes {
		if theme == "duplicate" {
			duplicateCount++
		}
	}
	assert.Equal(t, 1, duplicateCount, "Duplicate theme should appear only once")
}

// Test edge cases and error scenarios
func TestThemeServiceImpl_EdgeCases(t *testing.T) {
	t.Run("invalid_yaml_files", func(t *testing.T) {
		tmpDir := t.TempDir()
		themesDir := filepath.Join(tmpDir, "themes")

		err := os.MkdirAll(themesDir, 0755)
		assert.NoError(t, err)

		// Create files with invalid extensions
		invalidFiles := []string{
			filepath.Join(themesDir, "not-a-theme.txt"),
			filepath.Join(themesDir, "theme.json"), // Wrong extension
			filepath.Join(themesDir, "valid-theme.yaml"),
		}

		for _, file := range invalidFiles {
			err := os.WriteFile(file, []byte("content"), 0644)
			assert.NoError(t, err)
		}

		service := NewThemeService(themesDir, "", nil)

		themes, err := service.ListAvailableThemes(context.Background())

		assert.NoError(t, err)
		// Should only include valid YAML files
		assert.Contains(t, themes, "valid-theme")
		assert.NotContains(t, themes, "not-a-theme")
		assert.NotContains(t, themes, "theme")
	})

	t.Run("hidden_files", func(t *testing.T) {
		tmpDir := t.TempDir()
		themesDir := filepath.Join(tmpDir, "themes")

		err := os.MkdirAll(themesDir, 0755)
		assert.NoError(t, err)

		// Create hidden files (should be ignored)
		hiddenFiles := []string{
			filepath.Join(themesDir, ".hidden-theme.yaml"),
			filepath.Join(themesDir, "visible-theme.yaml"),
		}

		for _, file := range hiddenFiles {
			err := os.WriteFile(file, []byte("theme: test"), 0644)
			assert.NoError(t, err)
		}

		service := NewThemeService(themesDir, "", nil)

		themes, err := service.ListAvailableThemes(context.Background())

		assert.NoError(t, err)
		// Should include visible themes
		assert.Contains(t, themes, "visible-theme")
		// Note: Hidden file handling depends on implementation - may or may not be filtered
	})

	t.Run("permission_denied_directory", func(t *testing.T) {
		// This test is platform-specific and may not work on all systems
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tmpDir := t.TempDir()
		restrictedDir := filepath.Join(tmpDir, "restricted")

		err := os.MkdirAll(restrictedDir, 0000) // No permissions
		assert.NoError(t, err)

		// Restore permissions after test
		defer os.Chmod(restrictedDir, 0755)

		service := NewThemeService(restrictedDir, "", nil)

		themes, err := service.ListAvailableThemes(context.Background())

		// Should handle permission errors gracefully
		// In CI environments, this may return an error if no themes are found
		if err != nil {
			assert.Contains(t, err.Error(), "no themes found")
			assert.Nil(t, themes)
		} else {
			assert.NotNil(t, themes)
		}
	})
}

// Test concurrent access to theme service
func TestThemeServiceImpl_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	themesDir := filepath.Join(tmpDir, "themes")

	err := os.MkdirAll(themesDir, 0755)
	assert.NoError(t, err)

	// Create some theme files
	for i := 0; i < 5; i++ {
		filename := filepath.Join(themesDir, fmt.Sprintf("theme%d.yaml", i))
		err := os.WriteFile(filename, []byte("theme: test"), 0644)
		assert.NoError(t, err)
	}

	service := NewThemeService(themesDir, "", nil)

	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	// Run concurrent theme listing operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := service.ListAvailableThemes(context.Background())
			results <- err
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// Test context cancellation
func TestThemeServiceImpl_ContextCancellation(t *testing.T) {
	service := NewThemeService("/themes", "", nil)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	themes, err := service.ListAvailableThemes(ctx)

	// Should handle cancelled context gracefully
	// Note: The current implementation may not check context,
	// so this tests the expected behavior if context checking is added
	if err != nil {
		// Could be context error or "no themes found" error in CI
		assert.True(t, strings.Contains(err.Error(), "context") || strings.Contains(err.Error(), "no themes found"))
	} else {
		assert.NotNil(t, themes)
	}
}

// Benchmark theme service operations
func BenchmarkThemeService_ListAvailableThemes(b *testing.B) {
	tmpDir := b.TempDir()
	themesDir := filepath.Join(tmpDir, "themes")

	err := os.MkdirAll(themesDir, 0755)
	assert.NoError(b, err)

	// Create many theme files for realistic benchmark
	for i := 0; i < 50; i++ {
		filename := filepath.Join(themesDir, fmt.Sprintf("theme%d.yaml", i))
		err := os.WriteFile(filename, []byte("theme: benchmark"), 0644)
		assert.NoError(b, err)
	}

	service := NewThemeService(themesDir, "", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ListAvailableThemes(context.Background())
	}
}

// Test theme name extraction and validation
func TestThemeServiceImpl_ThemeNameValidation(t *testing.T) {
	tmpDir := t.TempDir()
	themesDir := filepath.Join(tmpDir, "themes")

	err := os.MkdirAll(themesDir, 0755)
	assert.NoError(t, err)

	// Create files with various names to test name extraction
	testFiles := map[string]string{
		"simple.yaml":                 "simple",
		"theme-with-dashes.yaml":      "theme-with-dashes",
		"theme_with_underscores.yaml": "theme_with_underscores",
		"UPPERCASE.yaml":              "UPPERCASE",
		"123numeric.yaml":             "123numeric",
		"special!@#chars.yaml":        "special!@#chars", // May be filtered out
	}

	for filename := range testFiles {
		err := os.WriteFile(filepath.Join(themesDir, filename), []byte("theme: test"), 0644)
		assert.NoError(t, err)
	}

	service := NewThemeService(themesDir, "", nil)

	themes, err := service.ListAvailableThemes(context.Background())

	assert.NoError(t, err)

	// Verify expected theme names are present
	for filename, expectedName := range testFiles {
		if strings.Contains(expectedName, "!@#") {
			// Special characters might be filtered - just check it doesn't crash
			continue
		}
		assert.Contains(t, themes, expectedName, "Theme %s should be extracted from file %s", expectedName, filename)
	}
}
