package desktop

import (
	"context"

	"github.com/ajramos/giztui/internal/services"
)

// ThemesEnabled reports whether theming is available.
func (a *API) ThemesEnabled() bool { return a.theme != nil }

// ListThemes returns the available theme names.
func (a *API) ListThemes(ctx context.Context) ([]string, error) {
	if a.theme == nil {
		return []string{}, nil
	}
	return a.theme.ListAvailableThemes(ctx)
}

// CurrentThemeName returns the name of the active theme.
func (a *API) CurrentThemeName(ctx context.Context) (string, error) {
	if a.theme == nil {
		return "", nil
	}
	return a.theme.GetCurrentTheme(ctx)
}

// GetThemeColors returns the flattened palette for a theme (empty name = the
// current theme), or nil when theming is unavailable.
func (a *API) GetThemeColors(ctx context.Context, name string) (*ThemeColors, error) {
	if a.theme == nil {
		return nil, nil
	}
	if name == "" {
		if cur, err := a.theme.GetCurrentTheme(ctx); err == nil {
			name = cur
		}
	}
	tc, err := a.theme.GetThemeConfig(ctx, name)
	if err != nil {
		return nil, err
	}
	return themeColorsFrom(name, tc), nil
}

func themeColorsFrom(name string, tc *services.ThemeConfig) *ThemeColors {
	return &ThemeColors{
		Name:        name,
		Bg:          tc.UIColors.BgColor,
		Fg:          tc.UIColors.FgColor,
		Border:      tc.UIColors.BorderColor,
		Accent:      tc.UIColors.FocusColor,
		Primary:     tc.UIColors.TitleColor,
		Danger:      tc.UIColors.ErrorColor,
		Warning:     tc.UIColors.WarningColor,
		Success:     tc.UIColors.SuccessColor,
		SelectionBg: tc.UIColors.SelectionBgColor,
		InputBg:     tc.UIColors.InputBgColor,
		Unread:      tc.EmailColors.UnreadColor,
		Muted:       tc.UIColors.FooterColor,
	}
}
