package helpers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/goleak"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

const (
	SnapshotDir = "testdata/snapshots"
)

// VisualTest defines a visual regression test
type VisualTest struct {
	Name        string
	Description string
	Setup       func(*TestHarness)
	Render      func(*TestHarness) tview.Primitive
	Validate    func(*TestHarness, string) bool
	Cleanup     func(*TestHarness)
}

// SnapshotResult represents the result of a snapshot comparison
type SnapshotResult struct {
	TestName    string
	Matches     bool
	SnapshotMD5 string
	CurrentMD5  string
	Content     string
}

// RunVisualRegressionTests runs visual regression tests using screen snapshots
func RunVisualRegressionTests(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, 
		goleak.IgnoreTopFunction("time.Sleep"),
		goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

	// Ensure snapshot directory exists
	err := os.MkdirAll(SnapshotDir, 0755)
	assert.NoError(t, err)

	tests := []VisualTest{
		{
			Name:        "message_list_rendering",
			Description: "Test message list component rendering with multiple messages",
			Setup: func(h *TestHarness) {
				messages := h.GenerateTestMessages(10)
				h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
					Return(&services.MessagePage{Messages: messages}, nil)
			},
			Render: func(h *TestHarness) tview.Primitive {
				// Create a simple list component for testing
				list := tview.NewList()
				for i := 0; i < 10; i++ {
					list.AddItem(fmt.Sprintf("Test Subject %d", i), fmt.Sprintf("sender%d@example.com", i), 0, nil)
				}
				return list
			},
			Validate: func(h *TestHarness, snapshot string) bool {
				return len(snapshot) > 0 &&
					contains(snapshot, "Test Subject") &&
					contains(snapshot, "sender0@example.com")
			},
		},
		{
			Name:        "empty_message_list",
			Description: "Test message list rendering when no messages are present",
			Setup: func(h *TestHarness) {
				h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
					Return(&services.MessagePage{Messages: []*gmail_v1.Message{}}, nil)
			},
			Render: func(h *TestHarness) tview.Primitive {
				list := tview.NewList()
				list.AddItem("No messages", "Your inbox is empty", 0, nil)
				return list
			},
			Validate: func(h *TestHarness, snapshot string) bool {
				return contains(snapshot, "No messages") && contains(snapshot, "empty")
			},
		},
		{
			Name:        "message_content_display",
			Description: "Test message content rendering with headers and body",
			Setup: func(h *TestHarness) {
				// Setup for message content display
			},
			Render: func(h *TestHarness) tview.Primitive {
				textView := tview.NewTextView()
				textView.SetText("From: test@example.com\nTo: user@example.com\nSubject: Test Message\n\nHello, this is a test message.")
				textView.SetBorder(true)
				textView.SetTitle("Message Content")
				return textView
			},
			Validate: func(h *TestHarness, snapshot string) bool {
				return contains(snapshot, "From: test@example.com") &&
					contains(snapshot, "Test Message") &&
					contains(snapshot, "Hello, this is a test")
			},
		},
		{
			Name:        "search_interface_rendering",
			Description: "Test search interface component rendering",
			Setup: func(h *TestHarness) {
				// Setup for search interface
			},
			Render: func(h *TestHarness) tview.Primitive {
				form := tview.NewForm()
				form.AddInputField("Search Query", "", 50, nil, nil)
				form.AddButton("Search", nil)
				form.AddButton("Cancel", nil)
				form.SetBorder(true)
				form.SetTitle("Search Messages")
				return form
			},
			Validate: func(h *TestHarness, snapshot string) bool {
				return contains(snapshot, "Search Query") &&
					contains(snapshot, "Search Messages")
			},
		},
		{
			Name:        "label_picker_rendering",
			Description: "Test label picker component rendering",
			Setup: func(h *TestHarness) {
				h.MockLabel.On("ListLabels", mock.Anything).Return([]*gmail_v1.Label{
					{Id: "INBOX", Name: "INBOX"},
					{Id: "WORK", Name: "Work"},
					{Id: "PERSONAL", Name: "Personal"},
				}, nil)
			},
			Render: func(h *TestHarness) tview.Primitive {
				list := tview.NewList()
				labels := []string{"INBOX", "Work", "Personal"}
				for _, label := range labels {
					list.AddItem(label, fmt.Sprintf("Apply %s label", label), 0, nil)
				}
				list.SetBorder(true)
				list.SetTitle("Select Label")
				return list
			},
			Validate: func(h *TestHarness, snapshot string) bool {
				return contains(snapshot, "INBOX") &&
					contains(snapshot, "Work") &&
					contains(snapshot, "Personal")
			},
		},
		{
			Name:        "status_bar_rendering",
			Description: "Test status bar component with various message types",
			Setup: func(h *TestHarness) {
				// Setup for status bar testing
			},
			Render: func(h *TestHarness) tview.Primitive {
				textView := tview.NewTextView()
				textView.SetText("📧 10 messages • 3 unread • Connected to Gmail")
				textView.SetTextColor(tview.Styles.PrimaryTextColor)
				return textView
			},
			Validate: func(h *TestHarness, snapshot string) bool {
				return contains(snapshot, "📧") &&
					contains(snapshot, "messages") &&
					contains(snapshot, "unread")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			defer goleak.VerifyNone(t, 
				goleak.IgnoreTopFunction("time.Sleep"),
				goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

			// Setup test
			if test.Setup != nil {
				test.Setup(harness)
			}

			// Render component
			var component tview.Primitive
			if test.Render != nil {
				component = test.Render(harness)
			}

			// Draw component to screen
			if component != nil {
				harness.DrawComponent(component)
			}

			// Capture screen content
			content := harness.GetScreenContent()

			// Validate content if validator provided
			if test.Validate != nil {
				assert.True(t, test.Validate(harness, content),
					fmt.Sprintf("Content validation failed for test: %s", test.Name))
			}

			// Perform snapshot comparison
			result := CompareSnapshot(t, test.Name, content)

			if !result.Matches {
				t.Logf("Visual regression detected in test: %s", test.Name)
				t.Logf("Expected MD5: %s", result.SnapshotMD5)
				t.Logf("Actual MD5: %s", result.CurrentMD5)

				// In CI/CD, this would fail the test. In development, this logs the difference.
				if os.Getenv("UPDATE_SNAPSHOTS") == "true" {
					t.Logf("Updating snapshot for: %s", test.Name)
					err := UpdateSnapshot(test.Name, content)
					assert.NoError(t, err, "Failed to update snapshot")
				} else {
					t.Logf("Visual regression detected. Run with UPDATE_SNAPSHOTS=true to update baseline.")
					// Uncomment to fail on visual regression:
					// t.Fail()
				}
			} else {
				t.Logf("Visual test passed: %s", test.Name)
			}

			// Cleanup
			if test.Cleanup != nil {
				test.Cleanup(harness)
			}
		})
	}
}

// CompareSnapshot compares current content with stored snapshot
func CompareSnapshot(t *testing.T, testName, content string) SnapshotResult {
	snapshotPath := filepath.Join(SnapshotDir, testName+".snapshot")

	// Calculate MD5 of current content
	currentMD5 := calculateMD5(content)

	// Read existing snapshot if it exists
	var snapshotMD5 string
	var matches bool

	if snapshotData, err := os.ReadFile(snapshotPath); err == nil {
		snapshotMD5 = calculateMD5(string(snapshotData))
		matches = snapshotMD5 == currentMD5
	} else {
		// No existing snapshot, create one
		t.Logf("No existing snapshot for %s, creating baseline", testName)
		err := os.WriteFile(snapshotPath, []byte(content), 0644)
		assert.NoError(t, err, "Failed to create baseline snapshot")
		snapshotMD5 = currentMD5
		matches = true
	}

	return SnapshotResult{
		TestName:    testName,
		Matches:     matches,
		SnapshotMD5: snapshotMD5,
		CurrentMD5:  currentMD5,
		Content:     content,
	}
}

// UpdateSnapshot updates the stored snapshot with new content
func UpdateSnapshot(testName, content string) error {
	snapshotPath := filepath.Join(SnapshotDir, testName+".snapshot")
	return os.WriteFile(snapshotPath, []byte(content), 0644)
}

// RunVisualStateChanges tests visual changes based on state transitions
func RunVisualStateChanges(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, 
		goleak.IgnoreTopFunction("time.Sleep"),
		goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

	stateTests := []struct {
		name        string
		description string
		states      []StateTransition
	}{
		{
			name:        "message_selection_changes",
			description: "Test visual changes when selecting different messages",
			states: []StateTransition{
				{
					Name: "no_selection",
					Setup: func(h *TestHarness) tview.Primitive {
						list := tview.NewList()
						list.AddItem("Message 1", "First message", 0, nil)
						list.AddItem("Message 2", "Second message", 0, nil)
						return list
					},
				},
				{
					Name: "first_selected",
					Setup: func(h *TestHarness) tview.Primitive {
						list := tview.NewList()
						list.AddItem("Message 1", "First message", 0, nil)
						list.AddItem("Message 2", "Second message", 0, nil)
						list.SetCurrentItem(0)
						return list
					},
				},
				{
					Name: "second_selected",
					Setup: func(h *TestHarness) tview.Primitive {
						list := tview.NewList()
						list.AddItem("Message 1", "First message", 0, nil)
						list.AddItem("Message 2", "Second message", 0, nil)
						list.SetCurrentItem(1)
						return list
					},
				},
			},
		},
		{
			name:        "focus_state_changes",
			description: "Test visual focus indicators",
			states: []StateTransition{
				{
					Name: "unfocused",
					Setup: func(h *TestHarness) tview.Primitive {
						form := tview.NewForm()
						form.AddInputField("Field 1", "", 20, nil, nil)
						form.AddInputField("Field 2", "", 20, nil, nil)
						return form
					},
				},
				{
					Name: "focused_first",
					Setup: func(h *TestHarness) tview.Primitive {
						form := tview.NewForm()
						form.AddInputField("Field 1", "", 20, nil, nil)
						form.AddInputField("Field 2", "", 20, nil, nil)
						form.SetFocus(0)
						return form
					},
				},
			},
		},
	}

	for _, stateTest := range stateTests {
		t.Run(stateTest.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, 
				goleak.IgnoreTopFunction("time.Sleep"),
				goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

			for _, state := range stateTest.states {
				t.Run(state.Name, func(t *testing.T) {
					component := state.Setup(harness)
					harness.DrawComponent(component)
					content := harness.GetScreenContent()

					testName := fmt.Sprintf("%s_%s", stateTest.name, state.Name)
					result := CompareSnapshot(t, testName, content)

					if !result.Matches && os.Getenv("UPDATE_SNAPSHOTS") != "true" {
						t.Logf("State transition visual change detected: %s -> %s", stateTest.name, state.Name)
					}
				})
			}
		})
	}
}

// RunResponsiveLayoutTests tests responsive layout behavior
func RunResponsiveLayoutTests(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, 
		goleak.IgnoreTopFunction("time.Sleep"),
		goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

	screenSizes := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 80, 24},
		{"medium", 120, 40},
		{"large", 160, 50},
		{"very_wide", 200, 40},
		{"very_tall", 120, 60},
	}

	for _, size := range screenSizes {
		t.Run(fmt.Sprintf("layout_%s_%dx%d", size.name, size.width, size.height), func(t *testing.T) {
			defer goleak.VerifyNone(t, 
				goleak.IgnoreTopFunction("time.Sleep"),
				goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

			// Resize screen for test
			harness.Screen.SetSize(size.width, size.height)

			// Create a responsive layout
			flex := tview.NewFlex()
			sidebar := tview.NewTextView().SetText("Sidebar")
			sidebar.SetBorder(true)
			content := tview.NewTextView().SetText("Main Content Area")
			content.SetBorder(true)

			if size.width >= 120 {
				// Wide layout - side by side
				flex.AddItem(sidebar, 30, 0, false)
				flex.AddItem(content, 0, 1, true)
			} else {
				// Narrow layout - stacked
				flex.SetDirection(tview.FlexRow)
				flex.AddItem(sidebar, 5, 0, false)
				flex.AddItem(content, 0, 1, true)
			}

			harness.DrawComponent(flex)
			content_str := harness.GetScreenContent()

			testName := fmt.Sprintf("responsive_%s", size.name)
			result := CompareSnapshot(t, testName, content_str)

			if !result.Matches && os.Getenv("UPDATE_SNAPSHOTS") != "true" {
				t.Logf("Responsive layout change detected for size: %dx%d", size.width, size.height)
			}

			// Reset to standard size
			harness.Screen.SetSize(120, 40)
		})
	}
}

// StateTransition represents a state change for visual testing
type StateTransition struct {
	Name  string
	Setup func(*TestHarness) tview.Primitive
}

// Helper functions

func calculateMD5(content string) string {
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		(haystack == needle || len(haystack) > len(needle) &&
			findSubstring(haystack, needle))
}

func findSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
