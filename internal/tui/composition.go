package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// CompositionPanel represents the email composition UI with proper layout and focus management
type CompositionPanel struct {
	*tview.Flex
	app *App
	
	// UI Components
	headerSection *tview.Form
	bodySection   *tview.TextView // TextView with enhanced direct editing capability
	buttonSection *tview.Flex
	
	// Input fields
	toField      *tview.InputField
	ccField      *tview.InputField
	bccField     *tview.InputField
	subjectField *tview.InputField
	
	// Action buttons
	sendButton   *tview.Button
	draftButton  *tview.Button
	cancelButton *tview.Button
	ccBccToggle  *tview.Button
	
	// State management
	composition    *services.Composition
	isVisible      bool
	ccBccVisible   bool
	currentFocusIndex int
	focusableItems []tview.Primitive
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
	
	// Create header section with input fields
	c.headerSection = tview.NewForm()
	c.headerSection.SetBackgroundColor(componentColors.Background.Color())
	c.headerSection.SetBorder(true)
	c.headerSection.SetTitle(" Recipients & Subject ")
	c.headerSection.SetTitleColor(componentColors.Title.Color())
	c.headerSection.SetBorderColor(componentColors.Border.Color())
	
	// Set Form-level field styling with compose theme colors (applied first)
	c.headerSection.SetFieldBackgroundColor(componentColors.Background.Color()) // Dark slate blue
	c.headerSection.SetFieldTextColor(componentColors.Text.Color()) // Light blue-gray text
	c.headerSection.SetLabelColor(componentColors.Title.Color()) // Cyan labels (not yellow)
	c.headerSection.SetButtonBackgroundColor(componentColors.Border.Color())
	c.headerSection.SetButtonTextColor(componentColors.Text.Color())
	
	// Create individual input fields with proper theming
	c.toField = tview.NewInputField()
	c.toField.SetLabel("To: ")
	c.toField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.toField.SetFieldTextColor(componentColors.Text.Color())
	c.toField.SetLabelColor(componentColors.Title.Color())
	
	c.ccField = tview.NewInputField()
	c.ccField.SetLabel("CC: ")
	c.ccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.ccField.SetFieldTextColor(componentColors.Text.Color())
	c.ccField.SetLabelColor(componentColors.Title.Color())
	
	c.bccField = tview.NewInputField()
	c.bccField.SetLabel("BCC: ")
	c.bccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.bccField.SetFieldTextColor(componentColors.Text.Color())
	c.bccField.SetLabelColor(componentColors.Title.Color())
	
	c.subjectField = tview.NewInputField()
	c.subjectField.SetLabel("Subject: ")
	c.subjectField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.subjectField.SetFieldTextColor(componentColors.Text.Color())
	c.subjectField.SetLabelColor(componentColors.Title.Color())
	
	// Create CC/BCC toggle button
	c.ccBccToggle = tview.NewButton("+CC/BCC")
	c.ccBccToggle.SetBackgroundColor(componentColors.Border.Color())
	c.ccBccToggle.SetLabelColor(componentColors.Text.Color())
	
	// Create message body with enhanced TextView for direct editing (no modal popup)
	c.bodySection = tview.NewTextView()
	c.bodySection.SetBackgroundColor(componentColors.Background.Color())
	c.bodySection.SetTextColor(componentColors.Text.Color())
	c.bodySection.SetBorder(true)
	c.bodySection.SetTitle(" Message Body (Click and type directly) ")
	c.bodySection.SetTitleColor(componentColors.Title.Color())
	c.bodySection.SetBorderColor(componentColors.Border.Color())
	c.bodySection.SetWrap(true)
	c.bodySection.SetWordWrap(true)
	c.bodySection.SetDynamicColors(false)
	// Make TextView scrollable and focused-editable
	c.bodySection.SetScrollable(true)
	
	// Create action buttons with improved styling
	c.sendButton = tview.NewButton("Send (Ctrl+Enter)")
	c.sendButton.SetBackgroundColor(componentColors.Accent.Color()) // Green for Send button prominence
	c.sendButton.SetLabelColor(componentColors.Background.Color()) // Dark text on green background
	
	c.draftButton = tview.NewButton("Save Draft")
	c.draftButton.SetBackgroundColor(componentColors.Border.Color())
	c.draftButton.SetLabelColor(componentColors.Text.Color())
	
	c.cancelButton = tview.NewButton("Cancel (Esc)")
	c.cancelButton.SetBackgroundColor(componentColors.Border.Color())
	c.cancelButton.SetLabelColor(componentColors.Text.Color())
	
	// Create button section with proper styling
	c.buttonSection = tview.NewFlex()
	c.buttonSection.SetDirection(tview.FlexColumn)
	c.buttonSection.SetBackgroundColor(componentColors.Background.Color())
	c.buttonSection.SetBorder(true) // Add border for visual separation
	c.buttonSection.SetTitle(" Actions ")
	c.buttonSection.SetTitleColor(componentColors.Title.Color())
	c.buttonSection.SetBorderColor(componentColors.Border.Color())
	
	// Apply ForceFilledBorderFlex for consistent background rendering (like other bordered Flex containers)
	ForceFilledBorderFlex(c.buttonSection)
	
	// IMPORTANT: Reapply theme colors AFTER ForceFilledBorderFlex (it overrides colors)
	c.buttonSection.SetBackgroundColor(componentColors.Background.Color()) // Dark slate blue, not black
	c.buttonSection.SetTitleColor(componentColors.Title.Color()) // Cyan, not white
	c.buttonSection.SetBorderColor(componentColors.Border.Color())
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
	
	// Setup header section with form fields
	c.setupHeaderSection()
	
	// Setup button section with actions
	c.setupButtonSection()
	
	// Layout: Header (fixed) → Body (expand) → Buttons (fixed)
	c.Flex.AddItem(c.headerSection, 0, 1, false)  // Header section - compact
	c.Flex.AddItem(c.bodySection, 0, 4, false)    // Body section - most space
	c.Flex.AddItem(c.buttonSection, 3, 0, false)  // Button section - fixed height
}

// setupHeaderSection configures the header form with dynamic CC/BCC visibility
func (c *CompositionPanel) setupHeaderSection() {
	// Add To field (always visible)
	c.headerSection.AddFormItem(c.toField)
	
	// Add Subject field (always visible)
	c.headerSection.AddFormItem(c.subjectField)
	
	// Add CC/BCC toggle button
	c.headerSection.AddButton("+CC/BCC", c.toggleCCBCC)
	
	// Initially hide CC/BCC fields
	c.updateCCBCCVisibility()
	
	// IMPORTANT: Apply field styling AFTER fields are added to Form
	c.applyFieldStyling()
}

// applyFieldStyling applies theme colors to form fields AFTER they're added to the Form
func (c *CompositionPanel) applyFieldStyling() {
	componentColors := c.app.GetComponentColors("compose")
	
	// Debug logging for field styling
	if c.app.logger != nil {
		c.app.logger.Printf("DEBUG applyFieldStyling: Applying colors AFTER fields added to Form")
		c.app.logger.Printf("  Field bg: %s, Field text: %s, Labels: %s", 
			string(componentColors.Background), string(componentColors.Text), string(componentColors.Title))
	}
	
	// Re-apply Form-level styling (crucial for tview Forms)
	c.headerSection.SetFieldBackgroundColor(componentColors.Background.Color())
	c.headerSection.SetFieldTextColor(componentColors.Text.Color())
	c.headerSection.SetLabelColor(componentColors.Title.Color()) // This should make labels cyan
	
	// Re-apply individual field styling for extra emphasis
	c.toField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.toField.SetFieldTextColor(componentColors.Text.Color())
	c.toField.SetLabelColor(componentColors.Title.Color())
	
	c.subjectField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.subjectField.SetFieldTextColor(componentColors.Text.Color())
	c.subjectField.SetLabelColor(componentColors.Title.Color())
	
	// Apply to CC/BCC fields as well
	c.ccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.ccField.SetFieldTextColor(componentColors.Text.Color())
	c.ccField.SetLabelColor(componentColors.Title.Color())
	
	c.bccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.bccField.SetFieldTextColor(componentColors.Text.Color())
	c.bccField.SetLabelColor(componentColors.Title.Color())
}

// setupButtonSection arranges action buttons horizontally
func (c *CompositionPanel) setupButtonSection() {
	// Add buttons with spacing
	c.buttonSection.AddItem(c.sendButton, 0, 1, false)
	c.buttonSection.AddItem(tview.NewBox(), 0, 1, false) // spacer
	c.buttonSection.AddItem(c.draftButton, 0, 1, false)
	c.buttonSection.AddItem(tview.NewBox(), 0, 1, false) // spacer
	c.buttonSection.AddItem(c.cancelButton, 0, 1, false)
	c.buttonSection.AddItem(tview.NewBox(), 0, 2, false) // larger spacer to push buttons left
	
	// Configure button actions
	c.sendButton.SetSelectedFunc(func() { go c.sendComposition() })
	c.draftButton.SetSelectedFunc(func() { go c.saveDraft() })
	c.cancelButton.SetSelectedFunc(func() { c.hide() })
}

// setupInputHandling implements comprehensive input capture to prevent global shortcuts
func (c *CompositionPanel) setupInputHandling() {
	// Main panel input capture - blocks ALL global shortcuts
	c.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
		
		// Allow all other keys to pass through to focused component
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
	
	// Enhanced input capture for TextView body section to enable direct editing
	c.bodySection.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			return event // Allow ESC to bubble up
		case tcell.KeyTab, tcell.KeyBacktab:
			return event // Allow Tab navigation to bubble up
		case tcell.KeyCtrlJ:
			return event // Allow Ctrl+J to bubble up for send
		case tcell.KeyEnter:
			// Handle Enter key for newlines in direct editing mode
			c.handleDirectBodyInput("\n")
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			// Handle backspace
			c.handleBackspaceInBody()
			return nil
		}
		
		// Handle character input for direct editing
		if event.Rune() != 0 {
			c.handleDirectBodyInput(string(event.Rune()))
			return nil
		}
		
		// Block other global shortcuts when body is focused
		return event
	})
}

// handleDirectBodyInput appends text to the body section for direct editing
func (c *CompositionPanel) handleDirectBodyInput(text string) {
	currentText := c.bodySection.GetText(false)
	newText := currentText + text
	c.bodySection.SetText(newText)
	
	// Update composition in real-time
	if c.composition != nil {
		c.composition.Body = newText
	}
}

// handleBackspaceInBody removes the last character from body text
func (c *CompositionPanel) handleBackspaceInBody() {
	currentText := c.bodySection.GetText(false)
	if len(currentText) > 0 {
		// Remove last character (handling UTF-8 properly)
		runes := []rune(currentText)
		if len(runes) > 0 {
			newText := string(runes[:len(runes)-1])
			c.bodySection.SetText(newText)
			
			// Update composition in real-time
			if c.composition != nil {
				c.composition.Body = newText
			}
		}
	}
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
			c.UpdateTheme() // Ensure theme is applied when shown
			c.currentFocusIndex = 0
			c.focusCurrent() // Focus first field (To)
		})
	}()
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
	
	// Note: TextView doesn't have SetChangedFunc - we'll handle body updates differently
	// Body updates will be handled on send/save actions
}

// toggleCCBCC toggles the visibility of CC and BCC fields
func (c *CompositionPanel) toggleCCBCC() {
	c.ccBccVisible = !c.ccBccVisible
	c.updateCCBCCVisibility()
	c.updateFocusOrder() // Rebuild focus order
}

// updateCCBCCVisibility shows or hides CC/BCC fields dynamically
func (c *CompositionPanel) updateCCBCCVisibility() {
	// Clear and rebuild header section
	c.headerSection.Clear(true)
	
	// Add To field (always visible)
	c.headerSection.AddFormItem(c.toField)
	
	// Add CC/BCC fields if visible
	if c.ccBccVisible {
		c.headerSection.AddFormItem(c.ccField)
		c.headerSection.AddFormItem(c.bccField)
		c.ccBccToggle.SetLabel("-CC/BCC")
	} else {
		c.ccBccToggle.SetLabel("+CC/BCC")
	}
	
	// Add Subject field (always visible)
	c.headerSection.AddFormItem(c.subjectField)
	
	// Add CC/BCC toggle button
	c.headerSection.AddButton(c.ccBccToggle.GetLabel(), c.toggleCCBCC)
}

// updateFocusOrder rebuilds the focus cycle based on current field visibility
func (c *CompositionPanel) updateFocusOrder() {
	c.focusableItems = make([]tview.Primitive, 0)
	
	// Add fields in tab order
	c.focusableItems = append(c.focusableItems, c.toField)
	
	if c.ccBccVisible {
		c.focusableItems = append(c.focusableItems, c.ccField)
		c.focusableItems = append(c.focusableItems, c.bccField)
	}
	
	c.focusableItems = append(c.focusableItems, c.subjectField)
	c.focusableItems = append(c.focusableItems, c.bodySection)
	c.focusableItems = append(c.focusableItems, c.sendButton)
	c.focusableItems = append(c.focusableItems, c.draftButton)
	c.focusableItems = append(c.focusableItems, c.cancelButton)
	
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
	
	c.app.SetFocus(c.focusableItems[c.currentFocusIndex])
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

// sendComposition sends the current composition
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
	
	// Send composition
	c.app.GetErrorHandler().ShowProgress(c.app.ctx, "Sending email...")
	
	err := compositionService.SendComposition(context.Background(), c.composition)
	c.app.GetErrorHandler().ClearProgress()
	
	if err != nil {
		c.app.GetErrorHandler().ShowError(c.app.ctx, fmt.Sprintf("Failed to send email: %v", err))
		return
	}
	
	c.app.GetErrorHandler().ShowSuccess(c.app.ctx, "Email sent successfully!")
	c.hide()
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
		c.composition.Body = c.bodySection.GetText(false) // TextView GetText() needs boolean parameter
	}
}


// hide hides the composition panel and restores focus
func (c *CompositionPanel) hide() {
	c.isVisible = false
	c.composition = nil
	c.currentFocusIndex = 0
	c.ccBccVisible = false
	
	// Clear form fields
	c.toField.SetText("")
	c.ccField.SetText("")
	c.bccField.SetText("")
	c.subjectField.SetText("")
	c.bodySection.SetText("")
	
	// Update CC/BCC visibility to default
	c.updateCCBCCVisibility()
	c.updateFocusOrder()
	
	// Remove the composition page
	c.app.Pages.RemovePage("compose")
	
	// Return focus to main view
	if list := c.app.views["list"]; list != nil {
		c.app.SetFocus(list)
		c.app.currentFocus = "list"
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
		c.app.logger.Printf("  Button section: bg=%s (background), border=%s", string(componentColors.Background), string(componentColors.Border))
		c.app.logger.Printf("DEBUG FIELD STYLING:")
		c.app.logger.Printf("  Form field background: %s (should be dark slate #1e2540)", string(componentColors.Background))
		c.app.logger.Printf("  Form field text: %s (should be light blue-gray #c5d1eb)", string(componentColors.Text))
		c.app.logger.Printf("  Form labels: %s (should be cyan #4fc3f7, NOT yellow)", string(componentColors.Title))
		c.app.logger.Printf("  Expected vs Actual - Background: %s, Labels: %s", string(componentColors.Background), string(componentColors.Title))
	}
	
	// Update main container
	c.Flex.SetBackgroundColor(componentColors.Background.Color())
	c.Flex.SetTitleColor(componentColors.Title.Color())
	c.Flex.SetBorderColor(componentColors.Border.Color())
	
	// Update header section with complete Form-level styling
	c.headerSection.SetBackgroundColor(componentColors.Background.Color())
	c.headerSection.SetTitleColor(componentColors.Title.Color())
	c.headerSection.SetBorderColor(componentColors.Border.Color())
	// Apply Form-level field styling with compose theme colors
	c.headerSection.SetFieldBackgroundColor(componentColors.Background.Color()) // Dark slate blue
	c.headerSection.SetFieldTextColor(componentColors.Text.Color()) // Light blue-gray text
	c.headerSection.SetLabelColor(componentColors.Title.Color()) // Cyan labels (not yellow)
	c.headerSection.SetButtonBackgroundColor(componentColors.Border.Color())
	c.headerSection.SetButtonTextColor(componentColors.Text.Color())
	
	// Update individual input fields
	c.toField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.toField.SetFieldTextColor(componentColors.Text.Color())
	c.toField.SetLabelColor(componentColors.Title.Color())
	
	c.ccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.ccField.SetFieldTextColor(componentColors.Text.Color())
	c.ccField.SetLabelColor(componentColors.Title.Color())
	
	c.bccField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.bccField.SetFieldTextColor(componentColors.Text.Color())
	c.bccField.SetLabelColor(componentColors.Title.Color())
	
	c.subjectField.SetFieldBackgroundColor(componentColors.Background.Color())
	c.subjectField.SetFieldTextColor(componentColors.Text.Color())
	c.subjectField.SetLabelColor(componentColors.Title.Color())
	
	// Update body section (enhanced TextView)
	c.bodySection.SetBackgroundColor(componentColors.Background.Color())
	c.bodySection.SetTextColor(componentColors.Text.Color())
	c.bodySection.SetTitleColor(componentColors.Title.Color())
	c.bodySection.SetBorderColor(componentColors.Border.Color())
	
	// Update buttons with improved styling
	c.sendButton.SetBackgroundColor(componentColors.Accent.Color()) // Green for Send button prominence
	c.sendButton.SetLabelColor(componentColors.Background.Color()) // Dark text on green background
	
	c.draftButton.SetBackgroundColor(componentColors.Border.Color())
	c.draftButton.SetLabelColor(componentColors.Text.Color())
	
	c.cancelButton.SetBackgroundColor(componentColors.Border.Color())
	c.cancelButton.SetLabelColor(componentColors.Text.Color())
	
	c.ccBccToggle.SetBackgroundColor(componentColors.Border.Color())
	c.ccBccToggle.SetLabelColor(componentColors.Text.Color())
	
	// Update button section with complete styling
	c.buttonSection.SetBackgroundColor(componentColors.Background.Color())
	c.buttonSection.SetTitleColor(componentColors.Title.Color())
	c.buttonSection.SetBorderColor(componentColors.Border.Color())
	
	// Apply ForceFilledBorderFlex for consistent background rendering
	ForceFilledBorderFlex(c.buttonSection)
	
	// IMPORTANT: Reapply theme colors AFTER ForceFilledBorderFlex (it overrides colors)
	c.buttonSection.SetBackgroundColor(componentColors.Background.Color()) // Dark slate blue, not black
	c.buttonSection.SetTitleColor(componentColors.Title.Color()) // Cyan, not white
	c.buttonSection.SetBorderColor(componentColors.Border.Color())
	
	// IMPORTANT: Re-apply field styling after theme update
	c.applyFieldStyling()
}