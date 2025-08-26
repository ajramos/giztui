package tui

import (
	"fmt"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// Theme-aware color helper functions for the App
// These replace all hardcoded tcell.Color* constants and [color] tags

// GetStatusColor returns theme-aware status message colors
// Replaces: tcell.ColorRed, tcell.ColorGreen, tcell.ColorYellow, tcell.ColorBlue
func (a *App) GetStatusColor(level string) tcell.Color {
	if a.currentTheme == nil {
		// Fallback to hardcoded colors if no theme is loaded
		switch level {
		case "error":
			return tcell.ColorRed
		case "success":
			return tcell.ColorGreen
		case "warning":
			return tcell.ColorYellow
		case "info":
			return tcell.ColorBlue
		case "progress":
			return tcell.ColorOrange
		default:
			return tcell.ColorWhite
		}
	}

	switch level {
	case "error":
		return a.currentTheme.Status.Error.Color()
	case "success":
		return a.currentTheme.Status.Success.Color()
	case "warning":
		return a.currentTheme.Status.Warning.Color()
	case "info":
		return a.currentTheme.Status.Info.Color()
	case "progress":
		return a.currentTheme.Status.Progress.Color()
	default:
		return a.currentTheme.UI.InfoColor.Color()
	}
}

// GetColorTag returns theme-aware color tags for text markup
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
		color = a.currentTheme.Tags.Title
	case "header":
		color = a.currentTheme.Tags.Header
	case "emphasis":
		color = a.currentTheme.Tags.Emphasis
	case "secondary":
		color = a.currentTheme.Tags.Secondary
	case "link":
		color = a.currentTheme.Tags.Link
	case "code":
		color = a.currentTheme.Tags.Code
	default:
		color = a.currentTheme.Body.FgColor
	}

	return fmt.Sprintf("[%s]", color.String())
}

// GetEndTag returns the closing color tag
func (a *App) GetEndTag() string {
	return "[-]"
}

// GetComponentColors returns theme-aware colors for specific UI components
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

	switch component {
	case "ai":
		return a.currentTheme.Components.AI
	case "slack":
		return a.currentTheme.Components.Slack
	case "obsidian":
		return a.currentTheme.Components.Obsidian
	case "links":
		return a.currentTheme.Components.Links
	case "stats":
		return a.currentTheme.Components.Stats
	case "prompts":
		return a.currentTheme.Components.Prompts
	case "labels": // For label picker
		return config.ComponentColorSet{
			Border:     a.currentTheme.Frame.Border.FgColor,
			Title:      a.currentTheme.UI.TitleColor,
			Background: a.currentTheme.Body.BgColor,
			Text:       a.currentTheme.Body.FgColor,
			Accent:     a.currentTheme.UI.LabelColor, // Use theme's label color
		}
	case "themes": // For theme picker itself
		return config.ComponentColorSet{
			Border:     a.currentTheme.Frame.Border.FgColor,
			Title:      a.currentTheme.UI.TitleColor,
			Background: a.currentTheme.Body.BgColor,
			Text:       a.currentTheme.Body.FgColor,
			Accent:     a.currentTheme.UI.InfoColor,
		}
	default:
		// Return generic UI colors as fallback
		return config.ComponentColorSet{
			Border:     a.currentTheme.Frame.Border.FgColor,
			Title:      a.currentTheme.UI.TitleColor,
			Background: a.currentTheme.Body.BgColor,
			Text:       a.currentTheme.Body.FgColor,
			Accent:     a.currentTheme.UI.InfoColor,
		}
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
		return tcell.ColorYellow // Fallback
	}
	return a.currentTheme.UI.TitleColor.Color()
}

// getThemeFooterColor returns theme-aware footer color (new implementation)
func (a *App) getThemeFooterColor() tcell.Color {
	if a.currentTheme == nil {
		return tcell.ColorGray // Fallback
	}
	return a.currentTheme.UI.FooterColor.Color()
}

// getThemeHintColor returns theme-aware hint color (new implementation)
func (a *App) getThemeHintColor() tcell.Color {
	if a.currentTheme == nil {
		return tcell.ColorGray // Fallback
	}
	return a.currentTheme.UI.HintColor.Color()
}

// Additional helper for backward compatibility with existing error handler integration
func (a *App) getStatusColorCompat(level string) tcell.Color {
	return a.GetStatusColor(level)
}

// GetInputFieldColors returns theme-aware colors for input fields
func (a *App) GetInputFieldColors() (bgColor, textColor tcell.Color) {
	if a.currentTheme == nil {
		return tview.Styles.PrimitiveBackgroundColor, tview.Styles.PrimaryTextColor
	}
	return a.currentTheme.Body.BgColor.Color(), a.currentTheme.Body.FgColor.Color()
}

// ConfigureInputFieldTheme applies consistent theme colors to input fields
func (a *App) ConfigureInputFieldTheme(field *tview.InputField, component string) *tview.InputField {
	bgColor, textColor := a.GetInputFieldColors()

	// Configure basic field colors
	field.SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(textColor).
		SetLabelColor(a.getTitleColor()).
		SetPlaceholderTextColor(a.getHintColor())

	// For advanced search, apply more aggressive styling to override form defaults
	if component == "advanced" {
		// Force the background color and remove any borders that might cause color issues
		field.SetBackgroundColor(bgColor).
			SetBorder(false).
			SetBorderColor(bgColor)
	}

	return field
}

// GetSearchFieldColors returns component-specific colors for search fields
func (a *App) GetSearchFieldColors(component string) (bgColor, textColor, labelColor tcell.Color) {
	if a.currentTheme == nil {
		return tview.Styles.PrimitiveBackgroundColor, tview.Styles.PrimaryTextColor, tcell.ColorYellow
	}

	switch component {
	case "advanced":
		// Advanced search uses slightly different styling
		return a.currentTheme.Body.BgColor.Color(), a.currentTheme.Body.FgColor.Color(), a.currentTheme.UI.TitleColor.Color()
	case "simple", "overlay":
		// Simple search and overlay search
		return a.currentTheme.Body.BgColor.Color(), a.currentTheme.Body.FgColor.Color(), a.currentTheme.UI.TitleColor.Color()
	default:
		return a.currentTheme.Body.BgColor.Color(), a.currentTheme.Body.FgColor.Color(), a.currentTheme.UI.TitleColor.Color()
	}
}
