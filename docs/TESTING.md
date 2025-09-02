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

---

This testing framework provides a robust foundation for ensuring the quality and reliability of GizTUI. The emphasis on component-level testing, comprehensive mocking, and visual regression detection makes it particularly well-suited for TUI applications where traditional testing approaches fall short.