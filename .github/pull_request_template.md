# Pull Request

## 📋 **Description**
Brief description of the changes and why they were made.

## 🏗️ **Architecture Compliance Checklist**

### ✅ **Service Layer** (if applicable)
- [ ] Business logic is implemented in `internal/services/`
- [ ] Service implements an interface defined in `interfaces.go`
- [ ] Service is properly initialized in `initServices()`
- [ ] No business logic in UI components

### ✅ **Error Handling**
- [ ] Uses `app.GetErrorHandler()` for all user feedback
- [ ] Appropriate error levels used (Error, Warning, Success, Info)
- [ ] No direct `fmt.Printf` or `log.Printf` for user-facing messages

### ✅ **Thread Safety**
- [ ] Uses thread-safe accessor methods (`GetCurrentView()`, `SetCurrentMessageID()`, etc.)
- [ ] No direct access to app struct fields
- [ ] New state fields have proper mutex protection

### ✅ **UI Components**
- [ ] UI components focus on presentation only
- [ ] Business logic delegated to services
- [ ] Proper separation of concerns maintained

### ✅ **Testing** (if applicable)
- [ ] Unit tests added for new services
- [ ] Integration tests for complex workflows
- [ ] UI tests for new components

## 🧪 **Testing**
- [ ] `make ci` passes (canonical local gate: lint, vet, tests, race, security, architecture, coverage, desktop)
- [ ] `make pre-commit-check` alias used where appropriate
- [ ] Manual testing completed
- [ ] No regressions in existing functionality

## 📝 **Related Issues**
Closes #issue_number

## ✅ **Definition of Done**
- [ ] Command parity honored (every new shortcut has a `:command` with an alias)
- [ ] In-app `:help` updated if keys/commands changed
- [ ] Config options added to `DefaultConfig()` and the self-migration path
- [ ] Documentation updated if behavior or config changed