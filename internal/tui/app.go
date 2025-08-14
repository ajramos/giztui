package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ajramos/gmail-tui/internal/cache"
	calclient "github.com/ajramos/gmail-tui/internal/calendar"
	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/llm"
	"github.com/ajramos/gmail-tui/internal/render"
	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// App encapsulates the terminal UI and the Gmail client
type App struct {
	*tview.Application
	Pages    *Pages
	Config   *config.Config
	Client   *gmail.Client
	Calendar *calclient.Client
	LLM      llm.Provider
	Keys     config.KeyBindings
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	views    map[string]tview.Primitive
	cmdBuff  *CmdBuff
	running  bool
	flash    *Flash
	actions  *KeyActions
	// Email renderer
	emailRenderer *render.EmailRenderer
	// State management
	ids           []string
	messagesMeta  []*gmailapi.Message
	draftMode     bool
	draftIDs      []string
	showHelp      bool
	currentView   string
	currentFocus  string // Track current focus: "list" or "text"
	previousFocus string // Track previous focus before modal
	// Command system (k9s style)
	cmdMode         bool     // Whether we're in command mode
	cmdBuffer       string   // Current command buffer
	cmdHistory      []string // Command history
	cmdHistoryIndex int      // Current position in history
	cmdSuggestion   string   // Current command suggestion
	// Layout management
	currentLayout    LayoutType
	screenWidth      int
	screenHeight     int
	currentMessageID string // Added for label command execution
	nextPageToken    string // Gmail pagination

	// Search/Filter state
	searchMode    string // "" | "remote" | "local"
	currentQuery  string
	localFilter   string
	searchHistory []string
	// Local filter base snapshot (used only while searchMode=="local")
	baseIDs           []string
	baseMessagesMeta  []*gmailapi.Message
	baseNextPageToken string
	baseSelectionID   string
	// AI Summary pane
	aiSummaryView    *tview.TextView
	aiSummaryVisible bool
	aiSummaryCache   map[string]string // messageID -> summary
	aiInFlight       map[string]bool   // messageID -> generating
	// AI label suggestion cache
	aiLabelsCache map[string][]string // messageID -> suggestions

	// Markdown rendering
	markdownEnabled   bool
	markdownCache     map[string]string // messageID -> rendered ANSI (header+body)
	markdownTogglePer map[string]bool   // messageID -> prefer markdown

	// Message content cache (to avoid refetch on toggles)
	messageCache map[string]*gmail.Message

	// Calendar invite cache (parsed from text/calendar parts)
	inviteCache map[string]Invite // messageID -> invite metadata

	// Cache store (SQLite)
	cacheStore *cache.Store

	// Debug logging
	debug   bool
	logger  *log.Logger
	logFile *os.File

	// Labels contextual panel
	labelsView     *tview.Flex
	labelsVisible  bool
	labelsExpanded bool
	// RSVP side panel state
	rsvpVisible bool

	// Bulk selection
	selected map[string]bool // messageID -> selected
	bulkMode bool

	// VIM-style navigation
	vimSequence    string    // Track VIM key sequences like "gg"
	vimTimeout     time.Time // Timeout for key sequences

	// UI lifecycle flags
	uiReady          bool // true after first draw
	welcomeAnimating bool // avoid multiple spinner goroutines
	welcomeEmail     string

	// Formatting toggles
	llmTouchUpEnabled bool

	// Services (new architecture)
	emailService services.EmailService
	aiService    services.AIService
	labelService services.LabelService
	cacheService services.CacheService
	repository   services.MessageRepository
	errorHandler *ErrorHandler
}

// Pages manages the application pages and navigation
type Pages struct {
	*tview.Pages
	stack *Stack
}

// Stack manages navigation history
type Stack struct {
	items []string
	mu    sync.RWMutex
}

// CmdBuff manages command input and history
type CmdBuff struct {
	buff       []rune
	suggestion string
	listeners  map[BuffWatcher]struct{}
	kind       BufferKind
	active     bool
	mu         sync.RWMutex
}

// BufferKind represents the type of buffer
type BufferKind int

const (
	BuffCmd BufferKind = iota
	BuffFilter
)

// BuffWatcher interface for buffer changes
type BuffWatcher interface {
	BufferChanged([]rune)
}

// Flash manages notifications and messages
type Flash struct {
	textView tview.Primitive
	mu       sync.RWMutex
}

// NewFlash creates a new flash notification
func NewFlash() *Flash {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetBorder(true).
		SetBorderColor(tcell.ColorYellow)

	flash := &Flash{
		textView: textView,
	}
	return flash
}

// KeyActions manages keyboard shortcuts
type KeyActions struct {
	actions map[tcell.Key]KeyAction
	mx      sync.RWMutex
}

// KeyAction represents a keyboard action
type KeyAction struct {
	Description string
	Action      ActionHandler
	Visible     bool
	Shared      bool
}

// ActionHandler function type for key actions
type ActionHandler func(*tcell.EventKey) *tcell.EventKey

// LayoutType represents different layout configurations
type LayoutType int

const (
	LayoutWide   LayoutType = iota // Wide screen: side-by-side
	LayoutMedium                   // Medium screen: stacked with larger text
	LayoutNarrow                   // Narrow screen: full-width list/text
	LayoutMobile                   // Mobile-like: single column with compact design
)

// NewKeyActions creates a new key actions manager
func NewKeyActions() *KeyActions {
	return &KeyActions{
		actions: make(map[tcell.Key]KeyAction),
	}
}

// NewApp creates a new TUI application following k9s patterns
func NewApp(client *gmail.Client, calendarClient *calclient.Client, llmClient llm.Provider, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		Application:       tview.NewApplication(),
		Config:            cfg,
		Client:            client,
		Calendar:          calendarClient,
		LLM:               llmClient,
		Keys:              cfg.Keys,
		ctx:               ctx,
		cancel:            cancel,
		views:             make(map[string]tview.Primitive),
		cmdBuff:           NewCmdBuff(),
		flash:             NewFlash(),
		actions:           NewKeyActions(),
		emailRenderer:     render.NewEmailRenderer(),
		ids:               []string{},
		messagesMeta:      []*gmailapi.Message{},
		draftMode:         false,
		draftIDs:          []string{},
		showHelp:          false,
		currentView:       "messages",
		currentFocus:      "list",
		previousFocus:     "list", // Initialize previous focus
		cmdMode:           false,
		cmdBuffer:         "",
		cmdHistory:        make([]string, 0),
		cmdHistoryIndex:   -1,
		currentLayout:     LayoutMedium,
		screenWidth:       80,
		screenHeight:      25,
		currentMessageID:  "", // Initialize currentMessageID
		nextPageToken:     "",
		searchMode:        "",
		currentQuery:      "",
		localFilter:       "",
		searchHistory:     make([]string, 0, 10),
		baseIDs:           nil,
		baseMessagesMeta:  nil,
		baseNextPageToken: "",
		baseSelectionID:   "",
		aiSummaryCache:    make(map[string]string),
		aiInFlight:        make(map[string]bool),
		aiLabelsCache:     make(map[string][]string),
		markdownEnabled:   true,
		markdownCache:     make(map[string]string),
		markdownTogglePer: make(map[string]bool),
		messageCache:      make(map[string]*gmail.Message),
		inviteCache:       make(map[string]Invite),
		debug:             true,
		logger:            log.New(os.Stdout, "[gmail-tui] ", log.LstdFlags|log.Lmicroseconds),
		logFile:           nil,
		selected:          make(map[string]bool),
		bulkMode:          false,
		llmTouchUpEnabled: false,
	}

	// Initialize file logger (logging.go)
	app.initLogger()

	// Initialize pages
	app.Pages = NewPages()

	// Initialize components
	app.initComponents()

	// Apply theme to renderer (best-effort)
	app.applyTheme()

	// Set up key bindings
	app.bindKeys()

	// Initialize views
	app.initViews()

	// Recalcular en resize de forma segura (sin llamadas de red)
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		// Mark UI as ready on first draw
		if !app.uiReady {
			app.uiReady = true
		}
		w, h := screen.Size()
		if w != app.screenWidth || h != app.screenHeight {
			app.screenWidth, app.screenHeight = w, h
			app.reformatListItems()
		}
		return false
	})

	// Initialize services
	app.initServices()

	return app
}

// RegisterCacheStore wires a cache.Store into the App for local caching features
func (a *App) RegisterCacheStore(store *cache.Store) {
	a.cacheStore = store
	// Re-initialize cache service if store is available
	if a.cacheStore != nil && a.cacheService == nil {
		a.cacheService = services.NewCacheService(a.cacheStore)
		// Re-initialize AI service with cache if LLM is available
		if a.LLM != nil && a.aiService == nil {
			a.aiService = services.NewAIService(a.LLM, a.cacheService, a.Config)
		}
	}
}

// initServices initializes the service layer for better architecture
func (a *App) initServices() {
	// Initialize repository
	a.repository = services.NewMessageRepository(a.Client)

	// Initialize label service
	a.labelService = services.NewLabelService(a.Client)

	// Initialize cache service if store is available
	if a.cacheStore != nil {
		a.cacheService = services.NewCacheService(a.cacheStore)
	}

	// Initialize AI service if LLM provider is available
	if a.LLM != nil {
		a.aiService = services.NewAIService(a.LLM, a.cacheService, a.Config)
	}

	// Initialize email service
	a.emailService = services.NewEmailService(a.repository, a.Client, a.emailRenderer)

	// Initialize error handler
	a.initErrorHandler()
}

// initErrorHandler initializes the centralized error handler
func (a *App) initErrorHandler() {
	// Find status view
	var statusView *tview.TextView
	if view, exists := a.views["status"]; exists {
		if tv, ok := view.(*tview.TextView); ok {
			statusView = tv
		}
	}

	// Find flash view
	var flashView *tview.TextView
	if a.flash != nil && a.flash.textView != nil {
		if tv, ok := a.flash.textView.(*tview.TextView); ok {
			flashView = tv
		}
	}

	// Create error handler
	a.errorHandler = NewErrorHandler(a.Application, statusView, flashView, a.logger)
}

// Thread-safe state access methods

// GetCurrentView returns the current view name thread-safely
func (a *App) GetCurrentView() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentView
}

// SetCurrentView sets the current view name thread-safely
func (a *App) SetCurrentView(view string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.currentView = view
}

// GetCurrentMessageID returns the current message ID thread-safely
func (a *App) GetCurrentMessageID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentMessageID
}

// SetCurrentMessageID sets the current message ID thread-safely
func (a *App) SetCurrentMessageID(messageID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.currentMessageID = messageID
}

// GetMessageIDs returns a copy of message IDs thread-safely
func (a *App) GetMessageIDs() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ids := make([]string, len(a.ids))
	copy(ids, a.ids)
	return ids
}

// SetMessageIDs sets message IDs thread-safely
func (a *App) SetMessageIDs(ids []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ids = make([]string, len(ids))
	copy(a.ids, ids)
}

// setMessageIDsUnsafe sets message IDs without locking (for use when mutex is already held)
func (a *App) setMessageIDsUnsafe(ids []string) {
	a.ids = make([]string, len(ids))
	copy(a.ids, ids)
}

// AppendMessageID appends a message ID thread-safely
func (a *App) AppendMessageID(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ids = append(a.ids, id)
}

// appendMessageIDUnsafe appends a message ID without locking (for use when mutex is already held)
func (a *App) appendMessageIDUnsafe(id string) {
	a.ids = append(a.ids, id)
}

// ClearMessageIDs clears all message IDs thread-safely
func (a *App) ClearMessageIDs() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ids = []string{}
}

// clearMessageIDsUnsafe clears all message IDs without locking (for use when mutex is already held)
func (a *App) clearMessageIDsUnsafe() {
	a.ids = []string{}
}

// RemoveMessageIDAt removes a message ID at the specified index thread-safely
func (a *App) RemoveMessageIDAt(index int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.ids) {
		return false
	}
	a.ids = append(a.ids[:index], a.ids[index+1:]...)
	return true
}

// RemoveMessageIDByValue removes the first occurrence of a message ID thread-safely
func (a *App) RemoveMessageIDByValue(id string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, msgID := range a.ids {
		if msgID == id {
			a.ids = append(a.ids[:i], a.ids[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveMessageIDsInPlace removes IDs that exist in the provided map, using in-place filtering
func (a *App) RemoveMessageIDsInPlace(toRemove map[string]bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	i := 0
	for i < len(a.ids) {
		if _, ok := toRemove[a.ids[i]]; ok {
			a.ids = append(a.ids[:i], a.ids[i+1:]...)
		} else {
			i++
		}
	}
}

// IsRunning returns whether the app is running thread-safely
func (a *App) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// SetRunning sets the running state thread-safely
func (a *App) SetRunning(running bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.running = running
}

// GetScreenSize returns the current screen dimensions thread-safely
func (a *App) GetScreenSize() (int, int) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.screenWidth, a.screenHeight
}

// SetScreenSize sets the screen dimensions thread-safely
func (a *App) SetScreenSize(width, height int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.screenWidth = width
	a.screenHeight = height
}

// GetErrorHandler returns the error handler for centralized error handling
func (a *App) GetErrorHandler() *ErrorHandler {
	return a.errorHandler
}

// GetServices returns the service instances for business logic operations
func (a *App) GetServices() (services.EmailService, services.AIService, services.LabelService, services.CacheService, services.MessageRepository) {
	return a.emailService, a.aiService, a.labelService, a.cacheService, a.repository
}

// applyTheme loads theme colors and updates the email renderer
func (a *App) applyTheme() {
	// Try to load theme from skins directory; fallback to defaults
	loader := config.NewThemeLoader("skins")
	if theme, err := loader.LoadThemeFromFile("gmail-dark.yaml"); err == nil {
		a.emailRenderer.UpdateFromConfig(theme)
		// Aplicar a estilos globales
		tview.Styles.PrimitiveBackgroundColor = theme.Body.BgColor.Color()
		tview.Styles.PrimaryTextColor = theme.Body.FgColor.Color()
		tview.Styles.BorderColor = theme.Frame.Border.FgColor.Color()
		tview.Styles.FocusColor = theme.Frame.Border.FocusColor.Color()
	} else {
		def := config.DefaultColors()
		a.emailRenderer.UpdateFromConfig(def)
		tview.Styles.PrimitiveBackgroundColor = def.Body.BgColor.Color()
		tview.Styles.PrimaryTextColor = def.Body.FgColor.Color()
		tview.Styles.BorderColor = def.Frame.Border.FgColor.Color()
		tview.Styles.FocusColor = def.Frame.Border.FocusColor.Color()
	}
	// After updating global styles, also force background colors on existing widgets
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	}
	if header, ok := a.views["header"].(*tview.TextView); ok {
		header.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	}
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	}
	if a.aiSummaryView != nil {
		a.aiSummaryView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	}
}

// (moved to messages.go)

// NewPages creates a new Pages instance
func NewPages() *Pages {
	return &Pages{
		Pages: tview.NewPages(),
		stack: &Stack{
			items: make([]string, 0),
		},
	}
}

// NewCmdBuff creates a new command buffer
func NewCmdBuff() *CmdBuff {
	return &CmdBuff{
		buff:      make([]rune, 0),
		listeners: make(map[BuffWatcher]struct{}),
		kind:      BuffCmd,
		active:    false,
	}
}

// (moved to layout.go) initComponents

// initViews initializes the main views
// (moved to layout.go) initViews

// createMainLayout creates the main application layout
// (moved to layout.go) createMainLayout

// createStatusBar creates the status bar
// (moved to layout.go) createStatusBar

// (moved to status.go) showStatusMessage / setStatusPersistent

// (moved to layout.go) createHelpView/createSearchView

// generateHelpText generates the help text
func (a *App) generateHelpText() string {
	var help strings.Builder

	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("🐕 GizTUI - Help & Shortcuts\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	help.WriteString("🧭 Navigation\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("Enter     👁️  View selected message\n")
	help.WriteString("r         🔄 Refresh messages\n")
	help.WriteString("s         🔍 Search messages\n")
	help.WriteString("F         📫 Quick search: from current sender\n")
	help.WriteString("T         📤 Quick search: to current sender (includes Sent)\n")
	help.WriteString("S         🧵 Quick search: by current subject\n")
	help.WriteString("u         🔴 Show unread messages\n")
	help.WriteString("D         📝 View drafts\n")
	help.WriteString("A         📎 Show attachments\n")
	help.WriteString("l         🏷️  Manage labels\n\n")

	help.WriteString("✉️  Message Actions\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("R         💬 Reply to message\n")
	help.WriteString("n         ✏️  Compose new message\n")
	help.WriteString("t         👁️  Toggle read/unread\n")
	help.WriteString("d         🗑️  Move to trash\n")
	help.WriteString("a         �� Archive message\n")
	help.WriteString("m         📦 Move message\n\n")

	if a.LLM != nil {
		help.WriteString("🤖 AI Features\n")
		help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		help.WriteString("y         📝 Summarize message\n")
		help.WriteString("g         🤖 Generate reply\n")
		help.WriteString("o         🏷️  Suggest label\n\n")
	}

	help.WriteString("⚙️  Application\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("q         🚪 Quit application\n")
	help.WriteString("?         ❓ Toggle this help screen\n")

	return help.String()
}

// Run starts the TUI application
func (a *App) Run() error {
	// Set root to pages
	a.SetRoot(a.Pages, true)

	// Check if client is available
	if a.Client == nil {
		// Welcome screen in setup mode (no credentials)
		a.showWelcomeScreen(false, "")
	} else {
		// Welcome screen in loading mode with best-effort account email (fetch async)
		a.showWelcomeScreen(true, "")
		go func() {
			if a.Client != nil {
				if email, err := a.Client.ActiveAccountEmail(a.ctx); err == nil && email != "" {
					a.welcomeEmail = email
					a.QueueUpdateDraw(func() {
						// Re-render welcome with account email if still loading
						if text, ok := a.views["text"].(*tview.TextView); ok {
							text.SetText(a.buildWelcomeText(true, a.welcomeEmail, 0))
						}
						// Also refresh status bar baseline to include the email
						if status, ok := a.views["status"].(*tview.TextView); ok {
							status.SetText(a.statusBaseline())
						}
					})
				}
			}
		}()
		// Load messages in background
		go a.reloadMessages()
	}

	// Start the application
	return a.Application.Run()
}

// getActiveAccountEmail returns the current account email if available.
// For now, we do not have a reliable accessor from the Gmail client, so we
// return an empty string as a safe default.
// getActiveAccountEmail remains as a compatibility stub if needed elsewhere.
func (a *App) getActiveAccountEmail() string { return "" }

// (moved to keys.go) bindKeys

// handleCommandInput handles input when in command mode
// (moved to commands.go) handleCommandInput

// updateCommandBar updates the command bar display
// (moved to commands.go) updateCommandBar

// generateCommandSuggestion generates a suggestion based on the current command buffer
// (moved to commands.go) generateCommandSuggestion

// completeCommand completes the current command with the suggestion
// (moved to commands.go) completeCommand

// toggleHelp toggles the help display
func (a *App) toggleHelp() {
	if a.showHelp {
		a.Pages.SwitchToPage("main")
		a.showHelp = false
	} else {
		a.Pages.SwitchToPage("help")
		a.showHelp = true
	}
}

// (moved to messages.go)

// loadMoreMessages fetches the next page of inbox and appends to list
// (moved to messages.go)

// showMessage displays a message in the text view
// (moved to messages.go)

// showMessageWithoutFocus loads the message content but does not change focus
// (moved to messages.go)

// performSearch executes the search query
func (a *App) performSearch(query string) {
	if strings.TrimSpace(query) == "" {
		a.showError("Search query cannot be empty")
		return
	}

	// Update UI to searching state
	a.QueueUpdateDraw(func() {
		if list, ok := a.views["list"].(*tview.Table); ok {
			list.Clear()
			list.SetTitle(fmt.Sprintf(" 🔍 Searching: %s ", query))
		}
	})

	// Build effective query
	originalQuery := strings.TrimSpace(query)
	q := originalQuery
	if !strings.Contains(q, "in:") && !strings.Contains(q, "label:") {
		q = q + " -in:sent -in:draft -in:chat -in:spam -in:trash in:inbox"
	}

	// Stream search results progresivamente como en la carga inicial
	messages, next, err := a.Client.SearchMessagesPage(q, 50, "")
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.showError(fmt.Sprintf("❌ Search error: %v", err))
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetTitle(" ❌ Search failed ")
			}
		})
		return
	}

	// Reset state and show spinner
	a.ClearMessageIDs()
	a.messagesMeta = []*gmailapi.Message{}
	a.nextPageToken = next
	a.searchMode = "remote"
	a.currentQuery = q

	var spinnerStop chan struct{}
	if _, ok := a.views["list"].(*tview.Table); ok {
		spinnerStop = make(chan struct{})
		go func() {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			ticker := time.NewTicker(150 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-spinnerStop:
					return
				case <-ticker.C:
					prog := len(a.ids)
					total := len(messages)
					a.QueueUpdateDraw(func() {
						if tb, ok := a.views["list"].(*tview.Table); ok {
							tb.SetTitle(fmt.Sprintf(" %s Searching… (%d/%d) — %s ", frames[i%len(frames)], prog, total, originalQuery))
						}
					})
					i++
				}
			}
		}()
	}

	// Prepare label map and show system labels in list for search results (mixed scopes)
	if labels, err := a.Client.ListLabels(); err == nil {
		m := make(map[string]string, len(labels))
		for _, l := range labels {
			m[l.Id] = l.Name
		}
		a.emailRenderer.SetLabelMap(m)
	}
	a.emailRenderer.SetShowSystemLabelsInList(true)

	screenWidth := a.getFormatWidth()
	for _, msg := range messages {
		a.AppendMessageID(msg.Id)
		meta, err := a.Client.GetMessage(msg.Id)
		if err != nil {
			continue
		}
		a.messagesMeta = append(a.messagesMeta, meta)
		text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
		a.QueueUpdateDraw(func() {
			if table, ok := a.views["list"].(*tview.Table); ok {
				row := table.GetRowCount()
				table.SetCell(row, 0, tview.NewTableCell(text).SetExpansion(1))
			}
			a.reformatListItems()
		})
	}
	if spinnerStop != nil {
		close(spinnerStop)
	}
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 🔍 Search Results (%d) — %s ", len(a.ids), originalQuery))
			if table.GetRowCount() > 0 {
				table.Select(0, 0)
				if len(a.ids) > 0 {
					firstID := a.ids[0]
					a.SetCurrentMessageID(firstID)
					go a.showMessageWithoutFocus(firstID)
					if a.aiSummaryVisible {
						go a.generateOrShowSummary(firstID)
					}
				}
			}
		}
		// Keep policy for system labels on list while user is in search mode
		a.emailRenderer.SetShowSystemLabelsInList(true)
	})
}

// (moved to status.go) showError/showInfo

// Placeholder methods for functionality that will be implemented later
// (moved to messages.go) loadDrafts

// (moved to messages.go) composeMessage

// (moved to messages.go) listUnreadMessages

// (moved to messages.go) toggleMarkReadUnread

// updateMessageDisplay updates the display of a specific message in the list
func (a *App) updateMessageDisplay(index int, isUnread bool) {
	table, ok := a.views["list"].(*tview.Table)
	if !ok {
		return
	}

	// Get the current message
	if index < 0 || index >= len(a.ids) {
		return
	}

	messageID := a.ids[index]
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		return
	}

	// Determine current list width (fallback to 80)
	screenWidth := a.getListWidth()

	// Use the email renderer to format the message
	formattedText, _ := a.emailRenderer.FormatEmailList(message, screenWidth)

	// Add unread indicator
	if isUnread {
		formattedText = "● " + formattedText
	} else {
		formattedText = "○ " + formattedText
	}

	// Update the item in the table
	table.SetCell(index, 0, tview.NewTableCell(formattedText).SetExpansion(1))
}

// updateBaseCachedMessageLabels mirrors updateCachedMessageLabels but for the base snapshot (local filter)
func (a *App) updateBaseCachedMessageLabels(messageID, labelID string, applied bool) {
	if a.searchMode != "local" || a.baseIDs == nil {
		return
	}
	// Find index in baseIDs
	idx := -1
	for i, id := range a.baseIDs {
		if id == messageID {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(a.baseMessagesMeta) || a.baseMessagesMeta[idx] == nil {
		return
	}
	msg := a.baseMessagesMeta[idx]
	if applied {
		exists := false
		for _, l := range msg.LabelIds {
			if l == labelID {
				exists = true
				break
			}
		}
		if !exists {
			msg.LabelIds = append(msg.LabelIds, labelID)
		}
	} else {
		out := msg.LabelIds[:0]
		for _, l := range msg.LabelIds {
			if l != labelID {
				out = append(out, l)
			}
		}
		msg.LabelIds = out
	}
}

// moved to messages_actions.go

// (moved to labels.go) manageLabels

// showMessageLabelsView displays labels for a specific message
// (moved to labels.go) showMessageLabelsView

// toggleLabelForMessage toggles a label asynchronously and invokes onDone when finished
// (moved to labels.go) toggleLabelForMessage

// showMessagesWithLabel shows messages that have a specific label
// (moved to labels.go) showMessagesWithLabel

// showMessagesForLabel displays messages that have a specific label
// (moved to labels.go) showMessagesForLabel

// createNewLabelFromView creates a new label from the labels view
// (moved to labels.go) createNewLabelFromView

// deleteSelectedLabel deletes the selected label (placeholder for now)
// (moved to labels.go) deleteSelectedLabel

// formatRelativeTime formats a date like Gmail (e.g., "2h", "3d", "Jan 15")
func formatRelativeTime(date time.Time) string {
	now := time.Now()
	diff := now.Sub(date)

	if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes < 1 {
			return "now"
		}
		return fmt.Sprintf("%dm", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else {
		return date.Format("Jan 2")
	}
}

// (moved to layout.go) updateFocusIndicators

// toggleFocus switches focus between list and text view
// (moved to keys.go) toggleFocus

// restoreFocusAfterModal restores focus to the appropriate view after closing a modal
// (moved to keys.go) restoreFocusAfterModal

// (moved to messages.go) archiveSelected

// (moved to messages.go) replySelected

// (moved to messages.go) showAttachments

// summarizeSelected summarizes the selected message using LLM
func (a *App) summarizeSelected() {
	if a.LLM == nil {
		a.showStatusMessage("LLM disabled")
		return
	}
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}
	// Load content
	m, err := a.Client.GetMessageWithContent(messageID)
	if err != nil {
		a.showError("Failed to load message")
		return
	}
	body := m.PlainText
	if len([]rune(body)) > 8000 {
		body = string([]rune(body)[:8000])
	}
	// Show immediate status
	a.QueueUpdateDraw(func() { a.setStatusPersistent("🧠 Summarizing…") })
	go func() {
		resp, err := a.LLM.Generate("Summarize in 3 bullet points (keep language).\n\n" + body)
		if err != nil {
			a.QueueUpdateDraw(func() { a.showLLMError("inline summarize", err) })
			return
		}
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				prev := text.GetText(true)
				text.SetDynamicColors(true)
				text.SetText("— AI Summary —\n" + resp + "\n\n" + prev)
				text.ScrollToBeginning()
			}
			a.showStatusMessage("✅ Summary ready")
		})
	}()
}

// generateReply generates a reply using LLM
func (a *App) generateReply() {
	a.showInfo("Generate reply functionality not yet implemented")
}

// (moved to ai.go) suggestLabel

// (moved to ai.go) showLabelSuggestions

// createCommandBar creates the command bar component (k9s style)
// (moved to commands.go) createCommandBar

// showCommandBar displays the command bar and enters command mode
// (moved to commands.go) showCommandBar

// hideCommandBar hides the command bar and exits command mode
// (moved to commands.go) hideCommandBar

// executeCommand executes the current command
// (moved to commands.go) executeCommand

// (moved to commands.go) executeLabelsCommand

// (moved to labels.go) executeLabelAdd

// (moved to labels.go) executeLabelRemove

// (moved to commands.go) executeSearchCommand

// (moved to commands.go) executeInboxCommand

// (moved to commands.go) executeComposeCommand

// (moved to commands.go) executeHelpCommand

// (moved to commands.go) executeQuitCommand

// getCurrentMessageID gets the ID of the currently selected message
// (moved to messages.go)

// addToHistory adds a command to the history
// (moved to commands.go) addToHistory

// getListWidth returns current inner width of the list view or a sensible fallback
// (moved to messages.go)

// getFormatWidth devuelve el ancho disponible para el texto de las filas
// (moved to messages.go)

// refreshMessageContent reloads the message and updates the text view without changing focus
// (moved to messages.go)

// refreshMessageContentWithOverride reloads message and overrides labels shown with provided names
// (moved to messages.go)

// (moved to markdown.go)

// renderMessageContent builds header + body (Markdown or plain text)
// (moved to markdown.go)

// updateCachedMessageLabels updates the cached labels for a message ID
// (moved to labels.go) updateCachedMessageLabels

// moveSelected opens the labels picker to choose a destination label, applies it, then archives the message
// (moved to labels.go) moveSelected

// showMoveLabelsView lets user choose a label to apply and then archives the message (move semantics)
// (moved to labels.go) showMoveLabelsView

// filterAndSortLabels filters out system labels and returns a name-sorted slice
// (moved to labels.go) filterAndSortLabels

// partitionAndSortLabels returns two sorted slices: labels applied to current and the rest
// (moved to labels.go) partitionAndSortLabels

// (moved to ai.go) toggleAISummary

// (moved to ai.go) generateOrShowSummary

// showAllLabelsPicker shows a list of all actionable labels to apply one to the message
// (moved to labels.go) showAllLabelsPicker

// applyLabelAndRefresh aplica una etiqueta usando el mismo mecanismo que en la vista de 'l'
// y refresca el contenido del mensaje cuando termina
// (moved to labels.go) applyLabelAndRefresh
