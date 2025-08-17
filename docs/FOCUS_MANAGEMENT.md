# 🎯 Focus Management Architecture

## Overview

Gmail TUI uses a **centralized focus management system** that coordinates UI element highlighting, state transitions, and user navigation. Understanding this system is crucial for implementing new UI components consistently.

## Core Focus States

The application recognizes these primary focus states via `a.currentFocus`:

| Focus State | Purpose | UI Behavior |
|-------------|---------|-------------|
| `"text"` | Message content view | Main message reading area |
| `"list"` | Message list | Inbox/message navigation |
| `"labels"` | Side panel pickers | Labels, links, and other modal pickers |
| `"search"` | Search/command input | Search bar and command bar |
| `"summary"` | AI summary panel | AI-generated content display |

## 🏗️ Side Panel Picker Pattern

### **The "labels" Focus Reuse Strategy**

Multiple pickers (labels, links, prompts) **intentionally reuse** `currentFocus = "labels"` because they share identical UI behavior:

```go
// ✅ Correct pattern for all side panel pickers
a.currentFocus = "labels"  // Reuse existing focus infrastructure
a.updateFocusIndicators("labels")
a.labelsVisible = true
```

### **Why This Works**

1. **Shared Layout**: All pickers use the same `a.labelsView` container
2. **Identical Positioning**: Same side panel placement and sizing
3. **Common Interactions**: Same Esc-to-close, arrow navigation, Enter-to-select
4. **Focus Restoration**: All return focus to text view when closed
5. **Visual Consistency**: Same yellow border highlighting behavior

### **Implementation Examples**

```go
// Labels Picker
a.currentFocus = "labels"
a.updateFocusIndicators("labels")

// Link Picker  
a.currentFocus = "labels"  // Reuse - not "links"!
a.updateFocusIndicators("labels")

// Prompt Picker
a.currentFocus = "labels"  // Reuse - not "prompts"!
a.updateFocusIndicators("labels")
```

## 🚨 **Anti-Pattern: Command vs Keyboard Inconsistency**

### **The Slack Widget Issue**

**Problem**: `:slack` command doesn't properly set focus, but `K` key does.

**Root Cause**: Command execution doesn't follow the focus management pattern:

```go
// ❌ Current command pattern (broken focus)
func (a *App) executeSlackCommand(args []string) {
    // Opens widget but doesn't set focus properly
    go a.openSlackForwarding()  // Missing focus management
}

// ✅ Keyboard shortcut pattern (working focus)  
case 'K':
    // Properly manages focus state
    a.currentFocus = "labels"
    a.updateFocusIndicators("labels")
    go a.openSlackForwarding()
```

### **The Fix Pattern**

Commands that open UI panels should follow the same pattern as keyboard shortcuts:

```go
func (a *App) executeSlackCommand(args []string) {
    // Set focus state like the keyboard shortcut
    a.currentFocus = "labels"
    a.updateFocusIndicators("labels")
    go a.openSlackForwarding()
}
```

## 📋 **Best Practices**

### **For New Side Panel Pickers**

1. **Reuse "labels" focus state** - Don't create new focus types
2. **Use shared container** - `a.labelsView` for consistent positioning  
3. **Follow established patterns** - Copy from existing picker implementations
4. **Test both access methods** - Keyboard shortcut AND command should behave identically

### **For Command Implementation**

1. **Mirror keyboard shortcuts** - Commands should replicate exact behavior
2. **Set focus state** - Always call `a.updateFocusIndicators()` 
3. **Handle UI state** - Set visibility flags (`a.labelsVisible = true`)
4. **Ensure restoration** - Focus should return to previous state when closed

### **For Focus Debugging**

1. **Check `a.currentFocus` value** - Should match expected state
2. **Verify `updateFocusIndicators()` calls** - Required for border highlighting
3. **Test both access paths** - Keyboard AND command should work identically
4. **Watch for race conditions** - Focus changes in goroutines need careful ordering

## 🎯 **Architecture Benefits**

### **Consistency**
- **Single source of truth** for focus behavior
- **Uniform user experience** across all pickers
- **Reduced cognitive load** for users

### **Maintainability**  
- **No duplicate focus logic** across components
- **Centralized focus state management**
- **Easier debugging** with known patterns

### **Reliability**
- **Well-tested focus transitions** (reused from labels)
- **Fewer edge cases** to handle
- **Consistent border highlighting** behavior

## 🔧 **Common Issues & Solutions**

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **No border highlighting** | Widget opens but no yellow border | Set `currentFocus = "labels"` + `updateFocusIndicators("labels")` |
| **Command focus broken** | `:command` works differently than `Key` | Mirror keyboard shortcut focus management in command |
| **Focus not restored** | Can't return to previous view | Ensure proper focus restoration in close handlers |
| **Multiple pickers conflict** | UI state confusion between pickers | Use shared `a.labelsVisible` state management |

## 🚀 **Implementation Checklist**

When adding a new side panel picker:

- [ ] Use `currentFocus = "labels"` (not custom focus type)
- [ ] Call `a.updateFocusIndicators("labels")`  
- [ ] Set `a.labelsVisible = true`
- [ ] Use `a.labelsView` container
- [ ] Implement both keyboard shortcut and command
- [ ] Test focus restoration with Esc key
- [ ] Verify border highlighting works
- [ ] Ensure command behavior matches keyboard shortcut

This architecture ensures **consistent, maintainable, and user-friendly** focus management across the entire application.