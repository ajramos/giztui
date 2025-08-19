# Enhanced Content Navigation - Test Plan

This document provides comprehensive test scenarios for the Enhanced Message Content Browsability System implemented in Gmail TUI.

## 🎯 **Test Overview**

The Enhanced Content Navigation System addresses the following user pain points:
- Slow line-by-line navigation in long messages
- Lack of local search within message content
- Need for faster content browsing capabilities

## 📋 **Test Environment Setup**

### Prerequisites
1. Build the application: `make build`
2. Ensure you have messages with various content types:
   - Short messages (< 50 lines)
   - Long messages (> 100 lines)
   - Messages with repeated words/phrases
   - Messages with multiple paragraphs
   - Messages with mixed content (plain text, HTML rendered)

### Configuration Verification
The content navigation shortcuts work **out of the box** with these default key bindings:

**Default Shortcuts (No configuration required):**
- `content_search`: `/` (vim-style search)
- `search_next`: `n` (next match)
- `search_prev`: `N` (previous match) 
- `fast_up`: `ctrl+k` (paragraph up)
- `fast_down`: `ctrl+j` (paragraph down)
- `word_left`: `ctrl+h` (word left)
- `word_right`: `ctrl+l` (word right)
- `goto_top`: `gg` (go to top)
- `goto_bottom`: `G` (go to bottom)

**Optional Customization:**
If you want to customize these shortcuts, add them to your `config.json`:
```json
{
  "KeyBindings": {
    "content_search": "/",
    "search_next": "n", 
    "search_prev": "N",
    "fast_up": "ctrl+k",
    "fast_down": "ctrl+j",
    "word_left": "ctrl+h",
    "word_right": "ctrl+l",
    "goto_top": "gg",
    "goto_bottom": "G"
  }
}
```



## 🧪 **Test Scenarios**

### **1. Basic Search Functionality**

#### Test 1.1: Open Content Search
**Steps:**
1. Select a message with content and press Enter to view message
2. **CRITICAL**: Ensure focus is on message content area (not message list)
   - The message content area should be visually highlighted/focused
   - If focus is on message list, press Tab to cycle to "text" focus
   - You should see the message content area is the active focus
3. Press `/` key

**Expected Result:**
- Command bar opens at bottom with "/" pre-filled (NOT email list search)
- Cursor is positioned after "/" ready for typing search term
- Command bar shows the familiar "🐶>" prompt

**Alternative Method:**
- Press `:` to open command bar manually
- Type "/" followed by your search term

**Common Issue:** If email list search appears instead, focus is not on message content - press Tab to cycle focus.

**Result:**
Comment:

#### Test 1.2: Perform Basic Search
**Steps:**
1. Open content search (/ or :/ )
2. Type a word that exists in the message (e.g., "/email") 
3. Press Enter

**Expected Result:**
- Command bar closes
- Focus remains on message content (not message list)
- All matches are highlighted in yellow in the message content
- Cursor jumps to first match
- Status message shows "Found X matches for 'email'"
- Match counter shows "Match 1/X for 'email'"

**Result:**
Comment:

#### Test 1.3: Search with No Results
**Steps:**
1. Open content search (/)
2. Type a word that doesn't exist (e.g., "xyzabc123")
3. Press Enter

**Expected Result:**
- Command bar closes
- Warning message: "No matches found for 'xyzabc123'"
- No highlighting in content
- Content remains at current position

**Result:**
Comment:

#### Test 1.4: Empty Search Query
**Steps:**
1. Open content search (/)
2. Press Enter without typing anything

**Expected Result:**
- Warning message: "Empty search query"
- Command bar closes
- No changes to content

#### Test 1.5: Cancel Search
**Steps:**
1. Open content search (/)
2. Type some text
3. Press ESC

**Expected Result:**
- Command bar closes immediately
- No search is performed
- Content remains unchanged
- Focus returns to text view

**Result:**
Comment:

---

### **2. Search Navigation**

#### Test 2.1: Navigate to Next Match
**Steps:**
1. Search for a word with multiple occurrences (e.g., "the")
2. Press `n` key multiple times

**Expected Result:**
- Cursor jumps to next match each time
- Match counter updates: "Match 2/X", "Match 3/X", etc.
- When reaching last match, next press wraps to first match
- Content scrolls to keep current match visible

**Result:**
Comment:

#### Test 2.2: Navigate to Previous Match
**Steps:**
1. Search for a word with multiple occurrences
2. Press `n` a few times to move forward
3. Press `N` (Shift+n) multiple times

**Expected Result:**
- Cursor jumps to previous match each time
- Match counter decrements: "Match 2/X", "Match 1/X", etc.
- When reaching first match, next press wraps to last match
- Content scrolls appropriately

**Result:**
Comment:

#### Test 2.3: Navigation Without Active Search
**Steps:**
1. Ensure no active search (press ESC if needed)
2. Press `n` or `N`

**Expected Result:**
- Warning message: "No active search - use / to search"
- No cursor movement
- No changes to content

**Result:**
Comment:

---

### **3. Fast Navigation Modes**

#### Test 3.1: Paragraph Navigation Down
**Steps:**
1. Open a long message with multiple paragraphs
2. Position cursor at top
3. Press `Ctrl+J` multiple times

**Expected Result:**
- Cursor jumps to beginning of next paragraph each time
- Content scrolls to keep cursor visible
- Info message: "Fast navigation down" (briefly)
- Navigation stops at end of content

**Result:**
Comment:

#### Test 3.2: Paragraph Navigation Up
**Steps:**
1. Position cursor near bottom of long message
2. Press `Ctrl+K` multiple times

**Expected Result:**
- Cursor jumps to beginning of previous paragraph each time
- Content scrolls appropriately
- Info message: "Fast navigation up" (briefly)
- Navigation stops at beginning of content

**Result:**
Comment:

#### Test 3.3: Word Navigation Right
**Steps:**
1. Position cursor at beginning of a line with multiple words
2. Press `Ctrl+L` multiple times

**Expected Result:**
- Cursor jumps word by word to the right
- Stops at word boundaries (spaces, punctuation)
- No status messages (word navigation is silent)
- Navigation continues across lines

**Result:**
Comment:

#### Test 3.4: Word Navigation Left
**Steps:**
1. Position cursor at end of a line with multiple words
2. Press `Ctrl+H` multiple times

**Expected Result:**
- Cursor jumps word by word to the left
- Stops at word boundaries
- Navigation continues across lines when needed

**Result:**
Comment:

#### Test 3.5: Go to Top
**Steps:**
1. Scroll to middle or bottom of long message
2. Press `g` twice quickly (`gg`)

**Expected Result:**
- Cursor jumps immediately to top of content
- Content scrolls to show beginning
- Info message: "Top of content" (briefly)

**Result:**
Comment:

#### Test 3.6: Go to Bottom
**Steps:**
1. Position cursor at top or middle of message
2. Press `G` (Shift+g)

**Expected Result:**
- Cursor jumps immediately to bottom of content
- Content scrolls to show end
- Info message: "Bottom of content" (briefly)

**Result:**
Comment:

---

### **4. Search and Navigation Integration**

#### Test 4.1: Search + Navigation Combination
**Steps:**
1. Search for a common word (e.g., "and")
2. Use `n` to move to match #3
3. Press `Ctrl+K` for paragraph navigation
4. Press `n` again

**Expected Result:**
- Search results remain active after paragraph navigation
- `n` continues to work with original search results
- Match counter updates correctly
- Both search and fast navigation work together

**Result:**
Comment:

#### Test 4.2: Clear Search with ESC
**Steps:**
1. Perform a search with results
2. Navigate between matches with `n`/`N`
3. Press ESC

**Expected Result:**
- All yellow highlighting disappears
- Info message: "Search cleared"
- Subsequent `n`/`N` presses show "No active search" warning
- Content remains at current position

**Result:**
Comment:

#### Test 4.3: New Search Replaces Previous
**Steps:**
1. Search for "email" 
2. Navigate to match #2 with `n`
3. Open new search with `/`
4. Search for "message"

**Expected Result:**
- Previous "email" highlights disappear
- New "message" highlights appear
- Cursor moves to first "message" match
- Match counter resets to "Match 1/X for 'message'"

**Result:**
Comment:

---

### **5. Edge Cases and Error Handling**

#### Test 5.1: Search in Empty Message
**Steps:**
1. Navigate to a message with no content or very short content
2. Try to search with `/`

**Expected Result:**
- Warning: "No content to search"
- Search overlay doesn't appear
- No errors or crashes

**Result:**
Comment:

#### Test 5.2: Very Long Search Terms
**Steps:**
1. Open search overlay
2. Type a very long search term (>100 characters)
3. Press Enter

**Expected Result:**
- Search processes normally
- Results display correctly (or "no matches" if term doesn't exist)
- No performance issues or crashes

**Result:**
Comment:

#### Test 5.3: Special Characters in Search
**Steps:**
1. Search for terms with special characters: `@`, `.`, `()`, `[]`, etc.
2. Test email addresses, URLs, file paths

**Expected Result:**
- Search finds exact matches for special characters
- Highlighting works correctly
- No regex interpretation of special characters

**Result:**
Comment:

#### Test 5.4: Case Sensitivity
**Steps:**
1. Search for "Email" (with capital E)
2. Verify it matches "email", "EMAIL", "Email" in content

**Expected Result:**
- Search is case-insensitive by default
- All variations are found and highlighted
- Match counter includes all case variations

**Result:**
Comment:

#### Test 5.5: Navigation at Content Boundaries
**Steps:**
1. Use `Ctrl+K` at very top of message
2. Use `Ctrl+J` at very bottom of message
3. Use word navigation at beginning/end of lines

**Expected Result:**
- No errors when already at boundaries
- Navigation commands handle edge cases gracefully
- No cursor position corruption

**Result:**
Comment:

---

### **6. Performance Testing**

#### Test 6.1: Large Message Search
**Steps:**
1. Find or create a very long message (>1000 lines)
2. Search for a common word that appears many times
3. Navigate through all matches with `n`

**Expected Result:**
- Search completes within 1-2 seconds
- Highlighting appears promptly
- Navigation between matches is smooth
- No memory leaks or performance degradation

**Result:**
Comment:

#### Test 6.2: Rapid Key Presses
**Steps:**
1. Perform search with many results
2. Rapidly press `n` key multiple times
3. Rapidly press `Ctrl+J` multiple times

**Expected Result:**
- Commands process smoothly without lag
- No commands are dropped or duplicated
- Status messages appear appropriately
- No crashes or errors

**Result:**
Comment:

---

### **7. Integration Testing**

#### Test 7.1: Focus Management
**Steps:**
1. Start in message list (`Tab` to "text" focus if needed)
2. Open search overlay with `/`
3. Cancel with ESC
4. Verify focus returns to text view

**Expected Result:**
- Focus transitions work correctly
- Text view remains focused after search operations
- Other UI elements (AI panel, labels) aren't affected

#### Test 7.2: Search During AI Summary
**Steps:**
1. Open AI summary panel
2. Switch focus to text view
3. Perform content search
4. Verify AI panel remains functional

**Expected Result:**
- Search works independently of AI features
- AI panel functionality is not affected
- Both features can be used simultaneously

**Result:**
Comment:

#### Test 7.3: Search with Different Message Types
**Steps:**
1. Test search in plain text messages
2. Test search in HTML messages
3. Test search in messages with attachments
4. Test search in calendar invitations

**Expected Result:**
- Search works consistently across all message types
- Content rendering doesn't interfere with search
- All message formats are searchable

**Result:**
Comment:

---

### **8. Keyboard Shortcut Conflicts**

#### Test 8.1: Verify No Conflicts with Existing Shortcuts
**Steps:**
1. Verify `/`, `n`, `N` don't conflict with existing Gmail TUI shortcuts
2. Test `Ctrl+K`, `Ctrl+J`, `Ctrl+H`, `Ctrl+L` don't break existing functionality
3. Verify `gg` and `G` work only in content, not in message list

**Expected Result:**
- New shortcuts only work when focus is on message content
- Existing shortcuts continue to work in their respective contexts
- No unintended side effects in other UI areas

**Result:**
Comment:

---

## 🐛 **Common Issues and Troubleshooting**

### Issue: Content search doesn't work (opens email list search instead)
**Cause:** The `/` key is being intercepted by the global email list search handler
**Solution:** 
- Ensure focus is on message content (not message list)
- Press Tab to cycle focus to "text" (message content area)
- The message content area should be highlighted/focused
- Now `/` should open command bar with "/" pre-filled

**Alternative:** Use `:` to open command bar manually, then type "/<term>"

**Technical Fix Applied:** Modified global key handler in `keys.go` to check `currentFocus != "text"` before handling `/` for email list search

### Issue: Navigation shortcuts don't work
**Check:**
- Verify `currentFocus = "text"` in the code
- Ensure shortcuts are configured in config.json

### Issue: Search results not highlighted
**Check:**
- Verify tview color tags are working: `[black:yellow:b]text[white:-:-]`
- Check if content has conflicting color tags

### Issue: Performance problems with large messages
**Check:**
- Monitor memory usage during search operations
- Verify search algorithm handles large texts efficiently

---

## ✅ **Test Completion Checklist**

- [ ] All basic search scenarios pass
- [ ] Search navigation works correctly
- [ ] Fast navigation modes function properly
- [ ] Search and navigation integration works
- [ ] Edge cases are handled gracefully
- [ ] Performance is acceptable for large messages
- [ ] Integration with existing features works
- [ ] No keyboard shortcut conflicts exist
- [ ] Error handling provides appropriate feedback
- [ ] UI focus management works correctly

## 📝 **Test Results Template**

```markdown
## Test Results - [Date]

### Environment
- OS: [Operating System]
- Gmail TUI Build: [Version/Commit]
- Test Messages: [Description of test data]

### Results Summary
- Tests Passed: X/Y
- Critical Issues: [Number]
- Minor Issues: [Number]

### Issues Found
1. [Issue Description] - Priority: [High/Medium/Low]
   - Steps to reproduce: [Steps]
   - Expected: [Expected behavior]
   - Actual: [Actual behavior]

### Recommendations
- [Any recommendations for improvements]
```

---

**Note**: This test plan covers the core functionality. Additional testing may be needed based on your specific use cases and message types.