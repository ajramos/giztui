package services

import (
	"context"
	"time"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/obsidian"
	"github.com/ajramos/giztui/internal/prompts"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// MessageRepository handles message data operations
type MessageRepository interface {
	GetMessages(ctx context.Context, opts QueryOptions) (*MessagePage, error)
	GetMessage(ctx context.Context, id string) (*gmail.Message, error)
	SearchMessages(ctx context.Context, query string, opts QueryOptions) (*MessagePage, error)
	UpdateMessage(ctx context.Context, id string, updates MessageUpdates) error
	GetDrafts(ctx context.Context, maxResults int64) ([]*gmail_v1.Draft, error)
	GetDraft(ctx context.Context, draftID string) (*gmail_v1.Draft, error)
}

// EmailService handles email business logic
type EmailService interface {
	MarkAsRead(ctx context.Context, messageID string) error
	MarkAsUnread(ctx context.Context, messageID string) error
	BulkMarkAsRead(ctx context.Context, messageIDs []string) error
	BulkMarkAsUnread(ctx context.Context, messageIDs []string) error
	ArchiveMessage(ctx context.Context, messageID string) error
	ArchiveMessageAsMove(ctx context.Context, messageID, labelID, labelName string) error
	TrashMessage(ctx context.Context, messageID string) error
	SendMessage(ctx context.Context, from, to, subject, body string, cc, bcc []string) error
	ReplyToMessage(ctx context.Context, originalID, replyBody string, send bool, cc []string) error
	BulkArchive(ctx context.Context, messageIDs []string) error
	BulkTrash(ctx context.Context, messageIDs []string) error
	SaveMessageToFile(ctx context.Context, messageID, filePath string) error
	MoveToSystemFolder(ctx context.Context, messageID, systemFolderID, folderName string) error
}

// LabelService handles label operations
type LabelService interface {
	ListLabels(ctx context.Context) ([]*gmail_v1.Label, error)
	CreateLabel(ctx context.Context, name string) (*gmail_v1.Label, error)
	RenameLabel(ctx context.Context, labelID, newName string) (*gmail_v1.Label, error)
	DeleteLabel(ctx context.Context, labelID string) error
	ApplyLabel(ctx context.Context, messageID, labelID string) error
	RemoveLabel(ctx context.Context, messageID, labelID string) error
	BulkApplyLabel(ctx context.Context, messageIDs []string, labelID string) error
	BulkRemoveLabel(ctx context.Context, messageIDs []string, labelID string) error
	GetMessageLabels(ctx context.Context, messageID string) ([]string, error)
}

// LabelVisibility defines label visibility options
type LabelVisibility string

const (
	LabelVisibilityShow LabelVisibility = "labelShow"
	LabelVisibilityHide LabelVisibility = "labelHide"
)

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
	ApplyPromptStream(ctx context.Context, messageContent string, promptID int, variables map[string]string, onToken func(string)) (*PromptResult, error)
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

	// CRUD operations for prompt templates
	CreatePrompt(ctx context.Context, name, description, promptText, category string) (int, error)
	UpdatePrompt(ctx context.Context, id int, name, description, promptText, category string) error
	DeletePrompt(ctx context.Context, id int) error
	FindPromptByName(ctx context.Context, name string) (*PromptTemplate, error)

	// File operations for prompt templates
	CreateFromFile(ctx context.Context, filePath string) (int, error)
	ExportToFile(ctx context.Context, id int, filePath string) error
}

// ContentNavigationService handles content search and navigation within message text
type ContentNavigationService interface {
	// Search operations
	SearchContent(ctx context.Context, content string, query string, caseSensitive bool) (*ContentSearchResult, error)
	FindNextMatch(ctx context.Context, searchResult *ContentSearchResult, currentPosition int) (int, error)
	FindPreviousMatch(ctx context.Context, searchResult *ContentSearchResult, currentPosition int) (int, error)

	// Navigation operations
	FindNextParagraph(ctx context.Context, content string, currentPosition int) (int, error)
	FindPreviousParagraph(ctx context.Context, content string, currentPosition int) (int, error)
	FindNextWord(ctx context.Context, content string, currentPosition int) (int, error)
	FindPreviousWord(ctx context.Context, content string, currentPosition int) (int, error)

	// Position operations
	GetLineFromPosition(ctx context.Context, content string, position int) (int, error)
	GetPositionFromLine(ctx context.Context, content string, line int) (int, error)
	GetContentLength(ctx context.Context, content string) int
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

// ContentSearchResult holds search results for content within a message
type ContentSearchResult struct {
	Query         string        `json:"query"`
	CaseSensitive bool          `json:"case_sensitive"`
	Matches       []int         `json:"matches"`     // Positions of matches in the content
	MatchCount    int           `json:"match_count"` // Total number of matches
	Content       string        `json:"-"`           // Original content (not serialized)
	Duration      time.Duration `json:"duration"`
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
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	UsageCount int    `json:"usage_count"`
	IsFavorite bool   `json:"is_favorite"`
	LastUsed   string `json:"last_used"`
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

// AttachmentService handles attachment extraction and download operations
type AttachmentService interface {
	GetMessageAttachments(ctx context.Context, messageID string) ([]AttachmentInfo, error)
	DownloadAttachment(ctx context.Context, messageID, attachmentID, savePath string) (string, error)
	DownloadAttachmentWithFilename(ctx context.Context, messageID, attachmentID, savePath, suggestedFilename string) (string, error)
	OpenAttachment(ctx context.Context, filePath string) error
	GetDefaultDownloadPath() string
}

// AttachmentInfo represents an attachment found in an email message
type AttachmentInfo struct {
	Index        int    `json:"index"`         // Reference number [1], [2], etc.
	AttachmentID string `json:"attachment_id"` // Gmail attachment ID
	Filename     string `json:"filename"`      // Original filename
	MimeType     string `json:"mime_type"`     // MIME type (application/pdf, image/png, etc.)
	Size         int64  `json:"size"`          // Size in bytes
	Type         string `json:"type"`          // Category: "document", "image", "archive", "spreadsheet", etc.
	Inline       bool   `json:"inline"`        // Whether it's an inline image/attachment
	ContentID    string `json:"content_id"`    // Content-ID for inline attachments
}

// GmailWebService handles opening Gmail messages in web interface
type GmailWebService interface {
	OpenMessageInWeb(ctx context.Context, messageID string) error
	ValidateMessageID(messageID string) error
	GenerateGmailWebURL(messageID string) string
}

// ThemeService handles theme operations
type ThemeService interface {
	// Theme discovery and listing
	ListAvailableThemes(ctx context.Context) ([]string, error)
	GetCurrentTheme(ctx context.Context) (string, error)

	// Theme application
	ApplyTheme(ctx context.Context, name string) error

	// Theme preview and information
	PreviewTheme(ctx context.Context, name string) (*ThemeConfig, error)
	GetThemeConfig(ctx context.Context, name string) (*ThemeConfig, error)

	// Theme validation
	ValidateTheme(ctx context.Context, name string) error
}

// ThemeConfig represents a theme configuration for preview and display
type ThemeConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Color information for preview
	EmailColors struct {
		UnreadColor    string `json:"unread_color"`
		ReadColor      string `json:"read_color"`
		ImportantColor string `json:"important_color"`
		SentColor      string `json:"sent_color"`
		DraftColor     string `json:"draft_color"`
	} `json:"email_colors"`

	UIColors struct {
		// Basic UI colors
		FgColor     string `json:"fg_color"`
		BgColor     string `json:"bg_color"`
		BorderColor string `json:"border_color"`
		FocusColor  string `json:"focus_color"`

		// Component colors (previously hardcoded)
		TitleColor  string `json:"title_color"`
		FooterColor string `json:"footer_color"`
		HintColor   string `json:"hint_color"`

		// Selection colors
		SelectionBgColor string `json:"selection_bg_color"`
		SelectionFgColor string `json:"selection_fg_color"`

		// Status colors
		ErrorColor   string `json:"error_color"`
		SuccessColor string `json:"success_color"`
		WarningColor string `json:"warning_color"`
		InfoColor    string `json:"info_color"`

		// Input colors
		InputBgColor string `json:"input_bg_color"`
		InputFgColor string `json:"input_fg_color"`
		LabelColor   string `json:"label_color"`
	} `json:"ui_colors"`
}

// DisplayService handles display and UI state operations
type DisplayService interface {
	// Header visibility management
	ToggleHeaderVisibility() bool
	SetHeaderVisibility(visible bool)
	IsHeaderVisible() bool
}

// QueryService handles saved query operations
type QueryService interface {
	// Query management
	SaveQuery(ctx context.Context, name, query, description, category string) (*SavedQueryInfo, error)
	GetQuery(ctx context.Context, name string) (*SavedQueryInfo, error)
	GetQueryByID(ctx context.Context, id int64) (*SavedQueryInfo, error)
	ListQueries(ctx context.Context, category string) ([]*SavedQueryInfo, error)
	SearchQueries(ctx context.Context, searchTerm string) ([]*SavedQueryInfo, error)
	DeleteQuery(ctx context.Context, id int64) error
	DeleteQueryByName(ctx context.Context, name string) error

	// Query usage tracking
	RecordQueryUsage(ctx context.Context, id int64) error

	// Query organization
	GetCategories(ctx context.Context) ([]string, error)
	UpdateQueryCategory(ctx context.Context, id int64, category string) error
}

// SavedQueryInfo represents information about a saved query
type SavedQueryInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	Description string `json:"description"`
	Category    string `json:"category"`
	UseCount    int    `json:"use_count"`
	LastUsed    int64  `json:"last_used"`
	CreatedAt   int64  `json:"created_at"`
}

// ThreadService handles message threading operations
type ThreadService interface {
	// Thread management
	GetThreads(ctx context.Context, opts ThreadQueryOptions) (*ThreadPage, error)
	GetThreadMessages(ctx context.Context, threadID string, opts MessageQueryOptions) ([]*gmail_v1.Message, error)
	GetThreadInfo(ctx context.Context, threadID string) (*ThreadInfo, error)

	// Thread state management
	SetThreadExpanded(ctx context.Context, accountEmail, threadID string, expanded bool) error
	IsThreadExpanded(ctx context.Context, accountEmail, threadID string) (bool, error)
	ExpandAllThreads(ctx context.Context, accountEmail string) error
	CollapseAllThreads(ctx context.Context, accountEmail string) error

	// Thread summaries and AI integration
	GenerateThreadSummary(ctx context.Context, threadID string, options ThreadSummaryOptions) (*ThreadSummaryResult, error)
	GenerateThreadSummaryStream(ctx context.Context, threadID string, options ThreadSummaryOptions, onToken func(string)) (*ThreadSummaryResult, error)
	GetCachedThreadSummary(ctx context.Context, accountEmail, threadID string) (*ThreadSummaryResult, error)

	// Thread search and navigation
	SearchWithinThread(ctx context.Context, threadID, query string) (*ThreadSearchResult, error)
	GetNextThread(ctx context.Context, currentThreadID string) (string, error)
	GetPreviousThread(ctx context.Context, currentThreadID string) (string, error)

	// Thread organization
	GetThreadsByLabel(ctx context.Context, labelID string, opts ThreadQueryOptions) (*ThreadPage, error)
	GetUnreadThreads(ctx context.Context, opts ThreadQueryOptions) (*ThreadPage, error)

	// Bulk thread operations
	BulkExpandThreads(ctx context.Context, accountEmail string, threadIDs []string) error
	BulkCollapseThreads(ctx context.Context, accountEmail string, threadIDs []string) error
}

// UndoService handles undo operations for reversible actions
type UndoService interface {
	// Record an action for potential undo
	RecordAction(ctx context.Context, action *UndoableAction) error

	// Undo the last recorded action
	UndoLastAction(ctx context.Context) (*UndoResult, error)

	// Check if undo is available
	HasUndoableAction() bool

	// Get description of what will be undone
	GetUndoDescription() string

	// Clear undo history (e.g., after app restart)
	ClearUndoHistory() error
}

// Threading-related data structures

// ThreadInfo represents metadata about a conversation thread
type ThreadInfo struct {
	ThreadID      string    `json:"thread_id"`
	MessageCount  int       `json:"message_count"`
	UnreadCount   int       `json:"unread_count"`
	Participants  []string  `json:"participants"`
	Subject       string    `json:"subject"`
	LatestDate    time.Time `json:"latest_date"`
	HasAttachment bool      `json:"has_attachment"`
	Labels        []string  `json:"labels"`
	IsExpanded    bool      `json:"is_expanded"`
	RootMessageID string    `json:"root_message_id"`
}

// ThreadPage represents a page of conversation threads
type ThreadPage struct {
	Threads       []*ThreadInfo `json:"threads"`
	NextPageToken string        `json:"next_page_token"`
	TotalCount    int           `json:"total_count"`
}

// ThreadQueryOptions specifies options for querying threads
type ThreadQueryOptions struct {
	MaxResults  int64    `json:"max_results"`
	PageToken   string   `json:"page_token"`
	LabelIDs    []string `json:"label_ids"`
	Query       string   `json:"query"`
	IncludeRead bool     `json:"include_read"`
}

// MessageQueryOptions specifies options for querying messages within a thread
type MessageQueryOptions struct {
	IncludeDeleted bool   `json:"include_deleted"`
	Format         string `json:"format"`     // "minimal", "full", "raw", "metadata"
	SortOrder      string `json:"sort_order"` // "asc", "desc"
}

// ThreadSummaryOptions specifies options for generating thread summaries
type ThreadSummaryOptions struct {
	MaxLength       int    `json:"max_length"`
	Language        string `json:"language"`
	StreamEnabled   bool   `json:"stream_enabled"`
	UseCache        bool   `json:"use_cache"`
	ForceRegenerate bool   `json:"force_regenerate"`
	AccountEmail    string `json:"account_email"`
	SummaryType     string `json:"summary_type"` // "conversation", "action_items", "key_points"
}

// ThreadSummaryResult represents the result of a thread summary generation
type ThreadSummaryResult struct {
	ThreadID     string        `json:"thread_id"`
	Summary      string        `json:"summary"`
	SummaryType  string        `json:"summary_type"`
	FromCache    bool          `json:"from_cache"`
	Language     string        `json:"language"`
	Duration     time.Duration `json:"duration"`
	MessageCount int           `json:"message_count"`
	CreatedAt    time.Time     `json:"created_at"`
}

// ThreadSearchResult represents search results within a thread
type ThreadSearchResult struct {
	ThreadID   string        `json:"thread_id"`
	Query      string        `json:"query"`
	Matches    []ThreadMatch `json:"matches"`
	MatchCount int           `json:"match_count"`
	Duration   time.Duration `json:"duration"`
}

// ThreadMatch represents a search match within a thread
type ThreadMatch struct {
	MessageID string `json:"message_id"`
	Position  int    `json:"position"`
	Context   string `json:"context"`
	MatchText string `json:"match_text"`
}

// ThreadingConfig represents threading configuration (mirrored from config package to avoid circular imports)
type ThreadingConfig struct {
	Enabled              bool   `json:"enabled"`
	DefaultView          string `json:"default_view"`
	AutoExpandUnread     bool   `json:"auto_expand_unread"`
	ShowThreadCount      bool   `json:"show_thread_count"`
	IndentReplies        bool   `json:"indent_replies"`
	MaxThreadDepth       int    `json:"max_thread_depth"`
	ThreadSummaryEnabled bool   `json:"thread_summary_enabled"`
	PreserveThreadState  bool   `json:"preserve_thread_state"`
}

// Undo-related data structures

// UndoActionType represents the type of action that can be undone
type UndoActionType string

const (
	UndoActionArchive     UndoActionType = "archive"
	UndoActionUnarchive   UndoActionType = "unarchive"
	UndoActionTrash       UndoActionType = "trash"
	UndoActionRestore     UndoActionType = "restore"
	UndoActionMarkRead    UndoActionType = "mark_read"
	UndoActionMarkUnread  UndoActionType = "mark_unread"
	UndoActionLabelAdd    UndoActionType = "label_add"
	UndoActionLabelRemove UndoActionType = "label_remove"
	UndoActionMove        UndoActionType = "move"
)

// ActionState represents the previous state of a message for undo operations
type ActionState struct {
	Labels    []string `json:"labels"`   // Previous labels
	IsRead    bool     `json:"is_read"`  // Previous read state
	IsInInbox bool     `json:"is_inbox"` // Whether message was in inbox
}

// UndoableAction represents an action that can be undone
type UndoableAction struct {
	ID          string                 `json:"id"`          // Unique action ID
	Type        UndoActionType         `json:"type"`        // Type of action
	MessageIDs  []string               `json:"message_ids"` // Affected message IDs
	Timestamp   time.Time              `json:"timestamp"`   // When action was performed
	PrevState   map[string]ActionState `json:"prev_state"`  // Previous state for reversal
	Description string                 `json:"description"` // Human-readable description
	IsBulk      bool                   `json:"is_bulk"`     // Whether it was a bulk operation
	ExtraData   map[string]interface{} `json:"extra_data"`  // Additional data for specific action types
}

// UndoResult represents the result of an undo operation
type UndoResult struct {
	Success      bool                   `json:"success"`       // Whether undo was successful
	Description  string                 `json:"description"`   // Description of what was undone
	MessageCount int                    `json:"message_count"` // Number of messages affected
	Errors       []string               `json:"errors"`        // Any errors that occurred
	ActionType   UndoActionType         `json:"action_type"`   // Type of action that was undone
	MessageIDs   []string               `json:"message_ids"`   // IDs of messages affected
	ExtraData    map[string]interface{} `json:"extra_data"`    // Additional data for cache updates
}

// CompositionService handles email composition operations
type CompositionService interface {
	// Composition lifecycle
	CreateComposition(ctx context.Context, compositionType CompositionType, originalMessageID string) (*Composition, error)
	LoadDraftComposition(ctx context.Context, draftID string) (*Composition, error)
	SaveDraft(ctx context.Context, composition *Composition) (string, error)
	DeleteComposition(ctx context.Context, compositionID string) error
	SendComposition(ctx context.Context, composition *Composition) error

	// Validation & processing
	ValidateComposition(composition *Composition) []ValidationError
	ProcessReply(ctx context.Context, originalMessageID string) (*ReplyContext, error)
	ProcessReplyAll(ctx context.Context, originalMessageID string) (*ReplyAllContext, error)
	ProcessForward(ctx context.Context, originalMessageID string) (*ForwardContext, error)

	// Templates & suggestions
	GetTemplates(ctx context.Context, category string) ([]*EmailTemplate, error)
	ApplyTemplate(ctx context.Context, composition *Composition, templateID string) error
	GetRecipientSuggestions(ctx context.Context, query string) ([]Recipient, error)
}

// Composition-related data structures

// CompositionType represents different types of email composition
type CompositionType string

const (
	CompositionTypeNew      CompositionType = "new"
	CompositionTypeReply    CompositionType = "reply"
	CompositionTypeReplyAll CompositionType = "reply_all"
	CompositionTypeForward  CompositionType = "forward"
	CompositionTypeDraft    CompositionType = "draft"
)

// Composition represents an email being composed
type Composition struct {
	ID          string          `json:"id"`
	Type        CompositionType `json:"type"`
	To          []Recipient     `json:"to"`
	CC          []Recipient     `json:"cc"`
	BCC         []Recipient     `json:"bcc"`
	Subject     string          `json:"subject"`
	Body        string          `json:"body"`
	Attachments []Attachment    `json:"attachments"`
	OriginalID  string          `json:"original_id,omitempty"`
	DraftID     string          `json:"draft_id,omitempty"`
	IsDraft     bool            `json:"is_draft"`
	CreatedAt   time.Time       `json:"created_at"`
	ModifiedAt  time.Time       `json:"modified_at"`
}

// Recipient represents an email recipient
type Recipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// ValidationError represents a validation error for composition
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ReplyContext contains context information for replying to a message
type ReplyContext struct {
	OriginalMessage *gmail.Message `json:"-"` // Don't serialize the full message
	Recipients      []Recipient    `json:"recipients"`
	Subject         string         `json:"subject"`
	QuotedBody      string         `json:"quoted_body"`
	ThreadID        string         `json:"thread_id,omitempty"`
	OriginalSender  Recipient      `json:"original_sender"`
	OriginalDate    time.Time      `json:"original_date"`
}

// ReplyAllContext contains context information for replying to all recipients
type ReplyAllContext struct {
	OriginalMessage *gmail.Message `json:"-"`          // Don't serialize the full message
	Recipients      []Recipient    `json:"recipients"` // To recipients (including original sender)
	CC              []Recipient    `json:"cc"`         // CC recipients from original
	Subject         string         `json:"subject"`
	QuotedBody      string         `json:"quoted_body"`
	ThreadID        string         `json:"thread_id,omitempty"`
	OriginalSender  Recipient      `json:"original_sender"`
	OriginalDate    time.Time      `json:"original_date"`
}

// ForwardContext contains context information for forwarding a message
type ForwardContext struct {
	OriginalMessage *gmail.Message `json:"-"` // Don't serialize the full message
	Subject         string         `json:"subject"`
	ForwardedBody   string         `json:"forwarded_body"`
	OriginalSender  Recipient      `json:"original_sender"`
	OriginalDate    time.Time      `json:"original_date"`
}

// EmailTemplate represents a reusable email template
type EmailTemplate struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Category   string            `json:"category"`
	Subject    string            `json:"subject"`
	Body       string            `json:"body"`
	Variables  []string          `json:"variables"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
	ModifiedAt time.Time         `json:"modified_at"`
}

// Attachment represents a file attachment (reusing existing pattern if available)
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	FilePath string `json:"file_path,omitempty"`
	Data     []byte `json:"-"` // Don't serialize attachment data
}
