package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/db"
	"gopkg.in/yaml.v3"
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

// ApplyPromptStream applies a prompt with streaming support
func (s *PromptServiceImpl) ApplyPromptStream(ctx context.Context, messageContent string, promptID int, variables map[string]string, onToken func(string)) (*PromptResult, error) {
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

	// Apply the prompt using the AI service with streaming
	result, err := s.aiService.ApplyCustomPromptStream(ctx, messageContent, prompt, variables, onToken)
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

// GetUsageStats retrieves usage statistics for all prompts
func (s *PromptServiceImpl) GetUsageStats(ctx context.Context) (*UsageStats, error) {
	if s.store == nil {
		return nil, fmt.Errorf("cache store not available")
	}

	// Get all prompts with usage data
	prompts, err := s.store.ListPromptTemplates(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	// Calculate statistics
	var totalUsage int
	var uniquePrompts int
	var lastUsed time.Time
	var topPrompts []PromptUsageStat
	var favoritePrompts []PromptUsageStat

	for _, prompt := range prompts {
		if prompt.UsageCount > 0 {
			uniquePrompts++
			totalUsage += prompt.UsageCount

			// Track latest usage (approximate using created_at for now)
			createdTime := time.Unix(prompt.CreatedAt, 0)
			if createdTime.After(lastUsed) {
				lastUsed = createdTime
			}

			// Create usage stat
			stat := PromptUsageStat{
				ID:         prompt.ID,
				Name:       prompt.Name,
				Category:   prompt.Category,
				UsageCount: prompt.UsageCount,
				IsFavorite: prompt.IsFavorite,
				LastUsed:   createdTime.Format("2006-01-02 15:04"),
			}

			// Add to top prompts list
			topPrompts = append(topPrompts, stat)

			// Add to favorites if applicable
			if prompt.IsFavorite {
				favoritePrompts = append(favoritePrompts, stat)
			}
		}
	}

	// Sort top prompts by usage count (descending)
	sort.Slice(topPrompts, func(i, j int) bool {
		return topPrompts[i].UsageCount > topPrompts[j].UsageCount
	})

	// Limit to top 10
	if len(topPrompts) > 10 {
		topPrompts = topPrompts[:10]
	}

	return &UsageStats{
		TopPrompts:      topPrompts,
		TotalUsage:      totalUsage,
		UniquePrompts:   uniquePrompts,
		LastUsed:        lastUsed,
		FavoritePrompts: favoritePrompts,
	}, nil
}

// PromptFrontMatter represents the YAML front matter in a prompt markdown file
type PromptFrontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Category    string `yaml:"category"`
}

// CreatePrompt creates a new prompt template
func (s *PromptServiceImpl) CreatePrompt(ctx context.Context, name, description, promptText, category string) (int, error) {
	if s.store == nil {
		return 0, fmt.Errorf("store not available")
	}
	return s.store.CreatePromptTemplate(ctx, name, description, promptText, category)
}

// UpdatePrompt updates an existing prompt template
func (s *PromptServiceImpl) UpdatePrompt(ctx context.Context, id int, name, description, promptText, category string) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}
	return s.store.UpdatePromptTemplate(ctx, id, name, description, promptText, category)
}

// DeletePrompt deletes a prompt template
func (s *PromptServiceImpl) DeletePrompt(ctx context.Context, id int) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}
	return s.store.DeletePromptTemplate(ctx, id)
}

// FindPromptByName finds a prompt template by name
func (s *PromptServiceImpl) FindPromptByName(ctx context.Context, name string) (*PromptTemplate, error) {
	if s.store == nil {
		return nil, fmt.Errorf("store not available")
	}
	return s.store.FindPromptByName(ctx, name)
}

// CreateFromFile creates a prompt template from a markdown file with front matter
func (s *PromptServiceImpl) CreateFromFile(ctx context.Context, filePath string) (int, error) {
	if s.store == nil {
		return 0, fmt.Errorf("store not available")
	}

	// Expand tilde in path
	if strings.HasPrefix(filePath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0, fmt.Errorf("cannot get home directory: %w", err)
		}
		if filePath == "~" {
			filePath = home
		} else {
			filePath = filepath.Join(home, filePath[2:])
		}
	}

	// Validate path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		return 0, fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse front matter and content
	frontMatter, promptText, err := s.parseFrontMatter(content)
	if err != nil {
		return 0, fmt.Errorf("failed to parse front matter in %s: %w", filePath, err)
	}

	// Validate required fields
	if strings.TrimSpace(frontMatter.Name) == "" {
		return 0, fmt.Errorf("prompt name is required in front matter")
	}
	if strings.TrimSpace(frontMatter.Category) == "" {
		return 0, fmt.Errorf("prompt category is required in front matter")
	}
	if strings.TrimSpace(promptText) == "" {
		return 0, fmt.Errorf("prompt content cannot be empty")
	}

	// Check if prompt with same name already exists
	existing, err := s.store.FindPromptByName(ctx, frontMatter.Name)
	if err == nil && existing != nil {
		// Prompt exists, update it
		return existing.ID, s.store.UpdatePromptTemplate(ctx, existing.ID, frontMatter.Name, frontMatter.Description, promptText, frontMatter.Category)
	}

	// Create new prompt
	return s.store.CreatePromptTemplate(ctx, frontMatter.Name, frontMatter.Description, promptText, frontMatter.Category)
}

// ExportToFile exports a prompt template to a markdown file with front matter
func (s *PromptServiceImpl) ExportToFile(ctx context.Context, id int, filePath string) error {
	if s.store == nil {
		return fmt.Errorf("store not available")
	}

	// Get the prompt template
	prompt, err := s.store.GetPromptTemplate(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get prompt template: %w", err)
	}

	// Expand tilde in path
	if strings.HasPrefix(filePath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get home directory: %w", err)
		}
		if filePath == "~" {
			filePath = home
		} else {
			filePath = filepath.Join(home, filePath[2:])
		}
	}

	// Create front matter
	frontMatter := PromptFrontMatter{
		Name:        prompt.Name,
		Description: prompt.Description,
		Category:    prompt.Category,
	}

	// Generate markdown content
	content, err := s.generateMarkdownContent(frontMatter, prompt.PromptText)
	if err != nil {
		return fmt.Errorf("failed to generate markdown content: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(filePath, content, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// parseFrontMatter parses YAML front matter from markdown content
func (s *PromptServiceImpl) parseFrontMatter(content []byte) (PromptFrontMatter, string, error) {
	var frontMatter PromptFrontMatter

	// Convert to string for parsing
	text := string(content)

	// Check if file starts with front matter
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return frontMatter, "", fmt.Errorf("file must start with YAML front matter (---)")
	}

	// Find the end of front matter
	lines := strings.Split(text, "\n")
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return frontMatter, "", fmt.Errorf("front matter not properly closed with ---")
	}

	// Extract front matter YAML
	yamlContent := strings.Join(lines[1:endIdx], "\n")
	if err := yaml.Unmarshal([]byte(yamlContent), &frontMatter); err != nil {
		return frontMatter, "", fmt.Errorf("failed to parse YAML front matter: %w", err)
	}

	// Extract prompt content (everything after front matter)
	promptLines := lines[endIdx+1:]
	promptText := strings.Join(promptLines, "\n")
	promptText = strings.TrimSpace(promptText)

	return frontMatter, promptText, nil
}

// generateMarkdownContent generates markdown content with front matter
func (s *PromptServiceImpl) generateMarkdownContent(frontMatter PromptFrontMatter, promptText string) ([]byte, error) {
	var buf bytes.Buffer

	// Write front matter
	buf.WriteString("---\n")
	yamlData, err := yaml.Marshal(frontMatter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal front matter: %w", err)
	}
	buf.Write(yamlData)
	buf.WriteString("---\n\n")

	// Write prompt content
	buf.WriteString(promptText)
	buf.WriteString("\n")

	return buf.Bytes(), nil
}
