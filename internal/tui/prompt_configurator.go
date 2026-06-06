package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// promptConfiguratorContext describes what messages the configurator will act upon when Apply is pressed.
type promptConfiguratorContext struct {
	// mode is "single", "bulk", or "draft" (no context, draft-only).
	mode string
	// messageID is set when mode == "single".
	messageID string
	// messageIDs is set when mode == "bulk".
	messageIDs []string
	// categoryName, if non-empty, indicates the context came from an action plan category.
	categoryName string
}

// promptConfiguratorState holds the mutable state of the configurator panel.
type promptConfiguratorState struct {
	ctx             promptConfiguratorContext
	currentPrompt   string
	suggestedName   string
	suggestedDesc   string
	detectedMode    string
	intentInput     *tview.InputField
	promptArea      *EditableTextView
	refineInput     *tview.InputField
	statusLine      *tview.TextView
	container       *tview.Flex
	streamingCancel context.CancelFunc
}

// openPromptConfigurator opens the configurator panel with the given context.
func (a *App) openPromptConfigurator(pctx promptConfiguratorContext) {
	if a.logger != nil {
		a.logger.Printf("openPromptConfigurator: mode=%s msgCount=%d", pctx.mode, len(pctx.messageIDs))
	}

	// Defensive: if a previous configurator is still mounted, close it first
	// to avoid leaking its streaming context.
	if a.promptConfiguratorState != nil {
		a.closePromptConfigurator()
	}

	if a.GetPromptGeneratorService() == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt generator service not available - check LLM configuration")
		return
	}

	state := &promptConfiguratorState{ctx: pctx}

	colors := a.GetComponentColors("prompts")
	bgColor := colors.Background.Color()

	// Intent input
	state.intentInput = tview.NewInputField().
		SetLabel("Intent: ").
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())
	state.intentInput.SetBackgroundColor(bgColor)
	state.intentInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.closePromptConfigurator()
			return
		}
		if key == tcell.KeyEnter {
			intent := state.intentInput.GetText()
			if intent == "" {
				return
			}
			go a.generateConfiguratorPrompt(intent)
		}
	})

	// Editable prompt area — uses the project's EditableTextView (derailed/tview has no TextArea).
	state.promptArea = NewEditableTextView(a).
		SetPlaceholder("Generated prompt will appear here. Edit freely.").
		SetBackgroundColor(bgColor).
		SetTextColor(colors.Text.Color()).
		SetBorder(true).
		SetTitle(" 📝 Editable prompt ").
		SetTitleColor(colors.Title.Color())

	state.promptArea.SetKeyHandler(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			// Regenerate using whatever intent the user last typed.
			intent := state.intentInput.GetText()
			if intent != "" {
				go a.generateConfiguratorPrompt(intent)
			}
			return nil
		case tcell.KeyCtrlS:
			// Task 14 will implement savePromptFromConfigurator
			a.GetErrorHandler().ShowInfo(a.ctx, "Save: not yet implemented (Task 14)")
			return nil
		case tcell.KeyCtrlG:
			go a.applyConfiguratorPrompt()
			return nil
		case tcell.KeyEscape:
			a.closePromptConfigurator()
			return nil
		}
		return event
	})

	// Refine input
	state.refineInput = tview.NewInputField().
		SetLabel("Refine: ").
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())
	state.refineInput.SetBackgroundColor(bgColor)
	state.refineInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.closePromptConfigurator()
			return
		}
		if key == tcell.KeyEnter {
			refinement := state.refineInput.GetText()
			if refinement == "" {
				return
			}
			// Use whatever is currently in the editable box as the source.
			current := state.promptArea.GetText()
			go a.refineConfiguratorPrompt(current, refinement)
		}
	})

	// Status line
	state.statusLine = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetText(" Ctrl+G apply  Ctrl+R refine  Ctrl+S save  Esc cancel ")
	state.statusLine.SetTextColor(colors.Text.Color())
	state.statusLine.SetBackgroundColor(bgColor)

	// Container
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.SetBackgroundColor(bgColor)
	state.container.SetBorder(true)
	state.container.SetTitle(promptConfiguratorTitle(pctx))
	state.container.SetTitleColor(colors.Title.Color())
	state.container.AddItem(state.intentInput, 1, 0, true)
	state.container.AddItem(state.promptArea, 0, 1, false)
	state.container.AddItem(state.refineInput, 1, 0, false)
	state.container.AddItem(state.statusLine, 1, 0, false)

	a.promptConfiguratorState = state

	// Attach to the content split — same pattern as openPromptPicker / openBulkPromptPicker.
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = state.container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}

	a.SetFocus(state.intentInput)
	a.currentFocus = "prompt_configurator"
	a.updateFocusIndicators("prompt_configurator")
	a.setActivePicker(PickerPromptConfigurator)
}

// closePromptConfigurator closes the configurator and restores the original view.
// Synchronous cleanup — NEVER use QueueUpdateDraw in close paths (CLAUDE.md rule).
func (a *App) closePromptConfigurator() {
	if a.promptConfiguratorState != nil && a.promptConfiguratorState.streamingCancel != nil {
		a.promptConfiguratorState.streamingCancel()
		a.promptConfiguratorState.streamingCancel = nil
	}

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.ResizeItem(a.labelsView, 0, 0)
		}
	}

	a.setActivePicker(PickerNone)
	a.promptConfiguratorState = nil

	if list, ok := a.views["list"].(*tview.Table); ok {
		a.SetFocus(list)
	}
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// generateConfiguratorPrompt invokes the LLM streaming to fill the editable prompt area.
func (a *App) generateConfiguratorPrompt(intent string) {
	state := a.promptConfiguratorState
	if state == nil {
		return
	}

	gen := a.GetPromptGeneratorService()
	if gen == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt generator service not available")
		return
	}

	a.GetErrorHandler().ShowProgress(a.ctx, "Generating prompt...")
	defer a.GetErrorHandler().ClearProgress() // I5: clear progress in ALL exit paths

	// Clear and show loading
	a.QueueUpdateDraw(func() {
		if state.promptArea != nil {
			state.promptArea.SetText("Generating...")
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	// C1: register cancel BOTH at state level (used by closePromptConfigurator)
	// AND at app level (used by the global ESC handler in keys.go).
	state.streamingCancel = cancel
	a.streamingCancel = cancel
	defer func() {
		cancel()
		state.streamingCancel = nil
		a.streamingCancel = nil
	}()

	var accumulator string

	result, err := gen.GenerateFromIntentStream(ctx, intent, services.PromptGenerationOptions{
		TargetMode: state.ctx.mode,
	}, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		accumulator += token
		// Direct UI update — CLAUDE.md prohibits QueueUpdateDraw inside streaming callbacks.
		// I1: use captured `state` (not a.promptConfiguratorState) to avoid writing
		// into a different panel instance if the user closed/reopened mid-stream.
		if ctx.Err() == nil && state.promptArea != nil {
			state.promptArea.SetText(accumulator)
		}
	})

	if err != nil {
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Prompt generation canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to generate prompt: %v", err))
		return
	}

	// Final update with the parsed PromptText.
	// I1: use captured `state` here too.
	a.QueueUpdateDraw(func() {
		state.currentPrompt = result.PromptText
		state.suggestedName = result.SuggestedName
		state.suggestedDesc = result.SuggestedDesc
		state.detectedMode = result.DetectedMode
		if state.promptArea != nil {
			state.promptArea.SetText(result.PromptText)
		}
	})

	a.GetErrorHandler().ShowSuccess(a.ctx, "Prompt generated. Edit or refine before applying.")
}

// refineConfiguratorPrompt invokes the LLM streaming to refine the current prompt.
func (a *App) refineConfiguratorPrompt(currentPrompt string, refinement string) {
	state := a.promptConfiguratorState
	if state == nil {
		return
	}

	gen := a.GetPromptGeneratorService()
	if gen == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt generator service not available")
		return
	}

	a.GetErrorHandler().ShowProgress(a.ctx, "Refining prompt...")
	defer a.GetErrorHandler().ClearProgress() // I5: clear progress in ALL exit paths

	// Show loading while preserving the previous prompt as fallback if cancelled.
	previous := currentPrompt
	a.QueueUpdateDraw(func() {
		if state.promptArea != nil {
			state.promptArea.SetText("Refining...")
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	// C1: register cancel BOTH at state level (used by closePromptConfigurator)
	// AND at app level (used by the global ESC handler in keys.go).
	state.streamingCancel = cancel
	a.streamingCancel = cancel
	defer func() {
		cancel()
		state.streamingCancel = nil
		a.streamingCancel = nil
	}()

	var accumulator string

	result, err := gen.RefinePromptStream(ctx, currentPrompt, refinement, services.PromptGenerationOptions{
		TargetMode: state.ctx.mode,
	}, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		accumulator += token
		// Direct UI update — CLAUDE.md prohibits QueueUpdateDraw inside streaming callbacks.
		// I1: use captured `state` (not a.promptConfiguratorState) to avoid writing
		// into a different panel instance if the user closed/reopened mid-stream.
		if ctx.Err() == nil && state.promptArea != nil {
			state.promptArea.SetText(accumulator)
		}
	})

	if err != nil {
		// Restore previous prompt on cancellation or failure.
		// I1: captured state.
		a.QueueUpdateDraw(func() {
			if state.promptArea != nil {
				state.promptArea.SetText(previous)
			}
		})
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Refinement canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to refine prompt: %v", err))
		return
	}

	// Final update with the parsed result.
	// I1: use captured `state`.
	a.QueueUpdateDraw(func() {
		state.currentPrompt = result.PromptText
		state.suggestedName = result.SuggestedName
		state.suggestedDesc = result.SuggestedDesc
		state.detectedMode = result.DetectedMode
		if state.promptArea != nil {
			state.promptArea.SetText(result.PromptText)
		}
		if state.refineInput != nil {
			state.refineInput.SetText("")
		}
	})

	a.GetErrorHandler().ShowSuccess(a.ctx, "Prompt refined.")
}

// applyConfiguratorPrompt runs the current prompt against the context defined when the panel was opened.
func (a *App) applyConfiguratorPrompt() {
	state := a.promptConfiguratorState
	if state == nil {
		return
	}

	current := state.promptArea.GetText()
	if current == "" {
		a.GetErrorHandler().ShowWarning(a.ctx, "Prompt is empty — generate or type one first")
		return
	}

	switch state.ctx.mode {
	case "single":
		if state.ctx.messageID == "" {
			a.GetErrorHandler().ShowWarning(a.ctx, "No message context — Apply disabled in draft mode")
			return
		}
		go a.applyEphemeralPromptToMessage(state.ctx.messageID, current, state.suggestedName)
	case "bulk":
		if len(state.ctx.messageIDs) == 0 {
			a.GetErrorHandler().ShowWarning(a.ctx, "No messages in bulk context — Apply disabled")
			return
		}
		go a.applyEphemeralPromptToBulk(state.ctx.messageIDs, current, state.suggestedName)
	default:
		a.GetErrorHandler().ShowWarning(a.ctx, "No message context — save the prompt first, then use it from the picker")
	}
}

// applyEphemeralPromptToMessage runs an unsaved prompt against a single message.
func (a *App) applyEphemeralPromptToMessage(messageID string, promptText string, displayName string) {
	a.closePromptConfigurator()

	_, aiService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
	if aiService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "AI service not available")
		return
	}

	message, err := a.Client.GetMessageWithContent(messageID)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Failed to load message content")
		return
	}

	content := message.PlainText
	if len([]rune(content)) > 8000 {
		content = string([]rune(content)[:8000])
	}

	name := displayName
	if name == "" {
		name = "Custom Prompt"
	}

	a.QueueUpdateDraw(func() {
		if !a.aiSummaryVisible {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.aiSummaryView, 0, 1)
			}
			a.aiSummaryVisible = true
		}
		if a.aiSummaryView != nil {
			a.aiPanelInPromptMode = true
			a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 %s ", name))
			a.aiSummaryView.SetText("🤖 Applying prompt...")
			a.aiSummaryView.ScrollToBeginning()
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	a.streamingCancel = cancel
	defer func() {
		cancel()
		a.streamingCancel = nil
	}()

	var b strings.Builder
	result, err := aiService.ApplyCustomPromptStream(ctx, content, promptText, nil, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		b.WriteString(token)
		if ctx.Err() == nil && a.aiSummaryView != nil {
			a.aiSummaryView.SetText(b.String())
			a.aiSummaryView.ScrollToEnd()
		}
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Apply canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply prompt: %v", err))
		return
	}

	a.QueueUpdateDraw(func() {
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetText(result)
			a.aiSummaryView.ScrollToBeginning()
		}
	})
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s", name))
}

// applyEphemeralPromptToBulk runs an unsaved prompt against a bulk selection.
func (a *App) applyEphemeralPromptToBulk(messageIDs []string, promptText string, displayName string) {
	a.closePromptConfigurator()

	_, aiService, _, _, repository, _, _, _, _, _, _, _ := a.GetServices()
	if aiService == nil || repository == nil {
		a.GetErrorHandler().ShowError(a.ctx, "AI or repository service not available")
		return
	}

	name := displayName
	if name == "" {
		name = "Custom Bulk Prompt"
	}

	// Build combined content from messages using PlainText (repository.GetMessage calls GetMessageWithContent).
	var combined strings.Builder
	combined.WriteString("---START EMAILS---\n")
	for i, id := range messageIDs {
		msg, err := repository.GetMessage(a.ctx, id)
		if err != nil || msg == nil {
			continue
		}
		fmt.Fprintf(&combined, "---START EMAIL %d---\n", i+1)
		content := msg.PlainText
		if len([]rune(content)) > 2000 {
			content = string([]rune(content)[:2000])
		}
		if content != "" {
			combined.WriteString(content)
		} else if msg.Snippet != "" {
			combined.WriteString(msg.Snippet)
		}
		fmt.Fprintf(&combined, "\n---END EMAIL %d---\n", i+1)
	}
	combined.WriteString("---END OF EMAILS---\n")

	// Substitute placeholders.
	finalPrompt := promptText
	finalPrompt = strings.ReplaceAll(finalPrompt, "{{messages}}", combined.String())
	finalPrompt = strings.ReplaceAll(finalPrompt, "{{body}}", combined.String())

	a.QueueUpdateDraw(func() {
		if !a.aiSummaryVisible {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.aiSummaryView, 0, 1)
			}
			a.aiSummaryVisible = true
		}
		if a.aiSummaryView != nil {
			a.aiPanelInPromptMode = true
			a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 %s (%d msgs) ", name, len(messageIDs)))
			a.aiSummaryView.SetText("🤖 Applying bulk prompt...")
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	a.streamingCancel = cancel
	defer func() {
		cancel()
		a.streamingCancel = nil
	}()

	var b strings.Builder
	result, err := aiService.ApplyCustomPromptStream(ctx, combined.String(), finalPrompt, nil, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		b.WriteString(token)
		if ctx.Err() == nil && a.aiSummaryView != nil {
			a.aiSummaryView.SetText(b.String())
			a.aiSummaryView.ScrollToEnd()
		}
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Bulk apply canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply bulk prompt: %v", err))
		return
	}

	a.QueueUpdateDraw(func() {
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetText(result)
			a.aiSummaryView.ScrollToBeginning()
		}
	})
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s (%d msgs)", name, len(messageIDs)))
}

// promptConfiguratorTitle returns the panel title appropriate for the context.
func promptConfiguratorTitle(pctx promptConfiguratorContext) string {
	switch pctx.mode {
	case "single":
		return " ✨ Prompt Configurator (1 msg scoped) "
	case "bulk":
		if pctx.categoryName != "" {
			return fmt.Sprintf(" ✨ Prompt Configurator (%d msgs from %q) ", len(pctx.messageIDs), pctx.categoryName)
		}
		return fmt.Sprintf(" ✨ Prompt Configurator (%d msgs scoped) ", len(pctx.messageIDs))
	default:
		return " ✨ Prompt Configurator (draft only) "
	}
}
