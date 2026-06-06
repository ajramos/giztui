package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAIService is a local testify mock for the AIService interface.
type mockAIService struct {
	mock.Mock
}

func (m *mockAIService) GenerateSummary(ctx context.Context, content string, options SummaryOptions) (*SummaryResult, error) {
	args := m.Called(ctx, content, options)
	if v := args.Get(0); v != nil {
		return v.(*SummaryResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAIService) GenerateSummaryStream(ctx context.Context, content string, options SummaryOptions, onToken func(string)) (*SummaryResult, error) {
	args := m.Called(ctx, content, options, onToken)
	if v := args.Get(0); v != nil {
		return v.(*SummaryResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAIService) GenerateReply(ctx context.Context, content string, options ReplyOptions) (string, error) {
	args := m.Called(ctx, content, options)
	return args.String(0), args.Error(1)
}

func (m *mockAIService) SuggestLabels(ctx context.Context, content string, availableLabels []string) ([]string, error) {
	args := m.Called(ctx, content, availableLabels)
	if v := args.Get(0); v != nil {
		return v.([]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAIService) FormatContent(ctx context.Context, content string, options FormatOptions) (string, error) {
	args := m.Called(ctx, content, options)
	return args.String(0), args.Error(1)
}

func (m *mockAIService) ApplyCustomPrompt(ctx context.Context, content string, prompt string, variables map[string]string) (string, error) {
	args := m.Called(ctx, content, prompt, variables)
	return args.String(0), args.Error(1)
}

func (m *mockAIService) ApplyCustomPromptStream(ctx context.Context, content string, prompt string, variables map[string]string, onToken func(string)) (string, error) {
	args := m.Called(ctx, content, prompt, variables, onToken)
	return args.String(0), args.Error(1)
}

// TestNewPromptGeneratorService verifies the constructor stores the AIService.
func TestNewPromptGeneratorService(t *testing.T) {
	service := NewPromptGeneratorService(nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.aiService)
}

// TestPromptGeneratorServiceImpl_GenerateFromIntent_NilAIService verifies error path.
func TestPromptGeneratorServiceImpl_GenerateFromIntent_NilAIService(t *testing.T) {
	service := &PromptGeneratorServiceImpl{aiService: nil}

	result, err := service.GenerateFromIntent(context.Background(), "find urgent emails", PromptGenerationOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestPromptGeneratorServiceImpl_parseGeneratedOutput_HappyPath verifies parsing.
func TestPromptGeneratorServiceImpl_parseGeneratedOutput_HappyPath(t *testing.T) {
	service := &PromptGeneratorServiceImpl{}

	raw := `You are an email triage assistant.
Analyze the email {{body}} and identify urgency.

__NAME__: triage-urgency
__DESC__: classify single email by urgency level
__MODE__: single`

	result := service.parseGeneratedOutput(raw, 0)

	assert.Equal(t, "triage-urgency", result.SuggestedName)
	assert.Equal(t, "classify single email by urgency level", result.SuggestedDesc)
	assert.Equal(t, "single", result.DetectedMode)
	assert.Contains(t, result.PromptText, "{{body}}")
	assert.NotContains(t, result.PromptText, "__NAME__")
}

// TestPromptGeneratorServiceImpl_parseGeneratedOutput_MissingMetadata verifies graceful handling.
func TestPromptGeneratorServiceImpl_parseGeneratedOutput_MissingMetadata(t *testing.T) {
	service := &PromptGeneratorServiceImpl{}

	raw := `You are an email assistant. Analyze {{body}} please.`

	result := service.parseGeneratedOutput(raw, 0)

	assert.Equal(t, "", result.SuggestedName)
	assert.Equal(t, "", result.SuggestedDesc)
	assert.Equal(t, "", result.DetectedMode)
	assert.Equal(t, "You are an email assistant. Analyze {{body}} please.", result.PromptText)
}

// TestPromptGeneratorServiceImpl_buildGenerationPrompt_IncludesIntent verifies meta-prompt assembly.
func TestPromptGeneratorServiceImpl_buildGenerationPrompt_IncludesIntent(t *testing.T) {
	service := &PromptGeneratorServiceImpl{}

	prompt := service.buildGenerationPrompt("identify urgent replies needed", PromptGenerationOptions{
		OutputFormat: "json",
		TargetMode:   "bulk",
	})

	assert.Contains(t, prompt, "identify urgent replies needed")
	assert.Contains(t, prompt, "json")
	assert.Contains(t, prompt, "bulk")
	assert.Contains(t, prompt, "__NAME__")
	assert.Contains(t, prompt, "{{body}}")
	assert.Contains(t, prompt, "{{messages}}")
}

// TestPromptGeneratorServiceImpl_GenerateFromIntent_Success verifies a happy path with a mocked AI.
func TestPromptGeneratorServiceImpl_GenerateFromIntent_Success(t *testing.T) {
	mockAI := &mockAIService{}

	canned := `Analyze the email {{body}} and identify urgency.

__NAME__: triage-urgency
__DESC__: classify by urgency
__MODE__: single`

	mockAI.On("ApplyCustomPrompt",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return(canned, nil)

	service := NewPromptGeneratorService(mockAI)

	result, err := service.GenerateFromIntent(context.Background(), "classify by urgency", PromptGenerationOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "triage-urgency", result.SuggestedName)
	assert.Equal(t, "single", result.DetectedMode)
	assert.Contains(t, result.PromptText, "{{body}}")
}

// TestPromptGeneratorServiceImpl_GenerateFromIntent_AIServiceFailure verifies error surfacing.
func TestPromptGeneratorServiceImpl_GenerateFromIntent_AIServiceFailure(t *testing.T) {
	mockAI := &mockAIService{}

	mockAI.On("ApplyCustomPrompt",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return("", assert.AnError)

	service := NewPromptGeneratorService(mockAI)

	result, err := service.GenerateFromIntent(context.Background(), "anything", PromptGenerationOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestPromptGeneratorServiceImpl_RefinePrompt_Success verifies refinement happy path.
func TestPromptGeneratorServiceImpl_RefinePrompt_Success(t *testing.T) {
	mockAI := &mockAIService{}

	refined := `Analyze the email {{body}} and output ONLY valid JSON with field "urgency_level".

__NAME__: triage-urgency-json
__DESC__: classify urgency as JSON
__MODE__: single`

	mockAI.On("ApplyCustomPrompt",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return(refined, nil)

	service := NewPromptGeneratorService(mockAI)

	current := "Analyze the email {{body}} and identify urgency."
	result, err := service.RefinePrompt(context.Background(), current, "output as JSON", PromptGenerationOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.PromptText, "JSON")
	assert.Equal(t, "triage-urgency-json", result.SuggestedName)
}

// TestPromptGeneratorServiceImpl_RefinePrompt_EmptyCurrent verifies validation.
func TestPromptGeneratorServiceImpl_RefinePrompt_EmptyCurrent(t *testing.T) {
	service := NewPromptGeneratorService(nil)

	result, err := service.RefinePrompt(context.Background(), "", "make it stricter", PromptGenerationOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
}
