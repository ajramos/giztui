package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/uber-go/goleak"
)

// AsyncOperation defines an asynchronous operation to test
type AsyncOperation struct {
	Name        string
	Setup       func(*TestHarness)
	Execute     func(*TestHarness) error
	Validate    func(*TestHarness) bool
	Timeout     time.Duration
	ShouldBlock bool
}

// AsyncOperationResult represents the result of an async operation
type AsyncOperationResult struct {
	Success     bool
	Error       error
	Duration    time.Duration
	StateChange bool
}

// TestAsyncOperations runs comprehensive tests for asynchronous operations
func TestAsyncOperations(t *testing.T, harness *TestHarness) {
	// Ensure no goroutine leaks
	defer goleak.VerifyNone(t)

	operations := []AsyncOperation{
		{
			Name: "message_loading",
			Setup: func(h *TestHarness) {
				// Setup mock to return messages after delay
				h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						time.Sleep(100 * time.Millisecond)
					}).
					Return(&services.MessagePage{
						Messages:      h.GenerateTestMessages(50),
						NextPageToken: "",
					}, nil)
			},
			Execute: func(h *TestHarness) error {
				// Start async load
				return h.App.LoadMessagesAsync()
			},
			Validate: func(h *TestHarness) bool {
				// Wait for completion
				success := h.WaitForCondition(func() bool {
					return !h.App.IsLoading() && h.App.GetMessageCount() == 50
				}, 2*time.Second)
				
				return success
			},
			Timeout:     2 * time.Second,
			ShouldBlock: false,
		},
		{
			Name: "ai_summary_generation",
			Setup: func(h *TestHarness) {
				h.App.SetCurrentMessageID("msg_1")
				h.MockAI.On("GenerateSummary", mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						time.Sleep(200 * time.Millisecond)
					}).
					Return(&services.SummaryResult{
						Summary: "Generated summary",
					}, nil)
			},
			Execute: func(h *TestHarness) error {
				return h.App.GenerateAISummaryAsync("msg_1")
			},
			Validate: func(h *TestHarness) bool {
				success := h.WaitForCondition(func() bool {
					return h.App.GetAISummary("msg_1") == "Generated summary"
				}, 3*time.Second)
				
				return success
			},
			Timeout:     3 * time.Second,
			ShouldBlock: false,
		},
		{
			Name: "bulk_label_application",
			Setup: func(h *TestHarness) {
				h.App.SetSelectedMessages([]string{"msg_1", "msg_2", "msg_3"})
				h.MockLabel.On("ApplyLabel", mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						time.Sleep(50 * time.Millisecond)
					}).
					Return(nil)
			},
			Execute: func(h *TestHarness) error {
				return h.App.ApplyLabelToSelectedAsync("Important")
			},
			Validate: func(h *TestHarness) bool {
				success := h.WaitForCondition(func() bool {
					return h.App.GetLabelApplicationStatus() == "completed"
				}, 2*time.Second)
				
				return success
			},
			Timeout:     2 * time.Second,
			ShouldBlock: false,
		},
		{
			Name: "search_operation",
			Setup: func(h *TestHarness) {
				h.MockSearch.On("Search", mock.Anything, "test query", mock.Anything).
					Run(func(args mock.Arguments) {
						time.Sleep(150 * time.Millisecond)
					}).
					Return(&services.SearchResult{
						Messages: h.GenerateTestMessages(25),
						Query:    "test query",
					}, nil)
			},
			Execute: func(h *TestHarness) error {
				return h.App.SearchAsync("test query")
			},
			Validate: func(h *TestHarness) bool {
				success := h.WaitForCondition(func() bool {
					return h.App.GetSearchResults() != nil && len(h.App.GetSearchResults().Messages) == 25
				}, 2*time.Second)
				
				return success
			},
			Timeout:     2 * time.Second,
			ShouldBlock: false,
		},
	}

	for _, op := range operations {
		t.Run(op.Name, func(t *testing.T) {
			// Setup operation
			if op.Setup != nil {
				op.Setup(harness)
			}

			// Execute operation
			start := time.Now()
			err := op.Execute(harness)
			duration := time.Since(start)

			// Validate operation
			if op.ShouldBlock {
				// Operation should block and take significant time
				assert.True(t, duration > 100*time.Millisecond, "Operation should block but completed too quickly")
			} else {
				// Operation should not block
				assert.True(t, duration < 50*time.Millisecond, "Operation blocked but should be non-blocking")
			}

			// Validate result
			if op.Validate != nil {
				success := op.Validate(harness)
				assert.True(t, success, "Async operation validation failed: %s", op.Name)
			}

			// Verify mock expectations
			harness.MockRepo.AssertExpectations(t)
			harness.MockAI.AssertExpectations(t)
			harness.MockLabel.AssertExpectations(t)
			harness.MockSearch.AssertExpectations(t)
		})
	}
}

// TestAsyncOperationCancellation tests cancellation of async operations
func TestAsyncOperationCancellation(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t)

	t.Run("cancel_message_loading", func(t *testing.T) {
		// Setup long-running operation
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				time.Sleep(500 * time.Millisecond)
			}).
			Return(&services.MessagePage{
				Messages:      harness.GenerateTestMessages(100),
				NextPageToken: "",
			}, nil)

		// Start operation
		err := harness.App.LoadMessagesAsync()
		assert.NoError(t, err)

		// Wait a bit for operation to start
		time.Sleep(50 * time.Millisecond)

		// Cancel operation
		harness.App.CancelAsyncOperations()

		// Verify operation was cancelled
		success := harness.WaitForCondition(func() bool {
			return !harness.App.IsLoading()
		}, 1*time.Second)

		assert.True(t, success, "Operation should have been cancelled")
	})

	t.Run("cancel_ai_operation", func(t *testing.T) {
		// Setup long-running AI operation
		harness.App.SetCurrentMessageID("msg_1")
		harness.MockAI.On("GenerateSummary", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				time.Sleep(300 * time.Millisecond)
			}).
			Return(&services.SummaryResult{
				Summary: "Should not see this",
			}, nil)

		// Start operation
		err := harness.App.GenerateAISummaryAsync("msg_1")
		assert.NoError(t, err)

		// Wait a bit for operation to start
		time.Sleep(50 * time.Millisecond)

		// Cancel operation
		harness.App.CancelAsyncOperations()

		// Verify operation was cancelled
		success := harness.WaitForCondition(func() bool {
			return !harness.App.IsAIOperationInProgress("msg_1")
		}, 1*time.Second)

		assert.True(t, success, "AI operation should have been cancelled")
	})
}

// TestAsyncOperationTimeout tests timeout handling
func TestAsyncOperationTimeout(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t)

	t.Run("timeout_message_loading", func(t *testing.T) {
		// Setup operation that takes longer than timeout
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				time.Sleep(2 * time.Second)
			}).
			Return(&services.MessagePage{
				Messages:      harness.GenerateTestMessages(50),
				NextPageToken: "",
			}, nil)

		// Start operation with timeout
		ctx, cancel := context.WithTimeout(harness.Ctx, 500*time.Millisecond)
		defer cancel()

		err := harness.App.LoadMessagesWithContext(ctx)
		assert.NoError(t, err)

		// Wait for timeout
		success := harness.WaitForCondition(func() bool {
			return !harness.App.IsLoading()
		}, 1*time.Second)

		assert.True(t, success, "Operation should have timed out")
	})
}

// TestAsyncOperationErrorHandling tests error handling in async operations
func TestAsyncOperationErrorHandling(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t)

	t.Run("network_error_handling", func(t *testing.T) {
		// Setup mock to return error
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(nil, assert.AnError)

		// Start operation
		err := harness.App.LoadMessagesAsync()
		assert.NoError(t, err)

		// Wait for error handling
		success := harness.WaitForCondition(func() bool {
			return harness.App.GetLastError() != nil
		}, 1*time.Second)

		assert.True(t, success, "Error should have been handled")
		assert.Contains(t, harness.App.GetLastError().Error(), "assert.AnError")
	})

	t.Run("partial_failure_handling", func(t *testing.T) {
		// Setup mock to return partial results then error
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{
				Messages:      harness.GenerateTestMessages(10),
				NextPageToken: "next_page",
			}, nil).
			Once()

		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(nil, assert.AnError).
			Once()

		// Start operation
		err := harness.App.LoadMessagesAsync()
		assert.NoError(t, err)

		// Wait for partial results
		success := harness.WaitForCondition(func() bool {
			return harness.App.GetMessageCount() == 10
		}, 1*time.Second)

		assert.True(t, success, "Partial results should have been loaded")

		// Try to load next page (should fail)
		err = harness.App.LoadNextPage()
		assert.Error(t, err)
	})
}

// TestAsyncOperationConcurrency tests concurrent async operations
func TestAsyncOperationConcurrency(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t)

	t.Run("concurrent_message_loading", func(t *testing.T) {
		// Setup multiple concurrent operations
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				time.Sleep(100 * time.Millisecond)
			}).
			Return(&services.MessagePage{
				Messages:      harness.GenerateTestMessages(20),
				NextPageToken: "",
			}, nil).
			Times(3)

		// Start multiple concurrent operations
		errors := make(chan error, 3)
		for i := 0; i < 3; i++ {
			go func() {
				err := harness.App.LoadMessagesAsync()
				errors <- err
			}()
		}

		// Wait for all operations to complete
		for i := 0; i < 3; i++ {
			err := <-errors
			assert.NoError(t, err)
		}

		// Verify all operations completed
		harness.MockRepo.AssertExpectations(t)
	})

	t.Run("concurrent_ai_operations", func(t *testing.T) {
		// Setup concurrent AI operations
		harness.MockAI.On("GenerateSummary", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				time.Sleep(150 * time.Millisecond)
			}).
			Return(&services.SummaryResult{
				Summary: "Concurrent summary",
			}, nil).
			Times(2)

		// Start concurrent AI operations
		errors := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func(id int) {
				harness.App.SetCurrentMessageID(fmt.Sprintf("msg_%d", id))
				err := harness.App.GenerateAISummaryAsync(fmt.Sprintf("msg_%d", id))
				errors <- err
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < 2; i++ {
			err := <-errors
			assert.NoError(t, err)
		}

		// Verify all operations completed
		harness.MockAI.AssertExpectations(t)
	})
}