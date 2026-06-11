package services

import (
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
