package helpers

import (
	"testing"
)

// TestKeyboardShortcutsFramework demonstrates the keyboard shortcuts testing framework
func TestKeyboardShortcutsFramework(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	t.Run("KeyboardShortcuts", func(t *testing.T) {
		RunKeyboardShortcutsTests(t, harness)
	})

	t.Run("ShortcutRegression", func(t *testing.T) {
		RunKeyboardShortcutRegressionTests(t, harness)
	})

	t.Run("ShortcutCombinations", func(t *testing.T) {
		RunKeyboardShortcutCombinationsTests(t, harness)
	})
}
