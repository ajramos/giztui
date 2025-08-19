package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// EnhancedTextView wraps tview.TextView with content navigation and search capabilities
type EnhancedTextView struct {
	*tview.TextView
	
	// Services
	contentNavService services.ContentNavigationService
	
	// Content and search state
	content            string
	currentSearchResult *services.ContentSearchResult
	currentMatchIndex  int  // Current match being highlighted
	
	// Navigation state
	currentPosition    int  // Current cursor position in content
	
	// App reference for keyboard config and error handling
	app *App
}

// NewEnhancedTextView creates a new enhanced text view with navigation capabilities
func NewEnhancedTextView(app *App) *EnhancedTextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetScrollable(true)
		
	enhanced := &EnhancedTextView{
		TextView:          textView,
		contentNavService: nil, // Will be lazily initialized
		app:              app,
		currentPosition:   0,
		currentMatchIndex: -1,
	}
	
	// Set up input capture for content navigation
	enhanced.setupInputCapture()
	
	return enhanced
}

// getContentNavService lazily initializes and returns the content navigation service
func (e *EnhancedTextView) getContentNavService() services.ContentNavigationService {
	if e.contentNavService == nil {
		e.contentNavService = e.app.GetContentNavService()
	}
	return e.contentNavService
}

// hasContentNavService checks if the content navigation service is available
func (e *EnhancedTextView) hasContentNavService() bool {
	return e.getContentNavService() != nil
}

// SetContent sets the content and resets navigation state
func (e *EnhancedTextView) SetContent(content string) *EnhancedTextView {
	e.content = content
	e.currentPosition = 0
	e.currentMatchIndex = -1
	e.currentSearchResult = nil
	e.TextView.SetText(content)
	return e
}

// GetContent returns the current content
func (e *EnhancedTextView) GetContent() string {
	return e.content
}

// HasActiveSearch returns true if there's an active search with matches
func (e *EnhancedTextView) HasActiveSearch() bool {
	return e.currentSearchResult != nil && e.currentSearchResult.MatchCount > 0
}

// setupInputCapture configures keyboard shortcuts for content navigation and search
func (e *EnhancedTextView) setupInputCapture() {
	e.TextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Only handle navigation if we're focused on text content
		if e.app.currentFocus != "text" {
			return event
		}
		
		key := event.Key()
		char := event.Rune()
		
		// Handle different key combinations
		switch {
		// Content search: /
		case char == '/' && e.app.Keys.ContentSearch == "/":
			e.startContentSearchCommand()
			return nil
			
		// Search next: n
		case char == 'n' && e.app.Keys.SearchNext == "n":
			e.searchNext()
			return nil
			
		// Search previous: N
		case char == 'N' && e.app.Keys.SearchPrev == "N":
			e.searchPrevious()
			return nil
			
		// Fast navigation up: Ctrl+K
		case key == tcell.KeyCtrlK && e.app.Keys.FastUp == "ctrl+k":
			e.fastNavigateUp()
			return nil
			
		// Fast navigation down: Ctrl+J
		case key == tcell.KeyCtrlJ && e.app.Keys.FastDown == "ctrl+j":
			e.fastNavigateDown()
			return nil
			
		// Word navigation left: Ctrl+H
		case key == tcell.KeyCtrlH && e.app.Keys.WordLeft == "ctrl+h":
			e.wordNavigateLeft()
			return nil
			
		// Word navigation right: Ctrl+L  
		case key == tcell.KeyCtrlL && e.app.Keys.WordRight == "ctrl+l":
			e.wordNavigateRight()
			return nil
			
		// Note: VIM navigation (gg, G) is handled at App level in handleVimNavigation
		// These keys are not handled here to avoid conflicts
			
		// ESC key: clear search highlights
		case key == tcell.KeyEscape:
			e.clearSearch()
			// Don't return nil - let ESC propagate for other handlers
		}
		
		// Return the event to allow normal text view navigation
		return event
	})
}

// startContentSearchCommand opens the command bar with search prefix
func (e *EnhancedTextView) startContentSearchCommand() {
	if !e.hasContentNavService() {
		go func() {
			e.app.GetErrorHandler().ShowError(context.Background(), "Content navigation service not available")
		}()
		return
	}
	
	if e.content == "" {
		go func() {
			e.app.GetErrorHandler().ShowWarning(context.Background(), "No content to search")
		}()
		return
	}
	
	// Open command bar with content search prefix - using "/" for content search
	e.app.showCommandBarWithPrefix("/")
}

// openContentSearchOverlay creates a search overlay for content search
func (e *EnhancedTextView) openContentSearchOverlay() {
	title := "🔍 Search Content"
	
	// Create input field for search query
	input := tview.NewInputField().
		SetLabel("🔍 ").
		SetLabelColor(tcell.ColorYellow).
		SetFieldWidth(0).
		SetPlaceholder("Enter search term...")
	
	// Store input reference for cleanup
	e.app.views["contentSearchInput"] = input
	
	// Create help text
	help := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	help.SetTextColor(tcell.ColorGray)
	help.SetText("Enter=search, ESC=cancel | After search: n=next, N=previous")
	
	// Create the overlay container
	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).SetTitle(title).SetTitleColor(tcell.ColorYellow)
	
	// Layout: spacer, input, help, spacer
	topSpacer := tview.NewBox()
	bottomSpacer := tview.NewBox()
	box.AddItem(topSpacer, 0, 1, false)
	box.AddItem(input, 1, 0, true)
	box.AddItem(help, 1, 0, false)
	box.AddItem(bottomSpacer, 0, 1, false)
	
	// Set up input handling
	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			query := strings.TrimSpace(input.GetText())
			if query != "" {
				e.performContentSearch(query)
			}
			e.closeContentSearchOverlay()
		case tcell.KeyEscape:
			e.closeContentSearchOverlay()
		}
	})
	
	// Handle input capture for additional controls
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.closeContentSearchOverlay()
			return nil
		}
		return event
	})
	
	// Show the overlay by adding it to the content split area temporarily
	if split, ok := e.app.views["contentSplit"].(*tview.Flex); ok {
		// Store the current labels view if any
		e.app.previousFocus = e.app.currentFocus
		
		// Hide any existing side panels
		if e.app.labelsVisible && e.app.labelsView != nil {
			split.RemoveItem(e.app.labelsView)
		}
		
		// Add the search overlay
		e.app.labelsView = box
		split.AddItem(e.app.labelsView, 0, 1, true)
		split.ResizeItem(e.app.labelsView, 0, 1)
		
		// Set focus and update state
		e.app.currentFocus = "contentSearch"
		e.app.labelsVisible = true
		e.app.updateFocusIndicators("contentSearch")
		e.app.SetFocus(input)
	}
}

// searchNext navigates to the next search match
func (e *EnhancedTextView) searchNext() {
	if !e.hasContentNavService() {
		go func() {
			e.app.GetErrorHandler().ShowError(context.Background(), "Content navigation service not available")
		}()
		return
	}
	
	if e.currentSearchResult == nil || e.currentSearchResult.MatchCount == 0 {
		go func() {
			e.app.GetErrorHandler().ShowWarning(context.Background(), "No active search - use / to search")
		}()
		return
	}
	
	ctx := context.Background()
	nextPos, err := e.getContentNavService().FindNextMatch(ctx, e.currentSearchResult, e.currentPosition)
	if err != nil {
		go func() {
			e.app.GetErrorHandler().ShowError(ctx, "Failed to find next match")
		}()
		return
	}
	
	if nextPos != -1 {
		e.currentPosition = nextPos
		e.updateMatchIndex()
		e.scrollToPosition(nextPos)
		e.showMatchStatus()
	}
}

// searchPrevious navigates to the previous search match
func (e *EnhancedTextView) searchPrevious() {
	if !e.hasContentNavService() {
		go func() {
			e.app.GetErrorHandler().ShowError(context.Background(), "Content navigation service not available")
		}()
		return
	}
	
	if e.currentSearchResult == nil || e.currentSearchResult.MatchCount == 0 {
		go func() {
			e.app.GetErrorHandler().ShowWarning(context.Background(), "No active search - use / to search")
		}()
		return
	}
	
	ctx := context.Background()
	prevPos, err := e.getContentNavService().FindPreviousMatch(ctx, e.currentSearchResult, e.currentPosition)
	if err != nil {
		go func() {
			e.app.GetErrorHandler().ShowError(ctx, "Failed to find previous match")
		}()
		return
	}
	
	if prevPos != -1 {
		e.currentPosition = prevPos
		e.updateMatchIndex()
		e.scrollToPosition(prevPos)
		e.showMatchStatus()
	}
}

// fastNavigateUp navigates up by paragraphs
func (e *EnhancedTextView) fastNavigateUp() {
	if !e.hasContentNavService() {
		return // Fail silently for navigation functions
	}
	
	ctx := context.Background()
	
	if e.app.logger != nil {
		e.app.logger.Printf("FAST NAV UP: currentPosition=%d, contentLength=%d", e.currentPosition, len(e.content))
	}
	
	prevParagraph, err := e.getContentNavService().FindPreviousParagraph(ctx, e.content, e.currentPosition)
	if err != nil {
		if e.app.logger != nil {
			e.app.logger.Printf("FastNavigateUp error: %v", err)
		}
		return
	}
	
	if e.app.logger != nil {
		e.app.logger.Printf("FAST NAV UP: FindPreviousParagraph returned %d", prevParagraph)
	}
	
	if prevParagraph != e.currentPosition {
		oldPos := e.currentPosition
		e.currentPosition = prevParagraph
		e.scrollToPosition(prevParagraph)
		
		// Get line numbers for better feedback
		oldLine, _ := e.getContentNavService().GetLineFromPosition(ctx, e.content, oldPos)
		newLine, _ := e.getContentNavService().GetLineFromPosition(ctx, e.content, prevParagraph)
		
		go func() {
			e.app.GetErrorHandler().ShowInfo(ctx, fmt.Sprintf("↑ Paragraph up (line %d → %d)", oldLine, newLine))
		}()
	} else {
		// At boundary - provide feedback
		go func() {
			e.app.GetErrorHandler().ShowInfo(ctx, "↑ Already at beginning")
		}()
	}
}

// fastNavigateDown navigates down by paragraphs
func (e *EnhancedTextView) fastNavigateDown() {
	if !e.hasContentNavService() {
		return // Fail silently for navigation functions
	}
	
	ctx := context.Background()
	nextParagraph, err := e.getContentNavService().FindNextParagraph(ctx, e.content, e.currentPosition)
	if err != nil {
		return
	}
	
	if nextParagraph != e.currentPosition {
		e.currentPosition = nextParagraph
		e.scrollToPosition(nextParagraph)
		go func() {
			e.app.GetErrorHandler().ShowInfo(ctx, "Fast navigation down")
		}()
	}
}

// wordNavigateLeft navigates left by words
func (e *EnhancedTextView) wordNavigateLeft() {
	if !e.hasContentNavService() {
		return // Fail silently for navigation functions
	}
	
	ctx := context.Background()
	prevWord, err := e.getContentNavService().FindPreviousWord(ctx, e.content, e.currentPosition)
	if err != nil {
		if e.app.logger != nil {
			e.app.logger.Printf("WordNavigateLeft error: %v", err)
		}
		return
	}
	
	if prevWord != e.currentPosition {
		e.currentPosition = prevWord
		e.scrollToPosition(prevWord)
		
		// Get line for better feedback
		newLine, _ := e.getContentNavService().GetLineFromPosition(ctx, e.content, prevWord)
		
		// Show brief feedback with current position
		go func() {
			e.app.GetErrorHandler().ShowInfo(ctx, fmt.Sprintf("← Word (line %d)", newLine))
		}()
	} else {
		go func() {
			e.app.GetErrorHandler().ShowInfo(ctx, "← Beginning of content")
		}()
	}
}

// wordNavigateRight navigates right by words
func (e *EnhancedTextView) wordNavigateRight() {
	if !e.hasContentNavService() {
		return // Fail silently for navigation functions
	}
	
	ctx := context.Background()
	nextWord, err := e.getContentNavService().FindNextWord(ctx, e.content, e.currentPosition)
	if err != nil {
		if e.app.logger != nil {
			e.app.logger.Printf("WordNavigateRight error: %v", err)
		}
		return
	}
	
	if nextWord != e.currentPosition {
		e.currentPosition = nextWord
		e.scrollToPosition(nextWord)
		
		// Get line for better feedback
		newLine, _ := e.getContentNavService().GetLineFromPosition(ctx, e.content, nextWord)
		
		// Show brief feedback with current position
		go func() {
			e.app.GetErrorHandler().ShowInfo(ctx, fmt.Sprintf("→ Word (line %d)", newLine))
		}()
	} else {
		go func() {
			e.app.GetErrorHandler().ShowInfo(ctx, "→ End of content")
		}()
	}
}

// handleGotoTopSequence handles vim-style gg sequence
func (e *EnhancedTextView) handleGotoTopSequence() {
	// Simple implementation for now - go to top immediately
	// TODO: Implement proper vim-style sequence handling with timeout
	e.gotoTop()
}

// GotoTop navigates to the beginning of content (public method)
func (e *EnhancedTextView) GotoTop() {
	e.gotoTop()
}

// gotoTop navigates to the beginning of content
func (e *EnhancedTextView) gotoTop() {
	e.currentPosition = 0
	e.scrollToPosition(0)
	go func() {
		e.app.GetErrorHandler().ShowInfo(context.Background(), "Top of content")
	}()
}

// GotoBottom navigates to the end of content (public method)
func (e *EnhancedTextView) GotoBottom() {
	e.gotoBottom()
}

// gotoBottom navigates to the end of content
func (e *EnhancedTextView) gotoBottom() {
	if !e.hasContentNavService() {
		return // Fail silently for navigation functions
	}
	
	contentLength := e.getContentNavService().GetContentLength(context.Background(), e.content)
	oldPos := e.currentPosition
	
	// Set position to the last actual character, not past the end
	if contentLength > 0 {
		e.currentPosition = contentLength - 1
	} else {
		e.currentPosition = 0
	}
	
	e.scrollToPosition(e.currentPosition)
	
	if e.app.logger != nil {
		e.app.logger.Printf("GOTO BOTTOM: position %d → %d (contentLength=%d)", oldPos, e.currentPosition, contentLength)
	}
	
	// Also scroll to the actual last line of the TextView for better UX
	lines := len(strings.Split(e.content, "\n"))
	if lines > 0 {
		e.TextView.ScrollToEnd()
		if e.app.logger != nil {
			e.app.logger.Printf("GOTO BOTTOM: Scrolled TextView to end, total lines=%d", lines)
		}
	}
	
	go func() {
		e.app.GetErrorHandler().ShowInfo(context.Background(), fmt.Sprintf("Bottom of content (pos %d)", e.currentPosition))
	}()
}

// clearSearch clears current search results and highlights
func (e *EnhancedTextView) clearSearch() {
	if e.currentSearchResult != nil {
		e.currentSearchResult = nil
		e.currentMatchIndex = -1
		// Refresh content without highlights
		e.TextView.SetText(e.content)
		go func() {
			e.app.GetErrorHandler().ShowInfo(context.Background(), "Search cleared")
		}()
	}
}

// scrollToPosition scrolls to a specific character position in the content
func (e *EnhancedTextView) scrollToPosition(position int) {
	if position < 0 || position > len(e.content) {
		return
	}
	
	if !e.hasContentNavService() {
		return // Fail silently for navigation functions
	}
	
	ctx := context.Background()
	line, err := e.getContentNavService().GetLineFromPosition(ctx, e.content, position)
	if err != nil {
		return
	}
	
	// Scroll to the line (tview uses 0-based indexing)
	e.TextView.ScrollTo(line-1, 0)
}

// updateMatchIndex updates the current match index based on current position
func (e *EnhancedTextView) updateMatchIndex() {
	if e.currentSearchResult == nil {
		return
	}
	
	for i, pos := range e.currentSearchResult.Matches {
		if pos == e.currentPosition {
			e.currentMatchIndex = i
			return
		}
	}
}

// showMatchStatus shows current search match status
func (e *EnhancedTextView) showMatchStatus() {
	if e.currentSearchResult == nil || e.currentSearchResult.MatchCount == 0 {
		return
	}
	
	matchNum := e.currentMatchIndex + 1
	totalMatches := e.currentSearchResult.MatchCount
	query := e.currentSearchResult.Query
	
	go func() {
		e.app.GetErrorHandler().ShowInfo(context.Background(), 
			fmt.Sprintf("Match %d/%d for '%s'", matchNum, totalMatches, query))
	}()
}

// closeContentSearchOverlay closes the search overlay and restores focus to text view
func (e *EnhancedTextView) closeContentSearchOverlay() {
	if split, ok := e.app.views["contentSplit"].(*tview.Flex); ok {
		// Remove the search overlay
		if e.app.labelsView != nil {
			split.RemoveItem(e.app.labelsView)
		}
		
		// Clear the search overlay state
		e.app.labelsView = nil
		e.app.labelsVisible = false
		
		// Restore focus to the text view
		e.app.currentFocus = e.app.previousFocus
		if e.app.currentFocus == "" {
			e.app.currentFocus = "text"
		}
		
		// Update focus indicators and set focus
		e.app.updateFocusIndicators(e.app.currentFocus)
		e.app.SetFocus(e.TextView)
		
		// Clean up the input reference
		delete(e.app.views, "contentSearchInput")
	}
}

// performContentSearch executes the search and highlights results in the content
func (e *EnhancedTextView) performContentSearch(query string) {
	if !e.hasContentNavService() {
		go func() {
			e.app.GetErrorHandler().ShowError(context.Background(), "Content navigation service not available")
		}()
		return
	}
	
	if query == "" {
		go func() {
			e.app.GetErrorHandler().ShowWarning(context.Background(), "Empty search query")
		}()
		return
	}

	ctx := context.Background()
	
	// Perform the search using the content navigation service
	searchResult, err := e.getContentNavService().SearchContent(ctx, e.content, query, false) // Default to case insensitive
	if err != nil {
		go func() {
			e.app.GetErrorHandler().ShowError(ctx, "Search failed: "+err.Error())
		}()
		return
	}

	// Store search results
	e.currentSearchResult = searchResult
	e.currentMatchIndex = -1

	if searchResult.MatchCount == 0 {
		go func() {
			e.app.GetErrorHandler().ShowWarning(ctx, fmt.Sprintf("No matches found for '%s'", query))
		}()
		return
	}

	// Navigate to first match
	firstMatch, err := e.getContentNavService().FindNextMatch(ctx, searchResult, -1)
	if err != nil {
		go func() {
			e.app.GetErrorHandler().ShowError(ctx, "Failed to navigate to first match")
		}()
		return
	}

	if firstMatch != -1 {
		e.currentPosition = firstMatch
		e.updateMatchIndex()
		e.scrollToPosition(firstMatch)
		
		// Highlight the search results in the content
		e.highlightSearchResults(query, searchResult.Matches)
		
		// Show search status
		go func() {
			e.app.GetErrorHandler().ShowSuccess(ctx, 
				fmt.Sprintf("Found %d matches for '%s'", searchResult.MatchCount, query))
		}()
		
		// Show current match status
		e.showMatchStatus()
	}
}

// highlightSearchResults highlights all search matches in the displayed content
func (e *EnhancedTextView) highlightSearchResults(query string, matches []int) {
	if len(matches) == 0 {
		return
	}

	// Create highlighted content with tview color tags
	highlightedContent := e.content
	queryLen := len(query)
	
	// Process matches in reverse order to avoid position shifts
	for i := len(matches) - 1; i >= 0; i-- {
		pos := matches[i]
		if pos+queryLen <= len(e.content) {
			// Extract the actual text at this position (preserve original case)
			actualText := e.content[pos : pos+queryLen]
			
			// Wrap with tview highlight colors
			highlighted := fmt.Sprintf("[black:yellow:b]%s[white:-:-]", actualText)
			
			// Replace in the content
			highlightedContent = highlightedContent[:pos] + highlighted + highlightedContent[pos+queryLen:]
		}
	}
	
	// Update the text view with highlighted content
	e.TextView.SetText(highlightedContent)
}