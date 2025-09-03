package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHealthMonitor is a mock implementation of HealthMonitor
type MockHealthMonitor struct {
	mock.Mock
}

func (m *MockHealthMonitor) GetCurrentHealth() SystemHealth {
	args := m.Called()
	return args.Get(0).(SystemHealth)
}

func (m *MockHealthMonitor) UpdateHealth(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockQueueManager is a mock implementation of QueueManager
type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) GetQueueDepth(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockQueueManager) GetStats(ctx context.Context) (QueueStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(QueueStats), args.Error(1)
}

func (m *MockQueueManager) QueryEvents(ctx context.Context, filter queue.EventQueryFilter) ([]queue.QueuedEvent, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]queue.QueuedEvent), args.Get(1).(int64), args.Error(2)
}

func (m *MockQueueManager) GetEventStats(ctx context.Context) (queue.EventStatistics, error) {
	args := m.Called(ctx)
	return args.Get(0).(queue.EventStatistics), args.Error(1)
}

func (m *MockQueueManager) ClearEvents(ctx context.Context, criteria queue.EventClearCriteria) (int64, error) {
	args := m.Called(ctx, criteria)
	return args.Get(0).(int64), args.Error(1)
}

// MockTierDetector is a mock implementation of TierDetector
type MockTierDetector struct {
	mock.Mock
}

func (m *MockTierDetector) GetCurrentTier() Tier {
	args := m.Called()
	return args.Get(0).(Tier)
}

func (m *MockTierDetector) GetCurrentResources() SystemResources {
	args := m.Called()
	return args.Get(0).(SystemResources)
}

// Test setup helper for status endpoints
func setupStatusTestHandlers() (*Handlers, *MockAdapterRegistry, *MockDoorController, *MockHealthMonitor, *MockQueueManager, *MockTierDetector) {
	handlers, mockAdapterRegistry, mockDoorController := setupTestHandlers()
	
	// Set start time to 1 hour ago to ensure uptime > 0
	handlers.startTime = time.Now().Add(-time.Hour)
	
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	
	// Update handlers with additional mocks
	handlers.healthMonitor = mockHealthMonitor
	handlers.queueManager = mockQueueManager
	handlers.tierDetector = mockTierDetector
	
	return handlers, mockAdapterRegistry, mockDoorController, mockHealthMonitor, mockQueueManager, mockTierDetector
}

func TestHandlers_HealthCheck_WithHealthMonitor(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockHealthMonitor)
		expectedStatus int
		expectedHealth string
	}{
		{
			name: "healthy status",
			setupMocks: func(hm *MockHealthMonitor) {
				hm.On("UpdateHealth", mock.Anything).Return(nil)
				hm.On("GetCurrentHealth").Return(SystemHealth{
					Status:    "healthy",
					Timestamp: time.Now().UTC(),
					Version:   "test-version",
					DeviceID:  "test-device-id",
					Uptime:    time.Hour,
					QueueDepth: 5,
					AdapterStatus: []AdapterStatus{
						{
							Name:      "test-adapter",
							Status:    "active",
							LastEvent: time.Now().Add(-time.Minute),
							UpdatedAt: time.Now(),
						},
					},
					Resources: SystemResources{
						CPUCores:    4,
						MemoryGB:    8.0,
						CPUUsage:    25.5,
						MemoryUsage: 60.0,
						DiskUsage:   45.0,
						LastUpdated: time.Now(),
					},
					Tier: "standard",
					LastEventTime: func() *time.Time { t := time.Now().Add(-time.Minute); return &t }(),
				})
			},
			expectedStatus: http.StatusOK,
			expectedHealth: "healthy",
		},
		{
			name: "unhealthy status",
			setupMocks: func(hm *MockHealthMonitor) {
				hm.On("UpdateHealth", mock.Anything).Return(nil)
				hm.On("GetCurrentHealth").Return(SystemHealth{
					Status:    "unhealthy",
					Timestamp: time.Now().UTC(),
					Version:   "test-version",
					DeviceID:  "test-device-id",
					Uptime:    time.Hour,
					QueueDepth: 0,
					AdapterStatus: []AdapterStatus{
						{
							Name:         "test-adapter",
							Status:       "error",
							ErrorMessage: "Connection failed",
							UpdatedAt:    time.Now(),
						},
					},
					Resources: SystemResources{
						CPUCores:    4,
						MemoryGB:    8.0,
						CPUUsage:    95.0,
						MemoryUsage: 90.0,
						DiskUsage:   85.0,
						LastUpdated: time.Now(),
					},
					Tier: "basic",
				})
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "unhealthy",
		},
		{
			name: "degraded status",
			setupMocks: func(hm *MockHealthMonitor) {
				hm.On("UpdateHealth", mock.Anything).Return(nil)
				hm.On("GetCurrentHealth").Return(SystemHealth{
					Status:    "degraded",
					Timestamp: time.Now().UTC(),
					Version:   "test-version",
					DeviceID:  "test-device-id",
					Uptime:    time.Hour,
					QueueDepth: 10,
					AdapterStatus: []AdapterStatus{
						{
							Name:      "test-adapter-1",
							Status:    "active",
							UpdatedAt: time.Now(),
						},
						{
							Name:         "test-adapter-2",
							Status:       "error",
							ErrorMessage: "Intermittent failures",
							UpdatedAt:    time.Now(),
						},
					},
					Resources: SystemResources{
						CPUCores:    4,
						MemoryGB:    8.0,
						CPUUsage:    75.0,
						MemoryUsage: 80.0,
						DiskUsage:   70.0,
						LastUpdated: time.Now(),
					},
					Tier: "standard",
				})
			},
			expectedStatus: http.StatusOK,
			expectedHealth: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _, _, mockHealthMonitor, _, _ := setupStatusTestHandlers()
			tt.setupMocks(mockHealthMonitor)

			// Create HTTP request
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()

			// Call handler
			handlers.HealthCheck(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response
			var response HealthCheckResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verify response fields
			assert.Equal(t, tt.expectedHealth, response.Status)
			assert.Equal(t, "test-version", response.Version)
			assert.Equal(t, "test-device-id", response.DeviceID)
			assert.NotZero(t, response.Timestamp)
			assert.NotZero(t, response.Uptime)

			// Verify mocks
			mockHealthMonitor.AssertExpectations(t)
		})
	}
}

func TestHandlers_HealthCheck_WithoutHealthMonitor(t *testing.T) {
	handlers, _, _ := setupTestHandlers()
	// Don't set health monitor (nil)
	// Set start time to 1 hour ago to ensure uptime > 0
	handlers.startTime = time.Now().Add(-time.Hour)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handlers.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response HealthCheckResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Should fall back to basic health check
	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "test-version", response.Version)
	assert.Equal(t, "test-device-id", response.DeviceID)
	assert.NotZero(t, response.Timestamp)
	assert.True(t, response.Uptime > 0) // Use True instead of NotZero for duration
}

func TestHandlers_DeviceStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockAdapterRegistry, *MockQueueManager, *MockTierDetector)
		expectedStatus int
		expectedDevice string
	}{
		{
			name: "healthy device with all components",
			setupMocks: func(ar *MockAdapterRegistry, qm *MockQueueManager, td *MockTierDetector) {
				// Mock adapters
				mockAdapter1 := &MockHardwareAdapter{}
				mockAdapter1.On("GetStatus").Return(types.AdapterStatus{
					Name:      "adapter-1",
					Status:    "active",
					LastEvent: time.Now().Add(-time.Minute),
					UpdatedAt: time.Now(),
				})
				
				mockAdapter2 := &MockHardwareAdapter{}
				mockAdapter2.On("GetStatus").Return(types.AdapterStatus{
					Name:      "adapter-2",
					Status:    "active",
					LastEvent: time.Now().Add(-2 * time.Minute),
					UpdatedAt: time.Now(),
				})
				
				ar.On("GetAllAdapters").Return([]adapters.HardwareAdapter{mockAdapter1, mockAdapter2})
				
				// Mock queue manager
				qm.On("GetQueueDepth", mock.Anything).Return(5, nil)
				qm.On("GetStats", mock.Anything).Return(QueueStats{
					QueueDepth:      5,
					PendingEvents:   3,
					SentEvents:      100,
					FailedEvents:    2,
					LastSentAt:      time.Now().Add(-time.Minute),
					LastFailureAt:   time.Now().Add(-time.Hour),
					OldestEventTime: time.Now().Add(-5 * time.Minute),
				}, nil)
				
				// Mock tier detector
				td.On("GetCurrentResources").Return(SystemResources{
					CPUCores:    8,
					MemoryGB:    16.0,
					CPUUsage:    30.0,
					MemoryUsage: 45.0,
					DiskUsage:   60.0,
					LastUpdated: time.Now(),
				})
				td.On("GetCurrentTier").Return(Tier("premium"))
			},
			expectedStatus: http.StatusOK,
			expectedDevice: "healthy",
		},
		{
			name: "degraded device with some adapter errors",
			setupMocks: func(ar *MockAdapterRegistry, qm *MockQueueManager, td *MockTierDetector) {
				// Mock adapters - one healthy, one error
				mockAdapter1 := &MockHardwareAdapter{}
				mockAdapter1.On("GetStatus").Return(types.AdapterStatus{
					Name:      "adapter-1",
					Status:    "active",
					LastEvent: time.Now().Add(-time.Minute),
					UpdatedAt: time.Now(),
				})
				
				mockAdapter2 := &MockHardwareAdapter{}
				mockAdapter2.On("GetStatus").Return(types.AdapterStatus{
					Name:         "adapter-2",
					Status:       "error",
					ErrorMessage: "Connection timeout",
					UpdatedAt:    time.Now(),
				})
				
				ar.On("GetAllAdapters").Return([]adapters.HardwareAdapter{mockAdapter1, mockAdapter2})
				
				// Mock queue manager
				qm.On("GetQueueDepth", mock.Anything).Return(10, nil)
				qm.On("GetStats", mock.Anything).Return(QueueStats{
					QueueDepth:      10,
					PendingEvents:   8,
					SentEvents:      50,
					FailedEvents:    5,
					LastSentAt:      time.Now().Add(-2 * time.Minute),
					LastFailureAt:   time.Now().Add(-time.Minute),
					OldestEventTime: time.Now().Add(-10 * time.Minute),
				}, nil)
				
				// Mock tier detector
				td.On("GetCurrentResources").Return(SystemResources{
					CPUCores:    4,
					MemoryGB:    8.0,
					CPUUsage:    70.0,
					MemoryUsage: 75.0,
					DiskUsage:   80.0,
					LastUpdated: time.Now(),
				})
				td.On("GetCurrentTier").Return(Tier("standard"))
			},
			expectedStatus: http.StatusOK,
			expectedDevice: "degraded",
		},
		{
			name: "unhealthy device with all adapters in error",
			setupMocks: func(ar *MockAdapterRegistry, qm *MockQueueManager, td *MockTierDetector) {
				// Mock adapters - all in error
				mockAdapter1 := &MockHardwareAdapter{}
				mockAdapter1.On("GetStatus").Return(types.AdapterStatus{
					Name:         "adapter-1",
					Status:       "error",
					ErrorMessage: "Hardware failure",
					UpdatedAt:    time.Now(),
				})
				
				mockAdapter2 := &MockHardwareAdapter{}
				mockAdapter2.On("GetStatus").Return(types.AdapterStatus{
					Name:         "adapter-2",
					Status:       "error",
					ErrorMessage: "Network unreachable",
					UpdatedAt:    time.Now(),
				})
				
				ar.On("GetAllAdapters").Return([]adapters.HardwareAdapter{mockAdapter1, mockAdapter2})
				
				// Mock queue manager
				qm.On("GetQueueDepth", mock.Anything).Return(0, nil)
				qm.On("GetStats", mock.Anything).Return(QueueStats{
					QueueDepth:      0,
					PendingEvents:   0,
					SentEvents:      10,
					FailedEvents:    20,
					LastSentAt:      time.Now().Add(-time.Hour),
					LastFailureAt:   time.Now().Add(-time.Minute),
					OldestEventTime: time.Now().Add(-2 * time.Hour),
				}, nil)
				
				// Mock tier detector
				td.On("GetCurrentResources").Return(SystemResources{
					CPUCores:    2,
					MemoryGB:    4.0,
					CPUUsage:    95.0,
					MemoryUsage: 90.0,
					DiskUsage:   95.0,
					LastUpdated: time.Now(),
				})
				td.On("GetCurrentTier").Return(Tier("basic"))
			},
			expectedStatus: http.StatusOK,
			expectedDevice: "unhealthy",
		},
		{
			name: "degraded device with no adapters",
			setupMocks: func(ar *MockAdapterRegistry, qm *MockQueueManager, td *MockTierDetector) {
				ar.On("GetAllAdapters").Return([]adapters.HardwareAdapter{})
				
				qm.On("GetQueueDepth", mock.Anything).Return(0, nil)
				qm.On("GetStats", mock.Anything).Return(QueueStats{}, nil)
				
				td.On("GetCurrentResources").Return(SystemResources{
					CPUCores:    4,
					MemoryGB:    8.0,
					CPUUsage:    20.0,
					MemoryUsage: 30.0,
					DiskUsage:   40.0,
					LastUpdated: time.Now(),
				})
				td.On("GetCurrentTier").Return(Tier("standard"))
			},
			expectedStatus: http.StatusOK,
			expectedDevice: "degraded",
		},
		{
			name: "unhealthy device with queue manager error",
			setupMocks: func(ar *MockAdapterRegistry, qm *MockQueueManager, td *MockTierDetector) {
				ar.On("GetAllAdapters").Return([]adapters.HardwareAdapter{})
				
				// Queue manager returns error
				qm.On("GetQueueDepth", mock.Anything).Return(-1, assert.AnError)
				qm.On("GetStats", mock.Anything).Return(QueueStats{}, assert.AnError)
				
				td.On("GetCurrentResources").Return(SystemResources{
					CPUCores:    4,
					MemoryGB:    8.0,
					CPUUsage:    50.0,
					MemoryUsage: 60.0,
					DiskUsage:   70.0,
					LastUpdated: time.Now(),
				})
				td.On("GetCurrentTier").Return(Tier("standard"))
			},
			expectedStatus: http.StatusOK,
			expectedDevice: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, mockAdapterRegistry, _, _, mockQueueManager, mockTierDetector := setupStatusTestHandlers()
			tt.setupMocks(mockAdapterRegistry, mockQueueManager, mockTierDetector)

			// Create HTTP request
			req := httptest.NewRequest("GET", "/api/v1/status", nil)
			w := httptest.NewRecorder()

			// Call handler
			handlers.DeviceStatus(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response
			var response DeviceStatusResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verify response fields
			assert.Equal(t, tt.expectedDevice, response.Status)
			assert.Equal(t, "test-device-id", response.DeviceID)
			assert.Equal(t, "test-version", response.Version)
			assert.NotZero(t, response.Timestamp)
			assert.True(t, response.Uptime > 0) // Use True instead of NotZero for duration

			// Verify mocks
			mockAdapterRegistry.AssertExpectations(t)
			mockQueueManager.AssertExpectations(t)
			mockTierDetector.AssertExpectations(t)
		})
	}
}

func TestHandlers_DeviceStatus_WithNilDependencies(t *testing.T) {
	handlers, mockAdapterRegistry, _ := setupTestHandlers()
	// Don't set additional dependencies (nil), but mock the adapter registry
	// Set start time to 1 hour ago to ensure uptime > 0
	handlers.startTime = time.Now().Add(-time.Hour)
	mockAdapterRegistry.On("GetAllAdapters").Return([]adapters.HardwareAdapter{})

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	handlers.DeviceStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response DeviceStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Should handle nil dependencies gracefully
	assert.Equal(t, "degraded", response.Status) // No adapters = degraded
	assert.Equal(t, "test-device-id", response.DeviceID)
	assert.Equal(t, "test-version", response.Version)
	assert.NotZero(t, response.Timestamp)
	assert.True(t, response.Uptime > 0) // Use True instead of NotZero for duration
	assert.Equal(t, 0, response.QueueDepth)
	assert.Empty(t, response.AdapterStatus)
	assert.Empty(t, response.Tier)
	
	mockAdapterRegistry.AssertExpectations(t)
}

func TestHandlers_DeviceMetrics(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockAdapterRegistry, *MockDoorController, *MockQueueManager, *MockTierDetector)
	}{
		{
			name: "comprehensive metrics",
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController, qm *MockQueueManager, td *MockTierDetector) {
				// Mock adapters
				mockAdapter1 := &MockHardwareAdapter{}
				mockAdapter1.On("Name").Return("adapter-1")
				mockAdapter1.On("GetStatus").Return(types.AdapterStatus{
					Name:      "adapter-1",
					Status:    "active",
					LastEvent: time.Now().Add(-time.Minute),
					UpdatedAt: time.Now(),
				})
				
				mockAdapter2 := &MockHardwareAdapter{}
				mockAdapter2.On("Name").Return("adapter-2")
				mockAdapter2.On("GetStatus").Return(types.AdapterStatus{
					Name:         "adapter-2",
					Status:       "error",
					ErrorMessage: "Connection timeout",
					UpdatedAt:    time.Now(),
				})
				
				ar.On("GetAllAdapters").Return([]adapters.HardwareAdapter{mockAdapter1, mockAdapter2})
				
				// Mock door controller stats
				dc.On("GetStats").Return(map[string]interface{}{
					"totalRequests":       int64(1000),
					"totalErrors":         int64(50),
					"averageResponseTime": 150.5,
					"adapter-1": map[string]interface{}{
						"eventCount":     int64(500),
						"errorCount":     int64(10),
						"responseTimeMs": 120.0,
					},
					"adapter-2": map[string]interface{}{
						"eventCount":     int64(300),
						"errorCount":     int64(40),
						"responseTimeMs": 200.0,
					},
				})
				
				// Mock queue manager
				qm.On("GetStats", mock.Anything).Return(QueueStats{
					QueueDepth:      15,
					PendingEvents:   10,
					SentEvents:      800,
					FailedEvents:    25,
					LastSentAt:      time.Now().Add(-30 * time.Second),
					LastFailureAt:   time.Now().Add(-5 * time.Minute),
					OldestEventTime: time.Now().Add(-15 * time.Minute),
				}, nil)
				
				// Mock tier detector
				td.On("GetCurrentResources").Return(SystemResources{
					CPUCores:    8,
					MemoryGB:    16.0,
					CPUUsage:    45.0,
					MemoryUsage: 65.0,
					DiskUsage:   55.0,
					LastUpdated: time.Now(),
				})
			},
		},
		{
			name: "minimal metrics with errors",
			setupMocks: func(ar *MockAdapterRegistry, dc *MockDoorController, qm *MockQueueManager, td *MockTierDetector) {
				ar.On("GetAllAdapters").Return([]adapters.HardwareAdapter{})
				
				dc.On("GetStats").Return(map[string]interface{}{
					"totalRequests": int64(0),
					"totalErrors":   int64(0),
				})
				
				qm.On("GetStats", mock.Anything).Return(QueueStats{}, assert.AnError)
				
				td.On("GetCurrentResources").Return(SystemResources{
					CPUCores:    2,
					MemoryGB:    4.0,
					CPUUsage:    80.0,
					MemoryUsage: 85.0,
					DiskUsage:   90.0,
					LastUpdated: time.Now(),
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, mockAdapterRegistry, mockDoorController, _, mockQueueManager, mockTierDetector := setupStatusTestHandlers()
			tt.setupMocks(mockAdapterRegistry, mockDoorController, mockQueueManager, mockTierDetector)

			// Create HTTP request
			req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
			w := httptest.NewRecorder()

			// Call handler
			handlers.DeviceMetrics(w, req)

			// Check status code
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse response
			var response DeviceMetricsResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verify response structure
			assert.NotZero(t, response.Timestamp)
			assert.NotZero(t, response.Uptime)
			assert.NotNil(t, response.QueueMetrics)
			assert.NotNil(t, response.AdapterMetrics)
			assert.NotNil(t, response.SystemMetrics)
			assert.NotNil(t, response.PerformanceStats)

			// Verify mocks
			mockAdapterRegistry.AssertExpectations(t)
			mockDoorController.AssertExpectations(t)
			mockQueueManager.AssertExpectations(t)
			mockTierDetector.AssertExpectations(t)
		})
	}
}

func TestHandlers_DeviceMetrics_WithNilDependencies(t *testing.T) {
	handlers, mockAdapterRegistry, mockDoorController := setupTestHandlers()
	// Don't set additional dependencies (nil), but mock the required ones
	// Set start time to 1 hour ago to ensure uptime > 0
	handlers.startTime = time.Now().Add(-time.Hour)
	mockAdapterRegistry.On("GetAllAdapters").Return([]adapters.HardwareAdapter{})
	mockDoorController.On("GetStats").Return(map[string]interface{}{})

	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	w := httptest.NewRecorder()

	handlers.DeviceMetrics(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response DeviceMetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Should handle nil dependencies gracefully
	assert.NotZero(t, response.Timestamp)
	assert.True(t, response.Uptime > 0) // Use True instead of NotZero for duration
	assert.Empty(t, response.AdapterMetrics)
	assert.Zero(t, response.QueueMetrics.QueueDepth)
	assert.Zero(t, response.SystemMetrics.CPUUsage)
	assert.Zero(t, response.PerformanceStats.RequestsPerSecond)
	
	mockAdapterRegistry.AssertExpectations(t)
	mockDoorController.AssertExpectations(t)
}

func TestStatusEndpointsDataAccuracy(t *testing.T) {
	t.Run("health check data accuracy", func(t *testing.T) {
		handlers, _, _, mockHealthMonitor, _, _ := setupStatusTestHandlers()
		
		expectedTime := time.Now().UTC()
		expectedUptime := 2 * time.Hour
		
		mockHealthMonitor.On("UpdateHealth", mock.Anything).Return(nil)
		mockHealthMonitor.On("GetCurrentHealth").Return(SystemHealth{
			Status:    "healthy",
			Timestamp: expectedTime,
			Version:   "v1.2.3",
			DeviceID:  "device-123",
			Uptime:    expectedUptime,
			QueueDepth: 42,
			AdapterStatus: []AdapterStatus{
				{
					Name:      "test-adapter",
					Status:    "active",
					LastEvent: expectedTime.Add(-5 * time.Minute),
					UpdatedAt: expectedTime,
				},
			},
			Resources: SystemResources{
				CPUCores:    8,
				MemoryGB:    16.0,
				CPUUsage:    35.7,
				MemoryUsage: 62.3,
				DiskUsage:   78.9,
				LastUpdated: expectedTime,
			},
			Tier: "premium",
			LastEventTime: &expectedTime,
		})

		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		handlers.HealthCheck(w, req)

		var response HealthCheckResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Verify exact data accuracy
		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, expectedTime, response.Timestamp)
		assert.Equal(t, "v1.2.3", response.Version)
		assert.Equal(t, "device-123", response.DeviceID)
		assert.Equal(t, expectedUptime, response.Uptime)
		assert.Equal(t, 42, response.QueueDepth)
		assert.Equal(t, "premium", response.Tier)
		assert.Equal(t, expectedTime, *response.LastEventTime)
		
		// Verify adapter status accuracy
		assert.Len(t, response.AdapterStatus, 1)
		assert.Equal(t, "test-adapter", response.AdapterStatus[0].Name)
		assert.Equal(t, "active", response.AdapterStatus[0].Status)
		assert.Equal(t, expectedTime.Add(-5*time.Minute), response.AdapterStatus[0].LastEvent)
		
		// Verify resource accuracy
		assert.Equal(t, 8, response.Resources.CPUCores)
		assert.Equal(t, 16.0, response.Resources.MemoryGB)
		assert.Equal(t, 35.7, response.Resources.CPUUsage)
		assert.Equal(t, 62.3, response.Resources.MemoryUsage)
		assert.Equal(t, 78.9, response.Resources.DiskUsage)

		mockHealthMonitor.AssertExpectations(t)
	})

	t.Run("device status data accuracy", func(t *testing.T) {
		handlers, mockAdapterRegistry, _, _, mockQueueManager, mockTierDetector := setupStatusTestHandlers()
		
		expectedTime := time.Now().UTC()
		
		// Mock adapter with specific data
		mockAdapter := &MockHardwareAdapter{}
		mockAdapter.On("GetStatus").Return(types.AdapterStatus{
			Name:         "precise-adapter",
			Status:       "active",
			LastEvent:    expectedTime.Add(-3 * time.Minute),
			ErrorMessage: "",
			UpdatedAt:    expectedTime,
		})
		
		mockAdapterRegistry.On("GetAllAdapters").Return([]adapters.HardwareAdapter{mockAdapter})
		
		mockQueueManager.On("GetQueueDepth", mock.Anything).Return(17, nil)
		mockQueueManager.On("GetStats", mock.Anything).Return(QueueStats{
			LastSentAt: expectedTime.Add(-2 * time.Minute),
		}, nil)
		
		mockTierDetector.On("GetCurrentResources").Return(SystemResources{
			CPUCores:    4,
			MemoryGB:    8.0,
			CPUUsage:    42.5,
			MemoryUsage: 67.8,
			DiskUsage:   55.2,
			LastUpdated: expectedTime,
		})
		mockTierDetector.On("GetCurrentTier").Return(Tier("standard"))

		req := httptest.NewRequest("GET", "/api/v1/status", nil)
		w := httptest.NewRecorder()

		handlers.DeviceStatus(w, req)

		var response DeviceStatusResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Verify exact data accuracy
		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, "test-device-id", response.DeviceID)
		assert.Equal(t, "test-version", response.Version)
		assert.Equal(t, 17, response.QueueDepth)
		assert.Equal(t, "standard", response.Tier)
		assert.Equal(t, expectedTime.Add(-2*time.Minute), *response.LastEventTime)
		
		// Verify adapter status accuracy
		assert.Len(t, response.AdapterStatus, 1)
		assert.Equal(t, "precise-adapter", response.AdapterStatus[0].Name)
		assert.Equal(t, "active", response.AdapterStatus[0].Status)
		assert.Equal(t, expectedTime.Add(-3*time.Minute), response.AdapterStatus[0].LastEvent)
		assert.Empty(t, response.AdapterStatus[0].ErrorMessage)
		assert.Equal(t, expectedTime, response.AdapterStatus[0].UpdatedAt)
		
		// Verify resource accuracy
		assert.Equal(t, 4, response.Resources.CPUCores)
		assert.Equal(t, 8.0, response.Resources.MemoryGB)
		assert.Equal(t, 42.5, response.Resources.CPUUsage)
		assert.Equal(t, 67.8, response.Resources.MemoryUsage)
		assert.Equal(t, 55.2, response.Resources.DiskUsage)
		assert.Equal(t, expectedTime, response.Resources.LastUpdated)

		mockAdapterRegistry.AssertExpectations(t)
		mockQueueManager.AssertExpectations(t)
		mockTierDetector.AssertExpectations(t)
	})
}