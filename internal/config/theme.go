package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ThemeLoader handles loading and applying themes
type ThemeLoader struct {
	themesDir string
}

// NewThemeLoader creates a new theme loader
func NewThemeLoader(themesDir string) *ThemeLoader {
	return &ThemeLoader{
		themesDir: themesDir,
	}
}

// LoadThemeFromFile loads a theme from a YAML file
func (tl *ThemeLoader) LoadThemeFromFile(filename string) (*ColorsConfig, error) {
	// Try to load from themes directory first
	filepath := filepath.Join(tl.themesDir, filename)
	if !fileExists(filepath) {
		// Try absolute path
		filepath = filename
		if !fileExists(filepath) {
			return nil, fmt.Errorf("theme file not found: %s", filename)
		}
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}

	var theme struct {
		GmailTUI *ColorsConfig `yaml:"gmailTUI"`
	}

	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse theme file: %w", err)
	}

	if theme.GmailTUI == nil {
		return nil, fmt.Errorf("invalid theme file: missing gmailTUI section")
	}

	return theme.GmailTUI, nil
}

// ListAvailableThemes returns a list of available theme files
func (tl *ThemeLoader) ListAvailableThemes() ([]string, error) {
	var themes []string

	entries, err := os.ReadDir(tl.themesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read themes directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			themes = append(themes, entry.Name())
		}
	}

	return themes, nil
}

// SaveThemeToFile saves a theme configuration to a YAML file
func (tl *ThemeLoader) SaveThemeToFile(theme *ColorsConfig, filename string) error {
	// Ensure themes directory exists
	if err := os.MkdirAll(tl.themesDir, 0755); err != nil {
		return fmt.Errorf("failed to create themes directory: %w", err)
	}

	filepath := filepath.Join(tl.themesDir, filename)

	// Create theme structure
	themeData := struct {
		GmailTUI *ColorsConfig `yaml:"gmailTUI"`
	}{
		GmailTUI: theme,
	}

	data, err := yaml.Marshal(themeData)
	if err != nil {
		return fmt.Errorf("failed to marshal theme: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write theme file: %w", err)
	}

	return nil
}

// ValidateTheme validates a theme configuration
func (tl *ThemeLoader) ValidateTheme(theme *ColorsConfig) error {
	if theme == nil {
		return fmt.Errorf("theme is nil")
	}

	// Validate required colors
	requiredColors := []struct {
		name  string
		color Color
	}{
		{"Body.FgColor", theme.Body.FgColor},
		{"Body.BgColor", theme.Body.BgColor},
		{"Email.UnreadColor", theme.Email.UnreadColor},
		{"Email.ReadColor", theme.Email.ReadColor},
	}

	for _, req := range requiredColors {
		if req.color == "" {
			return fmt.Errorf("missing required color: %s", req.name)
		}
	}

	return nil
}

// CreateDefaultTheme creates a default theme if none exists
func (tl *ThemeLoader) CreateDefaultTheme() error {
	// Check if default theme already exists
	defaultThemePath := filepath.Join(tl.themesDir, "gmail-dark.yaml")
	if fileExists(defaultThemePath) {
		return nil // Theme already exists
	}

	// Create default theme
	defaultTheme := DefaultColors()
	return tl.SaveThemeToFile(defaultTheme, "gmail-dark.yaml")
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ApplyThemeToApp applies a theme to the application
func ApplyThemeToApp(theme *ColorsConfig, app interface{}) error {
	// This is a placeholder for applying theme to the app
	// In a real implementation, you would update the app's colors
	fmt.Printf("Applying theme with %d color configurations\n", len(theme.Email.UnreadColor))
	return nil
}

// GetThemePreview generates a preview of the theme colors
func GetThemePreview(theme *ColorsConfig) string {
	preview := "🎨 Theme Preview:\n\n"

	preview += fmt.Sprintf("📧 Email Colors:\n")
	preview += fmt.Sprintf("  • Unread: %s\n", theme.Email.UnreadColor)
	preview += fmt.Sprintf("  • Read: %s\n", theme.Email.ReadColor)
	preview += fmt.Sprintf("  • Important: %s\n", theme.Email.ImportantColor)
	preview += fmt.Sprintf("  • Sent: %s\n", theme.Email.SentColor)
	preview += fmt.Sprintf("  • Draft: %s\n", theme.Email.DraftColor)

	preview += fmt.Sprintf("\n🎨 UI Colors:\n")
	preview += fmt.Sprintf("  • Text: %s\n", theme.Body.FgColor)
	preview += fmt.Sprintf("  • Background: %s\n", theme.Body.BgColor)
	preview += fmt.Sprintf("  • Border: %s\n", theme.Frame.Border.FgColor)
	preview += fmt.Sprintf("  • Focus: %s\n", theme.Frame.Border.FocusColor)

	return preview
}
