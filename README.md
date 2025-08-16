# 📨 Gmail TUI - Gmail Client with Local AI

A **TUI (Text-based User Interface)** Gmail client developed in **Go** that uses the **Gmail API** via OAuth2 and features **local AI integration** through Ollama.

## ✨ Features

### 📬 Core Gmail Functionality
- ✅ View inbox and labels
- ✅ Read emails
- ✅ Mark as read/unread
- ✅ Archive and move to trash
- ✅ Manage labels (add, remove, create)
- ✅ Load more messages (when list is focused)
- ✅ Search and navigation support with VIM-style commands (`:5`, `G`, `gg`)
- 🚧 WIP: Compose, Reply, Drafts, Attachments

### 🧠 AI Features with LLM (Ollama & Bedrock)
- ✅ **Summarize emails** - Generate concise email summaries
- ✅ **AI summaries local cache (SQLite)** - Reuse previously generated summaries across sessions
- ✅ **Streaming summaries (Ollama)** - Incremental tokens render live in the summary pane
- ✅ **Streaming cancellation** - Press Esc to instantly cancel any streaming operation
- ✅ **Recommend labels** - Suggest appropriate labels for emails
- ✅ **Configurable prompts** - All prompts are customizable
- 🧪 **Generate replies** - Experimental (placeholder implementation)

### 🚀 **Prompt Library System** 🆕
- ✅ **Custom prompt templates** - Predefined prompts for different use cases
- ✅ **Variable substitution** - Auto-complete `{{from}}`, `{{subject}}`, `{{body}}`, `{{date}}`
- ✅ **Streaming LLM responses** - Real-time token streaming for prompt results
- ✅ **Interruptible streaming** - Cancel any prompt operation instantly with Esc
- ✅ **Smart caching** - Cache prompt results to avoid re-processing
- ✅ **Split-view interface** - Prompt picker appears like labels (not full-screen modal)
- ✅ **Category organization** - Organize prompts by purpose (Summary, Analysis, Action Items, etc.)
- ✅ **Usage tracking** - Monitor which prompts are used most frequently

### 🔥 **Bulk Prompts** 🆕
- ✅ **Multi-email analysis** - Apply prompts to multiple emails simultaneously
- ✅ **Consolidated insights** - Get unified analysis across multiple messages
- ✅ **Cloud product tracking** - Specialized prompts for AWS/Azure/GCP updates
- ✅ **Project monitoring** - Consolidate project status from multiple emails
- ✅ **Trend analysis** - Identify patterns across multiple sources
- ✅ **Efficient processing** - Async processing with progress indicators
- ✅ **Responsive controls** - Cancel bulk operations instantly with Esc

### 📝 **Obsidian Integration** 🆕
- ✅ **Email ingestion** - Send emails directly to Obsidian as Markdown notes
- ✅ **Second brain system** - Organize emails in `00-Inbox` folder
- ✅ **Configurable template** - Single, customizable Markdown template
- ✅ **Variable substitution** - Auto-complete `{{subject}}`, `{{body}}`, `{{from}}`, etc.
- ✅ **Duplicate prevention** - SQLite-based history tracking
- ✅ **Attachment support** - Include email attachments in notes
- ✅ **Keyboard shortcut** - `Shift+O` for quick ingestion
- ✅ **Panel interface** - Clean side panel (not modal) for template preview

### 📱 Adaptive Layout System
- ✅ **Responsive design** - Automatically adapts to terminal size
- ✅ **Multiple layout modes** - Wide, medium, narrow, and mobile layouts
- ✅ **Real-time resizing** - Layout changes as you resize your terminal
- ✅ **Fullscreen mode** - Press 'f' for fullscreen text view
- ✅ **Focus switching** - Press 't' to toggle between list and text focus

### 🎯 User Experience
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
gmail-tui/
├── cmd/gmail-tui/          # Main application entry point
│   └── main.go            # Application entry point
├── internal/               # Private application code
│   ├── cache/             # SQLite caching system
│   │   └── store.go       # Cache store implementation
│   ├── calendar/          # Google Calendar integration
│   │   └── client.go      # Calendar API client
│   ├── config/            # Configuration management & theming
│   │   ├── config.go      # Configuration loading & validation
│   │   ├── colors.go      # Color scheme management
│   │   ├── theme.go       # Theme system
│   │   └── manager.go     # Configuration manager
│   ├── db/                # 🆕 Database layer
│   │   ├── store.go       # Main database store
│   │   ├── cache_store.go # AI summary caching
│   │   └── prompt_store.go # 🆕 Prompt library storage
│   ├── gmail/             # Gmail API client wrapper
│   │   └── client.go      # Gmail API client
│   ├── llm/               # Multi-provider LLM support
│   │   ├── factory.go     # LLM provider factory
│   │   ├── ollama.go      # Ollama provider
│   │   └── bedrock.go     # Amazon Bedrock provider
│   ├── prompts/           # 🆕 Prompt system
│   │   └── types.go       # Prompt data types
│   ├── render/            # Email rendering & formatting
│   │   ├── email.go       # Email renderer
│   │   └── format.go      # Formatting utilities
│   ├── services/          # 🆕 Business logic service layer
│   │   ├── interfaces.go  # Service contracts
│   │   ├── email_service.go    # Email operations
│   │   ├── ai_service.go       # AI/LLM operations
│   │   ├── label_service.go    # Label management
│   │   ├── cache_service.go    # Cache operations
│   │   ├── prompt_service.go   # 🆕 Prompt library management
│   │   └── repository.go       # Data access layer
│   └── tui/               # Terminal User Interface
│       ├── app.go         # Main application with service integration
│       ├── error_handler.go   # 🆕 Centralized error handling
│       ├── layout.go      # UI layout management
│       ├── keys.go        # Keyboard shortcuts & input handling
│       ├── messages.go    # Message list & content display
│       ├── messages_actions.go # Message actions (archive, trash, etc.)
│       ├── messages_bulk.go   # Bulk message operations
│       ├── labels.go      # Label management UI
│       ├── ai.go          # AI summary & LLM features
│       ├── prompts.go     # 🆕 Prompt library UI
│       ├── markdown.go    # Markdown rendering & LLM touch-up
│       ├── commands.go    # Command bar & execution
│       ├── status.go      # Status bar & notifications
│       ├── welcome.go     # Welcome screen
│       ├── logging.go     # Logging setup
│       └── list_helpers.go # List manipulation utilities
├── pkg/                   # Reusable packages
│   └── auth/              # OAuth2 authentication
│       └── oauth.go       # OAuth2 implementation
├── docs/                  # Documentation
│   ├── ARCHITECTURE.md    # Architecture documentation
│   ├── COLORS.md          # Color system documentation
│   ├── gmail-filters-and-search-operators.md # Search operators
│   └── search_ux_and_roadmap.md # Search UX roadmap
├── scripts/               # Build & development scripts
│   └── check-architecture.sh # Architecture compliance checker
├── skins/                 # Theme skins
│   ├── gmail-dark.yaml    # Dark theme
│   ├── gmail-light.yaml   # Light theme
│   └── custom-example.yaml # Custom theme example
├── examples/              # Usage examples
│   ├── config.json        # Configuration example
│   ├── credentials.json.example # Credentials template
│   └── theme_demo.go      # Theme demonstration
├── .github/               # GitHub workflows
│   └── workflows/         # CI/CD workflows
│       └── ci.yml         # Continuous integration
├── build/                 # Build artifacts
├── .claude/               # Claude AI configuration
├── .cursor/               # Cursor IDE configuration
├── Makefile               # Build & development tasks
├── go.mod                 # Go module definition
├── go.sum                 # Go dependencies checksums
├── .pre-commit-config.yaml # Pre-commit hooks
├── .golangci.yml          # Go linter configuration
├── CLAUDE.md              # Claude AI development notes
├── TODO.md                # Development roadmap & tasks
└── README.md              # This file
```

### 🔧 Service Architecture

The application now follows a **layered architecture** with clear separation between UI, business logic, and data access:

#### 📊 **Service Layer** (`internal/services/`)
- **EmailService**: High-level email operations (compose, send, archive, etc.)
- **AIService**: LLM integration with caching and streaming support  
- **LabelService**: Gmail label management operations
- **CacheService**: SQLite-based caching for AI summaries
- **PromptService**: 🆕 Prompt library management with caching and usage tracking
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
   emailService, aiService, labelService, cacheService, repository := app.GetServices()
   ```

#### 🛡️ **Benefits**
- **Better Testability** - Services can be easily mocked and unit tested
- **Cleaner Code** - UI components focus on presentation, not business logic
- **Thread Safety** - Proper mutex protection for concurrent operations
- **Consistent UX** - Centralized error handling provides uniform user feedback
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
~/.config/gmail-tui/
├── config.json      # Application configuration
├── credentials.json # Gmail API credentials (OAuth2)
└── token.json      # OAuth2 token cache
```

### Setup Steps:

1. **Create the configuration directory:**
   ```bash
   mkdir -p ~/.config/gmail-tui
   ```

2. **Download Gmail API credentials:**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable the Gmail API
   - Create OAuth2 credentials (Desktop application)
   - Download the JSON file and save it as `~/.config/gmail-tui/credentials.json`
   - See `examples/credentials.json.example` for the expected format

3. **Copy the example configuration:**
   ```bash
   cp examples/config.json ~/.config/gmail-tui/config.json
   ```

4. **Optional: Configure Ollama for AI features:**
   - Install [Ollama](https://ollama.ai/)
   - Start Ollama service
   - Update the configuration with your preferred model

## 🎮 Usage

### Basic commands

```bash
# Run with default configuration
./gmail-tui

# Specify custom credentials
./gmail-tui --credentials ~/path/to/credentials.json

# Configure LLM (Ollama example)
./gmail-tui --config ~/.config/gmail-tui/config.json

# Use custom configuration file
./gmail-tui --config ~/custom-config.json
```

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `Enter` | View selected message |
| `r` | Refresh (in drafts mode, reload drafts) |
| `n` | Load more when list is focused; otherwise compose new (WIP) |
| `R` | Reply (WIP) |
| `s` | Search |
| `/` | Local filter |
| `F` | Quick search: from current sender |
| `T` | Quick search: to current sender (includes Sent) |
| `S` | Quick search: by current subject |
| `:` | Open command bar (k9s-style) |
| `u` | Show unread |
| `t` | Toggle read/unread |
| `d` | Move to trash |
| `a` | Archive |
| `D` | View drafts (experimental) |
| `A` | View attachments (WIP) |
| `w` | Save current message to file (.txt, rendered) |
| `W` | Save current message as raw .eml (server format) |
| `l` | Manage labels (contextual panel) |
| `m` | Move message (choose label) |
| `M` | Toggle Markdown rendering |
| `y` | Toggle AI summary |
| `Y` | Regenerate AI summary (force refresh; ignores cache) |
| `g` | Generate reply (experimental) |
| `p` | Open prompt picker (single message) or bulk prompt picker (bulk mode) |
| `O` | 🆕 **Ingest email to Obsidian** |
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

**Examples:**
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
| `m` | Move selected messages to label |
| `p` | Apply AI prompt to all selected messages |
| `Esc` | Exit bulk mode |

**Bulk Mode Status Bar:**
- Shows current selection count
- Displays available actions: `space/v=select, *=all, a=archive, d=trash, m=move, p=prompt, ESC=exit`

### AI Features (LLM)

| Key | Action |
|-----|--------|
| `y` | Summarize message |
| `Y` | Regenerate AI summary (force refresh) |
| `g` | Generate reply (experimental) |
| `o` | Suggest label |
| `P` | 🆕 **Open Prompt Library** |

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

#### LLM Configuration (providers)

You can use a pluggable LLM provider. Configure in `~/.config/gmail-tui/config.json`:

```json
{
  "llm_enabled": true,
  "llm_provider": "ollama",          // ollama|bedrock (supported now)
  "llm_model": "llama3.1:8b",
  "llm_endpoint": "http://localhost:11434/api/generate",
  "llm_api_key": "",
  "llm_timeout": "20s",
  "llm_stream_enabled": true
}
```

Ollama specific legacy fields (`ollama_endpoint`, `ollama_model`, `ollama_timeout`) are still supported for backward compatibility.

##### Amazon Bedrock (on-demand)

To use Amazon Bedrock instead of Ollama:

```json
{
  "llm_enabled": true,
  "llm_provider": "bedrock",
  "llm_model": "us.anthropic.claude-3-5-haiku-20241022-v1:0", // on-demand ID, include region/vendor and revision :0
  "llm_region": "us-east-1",
  "llm_timeout": "20s"
}
```

Notes for Bedrock on-demand:

- Always include the revision suffix `:0` in `llm_model` for on-demand model IDs.
- In many accounts, you must include the regional vendor prefix, e.g. `us.anthropic...` rather than just `anthropic...`.
- Make sure your AWS credentials are set (e.g., `AWS_PROFILE=your-profile`) and the region has access to the model.
- Alternatively, you can provide an inference profile ARN in `llm_model` and it will be sent via `ModelId`.

Run with a custom config:

```bash
AWS_PROFILE=your-profile ./gmail-tui --config ~/.config/gmail-tui/config.bedrock.json
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

Prompts for AI features are configurable via `~/.config/gmail-tui/config.json`.

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

- **Default location**: `~/.config/gmail-tui/cache/{account_email}.sqlite3`
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

Supported commands: `labels`, `search`, `inbox`, `compose`, `help`, `quit`, `cache`

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

## 🎨 Themes and Color System

- Skins are stored in `skins/` (`gmail-dark.yaml`, `gmail-light.yaml`, `custom-example.yaml`).
- See the detailed documentation in `docs/COLORS.md`.
- Email list rendering colors are driven by message state (unread, important, sent, draft) and are configurable.

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

Config fields (in `~/.config/gmail-tui/config.json`):

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
- Location: `~/.config/gmail-tui/gmail-tui-{account}.db`

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
sqlite3 ~/.config/gmail-tui/gmail-tui-{your-email}.db

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
sqlite3 ~/.config/gmail-tui/gmail-tui-{your-email}.db
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
sqlite3 ~/.config/gmail-tui/gmail-tui-{your-email}.db \
  ".dump prompt_templates" > prompts_backup.sql

# Restore from backup
sqlite3 ~/.config/gmail-tui/gmail-tui-{your-email}.db \
  ".read prompts_backup.sql"

# Export prompts to CSV
sqlite3 -header -csv ~/.config/gmail-tui/gmail-tui-{your-email}.db \
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
Logging: set `"log_file"` in `config.json` to direct logs to a custom path; defaults to `~/.config/gmail-tui/gmail-tui.log`.

### Internals

- Deterministic formatter lives in `internal/render/format.go`
- TUI integration in `internal/tui/markdown.go`
- LLM providers in `internal/llm/` (Ollama and Bedrock). Provider is chosen from config/flags.

## 🗺️ Project Status & Roadmap

- For up-to-date feature status and planned work, see `TODO.md`.

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

Add this section to your `~/.config/gmail-tui/config.json`:

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
    "template": "---\ntitle: \"{{subject}}\"\ndate: {{date}}\nfrom: {{from}}\ntype: email\nstatus: inbox\nlabels: {{labels}}\nmessage_id: {{message_id}}\n---\n\n# {{subject}}\n\n**From:** {{from}}  \n**Date:** {{date}}  \n**Labels:** {{labels}}\n\n{% if comment %}**Personal Note:** {{comment}}\n\n{% endif %}---\n\n{{body}}\n\n---\n\n*Ingested from Gmail on {{ingest_date}}*"
  }
}
```

### Usage

1. **Select an email** in the message list
2. **Press `Shift+O`** to open the "Send to Obsidian" panel
3. **Review the template** that will be used (displayed at the top)
4. **Optional**: Add a personal comment in the "Pre-message:" field
5. **Press `Enter`** to ingest the email to Obsidian
6. **Press `Esc`** to cancel at any time
7. **Use `Tab`** to navigate between template view and comment field

**Note**: The panel closes immediately when you press Enter, and the ingestion runs asynchronously. You'll see progress and success messages in the status bar.

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
- `{{comment}}` - **🆕 Personal comment added by user**

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
`~/.config/gmail-tui/cache/{account_email}.sqlite3`

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