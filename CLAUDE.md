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
    }
    a.currentMessageID = "new-id"           // Direct field access
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

## 📖 **Documentation**

Always update:
- `docs/ARCHITECTURE.md` for architectural changes
- `README.md` for user-facing features
- `TODO.md` for completed tasks
- Code comments for complex logic

---

**Remember**: This architecture exists to make the codebase maintainable, testable, and robust. Follow these patterns consistently, and the code will remain high-quality as it grows.