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

	header          *tview.TextView
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

	state.header = tview.NewTextView().SetDynamicColors(true)
	state.header.SetBackgroundColor(bg)
	state.header.SetTextColor(colors.Text.Color())

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
	})

	state.footer = tview.NewTextView().SetDynamicColors(true)
	state.footer.SetBackgroundColor(bg)
	state.footer.SetTextColor(colors.Text.Color())
	state.footer.SetText("[↑↓] navigate  [Enter] action  [:] palette  [p] configurator  [Esc] close")

	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.SetBackgroundColor(bg)
	state.container.SetBorder(true)
	state.container.SetTitle("📋 Action Plan")
	state.container.SetTitleColor(colors.Title.Color())
	state.container.SetBorderColor(colors.Border.Color())
	state.container.AddItem(state.header, 1, 0, false)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)

	a.actionPlanState = state

	// Mount in the shared right-panel slot (same pattern as openPromptConfigurator).
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = state.container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}

	state.tree.SetInputCapture(a.actionPlanInputCapture(state))
	a.SetFocus(state.tree)
	a.currentFocus = "action_plan"
	a.updateFocusIndicators("action_plan")
	a.setActivePicker(PickerActionPlan)

	// Launch analysis in the background. ctx cancel is registered both on the state
	// (for closeActionPlanPanel) and on the App (for the global ESC handler in keys.go).
	ctx, cancel := context.WithCancel(a.ctx)
	state.streamingCancel = cancel
	a.streamingCancel = cancel

	batchSize := a.Config.InboxAnalyzer.BatchSize
	maxBatches := a.Config.InboxAnalyzer.MaxBatches

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
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText},
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

// renderActionPlanPanel refreshes header + body from the current state. It performs raw
// SetText calls and so must run on the UI thread — callers from the analysis goroutine
// wrap it in QueueUpdateDraw; UI-thread callers (key handlers) may call it directly.
func (a *App) renderActionPlanPanel(state *actionPlanState) {
	if state == nil || state.plan == nil {
		return
	}
	p := state.plan
	status := "analyzing"
	if !state.analyzing.Load() {
		status = "done"
	}
	state.header.SetText(fmt.Sprintf("[::b]Action Plan · %s • batch %d/%d • %s[::-]", state.scopeLabel, p.BatchesDone, p.BatchesTotal, status))
	if state.selectedCategory >= len(p.Categories) {
		state.selectedCategory = len(p.Categories) - 1
	}
	if state.selectedCategory < 0 {
		state.selectedCategory = 0
	}
	a.rebuildActionPlanTree(state)
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
		checked := len(checkedIDs(c.MessageIDs, state.excluded))
		label := fmt.Sprintf("%s · %d/%d · %s · %s", actionVerbLabel(c.Action), checked, len(c.MessageIDs), c.Name, strings.ToUpper(c.Priority))
		node := tview.NewTreeNode(label).SetSelectable(true).SetColor(colors.Text.Color())
		node.SetReference(i) // category index
		for _, id := range c.MessageIDs {
			box := "[x]"
			if state.excluded[id] {
				box = "[ ]"
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
	children := state.root.GetChildren()
	if len(children) == 0 {
		state.tree.SetCurrentNode(state.root)
		return
	}
	// Restore an email-node selection if one was active and still present/visible.
	if state.selectedMsgID != "" {
		for _, cat := range children {
			if !cat.IsExpanded() {
				continue
			}
			for _, child := range cat.GetChildren() {
				if ref, ok := child.GetReference().(emailRef); ok && ref.msgID == state.selectedMsgID {
					state.tree.SetCurrentNode(child)
					return
				}
			}
		}
		// fall through to category selection if the email node is no longer visible
	}
	if state.selectedCategory < 0 {
		state.selectedCategory = 0
	}
	if state.selectedCategory >= len(children) {
		state.selectedCategory = len(children) - 1
	}
	state.tree.SetCurrentNode(children[state.selectedCategory])
}

// updateActionPlanFooter updates the footer text based on current state.
// Task 9 will implement this fully; for now it is a no-op stub.
func (a *App) updateActionPlanFooter(state *actionPlanState) {}

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
				if idx, ok := cur.GetReference().(int); ok { // category node
					state.expanded[idx] = !state.expanded[idx]
					cur.SetExpanded(state.expanded[idx])
				}
			}
			return nil
		case tcell.KeyLeft:
			if cur != nil {
				if idx, ok := cur.GetReference().(int); ok {
					state.expanded[idx] = false
					cur.SetExpanded(false)
				}
			}
			return nil
		}

		key := string(ev.Rune())

		// Space toggles the excluded state of an email node.
		if ev.Rune() == ' ' {
			if cur != nil {
				if ref, ok := cur.GetReference().(emailRef); ok {
					state.excluded[ref.msgID] = !state.excluded[ref.msgID]
					a.renderActionPlanPanel(state) // re-render [x]/[ ] + counts; selection restored via selectedMsgID
				}
			}
			return nil
		}

		// Escape hatches (available any time).
		if key == a.Keys.CommandMode { // ':'
			a.actionPlanOpenPalette(state)
			return nil
		}
		if key == a.Keys.Prompt { // 'p'
			a.actionPlanOpenConfigurator(state)
			return nil
		}

		// Quick-actions are blocked until analysis finishes (avoids racing the plan).
		if state.analyzing.Load() {
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
		}
		return ev
	}
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
	ids := make([]string, len(cat.MessageIDs))
	copy(ids, cat.MessageIDs)
	catName := cat.Name
	label := cat.Label

	emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

	go func() {
		var err error
		switch action {
		case "archive":
			err = emailService.BulkArchive(a.ctx, ids)
		case "mark_read":
			err = emailService.BulkMarkAsRead(a.ctx, ids)
		case "trash":
			err = emailService.BulkTrash(a.ctx, ids)
		case "label":
			if label == "" {
				a.GetErrorHandler().ShowWarning(a.ctx, "Category has no label to apply")
				return
			}
			err = a.applyActionPlanLabel(labelService, ids, label)
		default:
			return
		}
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
	return labelService.BulkApplyLabel(a.ctx, ids, labelID)
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

// actionPlanOpenPalette sets a virtual bulk selection over the category's messages and
// opens the command palette (the ':' escape hatch).
func (a *App) actionPlanOpenPalette(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	ids := make([]string, len(cat.MessageIDs))
	copy(ids, cat.MessageIDs)
	a.setVirtualBulkSelection(ids)
	a.closeActionPlanPanel()
	a.showCommandBar()
}

// actionPlanOpenConfigurator opens the bulk prompt picker scoped to the category (the
// 'p' escape hatch).
func (a *App) actionPlanOpenConfigurator(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	ids := make([]string, len(cat.MessageIDs))
	copy(ids, cat.MessageIDs)
	a.setVirtualBulkSelection(ids)
	a.closeActionPlanPanel()
	go a.openBulkPromptPicker()
}

// setVirtualBulkSelection marks the given IDs as selected and enables bulk mode so the
// existing command palette / bulk picker operate on exactly these messages.
func (a *App) setVirtualBulkSelection(ids []string) {
	a.mu.Lock()
	if a.selected == nil {
		a.selected = make(map[string]bool)
	} else {
		for k := range a.selected {
			delete(a.selected, k)
		}
	}
	for _, id := range ids {
		a.selected[id] = true
	}
	a.bulkMode = true
	a.mu.Unlock()
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
