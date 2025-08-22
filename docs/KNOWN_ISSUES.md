# Known Issues

## Border Rendering Inconsistency in tview Components

### Issue Description
**Type**: Visual Bug  
**Severity**: Low (Cosmetic)  
**Status**: Deferred  
**Date Identified**: August 22, 2025  

The application exhibits inconsistent border rendering between different tview component types, specifically:

- **`tview.Table`** components (Messages list): Render borders with "filled/solid" appearance
- **`tview.Flex`** components (Message Content, Theme Picker, etc.): Render borders with "hollow/transparent" appearance

### Visual Evidence
When comparing the Messages list (Table) with Message Content (Flex) and Theme Picker (Flex), the borders appear visually different despite using identical styling configuration:

```go
// Identical styling applied to all components
SetBorder(true).
SetBorderColor(tview.Styles.PrimitiveBackgroundColor).
SetBorderAttributes(tcell.AttrBold)
```

### Investigation Summary
Multiple approaches were attempted to resolve this inconsistency:

1. **Component-level styling changes**: Modified individual border colors and attributes
2. **Wrapper hierarchy approach**: Created background wrapper containers for inheritance
3. **Root-level styling**: Applied background at application root level
4. **Theme configuration changes**: Modified theme YAML files for border colors
5. **Global style overrides**: Programmatically overrode tview.Styles values

**Result**: None of the approaches successfully unified the border appearance between Table and Flex components.

### Root Cause Analysis
This appears to be a **library-level bug** in the tview framework where:

- `tview.Table` components have different internal border rendering logic than `tview.Flex` components
- The border background area is handled differently between component types
- Setting `SetBorderColor(tview.Styles.PrimitiveBackgroundColor)` creates different visual effects depending on the component type

### Impact Assessment
- **Functional Impact**: None - all features work correctly
- **Visual Impact**: Minor inconsistency in UI appearance
- **User Experience**: Does not affect usability or core functionality
- **Development Impact**: No blocking issues for new features

### Workaround Options
1. **Accept inconsistency**: Live with the visual difference as it doesn't affect functionality
2. **Library fork**: Fork tview and patch the border rendering logic
3. **Component migration**: Replace all Flex containers with Table components (high risk)
4. **Custom rendering**: Implement custom border drawing logic (complex)

### Dependencies
- **Library**: `github.com/derailed/tview v0.8.5`
- **Related**: `github.com/derailed/tcell/v2 v2.3.1-rc.4`

### Files Affected
- `internal/tui/layout.go` - Main layout components
- `internal/tui/themes.go` - Theme picker component  
- `themes/*.yaml` - Theme configuration files

### Deferred Resolution
This issue has been **deferred** due to:
- Low priority (cosmetic only)
- High complexity of potential fixes
- Risk of breaking existing functionality
- More important features requiring attention

### Future Investigation
Consider revisiting when:
- Upgrading to newer tview versions
- Major UI refactoring is planned
- Community solutions become available
- User feedback indicates higher priority

---

## Theme Preview Color Tag Processing Issue

### Issue Description
**Type**: Visual Bug  
**Severity**: Low (Cosmetic)  
**Status**: Deferred  
**Date Identified**: August 22, 2025  

Theme preview in the Theme Picker displays color tags as literal text instead of rendering them as colored text.

### Visual Evidence
When viewing theme previews, color tags appear as literal strings:
- Shows: `[#ffb86c]Primary Color[-]` 
- Should show: Colored text with "Primary Color" in the actual color

### Investigation Summary
Multiple approaches were attempted to resolve the color tag processing:

1. **Hex-to-named color mapping**: Implemented `hexToNamedColor()` function to convert hex values to tview-compatible named colors
2. **Dynamic color configuration**: Verified `SetDynamicColors(true)` was properly configured
3. **Color tag syntax validation**: Tested both hex (`[#ffb86c]`) and named (`[orange]`) color formats
4. **TextView configuration**: Ensured proper setup of color processing attributes

**Result**: None of the approaches successfully enabled color tag processing in the theme preview context.

### Root Cause Analysis
This appears to be a **library limitation** in the tview framework where:

- Dynamic color processing may not work in all contexts, particularly in TextView components with complex formatting
- Color tags (`[color]text[-]`) require specific rendering conditions that aren't met in the theme preview implementation
- The combination of dynamic content generation and color tag processing may have compatibility issues

### Impact Assessment
- **Functional Impact**: None - theme selection and application work correctly
- **Visual Impact**: Theme previews show color codes instead of actual colors
- **User Experience**: Users can still identify and select themes, but preview is less intuitive
- **Development Impact**: No blocking issues for theme functionality

### Workaround Options
1. **Accept limitation**: Live with text-based color indicators in theme previews
2. **Alternative preview**: Use color symbols (●) without color tags for visual distinction
3. **External preview**: Generate theme samples outside the main UI context
4. **Library upgrade**: Wait for potential fixes in newer tview versions

### Dependencies
- **Library**: `github.com/derailed/tview v0.8.5`
- **Related**: `github.com/derailed/tcell/v2 v2.3.1-rc.4`

### Files Affected
- `internal/tui/themes.go` - Theme picker and preview formatting
- `internal/tui/theme_helpers.go` - Color processing utilities

### Deferred Resolution
This issue has been **deferred** due to:
- Low priority (cosmetic only)
- Core theme functionality works correctly
- Library-level limitation requiring external fixes
- User can still effectively preview and select themes

### Future Investigation
Consider revisiting when:
- Upgrading to newer tview versions
- Alternative theme preview approaches are developed
- Community solutions for color tag processing become available
- Enhanced theme preview becomes a user-requested feature

---

*For additional context and research approaches, see the LLM research prompt in this directory.*