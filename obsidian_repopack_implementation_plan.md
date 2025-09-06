# 📧 Obsidian Repopack Feature Implementation Plan

## 🎯 Feature Overview
Extend the existing Obsidian integration to support "repopack mode" - combining multiple selected emails into a single Markdown file, similar to how bulk prompts compile message content.

## 🏗️ Architecture & Components

### 1. Service Layer Extensions
**File**: `internal/services/obsidian_service.go`
- Add `IngestEmailsToSingleFile()` method to ObsidianService interface
- Reuse bulk prompt compilation format (`combineMessageContents()`) from `BulkPromptServiceImpl`
- Create new template variables for repopack format (message count, compilation date, etc.)

### 2. UI Enhancement
**File**: `internal/tui/obsidian.go`
- Add checkbox to existing Obsidian picker using `tview.Form.AddCheckbox()`
- Update `openObsidianIngestPanel()` and `openBulkObsidianPanel()` to include mode toggle
- Implement proper form navigation (Tab between comment input and checkbox)
- Apply hierarchical theming using `GetComponentColors("obsidian")`

### 3. Data Structures
**File**: `internal/obsidian/types.go`
- Add `RepopackMode bool` field to `ObsidianOptions`
- Add `RepopackMetadata` to track compilation info
- Extend `ObsidianIngestResult` to indicate repopack mode

## 📋 Implementation Steps

### Phase 1: Service Layer Implementation
1. **Add interface method** to `internal/services/interfaces.go`:
   - `IngestEmailsToSingleFile(ctx, messages, accountEmail, options) (*ObsidianIngestResult, error)`

2. **Implement service method** in `obsidian_service.go`:
   - Reuse message compilation logic from `BulkPromptServiceImpl.combineMessageContents()`
   - Create repopack template with metadata header
   - Generate single filename with date and message count

3. **Update data structures** in `obsidian/types.go`:
   - Add `RepopackMode bool` to `ObsidianOptions`
   - Add compilation metadata fields

### Phase 2: UI Integration (CORRECTED)
1. **Update single message picker** (`openObsidianIngestPanel`):
   - Replace simple input with `tview.Form`
   - Add comment input field to form
   - Add "Repopack Mode" checkbox (disabled for single messages)
   - Implement proper Tab navigation

2. **Update bulk picker** (`openBulkObsidianPanel`):
   - Replace simple input with `tview.Form`
   - Add comment input field
   - Add "Repopack Mode" checkbox (enabled, default unchecked)
   - Handle checkbox state in submission

3. **Focus Management** (CORRECTED):
   - Use `currentFocus = "obsidian"` (maintain existing Obsidian focus state)
   - Use `updateFocusIndicators("obsidian")`
   - Use `setActivePicker(PickerObsidian)`
   - Follow established Obsidian picker pattern already in codebase

### Phase 3: Logic Implementation
1. **Update action handlers**:
   - Modify `performObsidianIngest()` to check repopack mode
   - Modify `performBulkObsidianIngest()` to handle mode selection
   - Route to appropriate service method based on checkbox state

2. **Template Integration**:
   - Create repopack template with email compilation format
   - Add variables: `{{message_count}}`, `{{compilation_date}}`, `{{messages}}`
   - Maintain existing template customization support

### Phase 4: Command Parity & Help
1. **Command support** in `internal/tui/commands.go`:
   - Add `:obsidian repack` command variant
   - Support bulk mode automatically
   - Provide short alias `:obs repack`

2. **Help system** updates:
   - Document repopack mode in help text
   - Explain checkbox functionality

## 🎨 UI Design

### Single Message Mode
```
┌─ 📥 Send to Obsidian ──────────────┐
│ Template preview...                │
│                                    │
│ 💬 Pre-message: [input field...] │
│ ☐ Repopack Mode (disabled)        │
│                                    │
│ Tab to navigate | Enter: ingest   │
└────────────────────────────────────┘
```

### Bulk Mode
```
┌─ 📥 Send 5 Messages to Obsidian ───┐
│ Template preview...                │
│                                    │
│ 💬 Bulk comment: [input field...] │
│ ☐ Repopack Mode (combine into one)│
│                                    │
│ Tab to navigate | Enter: ingest   │
└────────────────────────────────────┘
```

## 🔧 Technical Implementation Details

### Form Navigation Pattern
```go
form := tview.NewForm()
form.AddInputField("💬 Comment:", "", 50, nil, func(text string) { comment = text })
form.AddCheckbox("📦 Repopack Mode", false, func(label string, checked bool) { repopackMode = checked })

form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
    switch event.Key() {
    case tcell.KeyEscape:
        a.closeObsidianPanel()
        return nil
    case tcell.KeyEnter:
        go a.performObsidianIngest(message, accountEmail, comment, repopackMode)
        return nil
    }
    return event
})
```

### Service Method Signature
```go
func (s *ObsidianServiceImpl) IngestEmailsToSingleFile(
    ctx context.Context, 
    messages []*gmail.Message, 
    accountEmail string, 
    options obsidian.ObsidianOptions,
) (*obsidian.ObsidianIngestResult, error)
```

### Repopack Template Format
```markdown
---
title: "Email Compilation - {{compilation_date}}"
date: {{compilation_date}}
type: email_repopack
message_count: {{message_count}}
account: {{account_email}}
---

# 📧 Email Repopack - {{message_count}} Messages

**Compiled:** {{compilation_date}}
**Comment:** {{comment}}

---

{{messages}}

---

*Compiled from Gmail using GizTUI repopack mode*
```

## 🧪 Testing Strategy

### Component Tests
- Form navigation with Tab key
- Checkbox state management
- Theme application consistency
- ESC key handling

### Service Tests  
- Repopack template rendering
- Message compilation format
- Single vs bulk mode routing
- Error handling

### Integration Tests
- End-to-end repopack workflow
- Command parity verification
- Focus management compliance

## 📚 Documentation Updates

### README.md
- Add repopack feature to Obsidian integration section
- Include new keyboard/command patterns

### FEATURES.md
- Document repopack mode functionality
- Explain template customization options

## ✅ Architectural Compliance

- ✅ **Service-First**: Business logic in `ObsidianService`
- ✅ **Error Handling**: Uses `GetErrorHandler()` throughout
- ✅ **Thread Safety**: Uses accessor methods, no direct field access
- ✅ **ESC Handling**: Synchronous cleanup, no `QueueUpdateDraw()` in handlers
- ✅ **Command Parity**: `:obsidian repack` command equivalent
- ✅ **Focus Management**: Uses dedicated "obsidian" focus state (CORRECTED)
- ✅ **Bulk Support**: Automatically supports bulk mode
- ✅ **Theming**: Uses `GetComponentColors("obsidian")` system
- ✅ **Logging**: Uses `a.logger` for structured logging

## 🎯 Success Criteria

1. **Single Message**: Checkbox disabled, normal behavior unchanged
2. **Bulk Mode**: Checkbox enabled, toggles between individual files vs single repopack
3. **Template Integration**: Repopack format reuses bulk prompt compilation
4. **UI Consistency**: Follows established picker patterns and theming
5. **Command Parity**: `:obsidian repack` works identically to UI checkbox
6. **Focus Management**: Proper navigation and ESC handling

This implementation extends the existing Obsidian feature with minimal disruption while providing the requested "repopack" functionality that combines multiple emails into a single, organized Markdown file.