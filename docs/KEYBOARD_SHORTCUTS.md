# ⌨️ GizTUI Keyboard Shortcuts

Complete keyboard shortcut reference for GizTUI - the AI-powered Gmail terminal client.

## 🎯 Essential Shortcuts

### Core Navigation
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View selected message | Open message for reading |
| `q` | Quit | Exit the application |
| `?` | Help | Show help screen with shortcuts |
| `Esc` | Cancel/Back | Cancel current operation or go back |
| `↑↓` | Navigate | Move up/down in lists |
| `←→` | Navigate | Move left/right in content |

### Basic Email Operations
| Key | Action | Description |
|-----|--------|-------------|
| `r` | Toggle read/unread | Mark message as read or unread |
| `a` | Archive | Move message to archive |
| `d` | Trash | Move message to trash |
| `u` | Show unread | Filter to show only unread messages |
| `s` | Search | Open search interface |
| `U` | Undo | Reverse last action (archive, trash, read/unread, labels) |

## 📧 Email Management

### Message Operations
| Key | Action | Description |
|-----|--------|-------------|
| `c` | Compose | Create new email with CC/BCC support |
| `R` | Reply | Reply to current message |
| `E` | Reply all | Reply to all recipients |
| `w` | Forward | Forward current message |
| `D` | Drafts | View and edit draft messages |
| `N` | Load more | Fetch next 50 messages |
| `B` | Archived | Show archived messages |
| `r` | Refresh | Refresh current view |

### Message Content
| Key | Action | Description |
|-----|--------|-------------|
| `w` | Save as text | Save message as rendered .txt file |
| `W` | Save as raw | Save message as raw .eml file |
| `h` | Toggle headers | Show/hide email headers |
| `f` | Fullscreen | Toggle fullscreen message view |
| `t` | Focus toggle | Switch focus between list and content |
| `M` | Toggle Markdown | Toggle between Markdown-rendered view and raw/plain view (per message) |

## 🔍 Search & Navigation

### Search Operations
| Key | Action | Description |
|-----|--------|-------------|
| `s` | Search | Open Gmail search interface |
| `/` | Local filter | Filter current messages locally |
| `F` | Quick search: From | Search emails from current sender |
| `T` | Quick search: To | Search emails to current sender |
| `S` | Quick search: Subject | Search by current subject |
| `B` | Quick search: Archived | Search archived messages |

### Content Search (Within Message)
| Key | Action | Description |
|-----|--------|-------------|
| `/searchterm` | Search content | Search within message and highlight matches |
| `n` | Next match | Navigate to next search match |
| `N` | Previous match | Navigate to previous search match |
| `Esc` | Clear search | Clear search highlights |

### VIM-Style Navigation
| Key | Action | Description |
|-----|--------|-------------|
| `gg` | Go to first | Jump to first message |
| `G` | Go to last | Jump to last message |
| `:5` + Enter | Jump to line | Jump to message number 5 |
| `:$` + Enter | Jump to end | Jump to last message |

### Content Navigation (Within Message)
| Key | Action | Description |
|-----|--------|-------------|
| `gg` | Top of message | Go to top of message content |
| `G` | Bottom of message | Go to bottom of message content |
| `Ctrl+K` | Paragraph up | Navigate up by paragraphs (10 lines) |
| `Ctrl+J` | Paragraph down | Navigate down by paragraphs (10 lines) |
| `Ctrl+H` | Word left | Navigate left by words |
| `Ctrl+L` | Word right | Navigate right by words |

## 🎯 VIM Range Operations

Execute operations on multiple messages using VIM-style range syntax: `{operation}{count}{operation}`

| Range Command | Action | Example |
|---------------|---------|---------|
| `t3t` | Toggle read for 3 messages | Toggles read status for messages 1-3 |
| `a5a` | Archive 5 messages | Archives messages 1-5 |
| `d2d` | Delete 2 messages | Moves messages 1-2 to trash |
| `s4s` | Select 4 messages | Selects messages 1-4 for bulk mode |
| `m7m` | Move 7 messages | Opens move dialog for messages 1-7 |
| `l3l` | Label 3 messages | Opens label picker for messages 1-3 |

**How it works:**
1. Press operation key (`t`, `a`, `d`, etc.)
2. Type count (`3`, `5`, `2`, etc.)
3. Press operation key again (`t`, `a`, `d`, etc.)

## 🏷️ Labels & Organization

### Label Management
| Key | Action | Description |
|-----|--------|-------------|
| `l` | Labels | Manage message labels (contextual panel) |
| `o` | Suggest label | AI-powered label suggestions |
| `m` | Move | Enhanced move panel with system folders + labels |

## 🔥 Bulk Operations

### Bulk Mode
| Key | Action | Description |
|-----|--------|-------------|
| `v` | Enter bulk mode | Enter bulk selection mode |
| `b` | Enter bulk mode | Alternative to enter bulk selection |
| `space` | Toggle selection | Select/deselect current message |
| `*` | Select all | Select all visible messages |

### Bulk Actions
| Key | Action | Description |
|-----|--------|-------------|
| `a` | Archive selected | Archive all selected messages |
| `d` | Trash selected | Move selected messages to trash |
| `t` | Toggle read selected | Toggle read/unread for selected messages |
| `m` | Move selected | Move selected messages to folder/label |
| `p` | Bulk prompts | Apply AI prompt to selected messages |
| `K` | Slack forward | Forward selected messages to Slack |
| `O` | Obsidian ingest | Ingest selected messages to Obsidian |

## 🧵 Message Threading

### Threading Operations
| Key | Action | Description |
|-----|--------|-------------|
| `T` | Toggle threading | Switch between thread and flat view |
| `Enter` | Expand/collapse | Expand or collapse thread (when on thread root) |
| `E` | Expand all | Expand all threads in current view |
| `C` | Collapse all | Collapse all threads to show only root messages |
| `Shift+T` | Thread summary | Generate AI summary of selected thread |

## 🧠 AI Features

### AI Operations
| Key | Action | Description |
|-----|--------|-------------|
| `y` | AI summary | Generate/show AI summary of current message |
| `j` | Regenerate summary | Force regenerate AI summary (ignore cache) |
| `p` | Prompt picker | Open AI prompt library (single or bulk mode) |
| `Ctrl+P` | Preview prompt | In the prompt picker, preview the highlighted prompt's description + full template in a popup (`Esc`/`Ctrl+P` to close). Works from the search field or the list. |
| `g` | Generate reply | Experimental AI reply generation |
| `P` | Action Plan | AI-triage unread inbox into actionable groups |
| `Esc` | Cancel AI | Cancel any active streaming AI operation |

## ✨ Prompt Configurator

The Prompt Configurator lets you generate AI prompt templates from natural-language intent instead of writing them by hand. Open it from the prompt picker: press `p` on a message (or in bulk mode after selecting messages), then choose **✨ Create new with AI...** as the first item in the picker. The configurator panel opens alongside the current message(s).

### Panel Layout

The configurator has three areas cycled via `Tab` / `Shift+Tab`:

1. **Intent** (top) — describe what you want in plain language, press `Enter` to generate a prompt
2. **Editable prompt** (middle) — the generated prompt appears here; edit it freely
3. **Refine** (bottom) — type a refinement instruction (e.g. "output as JSON"), press `Enter` to refine

### In-Panel Keys
| Key | Action | Description |
|-----|--------|-------------|
| `Tab` | Focus next | Cycle focus: Intent → Editable → Refine |
| `Shift+Tab` | Focus previous | Cycle focus in reverse |
| `Enter` | Generate / Refine | Generate prompt from Intent, or refine from Refine input |
| `Ctrl+G` | Apply prompt | Apply current prompt to the scoped message(s); result appears in the AI panel |
| `Ctrl+R` | Regenerate | Re-generate the prompt from the Intent input |
| `Ctrl+S` | Save prompt | Open save dialog (name / description / category) to store prompt in your library |
| `Esc` | Close | Close the configurator; cancels any in-progress LLM call |

### Prompt Configurator Commands
| Command | Alias | Description |
|---------|-------|-------------|
| `:prompt-new` | `:pn` | Open the configurator with current context (single message, bulk selection, or draft) |
| `:prompt-refine <instruction>` | `:prf` | Refine the active prompt with the given instruction |
| `:prompt-save` | `:ps` | Open the save dialog for the currently active prompt |

### Notes

- **Saved prompts** appear in both the single-message prompt picker and the bulk prompt picker after saving; they are immediately available for reuse.
- **Unsaved prompts** are discarded when you close the configurator with `Esc` — use `Ctrl+S` before closing to keep them.
- **LLM model quality matters**: prompt generation and refinement quality depends on the model configured for your LLM backend. A small model (e.g. 1.5B parameters) may produce poor or incomplete prompts — a 7B+ model is recommended for best results.
- The in-panel keys (`Ctrl+G`, `Ctrl+R`, `Ctrl+S`) are currently hardcoded. Config keys (`keys.prompt_apply`, `keys.prompt_regenerate`, `keys.save_prompt`) exist in `config.json` but are reserved for future configurable binding support.

## 📋 Inbox Action Plan

The Action Plan groups messages into actionable categories and presents a panel where you dispatch each group with one keystroke. It runs in **fast mode** — it uses only the subject, sender and snippet already in memory, so it makes **no extra Gmail API calls**. Open it with `P` (capital P) or `:action-plan`.

**Scope — selection-first:** if you have messages selected (via `v` / `space` in bulk mode), the Action Plan analyzes those selected messages only. Otherwise it falls back to the **unread** messages currently loaded in the list. The panel header shows which scope is active, e.g. `12 selected` or `47 unread (inbox)`.

Messages are processed in batches (default 50); categories stream into the panel as each batch completes. Press `Esc` at any time to cancel — categories rendered so far stay visible until you close.

### Panel Structure

The panel is a two-level tree. **Category nodes** are the top level (e.g. `[a] Archive 18 Newsletters  3/5`). Each category can be expanded to show the individual emails inside it.

### In-Panel Keys
| Key | Action | Description |
|-----|--------|-------------|
| `↑` / `↓` | Navigate | Move between nodes (categories and emails) |
| `Enter` / `→` | Expand category | Show the emails inside the highlighted category |
| `←` | Collapse category | Hide the email children of the highlighted category |
| `space` | Exclude / include email | When on an email node, toggle it off (`[ ]`) or back on (`[x]`); actions apply only to checked emails |
| archive key | Archive | Archive the highlighted category's checked emails |
| trash key | Trash | Trash the highlighted category's checked emails |
| label key | Label | Apply the category's suggested label to its checked emails (created if needed) |
| toggle-read key | Mark read | Mark the highlighted category's checked emails as read |
| `Ctrl+R` | Remember rule | Open an editable modal pre-seeded with a preference rule suggestion; editing + `Enter` saves it for future analyses |
| `Esc` | Close / Cancel | Cancel an in-progress analysis, or close the panel |

The action keys (archive / trash / label / toggle-read) reuse your configured bindings from the message list. Each category shows its suggested action's key and checked count in brackets, e.g. `[a] Archive 5/18 Newsletters` (5 checked out of 18). A "Read manually" bucket lists messages the LLM declined to categorize.

### Learning Rules

`Ctrl+R` saves a natural-language preference rule (e.g. "Never trash emails from tldr.tech"). Rules are injected into the analyzer prompt on future analyses so the LLM honors them. Manage saved rules with `:action-plan rules` (in the manager: `a` add, `d` delete, `Esc` close).

### Commands
| Command | Aliases | Description |
|---------|---------|-------------|
| `:action-plan` | `:plan`, `:ap` | Open the Action Plan (selection-first; falls back to unread inbox) |
| `:action-plan with-prompt <name-or-id>` | — | Open the Action Plan using one of your saved prompts as the analyzer (falls back to the default if not found) |
| `:action-plan rules` | — | Open the preference rules manager (add / delete saved learning rules) |

### Configuration
Settings live under `inbox_analyzer` in `~/.config/giztui/config.json`:

| Setting | Default | Description |
|---------|---------|-------------|
| `batch_size` | 50 | Messages sent to the LLM per batch |
| `max_batches` | 10 | Safety cap on the number of batches per run |
| `default_prompt_id` | `""` | Reserved for a future default override prompt |

The shortcut is configurable via `keys.action_plan` (default `"P"`).

### Notes
- **Local / small models:** the default `batch_size` of 50 is tuned for capable cloud models. A small local model (e.g. a 7B Ollama model) can struggle to follow the JSON schema across 50 messages and may return no categories. If you see "no actionable groups", lower `inbox_analyzer.batch_size` to **15–20**.
- **Model quality matters:** category quality depends entirely on the configured LLM. The code degrades gracefully (malformed output → one repair retry → messages fall back to "Read manually"), but a stronger model yields better groupings.
- If there are no messages to analyze (no selection and no unread), you'll get an info message instead of opening the panel.

## ⚙️ Customizing Shortcuts

**All shortcuts listed above can be customized** in your `config.json` file. See [CONFIGURATION.md](CONFIGURATION.md) for detailed customization guidance.

### Shortcut Precedence
When you customize shortcuts, the priority order is:
1. **Your configured shortcuts** (highest priority - always used)
2. **Hardcoded shortcuts** (only used if not configured) 
3. **Auto-generated shortcuts** (lowest priority - can be overridden)

### Auto-Generated Shortcuts
- If you configure `"summarize": "x"`, the system automatically creates `"X"` (uppercase) for force regenerate
- **Your explicit configuration always wins**: If you configure `"load_more": "Y"`, it will override any auto-generated "Y" mapping
- **Recommended**: Use explicit `"force_regenerate_summary"` parameter to avoid conflicts

### Examples
```json
{
  "keys": {
    "summarize": "y",                     // 'y' for summary
    "force_regenerate_summary": "j",      // 'j' for force regenerate (explicit, no conflicts)
    "load_more": "Y"                      // 'Y' for load more
  }
}
```

## 🔌 Integrations

### Slack Integration
| Key | Action | Description |
|-----|--------|-------------|
| `K` | Forward to Slack | Send current/selected messages to Slack |

### Obsidian Integration
| Key | Action | Description |
|-----|--------|-------------|
| `Shift+O` | Ingest to Obsidian | Send current/selected messages to Obsidian with mode option |

**Repopack Mode:** When using `Shift+O` in bulk mode, check the "📦 Combined file:" checkbox to create a single consolidated Markdown file instead of individual files. Use `:obsidian repack` or `:obs repack` commands to open the picker with repopack mode pre-selected.

### Calendar Integration
| Key | Action | Description |
|-----|--------|-------------|
| `Shift+V` | RSVP to meeting | Respond to calendar invitations |

## 🔗 Productivity Tools

### Link Management
| Key | Action | Description |
|-----|--------|-------------|
| `L` | Link picker | Open link picker for current message |
| `Enter` | Open link | Open selected link in browser |
| `Ctrl+Y` | Copy link | Copy selected link to clipboard |
| `1-9` | Quick open | Open link by number |

### Attachment Management
| Key | Action | Description |
|-----|--------|-------------|
| `A` | Attachment picker | Open attachment picker for current message |
| `Enter` | Download | Download selected attachment |
| `Ctrl+S` | Save as | Save attachment with custom name |
| `1-9` | Quick download | Download attachment by number |

### Gmail Web Integration
| Key | Action | Description |
|-----|--------|-------------|
| `O` | Open in Gmail | Open current message in Gmail web interface |

### Account Management
| Key | Action | Description |
|-----|--------|-------------|
| `:accounts` | Account picker | Open account picker for switching between accounts |
| `Enter` | Switch account | Switch to selected account |
| `1-9` | Quick switch | Switch to account by number |

## 📋 Command System

### Command Mode
| Key | Action | Description |
|-----|--------|-------------|
| `:` | Command mode | Enter command mode (k9s-style) |
| `Tab` | Auto-complete | Auto-complete commands |
| `↑↓` | History | Navigate command history |
| `Enter` | Execute | Execute command |
| `Esc` | Cancel | Cancel command mode |

### Essential Commands
| Command | Shortcut Equivalent | Description |
|---------|-------------------|-------------|
| `:help` | `?` | Show help screen |
| `:quit` or `:q` | `q` | Exit application |
| `:search <query>` | `s` | Search emails |
| `:unread` | `u` | Show unread messages |
| `:archive` or `:a` | `a` | Archive message(s) |
| `:trash` or `:d` | `d` | Move to trash |
| `:labels` or `:l` | `l` | Manage labels |
| `:compose` | `c` | Compose new message |
| `:reply` or `:r` | `R` | Reply to message |
| `:forward` or `:f` | `w` | Forward message |
| `:drafts` | `D` | View drafts |
| `:accounts` | - | Open account picker |

### Thread Commands
| Command | Shortcut Equivalent | Description |
|---------|-------------------|-------------|
| `:threads` | `T` | Switch to threaded view |
| `:flatten` | `T` | Switch to flat view |
| `:thread-summary` | `Shift+T` | Generate thread summary |
| `:expand-all` | `E` | Expand all threads |
| `:collapse-all` | `C` | Collapse all threads |

### Integration Commands
| Command | Shortcut Equivalent | Description |
|---------|-------------------|-------------|
| `:slack` | `K` | Forward to Slack |
| `:obsidian` | `Shift+O` | Ingest to Obsidian (individual files) |
| `:obsidian repack` | - | Create combined repopack file |
| `:obs repack` | - | Short alias for obsidian repack |
| `:links` | `L` | Open link picker |
| `:attachments` | `A` | Open attachment picker |
| `:gmail` or `:web` | `O` | Open in Gmail web |

### Utility Commands
| Command | Description |
|---------|-------------|
| `:themes` | List available themes |
| `:theme set <name>` | Switch to theme |
| `:refresh` | Refresh current view |
| `:autorefresh` / `:arr` | Toggle background inbox auto-refresh on/off |
| `:autorefresh <duration>` / `:arr 2m` | Enable auto-refresh and set the poll interval at runtime (min 1m) |
| `:undo` | Undo last action |
| `:version` | Show version information |
| `:config` | Show configuration |
| `:config migrate` | Add missing default options to your config.json (backup written) |
| `:markdown` or `:md` | Toggle Markdown ↔ raw rendering for the current message (same as `M`) |
| `:touch-up` | Toggle LLM whitespace touch-up for the current message |

### Performance Commands
| Command | Description |
|---------|-------------|
| `:preload status` | Show preloading status and statistics |
| `:preload on` | Enable background preloading |
| `:preload off` | Disable background preloading |
| `:preload clear` | Clear all preloaded caches |
| `:preload next on/off` | Control next page preloading |
| `:preload adjacent on/off` | Control adjacent message preloading |

### Prompt Management Commands
| Command | Shortcut | Description |
|---------|----------|-------------|
| `:prompt stats` or `:prompt s` | `p` (opens prompt picker) | Show prompt usage statistics |
| `:prompt list` or `:prompt l` | - | Manage prompts |
| `:prompt create` or `:prompt c` | - | Create new prompt |
| `:prompt update` or `:prompt u` | - | Update existing prompt |
| `:prompt delete` or `:prompt d` | - | Delete prompt |
| `:prompt export` or `:prompt e` | - | Export prompts |

## 🎨 Theme & UI

### Theme Operations
| Key | Action | Description |
|-----|--------|-------------|
| `:themes` | List themes | Show available themes |
| `:theme set dracula` | Switch theme | Change to Dracula theme |

### Available Themes
- `slate-blue` (default)
- `dracula`
- `gmail-dark`
- `gmail-light`
- `custom-example`

## 🎮 Customization

### Configurable Shortcuts

All shortcuts can be customized in `~/.config/giztui/config.json`. You can override any default shortcut to match your workflow preferences.

```json
{
  "shortcuts": {
    "ai_summary": "s",
    "quick_search_from": "f",
    "obsidian_ingest": "o",
    "bulk_select": "space",
    "compose": "c",
    "reply": "R",
    "archive": "a",
    "trash": "d"
  }
}
```

### Shortcut Formats
- **Single character**: `"q"`, `"s"`, `"a"`
- **Ctrl combinations**: `"Ctrl+s"`, `"Ctrl+k"`
- **Shift combinations**: `"Shift+o"`, `"Shift+t"`
- **Function keys**: `"F1"`, `"F2"`, `"F12"`
- **Special keys**: `"space"`, `"tab"`, `"enter"`, `"esc"`

### Popular Customization Examples

#### VIM-Style Shortcuts
```json
{
  "shortcuts": {
    "compose": "i",
    "search": "/",
    "help": ":h"
  }
}
```

#### Emacs-Style Shortcuts
```json
{
  "shortcuts": {
    "compose": "Ctrl+x",
    "search": "Ctrl+s",
    "quit": "Ctrl+x"
  }
}
```

#### Function Key Layout
```json
{
  "shortcuts": {
    "compose": "F1",
    "search": "F2",
    "ai_summary": "F3",
    "help": "F12"
  }
}
```

### Customization Best Practices

#### Choose Intuitive Keys
- Use keys that relate to the action (e.g., 'r' for reply, 'c' for compose)
- Consider your muscle memory from other applications
- Avoid conflicts with navigation keys (arrows, Tab, Enter, Esc)

#### Maintain Consistency
- Use similar patterns across related actions
- Keep frequently used shortcuts easily accessible
- Consider ergonomics for your keyboard layout

#### Test and Iterate
- Start with a few customizations
- Gradually adapt your workflow
- Keep a backup of working configurations
- Document your custom setup

## 💡 Tips & Tricks

### Efficiency Tips
1. **Use range operations** - `a5a` is faster than selecting 5 messages individually
2. **Learn command aliases** - `:a` is quicker than `:archive`
3. **Use quick search** - `F` to search from current sender instantly
4. **Bulk mode shortcuts** - Select multiple with `space`, then `p` for bulk AI analysis
5. **Content search** - Use `/term` to find specific content within long messages

### Power User Shortcuts
1. **Thread management** - `T` to toggle view, `E`/`C` to expand/collapse all
2. **AI workflow** - `y` for summary, then `p` for detailed analysis
3. **Integration workflow** - `K` to share via Slack, `Shift+O` to save in Obsidian (individual/repopack)
4. **Search mastery** - Combine `/` for local filter with `s` for Gmail search

### Context Awareness
- **Message list mode**: VIM keys (`gg`, `G`) navigate messages
- **Message content mode**: VIM keys navigate within message content
- **Bulk mode**: Most operations apply to selected messages
- **Command mode**: Tab completion shows available options

## 📚 Learn More

- [Getting Started Guide](GETTING_STARTED.md) - Setup and first steps
- [Features Documentation](FEATURES.md) - Complete feature list
- [Configuration Guide](CONFIGURATION.md) - Customization options
- [User Guide](USER_GUIDE.md) - Detailed usage instructions

---

## 🎯 Quick Reference Card

**Essential Navigation:** `↑↓` navigate, `Enter` open, `q` quit, `?` help  
**Email Operations:** `r` read/unread, `a` archive, `d` trash, `U` undo  
**Search:** `s` search, `/` filter, `F` from sender, `S` by subject  
**AI Features:** `y` summary, `p` prompts, `Y` regenerate  
**Bulk Operations:** `v` bulk mode, `space` select, `*` select all  
**Integrations:** `K` Slack, `Shift+O` Obsidian, `L` links, `A` attachments  
**Rendering:** `M` toggle Markdown/raw view, `:touch-up` toggle LLM whitespace cleanup  
**Commands:** `:` command mode, `:help` help, `:q` quit, `:search <term>` search