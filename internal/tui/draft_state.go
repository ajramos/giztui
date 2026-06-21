package tui

import "sync"

// draftState holds draft-compose mode, extracted from the App god object. mode is read from key/command
// handlers and written from a reload path, so it has its own mutex (the reads were previously
// unprotected).
type draftState struct {
	mu   sync.Mutex
	mode bool
}

func (d *draftState) isMode() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.mode
}

func (d *draftState) setMode(v bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.mode = v
}
