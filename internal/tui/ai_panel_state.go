package tui

import (
	"context"
	"sync"
	"sync/atomic"
)

// aiPanelState groups AI-summary-pane state extracted from the App god object.
//
// visible is read/written from both streaming goroutines and the event loop, so it is an
// atomic.Bool (it was a racy plain bool). inPromptMode is event-loop only. streamingCancel is set
// from streaming goroutines and read/called from the ESC handler, so it is mutex-guarded —
// cancelStreaming() reads, calls, and clears it atomically, fixing a nil-call race where the ESC
// handler could observe a non-nil func a goroutine then set to nil.
type aiPanelState struct {
	visible      atomic.Bool
	inPromptMode bool

	mu              sync.Mutex
	streamingCancel context.CancelFunc
}

// setStreamingCancel records the active streaming cancel func (replacing any previous).
func (s *aiPanelState) setStreamingCancel(cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamingCancel = cancel
}

// cancelStreaming cancels the active streaming op (if any) and clears it, atomically. It reports
// whether a cancellation actually happened, so callers can branch on it (e.g. the ESC handler hides
// the AI panel only when it just cancelled a live stream). Safe to call when none is active.
func (s *aiPanelState) cancelStreaming() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streamingCancel != nil {
		s.streamingCancel()
		s.streamingCancel = nil
		return true
	}
	return false
}

// clearStreamingCancel drops the reference without calling it (goroutine cleanup defers, where the
// context is already done).
func (s *aiPanelState) clearStreamingCancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamingCancel = nil
}

// isStreaming reports whether a streaming op is currently active (used for the ESC debug log).
func (s *aiPanelState) isStreaming() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streamingCancel != nil
}
