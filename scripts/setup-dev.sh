#!/bin/bash

# GizTUI Developer Setup Script
# This script sets up the development environment for new contributors

set -e  # Exit on any error

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "\n${BLUE}=====================================${NC}"
    echo -e "${BLUE}🚀 GizTUI Developer Setup${NC}"
    echo -e "${BLUE}=====================================${NC}\n"
}

print_step() {
    echo -e "${GREEN}➤ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

check_command() {
    if command -v "$1" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

install_go_tool() {
    local tool=$1
    local package=$2

    print_step "Installing $tool..."
    if check_command "$tool"; then
        print_success "$tool is already installed"
    else
        go install "$package" || {
            print_error "Failed to install $tool"
            exit 1
        }
        print_success "$tool installed successfully"
    fi
}

# Main setup function
main() {
    print_header

    # Check Go installation
    print_step "Checking Go installation..."
    if ! check_command "go"; then
        print_error "Go is not installed. Please install Go 1.23+ from https://golang.org/dl/"
        exit 1
    fi

    GO_VERSION=$(go version | cut -d' ' -f3)
    print_success "Go is installed: $GO_VERSION"

    # Check Git installation
    print_step "Checking Git installation..."
    if ! check_command "git"; then
        print_error "Git is not installed. Please install Git first."
        exit 1
    fi
    print_success "Git is installed: $(git --version)"

    # Install dependencies
    print_step "Installing Go dependencies..."
    go mod download
    print_success "Dependencies installed"

    # Install development tools
    print_step "Installing development tools..."

    # golangci-lint
    install_go_tool "golangci-lint" "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0"

    # mockery for test mocks
    install_go_tool "mockery" "github.com/vektra/mockery/v2@latest"

    # govulncheck for security scanning
    install_go_tool "govulncheck" "golang.org/x/vuln/cmd/govulncheck@latest"

    # Setup pre-commit hooks
    print_step "Setting up pre-commit hooks..."
    if check_command "pre-commit"; then
        pre-commit install
        print_success "Pre-commit hooks installed"
    else
        print_warning "pre-commit is not installed. Installing via pip..."
        if check_command "pip3"; then
            pip3 install pre-commit
            pre-commit install
            print_success "Pre-commit hooks installed"
        elif check_command "pip"; then
            pip install pre-commit
            pre-commit install
            print_success "Pre-commit hooks installed"
        else
            print_warning "pip not found. Please install pre-commit manually:"
            echo "  pip install pre-commit"
            echo "  make setup-hooks"
        fi
    fi

    # Run initial checks
    print_step "Running initial quality checks..."

    # Format check
    if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
        print_warning "Code formatting issues found. Running go fmt..."
        go fmt ./...
        print_success "Code formatted"
    else
        print_success "Code is properly formatted"
    fi

    # Go vet
    print_step "Running go vet..."
    go vet -composites=false ./...
    print_success "Go vet passed"

    # Linting
    if check_command "golangci-lint"; then
        print_step "Running linting..."
        golangci-lint run --config=.golangci.yml
        print_success "Linting passed"
    fi

    # Generate mocks
    print_step "Generating test mocks..."
    if [ -f "Makefile" ]; then
        make test-mocks
        print_success "Test mocks generated"
    else
        print_warning "Makefile not found, skipping mock generation"
    fi

    # Run essential tests
    print_step "Running essential tests..."
    go test -timeout 30s ./internal/config ./pkg/auth
    print_success "Essential tests passed"

    # Build the project
    print_step "Building the project..."
    if [ -f "Makefile" ]; then
        make build
        print_success "Project built successfully"
    else
        go build -o build/giztui ./cmd/giztui
        print_success "Project built successfully"
    fi

    # Final summary
    echo -e "\n${GREEN}=====================================${NC}"
    echo -e "${GREEN}🎉 Development environment setup complete!${NC}"
    echo -e "${GREEN}=====================================${NC}\n"

    echo -e "${BLUE}📋 Next steps:${NC}"
    echo -e "  1. Run ${YELLOW}'make help'${NC} to see available commands"
    echo -e "  2. Run ${YELLOW}'make dev'${NC} to start development mode"
    echo -e "  3. Run ${YELLOW}'make check-hooks'${NC} to test pre-commit hooks"
    echo -e "  4. Run ${YELLOW}'make test-all'${NC} to run the full test suite"
    echo -e "  5. Check out ${YELLOW}'docs/README.md'${NC} for development guides\n"

    echo -e "${BLUE}🔧 Useful commands:${NC}"
    echo -e "  ${YELLOW}make fmt${NC}          - Format code"
    echo -e "  ${YELLOW}make lint${NC}         - Run linting"
    echo -e "  ${YELLOW}make test${NC}         - Run tests"
    echo -e "  ${YELLOW}make pre-commit-check${NC} - Run all pre-commit checks"
    echo -e "  ${YELLOW}make clean${NC}        - Clean build artifacts\n"

    echo -e "${GREEN}Happy coding! 🚀${NC}\n"
}

# Run main function
main "$@"
