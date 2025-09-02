package monitoring

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gym-door-bridge/internal/auth"
)

func TestNewMonitoringFactory(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	assert.NotNil(t, factory)
	assert.Equal(t, logger, factory.logger)
}

func TestMonitoringFactory_CreateMockComponents(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	monitoringSystem, securityLogger, metricsReporter := factory.CreateMockComponents()
	
	// Verify monitoring system
	assert.NotNil(t, monitoringSystem)
	assert.Equal(t, "test-device", monitoringSystem.deviceID)
	assert.False(t, monitoringSystem.config.EnableCloudReporting)
	assert.NotNil(t, monitoringSystem.activeAlerts)
	
	// Verify security logger
	assert.NotNil(t, securityLogger)
	assert.Equal(t, logger, securityLogger.logger)
	assert.Equal(t, monitoringSystem, securityLogger.monitoringSystem)
	
	// Verify metrics reporter (should be mock)
	assert.NotNil(t, metricsReporter)
	mockReporter, ok := metricsReporter.(*MockMetricsReporter)
	assert.True(t, ok)
	assert.NotNil(t, mockReporter)
}

func TestMonitoringFactory_CreateSecurityLogger(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	// Create a mock monitoring system
	monitoringSystem, _, _ := factory.CreateMockComponents()
	
	config := DefaultSecurityLoggerConfig()
	config.FailureThreshold = 10
	config.TimeWindow = 10 * time.Minute
	
	securityLogger := factory.CreateSecurityLogger(monitoringSystem, config)
	
	assert.NotNil(t, securityLogger)
	assert.Equal(t, logger, securityLogger.logger)
	assert.Equal(t, monitoringSystem, securityLogger.monitoringSystem)
	assert.Equal(t, 10, securityLogger.failureThreshold)
	assert.Equal(t, 10*time.Minute, securityLogger.timeWindow)
}

func TestMonitoringFactory_ValidateConfiguration_Valid(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	
	err := factory.ValidateConfiguration(config)
	require.NoError(t, err)
}

func TestMonitoringFactory_ValidateConfiguration_InvalidQueueThreshold(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	config.QueueThresholdPercent = -10.0 // Invalid
	
	err := factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue threshold percent must be between 0 and 100")
	
	config.QueueThresholdPercent = 150.0 // Invalid
	err = factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue threshold percent must be between 0 and 100")
}

func TestMonitoringFactory_ValidateConfiguration_InvalidOfflineThreshold(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	config.OfflineThreshold = -1 * time.Minute // Invalid
	
	err := factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "offline threshold must be positive")
}

func TestMonitoringFactory_ValidateConfiguration_InvalidPerformanceThreshold(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	config.PerformanceDegradedThreshold = -10.0 // Invalid
	
	err := factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "performance degraded threshold must be between 0 and 100")
	
	config.PerformanceDegradedThreshold = 150.0 // Invalid
	err = factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "performance degraded threshold must be between 0 and 100")
}

func TestMonitoringFactory_ValidateConfiguration_InvalidMetricsInterval(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	config.MetricsCollectionInterval = -1 * time.Second // Invalid
	
	err := factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics collection interval must be positive")
}

func TestMonitoringFactory_ValidateConfiguration_InvalidAlertInterval(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	config.AlertCheckInterval = -1 * time.Second // Invalid
	
	err := factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alert check interval must be positive")
}

func TestMonitoringFactory_ValidateConfiguration_InvalidRetentionDays(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	config.MetricsRetentionDays = -1 // Invalid
	
	err := factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics retention days must be positive")
	
	config.MetricsRetentionDays = 7 // Valid
	config.AlertRetentionDays = -1  // Invalid
	
	err = factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alert retention days must be positive")
}

func TestMonitoringFactory_ValidateConfiguration_InvalidReportingInterval(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	config.EnableCloudReporting = true
	config.ReportingInterval = -1 * time.Second // Invalid when cloud reporting is enabled
	
	err := factory.ValidateConfiguration(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reporting interval must be positive when cloud reporting is enabled")
	
	// Should be valid when cloud reporting is disabled
	config.EnableCloudReporting = false
	err = factory.ValidateConfiguration(config)
	require.NoError(t, err)
}

func TestMonitoringFactory_CreateAlertHandlers(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	config := DefaultMonitoringConfig()
	
	handlers, err := factory.createAlertHandlers(config)
	require.NoError(t, err)
	
	// Should have at least log and console handlers
	assert.GreaterOrEqual(t, len(handlers), 2)
	
	// Check that we have the expected handler types
	var hasLogHandler, hasConsoleHandler bool
	for _, handler := range handlers {
		switch handler.(type) {
		case *LogAlertHandler:
			hasLogHandler = true
		case *ConsoleAlertHandler:
			hasConsoleHandler = true
		}
	}
	
	assert.True(t, hasLogHandler, "Should have log alert handler")
	assert.True(t, hasConsoleHandler, "Should have console alert handler")
}

func TestMonitoringFactory_CreateMetricsReporter(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := factory.createMetricsReporter(authenticator)
	
	assert.NotNil(t, reporter)
	
	// Should be a CloudMetricsReporter
	cloudReporter, ok := reporter.(*CloudMetricsReporter)
	assert.True(t, ok)
	assert.NotNil(t, cloudReporter)
}

func TestMonitoringFactory_GetAlertFilePath(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	alertFilePath, err := factory.getAlertFilePath()
	
	// Should not error and should return a valid path
	require.NoError(t, err)
	assert.NotEmpty(t, alertFilePath)
	assert.Contains(t, alertFilePath, "alerts.jsonl")
}

// Integration test for creating a complete monitoring system
func TestMonitoringFactory_Integration(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	// Create mock dependencies
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	
	config := DefaultMonitoringConfig()
	config.EnableCloudReporting = false // Disable for testing
	
	// Validate configuration first
	err := factory.ValidateConfiguration(config)
	require.NoError(t, err)
	
	// Create monitoring system without authenticator (cloud reporting disabled)
	monitoringSystem, err := factory.CreateMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		"test-device",
		nil, // No authenticator needed when cloud reporting is disabled
	)
	require.NoError(t, err)
	
	assert.NotNil(t, monitoringSystem)
	assert.Equal(t, "test-device", monitoringSystem.deviceID)
	assert.Equal(t, config, monitoringSystem.config)
	
	// Create security logger
	securityConfig := DefaultSecurityLoggerConfig()
	securityLogger := factory.CreateSecurityLogger(monitoringSystem, securityConfig)
	
	assert.NotNil(t, securityLogger)
	assert.Equal(t, monitoringSystem, securityLogger.monitoringSystem)
}

func TestMonitoringFactory_CreateMonitoringSystemWithCloudReporting(t *testing.T) {
	logger := logrus.New()
	factory := NewMonitoringFactory(logger)
	
	// Create mock dependencies
	mockHealthMonitor := &MockHealthMonitor{}
	mockQueueManager := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{}
	
	config := DefaultMonitoringConfig()
	config.EnableCloudReporting = true // Enable cloud reporting
	
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	// Create monitoring system with authenticator
	monitoringSystem, err := factory.CreateMonitoringSystem(
		config,
		mockHealthMonitor,
		mockQueueManager,
		mockTierDetector,
		"test-device",
		authenticator,
	)
	require.NoError(t, err)
	
	assert.NotNil(t, monitoringSystem)
	assert.Equal(t, "test-device", monitoringSystem.deviceID)
	assert.Equal(t, config, monitoringSystem.config)
	assert.NotNil(t, monitoringSystem.metricsReporter)
}