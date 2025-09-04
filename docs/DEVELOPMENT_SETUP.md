# 🛠️ Development Setup Guide

This guide helps new contributors set up their development environment for GizTUI development.

## 🚀 Quick Start

For experienced developers who want to get started immediately:

```bash
git clone https://github.com/ajramos/giztui.git
cd giztui
./scripts/setup-dev.sh
```

The setup script will handle everything automatically. Skip to [Next Steps](#next-steps) after running it.

## 📋 Prerequisites

Ensure you have the following installed:

- **Go 1.23+** - [Download from golang.org](https://golang.org/dl/)
- **Git** - [Download from git-scm.com](https://git-scm.com/)
- **Python 3.7+** (for pre-commit hooks) - Usually pre-installed on macOS/Linux

### Optional Tools
- **golangci-lint** - Will be installed automatically
- **mockery** - Will be installed automatically
- **pre-commit** - Will be installed automatically via pip

## 🔧 Manual Setup Steps

If you prefer manual setup or the script didn't work:

### 1. Clone and Setup Repository

```bash
git clone https://github.com/ajramos/giztui.git
cd giztui
go mod download
```

### 2. Install Development Tools

```bash
# Install golangci-lint for linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0

# Install mockery for generating test mocks
go install github.com/vektra/mockery/v2@latest

# Install vulnerability checker
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### 3. Setup Pre-commit Hooks

```bash
# Install pre-commit (choose one method)
pip install pre-commit        # Using pip
pip3 install pre-commit       # Using pip3
brew install pre-commit       # Using Homebrew (macOS)

# Install hooks
make setup-hooks
```

### 4. Verify Setup

```bash
# Run comprehensive checks
make pre-commit-check

# Generate test mocks
make test-mocks

# Run tests
make test

# Build project
make build
```

## 🎯 Development Workflow

### Pre-commit Checks (Automatic)

Pre-commit hooks will automatically run before each commit to ensure code quality. They check:

- **Code formatting** - Runs `gofmt` to ensure consistent formatting
- **Linting** - Runs `golangci-lint` to catch common issues
- **Go vet** - Checks for suspicious constructs
- **Essential tests** - Runs core tests to catch breaking changes

### Manual Quality Checks

You can run the same checks manually:

```bash
# Run all pre-commit checks
make check-hooks

# Run individual checks
make fmt                    # Format code
make lint                   # Run linting
make vet                    # Run go vet
make pre-commit-check       # All CI checks locally
```

### Testing

```bash
# Run all tests
make test-all

# Run specific test types
make test-unit              # Unit tests only
make test-tui               # TUI component tests
make test-integration       # Integration tests

# Run with coverage
make test-coverage

# Generate test mocks (when interfaces change)
make test-mocks
```

### Building and Running

```bash
# Build development version (with full build metadata)
make build

# Build and run
make dev

# Run without building (during development)
make run

# Build release binaries for all platforms
make release-build
```

#### Build Method Differences

GizTUI supports multiple build methods with different version information:

- **`make build`**: Full build metadata (Git commit, branch, build date, build user)
- **`go install`**: Automatic Git commit detection via Go 1.18+ VCS support
- **`go build`**: Basic build (may show "unknown" for some fields depending on context)

When developing, prefer `make build` for complete version information in testing.

## 📁 Project Structure

Understanding the codebase structure:

```
giztui/
├── cmd/giztui/           # Main application entry point
├── internal/             # Internal packages (not imported externally)
│   ├── config/          # Configuration handling
│   ├── services/        # Business logic services
│   ├── tui/            # Terminal UI components  
│   └── version/        # Version information
├── pkg/                 # Public packages (can be imported)
│   ├── auth/           # OAuth authentication
│   └── utils/          # Utility functions
├── test/               # Test helpers and integration tests
├── docs/               # Documentation
├── scripts/            # Development and deployment scripts
└── .github/workflows/  # CI/CD pipelines
```

### Key Architecture Patterns

- **Service-First Development**: All business logic goes in `internal/services/`
- **Dependency Injection**: Services are injected, not instantiated directly
- **Thread Safety**: Use accessor methods, never direct field access
- **Error Handling**: Use `ErrorHandler` for all user feedback
- **Theming**: Use `GetComponentColors()` for consistent UI theming

📖 **Read [ARCHITECTURE.md](ARCHITECTURE.md) for detailed patterns and requirements.**

## 🧪 Testing Strategy

Our testing approach includes:

- **Unit Tests**: Test individual functions and methods
- **TUI Tests**: Test user interface components with visual snapshots
- **Integration Tests**: Test service interactions
- **Visual Regression**: Ensure UI changes don't break layouts
- **Goroutine Leak Detection**: Prevent resource leaks

### Writing Tests

```bash
# Create a new test file
touch internal/services/my_feature_test.go

# Generate mocks for interfaces
make test-mocks

# Run tests in watch mode during development
go test -v ./internal/services/... -run TestMyFeature
```

## 🚦 CI/CD Pipeline

Our comprehensive pipeline runs:

1. **Code Quality**: Format check, linting, security scan
2. **Testing**: Unit, TUI, integration, and visual regression tests  
3. **Cross-platform Testing**: Ubuntu and macOS
4. **Security Analysis**: Vulnerability scanning and dependency review
5. **Build Artifacts**: Multi-platform binaries on successful tests

### Pipeline Files

- `.github/workflows/ci-comprehensive.yml` - Main CI/CD pipeline
- `.pre-commit-config.yaml` - Pre-commit hook configuration
- `.golangci.yml` - Linting configuration

The pre-commit hooks ensure you catch issues locally before they reach CI/CD.

## 🛠️ Useful Make Commands

```bash
# Development
make help                   # Show all available commands
make dev                    # Build and run in development mode
make clean                  # Clean build artifacts

# Quality & Testing  
make pre-commit-check       # Run same checks as CI locally
make test-all              # Run complete test suite
make lint                  # Run linting only
make coverage              # Generate test coverage report

# Pre-commit Management
make setup-hooks           # Install pre-commit hooks
make check-hooks           # Run hooks on all files  
make remove-hooks          # Remove pre-commit hooks

# Building & Release
make build                 # Build development binary
make release-build         # Build for all platforms
make debug                 # Build with debug symbols
```

## 📝 Code Standards

### Formatting and Style
- **Go formatting**: Use `gofmt` (automated via pre-commit)
- **Import organization**: Use `goimports` 
- **Linting**: Follow `golangci-lint` rules in `.golangci.yml`
- **Error handling**: Always handle errors appropriately

### Git Workflow
- **Commit messages**: Use conventional commits (feat, fix, docs, etc.)
- **Branch naming**: Use descriptive names (feature/ai-summaries, fix/oauth-timeout)
- **Pre-commit hooks**: Never bypass without good reason (`git commit --no-verify`)

### Code Organization
- **Services first**: Put business logic in services, not UI components
- **Interface-driven**: Define interfaces for all services
- **Dependency injection**: Inject dependencies rather than creating them
- **Thread safety**: Use mutexes and channels appropriately

## 🆘 Troubleshooting

### Common Setup Issues

**Pre-commit hooks not working:**
```bash
# Reinstall hooks
make remove-hooks
make setup-hooks
```

**Go tools not in PATH:**
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

**Tests failing locally:**
```bash
# Clean and regenerate mocks
rm -rf internal/services/mocks
make test-mocks
go clean -testcache
make test
```

**Linting failures:**
```bash
# Run linter with more details
golangci-lint run --config=.golangci.yml -v

# Fix auto-fixable issues
golangci-lint run --fix
```

### Getting Help

- **Documentation**: Check files in `docs/` directory
- **Architecture**: Read [ARCHITECTURE.md](ARCHITECTURE.md) for development patterns
- **Testing**: See [TESTING.md](TESTING.md) for testing guidelines
- **Issues**: Create GitHub issue for bugs or feature requests

## 🔗 Next Steps

After setup is complete:

1. **Read the Architecture Guide**: [ARCHITECTURE.md](ARCHITECTURE.md)
2. **Understand Testing**: [TESTING.md](TESTING.md) 
3. **Learn the Codebase**: Start with `internal/services/interfaces.go`
4. **Pick an Issue**: Look for "good first issue" labels
5. **Make Your First Change**: Follow the development workflow

## 🎯 Development Workflow Summary

1. **Create feature branch**: `git checkout -b feature/my-feature`
2. **Make changes**: Follow architecture patterns
3. **Test locally**: `make pre-commit-check && make test-all`
4. **Commit changes**: Pre-commit hooks run automatically
5. **Push and create PR**: CI/CD pipeline runs comprehensive checks
6. **Address feedback**: Iterate until approved
7. **Merge**: Maintainer merges after approval

---

**Ready to contribute? 🚀**

You're now set up for efficient GizTUI development. The pre-commit hooks will keep your code quality high, and the comprehensive test suite will catch regressions early.

Happy coding!