# TODO List - Gmail TUI Project

## 🎯 Feature Parity with MCP Server

### Email Management
- [ ] **Query emails** - Search and query emails with Gmail search syntax
- [ ] **Get email by ID** - Retrieve specific email content and metadata
- [ ] **Bulk get emails** - Get multiple emails by their IDs
- [ ] **Get unread emails** - List unread emails with count and preview
- [x] **Mark email as read** - Mark individual emails as read
- [ ] **Archive email** - Move emails to archive (remove from inbox)
- [ ] **Batch archive emails** - Archive multiple emails at once
- [ ] **List archived emails** - Show archived emails
- [ ] **Restore email to inbox** - Move archived emails back to inbox
- [ ] **Trash email** - Move emails to trash
- [ ] **Delete email permanently** - Permanently delete emails from trash

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
- [ ] **List labels** - Show all available Gmail labels
- [ ] **Create label** - Create new custom labels with visibility options
- [ ] **Apply label** - Apply labels to emails
- [ ] **Remove label** - Remove labels from emails
- [ ] **Delete label** - Delete custom labels

### Calendar Integration
- [ ] **List calendars** - Show all available calendars
- [ ] **Get calendar events** - Retrieve events from specific calendars with time range
- [ ] **Create calendar event** - Create new calendar events with Google Meet
- [ ] **Update calendar event** - Modify existing calendar events
- [ ] **Delete calendar event** - Remove calendar events

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

## Priority Levels

### High Priority (P0)
- Email listing and basic navigation (query_emails, get_email_by_id)
- Theme system functionality
- Basic help system
- Core testing infrastructure

### Medium Priority (P1)
- Email composition and sending (create_draft, send_email, reply_email)
- Calendar integration (list_calendars, get_calendar_events)
- Advanced help features
- Performance optimization

### Low Priority (P2)
- Advanced features (batch operations, labels, attachments)
- Custom theme creation
- Advanced testing scenarios
- Documentation polish

---

## Notes
- Focus on core functionality first
- Test each feature thoroughly before moving to the next
- Keep user experience in mind throughout development
- Document as you go
- Regular code reviews and refactoring
- Ensure complete feature parity with MCP server reference
