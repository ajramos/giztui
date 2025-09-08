package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	calclient "github.com/ajramos/giztui/internal/calendar"
	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/llm"
	"github.com/ajramos/giztui/internal/obsidian"
	"github.com/ajramos/giztui/internal/render"
	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// ActivePicker represents the currently active side panel picker
type ActivePicker string

const (
	PickerNone          ActivePicker = ""
	PickerLabels        ActivePicker = "labels"
	PickerDrafts        ActivePicker = "drafts"
	PickerObsidian      ActivePicker = "obsidian"
	PickerAttachments   ActivePicker = "attachments"
	PickerLinks         ActivePicker = "links"
	PickerPrompts       ActivePicker = "prompts"
	PickerBulkPrompts   ActivePicker = "bulk_prompts"
	PickerSavedQueries  ActivePicker = "saved_queries"
	PickerThemes        ActivePicker = "themes"
	PickerAI            ActivePicker = "ai_labels"
	PickerContentSearch ActivePicker = "content_search"
	PickerRSVP          ActivePicker = "rsvp"
	PickerAccounts      ActivePicker = "accounts"
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
	ids                     []string
	messagesMeta            []*gmailapi.Message
	currentThreads          []*services.ThreadInfo // Current threads for column system
	draftMode               bool
	draftIDs                []string
	showHelp                bool
	helpBackupText          string // Backup of text content before showing help
	helpBackupHeader        string // Backup of header content before showing help
	helpBackupTitle         string // Backup of text container title before showing help
	preloadStatusVisible    bool
	preloadBackupText       string // Backup of text content before showing preload status
	preloadBackupHeader     string // Backup of header content before showing preload status
	preloadBackupTitle      string // Backup of text container title before showing preload status
	promptStatsVisible      bool
	promptStatsBackupText   string // Backup of text content before showing prompt stats
	promptStatsBackupHeader string // Backup of header content before showing prompt stats
	promptStatsBackupTitle  string // Backup of text container title before showing prompt stats
	currentView             string
	currentFocus            string // Track current focus: "list" or "text"
	previousFocus           string // Track previous focus before modal
	// Command system (k9s style)
	cmdMode          bool     // Whether we're in command mode
	cmdBuffer        string   // Current command buffer
	cmdHistory       []string // Command history
	cmdHistoryIndex  int      // Current position in history
	cmdSuggestion    string   // Current command suggestion
	cmdFocusOverride string   // Override focus restoration for special commands
	// Prompt details state
	originalHeaderHeight int // Store original header height before hiding
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
	// Enhanced text view for content navigation and search
	enhancedTextView    *EnhancedTextView
	aiSummaryCache      map[string]string  // messageID -> summary
	aiInFlight          map[string]bool    // messageID -> generating
	aiPanelInPromptMode bool               // Track if panel is being used for prompt vs summary
	streamingCancel     context.CancelFunc // Cancel function for active streaming operations
	// AI label suggestion cache
	aiLabelsCache map[string][]string // messageID -> suggestions

	// Markdown rendering
	markdownEnabled   bool
	markdownCache     map[string]string // messageID -> rendered ANSI (header+body)
	markdownTogglePer map[string]bool   // messageID -> prefer markdown

	// Message content cache (to avoid refetch on toggles)
	messageCache   map[string]*gmail.Message
	messageCacheMu sync.RWMutex

	// Calendar invite cache (parsed from text/calendar parts)
	inviteCache map[string]Invite // messageID -> invite metadata

	// Database store (SQLite)
	dbStore *db.Store

	// Debug logging
	debug   bool
	logger  *log.Logger
	logFile *os.File

	// Side panel picker state management
	labelsView          *tview.Flex
	currentActivePicker ActivePicker // Replaces labelsVisible - tracks which picker is active
	labelsExpanded      bool

	// Slack contextual panel
	slackView    *tview.Flex
	slackVisible bool

	// Composition panel
	compositionPanel *CompositionPanel
	// RSVP side panel state managed by ActivePicker enum

	// Bulk selection
	selected map[string]bool // messageID -> selected
	bulkMode bool

	// VIM-style navigation
	vimSequence string    // Track VIM key sequences like "gg"
	vimTimeout  time.Time // Timeout for key sequences

	// VIM-style range operations
	vimOperationCount    int    // Track count in sequences (e.g., "5" in "s5s")
	vimOperationType     string // Track operation type (e.g., "s" in "s5s")
	vimOriginalMessageID string // Store message ID when VIM sequence started

	// UI lifecycle flags
	uiReady          bool // true after first draw
	welcomeAnimating bool // avoid multiple spinner goroutines
	welcomeEmail     string
	messagesLoading  bool // true when messages are being loaded

	// Formatting toggles
	llmTouchUpEnabled bool

	// Message display options
	showMessageNumbers bool

	// Services (new architecture)
	accountService     services.AccountService
	emailService       services.EmailService
	aiService          services.AIService
	labelService       services.LabelService
	cacheService       services.CacheService
	repository         services.MessageRepository
	compositionService services.CompositionService
	bulkPromptService  *services.BulkPromptServiceImpl
	promptService      services.PromptService
	slackService       services.SlackService
	obsidianService    services.ObsidianService
	linkService        services.LinkService
	attachmentService  services.AttachmentService
	gmailWebService    services.GmailWebService
	contentNavService  services.ContentNavigationService
	themeService       services.ThemeService
	displayService     services.DisplayService
	queryService       services.QueryService
	threadService      services.ThreadService
	undoService        services.UndoService
	preloaderService   services.MessagePreloader
	currentTheme       *config.ColorsConfig // Current theme cache for helper functions
	errorHandler       *ErrorHandler
}

// Pages manages the application pages and navigation
type Pages struct {
	*tview.Pages
	stack *Stack
}

// Stack manages navigation history
type Stack struct {
	items []string
	// OBLITERATED: unused field mu sync.RWMutex eliminated! 💥
}

// CmdBuff manages command input and history
type CmdBuff struct {
	buff []rune
	// OBLITERATED: unused field suggestion string eliminated! 💥
	listeners map[BuffWatcher]struct{}
	kind      BufferKind
	active    bool
	// OBLITERATED: unused field mu sync.RWMutex eliminated! 💥
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
		SetBorder(true)
		// Border color will be set by theme system when flash is shown

	flash := &Flash{
		textView: textView,
	}
	return flash
}

// UpdateBorderColor updates the flash border color with theme color
func (f *Flash) UpdateBorderColor(color tcell.Color) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if textView, ok := f.textView.(*tview.TextView); ok {
		textView.SetBorderColor(color)
	}
}

// KeyActions manages keyboard shortcuts
type KeyActions struct {
	actions map[tcell.Key]KeyAction
	// OBLITERATED: unused field mx sync.RWMutex eliminated! 💥
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
		emailRenderer:     render.NewEmailRenderer(cfg),
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
		logger:            log.New(os.Stdout, "[giztui] ", log.LstdFlags|log.Lmicroseconds),
		logFile:           nil,
		selected:          make(map[string]bool),
		bulkMode:          false,
		llmTouchUpEnabled: false,
		messagesLoading:   false,
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

	// Enhanced resize handling for responsive column system
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		// Mark UI as ready on first draw
		if !app.uiReady {
			app.uiReady = true
		}
		w, h := screen.Size()
		if w != app.screenWidth || h != app.screenHeight {
			app.screenWidth, app.screenHeight = w, h
			// Trigger comprehensive layout refresh on resize
			app.onWindowResize()
		}
		return false
	})

	// Initialize services
	app.initServices()

	return app
}

// onWindowResize handles window resize events with debouncing and comprehensive layout refresh
func (a *App) onWindowResize() {
	// Force complete table layout recalculation to prevent column overflow
	if table, ok := a.views["list"].(*tview.Table); ok {
		// Store current selection to restore after refresh
		currentRow, currentCol := table.GetSelection()

		// Force table reconstruction with new column widths
		a.refreshTableDisplay()

		// Restore selection if valid
		if currentRow > 0 && currentRow < table.GetRowCount() {
			table.Select(currentRow, currentCol)
		}
	}

	// Also refresh message content if there's a current message
	// This ensures message content adapts to new width
	if currentMessageID := a.getCurrentSelectedMessageID(); currentMessageID != "" {
		go func() {
			// Use a short delay to avoid excessive refreshes during window dragging
			time.Sleep(50 * time.Millisecond)
			if a.getCurrentSelectedMessageID() == currentMessageID {
				// Preserve current focus state during resize refresh
				currentFocusState := a.GetCurrentFocus()
				currentPickerState := a.currentActivePicker

				// Use showMessageWithoutFocus to avoid changing focus
				a.showMessageWithoutFocus(currentMessageID)

				// Restore previous focus and picker state
				if currentPickerState != PickerNone {
					// If a picker was active, restore its focus
					a.currentFocus = currentFocusState
					a.updateFocusIndicators(currentFocusState)
					// Restore picker focus by setting focus to the picker view
					if a.labelsView != nil {
						a.SetFocus(a.labelsView)
					}
				}
			}
		}()
	}
}

// RegisterDBStore wires a db.Store into the App for local data storage features
func (a *App) RegisterDBStore(store *db.Store) {
	a.dbStore = store
	if a.logger != nil {
		a.logger.Printf("RegisterDBStore: store registered, re-initializing services")
	}

	// Re-initialize all services with the new store
	a.reinitializeServices()
}

// reinitializeServices re-initializes services when store becomes available
func (a *App) reinitializeServices() {
	if a.logger != nil {
		a.logger.Printf("reinitializeServices: starting service re-initialization")
	}

	// Initialize cache service if store is available
	if a.dbStore != nil && a.cacheService == nil {
		cacheStore := db.NewCacheStore(a.dbStore)
		a.cacheService = services.NewCacheService(cacheStore)
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: cache service initialized: %v", a.cacheService != nil)
		}
	}

	// CRITICAL FIX: Re-create AI service if cache service was just created
	// The existing AI service was created without cache, so we need to recreate it
	if a.LLM != nil && a.cacheService != nil {
		a.aiService = services.NewAIService(a.LLM, a.cacheService, a.Config)
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: AI service re-created with cache: %v", a.aiService != nil)
		}
	} else if a.LLM != nil && a.aiService == nil {
		// Fallback: create AI service without cache if none exists
		a.aiService = services.NewAIService(a.LLM, a.cacheService, a.Config)
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: AI service initialized: %v", a.aiService != nil)
		}
	}

	// Initialize prompt service first (without bulk service for now)
	if a.dbStore != nil && a.aiService != nil && a.promptService == nil {
		promptStore := db.NewPromptStore(a.dbStore)
		a.promptService = services.NewPromptService(promptStore, a.aiService, nil) // Pass nil for now
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: prompt service initialized: %v", a.promptService != nil)
		}
	}

	// Initialize bulk prompt service if dependencies are available
	if a.repository != nil && a.aiService != nil && a.cacheService != nil && a.promptService != nil && a.bulkPromptService == nil {
		a.bulkPromptService = services.NewBulkPromptService(a.emailService, a.aiService, a.cacheService, a.repository, a.promptService)
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: bulk prompt service initialized: %v", a.bulkPromptService != nil)
		}
	}

	// Now update prompt service with bulk service
	if a.promptService != nil && a.bulkPromptService != nil {
		// We need to update the prompt service to include the bulk service
		// This is a bit of a hack, but it's the cleanest way to handle the circular dependency
		if promptService, ok := a.promptService.(*services.PromptServiceImpl); ok {
			promptService.SetBulkService(a.bulkPromptService)
		}
	}

	// Update bulk prompt service with prompt service
	if a.bulkPromptService != nil && a.promptService != nil {
		// We need to update the bulk prompt service to include the prompt service
		// This is a bit of a hack, but it's the cleanest way to handle the circular dependency
		a.bulkPromptService.SetPromptService(a.promptService)
	}

	// Initialize query service if database store is available
	if a.dbStore != nil && a.queryService == nil {
		queryStore := db.NewQueryStore(a.dbStore)
		a.queryService = services.NewQueryService(queryStore, a.Config)

		// Set account email if available
		if queryServiceImpl, ok := a.queryService.(*services.QueryServiceImpl); ok {
			// Try to get account email, use fallback if not available
			email := a.getActiveAccountEmail()
			if email == "" {
				email = "user@example.com" // Safe fallback
			}
			queryServiceImpl.SetAccountEmail(email)
			if a.logger != nil {
				a.logger.Printf("reinitializeServices: query service account email set to: %s", email)
			}
		}

		if a.logger != nil {
			a.logger.Printf("reinitializeServices: query service initialized: %v", a.queryService != nil)
		}
	}

	// Initialize Obsidian service if database store is available
	if a.dbStore != nil && a.obsidianService == nil {
		obsidianStore := db.NewObsidianStore(a.dbStore)

		// Get Obsidian config from app config
		var obsidianConfig *obsidian.ObsidianConfig
		if a.Config != nil && a.Config.Obsidian != nil {
			obsidianConfig = a.Config.Obsidian
			if a.logger != nil {
				a.logger.Printf("reinitializeServices: using Obsidian config from app config")
			}
		} else {
			// Fallback to default config if not available
			obsidianConfig = obsidian.DefaultObsidianConfig()
			if a.logger != nil {
				a.logger.Printf("reinitializeServices: using default Obsidian config")
			}
		}

		a.obsidianService = services.NewObsidianService(obsidianStore, obsidianConfig, a.logger)
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: obsidian service initialized: %v", a.obsidianService != nil)
		}
	}

	if a.logger != nil {
		a.logger.Printf("reinitializeServices: service re-initialization completed")
	}
}

// initServices initializes the service layer for better architecture
func (a *App) initServices() {
	if a.logger != nil {
		a.logger.Printf("initServices: starting service initialization")
	}

	// Initialize account service
	a.accountService = services.NewAccountService(a.Config)
	if a.logger != nil {
		a.logger.Printf("initServices: account service initialized: %v", a.accountService != nil)
	}

	// Update active account status to Connected since Gmail client is working
	// (The fact that we got this far means authentication was successful)
	if activeAccount, err := a.accountService.GetActiveAccount(a.ctx); err == nil {
		// Create a copy and update status
		updatedAccount := *activeAccount
		updatedAccount.Status = services.AccountStatusConnected

		// Try to get email from client if not already set
		if updatedAccount.Email == "" {
			if email, err := a.Client.ActiveAccountEmail(a.ctx); err == nil {
				updatedAccount.Email = email
			}
		}

		// Update the account in the service
		if err := a.accountService.UpdateAccount(a.ctx, &updatedAccount); err != nil {
			if a.logger != nil {
				a.logger.Printf("initServices: failed to update account status: %v", err)
			}
		} else if a.logger != nil {
			a.logger.Printf("initServices: updated active account status to Connected")
		}
	}

	// Initialize repository
	a.repository = services.NewMessageRepository(a.Client)
	if a.logger != nil {
		a.logger.Printf("initServices: repository initialized: %v", a.repository != nil)
	}

	// Initialize label service
	a.labelService = services.NewLabelService(a.Client)
	if a.logger != nil {
		a.logger.Printf("initServices: label service initialized: %v", a.labelService != nil)
	}

	// Initialize cache service if store is available
	if a.dbStore != nil {
		cacheStore := db.NewCacheStore(a.dbStore)
		a.cacheService = services.NewCacheService(cacheStore)
		if a.logger != nil {
			a.logger.Printf("initServices: cache service initialized: %v", a.cacheService != nil)
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: cache service NOT initialized - dbStore is nil")
		}
	}

	// Initialize AI service if LLM provider is available
	if a.LLM != nil {
		a.aiService = services.NewAIService(a.LLM, a.cacheService, a.Config)
		if a.logger != nil {
			a.logger.Printf("initServices: AI service initialized: %v", a.aiService != nil)
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: AI service NOT initialized - LLM is nil")
		}
	}

	// Initialize email service
	a.emailService = services.NewEmailService(a.repository, a.Client, a.emailRenderer)
	if a.logger != nil {
		a.logger.Printf("initServices: email service initialized: %v", a.emailService != nil)
	}
	// Wire logger to email service for debug output
	if emailServiceImpl, ok := a.emailService.(*services.EmailServiceImpl); ok && a.logger != nil {
		emailServiceImpl.SetLogger(a.logger)
	}

	// Initialize composition service
	a.compositionService = services.NewCompositionService(a.emailService, a.Client, a.repository)
	if a.logger != nil {
		a.logger.Printf("initServices: composition service initialized: %v", a.compositionService != nil)
	}
	// Wire logger to composition service for debug output
	if compositionServiceImpl, ok := a.compositionService.(*services.CompositionServiceImpl); ok && a.logger != nil {
		compositionServiceImpl.SetLogger(a.logger)
	}

	// Initialize link service
	a.linkService = services.NewLinkService(a.Client, a.emailRenderer)
	if a.logger != nil {
		a.logger.Printf("initServices: link service initialized: %v", a.linkService != nil)
	}

	// Initialize attachment service
	a.attachmentService = services.NewAttachmentService(a.Client, a.Config)
	if a.logger != nil {
		a.logger.Printf("initServices: attachment service initialized: %v", a.attachmentService != nil)
	}

	// Initialize Gmail web service
	a.gmailWebService = services.NewGmailWebService(a.linkService)
	if a.logger != nil {
		a.logger.Printf("initServices: gmail web service initialized: %v", a.gmailWebService != nil)
	}

	// Initialize bulk prompt service if dependencies are available
	if a.repository != nil && a.aiService != nil && a.cacheService != nil {
		// For now, pass nil as promptService to avoid circular dependency
		// It will be set later in reinitializeServices
		a.bulkPromptService = services.NewBulkPromptService(a.emailService, a.aiService, a.cacheService, a.repository, nil)
		if a.logger != nil {
			a.logger.Printf("initServices: bulk prompt service initialized: %v", a.bulkPromptService != nil)
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: bulk prompt service NOT initialized - repository=%v aiService=%v cacheService=%v",
				a.repository != nil, a.aiService != nil, a.cacheService != nil)
		}
	}

	// Initialize prompt service if database store is available
	if a.dbStore != nil && a.aiService != nil && a.bulkPromptService != nil {
		promptStore := db.NewPromptStore(a.dbStore)
		a.promptService = services.NewPromptService(promptStore, a.aiService, a.bulkPromptService)
		if a.logger != nil {
			a.logger.Printf("initServices: prompt service initialized: %v", a.promptService != nil)
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: prompt service NOT initialized - dbStore=%v aiService=%v bulkPromptService=%v",
				a.dbStore != nil, a.aiService != nil, a.bulkPromptService != nil)
		}
	}

	// Initialize Slack service if enabled in config
	if a.Config.Slack.Enabled {
		a.slackService = services.NewSlackService(a.Client, a.Config, a.aiService)
		if a.logger != nil {
			a.logger.Printf("initServices: slack service initialized: %v", a.slackService != nil)
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: slack service NOT initialized - SlackEnabled is false")
		}
	}

	// Initialize Obsidian service if database store is available
	if a.dbStore != nil {
		obsidianStore := db.NewObsidianStore(a.dbStore)
		// Get Obsidian config from app config
		var obsidianConfig *obsidian.ObsidianConfig
		if a.Config != nil && a.Config.Obsidian != nil {
			obsidianConfig = a.Config.Obsidian
			if a.logger != nil {
				a.logger.Printf("initServices: using Obsidian config from app config")
			}
		} else {
			// Fallback to default config if not available
			obsidianConfig = obsidian.DefaultObsidianConfig()
			// Set a reasonable vault path if not configured
			homeDir, err := os.UserHomeDir()
			if err == nil {
				obsidianConfig.VaultPath = filepath.Join(homeDir, "ObsidianVault")
			} else {
				obsidianConfig.VaultPath = "./ObsidianVault"
			}
			if a.logger != nil {
				a.logger.Printf("initServices: using default Obsidian config")
			}
		}

		a.obsidianService = services.NewObsidianService(obsidianStore, obsidianConfig, a.logger)
		if a.logger != nil {
			a.logger.Printf("initServices: obsidian service initialized: %v", a.obsidianService != nil)
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: obsidian service NOT initialized - dbStore=%v", a.dbStore != nil)
		}
	}

	// Initialize content navigation service (no dependencies)
	a.contentNavService = services.NewContentNavigationService()
	if a.logger != nil {
		a.logger.Printf("initServices: content navigation service initialized: %v", a.contentNavService != nil)
	}

	// Initialize theme service
	customThemeDir := ""
	if a.Config != nil && a.Config.Theme.CustomDir != "" {
		customThemeDir = a.Config.Theme.CustomDir
	}

	// Determine the built-in themes directory path
	// Check if we have an absolute path or need to resolve relative to executable location
	builtinThemesDir := "themes"
	if _, err := os.Stat(builtinThemesDir); os.IsNotExist(err) {
		// If themes directory doesn't exist in current dir, try relative to parent
		builtinThemesDir = "../themes"
		if _, err := os.Stat(builtinThemesDir); os.IsNotExist(err) {
			// If that doesn't exist either, try to find it relative to the executable
			if exe, err := os.Executable(); err == nil {
				exeDir := filepath.Dir(exe)
				builtinThemesDir = filepath.Join(exeDir, "..", "themes")
				if _, err := os.Stat(builtinThemesDir); os.IsNotExist(err) {
					// Last resort - try themes in the same directory as executable
					builtinThemesDir = filepath.Join(exeDir, "themes")
				}
			}
		}
	}

	// Create theme apply function that calls the app's applyTheme method
	applyThemeFunc := func(themeConfig *config.ColorsConfig) error {
		return a.applyThemeConfig(themeConfig)
	}

	a.themeService = services.NewThemeService(builtinThemesDir, customThemeDir, applyThemeFunc)
	if a.logger != nil {
		a.logger.Printf("initServices: theme service initialized: %v", a.themeService != nil)
	}

	// Initialize display service (no dependencies)
	a.displayService = services.NewDisplayService()
	if a.logger != nil {
		a.logger.Printf("initServices: display service initialized: %v", a.displayService != nil)
	}

	// Initialize thread service (database store and AI service are optional for basic functionality)
	a.threadService = services.NewThreadService(a.Client, a.dbStore, a.aiService)
	if a.logger != nil {
		a.logger.Printf("initServices: thread service initialized: %v (dbStore: %v, AI service: %v)",
			a.threadService != nil, a.dbStore != nil, a.aiService != nil)
	}

	// Initialize undo service (needs repository, labelService, and gmailClient)
	if a.repository != nil && a.labelService != nil && a.Client != nil {
		a.undoService = services.NewUndoService(a.repository, a.labelService, a.Client)
		if a.logger != nil {
			a.logger.Printf("initServices: undo service initialized: %v", a.undoService != nil)
		}

		// Wire logger to undo service for debug output
		if undoServiceImpl, ok := a.undoService.(*services.UndoServiceImpl); ok && a.logger != nil {
			undoServiceImpl.SetLogger(a.logger)
		}

		// Wire undo service to email service to enable undo recording
		if a.emailService != nil {
			if emailServiceImpl, ok := a.emailService.(*services.EmailServiceImpl); ok {
				emailServiceImpl.SetUndoService(a.undoService)
				if a.logger != nil {
					a.logger.Printf("initServices: undo service wired to email service")
				}
			}
		}

		// Wire undo service to label service to enable undo recording
		if a.labelService != nil {
			if labelServiceImpl, ok := a.labelService.(*services.LabelServiceImpl); ok {
				labelServiceImpl.SetUndoService(a.undoService)
				if a.logger != nil {
					a.logger.Printf("initServices: undo service wired to label service")
				}
			}
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: undo service NOT initialized - repository=%v labelService=%v Client=%v",
				a.repository != nil, a.labelService != nil, a.Client != nil)
		}
	}

	// Load theme from config with fallbacks
	themeName := "gmail-dark" // Default fallback
	if a.Config != nil && a.Config.Theme.Current != "" {
		themeName = a.Config.Theme.Current
	}

	if a.themeService != nil {
		if err := a.themeService.ApplyTheme(a.ctx, themeName); err != nil {
			if a.logger != nil {
				a.logger.Printf("Failed to load configured theme %s: %v", themeName, err)
			}
			// Try default theme as fallback
			if err := a.themeService.ApplyTheme(a.ctx, "gmail-dark"); err != nil {
				if a.logger != nil {
					a.logger.Printf("Failed to load default theme: %v", err)
				}
				// Continue with hardcoded colors as final fallback
			}
		} else if a.logger != nil {
			a.logger.Printf("Successfully loaded theme: %s", themeName)
		}
	}

	// Initialize preloader service (performance optimization)
	if a.Client != nil && a.Config != nil {
		// Convert config format to services format
		preloadConfig := &services.PreloadConfig{
			Enabled:                a.Config.Performance.Preloading.Enabled,
			NextPageEnabled:        a.Config.Performance.Preloading.NextPage.Enabled,
			NextPageThreshold:      a.Config.Performance.Preloading.NextPage.Threshold,
			NextPageMaxPages:       a.Config.Performance.Preloading.NextPage.MaxPages,
			AdjacentEnabled:        a.Config.Performance.Preloading.AdjacentMessages.Enabled,
			AdjacentCount:          a.Config.Performance.Preloading.AdjacentMessages.Count,
			BackgroundWorkers:      a.Config.Performance.Preloading.Limits.BackgroundWorkers,
			CacheSizeMB:            a.Config.Performance.Preloading.Limits.CacheSizeMB,
			APIQuotaReservePercent: a.Config.Performance.Preloading.Limits.APIQuotaReservePercent,
		}

		a.preloaderService = services.NewMessagePreloader(a.Client, preloadConfig, a.logger)
		if a.logger != nil {
			a.logger.Printf("initServices: preloader service initialized: %v (enabled: %v)",
				a.preloaderService != nil, preloadConfig.Enabled)
		}
	} else {
		if a.logger != nil {
			a.logger.Printf("initServices: preloader service NOT initialized - Client=%v Config=%v",
				a.Client != nil, a.Config != nil)
		}
	}

	if a.logger != nil {
		a.logger.Printf("initServices: service initialization completed")
	}

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
	a.errorHandler = NewErrorHandler(a.Application, a, statusView, flashView, a.logger)
}

// Thread-safe state access methods

// GetCurrentView returns the current view name thread-safely
func (a *App) GetCurrentView() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentView
}

// GetCurrentFocus returns the current focus state thread-safely
func (a *App) GetCurrentFocus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentFocus
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

// IsMessagesLoading returns whether messages are currently being loaded
func (a *App) IsMessagesLoading() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.messagesLoading
}

// SetMessagesLoading sets the messages loading state
func (a *App) SetMessagesLoading(loading bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messagesLoading = loading
}

// SetMessageIDs sets message IDs thread-safely
func (a *App) SetMessageIDs(ids []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ids = make([]string, len(ids))
	copy(a.ids, ids)
}

// AppendMessageID appends a message ID thread-safely
func (a *App) AppendMessageID(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ids = append(a.ids, id)
}

// ClearMessageIDs clears all message IDs thread-safely
func (a *App) ClearMessageIDs() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ids = []string{}
}

// Removed unused unsafe methods: setMessageIDsUnsafe, appendMessageIDUnsafe, clearMessageIDsUnsafe

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

// GetMessageFromCache returns a cached message thread-safely
func (a *App) GetMessageFromCache(messageID string) (*gmail.Message, bool) {
	a.messageCacheMu.RLock()
	defer a.messageCacheMu.RUnlock()
	message, exists := a.messageCache[messageID]
	return message, exists
}

// SetMessageInCache stores a message in cache thread-safely
func (a *App) SetMessageInCache(messageID string, message *gmail.Message) {
	a.messageCacheMu.Lock()
	defer a.messageCacheMu.Unlock()
	a.messageCache[messageID] = message
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
func (a *App) GetServices() (services.EmailService, services.AIService, services.LabelService, services.CacheService, services.MessageRepository, services.CompositionService, services.PromptService, services.ObsidianService, services.LinkService, services.GmailWebService, services.AttachmentService, services.DisplayService) {
	return a.emailService, a.aiService, a.labelService, a.cacheService, a.repository, a.compositionService, a.promptService, a.obsidianService, a.linkService, a.gmailWebService, a.attachmentService, a.displayService
}

// GetAccountService returns the account service instance
func (a *App) GetAccountService() services.AccountService {
	return a.accountService
}

// GetUndoService returns the undo service instance
func (a *App) GetUndoService() services.UndoService {
	return a.undoService
}

// GetPreloaderService returns the message preloader service instance
func (a *App) GetPreloaderService() services.MessagePreloader {
	return a.preloaderService
}

// performUndo performs the undo operation and provides user feedback
func (a *App) performUndo() {
	if a.undoService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Undo service not available")
		return
	}

	if !a.undoService.HasUndoableAction() {
		a.GetErrorHandler().ShowInfo(a.ctx, "No action to undo")
		return
	}

	// Perform the undo operation
	result, err := a.undoService.UndoLastAction(a.ctx)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Undo failed: %v", err))
		return
	}

	// Provide user feedback and handle UI updates
	if result.Success {
		// Show success message with refresh guidance
		message := a.getUndoStatusMessage(result)
		// Add label removal hint for archive undos (which might be move operations)
		if strings.Contains(result.Description, "Archive") {
			message += " (Note: Labels remain - remove manually if needed)"
		}
		go func() {
			a.GetErrorHandler().ShowSuccess(a.ctx, message)
		}()

		// Smart reload: only when necessary to show restored messages
		a.smartUndoReload(result)
	} else {
		// Show clear error - operation failed completely
		if len(result.Errors) > 0 {
			errorMsg := fmt.Sprintf("❌ Undo failed: %s", strings.Join(result.Errors, "; "))
			a.GetErrorHandler().ShowError(a.ctx, errorMsg)
		} else {
			a.GetErrorHandler().ShowError(a.ctx, "❌ Undo failed for unknown reason")
		}
	}
}

// getUndoStatusMessage returns appropriate status message with refresh hints
func (a *App) getUndoStatusMessage(result *services.UndoResult) string {
	baseMessage := fmt.Sprintf("Undone: %s", result.Description)

	// Add refresh guidance based on operation type and current view
	if strings.Contains(result.Description, "Unarchived") && a.currentQuery == "" {
		return baseMessage // Auto-refreshes, no hint needed
	} else if strings.Contains(result.Description, "Restored from trash") && a.currentQuery == "" {
		return baseMessage // Auto-refreshes, no hint needed
	} else if strings.Contains(result.Description, "Unarchived") || strings.Contains(result.Description, "Restored from trash") {
		return baseMessage + " (Press R to refresh if not visible)"
	} else if strings.Contains(result.Description, "label") {
		return baseMessage // UI updates immediately, no refresh needed
	} else if strings.Contains(result.Description, "Marked as") {
		return baseMessage // UI updates immediately, no refresh needed
	} else {
		return baseMessage + " (Press R to refresh)"
	}
}

// smartUndoReload only reloads when necessary to show restored messages
func (a *App) smartUndoReload(result *services.UndoResult) {
	if a.logger != nil {
		a.logger.Printf("RELOAD: smartUndoReload called with ActionType=%s, Description=%s", result.ActionType, result.Description)
		if result.ExtraData != nil {
			a.logger.Printf("RELOAD: ExtraData=%+v", result.ExtraData)
		}
	}

	if result == nil {
		return
	}

	// Determine if we need to reload based on operation type and current view
	needsReload := false

	if result.ActionType == services.UndoActionArchive && a.currentQuery == "" {
		// Archive undo - restore message to inbox list
		a.restoreMessagesToInboxList(result.MessageIDs)
		a.QueueUpdateDraw(func() {
			a.reformatListItems()
		})
		return
	} else if result.ActionType == services.UndoActionTrash && a.currentQuery == "" {
		// Trash undo - restore message to inbox list
		a.restoreMessagesToInboxList(result.MessageIDs)
		a.QueueUpdateDraw(func() {
			a.reformatListItems()
		})
		return
	} else if strings.Contains(result.Description, "Marked as unread") || strings.Contains(result.Description, "Marked as read") {
		// Read state changes - update cache immediately and refresh UI formatting
		a.updateCacheAfterReadStateUndo(result)
		a.QueueUpdateDraw(func() {
			a.reformatListItems()
		})
		return
	} else if result.ActionType == services.UndoActionLabelAdd || result.ActionType == services.UndoActionLabelRemove {
		// Label changes - update cache immediately and refresh UI
		a.updateCacheAfterLabelUndo(result)
		a.QueueUpdateDraw(func() {
			a.reformatListItems()
			if a.labelsExpanded {
				currentMsg := a.GetCurrentMessageID()
				if currentMsg != "" {
					a.expandLabelsBrowse(currentMsg)
				}
			}
			// Refresh message content if a message is currently being displayed
			currentMessageID := a.GetCurrentMessageID()
			if currentMessageID != "" {
				// Re-render the message content to show updated labels
				go a.refreshMessageContent(currentMessageID)
			}
		})
		return
	} else if result.ActionType == services.UndoActionMove {
		// Move changes - use consistent UI-only restoration for all moves
		if a.logger != nil {
			a.logger.Printf("RELOAD: UndoActionMove - using UI-only restoration (no server reload)")
		}

		// All move undos use the same fast UI restoration logic
		a.restoreMessagesToInboxList(result.MessageIDs)
		a.updateCacheAfterMoveUndo(result)
		a.QueueUpdateDraw(func() {
			a.reformatListItems()
			if a.labelsExpanded {
				currentMsg := a.GetCurrentMessageID()
				if currentMsg != "" {
					a.expandLabelsBrowse(currentMsg)
				}
			}
		})
		return
	}

	if needsReload {
		if a.logger != nil {
			a.logger.Printf("RELOAD: needsReload=true, starting reload goroutine")
		}
		go func() {
			// Small delay to let success message show first
			time.Sleep(200 * time.Millisecond)
			if a.logger != nil {
				a.logger.Printf("RELOAD: Calling reloadMessages()")
			}
			a.reloadMessages()
		}()
	} else {
		if a.logger != nil {
			a.logger.Printf("RELOAD: needsReload=false, no reload triggered")
		}
	}
}

// restoreMessagesToInboxList restores messages to the inbox view after undo operations
func (a *App) restoreMessagesToInboxList(messageIDs []string) {
	// Logging removed for simplicity

	// Only restore if we're viewing INBOX (no search query means we're in inbox)
	if a.currentQuery != "" {
		// Logging removed for simplicity
		return
	}

	for _, messageID := range messageIDs {
		// Logging removed for simplicity

		// Check if message is already in the list
		found := false
		for _, existingID := range a.ids {
			if existingID == messageID {
				found = true
				// Logging removed for simplicity
				break
			}
		}

		if !found {
			// Fetch the message metadata using Gmail client directly
			message, err := a.Client.GetMessage(messageID)
			if err != nil {
				// Logging removed for simplicity
				continue
			}

			// Logging removed for simplicity
			// Add to front of list (most recent)
			a.ids = append([]string{messageID}, a.ids...)
			a.messagesMeta = append([]*gmailapi.Message{message}, a.messagesMeta...)

			// CRITICAL FIX: Also add the row to the UI table
			if table, ok := a.views["list"].(*tview.Table); ok {
				// Shift all existing rows down by 1
				rowCount := table.GetRowCount()
				for i := rowCount; i > 0; i-- {
					if i-1 >= 0 {
						cell := table.GetCell(i-1, 0)
						if cell != nil {
							table.SetCell(i, 0, cell)
						}
					}
				}

				// Add the new message at the top (row 0)
				text, _ := a.emailRenderer.FormatEmailList(message, a.getFormatWidth())
				cell := tview.NewTableCell(text).
					SetExpansion(1).
					SetBackgroundColor(a.GetComponentColors("general").Background.Color())
				table.SetCell(0, 0, cell)

				// Update table title to reflect new count
				table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))

				// Logging removed for simplicity
			}
		}
	}

	// Logging removed for simplicity
}

// updateCacheAfterReadStateUndo updates local cache immediately after read state undo operations
func (a *App) updateCacheAfterReadStateUndo(result *services.UndoResult) {
	// Logging removed for simplicity

	for _, messageID := range result.MessageIDs {

		// Determine what state to restore based on the undo description
		// Pattern matching works for both single ("Marked as unread") and bulk ("Marked as unread 2 messages") operations
		if strings.Contains(result.Description, "Marked as unread") {
			// We undid a mark-as-read, so restore to unread (add UNREAD label)
			// OBLITERATED: empty logger branch eliminated! 💥
			a.updateCachedMessageLabels(messageID, "UNREAD", true)
		} else if strings.Contains(result.Description, "Marked as read") {
			// We undid a mark-as-unread, so restore to read (remove UNREAD label)
			// OBLITERATED: empty logger branch eliminated! 💥
			a.updateCachedMessageLabels(messageID, "UNREAD", false)
		}
		// OBLITERATED: empty else branch eliminated! 💥
	}

	// Logging removed for simplicity
}

// updateCacheAfterMoveUndo updates local cache immediately after move undo operations
func (a *App) updateCacheAfterMoveUndo(result *services.UndoResult) {
	// Logging removed for simplicity

	if result.ExtraData == nil {
		// Logging removed for simplicity
		return
	}

	// Get label name mapping for cache updates
	_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
	labels, err := labelService.ListLabels(a.ctx)
	if err != nil {
		// Logging removed for simplicity
		return // Silently fail, will refresh from server later
	}
	labelIDToName := make(map[string]string)
	for _, label := range labels {
		labelIDToName[label.Id] = label.Name
	}
	// Logging removed for simplicity

	for _, messageID := range result.MessageIDs {
		// OBLITERATED: empty logger branch eliminated! 💥

		// Move undo: add back INBOX label and remove applied labels
		// OBLITERATED: empty logger branch eliminated! 💥
		a.updateCachedMessageLabels(messageID, "INBOX", true)
		a.updateMessageCacheLabels(messageID, "INBOX", true)

		// Remove the applied labels
		if appliedLabels, ok := result.ExtraData["applied_labels"].([]string); ok {
			// OBLITERATED: empty logger branch eliminated! 💥
			for _, labelID := range appliedLabels {
				// OBLITERATED: empty logger branch eliminated! 💥
				a.updateCachedMessageLabels(messageID, labelID, false)
				if labelName, exists := labelIDToName[labelID]; exists {
					// OBLITERATED: empty logger branch eliminated! 💥
					a.updateMessageCacheLabels(messageID, labelName, false)
				}
				// OBLITERATED: empty else branch eliminated! 💥
			}
		}
		// OBLITERATED: empty else branch eliminated! 💥
	}
	// Logging removed for simplicity
}

// updateCacheAfterLabelUndo updates local cache immediately after label undo operations
func (a *App) updateCacheAfterLabelUndo(result *services.UndoResult) {
	if result.ExtraData == nil {
		return
	}

	// Get label name mapping for cache updates
	_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
	labels, err := labelService.ListLabels(a.ctx)
	if err != nil {
		return // Silently fail, will refresh from server later
	}
	labelIDToName := make(map[string]string)
	for _, label := range labels {
		labelIDToName[label.Id] = label.Name
	}

	for _, messageID := range result.MessageIDs {
		switch result.ActionType { // OBLITERATED: converted to tagged switch! 💥
		case services.UndoActionLabelAdd:
			// We undid a label add, so we removed labels
			if labelsRemoved, ok := result.ExtraData["added_labels"].([]string); ok {
				for _, labelID := range labelsRemoved {
					a.updateCachedMessageLabels(messageID, labelID, false)
					if labelName, exists := labelIDToName[labelID]; exists {
						a.updateMessageCacheLabels(messageID, labelName, false)
					}
				}
			}
		case services.UndoActionLabelRemove:
			// We undid a label remove, so we added labels back
			if labelsAdded, ok := result.ExtraData["removed_labels"].([]string); ok {
				for _, labelID := range labelsAdded {
					a.updateCachedMessageLabels(messageID, labelID, true)
					if labelName, exists := labelIDToName[labelID]; exists {
						a.updateMessageCacheLabels(messageID, labelName, true)
					}
				}
			}
		}
	}
}

// Removed unused undo restore functions: handleUndoUIRestore, attemptArchiveUndoRestore, attemptTrashUndoRestore

// GetThemeService returns the theme service instance
func (a *App) GetThemeService() services.ThemeService {
	return a.themeService
}

// GetQueryService returns the query service instance
func (a *App) GetQueryService() services.QueryService {
	return a.queryService
}

// GetSlackService returns the Slack service instance
func (a *App) GetSlackService() services.SlackService {
	return a.slackService
}

// GetCurrentQuery returns the current search query
func (a *App) GetCurrentQuery() string {
	return a.currentQuery
}

// GetContentNavService returns the content navigation service instance
func (a *App) GetContentNavService() services.ContentNavigationService {
	return a.contentNavService
}

// applyTheme loads theme colors and updates the email renderer
func (a *App) applyTheme() {
	// Try to load theme from themes directory; fallback to defaults
	loader := config.NewThemeLoader("themes")
	if theme, err := loader.LoadThemeFromFile("gmail-dark.yaml"); err == nil {
		a.emailRenderer.UpdateFromConfig(theme)
	} else {
		def := config.DefaultColors()
		a.emailRenderer.UpdateFromConfig(def)
	}

	// Get component colors for widget updates
	generalColors := a.GetComponentColors("general")
	// Apply component-specific colors to existing widgets
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetBackgroundColor(generalColors.Background.Color())
	}
	if header, ok := a.views["header"].(*tview.TextView); ok {
		header.SetBackgroundColor(generalColors.Background.Color())
	}
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetBackgroundColor(generalColors.Background.Color())
	}
	if a.aiSummaryView != nil {
		aiColors := a.GetComponentColors("ai")
		a.aiSummaryView.SetBackgroundColor(aiColors.Background.Color())
	}
}

// applyThemeConfig applies a specific theme configuration to the app
func (a *App) applyThemeConfig(theme *config.ColorsConfig) error {
	if theme == nil {
		return fmt.Errorf("theme configuration is nil")
	}

	// Cache current theme for helper functions
	a.currentTheme = theme

	// Update email renderer with theme colors
	a.emailRenderer.UpdateColorer(
		a.GetStatusColor("progress"),          // UnreadColor - orange/progress color
		a.currentTheme.UI.FooterColor.Color(), // ReadColor - gray for read messages
		a.GetStatusColor("error"),             // ImportantColor - red for important
		a.GetStatusColor("success"),           // SentColor - green for sent
		a.GetStatusColor("warning"),           // DraftColor - yellow for drafts
		a.currentTheme.Body.FgColor.Color(),   // DefaultColor - theme text color
	)

	// Update flash border color with theme
	a.flash.UpdateBorderColor(a.currentTheme.UI.TitleColor.Color())

	// Update config if theme name is available
	if theme.Name != "" && a.Config != nil {
		a.Config.Theme.Current = theme.Name
		// Async save to avoid blocking UI
		go func() {
			if err := a.saveConfigAsync(); err != nil && a.logger != nil {
				a.logger.Printf("Failed to save theme preference: %v", err)
			}
		}()
	}

	// Update email renderer
	a.emailRenderer.UpdateFromConfig(theme)

	// Note: No longer setting global tview.Styles - using component-specific colors instead
	generalColors := a.GetComponentColors("general")

	// Update existing widget colors
	if list, ok := a.views["list"].(*tview.Table); ok {
		list.SetBackgroundColor(generalColors.Background.Color())
		// Update title color with the new theme
		list.SetTitleColor(a.GetComponentColors("general").Title.Color())
		// Force table to refresh content with new email renderer colors
		if len(a.messagesMeta) > 0 { // OBLITERATED: unnecessary nil check eliminated! 💥
			// Trigger reformatting of list items to apply new theme colors
			a.refreshTableDisplay()
		}
	}
	if header, ok := a.views["header"].(*tview.TextView); ok {
		header.SetBackgroundColor(generalColors.Background.Color())
	}
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetBackgroundColor(generalColors.Background.Color())
	}
	if a.aiSummaryView != nil {
		aiColors := a.GetComponentColors("ai")
		a.aiSummaryView.SetBackgroundColor(aiColors.Background.Color())
		a.aiSummaryView.SetTitleColor(a.GetComponentColors("ai").Title.Color())
	}
	// Update text container title color if it exists
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
	}
	// Update command panel title color if it exists
	if cmdPanel, ok := a.views["cmdPanel"].(*tview.Flex); ok {
		cmdPanel.SetTitleColor(a.GetComponentColors("general").Title.Color())
	}
	// Update slack widget title color if it exists
	if a.slackView != nil {
		a.slackView.SetTitleColor(a.GetComponentColors("slack").Title.Color())
	}

	// Update picker components that were missing theme re-application
	pickerComponents := []struct {
		viewName  string
		component string
	}{
		{"prompts", "prompts"},
		{"obsidian", "obsidian"},
		{"search", "search"},
		{"attachments", "attachments"},
		{"saved_queries", "saved_queries"},
		{"themes", "themes"},
		{"labels", "labels"},
		{"links", "links"},
		{"slack", "slack"},
	}

	for _, pc := range pickerComponents {
		if view, exists := a.views[pc.viewName]; exists {
			colors := a.GetComponentColors(pc.component)
			// Update background color for different view types
			if list, ok := view.(*tview.List); ok {
				list.SetBackgroundColor(colors.Background.Color())
			} else if table, ok := view.(*tview.Table); ok {
				table.SetBackgroundColor(colors.Background.Color())
			} else if flex, ok := view.(*tview.Flex); ok {
				flex.SetBackgroundColor(colors.Background.Color())
			}
		}
	}

	// Update status bar colors if it exists using hierarchical v2.0 theme
	if statusBar, ok := a.views["status"].(*tview.TextView); ok {
		statusBar.SetBackgroundColor(theme.Interaction.StatusBar.Bg.Color())
		statusBar.SetTextColor(theme.Interaction.StatusBar.Fg.Color())
	}

	// Update composition panel theme if it exists
	if a.compositionPanel != nil {
		a.compositionPanel.UpdateTheme()
	}

	// Refresh borders for Flex containers that have been forced to use filled backgrounds
	// This ensures consistent border rendering when themes change
	a.RefreshBordersForFilledFlexes()

	return nil
}

// saveConfigAsync saves the configuration asynchronously
func (a *App) saveConfigAsync() error {
	if a.Config == nil {
		return fmt.Errorf("config is nil")
	}
	configPath := config.DefaultConfigPath()
	return a.Config.SaveConfig(configPath)
}

// Theme-aware color helper functions

// getDefaultTheme returns a minimal default theme for fallback purposes
func (a *App) getDefaultTheme() *config.ColorsConfig {
	return &config.ColorsConfig{
		Name:        "Default",
		Description: "Built-in fallback theme",
		Version:     "2.0",
		Foundation: config.FoundationColors{
			Background: config.Color("#000000"), // Black background
			Foreground: config.Color("#ffffff"), // White text
			Border:     config.Color("#808080"), // Gray borders
			Focus:      config.Color("#0080ff"), // Blue focus
		},
		Semantic: config.SemanticColors{
			Primary:   config.Color("#ffff00"), // Yellow primary
			Secondary: config.Color("#808080"), // Gray secondary
			Accent:    config.Color("#0080ff"), // Blue accent
			Success:   config.Color("#00ff00"), // Green success
			Warning:   config.Color("#ffff00"), // Yellow warning
			Error:     config.Color("#ff0000"), // Red error
			Info:      config.Color("#0080ff"), // Blue info
		},
	}
}

// getComponentColor resolves a color using the hierarchical theme system
func (a *App) getComponentColor(component config.ComponentType, colorType config.ColorType) tcell.Color {
	if a.currentTheme == nil {
		// Use default theme fallback colors instead of hardcoded tcell.Color constants
		fallbackTheme := a.getDefaultTheme()
		return fallbackTheme.GetComponentColor(component, colorType).Color()
	}
	return a.currentTheme.GetComponentColor(component, colorType).Color()
}

// getHintColor returns the theme's hint color or fallback to gray
func (a *App) getHintColor() tcell.Color {
	return a.getComponentColor(config.ComponentTypeGeneral, config.ColorTypeSecondary)
}

// getSelectionStyle returns the theme's cursor selection style or fallback
func (a *App) getSelectionStyle() tcell.Style {
	if a.currentTheme == nil {
		// Use default theme fallback instead of hardcoded colors
		fallbackTheme := a.getDefaultTheme()
		bgColor, fgColor := fallbackTheme.GetCursorSelectionColors()
		if bgColor == "" || fgColor == "" {
			// If no selection colors in default theme, use foundation colors
			bgColor = fallbackTheme.Foundation.Focus      // Blue background
			fgColor = fallbackTheme.Foundation.Foreground // White text
		}
		return tcell.StyleDefault.Foreground(fgColor.Color()).Background(bgColor.Color())
	}
	bgColor, fgColor := a.currentTheme.GetCursorSelectionColors()
	if bgColor == "" || fgColor == "" {
		// Legacy fallback
		bgColor = a.currentTheme.UI.SelectionBgColor
		fgColor = a.currentTheme.UI.SelectionFgColor
	}
	return tcell.StyleDefault.Foreground(fgColor.Color()).Background(bgColor.Color())
}

// OBLITERATED: unused getBulkSelectionStyle function eliminated! 💥

// OBLITERATED: unused getLabelColor function eliminated! 💥

// getMessageHeaderColor returns the theme's header color for email message headers
func (a *App) getMessageHeaderColor() tcell.Color {
	return a.getComponentColor(config.ComponentTypeGeneral, config.ColorTypeAccent)
}

// getStatusColor returns theme-aware colors for different status levels
func (a *App) getStatusColor(level string) tcell.Color {
	return a.GetStatusColor(level) // Use the new helper function
}

// OBLITERATED: Unused component-specific color methods eliminated! 💥
// All 6 unused color functions removed - use GetComponentColors() instead

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

	// Show current status
	if a.Config != nil && a.Config.Theme.Current != "" {
		help.WriteString(fmt.Sprintf("🎨 Theme: %s\n", a.Config.Theme.Current))
	}
	if a.LLM != nil {
		help.WriteString("🤖 AI: Enabled\n")
	}

	// Add separator line before navigation instructions
	help.WriteString("\n")
	help.WriteString("📖 NAVIGATION: Use /term to search, n/N for next/previous match, g/gg/G for navigation\n")
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("💡 Press '%s' or 'Esc' to return to main view\n\n", a.Keys.Help))

	// Quick Start Section
	help.WriteString("🚀 GETTING STARTED\n\n")
	help.WriteString(fmt.Sprintf("    %-8s  ❓  Toggle this help screen\n", a.Keys.Help))
	help.WriteString(fmt.Sprintf("    %-8s  👁️   View selected message\n", "Enter"))
	help.WriteString(fmt.Sprintf("    %-8s  🚪  Quit application\n", a.Keys.Quit))
	help.WriteString(fmt.Sprintf("    %-8s  💻  Command mode (type commands like :search, :help)\n\n", a.Keys.CommandMode))

	// Essential Operations
	help.WriteString("📧 MESSAGE BASICS\n\n")
	help.WriteString(fmt.Sprintf("    %-8s  💬  Reply to message\n", a.Keys.Reply))
	help.WriteString(fmt.Sprintf("    %-8s  👥  Reply to all recipients\n", a.Keys.ReplyAll))
	help.WriteString(fmt.Sprintf("    %-8s  ➡️   Forward message\n", a.Keys.Forward))
	help.WriteString(fmt.Sprintf("    %-8s  ✏️   Compose new message\n", a.Keys.Compose))
	help.WriteString(fmt.Sprintf("    %-8s  📁  Archive message\n", a.Keys.Archive))
	help.WriteString(fmt.Sprintf("    %-8s  🗑️   Move to trash\n", a.Keys.Trash))
	help.WriteString(fmt.Sprintf("    %-8s  👁️   Toggle read/unread\n", a.Keys.ToggleRead))
	help.WriteString(fmt.Sprintf("    %-8s  ↩️   Undo last action\n", a.Keys.Undo))
	help.WriteString(fmt.Sprintf("    %-8s  📦  Move message to folder\n", a.Keys.Move))
	help.WriteString(fmt.Sprintf("    %-8s  🔖  Manage labels\n", a.Keys.ManageLabels))
	help.WriteString(fmt.Sprintf("    %-8s  📝  View drafts\n\n", a.Keys.Drafts))

	// Navigation & Search
	help.WriteString("🧭 NAVIGATION & SEARCH\n\n")
	help.WriteString(fmt.Sprintf("    %-8s  🔄  Refresh messages\n", a.Keys.Refresh))
	help.WriteString(fmt.Sprintf("    %-8s  🔍  Search messages\n", a.Keys.Search))
	help.WriteString(fmt.Sprintf("    %-8s  ⬇️   Load next 50 messages\n", a.Keys.LoadMore))
	help.WriteString(fmt.Sprintf("    %-8s  🔴  Show unread messages\n", a.Keys.Unread))
	help.WriteString(fmt.Sprintf("    %-8s  📝  View drafts\n", a.Keys.Drafts))
	help.WriteString(fmt.Sprintf("    %-8s  📎  Attachment picker (view/download message attachments)\n", a.Keys.Attachments))
	help.WriteString(fmt.Sprintf("    %-8s  📫  Quick search: from current sender\n", a.Keys.SearchFrom))
	help.WriteString(fmt.Sprintf("    %-8s  📤  Quick search: to current sender (includes Sent)\n", a.Keys.SearchTo))
	help.WriteString(fmt.Sprintf("    %-8s  🧵  Quick search: by current subject\n", a.Keys.SearchSubject))
	help.WriteString(fmt.Sprintf("    %-8s  📦  Quick search: archived messages\n", a.Keys.Archived))
	help.WriteString("\n")

	// Content Navigation
	help.WriteString("📖 CONTENT NAVIGATION (When Viewing Message)\n\n")
	help.WriteString(fmt.Sprintf("    %-8s  🔍  Search within message content\n", a.Keys.ContentSearch))
	help.WriteString(fmt.Sprintf("    %-8s  ➡️   Next search match\n", a.Keys.SearchNext))
	help.WriteString(fmt.Sprintf("    %-8s  ⬅️   Previous search match\n", a.Keys.SearchPrev))
	help.WriteString(fmt.Sprintf("    %-8s  ⬆️   Go to top of message\n", a.Keys.GotoTop))
	help.WriteString(fmt.Sprintf("    %-8s  ⬇️   Go to bottom of message\n", a.Keys.GotoBottom))
	help.WriteString(fmt.Sprintf("    %-8s  🚀  Fast scroll up\n", a.Keys.FastUp))
	help.WriteString(fmt.Sprintf("    %-8s  🚀  Fast scroll down\n", a.Keys.FastDown))
	help.WriteString(fmt.Sprintf("    %-8s  ⬅️   Word left\n", a.Keys.WordLeft))
	help.WriteString(fmt.Sprintf("    %-8s  ➡️   Word right\n", a.Keys.WordRight))
	help.WriteString(fmt.Sprintf("    %-8s  📄  Toggle header visibility\n\n", a.Keys.ToggleHeaders))

	// Bulk Operations
	bulkStatus := "OFF"
	if a.bulkMode {
		bulkStatus = fmt.Sprintf("ON (%d selected)", len(a.selected))
	}
	help.WriteString(fmt.Sprintf("📦 BULK OPERATIONS (Currently: %s)\n\n", bulkStatus))
	help.WriteString(fmt.Sprintf("    %-8s  ✅  Enter bulk mode\n", a.Keys.BulkMode))
	help.WriteString(fmt.Sprintf("    %-8s  ➕  Toggle message selection (in bulk mode)\n", a.Keys.BulkSelect))
	help.WriteString("    *         🌟  Select all visible messages\n")
	help.WriteString(fmt.Sprintf("    %-8s  📁  Archive selected messages\n", a.Keys.Archive))
	help.WriteString(fmt.Sprintf("    %-8s  🗑️   Delete selected messages\n", a.Keys.Trash))
	help.WriteString(fmt.Sprintf("    %-8s  📦  Move selected messages\n", a.Keys.Move))
	help.WriteString(fmt.Sprintf("    %-8s  🎯  Apply bulk prompt to selected\n", a.Keys.Prompt))
	if a.Config.Slack.Enabled {
		help.WriteString(fmt.Sprintf("    %-8s  💬  Forward selected to Slack\n", a.Keys.Slack))
	}
	if a.Config.Obsidian.Enabled {
		help.WriteString(fmt.Sprintf("    %-8s  📝  Send selected to Obsidian (with repopack option)\n", a.Keys.Obsidian))
	}
	help.WriteString("    Esc       ❌  Exit bulk mode\n\n")

	// AI Features (if enabled)
	if a.LLM != nil {
		help.WriteString("🤖 AI FEATURES (✅ Available)\n\n")
		help.WriteString(fmt.Sprintf("    %-8s  📝  Summarize message\n", a.Keys.Summarize))
		help.WriteString("    Y         🔄  Regenerate summary (force refresh)\n")
		help.WriteString(fmt.Sprintf("    %-8s  🎯  Open Prompt Library\n", a.Keys.Prompt))
		help.WriteString(fmt.Sprintf("    %-8s  🤖  Generate reply draft\n", a.Keys.GenerateReply))
		help.WriteString(fmt.Sprintf("    %-8s  🔖  AI suggest label\n\n", a.Keys.SuggestLabel))
	}

	// Threading Features (if enabled)
	if a.IsThreadingEnabled() {
		threadingStatus := "flat"
		if a.GetCurrentThreadViewMode() == ThreadViewThread {
			threadingStatus = "threaded"
		}
		help.WriteString(fmt.Sprintf("🧵 MESSAGE THREADING (Current: %s)\n\n", threadingStatus))
		help.WriteString(fmt.Sprintf("    %-8s  🔄  Toggle between thread and flat view\n", a.Keys.ToggleThreading))
		help.WriteString(fmt.Sprintf("    %-8s  📂  Expand/collapse thread (when in thread view)\n", a.Keys.ExpandThread))
		help.WriteString(fmt.Sprintf("    %-8s  📤  Expand all threads\n", a.Keys.ExpandAllThreads))
		help.WriteString(fmt.Sprintf("    %-8s  📥  Collapse all threads\n", a.Keys.CollapseAllThreads))
		if a.LLM != nil {
			help.WriteString(fmt.Sprintf("    %-8s  🧵  Generate AI summary of thread\n", a.Keys.ThreadSummary))
		}
		help.WriteString(fmt.Sprintf("    %-8s  ⬆️   Navigate to next thread\n", a.Keys.NextThread))
		help.WriteString(fmt.Sprintf("    %-8s  ⬇️   Navigate to previous thread\n\n", a.Keys.PrevThread))
	}

	// VIM Power Operations
	help.WriteString("⚡ VIM POWER OPERATIONS\n\n")
	help.WriteString("    Pattern:  {operation}{count}{operation} (e.g., s5s, a3a, d7d)\n\n")
	help.WriteString(fmt.Sprintf("    %s5%s       ✅  Select next 5 messages\n", a.Keys.BulkSelect, a.Keys.BulkSelect))
	help.WriteString(fmt.Sprintf("    %s3%s       📁  Archive next 3 messages\n", a.Keys.Archive, a.Keys.Archive))
	help.WriteString(fmt.Sprintf("    %s7%s       🗑️   Delete next 7 messages\n", a.Keys.Trash, a.Keys.Trash))
	help.WriteString(fmt.Sprintf("    %s5%s       👁️   Toggle read status for next 5 messages\n", a.Keys.ToggleRead, a.Keys.ToggleRead))
	help.WriteString(fmt.Sprintf("    %s4%s       📦  Move next 4 messages\n", a.Keys.Move, a.Keys.Move))
	help.WriteString(fmt.Sprintf("    %s6%s       🔖  Label next 6 messages\n", a.Keys.ManageLabels, a.Keys.ManageLabels))
	if a.Config.Slack.Enabled {
		help.WriteString(fmt.Sprintf("    %s3%s       💬  Send next 3 messages to Slack\n", a.Keys.Slack, a.Keys.Slack))
	}
	if a.Config.Obsidian.Enabled {
		help.WriteString(fmt.Sprintf("    %s2%s       📝  Send next 2 messages to Obsidian\n", a.Keys.Obsidian, a.Keys.Obsidian))
	}
	if a.LLM != nil {
		help.WriteString(fmt.Sprintf("    %s8%s       🤖  Apply AI prompts to next 8 messages\n", a.Keys.Prompt, a.Keys.Prompt))
	}
	help.WriteString(fmt.Sprintf("    %-8s  ⬆️   Go to first message\n", a.Keys.GotoTop))
	help.WriteString(fmt.Sprintf("    %-8s  ⬇️   Go to last message\n\n", a.Keys.GotoBottom))

	// Additional Features
	help.WriteString("🔧 ADDITIONAL FEATURES\n\n")
	help.WriteString(fmt.Sprintf("    %-8s  💾  Save current search as bookmark\n", a.Keys.SaveQuery))
	help.WriteString(fmt.Sprintf("    %-8s  📚  Browse saved query bookmarks\n", a.Keys.QueryBookmarks))
	help.WriteString(fmt.Sprintf("    %-8s  🌐  Open message in Gmail web\n", a.Keys.OpenGmail))
	help.WriteString(fmt.Sprintf("    %-8s  💾  Save message content\n", a.Keys.SaveMessage))
	help.WriteString(fmt.Sprintf("    %-8s  📄  Save raw message\n", a.Keys.SaveRaw))
	help.WriteString(fmt.Sprintf("    %-8s  📅  RSVP to calendar event\n", a.Keys.RSVP))
	help.WriteString(fmt.Sprintf("    %-8s  🔗  Link picker (view/open message links)\n", a.Keys.LinkPicker))
	help.WriteString(fmt.Sprintf("    %-8s  🎨  Theme picker & preview\n", a.Keys.ThemePicker))
	if a.Config.Obsidian.Enabled {
		help.WriteString(fmt.Sprintf("    %-8s  📝  Send to Obsidian (individual files or repopack)\n", a.Keys.Obsidian))
	}
	if a.Config.Slack.Enabled {
		help.WriteString(fmt.Sprintf("    %-8s  💬  Forward to Slack\n", a.Keys.Slack))
	}
	help.WriteString(fmt.Sprintf("    %-8s  📋  Export as Markdown\n\n", a.Keys.Markdown))

	// Command Equivalents
	help.WriteString("💻 COMMAND EQUIVALENTS\n\n")
	help.WriteString("    Every keyboard shortcut has a command equivalent:\n\n")
	help.WriteString(fmt.Sprintf("    %-18s ✅  Same as %s5%s (select next 5)\n", ":select 5", a.Keys.BulkSelect, a.Keys.BulkSelect))
	help.WriteString(fmt.Sprintf("    %-18s 📁  Same as %s3%s (archive next 3)\n", ":archive 3", a.Keys.Archive, a.Keys.Archive))
	help.WriteString(fmt.Sprintf("    %-18s 🗑️   Same as %s7%s (delete next 7)\n", ":trash 7", a.Keys.Trash, a.Keys.Trash))
	help.WriteString(fmt.Sprintf("    %-18s ↩️   Same as %s (undo last action)\n", ":undo", a.Keys.Undo))
	help.WriteString(fmt.Sprintf("    %-18s ✏️   Same as %s (compose new message)\n", ":compose", a.Keys.Compose))
	help.WriteString(fmt.Sprintf("    %-18s 💬  Same as %s (reply to message)\n", ":reply", a.Keys.Reply))
	help.WriteString(fmt.Sprintf("    %-18s 👥  Same as %s (reply to all recipients)\n", ":reply-all", a.Keys.ReplyAll))
	help.WriteString(fmt.Sprintf("    %-18s 👥  Same as :reply-all (reply to all)\n", ":ra"))
	help.WriteString(fmt.Sprintf("    %-18s ➡️   Same as %s (forward message)\n", ":forward", a.Keys.Forward))
	help.WriteString(fmt.Sprintf("    %-18s ➡️   Same as :forward (forward message)\n", ":f"))
	help.WriteString(fmt.Sprintf("    %-18s 📝  Same as %s (view drafts)\n", ":drafts", a.Keys.Drafts))
	help.WriteString(fmt.Sprintf("    %-18s 📝  Same as :drafts (view drafts)\n", ":dr"))
	help.WriteString(fmt.Sprintf("    %-18s ✏️   Same as :compose (compose new message)\n", ":new"))
	help.WriteString(fmt.Sprintf("    %-18s 🔍  Search for 'term'\n", ":search term"))
	help.WriteString(fmt.Sprintf("    %-18s 💾  Save current search as bookmark\n", ":save-query"))
	help.WriteString(fmt.Sprintf("    %-18s 📚  Browse saved query bookmarks\n", ":bookmarks"))
	help.WriteString(fmt.Sprintf("    %-18s 🔍  Execute saved query by name\n", ":bookmark name"))
	if a.Config.Obsidian.Enabled {
		help.WriteString(fmt.Sprintf("    %-18s 📦  Create repopack with selected messages\n", ":obsidian repack"))
		help.WriteString(fmt.Sprintf("    %-18s 📦  Same as :obsidian repack (short alias)\n", ":obs repack"))
	}
	help.WriteString(fmt.Sprintf("    %-18s 🎨  Open theme picker\n", ":theme"))
	help.WriteString(fmt.Sprintf("    %-18s 📄  Toggle header visibility\n", ":headers"))
	help.WriteString(fmt.Sprintf("    %-18s 🔢  Toggle message numbers\n", ":numbers"))

	// Threading commands (if enabled)
	if a.IsThreadingEnabled() {
		help.WriteString(fmt.Sprintf("    %-18s 🧵  Same as %s (switch to threaded view)\n", ":threads", a.Keys.ToggleThreading))
		help.WriteString(fmt.Sprintf("    %-18s 📄  Same as %s (switch to flat view)\n", ":flatten", a.Keys.ToggleThreading))
		help.WriteString(fmt.Sprintf("    %-18s 📤  Same as %s (expand all threads)\n", ":expand-all", a.Keys.ExpandAllThreads))
		help.WriteString(fmt.Sprintf("    %-18s 📥  Same as %s (collapse all threads)\n", ":collapse-all", a.Keys.CollapseAllThreads))
		if a.LLM != nil {
			help.WriteString(fmt.Sprintf("    %-18s 🧵  Same as %s (generate thread summary)\n", ":thread-summary", a.Keys.ThreadSummary))
		}
	}

	// Performance commands
	help.WriteString(fmt.Sprintf("    %-18s ⚡  Show preloading status and statistics\n", ":preload status"))
	help.WriteString(fmt.Sprintf("    %-18s 🚀  Enable background preloading\n", ":preload on"))
	help.WriteString(fmt.Sprintf("    %-18s ⏸️   Disable background preloading\n", ":preload off"))
	help.WriteString(fmt.Sprintf("    %-18s 🧹  Clear all preloaded caches\n", ":preload clear"))

	// Prompt management commands
	help.WriteString(fmt.Sprintf("    %-18s 📊  Show prompt usage statistics\n", ":prompt stats"))
	help.WriteString(fmt.Sprintf("    %-18s 📋  Manage prompts\n", ":prompt list"))
	help.WriteString(fmt.Sprintf("    %-18s ➕  Create new prompt\n", ":prompt create"))
	help.WriteString(fmt.Sprintf("    %-18s ✏️   Update existing prompt\n", ":prompt update"))
	help.WriteString(fmt.Sprintf("    %-18s 🗑️   Delete prompt\n", ":prompt delete"))
	help.WriteString(fmt.Sprintf("    %-18s 📤  Export prompts\n", ":prompt export"))
	help.WriteString(fmt.Sprintf("    %-18s ❓  Show this help\n\n", ":help"))

	// Footer with tips
	help.WriteString("💡 TIPS\n\n")
	help.WriteString("    • All shortcuts are configurable in ~/.config/giztui/config.json\n")
	help.WriteString("    • Use Tab to cycle between panes (list ↔ content)\n")
	help.WriteString("    • Press Esc to cancel most operations or exit modes\n")
	help.WriteString("    • VIM range operations work with any action (s5s, a3a, d7d, etc.)\n")
	help.WriteString("    • Content search (/) highlights matches and enables n/N navigation\n")
	help.WriteString("    • Bulk mode allows selecting multiple messages for batch operations\n")

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
						// Only re-render welcome with account email if still loading (no messages loaded yet)
						// This prevents overwriting message content with welcome screen after parallel loading completes
						if text, ok := a.views["text"].(*tview.TextView); ok {
							currentMsgID := a.GetCurrentMessageID()
							if len(a.ids) == 0 && currentMsgID == "" {
								text.SetText(a.buildWelcomeText(true, a.welcomeEmail, 0))
							}
							// Otherwise, don't overwrite existing message content
						}
						// Always refresh status bar baseline to include the email
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
// getActiveAccountEmail returns the current account email if available.
func (a *App) getActiveAccountEmail() string {
	if email, err := a.Client.ActiveAccountEmail(a.ctx); err == nil && email != "" {
		return email
	}
	return "user@example.com" // fallback for when account email can't be retrieved
}

// (moved to keys.go) bindKeys

// handleCommandInput handles input when in command mode
// (moved to commands.go) handleCommandInput

// updateCommandBar updates the command bar display
// (moved to commands.go) updateCommandBar

// generateCommandSuggestion generates a suggestion based on the current command buffer
// (moved to commands.go) generateCommandSuggestion

// completeCommand completes the current command with the suggestion
// (moved to commands.go) completeCommand

// toggleHelp toggles the help display in the message content area
func (a *App) toggleHelp() {
	if a.showHelp {
		// Restore previous content
		a.showHelp = false

		// Restore text content through enhanced text view
		if a.enhancedTextView != nil && a.helpBackupText != "" {
			a.enhancedTextView.SetContent(a.helpBackupText)
			a.enhancedTextView.SetDynamicColors(true)
			a.enhancedTextView.ScrollToBeginning()
		} else {
			// Fallback to regular text view
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				text.SetText(a.helpBackupText)
				text.ScrollToBeginning()
			}
		}

		// Restore header content and visibility
		if header, ok := a.views["header"].(*tview.TextView); ok {
			header.SetDynamicColors(true)
			header.SetText(a.helpBackupHeader)
		}

		// Restore header height (make it visible again)
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			if header, ok := a.views["header"].(*tview.TextView); ok {
				textContainer.ResizeItem(header, a.originalHeaderHeight, 0)
			}
		}

		// Restore text container title
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			textContainer.SetTitle(a.helpBackupTitle)
			textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
		}

		// Clear backup content
		a.helpBackupText = ""
		a.helpBackupHeader = ""
		a.helpBackupTitle = ""

		// Update focus state and set focus to text view (unless composer is active)
		if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
			a.currentFocus = "text"
			a.SetFocus(a.views["text"])
			a.updateFocusIndicators("text")
		}
	} else {
		// Save current content before showing help
		if text, ok := a.views["text"].(*tview.TextView); ok {
			a.helpBackupText = text.GetText(false)
		}
		if header, ok := a.views["header"].(*tview.TextView); ok {
			a.helpBackupHeader = header.GetText(false)
		}
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			a.helpBackupTitle = textContainer.GetTitle()
		}

		// Show help content
		a.showHelp = true

		// Store current header height and hide header section
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			if header, ok := a.views["header"].(*tview.TextView); ok {
				// Calculate current header height before hiding it
				headerContent := header.GetText(false)
				a.originalHeaderHeight = a.calculateHeaderHeight(headerContent)

				// Clear header content and hide it completely
				header.SetDynamicColors(true)
				header.SetText("")
				textContainer.ResizeItem(header, 0, 0)
			}
		}

		// Display help title in text container border
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			textContainer.SetTitle(" 📚 Help & Shortcuts ")
			textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
		}

		// Display help content in enhanced text view with proper content setting
		helpContent := a.generateHelpText()
		if a.enhancedTextView != nil {
			a.enhancedTextView.SetContent(helpContent)
			a.enhancedTextView.SetDynamicColors(true)
			a.enhancedTextView.ScrollToBeginning()
		} else {
			// Fallback to regular text view if enhanced view not available
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				text.SetText(helpContent)
				text.ScrollToBeginning()
			}
		}

		// Update focus state and set focus to text view so users can search immediately (unless composer is active)
		if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
			a.currentFocus = "text"
			a.SetFocus(a.views["text"])
			a.updateFocusIndicators("text")
		}
	}
}

// showPreloadStatus displays preload status in full screen using help screen pattern
func (a *App) showPreloadStatus(statusContent string) {
	// Save current content before showing preload status
	if text, ok := a.views["text"].(*tview.TextView); ok {
		a.preloadBackupText = text.GetText(false)
	}
	if header, ok := a.views["header"].(*tview.TextView); ok {
		a.preloadBackupHeader = header.GetText(false)
	}
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		a.preloadBackupTitle = textContainer.GetTitle()
	}

	// Show preload status
	a.preloadStatusVisible = true

	// Store current header height and hide header section
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		if header, ok := a.views["header"].(*tview.TextView); ok {
			// Calculate current header height before hiding it
			headerContent := header.GetText(false)
			a.originalHeaderHeight = a.calculateHeaderHeight(headerContent)

			// Clear header content and hide it completely
			header.SetDynamicColors(true)
			header.SetText("")
			textContainer.ResizeItem(header, 0, 0)
		}
	}

	// Display preload status title in text container border
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitle(" 📦 Preloader Status ")
		textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
	}

	// Display preload status content in enhanced text view with proper content setting
	if a.enhancedTextView != nil {
		a.enhancedTextView.SetContent(statusContent)
		a.enhancedTextView.SetDynamicColors(true)
		a.enhancedTextView.ScrollToBeginning()
	} else {
		// Fallback to regular text view if enhanced view not available
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetDynamicColors(true)
			text.Clear()
			text.SetText(statusContent)
			text.ScrollToBeginning()
		}
	}

	// Update focus state and set focus to text view (unless composer is active)
	if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
		a.currentFocus = "text"
		a.SetFocus(a.views["text"])
		// Use QueueUpdateDraw only for focus indicators since we're now in goroutine context
		a.QueueUpdateDraw(func() {
			a.updateFocusIndicators("text")
		})
	}
}

// hidePreloadStatus hides the preload status screen and restores previous content
func (a *App) hidePreloadStatus() {
	if !a.preloadStatusVisible {
		return
	}

	// Restore previous content
	a.preloadStatusVisible = false

	// Restore text content through enhanced text view
	if a.enhancedTextView != nil && a.preloadBackupText != "" {
		a.enhancedTextView.SetContent(a.preloadBackupText)
		a.enhancedTextView.SetDynamicColors(true)
		a.enhancedTextView.ScrollToBeginning()
	} else {
		// Fallback to regular text view
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetDynamicColors(true)
			text.Clear()
			text.SetText(a.preloadBackupText)
			text.ScrollToBeginning()
		}
	}

	// Restore header content and visibility
	if header, ok := a.views["header"].(*tview.TextView); ok {
		header.SetDynamicColors(true)
		header.SetText(a.preloadBackupHeader)
	}

	// Restore header height (make it visible again)
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		if header, ok := a.views["header"].(*tview.TextView); ok {
			textContainer.ResizeItem(header, a.originalHeaderHeight, 0)
		}
	}

	// Restore text container title
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitle(a.preloadBackupTitle)
		textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
	}

	// Clear backup content
	a.preloadBackupText = ""
	a.preloadBackupHeader = ""
	a.preloadBackupTitle = ""

	// Update focus state and set focus to text view (unless composer is active)
	if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
		a.currentFocus = "text"
		a.SetFocus(a.views["text"])
		a.updateFocusIndicators("text")
	}
}

// showPromptStats displays prompt usage statistics in full screen using help screen pattern
func (a *App) showPromptStats(stats *services.UsageStats) {
	// Save current content before showing prompt stats
	if text, ok := a.views["text"].(*tview.TextView); ok {
		a.promptStatsBackupText = text.GetText(false)
	}
	if header, ok := a.views["header"].(*tview.TextView); ok {
		a.promptStatsBackupHeader = header.GetText(false)
	}
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		a.promptStatsBackupTitle = textContainer.GetTitle()
	}

	// Show prompt stats
	a.promptStatsVisible = true

	// Store current header height and hide header section
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		if header, ok := a.views["header"].(*tview.TextView); ok {
			// Calculate current header height before hiding it
			headerContent := header.GetText(false)
			a.originalHeaderHeight = a.calculateHeaderHeight(headerContent)

			// Clear header content and hide it completely
			header.SetDynamicColors(true)
			header.SetText("")
			textContainer.ResizeItem(header, 0, 0)
		}
	}

	// Display prompt stats title in text container border
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitle(" 📊 Prompt Usage Statistics ")
		textContainer.SetTitleColor(a.GetComponentColors("stats").Title.Color())
	}

	// Generate statistics content
	statsContent := a.generatePromptStatsContent(stats)

	// Display prompt stats content in enhanced text view with proper content setting
	if a.enhancedTextView != nil {
		a.enhancedTextView.SetContent(statsContent)
		a.enhancedTextView.SetDynamicColors(true)
		a.enhancedTextView.ScrollToBeginning()
	} else {
		// Fallback to regular text view if enhanced view not available
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetDynamicColors(true)
			text.Clear()
			text.SetText(statsContent)
			text.ScrollToBeginning()
		}
	}

	// Update focus state and set focus to text view (unless composer is active)
	if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
		a.currentFocus = "text"
		a.SetFocus(a.views["text"])
		// Use QueueUpdateDraw only for focus indicators since we're now in goroutine context
		a.QueueUpdateDraw(func() {
			a.updateFocusIndicators("text")
		})
	}
}

// hidePromptStats hides the prompt stats screen and restores previous content
func (a *App) hidePromptStats() {
	if !a.promptStatsVisible {
		return
	}

	// Restore previous content
	a.promptStatsVisible = false

	// Restore text content through enhanced text view
	if a.enhancedTextView != nil && a.promptStatsBackupText != "" {
		a.enhancedTextView.SetContent(a.promptStatsBackupText)
		a.enhancedTextView.SetDynamicColors(true)
		a.enhancedTextView.ScrollToBeginning()
	} else {
		// Fallback to regular text view
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetDynamicColors(true)
			text.Clear()
			text.SetText(a.promptStatsBackupText)
			text.ScrollToBeginning()
		}
	}

	// Restore header content and visibility
	if header, ok := a.views["header"].(*tview.TextView); ok {
		header.SetDynamicColors(true)
		header.SetText(a.promptStatsBackupHeader)
	}

	// Restore header height (make it visible again)
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		if header, ok := a.views["header"].(*tview.TextView); ok {
			textContainer.ResizeItem(header, a.originalHeaderHeight, 0)
		}
	}

	// Restore text container title
	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitle(a.promptStatsBackupTitle)
		textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
	}

	// Clear backup content
	a.promptStatsBackupText = ""
	a.promptStatsBackupHeader = ""
	a.promptStatsBackupTitle = ""

	// Update focus state and set focus to text view (unless composer is active)
	if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
		a.currentFocus = "text"
		a.SetFocus(a.views["text"])
		a.updateFocusIndicators("text")
	}
}

// generatePromptStatsContent generates the content for prompt statistics display
func (a *App) generatePromptStatsContent(stats *services.UsageStats) string {
	var content strings.Builder

	// Summary section
	content.WriteString("📊 USAGE SUMMARY\n\n")
	content.WriteString(fmt.Sprintf("Total Prompt Uses: %d\n", stats.TotalUsage))
	content.WriteString(fmt.Sprintf("Active Prompts: %d\n", stats.UniquePrompts))
	content.WriteString(fmt.Sprintf("Favorite Prompts: %d\n", len(stats.FavoritePrompts)))
	if !stats.LastUsed.IsZero() {
		content.WriteString(fmt.Sprintf("Last Used: %s\n", stats.LastUsed.Format("2006-01-02 15:04")))
	}
	content.WriteString("\n")

	// Top prompts section
	if len(stats.TopPrompts) > 0 {
		content.WriteString("🏆 TOP PROMPTS\n\n")
		for i, prompt := range stats.TopPrompts {
			icon := "📝"
			switch prompt.Category {
			case "bulk_analysis":
				icon = "🚀"
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			}

			favoriteIcon := ""
			if prompt.IsFavorite {
				favoriteIcon = " ⭐"
			}

			content.WriteString(fmt.Sprintf("%d. %s %s%s\n", i+1, icon, prompt.Name, favoriteIcon))
			content.WriteString(fmt.Sprintf("    Uses: %d | Category: %s | Last: %s\n",
				prompt.UsageCount, prompt.Category, prompt.LastUsed))
			content.WriteString("\n")
		}
	} else {
		content.WriteString("🏆 TOP PROMPTS\n\n")
		content.WriteString("No prompt usage recorded yet.\n")
		content.WriteString("Start using prompts to see statistics here!\n\n")
	}

	// Favorites section (if different from top)
	if len(stats.FavoritePrompts) > 0 && len(stats.FavoritePrompts) != len(stats.TopPrompts) {
		content.WriteString("⭐ FAVORITE PROMPTS\n\n")
		for _, prompt := range stats.FavoritePrompts {
			icon := "📝"
			switch prompt.Category {
			case "bulk_analysis":
				icon = "🚀"
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			}

			content.WriteString(fmt.Sprintf("• %s %s\n", icon, prompt.Name))
			content.WriteString(fmt.Sprintf("  Uses: %d | Category: %s\n",
				prompt.UsageCount, prompt.Category))
		}
		content.WriteString("\n")
	}

	// Usage information
	content.WriteString("📚 COMMAND USAGE\n\n")
	content.WriteString("  :prompt stats or :prompt s     - Show this statistics screen\n")
	content.WriteString("  :prompt list or :prompt l      - Manage prompts\n")
	content.WriteString("  :prompt create or :prompt c    - Create new prompt\n")
	content.WriteString("  :prompt update or :prompt u    - Update existing prompt\n")
	content.WriteString("  :prompt delete or :prompt d    - Delete prompt\n")
	content.WriteString("  :prompt export or :prompt e    - Export prompts\n")
	content.WriteString("\n")

	// Help text
	content.WriteString("Press ESC to return to previous view")

	return content.String()
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

	// Collect message IDs for parallel fetching
	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.Id
		a.AppendMessageID(msg.Id)
	}

	// Fetch message metadata in parallel (optimized for search results display)
	detailedMessages, err := a.Client.GetMessagesMetadataParallel(messageIDs, 10)
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.showError(fmt.Sprintf("❌ Error loading search results: %v", err))
		})
		return
	}

	screenWidth := a.getFormatWidth()
	for i, meta := range detailedMessages {
		if meta == nil {
			continue // Skip failed fetches
		}

		a.messagesMeta = append(a.messagesMeta, meta)
		text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)

		// Capture index for closure
		rowIndex := i
		a.QueueUpdateDraw(func() {
			if table, ok := a.views["list"].(*tview.Table); ok {
				table.SetCell(rowIndex, 0, tview.NewTableCell(text).SetExpansion(1))
			}
			a.refreshTableDisplay()
		})
	}
	if spinnerStop != nil {
		close(spinnerStop)
	}
	a.QueueUpdateDraw(func() {
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.SetTitle(fmt.Sprintf(" 🔍 Search Results (%d) — %s ", len(a.ids), originalQuery))
			if table.GetRowCount() > 1 {
				// Only auto-select if composition panel is not active
				if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
					table.Select(1, 0) // Select first message (row 1, since row 0 is header)
					if len(a.ids) > 0 {
						firstID := a.ids[0]
						a.SetCurrentMessageID(firstID)
						go a.showMessageWithoutFocus(firstID)
					}
				}
				// Close AI panel when loading new messages to avoid conflicts
				if a.aiSummaryVisible {
					if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
						split.ResizeItem(a.aiSummaryView, 0, 0)
					}
					a.aiSummaryVisible = false
					a.aiPanelInPromptMode = false
				}
			}
		}
		// Keep policy for system labels on list while user is in search mode
		a.emailRenderer.SetShowSystemLabelsInList(true)

		// Set focus to list and update focus indicators after search results are loaded
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
		a.SetFocus(a.views["list"])
	})
}

// (moved to status.go) showError/showInfo

// Placeholder methods for functionality that will be implemented later
// (moved to messages.go) loadDrafts

// (moved to messages.go) composeMessage

// (moved to messages.go) listUnreadMessages

// (moved to messages.go) toggleMarkReadUnread

// OBLITERATED: unused updateMessageDisplay function eliminated! 💥

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

// OBLITERATED: unused formatRelativeTime function eliminated! 💥

// (moved to layout.go) updateFocusIndicators

// toggleFocus switches focus between list and text view
// (moved to keys.go) toggleFocus

// restoreFocusAfterModal restores focus to the appropriate view after closing a modal
// (moved to keys.go) restoreFocusAfterModal

// (moved to messages.go) archiveSelected

// (moved to messages.go) replySelected

// (moved to messages.go) showAttachments

// Removed unused function: summarizeSelected

// generateReply generates a reply using LLM
func (a *App) generateReply() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		}()
		return
	}

	a.showCompositionWithStatusBar(services.CompositionTypeReply, messageID)
}

// showCompositionWithStatusBar shows the composition panel with persistent status bar
func (a *App) showCompositionWithStatusBar(compositionType services.CompositionType, originalMessageID string) {
	// Show the composition panel (this handles the business logic)
	a.compositionPanel.Show(compositionType, originalMessageID)

	// Create layout with composition panel + status bar
	compositionLayout := a.createCompositionLayoutWithStatus()

	// Add the combined layout as a page
	a.Pages.AddPage("compose_with_status", compositionLayout, true, true)

	// Update the status bar now that the page is active
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(a.statusBaseline())
	}
}

// showCompositionWithDraft shows the composition panel with a loaded draft and persistent status bar
func (a *App) showCompositionWithDraft(composition *services.Composition) {
	// Show the composition panel with the loaded draft
	a.compositionPanel.ShowWithComposition(composition)

	// Create layout with composition panel + status bar
	compositionLayout := a.createCompositionLayoutWithStatus()

	// Add the combined layout as a page
	a.Pages.AddPage("compose_with_status", compositionLayout, true, true)

	// Switch to the composition page to make it immediately visible
	a.Pages.SwitchToPage("compose_with_status")

	// Force UI redraw to make page switch visible
	a.Draw()

	// Update the status bar now that the page is active
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(a.statusBaseline())
	}

	// Simulate a Tab key to trigger the composition panel's focus management
	if a.compositionPanel != nil {
		tabEvent := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		a.compositionPanel.InputHandler()(tabEvent, nil)
	}
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

// SetFocus overrides the default tview.Application.SetFocus to add composition focus protection
func (a *App) SetFocus(primitive tview.Primitive) *tview.Application {
	// Logger removed - was always nil making this code unreachable

	// Check if composition panel is active and log potential focus stealing
	if a.compositionPanel != nil && a.compositionPanel.IsVisible() {
		// Allow focus to stay within composition panel components
		if primitive == a.compositionPanel.toField ||
			primitive == a.compositionPanel.ccField ||
			primitive == a.compositionPanel.bccField ||
			primitive == a.compositionPanel.subjectField ||
			primitive == a.compositionPanel.bodySection {
			// This is internal composition navigation - allow it
			if a.logger != nil {
				a.logger.Printf("✅ FOCUS: Internal composer navigation - ALLOWED")
			}
			return a.Application.SetFocus(primitive)
		}

		// Log external focus changes that might steal from composer
		if a.logger != nil {
			a.logger.Printf("⚠️ FOCUS: EXTERNAL focus change while composer active! This might steal focus!")
		}

		// For now, still allow the focus change but log it for debugging
		// In a more aggressive fix, we could block it here: return a.Application
	}

	return a.Application.SetFocus(primitive)
}

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

// Picker state management helper methods

// Removed unused picker helper functions: isAnyPickerActive, clearActivePicker

// isLabelsPickerActive returns true if the Labels picker is currently active
func (a *App) isLabelsPickerActive() bool {
	return a.currentActivePicker == PickerLabels
}

// setActivePicker sets the current active picker and logs the change for debugging
func (a *App) setActivePicker(picker ActivePicker) {
	if a.logger != nil {
		a.logger.Printf("Picker state change: %s -> %s", a.currentActivePicker, picker)
	}
	a.currentActivePicker = picker
}

// Shutdown gracefully shuts down the application services
func (a *App) Shutdown() {
	// Shutdown preloader service to stop background goroutines
	if a.preloaderService != nil {
		a.preloaderService.Shutdown()
	}
}
