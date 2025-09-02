package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock monitoring system for security logger tests
type MockMonitoringSystem struct {
	mock.Mock
	loggedEvents []SecurityEvent
}

func (m *MockMonitoringSystem) LogSecurityEvent(ctx context.Context, eventType, description string, metadata map[string]interface{}) error {
	args := m.Called(ctx, eventType, description, metadata)
	
	// Store the event for verification
	event := SecurityEvent{
		Type:        eventType,
		Description: description,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	m.loggedEvents = append(m.loggedEvents, event)
	
	return args.Error(0)
}

func (m *MockMonitoringSystem) GetLoggedEvents() []SecurityEvent {
	return m.loggedEvents
}

func (m *MockMonitoringSystem) Reset() {
	m.loggedEvents = nil
}

func TestNewSecurityLogger(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	assert.NotNil(t, securityLogger)
	assert.Equal(t, config.FailureThreshold, securityLogger.failureThreshold)
	assert.Equal(t, config.TimeWindow, securityLogger.timeWindow)
	assert.Empty(t, securityLogger.recentFailures)
}

func TestSecurityLogger_LogHMACValidationFailure(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Set up mock expectations
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "hmac_validation_failure", mock.Anything, mock.Anything).Return(nil)
	
	ctx := context.Background()
	details := map[string]interface{}{
		"signature": "invalid_signature",
		"timestamp": "1640995200",
	}
	
	err := securityLogger.LogHMACValidationFailure(ctx, "test-device", "192.168.1.100", "test-agent", details)
	require.NoError(t, err)
	
	// Verify that the failure was recorded
	assert.Equal(t, 1, securityLogger.GetRecentFailureCount())
	
	// Verify that the security event was logged
	loggedEvents := mockMonitoring.GetLoggedEvents()
	assert.Len(t, loggedEvents, 1)
	assert.Equal(t, "hmac_validation_failure", loggedEvents[0].Type)
	assert.Contains(t, loggedEvents[0].Description, "test-device")
	assert.Contains(t, loggedEvents[0].Description, "192.168.1.100")
	
	mockMonitoring.AssertExpectations(t)
}

func TestSecurityLogger_LogHMACValidationFailure_ThresholdExceeded(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	config.FailureThreshold = 3 // Lower threshold for testing
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Set up mock expectations - should be called multiple times
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "hmac_validation_failure", mock.Anything, mock.Anything).Return(nil)
	
	ctx := context.Background()
	details := map[string]interface{}{
		"signature": "invalid_signature",
	}
	
	// Log failures up to threshold
	for i := 0; i < config.FailureThreshold; i++ {
		err := securityLogger.LogHMACValidationFailure(ctx, "test-device", "192.168.1.100", "test-agent", details)
		require.NoError(t, err)
	}
	
	// Verify that all failures were recorded
	assert.Equal(t, config.FailureThreshold, securityLogger.GetRecentFailureCount())
	
	// Verify that security events were logged
	loggedEvents := mockMonitoring.GetLoggedEvents()
	assert.Len(t, loggedEvents, config.FailureThreshold)
	
	// The last event should mention multiple failures
	lastEvent := loggedEvents[len(loggedEvents)-1]
	assert.Contains(t, lastEvent.Description, "3 failures")
	
	mockMonitoring.AssertExpectations(t)
}

func TestSecurityLogger_LogAuthenticationFailure(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Set up mock expectations
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "authentication_failure", mock.Anything, mock.Anything).Return(nil)
	
	ctx := context.Background()
	err := securityLogger.LogAuthenticationFailure(ctx, "test-device", "192.168.1.100", "test-agent", "invalid credentials")
	require.NoError(t, err)
	
	// Verify that the security event was logged
	loggedEvents := mockMonitoring.GetLoggedEvents()
	assert.Len(t, loggedEvents, 1)
	assert.Equal(t, "authentication_failure", loggedEvents[0].Type)
	assert.Contains(t, loggedEvents[0].Description, "invalid credentials")
	assert.Equal(t, "192.168.1.100", loggedEvents[0].Metadata["source_ip"])
	assert.Equal(t, "invalid credentials", loggedEvents[0].Metadata["reason"])
	
	mockMonitoring.AssertExpectations(t)
}

func TestSecurityLogger_LogSuspiciousActivity(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Set up mock expectations
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "suspicious_activity", mock.Anything, mock.Anything).Return(nil)
	
	ctx := context.Background()
	details := map[string]interface{}{
		"pattern": "rapid_requests",
		"count":   50,
	}
	
	err := securityLogger.LogSuspiciousActivity(ctx, "test-device", "192.168.1.100", "unusual_pattern", details)
	require.NoError(t, err)
	
	// Verify that the security event was logged
	loggedEvents := mockMonitoring.GetLoggedEvents()
	assert.Len(t, loggedEvents, 1)
	assert.Equal(t, "suspicious_activity", loggedEvents[0].Type)
	assert.Contains(t, loggedEvents[0].Description, "unusual_pattern")
	assert.Equal(t, "unusual_pattern", loggedEvents[0].Metadata["activity_type"])
	assert.Equal(t, "rapid_requests", loggedEvents[0].Metadata["pattern"])
	assert.Equal(t, 50, loggedEvents[0].Metadata["count"])
	
	mockMonitoring.AssertExpectations(t)
}

func TestSecurityLogger_LogRateLimitExceeded(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Set up mock expectations
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "rate_limit_exceeded", mock.Anything, mock.Anything).Return(nil)
	
	ctx := context.Background()
	err := securityLogger.LogRateLimitExceeded(ctx, "test-device", "192.168.1.100", "/api/v1/checkin", 100)
	require.NoError(t, err)
	
	// Verify that the security event was logged
	loggedEvents := mockMonitoring.GetLoggedEvents()
	assert.Len(t, loggedEvents, 1)
	assert.Equal(t, "rate_limit_exceeded", loggedEvents[0].Type)
	assert.Contains(t, loggedEvents[0].Description, "/api/v1/checkin")
	assert.Contains(t, loggedEvents[0].Description, "100 requests")
	assert.Equal(t, "/api/v1/checkin", loggedEvents[0].Metadata["endpoint"])
	assert.Equal(t, 100, loggedEvents[0].Metadata["request_count"])
	
	mockMonitoring.AssertExpectations(t)
}

func TestSecurityLogger_LogInvalidRequest(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Set up mock expectations
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "invalid_request", mock.Anything, mock.Anything).Return(nil)
	
	ctx := context.Background()
	requestData := map[string]interface{}{
		"malformed_field": "invalid_value",
	}
	
	err := securityLogger.LogInvalidRequest(ctx, "test-device", "192.168.1.100", "test-agent", "malformed JSON", requestData)
	require.NoError(t, err)
	
	// Verify that the security event was logged
	loggedEvents := mockMonitoring.GetLoggedEvents()
	assert.Len(t, loggedEvents, 1)
	assert.Equal(t, "invalid_request", loggedEvents[0].Type)
	assert.Contains(t, loggedEvents[0].Description, "malformed JSON")
	assert.Equal(t, "malformed JSON", loggedEvents[0].Metadata["reason"])
	assert.Equal(t, requestData, loggedEvents[0].Metadata["request_data"])
	
	mockMonitoring.AssertExpectations(t)
}

func TestSecurityLogger_CleanupOldFailures(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	config.TimeWindow = 1 * time.Minute // Short window for testing
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Add some old failures
	now := time.Now()
	securityLogger.recentFailures = []time.Time{
		now.Add(-2 * time.Minute), // Old - should be removed
		now.Add(-30 * time.Second), // Recent - should be kept
		now.Add(-10 * time.Second), // Recent - should be kept
	}
	
	// Cleanup old failures
	securityLogger.cleanupOldFailures(now)
	
	// Should have only 2 recent failures
	assert.Equal(t, 2, len(securityLogger.recentFailures))
	
	// Verify the remaining failures are the recent ones
	assert.True(t, securityLogger.recentFailures[0].After(now.Add(-1*time.Minute)))
	assert.True(t, securityLogger.recentFailures[1].After(now.Add(-1*time.Minute)))
}

func TestSecurityLogger_GetRecentFailureCount(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	config.TimeWindow = 1 * time.Minute
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Add failures with different timestamps
	now := time.Now()
	securityLogger.recentFailures = []time.Time{
		now.Add(-2 * time.Minute), // Old - should not be counted
		now.Add(-30 * time.Second), // Recent - should be counted
		now.Add(-10 * time.Second), // Recent - should be counted
	}
	
	// Get recent failure count (should trigger cleanup)
	count := securityLogger.GetRecentFailureCount()
	
	// Should return only recent failures
	assert.Equal(t, 2, count)
}

func TestSecurityLogger_Reset(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Add some failures
	securityLogger.recentFailures = []time.Time{
		time.Now().Add(-1 * time.Minute),
		time.Now().Add(-30 * time.Second),
	}
	
	assert.Equal(t, 2, len(securityLogger.recentFailures))
	
	// Reset
	securityLogger.Reset()
	
	// Should be empty
	assert.Equal(t, 0, len(securityLogger.recentFailures))
}

func TestSecurityLogger_HTTPSecurityMiddleware(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Set up mock expectations for different status codes
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "authentication_failure", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "rate_limit_exceeded", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockMonitoring.On("LogSecurityEvent", mock.Anything, "invalid_request", mock.Anything, mock.Anything).Return(nil).Maybe()
	
	// Create middleware
	middleware := securityLogger.HTTPSecurityMiddleware("test-device")
	
	// Test unauthorized request
	unauthorizedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.Header.Set("User-Agent", "test-agent")
	
	rr := httptest.NewRecorder()
	middleware(unauthorizedHandler).ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	
	// Test rate limit exceeded
	rateLimitHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	
	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()
	middleware(rateLimitHandler).ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	
	// Test successful request (should not log security events)
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()
	middleware(successHandler).ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecurityLogger_GetClientIP(t *testing.T) {
	logger := logrus.New()
	mockMonitoring := &MockMonitoringSystem{}
	config := DefaultSecurityLoggerConfig()
	
	securityLogger := NewSecurityLogger(logger, mockMonitoring, config)
	
	// Test X-Forwarded-For header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"
	
	ip := securityLogger.getClientIP(req)
	assert.Equal(t, "192.168.1.100", ip)
	
	// Test X-Real-IP header (when X-Forwarded-For is not present)
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.200")
	req.RemoteAddr = "10.0.0.1:12345"
	
	ip = securityLogger.getClientIP(req)
	assert.Equal(t, "192.168.1.200", ip)
	
	// Test RemoteAddr fallback
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	
	ip = securityLogger.getClientIP(req)
	assert.Equal(t, "10.0.0.1:12345", ip)
}

func TestDefaultSecurityLoggerConfig(t *testing.T) {
	config := DefaultSecurityLoggerConfig()
	
	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 5*time.Minute, config.TimeWindow)
}