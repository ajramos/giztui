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

### 🧠 AI Features with Local LLM
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

#### LLM Configuration

You can use a pluggable LLM provider (Ollama by default). Configure in `~/.config/gmail-tui/config.json`:

```json
{
  "llm_enabled": true,
  "llm_provider": "ollama",          // ollama|openai|anthropic|custom (ollama supported now)
  "llm_model": "llama3.1:8b",
  "llm_endpoint": "http://localhost:11434/api/generate",
  "llm_api_key": "",
  "llm_timeout": "20s"
}
```

Ollama specific legacy fields (`ollama_endpoint`, `ollama_model`, `ollama_timeout`) are still supported for backward compatibility.

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
  - `ESC`: close panel
- Browse all labels:
  - From the panel select “🔎 Browse all labels…” to expand a full picker with search
  - Type to filter; `Enter` toggles; `ESC` returns to the quick panel
- Focus rules:
  - Tab cycles: text → labels → summary → list
  - Arrow keys act on the currently focused pane only
  - The app does not steal focus while background work runs

### 📦 Move Message (Contextual)

- Press `m` to open the side panel directly in "Browse all labels" mode
- Type to filter labels, `Enter` on a label will:
  - Apply the label (if not already applied)
  - Archive the message (move semantics)
  - Update the list and content in place
  - Close the panel automatically
- `ESC` closes the panel (no intermediate quick view)

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