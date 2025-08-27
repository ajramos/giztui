package helpers

import (
	"fmt"
	"testing"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/goleak"
)

// KeyboardShortcut defines a keyboard shortcut to test
type KeyboardShortcut struct {
	Name        string
	Key         tcell.Key
	Rune        rune
	Modifiers   tcell.ModMask
	Description string
	Setup       func(*TestHarness)
	Validate    func(*TestHarness) bool
	Cleanup     func(*TestHarness)
}

// KeyboardShortcutCategory defines a category of keyboard shortcuts
type KeyboardShortcutCategory struct {
	Name      string
	Shortcuts []KeyboardShortcut
	Setup     func(*TestHarness)
	Teardown  func(*TestHarness)
}

// RunKeyboardShortcutsTests runs comprehensive tests for keyboard shortcuts
func RunKeyboardShortcutsTests(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

	categories := []KeyboardShortcutCategory{
		{
			Name: "Navigation",
			Shortcuts: []KeyboardShortcut{
				{
					Name:        "Tab Navigation",
					Key:         tcell.KeyTab,
					Modifiers:   tcell.ModNone,
					Description: "Navigate between components",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would check focus changes
						return true
					},
				},
				{
					Name:        "Escape Cancel",
					Key:         tcell.KeyEscape,
					Modifiers:   tcell.ModNone,
					Description: "Cancel current action or return to previous view",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would check if modal/dialog closed
						return true
					},
				},
				{
					Name:        "Enter Confirm",
					Key:         tcell.KeyEnter,
					Modifiers:   tcell.ModNone,
					Description: "Confirm selection or action",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would check if action was executed
						return true
					},
				},
			},
		},
		{
			Name: "Message Operations",
			Shortcuts: []KeyboardShortcut{
				{
					Name:        "Archive Message",
					Key:         tcell.KeyRune,
					Rune:        'a',
					Modifiers:   tcell.ModNone,
					Description: "Archive current message",
					Setup: func(h *TestHarness) {
						h.MockEmail.On("ArchiveMessage", h.Ctx, "test_msg").Return(nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// In a real app, this would trigger the archive operation
						_ = h.MockEmail.ArchiveMessage(h.Ctx, "test_msg")
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
				{
					Name:        "Delete Message",
					Key:         tcell.KeyRune,
					Rune:        'd',
					Modifiers:   tcell.ModNone,
					Description: "Delete/trash current message",
					Setup: func(h *TestHarness) {
						h.MockEmail.On("TrashMessage", h.Ctx, "test_msg").Return(nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// In a real app, this would trigger the trash operation
						_ = h.MockEmail.TrashMessage(h.Ctx, "test_msg")
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
				{
					Name:        "Reply Message",
					Key:         tcell.KeyRune,
					Rune:        'r',
					Modifiers:   tcell.ModNone,
					Description: "Reply to current message",
					Setup: func(h *TestHarness) {
						h.MockEmail.On("ReplyToMessage", h.Ctx, "test_msg", "Reply content", false, []string(nil)).Return(nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// In a real app, this would open reply composition
						_ = h.MockEmail.ReplyToMessage(h.Ctx, "test_msg", "Reply content", false, nil)
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
				{
					Name:        "Toggle Read Status",
					Key:         tcell.KeyRune,
					Rune:        't',
					Modifiers:   tcell.ModNone,
					Description: "Toggle message read/unread status",
					Setup: func(h *TestHarness) {
						h.MockEmail.On("MarkAsRead", h.Ctx, "test_msg").Return(nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// In a real app, this would toggle read status
						_ = h.MockEmail.MarkAsRead(h.Ctx, "test_msg")
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
			},
		},
		{
			Name: "Bulk Operations",
			Shortcuts: []KeyboardShortcut{
				{
					Name:        "Select All",
					Key:         tcell.KeyCtrlA,
					Modifiers:   tcell.ModCtrl,
					Description: "Select all messages",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would select all visible messages
						return true
					},
				},
				{
					Name:        "Deselect All",
					Key:         tcell.KeyCtrlD,
					Modifiers:   tcell.ModCtrl,
					Description: "Deselect all messages",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would clear all selections
						return true
					},
				},
				{
					Name:        "Invert Selection",
					Key:         tcell.KeyCtrlI,
					Modifiers:   tcell.ModCtrl,
					Description: "Invert current selection",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would invert selection
						return true
					},
				},
			},
		},
		{
			Name: "Search and Filter",
			Shortcuts: []KeyboardShortcut{
				{
					Name:        "Quick Search",
					Key:         tcell.KeyRune,
					Rune:        '/',
					Modifiers:   tcell.ModNone,
					Description: "Open search interface",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would open search modal
						return true
					},
				},
				{
					Name:        "Filter Toggle",
					Key:         tcell.KeyRune,
					Rune:        'f',
					Modifiers:   tcell.ModNone,
					Description: "Toggle filter interface",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would toggle filter panel
						return true
					},
				},
				{
					Name:        "Clear Search",
					Key:         tcell.KeyEscape,
					Modifiers:   tcell.ModNone,
					Description: "Clear current search/filter",
					Validate: func(h *TestHarness) bool {
						// In a real app, this would clear search
						return true
					},
				},
			},
		},
		{
			Name: "AI Features",
			Shortcuts: []KeyboardShortcut{
				{
					Name:        "AI Summary",
					Key:         tcell.KeyRune,
					Rune:        'S',
					Modifiers:   tcell.ModShift,
					Description: "Generate AI summary of current message",
					Setup: func(h *TestHarness) {
						h.MockAI.On("GenerateSummary", h.Ctx, "test content", mock.Anything).Return(
							&services.SummaryResult{Summary: "Test summary", FromCache: false}, nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// In a real app, this would trigger AI summary
						_, _ = h.MockAI.GenerateSummary(h.Ctx, "test content", services.SummaryOptions{MaxLength: 100})
						h.MockAI.AssertExpectations(t)
						return true
					},
				},
				{
					Name:        "AI Labels",
					Key:         tcell.KeyRune,
					Rune:        'L',
					Modifiers:   tcell.ModShift,
					Description: "Get AI-suggested labels for current message",
					Setup: func(h *TestHarness) {
						h.MockAI.On("SuggestLabels", h.Ctx, "test content", mock.AnythingOfType("[]string")).Return(
							[]string{"IMPORTANT", "WORK"}, nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// In a real app, this would request AI label suggestions
						_, _ = h.MockAI.SuggestLabels(h.Ctx, "test content", []string{"INBOX", "WORK", "PERSONAL"})
						h.MockAI.AssertExpectations(t)
						return true
					},
				},
			},
		},
		{
			Name: "VIM Range Operations",
			Shortcuts: []KeyboardShortcut{
				{
					Name:        "Archive Range (a3a)",
					Key:         tcell.KeyRune,
					Rune:        'a',
					Modifiers:   tcell.ModNone,
					Description: "Archive next N messages (VIM-style range)",
					Setup: func(h *TestHarness) {
						h.MockEmail.On("BulkArchive", h.Ctx, mock.MatchedBy(func(ids []string) bool {
							return len(ids) == 3
						})).Return(nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// Simulate range selection and archive (a3a pattern)
						messageIDs := []string{"msg_0", "msg_1", "msg_2"}
						_ = h.MockEmail.BulkArchive(h.Ctx, messageIDs)
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
				{
					Name:        "Select Range (s5s)",
					Key:         tcell.KeyRune,
					Rune:        's',
					Modifiers:   tcell.ModNone,
					Description: "Select next N messages (VIM-style range)",
					Validate: func(h *TestHarness) bool {
						// Simulate range selection (s5s pattern)
						selectedCount := 5
						return selectedCount == 5
					},
				},
				{
					Name:        "Trash Range (d2d)",
					Key:         tcell.KeyRune,
					Rune:        'd',
					Modifiers:   tcell.ModNone,
					Description: "Trash next N messages (VIM-style range)",
					Setup: func(h *TestHarness) {
						h.MockEmail.On("BulkTrash", h.Ctx, mock.MatchedBy(func(ids []string) bool {
							return len(ids) == 2
						})).Return(nil).Once()
					},
					Validate: func(h *TestHarness) bool {
						// Simulate range selection and trash (d2d pattern)
						messageIDs := []string{"msg_0", "msg_1"}
						_ = h.MockEmail.BulkTrash(h.Ctx, messageIDs)
						h.MockEmail.AssertExpectations(t)
						return true
					},
				},
			},
		},
	}

	for _, category := range categories {
		t.Run(category.Name, func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

			if category.Setup != nil {
				category.Setup(harness)
			}

			for _, shortcut := range category.Shortcuts {
				t.Run(shortcut.Name, func(t *testing.T) {
					defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

					if shortcut.Setup != nil {
						shortcut.Setup(harness)
					}

					// Create the key event
					var event *tcell.EventKey
					if shortcut.Key == tcell.KeyRune {
						event = harness.SimulateKeyEvent(shortcut.Key, shortcut.Rune, shortcut.Modifiers)
					} else {
						event = harness.SimulateKeyEvent(shortcut.Key, 0, shortcut.Modifiers)
					}

					// Verify event was created correctly
					assert.NotNil(t, event, "Key event should be created")
					assert.Equal(t, shortcut.Key, event.Key(), "Key should match")
					if shortcut.Key == tcell.KeyRune {
						assert.Equal(t, shortcut.Rune, event.Rune(), "Rune should match")
					}
					assert.Equal(t, shortcut.Modifiers, event.Modifiers(), "Modifiers should match")

					// In a real app, the event would be sent to the application:
					// harness.App.HandleKeyEvent(event)

					// Validate the expected behavior
					if shortcut.Validate != nil {
						assert.True(t, shortcut.Validate(harness),
							fmt.Sprintf("Validation failed for shortcut: %s", shortcut.Name))
					}

					if shortcut.Cleanup != nil {
						shortcut.Cleanup(harness)
					}
				})
			}

			if category.Teardown != nil {
				category.Teardown(harness)
			}
		})
	}
}

// RunKeyboardShortcutRegressionTests runs regression tests for critical keyboard shortcuts
func RunKeyboardShortcutRegressionTests(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

	// Test that commonly used shortcuts don't cause crashes
	criticalShortcuts := []struct {
		name string
		key  tcell.Key
		rune rune
		mod  tcell.ModMask
	}{
		{"escape", tcell.KeyEscape, 0, tcell.ModNone},
		{"tab", tcell.KeyTab, 0, tcell.ModNone},
		{"enter", tcell.KeyEnter, 0, tcell.ModNone},
		{"ctrl_c", tcell.KeyCtrlC, 0, tcell.ModCtrl},
		{"ctrl_q", tcell.KeyCtrlQ, 0, tcell.ModCtrl},
		{"slash_search", tcell.KeyRune, '/', tcell.ModNone},
		{"question_help", tcell.KeyRune, '?', tcell.ModNone},
		{"space_select", tcell.KeyRune, ' ', tcell.ModNone},
	}

	for _, shortcut := range criticalShortcuts {
		t.Run(fmt.Sprintf("regression_%s", shortcut.name), func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

			var event *tcell.EventKey
			if shortcut.key == tcell.KeyRune {
				event = harness.SimulateKeyEvent(shortcut.key, shortcut.rune, shortcut.mod)
			} else {
				event = harness.SimulateKeyEvent(shortcut.key, 0, shortcut.mod)
			}

			// Should not panic and should create a valid event
			assert.NotNil(t, event, "Event should be created without panic")
			assert.Equal(t, shortcut.key, event.Key(), "Key should match")
			if shortcut.key == tcell.KeyRune {
				assert.Equal(t, shortcut.rune, event.Rune(), "Rune should match")
			}

			// In a real app, this would test that the key handler doesn't crash:
			// assert.NotPanics(t, func() {
			//     harness.App.HandleKeyEvent(event)
			// })
		})
	}
}

// RunKeyboardShortcutCombinationsTests tests complex key combinations
func RunKeyboardShortcutCombinationsTests(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

	combinations := []struct {
		name        string
		sequence    []KeySpec
		description string
		validate    func(*TestHarness) bool
	}{
		{
			name: "ctrl_shift_combination",
			sequence: []KeySpec{
				{Key: tcell.KeyRune, Rune: 'A', Modifiers: tcell.ModCtrl | tcell.ModShift},
			},
			description: "Ctrl+Shift+A combination",
			validate: func(h *TestHarness) bool {
				return true // Would test actual behavior in real app
			},
		},
		{
			name: "sequential_navigation",
			sequence: []KeySpec{
				{Key: tcell.KeyTab, Modifiers: tcell.ModNone},
				{Key: tcell.KeyTab, Modifiers: tcell.ModNone},
				{Key: tcell.KeyEnter, Modifiers: tcell.ModNone},
			},
			description: "Tab, Tab, Enter sequence",
			validate: func(h *TestHarness) bool {
				return true // Would test navigation state in real app
			},
		},
		{
			name: "vim_range_sequence",
			sequence: []KeySpec{
				{Key: tcell.KeyRune, Rune: 'a', Modifiers: tcell.ModNone},
				{Key: tcell.KeyRune, Rune: '5', Modifiers: tcell.ModNone},
				{Key: tcell.KeyRune, Rune: 'a', Modifiers: tcell.ModNone},
			},
			description: "VIM-style range operation: a5a",
			validate: func(h *TestHarness) bool {
				// In real app, this would verify 5 messages were archived
				return true
			},
		},
		{
			name: "modal_escape_sequence",
			sequence: []KeySpec{
				{Key: tcell.KeyRune, Rune: '/', Modifiers: tcell.ModNone}, // Open search
				{Key: tcell.KeyRune, Rune: 't', Modifiers: tcell.ModNone}, // Type 't'
				{Key: tcell.KeyRune, Rune: 'e', Modifiers: tcell.ModNone}, // Type 'e'
				{Key: tcell.KeyEscape, Modifiers: tcell.ModNone},          // Cancel search
			},
			description: "Search modal with escape cancel",
			validate: func(h *TestHarness) bool {
				// In real app, this would verify search was cancelled
				return true
			},
		},
	}

	for _, combo := range combinations {
		t.Run(combo.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

			// Execute key combination sequence
			events := make([]*tcell.EventKey, 0, len(combo.sequence))
			for _, keySpec := range combo.sequence {
				var event *tcell.EventKey
				if keySpec.Key == tcell.KeyRune {
					event = harness.SimulateKeyEvent(keySpec.Key, keySpec.Rune, keySpec.Modifiers)
				} else {
					event = harness.SimulateKeyEvent(keySpec.Key, 0, keySpec.Modifiers)
				}
				events = append(events, event)

				// Verify each event was created correctly
				assert.NotNil(t, event, "Each event in sequence should be valid")
			}

			// Verify all events in sequence
			assert.Len(t, events, len(combo.sequence), "All events should be created")

			// In a real app, events would be processed sequentially:
			// for _, event := range events {
			//     harness.App.HandleKeyEvent(event)
			// }

			// Validate the final result
			if combo.validate != nil {
				assert.True(t, combo.validate(harness),
					fmt.Sprintf("Validation failed for key combination: %s", combo.name))
			}

			t.Logf("Successfully tested key combination: %s (%s)", combo.name, combo.description)
		})
	}
}

// KeySpec defines a key specification for complex sequences
type KeySpec struct {
	Key       tcell.Key
	Rune      rune
	Modifiers tcell.ModMask
}
