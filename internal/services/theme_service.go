package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/gmail-tui/internal/config"
)

// ThemeServiceImpl implements ThemeService
type ThemeServiceImpl struct {
	currentTheme  string
	themesDir     string
	customThemeDir string
	themeLoader   *config.ThemeLoader
	applyThemeFunc func(*config.ColorsConfig) error // Function to apply theme to the app
}

// NewThemeService creates a new theme service
func NewThemeService(themesDir string, customThemeDir string, applyThemeFunc func(*config.ColorsConfig) error) *ThemeServiceImpl {
	return &ThemeServiceImpl{
		currentTheme:   "gmail-dark", // Default theme
		themesDir:      themesDir,
		customThemeDir: customThemeDir,
		themeLoader:    config.NewThemeLoader(themesDir),
		applyThemeFunc: applyThemeFunc,
	}
}

// ListAvailableThemes returns all available theme names from both directories
func (s *ThemeServiceImpl) ListAvailableThemes(ctx context.Context) ([]string, error) {
	themeMap := make(map[string]bool) // Use map to avoid duplicates
	var themes []string

	// 1. Get themes from built-in themes directory
	builtinThemes, err := s.getThemesFromDirectory(s.themesDir)
	if err == nil {
		for _, theme := range builtinThemes {
			if !themeMap[theme] {
				themes = append(themes, theme)
				themeMap[theme] = true
			}
		}
	}

	// 2. Get themes from custom themes directory (if specified)
	if s.customThemeDir != "" {
		customThemes, err := s.getThemesFromDirectory(s.customThemeDir)
		if err == nil {
			for _, theme := range customThemes {
				if !themeMap[theme] {
					themes = append(themes, theme)
					themeMap[theme] = true
				}
			}
		}
	}

	// 3. Get themes from user config directory
	userConfigDir, err := s.getUserConfigThemesDir()
	if err == nil {
		userThemes, err := s.getThemesFromDirectory(userConfigDir)
		if err == nil {
			for _, theme := range userThemes {
				if !themeMap[theme] {
					themes = append(themes, theme)
					themeMap[theme] = true
				}
			}
		}
	}

	if len(themes) == 0 {
		return nil, fmt.Errorf("no themes found in any theme directories")
	}

	return themes, nil
}

// GetCurrentTheme returns the name of the currently active theme
func (s *ThemeServiceImpl) GetCurrentTheme(ctx context.Context) (string, error) {
	return s.currentTheme, nil
}

// ApplyTheme applies the specified theme
func (s *ThemeServiceImpl) ApplyTheme(ctx context.Context, name string) error {
	// Load theme configuration
	themeConfig, err := s.loadThemeByName(name)
	if err != nil {
		return fmt.Errorf("failed to load theme '%s': %w", name, err)
	}

	// Apply theme using the provided function
	if s.applyThemeFunc != nil {
		if err := s.applyThemeFunc(themeConfig); err != nil {
			return fmt.Errorf("failed to apply theme '%s': %w", name, err)
		}
	}

	// Update current theme
	s.currentTheme = name
	return nil
}

// PreviewTheme returns theme configuration for preview without applying it
func (s *ThemeServiceImpl) PreviewTheme(ctx context.Context, name string) (*ThemeConfig, error) {
	return s.GetThemeConfig(ctx, name)
}

// GetThemeConfig returns theme configuration for display
func (s *ThemeServiceImpl) GetThemeConfig(ctx context.Context, name string) (*ThemeConfig, error) {
	// Load theme configuration
	colorsConfig, err := s.loadThemeByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load theme '%s': %w", name, err)
	}

	// Convert to ThemeConfig for display
	themeConfig := &ThemeConfig{
		Name:        name,
		Description: s.getThemeDescription(name),
	}

	// Email colors
	themeConfig.EmailColors.UnreadColor = colorsConfig.Email.UnreadColor.String()
	themeConfig.EmailColors.ReadColor = colorsConfig.Email.ReadColor.String()
	themeConfig.EmailColors.ImportantColor = colorsConfig.Email.ImportantColor.String()
	themeConfig.EmailColors.SentColor = colorsConfig.Email.SentColor.String()
	themeConfig.EmailColors.DraftColor = colorsConfig.Email.DraftColor.String()

	// Basic UI colors
	themeConfig.UIColors.FgColor = colorsConfig.Body.FgColor.String()
	themeConfig.UIColors.BgColor = colorsConfig.Body.BgColor.String()
	themeConfig.UIColors.BorderColor = colorsConfig.Frame.Border.FgColor.String()
	themeConfig.UIColors.FocusColor = colorsConfig.Frame.Border.FocusColor.String()
	
	// Component colors (previously hardcoded)
	themeConfig.UIColors.TitleColor = colorsConfig.UI.TitleColor.String()
	themeConfig.UIColors.FooterColor = colorsConfig.UI.FooterColor.String()
	themeConfig.UIColors.HintColor = colorsConfig.UI.HintColor.String()
	
	// Selection colors
	themeConfig.UIColors.SelectionBgColor = colorsConfig.UI.SelectionBgColor.String()
	themeConfig.UIColors.SelectionFgColor = colorsConfig.UI.SelectionFgColor.String()
	
	// Status colors
	themeConfig.UIColors.ErrorColor = colorsConfig.UI.ErrorColor.String()
	themeConfig.UIColors.SuccessColor = colorsConfig.UI.SuccessColor.String()
	themeConfig.UIColors.WarningColor = colorsConfig.UI.WarningColor.String()
	themeConfig.UIColors.InfoColor = colorsConfig.UI.InfoColor.String()
	
	// Input colors
	themeConfig.UIColors.InputBgColor = colorsConfig.UI.InputBgColor.String()
	themeConfig.UIColors.InputFgColor = colorsConfig.UI.InputFgColor.String()
	themeConfig.UIColors.LabelColor = colorsConfig.UI.LabelColor.String()

	return themeConfig, nil
}

// ValidateTheme checks if a theme is valid and can be loaded
func (s *ThemeServiceImpl) ValidateTheme(ctx context.Context, name string) error {
	_, err := s.loadThemeByName(name)
	return err
}

// Helper methods

// getThemesFromDirectory reads theme files from a directory
func (s *ThemeServiceImpl) getThemesFromDirectory(dir string) ([]string, error) {
	var themes []string

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return themes, nil // Return empty list if directory doesn't exist
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read themes directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			// Remove .yaml extension to get theme name
			themeName := strings.TrimSuffix(entry.Name(), ".yaml")
			themes = append(themes, themeName)
		}
	}

	return themes, nil
}

// loadThemeByName loads a theme configuration by name, checking all directories
func (s *ThemeServiceImpl) loadThemeByName(name string) (*config.ColorsConfig, error) {
	fileName := name + ".yaml"

	// Priority order: custom dir, user config dir, built-in dir
	dirs := []string{}
	
	if s.customThemeDir != "" {
		dirs = append(dirs, s.customThemeDir)
	}
	
	if userConfigDir, err := s.getUserConfigThemesDir(); err == nil {
		dirs = append(dirs, userConfigDir)
	}
	
	dirs = append(dirs, s.themesDir)

	// Try each directory in priority order
	for _, dir := range dirs {
		themePath := filepath.Join(dir, fileName)
		if _, err := os.Stat(themePath); err == nil {
			// Load theme from this directory
			loader := config.NewThemeLoader(dir)
			return loader.LoadThemeFromFile(fileName)
		}
	}

	return nil, fmt.Errorf("theme '%s' not found in any theme directory", name)
}

// getUserConfigThemesDir returns the user configuration themes directory
func (s *ThemeServiceImpl) getUserConfigThemesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "giztui", "themes"), nil
}

// getThemeDescription returns a description for known themes
func (s *ThemeServiceImpl) getThemeDescription(name string) string {
	descriptions := map[string]string{
		"gmail-dark":     "Dracula-based dark theme",
		"gmail-light":    "Clean light theme",
		"custom-example": "Demo custom theme",
		"high-contrast":  "High contrast theme for accessibility",
	}
	
	if desc, exists := descriptions[name]; exists {
		return desc
	}
	return "Custom theme"
}