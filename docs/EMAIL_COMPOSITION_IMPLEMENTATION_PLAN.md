# 📧 Email Composition Feature Implementation Plan

## Executive Summary

After thorough analysis of the GizTUI codebase, I've discovered that **email composition functionality is currently unimplemented** - all composition functions (`composeMessage`, `replySelected`, `generateReply`, `loadDrafts`) are placeholder stubs. However, the underlying service layer (`SendMessage`, `ReplyToMessage`) and architectural foundations are solid, creating a perfect greenfield opportunity to implement a comprehensive email composition system from scratch.

## Current State Analysis

### ✅ **What Exists & Works:**
- **Service Layer**: `EmailService.SendMessage()` and `ReplyToMessage()` fully implemented
- **Gmail Integration**: Underlying `gmailClient.SendMessage()` and `ReplyMessage()` working
- **Draft Support**: `MessageRepository.GetDrafts()` available
- **Key Bindings**: Compose, Reply, GenerateReply, Drafts shortcuts configured
- **Command System**: `:compose`, `:reply` commands exist
- **Architecture**: Service-first, ErrorHandler, theming, focus management all established

### ❌ **What Needs Implementation:**
- **Complete UI Layer**: All composition functions are unimplemented stubs
- **Composition Interface**: No composition panels, forms, or modals exist
- **Draft Management**: No draft creation, editing, or management UI
- **Recipient Management**: No to/cc/bcc handling with autocomplete
- **Rich Composition**: No subject/body editing with validation
- **Template System**: No email templates or signatures
- **AI Integration**: No AI-assisted composition features

## Proposed Implementation Architecture

### 1. **Service Layer Extensions (Service-First Pattern)**

#### **CompositionService Interface** (`internal/services/interfaces.go`)
```go
type CompositionService interface {
    // Composition lifecycle
    CreateComposition(ctx context.Context, type CompositionType, originalMessageID string) (*Composition, error)
    LoadDraftComposition(ctx context.Context, draftID string) (*Composition, error)
    SaveDraft(ctx context.Context, composition *Composition) (string, error)
    SendComposition(ctx context.Context, composition *Composition) error
    
    // Validation & processing
    ValidateComposition(composition *Composition) []ValidationError
    ProcessReply(ctx context.Context, originalMessageID string) (*ReplyContext, error)
    ProcessForward(ctx context.Context, originalMessageID string) (*ForwardContext, error)
    
    // Templates & suggestions
    GetTemplates(ctx context.Context, category string) ([]*EmailTemplate, error)
    ApplyTemplate(ctx context.Context, composition *Composition, templateID string) error
    GetRecipientSuggestions(ctx context.Context, query string) ([]Recipient, error)
}
```

#### **Composition Data Structures**
```go
type CompositionType string
const (
    CompositionTypeNew     CompositionType = "new"
    CompositionTypeReply   CompositionType = "reply"
    CompositionTypeReplyAll CompositionType = "reply_all"
    CompositionTypeForward CompositionType = "forward"
    CompositionTypeDraft   CompositionType = "draft"
)

type Composition struct {
    ID           string
    Type         CompositionType
    To           []Recipient
    CC           []Recipient
    BCC          []Recipient
    Subject      string
    Body         string
    Attachments  []Attachment
    OriginalID   string
    DraftID      string
    IsDraft      bool
    CreatedAt    time.Time
    ModifiedAt   time.Time
}

type Recipient struct {
    Email string `json:"email"`
    Name  string `json:"name,omitempty"`
}

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

type ReplyContext struct {
    OriginalMessage *gmail.Message
    Recipients      []Recipient
    Subject         string
    QuotedBody      string
    ThreadID        string
}

type EmailTemplate struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Category    string            `json:"category"`
    Subject     string            `json:"subject"`
    Body        string            `json:"body"`
    Variables   []string          `json:"variables"`
    Metadata    map[string]string `json:"metadata"`
}
```

### 2. **UI Component Architecture**

#### **Full-Screen Modal Composition Panel** 
- **Layout**: Full-screen modal overlay (not side panel like labels)
- **Components**: Multi-field form with tabbed focus management
- **Focus Flow**: To → CC → BCC → Subject → Body → Actions
- **Theme Integration**: `GetComponentColors("compose")` theming

#### **Component Structure**
```
CompositionPanel
├── HeaderSection (To/CC/BCC/Subject fields)
├── BodyEditor (Multi-line text editor)
├── AttachmentSection (File attachment management) 
├── ActionBar (Send/Save Draft/Cancel/AI Assist)
└── StatusBar (Validation errors, sending status)
```

#### **Focus Management Strategy**
- **New Focus State**: `currentFocus = "compose"` (dedicated focus state)
- **Field Navigation**: Tab/Shift+Tab cycling through form fields
- **ESC Behavior**: Direct synchronous cleanup (no QueueUpdateDraw)
- **Modal Restoration**: Return to previous focus after close

### 3. **Key Features & Workflows**

#### **Core Composition Workflows**
1. **New Composition** (`n` key, `:compose`)
   - Full-screen modal with empty form
   - Recipient autocomplete from contacts/history
   - Subject and body editing with validation
   
2. **Reply/Reply All** (`r` key, `:reply`)
   - Pre-populate recipients from original message
   - Quote original message in body with `>` prefix
   - Proper threading headers for Gmail
   
3. **Forward** (`:forward`)
   - Pre-populate subject with "Fwd:" prefix
   - Include original message as quoted text
   - Clear recipients for user selection
   
4. **Draft Management** (`:drafts`)
   - Side panel picker showing draft list (reuses `"labels"` focus)
   - Edit existing drafts in composition panel
   - Auto-save during composition

#### **Advanced Features**
1. **AI-Powered Assistance**
   - Generate reply button using existing AI service
   - Tone/style suggestions (professional, friendly, brief)
   - Grammar and spell-check assistance
   
2. **Template System**
   - Email templates for common responses
   - Variable substitution (`{{name}}`, `{{date}}`)
   - Template picker (side panel pattern)
   
3. **Smart Recipients**
   - Autocomplete from Gmail contacts and email history
   - Recent recipients prioritized
   - Validation of email format

### 4. **Integration Points**

#### **Keyboard Shortcuts (Using Existing Config)**
- **Compose**: `a.Keys.Compose` → `composeMessage(CompositionTypeNew)`
- **Reply**: `a.Keys.Reply` → `composeReply(false)` 
- **Reply All**: New shortcut → `composeReply(true)`
- **Forward**: New shortcut → `composeForward()`
- **Drafts**: `a.Keys.Drafts` → `showDraftsPicker()`

#### **Command Parity (Mandatory)**
- `:compose`, `:c` → New composition
- `:reply`, `:r` → Reply to current message  
- `:reply-all`, `:ra` → Reply all to current message
- `:forward`, `:f` → Forward current message
- `:drafts`, `:d` → Show drafts picker

#### **Bulk Mode Support**
- **Bulk Reply**: Reply to multiple selected messages (separate threads)
- **Bulk Forward**: Forward multiple messages as combined forward
- **Progress Indicators**: Use ErrorHandler for bulk operation progress

### 5. **Critical Architectural Compliance**

#### **Threading Patterns**
- **ESC Handling**: Direct synchronous cleanup, no `QueueUpdateDraw()`
- **Status Updates**: All user feedback via `GetErrorHandler()` async pattern  
- **AI Streaming**: Direct UI updates in streaming callbacks
- **Composition State**: Thread-safe access with proper mutexes

#### **Error Handling**
- **Validation Errors**: Real-time field validation with visual indicators
- **Send Failures**: Graceful retry with error details via ErrorHandler
- **Network Issues**: Proper timeout and retry logic with user feedback

#### **Focus & ESC Management**
```go
// ✅ Correct ESC cleanup pattern
func (a *App) hideCompositionPanel() {
    // Direct synchronous operations
    if modal, ok := a.views["compositionModal"]; ok {
        a.Pages.RemovePage("composition")
        a.currentFocus = a.previousFocus
        a.SetFocus(a.views[a.previousFocus])
        a.updateFocusIndicators(a.previousFocus)
    }
    a.compositionVisible = false
}
```

## Implementation Phases & Progress Tracking

### **Phase 1: Core Infrastructure** 
- [ ] **1.1** Create CompositionService interface in `internal/services/interfaces.go`
- [ ] **1.2** Implement CompositionService in `internal/services/composition_service.go`
- [ ] **1.3** Add composition data structures and validation logic
- [ ] **1.4** Add CompositionService to App struct in `internal/tui/app.go`
- [ ] **1.5** Initialize CompositionService in `initServices()` method
- [ ] **1.6** Update `GetServices()` method to return CompositionService
- [ ] **1.7** Add `compose` component theme colors to theme system

### **Phase 2: Basic Composition UI**
- [ ] **2.1** Create composition panel UI in `internal/tui/composition.go`
- [ ] **2.2** Implement full-screen modal layout with proper theming
- [ ] **2.3** Add To/CC/BCC/Subject input fields with focus management
- [ ] **2.4** Create multi-line body editor with proper text handling
- [ ] **2.5** Implement Tab/Shift+Tab focus cycling between fields
- [ ] **2.6** Add ESC key handling with synchronous cleanup
- [ ] **2.7** Add `currentFocus = "compose"` state management

### **Phase 3: Core Composition Workflows**
- [ ] **3.1** Implement `composeMessage()` for new email creation
- [ ] **3.2** Replace stub functions in `internal/tui/messages.go`
- [ ] **3.3** Add recipient validation and email format checking
- [ ] **3.4** Implement basic send functionality with EmailService integration
- [ ] **3.5** Add real-time validation with visual error indicators
- [ ] **3.6** Implement composition auto-save as draft
- [ ] **3.7** Add send confirmation and success feedback

### **Phase 4: Reply/Forward Workflows**
- [ ] **4.1** Implement `replySelected()` with message context processing
- [ ] **4.2** Add proper recipient extraction from original message
- [ ] **4.3** Implement message quoting with `>` prefix formatting
- [ ] **4.4** Add threading headers for Gmail compatibility
- [ ] **4.5** Implement reply-all functionality
- [ ] **4.6** Create forward workflow with "Fwd:" subject prefix
- [ ] **4.7** Add quoted text formatting for forwarded messages

### **Phase 5: Draft Management**
- [ ] **5.1** Implement `loadDrafts()` with draft picker UI
- [ ] **5.2** Create draft list using side panel picker pattern (reuse `"labels"` focus)
- [ ] **5.3** Add draft loading into composition panel for editing
- [ ] **5.4** Implement draft deletion and management operations
- [ ] **5.5** Add draft auto-save with periodic background saving
- [ ] **5.6** Add draft recovery after unexpected exits

### **Phase 6: Advanced Features**
- [ ] **6.1** Integrate AI-powered reply generation with existing AIService
- [ ] **6.2** Add recipient autocomplete from Gmail contacts and history
- [ ] **6.3** Implement email template system with template picker
- [ ] **6.4** Add variable substitution in templates (`{{name}}`, `{{date}}`)
- [ ] **6.5** Create tone/style suggestions for AI assistance
- [ ] **6.6** Add grammar and spell-check integration

### **Phase 7: Bulk Operations**
- [ ] **7.1** Implement bulk reply functionality for multiple messages
- [ ] **7.2** Add bulk forward with combined message forwarding
- [ ] **7.3** Add progress indicators using ErrorHandler patterns
- [ ] **7.4** Implement bulk send with error aggregation and reporting
- [ ] **7.5** Add bulk template application

### **Phase 8: Command System Integration**
- [ ] **8.1** Add composition commands to `internal/tui/commands.go`
- [ ] **8.2** Implement `:compose`, `:c` command with bulk mode support
- [ ] **8.3** Implement `:reply`, `:r` command with current message context
- [ ] **8.4** Add `:reply-all`, `:ra` command
- [ ] **8.5** Add `:forward`, `:f` command
- [ ] **8.6** Update command suggestions in `generateCommandSuggestion()`
- [ ] **8.7** Add command aliases to help system

### **Phase 9: Configuration & Customization**
- [ ] **9.1** Add composition-related configuration options
- [ ] **9.2** Add new keyboard shortcut configurations
- [ ] **9.3** Add reply-all and forward key bindings to KeyBindings struct
- [ ] **9.4** Add template configuration options
- [ ] **9.5** Add composition behavior settings (auto-save interval, etc.)

### **Phase 10: Testing & Documentation**
- [ ] **10.1** Create comprehensive test plan: `testplan_email_composition.md`
- [ ] **10.2** Write unit tests for CompositionService using test framework
- [ ] **10.3** Write component tests for composition UI with SimulationScreen
- [ ] **10.4** Add integration tests for complete composition workflows
- [ ] **10.5** Write visual regression tests for composition interface
- [ ] **10.6** Add performance tests for large message composition
- [ ] **10.7** Update help system with composition features
- [ ] **10.8** Update README.md with composition feature documentation

### **Phase 11: Polish & Production Readiness**
- [ ] **11.1** Add comprehensive error handling for all edge cases
- [ ] **11.2** Implement attachment handling and file selection
- [ ] **11.3** Add visual polish and accessibility improvements
- [ ] **11.4** Optimize performance for large compositions
- [ ] **11.5** Add internationalization support for composition interface
- [ ] **11.6** Run complete test suite and fix any remaining issues
- [ ] **11.7** Verify build success: `make build`
- [ ] **11.8** Run linting: `make fmt`, `make lint`

## Critical Success Criteria

### **Architectural Compliance**
- [ ] All business logic implemented in services layer
- [ ] No direct Gmail API calls in UI components
- [ ] All user feedback via `GetErrorHandler()` patterns
- [ ] Thread-safe state access using accessor methods
- [ ] Proper theming using `GetComponentColors("compose")`
- [ ] ESC key handling without `QueueUpdateDraw()` deadlocks

### **Feature Completeness**
- [ ] Complete email composition workflow (new, reply, forward)
- [ ] Draft management with auto-save and recovery
- [ ] Bulk composition operations support
- [ ] Command parity for all keyboard shortcuts
- [ ] AI integration for composition assistance
- [ ] Template system for common responses

### **User Experience**
- [ ] Intuitive keyboard navigation and focus management
- [ ] Real-time validation with helpful error messages
- [ ] Responsive UI that works across different terminal sizes
- [ ] Consistent theming across all composition interfaces
- [ ] Proper integration with existing GizTUI workflows

### **Quality Assurance**
- [ ] Comprehensive test coverage (unit, integration, visual)
- [ ] No regressions in existing functionality
- [ ] Proper error handling for all failure scenarios
- [ ] Performance optimization for large messages and bulk operations
- [ ] Complete documentation and help system updates

## Implementation Timeline

**Estimated Total Timeline: 8-12 weeks**

- **Week 1-2**: Phases 1-2 (Infrastructure + Basic UI)
- **Week 3-4**: Phases 3-4 (Core Workflows + Reply/Forward)
- **Week 5-6**: Phases 5-6 (Draft Management + Advanced Features)
- **Week 7-8**: Phases 7-8 (Bulk Operations + Commands)
- **Week 9-10**: Phases 9-10 (Configuration + Testing)
- **Week 11-12**: Phase 11 (Polish + Production Readiness)

## Risk Mitigation

### **Technical Risks**
- **Complex UI State Management**: Mitigated by following established focus patterns
- **Threading Deadlocks**: Mitigated by strict adherence to ESC handling patterns
- **Performance with Large Messages**: Mitigated by incremental UI updates and optimization

### **Integration Risks**
- **Gmail API Compatibility**: Mitigated by using existing EmailService patterns
- **Existing Feature Conflicts**: Mitigated by thorough regression testing
- **Theme System Integration**: Mitigated by following established theming patterns

## Success Metrics

- **Functional**: All composition workflows working end-to-end
- **Performance**: Sub-100ms UI response times for composition actions  
- **Quality**: >90% test coverage with zero critical bugs
- **Usability**: Intuitive workflows requiring minimal learning curve
- **Architectural**: Zero architectural violations or anti-patterns

---

This implementation plan provides a comprehensive roadmap for creating a world-class email composition system within GizTUI while maintaining architectural excellence and user experience standards.