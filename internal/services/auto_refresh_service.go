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
