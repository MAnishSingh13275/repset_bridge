package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient is a mock implementation of HTTPClientInterface for testing
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(ctx context.Context, req *Request) (*Response, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Response), args.Error(1)
}

func TestNewCheckinClient(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()

	client := NewCheckinClient(httpClient, logger)

	assert.NotNil(t, client)
	assert.Equal(t, httpClient, client.httpClient)
	assert.Equal(t, logger, client.logger)
}

func TestCheckinClient_SubmitEvents_Success(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	// Create test events
	events := []types.StandardEvent{
		{
			EventID:        "evt_123",
			ExternalUserID: "user_123",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device_123",
		},
		{
			EventID:        "evt_456",
			ExternalUserID: "user_456",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeExit,
			IsSimulated:    false,
			DeviceID:       "device_123",
		},
	}

	// Mock successful response
	responseBody := CheckinResponse{
		Success:      true,
		ProcessedIDs: []string{"evt_123", "evt_456"},
	}
	responseJSON, _ := json.Marshal(responseBody)

	httpClient.On("Do", mock.Anything, mock.MatchedBy(func(req *Request) bool {
		return req.Method == http.MethodPost &&
			req.Path == "/api/v1/checkin" &&
			req.RequireAuth == true
	})).Return(&Response{
		StatusCode: 200,
		Body:       responseJSON,
	}, nil)

	// Execute test
	ctx := context.Background()
	resp, err := client.SubmitEvents(ctx, events)

	// Verify results
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, []string{"evt_123", "evt_456"}, resp.ProcessedIDs)
	assert.Empty(t, resp.FailedIDs)

	httpClient.AssertExpectations(t)
}

func TestCheckinClient_SubmitEvents_EmptyEvents(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	ctx := context.Background()
	resp, err := client.SubmitEvents(ctx, []types.StandardEvent{})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Empty(t, resp.ProcessedIDs)
	assert.Empty(t, resp.FailedIDs)

	// Should not make any HTTP calls
	httpClient.AssertNotCalled(t, "Do")
}

func TestCheckinClient_SubmitEvents_HTTPError(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	events := []types.StandardEvent{
		{
			EventID:        "evt_123",
			ExternalUserID: "user_123",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			DeviceID:       "device_123",
		},
	}

	// Mock HTTP error
	httpClient.On("Do", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("network error"))

	ctx := context.Background()
	resp, err := client.SubmitEvents(ctx, events)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to submit events")

	httpClient.AssertExpectations(t)
}

func TestCheckinClient_SubmitEvents_PartialFailure(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	events := []types.StandardEvent{
		{
			EventID:        "evt_123",
			ExternalUserID: "user_123",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			DeviceID:       "device_123",
		},
		{
			EventID:        "evt_456",
			ExternalUserID: "user_456",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeExit,
			DeviceID:       "device_123",
		},
	}

	// Mock partial failure response
	responseBody := CheckinResponse{
		Success:      false,
		ProcessedIDs: []string{"evt_123"},
		FailedIDs:    []string{"evt_456"},
		ErrorMessage: "User not found for evt_456",
	}
	responseJSON, _ := json.Marshal(responseBody)

	httpClient.On("Do", mock.Anything, mock.Anything).Return(&Response{
		StatusCode: 200,
		Body:       responseJSON,
	}, nil)

	ctx := context.Background()
	resp, err := client.SubmitEvents(ctx, events)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, []string{"evt_123"}, resp.ProcessedIDs)
	assert.Equal(t, []string{"evt_456"}, resp.FailedIDs)
	assert.Equal(t, "User not found for evt_456", resp.ErrorMessage)

	httpClient.AssertExpectations(t)
}

func TestCheckinClient_SubmitSingleEvent(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	event := types.StandardEvent{
		EventID:        "evt_123",
		ExternalUserID: "user_123",
		Timestamp:      time.Now(),
		EventType:      types.EventTypeEntry,
		DeviceID:       "device_123",
	}

	responseBody := CheckinResponse{
		Success:      true,
		ProcessedIDs: []string{"evt_123"},
	}
	responseJSON, _ := json.Marshal(responseBody)

	httpClient.On("Do", mock.Anything, mock.Anything).Return(&Response{
		StatusCode: 200,
		Body:       responseJSON,
	}, nil)

	ctx := context.Background()
	resp, err := client.SubmitSingleEvent(ctx, event)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, []string{"evt_123"}, resp.ProcessedIDs)

	httpClient.AssertExpectations(t)
}

func TestCheckinClient_GenerateIdempotencyKey(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	key1, err1 := client.generateIdempotencyKey()
	key2, err2 := client.generateIdempotencyKey()

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
	assert.True(t, strings.HasPrefix(key1, "evt_"))
	assert.True(t, strings.HasPrefix(key2, "evt_"))
}

func TestCheckinClient_SubmitEvents_GeneratesIdempotencyKeys(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	// Create events without EventID
	events := []types.StandardEvent{
		{
			ExternalUserID: "user_123",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			DeviceID:       "device_123",
		},
	}

	var capturedRequest *Request
	httpClient.On("Do", mock.Anything, mock.MatchedBy(func(req *Request) bool {
		capturedRequest = req
		return true
	})).Return(&Response{
		StatusCode: 200,
		Body:       []byte(`{"success": true, "processedIds": ["generated_key"]}`),
	}, nil)

	ctx := context.Background()
	_, err := client.SubmitEvents(ctx, events)

	require.NoError(t, err)
	require.NotNil(t, capturedRequest)

	// Verify that idempotency key was generated
	checkinReq := capturedRequest.Body.(CheckinRequest)
	assert.NotEmpty(t, checkinReq.Events[0].EventID)
	assert.True(t, strings.HasPrefix(checkinReq.Events[0].EventID, "evt_"))

	httpClient.AssertExpectations(t)
}

func TestCheckinClient_ValidateEvents(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	tests := []struct {
		name        string
		events      []types.StandardEvent
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty events",
			events:      []types.StandardEvent{},
			expectError: true,
			errorMsg:    "no events to validate",
		},
		{
			name: "valid events",
			events: []types.StandardEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_123",
					Timestamp:      time.Now(),
					EventType:      types.EventTypeEntry,
					DeviceID:       "device_123",
				},
			},
			expectError: false,
		},
		{
			name: "missing external user ID",
			events: []types.StandardEvent{
				{
					EventID:   "evt_123",
					Timestamp: time.Now(),
					EventType: types.EventTypeEntry,
					DeviceID:  "device_123",
				},
			},
			expectError: true,
			errorMsg:    "externalUserId is required",
		},
		{
			name: "missing timestamp",
			events: []types.StandardEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_123",
					EventType:      types.EventTypeEntry,
					DeviceID:       "device_123",
				},
			},
			expectError: true,
			errorMsg:    "timestamp is required",
		},
		{
			name: "invalid event type",
			events: []types.StandardEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_123",
					Timestamp:      time.Now(),
					EventType:      "invalid",
					DeviceID:       "device_123",
				},
			},
			expectError: true,
			errorMsg:    "invalid event type",
		},
		{
			name: "missing device ID",
			events: []types.StandardEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_123",
					Timestamp:      time.Now(),
					EventType:      types.EventTypeEntry,
				},
			},
			expectError: true,
			errorMsg:    "deviceId is required",
		},
		{
			name: "timestamp too far in future",
			events: []types.StandardEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_123",
					Timestamp:      time.Now().Add(10 * time.Minute),
					EventType:      types.EventTypeEntry,
					DeviceID:       "device_123",
				},
			},
			expectError: true,
			errorMsg:    "timestamp is too far in the future",
		},
		{
			name: "timestamp too old",
			events: []types.StandardEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_123",
					Timestamp:      time.Now().Add(-8 * 24 * time.Hour),
					EventType:      types.EventTypeEntry,
					DeviceID:       "device_123",
				},
			},
			expectError: true,
			errorMsg:    "timestamp is too old",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.ValidateEvents(tt.events)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckinClient_GetMaxBatchSize(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	maxSize := client.GetMaxBatchSize()
	assert.Equal(t, 100, maxSize)
}

func TestCheckinClient_SplitIntoBatches(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	// Create test events
	events := make([]types.StandardEvent, 250)
	for i := range events {
		events[i] = types.StandardEvent{
			EventID:        fmt.Sprintf("evt_%d", i),
			ExternalUserID: fmt.Sprintf("user_%d", i),
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			DeviceID:       "device_123",
		}
	}

	tests := []struct {
		name           string
		events         []types.StandardEvent
		batchSize      int
		expectedBatches int
	}{
		{
			name:           "default batch size",
			events:         events,
			batchSize:      0, // Should use default (100)
			expectedBatches: 3, // 250 events / 100 = 3 batches
		},
		{
			name:           "custom batch size",
			events:         events,
			batchSize:      50,
			expectedBatches: 5, // 250 events / 50 = 5 batches
		},
		{
			name:           "single batch",
			events:         events[:50],
			batchSize:      100,
			expectedBatches: 1,
		},
		{
			name:           "empty events",
			events:         []types.StandardEvent{},
			batchSize:      100,
			expectedBatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := client.SplitIntoBatches(tt.events, tt.batchSize)
			assert.Len(t, batches, tt.expectedBatches)

			// Verify all events are included
			totalEvents := 0
			for _, batch := range batches {
				totalEvents += len(batch)
			}
			assert.Equal(t, len(tt.events), totalEvents)

			// Verify batch sizes
			expectedBatchSize := tt.batchSize
			if expectedBatchSize <= 0 {
				expectedBatchSize = client.GetMaxBatchSize()
			}

			for i, batch := range batches {
				if i < len(batches)-1 {
					// All batches except the last should be full size
					assert.Equal(t, expectedBatchSize, len(batch))
				} else {
					// Last batch can be smaller
					assert.LessOrEqual(t, len(batch), expectedBatchSize)
				}
			}
		})
	}
}

func TestCheckinClient_SubmitEventsInBatches_Success(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	// Create 150 test events (will be split into 2 batches of 100 and 50)
	events := make([]types.StandardEvent, 150)
	for i := range events {
		events[i] = types.StandardEvent{
			EventID:        fmt.Sprintf("evt_%d", i),
			ExternalUserID: fmt.Sprintf("user_%d", i),
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			DeviceID:       "device_123",
		}
	}

	// Mock successful responses for both batches
	httpClient.On("Do", mock.Anything, mock.Anything).Return(&Response{
		StatusCode: 200,
		Body:       []byte(`{"success": true, "processedIds": []}`),
	}, nil).Times(2)

	ctx := context.Background()
	responses, err := client.SubmitEventsInBatches(ctx, events)

	require.NoError(t, err)
	assert.Len(t, responses, 2) // Should have 2 batch responses
	for _, resp := range responses {
		assert.True(t, resp.Success)
	}

	httpClient.AssertExpectations(t)
}

func TestCheckinClient_SubmitEventsInBatches_ValidationError(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	// Create invalid events
	events := []types.StandardEvent{
		{
			EventID:   "evt_123",
			Timestamp: time.Now(),
			EventType: types.EventTypeEntry,
			DeviceID:  "device_123",
			// Missing ExternalUserID
		},
	}

	ctx := context.Background()
	responses, err := client.SubmitEventsInBatches(ctx, events)

	assert.Error(t, err)
	assert.Nil(t, responses)
	assert.Contains(t, err.Error(), "event validation failed")

	// Should not make any HTTP calls
	httpClient.AssertNotCalled(t, "Do")
}

func TestCheckinClient_SubmitEventsInBatches_PartialFailure(t *testing.T) {
	httpClient := &MockHTTPClient{}
	logger := logrus.New()
	client := NewCheckinClient(httpClient, logger)

	// Create 150 test events
	events := make([]types.StandardEvent, 150)
	for i := range events {
		events[i] = types.StandardEvent{
			EventID:        fmt.Sprintf("evt_%d", i),
			ExternalUserID: fmt.Sprintf("user_%d", i),
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			DeviceID:       "device_123",
		}
	}

	// First batch succeeds, second batch fails
	httpClient.On("Do", mock.Anything, mock.Anything).Return(&Response{
		StatusCode: 200,
		Body:       []byte(`{"success": true, "processedIds": []}`),
	}, nil).Once()

	httpClient.On("Do", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("network error")).Once()

	ctx := context.Background()
	responses, err := client.SubmitEventsInBatches(ctx, events)

	assert.Error(t, err)
	assert.Len(t, responses, 1) // Should have 1 successful response
	assert.True(t, responses[0].Success)
	assert.Contains(t, err.Error(), "failed to submit batch 2")

	httpClient.AssertExpectations(t)
}