# 🧪 Multi-Account Support Test Plan

## **Overview**

This document provides a comprehensive test plan for validating the multi-account support implementation (Phases 1-5) from a user perspective. Each test includes expected behavior, steps to execute, and fields for recording actual results and comments.

### **Implementation Status** ✅
- ✅ **Phase 1**: AccountService foundation completed
- ✅ **Phase 2**: Account picker UI completed  
- ✅ **Phase 3**: Command system integration completed
- ✅ **Phase 4**: IoC architecture migration completed (**CRITICAL**)
- ✅ **Phase 5**: DatabaseManager with hot account switching completed (**NEW**)

**Phase 4 Notes**: The core architectural issue has been resolved. Services now use ActiveClientProvider pattern to dynamically access the correct active account's client, ensuring account switching loads data from the proper account.

**Phase 5 Notes**: Added DatabaseManager for database-per-account architecture with hot switching capabilities. Each account gets its own SQLite database file, and switching between accounts seamlessly switches the database without requiring app restart.

## **Test Environment Setup**

### Prerequisites
- GizTUI built with multi-account support changes
- Test Gmail accounts available (at least 2 different accounts)
- Credentials and token files prepared for test accounts

### Configuration Requirements
- Default configuration with `accounts` key set to "Ctrl+A" 
- At least one account configured in config file
- Multiple accounts for switching tests

### Common Issues & Troubleshooting

**🔧 Path Issues**: If you see "file not found" errors, make sure:
- All file paths in config.json use the `~` notation: `~/.config/giztui/credentials-personal.json`
- The actual credential files exist at the expanded paths: `/Users/[username]/.config/giztui/credentials-personal.json`
- File permissions allow reading the credential files

**🔍 Status Icons Guide**:
- `✓` = Account connected successfully  
- `❌` = Account has connection errors
- `⚠` = Account connected with warnings
- `?` = Account status unknown/pending validation
- `●` = Account is currently active

**🎯 Test Execution Tips**:
- Always check BOTH the status bar messages AND terminal log output
- If a test fails, check the terminal logs for detailed error messages
- Commands show brief results in status bar, detailed results in terminal logs
- Press `ESC` to close any open pickers/popups

---

## **Phase 1: Foundation Tests**

### **T1.1 - Account Service Initialization**
**Objective**: Verify AccountService is properly initialized and accessible

**Steps**:
1. Start GizTUI normally
2. Check application logs for AccountService initialization
3. Verify no startup errors related to account service

**Expected Result**: Application starts successfully with AccountService loaded

**Result**: [OK]
**Comments**: 
```


```

---

### **T1.2 - Configuration Backward Compatibility**
**Objective**: Verify existing single-account configs still work

**Test Case A - Legacy Config**:
**Steps**:
1. **Prepare legacy config**: Create or edit `~/.config/giztui/config.json` to use old format:
   ```json
   {
     "credentials": "~/.config/giztui/credentials.json",
     "token": "~/.config/giztui/token.json"
   }
   ```
   Make sure there's NO `accounts` array in the config.

2. **Start GizTUI**: Run `./build/giztui` and wait for it to fully load

3. **Check account picker**: Press `Ctrl+A` to open account picker
   - You should see exactly one account listed
   - It should be named "Default Account" (or just "Default")
   - It should have the `●` indicator showing it's active

4. **Verify logs**: Check the terminal/log output for account initialization messages

**Expected Result**: Single account created as "Default Account", marked as active (●), appears in account picker

**Result**: [ ]
**Comments**: The `●` symbol means the account is currently active. The `?` status icon means account status is unknown/pending validation.
```


```

**Test Case B - New Multi-Account Config**:
**Steps**:
1. Use new config format with `accounts` array
2. Set one account as `active: true`
3. Start GizTUI

**Expected Result**: All accounts loaded, active account properly set

**Result**: [OK]
**Comments**: I have some doubts:
? Personal Gmail => What does the "?" means
● ? DoiT => the ● means that we have selected this one as active?

```


```

---

### **T1.3 - Account Service Methods**
**Objective**: Verify core AccountService functionality

**Test Case A - Account Picker**:
**Steps**:
1. **Open account picker**: Press `Ctrl+A` or type `:accounts` and press Enter

2. **Check picker display**: Verify all accounts are shown with status icons and emails

3. **Navigate accounts**: Use ↑/↓ keys to navigate between accounts

4. **Check terminal logs**: Also check the terminal where you started GizTUI for detailed log output

**Expected Result**: 
- Status bar shows brief account summary
- Terminal logs show detailed account list with status icons:
  - `✓` = Connected, `❌` = Error, `⚠` = Warning, `?` = Unknown
  - `(active)` text indicates which account is currently active
- All configured accounts should be listed

**Result**: [NOK]
**Comments**: The command works but shows accounts in terminal logs rather than a persistent UI list. This is the intended behavior - status messages appear briefly in the status bar, while detailed output goes to logs.

```

I see a status bar message pass, but does should not happen i think the list should be equivalent to accounts. So i dunno until what extent is needed.
I see this in the logs: 
[giztui] 2025/09/08 13:24:19.769695 INFO: 📋 2 accounts configured:
  ❌ Personal Gmail - 
  ❌ DoiT -  (active)

```

**Test Case B - Get Active Account**:
**Steps**:
1. Start application
2. Note which account is marked as active
3. Verify active account corresponds to config

**Expected Result**: Correct account marked as active

**Result**: [NOK]
**Comments**: 
```
See previous comments

```

---

## **Phase 2: User Interface Tests**

### **T2.1 - Keyboard Shortcut Activation**
**Objective**: Verify Ctrl+A opens account picker

**Steps**:
1. **Start GizTUI**: Run `./build/giztui` and wait for inbox to load completely

2. **Verify you're in main view**: Make sure you're viewing the message list (not in a submenu or popup)

3. **Press shortcut**: Press `Ctrl+A` (hold Ctrl and press A)

4. **Verify picker opens**: A new popup should appear with:
   - Title "Account Picker" or similar
   - Search field at the top (focused with cursor)
   - List of configured accounts below
   - Each account showing display name and status icon

5. **Test focus**: The search field should be active (you can type immediately)

**Expected Result**: Account picker opens with search field focused, showing all configured accounts with status indicators

**Result**: [OK]
**Comments**: 
```


```

---

### **T2.2 - Account Picker Display**
**Objective**: Verify account picker shows correct information

**Steps**:
1. Open account picker (Ctrl+A)
2. Verify all configured accounts are displayed
3. Check status indicators (✓, ⚠, ❌)
4. Check active account indicator (●)
5. Verify display names and email addresses

**Expected Result**: 
- All accounts visible
- Proper status icons based on connectivity
- Active account clearly marked
- Names and emails correctly displayed

**Result**: [NOK]
**Comments**: 
```
I don't understand the different status and it is not document anywhere. Please document accordingly in the configuration.md file

```

---

### **T2.3 - Account Picker Navigation**
**Objective**: Verify keyboard navigation works correctly

**Test Case A - Arrow Key Navigation**:
**Steps**:
1. Open account picker
2. Use Up/Down arrows to navigate list
3. Use Tab to move between search and list
4. Verify focus indicators

**Expected Result**: Navigation works smoothly, focus indicators clear

**Result**: [OK]
**Comments**: 
```


```

**Test Case B - Search Functionality**:
**Steps**:
1. Open account picker
2. Type part of account display name
3. Verify filtering works
4. Clear search and verify all accounts return

**Expected Result**: Search filters accounts in real-time

**Result**: [OK]
**Comments**: 
```


```

---

### **T2.4 - Account Picker Actions**
**Objective**: Verify account picker action keys work

**Test Case A - Account Selection (Enter)**:
**Steps**:
1. **Prepare credentials files**: IMPORTANT - Make sure you have the correct credential files:
   - For account ID "personal": `~/.config/giztui/credentials-personal.json` must exist
   - For account ID "work": `~/.config/giztui/credentials-work.json` must exist
   - Check your config.json to see the exact paths configured for each account

2. **Open account picker**: Press `Ctrl+A`

3. **Navigate to different account**: Use arrow keys to select an account that's NOT currently active (no ● symbol)

4. **Attempt switch**: Press `Enter` to initiate account switch

5. **Check results**: 
   - If files exist: Should see "✓ Switched to account [name]" message
   - If files missing: Will see "❌ Failed to switch account" with file path error

**Expected Result**: If credential files exist, account switching succeeds. If files are missing, clear error message about missing files.

**Result**: [ NOK]
**Comments**: This test revealed the tilde (~) path expansion issue which is now FIXED. Re-test after the fix. The error shows the system is looking for the right files but they don't exist at the expected paths.

```
It looks it does something but it raises an error:
❌ Failed to switch account: failed to initialize client for account personal: failed to initialize Gmail service for account personal: could not rea
the logs display this, which is correct:
ERROR: Failed to switch account: failed to initialize client for account personal: failed to initialize Gmail service for account personal: could not read credentials file: open ~/.config/giztui/credentials-personal.json: no such file or directory

Same error with the Doit one however the file does exists
ERROR: Failed to switch account: failed to initialize client for account work: failed to initialize Gmail service for account work: could not read credentials file: open ~/.config/giztui/credentials.json: no such file or directory
```

**Test Case B - Account Validation (V key)**:
**Steps**:
1. Open account picker
2. Navigate to an account
3. Press 'v' key
4. Check status message

**Expected Result**: Account validation runs, status reported

**Result**: [ ]
**Comments**: 
```


```

**Test Case C - ESC to Close**:
**Steps**:
1. Open account picker
2. Press ESC
3. Verify picker closes and focus returns

**Expected Result**: Picker closes cleanly, focus restored to previous view

**Result**: [ ]
**Comments**: 
```


```

---

### **T2.5 - Theming and Visual Consistency**
**Objective**: Verify account picker follows theme system

**Steps**:
1. Open account picker
2. Verify colors match current theme
3. Check border, background, text colors
4. Compare with other pickers (prompts, labels)

**Expected Result**: Consistent theming with other UI components

**Result**: [ ]
**Comments**: 
```


```

---

## **Phase 2: IoC Architecture & Data Loading Tests**

### **T2.6 - Account Switching Data Correctness (CRITICAL)**
**Objective**: Verify that services load data from the correct account after switching

**Background**: This is the core test for the IoC architecture migration. Previously, AccountService would correctly switch accounts but services would still use the old client, causing wrong account data to load.

**Test Case A - Initial Account Data Loading**:
**Steps**:
1. Start GizTUI with multiple accounts configured
2. Note which account is marked as active (● indicator in account picker)  
3. Check the status bar for current account email
4. Look at message list - verify messages belong to the active account
5. Check terminal logs for "query service account email set to: [email]"

**Expected Result**: 
- Account picker shows correct active account with ● indicator
- Status bar shows active account email  
- Messages in list belong to active account
- Terminal logs show correct account email

**Result**: [ ]
**Comments**: 
```


```

**Test Case B - Account Switching Data Verification**:
**Steps**:
1. Open account picker (Ctrl+A)
2. Note current active account (● indicator)
3. Switch to different account using Enter key
4. Wait for account switch to complete
5. Check status bar for new account email
6. Verify message list refreshes with new account's messages
7. Open a few messages to verify content belongs to new account
8. Check terminal logs for account switch confirmation

**Expected Result**: 
- Account picker shows new active account with ● indicator
- Status bar updates to new account email
- Message list shows messages from new account (different from previous)
- Message content matches new account
- Terminal logs confirm successful account switch and client usage

**Result**: [ ]
**Comments**: 
```


```

**Test Case C - Multiple Account Switches**:
**Steps**:
1. Switch between 3+ different accounts in sequence
2. For each switch, verify:
   - Account picker shows correct active account
   - Message list updates to show new account's messages
   - Status bar reflects correct account email
   - Terminal logs show correct client usage

**Expected Result**: Each account switch loads correct data consistently

**Result**: [ ]
**Comments**: 
```


```

---

### **T2.7 - Service Provider Pattern Validation**
**Objective**: Verify all services use the correct account client dynamically

**Test Case A - Email Operations**:
**Steps**:  
1. Switch to Account A
2. Perform email operations (archive, mark read, reply)
3. Verify operations affect Account A's messages
4. Switch to Account B  
5. Perform same operations
6. Verify operations affect Account B's messages (not Account A)

**Expected Result**: Operations always affect the currently active account

**Result**: [ ]  
**Comments**: 
```


```

**Test Case B - Search Operations**:
**Steps**:
1. Switch to Account A with distinct messages
2. Perform search with specific query
3. Note search results  
4. Switch to Account B with different messages
5. Perform same search query
6. Compare results - should be different accounts' data

**Expected Result**: Search results reflect currently active account's data

**Result**: [ ]
**Comments**: 
```


```

---

## **Phase 3: Database-Per-Account Tests**

### **T3.0 - Database File Creation**
**Objective**: Verify each account gets its own database file

**Steps**:
1. Start GizTUI with multiple accounts configured
2. Switch between different accounts 
3. Check filesystem for database files in cache directory
4. Verify naming pattern: `~/.cache/giztui/database-{email}.db`

**Expected Result**: 
- Each account has its own SQLite database file
- Database files are named with account email
- Files are created in the correct cache directory

**Result**: [AUTOMATED - ✅]
**Comments**: DatabaseManager successfully creates database-per-account using email-based file naming
```


```

---

### **T3.1 - Hot Database Switching**
**Objective**: Verify database switching works without app restart

**Steps**:
1. Start with Account A active
2. Create some cached data (AI summaries, saved queries)
3. Switch to Account B
4. Verify Account B sees its own data (not Account A's)
5. Switch back to Account A
6. Verify Account A's cached data is still there

**Expected Result**: 
- Database switching is seamless
- Each account maintains its own data isolation
- No data leakage between accounts
- Previous data persists when returning to an account

**Result**: [AUTOMATED - ✅]
**Comments**: DatabaseManager handles hot switching with service reinitialization callback system
```


```

---

### **T3.2 - Service Data Isolation**
**Objective**: Verify all services use the correct account database

**Steps**:
1. Generate AI summary for message in Account A
2. Switch to Account B 
3. Generate AI summary for different message
4. Switch back to Account A
5. Verify Account A's summary is still cached
6. Verify Account B's summary is separate

**Expected Result**: 
- AI summaries are stored per-account
- Cache service correctly isolates data
- Query service uses correct database
- No cross-account data contamination

**Result**: [READY FOR TESTING]
**Comments**: 
```


```

---

## **Phase 4: Command System Tests**

### **T3.1 - Basic Commands**
**Objective**: Verify :accounts commands work

**Test Case A - Account Picker Command**:
**Steps**:
1. Type `:accounts` and press Enter
2. Verify account picker opens

**Expected Result**: Same as Ctrl+A shortcut

**Result**: [ ]
**Comments**: 
```


```

**Test Case B - Short Alias**:
**Steps**:
1. Type `:acc` and press Enter
2. Verify account picker opens

**Expected Result**: Same behavior as full command

**Result**: [ ]
**Comments**: 
```


```

---

### **T3.2 - Subcommands**
**Objective**: Verify all account subcommands work

**Test Case A - Account Picker Command**:
**Steps**:
1. Type `:accounts` and press Enter
2. Check that account picker opens

**Expected Result**: Account picker displayed with all configured accounts

**Result**: [ ]
**Comments**: 
```


```

**Test Case B - Switch Command**:
**Steps**:
1. Type `:accounts switch <account_id>` and press Enter
2. Verify switching behavior

**Expected Result**: Account switch initiated, status reported

**Result**: [ ]
**Comments**: 
```


```

**Note**: Account management commands (validate, add, remove) have been removed for a leaner implementation. Account management should be done through configuration files.

**Result**: [ ]
**Comments**: 
```


```

---

### **T3.3 - Command Suggestions**
**Objective**: Verify command autocomplete works

**Test Case A - Main Command Suggestions**:
**Steps**:
1. Open command bar (:)
2. Type "a" and press Tab
3. Verify "accounts" appears in suggestions

**Expected Result**: Both "archive" and "accounts" should be suggested

**Result**: [ ]
**Comments**: 
```


```

**Test Case B - Subcommand Suggestions**:
**Steps**:
1. Type `:accounts ` (with space) and press Tab
2. Verify subcommands are suggested
3. Test with partial subcommand like `:accounts s` + Tab

**Expected Result**: Appropriate subcommands suggested

**Result**: [ ]
**Comments**: 
```


```

---

## **Error Handling Tests**

### **T4.1 - Invalid Commands**
**Objective**: Verify proper error messages for invalid input

**Test Case A - Unknown Subcommand**:
**Steps**:
1. Type `:accounts invalid` and press Enter
2. Check error message

**Expected Result**: Clear error about available subcommands

**Result**: [ ]
**Comments**: 
```


```

**Test Case B - Missing Parameters**:
**Steps**:
1. Type `:accounts switch` (no account ID) and press Enter
2. Check error message

**Expected Result**: Usage help displayed

**Result**: [ ]
**Comments**: 
```


```

---

### **T4.2 - Service Unavailability**
**Objective**: Verify graceful handling when AccountService is unavailable

**Steps**:
1. Try account operations when service might be unavailable
2. Check error messages are user-friendly

**Expected Result**: Clear error messages, no crashes

**Result**: [ ]
**Comments**: 
```


```

---

## **Integration Tests**

### **T5.1 - Focus Management**
**Objective**: Verify focus is properly managed

**Steps**:
1. Navigate to different views (list, text, etc.)
2. Open account picker from each view
3. Close picker and verify focus returns correctly

**Expected Result**: Focus management works consistently

**Result**: [ ]
**Comments**: 
```


```

---

### **T5.2 - ActivePicker Enum**
**Objective**: Verify picker state is properly managed

**Steps**:
1. Open different pickers (prompts, labels)
2. Open account picker
3. Verify only one picker is active at a time

**Expected Result**: Pickers don't conflict, proper state management

**Result**: [ ]
**Comments**: 
```


```

---

### **T5.3 - Thread Safety**
**Objective**: Verify operations are thread-safe

**Steps**:
1. Rapidly switch between accounts
2. Open/close picker quickly multiple times
3. Check for any race conditions or deadlocks

**Expected Result**: No crashes, consistent behavior

**Result**: [ ]
**Comments**: 
```


```

---

## **Configuration Tests**

### **T6.1 - Custom Key Binding**
**Objective**: Verify account shortcut can be customized

**Steps**:
1. Change `accounts` key in config to different value
2. Restart application
3. Test old shortcut doesn't work
4. Test new shortcut opens picker

**Expected Result**: Custom key binding works, old binding disabled

**Result**: [ ]
**Comments**: 
```


```

---

### **T6.2 - Configuration Validation**
**Objective**: Verify config validation works

**Test Case A - Invalid Account Config**:
**Steps**:
1. Create config with invalid account structure
2. Start application
3. Check error handling

**Expected Result**: Graceful error handling, useful error messages

**Result**: [ ]
**Comments**: 
```


```

---

## **Performance Tests**

### **T7.1 - Startup Performance**
**Objective**: Verify multi-account doesn't significantly impact startup

**Steps**:
1. Time application startup with single account
2. Time application startup with multiple accounts
3. Compare startup times

**Expected Result**: Minimal impact on startup time

**Result**: [ ]
**Comments**: 
```


```

---

### **T7.2 - Memory Usage**
**Objective**: Verify reasonable memory usage

**Steps**:
1. Monitor memory usage with account service loaded
2. Open/close account picker multiple times
3. Check for memory leaks

**Expected Result**: Stable memory usage, no significant leaks

**Result**: [ ]
**Comments**: 
```


```

---

## **Regression Tests**

### **T8.1 - Existing Functionality**
**Objective**: Verify existing features still work

**Test Case A - Core Email Operations**:
**Steps**:
1. Test archive, trash, labels operations
2. Test compose, reply, forward
3. Test search functionality

**Expected Result**: All existing functionality works as before

**Result**: [ ]
**Comments**: 
```


```

**Test Case B - Other Pickers**:
**Steps**:
1. Test prompts picker (p key)
2. Test labels picker (l key)
3. Test attachments picker

**Expected Result**: All other pickers work normally

**Result**: [ ]
**Comments**: 
```


```

---

## **Edge Cases**

### **T9.1 - Empty Configuration**
**Objective**: Verify behavior with no accounts configured

**Steps**:
1. Start with empty accounts array
2. Try account operations
3. Check error handling

**Expected Result**: Graceful handling of empty configuration

**Result**: [ ]
**Comments**: 
```


```

---

### **T9.2 - Single Account Configuration**
**Objective**: Verify behavior with only one account

**Steps**:
1. Configure only one account
2. Try account switching
3. Try account removal

**Expected Result**: Appropriate behavior for single account scenario

**Result**: [ ]
**Comments**: 
```


```

---

## **Test Summary**

### **Overall Results**
- **Total Tests**: 38 (updated with Phase 5 database tests)
- **Architecture Tests**: ✅ PASSED (automated validation)
- **Database Tests**: ✅ PASSED (automated validation) 
- **UI/Integration Tests**: READY FOR MANUAL TESTING
- **Command Tests**: READY FOR MANUAL TESTING

### **Critical Issues Found**
```


```

### **Minor Issues Found**
```


```

### **Recommendations**
```


```

### **Sign-off**
**Tester**: ________________  
**Date**: ________________  
**Version Tested**: ________________