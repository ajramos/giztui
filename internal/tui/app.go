package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/llm"
	"github.com/ajramos/gmail-tui/internal/render"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// App encapsulates the terminal UI and the Gmail client
type App struct {
	*tview.Application
	Pages   *Pages
	Config  *config.Config
	Client  *gmail.Client
	LLM     *llm.Client
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
	ids         []string
	draftMode   bool
	draftIDs    []string
	showHelp    bool
	currentView string
	// Layout management
	currentLayout LayoutType
	screenWidth   int
	screenHeight  int
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
func NewApp(client *gmail.Client, llm *llm.Client, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		Application:   tview.NewApplication(),
		Config:        cfg,
		Client:        client,
		LLM:           llm,
		Keys:          cfg.Keys,
		ctx:           ctx,
		cancel:        cancel,
		views:         make(map[string]tview.Primitive),
		cmdBuff:       NewCmdBuff(),
		flash:         NewFlash(),
		actions:       NewKeyActions(),
		emailRenderer: render.NewEmailRenderer(),
		ids:           []string{},
		draftMode:     false,
		draftIDs:      []string{},
		showHelp:      false,
		currentView:   "messages",
		currentLayout: LayoutMedium,
		screenWidth:   80,
		screenHeight:  25,
	}

	// Initialize pages
	app.Pages = NewPages()

	// Initialize components
	app.initComponents()

	// Set up key bindings
	app.bindKeys()

	// Initialize views
	app.initViews()

	return app
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
	list := tview.NewList().ShowSecondaryText(true)
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

	// Store components
	a.views["list"] = list
	a.views["text"] = text
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
}

// createMainLayout creates the main application layout
func (a *App) createMainLayout() tview.Primitive {
	layout := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add flash notification at the top (hidden by default)
	layout.AddItem(a.flash.textView, 0, 0, false)

	// Add main content area
	content := tview.NewFlex().SetDirection(tview.FlexRow)
	content.AddItem(a.views["list"], 0, 3, true)
	content.AddItem(a.views["text"], 0, 2, false)

	layout.AddItem(content, 0, 1, true)

	// Add status bar at the bottom
	statusBar := a.createStatusBar()
	layout.AddItem(statusBar, 1, 0, false)

	return layout
}

// createStatusBar creates the status bar
func (a *App) createStatusBar() tview.Primitive {
	status := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText("Gmail TUI | Press ? for help | Press q to quit")

	return status
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
	help.WriteString("a         📁 Archive message\n\n")

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
		// Only intercept specific keys, let navigation keys pass through
		switch event.Rune() {
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
			go a.manageLabels()
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
				go a.summarizeSelected()
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

	// Handle Enter key for viewing messages
	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			if index < len(a.ids) {
				go a.showMessage(a.ids[index])
			}
		})
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

	messages, err := a.Client.ListMessages(50)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error loading messages: %v", err))
		return
	}

	// Show success message if no messages
	if len(messages) == 0 {
		if list, ok := a.views["list"].(*tview.List); ok {
			list.SetTitle(" 📧 No messages found ")
		}
		a.showInfo("📧 No messages found in your inbox")
		return
	}

	// Get screen width for proper formatting
	screenWidth := 80
	if a.screenWidth > 0 {
		screenWidth = a.screenWidth
	}

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

		// Update title periodically
		if (i+1)%10 == 0 {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.SetTitle(fmt.Sprintf(" 🔄 Loading... (%d/%d) ", i+1, len(messages)))
			}
			a.Draw()
		}
	}

	if list, ok := a.views["list"].(*tview.List); ok {
		list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
	}

	// Set focus back to list
	a.SetFocus(a.views["list"])
}

// showMessage displays a message in the text view
func (a *App) showMessage(id string) {
	// Show loading message immediately
	if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetText("Loading message...")
		text.ScrollToBeginning()
	}
	a.Draw()

	// Load message content in background
	go func() {
		message, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading message: %v", err))
			return
		}

		var content strings.Builder

		// Clean Gmail-style header with blue color
		content.WriteString(fmt.Sprintf("Subject: %s\n", message.Subject))
		content.WriteString(fmt.Sprintf("From: %s\n", message.From))
		content.WriteString(fmt.Sprintf("Date: %s\n", message.Date.Format("Mon, Jan 2, 2006 at 3:04 PM")))

		// Show labels if any
		if len(message.Labels) > 0 {
			content.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(message.Labels, ", ")))
		}

		content.WriteString("\n")

		// Message content
		if message.PlainText != "" {
			content.WriteString(message.PlainText)
		} else {
			content.WriteString("No text content available")
		}

		// Update UI in main thread
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetText(content.String())
				// Scroll to the top of the text
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

		message, err := a.Client.GetMessageWithContent(msg.Id)
		if err != nil {
			if list, ok := a.views["list"].(*tview.List); ok {
				list.AddItem(fmt.Sprintf("⚠️  Error loading message %d", i+1), "Failed to load", 0, nil)
			}
			continue
		}

		subject := message.Subject
		if subject == "" {
			subject = "(No subject)"
		}

		from := message.From
		if from == "" {
			from = "(No sender)"
		} else {
			// Try to extract just the name part if it's in "Name <email@domain.com>" format
			if strings.Contains(from, "<") && strings.Contains(from, ">") {
				parts := strings.Split(from, "<")
				if len(parts) > 0 {
					name := strings.TrimSpace(parts[0])
					if name != "" {
						from = name
					}
				}
			}
		}

		// Check if unread
		unread := false
		for _, label := range message.Labels {
			if label == "UNREAD" {
				unread = true
				break
			}
		}

		// Format like Gmail: Sender | Subject | Date
		// Truncate sender name if too long (handle UTF-8 properly)
		senderName := from
		if len([]rune(senderName)) > 20 {
			runes := []rune(senderName)
			senderName = string(runes[:17]) + "..."
		}

		// Truncate subject if too long (handle UTF-8 properly)
		displaySubject := subject
		if len([]rune(displaySubject)) > 40 {
			runes := []rune(displaySubject)
			displaySubject = string(runes[:37]) + "..."
		}

		// Format date like Gmail (relative time)
		dateStr := "unknown"
		if !message.Date.IsZero() {
			dateStr = formatRelativeTime(message.Date)
		}

		// Create main text: Sender | Subject | Date
		// Use fixed width for sender name to ensure alignment
		senderWidth := 20
		subjectWidth := 40

		if len([]rune(senderName)) > senderWidth {
			runes := []rune(senderName)
			senderName = string(runes[:senderWidth-3]) + "..."
		}

		// Pad subject to fixed width
		if len([]rune(displaySubject)) > subjectWidth {
			runes := []rune(displaySubject)
			displaySubject = string(runes[:subjectWidth-3]) + "..."
		}

		// Create main text with fixed widths and date right-aligned
		// Use a table-like format with proper spacing
		mainText := fmt.Sprintf("%-*s | %-*s | %s",
			senderWidth, senderName,
			subjectWidth, displaySubject,
			dateStr)

		// Add unread indicator
		if unread {
			mainText = "● " + mainText
		} else {
			mainText = "○ " + mainText
		}

		if list, ok := a.views["list"].(*tview.List); ok {
			list.AddItem(mainText, "", 0, nil)
		}
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
	a.mu.Lock()
	defer a.mu.Unlock()

	// Get the currently selected item
	list, ok := a.views["list"].(*tview.List)
	if !ok {
		a.showError("❌ Could not access message list")
		return
	}

	selectedIndex := list.GetCurrentItem()
	if selectedIndex < 0 || selectedIndex >= len(a.ids) {
		a.showError("❌ No message selected")
		return
	}

	messageID := a.ids[selectedIndex]
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
			a.showInfo("✅ Message marked as read")
		} else {
			a.showError(fmt.Sprintf("❌ Error marking as read: %v", err2))
			return
		}
	} else {
		// Mark as unread
		err2 = a.Client.MarkAsUnread(messageID)
		if err2 == nil {
			a.showInfo("✅ Message marked as unread")
		} else {
			a.showError(fmt.Sprintf("❌ Error marking as unread: %v", err2))
			return
		}
	}

	// Update the UI to reflect the change
	a.updateMessageDisplay(selectedIndex, !isUnread)
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

	// Get screen width for proper formatting
	screenWidth := 80
	if a.screenWidth > 0 {
		screenWidth = a.screenWidth
	}

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
	a.showInfo("Trash functionality not yet implemented")
}

func (a *App) archiveSelected() {
	a.showInfo("Archive functionality not yet implemented")
}

func (a *App) replySelected() {
	a.showInfo("Reply functionality not yet implemented")
}

func (a *App) showAttachments() {
	a.showInfo("Attachments functionality not yet implemented")
}

func (a *App) manageLabels() {
	a.showInfo("Labels functionality not yet implemented")
}

func (a *App) summarizeSelected() {
	a.showInfo("Summarize functionality not yet implemented")
}

func (a *App) generateReply() {
	a.showInfo("Generate reply functionality not yet implemented")
}

func (a *App) suggestLabel() {
	a.showInfo("Suggest label functionality not yet implemented")
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

// toggleFocus switches focus between list and text view
func (a *App) toggleFocus() {
	currentFocus := a.GetFocus()

	if currentFocus == a.views["list"] {
		a.SetFocus(a.views["text"])
	} else {
		a.SetFocus(a.views["list"])
	}
}
