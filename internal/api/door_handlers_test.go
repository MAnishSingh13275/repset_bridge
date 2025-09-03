package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/types"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAdapterRegistry is a mock implementation of AdapterRegistry
type MockAdapterRegistry struct {
	mock.Mock
}

func (m *MockAdapterRegistry) GetAllAdapters() []adapters.HardwareAdapter {
	args := m.Called()
	return args.Get(0).([]adapters.HardwareAdapter)
}

func (m *MockAdapterRegistry) GetAdapter(name string) (adapters.HardwareAdapter, error) {
	args := m.Called(name)
	return args.Get(0).(adapters.HardwareAdapter), args.Error(1)
}

func (m *MockAdapterRegistry) GetActiveAdapters() []adapters.HardwareAdapter {
	args := m.Called()
	return args.Get(0).([]adapters.HardwareAdapter)
}

// MockDoorController is a mock implementation of DoorController
type MockDoorController struct {
	mock.Mock
}

func (m *MockDoorController) UnlockDoor(ctx context.Context, adapterName string, durationMs int) error {
	args := m.Called(ctx, adapterName, durationMs)
	return args.Error(0)
}

func (m *MockDoorController) GetStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// MockHardwareAdapter is a mock implementation of HardwareAdapter
type MockHardwareAdapter struct {
	mock.Mock
}

func (m *MockHardwareAdapter) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHardwareAdapter) Initialize(ctx context.Context, config types.AdapterConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockHardwareAdapter) StartListening(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHardwareAdapter) StopListening(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHardwareAdapter) UnlockDoor(ctx context.Context, durationMs int) error {
	args := m.Called(ctx, durationMs)
	return args.Error(0)
}

func (m *MockHardwareAdapter) GetStatus() types.AdapterStatus {
	args := m.Called()
	return args.Get(0).(types.AdapterStatus)
}

func (m *MockHardwareAdapter) OnEvent(callback types.EventCallback) {
	m.Called(callback)
}

func (m *MockHardwareAdapter) IsHealthy() bool {
	args := m.Called()
	return args.Bool(0)
}

// Test setup helper
func setupTestHandlers() (*Handlers, *MockAdapterRegistry, *MockDoorController) {
	cfg := config.DefaultConfig()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	
	handlers := NewHandlers(cfg, logger, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	return handlers, mockAdapterRegistry, mockDoorController
}

func TestDoorUnlockRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request DoorUnlockRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: DoorUnlockRequest{
				DurationMs: 5000,
			},
			wantErr: false,
		},
		{
			name: "duration too short",
			request: DoorUnlockRequest{
				DurationMs: 500,
			},
			wantErr: true,
			errMsg:  "durationMs must be at least 1000 milliseconds",
		},
		{
			name: "duration too long",
			request: DoorUnlockRequest{
				DurationMs: 35000,
			},
			wantErr: true,
			errMsg:  "durationMs must not exceed 30000 milliseconds",
		},
		{
			name: "minimum valid duration",
			request: DoorUnlockRequest{
				DurationMs: 1000,
			},
			wantErr: false,
		},
		{
			name: "maximum valid duration",
			request: DoorUnlockRequest{
				DurationMs: 30000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandlers_UnlockDoor(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockAdapterRegistry, *MockDoorController)
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful unlock",
			requestBody: DoorUnlockRequest{
				DurationMs:  5000,
				Reason:      "test unlock",
				RequestedBy: "test user",
			},
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				mockAdapter := &MockHardwareAdapter{}
				mockAdapter.On("Name").Return("test-adapter")
				
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{mockAdapter})
				dc.On("UnlockDoor", mock.Anything, "", 5000).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "unlock with specific adapter",
			requestBody: DoorUnlockRequest{
				DurationMs: 3000,
				Adapter:    "specific-adapter",
			},
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				// When specific adapter is provided, we don't call GetActiveAdapters for the response
				dc.On("UnlockDoor", mock.Anything, "specific-adapter", 3000).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "invalid JSON",
			requestBody: "invalid json",
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				// No mocks needed for this test
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "validation error - duration too short",
			requestBody: DoorUnlockRequest{
				DurationMs: 500,
			},
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				// No mocks needed for this test
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "door controller error",
			requestBody: DoorUnlockRequest{
				DurationMs: 5000,
			},
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				// When unlock fails, we return early without determining adapter name
				dc.On("UnlockDoor", mock.Anything, "", 5000).Return(fmt.Errorf("adapter not available"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "zero duration uses default",
			requestBody: DoorUnlockRequest{
				DurationMs: 0, // Should use default from config
			},
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{})
				dc.On("UnlockDoor", mock.Anything, "", 3000).Return(nil) // Default is 3000
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, mockAdapterRegistry, mockDoorController := setupTestHandlers()
			tt.setupMocks(mockAdapterRegistry, mockDoorController)

			// Create request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			// Create HTTP request
			req := httptest.NewRequest("POST", "/api/v1/door/unlock", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handlers.UnlockDoor(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedError {
				assert.Contains(t, response, "error")
			} else {
				assert.Equal(t, true, response["success"])
				assert.Contains(t, response, "message")
				assert.Contains(t, response, "timestamp")
			}

			// Verify mocks
			mockAdapterRegistry.AssertExpectations(t)
			mockDoorController.AssertExpectations(t)
		})
	}
}

func TestHandlers_LockDoor(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockAdapterRegistry, *MockDoorController)
		expectedStatus int
	}{
		{
			name: "successful lock",
			requestBody: DoorLockRequest{
				Reason:      "test lock",
				RequestedBy: "test user",
			},
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				mockAdapter := &MockHardwareAdapter{}
				mockAdapter.On("Name").Return("test-adapter")
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{mockAdapter})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "lock with empty body",
			requestBody: nil,
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "lock with specific adapter",
			requestBody: DoorLockRequest{
				Adapter: "specific-adapter",
			},
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				// When specific adapter is provided, we don't call GetActiveAdapters
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, mockAdapterRegistry, mockDoorController := setupTestHandlers()
			tt.setupMocks(mockAdapterRegistry, mockDoorController)

			// Create request body
			var body []byte
			var err error
			if tt.requestBody != nil {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			// Create HTTP request
			req := httptest.NewRequest("POST", "/api/v1/door/lock", bytes.NewBuffer(body))
			if tt.requestBody != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			
			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handlers.LockDoor(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response DoorLockResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.True(t, response.Success)
			assert.Contains(t, response.Message, "lock")
			assert.NotEmpty(t, response.RequestID)

			// Verify mocks
			mockAdapterRegistry.AssertExpectations(t)
			mockDoorController.AssertExpectations(t)
		})
	}
}

func TestHandlers_DoorStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockAdapterRegistry, *MockDoorController)
		expectedStatus int
		expectedLocked bool
		expectedDoorStatus string
	}{
		{
			name: "door locked - no recent unlock",
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				mockAdapter := &MockHardwareAdapter{}
				mockAdapter.On("Name").Return("test-adapter")
				
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{mockAdapter})
				dc.On("GetStats").Return(map[string]interface{}{
					"unlockCount":    int64(5),
					"lastUnlockTime": time.Now().Add(-10 * time.Minute), // 10 minutes ago
				})
			},
			expectedStatus:     http.StatusOK,
			expectedLocked:     true,
			expectedDoorStatus: DoorStatusLocked,
		},
		{
			name: "door unlocked - recent unlock",
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				mockAdapter := &MockHardwareAdapter{}
				mockAdapter.On("Name").Return("test-adapter")
				
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{mockAdapter})
				dc.On("GetStats").Return(map[string]interface{}{
					"unlockCount":    int64(5),
					"lastUnlockTime": time.Now().Add(-1 * time.Second), // 1 second ago
				})
			},
			expectedStatus:     http.StatusOK,
			expectedLocked:     false,
			expectedDoorStatus: DoorStatusUnlocked,
		},
		{
			name: "door status unknown - no active adapters",
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{})
				dc.On("GetStats").Return(map[string]interface{}{
					"unlockCount": int64(0),
				})
			},
			expectedStatus:     http.StatusOK,
			expectedLocked:     false,
			expectedDoorStatus: DoorStatusUnknown,
		},
		{
			name: "door locked - no unlock history",
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController) {
				mockAdapter := &MockHardwareAdapter{}
				mockAdapter.On("Name").Return("test-adapter")
				
				ar.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{mockAdapter})
				dc.On("GetStats").Return(map[string]interface{}{
					"unlockCount": int64(0),
				})
			},
			expectedStatus:     http.StatusOK,
			expectedLocked:     true,
			expectedDoorStatus: DoorStatusLocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, mockAdapterRegistry, mockDoorController := setupTestHandlers()
			tt.setupMocks(mockAdapterRegistry, mockDoorController)

			// Create HTTP request
			req := httptest.NewRequest("GET", "/api/v1/door/status", nil)
			
			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handlers.DoorStatus(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response DoorStatusResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedLocked, response.IsLocked)
			assert.Equal(t, tt.expectedDoorStatus, response.Status)
			assert.NotZero(t, response.Timestamp)

			// Verify mocks
			mockAdapterRegistry.AssertExpectations(t)
			mockDoorController.AssertExpectations(t)
		})
	}
}

func TestGetInt64FromStats(t *testing.T) {
	tests := []struct {
		name     string
		stats    map[string]interface{}
		key      string
		expected int64
	}{
		{
			name: "int64 value",
			stats: map[string]interface{}{
				"count": int64(42),
			},
			key:      "count",
			expected: 42,
		},
		{
			name: "int value",
			stats: map[string]interface{}{
				"count": int(42),
			},
			key:      "count",
			expected: 42,
		},
		{
			name: "missing key",
			stats: map[string]interface{}{
				"other": int64(42),
			},
			key:      "count",
			expected: 0,
		},
		{
			name: "wrong type",
			stats: map[string]interface{}{
				"count": "42",
			},
			key:      "count",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInt64FromStats(tt.stats, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}