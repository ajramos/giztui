# Multi-Account Support Testing Guide

This document provides a comprehensive overview of the test suite created for GizTUI's multi-account support implementation.

## 📋 Test Coverage Overview

The test suite provides comprehensive coverage for all aspects of multi-account functionality:

### 1. **Unit Tests** - Business Logic Core
**Location**: `internal/services/account_service_test.go`

- **Service Creation & Initialization**: Tests `NewAccountService` with various configuration scenarios
- **Account Management Operations**: CRUD operations (Create, Read, Update, Delete) for accounts
- **Account Switching Logic**: Proper activation/deactivation state management
- **Configuration Loading**: Multi-account vs legacy single-account configuration handling
- **Data Isolation**: Ensures returned account objects are copies, preventing data races
- **Thread Safety**: Concurrent access to account data with proper mutex protection
- **Error Scenarios**: Invalid configurations, missing accounts, duplicate IDs
- **Validation Logic**: Account field validation and constraint enforcement

**Key Test Categories**:
```bash
# Run unit tests
make test-unit
go test ./internal/services -v -run "TestAccount"
```

### 2. **Component Tests** - UI Behavior
**Location**: `internal/tui/accounts_test.go`

- **Account Picker UI**: Opening, closing, and navigation behavior
- **Account Display Logic**: Status icons, active indicators, filtering
- **User Interactions**: Keyboard navigation, account selection, management operations
- **State Management**: Picker state transitions and focus handling
- **Error Handling**: Service failures, UI error display, graceful degradation
- **Mock Integration**: Proper mocking of services for isolated UI testing

**Key Test Categories**:
```bash
# Run component tests
go test ./internal/tui -v -run "TestApp.*Account"
```

### 3. **Integration Tests** - End-to-End Workflows
**Location**: `test/integration/accounts_integration_test.go`

- **Keyboard Shortcuts**: `Ctrl+U` and configurable shortcuts for account picker
- **Command System**: `:accounts`, `:accounts switch`, etc.
- **Command Aliases**: Short forms like `:accounts sw`
- **Workflow Integration**: Complete user workflows from keyboard to service
- **UI-Service Integration**: Proper communication between UI components and services
- **Concurrent Operations**: Multiple simultaneous keyboard and command operations
- **State Consistency**: System state remains consistent across operations

**Key Test Categories**:
```bash
# Run integration tests
make test-integration
go test ./test/integration -v -run "TestAccounts"
```

### 4. **Configuration Tests** - Config Loading & Validation
**Location**: `internal/config/accounts_config_test.go`

- **Multi-Account Configuration**: Loading and parsing of account arrays
- **Legacy Compatibility**: Single-account configuration backward compatibility
- **Validation Rules**: Account ID uniqueness, required fields, active account limits
- **File Operations**: Config file reading, JSON parsing, error handling
- **Migration Logic**: Legacy to multi-account configuration migration
- **Default Values**: Proper application of default settings
- **Serialization**: Round-trip config save/load integrity

**Key Test Categories**:
```bash
# Run config tests
go test ./internal/config -v -run "TestAccount"
```

### 5. **Error Handling Tests** - Comprehensive Error Scenarios
**Location**: `test/errors/accounts_error_handling_test.go`

- **Network Failures**: Timeout handling, connection errors, API failures
- **File System Errors**: Missing credentials, permission denied, corrupted files
- **Service Errors**: Gmail API authentication failures, quota exceeded
- **Configuration Errors**: Malformed config, missing required fields
- **Edge Cases**: Empty account lists, nil services, concurrent errors
- **Recovery Scenarios**: Transient error recovery, partial failures
- **User Feedback**: Appropriate error message display and logging

**Key Test Categories**:
```bash
# Run error handling tests
go test ./test/errors -v -run "TestAccount"
```

### 6. **Concurrency Tests** - Thread Safety & Performance
**Location**: `test/concurrency/accounts_concurrency_test.go`

- **Concurrent Reads**: Multiple simultaneous account listing and retrieval operations
- **Concurrent Writes**: Account switching, addition, removal under high load
- **Mixed Operations**: Combined read/write operations with race condition detection
- **UI Concurrency**: Concurrent UI operations mixed with service calls
- **Memory Consistency**: Data integrity under concurrent access
- **Deadlock Prevention**: Timeout handling and proper lock ordering
- **Performance Testing**: Throughput and latency under concurrent load
- **Race Detection**: Built-in race detector compatibility

**Key Test Categories**:
```bash
# Run concurrency tests (with race detector)
go test -race ./test/concurrency -v -run "TestAccount"
```

## 🚀 Running the Complete Test Suite

### Individual Test Categories

```bash
# Unit tests only
make test-unit
go test ./internal/services -v -run "TestAccount.*"

# Component tests only  
go test ./internal/tui -v -run "TestApp.*Account"

# Integration tests only
make test-integration
go test ./test/integration -v -run "TestAccounts.*"

# Configuration tests only
go test ./internal/config -v -run "TestAccount.*"

# Error handling tests only
go test ./test/errors -v -run "TestAccount.*"

# Concurrency tests only (with race detector)
go test -race ./test/concurrency -v -run "TestAccount.*"
```

### Complete Multi-Account Test Suite

```bash
# Run all multi-account tests
make test-all
go test -v ./internal/services ./internal/tui ./internal/config ./test/integration ./test/errors ./test/concurrency -run ".*Account.*"

# Run with race detector (recommended)
go test -race -v ./internal/services ./internal/tui ./internal/config ./test/integration ./test/errors ./test/concurrency -run ".*Account.*"

# Run with coverage
make test-coverage
go test -coverprofile=coverage.out ./internal/services ./internal/tui ./internal/config ./test/integration ./test/errors ./test/concurrency -run ".*Account.*"
```

## 📊 Test Architecture & Patterns

### Mock Services
- **Controllable Behavior**: Mocks allow testing both success and failure scenarios
- **State Simulation**: Mock services maintain state for realistic testing
- **Error Injection**: Configurable error conditions for comprehensive error testing
- **Threading Support**: Thread-safe mocks for concurrency testing

### Test Harnesses
- **Integration Harness**: Complete test environment setup with mocked dependencies
- **Concurrency Harness**: Specialized framework for concurrent operation testing
- **Error Scenario Harness**: Systematic error condition simulation

### Assertion Patterns
- **State Verification**: Comprehensive state checks after operations
- **Message Verification**: Error handler message content and timing validation
- **Thread Safety**: Race condition detection and memory consistency verification
- **Performance Metrics**: Throughput and latency assertion for performance tests

## ⚡ Performance & Load Testing

### Concurrency Test Scenarios
- **High Contention**: Multiple workers accessing shared resources
- **Mixed Workloads**: Combined read/write operations with realistic ratios  
- **UI Integration**: Concurrent UI operations mixed with service calls
- **Memory Pressure**: Large numbers of concurrent operations

### Performance Benchmarks
```bash
# Run performance tests
go test -v ./test/concurrency -run "TestAccountService_PerformanceUnderLoad"

# Run with detailed timing
go test -v -timeout 30s ./test/concurrency -run "Performance"
```

## 🔍 Debugging & Troubleshooting

### Verbose Test Output
```bash
# Run with verbose logging
go test -v -args -test.verbose ./internal/services -run "TestAccount"

# Run specific test with detailed output
go test -v ./internal/services -run "TestAccountService_SwitchAccount/switches_to_existing_account"
```

### Race Condition Detection
```bash
# Enable race detector (essential for concurrency testing)
go test -race ./test/concurrency

# Race detector with verbose output
go test -race -v ./test/concurrency -run "TestAccountService_ConcurrentMixedOperations"
```

### Coverage Analysis
```bash
# Generate coverage report
make test-coverage
go tool cover -html=coverage.out -o coverage.html

# View coverage in browser
open coverage.html
```

## 📈 Test Metrics & Quality Gates

### Coverage Targets
- **Unit Test Coverage**: >90% for AccountService business logic
- **Integration Coverage**: >80% for end-to-end workflows  
- **Error Path Coverage**: >95% for error handling scenarios
- **Concurrency Coverage**: >85% for thread-safe operations

### Quality Assertions
- **Zero Race Conditions**: All tests must pass with `-race` flag
- **No Memory Leaks**: Verified through long-running concurrency tests
- **Consistent State**: System state verification after all operations
- **Error Recovery**: Proper cleanup and recovery from all error scenarios

### Performance Benchmarks
- **Response Time**: <100ms for typical account operations
- **Throughput**: >100 operations/second under concurrent load
- **Memory Usage**: Stable memory usage under sustained load
- **Error Rate**: <1% under normal operating conditions

## 🎯 Maintenance & Best Practices

### Adding New Tests
1. **Follow Existing Patterns**: Use established mock and assertion patterns
2. **Test Categories**: Classify tests as unit, integration, or concurrency
3. **Error Scenarios**: Include both success and failure cases
4. **Documentation**: Document complex test scenarios and setup requirements

### Mock Updates
- **Interface Changes**: Update mocks when service interfaces change
- **Behavior Consistency**: Ensure mock behavior matches real service behavior
- **Error Scenarios**: Keep error scenarios realistic and representative

### Continuous Integration
```bash
# CI pipeline test commands
make test-all          # Complete test suite
make test-race        # Race condition detection  
make test-coverage    # Coverage reporting
make lint             # Code quality checks
```

This comprehensive test suite ensures the multi-account support implementation is robust, thread-safe, and maintains high quality across all scenarios. The tests serve as both quality gates and living documentation of expected behavior.