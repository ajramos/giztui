package helpers

import (
	"testing"

	"github.com/ajramos/gmail-tui/internal/tui"
	"github.com/derailed/tview"
	"github.com/gkampitakis/go-snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// VisualTest defines a visual regression test
type VisualTest struct {
	Name        string
	Setup       func(*TestHarness)
	Component   tview.Primitive
	SnapshotKey string
	Validate    func(*TestHarness, string) bool
}

// TestVisualRegression runs visual regression tests using snapshots
func TestVisualRegression(t *testing.T, harness *TestHarness) {
	tests := []VisualTest{
		{
			Name: "message_list_rendering",
			Setup: func(h *TestHarness) {
				// Setup test messages
				h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).Return(&services.MessagePage{
					Messages:      h.GenerateTestMessages(10),
					NextPageToken: "",
				}, nil)
			},
			Component:   harness.App.GetMessageListComponent(),
			SnapshotKey: "message_list_10_messages",
			Validate: func(h *TestHarness, snapshot string) bool {
				// Validate snapshot contains expected content
				return assert.Contains(t, snapshot, "Test Subject") &&
					assert.Contains(t, snapshot, "sender@example.com")
			},
		},
		{
			Name: "empty_message_list",
			Setup: func(h *TestHarness) {
				// Setup empty message list
				h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).Return(&services.MessagePage{
					Messages:      []*services.Message{},
					NextPageToken: "",
				}, nil)
			},
			Component:   harness.App.GetMessageListComponent(),
			SnapshotKey: "message_list_empty",
			Validate: func(h *TestHarness, snapshot string) bool {
				// Validate empty state message
				return assert.Contains(t, snapshot, "No messages") ||
					assert.Contains(t, snapshot, "Empty")
			},
		},
		{
			Name: "message_detail_view",
			Setup: func(h *TestHarness) {
				// Setup message detail
				h.App.SetCurrentMessageID("msg_1")
				h.MockRepo.On("GetMessage", mock.Anything, "msg_1").Return(&services.Message{
					ID:      "msg_1",
					Subject: "Test Subject",
					From:    "sender@example.com",
					Body:    "This is a test message body with some content.",
				}, nil)
			},
			Component:   harness.App.GetMessageDetailComponent(),
			SnapshotKey: "message_detail_view",
			Validate: func(h *TestHarness, snapshot string) bool {
				return assert.Contains(t, snapshot, "Test Subject") &&
					assert.Contains(t, snapshot, "sender@example.com") &&
					assert.Contains(t, snapshot, "test message body")
			},
		},
		{
			Name: "label_management_view",
			Setup: func(h *TestHarness) {
				// Setup labels
				h.MockLabel.On("ListLabels", mock.Anything).Return([]*gmail_v1.Label{
					{Id: "label_1", Name: "Important"},
					{Id: "label_2", Name: "Work"},
					{Id: "label_3", Name: "Personal"},
				}, nil)
			},
			Component:   harness.App.GetLabelManagementComponent(),
			SnapshotKey: "label_management_view",
			Validate: func(h *TestHarness, snapshot string) bool {
				return assert.Contains(t, snapshot, "Important") &&
					assert.Contains(t, snapshot, "Work") &&
					assert.Contains(t, snapshot, "Personal")
			},
		},
		{
			Name: "ai_summary_panel",
			Setup: func(h *TestHarness) {
				// Setup AI summary
				h.App.SetCurrentMessageID("msg_1")
				h.App.SetAISummary("msg_1", "This is an AI-generated summary of the message content.")
			},
			Component:   harness.App.GetAISummaryComponent(),
			SnapshotKey: "ai_summary_panel",
			Validate: func(h *TestHarness, snapshot string) bool {
				return assert.Contains(t, snapshot, "AI-generated summary") &&
					assert.Contains(t, snapshot, "message content")
			},
		},
		{
			Name: "search_results_view",
			Setup: func(h *TestHarness) {
				// Setup search results
				h.MockSearch.On("Search", mock.Anything, "test query", mock.Anything).Return(&services.SearchResult{
					Messages: h.GenerateTestMessages(5),
					Query:    "test query",
				}, nil)
			},
			Component:   harness.App.GetSearchResultsComponent(),
			SnapshotKey: "search_results_view",
			Validate: func(h *TestHarness, snapshot string) bool {
				return assert.Contains(t, snapshot, "test query") &&
					assert.Contains(t, snapshot, "Test Subject")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Setup test
			if test.Setup != nil {
				test.Setup(harness)
			}

			// Draw component to screen
			harness.DrawComponent(test.Component)

			// Capture screen content
			snapshot := harness.GetScreenContent()

			// Validate snapshot content
			if test.Validate != nil {
				assert.True(t, test.Validate(harness, snapshot), "Snapshot validation failed: %s", test.Name)
			}

			// Compare with golden file using go-snaps
			snaps.MatchSnapshot(t, snapshot, test.SnapshotKey)
		})
	}
}

// TestVisualStateChanges tests visual changes based on state
func TestVisualStateChanges(t *testing.T, harness *TestHarness) {
	t.Run("loading_state_indicator", func(t *testing.T) {
		// Test loading state
		harness.App.SetLoadingState(true)
		harness.DrawComponent(harness.App.GetStatusComponent())
		
		loadingSnapshot := harness.GetScreenContent()
		assert.Contains(t, loadingSnapshot, "Loading") || assert.Contains(t, loadingSnapshot, "⏳")
		
		// Test non-loading state
		harness.App.SetLoadingState(false)
		harness.DrawComponent(harness.App.GetStatusComponent())
		
		normalSnapshot := harness.GetScreenContent()
		assert.NotContains(t, normalSnapshot, "Loading")
		
		// Compare snapshots
		snaps.MatchSnapshot(t, loadingSnapshot, "loading_state")
		snaps.MatchSnapshot(t, normalSnapshot, "normal_state")
	})

	t.Run("error_state_display", func(t *testing.T) {
		// Test error state
		harness.App.GetErrorHandler().ShowError(harness.Ctx, "Test error message")
		harness.DrawComponent(harness.App.GetErrorComponent())
		
		errorSnapshot := harness.GetScreenContent()
		assert.Contains(t, errorSnapshot, "Test error message")
		
		// Test success state
		harness.App.GetErrorHandler().ShowSuccess(harness.Ctx, "Test success message")
		harness.DrawComponent(harness.App.GetErrorComponent())
		
		successSnapshot := harness.GetScreenContent()
		assert.Contains(t, successSnapshot, "Test success message")
		
		// Compare snapshots
		snaps.MatchSnapshot(t, errorSnapshot, "error_state")
		snaps.MatchSnapshot(t, successSnapshot, "success_state")
	})

	t.Run("selection_state_visual", func(t *testing.T) {
		// Test unselected state
		harness.App.SetCurrentMessageID("msg_1")
		harness.App.SetSelectionState(false)
		harness.DrawComponent(harness.App.GetMessageListComponent())
		
		unselectedSnapshot := harness.GetScreenContent()
		
		// Test selected state
		harness.App.SetSelectionState(true)
		harness.DrawComponent(harness.App.GetMessageListComponent())
		
		selectedSnapshot := harness.GetScreenContent()
		
		// Snapshots should be different
		assert.NotEqual(t, unselectedSnapshot, selectedSnapshot)
		
		// Save snapshots
		snaps.MatchSnapshot(t, unselectedSnapshot, "unselected_state")
		snaps.MatchSnapshot(t, selectedSnapshot, "selected_state")
	})
}

// TestResponsiveLayout tests layout changes at different screen sizes
func TestResponsiveLayout(t *testing.T, harness *TestHarness) {
	screenSizes := []struct {
		width  int
		height int
		name   string
	}{
		{80, 24, "small_terminal"},
		{120, 40, "medium_terminal"},
		{160, 50, "large_terminal"},
		{200, 60, "wide_terminal"},
	}

	for _, size := range screenSizes {
		t.Run(size.name, func(t *testing.T) {
			// Resize screen
			harness.Screen.SetSize(size.width, size.height)
			
			// Draw main layout
			harness.DrawComponent(harness.App.GetMainLayout())
			
			// Capture snapshot
			snapshot := harness.GetScreenContent()
			
			// Validate layout fits screen
			assert.Len(t, snapshot, size.height+1) // +1 for newlines
			
			// Save snapshot for this size
			snaps.MatchSnapshot(t, snapshot, size.name)
		})
	}
}

// TestFocusIndicators tests visual focus indicators
func TestFocusIndicators(t *testing.T, harness *TestHarness) {
	t.Run("focus_navigation", func(t *testing.T) {
		// Test focus on different components
		components := []struct {
			name     string
			component tview.Primitive
			focusKey tcell.Key
		}{
			{"message_list", harness.App.GetMessageListComponent(), tcell.KeyTab},
			{"message_detail", harness.App.GetMessageDetailComponent(), tcell.KeyTab},
			{"labels", harness.App.GetLabelManagementComponent(), tcell.KeyTab},
			{"ai_panel", harness.App.GetAISummaryComponent(), tcell.KeyTab},
		}

		for _, comp := range components {
			t.Run(comp.name, func(t *testing.T) {
				// Set focus to component
				harness.App.SetFocus(comp.component)
				
				// Draw component
				harness.DrawComponent(comp.component)
				
				// Capture focused state
				focusedSnapshot := harness.GetScreenContent()
				
				// Should show focus indicator
				assert.Contains(t, focusedSnapshot, "▶") || 
					assert.Contains(t, focusedSnapshot, ">") ||
					assert.Contains(t, focusedSnapshot, "●")
				
				// Save snapshot
				snaps.MatchSnapshot(t, focusedSnapshot, comp.name+"_focused")
			})
		}
	})
}

// TestColorSchemes tests different color schemes
func TestColorSchemes(t *testing.T, harness *TestHarness) {
	t.Run("default_colors", func(t *testing.T) {
		harness.App.SetColorScheme("default")
		harness.DrawComponent(harness.App.GetMainLayout())
		
		defaultSnapshot := harness.GetScreenContent()
		snaps.MatchSnapshot(t, defaultSnapshot, "default_color_scheme")
	})

	t.Run("high_contrast_colors", func(t *testing.T) {
		harness.App.SetColorScheme("high-contrast")
		harness.DrawComponent(harness.App.GetMainLayout())
		
		highContrastSnapshot := harness.GetScreenContent()
		snaps.MatchSnapshot(t, highContrastSnapshot, "high_contrast_color_scheme")
	})

	t.Run("dark_theme", func(t *testing.T) {
		harness.App.SetColorScheme("dark")
		harness.DrawComponent(harness.App.GetMainLayout())
		
		darkSnapshot := harness.GetScreenContent()
		snaps.MatchSnapshot(t, darkSnapshot, "dark_theme")
	})
}

// TestAccessibilityFeatures tests accessibility-related visual elements
func TestAccessibilityFeatures(t *testing.T, harness *TestHarness) {
	t.Run("keyboard_shortcuts_help", func(t *testing.T) {
		// Show help
		harness.App.ShowHelp()
		harness.DrawComponent(harness.App.GetHelpComponent())
		
		helpSnapshot := harness.GetScreenContent()
		
		// Should show keyboard shortcuts
		assert.Contains(t, helpSnapshot, "Ctrl+A") ||
			assert.Contains(t, helpSnapshot, "Select All")
		
		snaps.MatchSnapshot(t, helpSnapshot, "keyboard_shortcuts_help")
	})

	t.Run("status_bar_information", func(t *testing.T) {
		// Test status bar shows current context
		harness.App.SetCurrentView("messages")
		harness.DrawComponent(harness.App.GetStatusComponent())
		
		statusSnapshot := harness.GetScreenContent()
		
		// Should show current view
		assert.Contains(t, statusSnapshot, "messages") ||
			assert.Contains(t, statusSnapshot, "Messages")
		
		snaps.MatchSnapshot(t, statusSnapshot, "status_bar_messages_view")
	})
}