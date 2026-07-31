# 📋 GizTUI Features

Complete feature documentation for GizTUI - the AI-powered Gmail terminal client.

## 👤 Multi-Account Management

### Account Support
- ✅ **Multiple Gmail accounts** - Configure and manage multiple Gmail accounts in one application
- ✅ **Hot account switching** - Switch between accounts without application restart
- ✅ **Account status validation** - Visual status indicators for connection health (✓ Connected, ❌ Error, ● Active)
- ✅ **Number shortcuts (1-9)** - Quick account switching using number keys in account picker
- ✅ **Separate databases per account** - Complete data isolation with account-specific SQLite databases
- ✅ **Backward compatibility** - Existing single-account configurations work unchanged
- ✅ **Account picker interface** - Search and keyboard navigation for account selection
- ✅ **Configurable shortcuts** - Customizable keyboard shortcut for account picker (default: `Ctrl+A`)

### Account Configuration
- ✅ **JSON configuration** - Define multiple accounts with display names and credential paths
- ✅ **Automatic migration** - Legacy single-account configs automatically upgraded
- ✅ **Active account management** - Designate which account is active at startup
- ✅ **Status bar integration** - Current account email displayed in status bar

### Account Commands
- ✅ **Command system integration** - Full `:accounts` command suite with aliases
- ✅ **Direct account switching** - `:accounts switch <account_id>` for command-line switching
- ✅ **Command suggestions** - Auto-complete and contextual suggestions for account operations

## 📬 Core Gmail Functionality

### Email Management
- ✅ **View inbox and labels** - Browse your Gmail inbox with label filtering
- ✅ **Read emails** - Rich email viewing with HTML-to-terminal rendering
- ✅ **Mark as read/unread** - Toggle read status individually or in bulk
- ✅ **Archive and move to trash** - Clean up your inbox efficiently
- ✅ **Manage labels** - Add, remove, and create Gmail labels
- ✅ **Load more messages** - Fetch additional messages when needed
- ✅ **Search and navigation** - VIM-style commands (`:5`, `G`, `gg`)
- ✅ **Enhanced content navigation** - Fast browsing within message content with search, paragraph jumping, and word navigation
- ✅ **Dynamic header visibility** - Toggle email headers to maximize content space
- ✅ **Smart recipient field truncation** - Intelligent truncation of long To/Cc recipient lists with "... and X more recipients" indicators to prevent important header fields (Labels, Date) from being cut off
- ✅ **Email Composition** - Compose, Reply, Reply-All, and Forward with CC/BCC support
- ✅ **Draft Management** - Create, edit, auto-save, and load drafts with side panel picker

### Advanced Email Operations
- ✅ **Undo functionality** - Reverse archive, trash, read/unread, and label operations
- ✅ **Bulk operations** - Select and process multiple messages simultaneously
- ✅ **VIM-style range operations** - Execute operations like `d3d` (delete 3), `a5a` (archive 5)
- ✅ **Enhanced move operations** - Context-aware system folders with regular labels
- ✅ **Message save options** - Save as .txt (rendered) or .eml (raw format)

## 🧵 Message Threading

### Smart Conversation Management
- ✅ **Smart conversation grouping** - Messages grouped by Gmail thread ID with visual hierarchy
- ✅ **Dual view modes** - Toggle between threaded conversations and flat chronological view
- ✅ **Visual indicators** - Thread count badges (📧 5), expand/collapse icons (▶️/▼️)
- ✅ **Thread state persistence** - Remember expanded/collapsed preferences between sessions
- ✅ **Bulk thread operations** - Select entire conversations for bulk actions
- ✅ **AI thread summaries** - Generate conversation overviews with context from all messages
- ✅ **Thread search** - Search within specific conversations for precise information
- ✅ **Auto-expand unread** - Automatically expand threads containing unread messages

### Threading Interface
**Threading Modes:**
- **Thread View** - Group messages by conversation with expand/collapse controls
- **Flat View** - Traditional chronological message list (default)

**Visual Indicators:**
- `📧 5` - Thread with 5 messages
- `▶️` - Collapsed thread (press Enter to expand)
- `▼️` - Expanded thread (press Enter to collapse)  
- `├─` - Reply message indentation
- `└─` - Last reply in thread

**Keyboard Shortcuts:**
| Key | Action |
|-----|--------|  
| `T` | Toggle between thread and flat view |
| `Enter` | Expand/collapse thread (when focused on thread root) |
| `E` | Expand all threads in current view |
| `C` | Collapse all threads to show only root messages |
| `Shift+T` | Generate AI summary of selected thread |

**Commands:**
- `:threads` - Switch to threaded conversation view
- `:flatten` - Switch to flat chronological view
- `:thread-summary` - Generate AI summary of conversation
- `:expand-all` - Expand all threads
- `:collapse-all` - Collapse all threads

## 🧠 AI Features with LLM

### Core AI Capabilities
- ✅ **Email summarization** - Generate concise email summaries with streaming support
- ✅ **AI summaries local cache** - SQLite-based caching for instant retrieval
- ✅ **Streaming summaries** - Incremental token rendering for Ollama
- ✅ **Streaming cancellation** - Press Esc to instantly cancel operations
- ✅ **Smart label suggestions** - AI-powered label recommendations
- ✅ **Configurable prompts** - Fully customizable AI prompt templates
- ✅ **Multiple LLM providers** - Support for Ollama (local) and Amazon Bedrock (cloud)

### Prompt Library System
- ✅ **Custom prompt templates** - Predefined and user-created prompts for various use cases
- ✅ **Variable substitution** - Auto-complete `{{from}}`, `{{subject}}`, `{{body}}`, `{{date}}`
- ✅ **Streaming LLM responses** - Real-time token streaming for immediate feedback
- ✅ **Interruptible streaming** - Cancel any prompt operation instantly with Esc
- ✅ **Smart caching** - Cache prompt results to avoid re-processing
- ✅ **Split-view interface** - Prompt picker appears as side panel
- ✅ **Category organization** - Organize prompts by purpose (Summary, Analysis, Action Items)
- ✅ **Usage tracking** - Monitor prompt usage patterns and effectiveness with `:prompt stats`
- ✅ **Statistics display** - Full-screen prompt usage analytics with top prompts and favorites
- ✅ **CRUD management** - Create, update, export, and delete templates via `:prompt` commands
- ✅ **YAML front matter** - Standard Markdown format with metadata headers
- ✅ **Management interface** - Browse all prompts including bulk analysis templates

## 🔥 Bulk Operations

### Multi-Message Processing
- ✅ **Multi-email analysis** - Apply prompts to multiple emails simultaneously
- ✅ **Bulk labeling** - Apply labels to multiple selected messages at once
- ✅ **Enhanced move panel** - Context-aware system folders with regular labels
- ✅ **Bulk moving** - Move multiple messages to system folders or labels
- ✅ **Search-enabled operations** - Filter labels during bulk operations
- ✅ **Consolidated insights** - Get unified analysis across multiple messages
- ✅ **Efficient processing** - Async processing with progress indicators
- ✅ **Responsive controls** - Cancel bulk operations instantly with Esc
- ✅ **Robust error handling** - Proper status updates and deadlock prevention

### Specialized Bulk Analysis
- ✅ **Cloud product tracking** - Specialized prompts for AWS/Azure/GCP updates
- ✅ **Project monitoring** - Consolidate project status from multiple emails
- ✅ **Trend analysis** - Identify patterns across multiple sources

## 🔌 Integration Features

### Slack Integration
- ✅ **Email forwarding to Slack** - Send emails to configured Slack channels
- ✅ **Bulk forwarding with comments** - Forward multiple emails with shared context
- ✅ **Multiple format styles** - Summary (AI-generated), Markdown (cleaned reader-style, between summary and full), Compact, Full (TUI-processed), Raw
- ✅ **AI-powered summaries** - Use custom AI prompts for intelligent summarization
- ✅ **Custom user messages** - Add optional context when forwarding emails
- ✅ **Multi-channel support** - Configure multiple Slack channels with webhooks
- ✅ **Variable substitution** - Dynamic prompts with email headers and content
- ✅ **TUI content fidelity** - "Full" format shows exactly what you see in the message widget
- ✅ **Progress tracking** - Real-time progress updates for bulk operations

### Obsidian Integration  
- ✅ **Email ingestion** - Send emails directly to Obsidian as Markdown notes
- ✅ **Bulk ingestion** - Process multiple selected emails with shared comments
- ✅ **Repopack mode** - Combine multiple emails into a single consolidated Markdown file
- ✅ **Configurable templates** - Customize both individual email and repopack output formats
- ✅ **Template customization** - Edit `~/.config/giztui/templates/obsidian/repopack.md` to personalize repopack files
- ✅ **Enhanced variables** - Use `{{title}}`, `{{batch_info}}`, `{{footer}}` and more in custom templates
- ✅ **Second brain system** - Organize emails in `00-Inbox` folder for processing
- ✅ **Variable substitution** - Auto-complete `{{subject}}`, `{{body}}`, `{{from}}`, etc.
- ✅ **Personal comments** - Add custom notes before ingestion (single & bulk)
- ✅ **Message compilation** - Structured message headers with metadata preservation
- ✅ **Duplicate prevention** - SQLite-based history tracking prevents re-ingestion
- ✅ **Attachment support** - Include email attachments in Obsidian notes
- ✅ **Keyboard shortcut** - `Shift+O` for quick ingestion
- ✅ **Panel interface** - Clean side panel (not modal) with repopack mode toggle
- ✅ **Command parity** - `:obsidian repack` command for bulk repopack operations

### Calendar Integration
- ✅ **Smart invitation detection** - Automatically detects calendar invitations in emails
- ✅ **Enhanced RSVP widget** - Press `Shift+V` to respond to meeting invitations
- ✅ **Meeting details display** - Shows title, organizer, date/time with proper formatting
- ✅ **iCalendar parsing** - Handles complex timezone-aware calendar data
- ✅ **Direct calendar integration** - Updates your Google Calendar with RSVP responses
- ✅ **Multiple response options** - Accept, Tentative, or Decline with one key press
- ✅ **Clean visual design** - Color-coded information with proper spacing

## 🔗 Productivity Tools

### Link Management
- ✅ **Smart link detection** - Automatically extract links from HTML and plain text emails
- ✅ **Quick access** - Press `L` to open link picker or use `:links` command
- ✅ **Cross-platform opening** - Native browser opening on macOS, Linux, Windows
- ✅ **Advanced search** - Filter links by text, domain (`domain:github.com`), or type
- ✅ **Visual categorization** - Icons for different link types (🌐 external, 📧 email, 📁 files)
- ✅ **Keyboard navigation** - Arrow keys to browse, Enter to open, 1-9 for quick access
- ✅ **Multiple protocols** - Support for HTTP/HTTPS, FTP/FTPS, and mailto links
- ✅ **Real clipboard copy** - Copy URLs with `Ctrl+Y` (cross-platform clipboard support)
- ✅ **Status bar preview** - See full URLs in status bar while navigating
- ✅ **Instant feedback** - Live URL display and success messages

### Attachment Management
- ✅ **Smart attachment detection** - Automatically extract attachments from email MIME structure
- ✅ **Quick access** - Press `A` to open attachment picker or use `:attachments` command
- ✅ **Cross-platform downloads** - Native file handling on macOS, Linux, Windows
- ✅ **Advanced search** - Filter attachments by name, type (`type:image`), or size
- ✅ **Visual categorization** - Icons for different file types (📄 docs, 🖼️ images, 📊 spreadsheets)
- ✅ **Keyboard navigation** - Arrow keys to browse, Enter to download, 1-9 for quick access
- ✅ **Multiple file types** - Support for documents, images, archives, audio, video, and more
- ✅ **Save controls** - Download to default location or use `Ctrl+S` to save as
- ✅ **Smart file naming** - Automatic filename conflict resolution with incremental numbering
- ✅ **Auto-open option** - Configurable automatic opening of downloaded files
- ✅ **Size-aware display** - Human-readable file sizes (KB, MB, GB) with MIME type info

### Gmail Web Integration
- ✅ **Open in Gmail** - Press `O` to open current message in Gmail web interface
- ✅ **Complex message handling** - Perfect for messages better viewed in full Gmail UI
- ✅ **Cross-platform browser opening** - Native browser launching on macOS, Linux, Windows
- ✅ **Command parity** - Use `:gmail`, `:web`, `:open-web`, or `:o` commands
- ✅ **Smart URL generation** - Direct links to specific Gmail messages
- ✅ **Configurable shortcut** - Customize the keyboard shortcut via `open_gmail` in config
- ✅ **Seamless workflow** - Select message in TUI, open in web for detailed work

## 🎨 User Interface & Experience

### Adaptive Layout System
- ✅ **Responsive design** - Automatically adapts to terminal size changes
- ✅ **Multiple layout modes** - Wide (≥120x30), Medium (≥80x25), Narrow (≥60x20), Mobile (<60x20)
- ✅ **Real-time resizing** - Layout updates as you resize terminal
- ✅ **Fullscreen mode** - Press 'f' for fullscreen text view
- ✅ **Focus switching** - Press 't' to toggle between list and text focus

### Theme System
- ✅ **Runtime theme switching** - Change themes instantly without restart
- ✅ **Multiple built-in themes** - Slate Blue (default), Dracula, Gmail Dark/Light, Custom Example
- ✅ **Custom theme support** - User themes in `~/.config/giztui/themes/`
- ✅ **Theme preview** - See colors before applying themes
- ✅ **Hierarchical color system** - Foundation → Semantic → Interaction → Component overrides

### User Experience Features
- ✅ **Fully customizable keyboard shortcuts** - Configure any shortcut through config.json
- ✅ **Multiple shortcut styles** - Support for Vim, Emacs, and custom shortcut schemes
- ✅ **25+ configurable actions** - Customize core email operations and additional features
- 🎨 **Inspired by `k9s`, `neomutt`, `alpine`** - Professional terminal application design
- ⌨️ **100% keyboard navigation** - Complete mouse-free operation
- ⚡ **Efficient and fast interface** - Optimized for productivity
- 🔧 **Highly configurable** - Extensive customization options
- 🔒 **Private** - No data sent to external cloud services

### Enhanced Navigation
- ✅ **VIM-style commands** - `gg` (first message), `G` (last message), `:5` (jump to message 5)
- ✅ **Content search** - `/searchterm` with `n`/`N` navigation and highlighting
- ✅ **Fast content navigation** - Paragraph jumping (`Ctrl+K/J`), word navigation (`Ctrl+H/L`)
- ✅ **Context-aware shortcuts** - Different behaviors when viewing message vs message list
- ✅ **Local filtering** - In-memory filter with `/` including label filters (`label:Personal`)
- ✅ **Advanced search form** - Multiple fields with quick options panel
- ✅ **Size-based search** - Filter by email size (`>1MB`, `<500KB`)
- ✅ **Date range filtering** - Flexible date searches with `after:`/`before:` operators

### Welcome & Status
- ✅ **Welcome screen** - Structured startup with account info and quick actions
- ✅ **Rich status information** - Context-aware status bar with operation feedback
- ✅ **Loading indicators** - Progress feedback for long operations
- ✅ **Error handling** - User-friendly error messages and recovery options

### Command System
- ✅ **Command parity** - Every keyboard shortcut has equivalent command
- ✅ **Auto-completion** - Tab completion for all commands with live suggestions
- ✅ **Context awareness** - Commands automatically detect bulk mode and act appropriately
- ✅ **Command history** - Navigation through previous commands
- ✅ **k9s-style interface** - Professional command bar with bordered panel

## 🔧 Technical Features

### Database & Caching
- ✅ **SQLite integration** - Embedded database for AI summaries, prompts, and history
- ✅ **Per-account separation** - Isolated databases by email account
- ✅ **Smart caching** - Cache AI results, prompt responses, and Obsidian history
- ✅ **Performance optimization** - Proper indexing and query optimization
- ✅ **Background preloading** - Intelligent message preloading for instant navigation
- ✅ **LRU cache management** - Efficient memory usage with Least Recently Used eviction
- ✅ **Worker pool architecture** - Concurrent background processing with resource limits

### Architecture
- ✅ **Service-oriented architecture** - Clean separation of UI and business logic
- ✅ **Thread-safe operations** - Mutex-protected accessor methods for app state
- ✅ **Centralized error handling** - Consistent user feedback with ErrorHandler
- ✅ **Dependency injection** - Services automatically initialized and injected

### Configuration System
- ✅ **Unified configuration** - Single `config.json` with hierarchical organization
- ✅ **Template file support** - External Markdown files for AI/Slack/Obsidian templates
- ✅ **Environment variable support** - Override paths via environment variables
- ✅ **Smart path resolution** - Relative paths resolved relative to config directory

### Performance Features
- ✅ **Next page preloading** - Preloads next page at 70% scroll threshold for instant "Load More"
- ✅ **Adjacent message preloading** - Preloads 3 messages around selection for smooth navigation
- ✅ **Configurable thresholds** - Customize scroll triggers, cache sizes, and worker limits
- ✅ **Resource management** - API quota reserves and memory limits prevent overuse
- ✅ **Runtime control** - `:preload` commands for live configuration changes
- ✅ **Smart eviction** - LRU-based cache eviction maintains optimal memory usage

## 🚀 Development & Quality

### Testing Framework
- ✅ **Comprehensive test suite** - Unit tests, integration tests, TUI component tests
- ✅ **Performance testing** - Load testing for bulk operations
- ✅ **Mock generation** - Automated mock generation with mockery
- ✅ **CI/CD pipeline** - GitHub Actions with comprehensive quality checks

### Cross-Platform Support
- ✅ **Multi-platform builds** - Linux, macOS (Intel/ARM), Windows (AMD64/ARM64)
- ✅ **Cross-platform file operations** - Native file handling across operating systems
- ✅ **Cross-platform browser integration** - Native browser opening on all platforms
- ✅ **Cross-platform clipboard** - Proper clipboard integration across platforms

---

## 📖 Learn More

- [Getting Started Guide](GETTING_STARTED.md) - Quick setup and first steps
- [Configuration Guide](CONFIGURATION.md) - Complete configuration reference  
- [Keyboard Shortcuts](KEYBOARD_SHORTCUTS.md) - Complete shortcut reference
- [Architecture Guide](ARCHITECTURE.md) - Development patterns and conventions
- [Theming Guide](THEMING.md) - Theme system and customization