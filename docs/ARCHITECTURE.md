# 🏗️ Gmail TUI Architecture Guide

This document outlines the architectural patterns and conventions for Gmail TUI development.

## 📋 **Development Principles**

### 🎯 **Core Rules**
1. **Service Layer First** - All business logic goes in `internal/services/`
2. **No Business Logic in UI** - TUI components only handle presentation
3. **Use Error Handler** - Always use `app.GetErrorHandler()` for user feedback
4. **Thread-Safe Access** - Use provided getter/setter methods for app state
5. **Dependency Injection** - Services are injected, never instantiated directly

## 🔧 **Service Layer Patterns**

### ✅ **When to Create a New Service**
- **Business Operations** - Any Gmail API operations (send, archive, label)
- **External Integrations** - LLM calls, calendar operations, cache operations
- **Complex Logic** - Multi-step workflows, data transformations
- **Reusable Operations** - Logic used by multiple UI components

### 📝 **Service Interface Template**
```go
// internal/services/my_service.go
package services

import "context"

type MyService interface {
    DoSomething(ctx context.Context, param string) error
    GetSomething(ctx context.Context, id string) (*MyType, error)
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

func (s *MyServiceImpl) DoSomething(ctx context.Context, param string) error {
    // Business logic here
    return nil
}
```

### 🔌 **Service Integration Steps**
1. **Define Interface** in `internal/services/interfaces.go`
2. **Implement Service** in dedicated file
3. **Add to App struct** in `internal/tui/app.go`
4. **Initialize in initServices()** method
5. **Return in GetServices()** method

## 🎨 **UI Component Patterns**

### ✅ **UI Component Responsibilities**
- **Presentation Logic** - Rendering, formatting, styling
- **User Input Handling** - Key bindings, mouse events
- **View State Management** - Focus, selection, visibility
- **Service Coordination** - Calling services, handling responses

### ❌ **What UI Components Should NOT Do**
- Direct Gmail API calls
- Business logic calculations
- Data transformations
- Complex error handling (use ErrorHandler instead)

### 📝 **UI Component Template**
```go
// internal/tui/my_component.go
func (a *App) handleMyAction() error {
    // 1. Get services
    emailService, _, _, _, _ := a.GetServices()
    
    // 2. Get current state (thread-safe)
    messageID := a.GetCurrentMessageID()
    
    // 3. Call service for business logic
    if err := emailService.DoSomething(a.ctx, messageID); err != nil {
        // 4. Use error handler for user feedback
        a.GetErrorHandler().ShowError(a.ctx, "Failed to do something")
        return err
    }
    
    // 5. Show success feedback
    a.GetErrorHandler().ShowSuccess(a.ctx, "Something completed successfully")
    
    // 6. Update UI state if needed
    a.refreshView()
    
    return nil
}
```

## 🔒 **Thread-Safe State Management**

### ✅ **Always Use Accessor Methods**
```go
// ✅ Correct - Thread-safe
currentView := app.GetCurrentView()
app.SetCurrentMessageID(messageID)
ids := app.GetMessageIDs() // Returns copy

// ❌ Wrong - Direct access
currentView := app.currentView      // Race condition
app.currentMessageID = messageID    // Race condition
```

### 📝 **Adding New State Fields**
1. Add private field to App struct
2. Add getter method with `RLock()`
3. Add setter method with `Lock()`
4. Use proper copying for slices/maps

## 🎯 **UI Focus Management**

For detailed information about focus management, border highlighting, and side panel patterns, see:
**📖 [Focus Management Guide](./FOCUS_MANAGEMENT.md)**

Key concepts:
- **Reuse "labels" focus state** for all side panel pickers (labels, links, prompts)
- **Command parity** - Ensure `:command` behaves identically to keyboard shortcuts
- **Consistent UI patterns** - All pickers use same container and focus restoration

### 🔧 **ActivePicker State Management**

Use the **ActivePicker enum system** for side panel picker state management:

```go
// ✅ Correct - Use specific picker enum values
a.setActivePicker(PickerLabels)     // Labels picker
a.setActivePicker(PickerDrafts)     // Drafts picker  
a.setActivePicker(PickerAttachments) // Attachments picker
a.setActivePicker(PickerNone)       // Close picker

// ✅ Check picker state
if a.isLabelsPickerActive() {
    // Only populate labels when Labels picker is active
    a.populateLabelsQuickView(messageID)
}

// ❌ Wrong - Never use shared boolean flags
a.labelsVisible = true  // Causes race conditions
```

**Available picker constants:**
- `PickerNone` - No picker active
- `PickerLabels` - Labels picker  
- `PickerDrafts` - Drafts picker
- `PickerAttachments` - Attachments picker
- `PickerObsidian` - Obsidian integration
- `PickerLinks` - Links picker
- `PickerPrompts` - Prompts picker
- `PickerBulkPrompts` - Bulk prompts picker
- `PickerSavedQueries` - Saved queries picker
- `PickerThemes` - Theme picker
- `PickerAI` - AI labels picker
- `PickerContentSearch` - Content search picker
- `PickerRSVP` - RSVP picker

**Benefits:**
- Prevents race conditions during screen resize/maximize
- Avoids wrong picker display after async operations
- Enables proper focus restoration after window events
- Provides clear debugging through specific state tracking

## 🚨 **Error Handling Patterns**

### ✅ **Use ErrorHandler for All User Feedback**
```go
// ✅ Correct
app.GetErrorHandler().ShowError(ctx, "Operation failed")
app.GetErrorHandler().ShowSuccess(ctx, "Operation completed")
app.GetErrorHandler().ShowProgress(ctx, "Loading...")

// ❌ Wrong
fmt.Printf("Error: %v\n", err)  // Direct output
log.Printf("Success")           // Only logs, no user feedback
```

### 📊 **Error Handling Levels**
- **ShowError()** - User-facing error messages
- **ShowWarning()** - Non-critical issues
- **ShowSuccess()** - Operation confirmations  
- **ShowInfo()** - General information
- **ShowProgress()** - Long-running operations

## 🧪 **Testing Patterns**

### 📝 **Service Testing Template**
```go
// internal/services/my_service_test.go
func TestMyService_DoSomething(t *testing.T) {
    // Arrange
    mockClient := &MockGmailClient{}
    service := NewMyService(mockClient, testConfig)
    
    // Act
    err := service.DoSomething(context.Background(), "test-param")
    
    // Assert
    assert.NoError(t, err)
    assert.True(t, mockClient.WasCalled("ExpectedMethod"))
}
```

### 🎯 **Testing Strategy**
- **Unit Tests** for services (business logic)
- **Integration Tests** for service + Gmail API
- **UI Tests** for component behavior
- **Mock Services** for UI testing

## 📋 **Development Checklist**

When adding a new feature, ensure:

### ✅ **Service Layer**
- [ ] Business logic is in a service, not UI
- [ ] Service implements an interface
- [ ] Service is properly initialized in `initServices()`
- [ ] Service uses context for cancellation
- [ ] Service handles errors appropriately

### ✅ **UI Integration** 
- [ ] UI calls service methods, not direct APIs
- [ ] Uses `GetErrorHandler()` for user feedback
- [ ] Uses thread-safe accessor methods for state
- [ ] No business logic in UI components
- [ ] Proper separation of concerns

### ✅ **Error Handling**
- [ ] All errors shown to user via ErrorHandler
- [ ] Appropriate error levels used
- [ ] Success messages for important operations
- [ ] Progress indication for long operations

### ✅ **Thread Safety**
- [ ] No direct access to app fields
- [ ] Uses provided getter/setter methods
- [ ] Proper copying for shared data structures
- [ ] Mutex protection for new state fields

## 🔄 **Migration Guide**

### Converting Existing Code
1. **Identify Business Logic** in UI components
2. **Extract to Service** with proper interface
3. **Update UI** to call service methods
4. **Add Error Handling** with ErrorHandler
5. **Make State Access Thread-Safe**

## 📚 **Examples**

See existing implementations:
- `internal/services/email_service.go` - Email operations
- `internal/services/ai_service.go` - LLM integration
- `internal/tui/app.go` - Service integration
- `internal/tui/error_handler.go` - Error handling

## 🎯 **Quick Reference**

### Service Creation
```bash
# 1. Add interface to interfaces.go
# 2. Create service implementation
# 3. Add to App struct
# 4. Initialize in initServices()
# 5. Add to GetServices() return
```

### UI Development
```bash
# 1. Get services from app.GetServices()
# 2. Use thread-safe state methods
# 3. Call service methods for business logic
# 4. Use ErrorHandler for feedback
# 5. Update UI state as needed
```

---

**Remember**: The goal is maintainable, testable, and robust code. When in doubt, follow the existing patterns in the codebase.