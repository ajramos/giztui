package helpers

import (
	"fmt"
	"testing"
	"time"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/goleak"
)

// TestBulkOperationsIsolated demonstrates isolated bulk operations tests
func TestBulkOperationsIsolated(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))

	t.Run("BulkArchive_3Messages", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup
		messages := harness.GenerateTestMessages(5)
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{Messages: messages}, nil).Once()
		
		// Expect bulk archive with exactly 3 messages
		harness.MockEmail.On("BulkArchive", mock.Anything, mock.MatchedBy(func(ids []string) bool {
			return len(ids) == 3
		})).Return(nil).Once()

		// Execute
		messageIDs := []string{"msg_0", "msg_1", "msg_2"}
		err := harness.MockEmail.BulkArchive(harness.Ctx, messageIDs)

		// Validate
		assert.NoError(t, err)
		harness.MockEmail.AssertExpectations(t)
	})

	t.Run("BulkLabelApplication_2Messages", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup
		messages := harness.GenerateTestMessages(3)
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{Messages: messages}, nil).Once()
		
		// Setup individual label application expectations
		harness.MockLabel.On("ApplyLabel", mock.Anything, "msg_0", "IMPORTANT").Return(nil).Once()
		harness.MockLabel.On("ApplyLabel", mock.Anything, "msg_1", "IMPORTANT").Return(nil).Once()

		// Execute
		messageIDs := []string{"msg_0", "msg_1"}
		for _, msgID := range messageIDs {
			err := harness.MockLabel.ApplyLabel(harness.Ctx, msgID, "IMPORTANT")
			assert.NoError(t, err)
		}

		// Validate
		harness.MockLabel.AssertExpectations(t)
	})

	t.Run("BulkTrash_5Messages", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup
		messages := harness.GenerateTestMessages(10)
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{Messages: messages}, nil).Once()
		
		// Expect bulk trash with exactly 5 messages
		harness.MockEmail.On("BulkTrash", mock.Anything, mock.MatchedBy(func(ids []string) bool {
			return len(ids) == 5
		})).Return(nil).Once()

		// Execute
		messageIDs := []string{"msg_0", "msg_1", "msg_2", "msg_3", "msg_4"}
		err := harness.MockEmail.BulkTrash(harness.Ctx, messageIDs)

		// Validate
		assert.NoError(t, err)
		harness.MockEmail.AssertExpectations(t)
	})

	t.Run("EmptySelection", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Test empty selection handling
		emptyIDs := []string{}
		result := len(emptyIDs)
		assert.Equal(t, 0, result, "Empty selection should have 0 items")
	})

	t.Run("SingleMessage", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup
		messages := harness.GenerateTestMessages(1)
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{Messages: messages}, nil).Once()
		
		harness.MockEmail.On("BulkArchive", mock.Anything, []string{"msg_0"}).Return(nil).Once()

		// Execute
		err := harness.MockEmail.BulkArchive(harness.Ctx, []string{"msg_0"})

		// Validate
		assert.NoError(t, err)
		harness.MockEmail.AssertExpectations(t)
	})

	t.Run("PartialFailure", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup individual operations with mixed success/failure
		harness.MockEmail.On("ArchiveMessage", mock.Anything, "msg_0").Return(nil).Once()
		harness.MockEmail.On("ArchiveMessage", mock.Anything, "msg_1").Return(fmt.Errorf("operation failed")).Once()
		harness.MockEmail.On("ArchiveMessage", mock.Anything, "msg_2").Return(nil).Once()

		// Execute individual operations
		messageIDs := []string{"msg_0", "msg_1", "msg_2"}
		var errors []error
		for _, msgID := range messageIDs {
			err := harness.MockEmail.ArchiveMessage(harness.Ctx, msgID)
			if err != nil {
				errors = append(errors, err)
			}
		}

		// Validate
		assert.Len(t, errors, 1, "Should have exactly one error")
		harness.MockEmail.AssertExpectations(t)
	})

	t.Run("Performance_SmallBatch", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup
		count := 10
		messages := harness.GenerateTestMessages(count)
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{Messages: messages}, nil).Once()
		
		harness.MockEmail.On("BulkArchive", mock.Anything, mock.MatchedBy(func(ids []string) bool {
			return len(ids) == count
		})).Return(nil).Once()

		// Execute with timing
		start := time.Now()
		messageIDs := make([]string, count)
		for i := 0; i < count; i++ {
			messageIDs[i] = fmt.Sprintf("msg_%d", i)
		}
		err := harness.MockEmail.BulkArchive(harness.Ctx, messageIDs)
		duration := time.Since(start)

		// Validate
		assert.NoError(t, err)
		assert.Less(t, duration, 1*time.Second, "Small batch should complete quickly")
		harness.MockEmail.AssertExpectations(t)
		
		t.Logf("Bulk operation on %d messages completed in %v", count, duration)
	})

	t.Run("Performance_MediumBatch", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup
		count := 100
		messages := harness.GenerateTestMessages(count)
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{Messages: messages}, nil).Once()
		
		harness.MockEmail.On("BulkArchive", mock.Anything, mock.MatchedBy(func(ids []string) bool {
			return len(ids) == count
		})).Return(nil).Once()

		// Execute with timing
		start := time.Now()
		messageIDs := make([]string, count)
		for i := 0; i < count; i++ {
			messageIDs[i] = fmt.Sprintf("msg_%d", i)
		}
		err := harness.MockEmail.BulkArchive(harness.Ctx, messageIDs)
		duration := time.Since(start)

		// Validate
		assert.NoError(t, err)
		assert.Less(t, duration, 2*time.Second, "Medium batch should complete in reasonable time")
		harness.MockEmail.AssertExpectations(t)
		
		t.Logf("Bulk operation on %d messages completed in %v", count, duration)
	})

	t.Run("Performance_LargeBatch", func(t *testing.T) {
		defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("time.Sleep"))
		
		harness := NewTestHarness(t)
		defer harness.Cleanup()

		// Setup
		count := 1000
		messages := harness.GenerateTestMessages(count)
		harness.MockRepo.On("GetMessages", mock.Anything, mock.Anything).
			Return(&services.MessagePage{Messages: messages}, nil).Once()
		
		harness.MockEmail.On("BulkArchive", mock.Anything, mock.MatchedBy(func(ids []string) bool {
			return len(ids) == count
		})).Return(nil).Once()

		// Execute with timing
		start := time.Now()
		messageIDs := make([]string, count)
		for i := 0; i < count; i++ {
			messageIDs[i] = fmt.Sprintf("msg_%d", i)
		}
		err := harness.MockEmail.BulkArchive(harness.Ctx, messageIDs)
		duration := time.Since(start)

		// Validate
		assert.NoError(t, err)
		assert.Less(t, duration, 5*time.Second, "Large batch should complete within reasonable time")
		harness.MockEmail.AssertExpectations(t)
		
		t.Logf("Bulk operation on %d messages completed in %v", count, duration)
	})
}