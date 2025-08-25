package services

import (
	"context"
	"testing"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMProvider implements llm.Provider for testing
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLLMProvider) Generate(prompt string) (string, error) {
	args := m.Called(prompt)
	return args.String(0), args.Error(1)
}

// MockCacheService implements CacheService interface for testing
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) GetSummary(ctx context.Context, accountEmail, messageID string) (string, bool, error) {
	args := m.Called(ctx, accountEmail, messageID)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockCacheService) SaveSummary(ctx context.Context, accountEmail, messageID, summary string) error {
	args := m.Called(ctx, accountEmail, messageID, summary)
	return args.Error(0)
}

func (m *MockCacheService) InvalidateSummary(ctx context.Context, accountEmail, messageID string) error {
	args := m.Called(ctx, accountEmail, messageID)
	return args.Error(0)
}

func (m *MockCacheService) ClearCache(ctx context.Context, accountEmail string) error {
	args := m.Called(ctx, accountEmail)
	return args.Error(0)
}

// Test AI Service constructor
func TestNewAIService(t *testing.T) {
	provider := &MockLLMProvider{}
	cacheService := &MockCacheService{}
	cfg := &config.Config{}
	
	service := NewAIService(provider, cacheService, cfg)
	
	assert.NotNil(t, service)
	assert.Equal(t, provider, service.provider)
	assert.Equal(t, cacheService, service.cacheService)
	assert.Equal(t, cfg, service.config)
}

func TestNewAIService_NilInputs(t *testing.T) {
	service := NewAIService(nil, nil, nil)
	assert.NotNil(t, service)
	assert.Nil(t, service.provider)
	assert.Nil(t, service.cacheService)
	assert.Nil(t, service.config)
}

// Test GenerateSummary validation errors
func TestAIServiceImpl_GenerateSummary_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	
	t.Run("nil_provider", func(t *testing.T) {
		service := &AIServiceImpl{provider: nil}
		
		result, err := service.GenerateSummary(ctx, "test content", SummaryOptions{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "AI provider not available")
	})
	
	t.Run("empty_content", func(t *testing.T) {
		provider := &MockLLMProvider{}
		service := &AIServiceImpl{provider: provider}
		
		result, err := service.GenerateSummary(ctx, "", SummaryOptions{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "content cannot be empty")
	})
	
	t.Run("whitespace_only_content", func(t *testing.T) {
		provider := &MockLLMProvider{}
		service := &AIServiceImpl{provider: provider}
		
		result, err := service.GenerateSummary(ctx, "   \n\t  ", SummaryOptions{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "content cannot be empty")
	})
}

// Test GenerateSummary without caching
func TestAIServiceImpl_GenerateSummary_NoCache(t *testing.T) {
	ctx := context.Background()
	provider := &MockLLMProvider{}
	cfg := &config.Config{}
	
	service := NewAIService(provider, nil, cfg) // No cache service
	
	// Setup provider to return generated summary
	expectedPrompt := "Briefly summarize the following email. Keep it concise and factual.\n\ntest content"
	provider.On("Generate", expectedPrompt).Return("Generated summary", nil)
	
	options := SummaryOptions{
		UseCache: false, // Caching disabled
	}
	
	result, err := service.GenerateSummary(ctx, "test content", options)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Generated summary", result.Summary)
	assert.False(t, result.FromCache)
	
	provider.AssertExpectations(t)
}

// Test GenerateSummary with cache hit
func TestAIServiceImpl_GenerateSummary_CacheHit(t *testing.T) {
	ctx := context.Background()
	provider := &MockLLMProvider{}
	cacheService := &MockCacheService{}
	cfg := &config.Config{}
	
	service := NewAIService(provider, cacheService, cfg)
	
	// Setup cache to return cached summary
	cacheService.On("GetSummary", ctx, "test@example.com", "msg123").Return("Cached summary", true, nil)
	
	options := SummaryOptions{
		UseCache:     true,
		AccountEmail: "test@example.com",
		MessageID:    "msg123",
	}
	
	result, err := service.GenerateSummary(ctx, "test content", options)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Cached summary", result.Summary)
	assert.True(t, result.FromCache)
	assert.True(t, result.Duration > 0)
	
	// Provider should not be called since we got cache hit
	provider.AssertNotCalled(t, "Generate")
	cacheService.AssertExpectations(t)
}

// Test GenerateSummary with cache miss
func TestAIServiceImpl_GenerateSummary_CacheMiss(t *testing.T) {
	ctx := context.Background()
	provider := &MockLLMProvider{}
	cacheService := &MockCacheService{}
	cfg := &config.Config{
		LLM: config.LLMConfig{
			SummarizePrompt: "Custom prompt: {{body}}",
		},
	}
	
	service := NewAIService(provider, cacheService, cfg)
	
	// Setup cache to return cache miss
	cacheService.On("GetSummary", ctx, "test@example.com", "msg123").Return("", false, nil)
	// Setup provider to return generated summary
	provider.On("Generate", "Custom prompt: test content").Return("Generated summary", nil)
	// Setup cache to save the generated summary
	cacheService.On("SaveSummary", ctx, "test@example.com", "msg123", "Generated summary").Return(nil)
	
	options := SummaryOptions{
		UseCache:     true,
		AccountEmail: "test@example.com",
		MessageID:    "msg123",
	}
	
	result, err := service.GenerateSummary(ctx, "test content", options)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Generated summary", result.Summary)
	assert.False(t, result.FromCache)
	assert.True(t, result.Duration > 0)
	
	provider.AssertExpectations(t)
	cacheService.AssertExpectations(t)
}