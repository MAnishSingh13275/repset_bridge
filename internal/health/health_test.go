package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/tier"
	"gym-door-bridge/internal/types"
)



type MockHardwareAdapter struct {
	name   string
	status adapters.AdapterStatus
}

func (m *MockHardwareAdapter) Name() string {
	return m.name
}

func (m *MockHardwareAdapter) Initialize(ctx context.Context, config adapters.AdapterConfig) error {
	return nil
}

func (m *MockHardwareAdapter) StartListening(ctx context.Context) error {
	return nil
}

func (m *MockHardwareAdapter) StopListening(ctx context.Context) error {
	return nil
}

func (m *MockHardwareAdapter) UnlockDoor(ctx context.Context, durationMs int) error {
	return nil
}

func (m *MockHardwareAdapter) GetStatus() adapters.AdapterStatus {
	return m.status
}

func (m *MockHardwareAdapter) OnEvent(callback func(event types.RawHardwareEvent)) {
}

func (m *MockHardwareAdapter) IsHealthy() bool {
	return m.status.Status == adapters.StatusActive
}

func TestHealthMonitor_GetCurrentHealth(t *testing.T) {
	// Setup
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierNormal,
		currentResources: tier.SystemResources{
			CPUCores:    4,
			MemoryGB:    8.0,
			CPUUsage:    25.0,
			MemoryUsage: 50.0,
			DiskUsage:   30.0,
			LastUpdated: time.Now(),
		},
	}
	
	adapterRegistry := NewSimpleAdapterRegistry()
	adapter := &MockHardwareAdapter{
		name: "test-adapter",
		status: adapters.AdapterStatus{
			Name:      "test-adapter",
			Status:    adapters.StatusActive,
			UpdatedAt: time.Now(),
		},
	}
	adapterRegistry.RegisterAdapter(adapter)
	
	config := DefaultHealthCheckConfig()
	monitor := NewHealthMonitor(
		config,
		mockQueue,
		mockTierDetector,
		adapterRegistry,
		WithLogger(logrus.New()),
		WithVersion("1.0.0"),
		WithDeviceID("test-device"),
	)
	
	// Mock expectations
	mockQueue.On("GetQueueDepth", mock.Anything).Return(5, nil)
	mockQueue.On("GetStats", mock.Anything).Return(queue.QueueStats{
		QueueDepth:   5,
		LastSentAt:   time.Now().Add(-1 * time.Minute),
	}, nil)
	
	// Test
	ctx := context.Background()
	err := monitor.UpdateHealth(ctx)
	require.NoError(t, err)
	
	health := monitor.GetCurrentHealth()
	
	// Assertions
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Equal(t, 5, health.QueueDepth)
	assert.Equal(t, tier.TierNormal, health.Tier)
	assert.Equal(t, "1.0.0", health.Version)
	assert.Equal(t, "test-device", health.DeviceID)
	assert.Len(t, health.AdapterStatus, 1)
	assert.Equal(t, "test-adapter", health.AdapterStatus[0].Name)
	assert.Equal(t, adapters.StatusActive, health.AdapterStatus[0].Status)
	
	mockQueue.AssertExpectations(t)
}

func TestHealthMonitor_DetermineOverallHealth(t *testing.T) {
	tests := []struct {
		name            string
		queueDepth      int
		adapterStatuses []adapters.AdapterStatus
		resources       tier.SystemResources
		expectedStatus  HealthStatus
	}{
		{
			name:       "healthy system",
			queueDepth: 10,
			adapterStatuses: []adapters.AdapterStatus{
				{Name: "adapter1", Status: adapters.StatusActive},
			},
			resources: tier.SystemResources{
				CPUUsage:    25.0,
				MemoryUsage: 50.0,
				DiskUsage:   30.0,
			},
			expectedStatus: HealthStatusHealthy,
		},
		{
			name:       "degraded - high queue depth",
			queueDepth: 6000, // More than half of normal tier max (10000)
			adapterStatuses: []adapters.AdapterStatus{
				{Name: "adapter1", Status: adapters.StatusActive},
			},
			resources: tier.SystemResources{
				CPUUsage:    25.0,
				MemoryUsage: 50.0,
				DiskUsage:   30.0,
			},
			expectedStatus: HealthStatusDegraded,
		},
		{
			name:       "degraded - adapter error",
			queueDepth: 10,
			adapterStatuses: []adapters.AdapterStatus{
				{Name: "adapter1", Status: adapters.StatusActive},
				{Name: "adapter2", Status: adapters.StatusError},
			},
			resources: tier.SystemResources{
				CPUUsage:    25.0,
				MemoryUsage: 50.0,
				DiskUsage:   30.0,
			},
			expectedStatus: HealthStatusDegraded,
		},
		{
			name:       "degraded - high resource usage",
			queueDepth: 10,
			adapterStatuses: []adapters.AdapterStatus{
				{Name: "adapter1", Status: adapters.StatusActive},
			},
			resources: tier.SystemResources{
				CPUUsage:    85.0, // Above 80% threshold
				MemoryUsage: 50.0,
				DiskUsage:   30.0,
			},
			expectedStatus: HealthStatusDegraded,
		},
		{
			name:       "unhealthy - queue depth error",
			queueDepth: -1, // Error condition
			adapterStatuses: []adapters.AdapterStatus{
				{Name: "adapter1", Status: adapters.StatusActive},
			},
			resources: tier.SystemResources{
				CPUUsage:    25.0,
				MemoryUsage: 50.0,
				DiskUsage:   30.0,
			},
			expectedStatus: HealthStatusUnhealthy,
		},
		{
			name:       "unhealthy - all adapters error",
			queueDepth: 10,
			adapterStatuses: []adapters.AdapterStatus{
				{Name: "adapter1", Status: adapters.StatusError},
				{Name: "adapter2", Status: adapters.StatusError},
			},
			resources: tier.SystemResources{
				CPUUsage:    25.0,
				MemoryUsage: 50.0,
				DiskUsage:   30.0,
			},
			expectedStatus: HealthStatusUnhealthy,
		},
		{
			name:       "unhealthy - critical resource usage",
			queueDepth: 10,
			adapterStatuses: []adapters.AdapterStatus{
				{Name: "adapter1", Status: adapters.StatusActive},
			},
			resources: tier.SystemResources{
				CPUUsage:    96.0, // Above 95% threshold
				MemoryUsage: 50.0,
				DiskUsage:   30.0,
			},
			expectedStatus: HealthStatusUnhealthy,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueueManager{}
			mockTierDetector := &MockTierDetector{
				currentTier: tier.TierNormal,
			}
			
			monitor := NewHealthMonitor(
				DefaultHealthCheckConfig(),
				mockQueue,
				mockTierDetector,
				nil,
			)
			
			status := monitor.determineOverallHealth(tt.queueDepth, tt.adapterStatuses, tt.resources)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestHealthMonitor_HTTPEndpoint(t *testing.T) {
	// Setup
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierNormal,
		currentResources: tier.SystemResources{
			CPUCores:    4,
			MemoryGB:    8.0,
			CPUUsage:    25.0,
			MemoryUsage: 50.0,
			DiskUsage:   30.0,
			LastUpdated: time.Now(),
		},
	}
	
	config := DefaultHealthCheckConfig()
	monitor := NewHealthMonitor(
		config,
		mockQueue,
		mockTierDetector,
		NewSimpleAdapterRegistry(),
	)
	
	// Mock expectations
	mockQueue.On("GetQueueDepth", mock.Anything).Return(5, nil)
	mockQueue.On("GetStats", mock.Anything).Return(queue.QueueStats{
		QueueDepth: 5,
		LastSentAt: time.Now().Add(-1 * time.Minute),
	}, nil)
	
	// Test HTTP endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	monitor.handleHealthCheck(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	var health SystemHealth
	err := json.Unmarshal(w.Body.Bytes(), &health)
	require.NoError(t, err)
	
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Equal(t, 5, health.QueueDepth)
	assert.Equal(t, tier.TierNormal, health.Tier)
	
	mockQueue.AssertExpectations(t)
}

func TestHealthMonitor_HTTPEndpoint_Unhealthy(t *testing.T) {
	// Setup
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierNormal,
		currentResources: tier.SystemResources{
			CPUUsage:    96.0, // Critical level
			MemoryUsage: 50.0,
			DiskUsage:   30.0,
		},
	}
	
	monitor := NewHealthMonitor(
		DefaultHealthCheckConfig(),
		mockQueue,
		mockTierDetector,
		NewSimpleAdapterRegistry(),
	)
	
	// Mock expectations
	mockQueue.On("GetQueueDepth", mock.Anything).Return(5, nil)
	mockQueue.On("GetStats", mock.Anything).Return(queue.QueueStats{}, nil)
	
	// Test HTTP endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	monitor.handleHealthCheck(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var health SystemHealth
	err := json.Unmarshal(w.Body.Bytes(), &health)
	require.NoError(t, err)
	
	assert.Equal(t, HealthStatusUnhealthy, health.Status)
	
	mockQueue.AssertExpectations(t)
}

func TestSimpleAdapterRegistry(t *testing.T) {
	registry := NewSimpleAdapterRegistry()
	
	// Test empty registry
	assert.Equal(t, 0, registry.Count())
	assert.Empty(t, registry.GetAllAdapters())
	
	// Test registering adapter
	adapter := &MockHardwareAdapter{
		name: "test-adapter",
		status: adapters.AdapterStatus{
			Name:   "test-adapter",
			Status: adapters.StatusActive,
		},
	}
	
	registry.RegisterAdapter(adapter)
	assert.Equal(t, 1, registry.Count())
	
	// Test getting adapter
	retrieved, err := registry.GetAdapter("test-adapter")
	require.NoError(t, err)
	assert.Equal(t, adapter, retrieved)
	
	// Test getting non-existent adapter
	_, err = registry.GetAdapter("non-existent")
	assert.Error(t, err)
	
	// Test getting all adapters
	allAdapters := registry.GetAllAdapters()
	assert.Len(t, allAdapters, 1)
	assert.Equal(t, adapter, allAdapters[0])
	
	// Test getting adapter status
	status, err := registry.GetAdapterStatus("test-adapter")
	require.NoError(t, err)
	assert.Equal(t, "test-adapter", status.Name)
	assert.Equal(t, adapters.StatusActive, status.Status)
	
	// Test unregistering adapter
	registry.UnregisterAdapter("test-adapter")
	assert.Equal(t, 0, registry.Count())
	
	_, err = registry.GetAdapter("test-adapter")
	assert.Error(t, err)
}