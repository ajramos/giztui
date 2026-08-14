# 📨 GizTUI - AI-Powered Gmail Terminal Client

A powerful **terminal Gmail client** built in **Go** that brings **AI intelligence** to your email workflow. Features local AI integration, advanced productivity tools, and seamless integrations with Slack, Obsidian, and more.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.25.13+-blue.svg)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)
![Release](https://img.shields.io/github/v/release/ajramos/giztui)

## ✨ Key Features

### 📬 **Complete Gmail Management**
- Full email operations: compose, reply, forward, archive, search, and label management
- Advanced threading with conversation grouping and AI summaries
- VIM-style navigation and bulk operations (`d3d` to delete 3, `a5a` to archive 5)
- Powerful Gmail search with filters, date ranges, and size-based queries

### 🧠 **AI-Powered Intelligence** 
- **Email summarization** with streaming support (Ollama & Amazon Bedrock)
- **Smart label suggestions** based on email content
- **Custom prompt library** with variable substitution and bulk analysis
- **Local caching** to avoid re-processing with SQLite storage

### ⚡ Deterministic rules (no AI)

- `:rules` — manage query → action rules (archive, trash, label, mark read, prompt) stored locally
- `:rules plan` — preview what your rules match as an Action Plan, instantly, no AI involved
- With `inbox_analyzer.deterministic_prefilter` enabled (default), the AI Inbox Action Plan resolves rule matches first (marked ⚡) and only sends the remaining emails to the LLM
- `:rules sync <n>` mirrors a rule as a real server-side Gmail filter (marked ☁); `:rules unsync <n>` removes the mirror

> **Note:** mirroring rules to Gmail needs the `gmail.settings.basic` OAuth scope, added in this version. Existing installs must re-authenticate once (delete the token file, e.g. `~/.config/giztui/token.json`, and restart) before `:rules sync` works. Everything else keeps working with the old token.

### 🔌 **Seamless Integrations**
- **Slack forwarding** - Send emails to configured channels with AI summaries
- **Obsidian ingestion** - Transform emails into structured markdown notes (individual files or combined repopack)
- **Calendar integration** - RSVP to meeting invitations directly from emails
- **Link & attachment management** - Quick access to URLs and file downloads

### 🎨 **Professional UI/UX**
- **Adaptive layout** that responds to terminal size changes
- **Custom themes** with runtime switching (Dracula, Slate Blue, Gmail Dark/Light)
- **100% keyboard navigation** with fully customizable shortcuts
- **Command system** with auto-completion and command parity

## 🚀 Quick Start

### Installation

**Download pre-built binaries** (recommended):
```bash
# Linux
curl -L https://github.com/ajramos/giztui/releases/latest/download/giztui-linux-amd64.tar.gz | tar -xz
sudo mv giztui /usr/local/bin/

# macOS
curl -L https://github.com/ajramos/giztui/releases/latest/download/giztui-darwin-amd64.tar.gz | tar -xz
mv giztui /usr/local/bin/

# Windows: Download giztui-windows-amd64.zip from releases page
```

**Or install with Go:**
```bash
go install github.com/ajramos/giztui/cmd/giztui@latest
```

### Desktop app (GUI)

There's also a **desktop client** (Wails) that shares the same service layer and
aims for keyboard-first parity with the TUI.

**macOS — Homebrew cask:**
```bash
brew tap ajramos/giztui
# Third-party taps must be trusted once before installing their casks:
brew trust ajramos/giztui
brew install --cask giztui-desktop
```

> The first `brew install` from a third-party tap will refuse to load the cask
> with *"Refusing to load cask … from untrusted tap"* until you run
> `brew trust ajramos/giztui` (or `brew trust --cask ajramos/giztui/giztui-desktop`).
> This is normal Homebrew behavior for taps outside `homebrew/core`.

**All platforms — direct download** (from the [latest release](https://github.com/ajramos/giztui/releases/latest)):
- macOS: `GizTUI-Desktop-<ver>-universal.dmg`
- Windows: `GizTUI-Desktop-<ver>-windows-amd64-setup.exe` (installer) or `…-windows-amd64.zip` (portable)
- Linux: `GizTUI-Desktop-<ver>-x86_64.AppImage` or `…-linux-amd64.tar.gz`

> The desktop builds are currently **unsigned**. macOS Gatekeeper / Windows
> SmartScreen will warn on first launch — on macOS open via right-click → Open
> (Homebrew installs already strip the quarantine attribute). See
> [docs/DESKTOP_DISTRIBUTION.md](docs/DESKTOP_DISTRIBUTION.md) for details and the
> signing/notarization plan. The desktop app reuses the same
> `~/.config/giztui/` credentials and token as the TUI.

### First Run

1. **Setup Gmail API credentials** ([detailed guide](docs/GETTING_STARTED.md#gmail-api-setup)):
   - **Enable Gmail API in Google Cloud Console** (required first step)
   - Create OAuth2 desktop credentials
   - Save as `~/.config/giztui/credentials.json`

2. **Run interactive setup:**
   ```bash
   giztui --setup
   ```

3. **Launch GizTUI:**
   ```bash
   giztui
   ```

### Enable AI Features (Optional)

**Local AI with Ollama:**
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model
ollama pull llama2

# Configure GizTUI
echo '{
  "llm": {
    "provider": "ollama",
    "ollama": {
      "model": "llama2"
    }
  }
}' > ~/.config/giztui/config.json
```

### Theme Configuration

GizTUI includes several built-in themes and supports custom themes:

**Built-in themes**: `slate-blue` (default), `gmail-dark`, `gmail-light`, `dracula`, `custom-example`

**Configure theme:**
```json
{
  "theme": {
    "current": "gmail-dark",
    "custom_dir": "/path/to/your/custom/themes"
  }
}
```

**Theme directory resolution** (priority order):
1. `custom_dir` - Your custom themes directory (if specified)  
2. `~/.config/giztui/themes/` - User themes directory
3. Built-in themes (embedded in binary)

**Runtime theme switching**: Press `H` to open theme picker with live preview.

> **⚠️ Important for `go install` users**: If themes don't work, ensure your config has the correct `theme.current` parameter (not `ui.theme`). See the [Configuration Guide](docs/CONFIGURATION.md#theme-settings) for details.

## 🎯 Essential Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `?` | Help | Show complete shortcuts |
| `s` | Search | Gmail search with auto-complete |
| `u` | Unread | Show unread messages |
| `a` | Archive | Archive current message |
| `d` | Trash | Move to trash |
| `c` | Compose | Create new email |
| `R` | Reply | Reply to email |
| `y` | AI Summary | Generate email summary |
| `p` | Prompts | Open AI prompt library |
| `K` | Slack | Forward to Slack |
| `Shift+O` | Obsidian | Ingest to Obsidian |
| `L` | Links | Quick link access |
| `A` | Attachments | Download attachments |
| `:` | Commands | Enter command mode |

**Bulk operations:** `v` to enter bulk mode, `space` to select, then use any action key.

**VIM-style ranges:** `a5a` archives 5 messages, `d3d` deletes 3, `t2t` toggles read on 2.

## 📊 What Makes GizTUI Different

### **🏗️ Architecture**
- **Service-oriented design** with clean separation of UI and business logic
- **Thread-safe operations** with proper error handling and recovery
- **Extensive testing** with unit tests, integration tests, and CI/CD pipeline

### **🔒 Privacy First**
- **No data leaves your machine** (except to Gmail and your configured integrations)
- **Local AI processing** with Ollama for complete privacy
- **Local SQLite caching** for performance without cloud dependency

### **⚡ Performance & Reliability** 
- **Efficient Gmail API usage** with smart caching and batch operations
- **Responsive UI** that handles large inboxes gracefully
- **Robust error handling** with user-friendly feedback and recovery options

### **🎮 Inspired by the Best**
- **k9s-style command interface** with auto-completion and shortcuts
- **VIM-like navigation** for power users who prefer keyboard efficiency
- **Modern terminal aesthetics** with themes and adaptive layouts

## 📚 Documentation

- **[📖 Getting Started](docs/GETTING_STARTED.md)** - Complete setup guide with troubleshooting
- **[✨ Features Overview](docs/FEATURES.md)** - Comprehensive feature documentation  
- **[⌨️ Keyboard Shortcuts](docs/KEYBOARD_SHORTCUTS.md)** - Complete shortcut reference
- **[⚙️ Configuration Guide](docs/CONFIGURATION.md)** - Detailed configuration options
- **[🖥️ Desktop Client](docs/DESKTOP.md)** - Wails desktop app architecture ([README](desktop/README.md))
- **[📚 Documentation Hub](docs/README.md)** - Navigate all documentation

## 🛠️ Development & Contributing

- **[🏗️ Architecture Guide](docs/ARCHITECTURE.md)** - Development patterns and conventions
- **[🎨 Theming Guide](docs/THEMING.md)** - Theme system and customization
- **[GitHub Repository](https://github.com/ajramos/giztui)** - Source code, issues, discussions

## 🖥️ Desktop Client (beta)

Prefer a windowed app? GizTUI also ships a **Wails desktop client** (a native
window with a React UI) that reuses the exact same Go service layer, so behavior
and configuration match the terminal. It's **keyboard-first** and aims for full
feature parity: triage, bulk actions, move-to-folder, labels, drafts, threading,
AI (summaries, prompts, inbox action plan with recategorize), saved searches,
calendar RSVP, themes, and multi-account switching.

It reuses your existing `~/.config/giztui/` config and OAuth token — no extra
setup if the TUI already works. On a fresh machine, first launch opens Google
sign-in in your system browser.

```sh
# from the repo root
cd desktop && wails dev      # live-reload dev
cd desktop && wails build    # native app → desktop/build/bin/
```

See the **[Desktop README](desktop/README.md)** for features and build details
and the **[Desktop Architecture Guide](docs/DESKTOP.md)** for how it's wired.

## 📦 Platform Support

- **Linux**: AMD64, ARM64
- **macOS**: Intel (AMD64), Apple Silicon (ARM64)  
- **Windows**: AMD64, ARM64

All platforms include:
- Native file handling and browser integration
- Cross-platform keyboard shortcuts
- Consistent feature parity

## 🎯 Use Cases

**📧 Email Power Users**
- Process large volumes of email efficiently
- Use AI to quickly understand and categorize messages  
- Bulk operations for newsletter management and cleanup

**🧠 Knowledge Workers**
- Integrate email insights into your second brain (Obsidian)
- Share important emails with teams via Slack
- Use AI prompts for meeting prep and email analysis

**💻 Terminal Enthusiasts** 
- Never leave the terminal for email management
- VIM-style navigation and operations
- Scriptable and automatable workflow integration

**🔒 Privacy-Conscious Users**
- Local AI processing with Ollama
- No cloud dependencies beyond Gmail API
- Complete control over your data and processing

## 🆘 Need Help?

- **[Getting Started Guide](docs/GETTING_STARTED.md)** - Setup and first steps
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Fixes for common problems (auth, rendering, loading)
- **[Known Issues](docs/KNOWN_ISSUES.md)** - Common problems and solutions
- **[GitHub Issues](https://github.com/ajramos/giztui/issues)** - Bug reports and feature requests
- **[GitHub Discussions](https://github.com/ajramos/giztui/discussions)** - Community support

## 📄 License

Released under the **MIT License**. See [LICENSE](LICENSE) for details.

## 🤖 Context Priming
Read README.md, CLAUDE.md, docs/*, and run git ls-files to understand this codebase.


---

**Ready to transform your Gmail workflow?** 🚀

[**Download the latest GizTUI release**](https://github.com/ajramos/giztui/releases/latest) | [**Get Started**](docs/GETTING_STARTED.md) | [**View Features**](docs/FEATURES.md)
