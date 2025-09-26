package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockQueueManager is a mock implementation of QueueManager for testing
type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) Initialize(ctx context.Context, config queue.QueueConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockQueueManager) Enqueue(ctx context.Context, event types.StandardEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockQueueManager) GetPendingEvents(ctx context.Context, limit int) ([]queue.QueuedEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]queue.QueuedEvent), args.Error(1)
}

func (m *MockQueueManager) MarkEventsSent(ctx context.Context, eventIDs []int64) error {
	args := m.Called(ctx, eventIDs)
	return args.Error(0)
}

func (m *MockQueueManager) MarkEventsFailed(ctx context.Context, eventIDs []int64, errorMessage string) error {
	args := m.Called(ctx, eventIDs, errorMessage)
	return args.Error(0)
}

func (m *MockQueueManager) GetStats(ctx context.Context) (queue.QueueStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(queue.QueueStats), args.Error(1)
}

func (m *MockQueueManager) Cleanup(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQueueManager) GetQueueDepth(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockQueueManager) IsQueueFull(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockQueueManager) ClearEvents(ctx context.Context, criteria queue.EventClearCriteria) (int64, error) {
	args := m.Called(ctx, criteria)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueueManager) QueryEvents(ctx context.Context, filter queue.EventQueryFilter) ([]queue.QueuedEvent, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]queue.QueuedEvent), args.Get(1).(int64), args.Error(2)
}

func (m *MockQueueManager) GetEventStats(ctx context.Context) (queue.EventStatistics, error) {
	args := m.Called(ctx)
	return args.Get(0).(queue.EventStatistics), args.Error(1)
}

func (m *MockQueueManager) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockCheckinClient is a mock implementation of CheckinClientInterface for testing
type MockCheckinClient struct {
	mock.Mock
}

func (m *MockCheckinClient) SubmitEvents(ctx context.Context, events []types.StandardEvent) (*CheckinResponse, error) {
	args := m.Called(ctx, events)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CheckinResponse), args.Error(1)
}

func TestNewSubmissionService(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()

	service := NewSubmissionService(queueManager, checkinClient, logger)

	assert.NotNil(t, service)
	assert.Equal(t, queueManager, service.queueManager)
	assert.Equal(t, checkinClient, service.checkinClient)
	assert.Equal(t, logger, service.logger)
	assert.Equal(t, DefaultSubmissionConfig(), service.config)
}

func TestSubmissionService_SetConfig(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	newConfig := SubmissionConfig{
		BatchSize:      100,
		RetryInterval:  60 * time.Second,
		MaxRetries:     10,
		SubmitInterval: 30 * time.Second,
		MaxConcurrency: 5,
	}

	service.SetConfig(newConfig)
	assert.Equal(t, newConfig, service.config)
}

func TestSubmissionService_SubmitPendingEvents_NoPendingEvents(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	// Mock no pending events
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return([]queue.QueuedEvent{}, nil)

	ctx := context.Background()
	result, err := service.SubmitPendingEvents(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalEvents)
	assert.Equal(t, 0, result.SuccessfulEvents)
	assert.Equal(t, 0, result.FailedEvents)
	assert.True(t, result.Duration > 0)

	queueManager.AssertExpectations(t)
}

func TestSubmissionService_SubmitPendingEvents_Success(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test queued events
	queuedEvents := []queue.QueuedEvent{
		{
			ID: 1,
			Event: types.StandardEvent{
				EventID:        "evt_123",
				ExternalUserID: "user_123",
				Timestamp:      time.Now(),
				EventType:      types.EventTypeEntry,
				DeviceID:       "device_123",
			},
			RetryCount: 0,
		},
		{
			ID: 2,
			Event: types.StandardEvent{
				EventID:        "evt_456",
				ExternalUserID: "user_456",
				Timestamp:      time.Now(),
				EventType:      types.EventTypeExit,
				DeviceID:       "device_123",
			},
			RetryCount: 1,
		},
	}

	// Mock queue manager
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return(queuedEvents, nil)
	queueManager.On("MarkEventsSent", mock.Anything, []int64{1, 2}).Return(nil)

	// Mock successful checkin response
	checkinResponse := &CheckinResponse{
		Success:      true,
		ProcessedIDs: []string{"evt_123", "evt_456"},
	}
	checkinClient.On("SubmitEvents", mock.Anything, mock.MatchedBy(func(events []types.StandardEvent) bool {
		return len(events) == 2 && events[0].EventID == "evt_123" && events[1].EventID == "evt_456"
	})).Return(checkinResponse, nil)

	ctx := context.Background()
	result, err := service.SubmitPendingEvents(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalEvents)
	assert.Equal(t, 2, result.SuccessfulEvents)
	assert.Equal(t, 0, result.FailedEvents)
	assert.Empty(t, result.Errors)

	queueManager.AssertExpectations(t)
	checkinClient.AssertExpectations(t)
}

func TestSubmissionService_SubmitPendingEvents_PartialFailure(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test queued events
	queuedEvents := []queue.QueuedEvent{
		{
			ID: 1,
			Event: types.StandardEvent{
				EventID:        "evt_123",
				ExternalUserID: "user_123",
				Timestamp:      time.Now(),
				EventType:      types.EventTypeEntry,
				DeviceID:       "device_123",
			},
			RetryCount: 0,
		},
		{
			ID: 2,
			Event: types.StandardEvent{
				EventID:        "evt_456",
				ExternalUserID: "user_456",
				Timestamp:      time.Now(),
				EventType:      types.EventTypeExit,
				DeviceID:       "device_123",
			},
			RetryCount: 1,
		},
	}

	// Mock queue manager
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return(queuedEvents, nil)
	queueManager.On("MarkEventsSent", mock.Anything, []int64{1}).Return(nil)
	queueManager.On("MarkEventsFailed", mock.Anything, []int64{2}, "User not found").Return(nil)

	// Mock partial failure response
	checkinResponse := &CheckinResponse{
		Success:      false,
		ProcessedIDs: []string{"evt_123"},
		FailedIDs:    []string{"evt_456"},
		ErrorMessage: "User not found",
	}
	checkinClient.On("SubmitEvents", mock.Anything, mock.Anything).Return(checkinResponse, nil)

	ctx := context.Background()
	result, err := service.SubmitPendingEvents(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalEvents)
	assert.Equal(t, 1, result.SuccessfulEvents)
	assert.Equal(t, 1, result.FailedEvents)

	queueManager.AssertExpectations(t)
	checkinClient.AssertExpectations(t)
}

func TestSubmissionService_SubmitPendingEvents_CheckinError(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test queued events
	queuedEvents := []queue.QueuedEvent{
		{
			ID: 1,
			Event: types.StandardEvent{
				EventID:        "evt_123",
				ExternalUserID: "user_123",
				Timestamp:      time.Now(),
				EventType:      types.EventTypeEntry,
				DeviceID:       "device_123",
			},
			RetryCount: 0,
		},
	}

	// Mock queue manager
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return(queuedEvents, nil)
	queueManager.On("MarkEventsFailed", mock.Anything, []int64{1}, mock.MatchedBy(func(msg string) bool {
		return strings.Contains(msg, "network error")
	})).Return(nil)

	// Mock checkin error
	checkinClient.On("SubmitEvents", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("network error"))

	ctx := context.Background()
	result, err := service.SubmitPendingEvents(ctx)

	require.NoError(t, err) // Service should not return error, just log it
	assert.Equal(t, 1, result.TotalEvents)
	assert.Equal(t, 0, result.SuccessfulEvents)
	assert.Equal(t, 1, result.FailedEvents)
	assert.NotEmpty(t, result.Errors)

	queueManager.AssertExpectations(t)
	checkinClient.AssertExpectations(t)
}

func TestSubmissionService_SubmitPendingEvents_SkipMaxRetries(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test queued events with one that exceeded max retries
	queuedEvents := []queue.QueuedEvent{
		{
			ID: 1,
			Event: types.StandardEvent{
				EventID:        "evt_123",
				ExternalUserID: "user_123",
				Timestamp:      time.Now(),
				EventType:      types.EventTypeEntry,
				DeviceID:       "device_123",
			},
			RetryCount: 0, // Within limit
		},
		{
			ID: 2,
			Event: types.StandardEvent{
				EventID:        "evt_456",
				ExternalUserID: "user_456",
				Timestamp:      time.Now(),
				EventType:      types.EventTypeExit,
				DeviceID:       "device_123",
			},
			RetryCount: 10, // Exceeds default max retries (5)
		},
	}

	// Mock queue manager - should only get events within retry limit
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return(queuedEvents, nil)
	queueManager.On("MarkEventsSent", mock.Anything, []int64{1}).Return(nil)

	// Mock successful response for the one valid event
	checkinResponse := &CheckinResponse{
		Success:      true,
		ProcessedIDs: []string{"evt_123"},
	}
	checkinClient.On("SubmitEvents", mock.Anything, mock.MatchedBy(func(events []types.StandardEvent) bool {
		// Should only submit the event that hasn't exceeded max retries
		return len(events) == 1 && events[0].EventID == "evt_123"
	})).Return(checkinResponse, nil)

	ctx := context.Background()
	result, err := service.SubmitPendingEvents(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalEvents) // Total includes both events
	assert.Equal(t, 1, result.SuccessfulEvents) // Only one was submitted
	assert.Equal(t, 0, result.FailedEvents)

	queueManager.AssertExpectations(t)
	checkinClient.AssertExpectations(t)
}

func TestSubmissionService_GetQueueStats(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	expectedStats := queue.QueueStats{
		QueueDepth:    10,
		PendingEvents: 5,
		SentEvents:    100,
		FailedEvents:  2,
	}

	queueManager.On("GetStats", mock.Anything).Return(expectedStats, nil)

	ctx := context.Background()
	stats, err := service.GetQueueStats(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedStats, stats)

	queueManager.AssertExpectations(t)
}

func TestSubmissionService_SplitQueuedEventsIntoBatches(t *testing.T) {
	queueManager := &MockQueueManager{}
	checkinClient := &MockCheckinClient{}
	logger := logrus.New()
	service := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test events
	events := make([]queue.QueuedEvent, 125)
	for i := range events {
		events[i] = queue.QueuedEvent{
			ID: int64(i + 1),
			Event: types.StandardEvent{
				EventID:        fmt.Sprintf("evt_%d", i),
				ExternalUserID: fmt.Sprintf("user_%d", i),
				Timestamp:      time.Now(),
				EventType:      types.EventTypeEntry,
				DeviceID:       "device_123",
			},
		}
	}

	tests := []struct {
		name           string
		events         []queue.QueuedEvent
		batchSize      int
		expectedBatches int
	}{
		{
			name:           "default batch size",
			events:         events,
			batchSize:      0, // Should use default (50)
			expectedBatches: 3, // 125 events / 50 = 3 batches
		},
		{
			name:           "custom batch size",
			events:         events,
			batchSize:      25,
			expectedBatches: 5, // 125 events / 25 = 5 batches
		},
		{
			name:           "single batch",
			events:         events[:25],
			batchSize:      50,
			expectedBatches: 1,
		},
		{
			name:           "empty events",
			events:         []queue.QueuedEvent{},
			batchSize:      50,
			expectedBatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := service.splitQueuedEventsIntoBatches(tt.events, tt.batchSize)
			assert.Len(t, batches, tt.expectedBatches)

			// Verify all events are included
			totalEvents := 0
			for _, batch := range batches {
				totalEvents += len(batch)
			}
			assert.Equal(t, len(tt.events), totalEvents)
		})
	}
}

func TestDefaultSubmissionConfig(t *testing.T) {
	config := DefaultSubmissionConfig()

	assert.Equal(t, 50, config.BatchSize)
	assert.Equal(t, 30*time.Second, config.RetryInterval)
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 10*time.Second, config.SubmitInterval)
	assert.Equal(t, 3, config.MaxConcurrency)
}