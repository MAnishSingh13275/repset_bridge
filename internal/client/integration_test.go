package client

import (
	"context"
	"testing"
	"time"

	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCheckinIntegration tests the integration between CheckinClient and SubmissionService
func TestCheckinIntegration(t *testing.T) {
	// Setup
	httpClient := &MockHTTPClient{}
	queueManager := &MockQueueManager{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Create clients
	checkinClient := NewCheckinClient(httpClient, logger)
	submissionService := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test events
	testEvents := []types.StandardEvent{
		{
			EventID:        "evt_integration_1",
			ExternalUserID: "user_123",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device_123",
		},
		{
			EventID:        "evt_integration_2",
			ExternalUserID: "user_456",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeExit,
			IsSimulated:    false,
			DeviceID:       "device_123",
		},
	}

	// Create queued events
	queuedEvents := []queue.QueuedEvent{
		{
			ID:         1,
			Event:      testEvents[0],
			RetryCount: 0,
		},
		{
			ID:         2,
			Event:      testEvents[1],
			RetryCount: 0,
		},
	}

	// Mock expectations
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return(queuedEvents, nil)
	queueManager.On("MarkEventsSent", mock.Anything, []int64{1, 2}).Return(nil)

	httpClient.On("Do", mock.Anything, mock.MatchedBy(func(req *Request) bool {
		// Verify the request structure
		if req.Method != "POST" || req.Path != "/api/v1/checkin" || !req.RequireAuth {
			return false
		}

		// Verify the request body contains our events
		checkinReq, ok := req.Body.(CheckinRequest)
		if !ok || len(checkinReq.Events) != 2 {
			return false
		}

		// Verify event details
		return checkinReq.Events[0].EventID == "evt_integration_1" &&
			checkinReq.Events[1].EventID == "evt_integration_2"
	})).Return(&Response{
		StatusCode: 200,
		Body:       []byte(`{"success": true, "processedIds": ["evt_integration_1", "evt_integration_2"]}`),
	}, nil)

	// Execute the integration test
	ctx := context.Background()
	result, err := submissionService.SubmitPendingEvents(ctx)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalEvents)
	assert.Equal(t, 2, result.SuccessfulEvents)
	assert.Equal(t, 0, result.FailedEvents)
	assert.Empty(t, result.Errors)
	assert.True(t, result.Duration >= 0) // Duration should be non-negative

	// Verify all mocks were called as expected
	httpClient.AssertExpectations(t)
	queueManager.AssertExpectations(t)
}

// TestCheckinIntegration_IdempotencyKeyGeneration tests that idempotency keys are generated when missing
func TestCheckinIntegration_IdempotencyKeyGeneration(t *testing.T) {
	// Setup
	httpClient := &MockHTTPClient{}
	queueManager := &MockQueueManager{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	checkinClient := NewCheckinClient(httpClient, logger)
	submissionService := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test event without EventID
	testEvent := types.StandardEvent{
		ExternalUserID: "user_123",
		Timestamp:      time.Now(),
		EventType:      types.EventTypeEntry,
		IsSimulated:    false,
		DeviceID:       "device_123",
		// EventID is intentionally empty
	}

	queuedEvent := queue.QueuedEvent{
		ID:         1,
		Event:      testEvent,
		RetryCount: 0,
	}

	// Mock expectations
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return([]queue.QueuedEvent{queuedEvent}, nil)
	queueManager.On("MarkEventsSent", mock.Anything, mock.Anything).Return(nil)

	// Capture the generated event ID
	var capturedEventID string
	httpClient.On("Do", mock.Anything, mock.MatchedBy(func(req *Request) bool {
		checkinReq, ok := req.Body.(CheckinRequest)
		if ok && len(checkinReq.Events) == 1 {
			capturedEventID = checkinReq.Events[0].EventID
			return capturedEventID != "" // Should have generated an ID
		}
		return false
	})).Return(&Response{
		StatusCode: 200,
		Body:       []byte(`{"success": true, "processedIds": ["` + capturedEventID + `"]}`),
	}, nil)

	// Execute test
	ctx := context.Background()
	result, err := submissionService.SubmitPendingEvents(ctx)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalEvents)
	assert.Equal(t, 1, result.SuccessfulEvents)
	assert.Equal(t, 0, result.FailedEvents)

	// Verify that an idempotency key was generated
	assert.NotEmpty(t, capturedEventID)
	assert.Contains(t, capturedEventID, "evt_") // Should have our prefix

	httpClient.AssertExpectations(t)
	queueManager.AssertExpectations(t)
}

// TestCheckinIntegration_ErrorHandling tests error handling in the integration
func TestCheckinIntegration_ErrorHandling(t *testing.T) {
	// Setup
	httpClient := &MockHTTPClient{}
	queueManager := &MockQueueManager{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	checkinClient := NewCheckinClient(httpClient, logger)
	submissionService := NewSubmissionService(queueManager, checkinClient, logger)

	// Create test event
	testEvent := types.StandardEvent{
		EventID:        "evt_error_test",
		ExternalUserID: "user_123",
		Timestamp:      time.Now(),
		EventType:      types.EventTypeEntry,
		DeviceID:       "device_123",
	}

	queuedEvent := queue.QueuedEvent{
		ID:         1,
		Event:      testEvent,
		RetryCount: 0,
	}

	// Mock expectations for failure scenario
	queueManager.On("GetPendingEvents", mock.Anything, 50).Return([]queue.QueuedEvent{queuedEvent}, nil)
	queueManager.On("MarkEventsFailed", mock.Anything, []int64{1}, mock.MatchedBy(func(msg string) bool {
		return msg != ""
	})).Return(nil)

	// Mock HTTP error
	httpClient.On("Do", mock.Anything, mock.Anything).Return(&Response{
		StatusCode: 500,
		Body:       []byte(`{"error": "internal server error"}`),
	}, assert.AnError)

	// Execute test
	ctx := context.Background()
	result, err := submissionService.SubmitPendingEvents(ctx)

	// Verify error handling
	require.NoError(t, err) // Service should not return error, just handle it
	assert.Equal(t, 1, result.TotalEvents)
	assert.Equal(t, 0, result.SuccessfulEvents)
	assert.Equal(t, 1, result.FailedEvents)
	assert.NotEmpty(t, result.Errors)

	httpClient.AssertExpectations(t)
	queueManager.AssertExpectations(t)
}