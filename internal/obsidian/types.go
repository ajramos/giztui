package obsidian

import (
	"time"
)

// ObsidianTemplate represents the single configurable template for email ingestion
type ObsidianTemplate struct {
	Content string `json:"content"` // The template content with variables
}

// ObsidianOptions contains configuration for email ingestion
type ObsidianOptions struct {
	AccountEmail   string                 `json:"account_email"`
	CustomMetadata map[string]interface{} `json:"custom_metadata"`
}

// ObsidianForwardRecord represents a record of an email forwarded to Obsidian
type ObsidianForwardRecord struct {
	ID           int                    `json:"id"`
	MessageID    string                 `json:"message_id"`
	AccountEmail string                 `json:"account_email"`
	ObsidianPath string                 `json:"obsidian_path"`
	TemplateUsed string                 `json:"template_used"`
	ForwardDate  time.Time              `json:"forward_date"`
	Status       string                 `json:"status"` // success, failed, pending
	ErrorMessage string                 `json:"error_message"`
	FileSize     int64                  `json:"file_size"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ObsidianConfig contains the configuration for the Obsidian integration
type ObsidianConfig struct {
	Enabled            bool   `json:"enabled"`
	VaultPath          string `json:"vault_path"`
	IngestFolder       string `json:"ingest_folder"`
	FilenameFormat     string `json:"filename_format"`
	HistoryEnabled     bool   `json:"history_enabled"`
	PreventDuplicates  bool   `json:"prevent_duplicates"`
	MaxFileSize        int64  `json:"max_file_size"`
	IncludeAttachments bool   `json:"include_attachments"`

	// Template configuration (file path takes precedence over inline)
	TemplateFile string `json:"template_file,omitempty"` // Path to template file (relative to config dir or absolute)
	Template     string `json:"template"`                // Inline template (fallback)
}

// DefaultObsidianConfig returns the default configuration
func DefaultObsidianConfig() *ObsidianConfig {
	return &ObsidianConfig{
		Enabled:            true,
		IngestFolder:       "00-Inbox",
		FilenameFormat:     "{{date}}_{{subject_slug}}_{{from_domain}}",
		HistoryEnabled:     true,
		PreventDuplicates:  true,
		MaxFileSize:        1048576, // 1MB
		IncludeAttachments: true,    // Always include attachments by default
		TemplateFile:       "templates/obsidian/email.md",
		Template: `---
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

*Ingested from Gmail on {{ingest_date}}*`,
	}
}

// ObsidianIngestResult represents the result of an email ingestion
type ObsidianIngestResult struct {
	Success      bool                   `json:"success"`
	FilePath     string                 `json:"file_path"`
	FileSize     int64                  `json:"file_size"`
	TemplateUsed string                 `json:"template_used"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// BulkObsidianResult represents the result of a bulk email ingestion
type BulkObsidianResult struct {
	TotalMessages   int           `json:"total_messages"`
	SuccessfulCount int           `json:"successful_count"`
	FailedCount     int           `json:"failed_count"`
	SuccessfulPaths []string      `json:"successful_paths"`
	FailedMessages  []string      `json:"failed_messages"`
	TotalSize       int64         `json:"total_size"`
	Duration        time.Duration `json:"duration"`
	CompletedAt     time.Time     `json:"completed_at"`
}
