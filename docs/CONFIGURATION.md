# ⚙️ GizTUI Configuration Guide

Complete configuration reference for GizTUI - the AI-powered Gmail terminal client.

## 📁 Configuration Structure

GizTUI uses a unified configuration directory with the following structure:

```
~/.config/giztui/
├── config.json           # Main application configuration
├── credentials.json      # Gmail API OAuth2 credentials
├── token.json           # OAuth2 token cache (auto-generated)
├── giztui.log           # Application logs
├── giztui-{email}.db    # SQLite database (per account)
└── templates/           # Template files
    ├── obsidian/
    │   └── email.md     # Obsidian ingestion template
    └── slack/
        └── summary.md   # Slack forwarding template
```

### Migration from gmail-tui

If you previously used Gmail TUI, migrate your configuration:

```bash
# One-time migration
cp -r ~/.config/gmail-tui/* ~/.config/giztui/
```

## 🔧 Main Configuration (config.json)

### Basic Structure

```json
{
  "gmail": {
    "max_results": 50,
    "timeout": "30s"
  },
  "ui": {
    "layout": {
      "default_breakpoint": "wide"
    }
  },
  "theme": {
    "current": "slate-blue"
  },
  "shortcuts": {
    "help": "?",
    "quit": "q"
  },
  "llm": {
    "provider": "ollama",
    "timeout": "2m"
  }
}
```

## 📧 Gmail Configuration

Configure Gmail API settings and behavior:

```json
{
  "gmail": {
    "max_results": 50,
    "timeout": "30s",
    "user_email": "your-email@gmail.com",
    "search_timeout": "10s",
    "batch_size": 25,
    "auto_refresh_interval": "5m"
  }
}
```

### Gmail Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `max_results` | integer | Messages to fetch per request | `50` |
| `timeout` | string | API request timeout | `30s` |
| `user_email` | string | Gmail account (optional) | Auto-detected |
| `search_timeout` | string | Search operation timeout | `10s` |
| `batch_size` | integer | Batch operation size | `25` |
| `auto_refresh_interval` | string | Auto-refresh interval | Disabled |

## 🎨 UI Configuration

### Theme Settings

```json
{
  "theme": {
    "current": "slate-blue",
    "custom_dir": "/path/to/custom/themes",
    "auto_apply": true
  }
}
```

**Available built-in themes:**
- `slate-blue` (default)
- `dracula`
- `gmail-dark`
- `gmail-light`
- `custom-example`

**Theme directory resolution (priority order):**
1. `custom_dir` - Custom themes directory (if specified)
2. `~/.config/giztui/themes/` - User config themes directory
3. Built-in themes directory - Embedded themes in binary

### Theme Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `current` | string | Active theme name | `slate-blue` |
| `custom_dir` | string | Path to custom themes directory | Empty (uses default resolution) |
| `auto_apply` | boolean | Automatically apply theme changes | `true` |

### Layout Configuration

```json
{
  "ui": {
    "layout": {
      "default_breakpoint": "wide",
      "message_list_width": 40,
      "content_width": 60,
      "show_line_numbers": true,
      "wrap_content": true,
      "max_content_width": 120
    }
  }
}
```

**Layout breakpoints:**
- `wide` (≥120x30) - Full layout with all panels
- `medium` (≥80x25) - Condensed layout
- `narrow` (≥60x20) - Minimal layout
- `mobile` (<60x20) - Single-panel layout

#### Recipient Field Truncation

Control how long recipient lists (To/Cc fields) are displayed in email headers:

```json
{
  "layout": {
    "max_recipient_lines": 3
  }
}
```

**Configuration Options:**
- `max_recipient_lines` (number): Maximum lines for To/Cc recipient fields before truncation
  - **Default:** `3`
  - **Valid range:** `1-10`
  - **Purpose:** Prevents very long recipient lists from hiding important header fields (Labels, Date, Subject)

**How it works:**
- When recipient lists exceed the line limit, they're intelligently truncated
- Shows "... and X more recipients" indicator with accurate count
- Adapts to terminal width - wider terminals show more recipients per line
- Only affects To/Cc fields; other header fields (Subject, From, Date, Labels) remain fully visible

**Example behaviors:**
```
# With max_recipient_lines: 2
To: user1@example.com, user2@company.com, 
    user3@domain.org ... and 5 more recipients

# With max_recipient_lines: 1  
To: user1@example.com ... and 7 more recipients
```

### Display Options

```json
{
  "ui": {
    "display": {
      "show_thread_count": true,
      "show_unread_count": true,
      "show_label_colors": true,
      "compact_headers": false,
      "show_attachment_icons": true,
      "date_format": "2006-01-02 15:04",
      "relative_dates": true
    }
  }
}
```

## ⌨️ Keyboard Shortcuts

### Default Shortcuts

```json
{
  "shortcuts": {
    "quit": "q",
    "help": "?",
    "search": "s",
    "unread": "u",
    "refresh": "r",
    "compose": "c",
    "reply": "R",
    "reply_all": "E",
    "forward": "w",
    "archive": "a",
    "trash": "d",
    "toggle_read": "t",
    "undo": "U",
    "labels": "l",
    "move": "m",
    "bulk_select": "space",
    "bulk_mode": "v",
    "select_all": "*"
  }
}
```

### AI & Integration Shortcuts

```json
{
  "shortcuts": {
    "ai_summary": "y",
    "ai_regenerate": "Y",
    "prompts": "p",
    "slack_forward": "K",
    "obsidian_ingest": "O",
    "links": "L",
    "attachments": "A",
    "open_gmail": "o",
    "toggle_headers": "h",
    "save_message": "w",
    "save_raw": "W"
  }
}
```

### Navigation Shortcuts

```json
{
  "shortcuts": {
    "focus_toggle": "f",
    "fullscreen": "F",
    "next_message": "j",
    "prev_message": "k",
    "first_message": "gg",
    "last_message": "G",
    "load_more": "N",
    "thread_toggle": "T",
    "expand_all": "E",
    "collapse_all": "C"
  }
}
```

### Search Shortcuts

```json
{
  "shortcuts": {
    "quick_search_from": "F",
    "quick_search_to": "T",
    "quick_search_subject": "S",
    "quick_search_archived": "B",
    "local_filter": "/",
    "advanced_search": "Ctrl+s"
  }
}
```

### Customization Tips

- Use single characters or key combinations
- Ctrl+key format: `"Ctrl+s"`
- Shift+key format: `"Shift+t"`
- Function keys: `"F1"`, `"F2"`, etc.
- Special keys: `"space"`, `"tab"`, `"enter"`, `"esc"`

### Shortcut Conflicts and Override Behavior

**Important**: Custom shortcuts override default shortcuts. Be aware of potential conflicts:

**Common Conflicts:**
- `search: "f"` conflicts with default `forward: "f"`
- `search_to: "X"` differs from default `search_to: "T"`

**VIM Timeout Configuration:**
```json
{
  "keys": {
    "vim_navigation_timeout_ms": 250,
    "vim_range_timeout_ms": 500
  }
}
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `vim_navigation_timeout_ms` | Timeout for VIM navigation sequences (e.g., "gg") | `1000ms` |
| `vim_range_timeout_ms` | Timeout for bulk operations (e.g., "d3d") | `2000ms` |

**Resolution Strategy:**
- Check default shortcuts before customizing
- Use `:help` command to see current key bindings  
- Test shortcuts after configuration changes

### ⌨️ Complete Keyboard Configuration

#### Configuration Structure
```json
{
  "keys": {
    "_comment": "All keyboard shortcuts are customizable",
    "summarize": "y",
    "load_more": "Y",
    "search": "s",
    "quit": "q"
  }
}
```

#### Shortcut Precedence System
GizTUI uses a three-tier precedence system for keyboard shortcuts:

1. **🎯 User Configured Shortcuts** (Highest Priority)
   - Your explicit configuration always wins
   - Defined in the `"keys"` section of config.json
   - Overrides any hardcoded or auto-generated shortcuts

2. **🔧 Hardcoded Shortcuts** (Medium Priority)  
   - Built-in shortcuts that only apply when not configured
   - Uses pattern: `if !isKeyConfigured(key) { /* hardcoded behavior */ }`
   - Examples: 'F' for search_from, 'K' for Slack (if not configured)

3. **🤖 Auto-Generated Shortcuts** (Lowest Priority)
   - Automatically created based on other configuration
   - Example: If `"summarize": "y"`, then "Y" auto-maps to force_regenerate_summary
   - Can be overridden by explicit configuration
   - **Recommended**: Use explicit `"force_regenerate_summary"` parameter to avoid conflicts

#### ✅ How Override Works
```json
{
  "keys": {
    "summarize": "y",        // Creates auto-mapping: Y → force_regenerate_summary  
    "load_more": "Y"         // ✅ OVERRIDES auto-mapping: Y → load_more
  }
}
```

#### 🎯 Recommended Approach (No Conflicts)
```json
{
  "keys": {
    "summarize": "y",                     // 'y' for summary
    "force_regenerate_summary": "j",      // 'j' for force regenerate (explicit)
    "load_more": "Y"                      // 'Y' for load more
  }
}
```

In the override example:
- ✅ **"Y" does load_more** (your configuration wins)
- ✅ **"y" does summarize** (your configuration)
- ❌ **"Y" does NOT do force_regenerate_summary** (overridden)

In the recommended approach:
- ✅ **"Y" does load_more** (explicit configuration)
- ✅ **"y" does summarize** (explicit configuration)  
- ✅ **"j" does force_regenerate_summary** (explicit configuration)
- 🚫 **No conflicts** - each function has its own key

#### 🚨 Important Notes
- **You can use ANY uppercase letter** (A-Z) for any function
- **Configured shortcuts always take precedence** over automatic mappings
- **No need to avoid uppercase letters** - the system respects your configuration

## 🔍 **Validation Settings**

GizTUI includes an intelligent validation system that warns when you might accidentally lose functionality. This is especially helpful when customizing shortcuts.

### **Validation Control**
```json
{
  "keys": {
    "validate_shortcuts": true  // Enable/disable validation warnings (default: true)
  }
}
```

### **When to Enable Validation (Default)**
- **New users** - helps avoid accidentally losing functionality
- **Keyboard-focused users** - ensures all features remain accessible via shortcuts
- **Safety-first approach** - get warnings before functionality is lost

### **When to Disable Validation**
- **Command-line preference** - you prefer `:search`, `:archive`, `:obsidian` over shortcuts
- **Minimal warnings** - you know what you're doing and want cleaner output
- **Custom workflow** - you have your own way to access overridden functionality

### **Example Scenarios**

#### ✅ **Validation Enabled (Default)**
```json
{
  "keys": {
    "validate_shortcuts": true,
    "bulk_select": "s"        // Overrides hardcoded 's' (search)
                              // ⚠️ Warning: search functionality will be lost, consider adding "search" alternative
  }
}
```

#### 🔕 **Validation Disabled**
```json
{
  "keys": {
    "validate_shortcuts": false,
    "bulk_select": "s"        // Overrides hardcoded 's' (search)
                              // No warning - assumes you use :search command instead
  }
}
```

#### ✅ **Smart Alternative (No Warning Either Way)**
```json
{
  "keys": {
    "validate_shortcuts": true,  // or false - doesn't matter
    "bulk_select": "s",         // Overrides hardcoded 's' (search)
    "search": "f"               // Provides alternative for search functionality
                                // ✅ No warning - alternative provided
  }
}
```

#### Available Configuration Keys
All shortcuts from [KEYBOARD_SHORTCUTS.md](KEYBOARD_SHORTCUTS.md) can be configured. Common examples:

```json
{
  "keys": {
    // Core operations
    "summarize": "y",
    "generate_reply": "g", 
    "reply": "r",
    "compose": "c",
    "load_more": "Y",
    "search": "s",
    "quit": "q",
    
    // Advanced features
    "obsidian": "O",
    "slack": "K", 
    "bulk_mode": "v",
    "command_mode": ":",
    
    // Content navigation  
    "search_next": "n",
    "search_prev": "N",
    "content_search": "/"
  }
}
```

## 🧠 AI Configuration

### Ollama (Local AI)

```json
{
  "llm": {
    "provider": "ollama",
    "timeout": "2m",
    "cache_enabled": true,
    "ollama": {
      "base_url": "http://localhost:11434",
      "model": "llama2",
      "temperature": 0.7,
      "max_tokens": 2000,
      "timeout": "120s"
    }
  }
}
```

### Amazon Bedrock (Cloud AI)

```json
{
  "llm": {
    "provider": "bedrock",
    "timeout": "2m",
    "cache_enabled": true,
    "bedrock": {
      "region": "us-east-1",
      "model": "anthropic.claude-3-sonnet-20240229-v1:0",
      "max_tokens": 2000,
      "temperature": 0.7,
      "timeout": "120s"
    }
  }
}
```

### AI Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `provider` | string | AI provider (`ollama` or `bedrock`) | `ollama` |
| `timeout` | string | Overall AI operation timeout | `2m` |
| `cache_enabled` | boolean | Enable SQLite result caching | `true` |
| `temperature` | number | AI creativity (0.0-1.0) | `0.7` |
| `max_tokens` | integer | Maximum response length | `2000` |

## 📝 Prompt Configuration

### Built-in Prompts

GizTUI includes several built-in AI prompts:

```json
{
  "llm": {
    "prompts": {
      "summary": "templates/prompts/summary.md",
      "label_suggestion": "Suggest appropriate Gmail labels for this email...",
      "reply_draft": "Generate a professional reply to this email..."
    }
  }
}
```

### Custom Prompts

Create custom prompt templates in `~/.config/giztui/templates/prompts/`:

**Example: `summary.md`**
```markdown
---
name: "Email Summary"
description: "Generate a concise summary of the email"
category: "summary"
---

Please provide a concise summary of this email:

**From:** {{from}}
**Subject:** {{subject}}
**Date:** {{date}}

**Content:**
{{body}}

Summarize in 2-3 sentences focusing on key points and any action items.
```

### Variable Substitution

Available variables in prompts:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{from}}` | Sender email | `john@example.com` |
| `{{to}}` | Recipient email | `you@gmail.com` |
| `{{subject}}` | Email subject | `Meeting Tomorrow` |
| `{{body}}` | Email content | Full email text |
| `{{date}}` | Email date | `2025-01-02 15:04:05` |
| `{{labels}}` | Gmail labels | `Work, Important` |
| `{{message_id}}` | Gmail message ID | `17a2b3c4d5e6f7g8` |
| `{{messages}}` | Multiple messages (bulk) | Combined content |

## 🔌 Integration Configuration

### Slack Integration

```json
{
  "slack": {
    "enabled": true,
    "channels": [
      {
        "name": "general",
        "webhook_url": "https://hooks.slack.com/services/...",
        "default_format": "summary"
      },
      {
        "name": "work-updates",
        "webhook_url": "https://hooks.slack.com/services/...",
        "default_format": "compact"
      }
    ],
    "default_channel": "general",
    "summary_prompt": "templates/slack/summary.md",
    "include_attachments": true,
    "max_message_length": 4000
  }
}
```

### Obsidian Integration

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
    "template_file": "templates/obsidian/email.md",
    "repopack_template_file": "templates/obsidian/repopack.md"
  }
}
```

### Obsidian Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `enabled` | boolean | Enable Obsidian integration | `true` |
| `vault_path` | string | Path to Obsidian vault | Required |
| `ingest_folder` | string | Folder for imported emails | `"00-Inbox"` |
| `filename_format` | string | Template for filename generation | `"{{date}}_{{subject_slug}}_{{from_domain}}"` |
| `history_enabled` | boolean | Track import history | `true` |
| `prevent_duplicates` | boolean | Prevent duplicate imports | `true` |
| `max_file_size` | integer | Maximum file size in bytes | `1048576` |
| `include_attachments` | boolean | Include email attachments | `true` |
| `template_file` | string | Path to email template file | `"templates/obsidian/email.md"` |
| `repopack_template_file` | string | Path to repopack template file for bulk mode | `"templates/obsidian/repopack.md"` |
```

#### Obsidian Template Example

Create `~/.config/giztui/templates/obsidian/email.md`:

```markdown
---
title: "{{subject}}"
date: {{date}}
from: {{from}}
type: email
status: inbox
tags: [email, {{labels}}]
---

# {{subject}}

**Email Details:**
- **From:** {{from}}
- **To:** {{to}}
- **Date:** {{date}}
- **Labels:** {{labels}}

{% if comment %}**Personal Note:** {{comment}}

{% endif %}**Content:**
{{body}}

**Message ID:** {{message_id}}
**Ingested:** {{ingest_date}}
```

## 🔍 Search Configuration

### Search Settings

```json
{
  "search": {
    "case_sensitive": false,
    "regex_enabled": true,
    "highlight_matches": true,
    "max_results": 100,
    "search_timeout": "30s",
    "include_archived": false,
    "auto_complete": true
  }
}
```

### Quick Search Templates

```json
{
  "search": {
    "quick_searches": {
      "unread": "is:unread",
      "important": "is:important",
      "today": "newer_than:1d",
      "this_week": "newer_than:7d",
      "has_attachment": "has:attachment",
      "large_emails": "size:>1M"
    }
  }
}
```

## 📎 Attachment Configuration

```json
{
  "attachments": {
    "download_dir": "~/Downloads/giztui",
    "auto_open": false,
    "max_file_size": 104857600,
    "allowed_types": ["pdf", "doc", "docx", "txt", "jpg", "png"],
    "preview_images": true,
    "organize_by_date": true
  }
}
```

## 🔧 Advanced Configuration

### Threading Configuration

```json
{
  "threading": {
    "enabled": true,
    "default_view": "flat",
    "auto_expand_unread": true,
    "show_thread_count": true,
    "indent_replies": true,
    "max_thread_depth": 10,
    "thread_summary_enabled": true,
    "preserve_thread_state": true
  }
}
```

### Threading Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `enabled` | boolean | Enable threading functionality | `true` |
| `default_view` | string | Default message view: "flat" or "thread" | `"flat"` |
| `auto_expand_unread` | boolean | Auto-expand threads with unread messages | `true` |
| `show_thread_count` | boolean | Show message count badges on threads | `true` |
| `indent_replies` | boolean | Indent reply messages in thread view | `true` |
| `max_thread_depth` | integer | Maximum thread nesting level | `10` |
| `thread_summary_enabled` | boolean | Enable AI-powered thread summaries | `true` |
| `preserve_thread_state` | boolean | Remember expanded/collapsed state | `true` |

### Performance Settings

Configure performance optimizations including background preloading for instant navigation:

```json
{
  "performance": {
    "_comment": "Performance optimization settings - background preloading improves navigation speed",
    "preloading": {
      "_comment": "Background message preloading for instant navigation",
      "enabled": true,
      "next_page": {
        "_comment": "Preload next page when scrolling reaches threshold",
        "enabled": true,
        "threshold": 0.7,
        "max_pages": 2
      },
      "adjacent_messages": {
        "_comment": "Preload messages around current selection for smooth navigation",
        "enabled": true,
        "count": 3
      },
      "limits": {
        "_comment": "Resource limits to prevent excessive memory/API usage",
        "background_workers": 3,
        "cache_size_mb": 50,
        "api_quota_reserve_percent": 20
      }
    },
    "cache_size": 1000,
    "background_sync": true,
    "lazy_loading": true,
    "compression_enabled": true,
    "max_memory_usage": "500MB"
  }
}
```

### Preloading Configuration Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `preloading.enabled` | boolean | Master toggle for background preloading | `true` |
| `next_page.enabled` | boolean | Enable next page preloading | `true` |
| `next_page.threshold` | number | Scroll threshold (0.0-1.0) to trigger preloading | `0.7` |
| `next_page.max_pages` | integer | Maximum pages to preload ahead | `2` |
| `adjacent_messages.enabled` | boolean | Enable adjacent message preloading | `true` |
| `adjacent_messages.count` | integer | Number of messages to preload around selection | `3` |
| `limits.background_workers` | integer | Maximum concurrent background workers | `3` |
| `limits.cache_size_mb` | integer | Maximum cache size in MB | `50` |
| `limits.api_quota_reserve_percent` | integer | Reserve percentage of API quota | `20` |

### Preloading Behavior

**Next Page Preloading:**
- Triggers when scrolling reaches the threshold (70% by default)
- Preloads the next page of messages in the background
- Eliminates waiting time when clicking "Load More"
- Respects `max_pages` limit to prevent excessive API usage

**Adjacent Message Preloading:**
- Preloads messages around the current selection
- Provides instant navigation between messages
- Configurable count (3 messages by default: 1 before, current, 2 after)
- Uses intelligent caching with LRU eviction

**Resource Management:**
- Worker pool limits concurrent background operations
- Cache size prevents excessive memory usage
- API quota reserve ensures interactive operations remain responsive
- Smart eviction based on Least Recently Used (LRU) algorithm

### Runtime Preloading Control

Use the `:preload` command for runtime control:

```bash
# Enable/disable preloading
:preload on
:preload off

# Check status
:preload status

# Clear caches
:preload clear

# Control specific features
:preload next on/off        # Next page preloading
:preload adjacent on/off    # Adjacent message preloading
```

### Logging Configuration

```json
{
  "logging": {
    "level": "info",
    "file": "~/.config/giztui/giztui.log",
    "max_size": "10MB",
    "max_backups": 3,
    "compress": true
  }
}
```

## 🌍 Environment Variables

Override configuration paths with environment variables:

```bash
# Configuration paths
export GMAIL_TUI_CONFIG=~/.config/giztui/config.json
export GMAIL_TUI_CREDENTIALS=~/.config/giztui/credentials.json
export GMAIL_TUI_TOKEN=~/.config/giztui/token.json

# Runtime settings
export GIZTUI_LOG_LEVEL=debug
export GIZTUI_CACHE_DIR=~/.cache/giztui
export GIZTUI_THEME=dracula
```

## 📋 Command Line Options

Override configuration settings from command line:

```bash
# Basic options
giztui --config custom-config.json
giztui --credentials /path/to/creds.json
giztui --setup

# Theme and UI options
giztui --theme dracula
giztui --layout compact

# AI provider options
giztui --llm-provider bedrock
giztui --llm-model claude-3
giztui --ollama-model llama2
```

## 🔒 Security Settings

### OAuth2 Security

```json
{
  "security": {
    "token_refresh_threshold": "5m",
    "max_retry_attempts": 3,
    "secure_token_storage": true,
    "revoke_on_exit": false
  }
}
```

### Privacy Settings

```json
{
  "privacy": {
    "cache_sensitive_data": false,
    "clear_cache_on_exit": false,
    "log_email_content": false,
    "anonymize_logs": true
  }
}
```

## 📊 Example Complete Configuration

Here's a complete example configuration with common customizations:

```json
{
  "gmail": {
    "max_results": 75,
    "timeout": "45s"
  },
  "ui": {
    "layout": {
      "default_breakpoint": "wide",
      "show_line_numbers": true
    },
    "display": {
      "show_thread_count": true,
      "relative_dates": true,
      "compact_headers": true
    }
  },
  "theme": {
    "current": "dracula",
    "auto_apply": true
  },
  "shortcuts": {
    "ai_summary": "s",
    "quick_search_from": "f",
    "obsidian_ingest": "o",
    "bulk_select": "space"
  },
  "llm": {
    "provider": "ollama",
    "timeout": "3m",
    "cache_enabled": true,
    "ollama": {
      "model": "llama3",
      "temperature": 0.5,
      "max_tokens": 1500
    }
  },
  "slack": {
    "enabled": true,
    "channels": [
      {
        "name": "email-updates",
        "webhook_url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
        "default_format": "summary"
      }
    ]
  },
  "obsidian": {
    "enabled": true,
    "vault_path": "~/Obsidian/SecondBrain",
    "template_file": "templates/obsidian/email.md",
    "repopack_template_file": "templates/obsidian/repopack.md"
  },
  "search": {
    "highlight_matches": true,
    "case_sensitive": false
  },
  "threading": {
    "enabled": true,
    "default_view": "thread",
    "auto_expand_unread": true
  },
  "performance": {
    "preloading": {
      "enabled": true,
      "next_page": {
        "enabled": true,
        "threshold": 0.8,
        "max_pages": 1
      },
      "adjacent_messages": {
        "enabled": true,
        "count": 2
      },
      "limits": {
        "background_workers": 2,
        "cache_size_mb": 30,
        "api_quota_reserve_percent": 25
      }
    }
  }
}
```

## 🔧 Troubleshooting Configuration

### Common Issues

1. **Configuration not loading**
   - Check JSON syntax with `jq . ~/.config/giztui/config.json`
   - Verify file permissions are readable
   - Check logs for parsing errors

2. **Shortcuts not working**
   - Ensure no conflicts between shortcuts
   - Check terminal key capture (some terminals intercept certain keys)
   - Verify quotes around multi-key shortcuts

3. **AI features not working**
   - Verify Ollama is running: `ollama list`
   - Check model name matches exactly
   - Test API connectivity

4. **Theme not applying**
   - Verify theme file exists
   - Check theme YAML syntax
   - Ensure theme name matches filename

### Configuration Validation

Use the built-in validation:

```bash
# Validate configuration
giztui --validate-config

# Show current configuration
giztui --show-config

# Reset to defaults
giztui --reset-config
```

## 📚 Learn More

- [Getting Started Guide](GETTING_STARTED.md) - Setup and first steps
- [Features Documentation](FEATURES.md) - Complete feature list
- [Keyboard Shortcuts](KEYBOARD_SHORTCUTS.md) - Complete shortcut reference
- [Theming Guide](THEMING.md) - Theme customization