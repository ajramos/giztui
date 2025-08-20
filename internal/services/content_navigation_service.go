package services

import (
	"context"
	"strings"
	"time"
	"unicode"
)

// ContentNavigationServiceImpl implements the ContentNavigationService interface
type ContentNavigationServiceImpl struct {
	// No dependencies needed for content navigation operations
}

// NewContentNavigationService creates a new content navigation service
func NewContentNavigationService() *ContentNavigationServiceImpl {
	return &ContentNavigationServiceImpl{}
}

// SearchContent searches for a query within content and returns all match positions
func (s *ContentNavigationServiceImpl) SearchContent(ctx context.Context, content string, query string, caseSensitive bool) (*ContentSearchResult, error) {
	start := time.Now()

	if query == "" {
		return &ContentSearchResult{
			Query:         query,
			CaseSensitive: caseSensitive,
			Matches:       []int{},
			MatchCount:    0,
			Content:       content,
			Duration:      time.Since(start),
		}, nil
	}

	var matches []int
	searchText := content
	searchQuery := query

	// Handle case sensitivity
	if !caseSensitive {
		searchText = strings.ToLower(content)
		searchQuery = strings.ToLower(query)
	}

	// Find all matches
	pos := 0
	for {
		index := strings.Index(searchText[pos:], searchQuery)
		if index == -1 {
			break
		}
		actualPos := pos + index
		matches = append(matches, actualPos)
		pos = actualPos + 1 // Move past this match to find overlapping matches
	}

	return &ContentSearchResult{
		Query:         query,
		CaseSensitive: caseSensitive,
		Matches:       matches,
		MatchCount:    len(matches),
		Content:       content,
		Duration:      time.Since(start),
	}, nil
}

// FindNextMatch finds the next match position after currentPosition
func (s *ContentNavigationServiceImpl) FindNextMatch(ctx context.Context, searchResult *ContentSearchResult, currentPosition int) (int, error) {
	if searchResult == nil || searchResult.MatchCount == 0 {
		return -1, nil
	}

	// Find the first match after currentPosition
	for _, pos := range searchResult.Matches {
		if pos > currentPosition {
			return pos, nil
		}
	}

	// If no match found after currentPosition, wrap to first match
	return searchResult.Matches[0], nil
}

// FindPreviousMatch finds the previous match position before currentPosition
func (s *ContentNavigationServiceImpl) FindPreviousMatch(ctx context.Context, searchResult *ContentSearchResult, currentPosition int) (int, error) {
	if searchResult == nil || searchResult.MatchCount == 0 {
		return -1, nil
	}

	// Find the last match before currentPosition (search backwards)
	for i := len(searchResult.Matches) - 1; i >= 0; i-- {
		pos := searchResult.Matches[i]
		if pos < currentPosition {
			return pos, nil
		}
	}

	// If no match found before currentPosition, wrap to last match
	return searchResult.Matches[len(searchResult.Matches)-1], nil
}

// FindNextParagraph finds the next paragraph boundary (empty line or double newline)
func (s *ContentNavigationServiceImpl) FindNextParagraph(ctx context.Context, content string, currentPosition int) (int, error) {
	if currentPosition >= len(content) {
		return len(content), nil
	}

	// Find the next double newline or empty line (true paragraph boundary)
	for i := currentPosition + 1; i < len(content)-1; i++ {
		if content[i] == '\n' {
			// Check for empty line (two consecutive newlines)
			if i+1 < len(content) && content[i+1] == '\n' {
				return i + 2, nil // Return position after the empty line
			}
			// Check for line that only contains whitespace
			lineStart := i + 1
			lineEnd := lineStart
			for lineEnd < len(content) && content[lineEnd] != '\n' {
				lineEnd++
			}
			line := strings.TrimSpace(content[lineStart:lineEnd])
			if line == "" {
				return lineEnd + 1, nil // Return position after the empty line
			}
		}
	}

	// If no paragraph boundaries found, navigate forward by multiple lines (fast navigation)
	linesDown := 0
	targetLines := 10 // Navigate down by 10 lines for "paragraph" navigation

	for i := currentPosition + 1; i < len(content); i++ {
		if content[i] == '\n' {
			linesDown++
			if linesDown >= targetLines {
				// Found 10 lines down, return position after this newline
				if i+1 < len(content) {
					return i + 1, nil
				}
				return len(content), nil
			}
		}
	}

	return len(content), nil // End of content
}

// FindPreviousParagraph finds the previous paragraph boundary
func (s *ContentNavigationServiceImpl) FindPreviousParagraph(ctx context.Context, content string, currentPosition int) (int, error) {
	if currentPosition <= 0 {
		return 0, nil
	}

	// Find the previous double newline or empty line (true paragraph boundary)
	for i := currentPosition - 1; i > 0; i-- {
		if content[i] == '\n' && i > 0 && content[i-1] == '\n' {
			result := i + 1

			// Skip this boundary if it's the same as our current position (we're already at a boundary)
			if result == currentPosition {
				continue
			}

			return result, nil
		}
	}

	// If no paragraph boundaries found, navigate backward by multiple lines (fast navigation)
	linesUp := 0
	targetLines := 10 // Navigate up by 10 lines for "paragraph" navigation

	for i := currentPosition - 1; i >= 0; i-- {
		if content[i] == '\n' {
			linesUp++
			if linesUp >= targetLines {
				// Found 10 lines up, return position after this newline
				return i + 1, nil
			}
		}
	}

	// If we have fewer than 10 lines total, go to beginning
	return 0, nil
}

// FindNextWord finds the next word boundary
func (s *ContentNavigationServiceImpl) FindNextWord(ctx context.Context, content string, currentPosition int) (int, error) {
	if currentPosition >= len(content) {
		return len(content), nil
	}

	// Skip current word
	i := currentPosition
	for i < len(content) && !unicode.IsSpace(rune(content[i])) {
		i++
	}
	// Skip whitespace
	for i < len(content) && unicode.IsSpace(rune(content[i])) {
		i++
	}

	return i, nil
}

// FindPreviousWord finds the previous word boundary
func (s *ContentNavigationServiceImpl) FindPreviousWord(ctx context.Context, content string, currentPosition int) (int, error) {
	if currentPosition <= 0 {
		return 0, nil
	}

	// Skip whitespace backwards
	i := currentPosition - 1
	for i >= 0 && unicode.IsSpace(rune(content[i])) {
		i--
	}
	// Skip current word backwards
	for i >= 0 && !unicode.IsSpace(rune(content[i])) {
		i--
	}

	return i + 1, nil
}

// GetLineFromPosition converts a character position to a line number (1-based)
func (s *ContentNavigationServiceImpl) GetLineFromPosition(ctx context.Context, content string, position int) (int, error) {
	if position < 0 || position > len(content) {
		return 1, nil
	}

	line := 1
	for i := 0; i < position && i < len(content); i++ {
		if content[i] == '\n' {
			line++
		}
	}

	return line, nil
}

// GetPositionFromLine converts a line number (1-based) to character position
func (s *ContentNavigationServiceImpl) GetPositionFromLine(ctx context.Context, content string, line int) (int, error) {
	if line <= 1 {
		return 0, nil
	}

	currentLine := 1
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			currentLine++
			if currentLine == line {
				return i + 1, nil
			}
		}
	}

	return len(content), nil // If line number is beyond content, return end
}

// GetContentLength returns the total length of content in characters
func (s *ContentNavigationServiceImpl) GetContentLength(ctx context.Context, content string) int {
	return len(content)
}
