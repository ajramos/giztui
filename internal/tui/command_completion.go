package tui

import (
	"sort"
	"strings"
)

// argCompleter returns full-replacement candidates for the argument text `rest` (everything the user
// typed after "command "). Each returned string replaces `rest` entirely; the engine prepends
// "command ". Passing the whole arg text lets a completer honor its command's grammar (subcommands,
// names with spaces). Implementations must NOT block the event loop (no network) — read already-loaded
// state only.
type argCompleter func(a *App, rest string) []string

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
	{name: "labels", aliases: []string{"l"}, completeArg: completeLabelsArg},
	{name: "links", aliases: []string{"link"}},
	{name: "attachments", aliases: []string{"attach"}},
	{name: "gmail", aliases: []string{"web", "open-web", "o"}},
	{name: "search", completeArg: completeSearchArg},
	{name: "slack", aliases: []string{"sl"}},
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
	{name: "move", aliases: []string{"mv"}},
	{name: "label", aliases: []string{"lbl"}},
	{name: "obsidian", aliases: []string{"obs"}},
	{name: "accounts", aliases: []string{"acc"}, completeArg: completeAccountsArg},
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
	{name: "bookmark", aliases: []string{"query"}, completeArg: completeBookmarkArg},
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

	// Argument completion: "command<space>...". The completer is handed everything after the command
	// and returns full replacements for that argument text.
	if i := strings.IndexByte(text, ' '); i >= 0 {
		spec := lookupCommand(text[:i])
		if spec == nil || spec.completeArg == nil {
			return nil
		}
		cands := spec.completeArg(a, text[i+1:])
		if len(cands) == 0 {
			return nil
		}
		out := make([]string, 0, len(cands))
		for _, c := range cands {
			out = append(out, text[:i]+" "+c)
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

// splitLastToken splits arg text into the head (everything up to and including the last space) and
// the final, partial token. A trailing space yields prefix=="" (completing a fresh token).
func splitLastToken(rest string) (head, prefix string) {
	if ls := strings.LastIndexByte(rest, ' '); ls >= 0 {
		return rest[:ls+1], rest[ls+1:]
	}
	return "", rest
}

// firstToken returns the first whitespace-separated token of rest, lowercased ("" if none).
func firstToken(rest string) string {
	f := strings.Fields(rest)
	if len(f) == 0 {
		return ""
	}
	return strings.ToLower(f[0])
}

// withHead prepends head to each match, returning nil when there are no matches.
func withHead(head string, matches []string) []string {
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, head+m)
	}
	return out
}

// --- argument completers (each honors its command's real grammar) ---

// completeSearchArg completes the current token with a Gmail search operator, at any position.
func completeSearchArg(a *App, rest string) []string {
	head, prefix := splitLastToken(rest)
	return withHead(head, filterByPrefix(gmailSearchOperators, prefix))
}

// completeLabelsArg: ':labels <subcommand> [name]'. First token → add/list/remove; after add/remove
// → a label name (from the pre-fetched a.cmd.labelNames).
func completeLabelsArg(a *App, rest string) []string {
	head, prefix := splitLastToken(rest)
	if head == "" {
		return withHead("", filterByPrefix([]string{"add", "list", "remove"}, prefix))
	}
	switch firstToken(rest) {
	case "add", "remove", "rm":
		return withHead(head, filterByPrefix(a.cmd.labelNames, prefix))
	}
	return nil
}

// completePromptArg: ':prompt <subcommand>' is a subcommand dispatcher — only the subcommand keyword
// is completable (prompt *names* are not command arguments; :prompt with no args opens the manager).
func completePromptArg(a *App, rest string) []string {
	head, prefix := splitLastToken(rest)
	if head != "" {
		return nil
	}
	return withHead("", filterByPrefix([]string{"create", "delete", "export", "list", "stats", "update"}, prefix))
}

// completeThemeArg: ':theme <subcommand> [name]'. First token → list/preview/set; after set/preview
// → a theme name (from the pre-fetched a.cmd.themeNames).
func completeThemeArg(a *App, rest string) []string {
	head, prefix := splitLastToken(rest)
	if head == "" {
		return withHead("", filterByPrefix([]string{"list", "preview", "set"}, prefix))
	}
	switch firstToken(rest) {
	case "set", "preview":
		return withHead(head, filterByPrefix(a.cmd.themeNames, prefix))
	}
	return nil
}

// completeAccountsArg: ':accounts switch <id>'. First token → switch; after switch → an account id.
func completeAccountsArg(a *App, rest string) []string {
	head, prefix := splitLastToken(rest)
	if head == "" {
		return withHead("", filterByPrefix([]string{"switch"}, prefix))
	}
	if firstToken(rest) == "switch" {
		return withHead(head, filterByPrefix(a.accountIDs(), prefix))
	}
	return nil
}

// completeBookmarkArg: ':bookmark <saved-query name>'. The whole rest is the name (the command joins
// args with spaces), so it is matched as a unit against the saved-query names.
func completeBookmarkArg(a *App, rest string) []string {
	return filterByPrefix(a.cmd.queryNames, rest)
}

// accountIDs returns the configured account IDs (in-memory config; no I/O).
func (a *App) accountIDs() []string {
	if a.Config == nil {
		return nil
	}
	ids := make([]string, 0, len(a.Config.Accounts))
	for _, ac := range a.Config.Accounts {
		ids = append(ids, ac.ID)
	}
	return ids
}

// completionThemeNames / completionQueryNames fetch the dynamic argument lists off the event loop
// (called from the command-bar open goroutine; they touch the DB / disk).
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

// levenshtein returns the edit distance between a and b (two-row DP, O(len(a)*len(b))).
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// closestCommand returns the canonical command name nearest to typed (case-insensitive Levenshtein
// over every registry name and alias of length >= 3), or ("", false) when there is no confident
// suggestion. Guards: typed must be >= 3 chars, and the best distance must be in (0, 2].
func closestCommand(typed string) (string, bool) {
	typed = strings.ToLower(strings.TrimSpace(typed))
	if len(typed) < 3 {
		return "", false
	}
	best := ""
	bestDist := 1 << 30
	consider := func(candidate, canonical string) {
		if len(candidate) < 3 {
			return
		}
		d := levenshtein(typed, strings.ToLower(candidate))
		if d < bestDist {
			bestDist = d
			best = canonical
		}
	}
	for i := range commandRegistry {
		s := &commandRegistry[i]
		consider(s.name, s.name)
		for _, al := range s.aliases {
			consider(al, s.name)
		}
	}
	if best != "" && bestDist > 0 && bestDist <= 2 {
		return best, true
	}
	return "", false
}
