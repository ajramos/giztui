package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/ajramos/gmail-tui/internal/tui"
	"github.com/derailed/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"
)

// TestHarness provides utilities for testing TUI components
type TestHarness struct {
	Screen     tcell.SimulationScreen
	App        *tui.App
	MockEmail  *services.MockEmailService
	MockAI     *services.MockAIService
	MockLabel  *services.MockLabelService
	MockCache  *services.MockCacheService
	MockRepo   *services.MockMessageRepository
	MockSlack  *services.MockSlackService
	MockSearch *services.MockSearchService
	MockPrompt *services.MockPromptService
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
	mockEmail := &services.MockEmailService{}
	mockAI := &services.MockAIService{}
	mockLabel := &services.MockLabelService{}
	mockCache := &services.MockCacheService{}
	mockRepo := &services.MockMessageRepository{}
	mockSlack := &services.MockSlackService{}
	mockSearch := &services.MockSearchService{}
	mockPrompt := &services.MockPromptService{}

	// Create app with mocked services
	app := tui.NewTestApp(ctx, &services.ServiceFactory{
		EmailService:  mockEmail,
		AIService:     mockAI,
		LabelService:  mockLabel,
		CacheService:  mockCache,
		Repository:    mockRepo,
		SlackService:  mockSlack,
		SearchService: mockSearch,
		PromptService: mockPrompt,
	})

	return &TestHarness{
		Screen:     screen,
		App:        app,
		MockEmail:  mockEmail,
		MockAI:     mockAI,
		MockLabel:  mockLabel,
		MockCache:  mockCache,
		MockRepo:   mockRepo,
		MockSlack:  mockSlack,
		MockSearch: mockSearch,
		MockPrompt: mockPrompt,
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
		h.App.HandleKeyEvent(event)
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
func (h *TestHarness) GenerateTestMessages(count int) []*services.Message {
	messages := make([]*services.Message, count)
	for i := 0; i < count; i++ {
		messages[i] = &services.Message{
			ID:      fmt.Sprintf("msg_%d", i),
			Subject: fmt.Sprintf("Test Subject %d", i),
			From:    fmt.Sprintf("sender%d@example.com", i),
			Date:    time.Now().Add(-time.Duration(i) * time.Hour),
			Read:    i%2 == 0, // Alternate read/unread
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