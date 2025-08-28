package tui

import (
	"fmt"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// Theme-aware color helper functions for the App
// These replace all hardcoded tcell.Color* constants and [color] tags

// GetStatusColor returns theme-aware status message colors using hierarchical system
// Replaces: tcell.ColorRed, tcell.ColorGreen, tcell.ColorYellow, tcell.ColorBlue
func (a *App) GetStatusColor(level string) tcell.Color {
	if a.currentTheme == nil {
		// Use default theme fallback instead of hardcoded colors
		fallbackTheme := a.getDefaultTheme()
		switch level {
		case "error":
			return fallbackTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeError).Color()
		case "success":
			return fallbackTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeSuccess).Color()
		case "warning":
			return fallbackTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeWarning).Color()
		case "info":
			return fallbackTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeInfo).Color()
		case "progress":
			return fallbackTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeInfo).Color()
		default:
			return fallbackTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeForeground).Color()
		}
	}

	switch level {
	case "error":
		return a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeError).Color()
	case "success":
		return a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeSuccess).Color()
	case "warning":
		return a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeWarning).Color()
	case "info":
		return a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeInfo).Color()
	case "progress":
		return a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeInfo).Color()
	default:
		return a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeInfo).Color()
	}
}

// GetColorTag returns theme-aware color tags for text markup using hierarchical system
// Replaces: [yellow], [blue], [green], [red], [gray], [dim], etc.
func (a *App) GetColorTag(purpose string) string {
	if a.currentTheme == nil {
		// Fallback to hardcoded color names if no theme is loaded
		switch purpose {
		case "title":
			return "[yellow]"
		case "header":
			return "[green]"
		case "emphasis":
			return "[orange]"
		case "secondary":
			return "[gray]"
		case "link":
			return "[blue]"
		case "code":
			return "[purple]"
		default:
			return "[white]"
		}
	}

	var color config.Color
	switch purpose {
	case "title":
		color = a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypePrimary)
	case "header":
		color = a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeAccent)
	case "emphasis":
		color = a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeWarning)
	case "secondary":
		color = a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeSecondary)
	case "link":
		color = a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeAccent)
	case "code":
		color = a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeAccent)
	default:
		color = a.currentTheme.GetComponentColor(config.ComponentTypeGeneral, config.ColorTypeForeground)
	}

	return fmt.Sprintf("[%s]", color.String())
}

// GetEndTag returns the closing color tag
func (a *App) GetEndTag() string {
	return "[-]"
}

// GetComponentColors returns theme-aware colors for specific UI components using the new hierarchical system
// Replaces: hardcoded component-specific colors
func (a *App) GetComponentColors(component string) config.ComponentColorSet {
	if a.currentTheme == nil {
		// Fallback to generic colors if no theme is loaded
		return config.ComponentColorSet{
			Border:     config.NewColor("#44475a"),
			Title:      config.NewColor("#f1fa8c"),
			Background: config.NewColor("#282a36"),
			Text:       config.NewColor("#f8f8f2"),
			Accent:     config.NewColor("#8be9fd"),
		}
	}

	var componentType config.ComponentType
	switch component {
	case "ai":
		componentType = config.ComponentTypeAI
	case "slack":
		componentType = config.ComponentTypeSlack
	case "obsidian":
		componentType = config.ComponentTypeObsidian
	case "links":
		componentType = config.ComponentTypeLinks
	case "stats":
		componentType = config.ComponentTypeStats
	case "prompts":
		componentType = config.ComponentTypePrompts
	case "search":
		componentType = config.ComponentTypeSearch
	case "attachments":
		componentType = config.ComponentTypeAttachments
	case "saved_queries":
		componentType = config.ComponentTypeSavedQueries
	case "labels":
		componentType = config.ComponentTypeLabels
	case "themes":
		componentType = config.ComponentTypeThemes
	default:
		componentType = config.ComponentTypeGeneral
	}

	// Use hierarchical color resolution for each component color type
	bgColor := a.currentTheme.GetComponentColor(componentType, config.ColorTypeBackground)
	titleColor := a.currentTheme.GetComponentColor(componentType, config.ColorTypePrimary)
	
	return config.ComponentColorSet{
		Border:     a.currentTheme.GetComponentColor(componentType, config.ColorTypeBorder),
		Title:      titleColor,
		Background: bgColor,
		Text:       a.currentTheme.GetComponentColor(componentType, config.ColorTypeForeground),
		Accent:     a.currentTheme.GetComponentColor(componentType, config.ColorTypeAccent),
	}
}

// Formatted text helpers - replace hardcoded color tags with theme-aware ones

// FormatTitle formats text with title color
// Replaces: "[yellow]text[white]" or "[yellow]text[-]"
func (a *App) FormatTitle(text string) string {
	return a.GetColorTag("title") + text + a.GetEndTag()
}

// FormatHeader formats text with header color
// Replaces: "[green]text[-]"
func (a *App) FormatHeader(text string) string {
	return a.GetColorTag("header") + text + a.GetEndTag()
}

// FormatEmphasis formats text with emphasis color
// Replaces: "[orange]text[-]" or "[bold]text[-]"
func (a *App) FormatEmphasis(text string) string {
	return a.GetColorTag("emphasis") + text + a.GetEndTag()
}

// FormatSecondary formats text with secondary/dimmed color
// Replaces: "[dim]text[-]" or "[gray]text[-]"
func (a *App) FormatSecondary(text string) string {
	return a.GetColorTag("secondary") + text + a.GetEndTag()
}

// FormatLink formats text with link color
// Replaces: "[blue]text[-]"
func (a *App) FormatLink(text string) string {
	return a.GetColorTag("link") + text + a.GetEndTag()
}

// FormatCode formats text with code color
// Replaces: "[purple]text[-]"
func (a *App) FormatCode(text string) string {
	return a.GetColorTag("code") + text + a.GetEndTag()
}

// Legacy compatibility helpers - maintain existing App methods but make them theme-aware
// Note: These are alternatives to existing methods that will replace them

// getThemeTitleColor returns theme-aware title color (new implementation)
func (a *App) getThemeTitleColor() tcell.Color {
	if a.currentTheme == nil {
		// Use hierarchical theme system instead of hardcoded color
		return a.getComponentColor(config.ComponentTypeGeneral, config.ColorTypePrimary)
	}
	return a.currentTheme.UI.TitleColor.Color()
}

// getThemeFooterColor returns theme-aware footer color (new implementation)
func (a *App) getThemeFooterColor() tcell.Color {
	if a.currentTheme == nil {
		// Use hierarchical theme system instead of hardcoded color
		return a.getComponentColor(config.ComponentTypeGeneral, config.ColorTypeSecondary)
	}
	return a.currentTheme.UI.FooterColor.Color()
}

// getThemeHintColor returns theme-aware hint color (new implementation)
func (a *App) getThemeHintColor() tcell.Color {
	if a.currentTheme == nil {
		// Use hierarchical theme system instead of hardcoded color
		return a.getComponentColor(config.ComponentTypeGeneral, config.ColorTypeSecondary)
	}
	return a.currentTheme.UI.HintColor.Color()
}

// Additional helper for backward compatibility with existing error handler integration
func (a *App) getStatusColorCompat(level string) tcell.Color {
	return a.GetStatusColor(level)
}

// GetInputFieldColors returns theme-aware colors for input fields using hierarchical system
func (a *App) GetInputFieldColors() (bgColor, textColor tcell.Color) {
	if a.currentTheme == nil {
		// Use hierarchical default colors as fallback
		generalColors := a.GetComponentColors("general")
		return generalColors.Background.Color(), generalColors.Text.Color()
	}
	
	// Use new hierarchical system with fallback to legacy
	bg, fg, _ := a.currentTheme.GetInputColors()
	if bg == "" || fg == "" {
		// Legacy fallback
		return a.currentTheme.Body.BgColor.Color(), a.currentTheme.Body.FgColor.Color()
	}
	return bg.Color(), fg.Color()
}

// ConfigureInputFieldTheme applies consistent theme colors to input fields using hierarchical system
func (a *App) ConfigureInputFieldTheme(field *tview.InputField, component string) *tview.InputField {
	// Use hierarchical theme system for consistent theming
	var componentColors config.ComponentColorSet
	switch component {
	case "advanced", "overlay":
		componentColors = a.GetComponentColors("search")
	default:
		componentColors = a.GetComponentColors("general")
	}

	// Configure basic field colors
	field.SetFieldBackgroundColor(componentColors.Background.Color()).
		SetFieldTextColor(componentColors.Text.Color()).
		SetLabelColor(componentColors.Title.Color()).
		SetPlaceholderTextColor(a.getHintColor())

	// For advanced search, apply more aggressive styling to override form defaults
	if component == "advanced" {
		// Force the background color and remove any borders that might cause color issues
		field.SetBackgroundColor(componentColors.Background.Color()).
			SetBorder(false).
			SetBorderColor(componentColors.Background.Color())
	}

	return field
}

// GetSearchFieldColors returns component-specific colors for search fields using hierarchical system
func (a *App) GetSearchFieldColors(component string) (bgColor, textColor, labelColor tcell.Color) {
	if a.currentTheme == nil {
		// Use hierarchical default colors as fallback
		generalColors := a.GetComponentColors("general")
		return generalColors.Background.Color(), generalColors.Text.Color(), generalColors.Accent.Color()
	}

	// Use hierarchical system for all search components
	bg, fg, label := a.currentTheme.GetInputColors()
	if bg == "" || fg == "" || label == "" {
		// Legacy fallback
		return a.currentTheme.Body.BgColor.Color(), a.currentTheme.Body.FgColor.Color(), a.currentTheme.UI.TitleColor.Color()
	}
	return bg.Color(), fg.Color(), label.Color()
}
