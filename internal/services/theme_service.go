package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/giztui/internal/config"
)

// ThemeUpdateCallback represents a function that gets called when theme changes
type ThemeUpdateCallback func(*config.ColorsConfig) error

// ComponentRegistration represents a component that can receive theme updates
type ComponentRegistration struct {
	name     string
	callback ThemeUpdateCallback
}

// ThemeServiceImpl implements ThemeService
type ThemeServiceImpl struct {
	currentTheme   string
	themesDir      string
	customThemeDir string
	themeLoader    *config.ThemeLoader
	applyThemeFunc func(*config.ColorsConfig) error // Function to apply theme to the app

	// Component registration system
	registeredComponents []ComponentRegistration
	currentThemeConfig   *config.ColorsConfig // Cache current theme for new registrations
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

	// Cache the theme configuration
	s.currentThemeConfig = themeConfig

	// Notify all registered components
	if err := s.notifyComponents(themeConfig); err != nil {
		return fmt.Errorf("failed to notify components of theme change: %w", err)
	}

	// Update current theme
	s.currentTheme = name
	return nil
}

// RegisterComponent registers a component to receive theme updates
func (s *ThemeServiceImpl) RegisterComponent(name string, callback ThemeUpdateCallback) error {
	// Add to registered components
	s.registeredComponents = append(s.registeredComponents, ComponentRegistration{
		name:     name,
		callback: callback,
	})

	// If we have a current theme, apply it to the new component immediately
	if s.currentThemeConfig != nil {
		if err := callback(s.currentThemeConfig); err != nil {
			return fmt.Errorf("failed to apply current theme to component '%s': %w", name, err)
		}
	}

	return nil
}

// UnregisterComponent removes a component from theme updates
func (s *ThemeServiceImpl) UnregisterComponent(name string) {
	for i, component := range s.registeredComponents {
		if component.name == name {
			// Remove component from slice
			s.registeredComponents = append(s.registeredComponents[:i], s.registeredComponents[i+1:]...)
			break
		}
	}
}

// GetCurrentThemeConfig returns the currently loaded theme configuration
func (s *ThemeServiceImpl) GetCurrentThemeConfig() *config.ColorsConfig {
	return s.currentThemeConfig
}

// notifyComponents sends theme updates to all registered components
func (s *ThemeServiceImpl) notifyComponents(themeConfig *config.ColorsConfig) error {
	var errors []string

	for _, component := range s.registeredComponents {
		if err := component.callback(themeConfig); err != nil {
			errors = append(errors, fmt.Sprintf("component '%s': %v", component.name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("theme update errors: %s", strings.Join(errors, "; "))
	}

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

	// Email colors - mapped to v2.0 hierarchical theme system
	themeConfig.EmailColors.UnreadColor = colorsConfig.Semantic.Accent.String()     // Cyan/blue for attention
	themeConfig.EmailColors.ReadColor = colorsConfig.Foundation.Foreground.String() // Default text color
	themeConfig.EmailColors.ImportantColor = colorsConfig.Semantic.Warning.String() // Orange/yellow for importance
	themeConfig.EmailColors.SentColor = colorsConfig.Semantic.Success.String()      // Green for sent items
	themeConfig.EmailColors.DraftColor = colorsConfig.Semantic.Secondary.String()   // Gray for drafts

	// Basic UI colors - mapped to v2.0 hierarchical theme system
	themeConfig.UIColors.FgColor = colorsConfig.Foundation.Foreground.String()
	themeConfig.UIColors.BgColor = colorsConfig.Foundation.Background.String()
	themeConfig.UIColors.BorderColor = colorsConfig.Foundation.Border.String()
	themeConfig.UIColors.FocusColor = colorsConfig.Foundation.Focus.String()

	// Component colors - mapped to v2.0 hierarchical theme system
	themeConfig.UIColors.TitleColor = colorsConfig.Semantic.Primary.String()
	themeConfig.UIColors.FooterColor = colorsConfig.Semantic.Secondary.String()
	themeConfig.UIColors.HintColor = colorsConfig.Semantic.Info.String()

	// Selection colors - mapped to interaction layer
	themeConfig.UIColors.SelectionBgColor = colorsConfig.Interaction.Selection.Cursor.Bg.String()
	themeConfig.UIColors.SelectionFgColor = colorsConfig.Interaction.Selection.Cursor.Fg.String()

	// Status colors - mapped to semantic layer
	themeConfig.UIColors.ErrorColor = colorsConfig.Semantic.Error.String()
	themeConfig.UIColors.SuccessColor = colorsConfig.Semantic.Success.String()
	themeConfig.UIColors.WarningColor = colorsConfig.Semantic.Warning.String()
	themeConfig.UIColors.InfoColor = colorsConfig.Semantic.Info.String()

	// Input colors - mapped to interaction layer
	themeConfig.UIColors.InputBgColor = colorsConfig.Interaction.Input.Bg.String()
	themeConfig.UIColors.InputFgColor = colorsConfig.Interaction.Input.Fg.String()
	themeConfig.UIColors.LabelColor = colorsConfig.Interaction.Input.Label.String()

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
