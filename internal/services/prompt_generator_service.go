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
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(intent) == "" {
		return nil, fmt.Errorf("intent cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildGenerationPrompt(intent, opts)

	// AIService validates content non-empty but only uses prompt; pass intent as content.
	raw, err := s.aiService.ApplyCustomPrompt(ctx, intent, metaPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}

// RefinePrompt applies a refinement to an existing prompt.
//
// Validation order is load-bearing: empty currentPrompt is checked first,
// then nil aiService, then empty refinement. This matches RefinePromptStream
// and is exercised by the EmptyCurrent and EmptyRefinement tests.
func (s *PromptGeneratorServiceImpl) RefinePrompt(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions) (*GeneratedPrompt, error) {
	if strings.TrimSpace(currentPrompt) == "" {
		return nil, fmt.Errorf("current prompt cannot be empty")
	}
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(refinement) == "" {
		return nil, fmt.Errorf("refinement instruction cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildRefinementPrompt(currentPrompt, refinement, opts)

	// AIService validates content non-empty but only uses prompt; pass currentPrompt as content.
	raw, err := s.aiService.ApplyCustomPrompt(ctx, currentPrompt, metaPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM refinement failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}

// GenerateFromIntentStream is the streaming version of GenerateFromIntent.
func (s *PromptGeneratorServiceImpl) GenerateFromIntentStream(ctx context.Context, intent string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(intent) == "" {
		return nil, fmt.Errorf("intent cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildGenerationPrompt(intent, opts)

	// AIService validates content non-empty but only uses prompt; pass intent as content.
	raw, err := s.aiService.ApplyCustomPromptStream(ctx, intent, metaPrompt, nil, func(token string) {
		if onToken != nil {
			onToken(token)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("LLM streaming generation failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}

// RefinePromptStream is the streaming version of RefinePrompt.
//
// Validation order is load-bearing: empty currentPrompt is checked first,
// then nil aiService, then empty refinement. Mirrors RefinePrompt.
func (s *PromptGeneratorServiceImpl) RefinePromptStream(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	if strings.TrimSpace(currentPrompt) == "" {
		return nil, fmt.Errorf("current prompt cannot be empty")
	}
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(refinement) == "" {
		return nil, fmt.Errorf("refinement instruction cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildRefinementPrompt(currentPrompt, refinement, opts)

	// AIService validates content non-empty but only uses prompt; pass currentPrompt as content.
	raw, err := s.aiService.ApplyCustomPromptStream(ctx, currentPrompt, metaPrompt, nil, func(token string) {
		if onToken != nil {
			onToken(token)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("LLM streaming refinement failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}

// buildGenerationPrompt constructs the meta-prompt sent to the LLM for intent-based generation.
func (s *PromptGeneratorServiceImpl) buildGenerationPrompt(intent string, opts PromptGenerationOptions) string {
	var b strings.Builder
	b.WriteString("You are a prompt engineer. The user describes what they want to do with their email, ")
	b.WriteString("and you write a reusable INSTRUCTION PROMPT that another AI assistant will run against the email content.\n\n")
	b.WriteString("The prompt you write MUST:\n")
	b.WriteString("- Be a complete, self-contained instruction telling the assistant what to do with the email.\n")
	b.WriteString("- Contain the placeholder {{body}} on its own line where the email body will be inserted ")
	b.WriteString("(use {{messages}} instead when the task is about multiple emails at once).\n")
	b.WriteString("- NEVER be just the placeholder alone. The placeholder is NOT the prompt — the instructions are.\n")
	b.WriteString("- Be written as if speaking to the assistant, not to the user.\n\n")
	b.WriteString("Example. If the user intent is \"summarize in 3 bullets\", a correct answer is:\n\n")
	b.WriteString("Summarize the following email in exactly 3 concise bullet points, capturing the key facts.\n\n")
	b.WriteString("{{body}}\n\n")
	b.WriteString("__NAME__: summarize-3-bullets\n")
	b.WriteString("__DESC__: Summarize an email into three bullet points\n")
	b.WriteString("__MODE__: single\n\n")
	b.WriteString("Now write the prompt for the user's intent below. Output ONLY the instruction prompt ")
	b.WriteString("(no code fences, no preamble) followed by the metadata block, each field on its own line:\n")
	b.WriteString("   __NAME__: short kebab-case name (max 40 chars)\n")
	b.WriteString("   __DESC__: one-line description (max 120 chars)\n")
	b.WriteString("   __MODE__: one of single|bulk|analyzer\n\n")
	if opts.OutputFormat != "" {
		fmt.Fprintf(&b, "Constraint: the prompt MUST instruct the assistant to output in %s format.\n", opts.OutputFormat)
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
	b.WriteString("You are a prompt engineer. Below is an existing INSTRUCTION PROMPT and a refinement the user wants applied.\n\n")
	b.WriteString("Rewrite the prompt applying the refinement. The result MUST stay a complete instruction that ")
	b.WriteString("keeps the {{body}} (or {{messages}}) placeholder. Output ONLY the revised instruction prompt ")
	b.WriteString("(no code fences, no preamble) followed by the metadata block:\n")
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
		case strings.HasPrefix(trimmed, "```"):
			// Drop Markdown code-fence lines (```, ```plaintext, etc.) that models
			// often wrap the prompt in — they are not part of the template.
			continue
		default:
			bodyLines = append(bodyLines, line)
		}
	}
	result.PromptText = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	return result
}
