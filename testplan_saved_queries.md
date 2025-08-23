# Test Plan: Saved Queries Feature

## Feature Overview
The Saved Queries feature allows users to save, organize, and quickly access frequent search patterns as bookmarks. Users can save current searches using an Obsidian-style bottom-right input panel, browse existing bookmarks with a prompts-style picker interface, and execute saved queries instantly through keyboard shortcuts, number keys (1-9), and commands. The feature provides consistent UI patterns and theming throughout.

## Prerequisites

### Configuration Requirements
- Valid Gmail TUI configuration with OAuth credentials
- Internet connection for Gmail API access  
- Test Gmail account with diverse message types for search testing
- SQLite database permissions for bookmark storage

### Sample Data Setup
1. **Diverse Message Types** - Ensure test account has:
   - Messages with attachments (`has:attachment`)
   - Messages from specific senders (`from:boss@company.com`)
   - Messages with labels (`label:important`)
   - Unread messages (`is:unread`)
   - Messages in date ranges (`after:2024-01-01`)
   - Large messages (`larger:1MB`)

2. **Search Query Examples** for testing:
   - Simple: `has:attachment`
   - Complex: `from:notifications@github.com has:attachment after:2024-01-01`
   - Label-based: `label:work is:unread`
   - Date range: `before:2023-12-31 larger:5MB`
   - Subject-based: `subject:"weekly report" from:team@company.com`

### Environment Verification
- Verify `~/.config/giztui/` directory permissions
- Test database creation/migration permissions
- Confirm keyboard shortcuts not conflicting with system shortcuts
- **Verify theme system integration and consistent styling**
- **Test focus management and border highlighting**
- **Verify UI pattern consistency with existing pickers and panels**

## Test Scenarios

### Test Case: Basic Save Query Functionality
**Objective**: Verify users can save current searches as bookmarks using bottom-right input panel
**Steps**:
1. Launch Gmail TUI and perform search: `:search has:attachment`
2. Press `Z` key while search results are displayed
3. **UI Verification**: Verify bottom-right input panel appears (not modal)
   - Panel should use theme colors (not generic colors)
   - Border should be highlighted when focused
   - Query preview should be visible in panel body
4. **Input Field Verification**: 
   - Name input field should have same styling as search inputs (component background/text colors)
   - Auto-generated name should be pre-filled
   - Placeholder text should guide user
5. Modify name to "Attachment Search" and press Enter
6. Verify success message appears and panel closes
7. Verify focus returns to message list

**Expected Result**: 
- Query saved successfully with Obsidian-style bottom-right panel UI
- Consistent input field styling matching search inputs
- Proper focus management and border highlighting
- Query preview visible in panel
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Query Detection Enhancement
**Objective**: Verify improved query detection from different search result states
**Steps**:
1. **Test Initial Search State**: 
   - Perform search: `:search has:attachment`
   - While "🔍 Searching: has:attachment" title shows, press `Z`
   - Verify query "has:attachment" is detected correctly
2. **Test Completed Search State**:
   - Wait for search to complete: "🔍 Search Results (10) — has:attachment"
   - Press `Z` and verify query "has:attachment" is detected
3. **Test Advanced Search**:
   - Perform complex search: `:search from:boss@company.com has:attachment after:2024-01-01`
   - Press `Z` and verify complete query is detected (not just partial)
4. **Test Fallback Detection**:
   - Perform search, navigate away, then navigate back to results
   - Press `Z` and verify query still detected from app state

**Expected Result**: Query detection works in all search states, handles complex queries correctly
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Command Save Query with Inline Name
**Objective**: Verify `:save-query` command with provided name
**Steps**:
1. Perform search: `:search from:boss@company.com is:unread`  
2. Execute command: `:save-query Boss Unread Messages`
3. Verify query saved without dialog
4. Check success message displays correct name

**Expected Result**: Query saved directly with provided name in "general" category
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Browse Saved Queries Interface (Prompts-Style Picker)
**Objective**: Verify bookmark browser follows prompts-style picker pattern
**Steps**:
1. Ensure multiple saved queries exist (create 5+ test queries)
2. Press `Q` key to open bookmarks browser
3. **UI Pattern Verification**:
   - Picker should appear in contentSplit area (like prompts/links/attachments)
   - Border should be highlighted when focused
   - Should use theme colors consistently
   - Footer hints should show: "Enter/1-9 to execute | d/D to delete | Esc to cancel"
4. **Input Field Styling**: Verify search input matches other picker styling
5. **Content Verification**: List shows queries with icons, names, usage counts
6. Test search functionality with partial terms
7. Use number keys (1-9) for quick access
8. Test delete functionality with 'd' key
9. Press Esc to close without executing

**Expected Result**: 
- Picker follows prompts-style UI pattern exactly
- Consistent theming and input field styling
- All keyboard shortcuts work as advertised in footer
- Border highlighting works correctly
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Execute Saved Query from Browser
**Objective**: Verify query execution from bookmark browser
**Steps**:
1. Open bookmarks browser with `Q` key
2. Navigate to a saved query using arrow keys
3. Press Enter to execute the query
4. Verify picker closes and search executes
5. Check that usage count increments for the query
6. Verify "last used" timestamp updates

**Expected Result**: Query executes, picker closes, usage statistics update correctly
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Direct Query Execution by Name
**Objective**: Verify `:bookmark` command functionality
**Steps**:
1. Save query named "Important Unread"
2. Execute command: `:bookmark Important Unread`
3. Verify query executes without opening picker
4. Test with partial name: `:bookmark Important`
5. Test tab completion for bookmark names

**Expected Result**: Direct execution works, tab completion suggests saved queries
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Query Search and Filtering  
**Objective**: Verify search within saved queries works correctly
**Steps**:
1. Create queries with diverse names: "Work Emails", "Personal Stuff", "Attachments Only"
2. Open bookmarks browser and enter "work" in search
3. Verify only "Work Emails" appears in results
4. Clear search and enter "email" 
5. Verify "Work Emails" appears (partial match)
6. Test case-insensitive search with "WORK"

**Expected Result**: Real-time filtering works, case-insensitive, partial matches supported
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Category Organization
**Objective**: Verify category-based organization works
**Steps**:
1. Save queries in different categories: "work", "personal", "archive"
2. Use `:bookmarks work` to filter by category
3. Verify only work-related queries appear
4. Test `:bookmarks` with no arguments shows all categories
5. Verify category display in query list

**Expected Result**: Category filtering works correctly, all queries accessible
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Query Management - Edit and Delete
**Objective**: Verify users can manage existing saved queries
**Steps**:
1. Save a test query
2. Open bookmarks browser and navigate to the query
3. Press 'E' key to edit (should show coming soon message)
4. Press 'D' key to delete the query
5. Verify confirmation and deletion succeeds
6. Verify query no longer appears in browser

**Expected Result**: Delete functionality works, edit shows placeholder message
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: No Current Search Error Handling
**Objective**: Verify proper error handling when no search is active
**Steps**:
1. Start Gmail TUI with inbox view (no search active)
2. Press `Z` key to save query
3. Verify appropriate error message
4. Try command `:save-query Test` without active search
5. Verify same error message appears

**Expected Result**: Clear error message: "No current search to save. Perform a search first."
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Empty Bookmarks State
**Objective**: Verify handling when no saved queries exist
**Steps**:
1. Ensure clean database (no saved queries)
2. Press `Q` key to open bookmarks
3. Verify helpful message appears
4. Try `:bookmarks` command with empty state
5. Verify same informative message

**Expected Result**: User-friendly message suggesting how to save first query
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Keyboard Shortcut Configuration
**Objective**: Verify keyboard shortcuts are configurable
**Steps**:
1. Check default config shows `SaveQuery: "Z"` and `QueryBookmarks: "Q"`
2. Modify config to use different keys (e.g., `SaveQuery: "S"`)  
3. Restart Gmail TUI
4. Verify new shortcut works and old shortcut is disabled
5. Test that conflicting keys are properly handled

**Expected Result**: Custom keyboard shortcuts work, no conflicts with existing keys
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Command Autocompletion
**Objective**: Verify tab completion works for all saved query commands
**Steps**:
1. Type `:save` and press Tab - should complete to `:save-query`
2. Type `:book` and press Tab - should complete to `:bookmarks`
3. Save query named "Test Query" 
4. Type `:bookmark Te` and press Tab - should complete query name
5. Test multiple queries with similar names

**Expected Result**: Tab completion works for commands and bookmark names
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Help System Integration
**Objective**: Verify help system documents saved queries feature
**Steps**:
1. Press `?` to open help
2. Search for `/save` in help content
3. Verify saved queries section appears with correct keyboard shortcuts
4. Check command equivalents section lists saved query commands
5. Verify help shows current configured keyboard shortcuts

**Expected Result**: Help system shows saved queries with correct shortcuts and commands
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Database Persistence and Migration
**Objective**: Verify queries persist across application restarts
**Steps**:
1. Save multiple queries with different categories and names
2. Restart Gmail TUI application
3. Open bookmarks browser and verify all queries persist
4. Check usage counts and last used dates are maintained
5. Execute a query and verify statistics update persist

**Expected Result**: All saved queries persist across restarts, statistics maintained
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: UI Pattern Consistency 
**Objective**: Verify saved queries UI matches established patterns exactly
**Steps**:
1. **Save Query Panel Pattern**:
   - Press `Z` to save a query
   - Verify panel appears in bottom-right (contentSplit area) like Obsidian
   - Verify border highlighting matches other focused panels
   - Verify query preview is visible and themed correctly
   - Verify input field styling matches search inputs (not different styling)
2. **Bookmarks Browser Pattern**:
   - Press `Q` to browse queries  
   - Verify picker appears in contentSplit area like prompts/links/attachments
   - Verify 3-line spacing for search input (not 1-line)
   - Verify footer hints are present and consistently styled
   - Verify border highlighting when focused
3. **Input Field Consistency**:
   - Compare save query input field with search inputs in pickers
   - Compare with Obsidian and Slack input fields
   - Verify all use same background, text, and label colors
   - Verify no input fields use accent color for background

**Expected Result**: All UI patterns exactly match established conventions
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Focus Management and Border Highlighting
**Objective**: Verify proper focus indication and ESC key handling
**Steps**:
1. **Save Query Panel**:
   - Press `Z`, verify border highlights immediately
   - Press ESC, verify border returns to normal and focus restored to list
   - Verify no UI deadlocks or hanging
2. **Bookmarks Browser**:
   - Press `Q`, verify border highlights immediately  
   - Navigate between search input and list, verify focus moves correctly
   - Press ESC, verify clean exit and focus restoration
3. **Cross-Panel Testing**:
   - Open save panel, then ESC, then open bookmarks - verify both work
   - Test rapid open/close cycles - verify no memory leaks or focus issues

**Expected Result**: All focus changes show proper border highlighting, clean ESC handling
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Footer Hint Consistency
**Objective**: Verify all picker footers use consistent colors and formatting
**Steps**:
1. Open each picker and compare footer styling:
   - Saved queries (`Q` key): "Enter/1-9 to execute | d/D to delete | Esc to cancel"
   - Attachments picker: "Enter/1-9 to download | Ctrl+S to save as | Esc to cancel"  
   - Themes picker: "Enter: preview | Space: apply | Esc: cancel"
   - Links picker: "Enter/1-9 to open | Ctrl+Y to copy | Esc to cancel"
2. Verify all footers use same color (getFooterColor(), not component colors)
3. Verify consistent formatting and alignment

**Expected Result**: All picker footers have identical styling and color
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Input Field Styling Standardization
**Objective**: Verify all input fields across app have identical styling
**Steps**:
1. **Compare input field styling across panels**:
   - Save query name input (`Z` key)
   - Obsidian pre-message input (obsidian panel)
   - Slack pre-message input (slack panel)  
   - Search inputs in all pickers (saved queries, prompts, attachments, etc.)
2. **Visual verification**:
   - All should use component background color (not accent)
   - All should use component text color
   - All should have consistent focus behavior
   - No input should look different from others

**Expected Result**: All input fields have identical look and feel
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Theme Integration
**Objective**: Verify saved queries picker respects theme settings
**Steps**:
1. Switch to different theme using `:theme set gmail-light`
2. Open saved queries picker with `Q` key
3. Verify colors match current theme
4. Switch to another theme and repeat
5. Check consistency with other pickers (attachments, links)

**Expected Result**: Picker colors consistent with selected theme
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Query Name Validation and Limits
**Objective**: Verify proper validation of query names and descriptions
**Steps**:
1. Try saving query with empty name - should show error
2. Try saving query with very long name (>100 chars) - should handle gracefully
3. Save query with special characters in name
4. Try saving duplicate query names - should update existing
5. Test various category names including special characters

**Expected Result**: Proper validation, helpful error messages, duplicate handling works
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Usage Statistics and Sorting
**Objective**: Verify usage tracking and sorting functionality
**Steps**:
1. Create multiple queries: A, B, C
2. Execute query C several times, B once, leave A unused
3. Open bookmarks browser
4. Verify queries sorted by usage (C, B, A order)
5. Check usage counts display correctly
6. Verify "last used" timestamps are reasonable

**Expected Result**: Queries sorted by usage/recency, statistics accurate
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Complex Query Preservation
**Objective**: Verify complex Gmail search queries are preserved correctly
**Steps**:
1. Create complex query: `from:notifications@github.com has:attachment after:2024-01-01 -label:spam`
2. Save as "GitHub Attachments Recent"
3. Execute the saved query
4. Verify all search parameters are correctly applied
5. Check that special characters and operators are preserved

**Expected Result**: Complex queries execute identically to original search
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Concurrent Usage and Error Recovery
**Objective**: Verify system handles edge cases and concurrent operations
**Steps**:
1. Open multiple command modes and try saving queries simultaneously
2. Delete database file while application running
3. Save query while network is disconnected
4. Try accessing bookmarks with corrupted database
5. Verify graceful error handling and recovery

**Expected Result**: Graceful error handling, no crashes, helpful error messages
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

## Expected Results Summary

### Success Criteria
- ✅ Users can save current searches as bookmarks with custom names/categories
- ✅ Bookmark browser provides intuitive search and filtering
- ✅ Direct query execution works via commands and keyboard shortcuts  
- ✅ Usage statistics accurately track frequency and recency
- ✅ All keyboard shortcuts are configurable and conflict-free
- ✅ Command parity provides equivalent functionality to shortcuts
- ✅ Help system comprehensively documents the feature
- ✅ Database operations are reliable and persist across restarts
- ✅ Theme integration maintains visual consistency
- ✅ Error handling is graceful with clear user feedback

### Performance Criteria
- Bookmark browser opens within 500ms with <100 saved queries
- Query search/filtering provides instant feedback
- Database operations complete within 100ms for typical datasets
- Application startup handles database migration seamlessly
- No memory leaks during repeated bookmark operations

## Regression Tests

### Test Case: Existing Functionality Unaffected
**Objective**: Verify saved queries don't break existing features
**Steps**:
1. Test all existing keyboard shortcuts still work correctly
2. Verify original search functionality (`s` key, `:search` command) unchanged  
3. Test that `Z` and `Q` keys previously unused are now functional
4. Confirm other command autocompletion still works
5. Verify help system navigation and content still correct

**Expected Result**: All existing functionality works identically to before
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

### Test Case: Database Schema Compatibility  
**Objective**: Verify new database schema doesn't affect existing data
**Steps**:
1. Test that existing AI summaries, prompt results still accessible
2. Verify other database operations (Obsidian history, etc.) work
3. Check that database migration doesn't corrupt existing data
4. Confirm application starts normally with existing database

**Expected Result**: All existing database functionality remains intact
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

## Cleanup

### Post-Test Cleanup Steps
1. Remove all test saved queries from database
2. Reset keyboard shortcuts to default configuration  
3. Clear any test messages created during testing
4. Verify application returns to normal operation
5. Document any issues found during test execution

### Environment Reset
1. Restart Gmail TUI to ensure clean state
2. Verify no background database locks remain
3. Check disk space usage returned to normal
4. Confirm database file size is reasonable after cleanup

---

**Note**: This test plan should be executed by a user with Gmail TUI configured and access to a test Gmail account with diverse message types. Each test case should be executed independently and results documented for comprehensive feature validation.