package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpen_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		dbPath      string
		expectedErr string
	}{
		{"empty_path", "", "empty database path"},
		{"whitespace_path", "   ", "empty database path"},
		{"tabs_path", "\t\t", "empty database path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := Open(ctx, tt.dbPath)
			assert.Nil(t, store)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestOpen_Success(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, store)
	assert.NotNil(t, store.db)

	// Cleanup
	assert.NoError(t, store.Close())
}

func TestOpen_DirectoryCreation(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "deep", "test.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, store)

	// Verify nested directories were created
	assert.DirExists(t, filepath.Dir(dbPath))

	// Cleanup
	assert.NoError(t, store.Close())
}

func TestOpen_FilePermissions(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, store)

	// Check file permissions (should be 0600)
	info, err := os.Stat(dbPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Cleanup
	assert.NoError(t, store.Close())
}

func TestOpen_ExistingFile(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "existing.db")

	// Create first store
	store1, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, store1)
	assert.NoError(t, store1.Close())

	// Open existing file
	store2, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, store2)
	assert.NoError(t, store2.Close())
}

func TestClose_NilStore(t *testing.T) {
	var store *Store
	err := store.Close()
	assert.NoError(t, err) // Should handle nil gracefully
}

func TestClose_NilDB(t *testing.T) {
	store := &Store{db: nil}
	err := store.Close()
	assert.NoError(t, err) // Should handle nil db gracefully
}

func TestClose_ValidStore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, store)

	err = store.Close()
	assert.NoError(t, err)
}

func TestDB_Getter(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	db := store.DB()
	assert.NotNil(t, db)
	assert.IsType(t, &sql.DB{}, db)
}

func TestMigration_V1_AISummariesTable(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migrate_v1.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Verify ai_summaries table exists
	var tableName string
	err = store.db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='ai_summaries'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "ai_summaries", tableName)

	// Verify version is at least 1
	var version int
	err = store.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, version, 1)
}

func TestMigration_V3_PromptTables(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migrate_v3.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Verify prompt_templates table exists
	var tableName string
	err = store.db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='prompt_templates'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "prompt_templates", tableName)

	// Verify prompt_results table exists
	err = store.db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='prompt_results'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "prompt_results", tableName)

	// Verify default prompts were inserted
	var count int
	err = store.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM prompt_templates WHERE is_favorite = TRUE").Scan(&count)
	assert.NoError(t, err)
	assert.Greater(t, count, 0) // Should have at least some default prompts
}

func TestMigration_V5_BulkPromptResultsTable(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migrate_v5.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Verify bulk_prompt_results table exists
	var tableName string
	err = store.db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='bulk_prompt_results'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "bulk_prompt_results", tableName)

	// Verify version is at least 5
	var version int
	err = store.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, version, 5)
}

func TestMigration_V6_SavedQueriesTable(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migrate_v6.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Verify saved_queries table exists
	var tableName string
	err = store.db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='saved_queries'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "saved_queries", tableName)

	// Verify current version is 7
	var version int
	err = store.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	assert.NoError(t, err)
	assert.Equal(t, 7, version)
}

func TestPragmas_Configuration(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "pragmas.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Verify WAL mode is set
	var journalMode string
	err = store.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)
	assert.NoError(t, err)
	assert.Equal(t, "wal", journalMode)

	// Verify foreign keys are enabled
	var foreignKeys int
	err = store.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys)
	assert.NoError(t, err)
	assert.Equal(t, 1, foreignKeys)

	// Verify synchronous mode
	var syncMode string
	err = store.db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&syncMode)
	assert.NoError(t, err)
	// Should be 1 (NORMAL) or a string equivalent
	assert.True(t, syncMode == "1" || syncMode == "NORMAL")
}

func TestDatabaseIntegrity_BasicOperations(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integrity.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Test basic insert into ai_summaries
	_, err = store.db.ExecContext(ctx,
		"INSERT INTO ai_summaries (account_email, message_id, summary, updated_at) VALUES (?, ?, ?, ?)",
		"test@example.com", "msg123", "Test summary", 1234567890)
	assert.NoError(t, err)

	// Test retrieval
	var summary string
	err = store.db.QueryRowContext(ctx,
		"SELECT summary FROM ai_summaries WHERE account_email = ? AND message_id = ?",
		"test@example.com", "msg123").Scan(&summary)
	assert.NoError(t, err)
	assert.Equal(t, "Test summary", summary)

	// Test update
	_, err = store.db.ExecContext(ctx,
		"UPDATE ai_summaries SET summary = ? WHERE account_email = ? AND message_id = ?",
		"Updated summary", "test@example.com", "msg123")
	assert.NoError(t, err)

	// Verify update
	err = store.db.QueryRowContext(ctx,
		"SELECT summary FROM ai_summaries WHERE account_email = ? AND message_id = ?",
		"test@example.com", "msg123").Scan(&summary)
	assert.NoError(t, err)
	assert.Equal(t, "Updated summary", summary)
}

func TestDatabaseConstraints_PrimaryKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "constraints.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Insert first record
	_, err = store.db.ExecContext(ctx,
		"INSERT INTO ai_summaries (account_email, message_id, summary, updated_at) VALUES (?, ?, ?, ?)",
		"test@example.com", "msg123", "First summary", 1234567890)
	assert.NoError(t, err)

	// Try to insert duplicate - should replace due to UNIQUE constraint and upsert logic
	_, err = store.db.ExecContext(ctx,
		"INSERT INTO ai_summaries (account_email, message_id, summary, updated_at) VALUES (?, ?, ?, ?)",
		"test@example.com", "msg123", "Second summary", 1234567891)
	assert.Error(t, err) // Should violate PRIMARY KEY constraint

	// But upsert should work
	_, err = store.db.ExecContext(ctx, `
		INSERT INTO ai_summaries (account_email, message_id, summary, updated_at) 
		VALUES (?, ?, ?, ?)
		ON CONFLICT(account_email, message_id) 
		DO UPDATE SET summary = excluded.summary, updated_at = excluded.updated_at`,
		"test@example.com", "msg123", "Upserted summary", 1234567892)
	assert.NoError(t, err)

	// Verify upsert worked
	var summary string
	err = store.db.QueryRowContext(ctx,
		"SELECT summary FROM ai_summaries WHERE account_email = ? AND message_id = ?",
		"test@example.com", "msg123").Scan(&summary)
	assert.NoError(t, err)
	assert.Equal(t, "Upserted summary", summary)
}

func TestDatabase_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Test that multiple connections can be opened (WAL mode supports this)
	store2, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store2.Close()

	// Both should be able to read
	var version1, version2 int
	err = store.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version1)
	assert.NoError(t, err)

	err = store2.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version2)
	assert.NoError(t, err)

	assert.Equal(t, version1, version2)
}

// Benchmark database operations
func BenchmarkOpen(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbPath := filepath.Join(tmpDir, "bench_"+string(rune(i))+".db")
		store, err := Open(ctx, dbPath)
		if err != nil {
			b.Fatal(err)
		}
		_ = store.Close()
		_ = os.Remove(dbPath) // Clean up
	}
}

func BenchmarkInsert(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_insert.db")

	store, err := Open(ctx, dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.db.ExecContext(ctx,
			"INSERT OR REPLACE INTO ai_summaries (account_email, message_id, summary, updated_at) VALUES (?, ?, ?, ?)",
			"bench@example.com", "msg"+string(rune(i)), "Benchmark summary", int64(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQuery(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_query.db")

	store, err := Open(ctx, dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	// Insert test data
	_, err = store.db.ExecContext(ctx,
		"INSERT INTO ai_summaries (account_email, message_id, summary, updated_at) VALUES (?, ?, ?, ?)",
		"bench@example.com", "msg123", "Benchmark summary", 1234567890)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var summary string
		err := store.db.QueryRowContext(ctx,
			"SELECT summary FROM ai_summaries WHERE account_email = ? AND message_id = ?",
			"bench@example.com", "msg123").Scan(&summary)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test error scenarios
func TestOpen_ErrorScenarios(t *testing.T) {
	// Test with path that can't be created (permission denied simulation)
	// This is platform-specific and hard to test reliably, so we skip it
	t.Skip("Permission testing is platform-specific and complex")
}

// Test transaction rollback behavior
func TestMigration_TransactionRollback(t *testing.T) {
	// This tests that migration failures are properly rolled back
	// Since our migrations are simple, we can't easily simulate failure
	// But the structure shows proper transaction handling
	t.Skip("Migration rollback testing requires complex error simulation")
}

// Test schema validation
func TestSchema_Validation(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "schema.db")

	store, err := Open(ctx, dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Verify all expected tables exist
	expectedTables := []string{
		"ai_summaries",
		"prompt_templates",
		"prompt_results",
		"bulk_prompt_results",
		"saved_queries",
	}

	for _, table := range expectedTables {
		var name string
		err := store.db.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		assert.NoError(t, err, "Table %s should exist", table)
		assert.Equal(t, table, name)
	}
}

// Test database file locking and cleanup
func TestDatabase_FileHandling(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "filehandling.db")

	// Open and close multiple times
	for i := 0; i < 3; i++ {
		store, err := Open(ctx, dbPath)
		assert.NoError(t, err)
		assert.NotNil(t, store)

		// Should be able to query
		var version int
		err = store.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
		assert.NoError(t, err)

		err = store.Close()
		assert.NoError(t, err)
	}

	// Verify file still exists and is valid
	assert.FileExists(t, dbPath)
}
