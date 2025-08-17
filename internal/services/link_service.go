package services

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/render"
)

// LinkServiceImpl implements LinkService
type LinkServiceImpl struct {
	gmailClient   *gmail.Client
	emailRenderer *render.EmailRenderer
}

// NewLinkService creates a new link service
func NewLinkService(gmailClient *gmail.Client, emailRenderer *render.EmailRenderer) *LinkServiceImpl {
	return &LinkServiceImpl{
		gmailClient:   gmailClient,
		emailRenderer: emailRenderer,
	}
}

// GetMessageLinks extracts all links from a message
func (s *LinkServiceImpl) GetMessageLinks(ctx context.Context, messageID string) ([]LinkInfo, error) {
	if messageID == "" {
		return nil, fmt.Errorf("messageID cannot be empty")
	}

	// Get message content
	message, err := s.gmailClient.GetMessageWithContent(messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message content: %w", err)
	}

	// Extract links using the existing render functionality
	links := s.extractLinksFromMessage(message)

	// Convert to LinkInfo format with categorization
	var linkInfos []LinkInfo
	for _, link := range links {
		linkInfo := LinkInfo{
			Index: link.Index,
			URL:   link.URL,
			Text:  link.Text,
			Type:  s.categorizeLink(link.URL),
		}
		linkInfos = append(linkInfos, linkInfo)
	}

	return linkInfos, nil
}

// OpenLink opens a URL using the system default browser
func (s *LinkServiceImpl) OpenLink(ctx context.Context, url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Validate URL first
	if err := s.ValidateURL(url); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Open URL based on operating system
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Start the command (non-blocking)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open URL: %w", err)
	}

	return nil
}

// ValidateURL validates a URL for security and format
func (s *LinkServiceImpl) ValidateURL(urlStr string) error {
	if strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check for supported schemes
	switch strings.ToLower(parsedURL.Scheme) {
	case "http", "https", "mailto", "file", "ftp", "ftps":
		// Allowed schemes
	case "":
		// If no scheme, assume https for web URLs
		if strings.Contains(urlStr, ".") {
			return nil
		}
		return fmt.Errorf("URL missing scheme")
	default:
		return fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	return nil
}

// extractLinksFromMessage extracts links from a message using existing render logic
func (s *LinkServiceImpl) extractLinksFromMessage(message *gmail.Message) []render.LinkRef {
	var links []render.LinkRef

	// Try HTML content first
	if strings.TrimSpace(message.HTML) != "" {
		// Use similar logic to render.renderHTMLToText but focused on link extraction
		htmlLinks := s.extractLinksFromHTML(message.HTML)
		links = append(links, htmlLinks...)
	}

	// If no HTML links or as fallback, check plain text
	if len(links) == 0 && strings.TrimSpace(message.PlainText) != "" {
		plainLinks := s.extractLinksFromPlainText(message.PlainText)
		links = append(links, plainLinks...)
	}

	return links
}

// extractLinksFromHTML extracts links from HTML content
func (s *LinkServiceImpl) extractLinksFromHTML(htmlContent string) []render.LinkRef {
	var links []render.LinkRef
	
	// Use regex to find href attributes (simplified approach)
	// In a full implementation, you'd want to use an HTML parser
	hrefRegex := regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["'][^>]*>([^<]+)</a>`)
	matches := hrefRegex.FindAllStringSubmatch(htmlContent, -1)
	
	for i, match := range matches {
		if len(match) >= 3 {
			url := strings.TrimSpace(match[1])
			text := strings.TrimSpace(match[2])
			if url != "" {
				links = append(links, render.LinkRef{
					Index: i + 1,
					URL:   url,
					Text:  text,
				})
			}
		}
	}

	return links
}

// extractLinksFromPlainText extracts URLs from plain text
func (s *LinkServiceImpl) extractLinksFromPlainText(plainText string) []render.LinkRef {
	// Use the same regex as in render/format.go
	urlRegex := regexp.MustCompile(`(?i)\bhttps?://[\w\-\._~:/%\?#\[\]@!$&'()*+,;=]+`)
	matches := urlRegex.FindAllString(plainText, -1)
	
	var links []render.LinkRef
	seen := make(map[string]bool)
	
	for i, match := range matches {
		if !seen[match] {
			links = append(links, render.LinkRef{
				Index: i + 1,
				URL:   match,
				Text:  match, // For plain text, URL is both the link and the text
			})
			seen[match] = true
		}
	}

	return links
}

// categorizeLink determines the type of link
func (s *LinkServiceImpl) categorizeLink(urlStr string) string {
	if urlStr == "" {
		return "unknown"
	}

	// Parse URL to get scheme and host
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "plain"
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	host := strings.ToLower(parsedURL.Host)

	switch scheme {
	case "mailto":
		return "email"
	case "file":
		return "file"
	case "http", "https":
		// Further categorize web links
		if host == "" {
			return "html"
		}
		// Check if it's an external domain (you could enhance this with a whitelist)
		if strings.Contains(host, "github.com") || strings.Contains(host, "docs.") {
			return "external"
		}
		return "html"
	case "ftp", "ftps":
		return "file"
	default:
		return "plain"
	}
}