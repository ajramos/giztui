package services

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//go:embed inbox_analyzer_prompt.txt
var defaultAnalyzerPrompt string

// analyzerRawCategory mirrors the JSON the LLM returns for one category.
type analyzerRawCategory struct {
	Name        string `json:"name"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Label       string `json:"label"`
	Messages    []int  `json:"messages"`
}

type analyzerRawResponse struct {
	Categories   []analyzerRawCategory `json:"categories"`
	ReadManually []int                 `json:"read_manually"`
}

// extractJSONObject returns the first balanced {...} object in s, stripping markdown
// fences and surrounding prose. Returns "" if no object is found.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// parseAnalyzerResponse parses an LLM batch result into categories with concrete message
// IDs, plus the list of per-batch local indices the LLM put in "read_manually".
// batchIDs[i] is the concrete message ID for local number i+1.
func parseAnalyzerResponse(raw string, batchIDs []string) ([]ActionPlanCategory, []int, error) {
	obj := extractJSONObject(raw)
	if obj == "" {
		return nil, nil, fmt.Errorf("no JSON object in analyzer response")
	}
	var parsed analyzerRawResponse
	if err := json.Unmarshal([]byte(obj), &parsed); err != nil {
		return nil, nil, fmt.Errorf("malformed analyzer JSON: %w", err)
	}

	resolve := func(nums []int) []string {
		ids := make([]string, 0, len(nums))
		for _, n := range nums {
			if n >= 1 && n <= len(batchIDs) {
				ids = append(ids, batchIDs[n-1])
			}
		}
		return ids
	}

	cats := make([]ActionPlanCategory, 0, len(parsed.Categories))
	for _, rc := range parsed.Categories {
		ids := resolve(rc.Messages)
		if len(ids) == 0 {
			continue // category with no resolvable messages is useless
		}
		cats = append(cats, ActionPlanCategory{
			Name:        strings.TrimSpace(rc.Name),
			Priority:    normalizePriority(rc.Priority),
			Description: strings.TrimSpace(rc.Description),
			Action:      normalizeAction(rc.Action),
			Label:       strings.TrimSpace(rc.Label),
			MessageIDs:  ids,
		})
	}
	return cats, parsed.ReadManually, nil
}

func normalizePriority(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "high", "medium", "low":
		return strings.ToLower(strings.TrimSpace(p))
	default:
		return "medium"
	}
}

func normalizeAction(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "archive", "mark_read", "trash", "label", "none":
		return strings.ToLower(strings.TrimSpace(a))
	default:
		return "none"
	}
}

// splitBatches divides messages into batches of at most size, capped at maxBatches.
// Messages beyond the cap are dropped (the caller surfaces this to the user).
func splitBatches(messages []AnalyzerMessage, size, maxBatches int) [][]AnalyzerMessage {
	if size <= 0 {
		size = 50
	}
	if maxBatches <= 0 {
		maxBatches = 10
	}
	var batches [][]AnalyzerMessage
	for i := 0; i < len(messages); i += size {
		if len(batches) >= maxBatches {
			break
		}
		end := i + size
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}
	return batches
}

// suppress unused import until Analyze is implemented in Task 6
var _ = context.Background
var _ = time.Now
