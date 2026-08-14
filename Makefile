# GizTUI Makefile

.PHONY: help build run test clean lint fmt vet coverage install deps tidy theme-demo version release release-build cross-build

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
GO_BIN ?= $(shell go env GOPATH)/bin
GOLANGCI_LINT_VERSION ?= v2.6.2
GOVULNCHECK_VERSION ?= v1.7.0
ACTIONLINT_VERSION ?= v1.7.12
MOCKERY_VERSION ?= v2.53.5
GOLANGCI_LINT_BIN ?= $(GO_BIN)/golangci-lint
GOVULNCHECK_BIN ?= $(GO_BIN)/govulncheck
ACTIONLINT_BIN ?= $(GO_BIN)/actionlint
MOCKERY_BIN ?= $(GO_BIN)/mockery
CI_VENV ?= .venv-ci
CI_PYTHON ?= $(CI_VENV)/bin/python
PLAYWRIGHT_INSTALL_ARGS ?= chromium

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
	go mod download

tidy: ## Normalize module files after dependency changes
	@echo "$(GREEN)Tidying Go modules...$(NC)"
	go mod tidy
	cd desktop && go mod tidy

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
	@test -x "$(GOLANGCI_LINT_BIN)" || { echo "$(RED)Missing pinned golangci-lint; run 'make ci-tools'$(NC)"; exit 1; }
	$(GOLANGCI_LINT_BIN) config verify
	$(GOLANGCI_LINT_BIN) run --config=.golangci.yml

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
.PHONY: setup-hooks check-hooks remove-hooks pre-commit-check ci ci-tools ci-tools-check ci-quality ci-go ci-desktop ci-security ci-architecture quality-baseline-update

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

ci-tools: ## Install the pinned tools used by local and CI validation
	@echo "$(GREEN)Installing pinned CI tools...$(NC)"
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	go install github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)
	python3 -m venv "$(CI_VENV)"
	$(CI_PYTHON) -m pip install -r requirements-ci.txt

ci-tools-check:
	@test -x "$(GOLANGCI_LINT_BIN)" || { echo "$(RED)Missing golangci-lint $(GOLANGCI_LINT_VERSION); run 'make ci-tools'$(NC)"; exit 1; }
	@test -x "$(GOVULNCHECK_BIN)" || { echo "$(RED)Missing govulncheck $(GOVULNCHECK_VERSION); run 'make ci-tools'$(NC)"; exit 1; }
	@test -x "$(ACTIONLINT_BIN)" || { echo "$(RED)Missing actionlint $(ACTIONLINT_VERSION); run 'make ci-tools'$(NC)"; exit 1; }
	@test -x "$(CI_PYTHON)" || { echo "$(RED)Missing CI virtualenv; run 'make ci-tools'$(NC)"; exit 1; }
	@$(GOLANGCI_LINT_BIN) version | grep -q "version $(patsubst v%,%,$(GOLANGCI_LINT_VERSION))" || { echo "$(RED)Wrong golangci-lint version; run 'make ci-tools'$(NC)"; exit 1; }
	@$(GOVULNCHECK_BIN) -version | grep -q "Scanner: govulncheck@$(GOVULNCHECK_VERSION)" || { echo "$(RED)Wrong govulncheck version; run 'make ci-tools'$(NC)"; exit 1; }
	@$(ACTIONLINT_BIN) -version | grep -q "$(patsubst v%,%,$(ACTIONLINT_VERSION))" || { echo "$(RED)Wrong actionlint version; run 'make ci-tools'$(NC)"; exit 1; }
	@$(CI_PYTHON) -c 'import lizard; assert lizard.version == "1.23.0"' || { echo "$(RED)Wrong lizard version; run 'make ci-tools'$(NC)"; exit 1; }

ci-quality: ci-tools-check ## Run formatting, vet, and pinned lint checks
	@echo "$(GREEN)Running Go quality checks...$(NC)"
	@test -z "$$(gofmt -s -l .)" || { echo "$(RED)Go files require formatting:$(NC)"; gofmt -s -l .; exit 1; }
	go vet -composites=false ./...
	$(GOLANGCI_LINT_BIN) config verify
	$(GOLANGCI_LINT_BIN) run --config=.golangci.yml
	$(ACTIONLINT_BIN) .github/workflows/*.yml
	scripts/test-release-scripts.sh

ci-go: ## Run the complete root Go build, test, race, and coverage gate
	@echo "$(GREEN)Running root Go gate...$(NC)"
	go mod verify
	go build ./...
	go test -timeout 5m -coverprofile=coverage.out ./internal/... ./test/helpers ./test ./pkg/...
	scripts/check-coverage-ratchet.sh coverage.out
	go test -race -timeout 10m ./internal/services ./internal/tui ./pkg/desktop

ci-desktop: ## Run frontend and nested desktop module validation in dependency order
	@echo "$(GREEN)Running desktop gate...$(NC)"
	cd desktop/frontend && npm ci
	cd desktop/frontend && npm audit --package-lock-only --audit-level=high
	cd desktop/frontend && npm test
	cd desktop/frontend && npm run build
	cd desktop && go mod verify
	cd desktop && go test -timeout 5m ./...
	cd desktop && go build ./...
	cd desktop && go vet ./...
	cd desktop/frontend && npx playwright install $(PLAYWRIGHT_INSTALL_ARGS)
	cd desktop/frontend && npm run test:e2e

ci-security: ci-tools-check ## Run Go and npm vulnerability policy checks
	@echo "$(GREEN)Running vulnerability gates...$(NC)"
	$(GOVULNCHECK_BIN) ./...
	cd desktop && $(GOVULNCHECK_BIN) ./...
	cd desktop/frontend && npm audit --package-lock-only --audit-level=high

ci-architecture: ci-tools-check ## Run architecture and complexity ratchets
	@echo "$(GREEN)Running architecture and quality ratchets...$(NC)"
	scripts/check-architecture.sh
	$(CI_PYTHON) scripts/check-quality-ratchet.py

ci-docs: ci-tools-check ## Validate JSON, YAML, and markdown internal links
	@echo "$(GREEN)Running documentation gate...$(NC)"
	$(CI_PYTHON) scripts/check-docs.py

quality-baseline-update: ci-tools-check ## Deliberately accept the reviewed current source metrics
	$(CI_PYTHON) scripts/check-quality-ratchet.py --write
	$(CI_PYTHON) scripts/check-quality-ratchet.py

ci: ## Run the same fail-closed product gate used by CI
	@$(MAKE) ci-quality
	@$(MAKE) ci-go
	@$(MAKE) ci-security
	@$(MAKE) ci-architecture
	@$(MAKE) ci-docs
	@$(MAKE) ci-desktop
	@echo "$(GREEN)Canonical CI gate passed.$(NC)"

pre-commit-check: ci ## Run the canonical CI gate locally

# Testing commands
.PHONY: test test-unit test-integration test-tui test-coverage test-mocks test-snapshots-update test-all

# Generate mocks for testing
test-mocks: ## Generate mocks using mockery
	@echo "$(GREEN)Generating mocks for testing...$(NC)"
	@test -x "$(MOCKERY_BIN)" || { echo "$(RED)Missing mockery $(MOCKERY_VERSION); install with 'go install github.com/vektra/mockery/v2@$(MOCKERY_VERSION)'$(NC)"; exit 1; }
	@echo "$(YELLOW)Cleaning existing mocks...$(NC)"
	@rm -rf internal/services/mocks
	@mkdir -p internal/services/mocks
	@$(MOCKERY_BIN) --dir=internal/services --name=EmailService --output=internal/services/mocks --outpkg=mocks --filename=email_service.go
	@$(MOCKERY_BIN) --dir=internal/services --name=AIService --output=internal/services/mocks --outpkg=mocks --filename=ai_service.go
	@$(MOCKERY_BIN) --dir=internal/services --name=LabelService --output=internal/services/mocks --outpkg=mocks --filename=label_service.go
	@$(MOCKERY_BIN) --dir=internal/services --name=CacheService --output=internal/services/mocks --outpkg=mocks --filename=cache_service.go
	@$(MOCKERY_BIN) --dir=internal/services --name=MessageRepository --output=internal/services/mocks --outpkg=mocks --filename=message_repository.go
	@$(MOCKERY_BIN) --dir=internal/services --name=SearchService --output=internal/services/mocks --outpkg=mocks --filename=search_service.go
	@$(MOCKERY_BIN) --dir=internal/services --name=PromptGeneratorService --output=internal/services/mocks --outpkg=mocks --filename=PromptGeneratorService.go
	@$(MOCKERY_BIN) --dir=internal/services --name=DeterministicRulesService --output=internal/services/mocks --outpkg=mocks --filename=deterministic_rules_service.go
	@echo "$(GREEN)Mocks generated successfully$(NC)"

# Run unit tests
test-unit: ## Run unit tests
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test -v ./internal/services/... -race

# Run TUI component tests
test-tui: ## Run TUI component tests
	@echo "$(GREEN)Running TUI component tests...$(NC)"
	go test -v ./internal/tui ./test/helpers/... -race

# Run integration tests
test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	go test -v ./test/... -race

# Run all tests with coverage
test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./internal/... ./test/helpers ./test ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Update snapshots (use with caution)
test-snapshots-update: ## Update test snapshots
	@echo "$(GREEN)Updating test snapshots...$(NC)"
	UPDATE_SNAPSHOTS=true go test -v ./test/helpers/...

# Run all tests
test-all: test-mocks test-unit test-tui test-integration test-coverage ## Run all tests

# Test specific component
test-messages: ## Test message handling
	@echo "$(GREEN)Testing message handling...$(NC)"
	go test -v ./internal/tui -run 'Message' -race

test-labels: ## Test label management
	@echo "$(GREEN)Testing label management...$(NC)"
	go test -v ./internal/tui -run 'Label' -race

test-ai: ## Test AI features
	@echo "$(GREEN)Testing AI features...$(NC)"
	go test -v ./internal/tui -run 'AI|Summary|Prompt' -race

# Performance testing
test-performance: ## Run performance tests
	@echo "$(GREEN)Running performance tests...$(NC)"
	go test -run='^$$' -bench=. -benchmem ./...

# Load testing
test-load: ## Run load tests
	@echo "$(GREEN)Running load tests...$(NC)"
	go test -run='^$$' -bench=. -benchmem -benchtime=30s ./test/helpers/...

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
