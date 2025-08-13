package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/llm"
	"github.com/ajramos/gmail-tui/internal/render"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// App encapsulates the terminal UI and the Gmail client
type App struct {
	*tview.Application
	Pages   *Pages
	Config  *config.Config
	Client  *gmail.Client
	LLM     llm.Provider
	Keys    config.KeyBindings
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	views   map[string]tview.Primitive
	cmdBuff *CmdBuff
	running bool
	flash   *Flash
	actions *KeyActions
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

	// UI lifecycle flags
	uiReady          bool // true after first draw
	welcomeAnimating bool // avoid multiple spinner goroutines
	welcomeEmail     string

	// Formatting toggles
	llmTouchUpEnabled bool
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
func NewApp(client *gmail.Client, llmClient llm.Provider, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		Application:       tview.NewApplication(),
		Config:            cfg,
		Client:            client,
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

	return app
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
	a.ids = []string{}
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
		a.ids = append(a.ids, msg.Id)
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
					a.currentMessageID = firstID
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

func (a *App) trashSelected() {
	var messageID string
	var selectedIndex int = -1

	// Get the current message ID based on focus
	if a.currentFocus == "list" {
		// Get from list view (Table)
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}

		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		// Get from text view - read selection from Table
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}

		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "summary" {
		// From AI summary: operate on the selected row in the table
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else {
		a.showError("❌ Unknown focus state")
		return
	}

	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}

	// Get the current message to show confirmation
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}

	// Extract subject for confirmation
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Move message to trash
	err = a.Client.TrashMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error moving to trash: %v", err))
		return
	}

	// Show success message
	a.showStatusMessage(fmt.Sprintf("🗑️  Moved to trash: %s", subject))

	// Remove the message from the list and adjust selection (UI thread)
	if selectedIndex >= 0 && selectedIndex < len(a.ids) {
		a.QueueUpdateDraw(func() {
			list, ok := a.views["list"].(*tview.Table)
			if !ok {
				return
			}
			count := list.GetRowCount()
			if count == 0 {
				return
			}

			// Determine index to remove; fix selection if it's invalid
			removeIndex, _ := list.GetSelection()
			if removeIndex < 0 || removeIndex >= count {
				removeIndex = 0
			}

			// Compute next selection relative to the current list before removal
			next := -1
			if count > 1 {
				next = removeIndex
				if next >= count-1 {
					next = count - 2
				}
				if next < 0 {
					next = 0
				}
				// Ensure table has a valid current selection before removal
				list.Select(removeIndex, 0)
			}

			// Remove visually with safe pre-selection to avoid tview RemoveItem bug when removing current index 0
			if count == 1 {
				// Update caches
				if removeIndex >= 0 && removeIndex < len(a.ids) {
					a.ids = append(a.ids[:removeIndex], a.ids[removeIndex+1:]...)
				}
				if removeIndex >= 0 && removeIndex < len(a.messagesMeta) {
					a.messagesMeta = append(a.messagesMeta[:removeIndex], a.messagesMeta[removeIndex+1:]...)
				}
				list.Clear()
				next = -1
			} else {
				// Choose a pre-selection different from the removal index
				preSelect := removeIndex - 1
				if removeIndex == 0 {
					preSelect = 1
				}
				if preSelect < 0 {
					preSelect = 0
				}
				if preSelect >= count {
					preSelect = count - 1
				}
				list.Select(preSelect, 0)

				// Update caches prior to visual removal
				if removeIndex >= 0 && removeIndex < len(a.ids) {
					a.ids = append(a.ids[:removeIndex], a.ids[removeIndex+1:]...)
				}
				if removeIndex >= 0 && removeIndex < len(a.messagesMeta) {
					a.messagesMeta = append(a.messagesMeta[:removeIndex], a.messagesMeta[removeIndex+1:]...)
				}

				// Now remove the visual item
				if removeIndex >= 0 && removeIndex < list.GetRowCount() {
					list.RemoveRow(removeIndex)
				}

				// Determine next selection post-removal: keep the same visual position if possible
				newCount := list.GetRowCount()
				if next >= 0 && next < newCount {
					list.Select(next, 0)
				} else if newCount > 0 {
					next = 0
					list.Select(0, 0)
				} else {
					next = -1
				}
			}

			// Update title after caches changed
			list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))

			// Update message content pane
			if text, ok := a.views["text"].(*tview.TextView); ok {
				if next >= 0 && next < len(a.ids) {
					go a.showMessageWithoutFocus(a.ids[next])
					if a.aiSummaryVisible {
						go a.generateOrShowSummary(a.ids[next])
					}
				} else {
					text.SetText("No messages")
					text.ScrollToBeginning()
					if a.aiSummaryVisible && a.aiSummaryView != nil {
						a.aiSummaryView.SetText("")
					}
				}
			}
			// Also propagate to base snapshot if in local filter
			if messageID != "" {
				a.baseRemoveByID(messageID)
			}
		})
	}
}

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
