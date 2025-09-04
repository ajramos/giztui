package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ajramos/giztui/internal/db"
	"github.com/ajramos/giztui/internal/gmail"
	gmailapi "google.golang.org/api/gmail/v1"
)

// threadMessageCache represents cached thread messages with TTL
type threadMessageCache struct {
	messages  []*gmailapi.Message
	timestamp time.Time
	ttl       time.Duration
}

func (c *threadMessageCache) isExpired() bool {
	return time.Since(c.timestamp) > c.ttl
}

// ThreadServiceImpl implements ThreadService
type ThreadServiceImpl struct {
	gmailClient *gmail.Client
	dbStore     *db.Store
	aiService   AIService

	// In-memory state tracking when database is not available
	memoryState sync.Map // key: "accountEmail:threadID" -> value: bool (expanded state)

	// Message cache for improved performance
	messageCache sync.Map // key: "threadID" -> value: *threadMessageCache
}

// NewThreadService creates a new thread service
func NewThreadService(gmailClient *gmail.Client, dbStore *db.Store, aiService AIService) *ThreadServiceImpl {
	return &ThreadServiceImpl{
		gmailClient: gmailClient,
		dbStore:     dbStore,
		aiService:   aiService,
	}
}

// ClearMessageCache removes expired entries from the message cache
func (s *ThreadServiceImpl) ClearMessageCache() {
	s.messageCache.Range(func(key, value interface{}) bool {
		if cache, ok := value.(*threadMessageCache); ok && cache.isExpired() {
			s.messageCache.Delete(key)
		}
		return true
	})
}

// GetThreads retrieves conversation threads from Gmail
func (s *ThreadServiceImpl) GetThreads(ctx context.Context, opts ThreadQueryOptions) (*ThreadPage, error) {
	if s.gmailClient == nil || s.gmailClient.Service == nil {
		return nil, fmt.Errorf("gmail client not initialized")
	}

	// Build Gmail threads query
	call := s.gmailClient.Service.Users.Threads.List("me")

	if opts.MaxResults > 0 {
		call = call.MaxResults(opts.MaxResults)
	}
	if opts.PageToken != "" {
		call = call.PageToken(opts.PageToken)
	}
	if opts.Query != "" {
		call = call.Q(opts.Query)
	}
	if len(opts.LabelIDs) > 0 {
		call = call.LabelIds(opts.LabelIDs...)
	}

	threadsResult, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch threads: %w", err)
	}

	// Convert Gmail threads to ThreadInfo structures
	// Note: Threads.List only returns minimal thread data, we need to fetch full thread details
	var threadInfos []*ThreadInfo
	for _, thread := range threadsResult.Threads {
		// Get thread data with minimal format for faster loading
		fullThread, err := s.gmailClient.Service.Users.Threads.Get("me", thread.Id).Format("metadata").Do()
		if err != nil {
			// Skip thread on error and continue processing
			continue
		}

		threadInfo, err := s.buildThreadInfo(ctx, fullThread)
		if err != nil {
			// Skip thread on error and continue processing
			continue
		}
		threadInfos = append(threadInfos, threadInfo)
	}

	return &ThreadPage{
		Threads:       threadInfos,
		NextPageToken: threadsResult.NextPageToken,
		TotalCount:    int(threadsResult.ResultSizeEstimate),
	}, nil
}

// GetThreadMessages retrieves all messages in a thread with smart caching
func (s *ThreadServiceImpl) GetThreadMessages(ctx context.Context, threadID string, opts MessageQueryOptions) ([]*gmailapi.Message, error) {
	if threadID == "" {
		return nil, fmt.Errorf("threadID cannot be empty")
	}

	// Check cache first (5 minute TTL for thread messages)
	if cached, ok := s.messageCache.Load(threadID); ok {
		if cache, ok := cached.(*threadMessageCache); ok && !cache.isExpired() {
			// Return cached messages, sorted as requested
			messages := make([]*gmailapi.Message, len(cache.messages))
			copy(messages, cache.messages)

			// Apply sorting
			s.sortMessages(messages, opts.SortOrder)
			return messages, nil
		}
		// Cache expired, remove it
		s.messageCache.Delete(threadID)
	}

	// Cache miss or expired - fetch from Gmail API
	thread, err := s.gmailClient.Service.Users.Threads.Get("me", threadID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread messages: %w", err)
	}

	messages := thread.Messages

	// Cache the messages for future use (5 minute TTL)
	cache := &threadMessageCache{
		messages:  make([]*gmailapi.Message, len(messages)),
		timestamp: time.Now(),
		ttl:       5 * time.Minute,
	}
	copy(cache.messages, messages)
	s.messageCache.Store(threadID, cache)

	// Apply sorting to the returned messages
	s.sortMessages(messages, opts.SortOrder)
	return messages, nil
}

// sortMessages applies the requested sort order to messages
func (s *ThreadServiceImpl) sortMessages(messages []*gmailapi.Message, sortOrder string) {
	switch sortOrder {
	case "asc":
		sort.Slice(messages, func(i, j int) bool {
			return extractInternalDate(messages[i]) < extractInternalDate(messages[j])
		})
	case "desc":
		sort.Slice(messages, func(i, j int) bool {
			return extractInternalDate(messages[i]) > extractInternalDate(messages[j])
		})
	}
}

// GetThreadInfo retrieves metadata about a specific thread
func (s *ThreadServiceImpl) GetThreadInfo(ctx context.Context, threadID string) (*ThreadInfo, error) {
	if threadID == "" {
		return nil, fmt.Errorf("threadID cannot be empty")
	}

	thread, err := s.gmailClient.Service.Users.Threads.Get("me", threadID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread info: %w", err)
	}

	return s.buildThreadInfo(ctx, thread)
}

// SetThreadExpanded sets the expansion state of a thread for a user
func (s *ThreadServiceImpl) SetThreadExpanded(ctx context.Context, accountEmail, threadID string, expanded bool) error {
	if s.dbStore == nil {
		// Use in-memory state when database is not available
		key := accountEmail + ":" + threadID
		s.memoryState.Store(key, expanded)
		return nil
	}

	query := `INSERT OR REPLACE INTO thread_state (account_email, thread_id, is_expanded, last_updated)
			  VALUES (?, ?, ?, ?)`

	_, err := s.dbStore.DB().Exec(query, accountEmail, threadID, expanded, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set thread expansion state: %w", err)
	}

	return nil
}

// IsThreadExpanded checks if a thread is expanded for a user
func (s *ThreadServiceImpl) IsThreadExpanded(ctx context.Context, accountEmail, threadID string) (bool, error) {
	if s.dbStore == nil {
		// Use in-memory state when database is not available
		key := accountEmail + ":" + threadID
		if value, exists := s.memoryState.Load(key); exists {
			return value.(bool), nil
		}
		// Default to collapsed if not found in memory
		return false, nil
	}

	var expanded bool
	query := `SELECT is_expanded FROM thread_state WHERE account_email = ? AND thread_id = ?`

	row := s.dbStore.DB().QueryRow(query, accountEmail, threadID)
	err := row.Scan(&expanded)
	if err != nil {
		// If no record exists, default to collapsed (false)
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check thread expansion state: %w", err)
	}

	return expanded, nil
}

// ExpandAllThreads expands all threads for a user
func (s *ThreadServiceImpl) ExpandAllThreads(ctx context.Context, accountEmail string) error {
	if s.dbStore == nil {
		// Thread state persistence not available without database
		return nil
	}

	query := `UPDATE thread_state SET is_expanded = true, last_updated = ? WHERE account_email = ?`

	_, err := s.dbStore.DB().Exec(query, time.Now(), accountEmail)
	if err != nil {
		return fmt.Errorf("failed to expand all threads: %w", err)
	}

	return nil
}

// CollapseAllThreads collapses all threads for a user
func (s *ThreadServiceImpl) CollapseAllThreads(ctx context.Context, accountEmail string) error {
	if s.dbStore == nil {
		// Thread state persistence not available without database
		return nil
	}

	query := `UPDATE thread_state SET is_expanded = false, last_updated = ? WHERE account_email = ?`

	_, err := s.dbStore.DB().Exec(query, time.Now(), accountEmail)
	if err != nil {
		return fmt.Errorf("failed to collapse all threads: %w", err)
	}

	return nil
}

// GenerateThreadSummary generates an AI summary of a thread
func (s *ThreadServiceImpl) GenerateThreadSummary(ctx context.Context, threadID string, options ThreadSummaryOptions) (*ThreadSummaryResult, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	// Check cache first if enabled
	if options.UseCache && !options.ForceRegenerate {
		if cached, err := s.GetCachedThreadSummary(ctx, options.AccountEmail, threadID); err == nil && cached != nil {
			return cached, nil
		}
	}

	// Get thread messages
	messages, err := s.GetThreadMessages(ctx, threadID, MessageQueryOptions{
		Format:    "full",
		SortOrder: "asc",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get thread messages for summary: %w", err)
	}

	// Build combined content for AI processing
	var contentBuilder strings.Builder
	contentBuilder.WriteString("---START THREAD---\n")

	for i, msg := range messages {
		contentBuilder.WriteString(fmt.Sprintf("---START MESSAGE %d---\n", i+1))

		// Extract message content
		plainText := gmail.ExtractPlainText(msg)
		if plainText != "" {
			contentBuilder.WriteString(plainText)
		} else {
			contentBuilder.WriteString("[No content available]")
		}

		contentBuilder.WriteString(fmt.Sprintf("\n---END MESSAGE %d---\n", i+1))
	}
	contentBuilder.WriteString("---END THREAD---\n")

	// Generate summary using AI service
	start := time.Now()
	summaryOptions := SummaryOptions{
		MaxLength:    options.MaxLength,
		Language:     options.Language,
		UseCache:     false, // We handle caching at thread level
		AccountEmail: options.AccountEmail,
	}

	result, err := s.aiService.GenerateSummary(ctx, contentBuilder.String(), summaryOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thread summary: %w", err)
	}

	threadSummary := &ThreadSummaryResult{
		ThreadID:     threadID,
		Summary:      result.Summary,
		SummaryType:  options.SummaryType,
		FromCache:    false,
		Language:     result.Language,
		Duration:     time.Since(start),
		MessageCount: len(messages),
		CreatedAt:    time.Now(),
	}

	// Cache the result
	if options.UseCache {
		go s.cacheThreadSummary(ctx, options.AccountEmail, threadSummary)
	}

	return threadSummary, nil
}

// GenerateThreadSummaryStream generates an AI summary with streaming
func (s *ThreadServiceImpl) GenerateThreadSummaryStream(ctx context.Context, threadID string, options ThreadSummaryOptions, onToken func(string)) (*ThreadSummaryResult, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	// Check cache first if enabled
	if options.UseCache && !options.ForceRegenerate {
		if cached, err := s.GetCachedThreadSummary(ctx, options.AccountEmail, threadID); err == nil && cached != nil {
			// For cached results, we simulate streaming by sending the full summary
			if onToken != nil {
				onToken(cached.Summary)
			}
			return cached, nil
		}
	}

	// Get thread messages
	messages, err := s.GetThreadMessages(ctx, threadID, MessageQueryOptions{
		Format:    "full",
		SortOrder: "asc",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get thread messages for summary: %w", err)
	}

	// Build combined content
	var contentBuilder strings.Builder
	contentBuilder.WriteString("---START THREAD---\n")

	for i, msg := range messages {
		contentBuilder.WriteString(fmt.Sprintf("---START MESSAGE %d---\n", i+1))
		plainText := gmail.ExtractPlainText(msg)
		if plainText != "" {
			contentBuilder.WriteString(plainText)
		}
		contentBuilder.WriteString(fmt.Sprintf("\n---END MESSAGE %d---\n", i+1))
	}
	contentBuilder.WriteString("---END THREAD---\n")

	// Generate streaming summary
	start := time.Now()
	summaryOptions := SummaryOptions{
		MaxLength:     options.MaxLength,
		Language:      options.Language,
		StreamEnabled: options.StreamEnabled,
		UseCache:      false,
		AccountEmail:  options.AccountEmail,
	}

	result, err := s.aiService.GenerateSummaryStream(ctx, contentBuilder.String(), summaryOptions, onToken)
	if err != nil {
		return nil, fmt.Errorf("failed to generate streaming thread summary: %w", err)
	}

	threadSummary := &ThreadSummaryResult{
		ThreadID:     threadID,
		Summary:      result.Summary,
		SummaryType:  options.SummaryType,
		FromCache:    false,
		Language:     result.Language,
		Duration:     time.Since(start),
		MessageCount: len(messages),
		CreatedAt:    time.Now(),
	}

	// Cache the result
	if options.UseCache {
		go s.cacheThreadSummary(ctx, options.AccountEmail, threadSummary)
	}

	return threadSummary, nil
}

// GetCachedThreadSummary retrieves a cached thread summary
func (s *ThreadServiceImpl) GetCachedThreadSummary(ctx context.Context, accountEmail, threadID string) (*ThreadSummaryResult, error) {
	if s.dbStore == nil {
		// No cache available without database
		return nil, fmt.Errorf("cache not available")
	}

	query := `SELECT summary, summary_type, language, message_count, cached_at
			  FROM thread_summary_cache
			  WHERE account_email = ? AND thread_id = ?`

	var summary, summaryType, language string
	var messageCount int
	var cachedAt time.Time

	row := s.dbStore.DB().QueryRow(query, accountEmail, threadID)
	err := row.Scan(&summary, &summaryType, &language, &messageCount, &cachedAt)
	if err != nil {
		return nil, fmt.Errorf("thread summary not found in cache: %w", err)
	}

	return &ThreadSummaryResult{
		ThreadID:     threadID,
		Summary:      summary,
		SummaryType:  summaryType,
		FromCache:    true,
		Language:     language,
		Duration:     0, // Cached result
		MessageCount: messageCount,
		CreatedAt:    cachedAt,
	}, nil
}

// SearchWithinThread searches for content within a specific thread
func (s *ThreadServiceImpl) SearchWithinThread(ctx context.Context, threadID, query string) (*ThreadSearchResult, error) {
	start := time.Now()

	messages, err := s.GetThreadMessages(ctx, threadID, MessageQueryOptions{
		Format: "full",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get thread messages for search: %w", err)
	}

	var matches []ThreadMatch
	queryLower := strings.ToLower(query)

	for _, msg := range messages {
		plainText := gmail.ExtractPlainText(msg)
		plainTextLower := strings.ToLower(plainText)

		// Find all occurrences of the query in this message
		startPos := 0
		for {
			pos := strings.Index(plainTextLower[startPos:], queryLower)
			if pos == -1 {
				break
			}

			actualPos := startPos + pos

			// Extract context around the match
			contextStart := actualPos - 50
			if contextStart < 0 {
				contextStart = 0
			}
			contextEnd := actualPos + len(query) + 50
			if contextEnd > len(plainText) {
				contextEnd = len(plainText)
			}

			context := plainText[contextStart:contextEnd]
			matchText := plainText[actualPos : actualPos+len(query)]

			matches = append(matches, ThreadMatch{
				MessageID: msg.Id,
				Position:  actualPos,
				Context:   context,
				MatchText: matchText,
			})

			startPos = actualPos + 1
		}
	}

	return &ThreadSearchResult{
		ThreadID:   threadID,
		Query:      query,
		Matches:    matches,
		MatchCount: len(matches),
		Duration:   time.Since(start),
	}, nil
}

// GetNextThread and GetPreviousThread would need thread ordering logic
func (s *ThreadServiceImpl) GetNextThread(ctx context.Context, currentThreadID string) (string, error) {
	// Implementation would depend on how threads are ordered in the UI
	// For now, return empty to indicate no next thread
	return "", fmt.Errorf("next thread navigation not implemented")
}

func (s *ThreadServiceImpl) GetPreviousThread(ctx context.Context, currentThreadID string) (string, error) {
	// Implementation would depend on how threads are ordered in the UI
	// For now, return empty to indicate no previous thread
	return "", fmt.Errorf("previous thread navigation not implemented")
}

// GetThreadsByLabel retrieves threads filtered by label
func (s *ThreadServiceImpl) GetThreadsByLabel(ctx context.Context, labelID string, opts ThreadQueryOptions) (*ThreadPage, error) {
	opts.LabelIDs = []string{labelID}
	return s.GetThreads(ctx, opts)
}

// GetUnreadThreads retrieves threads with unread messages
func (s *ThreadServiceImpl) GetUnreadThreads(ctx context.Context, opts ThreadQueryOptions) (*ThreadPage, error) {
	opts.Query = "is:unread " + opts.Query
	return s.GetThreads(ctx, opts)
}

// BulkExpandThreads expands multiple threads
func (s *ThreadServiceImpl) BulkExpandThreads(ctx context.Context, accountEmail string, threadIDs []string) error {
	for _, threadID := range threadIDs {
		if err := s.SetThreadExpanded(ctx, accountEmail, threadID, true); err != nil {
			return fmt.Errorf("failed to expand thread %s: %w", threadID, err)
		}
	}
	return nil
}

// BulkCollapseThreads collapses multiple threads
func (s *ThreadServiceImpl) BulkCollapseThreads(ctx context.Context, accountEmail string, threadIDs []string) error {
	for _, threadID := range threadIDs {
		if err := s.SetThreadExpanded(ctx, accountEmail, threadID, false); err != nil {
			return fmt.Errorf("failed to collapse thread %s: %w", threadID, err)
		}
	}
	return nil
}

// Helper methods

// buildThreadInfo constructs ThreadInfo from Gmail thread data
func (s *ThreadServiceImpl) buildThreadInfo(ctx context.Context, thread *gmailapi.Thread) (*ThreadInfo, error) {
	if len(thread.Messages) == 0 {
		return nil, fmt.Errorf("thread has no messages")
	}

	// Use the first message as the root
	rootMsg := thread.Messages[0]

	// Extract participants
	participants := make(map[string]bool)
	var labels []string
	var hasAttachment bool
	var unreadCount int
	var latestDate time.Time

	for _, msg := range thread.Messages {
		// Collect participants
		if from := extractHeader(msg, "From"); from != "" {
			participants[from] = true
		}
		if to := extractHeader(msg, "To"); to != "" {
			for _, addr := range strings.Split(to, ",") {
				participants[strings.TrimSpace(addr)] = true
			}
		}

		// Check for attachments
		if hasAttachmentInMessage(msg) {
			hasAttachment = true
		}

		// Count unread messages
		for _, labelID := range msg.LabelIds {
			if labelID == "UNREAD" {
				unreadCount++
				break
			}
		}

		// Track latest date
		msgDate := time.Unix(0, msg.InternalDate*int64(time.Millisecond))
		if msgDate.After(latestDate) {
			latestDate = msgDate
		}

		// Collect labels from the root message
		if msg.Id == rootMsg.Id {
			labels = msg.LabelIds
		}
	}

	// Convert participants map to slice, ensuring root message sender is first
	var participantList []string
	rootSender := extractHeader(rootMsg, "From")
	if rootSender != "" {
		participantList = append(participantList, rootSender)
	}
	// Add other participants (excluding the root sender to avoid duplicates)
	for participant := range participants {
		if participant != rootSender {
			participantList = append(participantList, participant)
		}
	}

	return &ThreadInfo{
		ThreadID:      thread.Id,
		MessageCount:  len(thread.Messages),
		UnreadCount:   unreadCount,
		Participants:  participantList,
		Subject:       extractHeader(rootMsg, "Subject"),
		LatestDate:    latestDate,
		HasAttachment: hasAttachment,
		Labels:        labels,
		IsExpanded:    false, // Will be set by UI based on user preferences
		RootMessageID: rootMsg.Id,
	}, nil
}

// cacheThreadSummary caches a thread summary result
func (s *ThreadServiceImpl) cacheThreadSummary(ctx context.Context, accountEmail string, result *ThreadSummaryResult) {
	if s.dbStore == nil {
		// Caching not available without database
		return
	}

	query := `INSERT OR REPLACE INTO thread_summary_cache
			  (account_email, thread_id, summary, summary_type, language, message_count, cached_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.dbStore.DB().Exec(query, accountEmail, result.ThreadID, result.Summary,
		result.SummaryType, result.Language, result.MessageCount, result.CreatedAt)
	if err != nil {
		// Ignore cache errors and continue operation
		return
	}
}

// Helper functions

func extractHeader(msg *gmailapi.Message, headerName string) string {
	if msg.Payload == nil || msg.Payload.Headers == nil {
		return ""
	}

	for _, header := range msg.Payload.Headers {
		if header.Name == headerName {
			return header.Value
		}
	}

	return ""
}

func extractInternalDate(msg *gmailapi.Message) int64 {
	return msg.InternalDate
}

func hasAttachmentInMessage(msg *gmailapi.Message) bool {
	if msg.Payload == nil {
		return false
	}

	// Check if message has parts with attachments
	return hasAttachmentInPart(msg.Payload)
}

func hasAttachmentInPart(part *gmailapi.MessagePart) bool {
	if part.Body != nil && part.Body.AttachmentId != "" {
		return true
	}

	for _, subpart := range part.Parts {
		if hasAttachmentInPart(subpart) {
			return true
		}
	}

	return false
}
