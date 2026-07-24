# GizTUI Desktop (Wails) — build helpers.
#
# The desktop app is a nested Go module; these targets wrap the Wails CLI so you
# don't have to remember the flags. Version is read from the repo-root VERSION
# file and injected into internal/version.Version (what the app's Version() shows).

WAILS_VERSION := v2.10.2
VERSION := $(shell tr -d '[:space:]' < ../VERSION)
LDFLAGS := -X 'github.com/ajramos/giztui/internal/version.Version=$(VERSION)'

.PHONY: help deps build build-universal dmg dev clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

deps: ## Install the Wails CLI (pinned)
	go install github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)

build: ## Build the app for the host platform
	wails build -clean -ldflags "$(LDFLAGS)"

build-universal: ## Build a macOS universal (Intel+ARM) app
	wails build -clean -platform darwin/universal -ldflags "$(LDFLAGS)"

dmg: ## Build the universal app and package a .dmg (macOS only)
	./scripts/build-dmg.sh

dev: ## Run the app in dev mode (hot reload)
	wails dev

clean: ## Remove build artifacts
	rm -rf build/bin
