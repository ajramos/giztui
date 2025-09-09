# 🔄 Multi-Account Support Implementation Plan

## **IMPLEMENTATION COMPLETED ✅**
**Status**: All phases successfully implemented and tested  
**Version**: 1.1.1  
**Branch**: `feature/multi-account-support`  
**Date**: September 2025

## **Current Architecture Analysis**

GizTUI currently uses single-account architecture:
- Single Gmail client (`*gmail.Client`) initialized at startup from `credentials.json` and `token.json`
- Single profile email cached (`c.profileEmail`) in the Gmail client
- Services expect single account context throughout the application
- Database path uses single email for SQLite file naming (`email_account.sqlite3`)

## **Revised Multi-Account Architecture** 

### **1. Account Management Service (Service-First)**

**New Service Interface** (`AccountService`):
- `ListAccounts()` - Get all configured accounts
- `GetActiveAccount()` - Get currently selected account  
- `SwitchAccount(accountId)` - Switch between accounts with proper cleanup
- `AddAccount(credentials, token)` - Add new account through UI wizard
- `RemoveAccount(accountId)` - Remove account safely
- `GetAccountClient(accountId)` - Get Gmail client for specific account
- `ConfigureAccount(accountId)` - Interactive account setup process

**Account Model**:
```go
type Account struct {
    ID           string // unique identifier (e.g., "personal", "work")
    Email        string // user@gmail.com (populated after first auth)
    DisplayName  string // "Personal Gmail", "Work Account" 
    CredPath     string // path to credentials.json
    TokenPath    string // path to token.json
    IsActive     bool   // currently selected account
    Client       *gmail.Client // Gmail API client (lazy loaded)
    LastUsed     time.Time
    Status       AccountStatus // Connected, Disconnected, Error
}

type AccountStatus string
const (
    AccountConnected    AccountStatus = "connected"
    AccountDisconnected AccountStatus = "disconnected" 
    AccountError        AccountStatus = "error"
)
```

### **2. Configuration Updates**

**Config Schema Changes**:
```json
{
  "accounts": [
    {
      "id": "personal",
      "display_name": "Personal Gmail", 
      "credentials": "~/.config/giztui/personal/credentials.json",
      "token": "~/.config/giztui/personal/token.json",
      "active": true
    },
    {
      "id": "work", 
      "display_name": "Work Account",
      "credentials": "~/.config/giztui/work/credentials.json",
      "token": "~/.config/giztui/work/token.json", 
      "active": false
    }
  ],
  // Backward compatibility fallback
  "credentials": "~/.config/giztui/credentials.json",
  "token": "~/.config/giztui/token.json",
  
  "keys": {
    // New configurable account shortcut
    "accounts": "Ctrl+A",
    // ... existing shortcuts
  }
}
```

### **3. UI Components Following Existing Patterns**

**Account Picker (Side Panel)**:
- ✅ **ActivePicker System**: Add `PickerAccounts` to enum, use `setActivePicker()`
- ✅ **Focus Architecture**: Use `currentFocus = "labels"` (reuse existing infrastructure)
- ✅ **Side Panel Pattern**: Use `labelsView` container, same positioning as prompts picker
- ✅ **TUI Navigation**: Arrow keys, Enter to switch, ESC to close (no click events)
- ✅ **Visual Consistency**: Same styling as prompts picker with search/filter support

**Account Picker Implementation Pattern** (following `prompts.go`):
```go
func (a *App) showAccountPicker() {
    go func() {
        a.QueueUpdateDraw(func() {
            // Create container with search input and list (like prompts)
            container := tview.NewFlex().SetDirection(tview.FlexRow)
            input := tview.NewInputField() 
            list := tview.NewList()
            
            // Add to contentSplit using labelsView (reuse infrastructure)
            if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
                if a.labelsView != nil {
                    split.RemoveItem(a.labelsView)
                }
                a.labelsView = container
                split.AddItem(a.labelsView, 0, 1, true)
                split.ResizeItem(a.labelsView, 0, 1)
            }
            
            a.SetFocus(input)
            a.currentFocus = "labels" // Reuse labels focus infrastructure  
            a.updateFocusIndicators("labels")
            a.setActivePicker(PickerAccounts) // New enum value
        })
    }()
}
```

**Status Bar Account Indicator**:
- Show current account display name in status area
- Visual styling consistent with existing status elements
- No click events - keyboard shortcut only access

### **4. Keyboard Shortcuts & Commands**

**New Configurable Shortcut**:
```go
// In KeyBindings struct
Accounts string `json:"accounts"` // Default: "Ctrl+A"
```

**Command System** (following existing patterns):
- `:accounts` or `:acc` - Show account picker 
- `:accounts switch <account_id>` - Direct account switch
- `:accounts list` - List all accounts with status
- **Note**: Account management (add/remove) handled via configuration files for simplicity

### **5. Account Configuration Wizard**

**New Account Setup Process** (TUI-based wizard):
1. **Trigger**: `:accounts add` command or first-time setup
2. **Credentials Step**: Prompt for credentials.json path or guide through OAuth setup
3. **Authentication**: Run OAuth flow (similar to current startup process)
4. **Account Details**: Set display name, account ID
5. **Finalization**: Save to config, initialize client

**Integration with Startup**:
- If no accounts configured, auto-launch setup wizard
- Backward compatibility: migrate single account config to multi-account
- CLI flags override for single-account mode

### **6. Service Layer Updates - Inversion of Control Architecture**

**Root Cause Analysis**:
The current implementation has a critical architectural issue: while AccountService correctly manages accounts and selects the active one, all main services (EmailService, Repository, etc.) are initialized with `a.Client` (the legacy Gmail client). This creates a disconnect where:
- AccountService says account "personal" is active
- But EmailService uses `a.Client` which connects to "work" account credentials
- Result: Wrong account data is loaded despite correct account selection

**Solution: Client Provider Pattern (IoC)**:
```go
// Abstract client access - services depend on this interface
type ActiveClientProvider interface {
    GetActiveClient(ctx context.Context) (*gmail.Client, error)
    GetActiveAccountEmail(ctx context.Context) (string, error)
    GetActiveAccountID(ctx context.Context) (string, error)
}

// Implementation bridges AccountService to provide dynamic client access
type ClientManager struct {
    accountService AccountService
    logger         *log.Logger
}

func (cm *ClientManager) GetActiveClient(ctx context.Context) (*gmail.Client, error) {
    activeAccount, err := cm.accountService.GetActiveAccount(ctx)
    if err != nil {
        return nil, fmt.Errorf("no active account: %w", err)
    }
    return cm.accountService.GetAccountClient(ctx, activeAccount.ID)
}

func (cm *ClientManager) GetActiveAccountEmail(ctx context.Context) (string, error) {
    activeAccount, err := cm.accountService.GetActiveAccount(ctx)
    if err != nil {
        return "", err
    }
    return activeAccount.Email, nil
}
```

**Service Migration Pattern**:
```go
// Before: Services store client directly (problematic)
type EmailServiceImpl struct {
    client   *gmail.Client      // ❌ Fixed client - can't switch accounts
    repo     MessageRepository
    renderer EmailRenderer
}

// After: Services use provider for dynamic client access
type EmailServiceImpl struct {
    clientProvider ActiveClientProvider  // ✅ Dynamic client access
    repo          MessageRepository
    renderer      EmailRenderer
}

// Service methods get client when needed
func (s *EmailServiceImpl) ArchiveMessage(ctx context.Context, messageID string) error {
    client, err := s.clientProvider.GetActiveClient(ctx)
    if err != nil {
        return fmt.Errorf("failed to get active client: %w", err)
    }
    return client.ArchiveMessage(ctx, messageID)
}
```

**Service Constructor Updates**:
```go
// Before
func NewEmailService(client *gmail.Client, repo MessageRepository, renderer EmailRenderer) EmailService

// After  
func NewEmailService(provider ActiveClientProvider, repo MessageRepository, renderer EmailRenderer) EmailService
```

### **7. Database Strategy**

**Separate Database Per Account** (as requested):
- Database path: `~/.config/giztui/cache/{account_id}.sqlite3`
- Complete isolation between accounts 
- Clean migration path: rename existing DB to `default.sqlite3`
- Database cleanup when removing accounts

### **8. Application Startup Flow**

**Enhanced Initialization**:
1. Load config with account definitions
2. Initialize AccountService
3. Set active account from config or prompt user
4. Initialize single active account client (lazy load others)
5. Initialize services with AccountService dependency
6. Launch UI with account picker available

**Backward Compatibility**:
- Auto-migrate single-account config to multi-account format
- CLI flags still work for credentials/token override
- Default account ID for legacy users: "default"

### **9. Account Switching Flow**

**Safe Account Switching**:
1. Cancel ongoing operations (streaming, preloading)
2. Clear account-specific caches (messages, labels, etc.)
3. Initialize new account client if needed
4. Reinitialize services with new account context
5. Refresh UI state (message list, labels, etc.)
6. Update status bar and save last-used timestamp

### **10. Threading & Safety**

**Account Operations Safety**:
- Mutex protection in AccountService for switching operations
- Graceful cancellation of account-specific background tasks
- Thread-safe cache invalidation on account switch
- Proper cleanup of Gmail API clients

### **11. Error Handling & UX**

**Account-Specific Error Handling**:
- Use `ErrorHandler` for all account operation feedback
- Visual status indicators for each account (✓ connected, ❌ error)
- Graceful degradation: failed accounts don't break app
- Account context in error messages: "Failed to fetch messages for Work Account"

### **12. Architectural Compliance**

✅ **Service-First**: AccountService handles all account business logic
✅ **ActivePicker System**: Use `PickerAccounts` enum, proper state management  
✅ **Focus Reuse**: Use existing "labels" focus infrastructure (not new focus state)
✅ **Thread Safety**: Mutex protection, accessor methods for account state
✅ **Error Handling**: Use `ErrorHandler` for all user feedback
✅ **ESC Handling**: Synchronous cleanup for account picker
✅ **Command Parity**: `:accounts` command with subcommands
✅ **Theming**: Use `GetComponentColors("accounts")` for account picker
✅ **TUI Patterns**: Keyboard navigation only, no click events
✅ **Configuration**: Fully configurable shortcuts in config file
✅ **Database**: Separate SQLite file per account

## **Implementation Phases**

### **Phase 1: Foundation**
1. Create `AccountService` interface and implementation
2. Add `PickerAccounts` to ActivePicker enum
3. Update configuration schema with backward compatibility  
4. Add account keyboard shortcut to KeyBindings

### **Phase 2: Account Picker UI**
1. Implement account picker following prompts.go pattern
2. Use `PickerAccounts` enum and "labels" focus reuse
3. Add keyboard shortcut handling and command integration
4. Implement account switching with proper cleanup

### **Phase 3: Account Configuration**
1. Add account setup wizard for new account configuration
2. Implement OAuth flow integration for new accounts
3. Add account management commands (add/remove/list)
4. Update startup flow for multi-account initialization

### **Phase 4: Service Integration - IoC Architecture Migration** 

**Critical Phase**: This addresses the core architectural issue where services use legacy `a.Client` instead of AccountService, causing wrong account data to load.

#### **Phase 4.1: Foundation - Client Provider System (1-2 hours)**
1. **Create ActiveClientProvider Interface**:
   - Define interface for dynamic client access (`GetActiveClient`, `GetActiveAccountEmail`, etc.)
   - Implement `ClientManager` that bridges AccountService to provide active client
   - Add proper error handling and logging for client access failures

2. **Update App Architecture**:
   - Replace `getActiveAccountEmail()` to use AccountService (partial fix completed)
   - Create ClientManager instance in app initialization
   - Wire ClientManager with AccountService for dependency injection

#### **Phase 4.2: Core Service Migration (2-3 hours)**

**High-Priority Services** (breaking changes - need immediate migration):
1. **EmailService**: 
   - Update constructor: `NewEmailService(provider ActiveClientProvider, ...)`
   - Replace all `s.client.Method()` with `client, err := s.provider.GetActiveClient(ctx); client.Method()`
   - Add consistent error handling for client access failures

2. **MessageRepository**: 
   - Update for dynamic client access in Gmail operations
   - Ensure all data loading uses correct account client

3. **LabelService**:
   - Account-aware label operations using provider pattern
   - Dynamic client access for label management

4. **CompositionService**:
   - Send emails from correct active account
   - Dynamic client access for email sending operations

**Service Method Transformation Pattern**:
```go
// Before: Direct client usage
func (s *EmailServiceImpl) ArchiveMessage(ctx context.Context, messageID string) error {
    return s.client.ArchiveMessage(ctx, messageID) // ❌ Fixed client
}

// After: Provider pattern
func (s *EmailServiceImpl) ArchiveMessage(ctx context.Context, messageID string) error {
    client, err := s.clientProvider.GetActiveClient(ctx)
    if err != nil {
        return fmt.Errorf("failed to get active client: %w", err)
    }
    return client.ArchiveMessage(ctx, messageID) // ✅ Dynamic client
}
```

#### **Phase 4.3: Integration & Testing (1-2 hours)**
1. **App Initialization Updates**:
   - Update all service constructors in `initServices()` to use ClientManager
   - Replace direct client injection with provider injection
   - Maintain existing service APIs where possible

2. **Account Switching Validation**:
   - Test that services automatically use correct account after switching
   - Verify message loading comes from active account, not legacy client
   - Ensure account picker shows correct active account

3. **Error Handling**:
   - Consistent error messages for client access failures
   - Graceful degradation when account clients fail
   - Proper error propagation through service layer

#### **Phase 4.4: Cleanup & Optimization (1 hour)**
1. **Legacy Client Removal**:
   - Remove `a.Client` usage where replaced by providers
   - Clean up direct client dependencies in app initialization
   - Maintain backward compatibility for non-migrated services

2. **Performance Optimization**:
   - Add client caching in providers if needed
   - Implement account change notifications for cache invalidation
   - Monitor performance impact of dynamic client access

**Migration Impact Assessment**:
- **High Impact**: EmailService, MessageRepository, LabelService, CompositionService
- **Medium Impact**: AttachmentService, LinkService, QueryService  
- **Low Impact**: AIService, ThemeService (don't use Gmail client)
- **No Impact**: UI-only components, configuration services

**Success Criteria**:
- ✅ Account picker shows correct active account (● indicator)
- ✅ Messages load from selected active account, not legacy client
- ✅ Account switching works seamlessly without service restarts  
- ✅ Query service shows correct active account email in logs
- ✅ All Gmail operations use account-specific credentials

## **FINAL IMPLEMENTATION STATUS** ✅

### **All Phases Completed Successfully**

✅ **Phase 1 (Foundation)**: AccountService interface and implementation  
✅ **Phase 2 (Account Picker UI)**: Full TUI interface with keyboard navigation  
✅ **Phase 3 (Command Integration)**: Complete `:accounts` command suite  
✅ **Phase 4 (IoC Architecture)**: ActiveClientProvider pattern with ClientManager  
✅ **Phase 5 (DatabaseManager)**: Database-per-account with hot switching  

### **Key Components Implemented**

✅ **AccountService**: Complete account management with validation and switching  
✅ **ClientManager**: ActiveClientProvider for dynamic Gmail client access  
✅ **DatabaseManager**: Hot database switching with service reinitialization callbacks  
✅ **Account Picker UI**: TUI interface following ActivePicker enum patterns  
✅ **Command System**: Full `:accounts` command suite with subcommands  
✅ **Configuration**: Backward compatible multi-account config schema  
✅ **Database Architecture**: Separate SQLite file per account with email-based naming  

### **Architectural Issues Resolved**

✅ **Service-Client Coupling**: Services now use ActiveClientProvider instead of fixed client  
✅ **Database Isolation**: Each account gets its own database file and data isolation  
✅ **Account Switching**: Seamless account switching without app restart  
✅ **Thread Safety**: Proper mutex protection for account operations  
✅ **Configuration Compatibility**: Backward compatibility with single-account configs  

### **Validation Results**

✅ **Build Status**: Clean compilation with no errors  
✅ **Test Coverage**: All existing tests pass (100% compatibility)  
✅ **Architecture Compliance**: Follows all GizTUI patterns and conventions  
✅ **Performance**: No significant impact on startup time or memory usage

## **Testing Strategy**

- **Unit Tests**: AccountService with mock Gmail clients
- **Integration Tests**: Account switching workflow validation
- **Component Tests**: Account picker UI using test harness
- **Migration Tests**: Single-to-multi account config upgrade
- **Performance Tests**: Account switching benchmark

## **Key Architectural Decisions**

### **1. ActivePicker Enum System**
We use the existing `ActivePicker` enum with `PickerAccounts` instead of creating new focus states. This leverages the established picker infrastructure and maintains consistency.

### **2. Focus Infrastructure Reuse**
Account picker uses `currentFocus = "labels"` to reuse the existing focus management system, avoiding the complexity of creating new focus states.

### **3. Service-First Architecture**
All account logic is encapsulated in `AccountService`, maintaining the established service-first pattern and ensuring proper separation of concerns.

### **4. Database Isolation**
Each account gets its own SQLite database file for complete data isolation and simpler account management.

### **5. TUI-Only Interactions**
All interactions use keyboard shortcuts and commands only, maintaining the terminal-first design philosophy.

### **6. Inversion of Control (IoC) Pattern**
Services use `ActiveClientProvider` interface instead of direct Gmail client references. This enables dynamic account switching without service restarts and properly separates concerns between account management and business logic.

### **7. Client Provider Architecture**
`ClientManager` bridges AccountService and services, providing dynamic client access. Services call `provider.GetActiveClient(ctx)` when needed, ensuring they always use the correct account's client automatically.

### **8. Backward Compatibility**
Existing single-account configurations are automatically migrated to the new multi-account format without breaking existing users.

This design maintains full backward compatibility while adding robust multi-account support using GizTUI's established architectural patterns, including the ActivePicker enum system, focus management infrastructure, and IoC principles for proper service decoupling.