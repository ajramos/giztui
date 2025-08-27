# Gmail TUI Theming Guide

A comprehensive guide for creating themes and implementing theme-compliant components in Gmail TUI.

## 📖 Overview

Gmail TUI uses a hierarchical theme system that provides consistent, maintainable, and flexible color management. This guide covers everything from creating custom themes to implementing theme-compliant UI components.

## 🎨 Theme Architecture

### Hierarchical Structure

The theme system follows a four-layer hierarchy for color resolution:

1. **Component Overrides** - Highest priority, component-specific colors
2. **Semantic Colors** - Meaning-based colors (primary, accent, etc.)  
3. **Foundation Colors** - Base colors for all components
4. **Legacy Fallback** - Backward compatibility with v1.0 themes

### Color Resolution Flow

```
Component Override → Semantic → Foundation → Legacy → Default
```

When a component requests a color:
1. Check if component has a specific override
2. Fall back to semantic color (e.g., "primary")
3. Fall back to foundation color (e.g., "foreground")
4. Fall back to legacy structure if available
5. Use system default as last resort

## 📝 Creating Custom Themes

### Theme File Structure

Create a new YAML file in the `themes/` directory:

```yaml
# themes/my-custom-theme.yaml
name: "My Custom Theme"
description: "A beautiful custom theme"
version: "2.0"

gmailTUI:
  # Foundation - Base colors for all components
  foundation:
    background: "#1a1a1a"    # Primary background
    foreground: "#e0e0e0"    # Primary text
    border: "#404040"        # Default borders
    focus: "#0080ff"         # Focus highlights

  # Semantic - Meaning-based colors
  semantic:
    primary: "#00d4aa"       # Titles, main actions
    secondary: "#888888"     # Supporting elements
    accent: "#ff6b35"        # Links, highlights
    success: "#00d4aa"       # Success states
    warning: "#ffa500"       # Warning states
    error: "#ff4757"         # Error states
    info: "#0080ff"          # Info states

  # Interaction - User interaction states
  interaction:
    selection:
      cursor:
        bg: "#333333"        # Single item selection
        fg: "#ffffff"        # Selection text
      bulk:
        bg: "#2a2a2a"        # Multi-item selection
        fg: "#ffffff"        # Bulk selection text
    input:
      bg: "#2d2d2d"          # Input background
      fg: "#ffffff"          # Input text
      label: "#00d4aa"       # Input labels
    statusBar:
      bg: "#1a1a1a"          # Status bar background
      fg: "#e0e0e0"          # Status bar text

  # Component Overrides (optional)
  overrides:
    ai:
      primary: "#9c27b0"     # Purple for AI components
      accent: "#e1bee7"      # Light purple accent
    search:
      primary: "#2196f3"     # Blue for search components
      background: "#1e1e1e"  # Darker search background
```

### Supported Color Formats

- **Hex**: `#ff5555`, `#f55` (short form)
- **RGB**: `rgb(255, 85, 85)`
- **HSL**: `hsl(0, 100%, 67%)`
- **Named**: `red`, `blue`, `green`, `cyan`, `magenta`, `yellow`, `white`, `black`
- **ANSI**: `1` (red), `2` (green), `3` (yellow), etc.
- **Default**: `default` (uses terminal default)

### Theme Validation

Themes are automatically validated on load:

```go
// Valid theme structure
✅ All required sections present
✅ Color values parseable  
✅ No circular references
✅ Legacy compatibility maintained

// Invalid theme structure
❌ Missing foundation.background
❌ Invalid color format "#gggggg"
❌ Conflicting color definitions
```

## 🛠️ Implementing Theme-Compliant Components

### Best Practices

#### 1. Always Use Component Colors

```go
// ✅ CORRECT - Use hierarchical system
searchColors := app.GetComponentColors("search")
container.SetBackgroundColor(searchColors.Background.Color())
container.SetTitleColor(searchColors.Title.Color())

// ❌ WRONG - Hardcoded colors
container.SetBackgroundColor(tcell.ColorBlue)
container.SetTitleColor(tcell.ColorYellow)
```

#### 2. Register Your Component

```go
// Register component for automatic theme updates
app.themeService.RegisterComponent("mycomponent", config.ComponentTypeGeneral)
```

#### 3. Use Semantic Color Types

```go
// Map UI elements to semantic meanings
border := componentColors.Border    // Uses semantic.primary or foundation.border
title := componentColors.Title      // Uses semantic.primary  
text := componentColors.Text        // Uses foundation.foreground
accent := componentColors.Accent    // Uses semantic.accent
bg := componentColors.Background    // Uses foundation.background
```

### Component Implementation Template

```go
package tui

import (
    "github.com/ajramos/gmail-tui/internal/config"
    "github.com/rivo/tview"
)

func (a *App) createMyComponent() *tview.Flex {
    // Get component colors using hierarchical system
    colors := a.GetComponentColors("mycomponent")
    
    // Create container
    container := tview.NewFlex().SetDirection(tview.FlexRow)
    container.SetBorder(true).
        SetTitle("My Component").
        SetTitleColor(colors.Title.Color()).
        SetBackgroundColor(colors.Background.Color()).
        SetBorderColor(colors.Border.Color())
    
    // Create input field
    input := tview.NewInputField().
        SetLabel("Input: ").
        SetFieldBackgroundColor(colors.Background.Color()).
        SetFieldTextColor(colors.Text.Color()).
        SetLabelColor(colors.Title.Color()).
        SetPlaceholderTextColor(a.getHintColor())
    
    // Create list
    list := tview.NewList()
    list.SetBackgroundColor(colors.Background.Color()).
        SetMainTextColor(colors.Text.Color()).
        SetSelectedBackgroundColor(colors.Accent.Color()).
        SetSelectedTextColor(colors.Background.Color())
    
    container.AddItem(input, 1, 0, false)
    container.AddItem(list, 0, 1, true)
    
    return container
}
```

### Component Color Mapping Guide

| UI Element | Color Type | Usage |
|------------|------------|-------|
| Container Background | `Background` | Main component background |
| Container Border | `Border` | Component borders |  
| Container Title | `Title` | Component titles, headers |
| Text Content | `Text` | Main readable text |
| Accents/Highlights | `Accent` | Links, selected items, emphasis |

### Modal Components

For modal overlays and popup components:

```go
func (a *App) createModalComponent() {
    colors := a.GetComponentColors("modal")
    
    modal := tview.NewFlex().SetDirection(tview.FlexRow)
    modal.SetBorder(true).
        SetTitle("Modal Title").
        SetTitleColor(colors.Title.Color()).
        SetBackgroundColor(colors.Background.Color()).
        SetBorderColor(colors.Border.Color())
    
    // Apply ForceFilledBorderFlex for consistent rendering
    ForceFilledBorderFlex(modal)
    
    // Re-apply title styling after ForceFilledBorderFlex
    modal.SetTitleColor(colors.Title.Color())
    
    // Add as modal page
    a.Pages.AddPage("myModal", modal, true, true)
}
```

## 🎯 Component Types

### Predefined Component Types

| Component | Usage |
|-----------|--------|
| `general` | Default/generic components |
| `search` | Search overlays, filters |
| `ai` | AI-powered features |
| `slack` | Slack integration |
| `obsidian` | Obsidian integration |
| `stats` | Statistics and analytics |
| `prompts` | User prompts and dialogs |

### Adding New Component Types

1. **Define Component Type**:
```go
// internal/config/colors.go
const (
    ComponentTypeMyFeature config.ComponentType = "myfeature"
)
```

2. **Register Component**:
```go
app.themeService.RegisterComponent("myfeature", config.ComponentTypeMyFeature)
```

3. **Use in Themes**:
```yaml
overrides:
  myfeature:
    primary: "#custom-color"
    background: "#custom-bg"
```

## 🐛 Troubleshooting

### Common Issues

#### Colors Not Applying

**Problem**: Component colors not reflecting theme
**Solution**: Verify component is using `GetComponentColors()` API

```go
// Check if using correct API
colors := app.GetComponentColors("mycomponent") // ✅ Correct
colors := app.GetInputFieldColors()             // ❌ Legacy API
```

#### Inconsistent Theming

**Problem**: Some elements themed, others not
**Solution**: Ensure all UI elements use component colors

```go
// Theme all elements consistently
container.SetBackgroundColor(colors.Background.Color())  // ✅
text.SetBackgroundColor(colors.Background.Color())       // ✅
spacer.SetBackgroundColor(colors.Background.Color())     // ✅
```

#### Theme Not Loading

**Problem**: Custom theme not appearing
**Solution**: Check YAML syntax and file location

```bash
# Verify theme file location
ls themes/my-theme.yaml

# Validate YAML syntax
yamllint themes/my-theme.yaml
```

#### Legacy Theme Compatibility

**Problem**: Old themes not working
**Solution**: System automatically provides legacy fallbacks

```yaml
# Legacy structure (v1.0) - automatically supported
gmailTUI:
  body:
    fgColor: "#ffffff"
    bgColor: "#000000"
  ui:
    titleColor: "#yellow"
```

### Debug Theme Resolution

```go
// Debug color resolution
color := app.currentTheme.GetComponentColor(
    config.ComponentTypeSearch,
    config.ColorTypePrimary
)
fmt.Printf("Resolved color: %s (from %s)\n", color.String(), color.Source())
```

## 📚 Best Practices Summary

### DO ✅

- Use `GetComponentColors()` for all theming
- Register components with the theme service
- Apply colors to all UI elements consistently
- Use semantic color types appropriately
- Test themes in different terminal environments
- Provide meaningful component names

### DON'T ❌

- Hardcode colors in UI components
- Mix legacy and modern APIs
- Skip color application for any UI elements
- Use component overrides unnecessarily
- Ignore theme validation errors
- Create circular color dependencies

## 🔗 Related Documentation

- [COLORS.md](./COLORS.md) - Theme system architecture and API reference
- [ARCHITECTURE.md](./ARCHITECTURE.md) - Overall application architecture
- [Theme Files](../themes/) - Example theme implementations

---

**The hierarchical theme system ensures consistent, maintainable, and beautiful UI across all components.** 🎨