# Gmail TUI Theme System

Gmail TUI implements a modern hierarchical theme system that provides consistent, maintainable, and flexible color management across all UI components.

## 🎨 Hierarchical Theme Architecture

### Theme Structure Layers

1. **Foundation Colors** - Base colors for all components
   - `background` - Primary background color
   - `foreground` - Primary text color  
   - `border` - Default border color
   - `focus` - Focus highlight color

2. **Semantic Colors** - Meaning-based colors
   - `primary` - Main actions, titles
   - `secondary` - Supporting elements
   - `accent` - Highlights, links
   - `success/warning/error/info` - Status states

3. **Interaction Colors** - User interaction states
   - `selection.cursor` - Single item selection
   - `selection.bulk` - Multi-item selection
   - `input` - Input field styling
   - `statusBar` - Status bar colors

4. **Component Overrides** - Specialized component colors
   - Component-specific color customization
   - Overrides semantic/foundation defaults

## 📁 File Structure

```
gmail-tui/
├── themes/
│   ├── gmail-dark.yaml     # Dark (Dracula-inspired)
│   └── gmail-light.yaml    # Light
├── internal/
│   ├── config/
│   │   └── colors.go       # Base color system
│   └── render/
│       └── email.go        # Email renderer
└── docs/
    └── COLORS.md          # This documentation
```

## 🎯 Email State Colors

### Primary States

| State | Color | Description |
|-------|-------|-------------|
| **Unread** | `#ffb86c` (Orange) | New unread emails |
| **Read** | `#6272a4` (Gray) | Already read emails |
| **Important** | `#ff5555` (Red) | Marked as important |
| **Sent** | `#50fa7b` (Green) | Sent by the user |
| **Draft** | `#f1fa8c` (Yellow) | Saved drafts |

### Secondary States

| State | Color | Description |
|-------|-------|-------------|
| **From (Unread)** | `#ffb86c` | Highlight sender for unread |
| **From (Important)** | `#ff5555` | Sender of important email |
| **Subject (Unread)** | `#ffffff` | Bright white subject |
| **Subject (Read)** | `#6272a4` | Gray subject |

## 📝 Theme File Format

### Modern Theme Structure (v2.0+)

```yaml
gmailTUI:
  # Foundation colors - base for all components
  foundation:
    background: "#1e2540"       # Primary background
    foreground: "#c5d1eb"       # Primary text
    border: "#3f4a5f"          # Default borders
    focus: "#64b5f6"           # Focus highlights

  # Semantic colors - meaning-based
  semantic:
    primary: "#81c784"         # Titles, main actions
    secondary: "#7986cb"       # Supporting elements  
    accent: "#4fc3f7"          # Links, highlights
    success: "#81c784"         # Success states
    warning: "#ffb74d"         # Warning states
    error: "#f48fb1"          # Error states
    info: "#4fc3f7"           # Info states

  # Interaction colors - user states
  interaction:
    selection:
      cursor:
        bg: "#2d3748"          # Single item cursor
        fg: "#c5d1eb"          # Cursor text
      bulk:
        bg: "#1a202c"          # Multi-selection
        fg: "#c5d1eb"          # Bulk text
    input:
      bg: "#2d3748"            # Input background
      fg: "#c5d1eb"            # Input text
      label: "#81c784"         # Input labels
    statusBar:
      bg: "#3f4a5f"            # Status background
      fg: "#c5d1eb"            # Status text

  # Component overrides (optional)
  overrides:
    ai:
      primary: "#ba68c8"       # Purple AI titles
      accent: "#f48fb1"        # Pink AI accents
    search:
      primary: "#4fc3f7"       # Custom search titles
```

### Supported Color Formats

- **Hexadecimal**: `#ff5555`
- **Color names**: `red`, `blue`, `green`
- **ANSI codes**: `1`, `2`, `3`
- **Default**: `default` (terminal default color)

## 🔧 Technical Implementation

### Color Resolution Hierarchy

Colors are resolved in priority order:
1. **Component Override** → 2. **Semantic** → 3. **Foundation** → 4. **Legacy Fallback**

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

### Component Registration

```go
// Register components to use hierarchical theming
app.themeService.RegisterComponent("search", config.ComponentTypeSearch)
app.themeService.RegisterComponent("ai", config.ComponentTypeAI)
app.themeService.RegisterComponent("slack", config.ComponentTypeSlack)

// Components automatically receive theme updates
componentColors := app.GetComponentColors("search")
```

### Legacy Theme Support

The system maintains full backward compatibility:

```go
// Legacy structure (v1.0) - still supported
gmailTUI:
  body:
    fgColor: "#f8f8f2"
    bgColor: "#282a36"
  ui:
    titleColor: "#f1fa8c"
    inputBgColor: "#44475a"
```

## 🎨 Predefined Themes

### Dark (Dracula)

Inspired by the Dracula palette, optimized for low-light use.

**Highlights:**
- Dark background (`#282a36`)
- Light text (`#f8f8f2`)
- Purple accents (`#bd93f9`)
- Semantic colors by state

### Light

Designed for daylight and bright environments.

**Highlights:**
- Light background (`#ecf0f1`)
- Dark text (`#2c3e50`)
- Blue accents (`#3498db`)
- Optimized contrast

## 🚀 Advanced Usage

### Create a Custom Theme

1. **Copy an existing theme**:
   ```bash
   cp themes/gmail-dark.yaml themes/my-theme.yaml
   ```

2. **Edit colors**:
   ```yaml
   gmailTUI:
     email:
       unreadColor: "#ff6b6b"
       readColor: "#4ecdc4"
   ```

3. **Apply the theme**:
   ```go
   colors := config.LoadColorsFromFile("themes/my-theme.yaml")
   app.emailRenderer.UpdateFromConfig(colors)
   ```

### Dynamic Colors

Colors are applied dynamically based on email state:

- **Unread**: Attention-grabbing orange
- **Important**: Warning red
- **Draft**: Caution yellow
- **Sent**: Confirmation green
- **Read**: Subtle gray

## 🔍 Benefits

### For Users

✅ **Instant visual cues** — Clear states without reading text  
✅ **Full customization** — Themes tailored to preferences  
✅ **Improved accessibility** — Optimized contrast  
✅ **Consistent experience** — Same colors across the app  

### For Developers

✅ **Modular architecture** — Easy to extend  
✅ **External configuration** — No recompilation  
✅ **Code reuse** — Established patterns  
✅ **Simplified testing** — Predictable colors  

## 📋 Next Improvements

- [ ] **Automatic themes** — Detect system preference
- [ ] **Smooth transitions** — Animated theme switching
- [ ] **Custom palettes** — Theme generator
- [ ] **Export/Import** — Share themes
- [ ] **High-contrast mode** — Advanced accessibility

---

**The Gmail TUI color system delivers a rich and customizable visual experience.** 🎨

