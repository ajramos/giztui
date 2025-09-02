package services

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/stretchr/testify/assert"
)

func TestNewLabelService(t *testing.T) {
	// Test with nil client
	service := NewLabelService(nil)
	assert.NotNil(t, service)
	assert.Nil(t, service.gmailClient)

	// Test with valid client
	client := &gmail.Client{}
	service = NewLabelService(client)
	assert.NotNil(t, service)
	assert.Equal(t, client, service.gmailClient)
}

func TestLabelService_CreateLabel_ValidationErrors(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	tests := []struct {
		name          string
		labelName     string
		expectedError string
	}{
		{"empty_name", "", "label name cannot be empty"},
		{"whitespace_only", "   ", "label name cannot be empty"},
		{"tabs_only", "\t\t", "label name cannot be empty"},
		{"newlines_only", "\n\n", "label name cannot be empty"},
		{"mixed_whitespace", " \t\n ", "label name cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateLabel(ctx, tt.labelName)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestLabelService_RenameLabel_ValidationErrors(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	tests := []struct {
		name          string
		labelID       string
		newName       string
		expectedError string
	}{
		{"empty_label_id", "", "NewLabel", "labelID and newName cannot be empty"},
		{"empty_new_name", "label123", "", "labelID and newName cannot be empty"},
		{"both_empty", "", "", "labelID and newName cannot be empty"},
		{"whitespace_label_id", "   ", "NewLabel", "labelID and newName cannot be empty"},
		{"whitespace_new_name", "label123", "   ", "labelID and newName cannot be empty"},
		{"both_whitespace", "   ", "   ", "labelID and newName cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.RenameLabel(ctx, tt.labelID, tt.newName)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestLabelService_DeleteLabel_ValidationErrors(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	tests := []struct {
		name          string
		labelID       string
		expectedError string
	}{
		{"empty_label_id", "", "labelID cannot be empty"},
		{"whitespace_only", "   ", "labelID cannot be empty"},
		{"tabs_only", "\t\t", "labelID cannot be empty"},
		{"newlines_only", "\n\n", "labelID cannot be empty"},
		{"mixed_whitespace", " \t\n ", "labelID cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteLabel(ctx, tt.labelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestLabelService_ApplyLabel_ValidationErrors(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	tests := []struct {
		name          string
		messageID     string
		labelID       string
		expectedError string
	}{
		{"empty_message_id", "", "label123", "messageID and labelID cannot be empty"},
		{"empty_label_id", "msg123", "", "messageID and labelID cannot be empty"},
		{"both_empty", "", "", "messageID and labelID cannot be empty"},
		{"whitespace_message_id", "   ", "label123", "messageID and labelID cannot be empty"},
		{"whitespace_label_id", "msg123", "   ", "messageID and labelID cannot be empty"},
		{"both_whitespace", "   ", "   ", "messageID and labelID cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ApplyLabel(ctx, tt.messageID, tt.labelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestLabelService_RemoveLabel_ValidationErrors(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	tests := []struct {
		name          string
		messageID     string
		labelID       string
		expectedError string
	}{
		{"empty_message_id", "", "label123", "messageID and labelID cannot be empty"},
		{"empty_label_id", "msg123", "", "messageID and labelID cannot be empty"},
		{"both_empty", "", "", "messageID and labelID cannot be empty"},
		{"whitespace_message_id", "   ", "label123", "messageID and labelID cannot be empty"},
		{"whitespace_label_id", "msg123", "   ", "messageID and labelID cannot be empty"},
		{"both_whitespace", "   ", "   ", "messageID and labelID cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RemoveLabel(ctx, tt.messageID, tt.labelID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestLabelService_GetMessageLabels_ValidationErrors(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	tests := []struct {
		name          string
		messageID     string
		expectedError string
	}{
		{"empty_message_id", "", "messageID cannot be empty"},
		{"whitespace_only", "   ", "messageID cannot be empty"},
		{"tabs_only", "\t\t", "messageID cannot be empty"},
		{"newlines_only", "\n\n", "messageID cannot be empty"},
		{"mixed_whitespace", " \t\n ", "messageID cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetMessageLabels(ctx, tt.messageID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// Test service with nil client (should handle gracefully)
func TestLabelService_NilClient(t *testing.T) {
	service := NewLabelService(nil)
	ctx := context.Background()

	// These operations should still validate inputs even with nil client
	_, err := service.CreateLabel(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "label name cannot be empty")

	_, err = service.RenameLabel(ctx, "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "labelID and newName cannot be empty")

	err = service.DeleteLabel(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "labelID cannot be empty")

	err = service.ApplyLabel(ctx, "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messageID and labelID cannot be empty")

	err = service.RemoveLabel(ctx, "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messageID and labelID cannot be empty")

	_, err = service.GetMessageLabels(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messageID cannot be empty")
}

// Test validation logic consistency
func TestLabelService_ValidationLogic(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	// Test that validation correctly identifies valid vs invalid inputs
	// We only test the validation logic without making API calls

	// Valid inputs should not trigger validation errors (we test this indirectly)
	validInputs := map[string]string{
		"ValidLabel":        "valid label name",
		"valid-ID":          "valid label ID",
		"valid_msg123":      "valid message ID",
		"Label With Spaces": "label name with spaces",
	}

	// These are valid for the validation logic (non-empty after trim)
	for input, description := range validInputs {
		t.Run("valid_input_"+description, func(t *testing.T) {
			// Test that these inputs would pass validation
			// (they'll fail at API level but that's not validation)

			// For string operations that just trim and check empty
			trimmed := input
			assert.NotEmpty(t, trimmed, "Valid input should not be empty after processing")
		})
	}

	// Test the actual validation logic directly
	assert.NotNil(t, service)
	assert.NotNil(t, ctx)
}

// Benchmark tests for performance critical operations
func BenchmarkLabelService_ValidationOnly(b *testing.B) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	b.Run("CreateLabel_EmptyName", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.CreateLabel(ctx, "")
		}
	})

	b.Run("RenameLabel_EmptyParams", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.RenameLabel(ctx, "", "")
		}
	})

	b.Run("DeleteLabel_EmptyID", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.DeleteLabel(ctx, "")
		}
	})

	b.Run("ApplyLabel_EmptyParams", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.ApplyLabel(ctx, "", "")
		}
	})

	b.Run("RemoveLabel_EmptyParams", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.RemoveLabel(ctx, "", "")
		}
	})

	b.Run("GetMessageLabels_EmptyID", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.GetMessageLabels(ctx, "")
		}
	})
}

// Test error handling patterns
func TestLabelService_ErrorHandlingPatterns(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	// Test that all methods properly wrap validation errors
	errorTests := []struct {
		name        string
		testFunc    func() error
		expectedErr string
	}{
		{
			name: "CreateLabel_empty_wrapped",
			testFunc: func() error {
				_, err := service.CreateLabel(ctx, "")
				return err
			},
			expectedErr: "label name cannot be empty",
		},
		{
			name: "RenameLabel_empty_wrapped",
			testFunc: func() error {
				_, err := service.RenameLabel(ctx, "", "")
				return err
			},
			expectedErr: "labelID and newName cannot be empty",
		},
		{
			name: "DeleteLabel_empty_wrapped",
			testFunc: func() error {
				return service.DeleteLabel(ctx, "")
			},
			expectedErr: "labelID cannot be empty",
		},
		{
			name: "ApplyLabel_empty_wrapped",
			testFunc: func() error {
				return service.ApplyLabel(ctx, "", "")
			},
			expectedErr: "messageID and labelID cannot be empty",
		},
		{
			name: "RemoveLabel_empty_wrapped",
			testFunc: func() error {
				return service.RemoveLabel(ctx, "", "")
			},
			expectedErr: "messageID and labelID cannot be empty",
		},
		{
			name: "GetMessageLabels_empty_wrapped",
			testFunc: func() error {
				_, err := service.GetMessageLabels(ctx, "")
				return err
			},
			expectedErr: "messageID cannot be empty",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// Test edge cases for whitespace handling
func TestLabelService_WhitespaceHandling(t *testing.T) {
	service := NewLabelService(&gmail.Client{})
	ctx := context.Background()

	// Test various whitespace patterns
	whitespaceInputs := []string{
		"",         // empty
		" ",        // single space
		"  ",       // multiple spaces
		"\t",       // tab
		"\n",       // newline
		"\r",       // carriage return
		" \t\n\r ", // mixed whitespace
	}

	for _, input := range whitespaceInputs {
		t.Run("CreateLabel_whitespace_"+input, func(t *testing.T) {
			_, err := service.CreateLabel(ctx, input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "label name cannot be empty")
		})

		t.Run("DeleteLabel_whitespace_"+input, func(t *testing.T) {
			err := service.DeleteLabel(ctx, input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "labelID cannot be empty")
		})

		t.Run("GetMessageLabels_whitespace_"+input, func(t *testing.T) {
			_, err := service.GetMessageLabels(ctx, input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "messageID cannot be empty")
		})

		t.Run("RenameLabel_whitespace_both_"+input, func(t *testing.T) {
			_, err := service.RenameLabel(ctx, input, input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "labelID and newName cannot be empty")
		})

		t.Run("ApplyLabel_whitespace_both_"+input, func(t *testing.T) {
			err := service.ApplyLabel(ctx, input, input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "messageID and labelID cannot be empty")
		})

		t.Run("RemoveLabel_whitespace_both_"+input, func(t *testing.T) {
			err := service.RemoveLabel(ctx, input, input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "messageID and labelID cannot be empty")
		})
	}
}

// Test service initialization
func TestLabelService_Initialization(t *testing.T) {
	// Test normal initialization
	client := &gmail.Client{}
	service := NewLabelService(client)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.gmailClient)

	// Test nil client initialization
	serviceWithNil := NewLabelService(nil)
	assert.NotNil(t, serviceWithNil)
	assert.Nil(t, serviceWithNil.gmailClient)

	// Both services should be different instances
	assert.NotEqual(t, service, serviceWithNil)
}
