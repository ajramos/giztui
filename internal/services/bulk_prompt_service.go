package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/gmail"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BulkPromptServiceImpl implements bulk prompt operations
type BulkPromptServiceImpl struct {
	emailService  EmailService
	aiService     AIService
	cacheService  CacheService
	repository    MessageRepository
	promptService PromptService
}

// NewBulkPromptService creates a new bulk prompt service
func NewBulkPromptService(
	emailService EmailService,
	aiService AIService,
	cacheService CacheService,
	repository MessageRepository,
	promptService PromptService,
) *BulkPromptServiceImpl {
	return &BulkPromptServiceImpl{
		emailService:  emailService,
		aiService:     aiService,
		cacheService:  cacheService,
		repository:    repository,
		promptService: promptService,
	}
}

// SetPromptService sets the prompt service reference
func (s *BulkPromptServiceImpl) SetPromptService(promptService PromptService) {
	s.promptService = promptService
}

// ApplyBulkPrompt applies a prompt to multiple messages and returns a consolidated result
func (s *BulkPromptServiceImpl) ApplyBulkPrompt(
	ctx context.Context,
	accountEmail string,
	messageIDs []string,
	promptID int,
	variables map[string]string,
) (*BulkPromptResult, error) {
	if len(messageIDs) == 0 {
		return nil, fmt.Errorf("no message IDs provided")
	}

	startTime := time.Now()

	// Check cache first via prompt service
	if s.promptService != nil {
		if cachedResult, err := s.promptService.GetCachedBulkResult(ctx, accountEmail, messageIDs, promptID); err == nil && cachedResult != nil {
			return cachedResult, nil
		}
	}

	// Load all message contents
	messageContents := make([]string, 0, len(messageIDs))
	successfulIDs := make([]string, 0, len(messageIDs))

	for _, messageID := range messageIDs {
		message, err := s.repository.GetMessage(ctx, messageID)
		if err != nil {
			// Log error but continue with other messages
			fmt.Printf("bulk prompt service: failed to get message %s: %v\n", messageID, err)
			continue
		}

		// Extract content (you might need to adjust this based on your message structure)
		content := s.extractMessageContent(message)

		if content != "" {
			messageContents = append(messageContents, content)
			successfulIDs = append(successfulIDs, messageID)
		} else {
			// Content extraction failed for this message
		}
	}

	if len(messageContents) == 0 {
		return nil, fmt.Errorf("failed to load content from any of the %d messages", len(messageIDs))
	}

	// Combine all message contents into a single context
	combinedContent := s.combineMessageContents(messageContents, successfulIDs)

	// Get the actual prompt template from the database
	promptTemplate, err := s.promptService.GetPrompt(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt template: %w", err)
	}

	// Build the final prompt with the actual template and content
	finalPrompt := s.buildBulkPrompt(promptTemplate.PromptText, combinedContent, variables)

	// Save the final prompt to a file for debugging
	s.savePromptToFile(promptID, promptTemplate.Name, successfulIDs, promptTemplate.PromptText, finalPrompt, combinedContent)

	result, err := s.aiService.ApplyCustomPrompt(ctx, combinedContent, finalPrompt, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to apply bulk prompt: %w", err)
	}

	// Cache the result via prompt service
	if s.promptService != nil {
		_ = s.promptService.SaveBulkResult(ctx, accountEmail, successfulIDs, promptID, result)
		// Increment usage count for the prompt
		_ = s.promptService.IncrementUsage(ctx, promptID)
	}

	return &BulkPromptResult{
		PromptID:     promptID,
		MessageCount: len(successfulIDs),
		Summary:      result,
		MessageIDs:   successfulIDs,
		Duration:     time.Since(startTime),
		FromCache:    false,
		CreatedAt:    time.Now(),
	}, nil
}

// ApplyBulkPromptStream applies a prompt to multiple messages with streaming support
func (s *BulkPromptServiceImpl) ApplyBulkPromptStream(ctx context.Context, accountEmail string, messageIDs []string, promptID int, variables map[string]string, onToken func(string)) (*BulkPromptResult, error) {
	startTime := time.Now()

	// Check cache first via prompt service
	if s.promptService != nil {
		if cachedResult, err := s.promptService.GetCachedBulkResult(ctx, accountEmail, messageIDs, promptID); err == nil && cachedResult != nil {
			return cachedResult, nil
		}
	}

	// Get the prompt template
	promptTemplate, err := s.promptService.GetPrompt(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt template: %w", err)
	}

	// Load message contents
	messageContents := make([]string, 0, len(messageIDs))
	successfulIDs := make([]string, 0, len(messageIDs))

	for _, messageID := range messageIDs {
		message, err := s.repository.GetMessage(ctx, messageID)
		if err != nil {
			continue // Skip failed messages, don't fail the entire operation
		}

		content := s.extractMessageContent(message)
		if content != "" {
			messageContents = append(messageContents, content)
			successfulIDs = append(successfulIDs, messageID)
		}
	}

	if len(messageContents) == 0 {
		return nil, fmt.Errorf("no valid messages found")
	}

	// Combine message contents for LLM
	combinedContent := s.combineMessageContents(messageContents, successfulIDs)

	// Build the final prompt using the actual template text
	finalPrompt := s.buildBulkPrompt(promptTemplate.PromptText, combinedContent, variables)

	// Save the final prompt to a file for debugging
	s.savePromptToFile(promptID, promptTemplate.Name, successfulIDs, promptTemplate.PromptText, finalPrompt, combinedContent)

	// Use streaming AI service
	result, err := s.aiService.ApplyCustomPromptStream(ctx, combinedContent, finalPrompt, variables, onToken)
	if err != nil {
		return nil, fmt.Errorf("failed to apply bulk prompt with streaming: %w", err)
	}

	// Cache the result via prompt service
	if s.promptService != nil {
		_ = s.promptService.SaveBulkResult(ctx, accountEmail, successfulIDs, promptID, result)
		// Increment usage count for the prompt
		_ = s.promptService.IncrementUsage(ctx, promptID)
	}

	return &BulkPromptResult{
		PromptID:     promptID,
		MessageCount: len(successfulIDs),
		Summary:      result,
		MessageIDs:   successfulIDs,
		Duration:     time.Since(startTime),
		FromCache:    false,
		CreatedAt:    time.Now(),
	}, nil
}

// extractMessageContent extracts the main content from a Gmail message
func (s *BulkPromptServiceImpl) extractMessageContent(message *gmail.Message) string {
	if message == nil {
		return ""
	}

	// First try to get from payload parts (most reliable)
	if message.Payload != nil && len(message.Payload.Parts) > 0 {
		for _, part := range message.Payload.Parts {
			if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
				// Decode base64 data
				if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
					return string(decoded)
				}
			}
		}
	}

	// If no parts or no text/plain, try to get from payload body directly
	if message.Payload != nil && message.Payload.Body != nil && message.Payload.Body.Data != "" {
		if decoded, err := base64.URLEncoding.DecodeString(message.Payload.Body.Data); err == nil {
			return string(decoded)
		}
	}

	// Fallback to snippet if available
	if message.Snippet != "" {
		return message.Snippet
	}

	// Last resort: return a placeholder
	return "[No readable content found]"
}

// combineMessageContents combines multiple message contents into a single context
func (s *BulkPromptServiceImpl) combineMessageContents(contents []string, messageIDs []string) string {
	if len(contents) == 0 {
		return ""
	}

	var combined strings.Builder
	combined.WriteString("---START EMAILS---\n")

	for i, content := range contents {
		combined.WriteString(fmt.Sprintf("---START EMAIL %d---\n", i+1))

		// Clean and process the content heuristically for better LLM consumption
		cleanContent := s.cleanEmailContent(content)
		combined.WriteString(cleanContent)

		combined.WriteString(fmt.Sprintf("\n---END EMAIL %d---\n", i+1))
	}

	combined.WriteString("---END OF EMAILS---\n")

	return combined.String()
}

// cleanEmailContent processes email content heuristically to make it more digestible for LLM
func (s *BulkPromptServiceImpl) cleanEmailContent(content string) string {
	if content == "" {
		return "[Empty email]"
	}

	// Remove excessive URLs and tracking links
	lines := strings.Split(content, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip pure URL lines (tracking/unsubscribe links)
		if strings.HasPrefix(line, "https://") && len(line) > 50 {
			continue
		}

		// Skip common email footers
		if strings.Contains(strings.ToLower(line), "unsubscribe") ||
			strings.Contains(strings.ToLower(line), "privacy policy") ||
			strings.Contains(strings.ToLower(line), "powered by") ||
			strings.Contains(strings.ToLower(line), "support@") {
			continue
		}

		// Clean up encoded characters (basic cleanup)
		line = strings.ReplaceAll(line, "-2F", "/")
		line = strings.ReplaceAll(line, "-2B", "+")
		line = strings.ReplaceAll(line, "-3D", "=")

		// Limit line length for readability
		if len(line) > 200 {
			line = line[:200] + "..."
		}

		cleanLines = append(cleanLines, line)

		// Limit total lines to prevent overwhelming the LLM
		if len(cleanLines) >= 20 {
			cleanLines = append(cleanLines, "[Content truncated for brevity...]")
			break
		}
	}

	if len(cleanLines) == 0 {
		return "[No meaningful content found]"
	}

	return strings.Join(cleanLines, "\n")
}

// buildBulkPrompt builds a prompt specifically for bulk analysis
func (s *BulkPromptServiceImpl) buildBulkPrompt(promptText string, combinedContent string, variables map[string]string) string {
	// Use the actual prompt template from the database
	prompt := promptText

	// Always replace {{messages}} with the combined content
	prompt = strings.ReplaceAll(prompt, "{{messages}}", combinedContent)

	// Also replace {{body}} for backward compatibility (in case some prompts still use it)
	prompt = strings.ReplaceAll(prompt, "{{body}}", combinedContent)

	// Replace any other variables in the prompt
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	return prompt
}

// savePromptToFile saves the prompt and content to a file for debugging
func (s *BulkPromptServiceImpl) savePromptToFile(promptID int, promptName string, messageIDs []string, originalPrompt string, finalPrompt string, combinedContent string) {
	// Use the configuration directory for saved files
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return // Silently fail if we can't get home directory
	}

	savedDir := filepath.Join(homeDir, ".config", "gmail-tui", "saved")
	if err := os.MkdirAll(savedDir, 0755); err != nil {
		return // Silently fail if we can't create directory
	}

	// Create filename with timestamp, prompt name, and message IDs
	timestamp := time.Now().Format("20060102_150405")
	messageIDsStr := strings.Join(messageIDs, "_")
	filename := fmt.Sprintf("%s_%s_%s.txt", timestamp, strings.ReplaceAll(promptName, " ", "_"), messageIDsStr)

	// Ensure filename is safe for filesystem
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	filepath := filepath.Join(savedDir, filename)

	// Create file content
	content := fmt.Sprintf("=== BULK PROMPT DEBUG INFO ===\n\n")
	content += fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
	content += fmt.Sprintf("Prompt ID: %d\n", promptID)
	content += fmt.Sprintf("Prompt Name: %s\n", promptName)
	content += fmt.Sprintf("Message IDs: %s\n", strings.Join(messageIDs, ", "))
	content += fmt.Sprintf("Message Count: %d\n\n", len(messageIDs))

	content += "=== ORIGINAL PROMPT FROM DATABASE ===\n\n"
	content += originalPrompt
	content += "\n\n"

	content += "=== FINAL PROMPT SENT TO LLM (AFTER VARIABLE SUBSTITUTION) ===\n\n"
	content += finalPrompt
	content += "\n\n"

	content += "=== RAW EMAIL CONTENT (FOR COMPARISON) ===\n\n"
	content += "Note: This shows the raw content before heuristic cleaning.\n"
	content += "The cleaned version is what gets inserted into {{messages}} in the prompt above.\n\n"

	// Show a sample of the raw content for comparison
	if len(messageIDs) > 0 {
		content += fmt.Sprintf("Sample raw content from emails:\n")
		if len(combinedContent) > 1000 {
			content += combinedContent[:1000] + "\n[... truncated for brevity - see cleaned version in final prompt above ...]\n"
		} else {
			content += combinedContent + "\n"
		}
	}

	content += "\n=== END DEBUG INFO ===\n"

	// Write to file
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		// Silently fail if we can't write file
		return
	}
}

// GetCachedBulkResult retrieves a cached bulk prompt result
func (s *BulkPromptServiceImpl) GetCachedBulkResult(ctx context.Context, accountEmail string, messageIDs []string, promptID int) (*BulkPromptResult, error) {
	if s.cacheService == nil {
		return nil, fmt.Errorf("cache service not available")
	}

	// Create a cache key based on sorted message IDs for consistency
	sortedIDs := make([]string, len(messageIDs))
	copy(sortedIDs, messageIDs)
	sort.Strings(sortedIDs)
	cacheKey := fmt.Sprintf("bulk_%d_%s", promptID, strings.Join(sortedIDs, "_"))

	// Try to get from cache
	if cached, exists, err := s.cacheService.GetSummary(ctx, accountEmail, cacheKey); err == nil && exists {
		return &BulkPromptResult{
			PromptID:     promptID,
			MessageCount: len(messageIDs),
			Summary:      cached,
			MessageIDs:   messageIDs,
			Duration:     0, // Since it's from cache
			FromCache:    true,
			AccountEmail: accountEmail,
			CreatedAt:    time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("no cached result found")
}
