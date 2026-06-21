package tui

import "sync"

// layoutState holds screen geometry extracted from the App god object. width/height are read on the
// event loop and from resize handling, so they keep their own RWMutex (previously the App-wide a.mu).
// currentLayout is event-loop only.
type layoutState struct {
	mu            sync.RWMutex
	width         int
	height        int
	currentLayout LayoutType
}

func (l *layoutState) size() (int, int) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.width, l.height
}

func (l *layoutState) setSize(w, h int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.width = w
	l.height = h
}
