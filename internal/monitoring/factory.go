package monitoring

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/queue"
)

// MonitoringFactory creates and configures monitoring components
type MonitoringFactory struct {
	logger *logrus.Logger
}

// NewMonitoringFactory creates a new monitoring factory
func NewMonitoringFactory(logger *logrus.Logger) *MonitoringFactory {
	return &MonitoringFactory{
		logger: logger,
	}
}

// CreateMonitoringSystem creates a fully configured monitoring system
func (f *MonitoringFactory) CreateMonitoringSystem(
	config MonitoringConfig,
	healthMonitor HealthMonitor,
	queueManager queue.QueueManager,
	tierDetector TierDetector,
	deviceID string,
	authenticator *auth.HMACAuthenticator,
) (*MonitoringSystem, error) {
	
	// Create monitoring system
	monitoringSystem := NewMonitoringSystem(
		config,
		healthMonitor,
		queueManager,
		tierDetector,
		WithLogger(f.logger),
		WithDeviceID(deviceID),
	)
	
	// Create and configure alert handlers
	alertHandlers, err := f.createAlertHandlers(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert handlers: %w", err)
	}
	
	// Add alert handlers to monitoring system
	for _, handler := range alertHandlers {
		monitoringSystem.AddAlertHandler(handler)
	}
	
	// Create metrics reporter if cloud reporting is enabled
	if config.EnableCloudReporting && authenticator != nil {
		metricsReporter := f.createMetricsReporter(authenticator)
		monitoringSystem = NewMonitoringSystem(
			config,
			healthMonitor,
			queueManager,
			tierDetector,
			WithLogger(f.logger),
			WithDeviceID(deviceID),
			WithMetricsReporter(metricsReporter),
		)
		
		// Re-add alert handlers
		for _, handler := range alertHandlers {
			monitoringSystem.AddAlertHandler(handler)
		}
	}
	
	return monitoringSystem, nil
}

// CreateSecurityLogger creates a security logger with the monitoring system
func (f *MonitoringFactory) CreateSecurityLogger(
	monitoringSystem MonitoringSystemInterface,
	config SecurityLoggerConfig,
) *SecurityLogger {
	return NewSecurityLogger(f.logger, monitoringSystem, config)
}

// createAlertHandlers creates the appropriate alert handlers based on configuration
func (f *MonitoringFactory) createAlertHandlers(config MonitoringConfig) ([]AlertHandler, error) {
	var handlers []AlertHandler
	
	// Always add log handler
	logHandler := NewLogAlertHandler(f.logger)
	handlers = append(handlers, logHandler)
	
	// Add console handler for critical alerts
	consoleHandler := NewConsoleAlertHandler(f.logger)
	handlers = append(handlers, consoleHandler)
	
	// Add file handler if we can determine a suitable location
	alertFilePath, err := f.getAlertFilePath()
	if err != nil {
		f.logger.WithError(err).Warn("Could not determine alert file path, skipping file handler")
	} else {
		fileHandler := NewFileAlertHandler(f.logger, alertFilePath)
		handlers = append(handlers, fileHandler)
	}
	
	return handlers, nil
}

// createMetricsReporter creates a metrics reporter for cloud reporting
func (f *MonitoringFactory) createMetricsReporter(authenticator *auth.HMACAuthenticator) MetricsReporter {
	config := DefaultCloudMetricsReporterConfig()
	
	// In a real implementation, you might read this from configuration
	// For now, we'll use the default configuration
	return NewCloudMetricsReporter(f.logger, config, authenticator)
}

// getAlertFilePath determines the appropriate path for the alert file
func (f *MonitoringFactory) getAlertFilePath() (string, error) {
	// Try to use a system-appropriate location
	var baseDir string
	
	// On Windows, use %PROGRAMDATA%
	if programData := os.Getenv("PROGRAMDATA"); programData != "" {
		baseDir = filepath.Join(programData, "GymDoorBridge")
	} else if home := os.Getenv("HOME"); home != "" {
		// On Unix-like systems, use ~/.local/share
		baseDir = filepath.Join(home, ".local", "share", "gym-door-bridge")
	} else {
		// Fallback to current directory
		baseDir = "."
	}
	
	// Create directory if it doesn't exist
	alertDir := filepath.Join(baseDir, "alerts")
	if err := os.MkdirAll(alertDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create alert directory: %w", err)
	}
	
	return filepath.Join(alertDir, "alerts.jsonl"), nil
}

// CreateMockComponents creates mock components for testing
func (f *MonitoringFactory) CreateMockComponents() (*MonitoringSystem, *SecurityLogger, MetricsReporter) {
	// Create mock metrics reporter
	mockReporter := NewMockMetricsReporter(f.logger)
	
	// Create minimal monitoring system for testing
	config := DefaultMonitoringConfig()
	config.EnableCloudReporting = false // Disable for testing
	
	// Note: In a real test, you'd provide mock implementations of these dependencies
	// For now, we'll create a minimal system that can be used for testing
	monitoringSystem := &MonitoringSystem{
		config:       config,
		logger:       f.logger,
		deviceID:     "test-device",
		activeAlerts: make(map[string]Alert),
	}
	
	// Create security logger
	securityConfig := DefaultSecurityLoggerConfig()
	securityLogger := NewSecurityLogger(f.logger, monitoringSystem, securityConfig)
	
	return monitoringSystem, securityLogger, mockReporter
}

// ValidateConfiguration validates the monitoring configuration
func (f *MonitoringFactory) ValidateConfiguration(config MonitoringConfig) error {
	if config.QueueThresholdPercent <= 0 || config.QueueThresholdPercent > 100 {
		return fmt.Errorf("queue threshold percent must be between 0 and 100, got %.2f", config.QueueThresholdPercent)
	}
	
	if config.OfflineThreshold <= 0 {
		return fmt.Errorf("offline threshold must be positive, got %v", config.OfflineThreshold)
	}
	
	if config.PerformanceDegradedThreshold <= 0 || config.PerformanceDegradedThreshold > 100 {
		return fmt.Errorf("performance degraded threshold must be between 0 and 100, got %.2f", config.PerformanceDegradedThreshold)
	}
	
	if config.MetricsCollectionInterval <= 0 {
		return fmt.Errorf("metrics collection interval must be positive, got %v", config.MetricsCollectionInterval)
	}
	
	if config.AlertCheckInterval <= 0 {
		return fmt.Errorf("alert check interval must be positive, got %v", config.AlertCheckInterval)
	}
	
	if config.MetricsRetentionDays <= 0 {
		return fmt.Errorf("metrics retention days must be positive, got %d", config.MetricsRetentionDays)
	}
	
	if config.AlertRetentionDays <= 0 {
		return fmt.Errorf("alert retention days must be positive, got %d", config.AlertRetentionDays)
	}
	
	if config.EnableCloudReporting && config.ReportingInterval <= 0 {
		return fmt.Errorf("reporting interval must be positive when cloud reporting is enabled, got %v", config.ReportingInterval)
	}
	
	return nil
}