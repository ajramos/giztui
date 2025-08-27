package helpers

import (
	"testing"
)

// TestAsyncOperationsFramework demonstrates the async operations testing framework
func TestAsyncOperationsFramework(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	t.Run("AsyncOperations", func(t *testing.T) {
		RunAsyncOperationsTests(t, harness)
	})

	t.Run("AsyncCancellation", func(t *testing.T) {
		RunAsyncOperationCancellationTests(t, harness)
	})

	t.Run("AsyncTimeouts", func(t *testing.T) {
		RunAsyncOperationTimeoutTests(t, harness)
	})
}
