# 🛠️ Development Setup Guide

This guide helps new contributors set up their development environment for GizTUI development.

## 🚀 Quick Start

For experienced developers who want to get started immediately:

```bash
git clone https://github.com/ajramos/giztui.git
cd giztui
./scripts/setup-dev.sh
```

The setup script installs pinned quality tools and runs focused checks. Run
`make ci` afterward to verify the complete local product gate.

## 📋 Prerequisites

Ensure you have the following installed:

- **Go 1.25.13+** - [Download from go.dev](https://go.dev/dl/)
- **Git** - [Download from git-scm.com](https://git-scm.com/)
- **Python 3 with `venv` and pip** - Required for the pinned lizard environment
- **Node.js 20.19+ or 22.12+ and npm** - Required by the canonical gate

### Development Tools
- **pre-commit** - Optional local hook runner
- **Pinned CI tools** - Installed into isolated/local tool locations by `make ci-tools`

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
# Install the exact linter, scanner, workflow checker, and lizard versions used by CI
make ci-tools
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
# Run the complete local product gate
make ci

# Run tests
make test

# Build project
make build
```

## 🎯 Development Workflow

### Pre-commit Checks (Automatic)

The optional pre-commit hooks provide fast feedback. They check:

- **Code formatting** - Runs `gofmt` to ensure consistent formatting
- **Linting** - Runs `golangci-lint` to catch common issues
- **Go vet** - Checks for suspicious constructs
- **Essential tests** - Runs core tests to catch breaking changes

They are intentionally smaller than `make ci` and do not replace it.

### Manual Quality Checks

Run the canonical local checks manually:

```bash
# Run the complete local CI gate
make ci

# Run individual checks
make fmt                    # Format code
make lint                   # Run linting
make vet                    # Run go vet
make pre-commit-check       # Alias for the complete canonical CI gate
```

### Testing

```bash
# Run the canonical product gate, including desktop unit and E2E tests
make ci

# Run specific test types
make test-unit              # Unit tests only
make test-tui               # TUI component tests
make test-integration       # Integration tests

# Run with coverage
make test-coverage

# Regenerate checked-in mocks only when a covered interface changes
# (requires the pinned mockery version printed by the target)
make test-mocks
```

### Building and Running

```bash
# Build development version (with full build metadata)
make build

# Build and run
make dev

# Build and run (the run target depends on build)
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
│   └── desktop/        # Front-end-independent desktop API and DTOs
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
# Run tests in watch mode during development
go test -v ./internal/services/... -run TestMyFeature
```

## 🚦 CI/CD Pipeline

Our comprehensive pipeline runs:

1. **Code Quality**: Format check, linting, security scan
2. **Testing**: Root Go, race, desktop Go, Vitest, and Playwright tests
3. **Cross-platform Testing**: Ubuntu and macOS
4. **Security Analysis**: Vulnerability scanning and dependency review
5. **Required Result**: Stable `required` aggregator over every mandatory job

### Pipeline Files

- `.github/workflows/ci-comprehensive.yml` - Main CI/CD pipeline
- `.pre-commit-config.yaml` - Pre-commit hook configuration
- `.golangci.yml` - Linting configuration

The pre-commit hooks provide quick feedback; `make ci` remains the completion
gate.

## 🛠️ Useful Make Commands

```bash
# Development
make help                   # Show all available commands
make dev                    # Build and run in development mode
make clean                  # Clean build artifacts

# Quality & Testing  
make ci                     # Run the canonical fail-closed local product gate
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
- **Import organization**: Keep imports in `gofmt`-compatible groups
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
# Re-run without cached test results
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
3. **Test locally**: `make pre-commit-check`
4. **Commit changes**: Pre-commit hooks run automatically
5. **Push and create PR**: CI/CD pipeline runs comprehensive checks
6. **Address feedback**: Iterate until approved
7. **Merge**: Maintainer merges after approval

---

**Ready to contribute? 🚀**

You're now set up for efficient GizTUI development. The pre-commit hooks will keep your code quality high, and the comprehensive test suite will catch regressions early.

Happy coding!
