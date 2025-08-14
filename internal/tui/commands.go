package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// emojiBox draws a single piece of text (emoji-safe) at its top-left without markup.
type emojiBox struct {
	*tview.Box
	text  string
	color tcell.Color
}

func newEmojiBox(text string, color tcell.Color) *emojiBox {
	b := &emojiBox{Box: tview.NewBox(), text: text, color: color}
	b.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	return b
}

func (e *emojiBox) Draw(screen tcell.Screen) {
	e.Box.DrawForSubclass(screen, e)
	x, y, w, _ := e.GetInnerRect()
	if w <= 0 {
		return
	}
	// Print handles wide runes properly.
	tview.Print(screen, e.text, x, y, w, tview.AlignLeft, e.color)
}

// createCommandBar creates the command bar component (k9s style)
func (a *App) createCommandBar() tview.Primitive {
	cmdBar := tview.NewTextView()
	cmdBar.SetDynamicColors(true)
	cmdBar.SetTextAlign(tview.AlignLeft)
	// Inner widget without its own border; the panel provides the border and title
	cmdBar.SetBorder(false)
	cmdBar.SetText("")
	cmdBar.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	cmdBar.SetTextColor(tview.Styles.PrimaryTextColor)

	// Store reference to command bar
	a.views["cmdBar"] = cmdBar

	return cmdBar
}

// showCommandBar displays the command bar and enters command mode
func (a *App) showCommandBar() {
	a.cmdMode = true
	a.cmdBuffer = ""
	a.cmdSuggestion = ""

	// Build prompt pieces with an emoji-safe custom box
	dog := newEmojiBox("🐶>", tview.Styles.PrimaryTextColor)

	input := tview.NewInputField()
	input.SetFieldWidth(0)
	input.SetPlaceholder("")
	input.SetPlaceholderTextColor(tview.Styles.PrimaryTextColor)
	input.SetBorder(false)
	input.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	input.SetText("")
	input.SetDoneFunc(nil) // ensure we set it after capture
	// Start at end of history
	a.cmdHistoryIndex = len(a.cmdHistory)
	// Behaviors: Enter executes, ESC closes, Tab completes, Up/Down history
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			cmd := input.GetText()
			a.executeCommand(cmd)
			a.hideCommandBar()
		}
	})
	// Suggestion view on the right
	hint := tview.NewTextView()
	hint.SetBorder(false)
	hint.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	hint.SetText("")
	hint.SetTextColor(tcell.ColorGray)

	input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEscape:
			a.hideCommandBar()
			return nil
		case tcell.KeyTab:
			// Complete using context-aware suggestion; may return full-line replacement
			cur := strings.TrimSpace(input.GetText())
			s := a.generateCommandSuggestion(cur)
			if s != "" && s != cur {
				input.SetText(s)
			}
			return nil
		case tcell.KeyUp:
			if a.cmdHistoryIndex > 0 {
				a.cmdHistoryIndex--
				if a.cmdHistoryIndex >= 0 && a.cmdHistoryIndex < len(a.cmdHistory) {
					input.SetText(a.cmdHistory[a.cmdHistoryIndex])
				}
			}
			return nil
		case tcell.KeyDown:
			if a.cmdHistoryIndex < len(a.cmdHistory)-1 {
				a.cmdHistoryIndex++
				if a.cmdHistoryIndex >= 0 && a.cmdHistoryIndex < len(a.cmdHistory) {
					input.SetText(a.cmdHistory[a.cmdHistoryIndex])
				}
			} else {
				a.cmdHistoryIndex = len(a.cmdHistory)
				input.SetText("")
			}
			return nil
		}
		return ev
	})
	// Keep cmdBuffer in sync (for history/addToHistory consistency if used elsewhere)
	input.SetChangedFunc(func(text string) {
		a.cmdBuffer = text
		// Update live hint based on current buffer
		cur := strings.TrimSpace(text)
		s := a.generateCommandSuggestion(cur)
		if s != "" && s != cur {
			hint.SetText("[" + s + "]")
		} else {
			hint.SetText("")
		}
	})

	row := tview.NewFlex().SetDirection(tview.FlexColumn)
	row.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	row.AddItem(dog, 3, 0, false)
	row.AddItem(input, 0, 1, true)
	row.AddItem(hint, 0, 1, false)

	// Mount into cmdPanel and resize panel height
	if cp, ok := a.views["cmdPanel"].(*tview.Flex); ok {
		cp.Clear()
		cp.AddItem(row, 1, 0, true)
		if mainFlex, ok2 := a.views["mainFlex"].(*tview.Flex); ok2 {
			mainFlex.ResizeItem(cp, 3, 0)
		}
	}

	a.views["cmdPromptDog"] = dog
	a.views["cmdInput"] = input
	a.views["cmdHint"] = hint
	a.currentFocus = "cmd"
	a.updateFocusIndicators("cmd")
	a.SetFocus(input)
}

// hideCommandBar hides the command bar and exits command mode
func (a *App) hideCommandBar() {
	a.cmdMode = false
	a.cmdBuffer = ""
	a.cmdSuggestion = ""

	if cmdBar, ok := a.views["cmdBar"].(*tview.TextView); ok {
		cmdBar.SetText("")
		cmdBar.SetBorderColor(tcell.ColorBlue)
	}
	// Hide cmdPanel by clearing its content and resizing to height 0
	if cp, ok := a.views["cmdPanel"].(*tview.Flex); ok {
		cp.Clear()
		if mainFlex, ok2 := a.views["mainFlex"].(*tview.Flex); ok2 {
			mainFlex.ResizeItem(cp, 0, 0)
		}
	}

	a.restoreFocusAfterModal()
}

// handleCommandInput handles input when in command mode
func (a *App) handleCommandInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		a.hideCommandBar()
		return nil
	case tcell.KeyEnter:
		a.executeCommand(a.cmdBuffer)
		a.hideCommandBar()
		return nil
	case tcell.KeyTab:
		a.completeCommand()
		return nil
	case tcell.KeyBackspace, tcell.KeyDelete:
		if len(a.cmdBuffer) > 0 {
			a.cmdBuffer = a.cmdBuffer[:len(a.cmdBuffer)-1]
			a.updateCommandBar()
		}
		return nil
	case tcell.KeyUp:
		if a.cmdHistoryIndex > 0 {
			a.cmdHistoryIndex--
			if a.cmdHistoryIndex >= 0 && a.cmdHistoryIndex < len(a.cmdHistory) {
				a.cmdBuffer = a.cmdHistory[a.cmdHistoryIndex]
				a.updateCommandBar()
			}
		}
		return nil
	case tcell.KeyDown:
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

	if event.Rune() != 0 {
		a.cmdBuffer += string(event.Rune())
		a.updateCommandBar()
		return nil
	}
	return event
}

// updateCommandBar updates the command bar display
func (a *App) updateCommandBar() {
	// Kept for backward compatibility if cmdBar is used elsewhere; no-op with new InputField
}

// generateCommandSuggestion generates a suggestion based on the current command buffer
func (a *App) generateCommandSuggestion(buffer string) string {
	if buffer == "" {
		return ""
	}

	// First-level suggestions
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
		"g":       {"G"},
		"G":       {"G"},
		"1":       {"1"},
		"$":       {"$"},
		"5":       {"5"},
		"10":      {"10"},
	}

	if suggestions, exists := commands[buffer]; exists && len(suggestions) > 0 {
		return suggestions[0]
	}
	for cmd, suggestions := range commands {
		if strings.HasPrefix(cmd, buffer) && cmd != buffer {
			return suggestions[0]
		}
	}

	// Contextual arguments for 'search'
	if strings.HasPrefix(buffer, "search ") || strings.HasPrefix(buffer, "s ") {
		arg := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(buffer, "search"), "s"))
		arg = strings.TrimSpace(arg)
		// If empty or partial, suggest full tokens
		// Support: from:current | to:current | subject:current | domain:current
		opts := []string{"search from:current", "search to:current", "search subject:current", "search domain:current"}
		if arg == "" {
			return opts[0]
		}
		// Suggest the first option that starts with current arg
		for _, o := range opts {
			if strings.HasPrefix(o, buffer) {
				return o
			}
		}
		// If user typed token start after space (e.g., "search f"), expand to from:current
		tail := strings.TrimSpace(strings.TrimPrefix(buffer, "search"))
		tail = strings.TrimSpace(tail)
		lower := strings.ToLower(tail)
		switch {
		case strings.HasPrefix("from:current", lower):
			return "search from:current"
		case strings.HasPrefix("to:current", lower):
			return "search to:current"
		case strings.HasPrefix("subject:current", lower):
			return "search subject:current"
		case strings.HasPrefix("domain:current", lower):
			return "search domain:current"
		}
	}

	// Contextual help for G command
	if strings.HasPrefix(buffer, "G ") {
		return "G <message_number>"
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

// executeCommand executes the current command
func (a *App) executeCommand(cmd string) {
	a.addToHistory(cmd)

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
	case "summary":
		a.executeSummaryCommand(args)
	case "rsvp":
		a.executeRSVPCommand(args)
	case "inbox", "i":
		a.executeInboxCommand(args)
	case "compose", "c":
		a.executeComposeCommand(args)
	case "help", "h", "?":
		a.executeHelpCommand(args)
	case "quit", "q":
		a.executeQuitCommand(args)
	case "g", "G":
		a.executeGoToCommand(args)
	default:
		// Check for numeric shortcuts like :1, :$
		if matched := a.executeNumericShortcut(command); !matched {
			a.showError(fmt.Sprintf("Unknown command: %s", command))
		}
	}
}

// executeRSVPCommand handles :rsvp commands
func (a *App) executeRSVPCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: rsvp <accept|tentative|decline>")
		return
	}
	choice := strings.ToLower(args[0])
	switch choice {
	case "accept", "yes", "y":
		go a.sendRSVP("ACCEPTED", "")
	case "tentative", "maybe", "m":
		go a.sendRSVP("TENTATIVE", "")
	case "decline", "no", "n":
		go a.sendRSVP("DECLINED", "")
	default:
		a.showError("Usage: rsvp <accept|tentative|decline>")
	}
}

// addToHistory adds a command to the history
func (a *App) addToHistory(cmd string) {
	if cmd == "" || (len(a.cmdHistory) > 0 && a.cmdHistory[len(a.cmdHistory)-1] == cmd) {
		return
	}
	a.cmdHistory = append(a.cmdHistory, cmd)
	if len(a.cmdHistory) > 100 {
		a.cmdHistory = a.cmdHistory[1:]
	}
	a.cmdHistoryIndex = len(a.cmdHistory)
}

// executeLabelsCommand handles labels-related commands
func (a *App) executeLabelsCommand(args []string) {
	if len(args) == 0 {
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

// executeSearchCommand handles search commands
func (a *App) executeSearchCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: search <query>")
		return
	}
	// Support contextual shorthands: from:current | to:current | subject:current | domain:current
	if len(args) == 1 && strings.Contains(args[0], ":") {
		parts := strings.SplitN(args[0], ":", 2)
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := ""
		if len(parts) > 1 {
			val = strings.ToLower(strings.TrimSpace(parts[1]))
		}
		switch key {
		case "from":
			if val == "current" {
				go a.searchByFromCurrent()
				return
			}
		case "to":
			if val == "current" {
				go a.searchByToCurrent()
				return
			}
		case "subject":
			if val == "current" {
				go a.searchBySubjectCurrent()
				return
			}
		case "domain":
			if val == "current" {
				go a.searchByDomainCurrent()
				return
			}
		}
	}
	query := strings.Join(args, " ")
	go a.performSearch(query)
}

// executeSummaryCommand handles :summary commands
func (a *App) executeSummaryCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: summary <refresh>")
		return
	}
	switch strings.ToLower(args[0]) {
	case "refresh", "regenerate", "update":
		go a.forceRegenerateSummary()
	default:
		a.showError("Usage: summary <refresh>")
	}
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
	a.closeLogger()
	a.Stop()
}

// executeGoToCommand handles :G command (go to last message) and numeric navigation
func (a *App) executeGoToCommand(args []string) {
	list, ok := a.views["list"].(*tview.Table)
	if !ok {
		return // Silently fail like the working G key
	}
	
	// Check if we have any messages
	if len(a.ids) == 0 {
		return // No messages to navigate to
	}
	
	// Default to last message (:G behavior) 
	// Last message is at table row = len(a.ids) - 1
	targetRow := len(a.ids) - 1
	
	// If argument provided (called from :5 style commands), calculate target row
	if len(args) > 0 {
		if num, err := strconv.Atoi(args[0]); err == nil && num >= 1 {
			// Convert 1-based user input to 0-based table row
			// User message 1 = table row 0, message 2 = table row 1, etc.
			maxMessage := len(a.ids) // Total number of messages
			if num > maxMessage {
				targetRow = len(a.ids) - 1 // Go to last message if number too high
			} else {
				targetRow = num - 1 // User message N = table row N-1
			}
		}
	}
	
	// Use the same simple approach as the working direct G key
	list.Select(targetRow, 0)
}

// executeNumericShortcut handles :1, :$, and pure numbers for navigation
func (a *App) executeNumericShortcut(command string) bool {
	switch command {
	case "1":
		a.executeGoToFirst()
		return true
	case "$":
		a.executeGoToCommand([]string{}) // No args = last message
		return true
	default:
		// Check if it's a pure number
		if num, err := strconv.Atoi(command); err == nil && num >= 1 {
			a.executeGoToCommand([]string{command})
			return true
		}
	}
	return false
}

// executeGoToFirst navigates to the first message (gg and :1 behavior)
func (a *App) executeGoToFirst() {
	list, ok := a.views["list"].(*tview.Table)
	if !ok {
		return // Silently fail like the working G key
	}
	
	// Check if we have any messages
	if len(a.ids) == 0 {
		return // No messages to navigate to
	}
	
	// First message is at table row 0 (maps to a.ids[0])
	list.Select(0, 0)
}

