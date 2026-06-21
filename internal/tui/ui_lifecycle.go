package tui

import "sync/atomic"

// uiLifecycle holds startup/welcome lifecycle flags extracted from the App god object. Both are
// touched from the welcome-screen goroutine and the event loop, so they are atomic.Bool (previously
// plain bools — a latent data race).
type uiLifecycle struct {
	ready            atomic.Bool
	welcomeAnimating atomic.Bool
}
