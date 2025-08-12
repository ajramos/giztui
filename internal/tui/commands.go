package tui

import (
	"fmt"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

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

	if cmdBar, ok := a.views["cmdBar"].(*tview.TextView); ok {
		cmdBar.SetText(":")
		cmdBar.SetTextColor(tview.Styles.PrimaryTextColor)
		cmdBar.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		cmdBar.SetBorderColor(tcell.ColorYellow)
	}

	a.SetFocus(a.views["cmdBar"])
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
	if cmdBar, ok := a.views["cmdBar"].(*tview.TextView); ok {
		suggestion := a.generateCommandSuggestion(a.cmdBuffer)
		a.cmdSuggestion = suggestion

		displayText := fmt.Sprintf(":%s", a.cmdBuffer)
		if suggestion != "" && suggestion != a.cmdBuffer {
			displayText += fmt.Sprintf(" [%s]", suggestion)
		}

		cmdBar.SetText(displayText)
		cmdBar.SetTextColor(tview.Styles.PrimaryTextColor)
		cmdBar.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	}
}

// generateCommandSuggestion generates a suggestion based on the current command buffer
func (a *App) generateCommandSuggestion(buffer string) string {
	if buffer == "" {
		return ""
	}

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

	if suggestions, exists := commands[buffer]; exists && len(suggestions) > 0 {
		return suggestions[0]
	}
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
