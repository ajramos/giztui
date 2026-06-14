package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// buildAnalyzerMessages converts already-loaded message metadata into the lightweight
// AnalyzerMessage list the InboxAnalyzerService consumes. Only UNREAD messages are
// included. No Gmail calls are made — everything comes from in-memory metadata.
func buildAnalyzerMessages(metas []*gmailapi.Message) []services.AnalyzerMessage {
	out := make([]services.AnalyzerMessage, 0, len(metas))
	for _, m := range metas {
		if m == nil {
			continue
		}
		if !isUnreadMeta(m) {
			continue
		}
		out = append(out, services.AnalyzerMessage{
			ID:      m.Id,
			Subject: extractHeaderValue(m, "Subject"),
			From:    extractHeaderValue(m, "From"),
			Snippet: m.Snippet,
		})
	}
	return out
}

// buildAnalyzerMessagesForSelection converts the explicitly-selected messages into
// AnalyzerMessages. Unlike buildAnalyzerMessages it does NOT filter by UNREAD — an
// explicit selection counts regardless of read state.
func buildAnalyzerMessagesForSelection(metas []*gmailapi.Message, selected map[string]bool) []services.AnalyzerMessage {
	out := make([]services.AnalyzerMessage, 0, len(selected))
	for _, m := range metas {
		if m == nil || !selected[m.Id] {
			continue
		}
		out = append(out, services.AnalyzerMessage{
			ID:      m.Id,
			Subject: extractHeaderValue(m, "Subject"),
			From:    extractHeaderValue(m, "From"),
			Snippet: m.Snippet,
		})
	}
	return out
}

// isUnreadMeta reports whether a raw message metadata carries the UNREAD label.
func isUnreadMeta(m *gmailapi.Message) bool {
	for _, l := range m.LabelIds {
		if l == "UNREAD" {
			return true
		}
	}
	return false
}

// emailRef identifies an email node within a category.
type emailRef struct {
	catIndex int
	msgID    string
}

// actionPlanState holds the mutable state of the Action Plan panel.
type actionPlanState struct {
	plan             *services.ActionPlan
	selectedCategory int
	analyzing        atomic.Bool // true while batches are still streaming; blocks quick-actions

	customPromptText string // override prompt text, "" = default
	scopeLabel       string // "N selected" or "N unread (inbox)"

	excluded      map[string]bool              // message IDs toggled OFF (skip on action)
	expanded      map[int]bool                 // category index → expanded?
	metaByID      map[string]*gmailapi.Message // subject/from lookup for email nodes
	selectedMsgID string                       // msgID of selected email node, "" if a category is selected

	tree            *tview.TreeView
	root            *tview.TreeNode
	footer          *tview.TextView
	container       *tview.Flex
	streamingCancel context.CancelFunc
}

// checkedIDs returns the subset of ids not present in excluded, preserving order.
func checkedIDs(ids []string, excluded map[string]bool) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !excluded[id] {
			out = append(out, id)
		}
	}
	return out
}

// actionVerbLabel maps an action token to a human verb for the category header.
func actionVerbLabel(action string) string {
	switch action {
	case "archive":
		return "Archive"
	case "mark_read":
		return "Mark read"
	case "trash":
		return "Trash"
	case "label":
		return "Label"
	case "summarize":
		return "summarize"
	default:
		return "Review"
	}
}

// actionKeyHint returns the configured key for the action's quick-action, or "" if none.
func (a *App) actionKeyHint(action string) string {
	switch action {
	case "archive":
		return a.Keys.Archive
	case "mark_read":
		return a.Keys.ToggleRead
	case "trash":
		return a.Keys.Trash
	case "label":
		return a.Keys.ManageLabels
	case "summarize":
		return a.Keys.Summarize
	default:
		return ""
	}
}

// openActionPlanPanel opens the Action Plan panel using the built-in default prompt.
func (a *App) openActionPlanPanel() {
	a.openActionPlanWithText("")
}

// openActionPlanWithText opens the panel; customPromptText=="" uses the default prompt.
func (a *App) openActionPlanWithText(customPromptText string) {
	if a.GetInboxAnalyzerService() == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Inbox analyzer not available — check LLM configuration")
		return
	}
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}

	// Scope: selection-first (analyze the user's bulk selection if any), else fall
	// back to the unread inbox already in memory.
	a.mu.RLock()
	metas := make([]*gmailapi.Message, len(a.messagesMeta))
	copy(metas, a.messagesMeta)
	selected := make(map[string]bool, len(a.selected))
	for id, ok := range a.selected {
		if ok {
			selected[id] = true
		}
	}
	a.mu.RUnlock()

	var messages []services.AnalyzerMessage
	scopeLabel := ""
	if len(selected) > 0 {
		messages = buildAnalyzerMessagesForSelection(metas, selected)
		scopeLabel = fmt.Sprintf("%d selected", len(messages))
	} else {
		messages = buildAnalyzerMessages(metas)
		scopeLabel = fmt.Sprintf("%d unread (inbox)", len(messages))
	}
	if len(messages) == 0 {
		a.GetErrorHandler().ShowInfo(a.ctx, "No messages to analyze. Select messages (v/space) or try :search is:unread.")
		return
	}

	colors := a.GetComponentColors("ai")
	bg := colors.Background.Color()

	// Build metaByID lookup for subject/from display in email child nodes.
	metaByID := make(map[string]*gmailapi.Message, len(metas))
	for _, m := range metas {
		if m != nil {
			metaByID[m.Id] = m
		}
	}

	state := &actionPlanState{
		selectedCategory: 0,
		customPromptText: customPromptText,
		scopeLabel:       scopeLabel,
		excluded:         make(map[string]bool),
		expanded:         make(map[int]bool),
		metaByID:         metaByID,
	}
	state.analyzing.Store(true)

	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root).SetCurrentNode(state.root)
	state.tree.SetTopLevel(1) // hide the empty root; categories are the visible top level
	state.tree.SetBackgroundColor(bg)
	state.tree.SetGraphics(true)
	state.tree.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		switch ref := node.GetReference().(type) {
		case int:
			state.selectedCategory = ref
			state.selectedMsgID = ""
		case emailRef:
			state.selectedCategory = ref.catIndex
			state.selectedMsgID = ref.msgID
		}
		a.updateActionPlanFooter(state)
		// tview postpones cursor movement to draw time (process()), and Flex defers the
		// FOCUSED item's Draw to last — so this callback runs AFTER the footer already
		// painted this frame, leaving it one keystroke behind the cursor. Force one more
		// repaint so the footer tracks the highlighted node live. (go avoids QueueUpdateDraw
		// deadlocking when invoked from the UI goroutine mid-draw.)
		go a.QueueUpdateDraw(func() {})
	})

	// Footer matches the other pickers: right-aligned, " X to Y | … " phrasing.
	state.footer = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight)
	state.footer.SetBackgroundColor(bg)
	state.footer.SetTextColor(colors.Text.Color())

	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.SetBackgroundColor(bg)
	state.container.SetBorder(true)
	// The status/summary lives in the border title (no separate header row).
	state.container.SetTitle(actionPlanTitleText(scopeLabel, 0, 0, 0, true))
	state.container.SetTitleColor(colors.Title.Color())
	state.container.SetBorderColor(colors.Border.Color())
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)

	// Immediate "analyzing" feedback so the panel isn't blank before the first batch.
	state.root.AddChild(tview.NewTreeNode("⏳ Analyzing your messages…").
		SetSelectable(false).SetColor(colors.Text.Color()))
	state.tree.SetInputCapture(a.actionPlanInputCapture(state))

	// Mount, focus and activation must run on the UI thread. openActionPlanWithText is
	// invoked via `go a.openActionPlanPanel()`, so doing these directly would mutate the
	// live layout off-thread and nothing would repaint until the next input event — the
	// panel would be invisible and unfocused until a stray keypress. QueueUpdateDraw
	// marshals onto the UI thread AND forces a redraw, the same pattern openLinkPicker uses.
	a.QueueUpdateDraw(func() {
		a.actionPlanState = state
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.RemoveItem(a.labelsView)
			}
			a.labelsView = state.container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.SetFocus(state.tree)
		a.updateActionPlanFooter(state)
		a.currentFocus = "action_plan"
		a.updateFocusIndicators("action_plan")
		a.setActivePicker(PickerActionPlan)
	})

	// Launch analysis in the background. ctx cancel is registered both on the state
	// (for closeActionPlanPanel) and on the App (for the global ESC handler in keys.go).
	ctx, cancel := context.WithCancel(a.ctx)
	state.streamingCancel = cancel
	a.streamingCancel = cancel

	batchSize := a.Config.InboxAnalyzer.BatchSize
	maxBatches := a.Config.InboxAnalyzer.MaxBatches
	bodyCharLimit := a.Config.InboxAnalyzer.BodyCharLimit

	var userRules []string
	if svc := a.GetAnalyzerRulesService(); svc != nil {
		if rs, err := svc.ListRules(a.ctx); err == nil {
			for _, r := range rs {
				userRules = append(userRules, r.RuleText)
			}
		}
	}

	// Existing user labels so the analyzer prefers them for the "label" action (read outside the
	// analysis goroutine to avoid racing it).
	availableLabels := a.userLabelNames()

	// Enrich the analyzer context with each email's plain-text body (opt-in). Cap to what is
	// actually analyzed (BatchSize x MaxBatches) so we never fetch bodies for messages the
	// analyzer would drop. Failures degrade gracefully to the snippet (Body left empty).
	if a.Config.InboxAnalyzer.IncludeBody {
		bs, mb := batchSize, maxBatches
		if bs <= 0 {
			bs = 50
		}
		if mb <= 0 {
			mb = 10
		}
		if capN := bs * mb; len(messages) > capN {
			messages = messages[:capN]
		}
		ids := make([]string, len(messages))
		for i := range messages {
			ids[i] = messages[i].ID
		}
		emailService, _, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
		if emailService != nil {
			go a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Fetching email bodies for %d messages…", len(ids)))
			bodies, err := emailService.GetMessagePlainTexts(a.ctx, ids, 0)
			// ShowProgress sets a PERSISTENT status; clear it once the fetch returns so it
			// never lingers (it is not auto-cleared like transient messages).
			go a.GetErrorHandler().ClearPersistentMessage()
			if err == nil {
				for i := range messages {
					if body := bodies[messages[i].ID]; body != "" {
						messages[i].Body = body
					}
				}
			}
		}
	}

	go func() {
		defer func() {
			cancel()
			state.streamingCancel = nil
			// Only clear the app-level cancel if THIS panel is still active. If the
			// panel was closed and reopened, a.streamingCancel belongs to the new
			// panel's goroutine and must not be clobbered. (func values are not
			// comparable, so we gate on the state pointer instead.)
			if a.actionPlanState == state {
				a.streamingCancel = nil
			}
		}()

		_, err := a.GetInboxAnalyzerService().Analyze(ctx, messages,
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules, BodyCharLimit: bodyCharLimit, AvailableLabels: availableLabels},
			func(p *services.ActionPlan) {
				// Per-batch progress callback (low frequency, NOT per-token). Marshal the
				// render onto the UI thread via QueueUpdateDraw: a bare SetText from a
				// worker goroutine updates the buffer but never repaints the screen until
				// the next input event (this is exactly the pattern bulk_prompts.go uses
				// for its final render). The guard skips a closed/reopened panel.
				if ctx.Err() != nil {
					return
				}
				a.QueueUpdateDraw(func() {
					if a.actionPlanState != state {
						return
					}
					state.plan = p
					a.renderActionPlanPanel(state)
				})
			})

		if ctx.Err() != nil || a.actionPlanState != state {
			return // cancelled or panel replaced; nothing to report
		}
		state.analyzing.Store(false)
		if err != nil {
			if state.plan == nil {
				a.GetErrorHandler().ShowError(a.ctx, "⚠ LLM unavailable. Try again later.")
				return
			}
			a.GetErrorHandler().ShowWarning(a.ctx, "Analysis interrupted — showing partial plan.")
		}
		// Final render on the UI thread so the completed plan is actually painted.
		a.QueueUpdateDraw(func() {
			if a.actionPlanState == state {
				a.renderActionPlanPanel(state)
			}
		})
		switch {
		case state.plan != nil && state.plan.Degraded:
			a.GetErrorHandler().ShowInfo(a.ctx, "ℹ Plan rendered with limited actions — some LLM output was malformed.")
		case state.plan != nil && len(state.plan.Categories) == 0 && len(state.plan.ReadManually) == 0:
			a.GetErrorHandler().ShowInfo(a.ctx, "ℹ Analyzer returned no actionable groups. Press Esc and retry, or try a custom prompt.")
		}
	}()
}

// actionPlanTitleText builds the panel's border title, which carries the live status.
// Before the first batch (batchesTotal==0) it shows "analyzing…"; while batches run it
// shows progress; when done it summarizes the number of groups.
func actionPlanTitleText(scopeLabel string, batchesDone, batchesTotal, groups int, analyzing bool) string {
	if batchesTotal == 0 {
		return fmt.Sprintf(" 📋 Action Plan · %s · analyzing… ", scopeLabel)
	}
	if analyzing {
		return fmt.Sprintf(" 📋 Action Plan · %s · batch %d/%d ", scopeLabel, batchesDone, batchesTotal)
	}
	return fmt.Sprintf(" 📋 Action Plan · %s · %d groups · done ", scopeLabel, groups)
}

// renderActionPlanPanel refreshes the title + tree from the current state. It performs raw
// SetText/SetTitle calls and so must run on the UI thread — callers from the analysis
// goroutine wrap it in QueueUpdateDraw; UI-thread callers (key handlers) may call it directly.
func (a *App) renderActionPlanPanel(state *actionPlanState) {
	if state == nil || state.plan == nil {
		return
	}
	p := state.plan
	state.container.SetTitle(actionPlanTitleText(state.scopeLabel, p.BatchesDone, p.BatchesTotal, len(p.Categories), state.analyzing.Load()))
	// Only clamp an over-range index down; -1 (the read-manually node) is a valid
	// selection and must survive — rebuildActionPlanTree restores it by ref.
	if len(p.Categories) > 0 && state.selectedCategory >= len(p.Categories) {
		state.selectedCategory = len(p.Categories) - 1
	}
	a.rebuildActionPlanTree(state)
}

// topLevelNodeLabel builds the display label for a top-level node: a category (i>=0)
// or the read-manually node (i==-1). The chevron reflects state.expanded[i] so callers
// can refresh it the moment a node is expanded/collapsed, not only on a full rebuild.
func (a *App) topLevelNodeLabel(state *actionPlanState, i int) string {
	chevron := "▶"
	if state.expanded[i] {
		chevron = "▼"
	}
	if i < 0 { // read-manually pseudo-node
		return fmt.Sprintf("%s Read manually · %d", chevron, len(state.plan.ReadManually))
	}
	c := state.plan.Categories[i]
	checked := len(checkedIDs(c.MessageIDs, state.excluded))
	return fmt.Sprintf("%s %s · %d/%d · %s · %s", chevron, actionVerbLabel(c.Action), checked, len(c.MessageIDs), c.Name, strings.ToUpper(c.Priority))
}

// syncActionPlanNode refreshes state+footer for an already-current node (expand/collapse);
// unlike syncSelectionToNode it does NOT move the cursor — the caller owns that.
// It refreshes a top-level node after an expand/collapse: it updates the chevron in the
// label in place (no full rebuild) and re-syncs the selection/footer to this node, so the
// footer can't keep showing email-level hints on a category header.
func (a *App) syncActionPlanNode(state *actionPlanState, node *tview.TreeNode, idx int) {
	node.SetText(a.topLevelNodeLabel(state, idx))
	state.selectedCategory = idx
	state.selectedMsgID = ""
	a.updateActionPlanFooter(state)
}

// syncSelectionToNode makes node the current node AND derives the selection state
// (selectedCategory/selectedMsgID) from its reference, then refreshes the footer.
// SetCurrentNode does NOT fire SetChangedFunc, so callers that relocate the cursor
// programmatically (e.g. rebuildActionPlanTree) must use this to keep state, cursor,
// and footer in lockstep — the node reference is the single source of truth.
func (a *App) syncSelectionToNode(state *actionPlanState, node *tview.TreeNode) {
	if state == nil || state.tree == nil || node == nil {
		return
	}
	state.tree.SetCurrentNode(node)
	switch ref := node.GetReference().(type) {
	case int:
		state.selectedCategory = ref
		state.selectedMsgID = ""
	case emailRef:
		state.selectedCategory = ref.catIndex
		state.selectedMsgID = ref.msgID
	}
	a.updateActionPlanFooter(state)
}

// rebuildActionPlanTree repopulates the tree from state.plan, preserving the
// selected node (category or email). Categories are root nodes; email children
// are nested under their category and shown when the category is expanded.
func (a *App) rebuildActionPlanTree(state *actionPlanState) {
	if state.plan == nil || state.root == nil {
		return
	}
	colors := a.GetComponentColors("ai")
	state.root.ClearChildren()
	for i, c := range state.plan.Categories {
		node := tview.NewTreeNode(a.topLevelNodeLabel(state, i)).SetSelectable(true).SetColor(colors.Text.Color())
		node.SetReference(i) // category index
		for _, id := range c.MessageIDs {
			box := "☑" // same glyph the inbox uses for bulk selection
			if state.excluded[id] {
				box = "☐"
			}
			subj, from := "(unknown)", ""
			if m := state.metaByID[id]; m != nil {
				subj = extractHeaderValue(m, "Subject")
				from = extractHeaderValue(m, "From")
			}
			child := tview.NewTreeNode(fmt.Sprintf("%s %s — %s", box, subj, from)).
				SetSelectable(true).
				SetColor(colors.Text.Color())
			child.SetReference(emailRef{catIndex: i, msgID: id})
			node.AddChild(child)
		}
		node.SetExpanded(state.expanded[i]) // default collapsed (zero value false)
		state.root.AddChild(node)
	}
	// Read-manually pseudo-node (ref -1): messages the LLM declined to categorize. Shown
	// so nothing the user selected silently disappears. Non-actionable (no checkboxes).
	if len(state.plan.ReadManually) > 0 {
		const rmIdx = -1
		rm := tview.NewTreeNode(a.topLevelNodeLabel(state, rmIdx)).
			SetSelectable(true).SetColor(colors.Text.Color())
		rm.SetReference(rmIdx)
		for _, m := range state.plan.ReadManually {
			child := tview.NewTreeNode(fmt.Sprintf("• %s — %s", m.Subject, m.From)).
				SetSelectable(true).SetColor(colors.Text.Color())
			child.SetReference(emailRef{catIndex: rmIdx, msgID: m.ID})
			rm.AddChild(child)
		}
		rm.SetExpanded(state.expanded[rmIdx])
		state.root.AddChild(rm)
	}
	children := state.root.GetChildren()
	if len(children) == 0 {
		state.tree.SetCurrentNode(state.root)
		state.selectedMsgID = ""
		a.updateActionPlanFooter(state)
		return
	}
	// Restore an email-node selection if one was active and still present/visible.
	if state.selectedMsgID != "" {
		for _, parent := range children {
			if !parent.IsExpanded() {
				continue
			}
			for _, child := range parent.GetChildren() {
				if ref, ok := child.GetReference().(emailRef); ok && ref.msgID == state.selectedMsgID {
					a.syncSelectionToNode(state, child)
					return
				}
			}
		}
		// fall through to top-level selection if the email node is no longer visible
	}
	// Restore the top-level node whose int ref matches selectedCategory (categories are
	// 0..n-1; the read-manually node is -1). Falls back to the first node.
	for _, n := range children {
		if ref, ok := n.GetReference().(int); ok && ref == state.selectedCategory {
			a.syncSelectionToNode(state, n)
			return
		}
	}
	a.syncSelectionToNode(state, children[0])
}

// actionPlanFooterKeys holds the configurable bindings advertised in the footer, so the hints
// always reflect the user's config (not hardcoded letters).
type actionPlanFooterKeys struct {
	viewPrompt, remember, move, skip string
}

// prettyKeyLabel renders a config binding for display in footers (e.g. "ctrl+r" → "Ctrl+R",
// "shift+t" → "Shift+T", "space" → "Space").
func prettyKeyLabel(k string) string {
	switch {
	case strings.HasPrefix(k, "ctrl+"):
		return "Ctrl+" + strings.ToUpper(k[len("ctrl+"):])
	case strings.HasPrefix(k, "shift+"):
		return "Shift+" + strings.ToUpper(k[len("shift+"):])
	case k == "space":
		return "Space"
	case k == "":
		return "?"
	default:
		return k
	}
}

// actionPlanFooterText builds the context-aware footer, styled like the other pickers
// (" X to Y  |  … "). onCategory=true means a category (or the read-manually node) is
// highlighted; key/verb/count describe its suggested action. The keys arg supplies the
// configured bindings so hints track config (e.g. view-prompt is "i", not a hardcoded "v").
func actionPlanFooterText(onCategory bool, key, action string, checkedCount int, keys actionPlanFooterKeys) string {
	if onCategory {
		parts := make([]string, 0, 6)
		if key != "" && action != "none" && action != "" {
			parts = append(parts, fmt.Sprintf("%s to %s (%d)", key, actionRuleVerbShort(action), checkedCount))
		}
		parts = append(parts,
			"Enter to expand",
			fmt.Sprintf("%s prompt", prettyKeyLabel(keys.viewPrompt)),
			fmt.Sprintf("%s to remember", prettyKeyLabel(keys.remember)),
			"Tab to inbox", "Esc to close")
		return " " + strings.Join(parts, "  |  ") + " "
	}
	return fmt.Sprintf(" %s to skip  |  %s to move  |  %s prompt  |  %s to remember sender  |  Tab to inbox  |  Esc to close ",
		prettyKeyLabel(keys.skip), prettyKeyLabel(keys.move), prettyKeyLabel(keys.viewPrompt), prettyKeyLabel(keys.remember))
}

// actionRuleVerbShort is the short imperative verb for the footer.
func actionRuleVerbShort(action string) string {
	switch action {
	case "archive":
		return "archive"
	case "mark_read":
		return "read"
	case "trash":
		return "trash"
	case "label":
		return "label"
	case "summarize":
		return "digest"
	default:
		return "act"
	}
}

// updateActionPlanFooter updates the footer text based on current navigation state.
func (a *App) updateActionPlanFooter(state *actionPlanState) {
	if state == nil || state.footer == nil {
		return
	}
	// Decide category-vs-email from selectedMsgID (set authoritatively from the changed
	// node), NOT GetCurrentNode() — the latter lagged a step behind and desynced the footer.
	onCategory := state.selectedMsgID == ""
	cat := a.currentActionPlanCategory(state)
	key, action, count := "", "none", 0
	if cat != nil {
		action = cat.Action
		key = a.actionKeyHint(cat.Action)
		count = len(checkedIDs(cat.MessageIDs, state.excluded))
	}
	state.footer.SetText(actionPlanFooterText(onCategory, key, action, count, actionPlanFooterKeys{
		viewPrompt: a.Keys.ViewPrompt,
		remember:   a.Keys.RememberRule,
		move:       a.Keys.Move,
		skip:       a.Keys.BulkSelect,
	}))
}

// closeActionPlanPanel closes the panel and restores the list view. Synchronous — no
// QueueUpdateDraw (CLAUDE.md ESC rule).
func (a *App) closeActionPlanPanel() {
	if a.actionPlanState != nil && a.actionPlanState.streamingCancel != nil {
		a.actionPlanState.streamingCancel()
		a.actionPlanState.streamingCancel = nil
	}
	a.streamingCancel = nil

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.ResizeItem(a.labelsView, 0, 0)
		}
	}

	a.setActivePicker(PickerNone)
	a.actionPlanState = nil

	if list, ok := a.views["list"].(*tview.Table); ok {
		a.SetFocus(list)
	}
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// focusInboxFromActionPlan moves focus to the message list while leaving the Action Plan
// panel mounted and its analysis running, so the user can read mail mid-analysis. Tab
// (handled in keys.go) returns focus to the panel; Esc on the inbox does not close it.
func (a *App) focusInboxFromActionPlan() {
	if list, ok := a.views["list"].(*tview.Table); ok {
		a.SetFocus(list)
	}
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// focusActionPlanFromInbox returns focus to the Action Plan panel tree.
func (a *App) focusActionPlanFromInbox() {
	if a.actionPlanState == nil || a.actionPlanState.tree == nil {
		return
	}
	a.SetFocus(a.actionPlanState.tree)
	a.currentFocus = "action_plan"
	a.updateFocusIndicators("action_plan")
}

// actionPlanInputCapture handles all key input while the Action Plan panel is focused.
func (a *App) actionPlanInputCapture(state *actionPlanState) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		// ESC: synchronous close (no QueueUpdateDraw).
		if ev.Key() == tcell.KeyEscape {
			a.closeActionPlanPanel()
			return nil
		}

		cur := state.tree.GetCurrentNode()

		// Navigation works during and after analysis.
		switch ev.Key() {
		case tcell.KeyUp, tcell.KeyDown:
			return ev // let TreeView move the cursor natively
		case tcell.KeyEnter, tcell.KeyRight:
			if cur != nil {
				if idx, ok := cur.GetReference().(int); ok { // category / read-manually node
					state.expanded[idx] = !state.expanded[idx]
					cur.SetExpanded(state.expanded[idx])
					a.syncActionPlanNode(state, cur, idx) // refresh chevron + footer
				}
			}
			return nil
		case tcell.KeyLeft:
			if cur != nil {
				if idx, ok := cur.GetReference().(int); ok {
					state.expanded[idx] = false
					cur.SetExpanded(false)
					a.syncActionPlanNode(state, cur, idx)
				}
			}
			return nil
		}

		if a.matchesConfiguredKey(ev, a.Keys.RememberRule) {
			from, action, negate := "", "none", false
			if cat := a.currentActionPlanCategory(state); cat != nil {
				action = cat.Action
				if len(cat.MessageIDs) > 0 {
					if m := state.metaByID[cat.MessageIDs[0]]; m != nil {
						from = extractHeaderValue(m, "From")
					}
				}
			}
			if cur != nil {
				if ref, ok := cur.GetReference().(emailRef); ok {
					if m := state.metaByID[ref.msgID]; m != nil {
						from = extractHeaderValue(m, "From")
					}
					negate = true // on a specific email, default to a prohibition
				}
			}
			suggestion := ""
			// Only pre-seed a suggestion when we have a sender to anchor it to;
			// otherwise (e.g. Ctrl+R before the first batch arrives) open a blank
			// input rather than a meaningless "Always review emails from ".
			if from != "" {
				if svc := a.GetAnalyzerRulesService(); svc != nil {
					suggestion = svc.SuggestRuleFromContext(from, action, negate)
				}
			}
			a.showRememberRuleModal(suggestion)
			return nil
		}

		key := string(ev.Rune())

		// Toggle the excluded state of an email node (configurable; reuses bulk_select, default "space").
		if a.matchesConfiguredKey(ev, a.Keys.BulkSelect) {
			if cur != nil {
				if ref, ok := cur.GetReference().(emailRef); ok {
					state.excluded[ref.msgID] = !state.excluded[ref.msgID]
					a.renderActionPlanPanel(state) // re-render [x]/[ ] + counts; selection restored via selectedMsgID
				}
			}
			return nil
		}

		// Quick-actions are blocked until analysis finishes (avoids racing the plan).
		if state.analyzing.Load() {
			return nil
		}
		// 'm' moves: on an email node, that one email; on a category or read-manually header,
		// the whole group.
		if a.matchesConfiguredKey(ev, a.Keys.Move) && cur != nil {
			switch ref := cur.GetReference().(type) {
			case emailRef:
				a.showActionPlanMoveInline(state, ref.catIndex, ref.msgID)
				return nil
			case int:
				a.showActionPlanBulkMoveInline(state, ref)
				return nil
			}
		}
		// Open the effective-prompt viewer (configurable; default "i"). Blocked during analysis.
		if a.matchesConfiguredKey(ev, a.Keys.ViewPrompt) {
			a.showActionPlanPromptView(state)
			return nil
		}
		switch key {
		case a.Keys.Archive:
			a.executeActionPlanAction(state, "archive")
			return nil
		case a.Keys.ToggleRead:
			a.executeActionPlanAction(state, "mark_read")
			return nil
		case a.Keys.Trash:
			a.executeActionPlanAction(state, "trash")
			return nil
		case a.Keys.ManageLabels:
			a.executeActionPlanAction(state, "label")
			return nil
		case a.Keys.Summarize:
			a.dispatchActionPlanSummarize(state)
			return nil
		}
		return ev
	}
}

// userLabelNames returns the names of the user's own labels (excluding Gmail system labels),
// for feeding the analyzer so it prefers existing labels. Errors degrade to an empty slice.
func (a *App) userLabelNames() []string {
	_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
	if labelService == nil {
		return nil
	}
	labels, err := labelService.ListLabels(a.ctx)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		if l != nil && l.Type == "user" {
			out = append(out, l.Name)
		}
	}
	return out
}

// currentActionPlanCategory returns the selected category or nil.
func (a *App) currentActionPlanCategory(state *actionPlanState) *services.ActionPlanCategory {
	if state.plan == nil || state.selectedCategory < 0 || state.selectedCategory >= len(state.plan.Categories) {
		return nil
	}
	return &state.plan.Categories[state.selectedCategory]
}

// executeActionPlanAction runs a bulk action on the selected category's messages.
func (a *App) executeActionPlanAction(state *actionPlanState, action string) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	ids := checkedIDs(cat.MessageIDs, state.excluded)
	if len(ids) == 0 {
		// go: ShowWarning wraps QueueUpdateDraw; calling it synchronously here (this runs on the
		// UI goroutine from the key handler) would deadlock and hang the app.
		go a.GetErrorHandler().ShowWarning(a.ctx, "All emails in this category are excluded — nothing to do")
		return
	}
	catName := cat.Name
	label := cat.Label

	emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

	go func() {
		var err error
		switch action {
		case "archive":
			err = emailService.BulkArchive(a.ctx, ids, a.bulkProgress(a.ctx, "Archiving"))
		case "mark_read":
			err = emailService.BulkMarkAsRead(a.ctx, ids, a.bulkProgress(a.ctx, "Marking read"))
		case "trash":
			err = emailService.BulkTrash(a.ctx, ids, a.bulkProgress(a.ctx, "Trashing"))
		case "label":
			if label == "" {
				a.GetErrorHandler().ShowWarning(a.ctx, "Category has no label to apply")
				return
			}
			err = a.applyActionPlanLabel(labelService, ids, label)
		default:
			return
		}
		a.GetErrorHandler().ClearPersistentMessage()
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Action failed on %q: %v", catName, err))
			return
		}
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("✓ %s applied to %d messages (%s)", actionVerbLabel(action), len(ids), catName))
		a.QueueUpdateDraw(func() {
			if a.actionPlanState == state {
				a.removeActionPlanCategory(state, catName)
			}
		})
	}()
}

// applyActionPlanLabel resolves a label name to an ID (creating it if needed) and applies
// it to the messages in bulk.
func (a *App) applyActionPlanLabel(labelService services.LabelService, ids []string, labelName string) error {
	labelID, err := a.resolveOrCreateLabelID(labelService, labelName)
	if err != nil {
		return err
	}
	return labelService.BulkApplyLabel(a.ctx, ids, labelID, a.bulkProgress(a.ctx, "Applying label"))
}

// resolveOrCreateLabelID finds a label by name (case-insensitive) or creates it.
func (a *App) resolveOrCreateLabelID(labelService services.LabelService, name string) (string, error) {
	labels, err := labelService.ListLabels(a.ctx)
	if err != nil {
		return "", err
	}
	for _, l := range labels {
		if strings.EqualFold(l.Name, name) {
			return l.Id, nil
		}
	}
	created, err := labelService.CreateLabel(a.ctx, name)
	if err != nil {
		return "", err
	}
	return created.Id, nil
}

// removeActionPlanCategory drops a completed category and re-renders.
func (a *App) removeActionPlanCategory(state *actionPlanState, name string) {
	if state.plan == nil {
		return
	}
	kept := state.plan.Categories[:0]
	for _, c := range state.plan.Categories {
		if c.Name != name {
			kept = append(kept, c)
		}
	}
	state.plan.Categories = kept
	a.renderActionPlanPanel(state)
}

// openActionPlanWithPrompt opens the panel using a saved prompt (by name or numeric id)
// as the analyzer override. Falls back to the default prompt if the prompt is not found.
func (a *App) openActionPlanWithPrompt(nameOrID string) {
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Prompt library unavailable — using default analyzer prompt")
		a.openActionPlanWithText("")
		return
	}

	var tmpl *services.PromptTemplate
	var err error
	if id, convErr := strconv.Atoi(nameOrID); convErr == nil {
		tmpl, err = promptService.GetPrompt(a.ctx, id)
	} else {
		tmpl, err = promptService.FindPromptByName(a.ctx, nameOrID)
	}
	if err != nil || tmpl == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("⚠ Prompt %q not found. Using default analyzer prompt.", nameOrID))
		a.openActionPlanWithText("")
		return
	}
	a.openActionPlanWithText(tmpl.PromptText)
}
