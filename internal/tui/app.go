package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	// AI Summary pane
	aiSummaryView    *tview.TextView
	aiSummaryVisible bool
	aiSummaryCache   map[string]string // messageID -> summary
	aiInFlight       map[string]bool   // messageID -> generating
	// AI label suggestion cache
	aiLabelsCache map[string][]string // messageID -> suggestions
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
func NewApp(client *gmail.Client, llmClient *llm.Client, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		Application:      tview.NewApplication(),
		Config:           cfg,
		Client:           client,
		LLM:              llmClient,
		Keys:             cfg.Keys,
		ctx:              ctx,
		cancel:           cancel,
		views:            make(map[string]tview.Primitive),
		cmdBuff:          NewCmdBuff(),
		flash:            NewFlash(),
		actions:          NewKeyActions(),
		emailRenderer:    render.NewEmailRenderer(),
		ids:              []string{},
		messagesMeta:     []*gmailapi.Message{},
		draftMode:        false,
		draftIDs:         []string{},
		showHelp:         false,
		currentView:      "messages",
		currentFocus:     "list",
		previousFocus:    "list", // Initialize previous focus
		cmdMode:          false,
		cmdBuffer:        "",
		cmdHistory:       make([]string, 0),
		cmdHistoryIndex:  -1,
		currentLayout:    LayoutMedium,
		screenWidth:      80,
		screenHeight:     25,
		currentMessageID: "", // Initialize currentMessageID
		nextPageToken:    "",
		aiSummaryCache:   make(map[string]string),
		aiInFlight:       make(map[string]bool),
		aiLabelsCache:    make(map[string][]string),
	}

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
		return
	}
	def := config.DefaultColors()
	a.emailRenderer.UpdateFromConfig(def)
	tview.Styles.PrimitiveBackgroundColor = def.Body.BgColor.Color()
	tview.Styles.PrimaryTextColor = def.Body.FgColor.Color()
	tview.Styles.BorderColor = def.Frame.Border.FgColor.Color()
	tview.Styles.FocusColor = def.Frame.Border.FocusColor.Color()
}

// reformatListItems recalculates list item strings for current screen width
func (a *App) reformatListItems() {
	list, ok := a.views["list"].(*tview.List)
	if !ok || len(a.ids) == 0 {
		return
	}
	for i := range a.ids {
		if i >= len(a.messagesMeta) || a.messagesMeta[i] == nil {
			continue
		}
		msg := a.messagesMeta[i]
		text, _ := a.emailRenderer.FormatEmailList(msg, a.screenWidth)
		unread := false
		for _, l := range msg.LabelIds {
			if l == "UNREAD" {
				unread = true
				break
			}
		}
		if unread {
			text = "● " + text
		} else {
			text = "○ " + text
		}
		list.SetItemText(i, text, "")
	}
}

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

// initComponents initializes the main UI components
func (a *App) initComponents() {
	// Create main list component
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).
		SetBorderColor(tcell.ColorBlue).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📧 Messages ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)

	// Create main text view
	text := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	text.SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 📄 Message Content ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)

	// Create AI Summary view (hidden by default)
	ai := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetScrollable(true)
	ai.SetBorder(true).
		SetBorderColor(tcell.ColorYellow).
		SetBorderAttributes(tcell.AttrBold).
		SetTitle(" 🤖 AI Summary ").
		SetTitleColor(tcell.ColorYellow).
		SetTitleAlign(tview.AlignCenter)

	// Store components
	a.views["list"] = list
	a.views["text"] = text
	a.aiSummaryView = ai
}

// initViews initializes the main views
func (a *App) initViews() {
	// Create main layout
	mainLayout := a.createMainLayout()

	// Add main view
	a.Pages.AddPage("main", mainLayout, true, true)

	// Add help view
	helpView := a.createHelpView()
	a.Pages.AddPage("help", helpView, true, false)

	// Add search view
	searchView := a.createSearchView()
	a.Pages.AddPage("search", searchView, true, false)

	// Initialize focus indicators
	a.updateFocusIndicators("list")
}

// createMainLayout creates the main application layout
func (a *App) createMainLayout() tview.Primitive {
	// Create the main flex container (vertical layout - one below the other)
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add flash notification at the top (hidden by default)
	mainFlex.AddItem(a.flash.textView, 0, 0, false)

	// Add list of messages (takes 40% of available height)
	mainFlex.AddItem(a.views["list"], 0, 40, true)

	// Message content row: split into content | AI summary (hidden initially)
	contentSplit := tview.NewFlex().SetDirection(tview.FlexColumn)
	contentSplit.AddItem(a.views["text"], 0, 1, false)
	contentSplit.AddItem(a.aiSummaryView, 0, 0, false) // weight 0 = hidden
	a.views["contentSplit"] = contentSplit
	// Add message content (takes 40% of available height)
	mainFlex.AddItem(contentSplit, 0, 40, false)

	// Add command bar (hidden by default, appears when : is pressed)
	cmdBar := a.createCommandBar()
	mainFlex.AddItem(cmdBar, 1, 0, false)

	// Add status bar at the bottom
	statusBar := a.createStatusBar()
	a.views["status"] = statusBar // Store status bar as a view
	mainFlex.AddItem(statusBar, 1, 0, false)

	return mainFlex
}

// createStatusBar creates the status bar
func (a *App) createStatusBar() tview.Primitive {
	status := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText("Gmail TUI | Press ? for help | Press q to quit")

	return status
}

// showStatusMessage displays a message in the status bar
func (a *App) showStatusMessage(msg string) {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(fmt.Sprintf("Gmail TUI | %s | Press ? for help | Press q to quit", msg))
		// Clear the message after 3 seconds
		go func() {
			time.Sleep(3 * time.Second)
			a.QueueUpdateDraw(func() {
				if status, ok := a.views["status"].(*tview.TextView); ok {
					status.SetText("Gmail TUI | Press ? for help | Press q to quit")
				}
			})
		}()
	}
}

// setStatusPersistent sets the status bar text without auto-clearing
func (a *App) setStatusPersistent(msg string) {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(fmt.Sprintf("Gmail TUI | %s | Press ? for help | Press q to quit", msg))
	}
}

// createHelpView creates the help view
func (a *App) createHelpView() tview.Primitive {
	help := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetScrollable(true)

	help.SetText(a.generateHelpText())

	return help
}

// createSearchView creates the search view
func (a *App) createSearchView() tview.Primitive {
	searchInput := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(50).
		SetPlaceholder("Enter search terms (e.g., from:user@example.com, subject:meeting)").
		SetPlaceholderTextColor(tcell.ColorGray)

	// Set the done function after creating the input field
	searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			query := strings.TrimSpace(searchInput.GetText())
			if query != "" {
				go a.performSearch(query)
			}
			a.Pages.SwitchToPage("main")
			a.SetFocus(a.views["list"])
		} else if key == tcell.KeyEscape {
			a.Pages.SwitchToPage("main")
			a.SetFocus(a.views["list"])
		}
	})

	return searchInput
}

// generateHelpText generates the help text
func (a *App) generateHelpText() string {
	var help strings.Builder

	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("🐕 Gmail TUI - Help & Shortcuts\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	help.WriteString("🧭 Navigation\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("Enter     👁️  View selected message\n")
	help.WriteString("r         🔄 Refresh messages\n")
	help.WriteString("s         🔍 Search messages\n")
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
		// Show error message if no client
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetText("❌ Error: Gmail client not initialized\n\n" +
				"To fix this:\n" +
				"1. Download credentials from Google Cloud Console\n" +
				"2. Place them in ~/.config/gmail-tui/credentials.json\n" +
				"3. Run the application again\n\n" +
				"For more information, see the README.md file\n\n" +
				"Press '?' for help or 'q' to quit")
		}
	} else {
		// Show welcome message and load messages
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetText("Welcome to Gmail TUI!\n\n" +
				"Loading your messages...\n\n" +
				"Press '?' for help or 'q' to quit")
		}
		// Load messages in background
		go a.reloadMessages()
	}

	// Start the application
	return a.Application.Run()
}

// bindKeys sets up keyboard shortcuts
func (a *App) bindKeys() {
	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If we're in command mode, handle command input
		if a.cmdMode {
			return a.handleCommandInput(event)
		}

		// Only intercept specific keys, let navigation keys pass through
		switch event.Rune() {
		case ':':
			// Enter command mode (k9s style)
			a.showCommandBar()
			return nil
		case '?':
			a.toggleHelp()
			return nil
		case 'q':
			a.cancel()
			a.Stop()
			return nil
		case 'r':
			if a.draftMode {
				go a.loadDrafts()
			} else {
				go a.reloadMessages()
			}
			return nil
		case 'n':
			// If list has focus and Shift not pressed, load next page of messages; otherwise compose
			if a.currentFocus == "list" && (event.Modifiers()&tcell.ModShift) == 0 {
				go a.loadMoreMessages()
				return nil
			}
			go a.composeMessage(false)
			return nil
		case 's':
			a.Pages.SwitchToPage("search")
			a.SetFocus(a.views["search"])
			return nil
		case 'u':
			go a.listUnreadMessages()
			return nil
		case 't':
			go a.toggleMarkReadUnread()
			return nil
		case 'd':
			go a.trashSelected()
			return nil
		case 'a':
			go a.archiveSelected()
			return nil
		case 'R':
			go a.replySelected()
			return nil
		case 'D':
			go a.loadDrafts()
			return nil
		case 'A':
			go a.showAttachments()
			return nil
		case 'l':
			// Open contextual labels view immediately
			a.manageLabels()
			return nil
		case 'm':
			// Open move view immediately (synchronous) to avoid extra key press
			a.moveSelected()
			return nil
		case 'o':
			go a.suggestLabel()
			return nil
		}

		// Handle Tab key for switching focus
		if event.Key() == tcell.KeyTab {
			a.toggleFocus()
			return nil
		}

		// LLM features
		if a.LLM != nil {
			switch event.Rune() {
			case 'y':
				a.toggleAISummary()
				return nil
			case 'g':
				go a.generateReply()
				return nil
			case 'o':
				go a.suggestLabel()
				return nil
			}
		}

		// Let all other events pass through to the focused component
		return event
	})

	// Handle Enter key for viewing messages and show position in status
	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			if index < len(a.ids) {
				go a.showMessage(a.ids[index])
			}
		})
		list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			if index >= 0 && index < len(a.ids) {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", index+1, len(a.ids)))
			}
		})
	}
}

// handleCommandInput handles input when in command mode
func (a *App) handleCommandInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		// Exit command mode
		a.hideCommandBar()
		return nil
	case tcell.KeyEnter:
		// Execute command
		a.executeCommand(a.cmdBuffer)
		a.hideCommandBar()
		return nil
	case tcell.KeyTab:
		// Auto-complete command
		a.completeCommand()
		return nil
	case tcell.KeyBackspace, tcell.KeyDelete:
		// Delete last character
		if len(a.cmdBuffer) > 0 {
			a.cmdBuffer = a.cmdBuffer[:len(a.cmdBuffer)-1]
			a.updateCommandBar()
		}
		return nil
	case tcell.KeyUp:
		// Navigate command history up
		if a.cmdHistoryIndex > 0 {
			a.cmdHistoryIndex--
			if a.cmdHistoryIndex >= 0 && a.cmdHistoryIndex < len(a.cmdHistory) {
				a.cmdBuffer = a.cmdHistory[a.cmdHistoryIndex]
				a.updateCommandBar()
			}
		}
		return nil
	case tcell.KeyDown:
		// Navigate command history down
		if a.cmdHistoryIndex < len(a.cmdHistory)-1 {
			a.cmdHistoryIndex++
			if a.cmdHistoryIndex >= 0 && a.cmdHistoryIndex < len(a.cmdHistory) {
				a.cmdBuffer = a.cmdHistory[a.cmdHistoryIndex]
				a.updateCommandBar()
			} else {
				a.cmdBuffer = ""
				a.cmdHistoryIndex = len(a.cmdHistory)
			}
			a.updateCommandBar()
		}
		return nil
	}

	// Handle regular character input
	if event.Rune() != 0 {
		a.cmdBuffer += string(event.Rune())
		a.updateCommandBar()
		return nil
	}

	return event
}

// updateCommandBar updates the command bar display
func (a *App) updateCommandBar() {
	if cmdBar, ok := a.views["cmdBar"].(*tview.TextView); ok {
		// Generate suggestion based on current buffer
		suggestion := a.generateCommandSuggestion(a.cmdBuffer)
		a.cmdSuggestion = suggestion

		// Display command with suggestion
		displayText := fmt.Sprintf(":%s", a.cmdBuffer)
		if suggestion != "" && suggestion != a.cmdBuffer {
			displayText += fmt.Sprintf(" [%s]", suggestion)
		}

		cmdBar.SetText(displayText)
		cmdBar.SetTextColor(tcell.ColorYellow)
		cmdBar.SetBackgroundColor(tcell.ColorBlack)
	}
}

// generateCommandSuggestion generates a suggestion based on the current command buffer
func (a *App) generateCommandSuggestion(buffer string) string {
	if buffer == "" {
		return ""
	}

	// Available commands
	commands := map[string][]string{
		"l":       {"labels", "list"},
		"la":      {"labels"},
		"lab":     {"labels"},
		"labe":    {"labels"},
		"label":   {"labels"},
		"labels":  {"labels"},
		"s":       {"search"},
		"se":      {"search"},
		"sea":     {"search"},
		"sear":    {"search"},
		"searc":   {"search"},
		"search":  {"search"},
		"i":       {"inbox"},
		"in":      {"inbox"},
		"inb":     {"inbox"},
		"inbo":    {"inbox"},
		"inbox":   {"inbox"},
		"c":       {"compose"},
		"co":      {"compose"},
		"com":     {"compose"},
		"comp":    {"compose"},
		"compo":   {"compose"},
		"compos":  {"compose"},
		"compose": {"compose"},
		"h":       {"help"},
		"he":      {"help"},
		"hel":     {"help"},
		"help":    {"help"},
		"q":       {"quit"},
		"qu":      {"quit"},
		"qui":     {"quit"},
		"quit":    {"quit"},
	}

	// Check for exact matches first
	if suggestions, exists := commands[buffer]; exists && len(suggestions) > 0 {
		return suggestions[0]
	}

	// Check for partial matches
	for cmd, suggestions := range commands {
		if strings.HasPrefix(cmd, buffer) && cmd != buffer {
			return suggestions[0]
		}
	}

	return ""
}

// completeCommand completes the current command with the suggestion
func (a *App) completeCommand() {
	if a.cmdSuggestion != "" && a.cmdSuggestion != a.cmdBuffer {
		a.cmdBuffer = a.cmdSuggestion
		a.updateCommandBar()
	}
}

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

// reloadMessages loads messages from the inbox
func (a *App) reloadMessages() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.draftMode = false
	if list, ok := a.views["list"].(*tview.List); ok {
		list.Clear()
	}
	a.ids = []string{}
	a.messagesMeta = []*gmailapi.Message{}

	// Show loading message
	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetTitle(" 🔄 Loading messages... ")
	}
	a.Draw()

	// Check if client is available
	if a.Client == nil {
		a.showError("❌ Gmail client not initialized")
		return
	}

	messages, next, err := a.Client.ListMessagesPage(50, "")
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading messages: %v", err))
		return
	}
	a.nextPageToken = next

	// Show success message if no messages
	if len(messages) == 0 {
		a.QueueUpdateDraw(func() {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.SetTitle(" 📧 No messages found ")
			}
		})
		a.showInfo("📧 No messages found in your inbox")
		return
	}

	// Usar ancho disponible actual del list (simple, sin watchers)
	screenWidth := a.getFormatWidth()

	// Process messages using the email renderer
	for i, msg := range messages {
		a.ids = append(a.ids, msg.Id)

		// Get only metadata, not full content
		message, err := a.Client.GetMessage(msg.Id)
		if err != nil {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.AddItem(fmt.Sprintf("⚠️  Error loading message %d", i+1), "Failed to load", 0, nil)
			}
			continue
		}

		// Use the email renderer to format the message
		formattedText, _ := a.emailRenderer.FormatEmailList(message, screenWidth)

		// Add unread indicator
		unread := false
		for _, labelId := range message.LabelIds {
			if labelId == "UNREAD" {
				unread = true
				break
			}
		}

		if unread {
			formattedText = "● " + formattedText
		} else {
			formattedText = "○ " + formattedText
		}

		if list, ok := a.views["list"].(*tview.List); ok {
			// Add item with color using the standard method
			list.AddItem(formattedText, "", 0, nil)
		}

		// cache meta for resize re-rendering
		a.messagesMeta = append(a.messagesMeta, message)

		// Update title periodically
		if (i+1)%10 == 0 {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.SetTitle(fmt.Sprintf(" 🔄 Loading... (%d/%d) ", i+1, len(messages)))
			}
			a.Draw()
		}
	}

	a.QueueUpdateDraw(func() {
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
			// Also update status with current selection
			idx := list.GetCurrentItem()
			if idx >= 0 && len(a.ids) > 0 {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", idx+1, len(a.ids)))
			}
		}
	})

	// Set focus back to list only if we're on the main page
	if pageName, _ := a.Pages.GetFrontPage(); pageName == "main" {
		a.SetFocus(a.views["list"])
	}
}

// loadMoreMessages fetches the next page of inbox and appends to list
func (a *App) loadMoreMessages() {
	if a.nextPageToken == "" {
		a.showStatusMessage("No more messages")
		return
	}
	a.setStatusPersistent("Loading next 50 messages…")
	messages, next, err := a.Client.ListMessagesPage(50, a.nextPageToken)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading more: %v", err))
		return
	}
	// Append
	screenWidth := a.getFormatWidth()
	for _, msg := range messages {
		a.ids = append(a.ids, msg.Id)
		meta, err := a.Client.GetMessage(msg.Id)
		if err != nil {
			continue
		}
		a.messagesMeta = append(a.messagesMeta, meta)
		text, _ := a.emailRenderer.FormatEmailList(meta, screenWidth)
		unread := false
		for _, l := range meta.LabelIds {
			if l == "UNREAD" {
				unread = true
				break
			}
		}
		if unread {
			text = "● " + text
		} else {
			text = "○ " + text
		}
		if list, ok := a.views["list"].(*tview.List); ok {
			list.AddItem(text, "", 0, nil)
		}
	}
	a.nextPageToken = next
	a.QueueUpdateDraw(func() {
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
			idx := list.GetCurrentItem()
			if idx >= 0 && len(a.ids) > 0 {
				a.setStatusPersistent(fmt.Sprintf("Message %d/%d", idx+1, len(a.ids)))
			}
		}
	})
}

// showMessage displays a message in the text view
func (a *App) showMessage(id string) {
	// Show loading message immediately
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetText("Loading message...")
		text.ScrollToBeginning()
	}

	// Automatically switch focus to text view when viewing a message
	a.SetFocus(a.views["text"])
	a.currentFocus = "text"
	a.updateFocusIndicators("text")
	a.currentMessageID = id

	a.Draw()

	// Load message content in background
	go func() {
		message, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading message: %v", err))
			return
		}

		var content strings.Builder

		// Styled header according to theme
		header := a.emailRenderer.FormatHeaderStyled(
			message.Subject,
			message.From,
			message.Date,
			message.Labels,
		)
		content.WriteString(header)

		// Message content
		if message.PlainText != "" {
			content.WriteString(message.PlainText)
		} else {
			content.WriteString("No text content available")
		}

		// Update UI in main thread
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.SetText(content.String())
				// Scroll to the top of the text
				text.ScrollToBeginning()
			}
			// If AI pane is visible, refresh summary for this message
			if a.aiSummaryVisible {
				a.generateOrShowSummary(id)
			}
		})
	}()
}

// showMessageWithoutFocus loads the message content but does not change focus
func (a *App) showMessageWithoutFocus(id string) {
	// Show loading message
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetText("Loading message...")
		text.ScrollToBeginning()
	}
	a.currentMessageID = id

	go func() {
		message, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading message: %v", err))
			return
		}

		var content strings.Builder
		header := a.emailRenderer.FormatHeaderStyled(
			message.Subject,
			message.From,
			message.Date,
			message.Labels,
		)
		content.WriteString(header)
		if message.PlainText != "" {
			content.WriteString(message.PlainText)
		} else {
			content.WriteString("No text content available")
		}

		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.SetText(content.String())
				text.ScrollToBeginning()
			}
		})
	}()
}

// performSearch executes the search query
func (a *App) performSearch(query string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if strings.TrimSpace(query) == "" {
		a.showError("Search query cannot be empty")
		return
	}

	if list, ok := a.views["list"].(*tview.List); ok {
		list.Clear()
	}
	a.ids = []string{}
	a.messagesMeta = []*gmailapi.Message{}

	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetTitle(fmt.Sprintf(" 🔍 Searching: %s ", query))
	}
	a.Draw()

	// Perform search
	messages, err := a.Client.SearchMessages(query, 50)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Search error: %v", err))
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetTitle(" ❌ Search failed ")
		}
		return
	}

	if len(messages) == 0 {
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetTitle(fmt.Sprintf(" 🔍 No results for: %s ", query))
		}
		a.showInfo(fmt.Sprintf("🔍 No messages found for query: %s", query))
		return
	}

	// Display search results
	for i, msg := range messages {
		a.ids = append(a.ids, msg.Id)
		meta, err := a.Client.GetMessage(msg.Id)
		if err != nil {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.AddItem(fmt.Sprintf("⚠️  Error loading message %d", i+1), "Failed to load", 0, nil)
			}
			continue
		}
		text, _ := a.emailRenderer.FormatEmailList(meta, a.getFormatWidth())
		unread := false
		for _, l := range meta.LabelIds {
			if l == "UNREAD" {
				unread = true
				break
			}
		}
		if unread {
			text = "● " + text
		} else {
			text = "○ " + text
		}
		if list, ok := a.views["list"].(*tview.List); ok {
			list.AddItem(text, "", 0, nil)
		}
		a.messagesMeta = append(a.messagesMeta, meta)
	}

	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetTitle(fmt.Sprintf(" 🔍 Search Results (%d) for: %s ", len(a.ids), query))
	}
	a.SetFocus(a.views["list"])
}

// Utility methods
func (a *App) showError(msg string) {
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetText(fmt.Sprintf("❌ Error: %s", msg))
	}
}

func (a *App) showInfo(msg string) {
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetText(fmt.Sprintf("ℹ️  Info: %s", msg))
	}
}

// Placeholder methods for functionality that will be implemented later
func (a *App) loadDrafts() {
	a.showInfo("Drafts functionality not yet implemented")
}

func (a *App) composeMessage(draft bool) {
	a.showInfo("Compose message functionality not yet implemented")
}

func (a *App) listUnreadMessages() {
	a.showInfo("Unread messages functionality not yet implemented")
}

func (a *App) toggleMarkReadUnread() {
	var messageID string
	var selectedIndex int = -1

	// Get the current message ID based on focus
	if a.currentFocus == "list" {
		// Get from list view
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex = list.GetCurrentItem()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}

		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		// Get from text view - we need to find the currently displayed message
		// Since we don't store the current message ID, we'll need to get it from the list
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex = list.GetCurrentItem()
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

	// Get the current message to check its read status
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}

	// Check if message is currently unread
	isUnread := false
	for _, labelID := range message.LabelIds {
		if labelID == "UNREAD" {
			isUnread = true
			break
		}
	}

	// Toggle the read status
	var err2 error
	if isUnread {
		// Mark as read
		err2 = a.Client.MarkAsRead(messageID)
		if err2 == nil {
			a.showStatusMessage("✅ Message marked as read")
		} else {
			a.showError(fmt.Sprintf("❌ Error marking as read: %v", err2))
			return
		}
	} else {
		// Mark as unread
		err2 = a.Client.MarkAsUnread(messageID)
		if err2 == nil {
			a.showStatusMessage("✅ Message marked as unread")
		} else {
			a.showError(fmt.Sprintf("❌ Error marking as unread: %v", err2))
			return
		}
	}

	// Update the UI to reflect the change
	if selectedIndex >= 0 {
		finalIndex := selectedIndex
		finalIsUnread := !isUnread
		a.QueueUpdateDraw(func() {
			a.updateMessageDisplay(finalIndex, finalIsUnread)
		})
	}
}

// updateMessageDisplay updates the display of a specific message in the list
func (a *App) updateMessageDisplay(index int, isUnread bool) {
	list, ok := a.views["list"].(*tview.List)
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

	// Update the item in the list
	list.SetItemText(index, formattedText, "")
}

func (a *App) trashSelected() {
	var messageID string
	var selectedIndex int = -1

	// Get the current message ID based on focus
	if a.currentFocus == "list" {
		// Get from list view
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex = list.GetCurrentItem()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}

		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		// Get from text view - we need to find the currently displayed message
		// Since we don't store the current message ID, we'll need to get it from the list
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex = list.GetCurrentItem()
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

	// Remove the message from the list and adjust selection
	if selectedIndex >= 0 && selectedIndex < len(a.ids) {
		// Remove from slices
		a.ids = append(a.ids[:selectedIndex], a.ids[selectedIndex+1:]...)
		if selectedIndex < len(a.messagesMeta) {
			a.messagesMeta = append(a.messagesMeta[:selectedIndex], a.messagesMeta[selectedIndex+1:]...)
		}

		a.QueueUpdateDraw(func() {
			if list, ok := a.views["list"].(*tview.List); ok {
				// Remove the item from the list
				list.RemoveItem(selectedIndex)

				// Choose next selection index
				next := selectedIndex
				if next >= list.GetItemCount() {
					next = list.GetItemCount() - 1
				}
				if next >= 0 {
					list.SetCurrentItem(next)
				}

				// Update the title with new count
				list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
			}
			// Clear message content if the deleted item was being viewed
			if a.currentFocus == "text" {
				if text, ok := a.views["text"].(*tview.TextView); ok {
					text.SetText("Message removed")
					text.ScrollToBeginning()
				}
			}
		})
	}
}

func (a *App) manageLabels() {
	// Get the current message ID
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}

	// Load all available labels
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading labels: %v", err))
		return
	}

	// Get the current message to see which labels it has
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}

	// Create contextual labels view for the selected message
	a.showMessageLabelsView(labels, message)
}

// showMessageLabelsView displays labels for a specific message
func (a *App) showMessageLabelsView(labels []*gmailapi.Label, message *gmailapi.Message) {
	// Create labels list view
	labelsList := tview.NewList()
	labelsList.SetBorder(true)
	labelsList.SetTitle(" 🏷️  Message Labels ")

	// Get current message labels
	currentLabels := make(map[string]bool)
	if message.LabelIds != nil {
		for _, labelID := range message.LabelIds {
			currentLabels[labelID] = true
		}
	}

	// Extract subject for display
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Partition applied vs not-applied and sort each group; applied first
	applied, notApplied := a.partitionAndSortLabels(labels, currentLabels)
	for _, label := range append(applied, notApplied...) {
		// Skip system labels that start with CATEGORY_ or are special
		// (already filtered in helper)

		// Store label info for the callback (avoid capturing loop vars directly)
		labelID := label.Id
		labelName := label.Name

		// Determine current applied state
		isApplied := currentLabels[labelID]

		// Create display text with label info and status (no secondary text)
		var displayText string
		if isApplied {
			displayText = fmt.Sprintf("✅ %s", labelName)
		} else {
			displayText = fmt.Sprintf("○ %s", labelName)
		}

		labelsList.AddItem(displayText, "", 0, func() {
			// Capture current index and state at click time
			index := labelsList.GetCurrentItem()
			currentlyApplied := currentLabels[labelID]
			// Async toggle
			a.toggleLabelForMessage(message.Id, labelID, labelName, currentlyApplied, func(newApplied bool, err error) {
				if err != nil {
					return
				}
				// Update local state map
				currentLabels[labelID] = newApplied
				// Update UI immediately
				newText := fmt.Sprintf("○ %s", labelName)
				if newApplied {
					newText = fmt.Sprintf("✅ %s", labelName)
				}
				a.QueueUpdateDraw(func() {
					labelsList.SetItemText(index, newText, "")
				})
				// Update cached meta for main list (for UNREAD/star, etc.)
				a.updateCachedMessageLabels(message.Id, labelID, newApplied)
			})
		})
	}

	// Add "Create new label" option at the end
	labelsList.AddItem("➕ Create new label", "Press Enter to create", 0, func() {
		a.createNewLabelFromView()
	})

	// Ensure first item is selected to enable immediate arrow navigation
	if labelsList.GetItemCount() > 0 {
		labelsList.SetCurrentItem(0)
	}

	// Set up key bindings for the labels view
	labelsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Return to main view
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			// Refresh currently displayed message content to reflect new labels
			go a.refreshMessageContent(message.Id)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'n' {
				// Create new label
				a.createNewLabelFromView()
				return nil
			}
			if event.Rune() == 'r' {
				// Refresh labels view
				go a.manageLabels()
				return nil
			}
		}
		return event
	})

	// Create the labels view page
	labelsView := tview.NewFlex().SetDirection(tview.FlexRow)

	// Title with message subject
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetText(fmt.Sprintf("🏷️  Labels for: %s", subject))
	title.SetTextColor(tcell.ColorYellow)
	title.SetBorder(true)

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter: Toggle label | n: Create new label | r: Refresh | ESC: Back")
	instructions.SetTextColor(tcell.ColorGray)

	labelsView.AddItem(title, 3, 0, false)
	labelsView.AddItem(labelsList, 0, 1, true)
	labelsView.AddItem(instructions, 2, 0, false)

	// Add labels view to pages
	a.Pages.AddPage("messageLabels", labelsView, true, true)
	a.Pages.SwitchToPage("messageLabels")
	a.SetFocus(labelsList)
}

// toggleLabelForMessage toggles a label asynchronously and invokes onDone when finished
func (a *App) toggleLabelForMessage(messageID, labelID, labelName string, isCurrentlyApplied bool, onDone func(newApplied bool, err error)) {
	go func() {
		if isCurrentlyApplied {
			if err := a.Client.RemoveLabel(messageID, labelID); err != nil {
				a.showError(fmt.Sprintf("❌ Error removing label %s: %v", labelName, err))
				onDone(isCurrentlyApplied, err)
				return
			}
			a.showStatusMessage(fmt.Sprintf("🏷️  Removed label: %s", labelName))
			onDone(false, nil)
			return
		}
		if err := a.Client.ApplyLabel(messageID, labelID); err != nil {
			a.showError(fmt.Sprintf("❌ Error applying label %s: %v", labelName, err))
			onDone(isCurrentlyApplied, err)
			return
		}
		a.showStatusMessage(fmt.Sprintf("🏷️  Applied label: %s", labelName))
		onDone(true, nil)
	}()
}

// showMessagesWithLabel shows messages that have a specific label
func (a *App) showMessagesWithLabel(labelID, labelName string) {
	// Search for messages with this label
	query := fmt.Sprintf("label:%s", labelName)
	messages, err := a.Client.SearchMessages(query, 50)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error searching messages with label %s: %v", labelName, err))
		return
	}

	// Create messages view for this label
	a.showMessagesForLabel(messages, labelName)
}

// showMessagesForLabel displays messages that have a specific label
func (a *App) showMessagesForLabel(messages []*gmailapi.Message, labelName string) {
	// Create messages list for this label
	messagesList := tview.NewList()
	messagesList.SetBorder(true)
	messagesList.SetTitle(fmt.Sprintf(" 📧 Messages with label: %s ", labelName))

	// Clear current IDs and populate with new messages
	a.ids = []string{}

	for i, msg := range messages {
		a.ids = append(a.ids, msg.Id)

		// Get message details
		message, err := a.Client.GetMessageWithContent(msg.Id)
		if err != nil {
			messagesList.AddItem(fmt.Sprintf("⚠️  Error loading message %d", i+1), "Failed to load", 0, nil)
			continue
		}

		subject := message.Subject
		if subject == "" {
			subject = "(No subject)"
		}

		// Format the display text
		displayText := fmt.Sprintf("%s", subject)
		secondaryText := fmt.Sprintf("From: %s | %s", message.From, formatRelativeTime(message.Date))

		messagesList.AddItem(displayText, secondaryText, 0, func() {
			// Show the selected message
			if len(a.ids) > 0 {
				go a.showMessage(a.ids[0])
			}
		})
	}

	if len(messages) == 0 {
		messagesList.AddItem("No messages with this label", "", 0, nil)
	}

	// Set up key bindings
	messagesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Return to labels view
			a.Pages.SwitchToPage("labels")
			return nil
		case 't':
			// Toggle read/unread for selected message
			go a.toggleMarkReadUnread()
			return nil
		case 'd':
			// Trash selected message
			go a.trashSelected()
			return nil
		}
		return event
	})

	// Create the messages view page
	pageName := fmt.Sprintf("messages_%s", labelName)
	a.Pages.AddPage(pageName, messagesList, true, true)
	a.Pages.SwitchToPage(pageName)
	a.SetFocus(messagesList)
}

// createNewLabelFromView creates a new label from the labels view
func (a *App) createNewLabelFromView() {
	// Create input field for new label name
	inputField := tview.NewInputField().
		SetLabel("Label name: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
			return len(textToCheck) > 0 && len(textToCheck) < 50
		})

	// Create modal for new label
	modal := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetText("🏷️  Create New Label")
	title.SetTextColor(tcell.ColorYellow)
	title.SetBorder(true)

	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter label name and press Enter | ESC to cancel")
	instructions.SetTextColor(tcell.ColorGray)

	modal.AddItem(title, 3, 0, false)
	modal.AddItem(inputField, 3, 0, true)
	modal.AddItem(instructions, 2, 0, false)

	// Handle input
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			labelName := strings.TrimSpace(inputField.GetText())
			if labelName != "" {
				go func() {
					_, err := a.Client.CreateLabel(labelName)
					if err != nil {
						a.showError(fmt.Sprintf("❌ Error creating label: %v", err))
						return
					}

					a.showStatusMessage(fmt.Sprintf("🏷️  Created label: %s", labelName))

					// Return to labels view and refresh
					a.QueueUpdateDraw(func() {
						a.Pages.SwitchToPage("labels")
						// Refresh the labels view
						go a.manageLabels()
					})
				}()
			}
		case tcell.KeyEscape:
			a.Pages.SwitchToPage("labels")
		}
	})

	// Add modal to pages
	a.Pages.AddPage("createLabel", modal, true, true)
	a.Pages.SwitchToPage("createLabel")
	a.SetFocus(inputField)
}

// deleteSelectedLabel deletes the selected label (placeholder for now)
func (a *App) deleteSelectedLabel(labelsList *tview.List) {
	a.showInfo("Delete label functionality not yet implemented")
}

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

// updateFocusIndicators updates the visual indicators for the focused view
func (a *App) updateFocusIndicators(focusedView string) {
	// Reset all borders to default
	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetBorderColor(tcell.ColorGray)
	}
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetBorderColor(tcell.ColorGray)
	}
	if a.aiSummaryView != nil {
		a.aiSummaryView.SetBorderColor(tcell.ColorGray)
	}

	// Set focused view border to bright color
	switch focusedView {
	case "list":
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetBorderColor(tcell.ColorYellow)
		}
	case "text":
		if text, ok := a.views["text"].(*tview.TextView); ok {
			text.SetBorderColor(tcell.ColorYellow)
		}
	case "summary":
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetBorderColor(tcell.ColorYellow)
		}
	}
}

// toggleFocus switches focus between list and text view
func (a *App) toggleFocus() {
	currentFocus := a.GetFocus()

	if currentFocus == a.views["list"] {
		a.SetFocus(a.views["text"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	} else if currentFocus == a.views["text"] {
		if a.aiSummaryVisible {
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		} else {
			a.SetFocus(a.views["list"])
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
		}
	} else if a.aiSummaryVisible && currentFocus == a.aiSummaryView {
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	} else {
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	}
}

// restoreFocusAfterModal restores focus to the appropriate view after closing a modal
func (a *App) restoreFocusAfterModal() {
	// Simple approach: always restore focus to list view
	a.SetFocus(a.views["list"])
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// archiveSelected archives the selected message
func (a *App) archiveSelected() {
	var messageID string
	var selectedIndex int = -1

	// Determine current selection based on focus
	if a.currentFocus == "list" {
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex = list.GetCurrentItem()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		list, ok := a.views["list"].(*tview.List)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex = list.GetCurrentItem()
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

	// Fetch message to get subject for confirmation/status
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Archive: remove INBOX label
	if err := a.Client.ArchiveMessage(messageID); err != nil {
		a.showError(fmt.Sprintf("❌ Error archiving message: %v", err))
		return
	}

	a.showStatusMessage(fmt.Sprintf("📥 Archived: %s", subject))

	// Remove from current list (since we show only INBOX)
	if selectedIndex >= 0 && selectedIndex < len(a.ids) {
		a.ids = append(a.ids[:selectedIndex], a.ids[selectedIndex+1:]...)
		if selectedIndex < len(a.messagesMeta) {
			a.messagesMeta = append(a.messagesMeta[:selectedIndex], a.messagesMeta[selectedIndex+1:]...)
		}
		a.QueueUpdateDraw(func() {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.RemoveItem(selectedIndex)
				list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
			}
		})
	}
}

// replySelected replies to the selected message
func (a *App) replySelected() {
	a.showInfo("Reply functionality not yet implemented")
}

// showAttachments shows attachments for the selected message
func (a *App) showAttachments() {
	a.showInfo("Attachments functionality not yet implemented")
}

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
			a.QueueUpdateDraw(func() { a.showStatusMessage("⚠️ LLM error") })
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

// suggestLabel suggests a label using LLM
func (a *App) suggestLabel() {
	if a.LLM == nil {
		a.showStatusMessage("⚠️ LLM disabled")
		return
	}
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}
	// If cached suggestions exist, show picker immediately
	if cached, ok := a.aiLabelsCache[messageID]; ok && len(cached) > 0 {
		a.showLabelSuggestions(messageID, cached)
		return
	}
	a.setStatusPersistent("🏷️ Suggesting labels…")
	go func() {
		// Load message content
		m, err := a.Client.GetMessageWithContent(messageID)
		if err != nil {
			a.showError("❌ Error loading message")
			return
		}
		// Load available labels
		labels, err := a.Client.ListLabels()
		if err != nil || len(labels) == 0 {
			a.showError("❌ Error loading labels")
			return
		}
		// Build allowed label names and a lookup map
		allowed := make([]string, 0, len(labels))
		nameToID := make(map[string]string, len(labels))
		for _, l := range labels {
			if strings.HasPrefix(l.Id, "CATEGORY_") || l.Id == "INBOX" || l.Id == "SENT" || l.Id == "DRAFT" || l.Id == "SPAM" || l.Id == "TRASH" || l.Id == "CHAT" || (strings.HasSuffix(l.Id, "_STARRED") && l.Id != "STARRED") {
				continue
			}
			allowed = append(allowed, l.Name)
			nameToID[l.Name] = l.Id
		}
		sort.Slice(allowed, func(i, j int) bool { return strings.ToLower(allowed[i]) < strings.ToLower(allowed[j]) })
		// Prompt: request JSON array of existing names
		body := m.PlainText
		if len([]rune(body)) > 6000 {
			body = string([]rune(body)[:6000])
		}
		prompt := "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: " + strings.Join(allowed, ", ") + "\n\nEmail:\n" + body
		resp, err := a.LLM.Generate(prompt)
		if err != nil {
			a.showStatusMessage("⚠️ LLM error")
			return
		}
		// Parse JSON array
		var arr []string
		if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &arr); err != nil {
			// fallback: try comma split
			parts := strings.Split(resp, ",")
			for _, p := range parts {
				if s := strings.TrimSpace(strings.Trim(p, "\"[]")); s != "" {
					arr = append(arr, s)
				}
			}
		}
		// Filter to allowed and unique
		uniq := make([]string, 0, 3)
		seen := make(map[string]struct{})
		for _, s := range arr {
			if _, ok := nameToID[s]; !ok {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			uniq = append(uniq, s)
			if len(uniq) == 3 {
				break
			}
		}
		if len(uniq) == 0 {
			a.showStatusMessage("ℹ️ No label suggestion")
			return
		}
		a.aiLabelsCache[messageID] = uniq
		a.QueueUpdateDraw(func() { a.showLabelSuggestions(messageID, uniq); a.showStatusMessage("✅ Suggestions ready") })
	}()
}

// showLabelSuggestions displays a picker to apply one or all suggested labels
func (a *App) showLabelSuggestions(messageID string, suggestions []string) {
	picker := tview.NewList().ShowSecondaryText(false)
	picker.SetBorder(true)
	picker.SetTitle(" 🏷️ Apply suggested label(s) ")

	// Load labels for ID lookup
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError("❌ Error loading labels")
		return
	}
	nameToID := make(map[string]string, len(labels))
	for _, l := range labels {
		nameToID[l.Name] = l.Id
	}

	// Apply one
	for _, name := range suggestions {
		labelName := name
		picker.AddItem(labelName, "Enter to apply", 0, func() {
			if id, ok := nameToID[labelName]; ok {
				go func() {
					if err := a.Client.ApplyLabel(messageID, id); err != nil {
						a.showError("❌ Error applying label")
						return
					}
					a.updateCachedMessageLabels(messageID, id, true)
					a.showStatusMessage("✅ Applied: " + labelName)
					a.Pages.SwitchToPage("main")
					a.restoreFocusAfterModal()
				}()
			}
		})
	}
	// Apply all (with emoji)
	picker.AddItem("✅ Apply all", "Apply all suggested labels", 0, func() {
		go func() {
			for _, name := range suggestions {
				if id, ok := nameToID[name]; ok {
					_ = a.Client.ApplyLabel(messageID, id)
					a.updateCachedMessageLabels(messageID, id, true)
				}
			}
			a.showStatusMessage("✅ Applied all suggestions")
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
		}()
	})

	// Pick from all available labels (no creation)
	picker.AddItem("🗂️  Pick from all labels", "Open full label list to apply", 0, func() {
		a.showAllLabelsPicker(messageID)
	})

	// Add custom label (create if missing) and apply
	picker.AddItem("➕ Add custom label", "Create or pick a label and apply", 0, func() {
		input := tview.NewInputField().
			SetLabel("Label name: ").
			SetFieldWidth(30)
		modal := tview.NewFlex().SetDirection(tview.FlexRow)
		title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
		title.SetBorder(true)
		title.SetText("Enter a label name | Enter=apply, ESC=cancel")
		modal.AddItem(title, 3, 0, false)
		modal.AddItem(input, 3, 0, true)

		input.SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEscape {
				a.Pages.SwitchToPage("aiLabelSuggestions")
				a.SetFocus(picker)
				return
			}
			if key == tcell.KeyEnter {
				name := strings.TrimSpace(input.GetText())
				if name == "" {
					return
				}
				go func() {
					// Resolve or create label
					id, ok := nameToID[name]
					if !ok {
						// Try to find case-insensitive match
						for n, i := range nameToID {
							if strings.EqualFold(n, name) {
								id = i
								ok = true
								break
							}
						}
					}
					if !ok {
						// Create new label
						created, err := a.Client.CreateLabel(name)
						if err != nil {
							a.showError("❌ Error creating label")
							return
						}
						id = created.Id
						nameToID[name] = id
					}
					// Apply
					if err := a.Client.ApplyLabel(messageID, id); err != nil {
						a.showError("❌ Error applying label")
						return
					}
					a.updateCachedMessageLabels(messageID, id, true)
					a.showStatusMessage("✅ Applied: " + name)
					// Return to main
					a.QueueUpdateDraw(func() { a.Pages.SwitchToPage("main"); a.restoreFocusAfterModal() })
				}()
			}
		})

		a.Pages.AddPage("aiLabelAddCustom", modal, true, true)
		a.Pages.SwitchToPage("aiLabelAddCustom")
		a.SetFocus(input)
	})

	picker.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			return nil
		}
		return e
	})

	// Show page
	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select label to apply | Enter=apply, ESC=back")
	v.AddItem(title, 3, 0, false)
	v.AddItem(picker, 0, 1, true)
	a.Pages.AddPage("aiLabelSuggestions", v, true, true)
	a.Pages.SwitchToPage("aiLabelSuggestions")
	if picker.GetItemCount() > 0 {
		picker.SetCurrentItem(0)
	}
	a.SetFocus(picker)
}

// createCommandBar creates the command bar component (k9s style)
func (a *App) createCommandBar() tview.Primitive {
	cmdBar := tview.NewTextView()
	cmdBar.SetDynamicColors(true)
	cmdBar.SetTextAlign(tview.AlignLeft)
	cmdBar.SetBorder(true)
	cmdBar.SetBorderColor(tcell.ColorBlue)
	cmdBar.SetBorderAttributes(tcell.AttrBold)
	cmdBar.SetTitle(" 💻 Command ")
	cmdBar.SetTitleColor(tcell.ColorYellow)
	cmdBar.SetTitleAlign(tview.AlignCenter)
	cmdBar.SetText("")
	cmdBar.SetBackgroundColor(tcell.ColorBlack)
	cmdBar.SetTextColor(tcell.ColorYellow)

	// Store reference to command bar
	a.views["cmdBar"] = cmdBar

	return cmdBar
}

// showCommandBar displays the command bar and enters command mode
func (a *App) showCommandBar() {
	a.cmdMode = true
	a.cmdBuffer = ""
	a.cmdSuggestion = ""

	// Update command bar display
	if cmdBar, ok := a.views["cmdBar"].(*tview.TextView); ok {
		cmdBar.SetText(":")
		cmdBar.SetTextColor(tcell.ColorYellow)
		cmdBar.SetBackgroundColor(tcell.ColorBlack)
		cmdBar.SetBorderColor(tcell.ColorYellow) // Highlight border when active
	}

	// Set focus to command bar
	a.SetFocus(a.views["cmdBar"])
}

// hideCommandBar hides the command bar and exits command mode
func (a *App) hideCommandBar() {
	a.cmdMode = false
	a.cmdBuffer = ""
	a.cmdSuggestion = ""

	// Clear command bar display
	if cmdBar, ok := a.views["cmdBar"].(*tview.TextView); ok {
		cmdBar.SetText("")
		cmdBar.SetBorderColor(tcell.ColorBlue) // Restore normal border color
	}

	// Restore focus to previous view
	a.restoreFocusAfterModal()
}

// executeCommand executes the current command
func (a *App) executeCommand(cmd string) {
	// Add to history
	a.addToHistory(cmd)

	// Parse and execute command
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "labels", "l":
		a.executeLabelsCommand(args)
	case "search", "s":
		a.executeSearchCommand(args)
	case "inbox", "i":
		a.executeInboxCommand(args)
	case "compose", "c":
		a.executeComposeCommand(args)
	case "help", "h", "?":
		a.executeHelpCommand(args)
	case "quit", "q":
		a.executeQuitCommand(args)
	default:
		a.showError(fmt.Sprintf("Unknown command: %s", command))
	}
}

// executeLabelsCommand handles labels-related commands
func (a *App) executeLabelsCommand(args []string) {
	if len(args) == 0 {
		// Show labels view
		go a.manageLabels()
		return
	}

	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "add", "a":
		if len(args) < 2 {
			a.showError("Usage: labels add <label_name>")
			return
		}
		a.executeLabelAdd(args[1:])
	case "remove", "r", "rm":
		if len(args) < 2 {
			a.showError("Usage: labels remove <label_name>")
			return
		}
		a.executeLabelRemove(args[1:])
	case "list", "ls":
		go a.manageLabels()
	default:
		a.showError(fmt.Sprintf("Unknown labels subcommand: %s", subcommand))
	}
}

// executeLabelAdd adds a label to the current message
func (a *App) executeLabelAdd(args []string) {
	labelName := strings.Join(args, " ")
	if labelName == "" {
		a.showError("Label name cannot be empty")
		return
	}

	// Get current message ID
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}

	// Create label if it doesn't exist, then apply it
	go func() {
		label, err := a.Client.CreateLabel(labelName)
		if err != nil {
			// Label might already exist, try to find it
			labels, err := a.Client.ListLabels()
			if err != nil {
				a.showError(fmt.Sprintf("❌ Error creating/finding label: %v", err))
				return
			}

			// Find existing label
			for _, l := range labels {
				if l.Name == labelName {
					label = l
					break
				}
			}

			if label == nil {
				a.showError(fmt.Sprintf("❌ Error creating label: %v", err))
				return
			}
		}

		// Apply label to message
		err = a.Client.ApplyLabel(messageID, label.Id)
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error applying label: %v", err))
			return
		}

		a.showStatusMessage(fmt.Sprintf("🏷️  Applied label: %s", labelName))
	}()
}

// executeLabelRemove removes a label from the current message
func (a *App) executeLabelRemove(args []string) {
	labelName := strings.Join(args, " ")
	if labelName == "" {
		a.showError("Label name cannot be empty")
		return
	}

	// Get current message ID
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}

	// Find and remove label
	go func() {
		labels, err := a.Client.ListLabels()
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading labels: %v", err))
			return
		}

		// Find label by name
		var labelID string
		for _, l := range labels {
			if l.Name == labelName {
				labelID = l.Id
				break
			}
		}

		if labelID == "" {
			a.showError(fmt.Sprintf("❌ Label not found: %s", labelName))
			return
		}

		// Remove label from message
		err = a.Client.RemoveLabel(messageID, labelID)
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error removing label: %v", err))
			return
		}

		a.showStatusMessage(fmt.Sprintf("🏷️  Removed label: %s", labelName))
	}()
}

// executeSearchCommand handles search commands
func (a *App) executeSearchCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: search <query>")
		return
	}

	query := strings.Join(args, " ")
	go a.performSearch(query)
}

// executeInboxCommand handles inbox commands
func (a *App) executeInboxCommand(args []string) {
	go a.reloadMessages()
}

// executeComposeCommand handles compose commands
func (a *App) executeComposeCommand(args []string) {
	go a.composeMessage(false)
}

// executeHelpCommand handles help commands
func (a *App) executeHelpCommand(args []string) {
	a.toggleHelp()
}

// executeQuitCommand handles quit commands
func (a *App) executeQuitCommand(args []string) {
	a.cancel()
	a.Stop()
}

// getCurrentMessageID gets the ID of the currently selected message
func (a *App) getCurrentMessageID() string {
	if a.currentFocus == "list" {
		if list, ok := a.views["list"].(*tview.List); ok {
			selectedIndex := list.GetCurrentItem()
			if selectedIndex >= 0 && selectedIndex < len(a.ids) {
				return a.ids[selectedIndex]
			}
		}
	} else if a.currentFocus == "text" {
		// If we're in text view, get the current message from list
		if list, ok := a.views["list"].(*tview.List); ok {
			selectedIndex := list.GetCurrentItem()
			if selectedIndex >= 0 && selectedIndex < len(a.ids) {
				return a.ids[selectedIndex]
			}
		}
	}
	return ""
}

// addToHistory adds a command to the history
func (a *App) addToHistory(cmd string) {
	// Don't add empty commands or duplicates
	if cmd == "" || (len(a.cmdHistory) > 0 && a.cmdHistory[len(a.cmdHistory)-1] == cmd) {
		return
	}

	a.cmdHistory = append(a.cmdHistory, cmd)
	if len(a.cmdHistory) > 100 {
		a.cmdHistory = a.cmdHistory[1:]
	}
	a.cmdHistoryIndex = len(a.cmdHistory)
}

// getListWidth returns current inner width of the list view or a sensible fallback
func (a *App) getListWidth() int {
	if list, ok := a.views["list"].(*tview.List); ok {
		_, _, w, _ := list.GetInnerRect()
		if w > 0 {
			return w
		}
	}
	if a.screenWidth > 0 {
		return a.screenWidth
	}
	return 80
}

// getFormatWidth devuelve el ancho disponible para el texto de las filas
func (a *App) getFormatWidth() int {
	if list, ok := a.views["list"].(*tview.List); ok {
		_, _, w, _ := list.GetInnerRect()
		if w > 10 {
			// Reservamos 2 para el prefijo ●/○
			return w - 2
		}
	}
	// Fallback conservador
	if a.screenWidth > 0 {
		return a.screenWidth - 2
	}
	return 78
}

// refreshMessageContent reloads the message and updates the text view without changing focus
func (a *App) refreshMessageContent(id string) {
	if id == "" {
		return
	}
	go func() {
		m, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			return
		}
		var content strings.Builder
		header := a.emailRenderer.FormatHeaderStyled(m.Subject, m.From, m.Date, m.Labels)
		content.WriteString(header)
		if m.PlainText != "" {
			content.WriteString(m.PlainText)
		} else {
			content.WriteString("No text content available")
		}
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.SetText(content.String())
				text.ScrollToBeginning()
			}
		})
	}()
}

// refreshMessageContentWithOverride reloads message and overrides labels shown with provided names
func (a *App) refreshMessageContentWithOverride(id string, labelsOverride []string) {
	if id == "" {
		return
	}
	go func() {
		m, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			return
		}
		// Merge override labels
		if len(labelsOverride) > 0 {
			// Build set for merge
			seen := make(map[string]struct{}, len(m.Labels)+len(labelsOverride))
			merged := make([]string, 0, len(m.Labels)+len(labelsOverride))
			for _, l := range m.Labels {
				if _, ok := seen[l]; !ok {
					seen[l] = struct{}{}
					merged = append(merged, l)
				}
			}
			for _, l := range labelsOverride {
				if _, ok := seen[l]; !ok {
					seen[l] = struct{}{}
					merged = append(merged, l)
				}
			}
			m.Labels = merged
		}

		var content strings.Builder
		header := a.emailRenderer.FormatHeaderStyled(
			m.Subject,
			m.From,
			m.Date,
			m.Labels,
		)
		content.WriteString(header)
		if m.PlainText != "" {
			content.WriteString(m.PlainText)
		} else {
			content.WriteString("No text content available")
		}

		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.SetText(content.String())
				text.ScrollToBeginning()
			}
		})
	}()
}

// updateCachedMessageLabels updates the cached labels for a message ID
func (a *App) updateCachedMessageLabels(messageID, labelID string, applied bool) {
	// Find index
	var idx = -1
	for i, id := range a.ids {
		if id == messageID {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(a.messagesMeta) || a.messagesMeta[idx] == nil {
		return
	}
	msg := a.messagesMeta[idx]
	if applied {
		// add if not exists
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
		// remove
		out := msg.LabelIds[:0]
		for _, l := range msg.LabelIds {
			if l != labelID {
				out = append(out, l)
			}
		}
		msg.LabelIds = out
	}
}

// moveSelected opens the labels picker to choose a destination label, applies it, then archives the message
func (a *App) moveSelected() {
	// Get the current message ID
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}

	// Load available labels and message metadata
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading labels: %v", err))
		return
	}
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}

	a.showMoveLabelsView(labels, message)
}

// showMoveLabelsView lets user choose a label to apply and then archives the message (move semantics)
func (a *App) showMoveLabelsView(labels []*gmailapi.Label, message *gmailapi.Message) {
	picker := tview.NewList().ShowSecondaryText(false)
	picker.SetBorder(true)
	picker.SetTitle(" 📦 Move to label ")

	// Build list of candidate labels with applied first
	curr := make(map[string]bool)
	for _, l := range message.LabelIds {
		curr[l] = true
	}
	applied, notApplied := a.partitionAndSortLabels(labels, curr)
	for _, label := range append(applied, notApplied...) {
		// Store values for closure
		labelID := label.Id
		labelName := label.Name
		picker.AddItem(labelName, "", 0, func() {
			go func() {
				// Apply label if not already present
				has := false
				for _, l := range message.LabelIds {
					if l == labelID {
						has = true
						break
					}
				}
				if !has {
					if err := a.Client.ApplyLabel(message.Id, labelID); err != nil {
						a.showError(fmt.Sprintf("❌ Error applying label: %v", err))
						return
					}
					// Update cache
					a.updateCachedMessageLabels(message.Id, labelID, true)
				}
				// Archive (remove INBOX)
				if err := a.Client.ArchiveMessage(message.Id); err != nil {
					a.showError(fmt.Sprintf("❌ Error archiving: %v", err))
					return
				}
				a.showStatusMessage(fmt.Sprintf("📦 Moved to: %s", labelName))

				// Remove from current list since we show INBOX only
				a.QueueUpdateDraw(func() {
					// Find index
					idx := -1
					for i, id := range a.ids {
						if id == message.Id {
							idx = i
							break
						}
					}
					if idx >= 0 {
						a.ids = append(a.ids[:idx], a.ids[idx+1:]...)
						if idx < len(a.messagesMeta) {
							a.messagesMeta = append(a.messagesMeta[:idx], a.messagesMeta[idx+1:]...)
						}
						if list, ok := a.views["list"].(*tview.List); ok {
							list.RemoveItem(idx)
							list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
						}
					}
					// Return to main
					a.Pages.SwitchToPage("main")
					a.restoreFocusAfterModal()
				})
			}()
		})
	}

	// Basic keys
	picker.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			return nil
		}
		return event
	})

	// Container view
	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select destination label and press Enter. ESC to cancel")
	v.AddItem(title, 3, 0, false)
	v.AddItem(picker, 0, 1, true)

	a.Pages.AddPage("moveLabels", v, true, true)
	a.Pages.SwitchToPage("moveLabels")
	if picker.GetItemCount() > 0 {
		picker.SetCurrentItem(0)
	}
	a.SetFocus(picker)
}

// filterAndSortLabels filters out system labels and returns a name-sorted slice
func (a *App) filterAndSortLabels(labels []*gmailapi.Label) []*gmailapi.Label {
	filtered := make([]*gmailapi.Label, 0, len(labels))
	for _, l := range labels {
		if strings.HasPrefix(l.Id, "CATEGORY_") || l.Id == "INBOX" || l.Id == "SENT" || l.Id == "DRAFT" ||
			l.Id == "SPAM" || l.Id == "TRASH" || l.Id == "CHAT" || (strings.HasSuffix(l.Id, "_STARRED") && l.Id != "STARRED") {
			continue
		}
		filtered = append(filtered, l)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})
	return filtered
}

// partitionAndSortLabels returns two sorted slices: labels applied to current and the rest
func (a *App) partitionAndSortLabels(labels []*gmailapi.Label, current map[string]bool) ([]*gmailapi.Label, []*gmailapi.Label) {
	filtered := a.filterAndSortLabels(labels)
	applied := make([]*gmailapi.Label, 0)
	notApplied := make([]*gmailapi.Label, 0)
	for _, l := range filtered {
		if current[l.Id] {
			applied = append(applied, l)
		} else {
			notApplied = append(notApplied, l)
		}
	}
	// Already sorted by name from filterAndSortLabels; preserve order
	return applied, notApplied
}

// toggleAISummary shows/hides the AI summary pane and triggers generation if needed
func (a *App) toggleAISummary() {
	// If pane is visible and we're currently in the summary, hide it
	if a.aiSummaryVisible && a.currentFocus == "summary" {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.aiSummaryView, 0, 0)
		}
		a.aiSummaryVisible = false
		a.SetFocus(a.views["text"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
		a.showStatusMessage("🙈 AI summary hidden")
		return
	}

	// Ensure we have a message ID
	mid := a.getCurrentMessageID()
	if mid == "" && len(a.ids) > 0 {
		// Force display of selected message
		if list, ok := a.views["list"].(*tview.List); ok {
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(a.ids) {
				mid = a.ids[idx]
				go a.showMessage(mid)
			}
		}
	}
	if mid == "" {
		a.showError("No message selected")
		return
	}

	// Ensure message content is loaded without stealing focus (do this first)
	if mid != "" {
		a.showMessageWithoutFocus(mid)
	}

	// Show pane (50/50) and focus immediately (we are on the UI thread)
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.aiSummaryView, 0, 1)
	}
	a.aiSummaryVisible = true
	a.SetFocus(a.aiSummaryView)
	a.currentFocus = "summary"
	if a.aiSummaryView != nil {
		a.aiSummaryView.SetBorderColor(tcell.ColorYellow)
	}
	a.updateFocusIndicators("summary")

	// Populate pane
	a.generateOrShowSummary(mid)
}

// generateOrShowSummary shows cached summary or triggers generation if missing
func (a *App) generateOrShowSummary(messageID string) {
	if a.aiSummaryView == nil {
		return
	}
	if sum, ok := a.aiSummaryCache[messageID]; ok && sum != "" {
		a.aiSummaryView.SetText(sum)
		a.aiSummaryView.ScrollToBeginning()
		a.setStatusPersistent("🤖 Summary loaded from cache")
		return
	}
	if a.aiInFlight[messageID] {
		a.aiSummaryView.SetText("🧠 Summarizing…")
		a.aiSummaryView.ScrollToBeginning()
		a.setStatusPersistent("🧠 Summarizing…")
		return
	}
	// Start generation
	a.aiSummaryView.SetText("🧠 Summarizing…")
	a.aiSummaryView.ScrollToBeginning()
	a.setStatusPersistent("🧠 Summarizing…")
	a.aiInFlight[messageID] = true
	go func(id string) {
		m, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			a.QueueUpdateDraw(func() {
				a.aiSummaryView.SetText("⚠️ Error loading message")
				a.showStatusMessage("⚠️ Error loading message")
			})
			delete(a.aiInFlight, id)
			return
		}
		if a.LLM == nil {
			a.QueueUpdateDraw(func() { a.aiSummaryView.SetText("⚠️ LLM disabled"); a.showStatusMessage("⚠️ LLM disabled") })
			delete(a.aiInFlight, id)
			return
		}
		body := m.PlainText
		if len([]rune(body)) > 8000 {
			body = string([]rune(body)[:8000])
		}
		resp, err := a.LLM.Generate("Summarize in 3 bullet points (keep language).\n\n" + body)
		if err != nil {
			a.QueueUpdateDraw(func() { a.aiSummaryView.SetText("⚠️ LLM error"); a.showStatusMessage("⚠️ LLM error") })
			delete(a.aiInFlight, id)
			return
		}
		a.aiSummaryCache[id] = resp
		delete(a.aiInFlight, id)
		a.QueueUpdateDraw(func() {
			a.aiSummaryView.SetText(resp)
			a.aiSummaryView.ScrollToBeginning()
			a.showStatusMessage("✅ Summary ready")
		})
	}(messageID)
}

// showAllLabelsPicker shows a list of all actionable labels to apply one to the message
func (a *App) showAllLabelsPicker(messageID string) {
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError("❌ Error loading labels")
		return
	}
	// Get current message labels to mark applied ones
	msg, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError("❌ Error loading message")
		return
	}
	current := make(map[string]bool, len(msg.LabelIds))
	for _, lid := range msg.LabelIds {
		current[lid] = true
	}
	// Build sorted actionable labels with applied first
	applied, notApplied := a.partitionAndSortLabels(labels, current)
	all := append(applied, notApplied...)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true)
	list.SetTitle(" 🗂️  All Labels ")

	// Map name -> id
	nameToID := make(map[string]string, len(all))
	for _, l := range all {
		nameToID[l.Name] = l.Id
	}

	for _, l := range all {
		lbl := l.Name
		icon := "○ "
		if current[l.Id] {
			icon = "✅ "
		}
		display := icon + lbl
		list.AddItem(display, "", 0, func() {
			if id, ok := nameToID[lbl]; ok {
				a.applyLabelAndRefresh(messageID, id, lbl)
				a.showStatusMessage("✅ Applied: " + lbl)
				a.Pages.SwitchToPage("main")
				a.restoreFocusAfterModal()
			}
		})
	}

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("aiLabelSuggestions")
			return nil
		}
		return e
	})

	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select a label to apply | Enter=apply, ESC=back")
	v.AddItem(title, 3, 0, false)
	v.AddItem(list, 0, 1, true)
	a.Pages.AddPage("aiAllLabels", v, true, true)
	a.Pages.SwitchToPage("aiAllLabels")
	if list.GetItemCount() > 0 {
		list.SetCurrentItem(0)
	}
	a.SetFocus(list)
}

// applyLabelAndRefresh aplica una etiqueta usando el mismo mecanismo que en la vista de 'l'
// y refresca el contenido del mensaje cuando termina
func (a *App) applyLabelAndRefresh(messageID, labelID, labelName string) {
	// Asumimos que queremos aplicar (no togglear a quitar), por lo que pasamos isCurrentlyApplied=false
	a.toggleLabelForMessage(messageID, labelID, labelName, false, func(newApplied bool, err error) {
		if err != nil {
			return
		}
		if newApplied {
			// Mantener cache de metadatos coherente
			a.updateCachedMessageLabels(messageID, labelID, true)
			// Refrescar contenido desde servidor (evita desincronización)
			a.refreshMessageContent(messageID)
		}
	})
}
