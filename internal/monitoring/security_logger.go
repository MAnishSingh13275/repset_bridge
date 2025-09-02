package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// MonitoringSystemInterface defines the interface for logging security events
type MonitoringSystemInterface interface {
	LogSecurityEvent(ctx context.Context, eventType, description string, metadata map[string]interface{}) error
}

// SecurityLogger handles logging of security events and HMAC failures
type SecurityLogger struct {
	logger            *logrus.Logger
	monitoringSystem  MonitoringSystemInterface
	failureThreshold  int           // Number of failures before generating alert
	timeWindow        time.Duration // Time window for counting failures
	recentFailures    []time.Time   // Track recent failures
}

// SecurityLoggerConfig holds configuration for the security logger
type SecurityLoggerConfig struct {
	FailureThreshold int           `json:"failureThreshold"` // Number of failures before alert
	TimeWindow       time.Duration `json:"timeWindow"`       // Time window for counting failures
}

// DefaultSecurityLoggerConfig returns default configuration
func DefaultSecurityLoggerConfig() SecurityLoggerConfig {
	return SecurityLoggerConfig{
		FailureThreshold: 5,
		TimeWindow:       5 * time.Minute,
	}
}

// NewSecurityLogger creates a new security logger
func NewSecurityLogger(logger *logrus.Logger, monitoringSystem MonitoringSystemInterface, config SecurityLoggerConfig) *SecurityLogger {
	return &SecurityLogger{
		logger:           logger,
		monitoringSystem: monitoringSystem,
		failureThreshold: config.FailureThreshold,
		timeWindow:       config.TimeWindow,
		recentFailures:   make([]time.Time, 0),
	}
}

// LogHMACValidationFailure logs HMAC validation failures and generates security events
func (s *SecurityLogger) LogHMACValidationFailure(ctx context.Context, deviceID, sourceIP, userAgent string, details map[string]interface{}) error {
	now := time.Now()
	
	// Add to recent failures
	s.recentFailures = append(s.recentFailures, now)
	
	// Clean up old failures outside the time window
	s.cleanupOldFailures(now)
	
	// Log the failure
	s.logger.WithFields(logrus.Fields{
		"event_type":    "hmac_validation_failure",
		"device_id":     deviceID,
		"source_ip":     sourceIP,
		"user_agent":    userAgent,
		"failure_count": len(s.recentFailures),
		"details":       details,
	}).Warn("HMAC validation failure detected")
	
	// Prepare metadata
	metadata := map[string]interface{}{
		"source_ip":     sourceIP,
		"user_agent":    userAgent,
		"failure_count": len(s.recentFailures),
		"time_window":   s.timeWindow.String(),
	}
	
	// Add additional details if provided
	for k, v := range details {
		metadata[k] = v
	}
	
	// Generate security event
	description := fmt.Sprintf("HMAC validation failed for device %s from IP %s", deviceID, sourceIP)
	if len(s.recentFailures) > 1 {
		description = fmt.Sprintf("HMAC validation failed for device %s from IP %s (%d failures in %v)", 
			deviceID, sourceIP, len(s.recentFailures), s.timeWindow)
	}
	
	if err := s.monitoringSystem.LogSecurityEvent(ctx, "hmac_validation_failure", description, metadata); err != nil {
		return fmt.Errorf("failed to log security event: %w", err)
	}
	
	// Generate alert if threshold exceeded
	if len(s.recentFailures) >= s.failureThreshold {
		return s.generateSecurityAlert(ctx, deviceID, sourceIP, len(s.recentFailures))
	}
	
	return nil
}

// LogAuthenticationFailure logs general authentication failures
func (s *SecurityLogger) LogAuthenticationFailure(ctx context.Context, deviceID, sourceIP, userAgent, reason string) error {
	s.logger.WithFields(logrus.Fields{
		"event_type": "authentication_failure",
		"device_id":  deviceID,
		"source_ip":  sourceIP,
		"user_agent": userAgent,
		"reason":     reason,
	}).Warn("Authentication failure detected")
	
	metadata := map[string]interface{}{
		"source_ip":  sourceIP,
		"user_agent": userAgent,
		"reason":     reason,
	}
	
	description := fmt.Sprintf("Authentication failed for device %s from IP %s: %s", deviceID, sourceIP, reason)
	
	return s.monitoringSystem.LogSecurityEvent(ctx, "authentication_failure", description, metadata)
}

// LogSuspiciousActivity logs suspicious activity patterns
func (s *SecurityLogger) LogSuspiciousActivity(ctx context.Context, deviceID, sourceIP, activityType string, details map[string]interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"event_type":    "suspicious_activity",
		"device_id":     deviceID,
		"source_ip":     sourceIP,
		"activity_type": activityType,
		"details":       details,
	}).Warn("Suspicious activity detected")
	
	metadata := map[string]interface{}{
		"source_ip":     sourceIP,
		"activity_type": activityType,
	}
	
	// Add additional details
	for k, v := range details {
		metadata[k] = v
	}
	
	description := fmt.Sprintf("Suspicious activity detected for device %s from IP %s: %s", deviceID, sourceIP, activityType)
	
	return s.monitoringSystem.LogSecurityEvent(ctx, "suspicious_activity", description, metadata)
}

// LogRateLimitExceeded logs rate limiting events
func (s *SecurityLogger) LogRateLimitExceeded(ctx context.Context, deviceID, sourceIP, endpoint string, requestCount int) error {
	s.logger.WithFields(logrus.Fields{
		"event_type":    "rate_limit_exceeded",
		"device_id":     deviceID,
		"source_ip":     sourceIP,
		"endpoint":      endpoint,
		"request_count": requestCount,
	}).Warn("Rate limit exceeded")
	
	metadata := map[string]interface{}{
		"source_ip":     sourceIP,
		"endpoint":      endpoint,
		"request_count": requestCount,
	}
	
	description := fmt.Sprintf("Rate limit exceeded for device %s from IP %s on endpoint %s (%d requests)", 
		deviceID, sourceIP, endpoint, requestCount)
	
	return s.monitoringSystem.LogSecurityEvent(ctx, "rate_limit_exceeded", description, metadata)
}

// LogInvalidRequest logs invalid or malformed requests
func (s *SecurityLogger) LogInvalidRequest(ctx context.Context, deviceID, sourceIP, userAgent, reason string, requestData interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"event_type":   "invalid_request",
		"device_id":    deviceID,
		"source_ip":    sourceIP,
		"user_agent":   userAgent,
		"reason":       reason,
		"request_data": requestData,
	}).Info("Invalid request detected")
	
	metadata := map[string]interface{}{
		"source_ip":    sourceIP,
		"user_agent":   userAgent,
		"reason":       reason,
		"request_data": requestData,
	}
	
	description := fmt.Sprintf("Invalid request from device %s at IP %s: %s", deviceID, sourceIP, reason)
	
	return s.monitoringSystem.LogSecurityEvent(ctx, "invalid_request", description, metadata)
}

// generateSecurityAlert generates a security alert when thresholds are exceeded
func (s *SecurityLogger) generateSecurityAlert(ctx context.Context, deviceID, sourceIP string, failureCount int) error {
	alert := Alert{
		ID:          fmt.Sprintf("security_alert_%s_%d", deviceID, time.Now().Unix()),
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityHigh,
		Title:       "Security Alert: Multiple Authentication Failures",
		Description: fmt.Sprintf("Device %s from IP %s has %d authentication failures in %v", 
			deviceID, sourceIP, failureCount, s.timeWindow),
		Timestamp:   time.Now(),
		DeviceID:    deviceID,
		Metadata: map[string]interface{}{
			"source_ip":      sourceIP,
			"failure_count":  failureCount,
			"time_window":    s.timeWindow.String(),
			"threshold":      s.failureThreshold,
		},
	}
	
	// This would normally be handled by the monitoring system's alert generation
	// but we're calling it directly here for immediate security response
	s.logger.WithFields(logrus.Fields{
		"alert_id":     alert.ID,
		"device_id":    deviceID,
		"source_ip":    sourceIP,
		"failure_count": failureCount,
	}).Error("Security alert generated due to multiple authentication failures")
	
	return nil
}

// cleanupOldFailures removes failures outside the time window
func (s *SecurityLogger) cleanupOldFailures(now time.Time) {
	cutoff := now.Add(-s.timeWindow)
	
	// Find the first failure within the time window
	validIndex := 0
	for i, failure := range s.recentFailures {
		if failure.After(cutoff) {
			validIndex = i
			break
		}
		validIndex = len(s.recentFailures) // All failures are old
	}
	
	// Keep only recent failures
	if validIndex > 0 {
		s.recentFailures = s.recentFailures[validIndex:]
	}
}

// GetRecentFailureCount returns the number of recent failures
func (s *SecurityLogger) GetRecentFailureCount() int {
	s.cleanupOldFailures(time.Now())
	return len(s.recentFailures)
}

// Reset clears all tracked failures (useful for testing)
func (s *SecurityLogger) Reset() {
	s.recentFailures = make([]time.Time, 0)
}

// HTTPSecurityMiddleware creates HTTP middleware for security logging
func (s *SecurityLogger) HTTPSecurityMiddleware(deviceID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Extract client information
			sourceIP := s.getClientIP(r)
			userAgent := r.UserAgent()
			
			// Create a response writer wrapper to capture status codes
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			
			// Call the next handler
			next.ServeHTTP(wrapper, r)
			
			// Log security events based on response status
			switch {
			case wrapper.statusCode == http.StatusUnauthorized:
				s.LogAuthenticationFailure(ctx, deviceID, sourceIP, userAgent, "unauthorized")
			case wrapper.statusCode == http.StatusForbidden:
				s.LogAuthenticationFailure(ctx, deviceID, sourceIP, userAgent, "forbidden")
			case wrapper.statusCode == http.StatusTooManyRequests:
				s.LogRateLimitExceeded(ctx, deviceID, sourceIP, r.URL.Path, 1)
			case wrapper.statusCode >= 400 && wrapper.statusCode < 500:
				s.LogInvalidRequest(ctx, deviceID, sourceIP, userAgent, 
					fmt.Sprintf("HTTP %d", wrapper.statusCode), nil)
			}
		})
	}
}

// getClientIP extracts the client IP address from the request
func (s *SecurityLogger) getClientIP(r *http.Request) string {
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

// responseWriterWrapper wraps http.ResponseWriter to capture status codes
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}