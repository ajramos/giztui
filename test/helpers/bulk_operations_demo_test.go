package helpers

import (
	"testing"
)

// TestBulkOperationsFramework demonstrates the bulk operations testing framework
func TestBulkOperationsFramework(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	t.Run("BulkOperations", func(t *testing.T) {
		RunBulkOperationsTests(t, harness)
	})

	t.Run("BulkEdgeCases", func(t *testing.T) {
		RunBulkOperationEdgeCasesTests(t, harness)
	})

	t.Run("BulkPerformance", func(t *testing.T) {
		RunBulkOperationPerformanceTests(t, harness)
	})
}
