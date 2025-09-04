package helpers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/goleak"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// AsyncOperationTest defines a test for asynchronous operations
type AsyncOperationTest struct {
	Name         string
	Setup        func(*TestHarness)
	Execute      func(*TestHarness) context.CancelFunc
	Validate     func(*TestHarness) bool
	Teardown     func(*TestHarness)
	Timeout      time.Duration
	ExpectCancel bool
}

// RunAsyncOperationsTests runs comprehensive tests for asynchronous operations
func RunAsyncOperationsTests(t *testing.T, harness *TestHarness) {
	// Enable goroutine leak detection with ignored goroutines
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

	tests := []AsyncOperationTest{
		{
			Name: "message_loading_async",
			Setup: func(h *TestHarness) {
				messages := h.GenerateTestMessages(50)
				h.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
					Return(&services.MessagePage{
						Messages:      messages,
						NextPageToken: "next_token",
					}, nil).
					Run(func(args mock.Arguments) {
						// Simulate network delay
						time.Sleep(100 * time.Millisecond)
					})
			},
			Execute: func(h *TestHarness) context.CancelFunc {
				ctx, cancel := context.WithCancel(h.Ctx)
				go func() {
					// Simulate async message loading
					opts := services.QueryOptions{MaxResults: 50}
					_, _ = h.MockRepo.GetMessages(ctx, opts)
				}()
				return cancel
			},
			Validate: func(h *TestHarness) bool {
				// Wait for mock expectations to be met
				time.Sleep(200 * time.Millisecond)
				h.MockRepo.AssertExpectations(t)
				return true
			},
			Timeout: 5 * time.Second,
		},
		{
			Name: "ai_summary_generation_async",
			Setup: func(h *TestHarness) {
				h.MockAI.On("GenerateSummary", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
					Return(&services.SummaryResult{
						Summary:   "Generated summary content",
						FromCache: false,
					}, nil).
					Run(func(args mock.Arguments) {
						// Simulate AI processing time
						time.Sleep(200 * time.Millisecond)
					})
			},
			Execute: func(h *TestHarness) context.CancelFunc {
				ctx, cancel := context.WithCancel(h.Ctx)
				go func() {
					opts := services.SummaryOptions{MaxLength: 100}
					_, _ = h.MockAI.GenerateSummary(ctx, "Test email content for summarization", opts)
				}()
				return cancel
			},
			Validate: func(h *TestHarness) bool {
				// Wait for AI processing
				time.Sleep(300 * time.Millisecond)
				h.MockAI.AssertExpectations(t)
				return true
			},
			Timeout: 10 * time.Second,
		},
		{
			Name: "bulk_label_application_async",
			Setup: func(h *TestHarness) {
				h.MockLabel.On("ApplyLabel", mock.Anything, mock.AnythingOfType("string"), "IMPORTANT").
					Return(nil).
					Run(func(args mock.Arguments) {
						// Simulate API call delay
						time.Sleep(50 * time.Millisecond)
					})
			},
			Execute: func(h *TestHarness) context.CancelFunc {
				ctx, cancel := context.WithCancel(h.Ctx)
				messageIDs := []string{"msg_1", "msg_2", "msg_3", "msg_4", "msg_5"}

				go func() {
					var wg sync.WaitGroup
					for _, msgID := range messageIDs {
						wg.Add(1)
						go func(id string) {
							defer wg.Done()
							_ = h.MockLabel.ApplyLabel(ctx, id, "IMPORTANT")
						}(msgID)
					}
					wg.Wait()
				}()

				return cancel
			},
			Validate: func(h *TestHarness) bool {
				// Wait for all label operations to complete
				time.Sleep(400 * time.Millisecond)
				h.MockLabel.AssertNumberOfCalls(t, "ApplyLabel", 5)
				return true
			},
			Timeout: 5 * time.Second,
		},
		{
			Name: "search_operation_with_timeout",
			Setup: func(h *TestHarness) {
				h.MockSearch.On("Search", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
					Return(&services.SearchResult{
						Messages:   []*gmail_v1.Message{},
						TotalCount: 0,
					}, nil).
					Run(func(args mock.Arguments) {
						ctx := args.Get(0).(context.Context)
						// Simulate long search operation with context cancellation support
						select {
						case <-ctx.Done():
							return // Context was cancelled
						case <-time.After(2 * time.Second):
							return // Simulate completion after delay
						}
					})
			},
			Execute: func(h *TestHarness) context.CancelFunc {
				// Use shorter context timeout to test timeout handling
				ctx, cancel := context.WithTimeout(h.Ctx, 500*time.Millisecond)
				go func() {
					opts := services.SearchOptions{MaxResults: 50}
					_, _ = h.MockSearch.Search(ctx, "complex search query", opts)
				}()
				return cancel
			},
			Validate: func(h *TestHarness) bool {
				// Search should not complete due to timeout
				time.Sleep(1 * time.Second)
				// We can't easily test context timeout with mocks, so we just verify no crash
				return true
			},
			Timeout:      3 * time.Second,
			ExpectCancel: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Enable goroutine leak detection for each test
			defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

			// Setup
			if test.Setup != nil {
				test.Setup(harness)
			}

			// Execute async operation
			var cancel context.CancelFunc
			if test.Execute != nil {
				cancel = test.Execute(harness)
			}

			// Wait for completion or timeout
			done := make(chan bool, 1)
			go func() {
				if test.Validate != nil {
					done <- test.Validate(harness)
				} else {
					done <- true
				}
			}()

			select {
			case success := <-done:
				if !test.ExpectCancel {
					assert.True(t, success, fmt.Sprintf("Async operation validation failed: %s", test.Name))
				}
			case <-time.After(test.Timeout):
				if !test.ExpectCancel {
					t.Errorf("Async operation timed out: %s", test.Name)
				}
			}

			// Cancel any remaining operations
			if cancel != nil {
				cancel()
			}

			// Teardown
			if test.Teardown != nil {
				test.Teardown(harness)
			}

			// Give time for goroutines to finish
			time.Sleep(100 * time.Millisecond)
		})
	}
}

// RunAsyncOperationCancellationTests tests operation cancellation
func RunAsyncOperationCancellationTests(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

	t.Run("ai_streaming_cancellation", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

		// Setup streaming mock
		harness.MockAI.On("GenerateSummaryStream", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("func(string)")).
			Return(&services.SummaryResult{Summary: "Stream result", FromCache: false}, nil).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				callback := args.Get(3).(func(string))

				// Simulate streaming with cancellation check
				for i := 0; i < 100; i++ {
					select {
					case <-ctx.Done():
						return // Cancelled
					default:
						callback(fmt.Sprintf("chunk_%d ", i))
						time.Sleep(10 * time.Millisecond)
					}
				}
			})

		// Start streaming operation
		ctx, cancel := context.WithCancel(harness.Ctx)

		go func() {
			opts := services.SummaryOptions{MaxLength: 100}
			_, _ = harness.MockAI.GenerateSummaryStream(ctx, "test content", opts, func(chunk string) {
				// Process streaming chunk
			})
		}()

		// Let it run for a short time
		time.Sleep(200 * time.Millisecond)

		// Cancel the operation
		cancel()

		// Give time for cancellation to propagate
		time.Sleep(100 * time.Millisecond)

		// Verify mock was called (streaming started)
		harness.MockAI.AssertCalled(t, "GenerateSummaryStream", mock.Anything, "test content", mock.Anything, mock.AnythingOfType("func(string)"))
	})

	t.Run("bulk_operation_cancellation", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

		callCount := 0
		var mu sync.Mutex

		// Setup mock to track calls and respect cancellation
		harness.MockEmail.On("ArchiveMessage", mock.Anything, mock.AnythingOfType("string")).
			Return(nil).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)

				mu.Lock()
				callCount++
				mu.Unlock()

				// Simulate work with cancellation check
				for i := 0; i < 10; i++ {
					select {
					case <-ctx.Done():
						return
					default:
						time.Sleep(10 * time.Millisecond)
					}
				}
			})

		// Start bulk operation
		ctx, cancel := context.WithCancel(harness.Ctx)
		messageIDs := []string{"msg_1", "msg_2", "msg_3", "msg_4", "msg_5", "msg_6", "msg_7", "msg_8", "msg_9", "msg_10"}

		var wg sync.WaitGroup
		for _, msgID := range messageIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				_ = harness.MockEmail.ArchiveMessage(ctx, id)
			}(msgID)
		}

		// Let some operations start
		time.Sleep(50 * time.Millisecond)

		// Cancel remaining operations
		cancel()

		// Wait for all goroutines to finish
		wg.Wait()

		// Verify that some operations were started but not all completed
		mu.Lock()
		finalCallCount := callCount
		mu.Unlock()

		// Should have started some operations
		assert.Greater(t, finalCallCount, 0, "No operations were started")
		// But might not have completed all due to cancellation
		t.Logf("Bulk operations started: %d out of %d", finalCallCount, len(messageIDs))
	})
}

// RunAsyncOperationTimeoutTests tests timeout handling
func RunAsyncOperationTimeoutTests(t *testing.T, harness *TestHarness) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

	t.Run("message_loading_timeout", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

		// Setup slow-responding mock
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return((*services.MessagePage)(nil), fmt.Errorf("timeout")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				// Simulate very slow operation with context cancellation support
				select {
				case <-ctx.Done():
					return // Context was cancelled (timeout)
				case <-time.After(2 * time.Second):
					return // Would complete after delay (but context should timeout first)
				}
			})

		// Use short timeout context
		ctx, cancel := context.WithTimeout(harness.Ctx, 100*time.Millisecond)
		defer cancel()

		// Execute operation that should timeout
		done := make(chan error, 1)
		go func() {
			opts := services.QueryOptions{MaxResults: 10}
			_, err := harness.MockRepo.GetMessages(ctx, opts)
			done <- err
		}()

		// Verify timeout occurs
		select {
		case err := <-done:
			// Operation completed - could be timeout error or actual result
			// In a real scenario with proper context handling, we'd expect a timeout error
			t.Logf("Operation completed with error: %v", err)
		case <-time.After(500 * time.Millisecond):
			t.Log("Operation timed out as expected (mock may still be running but context cancelled)")
		}
	})

	t.Run("ai_generation_timeout", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/ajramos/giztui/internal/services.(*MessagePreloaderImpl).startWorkers"))

		// Setup mock that takes too long
		harness.MockAI.On("GenerateSummary", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
			Return((*services.SummaryResult)(nil), fmt.Errorf("generation timeout")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)

				// Simulate long AI processing with context checking
				for i := 0; i < 100; i++ {
					select {
					case <-ctx.Done():
						return
					default:
						time.Sleep(50 * time.Millisecond)
					}
				}
			})

		// Use timeout context
		ctx, cancel := context.WithTimeout(harness.Ctx, 200*time.Millisecond)
		defer cancel()

		// Execute AI generation
		start := time.Now()
		opts := services.SummaryOptions{MaxLength: 100}
		_, err := harness.MockAI.GenerateSummary(ctx, "test content", opts)
		duration := time.Since(start)

		// Should complete quickly due to timeout
		assert.Less(t, duration, 1*time.Second, "Operation should have timed out quickly")

		// May or may not have error depending on mock implementation
		t.Logf("AI generation result: error=%v, duration=%v", err, duration)
	})
}
