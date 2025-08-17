package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/db"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/obsidian"
)

// ObsidianServiceImpl implements ObsidianService
type ObsidianServiceImpl struct {
	store  *db.ObsidianStore
	config *obsidian.ObsidianConfig
	logger *log.Logger
}

// NewObsidianService creates a new Obsidian service
func NewObsidianService(store *db.ObsidianStore, config *obsidian.ObsidianConfig, logger *log.Logger) *ObsidianServiceImpl {
	if config == nil {
		config = obsidian.DefaultObsidianConfig()
	}

	service := &ObsidianServiceImpl{
		store:  store,
		config: config,
		logger: logger,
	}

	// Initialize the database table if it doesn't exist
	if store != nil {
		if err := store.InitializeTable(context.Background()); err != nil {
			// Log error but don't fail service creation
			if logger != nil {
				logger.Printf("Warning: failed to initialize Obsidian table: %v", err)
			}
		}
	}

	return service
}

// IngestEmailToObsidian ingests an email to Obsidian
func (s *ObsidianServiceImpl) IngestEmailToObsidian(ctx context.Context, message *gmail.Message, options obsidian.ObsidianOptions) (*obsidian.ObsidianIngestResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("obsidian store not available")
	}

	// Validate options
	if err := s.validateOptions(options); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	// Check if already forwarded (if enabled)
	if s.config.PreventDuplicates {
		alreadyForwarded, err := s.store.CheckIfAlreadyForwarded(ctx, message.Id, options.AccountEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to check forward status: %w", err)
		}
		if alreadyForwarded {
			return &obsidian.ObsidianIngestResult{
				Success:      false,
				ErrorMessage: "email already ingested to Obsidian",
			}, nil
		}
	}

	// Format email content using the single template from config
	content, err := s.formatEmailForObsidian(message, options)
	if err != nil {
		return nil, fmt.Errorf("failed to format email: %w", err)
	}

	// Generate file path
	filePath, err := s.generateFilePath(message)
	if err != nil {
		return nil, fmt.Errorf("failed to generate file path: %w", err)
	}

	// Log debug information
	if s.logger != nil {
		s.logger.Printf("Obsidian ingestion: creating file at %s (content length: %d)", filePath, len(content))
	}

	// Create file in Obsidian (for now, create local file as placeholder)
	err = s.createObsidianFile(filePath, content)
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("Obsidian ingestion failed for message %s: %v", message.Id, err)
		}
		// Record failure
		s.recordForwardFailure(ctx, message, options, err)
		return &obsidian.ObsidianIngestResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to create file: %v", err),
		}, nil
	}

	if s.logger != nil {
		s.logger.Printf("Obsidian ingestion successful: created file %s", filePath)
	}

	// Record success
	record := &obsidian.ObsidianForwardRecord{
		MessageID:    message.Id,
		AccountEmail: options.AccountEmail,
		ObsidianPath: filePath,
		TemplateUsed: "config_template", // Single template from config
		ForwardDate:  time.Now(),
		Status:       "success",
		FileSize:     int64(len(content)),
		Metadata: map[string]interface{}{
			"subject": message.Subject,
			"from":    s.extractHeader(message, "From"),
			"date":    s.extractHeader(message, "Date"),
			"labels":  message.LabelIds,
		},
	}

	if err := s.store.RecordForward(ctx, record); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to record forward: %v\n", err)
	}

	return &obsidian.ObsidianIngestResult{
		Success:      true,
		FilePath:     filePath,
		FileSize:     int64(len(content)),
		TemplateUsed: "config_template",
		Metadata:     record.Metadata,
	}, nil
}

// validateOptions validates the ingestion options
func (s *ObsidianServiceImpl) validateOptions(options obsidian.ObsidianOptions) error {
	if options.AccountEmail == "" {
		return fmt.Errorf("account email is required")
	}

	return nil
}

// formatEmailForObsidian formats an email using the template from config (file or inline)
func (s *ObsidianServiceImpl) formatEmailForObsidian(message *gmail.Message, options obsidian.ObsidianOptions) (string, error) {
	// Use the same template loading pattern as other services
	fallback := `---
title: "{{subject}}"
date: {{date}}
from: {{from}}
type: email
status: inbox
labels: {{labels}}
message_id: {{message_id}}
---

# {{subject}}

**From:** {{from}}  
**Date:** {{date}}  
**Labels:** {{labels}}

{{comment}}

---

{{body}}

---

*Ingested from Gmail on {{ingest_date}}*`

	content := config.LoadTemplate(s.config.TemplateFile, s.config.Template, fallback)

	// Extract message content
	body := message.PlainText
	if body == "" && message.Snippet != "" {
		body = message.Snippet
	}

	// Truncate if too long
	if len([]rune(body)) > 8000 {
		body = string([]rune(body)[:8000])
	}

	// Extract comment from options
	comment := ""
	if options.CustomMetadata != nil {
		if commentValue, exists := options.CustomMetadata["comment"]; exists {
			if commentStr, ok := commentValue.(string); ok && commentStr != "" {
				comment = fmt.Sprintf("> **Note:** %s\n", commentStr)
			}
		}
	}

	// Prepare variables for substitution
	variables := map[string]string{
		"subject":     message.Subject,
		"from":        s.extractHeader(message, "From"),
		"to":          s.extractHeader(message, "To"),
		"cc":          s.extractHeader(message, "Cc"),
		"date":        s.extractHeader(message, "Date"),
		"body":        body,
		"labels":      strings.Join(message.LabelIds, ", "),
		"message_id":  message.Id,
		"ingest_date": time.Now().Format("2006-01-02 15:04:05"),
		"comment":     comment,
	}

	// Replace variables in template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content, nil
}

// generateFilePath generates the file path for the Obsidian note
func (s *ObsidianServiceImpl) generateFilePath(message *gmail.Message) (string, error) {
	// Always use 00-Inbox as specified
	ingestFolder := s.config.IngestFolder

	// Generate filename
	filename := s.generateFilename(message)

	// Create full path
	fullPath := filepath.Join(s.config.VaultPath, ingestFolder, filename)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return fullPath, nil
}

// generateFilename generates a filename for the email
func (s *ObsidianServiceImpl) generateFilename(message *gmail.Message) string {
	date := time.Now().Format("2006-01-02")
	subject := s.sanitizeFilename(message.Subject)
	from := s.extractDomain(s.extractHeader(message, "From"))

	// Format: YYYY-MM-DD_Subject_FromDomain.md
	return fmt.Sprintf("%s_%s_%s.md", date, subject, from)
}

// sanitizeFilename sanitizes a filename by removing invalid characters
func (s *ObsidianServiceImpl) sanitizeFilename(filename string) string {
	// Remove invalid characters
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	filename = reg.ReplaceAllString(filename, "")

	// Replace spaces with underscores
	filename = strings.ReplaceAll(filename, " ", "_")

	// Limit length
	if len(filename) > 100 {
		filename = filename[:100]
	}

	// Remove trailing underscores
	filename = strings.Trim(filename, "_")

	// Ensure it's not empty
	if filename == "" {
		filename = "untitled"
	}

	return filename
}

// extractDomain extracts the domain from an email address
func (s *ObsidianServiceImpl) extractDomain(email string) string {
	if email == "" {
		return "unknown"
	}

	// Simple domain extraction
	parts := strings.Split(email, "@")
	if len(parts) > 1 {
		return parts[1]
	}

	return "unknown"
}

// extractHeader extracts a header value from a message
func (s *ObsidianServiceImpl) extractHeader(message *gmail.Message, headerName string) string {
	if message.Payload == nil || message.Payload.Headers == nil {
		return ""
	}

	for _, header := range message.Payload.Headers {
		if header.Name == headerName {
			return header.Value
		}
	}
	return ""
}

// createObsidianFile creates a file in the Obsidian vault
func (s *ObsidianServiceImpl) createObsidianFile(filePath, content string) error {
	// For now, create a local file as placeholder
	// TODO: Integrate with MCP Obsidian service

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// recordForwardFailure records a failed forward attempt
func (s *ObsidianServiceImpl) recordForwardFailure(ctx context.Context, message *gmail.Message, options obsidian.ObsidianOptions, err error) {
	if s.store == nil {
		return
	}

	record := &obsidian.ObsidianForwardRecord{
		MessageID:    message.Id,
		AccountEmail: options.AccountEmail,
		ObsidianPath: "",
		TemplateUsed: "config_template", // Single template from config
		ForwardDate:  time.Now(),
		Status:       "failed",
		ErrorMessage: err.Error(),
		FileSize:     0,
		Metadata: map[string]interface{}{
			"subject": message.Subject,
			"from":    s.extractHeader(message, "From"),
			"date":    s.extractHeader(message, "Date"),
		},
	}

	_ = s.store.RecordForward(ctx, record)
}

// GetObsidianTemplates returns the single template from config
func (s *ObsidianServiceImpl) GetObsidianTemplates(ctx context.Context) ([]*obsidian.ObsidianTemplate, error) {
	// Return single template from config
	template := &obsidian.ObsidianTemplate{
		Content: s.config.Template,
	}
	return []*obsidian.ObsidianTemplate{template}, nil
}

// ValidateObsidianConnection validates the connection to Obsidian
func (s *ObsidianServiceImpl) ValidateObsidianConnection(ctx context.Context) error {
	if s.config.VaultPath == "" {
		return fmt.Errorf("vault path not configured")
	}

	// Check if vault directory exists
	if _, err := os.Stat(s.config.VaultPath); os.IsNotExist(err) {
		return fmt.Errorf("vault directory does not exist: %s", s.config.VaultPath)
	}

	// Check if 00-Inbox directory exists or can be created
	inboxPath := filepath.Join(s.config.VaultPath, s.config.IngestFolder)
	if err := os.MkdirAll(inboxPath, 0755); err != nil {
		return fmt.Errorf("failed to create inbox directory: %w", err)
	}

	return nil
}

// GetObsidianVaultPath returns the configured vault path
func (s *ObsidianServiceImpl) GetObsidianVaultPath() string {
	return s.config.VaultPath
}

// GetConfig returns the current configuration
func (s *ObsidianServiceImpl) GetConfig() *obsidian.ObsidianConfig {
	return s.config
}

// UpdateConfig updates the configuration
func (s *ObsidianServiceImpl) UpdateConfig(config *obsidian.ObsidianConfig) {
	if config != nil {
		s.config = config
	}
}

// IngestBulkEmailsToObsidian ingests multiple emails to Obsidian with progress tracking
func (s *ObsidianServiceImpl) IngestBulkEmailsToObsidian(ctx context.Context, messages []*gmail.Message, accountEmail string, onProgress func(int, int, error)) (*obsidian.BulkObsidianResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	startTime := time.Now()
	var successfulPaths []string
	var failedMessages []string
	var totalSize int64

	for i, message := range messages {
		// Call progress callback
		if onProgress != nil {
			onProgress(i, len(messages), nil)
		}

		// Create options for this message
		options := obsidian.ObsidianOptions{
			AccountEmail: accountEmail,
			CustomMetadata: map[string]interface{}{
				"bulk_operation": true,
				"batch_index":    i + 1,
				"batch_total":    len(messages),
			},
		}

		// Ingest this message
		result, err := s.IngestEmailToObsidian(ctx, message, options)
		if err != nil {
			failedMessages = append(failedMessages, message.Id)
			if onProgress != nil {
				onProgress(i+1, len(messages), err)
			}
			continue
		}

		if result != nil && result.Success {
			successfulPaths = append(successfulPaths, result.FilePath)
			totalSize += result.FileSize
		} else {
			failedMessages = append(failedMessages, message.Id)
		}
	}

	// Final progress callback
	if onProgress != nil {
		onProgress(len(messages), len(messages), nil)
	}

	return &obsidian.BulkObsidianResult{
		TotalMessages:   len(messages),
		SuccessfulCount: len(successfulPaths),
		FailedCount:     len(failedMessages),
		SuccessfulPaths: successfulPaths,
		FailedMessages:  failedMessages,
		TotalSize:       totalSize,
		Duration:        time.Since(startTime),
		CompletedAt:     time.Now(),
	}, nil
}
