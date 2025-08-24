# Test Plan: Message Threading Feature

## Feature Overview
This test plan verifies the complete message threading functionality implementation in Gmail TUI, including threaded conversation display, keyboard shortcuts, command interfaces, AI integration, and configuration options.

## Prerequisites

### Setup Requirements
- Gmail TUI built successfully with latest threading implementation
- Gmail account with conversation threads (replies to emails)
- Test emails with reply chains containing at least 3 messages in conversation
- LLM provider configured (Ollama/Bedrock) for AI summary testing
- SQLite database initialized with threading schema (v7 migration)

### Test Data Preparation
1. **Create test threads via Gmail web interface:**
   - Send email to yourself with subject "Test Thread 1"
   - Reply to create 2-3 message conversation
   - Create additional threads with different message counts
   - Ensure some threads have unread messages
   - Include threads with attachments and different participants

2. **Configuration verification:**
   ```bash
   # Verify config contains threading section
   cat ~/.config/giztui/config.json | grep -A 10 '"threading"'
   ```

## Test Scenarios

### **1. Basic Threading Functionality**

#### **Test 1.1: Thread View Toggle**
**Objective**: Verify switching between flat and threaded view modes
**Steps**:
1. Launch Gmail TUI in default flat mode
2. Verify message list shows individual messages chronologically
3. Press `T` key to toggle to thread mode
4. Verify: Messages group by conversation thread
5. Verify: Thread count badges appear (📧 5)
6. Press `T` again to return to flat mode
7. Verify: Returns to flat chronological view

**Expected Result**: 
- Mode switching works smoothly without errors
- UI updates appropriately for each mode
- Status messages confirm mode changes ("📧 Switched to threaded view", "📄 Switched to flat view")

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 1.2: Thread Expansion/Collapse**
**Objective**: Verify thread expand/collapse functionality
**Steps**:
1. Switch to thread mode (`T` key)
2. Navigate to a collapsed thread (shows ▶️ icon)
3. Press `Enter` to expand thread
4. Verify: Thread expands showing all replies with indentation (├─, └─)
5. Press `Enter` again on expanded thread
6. Verify: Thread collapses to show only root message
7. Check status messages for feedback

**Expected Result**:
- Threads expand/collapse correctly on Enter key
- Visual hierarchy with proper indentation
- Icons update (▶️ ↔ ▼️)
- Status feedback: "▼️ Thread expanded" / "▶️ Thread collapsed"

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 1.3: Bulk Thread Operations**
**Objective**: Verify expand all / collapse all functionality
**Steps**:
1. Switch to thread mode with multiple threads visible
2. Press `E` key (Expand All Threads)
3. Verify: All threads expand simultaneously
4. Press `C` key (Collapse All Threads)  
5. Verify: All threads collapse to root messages
6. Check for proper status messages and progress indicators

**Expected Result**:
- All threads expand/collapse as expected
- Operations complete without hanging or errors
- Success messages: "📤 All threads expanded", "📥 All threads collapsed"

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

### **2. Command Parity Testing**

#### **Test 2.1: Threading Commands**
**Objective**: Verify all threading commands work identically to keyboard shortcuts
**Steps**:
1. Type `:threads` and press Enter
2. Verify: Switches to thread mode (same as `T` key)
3. Type `:flatten` and press Enter  
4. Verify: Switches to flat mode (same as `T` key)
5. Switch to thread mode, type `:expand-all` and press Enter
6. Verify: Expands all threads (same as `E` key)
7. Type `:collapse-all` and press Enter
8. Verify: Collapses all threads (same as `C` key)

**Expected Result**: 
- All commands produce identical results to their keyboard shortcuts
- Command autocompletion works (`:th` → `:threads`)
- Status messages are consistent between commands and shortcuts

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

### **3. AI Integration Testing**

#### **Test 3.1: Thread Summary Generation**
**Objective**: Verify AI-powered thread summaries work correctly
**Prerequisites**: LLM provider configured and available
**Steps**:
1. Switch to thread mode
2. Select a multi-message thread (3+ messages)
3. Press `Shift+T` to generate thread summary
4. Verify: Progress indicator shows "🧠 Generating thread summary..."
5. Wait for summary completion
6. Verify: AI panel displays with coherent conversation overview
7. Try `:thread-summary` command on different thread
8. Verify: Command produces same result as keyboard shortcut

**Expected Result**:
- Summary includes key points from all messages in thread
- AI panel appears with streaming or completed summary
- Success message: "🧠 Thread summary generated (X messages)"
- Cached summaries load instantly on repeat requests

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 3.2: Thread Summary Caching**
**Objective**: Verify thread summary caching functionality
**Steps**:
1. Generate summary for a thread (Test 3.1)
2. Navigate away from thread
3. Return to same thread and request summary again (`Shift+T`)
4. Verify: Summary loads instantly from cache
5. Check status message indicates cached result
6. Test cache across application restarts

**Expected Result**:
- Cached summaries load immediately without LLM call
- Status message: "🧠 Thread summary loaded from cache"
- Cache persists across app sessions

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

### **4. Configuration Testing**

#### **Test 4.1: Custom Threading Shortcuts**  
**Objective**: Verify configurable keyboard shortcuts work correctly
**Steps**:
1. Modify `~/.config/giztui/config.json`:
   ```json
   {
     "keys": {
       "toggle_threading": "X",
       "expand_all_threads": "Y",
       "collapse_all_threads": "Z"
     }
   }
   ```
2. Restart Gmail TUI
3. Press `X` key to toggle threading
4. Press `Y` to expand all threads (in thread mode)
5. Press `Z` to collapse all threads
6. Verify all custom shortcuts work as expected

**Expected Result**:
- Custom shortcuts override defaults
- Threading functionality works with new key bindings
- Help system shows updated shortcuts

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 4.2: Threading Configuration Options**
**Objective**: Verify threading configuration settings work correctly
**Steps**:
1. Test default view setting:
   ```json
   {"threading": {"default_view": "thread"}}
   ```
2. Restart app, verify starts in thread mode
3. Test auto-expand unread:
   ```json
   {"threading": {"auto_expand_unread": true}}
   ```
4. Verify threads with unread messages auto-expand
5. Test thread count display:
   ```json
   {"threading": {"show_thread_count": false}}
   ```
6. Verify thread count badges disappear
7. Test threading disabled:
   ```json
   {"threading": {"enabled": false}}
   ```
8. Verify threading shortcuts show error messages

**Expected Result**:
- All configuration options take effect as documented
- Invalid configurations fail gracefully
- Threading can be completely disabled

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

### **5. Integration Testing**

#### **Test 5.1: Bulk Operations with Threading**
**Objective**: Verify bulk mode works correctly with threaded view
**Steps**:
1. Switch to thread mode
2. Enter bulk mode (`v` or `b` key)
3. Select multiple threads using space bar
4. Apply bulk operations (archive, label, trash) 
5. Verify: All messages in selected threads are affected
6. Test bulk prompt application to entire threads
7. Exit bulk mode and verify threading state preserved

**Expected Result**:
- Bulk operations work with entire thread conversations
- Thread state (expanded/collapsed) preserved during bulk operations
- Progress indicators show correct message counts

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 5.2: Search Integration**
**Objective**: Verify search functionality works with threading
**Steps**:
1. Perform search query that matches thread content
2. Switch to thread mode with search results
3. Verify: Matching threads are displayed appropriately
4. Test local search (`/`) within thread mode
5. Verify: Search highlights work in both modes
6. Test saved queries with threading mode

**Expected Result**:
- Search results display correctly in both flat and thread modes
- Thread grouping preserved in search results
- Local search works within thread content

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

### **6. Error Handling and Edge Cases**

#### **Test 6.1: Network Issues**
**Objective**: Verify graceful handling of network problems
**Steps**:
1. Disconnect network connection
2. Attempt to switch to thread mode
3. Try to expand threads
4. Attempt thread summary generation
5. Reconnect network
6. Verify: Operations recover properly

**Expected Result**:
- Clear error messages via ErrorHandler
- No application crashes or hangs
- Graceful recovery when network restored
- Thread state preserved across network issues

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 6.2: Empty and Single-Message Threads**
**Objective**: Verify handling of edge case threads
**Steps**:
1. Test with single-message threads (no replies)
2. Verify: Display shows appropriate indicators
3. Test expand/collapse on single messages
4. Test thread summary on single messages
5. Verify: No errors or unexpected behavior

**Expected Result**:
- Single-message threads display cleanly
- Operations handle edge cases gracefully
- No crashes or infinite loops

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 6.3: Large Thread Performance**
**Objective**: Verify performance with large conversation threads
**Prerequisites**: Thread with 10+ messages
**Steps**:
1. Navigate to large thread (10+ messages)
2. Expand thread and measure response time
3. Scroll through expanded thread content
4. Generate AI summary for large thread
5. Monitor memory usage and performance
6. Test with multiple large threads expanded

**Expected Result**:
- Responsive performance with large threads
- Smooth scrolling and navigation
- AI summary generation completes within reasonable time
- No memory leaks or performance degradation

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

### **7. Help System and Documentation**

#### **Test 7.1: Help System Updates**
**Objective**: Verify help system shows threading information correctly
**Steps**:
1. Press `?` to open help system
2. Verify: Threading section appears with current mode status
3. Test help search: `/threading`, `/thread`, `/flat`
4. Verify: Threading shortcuts shown with configured keys
5. Check command equivalents section includes threading commands
6. Verify: Threading help only appears when enabled in config

**Expected Result**:
- Help system shows complete threading documentation
- Search finds threading-related content
- Dynamic key display shows user's configured shortcuts
- Conditional display based on threading.enabled setting

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

### **8. Database and State Persistence**

#### **Test 8.1: Thread State Persistence**
**Objective**: Verify thread expand/collapse state persists across sessions
**Steps**:
1. Switch to thread mode
2. Expand several threads and collapse others
3. Close Gmail TUI application completely
4. Restart Gmail TUI
5. Return to thread mode
6. Verify: Thread expansion states are remembered
7. Test with different Gmail accounts (if available)

**Expected Result**:
- Thread states persist correctly across app restarts
- Each account maintains separate thread state
- Database migrations work correctly (v7 schema)

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 8.2: Database Migration**
**Objective**: Verify threading database schema migration works correctly
**Steps**:
1. Start with clean/old database (pre-v7)
2. Launch Gmail TUI with threading implementation
3. Verify: Migration to v7 completes successfully
4. Check database contains threading tables:
   - `thread_state`
   - `thread_cache`
   - `thread_summary_cache`
5. Test threading functionality works with migrated database

**Expected Result**:
- Migration completes without errors
- All threading tables created with correct schema
- No data loss during migration

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

## Regression Testing

### **9. Existing Feature Compatibility**

#### **Test 9.1: Flat Mode Unchanged**
**Objective**: Verify existing flat mode functionality not affected
**Steps**:
1. Ensure threading disabled or in flat mode
2. Test all existing shortcuts and commands
3. Verify: All original functionality works unchanged
4. Test: Message navigation, bulk operations, AI features
5. Verify: No performance regression in flat mode

**Expected Result**: 
- Flat mode functions identically to pre-threading implementation
- No performance impact when threading disabled
- All existing features work without changes

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

#### **Test 9.2: Keyboard Shortcut Conflicts**
**Objective**: Verify new threading shortcuts don't conflict with existing ones
**Steps**:
1. Test each existing keyboard shortcut
2. Verify: No unintended threading actions triggered
3. Test with various configuration combinations
4. Check for shortcut conflicts in help documentation

**Expected Result**: 
- No shortcut conflicts with existing functionality
- Threading shortcuts only active when feature enabled
- Clear conflict resolution in documentation

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

## Performance Testing

### **10. Performance Benchmarks**

#### **Test 10.1: Threading Mode Performance**
**Objective**: Measure performance impact of threading features
**Steps**:
1. Load 100+ messages in flat mode, measure load time
2. Switch to thread mode with same dataset, measure switch time
3. Test thread expansion/collapse response times
4. Compare memory usage: flat vs threaded mode
5. Test AI summary generation performance across multiple threads

**Expected Result**:
- Thread mode load time within 2x of flat mode
- Thread operations complete within 500ms
- Memory usage increase less than 50% over flat mode
- No memory leaks during extended usage

**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]

---

## Cleanup Steps

After completing all tests:

1. **Reset Configuration**: Restore original `config.json`
2. **Database Cleanup**: Clear test thread states if needed
3. **Test Data**: Archive or delete test email threads
4. **Documentation**: Update any findings in threading documentation
5. **Bug Reports**: File issues for any failed test cases

## Test Summary

**Total Test Cases**: 21  
**Passed**: [To be filled]  
**Failed**: [To be filled]  
**Blocked**: [To be filled]  

## Critical Success Criteria

For threading feature to be considered ready for release:

1. ✅ **Core Functionality**: All basic threading operations work correctly
2. ✅ **Command Parity**: All keyboard shortcuts have equivalent commands  
3. ✅ **Configuration**: Threading can be enabled/disabled and customized
4. ✅ **AI Integration**: Thread summaries generate and cache correctly
5. ✅ **Performance**: No significant performance regression
6. ✅ **Regression**: Existing functionality unaffected
7. ✅ **Documentation**: Help system and README fully updated
8. ✅ **Persistence**: Thread states survive app restarts

## Notes

- Test with different terminal sizes and layouts
- Verify with both light and dark themes  
- Test with various LLM providers (Ollama, Bedrock)
- Consider testing with different Gmail account types (personal, workspace)
- Monitor for any deadlocks or UI hanging issues
- Verify ESC key properly cancels all threading operations