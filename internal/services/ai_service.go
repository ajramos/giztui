package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/llm"
)

// AIServiceImpl implements AIService
type AIServiceImpl struct {
	provider     llm.Provider
	cacheService CacheService
	config       *config.Config
}

// NewAIService creates a new AI service
func NewAIService(provider llm.Provider, cacheService CacheService, config *config.Config) *AIServiceImpl {
	return &AIServiceImpl{
		provider:     provider,
		cacheService: cacheService,
		config:       config,
	}
}

func (s *AIServiceImpl) GenerateSummary(ctx context.Context, content string, options SummaryOptions) (*SummaryResult, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("AI provider not available")
	}

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	start := time.Now()

	// Check cache first if enabled and not forcing regeneration
	if options.UseCache && !options.ForceRegenerate && s.cacheService != nil {
		if cached, found, err := s.cacheService.GetSummary(ctx, options.AccountEmail, options.MessageID); err == nil && found {
			return &SummaryResult{
				Summary:   cached,
				FromCache: true,
				Duration:  time.Since(start),
			}, nil
		}
	}

	// Truncate content if too long
	maxLength := 8000
	if options.MaxLength > 0 {
		maxLength = options.MaxLength
	}

	if len([]rune(content)) > maxLength {
		content = string([]rune(content)[:maxLength])
	}

	// Build prompt
	prompt := s.config.LLM.GetSummarizePrompt()
	if prompt == "" {
		prompt = "Briefly summarize the following email. Keep it concise and factual.\n\n{{body}}"
	}

	prompt = strings.ReplaceAll(prompt, "{{body}}", content)

	// Generate summary
	summary, err := s.provider.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Cache the result if caching is enabled
	if options.UseCache && s.cacheService != nil {
		if err := s.cacheService.SaveSummary(ctx, options.AccountEmail, options.MessageID, summary); err != nil {
			// Save to cache failed, but don't fail the entire operation
			// Note: Cache failures are logged within the cache service if needed
		}
	}

	return &SummaryResult{
		Summary:   summary,
		FromCache: false,
		Language:  options.Language,
		Duration:  time.Since(start),
	}, nil
}

// GenerateSummaryStream generates a summary with streaming support
func (s *AIServiceImpl) GenerateSummaryStream(ctx context.Context, content string, options SummaryOptions, onToken func(string)) (*SummaryResult, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("AI provider not available")
	}

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	start := time.Now()

	// Check cache first if enabled and not forcing regeneration
	if options.UseCache && !options.ForceRegenerate && s.cacheService != nil {
		if cached, found, err := s.cacheService.GetSummary(ctx, options.AccountEmail, options.MessageID); err == nil && found {
			return &SummaryResult{
				Summary:   cached,
				FromCache: true,
				Duration:  time.Since(start),
			}, nil
		}
	}

	// Truncate content if too long
	maxLength := 8000
	if options.MaxLength > 0 {
		maxLength = options.MaxLength
	}

	if len([]rune(content)) > maxLength {
		content = string([]rune(content)[:maxLength])
	}

	// Build prompt
	prompt := s.config.LLM.GetSummarizePrompt()
	if prompt == "" {
		prompt = "Briefly summarize the following email. Keep it concise and factual.\n\n{{body}}"
	}

	prompt = strings.ReplaceAll(prompt, "{{body}}", content)

	// Check if provider supports streaming
	if streamer, ok := s.provider.(interface {
		GenerateStream(context.Context, string, func(string)) error
	}); ok {
		var result strings.Builder

		err := streamer.GenerateStream(ctx, prompt, func(token string) {
			result.WriteString(token)
			if onToken != nil {
				onToken(token)
			}
		})

		if err != nil {
			return nil, fmt.Errorf("failed to generate summary: %w", err)
		}

		summary := result.String()

		// Cache the result if caching is enabled
		if options.UseCache && s.cacheService != nil {
			if err := s.cacheService.SaveSummary(ctx, options.AccountEmail, options.MessageID, summary); err != nil {
				// Save to cache failed, but don't fail the entire operation
				// Note: Cache failures are logged within the cache service if needed
			}
		}

		return &SummaryResult{
			Summary:   summary,
			FromCache: false,
			Language:  options.Language,
			Duration:  time.Since(start),
		}, nil
	}

	// Fallback to non-streaming if streaming not supported
	return s.GenerateSummary(ctx, content, options)
}

func (s *AIServiceImpl) GenerateReply(ctx context.Context, content string, options ReplyOptions) (string, error) {
	if s.provider == nil {
		return "", fmt.Errorf("AI provider not available")
	}

	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	// Build prompt
	prompt := s.config.LLM.GetReplyPrompt()
	if prompt == "" {
		prompt = "Write a professional and friendly reply to the following email. Keep the same language as the input.\n\n{{body}}"
	}

	prompt = strings.ReplaceAll(prompt, "{{body}}", content)

	// Add tone and length modifiers if specified
	if options.Tone != "" {
		prompt = fmt.Sprintf("Write a %s reply to the following email.\n\n%s", options.Tone, content)
	}

	reply, err := s.provider.Generate(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate reply: %w", err)
	}

	return reply, nil
}

func (s *AIServiceImpl) SuggestLabels(ctx context.Context, content string, availableLabels []string) ([]string, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("AI provider not available")
	}

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	if len(availableLabels) == 0 {
		return nil, fmt.Errorf("no available labels provided")
	}

	// Build prompt
	prompt := s.config.LLM.GetLabelPrompt()
	if prompt == "" {
		prompt = "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: {{labels}}\n\nEmail:\n{{body}}"
	}

	labelsStr := strings.Join(availableLabels, ", ")
	prompt = strings.ReplaceAll(prompt, "{{labels}}", labelsStr)
	prompt = strings.ReplaceAll(prompt, "{{body}}", content)

	response, err := s.provider.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate label suggestions: %w", err)
	}

	// Parse JSON response (simplified - in practice you'd use proper JSON parsing)
	response = strings.TrimSpace(response)
	if !strings.HasPrefix(response, "[") || !strings.HasSuffix(response, "]") {
		return nil, fmt.Errorf("invalid response format from AI provider")
	}

	// Simple parsing - remove brackets and split by comma
	response = strings.Trim(response, "[]")
	suggestions := strings.Split(response, ",")

	var validSuggestions []string
	for _, suggestion := range suggestions {
		suggestion = strings.Trim(strings.TrimSpace(suggestion), "\"")
		// Verify suggestion is in available labels
		for _, available := range availableLabels {
			if strings.EqualFold(suggestion, available) {
				validSuggestions = append(validSuggestions, available)
				break
			}
		}
	}

	return validSuggestions, nil
}

func (s *AIServiceImpl) FormatContent(ctx context.Context, content string, options FormatOptions) (string, error) {
	if s.provider == nil {
		return content, nil // Return original content if no AI provider
	}

	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	if !options.TouchUpMode {
		return content, nil // No formatting requested
	}

	// Build prompt for content formatting
	prompt := s.config.LLM.GetTouchUpPrompt()
	if prompt == "" {
		prompt = "You are a formatting assistant. Do NOT paraphrase, translate, or summarize. Your goals: (1) Adjust whitespace and line breaks to improve terminal readability within a wrap width of {{wrap_width}}; (2) Remove strictly duplicated sections or paragraphs. Output only the adjusted text.\n\n{{body}}"
	}

	wrapWidth := "80"
	if options.WrapWidth > 0 {
		wrapWidth = fmt.Sprintf("%d", options.WrapWidth)
	}

	prompt = strings.ReplaceAll(prompt, "{{wrap_width}}", wrapWidth)
	prompt = strings.ReplaceAll(prompt, "{{body}}", content)

	formatted, err := s.provider.Generate(prompt)
	if err != nil {
		// Return original content if formatting fails
		return content, nil
	}

	return formatted, nil
}

func (s *AIServiceImpl) ApplyCustomPrompt(ctx context.Context, content string, prompt string, variables map[string]string) (string, error) {
	if s.provider == nil {
		return "", fmt.Errorf("AI provider not available")
	}

	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	// Generate response using the custom prompt
	result, err := s.provider.Generate(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to apply custom prompt: %w", err)
	}

	return result, nil
}

// ApplyCustomPromptStream applies a custom prompt with streaming support
func (s *AIServiceImpl) ApplyCustomPromptStream(ctx context.Context, content string, prompt string, variables map[string]string, onToken func(string)) (string, error) {
	if s.provider == nil {
		return "", fmt.Errorf("AI provider not available")
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("content cannot be empty")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	// Check if provider supports streaming
	if streamer, ok := s.provider.(interface {
		GenerateStream(context.Context, string, func(string)) error
	}); ok {
		var result strings.Builder

		err := streamer.GenerateStream(ctx, prompt, func(token string) {
			result.WriteString(token)
			if onToken != nil {
				onToken(token)
			}
		})

		if err != nil {
			return "", fmt.Errorf("failed to apply custom prompt with streaming: %w", err)
		}

		return result.String(), nil
	}

	// Fallback to non-streaming if not supported
	return s.ApplyCustomPrompt(ctx, content, prompt, variables)
}
