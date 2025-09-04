package services

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ajramos/giztui/internal/gmail"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// MessagePreloaderImpl implements the MessagePreloader interface
type MessagePreloaderImpl struct {
	client   *gmail.Client
	config   *PreloadConfig
	logger   *log.Logger
	configMu sync.RWMutex

	// Cache management
	messageCache   map[string]*CacheItem
	pageCache      map[string]*PageCacheItem
	cacheMu        sync.RWMutex
	evictionList   *list.List
	maxMemoryBytes int64
	currentMemory  int64

	// Background workers
	workerPool   chan struct{}
	taskQueue    chan *preloadTask
	activeQueue  chan *preloadTask
	activeTasks  int32
	shutdown     chan struct{}
	shutdownOnce sync.Once

	// Statistics
	stats   PreloadStatistics
	statsMu sync.RWMutex
}

// CacheItem represents a cached message with metadata
type CacheItem struct {
	Message    *gmail_v1.Message
	Size       int64
	Timestamp  time.Time
	AccessTime time.Time
	element    *list.Element // for LRU eviction
}

// PageCacheItem represents a cached page of messages with next token
type PageCacheItem struct {
	Messages      []*gmail_v1.Message
	NextPageToken string
	Timestamp     time.Time
	Size          int64
}

// preloadTask represents a background preloading task
type preloadTask struct {
	Type       string // "next_page" or "adjacent"
	MessageIDs []string
	PageToken  string
	Query      string
	MaxResults int64
	Priority   int // 1=high, 2=normal, 3=low
	CreatedAt  time.Time
	Context    context.Context
}

// NewMessagePreloader creates a new MessagePreloader instance
func NewMessagePreloader(client *gmail.Client, config *PreloadConfig, logger *log.Logger) *MessagePreloaderImpl {
	if config == nil {
		config = DefaultPreloadConfig()
	}

	// Validate and set defaults
	if config.BackgroundWorkers <= 0 {
		config.BackgroundWorkers = 3
	}
	if config.CacheSizeMB <= 0 {
		config.CacheSizeMB = 50
	}
	if config.NextPageThreshold <= 0 || config.NextPageThreshold >= 1 {
		config.NextPageThreshold = 0.7
	}
	if config.AdjacentCount <= 0 {
		config.AdjacentCount = 3
	}

	p := &MessagePreloaderImpl{
		client:         client,
		config:         config,
		logger:         logger,
		messageCache:   make(map[string]*CacheItem),
		pageCache:      make(map[string]*PageCacheItem),
		evictionList:   list.New(),
		maxMemoryBytes: int64(config.CacheSizeMB) * 1024 * 1024, // Convert MB to bytes
		workerPool:     make(chan struct{}, config.BackgroundWorkers),
		taskQueue:      make(chan *preloadTask, 100), // Buffer for tasks
		activeQueue:    make(chan *preloadTask, config.BackgroundWorkers),
		shutdown:       make(chan struct{}),
		stats: PreloadStatistics{
			NextPageRequests:     0,
			AdjacentRequests:     0,
			PreloadHits:          0,
			PreloadMisses:        0,
			CacheHitRate:         0.0,
			AveragePreloadTime:   0,
			TotalDataPreloadedMB: 0.0,
		},
	}

	// Initialize worker pool
	for i := 0; i < config.BackgroundWorkers; i++ {
		p.workerPool <- struct{}{}
	}

	// Start background workers
	go p.startWorkers()

	return p
}

// DefaultPreloadConfig returns the default preloading configuration
func DefaultPreloadConfig() *PreloadConfig {
	return &PreloadConfig{
		Enabled:                true,
		NextPageEnabled:        true,
		NextPageThreshold:      0.7,
		NextPageMaxPages:       2,
		AdjacentEnabled:        true,
		AdjacentCount:          3,
		BackgroundWorkers:      3,
		CacheSizeMB:            50,
		APIQuotaReservePercent: 20,
	}
}

// PreloadNextPage preloads the next page of messages in background
func (p *MessagePreloaderImpl) PreloadNextPage(ctx context.Context, currentPageToken string, query string, maxResults int64) error {
	if !p.IsEnabled() || !p.IsNextPageEnabled() {
		return nil // Preloading disabled
	}

	// Check if we have a valid page token to preload from
	if currentPageToken == "" {
		return fmt.Errorf("no next page token available")
	}

	// Check if already cached
	if _, exists := p.GetCachedMessages(ctx, currentPageToken); exists {
		return nil // Already cached
	}

	// Create preload task
	task := &preloadTask{
		Type:       "next_page",
		PageToken:  currentPageToken,
		Query:      query,
		MaxResults: maxResults,
		Priority:   2, // Normal priority
		CreatedAt:  time.Now(),
		Context:    ctx,
	}

	// Queue task for background processing
	select {
	case p.taskQueue <- task:
		p.statsMu.Lock()
		p.stats.NextPageRequests++
		p.statsMu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue full, skip this preload request
		return fmt.Errorf("preload queue full")
	}
}

// PreloadAdjacentMessages preloads messages around the current selection
func (p *MessagePreloaderImpl) PreloadAdjacentMessages(ctx context.Context, currentMessageID string, messageIDs []string) error {
	if !p.IsEnabled() || !p.IsAdjacentEnabled() {
		return nil // Preloading disabled
	}

	// Find current message index
	currentIndex := -1
	for i, id := range messageIDs {
		if id == currentMessageID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return fmt.Errorf("current message not found in list")
	}

	// Calculate adjacent message IDs to preload
	adjacentIDs := []string{}
	adjacentCount := p.config.AdjacentCount

	// Add messages before current
	start := maxInt(0, currentIndex-adjacentCount/2)
	// Add messages after current
	end := minInt(len(messageIDs), currentIndex+adjacentCount/2+1)

	for i := start; i < end; i++ {
		if i != currentIndex { // Don't include current message
			// Check if already cached
			if _, exists := p.GetCachedMessage(ctx, messageIDs[i]); !exists {
				adjacentIDs = append(adjacentIDs, messageIDs[i])
			}
		}
	}

	if len(adjacentIDs) == 0 {
		return nil // All adjacent messages already cached
	}

	// Create preload task
	task := &preloadTask{
		Type:       "adjacent",
		MessageIDs: adjacentIDs,
		Priority:   1, // High priority for adjacent messages
		CreatedAt:  time.Now(),
		Context:    ctx,
	}

	// Queue task for background processing
	select {
	case p.taskQueue <- task:
		p.statsMu.Lock()
		p.stats.AdjacentRequests++
		p.statsMu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue full, skip this preload request
		return fmt.Errorf("preload queue full")
	}
}

// GetCachedMessages retrieves cached messages for a page token
func (p *MessagePreloaderImpl) GetCachedMessages(ctx context.Context, pageToken string) ([]*gmail_v1.Message, bool) {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	pageItem, exists := p.pageCache[pageToken]
	if exists {
		p.statsMu.Lock()
		p.stats.PreloadHits++
		p.statsMu.Unlock()
		return pageItem.Messages, true
	}

	p.statsMu.Lock()
	p.stats.PreloadMisses++
	p.statsMu.Unlock()
	return nil, false
}

// GetCachedMessagesWithToken retrieves cached messages and next page token for a page token
func (p *MessagePreloaderImpl) GetCachedMessagesWithToken(ctx context.Context, pageToken string) ([]*gmail_v1.Message, string, bool) {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	pageItem, exists := p.pageCache[pageToken]
	if exists {
		p.statsMu.Lock()
		p.stats.PreloadHits++
		p.statsMu.Unlock()
		return pageItem.Messages, pageItem.NextPageToken, true
	}

	p.statsMu.Lock()
	p.stats.PreloadMisses++
	p.statsMu.Unlock()
	return nil, "", false
}

// GetCachedMessage retrieves a cached individual message
func (p *MessagePreloaderImpl) GetCachedMessage(ctx context.Context, messageID string) (*gmail_v1.Message, bool) {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	item, exists := p.messageCache[messageID]
	if exists {
		// Update access time for LRU
		item.AccessTime = time.Now()
		// Move to front of eviction list
		p.evictionList.MoveToFront(item.element)

		p.statsMu.Lock()
		p.stats.PreloadHits++
		p.statsMu.Unlock()
		return item.Message, true
	}

	p.statsMu.Lock()
	p.stats.PreloadMisses++
	p.statsMu.Unlock()
	return nil, false
}

// ClearCache clears all cached data
func (p *MessagePreloaderImpl) ClearCache(ctx context.Context) error {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	// Clear all caches
	p.messageCache = make(map[string]*CacheItem)
	p.pageCache = make(map[string]*PageCacheItem)
	p.evictionList = list.New()
	p.currentMemory = 0

	return nil
}

// Configuration methods
func (p *MessagePreloaderImpl) IsEnabled() bool {
	p.configMu.RLock()
	defer p.configMu.RUnlock()
	return p.config.Enabled
}

func (p *MessagePreloaderImpl) IsNextPageEnabled() bool {
	p.configMu.RLock()
	defer p.configMu.RUnlock()
	return p.config.Enabled && p.config.NextPageEnabled
}

func (p *MessagePreloaderImpl) IsAdjacentEnabled() bool {
	p.configMu.RLock()
	defer p.configMu.RUnlock()
	return p.config.Enabled && p.config.AdjacentEnabled
}

func (p *MessagePreloaderImpl) UpdateConfig(config *PreloadConfig) error {
	p.configMu.Lock()
	defer p.configMu.Unlock()

	// Validate config
	if config.BackgroundWorkers <= 0 || config.BackgroundWorkers > 15 {
		return fmt.Errorf("invalid background workers count: %d", config.BackgroundWorkers)
	}
	if config.CacheSizeMB <= 0 || config.CacheSizeMB > 500 {
		return fmt.Errorf("invalid cache size: %d MB", config.CacheSizeMB)
	}

	p.config = config
	p.maxMemoryBytes = int64(config.CacheSizeMB) * 1024 * 1024

	// If cache size reduced, trigger eviction
	if p.currentMemory > p.maxMemoryBytes {
		go p.evictExcessMemory()
	}

	return nil
}

func (p *MessagePreloaderImpl) GetStatus() *PreloadStatus {
	p.configMu.RLock()
	p.cacheMu.RLock()
	p.statsMu.RLock()
	defer p.configMu.RUnlock()
	defer p.cacheMu.RUnlock()
	defer p.statsMu.RUnlock()

	// Calculate hit rate
	totalRequests := p.stats.PreloadHits + p.stats.PreloadMisses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(p.stats.PreloadHits) / float64(totalRequests)
	}

	return &PreloadStatus{
		Enabled:              p.config.Enabled,
		NextPageEnabled:      p.config.NextPageEnabled,
		AdjacentEnabled:      p.config.AdjacentEnabled,
		CacheSize:            len(p.messageCache),
		CacheMemoryUsageMB:   float64(p.currentMemory) / (1024 * 1024),
		ActivePreloadTasks:   int(atomic.LoadInt32(&p.activeTasks)),
		LastPreloadActivity:  time.Now(), // Would track actual last activity
		TotalPreloadRequests: p.stats.NextPageRequests + p.stats.AdjacentRequests,
		PreloadHits:          p.stats.PreloadHits,
		PreloadMisses:        p.stats.PreloadMisses,
		BackgroundWorkers:    p.config.BackgroundWorkers,
		Config:               p.config,
		Statistics: &PreloadStatistics{
			NextPageRequests:     p.stats.NextPageRequests,
			AdjacentRequests:     p.stats.AdjacentRequests,
			CacheHitRate:         hitRate,
			AveragePreloadTime:   p.stats.AveragePreloadTime,
			TotalDataPreloadedMB: p.stats.TotalDataPreloadedMB,
		},
	}
}

// Private methods

// startWorkers starts the background worker goroutines
func (p *MessagePreloaderImpl) startWorkers() {
	for {
		select {
		case task := <-p.taskQueue:
			// Try to get a worker from the pool
			select {
			case <-p.workerPool:
				// Got a worker, process the task
				atomic.AddInt32(&p.activeTasks, 1)
				go p.processTask(task)
			case <-p.shutdown:
				return
			default:
				// No workers available, put task back or drop it based on priority
				if task.Priority <= 1 { // High priority tasks
					select {
					case p.activeQueue <- task:
						// Queued for retry
					default:
						// Drop task if active queue is full
					}
				}
				// Drop normal/low priority tasks if no workers available
			}
		case <-p.shutdown:
			return
		}
	}
}

// processTask processes a background preload task
func (p *MessagePreloaderImpl) processTask(task *preloadTask) {
	defer func() {
		// Return worker to pool
		p.workerPool <- struct{}{}
		atomic.AddInt32(&p.activeTasks, -1)
	}()

	startTime := time.Now()

	switch task.Type {
	case "next_page":
		p.processNextPageTask(task)
	case "adjacent":
		p.processAdjacentTask(task)
	}

	// Update statistics
	duration := time.Since(startTime)
	p.statsMu.Lock()
	if p.stats.AveragePreloadTime == 0 {
		p.stats.AveragePreloadTime = duration
	} else {
		// Moving average
		p.stats.AveragePreloadTime = (p.stats.AveragePreloadTime + duration) / 2
	}
	p.statsMu.Unlock()
}

// processNextPageTask handles next page preloading
func (p *MessagePreloaderImpl) processNextPageTask(task *preloadTask) {
	_, cancel := context.WithTimeout(task.Context, 30*time.Second)
	defer cancel()

	var messages []*gmail_v1.Message
	var nextPageToken string
	var err error

	// Use appropriate API method based on whether we have a query
	if task.Query != "" {
		// For search queries, use search API
		messages, nextPageToken, err = p.client.SearchMessagesPage(task.Query, task.MaxResults, task.PageToken)
	} else {
		// For inbox listing, use list API
		messages, nextPageToken, err = p.client.ListMessagesPage(task.MaxResults, task.PageToken)
	}

	if err != nil {
		// Log error but don't fail the preload operation
		if p.logger != nil {
			p.logger.Printf("[PRELOAD ERROR] Failed to fetch next page for token='%s': %v", task.PageToken, err)
		}
		return
	}

	// If we got messages, fetch their metadata in parallel using the existing parallel system
	if len(messages) > 0 {
		messageIDs := make([]string, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.Id
		}

		// Use the existing parallel metadata fetching (optimized for lists)
		detailedMessages, err := p.client.GetMessagesMetadataParallel(messageIDs, minInt(p.config.BackgroundWorkers, len(messageIDs)))
		if err != nil {
			// Log error but don't fail
			return
		}

		// Store in page cache with next token
		p.cacheMu.Lock()
		defer p.cacheMu.Unlock()

		// Cache the detailed messages for this page along with next token
		// Use a cache key that includes the query for differentiation
		cacheKey := task.PageToken
		if task.Query != "" {
			cacheKey = task.Query + ":" + task.PageToken
		}

		// Calculate cache size for this page
		var pageSize int64
		for _, msg := range detailedMessages {
			pageSize += p.estimateMessageSize(msg)
		}

		p.pageCache[cacheKey] = &PageCacheItem{
			Messages:      detailedMessages,
			NextPageToken: nextPageToken,
			Timestamp:     time.Now(),
			Size:          pageSize,
		}

		// Update statistics
		p.statsMu.Lock()
		p.stats.TotalDataPreloadedMB += float64(pageSize) / (1024 * 1024)
		p.statsMu.Unlock()

		// Log successful completion
		if p.logger != nil {
			queryStr := ""
			if task.Query != "" {
				queryStr = fmt.Sprintf(" (query: '%s')", task.Query)
			}
			p.logger.Printf("✅ PRELOAD COMPLETE: Cached %d messages%s for token='%s', nextToken='%s', size=%.2fMB",
				len(detailedMessages), queryStr, task.PageToken, nextPageToken, float64(pageSize)/(1024*1024))
		}
	}
}

// processAdjacentTask handles adjacent message preloading
func (p *MessagePreloaderImpl) processAdjacentTask(task *preloadTask) {
	ctx, cancel := context.WithTimeout(task.Context, 30*time.Second)
	defer cancel()

	// Filter out already cached messages
	uncachedIDs := make([]string, 0, len(task.MessageIDs))
	for _, messageID := range task.MessageIDs {
		// Check if still needed (not cancelled, not already cached)
		select {
		case <-ctx.Done():
			return
		default:
		}

		if _, exists := p.GetCachedMessage(ctx, messageID); !exists {
			uncachedIDs = append(uncachedIDs, messageID)
		}
	}

	// If no messages to preload, exit early
	if len(uncachedIDs) == 0 {
		return
	}

	// Use existing parallel loading to fetch adjacent messages metadata
	messages, err := p.client.GetMessagesMetadataParallel(uncachedIDs, minInt(p.config.BackgroundWorkers, len(uncachedIDs)))
	if err != nil {
		// Log error but don't fail
		if p.logger != nil {
			p.logger.Printf("[PRELOAD ERROR] Failed to fetch adjacent messages: %v", err)
		}
		return
	}

	// Cache the fetched messages
	cachedCount := 0
	for _, message := range messages {
		if message != nil {
			p.cacheMessage(message.Id, message)
			cachedCount++
		}
	}

	// Log successful completion
	if cachedCount > 0 && p.logger != nil {
		p.logger.Printf("✅ PRELOAD ADJACENT: Cached %d adjacent messages", cachedCount)
	}
}

// cacheMessage adds a message to the cache with LRU management
func (p *MessagePreloaderImpl) cacheMessage(messageID string, message *gmail_v1.Message) {
	if message == nil {
		return // Don't cache nil messages
	}

	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	// Check if already cached
	if _, exists := p.messageCache[messageID]; exists {
		return // Already cached
	}

	// Estimate message size based on content
	messageSize := p.estimateMessageSize(message)

	// Check if we need to evict items to make room
	for p.currentMemory+messageSize > p.maxMemoryBytes && p.evictionList.Len() > 0 {
		p.evictLRUItem()
	}

	// Create cache item
	item := &CacheItem{
		Message:    message,
		Size:       messageSize,
		Timestamp:  time.Now(),
		AccessTime: time.Now(),
	}

	// Add to eviction list (most recently used)
	item.element = p.evictionList.PushFront(messageID)

	// Store in cache
	p.messageCache[messageID] = item
	p.currentMemory += messageSize

	// Update statistics
	p.statsMu.Lock()
	p.stats.TotalDataPreloadedMB += float64(messageSize) / (1024 * 1024)
	p.statsMu.Unlock()
}

// estimateMessageSize estimates the memory size of a Gmail message
func (p *MessagePreloaderImpl) estimateMessageSize(message *gmail_v1.Message) int64 {
	if message == nil {
		return 0
	}

	size := int64(1024) // Base size for metadata

	// Add size for payload
	if message.Payload != nil {
		// Add size for body data
		if message.Payload.Body != nil && message.Payload.Body.Data != "" {
			size += int64(len(message.Payload.Body.Data))
		}

		// Add size for headers
		if message.Payload.Headers != nil {
			for _, header := range message.Payload.Headers {
				size += int64(len(header.Name) + len(header.Value))
			}
		}

		// Add size for parts (recursive)
		size += p.estimatePartsSize(message.Payload.Parts)
	}

	// Cap the maximum size to prevent memory issues
	if size > 10*1024*1024 { // 10MB max per message
		size = 10 * 1024 * 1024
	}

	return size
}

// estimatePartsSize estimates the size of message parts recursively
func (p *MessagePreloaderImpl) estimatePartsSize(parts []*gmail_v1.MessagePart) int64 {
	if parts == nil {
		return 0
	}

	var totalSize int64
	for _, part := range parts {
		if part != nil {
			if part.Body != nil && part.Body.Data != "" {
				totalSize += int64(len(part.Body.Data))
			}
			if part.Headers != nil {
				for _, header := range part.Headers {
					totalSize += int64(len(header.Name) + len(header.Value))
				}
			}
			// Add size of nested parts
			totalSize += p.estimatePartsSize(part.Parts)
		}
	}
	return totalSize
}

// evictLRUItem removes the least recently used item from cache
func (p *MessagePreloaderImpl) evictLRUItem() {
	// Get least recently used item
	element := p.evictionList.Back()
	if element == nil {
		return
	}

	// Remove from eviction list
	p.evictionList.Remove(element)

	// Remove from cache
	messageID := element.Value.(string)
	if item, exists := p.messageCache[messageID]; exists {
		delete(p.messageCache, messageID)
		p.currentMemory -= item.Size
	}
}

// evictExcessMemory removes items until under memory limit
func (p *MessagePreloaderImpl) evictExcessMemory() {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	for p.currentMemory > p.maxMemoryBytes && len(p.messageCache) > 0 {
		p.evictLRUItem()
	}
}

// Shutdown gracefully stops the preloader
func (p *MessagePreloaderImpl) Shutdown() {
	p.shutdownOnce.Do(func() {
		close(p.shutdown)
		// Clear all caches to free memory
		_ = p.ClearCache(context.Background()) // Error not actionable during shutdown
	})
}

// Helper functions
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
