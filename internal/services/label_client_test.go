package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// MockLabelClient implements LabelClient for unit-testing LabelService.
type MockLabelClient struct{ mock.Mock }

func (m *MockLabelClient) ApplyLabel(messageID, labelID string) error {
	return m.Called(messageID, labelID).Error(0)
}
func (m *MockLabelClient) RemoveLabel(messageID, labelID string) error {
	return m.Called(messageID, labelID).Error(0)
}
func (m *MockLabelClient) CreateLabel(name string) (*gmail_v1.Label, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail_v1.Label), args.Error(1)
}
func (m *MockLabelClient) DeleteLabel(labelID string) error { return m.Called(labelID).Error(0) }
func (m *MockLabelClient) RenameLabel(labelID, newName string) (*gmail_v1.Label, error) {
	args := m.Called(labelID, newName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail_v1.Label), args.Error(1)
}
func (m *MockLabelClient) ListLabels() ([]*gmail_v1.Label, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gmail_v1.Label), args.Error(1)
}
func (m *MockLabelClient) GetMessage(id string) (*gmail_v1.Message, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail_v1.Message), args.Error(1)
}
func (m *MockLabelClient) ExtractLabels(msg *gmail_v1.Message) []string {
	return m.Called(msg).Get(0).([]string)
}

func TestLabelService_BulkApplyLabel(t *testing.T) {
	c := &MockLabelClient{}
	svc := NewLabelService(c)
	ctx := context.Background()
	assert.Error(t, svc.BulkApplyLabel(ctx, nil, "L1"))         // no IDs
	assert.Error(t, svc.BulkApplyLabel(ctx, []string{"a"}, "")) // empty labelID

	c.On("ApplyLabel", "a", "L1").Return(nil)
	c.On("ApplyLabel", "b", "L1").Return(nil)
	var prog [][2]int
	err := svc.BulkApplyLabel(ctx, []string{"a", "b"}, "L1", func(d, total int) {
		prog = append(prog, [2]int{d, total})
	})
	assert.NoError(t, err)
	assert.Equal(t, [][2]int{{1, 2}, {2, 2}}, prog)
	c.AssertExpectations(t)
}

func TestLabelService_BulkApplyLabel_PartialFailure(t *testing.T) {
	c := &MockLabelClient{}
	svc := NewLabelService(c)
	c.On("ApplyLabel", "a", "L1").Return(nil)
	c.On("ApplyLabel", "b", "L1").Return(errors.New("nope"))
	err := svc.BulkApplyLabel(context.Background(), []string{"a", "b"}, "L1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply label to b")
}

func TestLabelService_ApplyLabel(t *testing.T) {
	c := &MockLabelClient{}
	svc := NewLabelService(c)
	ctx := context.Background()
	assert.Error(t, svc.ApplyLabel(ctx, "", "L1")) // validation
	c.On("ApplyLabel", "m1", "L1").Return(nil)
	assert.NoError(t, svc.ApplyLabel(ctx, "m1", "L1"))
	c.AssertExpectations(t)
}

func TestLabelService_CreateLabel(t *testing.T) {
	c := &MockLabelClient{}
	svc := NewLabelService(c)
	ctx := context.Background()
	if _, err := svc.CreateLabel(ctx, "  "); err == nil {
		t.Error("empty name should error")
	}
	c.On("CreateLabel", "Work").Return(&gmail_v1.Label{Id: "L9", Name: "Work"}, nil)
	got, err := svc.CreateLabel(ctx, "Work")
	assert.NoError(t, err)
	assert.Equal(t, "L9", got.Id)
	c.AssertExpectations(t)
}

func TestLabelService_ListLabels(t *testing.T) {
	c := &MockLabelClient{}
	svc := NewLabelService(c)
	c.On("ListLabels").Return([]*gmail_v1.Label{{Id: "L1", Name: "Work"}, {Id: "L2", Name: "Home"}}, nil)
	got, err := svc.ListLabels(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, got)
}
