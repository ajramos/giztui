package desktop

import "time"

// MessageSummary is a lightweight message representation for list views.
// It is JSON-serializable and safe to expose to the frontend.
type MessageSummary struct {
	ID       string    `json:"id"`
	ThreadID string    `json:"threadId"`
	Subject  string    `json:"subject"`
	From     string    `json:"from"`
	Snippet  string    `json:"snippet"`
	Date     time.Time `json:"date"`
	Unread   bool      `json:"unread"`
	Labels   []string  `json:"labels"`
}

// MessageDetail is the full message representation for the reading pane.
type MessageDetail struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"threadId"`
	Subject   string    `json:"subject"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Cc        string    `json:"cc"`
	Date      time.Time `json:"date"`
	Unread    bool      `json:"unread"`
	Labels    []string  `json:"labels"`
	PlainText string    `json:"plainText"`
	HTML      string    `json:"html"`
}

// MessageList is a page of message summaries plus a pagination cursor.
type MessageList struct {
	Messages      []MessageSummary `json:"messages"`
	NextPageToken string           `json:"nextPageToken"`
}

// Label is a JSON-serializable Gmail label.
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DraftSummary is a lightweight draft for the drafts list.
type DraftSummary struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Snippet string `json:"snippet"`
}

// DraftDetail is a draft loaded for editing.
type DraftDetail struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Cc      string `json:"cc"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// AccountInfo is a JSON-serializable account for the account switcher.
type AccountInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Active      bool   `json:"active"`
}

// ThemeColors is a flattened theme palette for the frontend to apply as CSS
// variables.
type ThemeColors struct {
	Name        string `json:"name"`
	Bg          string `json:"bg"`
	Fg          string `json:"fg"`
	Border      string `json:"border"`
	Accent      string `json:"accent"`
	Primary     string `json:"primary"`
	Danger      string `json:"danger"`
	Warning     string `json:"warning"`
	Success     string `json:"success"`
	SelectionBg string `json:"selectionBg"`
	InputBg     string `json:"inputBg"`
	Unread      string `json:"unread"`
	Muted       string `json:"muted"`
}

// AnalyzerInput is the lightweight message data the inbox analyzer needs,
// passed from the frontend (which already has it) to avoid re-fetching.
type AnalyzerInput struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	From    string `json:"from"`
	Snippet string `json:"snippet"`
}

// PlanCategory is one category of the inbox action plan.
type PlanCategory struct {
	Name        string   `json:"name"`
	Priority    string   `json:"priority"`
	Description string   `json:"description"`
	Action      string   `json:"action"`
	Label       string   `json:"label"`
	MessageIDs  []string `json:"messageIds"`
	// ByRule is true for categories resolved by a deterministic rule (the
	// first pass), so the UI can mark them distinctly from AI categories.
	ByRule bool `json:"byRule"`
}

// ActionPlanResult is the AI inbox action plan.
type ActionPlanResult struct {
	Categories    []PlanCategory `json:"categories"`
	TotalAnalyzed int            `json:"totalAnalyzed"`
	ReadManually  int            `json:"readManually"`
}

// SavedQuery is a JSON-serializable saved Gmail search.
type SavedQuery struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// Link is a JSON-serializable link extracted from a message body.
type Link struct {
	Index int    `json:"index"`
	URL   string `json:"url"`
	Text  string `json:"text"`
	Type  string `json:"type"`
}

// Prompt is a JSON-serializable AI prompt template.
type Prompt struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// UsageStat is a single prompt's usage count for the stats panel.
type UsageStat struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	UsageCount int    `json:"usageCount"`
}

// UsageStats summarizes AI prompt usage.
type UsageStats struct {
	TotalUsage    int         `json:"totalUsage"`
	UniquePrompts int         `json:"uniquePrompts"`
	TopPrompts    []UsageStat `json:"topPrompts"`
}

// ConfigInfo is a read-only snapshot of the effective configuration, shown by
// the desktop's ":config" panel.
type ConfigInfo struct {
	ConfigPath   string `json:"configPath"`
	LogPath      string `json:"logPath"`
	Account      string `json:"account"`
	LLMProvider  string `json:"llmProvider"`
	LLMModel     string `json:"llmModel"`
	Theme        string `json:"theme"`
	ObsidianOn   bool   `json:"obsidianOn"`
	SlackOn      bool   `json:"slackOn"`
	AutoRefresh  bool   `json:"autoRefresh"`
	DownloadPath string `json:"downloadPath"`
}

// Invite describes a calendar invitation detected in a message, so the reader
// can offer Accept / Tentative / Decline.
type Invite struct {
	IsInvite  bool   `json:"isInvite"`
	UID       string `json:"uid"`
	Summary   string `json:"summary"`
	Organizer string `json:"organizer"`
	DtStart   string `json:"dtStart"`
	DtEnd     string `json:"dtEnd"`
}

// AutoRefreshSettings tells the frontend whether to poll the inbox for new mail
// and how often, mirroring the user's auto_refresh config.
type AutoRefreshSettings struct {
	Enabled         bool `json:"enabled"`
	IntervalSeconds int  `json:"intervalSeconds"`
}

// AnalyzerRule is a free-text preference rule for the inbox analyzer.
type AnalyzerRule struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

// PromptDetail carries the full prompt template, including its editable text.
type PromptDetail struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Text        string `json:"text"`
}

// Attachment is a JSON-serializable message attachment. It mirrors the fields of
// services.AttachmentInfo but lives in this public package so the nested Wails
// module (which cannot import internal/...) can consume it.
type Attachment struct {
	AttachmentID string `json:"attachmentId"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	Type         string `json:"type"`
	Inline       bool   `json:"inline"`
}
