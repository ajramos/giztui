# 🤖 Claude Code Development Guide

This file provides essential architectural patterns and requirements for Claude Code when working on GizTUI.

## 📝 **Git Commit Guidelines**

When committing changes, **DO NOT** include Claude signatures or co-authored by lines in commit messages. Keep commit messages clean and focused on the actual changes.

## 🏗️ **Core Architecture Requirements**

### 🎯 **Service-First Development (MANDATORY)**
- **ALL business logic** must go in `internal/services/`
- **UI components** only handle presentation and user input
- **NEVER** put Gmail API calls, LLM calls, or complex logic in TUI components

### 📝 **New Feature Development Steps**
1. **Define Service Interface** in `internal/services/interfaces.go`
2. **Implement Service** in dedicated file (e.g., `my_service.go`)
3. **Add Service to App** struct in `internal/tui/app.go`
4. **Initialize Service** in `initServices()` method
5. **Update GetServices()** method to return new service
6. **UI Integration** - call service methods from UI components

## 🚨 **Critical Patterns**

### **Error Handling (MANDATORY)**
- **ALWAYS** use `app.GetErrorHandler()` for user feedback
- **NEVER** use `fmt.Printf`, `log.Printf`, or direct output for user messages
- **Required methods**: `ShowError()`, `ShowSuccess()`, `ShowWarning()`, `ShowInfo()`

### **Thread Safety (MANDATORY)**
- **ALWAYS** use accessor methods: `GetCurrentView()`, `SetCurrentMessageID()`, etc.
- **NEVER** access app struct fields directly
- **ALWAYS** use proper mutex protection for new state fields

### **Picker State Management (MANDATORY)**
- **ALWAYS** use `ActivePicker` enum system for side panel pickers
- **NEVER** use shared boolean flags like `labelsVisible`
- **ALWAYS** use `setActivePicker()` and `isLabelsPickerActive()` methods

```go
// ✅ CORRECT - Use specific picker enum
a.setActivePicker(PickerLabels)
if a.isLabelsPickerActive() {
    a.populateLabelsQuickView(messageID)
}

// ❌ WRONG - Shared boolean causes race conditions
a.labelsVisible = true  // Multiple pickers conflict
if a.labelsVisible {    // Wrong picker may trigger
    a.populateLabelsQuickView(messageID)
}
```

### **Theming (MANDATORY)**
- **ALWAYS** use `app.GetComponentColors("component")` for all UI theming
- **NEVER** use deprecated theme methods or hardcoded colors
- **Component selection rules**:
  - **Feature components**: `ai`, `slack`, `obsidian`, `links`, `stats`, `prompts`
  - **System components**: `general`, `search`
  - **Picker components**: `attachments`, `saved_queries`, `labels`

```go
// ✅ CORRECT - Use hierarchical theme system
componentColors := app.GetComponentColors("search")
container.SetBackgroundColor(componentColors.Background.Color())
container.SetTitleColor(componentColors.Title.Color())
container.SetBorderColor(componentColors.Border.Color())
```

## ⚡ **Critical Threading Patterns**

### **ESC Key Handling (CRITICAL)**
- **NEVER** use `QueueUpdateDraw()` in ESC handlers or cleanup functions
- **ALWAYS** use synchronous operations for UI cleanup to prevent deadlocks

### **Status Messages (CRITICAL)**
- **ALWAYS** use `ErrorHandler` for ALL status operations
- **NEVER** use direct status methods (`setStatusPersistent`, `showStatusMessage`)
- **NEVER** wrap ErrorHandler calls in `QueueUpdateDraw()`

```go
// ✅ CORRECT - From key handlers, use goroutine
func (a *App) handleKey() *tcell.EventKey {
    // Do UI operations first (synchronous)
    a.reformatListItems()
    
    // Then ErrorHandler calls asynchronously
    go func() {
        a.GetErrorHandler().ShowInfo(a.ctx, "Message")
    }()
    return nil
}
```

### **Streaming Callbacks (CRITICAL)**
- Streaming callbacks **MUST NEVER** use `QueueUpdateDraw()`
- **ALWAYS** use direct UI updates to prevent ESC deadlocks

```go
// ✅ CORRECT - Direct UI update in streaming callback
err := streamer.GenerateStream(ctx, prompt, func(token string) {
    select {
    case <-ctx.Done():
        return // Exit early if cancelled
    default:
    }
    
    // Direct update - NEVER QueueUpdateDraw
    if ctx.Err() == nil && a.aiSummaryView != nil {
        a.aiSummaryView.SetText(token)
    }
})
```

## ❌ **Critical Anti-Patterns**

### **NEVER Do These (Causes Deadlocks)**
```go
// ❌ Business logic in UI
messages, err := a.Client.GetMessages() // Direct API call in UI

// ❌ Direct output
fmt.Printf("Error: %v\n", err)

// ❌ Direct field access
a.currentMessageID = "new-id"

// ❌ QueueUpdateDraw in ESC handlers
a.QueueUpdateDraw(func() { /* cleanup */ })

// ❌ ErrorHandler in QueueUpdateDraw
a.QueueUpdateDraw(func() {
    a.GetErrorHandler().ShowProgress(ctx, "msg") // DEADLOCK!
})

// ❌ Hardcoded colors/deprecated theme methods
container.SetBackgroundColor(tcell.ColorBlue)
container.SetTitleColor(a.getTitleColor()) // REMOVED

// ❌ Shared picker boolean flags
a.labelsVisible = true  // Race conditions with multiple pickers
if a.labelsVisible {    // Wrong picker may be active
    // Business logic
}
```

## 📋 **Essential Code Templates**

### **Service Implementation**
```go
// internal/services/my_service.go
type MyService interface {
    DoOperation(ctx context.Context, param string) error
}

type MyServiceImpl struct {
    client *gmail.Client
    config *config.Config
}

func NewMyService(client *gmail.Client, config *config.Config) *MyServiceImpl {
    return &MyServiceImpl{client: client, config: config}
}

func (s *MyServiceImpl) DoOperation(ctx context.Context, param string) error {
    // Implementation here
    return nil
}
```

### **UI Integration**
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

## 🎮 **Command Parity (MANDATORY)**

**Every keyboard shortcut MUST have an equivalent command**:
- Commands **MUST** support bulk mode automatically  
- Commands **MUST** provide short aliases
- Add to `executeCommand()` in `internal/tui/commands.go`
- Add to command suggestions in `generateCommandSuggestion()`

## 🛠️ **Build & Test Commands**
- `make build` - Build the application
- `make test` - Run tests
- `make fmt` - Format code

## 🎯 **Development Workflow**

When implementing a new feature:
1. **Analyze Requirements** - Identify what services are needed
2. **Check Existing Services** - Reuse if possible, extend if needed
3. **Design Service Interface** - Define clean contracts
4. **Validate with user** - Confirm approach before implementing
5. **Implement Service** - Business logic only
6. **Integrate with UI** - Presentation logic only
7. **Add Error Handling** - Use ErrorHandler consistently
8. **Ensure Thread Safety** - Use accessor methods
9. **Command Parity** - Add equivalent command
10. **Test Integration** - Verify build and functionality

## 📚 **Reference Documentation**

For detailed information, see:
- `docs/ARCHITECTURE.md` - Complete architectural patterns
- `docs/THEMING.md` - Theme system usage and component guidelines  
- `docs/FOCUS_MANAGEMENT.md` - UI focus patterns and side panel behavior
- `internal/services/interfaces.go` - All service contracts
- `internal/tui/error_handler.go` - Error handling patterns
- `internal/tui/keys.go` - ESC key handling examples

---

**Remember**: This architecture ensures maintainable, testable, and robust code. Follow these patterns consistently for high-quality development.