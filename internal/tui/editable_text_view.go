package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// EditableTextView provides multiline text editing capabilities using proxy pattern
// Uses composition instead of embedding to avoid method promotion confusion
type EditableTextView struct {
	// Underlying TextView (composition, not embedding)
	textView *tview.TextView
	app      *App

	// Editing state
	text         string
	cursorLine   int
	cursorColumn int
	lines        []string

	// Editing capabilities
	isEditable bool
	changeFunc func(string)

	// Placeholder support
	placeholder       string
	placeholderColor  tcell.Color
	originalTextColor tcell.Color // Store original text color to restore after placeholder

	// Display state
	updating bool // Prevents recursive updateDisplay calls
}

// NewEditableTextView creates a new multiline text editor using proxy pattern
func NewEditableTextView(app *App) *EditableTextView {
	// Create a clean TextView for our proxy
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetScrollable(true)

	editable := &EditableTextView{
		textView:     textView,
		app:          app,
		text:         "",
		cursorLine:   0,
		cursorColumn: 0,
		lines:        []string{""},
		isEditable:   true,
	}

	// Input handling is done through InputHandler() method, not SetInputCapture

	return editable
}

// Proxy Pattern: Implement key tview.Primitive methods by delegating to textView

// Draw delegates drawing to the underlying TextView
func (e *EditableTextView) Draw(screen tcell.Screen) {
	if e.textView != nil {
		e.textView.Draw(screen)
	}
}

// GetRect delegates to the underlying TextView
func (e *EditableTextView) GetRect() (int, int, int, int) {
	if e.textView != nil {
		return e.textView.GetRect()
	}
	return 0, 0, 0, 0
}

// SetRect delegates to the underlying TextView
func (e *EditableTextView) SetRect(x, y, width, height int) {
	if e.textView != nil {
		e.textView.SetRect(x, y, width, height)
	}
}

// InputHandler delegates to TextView's input handler (which we override)
func (e *EditableTextView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if e.textView != nil {
		return e.textView.InputHandler()
	}
	return nil
}

// setupTextViewInputHandler sets a custom input handler on the underlying TextView
func (e *EditableTextView) setupTextViewInputHandler() {
	if e.textView == nil {
		return
	}

	// Set custom input handler directly on the TextView
	e.textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		if !e.isEditable {
			return event // Pass through if not editable
		}

		switch event.Key() {
		case tcell.KeyEscape:
			// Allow ESC to bubble up (composition cancel)
			return event
		case tcell.KeyTab, tcell.KeyBacktab:
			// Allow Tab navigation to bubble up
			return event
		case tcell.KeyCtrlJ:
			// Allow Ctrl+J to bubble up (send composition)
			return event
		case tcell.KeyEnter:
			// Handle newline insertion
			e.insertNewline()
			return nil // CONSUME the event
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			// Handle backspace
			e.handleBackspace()
			return nil // CONSUME the event
		case tcell.KeyDelete:
			// Handle delete key
			e.handleDelete()
			return nil // CONSUME the event
		case tcell.KeyUp:
			// Move cursor up
			e.moveCursorUp()
			return nil // CONSUME the event
		case tcell.KeyDown:
			// Move cursor down
			e.moveCursorDown()
			return nil // CONSUME the event
		case tcell.KeyLeft:
			// Move cursor left
			e.moveCursorLeft()
			return nil // CONSUME the event
		case tcell.KeyRight:
			// Move cursor right
			e.moveCursorRight()
			return nil // CONSUME the event
		case tcell.KeyHome:
			// Move to beginning of line
			e.cursorColumn = 0
			e.updateDisplay()
			return nil // CONSUME the event
		case tcell.KeyEnd:
			// Move to end of line using rune-based length
			if e.cursorLine < len(e.lines) {
				e.cursorColumn = len([]rune(e.lines[e.cursorLine]))
			}
			e.updateDisplay()
			return nil // CONSUME the event
		}

		// Handle character input
		if event.Rune() != 0 && unicode.IsPrint(event.Rune()) {
			e.insertCharacter(event.Rune())
			return nil // CONSUME the event - critical for blocking global shortcuts
		}

		// For unhandled keys, pass them through
		return event
	})
}

// Focus delegates focus to the underlying TextView (but TextView has custom input handler)
func (e *EditableTextView) Focus(delegate func(p tview.Primitive)) {

	// Set our custom input handler on the TextView before focusing it
	e.setupTextViewInputHandler()

	// Delegate focus to TextView (which now has our custom handler)
	if e.textView != nil {
		delegate(e.textView)
	}

	// Update display to hide placeholder and show cursor when focused
	e.updateDisplay()
}

// HasFocus checks if the underlying TextView has focus
func (e *EditableTextView) HasFocus() bool {
	if e.textView != nil {
		hasFocus := e.textView.HasFocus()
		return hasFocus
	}
	return false
}

// Blur removes focus from the underlying TextView
func (e *EditableTextView) Blur() {

	if e.textView != nil {
		e.textView.Blur()
	}

	// Update display to show placeholder when not focused (if text is empty)
	e.updateDisplay()
}

// GetFocusable delegates to the underlying TextView
func (e *EditableTextView) GetFocusable() tview.Focusable {
	if e.textView != nil {
		return e.textView.GetFocusable()
	}
	return e
}

// MouseHandler delegates to the underlying TextView
func (e *EditableTextView) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	if e.textView != nil {
		return e.textView.MouseHandler()
	}
	return nil
}

// Proxy methods for TextView functionality

// SetBackgroundColor delegates to TextView
func (e *EditableTextView) SetBackgroundColor(color tcell.Color) *EditableTextView {
	if e.textView != nil {
		e.textView.SetBackgroundColor(color)
	}
	return e
}

// SetTextColor delegates to TextView and stores the color for placeholder restoration
func (e *EditableTextView) SetTextColor(color tcell.Color) *EditableTextView {
	if e.textView != nil {
		e.textView.SetTextColor(color)
		e.originalTextColor = color // Store for placeholder restoration
	}
	return e
}

// SetBorderColor delegates to TextView
func (e *EditableTextView) SetBorderColor(color tcell.Color) *EditableTextView {
	if e.textView != nil {
		e.textView.SetBorderColor(color)
	}
	return e
}

// SetBorder delegates to TextView
func (e *EditableTextView) SetBorder(show bool) *EditableTextView {
	if e.textView != nil {
		e.textView.SetBorder(show)
	}
	return e
}

// SetTitle delegates to TextView
func (e *EditableTextView) SetTitle(title string) *EditableTextView {
	if e.textView != nil {
		e.textView.SetTitle(title)
	}
	return e
}

// SetTitleColor delegates to TextView
func (e *EditableTextView) SetTitleColor(color tcell.Color) *EditableTextView {
	if e.textView != nil {
		e.textView.SetTitleColor(color)
	}
	return e
}

// Editing functionality

// SetText sets the text content and updates cursor position
func (e *EditableTextView) SetText(text string) {
	e.text = text
	e.lines = strings.Split(text, "\n")
	if len(e.lines) == 0 {
		e.lines = []string{""}
	}

	// Reset cursor to start
	e.cursorLine = 0
	e.cursorColumn = 0

	e.updateDisplay()
}

// GetText returns the current text content
func (e *EditableTextView) GetText() string {
	return e.text
}

// SetChangedFunc sets the callback for text changes
func (e *EditableTextView) SetChangedFunc(changed func(string)) {
	e.changeFunc = changed
}

// SetEditable enables or disables editing mode
func (e *EditableTextView) SetEditable(editable bool) {
	e.isEditable = editable
}

// SetPlaceholder sets the placeholder text to show when empty
func (e *EditableTextView) SetPlaceholder(placeholder string) *EditableTextView {
	e.placeholder = placeholder
	e.updateDisplay() // Refresh display to show placeholder if text is empty
	return e
}

// SetPlaceholderTextColor sets the color for placeholder text
func (e *EditableTextView) SetPlaceholderTextColor(color tcell.Color) *EditableTextView {
	e.placeholderColor = color
	e.updateDisplay() // Refresh display to apply new placeholder color
	return e
}

// Note: Input handling is now done through the InputHandler() method
// rather than SetInputCapture, which provides better integration with tview's
// focus system and ensures proper event handling precedence

// Text editing methods (same as before, but simpler)

// insertCharacter inserts a character at the current cursor position
func (e *EditableTextView) insertCharacter(ch rune) {
	if e.cursorLine >= len(e.lines) {
		e.lines = append(e.lines, "")
		e.cursorLine = len(e.lines) - 1
	}

	line := e.lines[e.cursorLine]
	runes := []rune(line)

	// Ensure cursor position is valid in rune space
	if e.cursorColumn > len(runes) {
		e.cursorColumn = len(runes)
	}

	// Insert character at cursor position using rune-based indexing
	newRunes := make([]rune, len(runes)+1)
	copy(newRunes, runes[:e.cursorColumn])
	newRunes[e.cursorColumn] = ch
	copy(newRunes[e.cursorColumn+1:], runes[e.cursorColumn:])

	e.lines[e.cursorLine] = string(newRunes)
	e.cursorColumn++

	e.textChanged()
}

// insertNewline inserts a newline at the current cursor position
func (e *EditableTextView) insertNewline() {
	if e.cursorLine >= len(e.lines) {
		e.lines = append(e.lines, "")
		e.cursorLine = len(e.lines) - 1
	}

	line := e.lines[e.cursorLine]
	runes := []rune(line)

	// Ensure cursor position is valid in rune space
	if e.cursorColumn > len(runes) {
		e.cursorColumn = len(runes)
	}

	// Split line at cursor position using rune-based indexing
	leftPart := string(runes[:e.cursorColumn])
	rightPart := string(runes[e.cursorColumn:])

	// Update current line and insert new line
	e.lines[e.cursorLine] = leftPart
	newLines := make([]string, len(e.lines)+1)
	copy(newLines, e.lines[:e.cursorLine+1])
	newLines[e.cursorLine+1] = rightPart
	copy(newLines[e.cursorLine+2:], e.lines[e.cursorLine+1:])
	e.lines = newLines

	// Move cursor to beginning of new line
	e.cursorLine++
	e.cursorColumn = 0

	e.textChanged()
}

// handleBackspace handles backspace key
func (e *EditableTextView) handleBackspace() {
	if e.cursorColumn > 0 {
		// Remove character before cursor using rune-based indexing
		line := e.lines[e.cursorLine]
		runes := []rune(line)

		if e.cursorColumn <= len(runes) {
			newRunes := make([]rune, len(runes)-1)
			copy(newRunes, runes[:e.cursorColumn-1])
			copy(newRunes[e.cursorColumn-1:], runes[e.cursorColumn:])
			e.lines[e.cursorLine] = string(newRunes)
			e.cursorColumn--
		}
	} else if e.cursorLine > 0 {
		// Join current line with previous line
		prevLine := e.lines[e.cursorLine-1]
		currentLine := e.lines[e.cursorLine]

		// Remove current line
		newLines := make([]string, len(e.lines)-1)
		copy(newLines, e.lines[:e.cursorLine])
		copy(newLines[e.cursorLine:], e.lines[e.cursorLine+1:])
		e.lines = newLines

		// Move cursor to end of previous line (rune-based length)
		e.cursorLine--
		e.cursorColumn = len([]rune(prevLine))

		// Join the lines
		e.lines[e.cursorLine] = prevLine + currentLine
	}

	e.textChanged()
}

// handleDelete handles delete key
func (e *EditableTextView) handleDelete() {
	if e.cursorLine >= len(e.lines) {
		return
	}

	line := e.lines[e.cursorLine]
	runes := []rune(line)

	if e.cursorColumn < len(runes) {
		// Remove character at cursor using rune-based indexing
		newRunes := make([]rune, len(runes)-1)
		copy(newRunes, runes[:e.cursorColumn])
		copy(newRunes[e.cursorColumn:], runes[e.cursorColumn+1:])
		e.lines[e.cursorLine] = string(newRunes)
	} else if e.cursorLine < len(e.lines)-1 {
		// Join with next line
		nextLine := e.lines[e.cursorLine+1]

		// Remove next line
		newLines := make([]string, len(e.lines)-1)
		copy(newLines, e.lines[:e.cursorLine+1])
		copy(newLines[e.cursorLine+1:], e.lines[e.cursorLine+2:])
		e.lines = newLines

		// Join the lines
		e.lines[e.cursorLine] = line + nextLine
	}

	e.textChanged()
}

// Cursor movement methods

// moveCursorUp moves cursor up one line
func (e *EditableTextView) moveCursorUp() {
	if e.cursorLine > 0 {
		e.cursorLine--
		// Clamp column to line length using rune-based length
		lineLength := len([]rune(e.lines[e.cursorLine]))
		if e.cursorColumn > lineLength {
			e.cursorColumn = lineLength
		}
		e.updateDisplay()
	}
}

// moveCursorDown moves cursor down one line
func (e *EditableTextView) moveCursorDown() {
	if e.cursorLine < len(e.lines)-1 {
		e.cursorLine++
		// Clamp column to line length using rune-based length
		lineLength := len([]rune(e.lines[e.cursorLine]))
		if e.cursorColumn > lineLength {
			e.cursorColumn = lineLength
		}
		e.updateDisplay()
	}
}

// moveCursorLeft moves cursor left one character
func (e *EditableTextView) moveCursorLeft() {
	if e.cursorColumn > 0 {
		e.cursorColumn--
		e.updateDisplay()
	} else if e.cursorLine > 0 {
		// Move to end of previous line using rune-based length
		e.cursorLine--
		e.cursorColumn = len([]rune(e.lines[e.cursorLine]))
		e.updateDisplay()
	}
}

// moveCursorRight moves cursor right one character
func (e *EditableTextView) moveCursorRight() {
	if e.cursorLine < len(e.lines) {
		lineLength := len([]rune(e.lines[e.cursorLine]))
		if e.cursorColumn < lineLength {
			e.cursorColumn++
			e.updateDisplay()
		} else if e.cursorLine < len(e.lines)-1 {
			// Move to beginning of next line
			e.cursorLine++
			e.cursorColumn = 0
			e.updateDisplay()
		}
	}
}

// Text management methods

// textChanged updates the text content and calls change callback
func (e *EditableTextView) textChanged() {
	e.text = strings.Join(e.lines, "\n")
	e.updateDisplay()

	// Call change callback if not already updating (prevents loops)
	if e.changeFunc != nil && !e.updating {
		e.changeFunc(e.text)
	}
}

// updateDisplay updates the TextView display with cursor
func (e *EditableTextView) updateDisplay() {
	// Prevent recursive calls
	if e.updating || e.textView == nil {
		return
	}
	e.updating = true
	defer func() {
		e.updating = false
	}()

	// Check if content is empty and we have a placeholder
	isEmpty := len(e.lines) == 1 && e.lines[0] == ""
	if isEmpty && e.placeholder != "" && !e.HasFocus() {
		// Show placeholder text with theme-aware color
		// Use the placeholder color that was set, fallback to dim text
		if e.placeholderColor != tcell.ColorDefault {
			// Apply the theme-aware placeholder color directly to the TextView
			e.textView.SetTextColor(e.placeholderColor)
			e.textView.SetText(e.placeholder)
			// Note: Text color will be restored by theme system when content is shown
		} else {
			// Fallback to grey if no specific color was set
			placeholderText := fmt.Sprintf("[grey]%s[-]", e.placeholder)
			e.textView.SetText(placeholderText)
		}
		return
	}

	// Restore normal text color if we had changed it for placeholder
	// This is important when transitioning from placeholder to content
	if e.originalTextColor != tcell.ColorDefault {
		e.textView.SetTextColor(e.originalTextColor)
	}

	// Create display text with cursor indicator
	displayLines := make([]string, len(e.lines))
	copy(displayLines, e.lines)

	// Add cursor indicator (█ character) at current position when focused
	if e.HasFocus() && e.cursorLine < len(displayLines) {
		line := displayLines[e.cursorLine]
		runes := []rune(line)

		if e.cursorColumn <= len(runes) {
			// Insert cursor character using rune-based indexing
			cursorRune := '█'
			if e.cursorColumn == len(runes) {
				// Cursor at end of line
				newRunes := make([]rune, len(runes)+1)
				copy(newRunes, runes)
				newRunes[len(runes)] = cursorRune
				displayLines[e.cursorLine] = string(newRunes)
			} else {
				// Cursor in middle of line - replace character at cursor position
				newRunes := make([]rune, len(runes))
				copy(newRunes, runes)
				newRunes[e.cursorColumn] = cursorRune
				displayLines[e.cursorLine] = string(newRunes)
			}
		}
	}

	// Update the TextView content (clean delegation)
	displayText := strings.Join(displayLines, "\n")
	e.textView.SetText(displayText)

	// Smart scroll management - only scroll when cursor is outside visible area
	if e.HasFocus() {
		e.smartScroll()
	}
}

// smartScroll implements intelligent scrolling that preserves context
func (e *EditableTextView) smartScroll() {
	if e.textView == nil {
		return
	}

	// Get viewport height
	viewportHeight := e.getViewportHeight()
	if viewportHeight <= 0 {
		// Fallback to basic scrolling if we can't determine viewport height
		e.textView.ScrollTo(e.cursorLine, 0)
		return
	}

	// Get current scroll position
	scrollRow, _ := e.textView.GetScrollOffset()

	// Calculate visible range
	visibleStartLine := scrollRow
	visibleEndLine := scrollRow + viewportHeight - 1

	// Check if cursor is already visible
	if e.cursorLine >= visibleStartLine && e.cursorLine <= visibleEndLine {
		// Cursor is already visible, no need to scroll
		if e.app.logger != nil {
			e.app.logger.Printf("EditableTextView: Cursor line %d already visible (range: %d-%d), no scroll needed",
				e.cursorLine, visibleStartLine, visibleEndLine)
		}
		return
	}

	// Determine optimal scroll position
	var targetScrollRow int

	if e.cursorLine < visibleStartLine {
		// Cursor is above visible area - scroll up but keep some context
		contextLines := viewportHeight / 4 // Show 25% context above cursor
		if contextLines < 1 {
			contextLines = 1
		}
		targetScrollRow = e.cursorLine - contextLines
		if targetScrollRow < 0 {
			targetScrollRow = 0
		}
	} else {
		// Cursor is below visible area - scroll down but keep context
		contextLines := viewportHeight / 3 // Show 33% context below cursor
		if contextLines < 1 {
			contextLines = 1
		}
		targetScrollRow = e.cursorLine - viewportHeight + contextLines + 1
		if targetScrollRow < 0 {
			targetScrollRow = 0
		}
	}

	// Perform the smart scroll
	if e.app.logger != nil {
		e.app.logger.Printf("EditableTextView: Smart scroll from line %d to %d (cursor at %d, viewport height %d)",
			scrollRow, targetScrollRow, e.cursorLine, viewportHeight)
	}
	e.textView.ScrollTo(targetScrollRow, 0)
}

// getViewportHeight calculates the visible text area height
func (e *EditableTextView) getViewportHeight() int {
	if e.textView == nil {
		return 0
	}

	// Get the inner rectangle (content area without borders)
	_, _, _, height := e.textView.GetInnerRect()

	// TextView height represents the number of visible text lines
	if height > 0 {
		return height
	}

	// Fallback: use the full rectangle if GetInnerRect doesn't work
	_, _, _, fullHeight := e.textView.GetRect()

	// Account for potential borders (subtract 2 if bordered)
	if fullHeight > 2 {
		return fullHeight - 2
	}

	return fullHeight
}

// Utility methods

// GetCursorPosition returns current cursor line and column
func (e *EditableTextView) GetCursorPosition() (int, int) {
	return e.cursorLine, e.cursorColumn
}

// SetCursorPosition sets cursor to specified line and column
func (e *EditableTextView) SetCursorPosition(line, column int) {
	if line >= 0 && line < len(e.lines) {
		e.cursorLine = line
		lineLength := len([]rune(e.lines[line]))
		if column >= 0 && column <= lineLength {
			e.cursorColumn = column
		} else {
			e.cursorColumn = lineLength
		}
		e.updateDisplay()
	}
}

// Public methods for external input handling
// These methods are called by the composition panel to handle input directly

// HandleCharInput handles character input from external sources
func (e *EditableTextView) HandleCharInput(char rune) {
	if e.isEditable && unicode.IsPrint(char) {
		e.insertCharacter(char)
	}
}

// HandleEnter handles enter key from external sources
func (e *EditableTextView) HandleEnter() {
	if e.isEditable {
		e.insertNewline()
	}
}

// HandleBackspace handles backspace key from external sources
func (e *EditableTextView) HandleBackspace() {
	if e.isEditable {
		e.handleBackspace()
	}
}

// HandleDelete handles delete key from external sources
func (e *EditableTextView) HandleDelete() {
	if e.isEditable {
		e.handleDelete()
	}
}

// HandleArrowUp handles up arrow key from external sources
func (e *EditableTextView) HandleArrowUp() {
	if e.isEditable {
		e.moveCursorUp()
	}
}

// HandleArrowDown handles down arrow key from external sources
func (e *EditableTextView) HandleArrowDown() {
	if e.isEditable {
		e.moveCursorDown()
	}
}

// HandleArrowLeft handles left arrow key from external sources
func (e *EditableTextView) HandleArrowLeft() {
	if e.isEditable {
		e.moveCursorLeft()
	}
}

// HandleArrowRight handles right arrow key from external sources
func (e *EditableTextView) HandleArrowRight() {
	if e.isEditable {
		e.moveCursorRight()
	}
}
