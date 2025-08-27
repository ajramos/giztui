# Theme Architecture Redesign Test Plan

## Feature Overview
This test plan validates the complete redesign of the Gmail TUI theme system from a legacy structure to a hierarchical color resolution system with automatic component registration and improved maintainability.

## Prerequisites

### Setup Requirements
1. **Build Environment**: Go 1.21+ with make utility
2. **Configuration**: Basic config with at least one email account configured
3. **Sample Data**: Test with sample emails (inbox, sent, drafts) for visual validation
4. **Terminal**: Terminal with good color support (24-bit color preferred)
5. **Theme Files**: Access to both built-in and custom theme files

### Pre-test Verification
```bash
# Verify build environment
make build
./build/gmail-tui --help

# Verify theme directory exists
ls -la ~/.config/giztui/themes/
ls -la themes/

# Verify test data access (optional)
# Have sample emails in different states (read/unread/important/draft)
```

## Test Scenarios

### 1. Theme Loading and Structure Tests

#### Test Case 1.1: New Hierarchical Theme Loading
**Objective**: Verify new hierarchical themes (v2.0) load correctly
**Steps**:
1. Create new theme file with hierarchical structure:
   ```yaml
   name: "Test Hierarchical"
   version: "2.0"
   gmailTUI:
     foundation:
       background: "#282a36"
       foreground: "#f8f8f2"
     semantic:
       primary: "#ff5555"
       accent: "#8be9fd"
   ```
2. Apply theme using `:theme test-hierarchical`
3. Navigate through different UI components
4. Verify colors resolve correctly from hierarchical structure

**Expected Result**: Theme loads successfully, colors display according to hierarchical rules
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 1.2: Legacy Theme Compatibility  
**Objective**: Verify legacy themes (v1.0) still work correctly
**Steps**:
1. Apply existing legacy theme (gmail-dark, slate-blue)
2. Navigate through all UI components
3. Verify all colors display as expected
4. Check no errors in logs

**Expected Result**: Legacy themes work without modification, backward compatibility maintained
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 1.3: Mixed Theme Structure Support
**Objective**: Verify themes with both new and legacy structures work
**Steps**:
1. Use slate-blue theme (has both structures)
2. Apply theme and navigate UI
3. Verify hierarchical colors take precedence where available
4. Verify legacy fallback for missing hierarchical colors

**Expected Result**: New structure takes precedence, legacy provides fallback
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 2. Color Resolution Tests

#### Test Case 2.1: Hierarchical Color Resolution
**Objective**: Verify color resolution follows correct hierarchy: component override → semantic → foundation → fallback
**Steps**:
1. Create test theme with all hierarchy levels defined
2. Verify AI component uses override colors (purple for title)
3. Verify general components use semantic colors (green for titles)
4. Verify border/background uses foundation colors
5. Verify fallback colors when theme is missing

**Expected Result**: Color resolution follows exact hierarchy order
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 2.2: Component-Specific Color Overrides
**Objective**: Verify component overrides work for AI, Prompts, Slack, etc.
**Steps**:
1. Apply slate-blue theme with component overrides
2. Open AI summary panel - verify purple title color
3. Apply bulk prompts - verify pink title color  
4. Open Slack integration - verify uses general green colors
5. Compare with general UI components

**Expected Result**: Component overrides display correctly, others use semantic colors
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 2.3: Selection Color Differentiation
**Objective**: Verify cursor vs bulk selection use different colors
**Steps**:
1. Navigate email list with arrow keys - observe cursor selection color
2. Select multiple messages with space key - observe bulk selection color
3. Mix cursor navigation with bulk selection
4. Verify visual distinction between selection types

**Expected Result**: Cursor and bulk selections use distinctly different colors
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 3. Component Theming Tests

#### Test Case 3.1: All UI Components Use Themed Colors
**Objective**: Verify no hardcoded colors remain in UI components
**Steps**:
1. Apply theme with distinctive colors (bright red titles, blue backgrounds)
2. Navigate through all UI areas:
   - Message list and content
   - Status bar
   - Input fields and search
   - Help system
   - Modal dialogs (labels, themes)
   - AI panels and prompts
3. Verify all elements use themed colors

**Expected Result**: All UI components reflect theme colors, no hardcoded colors visible
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 3.2: Status Message Colors
**Objective**: Verify status messages use theme-appropriate colors
**Steps**:
1. Trigger success messages (archive, label operations)
2. Trigger error messages (network failures, invalid operations)
3. Trigger warning messages (incomplete operations)
4. Trigger info messages (help, status updates)
5. Verify all use themed colors from semantic hierarchy

**Expected Result**: Status messages use correct semantic colors (success=green, error=red, etc.)
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 3.3: Input Field and Form Theming
**Objective**: Verify input fields use hierarchical interaction colors
**Steps**:
1. Open search interface (both simple and advanced)
2. Open command prompt (`:`)
3. Open label picker and theme picker
4. Verify background, text, and label colors match theme
5. Test with multiple themes

**Expected Result**: Input fields use interaction.input colors from theme
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 4. Theme Switching Tests

#### Test Case 4.1: Seamless Theme Switching
**Objective**: Verify theme changes apply across all components immediately
**Steps**:
1. Start with gmail-dark theme
2. Switch to slate-blue using `:theme slate-blue`
3. Observe all UI components update colors immediately
4. Switch back to gmail-dark
5. No refresh or restart required

**Expected Result**: All components update colors instantly when theme changes
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 4.2: Theme Persistence
**Objective**: Verify theme selection persists across app restarts
**Steps**:
1. Apply non-default theme
2. Restart application
3. Verify same theme is still active
4. Check config file updated correctly

**Expected Result**: Theme selection saved and restored on restart
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 4.3: Theme Preview Without Application
**Objective**: Verify theme preview works without changing active theme
**Steps**:
1. Use theme service preview functionality (if available via commands)
2. Preview different themes
3. Verify active theme remains unchanged
4. Verify preview shows correct colors

**Expected Result**: Theme preview works without affecting active theme
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 5. Visual Consistency Tests

#### Test Case 5.1: Consistent Color Application
**Objective**: Verify similar UI elements use consistent colors across the app
**Steps**:
1. Apply theme and note title colors in different areas
2. Verify all panel titles use same color
3. Verify all borders use same color
4. Verify all selection highlighting consistent
5. Check across different screen sizes/layouts

**Expected Result**: Similar elements use identical colors throughout application
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 5.2: Accessibility and Readability
**Objective**: Verify theme colors provide adequate contrast and readability
**Steps**:
1. Test with high-contrast themes
2. Verify text readable against backgrounds
3. Test selection highlighting visibility
4. Check status message visibility
5. Verify border distinction

**Expected Result**: All text readable, UI elements clearly distinguishable
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 6. Edge Cases and Error Handling Tests

#### Test Case 6.1: Invalid Theme Handling
**Objective**: Verify graceful handling of malformed theme files
**Steps**:
1. Create theme file with invalid YAML syntax
2. Attempt to load theme
3. Create theme file missing required colors
4. Verify error messages and fallback behavior

**Expected Result**: Invalid themes rejected gracefully, fallback colors used
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 6.2: Missing Theme File Handling
**Objective**: Verify behavior when theme files are missing
**Steps**:
1. Configure theme name that doesn't exist
2. Start application
3. Verify fallback to default theme
4. Check error handling and user feedback

**Expected Result**: Missing themes handled gracefully, default theme used
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 6.3: Theme Validation
**Objective**: Verify theme validation catches required field issues
**Steps**:
1. Create theme missing foundation.background
2. Create theme missing semantic.primary
3. Attempt to load themes
4. Verify validation errors reported

**Expected Result**: Theme validation prevents loading of incomplete themes
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 7. Performance Tests

#### Test Case 7.1: Theme Loading Performance
**Objective**: Verify theme loading doesn't impact startup performance
**Steps**:
1. Measure app startup time with various themes
2. Test with large theme files
3. Test with many theme files in directory
4. Compare with baseline performance

**Expected Result**: Theme loading adds minimal overhead to startup
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 7.2: Theme Switching Performance
**Objective**: Verify theme switching is instantaneous
**Steps**:
1. Switch between themes rapidly
2. Observe UI update speed
3. Test with complex themes
4. Measure memory usage during switches

**Expected Result**: Theme switching happens immediately without lag
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 8. Integration Tests

#### Test Case 8.1: Theme Integration with AI Features
**Objective**: Verify AI components use theme colors correctly
**Steps**:
1. Generate AI summaries with different themes
2. Apply AI prompts (single and bulk)
3. Verify AI panel titles use component-specific colors
4. Check streaming content respects theme colors

**Expected Result**: AI features fully integrated with theme system
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 8.2: Theme Integration with Bulk Operations
**Objective**: Verify bulk operations use theme colors correctly
**Steps**:
1. Enter bulk mode and select messages
2. Apply bulk operations (archive, label, delete)
3. Verify bulk selection highlighting
4. Check bulk operation status messages

**Expected Result**: Bulk operations use correct theme colors and selection highlighting
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 8.3: Theme Integration with Help System
**Objective**: Verify help system uses theme colors
**Steps**:
1. Open help with `?` key
2. Search within help using `/term`
3. Navigate help content
4. Verify all text and highlighting uses theme colors

**Expected Result**: Help system fully themed, searchable content properly highlighted
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 9. Regression Tests

#### Test Case 9.1: Existing Functionality Preservation
**Objective**: Verify all existing features work after theme redesign
**Steps**:
1. Test core email operations (read, compose, reply, archive)
2. Test search and filtering
3. Test keyboard shortcuts and commands
4. Test AI features and integrations
5. Test bulk operations

**Expected Result**: All existing functionality works unchanged
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

#### Test Case 9.2: Configuration Compatibility
**Objective**: Verify existing config files work with new theme system
**Steps**:
1. Use existing config file with old theme references
2. Verify automatic migration or compatibility handling
3. Check theme settings preserved correctly
4. Test config save/load cycle

**Expected Result**: Existing configurations work without modification
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### 10. User Workflow Tests

#### Test Case 10.1: Complete User Scenarios
**Objective**: Verify theme system doesn't interfere with normal usage
**Steps**:
1. **Daily Email Workflow**:
   - Check inbox
   - Read and respond to emails
   - Archive and organize messages
   - Apply labels and filters
2. **AI-Enhanced Workflow**:
   - Generate email summaries
   - Apply AI prompts for responses
   - Use bulk AI operations
3. **Customization Workflow**:
   - Browse and apply different themes
   - Switch themes based on preference/time of day
   - Verify theme persistence

**Expected Result**: All workflows function smoothly with theme system
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

## Expected Results Summary

### Critical Success Criteria
- [x] Application builds successfully without errors
- [ ] All existing functionality remains intact
- [ ] New hierarchical theme structure works correctly
- [ ] Legacy theme compatibility maintained
- [ ] Component-specific color overrides function properly
- [ ] Cursor vs bulk selection color differentiation working
- [ ] Theme switching applies immediately to all components
- [ ] No hardcoded colors visible in UI

### Performance Benchmarks
- Theme loading adds < 50ms to startup time
- Theme switching completes in < 100ms
- Memory usage increase < 5MB for theme system

### Visual Quality Targets
- All text maintains adequate contrast ratio (4.5:1 minimum)
- UI elements clearly distinguishable with any theme
- Consistent color application across all components
- Professional appearance with all provided themes

## Cleanup Steps

After completing all tests:

1. **Remove Test Files**: Clean up any temporary theme files created during testing
2. **Reset Configuration**: Restore original theme settings if changed
3. **Document Issues**: Record any bugs or inconsistencies found
4. **Performance Baseline**: Record final performance measurements
5. **User Experience Notes**: Document any UX improvements or concerns

## Risk Assessment

### High Risk Areas
- **Legacy Theme Compatibility**: Critical for existing users
- **Color Resolution Logic**: Complex hierarchy must work correctly  
- **Component Integration**: All components must use themed colors
- **Performance Impact**: Theme system must not slow down application

### Medium Risk Areas  
- **Theme Validation**: Important for preventing bad themes
- **Edge Case Handling**: Graceful failure modes required
- **Visual Consistency**: Important for professional appearance

### Low Risk Areas
- **Theme Preview**: Nice-to-have feature, not critical
- **Advanced Color Features**: Enhancements, not core functionality

## Success Metrics

- [ ] **100% Test Case Pass Rate**: All critical test cases must pass
- [ ] **Zero Regression Issues**: No existing functionality broken
- [ ] **Performance Targets Met**: Loading and switching performance acceptable
- [ ] **Visual Quality Maintained**: All themes look professional and readable
- [ ] **User Workflow Uninterrupted**: Daily usage patterns work seamlessly

This comprehensive test plan ensures the theme architecture redesign provides a robust, maintainable, and user-friendly theming system while maintaining full backward compatibility and performance standards.