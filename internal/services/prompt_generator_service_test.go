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

// TestPromptGeneratorServiceImpl_GenerateFromIntentStream_Success verifies tokens are emitted.
func TestPromptGeneratorServiceImpl_GenerateFromIntentStream_Success(t *testing.T) {
	mockAI := &mockAIService{}

	full := `Analyze {{body}}.

__NAME__: simple
__DESC__: simple prompt
__MODE__: single`

	mockAI.On("ApplyCustomPromptStream",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
		mock.AnythingOfType("func(string)"),
	).Run(func(args mock.Arguments) {
		// Invoke the streaming callback with chunks.
		cb := args.Get(4).(func(string))
		cb("Analyze ")
		cb("{{body}}.\n\n")
		cb("__NAME__: simple\n")
		cb("__DESC__: simple prompt\n")
		cb("__MODE__: single")
	}).Return(full, nil)

	service := NewPromptGeneratorService(mockAI)

	var tokens []string
	result, err := service.GenerateFromIntentStream(context.Background(), "describe", PromptGenerationOptions{}, func(t string) {
		tokens = append(tokens, t)
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []string{
		"Analyze ",
		"{{body}}.\n\n",
		"__NAME__: simple\n",
		"__DESC__: simple prompt\n",
		"__MODE__: single",
	}, tokens)
	assert.Equal(t, "simple", result.SuggestedName)
}

// TestPromptGeneratorServiceImpl_RefinePromptStream_Success verifies streaming refinement.
func TestPromptGeneratorServiceImpl_RefinePromptStream_Success(t *testing.T) {
	mockAI := &mockAIService{}

	refined := `Analyze {{body}}. Output JSON.

__NAME__: refined-json
__DESC__: refined to JSON output
__MODE__: single`

	mockAI.On("ApplyCustomPromptStream",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
		mock.AnythingOfType("func(string)"),
	).Run(func(args mock.Arguments) {
		cb := args.Get(4).(func(string))
		cb("Analyze ")
		cb("{{body}}. Output JSON.\n\n")
		cb("__NAME__: refined-json\n")
		cb("__DESC__: refined to JSON output\n")
		cb("__MODE__: single")
	}).Return(refined, nil)

	service := NewPromptGeneratorService(mockAI)

	var tokens []string
	result, err := service.RefinePromptStream(
		context.Background(),
		"Analyze {{body}}.",
		"output as JSON",
		PromptGenerationOptions{},
		func(t string) { tokens = append(tokens, t) },
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []string{
		"Analyze ",
		"{{body}}. Output JSON.\n\n",
		"__NAME__: refined-json\n",
		"__DESC__: refined to JSON output\n",
		"__MODE__: single",
	}, tokens)
	assert.Equal(t, "refined-json", result.SuggestedName)
	assert.Contains(t, result.PromptText, "JSON")
}

// TestPromptGeneratorServiceImpl_RefinePrompt_EmptyRefinement verifies the third guard fires
// when current is non-empty, aiService is present, but refinement is empty.
func TestPromptGeneratorServiceImpl_RefinePrompt_EmptyRefinement(t *testing.T) {
	mockAI := &mockAIService{}
	// Note: NO mockAI.On(...) — the call must never reach the AI service.

	service := NewPromptGeneratorService(mockAI)

	result, err := service.RefinePrompt(
		context.Background(),
		"Analyze {{body}}.", // non-empty current
		"",                  // empty refinement -- this is the guard under test
		PromptGenerationOptions{},
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "refinement")
}
