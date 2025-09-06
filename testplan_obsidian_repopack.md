# 📦 Obsidian Repopack Feature - Test Plan

## 🎯 Test Objectives

This comprehensive test plan validates the Obsidian Repopack feature implementation, ensuring:
- Service layer functionality works correctly
- UI integration follows established patterns  
- Command parity is maintained
- Error handling is robust
- Performance is acceptable

## 📋 Test Categories

### 1. 🧪 Build & Functionality Tests

#### 1.1 Compilation Tests
**Test ID**: BUILD-001
**Objective**: Verify the repopack feature compiles and builds successfully
**Steps**:
1. Run `make build` in the project directory
2. Check for compilation errors related to Obsidian repopack functionality
3. Verify binary is created successfully

**Expected Results**:
- Build completes without errors
- No missing method or type errors for repopack functionality
- Binary file is created and executable

#### 1.2 Service Interface Tests
**Test ID**: SVC-001
**Objective**: Verify Obsidian service interface supports repopack mode
**Steps**:
1. Start GizTUI and ensure it loads without crashing
2. Check logs for any Obsidian service initialization errors
3. Verify no runtime errors during startup

**Expected Results**:
- Application starts successfully
- Obsidian service initializes properly
- No interface-related runtime errors
- All required services are available

#### 1.3 Configuration Integration Tests
**Test ID**: CFG-001
**Objective**: Ensure repopack feature integrates with existing configuration
**Steps**:
1. Check that existing Obsidian configuration still works
2. Verify repopack doesn't break existing single-file ingestion
3. Test with different Obsidian vault configurations

**Expected Results**:
- Existing Obsidian features remain functional
- Configuration parsing works correctly
- No regression in existing functionality

#### 1.4 Help System Integration Tests
**Test ID**: HELP-001
**Objective**: Verify repopack feature appears in help system
**Steps**:
1. Press `?` or `:help` to open help system
2. Search for "repack" or "obsidian" functionality
3. Verify command documentation is present

**Expected Results**:
- Help system mentions repopack functionality
- Command documentation includes `:obsidian repack`
- Keyboard shortcuts are properly documented

### 2. 🎨 UI Integration Tests

#### 2.1 Single Message Picker Tests
**Test ID**: UI-001
**Objective**: Validate single message form behavior
**Steps**:
1. Open single message Obsidian picker
2. Verify checkbox is present but labeled as disabled
3. Test Tab navigation between form elements
4. Submit form and verify repopack mode is ignored

**Expected Results**:
- Form displays correctly with comment input and checkbox
- Checkbox shows "(disabled for single message)"
- Tab navigation works smoothly
- Single message flow unchanged (ignores repopack setting)

#### 2.2 Bulk Message Picker Tests
**Test ID**: UI-002
**Objective**: Validate bulk message form behavior
**Steps**:
1. Select multiple messages and enter bulk mode
2. Open bulk Obsidian picker
3. Test checkbox functionality (enabled/disabled states)
4. Verify form submission with both modes

**Expected Results**:
- Checkbox is enabled and functional
- Both regular and repopack modes work correctly
- Form validation prevents empty submissions
- Progress indicators show correct message counts

#### 2.3 Focus Management Tests  
**Test ID**: UI-003
**Objective**: Verify proper focus handling
**Steps**:
1. Open Obsidian picker via keyboard shortcut
2. Test ESC key handling
3. Verify focus restoration after modal close
4. Test Tab cycling through form elements

**Expected Results**:
- Focus follows established "obsidian" pattern
- ESC key closes picker and restores focus to message list
- No focus deadlocks or UI blocking
- Visual focus indicators work correctly

#### 2.4 Theme Integration Tests
**Test ID**: UI-004
**Objective**: Test theming consistency  
**Steps**:
1. Switch between different themes
2. Open Obsidian picker in each theme
3. Verify color consistency across all form elements
4. Test with both light and dark themes

**Expected Results**:
- All form elements use `GetComponentColors("obsidian")`
- Colors remain readable in all themes
- No hardcoded colors visible
- Theme switching works without restart

### 3. 💻 Command Parity Tests

#### 3.1 Command Recognition Tests
**Test ID**: CMD-001
**Objective**: Verify command parsing works correctly
**Steps**:
1. Type `:obsidian repack` in command mode
2. Test short aliases `:obs repack` and `:obs r`
3. Verify command suggestions work with Tab completion
4. Test error handling for invalid syntax

**Expected Results**:
- All command variants are recognized
- Tab completion suggests `repack` for obsidian commands
- Error messages are helpful and accurate
- Command history preserves repack commands

#### 3.2 Bulk Mode Command Tests
**Test ID**: CMD-002
**Objective**: Test command behavior in bulk mode
**Steps**:
1. Select multiple messages
2. Execute `:obsidian repack` command
3. Verify repack-specific UI opens
4. Compare behavior with keyboard shortcut equivalent

**Expected Results**:
- Command works in bulk mode and opens repack UI
- Behavior matches keyboard shortcut exactly
- Single message mode shows appropriate message
- Command and keyboard shortcuts are functionally identical

#### 3.3 Command Suggestion Tests
**Test ID**: CMD-003
**Objective**: Validate auto-completion system
**Steps**:
1. Type partial commands: `:obs`, `:obs r`, `:obsidian r`
2. Use Tab to complete suggestions
3. Test edge cases and multiple suggestions
4. Verify normalization (repopack → repack)

**Expected Results**:
- Partial commands complete correctly
- `repopack` normalizes to `repack`
- Tab completion is intuitive and responsive
- No suggestion conflicts with existing commands

### 4. 🔄 Integration Tests

#### 4.1 End-to-End Workflow Tests
**Test ID**: INT-001
**Objective**: Test complete repopack workflow
**Steps**:
1. Select 3-5 test messages in bulk mode
2. Open Obsidian picker and enable repack mode
3. Add comment and submit
4. Verify file creation and content accuracy
5. Check database record creation

**Expected Results**:
- Single repopack file created with all selected messages
- File contains proper frontmatter and message compilation
- Database records track repopack operation
- Success message indicates repopack completion

#### 4.2 Error Handling Integration Tests
**Test ID**: INT-002
**Objective**: Verify error scenarios are handled gracefully
**Steps**:
1. Test with invalid Obsidian vault path
2. Test with messages that fail to load
3. Test with network interruption during operation
4. Test with insufficient disk space (if possible)

**Expected Results**:
- Error messages are user-friendly and actionable
- Partial failures are reported accurately
- No data corruption or inconsistent state
- Application remains stable after errors

#### 4.3 Performance Tests
**Test ID**: INT-003
**Objective**: Validate performance with large message sets
**Steps**:
1. Select 20+ messages for repopack
2. Monitor memory usage during processing
3. Measure file creation time
4. Test UI responsiveness during operation

**Expected Results**:
- Memory usage remains reasonable (< 100MB additional)
- Processing time scales linearly with message count
- UI remains responsive with progress indicators
- No memory leaks after operation completion

### 5. 🛡️ Security & Robustness Tests

#### 5.1 Input Validation Tests
**Test ID**: SEC-001
**Objective**: Test input sanitization and validation
**Steps**:
1. Submit empty comments and verify handling
2. Test with very long comments (>1000 chars)
3. Try special characters in comments
4. Test with malformed message data

**Expected Results**:
- Empty inputs handled gracefully
- Long inputs are properly truncated or validated
- Special characters don't break template rendering
- Malformed data doesn't cause crashes

#### 5.2 File System Security Tests
**Test ID**: SEC-002
**Objective**: Verify secure file operations
**Steps**:
1. Test with non-existent vault paths
2. Verify file permissions are set correctly
3. Test directory traversal protection
4. Check file overwrite behavior

**Expected Results**:
- Invalid paths are rejected safely
- Files created with appropriate permissions (0600)
- No directory traversal vulnerabilities
- Existing files are handled appropriately

### 6. 📱 User Experience Tests

#### 6.1 Accessibility Tests
**Test ID**: UX-001
**Objective**: Ensure feature is accessible and intuitive
**Steps**:
1. Test keyboard-only navigation through repopack flow
2. Verify screen reader compatibility (if applicable)
3. Test with different terminal sizes
4. Evaluate help text clarity

**Expected Results**:
- All functionality accessible via keyboard
- UI scales appropriately to terminal size
- Help text is clear and comprehensive
- No accessibility barriers identified

#### 6.2 Documentation Coverage Tests
**Test ID**: UX-002
**Objective**: Verify complete documentation coverage
**Steps**:
1. Check help system mentions repopack feature
2. Verify command documentation is complete
3. Test help search functionality for "repack"
4. Review error message clarity

**Expected Results**:
- Help system documents repopack functionality
- All commands and shortcuts are documented
- Help search finds repack-related content
- Error messages guide users to solutions

## 📝 Manual Testing Procedures

### Quick Repopack Test (5 minutes)
**Objective**: Verify basic repopack functionality works

**Steps**:
1. **Setup**: Start GizTUI with at least 3 emails in inbox
2. **Bulk Mode**: Press `v` to enter bulk mode
3. **Select Messages**: Use Space to select 2-3 messages
4. **Open Obsidian**: Press `O` (capital O)
5. **Enable Repopack**: Check the "Repopack Mode" checkbox
6. **Submit**: Press Enter to create repopack
7. **Verify**: Check Obsidian vault for new repopack file

**Success Criteria**:
- ✅ UI opens without hanging
- ✅ Checkbox is functional
- ✅ Repopack file created with all messages
- ✅ File follows `YYYY-MM-DD_repopack_N_messages.md` format

### Comprehensive Manual Test Suite

#### Test A: Single Message in Bulk Mode (Hanging Issue)
**Steps**:
1. Press `v` to enter bulk mode
2. Select exactly 1 message with Space
3. Press `O` to open Obsidian panel
4. Wait 5 seconds for UI to load
5. Try pressing Tab to navigate form
6. Press Esc to close panel

**Expected**: No hanging, UI responds normally

#### Test B: Repack vs Regular Mode Comparison
**Steps**:
1. Select 3 messages in bulk mode
2. Press `O` and submit WITHOUT checking repopack (regular mode)
3. Note result: 3 separate files created
4. Select same 3 messages again
5. Press `O` and submit WITH repopack checked
6. Note result: 1 combined file created

**Expected**: Clear difference in output files

#### Test C: Command Parity Test
**Steps**:
1. Select multiple messages in bulk mode
2. Type `:obsidian repack` command
3. Compare UI to pressing `O` + checking repopack checkbox
4. Try aliases: `:obs repack`, `:obs r`

**Expected**: All commands work identically

#### Test D: Error Handling Test
**Steps**:
1. Try repopack with invalid Obsidian configuration
2. Try with no messages selected
3. Try with 0-byte messages
4. Press Esc during repopack processing

**Expected**: Graceful error messages, no crashes

## 🏃‍♂️ Test Execution Guide

### Prerequisites
1. GizTUI build with repopack feature
2. Valid Gmail API access
3. Configured Obsidian vault path
4. Test email messages in Gmail account
5. Logging enabled for debugging (optional)

### Quick Test Execution Order
1. **Manual Quick Test** (5 minutes) - Verify basic functionality
2. **Build Tests** (BUILD-001)
3. **UI Integration Tests** (UI-001 through UI-004)  
4. **Command Parity Tests** (CMD-001 through CMD-003)
5. **Manual Test Suite** (Tests A through D) - 15 minutes total

### Full Test Execution Order
1. **Build & Functionality Tests** (BUILD-001 through HELP-001)
2. **UI Integration Tests** (UI-001 through UI-004)  
3. **Command Parity Tests** (CMD-001 through CMD-003)
4. **Integration Tests** (INT-001 through INT-003)
5. **Security Tests** (SEC-001 through SEC-002)
6. **User Experience Tests** (UX-001 through UX-002)
7. **Manual Test Suite** (Tests A through D)

### Test Environment
- **OS**: macOS Darwin 24.6.0 (primary), Linux (secondary)
- **Go Version**: Latest stable
- **Terminal**: Various terminal emulators
- **Gmail**: Live account with test data

### Success Criteria
- **All tests pass**: No failing test cases
- **Performance targets met**: < 100MB memory, < 5s processing for 10 messages
- **No regressions**: Existing Obsidian functionality unchanged
- **Documentation complete**: All features documented in help system

### Rollback Plan
If critical issues are discovered:
1. Document failing test cases
2. Create GitHub issues for tracking
3. Consider feature flag to disable repopack mode
4. Maintain backward compatibility with existing Obsidian integration

## 📊 Test Results Template

```
Test ID: ___________
Date: ___________
Tester: ___________
Environment: ___________

PASS / FAIL

Issues Found:
- [ ] Issue 1: Description
- [ ] Issue 2: Description

Notes:
___________

Recommendations:
___________
```

## 🔧 Automated Testing

### Unit Tests Location
- `internal/services/obsidian_service_test.go` - Service layer tests
- `internal/tui/obsidian_test.go` - UI integration tests

### Integration Tests Location  
- `test/integration/obsidian_repopack_test.go` - End-to-end workflow tests

### Test Commands
```bash
# Run unit tests
make test-unit

# Run integration tests  
make test-integration

# Run all tests
make test-all

# Run with coverage
make test-coverage
```

This comprehensive test plan ensures the Obsidian Repopack feature is thoroughly validated before release, maintaining GizTUI's high quality standards.