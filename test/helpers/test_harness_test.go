package helpers

import (
	"fmt"
	"testing"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTestHarness_Creation tests basic test harness creation
func TestTestHarness_Creation(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Verify harness was created properly
	require.NotNil(t, harness)
	require.NotNil(t, harness.Screen)
	require.NotNil(t, harness.App)
	require.NotNil(t, harness.MockEmail)
	require.NotNil(t, harness.MockAI)
	require.NotNil(t, harness.MockLabel)
	require.NotNil(t, harness.MockCache)
	require.NotNil(t, harness.MockRepo)
	require.NotNil(t, harness.MockSearch)

	// Verify screen dimensions
	width, height := harness.Screen.Size()
	assert.Equal(t, 120, width)
	assert.Equal(t, 40, height)
}

// TestTestHarness_ComponentDrawing tests drawing components to the simulation screen
func TestTestHarness_ComponentDrawing(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Create a simple text view component
	textView := tview.NewTextView()
	textView.SetText("Hello, Test World!")

	// Draw the component to the screen
	harness.DrawComponent(textView)

	// Verify the content appears on screen
	harness.AssertScreenContains(t, "Hello, Test World!")
}

// TestTestHarness_KeyEvents tests key event simulation
func TestTestHarness_KeyEvents(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Test basic key event creation
	event := harness.SimulateKeyEvent(tcell.KeyEnter, 0, tcell.ModNone)
	assert.NotNil(t, event)
	assert.Equal(t, tcell.KeyEnter, event.Key())
}

// TestTestHarness_ScreenSnapshot tests screen snapshot functionality
func TestTestHarness_ScreenSnapshot(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Create a test component
	textView := tview.NewTextView()
	textView.SetText("Snapshot Test")
	harness.DrawComponent(textView)

	// Capture snapshot
	snapshot1 := harness.GetScreenSnapshot()
	require.NotNil(t, snapshot1)
	assert.Equal(t, 120, snapshot1.Width)
	assert.Equal(t, 40, snapshot1.Height)

	// Verify snapshot content
	content := snapshot1.GetContentString()
	assert.Contains(t, content, "Snapshot Test")

	// Create another identical snapshot
	snapshot2 := harness.GetScreenSnapshot()
	
	// Verify snapshots are equal
	assert.True(t, snapshot1.Equals(snapshot2))

	// Change screen content
	textView.SetText("Changed Content")
	harness.DrawComponent(textView)
	
	// Capture new snapshot
	snapshot3 := harness.GetScreenSnapshot()
	
	// Verify snapshots are different
	assert.False(t, snapshot1.Equals(snapshot3))
}

// TestTestHarness_WaitForCondition tests condition waiting functionality
func TestTestHarness_WaitForCondition(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Test condition that is immediately true
	result := harness.WaitForCondition(func() bool {
		return true
	}, 1*time.Second)
	assert.True(t, result)

	// Test condition that becomes true after a delay
	counter := 0
	result = harness.WaitForCondition(func() bool {
		counter++
		return counter > 3
	}, 1*time.Second)
	assert.True(t, result)
	assert.True(t, counter > 3)

	// Test condition that times out
	result = harness.WaitForCondition(func() bool {
		return false
	}, 100*time.Millisecond)
	assert.False(t, result)
}

// TestTestHarness_ScreenRegion tests regional screen content capture
func TestTestHarness_ScreenRegion(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Create a text view with specific content
	textView := tview.NewTextView()
	textView.SetText("Top Left\nBottom Right")
	harness.DrawComponent(textView)

	// Test region capture
	topRegion := harness.GetScreenRegion(0, 0, 10, 5)
	assert.Contains(t, topRegion, "Top Left")

	// Test assertion on region
	harness.AssertScreenRegion(t, 0, 0, 20, 10, "Top Left")
}

// TestTestHarness_MockSetup tests mock service setup
func TestTestHarness_MockSetup(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Setup common mock expectations
	harness.SetupMockExpectations()

	// Verify mock setup worked
	assert.NotNil(t, harness.MockRepo)
	
	// The mock should be ready to use (though we can't easily test the expectations
	// without triggering actual service calls)
}

// TestTestHarness_MessageGeneration tests test message generation
func TestTestHarness_MessageGeneration(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Generate test messages
	messages := harness.GenerateTestMessages(5)
	
	require.Len(t, messages, 5)
	
	// Verify message structure
	for i, msg := range messages {
		assert.Equal(t, fmt.Sprintf("msg_%d", i), msg.Id)
		assert.Equal(t, fmt.Sprintf("thread_%d", i), msg.ThreadId)
	}
}

// TestTestHarness_TypingSimulation tests typing simulation functionality
func TestTestHarness_TypingSimulation(t *testing.T) {
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// This test verifies that typing simulation doesn't crash
	// In a real scenario, this would be tested with an actual input field
	harness.SimulateTyping("hello world")

	// Test with special characters
	harness.SimulateTyping("test@example.com")
	
	// Test with unicode
	harness.SimulateTyping("Testing 🎯")
}