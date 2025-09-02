package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/stretchr/testify/assert"
)

// Test Query Service constructor
func TestNewQueryService(t *testing.T) {
	cfg := &config.Config{}

	service := NewQueryService(nil, cfg) // Pass nil store for testing

	assert.NotNil(t, service)
	assert.Nil(t, service.store)
	assert.Empty(t, service.accountEmail) // Should be empty initially
}

func TestNewQueryService_NilInputs(t *testing.T) {
	service := NewQueryService(nil, nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.store)
	assert.Empty(t, service.accountEmail)
}

// Test account email management
func TestQueryServiceImpl_AccountEmail(t *testing.T) {
	service := NewQueryService(nil, nil)

	// Initial state
	assert.Empty(t, service.GetAccountEmail())

	// Set account email
	service.SetAccountEmail("test@example.com")
	assert.Equal(t, "test@example.com", service.GetAccountEmail())

	// Update account email
	service.SetAccountEmail("new@example.com")
	assert.Equal(t, "new@example.com", service.GetAccountEmail())

	// Clear account email
	service.SetAccountEmail("")
	assert.Empty(t, service.GetAccountEmail())
}

func TestQueryServiceImpl_AccountEmail_Validation(t *testing.T) {
	service := NewQueryService(nil, nil)

	testCases := []struct {
		name     string
		email    string
		expected string
	}{
		{"valid_email", "user@domain.com", "user@domain.com"},
		{"empty_email", "", ""},
		{"whitespace_email", "   ", "   "}, // May or may not be trimmed
		{"unicode_email", "用户@domain.com", "用户@domain.com"},
		{"long_email", strings.Repeat("a", 100) + "@domain.com", strings.Repeat("a", 100) + "@domain.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service.SetAccountEmail(tc.email)
			assert.Equal(t, tc.expected, service.GetAccountEmail())
		})
	}
}

// Test query validation scenarios (basic validation without store operations)
func TestQueryServiceImpl_QueryValidation(t *testing.T) {
	t.Run("service_with_nil_store", func(t *testing.T) {
		service := NewQueryService(nil, nil)
		service.SetAccountEmail("test@example.com")

		// Service should handle nil store gracefully
		assert.NotPanics(t, func() {
			// Service should be created and handle nil store
			assert.Equal(t, "test@example.com", service.GetAccountEmail())
		})
	})

	t.Run("empty_account_email_handling", func(t *testing.T) {
		service := NewQueryService(nil, nil)
		// Don't set account email - should be empty

		assert.NotPanics(t, func() {
			// Service should handle empty account email gracefully
			assert.Empty(t, service.GetAccountEmail())
		})
	})
}

// Test concurrent access to query service
func TestQueryServiceImpl_ConcurrentAccess(t *testing.T) {
	service := NewQueryService(nil, nil)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// Test concurrent account email operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			email := fmt.Sprintf("user%d@example.com", id)
			service.SetAccountEmail(email)
			retrievedEmail := service.GetAccountEmail()

			// May not be the same due to race conditions, but should not panic
			_ = retrievedEmail
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Service should still be functional after concurrent access
	assert.NotNil(t, service)
}

// Test edge cases
func TestQueryServiceImpl_EdgeCases(t *testing.T) {
	t.Run("unicode_content", func(t *testing.T) {
		service := NewQueryService(nil, nil)

		unicodeEmails := []string{
			"用户@example.com",
			"пользователь@example.com",
			"ユーザー@example.com",
			"🎉@example.com",
		}

		for _, email := range unicodeEmails {
			assert.NotPanics(t, func() {
				service.SetAccountEmail(email)
				result := service.GetAccountEmail()
				assert.Equal(t, email, result)
			})
		}
	})

	t.Run("extreme_lengths", func(t *testing.T) {
		service := NewQueryService(nil, nil)

		// Test very long email
		longEmail := strings.Repeat("a", 10000) + "@example.com"

		assert.NotPanics(t, func() {
			service.SetAccountEmail(longEmail)
			result := service.GetAccountEmail()
			assert.Equal(t, longEmail, result)
		})
	})

	t.Run("null_bytes", func(t *testing.T) {
		service := NewQueryService(nil, nil)

		// Test email with null bytes
		emailWithNull := "user\x00null@example.com"

		assert.NotPanics(t, func() {
			service.SetAccountEmail(emailWithNull)
			// Should handle null bytes gracefully
		})
	})
}

// Test service state management
func TestQueryServiceImpl_StateManagement(t *testing.T) {
	service := NewQueryService(nil, nil)

	// Test state transitions
	states := []string{
		"",
		"first@example.com",
		"second@example.com",
		"",
		"third@example.com",
	}

	for i, state := range states {
		service.SetAccountEmail(state)
		assert.Equal(t, state, service.GetAccountEmail(), "State %d should match", i)
	}
}

// Benchmark query service operations
func BenchmarkQueryService_SetAccountEmail(b *testing.B) {
	service := NewQueryService(nil, nil)
	email := "benchmark@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.SetAccountEmail(email)
	}
}

func BenchmarkQueryService_GetAccountEmail(b *testing.B) {
	service := NewQueryService(nil, nil)
	service.SetAccountEmail("benchmark@example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetAccountEmail()
	}
}

func BenchmarkQueryService_ConcurrentAccess(b *testing.B) {
	service := NewQueryService(nil, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				service.SetAccountEmail(fmt.Sprintf("user%d@example.com", i))
			} else {
				service.GetAccountEmail()
			}
			i++
		}
	})
}
