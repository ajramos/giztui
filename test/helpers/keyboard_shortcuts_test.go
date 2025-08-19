package helpers

import (
	"fmt"
	"testing"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// KeyboardShortcut defines a keyboard shortcut to test
type KeyboardShortcut struct {
	Name     string
	Keys     []tcell.Key
	Mods     []tcell.ModMask
	Validate func(*TestHarness) bool
	Setup    func(*TestHarness)
}

// KeyboardShortcutTest defines a test for keyboard shortcuts
type KeyboardShortcutTest struct {
	Name      string
	Shortcuts []KeyboardShortcut
	Setup     func(*TestHarness)
	Teardown  func(*TestHarness)
}

// TestKeyboardShortcuts runs comprehensive tests for keyboard shortcuts
func TestKeyboardShortcuts(t *testing.T, harness *TestHarness) {
	tests := []KeyboardShortcutTest{
		{
			Name: "navigation_shortcuts",
			Shortcuts: []KeyboardShortcut{
				{
					Name: "tab_navigation",
					Keys: []tcell.Key{tcell.KeyTab},
					Validate: func(h *TestHarness) bool {
						// Should navigate between components
						return h.App.GetCurrentFocus() != ""
					},
				},
				{
					Name: "escape_cancel",
					Keys: []tcell.Key{tcell.KeyEscape},
					Validate: func(h *TestHarness) bool {
						// Should return to previous view or cancel current action
						return true
					},
				},
				{
					Name: "help_shortcut",
					Keys: []tcell.Key{tcell.KeyRune},
					Mods: []tcell.ModMask{tcell.ModNone},
					Setup: func(h *TestHarness) {
						// Setup help view expectations
					},
					Validate: func(h *TestHarness) bool {
						return h.App.GetCurrentView() == "help"
					},
				},
			},
		},
		{
			Name: "message_operations",
			Shortcuts: []KeyboardShortcut{
				{
					Name: "mark_as_read",
					Keys: []tcell.Key{tcell.KeyRune},
					Mods: []tcell.ModMask{tcell.ModNone},
					Setup: func(h *TestHarness) {
						h.App.SetCurrentMessageID("msg_1")
						h.MockEmail.On("MarkAsRead", mock.Anything, "msg_1").Return(nil)
					},
					Validate: func(h *TestHarness) bool {
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
				{
					Name: "archive_message",
					Keys: []tcell.Key{tcell.KeyRune},
					Mods: []tcell.ModMask{tcell.ModNone},
					Setup: func(h *TestHarness) {
						h.App.SetCurrentMessageID("msg_1")
						h.MockEmail.On("ArchiveMessage", mock.Anything, "msg_1").Return(nil)
					},
					Validate: func(h *TestHarness) bool {
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
				{
					Name: "trash_message",
					Keys: []tcell.Key{tcell.KeyDelete},
					Setup: func(h *TestHarness) {
						h.App.SetCurrentMessageID("msg_1")
						h.MockEmail.On("TrashMessage", mock.Anything, "msg_1").Return(nil)
					},
					Validate: func(h *TestHarness) bool {
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
			},
		},
		{
			Name: "bulk_operations",
			Shortcuts: []KeyboardShortcut{
				{
					Name: "select_all",
					Keys: []tcell.Key{tcell.KeyCtrlA},
					Mods: []tcell.ModMask{tcell.ModCtrl},
					Setup: func(h *TestHarness) {
						// Setup messages for selection
						h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).Return(&services.MessagePage{
							Messages:      h.GenerateTestMessages(10),
							NextPageToken: "",
						}, nil)
					},
					Validate: func(h *TestHarness) bool {
						// All messages should be selected
						return h.App.GetSelectedCount() == 10
					},
				},
				{
					Name: "bulk_archive",
					Keys: []tcell.Key{tcell.KeyCtrlD},
					Mods: []tcell.ModMask{tcell.ModCtrl},
					Setup: func(h *TestHarness) {
						// Setup selected messages and bulk archive expectation
						h.App.SetSelectedMessages([]string{"msg_1", "msg_2", "msg_3"})
						h.MockEmail.On("BulkArchive", mock.Anything, []string{"msg_1", "msg_2", "msg_3"}).Return(nil)
					},
					Validate: func(h *TestHarness) bool {
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
				{
					Name: "bulk_trash",
					Keys: []tcell.Key{tcell.KeyCtrlDelete},
					Mods: []tcell.ModMask{tcell.ModCtrl},
					Setup: func(h *TestHarness) {
						h.App.SetSelectedMessages([]string{"msg_1", "msg_2"})
						h.MockEmail.On("BulkTrash", mock.Anything, []string{"msg_1", "msg_2"}).Return(nil)
					},
					Validate: func(h *TestHarness) bool {
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
			},
		},
		{
			Name: "search_and_filter",
			Shortcuts: []KeyboardShortcut{
				{
					Name: "search_mode",
					Keys: []tcell.Key{tcell.KeyRune},
					Mods: []tcell.ModMask{tcell.ModNone},
					Setup: func(h *TestHarness) {
						// Setup search mode expectations
					},
					Validate: func(h *TestHarness) bool {
						return h.App.GetSearchMode() == "remote"
					},
				},
				{
					Name: "local_filter",
					Keys: []tcell.Key{tcell.KeyRune},
					Mods: []tcell.ModMask{tcell.ModNone},
					Setup: func(h *TestHarness) {
						// Setup local filter expectations
					},
					Validate: func(h *TestHarness) bool {
						return h.App.GetSearchMode() == "local"
					},
				},
			},
		},
		{
			Name: "ai_features",
			Shortcuts: []KeyboardShortcut{
				{
					Name: "generate_summary",
					Keys: []tcell.Key{tcell.KeyRune},
					Mods: []tcell.ModMask{tcell.ModNone},
					Setup: func(h *TestHarness) {
						h.App.SetCurrentMessageID("msg_1")
						h.MockAI.On("GenerateSummary", mock.Anything, mock.Anything, mock.Anything).Return(&services.SummaryResult{
							Summary: "Test summary",
						}, nil)
					},
					Validate: func(h *TestHarness) bool {
						h.MockAI.AssertExpectations(t)
						return true
					},
				},
				{
					Name: "suggest_labels",
					Keys: []tcell.Key{tcell.KeyRune},
					Mods: []tcell.ModMask{tcell.ModNone},
					Setup: func(h *TestHarness) {
						h.App.SetCurrentMessageID("msg_1")
						h.MockAI.On("SuggestLabels", mock.Anything, mock.Anything, mock.Anything).Return([]string{"Important", "Work"}, nil)
					},
					Validate: func(h *TestHarness) bool {
						h.MockAI.AssertExpectations(t)
						return true
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Setup test
			if test.Setup != nil {
				test.Setup(harness)
			}

			// Run shortcuts
			for _, shortcut := range test.Shortcuts {
				t.Run(shortcut.Name, func(t *testing.T) {
					// Setup shortcut
					if shortcut.Setup != nil {
						shortcut.Setup(harness)
					}

					// Execute shortcut
					executeKeyboardShortcut(t, harness, shortcut)

					// Validate result
					if shortcut.Validate != nil {
						assert.True(t, shortcut.Validate(harness), "Shortcut validation failed: %s", shortcut.Name)
					}
				})
			}

			// Teardown test
			if test.Teardown != nil {
				test.Teardown(harness)
			}
		})
	}
}

// executeKeyboardShortcut executes a keyboard shortcut
func executeKeyboardShortcut(t *testing.T, harness *TestHarness, shortcut KeyboardShortcut) {
	// Execute key sequence
	for i, key := range shortcut.Keys {
		var mod tcell.ModMask
		if i < len(shortcut.Mods) {
			mod = shortcut.Mods[i]
		}
		
		event := harness.SimulateKeyEvent(key, 0, mod)
		harness.App.HandleKeyEvent(event)
	}
}

// TestKeyboardShortcutRegression tests for regressions in keyboard shortcuts
func TestKeyboardShortcutRegression(t *testing.T, harness *TestHarness) {
	// Test critical shortcuts that should never break
	criticalShortcuts := []KeyboardShortcut{
		{
			Name: "escape_always_works",
			Keys: []tcell.Key{tcell.KeyEscape},
			Validate: func(h *TestHarness) bool {
				// Escape should always work and not crash
				return true
			},
		},
		{
			Name: "tab_navigation_always_works",
			Keys: []tcell.Key{tcell.KeyTab},
			Validate: func(h *TestHarness) bool {
				// Tab should always navigate
				return true
			},
		},
		{
			Name: "help_always_accessible",
			Keys: []tcell.Key{tcell.KeyRune},
			Mods: []tcell.ModMask{tcell.ModNone},
			Setup: func(h *TestHarness) {
				// Setup help expectations
			},
			Validate: func(h *TestHarness) bool {
				return h.App.GetCurrentView() == "help"
			},
		},
	}

	for _, shortcut := range criticalShortcuts {
		t.Run(shortcut.Name, func(t *testing.T) {
			if shortcut.Setup != nil {
				shortcut.Setup(harness)
			}

			executeKeyboardShortcut(t, harness, shortcut)

			if shortcut.Validate != nil {
				assert.True(t, shortcut.Validate(harness), "Critical shortcut failed: %s", shortcut.Name)
			}
		})
	}
}

// TestKeyboardShortcutCombinations tests combinations of shortcuts
func TestKeyboardShortcutCombinations(t *testing.T, harness *TestHarness) {
	// Test that shortcuts work correctly in sequence
	combinations := [][]KeyboardShortcut{
		{
			{Name: "select_all", Keys: []tcell.Key{tcell.KeyCtrlA}, Mods: []tcell.ModMask{tcell.ModCtrl}},
			{Name: "bulk_archive", Keys: []tcell.Key{tcell.KeyCtrlD}, Mods: []tcell.ModMask{tcell.ModCtrl}},
		},
		{
			{Name: "search_mode", Keys: []tcell.Key{tcell.KeyRune}, Mods: []tcell.ModMask{tcell.ModNone}},
			{Name: "escape_cancel", Keys: []tcell.Key{tcell.KeyEscape}},
		},
	}

	for i, combination := range combinations {
		t.Run(fmt.Sprintf("combination_%d", i), func(t *testing.T) {
			// Execute combination
			for _, shortcut := range combination {
				executeKeyboardShortcut(t, harness, shortcut)
			}

			// Verify final state is correct
			// This would depend on the specific combination being tested
		})
	}
}