---
description: "Implement a new feature for Gmail TUI with full architecture compliance"
---

# Feature Implementation: $ARGUMENTS

Implement the following feature for Gmail TUI with complete architectural compliance and documentation.

## Feature Requirements
**Feature Description:** $ARGUMENTS

## MANDATORY Implementation Requirements

### 1. Architecture (Service-First Development)
- **ALL business logic MUST go in `internal/services/`**  
- Define service interface in `internal/services/interfaces.go`
- Implement service in dedicated file (e.g., `feature_service.go`)
- Add service to App struct in `internal/tui/app.go`
- Initialize service in `initServices()` method
- Update `GetServices()` method to return new service
- UI components should ONLY handle presentation and user input

### 2. Error Handling (CRITICAL)
- **ALWAYS use `app.GetErrorHandler()` for ALL user feedback**
- **NEVER use `fmt.Printf`, `log.Printf`, or direct output**
- Required methods: `ShowError()`, `ShowSuccess()`, `ShowWarning()`, `ShowInfo()`
- Follow async patterns for status updates from goroutines

### 3. Thread Safety
- **ALWAYS use accessor methods**: `GetCurrentView()`, `SetCurrentMessageID()`, etc.
- **NEVER access app struct fields directly**
- Use proper mutex protection for new state fields

### 4. ESC Key Handling (PREVENTS DEADLOCKS)
- **NEVER use `QueueUpdateDraw()` in ESC handlers or cleanup functions**
- **ALWAYS use synchronous operations for UI cleanup**
- Pattern: ESC handlers call cleanup functions with direct UI operations
- Examples: Use synchronous `split.ResizeItem()`, `SetFocus()`, etc.

### 5. Command Parity (MANDATORY)
- **Every keyboard shortcut MUST have an equivalent command**
- Commands MUST support bulk mode automatically  
- Commands MUST provide short aliases
- Add command case to `executeCommand()` in `internal/tui/commands.go`
- Add to command suggestions in `generateCommandSuggestion()`

### 6. Labels Focus Approach
- Study `internal/tui/labels.go` for the established pattern
- Labels picker uses side panel architecture with proper focus management
- Bulk operations on labels follow specific async patterns
- Filter-to-single-result behavior with Enter key support
- Cache management for label operations (`updateCachedMessageLabels`)

### 7. Side Panel Picker Architecture
- Follow patterns from `docs/FOCUS_MANAGEMENT.md`
- Use standardized picker creation and focus management
- Implement proper ESC key handling for side panels
- Support keyboard navigation and filtering
- Maintain focus state across operations

### 8. Bulk Operations Patterns  
- **ALL features must support bulk mode automatically**
- Check `a.bulkMode && len(a.selected) > 0` in operation handlers
- Use `go a.featureActionBulk()` vs `go a.featureAction()` pattern
- Progress indicators for bulk operations: `ShowProgress()` with counts
- Proper error aggregation and reporting for bulk failures

### 9. Modal/Picker Integration Tricks
- Use `restoreFocusAfterModal()` pattern for focus restoration
- Implement `cmdFocusOverride` for special focus cases
- Handle Tab cycling through pickers properly
- Support search/filter within pickers
- Graceful degradation when pickers unavailable

### 10. Streaming/Async Patterns (CRITICAL)
- **For streaming callbacks (AI, LLM): NEVER use `QueueUpdateDraw()`**
- Use direct UI updates in streaming callbacks: `view.SetText(content)`
- Always check `ctx.Done()` in streaming callbacks
- Pattern: Build content, check context, update directly
- ESC cancellation must work immediately with streaming operations

### 11. External Integration Architecture
- Slack integration: Follow picker → action → result pattern
- Obsidian integration: File creation with proper error handling  
- Use service interfaces for external integrations
- Graceful degradation when external services unavailable
- Configuration-driven integration (enable/disable features)

### 12. Shortcut Configuration Customization
- **Always use `a.Keys.*` fields, NEVER hardcoded keys**
- New shortcuts must be configurable in `config.json`
- Key binding validation: ensure no conflicts with existing shortcuts
- Help system must display user's configured shortcuts dynamically
- Provide sensible defaults for missing key configurations

### 13. Theming System Integration
- **Use `a.currentTheme` for ALL color decisions**
- Use `a.GetComponentColors("feature_name")` pattern for component colors
- Support live theme switching without restart
- Ensure readability across all theme variants (light/dark/custom)
- Test with different themes during implementation

### 14. Logging System (CRITICAL for Debugging)
- **Always use `a.logger` for ALL logging operations**
- **NEVER use `fmt.Printf`, `log.Printf`, or other direct output**
- Structured logging: `a.logger.Printf("operation: details, param1=%v", val1)`
- Log service operations: entry/exit with parameters and results
- Log UI state changes: focus changes, mode switches, etc.
- Log error conditions with full context
- Thread-safe: `a.logger` safe for use from goroutines
- **NEVER log sensitive data**: email content, passwords, personal info

### 15. Help System Updates (MANDATORY)
- **Update `generateHelpText()` in `internal/tui/app.go`**
- Add new keyboard shortcuts to appropriate sections
- Include command aliases for new commands
- Use dynamic key references (`a.Keys.*` fields) 
- Test help search functionality (`/term`)

### 16. Configuration Support
- Add configuration options to `internal/config/config.go` if needed
- Support customizable keyboard shortcuts in config
- Feature enable/disable flags where appropriate
- Provide sensible defaults

### 17. Testing & Validation
- **Create comprehensive test plan**: `testplan_[feature_name].md`  
- **Use the testing framework in `test/helpers/`** for automated testing
- **Unit tests**: Test service logic using mock dependencies
- **Component tests**: Test TUI components with test harness and SimulationScreen
- **Integration tests**: Test complete workflows using the integration test framework
- **Visual regression tests**: Test UI consistency with snapshot testing
- **Async operation tests**: Test goroutine management and cancellation patterns
- **Bulk operation tests**: Test multi-message operation validation
- **Keyboard shortcut tests**: Test user interaction simulation
- **Performance tests**: Test critical operation benchmarks
- Test all functionality: happy path, edge cases, error conditions
- Test keyboard shortcuts and command equivalents
- Test bulk mode support
- Test ESC key behavior and cleanup
- Test picker/modal integration
- Test theming integration
- Test external service integration
- **Run build verification**: `make build`
- **Run test framework**: `make test-all` or `make test-unit` for service tests
- Run linting: `make lint`, `make fmt` if available

## Documentation Requirements

### 1. README Updates
- Add feature description to main README.md
- Include new keyboard shortcuts in shortcuts table  
- Document new commands with examples
- Update feature list

### 2. Architecture Documentation  
- Update `docs/ARCHITECTURE.md` if architectural changes made
- Document new service interfaces
- Explain integration patterns
- Update `docs/FOCUS_MANAGEMENT.md` if focus patterns added

### 3. Test Plan Creation
- Create detailed `testplan_[feature_name].md`
- Include step-by-step testing procedures
- Cover all user interactions and edge cases
- Document expected vs actual results template

## Code Quality Standards

### Required Patterns
- Follow existing code style and conventions
- Use established error handling patterns  
- Implement proper logging throughout
- Follow naming conventions from existing codebase
- Study existing implementations for patterns

### Anti-Patterns to AVOID
- Business logic in UI components
- Direct field access instead of accessor methods
- `QueueUpdateDraw()` in ESC handlers (causes deadlocks)
- `QueueUpdateDraw()` in streaming callbacks (causes deadlocks)
- Missing command parity
- Missing bulk mode support
- Hardcoded keys instead of `a.Keys.*`
- Missing theming integration
- Missing logging
- Missing error handling with ErrorHandler

## Completion Checklist

Before considering complete, verify ALL of these:

- [ ] Service interface defined and implemented
- [ ] UI integration follows service-first pattern  
- [ ] Error handling uses ErrorHandler throughout
- [ ] Thread safety with accessor methods
- [ ] ESC key handling implemented correctly
- [ ] Bulk mode support implemented
- [ ] Keyboard shortcut uses `a.Keys.*` configuration
- [ ] Command equivalent created with bulk support
- [ ] Picker/modal integration if applicable
- [ ] External service integration if applicable
- [ ] Theming integration implemented
- [ ] Comprehensive logging added
- [ ] Help system updated with new feature
- [ ] README.md updated
- [ ] Configuration support added
- [ ] Test plan created and ready for execution
- [ ] Automated tests written using testing framework in `test/helpers/`
- [ ] Service unit tests with mock dependencies 
- [ ] Component tests with test harness and SimulationScreen
- [ ] Integration tests for complete workflows
- [ ] Visual regression tests for UI consistency
- [ ] Build successful (`make build`)
- [ ] All tests passing (`make test-all`)
- [ ] No linting errors

## Reference Files & Patterns

**Study these existing implementations:**
- `internal/tui/labels.go` - Labels focus approach and side panel picker
- `internal/tui/bulk_prompts.go` - Bulk operations and ESC handling  
- `internal/services/email_service.go` - Email operations pattern
- `internal/services/ai_service.go` - LLM integration and streaming
- `internal/tui/app.go` - Service integration and theming patterns
- `internal/tui/error_handler.go` - Error handling pattern
- `internal/tui/keys.go` - ESC key handling with streaming cancellation
- `internal/tui/threads.go` - Logging patterns and async operations
- `internal/config/config.go` - Configuration and key binding structure
- `docs/FOCUS_MANAGEMENT.md` - Side panel picker architecture

**Testing Framework References:**
- `test/helpers/test_harness.go` - Central testing utility with tcell.SimulationScreen
- `test/helpers/integration_test.go` - End-to-end workflow testing patterns
- `test/helpers/visual_regression_test.go` - UI consistency testing with snapshots
- `test/helpers/bulk_operations_test.go` - Multi-message operation testing
- `test/helpers/async_operations_test.go` - Goroutine and cancellation testing
- `test/helpers/keyboard_shortcuts_test.go` - User interaction testing
- `docs/testing-framework.md` - Comprehensive testing framework documentation

**CRITICAL:** Read `CLAUDE.md` thoroughly - it contains the essential architectural patterns, debugging lessons, and "tricks" that prevent bugs and ensure features integrate seamlessly.

Please implement this feature following ALL requirements above, especially the Gmail TUI-specific patterns and architectural tricks that make features robust and user-friendly.