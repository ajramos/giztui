package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/db"
)

// PromptServiceImpl implements PromptService
type PromptServiceImpl struct {
	store       *db.PromptStore
	aiService   AIService
	bulkService *BulkPromptServiceImpl
}

// NewPromptService creates a new prompt service
func NewPromptService(store *db.PromptStore, aiService AIService, bulkService *BulkPromptServiceImpl) *PromptServiceImpl {
	return &PromptServiceImpl{
		store:       store,
		aiService:   aiService,
		bulkService: bulkService,
	}
}

func (s *PromptServiceImpl) ListPrompts(ctx context.Context, category string) ([]*PromptTemplate, error) {
	if s.store == nil {
		return nil, fmt.Errorf("cache store not available")
	}

	return s.store.ListPromptTemplates(ctx, category)
}

func (s *PromptServiceImpl) GetPrompt(ctx context.Context, id int) (*PromptTemplate, error) {
	if s.store == nil {
		return nil, fmt.Errorf("cache store not available")
	}

	return s.store.GetPromptTemplate(ctx, id)
}

func (s *PromptServiceImpl) ApplyPrompt(ctx context.Context, messageContent string, promptID int, variables map[string]string) (*PromptResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("cache store not available")
	}

	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	// Get the prompt template
	template, err := s.store.GetPromptTemplate(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt template: %w", err)
	}

	// Replace variables in the prompt
	prompt := template.PromptText

	// Always replace {{body}} with the message content
	if variables == nil {
		variables = make(map[string]string)
	}
	variables["body"] = messageContent

	// Replace all variables in the prompt
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	// Apply the prompt using the AI service
	result, err := s.aiService.ApplyCustomPrompt(ctx, messageContent, prompt, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to apply prompt: %w", err)
	}

	// Increment usage count
	_ = s.store.IncrementPromptUsage(ctx, promptID)

	return &PromptResult{
		PromptID:   promptID,
		ResultText: result,
	}, nil
}

func (s *PromptServiceImpl) IncrementUsage(ctx context.Context, promptID int) error {
	if s.store == nil {
		return fmt.Errorf("cache store not available")
	}

	return s.store.IncrementPromptUsage(ctx, promptID)
}

// SetBulkService sets the bulk service reference
func (s *PromptServiceImpl) SetBulkService(bulkService *BulkPromptServiceImpl) {
	s.bulkService = bulkService
}

func (s *PromptServiceImpl) SaveResult(ctx context.Context, accountEmail, messageID string, promptID int, resultText string) error {
	if s.store == nil {
		return fmt.Errorf("cache store not available")
	}

	return s.store.SavePromptResult(ctx, accountEmail, messageID, promptID, resultText)
}

// ApplyBulkPrompt applies a prompt to multiple messages
func (s *PromptServiceImpl) ApplyBulkPrompt(ctx context.Context, accountEmail string, messageIDs []string, promptID int, variables map[string]string) (*BulkPromptResult, error) {
	if s.bulkService == nil {
		return nil, fmt.Errorf("bulk prompt service not available")
	}
	return s.bulkService.ApplyBulkPrompt(ctx, accountEmail, messageIDs, promptID, variables)
}

// GetCachedBulkResult retrieves a cached bulk prompt result
func (s *PromptServiceImpl) GetCachedBulkResult(ctx context.Context, accountEmail string, messageIDs []string, promptID int) (*BulkPromptResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("store not available")
	}

	// Create a cache key consistent with bulk prompt service
	sortedIDs := make([]string, len(messageIDs))
	copy(sortedIDs, messageIDs)
	sort.Strings(sortedIDs)
	cacheKey := fmt.Sprintf("bulk_%d_%s", promptID, strings.Join(sortedIDs, "_"))

	// Get from dedicated bulk prompt results table
	dbResult, err := s.store.GetBulkPromptResult(ctx, accountEmail, cacheKey)
	if err != nil {
		return nil, err
	}
	if dbResult == nil {
		return nil, nil // No cached result found
	}

	// Convert DB result to service result
	messageIDsList := strings.Split(dbResult.MessageIDs, ",")
	return &BulkPromptResult{
		PromptID:     dbResult.PromptID,
		MessageCount: dbResult.MessageCount,
		Summary:      dbResult.ResultText,
		MessageIDs:   messageIDsList,
		Duration:     0, // No duration info stored
		FromCache:    true,
		CreatedAt:    time.Unix(dbResult.CreatedAt, 0),
	}, nil
}

// SaveBulkResult saves a bulk prompt result
func (s *PromptServiceImpl) SaveBulkResult(ctx context.Context, accountEmail string, messageIDs []string, promptID int, resultText string) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}

	// Create a cache key consistent with bulk prompt service
	sortedIDs := make([]string, len(messageIDs))
	copy(sortedIDs, messageIDs)
	sort.Strings(sortedIDs)
	cacheKey := fmt.Sprintf("bulk_%d_%s", promptID, strings.Join(sortedIDs, "_"))

	// Save to dedicated bulk prompt results table
	return s.store.SaveBulkPromptResult(ctx, accountEmail, cacheKey, promptID, len(messageIDs), messageIDs, resultText)
}

func (s *PromptServiceImpl) GetCachedResult(ctx context.Context, accountEmail, messageID string, promptID int) (*PromptResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("cache store not available")
	}

	return s.store.GetPromptResult(ctx, accountEmail, messageID, promptID)
}

// ApplyBulkPromptStream delegates to the bulk prompt service with streaming
func (s *PromptServiceImpl) ApplyBulkPromptStream(ctx context.Context, accountEmail string, messageIDs []string, promptID int, variables map[string]string, onToken func(string)) (*BulkPromptResult, error) {
	if s.bulkService == nil {
		return nil, fmt.Errorf("bulk prompt service not available")
	}
	return s.bulkService.ApplyBulkPromptStream(ctx, accountEmail, messageIDs, promptID, variables, onToken)
}

// ClearPromptCache clears all prompt results for the given account
func (s *PromptServiceImpl) ClearPromptCache(ctx context.Context, accountEmail string) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}
	return s.store.ClearPromptCache(ctx, accountEmail)
}

// ClearAllPromptCaches clears all prompt results for all accounts (admin function)
func (s *PromptServiceImpl) ClearAllPromptCaches(ctx context.Context) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}
	return s.store.ClearAllPromptCaches(ctx)
}
