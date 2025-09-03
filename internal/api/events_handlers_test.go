package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/types"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandlers_GetEvents(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		mockEvents     []queue.QueuedEvent
		mockTotal      int64
		mockError      error
		expectedStatus int
		expectedEvents int
		expectError    bool
	}{
		{
			name: "successful query with no filters",
			mockEvents: []queue.QueuedEvent{
				{
					ID: 1,
					Event: types.StandardEvent{
						EventID:        "event1",
						ExternalUserID: "user1",
						Timestamp:      time.Now(),
						EventType:      types.EventTypeEntry,
						IsSimulated:    false,
						DeviceID:       "device1",
					},
					CreatedAt:  time.Now(),
					RetryCount: 0,
				},
				{
					ID: 2,
					Event: types.StandardEvent{
						EventID:        "event2",
						ExternalUserID: "user2",
						Timestamp:      time.Now(),
						EventType:      types.EventTypeExit,
						IsSimulated:    true,
						DeviceID:       "device1",
					},
					CreatedAt:  time.Now(),
					RetryCount: 0,
				},
			},
			mockTotal:      2,
			expectedStatus: http.StatusOK,
			expectedEvents: 2,
		},
		{
			name: "query with filters",
			queryParams: map[string]string{
				"eventType":   "entry",
				"userId":      "user1",
				"isSimulated": "false",
				"limit":       "10",
				"offset":      "0",
				"sortBy":      "timestamp",
				"sortOrder":   "desc",
			},
			mockEvents: []queue.QueuedEvent{
				{
					ID: 1,
					Event: types.StandardEvent{
						EventID:        "event1",
						ExternalUserID: "user1",
						Timestamp:      time.Now(),
						EventType:      types.EventTypeEntry,
						IsSimulated:    false,
						DeviceID:       "device1",
					},
					CreatedAt:  time.Now(),
					RetryCount: 0,
				},
			},
			mockTotal:      1,
			expectedStatus: http.StatusOK,
			expectedEvents: 1,
		},
		{
			name: "query with time range",
			queryParams: map[string]string{
				"startTime": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"endTime":   time.Now().Format(time.RFC3339),
			},
			mockEvents:     []queue.QueuedEvent{},
			mockTotal:      0,
			expectedStatus: http.StatusOK,
			expectedEvents: 0,
		},
		{
			name: "invalid time format",
			queryParams: map[string]string{
				"startTime": "invalid-time",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid boolean format",
			queryParams: map[string]string{
				"isSimulated": "invalid-bool",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid limit",
			queryParams: map[string]string{
				"limit": "invalid-number",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "limit too high",
			queryParams: map[string]string{
				"limit": "2000",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid event type",
			queryParams: map[string]string{
				"eventType": "invalid-type",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "queue manager error",
			mockError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			cfg := &config.Config{}
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
			
			mockQueueManager := &MockQueueManager{}
			
			handlers := NewHandlers(cfg, logger, nil, nil, nil, mockQueueManager, nil, nil, "test-version", "test-device")
			
			// Setup mock expectations if not expecting validation error
			if !tt.expectError || tt.mockError != nil {
				mockQueueManager.On("QueryEvents", mock.Anything, mock.AnythingOfType("queue.EventQueryFilter")).
					Return(tt.mockEvents, tt.mockTotal, tt.mockError)
			}
			
			// Create request
			req := httptest.NewRequest("GET", "/api/v1/events", nil)
			
			// Add query parameters
			if len(tt.queryParams) > 0 {
				q := req.URL.Query()
				for key, value := range tt.queryParams {
					q.Add(key, value)
				}
				req.URL.RawQuery = q.Encode()
			}
			
			w := httptest.NewRecorder()
			
			// Execute
			handlers.GetEvents(w, req)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectError {
				var errorResp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResp)
				assert.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
			} else {
				var response EventsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEvents, len(response.Events))
				assert.Equal(t, tt.mockTotal, response.Total)
				
				// Verify event data
				for i, event := range response.Events {
					assert.Equal(t, tt.mockEvents[i].Event.EventID, event.EventID)
					assert.Equal(t, tt.mockEvents[i].Event.ExternalUserID, event.ExternalUserID)
					assert.Equal(t, tt.mockEvents[i].Event.EventType, event.EventType)
					assert.Equal(t, tt.mockEvents[i].Event.IsSimulated, event.IsSimulated)
				}
			}
			
			mockQueueManager.AssertExpectations(t)
		})
	}
}

func TestHandlers_GetEventStats(t *testing.T) {
	tests := []struct {
		name           string
		mockStats      queue.EventStatistics
		mockError      error
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful stats retrieval",
			mockStats: queue.EventStatistics{
				TotalEvents: 100,
				EventsByType: map[string]int64{
					"entry": 60,
					"exit":  35,
					"denied": 5,
				},
				EventsByHour: map[string]int64{
					"2023-12-01 10:00:00": 10,
					"2023-12-01 11:00:00": 15,
				},
				EventsByDay: map[string]int64{
					"2023-12-01": 50,
					"2023-12-02": 50,
				},
				PendingEvents:   10,
				SentEvents:      85,
				FailedEvents:    5,
				UniqueUsers:     25,
				SimulatedEvents: 20,
				AveragePerHour:  4.2,
				AveragePerDay:   50.0,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "queue manager error",
			mockError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			cfg := &config.Config{}
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			
			mockQueueManager := &MockQueueManager{}
			
			handlers := NewHandlers(cfg, logger, nil, nil, nil, mockQueueManager, nil, nil, "test-version", "test-device")
			
			// Setup mock expectations
			mockQueueManager.On("GetEventStats", mock.Anything).Return(tt.mockStats, tt.mockError)
			
			// Create request
			req := httptest.NewRequest("GET", "/api/v1/events/stats", nil)
			w := httptest.NewRecorder()
			
			// Execute
			handlers.GetEventStats(w, req)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectError {
				var errorResp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResp)
				assert.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
			} else {
				var response EventStatsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockStats.TotalEvents, response.TotalEvents)
				assert.Equal(t, tt.mockStats.PendingEvents, response.PendingEvents)
				assert.Equal(t, tt.mockStats.SentEvents, response.SentEvents)
				assert.Equal(t, tt.mockStats.FailedEvents, response.FailedEvents)
				assert.Equal(t, tt.mockStats.UniqueUsers, response.UniqueUsers)
				assert.Equal(t, tt.mockStats.SimulatedEvents, response.SimulatedEvents)
				assert.Equal(t, tt.mockStats.EventsByType, response.EventsByType)
				assert.Equal(t, tt.mockStats.EventsByHour, response.EventsByHour)
				assert.Equal(t, tt.mockStats.EventsByDay, response.EventsByDay)
			}
			
			mockQueueManager.AssertExpectations(t)
		})
	}
}

func TestHandlers_ClearEvents(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    EventClearRequest
		mockDeleted    int64
		mockError      error
		expectedStatus int
		expectError    bool
	}{
		{
			name: "successful clear all events",
			requestBody: EventClearRequest{
				Confirm: true,
				Reason:  "Maintenance cleanup",
			},
			mockDeleted:    50,
			expectedStatus: http.StatusOK,
		},
		{
			name: "clear events older than date",
			requestBody: EventClearRequest{
				OlderThan: func() *time.Time { t := time.Now().Add(-30 * 24 * time.Hour); return &t }(),
				Confirm:   true,
				Reason:    "Archive old events",
			},
			mockDeleted:    25,
			expectedStatus: http.StatusOK,
		},
		{
			name: "clear only sent events",
			requestBody: EventClearRequest{
				OnlySent: true,
				Confirm:  true,
				Reason:   "Clean up sent events",
			},
			mockDeleted:    40,
			expectedStatus: http.StatusOK,
		},
		{
			name: "clear only failed events",
			requestBody: EventClearRequest{
				OnlyFailed: true,
				Confirm:    true,
				Reason:     "Clean up failed events",
			},
			mockDeleted:    5,
			expectedStatus: http.StatusOK,
		},
		{
			name: "clear specific event type",
			requestBody: EventClearRequest{
				EventType: "entry",
				Confirm:   true,
				Reason:    "Clean up entry events",
			},
			mockDeleted:    30,
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing confirmation",
			requestBody: EventClearRequest{
				Confirm: false,
				Reason:  "Test",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "invalid event type",
			requestBody: EventClearRequest{
				EventType: "invalid-type",
				Confirm:   true,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "both onlySent and onlyFailed true",
			requestBody: EventClearRequest{
				OnlySent:   true,
				OnlyFailed: true,
				Confirm:    true,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "queue manager error",
			requestBody: EventClearRequest{
				Confirm: true,
				Reason:  "Test",
			},
			mockError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			cfg := &config.Config{}
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			
			mockQueueManager := &MockQueueManager{}
			
			handlers := NewHandlers(cfg, logger, nil, nil, nil, mockQueueManager, nil, nil, "test-version", "test-device")
			
			// Setup mock expectations if not expecting validation error
			if !tt.expectError || tt.mockError != nil {
				mockQueueManager.On("ClearEvents", mock.Anything, mock.AnythingOfType("queue.EventClearCriteria")).
					Return(tt.mockDeleted, tt.mockError)
			}
			
			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("DELETE", "/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			// Execute
			handlers.ClearEvents(w, req)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectError {
				var errorResp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResp)
				assert.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
			} else {
				var response EventClearResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, tt.mockDeleted, response.DeletedCount)
				assert.NotEmpty(t, response.Message)
			}
			
			mockQueueManager.AssertExpectations(t)
		})
	}
}

func TestHandlers_GetEvents_NoQueueManager(t *testing.T) {
	// Setup handlers without queue manager
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "test-version", "test-device")
	
	// Create request
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handlers.GetEvents(w, req)
	
	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, "QUEUE_MANAGER_UNAVAILABLE", errorResp.Code)
}

func TestHandlers_GetEventStats_NoQueueManager(t *testing.T) {
	// Setup handlers without queue manager
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "test-version", "test-device")
	
	// Create request
	req := httptest.NewRequest("GET", "/api/v1/events/stats", nil)
	w := httptest.NewRecorder()
	
	// Execute
	handlers.GetEventStats(w, req)
	
	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, "QUEUE_MANAGER_UNAVAILABLE", errorResp.Code)
}

func TestHandlers_ClearEvents_NoQueueManager(t *testing.T) {
	// Setup handlers without queue manager
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "test-version", "test-device")
	
	// Create request
	requestBody := EventClearRequest{
		Confirm: true,
		Reason:  "Test",
	}
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("DELETE", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Execute
	handlers.ClearEvents(w, req)
	
	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var errorResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, "QUEUE_MANAGER_UNAVAILABLE", errorResp.Code)
}

func TestEventQueryRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		request     EventQueryRequest
		expectError bool
		errorMsg    string
	}{
		{
			name:    "valid request with defaults",
			request: EventQueryRequest{},
		},
		{
			name: "valid request with all fields",
			request: EventQueryRequest{
				StartTime:   func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
				EndTime:     func() *time.Time { t := time.Now(); return &t }(),
				EventType:   "entry",
				UserID:      "user123",
				IsSimulated: func() *bool { b := false; return &b }(),
				Limit:       50,
				Offset:      10,
				SortBy:      "timestamp",
				SortOrder:   "desc",
			},
		},
		{
			name: "negative limit",
			request: EventQueryRequest{
				Limit: -1,
			},
			expectError: true,
			errorMsg:    "limit must be non-negative",
		},
		{
			name: "limit too high",
			request: EventQueryRequest{
				Limit: 2000,
			},
			expectError: true,
			errorMsg:    "limit must not exceed 1000",
		},
		{
			name: "negative offset",
			request: EventQueryRequest{
				Offset: -1,
			},
			expectError: true,
			errorMsg:    "offset must be non-negative",
		},
		{
			name: "invalid event type",
			request: EventQueryRequest{
				EventType: "invalid",
			},
			expectError: true,
			errorMsg:    "eventType must be one of: entry, exit, denied",
		},
		{
			name: "invalid sort field",
			request: EventQueryRequest{
				SortBy: "invalid",
			},
			expectError: true,
			errorMsg:    "sortBy must be one of: timestamp, eventType, userId",
		},
		{
			name: "invalid sort order",
			request: EventQueryRequest{
				SortOrder: "invalid",
			},
			expectError: true,
			errorMsg:    "sortOrder must be either 'asc' or 'desc'",
		},
		{
			name: "start time after end time",
			request: EventQueryRequest{
				StartTime: func() *time.Time { t := time.Now(); return &t }(),
				EndTime:   func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
			},
			expectError: true,
			errorMsg:    "startTime must be before endTime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Check defaults are set
				if tt.request.Limit == 0 {
					assert.Equal(t, 100, tt.request.Limit)
				}
				if tt.request.SortBy == "" {
					assert.Equal(t, "timestamp", tt.request.SortBy)
				}
				if tt.request.SortOrder == "" {
					assert.Equal(t, "desc", tt.request.SortOrder)
				}
			}
		})
	}
}

func TestEventClearRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		request     EventClearRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: EventClearRequest{
				Confirm: true,
				Reason:  "Test cleanup",
			},
		},
		{
			name: "valid request with criteria",
			request: EventClearRequest{
				OlderThan:  func() *time.Time { t := time.Now().Add(-30 * 24 * time.Hour); return &t }(),
				EventType:  "entry",
				OnlySent:   true,
				Confirm:    true,
				Reason:     "Archive old entries",
			},
		},
		{
			name: "missing confirmation",
			request: EventClearRequest{
				Confirm: false,
			},
			expectError: true,
			errorMsg:    "confirm must be true",
		},
		{
			name: "invalid event type",
			request: EventClearRequest{
				EventType: "invalid",
				Confirm:   true,
			},
			expectError: true,
			errorMsg:    "eventType must be one of: entry, exit, denied",
		},
		{
			name: "both onlySent and onlyFailed true",
			request: EventClearRequest{
				OnlySent:   true,
				OnlyFailed: true,
				Confirm:    true,
			},
			expectError: true,
			errorMsg:    "onlySent and onlyFailed cannot both be true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			
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