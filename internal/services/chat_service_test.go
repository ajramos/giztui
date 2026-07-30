package services

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestChatService_NilAIService verifies the guard when no AI service is wired.
func TestChatService_NilAIService(t *testing.T) {
	svc := NewChatService(nil, nil)
	_, err := svc.SendMessageStream(context.Background(), "s1", "email body", "hello", nil)
	assert.Error(t, err)
}

// TestChatService_EmptyMessage rejects a blank user message.
func TestChatService_EmptyMessage(t *testing.T) {
	mockAI := &mockAIService{}
	svc := NewChatService(mockAI, nil)
	_, err := svc.SendMessageStream(context.Background(), "s1", "email body", "   ", nil)
	assert.Error(t, err)
	mockAI.AssertNotCalled(t, "ApplyCustomPromptStream")
}

// TestChatService_SendGroundsAndRecordsHistory checks the prompt is grounded on
// the email + question, tokens stream through, and the exchange is recorded.
func TestChatService_SendGroundsAndRecordsHistory(t *testing.T) {
	mockAI := &mockAIService{}
	var capturedPrompts []string
	mockAI.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedPrompts = append(capturedPrompts, args.String(1))
			if cb, ok := args.Get(3).(func(string)); ok && cb != nil {
				cb("Hi ")
				cb("there")
			}
		}).
		Return("Hi there", nil)

	svc := NewChatService(mockAI, nil)

	var streamed strings.Builder
	reply, err := svc.SendMessageStream(context.Background(), "msg-1",
		"Meeting moved to Friday 3pm.", "When is the meeting?",
		func(tok string) { streamed.WriteString(tok) })

	assert.NoError(t, err)
	assert.Equal(t, "Hi there", reply)
	assert.Equal(t, "Hi there", streamed.String())

	// First prompt is grounded on the email body and the question.
	assert.Len(t, capturedPrompts, 1)
	assert.Contains(t, capturedPrompts[0], "Meeting moved to Friday 3pm.")
	assert.Contains(t, capturedPrompts[0], "When is the meeting?")

	// History now holds the user + assistant turns.
	hist := svc.GetHistory("msg-1")
	assert.Equal(t, []ChatTurn{
		{Role: "user", Text: "When is the meeting?"},
		{Role: "assistant", Text: "Hi there"},
	}, hist)

	// A second turn re-sends the prior exchange as transcript.
	_, err = svc.SendMessageStream(context.Background(), "msg-1",
		"Meeting moved to Friday 3pm.", "And where?", nil)
	assert.NoError(t, err)
	assert.Len(t, capturedPrompts, 2)
	assert.Contains(t, capturedPrompts[1], "When is the meeting?")
	assert.Contains(t, capturedPrompts[1], "And where?")
	assert.Len(t, svc.GetHistory("msg-1"), 4)
}

// TestChatService_ClearSession drops a session's history and isolates sessions.
func TestChatService_ClearSession(t *testing.T) {
	mockAI := &mockAIService{}
	mockAI.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("ok", nil)
	svc := NewChatService(mockAI, nil)

	_, _ = svc.SendMessageStream(context.Background(), "a", "body", "hi", nil)
	_, _ = svc.SendMessageStream(context.Background(), "b", "body", "hi", nil)
	assert.Len(t, svc.GetHistory("a"), 2)

	svc.ClearSession("a")
	assert.Empty(t, svc.GetHistory("a"))
	assert.Len(t, svc.GetHistory("b"), 2) // other session untouched
}

// TestChatService_ErrorNotRecorded ensures a failed turn does not poison history.
func TestChatService_ErrorNotRecorded(t *testing.T) {
	mockAI := &mockAIService{}
	mockAI.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", assert.AnError)
	svc := NewChatService(mockAI, nil)

	_, err := svc.SendMessageStream(context.Background(), "s", "body", "hi", nil)
	assert.Error(t, err)
	assert.Empty(t, svc.GetHistory("s"))
}
