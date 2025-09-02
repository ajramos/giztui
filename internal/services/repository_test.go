package services

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/stretchr/testify/assert"
)

func TestNewMessageRepository(t *testing.T) {
	// Test with nil client
	repo := NewMessageRepository(nil)
	assert.NotNil(t, repo)
	assert.Nil(t, repo.gmailClient)

	// Test with valid client
	client := &gmail.Client{}
	repo = NewMessageRepository(client)
	assert.NotNil(t, repo)
	assert.Equal(t, client, repo.gmailClient)
}

func TestMessageRepositoryImpl_GetMessage_ValidationErrors(t *testing.T) {
	repo := NewMessageRepository(&gmail.Client{})
	ctx := context.Background()

	// Test empty message ID
	msg, err := repo.GetMessage(ctx, "")
	assert.Nil(t, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message ID cannot be empty")
}

func TestMessageRepositoryImpl_SearchMessages_ValidationErrors(t *testing.T) {
	repo := NewMessageRepository(&gmail.Client{})
	ctx := context.Background()
	opts := QueryOptions{MaxResults: 10}

	// Test empty search query
	result, err := repo.SearchMessages(ctx, "", opts)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search query cannot be empty")
}

func TestMessageRepositoryImpl_UpdateMessage_ValidationErrors(t *testing.T) {
	repo := NewMessageRepository(&gmail.Client{})
	ctx := context.Background()
	updates := MessageUpdates{}

	// Test empty message ID
	err := repo.UpdateMessage(ctx, "", updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message ID cannot be empty")
}

// Test the QueryOptions struct functionality
func TestQueryOptions_DefaultValues(t *testing.T) {
	opts := QueryOptions{}
	assert.Equal(t, int64(0), opts.MaxResults)
	assert.Equal(t, "", opts.PageToken)

	opts = QueryOptions{MaxResults: 50, PageToken: "test-token"}
	assert.Equal(t, int64(50), opts.MaxResults)
	assert.Equal(t, "test-token", opts.PageToken)
}

// Test the MessageUpdates struct functionality
func TestMessageUpdates_Functionality(t *testing.T) {
	updates := MessageUpdates{}
	assert.Nil(t, updates.AddLabels)
	assert.Nil(t, updates.RemoveLabels)
	assert.Nil(t, updates.MarkAsRead)

	// Test with values
	markAsRead := true
	updates = MessageUpdates{
		AddLabels:    []string{"label1", "label2"},
		RemoveLabels: []string{"label3"},
		MarkAsRead:   &markAsRead,
	}
	assert.Equal(t, []string{"label1", "label2"}, updates.AddLabels)
	assert.Equal(t, []string{"label3"}, updates.RemoveLabels)
	assert.NotNil(t, updates.MarkAsRead)
	assert.True(t, *updates.MarkAsRead)
}

// Test MessagePage struct functionality
func TestMessagePage_Functionality(t *testing.T) {
	page := &MessagePage{
		Messages:      nil,
		NextPageToken: "next-token",
		TotalCount:    0,
	}
	assert.Nil(t, page.Messages)
	assert.Equal(t, "next-token", page.NextPageToken)
	assert.Equal(t, 0, page.TotalCount)
}

// Benchmark tests for performance critical operations
func BenchmarkMessageRepository_ValidationOnly(b *testing.B) {
	repo := NewMessageRepository(&gmail.Client{})
	ctx := context.Background()

	b.Run("GetMessage_EmptyID", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = repo.GetMessage(ctx, "")
		}
	})

	b.Run("SearchMessages_EmptyQuery", func(b *testing.B) {
		opts := QueryOptions{MaxResults: 10}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = repo.SearchMessages(ctx, "", opts)
		}
	})

	b.Run("UpdateMessage_EmptyID", func(b *testing.B) {
		updates := MessageUpdates{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = repo.UpdateMessage(ctx, "", updates)
		}
	})
}

// Test repository error handling patterns
func TestMessageRepository_ErrorHandling(t *testing.T) {
	repo := NewMessageRepository(&gmail.Client{})
	ctx := context.Background()

	tests := []struct {
		name           string
		testFunc       func() error
		expectedErrors []string
	}{
		{
			name: "GetMessage_empty_id",
			testFunc: func() error {
				_, err := repo.GetMessage(ctx, "")
				return err
			},
			expectedErrors: []string{"message ID cannot be empty"},
		},
		{
			name: "SearchMessages_empty_query",
			testFunc: func() error {
				opts := QueryOptions{MaxResults: 10}
				_, err := repo.SearchMessages(ctx, "", opts)
				return err
			},
			expectedErrors: []string{"search query cannot be empty"},
		},
		{
			name: "UpdateMessage_empty_id",
			testFunc: func() error {
				updates := MessageUpdates{}
				return repo.UpdateMessage(ctx, "", updates)
			},
			expectedErrors: []string{"message ID cannot be empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			assert.Error(t, err)

			// Check that error contains at least one of the expected error messages
			found := false
			for _, expectedErr := range tt.expectedErrors {
				if assert.Contains(t, err.Error(), expectedErr) {
					found = true
					break
				}
			}
			assert.True(t, found, "Error should contain one of: %v, got: %s", tt.expectedErrors, err.Error())
		})
	}
}

// Test repository with nil client
func TestMessageRepository_NilClient(t *testing.T) {
	repo := NewMessageRepository(nil)
	ctx := context.Background()

	// These operations should still validate inputs even with nil client
	_, err := repo.GetMessage(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message ID cannot be empty")

	opts := QueryOptions{MaxResults: 10}
	_, err = repo.SearchMessages(ctx, "", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search query cannot be empty")

	updates := MessageUpdates{}
	err = repo.UpdateMessage(ctx, "", updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message ID cannot be empty")
}

// Test additional repository scenarios
func TestMessageRepository_AdditionalScenarios(t *testing.T) {
	// Test that repository can be created successfully
	repo := NewMessageRepository(&gmail.Client{})
	assert.NotNil(t, repo)

	// Test context handling
	ctx := context.Background()
	assert.NotNil(t, ctx)

	// Test QueryOptions with different values
	opts1 := QueryOptions{MaxResults: 100}
	opts2 := QueryOptions{MaxResults: 50, PageToken: "token123"}
	assert.Equal(t, int64(100), opts1.MaxResults)
	assert.Equal(t, "token123", opts2.PageToken)
}
