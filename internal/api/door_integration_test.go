package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/types"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDoorController is a test implementation of DoorController
type TestDoorController struct {
	unlockCount    int64
	lastUnlockTime time.Time
	failureCount   int64
	shouldFail     bool
}

func (t *TestDoorController) UnlockDoor(ctx context.Context, adapterName string, durationMs int) error {
	if t.shouldFail {
		t.failureCount++
		return assert.AnError
	}
	
	t.unlockCount++
	t.lastUnlockTime = time.Now()
	return nil
}

func (t *TestDoorController) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"unlockCount":    t.unlockCount,
		"lastUnlockTime": t.lastUnlockTime,
		"failureCount":   t.failureCount,
	}
}

// TestAdapterRegistry is a test implementation of AdapterRegistry
type TestAdapterRegistry struct {
	adapters []adapters.HardwareAdapter
}

func (t *TestAdapterRegistry) GetAllAdapters() []adapters.HardwareAdapter {
	return t.adapters
}

func (t *TestAdapterRegistry) GetAdapter(name string) (adapters.HardwareAdapter, error) {
	for _, adapter := range t.adapters {
		if adapter.Name() == name {
			return adapter, nil
		}
	}
	return nil, assert.AnError
}

func (t *TestAdapterRegistry) GetActiveAdapters() []adapters.HardwareAdapter {
	var active []adapters.HardwareAdapter
	for _, adapter := range t.adapters {
		if adapter.IsHealthy() {
			active = append(active, adapter)
		}
	}
	return active
}

// TestHardwareAdapter is a test implementation of HardwareAdapter
type TestHardwareAdapter struct {
	name      string
	healthy   bool
	unlockErr error
}

func (t *TestHardwareAdapter) Name() string {
	return t.name
}

func (t *TestHardwareAdapter) Initialize(ctx context.Context, config types.AdapterConfig) error {
	return nil
}

func (t *TestHardwareAdapter) StartListening(ctx context.Context) error {
	return nil
}

func (t *TestHardwareAdapter) StopListening(ctx context.Context) error {
	return nil
}

func (t *TestHardwareAdapter) UnlockDoor(ctx context.Context, durationMs int) error {
	return t.unlockErr
}

func (t *TestHardwareAdapter) GetStatus() types.AdapterStatus {
	status := types.StatusActive
	if !t.healthy {
		status = types.StatusError
	}
	
	return types.AdapterStatus{
		Name:      t.name,
		Status:    status,
		UpdatedAt: time.Now(),
	}
}

func (t *TestHardwareAdapter) OnEvent(callback types.EventCallback) {
	// No-op for tests
}

func (t *TestHardwareAdapter) IsHealthy() bool {
	return t.healthy
}

// setupIntegrationTest sets up a complete test environment
func setupIntegrationTest() (*mux.Router, *TestDoorController, *TestAdapterRegistry) {
	cfg := config.DefaultConfig()
	cfg.APIServer.Auth.Enabled = false // Disable auth for integration tests
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	
	// Create test adapters
	testAdapter1 := &TestHardwareAdapter{
		name:    "test-adapter-1",
		healthy: true,
	}
	testAdapter2 := &TestHardwareAdapter{
		name:    "test-adapter-2",
		healthy: true,
	}
	
	adapterRegistry := &TestAdapterRegistry{
		adapters: []adapters.HardwareAdapter{testAdapter1, testAdapter2},
	}
	
	doorController := &TestDoorController{}
	
	// Create handlers
	handlers := NewHandlers(cfg, logger, adapterRegistry, doorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	// Set up router with routes
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	
	// Door control endpoints
	api.HandleFunc("/door/unlock", handlers.UnlockDoor).Methods("POST")
	api.HandleFunc("/door/lock", handlers.LockDoor).Methods("POST")
	api.HandleFunc("/door/status", handlers.DoorStatus).Methods("GET")
	
	return router, doorController, adapterRegistry
}

func TestDoorControlIntegration_UnlockFlow(t *testing.T) {
	router, doorController, _ := setupIntegrationTest()
	
	// Test unlock request
	unlockReq := DoorUnlockRequest{
		DurationMs:  5000,
		Reason:      "integration test",
		RequestedBy: "test-user",
	}
	
	body, err := json.Marshal(unlockReq)
	require.NoError(t, err)
	
	req := httptest.NewRequest("POST", "/api/v1/door/unlock", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var unlockResp DoorUnlockResponse
	err = json.Unmarshal(w.Body.Bytes(), &unlockResp)
	require.NoError(t, err)
	
	assert.True(t, unlockResp.Success)
	assert.Equal(t, 5000, unlockResp.Duration)
	assert.Equal(t, "test-adapter-1", unlockResp.Adapter) // First active adapter
	assert.NotEmpty(t, unlockResp.RequestID)
	
	// Verify door controller state
	stats := doorController.GetStats()
	assert.Equal(t, int64(1), stats["unlockCount"])
	
	// Test status after unlock (should show unlocked)
	statusReq := httptest.NewRequest("GET", "/api/v1/door/status", nil)
	statusW := httptest.NewRecorder()
	router.ServeHTTP(statusW, statusReq)
	
	assert.Equal(t, http.StatusOK, statusW.Code)
	
	var statusResp DoorStatusResponse
	err = json.Unmarshal(statusW.Body.Bytes(), &statusResp)
	require.NoError(t, err)
	
	assert.False(t, statusResp.IsLocked)
	assert.Equal(t, DoorStatusUnlocked, statusResp.Status)
	assert.Equal(t, int64(1), statusResp.UnlockCount)
	assert.Len(t, statusResp.ActiveAdapters, 2)
	assert.NotNil(t, statusResp.LastUnlockTime)
}

func TestDoorControlIntegration_LockFlow(t *testing.T) {
	router, _, _ := setupIntegrationTest()
	
	// Test lock request
	lockReq := DoorLockRequest{
		Reason:      "integration test lock",
		RequestedBy: "test-user",
	}
	
	body, err := json.Marshal(lockReq)
	require.NoError(t, err)
	
	req := httptest.NewRequest("POST", "/api/v1/door/lock", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var lockResp DoorLockResponse
	err = json.Unmarshal(w.Body.Bytes(), &lockResp)
	require.NoError(t, err)
	
	assert.True(t, lockResp.Success)
	assert.Contains(t, lockResp.Message, "lock")
	assert.Equal(t, "test-adapter-1", lockResp.Adapter)
	assert.NotEmpty(t, lockResp.RequestID)
}

func TestDoorControlIntegration_StatusWithoutUnlock(t *testing.T) {
	router, _, _ := setupIntegrationTest()
	
	// Test status without any prior unlock
	req := httptest.NewRequest("GET", "/api/v1/door/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var statusResp DoorStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &statusResp)
	require.NoError(t, err)
	
	assert.True(t, statusResp.IsLocked)
	assert.Equal(t, DoorStatusLocked, statusResp.Status)
	assert.Equal(t, int64(0), statusResp.UnlockCount)
	assert.Len(t, statusResp.ActiveAdapters, 2)
	assert.Nil(t, statusResp.LastUnlockTime)
}

func TestDoorControlIntegration_NoActiveAdapters(t *testing.T) {
	router, _, adapterRegistry := setupIntegrationTest()
	
	// Make all adapters unhealthy
	for _, adapter := range adapterRegistry.adapters {
		if testAdapter, ok := adapter.(*TestHardwareAdapter); ok {
			testAdapter.healthy = false
		}
	}
	
	// Test status with no active adapters
	req := httptest.NewRequest("GET", "/api/v1/door/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var statusResp DoorStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &statusResp)
	require.NoError(t, err)
	
	assert.False(t, statusResp.IsLocked) // Can't determine lock status
	assert.Equal(t, DoorStatusUnknown, statusResp.Status)
	assert.Len(t, statusResp.ActiveAdapters, 0)
}

func TestDoorControlIntegration_UnlockFailure(t *testing.T) {
	router, doorController, _ := setupIntegrationTest()
	
	// Make door controller fail
	doorController.shouldFail = true
	
	unlockReq := DoorUnlockRequest{
		DurationMs: 3000,
	}
	
	body, err := json.Marshal(unlockReq)
	require.NoError(t, err)
	
	req := httptest.NewRequest("POST", "/api/v1/door/unlock", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Check error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var errorResp ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)
	
	assert.Equal(t, "true", errorResp.Error)
	assert.Equal(t, "UNLOCK_FAILED", errorResp.Code)
	assert.Contains(t, errorResp.Message, "Failed to unlock door")
	assert.NotEmpty(t, errorResp.RequestID)
}

func TestDoorControlIntegration_ValidationErrors(t *testing.T) {
	router, _, _ := setupIntegrationTest()
	
	tests := []struct {
		name        string
		requestBody interface{}
		expectedCode string
	}{
		{
			name:        "invalid JSON",
			requestBody: "invalid json",
			expectedCode: "INVALID_JSON",
		},
		{
			name: "duration too short",
			requestBody: DoorUnlockRequest{
				DurationMs: 500,
			},
			expectedCode: "VALIDATION_ERROR",
		},
		{
			name: "duration too long",
			requestBody: DoorUnlockRequest{
				DurationMs: 35000,
			},
			expectedCode: "VALIDATION_ERROR",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error
			
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}
			
			req := httptest.NewRequest("POST", "/api/v1/door/unlock", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusBadRequest, w.Code)
			
			var errorResp ErrorResponse
			err = json.Unmarshal(w.Body.Bytes(), &errorResp)
			require.NoError(t, err)
			
			assert.Equal(t, "true", errorResp.Error)
			assert.Equal(t, tt.expectedCode, errorResp.Code)
		})
	}
}

func TestDoorControlIntegration_StatusAfterTimeExpiry(t *testing.T) {
	router, doorController, _ := setupIntegrationTest()
	
	// Simulate an unlock that happened more than the default duration ago
	doorController.unlockCount = 1
	doorController.lastUnlockTime = time.Now().Add(-10 * time.Minute) // 10 minutes ago
	
	req := httptest.NewRequest("GET", "/api/v1/door/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var statusResp DoorStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &statusResp)
	require.NoError(t, err)
	
	// Door should be locked again after time expiry
	assert.True(t, statusResp.IsLocked)
	assert.Equal(t, DoorStatusLocked, statusResp.Status)
	assert.Equal(t, int64(1), statusResp.UnlockCount)
	assert.NotNil(t, statusResp.LastUnlockTime)
}