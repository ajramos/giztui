package services

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PromptGeneratorServiceImpl implements PromptGeneratorService using AIService.
type PromptGeneratorServiceImpl struct {
	aiService AIService
}

// NewPromptGeneratorService creates a new generator service.
func NewPromptGeneratorService(aiService AIService) *PromptGeneratorServiceImpl {
	return &PromptGeneratorServiceImpl{aiService: aiService}
}

// GenerateFromIntent produces a prompt template from natural-language intent.
func (s *PromptGeneratorServiceImpl) GenerateFromIntent(ctx context.Context, intent string, opts PromptGenerationOptions) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// RefinePrompt applies a refinement to an existing prompt.
func (s *PromptGeneratorServiceImpl) RefinePrompt(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateFromIntentStream is the streaming version of GenerateFromIntent.
func (s *PromptGeneratorServiceImpl) GenerateFromIntentStream(ctx context.Context, intent string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// RefinePromptStream is the streaming version of RefinePrompt.
func (s *PromptGeneratorServiceImpl) RefinePromptStream(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// buildGenerationPrompt constructs the meta-prompt sent to the LLM for intent-based generation.
func (s *PromptGeneratorServiceImpl) buildGenerationPrompt(intent string, opts PromptGenerationOptions) string {
	var b strings.Builder
	b.WriteString("You are a prompt engineer. The user describes an intent and you write a clean, reusable prompt template they can apply to email content.\n\n")
	b.WriteString("Rules for your output:\n")
	b.WriteString("1. Use {{body}} as placeholder for a single email body, or {{messages}} for multiple email bodies, depending on the intent.\n")
	b.WriteString("2. Return ONLY the prompt template followed by metadata on separate lines.\n")
	b.WriteString("3. Metadata format (each on its own line, after the template):\n")
	b.WriteString("   __NAME__: short kebab-case name (max 40 chars)\n")
	b.WriteString("   __DESC__: one-line description (max 120 chars)\n")
	b.WriteString("   __MODE__: one of single|bulk|analyzer\n\n")
	if opts.OutputFormat != "" {
		fmt.Fprintf(&b, "Constraint: the prompt MUST instruct the LLM to output in %s format.\n", opts.OutputFormat)
	}
	if opts.TargetMode != "" {
		fmt.Fprintf(&b, "Constraint: target this prompt for the %s context.\n", opts.TargetMode)
	}
	fmt.Fprintf(&b, "\nUser intent:\n%s\n", intent)
	return b.String()
}

// buildRefinementPrompt constructs the meta-prompt for refinement.
func (s *PromptGeneratorServiceImpl) buildRefinementPrompt(currentPrompt string, refinement string, opts PromptGenerationOptions) string {
	var b strings.Builder
	b.WriteString("You are a prompt engineer. The user has a current prompt and wants to apply a refinement.\n\n")
	b.WriteString("Return the FULL revised prompt followed by the same metadata block as before:\n")
	b.WriteString("   __NAME__: short kebab-case name\n")
	b.WriteString("   __DESC__: one-line description\n")
	b.WriteString("   __MODE__: one of single|bulk|analyzer\n\n")
	fmt.Fprintf(&b, "Current prompt:\n%s\n\n", currentPrompt)
	fmt.Fprintf(&b, "Refinement requested:\n%s\n", refinement)
	return b.String()
}

// parseGeneratedOutput extracts the prompt body and metadata from the LLM response.
func (s *PromptGeneratorServiceImpl) parseGeneratedOutput(raw string, duration time.Duration) *GeneratedPrompt {
	result := &GeneratedPrompt{Duration: duration}
	lines := strings.Split(raw, "\n")
	var bodyLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "__NAME__:"):
			result.SuggestedName = strings.TrimSpace(strings.TrimPrefix(trimmed, "__NAME__:"))
		case strings.HasPrefix(trimmed, "__DESC__:"):
			result.SuggestedDesc = strings.TrimSpace(strings.TrimPrefix(trimmed, "__DESC__:"))
		case strings.HasPrefix(trimmed, "__MODE__:"):
			result.DetectedMode = strings.TrimSpace(strings.TrimPrefix(trimmed, "__MODE__:"))
		default:
			bodyLines = append(bodyLines, line)
		}
	}
	result.PromptText = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	return result
}
