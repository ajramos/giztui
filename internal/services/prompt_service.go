package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/db"
)

// PromptServiceImpl implements PromptService
type PromptServiceImpl struct {
	store     *db.PromptStore
	aiService AIService
}

// NewPromptService creates a new prompt service
func NewPromptService(store *db.PromptStore, aiService AIService) *PromptServiceImpl {
	return &PromptServiceImpl{
		store:     store,
		aiService: aiService,
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

func (s *PromptServiceImpl) SaveResult(ctx context.Context, accountEmail, messageID string, promptID int, resultText string) error {
	if s.store == nil {
		return fmt.Errorf("cache store not available")
	}

	return s.store.SavePromptResult(ctx, accountEmail, messageID, promptID, resultText)
}

func (s *PromptServiceImpl) GetCachedResult(ctx context.Context, accountEmail, messageID string, promptID int) (*PromptResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("cache store not available")
	}

	return s.store.GetPromptResult(ctx, accountEmail, messageID, promptID)
}
