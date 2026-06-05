# TODO List - GizTUI Project

## 📋 Active Backlog

The active backlog has been migrated to **GitHub Issues**:
👉 **https://github.com/ajramos/giztui/issues**

This file is preserved as a historical record of completed work.
For the active roadmap, use the issues view with the `priority:*` and `area:*` labels.

---

## ✅ DONE - Completed Features

### Authentication & Configuration
- [x] **Token refresh handling** - Fixed OAuth2 token expiration and refresh issues
- [x] **Color label instructions** - Fixed authorization instructions with proper color formatting
- [x] **VIM prefix removal** - Removed unnecessary "VIM:" prefix from interface
- [x] **Configurable timeout** - Made timeout configurable for better user control
- [x] **Configuration directory migration** - Migrated from `~/.config/gmail-tui/` to `~/.config/giztui/`
- [x] **Configurable key bindings** - Implemented customizable keyboard shortcuts in configuration
- [x] **Theme configuration system** - Implemented customizable themes
- [x] **Configuration improvements** - Grouped LLM configuration under single object and moved templates to files
- [x] **Default configuration** - `DefaultConfig()` ships sensible defaults; users can boot without any config file
- [x] **Configuration documentation** - `docs/CONFIGURATION.md` documents all options

### Core Functionality
- [x] **Command parity with shortcuts** - Every keyboard shortcut has an equivalent command (`:` mode)
- [x] **Create equivalent command for shortcuts: prompts** - Implemented command mode for all shortcuts
- [x] **Enhanced message content navigation** - Better ways to browse message content beyond line-by-line navigation
- [x] **Text search functionality** - Added `/` search inside email body with navigation
- [x] **Calendar invitation enhancements** - Added date/time summary when showing Accept/Decline options
- [x] **Message header wrapping** - Fixed long CC/BCC headers that didn't wrap properly
- [x] **Enhanced bulk keyboard shortcuts** - Advanced bulk operations like `d5d` (delete next 5), `a3a` (archive next 3), etc.
- [x] **Link opening functionality** - Designed and implemented UX for opening links from emails
- [x] **Slack integration improvements** - Added comment support to Slack templates
- [x] **Slack command focus fix** - Fixed focus management when using :slack command
- [x] **UI for creating new prompt templates** - Built interface for template creation
- [x] **Bulk select s2s configuration** - Made bulk select operations configurable
- [x] **Save searches functionality** - Complete saved queries/bookmarks system with UI patterns, keyboard shortcuts, commands, and database persistence
- [x] **Multi-account** support for Email

### Email Management
- [x] **Query emails** - Search and query emails with Gmail search syntax
- [x] **Mark email as read** - Mark individual emails as read
- [x] **Archive email** - Move emails to archive (remove from inbox)
- [x] **Batch archive emails** - Archive multiple emails at once
- [x] **Trash email** - Move emails to trash
- [x] **Move email to folder** - Add a label and archive the email
- [x] **Batch move email to folder** - Add a label and archive multiple emails
- [x] **Open email in browser** - Open emails in browser for full viewing
- [x] **Dynamic header hiding** - Ability to hide email headers dynamically
- [x] **Fetch next 50 messages** - Restored shortcut for fetching more messages
- [x] **Picker component for message content** - Implemented picker-style navigation for message content
- [x] **Space for select configuration** - Made space key configurable for selection
- [x] **Bulk operations configuration** - Made sXs bulk operations configurable
- [x] **Get unread emails** - List unread emails with count and preview
- [x] **List archived emails** - Show archived emails
- [x] **Undo functionality** - Undo last action
- [x] **Undo/redo for destructive actions** - Allow users to undo archive, delete, move operations
- [x] **Message threading** - Show message threads and conversations
- [x] **Move email to Spam** - Move to Spam
- [x] **Move email to Inbox** - Move to Inbox
- [x] **Restore email to inbox** - Move archived emails back to inbox

### Email Composition - Core Features
- [x] **Complete composition UI** - Full-screen modal composition panel with proper theming and focus management
- [x] **Create new emails** - Compose new emails with To/CC/BCC/Subject/Body fields and validation
- [x] **Reply to emails** - Reply to existing email threads with proper context and quoted text
- [x] **Reply-all functionality** - Reply to all recipients with proper recipient extraction and exclusion
- [x] **Forward emails** - Forward emails with "Fwd:" prefix and proper quoted message formatting
- [x] **Draft management** - Create, edit, save, and delete email drafts with picker UI
- [x] **Send email functionality** - Send emails directly via Gmail with CC/BCC support and UTF-8 encoding
- [x] **Command system integration** - All composition commands (`:compose`, `:reply`, `:forward`, `:drafts`) with shortcuts
- [x] **Real-time validation** - Email format validation, recipient checking, and visual error indicators
- [x] **Auto-save drafts** - Automatic draft saving during composition
- [x] **Keyboard navigation** - Complete Tab/Shift+Tab focus cycling and ESC handling
- [x] **Message context processing** - Proper threading headers, recipient extraction, and Gmail compatibility
- [x] **Multi-line text editing** - Advanced EditableTextView with cursor visibility and scroll management

### Labels and Organization
- [x] **Create label** - Create new custom labels with visibility options
- [x] **Delete label** - Delete custom labels
- [x] **List labels** - Show all available Gmail labels
- [x] **Apply label** - Apply labels to emails
- [x] **Remove label** - Remove labels from emails
- [x] **Contextual labels panel** - Side panel with quick toggle and live refresh
- [x] **Browse all labels with search** - Full picker with incremental filter and ESC back
- [x] **Local Search label** - Fuzzy search labels (server-side is done; local fuzzy TBD)
- [x] **Visualization of important labels as colors** - Each label has a color in message lists
- [x] **Message headers styling** - Different text colors for From, To, Subject, Date, Labels

### Calendar Integration
- [x] **Accept Calendar invitations** - Full calendar invitation acceptance functionality

### AI Capabilities
- [x] **Email summarization** - Creates a summary of the email
- [x] **Label assignation suggestion** - Given an email provides label selection suggestions
- [x] **Bulk prompt processing** - Apply prompts to multiple emails simultaneously for consolidated analysis
- [x] **Fix AI Summary and Prompt Application** - Resolved Escape key hanging issues
- [x] **Stream LLM output** - Implemented streaming instead of full response waiting
- [x] **Prompt streaming fix** - LLM response now streams properly
- [x] **Reply draft suggestion** - `generateReply()` exposes `AIService.GenerateReply` via configurable shortcut

### Command System Enhancements
- [x] **Command autocompletion** - Auto-complete commands as you type (e.g., :la → labels)
- [x] **Command bar border** - Add visual border to command bar for better UX
- [x] **Command bar focus** - Automatically focus command bar when activated
- [x] **Command suggestions** - Show suggestions in brackets when typing commands

### Interface Improvements
- [x] **Vertical layout** - Stacked layout with list, content, commands, and status
- [x] **Keyboard navigation** - Tab cycles panes; arrows respect focused pane
- [x] **Quick navigation** - Jump to specific messages or labels
- [x] **Bulk operations** - Select multiple messages for bulk actions
- [x] **Vim-style visual mode** - Added 'v' key as alternative to 'b' for entering bulk mode
- [x] **Keyboard shortcuts display** - Show available shortcuts in a legend or a help page or similar
- [x] **Progress indicators** - Show loading progress for long operations
- [x] **Search highlighting** - Highlight search terms in results
- [x] **Status bar experience** - Improve status bar functionality and UX
- [x] **Brush-up stats** - Revamp this feature
- [x] **Responsive design** - Handle terminal resizing gracefully via responsive breakpoint system
- [x] **Loading indicators** - `ErrorHandler.ShowProgress` + status-bar feedback cover long operations
- [x] When I perform a local search with /term and press Enter, focus moves to the message list but its border is not highlighted. Also, I cannot return to the search widget using Tab. We could either 1) include it in the tab order to allow refining the search, or 2) close the widget immediately after launching the search and open a new one if needed.
- [x] Welcome screen doesn't pool the shortcuts from the customization.
- [x] When I'm in a panel other than Labels (e.g., "Drafts") and I maximize the screen, after repaint it shows the Labels panel instead of Drafts. This also happens when the initial 50 messages are loaded and i before they finished loading i open the drafts pickers, when the 50 messages finished loading the labels picker is opened (as if i have pressed the l)

### Message Rendering
- [x] **HTML message processing** - Substituted markdown rendering with improved HTML processing
- [x] **Hyperlink handling** - Remove hyperlinks and add them at the end as references
- [x] **Raw message rendering** - Ability to render original raw message (saved with W key)

### Search & Filter Features
- [x] **Size-based email search** - Search by email size (>1MB, <500KB, etc.)
- [x] **Attachment filter fix** - Resolved issues with has:attachment filter
- [x] **Search by date** - Search by date with enhanced date filtering
- [x] **Date range search improvements** - Enhanced date filtering with after:/before: operators
- [x] **Folder/scope selection UX** - Fix advanced search page updates and orphan letter issues
- [x] change the advance search date within icon for a safer one

### Plugin System
- [x] **Plugin example implementations** - Reference plugins for Obsidian and Slack integration
- [x] **Obsidian integration** - Send items to Obsidian for note-taking
- [x] **Slack integration** - Send slack messages in bulk
- [x] **Bulk email processing** - Treat several emails in batch with one prompt

### Help & Legend System
- [x] **Legend improvements** - Enhanced help/legend system
- [x] **Review help content** - Check existing help documentation
- [x] **Keyboard shortcuts** - Document all keyboard shortcuts
- [x] **Command reference** - Create comprehensive command reference
- [x] **Help search** - Add search functionality to help system
- [x] **Help formatting** - Ensure help text is properly formatted

### Attachments
- [x] **Get attachment** - Download email attachments *(Core functionality complete)*

### Testing & Quality
- [x] **Config package tests** - `internal/config/config_test.go` covers loading, defaults, validation helpers
- [x] **Gmail client tests** - `internal/gmail/client_test.go` + `client_parallel_test.go`
- [x] **TUI component tests** - `internal/tui/*_test.go` + `test/helpers/` harness
- [x] **Theme system tests** - `internal/services/theme_service_test.go`
- [x] **OAuth tests** - `pkg/auth/oauth_test.go`
- [x] **Error handling tests** - `internal/tui/error_handler_test.go` + service-level error paths
- [x] **Test setup** - `make test-mocks` + `TestHarness` framework + GitHub Actions runners
- [x] **Mock Gmail API** - generated mocks live in `internal/services/mocks/`
- [x] **Test data / fixtures** - test fixtures embedded in OAuth and config tests
- [x] **CI/CD integration** - `.github/workflows/ci-comprehensive.yml` runs on every push

### Infrastructure & Polish
- [x] **Service Layer Architecture** - Extracted business logic into dedicated services
  - EmailService for email operations
  - AIService for LLM integration
  - LabelService for label management
  - CacheService for SQLite caching
  - MessageRepository for data access abstraction
- [x] **Centralized Error Handling** - Consistent user feedback with ErrorHandler
- [x] **Thread-Safe State Management** - Mutex-protected accessor methods for app state
- [x] **Service Integration** - Services automatically initialized and injected into TUI
- [x] **Improved Code Organization** - Better separation of UI and business logic concerns
- [x] **Execution parameters review** - Resolved duplication between llm and ollama configurations
- [x] **Review log file name**, now it is under $CONFIG/gmail-tui.log it should be $CONFIG/giztui.log
- [x] **Review makefile to reflect giztui**
- [x] **Logging** - centralised logger (`internal/tui/logging.go`) used across the codebase
- [x] **Keyboard shortcuts (system)** - configurable keybindings system fully wired

### Deployment & Distribution
- [x] **Cross-platform builds** - 6-target build (linux/macos/windows × amd64/arm64) in Makefile + release workflow
- [x] **Release process** - automated GitHub Actions workflow + `docs/RELEASE_PROCEDURE.md` + `/release` slash command
- [x] **Version management** - `VERSION` + `internal/version/version.go` + `CHANGELOG.md` in sync via release procedure
- [x] **Installation guide** - `docs/GETTING_STARTED.md`
- [x] **User manual** - `docs/FEATURES.md` + `docs/KEYBOARD_SHORTCUTS.md`
- [x] **Developer guide** - `docs/ARCHITECTURE.md` + `CLAUDE.md`
- [x] **CI pipeline implementation** - `Comprehensive CI/CD Pipeline` workflow active

### Bug Fixes
- [x] Screen garbage
- [x] **Message list duplication bug** - Fixed issue where moved emails were removed but count remained at 50, causing duplicate messages
- [x] **Unnecessary message list reload** - Fixed reload after move operations (August 2025)
- [x] **Self-emailed messages behavior** - Investigated and resolved behavior issues
- [x] **Message auto-selection** - After loading messages, auto-select and render the first one
- [x] **README updates** - Updated outdated README sections
- [x] When an UNDO is performed, focus is lost (it should return to the message list)
- [x] When I press "gg" the cursor go to the top but doesn't select the message.
- [x] there's an issue when numbering :69 goes to :68, etc...
- [x] **Status bar emoji corruption** - replaced EAW-Ambiguous icons (`ℹ️`, `⚠️` with VS16) with unambiguously-wide circles (`🔵`, `🟡`) to stop status-bar text corruption after the icon prefix (commit `ef37b54`)
- [x] **Bulk select checkbox refresh** - fixed `*` (select all) so the `☑` glyph updates on every row even when the numbers column shifts flags from column 0 to column 1 (commit `2c17b62`)
- [x] **Help panic on `?` (#6)** - nil dereference in `generateHelpText` when `Config.Obsidian` was nil on fresh installs; introduced `Config.IsObsidianEnabled()` helper (commit `b9bd285`)

### Theme System
- [x] **Review theme loading** - Verify theme files are loaded correctly
- [x] **Test theme switching** - Implement and test theme switching functionality
- [x] **Validate theme format** - Ensure YAML theme files are properly parsed
- [x] **Theme preview** - Add theme preview functionality in demo
- [x] **Custom theme creation** - Allow users to create custom themes
- [x] **Theme validation** - Validate theme structure and required fields
- [x] **Review gmail-dark.yaml** - Check dark theme implementation
- [x] **Review gmail-light.yaml** - Check light theme implementation
- [x] **Review custom-example.yaml** - Verify example theme structure
- [x] **Theme documentation** - Document theme format and options

### Session 2026-06-03 / 2026-06-05 — release v1.2.4 + CI rescue + backlog migration
- [x] **CLAUDE.md refresh** - fixed stale `GetServices()` example (12 return values, not 5), added `make pre-commit-check` guidance, project layout section, scoped `make test` note (commit `6a97aa1`)
- [x] **Release v1.2.4 published** - GitHub Actions workflow produced 6 platform binaries + checksums; tagged and live at https://github.com/ajramos/giztui/releases/tag/v1.2.4
- [x] **Lint debt cleanup** - `make pre-commit-check` now passes with 0 issues. Annotated 9 gosec findings with justified `// #nosec` markers (G304 on validated DB paths, G204 on hardcoded per-OS open binaries, G117 on intentional OAuth token persistence, G101 on test fixture paths) and refactored 13+ `WriteString(fmt.Sprintf(...))` call-sites to `fmt.Fprintf(&builder, ...)` (commits `df8db83`, `873c35a`)
- [x] **GitHub Actions modernised to Node.js 24** - bumped checkout/setup-go/cache/upload-artifact/dependency-review/github-script/codecov/codeql/gh-release to current major versions; pinned `aquasecurity/trivy-action` from `@master` to `@v0.36.0` (commit `07fde36`)
- [x] **Go toolchain + deps security update** - bumped Go to 1.25.11 and `golang.org/x/net` to v0.55.0, resolving 9 govulncheck advisories (GO-2026-5025..5039) reported by the re-enabled CI pipeline (commit `ae26bc1`)
- [x] **codecov-action input rename** - `file` → `files` (singular input was silently ignored since codecov-action v4) (commit `c4bc854`)
- [x] **Comprehensive CI/CD Pipeline rescued** - workflow had been auto-disabled by GitHub due to inactivity since Feb 2026; re-enabled, all jobs green
- [x] **`.claude/` infrastructure committed** - added `cve-security-analyzer` agent and `release` slash command from a previously-unsynced work machine, with minor adjustments (`KillShell` → `KillBash` for consistency; `make test/lint/vet` → `make pre-commit-check`)
- [x] **TODO.md backlog migrated to GitHub Issues** - the active backlog (35 issues, #7-41) now lives in https://github.com/ajramos/giztui/issues with `area:*` and `priority:*` labels; vague/duplicate items dropped during triage; items already covered by code recognised as zombi and moved here

---

## Notes
- Focus on core functionality first
- Test each feature thoroughly before moving to the next
- Keep user experience in mind throughout development
- Document as you go
- Regular code reviews and refactoring
- Ensure complete feature parity with MCP server reference
