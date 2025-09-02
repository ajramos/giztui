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
		// Prefer the most up-to-date email while loading
		effEmail := accountEmail
		if loading && a.welcomeEmail != "" {
			effEmail = a.welcomeEmail
		}
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetDynamicColors(true)
			text.Clear()
			text.SetText(a.buildWelcomeText(loading, effEmail, dots))
			text.ScrollToBeginning()
		}
		// Do not change focus on startup; keep it in the list for better UX
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

	// Title (avoid unmatched closing tags)
	b.WriteString("[yellow::b]📨 GizTUI — Your terminal for Gmail[-:-:-]\n\n")

	// Subtitle / description
	b.WriteString("A k9s-like terminal for Gmail.\n\n")

	// Account line (if available)
	if strings.TrimSpace(accountEmail) != "" {
		b.WriteString("[green::b]Signed in as:[-:-:-] ")
		b.WriteString(accountEmail)
		b.WriteString("\n\n")
	}

	if loading {
		// Shortcuts while logged in - use configured shortcuts
		shortcuts := a.getWelcomeShortcuts(true)
		b.WriteString(fmt.Sprintf("[white::b]Shortcuts:[-:-:-]  %s\n\n", shortcuts))
		// Do not duplicate loading text; progress is visible in the list title/spinner
		return b.String()
	}

	// Setup guide for first run / missing credentials
	configuredCredPath := strings.TrimSpace(a.Config.Credentials)
	if configuredCredPath == "" {
		configuredCredPath, _ = config.DefaultCredentialPaths()
	}
	b.WriteString("[red::b]Credentials not found.[-:-:-]\n\n")
	b.WriteString("Setup:\n")
	b.WriteString("  1. Download OAuth credentials from Google Cloud Console.\n")
	b.WriteString(fmt.Sprintf("  2. Place the file at `%s`.\n", configuredCredPath))
	b.WriteString("  3. Restart the application.\n\n")
	b.WriteString("See README.md for details.\n\n")
	// Use configured shortcuts for credentials missing state
	shortcuts := a.getWelcomeShortcuts(false)
	b.WriteString(fmt.Sprintf("[white::b]Shortcuts:[-:-:-]  %s\n", shortcuts))
	return b.String()
}

// getWelcomeShortcuts builds the shortcuts string dynamically from user configuration
// loggedIn parameter determines which shortcuts to show (logged in vs credentials missing)
func (a *App) getWelcomeShortcuts(loggedIn bool) string {

	shortcuts := []string{}

	// Helper function to add shortcut with fallback
	addShortcut := func(key, fallback, label string) {
		displayKey := key
		if displayKey == "" {
			displayKey = fallback
		}
		if displayKey != "" {
			shortcuts = append(shortcuts, fmt.Sprintf("[%s %s]", displayKey, label))
		}
	}

	if loggedIn {
		// Logged in state - show main functionality shortcuts
		addShortcut(a.Keys.Help, "?", "Help")
		addShortcut(a.Keys.Search, "s", "Search")
		addShortcut(a.Keys.Unread, "u", "Unread")
		addShortcut(a.Keys.CommandMode, ":", "Commands")

	} else {
		// Credentials missing state - show basic shortcuts only
		addShortcut(a.Keys.Help, "?", "Help")
		addShortcut(a.Keys.Quit, "q", "Quit")

	}

	result := strings.Join(shortcuts, "  ")

	return result
}
