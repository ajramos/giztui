package services

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
)

// DatabaseManagerImpl implements DatabaseManager interface for multi-account database management
type DatabaseManagerImpl struct {
	config              *config.Config
	logger              *log.Logger
	mu                  sync.RWMutex
	currentStore        *db.Store
	currentAccountEmail string

	// Callback function to reinitialize database-dependent services
	serviceReinitCallback func(*db.Store) error
}

// NewDatabaseManager creates a new DatabaseManager instance
func NewDatabaseManager(config *config.Config, logger *log.Logger) *DatabaseManagerImpl {
	return &DatabaseManagerImpl{
		config: config,
		logger: logger,
	}
}

// SetServiceReinitCallback sets the callback function to reinitialize services when database changes
func (dm *DatabaseManagerImpl) SetServiceReinitCallback(callback func(*db.Store) error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.serviceReinitCallback = callback
}

// SwitchToAccountDatabase switches to the database for the specified account
func (dm *DatabaseManagerImpl) SwitchToAccountDatabase(ctx context.Context, accountEmail string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.logger != nil {
		dm.logger.Printf("DatabaseManager: switching to database for account: %s", accountEmail)
	}

	// If already using the correct database, no need to switch
	if dm.currentStore != nil && dm.currentAccountEmail == accountEmail {
		if dm.logger != nil {
			dm.logger.Printf("DatabaseManager: already using database for account %s, no switch needed", accountEmail)
		}
		return nil
	}

	// Close current database connection if exists
	if dm.currentStore != nil {
		if dm.logger != nil {
			dm.logger.Printf("DatabaseManager: closing current database for account: %s", dm.currentAccountEmail)
		}
		if err := dm.currentStore.Close(); err != nil {
			if dm.logger != nil {
				dm.logger.Printf("DatabaseManager: warning - failed to close current database: %v", err)
			}
		}
		dm.currentStore = nil
		dm.currentAccountEmail = ""
	}

	// Don't open database if LLM cache is disabled
	if !dm.config.LLM.CacheEnabled {
		if dm.logger != nil {
			dm.logger.Printf("DatabaseManager: LLM cache disabled, not opening database for account: %s", accountEmail)
		}
		return nil
	}

	// Determine database path for the account
	dbPath, err := dm.getDatabasePathForAccount(accountEmail)
	if err != nil {
		return fmt.Errorf("failed to determine database path for account %s: %w", accountEmail, err)
	}

	if dm.logger != nil {
		dm.logger.Printf("DatabaseManager: opening database at path: %s", dbPath)
	}

	// Open new database connection
	store, err := db.Open(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database for account %s at %s: %w", accountEmail, dbPath, err)
	}

	// Update current state
	dm.currentStore = store
	dm.currentAccountEmail = accountEmail

	if dm.logger != nil {
		dm.logger.Printf("DatabaseManager: successfully switched to database for account: %s", accountEmail)
	}

	// Reinitialize database-dependent services with new store
	if dm.serviceReinitCallback != nil {
		if dm.logger != nil {
			dm.logger.Printf("DatabaseManager: reinitializing database-dependent services")
		}
		if err := dm.serviceReinitCallback(store); err != nil {
			if dm.logger != nil {
				dm.logger.Printf("DatabaseManager: warning - failed to reinitialize services: %v", err)
			}
			return fmt.Errorf("failed to reinitialize services with new database: %w", err)
		}
		if dm.logger != nil {
			dm.logger.Printf("DatabaseManager: successfully reinitialized database-dependent services")
		}
	}

	return nil
}

// GetCurrentStore returns the currently active database store
func (dm *DatabaseManagerImpl) GetCurrentStore() *db.Store {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.currentStore
}

// Close closes the current database connection
func (dm *DatabaseManagerImpl) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.currentStore != nil {
		if dm.logger != nil {
			dm.logger.Printf("DatabaseManager: closing database for account: %s", dm.currentAccountEmail)
		}
		err := dm.currentStore.Close()
		dm.currentStore = nil
		dm.currentAccountEmail = ""
		return err
	}

	return nil
}

// IsInitialized returns true if a database is currently open
func (dm *DatabaseManagerImpl) IsInitialized() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.currentStore != nil
}

// GetCurrentAccountEmail returns the email of the account whose database is currently open
func (dm *DatabaseManagerImpl) GetCurrentAccountEmail() string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.currentAccountEmail
}

// getDatabasePathForAccount determines the database file path for a given account email
func (dm *DatabaseManagerImpl) getDatabasePathForAccount(accountEmail string) (string, error) {
	// Use the same logic as main.go to determine database path
	baseDir := config.DefaultCacheDir()
	if dm.config.LLM.CachePath != "" {
		baseDir = dm.config.LLM.CachePath
	}

	dbPath := baseDir
	if ext := filepath.Ext(baseDir); ext == "" || ext == "." {
		// Sanitize email for use as filename
		safe := strings.ToLower(strings.TrimSpace(accountEmail))
		safe = strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "_", " ", "_").Replace(safe)
		if safe == "" {
			safe = "default"
		}
		dbPath = filepath.Join(baseDir, safe+".sqlite3")
	}

	return dbPath, nil
}
