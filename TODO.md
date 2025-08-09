# TODO List - Gmail TUI Project

## 🎯 Feature Parity with MCP Server

### Email Management
- [ ] **Query emails** - Search and query emails with Gmail search syntax
- [ ] **Get unread emails** - List unread emails with count and preview
- [x] **Mark email as read** - Mark individual emails as read
- [x] **Archive email** - Move emails to archive (remove from inbox)
- [ ] **Batch archive emails** - Archive multiple emails at once
- [ ] **List archived emails** - Show archived emails
- [ ] **Restore email to inbox** - Move archived emails back to inbox
- [x] **Trash email** - Move emails to trash
- [ ] **Delete email permanently** - Permanently delete emails from trash
- [x] **Move email to folder** - Add a label and archive the email
- [ ] **Batch move email to folder** - Add a label and archive multiple emails
- [ ] **Open email in browser** - Given a email open it in the browser



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
- [X] **List labels** - Show all available Gmail labels
- [X] **Create label** - Create new custom labels with visibility options
- [X] **Apply label** - Apply labels to emails
- [X] **Remove label** - Remove labels from emails
- [X] **Contextual labels panel** - Side panel with quick toggle and live refresh
- [X] **Browse all labels with search** - Full picker with incremental filter and ESC back
- [ ] **Delete label** - Delete custom labels
- [x] **Local Search label** - Fuzzy search labels (server-side is done; local fuzzy TBD)
- [ ] **Labels assignation rules engine** - Define rules to suggest labels (similar to filters in gmail)

### Calendar Integration
- [ ] **List calendars** - Show all available calendars
- [ ] **Get calendar events** - Retrieve events from specific calendars with time range
- [ ] **Create calendar event** - Create new calendar events with Google Meet
- [ ] **Update calendar event** - Modify existing calendar events
- [ ] **Delete calendar event** - Remove calendar events

## 🎨 AI Capabilities
- [x] **Email summaritzation** - Creates a summary of the email 
- [x] **Label assignation suggestion** - Given a email provides the label selection
- [ ] **Reply draft suggestion** - Given a email provides a draft of the reply


## 🎨 UX Improvements

### Command System Enhancements
- [x] **Command autocompletion** - Auto-complete commands as you type (e.g., :la → labels)
- [x] **Command bar border** - Add visual border to command bar for better UX
- [x] **Command bar focus** - Automatically focus command bar when activated
- [x] **Command suggestions** - Show suggestions in brackets when typing commands
- [ ] **Command history search** - Search through command history
- [ ] **Command aliases** - Support custom command aliases
- [ ] **Command help** - Show help for specific commands
- [ ] **Command validation** - Validate commands before execution

### Interface Improvements
- [x] **Vertical layout** - Stacked layout with list, content, commands, and status
- [ ] **Keyboard shortcuts display** - Show available shortcuts in status bar
- [ ] **Progress indicators** - Show loading progress for long operations
- [ ] **Error handling** - Better error messages and recovery
- [ ] **Confirmation dialogs** - Confirm destructive actions
- [ ] **Undo functionality** - Undo last action
- [ ] **Search highlighting** - Highlight search terms in results
- [ ] **Message threading** - Show message threads and conversations
- [ ] **Message headers (From, To, Subject, Date, Labels)** - Use different text color for these texts
- [ ] **Visualization of important labels as emojis in the message lists** Each label should have an emoji
- [ ] **Configuration for labels adding icons** Icons for each Label.

### Navigation Enhancements
- [ ] **Quick navigation** - Jump to specific messages or labels
- [ ] **Bookmarks** - Bookmark important messages
- [ ] **Recent messages** - Quick access to recently viewed messages
- [ ] **Message filtering** - Filter messages by various criteria
- [ ] **Sort options** - Sort messages by date, sender, subject, etc.
- [ ] **Bulk operations** - Select multiple messages for bulk actions
- [X] **Keyboard navigation** - Tab cycles panes; arrows respect focused pane

### Message rendering
- [ ] Markdown rendering for HTML messages.


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
