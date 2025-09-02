# 🎨 GizTUI Theming System

A comprehensive guide for creating themes and implementing theme-compliant components in GizTUI.

## 📖 Overview

GizTUI implements a modern hierarchical theme system that provides consistent, maintainable, and flexible color management across all UI components. This system ensures visual consistency while allowing complete customization of the user interface.

## 🎨 Hierarchical Theme Architecture

### Theme Structure Layers

The theme system follows a three-layer hierarchy for color resolution:

1. **Component Overrides** - Highest priority, component-specific colors
2. **Semantic Colors** - Meaning-based colors (primary, accent, etc.)  
3. **Foundation Colors** - Base colors for all components

### Color Resolution Flow

```
Component Override → Semantic → Foundation → Default
```

When a component requests a color:
1. Check if component has a specific override
2. Fall back to semantic color (e.g., "primary")
3. Fall back to foundation color (e.g., "foreground")
4. Use system default as last resort

### Foundation Colors - Base for All Components
- `background` - Primary background color
- `foreground` - Primary text color  
- `border` - Default border color
- `focus` - Focus highlight color

### Semantic Colors - Meaning-Based Colors
- `primary` - Main actions, titles
- `secondary` - Supporting elements
- `accent` - Highlights, links
- `success/warning/error/info` - Status states

### Interaction Colors - User Interaction States
- `selection.cursor` - Single item selection
- `selection.bulk` - Multi-item selection
- `input` - Input field styling
- `statusBar` - Status bar colors

### Component Overrides - Specialized Colors
Component-specific color customization that overrides semantic/foundation defaults.

## 📁 File Structure

```
gmail-tui/
├── themes/
│   ├── slate-blue.yaml     # Default theme
│   ├── gmail-dark.yaml     # Dark theme  
│   ├── gmail-light.yaml    # Light theme
│   └── custom-example.yaml # Example custom theme
├── internal/
│   ├── config/
│   │   └── colors.go       # Theme system implementation
│   └── tui/
│       └── *.go           # UI components using theme system
└── docs/
    └── THEMING.md         # This documentation
```

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

// Invalid theme structure
❌ Missing foundation.background
❌ Invalid color format "#gggggg"
❌ Conflicting color definitions
```

## 🎯 Component System

### All Supported Components

GizTUI supports these 11 components, each with full theme integration:

| Component | Usage | Code Calls |
|-----------|-------|------------|
| `general` | Default UI layouts, containers | 25+ calls |
| `search` | Search panels, forms, filters | 12 calls |
| `attachments` | Attachment picker (A key) | 8 calls |
| `obsidian` | Obsidian integration features | 8 calls |
| `saved_queries` | Query bookmarks (Q/Z keys) | 6 calls |
| `slack` | Slack integration features | 6 calls |
| `prompts` | User prompts and dialogs | 3 calls |
| `ai` | AI-powered features | 2 calls |
| `labels` | Label management (l key) | 2 calls |
| `stats` | Statistics and analytics | 1 call |
| `links` | Link picker (L key) | 1 call |

### Component Selection Rules

1. **Feature Components**: AI, Slack, Obsidian, Links, Stats, Prompts - Use specific component colors
2. **System Components**: general, search - Use system component colors
3. **Picker Components**: attachments, saved_queries, labels - Use picker-specific component colors
4. **Content Display**: Use foundation/semantic colors when no component applies

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

## 🛠️ Implementing Theme-Compliant Components

### Mandatory Patterns for Developers

#### ✅ ALWAYS Use Component Colors

```go
// ✅ CORRECT - Use hierarchical system
componentColors := app.GetComponentColors("search")
container.SetBackgroundColor(componentColors.Background.Color())
container.SetTitleColor(componentColors.Title.Color())
container.SetBorderColor(componentColors.Border.Color())

// Apply to all UI elements consistently
input.SetFieldBackgroundColor(componentColors.Background.Color())
input.SetFieldTextColor(componentColors.Text.Color())
input.SetLabelColor(componentColors.Title.Color())

list.SetBackgroundColor(componentColors.Background.Color())
list.SetMainTextColor(componentColors.Text.Color())
list.SetSelectedBackgroundColor(componentColors.Accent.Color())
```

#### ❌ NEVER Use These Deprecated Patterns

```go
// ❌ NEVER - Hardcoded colors
container.SetBackgroundColor(tcell.ColorBlue)
container.SetTitleColor(tcell.ColorYellow)

// ❌ NEVER - Legacy theme methods  
container.SetTitleColor(a.getTitleColor())        // REMOVED
container.SetBackgroundColor(a.getFooterColor())  // DEPRECATED

// ❌ NEVER - Hardcoded styles
container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
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
    
    // Create container with proper theming
    container := tview.NewFlex().SetDirection(tview.FlexRow)
    container.SetBorder(true).
        SetTitle("My Component").
        SetTitleColor(colors.Title.Color()).
        SetBackgroundColor(colors.Background.Color()).
        SetBorderColor(colors.Border.Color())
    
    // Create themed input field
    input := tview.NewInputField().
        SetLabel("Input: ").
        SetFieldBackgroundColor(colors.Background.Color()).
        SetFieldTextColor(colors.Text.Color()).
        SetLabelColor(colors.Title.Color())
    
    // Create themed list
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

### Modal Components Pattern

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

## 🔧 Technical Implementation

### Modern Theme API

```go
// Get component-specific colors using hierarchical system
componentColors := app.GetComponentColors("search")

// Use component colors for consistent theming
container.SetBackgroundColor(componentColors.Background.Color())
container.SetBorderColor(componentColors.Border.Color())
container.SetTitleColor(componentColors.Title.Color())

input.SetFieldBackgroundColor(componentColors.Background.Color())
input.SetFieldTextColor(componentColors.Text.Color())
input.SetLabelColor(componentColors.Title.Color())
```

### Component Color Set

```go
type ComponentColorSet struct {
    Border     Color // Component border color
    Title      Color // Component title color  
    Background Color // Component background color
    Text       Color // Component text color
    Accent     Color // Component accent/highlight color
}
```

### Direct Color Access

```go
// Get specific colors using the hierarchy
titleColor := app.currentTheme.GetComponentColor(
    config.ComponentTypeSearch, 
    config.ColorTypePrimary
).Color()

bgColor := app.currentTheme.GetComponentColor(
    config.ComponentTypeGeneral,
    config.ColorTypeBackground
).Color()
```

## 🎨 Built-in Themes

### Slate Blue (Default)
Modern dark theme with blue/slate color palette and cyan accents, optimized for low-light use.

### Dracula
Official Dracula theme implementation with characteristic dark purple background and vibrant accent colors (purple, pink, cyan, green, orange, red, yellow). Based on the official Dracula color specification.

### Gmail Dark  
Dark theme inspired by Dracula palette with purple accents.

### Gmail Light
Clean light theme designed for daylight and bright environments with blue accents and optimized contrast.

### Custom Example
Demonstration theme showing customization possibilities with example component overrides.

## 🚀 Advanced Usage

### Create a Custom Theme

1. **Copy an existing theme**:
   ```bash
   cp themes/slate-blue.yaml themes/my-theme.yaml
   ```

2. **Edit colors**:
   ```yaml
   gmailTUI:
     foundation:
       background: "#1a1a1a"
       foreground: "#e0e0e0"
     semantic:
       primary: "#00ff88" 
       accent: "#ff6b35"
   ```

3. **Apply the theme**:
   ```bash
   :theme set my-theme
   ```

### Theme Switching Commands

- `:theme` or `:th` - Show current theme  
- `:theme list` - List all available themes
- `:theme set <name>` - Switch to specified theme
- `:theme preview <name>` - Preview theme before applying

## 🐛 Troubleshooting

### Common Issues

#### Colors Not Applying

**Problem**: Component colors not reflecting theme  
**Solution**: Verify component is using `GetComponentColors()` API

```go
// Check if using correct API
colors := app.GetComponentColors("mycomponent") // ✅ Correct
colors := app.GetInputFieldColors()             // ❌ Deprecated
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
- Apply colors to all UI elements consistently
- Use semantic color types appropriately
- Test themes in different terminal environments
- Provide meaningful component names
- Follow the component selection rules

### DON'T ❌

- Hardcode colors in UI components
- Use deprecated theme methods
- Skip color application for any UI elements
- Use component overrides unnecessarily
- Ignore theme validation errors
- Create circular color dependencies

## 🔍 Benefits

### For Users

✅ **Instant visual cues** — Clear states without reading text  
✅ **Full customization** — Themes tailored to preferences  
✅ **Improved accessibility** — Optimized contrast  
✅ **Consistent experience** — Same colors across the app  
✅ **Runtime switching** — Change themes instantly without restart

### For Developers

✅ **Modular architecture** — Easy to extend  
✅ **External configuration** — No recompilation  
✅ **Code reuse** — Established patterns  
✅ **Simplified testing** — Predictable colors
✅ **Clear guidelines** — Documented patterns prevent mistakes

## 📋 Next Improvements

- [ ] **Automatic themes** — Detect system preference
- [ ] **Smooth transitions** — Animated theme switching
- [ ] **Custom palettes** — Theme generator
- [ ] **Export/Import** — Share themes
- [ ] **High-contrast mode** — Advanced accessibility

---

**The hierarchical theme system ensures consistent, maintainable, and beautiful UI across all components.** 🎨