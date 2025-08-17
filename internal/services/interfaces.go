package services

import (
	"context"
	"time"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/obsidian"
	"github.com/ajramos/gmail-tui/internal/prompts"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// MessageRepository handles message data operations
type MessageRepository interface {
	GetMessages(ctx context.Context, opts QueryOptions) (*MessagePage, error)
	GetMessage(ctx context.Context, id string) (*gmail.Message, error)
	SearchMessages(ctx context.Context, query string, opts QueryOptions) (*MessagePage, error)
	UpdateMessage(ctx context.Context, id string, updates MessageUpdates) error
	GetDrafts(ctx context.Context, maxResults int64) ([]*gmail_v1.Draft, error)
}

// EmailService handles email business logic
type EmailService interface {
	MarkAsRead(ctx context.Context, messageID string) error
	MarkAsUnread(ctx context.Context, messageID string) error
	ArchiveMessage(ctx context.Context, messageID string) error
	TrashMessage(ctx context.Context, messageID string) error
	SendMessage(ctx context.Context, from, to, subject, body string) error
	ReplyToMessage(ctx context.Context, originalID, replyBody string, send bool, cc []string) error
	BulkArchive(ctx context.Context, messageIDs []string) error
	BulkTrash(ctx context.Context, messageIDs []string) error
	SaveMessageToFile(ctx context.Context, messageID, filePath string) error
}

// LabelService handles label operations
type LabelService interface {
	ListLabels(ctx context.Context) ([]*gmail_v1.Label, error)
	CreateLabel(ctx context.Context, name string) (*gmail_v1.Label, error)
	RenameLabel(ctx context.Context, labelID, newName string) (*gmail_v1.Label, error)
	DeleteLabel(ctx context.Context, labelID string) error
	ApplyLabel(ctx context.Context, messageID, labelID string) error
	RemoveLabel(ctx context.Context, messageID, labelID string) error
	GetMessageLabels(ctx context.Context, messageID string) ([]string, error)
}

// AIService handles AI-related operations
type AIService interface {
	GenerateSummary(ctx context.Context, content string, options SummaryOptions) (*SummaryResult, error)
	GenerateSummaryStream(ctx context.Context, content string, options SummaryOptions, onToken func(string)) (*SummaryResult, error)
	GenerateReply(ctx context.Context, content string, options ReplyOptions) (string, error)
	SuggestLabels(ctx context.Context, content string, availableLabels []string) ([]string, error)
	FormatContent(ctx context.Context, content string, options FormatOptions) (string, error)
	ApplyCustomPrompt(ctx context.Context, content string, prompt string, variables map[string]string) (string, error)
	ApplyCustomPromptStream(ctx context.Context, content string, prompt string, variables map[string]string, onToken func(string)) (string, error)
}

// CacheService handles caching operations
type CacheService interface {
	GetSummary(ctx context.Context, accountEmail, messageID string) (string, bool, error)
	SaveSummary(ctx context.Context, accountEmail, messageID, summary string) error
	InvalidateSummary(ctx context.Context, accountEmail, messageID string) error
	ClearCache(ctx context.Context, accountEmail string) error
}

// SlackService handles Slack integration operations
type SlackService interface {
	ForwardEmail(ctx context.Context, messageID string, options SlackForwardOptions) error
	ValidateWebhook(ctx context.Context, webhookURL string) error
	ListConfiguredChannels(ctx context.Context) ([]SlackChannel, error)
}

// SearchService handles search operations
type SearchService interface {
	Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error)
	BuildQuery(ctx context.Context, criteria SearchCriteria) (string, error)
	GetSearchHistory(ctx context.Context) ([]string, error)
	SaveSearchHistory(ctx context.Context, query string) error
}

// PromptService handles prompt template operations
type PromptService interface {
	ListPrompts(ctx context.Context, category string) ([]*PromptTemplate, error)
	GetPrompt(ctx context.Context, id int) (*PromptTemplate, error)
	ApplyPrompt(ctx context.Context, messageContent string, promptID int, variables map[string]string) (*PromptResult, error)
	GetCachedResult(ctx context.Context, accountEmail, messageID string, promptID int) (*PromptResult, error)
	IncrementUsage(ctx context.Context, promptID int) error
	GetUsageStats(ctx context.Context) (*UsageStats, error)
	SaveResult(ctx context.Context, accountEmail, messageID string, promptID int, resultText string) error

	// NUEVO: Aplicar prompt a múltiples mensajes
	ApplyBulkPrompt(ctx context.Context, accountEmail string, messageIDs []string, promptID int, variables map[string]string) (*BulkPromptResult, error)
	ApplyBulkPromptStream(ctx context.Context, accountEmail string, messageIDs []string, promptID int, variables map[string]string, onToken func(string)) (*BulkPromptResult, error)
	GetCachedBulkResult(ctx context.Context, accountEmail string, messageIDs []string, promptID int) (*BulkPromptResult, error)
	SaveBulkResult(ctx context.Context, accountEmail string, messageIDs []string, promptID int, resultText string) error

	// Cache management
	ClearPromptCache(ctx context.Context, accountEmail string) error
	ClearAllPromptCaches(ctx context.Context) error
}

// Data structures

type QueryOptions struct {
	MaxResults int64
	PageToken  string
	LabelIDs   []string
	Query      string
}

type MessagePage struct {
	Messages      []*gmail_v1.Message
	NextPageToken string
	TotalCount    int
}

type MessageUpdates struct {
	AddLabels    []string
	RemoveLabels []string
	MarkAsRead   *bool
}

type SummaryOptions struct {
	MaxLength       int
	Language        string
	StreamEnabled   bool
	UseCache        bool
	ForceRegenerate bool
	MessageID       string
	AccountEmail    string
}

type SummaryResult struct {
	Summary   string
	FromCache bool
	Language  string
	Duration  time.Duration
}

type ReplyOptions struct {
	Language string
	Tone     string
	Length   string
}

type FormatOptions struct {
	WrapWidth      int
	EnableMarkdown bool
	TouchUpMode    bool
}

type SearchOptions struct {
	MaxResults int64
	PageToken  string
	SortBy     string
	SortOrder  string
}

type SearchCriteria struct {
	From          string
	To            string
	Subject       string
	HasWords      string
	DoesntHave    string
	Size          string
	DateWithin    string
	HasAttachment bool
	Labels        []string
	Folders       []string
}

type SearchResult struct {
	Messages      []*gmail_v1.Message
	NextPageToken string
	TotalCount    int
	Query         string
	Duration      time.Duration
}

// Prompt-related data structures
type PromptTemplate = prompts.PromptTemplate
type PromptResult = prompts.PromptResult

type PromptApplyOptions struct {
	AccountEmail string
	MessageID    string
	Variables    map[string]string
}

// NUEVO: Resultado de bulk prompt
type BulkPromptResult struct {
	PromptID     int
	MessageCount int
	Summary      string
	MessageIDs   []string
	Duration     time.Duration
	FromCache    bool
	AccountEmail string
	CreatedAt    time.Time
}

// UsageStats represents prompt usage statistics
type UsageStats struct {
	TopPrompts      []PromptUsageStat `json:"top_prompts"`
	TotalUsage      int               `json:"total_usage"`
	UniquePrompts   int               `json:"unique_prompts"`
	LastUsed        time.Time         `json:"last_used"`
	FavoritePrompts []PromptUsageStat `json:"favorite_prompts"`
}

// PromptUsageStat represents usage statistics for a single prompt
type PromptUsageStat struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	UsageCount  int    `json:"usage_count"`
	IsFavorite  bool   `json:"is_favorite"`
	LastUsed    string `json:"last_used"`
}

// Slack-related data structures
type SlackForwardOptions struct {
	ChannelID        string // Internal channel identifier
	WebhookURL       string // Slack webhook URL
	ChannelName      string // Display name for user feedback
	UserMessage      string // Optional user message: "Hey guys, heads up with this email"
	FormatStyle      string // "summary", "compact", "full", "raw"
	ProcessedContent string // TUI-processed content for "full" format (optional)
}

type SlackChannel struct {
	ID          string `json:"id"`          // Internal ID
	Name        string `json:"name"`        // Display name: "team-updates", "personal-dm"
	WebhookURL  string `json:"webhook_url"` // Slack webhook URL
	Default     bool   `json:"default"`     // Default selection
	Description string `json:"description"` // Optional description
}

// ObsidianService handles Obsidian integration operations
type ObsidianService interface {
	IngestEmailToObsidian(ctx context.Context, message *gmail.Message, options obsidian.ObsidianOptions) (*obsidian.ObsidianIngestResult, error)
	IngestBulkEmailsToObsidian(ctx context.Context, messages []*gmail.Message, accountEmail string, onProgress func(int, int, error)) (*obsidian.BulkObsidianResult, error)
	GetObsidianTemplates(ctx context.Context) ([]*obsidian.ObsidianTemplate, error)
	ValidateObsidianConnection(ctx context.Context) error
	GetObsidianVaultPath() string
	GetConfig() *obsidian.ObsidianConfig
	UpdateConfig(config *obsidian.ObsidianConfig)
}

// LinkService handles link extraction and opening operations
type LinkService interface {
	GetMessageLinks(ctx context.Context, messageID string) ([]LinkInfo, error)
	OpenLink(ctx context.Context, url string) error
	ValidateURL(url string) error
}

// LinkInfo represents a link found in an email message
type LinkInfo struct {
	Index int    `json:"index"` // Reference number [1], [2], etc.
	URL   string `json:"url"`   // Full URL
	Text  string `json:"text"`  // Link text/description
	Type  string `json:"type"`  // "html" or "plain" or "email" or "file"
}
