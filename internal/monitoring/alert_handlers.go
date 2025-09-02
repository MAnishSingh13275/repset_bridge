package monitoring

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// LogAlertHandler logs alerts to the system log
type LogAlertHandler struct {
	logger *logrus.Logger
}

// NewLogAlertHandler creates a new log-based alert handler
func NewLogAlertHandler(logger *logrus.Logger) *LogAlertHandler {
	return &LogAlertHandler{
		logger: logger,
	}
}

// HandleAlert logs the alert with appropriate severity
func (h *LogAlertHandler) HandleAlert(ctx context.Context, alert Alert) error {
	logEntry := h.logger.WithFields(logrus.Fields{
		"alert_id":    alert.ID,
		"alert_type":  string(alert.Type),
		"severity":    string(alert.Severity),
		"device_id":   alert.DeviceID,
		"timestamp":   alert.Timestamp,
		"resolved":    alert.Resolved,
		"metadata":    alert.Metadata,
	})

	message := fmt.Sprintf("[%s] %s: %s", alert.Severity, alert.Title, alert.Description)

	switch alert.Severity {
	case AlertSeverityCritical:
		logEntry.Error(message)
	case AlertSeverityHigh:
		logEntry.Warn(message)
	case AlertSeverityMedium:
		logEntry.Info(message)
	case AlertSeverityLow:
		logEntry.Debug(message)
	default:
		logEntry.Info(message)
	}

	return nil
}

// FileAlertHandler writes alerts to a file for external monitoring systems
type FileAlertHandler struct {
	logger   *logrus.Logger
	filePath string
}

// NewFileAlertHandler creates a new file-based alert handler
func NewFileAlertHandler(logger *logrus.Logger, filePath string) *FileAlertHandler {
	return &FileAlertHandler{
		logger:   logger,
		filePath: filePath,
	}
}

// HandleAlert writes the alert to a file in JSON format
func (h *FileAlertHandler) HandleAlert(ctx context.Context, alert Alert) error {
	// Ensure directory exists
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create alert directory: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(h.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open alert file: %w", err)
	}
	defer file.Close()

	// Write alert as JSON line
	alertLine := fmt.Sprintf(`{"id":"%s","type":"%s","severity":"%s","title":"%s","description":"%s","timestamp":"%s","deviceId":"%s","resolved":%t}%s`,
		alert.ID,
		alert.Type,
		alert.Severity,
		alert.Title,
		alert.Description,
		alert.Timestamp.Format(time.RFC3339),
		alert.DeviceID,
		alert.Resolved,
		"\n",
	)

	if _, err := file.WriteString(alertLine); err != nil {
		return fmt.Errorf("failed to write alert to file: %w", err)
	}

	h.logger.Debug("Alert written to file", "file", h.filePath, "alert_id", alert.ID)
	return nil
}

// ConsoleAlertHandler prints critical alerts to console for immediate attention
type ConsoleAlertHandler struct {
	logger *logrus.Logger
}

// NewConsoleAlertHandler creates a new console-based alert handler
func NewConsoleAlertHandler(logger *logrus.Logger) *ConsoleAlertHandler {
	return &ConsoleAlertHandler{
		logger: logger,
	}
}

// HandleAlert prints critical alerts to console
func (h *ConsoleAlertHandler) HandleAlert(ctx context.Context, alert Alert) error {
	// Only print critical and high severity alerts to console
	if alert.Severity != AlertSeverityCritical && alert.Severity != AlertSeverityHigh {
		return nil
	}

	// Format alert for console display
	timestamp := alert.Timestamp.Format("2006-01-02 15:04:05")
	
	fmt.Printf("\n=== ALERT [%s] ===\n", alert.Severity)
	fmt.Printf("Time: %s\n", timestamp)
	fmt.Printf("Type: %s\n", alert.Type)
	fmt.Printf("Title: %s\n", alert.Title)
	fmt.Printf("Description: %s\n", alert.Description)
	if alert.DeviceID != "" {
		fmt.Printf("Device: %s\n", alert.DeviceID)
	}
	fmt.Printf("Status: %s\n", func() string {
		if alert.Resolved {
			return "RESOLVED"
		}
		return "ACTIVE"
	}())
	fmt.Printf("========================\n\n")

	return nil
}

// CompositeAlertHandler combines multiple alert handlers
type CompositeAlertHandler struct {
	handlers []AlertHandler
	logger   *logrus.Logger
}

// NewCompositeAlertHandler creates a new composite alert handler
func NewCompositeAlertHandler(logger *logrus.Logger, handlers ...AlertHandler) *CompositeAlertHandler {
	return &CompositeAlertHandler{
		handlers: handlers,
		logger:   logger,
	}
}

// HandleAlert sends the alert to all configured handlers
func (h *CompositeAlertHandler) HandleAlert(ctx context.Context, alert Alert) error {
	var lastError error
	successCount := 0

	for i, handler := range h.handlers {
		if err := handler.HandleAlert(ctx, alert); err != nil {
			h.logger.WithError(err).WithField("handler_index", i).Error("Alert handler failed")
			lastError = err
		} else {
			successCount++
		}
	}

	// Return error only if all handlers failed
	if successCount == 0 && lastError != nil {
		return fmt.Errorf("all alert handlers failed, last error: %w", lastError)
	}

	return nil
}

// AddHandler adds a new handler to the composite handler
func (h *CompositeAlertHandler) AddHandler(handler AlertHandler) {
	h.handlers = append(h.handlers, handler)
}