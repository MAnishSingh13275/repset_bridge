package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/auth"
)

// CloudMetricsReporter sends metrics, alerts, and security events to the cloud admin portal
type CloudMetricsReporter struct {
	logger        *logrus.Logger
	httpClient    *http.Client
	baseURL       string
	authenticator *auth.HMACAuthenticator
}

// CloudMetricsReporterConfig holds configuration for the cloud metrics reporter
type CloudMetricsReporterConfig struct {
	BaseURL        string        `json:"baseUrl"`        // Base URL for the admin portal API
	Timeout        time.Duration `json:"timeout"`        // HTTP request timeout
	RetryAttempts  int           `json:"retryAttempts"`  // Number of retry attempts
	RetryDelay     time.Duration `json:"retryDelay"`     // Delay between retry attempts
}

// DefaultCloudMetricsReporterConfig returns default configuration
func DefaultCloudMetricsReporterConfig() CloudMetricsReporterConfig {
	return CloudMetricsReporterConfig{
		BaseURL:       "https://api.yourdomain.com",
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    5 * time.Second,
	}
}

// NewCloudMetricsReporter creates a new cloud metrics reporter
func NewCloudMetricsReporter(
	logger *logrus.Logger,
	config CloudMetricsReporterConfig,
	authenticator *auth.HMACAuthenticator,
) *CloudMetricsReporter {
	return &CloudMetricsReporter{
		logger:        logger,
		baseURL:       config.BaseURL,
		authenticator: authenticator,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ReportMetrics sends performance metrics to the cloud
func (r *CloudMetricsReporter) ReportMetrics(ctx context.Context, metrics PerformanceMetrics) error {
	endpoint := "/api/v1/devices/metrics"
	
	// Prepare request payload
	payload := map[string]interface{}{
		"deviceId":        metrics.DeviceID,
		"timestamp":       metrics.Timestamp.Format(time.RFC3339),
		"queueDepth":      metrics.QueueDepth,
		"queueCapacity":   metrics.QueueCapacity,
		"cpuUsage":        metrics.CPUUsage,
		"memoryUsage":     metrics.MemoryUsage,
		"diskUsage":       metrics.DiskUsage,
		"networkLatency":  metrics.NetworkLatency.Milliseconds(),
		"adapterStatuses": metrics.AdapterStatuses,
		"errorRate":       metrics.ErrorRate,
		"tier":            string(metrics.Tier),
		"metadata":        metrics.Metadata,
	}
	
	if err := r.sendRequest(ctx, "POST", endpoint, payload); err != nil {
		return fmt.Errorf("failed to report metrics: %w", err)
	}
	
	r.logger.Debug("Metrics reported to cloud", "device_id", metrics.DeviceID, "timestamp", metrics.Timestamp)
	return nil
}

// ReportAlert sends an alert to the cloud
func (r *CloudMetricsReporter) ReportAlert(ctx context.Context, alert Alert) error {
	endpoint := "/api/v1/devices/alerts"
	
	// Prepare request payload
	payload := map[string]interface{}{
		"id":          alert.ID,
		"type":        string(alert.Type),
		"severity":    string(alert.Severity),
		"title":       alert.Title,
		"description": alert.Description,
		"timestamp":   alert.Timestamp.Format(time.RFC3339),
		"deviceId":    alert.DeviceID,
		"metadata":    alert.Metadata,
		"resolved":    alert.Resolved,
	}
	
	if alert.ResolvedAt != nil {
		payload["resolvedAt"] = alert.ResolvedAt.Format(time.RFC3339)
	}
	
	if err := r.sendRequest(ctx, "POST", endpoint, payload); err != nil {
		return fmt.Errorf("failed to report alert: %w", err)
	}
	
	r.logger.Debug("Alert reported to cloud", "alert_id", alert.ID, "type", alert.Type, "severity", alert.Severity)
	return nil
}

// ReportSecurityEvent sends a security event to the cloud
func (r *CloudMetricsReporter) ReportSecurityEvent(ctx context.Context, event SecurityEvent) error {
	endpoint := "/api/v1/devices/security-events"
	
	// Prepare request payload
	payload := map[string]interface{}{
		"id":          event.ID,
		"type":        event.Type,
		"severity":    string(event.Severity),
		"description": event.Description,
		"timestamp":   event.Timestamp.Format(time.RFC3339),
		"deviceId":    event.DeviceID,
		"sourceIp":    event.SourceIP,
		"userAgent":   event.UserAgent,
		"metadata":    event.Metadata,
	}
	
	if err := r.sendRequest(ctx, "POST", endpoint, payload); err != nil {
		return fmt.Errorf("failed to report security event: %w", err)
	}
	
	r.logger.Debug("Security event reported to cloud", "event_id", event.ID, "type", event.Type, "severity", event.Severity)
	return nil
}

// sendRequest sends an authenticated HTTP request to the cloud API
func (r *CloudMetricsReporter) sendRequest(ctx context.Context, method, endpoint string, payload interface{}) error {
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	// Create HTTP request
	url := r.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", r.authenticator.GetDeviceID())
	
	// Add HMAC authentication
	timestamp := time.Now().Unix()
	signature, err := r.authenticator.SignRequest(jsonData, timestamp)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}
	
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-Signature", signature)
	
	// Send request with retries
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		
		resp, err := r.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			r.logger.WithError(err).WithField("attempt", attempt+1).Warn("Request failed, retrying")
			continue
		}
		
		// Check response status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}
		
		// Read error response
		var errorBody bytes.Buffer
		errorBody.ReadFrom(resp.Body)
		resp.Body.Close()
		
		lastErr = fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, errorBody.String())
		
		// Don't retry on client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}
		
		r.logger.WithField("status_code", resp.StatusCode).WithField("attempt", attempt+1).Warn("Request failed, retrying")
	}
	
	return lastErr
}

// MockMetricsReporter is a mock implementation for testing
type MockMetricsReporter struct {
	logger          *logrus.Logger
	reportedMetrics []PerformanceMetrics
	reportedAlerts  []Alert
	reportedEvents  []SecurityEvent
}

// NewMockMetricsReporter creates a new mock metrics reporter
func NewMockMetricsReporter(logger *logrus.Logger) *MockMetricsReporter {
	return &MockMetricsReporter{
		logger: logger,
	}
}

// ReportMetrics stores metrics for testing
func (m *MockMetricsReporter) ReportMetrics(ctx context.Context, metrics PerformanceMetrics) error {
	m.reportedMetrics = append(m.reportedMetrics, metrics)
	m.logger.Debug("Mock: Metrics reported", "device_id", metrics.DeviceID)
	return nil
}

// ReportAlert stores alerts for testing
func (m *MockMetricsReporter) ReportAlert(ctx context.Context, alert Alert) error {
	m.reportedAlerts = append(m.reportedAlerts, alert)
	m.logger.Debug("Mock: Alert reported", "alert_id", alert.ID)
	return nil
}

// ReportSecurityEvent stores security events for testing
func (m *MockMetricsReporter) ReportSecurityEvent(ctx context.Context, event SecurityEvent) error {
	m.reportedEvents = append(m.reportedEvents, event)
	m.logger.Debug("Mock: Security event reported", "event_id", event.ID)
	return nil
}

// GetReportedMetrics returns all reported metrics (for testing)
func (m *MockMetricsReporter) GetReportedMetrics() []PerformanceMetrics {
	return m.reportedMetrics
}

// GetReportedAlerts returns all reported alerts (for testing)
func (m *MockMetricsReporter) GetReportedAlerts() []Alert {
	return m.reportedAlerts
}

// GetReportedSecurityEvents returns all reported security events (for testing)
func (m *MockMetricsReporter) GetReportedSecurityEvents() []SecurityEvent {
	return m.reportedEvents
}

// Reset clears all stored data (for testing)
func (m *MockMetricsReporter) Reset() {
	m.reportedMetrics = nil
	m.reportedAlerts = nil
	m.reportedEvents = nil
}