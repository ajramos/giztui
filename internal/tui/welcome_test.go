package tui

import (
	"strings"
	"testing"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/derailed/tview"
)

func TestGetWelcomeShortcuts_CustomConfig(t *testing.T) {
	// Create an App with custom key configuration
	app := &App{
		Keys: config.KeyBindings{
			Help:        "F1",
			Search:      "/",
			Unread:      "U",
			CommandMode: ";",
			Quit:        "Q",
		},
	}

	// Test logged in shortcuts
	loggedInShortcuts := app.getWelcomeShortcuts(true)
	if !strings.Contains(loggedInShortcuts, "[F1 Help]") {
		t.Errorf("Expected custom Help shortcut F1, got: %s", loggedInShortcuts)
	}
	if !strings.Contains(loggedInShortcuts, "[/ Search]") {
		t.Errorf("Expected custom Search shortcut /, got: %s", loggedInShortcuts)
	}
	if !strings.Contains(loggedInShortcuts, "[U Unread]") {
		t.Errorf("Expected custom Unread shortcut U, got: %s", loggedInShortcuts)
	}
	if !strings.Contains(loggedInShortcuts, "[; Commands]") {
		t.Errorf("Expected custom Commands shortcut ;, got: %s", loggedInShortcuts)
	}

	// Test credentials missing shortcuts
	credentialsMissingShortcuts := app.getWelcomeShortcuts(false)
	if !strings.Contains(credentialsMissingShortcuts, "[F1 Help]") {
		t.Errorf("Expected custom Help shortcut F1, got: %s", credentialsMissingShortcuts)
	}
	if !strings.Contains(credentialsMissingShortcuts, "[Q Quit]") {
		t.Errorf("Expected custom Quit shortcut Q, got: %s", credentialsMissingShortcuts)
	}
}

func TestGetWelcomeShortcuts_DefaultFallback(t *testing.T) {
	// Create an App with empty key configuration to test fallbacks
	app := &App{
		Keys: config.KeyBindings{
			// Empty configuration - should use fallbacks
		},
	}

	// Test logged in shortcuts with fallbacks
	loggedInShortcuts := app.getWelcomeShortcuts(true)
	if !strings.Contains(loggedInShortcuts, "[? Help]") {
		t.Errorf("Expected default Help shortcut fallback ?, got: %s", loggedInShortcuts)
	}
	if !strings.Contains(loggedInShortcuts, "[s Search]") {
		t.Errorf("Expected default Search shortcut fallback s, got: %s", loggedInShortcuts)
	}
	if !strings.Contains(loggedInShortcuts, "[u Unread]") {
		t.Errorf("Expected default Unread shortcut fallback u, got: %s", loggedInShortcuts)
	}
	if !strings.Contains(loggedInShortcuts, "[: Commands]") {
		t.Errorf("Expected default Commands shortcut fallback :, got: %s", loggedInShortcuts)
	}

	// Test credentials missing shortcuts with fallbacks
	credentialsMissingShortcuts := app.getWelcomeShortcuts(false)
	if !strings.Contains(credentialsMissingShortcuts, "[? Help]") {
		t.Errorf("Expected default Help shortcut fallback ?, got: %s", credentialsMissingShortcuts)
	}
	if !strings.Contains(credentialsMissingShortcuts, "[q Quit]") {
		t.Errorf("Expected default Quit shortcut fallback q, got: %s", credentialsMissingShortcuts)
	}
}

func TestBuildWelcomeText_UsesCustomShortcuts(t *testing.T) {
	// Create an App with custom configuration
	app := &App{
		Application: tview.NewApplication(),
		Config: &config.Config{
			Credentials: "",
		},
		Keys: config.KeyBindings{
			Help:        "F1",
			Search:      "/",
			Unread:      "U",
			CommandMode: ";",
			Quit:        "Q",
		},
	}

	// Test loading state (logged in)
	loadingText := app.buildWelcomeText(true, "test@example.com", 0)
	if !strings.Contains(loadingText, "[F1 Help]") {
		t.Errorf("Loading welcome text should contain custom Help shortcut F1")
	}
	if !strings.Contains(loadingText, "[/ Search]") {
		t.Errorf("Loading welcome text should contain custom Search shortcut /")
	}

	// Test credentials missing state
	credentialsText := app.buildWelcomeText(false, "", 0)
	if !strings.Contains(credentialsText, "[F1 Help]") {
		t.Errorf("Credentials missing welcome text should contain custom Help shortcut F1")
	}
	if !strings.Contains(credentialsText, "[Q Quit]") {
		t.Errorf("Credentials missing welcome text should contain custom Quit shortcut Q")
	}
}

func TestWelcomeShortcuts_Integration(t *testing.T) {
	// Test that the shortcuts are displayed correctly in different states

	// Create app with mixed custom and default shortcuts
	app := &App{
		Application: tview.NewApplication(),
		Config: &config.Config{
			Credentials: "/path/to/creds.json",
		},
		Keys: config.KeyBindings{
			Help:        "F1", // Custom
			Search:      "",   // Empty, should use fallback 's'
			Unread:      "U",  // Custom
			CommandMode: "",   // Empty, should use fallback ':'
			Quit:        "Q",  // Custom
		},
	}

	// Test mixed configuration
	shortcuts := app.getWelcomeShortcuts(true)
	expectedShortcuts := []string{"[F1 Help]", "[s Search]", "[U Unread]", "[: Commands]"}

	for _, expected := range expectedShortcuts {
		if !strings.Contains(shortcuts, expected) {
			t.Errorf("Expected shortcut %s not found in: %s", expected, shortcuts)
		}
	}
}
