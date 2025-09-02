---
description: "Debug a problematic feature in Gmail TUI with systematic analysis"
---

# Feature Debugging: $ARGUMENTS

Debug and resolve the following issue in Gmail TUI using systematic analysis and architectural compliance.

## Issue Details
**Problem Description:** $ARGUMENTS

## MANDATORY Debugging Process

### 1. Issue Analysis & Categorization
- **Analyze the problem description** thoroughly
- **Identify affected components** (UI, services, integrations)
- **Categorize issue type** (crash, hang, incorrect behavior, performance, UI)
- **Assess severity and user impact**
- **Reproduce the issue** if possible with available information
- **Check recent changes** that might have introduced the issue

### 2. Log Analysis (Primary Debugging Tool)
- **Examine `a.logger` output** for the affected operations
- **Look for error patterns** and execution flow
- **Trace service method calls** and their parameters
- **Check UI state changes** logged during issue occurrence  
- **Identify timing patterns** or race conditions in logs
- **Missing logs** often indicate where problems occur

### 3. Code Investigation
- **Examine relevant source files** based on issue description
- **Trace execution paths** through the codebase
- **Check for architectural violations**:
  - Business logic in UI components
  - Direct field access instead of accessors
  - Missing error handling with ErrorHandler
  - `QueueUpdateDraw()` in ESC handlers or streaming callbacks
- **Review recent changes** in Git history
- **Look for thread safety issues**

### 4. Common Issue Patterns

#### **Application Hanging/Deadlock**
- Check for `QueueUpdateDraw()` in ESC handlers (most common cause)
- Look for `QueueUpdateDraw()` in streaming callbacks (AI features)
- Examine goroutine synchronization and channel blocking
- Check for circular dependencies in UI updates
- Review mutex usage and potential deadlocks

#### **Feature Not Working**
- Verify service initialization in `initServices()`
- Check service method implementations and error handling
- Test keyboard shortcut and command parity
- Examine UI integration patterns and focus management
- Validate bulk mode support

#### **Performance Issues**  
- Profile memory usage and identify leaks
- Check for inefficient algorithms or unnecessary API calls
- Examine caching strategies and database queries
- Look for blocking operations on UI thread
- Review streaming and async operation patterns

#### **UI Display Issues**
- Check theme integration and color usage
- Validate screen width calculations and text formatting
- Test with different terminal sizes and themes
- Verify focus management and Tab cycling
- Examine picker/modal integration

#### **Labels/Picker Issues**
- Check side panel picker architecture compliance
- Verify focus restoration patterns
- Test filter-to-single-result behavior
- Examine cache management for operations
- Check bulk operation support

#### **External Integration Problems**
- Test service availability and error handling
- Check configuration-driven integration flags
- Verify graceful degradation when services unavailable
- Examine authentication and API integration
- Test timeout and retry mechanisms

### 5. Root Cause Analysis
- **Distinguish symptoms from actual causes**
- **Identify the underlying issue** (not just visible problem)
- **Assess if it's a regression** or existing bug
- **Determine scope of impact** (single feature vs system-wide)
- **Consider architectural implications**

### 6. Fix Strategy Development
- **Design targeted fixes** that address root cause
- **Consider side effects** and potential regressions
- **Plan testing approach** to verify fix
- **Assess if architectural changes** are needed
- **Follow established patterns** from CLAUDE.md

### 7. Implementation & Testing
- **Apply fixes following architectural requirements**:
  - Maintain service-first architecture
  - Use ErrorHandler for all user feedback
  - Ensure thread safety with accessor methods
  - Proper ESC key handling
  - Comprehensive logging of changes
- **Add safeguards** to prevent similar issues
- **Use testing framework for verification**:
  - Write unit tests for service fixes using mock dependencies
  - Create component tests with test harness and SimulationScreen  
  - Add integration tests for end-to-end workflow validation
  - Use visual regression tests to catch UI inconsistencies
  - Test async operations and goroutine management
  - Validate bulk operation behavior
  - Test keyboard shortcut functionality
- **Test thoroughly** including edge cases and bulk operations
- **Verify no regressions** in related functionality

## Debugging Toolkit

### **Log Analysis Patterns**
```bash
# Filter logs for specific operations
grep "serviceName:" app.log | grep "operation"

# Check for error patterns
grep "error\|failed\|Error" app.log

# Trace specific user actions
grep "focusManager\|keyHandler" app.log
```

### **Testing Framework for Debugging**
```bash
# Run relevant tests to reproduce and verify fixes
make test-unit          # Test service logic in isolation
make test-tui           # Test TUI components with simulation
make test-integration   # Test end-to-end workflows
make test-all          # Run comprehensive test suite

# Test specific components during debugging
go test ./test/helpers -run TestBulkOperations    # Bulk operation issues
go test ./test/helpers -run TestKeyboardShortcuts # Keyboard shortcut problems
go test ./test/helpers -run TestAsyncOperations   # Async/goroutine issues
go test ./test/helpers -run TestVisualRegression  # UI display problems
```

### **Test Harness for Issue Reproduction**
Use the test harness to reproduce issues in isolation:
```go
// Create isolated test environment
harness := helpers.NewTestHarness(t)
defer harness.Cleanup()

// Simulate problematic user interactions
harness.SimulateKeyEvent(tcell.KeyEscape, 0, tcell.ModNone)
harness.SimulateTyping("problematic input")

// Capture screen state for analysis
content := harness.GetScreenContent()
assert.Contains(t, content, "expected behavior")
```

### **Code Investigation Checklist**
- [ ] Service method entry/exit logging present
- [ ] Error conditions properly logged
- [ ] UI state changes tracked
- [ ] ESC handlers use synchronous operations only
- [ ] Streaming callbacks avoid `QueueUpdateDraw()`
- [ ] Bulk mode support implemented
- [ ] Thread safety with accessor methods
- [ ] ErrorHandler used for user feedback

### **Focus/Modal Integration Issues**
- Check `restoreFocusAfterModal()` usage
- Verify `cmdFocusOverride` implementation
- Test Tab cycling behavior
- Examine side panel picker patterns
- Validate ESC key behavior

### **Performance Debugging**
- Profile with `go tool pprof`
- Check goroutine counts and memory usage
- Examine database query patterns
- Look for caching inefficiencies
- Test with large datasets

## Fix Implementation Requirements

### **Architecture Compliance**
- Follow service-first development patterns
- Use established error handling patterns
- Maintain thread safety requirements
- Implement proper logging throughout fix
- Ensure theme and configuration integration

### **Documentation Updates**
- Update `CLAUDE.md` with debugging lessons learned
- Document root cause and fix approach
- Add safeguards to prevent recurrence
- Update test plans if needed

### **Testing & Validation**
- **Use testing framework for comprehensive validation**:
  - Write unit tests to prevent regression of the specific issue
  - Create component tests to verify TUI behavior  
  - Add integration tests for end-to-end workflow validation
  - Use visual regression tests for UI-related fixes
- Test the specific issue reproduction case
- Test related functionality for regressions
- Test bulk operations if applicable
- Test different themes and configurations
- Verify ESC key behavior
- Run `make build` and testing commands: `make test-all`
- Run linting

## Post-Debug Checklist

- [ ] Root cause clearly identified and documented
- [ ] Fix addresses cause, not just symptoms
- [ ] No architectural violations introduced
- [ ] Automated tests added using testing framework in `test/helpers/`
- [ ] Unit tests written for service fixes with mock dependencies
- [ ] Component tests created with test harness for TUI behavior
- [ ] Integration tests added for end-to-end validation
- [ ] Visual regression tests included for UI-related fixes
- [ ] Comprehensive logging added for future debugging
- [ ] ESC key handling verified correct
- [ ] Bulk operations tested if applicable
- [ ] No regressions in related features
- [ ] Build successful and linting clean
- [ ] All tests passing (`make test-all`)
- [ ] Lessons learned documented in CLAUDE.md

## Reference Files for Debugging

**Study these for common patterns:**
- `internal/tui/threads.go` - Recent debugging examples and logging patterns
- `internal/tui/bulk_prompts.go` - ESC handling and bulk operations
- `internal/tui/labels.go` - Side panel picker debugging
- `internal/tui/keys.go` - ESC key handling patterns
- `CLAUDE.md` - Historical debugging sessions and solutions

**Testing Framework for Debugging:**
- `test/helpers/test_harness.go` - Test harness for isolated issue reproduction
- `test/helpers/integration_test.go` - End-to-end workflow testing patterns
- `test/helpers/visual_regression_test.go` - UI consistency testing
- `test/helpers/bulk_operations_test.go` - Bulk operation debugging patterns
- `test/helpers/async_operations_test.go` - Goroutine and cancellation testing
- `docs/TESTING.md` - Complete testing framework documentation and patterns

**Log Analysis Tools:**
- `a.logger.Printf()` - Current logging approach
- Recent debugging sessions in CLAUDE.md
- Focus management patterns in `docs/FOCUS_MANAGEMENT.md`

**CRITICAL:** The logging system is your primary debugging tool. If logs don't show the issue clearly, first add more logging, then reproduce the issue.

Please debug this issue following the systematic approach above, focusing on log analysis and architectural compliance in your solution.