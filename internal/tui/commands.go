package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"gopkg.in/yaml.v3"
)

// emojiBox draws a single piece of text (emoji-safe) at its top-left without markup.
type emojiBox struct {
	*tview.Box
	text  string
	color tcell.Color
}

func newEmojiBox(text string, color tcell.Color, backgroundColor tcell.Color) *emojiBox {
	b := &emojiBox{Box: tview.NewBox(), text: text, color: color}
	b.SetBackgroundColor(backgroundColor)
	return b
}

func (e *emojiBox) Draw(screen tcell.Screen) {
	e.DrawForSubclass(screen, e) // OBLITERATED: embedded field selector eliminated! 💥
	x, y, w, _ := e.GetInnerRect()
	if w <= 0 {
		return
	}
	// Print handles wide runes properly.
	tview.Print(screen, e.text, x, y, w, tview.AlignLeft, e.color)
}

// Removed unused function: createCommandBar

// showCommandBar displays the command bar and enters command mode
func (a *App) showCommandBar() {
	a.showCommandBarWithPrefix("")
}

// showCommandBarWithPrefix displays the command bar with a prefilled command
func (a *App) showCommandBarWithPrefix(prefix string) {
	a.cmdMode = true
	a.cmdBuffer = prefix
	a.cmdSuggestion = ""

	// Build prompt pieces with an emoji-safe custom box
	generalColors := a.GetComponentColors("general")
	dog := newEmojiBox("🐶>", generalColors.Text.Color(), generalColors.Background.Color())

	input := tview.NewInputField()
	input.SetFieldWidth(0)
	input.SetPlaceholder("")
	input.SetBorder(false)
	input.SetBackgroundColor(generalColors.Background.Color())
	a.ConfigureInputFieldTheme(input, "command")
	input.SetText(prefix)
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
	hint.SetBackgroundColor(generalColors.Background.Color())
	hint.SetText("")
	hint.SetTextColor(a.getHintColor())

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
	row.SetBackgroundColor(generalColors.Background.Color())

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
		cmdBar.SetBorderColor(a.GetStatusColor("info"))
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
		"/":              {"/"},
		"l":              {"labels", "links", "list"},
		"la":             {"labels"},
		"lab":            {"labels"},
		"labe":           {"labels"},
		"label":          {"labels"},
		"labels":         {"labels"},
		"li":             {"links"},
		"lin":            {"links"},
		"link":           {"links"},
		"links":          {"links"},
		"at":             {"attachments"},
		"att":            {"attachments"},
		"atta":           {"attachments"},
		"attac":          {"attachments"},
		"attach":         {"attachments"},
		"attachm":        {"attachments"},
		"attachme":       {"attachments"},
		"attachmen":      {"attachments"},
		"attachment":     {"attachments"},
		"attachments":    {"attachments"},
		"s":              {"search", "slack"},
		"sl":             {"slack"},
		"sla":            {"slack"},
		"slac":           {"slack"},
		"slack":          {"slack"},
		"se":             {"search"},
		"sea":            {"search"},
		"sear":           {"search"},
		"searc":          {"search"},
		"search":         {"search"},
		"i":              {"inbox"},
		"in":             {"inbox"},
		"inb":            {"inbox"},
		"inbo":           {"inbox"},
		"inbox":          {"inbox"},
		"c":              {"compose"},
		"co":             {"compose"},
		"com":            {"compose"},
		"comp":           {"compose"},
		"compo":          {"compose"},
		"compos":         {"compose"},
		"compose":        {"compose"},
		"r":              {"reply"},
		"re":             {"reply"},
		"rep":            {"reply"},
		"repl":           {"reply"},
		"reply":          {"reply"},
		"ra":             {"reply-all"},
		"reply-":         {"reply-all"},
		"reply-a":        {"reply-all"},
		"reply-al":       {"reply-all"},
		"reply-all":      {"reply-all"},
		"f":              {"forward"},
		"fo":             {"forward"},
		"for":            {"forward"},
		"forw":           {"forward"},
		"forwa":          {"forward"},
		"forwar":         {"forward"},
		"forward":        {"forward"},
		"dr":             {"drafts"},
		"dra":            {"drafts"},
		"draf":           {"drafts"},
		"draft":          {"drafts"},
		"drafts":         {"drafts"},
		"h":              {"help"},
		"he":             {"help", "headers"},
		"hea":            {"headers"},
		"head":           {"headers"},
		"heade":          {"headers"},
		"header":         {"headers"},
		"headers":        {"headers"},
		"hel":            {"help"},
		"help":           {"help"},
		"nu":             {"numbers"},
		"num":            {"numbers"},
		"numb":           {"numbers"},
		"numbe":          {"numbers"},
		"number":         {"numbers"},
		"numbers":        {"numbers"},
		"pre":            {"preload"},
		"prel":           {"preload"},
		"prelo":          {"preload"},
		"preloa":         {"preload"},
		"preload":        {"preload"},
		"pl":             {"preload"},
		"q":              {"quit"},
		"qu":             {"quit"},
		"qui":            {"quit"},
		"quit":           {"quit"},
		"g":              {"G"},
		"G":              {"G"},
		"1":              {"1"},
		"$":              {"$"},
		"5":              {"5"},
		"10":             {"10"},
		"st":             {"stats"},
		"sta":            {"stats"},
		"stat":           {"stats"},
		"stats":          {"stats"},
		"u":              {"usage"},
		"us":             {"usage"},
		"usa":            {"usage"},
		"usag":           {"usage"},
		"usage":          {"usage"},
		"sel":            {"select"},
		"sele":           {"select"},
		"selec":          {"select"},
		"select":         {"select"},
		"m":              {"move", "more"},
		"mov":            {"move"},
		"move":           {"move"},
		"mv":             {"move"},
		"lbl":            {"label"},
		"lo":             {"load"},
		"loa":            {"load"},
		"load":           {"load"},
		"mor":            {"more"},
		"more":           {"more"},
		"n":              {"next", "numbers"},
		"ne":             {"next"},
		"nex":            {"next"},
		"next":           {"next"},
		"o":              {"obsidian"},
		"ob":             {"obsidian"},
		"obs":            {"obsidian"},
		"obsi":           {"obsidian"},
		"obsid":          {"obsidian"},
		"obsidi":         {"obsidian"},
		"obsidian":       {"obsidian"},
		"p":              {"prompt"},
		"pr":             {"prompt"},
		"pro":            {"prompt"},
		"prom":           {"prompt"},
		"promp":          {"prompt"},
		"prompt":         {"prompt"},
		"a":              {"archive"},
		"ar":             {"archive"},
		"arc":            {"archive"},
		"arch":           {"archive"},
		"archi":          {"archive"},
		"archiv":         {"archive"},
		"archive":        {"archive"},
		"archived":       {"archived"},
		"b":              {"archived"},
		"unr":            {"unread"},
		"unre":           {"unread"},
		"unrea":          {"unread"},
		"unread":         {"unread"},
		"un":             {"undo", "unread"},
		"und":            {"undo"},
		"undo":           {"undo"},
		"U":              {"undo"},
		"d":              {"trash"},
		"tr":             {"trash"},
		"tra":            {"trash"},
		"tras":           {"trash"},
		"trash":          {"trash"},
		"t":              {"read"},
		"rea":            {"read"},
		"read":           {"read"},
		"toggle":         {"read"},
		"toggle-":        {"read", "headers"},
		"toggle-h":       {"headers"},
		"toggle-he":      {"headers"},
		"toggle-hea":     {"headers"},
		"toggle-head":    {"headers"},
		"toggle-heade":   {"headers"},
		"toggle-header":  {"headers"},
		"toggle-headers": {"headers"},
		"toggle-r":       {"read"},
		"toggle-re":      {"read"},
		"toggle-rea":     {"read"},
		"toggle-read":    {"read"},
		"sa":             {"save-query"},
		"sav":            {"save-query"},
		"save":           {"save-query"},
		"save-":          {"save-query"},
		"save-q":         {"save-query"},
		"save-qu":        {"save-query"},
		"save-que":       {"save-query"},
		"save-quer":      {"save-query"},
		"save-query":     {"save-query"},
		"sq":             {"save-query"},
		"bo":             {"bookmarks"},
		"boo":            {"bookmarks"},
		"book":           {"bookmarks"},
		"bookm":          {"bookmarks"},
		"bookma":         {"bookmarks"},
		"bookmar":        {"bookmarks"},
		"bookmark":       {"bookmark", "bookmarks"},
		"bookmarks":      {"bookmarks"},
		"bm":             {"bookmarks"},
		"que":            {"queries", "query"},
		"quer":           {"queries", "query"},
		"queri":          {"queries"},
		"querie":         {"queries"},
		"queries":        {"queries"},
		"query":          {"query"},
		"qb":             {"bookmarks"},
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

	// Contextual arguments for 'bookmark' commands - complete query names
	if strings.HasPrefix(buffer, "bookmark ") || strings.HasPrefix(buffer, "query ") {
		// Get the partial query name
		prefix := "bookmark "
		if strings.HasPrefix(buffer, "query ") {
			prefix = "query "
		}

		partialName := strings.TrimSpace(strings.TrimPrefix(buffer, prefix))

		// Get query service and list saved queries
		queryService := a.GetQueryService()
		if queryService != nil {
			// Set account email if available
			if queryServiceImpl, ok := queryService.(*services.QueryServiceImpl); ok {
				if email := a.getActiveAccountEmail(); email != "" {
					queryServiceImpl.SetAccountEmail(email)
				}
			}

			// Get saved queries
			queries, err := queryService.ListQueries(a.ctx, "")
			if err == nil && len(queries) > 0 {
				// Find first query name that starts with the partial name (case-insensitive)
				partialLower := strings.ToLower(partialName)
				for _, query := range queries {
					queryNameLower := strings.ToLower(query.Name)
					if strings.HasPrefix(queryNameLower, partialLower) {
						return prefix + query.Name
					}
				}
			}
		}
	}

	// Contextual help for G command
	if strings.HasPrefix(buffer, "G ") {
		return "G <message_number>"
	}

	// Contextual suggestions for 'prompt' commands
	if strings.HasPrefix(buffer, "prompt ") {
		tail := strings.TrimSpace(strings.TrimPrefix(buffer, "prompt"))
		lower := strings.ToLower(tail)
		switch {
		case strings.HasPrefix("stats", lower):
			return "prompt stats"
		case strings.HasPrefix("statistics", lower):
			return "prompt statistics"
		case strings.HasPrefix("list", lower):
			return "prompt list"
		case strings.HasPrefix("create", lower):
			return "prompt create"
		case strings.HasPrefix("update", lower):
			return "prompt update"
		case strings.HasPrefix("delete", lower):
			return "prompt delete"
		case strings.HasPrefix("export", lower):
			return "prompt export"
		case lower == "s":
			return "prompt stats"
		case lower == "l":
			return "prompt list"
		case lower == "c":
			return "prompt create"
		case lower == "u":
			return "prompt update"
		case lower == "d":
			return "prompt delete"
		case lower == "e":
			return "prompt export"
		}
	}

	// Contextual suggestions for 'obsidian' commands
	if strings.HasPrefix(buffer, "obsidian ") || strings.HasPrefix(buffer, "obs ") {
		prefix := "obsidian "
		if strings.HasPrefix(buffer, "obs ") {
			prefix = "obs "
		}

		tail := strings.TrimSpace(strings.TrimPrefix(buffer, strings.TrimSpace(prefix)))
		lower := strings.ToLower(tail)

		switch {
		case strings.HasPrefix("repack", lower):
			return prefix + "repack"
		case strings.HasPrefix("repopack", lower):
			return prefix + "repack" // Normalize to repack
		case lower == "r":
			return prefix + "repack"
		case tail == "":
			return prefix + "repack" // Default suggestion for obsidian subcommands
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

	// Special handling for content search commands that start with "/"
	if strings.HasPrefix(command, "/") && len(command) > 1 {
		// Extract search term from "/term" -> "term"
		searchTerm := command[1:]
		// Also include any additional args: "/term more words"
		allArgs := append([]string{searchTerm}, args...)
		a.executeContentSearch(allArgs)
		return
	}

	switch command {
	case "labels", "l":
		a.executeLabelsCommand(args)
	case "links", "link":
		a.executeLinksCommand(args)
	case "attachments", "attach":
		a.executeAttachmentsCommand(args)
	case "gmail", "web", "open-web", "o":
		a.executeGmailWebCommand(args)
	case "/":
		a.executeContentSearch(args)
	case "search":
		a.executeSearchCommand(args)
	case "slack", "sl":
		a.executeSlackCommand(args)
	case "s":
		// Handle ambiguous "s" - prioritize search if has args, slack if no args
		if len(args) > 0 {
			a.executeSearchCommand(args)
		} else {
			a.executeSlackCommand(args)
		}
	case "summary":
		a.executeSummaryCommand(args)
	case "rsvp":
		a.executeRSVPCommand(args)
	case "inbox", "i":
		a.executeInboxCommand(args)
	case "compose", "c":
		a.executeComposeCommand(args)
	case "headers", "toggle-headers":
		a.executeToggleHeadersCommand(args)

	// Threading commands
	case "threads", "thr":
		a.executeThreadsCommand(args)
	case "flatten", "flat":
		a.executeFlattenCommand(args)
	case "thread-summary", "th-sum":
		a.executeThreadSummaryCommand(args)
	case "expand-all", "expand":
		a.executeExpandAllCommand(args)
	case "collapse-all", "collapse":
		a.executeCollapseAllCommand(args)

	case "help", "h", "?":
		a.executeHelpCommand(args)
	case "numbers", "n":
		a.executeNumbersCommand(args)
	case "quit", "q":
		a.executeQuitCommand(args)
	case "cache":
		a.executeCacheCommand(args)
	case "preload", "pl":
		a.executePreloadCommand(args)
	case "stats", "usage":
		a.executeStatsCommand(args)
	case "g", "G":
		a.executeGoToCommand(args)
	case "archive", "a":
		a.executeArchiveCommand(args)
	case "trash", "d":
		a.executeTrashCommand(args)
	case "read", "toggle-read", "t":
		a.executeToggleReadCommand(args)
	case "new":
		a.executeComposeCommand(args) // "new" as alias for compose
	case "reply", "r":
		a.executeReplyCommand(args)
	case "reply-all", "ra":
		a.executeReplyAllCommand(args)
	case "forward", "f":
		a.executeForwardCommand(args)
	case "drafts", "dr":
		a.executeDraftsCommand(args)
	case "refresh":
		a.executeRefreshCommand(args)
	case "load", "more", "next":
		a.executeLoadMoreCommand(args)
	case "unread", "u":
		a.executeUnreadCommand(args)
	case "undo", "U":
		a.executeUndoCommand(args)
	case "archived", "arch-search", "b":
		a.executeArchivedCommand(args)
	case "select", "sel":
		a.executeSelectCommand(args)
	case "move", "mv":
		a.executeMoveCommand(args)
	case "label", "lbl":
		a.executeLabelCommand(args)
	case "obsidian", "obs":
		a.executeObsidianCommand(args)
	case "prompt", "pr", "p":
		a.executePromptCommand(args)
	case "theme", "th":
		if len(args) == 0 {
			// Open theme picker UI if no subcommands
			go a.openThemePicker()
		} else {
			a.executeThemeCommand(args)
		}
	case "save-query", "save", "sq":
		a.executeSaveQueryCommand(args)
	case "bookmarks", "queries", "bm", "qb":
		a.executeBookmarksCommand(args)
	case "bookmark", "query":
		a.executeBookmarkCommand(args)
	default:
		// Check for numeric shortcuts like :1, :$
		if matched := a.executeNumericShortcut(command); !matched {
			a.showError(fmt.Sprintf("Unknown command: %s", command))
		}
	}
}

// executeSlackCommand handles :slack commands
func (a *App) executeSlackCommand(args []string) {
	// Check if Slack is enabled
	if !a.Config.Slack.Enabled {
		a.showError("Slack integration is not enabled in configuration")
		return
	}

	var messageID string

	// Handle optional message number argument
	if len(args) > 0 {
		// Parse message number (1-based like :5 command)
		if num, err := strconv.Atoi(args[0]); err == nil && num >= 1 {
			// Check if we have messages loaded
			if len(a.ids) == 0 {
				a.showError("No messages loaded")
				return
			}

			// Convert 1-based user input to 0-based array index
			maxMessage := len(a.ids)
			if num > maxMessage {
				a.showError(fmt.Sprintf("Message %d not found (only %d messages loaded)", num, maxMessage))
				return
			}

			// Get message index from the specified position
			messageIndex := num - 1 // Convert to 0-based index
			// messageID = a.ids[messageIndex] // Removed ineffectual assignment - showSlackForwardDialog gets current message internally

			// Also select the message in the UI for consistency
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.Select(messageIndex, 0)
			}
		} else {
			a.showError(fmt.Sprintf("Invalid message number: %s", args[0]))
			return
		}
	} else {
		// Use cached message ID (for undo functionality) with sync fallback
		messageID = a.GetCurrentMessageID()

		// Ensure cache is synchronized with cursor position
		if a.logger != nil {
			cursorID := a.getCurrentSelectedMessageID()
			// If they don't match, sync the cached state
			if messageID != cursorID && cursorID != "" {
				messageID = cursorID
				a.SetCurrentMessageID(messageID)
			}
		}

		if messageID == "" {
			a.showError("No message selected")
			return
		}
	}

	// Show the Slack forwarding panel
	// Ensure proper focus management like keyboard shortcut
	go a.showSlackForwardDialog()
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

// executeLinksCommand handles links commands
func (a *App) executeLinksCommand(args []string) {
	// Simple command - just open the link picker
	go a.openLinkPicker()
}

// executeAttachmentsCommand handles attachment commands
func (a *App) executeAttachmentsCommand(args []string) {
	// Simple command - just open the attachment picker
	go a.openAttachmentPicker()
}

// executeGmailWebCommand handles opening Gmail in web interface
func (a *App) executeGmailWebCommand(args []string) {
	go a.openEmailInGmail()
}

// executeContentSearch handles content search within message text
func (a *App) executeContentSearch(args []string) {
	if a.enhancedTextView == nil {
		a.showError("Content search not available")
		return
	}

	query := strings.Join(args, " ")
	if query == "" {
		a.showError("Usage: /<term> for content search")
		return
	}

	// Use the enhanced text view's search functionality
	a.enhancedTextView.performContentSearch(query)

	// CRITICAL: Set a flag to prevent restoreFocusAfterModal from overriding our focus
	// We'll set focus to EnhancedTextView immediately after command execution
	a.cmdFocusOverride = "enhanced-text"

	// DEBUGGING: Also try direct focus setting as backup
	go func() {
		// Wait for command bar to close, then force focus to EnhancedTextView
		for i := 0; i < 10; i++ { // Try up to 10 times over 1 second
			time.Sleep(100 * time.Millisecond)
			if !a.cmdMode { // Command bar closed
				a.QueueUpdateDraw(func() {
					if a.enhancedTextView != nil {
						a.SetFocus(a.enhancedTextView)
						a.currentFocus = "text"
						a.updateFocusIndicators("text")
					}
				})
				break
			}
		}
	}()
}

// executeSearchCommand handles email search commands
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

// executeToggleHeadersCommand handles header toggle commands
func (a *App) executeToggleHeadersCommand(args []string) {
	a.toggleHeaderVisibility()
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
	// Last message is at table row = len(a.ids) (accounting for header at row 0)
	targetRow := len(a.ids)

	// If argument provided (called from :5 style commands), calculate target row
	if len(args) > 0 {
		if num, err := strconv.Atoi(args[0]); err == nil && num >= 1 {
			// Convert 1-based user input to table row (accounting for header row)
			// User message 1 = table row 1, message 2 = table row 2, etc.
			maxMessage := len(a.ids) // Total number of messages
			if num > maxMessage {
				targetRow = len(a.ids) // Go to last message if number too high
			} else {
				targetRow = num // User message N = table row N (header is row 0)
			}
		}
	}

	// Use the same simple approach as the working direct G key
	list.Select(targetRow, 0)

	// Manually trigger preloading since SetSelectionChangedFunc may not fire for programmatic selection
	a.triggerPreloadingForMessage(targetRow - 1) // Convert table row to message index
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

	// First message is at table row 1 (row 0 is header, maps to a.ids[0])
	// This matches the SetSelectionChangedFunc logic: messageIndex = row - 1
	list.Select(1, 0)

	// Manually trigger preloading since SetSelectionChangedFunc may not fire for programmatic selection
	a.triggerPreloadingForMessage(0) // First message is at index 0
}

// executeCacheCommand handles cache-related commands
func (a *App) executeCacheCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: cache <clear|info>")
		return
	}

	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "clear", "clean":
		a.executeCacheClear(args[1:])
	case "info", "status":
		a.executeCacheInfo(args[1:])
	default:
		a.showError(fmt.Sprintf("Unknown cache subcommand: %s. Usage: cache <clear|info>", subcommand))
	}
}

// executePreloadCommand handles preloading-related commands
func (a *App) executePreloadCommand(args []string) {
	preloader := a.GetPreloaderService()
	if preloader == nil {
		a.showError("Preloader service not available")
		return
	}

	if len(args) == 0 {
		// Show current status when no args
		a.executePreloadStatus(args)
		return
	}

	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "on", "enable":
		a.executePreloadEnable(args[1:])
	case "off", "disable":
		a.executePreloadDisable(args[1:])
	case "status", "info":
		a.executePreloadStatus(args[1:])
	case "clear", "clean":
		a.executePreloadClear(args[1:])
	case "next", "nextpage":
		a.executePreloadNext(args[1:])
	case "adjacent", "adj":
		a.executePreloadAdjacent(args[1:])
	default:
		a.showError(fmt.Sprintf("Unknown preload subcommand: %s. Usage: preload|pl <on|off|status|clear|next|adjacent>", subcommand))
	}
}

// executePreloadStatus shows preloader status
func (a *App) executePreloadStatus(args []string) {
	preloader := a.GetPreloaderService()
	if preloader == nil {
		a.showError("Preloader service not available")
		return
	}

	status := preloader.GetStatus()
	if status == nil {
		a.showError("Could not get preloader status")
		return
	}

	// Build status message
	var statusMsg strings.Builder
	statusMsg.WriteString("📦 Preloader Status:\n\n")
	statusMsg.WriteString(fmt.Sprintf("Enabled: %v\n", status.Enabled))
	statusMsg.WriteString(fmt.Sprintf("Next Page: %v\n", status.NextPageEnabled))
	statusMsg.WriteString(fmt.Sprintf("Adjacent Messages: %v\n", status.AdjacentEnabled))
	statusMsg.WriteString(fmt.Sprintf("Cache Size: %d messages (%.1f MB)\n", status.CacheSize, status.CacheMemoryUsageMB))
	statusMsg.WriteString(fmt.Sprintf("Active Tasks: %d\n", status.ActivePreloadTasks))
	statusMsg.WriteString(fmt.Sprintf("Background Workers: %d\n", status.BackgroundWorkers))

	if status.Statistics != nil {
		hitRate := status.Statistics.CacheHitRate * 100
		statusMsg.WriteString("\nStatistics:\n")
		statusMsg.WriteString(fmt.Sprintf("  Cache Hit Rate: %.1f%%\n", hitRate))
		statusMsg.WriteString(fmt.Sprintf("  Cache Hits: %d\n", status.PreloadHits))
		statusMsg.WriteString(fmt.Sprintf("  Cache Misses: %d\n", status.PreloadMisses))
		statusMsg.WriteString(fmt.Sprintf("  Next Page Requests: %d\n", status.Statistics.NextPageRequests))
		statusMsg.WriteString(fmt.Sprintf("  Adjacent Requests: %d\n", status.Statistics.AdjacentRequests))
		statusMsg.WriteString(fmt.Sprintf("  Data Preloaded: %.1f MB\n", status.Statistics.TotalDataPreloadedMB))
		if status.Statistics.AveragePreloadTime > 0 {
			statusMsg.WriteString(fmt.Sprintf("  Avg Preload Time: %v\n", status.Statistics.AveragePreloadTime))
		}
	}

	// Add usage information
	statusMsg.WriteString("\nUsage:\n")
	statusMsg.WriteString("  :preload or :pl       - Show this status\n")
	statusMsg.WriteString("  :preload status       - Show this status\n")
	statusMsg.WriteString("  :preload on|off       - Enable/disable all preloading\n")
	statusMsg.WriteString("  :preload next on|off  - Enable/disable next page preloading\n")
	statusMsg.WriteString("  :preload adj on|off   - Enable/disable adjacent message preloading\n")
	statusMsg.WriteString("  :preload clear        - Clear preload cache\n")
	statusMsg.WriteString("\nPress ESC to return to previous view\n")

	// Call showPreloadStatus in goroutine to avoid command context issues
	statusContent := statusMsg.String()
	go func() {
		a.showPreloadStatus(statusContent)
	}()
}

// executePreloadEnable enables preloading features
func (a *App) executePreloadEnable(args []string) {
	preloader := a.GetPreloaderService()
	if preloader == nil {
		a.showError("Preloader service not available")
		return
	}

	if len(args) == 0 {
		// Enable all preloading
		config := preloader.GetStatus().Config
		config.Enabled = true
		config.NextPageEnabled = true
		config.AdjacentEnabled = true

		if err := preloader.UpdateConfig(config); err != nil {
			a.showError(fmt.Sprintf("Failed to enable preloading: %v", err))
			return
		}
		a.showSuccess("Preloading enabled (all features)")
		return
	}

	feature := strings.ToLower(args[0])
	config := preloader.GetStatus().Config

	switch feature {
	case "next", "nextpage":
		config.NextPageEnabled = true
		a.showSuccess("Next page preloading enabled")
	case "adjacent", "adj":
		config.AdjacentEnabled = true
		a.showSuccess("Adjacent message preloading enabled")
	default:
		a.showError(fmt.Sprintf("Unknown preload feature: %s. Use: next, adjacent", feature))
		return
	}

	if err := preloader.UpdateConfig(config); err != nil {
		a.showError(fmt.Sprintf("Failed to update preloader config: %v", err))
	}
}

// executePreloadDisable disables preloading features
func (a *App) executePreloadDisable(args []string) {
	preloader := a.GetPreloaderService()
	if preloader == nil {
		a.showError("Preloader service not available")
		return
	}

	if len(args) == 0 {
		// Disable all preloading
		config := preloader.GetStatus().Config
		config.Enabled = false
		config.NextPageEnabled = false
		config.AdjacentEnabled = false

		if err := preloader.UpdateConfig(config); err != nil {
			a.showError(fmt.Sprintf("Failed to disable preloading: %v", err))
			return
		}
		a.showSuccess("Preloading disabled (all features)")
		return
	}

	feature := strings.ToLower(args[0])
	config := preloader.GetStatus().Config

	switch feature {
	case "next", "nextpage":
		config.NextPageEnabled = false
		a.showSuccess("Next page preloading disabled")
	case "adjacent", "adj":
		config.AdjacentEnabled = false
		a.showSuccess("Adjacent message preloading disabled")
	default:
		a.showError(fmt.Sprintf("Unknown preload feature: %s. Use: next, adjacent", feature))
		return
	}

	if err := preloader.UpdateConfig(config); err != nil {
		a.showError(fmt.Sprintf("Failed to update preloader config: %v", err))
	}
}

// executePreloadClear clears the preload cache
func (a *App) executePreloadClear(args []string) {
	preloader := a.GetPreloaderService()
	if preloader == nil {
		a.showError("Preloader service not available")
		return
	}

	if err := preloader.ClearCache(a.ctx); err != nil {
		a.showError(fmt.Sprintf("Failed to clear preload cache: %v", err))
		return
	}

	a.showSuccess("Preload cache cleared")
}

// executePreloadNext controls next page preloading
func (a *App) executePreloadNext(args []string) {
	preloader := a.GetPreloaderService()
	if preloader == nil {
		a.showError("Preloader service not available")
		return
	}

	if len(args) == 0 {
		// Show current status
		enabled := preloader.IsNextPageEnabled()
		a.showInfo(fmt.Sprintf("Next page preloading: %v", enabled))
		return
	}

	action := strings.ToLower(args[0])
	config := preloader.GetStatus().Config

	switch action {
	case "on", "enable":
		config.NextPageEnabled = true
		a.showSuccess("Next page preloading enabled")
	case "off", "disable":
		config.NextPageEnabled = false
		a.showSuccess("Next page preloading disabled")
	default:
		a.showError(fmt.Sprintf("Unknown action: %s. Use: on, off", action))
		return
	}

	if err := preloader.UpdateConfig(config); err != nil {
		a.showError(fmt.Sprintf("Failed to update config: %v", err))
	}
}

// executePreloadAdjacent controls adjacent message preloading
func (a *App) executePreloadAdjacent(args []string) {
	preloader := a.GetPreloaderService()
	if preloader == nil {
		a.showError("Preloader service not available")
		return
	}

	if len(args) == 0 {
		// Show current status
		enabled := preloader.IsAdjacentEnabled()
		a.showInfo(fmt.Sprintf("Adjacent message preloading: %v", enabled))
		return
	}

	action := strings.ToLower(args[0])
	config := preloader.GetStatus().Config

	switch action {
	case "on", "enable":
		config.AdjacentEnabled = true
		a.showSuccess("Adjacent message preloading enabled")
	case "off", "disable":
		config.AdjacentEnabled = false
		a.showSuccess("Adjacent message preloading disabled")
	default:
		a.showError(fmt.Sprintf("Unknown action: %s. Use: on, off", action))
		return
	}

	if err := preloader.UpdateConfig(config); err != nil {
		a.showError(fmt.Sprintf("Failed to update config: %v", err))
	}
}

// executeStatsCommand handles the stats/usage command (redirects to new :prompt stats)
func (a *App) executeStatsCommand(args []string) {
	// Redirect to new command structure
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, "Command moved! Use ':prompt stats' instead of ':stats'")
	}()

	// Execute the new prompt stats command directly
	a.executePromptStats([]string{})
}

// executeCacheClear clears prompt caches
func (a *App) executeCacheClear(args []string) {
	// Get services
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.showError("Prompt service not available")
		return
	}

	accountEmail := a.getActiveAccountEmail()

	go func() {
		if len(args) > 0 && strings.ToLower(args[0]) == "all" {
			// Clear all caches for all accounts (admin function)
			if err := promptService.ClearAllPromptCaches(a.ctx); err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to clear all caches: %v", err))
				return
			}
			a.GetErrorHandler().ShowSuccess(a.ctx, "All prompt caches cleared successfully")
		} else {
			// Clear caches for current account
			if err := promptService.ClearPromptCache(a.ctx, accountEmail); err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to clear cache: %v", err))
				return
			}
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Prompt cache cleared for %s", accountEmail))
		}
	}()
}

// executeCacheInfo shows cache information
func (a *App) executeCacheInfo(args []string) {
	accountEmail := a.getActiveAccountEmail()

	go func() {
		// Get services to check if database is available
		_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
		if promptService == nil {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
			return
		}

		// Create safe filename
		safeEmail := strings.ToLower(strings.ReplaceAll(accountEmail, "@", "_"))

		// Show basic cache info with simple message
		infoMsg := fmt.Sprintf("Cache info: %s | DB: %s.sqlite3", accountEmail, safeEmail)
		a.GetErrorHandler().ShowInfo(a.ctx, infoMsg)
	}()
}

// executeNumbersCommand handles :numbers/:n commands (toggle message number display)
func (a *App) executeNumbersCommand(args []string) {
	// Toggle the display of message numbers
	a.showMessageNumbers = !a.showMessageNumbers

	// Trigger UI redraw in a goroutine to avoid hanging
	go func() {
		a.refreshTableDisplay()
		if a.showMessageNumbers {
			a.GetErrorHandler().ShowInfo(a.ctx, "Message numbers enabled")
		} else {
			a.GetErrorHandler().ShowInfo(a.ctx, "Message numbers disabled")
		}
	}()
}

// executeArchiveCommand handles :archive/:a commands
func (a *App) executeArchiveCommand(args []string) {
	// Check if count argument provided for range operation
	if len(args) > 0 {
		count, err := strconv.Atoi(args[0])
		if err != nil || count <= 0 {
			a.showError("Usage: archive [count] (positive number)")
			return
		}

		// Get current position
		startIndex := a.getCurrentSelectedMessageIndex()
		if startIndex < 0 {
			a.showError("No message selected")
			return
		}

		a.archiveRange(startIndex, count)
		return
	}

	// Check if we're in bulk mode with selections
	if a.bulkMode && len(a.selected) > 0 {
		go a.archiveSelectedBulk()
	} else {
		go a.archiveSelected()
	}
}

// executeTrashCommand handles :trash/:d commands
func (a *App) executeTrashCommand(args []string) {
	// Check if count argument provided for range operation
	if len(args) > 0 {
		count, err := strconv.Atoi(args[0])
		if err != nil || count <= 0 {
			a.showError("Usage: trash [count] (positive number)")
			return
		}

		// Get current position
		startIndex := a.getCurrentSelectedMessageIndex()
		if startIndex < 0 {
			a.showError("No message selected")
			return
		}

		a.trashRange(startIndex, count)
		return
	}

	// Check if we're in bulk mode with selections
	if a.bulkMode && len(a.selected) > 0 {
		go a.trashSelectedBulk()
	} else {
		go a.trashSelected()
	}
}

// executeToggleReadCommand handles :read/:toggle-read/:t commands
func (a *App) executeToggleReadCommand(args []string) {
	// Check if count argument provided for range operation
	if len(args) > 0 {
		count, err := strconv.Atoi(args[0])
		if err != nil || count <= 0 {
			a.showError("Usage: read [count] (positive number)")
			return
		}

		// Get current position
		startIndex := a.getCurrentSelectedMessageIndex()
		if startIndex < 0 {
			a.showError("No message selected")
			return
		}

		a.toggleReadRange(startIndex, count)
		return
	}

	// Check if we're in bulk mode with selections
	if a.bulkMode && len(a.selected) > 0 {
		go a.toggleMarkReadUnreadBulk()
	} else {
		go a.toggleMarkReadUnread()
	}
}

// executeReplyCommand handles :reply/:r commands
func (a *App) executeReplyCommand(args []string) {
	go a.replySelected()
}

// executeRefreshCommand handles :refresh commands
func (a *App) executeRefreshCommand(args []string) {
	if a.draftMode {
		go a.loadDrafts()
	} else {
		go a.reloadMessages()
	}
}

// executeLoadMoreCommand handles :load/:more/:next commands
func (a *App) executeLoadMoreCommand(args []string) {
	// Only load more when focused on list
	if a.currentFocus == "list" {
		go a.loadMoreMessages()
	} else {
		a.GetErrorHandler().ShowWarning(a.ctx, "Load more only available when message list is focused")
	}
}

// executeUnreadCommand handles :unread/:u commands
func (a *App) executeUnreadCommand(args []string) {
	go a.listUnreadMessages()
}

// executeUndoCommand handles :undo command
func (a *App) executeUndoCommand(args []string) {
	go a.performUndo()
}

// executeArchivedCommand handles :archived/:arch-search commands
func (a *App) executeArchivedCommand(args []string) {
	go a.listArchivedMessages()
}

// executeSelectCommand handles :select commands for range selection
func (a *App) executeSelectCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: select <count>")
		return
	}

	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		a.showError("Usage: select <count> (positive number)")
		return
	}

	// Get current position
	startIndex := a.getCurrentSelectedMessageIndex()
	if startIndex < 0 {
		a.showError("No message selected")
		return
	}

	a.selectRange(startIndex, count)
}

// executeMoveCommand handles :move commands for range move operations
func (a *App) executeMoveCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: move <count>")
		return
	}

	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		a.showError("Usage: move <count> (positive number)")
		return
	}

	// Get current position
	startIndex := a.getCurrentSelectedMessageIndex()
	if startIndex < 0 {
		a.showError("No message selected")
		return
	}

	a.moveRange(startIndex, count)
}

// executeLabelCommand handles :label commands for range labeling operations
func (a *App) executeLabelCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: label <count>")
		return
	}

	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		a.showError("Usage: label <count> (positive number)")
		return
	}

	// Get current position
	startIndex := a.getCurrentSelectedMessageIndex()
	if startIndex < 0 {
		a.showError("No message selected")
		return
	}

	a.labelRange(startIndex, count)
}

// executeObsidianCommand handles :obsidian commands for Obsidian operations
func (a *App) executeObsidianCommand(args []string) {
	if len(args) == 0 {
		a.showError("Usage: obsidian <count> | obsidian repack")
		return
	}

	// Check for repack subcommand
	if strings.ToLower(args[0]) == "repack" || strings.ToLower(args[0]) == "repopack" {
		// Handle repack mode
		if a.bulkMode && len(a.selected) > 0 {
			// Open bulk Obsidian picker with focus on repack mode
			go a.openBulkObsidianPanelWithRepack()
		} else {
			messageID := a.GetCurrentMessageID()
			if messageID == "" {
				a.showError("No message selected for Obsidian repack")
				return
			}
			// For single message, just open normal Obsidian panel
			// (repack mode doesn't make sense for single message)
			message, err := a.Client.GetMessageWithContent(messageID)
			if err != nil {
				a.showError("Failed to load message content")
				return
			}
			go a.openObsidianIngestPanel(message)
		}
		return
	}

	// Handle range operations (existing functionality)
	count, err := strconv.Atoi(args[0])
	if err != nil || count <= 0 {
		a.showError("Usage: obsidian <count> | obsidian repack")
		return
	}

	// Get current position
	startIndex := a.getCurrentSelectedMessageIndex()
	if startIndex < 0 {
		a.showError("No message selected")
		return
	}

	a.obsidianRange(startIndex, count)
}

// executePromptCommand handles :prompt commands for prompt template management
func (a *App) executePromptCommand(args []string) {
	if len(args) == 0 {
		// Default to list command
		a.openPromptManager()
		return
	}

	subCommand := strings.ToLower(args[0])
	subArgs := args[1:]

	switch subCommand {
	case "list", "l":
		a.openPromptManager()
	case "create", "c":
		a.executePromptCreate(subArgs)
	case "update", "u":
		a.executePromptUpdate(subArgs)
	case "export", "e":
		a.executePromptExport(subArgs)
	case "delete", "d":
		a.executePromptDelete(subArgs)
	case "stats", "statistics", "s":
		a.executePromptStats(subArgs)
	default:
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Unknown prompt command: %s. Use 'list', 'create', 'update', 'export', 'delete', or 'stats'", subCommand))
		}()
	}
}

// openPromptManager opens the enhanced prompt picker for management
func (a *App) openPromptManager() {
	go a.openPromptPickerForManagement()
}

// executePromptCreate creates a prompt from a markdown file
func (a *App) executePromptCreate(args []string) {
	if len(args) == 0 {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Usage: prompt create <file_path>")
		}()
		return
	}

	filePath := args[0]

	// Get services
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		}()
		return
	}

	go func() {
		// Add timeout protection for file operations
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()

		id, err := promptService.CreateFromFile(ctx, filePath)
		if err != nil {
			// Use separate goroutine for ErrorHandler to avoid potential deadlocks
			go func() {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to create prompt: %v", err))
			}()
			return
		}

		// Use separate goroutine for ErrorHandler to avoid potential deadlocks
		go func() {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Created prompt with ID %d", id))
		}()
	}()
}

// executePromptUpdate updates an existing prompt from a markdown file
func (a *App) executePromptUpdate(args []string) {
	if len(args) < 2 {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Usage: prompt update <id|name> <file_path>")
		}()
		return
	}

	identifier := args[0]
	filePath := args[1]

	// Get services
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		}()
		return
	}

	go func() {
		var promptID int
		var promptName string

		// Try to parse as ID first
		if id, parseErr := strconv.Atoi(identifier); parseErr == nil {
			promptID = id
			// Get prompt to show name in confirmation
			if prompt, err := promptService.GetPrompt(a.ctx, id); err == nil {
				promptName = prompt.Name
			} else {
				promptName = fmt.Sprintf("ID %d", id)
			}
		} else {
			// Try to find by name
			prompt, findErr := promptService.FindPromptByName(a.ctx, identifier)
			if findErr != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Prompt not found: %s", identifier))
				return
			}
			promptID = prompt.ID
			promptName = prompt.Name
		}

		// Read and parse the new file content
		// Expand tilde in path
		if strings.HasPrefix(filePath, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Cannot get home directory: %v", err))
				return
			}
			if filePath == "~" {
				filePath = home
			} else {
				filePath = filepath.Join(home, filePath[2:])
			}
		}

		// Validate path to prevent directory traversal
		cleanPath := filepath.Clean(filePath)
		if strings.Contains(cleanPath, "..") {
			a.GetErrorHandler().ShowError(a.ctx, "Invalid file path: contains directory traversal")
			return
		}

		// Read file content
		content, err := os.ReadFile(cleanPath)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to read file %s: %v", filePath, err))
			return
		}

		// Parse front matter manually (same logic as in service)
		text := string(content)
		if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
			a.GetErrorHandler().ShowError(a.ctx, "File must start with YAML front matter (---)")
			return
		}

		// Find the end of front matter
		lines := strings.Split(text, "\n")
		endIdx := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				endIdx = i
				break
			}
		}

		if endIdx == -1 {
			a.GetErrorHandler().ShowError(a.ctx, "Front matter not properly closed with ---")
			return
		}

		// Extract front matter YAML
		yamlContent := strings.Join(lines[1:endIdx], "\n")
		var frontMatter struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Category    string `yaml:"category"`
		}

		if err := yaml.Unmarshal([]byte(yamlContent), &frontMatter); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to parse YAML front matter: %v", err))
			return
		}

		// Extract prompt content
		promptLines := lines[endIdx+1:]
		promptText := strings.Join(promptLines, "\n")
		promptText = strings.TrimSpace(promptText)

		// Validate required fields
		if strings.TrimSpace(frontMatter.Name) == "" {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt name is required in front matter")
			return
		}
		if strings.TrimSpace(frontMatter.Category) == "" {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt category is required in front matter")
			return
		}
		if strings.TrimSpace(promptText) == "" {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt content cannot be empty")
			return
		}

		// Update the prompt
		if err := promptService.UpdatePrompt(a.ctx, promptID, frontMatter.Name, frontMatter.Description, promptText, frontMatter.Category); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to update prompt: %v", err))
			return
		}

		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Updated prompt: %s", promptName))
	}()
}

// executePromptExport exports a prompt to a markdown file
func (a *App) executePromptExport(args []string) {
	if len(args) < 2 {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Usage: prompt export <id|name> <file_path>")
		}()
		return
	}

	identifier := args[0]
	filePath := args[1]

	// Get services
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		}()
		return
	}

	go func() {
		// Add timeout protection for file operations
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()

		var promptID int
		var err error

		// Try to parse as ID first
		if id, parseErr := strconv.Atoi(identifier); parseErr == nil {
			promptID = id
			// Validate that the ID exists by trying to get the prompt
			if _, validateErr := promptService.GetPrompt(ctx, promptID); validateErr != nil {
				go func() {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Prompt with ID %d not found", promptID))
				}()
				return
			}
		} else {
			// Try to find by name
			prompt, findErr := promptService.FindPromptByName(ctx, identifier)
			if findErr != nil {
				go func() {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Prompt not found: %s", identifier))
				}()
				return
			}
			promptID = prompt.ID
		}

		if err = promptService.ExportToFile(ctx, promptID, filePath); err != nil {
			go func() {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to export prompt: %v", err))
			}()
			return
		}

		go func() {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Exported prompt to %s", filePath))
		}()
	}()
}

// executePromptDelete deletes a prompt
func (a *App) executePromptDelete(args []string) {
	if len(args) == 0 {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Usage: prompt delete <id|name>")
		}()
		return
	}

	identifier := args[0]

	// Get services
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		}()
		return
	}

	go func() {
		// Add timeout protection for database operations
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()

		var promptID int
		var promptName string

		// Try to parse as ID first
		if id, parseErr := strconv.Atoi(identifier); parseErr == nil {
			promptID = id
			// Get prompt to show name in confirmation and validate existence
			if prompt, err := promptService.GetPrompt(ctx, id); err == nil {
				promptName = prompt.Name
			} else {
				go func() {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Prompt with ID %d not found", id))
				}()
				return
			}
		} else {
			// Try to find by name
			prompt, findErr := promptService.FindPromptByName(ctx, identifier)
			if findErr != nil {
				go func() {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Prompt not found: %s", identifier))
				}()
				return
			}
			promptID = prompt.ID
			promptName = prompt.Name
		}

		// Delete the prompt
		if err := promptService.DeletePrompt(ctx, promptID); err != nil {
			go func() {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to delete prompt: %v", err))
			}()
			return
		}

		go func() {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Deleted prompt: %s", promptName))
		}()
	}()
}

// executePromptStats handles :prompt stats command to show usage statistics
func (a *App) executePromptStats(args []string) {
	// Get prompt service
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		}()
		return
	}

	// Get usage statistics
	go func() {
		stats, err := promptService.GetUsageStats(a.ctx)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to get usage stats: %v", err))
			return
		}

		// Show stats in full-screen display (following preload status pattern)
		a.showPromptStats(stats)
	}()
}

// executeThemeCommand handles :theme commands for theme management
func (a *App) executeThemeCommand(args []string) {
	themeService := a.GetThemeService()
	if themeService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Theme service not available")
		}()
		return
	}

	if len(args) == 0 {
		// Default: show current theme
		go func() {
			if currentTheme, err := themeService.GetCurrentTheme(a.ctx); err == nil {
				a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Current theme: %s", currentTheme))
			} else {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to get current theme: %v", err))
			}
		}()
		return
	}

	subCommand := strings.ToLower(args[0])
	subArgs := args[1:]

	switch subCommand {
	case "list", "l":
		a.executeThemeList()
	case "set", "s":
		if len(subArgs) > 0 {
			a.executeThemeSet(subArgs[0])
		} else {
			go func() {
				a.GetErrorHandler().ShowError(a.ctx, "Usage: theme set <theme-name>")
			}()
		}
	case "preview", "p":
		if len(subArgs) > 0 {
			a.executeThemePreview(subArgs[0])
		} else {
			go func() {
				a.GetErrorHandler().ShowError(a.ctx, "Usage: theme preview <theme-name>")
			}()
		}
	default:
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Unknown theme command: %s. Use 'list', 'set', or 'preview'", subCommand))
		}()
	}
}

// executeThemeList lists all available themes
func (a *App) executeThemeList() {
	themeService := a.GetThemeService()
	if themeService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Theme service not available")
		}()
		return
	}

	go func() {
		themes, err := themeService.ListAvailableThemes(a.ctx)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to list themes: %v", err))
			return
		}

		currentTheme, _ := themeService.GetCurrentTheme(a.ctx)

		output := "🎨 Available Themes\n"
		output += "==================\n\n"
		for _, theme := range themes {
			marker := "○"
			if theme == currentTheme {
				marker = "✅"
			}
			output += fmt.Sprintf("  %s %s", marker, theme)
			if theme == currentTheme {
				output += " (current)"
			}
			output += "\n"
		}
		output += "\n💡 Commands:\n"
		output += "   :theme set <name>     - Switch to theme\n"
		output += "   :theme preview <name> - Preview theme\n"
		output += "   :theme                - Open theme picker\n"
		output += "   H                     - Open theme picker (shortcut)\n"

		// Display in text view like theme preview
		a.QueueUpdateDraw(func() {
			// Update the text container title
			if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
				textContainer.SetTitle(" 🎨 Theme List ")
				textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())

				// Store and hide message headers
				if header, ok := a.views["header"].(*tview.TextView); ok {
					headerContent := header.GetText(false)
					a.originalHeaderHeight = a.calculateHeaderHeight(headerContent)
					textContainer.ResizeItem(header, 0, 0)
				}
			}

			if textView, ok := a.views["text"].(*tview.TextView); ok {
				textView.SetText(output)
				textView.ScrollToBeginning()
				a.SetFocus(textView)
				a.currentFocus = "text"
				a.updateFocusIndicators("text")
			}
			// Also update enhanced text view if available
			if a.enhancedTextView != nil {
				a.enhancedTextView.SetContent(output)
			}
		})

		// Show info in status
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Listed %d themes | Press :theme to open picker", len(themes)))
		}()
	}()
}

// executeThemeSet switches to the specified theme
func (a *App) executeThemeSet(themeName string) {
	themeService := a.GetThemeService()
	if themeService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Theme service not available")
		}()
		return
	}

	go func() {
		if err := themeService.ApplyTheme(a.ctx, themeName); err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply theme '%s': %v", themeName, err))
			return
		}

		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("🎨 Applied theme: %s", themeName))
	}()
}

// executeThemePreview shows a preview of the specified theme
func (a *App) executeThemePreview(themeName string) {
	themeService := a.GetThemeService()
	if themeService == nil {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Theme service not available")
		}()
		return
	}

	go func() {
		themeConfig, err := themeService.PreviewTheme(a.ctx, themeName)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to preview theme '%s': %v", themeName, err))
			return
		}

		// Create preview text
		preview := fmt.Sprintf("🎨 Theme Preview: %s\n\n", themeConfig.Name)
		if themeConfig.Description != "" {
			preview += fmt.Sprintf("Description: %s\n\n", themeConfig.Description)
		}

		preview += "📧 Email Colors:\n"
		preview += fmt.Sprintf("  🔵 Unread:     %s\n", themeConfig.EmailColors.UnreadColor)
		preview += fmt.Sprintf("  ⚪ Read:       %s\n", themeConfig.EmailColors.ReadColor)
		preview += fmt.Sprintf("  🔴 Important:  %s\n", themeConfig.EmailColors.ImportantColor)
		preview += fmt.Sprintf("  🟢 Sent:       %s\n", themeConfig.EmailColors.SentColor)
		preview += fmt.Sprintf("  🟡 Draft:      %s\n", themeConfig.EmailColors.DraftColor)

		preview += "\n🎨 UI Colors:\n"
		preview += fmt.Sprintf("  📝 Text:       %s\n", themeConfig.UIColors.FgColor)
		preview += fmt.Sprintf("  🖼️  Background: %s\n", themeConfig.UIColors.BgColor)
		preview += fmt.Sprintf("  🔲 Border:     %s\n", themeConfig.UIColors.BorderColor)
		preview += fmt.Sprintf("  ✨ Focus:      %s\n", themeConfig.UIColors.FocusColor)

		preview += fmt.Sprintf("\n💡 Use ':theme set %s' to apply this theme", themeName)

		a.GetErrorHandler().ShowInfo(a.ctx, preview)
	}()
}

// executeSaveQueryCommand handles :save-query commands
func (a *App) executeSaveQueryCommand(args []string) {
	// Optional: accept name as argument
	if len(args) > 0 {
		// Save with provided name
		name := strings.Join(args, " ")
		currentQuery := a.getCurrentSearchQuery()
		if strings.TrimSpace(currentQuery) == "" {
			a.showError("No current search to save. Perform a search first.")
			return
		}

		// Get query service
		queryService := a.GetQueryService()
		if queryService == nil {
			a.showError("Query service not available")
			return
		}

		// Set account email if available
		if queryServiceImpl, ok := queryService.(*services.QueryServiceImpl); ok {
			if email := a.getActiveAccountEmail(); email != "" {
				queryServiceImpl.SetAccountEmail(email)
			}
		}

		go func() {
			_, err := queryService.SaveQuery(a.ctx, name, currentQuery, "", "general")
			if err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to save query: %v", err))
			} else {
				a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Saved query: %s", name))
			}
		}()
	} else {
		// Open save dialog
		a.showSaveCurrentQueryDialog()
	}
}

// executeBookmarksCommand handles :bookmarks commands
func (a *App) executeBookmarksCommand(args []string) {
	// Use the existing showSavedQueriesPicker method which handles loading and display
	a.showSavedQueriesPicker()
}

// executeBookmarkCommand handles :bookmark <name> commands
func (a *App) executeBookmarkCommand(args []string) {
	if len(args) == 0 {
		// If no name provided, open bookmarks picker
		a.executeBookmarksCommand(args)
		return
	}

	name := strings.Join(args, " ")
	a.executeQueryByName(name)
}

// Threading command implementations

// executeThreadsCommand handles :threads command
func (a *App) executeThreadsCommand(args []string) {
	if !a.IsThreadingEnabled() {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Threading is disabled in configuration")
		}()
		return
	}

	// Set focus state like the keyboard shortcut
	a.currentFocus = "list"
	a.updateFocusIndicators("list")

	// Switch to thread mode
	a.SetCurrentThreadViewMode(ThreadViewThread)
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, "📧 Switched to threaded view")
	}()

	// Refresh the view to show threads
	go a.refreshThreadView()
}

// executeFlattenCommand handles :flatten command
func (a *App) executeFlattenCommand(args []string) {
	// Set focus state like the keyboard shortcut
	a.currentFocus = "list"
	a.updateFocusIndicators("list")

	// Switch to flat mode
	a.SetCurrentThreadViewMode(ThreadViewFlat)
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, "📄 Switched to flat view")
	}()

	// Refresh the view to show flat messages
	go a.refreshFlatView()
}

// executeThreadSummaryCommand handles :thread-summary command
func (a *App) executeThreadSummaryCommand(args []string) {
	if !a.IsThreadingEnabled() {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Threading is disabled in configuration")
		}()
		return
	}

	if a.GetCurrentThreadViewMode() != ThreadViewThread {
		go func() {
			a.GetErrorHandler().ShowWarning(a.ctx, "Thread summary only available in threaded view")
		}()
		return
	}

	go func() { _ = a.GenerateThreadSummary() }()
}

// executeExpandAllCommand handles :expand-all command
func (a *App) executeExpandAllCommand(args []string) {
	if !a.IsThreadingEnabled() {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Threading is disabled in configuration")
		}()
		return
	}

	if a.GetCurrentThreadViewMode() != ThreadViewThread {
		go func() {
			a.GetErrorHandler().ShowWarning(a.ctx, "Thread expansion only available in threaded view")
		}()
		return
	}

	go func() { _ = a.ExpandAllThreads() }()
}

// executeCollapseAllCommand handles :collapse-all command
func (a *App) executeCollapseAllCommand(args []string) {
	if !a.IsThreadingEnabled() {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "Threading is disabled in configuration")
		}()
		return
	}

	if a.GetCurrentThreadViewMode() != ThreadViewThread {
		go func() {
			a.GetErrorHandler().ShowWarning(a.ctx, "Thread collapse only available in threaded view")
		}()
		return
	}

	go func() { _ = a.CollapseAllThreads() }()
}

// executeReplyAllCommand handles :reply-all/:ra commands
func (a *App) executeReplyAllCommand(args []string) {
	go a.replyAllSelected()
}

// executeForwardCommand handles :forward/:f commands
func (a *App) executeForwardCommand(args []string) {
	go a.forwardSelected()
}

// executeDraftsCommand handles :drafts/:dr commands
func (a *App) executeDraftsCommand(args []string) {
	go a.loadDrafts()
}
