package services

import (
	"context"
	"time"

	"github.com/ajramos/gmail-tui/internal/gmail"
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
	GenerateReply(ctx context.Context, content string, options ReplyOptions) (string, error)
	SuggestLabels(ctx context.Context, content string, availableLabels []string) ([]string, error)
	FormatContent(ctx context.Context, content string, options FormatOptions) (string, error)
	ApplyCustomPrompt(ctx context.Context, content string, prompt string, variables map[string]string) (string, error)
}

// CacheService handles caching operations
type CacheService interface {
	GetSummary(ctx context.Context, accountEmail, messageID string) (string, bool, error)
	SaveSummary(ctx context.Context, accountEmail, messageID, summary string) error
	InvalidateSummary(ctx context.Context, accountEmail, messageID string) error
	ClearCache(ctx context.Context, accountEmail string) error
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
	SaveResult(ctx context.Context, accountEmail, messageID string, promptID int, resultText string) error
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
