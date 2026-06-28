package tui

import "sync"

// focusState holds which named view currently has focus, plus the message-view mode, extracted from
// the App god object. Touched from the event loop and from goroutines (picker/AI/attachment paths),
// so it is mutex-guarded — previously two unsynchronized string fields. Non-copyable (sync.RWMutex):
// use as a field accessed via a.focus.* with pointer-receiver methods.
type focusState struct {
	mu      sync.RWMutex
	current string // "list" | "text" | "labels" | "summary" | "slack" | "prompts" | ...
	view    string // "messages" | "thread" | "flat"
}

func (f *focusState) cur() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

func (f *focusState) set(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = name
}

func (f *focusState) is(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current == name
}

func (f *focusState) viewName() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.view
}

func (f *focusState) setView(v string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.view = v
}
