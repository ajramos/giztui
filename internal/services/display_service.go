package services

import (
	"sync"
)

// DisplayServiceImpl implements the DisplayService interface
type DisplayServiceImpl struct {
	mu             sync.RWMutex
	headerVisible  bool
	markdownRender bool
}

// NewDisplayService creates a new DisplayService. markdownDefault sets the
// initial markdown-rendering mode.
func NewDisplayService(markdownDefault bool) *DisplayServiceImpl {
	return &DisplayServiceImpl{
		headerVisible:  true,
		markdownRender: markdownDefault,
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

// ToggleMarkdownRendering flips markdown rendering and returns the new state.
func (d *DisplayServiceImpl) ToggleMarkdownRendering() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.markdownRender = !d.markdownRender
	return d.markdownRender
}

// SetMarkdownRendering sets the markdown rendering state.
func (d *DisplayServiceImpl) SetMarkdownRendering(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.markdownRender = enabled
}

// IsMarkdownRendering returns the current markdown rendering state.
func (d *DisplayServiceImpl) IsMarkdownRendering() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.markdownRender
}
