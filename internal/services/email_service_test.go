package services

import (
	"context"
	"errors"
	"testing"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/ajramos/giztui/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// MockEmailRepository implements MessageRepository for testing
type MockEmailRepository struct {
	mock.Mock
}

func (m *MockEmailRepository) GetMessages(ctx context.Context, opts QueryOptions) (*MessagePage, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*MessagePage), args.Error(1)
}

func (m *MockEmailRepository) GetMessage(ctx context.Context, id string) (*gmail.Message, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.Message), args.Error(1)
}

func (m *MockEmailRepository) SearchMessages(ctx context.Context, query string, opts QueryOptions) (*MessagePage, error) {
	args := m.Called(ctx, query, opts)
	return args.Get(0).(*MessagePage), args.Error(1)
}

func (m *MockEmailRepository) UpdateMessage(ctx context.Context, id string, updates MessageUpdates) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockEmailRepository) GetDrafts(ctx context.Context, maxResults int64) ([]*gmail_v1.Draft, error) {
	args := m.Called(ctx, maxResults)
	return args.Get(0).([]*gmail_v1.Draft), args.Error(1)
}

func (m *MockEmailRepository) GetDraft(ctx context.Context, draftID string) (*gmail_v1.Draft, error) {
	args := m.Called(ctx, draftID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail_v1.Draft), args.Error(1)
}

// MockGmailServiceClient implements the Gmail client methods used by EmailService
type MockGmailServiceClient struct {
	mock.Mock
}

func (m *MockGmailServiceClient) TrashMessage(messageID string) error {
	args := m.Called(messageID)
	return args.Error(0)
}

func (m *MockGmailServiceClient) SendMessage(from, to, subject, body string, cc, bcc []string) (string, error) {
	args := m.Called(from, to, subject, body, cc, bcc)
	return args.String(0), args.Error(1)
}

func (m *MockGmailServiceClient) ReplyMessage(originalID, replyBody string, send bool, cc []string) (string, error) {
	args := m.Called(originalID, replyBody, send, cc)
	return args.String(0), args.Error(1)
}

func (m *MockGmailServiceClient) GetMessageWithContent(id string) (*gmail.Message, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.Message), args.Error(1)
}

func TestNewEmailService(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}

	service := NewEmailService(repo, client, renderer)

	assert.NotNil(t, service)
	assert.Equal(t, repo, service.repo)
	assert.Equal(t, client, service.gmailClient)
	assert.Equal(t, renderer, service.renderer)
}

func TestEmailService_MarkAsRead_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test empty message ID
	err := service.MarkAsRead(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messageID cannot be empty")
}

func TestEmailService_MarkAsRead_Success(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Mock successful update
	expectedUpdates := MessageUpdates{RemoveLabels: []string{"UNREAD"}}
	repo.On("UpdateMessage", ctx, "msg123", expectedUpdates).Return(nil)

	err := service.MarkAsRead(ctx, "msg123")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEmailService_MarkAsUnread_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test empty message ID
	err := service.MarkAsUnread(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messageID cannot be empty")
}

func TestEmailService_MarkAsUnread_Success(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Mock successful update
	expectedUpdates := MessageUpdates{AddLabels: []string{"UNREAD"}}
	repo.On("UpdateMessage", ctx, "msg123", expectedUpdates).Return(nil)

	err := service.MarkAsUnread(ctx, "msg123")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEmailService_ArchiveMessage_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test empty message ID
	err := service.ArchiveMessage(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messageID cannot be empty")
}

func TestEmailService_ArchiveMessage_Success(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Mock successful update
	expectedUpdates := MessageUpdates{RemoveLabels: []string{"INBOX"}}
	repo.On("UpdateMessage", ctx, "msg123", expectedUpdates).Return(nil)

	err := service.ArchiveMessage(ctx, "msg123")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEmailService_TrashMessage_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test empty message ID
	err := service.TrashMessage(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messageID cannot be empty")
}

func TestEmailService_SendMessage_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	tests := []struct {
		name     string
		to       string
		subject  string
		body     string
		expected string
	}{
		{"empty_to", "", "Subject", "Body", "to, subject, and body cannot be empty"},
		{"empty_subject", "test@example.com", "", "Body", "to, subject, and body cannot be empty"},
		{"empty_body", "test@example.com", "Subject", "", "to, subject, and body cannot be empty"},
		{"all_empty", "", "", "", "to, subject, and body cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SendMessage(ctx, "from@example.com", tt.to, tt.subject, tt.body, nil, nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestEmailService_ReplyToMessage_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	tests := []struct {
		name       string
		originalID string
		replyBody  string
		expected   string
	}{
		{"empty_original_id", "", "Reply body", "originalID and replyBody cannot be empty"},
		{"empty_reply_body", "msg123", "", "originalID and replyBody cannot be empty"},
		{"both_empty", "", "", "originalID and replyBody cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ReplyToMessage(ctx, tt.originalID, tt.replyBody, true, nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestEmailService_BulkArchive_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test empty message IDs list
	err := service.BulkArchive(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no message IDs provided")

	// Test nil message IDs list
	err = service.BulkArchive(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no message IDs provided")
}

func TestEmailService_BulkArchive_Success(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	messageIDs := []string{"msg1", "msg2", "msg3"}
	expectedUpdates := MessageUpdates{RemoveLabels: []string{"INBOX"}}

	// Mock successful updates for all messages
	for _, id := range messageIDs {
		repo.On("UpdateMessage", ctx, id, expectedUpdates).Return(nil)
	}

	err := service.BulkArchive(ctx, messageIDs)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEmailService_BulkArchive_PartialFailure(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	messageIDs := []string{"msg1", "msg2", "msg3"}
	expectedUpdates := MessageUpdates{RemoveLabels: []string{"INBOX"}}

	// Mock mixed results
	repo.On("UpdateMessage", ctx, "msg1", expectedUpdates).Return(nil)
	repo.On("UpdateMessage", ctx, "msg2", expectedUpdates).Return(errors.New("API error"))
	repo.On("UpdateMessage", ctx, "msg3", expectedUpdates).Return(nil)

	err := service.BulkArchive(ctx, messageIDs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bulk archive errors")
	assert.Contains(t, err.Error(), "failed to archive msg2")
	repo.AssertExpectations(t)
}

func TestEmailService_BulkTrash_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test empty message IDs list
	err := service.BulkTrash(ctx, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no message IDs provided")

	// Test nil message IDs list
	err = service.BulkTrash(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no message IDs provided")
}

func TestEmailService_SaveMessageToFile_ValidationErrors(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	tests := []struct {
		name      string
		messageID string
		filePath  string
		expected  string
	}{
		{"empty_message_id", "", "/path/to/file.txt", "messageID and filePath cannot be empty"},
		{"empty_file_path", "msg123", "", "messageID and filePath cannot be empty"},
		{"both_empty", "", "", "messageID and filePath cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SaveMessageToFile(ctx, tt.messageID, tt.filePath)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

// Test error handling patterns
func TestEmailService_ErrorHandling(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test repository error propagation
	repo.On("UpdateMessage", ctx, "msg123", mock.AnythingOfType("MessageUpdates")).Return(errors.New("repository error"))

	err := service.MarkAsRead(ctx, "msg123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository error")
	repo.AssertExpectations(t)
}

// Benchmark tests for performance critical operations
func BenchmarkEmailService_ValidationOnly(b *testing.B) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	b.Run("MarkAsRead_EmptyID", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.MarkAsRead(ctx, "")
		}
	})

	b.Run("SendMessage_EmptyFields", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.SendMessage(ctx, "from@example.com", "", "", "", nil, nil)
		}
	})

	b.Run("BulkArchive_EmptyList", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.BulkArchive(ctx, []string{})
		}
	})
}

// Test business logic consistency
func TestEmailService_BusinessLogicConsistency(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test that MarkAsRead removes UNREAD label
	expectedReadUpdates := MessageUpdates{RemoveLabels: []string{"UNREAD"}}
	repo.On("UpdateMessage", ctx, "msg123", expectedReadUpdates).Return(nil)

	err := service.MarkAsRead(ctx, "msg123")
	assert.NoError(t, err)

	// Test that MarkAsUnread adds UNREAD label
	expectedUnreadUpdates := MessageUpdates{AddLabels: []string{"UNREAD"}}
	repo.On("UpdateMessage", ctx, "msg456", expectedUnreadUpdates).Return(nil)

	err = service.MarkAsUnread(ctx, "msg456")
	assert.NoError(t, err)

	// Test that ArchiveMessage removes INBOX label
	expectedArchiveUpdates := MessageUpdates{RemoveLabels: []string{"INBOX"}}
	repo.On("UpdateMessage", ctx, "msg789", expectedArchiveUpdates).Return(nil)

	err = service.ArchiveMessage(ctx, "msg789")
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

// Test edge cases for bulk operations
func TestEmailService_BulkOperations_EdgeCases(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)
	ctx := context.Background()

	// Test single message bulk operation
	messageIDs := []string{"msg1"}
	expectedUpdates := MessageUpdates{RemoveLabels: []string{"INBOX"}}
	repo.On("UpdateMessage", ctx, "msg1", expectedUpdates).Return(nil)

	err := service.BulkArchive(ctx, messageIDs)
	assert.NoError(t, err)
	repo.AssertExpectations(t)

	// Test bulk operation with duplicate message IDs
	duplicateIDs := []string{"msg1", "msg1", "msg2"}
	for _, id := range duplicateIDs {
		repo.On("UpdateMessage", ctx, id, expectedUpdates).Return(nil)
	}

	err = service.BulkArchive(ctx, duplicateIDs)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// Test file operations validation logic only
func TestEmailService_FileOperations_Validation(t *testing.T) {
	repo := &MockEmailRepository{}
	client := &gmail.Client{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, client, renderer)

	// Only test validation logic, don't try to make API calls
	// These should pass validation but we won't call the method to avoid API calls

	// Test validation directly - this is already covered by SaveMessageToFile_ValidationErrors
	// So this test just confirms the service can be created properly
	assert.NotNil(t, service)
	assert.NotNil(t, service.gmailClient)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.renderer)
}

// Test service initialization with nil dependencies
func TestEmailService_NilDependencies(t *testing.T) {
	// Test with nil repository
	service := NewEmailService(nil, &gmail.Client{}, &render.EmailRenderer{})
	assert.NotNil(t, service)
	assert.Nil(t, service.repo)

	// Test with nil client
	service = NewEmailService(&MockEmailRepository{}, nil, &render.EmailRenderer{})
	assert.NotNil(t, service)
	assert.Nil(t, service.gmailClient)

	// Test with nil renderer
	service = NewEmailService(&MockEmailRepository{}, &gmail.Client{}, nil)
	assert.NotNil(t, service)
	assert.Nil(t, service.renderer)
}

// Test comprehensive validation scenarios
func TestEmailService_ComprehensiveValidation(t *testing.T) {
	repo := &MockEmailRepository{}
	renderer := &render.EmailRenderer{}
	service := NewEmailService(repo, &gmail.Client{}, renderer)
	ctx := context.Background()

	// Test that all validation methods work correctly without making API calls
	tests := []struct {
		name        string
		testFunc    func() error
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "MarkAsRead_empty_id",
			testFunc:    func() error { return service.MarkAsRead(ctx, "") },
			shouldError: true,
			errorMsg:    "messageID cannot be empty",
		},
		{
			name:        "ArchiveMessage_empty_id",
			testFunc:    func() error { return service.ArchiveMessage(ctx, "") },
			shouldError: true,
			errorMsg:    "messageID cannot be empty",
		},
		{
			name:        "TrashMessage_empty_id",
			testFunc:    func() error { return service.TrashMessage(ctx, "") },
			shouldError: true,
			errorMsg:    "messageID cannot be empty",
		},
		{
			name:        "SendMessage_empty_fields",
			testFunc:    func() error { return service.SendMessage(ctx, "from@test.com", "", "", "", nil, nil) },
			shouldError: true,
			errorMsg:    "to, subject, and body cannot be empty",
		},
		{
			name:        "SaveMessageToFile_empty_fields",
			testFunc:    func() error { return service.SaveMessageToFile(ctx, "", "") },
			shouldError: true,
			errorMsg:    "messageID and filePath cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
