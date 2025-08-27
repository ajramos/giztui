# 🤖 Claude Code Development Guide

This file provides context for Claude Code (AI assistant) when working on Gmail TUI.

## 📝 **Git Commit Guidelines**

When committing changes, **DO NOT** include Claude signatures or co-authored by lines in commit messages. Keep commit messages clean and focused on the actual changes.

## 🏗️ **Mandatory Architecture Patterns**

Claude should **ALWAYS** follow these patterns when developing new features:

### 🎯 **Service-First Development**
- **ALL business logic** must go in `internal/services/`
- **UI components** should only handle presentation and user input
- **Never** put Gmail API calls, LLM calls, or complex logic in TUI components

### 📝 **Required Steps for New Features**
1. **Define Service Interface** in `internal/services/interfaces.go`
2. **Implement Service** in dedicated file (e.g., `my_service.go`)
3. **Add Service to App** struct in `internal/tui/app.go`
4. **Initialize Service** in `initServices()` method
5. **Update GetServices()** method to return new service
6. **UI Integration** - call service methods from UI components

### 🚨 **Error Handling Requirements**
- **ALWAYS** use `app.GetErrorHandler()` for user feedback
- **NEVER** use `fmt.Printf`, `log.Printf`, or direct output for user messages
- **Required methods**: `ShowError()`, `ShowSuccess()`, `ShowWarning()`, `ShowInfo()`

### 🔒 **Thread Safety Requirements**
- **ALWAYS** use accessor methods: `GetCurrentView()`, `SetCurrentMessageID()`, etc.
- **NEVER** access app struct fields directly
- **ALWAYS** use proper mutex protection for new state fields

### ⚡ **ESC Key Handling Requirements**
- **CRITICAL**: **NEVER** use `QueueUpdateDraw()` in ESC handlers or cleanup functions
- **ALWAYS** use synchronous operations for UI cleanup to prevent deadlocks
- **Streaming Cancellation**: Main ESC handler in `keys.go` cancels streaming FIRST, then delegates cleanup
- **Pattern**: ESC handlers should call cleanup functions that do direct UI operations
- **Examples**: `exitBulkMode()`, `hideAIPanel()` use synchronous `split.ResizeItem()`, `SetFocus()`, etc.

#### 🚨 **Anti-Pattern - Causes Hanging:**
```go
// ❌ NEVER DO THIS - Causes UI thread deadlock
func (a *App) badCleanup() {
    a.QueueUpdateDraw(func() {
        // UI operations here
    })
}
```

#### ✅ **Correct Pattern - Works Immediately:**
```go
// ✅ ALWAYS DO THIS - Direct synchronous operations
func (a *App) goodCleanup() {
    // Direct UI operations
    if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
        split.ResizeItem(panel, 0, 0)
    }
    a.SetFocus(a.views["list"])
    a.currentFocus = "list"
}
```

### 📨 **Status Message Best Practices**
- **ALWAYS** use `ErrorHandler` for ALL status operations (progress, success, error, warnings)
- **NEVER** use direct status methods (`setStatusPersistent`, `showStatusMessage`) - these are deprecated
- **Consistent baseline**: ErrorHandler ensures all status messages show with proper app baseline
- **CRITICAL**: **NEVER** wrap ErrorHandler calls in `QueueUpdateDraw()` - ErrorHandler handles UI threading internally

#### ✅ **Correct Status Patterns for Bulk Operations:**
```go
// ✅ Progress updates - called directly from goroutines
for i, item := range items {
    // ErrorHandler handles UI threading internally
    a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Processing %d/%d messages…", i+1, len(items)))
    // ... process item
}

// ✅ Clear progress when done
a.GetErrorHandler().ClearProgress()

// ✅ Final results
if failed == 0 {
    a.GetErrorHandler().ShowSuccess(a.ctx, "✅ All messages processed!")
} else {
    a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("⚠️ %d processed (%d failed)", successful, failed))
}

// ✅ From key handlers - ALWAYS use goroutine to avoid deadlock
func (a *App) handleKey() *tcell.EventKey {
    // Do UI operations first (synchronous)
    a.reformatListItems()
    list.SetSelectedStyle(style)
    
    // Then ErrorHandler calls asynchronously
    go func() {
        a.GetErrorHandler().ShowInfo(a.ctx, "Message")
    }()
    return nil
}

// ✅ From nested goroutines - ALWAYS use separate goroutines for ErrorHandler
go func() {
    // Business logic (label operations, API calls, etc.)
    for i, item := range items {
        // Process item...
        
        // Progress updates asynchronously to avoid deadlock
        go func(idx, total int) {
            a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Processing %d/%d…", idx, total))
        }(i+1, len(items))
    }
    
    // UI updates synchronously
    a.QueueUpdateDraw(func() {
        // Update UI state...
    })
    
    // Final status asynchronously
    go func() {
        a.GetErrorHandler().ClearProgress()
        a.GetErrorHandler().ShowSuccess(a.ctx, "Completed!")
    }()
}()
```

#### ❌ **Dangerous Anti-Patterns - CAUSE DEADLOCKS:**
```go
// ❌ DEADLOCK RISK - Never wrap ErrorHandler in QueueUpdateDraw
a.QueueUpdateDraw(func() {
    a.GetErrorHandler().ShowProgress(a.ctx, "Processing...")  // DEADLOCK!
})

// ❌ DEADLOCK RISK - Never call ErrorHandler from synchronous key handlers
func (a *App) handleKey() *tcell.EventKey {
    a.GetErrorHandler().ShowInfo(a.ctx, "Message")  // DEADLOCK!
    return nil
}

// ❌ DEPRECATED - Inconsistent baseline, direct status methods
a.setStatusPersistent("Processing...")
a.showStatusMessage("Success!")
a.setStatusPersistent("")
```

### 🌊 **Streaming Callback Best Practices**

**CRITICAL**: Streaming callbacks (LLM response handlers) must NEVER use `QueueUpdateDraw()` as this causes deadlocks when ESC is pressed during streaming.

#### ✅ **Correct Streaming Pattern:**
```go
err := streamer.GenerateStream(ctx, prompt, func(token string) {
    // Always check context first
    select {
    case <-ctx.Done():
        return // Exit early if cancelled
    default:
    }
    
    // Build result
    b.WriteString(token)
    currentText := sanitizeForTerminal(b.String())
    
    // CRITICAL: Use direct UI update - NEVER QueueUpdateDraw
    if ctx.Err() == nil && a.aiSummaryView != nil {
        a.aiSummaryView.SetText(currentText) // Direct update
    }
})
```

#### ❌ **Anti-Pattern - CAUSES ESC DEADLOCK:**
```go
// ❌ NEVER DO THIS - Causes ESC hanging during streaming
err := streamer.GenerateStream(ctx, prompt, func(token string) {
    a.QueueUpdateDraw(func() {           // DEADLOCK RISK!
        a.aiSummaryView.SetText(token)   // Queued operation blocks ESC
    })
})
```

#### 🔍 **Why This Causes Deadlocks:**
1. **Streaming callback queues UI operation** via `QueueUpdateDraw()`
2. **User presses ESC** - tries to execute synchronous cleanup
3. **UI thread deadlock** - ESC waits for queued operations, streaming continues
4. **Application hangs** - neither ESC nor streaming can complete

#### 📋 **ErrorHandler Method Guide:**
- `ShowProgress(ctx, msg)` - For ongoing operations (doesn't auto-clear)
- `ClearProgress()` - Clear progress messages
- `ShowSuccess(ctx, msg)` - Success messages (auto-clear after 3s)
- `ShowError(ctx, msg)` - Error messages (auto-clear after 3s)
- `ShowWarning(ctx, msg)` - Warning messages (auto-clear after 3s)
- `ShowInfo(ctx, msg)` - Info messages (auto-clear after 3s)

## 📋 **Code Templates**

### Service Implementation Template
```go
// internal/services/my_service.go
package services

import "context"

type MyService interface {
    DoOperation(ctx context.Context, param string) error
}

type MyServiceImpl struct {
    client *gmail.Client
    config *config.Config
}

func NewMyService(client *gmail.Client, config *config.Config) *MyServiceImpl {
    return &MyServiceImpl{
        client: client,
        config: config,
    }
}

func (s *MyServiceImpl) DoOperation(ctx context.Context, param string) error {
    // Implementation here
    return nil
}
```

### UI Integration Template
```go
// internal/tui/my_component.go
func (a *App) handleNewFeature() error {
    // 1. Get services
    emailService, aiService, labelService, cacheService, repository := a.GetServices()
    
    // 2. Get state thread-safely
    messageID := a.GetCurrentMessageID()
    
    // 3. Call service
    if err := emailService.DoOperation(a.ctx, messageID); err != nil {
        a.GetErrorHandler().ShowError(a.ctx, "Operation failed")
        return err
    }
    
    // 4. Show success
    a.GetErrorHandler().ShowSuccess(a.ctx, "Operation completed")
    return nil
}
```

## 📚 **Help System Maintenance**

**CRITICAL**: The help system must be updated whenever new features are added to ensure users can discover and use all functionality.

### 🎯 **When to Update Help**

**ALWAYS** update help when adding:
- New keyboard shortcuts
- New commands (:command style)
- New features or functionality
- Changes to existing shortcuts
- New configuration options that affect user workflow

### 📍 **Help Content Location**

Help content is generated by `generateHelpText()` in `internal/tui/app.go`. This function builds the entire help display that users see when pressing `?`.

### 🔧 **Help Update Process**

When adding new features, **ALWAYS** follow these steps:

#### **1. Add to Appropriate Section**
Help is organized into logical sections. Add new features to the most relevant section:

- **🚀 GETTING STARTED** - Basic operations for new users
- **📧 MESSAGE BASICS** - Core email operations (reply, compose, archive, etc.)
- **🧭 NAVIGATION & SEARCH** - Finding and browsing messages
- **📖 CONTENT NAVIGATION** - Within-message navigation and search
- **📦 BULK OPERATIONS** - Multi-message operations
- **🤖 AI FEATURES** - LLM-powered functionality
- **⚡ VIM POWER OPERATIONS** - Range operations (s5s, a3a, etc.)
- **🔧 ADDITIONAL FEATURES** - Advanced features (themes, exports, integrations)
- **💻 COMMAND EQUIVALENTS** - :command alternatives

#### **2. Follow Format Conventions**
```go
// Use consistent formatting for help entries
help.WriteString(fmt.Sprintf("    %-8s  🎯  Description of action\n", a.Keys.NewFeature))

// For fixed keys (not configurable)
help.WriteString("    F         📫  Quick search: from current sender\n")

// For VIM range operations
help.WriteString(fmt.Sprintf("    %s3%s       📁  Archive next 3 messages\n", a.Keys.Archive, a.Keys.Archive))
```

#### **3. Use Dynamic Key References**
**ALWAYS** use `a.Keys.*` fields instead of hardcoded keys:
```go
// ✅ CORRECT - Uses configurable key
help.WriteString(fmt.Sprintf("    %-8s  🔄  Refresh messages\n", a.Keys.Refresh))

// ❌ WRONG - Hardcoded key
help.WriteString("    r         🔄  Refresh messages\n")
```

#### **4. Add Command Equivalents**
If the feature has a command equivalent, add it to the COMMAND EQUIVALENTS section:
```go
help.WriteString("    :newfeature   🎯  Same as [key] (new feature)\n")
```

#### **5. Update Status Display**
If adding conditional features, show their status:
```go
// For optional features that can be enabled/disabled
if a.Config.NewFeature.Enabled {
    help.WriteString(fmt.Sprintf("    %-8s  🎯  New feature action\n", a.Keys.NewFeature))
}
```

### 📋 **Help Update Checklist**

When adding a new feature, verify:
- [ ] Added to appropriate help section
- [ ] Uses `a.Keys.*` for configurable shortcuts
- [ ] Follows consistent formatting (icon, spacing, description)
- [ ] Added command equivalent if applicable
- [ ] Conditional display for optional features
- [ ] Tested that `/term` search finds the new feature
- [ ] Verified help content displays correctly
- [ ] **Created test plan file** (see Test Plan Requirements below)

### 🧪 **Testing Help Updates**

After updating help:
1. **Build and run**: Ensure no compilation errors
2. **View help**: Press `?` to view help content
3. **Test search**: Use `/newfeature` to find new additions
4. **Check formatting**: Ensure consistent alignment and spacing
5. **Verify keys**: Test that displayed shortcuts actually work

### 🎯 **Help Content Best Practices**

- **Be Concise**: One line per feature, clear action description
- **Use Emojis**: Visual icons help users scan and categorize features
- **Group Logically**: Related features should be in the same section
- **Show Context**: Include current status (bulk mode, AI availability, etc.)
- **Maintain Order**: Keep similar features together for easy discovery

### 📖 **Help System Architecture**

The help system uses the message content area for display, enabling:
- **Content Search**: `/term` to find specific features
- **Navigation**: `n`/`N` for search results, `g`/`gg`/`G` for scrolling
- **Responsive**: Adapts to terminal width automatically
- **Searchable**: All text is indexed for instant search

Remember: **Users discover features through help**, so incomplete or outdated help directly impacts user experience and feature adoption.

## 🚨 **Anti-Patterns to AVOID**

### ❌ **Never Do This**
```go
// ❌ Business logic in UI
func (a *App) badExample() {
    messages, err := a.Client.GetMessages() // Direct API call in UI
    if err != nil {
        fmt.Printf("Error: %v\n", err)     // Direct output
        a.setStatusPersistent("Error!")     // Deprecated status method
    }
    a.currentMessageID = "new-id"           // Direct field access
    a.showStatusMessage("Done")             // Deprecated status method
}
```

### ✅ **Always Do This**
```go
// ✅ Proper architecture
func (a *App) goodExample() {
    emailService, _, _, _, _ := a.GetServices()
    messageID := a.GetCurrentMessageID()
    
    if err := emailService.LoadMessages(a.ctx); err != nil {
        a.GetErrorHandler().ShowError(a.ctx, "Failed to load messages")
        return
    }
    
    a.SetCurrentMessageID("new-id")
    a.GetErrorHandler().ShowSuccess(a.ctx, "Messages loaded")
}
```

## 📚 **Reference Examples**

Study these existing implementations:
- `internal/services/email_service.go` - Email operations pattern
- `internal/services/ai_service.go` - LLM integration pattern  
- `internal/tui/app.go` - Service integration pattern
- `internal/tui/error_handler.go` - Error handling pattern
- `internal/tui/keys.go` - ESC key handling with streaming cancellation
- `internal/tui/bulk_prompts.go` - Synchronous UI cleanup (`exitBulkMode`, `hideAIPanel`)
- `docs/FOCUS_MANAGEMENT.md` - UI focus patterns and side panel picker architecture

## 🎯 **Development Workflow**

When asked to implement a new feature:

1. **Analyze Requirements** - Identify what services are needed
2. **Check Existing Services** - Reuse if possible, extend if needed
3. **Design Service Interface** - Define clean contracts
4. **Validate with the user the proposed solution**** - Make sure the customer agrees on the approach 
5. **Implement Service** - Business logic only
6. **Integrate with UI** - Presentation logic only
7. **Add Error Handling** - Use ErrorHandler consistently
8. **Ensure Thread Safety** - Use accessor methods
9. **Command Parity** - Add equivalent command for any new keyboard shortcut
10. **Test Integration** - Verify build and functionality

### 🎮 **Command Parity Requirements**

When implementing features with keyboard shortcuts, **ALWAYS** ensure command parity:

#### **Mandatory Pattern:**
- **Every keyboard shortcut MUST have an equivalent command**
- **Commands MUST support bulk mode automatically** 
- **Commands MUST provide short aliases** (e.g., `:archive` and `:a`)
- **Commands MUST work with existing autocompletion**

#### **Implementation Steps:**
1. **Add command case** to `executeCommand()` in `internal/tui/commands.go`
2. **Create execution function** following bulk-aware pattern:
   ```go
   func (a *App) executeMyCommand(args []string) {
       // Check bulk mode and selected messages
       if a.bulkMode && len(a.selected) > 0 {
           go a.myActionBulk()
       } else {
           go a.myAction()
       }
   }
   ```
3. **Add to command suggestions** in `generateCommandSuggestion()`
4. **Update README** with command parity table
5. **Test both keyboard and command interfaces**

#### **Examples:**
```go
// ✅ Correct command parity implementation
case "archive", "a":
    a.executeArchiveCommand(args)
case "trash", "d": 
    a.executeTrashCommand(args)
case "read", "toggle-read", "t":
    a.executeToggleReadCommand(args)
```

#### **Benefits:**
- **Accessibility** - Users can discover functionality through commands
- **Consistency** - Every action has multiple ways to access
- **Bulk support** - Commands automatically detect and respect bulk mode
- **Discoverability** - Tab completion helps users learn available actions

## 🛠️ **Build & Test Commands**
- `make build` - Build the application
- `make test` - Run tests
- `make fmt` - Format code
- `make lint` - Lint code (if configured)

## 🔄 **When Modifying Existing Code**

1. **Identify Architecture Violations** - Look for business logic in UI
2. **Extract to Services** - Move logic to appropriate service
3. **Update UI Integration** - Use service methods
4. **Add Thread Safety** - Use accessor methods
5. **Improve Error Handling** - Use ErrorHandler
6. **Fix ESC Handling** - Replace `QueueUpdateDraw` with synchronous operations

## 🐛 **Recent Debugging & Fixes (August 2025)**

### 🔧 **Config Structure Improvement - Theme Section (August 2025)**
Successfully refactored theme configuration to use more logical nested structure:

#### **Breaking Change - Config Migration Required:**
- **Old structure** (deprecated): 
  ```json
  {
    "layout": {
      "current_theme": "slate-blue",
      "custom_theme_dir": "/path/to/themes"
    }
  }
  ```
- **New structure** (current):
  ```json
  {
    "theme": {
      "current": "slate-blue", 
      "custom_dir": "/path/to/themes"
    }
  }
  ```

#### **Migration Required:**
Users with existing config files need to:
1. Move `layout.current_theme` → `theme.current`
2. Move `layout.custom_theme_dir` → `theme.custom_dir` 
3. Remove theme fields from layout section

#### **Rationale:**
- **Cleaner organization**: Theme settings grouped logically under `theme` object
- **Eliminated redundancy**: No need for `_theme` suffix when already under theme section
- **Better maintainability**: Theme-related config changes isolated from layout settings

#### **Files Updated:**
- `internal/config/config.go` - Added ThemeConfig struct, updated Config struct
- `internal/tui/app.go` - Updated all theme config references
- `examples/*.json` - Updated example configurations
- `README.md` - Updated documentation with new config structure

### 🔧 **Content Navigation Service Nil Pointer Fix (August 2025)**
Successfully resolved critical nil pointer dereference crash in Enhanced Content Navigation system:

#### **Issue Fixed:**
- **Runtime Panic**: `runtime error: invalid memory address or nil pointer dereference` in `EnhancedTextView.performContentSearch`
- **Root Cause**: Service initialization timing issue - `EnhancedTextView` created in `initComponents()` before `initServices()` runs
- **Impact**: Application crash when users tried to use content search functionality

#### **Solution Applied:**
```go
// ❌ Old problematic pattern (caused nil pointer crash)
func NewEnhancedTextView(app *App) *EnhancedTextView {
    enhanced := &EnhancedTextView{
        contentNavService: app.GetContentNavService(), // nil during initComponents()
    }
    return enhanced
}

// ✅ New fixed pattern (lazy initialization)
func NewEnhancedTextView(app *App) *EnhancedTextView {
    enhanced := &EnhancedTextView{
        contentNavService: nil, // Will be lazily initialized
    }
    return enhanced
}

func (e *EnhancedTextView) getContentNavService() services.ContentNavigationService {
    if e.contentNavService == nil {
        e.contentNavService = e.app.GetContentNavService()
    }
    return e.contentNavService
}
```

#### **Key Improvements:**
- **Lazy Initialization**: Service loaded on first use, not during UI component creation
- **Defensive Guards**: All navigation functions check service availability before use
- **Graceful Degradation**: Navigation functions fail silently, search functions show user-friendly errors
- **Threading Safety**: Maintains existing async error handling patterns

#### **Files Modified:**
- `internal/tui/enhanced_text_view.go` - Added lazy initialization and defensive guards
- Fixed all direct `e.contentNavService` calls to use `e.getContentNavService()`
- Added `hasContentNavService()` helper for availability checks

#### **Testing Verified:**
- ✅ Application builds successfully without compilation errors
- ✅ No runtime nil pointer crashes during UI initialization  
- ✅ Content navigation system gracefully handles service unavailability
- ✅ Maintains backward compatibility with existing TUI architecture

### 🔧 **Content Search Focus Restoration Fix (August 2025)**
Successfully resolved focus management issue where content search commands returned focus to message list instead of message content:

#### **Issue Fixed:**
- **Focus Problem**: After using `/term` content search, focus returned to message list instead of message content
- **User Impact**: Pressing `n` (next match) was interpreted as "new message composition" instead of "search next"  
- **Root Cause**: `restoreFocusAfterModal()` always restored focus to list, overriding content search focus needs

#### **Solution Applied:**
```go
// ✅ New focus override system
type App struct {
    // ... existing fields ...
    cmdFocusOverride string  // Override focus restoration (e.g., "text" for content search)
}

func (a *App) executeContentSearch(args []string) {
    // Set focus override before executing search
    a.cmdFocusOverride = "text"
    a.enhancedTextView.performContentSearch(query)
}

func (a *App) restoreFocusAfterModal() {
    // Check for focus override first
    if a.cmdFocusOverride != "" {
        targetFocus := a.cmdFocusOverride
        a.cmdFocusOverride = "" // Clear after use
        // Restore to specified target (e.g., text view)
        // ...
    }
    // Default: restore to list
}
```

#### **Key Improvements:**
- **Contextual Focus**: Commands can specify where focus should return to
- **Clean Architecture**: Override system is self-clearing and doesn't affect other modals
- **User Experience**: Content search now properly keeps focus on message content for follow-up navigation (`n`, `N`, etc.)
- **Backward Compatibility**: Default behavior unchanged for other commands

#### **Files Modified:**
- `internal/tui/app.go` - Added `cmdFocusOverride` field to App struct
- `internal/tui/keys.go` - Enhanced `restoreFocusAfterModal()` with override logic  
- `internal/tui/commands.go` - Set focus override in `executeContentSearch()`

#### **Testing Verified:**
- ✅ Content search (`/term`) returns focus to message content, not message list
- ✅ Navigation keys (`n`, `N`) work correctly after content search
- ✅ Other commands still restore focus to list as expected
- ✅ No side effects on existing modal/command behavior

### 🔧 **Content Navigation Key Handler Fix (August 2025)**
Successfully resolved issue where `n` key was intercepted by global handler instead of EnhancedTextView:

#### **Issue Fixed:**
- **Key Interception**: Global `n` key handler for "compose message" was intercepting the key even when focus was on text
- **User Impact**: After content search, pressing `n` triggered "compose message" instead of "search next match"
- **Root Cause**: Global key handler processed keys before view-specific input capture could handle them

#### **Solution Applied:**
```go
// ❌ Old problematic pattern
case 'n':
    if a.currentFocus == "list" {
        go a.loadMoreMessages()
        return nil
    }
    go a.composeMessage(false)  // Always executed regardless of focus!
    return nil

// ✅ New focus-aware pattern  
case 'n':
    if a.currentFocus == "list" {
        go a.loadMoreMessages()
        return nil
    } else if a.currentFocus != "text" {
        go a.composeMessage(false)  // Only if NOT focused on text
        return nil
    }
    // If focus is on text, let EnhancedTextView handle it
```

#### **Key Improvements:**
- **Focus-Aware Processing**: Global handler checks focus before consuming keys
- **Proper Key Delegation**: When focus is on text, keys pass through to EnhancedTextView
- **Backward Compatibility**: List and other focus contexts unchanged
- **VIM Navigation Support**: `g`, `gg`, `G` keys already properly delegated via existing `handleVimSequence()` focus checks

#### **Files Modified:**
- `internal/tui/keys.go` - Enhanced global `n` key handler with focus-aware logic

#### **Testing Verified:**
- ✅ `n` key works for "search next" when focus is on message content
- ✅ `n` key still works for "compose message" when focus is not on text
- ✅ `N` key works for "search previous" (no conflicts found)
- ✅ VIM keys (`g`, `gg`, `G`) properly delegated to content navigation
- ✅ Ctrl combinations (Ctrl+K/J/H/L) work without conflicts

### 🔧 **EnhancedTextView Focus Routing Fix (August 2025)**
Successfully resolved critical focus routing issue where input capture was not receiving key events:

#### **Issue Fixed:**
- **Focus Mismatch**: `a.views["text"]` stored regular TextView, but input capture was on EnhancedTextView wrapper
- **Key Routing Failure**: When focusing `a.views["text"]`, keys went to TextView without custom input capture
- **User Impact**: Content navigation keys (`n`, `N`, `g`, etc.) never reached the EnhancedTextView handlers

#### **Root Cause Analysis:**
```go
// ❌ Problem: Focus mismatch
enhancedText := NewEnhancedTextView(a)
text := enhancedText.TextView              // Inner TextView
text.SetInputCapture(...)                  // ❌ Wrong! Capture set on inner view
a.views["text"] = text                     // Store inner TextView
a.SetFocus(a.views["text"])               // Focus inner TextView (no input capture!)

// ✅ Solution: Focus the wrapper with input capture
enhancedText.TextView.SetInputCapture(...) // ✅ Correct! Capture on wrapper
a.SetFocus(enhancedText)                   // Focus wrapper with input capture
```

#### **Solution Applied:**
- **Helper Function**: Created `SetTextViewFocus()` that properly focuses EnhancedTextView when available
- **Focus Ring Update**: Updated Tab cycling to use EnhancedTextView for proper input capture  
- **Message Selection**: Fixed Enter key on messages to focus EnhancedTextView, not inner TextView
- **Focus Restoration**: All focus restoration now uses EnhancedTextView for key capture

#### **Key Improvements:**
- **Proper Key Routing**: All content navigation keys now reach EnhancedTextView's input capture
- **Backward Compatibility**: Falls back to regular TextView if EnhancedTextView unavailable
- **Consistent Focus Management**: All text view focus operations use the same helper function
- **Enhanced UX**: Users now get full content navigation when viewing messages

#### **Files Modified:**
- `internal/tui/app.go` - Added `SetTextViewFocus()` helper function
- `internal/tui/keys.go` - Updated focus ring and restoration to use EnhancedTextView
- `internal/tui/messages.go` - Fixed message selection to focus EnhancedTextView

#### **Testing Verified:**
- ✅ EnhancedTextView properly receives focus when viewing messages
- ✅ Content navigation keys (`n`, `N`, `g`, `gg`, `G`) work correctly
- ✅ Tab cycling includes EnhancedTextView with proper input capture
- ✅ Focus restoration after commands focuses EnhancedTextView
- ✅ Backward compatibility maintained if EnhancedTextView unavailable

### 🔧 **Bulk Operations Debugging Session**
Successfully resolved critical issues in bulk operations that were causing hangs and incomplete functionality:

#### **Issues Fixed:**
1. **Bulk Labeling Hang on Filtered Search** - When filtering labels to a single result and pressing Enter, system would hang
2. **Cache Update Bug** - `updateCachedMessageLabels` was calling `updateBaseCachedMessageLabels` inside a loop incorrectly
3. **Mixed Status Handling** - Inconsistent use of `showStatusMessage` vs `GetErrorHandler()` causing UI deadlocks
4. **Missing Bulk Mode Support** - Enter key handler in label search didn't support bulk operations

#### **Root Causes:**
- **Threading Deadlocks**: Nested `QueueUpdateDraw()` calls when `showStatusMessage()` was called from goroutines
- **Missing Logic Paths**: Search/filter Enter key handler only supported single messages, not bulk mode
- **Cache Corruption**: Label cache updates were happening multiple times per message due to loop placement

#### **Solutions Applied:**
```go
// ❌ Old problematic pattern (caused deadlocks)
func (a *App) oldBulkOperation() {
    go func() {
        // ... do work ...
        a.showStatusMessage("Done") // Used QueueUpdateDraw internally
    }()
}

// ✅ New fixed pattern (deadlock-free)
func (a *App) newBulkOperation() {
    go func() {
        // ... do work ...
        go func() {
            a.GetErrorHandler().ShowSuccess(a.ctx, "Done") // Async ErrorHandler
        }()
    }()
}
```

#### **Key Learning:**
- **Never call `showStatusMessage()` from goroutines** - it uses `QueueUpdateDraw` internally
- **Always use `GetErrorHandler()` for status updates** - it's designed for async operations
- **Add bulk mode checks to ALL user interaction handlers** - not just list item callbacks
- **Debug with logging first** - determine exact hang location before fixing

#### **Files Modified:**
- `internal/tui/labels.go` - Fixed Enter key handler to support bulk mode
- `internal/tui/labels.go` - Fixed `updateCachedMessageLabels` cache bug
- `internal/tui/bulk_prompts.go` - Enhanced debug logging
- `internal/tui/keys.go` - Improved ESC key handling consistency

#### **Testing Verified:**
- ✅ Bulk labeling with filtered search works correctly
- ✅ Multiple message selection + label application succeeds
- ✅ No deadlocks when using ErrorHandler for status updates
- ✅ Cache updates happen correctly (once per message)

## 🧪 **Testing Framework Integration**

**MANDATORY**: All new feature development MUST include proper testing using the integrated testing framework.

### 🏗️ **Testing Infrastructure**

The project now includes a comprehensive testing framework with:

#### **Core Components**
- **Test Harness** (`test/helpers/test_harness.go`) - Central testing utility with tcell simulation screen
- **Mock Generation** - Automated mocks for all service interfaces using mockery
- **CI/CD Pipeline** - GitHub Actions workflow for automated testing across platforms
- **Makefile Targets** - Comprehensive test commands for different test types

#### **Available Test Commands**
```bash
# Generate mocks (run first)
make test-mocks

# Run basic tests
make test               # All current working tests
make test-unit          # Service layer unit tests  
make test-tui           # TUI component tests
make test-coverage      # Tests with coverage report

# Advanced testing (when app methods are implemented)
make test-integration   # Integration tests
make test-performance   # Performance benchmarks
make test-all          # Complete test suite
```

#### **Mock Services Available**
All service interfaces have generated mocks:
- `mocks.EmailService` - Email operations
- `mocks.AIService` - AI/LLM operations  
- `mocks.LabelService` - Label management
- `mocks.CacheService` - Caching operations
- `mocks.MessageRepository` - Data access
- `mocks.SearchService` - Search functionality

### 🎯 **Testing Best Practices**

#### **For New Features**
1. **Create Test File** - Add `*_test.go` files alongside implementation
2. **Use Test Harness** - Import `github.com/ajramos/gmail-tui/test/helpers`
3. **Mock Dependencies** - Use generated mocks for service isolation
4. **Test All Paths** - Cover success, failure, and edge cases
5. **Include Benchmarks** - Add performance tests for critical paths

#### **Test Structure Pattern**
```go
func TestNewFeature(t *testing.T) {
    harness := helpers.NewTestHarness(t)
    defer harness.Cleanup()

    // Setup mocks
    harness.MockEmail.On("MethodName", mock.Anything).Return(expectedResult, nil)

    // Test implementation
    result := serviceUnderTest.DoSomething()

    // Verify
    assert.Equal(t, expected, result)
    harness.MockEmail.AssertExpectations(t)
}
```

#### **Integration Requirements**
- **Service Tests** - Test business logic with mocked dependencies
- **Component Tests** - Test TUI components with simulation screen
- **Integration Tests** - Test service interaction (when available)
- **Performance Tests** - Benchmark critical operations

### 🚀 **Future Comprehensive Testing**

The framework includes advanced testing capabilities ready for integration when TUI app methods are implemented:

#### **Waiting for Implementation**
- **Keyboard Shortcuts Testing** - User interaction simulation
- **Bulk Operations Testing** - Multi-message operation validation  
- **Async Operations Testing** - Goroutine leak detection and cancellation
- **Visual Regression Testing** - UI consistency with snapshots

#### **Ready to Integrate** 
Advanced test files are in `test/helpers/future/` and will be moved to active testing once these app methods exist:
- `HandleKeyEvent()`, `LoadMessagesAsync()`, `GetMessageCount()`
- `SetSelectedMessages()`, `ApplyLabelToSelectedAsync()`  
- `GetMessageListComponent()`, `GetMessageContentComponent()`
- `ShowSearchInterface()`, `ShowAIPanel()`

### 📊 **CI/CD Integration**

The testing framework includes automated CI/CD with:
- **Matrix Testing** - Multiple Go versions (1.21, 1.22, 1.23) and OS (Ubuntu, macOS)
- **Security Scanning** - Trivy vulnerability detection
- **Coverage Reporting** - Code coverage tracking and reporting
- **Visual Regression** - Automated UI consistency checking  
- **Notifications** - PR comments and Slack notifications

## 📋 **Test Plan Requirements**

**MANDATORY**: Every feature development MUST include a comprehensive test plan file to verify functionality.

### 🧪 **Test Plan Structure**
Each new feature must include a `testplan_[feature_name].md` file with:

#### **Required Sections:**
1. **Feature Overview** - Brief description of what's being tested
2. **Prerequisites** - Setup requirements, dependencies, configuration needed
3. **Test Scenarios** - Detailed step-by-step test cases covering:
   - **Happy Path** - Normal expected usage flows
   - **Edge Cases** - Boundary conditions, error states, unusual inputs
   - **Integration** - How feature interacts with existing functionality
   - **Regression** - Verify existing features still work
4. **Expected Results** - Clear success criteria for each test case
5. **Cleanup** - Steps to reset system state after testing

#### **Test Case Format:**
```markdown
### Test Case: [Descriptive Name]
**Objective**: What this test verifies
**Steps**:
1. Step 1 with specific actions
2. Step 2 with expected intermediate results
3. Step 3 with final verification

**Expected Result**: Specific, measurable outcome
**Actual Result**: [To be filled during testing]
**Status**: [PASS/FAIL/BLOCKED]
```

#### **Mandatory Test Categories:**
- **Keyboard Shortcuts** - All new key bindings work correctly
- **Command Parity** - Commands and shortcuts have equivalent functionality  
- **Bulk Mode** - Feature works with multiple selected items
- **Error Handling** - Graceful failure modes and user feedback
- **Focus Management** - Proper focus behavior and ESC key handling
- **Threading** - No deadlocks or race conditions
- **Performance** - Acceptable response times under normal load

#### **Test Plan Example Structure:**
```
testplan_content_search.md
├── Feature Overview
├── Prerequisites
│   ├── Configuration requirements
│   ├── Sample data setup
│   └── Environment verification
├── Test Scenarios
│   ├── Basic Search Functionality
│   ├── Navigation (n/N keys)
│   ├── Integration with Help System
│   ├── Focus Management
│   ├── Error Conditions
│   └── Regression Tests
├── Expected Results
└── Cleanup Steps
```

### 📝 **Help System Maintenance**

**CRITICAL**: The help system must be updated whenever new features, shortcuts, or commands are added.

#### **Help Update Checklist:**
1. **Update `generateHelpText()` in `internal/tui/app.go`**
2. **Add new keyboard shortcuts** to appropriate sections (NAVIGATION, ACTIONS, etc.)
3. **Include command aliases** for any new commands
4. **Update conditional features** (AI, themes, etc.) if applicable
5. **Test help content** using `/term` search to verify findability
6. **Verify formatting** doesn't break with new content
7. **Test navigation** (g/gg/G, n/N) works correctly
8. **Create test plan** for help system changes
9. **Update documentation** if help structure changes

#### **Help Content Format Conventions:**
- **Consistent spacing**: One blank line between sections, two before major headers
- **Emoji prefixes**: Use consistent emojis for section headers (📧, ⚡, 🔍, etc.)
- **Dynamic keys**: Reference `a.Keys.*` fields for configurable shortcuts
- **Conditional content**: Use feature flags for optional functionality
- **Clean layout**: No decorative separators, focus on readability
- **Search-friendly**: Use keywords that users would naturally search for

#### **Testing Help System Changes:**
1. **Visual verification**: Help content displays cleanly without formatting issues
2. **Search functionality**: `/term` finds relevant content for all new features
3. **Navigation**: All content navigation keys work (g/gg/G, n/N, ESC)
4. **Focus management**: Help mode properly handles focus and restoration
5. **Header toggling**: `h` key works correctly in normal mode (not in help mode)
6. **Content preservation**: Message content properly restored after help close

## 📖 **Documentation**

Always update:
- `docs/ARCHITECTURE.md` for architectural changes
- `docs/FOCUS_MANAGEMENT.md` for UI focus patterns and side panel behavior
- `README.md` for user-facing features
- `TODO.md` for completed tasks
- `CLAUDE.md` for debugging sessions and architectural lessons learned
- `testplan_[feature_name].md` for each new feature
- Code comments for complex logic

---

**Remember**: This architecture exists to make the codebase maintainable, testable, and robust. Follow these patterns consistently, and the code will remain high-quality as it grows.
- please do not include your signature on git commits