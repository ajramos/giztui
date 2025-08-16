# TODO List - Gmail TUI Project (Merged & Extended)

## 🎯 Roadmap

### Priorities

- ✅ View inbox and labels  
- ✅ Read emails  
- ✅ Mark as read/unread  
- ✅ Archive and move to trash  
- ✅ Manage labels (add, remove, create)  
- ✅ Load more messages (when list is focused)  
- ✅ Basic search and navigation support  
- 🚧 Compose, Reply, Drafts, Attachments (WIP)  
- 🚧 Streaming LLM output for progressive response (Pending)  
- 🚧 Improve status bar experience and legends  
- 🚧 Advanced filter and search improvements  
- 🚧 Planning integration with Obsidian and Slack for exporting emails and alerts  

---

## 📬 Email Management

- ✅ Query emails with Gmail search syntax  
- ✅ Get unread emails with count and preview  
- ✅ List archived emails  
- ✅ Restore email to inbox  
- ✅ Delete emails permanently from trash  
- ✅ Move email to Spam folder  
- ✅ Move email to Inbox  
- ✅ Mark email as read  
- ✅ Archive email (remove from inbox)  
- ✅ Batch archive multiple emails  
- ✅ Trash email  
- ✅ Move email to label folder by adding label + archiving  
- ✅ Batch move emails to label folder  
- 🚧 Create draft emails (Pending)  
- 🚧 Send emails with CC/BCC support (Pending)  
- 🚧 Reply to emails (Pending)  
- 🚧 Forward emails (Pending)  
- 🚧 Delete draft emails (Pending)  
- 🚧 List drafts (Pending)  
- 🚧 Improve attachment handling and display (WIP)

---

## 🔎 Search & Filters

- ✅ Use Gmail operators for email querying (from:, to:, subject:, has:attachment:, is:unread:, etc.)  
- 🚧 Fix has:attachment filter  
- 🚧 Improve date range search (after:, before:)  
- 🚧 Add size-based email search  
- 🚧 Implement incremental search inside opened emails (Vim-style /)  
- 🚧 Local fuzzy search for labels  
- 🚧 Support for complex and saved filters  
- 🚧 Add bookmarks and customizable filters  

---

## 🧠 AI Capabilities

- ✅ Email summarization (configurable language and prompt)  
- ✅ AI label assignment suggestions  
- 🚧 Experimental: generate reply drafts  
- 🚧 Advanced prompt configuration for summarization, replying, and labeling  
- 🚧 Streaming LLM responses in the UI  
- 🚧 Contextual AI action panel (summarize, label, reply)  

---

## 📁 Workflow Integrations

- 🚧 Export email or summaries to Obsidian (REST API/plugin, shortcuts, customizable rules)  
- 🚧 Manual and automatic sending to Slack via Webhooks (critical emails, summaries)  
- 🚧 Create tasks or events in external managers (Google Calendar, Notion, etc.)  
- 🚧 Calendar invitation detection and handling (future phase)  

---

## 🎨 UX and Personalization

- ✅ Responsive layouts (wide, medium, narrow, mobile)  
- ✅ Fullscreen toggle & switching focus between sections  
- ✅ k9s-style command panel with autocomplete and suggestions  
- ✅ Keyboard shortcut configuration via JSON  
- 🚧 Implement undo/redo for destructive actions  
- 🚧 Add internal logs panel and troubleshooting tools  
- 🚧 Improve accessibility: keyboard-only navigation, screen reader hints  

---

## ⚡ Caching & Telemetry (Local / Offline)

- 🚧 Local caching of emails and attachments (configurable)  
- 🚧 Efficient syncing with Gmail (partial offline mode)  
- 🚧 Local indexing for fast searches  
- 🚧 Privacy-first local telemetry: usage, shortcuts, timings, errors  
- 🚧 Telemetry viewer & productivity dashboard inside TUI  
- 🚧 Easy export/reset of telemetry data, no remote upload  

---

## 🔧 Testing & Quality

- ✅ Unit tests including Gmail API mocks  
- 🚧 Tests for complex functionalities (shortcuts, filters, AI)  
- 🚧 CI pipeline for continuous quality assurance  

---

## 🔌 Extensibility & Plugins

- 🚧 Basic plugin system with interface and management  
- 🚧 Example plugins for Obsidian and Slack integration  
- 🚧 Configurable plugin shortcuts and custom actions  

---

## 📊 (Optional) Personal Productivity Metrics

- Usage statistics (emails processed, AI calls, response times, etc.)  
- Simple TUI graphs/reports for weekly/monthly review  

---

**Notes**
- Dynamic prioritization according to user feedback  
- Feature modules can be enabled/disabled without recompilation  
- Document all key features with usage examples  
- Keyboard-only workflows prioritized  

---
