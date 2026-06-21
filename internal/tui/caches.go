package tui

import (
	"sync"

	"github.com/ajramos/giztui/internal/gmail"
)

// appCaches consolidates App's in-memory caches behind one RWMutex. Previously messageCache and
// renderCache had their own mutexes while inviteCache and aiInFlight had none (a latent
// concurrent-map race). One mutex is sufficient — contention is negligible.
//
// Note: the message cache stores *gmail.Message (the project's wrapper around
// *google.golang.org/api/gmail/v1.Message), matching the original App.messageCache type.
type appCaches struct {
	mu         sync.RWMutex
	message    map[string]*gmail.Message
	render     map[string]string
	invite     map[string]Invite
	aiInFlight map[string]bool
}

func newAppCaches() *appCaches {
	return &appCaches{
		message:    make(map[string]*gmail.Message),
		render:     make(map[string]string),
		invite:     make(map[string]Invite),
		aiInFlight: make(map[string]bool),
	}
}

// --- message cache -------------------------------------------------------

func (c *appCaches) messageGet(id string) (*gmail.Message, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.message[id]
	return m, ok
}

func (c *appCaches) messageSet(id string, msg *gmail.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.message[id] = msg
}

// --- render cache --------------------------------------------------------

func (c *appCaches) renderGet(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.render[key]
	return v, ok
}

func (c *appCaches) renderSet(key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.render[key] = val
}

func (c *appCaches) renderDelete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.render, key)
}

func (c *appCaches) renderLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.render)
}

// renderEvictOneIfFull evicts an arbitrary entry when the cache is at or above max
// and key is not already present, then stores key=val. This preserves the original
// setRenderCache eviction semantics under a single lock to stay atomic.
func (c *appCaches) renderEvictOneIfFull(key, val string, max int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.render[key]; !exists && len(c.render) >= max {
		for k := range c.render {
			delete(c.render, k)
			break
		}
	}
	c.render[key] = val
}

// --- invite cache --------------------------------------------------------

func (c *appCaches) inviteGet(id string) (Invite, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	inv, ok := c.invite[id]
	return inv, ok
}

func (c *appCaches) inviteSet(id string, inv Invite) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.invite[id] = inv
}

func (c *appCaches) inviteDelete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.invite, id)
}

func (c *appCaches) inviteLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.invite)
}

// inviteFindAnyWithUID returns the first cached invite that has a non-empty UID.
// Mirrors the RSVP fallback that ranges over the invite cache looking for any
// usable invite when a per-message lookup misses.
func (c *appCaches) inviteFindAnyWithUID() (Invite, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, inv := range c.invite {
		if inv.UID != "" {
			return inv, true
		}
	}
	return Invite{}, false
}

// --- aiInFlight cache -----------------------------------------------------

func (c *appCaches) aiInFlightHas(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.aiInFlight[id]
}

func (c *appCaches) aiInFlightSet(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aiInFlight[id] = true
}

func (c *appCaches) aiInFlightDelete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.aiInFlight, id)
}

// aiInFlightCancelFirst finds the first active in-flight entry, marks it inactive
// (sets it to false, matching the original cancel-on-search behavior), and reports
// whether one was cancelled. Done under a single lock to preserve atomicity of the
// original range-then-mutate-then-break sequence.
func (c *appCaches) aiInFlightCancelFirst() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.aiInFlight {
		if c.aiInFlight[k] {
			c.aiInFlight[k] = false
			return true
		}
	}
	return false
}
