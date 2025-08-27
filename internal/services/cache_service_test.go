package services

import (
	"context"
	"testing"

	"github.com/ajramos/gmail-tui/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestNewCacheService(t *testing.T) {
	store := &db.CacheStore{}
	service := NewCacheService(store)

	assert.NotNil(t, service)
	assert.Equal(t, store, service.store)
}

func TestNewCacheService_NilStore(t *testing.T) {
	service := NewCacheService(nil)
	assert.NotNil(t, service)
	assert.Nil(t, service.store)
}

func TestCacheService_GetSummary_NilStore(t *testing.T) {
	service := NewCacheService(nil)
	ctx := context.Background()

	summary, found, err := service.GetSummary(ctx, "test@example.com", "msg123")

	assert.Equal(t, "", summary)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not available")
}

func TestCacheService_GetSummary_ValidationErrors(t *testing.T) {
	// Use a real store but we'll only test validation, not actual database operations
	store := &db.CacheStore{} // This will be nil internally but that's ok for validation tests
	service := NewCacheService(store)
	ctx := context.Background()

	tests := []struct {
		name          string
		accountEmail  string
		messageID     string
		expectedError string
	}{
		{
			name:          "empty_account_email",
			accountEmail:  "",
			messageID:     "msg123",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "empty_message_id",
			accountEmail:  "test@example.com",
			messageID:     "",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "whitespace_only_account_email",
			accountEmail:  "   ",
			messageID:     "msg123",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "whitespace_only_message_id",
			accountEmail:  "test@example.com",
			messageID:     "   ",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "both_empty",
			accountEmail:  "",
			messageID:     "",
			expectedError: "accountEmail and messageID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, found, err := service.GetSummary(ctx, tt.accountEmail, tt.messageID)

			assert.Equal(t, "", summary)
			assert.False(t, found)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCacheService_SaveSummary_NilStore(t *testing.T) {
	service := NewCacheService(nil)
	ctx := context.Background()

	err := service.SaveSummary(ctx, "test@example.com", "msg123", "summary")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not available")
}

func TestCacheService_SaveSummary_ValidationErrors(t *testing.T) {
	// Use a real store but we'll only test validation, not actual database operations
	store := &db.CacheStore{} // This will be nil internally but that's ok for validation tests
	service := NewCacheService(store)
	ctx := context.Background()

	tests := []struct {
		name          string
		accountEmail  string
		messageID     string
		summary       string
		expectedError string
	}{
		{
			name:          "empty_account_email",
			accountEmail:  "",
			messageID:     "msg123",
			summary:       "test summary",
			expectedError: "accountEmail, messageID, and summary cannot be empty",
		},
		{
			name:          "empty_message_id",
			accountEmail:  "test@example.com",
			messageID:     "",
			summary:       "test summary",
			expectedError: "accountEmail, messageID, and summary cannot be empty",
		},
		{
			name:          "empty_summary",
			accountEmail:  "test@example.com",
			messageID:     "msg123",
			summary:       "",
			expectedError: "accountEmail, messageID, and summary cannot be empty",
		},
		{
			name:          "whitespace_only_account_email",
			accountEmail:  "   ",
			messageID:     "msg123",
			summary:       "test summary",
			expectedError: "accountEmail, messageID, and summary cannot be empty",
		},
		{
			name:          "whitespace_only_message_id",
			accountEmail:  "test@example.com",
			messageID:     "   ",
			summary:       "test summary",
			expectedError: "accountEmail, messageID, and summary cannot be empty",
		},
		{
			name:          "whitespace_only_summary",
			accountEmail:  "test@example.com",
			messageID:     "msg123",
			summary:       "   ",
			expectedError: "accountEmail, messageID, and summary cannot be empty",
		},
		{
			name:          "all_empty",
			accountEmail:  "",
			messageID:     "",
			summary:       "",
			expectedError: "accountEmail, messageID, and summary cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SaveSummary(ctx, tt.accountEmail, tt.messageID, tt.summary)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCacheService_InvalidateSummary_NilStore(t *testing.T) {
	service := NewCacheService(nil)
	ctx := context.Background()

	err := service.InvalidateSummary(ctx, "test@example.com", "msg123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not available")
}

func TestCacheService_InvalidateSummary_ValidationErrors(t *testing.T) {
	// Use a real store but we'll only test validation, not actual database operations
	store := &db.CacheStore{} // This will be nil internally but that's ok for validation tests
	service := NewCacheService(store)
	ctx := context.Background()

	tests := []struct {
		name          string
		accountEmail  string
		messageID     string
		expectedError string
	}{
		{
			name:          "empty_account_email",
			accountEmail:  "",
			messageID:     "msg123",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "empty_message_id",
			accountEmail:  "test@example.com",
			messageID:     "",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "whitespace_only_account_email",
			accountEmail:  "   ",
			messageID:     "msg123",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "whitespace_only_message_id",
			accountEmail:  "test@example.com",
			messageID:     "   ",
			expectedError: "accountEmail and messageID cannot be empty",
		},
		{
			name:          "both_empty",
			accountEmail:  "",
			messageID:     "",
			expectedError: "accountEmail and messageID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.InvalidateSummary(ctx, tt.accountEmail, tt.messageID)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCacheService_ClearCache_NilStore(t *testing.T) {
	service := NewCacheService(nil)
	ctx := context.Background()

	err := service.ClearCache(ctx, "test@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not available")
}

func TestCacheService_ClearCache_ValidationAndNotImplemented(t *testing.T) {
	// Use a real store but we'll only test validation, not actual database operations
	store := &db.CacheStore{} // This will be nil internally but that's ok for validation tests
	service := NewCacheService(store)
	ctx := context.Background()

	// Test empty account email validation
	err := service.ClearCache(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accountEmail cannot be empty")

	// Test whitespace only account email validation
	err = service.ClearCache(ctx, "   ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accountEmail cannot be empty")

	// Test not implemented functionality
	err = service.ClearCache(ctx, "test@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clear cache not implemented")
}

// Test edge cases for input validation
func TestCacheService_EdgeCaseInputs(t *testing.T) {
	store := &db.CacheStore{}
	service := NewCacheService(store)
	ctx := context.Background()

	// Test with very long inputs (should pass validation but fail on db operation)
	longString := make([]byte, 10000)
	for i := range longString {
		longString[i] = 'a'
	}
	longStringValue := string(longString)

	t.Run("very_long_account_email", func(t *testing.T) {
		_, _, err := service.GetSummary(ctx, longStringValue, "msg123")
		// Should pass validation but fail on database operation (which is expected)
		assert.Error(t, err)
	})

	t.Run("very_long_message_id", func(t *testing.T) {
		_, _, err := service.GetSummary(ctx, "test@example.com", longStringValue)
		// Should pass validation but fail on database operation (which is expected)
		assert.Error(t, err)
	})

	t.Run("very_long_summary", func(t *testing.T) {
		err := service.SaveSummary(ctx, "test@example.com", "msg123", longStringValue)
		// Should pass validation but fail on database operation (which is expected)
		assert.Error(t, err)
	})
}

// Benchmark tests for performance critical operations
func BenchmarkCacheService_GetSummary_ValidationOnly(b *testing.B) {
	service := NewCacheService(nil) // Use nil store to only benchmark validation
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = service.GetSummary(ctx, "test@example.com", "msg123")
	}
}

func BenchmarkCacheService_SaveSummary_ValidationOnly(b *testing.B) {
	service := NewCacheService(nil) // Use nil store to only benchmark validation
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.SaveSummary(ctx, "test@example.com", "msg123", "Test summary")
	}
}
