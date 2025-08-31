# 📨 Gmail TUI - Gmail Client with Local AI

A **TUI (Terminal User Interface)** Gmail client developed in **Go** that uses the **Gmail API** via OAuth2 and features **local AI integration** through Ollama.

## ✨ Features

### 📬 Core Gmail Functionality
- ✅ View inbox and labels
- ✅ Read emails
- ✅ Mark as read/unread
- ✅ Archive and move to trash
- ✅ Manage labels (add, remove, create)
- ✅ Load more messages (when list is focused)
- ✅ Search and navigation support with VIM-style commands (`:5`, `G`, `gg`)
- ✅ **Enhanced content navigation** - Fast browsing within message content with search, paragraph jumping, and word navigation
- ✅ **Dynamic header visibility** 🆕 - Toggle email headers to maximize content space for complex messages
- ✅ **Email Composition** - Compose, Reply, Reply-All, and Forward messages with CC/BCC support
- ✅ **Draft Management** - Create, edit, auto-save, and load drafts with side panel picker
- 🚧 WIP: Attachments

### 🧵 **Message Threading** 🆕
- ✅ **Smart conversation grouping** - Messages grouped by Gmail thread ID with visual hierarchy
- ✅ **Dual view modes** - Toggle between threaded conversations and flat chronological view
- ✅ **Visual indicators** - Thread count badges (📧 5), expand/collapse icons (▶️/▼️), indented replies
- ✅ **Thread state persistence** - Remember expanded/collapsed preferences between sessions
- ✅ **Bulk thread operations** - Select entire conversations for bulk actions
- ✅ **AI thread summaries** - Generate conversation overviews with context from all messages
- ✅ **Thread search** - Search within specific conversations for precise information
- ✅ **Auto-expand unread** - Automatically expand threads containing unread messages
- ✅ **Keyboard shortcuts** - `T` to toggle modes, `E`/`C` for expand/collapse all, `Shift+T` for summaries
- ✅ **Command parity** - `:threads`, `:flatten`, `:thread-summary`, `:expand-all`, `:collapse-all`
- ✅ **Smart focus management** - Enter expands/collapses in thread mode, shows message in flat mode
- ✅ **Configurable behavior** - Control threading preferences, indentation, and summary features

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

**Configuration:**
```json
{
  "threading": {
    "enabled": true,
    "default_view": "flat",
    "auto_expand_unread": true,
    "show_thread_count": true,  
    "indent_replies": true,
    "thread_summary_enabled": true
  }
}
```

### 🧠 AI Features with LLM (Ollama & Bedrock)
- ✅ **Summarize emails** - Generate concise email summaries
- ✅ **AI summaries local cache (SQLite)** - Reuse previously generated summaries across sessions
- ✅ **Streaming summaries (Ollama)** - Incremental tokens render live in the summary pane
- ✅ **Streaming cancellation** - Press Esc to instantly cancel any streaming operation
- ✅ **Recommend labels** - Suggest appropriate labels for emails
- ✅ **Configurable prompts** - All prompts are customizable
- 🧪 **Generate replies** - Experimental (placeholder implementation)

### 💬 **Slack Integration** 🆕
- ✅ **Email forwarding to Slack** - Send emails to configured Slack channels (single & bulk)
- ✅ **Bulk forwarding with comments** - Forward multiple selected emails with shared comments
- ✅ **Multiple format styles** - Choose between summary, compact, full, or raw formats
- ✅ **AI-powered summaries** - Use custom AI prompts for intelligent email summarization
- ✅ **Custom user messages** - Add optional context when forwarding emails
- ✅ **Multi-channel support** - Configure multiple Slack channels with webhooks
- ✅ **Variable substitution** - Dynamic prompts with email headers and content
- ✅ **TUI content fidelity** - "Full" format uses exact same processed content as displayed
- ✅ **Progress tracking** - Real-time progress updates for bulk operations

### 🚀 **Prompt Library System** 🆕
- ✅ **Custom prompt templates** - Predefined prompts for different use cases
- ✅ **Variable substitution** - Auto-complete `{{from}}`, `{{subject}}`, `{{body}}`, `{{date}}`
- ✅ **Streaming LLM responses** - Real-time token streaming for prompt results
- ✅ **Interruptible streaming** - Cancel any prompt operation instantly with Esc
- ✅ **Smart caching** - Cache prompt results to avoid re-processing
- ✅ **Split-view interface** - Prompt picker appears like labels (not full-screen modal)
- ✅ **Category organization** - Organize prompts by purpose (Summary, Analysis, Action Items, etc.)
- ✅ **Usage tracking** - Monitor which prompts are used most frequently
- ✅ **CRUD management** - Create, update, export, and delete prompt templates via commands
- ✅ **YAML front matter** - Standard Markdown format with metadata headers
- ✅ **Management interface** - Browse all prompts including bulk analysis templates
- ✅ **Prompt details view** - Full template preview with metadata and usage stats
- ✅ **Command shortcuts** - `:prompt`, `:pr`, `:p` for quick management access
- ✅ **File operations** - Export prompts to Markdown files, import from YAML
- ✅ **Dynamic headers** - Message headers adapt to content size during prompt viewing

### 🔥 **Bulk Operations** 🆕
- ✅ **Multi-email analysis** - Apply prompts to multiple emails simultaneously
- ✅ **Bulk labeling** - Apply labels to multiple selected messages at once
- ✅ **Bulk moving** - Move multiple messages with label+archive in one operation
- ✅ **Search-enabled operations** - Filter labels during bulk operations for quick selection
- ✅ **Consolidated insights** - Get unified analysis across multiple messages
- ✅ **Cloud product tracking** - Specialized prompts for AWS/Azure/GCP updates
- ✅ **Project monitoring** - Consolidate project status from multiple emails
- ✅ **Trend analysis** - Identify patterns across multiple sources
- ✅ **Efficient processing** - Async processing with progress indicators
- ✅ **Responsive controls** - Cancel bulk operations instantly with Esc
- ✅ **Robust error handling** - Proper status updates and deadlock prevention

### 📝 **Obsidian Integration** 🆕
- ✅ **Email ingestion** - Send emails directly to Obsidian as Markdown notes
- ✅ **Bulk ingestion** - Process multiple selected emails with shared comments
- ✅ **Second brain system** - Organize emails in `00-Inbox` folder
- ✅ **Configurable template** - Single, customizable Markdown template
- ✅ **Variable substitution** - Auto-complete `{{subject}}`, `{{body}}`, `{{from}}`, etc.
- ✅ **Personal comments** - Add custom notes before ingestion (single & bulk)
- ✅ **Duplicate prevention** - SQLite-based history tracking
- ✅ **Attachment support** - Include email attachments in notes
- ✅ **Keyboard shortcut** - `Shift+O` for quick ingestion
- ✅ **Panel interface** - Clean side panel (not modal) for template preview

### 🔗 **Link Picker** 🆕
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

### 📎 **Attachment Picker** 🆕
- ✅ **Smart attachment detection** - Automatically extract attachments from email MIME structure
- ✅ **Quick access** - Press `A` to open attachment picker or use `:attachments` command
- ✅ **Cross-platform downloads** - Native file handling on macOS, Linux, Windows
- ✅ **Advanced search** - Filter attachments by name, type (`type:image`), or size
- ✅ **Visual categorization** - Icons for different file types (📄 docs, 🖼️ images, 📊 spreadsheets, 📦 archives)
- ✅ **Keyboard navigation** - Arrow keys to browse, Enter to download, 1-9 for quick access
- ✅ **Multiple file types** - Support for documents, images, archives, audio, video, and more
- ✅ **Save controls** - Download to default location or use `Ctrl+S` to save as
- ✅ **Smart file naming** - Automatic filename conflict resolution with incremental numbering
- ✅ **Auto-open option** - Configurable automatic opening of downloaded files
- ✅ **Size-aware display** - Human-readable file sizes (KB, MB, GB) with MIME type info

### 🌐 **Gmail Web Integration** 🆕
- ✅ **Open in Gmail** - Press `O` to open current message in Gmail web interface
- ✅ **Complex message handling** - Perfect for messages better viewed in full Gmail UI
- ✅ **Cross-platform browser opening** - Native browser launching on macOS, Linux, Windows
- ✅ **Command parity** - Use `:gmail`, `:web`, `:open-web`, or `:o` commands
- ✅ **Smart URL generation** - Direct links to specific Gmail messages
- ✅ **Configurable shortcut** - Customize the keyboard shortcut via `open_gmail` in config
- ✅ **Seamless workflow** - Select message in TUI, open in web for detailed work

### 📅 **Calendar Integration** 🆕
- ✅ **Smart invitation detection** - Automatically detects calendar invitations in emails
- ✅ **Enhanced RSVP widget** - Press `Shift+V` to respond to meeting invitations
- ✅ **Meeting details display** - Shows title, organizer, date/time in beautiful colors
- ✅ **iCalendar parsing** - Handles complex timezone-aware calendar data
- ✅ **Direct calendar integration** - Updates your Google Calendar with RSVP responses
- ✅ **Multiple response options** - Accept, Tentative, or Decline with one key press
- ✅ **Clean visual design** - Color-coded information with proper spacing

### 📱 Adaptive Layout System
- ✅ **Responsive design** - Automatically adapts to terminal size
- ✅ **Multiple layout modes** - Wide, medium, narrow, and mobile layouts
- ✅ **Real-time resizing** - Layout changes as you resize your terminal
- ✅ **Fullscreen mode** - Press 'f' for fullscreen text view
- ✅ **Focus switching** - Press 't' to toggle between list and text focus

### 🎯 User Experience
- ✅ **Fully customizable keyboard shortcuts** - Configure any shortcut through config.json
- ✅ **Multiple shortcut styles** - Support for Vim, Emacs, and custom shortcut schemes
- ✅ **25+ configurable actions** - Customize core email operations and additional features
- 🎨 **Inspired by `k9s`, `neomutt`, `alpine`**
- ⌨️ **100% keyboard navigation**
- ⚡ **Efficient and fast interface**
- 🔧 **Highly configurable**
- 🔒 **Private** - No data sent to external cloud services

#### Welcome Screen
- On startup, a structured Welcome screen appears:
  - Title and short description
  - Quick actions: `[? Help] [s Search] [u Unread] [: Commands]`
  - If authenticated, shows `Account: <your@email>`
  - A non-blocking “⏳ Loading inbox…” indicator while the inbox loads in the background
- If credentials are missing, the Welcome screen shows a compact setup guide with the credentials path.

## 🏗️ Architecture

Gmail TUI uses a **clean, service-oriented architecture** with proper separation of concerns, thread-safe state management, and centralized error handling.

### 📁 Project Structure
```
giztui/
├── cmd/                    # Application entry points
│   └── main.go            # Main application entry
├── internal/              # Private application packages
│   ├── config/           # Configuration management
│   │   └── config.go     # Config struct and loading
│   ├── db/               # Database and caching layer
│   │   ├── cache.go      # SQLite cache implementation
│   │   ├── migrations.go # Database migrations
│   │   └── obsidian_store.go # Obsidian integration storage
│   ├── obsidian/         # Obsidian integration module
│   │   ├── ingestion.go  # Email ingestion to Obsidian
│   │   └── templates.go  # Template processing
│   ├── services/         # Business logic layer
│   │   ├── interfaces.go # Service interfaces
│   │   ├── email_service.go      # Email operations
│   │   ├── ai_service.go         # LLM integration
│   │   ├── label_service.go      # Label management
│   │   ├── cache_service.go      # Caching operations
│   │   ├── bulk_prompt_service.go # Bulk prompt processing
│   │   ├── slack_service.go      # Slack integration
│   │   └── repository.go         # Data access abstraction
│   └── tui/              # Terminal user interface
│       ├── app.go        # Main application struct
│       ├── keys.go       # Keyboard shortcuts
│       ├── commands.go   # Command handling
│       ├── ui.go         # UI layout and components
│       ├── email.go      # Email display logic
│       ├── labels.go     # Label management UI
│       ├── search.go     # Search functionality
│       ├── bulk_prompts.go # Bulk prompt operations UI
│       ├── slack.go      # Slack integration UI
│       └── error_handler.go # Centralized error handling
├── examples/             # Configuration examples
│   └── config.json      # Example configuration
├── docs/                # Documentation and proposals
│   ├── ARCHITECTURE.md  # Detailed architecture guide
│   └── proposals/       # Design proposals and RFCs
└── README.md           # This file
```

### 🔧 Service Architecture

The application follows a **robust, service-oriented architecture** with clear separation between UI, business logic, and data access:

#### 📊 **Service Layer** (`internal/services/`)
- **EmailService**: High-level email operations (compose, send, archive, etc.)
- **AIService**: LLM integration with caching and streaming support  
- **LabelService**: Gmail label management operations
- **CacheService**: SQLite-based caching for AI summaries
- **PromptService**: 🆕 Prompt library management with caching and usage tracking
- **ObsidianService**: 🆕 Email-to-Obsidian ingestion with template support
- **BulkPromptService**: 🆕 Bulk prompt processing with progress tracking and caching
- **SlackService**: 🆕 Slack integration for email forwarding with multiple format styles
- **MessageRepository**: Data access abstraction for Gmail API

#### 🎯 **Key Architectural Improvements**

1. **Service Interfaces** - Clean contracts for business logic
   ```go
   type EmailService interface {
       ArchiveMessage(ctx context.Context, messageID string) error
       TrashMessage(ctx context.Context, messageID string) error
       // ... other operations
   }
   ```

2. **Centralized Error Handling** - Consistent user feedback across the app
   ```go
   app.GetErrorHandler().ShowError(ctx, "Failed to archive message")
   app.GetErrorHandler().ShowSuccess(ctx, "Message archived successfully")
   ```

3. **Thread-Safe State Management** - Safe concurrent access to app state
   ```go
   currentView := app.GetCurrentView()        // Thread-safe read
   app.SetCurrentMessageID(messageID)         // Thread-safe write
   ```

4. **Dependency Injection** - Services are automatically initialized and injected
   ```go
   emailService, aiService, labelService, cacheService, repository, promptService, obsidianService := app.GetServices()
   ```

5. **🆕 Enhanced Bulk Operations** - Consistent patterns for all bulk operations
   ```go
   // All bulk operations follow the same pattern:
   // 1. Progress tracking with ErrorHandler
   // 2. Async processing with status updates  
   // 3. Deadlock-free UI updates
   // 4. Proper cleanup and state management
   ```

6. **🆕 Improved Threading Model** - Prevents UI deadlocks
   ```go
   // Fixed: ErrorHandler calls outside QueueUpdateDraw to avoid deadlocks
   a.QueueUpdateDraw(func() {
       // UI updates only
   })
   // Status messages outside to prevent nested QueueUpdateDraw
   a.GetErrorHandler().ShowSuccess(ctx, "Operation completed")
   ```

#### 🛡️ **Benefits**
- **Better Testability** - Services can be easily mocked and unit tested
- **Cleaner Code** - UI components focus on presentation, not business logic
- **Thread Safety** - Proper mutex protection for concurrent operations
- **Consistent UX** - Centralized error handling provides uniform user feedback
- **Deadlock Prevention** - 🆕 Improved threading prevents UI hangs
- **Maintainability** - Clear separation makes the codebase easier to understand and modify
- **Extensibility** - New features can be added by implementing service interfaces

### 🔄 **Data Flow**
```
User Input → TUI Components → Services → Repository → Gmail API
                ↓                ↓
           Error Handler ← Business Logic
```

This architecture ensures that business logic is separated from UI concerns, making the application more maintainable, testable, and robust.

## 🚀 Installation

### Prerequisites

1. **Go 1.21+** - [Download](https://golang.org/dl/)
2. **Ollama** (optional, for AI) - [Install](https://ollama.ai/)
3. **Gmail API Credentials** - [Setup](#gmail-api-setup)

### Installation from source

```bash
# Clone the repository
git clone https://github.com/ajramos/gmail-tui.git
cd gmail-tui

# Install dependencies
go mod tidy

# Build
go build -o gmail-tui cmd/gmail-tui/main.go

# Run
./gmail-tui
```

### Installation with Go install

```bash
go install github.com/ajramos/gmail-tui/cmd/gmail-tui@latest
```

## 📁 Configuration

The application uses a unified configuration directory structure:

```
~/.config/giztui/
├── config.json      # Application configuration
├── credentials.json # Gmail API credentials (OAuth2)
└── token.json      # OAuth2 token cache
```

**Migration from gmail-tui**: If you previously used Gmail TUI with the old `~/.config/gmail-tui/` directory, simply copy your files to the new location:

```bash
# One-time migration (if you have an old gmail-tui directory)
cp -r ~/.config/gmail-tui/* ~/.config/giztui/
```

### Setup Steps:

1. **Create the configuration directory:**
   ```bash
   mkdir -p ~/.config/giztui
   ```

2. **Download Gmail API credentials:**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable the Gmail API
   - Create OAuth2 credentials (Desktop application)
   - Download the JSON file and save it as `~/.config/giztui/credentials.json`
   - See `examples/credentials.json.example` for the expected format

3. **Copy the example configuration:**
   ```bash
   cp examples/config.json ~/.config/giztui/config.json
   ```

4. **Optional: Configure Ollama for AI features:**
   - Install [Ollama](https://ollama.ai/)
   - Start Ollama service
   - Update the configuration with your preferred model

## 🎮 Usage

### Basic commands

```bash
# First time setup (interactive wizard)
./gmail-tui --setup

# Run with default configuration (zero parameters needed)
./gmail-tui

# Use custom configuration file
./gmail-tui --config ~/custom-config.json

# Specify custom credentials (rarely needed)
./gmail-tui --credentials ~/path/to/credentials.json
```

### Environment Variables

For advanced users or automation:

```bash
# Override default paths
export GMAIL_TUI_CONFIG=~/.config/giztui/config.json
export GMAIL_TUI_CREDENTIALS=~/.config/giztui/credentials.json
export GMAIL_TUI_TOKEN=~/.config/giztui/token.json

# Run with environment settings
./gmail-tui
```

### Command Line Options

The CLI has been simplified to focus on essential parameters:

```bash
Gmail TUI - Terminal-based Gmail client

Usage:
  gmail-tui [options]

Examples:
  gmail-tui                        # Run with default configuration
  gmail-tui --setup                # Run interactive setup wizard
  gmail-tui --config custom.json   # Use custom configuration

Options:
  --config string
        Path to JSON configuration file (default: ~/.config/giztui/config.json)
  --credentials string
        Path to OAuth client credentials JSON (default: ~/.config/giztui/credentials.json)
  --setup
        Run interactive setup wizard

Environment Variables:
  GMAIL_TUI_CONFIG      Override default config file path
  GMAIL_TUI_CREDENTIALS Override default credentials file path
  GMAIL_TUI_TOKEN       Override default token file path

For all other settings (LLM, timeouts, etc.), edit the config file.
```

### 🎯 **Command Parity with Shortcuts**

Every keyboard shortcut has an equivalent command for better accessibility and discoverability:

| Keyboard | Command | Action |
|----------|---------|--------|
| `a` | `:archive` or `:a` | Archive message(s) |
| `d` | `:trash` or `:d` | Move to trash |
| `t` | `:read` or `:toggle-read` or `:t` | Toggle read/unread status |
| `U` | `:undo` or `:U` | 🆕 **Undo last action** |
| `r` | `:reply` or `:r` | Reply to message |
| `E` | `:reply-all` or `:ra` | Reply to all recipients |
| `w` | `:forward` or `:f` | Forward message |
| `c` | `:compose` or `:new` | Compose new message |
| `D` | `:drafts` or `:dr` | View and edit drafts |
| `N` | `:load` or `:more` or `:next` | Load next 50 messages |
| `u` | `:unread` or `:u` | Show unread messages |
| `B` | `:archived` or `:b` | Show archived messages |
| `R` | `:refresh` | Refresh message list |
| `s` | `:search` | Search messages |
| `l` | `:labels` or `:l` | Manage labels |
| `L` | `:links` or `:link` | Open link picker |
| `A` | `:attachments` or `:attach` | Open attachment picker |
| `h` | `:headers` or `:toggle-headers` | Toggle header visibility |
| `space` | N/A | Select/deselect message in bulk mode (configurable via `bulk_select`) |
| `K` | `:slack` | Forward to Slack |

**Features:**
- ✅ **Bulk mode support** - Commands automatically detect bulk mode and act on selected messages
- ✅ **Context awareness** - `:refresh` loads drafts when in draft mode, messages otherwise
- ✅ **Short aliases** - Most commands have both full names and short aliases
- ✅ **Autocompletion** - All commands work with the existing Tab completion system

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `Enter` | View selected message |
| `r` | Refresh (in drafts mode, reload drafts) |
| `c` | ✅ **Compose new message** - Full composition with CC/BCC |
| `N` | Load next 50 messages (when list is focused) |
| `R` | ✅ **Reply to message** - Full composition with threading |
| `s` | Search |
| `/` | Local filter |
| `F` | Quick search: from current sender |
| `T` | Quick search: to current sender (includes Sent) |
| `S` | Quick search: by current subject |
| `B` | Quick search: archived messages |
| `:` | Open command bar (k9s-style) |
| `u` | Show unread |
| `t` | Toggle read/unread |
| `U` | 🆕 **Undo last action** - Reverse archive, trash, read/unread, or label operations |
| `d` | Move to trash |
| `a` | Archive |
| `D` | ✅ **View and edit drafts** - Side panel picker with auto-save |
| `A` | View attachments (WIP) |
| `w` | Save current message to file (.txt, rendered) |
| `W` | Save current message as raw .eml (server format) |
| `l` | Manage labels (contextual panel) |
| `L` | 🆕 **Open link picker** |
| `m` | Move message (choose label) |
| `M` | Toggle Markdown rendering |
| `h` | 🆕 **Toggle header visibility** - Hide/show email headers to maximize content space |
| `y` | Toggle AI summary |
| `Y` | Regenerate AI summary (force refresh; ignores cache) |
| `g` | Generate reply (experimental) |
| `p` | Open prompt picker (single message) or bulk prompt picker (bulk mode) |
| `K` | Forward email to Slack |
| `O` | 🆕 **Ingest email(s) to Obsidian** (single or bulk mode) |
| `Esc` | Cancel active streaming operations (AI summary, prompts, bulk prompts) |

#### 🏃 VIM-Style Navigation

Gmail TUI supports VIM-style navigation for efficient message browsing:

**Command-based navigation (`:` prefix):**
| Command | Action |
|---------|--------|
| `:5` | Jump to message 5 |
| `:1` | Jump to first message |
| `:$` | Jump to last message |
| `:G` | Jump to last message |

**Direct shortcuts (VIM-style):**
| Key | Action |
|-----|--------|
| `G` | Jump to last message |
| `gg` | Jump to first message (press 'g' twice quickly) |

#### 📖 **Enhanced Content Navigation** 🆕

Gmail TUI provides fast navigation within individual message content for better browsability of long emails:

**Content Search:**
| Key/Command | Action |
|-------------|--------|
| `/searchterm` | Search within message content and highlight matches |
| `n` | Navigate to next search match |
| `N` | Navigate to previous search match |
| `Esc` | Clear search highlights |

**Fast Content Navigation:**
| Key | Action |
|-----|--------|
| `gg` | Go to top of message content (when viewing message) |
| `G` | Go to bottom of message content (when viewing message) |
| `Ctrl+K` | Navigate up by paragraphs (10 lines) |
| `Ctrl+J` | Navigate down by paragraphs (10 lines) |
| `Ctrl+H` | Navigate left by words |
| `Ctrl+L` | Navigate right by words |

**Features:**
- ✅ **Context-aware** - VIM keys work differently when viewing message content vs message list
- ✅ **Visual feedback** - Shows current position and line numbers during navigation
- ✅ **Smart search** - Highlights all matches with yellow background
- ✅ **Boundary handling** - Graceful behavior at content edges

#### 🎯 VIM Range Operations

Gmail TUI supports VIM-style range operations for efficient bulk actions:

**Range operation syntax:** `{operation}{count}{operation}`

| Range Command | Action | Example |
|---------------|---------|---------|
| `t3t` | Toggle read status for 3 messages | Toggles messages 1-3 |
| `a5a` | Archive 5 messages | Archives messages 1-5 |
| `d2d` | Delete (trash) 2 messages | Moves messages 1-2 to trash |
| `s4s` | Select 4 messages (bulk mode) | Selects messages 1-4 |
| `m7m` | Move 7 messages | Opens move dialog for messages 1-7 |
| `l3l` | Label 3 messages | Opens label picker for messages 1-3 |

**How it works:**
1. **First key** - Start operation (`t`, `a`, `d`, etc.)
2. **Number** - Specify count (`3`, `5`, `2`, etc.) 
3. **Same key** - Complete operation (`t`, `a`, `d`, etc.)

**Examples:**
- Type `t3t` → Shows "Toggling read status for 3 messages (t3t)"
- Type `a5a` → Shows "Archiving 5 messages (a5a)"  
- Type `d2d` → Shows "Moving 2 messages to trash (d2d)"

**Features:**
- ✅ **Real-time feedback** - Status shows exact VIM sequence typed
- ✅ **Range validation** - Automatically limits to available messages
- ✅ **Configurable timeouts** - Customize timing for VIM sequences (see Configuration section)
- ✅ **Timeout fallback** - Single operation if no count provided within timeout
- ✅ **ESC cancellation** - Cancel any range operation with Escape

**Navigation examples:**
- Type `:10` + Enter → Jump to message 10
- Press `G` → Jump to last message  
- Press `g` then `g` quickly → Jump to first message
- Type `:$` + Enter → Jump to last message

#### 🔧 Other shortcuts

| Key | Action |
|-----|--------|
| `o` | Suggest label |
| `q` | Quit |

#### 🚀 **Bulk Operations** 🆕

Bulk operations allow you to select multiple messages and perform actions on them simultaneously:

| Key | Action |
|-----|--------|
| `v`, `b` or `space` | Enter bulk mode and select current message |
| `space` | Toggle selection of current message (in bulk mode) |
| `*` | Select all visible messages |
| `a` | Archive selected messages |
| `d` | Move selected messages to trash |
| `t` | 🆕 **Toggle read/unread status for selected messages** |
| `m` | Move selected messages to label |
| `p` | Apply AI prompt to all selected messages |
| `K` | 🆕 **Forward selected messages to Slack** |
| `O` | 🆕 **Ingest selected messages to Obsidian** |
| `Esc` | Exit bulk mode |

**Bulk Mode Status Bar:**
- Shows current selection count
- Displays available actions: `space/v=select, *=all, a=archive, d=trash, t=read/unread, m=move, p=prompt, K=slack, O=obsidian, ESC=exit`

### AI Features (LLM)

| Key | Action |
|-----|--------|
| `y` | Summarize message |
| `Y` | Regenerate AI summary (force refresh) |
| `g` | Generate reply (experimental) |
| `o` | Suggest label |
| `P` | 🆕 **Open Prompt Library** |

### 💬 **Slack Integration** 🆕

| Key | Action |
|-----|--------|
| `K` | Forward email to Slack (contextual panel) |
| `:slack` | Forward email to Slack (command equivalent) |
| `:slack 5` | Forward message #5 to Slack (command with message number) |
| `:numbers` or `:n` | Toggle message number display in list |

**Message Numbers:**
- **`:numbers`** or **`:n`** toggles display of message numbers (1, 2, 3...) in the message list
- Numbers appear at the leftmost position and are right-aligned for clean formatting
- Use with navigation commands like `:5` (jump to message 5) or `:slack 10` (forward message 10)
- Perfect for quickly referencing specific messages by their position

### 🌐 **Gmail Web & Link Commands** 🆕

| Key | Action |
|-----|--------|
| `O` | Open current message in Gmail web interface |
| `L` | Open link picker for message links |
| `A` | Open attachment picker for message attachments |
| `:gmail`, `:web`, `:open-web`, `:o` | Open current message in Gmail web (command equivalents) |
| `:links` or `:link` | Open link picker (command equivalent) |
| `:attachments` or `:attach` | Open attachment picker (command equivalent) |

**Single Email Usage:**
1. **Select a message** in the message list
2. **Press `K`** or **type `:slack`** to open the Slack forwarding panel
3. **Or type `:slack 5`** to open panel for message #5 (auto-selects message)
4. **Choose a channel** from the configured list
5. **Add optional message** (e.g., "Hey team, heads up with this email")
6. **Press Enter** to send or **Tab** to switch focus between channel list and message input
7. **Press Esc** to close the panel

**Bulk Email Usage:** 🆕
1. **Enter bulk mode** by pressing `v`, `b`, or `space` on a message
2. **Select multiple emails** using `space` to toggle, `*` to select all
3. **Press `K`** to open the bulk Slack forwarding panel  
4. **Choose a channel** from the configured list (use arrow keys)
5. **Add bulk comment** that will be included with ALL forwarded emails
6. **Use Tab** to navigate between channel list and comment input field
7. **Press Enter** to forward all selected emails with progress tracking
8. **Press Esc** to cancel bulk forwarding

**Format Styles:**
- **📄 Summary** - AI-generated summary using your custom prompt template
- **📦 Compact** - Headers + 200 character preview (From, Subject, body snippet)
- **📧 Full** - Complete email with TUI-processed content (includes LLM touch-up if enabled)
- **🔧 Raw** - Minimal processing, basic HTML-to-text conversion only

**Features:**
- ✅ **Single & bulk forwarding** - Forward one email or multiple emails simultaneously 
- ✅ **Bulk comments** - Add shared comments that appear on all forwarded emails
- ✅ **Progress tracking** - Real-time progress updates for bulk operations (e.g., "Forwarding 3/5 to #general...")
- ✅ **Multi-channel support** - Configure multiple Slack channels with individual webhooks
- ✅ **Smart variable substitution** - AI prompts can use `{{body}}`, `{{subject}}`, `{{from}}`, `{{to}}`, `{{cc}}`, `{{bcc}}`, `{{date}}`, and more
- ✅ **TUI content fidelity** - "Full" format shows exactly what you see in the message widget
- ✅ **Contextual UI** - Panel appears as a widget on the home screen (like labels)
- ✅ **Optional user messages** - Add personal context when forwarding emails
- ✅ **Robust error handling** - Individual failures don't stop bulk operations

#### 🚀 **Prompt Library System** 🆕

The Prompt Library allows you to apply custom AI prompts to emails for various purposes like analysis, action item extraction, or custom processing.

**Usage:**
1. **Select a message** in the message list
2. **Press `P`** to open the Prompt Library
3. **Choose a prompt** from the list or search by typing
4. **View results** in the AI panel with real-time streaming

**Available Prompts:**
- **Quick Summary** - Concise email summary
- **Action Items** - Extract actionable tasks and deadlines
- **Key Points** - Identify main topics and insights
- **Follow-up Required** - Determine if response is needed
- **Custom Prompts** - Add your own prompt templates

**Features:**
- ✅ **Variable Substitution** - Auto-complete `{{from}}`, `{{subject}}`, `{{body}}`, `{{date}}`, `{{messages}}`
- ✅ **Streaming Responses** - Real-time token streaming for immediate feedback
- ✅ **Smart Caching** - Results cached to avoid re-processing
- ✅ **Usage Tracking** - Monitor prompt usage patterns
- ✅ **Split-View Interface** - Non-intrusive prompt picker (like labels)

#### 🔄 **Variable Substitution**

The prompt system supports different variables depending on the context:

**Single Message Prompts** (when pressing `p` on one message):
- `{{from}}` - Sender's email address
- `{{subject}}` - Email subject line
- `{{date}}` - Email date
- `{{body}}` - Single message content

**Bulk Message Prompts** (when pressing `p` in bulk mode with multiple selected messages):
- `{{body}}` - Combined content from all selected messages (legacy support)
- `{{messages}}` - Combined content from all selected messages (recommended for bulk)

**Combined Message Format:**
When using bulk prompts, the `{{body}}` or `{{messages}}` variable contains all selected messages formatted like:
```
---START EMAILS---
---START EMAIL 1---
[First email content]
---END EMAIL 1---
---START EMAIL 2---
[Second email content]
---END EMAIL 2---
---END OF EMAILS---
```

**Example Single Message Prompt:**
```
Extract action items and deadlines from this email:

From: {{from}}
Subject: {{subject}}
Date: {{date}}

Content: {{body}}

Please identify:
1. Specific action items
2. Deadlines mentioned
3. Follow-up required
4. Priority level
```

**Example Bulk Message Prompt:**
```
Analyze these project update emails and provide a consolidated summary:

{{messages}}

Please organize the information by:
1. **Key Achievements** - What was accomplished
2. **Current Issues** - Problems or blockers mentioned
3. **Upcoming Deadlines** - Important dates across all emails
4. **Action Items** - Tasks that need attention
5. **Overall Project Status** - High-level assessment

Format your response with clear sections and bullet points.
```

#### 💬 **Slack Configuration** 🆕

Enable Slack integration by adding the configuration to `~/.config/giztui/config.json`:

```json
{
  "slack": {
    "enabled": true,
    "channels": [
      {
        "id": "general",
        "name": "General",
        "webhook_url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
        "default": true,
        "description": "General team updates"
      },
      {
        "id": "urgent",
        "name": "Urgent Alerts",
        "webhook_url": "https://hooks.slack.com/services/YOUR/URGENT/WEBHOOK",
        "default": false,
        "description": "For urgent notifications only"
      }
    ],
    "defaults": {
      "format_style": "summary"
    },
    "summary_prompt": "You are a precise email summarizer. Extract only factual information from the email below. Do not add opinions, interpretations, or information not present in the original email.\n\nRequirements:\n- Maximum {{max_words}} words\n- Preserve exact names, dates, numbers, and technical terms\n- If forwarding urgent/important items, start with \"[URGENT]\" or \"[ACTION REQUIRED]\" only if explicitly stated\n- Do not infer emotions or intentions not explicitly stated\n- If email contains meeting details, preserve exact time/date/location\n- If email contains action items, list them exactly as written\n\nEmail to summarize:\n{{body}}\n\nProvide only the factual summary, nothing else."
  }
}
```

**Configuration Fields:**

- **`enabled`** - Enable/disable Slack integration
- **`channels[]`** - Array of configured Slack channels:
  - **`id`** - Unique identifier for the channel
  - **`name`** - Display name shown in the UI
  - **`webhook_url`** - Slack webhook URL for posting messages
  - **`default`** - Whether this channel is pre-selected in the UI
  - **`description`** - Optional description (not shown in UI)
- **`defaults.format_style`** - Default format: `"summary"`, `"compact"`, `"full"`, or `"raw"`
- **`summary_prompt`** - AI prompt template for generating email summaries

**Available Variables for Prompts:**
All email headers and content are available as variables in your custom prompts:
- **Core**: `{{body}}`, `{{subject}}`, `{{from}}`, `{{date}}`
- **Recipients**: `{{to}}`, `{{cc}}`, `{{bcc}}`
- **Technical**: `{{reply-to}}`, `{{message-id}}`, `{{in-reply-to}}`, `{{references}}`
- **Special**: `{{max_words}}` - Word limit for summaries

**Setup Steps:**
1. **Create Slack webhook URLs** in your Slack workspace
2. **Add webhook URLs** to the channel configurations
3. **Set `"enabled": true`** to activate the feature
4. **Customize format style** and summary prompt as needed
5. **Press `K`** in Gmail TUI to start forwarding emails

#### Configuration Directory Structure

GizTUI uses a unified configuration directory at `~/.config/giztui/`:

```
~/.config/giztui/               # Main configuration directory
├── config.json                # Main configuration file
├── credentials.json            # OAuth credentials
├── token.json                  # OAuth token
├── cache/                      # SQLite cache files
└── templates/                  # Template files
    ├── ai/                     # AI/LLM prompt templates
    │   ├── summarize.md
    │   ├── reply.md
    │   ├── label.md
    │   └── touch_up.md
    ├── slack/                  # Slack integration templates
    │   └── summary.md
    └── obsidian/               # Obsidian integration templates
        └── email.md
```

#### Path Resolution Rules

- **Absolute paths**: Full paths starting with `/` or `~` (e.g., `~/.config/giztui/credentials.json`)
- **Relative paths**: Resolved relative to the config directory `~/.config/giztui/` (e.g., `templates/ai/summarize.md` → `~/.config/giztui/templates/ai/summarize.md`)

#### LLM Configuration (providers)

Configure AI/LLM settings under the unified `llm` object in `~/.config/giztui/config.json`:

```json
{
  "llm": {
    "enabled": true,
    "provider": "ollama",
    "model": "llama3.2:latest",
    "endpoint": "http://localhost:11434/api/generate",
    "api_key": "",
    "timeout": "20s",
    "stream_enabled": true,
    "stream_chunk_ms": 60,
    "cache_enabled": true,
    "cache_path": "",
    "summarize_template": "templates/ai/summarize.md",
    "reply_template": "templates/ai/reply.md",
    "label_template": "templates/ai/label.md",
    "touch_up_template": "templates/ai/touch_up.md"
  }
}
```

#### Template Files System

AI prompts are now stored in external Markdown files for better editing and version control:

**Directory Structure:**
```
~/.config/giztui/
├── config.json
└── templates/
    ├── ai/
    │   ├── summarize.md
    │   ├── reply.md
    │   ├── label.md
    │   └── touch_up.md
    └── slack/
        └── summary.md
```

**Template Path Examples:**
- `"summarize_template": "templates/ai/summarize.md"` → `~/.config/giztui/templates/ai/summarize.md`
- `"summary_template": "templates/slack/summary.md"` → `~/.config/giztui/templates/slack/summary.md`
- `"template": "templates/obsidian/email.md"` → `~/.config/giztui/templates/obsidian/email.md`
- `"template": "/path/to/custom/template.md"` → `/path/to/custom/template.md` (absolute)
- `"template": "~/my-templates/custom.md"` → `~/my-templates/custom.md` (home directory)

**Template Loading Priority:**
1. **Template files** (if path specified and file exists) - takes precedence
2. **Inline prompts** (if specified in config) - fallback for simple cases
3. **Built-in defaults** (if neither above are available)

This file-first priority design ensures that when you specify a template file path, it will always be used (no need for empty `*_prompt` fields to override defaults).

**Template Setup:**
To use template files, copy the default templates from the repository:
```bash
# Copy default templates to your config directory
cp -r templates/ ~/.config/giztui/
```

The repository includes ready-to-use template files in the `templates/` directory that you can customize:
- **AI templates**: `templates/ai/` - For LLM prompts (summarize, reply, label, touch_up)
- **Slack templates**: `templates/slack/` - For Slack integration prompts  
- **Obsidian templates**: `templates/obsidian/` - For Obsidian note formatting

**Benefits:**
- Easy editing with proper syntax highlighting in your favorite editor
- Better version control for custom prompts
- Cleaner configuration files
- Shareable template collections
- Default templates included in repository for easy setup

#### Keyboard Configuration

Customize keyboard shortcuts and VIM sequence timeouts in `~/.config/giztui/config.json`:

```json
{
  "keys": {
    "vim_navigation_timeout_ms": 1000,
    "vim_range_timeout_ms": 2000,
    "compose": "n",
    "trash": "d", 
    "archive": "a",
    "toggle_read": "t",
    "manage_labels": "l",
    "content_search": "/",
    "search_next": "n",
    "search_prev": "N",
    "fast_up": "ctrl+k",
    "fast_down": "ctrl+j",
    "goto_top": "gg",
    "goto_bottom": "G"
  }
}
```

**VIM Timeout Configuration:**
- `vim_navigation_timeout_ms` - Timeout for `gg` navigation sequences (default: 1000ms)
- `vim_range_timeout_ms` - Timeout for bulk operations like `d3d` (default: 2000ms)

Customize these values to make VIM sequences faster or slower based on your typing speed:
- **Faster users**: Set lower values (e.g., 500ms for navigation, 1000ms for ranges)
- **Slower typists**: Set higher values (e.g., 1500ms for navigation, 3000ms for ranges)

##### Amazon Bedrock (on-demand)

To use Amazon Bedrock instead of Ollama:

```json
{
  "llm": {
    "enabled": true,
    "provider": "bedrock",
    "model": "us.anthropic.claude-3-5-haiku-20241022-v1:0",
    "region": "us-east-1",
    "timeout": "20s"
  }
}
```

Notes for Bedrock on-demand:

- Always include the revision suffix `:0` in `llm_model` for on-demand model IDs.
- In many accounts, you must include the regional vendor prefix, e.g. `us.anthropic...` rather than just `anthropic...`.
- Make sure your AWS credentials are set (e.g., `AWS_PROFILE=your-profile`) and the region has access to the model.
- Alternatively, you can provide an inference profile ARN in `llm_model` and it will be sent via `ModelId`.

Run with a custom config:

```bash
AWS_PROFILE=your-profile ./gmail-tui --config ~/.config/giztui/config.bedrock.json
```

Minimal debugging example (standalone):

```bash
# Build the example
go build -o build/bedrock_text ./examples/bedrock_text.go

# Invoke on-demand
AWS_PROFILE=your-profile ./build/bedrock_text \
  --region us-east-1 \
  --model us.anthropic.claude-3-5-haiku-20241022-v1:0 \
  --prompt "Summarize this in one line"
```

References: Bedrock Go v2 examples for streaming and invocation (ModelId) — see AWS examples repository.

#### Prompt templates

Prompts for AI features are configurable via `~/.config/giztui/config.json`.

- `summarize_prompt`: Used when pressing `y` to summarize the current email. Supports the placeholder `{{body}}` which is replaced with the email plain text.
- `label_prompt`: Used when pressing `o` to suggest labels. Supports placeholders `{{labels}}` (comma-separated list of allowed labels) and `{{body}}` (email plain text).

Example configuration snippet:

```json
{
  "summarize_prompt": "Briefly summarize the following email in the same language as the input. Keep it concise and factual.\n\n{{body}}",
  "label_prompt": "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: {{labels}}\n\nEmail:\n{{body}}"
}
```

Notes:
- If a prompt is empty or missing, a sensible default will be used.
- Changes to `config.json` are picked up on application start. Please restart the app after editing the configuration.
- When streaming is enabled and supported (Ollama), summaries appear incrementally with status “🧠 Streaming summary…”. If streaming is unavailable, it falls back to a single final render.

## 🧰 Local Cache (SQLite)

The app uses an embedded SQLite database (no external server) to cache AI summaries and prompt results:

- **Default location**: `~/.config/giztui/cache/{account_email}.sqlite3`
- **Per-account separation** by filename
- **PRAGMAs tuned for TUI** (WAL, foreign keys, timeouts)
- **Multiple cache types**: AI summaries, single prompts, bulk prompts

### 📊 **Cache Tables:**
- `ai_summaries` - AI-generated email summaries
- `prompt_results` - Single message prompt results  
- `bulk_prompt_results` - Multi-message bulk prompt results
- `prompt_templates` - Custom prompt templates

### 🎯 **Cache Benefits:**
- **Instant retrieval** of previously processed prompts
- **Cost savings** by avoiding duplicate LLM calls
- **Offline access** to cached results
- **Performance boost** for repeated operations

Configuration snippet:

```json
{
  "ai_summary_cache_enabled": true,
  "ai_summary_cache_path": ""
}
```

- If `ai_summary_cache_path` is empty, a sensible per-account default is used; otherwise, the given path is used as the DB file or directory.

### Summary refresh

- Press `Y` (uppercase) to forcefully regenerate the AI summary for the current message (ignores cache).
- Command mode: `:summary refresh`.

### 🗑️ **Cache Management**

Manage your local cache through command mode for better performance and storage control:

**Cache Information:**
```bash
:cache info          # Show current account and database location
```

**Clear Cache:**
```bash
:cache clear         # Clear all prompt caches for current account
:cache clear all     # Clear all caches for all accounts (admin)
```

**Cache Commands:**
- `:cache info` - Display account email and database file location
- `:cache clear` - Remove all cached prompt results for your account
- `:cache clear all` - Remove all cached results for all accounts
- Cache operations run asynchronously and show success/error messages

**When to Clear Cache:**
- After changing LLM providers or models
- When experiencing unexpected cached results
- To free up disk space
- After major prompt template changes

### Layout Controls

| Key | Action |
|-----|--------|
| `f` | Toggle fullscreen text view |
| `t` | Toggle focus between list and text |

### 🧭 Command Mode (k9s-style)

- Press `:` to open the command bar. It appears between the message content and the status bar.
- Look & feel: bordered panel with title, prompt icon and a `>` chevron.
- Focus: the input takes focus automatically; `ESC` closes. If the command bar ever loses focus (e.g., due to background loads), it auto-hides for consistency.
- Autocompletion: type partial commands and press `Tab` to complete (e.g., `:la` → `labels`).
- Suggestions: shown live in brackets on the right. `↑/↓` navigate history; `Enter` executes.

Supported commands: `labels`, `search`, `slack`, `inbox`, `compose`, `help`, `quit`, `cache`

RSVP (meeting invites):

- Detection: when an email contains a `text/calendar` invitation (METHOD:REQUEST), the status bar shows “📅 Calendar invite detected — press V to RSVP”.
- `V` opens a side panel to respond (ACCEPT / TENTATIVE / DECLINE).
- Equivalent command: `:rsvp accept|tentative|decline`.
- The response updates your attendance directly via Google Calendar API (Calendar scope required). Email-based ICS replies are no longer used.

Contextual search shorthands supported in command mode:

```
:search from:current      # messages from the current sender (Inbox scope by default)
:search to:current        # messages to the current sender (includes Sent; excludes spam/trash)
:search subject:current   # messages with the exact current subject
:search domain:current    # messages from the current sender's domain (@example.com)
```

### 🏷️ Labels Management (Contextual)

- Press `l` to open a side panel with labels for the current message. The panel follows the selected message.
- Status:
  - `✅` applied to message
  - `○` not applied
- Actions:
  - `Enter`: toggle label (applies immediately and refreshes message content)
  - `n`: create label, `r`: refresh
  - `ESC`: close panel (hint at bottom-right)
- Browse all labels:
  - From the panel select “🔍 Browse all labels…” to expand a full picker with search
  - Type to filter; in the search field, `Enter` applies the 1st visible match; on the list, `Enter` toggles; `ESC` returns to the quick panel (hint shown)
- Suggested labels (`o`): opens a side panel with top suggestions (same UX as labels panel), includes “🔍 Browse all labels…” and “➕ Add custom label…”. Applied suggestions are marked with `✅`.
- Focus rules:
  - Tab cycles: text → labels → summary → list
  - Arrow keys act on the currently focused pane only
  - The app does not steal focus while background work runs

#### Edit/Remove existing labels

- In the labels side panel (press `l`), you now have:
  - `📝 Edit existing label…`: Opens a picker with search titled “📝 Select label to edit”.
    - Type to filter; `Enter` on first match selects the label.
    - Inline editor titled “📝 Edit label” with the current name pre-filled. `Enter` renames, `Esc` goes back.
    - After renaming, only the current message header is refreshed; caches are updated to reflect the new label name immediately.
  - `🗑 Remove existing label…`: Opens a picker titled “🗑 Select label to remove”.
    - `Enter` selects; confirmation screen titled “🗑 Remove label”. `Enter` confirms, `Esc` cancels.
    - After deletion, only the current message header is refreshed; caches are updated and side panel is rebuilt.

Navigation niceties:
- From the list in any picker, pressing Arrow Up on the first item moves focus back to the search field.
- From the search field, using Arrow keys moves focus to the list.
- This mirrors the behavior in “🔍 Browse all labels…”.

### 📦 Move Message (Contextual)

- Press `m` to open the side panel directly in "Browse all labels" mode
  - Type to filter labels. In the search field, `Enter` applies/moves the 1st visible match. `Enter` on a list item will:
  - Apply the label (if not already applied)
  - Archive the message (move semantics)
  - Update the list and content in place
  - Close the panel automatically
- `ESC` closes the panel (no intermediate quick view)

### 🔎 Search & Filter UX

- Local filter (`/`):
  - Applies an in-memory filter over the current list (works on Inbox and on Search Results).
  - Matches against Subject, From, To, Snippet, and also against visible label chips (e.g., `Personal`, `AWS`).
  - Supports simple label filters with `label:<name>` tokens. Examples: `label:Personal`, `report label:AWS`.
  - Actions while filtered (archive, trash, move, label) propagate to the underlying snapshot (Inbox or Search Results).
  - Press `ESC` to exit the filter instantly without network calls, restoring the original list with your changes reflected.
  - After removing an item (archive/trash/move), the selection stays on the same visual position.

- Remote search (`s`):
  - Runs a Gmail query and shows a new result list.
  - Press `ESC` to return to the inbox; the inbox is reloaded from the server (source of truth).
  - Quick searches:
    - `F` → `from:<sender>`
    - `T` → `to:<sender>` (uses `in:anywhere -in:spam -in:trash` so Sent mail is included)
    - `S` → `subject:"<exact subject>"`

 - Advanced search (`Ctrl+F`):
   - Shows a form with fields: `From`, `To`, `Subject`, `Has the words`, `Doesn't have`, `Size`, `Date within`, `Search`, and `Has attachment`.
   - `Search` field with quick options: toggle the “📂 Search options” panel with `Enter/Tab`. The panel includes icons and a live filter.
     - Folders: All Mail, Inbox, Sent, Drafts, Spam, Trash
     - Anywhere: Mail & Spam & Trash
     - State: Read, Unread
     - Categories: social, updates, forums, promotions
     - Labels: all user labels
     - Navigation: type to filter; use `↑/↓` to navigate; `Enter` applies to the `Search` field and closes the panel; `Esc` closes the panel.
    - `Size`: accepts `>NKB`, `<NMB` or bytes without unit (e.g., `>1024`). Only KB/MB/bytes are supported. If invalid, a status message is shown and the form stays open.
    - `Date within`: accepts `Nd`, `Nw`, `Nm`, `Ny` and maps to a symmetric range using `after:`/`before:` around today. Example (today 2025/08/13): `3d` → `after:2025/8/10 before:2025/8/17`.
    - `📎 Has attachment`: checkbox to require attachments (`has:attachment`).
    - `Esc` behavior: if the right “Search options” panel is open, first `Esc` closes it and returns focus to the `Search` field; otherwise `Esc` exits advanced search and opens the simple search overlay.
   - Vertical layout: form on top (50%) and message content below for context.
   - Execution: move focus to the `🔎 Search` button and press Enter. The advanced search view closes and the search runs.

### 📎 Icons & 🏷️ Label Chips in the list

- The list shows attachment (📎) and calendar invite/update (🗓️) icons computed from message metadata.
- Label chips are rendered in Title Case and limited to 3; extra labels appear as `+N`.
- In Search Results we also display system labels (e.g., `Trash`, `Sent`) to provide context across folders; in Inbox normal view, redundant system labels are hidden.
- State labels like Unread/Important/Starred are not shown as chips because they are already represented with colors.

### 📐 Vertical Layout

- Messages list on top, message content below, command bar and status at bottom
- Proportions optimized for readability
- Smart focus: app avoids stealing focus while loading in background

## 📱 Layout System

The application features an **adaptive layout system** that automatically adjusts to your terminal size:

### Layout Types

- **🖥️ Wide Layout** (≥120x30): Side-by-side layout with list and text view
- **📺 Medium Layout** (≥80x25): Stacked layout with list on top, text below
- **📱 Narrow Layout** (≥60x20): Full-width layout optimized for small screens
- **📲 Mobile Layout** (<60x20): Compact layout for very small terminals

### Layout Configuration

You can customize the layout behavior in your `config.json`:

```json
{
  "layout": {
    "auto_resize": true,
    "wide_breakpoint": {
      "width": 120,
      "height": 30
    },
    "medium_breakpoint": {
      "width": 80,
      "height": 25
    },
    "narrow_breakpoint": {
      "width": 60,
      "height": 20
    },
    "default_layout": "auto",
    "show_borders": true,
    "show_titles": true,
    "compact_mode": false,
    "color_scheme": "default"
  }
}
```

### Layout Features

- **🔄 Auto-resize**: Layout automatically changes when you resize your terminal
- **🔍 Fullscreen mode**: Press 'f' to view text content in fullscreen
- **🎯 Smart focus**: Press 't' to switch focus between list and text areas
- **⚡ Performance**: Optimized rendering for each layout type

## 🎨 Theme Configuration

Gmail TUI supports customizable color themes to personalize your email experience with runtime theme switching.

### Available Commands
- `:theme` or `:th` - Show current theme  
- `:theme list` - List all available themes
- `:theme set <name>` - Switch to specified theme
- `:theme preview <name>` - Preview theme before applying

### Built-in Themes
- **slate-blue** - Modern dark theme with blue/slate palette and cyan accents ⭐ (default)
- **dracula** - Official Dracula theme with dark background and vibrant accent colors
- **gmail-dark** - Dark theme based on Dracula color scheme
- **gmail-light** - Clean light theme for bright environments
- **custom-example** - Demo custom theme showing customization possibilities

### Theme Management
```bash
# List available themes
:theme list

# Switch to light theme  
:theme set gmail-light

# Preview a theme before applying
:theme preview gmail-dark

# Check current active theme
:theme
```

### Custom Themes
1. **Built-in themes** - Located in `themes/` directory (shipped with application)
2. **User themes** - Place custom themes in `~/.config/giztui/themes/`
3. **Custom directory** - Configure `custom_dir` in config.json theme section for alternate location

**Create custom theme:**
```bash
# Copy existing theme as template
cp themes/gmail-dark.yaml ~/.config/giztui/themes/my-theme.yaml
# Edit colors in YAML file
# Apply with :theme set my-theme
```

### Configuration
Set default theme and custom theme directory in `config.json`:
```json
{
  "theme": {
    "current": "slate-blue",
    "custom_dir": "/path/to/custom/themes"  
  }
}
```

### Theme Features
- ✅ **Runtime switching** - Change themes instantly without restart
- ✅ **Visual preview** - See colors before applying themes
- ✅ **Command parity** - Every UI action has equivalent command  
- ✅ **Multi-directory support** - Built-in + user custom + configurable themes
- ✅ **Developer guidelines** - Maintain theme consistency (`docs/THEMING.md`)

See detailed documentation: `docs/THEMING.md`

## Terminal rendering: deterministic + optional LLM touch-up

The message content pane now uses a deterministic formatter designed for terminal readability:

- Preserves quotes (>), code/pre and PGP/SMIME blocks (no wrapping or changes inside)
- Converts HTML to text with numeric link references: `text [n]` in body and a `[LINKS]` section listing `(n) URL`
- Renders lists, headings and simple ASCII tables
- Adds `[ATTACHMENTS]` and `[IMAGES]` sections from MIME metadata
- Wraps lines to the available width without breaking words/URLs

An optional LLM “touch-up” layer can adjust whitespace/line breaks for nicer layout without changing content.

### Keyboard

- `M` — Toggle LLM touch-up ON/OFF for the current message view (whitespace-only formatting)
- Indicator in the status bar:
  - `🧾` deterministic only
  - `🧠` LLM touch-up enabled

Notes:
- Moving with arrow keys previews messages using deterministic formatting only (no LLM calls). LLM is applied when you open a message (Enter) and the indicator is `🧠`.
- The status bar shows progress like “🧠 Optimizing format with LLM…” while processing.
- When streaming is enabled and supported (Ollama), summaries appear incrementally with status “🧠 Streaming summary…”. If streaming is unavailable, it falls back to a single final render.

### Configuration

Config fields (in `~/.config/giztui/config.json`):

```json
{
  "LLMEnabled": true,
  "LLMProvider": "ollama",        // or "bedrock"
  "LLMEndpoint": "http://localhost:11434/api/generate", // Ollama
  "LLMRegion": "us-east-1",      // Bedrock
  "LLMModel": "llama3.2:latest",
  "LLMTimeout": "20s",
  "LLMStreamEnabled": true,       // Enable streaming for Ollama
  "AISummaryCacheEnabled": true,  // Enable AI summary caching
  "PromptLibraryEnabled": true    // 🆕 Enable Prompt Library system
}
```

#### 🚀 **Prompt Library Configuration** 🆕

The Prompt Library system is automatically initialized with default prompts on first use. You can customize the system:

**Database Location:**
- Prompts and results are stored in the same SQLite database as AI summaries
- Location: `~/.config/giztui/gmail-tui-{account}.db`

**Default Prompts:**
The system comes with pre-configured prompts:
- **Quick Summary** - Concise email summary
- **Action Items** - Extract actionable tasks and deadlines  
- **Key Points** - Identify main topics and insights
- **Follow-up Required** - Determine if response is needed

**Custom Prompts:**
You can add your own prompt templates directly to the database using SQLite commands:

```bash
# Connect to your database (replace {your-email} with your actual email)
sqlite3 ~/.config/giztui/gmail-tui-{your-email}.db

# Add a custom prompt
INSERT INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite) 
VALUES (
    'Prompt Name', 
    'Description of what this prompt does', 
    'Your prompt text with variables {{from}} {{subject}} {{body}} {{date}}', 
    'category', 
    strftime('%s', 'now'), 
    0
);

# Verify the prompt was added
SELECT * FROM prompt_templates WHERE name = 'Prompt Name';
```

**Example - Sentiment Analysis Prompt:**
```sql
INSERT INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite) 
VALUES (
    'Sentiment Analysis', 
    'Analyze the emotional tone and sentiment of the email', 
    'Analyze the emotional tone and sentiment of this email from {{from}} with subject "{{subject}}":\n\n{{body}}\n\nPlease provide:\n1. Overall sentiment (positive/negative/neutral)\n2. Emotional indicators\n3. Tone analysis\n4. Recommendations for response', 
    'analysis', 
    strftime('%s', 'now'), 
    0
);
```

**Available Categories:**
- `summary` - For summarization prompts
- `analysis` - For analysis and insights
- `action` - For action items and tasks
- `bulk_analysis` - For multi-email bulk operations
- `custom` - For your own categories

### 🗄️ **Database Prompt Management**

The prompt library uses SQLite for storage. You can directly manage prompts using standard SQL commands:

**Connect to Database:**
```bash
# Replace {your-email} with your actual email address
sqlite3 ~/.config/giztui/gmail-tui-{your-email}.db
```

**View Existing Prompts:**
```sql
-- List all prompts with basic info
SELECT id, name, category, is_favorite, usage_count FROM prompt_templates;

-- View a specific prompt's details
SELECT * FROM prompt_templates WHERE name = 'Quick Summary';

-- List prompts by category
SELECT id, name, description FROM prompt_templates WHERE category = 'analysis';
```

**Add New Prompts:**
```sql
-- Basic prompt template
INSERT INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite) 
VALUES (
    'Email Classifier',
    'Classify emails into categories',
    'Classify this email from {{from}} with subject "{{subject}}" into one of: Important, Spam, Newsletter, Personal, Work.\n\nEmail content:\n{{body}}\n\nProvide only the category name.',
    'analysis',
    strftime('%s', 'now'),
    0
);

-- Bulk analysis prompt (use {{messages}} or {{body}} for combined content)
INSERT INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite) 
VALUES (
    'Weekly Team Updates',
    'Summarize team updates from multiple emails',
    'Analyze these team update emails and provide a consolidated weekly summary:\n\n{{messages}}\n\nPlease organize by:\n1. Key Achievements\n2. Blockers & Issues\n3. Upcoming Priorities\n4. Action Items',
    'bulk_analysis',
    strftime('%s', 'now'),
    1
);
```

**Modify Existing Prompts:**
```sql
-- Update prompt text
UPDATE prompt_templates 
SET prompt_text = 'New improved prompt text with {{variables}}'
WHERE name = 'Quick Summary';

-- Change category
UPDATE prompt_templates 
SET category = 'custom' 
WHERE id = 5;

-- Mark as favorite
UPDATE prompt_templates 
SET is_favorite = 1 
WHERE name = 'Email Classifier';

-- Update description
UPDATE prompt_templates 
SET description = 'Updated description' 
WHERE id = 3;
```

**Delete Prompts:**
```sql
-- Delete by name
DELETE FROM prompt_templates WHERE name = 'Unwanted Prompt';

-- Delete by ID
DELETE FROM prompt_templates WHERE id = 10;

-- Delete all prompts in a category
DELETE FROM prompt_templates WHERE category = 'old_category';

-- Clear usage statistics (reset counters)
UPDATE prompt_templates SET usage_count = 0;
```

**Backup and Restore:**
```bash
# Backup prompts to SQL file
sqlite3 ~/.config/giztui/gmail-tui-{your-email}.db \
  ".dump prompt_templates" > prompts_backup.sql

# Restore from backup
sqlite3 ~/.config/giztui/gmail-tui-{your-email}.db \
  ".read prompts_backup.sql"

# Export prompts to CSV
sqlite3 -header -csv ~/.config/giztui/gmail-tui-{your-email}.db \
  "SELECT * FROM prompt_templates;" > prompts.csv
```

**Database Schema:**
```sql
-- View the prompt_templates table structure
.schema prompt_templates

-- Example output:
-- CREATE TABLE prompt_templates (
--     id INTEGER PRIMARY KEY AUTOINCREMENT,
--     name TEXT NOT NULL,
--     description TEXT,
--     prompt_text TEXT NOT NULL,
--     category TEXT DEFAULT 'general',
--     created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
--     is_favorite BOOLEAN DEFAULT 0,
--     usage_count INTEGER DEFAULT 0
-- );
```

**Tips:**
- Always restart Gmail TUI after modifying prompts directly in the database
- Use single quotes for SQL strings to avoid escaping issues
- Test new prompts with short emails first
- Keep prompt names unique for easier management
- Use meaningful categories to organize your prompts
- For bulk prompts, use `{{messages}}` (recommended) or `{{body}}` for combined message content
- For single prompts, use `{{from}}`, `{{subject}}`, `{{date}}`, `{{body}}` for message details
- Bulk prompts (category `bulk_analysis`) only appear in bulk mode picker
- Regular prompts are filtered out from bulk mode picker

CLI flags override config (subset): `--llm-provider`, `--llm-model`, `--llm-region`, `--ollama-endpoint`, `--ollama-model`, `--ollama-timeout`.
Logging: set `"log_file"` in `config.json` to direct logs to a custom path; defaults to `~/.config/giztui/giztui.log`.

### Internals

- Deterministic formatter lives in `internal/render/format.go`
- TUI integration in `internal/tui/markdown.go`
- LLM providers in `internal/llm/` (Ollama and Bedrock). Provider is chosen from config/flags.

## 🗺️ Project Status & Roadmap

- For up-to-date feature status and planned work, see `TODO.md`.

## ⚠️ Known Issues

### UI and Focus Issues
- **`:slack` command focus** - When using the `:slack` command, focus doesn't automatically go to the Slack forwarding widget. Use the `K` key instead for proper focus behavior.
- **Advanced search UI** - The advanced search scope selection has visual issues where the page doesn't update cleanly, leaving orphan letters when navigating up and down through options.

### Pending Features  
- **Slack template comments** - The `{{comment}}` variable is not yet available in Slack summary prompt templates. User messages are displayed separately above the summary.
- **ErrorHandler migration** - Some operations still need to be migrated to use the centralized ErrorHandler for consistent user feedback.

## 📝 Obsidian Integration

Gmail TUI includes a powerful Obsidian integration that allows you to ingest emails directly to your second brain system.

### Features

- **Single configurable template** - One template for all emails with variable substitution
- **Personal comments** - **🆕 Add personal notes about emails before ingestion**
- **Duplicate prevention** - SQLite-based history tracking prevents re-ingestion
- **Attachment support** - Include email attachments by default
- **Clean interface** - Side panel (not modal) for template preview and comment input
- **Organized structure** - All emails go to `00-Inbox` folder for second brain processing
- **Immediate feedback** - Panel closes instantly, operation runs asynchronously
- **Keyboard navigation** - Tab between template view and comment field

### Configuration

Add this section to your `~/.config/giztui/config.json`:

```json
{
  "obsidian": {
    "enabled": true,
    "vault_path": "~/Documents/Obsidian/MyVault",
    "ingest_folder": "00-Inbox",
    "filename_format": "{{date}}_{{subject_slug}}_{{from_domain}}",
    "history_enabled": true,
    "prevent_duplicates": true,
    "max_file_size": 1048576,
    "include_attachments": true,
    "template": "templates/obsidian/email.md"
  }
}
```

**Template Configuration:**
- **File path**: Use `"template": "templates/obsidian/email.md"` to load from `~/.config/giztui/templates/obsidian/email.md`
- **Inline template**: Use `"template": "---\ntitle: \"{{subject}}\"...` for inline Markdown template
- **Path resolution**: Relative paths are resolved relative to `~/.config/giztui/`

### Usage

#### Single Email Ingestion
1. **Select an email** in the message list
2. **Press `Shift+O`** to open the "Send to Obsidian" panel
3. **Review the template** that will be used (displayed at the top)
4. **Optional**: Add a personal comment in the "Pre-message:" field
5. **Press `Enter`** to ingest the email to Obsidian
6. **Press `Esc`** to cancel at any time
7. **Use `Tab`** to navigate between template view and comment field

#### Bulk Email Ingestion 🆕
1. **Enter bulk mode** by pressing `v`, `b`, or `space` on an email
2. **Select multiple emails** using `space` or `*` (select all)
3. **Press `Shift+O`** to open the bulk Obsidian panel
4. **Review the bulk template** information
5. **Optional**: Add a shared comment for all selected emails in the "Bulk comment:" field
6. **Press `Enter`** to ingest all selected emails with the shared comment
7. **Press `Esc`** to cancel at any time

**Note**: The panel closes immediately when you press Enter, and the ingestion runs asynchronously. You'll see progress and success messages in the status bar showing the processing status for each email.

### Template Variables

- `{{subject}}` - Email subject
- `{{from}}` - Sender email
- `{{to}}` - Recipient email
- `{{cc}}` - CC recipients
- `{{body}}` - Email content
- `{{date}}` - Email date
- `{{labels}}` - Gmail labels
- `{{message_id}}` - Gmail message ID
- `{{ingest_date}}` - Date of ingestion

### Configuration Options

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `enabled` | boolean | Enable/disable Obsidian integration | `true` |
| `vault_path` | string | Path to your Obsidian vault | `~/Documents/Obsidian/MyVault` |
| `ingest_folder` | string | Folder where emails are saved | `00-Inbox` |
| `filename_format` | string | Format for generated filenames | `{{date}}_{{subject_slug}}_{{from_domain}}` |
| `history_enabled` | boolean | Track ingestion history | `true` |
| `prevent_duplicates` | boolean | Prevent duplicate ingestions | `true` |
| `max_file_size` | integer | Maximum file size in bytes | `1048576` (1MB) |
| `include_attachments` | boolean | Include email attachments | `true` |
| `template` | string | Markdown template for emails | See example above |

### Customizing Templates

You can customize the template for different types of emails. Here are some examples:

**Meeting Template:**
```markdown
---
title: "{{subject}}"
date: {{date}}
from: {{from}}
type: meeting
status: inbox
tags: [meeting, action-items]
---

# {{subject}}

**Meeting Details:**
- **From:** {{from}}
- **Date:** {{date}}
- **Type:** Meeting/Follow-up

{% if comment %}**Personal Note:** {{comment}}

{% endif %}**Action Items:**
- [ ] 

**Notes:**
{{body}}

**Next Meeting:**
- [ ] Schedule follow-up

---

*Ingested from Gmail on {{ingest_date}}*`
```

**Project Template:**
```markdown
---
title: "{{subject}}"
date: {{date}}
from: {{from}}
type: project
status: inbox
tags: [project, update]
---

# {{subject}}

**Project Details:**
- **From:** {{from}}
- **Date:** {{date}}
- **Project:** 

**Key Updates:**
- 

**Next Steps:**
- [ ] 

**Content:**
{{body}}

---

*Ingested from Gmail on {{ingest_date}}*`
```

### Troubleshooting

**Common Issues:**
- **"Vault path not found"** - Verify the `vault_path` exists and is accessible
- **"Permission denied"** - Check write permissions on the vault directory
- **Emails not ingesting** - Ensure `enabled` is `true` and restart the app
- **Duplicate prevention** - Check the `obsidian_forward_history` table in SQLite

**Database Location:**
The ingestion history is stored in the same SQLite database as other features:
`~/.config/giztui/cache/{account_email}.sqlite3`

**Table Structure:**
```sql
CREATE TABLE obsidian_forward_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    account_email TEXT NOT NULL,
    obsidian_path TEXT NOT NULL,
    template_used TEXT,
    forward_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'success',
    error_message TEXT,
    file_size INTEGER,
    metadata TEXT
);
```