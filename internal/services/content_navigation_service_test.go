package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data constants
const (
	testContent = `Hello World!
This is a test document.

This is paragraph two.
With multiple lines.

Final paragraph here.`

	searchableContent = `The quick brown fox jumps over the lazy dog.
The quick brown fox jumps over the lazy dog.
Another line with different content.
The fox is brown and quick.`

	emailContent = `From: test@example.com
Subject: Test Email

Dear User,

This is a test email message with multiple paragraphs.
Each paragraph contains important information.

Best regards,
Test User

---
Original Message
From: original@example.com
Subject: Re: Test

This is a quoted message with more content.
Multiple lines here as well.`
)

func TestNewContentNavigationService(t *testing.T) {
	service := NewContentNavigationService()
	assert.NotNil(t, service)
	assert.IsType(t, &ContentNavigationServiceImpl{}, service)
}

func TestContentNavigationService_SearchContent_EmptyQuery(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	result, err := service.SearchContent(ctx, testContent, "", false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "", result.Query)
	assert.False(t, result.CaseSensitive)
	assert.Equal(t, 0, result.MatchCount)
	assert.Empty(t, result.Matches)
	assert.Equal(t, testContent, result.Content)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestContentNavigationService_SearchContent_CaseSensitive(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name          string
		content       string
		query         string
		caseSensitive bool
		expectedCount int
		expectedPos   []int
	}{
		{
			name:          "case sensitive match",
			content:       "Hello hello HELLO",
			query:         "Hello",
			caseSensitive: true,
			expectedCount: 1,
			expectedPos:   []int{0},
		},
		{
			name:          "case insensitive match",
			content:       "Hello hello HELLO",
			query:         "Hello",
			caseSensitive: false,
			expectedCount: 3,
			expectedPos:   []int{0, 6, 12},
		},
		{
			name:          "case sensitive no match",
			content:       "hello world",
			query:         "Hello",
			caseSensitive: true,
			expectedCount: 0,
			expectedPos:   nil,
		},
		{
			name:          "case insensitive with match",
			content:       "hello world",
			query:         "Hello",
			caseSensitive: false,
			expectedCount: 1,
			expectedPos:   []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.SearchContent(ctx, tt.content, tt.query, tt.caseSensitive)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.query, result.Query)
			assert.Equal(t, tt.caseSensitive, result.CaseSensitive)
			assert.Equal(t, tt.expectedCount, result.MatchCount)
			assert.Equal(t, tt.expectedPos, result.Matches)
			assert.Equal(t, tt.content, result.Content)
		})
	}
}

func TestContentNavigationService_SearchContent_OverlappingMatches(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Test overlapping pattern matches (e.g., "aaa" in "aaaa")
	result, err := service.SearchContent(ctx, "aaaa", "aa", false)

	assert.NoError(t, err)
	assert.Equal(t, 3, result.MatchCount) // positions 0, 1, 2
	assert.Equal(t, []int{0, 1, 2}, result.Matches)
}

func TestContentNavigationService_SearchContent_MultipleOccurrences(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	result, err := service.SearchContent(ctx, searchableContent, "fox", false)

	assert.NoError(t, err)
	assert.Equal(t, 3, result.MatchCount)
	// Should find "fox" in first two lines and in "The fox is brown"
	expectedPositions := []int{16, 61, 131}
	assert.Equal(t, expectedPositions, result.Matches)
}

func TestContentNavigationService_SearchContent_NoMatches(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	result, err := service.SearchContent(ctx, testContent, "nonexistent", false)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.MatchCount)
	assert.Empty(t, result.Matches)
}

func TestContentNavigationService_FindNextMatch(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Create search result with multiple matches
	searchResult := &ContentSearchResult{
		Matches:    []int{5, 15, 25, 35},
		MatchCount: 4,
	}

	tests := []struct {
		name            string
		currentPos      int
		expectedNextPos int
	}{
		{"from beginning", 0, 5},
		{"from middle", 10, 15},
		{"from between matches", 20, 25},
		{"wrap to beginning", 40, 5}, // Beyond last match, wraps to first
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextPos, err := service.FindNextMatch(ctx, searchResult, tt.currentPos)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedNextPos, nextPos)
		})
	}
}

func TestContentNavigationService_FindNextMatch_EmptyResult(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name         string
		searchResult *ContentSearchResult
	}{
		{"nil search result", nil},
		{"empty matches", &ContentSearchResult{Matches: []int{}, MatchCount: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextPos, err := service.FindNextMatch(ctx, tt.searchResult, 10)
			assert.NoError(t, err)
			assert.Equal(t, -1, nextPos)
		})
	}
}

func TestContentNavigationService_FindPreviousMatch(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Create search result with multiple matches
	searchResult := &ContentSearchResult{
		Matches:    []int{5, 15, 25, 35},
		MatchCount: 4,
	}

	tests := []struct {
		name            string
		currentPos      int
		expectedPrevPos int
	}{
		{"from end", 40, 35},
		{"from middle", 20, 15},
		{"from between matches", 18, 15},
		{"wrap to end", 3, 35}, // Before first match, wraps to last
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevPos, err := service.FindPreviousMatch(ctx, searchResult, tt.currentPos)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPrevPos, prevPos)
		})
	}
}

func TestContentNavigationService_FindPreviousMatch_EmptyResult(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name         string
		searchResult *ContentSearchResult
	}{
		{"nil search result", nil},
		{"empty matches", &ContentSearchResult{Matches: []int{}, MatchCount: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevPos, err := service.FindPreviousMatch(ctx, tt.searchResult, 10)
			assert.NoError(t, err)
			assert.Equal(t, -1, prevPos)
		})
	}
}

func TestContentNavigationService_FindNextParagraph(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name        string
		content     string
		currentPos  int
		expectedPos int
	}{
		{
			name:        "next paragraph with double newline",
			content:     "Para 1\n\nPara 2\n\nPara 3",
			currentPos:  0,
			expectedPos: 8, // After "Para 1\n\n"
		},
		{
			name:        "next paragraph with empty line",
			content:     "Para 1\n   \nPara 2",
			currentPos:  0,
			expectedPos: 11, // After empty line
		},
		{
			name:        "no paragraph boundaries - use line count",
			content:     strings.Repeat("Line\n", 20),
			currentPos:  0,
			expectedPos: 50, // After 10 lines (5 chars per line)
		},
		{
			name:        "at end of content",
			content:     "Short content",
			currentPos:  12,
			expectedPos: 13, // End of content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextPos, err := service.FindNextParagraph(ctx, tt.content, tt.currentPos)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPos, nextPos)
		})
	}
}

func TestContentNavigationService_FindPreviousParagraph(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name        string
		content     string
		currentPos  int
		expectedPos int
	}{
		{
			name:        "previous paragraph with double newline",
			content:     "Para 1\n\nPara 2\n\nPara 3",
			currentPos:  20,
			expectedPos: 16, // Start of "Para 3"
		},
		{
			name:        "at beginning",
			content:     "Para 1\n\nPara 2",
			currentPos:  0,
			expectedPos: 0,
		},
		{
			name:        "no paragraph boundaries - use line count",
			content:     strings.Repeat("Line\n", 20),
			currentPos:  80,
			expectedPos: 35, // 10 lines back from position 80
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevPos, err := service.FindPreviousParagraph(ctx, tt.content, tt.currentPos)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPos, prevPos)
		})
	}
}

func TestContentNavigationService_FindNextWord(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name        string
		content     string
		currentPos  int
		expectedPos int
	}{
		{
			name:        "basic word navigation",
			content:     "hello world test",
			currentPos:  0,
			expectedPos: 6, // Start of "world"
		},
		{
			name:        "skip whitespace",
			content:     "word1   word2",
			currentPos:  0,
			expectedPos: 8, // Start of "word2"
		},
		{
			name:        "at end of content",
			content:     "hello world",
			currentPos:  10,
			expectedPos: 11, // End of content
		},
		{
			name:        "multiple spaces",
			content:     "a    b",
			currentPos:  0,
			expectedPos: 5, // Start of "b"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextPos, err := service.FindNextWord(ctx, tt.content, tt.currentPos)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPos, nextPos)
		})
	}
}

func TestContentNavigationService_FindPreviousWord(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name        string
		content     string
		currentPos  int
		expectedPos int
	}{
		{
			name:        "basic previous word",
			content:     "hello world test",
			currentPos:  12, // In "test"
			expectedPos: 6,  // Start of "world"
		},
		{
			name:        "skip whitespace backwards",
			content:     "word1   word2",
			currentPos:  12, // At end of "word2"
			expectedPos: 8,  // Start of "word2" (one word back)
		},
		{
			name:        "at beginning",
			content:     "hello world",
			currentPos:  0,
			expectedPos: 0,
		},
		{
			name:        "multiple spaces backwards",
			content:     "a    b",
			currentPos:  5,
			expectedPos: 0, // Start of "a"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevPos, err := service.FindPreviousWord(ctx, tt.content, tt.currentPos)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPos, prevPos)
		})
	}
}

func TestContentNavigationService_GetLineFromPosition(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	content := "Line 1\nLine 2\nLine 3\nLine 4"

	tests := []struct {
		name         string
		position     int
		expectedLine int
	}{
		{"beginning", 0, 1},
		{"end of first line", 6, 1},
		{"start of second line", 7, 2},
		{"middle of second line", 10, 2},
		{"start of third line", 14, 3},
		{"end of content", 27, 4},
		{"beyond content", 100, 1},
		{"negative position", -5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := service.GetLineFromPosition(ctx, content, tt.position)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLine, line)
		})
	}
}

func TestContentNavigationService_GetPositionFromLine(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	content := "Line 1\nLine 2\nLine 3\nLine 4"

	tests := []struct {
		name             string
		line             int
		expectedPosition int
	}{
		{"first line", 1, 0},
		{"second line", 2, 7},  // After "Line 1\n"
		{"third line", 3, 14},  // After "Line 1\nLine 2\n"
		{"fourth line", 4, 21}, // After "Line 1\nLine 2\nLine 3\n"
		{"beyond content", 10, 27},
		{"line zero", 0, 0},
		{"negative line", -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			position, err := service.GetPositionFromLine(ctx, content, tt.line)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPosition, position)
		})
	}
}

func TestContentNavigationService_GetContentLength(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	tests := []struct {
		name           string
		content        string
		expectedLength int
	}{
		{"empty string", "", 0},
		{"single character", "a", 1},
		{"multiline content", "Line 1\nLine 2", 13},
		{"unicode content", "Hello 世界", len("Hello 世界")},
		{"email content", emailContent, len(emailContent)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length := service.GetContentLength(ctx, tt.content)
			assert.Equal(t, tt.expectedLength, length)
		})
	}
}

// Integration tests combining multiple operations
func TestContentNavigationService_SearchAndNavigate_Integration(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	content := "The quick brown fox. The lazy dog. The end."

	// Search for "The"
	searchResult, err := service.SearchContent(ctx, content, "The", true)
	require.NoError(t, err)
	require.Equal(t, 3, searchResult.MatchCount)

	// Navigate through matches
	pos1, err := service.FindNextMatch(ctx, searchResult, -1)
	assert.NoError(t, err)
	assert.Equal(t, searchResult.Matches[0], pos1) // First "The"

	pos2, err := service.FindNextMatch(ctx, searchResult, pos1)
	assert.NoError(t, err)
	assert.Equal(t, searchResult.Matches[1], pos2) // Second "The"

	pos3, err := service.FindNextMatch(ctx, searchResult, pos2)
	assert.NoError(t, err)
	assert.Equal(t, searchResult.Matches[2], pos3) // Third "The"

	// Navigate back
	prevPos, err := service.FindPreviousMatch(ctx, searchResult, pos3)
	assert.NoError(t, err)
	assert.Equal(t, searchResult.Matches[1], prevPos) // Back to second "The"
}

func TestContentNavigationService_EmailContentNavigation_Integration(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Test with realistic email content
	searchResult, err := service.SearchContent(ctx, emailContent, "message", false)
	require.NoError(t, err)
	require.Greater(t, searchResult.MatchCount, 0)

	// Test paragraph navigation
	firstMatchPos := searchResult.Matches[0]
	line, err := service.GetLineFromPosition(ctx, emailContent, firstMatchPos)
	assert.NoError(t, err)
	assert.Greater(t, line, 0)

	// Test word navigation around the match
	nextWordPos, err := service.FindNextWord(ctx, emailContent, firstMatchPos)
	assert.NoError(t, err)
	assert.Greater(t, nextWordPos, firstMatchPos)

	prevWordPos, err := service.FindPreviousWord(ctx, emailContent, firstMatchPos)
	assert.NoError(t, err)
	assert.Less(t, prevWordPos, firstMatchPos)
}

// Edge case tests
func TestContentNavigationService_EdgeCases(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	t.Run("empty content", func(t *testing.T) {
		result, err := service.SearchContent(ctx, "", "test", false)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.MatchCount)

		pos, err := service.FindNextParagraph(ctx, "", 0)
		assert.NoError(t, err)
		assert.Equal(t, 0, pos)
	})

	t.Run("single character content", func(t *testing.T) {
		content := "a"

		nextWord, err := service.FindNextWord(ctx, content, 0)
		assert.NoError(t, err)
		assert.Equal(t, 1, nextWord)

		prevWord, err := service.FindPreviousWord(ctx, content, 1)
		assert.NoError(t, err)
		assert.Equal(t, 0, prevWord)
	})

	t.Run("only whitespace content", func(t *testing.T) {
		content := "   \n\n   \n"

		nextPara, err := service.FindNextParagraph(ctx, content, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, nextPara, 0)

		length := service.GetContentLength(ctx, content)
		assert.Equal(t, len(content), length)
	})
}

// Performance benchmarks
func BenchmarkContentNavigationService_SearchContent(b *testing.B) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Create large content for benchmarking
	largeContent := strings.Repeat(searchableContent+"\n", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.SearchContent(ctx, largeContent, "fox", false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContentNavigationService_FindNextMatch(b *testing.B) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Create search result with many matches
	matches := make([]int, 1000)
	for i := range matches {
		matches[i] = i * 10
	}
	searchResult := &ContentSearchResult{
		Matches:    matches,
		MatchCount: len(matches),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.FindNextMatch(ctx, searchResult, 500)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContentNavigationService_FindNextParagraph(b *testing.B) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Create content with many paragraphs
	paragraphs := make([]string, 1000)
	for i := range paragraphs {
		paragraphs[i] = "This is paragraph " + string(rune(i))
	}
	content := strings.Join(paragraphs, "\n\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.FindNextParagraph(ctx, content, 1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContentNavigationService_GetLineFromPosition(b *testing.B) {
	service := NewContentNavigationService()
	ctx := context.Background()

	// Create content with many lines
	lines := make([]string, 10000)
	for i := range lines {
		lines[i] = "This is line number " + string(rune(i))
	}
	content := strings.Join(lines, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetLineFromPosition(ctx, content, 50000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Context cancellation tests
func TestContentNavigationService_ContextCancellation(t *testing.T) {
	service := NewContentNavigationService()

	t.Run("search with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Service should still work even with cancelled context
		// (current implementation doesn't check context, which is acceptable for these operations)
		result, err := service.SearchContent(ctx, testContent, "test", false)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("navigation with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		searchResult := &ContentSearchResult{
			Matches:    []int{5, 15, 25},
			MatchCount: 3,
		}

		pos, err := service.FindNextMatch(ctx, searchResult, 10)
		assert.NoError(t, err)
		assert.Equal(t, 15, pos)
	})
}

// Validation tests for ContentSearchResult structure
func TestContentSearchResult_Validation(t *testing.T) {
	service := NewContentNavigationService()
	ctx := context.Background()

	result, err := service.SearchContent(ctx, searchableContent, "fox", true)
	require.NoError(t, err)

	// Verify all required fields are populated
	assert.Equal(t, "fox", result.Query)
	assert.True(t, result.CaseSensitive)
	assert.Equal(t, len(result.Matches), result.MatchCount)
	assert.Equal(t, searchableContent, result.Content)
	assert.Greater(t, result.Duration, time.Duration(0))

	// Verify matches are in ascending order
	for i := 1; i < len(result.Matches); i++ {
		assert.Greater(t, result.Matches[i], result.Matches[i-1])
	}

	// Verify all matches are within content bounds
	for _, pos := range result.Matches {
		assert.GreaterOrEqual(t, pos, 0)
		assert.Less(t, pos, len(searchableContent))
	}
}
