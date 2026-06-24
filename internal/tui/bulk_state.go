package tui

import "sync"

// bulkState holds bulk-selection state extracted from the App god object. `selected` is written on
// the event loop (Space toggles) and ranged from bulk-operation goroutines, so it is mutex-guarded —
// previously an unsynchronized map, a latent concurrent-map race. Leaf lock: methods never call back
// into App, so there is no lock-ordering risk with a.mu.
type bulkState struct {
	mu       sync.RWMutex
	mode     bool
	selected map[string]bool
}

func newBulkState() *bulkState { return &bulkState{selected: map[string]bool{}} }

func (b *bulkState) isMode() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.mode
}

func (b *bulkState) setMode(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mode = v
}

func (b *bulkState) isSelected(id string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.selected[id]
}

func (b *bulkState) add(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.selected[id] = true
}

func (b *bulkState) remove(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.selected, id)
}

// toggle flips membership and returns the new state (true = now selected).
func (b *bulkState) toggle(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.selected[id] {
		delete(b.selected, id)
		return false
	}
	b.selected[id] = true
	return true
}

func (b *bulkState) count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.selected)
}

// ids returns an independent copy of the selected IDs (safe to range while selection changes).
func (b *bulkState) ids() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.selected))
	for id := range b.selected {
		out = append(out, id)
	}
	return out
}

// clear empties the selection.
func (b *bulkState) clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.selected = map[string]bool{}
}
