# GizTUI Makefile

.PHONY: help build run test clean lint fmt vet coverage install deps theme-demo version release release-build cross-build

# Variables
BINARY_NAME=giztui
BUILD_DIR=build
MAIN_PATH=cmd/giztui/main.go

# Version information
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
BUILD_USER ?= $(shell whoami)

# Linker flags for version injection
LDFLAGS = -w -s \
	-X 'github.com/ajramos/giztui/internal/version.Version=$(VERSION)' \
	-X 'github.com/ajramos/giztui/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/ajramos/giztui/internal/version.GitBranch=$(GIT_BRANCH)' \
	-X 'github.com/ajramos/giztui/internal/version.BuildDate=$(BUILD_DATE)' \
	-X 'github.com/ajramos/giztui/internal/version.BuildUser=$(BUILD_USER)'

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

help: ## Show this help
	@echo "$(GREEN)GizTUI - Available commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

deps: ## Install dependencies
	@echo "$(GREEN)Installing dependencies...$(NC)"
	go mod tidy
	go mod download

build: deps ## Build the application with version injection
	@echo "$(GREEN)Building $(BINARY_NAME) v$(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Built $(BUILD_DIR)/$(BINARY_NAME) v$(VERSION)$(NC)"

run: build ## Run the application
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME)

install: ## Install the application
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	go install $(MAIN_PATH)

test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v ./internal/... ./test/helpers ./test ./pkg/...

test-race: ## Run tests with race detector
	@echo "$(GREEN)Running tests with race detector...$(NC)"
	go test -race -v ./...

coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated in coverage.html$(NC)"

lint: ## Run linting (requires golangci-lint)
	@echo "$(GREEN)Running linting...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint is not installed. Install it with:$(NC)"; \
		echo "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	go fmt ./...

vet: ## Verify code
	@echo "$(GREEN)Verifying code...$(NC)"
	go vet ./...

clean: ## Clean generated files
	@echo "$(GREEN)Cleaning...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean

dev: ## Development mode (build and run)
	@echo "$(GREEN)Development mode...$(NC)"
	@make build
	@make run

# Examples / Demos
theme-demo: deps ## Run the theme system demo (preview and validate themes)
	@echo "$(GREEN)Running theme demo...$(NC)"
	go run ./examples/theme_demo.go

# Legacy testing commands (replaced by more specific ones below)
# test-unit and test-integration moved to testing section below

# Version commands
version: ## Show version information
	@echo "$(GREEN)Version Information:$(NC)"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Build User: $(BUILD_USER)"

# Release commands
release-build: clean deps test ## Build release binaries for all platforms
	@echo "$(GREEN)Building release binaries for v$(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)

	@echo "$(YELLOW)Building Linux AMD64...$(NC)"
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

	@echo "$(YELLOW)Building Linux ARM64...$(NC)"
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

	@echo "$(YELLOW)Building macOS AMD64...$(NC)"
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)

	@echo "$(YELLOW)Building macOS ARM64...$(NC)"
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

	@echo "$(YELLOW)Building Windows AMD64...$(NC)"
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

	@echo "$(YELLOW)Building Windows ARM64...$(NC)"
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PATH)

	@echo "$(GREEN)Generating checksums...$(NC)"
	cd $(BUILD_DIR) && sha256sum * > checksums.txt

	@echo "$(GREEN)Release binaries built in $(BUILD_DIR)/$(NC)"
	@echo "$(YELLOW)Files:$(NC)"
	@ls -la $(BUILD_DIR)/

cross-build: ## Build for multiple platforms (same as release-build)
	@make release-build

release: release-build ## Prepare release (build binaries and generate archives)
	@echo "$(GREEN)Creating release archives...$(NC)"
	cd $(BUILD_DIR) && \
		tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
		tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
		tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
		tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
		zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe && \
		zip $(BINARY_NAME)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe

	@echo "$(GREEN)Generating archive checksums...$(NC)"
	cd $(BUILD_DIR) && sha256sum *.tar.gz *.zip > archive-checksums.txt

	@echo "$(GREEN)Release v$(VERSION) prepared successfully!$(NC)"
	@echo "$(YELLOW)Archives created:$(NC)"
	@ls -la $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip

# Debugging commands
debug: ## Build with debug information
	@echo "$(GREEN)Building with debug information...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_PATH)
	@echo "$(GREEN)Debug binary in $(BUILD_DIR)/$(BINARY_NAME)-debug$(NC)"

# Documentation commands
docs: ## Generate documentation
	@echo "$(GREEN)Generating documentation...$(NC)"
	@if command -v godoc >/dev/null 2>&1; then \
		echo "$(YELLOW)Running godoc on http://localhost:6060$(NC)"; \
		godoc -http=:6060; \
	else \
		echo "$(YELLOW)godoc is not installed. Install it with:$(NC)"; \
		echo "go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Profiling commands
profile: build ## Run with profiling
	@echo "$(GREEN)Running with profiling...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) -cpuprofile=cpu.prof -memprofile=mem.prof

# Benchmarking commands
bench: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(NC)"
	go test -bench=. ./...

# Dependency verification commands
check-deps: ## Verify dependencies
	@echo "$(GREEN)Verifying dependencies...$(NC)"
	go mod verify
	go list -m all

# Dependency update commands
update-deps: ## Update dependencies
	@echo "$(GREEN)Updating dependencies...$(NC)"
	go get -u ./...
	go mod tidy

# Developer setup commands
.PHONY: setup-hooks check-hooks remove-hooks pre-commit-check

setup-hooks: ## Install and configure pre-commit hooks
	@echo "$(GREEN)Setting up pre-commit hooks...$(NC)"
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "$(GREEN)Pre-commit hooks installed successfully$(NC)"; \
		echo "$(YELLOW)Run 'make check-hooks' to test the hooks$(NC)"; \
	else \
		echo "$(YELLOW)pre-commit is not installed. Install it with:$(NC)"; \
		echo "pip install pre-commit"; \
		echo "$(YELLOW)Then run 'make setup-hooks' again$(NC)"; \
	fi

check-hooks: ## Run pre-commit hooks on all files
	@echo "$(GREEN)Running pre-commit hooks on all files...$(NC)"
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "$(RED)pre-commit is not installed. Run 'make setup-hooks' first$(NC)"; \
		exit 1; \
	fi

remove-hooks: ## Remove pre-commit hooks
	@echo "$(GREEN)Removing pre-commit hooks...$(NC)"
	@if [ -f .git/hooks/pre-commit ]; then \
		rm .git/hooks/pre-commit; \
		echo "$(GREEN)Pre-commit hooks removed$(NC)"; \
	else \
		echo "$(YELLOW)No pre-commit hooks found$(NC)"; \
	fi

pre-commit-check: ## Run the same checks as CI locally (comprehensive check)
	@echo "$(GREEN)Running comprehensive pre-commit checks...$(NC)"
	@echo "$(YELLOW)1. Format check...$(NC)"
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "$(RED)Code formatting issues found:$(NC)"; \
		gofmt -s -l .; \
		echo "$(YELLOW)Run 'make fmt' to fix$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)2. Go vet...$(NC)"
	@go vet -composites=false ./...
	@echo "$(YELLOW)3. Golangci-lint...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci.yml; \
	else \
		echo "$(YELLOW)golangci-lint not found, skipping$(NC)"; \
	fi
	@echo "$(YELLOW)4. Essential tests...$(NC)"
	@go test -timeout 30s ./internal/config ./pkg/auth || exit 1
	@echo "$(GREEN)All pre-commit checks passed!$(NC)"

# Testing commands
.PHONY: test test-unit test-integration test-tui test-coverage test-mocks test-snapshots test-all

# Generate mocks for testing
test-mocks: ## Generate mocks using mockery
	@echo "$(GREEN)Generating mocks for testing...$(NC)"
	@echo "$(YELLOW)Cleaning existing mocks...$(NC)"
	@rm -rf internal/services/mocks
	@mkdir -p internal/services/mocks
	@MOCKERY_CMD=""; \
	if command -v mockery >/dev/null 2>&1; then \
		MOCKERY_CMD="mockery"; \
	elif [ -f $$HOME/go/bin/mockery ]; then \
		MOCKERY_CMD="$$HOME/go/bin/mockery"; \
	elif [ -f $$(go env GOPATH)/bin/mockery ]; then \
		MOCKERY_CMD="$$(go env GOPATH)/bin/mockery"; \
	fi; \
	if [ -n "$$MOCKERY_CMD" ]; then \
		$$MOCKERY_CMD --dir=internal/services --name=EmailService --output=internal/services/mocks --outpkg=mocks --filename=email_service.go; \
		$$MOCKERY_CMD --dir=internal/services --name=AIService --output=internal/services/mocks --outpkg=mocks --filename=ai_service.go; \
		$$MOCKERY_CMD --dir=internal/services --name=LabelService --output=internal/services/mocks --outpkg=mocks --filename=label_service.go; \
		$$MOCKERY_CMD --dir=internal/services --name=CacheService --output=internal/services/mocks --outpkg=mocks --filename=cache_service.go; \
		$$MOCKERY_CMD --dir=internal/services --name=MessageRepository --output=internal/services/mocks --outpkg=mocks --filename=message_repository.go; \
		$$MOCKERY_CMD --dir=internal/services --name=SearchService --output=internal/services/mocks --outpkg=mocks --filename=search_service.go; \
		echo "$(GREEN)Mocks generated successfully$(NC)"; \
	else \
		echo "$(YELLOW)mockery is not installed. Install it with:$(NC)"; \
		echo "go install github.com/vektra/mockery/v2@latest"; \
		echo "$(YELLOW)Note: You may need to add your Go bin directory to PATH:$(NC)"; \
		echo "export PATH=\$$PATH:\$$(go env GOPATH)/bin"; \
	fi

# Run unit tests
test-unit: ## Run unit tests
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test -v ./internal/services/... -race

# Run TUI component tests
test-tui: ## Run TUI component tests
	@echo "$(GREEN)Running TUI component tests...$(NC)"
	go test -v ./test/helpers/... -race

# Run integration tests
test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	go test -v ./test/integration/... -race

# Run all tests with coverage
test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./internal/... ./test/helpers ./test ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Update snapshots (use with caution)
test-snapshots-update: ## Update test snapshots
	@echo "$(GREEN)Updating test snapshots...$(NC)"
	go test -v ./test/helpers/... -update

# Run all tests
test-all: test-mocks test-unit test-tui test-integration test-coverage ## Run all tests

# Test specific component
test-messages: ## Test message handling
	@echo "$(GREEN)Testing message handling...$(NC)"
	go test -v ./internal/tui/messages* -race

test-labels: ## Test label management
	@echo "$(GREEN)Testing label management...$(NC)"
	go test -v ./internal/tui/labels* -race

test-ai: ## Test AI features
	@echo "$(GREEN)Testing AI features...$(NC)"
	go test -v ./internal/tui/ai* -race

# Performance testing
test-performance: ## Run performance tests
	@echo "$(GREEN)Running performance tests...$(NC)"
	go test -v -bench=. -benchmem ./test/performance/...

# Load testing
test-load: ## Run load tests
	@echo "$(GREEN)Running load tests...$(NC)"
	go test -v -bench=BenchmarkBulkOperations -benchtime=30s ./test/helpers/...

# Legacy mock generation commands (requires mockgen)
mocks: ## Generate mocks (legacy)
	@echo "$(GREEN)Generating mocks...$(NC)"
	@if command -v mockgen >/dev/null 2>&1; then \
		mockgen -source=internal/gmail/client.go -destination=internal/gmail/mocks.go; \
		mockgen -source=internal/llm/ollama.go -destination=internal/llm/mocks.go; \
	else \
		echo "$(YELLOW)mockgen is not installed. Install it with:$(NC)"; \
		echo "go install github.com/golang/mock/mockgen@latest"; \
	fi
