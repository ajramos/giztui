# 📨 Gmail TUI - Gmail Client with Local AI

A **TUI (Text-based User Interface)** Gmail client developed in **Go** that uses the **Gmail API** via OAuth2 and features **local AI integration** through Ollama.

## ✨ Features

### 📬 Core Gmail Functionality
- ✅ View inbox, drafts, and labels
- ✅ Read, reply, compose, and archive emails
- ✅ Mark as read/unread
- ✅ Delete and move to trash
- ✅ View attachments
- ✅ Manage labels (add, remove, create)
- ✅ Refresh mailboxes
- ✅ Basic search and navigation support

### 🧠 AI Features with Local LLM
- ✅ **Summarize emails** - Generate concise email summaries
- ✅ **Generate replies** - Create professional automatic responses
- ✅ **Recommend labels** - Suggest appropriate labels for emails
- ✅ **Configurable prompts** - All prompts are customizable

### 📱 Adaptive Layout System
- ✅ **Responsive design** - Automatically adapts to terminal size
- ✅ **Multiple layout modes** - Wide, medium, narrow, and mobile layouts
- ✅ **Real-time resizing** - Layout changes as you resize your terminal
- ✅ **Layout information** - Press 'l' to see current layout details
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

# Configure Ollama
./gmail-tui --ollama-endpoint http://localhost:11434/api/generate --ollama-model llama3

# Use custom configuration file
./gmail-tui --config ~/custom-config.json
```

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `Enter` | View selected message |
| `r` | Reply |
| `n` | Compose new message |
| `R` | Refresh messages |
| `s` | Search |
| `u` | Show unread |
| `t` | Toggle read/unread |
| `d` | Move to trash |
| `a` | Archive |
| `D` | View drafts |
| `A` | View attachments |
| `l` | Manage labels |
| `q` | Quit |

### AI Features (requires Ollama)

| Key | Action |
|-----|--------|
| `y` | Summarize message |
| `g` | Generate reply |
| `o` | Suggest label |

### Layout Controls

| Key | Action |
|-----|--------|
| `l` | Show layout information |
| `f` | Toggle fullscreen text view |
| `t` | Toggle focus between list and text |

### 🧭 Command Mode (k9s-style)

- Press `:` to open the command bar
- The command bar now has a border, takes focus automatically, and supports suggestions
- Autocompletion: type partial commands and press `Tab` to complete (e.g., `:la` → `labels`)
- Suggestions are shown in brackets while typing

Supported commands: `labels`, `search`, `inbox`, `compose`, `help`, `quit`

### 🏷️ Labels Management (Contextual)

- Press `l` on a selected message to open the contextual labels view
- Shows only actionable labels; names are human-friendly (e.g., "Social DoiT")
- Status:
  - `✅` applied to message
  - `○` not applied
- Toggle labels with `Enter` (non-blocking, UI updates instantly)
- `n`: create label, `r`: refresh, `ESC`: back
- Special handling: only `STARRED` is shown; colored star variants are hidden

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
- **📊 Layout info**: Press 'l' to see current layout and screen information
- **🔍 Fullscreen mode**: Press 'f' to view text content in fullscreen
- **🎯 Smart focus**: Press 't' to switch focus between list and text areas
- **⚡ Performance**: Optimized rendering for each layout type