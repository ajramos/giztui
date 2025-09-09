# Testing Guide

## Overview

GizTUI uses a comprehensive testing framework designed specifically for TUI applications. The framework follows service-oriented architecture principles and provides robust testing capabilities for Go applications built with tview.

## Quick Start

```bash
# Generate mocks first
make test-mocks

# Run all tests
make test-all

# Run specific test types
make test-unit      # Service layer tests
make test-tui       # TUI component tests
make test-integration # Integration tests
make test-coverage  # Tests with coverage report
```

## Testing Architecture

### Testing Layers

1. **Component Testing** - Tests individual TUI components using `tcell.SimulationScreen`
2. **Service Testing** - Tests business logic with mocked dependencies
3. **Integration Testing** - Tests with real services using VCR recording
4. **Visual Regression Testing** - Tests UI consistency using snapshots
5. **Performance Testing** - Benchmarks and load testing

### Key Principles

- **No Running Application Testing** - Avoid `app.Run()` in tests
- **Service-Oriented Testing** - Test business logic separately from UI
- **Component-Level Testing** - Test individual components in isolation
- **Mock-First Approach** - Use mocks for external dependencies
- **Visual Consistency** - Prevent UI regressions with snapshots

## Framework Components

### Test Harness

The central testing utility (`test/helpers/test_harness.go`) provides:

- `SimulationScreen` for TUI component testing
- Mocked service instances
- Keyboard event simulation
- Screen content capture
- State validation utilities

```go
harness := helpers.NewTestHarness(t)
defer harness.Cleanup()

// Test component
harness.DrawComponent(component)
harness.SimulateKeyEvent(tcell.KeyCtrlA, 0, tcell.ModCtrl)
harness.AssertScreenContains(t, "Expected Text")
```

### Specialized Test Helpers

#### Bulk Operations Testing
- Range selection testing
- Pattern-based selection
- Bulk archive/trash operations
- Edge case handling
- Performance validation

#### Keyboard Shortcuts Testing
- Navigation shortcuts
- Message operation shortcuts
- Bulk operation shortcuts
- Search and filter shortcuts
- AI feature shortcuts

#### Async Operations Testing
- Message loading
- AI summary generation
- Bulk label application
- Search operations
- Cancellation and timeout handling
- Goroutine leak detection

#### Visual Regression Testing
- Component rendering tests
- State change visualization
- Responsive layout testing
- Focus indicator testing
- Color scheme testing

## Writing Tests

### Basic Component Test

```go
func TestMessageList(t *testing.T) {
    harness := helpers.NewTestHarness(t)
    defer harness.Cleanup()
    
    // Setup mocks
    harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
        Return(&services.MessagePage{
            Messages: harness.GenerateTestMessages(10),
        }, nil)
    
    // Test component
    component := harness.App.GetMessageListComponent()
    harness.DrawComponent(component)
    
    // Validate
    harness.AssertScreenContains(t, "Test Subject")
}
```

### Keyboard Shortcut Test

```go
func TestSelectAllShortcut(t *testing.T) {
    harness := helpers.NewTestHarness(t)
    defer harness.Cleanup()
    
    // Setup
    harness.App.SetSelectedMessages([]string{})
    
    // Execute shortcut
    harness.SimulateKeyEvent(tcell.KeyCtrlA, 0, tcell.ModCtrl)
    
    // Validate
    assert.Equal(t, 10, harness.App.GetSelectedCount())
}
```

### Visual Regression Test

```go
func TestMessageListRendering(t *testing.T) {
    harness := helpers.NewTestHarness(t)
    defer harness.Cleanup()
    
    // Setup and render
    component := harness.App.GetMessageListComponent()
    harness.DrawComponent(component)
    
    // Capture and compare snapshot
    snapshot := harness.GetScreenContent()
    snaps.MatchSnapshot(t, snapshot, "message_list_rendering")
}
```

## Test Organization

```
test/
├── helpers/           # Testing utilities and frameworks
├── integration/       # Integration tests
├── fixtures/          # Test data and VCR cassettes
└── main_test.go      # Main test suite runner
```

## Mocking Strategy

### Service Mocks

All services are mocked using `testify/mock` and generated with `mockery`:

```bash
# Generate mocks
mockery --config .mockery.yaml
```

### Mock Expectations

```go
// Setup expectations
harness.MockEmail.On("ArchiveMessage", mock.Anything, "msg_1").Return(nil)

// Execute operation
harness.App.ArchiveMessage("msg_1")

// Verify expectations
harness.MockEmail.AssertExpectations(t)
```

## CI/CD Integration

The framework includes a comprehensive CI/CD pipeline:

- **Matrix Testing** - Multiple Go versions and OS combinations
- **Automated Testing** - Unit, TUI, integration, and performance tests
- **Visual Regression** - Automated UI consistency checking
- **Security Scanning** - Vulnerability detection with Trivy
- **Coverage Reporting** - Code coverage tracking

### Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: test-mocks
        name: Generate Mocks
        entry: make test-mocks
        language: system
        types: [go]
        
      - id: test-unit
        name: Run Unit Tests
        entry: make test-unit
        language: system
        types: [go]
```

## Best Practices

### Test Structure

1. **Arrange** - Setup test harness and mocks
2. **Act** - Execute the operation being tested
3. **Assert** - Validate the expected outcome
4. **Cleanup** - Ensure proper resource cleanup

### Naming Conventions

- Test functions: `Test<Component><Operation>`
- Test cases: Descriptive names using underscores
- Snapshot keys: Descriptive and hierarchical

### Performance Considerations

- Use `t.Parallel()` for independent tests
- Minimize test data size
- Use appropriate timeouts for async operations
- Monitor goroutine leaks with `goleak`

### Error Handling

- Test both success and failure scenarios
- Validate error messages and types
- Test edge cases and boundary conditions
- Ensure proper cleanup on errors

## Troubleshooting

### Common Issues

1. **Mock Expectations Not Met**
   - Check mock setup order
   - Verify method signatures match
   - Ensure mocks are properly reset between tests

2. **Snapshot Mismatches**
   - Review snapshot differences
   - Update snapshots if changes are intentional
   - Check for flaky test data

3. **Goroutine Leaks**
   - Use `goleak.VerifyNone(t)` in async tests
   - Ensure proper context cancellation
   - Check for unbuffered channels

4. **Test Timeouts**
   - Increase timeout values for slow operations
   - Use `harness.WaitForCondition()` for async validation
   - Check for blocking operations

### Debugging Tips

- Use `harness.GetScreenContent()` to inspect UI state
- Enable verbose logging with `-v` flag
- Use `t.Log()` for additional test information
- Check mock call history with `ExpectedCalls`

## Contributing

### Adding New Tests

1. Follow the established patterns in existing test files
2. Use the test harness for consistent setup
3. Add appropriate mock expectations
4. Include both positive and negative test cases
5. Update documentation for new test types

### Test Maintenance

1. Keep tests focused and atomic
2. Regular snapshot review and cleanup
3. Monitor test execution time
4. Update mocks when interfaces change
5. Maintain test data consistency

## Multi-Account Testing

### Multi-Account Test Coverage

The test suite provides comprehensive coverage for multi-account functionality across multiple test layers:

#### 1. **Unit Tests** - Business Logic Core
**Location**: `internal/services/account_service_test.go`

- **Service Creation & Initialization**: Tests `NewAccountService` with various configuration scenarios
- **Account Management Operations**: CRUD operations for accounts
- **Account Switching Logic**: Proper activation/deactivation state management
- **Configuration Loading**: Multi-account vs legacy single-account configuration handling
- **Data Isolation**: Ensures returned account objects are copies, preventing data races
- **Thread Safety**: Concurrent access to account data with proper mutex protection
- **Error Scenarios**: Invalid configurations, missing accounts, duplicate IDs
- **Validation Logic**: Account field validation and constraint enforcement

```bash
# Run multi-account unit tests
make test-unit
go test ./internal/services -v -run "TestAccount"
```

#### 2. **Component Tests** - UI Behavior
**Location**: `internal/tui/accounts_test.go`

- **Account Picker UI**: Opening, closing, and navigation behavior
- **Account Display Logic**: Status icons, active indicators, filtering
- **User Interactions**: Keyboard navigation, account selection, management operations
- **State Management**: Picker state transitions and focus handling
- **Error Handling**: Service failures, UI error display, graceful degradation
- **Mock Integration**: Proper mocking of services for isolated UI testing

```bash
# Run multi-account component tests
go test ./internal/tui -v -run "TestApp.*Account"
```

#### 3. **Integration Tests** - End-to-End Workflows
**Location**: `test/integration/accounts_integration_test.go`

- **Keyboard Shortcuts**: Account picker shortcuts and configurable shortcuts
- **Command System**: `:accounts`, `:accounts switch`, etc.
- **Command Aliases**: Short forms like `:accounts sw`
- **Workflow Integration**: Complete user workflows from keyboard to service
- **UI-Service Integration**: Proper communication between UI components and services
- **Concurrent Operations**: Multiple simultaneous keyboard and command operations
- **State Consistency**: System state remains consistent across operations

```bash
# Run multi-account integration tests
make test-integration
go test ./test/integration -v -run "TestAccounts"
```

#### 4. **Configuration Tests** - Config Loading & Validation
**Location**: `internal/config/accounts_config_test.go`

- **Multi-Account Configuration**: Loading and parsing of account arrays
- **Legacy Compatibility**: Single-account configuration backward compatibility
- **Validation Rules**: Account ID uniqueness, required fields, active account limits
- **File Operations**: Config file reading, JSON parsing, error handling
- **Migration Logic**: Legacy to multi-account configuration migration
- **Default Values**: Proper application of default settings
- **Serialization**: Round-trip config save/load integrity

```bash
# Run multi-account config tests
go test ./internal/config -v -run "TestAccount"
```

#### 5. **Concurrency Tests** - Thread Safety & Performance
**Location**: `test/concurrency/accounts_concurrency_test.go`

- **Concurrent Reads**: Multiple simultaneous account listing and retrieval operations
- **Concurrent Writes**: Account switching, addition, removal under high load
- **Mixed Operations**: Combined read/write operations with race condition detection
- **UI Concurrency**: Concurrent UI operations mixed with service calls
- **Memory Consistency**: Data integrity under concurrent access
- **Deadlock Prevention**: Timeout handling and proper lock ordering
- **Performance Testing**: Throughput and latency under concurrent load
- **Race Detection**: Built-in race detector compatibility

```bash
# Run multi-account concurrency tests (with race detector)
go test -race ./test/concurrency -v -run "TestAccount"
```

### Running Multi-Account Tests

#### Complete Multi-Account Test Suite
```bash
# Run all multi-account tests
make test-all
go test -v ./internal/services ./internal/tui ./internal/config ./test/integration ./test/concurrency -run ".*Account.*"

# Run with race detector (recommended)
go test -race -v ./internal/services ./internal/tui ./internal/config ./test/integration ./test/concurrency -run ".*Account.*"

# Run with coverage
make test-coverage
go test -coverprofile=coverage.out ./internal/services ./internal/tui ./internal/config ./test/integration ./test/concurrency -run ".*Account.*"
```

#### Performance & Load Testing
```bash
# Run performance tests
go test -v ./test/concurrency -run "TestAccountService_PerformanceUnderLoad"

# Run with detailed timing
go test -v -timeout 30s ./test/concurrency -run "Performance"
```

### Multi-Account Test Quality Gates

#### Coverage Targets
- **Unit Test Coverage**: >90% for AccountService business logic
- **Integration Coverage**: >80% for end-to-end workflows  
- **Error Path Coverage**: >95% for error handling scenarios
- **Concurrency Coverage**: >85% for thread-safe operations

#### Quality Assertions
- **Zero Race Conditions**: All tests must pass with `-race` flag
- **No Memory Leaks**: Verified through long-running concurrency tests
- **Consistent State**: System state verification after all operations
- **Error Recovery**: Proper cleanup and recovery from all error scenarios

#### Performance Benchmarks
- **Response Time**: <100ms for typical account operations
- **Throughput**: >100 operations/second under concurrent load
- **Memory Usage**: Stable memory usage under sustained load
- **Error Rate**: <1% under normal operating conditions

---

This testing framework provides a robust foundation for ensuring the quality and reliability of GizTUI. The emphasis on component-level testing, comprehensive mocking, and visual regression detection makes it particularly well-suited for TUI applications where traditional testing approaches fall short.