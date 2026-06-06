package tui

import (
	"context"
	"fmt"

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

	// Editable prompt area — uses the project's EditableTextView (derailed/tview has no TextArea).
	state.promptArea = NewEditableTextView(a).
		SetPlaceholder("Generated prompt will appear here. Edit freely.").
		SetBackgroundColor(bgColor).
		SetTextColor(colors.Text.Color()).
		SetBorder(true).
		SetTitle(" Editable prompt ").
		SetTitleColor(colors.Title.Color())

	// Refine input
	state.refineInput = tview.NewInputField().
		SetLabel("Refine: ").
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())
	state.refineInput.SetBackgroundColor(bgColor)

	// Status line
	state.statusLine = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetText(fmt.Sprintf(" %s apply  %s refine  %s save  Esc cancel ",
			a.Keys.PromptApply, a.Keys.PromptRegenerate, a.Keys.SavePrompt))
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
	a.currentFocus = "prompts"
	a.updateFocusIndicators("prompts")
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

// promptConfiguratorTitle returns the panel title appropriate for the context.
func promptConfiguratorTitle(pctx promptConfiguratorContext) string {
	switch pctx.mode {
	case "single":
		return " Prompt Configurator (1 msg scoped) "
	case "bulk":
		if pctx.categoryName != "" {
			return fmt.Sprintf(" Prompt Configurator (%d msgs from %q) ", len(pctx.messageIDs), pctx.categoryName)
		}
		return fmt.Sprintf(" Prompt Configurator (%d msgs scoped) ", len(pctx.messageIDs))
	default:
		return " Prompt Configurator (draft only) "
	}
}
