# 🔄 Undo Functionality Test Plan

## 📋 **Feature Overview**
Testing the comprehensive undo functionality that allows users to reverse email actions (archive, trash, mark read/unread, label operations) using the `U` key or `:undo` command.

## 🎯 **Prerequisites**
1. Gmail TUI application built with undo functionality
2. Valid Gmail account with test messages
3. Terminal with sufficient size (minimum 80x24)
4. Test messages in different states (read/unread, labeled/unlabeled, in inbox/archived)

## 🧪 **Test Scenarios**

### **1. Basic Undo Operations**

#### **Test Case: Archive Undo**
**Objective**: Verify that archiving a message can be undone
**Steps**:
1. Select a message in the inbox
2. Press archive key (default: `e`)
3. Verify message is removed from inbox view
4. Press `U` (undo key)
5. Verify message is restored to inbox
6. Verify success message shows "✅ Unarchived message"

**Expected Result**: Message returns to inbox, undo succeeds
**Status**: [FAIL]

**Comentarios**:
it undoes well but the system reloads the messages from the server, the message should appear back without reloading from the server.

#### **Test Case: Trash Undo**
**Objective**: Verify that trashing a message can be undone
**Steps**:
1. Select a message in the inbox
2. Press trash key (default: `d`)
3. Verify message is moved to trash
4. Press `U` (undo key)
5. Verify message is restored to original location

**Expected Result**: Message restored from trash with all original labels
**Status**: [FAIL]

**Comentarios**:
same as previous

#### **Test Case: Mark Read Undo**
**Objective**: Verify that marking a message as read can be undone
**Steps**:
1. Select an unread message (should show as bold/highlighted)
2. Press toggle read key (default: `t`)
3. Verify message is marked as read (no longer bold)
4. Press `U` (undo key)
5. Verify message is back to unread state

**Expected Result**: Message returns to unread state
**Status**: [PASS]

**Comentarios**:
does not work, it says no action to undo, but since i have a toggle function i wouldnt implement the undo for this non destructive actions.

#### **Test Case: Mark Unread Undo**
**Objective**: Verify that marking a message as unread can be undone
**Steps**:
1. Select a read message (not bold/highlighted)
2. Press toggle read key (default: `t`)
3. Verify message is marked as unread (becomes bold)
4. Press `U` (undo key)
5. Verify message is back to read state

**Expected Result**: Message returns to read state
**Status**: [PASS]

**Comentarios**:
does not work, it says no action to undo, but since i have a toggle function i wouldnt implement the undo for this non destructive actions.->

### **2. Command Parity**

#### **Test Case: :undo Command Equivalent**
**Objective**: Verify that `:undo` command works identically to `U` key
**Steps**:
1. Archive a message using `e`
2. Type `:undo` and press Enter
3. Verify message is restored to inbox
4. Archive same message again using `e`
5. Press `U` key
6. Verify both methods produce identical results

**Expected Result**: Both `:undo` and `U` work identically
**Status**: [FAIL]

**Comentarios**:
it undoes well but the system reloads the messages from the server, the message should appear back without reloading from the server.

#### **Test Case: Command Autocompletion**
**Objective**: Verify that undo command has proper autocompletion
**Steps**:
1. Type `:un` in command mode
2. Press Tab to complete
3. Verify it suggests "undo"
4. Type `:U` and verify it suggests "undo"

**Expected Result**: Autocompletion works for "un", "und", "undo", "U"
**Status**: [PASS]

**Comentarios**:
no comments

### **3. Edge Cases and Error Handling**

#### **Test Case: No Action to Undo**
**Objective**: Verify appropriate feedback when no undo is available
**Steps**:
1. Start fresh session (no previous actions)
2. Press `U` (undo key)
3. Verify appropriate info message is shown
4. Type `:undo` and press Enter
5. Verify same behavior

**Expected Result**: Shows "No action to undo" message
**Status**: [PASS]

**Comentarios**:
however, it says no actio to undo, is the spelling correct??? 

#### **Test Case: Undo After Successful Undo**
**Objective**: Verify that undo history is cleared after successful undo
**Steps**:
1. Archive a message
2. Press `U` to undo (should succeed)
3. Press `U` again immediately
4. Verify "No action to undo" message

**Expected Result**: Second undo shows no action available (single-level undo)
**Status**: [PASS]

**Comentarios**:
no comments

#### **Test Case: Message State Changed Externally**
**Objective**: Test undo behavior when message state changed outside TUI
**Steps**:
1. Archive a message in TUI
2. Using Gmail web interface, manually delete the message
3. In TUI, press `U` to undo
4. Verify appropriate error handling

**Expected Result**: Graceful error handling with informative message
**Status**: [fail]

**Comentarios**:
if i delete from the web ui, then i try to delete from TUI, it works, but when I press U to undo it says the UNDO was correct however the message never comes back after refresh

#### **Test Case: Network Failure During Undo**
**Objective**: Test undo behavior during network issues
**Steps**:
1. Archive a message
2. Disconnect network/internet
3. Press `U` to undo
4. Verify appropriate error handling
5. Reconnect network and retry

**Expected Result**: Error message shown, retry works after reconnection
**Status**: [PASS/FAIL/BLOCKED]

**Comentarios**:
didnt try

### **4. Label Operations**

#### **Test Case: Label Addition Undo**
**Objective**: Verify that adding labels can be undone
**Steps**:
1. Select a message
2. Add a label using label manager
3. Verify label is applied
4. Press `U` to undo
5. Verify label is removed

**Expected Result**: Added label is removed from message
**Status**: [FAIL]

**Comentarios**:
It doesnt work, it says no action to undo

#### **Test Case: Label Removal Undo**
**Objective**: Verify that removing labels can be undone
**Steps**:
1. Select a message with existing labels
2. Remove a label using label manager
3. Verify label is removed
4. Press `U` to undo
5. Verify label is re-added

**Expected Result**: Removed label is restored to message
**Status**: [FAIL]

**Comentarios**:
same as previous 

#### **Test Case: Move Operation Undo**
**Objective**: Verify that move operations (apply label + archive) can be undone
**Steps**:
1. Select a message in inbox
2. Press move key (default: `m`)
3. Select a destination label (e.g., "Work")
4. Verify message is moved (labeled + archived)
5. Press `U` to undo
6. Verify message is restored to inbox without the applied label

**Expected Result**: Message restored to inbox, applied label removed
**Status**: [PASS/FAIL/BLOCKED]

**Comentarios**:
<!-- Add test results here -->

#### **Test Case: Bulk Move Operation Undo**
**Objective**: Verify that bulk move operations can be undone
**Steps**:
1. Enter bulk mode (default: `v`)
2. Select multiple messages (3-5 messages)
3. Press move key to open bulk move panel
4. Select a destination label
5. Verify all messages are moved (labeled + archived)
6. Press `U` to undo
7. Verify all messages restored to inbox without applied labels

**Expected Result**: All moved messages restored to original state
**Status**: [PASS/FAIL/BLOCKED]

**Comentarios**:
<!-- Add test results here -->

#### **Test Case: VIM Range Move Undo (m3m)**
**Objective**: Verify that VIM range move operations can be undone
**Steps**:
1. Position cursor on first message to move
2. Press `m3m` (move 3 messages)
3. Select destination label
4. Verify 3 messages are moved
5. Press `U` to undo
6. Verify all 3 messages restored to inbox

**Expected Result**: All 3 messages restored from single undo action
**Status**: [PASS/FAIL/BLOCKED]

**Comentarios**:
<!-- Add test results here -->

### **5. Bulk Operations**

#### **Test Case: Bulk Archive Undo**
**Objective**: Verify that bulk archive operations can be undone
**Steps**:
1. Enter bulk mode (default: `w`)
2. Select multiple messages (3-5 messages)
3. Archive selected messages
4. Press `U` to undo
5. Verify all messages are restored to inbox

**Expected Result**: All bulk-archived messages restored
**Status**: [FAIL]

**Comentarios**:
It shows a message saying the undo was succesful but i cannot see the messages back in the list until i refresh manually from the server. By the way there is a small cosmetic issue, of emoji duplication, on undoing success it says "✅ 🔄 Undone: Unarchived 2 messages" you dont need the second emoji.

#### **Test Case: Bulk Trash Undo**  
**Objective**: Verify that bulk trash operations can be undone
**Steps**:
1. Enter bulk mode
2. Select multiple messages
3. Trash selected messages
4. Press `U` to undo
5. Verify all messages are restored from trash

**Expected Result**: All bulk-trashed messages restored with original labels
**Status**: [FAIL]

**Comentarios**:
same as previous

### **6. UI and Help Integration**

#### **Test Case: Help System Integration**
**Objective**: Verify undo functionality is documented in help
**Steps**:
1. Press `?` to open help
2. Search for "undo" using `/undo`
3. Verify undo shortcut is listed in MESSAGE BASICS section
4. Verify `:undo` command is listed in COMMAND EQUIVALENTS
5. Verify help text is accurate and clear

**Expected Result**: Undo functionality properly documented and searchable
**Status**: [PASS]

**Comentarios**:
no comments

#### **Test Case: Status Message Integration**
**Objective**: Verify undo operations show appropriate status messages
**Steps**:
1. Archive a message
2. Verify status shows "Message archived (U to undo)"
3. Press `U` to undo
4. Verify success status shows "✅ Unarchived message"
5. Test with trash, read/unread operations

**Expected Result**: Clear, helpful status messages for all operations
**Status**: [FAIL]

**Comentarios**:
messages are similar but on the archival it doesnt mention the undo

### **7. Performance and Threading**

#### **Test Case: Undo Responsiveness**
**Objective**: Verify undo operations are responsive and don't block UI
**Steps**:
1. Archive a message
2. Immediately press `U` multiple times rapidly
3. Verify UI remains responsive
4. Verify only one undo operation occurs
5. Test during high message load scenarios

**Expected Result**: UI remains responsive, no race conditions
**Status**: [PASS]

**Comentarios**:


#### **Test Case: Concurrent Operations**
**Objective**: Test undo with concurrent UI operations
**Steps**:
1. Start archiving multiple messages
2. Press `U` during the operation
3. Verify undo works correctly
4. Test with searches, refreshes during undo

**Expected Result**: Undo works correctly with concurrent operations
**Status**: [PASS]

**Comentarios**:


### **8. Configuration and Customization**

#### **Test Case: Custom Key Binding**
**Objective**: Verify undo works with custom key bindings
**Steps**:
1. Modify config to change undo key from "U" to "Z"
2. Restart application
3. Archive a message
4. Press "Z" to undo
5. Verify custom binding works
6. Verify help shows correct custom key

**Expected Result**: Custom key binding works, help updated accordingly
**Status**: [PASS]

**Comentarios**:
<!-- Añade aquí tus comentarios, observaciones o problemas encontrados -->

### **9. Integration Tests**

#### **Test Case: Multiple Action Sequence**
**Objective**: Test undo with sequence of different actions
**Steps**:
1. Mark message as read
2. Add a label
3. Archive message
4. Press `U` to undo (should undo archive only)
5. Verify only archive is undone, read state and label remain

**Expected Result**: Only last action (archive) is undone
**Status**: [PASS/FAIL/BLOCKED]

**Comentarios**:
<!-- Añade aquí tus comentarios, observaciones o problemas encontrados -->

#### **Test Case: Refresh After Undo**
**Objective**: Verify message list updates correctly after undo
**Steps**:
1. Archive a message
2. Press `U` to undo
3. Press refresh key (default: `R`)
4. Verify message still appears in correct location
5. Verify message state is consistent

**Expected Result**: Message list and states remain consistent after refresh
**Status**: [PASS/FAIL/BLOCKED]

**Comentarios**:
<!-- Añade aquí tus comentarios, observaciones o problemas encontrados -->

## 🔧 **Setup Instructions**
1. Ensure test Gmail account has at least 10 messages in various states
2. Create test labels for label operation testing
3. Configure logging if needed for debugging
4. Verify network connectivity for error scenario testing

## ✅ **Success Criteria**
- All basic undo operations work correctly
- Command parity between `U` key and `:undo` command
- Appropriate error handling and user feedback
- Help system properly documents functionality
- No UI blocking or race conditions
- Bulk operations fully supported
- Custom key bindings work correctly

## 🔍 **Cleanup Steps**
1. Reset test messages to original states
2. Clear any test labels created
3. Restore default configuration
4. Verify no persistent state changes

## 📝 **Notes**
- Test with both light and dark themes
- Test with different terminal sizes
- Document any performance observations
- Report any unexpected behaviors or edge cases
- Verify undo works with all supported Gmail label types (INBOX, UNREAD, IMPORTANT, etc.)

---

**Total Test Cases**: 18
**Priority**: High (core functionality)
**Estimated Testing Time**: 2-3 hours
**Dependencies**: Gmail API connectivity, test messages setup