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

// mergeCategories merges incoming categories into existing, unioning message IDs of
// categories that share a name (case-insensitive) and appending new ones.
func mergeCategories(existing, incoming []ActionPlanCategory) []ActionPlanCategory {
	indexByName := make(map[string]int, len(existing))
	for i, c := range existing {
		indexByName[strings.ToLower(c.Name)] = i
	}
	for _, inc := range incoming {
		key := strings.ToLower(inc.Name)
		if idx, ok := indexByName[key]; ok {
			existing[idx].MessageIDs = unionIDs(existing[idx].MessageIDs, inc.MessageIDs)
			continue
		}
		indexByName[key] = len(existing)
		existing = append(existing, inc)
	}
	return existing
}

// unionIDs appends b's IDs to a, skipping IDs already present, preserving order.
func unionIDs(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			a = append(a, id)
			seen[id] = true
		}
	}
	return a
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
		subject := m.Subject
		if subject == "" {
			subject = "(no subject)"
		}
		fmt.Fprintf(&b, "%d. Subject: %s | From: %s | %s\n", i+1, subject, m.From, m.Snippet)
	}
	return b.String()
}

// buildBatchPrompt injects the batch payload into the chosen prompt template.
func buildBatchPrompt(promptText, payload string) string {
	if strings.Contains(promptText, "{{messages}}") {
		return strings.ReplaceAll(promptText, "{{messages}}", payload)
	}
	return promptText + "\n\n" + payload
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
	result, err := s.aiService.ApplyCustomPromptStream(ctx, prompt, nil, func(string) {})
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
	result2, err := s.aiService.ApplyCustomPromptStream(ctx, repair, nil, func(string) {})
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
