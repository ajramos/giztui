# Known Issues

## ✅ Border Rendering Inconsistency in tview Components (RESOLVED)

### Issue Description
**Type**: Visual Bug  
**Severity**: Low (Cosmetic)  
**Status**: ✅ RESOLVED  
**Date Identified**: August 22, 2025  
**Date Resolved**: August 22, 2025  

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

### ✅ Resolution Implemented
This issue has been **resolved** using a targeted workaround:

**Solution**: `ForceFilledBorderFlex()` function in `internal/tui/layout.go`
- **Approach**: Replaces internal Box of Flex components with fresh `tview.NewBox()` (dontClear=false)
- **Root cause**: Table uses `dontClear=false` while Flex uses `dontClear=true` internally
- **Applied to**: textContainer, labelsFlex, slackFlex, cmdPanel
- **Theme integration**: `RefreshBordersForFilledFlexes()` ensures consistency across theme changes

**Benefits**:
- ✅ Consistent border appearance between Table and Flex components
- ✅ Non-intrusive workaround (no tview source modification required)
- ✅ Integrated with theme system for dynamic updates
- ✅ Well-documented implementation with clear technical reasoning

**Trade-offs**:
- ⚠️ Direct manipulation of tview internals (maintainability risk)
- ⚠️ Requires manual title styling reapplication
- ⚠️ May need updates if tview internal structure changes

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

## ⚠️ Undo Functionality Limitations

### Issue Description
**Type**: Functional Limitation  
**Severity**: Medium  
**Status**: Documented Limitation  
**Date Identified**: August 23, 2025  

The undo functionality has one remaining limitation affecting move operations:

#### 1. Move Operation Undo Limitation
**Problem**: Move operation undo only restores message to inbox, does not remove the applied label.

**Example Flow**:
1. Move message to "Work" label (applies "Work" label + archives message)
2. Press `U` to undo
3. ✅ Message returns to inbox 
4. ❌ "Work" label remains on the message

**Root Cause**: Move operations consist of two separate service calls:
- `LabelService.ApplyLabel()` records `UndoActionLabelAdd`  
- `EmailService.ArchiveMessage()` records `UndoActionArchive` (overwrites the first)
- Only the archive action gets recorded for undo

#### 2. Label Operations Undo (RESOLVED)  
**Problem**: Adding or removing labels individually was recording undo actions but undo failed silently.

**Example Flow**:
1. Select message and press `l` to open label manager
2. Add or remove a label
3. Press `U` to undo
4. ❌ Nothing happened - no message, no undo, silent failure

**Root Cause**: Circular dependency in undo architecture:
- UndoService called LabelService to reverse operations
- LabelService recorded new undo actions, overwriting original
- Created infinite loop causing silent failure

**Solution Applied**: Modified undo operations to use Gmail client directly instead of service layer, bypassing circular dependency issue.

### Impact Assessment
- **Move Undo**: Partial functionality - message restored but manual cleanup needed
- **Label Undo**: ✅ **RESOLVED** - Label operations can now be undone properly
- **User Experience**: Mostly consistent undo behavior, only move operations have limitations
- **Workaround Available**: Manual label management using `l` key for move operations

### Workaround
1. **For Move Undo**: After pressing `U`, manually remove unwanted labels using `l` key
2. **For Label Operations**: ✅ **NO WORKAROUND NEEDED** - Undo now works correctly

### Dependencies
- **Architecture**: Service layer undo recording system
- **Files Affected**: 
  - `internal/tui/labels.go` - Label UI operations
  - `internal/services/undo_service.go` - Undo logic
  - `internal/services/label_service.go` - Label business logic

### Status
**Documented Limitation** - Complex architectural changes required to fix properly without introducing application stability issues.

---

*For additional context and research approaches, see the LLM research prompt in this directory.*