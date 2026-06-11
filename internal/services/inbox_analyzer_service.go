package services

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
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
// fences and surrounding prose. Braces inside JSON string literals are ignored. Returns
// "" if no object is found.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
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

	claimed := make(map[string]bool)
	resolve := func(nums []int) []string {
		ids := make([]string, 0, len(nums))
		for _, n := range nums {
			if n < 1 || n > len(batchIDs) {
				continue
			}
			id := batchIDs[n-1]
			if claimed[id] {
				continue // already assigned to an earlier category
			}
			claimed[id] = true
			ids = append(ids, id)
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
	s := strings.ToLower(strings.TrimSpace(p))
	switch s {
	case "high", "medium", "low":
		return s
	default:
		return "medium"
	}
}

func normalizeAction(a string) string {
	s := strings.ToLower(strings.TrimSpace(a))
	switch s {
	case "archive", "mark_read", "trash", "label", "none":
		return s
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
		// Sub-slices share the messages backing array; callers must not mutate messages
		// after this call.
		batches = append(batches, messages[i:end])
	}
	return batches
}

// mergeCategories merges incoming categories into existing, unioning message IDs of
// categories that share a name (case-insensitive) and appending new ones.
// It does not mutate the caller's existing slice or its elements.
func mergeCategories(existing, incoming []ActionPlanCategory) []ActionPlanCategory {
	// Work on a shallow copy so the caller's slice elements are not mutated.
	out := make([]ActionPlanCategory, len(existing))
	copy(out, existing)
	indexByName := make(map[string]int, len(out))
	for i, c := range out {
		indexByName[strings.ToLower(c.Name)] = i
	}
	for _, inc := range incoming {
		key := strings.ToLower(inc.Name)
		if idx, ok := indexByName[key]; ok {
			out[idx].MessageIDs = unionIDs(out[idx].MessageIDs, inc.MessageIDs)
			continue
		}
		indexByName[key] = len(out)
		out = append(out, inc)
	}
	return out
}

// unionIDs returns a new slice containing a's IDs followed by b's IDs not already
// present, preserving order. It does not mutate either input's backing array.
func unionIDs(a, b []string) []string {
	out := make([]string, len(a))
	copy(out, a)
	seen := make(map[string]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			out = append(out, id)
			seen[id] = true
		}
	}
	return out
}

// InboxAnalyzerServiceImpl implements InboxAnalyzerService using the AIService directly.
type InboxAnalyzerServiceImpl struct {
	aiService AIService
}

// NewInboxAnalyzerService creates an inbox analyzer backed by the given AIService.
func NewInboxAnalyzerService(aiService AIService) *InboxAnalyzerServiceImpl {
	return &InboxAnalyzerServiceImpl{aiService: aiService}
}

// buildBatchPayload renders one batch as a compact, numbered list the LLM can reference
// by number. Numbering is local to the batch (1-based).
func buildBatchPayload(batch []AnalyzerMessage) string {
	var b strings.Builder
	for i, m := range batch {
		subject := strings.ReplaceAll(m.Subject, "\n", " ")
		if strings.TrimSpace(subject) == "" {
			subject = "(no subject)"
		}
		snippet := strings.ReplaceAll(m.Snippet, "\n", " ")
		fmt.Fprintf(&b, "%d. Subject: %s | From: %s | %s\n", i+1, subject, m.From, snippet)
	}
	return b.String()
}

// truncateForAnalyzer collapses runs of whitespace (incl. newlines) to single spaces, trims
// the ends, and cuts to at most limit runes on a rune boundary. limit <= 0 skips the cut.
func truncateForAnalyzer(text string, limit int) string {
	collapsed := strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if limit <= 0 {
		return collapsed
	}
	r := []rune(collapsed)
	if len(r) <= limit {
		return collapsed
	}
	return string(r[:limit])
}

// buildBatchPrompt injects the batch payload into the chosen prompt template.
func buildBatchPrompt(promptText, payload string) string {
	if strings.Contains(promptText, "{{messages}}") {
		return strings.ReplaceAll(promptText, "{{messages}}", payload)
	}
	return promptText + "\n\n" + payload
}

// prependUserRules adds a "## User preferences" block before the analyzer prompt
// so the LLM honors the user's free-text rules. Empty rules → prompt unchanged.
func prependUserRules(promptText string, rules []string) string {
	clean := make([]string, 0, len(rules))
	for _, r := range rules {
		if strings.TrimSpace(r) != "" {
			clean = append(clean, "- "+strings.TrimSpace(r))
		}
	}
	if len(clean) == 0 {
		return promptText
	}
	return "## User preferences (respect these rules)\n" + strings.Join(clean, "\n") + "\n\n" + promptText
}

func (s *InboxAnalyzerServiceImpl) Analyze(ctx context.Context, messages []AnalyzerMessage, opts InboxAnalyzerOptions, onProgress func(*ActionPlan)) (*ActionPlan, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	promptText := opts.CustomPromptText
	if strings.TrimSpace(promptText) == "" {
		promptText = defaultAnalyzerPrompt
	}

	promptText = prependUserRules(promptText, opts.UserRules)

	batches := splitBatches(messages, opts.BatchSize, opts.MaxBatches)
	plan := &ActionPlan{BatchesTotal: len(batches)}

	for bi, batch := range batches {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		payload := buildBatchPayload(batch)
		batchIDs := make([]string, len(batch))
		for i, m := range batch {
			batchIDs[i] = m.ID
		}

		cats, readIdx, err := s.runBatch(ctx, promptText, payload, batchIDs)
		if err != nil {
			if bi == 0 {
				// First batch hard failure → abort so the panel never opens.
				return nil, err
			}
			// Intermediate batch failure → keep what we have, mark interrupted.
			return plan, err
		}
		if cats == nil {
			// Degraded batch: surface every message in this batch as read-manually
			// so nothing is silently lost, and flag the plan.
			plan.Degraded = true
			plan.ReadManually = append(plan.ReadManually, batch...)
		} else {
			plan.Categories = mergeCategories(plan.Categories, cats)
			plan.ReadManually = append(plan.ReadManually, resolveMessages(readIdx, batch)...)
		}

		plan.TotalAnalyzed += len(batch)
		plan.BatchesDone = bi + 1
		if onProgress != nil {
			onProgress(plan)
		}
	}

	return plan, nil
}

// runBatch streams one batch through the LLM and parses it, with a single repair retry
// on malformed JSON. Returns (nil, nil, nil) when both attempts are unparseable (caller
// treats this as a degraded batch).
func (s *InboxAnalyzerServiceImpl) runBatch(ctx context.Context, promptText, payload string, batchIDs []string) ([]ActionPlanCategory, []int, error) {
	prompt := buildBatchPrompt(promptText, payload)
	result, err := s.aiService.ApplyCustomPromptStream(ctx, prompt, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	cats, readIdx, perr := parseAnalyzerResponse(result, batchIDs)
	if perr == nil {
		return cats, readIdx, nil
	}

	// Repair retry: re-ask with a strict instruction.
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	repair := prompt + "\n\nIMPORTANT: Your previous answer was not valid JSON. Reply with ONLY the JSON object described above, no prose, no markdown."
	result2, err := s.aiService.ApplyCustomPromptStream(ctx, repair, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	cats, readIdx, perr = parseAnalyzerResponse(result2, batchIDs)
	if perr != nil {
		return nil, nil, nil // degrade
	}
	return cats, readIdx, nil
}

// resolveMessages maps per-batch local indices (1-based) to AnalyzerMessage values.
func resolveMessages(indices []int, batch []AnalyzerMessage) []AnalyzerMessage {
	out := make([]AnalyzerMessage, 0, len(indices))
	for _, n := range indices {
		if n >= 1 && n <= len(batch) {
			out = append(out, batch[n-1])
		}
	}
	return out
}
