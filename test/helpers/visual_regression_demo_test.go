package helpers

import (
	"testing"
)

// TestVisualRegressionFramework demonstrates the visual regression testing framework
func TestVisualRegressionFramework(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	t.Run("VisualRegression", func(t *testing.T) {
		RunVisualRegressionTests(t, harness)
	})

	t.Run("VisualStateChanges", func(t *testing.T) {
		RunVisualStateChanges(t, harness)
	})

	t.Run("ResponsiveLayout", func(t *testing.T) {
		RunResponsiveLayoutTests(t, harness)
	})
}