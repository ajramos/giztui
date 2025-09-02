package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/services"
	"github.com/ajramos/giztui/internal/services/mocks"
	"github.com/ajramos/giztui/internal/tui"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

	// Create test configuration
	testConfig := &config.Config{
		Keys: config.DefaultKeyBindings(),
	}

	// Create minimal Gmail client for testing
	// Note: This will have nil service, but that's okay for pure TUI testing
	testClient := gmail.NewClient(nil)

	// Create test app with minimal dependencies
	// The app will initialize its own services via initServices()
	testApp := tui.NewApp(testClient, nil, nil, testConfig)

	// Note: We can't override the app's screen directly, so we'll use the simulation screen
	// independently for UI component testing

	return &TestHarness{
		Screen:     screen,
		App:        testApp,
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
		// Send event to app if available (this will work once app event handling is ready)
		if h.App != nil {
			h.App.QueueEvent(event)
		}
	}
}

// SimulateTyping simulates typing a string of characters
func (h *TestHarness) SimulateTyping(text string) {
	for _, ch := range text {
		event := h.SimulateKeyEvent(tcell.KeyRune, ch, tcell.ModNone)
		if h.App != nil {
			h.App.QueueEvent(event)
		}
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
		Messages:      h.GenerateTestMessages(10),
		NextPageToken: "",
	}, nil)
}

// Advanced testing methods for Phase 4

// WaitForUIUpdate waits for UI updates to complete
func (h *TestHarness) WaitForUIUpdate(timeout time.Duration) bool {
	return h.WaitForCondition(func() bool {
		// Allow time for any queued UI updates to process
		time.Sleep(10 * time.Millisecond)
		return true
	}, timeout)
}

// AssertScreenRegion checks content in a specific screen region
func (h *TestHarness) AssertScreenRegion(t *testing.T, x, y, width, height int, expectedText string) {
	content := h.GetScreenRegion(x, y, width, height)
	assert.Contains(t, content, expectedText)
}

// GetScreenRegion captures content from a specific screen region
func (h *TestHarness) GetScreenRegion(x, y, width, height int) string {
	var output string
	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			char, _, _, _ := h.Screen.GetContent(col, row)
			output += string(char)
		}
		output += "\n"
	}
	return output
}

// SimulateMouseClick simulates a mouse click at specified coordinates
func (h *TestHarness) SimulateMouseClick(x, y int) *tcell.EventMouse {
	event := tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone)
	if h.App != nil {
		h.App.QueueEvent(event)
	}
	return event
}

// GetScreenSnapshot returns a structured snapshot of the screen for comparison
func (h *TestHarness) GetScreenSnapshot() *ScreenSnapshot {
	width, height := h.Screen.Size()
	cells := make([][]ScreenCell, height)

	for y := 0; y < height; y++ {
		cells[y] = make([]ScreenCell, width)
		for x := 0; x < width; x++ {
			char, _, style, _ := h.Screen.GetContent(x, y)
			cells[y][x] = ScreenCell{
				Char:  char,
				Style: style,
			}
		}
	}

	return &ScreenSnapshot{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}

// ScreenSnapshot represents a snapshot of the screen for testing
type ScreenSnapshot struct {
	Width  int
	Height int
	Cells  [][]ScreenCell
}

// ScreenCell represents a single cell in the terminal screen
type ScreenCell struct {
	Char  rune
	Style tcell.Style
}

// Equals compares two screen snapshots for equality
func (s *ScreenSnapshot) Equals(other *ScreenSnapshot) bool {
	if s.Width != other.Width || s.Height != other.Height {
		return false
	}

	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			if s.Cells[y][x] != other.Cells[y][x] {
				return false
			}
		}
	}

	return true
}

// GetContentString returns the snapshot as a readable string
func (s *ScreenSnapshot) GetContentString() string {
	var output string
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			output += string(s.Cells[y][x].Char)
		}
		output += "\n"
	}
	return output
}
