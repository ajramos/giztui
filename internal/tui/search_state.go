package tui

import (
	"sync"

	gmailapi "google.golang.org/api/gmail/v1"
)

// searchState holds the search/filter state, extracted from the App god object. mode + query are
// read by the next-page preload goroutines, so they are guarded by mu; localFilter and the base
// snapshot are event-loop-only. The base snapshot is a copy of the inbox view taken when a local
// filter starts, so it can be restored on exit. sync.RWMutex makes searchState non-copyable: use it
// as a field accessed via a.search.* with pointer-receiver methods.
type searchState struct {
	mu          sync.RWMutex
	mode        string // "" | "remote" | "local"
	query       string // current query
	localFilter string // event-loop only

	// Local-filter base snapshot (event-loop only).
	baseIDs           []string
	baseMessagesMeta  []*gmailapi.Message
	baseNextPageToken string
	baseSelectionID   string
}

func (s *searchState) Mode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *searchState) SetMode(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = m
}

func (s *searchState) Query() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.query
}

func (s *searchState) SetQuery(q string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.query = q
}

// clear resets mode, query, and localFilter to empty (used when exiting a search/filter).
func (s *searchState) clear() {
	s.mu.Lock()
	s.mode = ""
	s.query = ""
	s.mu.Unlock()
	s.localFilter = ""
}

// captureSnapshot stores an independent copy of the current inbox view as the local-filter base.
func (s *searchState) captureSnapshot(ids []string, meta []*gmailapi.Message, token, selID string) {
	s.baseIDs = append([]string(nil), ids...)
	s.baseMessagesMeta = append([]*gmailapi.Message(nil), meta...)
	s.baseNextPageToken = token
	s.baseSelectionID = selID
}

// snapshot returns independent copies of the base ids/meta plus the token and selection id.
func (s *searchState) snapshot() (ids []string, meta []*gmailapi.Message, token, selID string) {
	ids = append([]string(nil), s.baseIDs...)
	meta = append([]*gmailapi.Message(nil), s.baseMessagesMeta...)
	return ids, meta, s.baseNextPageToken, s.baseSelectionID
}

// removeFromSnapshotByID drops one message (and its aligned meta) from the base snapshot.
func (s *searchState) removeFromSnapshotByID(id string) {
	idx := -1
	for i, x := range s.baseIDs {
		if x == id {
			idx = i
			break
		}
	}
	if idx >= 0 {
		s.baseIDs = append(s.baseIDs[:idx], s.baseIDs[idx+1:]...)
		if idx < len(s.baseMessagesMeta) {
			s.baseMessagesMeta = append(s.baseMessagesMeta[:idx], s.baseMessagesMeta[idx+1:]...)
		}
	}
}

// removeFromSnapshotByIDs drops a set of messages from the base snapshot, preserving order/alignment.
func (s *searchState) removeFromSnapshotByIDs(ids []string) {
	rm := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		rm[id] = struct{}{}
	}
	newIDs := s.baseIDs[:0]
	newMeta := s.baseMessagesMeta[:0]
	for i, id := range s.baseIDs {
		if _, ok := rm[id]; ok {
			continue
		}
		newIDs = append(newIDs, id)
		if i < len(s.baseMessagesMeta) {
			newMeta = append(newMeta, s.baseMessagesMeta[i])
		}
	}
	s.baseIDs = append([]string(nil), newIDs...)
	s.baseMessagesMeta = append([]*gmailapi.Message(nil), newMeta...)
}
