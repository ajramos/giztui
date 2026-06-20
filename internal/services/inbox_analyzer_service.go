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

	// Reconcile read-manually: include the LLM's list AND any batch message it omitted from
	// every category and from read_manually. Weak models sometimes return only a subset of the
	// messages; without this, those would silently vanish from the plan. The invariant the prompt
	// states — every message appears in a category OR read_manually — is enforced here.
	readManually := make([]int, 0, len(batchIDs))
	seenRead := make(map[int]bool)
	addRead := func(n int) {
		if n < 1 || n > len(batchIDs) || seenRead[n] {
			return
		}
		if claimed[batchIDs[n-1]] {
			return // already placed in a category
		}
		claimed[batchIDs[n-1]] = true
		seenRead[n] = true
		readManually = append(readManually, n)
	}
	for _, n := range parsed.ReadManually {
		addRead(n)
	}
	for n := 1; n <= len(batchIDs); n++ {
		addRead(n)
	}
	return cats, readManually, nil
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
// by number. Numbering is local to the batch (1-based). When a message has a Body it is
// rendered (truncated to bodyCharLimit) on its own line; otherwise the Snippet is used inline.
func buildBatchPayload(batch []AnalyzerMessage, bodyCharLimit int) string {
	var b strings.Builder
	for i, m := range batch {
		subject := strings.ReplaceAll(m.Subject, "\n", " ")
		if strings.TrimSpace(subject) == "" {
			subject = "(no subject)"
		}
		if strings.TrimSpace(m.Body) != "" {
			fmt.Fprintf(&b, "%d. Subject: %s | From: %s\n   %s\n", i+1, subject, m.From, truncateForAnalyzer(m.Body, bodyCharLimit))
			continue
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

// BuildPromptPreview assembles the analyzer prompt the way Analyze does: the base prompt (role +
// rules + JSON format) leads, and the context block (existing labels + user interests) is injected
// at the {{context}} marker right before the emails. {{messages}} stays a literal placeholder.
func (s *InboxAnalyzerServiceImpl) BuildPromptPreview(opts InboxAnalyzerOptions) string {
	base := opts.CustomPromptText
	if strings.TrimSpace(base) == "" {
		base = defaultAnalyzerPrompt
	}
	return injectAnalyzerContext(base, analyzerContextBlock(opts.UserRules, opts.AvailableLabels))
}

// resolveExistingLabel returns the canonical existing label matching `suggested` (case- and
// surrounding-whitespace-insensitive), or ok=false when none matches. No fuzzy matching.
func resolveExistingLabel(suggested string, existing []string) (string, bool) {
	s := strings.TrimSpace(suggested)
	if s == "" {
		return "", false
	}
	for _, e := range existing {
		if strings.EqualFold(strings.TrimSpace(e), s) {
			return e, true
		}
	}
	return "", false
}

// enforceLabelPolicy resolves each "label" category against the existing user labels (in place).
// A match canonicalizes the category's Label. In strict mode (with a non-empty label set), a no-match
// category's messages are moved to ReadManually and the category is dropped — never creating a new
// label. With no available labels, enforcement is skipped so the analyzer degrades to prior behavior.
func enforceLabelPolicy(plan *ActionPlan, messages []AnalyzerMessage, availableLabels []string, strict bool) {
	if plan == nil {
		return
	}
	byID := make(map[string]AnalyzerMessage, len(messages))
	for _, m := range messages {
		byID[m.ID] = m
	}
	kept := plan.Categories[:0]
	for _, c := range plan.Categories {
		if c.Action != "label" {
			kept = append(kept, c)
			continue
		}
		if canonical, ok := resolveExistingLabel(c.Label, availableLabels); ok {
			c.Label = canonical
			kept = append(kept, c)
			continue
		}
		if strict && len(availableLabels) > 0 {
			for _, id := range c.MessageIDs {
				if m, found := byID[id]; found {
					plan.ReadManually = append(plan.ReadManually, m)
				}
			}
			continue // drop the invented-label category
		}
		kept = append(kept, c) // non-strict (or no label set): leave as-is
	}
	plan.Categories = kept
}

// availableLabelsBlock returns the "## Existing labels" context block telling the model to prefer
// an exact existing label for the "label" action. Empty labels → "".
func availableLabelsBlock(labels []string) string {
	clean := make([]string, 0, len(labels))
	for _, l := range labels {
		if strings.TrimSpace(l) != "" {
			clean = append(clean, strings.TrimSpace(l))
		}
	}
	if len(clean) == 0 {
		return ""
	}
	return "## Existing labels\n" +
		"These labels already exist in the mailbox: " + strings.Join(clean, ", ") + "\n" +
		"For the \"label\" action, Use ONLY a label from this exact list. Do NOT invent new labels. " +
		"If none of them fits the email, put the email in read_manually instead."
}

// userRulesBlock returns the "## User preferences and interests" context block so the LLM honors
// the user's free-text rules/interests. Empty rules → "".
func userRulesBlock(rules []string) string {
	clean := make([]string, 0, len(rules))
	for _, r := range rules {
		if strings.TrimSpace(r) != "" {
			clean = append(clean, "- "+strings.TrimSpace(r))
		}
	}
	if len(clean) == 0 {
		return ""
	}
	return "## User preferences and interests\n" +
		"Treat the following as BOTH action rules to respect AND interest/relevance signals. " +
		"When an email matches a stated interest, do NOT bury it in a bulk archive/trash group: " +
		"keep it visible, set that category's priority to \"high\", and note the matched interest " +
		"in the category description.\n" +
		strings.Join(clean, "\n")
}

// analyzerContextBlock joins the existing-labels and user-interests blocks (each omitted when
// empty) into a single context section. Returns "" when there is no context at all.
func analyzerContextBlock(rules, labels []string) string {
	parts := make([]string, 0, 2)
	if b := availableLabelsBlock(labels); b != "" {
		parts = append(parts, b)
	}
	if b := userRulesBlock(rules); b != "" {
		parts = append(parts, b)
	}
	return strings.Join(parts, "\n\n")
}

// injectAnalyzerContext places the context block at the {{context}} marker (the bundled template
// has one right before the emails, so the role/rules/format lead). For custom prompts without the
// marker it falls back to inserting the context before {{messages}}, or appending. Empty context
// removes the marker / leaves the prompt unchanged.
func injectAnalyzerContext(promptText, contextBlock string) string {
	if strings.Contains(promptText, "{{context}}") {
		if strings.TrimSpace(contextBlock) == "" {
			return strings.Replace(promptText, "{{context}}\n", "", 1)
		}
		return strings.Replace(promptText, "{{context}}", contextBlock+"\n", 1)
	}
	if strings.TrimSpace(contextBlock) == "" {
		return promptText
	}
	if strings.Contains(promptText, "{{messages}}") {
		return strings.Replace(promptText, "{{messages}}", contextBlock+"\n\n{{messages}}", 1)
	}
	return promptText + "\n\n" + contextBlock
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

	promptText = injectAnalyzerContext(promptText, analyzerContextBlock(opts.UserRules, opts.AvailableLabels))

	batches := splitBatches(messages, opts.BatchSize, opts.MaxBatches)
	plan := &ActionPlan{BatchesTotal: len(batches)}

	for bi, batch := range batches {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		payload := buildBatchPayload(batch, opts.BodyCharLimit)
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

	// Enforce the label policy on the FINAL plan only. The per-batch onProgress callback above may
	// have briefly shown invented-label categories on multi-batch inboxes; they are reconciled here
	// (canonicalized or moved to read-manually) before the plan is returned.
	enforceLabelPolicy(plan, messages, opts.AvailableLabels, opts.StrictLabels)
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
