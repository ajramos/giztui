package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/pkg/auth"
)

// AccountServiceImpl implements the AccountService interface
type AccountServiceImpl struct {
	config   *config.Config
	accounts map[string]*Account      // accountID -> Account
	clients  map[string]*gmail.Client // accountID -> Client
	activeID string
	mu       sync.RWMutex
}

// NewAccountService creates a new AccountService instance
func NewAccountService(cfg *config.Config) *AccountServiceImpl {
	service := &AccountServiceImpl{
		config:   cfg,
		accounts: make(map[string]*Account),
		clients:  make(map[string]*gmail.Client),
	}

	// Initialize accounts from config
	service.loadAccountsFromConfig()

	return service
}

// loadAccountsFromConfig initializes accounts from configuration
func (s *AccountServiceImpl) loadAccountsFromConfig() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load multi-account configuration
	for _, accountCfg := range s.config.Accounts {
		account := &Account{
			ID:          accountCfg.ID,
			DisplayName: accountCfg.DisplayName,
			CredPath:    accountCfg.Credentials,
			TokenPath:   accountCfg.Token,
			IsActive:    accountCfg.Active,
			Status:      AccountStatusUnknown,
			LastUsed:    time.Now(),
		}

		// Try to extract email from existing token if possible
		if email := s.extractEmailFromToken(account.TokenPath); email != "" {
			account.Email = email
		}

		s.accounts[account.ID] = account
		if account.IsActive {
			s.activeID = account.ID
		}
	}

	// Backward compatibility: if no accounts configured, create default from legacy config
	if len(s.accounts) == 0 && (s.config.Credentials != "" || s.config.Token != "") {
		defaultAccount := &Account{
			ID:          "default",
			DisplayName: "Default Account",
			CredPath:    s.config.Credentials,
			TokenPath:   s.config.Token,
			IsActive:    true,
			Status:      AccountStatusUnknown,
			LastUsed:    time.Now(),
		}

		// Try to extract email from existing token if possible
		if email := s.extractEmailFromToken(defaultAccount.TokenPath); email != "" {
			defaultAccount.Email = email
		}

		s.accounts["default"] = defaultAccount
		s.activeID = "default"
	}
}

// ListAccounts returns all configured accounts
func (s *AccountServiceImpl) ListAccounts(ctx context.Context) ([]*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]*Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		// Create a copy to avoid data races
		accountCopy := *account
		accounts = append(accounts, &accountCopy)
	}

	return accounts, nil
}

// GetActiveAccount returns the currently active account
func (s *AccountServiceImpl) GetActiveAccount(ctx context.Context) (*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.activeID == "" {
		return nil, fmt.Errorf("no active account configured")
	}

	account, exists := s.accounts[s.activeID]
	if !exists {
		return nil, fmt.Errorf("active account %s not found", s.activeID)
	}

	// Return a copy to avoid data races
	accountCopy := *account
	return &accountCopy, nil
}

// GetAccount returns a specific account by ID
func (s *AccountServiceImpl) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, exists := s.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	// Return a copy to avoid data races
	accountCopy := *account
	return &accountCopy, nil
}

// SwitchAccount switches to a different account
func (s *AccountServiceImpl) SwitchAccount(ctx context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate account exists
	account, exists := s.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Deactivate current account
	if s.activeID != "" {
		if currentAccount, exists := s.accounts[s.activeID]; exists {
			currentAccount.IsActive = false
		}
	}

	// Activate new account
	account.IsActive = true
	account.LastUsed = time.Now()
	s.activeID = accountID

	// Initialize client for new account if needed
	if _, exists := s.clients[accountID]; !exists {
		if err := s.initializeClient(ctx, accountID); err != nil {
			return fmt.Errorf("failed to initialize client for account %s: %w", accountID, err)
		}
	}

	return nil
}

// AddAccount adds a new account
func (s *AccountServiceImpl) AddAccount(ctx context.Context, account *Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate account
	if account.ID == "" {
		return fmt.Errorf("account ID cannot be empty")
	}

	// Check for duplicate ID
	if _, exists := s.accounts[account.ID]; exists {
		return fmt.Errorf("account with ID %s already exists", account.ID)
	}

	// Validate paths exist
	if account.CredPath != "" {
		credPath := s.expandPath(account.CredPath)
		if _, err := os.Stat(credPath); err != nil {
			return fmt.Errorf("credentials file not found: %s", credPath)
		}
	}

	// Set defaults
	if account.DisplayName == "" {
		account.DisplayName = account.ID
	}
	account.Status = AccountStatusUnknown
	account.LastUsed = time.Now()

	// Add to accounts map
	s.accounts[account.ID] = account

	return nil
}

// RemoveAccount removes an account
func (s *AccountServiceImpl) RemoveAccount(ctx context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate account exists
	account, exists := s.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Cannot remove the only account
	if len(s.accounts) == 1 {
		return fmt.Errorf("cannot remove the only account")
	}

	// If removing active account, switch to another
	if account.IsActive {
		// Find another account to activate
		for id, otherAccount := range s.accounts {
			if id != accountID {
				otherAccount.IsActive = true
				s.activeID = id
				break
			}
		}
	}

	// Remove from maps
	delete(s.accounts, accountID)
	delete(s.clients, accountID)

	return nil
}

// UpdateAccount updates an existing account
func (s *AccountServiceImpl) UpdateAccount(ctx context.Context, account *Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate account exists
	existingAccount, exists := s.accounts[account.ID]
	if !exists {
		return fmt.Errorf("account %s not found", account.ID)
	}

	// Update fields
	existingAccount.DisplayName = account.DisplayName
	existingAccount.CredPath = account.CredPath
	existingAccount.TokenPath = account.TokenPath
	existingAccount.Email = account.Email
	existingAccount.Status = account.Status

	// If paths changed, clear client to force re-initialization
	delete(s.clients, account.ID)

	return nil
}

// ConfigureAccount runs the interactive account configuration wizard
func (s *AccountServiceImpl) ConfigureAccount(ctx context.Context, accountID string) (*AccountSetupResult, error) {
	// This will be implemented in Phase 3 - Account Configuration
	return &AccountSetupResult{
		Success:  false,
		ErrorMsg: "account configuration wizard not yet implemented",
		NextStep: "manual_setup",
	}, nil
}

// ValidateAccount validates an account's configuration and connectivity
func (s *AccountServiceImpl) ValidateAccount(ctx context.Context, accountID string) (*AccountValidationResult, error) {
	s.mu.RLock()
	account, exists := s.accounts[accountID]
	s.mu.RUnlock()

	if !exists {
		return &AccountValidationResult{
			IsValid:    false,
			Status:     AccountStatusError,
			ErrorMsg:   fmt.Sprintf("account %s not found", accountID),
			LastTested: time.Now(),
		}, nil
	}

	// Try to initialize/get client
	client, err := s.GetAccountClient(ctx, accountID)
	if err != nil {
		s.mu.Lock()
		account.Status = AccountStatusError
		s.mu.Unlock()

		return &AccountValidationResult{
			IsValid:    false,
			Status:     AccountStatusError,
			ErrorMsg:   err.Error(),
			LastTested: time.Now(),
		}, nil
	}

	// Test connectivity by getting profile
	email, err := client.ActiveAccountEmail(ctx)
	if err != nil {
		s.mu.Lock()
		account.Status = AccountStatusDisconnected
		s.mu.Unlock()

		return &AccountValidationResult{
			IsValid:    false,
			Status:     AccountStatusDisconnected,
			ErrorMsg:   fmt.Sprintf("failed to connect: %v", err),
			LastTested: time.Now(),
		}, nil
	}

	// Update account with successful validation
	s.mu.Lock()
	account.Status = AccountStatusConnected
	account.Email = email
	s.mu.Unlock()

	return &AccountValidationResult{
		IsValid:    true,
		Status:     AccountStatusConnected,
		Email:      email,
		LastTested: time.Now(),
	}, nil
}

// GetAccountClient returns the Gmail client for a specific account
func (s *AccountServiceImpl) GetAccountClient(ctx context.Context, accountID string) (*gmail.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return existing client if available
	if client, exists := s.clients[accountID]; exists {
		return client, nil
	}

	// Initialize new client
	if err := s.initializeClient(ctx, accountID); err != nil {
		return nil, err
	}

	return s.clients[accountID], nil
}

// RefreshAccountClient refreshes the Gmail client for an account
func (s *AccountServiceImpl) RefreshAccountClient(ctx context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing client to force re-initialization
	delete(s.clients, accountID)

	// Initialize new client
	return s.initializeClient(ctx, accountID)
}

// initializeClient initializes a Gmail client for an account (must be called with lock held)
func (s *AccountServiceImpl) initializeClient(ctx context.Context, accountID string) error {
	account, exists := s.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Validate paths
	if account.CredPath == "" || account.TokenPath == "" {
		return fmt.Errorf("credentials or token path not configured for account %s", accountID)
	}

	// Expand paths
	credPath := s.expandPath(account.CredPath)
	tokenPath := s.expandPath(account.TokenPath)

	// Create Gmail service
	service, err := auth.NewGmailService(ctx, credPath, tokenPath,
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/gmail.modify",
		"https://www.googleapis.com/auth/gmail.compose",
		"https://www.googleapis.com/auth/calendar.events",
	)
	if err != nil {
		account.Status = AccountStatusError
		return fmt.Errorf("failed to initialize Gmail service for account %s: %w", accountID, err)
	}

	// Create Gmail client
	client := gmail.NewClient(service)
	s.clients[accountID] = client
	account.Client = client
	account.Status = AccountStatusConnected

	return nil
}

// expandPath expands ~ to home directory
func (s *AccountServiceImpl) expandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[2:])
}

// extractEmailFromToken attempts to extract email from an existing token file
func (s *AccountServiceImpl) extractEmailFromToken(tokenPath string) string {
	if tokenPath == "" {
		return ""
	}

	// Expand the path
	expandedPath := s.expandPath(tokenPath)

	// Check if token file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return ""
	}

	// Try to read and parse the token file to extract email
	// This is a best-effort approach - if it fails, we just return empty
	// #nosec G304 - This is reading user's own token file from config
	tokenData, err := os.ReadFile(expandedPath)
	if err != nil {
		return ""
	}

	// Simple JSON parsing to find "email" field in the token
	// OAuth2 tokens often contain user info including email
	var tokenInfo map[string]interface{}
	if err := json.Unmarshal(tokenData, &tokenInfo); err != nil {
		return ""
	}

	// Try different possible locations for email in token
	if email, ok := tokenInfo["email"].(string); ok && email != "" {
		return email
	}

	// Some tokens store user info in a nested structure
	if extra, ok := tokenInfo["extra"].(map[string]interface{}); ok {
		if email, ok := extra["email"].(string); ok && email != "" {
			return email
		}
	}

	return ""
}
