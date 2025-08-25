package test

import (
	"testing"

	"github.com/ajramos/gmail-tui/test/helpers"
)

// Main test suite runner that orchestrates all test categories
func TestMain(m *testing.M) {
	// Run tests
	m.Run()
}

// TestFrameworkBasics tests basic framework functionality
func TestFrameworkBasics(t *testing.T) {
	// This will run the TestFrameworkIntegration test from helpers/simple_test.go
	// The test framework is now integrated and ready for gradual enhancement
	t.Run("BasicFrameworkValidation", func(t *testing.T) {
		harness := helpers.NewTestHarness(t)
		defer harness.Cleanup()

		// Just verify we can create a test harness successfully
		if harness == nil {
			t.Fatal("Failed to create test harness")
		}
	})
}