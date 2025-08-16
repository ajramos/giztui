# 💾 **Saved Search Queries Feature Proposal**

## **Overview**
Allow users to save, name, and quickly recall frequently used advanced search queries, similar to bookmarks but for searches.

## **Architecture Design**

### **1. Database Schema** (`internal/db/search_store.go`)

```sql
CREATE TABLE IF NOT EXISTS saved_searches (
    id TEXT PRIMARY KEY,
    account_email TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    query TEXT NOT NULL,
    search_criteria_json TEXT, -- Store original form fields
    usage_count INTEGER DEFAULT 0,
    last_used DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    tags TEXT, -- Comma-separated tags for organization
    is_favorite BOOLEAN DEFAULT FALSE
);
```

### **2. Service Layer** (`internal/services/search_service.go`)

```go
type SavedSearchService interface {
    SaveSearch(ctx context.Context, search *SavedSearch) error
    ListSavedSearches(ctx context.Context, accountEmail string) ([]*SavedSearch, error)
    DeleteSavedSearch(ctx context.Context, searchID string) error
    UpdateSavedSearch(ctx context.Context, search *SavedSearch) error
    IncrementUsage(ctx context.Context, searchID string) error
    GetPopularSearches(ctx context.Context, accountEmail string, limit int) ([]*SavedSearch, error)
}

type SavedSearch struct {
    ID          string
    Name        string
    Description string
    Query       string
    Criteria    *SearchCriteria // Original form fields
    UsageCount  int
    LastUsed    time.Time
    Created     time.Time
    Tags        []string
    IsFavorite  bool
}

type SearchCriteria struct {
    From        string
    To          string
    Subject     string
    HasWords    string
    NotWords    string
    Size        string
    DateWithin  string
    Scope       string
    HasAttachment bool
}
```

### **3. UI Implementation**

## **User Experience**

### **Save Search Dialog**
```
┌─ 💾 Save Search Query ─────────────────────────────────────────┐
│ Name: [Work emails from last month                    ]        │
│ Description: [All work-related emails from the past 30 days]  │
│ Tags: [work, monthly, reports                         ]        │
│ ⭐ Mark as favorite                                            │
│                                                                │
│ Query Preview: from:@company.com newer_than:30d               │
│                                                                │
│ [Save] [Cancel]                                                │
└────────────────────────────────────────────────────────────────┘
```

### **Saved Searches Manager**
```
┌─ 💾 Saved Search Queries ─────────────────────────────────────────────┐
│ [Create New] [Import] [Export]                    🔍 Filter: [work ]   │
├───────────────────────────────────────────────────────────────────────┤
│ ⭐ Work emails from last month          📊 Used: 15  📅 2d ago        │
│    from:@company.com newer_than:30d                                    │
│                                                                        │
│ 🔍 Unread newsletters                   📊 Used: 8   📅 1w ago        │
│    is:unread category:promotions                                       │
│                                                                        │
│ 📧 Important from boss                  📊 Used: 23  📅 3h ago        │
│    from:boss@company.com is:important                                  │
│                                                                        │
│ 🏷️  Project Alpha emails               📊 Used: 5   📅 2w ago        │
│    label:"Project Alpha" OR subject:"alpha"                           │
├───────────────────────────────────────────────────────────────────────┤
│ Enter: Run | e: Edit | d: Delete | r: Rename | Esc: Back             │
└───────────────────────────────────────────────────────────────────────┘
```

### **Quick Access Integration**

**Enhanced Advanced Search Form**:
```
┌─ 🔎 Advanced Search ───────────────────┐  ┌─ 💾 Saved Queries ─────┐
│ 👤 From: [                    ]        │  │ ⭐ Work emails         │
│ 📩 To: [                      ]        │  │ 🔍 Unread newsletters  │
│ 🧾 Subject: [                 ]        │  │ 📧 Important from boss │
│ 🔎 Has the words: [           ]        │  │ 🏷️  Project Alpha      │
│ 🚫 Doesn't have: [            ]        │  │ ─────────────────────  │
│ 📦 Size: [                    ]        │  │ ➕ Save current search │
│ ⏱️  Date within: [             ]        │  │ 📂 Manage all searches │
│ 📂 Search: [All Mail          ]        │  └────────────────────────┘
│ 📎 Has attachment: [ ]                 │
│                                        │
│ [🔎 Search] [💾 Save] [📂 Load]         │
└────────────────────────────────────────┘
```

## **Key Features**

### **1. Smart Suggestions**
- Auto-suggest names based on search criteria
- Show similar existing searches to avoid duplicates
- Suggest tags based on content and previous searches

### **2. Search Analytics**
- Track usage frequency and last used date
- Show popular/trending searches
- Identify unused searches for cleanup

### **3. Organization Features**
- **Tags**: Categorize searches (work, personal, projects)
- **Favorites**: Quick access to most-used searches
- **Folders**: Group related searches together

### **4. Quick Access Methods**
- **Keyboard shortcut**: `Ctrl+S` to save current search
- **Search picker**: `Ctrl+L` to load saved search
- **Command integration**: `:search load <name>`

### **5. Import/Export**
- Export searches as JSON for backup
- Share search queries with team members
- Import common search templates

## **Integration Points**

### **Command System Enhancement**
```go
// New command syntax
:search save "work emails" "from:@company.com"
:search load "work emails"
:search list
:search delete "old search"
```

### **Keyboard Shortcuts**
- `Ctrl+S`: Save current search query
- `Ctrl+L`: Load saved search picker
- `Ctrl+Shift+S`: Quick save with auto-generated name

### **Status Bar Integration**
Show saved search indicator when using a saved query:
```
💾 "Work emails from last month" | Message 1/25 | ESC: Back
```

## **Implementation Plan**

### **Phase 1: Core Functionality**
- [ ] Create database schema and service layer
- [ ] Implement basic save/load functionality
- [ ] Add save dialog to advanced search form
- [ ] Create saved searches management interface

### **Phase 2: Enhanced UX**
- [ ] Add quick access panel to advanced search
- [ ] Implement usage tracking and analytics
- [ ] Add search suggestions and auto-naming
- [ ] Keyboard shortcuts integration

### **Phase 3: Advanced Features**
- [ ] Tags and organization system
- [ ] Import/export functionality
- [ ] Search templates and sharing
- [ ] Integration with command system

## **Benefits**

### **For Users**
- Faster access to complex searches
- Reduced cognitive load for remembering syntax
- Analytics to optimize email workflow
- Shareable search templates for teams

### **For Productivity**
- Quick access to frequently used searches
- Reduced time typing complex Gmail queries
- Better organization of search workflows
- Historical tracking of search patterns

## **Technical Considerations**

### **Data Storage**
- Local SQLite storage for fast access
- JSON serialization for search criteria
- Automatic cleanup of unused searches
- Backup/restore functionality

### **Performance**
- Efficient indexing for quick search/retrieval
- Lazy loading for large search collections
- Caching of frequently used searches
- Background usage analytics updates

### **User Experience**
- Non-intrusive save prompts
- Quick keyboard access
- Visual indicators for saved searches
- Seamless integration with existing search flow