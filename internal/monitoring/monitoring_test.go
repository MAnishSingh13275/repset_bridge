package monitoring

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
	"gym-door-bridge/internal/types"
)

// Mock implementations for testing

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

func (m *MockQueueManager) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockTierDetector struct {
	mock.Mock
}

func (m *MockTierDetector) GetCurrentResources() tier.SystemResources {
	args := m.Called()
	return args.Get(0).(tier.SystemResources)
}

func (m *MockTierDetector) GetCurrentTier() tier.Tier {
	args := m.Called()
	return args.Get(0).(tier.Tier)
}

type MockAlertHandler struct {
	mock.Mock
	handledAlerts []Alert
}

func (m *MockAlertHandler) HandleAlert(ctx context.Context, alert Alert) error {
	m.handledAlerts = append(m.handledAlerts, alert)
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertHandler) GetHandledAlerts() []Alert {
	return m.handledAlerts
}

func (m *MockAlertHandler) Reset() {
	m.handledAlerts = nil
}

// Test functions

func TestNewMonitoringSystem(t *testing.T) {
	logger := logrus.New()
	config := DefaultMonitoringConfig()
	
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	
	system := NewMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		WithLogger(logger),
		WithDeviceID("test-device"),
	)
	
	assert.NotNil(t, system)
	assert.Equal(t, "test-device", system.deviceID)
	assert.Equal(t, config, system.config)
	assert.NotNil(t, system.activeAlerts)
}

func TestMonitoringSystem_LogSecurityEvent(t *testing.T) {
	logger := logrus.New()
	config := DefaultMonitoringConfig()
	
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	mockReporter := NewMockMetricsReporter(logger)
	
	system := NewMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		WithLogger(logger),
		WithDeviceID("test-device"),
		WithMetricsReporter(mockReporter),
	)
	
	ctx := context.Background()
	metadata := map[string]interface{}{
		"source_ip": "192.168.1.100",
		"attempts":  3,
	}
	
	err := system.LogSecurityEvent(ctx, "hmac_validation_failure", "Test security event", metadata)
	require.NoError(t, err)
	
	// Check that security event was stored
	events := system.GetRecentSecurityEvents(10)
	assert.Len(t, events, 1)
	assert.Equal(t, "hmac_validation_failure", events[0].Type)
	assert.Equal(t, "Test security event", events[0].Description)
	assert.Equal(t, "test-device", events[0].DeviceID)
	
	// Check that it was reported to cloud
	reportedEvents := mockReporter.GetReportedSecurityEvents()
	assert.Len(t, reportedEvents, 1)
	assert.Equal(t, "hmac_validation_failure", reportedEvents[0].Type)
}

func TestMonitoringSystem_AlertGeneration(t *testing.T) {
	logger := logrus.New()
	config := DefaultMonitoringConfig()
	
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	mockAlertHandler := &MockAlertHandler{}
	
	system := NewMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		WithLogger(logger),
		WithDeviceID("test-device"),
	)
	
	system.AddAlertHandler(mockAlertHandler)
	
	// Set up mock expectations
	mockAlertHandler.On("HandleAlert", mock.Anything, mock.AnythingOfType("Alert")).Return(nil)
	
	ctx := context.Background()
	alert := Alert{
		ID:          "test-alert",
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityHigh,
		Title:       "Test Alert",
		Description: "This is a test alert",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
	}
	
	err := system.generateAlert(ctx, alert)
	require.NoError(t, err)
	
	// Check that alert was stored
	activeAlerts := system.GetActiveAlerts()
	assert.Len(t, activeAlerts, 1)
	assert.Equal(t, "test-alert", activeAlerts[0].ID)
	
	// Check that alert handler was called
	handledAlerts := mockAlertHandler.GetHandledAlerts()
	assert.Len(t, handledAlerts, 1)
	assert.Equal(t, "test-alert", handledAlerts[0].ID)
	
	mockAlertHandler.AssertExpectations(t)
}

func TestMonitoringSystem_QueueThresholdAlert(t *testing.T) {
	logger := logrus.New()
	config := DefaultMonitoringConfig()
	config.QueueThresholdPercent = 75.0
	
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	mockAlertHandler := &MockAlertHandler{}
	
	system := NewMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		WithLogger(logger),
		WithDeviceID("test-device"),
	)
	
	system.AddAlertHandler(mockAlertHandler)
	
	// Set up mock expectations for high queue usage
	mockQueueManager.On("GetStats", mock.Anything).Return(queue.QueueStats{
		QueueDepth: 8000, // 80% of 10000
	}, nil)
	
	mockHealthMonitor.On("GetCurrentHealth").Return(SystemHealth{
		Tier: tier.TierNormal,
	})
	
	mockAlertHandler.On("HandleAlert", mock.Anything, mock.AnythingOfType("Alert")).Return(nil)
	
	ctx := context.Background()
	err := system.checkQueueThresholdCondition(ctx, time.Now())
	require.NoError(t, err)
	
	// Check that alert was generated
	handledAlerts := mockAlertHandler.GetHandledAlerts()
	assert.Len(t, handledAlerts, 1)
	assert.Equal(t, AlertTypeQueueThreshold, handledAlerts[0].Type)
	assert.Equal(t, AlertSeverityMedium, handledAlerts[0].Severity)
	
	mockQueueManager.AssertExpectations(t)
	mockHealthMonitor.AssertExpectations(t)
	mockAlertHandler.AssertExpectations(t)
}

func TestMonitoringSystem_PerformanceDegradationAlert(t *testing.T) {
	logger := logrus.New()
	config := DefaultMonitoringConfig()
	config.PerformanceDegradedThreshold = 80.0
	
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	mockAlertHandler := &MockAlertHandler{}
	
	system := NewMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		WithLogger(logger),
		WithDeviceID("test-device"),
	)
	
	system.AddAlertHandler(mockAlertHandler)
	
	// Set up mock expectations for high resource usage
	mockHealthMonitor.On("GetCurrentHealth").Return(SystemHealth{
		Resources: tier.SystemResources{
			CPUUsage:    85.0, // Above threshold
			MemoryUsage: 70.0,
			DiskUsage:   60.0,
		},
		Tier: tier.TierNormal,
	})
	
	mockAlertHandler.On("HandleAlert", mock.Anything, mock.AnythingOfType("Alert")).Return(nil)
	
	ctx := context.Background()
	err := system.checkPerformanceDegradationCondition(ctx, time.Now())
	require.NoError(t, err)
	
	// Check that alert was generated
	handledAlerts := mockAlertHandler.GetHandledAlerts()
	assert.Len(t, handledAlerts, 1)
	assert.Equal(t, AlertTypePerformanceDegradation, handledAlerts[0].Type)
	assert.Equal(t, AlertSeverityMedium, handledAlerts[0].Severity)
	
	mockHealthMonitor.AssertExpectations(t)
	mockAlertHandler.AssertExpectations(t)
}

func TestMonitoringSystem_MetricsCollection(t *testing.T) {
	logger := logrus.New()
	config := DefaultMonitoringConfig()
	
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	
	system := NewMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		WithLogger(logger),
		WithDeviceID("test-device"),
	)
	
	// Set up mock expectations
	mockHealthMonitor.On("GetCurrentHealth").Return(SystemHealth{
		QueueDepth: 100,
		Resources: tier.SystemResources{
			CPUUsage:    25.0,
			MemoryUsage: 45.0,
			DiskUsage:   60.0,
		},
		Tier: tier.TierNormal,
	})
	
	mockQueueManager.On("GetStats", mock.Anything).Return(queue.QueueStats{
		QueueDepth: 100,
		LastSentAt: time.Now().Add(-5 * time.Minute),
	}, nil)
	
	ctx := context.Background()
	err := system.collectMetrics(ctx)
	require.NoError(t, err)
	
	// Check that metrics were collected
	currentMetrics := system.GetCurrentMetrics()
	assert.NotNil(t, currentMetrics)
	assert.Equal(t, "test-device", currentMetrics.DeviceID)
	assert.Equal(t, 100, currentMetrics.QueueDepth)
	assert.Equal(t, 25.0, currentMetrics.CPUUsage)
	assert.Equal(t, tier.TierNormal, currentMetrics.Tier)
	
	mockHealthMonitor.AssertExpectations(t)
	mockQueueManager.AssertExpectations(t)
}

func TestDefaultMonitoringConfig(t *testing.T) {
	config := DefaultMonitoringConfig()
	
	assert.Equal(t, 75.0, config.QueueThresholdPercent)
	assert.Equal(t, 5*time.Minute, config.OfflineThreshold)
	assert.Equal(t, 80.0, config.PerformanceDegradedThreshold)
	assert.Equal(t, 30*time.Second, config.MetricsCollectionInterval)
	assert.Equal(t, 1*time.Minute, config.AlertCheckInterval)
	assert.Equal(t, 7, config.MetricsRetentionDays)
	assert.Equal(t, 30, config.AlertRetentionDays)
	assert.True(t, config.EnableCloudReporting)
	assert.Equal(t, 5*time.Minute, config.ReportingInterval)
}

func TestAlert_String(t *testing.T) {
	alert := Alert{
		ID:          "test-alert",
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityHigh,
		Title:       "Test Alert",
		Description: "This is a test alert",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
	}
	
	// Test that alert fields are accessible
	assert.Equal(t, "test-alert", alert.ID)
	assert.Equal(t, AlertTypeSecurityEvent, alert.Type)
	assert.Equal(t, AlertSeverityHigh, alert.Severity)
	assert.Equal(t, "Test Alert", alert.Title)
	assert.Equal(t, "This is a test alert", alert.Description)
	assert.Equal(t, "test-device", alert.DeviceID)
	assert.False(t, alert.Resolved)
	assert.Nil(t, alert.ResolvedAt)
}

func TestSecurityEvent_Fields(t *testing.T) {
	event := SecurityEvent{
		ID:          "test-event",
		Type:        "hmac_validation_failure",
		Severity:    AlertSeverityHigh,
		Description: "Test security event",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
		SourceIP:    "192.168.1.100",
		UserAgent:   "test-agent",
		Metadata: map[string]interface{}{
			"attempts": 3,
		},
	}
	
	// Test that security event fields are accessible
	assert.Equal(t, "test-event", event.ID)
	assert.Equal(t, "hmac_validation_failure", event.Type)
	assert.Equal(t, AlertSeverityHigh, event.Severity)
	assert.Equal(t, "Test security event", event.Description)
	assert.Equal(t, "test-device", event.DeviceID)
	assert.Equal(t, "192.168.1.100", event.SourceIP)
	assert.Equal(t, "test-agent", event.UserAgent)
	assert.Equal(t, 3, event.Metadata["attempts"])
}

func TestPerformanceMetrics_Fields(t *testing.T) {
	now := time.Now()
	metrics := PerformanceMetrics{
		Timestamp:       now,
		DeviceID:        "test-device",
		QueueDepth:      100,
		QueueCapacity:   1000,
		CPUUsage:        25.0,
		MemoryUsage:     45.0,
		DiskUsage:       60.0,
		NetworkLatency:  50 * time.Millisecond,
		AdapterStatuses: map[string]string{"simulator": "active"},
		ErrorRate:       0.1,
		Tier:            tier.TierNormal,
	}
	
	// Test that performance metrics fields are accessible
	assert.Equal(t, now, metrics.Timestamp)
	assert.Equal(t, "test-device", metrics.DeviceID)
	assert.Equal(t, 100, metrics.QueueDepth)
	assert.Equal(t, 1000, metrics.QueueCapacity)
	assert.Equal(t, 25.0, metrics.CPUUsage)
	assert.Equal(t, 45.0, metrics.MemoryUsage)
	assert.Equal(t, 60.0, metrics.DiskUsage)
	assert.Equal(t, 50*time.Millisecond, metrics.NetworkLatency)
	assert.Equal(t, "active", metrics.AdapterStatuses["simulator"])
	assert.Equal(t, 0.1, metrics.ErrorRate)
	assert.Equal(t, tier.TierNormal, metrics.Tier)
}