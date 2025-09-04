# TODO List - GizTUI Project

## 📋 Detected ISSUES - Short term to fix.
- [ ] When an UNDO is performed, focus is lost (it should return to the message list)
- [ ] Standardize operations across widgets: editing and deleting drafts and saved queries from the picker
- [x] When I press "gg" the cursor go to the top but doesn't select the message.
- [x] there's an issue when numbering :69 goes to :68, etc...
- [x] change the advance search date within icon for a safer one

## 📋 PENDING - What's Left to Do

### High Priority
- [ ] Add a prompt to make an analysis of the inbox itself
- [ ] Chat with an email
- [ ] **Gmail filters support** - Add native Gmail filters/rules support within the TUI
- [x] **Status bar experience** - Improve status bar functionality and UX
- [ ] **Configure label colors** - Allow users to configure custom colors for Gmail labels
- [ ] **Configure headers view** - config file picks which columns are hidden.
- [x] **Brush-up stats** - Revamp this feature
- [ ] **Review database location**, now it is under $CONFIG/cache but i think it should be into a more generic location maybe on $CONFIG/db
- [x] **Review makefile to reflect giztui**
- [ ] **Contextual menu for message actions** - Create context menu for Labels, Archive, Delete, Apply Prompt, Summary, etc.

### Search & Filter Enhancements
- [ ] **Complex saved filters** - Support for advanced Gmail filter combinations and bookmarks
- [ ] **Local indexing for fast searches** - Build local search index for performance improvements

### Saved Queries - Future Enhancements
- [ ] **Query categories and filtering** - Add category-based organization for saved queries
- [ ] **Edit saved queries functionality** - Allow editing of existing saved query names and content
- [ ] **Advanced query validation** - Enhanced validation for query names and complex search expressions

### Email Management
- [ ] **Delete email permanently** - Permanently delete emails from trash

### Email Composition - Advanced Features
- [ ] **AI-powered reply generation** - Integrate with existing AIService for smart reply suggestions
- [ ] **Email template system** - Create reusable email templates with picker UI and variable substitution
- [ ] **Recipient autocomplete** - Smart recipient suggestions from Gmail contacts and email history
- [ ] **Template variable substitution** - Dynamic template variables ({{name}}, {{date}}, {{subject}})
- [ ] **Tone/style AI suggestions** - AI assistance for professional, friendly, brief writing styles
- [ ] **Grammar and spell-check integration** - Basic text improvement and validation suggestions
- [ ] **Bulk reply operations** - Reply to multiple selected messages simultaneously  
- [ ] **Bulk forward operations** - Forward multiple messages as combined or separate forwards
- [ ] **Advanced composition configuration** - Auto-save intervals, default signatures, reply behavior settings
- [ ] **Attachment handling in composition** - File selection, drag-and-drop, attachment preview
- [ ] **Comprehensive composition testing** - Unit tests, integration tests, and UI component tests

### Attachments
- [ ] **Advanced attachment search filters** - `type:image`, `type:pdf`, `size:>1mb` filtering
- [ ] **Bulk attachment operations** - Multi-select and bulk download attachments
- [ ] **Attachment preview support** - Image thumbnails, text preview, PDF first-page preview
- [ ] **Obsidian attachment integration** - Send attachments to Obsidian with note creation
- [ ] **Slack attachment integration** - Send attachments to Slack with message creation
- [ ] **Enhanced attachment metadata** - Creation dates, detailed MIME info, compression ratios
- [ ] **Download queue management** - Progress bars, pause/resume, background downloads

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

### Interface Improvements
- [ ] **Error handling** - Better error messages and recovery
- [ ] **Confirmation dialogs** - Confirm destructive actions
- [ ] **Configuration for labels adding icons** Icons for each Label.
- [x] **Undo/redo for destructive actions** - Allow users to undo archive, delete, move operations
- [ ] **Internal logs panel** - Add debugging/troubleshooting tools within TUI
- [ ] **Accessibility improvements** - Keyboard-only navigation enhancements and screen reader support
- [ ] **Local caching system** - Configurable local caching of emails and attachments for offline access
- [ ] **Efficient Gmail syncing** - Partial offline mode with smart sync optimization

### Navigation Enhancements
- [ ] **Sort options** - Sort messages by date, sender, subject, etc.
- [ ] **Bookmarks** - Bookmark important messages
- [ ] **Recent messages** - Quick access to recently viewed messages
- [ ] **Message filtering** - Filter messages by various criteria 

### Help System
- [ ] **Troubleshooting guide** - Add troubleshooting section
- [ ] **FAQ section** - Create frequently asked questions
- [ ] **Help navigation** - Implement help navigation within TUI
- [ ] **Contextual help** - Show context-specific help
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
*No known issues at the moment*

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
- [x] **Undo functionality** - Undo last action ✅ 
- [x] **Message threading** - Show message threads and conversations
- [x] **Move email to Spam** - Move to Spam
- [x] **Move email to Inbox** - Move to Inbox
- [x] **Restore email to inbox** - Move archived emails back to inbox

### Email Composition - Core Features ✅
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

### Bug Fixes
- [x] Screen garbage
- [x] **Message list duplication bug** - Fixed issue where moved emails were removed but count remained at 50, causing duplicate messages
- [x] **Unnecessary message list reload** - Fixed reload after move operations (August 2025)
- [x] **Self-emailed messages behavior** - Investigated and resolved behavior issues
- [x] **Message auto-selection** - After loading messages, auto-select and render the first one
- [x] **README updates** - Updated outdated README sections

### Theme System
- [x] **Review theme loading** - Verify theme files are loaded correctly ✅ 
- [x] **Test theme switching** - Implement and test theme switching functionality ✅ 
- [x] **Validate theme format** - Ensure YAML theme files are properly parsed ✅ 
- [x] **Theme preview** - Add theme preview functionality in demo ✅ 
- [x] **Custom theme creation** - Allow users to create custom themes ✅ 
- [x] **Theme validation** - Validate theme structure and required fields ✅ 
- [x] **Review gmail-dark.yaml** - Check dark theme implementation ✅ 
- [x] **Review gmail-light.yaml** - Check light theme implementation ✅ 
- [x] **Review custom-example.yaml** - Verify example theme structure ✅ 
- [x] **Theme documentation** - Document theme format and options ✅

---

## Notes
- Focus on core functionality first
- Test each feature thoroughly before moving to the next
- Keep user experience in mind throughout development
- Document as you go
- Regular code reviews and refactoring
- Ensure complete feature parity with MCP server reference