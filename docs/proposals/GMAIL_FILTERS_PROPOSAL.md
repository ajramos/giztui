# 🔧 **Gmail Filters Feature Proposal**

## **Overview**
Implement Gmail filters management directly in the TUI using Gmail's native Filter API, allowing users to create, edit, delete, and manage Gmail filters while maintaining the existing UI patterns and workflows.

## **Why Gmail Filter API Over Local Filtering**

### **Key Advantages:**
1. **Native Gmail Integration**: Filters work across all Gmail clients (web, mobile, TUI)
2. **Server-side Processing**: Gmail automatically applies filters to incoming emails
3. **No Synchronization Issues**: No need to maintain local state or sync between devices  
4. **Full Feature Parity**: Access to Gmail's complete filtering capabilities
5. **Persistence**: Filters remain active even when using other Gmail clients
6. **Performance**: Gmail's servers handle filtering, not your local application

## **Architecture Design**

### **1. Service Layer** (`internal/services/gmail_filter_service.go`)

```go
type GmailFilterService interface {
    // Gmail API Operations
    ListGmailFilters(ctx context.Context) ([]*GmailFilter, error)
    CreateGmailFilter(ctx context.Context, criteria *FilterCriteria, actions *FilterActions) (*GmailFilter, error)
    UpdateGmailFilter(ctx context.Context, filterID string, criteria *FilterCriteria, actions *FilterActions) (*GmailFilter, error)
    DeleteGmailFilter(ctx context.Context, filterID string) error
    
    // Local Management Operations
    SaveFilterTemplate(ctx context.Context, template *FilterTemplate) error
    ListFilterTemplates(ctx context.Context) ([]*FilterTemplate, error)
    DeleteFilterTemplate(ctx context.Context, templateID string) error
    TestFilterCriteria(ctx context.Context, criteria *FilterCriteria) ([]*gmail.Message, error)
    SyncFiltersFromGmail(ctx context.Context) error
}

type GmailFilter struct {
    ID       string                 `json:"id"`
    Criteria *gmail.FilterCriteria  `json:"criteria"`
    Action   *gmail.FilterAction    `json:"action"`
    Created  time.Time              `json:"created,omitempty"`
    IsActive bool                   `json:"isActive"`
}

type FilterCriteria struct {
    From            string
    To              string
    Subject         string
    Query           string
    HasAttachment   bool
    ExcludeChats    bool
    Size            int
    SizeComparison  string // "greater", "less"
}

type FilterActions struct {
    AddLabels       []string
    RemoveLabels    []string
    Forward         string
    MarkAsRead      bool
    MarkAsImportant bool
    NeverSpam       bool
    Archive         bool
    Delete          bool
}

type FilterTemplate struct {
    ID          string
    Name        string
    Description string
    Criteria    *FilterCriteria
    Actions     *FilterActions
    Category    string // "work", "newsletters", "security", "spam", etc.
    Created     time.Time
    UsageCount  int
}
```

### **2. Database Storage for Templates** (`internal/db/filter_store.go`)

```sql
CREATE TABLE IF NOT EXISTS filter_templates (
    id TEXT PRIMARY KEY,
    account_email TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    criteria_json TEXT NOT NULL,
    actions_json TEXT NOT NULL,
    category TEXT DEFAULT 'custom',
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Pre-populate with common templates
INSERT INTO filter_templates (id, account_email, name, description, criteria_json, actions_json, category) VALUES
('tpl_newsletters', '*', 'Newsletter Cleanup', 'Auto-delete newsletter emails', 
 '{"query":"from:(newsletter OR unsubscribe OR noreply) -is:important"}', 
 '{"delete":true,"markAsRead":true}', 'newsletters'),
('tpl_spam_cleanup', '*', 'Basic Spam Filter', 'Common spam patterns', 
 '{"query":"subject:(viagra OR lottery OR prince OR inheritance)"}', 
 '{"delete":true,"neverSpam":false}', 'spam');
```

### **3. Gmail API Integration** (`internal/gmail/client.go`)

```go
// Add to existing Gmail client
func (c *Client) ListFilters() ([]*gmail.Filter, error) {
    resp, err := c.service.Users.Settings.Filters.List("me").Do()
    if err != nil {
        return nil, fmt.Errorf("failed to list filters: %w", err)
    }
    return resp.Filter, nil
}

func (c *Client) CreateFilter(criteria *gmail.FilterCriteria, actions *gmail.FilterAction) (*gmail.Filter, error) {
    filter := &gmail.Filter{
        Criteria: criteria,
        Action:   actions,
    }
    
    resp, err := c.service.Users.Settings.Filters.Create("me", filter).Do()
    if err != nil {
        return nil, fmt.Errorf("failed to create filter: %w", err)
    }
    return resp, nil
}

func (c *Client) UpdateFilter(filterID string, criteria *gmail.FilterCriteria, actions *gmail.FilterAction) (*gmail.Filter, error) {
    filter := &gmail.Filter{
        Id:       filterID,
        Criteria: criteria,
        Action:   actions,
    }
    
    resp, err := c.service.Users.Settings.Filters.Update("me", filterID, filter).Do()
    if err != nil {
        return nil, fmt.Errorf("failed to update filter: %w", err)
    }
    return resp, nil
}

func (c *Client) DeleteFilter(filterID string) error {
    err := c.service.Users.Settings.Filters.Delete("me", filterID).Do()
    if err != nil {
        return fmt.Errorf("failed to delete filter: %w", err)
    }
    return nil
}

func (c *Client) TestFilterCriteria(criteria *gmail.FilterCriteria) ([]*gmail.Message, error) {
    // Convert criteria to search query and use existing search functionality
    query := buildQueryFromCriteria(criteria)
    return c.SearchMessages(query, 10) // Limit to 10 for testing
}
```

## **UI Integration Strategy**

### **1. Keyboard Shortcuts & Commands**

**New Keyboard Shortcuts:**
- `f` - Toggle filters panel (same pattern as `l` for labels)
- `F` - Create filter from current message/search
- `Ctrl+F` - Advanced search (already exists, extend for filter creation)

**Command Integration:**
```bash
:filters list                    # Show filters panel  
:filters create                  # Create new filter
:filters sync                    # Sync from Gmail
:filters test "from:spam"        # Test filter criteria
:filter enable <name/id>         # Enable specific filter
:filter disable <name/id>        # Disable specific filter
:filter delete <name/id>         # Delete specific filter
:filters templates               # Show filter templates
```

### **2. Panel-Based UI Integration** (`internal/tui/filters.go`)

**Filters Panel** (reuses existing side panel system):
```
┌─ 🔧 Gmail Filters ──────────────────────────────────────────────┐
│ [📥 Sync] [🆕 New] [📋 Templates]                    🔍 Filter   │
├─────────────────────────────────────────────────────────────────┤
│ ● 🏢 Work emails → +Work, Archive                               │
│   from:(@company.com)                                           │
│                                                                 │
│ ● 📰 Newsletters → Delete                                       │
│   from:(newsletter OR unsubscribe)                             │
│                                                                 │
│ ● 👨‍💼 Boss emails → +Important, +Boss                            │
│   from:boss@company.com                                         │
│                                                                 │
│ ○ 🚫 Spam cleanup → Delete, !Spam                              │
│   subject:(viagra OR lottery OR prince)                        │
├─────────────────────────────────────────────────────────────────┤
│ Enter: Edit | Space: Toggle | d: Delete | t: Test | Esc: Close │
└─────────────────────────────────────────────────────────────────┘
```

**Status Indicators:**
- `●` = Active filter
- `○` = Inactive filter  
- `🟢` = Recently triggered
- `🔴` = Error/conflict state

### **3. Filter Creation/Editing Interface**

**Enhanced Advanced Search → Filter Creation**:
```
┌─ 🔎 Advanced Search → 🔧 Create Filter ─────────────────────────────────┐
│ CRITERIA:                                                               │
│ 👤 From: [@company.com                 ]                               │
│ 📩 To: [                               ]                               │
│ 🧾 Subject: [                          ]                               │
│ 🔎 Has words: [                        ]                               │
│ 🚫 Doesn't have: [                     ]                               │
│ 📦 Size: [                             ]                               │
│ 📂 Search in: [All Mail                ]                               │
│ 📎 Has attachment: [ ]                                                 │
│                                                                         │
│ ACTIONS:                                                                │
│ 🏷️  Apply labels: [Work] [Important] [+Add]                            │
│ 🗑️  Remove labels: [Inbox] [+Add]                                      │
│ 📥 Archive message: [✓]                                                │
│ ✅ Mark as read: [ ]                                                    │
│ ⭐ Mark as important: [ ]                                               │
│ 📧 Forward to: [backup@example.com      ]                              │
│ 🚫 Never send to Spam: [ ]                                             │
│ 🗑️  Delete it: [ ]                                                     │
│                                                                         │
│ [🧪 Test Filter] [🔧 Create in Gmail] [💾 Save Template] [❌ Cancel]    │
└─────────────────────────────────────────────────────────────────────────┘
```

### **4. Filter Templates Interface**

**Templates Panel**:
```
┌─ 📋 Filter Templates ───────────────────────────────────────────────────┐
│ [🆕 Create] [📥 Import] [📤 Export]                     🔍 Category: All │
├─────────────────────────────────────────────────────────────────────────┤
│ 📰 Newsletter Cleanup                                       📊 Used: 23 │
│    Auto-delete newsletters and promotional emails                       │
│                                                                          │
│ 🚫 Basic Spam Filter                                       📊 Used: 15  │
│    Common spam patterns and keywords                                    │
│                                                                          │
│ 🏢 Work Email Organization                                  📊 Used: 8   │
│    Label and organize work-related emails                               │
│                                                                          │
│ 🔒 Security Alerts                                          📊 Used: 3   │
│    Important security notifications handling                            │
├─────────────────────────────────────────────────────────────────────────┤
│ Enter: Use | e: Edit | d: Delete | c: Create Filter | Esc: Back         │
└─────────────────────────────────────────────────────────────────────────┘
```

## **Workflow Integration Examples**

### **Workflow 1: Create Filter from Current Message**

**Current**: User views a spam message
**Action**: Press `F` 
**Result**: Advanced search form opens with pre-filled criteria from current message, plus actions panel

```go
func (a *App) createFilterFromCurrent() {
    messageID := a.GetCurrentMessageID()
    if messageID == "" {
        a.GetErrorHandler().ShowError(a.ctx, "No message selected")
        return
    }
    
    // Get current message and extract filter criteria
    message, err := a.Client.GetMessage(messageID)
    if err != nil {
        a.GetErrorHandler().ShowError(a.ctx, "Failed to load message")
        return
    }
    
    // Pre-fill criteria from message
    criteria := extractFilterCriteriaFromMessage(message)
    a.openFilterCreationForm(criteria, nil)
}
```

### **Workflow 2: Quick Filter Management**

**Current**: User is in inbox
**Action**: Press `f`
**Result**: Filters panel opens in right side (same as labels)

```go
func (a *App) manageFilters() {
    if a.filtersVisible {
        a.hideFiltersPanel()
        return
    }
    
    a.showFiltersPanel()
    a.currentFocus = "filters"
    a.updateFocusIndicators("filters")
}
```

### **Workflow 3: Filter from Search Results**

**Current**: User runs advanced search for work emails
**Action**: Press `F` to create filter
**Result**: Current search criteria automatically populate filter creation form

### **Workflow 4: Command-based Filter Management**

```bash
# Quick operations via command bar (:)
:f sync                          # Quick sync
:f list                          # Show filters panel
:f create from:spam              # Quick filter creation
:f disable "old newsletter"      # Disable by name
:f templates                     # Show templates
```

## **UI State Management**

### **Panel Visibility Integration** (`internal/tui/keys.go`)

```go
// Following existing pattern for labels
case 'f':
    if a.currentFocus == "search" {
        return nil // Don't interfere with search
    }
    a.manageFilters() // Toggle filters panel

case 'F':
    if a.currentFocus == "search" {
        return nil
    }
    a.createFilterFromCurrent() // Create filter from current context
```

### **Focus Management Integration**

```go
// Enhanced focus ring in toggleFocus()
if a.filtersVisible {
    ring = append(ring, a.filtersView)
    ringNames = append(ringNames, "filters")
}
```

### **ESC Key Behavior Integration**

```go
case tcell.KeyEscape:
    // If filters panel is visible, close it
    if a.filtersVisible {
        a.hideFiltersPanel()
        return nil
    }
    // Continue with existing ESC logic...
```

## **Implementation Plan**

### **Phase 1: Core Infrastructure**
- [ ] Create GmailFilterService interface and implementation
- [ ] Add Gmail API filter methods to gmail client
- [ ] Create database schema for filter templates
- [ ] Implement basic CRUD operations for filters

### **Phase 2: UI Integration**
- [ ] Create filters panel UI component
- [ ] Extend advanced search form for filter creation
- [ ] Add filter actions configuration panel
- [ ] Implement keyboard shortcuts and commands
- [ ] Add focus management and ESC handling

### **Phase 3: Advanced Features**
- [ ] Filter testing and preview functionality
- [ ] Filter templates system
- [ ] Import/export filters
- [ ] Filter conflict detection
- [ ] Usage analytics and optimization suggestions

### **Phase 4: Polish & Enhancement**
- [ ] Performance optimization
- [ ] Error handling and validation
- [ ] User documentation
- [ ] Integration testing

## **Status Bar Integration**

### **Filter Status Display**

```
📧 Inbox (47) | 🔧 5 filters active | 💾 "Work emails" | Message 1/47
```

### **Filter Action Feedback**

```
✅ Filter created: "Spam cleanup" | 🧪 Will affect ~12 existing messages
🔄 Syncing filters from Gmail... | ✅ Synced 8 filters from Gmail  
⚠️  Filter conflict detected: overlapping rules | 🗑️  Filter deleted: "Old project"
```

## **Command System Integration** (`internal/tui/commands.go`)

### **New Commands**

```go
func (a *App) executeFiltersCommand(args []string) {
    if len(args) == 0 {
        a.manageFilters()
        return
    }
    
    subcommand := strings.ToLower(args[0])
    switch subcommand {
    case "list", "show":
        a.manageFilters()
    case "create", "new":
        a.openFilterCreationForm(nil, nil)
    case "sync":
        go a.syncFiltersFromGmail()
    case "test":
        if len(args) > 1 {
            query := strings.Join(args[1:], " ")
            go a.testFilterCriteria(query)
        }
    case "templates":
        a.showFilterTemplates()
    default:
        a.showError(fmt.Sprintf("Unknown filters subcommand: %s", subcommand))
    }
}

func (a *App) executeFilterCommand(args []string) {
    if len(args) < 2 {
        a.showError("Usage: filter <enable|disable|delete> <name/id>")
        return
    }
    
    action := strings.ToLower(args[0])
    filterRef := strings.Join(args[1:], " ")
    
    switch action {
    case "enable":
        go a.enableFilter(filterRef)
    case "disable":
        go a.disableFilter(filterRef)
    case "delete":
        go a.deleteFilter(filterRef)
    default:
        a.showError("Usage: filter <enable|disable|delete> <name/id>")
    }
}
```

## **Error Handling & Validation**

### **Filter Validation**

```go
func validateFilterCriteria(criteria *FilterCriteria) error {
    if criteria.From == "" && criteria.To == "" && criteria.Subject == "" && 
       criteria.Query == "" && !criteria.HasAttachment {
        return fmt.Errorf("filter must have at least one criteria")
    }
    
    if criteria.Size > 0 && criteria.SizeComparison == "" {
        return fmt.Errorf("size comparison must be specified when size is set")
    }
    
    return nil
}

func validateFilterActions(actions *FilterActions) error {
    if !actions.Archive && !actions.Delete && !actions.MarkAsRead && 
       !actions.MarkAsImportant && len(actions.AddLabels) == 0 && 
       len(actions.RemoveLabels) == 0 && actions.Forward == "" {
        return fmt.Errorf("filter must have at least one action")
    }
    
    if actions.Delete && actions.Archive {
        return fmt.Errorf("cannot both delete and archive messages")
    }
    
    return nil
}
```

## **Benefits of This Integration**

### **1. Consistent UX**
- Same keyboard patterns as existing features (`f` like `l` for labels)
- Reuses established panel system
- Familiar ESC/navigation behavior
- Maintains existing workflow patterns

### **2. No New Pages**
- Everything uses existing layout system
- Panels slide in/out like labels and prompts
- Modal overlays for quick actions
- Preserves current message view

### **3. Workflow Efficiency**
- Quick filter creation from any context
- Command shortcuts for power users
- Visual feedback in status bar
- Seamless integration with advanced search

### **4. Gmail Native Integration**
- Filters work immediately across all Gmail clients
- No need to keep TUI running for filters to work
- Access to full Gmail filtering power
- Filters persist across devices and applications

### **5. Muscle Memory Preservation**
- `f` for filters (follows `l` for labels pattern)
- `F` for quick action (follows existing quick action pattern)
- Same ESC behavior as other panels
- Consistent command syntax

This approach maintains your application's design philosophy while adding powerful Gmail filter management through familiar interaction patterns and leveraging Gmail's native capabilities.

## **Future Enhancements**

### **Advanced Features for Future Consideration**
- [ ] Filter performance analytics
- [ ] Automatic filter suggestions based on email patterns
- [ ] Filter A/B testing
- [ ] Integration with saved searches
- [ ] Filter sharing and team collaboration
- [ ] Bulk filter operations
- [ ] Filter backup and restore
- [ ] Advanced conflict resolution