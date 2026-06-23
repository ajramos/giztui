package tui

import (
	"sort"
	"strings"
)

// argCompleter returns candidate completions for the current (last) argument token of a command.
// prefix is the partial token the user has typed (may be ""). Implementations must NOT block the
// event loop (no network) — read already-loaded state only.
type argCompleter func(a *App, prefix string) []string

// commandSpec is one entry in the command registry: a canonical command name, its aliases, and an
// optional argument completer. The registry mirrors the executeCommand switch and is the single
// source of truth for Tab completion.
type commandSpec struct {
	name        string
	aliases     []string
	completeArg argCompleter
}

// commandRegistry lists every top-level `:` command. Keep in sync with the executeCommand switch in
// commands.go. Adding a command here is all that's needed for it to autocomplete.
var commandRegistry = []commandSpec{
	{name: "labels", aliases: []string{"l"}, completeArg: completeLabelArg},
	{name: "links", aliases: []string{"link"}},
	{name: "attachments", aliases: []string{"attach"}},
	{name: "gmail", aliases: []string{"web", "open-web", "o"}},
	{name: "search", completeArg: completeSearchArg},
	{name: "slack", aliases: []string{"sl"}, completeArg: completeSlackArg},
	{name: "s"},
	{name: "summary"},
	{name: "rsvp"},
	{name: "inbox", aliases: []string{"i"}},
	{name: "compose", aliases: []string{"c"}},
	{name: "headers", aliases: []string{"toggle-headers"}},
	{name: "threads", aliases: []string{"thr"}},
	{name: "flatten", aliases: []string{"flat"}},
	{name: "thread-summary", aliases: []string{"th-sum"}},
	{name: "expand-all", aliases: []string{"expand"}},
	{name: "collapse-all", aliases: []string{"collapse"}},
	{name: "help", aliases: []string{"h"}},
	{name: "numbers", aliases: []string{"n"}},
	{name: "quit", aliases: []string{"q"}},
	{name: "cache"},
	{name: "preload", aliases: []string{"pl"}},
	{name: "stats", aliases: []string{"usage"}},
	{name: "g"},
	{name: "archive", aliases: []string{"a"}},
	{name: "trash", aliases: []string{"d"}},
	{name: "read", aliases: []string{"toggle-read", "t"}},
	{name: "new"},
	{name: "reply", aliases: []string{"r"}},
	{name: "reply-all", aliases: []string{"ra"}},
	{name: "forward", aliases: []string{"f"}},
	{name: "drafts", aliases: []string{"dr"}},
	{name: "refresh"},
	{name: "autorefresh", aliases: []string{"arr"}},
	{name: "config", aliases: []string{"cfg"}},
	{name: "load", aliases: []string{"more", "next"}},
	{name: "unread", aliases: []string{"u"}},
	{name: "undo"},
	{name: "archived", aliases: []string{"arch-search", "b"}},
	{name: "select", aliases: []string{"sel"}},
	{name: "move", aliases: []string{"mv"}, completeArg: completeLabelArg},
	{name: "label", aliases: []string{"lbl"}, completeArg: completeLabelArg},
	{name: "obsidian", aliases: []string{"obs"}},
	{name: "accounts", aliases: []string{"acc"}, completeArg: completeAccountArg},
	{name: "prompt", aliases: []string{"pr", "p"}, completeArg: completePromptArg},
	{name: "prompt-new", aliases: []string{"pn"}},
	{name: "prompt-refine", aliases: []string{"prf"}},
	{name: "prompt-save", aliases: []string{"ps"}},
	{name: "action-plan", aliases: []string{"plan", "ap"}},
	{name: "markdown", aliases: []string{"md"}},
	{name: "touch-up", aliases: []string{"touchup"}},
	{name: "theme", aliases: []string{"th"}, completeArg: completeThemeArg},
	{name: "save-query", aliases: []string{"save", "sq"}},
	{name: "bookmarks", aliases: []string{"queries", "bm", "qb"}},
	{name: "bookmark", aliases: []string{"query"}, completeArg: completeQueryArg},
}

// lookupCommand resolves a command token (name or alias, case-insensitive) to its spec, or nil.
func lookupCommand(token string) *commandSpec {
	token = strings.ToLower(token)
	for i := range commandRegistry {
		s := &commandRegistry[i]
		if strings.ToLower(s.name) == token {
			return s
		}
		for _, al := range s.aliases {
			if strings.ToLower(al) == token {
				return s
			}
		}
	}
	return nil
}

// matchesPrefix reports whether the spec's name or any alias starts with lowerPrefix.
func matchesPrefix(s *commandSpec, lowerPrefix string) bool {
	if strings.HasPrefix(strings.ToLower(s.name), lowerPrefix) {
		return true
	}
	for _, al := range s.aliases {
		if strings.HasPrefix(strings.ToLower(al), lowerPrefix) {
			return true
		}
	}
	return false
}

// commandCandidates returns the ordered Tab candidates for the given command-bar text. With no
// space yet it completes the command token (returns matching canonical names, sorted, de-duped).
// With a "command <args>" shape it delegates to the command's argument completer for the last token.
// Returns nil when nothing matches. The input is NOT trimmed of a trailing space (a trailing space
// means "complete the next, empty, argument").
func (a *App) commandCandidates(text string) []string {
	text = strings.TrimLeft(text, " ")
	if text == "" {
		return nil
	}

	// Argument completion: "command<space>...".
	if i := strings.IndexByte(text, ' '); i >= 0 {
		spec := lookupCommand(text[:i])
		if spec == nil || spec.completeArg == nil {
			return nil
		}
		rest := text[i+1:] // everything after "command "
		head := ""         // already-typed arg tokens, including the trailing space
		argPrefix := rest
		if ls := strings.LastIndexByte(rest, ' '); ls >= 0 {
			head = rest[:ls+1]
			argPrefix = rest[ls+1:]
		}
		cands := spec.completeArg(a, argPrefix)
		if len(cands) == 0 {
			return nil
		}
		linePrefix := text[:i] + " " + head
		out := make([]string, 0, len(cands))
		for _, c := range cands {
			out = append(out, linePrefix+c)
		}
		return out
	}

	// Command-token completion.
	lower := strings.ToLower(text)
	seen := map[string]bool{}
	var out []string
	for i := range commandRegistry {
		s := &commandRegistry[i]
		if matchesPrefix(s, lower) && !seen[s.name] {
			seen[s.name] = true
			out = append(out, s.name)
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

// filterByPrefix returns the items that start with prefix (case-insensitive), sorted
// case-insensitively. nil when nothing matches. Shared by all argument completers.
func filterByPrefix(items []string, prefix string) []string {
	lower := strings.ToLower(prefix)
	var out []string
	for _, s := range items {
		if s != "" && strings.HasPrefix(strings.ToLower(s), lower) {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

// gmailSearchOperators are the Gmail search operators offered after :search (see GMAIL_SEARCH_REFERENCE.md).
var gmailSearchOperators = []string{
	"from:", "to:", "cc:", "bcc:", "subject:", "has:attachment",
	"is:unread", "is:read", "is:starred", "is:important",
	"label:", "in:inbox", "in:sent", "in:spam", "in:trash",
	"after:", "before:", "newer_than:", "older_than:", "filename:",
	"larger:", "smaller:",
}

// completeLabelArg completes a label-name argument from the pre-fetched label list (a.cmd.labelNames,
// populated off the event loop when the command bar opens). The other dynamic completers follow the
// same pattern; the static ones (search/slack/accounts) read constants/in-memory config directly.
func completeLabelArg(a *App, prefix string) []string {
	return filterByPrefix(a.cmd.labelNames, prefix)
}
func completePromptArg(a *App, prefix string) []string {
	return filterByPrefix(a.cmd.promptNames, prefix)
}
func completeThemeArg(a *App, prefix string) []string {
	return filterByPrefix(a.cmd.themeNames, prefix)
}
func completeQueryArg(a *App, prefix string) []string {
	return filterByPrefix(a.cmd.queryNames, prefix)
}

// completeSearchArg completes Gmail search operators (static list; no I/O).
func completeSearchArg(a *App, prefix string) []string {
	return filterByPrefix(gmailSearchOperators, prefix)
}

// completeSlackArg completes configured Slack channel names (in-memory config; no I/O).
func completeSlackArg(a *App, prefix string) []string {
	if a.Config == nil {
		return nil
	}
	names := make([]string, 0, len(a.Config.Slack.Channels))
	for _, ch := range a.Config.Slack.Channels {
		names = append(names, ch.Name)
	}
	return filterByPrefix(names, prefix)
}

// completeAccountArg completes configured account IDs (for :accounts switch <id>; in-memory config).
func completeAccountArg(a *App, prefix string) []string {
	if a.Config == nil {
		return nil
	}
	ids := make([]string, 0, len(a.Config.Accounts))
	for _, ac := range a.Config.Accounts {
		ids = append(ids, ac.ID)
	}
	return filterByPrefix(ids, prefix)
}

// completionPromptNames / completionThemeNames / completionQueryNames fetch the dynamic argument
// lists off the event loop (called from the command-bar open goroutine; they touch the DB / disk).
func (a *App) completionPromptNames() []string {
	_, _, _, _, _, _, ps, _, _, _, _, _ := a.GetServices()
	if ps == nil {
		return nil
	}
	list, err := ps.ListPrompts(a.ctx, "")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, p := range list {
		if p != nil && p.Name != "" {
			out = append(out, p.Name)
		}
	}
	return out
}

func (a *App) completionThemeNames() []string {
	ts := a.GetThemeService()
	if ts == nil {
		return nil
	}
	names, err := ts.ListAvailableThemes(a.ctx)
	if err != nil {
		return nil
	}
	return names
}

func (a *App) completionQueryNames() []string {
	qs := a.GetQueryService()
	if qs == nil {
		return nil
	}
	list, err := qs.ListQueries(a.ctx, "")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, q := range list {
		if q != nil && q.Name != "" {
			out = append(out, q.Name)
		}
	}
	return out
}
