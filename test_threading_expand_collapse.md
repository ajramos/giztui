# Threading Expand/Collapse Fix Test Plan

## Issue Fixed
**Problem**: Thread expand/collapse showed "thread collapsed" message but thread remained visually expanded, and pressing Enter again duplicated thread messages.

**Root Cause**: State synchronization mismatch between ThreadService (database/memory state) and UI display state, plus incomplete row removal logic in collapse operations.

## Solution Applied
1. **State validation**: Added `checkUIThreadExpanded()` to validate UI state vs service state
2. **Synchronization logic**: Detect and fix state mismatches before expand/collapse operations  
3. **Fixed collapse detection**: Corrected logic to identify expanded messages using ThreadId metadata instead of empty IDs
4. **Improved collapse removal**: Enhanced `collapseThreadMessages()` with proper row identification and safer removal
5. **Comprehensive logging**: Added detailed logging throughout threading operations for debugging

## Files Modified
- `internal/tui/threads.go`: Enhanced threading state management and collapse logic
- Added `checkUIThreadExpanded()` helper function
- Improved `collapseThreadMessages()` with safer row removal
- Enhanced `ExpandThread()` with state validation
- Added state validation to `expandThreadAsync()`

## Key Improvements

### 1. State Synchronization
```go
// Check for state mismatch and synchronize
if isExpanded != uiHasExpandedMessages {
    if isExpanded && !uiHasExpandedMessages {
        // Force expand UI to match service state
        go a.expandThreadAsync(threadID, true)
        return nil
    } else if !isExpanded && uiHasExpandedMessages {
        // Force collapse UI to match service state  
        go a.expandThreadAsync(threadID, false)
        return nil
    }
}
```

### 2. Fixed Expanded Message Detection
- Corrected logic to identify expanded messages using `ThreadId` metadata instead of empty IDs
- Expanded messages have their actual message IDs but belong to the same thread
- Stops detection when hitting messages from different threads

### 3. Improved Collapse Logic
- Validates thread row index and ID before proceeding
- Properly identifies expanded messages using thread metadata
- Removes rows in reverse order using existing `removeTableRow()` function
- Handles mutex locking correctly to prevent deadlocks
- Comprehensive validation and logging

### 4. UI State Validation  
- `checkUIThreadExpanded()` verifies if thread has expanded messages in UI
- Uses ThreadId metadata to properly detect expanded messages
- Provides accurate UI state for synchronization decisions

## Test Scenarios

### Manual Testing Required

#### 1. Basic Expand/Collapse Test
- Enter threading mode (`T` key)
- Select a multi-message thread (shows ▶️)
- Press Enter to expand - should show ▼️ and expanded messages
- Press Enter again to collapse - should show ▶️ and remove expanded messages
- Verify "Thread collapsed" message matches actual UI state

#### 2. State Mismatch Recovery Test
- Expand a thread, then artificially create state mismatch (if possible through direct database manipulation)
- Press Enter on the thread
- System should detect mismatch and synchronize UI to service state
- Check logs for "STATE MISMATCH detected" messages

#### 3. Duplicate Prevention Test
- Expand a thread
- If collapse fails (UI remains expanded), press Enter again
- Should NOT duplicate messages - should properly collapse
- Verify no double messages appear in thread

#### 4. Multiple Threads Test
- Expand multiple different threads
- Collapse them in different orders
- Verify each collapse only affects its own thread messages
- Check that other expanded threads remain properly expanded

#### 5. Edge Case Testing
- Very large threads (>10 messages)
- Single-message threads (should not expand)
- Threads with loading errors
- Rapid expand/collapse operations

### Expected Results
- ✅ "Thread collapsed" message only appears when thread actually collapses
- ✅ No message duplication on repeated Enter presses
- ✅ UI display matches ThreadService state consistently
- ✅ Robust error handling for edge cases
- ✅ Clean logging shows state transitions clearly

### Log Analysis
Look for these log patterns:
```
ExpandThread: threadID=xxx, serviceExpanded=false, uiExpanded=true
ExpandThread: STATE MISMATCH detected! Service says false, UI shows true
collapseThreadMessages: removing N rows: [indices]
collapseThreadMessages: collapse complete, final table rows: X, ids length: Y
```

### Automated Test Coverage
- Build verification: `make build` passes
- Unit test verification: `make test-unit` passes  
- No regressions in existing functionality

## Success Criteria
1. ✅ Threads collapse properly when "thread collapsed" message appears
2. ✅ No message duplication occurs
3. ✅ State synchronization works correctly
4. ✅ Comprehensive logging aids debugging
5. ✅ No performance degradation
6. ✅ All existing threading features work correctly

## Performance Impact
- Minimal: Added state validation checks are O(n) where n = expanded messages per thread
- Improved: Better collapse logic prevents memory leaks from lingering UI elements  
- Enhanced: Detailed logging helps debug future issues quickly

## Future Improvements
- Add automated tests for threading state management
- Consider caching UI state to reduce repeated calculations
- Add visual indicators when state synchronization occurs