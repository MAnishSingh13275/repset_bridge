package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/queue"
)

// IntegrationExample demonstrates how to integrate the monitoring system
// with the existing gym-door-bridge components
type IntegrationExample struct {
	logger            *logrus.Logger
	monitoringSystem  *MonitoringSystem
	securityLogger    *SecurityLogger
	authenticator     *auth.HMACAuthenticator
}

// NewIntegrationExample creates a new integration example
func NewIntegrationExample(
	logger *logrus.Logger,
	healthMonitor HealthMonitor,
	queueManager queue.QueueManager,
	tierDetector TierDetector,
	authenticator *auth.HMACAuthenticator,
) (*IntegrationExample, error) {
	
	// Create monitoring factory
	factory := NewMonitoringFactory(logger)
	
	// Create monitoring configuration
	config := DefaultMonitoringConfig()
	config.QueueThresholdPercent = 80.0
	config.OfflineThreshold = 3 * time.Minute
	config.EnableCloudReporting = true
	
	// Validate configuration
	if err := factory.ValidateConfiguration(config); err != nil {
		return nil, fmt.Errorf("invalid monitoring configuration: %w", err)
	}
	
	// Create monitoring system
	monitoringSystem, err := factory.CreateMonitoringSystem(
		config,
		healthMonitor,
		queueManager,
		tierDetector,
		authenticator.GetDeviceID(),
		authenticator,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring system: %w", err)
	}
	
	// Create security logger
	securityConfig := DefaultSecurityLoggerConfig()
	securityConfig.FailureThreshold = 5
	securityConfig.TimeWindow = 5 * time.Minute
	
	securityLogger := factory.CreateSecurityLogger(monitoringSystem, securityConfig)
	
	return &IntegrationExample{
		logger:           logger,
		monitoringSystem: monitoringSystem,
		securityLogger:   securityLogger,
		authenticator:    authenticator,
	}, nil
}

// Start starts the monitoring system
func (e *IntegrationExample) Start(ctx context.Context) error {
	e.logger.Info("Starting monitoring integration example")
	
	// Start the monitoring system
	if err := e.monitoringSystem.Start(ctx); err != nil {
		return fmt.Errorf("failed to start monitoring system: %w", err)
	}
	
	e.logger.Info("Monitoring system started successfully")
	return nil
}

// Stop stops the monitoring system
func (e *IntegrationExample) Stop(ctx context.Context) error {
	e.logger.Info("Stopping monitoring integration example")
	
	// Stop the monitoring system
	if err := e.monitoringSystem.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop monitoring system: %w", err)
	}
	
	e.logger.Info("Monitoring system stopped successfully")
	return nil
}

// CreateSecurityMiddleware creates HTTP middleware that logs security events
func (e *IntegrationExample) CreateSecurityMiddleware() func(http.Handler) http.Handler {
	return e.securityLogger.HTTPSecurityMiddleware(e.authenticator.GetDeviceID())
}

// HandleHMACValidationFailure demonstrates how to handle HMAC validation failures
func (e *IntegrationExample) HandleHMACValidationFailure(ctx context.Context, r *http.Request, err error) {
	sourceIP := e.getClientIP(r)
	userAgent := r.UserAgent()
	
	details := map[string]interface{}{
		"error":     err.Error(),
		"endpoint":  r.URL.Path,
		"method":    r.Method,
		"timestamp": time.Now().Unix(),
	}
	
	// Log the HMAC validation failure
	if logErr := e.securityLogger.LogHMACValidationFailure(
		ctx,
		e.authenticator.GetDeviceID(),
		sourceIP,
		userAgent,
		details,
	); logErr != nil {
		e.logger.WithError(logErr).Error("Failed to log HMAC validation failure")
	}
}

// HandleAuthenticationFailure demonstrates how to handle general authentication failures
func (e *IntegrationExample) HandleAuthenticationFailure(ctx context.Context, r *http.Request, reason string) {
	sourceIP := e.getClientIP(r)
	userAgent := r.UserAgent()
	
	if err := e.securityLogger.LogAuthenticationFailure(
		ctx,
		e.authenticator.GetDeviceID(),
		sourceIP,
		userAgent,
		reason,
	); err != nil {
		e.logger.WithError(err).Error("Failed to log authentication failure")
	}
}

// HandleSuspiciousActivity demonstrates how to log suspicious activity
func (e *IntegrationExample) HandleSuspiciousActivity(ctx context.Context, activityType string, details map[string]interface{}) {
	if err := e.securityLogger.LogSuspiciousActivity(
		ctx,
		e.authenticator.GetDeviceID(),
		"unknown", // Source IP might not be available in all contexts
		activityType,
		details,
	); err != nil {
		e.logger.WithError(err).Error("Failed to log suspicious activity")
	}
}

// GetCurrentStatus returns the current monitoring status
func (e *IntegrationExample) GetCurrentStatus() MonitoringStatus {
	currentMetrics := e.monitoringSystem.GetCurrentMetrics()
	activeAlerts := e.monitoringSystem.GetActiveAlerts()
	recentSecurityEvents := e.monitoringSystem.GetRecentSecurityEvents(10)
	
	return MonitoringStatus{
		CurrentMetrics:       currentMetrics,
		ActiveAlerts:         activeAlerts,
		RecentSecurityEvents: recentSecurityEvents,
		RecentFailureCount:   e.securityLogger.GetRecentFailureCount(),
	}
}

// MonitoringStatus represents the current monitoring status
type MonitoringStatus struct {
	CurrentMetrics       *PerformanceMetrics `json:"currentMetrics,omitempty"`
	ActiveAlerts         []Alert             `json:"activeAlerts"`
	RecentSecurityEvents []SecurityEvent     `json:"recentSecurityEvents"`
	RecentFailureCount   int                 `json:"recentFailureCount"`
}

// getClientIP extracts the client IP from the request
func (e *IntegrationExample) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// ExampleHTTPHandler demonstrates how to integrate monitoring with HTTP handlers
func (e *IntegrationExample) ExampleHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// Extract authentication headers
		deviceID := r.Header.Get("X-Device-ID")
		signature := r.Header.Get("X-Signature")
		timestampStr := r.Header.Get("X-Timestamp")
		
		// Validate device ID matches
		if deviceID != e.authenticator.GetDeviceID() {
			e.HandleAuthenticationFailure(ctx, r, "device ID mismatch")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		// Parse timestamp
		timestamp, err := parseTimestamp(timestampStr)
		if err != nil {
			e.HandleAuthenticationFailure(ctx, r, "invalid timestamp")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		
		// Read request body for HMAC validation
		body, err := readRequestBody(r)
		if err != nil {
			e.HandleAuthenticationFailure(ctx, r, "failed to read request body")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		
		// Validate HMAC signature
		if err := e.authenticator.ValidateSignature(body, timestamp, signature); err != nil {
			e.HandleHMACValidationFailure(ctx, r, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		// Process the request normally
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}
}

// Helper functions for the example

func parseTimestamp(timestampStr string) (int64, error) {
	// Implementation would parse the timestamp string
	// This is a placeholder
	return time.Now().Unix(), nil
}

func readRequestBody(r *http.Request) ([]byte, error) {
	// Implementation would read and return the request body
	// This is a placeholder
	return []byte{}, nil
}

// ExampleUsage demonstrates how to use the monitoring integration
func ExampleUsage() {
	// This function shows how the monitoring system would be used
	// in the main application
	
	logger := logrus.New()
	
	// Create authenticator (would come from configuration)
	authenticator := auth.NewHMACAuthenticator("device-123", "secret-key")
	
	// Create other dependencies (these would be real implementations)
	var healthMonitor HealthMonitor
	var queueManager queue.QueueManager
	var tierDetector TierDetector
	
	// Create integration example
	integration, err := NewIntegrationExample(
		logger,
		healthMonitor,
		queueManager,
		tierDetector,
		authenticator,
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create monitoring integration")
	}
	
	// Start monitoring
	ctx := context.Background()
	if err := integration.Start(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to start monitoring")
	}
	
	// Create HTTP server with security middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/example", integration.ExampleHTTPHandler())
	
	// Apply security middleware
	securityMiddleware := integration.CreateSecurityMiddleware()
	handler := securityMiddleware(mux)
	
	// Start HTTP server (this is just an example)
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}
	
	logger.Info("Starting HTTP server with monitoring integration")
	
	// In a real application, you would handle graceful shutdown
	if err := server.ListenAndServe(); err != nil {
		logger.WithError(err).Error("HTTP server failed")
	}
	
	// Stop monitoring
	if err := integration.Stop(ctx); err != nil {
		logger.WithError(err).Error("Failed to stop monitoring")
	}
}