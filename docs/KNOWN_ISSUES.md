# Known Issues - GizTUI v1.0.0

Current known issues and limitations in GizTUI v1.0.0.

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
**Status**: ✅ RESOLVED  
**Date Identified**: August 23, 2025  

The undo functionality had limitations that have been resolved:

#### 1. Move Operation Undo (RESOLVED)
**Problem**: Move operation undo was only restoring message to inbox, did not remove the applied label.

**Example Flow**:
1. Move message to "Work" label (applies "Work" label + archives message)
2. Press `U` to undo
3. ✅ Message returns to inbox 
4. ✅ "Work" label is now properly removed

**Root Cause**: Move operations were using simplified archive undo instead of proper move undo logic.

**Solution Applied**: 
- Modified move undo to use proper undoMove() function that removes applied labels
- Added immediate cache updates for move operations
- Messages now appear immediately in inbox with applied labels removed

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

#### 3. Trash Operation Undo (RESOLVED)
**Problem**: Undoing a trash (`d` then `U`) restored the message to the inbox list, but on the next sync or app restart the message disappeared again — it was still in the Trash on the server.

**Example Flow**:
1. Trash a message with `d`
2. Press `U` to undo
3. ✅ Message returns to the inbox list
4. Close and reopen the app
5. ❌ (before fix) Message is gone again — still trashed on Gmail

**Root Cause**: Two compounding problems.
1. The trash undo reversed the operation with a label modify (`messages.modify` removing `TRASH`). Gmail does **not** allow removing the `TRASH` system label via `messages.modify`, so the server-side untrash never happened — only the local list was updated, masking the failure until the next reload.
2. Even after switching to `messages.untrash`, the message left the Trash but still did not reappear: untrash clears `TRASH` but does **not** reliably restore the `INBOX` label, and the inbox list is fetched filtered by the `INBOX` label (`Messages.List(...).LabelIds("INBOX")`).

**Solution Applied**: Trash undo now (a) calls the Gmail `messages.untrash` endpoint (`UntrashMessage`) to clear `TRASH`, then (b) re-applies the labels the message had before trashing (INBOX, UNREAD, …; skipping `SENT`/`DRAFT`, which cannot be added via modify). The desktop client's `Untrash`/`BulkUntrash` were given the same INBOX restore for parity. Covered by unit tests in `internal/services/undo_service_test.go`.

### Impact Assessment
- **Move Undo**: ✅ **RESOLVED** - Messages restored to inbox with applied labels removed
- **Label Undo**: ✅ **RESOLVED** - Label operations can now be undone properly
- **Trash Undo**: ✅ **RESOLVED** - Untrash now persists server-side (via `messages.untrash`), so restored messages survive a reload/restart
- **User Experience**: ✅ **CONSISTENT** - All undo operations now work correctly
- **Workaround Available**: ✅ **NO WORKAROUNDS NEEDED** - All functionality works as expected

### Workaround
✅ **NO WORKAROUNDS NEEDED** - All undo functionality now works correctly with immediate cache updates

### Dependencies
- **Architecture**: Service layer undo recording system
- **Files Affected**: 
  - `internal/tui/labels.go` - Label UI operations
  - `internal/services/undo_service.go` - Undo logic
  - `internal/services/label_service.go` - Label business logic

### Status
**Documented Limitation** - Complex architectural changes required to fix properly without introducing application stability issues.

---

## ⚠️ Draft Composition Focus Flow Minor UX Issue

### Issue Description
**Type**: User Experience Issue  
**Severity**: Low (Minor UX)  
**Status**: Known Limitation  
**Date Identified**: August 30, 2025  

When loading a draft for editing via the draft picker, the composition panel displays correctly but requires one additional key press to fully activate focus in the +CC/BCC area and achieve optimal focus flow.

### Behavior Description
**Current Flow**:
1. Press `D` to open draft picker
2. Select a draft and press Enter
3. ✅ Composition panel displays immediately with draft content loaded
4. ✅ Focus works correctly for most fields (To, Subject, Body)  
5. ⚠️ **Minor Issue**: Requires one key press (any key) to fully activate focus for +CC/BCC area

**Expected vs Actual**:
- **Expected**: Seamless focus flow from draft selection to composition editing
- **Actual**: 99% seamless with minor focus activation step needed

### Technical Investigation
Extensive investigation revealed this is a **complex UI synchronization issue**:

**Root Cause**: Draft composition requires different UI initialization sequence than new composition:
- **New compositions** (Ctrl+N): Natural tview page flow handles focus automatically
- **Draft compositions**: Explicit page switching + UI redraw creates timing conflict

**Attempted Solutions**:
1. **Remove Tab simulation**: ❌ Caused app to hang completely
2. **Add delay to Tab simulation**: ❌ Still required extra key press  
3. **Remove redundant page operations**: ❌ Caused focus issues and hanging
4. **Simplify to match new composition flow**: ❌ Draft loading needs explicit page management

**Current Workaround**: Tab key simulation approach provides reliable, stable operation with minor UX impact.

### Impact Assessment
- **Functional Impact**: None - all draft editing features work correctly
- **User Experience**: Minor - requires one extra key press for optimal focus
- **Workaround Available**: ✅ Current approach is stable and reliable
- **Blocking Issues**: None - does not prevent draft editing workflow

### Current Implementation
The draft composition uses a **Tab simulation approach** in `internal/tui/app.go`:

```go
// Simulate a Tab key to trigger the composition panel's focus management
if a.compositionPanel != nil {
    tabEvent := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
    a.compositionPanel.InputHandler()(tabEvent, nil)
}
```

This approach ensures:
- ✅ Stable, non-hanging operation
- ✅ Draft content loads correctly  
- ✅ All editing functionality works
- ⚠️ Minor focus activation step needed

### Comparison with Other Workflows
- **New Email Composition (Ctrl+N)**: Perfect focus flow
- **Reply/Forward**: Perfect focus flow
- **Draft Editing**: 99% perfect with minor focus activation needed

### Dependencies
- **Framework**: tview focus management system
- **Architecture**: Page-based UI switching with explicit Draft → Composition flow
- **Files Affected**: 
  - `internal/tui/app.go` - `showCompositionWithDraft()` method
  - `internal/tui/composition.go` - `ShowWithComposition()` method

### Resolution Status
**Acceptable Limitation** - The current implementation provides:
- Reliable, stable operation without hanging
- Full draft editing functionality
- Minor UX trade-off for system stability

The focus flow issue represents a **cosmetic user experience consideration** rather than a functional problem. All draft management features work correctly with this approach.

### Future Investigation
Consider revisiting when:
- tview framework focus management is updated
- Alternative draft loading approaches are developed  
- Focus flow becomes a critical user-reported concern
- Major UI architecture refactoring is undertaken

---

## ✅ Status Bar Emoji Rendering Issues (RESOLVED)

### Issue Description
**Type**: Terminal Compatibility Issue  
**Severity**: Medium (Visual)  
**Status**: ✅ RESOLVED  
**Date Identified**: September 3, 2025  
**Date Resolved**: September 3, 2025  

Status bar messages occasionally displayed "broken" or garbled text when using certain emojis, particularly in tview TextView components.

### Root Cause Analysis
**Technical Cause**: Multi-codepoint emoji sequences containing **Variation Selector-16 (U+FE0F)** causing tview width calculation errors.

**Problematic Emojis Identified**:
- `⏱️` = STOPWATCH + VARIATION SELECTOR-16 (2 codepoints)
- `⚠️` = WARNING SIGN + VARIATION SELECTOR-16 (2 codepoints)
- `ℹ️` = INFORMATION SOURCE + VARIATION SELECTOR-16 (2 codepoints)

**Why This Breaks**:
- tview has known issues rendering multi-codepoint emoji sequences (GitHub issues #161, #236)
- Width calculations become incorrect, causing text layout problems
- Status bar `SetDynamicColors(true)` compounds the rendering issues
- Terminal width miscalculation leads to "broken" or shifted text display

### ✅ Resolution Implemented
**Solution**: Replace multi-codepoint emojis with visually similar single-codepoint alternatives.

**Replacements Made**:
```
⏱️ (Stopwatch + VS-16) → ⏰ (Alarm Clock)    - Time/timing indicators
⚠️ (Warning + VS-16)   → ❗ (Exclamation)    - Warning messages  
ℹ️ (Info + VS-16)      → 💡 (Light Bulb)     - Info messages
```

**Files Updated**:
- `internal/tui/status.go` - showInfo() and showLLMError() functions
- `internal/tui/messages.go` - Date validation message
- `internal/tui/bulk_prompts.go` - Processing indicators

### Impact Assessment
- **Functional Impact**: ✅ **RESOLVED** - Status messages display correctly
- **Visual Impact**: ✅ **IMPROVED** - Consistent emoji rendering across terminals  
- **Compatibility**: ✅ **UNIVERSAL** - All remaining emojis verified as single-codepoint safe
- **Code Changes**: ✅ **MINIMAL** - Only 5 character replacements across 3 files

### Emoji Safety Verification
**Analysis performed on all remaining emojis**: ✅ All 15 remaining emojis (✅🧠🧾🤖📊📝🔄💾📄🚀💬📦👥💰📅) are confirmed single-codepoint and tview-compatible.

**Resolution Status**: **COMPLETE** - All problematic multi-codepoint emoji sequences eliminated from status messages.

---

*For additional context and research approaches, see the LLM research prompt in this directory.*