package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/config"
)

const (
	// chatMaxBodyRunes caps the grounding email body (matches the summary path).
	chatMaxBodyRunes = 8000
	// chatMaxTurns bounds how many prior turns are re-sent in the transcript, since
	// the LLM API is single-shot and the whole conversation is resent every turn.
	chatMaxTurns = 12
)

// ChatServiceImpl implements ChatService on top of AIService. It keeps an
// in-memory per-session history and builds a single grounded prompt per turn
// (email + transcript + question), reusing AIService.ApplyCustomPromptStream —
// so it works on every provider with no LLM-layer changes (Bedrock streams as a
// single token via the non-streaming fallback, which callers already handle).
type ChatServiceImpl struct {
	aiService AIService
	config    *config.Config
	mu        sync.RWMutex
	sessions  map[string][]ChatTurn
}

// NewChatService creates a ChatService backed by the given AIService.
func NewChatService(aiService AIService, cfg *config.Config) *ChatServiceImpl {
	return &ChatServiceImpl{
		aiService: aiService,
		config:    cfg,
		sessions:  make(map[string][]ChatTurn),
	}
}

// SendMessageStream answers userMessage grounded on emailContent + prior turns.
func (s *ChatServiceImpl) SendMessageStream(ctx context.Context, sessionID, emailContent, userMessage string, onToken func(string)) (string, error) {
	if s.aiService == nil {
		return "", fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", fmt.Errorf("message cannot be empty")
	}

	history := s.GetHistory(sessionID)
	prompt := s.buildChatPrompt(emailContent, history, userMessage)

	raw, err := s.aiService.ApplyCustomPromptStream(ctx, prompt, nil, func(token string) {
		if onToken != nil {
			onToken(token)
		}
	})
	if err != nil {
		return "", fmt.Errorf("chat generation failed: %w", err)
	}

	reply := strings.TrimSpace(raw)
	// Persist the exchange only after a successful reply so a failed turn doesn't
	// poison the history.
	s.appendTurns(sessionID,
		ChatTurn{Role: "user", Text: userMessage},
		ChatTurn{Role: "assistant", Text: reply},
	)
	return reply, nil
}

// GetHistory returns a copy of the session's turns.
func (s *ChatServiceImpl) GetHistory(sessionID string) []ChatTurn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := s.sessions[sessionID]
	out := make([]ChatTurn, len(h))
	copy(out, h)
	return out
}

// ClearSession drops a session's history.
func (s *ChatServiceImpl) ClearSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *ChatServiceImpl) appendTurns(sessionID string, turns ...ChatTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = append(s.sessions[sessionID], turns...)
}

// buildChatPrompt renders the chat template with the (bounded) email body, the
// recent transcript, and the new question.
func (s *ChatServiceImpl) buildChatPrompt(emailContent string, history []ChatTurn, userMessage string) string {
	body := emailContent
	if r := []rune(body); len(r) > chatMaxBodyRunes {
		body = string(r[:chatMaxBodyRunes]) + "\n[...email truncated...]"
	}

	if len(history) > chatMaxTurns {
		history = history[len(history)-chatMaxTurns:]
	}
	var tb strings.Builder
	for _, t := range history {
		role := "User"
		if t.Role == "assistant" {
			role = "Assistant"
		}
		fmt.Fprintf(&tb, "%s: %s\n", role, t.Text)
	}

	tmpl := ""
	if s.config != nil {
		tmpl = s.config.LLM.GetChatPrompt()
	}
	if strings.TrimSpace(tmpl) == "" {
		tmpl = "You are a helpful assistant answering questions about the email below.\n\n" +
			"--- EMAIL ---\n{{body}}\n--- END EMAIL ---\n\n{{transcript}}User: {{question}}\nAssistant:"
	}
	prompt := strings.ReplaceAll(tmpl, "{{body}}", body)
	prompt = strings.ReplaceAll(prompt, "{{transcript}}", tb.String())
	prompt = strings.ReplaceAll(prompt, "{{question}}", userMessage)
	return prompt
}
