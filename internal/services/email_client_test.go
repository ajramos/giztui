package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// Completes MockGmailServiceClient (defined in email_service_test.go) so it satisfies GmailClient.
func (m *MockGmailServiceClient) GetMessagesParallel(messageIDs []string, maxWorkers int) ([]*gmail_v1.Message, error) {
	args := m.Called(messageIDs, maxWorkers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gmail_v1.Message), args.Error(1)
}

func TestEmailService_BulkTrash_Success(t *testing.T) {
	client := &MockGmailServiceClient{}
	svc := NewEmailService(&MockEmailRepository{}, client, nil)
	ids := []string{"a", "b", "c"}
	for _, id := range ids {
		client.On("TrashMessage", id).Return(nil)
	}
	var progress [][2]int
	err := svc.BulkTrash(context.Background(), ids, func(done, total int) {
		progress = append(progress, [2]int{done, total})
	})
	assert.NoError(t, err)
	client.AssertExpectations(t)
	// Progress is reported once per message as (done, total).
	assert.Equal(t, [][2]int{{1, 3}, {2, 3}, {3, 3}}, progress)
}

func TestEmailService_BulkTrash_PartialFailure(t *testing.T) {
	client := &MockGmailServiceClient{}
	svc := NewEmailService(&MockEmailRepository{}, client, nil)
	client.On("TrashMessage", "a").Return(nil)
	client.On("TrashMessage", "b").Return(errors.New("boom"))
	err := svc.BulkTrash(context.Background(), []string{"a", "b"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to trash b")
}

func TestEmailService_SendMessage(t *testing.T) {
	client := &MockGmailServiceClient{}
	svc := NewEmailService(&MockEmailRepository{}, client, nil)
	ctx := context.Background()
	// Validation: empty subject/body/to.
	assert.Error(t, svc.SendMessage(ctx, "from", "", "subj", "body", nil, nil))
	// Success path delegates to the client.
	client.On("SendMessage", "from", "to@x.com", "subj", "body", []string(nil), []string(nil)).Return("msgid", nil)
	assert.NoError(t, svc.SendMessage(ctx, "from", "to@x.com", "subj", "body", nil, nil))
	client.AssertExpectations(t)
}

func TestEmailService_ReplyToMessage(t *testing.T) {
	client := &MockGmailServiceClient{}
	svc := NewEmailService(&MockEmailRepository{}, client, nil)
	ctx := context.Background()
	assert.Error(t, svc.ReplyToMessage(ctx, "", "body", true, nil))
	client.On("ReplyMessage", "orig", "reply body", true, []string(nil)).Return("id", nil)
	assert.NoError(t, svc.ReplyToMessage(ctx, "orig", "reply body", true, nil))
	client.AssertExpectations(t)
}
