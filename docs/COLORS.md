# Gmail TUI Color System

Gmail TUI implements a dynamic, k9s-inspired color system that allows full customization of the application's visual appearance.

## 🎨 Color System Architecture

### Configuration Layers

1. **Theme YAML files** (`skins/`)
   - Colors defined in YAML
   - Predefined themes (dark, light)
   - Full customization

2. **Application Configuration**
   - Dynamic theme loading
   - Global color application
   - Real-time updates (future work)

3. **Feature-specific Renderers**
   - Dynamic colors based on email state
   - Pluggable color functions
   - Built-in state logic

## 📁 File Structure

```
gmail-tui/
├── skins/
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

### YAML Structure

```yaml
gmailTUI:
  body:
    fgColor: "#f8f8f2"          # Main text
    bgColor: "#282a36"          # Main background
    logoColor: "#bd93f9"        # Logo

  frame:
    border:
      fgColor: "#44475a"        # Normal borders
      focusColor: "#6272a4"     # Focused borders
    
    title:
      fgColor: "#f8f8f2"        # Title
      bgColor: "#282a36"        # Title background
      highlightColor: "#f1fa8c" # Highlight
      counterColor: "#50fa7b"   # Counter
      filterColor: "#8be9fd"    # Filter

  table:
    fgColor: "#f8f8f2"          # Table text
    bgColor: "#282a36"          # Table background
    headerFgColor: "#50fa7b"    # Headers
    headerBgColor: "#282a36"    # Header background

  email:
    unreadColor: "#ffb86c"      # Unread
    readColor: "#6272a4"        # Read
    importantColor: "#ff5555"   # Important
    sentColor: "#50fa7b"        # Sent
    draftColor: "#f1fa8c"       # Drafts
```

### Supported Color Formats

- **Hexadecimal**: `#ff5555`
- **Color names**: `red`, `blue`, `green`
- **ANSI codes**: `1`, `2`, `3`
- **Default**: `default` (terminal default color)

## 🔧 Technical Implementation

### Email Renderer

```go
// EmailColorer handles email colors
type EmailColorer struct {
    UnreadColor    tcell.Color
    ReadColor      tcell.Color
    ImportantColor tcell.Color
    SentColor      tcell.Color
    DraftColor     tcell.Color
}

// ColorerFunc returns a color function for emails
func (ec *EmailColorer) ColorerFunc() func(*googleGmail.Message, string) tcell.Color {
    return func(message *googleGmail.Message, column string) tcell.Color {
        switch strings.ToUpper(column) {
        case "STATUS":
            if ec.isUnread(message) {
                return ec.UnreadColor
            }
            return ec.ReadColor
        case "FROM":
            if ec.isImportant(message) {
                return ec.ImportantColor
            }
            if ec.isUnread(message) {
                return ec.UnreadColor
            }
            return tcell.ColorWhite
        case "SUBJECT":
            if ec.isDraft(message) {
                return ec.DraftColor
            }
            if ec.isSent(message) {
                return ec.SentColor
            }
            if ec.isUnread(message) {
                return tcell.ColorWhite
            }
            return ec.ReadColor
        }
        return tcell.ColorWhite
    }
}
```

### State Detection

```go
// Helper methods to determine email state
func (ec *EmailColorer) isUnread(message *googleGmail.Message) bool {
    for _, labelId := range message.LabelIds {
        if labelId == "UNREAD" { return true }
    }
    return false
}

func (ec *EmailColorer) isImportant(message *googleGmail.Message) bool {
    importantLabels := []string{"IMPORTANT", "PRIORITY", "URGENT"}
    for _, labelId := range message.LabelIds {
        for _, important := range importantLabels {
            if strings.Contains(strings.ToUpper(labelId), important) { return true }
        }
    }
    return false
}
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
   cp skins/gmail-dark.yaml skins/my-theme.yaml
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
   colors := config.LoadColorsFromFile("skins/my-theme.yaml")
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

