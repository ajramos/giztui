# Auto-refresh Inbox Toggle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an opt-in, configurable background poll that detects new inbox mail and either prepends it incrementally (safe state) or shows a pending count in the status bar (unsafe state) — never disrupting the user's cursor or view.

**Architecture:** A new `AutoRefreshService` (service layer) owns enable/interval state and the new-mail detection (a Gmail list call + a pure diff). The `App` (TUI layer) owns the ticker goroutine lifecycle and decides, per tick, between an incremental prepend and a status-bar counter. The destructive `reloadMessages()` is NOT reused on the happy path; the safe path prepends only the new IDs to the in-memory model and re-renders via the existing `reformatListItems()`.

**Tech Stack:** Go, tview (TUI table), Gmail API (`internal/gmail/client.go`), existing service pattern (`internal/services/`), config (`internal/config/config.go`).

**Spec:** `docs/superpowers/specs/2026-06-08-auto-refresh-inbox-design.md`

---

## File Structure

- `internal/config/config.go` — `AutoRefreshConfig` struct + `Config.AutoRefresh` field + `Keys.AutoRefresh` + defaults + interval helper.
- `internal/services/interfaces.go` — `AutoRefreshService` interface contract.
- `internal/services/auto_refresh_service.go` — implementation: state accessors, pure `diffNewIDs`, `CheckForNewMessages`.
- `internal/services/auto_refresh_service_test.go` — unit tests (pure diff, interval clamping, enable/disable).
- `internal/tui/app.go` — service field, init, ticker lifecycle fields, `Shutdown()` teardown, pending-count accessors.
- `internal/tui/auto_refresh.go` — NEW: ticker lifecycle (`startAutoRefresh`/`stopAutoRefresh`), per-tick orchestration (`performAutoRefreshTick`), safe-state predicate, prepend helper.
- `internal/tui/auto_refresh_test.go` — NEW: predicate + prepend model-math tests.
- `internal/tui/status.go` — indicator in `statusBaseline()`.
- `internal/tui/commands.go` — `:autorefresh`/`:ar` command + routing + suggestion.
- `internal/tui/accounts.go` — reset baseline + counter on account switch.
- `docs/KEYBOARD_SHORTCUTS.md` — documentation.

**Key signatures discovered (use these exactly):**
- `func (c *Client) ListMessagesPage(maxResults int64, pageToken string) ([]*gmail.Message, string, error)` — always lists `LabelIds("INBOX")`; pass `""` as pageToken for the first page. Returned messages have `.Id` populated.
- `func (c *Client) GetMessagesMetadataParallel(messageIDs []string, maxWorkers int) ([]*gmail.Message, error)`
- `func (a *App) GetMessageIDs() []string`, `func (a *App) SetMessageIDs(ids []string)`, `func (a *App) IsMessagesLoading() bool`
- `func (a *App) GetCurrentMessageID() string`, `func (a *App) SetCurrentMessageID(string)`
- `func (a *App) reformatListItems()` — re-renders the table from `a.ids`/`a.messagesMeta` (calls `refreshTableDisplay()`), no network, no `table.Clear()` flash.
- App fields for the predicate: `searchMode string` (`""|"remote"|"local"`), `currentActivePicker ActivePicker` (`PickerNone == ""`), `bulkMode bool`, `compositionPanel` (`== nil || !IsVisible()`), `IsThreadingEnabled() bool`.
- Prepend precedent: `app.go:1691` does `a.messagesMeta = append([]*gmailapi.Message{message}, a.messagesMeta...)`.

---

## Task 1: Config — AutoRefreshConfig block, defaults, key field

**Files:**
- Modify: `internal/config/config.go` (Config struct ~line 64; `Keys` struct ~line 241; `DefaultConfig()` ~line 402)
- Test: `internal/config/config_test.go` (create if absent, else append)

- [ ] **Step 1: Write the failing test**

Append to `internal/config/config_test.go`:

```go
func TestDefaultConfigAutoRefresh(t *testing.T) {
	c := DefaultConfig()
	if c.AutoRefresh.Enabled {
		t.Error("auto_refresh.enabled should default to false (opt-in)")
	}
	if c.AutoRefresh.Interval != "5m" {
		t.Errorf("auto_refresh.interval default = %q, want \"5m\"", c.AutoRefresh.Interval)
	}
	// Interval helper parses and clamps below the 1m minimum.
	if got := c.AutoRefresh.ResolvedInterval(); got != 5*time.Minute {
		t.Errorf("ResolvedInterval() = %v, want 5m", got)
	}
	c.AutoRefresh.Interval = "10s"
	if got := c.AutoRefresh.ResolvedInterval(); got != time.Minute {
		t.Errorf("ResolvedInterval() clamp = %v, want 1m minimum", got)
	}
	c.AutoRefresh.Interval = "garbage"
	if got := c.AutoRefresh.ResolvedInterval(); got != 5*time.Minute {
		t.Errorf("ResolvedInterval() bad value = %v, want 5m fallback", got)
	}
}
```

Ensure `import "time"` is present in the test file.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestDefaultConfigAutoRefresh -v`
Expected: FAIL — `c.AutoRefresh` undefined.

- [ ] **Step 3: Add the config types, field, default, and key**

In `internal/config/config.go`, add near the other config sub-structs (e.g. just above `type Config struct {`):

```go
// AutoRefreshConfig controls opt-in background polling of the inbox for new mail.
type AutoRefreshConfig struct {
	Enabled  bool   `json:"enabled"`
	Interval string `json:"interval"` // Go duration string, e.g. "5m"; clamped to a 1m minimum
}

// autoRefreshMinInterval is the smallest allowed poll interval to avoid hammering the API.
const autoRefreshMinInterval = time.Minute

// autoRefreshDefaultInterval is used when Interval is empty or unparseable.
const autoRefreshDefaultInterval = 5 * time.Minute

// ResolvedInterval parses Interval, falling back to the default and clamping to the minimum.
func (a AutoRefreshConfig) ResolvedInterval() time.Duration {
	d, err := time.ParseDuration(a.Interval)
	if err != nil || d <= 0 {
		return autoRefreshDefaultInterval
	}
	if d < autoRefreshMinInterval {
		return autoRefreshMinInterval
	}
	return d
}
```

Add a field to `type Config struct { ... }`:

```go
	AutoRefresh AutoRefreshConfig `json:"auto_refresh"`
```

Add a field to the `Keys` struct (after `Refresh` ~line 241):

```go
	AutoRefresh string `json:"auto_refresh"` // Toggle background auto-refresh; unbound by default
```

In `DefaultConfig()` (~line 402), set the default block inside the returned `&Config{...}`:

```go
		AutoRefresh: AutoRefreshConfig{Enabled: false, Interval: "5m"},
```

Confirm `internal/config/config.go` already imports `"time"` (it uses `time` elsewhere; if not, add it). Leave `Keys.AutoRefresh` default as `""` (unbound) — do NOT add a default key value in the default `Keys` literal.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestDefaultConfigAutoRefresh -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add auto_refresh config block and unbound toggle key"
```

---

## Task 2: AutoRefreshService — interface + state impl

**Files:**
- Modify: `internal/services/interfaces.go` (add interface near other service interfaces)
- Create: `internal/services/auto_refresh_service.go`
- Test: `internal/services/auto_refresh_service_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/services/auto_refresh_service_test.go`:

```go
package services

import (
	"testing"
	"time"
)

func TestAutoRefreshServiceState(t *testing.T) {
	s := NewAutoRefreshService(nil, false, 5*time.Minute, time.Minute)
	if s.IsEnabled() {
		t.Error("should start disabled")
	}
	s.SetEnabled(true)
	if !s.IsEnabled() {
		t.Error("SetEnabled(true) failed")
	}
	if s.Interval() != 5*time.Minute {
		t.Errorf("Interval() = %v, want 5m", s.Interval())
	}
	// Below minimum is clamped.
	s.SetInterval(10 * time.Second)
	if s.Interval() != time.Minute {
		t.Errorf("SetInterval clamp = %v, want 1m", s.Interval())
	}
	// Zero/negative is ignored (keeps previous).
	s.SetInterval(0)
	if s.Interval() != time.Minute {
		t.Errorf("SetInterval(0) changed value to %v", s.Interval())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestAutoRefreshServiceState -v`
Expected: FAIL — `NewAutoRefreshService` undefined.

- [ ] **Step 3: Add the interface and implementation**

In `internal/services/interfaces.go`, add (near the other service interfaces):

```go
// AutoRefreshService owns the opt-in inbox auto-refresh state and new-mail detection.
type AutoRefreshService interface {
	IsEnabled() bool
	SetEnabled(enabled bool)
	Interval() time.Duration
	SetInterval(d time.Duration)
	// CheckForNewMessages lists the first inbox page and returns IDs not in knownIDs.
	CheckForNewMessages(ctx context.Context, knownIDs []string) (newIDs []string, err error)
}
```

Ensure `interfaces.go` imports `"context"` and `"time"` (it imports `context` already; add `time` if missing).

Create `internal/services/auto_refresh_service.go`:

```go
package services

import (
	"context"
	"sync"
	"time"

	"github.com/ajramos/giztui/internal/gmail"
)

// autoRefreshPageSize is how many inbox IDs to pull per detection tick.
const autoRefreshPageSize = 25

// AutoRefreshServiceImpl implements AutoRefreshService.
type AutoRefreshServiceImpl struct {
	mu          sync.RWMutex
	enabled     bool
	interval    time.Duration
	minInterval time.Duration
	client      *gmail.Client
}

// NewAutoRefreshService creates the service. client may be nil in tests that only
// exercise state accessors and the pure diff.
func NewAutoRefreshService(client *gmail.Client, enabled bool, interval, minInterval time.Duration) *AutoRefreshServiceImpl {
	if minInterval <= 0 {
		minInterval = time.Minute
	}
	if interval < minInterval {
		interval = minInterval
	}
	return &AutoRefreshServiceImpl{
		enabled:     enabled,
		interval:    interval,
		minInterval: minInterval,
		client:      client,
	}
}

func (s *AutoRefreshServiceImpl) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

func (s *AutoRefreshServiceImpl) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = enabled
}

func (s *AutoRefreshServiceImpl) Interval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.interval
}

func (s *AutoRefreshServiceImpl) SetInterval(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d <= 0 {
		return
	}
	if d < s.minInterval {
		d = s.minInterval
	}
	s.interval = d
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestAutoRefreshServiceState -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/interfaces.go internal/services/auto_refresh_service.go internal/services/auto_refresh_service_test.go
git commit -m "feat(services): add AutoRefreshService state and interface"
```

---

## Task 3: Detection — pure diff + CheckForNewMessages

**Files:**
- Modify: `internal/services/auto_refresh_service.go`
- Test: `internal/services/auto_refresh_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/auto_refresh_service_test.go`:

```go
func TestDiffNewIDs(t *testing.T) {
	known := []string{"b", "c", "d"}
	// Fetched is newest-first; "a" is new, "b"/"c" are known.
	got := diffNewIDs([]string{"a", "b", "c"}, known)
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("diffNewIDs = %v, want [a]", got)
	}
	// Nothing new.
	if got := diffNewIDs([]string{"b", "c"}, known); len(got) != 0 {
		t.Errorf("diffNewIDs = %v, want empty", got)
	}
	// Order preserved (newest-first) for multiple new IDs.
	if got := diffNewIDs([]string{"x", "y", "b"}, known); len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Errorf("diffNewIDs = %v, want [x y]", got)
	}
	// Empty known => all fetched are new.
	if got := diffNewIDs([]string{"a", "b"}, nil); len(got) != 2 {
		t.Errorf("diffNewIDs = %v, want 2", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestDiffNewIDs -v`
Expected: FAIL — `diffNewIDs` undefined.

- [ ] **Step 3: Implement diffNewIDs and CheckForNewMessages**

Append to `internal/services/auto_refresh_service.go`:

```go
// diffNewIDs returns the entries of fetched (in order) that are not present in knownIDs.
func diffNewIDs(fetched, knownIDs []string) []string {
	known := make(map[string]struct{}, len(knownIDs))
	for _, id := range knownIDs {
		known[id] = struct{}{}
	}
	var out []string
	for _, id := range fetched {
		if _, ok := known[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

// CheckForNewMessages lists the first inbox page and diffs against knownIDs.
func (s *AutoRefreshServiceImpl) CheckForNewMessages(ctx context.Context, knownIDs []string) ([]string, error) {
	if s.client == nil {
		return nil, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	msgs, _, err := s.client.ListMessagesPage(autoRefreshPageSize, "")
	if err != nil {
		return nil, err
	}
	fetched := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m != nil {
			fetched = append(fetched, m.Id)
		}
	}
	return diffNewIDs(fetched, knownIDs), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestDiffNewIDs -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/auto_refresh_service.go internal/services/auto_refresh_service_test.go
git commit -m "feat(services): add new-mail detection (pure diff + inbox poll)"
```

---

## Task 4: Wire service into App + pending-count state

**Files:**
- Modify: `internal/tui/app.go` (service fields ~line 215; `initServices()` ~line 609; struct fields near line 70)

- [ ] **Step 1: Add fields and accessors**

In the App service-field block (after `preloaderService services.MessagePreloader` ~line 215):

```go
	autoRefreshService      services.AutoRefreshService
```

In the App struct near the message-state fields (e.g. after `messagesMeta`), add ticker + counter state:

```go
	autoRefreshMu      sync.Mutex
	autoRefreshStop    chan struct{}
	autoRefreshRunning bool
	pendingNewCount    int
```

Add accessors near the other thread-safe accessors (e.g. after `ClearMessageIDs`):

```go
// GetPendingNewCount returns the count of detected-but-not-loaded new messages.
func (a *App) GetPendingNewCount() int {
	a.autoRefreshMu.Lock()
	defer a.autoRefreshMu.Unlock()
	return a.pendingNewCount
}

// SetPendingNewCount sets the pending new-message counter shown in the status bar.
func (a *App) SetPendingNewCount(n int) {
	a.autoRefreshMu.Lock()
	a.pendingNewCount = n
	a.autoRefreshMu.Unlock()
}
```

- [ ] **Step 2: Initialize the service in initServices()**

In `initServices()` (~line 609), after the other service constructions, add:

```go
	// Auto-refresh service (opt-in inbox polling)
	a.autoRefreshService = services.NewAutoRefreshService(
		a.Client,
		a.Config.AutoRefresh.Enabled,
		a.Config.AutoRefresh.ResolvedInterval(),
		time.Minute,
	)
```

Confirm `internal/tui/app.go` imports `"time"` and `"sync"` (both already used in app.go).

- [ ] **Step 3: Start the ticker on init if enabled**

At the end of `initServices()` (after the service is constructed), add:

```go
	if a.autoRefreshService.IsEnabled() {
		a.startAutoRefresh()
	}
```

(`startAutoRefresh` is defined in Task 5; this line will not compile until then — that is expected for the next task. If implementing strictly task-by-task with build checks, move this line into Task 5 Step 3.)

- [ ] **Step 4: Verify build is deferred to Task 5**

Run: `go build ./internal/tui/ 2>&1 | head`
Expected: FAIL — `a.startAutoRefresh` undefined (resolved in Task 5). This task has no standalone test; it is wiring.

- [ ] **Step 5: Commit (after Task 5 builds)**

Defer committing this task until Task 5 compiles. (Tasks 4 and 5 share one commit at the end of Task 5.)

---

## Task 5: Ticker lifecycle + Shutdown teardown

**Files:**
- Create: `internal/tui/auto_refresh.go`
- Modify: `internal/tui/app.go` (`Shutdown()` ~line 3318)
- Test: `internal/tui/auto_refresh_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/auto_refresh_test.go`:

```go
package tui

import "testing"

func TestAutoRefreshLifecycleIdempotent(t *testing.T) {
	a := &App{}
	a.ctx = context.Background()
	a.startAutoRefresh()
	if !a.isAutoRefreshRunning() {
		t.Fatal("expected running after start")
	}
	// Starting again must not panic or spawn a second ticker.
	a.startAutoRefresh()
	a.stopAutoRefresh()
	if a.isAutoRefreshRunning() {
		t.Fatal("expected stopped after stop")
	}
	// Stopping again must be a no-op.
	a.stopAutoRefresh()
}
```

Add `import "context"` to the test file.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestAutoRefreshLifecycleIdempotent -v`
Expected: FAIL — `startAutoRefresh` undefined.

- [ ] **Step 3: Implement the lifecycle**

Create `internal/tui/auto_refresh.go`:

```go
package tui

import "time"

// isAutoRefreshRunning reports whether the ticker goroutine is active.
func (a *App) isAutoRefreshRunning() bool {
	a.autoRefreshMu.Lock()
	defer a.autoRefreshMu.Unlock()
	return a.autoRefreshRunning
}

// startAutoRefresh launches the background ticker. Idempotent.
func (a *App) startAutoRefresh() {
	a.autoRefreshMu.Lock()
	if a.autoRefreshRunning {
		a.autoRefreshMu.Unlock()
		return
	}
	stop := make(chan struct{})
	a.autoRefreshStop = stop
	a.autoRefreshRunning = true
	a.autoRefreshMu.Unlock()

	interval := time.Minute
	if a.autoRefreshService != nil {
		interval = a.autoRefreshService.Interval()
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-a.ctx.Done():
				return
			case <-ticker.C:
				// Pick up interval changes without restarting the ticker goroutine.
				if a.autoRefreshService != nil {
					if cur := a.autoRefreshService.Interval(); cur > 0 && cur != interval {
						interval = cur
						ticker.Reset(interval)
					}
				}
				go a.performAutoRefreshTick()
			}
		}
	}()
}

// stopAutoRefresh stops the ticker goroutine. Idempotent.
func (a *App) stopAutoRefresh() {
	a.autoRefreshMu.Lock()
	defer a.autoRefreshMu.Unlock()
	if !a.autoRefreshRunning {
		return
	}
	close(a.autoRefreshStop)
	a.autoRefreshStop = nil
	a.autoRefreshRunning = false
}

// performAutoRefreshTick is implemented in Task 6.
func (a *App) performAutoRefreshTick() {}
```

In `Shutdown()` (`internal/tui/app.go` ~line 3318), add near the top of the teardown:

```go
	a.stopAutoRefresh()
```

Also move the `if a.autoRefreshService.IsEnabled() { a.startAutoRefresh() }` line here from Task 4 Step 3 if it was deferred.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestAutoRefreshLifecycleIdempotent -v`
Expected: PASS.

Then verify the package builds:
Run: `go build ./internal/tui/`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/auto_refresh.go internal/tui/auto_refresh_test.go
git commit -m "feat(tui): wire AutoRefreshService and ticker lifecycle"
```

---

## Task 6: Safe-state predicate + per-tick orchestration

**Files:**
- Modify: `internal/tui/auto_refresh.go`
- Test: `internal/tui/auto_refresh_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/auto_refresh_test.go`:

```go
func TestAutoRefreshSafeState(t *testing.T) {
	a := &App{}
	a.currentActivePicker = PickerNone
	a.searchMode = ""
	a.bulkMode = false
	if !a.isAutoRefreshSafeState() {
		t.Error("plain inbox with nothing open should be safe")
	}
	a.currentActivePicker = PickerLabels
	if a.isAutoRefreshSafeState() {
		t.Error("open picker must be unsafe")
	}
	a.currentActivePicker = PickerNone
	a.searchMode = "remote"
	if a.isAutoRefreshSafeState() {
		t.Error("search mode must be unsafe")
	}
	a.searchMode = ""
	a.bulkMode = true
	if a.isAutoRefreshSafeState() {
		t.Error("bulk mode must be unsafe")
	}
}

func TestAutoRefreshShouldPoll(t *testing.T) {
	a := &App{}
	a.searchMode = ""
	a.currentQuery = ""
	if !a.shouldAutoRefreshPoll() {
		t.Error("plain inbox should poll")
	}
	a.searchMode = "remote"
	if a.shouldAutoRefreshPoll() {
		t.Error("remote search must not poll")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run 'TestAutoRefreshSafeState|TestAutoRefreshShouldPoll' -v`
Expected: FAIL — `isAutoRefreshSafeState` / `shouldAutoRefreshPoll` undefined.

- [ ] **Step 3: Implement predicates and the tick body**

In `internal/tui/auto_refresh.go`, replace the placeholder `performAutoRefreshTick` and add the predicates:

```go
// refreshStatusBar repaints the status baseline so indicator changes (⟳, 📬N)
// show immediately. Must be called on the UI thread (inside QueueUpdateDraw).
func (a *App) refreshStatusBar() {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(a.statusBaseline())
	}
}

// shouldAutoRefreshPoll reports whether the displayed view is the plain inbox,
// i.e. auto-refresh should poll at all. Off-inbox views (search/folder/threading)
// idle the ticker.
func (a *App) shouldAutoRefreshPoll() bool {
	if a.searchMode != "" {
		return false
	}
	if a.currentQuery != "" {
		return false
	}
	if a.IsThreadingEnabled() && a.GetCurrentThreadViewMode() == ThreadViewThread {
		return false
	}
	return true
}

// isAutoRefreshSafeState reports whether it is safe to prepend new rows in place
// (vs. only bumping the status counter).
func (a *App) isAutoRefreshSafeState() bool {
	if !a.shouldAutoRefreshPoll() {
		return false
	}
	if a.currentActivePicker != PickerNone {
		return false
	}
	if a.bulkMode {
		return false
	}
	if a.compositionPanel != nil && a.compositionPanel.IsVisible() {
		return false
	}
	return true
}

// performAutoRefreshTick runs one detection cycle and applies the result.
func (a *App) performAutoRefreshTick() {
	if a.autoRefreshService == nil || !a.autoRefreshService.IsEnabled() {
		return
	}
	if a.IsMessagesLoading() || !a.shouldAutoRefreshPoll() {
		return
	}

	known := a.GetMessageIDs()
	newIDs, err := a.autoRefreshService.CheckForNewMessages(a.ctx, known)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("AUTO_REFRESH: detection error: %v", err)
		}
		return
	}
	if len(newIDs) == 0 {
		return
	}

	if a.isAutoRefreshSafeState() {
		a.prependNewMessages(newIDs)
		return
	}

	// Not safe: surface a pending counter without touching the list.
	a.SetPendingNewCount(len(newIDs))
	a.QueueUpdateDraw(func() {
		a.refreshStatusBar()
	})
}

// prependNewMessages is implemented in Task 7.
func (a *App) prependNewMessages(newIDs []string) {}
```

Note: `refreshStatusBar()` is the existing status repaint helper. If the codebase names it differently (search `func (a *App) refreshStatus` / `updateStatusBar` in `internal/tui/status.go`), use that exact name here and in Task 7/8.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run 'TestAutoRefreshSafeState|TestAutoRefreshShouldPoll' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/auto_refresh.go internal/tui/auto_refresh_test.go
git commit -m "feat(tui): add auto-refresh safe-state predicate and tick orchestration"
```

---

## Task 7: Incremental prepend (the silent reload)

**Files:**
- Modify: `internal/tui/auto_refresh.go`
- Test: `internal/tui/auto_refresh_test.go`

- [ ] **Step 1: Write the failing test (model math only)**

Append to `internal/tui/auto_refresh_test.go`. This tests the pure model transformation used by the prepend (cursor index shift), independent of tview rendering:

```go
func TestPrependModelMath(t *testing.T) {
	// Existing model: cursor on "c" (index 2).
	ids := []string{"a", "b", "c"}
	selectedID := "c"
	newIDs := []string{"x", "y"} // newest-first

	gotIDs, gotIdx := prependIDsAndLocate(newIDs, ids, selectedID)
	wantIDs := []string{"x", "y", "a", "b", "c"}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("ids = %v, want %v", gotIDs, wantIDs)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("ids = %v, want %v", gotIDs, wantIDs)
		}
	}
	if gotIdx != 4 { // "c" moved from 2 to 4 (+len(newIDs))
		t.Errorf("selected index = %d, want 4", gotIdx)
	}

	// Selection not found => index 0 (top).
	_, idx := prependIDsAndLocate(newIDs, ids, "missing")
	if idx != 0 {
		t.Errorf("missing selection index = %d, want 0", idx)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestPrependModelMath -v`
Expected: FAIL — `prependIDsAndLocate` undefined.

- [ ] **Step 3: Implement the pure helper and the UI prepend**

In `internal/tui/auto_refresh.go`, replace the placeholder `prependNewMessages` and add the pure helper:

```go
// prependIDsAndLocate returns the new id slice (newIDs prepended) and the row
// index (0-based, message-space) of selectedID in the new slice, or 0 if absent.
func prependIDsAndLocate(newIDs, existingIDs []string, selectedID string) ([]string, int) {
	merged := make([]string, 0, len(newIDs)+len(existingIDs))
	merged = append(merged, newIDs...)
	merged = append(merged, existingIDs...)
	for i, id := range merged {
		if id == selectedID {
			return merged, i
		}
	}
	return merged, 0
}

// prependNewMessages fetches metadata for newIDs and inserts them at the top of
// the list in place, preserving the user's cursor. No table.Clear(), no spinner.
func (a *App) prependNewMessages(newIDs []string) {
	// Fetch metadata for just the new arrivals (newest-first order preserved).
	metas, err := a.Client.GetMessagesMetadataParallel(newIDs, 10)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("AUTO_REFRESH: metadata fetch error: %v", err)
		}
		return
	}

	// Capture current selection by message ID.
	selectedID := a.GetCurrentMessageID()

	// Update the in-memory model under lock: prepend metas and ids.
	a.mu.Lock()
	a.messagesMeta = append(append([]*gmailapi.Message{}, metas...), a.messagesMeta...)
	a.mu.Unlock()

	mergedIDs, newSelIdx := prependIDsAndLocate(newIDs, a.GetMessageIDs(), selectedID)
	a.SetMessageIDs(mergedIDs)

	// Clear the pending counter — these are now loaded.
	a.SetPendingNewCount(0)

	count := len(newIDs)
	a.QueueUpdateDraw(func() {
		a.reformatListItems() // re-render rows from the model (no network, no clear-flash)
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.Select(newSelIdx+1, 0) // +1 for the header row
		}
		a.refreshStatusBar()
	})

	go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("📬 %d new message(s)", count))
}
```

Add imports to `internal/tui/auto_refresh.go`: `"fmt"`, `"time"`, `"github.com/derailed/tview"` (this codebase uses the **derailed/tview fork**, NOT rivo/tview — verified in `internal/tui/status.go`), and `gmailapi "google.golang.org/api/gmail/v1"` (the exact alias used in `internal/tui/app.go:24`). The `refreshStatusBar` helper added in Task 6 also requires the `derailed/tview` import.

- [ ] **Step 4: Run test + build**

Run: `go test ./internal/tui/ -run TestPrependModelMath -v`
Expected: PASS.
Run: `go build ./internal/tui/`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/auto_refresh.go internal/tui/auto_refresh_test.go
git commit -m "feat(tui): incremental prepend for safe-state auto-refresh"
```

---

## Task 8: Status bar indicator

**Files:**
- Modify: `internal/tui/status.go` (`statusBaseline()` ~line 82)
- Test: `internal/tui/status_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Append to (or create) `internal/tui/status_test.go`:

```go
package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/services"
)

func TestStatusBaselineAutoRefreshIndicator(t *testing.T) {
	a := &App{}
	a.autoRefreshService = services.NewAutoRefreshService(nil, false, time.Minute, time.Minute)

	// Disabled => no ⟳.
	if strings.Contains(a.statusBaseline(), "⟳") {
		t.Error("disabled auto-refresh should not show ⟳")
	}

	// Enabled => ⟳ present.
	a.autoRefreshService.SetEnabled(true)
	if !strings.Contains(a.statusBaseline(), "⟳") {
		t.Error("enabled auto-refresh should show ⟳")
	}

	// Pending count => 📬N present.
	a.SetPendingNewCount(3)
	if !strings.Contains(a.statusBaseline(), "📬3") {
		t.Errorf("pending count should show 📬3, got %q", a.statusBaseline())
	}
}
```

Note: `statusBaseline()` references `a.compositionPanel`, `a.welcomeEmail`, `a.llmTouchUpEnabled` — all zero-valued on a bare `&App{}`, which is fine. If `statusBaseline()` dereferences something that panics on a bare App, construct the minimum needed or guard with `a != nil` checks already present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestStatusBaselineAutoRefreshIndicator -v`
Expected: FAIL — no ⟳/📬 in output.

- [ ] **Step 3: Add the indicator to statusBaseline()**

In `internal/tui/status.go`, inside `statusBaseline()`, after the existing `🧠`/`🧾` block (~line 92), add:

```go
	if a != nil && a.autoRefreshService != nil && a.autoRefreshService.IsEnabled() {
		base += " | ⟳"
		if n := a.GetPendingNewCount(); n > 0 {
			base += fmt.Sprintf(" 📬%d", n)
		}
	}
```

Confirm `internal/tui/status.go` imports `"fmt"` (it uses `fmt` already).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestStatusBaselineAutoRefreshIndicator -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/status.go internal/tui/status_test.go
git commit -m "feat(tui): show auto-refresh status indicator and pending count"
```

---

## Task 9: Toggle command + keybinding + suggestion (command parity)

**Files:**
- Modify: `internal/tui/commands.go` (`executeCommand()` routing + `generateCommandSuggestion()`)
- Modify: `internal/tui/keys.go` (key handler — only fires if a key is configured)
- Test: `internal/tui/commands_test.go` (append; create if absent)

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/commands_test.go`:

```go
func TestToggleAutoRefreshFlipsState(t *testing.T) {
	a := &App{}
	a.ctx = context.Background()
	a.autoRefreshService = services.NewAutoRefreshService(nil, false, time.Minute, time.Minute)

	a.toggleAutoRefresh()
	if !a.autoRefreshService.IsEnabled() {
		t.Error("toggle should enable")
	}
	if !a.isAutoRefreshRunning() {
		t.Error("enabling should start the ticker")
	}
	a.toggleAutoRefresh()
	if a.autoRefreshService.IsEnabled() {
		t.Error("toggle should disable")
	}
	if a.isAutoRefreshRunning() {
		t.Error("disabling should stop the ticker")
	}
}
```

Ensure the test file imports `"context"`, `"time"`, and `services`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestToggleAutoRefreshFlipsState -v`
Expected: FAIL — `toggleAutoRefresh` undefined.

- [ ] **Step 3: Implement toggle, command, suggestion, and key**

Add to `internal/tui/auto_refresh.go`:

```go
// toggleAutoRefresh flips the session enable state and starts/stops the ticker.
func (a *App) toggleAutoRefresh() {
	if a.autoRefreshService == nil {
		return
	}
	enabled := !a.autoRefreshService.IsEnabled()
	a.autoRefreshService.SetEnabled(enabled)
	if enabled {
		a.startAutoRefresh()
		go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("⟳ Auto-refresh ON (every %s)", a.autoRefreshService.Interval()))
	} else {
		a.stopAutoRefresh()
		a.SetPendingNewCount(0)
		go a.GetErrorHandler().ShowInfo(a.ctx, "⟳ Auto-refresh OFF")
	}
	a.QueueUpdateDraw(func() { a.refreshStatusBar() })
}

// executeAutoRefreshCommand handles :autorefresh / :ar [duration].
func (a *App) executeAutoRefreshCommand(args []string) {
	if len(args) > 0 {
		if d, err := time.ParseDuration(args[0]); err == nil && d > 0 {
			a.autoRefreshService.SetInterval(d)
			// Restart ticker to apply immediately if running.
			if a.isAutoRefreshRunning() {
				a.stopAutoRefresh()
				a.startAutoRefresh()
			}
			go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("⟳ Auto-refresh interval set to %s", a.autoRefreshService.Interval()))
			a.QueueUpdateDraw(func() { a.refreshStatusBar() })
			return
		}
		a.GetErrorHandler().ShowWarning(a.ctx, "Usage: :autorefresh [duration] (e.g. :ar 2m)")
		return
	}
	a.toggleAutoRefresh()
}
```

Add `"time"` to the imports of `internal/tui/auto_refresh.go`.

In `internal/tui/commands.go`, inside `executeCommand()` add a case in the command switch (match the existing style — find the `case "refresh", "r":` block as a template):

```go
	case "autorefresh", "ar":
		a.executeAutoRefreshCommand(args)
```

In `generateCommandSuggestion()` (same file), add `autorefresh` to the suggestion list following the pattern used for `refresh` (add an entry like `{"autorefresh", "Toggle background inbox auto-refresh"}` matching the local struct/slice shape used there).

In `internal/tui/keys.go`, the dispatch is a `switch key { ... }` over a `string` key (verified ~line 172: `case a.Keys.Refresh:` runs `go a.reloadMessages(); return true`). Add a sibling case immediately after the `Keys.Refresh` case, mirroring that exact shape:

```go
	case a.Keys.AutoRefresh:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> autorefresh", key)
		}
		a.toggleAutoRefresh()
		return true
```

Because `Keys.AutoRefresh` defaults to `""` (unbound) and `key` is always a real, non-empty key string, this case can never match an actual keypress until the user binds a key in config — so no guard is needed and there is no risk of swallowing the empty key. (Cases are variables, so this does not create a duplicate-`case` compile error even if another key is also unbound.)

- [ ] **Step 4: Run test + build**

Run: `go test ./internal/tui/ -run TestToggleAutoRefreshFlipsState -v`
Expected: PASS.
Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/auto_refresh.go internal/tui/commands.go internal/tui/keys.go internal/tui/commands_test.go
git commit -m "feat(tui): add :autorefresh command, toggle, and optional keybinding"
```

---

## Task 10: Reset baseline + counter on account switch

**Files:**
- Modify: `internal/tui/accounts.go` (~line 381, after the reload triggered by an account switch)

- [ ] **Step 1: Locate the switch point**

Run: `grep -n "reloadMessages\|func (a \*App).*[Ss]witch" internal/tui/accounts.go`
Expected: shows the account-switch handler calling `reloadMessages()` around line 381.

- [ ] **Step 2: Add the reset**

Immediately before/after the `reloadMessages()` call in the account-switch path, add:

```go
	// Auto-refresh: new "new mail" baseline belongs to the newly active account.
	a.SetPendingNewCount(0)
```

The known-ID baseline resets implicitly because `reloadMessages()` rebuilds `a.ids` for the new account; the next tick diffs against that fresh set. No extra baseline field is needed.

- [ ] **Step 3: Build**

Run: `go build ./internal/tui/`
Expected: success.

- [ ] **Step 4: Quick check**

Run: `go test ./internal/tui/ -run TestAutoRefresh -v`
Expected: PASS (no regression).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/accounts.go
git commit -m "feat(tui): reset auto-refresh pending count on account switch"
```

---

## Task 11: Documentation

**Files:**
- Modify: `docs/KEYBOARD_SHORTCUTS.md`

- [ ] **Step 1: Document the command and config**

Add a row/section for auto-refresh. Match the existing table format in the file (find the `:refresh` / `R` row as a template). Include:

- Command: `:autorefresh` / `:ar` — toggle background inbox auto-refresh.
- Command: `:autorefresh <duration>` (e.g. `:ar 2m`) — set the poll interval at runtime.
- Keybinding: unbound by default; configurable via `keys.auto_refresh` in `config.json`.
- Config block:

```json
"auto_refresh": {
  "enabled": false,
  "interval": "5m"
}
```

- Behavior note: only polls while viewing the plain inbox; prepends new mail in place when safe, otherwise shows `📬 N` in the status bar (press `R` to load). Status shows `⟳` when enabled.

- [ ] **Step 2: Commit**

```bash
git add docs/KEYBOARD_SHORTCUTS.md
git commit -m "docs: document auto-refresh command, key, and config"
```

---

## Task 12: Full verification + real-app E2E

**Files:** none (verification only)

- [ ] **Step 1: Run the full pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests all pass. Fix any issues before proceeding.

- [ ] **Step 2: Build the binary**

Run: `make build`
Expected: success; binary reports the current version.

- [ ] **Step 3: Real-app E2E via tmux (hard-won lesson — drive the app)**

Use `/usr/bin/tmux` directly (NOT the zsh `tmux` alias, which is wrapped and breaks scripted use). With a real account:

1. Start the app in a tmux session.
2. Run `:ar 1m` to enable with a fast interval; confirm `⟳` appears in the status bar.
3. While sitting on the inbox, send yourself an email; within ~1 min confirm the new message is **prepended at the top** and the cursor stays on the message you had selected (no spinner, no flash).
4. Open a picker (e.g. labels) or start composing, send yourself another email; confirm the list does NOT change and `📬 N` appears in the status bar instead.
5. Close the picker, press `R`; confirm the pending counter clears.
6. Run `:ar` to toggle OFF; confirm `⟳` disappears and polling stops.

Capture the observations. If any step misbehaves, debug with superpowers:systematic-debugging before claiming done.

- [ ] **Step 4: Final commit (if E2E required fixes)**

```bash
git add -A
git commit -m "fix(tui): auto-refresh E2E corrections"
```

---

## Self-Review Notes

- **Spec coverage:** hybrid model (Tasks 6–7), opt-in config + 5m default + 1m clamp (Task 1), session toggle (Task 9), `AutoRefreshService` service-first (Tasks 2–3), `⟳`/`📬N` indicator (Task 8), multi-account reset (Task 10), inbox-only polling (Task 6 `shouldAutoRefreshPoll`), prepend-not-destructive-reload (Task 7), tests + E2E (all tasks + Task 12). Deletion reconciliation explicitly out of scope (spec) — no task, intentional.
- **Type consistency:** `AutoRefreshService` methods (`IsEnabled`/`SetEnabled`/`Interval`/`SetInterval`/`CheckForNewMessages`) are identical across Tasks 2, 3, 8, 9. App helpers (`startAutoRefresh`/`stopAutoRefresh`/`isAutoRefreshRunning`/`performAutoRefreshTick`/`shouldAutoRefreshPoll`/`isAutoRefreshSafeState`/`prependNewMessages`/`toggleAutoRefresh`/`GetPendingNewCount`/`SetPendingNewCount`) are named consistently across Tasks 4–10.
- **Resolved during planning (no longer open):** (1) status repaint — no pre-existing helper; the plan defines `refreshStatusBar()` (Task 6) using the verified pattern `status.SetText(a.statusBaseline())` against `a.views["status"].(*tview.TextView)`; (2) gmail api alias confirmed `gmailapi "google.golang.org/api/gmail/v1"` (`app.go:24`); (3) tview is the **derailed/tview** fork (Task 7 import corrected from rivo); (4) `keys.go` idiom confirmed as `switch key { case a.Keys.X: ... return true }` (Task 9 corrected, no invented `eventKeyMatches`).
- **Known caveat for Task 12 E2E (obs 1132):** the status bar uses tcell, where East-Asian-Width-ambiguous emojis + VS16 can desync column widths and corrupt the bar. `⟳` (U+27F3, no VS16) is a plain symbol and safe; `📬` (U+1F4EC) is a wide emoji like the existing `🧠`/`🧾` indicators, which already render fine — but watch the status bar for misalignment during E2E and swap `📬` for a plain marker (e.g. `*N` or `+N`) if corruption appears.
