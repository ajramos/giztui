# 🎨 Theme System Test Plan

This document provides a comprehensive test plan for Gmail TUI's theme system to ensure all functionality works correctly across different themes and configurations.

## 🧪 Test Environment Setup

### Prerequisites
- [ ] Clean build: `make build` succeeds
- [ ] Default themes present in `themes/` directory
- [ ] Custom theme directory configured (optional)
- [ ] Test themes available for comprehensive testing

### Test Data Preparation
```bash
# Verify default themes exist
ls themes/
# Expected: gmail-dark.yaml, gmail-light.yaml, custom-example.yaml

# Verify application builds successfully
make build

# Verify configuration files have theme settings
grep -A2 -B2 "theme_picker\|color_scheme" examples/config.json
grep -A2 -B2 "theme_picker\|color_scheme" ~/.config/giztui/config.json

# Test theme discovery from different working directories
cd examples && ./build/gmail-tui --help  # Should find ../themes
cd build && ./gmail-tui --help           # Should find ../themes
```

## 🔧 Core Functionality Tests

### Theme Path Resolution

#### Test 0.1: Theme Discovery from Root Directory
- [ ] **Setup**: Run application from root directory where themes/ exists
- [ ] **Command**: `:theme list`
- [ ] **Expected**: Finds themes in local themes/ directory
- [ ] **Expected**: Shows all 3 themes (gmail-dark, gmail-light, custom-example)

#### Test 0.2: Theme Discovery from Examples Directory  
- [ ] **Setup**: Run application from examples/ subdirectory
- [ ] **Command**: `:theme list`
- [ ] **Expected**: Finds themes in ../themes/ directory
- [ ] **Expected**: Shows all 3 themes (not just local examples/themes if it exists)

#### Test 0.3: Theme Discovery from Build Directory
- [ ] **Setup**: Run application from build/ subdirectory  
- [ ] **Command**: `:theme list`
- [ ] **Expected**: Finds themes via executable-relative path resolution
- [ ] **Expected**: All themes available regardless of working directory

#### Test 0.4: Missing Themes Directory
- [ ] **Setup**: Temporarily rename themes/ directory
- [ ] **Command**: `:theme list`
- [ ] **Expected**: Appropriate error about no themes found
- [ ] **Expected**: Application doesn't crash

### Theme Picker UI

#### Test 1.1: Open Theme Picker
- [ ] **Keyboard**: Press `H` key (configurable shortcut)
- [ ] **Command**: `:theme` (without arguments)
- [ ] **Expected**: Theme picker **side panel** opens (not modal)
- [ ] **Expected**: Side panel appears next to message content
- [ ] **Expected**: Search field focused initially
- [ ] **Expected**: All themes listed (gmail-dark, gmail-light, custom-example)
- [ ] **Expected**: Current theme marked with ✅ (green checkmark)
- [ ] **Expected**: Other themes marked with ○ (circle)
- [ ] **Expected**: Same icon pattern as labels picker
- [ ] **Expected**: Header title is yellow like other panels
- [ ] **Expected**: Border highlighted when focused

#### Test 1.2: Theme Picker Navigation
- [ ] **Keys**: ↑/↓ arrows, Page Up/Down
- [ ] **Expected**: Can navigate through theme list
- [ ] **Keys**: Tab between search field and list
- [ ] **Expected**: Focus moves correctly between components
- [ ] **Key**: ESC
- [ ] **Expected**: Closes theme picker and returns focus

#### Test 1.3: Theme Search and Filter
- [ ] **Action**: Type in search field
- [ ] **Expected**: Theme list filters in real-time
- [ ] **Action**: Clear search field
- [ ] **Expected**: All themes visible again

#### Test 1.4: Custom Themes in Picker
- [ ] **Setup**: Add custom theme to `~/.config/giztui/themes/`
- [ ] **Action**: Open theme picker
- [ ] **Expected**: Custom themes visible alongside built-in themes
- [ ] **Expected**: No duplicates if same name exists in multiple directories

### Theme Preview

#### Test 2.1: Preview Theme from Picker
- [ ] **Action**: Select theme in picker, press Enter
- [ ] **Expected**: Theme preview shows in text panel
- [ ] **Expected**: Preview shows theme name, colors, and descriptions
- [ ] **Expected**: Email colors displayed with hex values
- [ ] **Expected**: UI colors displayed with hex values  
- [ ] **Expected**: Text container title changes to "🎨 Theme Preview"
- [ ] **Expected**: Message headers hidden during preview

#### Test 2.2: Preview Different Themes
- [ ] **Action**: Navigate to different themes and preview
- [ ] **Expected**: Preview content updates for each theme
- [ ] **Expected**: Color values change appropriately
- [ ] **Expected**: Current active theme unchanged

### Theme Application

#### Test 3.1: Apply Theme with Space Key
- [ ] **Action**: Select theme in picker, press Space
- [ ] **Expected**: Theme applied immediately
- [ ] **Expected**: UI colors update throughout application
- [ ] **Expected**: Success message: "Applied theme: [theme-name]" (no duplicate emoji)
- [ ] **Expected**: Theme picker closes automatically
- [ ] **Verify**: Message list, text view, borders all use new theme colors

### Command Interface (Fallback)

#### Test 3.2: Command-based Theme Listing
- [ ] **Command**: `:theme list`
- [ ] **Expected**: Theme list displays in main text view (not just status bar)
- [ ] **Expected**: Shows all themes with current theme marked ✅
- [ ] **Expected**: Other themes marked with ○
- [ ] **Expected**: Text container title changes to "🎨 Theme List"
- [ ] **Expected**: Usage hints and commands displayed
- [ ] **Expected**: Status bar shows summary message

#### Test 3.3: Command-based Theme Switching
- [ ] **Command**: `:theme set gmail-light`
- [ ] **Expected**: Theme applied immediately
- [ ] **Expected**: Success message: "🎨 Applied theme: gmail-light"
- [ ] **Expected**: UI colors update throughout application

#### Test 3.4: Command-based Theme Preview  
- [ ] **Command**: `:theme preview gmail-dark`
- [ ] **Expected**: Preview information shown in status
- [ ] **Expected**: Current theme remains unchanged

#### Test 3.5: Invalid Commands
- [ ] **Command**: `:theme set nonexistent-theme`
- [ ] **Expected**: Error message about theme not found
- [ ] **Command**: `:theme preview invalid-theme`
- [ ] **Expected**: Error message about theme not found
- [ ] **Command**: `:theme set` (no args)
- [ ] **Expected**: Usage error message

### Current Theme Status

#### Test 4.1: Show Current Theme
- [ ] **Command**: `:theme`
- [ ] **Expected**: Shows current theme name (e.g., "Current theme: gmail-dark")

#### Test 4.2: Verify Theme Persistence
- [ ] **Command**: `:theme set gmail-light`
- [ ] **Restart application** (if possible in testing)
- [ ] **Expected**: Theme persists based on configuration

## 🎨 UI Integration Tests  

### Color Application

#### Test 5.1: Message List Colors
- [ ] **Switch to dark theme**
- [ ] **Verify**: Message list background is dark
- [ ] **Verify**: Unread messages use correct unread color
- [ ] **Verify**: Read messages use correct read color
- [ ] **Verify**: Important messages use correct important color

#### Test 5.2: Text View Colors
- [ ] **Switch to light theme**
- [ ] **Open a message**
- [ ] **Verify**: Text view background is light
- [ ] **Verify**: Text color is appropriate for light background
- [ ] **Verify**: Headers use theme colors

#### Test 5.3: Border and Focus Colors
- [ ] **Switch theme and navigate with Tab**
- [ ] **Verify**: Unfocused components use normal border color
- [ ] **Verify**: Focused component uses focus color
- [ ] **Verify**: Focus indicator updates when switching themes

#### Test 5.4: Modal and Dialog Colors
- [ ] **Open command bar** (`:`)
- [ ] **Verify**: Command bar uses theme colors
- [ ] **Open any modal dialog**
- [ ] **Verify**: Modal background and text use theme colors

#### Test 5.5: Status Message Colors
- [ ] **Switch themes**
- [ ] **Trigger success message** (e.g., archive message)
- [ ] **Trigger error message** (e.g., invalid command)
- [ ] **Verify**: Status messages use appropriate theme colors

### Layout Elements

#### Test 6.1: All UI Panels
- [ ] **Switch to different themes**
- [ ] **Verify each panel uses theme colors**:
  - [ ] Message list panel
  - [ ] Message content panel  
  - [ ] Header panel
  - [ ] AI summary panel (if visible)
  - [ ] Help panel (if visible)

#### Test 6.2: Component Focus States
- [ ] **Use Tab to cycle through components**
- [ ] **For each theme, verify**:
  - [ ] Focused component has focus color border
  - [ ] Unfocused components have normal border color
  - [ ] Focus colors are visually distinct

## ⚡ Performance Tests

#### Test 7.1: Theme Switching Speed
- [ ] **Switch themes rapidly**: `:theme set gmail-dark`, `:theme set gmail-light`
- [ ] **Expected**: Each switch is instant (< 100ms perceived)
- [ ] **Expected**: No UI flicker or delay
- [ ] **Expected**: No memory leaks from repeated switching

#### Test 7.2: Large Theme Directories
- [ ] **Setup**: Add many theme files to theme directories
- [ ] **Command**: `:theme list`  
- [ ] **Expected**: List loads quickly even with many themes
- [ ] **Expected**: No performance degradation

#### Test 7.3: Preview Modal Performance
- [ ] **Rapidly preview different themes**
- [ ] **Expected**: Preview modal opens quickly
- [ ] **Expected**: Color information displays immediately

## 🔧 Configuration Tests

#### Test 8.1: Default Theme Configuration
- [ ] **Setup**: Configure `"color_scheme": "gmail-light"` in layout section
- [ ] **Restart application** (or test startup)
- [ ] **Expected**: Application starts with light theme active
- [ ] **Verify**: Message list, borders, text colors all use light theme

#### Test 8.2: Custom Theme Directory
- [ ] **Setup**: Configure `"theme_dir": "/custom/path"` in layout section
- [ ] **Add custom theme YAML file to that directory**
- [ ] **Open theme picker**: Press `H` key
- [ ] **Expected**: Custom themes from specified directory visible
- [ ] **Expected**: Built-in themes also visible alongside custom themes

#### Test 8.3: Configurable Shortcut Key
- [ ] **Setup**: Change `"theme_picker": "T"` in keys section
- [ ] **Restart application**
- [ ] **Test**: Press `T` key (should open theme picker)
- [ ] **Test**: Press `H` key (should NOT open theme picker)
- [ ] **Expected**: Theme picker opens with new configured key

#### Test 8.4: Invalid Configuration Values
- [ ] **Setup**: Configure `"color_scheme": "invalid-theme"`
- [ ] **Start application**
- [ ] **Expected**: Falls back to default theme gracefully
- [ ] **Expected**: No application crash
- [ ] **Setup**: Configure `"theme_picker": ""` (empty)
- [ ] **Expected**: Theme picker accessible via command `:theme`

#### Test 8.5: Missing Configuration Fields
- [ ] **Setup**: Remove `"theme_picker"` field from config
- [ ] **Expected**: Uses default `"H"` key
- [ ] **Setup**: Remove `"color_scheme"` field from config
- [ ] **Expected**: Uses default theme (gmail-dark)

## 🚨 Error Handling Tests

#### Test 9.1: Missing Theme Files
- [ ] **Setup**: Remove theme files temporarily
- [ ] **Command**: `:theme set gmail-dark`
- [ ] **Expected**: Appropriate error message
- [ ] **Expected**: Current theme unchanged

#### Test 9.2: Corrupted Theme Files
- [ ] **Setup**: Create invalid YAML theme file
- [ ] **Command**: `:theme set invalid-theme`
- [ ] **Expected**: Parse error message
- [ ] **Expected**: Application doesn't crash

#### Test 9.3: Permission Issues
- [ ] **Setup**: Remove read permissions from theme directory
- [ ] **Command**: `:theme list`
- [ ] **Expected**: Permission error message
- [ ] **Expected**: Graceful fallback

## ♿ Accessibility Tests

#### Test 10.1: High Contrast Theme
- [ ] **Create or use high contrast theme**
- [ ] **Apply theme**
- [ ] **Verify**: Sufficient color contrast for readability
- [ ] **Verify**: Focus indicators clearly visible

#### Test 10.2: Color Blind Considerations
- [ ] **Test with multiple themes**
- [ ] **Verify**: Important state differences not only color-based
- [ ] **Verify**: Color combinations don't conflict for color blind users

## 📋 Command Integration Tests

#### Test 11.1: Keyboard Shortcuts and Command Aliases
- [ ] **Default Keyboard**: `H` key opens theme picker (configurable)
- [ ] **Command**: `:theme` opens theme picker (no arguments)
- [ ] **Aliases**: `:th` works same as `:theme`
- [ ] **Subcommands**: `:theme list`, `:theme l`, `:theme set`, `:theme s`, etc.
- [ ] **Configuration**: Theme picker key configurable via `config.json`
- [ ] **Integration**: Works with existing configurable shortcuts system
- [ ] **Expected**: All shortcuts and aliases work identically

#### Test 11.2: Tab Completion (if implemented)
- [ ] **Type**: `:theme <TAB>`
- [ ] **Expected**: Shows available subcommands
- [ ] **Type**: `:theme set <TAB>`
- [ ] **Expected**: Shows available theme names

#### Test 11.3: Command History
- [ ] **Execute**: Several theme commands
- [ ] **Use Up arrow** in command bar
- [ ] **Expected**: Can cycle through theme command history

## 📊 Results Template

```
Test Date: ___________
Tester: ______________
Gmail TUI Version: ___________

## Core Functionality
✅ PASSED: Theme discovery shows all built-in themes
✅ PASSED: Theme switching updates UI immediately  
✅ PASSED: Theme preview shows correct color information
❌ FAILED: Custom theme directory not working - needs investigation
⚠️  PARTIAL: Some UI components not updating on theme switch

## UI Integration  
✅ PASSED: Message list colors update correctly
✅ PASSED: Text view respects theme colors
❌ FAILED: Modal dialogs still using hardcoded colors
✅ PASSED: Focus states use correct theme colors

## Performance
✅ PASSED: Theme switching is instant
✅ PASSED: Large theme directories load quickly
✅ PASSED: No memory leaks detected

## Error Handling
✅ PASSED: Invalid theme names handled gracefully
✅ PASSED: Missing theme files show appropriate errors  
⚠️  PARTIAL: Corrupted theme files cause issues - needs better validation

## Configuration
❌ FAILED: Default theme configuration not working
✅ PASSED: Custom theme directory configuration works
✅ PASSED: Invalid config values fallback correctly

## Issues Found
1. Modal dialogs not respecting theme colors
2. Default theme config not being applied
3. Need better validation for corrupted theme files

## Recommendations
1. Update modal color application in applyThemeConfig
2. Fix config loading for default theme setting
3. Add YAML validation before theme loading
```

## 🎯 Test Success Criteria

**All tests must pass for theme system to be considered production ready:**

- ✅ All built-in themes available and functional
- ✅ Theme picker opens as **side panel** (not modal)
- ✅ Theme switching works instantly without issues  
- ✅ All UI components respect theme colors
- ✅ **Configurable shortcut key** works properly
- ✅ **Configuration file integration** (color_scheme, theme_dir, theme_picker)
- ✅ **Default theme loading** from config on startup
- ✅ **Custom theme directory** support works
- ✅ Error handling prevents crashes
- ✅ Performance acceptable under normal usage
- ✅ Accessibility requirements met
- ✅ **Side panel UX** matches existing prompt picker patterns

## 🔄 Continuous Testing

**After any theme-related changes:**
1. Run core functionality tests (Tests 1-4)
2. Verify UI integration (Tests 5-6) 
3. Check performance impact (Test 7)
4. Test error scenarios (Test 9)

**Before releases:**
- Execute full test plan
- Test with custom user themes  
- Verify backward compatibility
- Check accessibility compliance