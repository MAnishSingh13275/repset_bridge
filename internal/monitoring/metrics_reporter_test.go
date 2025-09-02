package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/tier"
)

func TestNewCloudMetricsReporter(t *testing.T) {
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	assert.NotNil(t, reporter)
	assert.Equal(t, config.BaseURL, reporter.baseURL)
	assert.Equal(t, authenticator, reporter.authenticator)
	assert.NotNil(t, reporter.httpClient)
}

func TestCloudMetricsReporter_ReportMetrics(t *testing.T) {
	// Create a test server to receive the metrics
	var receivedPayload map[string]interface{}
	var receivedHeaders http.Header
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture headers
		receivedHeaders = r.Header.Clone()
		
		// Decode the request body
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/devices/metrics", r.URL.Path)
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create reporter with test server URL
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	config.BaseURL = server.URL
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	// Create test metrics
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	metrics := PerformanceMetrics{
		Timestamp:       now,
		DeviceID:        "test-device",
		QueueDepth:      100,
		QueueCapacity:   1000,
		CPUUsage:        25.5,
		MemoryUsage:     45.2,
		DiskUsage:       60.8,
		NetworkLatency:  50 * time.Millisecond,
		AdapterStatuses: map[string]string{"simulator": "active"},
		ErrorRate:       0.1,
		Tier:            tier.TierNormal,
		Metadata: map[string]interface{}{
			"custom_field": "custom_value",
		},
	}
	
	ctx := context.Background()
	err := reporter.ReportMetrics(ctx, metrics)
	require.NoError(t, err)
	
	// Verify the payload
	assert.Equal(t, "test-device", receivedPayload["deviceId"])
	assert.Equal(t, "2024-01-01T12:00:00Z", receivedPayload["timestamp"])
	assert.Equal(t, float64(100), receivedPayload["queueDepth"])
	assert.Equal(t, float64(1000), receivedPayload["queueCapacity"])
	assert.Equal(t, 25.5, receivedPayload["cpuUsage"])
	assert.Equal(t, 45.2, receivedPayload["memoryUsage"])
	assert.Equal(t, 60.8, receivedPayload["diskUsage"])
	assert.Equal(t, float64(50), receivedPayload["networkLatency"]) // Converted to milliseconds
	assert.Equal(t, "normal", receivedPayload["tier"])
	assert.Equal(t, 0.1, receivedPayload["errorRate"])
	
	// Verify adapter statuses
	adapterStatuses := receivedPayload["adapterStatuses"].(map[string]interface{})
	assert.Equal(t, "active", adapterStatuses["simulator"])
	
	// Verify metadata
	metadata := receivedPayload["metadata"].(map[string]interface{})
	assert.Equal(t, "custom_value", metadata["custom_field"])
	
	// Verify authentication headers
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "test-device", receivedHeaders.Get("X-Device-ID"))
	assert.NotEmpty(t, receivedHeaders.Get("X-Timestamp"))
	assert.NotEmpty(t, receivedHeaders.Get("X-Signature"))
}

func TestCloudMetricsReporter_ReportAlert(t *testing.T) {
	// Create a test server to receive the alert
	var receivedPayload map[string]interface{}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request body
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/devices/alerts", r.URL.Path)
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create reporter with test server URL
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	config.BaseURL = server.URL
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	// Create test alert
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	resolvedAt := now.Add(5 * time.Minute)
	alert := Alert{
		ID:          "test-alert",
		Type:        AlertTypeQueueThreshold,
		Severity:    AlertSeverityMedium,
		Title:       "Test Alert",
		Description: "This is a test alert",
		Timestamp:   now,
		DeviceID:    "test-device",
		Metadata: map[string]interface{}{
			"queue_depth": 800,
		},
		Resolved:   true,
		ResolvedAt: &resolvedAt,
	}
	
	ctx := context.Background()
	err := reporter.ReportAlert(ctx, alert)
	require.NoError(t, err)
	
	// Verify the payload
	assert.Equal(t, "test-alert", receivedPayload["id"])
	assert.Equal(t, "queue_threshold", receivedPayload["type"])
	assert.Equal(t, "medium", receivedPayload["severity"])
	assert.Equal(t, "Test Alert", receivedPayload["title"])
	assert.Equal(t, "This is a test alert", receivedPayload["description"])
	assert.Equal(t, "2024-01-01T12:00:00Z", receivedPayload["timestamp"])
	assert.Equal(t, "test-device", receivedPayload["deviceId"])
	assert.Equal(t, true, receivedPayload["resolved"])
	assert.Equal(t, "2024-01-01T12:05:00Z", receivedPayload["resolvedAt"])
	
	// Verify metadata
	metadata := receivedPayload["metadata"].(map[string]interface{})
	assert.Equal(t, float64(800), metadata["queue_depth"])
}

func TestCloudMetricsReporter_ReportSecurityEvent(t *testing.T) {
	// Create a test server to receive the security event
	var receivedPayload map[string]interface{}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request body
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/devices/security-events", r.URL.Path)
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create reporter with test server URL
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	config.BaseURL = server.URL
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	// Create test security event
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	event := SecurityEvent{
		ID:          "test-event",
		Type:        "hmac_validation_failure",
		Severity:    AlertSeverityHigh,
		Description: "HMAC validation failed",
		Timestamp:   now,
		DeviceID:    "test-device",
		SourceIP:    "192.168.1.100",
		UserAgent:   "test-agent",
		Metadata: map[string]interface{}{
			"attempts": 3,
		},
	}
	
	ctx := context.Background()
	err := reporter.ReportSecurityEvent(ctx, event)
	require.NoError(t, err)
	
	// Verify the payload
	assert.Equal(t, "test-event", receivedPayload["id"])
	assert.Equal(t, "hmac_validation_failure", receivedPayload["type"])
	assert.Equal(t, "high", receivedPayload["severity"])
	assert.Equal(t, "HMAC validation failed", receivedPayload["description"])
	assert.Equal(t, "2024-01-01T12:00:00Z", receivedPayload["timestamp"])
	assert.Equal(t, "test-device", receivedPayload["deviceId"])
	assert.Equal(t, "192.168.1.100", receivedPayload["sourceIp"])
	assert.Equal(t, "test-agent", receivedPayload["userAgent"])
	
	// Verify metadata
	metadata := receivedPayload["metadata"].(map[string]interface{})
	assert.Equal(t, float64(3), metadata["attempts"])
}

func TestCloudMetricsReporter_HTTPError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()
	
	// Create reporter with test server URL
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	config.BaseURL = server.URL
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	// Create test metrics
	metrics := PerformanceMetrics{
		Timestamp: time.Now(),
		DeviceID:  "test-device",
	}
	
	ctx := context.Background()
	err := reporter.ReportMetrics(ctx, metrics)
	
	// Should return an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCloudMetricsReporter_NetworkError(t *testing.T) {
	// Create reporter with invalid URL
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	config.BaseURL = "http://invalid-url-that-does-not-exist.local"
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	// Create test metrics
	metrics := PerformanceMetrics{
		Timestamp: time.Now(),
		DeviceID:  "test-device",
	}
	
	ctx := context.Background()
	err := reporter.ReportMetrics(ctx, metrics)
	
	// Should return a network error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to report metrics")
}

func TestMockMetricsReporter(t *testing.T) {
	logger := logrus.New()
	mockReporter := NewMockMetricsReporter(logger)
	
	// Test reporting metrics
	metrics := PerformanceMetrics{
		Timestamp: time.Now(),
		DeviceID:  "test-device",
		QueueDepth: 100,
	}
	
	ctx := context.Background()
	err := mockReporter.ReportMetrics(ctx, metrics)
	require.NoError(t, err)
	
	// Verify metrics were stored
	reportedMetrics := mockReporter.GetReportedMetrics()
	assert.Len(t, reportedMetrics, 1)
	assert.Equal(t, "test-device", reportedMetrics[0].DeviceID)
	assert.Equal(t, 100, reportedMetrics[0].QueueDepth)
	
	// Test reporting alerts
	alert := Alert{
		ID:       "test-alert",
		Type:     AlertTypeSecurityEvent,
		Severity: AlertSeverityHigh,
		DeviceID: "test-device",
	}
	
	err = mockReporter.ReportAlert(ctx, alert)
	require.NoError(t, err)
	
	// Verify alert was stored
	reportedAlerts := mockReporter.GetReportedAlerts()
	assert.Len(t, reportedAlerts, 1)
	assert.Equal(t, "test-alert", reportedAlerts[0].ID)
	
	// Test reporting security events
	event := SecurityEvent{
		ID:       "test-event",
		Type:     "hmac_validation_failure",
		Severity: AlertSeverityHigh,
		DeviceID: "test-device",
	}
	
	err = mockReporter.ReportSecurityEvent(ctx, event)
	require.NoError(t, err)
	
	// Verify security event was stored
	reportedEvents := mockReporter.GetReportedSecurityEvents()
	assert.Len(t, reportedEvents, 1)
	assert.Equal(t, "test-event", reportedEvents[0].ID)
	
	// Test reset
	mockReporter.Reset()
	assert.Empty(t, mockReporter.GetReportedMetrics())
	assert.Empty(t, mockReporter.GetReportedAlerts())
	assert.Empty(t, mockReporter.GetReportedSecurityEvents())
}

func TestDefaultCloudMetricsReporterConfig(t *testing.T) {
	config := DefaultCloudMetricsReporterConfig()
	
	assert.Equal(t, "https://api.yourdomain.com", config.BaseURL)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 5*time.Second, config.RetryDelay)
}

func TestCloudMetricsReporter_RetryLogic(t *testing.T) {
	// Create a test server that fails twice then succeeds
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create reporter with test server URL
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	config.BaseURL = server.URL
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	// Create test metrics
	metrics := PerformanceMetrics{
		Timestamp: time.Now(),
		DeviceID:  "test-device",
	}
	
	ctx := context.Background()
	err := reporter.ReportMetrics(ctx, metrics)
	
	// Should succeed after retries
	require.NoError(t, err)
	assert.Equal(t, 3, attemptCount) // Should have made 3 attempts
}

func TestCloudMetricsReporter_ClientError_NoRetry(t *testing.T) {
	// Create a test server that returns a client error (4xx)
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
	}))
	defer server.Close()
	
	// Create reporter with test server URL
	logger := logrus.New()
	config := DefaultCloudMetricsReporterConfig()
	config.BaseURL = server.URL
	authenticator := auth.NewHMACAuthenticator("test-device", "test-key")
	
	reporter := NewCloudMetricsReporter(logger, config, authenticator)
	
	// Create test metrics
	metrics := PerformanceMetrics{
		Timestamp: time.Now(),
		DeviceID:  "test-device",
	}
	
	ctx := context.Background()
	err := reporter.ReportMetrics(ctx, metrics)
	
	// Should fail without retries for client errors
	require.Error(t, err)
	assert.Equal(t, 1, attemptCount) // Should have made only 1 attempt
	assert.Contains(t, err.Error(), "400")
}