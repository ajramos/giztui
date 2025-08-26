package services

import (
	"sync"
)

// DisplayServiceImpl implements the DisplayService interface
type DisplayServiceImpl struct {
	mu            sync.RWMutex
	headerVisible bool
}

// NewDisplayService creates a new DisplayService instance
func NewDisplayService() *DisplayServiceImpl {
	return &DisplayServiceImpl{
		headerVisible: true, // Default to visible
	}
}

// ToggleHeaderVisibility toggles header visibility and returns the new state
func (d *DisplayServiceImpl) ToggleHeaderVisibility() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.headerVisible = !d.headerVisible
	return d.headerVisible
}

// SetHeaderVisibility sets the header visibility state
func (d *DisplayServiceImpl) SetHeaderVisibility(visible bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.headerVisible = visible
}

// IsHeaderVisible returns the current header visibility state
func (d *DisplayServiceImpl) IsHeaderVisible() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.headerVisible
}
