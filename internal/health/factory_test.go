package health

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/tier"
)



func TestNewHealthSystem(t *testing.T) {
	// Setup
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierNormal,
		currentResources: tier.SystemResources{
			CPUUsage:    25.0,
			MemoryUsage: 50.0,
			DiskUsage:   30.0,
		},
	}
	mockHTTPClient := &MockHTTPClient{}
	logger := logrus.New()
	
	config := HealthSystemConfig{
		Health: HealthCheckConfig{
			Port: 8081,
			Path: "/health",
		},
		Heartbeat: HeartbeatConfig{
			Interval: 1 * time.Minute,
		},
		Metrics: MetricsConfig{
			Enabled: false,
		},
		Version:  "1.0.0",
		DeviceID: "test-device",
	}
	
	// Test creating health system
	healthSystem, err := NewHealthSystem(
		config,
		mockQueue,
		mockTierDetector,
		mockHTTPClient,
		logger,
	)
	
	require.NoError(t, err)
	assert.NotNil(t, healthSystem.Monitor)
	assert.NotNil(t, healthSystem.HeartbeatManager)
	assert.NotNil(t, healthSystem.AdapterRegistry)
	assert.NotNil(t, healthSystem.MetricsExporter)
	
	// Test that components are properly configured
	assert.Equal(t, 0, healthSystem.AdapterRegistry.Count())
}

func TestNewHealthSystem_WithMetrics(t *testing.T) {
	// Setup
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierFull,
	}
	mockHTTPClient := &MockHTTPClient{}
	logger := logrus.New()
	
	config := HealthSystemConfig{
		Health: DefaultHealthCheckConfig(),
		Heartbeat: GetTierHeartbeatConfig(tier.TierFull),
		Metrics: MetricsConfig{
			Enabled:   true,
			Port:      9093,
			Namespace: "test",
		},
		Version:  "1.0.0",
		DeviceID: "test-device",
	}
	
	// Test creating health system with metrics enabled
	healthSystem, err := NewHealthSystem(
		config,
		mockQueue,
		mockTierDetector,
		mockHTTPClient,
		logger,
	)
	
	require.NoError(t, err)
	
	// Verify that metrics exporter is the Prometheus implementation
	_, ok := healthSystem.MetricsExporter.(*PrometheusMetricsExporter)
	assert.True(t, ok, "Expected PrometheusMetricsExporter when metrics are enabled")
}

func TestHealthSystem_StartStop(t *testing.T) {
	// Setup
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierNormal,
		currentResources: tier.SystemResources{
			CPUUsage:    25.0,
			MemoryUsage: 50.0,
			DiskUsage:   30.0,
		},
	}
	mockHTTPClient := &MockHTTPClient{}
	logger := logrus.New()
	
	config := HealthSystemConfig{
		Health: HealthCheckConfig{
			Port: 8082,
			Path: "/health",
		},
		Heartbeat: HeartbeatConfig{
			Interval: 1 * time.Hour, // Long interval to prevent automatic heartbeats
		},
		Metrics: MetricsConfig{
			Enabled: false,
		},
		Version:  "1.0.0",
		DeviceID: "test-device",
	}
	
	healthSystem, err := NewHealthSystem(
		config,
		mockQueue,
		mockTierDetector,
		mockHTTPClient,
		logger,
	)
	require.NoError(t, err)
	
	// Mock expectations
	mockQueue.On("GetQueueDepth", mock.Anything).Return(5, nil).Maybe()
	mockQueue.On("GetStats", mock.Anything).Return(queue.QueueStats{
		QueueDepth: 5,
	}, nil).Maybe()
	mockHTTPClient.On("SendHeartbeat", mock.Anything, mock.Anything).Return(nil).Maybe()
	
	// Test start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = healthSystem.Start(ctx)
	require.NoError(t, err)
	
	// Give components time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test stop
	err = healthSystem.Stop(ctx)
	require.NoError(t, err)
	
	mockQueue.AssertExpectations(t)
}

func TestHealthSystem_UpdateTierConfiguration(t *testing.T) {
	// Setup
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierNormal,
	}
	mockHTTPClient := &MockHTTPClient{}
	logger := logrus.New()
	
	config := GetDefaultHealthSystemConfig(tier.TierNormal)
	
	healthSystem, err := NewHealthSystem(
		config,
		mockQueue,
		mockTierDetector,
		mockHTTPClient,
		logger,
	)
	require.NoError(t, err)
	
	// Test initial configuration
	initialStats := healthSystem.HeartbeatManager.GetStats()
	assert.Equal(t, 1*time.Minute, initialStats.Interval) // Normal tier interval
	
	// Update to Full tier
	healthSystem.UpdateTierConfiguration(tier.TierFull)
	
	// Test updated configuration
	updatedStats := healthSystem.HeartbeatManager.GetStats()
	assert.Equal(t, 30*time.Second, updatedStats.Interval) // Full tier interval
}

func TestGetDefaultHealthSystemConfig(t *testing.T) {
	tests := []struct {
		tier                    tier.Tier
		expectedHeartbeatInterval time.Duration
	}{
		{tier.TierLite, 5 * time.Minute},
		{tier.TierNormal, 1 * time.Minute},
		{tier.TierFull, 30 * time.Second},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			config := GetDefaultHealthSystemConfig(tt.tier)
			
			// Test health config
			assert.Equal(t, 8080, config.Health.Port)
			assert.Equal(t, "/health", config.Health.Path)
			assert.False(t, config.Health.EnableMetrics)
			
			// Test heartbeat config
			assert.Equal(t, tt.expectedHeartbeatInterval, config.Heartbeat.Interval)
			assert.Equal(t, 30*time.Second, config.Heartbeat.Timeout)
			assert.Equal(t, 3, config.Heartbeat.MaxRetries)
			
			// Test metrics config
			assert.False(t, config.Metrics.Enabled)
			assert.Equal(t, 9090, config.Metrics.Port)
			assert.Equal(t, "/metrics", config.Metrics.Path)
			assert.Equal(t, "gym_door_bridge", config.Metrics.Namespace)
			
			// Test default values
			assert.Equal(t, "unknown", config.Version)
			assert.Equal(t, "", config.DeviceID)
		})
	}
}

