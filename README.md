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

```
gmail-tui/
├── cmd/gmail-tui/          # Main application entry point
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── gmail/             # Gmail API client
│   ├── llm/               # Ollama client
│   └── tui/               # User interface
├── pkg/                   # Reusable packages
│   ├── auth/              # OAuth2 authentication
│   └── utils/             # General utilities
├── docs/                  # Documentation
├── examples/              # Usage examples
└── README.md
```

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
| `u` | Show unread |
| `t` | Toggle read/unread |
| `d` | Move to trash |
| `a` | Archive |
| `D` | View drafts (experimental) |
| `A` | View attachments (WIP) |
| `l` | Manage labels (contextual panel) |
| `m` | Move message (choose label) |
| `M` | Toggle Markdown rendering |
| `y` | Toggle AI summary |
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
  "llm_timeout": "20s"
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
  "summarize_prompt": "Resume brevemente el siguiente correo electrónico:\n\n{{body}}\n\nDevuelve el resumen en español en un párrafo.",
  "label_prompt": "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: {{labels}}\n\nEmail:\n{{body}}"
}
```

Notes:
- If a prompt is empty or missing, a sensible default will be used.
- Changes to `config.json` are picked up on application start. Please restart the app after editing the configuration.

### Layout Controls

| Key | Action |
|-----|--------|
| `f` | Toggle fullscreen text view |
| `t` | Toggle focus between list and text |

### 🧭 Command Mode (k9s-style)

- Press `:` to open the command bar
- The command bar now has a border, takes focus automatically, and supports suggestions
- Autocompletion: type partial commands and press `Tab` to complete (e.g., `:la` → `labels`)
- Suggestions are shown in brackets while typing

Supported commands: `labels`, `search`, `inbox`, `compose`, `help`, `quit`

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
  - Applies an in-memory filter over the current inbox page.
  - Actions while filtered (archive, trash, move, label) propagate to the underlying inbox snapshot.
  - Press `ESC` to exit the filter instantly without network calls, restoring the original list with your changes reflected.
  - After removing an item (archive/trash/move), the selection stays on the same visual position.

- Remote search (`s`):
  - Runs a Gmail query and shows a new result list.
  - Press `ESC` to return to the inbox; the inbox is reloaded from the server (source of truth).

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

## 🗺️ Project Status & Roadmap

- For up-to-date feature status and planned work, see `TODO.md`.