package helpers

import (
	"testing"
)

// TestIntegrationFramework demonstrates the integration testing framework
func TestIntegrationFramework(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	t.Run("IntegrationSuites", func(t *testing.T) {
		RunIntegrationTests(t, harness)
	})

	t.Run("ServiceIntegration", func(t *testing.T) {
		RunServiceIntegrationTests(t, harness)
	})

	t.Run("ErrorHandlingIntegration", func(t *testing.T) {
		RunErrorHandlingIntegrationTests(t, harness)
	})

	t.Run("PerformanceIntegration", func(t *testing.T) {
		RunPerformanceIntegrationTests(t, harness)
	})
}
