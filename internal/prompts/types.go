package prompts

// PromptTemplate represents a prompt template
type PromptTemplate struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PromptText  string `json:"prompt_text"`
	Category    string `json:"category"`
	CreatedAt   int64  `json:"created_at"`
	IsFavorite  bool   `json:"is_favorite"`
	UsageCount  int    `json:"usage_count"`
}

// PromptResult represents a prompt execution result
type PromptResult struct {
	ID           int    `json:"id"`
	AccountEmail string `json:"account_email"`
	MessageID    string `json:"message_id"`
	PromptID     int    `json:"prompt_id"`
	ResultText   string `json:"result_text"`
	CreatedAt    int64  `json:"created_at"`
}
