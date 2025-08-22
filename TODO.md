# TODO List - GizTUI Project

## 🎯 TODO - Active Roadmap
- [x] when token expires it doesn't handle the request of a new one... It raises the following error:
./build/gmail-tui
2025/08/21 09:02:18 Could not initialize Gmail service: could not refresh token: oauth2: "invalid_grant" "Token has been expired or revoked."
make: *** [run] Error 1
- [x] the instructions to get the token contain color labels that don't work properly:
[bold]Authorization required[reset]
1. Open this link: [blue]https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=420480303874-me950drigguet0fakg0aobsm4o4616p4.apps.googleusercontent.com&redirect_uri=http%3A%2F%2Flocalhost%3A8080&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.readonly+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.send+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.modify+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.compose+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fcalendar.events&state=state-token[reset]
2. Grant access to the application
3. You will be redirected automatically
- [x] Remove el prefijo "VIM:" que aparece
- [x] Be able to hide the headers from the email dynamically
- [x] Fetch next 50 messages doesn't have any shortcut any more
- [x] Explore the same component as the Picker for the message content
- [x] Space for select is not in the shortcuts in config
- [x] sXs is not configurable for bulk operations in the config
- [ ] add a prompt to make an analysis of the inbox itself

## 📋 PENDING - What's Left to Do

### High Priority
- [x] Bulk select s2s is not configurable.
- [x] **Theme configuration system** - Implement customizable themes
- [ ] **Status bar experience** - Improve status bar functionality and UX
- [ ] **Legend improvements** - Enhance help/legend system
- [ ] **Folder/scope selection UX** - Fix advanced search page updates and orphan letter issues
- [ ] **Gmail filters support** - Add native Gmail filters/rules support within the TUI
- [ ] **Save searches** - Implement bookmark/save functionality for search queries
- [ ] **Configure label colors** - Allow users to configure custom colors for Gmail labels
- [ ] **Review database location**, now it is under $CONFIG/cache but i think it should be into a more generic location maybe on $CONFIG/db

### Medium Priority
- [ ] **Contextual menu for message actions** - Create context menu for Labels, Archive, Delete, Apply Prompt, Summary, etc.
- [x] **UI for creating new prompt templates** - Build interface for template creation

### Search & Filter Enhancements
- [ ] **Complex saved filters** - Support for advanced Gmail filter combinations and bookmarks
- [x] **Improve date range search** - Enhanced date filtering with after:/before: operators
- [ ] **Local indexing for fast searches** - Build local search index for performance improvements

### Email Management
- [ ] **Get unread emails** - List unread emails with count and preview
- [ ] **List archived emails** - Show archived emails
- [ ] **Restore email to inbox** - Move archived emails back to inbox
- [ ] **Delete email permanently** - Permanently delete emails from trash
- [x] **Open email in browser** - Given a email open it in the browser
- [ ] **Move email to Spam** - Move to Spam
- [ ] **Move email to Inbox** - Move to Inbox

### Email Composition
- [ ] **Create draft email** - Create new email drafts
- [ ] **Send email** - Send emails directly via Gmail (with CC/BCC support)
- [ ] **Reply to email** - Reply to existing email threads
- [ ] **Forward email** - Forward emails to other recipients
- [ ] **Delete draft** - Remove draft emails
- [ ] **List drafts** - Show all draft emails

### Attachments
- [ ] **Get attachment** - Download email attachments
- [ ] **Bulk save attachments** - Save multiple attachments at once
- [ ] **List attachments** - Show attachments for specific emails

### Calendar Integration
- [ ] **List calendars** - Show all available calendars
- [ ] **Get calendar events** - Retrieve events from specific calendars with time range
- [ ] **Create calendar event** - Create new calendar events with Google Meet
- [ ] **Update calendar event** - Modify existing calendar events
- [ ] **Delete calendar event** - Remove calendar events

### AI Capabilities
- [ ] **Reply draft suggestion** - Given a email provides a draft of the reply

### Command System Enhancements
- [ ] **Command history search** - Search through command history
- [ ] **Command aliases** - Support custom command aliases
- [ ] **Command help** - Show help for specific commands
- [ ] **Command validation** - Validate commands before execution
- [x] **Configurable key bindings** - Be able to setup in the configuration file your key bindings

### Interface Improvements
- [ ] **Keyboard shortcuts display** - Show available shortcuts in a legend or a help page or similar
- [ ] **Progress indicators** - Show loading progress for long operations
- [ ] **Error handling** - Better error messages and recovery
- [ ] **Confirmation dialogs** - Confirm destructive actions
- [ ] **Undo functionality** - Undo last action
- [ ] **Search highlighting** - Highlight search terms in results
- [ ] **Message threading** - Show message threads and conversations
- [ ] **Configuration for labels adding icons** Icons for each Label.
- [ ] **Undo/redo for destructive actions** - Allow users to undo archive, delete, move operations
- [ ] **Internal logs panel** - Add debugging/troubleshooting tools within TUI
- [ ] **Accessibility improvements** - Keyboard-only navigation enhancements and screen reader support
- [ ] **Local caching system** - Configurable local caching of emails and attachments for offline access
- [ ] **Efficient Gmail syncing** - Partial offline mode with smart sync optimization

### Navigation Enhancements
- [ ] **Sort options** - Sort messages by date, sender, subject, etc.
- [ ] **Bookmarks** - Bookmark important messages
- [ ] **Recent messages** - Quick access to recently viewed messages
- [ ] **Message filtering** - Filter messages by various criteria

### Theme System
- [ ] **Review theme loading** - Verify theme files are loaded correctly
- [ ] **Test theme switching** - Implement and test theme switching functionality
- [ ] **Validate theme format** - Ensure YAML theme files are properly parsed
- [ ] **Theme preview** - Add theme preview functionality in demo
- [ ] **Custom theme creation** - Allow users to create custom themes
- [ ] **Theme validation** - Validate theme structure and required fields
- [ ] **Review gmail-dark.yaml** - Check dark theme implementation
- [ ] **Review gmail-light.yaml** - Check light theme implementation
- [ ] **Review custom-example.yaml** - Verify example theme structure
- [ ] **Theme documentation** - Document theme format and options

### Help System
- [ ] **Review help content** - Check existing help documentation
- [ ] **Keyboard shortcuts** - Document all keyboard shortcuts
- [ ] **Command reference** - Create comprehensive command reference
- [ ] **Troubleshooting guide** - Add troubleshooting section
- [ ] **FAQ section** - Create frequently asked questions
- [ ] **Help navigation** - Implement help navigation within TUI
- [ ] **Contextual help** - Show context-specific help
- [ ] **Help search** - Add search functionality to help system
- [ ] **Help formatting** - Ensure help text is properly formatted
- [ ] **Mantain a in-app log system** - A list of performed actions

### Testing
- [ ] **Config package tests** - Test configuration loading and validation
- [ ] **Gmail client tests** - Test Gmail API client functionality
- [ ] **TUI component tests** - Test terminal UI components
- [ ] **Theme system tests** - Test theme loading and application
- [ ] **OAuth tests** - Test authentication flow
- [ ] **End-to-end tests** - Test complete user workflows
- [ ] **API integration tests** - Test Gmail API integration
- [ ] **Theme integration tests** - Test theme system integration
- [ ] **Error handling tests** - Test error scenarios and recovery
- [ ] **Test setup** - Configure testing environment
- [ ] **Mock Gmail API** - Create mock Gmail API for testing
- [ ] **Test data** - Prepare test data and fixtures
- [ ] **CI/CD integration** - Integrate tests with CI/CD pipeline

### Infrastructure & Polish
- [ ] **Code review** - Review existing code for improvements
- [ ] **Error handling** - Improve error handling throughout
- [ ] **Logging** - Implement comprehensive logging
- [ ] **Documentation** - Update code documentation
- [ ] **Performance optimization** - Optimize performance bottlenecks
- [ ] **Loading indicators** - Add loading indicators for long operations
- [ ] **Error messages** - Improve user-friendly error messages
- [ ] **Keyboard shortcuts** - Implement intuitive keyboard shortcuts
- [ ] **Accessibility** - Ensure accessibility compliance
- [ ] **Responsive design** - Handle terminal resizing gracefully
- [ ] **Configuration validation** - Validate configuration files
- [ ] **Default configuration** - Set up sensible defaults
- [ ] **Configuration documentation** - Document all configuration options
- [ ] **Hot reload** - Implement configuration hot reloading

### 📊 Analytics & Telemetry
- [ ] **Privacy-first local telemetry** - Track usage, shortcuts, timings, and errors locally only
- [ ] **Built-in analytics dashboard** - TUI-based productivity metrics and usage statistics
- [ ] **Telemetry export/reset** - Easy data export and privacy controls (no remote upload)
- [ ] **Personal productivity reports** - Simple graphs for weekly/monthly email processing review

### 🔧 Development & Quality
- [ ] **Enhanced testing coverage** - Tests for complex functionalities (shortcuts, filters, AI)
- [ ] **CI pipeline implementation** - Continuous integration for quality assurance
- [ ] **Configurable plugin shortcuts** - Custom actions and keyboard shortcuts for plugins
- [ ] **Plugin system**: extensible architecture to add custom functionality without touching the core

### Deployment & Distribution
- [ ] **Cross-platform builds** - Support builds for different platforms
- [ ] **Release process** - Automate release process
- [ ] **Version management** - Implement proper versioning
- [ ] **Dependency management** - Keep dependencies updated
- [ ] **README updates** - Keep README current and comprehensive
- [ ] **Installation guide** - Create detailed installation instructions
- [ ] **User manual** - Create user manual
- [ ] **Developer guide** - Create developer documentation

---

## 🚧 Issues & Bugs

### Known Issues
- [x] Something is working weird, when I move emails, they are removed from the list but still I see 50 messages, it looks like the last X emails are repeated in the end of the list when X is the number of moved emails..

---

## ✅ DONE - Completed Features

### Core Functionality
- [x] Create equivalent command for shortcuts: prompts
- [x] Make timeout configurable
- [x] Fix p prompt streaming - LLM response now streams properly
- [x] **Enhanced message content navigation** - Implemented better ways to browse message content beyond line-by-line navigation
- [x] **Text search functionality** - Added `/` search inside email body with navigation
- [x] **Calendar invitation enhancements** - Added date/time summary when showing Accept/Decline options
- [x] **Message header wrapping** - Fixed long CC/BCC headers that didn't wrap properly
- [x] **Enhanced bulk keyboard shortcuts** - Implemented advanced bulk operations like `d5d` (delete next 5), `a3a` (archive next 3), etc.
- [x] **Link opening functionality** - Designed and implemented UX for opening links from emails
- [x] **Slack integration improvements** - Added comment support to Slack templates
- [x] **Slack command focus fix** - Fixed focus management when using :slack command
- [x] **Configuration improvements** - Grouped LLM configuration under single object and moved templates to files

### Search & Filter Features
- [x] **Size-based email search** - Added search by email size (>1MB, <500KB, etc.)
- [x] **Attachment filter fix** - Resolved issues with has:attachment filter
- [x] **Search by date** - Search by date

### Email Management (Completed)
- [x] **Query emails** - Search and query emails with Gmail search syntax
- [x] **Mark email as read** - Mark individual emails as read
- [x] **Archive email** - Move emails to archive (remove from inbox)
- [x] **Batch archive emails** - Archive multiple emails at once
- [x] **Trash email** - Move emails to trash
- [x] **Move email to folder** - Add a label and archive the email
- [x] **Batch move email to folder** - Add a label and archive multiple emails

### Labels and Organization (Completed)
- [x] **Create label** - Create new custom labels with visibility options
- [x] **Delete label** - Delete custom labels
- [x] **List labels** - Show all available Gmail labels
- [x] **Apply label** - Apply labels to emails
- [x] **Remove label** - Remove labels from emails
- [x] **Contextual labels panel** - Side panel with quick toggle and live refresh
- [x] **Browse all labels with search** - Full picker with incremental filter and ESC back
- [x] **Local Search label** - Fuzzy search labels (server-side is done; local fuzzy TBD)

### Calendar Integration (Completed)
- [x] **Accept Calendar invitations**

### AI Capabilities (Completed)
- [x] **Email summaritzation** - Creates a summary of the email 
- [x] **Label assignation suggestion** - Given a email provides the label selection
- [x] **Bulk prompt processing** - Apply prompts to multiple emails simultaneously for consolidated analysis

### Command System Enhancements (Completed)
- [x] **Command autocompletion** - Auto-complete commands as you type (e.g., :la → labels)
- [x] **Command bar border** - Add visual border to command bar for better UX
- [x] **Command bar focus** - Automatically focus command bar when activated
- [x] **Command suggestions** - Show suggestions in brackets when typing commands
- [x] **Command parity with shortcuts** - Every keyboard shortcut has an equivalent command (`:` mode)

### Infrastructure & Polish (Completed)
- [x] **Change configuration directory** - Migrate from `~/.config/gmail-tui/` to `~/.config/giztui/` to reflect current tool name

### Interface Improvements (Completed)
- [x] **Vertical layout** - Stacked layout with list, content, commands, and status
- [x] **Message headers (From, To, Subject, Date, Labels)** - Use different text color for these texts
- [x] **Visualization of important labels as colors in the message lists** Each label should have a a color

### Navigation Enhancements (Completed)
- [x] **Quick navigation** - Jump to specific messages or labels
- [x] **Bulk operations** - Select multiple messages for bulk actions
- [x] **Keyboard navigation** - Tab cycles panes; arrows respect focused pane

### Message Rendering (Completed)
- [x] ~~Markdown rendering for HTML messages~~.(Substituted by a process)
- [x] Remove hyperlinks and add them at the end as references.
- [x] Be able to render the original raw message. (It can be saved with W)

### Core Priorities (Completed)
- [x] Think about sending items to Obsidian and Slack
- [x] Send slack messages in bulk
- [x] Treat several emails in batch with one prompt.
- [x] Review execution parameters, there are some duplication with llm and ollama.
- [x] Fix AI SUmmary, Prompt Application (Single and bulk) issue with Escape. (it hangs when pressing Esc)
- [x] Fix unnecessary message list reload after move operations (August 2025)
- [x] Add 'v' key as alternative to 'b' for entering bulk mode (Vim-style visual mode)
- [x] Investigate behavior for self-emailed messages
- [x] After loading messages, auto-select and render the first one
- [x] Stream LLM output instead of full response
- [x] **Update README** - Parts of README are outdated and need refresh

### Plugin System (Completed)
- [x] **Plugin example implementations** - Reference plugins for Obsidian and Slack integration

### Bug Fixes (Completed)
- [x] **Message list duplication bug** - Fixed issue where moved emails were removed but count remained at 50, causing duplicate messages

### Architectural Refactoring (Completed - December 2024)
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

---

## Notes
- Focus on core functionality first
- Test each feature thoroughly before moving to the next
- Keep user experience in mind throughout development
- Document as you go
- Regular code reviews and refactoring
- Ensure complete feature parity with MCP server reference