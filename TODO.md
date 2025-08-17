# TODO List - GizTUI Project

## 🎯 Roadmap

## Priorities
- [ ] Improve status bar experience
- [ ] Improve legend
- [ ] Text search `/` inside the email body
- [ ] Think about sending items to Obsidian and Slack
- [ ] Open links (design UX)
- [ ] Theme configuration system
- [ ] Plugin system: extensible architecture to add custom functionality without touching the core
- [ ] Create equivalent command for shortcuts: prompts
- [ ] UI for creating new prompt templates
- [ ] Group llm configuration under the same object in config
- [ ] Move configuration file templates to files.
- [ ] when using :slack command the focus doesn't go to the send to slack widget.
- [x] Send slack messages in bulk
- [ ] Add comment to the template in Slack
- [ ] Usage of the ErrorHandler in the operations
- [x] Treat several emails in batch with one prompt.
- [x] Review execution parameters, there are some duplication with llm and ollama.
- [ ] Review look and feel of the folders, scope selection in the advance search, the page doesn't update well and leaves some orphan letters as you move up and down through the options.
- [x] Fix AI SUmmary, Prompt Application (Single and bulk) issue with Escape. (it hangs when pressing Esc)
- [x] Add 'v' key as alternative to 'b' for entering bulk mode (Vim-style visual mode)
- [x] Investigate behavior for self-emailed messages
- [x] After loading messages, auto-select and render the first one
- [x] Stream LLM output instead of full response

## 🆕 New Priority Items (August 2025)
- [ ] **Gmail filters support** - Add native Gmail filters/rules support within the TUI
- [ ] **Save searches** - Implement bookmark/save functionality for search queries
- [ ] **Configure label colors** - Allow users to configure custom colors for Gmail labels
- [ ] **Enhanced bulk keyboard shortcuts** - Implement advanced bulk operations like `d5d` (delete next 5 messages), `a3a` (archive next 3), etc.
- [ ] **Update README** - Parts of README are outdated and need refresh
- [ ] **Change configuration directory** - Migrate from `~/.config/gmail-tui/` to `~/.config/giztui/` to reflect current tool name
- [ ] **Command parity with shortcuts** - Ensure every keyboard shortcut has an equivalent command (`:` mode)

## 🔍 **Search & Filter Enhancements**
- [ ] **Fix has:attachment filter** - Resolve issues with attachment-based email filtering
- [ ] **Improve date range search** - Enhanced date filtering with after:/before: operators
- [ ] **Add size-based email search** - Search emails by size (>1MB, <500KB, etc.)
- [ ] **Local indexing for fast searches** - Build local search index for performance improvements
- [ ] **Complex saved filters** - Support for advanced Gmail filter combinations and bookmarks

## 🎨 **UX & Accessibility Improvements**
- [ ] **Undo/redo for destructive actions** - Allow users to undo archive, delete, move operations
- [ ] **Internal logs panel** - Add debugging/troubleshooting tools within TUI
- [ ] **Accessibility improvements** - Keyboard-only navigation enhancements and screen reader support
- [ ] **Local caching system** - Configurable local caching of emails and attachments for offline access
- [ ] **Efficient Gmail syncing** - Partial offline mode with smart sync optimization

## 📊 **Analytics & Telemetry**
- [ ] **Privacy-first local telemetry** - Track usage, shortcuts, timings, and errors locally only
- [ ] **Built-in analytics dashboard** - TUI-based productivity metrics and usage statistics
- [ ] **Telemetry export/reset** - Easy data export and privacy controls (no remote upload)
- [ ] **Personal productivity reports** - Simple graphs for weekly/monthly email processing review

## 🔧 **Development & Quality**
- [ ] **Enhanced testing coverage** - Tests for complex functionalities (shortcuts, filters, AI)
- [ ] **CI pipeline implementation** - Continuous integration for quality assurance
- [ ] **Plugin example implementations** - Reference plugins for Obsidian and Slack integration
- [ ] **Configurable plugin shortcuts** - Custom actions and keyboard shortcuts for plugins
- Contextual menu for messages actions: at this moment we can operate over th emessage with several things: Labels, archive, delete, apply prompt, do summary... maybe we want a contextual menu.

### Email Management
- [ ] **Get unread emails** - List unread emails with count and preview
- [ ] **List archived emails** - Show archived emails
- [ ] **Restore email to inbox** - Move archived emails back to inbox
- [ ] **Delete email permanently** - Permanently delete emails from trash
- [ ] **Open email in browser** - Given a email open it in the browser
- [ ] **Move email to Spam** - Move to Spam
- [ ] **Move email to Inbox** - Move to Inbox
- [ ] **Search by date** - Search by date
- [x] **Query emails** - Search and query emails with Gmail search syntax
- [x] **Mark email as read** - Mark individual emails as read
- [x] **Archive email** - Move emails to archive (remove from inbox)
- [x] **Batch archive emails** - Archive multiple emails at once
- [x] **Trash email** - Move emails to trash
- [x] **Move email to folder** - Add a label and archive the email
- [x] **Batch move email to folder** - Add a label and archive multiple emails

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

### Labels and Organization
- [ ] **Labels assignation rules engine** - Define rules to suggest labels (similar to filters in gmail)
- [x] **Create label** - Create new custom labels with visibility options
- [x] **Delete label** - Delete custom labels
- [x] **List labels** - Show all available Gmail labels
- [x] **Apply label** - Apply labels to emails
- [x] **Remove label** - Remove labels from emails
- [x] **Contextual labels panel** - Side panel with quick toggle and live refresh
- [x] **Browse all labels with search** - Full picker with incremental filter and ESC back
- [x] **Local Search label** - Fuzzy search labels (server-side is done; local fuzzy TBD)

### Calendar Integration
- [ ] **List calendars** - Show all available calendars
- [ ] **Get calendar events** - Retrieve events from specific calendars with time range
- [ ] **Create calendar event** - Create new calendar events with Google Meet
- [ ] **Update calendar event** - Modify existing calendar events
- [ ] **Delete calendar event** - Remove calendar events
- [x] **Accept Calendar invitations**

## 🎨 AI Capabilities
- [ ] **Reply draft suggestion** - Given a email provides a draft of the reply
- [x] **Email summaritzation** - Creates a summary of the email 
- [x] **Label assignation suggestion** - Given a email provides the label selection
- [x] **Bulk prompt processing** - Apply prompts to multiple emails simultaneously for consolidated analysis


## 🎨 UX Improvements

### Command System Enhancements
- [ ] **Command history search** - Search through command history
- [ ] **Command aliases** - Support custom command aliases
- [ ] **Command help** - Show help for specific commands
- [ ] **Command validation** - Validate commands before execution
- [ ] **Configurable key bindings** - Be able to setup in the configuration file your key bindings
- [x] **Command autocompletion** - Auto-complete commands as you type (e.g., :la → labels)
- [x] **Command bar border** - Add visual border to command bar for better UX
- [x] **Command bar focus** - Automatically focus command bar when activated
- [x] **Command suggestions** - Show suggestions in brackets when typing commands

### Interface Improvements
- [ ] **Keyboard shortcuts display** - Show available shortcuts in status bar
- [ ] **Progress indicators** - Show loading progress for long operations
- [ ] **Error handling** - Better error messages and recovery
- [ ] **Confirmation dialogs** - Confirm destructive actions
- [ ] **Undo functionality** - Undo last action
- [ ] **Search highlighting** - Highlight search terms in results
- [ ] **Message threading** - Show message threads and conversations
- [ ] **Configuration for labels adding icons** Icons for each Label.
- [x] **Vertical layout** - Stacked layout with list, content, commands, and status
- [x] **Message headers (From, To, Subject, Date, Labels)** - Use different text color for these texts
- [x] **Visualization of important labels as colors in the message lists** Each label should have a a color

### Navigation Enhancements
- [ ] **Quick navigation** - Jump to specific messages or labels
- [ ] **Bookmarks** - Bookmark important messages
- [ ] **Recent messages** - Quick access to recently viewed messages
- [ ] **Message filtering** - Filter messages by various criteria
- [ ] **Sort options** - Sort messages by date, sender, subject, etc.
- [x] **Bulk operations** - Select multiple messages for bulk actions
- [x] **Keyboard navigation** - Tab cycles panes; arrows respect focused pane

### Message rendering
- [x] ~~Markdown rendering for HTML messages~~.(Substituted by a process)
- [x] Remove hyperlinks and add them at the end as references.
- [x] Be able to render the original raw message. (It can be saved with W)

### 🏗️ **Architectural Refactoring (December 2024)**
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

## 🎨 Theme System

### Theme Functionality
- [ ] **Review theme loading** - Verify theme files are loaded correctly
- [ ] **Test theme switching** - Implement and test theme switching functionality
- [ ] **Validate theme format** - Ensure YAML theme files are properly parsed
- [ ] **Theme preview** - Add theme preview functionality in demo
- [ ] **Custom theme creation** - Allow users to create custom themes
- [ ] **Theme validation** - Validate theme structure and required fields

### Theme Files
- [ ] **Review gmail-dark.yaml** - Check dark theme implementation
- [ ] **Review gmail-light.yaml** - Check light theme implementation
- [ ] **Review custom-example.yaml** - Verify example theme structure
- [ ] **Theme documentation** - Document theme format and options

## 📖 Help System

### Help Content
- [ ] **Review help content** - Check existing help documentation
- [ ] **Keyboard shortcuts** - Document all keyboard shortcuts
- [ ] **Command reference** - Create comprehensive command reference
- [ ] **Troubleshooting guide** - Add troubleshooting section
- [ ] **FAQ section** - Create frequently asked questions

### Help Interface
- [ ] **Help navigation** - Implement help navigation within TUI
- [ ] **Contextual help** - Show context-specific help
- [ ] **Help search** - Add search functionality to help system
- [ ] **Help formatting** - Ensure help text is properly formatted
- [ ] **Mantain a in-app log system** - A list of performed actions

## 🧪 Testing

### Unit Tests
- [ ] **Config package tests** - Test configuration loading and validation
- [ ] **Gmail client tests** - Test Gmail API client functionality
- [ ] **TUI component tests** - Test terminal UI components
- [ ] **Theme system tests** - Test theme loading and application
- [ ] **OAuth tests** - Test authentication flow

### Integration Tests
- [ ] **End-to-end tests** - Test complete user workflows
- [ ] **API integration tests** - Test Gmail API integration
- [ ] **Theme integration tests** - Test theme system integration
- [ ] **Error handling tests** - Test error scenarios and recovery

### Test Infrastructure
- [ ] **Test setup** - Configure testing environment
- [ ] **Mock Gmail API** - Create mock Gmail API for testing
- [ ] **Test data** - Prepare test data and fixtures
- [ ] **CI/CD integration** - Integrate tests with CI/CD pipeline

## 🔧 Infrastructure & Polish

### Code Quality
- [ ] **Code review** - Review existing code for improvements
- [ ] **Error handling** - Improve error handling throughout
- [ ] **Logging** - Implement comprehensive logging
- [ ] **Documentation** - Update code documentation
- [ ] **Performance optimization** - Optimize performance bottlenecks

### User Experience
- [ ] **Loading indicators** - Add loading indicators for long operations
- [ ] **Error messages** - Improve user-friendly error messages
- [ ] **Keyboard shortcuts** - Implement intuitive keyboard shortcuts
- [ ] **Accessibility** - Ensure accessibility compliance
- [ ] **Responsive design** - Handle terminal resizing gracefully

### Configuration
- [ ] **Configuration validation** - Validate configuration files
- [ ] **Default configuration** - Set up sensible defaults
- [ ] **Configuration documentation** - Document all configuration options
- [ ] **Hot reload** - Implement configuration hot reloading

## 🚀 Deployment & Distribution

### Build System
- [ ] **Cross-platform builds** - Support builds for different platforms
- [ ] **Release process** - Automate release process
- [ ] **Version management** - Implement proper versioning
- [ ] **Dependency management** - Keep dependencies updated

### Documentation
- [ ] **README updates** - Keep README current and comprehensive
- [ ] **Installation guide** - Create detailed installation instructions
- [ ] **User manual** - Create user manual
- [ ] **Developer guide** - Create developer documentation

---

## Notes
- Focus on core functionality first
- Test each feature thoroughly before moving to the next
- Keep user experience in mind throughout development
- Document as you go
- Regular code reviews and refactoring
- Ensure complete feature parity with MCP server reference
