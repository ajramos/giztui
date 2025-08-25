package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/ajramos/gmail-tui/internal/services/mocks"
	"github.com/ajramos/gmail-tui/internal/tui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// TestHarness provides utilities for testing TUI components
type TestHarness struct {
	Screen     tcell.SimulationScreen
	App        *tui.App
	MockEmail  *mocks.EmailService
	MockAI     *mocks.AIService
	MockLabel  *mocks.LabelService
	MockCache  *mocks.CacheService
	MockRepo   *mocks.MessageRepository
	MockSearch *mocks.SearchService
	Ctx        context.Context
	Cancel     context.CancelFunc
}

// NewTestHarness creates a new test harness with mocked services
func NewTestHarness(t *testing.T) *TestHarness {
	// Create simulation screen
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	require.NoError(t, err)
	screen.SetSize(120, 40) // Standard terminal size for testing

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Create mocked services
	mockEmail := &mocks.EmailService{}
	mockAI := &mocks.AIService{}
	mockLabel := &mocks.LabelService{}
	mockCache := &mocks.CacheService{}
	mockRepo := &mocks.MessageRepository{}
	mockSearch := &mocks.SearchService{}

	// Note: We'll create a simplified test app since NewTestApp may not exist yet
	// This will need to be implemented in the TUI package or we'll use a different approach
	var app *tui.App
	// TODO: Implement proper test app creation after validating existing functionality

	return &TestHarness{
		Screen:     screen,
		App:        app,
		MockEmail:  mockEmail,
		MockAI:     mockAI,
		MockLabel:  mockLabel,
		MockCache:  mockCache,
		MockRepo:   mockRepo,
		MockSearch: mockSearch,
		Ctx:        ctx,
		Cancel:     cancel,
	}
}

// Cleanup cleans up test resources
func (h *TestHarness) Cleanup() {
	h.Cancel()
	h.Screen.Fini()
}

// DrawComponent draws a component to the simulation screen
func (h *TestHarness) DrawComponent(component tview.Primitive) {
	component.SetRect(0, 0, 120, 40)
	component.Draw(h.Screen)
}

// SimulateKeyEvent simulates a key press event
func (h *TestHarness) SimulateKeyEvent(key tcell.Key, ch rune, mod tcell.ModMask) *tcell.EventKey {
	event := tcell.NewEventKey(key, ch, mod)
	return event
}

// SimulateKeySequence simulates a sequence of key presses
func (h *TestHarness) SimulateKeySequence(keys []tcell.Key) {
	for _, key := range keys {
		event := h.SimulateKeyEvent(key, 0, tcell.ModNone)
		// TODO: Implement key event handling when App methods are available
		_ = event
	}
}

// GetScreenContent captures the current screen content as a string
func (h *TestHarness) GetScreenContent() string {
	width, height := h.Screen.Size()
	var output string

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			char, _, _, _ := h.Screen.GetContent(x, y)
			output += string(char)
		}
		output += "\n"
	}
	return output
}

// AssertScreenContains checks if the screen contains specific text
func (h *TestHarness) AssertScreenContains(t *testing.T, expectedText string) {
	content := h.GetScreenContent()
	assert.Contains(t, content, expectedText)
}

// AssertScreenNotContains checks if the screen doesn't contain specific text
func (h *TestHarness) AssertScreenNotContains(t *testing.T, expectedText string) {
	content := h.GetScreenContent()
	assert.NotContains(t, content, expectedText)
}

// WaitForCondition waits for a condition to be true with timeout
func (h *TestHarness) WaitForCondition(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// GenerateTestMessages creates test message data
func (h *TestHarness) GenerateTestMessages(count int) []*gmail_v1.Message {
	messages := make([]*gmail_v1.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = &gmail_v1.Message{
			Id:       fmt.Sprintf("msg_%d", i),
			ThreadId: fmt.Sprintf("thread_%d", i),
		}
	}
	return messages
}

// SetupMockExpectations sets up common mock expectations for testing
func (h *TestHarness) SetupMockExpectations() {
	// Setup common mock expectations here
	h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).Return(&services.MessagePage{
		Messages: h.GenerateTestMessages(10),
		NextPageToken: "",
	}, nil)
}