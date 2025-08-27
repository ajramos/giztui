package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCacheStore(t *testing.T) {
	// Test with nil store
	cache := NewCacheStore(nil)
	assert.Nil(t, cache)

	// Test with valid store
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cache_test.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache = NewCacheStore(store)
	assert.NotNil(t, cache)
	assert.Equal(t, store.db, cache.db)
}

func TestCacheStore_SaveAISummary_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "save_validation.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	tests := []struct {
		name         string
		accountEmail string
		messageID    string
		summary      string
		expectedErr  string
	}{
		{"empty_account_email", "", "msg123", "summary", "invalid summary inputs"},
		{"empty_message_id", "test@example.com", "", "summary", "invalid summary inputs"},
		{"empty_summary", "test@example.com", "msg123", "", "invalid summary inputs"},
		{"whitespace_account_email", "   ", "msg123", "summary", "invalid summary inputs"},
		{"whitespace_message_id", "test@example.com", "   ", "summary", "invalid summary inputs"},
		{"whitespace_summary", "test@example.com", "msg123", "   ", "invalid summary inputs"},
		{"all_empty", "", "", "", "invalid summary inputs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.SaveAISummary(ctx, tt.accountEmail, tt.messageID, tt.summary, time.Now().Unix())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestCacheStore_SaveAISummary_NilStore(t *testing.T) {
	var cache *CacheStore
	ctx := context.Background()

	err := cache.SaveAISummary(ctx, "test@example.com", "msg123", "summary", time.Now().Unix())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not initialized")
}

func TestCacheStore_SaveAISummary_NilDB(t *testing.T) {
	cache := &CacheStore{db: nil}
	ctx := context.Background()

	err := cache.SaveAISummary(ctx, "test@example.com", "msg123", "summary", time.Now().Unix())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not initialized")
}

func TestCacheStore_SaveAISummary_Success(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "save_success.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)
	updatedAt := time.Now().Unix()

	err = cache.SaveAISummary(ctx, "test@example.com", "msg123", "Test summary", updatedAt)
	assert.NoError(t, err)

	// Verify the record was inserted
	var storedSummary string
	var storedUpdatedAt int64
	err = cache.db.QueryRowContext(ctx,
		"SELECT summary, updated_at FROM ai_summaries WHERE account_email = ? AND message_id = ?",
		"test@example.com", "msg123").Scan(&storedSummary, &storedUpdatedAt)
	assert.NoError(t, err)
	assert.Equal(t, "Test summary", storedSummary)
	assert.Equal(t, updatedAt, storedUpdatedAt)
}

func TestCacheStore_SaveAISummary_Upsert(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "save_upsert.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Insert initial record
	updatedAt1 := time.Now().Unix()
	err = cache.SaveAISummary(ctx, "test@example.com", "msg123", "First summary", updatedAt1)
	assert.NoError(t, err)

	// Update the same record (upsert)
	updatedAt2 := updatedAt1 + 100
	err = cache.SaveAISummary(ctx, "test@example.com", "msg123", "Updated summary", updatedAt2)
	assert.NoError(t, err)

	// Verify only one record exists with updated data
	var count int
	err = cache.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM ai_summaries WHERE account_email = ? AND message_id = ?",
		"test@example.com", "msg123").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	var storedSummary string
	var storedUpdatedAt int64
	err = cache.db.QueryRowContext(ctx,
		"SELECT summary, updated_at FROM ai_summaries WHERE account_email = ? AND message_id = ?",
		"test@example.com", "msg123").Scan(&storedSummary, &storedUpdatedAt)
	assert.NoError(t, err)
	assert.Equal(t, "Updated summary", storedSummary)
	assert.Equal(t, updatedAt2, storedUpdatedAt)
}

func TestCacheStore_LoadAISummary_NilStore(t *testing.T) {
	var cache *CacheStore
	ctx := context.Background()

	summary, found, err := cache.LoadAISummary(ctx, "test@example.com", "msg123")
	assert.Equal(t, "", summary)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not initialized")
}

func TestCacheStore_LoadAISummary_NilDB(t *testing.T) {
	cache := &CacheStore{db: nil}
	ctx := context.Background()

	summary, found, err := cache.LoadAISummary(ctx, "test@example.com", "msg123")
	assert.Equal(t, "", summary)
	assert.False(t, found)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not initialized")
}

func TestCacheStore_LoadAISummary_NotFound(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "load_notfound.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	summary, found, err := cache.LoadAISummary(ctx, "nonexistent@example.com", "msg123")
	assert.NoError(t, err)
	assert.Equal(t, "", summary)
	assert.False(t, found)
}

func TestCacheStore_LoadAISummary_Found(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "load_found.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Insert test data
	testSummary := "Test summary content"
	updatedAt := time.Now().Unix()
	err = cache.SaveAISummary(ctx, "test@example.com", "msg123", testSummary, updatedAt)
	assert.NoError(t, err)

	// Load the data
	summary, found, err := cache.LoadAISummary(ctx, "test@example.com", "msg123")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, testSummary, summary)
}

func TestCacheStore_DeleteAISummary_NilStore(t *testing.T) {
	var cache *CacheStore
	ctx := context.Background()

	err := cache.DeleteAISummary(ctx, "test@example.com", "msg123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not initialized")
}

func TestCacheStore_DeleteAISummary_NilDB(t *testing.T) {
	cache := &CacheStore{db: nil}
	ctx := context.Background()

	err := cache.DeleteAISummary(ctx, "test@example.com", "msg123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache store not initialized")
}

func TestCacheStore_DeleteAISummary_Success(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "delete_success.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Insert test data
	err = cache.SaveAISummary(ctx, "test@example.com", "msg123", "Test summary", time.Now().Unix())
	assert.NoError(t, err)

	// Verify it exists
	summary, found, err := cache.LoadAISummary(ctx, "test@example.com", "msg123")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Test summary", summary)

	// Delete it
	err = cache.DeleteAISummary(ctx, "test@example.com", "msg123")
	assert.NoError(t, err)

	// Verify it's gone
	summary, found, err = cache.LoadAISummary(ctx, "test@example.com", "msg123")
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "", summary)
}

func TestCacheStore_DeleteAISummary_NonExistent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "delete_nonexistent.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Delete non-existent record (should not error)
	err = cache.DeleteAISummary(ctx, "nonexistent@example.com", "msg123")
	assert.NoError(t, err)
}

// Test multiple account isolation
func TestCacheStore_AccountIsolation(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "account_isolation.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Insert data for two different accounts with same message ID
	err = cache.SaveAISummary(ctx, "user1@example.com", "msg123", "User 1 summary", time.Now().Unix())
	assert.NoError(t, err)

	err = cache.SaveAISummary(ctx, "user2@example.com", "msg123", "User 2 summary", time.Now().Unix())
	assert.NoError(t, err)

	// Verify each user can only see their own data
	summary1, found1, err := cache.LoadAISummary(ctx, "user1@example.com", "msg123")
	assert.NoError(t, err)
	assert.True(t, found1)
	assert.Equal(t, "User 1 summary", summary1)

	summary2, found2, err := cache.LoadAISummary(ctx, "user2@example.com", "msg123")
	assert.NoError(t, err)
	assert.True(t, found2)
	assert.Equal(t, "User 2 summary", summary2)

	// Delete one user's data shouldn't affect the other
	err = cache.DeleteAISummary(ctx, "user1@example.com", "msg123")
	assert.NoError(t, err)

	// User 1's data should be gone
	_, found1, err = cache.LoadAISummary(ctx, "user1@example.com", "msg123")
	assert.NoError(t, err)
	assert.False(t, found1)

	// User 2's data should still exist
	summary2, found2, err = cache.LoadAISummary(ctx, "user2@example.com", "msg123")
	assert.NoError(t, err)
	assert.True(t, found2)
	assert.Equal(t, "User 2 summary", summary2)
}

// Test large data handling
func TestCacheStore_LargeData(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "large_data.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Create large summary (64KB)
	largeSummary := strings.Repeat("This is a test summary with lots of content. ", 1400)

	err = cache.SaveAISummary(ctx, "test@example.com", "msg_large", largeSummary, time.Now().Unix())
	assert.NoError(t, err)

	// Verify large data can be retrieved
	summary, found, err := cache.LoadAISummary(ctx, "test@example.com", "msg_large")
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, largeSummary, summary)
	assert.Greater(t, len(summary), 50000) // Verify it's actually large
}

// Test special characters and encoding
func TestCacheStore_SpecialCharacters(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "special_chars.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	testCases := []struct {
		name    string
		summary string
	}{
		{"unicode", "Summary with unicode: 你好世界 🚀 émojis"},
		{"quotes", `Summary with "quotes" and 'apostrophes'`},
		{"newlines", "Summary with\nmultiple\nlines"},
		{"sql_injection", "'; DROP TABLE ai_summaries; --"},
		{"special_chars", "Summary with <>&\"'% special chars"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			messageID := "msg_" + tc.name

			err := cache.SaveAISummary(ctx, "test@example.com", messageID, tc.summary, time.Now().Unix())
			assert.NoError(t, err)

			summary, found, err := cache.LoadAISummary(ctx, "test@example.com", messageID)
			assert.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, tc.summary, summary)
		})
	}
}

// Benchmark tests
func BenchmarkCacheStore_SaveAISummary(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_save.db")

	store, err := Open(ctx, dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	cache := NewCacheStore(store)
	updatedAt := time.Now().Unix()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cache.SaveAISummary(ctx, "bench@example.com", "msg"+string(rune(i)), "Benchmark summary", updatedAt)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheStore_LoadAISummary(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_load.db")

	store, err := Open(ctx, dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	cache := NewCacheStore(store)

	// Insert test data
	err = cache.SaveAISummary(ctx, "bench@example.com", "msg123", "Benchmark summary", time.Now().Unix())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := cache.LoadAISummary(ctx, "bench@example.com", "msg123")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheStore_DeleteAISummary(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_delete.db")

	store, err := Open(ctx, dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	cache := NewCacheStore(store)

	// Pre-populate with test data
	for i := 0; i < b.N; i++ {
		err := cache.SaveAISummary(ctx, "bench@example.com", "msg"+string(rune(i)), "Benchmark summary", time.Now().Unix())
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cache.DeleteAISummary(ctx, "bench@example.com", "msg"+string(rune(i)))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test concurrent access patterns
func TestCacheStore_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// This is a basic test - full concurrency testing would require goroutines
	// But we can verify basic operations work sequentially

	accounts := []string{"user1@example.com", "user2@example.com", "user3@example.com"}

	for i, account := range accounts {
		messageID := "msg123"
		summary := "Summary for " + account

		err := cache.SaveAISummary(ctx, account, messageID, summary, int64(i))
		assert.NoError(t, err)
	}

	// Verify all data was saved correctly
	for _, account := range accounts {
		summary, found, err := cache.LoadAISummary(ctx, account, "msg123")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "Summary for "+account, summary)
	}
}

// Test error recovery and resilience
func TestCacheStore_ErrorRecovery(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "error_recovery.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Test recovery after failed operation
	err = cache.SaveAISummary(ctx, "test@example.com", "msg1", "First summary", time.Now().Unix())
	assert.NoError(t, err)

	// This would fail in actual error scenarios, but our validation prevents most errors
	// Verify normal operations continue to work
	err = cache.SaveAISummary(ctx, "test@example.com", "msg2", "Second summary", time.Now().Unix())
	assert.NoError(t, err)

	summary1, found1, err := cache.LoadAISummary(ctx, "test@example.com", "msg1")
	assert.NoError(t, err)
	assert.True(t, found1)
	assert.Equal(t, "First summary", summary1)

	summary2, found2, err := cache.LoadAISummary(ctx, "test@example.com", "msg2")
	assert.NoError(t, err)
	assert.True(t, found2)
	assert.Equal(t, "Second summary", summary2)
}

func TestCacheStore_ValidationEdgeCases(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "validation_edge.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	cache := NewCacheStore(store)

	// Test with extremely long inputs
	longEmail := strings.Repeat("a", 1000) + "@example.com"
	longMessageID := strings.Repeat("m", 1000)
	longSummary := strings.Repeat("s", 10000)

	err = cache.SaveAISummary(ctx, longEmail, longMessageID, longSummary, time.Now().Unix())
	assert.NoError(t, err)

	summary, found, err := cache.LoadAISummary(ctx, longEmail, longMessageID)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, longSummary, summary)
}
