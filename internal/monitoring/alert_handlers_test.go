package monitoring

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLogAlertHandler(t *testing.T) {
	// Create a logger that writes to a buffer so we can check the output
	logger := logrus.New()
	var logOutput strings.Builder
	logger.SetOutput(&logOutput)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true, // Make output predictable for testing
	})
	
	handler := NewLogAlertHandler(logger)
	
	alert := Alert{
		ID:          "test-alert",
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityHigh,
		Title:       "Test Alert",
		Description: "This is a test alert",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
		Metadata: map[string]interface{}{
			"source_ip": "192.168.1.100",
		},
	}
	
	ctx := context.Background()
	err := handler.HandleAlert(ctx, alert)
	require.NoError(t, err)
	
	// Check that the alert was logged
	logContent := logOutput.String()
	assert.Contains(t, logContent, "Test Alert")
	assert.Contains(t, logContent, "This is a test alert")
	assert.Contains(t, logContent, "test-alert")
	assert.Contains(t, logContent, "security_event")
	assert.Contains(t, logContent, "high")
}

func TestFileAlertHandler(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	alertFile := filepath.Join(tempDir, "alerts.jsonl")
	
	logger := logrus.New()
	handler := NewFileAlertHandler(logger, alertFile)
	
	alert := Alert{
		ID:          "test-alert",
		Type:        AlertTypeQueueThreshold,
		Severity:    AlertSeverityMedium,
		Title:       "Queue Alert",
		Description: "Queue is getting full",
		Timestamp:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		DeviceID:    "test-device",
		Resolved:    false,
	}
	
	ctx := context.Background()
	err := handler.HandleAlert(ctx, alert)
	require.NoError(t, err)
	
	// Check that the file was created and contains the alert
	assert.FileExists(t, alertFile)
	
	content, err := os.ReadFile(alertFile)
	require.NoError(t, err)
	
	alertLine := string(content)
	assert.Contains(t, alertLine, `"id":"test-alert"`)
	assert.Contains(t, alertLine, `"type":"queue_threshold"`)
	assert.Contains(t, alertLine, `"severity":"medium"`)
	assert.Contains(t, alertLine, `"title":"Queue Alert"`)
	assert.Contains(t, alertLine, `"description":"Queue is getting full"`)
	assert.Contains(t, alertLine, `"deviceId":"test-device"`)
	assert.Contains(t, alertLine, `"resolved":false`)
	assert.Contains(t, alertLine, `"timestamp":"2024-01-01T12:00:00Z"`)
}

func TestFileAlertHandler_MultipleAlerts(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	alertFile := filepath.Join(tempDir, "alerts.jsonl")
	
	logger := logrus.New()
	handler := NewFileAlertHandler(logger, alertFile)
	
	alerts := []Alert{
		{
			ID:          "alert-1",
			Type:        AlertTypeSecurityEvent,
			Severity:    AlertSeverityHigh,
			Title:       "Security Alert 1",
			Description: "First security alert",
			Timestamp:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			DeviceID:    "test-device",
		},
		{
			ID:          "alert-2",
			Type:        AlertTypePerformanceDegradation,
			Severity:    AlertSeverityMedium,
			Title:       "Performance Alert",
			Description: "Performance is degraded",
			Timestamp:   time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
			DeviceID:    "test-device",
		},
	}
	
	ctx := context.Background()
	for _, alert := range alerts {
		err := handler.HandleAlert(ctx, alert)
		require.NoError(t, err)
	}
	
	// Check that both alerts were written to the file
	content, err := os.ReadFile(alertFile)
	require.NoError(t, err)
	
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, 2)
	
	// Check first alert
	assert.Contains(t, lines[0], `"id":"alert-1"`)
	assert.Contains(t, lines[0], `"title":"Security Alert 1"`)
	
	// Check second alert
	assert.Contains(t, lines[1], `"id":"alert-2"`)
	assert.Contains(t, lines[1], `"title":"Performance Alert"`)
}

func TestConsoleAlertHandler(t *testing.T) {
	logger := logrus.New()
	handler := NewConsoleAlertHandler(logger)
	
	// Test critical alert (should be printed)
	criticalAlert := Alert{
		ID:          "critical-alert",
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityCritical,
		Title:       "Critical Alert",
		Description: "This is critical",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
	}
	
	ctx := context.Background()
	err := handler.HandleAlert(ctx, criticalAlert)
	require.NoError(t, err)
	
	// Test low severity alert (should not be printed to console)
	lowAlert := Alert{
		ID:          "low-alert",
		Type:        AlertTypeQueueThreshold,
		Severity:    AlertSeverityLow,
		Title:       "Low Alert",
		Description: "This is low priority",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
	}
	
	err = handler.HandleAlert(ctx, lowAlert)
	require.NoError(t, err)
	
	// Note: We can't easily test console output in unit tests,
	// but we can verify that the handler doesn't return errors
}

func TestCompositeAlertHandler(t *testing.T) {
	logger := logrus.New()
	
	// Create temporary file for file handler
	tempDir := t.TempDir()
	alertFile := filepath.Join(tempDir, "alerts.jsonl")
	
	// Create individual handlers
	logHandler := NewLogAlertHandler(logger)
	fileHandler := NewFileAlertHandler(logger, alertFile)
	consoleHandler := NewConsoleAlertHandler(logger)
	
	// Create composite handler
	compositeHandler := NewCompositeAlertHandler(logger, logHandler, fileHandler, consoleHandler)
	
	alert := Alert{
		ID:          "composite-test",
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityHigh,
		Title:       "Composite Test Alert",
		Description: "Testing composite handler",
		Timestamp:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		DeviceID:    "test-device",
	}
	
	ctx := context.Background()
	err := compositeHandler.HandleAlert(ctx, alert)
	require.NoError(t, err)
	
	// Verify that the file handler wrote the alert
	assert.FileExists(t, alertFile)
	content, err := os.ReadFile(alertFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "composite-test")
}

func TestCompositeAlertHandler_AddHandler(t *testing.T) {
	logger := logrus.New()
	
	// Create composite handler with one handler
	logHandler := NewLogAlertHandler(logger)
	compositeHandler := NewCompositeAlertHandler(logger, logHandler)
	
	// Add another handler
	tempDir := t.TempDir()
	alertFile := filepath.Join(tempDir, "alerts.jsonl")
	fileHandler := NewFileAlertHandler(logger, alertFile)
	compositeHandler.AddHandler(fileHandler)
	
	alert := Alert{
		ID:          "add-handler-test",
		Type:        AlertTypeQueueThreshold,
		Severity:    AlertSeverityMedium,
		Title:       "Add Handler Test",
		Description: "Testing add handler functionality",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
	}
	
	ctx := context.Background()
	err := compositeHandler.HandleAlert(ctx, alert)
	require.NoError(t, err)
	
	// Verify that the added file handler worked
	assert.FileExists(t, alertFile)
	content, err := os.ReadFile(alertFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "add-handler-test")
}

func TestCompositeAlertHandler_HandlerFailure(t *testing.T) {
	logger := logrus.New()
	
	// Create a mock handler that always fails
	failingHandler := &MockAlertHandler{}
	failingHandler.On("HandleAlert", mock.Anything, mock.AnythingOfType("Alert")).Return(assert.AnError)
	
	// Create a working handler
	workingHandler := &MockAlertHandler{}
	workingHandler.On("HandleAlert", mock.Anything, mock.AnythingOfType("Alert")).Return(nil)
	
	// Create composite handler with both
	compositeHandler := NewCompositeAlertHandler(logger, failingHandler, workingHandler)
	
	alert := Alert{
		ID:          "failure-test",
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityHigh,
		Title:       "Failure Test",
		Description: "Testing handler failure",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
	}
	
	ctx := context.Background()
	err := compositeHandler.HandleAlert(ctx, alert)
	
	// Should not return error because one handler succeeded
	require.NoError(t, err)
	
	// Verify both handlers were called
	failingHandler.AssertExpectations(t)
	workingHandler.AssertExpectations(t)
	
	// Verify the working handler received the alert
	handledAlerts := workingHandler.GetHandledAlerts()
	assert.Len(t, handledAlerts, 1)
	assert.Equal(t, "failure-test", handledAlerts[0].ID)
}

func TestCompositeAlertHandler_AllHandlersFail(t *testing.T) {
	logger := logrus.New()
	
	// Create two mock handlers that both fail
	failingHandler1 := &MockAlertHandler{}
	failingHandler1.On("HandleAlert", mock.Anything, mock.AnythingOfType("Alert")).Return(assert.AnError)
	
	failingHandler2 := &MockAlertHandler{}
	failingHandler2.On("HandleAlert", mock.Anything, mock.AnythingOfType("Alert")).Return(assert.AnError)
	
	// Create composite handler with both failing handlers
	compositeHandler := NewCompositeAlertHandler(logger, failingHandler1, failingHandler2)
	
	alert := Alert{
		ID:          "all-fail-test",
		Type:        AlertTypeSecurityEvent,
		Severity:    AlertSeverityHigh,
		Title:       "All Fail Test",
		Description: "Testing all handlers failing",
		Timestamp:   time.Now(),
		DeviceID:    "test-device",
	}
	
	ctx := context.Background()
	err := compositeHandler.HandleAlert(ctx, alert)
	
	// Should return error because all handlers failed
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all alert handlers failed")
	
	// Verify both handlers were called
	failingHandler1.AssertExpectations(t)
	failingHandler2.AssertExpectations(t)
}