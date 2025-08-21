# 🎨 Theme Development Guidelines

This document provides guidelines for developers working on Gmail TUI to maintain consistency in theme color usage and ensure proper theme system integration.

## 🎯 Theme Color Usage Rules

### ✅ **DO Use Theme Colors**

Always use theme colors from the loaded configuration instead of hardcoding colors:

```go
// ✅ CORRECT - Use theme colors from global styles
list.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
text.SetTextColor(tview.Styles.PrimaryTextColor)
border.SetBorderColor(tview.Styles.BorderColor)
focusedBorder.SetBorderColor(tview.Styles.FocusColor)
```

### ❌ **DON'T Hardcode Colors**

Never hardcode colors as this breaks theming support:

```go
// ❌ WRONG - Hardcoded colors break theming
list.SetBackgroundColor(tcell.ColorBlack)
text.SetTextColor(tcell.NewRGBColor(255, 255, 255))
border.SetBorderColor(tcell.ColorGray)
```

## 🎨 Theme Color Categories

### **Body Colors** (`theme.Body.*`)
- **`FgColor`** - Primary text color for main content
- **`BgColor`** - Primary background color for the application
- **`LogoColor`** - Brand/accent color for highlights

**Usage:**
```go
textView.SetTextColor(tview.Styles.PrimaryTextColor)  // Uses Body.FgColor
textView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)  // Uses Body.BgColor
```

### **Frame Colors** (`theme.Frame.*`)
- **`Border.FgColor`** - Normal border color for unfocused components
- **`Border.FocusColor`** - Border color for focused/active components
- **`Title.*`** - Title bar colors and highlights

**Usage:**
```go
component.SetBorderColor(tview.Styles.BorderColor)  // Uses Frame.Border.FgColor
focusedComponent.SetBorderColor(tview.Styles.FocusColor)  // Uses Frame.Border.FocusColor
```

### **Email State Colors** (`theme.Email.*`)
- **`UnreadColor`** - Color for unread message indicators
- **`ReadColor`** - Color for read message indicators  
- **`ImportantColor`** - Color for important/priority messages
- **`SentColor`** - Color for sent message indicators
- **`DraftColor`** - Color for draft message indicators

**Usage:**
```go
// Email state colors are typically used in the email renderer
// Access through the loaded theme configuration
emailRenderer.UpdateFromConfig(themeConfig)
```

## 🔧 Implementation Patterns

### **1. Theme Application**
When applying themes, always update both global styles and existing widget colors:

```go
func (a *App) applyThemeConfig(theme *config.ColorsConfig) error {
    // 1. Update email renderer
    a.emailRenderer.UpdateFromConfig(theme)
    
    // 2. Update global styles
    tview.Styles.PrimitiveBackgroundColor = theme.Body.BgColor.Color()
    tview.Styles.PrimaryTextColor = theme.Body.FgColor.Color()
    tview.Styles.BorderColor = theme.Frame.Border.FgColor.Color()
    tview.Styles.FocusColor = theme.Frame.Border.FocusColor.Color()
    
    // 3. Update existing widget colors
    if list, ok := a.views["list"].(*tview.Table); ok {
        list.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
    }
    
    return nil
}
```

### **2. New Component Creation**
When creating new UI components, always use theme colors:

```go
func createNewComponent() *tview.TextView {
    component := tview.NewTextView()
    
    // ✅ Use theme colors
    component.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
    component.SetTextColor(tview.Styles.PrimaryTextColor)
    component.SetBorderColor(tview.Styles.BorderColor)
    
    return component
}
```

### **3. Focus Management**
Use appropriate colors for focused vs unfocused states:

```go
func updateFocusColors(component tview.Primitive, hasFocus bool) {
    if border, ok := component.(interface{ SetBorderColor(tcell.Color) }); ok {
        if hasFocus {
            border.SetBorderColor(tview.Styles.FocusColor)
        } else {
            border.SetBorderColor(tview.Styles.BorderColor)
        }
    }
}
```

## 📋 Development Checklist

When adding new UI components or features:

- [ ] **Use Global Styles**: All color assignments use `tview.Styles.*` values
- [ ] **No Hardcoded Colors**: No `tcell.Color*` or `tcell.NewRGBColor()` calls
- [ ] **Theme Refresh**: Component colors update when theme changes
- [ ] **Focus States**: Different colors for focused/unfocused states
- [ ] **Consistency**: Color usage matches existing patterns

## 🔍 Common Patterns

### **Modal/Dialog Components**
```go
modal := tview.NewModal()
modal.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
modal.SetTextColor(tview.Styles.PrimaryTextColor)
modal.SetButtonBackgroundColor(tview.Styles.BorderColor)
modal.SetButtonTextColor(tview.Styles.PrimaryTextColor)
```

### **List/Table Components**
```go
table := tview.NewTable()
table.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
table.SetBorderColor(tview.Styles.BorderColor)
// Individual cell colors should use email state colors when appropriate
```

### **Text Input Components**
```go
input := tview.NewInputField()
input.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
input.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
input.SetFieldTextColor(tview.Styles.PrimaryTextColor)
```

## 🧪 Testing Theme Integration

### **Manual Testing**
1. **Theme Switching**: Verify colors update when switching themes via `:theme set <name>`
2. **Component Focus**: Ensure focus colors change appropriately
3. **All Themes**: Test with both light and dark themes
4. **New Components**: Verify new components respect theme colors

### **Code Review**
- Search for hardcoded color patterns: `tcell.Color`, `NewRGBColor`, color hex codes
- Verify all UI components use `tview.Styles.*` for colors
- Check that new components are updated in `applyThemeConfig` method

## 🚨 Common Mistakes

### **1. Hardcoded Colors**
```go
// ❌ WRONG
component.SetBackgroundColor(tcell.ColorBlack)

// ✅ CORRECT  
component.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
```

### **2. Missing Theme Updates**
```go
// ❌ WRONG - Component won't update when theme changes
func (a *App) createComponent() {
    comp := tview.NewTextView()
    comp.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
    // Missing: Add to applyThemeConfig method
}

// ✅ CORRECT - Add to applyThemeConfig method
func (a *App) applyThemeConfig(theme *config.ColorsConfig) error {
    // ... existing code ...
    if comp, ok := a.views["newComponent"].(*tview.TextView); ok {
        comp.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
    }
    return nil
}
```

### **3. Focus Color Issues**
```go
// ❌ WRONG - Same color for focused and unfocused
component.SetBorderColor(tview.Styles.BorderColor)

// ✅ CORRECT - Different colors based on focus state
if focused {
    component.SetBorderColor(tview.Styles.FocusColor)
} else {
    component.SetBorderColor(tview.Styles.BorderColor)
}
```

## 📚 References

- **Color Configuration**: `internal/config/colors.go`
- **Theme Service**: `internal/services/theme_service.go`
- **Theme Commands**: `internal/tui/commands.go` (search for "executeTheme")
- **Application Integration**: `internal/tui/app.go` (search for "applyTheme")

## 💡 Best Practices

1. **Consistency**: Follow existing patterns in the codebase
2. **Accessibility**: Consider color contrast when creating themes
3. **Testing**: Always test with multiple themes
4. **Documentation**: Update this guide when adding new color categories
5. **Performance**: Theme switching should be instant

Remember: The goal is to make Gmail TUI fully themeable while maintaining code consistency and user experience quality!