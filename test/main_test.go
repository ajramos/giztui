package test

import (
	"testing"

	"github.com/ajramos/gmail-tui/test/helpers"
)

// TestMain runs all test suites
func TestMain(m *testing.M) {
	// Run tests
	m.Run()
}

// TestSuite runs the complete test suite
func TestSuite(t *testing.T) {
	t.Run("TUI_Components", func(t *testing.T) {
		harness := helpers.NewTestHarness(t)
		defer harness.Cleanup()

		// Test visual regression
		t.Run("Visual_Regression", func(t *testing.T) {
			helpers.TestVisualRegression(t, harness)
		})

		// Test keyboard shortcuts
		t.Run("Keyboard_Shortcuts", func(t *testing.T) {
			helpers.TestKeyboardShortcuts(t, harness)
		})

		// Test bulk operations
		t.Run("Bulk_Operations", func(t *testing.T) {
			helpers.TestBulkOperations(t, harness)
		})

		// Test async operations
		t.Run("Async_Operations", func(t *testing.T) {
			helpers.TestAsyncOperations(t, harness)
		})
	})

	t.Run("Edge_Cases", func(t *testing.T) {
		harness := helpers.NewTestHarness(t)
		defer harness.Cleanup()

		// Test bulk operation edge cases
		t.Run("Bulk_Operation_Edge_Cases", func(t *testing.T) {
			helpers.TestBulkOperationEdgeCases(t, harness)
		})

		// Test keyboard shortcut regression
		t.Run("Keyboard_Shortcut_Regression", func(t *testing.T) {
			helpers.TestKeyboardShortcutRegression(t, harness)
		})

		// Test keyboard shortcut combinations
		t.Run("Keyboard_Shortcut_Combinations", func(t *testing.T) {
			helpers.TestKeyboardShortcutCombinations(t, harness)
		})
	})

	t.Run("Async_Operation_Advanced", func(t *testing.T) {
		harness := helpers.NewTestHarness(t)
		defer harness.Cleanup()

		// Test async operation cancellation
		t.Run("Async_Operation_Cancellation", func(t *testing.T) {
			helpers.TestAsyncOperationCancellation(t, harness)
		})

		// Test async operation timeout
		t.Run("Async_Operation_Timeout", func(t *testing.T) {
			helpers.TestAsyncOperationTimeout(t, harness)
		})

		// Test async operation error handling
		t.Run("Async_Operation_Error_Handling", func(t *testing.T) {
			helpers.TestAsyncOperationErrorHandling(t, harness)
		})

		// Test async operation concurrency
		t.Run("Async_Operation_Concurrency", func(t *testing.T) {
			helpers.TestAsyncOperationConcurrency(t, harness)
		})
	})

	t.Run("Visual_Advanced", func(t *testing.T) {
		harness := helpers.NewTestHarness(t)
		defer harness.Cleanup()

		// Test visual state changes
		t.Run("Visual_State_Changes", func(t *testing.T) {
			helpers.TestVisualStateChanges(t, harness)
		})

		// Test responsive layout
		t.Run("Responsive_Layout", func(t *testing.T) {
			helpers.TestResponsiveLayout(t, harness)
		})

		// Test focus indicators
		t.Run("Focus_Indicators", func(t *testing.T) {
			helpers.TestFocusIndicators(t, harness)
		})

		// Test color schemes
		t.Run("Color_Schemes", func(t *testing.T) {
			helpers.TestColorSchemes(t, harness)
		})

		// Test accessibility features
		t.Run("Accessibility_Features", func(t *testing.T) {
			helpers.TestAccessibilityFeatures(t, harness)
		})
	})
}

// BenchmarkSuite runs performance benchmarks
func BenchmarkSuite(b *testing.B) {
	b.Run("Bulk_Operations", func(b *testing.B) {
		harness := helpers.NewTestHarness(b)
		defer harness.Cleanup()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Benchmark bulk operations
			harness.App.SetSelectedMessages([]string{"msg_1", "msg_2", "msg_3"})
			harness.App.ExecuteBulkOperation("archive")
		}
	})

	b.Run("Keyboard_Shortcuts", func(b *testing.B) {
		harness := helpers.NewTestHarness(b)
		defer harness.Cleanup()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Benchmark keyboard shortcut execution
			harness.SimulateKeyEvent(tcell.KeyCtrlA, 0, tcell.ModCtrl)
			harness.App.HandleKeyEvent(harness.SimulateKeyEvent(tcell.KeyCtrlA, 0, tcell.ModCtrl))
		}
	})

	b.Run("Async_Operations", func(b *testing.B) {
		harness := helpers.NewTestHarness(b)
		defer harness.Cleanup()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Benchmark async operation startup
			harness.App.LoadMessagesAsync()
		}
	})
}

// TestIntegration runs integration tests
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("Gmail_API_Integration", func(t *testing.T) {
		// Integration tests with real Gmail API (using VCR)
		// This would test the actual service implementations
	})

	t.Run("End_to_End_Workflows", func(t *testing.T) {
		// Test complete user workflows
		// This would test the full application flow
	})
}

// TestPropertyBased runs property-based tests
func TestPropertyBased(t *testing.T) {
	t.Run("Keyboard_Interaction_Properties", func(t *testing.T) {
		// Property-based tests for keyboard interactions
		// This would use quick.Check to test invariants
	})

	t.Run("State_Transition_Properties", func(t *testing.T) {
		// Property-based tests for state transitions
		// This would test that certain properties always hold
	})
}