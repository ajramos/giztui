package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
