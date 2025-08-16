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

## 🎯 **Development Workflow**

When asked to implement a new feature:

1. **Analyze Requirements** - Identify what services are needed
2. **Check Existing Services** - Reuse if possible, extend if needed
3. **Design Service Interface** - Define clean contracts
4. **Validate with the user the proposed solution**** - Make sure the customer agrees on the approach 
4. **Implement Service** - Business logic only
5. **Integrate with UI** - Presentation logic only
6. **Add Error Handling** - Use ErrorHandler consistently
7. **Ensure Thread Safety** - Use accessor methods
8. **Test Integration** - Verify build and functionality

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

## 📖 **Documentation**

Always update:
- `docs/ARCHITECTURE.md` for architectural changes
- `README.md` for user-facing features
- `TODO.md` for completed tasks
- `CLAUDE.md` for debugging sessions and architectural lessons learned
- Code comments for complex logic

---

**Remember**: This architecture exists to make the codebase maintainable, testable, and robust. Follow these patterns consistently, and the code will remain high-quality as it grows.