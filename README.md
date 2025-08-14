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
- ✅ Basic search and navigation support
- 🚧 WIP: Compose, Reply, Drafts, Attachments

### 🧠 AI Features with LLM (Ollama & Bedrock)
- ✅ **Summarize emails** - Generate concise email summaries
- ✅ **AI summaries local cache (SQLite)** - Reuse previously generated summaries across sessions
- ✅ **Streaming summaries (Ollama)** - Incremental tokens render live in the summary pane
- ✅ **Recommend labels** - Suggest appropriate labels for emails
- ✅ **Configurable prompts** - All prompts are customizable
- 🧪 **Generate replies** - Experimental (placeholder implementation)

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
├── internal/               # Private application code
│   ├── cache/             # SQLite caching system
│   ├── calendar/          # Google Calendar integration
│   ├── config/            # Configuration management & theming
│   ├── gmail/             # Gmail API client wrapper
│   ├── llm/               # Multi-provider LLM support (Ollama, Bedrock)
│   ├── render/            # Email rendering & formatting
│   ├── services/          # 🆕 Business logic service layer
│   │   ├── interfaces.go  # Service contracts
│   │   ├── email_service.go    # Email operations
│   │   ├── ai_service.go       # AI/LLM operations
│   │   ├── label_service.go    # Label management
│   │   ├── cache_service.go    # Cache operations
│   │   └── repository.go       # Data access layer
│   └── tui/               # Terminal User Interface
│       ├── app.go         # Main application with service integration
│       ├── error_handler.go   # 🆕 Centralized error handling
│       └── ...            # UI components & views
├── pkg/                   # Reusable packages
│   ├── auth/              # OAuth2 authentication
│   └── utils/             # General utilities
├── docs/                  # Documentation
├── examples/              # Usage examples
└── README.md
```

### 🔧 Service Architecture

The application now follows a **layered architecture** with clear separation between UI, business logic, and data access:

#### 📊 **Service Layer** (`internal/services/`)
- **EmailService**: High-level email operations (compose, send, archive, etc.)
- **AIService**: LLM integration with caching and streaming support  
- **LabelService**: Gmail label management operations
- **CacheService**: SQLite-based caching for AI summaries
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
| `o` | Suggest label |
| `q` | Quit |

### AI Features (LLM)

| Key | Action |
|-----|--------|
| `y` | Summarize message |
| `g` | Generate reply (experimental) |
| `o` | Suggest label |

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

The app uses an embedded SQLite database (no external server) to cache AI summaries:

- Default location: `~/.config/gmail-tui/cache/gmail-<account_email>.sqlite3`
- Per-account separation by filename
- PRAGMAs tuned for TUI (WAL, foreign keys, timeouts)

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

Supported commands: `labels`, `search`, `inbox`, `compose`, `help`, `quit`

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

```
{
  "LLMEnabled": true,
  "LLMProvider": "ollama",        // or "bedrock"
  "LLMEndpoint": "http://localhost:11434/api/generate", // Ollama
  "LLMRegion": "us-east-1",      // Bedrock
  "LLMModel": "llama3.2:latest",
  "LLMTimeout": "20s"
}
```

CLI flags override config (subset): `--llm-provider`, `--llm-model`, `--llm-region`, `--ollama-endpoint`, `--ollama-model`, `--ollama-timeout`.
Logging: set `"log_file"` in `config.json` to direct logs to a custom path; defaults to `~/.config/gmail-tui/gmail-tui.log`.

### Internals

- Deterministic formatter lives in `internal/render/format.go`
- TUI integration in `internal/tui/markdown.go`
- LLM providers in `internal/llm/` (Ollama and Bedrock). Provider is chosen from config/flags.

## 🗺️ Project Status & Roadmap

- For up-to-date feature status and planned work, see `TODO.md`.