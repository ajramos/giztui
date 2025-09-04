package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// CompositionPanel represents the email composition UI with proper layout and focus management
type CompositionPanel struct {
	*tview.Flex
	app *App

	// UI Components
	headerContainer *tview.Flex       // Container for header section with Form and toggle button
	headerSection   *tview.Form       // Form for email fields
	bodySection     *EditableTextView // EditableTextView for multiline text editing
	bodyContainer   *tview.Flex       // Container for body section with border
	buttonSection   *tview.Flex

	// Input fields
	toField      *tview.InputField
	ccField      *tview.InputField
	bccField     *tview.InputField
	subjectField *tview.InputField

	// Action buttons
	sendButton  *tview.Button
	draftButton *tview.Button
	ccBccToggle *tview.Button

	// Button section spacers (to apply theme colors)
	spacer1 *tview.Box // Left spacer to center buttons
	spacer2 *tview.Box // Between Send and Draft
	spacer3 *tview.Box // Right side push spacer

	// CC/BCC toggle container spacers
	toggleTopSpacer    *tview.Box // Top spacer around CC/BCC toggle
	toggleBottomSpacer *tview.Box // Bottom spacer around CC/BCC toggle

	// UI components for theming and focus
	hintTextView *tview.TextView
	buttonRow    *tview.Flex
	spacerLine   *tview.TextView

	// State management
	composition       *services.Composition
	isVisible         bool
	ccBccVisible      bool
	currentFocusIndex int
	focusableItems    []tview.Primitive

	// Auto-save functionality
	autoSaveTimer   *time.Timer
	autoSaveEnabled bool
	lastSaveContent string
}

// NewCompositionPanel creates a new composition panel with improved layout and focus management
func NewCompositionPanel(app *App) *CompositionPanel {
	panel := &CompositionPanel{
		Flex:              tview.NewFlex(),
		app:               app,
		ccBccVisible:      false,
		currentFocusIndex: 0,
		focusableItems:    make([]tview.Primitive, 0),
	}

	panel.createComponents()
	panel.setupLayout()
	panel.setupFocusManagement()
	panel.setupInputHandling()

	return panel
}

// createComponents initializes all UI components with proper theming
func (c *CompositionPanel) createComponents() {
	// Get composition-specific theme colors
	componentColors := c.app.GetComponentColors("compose")

	// Debug: Log the colors being retrieved
	if c.app.logger != nil {
		c.app.logger.Printf("Compose theme colors - Background: %s, Title: %s, Border: %s, Text: %s",
			componentColors.Background.String(),
			componentColors.Title.String(),
			componentColors.Border.String(),
			componentColors.Text.String())
	}

	// Create header section with input fields (no border - integrated into container)
	c.headerSection = tview.NewForm()
	c.headerSection.SetBackgroundColor(componentColors.Background.Color())

	// Set Form-level field styling with compose theme colors (applied first)
	c.headerSection.SetFieldBackgroundColor(componentColors.Background.Color()) // Dark slate blue
	c.headerSection.SetFieldTextColor(componentColors.Text.Color())             // Light blue-gray text
	c.headerSection.SetLabelColor(componentColors.Title.Color())                // Cyan labels (not yellow)
	c.headerSection.SetButtonBackgroundColor(componentColors.Border.Color())
	c.headerSection.SetButtonTextColor(componentColors.Text.Color())

	// Create individual input fields with proper theming and placeholders
	c.toField = tview.NewInputField()
	c.toField.SetLabel("To: ")
	c.toField.SetPlaceholder("recipient@example.com")
	c.toField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.toField.SetFieldTextColor(componentColors.Text.Color())
	c.toField.SetLabelColor(componentColors.Title.Color())
	c.toField.SetPlaceholderTextColor(c.app.getHintColor()) // Match Advanced Search placeholder color

	c.ccField = tview.NewInputField()
	c.ccField.SetLabel("CC: ")
	c.ccField.SetPlaceholder("cc@example.com")
	c.ccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.ccField.SetFieldTextColor(componentColors.Text.Color())
	c.ccField.SetLabelColor(componentColors.Title.Color())
	c.ccField.SetPlaceholderTextColor(c.app.getHintColor()) // Match Advanced Search placeholder color

	c.bccField = tview.NewInputField()
	c.bccField.SetLabel("BCC: ")
	c.bccField.SetPlaceholder("bcc@example.com")
	c.bccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.bccField.SetFieldTextColor(componentColors.Text.Color())
	c.bccField.SetLabelColor(componentColors.Title.Color())
	c.bccField.SetPlaceholderTextColor(c.app.getHintColor()) // Match Advanced Search placeholder color

	c.subjectField = tview.NewInputField()
	c.subjectField.SetLabel("Subject: ")
	c.subjectField.SetPlaceholder("Enter email subject")
	c.subjectField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.subjectField.SetFieldTextColor(componentColors.Text.Color())
	c.subjectField.SetLabelColor(componentColors.Title.Color())
	c.subjectField.SetPlaceholderTextColor(c.app.getHintColor()) // Match Advanced Search placeholder color

	// Create CC/BCC toggle button
	c.ccBccToggle = tview.NewButton("+CC/BCC")
	c.ccBccToggle.SetBackgroundColor(componentColors.Border.Color())
	c.ccBccToggle.SetLabelColor(componentColors.Text.Color())
	c.ccBccToggle.SetSelectedFunc(c.toggleCCBCC)

	// Create message body with EditableTextView for multiline text editing
	c.bodySection = NewEditableTextView(c.app)
	c.bodySection.SetBackgroundColor(componentColors.Background.Color())
	c.bodySection.SetTextColor(componentColors.Text.Color())
	c.bodySection.SetBorderColor(componentColors.Border.Color())
	c.bodySection.SetPlaceholder("Enter your message here...")
	c.bodySection.SetPlaceholderTextColor(c.app.getHintColor())

	// Create a container for the body section (no border to save space)
	c.bodyContainer = tview.NewFlex().SetDirection(tview.FlexRow)
	c.bodyContainer.SetBackgroundColor(componentColors.Background.Color())
	c.bodyContainer.SetBorder(false) // Remove border to gain space
	c.bodyContainer.AddItem(c.bodySection, 0, 1, false)

	// Create action buttons with modern emoji styling
	c.sendButton = tview.NewButton("📧 Send")
	c.sendButton.SetBackgroundColor(componentColors.Accent.Color()) // Green for Send button prominence
	c.sendButton.SetLabelColor(componentColors.Background.Color())  // Dark text on green background

	c.draftButton = tview.NewButton("💾 Save")
	c.draftButton.SetBackgroundColor(componentColors.Border.Color())
	c.draftButton.SetLabelColor(componentColors.Text.Color())

	// Cancel button removed - Esc key provides cancel functionality

	// Create button section spacers with theme colors
	c.spacer1 = tview.NewBox()
	c.spacer1.SetBackgroundColor(componentColors.Background.Color())

	c.spacer2 = tview.NewBox()
	c.spacer2.SetBackgroundColor(componentColors.Background.Color())

	c.spacer3 = tview.NewBox()
	c.spacer3.SetBackgroundColor(componentColors.Background.Color())

	// Create header container to hold Form and CC/BCC toggle (no border to save space)
	c.headerContainer = tview.NewFlex().SetDirection(tview.FlexColumn)
	c.headerContainer.SetBackgroundColor(componentColors.Background.Color())
	c.headerContainer.SetBorder(false) // Remove border to gain space

	// Create button section with proper styling (no border to save space)
	c.buttonSection = tview.NewFlex()
	c.buttonSection.SetDirection(tview.FlexColumn)
	c.buttonSection.SetBackgroundColor(componentColors.Background.Color())
	c.buttonSection.SetBorder(false) // Remove border to gain space

	// Note: Removed ForceFilledBorderFlex as it was causing black background issues
	// Theme colors should now apply correctly without interference
}

// setupLayout creates the improved three-section layout
func (c *CompositionPanel) setupLayout() {
	componentColors := c.app.GetComponentColors("compose")

	// Configure main container
	c.Flex.SetDirection(tview.FlexRow)
	c.Flex.SetBorder(true)
	c.Flex.SetTitle(" Compose Email ")
	c.Flex.SetBackgroundColor(componentColors.Background.Color())
	c.Flex.SetTitleColor(componentColors.Title.Color())
	c.Flex.SetBorderColor(componentColors.Border.Color())

	// Setup header container (form + toggle button)
	c.setupHeaderContainer()

	// Setup header section with form fields
	c.setupHeaderSection()

	// Setup button section with actions
	c.setupButtonSection()

	// Layout: Header (expanded) → Body (expand) → Buttons (fixed)
	// Initial sizing - will be updated by updateLayoutSizing() when CC/BCC is toggled
	c.Flex.AddItem(c.headerContainer, 0, 1, false) // Header container - compact without border
	c.Flex.AddItem(c.bodyContainer, 0, 4, false)   // Body container - most space
	c.Flex.AddItem(c.buttonSection, 3, 0, false)   // Button section - compact height
}

// setupHeaderContainer configures the composite header layout
func (c *CompositionPanel) setupHeaderContainer() {
	// Add the Form (email fields) - takes most of the space
	c.headerContainer.AddItem(c.headerSection, 0, 1, false)

	// Create a vertical container for the CC/BCC toggle button
	toggleContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	componentColors := c.app.GetComponentColors("compose")
	toggleContainer.SetBackgroundColor(componentColors.Background.Color())

	// Add some spacing and the toggle button
	c.toggleTopSpacer = tview.NewBox()
	c.toggleTopSpacer.SetBackgroundColor(componentColors.Background.Color())
	c.toggleBottomSpacer = tview.NewBox()
	c.toggleBottomSpacer.SetBackgroundColor(componentColors.Background.Color())

	toggleContainer.AddItem(c.toggleTopSpacer, 0, 1, false)    // Top spacer
	toggleContainer.AddItem(c.ccBccToggle, 1, 0, false)        // Button with fixed height
	toggleContainer.AddItem(c.toggleBottomSpacer, 0, 1, false) // Bottom spacer

	// Add the toggle container - small fixed width
	c.headerContainer.AddItem(toggleContainer, 12, 0, false) // Fixed width for button
}

// setupHeaderSection configures the header form with dynamic CC/BCC visibility
func (c *CompositionPanel) setupHeaderSection() {
	// Add To field (always visible)
	c.headerSection.AddFormItem(c.toField)

	// Add Subject field (always visible)
	c.headerSection.AddFormItem(c.subjectField)

	// Initially hide CC/BCC fields and set up visibility
	c.updateCCBCCVisibility()

	// IMPORTANT: Apply field styling AFTER fields are added to Form
	c.applyFieldStyling()
}

// applyFieldStyling applies theme colors to form fields AFTER they're added to the Form
func (c *CompositionPanel) applyFieldStyling() {
	componentColors := c.app.GetComponentColors("compose")

	// Re-apply Form-level styling (crucial for tview Forms)
	c.headerSection.SetFieldBackgroundColor(componentColors.Background.Color())
	c.headerSection.SetFieldTextColor(componentColors.Text.Color())
	c.headerSection.SetLabelColor(componentColors.Title.Color()) // This should make labels cyan

	// Re-apply individual field styling for extra emphasis
	c.toField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.toField.SetFieldTextColor(componentColors.Text.Color())
	c.toField.SetLabelColor(componentColors.Title.Color())
	c.toField.SetPlaceholderTextColor(c.app.getHintColor())

	c.subjectField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.subjectField.SetFieldTextColor(componentColors.Text.Color())
	c.subjectField.SetLabelColor(componentColors.Title.Color())
	c.subjectField.SetPlaceholderTextColor(c.app.getHintColor())

	// Apply to CC/BCC fields as well
	c.ccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.ccField.SetFieldTextColor(componentColors.Text.Color())
	c.ccField.SetLabelColor(componentColors.Title.Color())
	c.ccField.SetPlaceholderTextColor(c.app.getHintColor())

	c.bccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.bccField.SetFieldTextColor(componentColors.Text.Color())
	c.bccField.SetLabelColor(componentColors.Title.Color())
	c.bccField.SetPlaceholderTextColor(c.app.getHintColor())
}

// setupButtonSection arranges action buttons horizontally with hint text
func (c *CompositionPanel) setupButtonSection() {
	// Convert to vertical layout to include hint text
	c.buttonSection.SetDirection(tview.FlexRow)

	// Create horizontal container for buttons
	c.buttonRow = tview.NewFlex().SetDirection(tview.FlexColumn)
	componentColors := c.app.GetComponentColors("compose")
	c.buttonRow.SetBackgroundColor(componentColors.Background.Color())

	// Add buttons with themed spacers - align buttons to the right
	c.buttonRow.AddItem(c.spacer1, 0, 4, false) // Large left spacer to push buttons right
	c.buttonRow.AddItem(c.sendButton, 0, 1, false)
	c.buttonRow.AddItem(c.spacer2, 0, 1, false) // Spacer between buttons
	c.buttonRow.AddItem(c.draftButton, 0, 1, false)
	c.buttonRow.AddItem(c.spacer3, 0, 1, false) // Small right spacer for padding

	// Create hint text
	c.hintTextView = tview.NewTextView()
	c.hintTextView.SetText("Ctrl+Enter to Send | Esc to cancel")
	c.hintTextView.SetTextAlign(tview.AlignRight)
	c.hintTextView.SetBackgroundColor(componentColors.Background.Color())
	// Use compose component text color for consistency with other widgets
	c.hintTextView.SetTextColor(componentColors.Text.Color())
	c.hintTextView.SetBorder(false)

	// Create spacer line between buttons and hint text
	c.spacerLine = tview.NewTextView()
	c.spacerLine.SetText("")
	c.spacerLine.SetBackgroundColor(componentColors.Background.Color())
	c.spacerLine.SetBorder(false)

	// Add button row, spacer line, and hint text to main button section
	c.buttonSection.AddItem(c.buttonRow, 0, 1, false)    // Buttons take most space
	c.buttonSection.AddItem(c.spacerLine, 1, 0, false)   // Empty line spacer
	c.buttonSection.AddItem(c.hintTextView, 1, 0, false) // Hint text with fixed height

	// Configure button actions
	c.sendButton.SetSelectedFunc(func() { go c.sendComposition() })
	c.draftButton.SetSelectedFunc(func() { go c.saveDraft() })
}

// setupInputHandling implements comprehensive input capture to prevent global shortcuts
func (c *CompositionPanel) setupInputHandling() {
	// Main panel input capture - blocks ALL global shortcuts
	c.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle special navigation keys first
		switch event.Key() {
		case tcell.KeyEscape:
			c.hide()
			return nil
		case tcell.KeyTab:
			c.focusNext()
			return nil
		case tcell.KeyBacktab: // Shift+Tab
			c.focusPrevious()
			return nil
		case tcell.KeyCtrlJ: // Ctrl+J to send (Ctrl+Enter)
			go c.sendComposition()
			return nil
		}

		// Check if EditableTextView has focus and handle character input
		if c.bodySection != nil && c.bodySection.HasFocus() {
			// Log focus state for debugging
			if c.app.logger != nil {
				c.app.logger.Printf("=== CompositionPanel: EditableTextView has focus, handling key='%c' (rune=%d) ===", event.Rune(), event.Rune())
			}

			// For printable characters, forward to EditableTextView and consume the event
			if event.Key() == tcell.KeyRune && event.Rune() > 0 {
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding printable character '%c' to EditableTextView ===", event.Rune())
				}
				// Forward to EditableTextView's input handler
				c.bodySection.HandleCharInput(event.Rune())
				return nil // Consume the event to prevent global shortcuts
			}

			// For special keys like Enter, Backspace, etc., forward them too
			switch event.Key() {
			case tcell.KeyEnter:
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding Enter to EditableTextView ===")
				}
				c.bodySection.HandleEnter()
				return nil
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding Backspace to EditableTextView ===")
				}
				c.bodySection.HandleBackspace()
				return nil
			case tcell.KeyDelete:
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding Delete to EditableTextView ===")
				}
				c.bodySection.HandleDelete()
				return nil
			case tcell.KeyUp:
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding Up to EditableTextView ===")
				}
				c.bodySection.HandleArrowUp()
				return nil
			case tcell.KeyDown:
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding Down to EditableTextView ===")
				}
				c.bodySection.HandleArrowDown()
				return nil
			case tcell.KeyLeft:
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding Left to EditableTextView ===")
				}
				c.bodySection.HandleArrowLeft()
				return nil
			case tcell.KeyRight:
				if c.app.logger != nil {
					c.app.logger.Printf("=== CompositionPanel: Forwarding Right to EditableTextView ===")
				}
				c.bodySection.HandleArrowRight()
				return nil
			}
		}

		// Allow all other keys to pass through to focused component
		if c.app.logger != nil {
			c.app.logger.Printf("=== CompositionPanel: Allowing key to pass through to focused component ===")
		}
		return event
	})

	// Individual field input capture for specific behaviors
	c.setupFieldInputCapture()
}

// setupFocusManagement configures tab cycling and focus indicators
func (c *CompositionPanel) setupFocusManagement() {
	// Build initial focus order (will be updated when CC/BCC visibility changes)
	c.updateFocusOrder()
}

// setupFieldInputCapture adds input capture to prevent global shortcuts in text fields
func (c *CompositionPanel) setupFieldInputCapture() {
	// Capture for all input fields to prevent global shortcuts
	inputCapture := func(event *tcell.EventKey) *tcell.EventKey {
		// Allow ESC to bubble up for composition cancel
		if event.Key() == tcell.KeyEscape {
			return event
		}

		// Allow Tab/Shift+Tab to bubble up for focus navigation
		if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
			return event
		}

		// Allow Ctrl+J to bubble up for send
		if event.Key() == tcell.KeyCtrlJ {
			return event
		}

		// Block ALL other global shortcuts - only allow text editing keys
		return event
	}

	c.toField.SetInputCapture(inputCapture)
	c.ccField.SetInputCapture(inputCapture)
	c.bccField.SetInputCapture(inputCapture)
	c.subjectField.SetInputCapture(inputCapture)

	// EditableTextView has its own input capture for editing - no need to override
}

// Show displays the composition panel with improved focus management
func (c *CompositionPanel) Show(compositionType services.CompositionType, originalMessageID string) {
	go func() {
		_, _, _, _, _, compositionService, _, _, _, _, _, _ := c.app.GetServices()

		// Create new composition
		composition, err := compositionService.CreateComposition(c.app.ctx, compositionType, originalMessageID)
		if err != nil {
			c.app.GetErrorHandler().ShowError(c.app.ctx, fmt.Sprintf("Failed to create composition: %v", err))
			return
		}

		c.app.QueueUpdateDraw(func() {
			c.loadComposition(composition)
			c.isVisible = true
			if c.app.logger != nil {
				c.app.logger.Printf("📝 COMPOSER: Panel is now VISIBLE - setting up focus")
			}
			c.UpdateTheme()                   // Ensure theme is applied when shown
			c.updateSendButtonState("normal") // Reset send button state for new composition
			c.currentFocusIndex = 0
			c.focusCurrent()  // Focus first field (To)
			c.startAutoSave() // Enable auto-save for composition
		})
	}()
}

// ShowWithComposition displays the composition panel with a pre-loaded composition (for drafts)
func (c *CompositionPanel) ShowWithComposition(composition *services.Composition) {
	c.app.QueueUpdateDraw(func() {
		c.loadComposition(composition)
		c.isVisible = true
		if c.app.logger != nil {
			c.app.logger.Printf("📝 COMPOSER: Panel is now VISIBLE - loading existing composition")
		}
		c.UpdateTheme()                   // Ensure theme is applied when shown
		c.updateSendButtonState("normal") // Reset send button state for draft editing
		c.currentFocusIndex = 0
		c.focusCurrent()  // Focus first field (To)
		c.startAutoSave() // Enable auto-save for draft editing
	})
}

// loadComposition loads a composition into the form fields with improved data binding
func (c *CompositionPanel) loadComposition(composition *services.Composition) {
	c.composition = composition

	// Load data into input fields
	c.toField.SetText(strings.Join(c.formatRecipients(composition.To), ", "))
	c.ccField.SetText(strings.Join(c.formatRecipients(composition.CC), ", "))
	c.bccField.SetText(strings.Join(c.formatRecipients(composition.BCC), ", "))
	c.subjectField.SetText(composition.Subject)
	c.bodySection.SetText(composition.Body)

	// Show CC/BCC if they have content
	if len(composition.CC) > 0 || len(composition.BCC) > 0 {
		c.ccBccVisible = true
		c.updateCCBCCVisibility()
	}

	// Setup change handlers for real-time updates
	c.setupChangeHandlers()
}

// setupChangeHandlers configures real-time data binding for form fields
func (c *CompositionPanel) setupChangeHandlers() {
	// Update composition data when fields change
	c.toField.SetChangedFunc(func(text string) {
		if c.composition != nil {
			c.composition.To = c.parseRecipients(text)
		}
	})

	c.ccField.SetChangedFunc(func(text string) {
		if c.composition != nil {
			c.composition.CC = c.parseRecipients(text)
		}
	})

	c.bccField.SetChangedFunc(func(text string) {
		if c.composition != nil {
			c.composition.BCC = c.parseRecipients(text)
		}
	})

	c.subjectField.SetChangedFunc(func(text string) {
		if c.composition != nil {
			c.composition.Subject = text
		}
	})

	// Add body field change handler for real-time updates
	c.bodySection.SetChangedFunc(func(text string) {
		if c.composition != nil {
			c.composition.Body = text
		}
	})
}

// toggleCCBCC toggles the visibility of CC and BCC fields
func (c *CompositionPanel) toggleCCBCC() {
	c.ccBccVisible = !c.ccBccVisible
	c.updateCCBCCVisibility()
	c.updateFocusOrder()   // Rebuild focus order
	c.updateLayoutSizing() // Adjust header section size based on visibility
}

// updateCCBCCVisibility shows or hides CC/BCC fields dynamically
func (c *CompositionPanel) updateCCBCCVisibility() {
	// Clear and rebuild header section
	c.headerSection.Clear(true)

	// Add To field (always visible)
	c.headerSection.AddFormItem(c.toField)

	// Add CC/BCC fields if visible
	// Update and add CC/BCC toggle button
	if c.ccBccVisible {
		c.ccBccToggle.SetLabel("-CC/BCC")
		c.headerSection.AddFormItem(c.ccField)
		c.headerSection.AddFormItem(c.bccField)
	} else {
		c.ccBccToggle.SetLabel("+CC/BCC")
	}

	// Note: CC/BCC toggle button will be positioned separately in layout

	// Add Subject field (always visible)
	c.headerSection.AddFormItem(c.subjectField)
}

// updateLayoutSizing adjusts header section size based on CC/BCC visibility
func (c *CompositionPanel) updateLayoutSizing() {
	// Clear the main layout
	c.Flex.Clear()

	// Determine header size based on CC/BCC visibility (more compact without border)
	headerWeight := 1 // Very compact for To + Subject when CC/BCC is hidden
	if c.ccBccVisible {
		headerWeight = 2 // Slightly larger when CC/BCC + To + Subject visible
	}

	// Re-add items with appropriate sizing
	c.Flex.AddItem(c.headerContainer, 0, headerWeight, false) // Dynamic header size
	c.Flex.AddItem(c.bodyContainer, 0, 4, false)              // Body container - most space
	c.Flex.AddItem(c.buttonSection, 3, 0, false)              // Button section - compact height
}

// updateFocusOrder rebuilds the focus cycle based on current field visibility
func (c *CompositionPanel) updateFocusOrder() {
	c.focusableItems = make([]tview.Primitive, 0)

	// Add fields in tab order
	c.focusableItems = append(c.focusableItems, c.toField)

	// Add CC/BCC toggle button (always accessible)
	c.focusableItems = append(c.focusableItems, c.ccBccToggle)

	if c.ccBccVisible {
		c.focusableItems = append(c.focusableItems, c.ccField)
		c.focusableItems = append(c.focusableItems, c.bccField)
	}

	c.focusableItems = append(c.focusableItems, c.subjectField)
	c.focusableItems = append(c.focusableItems, c.bodySection)
	c.focusableItems = append(c.focusableItems, c.sendButton)
	c.focusableItems = append(c.focusableItems, c.draftButton)

	// Ensure current focus index is still valid
	if c.currentFocusIndex >= len(c.focusableItems) {
		c.currentFocusIndex = 0
	}
}

// focusNext moves focus to the next component in the cycle
func (c *CompositionPanel) focusNext() {
	if len(c.focusableItems) == 0 {
		return
	}

	c.currentFocusIndex = (c.currentFocusIndex + 1) % len(c.focusableItems)

	c.focusCurrent()
}

// focusPrevious moves focus to the previous component in the cycle
func (c *CompositionPanel) focusPrevious() {
	if len(c.focusableItems) == 0 {
		return
	}

	c.currentFocusIndex = (c.currentFocusIndex - 1 + len(c.focusableItems)) % len(c.focusableItems)
	c.focusCurrent()
}

// focusCurrent sets focus to the current component in the focus cycle
func (c *CompositionPanel) focusCurrent() {
	if len(c.focusableItems) == 0 || c.currentFocusIndex >= len(c.focusableItems) {
		return
	}

	focusTarget := c.focusableItems[c.currentFocusIndex]

	c.app.SetFocus(focusTarget)
}

// formatRecipients converts recipient structs to display strings
func (c *CompositionPanel) formatRecipients(recipients []services.Recipient) []string {
	result := make([]string, len(recipients))
	for i, r := range recipients {
		if r.Name != "" {
			result[i] = fmt.Sprintf("%s <%s>", r.Name, r.Email)
		} else {
			result[i] = r.Email
		}
	}
	return result
}

// parseRecipients converts comma-separated string to recipient structs
func (c *CompositionPanel) parseRecipients(text string) []services.Recipient {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	parts := strings.Split(text, ",")
	recipients := make([]services.Recipient, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse "Name <email@domain.com>" format
		if strings.Contains(part, "<") && strings.Contains(part, ">") {
			nameEnd := strings.Index(part, "<")
			emailStart := nameEnd + 1
			emailEnd := strings.Index(part, ">")

			if nameEnd > 0 && emailEnd > emailStart {
				name := strings.TrimSpace(part[:nameEnd])
				email := strings.TrimSpace(part[emailStart:emailEnd])
				recipients = append(recipients, services.Recipient{
					Name:  name,
					Email: email,
				})
				continue
			}
		}

		// Plain email format
		recipients = append(recipients, services.Recipient{
			Email: part,
		})
	}

	return recipients
}

// updateSendButtonState updates the send button appearance and state
func (c *CompositionPanel) updateSendButtonState(state string) {
	componentColors := c.app.GetComponentColors("compose")

	switch state {
	case "normal":
		c.sendButton.SetLabel("📧 Send")
		c.sendButton.SetBackgroundColor(componentColors.Accent.Color()) // Green for active
		c.sendButton.SetLabelColor(componentColors.Background.Color())
		// Re-enable button action
		c.sendButton.SetSelectedFunc(func() { go c.sendComposition() })
	case "sending":
		c.sendButton.SetLabel("Sending...")
		c.sendButton.SetBackgroundColor(componentColors.Border.Color()) // Gray for disabled
		c.sendButton.SetLabelColor(componentColors.Text.Color())
		// Don't override SetSelectedFunc completely - might interfere with ESC handling
		// Just make it a no-op for button clicks but keep the handler structure intact
		c.sendButton.SetSelectedFunc(func() {
			// Button disabled during sending - ignore clicks but don't block ESC
		})
	case "sent":
		c.sendButton.SetLabel("Sent!")
		c.sendButton.SetBackgroundColor(componentColors.Accent.Color()) // Green for success
		c.sendButton.SetLabelColor(componentColors.Background.Color())
		// Keep handler but make it no-op
		c.sendButton.SetSelectedFunc(func() {
			// Email sent - button disabled but ESC should still work
		})
	}
}

// sendComposition sends the current composition with enhanced feedback
func (c *CompositionPanel) sendComposition() {
	if c.composition == nil {
		c.app.GetErrorHandler().ShowError(c.app.ctx, "No composition to send")
		return
	}

	// Update composition with current form values
	c.updateCompositionFromForm()

	_, _, _, _, _, compositionService, _, _, _, _, _, _ := c.app.GetServices()

	// Validate composition
	validationErrors := compositionService.ValidateComposition(c.composition)
	if len(validationErrors) > 0 {
		errorMsg := "Validation errors:\n"
		for _, err := range validationErrors {
			errorMsg += fmt.Sprintf("- %s: %s\n", err.Field, err.Message)
		}
		c.app.GetErrorHandler().ShowError(c.app.ctx, errorMsg)
		return
	}

	// 1. Update button state to show sending
	c.updateSendButtonState("sending")

	// 2. Show initial progress in status bar
	c.app.GetErrorHandler().ShowProgress(c.app.ctx, "Preparing email...")

	// 3. Simplified send operation - avoid nested goroutines that can deadlock
	go func() {
		// Brief delay to show preparation step
		time.Sleep(200 * time.Millisecond)
		c.app.GetErrorHandler().ShowProgress(c.app.ctx, "Sending email...")

		// Send composition
		err := compositionService.SendComposition(context.Background(), c.composition)
		c.app.GetErrorHandler().ClearProgress()

		if err != nil {
			// Handle error case immediately
			c.app.QueueUpdateDraw(func() {
				c.updateSendButtonState("normal")
				c.app.GetErrorHandler().ShowError(c.app.ctx, fmt.Sprintf("Failed to send email: %v", err))
			})
			return
		}

		// Success case - handle immediately without complex nested logic
		c.app.QueueUpdateDraw(func() {
			c.updateSendButtonState("sent")
		})

		// Show success message
		recipientCount := len(c.composition.To) + len(c.composition.CC) + len(c.composition.BCC)
		successMsg := fmt.Sprintf("Email sent to %d recipient(s)!", recipientCount)
		c.app.GetErrorHandler().ShowSuccess(c.app.ctx, successMsg)

		// Auto-close after brief delay
		time.Sleep(1500 * time.Millisecond)
		c.app.QueueUpdateDraw(func() {
			c.hide()
		})
	}()
}

// saveDraft saves the current composition as a draft
func (c *CompositionPanel) saveDraft() {
	if c.composition == nil {
		c.app.GetErrorHandler().ShowError(c.app.ctx, "No composition to save")
		return
	}

	// Update composition with current form values
	c.updateCompositionFromForm()

	_, _, _, _, _, compositionService, _, _, _, _, _, _ := c.app.GetServices()

	c.app.GetErrorHandler().ShowProgress(c.app.ctx, "Saving draft...")

	draftID, err := compositionService.SaveDraft(context.Background(), c.composition)
	c.app.GetErrorHandler().ClearProgress()

	if err != nil {
		c.app.GetErrorHandler().ShowError(c.app.ctx, fmt.Sprintf("Failed to save draft: %v", err))
		return
	}

	c.composition.ID = draftID
	c.app.GetErrorHandler().ShowSuccess(c.app.ctx, "Draft saved successfully!")
}

// updateCompositionFromForm updates the composition with current form values
func (c *CompositionPanel) updateCompositionFromForm() {
	// Real-time updates handle most fields, but we need to get body text manually
	if c.composition != nil {
		c.composition.Body = c.bodySection.GetText() // InputField GetText() has no parameters
	}
}

// hide hides the composition panel and restores focus
func (c *CompositionPanel) hide() {
	if c.app.logger != nil {
		c.app.logger.Printf("📝 COMPOSER: Panel is now HIDDEN - restoring focus")
	}
	c.isVisible = false
	c.composition = nil
	c.currentFocusIndex = 0
	c.ccBccVisible = false
	c.stopAutoSave() // Disable auto-save when hiding

	// Clear form fields
	c.toField.SetText("")
	c.ccField.SetText("")
	c.bccField.SetText("")
	c.subjectField.SetText("")
	c.bodySection.SetText("")

	// Update CC/BCC visibility to default
	c.updateCCBCCVisibility()
	c.updateFocusOrder()

	// Remove the composition page (with status bar layout)
	c.app.Pages.RemovePage("compose_with_status")

	// Switch back to main page explicitly
	c.app.Pages.SwitchToPage("main")

	// Return focus to main view
	if list := c.app.views["list"]; list != nil {
		c.app.SetFocus(list)
		c.app.currentFocus = "list"

		// Check if we need to auto-select a message after closing composer
		if table, ok := list.(*tview.Table); ok {
			currentRow, _ := table.GetSelection()
			// If no message is selected and we have messages, select the first one
			if (currentRow < 1 || c.app.GetCurrentMessageID() == "") && len(c.app.ids) > 0 && table.GetRowCount() > 1 {
				table.Select(1, 0) // Select first message (row 1, since row 0 is header)
				firstID := c.app.ids[0]
				c.app.SetCurrentMessageID(firstID)
				go c.app.showMessageWithoutFocus(firstID)
			}
		}
	}
}

// IsVisible returns whether the composition panel is currently visible
func (c *CompositionPanel) IsVisible() bool {
	return c.isVisible
}

// GetCurrentComposition returns the current composition being edited
func (c *CompositionPanel) GetCurrentComposition() *services.Composition {
	return c.composition
}

// UpdateTheme updates the composition panel colors when theme changes
func (c *CompositionPanel) UpdateTheme() {
	if !c.isVisible {
		return // Don't update if not currently visible
	}

	// Get updated theme colors
	componentColors := c.app.GetComponentColors("compose")

	// Debug: Log theme colors being applied
	if c.app.logger != nil {
		c.app.logger.Printf("DEBUG COMPOSE THEME UpdateTheme called:")
		c.app.logger.Printf("  Background: %s", string(componentColors.Background))
		c.app.logger.Printf("  Text: %s", string(componentColors.Text))
		c.app.logger.Printf("  Title: %s", string(componentColors.Title))
		c.app.logger.Printf("  Border: %s", string(componentColors.Border))
		c.app.logger.Printf("  Accent: %s", string(componentColors.Accent))
		c.app.logger.Printf("DEBUG BUTTON STYLING:")
		c.app.logger.Printf("  Send button: bg=%s (accent), text=%s (background)", string(componentColors.Accent), string(componentColors.Background))
		c.app.logger.Printf("  Draft/Cancel buttons: bg=%s (border), text=%s", string(componentColors.Border), string(componentColors.Text))
		c.app.logger.Printf("  Button section: bg=%s (background), title=%s (title), border=%s", string(componentColors.Background), string(componentColors.Title), string(componentColors.Border))
		c.app.logger.Printf("  EXPECTED: bg=#1e2540 (dark slate), title=#4fc3f7 (cyan), border=#3f4a5f (medium slate)")
		c.app.logger.Printf("DEBUG FIELD STYLING:")
		c.app.logger.Printf("  Form field background: %s (should be dark slate #1e2540)", string(componentColors.Background))
		c.app.logger.Printf("  Form field text: %s (should be light blue-gray #c5d1eb)", string(componentColors.Text))
		c.app.logger.Printf("  Form labels: %s (should be cyan #4fc3f7, NOT yellow)", string(componentColors.Title))
	}

	// Update main container
	c.Flex.SetBackgroundColor(componentColors.Background.Color())
	c.Flex.SetTitleColor(componentColors.Title.Color())
	c.Flex.SetBorderColor(componentColors.Border.Color())

	// Update header container (no border)
	c.headerContainer.SetBackgroundColor(componentColors.Background.Color())

	// Update header section with complete Form-level styling
	c.headerSection.SetBackgroundColor(componentColors.Background.Color())
	c.headerSection.SetTitleColor(componentColors.Title.Color())
	c.headerSection.SetBorderColor(componentColors.Border.Color())
	// Apply Form-level field styling with compose theme colors
	c.headerSection.SetFieldBackgroundColor(componentColors.Background.Color()) // Dark slate blue
	c.headerSection.SetFieldTextColor(componentColors.Text.Color())             // Light blue-gray text
	c.headerSection.SetLabelColor(componentColors.Title.Color())                // Cyan labels (not yellow)
	c.headerSection.SetButtonBackgroundColor(componentColors.Border.Color())
	c.headerSection.SetButtonTextColor(componentColors.Text.Color())

	// Update individual input fields with placeholder colors
	c.toField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.toField.SetFieldTextColor(componentColors.Text.Color())
	c.toField.SetLabelColor(componentColors.Title.Color())
	c.toField.SetPlaceholderTextColor(c.app.getHintColor())

	c.ccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.ccField.SetFieldTextColor(componentColors.Text.Color())
	c.ccField.SetLabelColor(componentColors.Title.Color())
	c.ccField.SetPlaceholderTextColor(c.app.getHintColor())

	c.bccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.bccField.SetFieldTextColor(componentColors.Text.Color())
	c.bccField.SetLabelColor(componentColors.Title.Color())
	c.bccField.SetPlaceholderTextColor(c.app.getHintColor())

	c.subjectField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.subjectField.SetFieldTextColor(componentColors.Text.Color())
	c.subjectField.SetLabelColor(componentColors.Title.Color())
	c.subjectField.SetPlaceholderTextColor(c.app.getHintColor())

	// Update body section (EditableTextView)
	c.bodySection.SetBackgroundColor(componentColors.Background.Color())
	c.bodySection.SetTextColor(componentColors.Text.Color())
	c.bodySection.SetBorderColor(componentColors.Border.Color())
	c.bodySection.SetPlaceholderTextColor(c.app.getHintColor())

	// Update body container (no border)
	c.bodyContainer.SetBackgroundColor(componentColors.Background.Color())

	// Update buttons with improved styling
	c.sendButton.SetBackgroundColor(componentColors.Accent.Color()) // Green for Send button prominence
	c.sendButton.SetLabelColor(componentColors.Background.Color())  // Dark text on green background

	c.draftButton.SetBackgroundColor(componentColors.Border.Color())
	c.draftButton.SetLabelColor(componentColors.Text.Color())

	c.ccBccToggle.SetBackgroundColor(componentColors.Border.Color())
	c.ccBccToggle.SetLabelColor(componentColors.Text.Color())

	// Update button section with styling (no border)
	c.buttonSection.SetBackgroundColor(componentColors.Background.Color())

	// Update button section spacers to match theme background
	c.spacer1.SetBackgroundColor(componentColors.Background.Color())
	c.spacer2.SetBackgroundColor(componentColors.Background.Color())
	c.spacer3.SetBackgroundColor(componentColors.Background.Color())

	// Update CC/BCC toggle spacers to match theme background
	c.toggleTopSpacer.SetBackgroundColor(componentColors.Background.Color())
	c.toggleBottomSpacer.SetBackgroundColor(componentColors.Background.Color())

	// Update hint text, button row, and spacer line with theme colors
	if c.hintTextView != nil {
		c.hintTextView.SetTextColor(componentColors.Text.Color())
		c.hintTextView.SetBackgroundColor(componentColors.Background.Color())
	}
	if c.buttonRow != nil {
		c.buttonRow.SetBackgroundColor(componentColors.Background.Color())
	}
	if c.spacerLine != nil {
		c.spacerLine.SetBackgroundColor(componentColors.Background.Color())
	}

	// Note: Removed ForceFilledBorderFlex to fix black background issue
	// Button section colors now apply directly without interference

	// IMPORTANT: Re-apply field styling after theme update
	c.applyFieldStyling()
}

// startAutoSave enables periodic auto-saving of the composition as a draft
func (c *CompositionPanel) startAutoSave() {
	if c.autoSaveEnabled {
		return // Already enabled
	}

	c.autoSaveEnabled = true
	c.scheduleNextAutoSave()

	if c.app.logger != nil {
		c.app.logger.Printf("CompositionPanel: Auto-save started")
	}
}

// stopAutoSave disables auto-saving and cleans up the timer
func (c *CompositionPanel) stopAutoSave() {
	c.autoSaveEnabled = false

	if c.autoSaveTimer != nil {
		c.autoSaveTimer.Stop()
		c.autoSaveTimer = nil
	}

	if c.app.logger != nil {
		c.app.logger.Printf("CompositionPanel: Auto-save stopped")
	}
}

// scheduleNextAutoSave schedules the next auto-save operation
func (c *CompositionPanel) scheduleNextAutoSave() {
	if !c.autoSaveEnabled {
		return
	}

	// Auto-save every 30 seconds
	const autoSaveInterval = 30 * time.Second

	c.autoSaveTimer = time.AfterFunc(autoSaveInterval, func() {
		if c.autoSaveEnabled {
			c.performAutoSave()
			c.scheduleNextAutoSave() // Schedule next save
		}
	})
}

// performAutoSave saves the current composition as a draft if content has changed
func (c *CompositionPanel) performAutoSave() {
	if !c.autoSaveEnabled || c.composition == nil {
		return
	}

	// Capture current content for comparison
	currentContent := c.getCurrentContent()

	// Only save if content has changed
	if currentContent == c.lastSaveContent {
		return
	}

	// Update composition with current values
	c.updateCompositionFromFields()

	// Get composition service
	_, _, _, _, _, compositionService, _, _, _, _, _, _ := c.app.GetServices()

	// Save as draft
	go func() {
		draftID, err := compositionService.SaveDraft(context.Background(), c.composition)
		if err != nil {
			if c.app.logger != nil {
				c.app.logger.Printf("CompositionPanel: Auto-save failed: %v", err)
			}
			// Don't show error to user for background auto-save failures
			return
		}

		// Update the draft ID in the composition
		c.composition.DraftID = draftID
		c.lastSaveContent = currentContent

		if c.app.logger != nil {
			c.app.logger.Printf("CompositionPanel: Auto-saved as draft %s", draftID)
		}

		// Show subtle feedback to user
		go func() {
			c.app.GetErrorHandler().ShowInfo(context.Background(), "📝 Draft auto-saved")
		}()
	}()
}

// getCurrentContent returns a string representation of all composition content for change detection
func (c *CompositionPanel) getCurrentContent() string {
	if c.composition == nil {
		return ""
	}

	to := ""
	if c.toField != nil {
		to = c.toField.GetText()
	}

	subject := ""
	if c.subjectField != nil {
		subject = c.subjectField.GetText()
	}

	body := ""
	if c.bodySection != nil {
		body = c.bodySection.GetText()
	}

	cc := ""
	bcc := ""
	if c.ccBccVisible {
		if c.ccField != nil {
			cc = c.ccField.GetText()
		}
		if c.bccField != nil {
			bcc = c.bccField.GetText()
		}
	}

	return fmt.Sprintf("to:%s|cc:%s|bcc:%s|subject:%s|body:%s", to, cc, bcc, subject, body)
}

// updateCompositionFromFields updates the composition object with current field values
func (c *CompositionPanel) updateCompositionFromFields() {
	if c.composition == nil {
		return
	}

	// Update recipients
	if c.toField != nil {
		c.composition.To = c.parseRecipients(c.toField.GetText())
	}
	if c.ccField != nil {
		c.composition.CC = c.parseRecipients(c.ccField.GetText())
	}
	if c.bccField != nil {
		c.composition.BCC = c.parseRecipients(c.bccField.GetText())
	}

	// Update subject and body
	if c.subjectField != nil {
		c.composition.Subject = c.subjectField.GetText()
	}
	if c.bodySection != nil {
		c.composition.Body = c.bodySection.GetText()
	}
}
