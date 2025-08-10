package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/derailed/tview"
)

// createWelcomeView builds a composite welcome view for initial app state.
// It returns a tview primitive that can be mounted inside the text container.
// Note: For simplicity and to keep message rendering logic unchanged, we
// currently render the welcome content into the existing `text` view via
// showWelcomeScreen. This constructor remains available if we later decide
// to swap widgets in the `textContainer` instead of writing text.
func (a *App) createWelcomeView(loading bool, accountEmail string) tview.Primitive {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetScrollable(true)
	tv.SetBorder(false)
	tv.SetText(a.buildWelcomeText(loading, accountEmail, 0))
	return tv
}

// showWelcomeScreen renders the welcome content into the existing message
// content area. When loading is true, it shows a lightweight animated dots
// indicator for a short period without blocking input.
func (a *App) showWelcomeScreen(loading bool, accountEmail string) {
	// If UI loop not yet running, avoid QueueUpdate* which would deadlock.
	apply := func(dots int) {
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetDynamicColors(true)
			text.Clear()
			text.SetText(a.buildWelcomeText(loading, accountEmail, dots))
			text.ScrollToBeginning()
		}
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
		if v, ok := a.views["text"].(tview.Primitive); ok {
			a.SetFocus(v)
		}
	}

	if a.uiReady {
		a.QueueUpdateDraw(func() { apply(0) })
	} else {
		apply(0)
	}

	if loading {
		// Guard to prevent multiple concurrent animations
		if a.welcomeAnimating {
			return
		}
		a.welcomeAnimating = true
		// Simple non-blocking animated dots for a short time window
		go func() {
			ticker := time.NewTicker(250 * time.Millisecond)
			defer ticker.Stop()
			// Cap animation duration to avoid lingering goroutines
			timeout := time.NewTimer(5 * time.Second)
			defer timeout.Stop()
			dots := 0
			for {
				select {
				case <-ticker.C:
					dots = (dots + 1) % 4
					if a.uiReady {
						a.QueueUpdateDraw(func() { apply(dots) })
					} else {
						apply(dots)
					}
				case <-timeout.C:
					a.welcomeAnimating = false
					return
				}
			}
		}()
	}
}

// buildWelcomeText constructs the welcome content string using tview color tags.
// `dots` controls the loading indicator intensity (0-3).
func (a *App) buildWelcomeText(loading bool, accountEmail string, dots int) string {
	var b strings.Builder

	// Title
	b.WriteString("[yellow::b]📨 GizTUI — Your terminal for Gmail[-]\n\n")

	// Subtitle / description
	b.WriteString("Explore your Gmail inbox with a k9s-like experience.\n\n")

	// Account line (if available)
	if strings.TrimSpace(accountEmail) != "" {
		b.WriteString(fmt.Sprintf("[green::b]Account:[-] %s\n\n", accountEmail))
	}

	// Quick actions (chips)
	b.WriteString("[white::b]Quick actions:[-]  [? Help]  [s Search]  [u Unread]  [: Commands]\n\n")

	if loading {
		// Loading state with dots
		indicator := strings.Repeat(".", dots)
		b.WriteString(fmt.Sprintf("⏳ Loading inbox%s\n", indicator))
		b.WriteString("Messages will appear shortly. You can browse help or open search in the meantime.\n")
		return b.String()
	}

	// Setup guide for first run / missing credentials
	credPath, _ := config.DefaultCredentialPaths()
	b.WriteString("It looks like Gmail credentials are missing.\n\n")
	b.WriteString("Setup steps:\n")
	b.WriteString("  1. Download OAuth credentials from Google Cloud Console.\n")
	b.WriteString(fmt.Sprintf("  2. Place the file at `%s`.\n", credPath))
	b.WriteString("  3. Restart the application.\n\n")
	b.WriteString("See README.md for details. Press '?' for Help or 'q' to Quit.\n")
	return b.String()
}
