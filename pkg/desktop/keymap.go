package desktop

import "strings"

// KeyMap holds the resolved keyboard shortcuts for the actions the desktop
// client implements, read from the user's GizTUI config (with the same defaults
// as the TUI). Keys use TUI notation: single chars ("a"), "space", "gg", etc.
type KeyMap struct {
	Summarize    string `json:"summarize"`
	Prompt       string `json:"prompt"`
	Archive      string `json:"archive"`
	Trash        string `json:"trash"`
	ToggleRead   string `json:"toggleRead"`
	ManageLabels string `json:"manageLabels"`
	Compose      string `json:"compose"`
	Reply        string `json:"reply"`
	Forward      string `json:"forward"`
	Search       string `json:"search"`
	Refresh      string `json:"refresh"`
	LoadMore     string `json:"loadMore"`
	Drafts       string `json:"drafts"`
	OpenGmail    string `json:"openGmail"`
	BulkMode     string `json:"bulkMode"`
	BulkSelect   string `json:"bulkSelect"`
	Markdown     string `json:"markdown"`
	Attachments  string `json:"attachments"`
	Help         string `json:"help"`
	GotoTop      string `json:"gotoTop"`
	GotoBottom   string `json:"gotoBottom"`
	LinkPicker   string `json:"linkPicker"`
	ReplyAll     string `json:"replyAll"`
	SaveMessage  string `json:"saveMessage"`
	SuggestLabel string `json:"suggestLabel"`
	Obsidian     string `json:"obsidian"`
	Slack        string `json:"slack"`
	CommandMode  string `json:"commandMode"`
	Threading    string `json:"threading"`
	SavedQueries string `json:"savedQueries"`
	SaveQuery    string `json:"saveQuery"`
	VimTimeoutMs int    `json:"vimTimeoutMs"`
}

// DefaultKeyMap returns the built-in defaults, matching the TUI's
// DefaultKeyBindings for the subset of actions the desktop supports.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Summarize: "y", Prompt: "p", Archive: "a", Trash: "d", ToggleRead: "t",
		ManageLabels: "l", Compose: "c", Reply: "r", Forward: "f", Search: "s",
		Refresh: "R", LoadMore: "N", Drafts: "D", OpenGmail: "O", BulkMode: "v",
		BulkSelect: "space", Markdown: "M", Attachments: "A", Help: "?",
		GotoTop: "gg", GotoBottom: "G", LinkPicker: "L", ReplyAll: "E",
		SaveMessage: "w", SuggestLabel: "o", Obsidian: "O", Slack: "K",
		CommandMode: ":", Threading: "T", SavedQueries: "Q", SaveQuery: "Z",
		VimTimeoutMs: 1000,
	}
}

// KeyMap resolves the user's configured shortcuts, falling back to defaults for
// any binding they haven't customized.
func (s *Session) KeyMap() KeyMap {
	km := DefaultKeyMap()
	if s == nil || s.Config == nil {
		return km
	}
	k := s.Config.Keys
	km.Summarize = orDefault(k.Summarize, km.Summarize)
	km.Prompt = orDefault(k.Prompt, km.Prompt)
	km.Archive = orDefault(k.Archive, km.Archive)
	km.Trash = orDefault(k.Trash, km.Trash)
	km.ToggleRead = orDefault(k.ToggleRead, km.ToggleRead)
	km.ManageLabels = orDefault(k.ManageLabels, km.ManageLabels)
	km.Compose = orDefault(k.Compose, km.Compose)
	km.Reply = orDefault(k.Reply, km.Reply)
	km.Forward = orDefault(k.Forward, km.Forward)
	km.Search = orDefault(k.Search, km.Search)
	km.Refresh = orDefault(k.Refresh, km.Refresh)
	km.LoadMore = orDefault(k.LoadMore, km.LoadMore)
	km.Drafts = orDefault(k.Drafts, km.Drafts)
	km.OpenGmail = orDefault(k.OpenGmail, km.OpenGmail)
	km.BulkMode = orDefault(k.BulkMode, km.BulkMode)
	km.BulkSelect = orDefault(k.BulkSelect, km.BulkSelect)
	km.Markdown = orDefault(k.Markdown, km.Markdown)
	km.Attachments = orDefault(k.Attachments, km.Attachments)
	km.Help = orDefault(k.Help, km.Help)
	km.GotoTop = orDefault(k.GotoTop, km.GotoTop)
	km.GotoBottom = orDefault(k.GotoBottom, km.GotoBottom)
	km.LinkPicker = orDefault(k.LinkPicker, km.LinkPicker)
	km.ReplyAll = orDefault(k.ReplyAll, km.ReplyAll)
	km.SaveMessage = orDefault(k.SaveMessage, km.SaveMessage)
	km.SuggestLabel = orDefault(k.SuggestLabel, km.SuggestLabel)
	km.Obsidian = orDefault(k.Obsidian, km.Obsidian)
	km.Slack = orDefault(k.Slack, km.Slack)
	km.CommandMode = orDefault(k.CommandMode, km.CommandMode)
	km.Threading = orDefault(k.ToggleThreading, km.Threading)
	km.SavedQueries = orDefault(k.QueryBookmarks, km.SavedQueries)
	km.SaveQuery = orDefault(k.SaveQuery, km.SaveQuery)
	if k.VimNavigationTimeoutMs > 0 {
		km.VimTimeoutMs = k.VimNavigationTimeoutMs
	}
	return km
}

func orDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}
