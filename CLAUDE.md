# 🤖 Claude Code Development Guide

This file provides essential architectural patterns and requirements for Claude Code when working on GizTUI.

## 📚 **Quick Documentation Access**

**New to GizTUI development?** Start here:
- 📖 [Documentation Hub](docs/README.md) - Complete documentation navigation
- 🏗️ [Architecture Guide](docs/ARCHITECTURE.md) - Development patterns and conventions  
- 🧪 [Testing Guide](docs/TESTING.md) - Quality assurance framework
- 🎨 [Theming Guide](docs/THEMING.md) - UI component theming system

## 🗂️ **Project Layout**

- `cmd/giztui/main.go` - Application entry point
- `internal/services/` - All business logic (12+ services; see `interfaces.go` for canonical contracts)
- `internal/tui/` - Presentation layer (`app.go` is the central App struct, 3k+ lines)
- `internal/gmail/`, `internal/llm/`, `internal/cache/`, `internal/db/` - External integrations
- `pkg/auth/` - OAuth2 token handling
- `scripts/check-architecture.sh` - Architecture compliance validation script
- Runtime config: `~/.config/giztui/config.json`; OAuth creds: `~/.config/giztui/credentials.json`
- Interactive first-run setup: `giztui --setup`

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

**Available picker constants:**
- `PickerNone` - No picker active
- `PickerLabels` - Labels picker  
- `PickerDrafts` - Drafts picker
- `PickerObsidian` - Obsidian integration
- `PickerAttachments` - Attachments picker
- `PickerLinks` - Links picker
- `PickerPrompts` - Prompts picker
- `PickerBulkPrompts` - Bulk prompts picker
- `PickerSavedQueries` - Saved queries picker
- `PickerThemes` - Theme picker
- `PickerAI` - AI labels picker
- `PickerContentSearch` - Content search picker
- `PickerRSVP` - RSVP picker

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

**All supported components**: `general`, `search`, `attachments`, `obsidian`, `saved_queries`, `slack`, `prompts`, `ai`, `labels`, `stats`, `links`

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

**Available ErrorHandler methods**:
- `ShowError(ctx, message)` - Red error messages
- `ShowSuccess(ctx, message)` - Green success messages  
- `ShowWarning(ctx, message)` - Yellow warning messages
- `ShowInfo(ctx, message)` - Blue info messages
- `ShowProgress(ctx, message)` - Progress indicators
- `SetPersistentStatus(ctx, message)` - Long-term status

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
    // 1. Get services — GetServices() returns 12 values in this fixed order:
    //    EmailService, AIService, LabelService, CacheService, MessageRepository,
    //    CompositionService, PromptService, ObsidianService, LinkService,
    //    GmailWebService, AttachmentService, DisplayService.
    //    Use _ to discard the ones you don't need. See internal/services/interfaces.go.
    emailService, _, _, _, _, _, _, _, _, _, _, _ := a.GetServices()

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

**Examples of keyboard → command parity**:
- `a` (archive) → `:archive` or `:arch`
- `t` (trash) → `:trash` or `:tr`
- `l` (labels) → `:labels` or `:lab`
- `Ctrl+A` (select all) → `:select all` or `:sel all`
- `/` (search) → `:search <query>` or `:s <query>`

**Gmail search integration** - Commands should support Gmail search operators (see [GMAIL_SEARCH_REFERENCE.md](docs/GMAIL_SEARCH_REFERENCE.md)):
```
:search from:example@gmail.com has:attachment
:s subject:invoice after:2024/01/01
```

## 🛠️ **Build & Test Commands**

### Essential Commands
- `make build` - Build the application with version injection
- `make test` - Run tests (scoped to `./internal/... ./test/helpers ./test ./pkg/...`, not `./...`)
- `make pre-commit-check` - Run the same checks as CI locally (fmt + vet + golangci-lint + essential tests). **Use this before claiming work complete.**
- `make fmt` - Format code
- `make lint` - Run linting (requires golangci-lint)
- `make vet` - Verify code

### Testing Commands  
- `make test-all` - Run all tests (mocks + unit + tui + integration + coverage)
- `make test-unit` - Run unit tests
- `make test-tui` - Run TUI component tests
- `make test-integration` - Run integration tests
- `make test-mocks` - Generate mocks using mockery
- `make test-coverage` - Run tests with coverage
- `make test-race` - Run tests with race detector

### Development Commands
- `make dev` - Development mode (build and run)
- `make run` - Run the application
- `make debug` - Build with debug information
- `make clean` - Clean generated files
- `make deps` - Install dependencies
- `make install` - Install the application

### Release Commands
- `make release` - Prepare release (build binaries and generate archives)
- `make release-build` - Build release binaries for all platforms
- `make version` - Show version information

For complete list: `make help`

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
9. **Command Parity** - Add equivalent command (see [KEYBOARD_SHORTCUTS.md](docs/KEYBOARD_SHORTCUTS.md))
10. **Test Integration** - Verify build and functionality (see [TESTING.md](docs/TESTING.md))
11. **Update in-app `:help`** - Document new keys/commands/options on the `?` help screen (not just the docs). This is part of Definition of Done, not optional.
12. **Config defaults + migration** - If the feature adds a config option, add it to `DefaultConfig()` AND ensure the config self-migration path surfaces it to existing users' `config.json` on other machines (a default in code is not enough — users on an older config file must get the new key). Part of Definition of Done.
13. **Documentation** - Update relevant docs in [docs/](docs/) if needed

## 🚀 **Release Management**

When ready to create a release, follow the comprehensive procedure:
- **[Release Procedure Guide](docs/RELEASE_PROCEDURE.md)** - Complete step-by-step release process
- **Quality Gates** - Mandatory validation before any release
- **Semantic Versioning** - Proper MAJOR.MINOR.PATCH version management
- **Documentation Updates** - CHANGELOG.md and version file maintenance
- **GitHub Integration** - Automated release creation with assets

## 📚 **Reference Documentation**

For detailed information, see:
- `docs/ARCHITECTURE.md` - Complete architectural patterns
- `docs/THEMING.md` - Theme system usage and component guidelines  
- `docs/FOCUS_MANAGEMENT.md` - UI focus patterns and side panel behavior
- `docs/TESTING.md` - Testing framework and quality assurance
- `docs/RELEASE_PROCEDURE.md` - Comprehensive release management guide
- `docs/KEYBOARD_SHORTCUTS.md` - Command system and shortcut examples
- `docs/GMAIL_SEARCH_REFERENCE.md` - Gmail search operators and patterns
- `internal/services/interfaces.go` - All service contracts
- `internal/tui/error_handler.go` - Error handling patterns
- `internal/tui/keys.go` - ESC key handling examples

---

**Remember**: This architecture ensures maintainable, testable, and robust code. Follow these patterns consistently for high-quality development.