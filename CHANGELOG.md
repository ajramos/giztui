# Changelog

All notable changes to GizTUI (formerly Gmail TUI) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2025-09-09

### ✨ New Features

- **Multi-Account Support**: Complete implementation of database-per-account architecture enabling seamless switching between multiple Gmail accounts
- **Account Picker**: Interactive account selection with number shortcuts (1-9) for quick switching, similar to Links picker UX
- **Hot Account Switching**: Real-time account switching with proper service re-initialization and database context switching
- **Account Commands**: New `:accounts` command with full keyboard shortcut parity for account management
- **Enhanced OAuth Flow**: Account-specific authorization messages with improved user experience

### 🛠️ Technical Improvements

- **Unified Logger Architecture**: Comprehensive account selection logging with centralized logging infrastructure
- **Graceful Credential Fallback**: Multi-level credential fallback system for robust authentication handling
- **Service Initialization**: Improved database-dependent service initialization timing and coordination
- **Cache Management**: Account-aware cache service with proper invalidation during account switching
- **UI Consistency**: Enhanced multi-account UI patterns with proper error handling and validation

### 🐛 Bug Fixes

- **Gmail Client Updates**: Resolved Gmail client not updating properly during account switching
- **Database Connections**: Fixed Obsidian export database connection issues in multi-account scenarios  
- **Service Re-initialization**: Fixed cache and database services not being reinitialized during account switching
- **UI State Management**: Resolved multi-account UI inconsistencies and enhanced account picker display
- **Control Key Shortcuts**: Fixed control key shortcut customization not working properly

---

## [1.1.1] - 2025-09-07

### 🐛 Bug Fixes

- **Version Display Fix**: Fixed version display for `go install` builds showing correct v1.1.1 instead of outdated v1.0.2
- **Help Screen Layout**: Fixed command equivalent section column alignment for better readability
- **Build Method Consistency**: Ensured version fallback values stay synchronized with VERSION file

---

## [1.1.0] - 2025-09-06

### ✨ New Features

- **Obsidian Repopack Integration**: Complete email export functionality with comprehensive context
- **New Keyboard Shortcuts**: `o repack`, `obsidian repack`, `obs repack`
- **New Commands**: `:obsidian repack`, `:obs repack` with full command parity
- **Bulk Repack Support**: Apply repack operations to multiple selected emails
- **Smart Mode Detection**: Handles both count-based and repack operations seamlessly

### 🛠️ Technical Improvements

- **Comprehensive Test Coverage**: 95%+ coverage for all repack functionality
- **Enhanced Linting**: Updated golangci-lint configuration for better code quality
- **Robust Error Handling**: Improved validation and user feedback

---

## [1.0.2] - 2025-09-04

### ✨ New Features

#### Enhanced Version Detection System
- **Smart Version Detection**: No more "unknown" Git commit for `go install` builds
- **Automatic VCS Integration**: Leverage Go 1.18+ runtime/debug.BuildInfo for automatic Git information
- **Build Method Indication**: Clear differentiation between `make`, `go-install`, and `unknown` builds
- **VCS Status Detection**: Show modification status for development builds with uncommitted changes
- **Improved Version Display**: Better formatted version strings with meaningful build information

#### Comprehensive Pre-commit System
- **Enhanced Pre-commit Hooks**: Comprehensive hooks matching CI pipeline requirements exactly
- **Format & Lint Checking**: Automatic code formatting and linting before commits
- **Essential Test Runner**: Quick smoke tests to catch breaking changes early
- **Developer Setup Script**: One-script onboarding for new contributors (`scripts/setup-dev.sh`)

#### Streamlined CI/CD Pipeline
- **Consolidated Workflow**: Single comprehensive CI/CD pipeline replacing separate workflows
- **Multi-platform Testing**: Cross-platform validation (Ubuntu + macOS)
- **Enhanced Security**: Trivy vulnerability scanning and dependency review
- **Better Reporting**: Improved PR comments with detailed CI/CD results

### 🛠️ Developer Experience Improvements

#### New Make Commands
- `make setup-hooks` - Install and configure pre-commit hooks
- `make check-hooks` - Run pre-commit hooks on all files
- `make pre-commit-check` - Run same checks as CI locally
- `make remove-hooks` - Remove pre-commit hooks

#### Enhanced Documentation
- **Development Setup Guide**: Complete contributor onboarding documentation
- **Installation Guide Updates**: Clear explanations of version differences between build methods
- **Build Method Documentation**: Comprehensive explanation of `make build` vs `go install`

### 🔧 Technical Improvements

#### Configuration Enhancements
- **Updated golangci-lint Config**: Modern configuration with version 2 support
- **Improved Linting Rules**: Comprehensive linter setup with proper exclusions
- **Pre-commit Configuration**: Hooks that mirror CI pipeline exactly

#### Build System
- **Version Injection Compatibility**: Maintains full backward compatibility with custom build metadata
- **VCS Detection**: Automatic Git commit, time, and modification status detection
- **Release Process**: Streamlined and validated release workflow

### 📚 Documentation Updates

- Enhanced installation instructions with version information explanations
- New troubleshooting section for version-related issues
- Complete developer setup and contribution guidelines
- Updated architecture documentation for new version detection system

### 🎯 Benefits for Users

- **Better Debugging**: Meaningful version information regardless of installation method
- **Improved Traceability**: Clear build method and Git commit information
- **Enhanced Developer Experience**: Easier setup and contribution process
- **Quality Assurance**: Automated checks prevent common issues from reaching CI/CD

---

## [1.0.1] - 2025-09-04

### 🚀 Performance Improvements

#### Background Preloading System
- **Phase 2.4 Background Preloading**: Implement instant navigation with intelligent message preloading
- **70% Preloading Functionality**: Restore preloading with proper pagination token preservation
- **Preload Control**: Add comprehensive preload off command to disable all preloading features consistently
- **Focus Highlighting**: Implement proper focus highlighting for preload status screen

#### Gmail API Optimization
- **Metadata Optimization**: Achieve 70-80% bandwidth reduction through selective field requests
- **Load More Improvements**: Resolve focus and pagination issues in load more functionality

### ✨ New Features

#### UI/UX Enhancements
- **Smart Recipient Truncation**: Handle long To/Cc fields with intelligent truncation and configuration support
- **Full-Screen Prompt Statistics**: Transform stats display to comprehensive prompt statistics view
- **Keyboard Shortcut Validation**: Add optional keyboard shortcut validation with comprehensive coverage

#### Command System
- **Prompt Stats Command**: New `:prompt stats` command for detailed prompt usage analytics

### 🐛 Bug Fixes

#### Navigation & Controls
- **Navigation Issues**: Correct gg navigation and number command navigation problems
- **Focus Management**: Resolve focus issues in various UI components

#### Theme & Display
- **Emoji Rendering**: Fix status bar emoji rendering issues for better terminal compatibility
- **Save Search Query Dialog**: Improve theme consistency across all dialog elements
- **Advanced Search**: Replace problematic emoji in date field validation

#### Development & Build
- **Clean Build Process**: Remove development artifacts and ensure clean production builds
- **Configuration Integration**: Complete MaxRecipientLines feature with full config integration

### 🔧 Technical Improvements

- **Documentation Updates**: Comprehensive updates for new features and configuration options
- **Build System**: Enhanced cross-platform build process and artifact management
- **Code Quality**: Various code cleanup and optimization improvements

### 📋 Configuration

- **MaxRecipientLines**: New configuration option to control recipient field truncation behavior
- **Preloading Controls**: Enhanced configuration options for background preloading system

## [1.0.0] - 2025-09-02

### 🎉 Initial Stable Release

This marks the first stable release of GizTUI (formerly Gmail TUI), featuring a complete terminal-based Gmail client with AI integration, advanced UI/UX, and powerful productivity features.

### ✨ Core Gmail Features

#### Email Management
- **Full email operations**: Read, compose, reply, reply-all, forward, archive, trash, and restore
- **Advanced search**: Gmail query syntax support with contextual shortcuts (from:current, to:current, subject:current)
- **Enhanced move operations**: Context-aware system folders (Inbox, Trash, Archive, Spam) with regular labels
- **Message threading**: Smart conversation grouping with visual hierarchy and expand/collapse controls
- **Dual view modes**: Toggle between threaded conversations and flat chronological view
- **Bulk operations**: Multi-select messages for batch actions (archive, trash, move, label)
- **VIM-style navigation**: Range operations like `d3d` (delete 3), `a5a` (archive 5), `t2t` (toggle read 2)
- **Undo functionality**: Reverse archive, trash, read/unread, and label operations

#### Message Composition
- **Complete composition UI**: Full-screen modal with proper theming and focus management
- **Advanced recipient handling**: CC/BCC support with proper recipient extraction and exclusion
- **Draft management**: Create, edit, save, and delete drafts with picker UI
- **Auto-save drafts**: Automatic draft saving during composition
- **Message threading**: Proper threading headers and Gmail compatibility
- **Real-time validation**: Email format validation and visual error indicators

#### Labels and Organization
- **Full label management**: Create, delete, apply, and remove labels
- **Contextual labels panel**: Side panel with quick toggle and live refresh
- **Browse all labels**: Full picker with incremental search and ESC navigation
- **Visual label colors**: Each label displayed with unique colors in message lists
- **Enhanced bulk labeling**: Apply labels to multiple selected messages
- **Smart label suggestions**: AI-powered label recommendations

### 🧠 AI and LLM Integration

#### Core AI Features
- **Email summarization**: Generate concise email summaries with streaming support
- **Smart label suggestions**: AI-powered label recommendations based on email content
- **Streaming LLM responses**: Real-time token streaming for immediate feedback
- **Intelligent caching**: SQLite-based caching system for AI results to avoid duplicate processing
- **Multiple LLM providers**: Support for both Ollama (local) and Amazon Bedrock (cloud)

#### Prompt Library System
- **Custom prompt templates**: Predefined and user-created prompts for various use cases
- **Variable substitution**: Auto-complete `{{from}}`, `{{subject}}`, `{{body}}`, `{{date}}`, `{{messages}}`
- **Category organization**: Organize prompts by purpose (Summary, Analysis, Action Items, etc.)
- **Usage tracking**: Monitor prompt usage patterns and effectiveness
- **Split-view interface**: Prompt picker appears as side panel (not full-screen modal)
- **Bulk prompt processing**: Apply prompts to multiple emails simultaneously for consolidated analysis

#### Advanced AI Operations
- **Thread summaries**: Generate conversation overviews with context from all messages
- **Bulk email analysis**: Consolidated insights across multiple selected messages
- **Smart content processing**: Optional LLM touch-up for better email formatting
- **Interruption support**: Cancel any streaming operation instantly with ESC key

### 🔌 Integration Features

#### Slack Integration
- **Multi-channel support**: Configure multiple Slack channels with individual webhooks
- **Bulk forwarding**: Forward multiple emails simultaneously with shared comments
- **Multiple format styles**: Summary (AI-generated), Compact, Full (TUI-processed), Raw
- **Smart variable substitution**: AI prompts support email headers and content variables
- **Progress tracking**: Real-time progress updates for bulk operations
- **TUI content fidelity**: "Full" format shows exactly what you see in the message widget

#### Obsidian Integration
- **Email ingestion**: Send emails directly to Obsidian as Markdown notes
- **Bulk ingestion**: Process multiple selected emails with shared comments
- **Template system**: Single, customizable Markdown template with variable substitution
- **Duplicate prevention**: SQLite-based history tracking prevents re-ingestion
- **Attachment support**: Include email attachments in Obsidian notes
- **Second brain organization**: Organize emails in `00-Inbox` folder for processing

#### Calendar Integration
- **Smart invitation detection**: Automatically detect calendar invitations in emails
- **Enhanced RSVP handling**: Accept, Tentative, or Decline with Google Calendar API integration
- **Meeting details display**: Shows title, organizer, date/time with proper formatting
- **iCalendar parsing**: Handles complex timezone-aware calendar data

### 🎨 Advanced UI/UX

#### Theme System
- **Runtime theme switching**: Change themes instantly without restart (`/theme set <name>`)
- **Multiple built-in themes**: Slate Blue (default), Dracula, Gmail Dark/Light, Custom Example
- **Custom theme support**: User themes in `~/.config/giztui/themes/`
- **Theme preview**: See colors before applying themes
- **Hierarchical color system**: Foundation → Semantic → Interaction → Component overrides

#### Adaptive Layout System
- **Responsive design**: Automatically adapts to terminal size changes
- **Multiple layout types**: Wide (≥120x30), Medium (≥80x25), Narrow (≥60x20), Mobile (<60x20)
- **Smart focus management**: Proper focus cycling with Tab/Shift+Tab
- **Fullscreen mode**: Press 'f' for fullscreen text view
- **Real-time resizing**: Layout updates as you resize terminal

#### Enhanced Navigation
- **VIM-style commands**: `gg` (first message), `G` (last message), `:5` (jump to message 5)
- **Content search**: `/searchterm` with `n`/`N` navigation and highlighting
- **Fast content navigation**: Paragraph jumping (`Ctrl+K/J`), word navigation (`Ctrl+H/L`)
- **Context-aware shortcuts**: Different behaviors when viewing message vs message list
- **Enhanced content navigation**: Fast browsing within message content

#### Advanced Search and Filtering
- **Local filtering**: In-memory filter with `/` including label filters (`label:Personal`)
- **Advanced search form**: Multiple fields with quick options panel
- **Size-based search**: Filter by email size (`>1MB`, `<500KB`)
- **Date range filtering**: Flexible date searches with `after:`/`before:` operators
- **Search highlighting**: Visual highlighting of search terms in results

### 🔧 Productivity Features

#### Link and Attachment Management
- **Smart link extraction**: Automatically extract links from HTML and plain text emails
- **Link picker**: Press `L` for quick link access with search and categorization
- **Cross-platform opening**: Native browser opening on macOS, Linux, Windows
- **Attachment picker**: Press `A` for attachment management with download and preview
- **Smart file handling**: Automatic filename conflict resolution and cross-platform downloads

#### Command System
- **Command parity**: Every keyboard shortcut has equivalent command (`:archive`, `:trash`, etc.)
- **Auto-completion**: Tab completion for all commands with live suggestions
- **Context awareness**: Commands automatically detect bulk mode and act appropriately
- **Command history**: Navigation through previous commands
- **k9s-style interface**: Professional command bar with bordered panel

#### Bulk Operations
- **Advanced selection**: `v`/`b`/`space` for bulk mode, `*` for select all
- **Range operations**: VIM-style `d3d`, `a5a`, `t2t` for efficient batch actions
- **Bulk AI processing**: Apply prompts to multiple emails for consolidated analysis
- **Progress indicators**: Real-time feedback for long-running bulk operations

### 🏗️ Architecture and Development

#### Service-Oriented Architecture
- **Clean separation**: UI components handle only presentation, services handle business logic
- **Service layer**: EmailService, AIService, LabelService, CacheService, etc.
- **Centralized error handling**: Consistent user feedback with ErrorHandler
- **Thread-safe operations**: Mutex-protected accessor methods for app state
- **Dependency injection**: Services automatically initialized and injected

#### Database and Caching
- **SQLite integration**: Embedded database for AI summaries, prompts, and history
- **Per-account separation**: Isolated databases by email account
- **Smart caching**: Cache AI results, prompt responses, and Obsidian history
- **Performance optimization**: Proper indexing and query optimization

#### Configuration System
- **Unified configuration**: Single `config.json` with hierarchical organization
- **Template file support**: External Markdown files for AI/Slack/Obsidian templates
- **Environment variable support**: Override paths via environment variables
- **Smart path resolution**: Relative paths resolved relative to config directory

### 🧪 Testing and Quality

#### Comprehensive Testing Framework
- **Unit tests**: Service layer testing with mocks
- **Integration tests**: Full workflow testing
- **TUI component tests**: Terminal UI component validation
- **Performance tests**: Load testing for bulk operations
- **Mock generation**: Automated mock generation with mockery

#### CI/CD Pipeline
- **Automated testing**: GitHub Actions with comprehensive test suite
- **Multi-platform builds**: Linux, macOS (Intel/ARM), Windows
- **Code quality**: golangci-lint, go vet, format checking
- **Security scanning**: Vulnerability scanning with govulncheck

### 📋 Configuration Migration
- **Directory migration**: Automatic migration from `~/.config/gmail-tui/` to `~/.config/giztui/`
- **Backward compatibility**: Seamless upgrade path for existing users
- **Environment variables**: Updated environment variable names for consistency

### 🎯 User Experience Improvements
- **Welcome screen**: Structured startup with account info and quick actions
- **Status bar**: Rich status information with operation feedback
- **Error handling**: User-friendly error messages and recovery options
- **Loading indicators**: Progress feedback for long operations
- **Keyboard shortcuts**: Fully customizable keyboard shortcuts via configuration

### 🔨 Developer Experience
- **Clean codebase**: Well-organized project structure with clear separation of concerns
- **Comprehensive documentation**: Architecture docs, theming guide, development guide
- **Build system**: Makefile with development, testing, and release targets
- **Version management**: Proper semantic versioning with build-time injection

---

## Development Notes

### Migration from Gmail TUI
This release represents the stable v1.0.0 of the project formerly known as "Gmail TUI". All references to the old naming have been updated to "GizTUI" for consistency.

### Supported Platforms
- Linux (amd64)
- macOS (Intel and Apple Silicon)
- Windows (amd64)

### Requirements
- Go 1.23.0+
- Gmail API credentials
- Terminal with 256-color support
- Optional: Ollama for local AI features
- Optional: AWS credentials for Bedrock AI

### Breaking Changes
- Configuration directory changed from `~/.config/gmail-tui/` to `~/.config/giztui/`
- Binary name changed from `gmail-tui` to `giztui`
- Module path changed from `github.com/ajramos/gmail-tui` to `github.com/ajramos/giztui`

### Credits
GizTUI is inspired by excellent terminal applications like `k9s`, `neomutt`, and `alpine`, bringing modern AI capabilities and productivity features to terminal-based email management.

[1.0.0]: https://github.com/ajramos/giztui/releases/tag/v1.0.0