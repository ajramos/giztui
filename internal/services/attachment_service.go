package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/gmail"
	googleGmail "google.golang.org/api/gmail/v1"
)

// AttachmentServiceImpl implements AttachmentService
type AttachmentServiceImpl struct {
	gmailClient *gmail.Client
	config      *config.Config
}

// NewAttachmentService creates a new attachment service
func NewAttachmentService(gmailClient *gmail.Client, config *config.Config) *AttachmentServiceImpl {
	return &AttachmentServiceImpl{
		gmailClient: gmailClient,
		config:      config,
	}
}

// GetMessageAttachments extracts all attachments from a message
func (s *AttachmentServiceImpl) GetMessageAttachments(ctx context.Context, messageID string) ([]AttachmentInfo, error) {
	if messageID == "" {
		return nil, fmt.Errorf("messageID cannot be empty")
	}

	// Get message content
	message, err := s.gmailClient.GetMessage(messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Extract attachments from MIME structure
	attachments := s.extractAttachmentsFromMessage(message)

	return attachments, nil
}

// DownloadAttachment downloads an attachment to the specified path
func (s *AttachmentServiceImpl) DownloadAttachment(ctx context.Context, messageID, attachmentID, savePath string) (string, error) {
	return s.DownloadAttachmentWithFilename(ctx, messageID, attachmentID, savePath, "")
}

// DownloadAttachmentWithFilename downloads an attachment with a suggested filename
func (s *AttachmentServiceImpl) DownloadAttachmentWithFilename(ctx context.Context, messageID, attachmentID, savePath, suggestedFilename string) (string, error) {
	if messageID == "" || attachmentID == "" {
		return "", fmt.Errorf("messageID and attachmentID cannot be empty")
	}

	// Download attachment data
	data, extractedFilename, err := s.gmailClient.GetAttachment(messageID, attachmentID)
	if err != nil {
		return "", fmt.Errorf("failed to download attachment: %w", err)
	}

	// Use suggested filename if available, otherwise use extracted filename
	filename := extractedFilename
	if suggestedFilename != "" {
		filename = suggestedFilename
	}

	// Determine save path
	var finalPath string
	if savePath != "" {
		finalPath = savePath
	} else {
		// Use default download directory
		downloadDir := s.GetDefaultDownloadPath()
		finalPath = filepath.Join(downloadDir, filename)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Handle filename conflicts by adding suffix
	finalPath = s.resolveFilenameConflict(finalPath)

	// Write file
	if err := os.WriteFile(finalPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return finalPath, nil
}

// OpenAttachment opens a file using the system default application
func (s *AttachmentServiceImpl) OpenAttachment(ctx context.Context, filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filePath cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Open file based on operating system
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", filePath)
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", filePath)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", filePath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Start the command (non-blocking)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	return nil
}

// GetDefaultDownloadPath returns the default download directory
func (s *AttachmentServiceImpl) GetDefaultDownloadPath() string {
	// Check config for custom download path
	if s.config != nil && s.config.Attachments.DownloadPath != "" {
		// Expand home directory if needed
		if strings.HasPrefix(s.config.Attachments.DownloadPath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				return filepath.Join(home, s.config.Attachments.DownloadPath[2:])
			}
		}
		return s.config.Attachments.DownloadPath
	}

	// Default to ~/Downloads/gmail-attachments
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Downloads", "gmail-attachments")
	}

	// Fallback to current directory
	return "./gmail-attachments"
}

// extractAttachmentsFromMessage extracts attachment info from Gmail message
func (s *AttachmentServiceImpl) extractAttachmentsFromMessage(message *googleGmail.Message) []AttachmentInfo {
	var attachments []AttachmentInfo
	
	if message == nil || message.Payload == nil {
		return attachments
	}

	index := 1
	var walkParts func(part *googleGmail.MessagePart)
	walkParts = func(part *googleGmail.MessagePart) {
		if part == nil {
			return
		}

		// Check if this part is an attachment
		if (part.Body != nil && part.Body.AttachmentId != "") || part.Filename != "" {
			// Skip inline images that are embedded content unless they have meaningful filenames
			isInline := part.Body != nil && part.Body.AttachmentId == "" && strings.Contains(strings.ToLower(part.MimeType), "image")
			if !isInline || (part.Filename != "" && !strings.HasPrefix(part.Filename, "image")) {
				// Extract filename from part or use a default with proper extension
				filename := part.Filename
				if filename == "" {
					// Try to generate a filename based on MIME type
					extension := s.getExtensionFromMimeType(part.MimeType)
					filename = fmt.Sprintf("attachment_%d%s", index, extension)
				}

				attachment := AttachmentInfo{
					Index:        index,
					AttachmentID: "",
					Filename:     filename,
					MimeType:     part.MimeType,
					Size:         0,
					Type:         s.categorizeAttachment(part.MimeType, filename),
					Inline:       isInline,
					ContentID:    "",
				}

				// Get attachment ID and size if available
				if part.Body != nil && part.Body.AttachmentId != "" {
					attachment.AttachmentID = part.Body.AttachmentId
					attachment.Size = int64(part.Body.Size)
				}

				// Extract Content-ID from headers for inline attachments
				if part.Headers != nil {
					for _, header := range part.Headers {
						if header.Name == "Content-ID" {
							attachment.ContentID = strings.Trim(header.Value, "<>")
							attachment.Inline = true
						}
					}
				}

				// Only add if we have either an attachment ID or a filename
				if attachment.AttachmentID != "" || attachment.Filename != "" {
					attachments = append(attachments, attachment)
					index++
				}
			}
		}

		// Recursively process parts
		for _, subPart := range part.Parts {
			walkParts(subPart)
		}
	}

	walkParts(message.Payload)
	return attachments
}

// categorizeAttachment determines the category of an attachment
func (s *AttachmentServiceImpl) categorizeAttachment(mimeType, filename string) string {
	mimeType = strings.ToLower(mimeType)
	filename = strings.ToLower(filename)

	// Image types
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}

	// Document types
	if strings.Contains(mimeType, "pdf") || strings.HasSuffix(filename, ".pdf") {
		return "document"
	}
	if strings.Contains(mimeType, "word") || strings.HasSuffix(filename, ".doc") || strings.HasSuffix(filename, ".docx") {
		return "document"
	}
	if strings.Contains(mimeType, "text") || strings.HasSuffix(filename, ".txt") || strings.HasSuffix(filename, ".md") {
		return "document"
	}

	// Spreadsheet types
	if strings.Contains(mimeType, "sheet") || strings.Contains(mimeType, "excel") || 
	   strings.HasSuffix(filename, ".xls") || strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".csv") {
		return "spreadsheet"
	}

	// Presentation types
	if strings.Contains(mimeType, "presentation") || strings.Contains(mimeType, "powerpoint") ||
	   strings.HasSuffix(filename, ".ppt") || strings.HasSuffix(filename, ".pptx") {
		return "presentation"
	}

	// Archive types
	if strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "compressed") ||
	   strings.HasSuffix(filename, ".zip") || strings.HasSuffix(filename, ".tar") ||
	   strings.HasSuffix(filename, ".gz") || strings.HasSuffix(filename, ".rar") {
		return "archive"
	}

	// Audio types
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}

	// Video types
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}

	// Calendar types
	if strings.Contains(mimeType, "calendar") || strings.HasSuffix(filename, ".ics") {
		return "calendar"
	}

	// Default
	return "file"
}

// resolveFilenameConflict adds a suffix if the file already exists
func (s *AttachmentServiceImpl) resolveFilenameConflict(originalPath string) string {
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		return originalPath
	}

	dir := filepath.Dir(originalPath)
	base := filepath.Base(originalPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	for i := 1; i < 1000; i++ { // Reasonable limit
		newName := fmt.Sprintf("%s_%d%s", name, i, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}

	// Fallback with timestamp
	return filepath.Join(dir, fmt.Sprintf("%s_%s%s", name, strconv.FormatInt(int64(os.Getpid()), 10), ext))
}

// getExtensionFromMimeType returns appropriate file extension for MIME type
func (s *AttachmentServiceImpl) getExtensionFromMimeType(mimeType string) string {
	mimeType = strings.ToLower(mimeType)
	
	switch {
	case strings.Contains(mimeType, "pdf"):
		return ".pdf"
	case strings.Contains(mimeType, "png"):
		return ".png"
	case strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg"):
		return ".jpg"
	case strings.Contains(mimeType, "gif"):
		return ".gif"
	case strings.Contains(mimeType, "word") || strings.Contains(mimeType, "msword"):
		return ".doc"
	case strings.Contains(mimeType, "wordprocessingml"):
		return ".docx"
	case strings.Contains(mimeType, "excel") || strings.Contains(mimeType, "spreadsheet"):
		return ".xlsx"
	case strings.Contains(mimeType, "powerpoint") || strings.Contains(mimeType, "presentation"):
		return ".pptx"
	case strings.Contains(mimeType, "zip"):
		return ".zip"
	case strings.Contains(mimeType, "tar"):
		return ".tar"
	case strings.Contains(mimeType, "gzip"):
		return ".gz"
	case strings.Contains(mimeType, "text/plain"):
		return ".txt"
	case strings.Contains(mimeType, "text/csv"):
		return ".csv"
	case strings.Contains(mimeType, "application/json"):
		return ".json"
	case strings.Contains(mimeType, "application/xml") || strings.Contains(mimeType, "text/xml"):
		return ".xml"
	case strings.Contains(mimeType, "calendar"):
		return ".ics"
	default:
		return ""
	}
}