package logging

import (
	"fmt"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrorCategory represents different categories of errors for classification
type ErrorCategory string

const (
	// Hardware-related errors
	ErrorCategoryHardware ErrorCategory = "hardware"
	// Network-related errors
	ErrorCategoryNetwork ErrorCategory = "network"
	// Authentication/Security errors
	ErrorCategorySecurity ErrorCategory = "security"
	// Database/Storage errors
	ErrorCategoryStorage ErrorCategory = "storage"
	// Configuration errors
	ErrorCategoryConfig ErrorCategory = "config"
	// System resource errors
	ErrorCategoryResource ErrorCategory = "resource"
	// Service/Application errors
	ErrorCategoryService ErrorCategory = "service"
	// Unknown/Uncategorized errors
	ErrorCategoryUnknown ErrorCategory = "unknown"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	// Critical errors that require immediate attention
	ErrorSeverityCritical ErrorSeverity = "critical"
	// High priority errors that should be addressed soon
	ErrorSeverityHigh ErrorSeverity = "high"
	// Medium priority errors
	ErrorSeverityMedium ErrorSeverity = "medium"
	// Low priority errors or warnings
	ErrorSeverityLow ErrorSeverity = "low"
	// Informational messages
	ErrorSeverityInfo ErrorSeverity = "info"
)

// ErrorContext provides additional context for error logging
type ErrorContext struct {
	Category    ErrorCategory `json:"category"`
	Severity    ErrorSeverity `json:"severity"`
	Component   string        `json:"component"`
	Operation   string        `json:"operation"`
	UserID      string        `json:"user_id,omitempty"`
	DeviceID    string        `json:"device_id,omitempty"`
	AdapterName string        `json:"adapter_name,omitempty"`
	Recoverable bool          `json:"recoverable"`
	RetryCount  int           `json:"retry_count,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// StructuredError represents a structured error with context
type StructuredError struct {
	Err       error        `json:"error"`
	Context   ErrorContext `json:"context"`
	Timestamp time.Time    `json:"timestamp"`
	Stack     string       `json:"stack,omitempty"`
}

// Error implements the error interface
func (se *StructuredError) Error() string {
	if se.Err != nil {
		return se.Err.Error()
	}
	return "unknown error"
}

// Unwrap returns the underlying error
func (se *StructuredError) Unwrap() error {
	return se.Err
}

// NewStructuredError creates a new structured error with context
func NewStructuredError(err error, context ErrorContext) *StructuredError {
	structuredErr := &StructuredError{
		Err:       err,
		Context:   context,
		Timestamp: time.Now(),
	}

	// Capture stack trace for critical and high severity errors
	if context.Severity == ErrorSeverityCritical || context.Severity == ErrorSeverityHigh {
		structuredErr.Stack = captureStackTrace()
	}

	return structuredErr
}

// LogStructuredError logs a structured error with appropriate level and context
func LogStructuredError(logger *logrus.Logger, structuredErr *StructuredError) {
	if logger == nil || structuredErr == nil {
		return
	}

	// Create log entry with structured fields
	entry := logger.WithFields(logrus.Fields{
		"error_category":  structuredErr.Context.Category,
		"error_severity":  structuredErr.Context.Severity,
		"component":       structuredErr.Context.Component,
		"operation":       structuredErr.Context.Operation,
		"recoverable":     structuredErr.Context.Recoverable,
		"timestamp":       structuredErr.Timestamp,
	})

	// Add optional fields if present
	if structuredErr.Context.UserID != "" {
		entry = entry.WithField("user_id", structuredErr.Context.UserID)
	}
	if structuredErr.Context.DeviceID != "" {
		entry = entry.WithField("device_id", structuredErr.Context.DeviceID)
	}
	if structuredErr.Context.AdapterName != "" {
		entry = entry.WithField("adapter_name", structuredErr.Context.AdapterName)
	}
	if structuredErr.Context.RetryCount > 0 {
		entry = entry.WithField("retry_count", structuredErr.Context.RetryCount)
	}
	if structuredErr.Context.Metadata != nil {
		for key, value := range structuredErr.Context.Metadata {
			entry = entry.WithField(fmt.Sprintf("meta_%s", key), value)
		}
	}
	if structuredErr.Stack != "" {
		entry = entry.WithField("stack_trace", structuredErr.Stack)
	}

	// Log at appropriate level based on severity
	switch structuredErr.Context.Severity {
	case ErrorSeverityCritical:
		entry.Error(structuredErr.Error())
	case ErrorSeverityHigh:
		entry.Error(structuredErr.Error())
	case ErrorSeverityMedium:
		entry.Warn(structuredErr.Error())
	case ErrorSeverityLow:
		entry.Warn(structuredErr.Error())
	case ErrorSeverityInfo:
		entry.Info(structuredErr.Error())
	default:
		entry.Error(structuredErr.Error())
	}
}

// LogHardwareError logs hardware-related errors
func LogHardwareError(logger *logrus.Logger, err error, adapterName, operation string, recoverable bool) {
	context := ErrorContext{
		Category:    ErrorCategoryHardware,
		Severity:    ErrorSeverityHigh,
		Component:   "adapter",
		Operation:   operation,
		AdapterName: adapterName,
		Recoverable: recoverable,
	}
	
	structuredErr := NewStructuredError(err, context)
	LogStructuredError(logger, structuredErr)
}

// LogNetworkError logs network-related errors
func LogNetworkError(logger *logrus.Logger, err error, operation string, retryCount int, recoverable bool) {
	severity := ErrorSeverityMedium
	if retryCount > 3 {
		severity = ErrorSeverityHigh
	}

	context := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    severity,
		Component:   "client",
		Operation:   operation,
		Recoverable: recoverable,
		RetryCount:  retryCount,
	}
	
	structuredErr := NewStructuredError(err, context)
	LogStructuredError(logger, structuredErr)
}

// LogSecurityError logs security-related errors
func LogSecurityError(logger *logrus.Logger, err error, deviceID, operation string) {
	context := ErrorContext{
		Category:    ErrorCategorySecurity,
		Severity:    ErrorSeverityCritical,
		Component:   "auth",
		Operation:   operation,
		DeviceID:    deviceID,
		Recoverable: false,
	}
	
	structuredErr := NewStructuredError(err, context)
	LogStructuredError(logger, structuredErr)
}

// LogStorageError logs database/storage-related errors
func LogStorageError(logger *logrus.Logger, err error, operation string, recoverable bool) {
	severity := ErrorSeverityHigh
	if !recoverable {
		severity = ErrorSeverityCritical
	}

	context := ErrorContext{
		Category:    ErrorCategoryStorage,
		Severity:    severity,
		Component:   "database",
		Operation:   operation,
		Recoverable: recoverable,
	}
	
	structuredErr := NewStructuredError(err, context)
	LogStructuredError(logger, structuredErr)
}

// LogResourceError logs system resource-related errors
func LogResourceError(logger *logrus.Logger, err error, resourceType string, currentValue, threshold float64) {
	context := ErrorContext{
		Category:    ErrorCategoryResource,
		Severity:    ErrorSeverityMedium,
		Component:   "tier",
		Operation:   "resource_monitoring",
		Recoverable: true,
		Metadata: map[string]interface{}{
			"resource_type":    resourceType,
			"current_value":    currentValue,
			"threshold":        threshold,
			"usage_percentage": (currentValue / threshold) * 100,
		},
	}
	
	structuredErr := NewStructuredError(err, context)
	LogStructuredError(logger, structuredErr)
}

// LogServiceError logs service/application-related errors
func LogServiceError(logger *logrus.Logger, err error, serviceName, operation string, recoverable bool) {
	severity := ErrorSeverityMedium
	if !recoverable {
		severity = ErrorSeverityHigh
	}

	context := ErrorContext{
		Category:    ErrorCategoryService,
		Severity:    severity,
		Component:   serviceName,
		Operation:   operation,
		Recoverable: recoverable,
	}
	
	structuredErr := NewStructuredError(err, context)
	LogStructuredError(logger, structuredErr)
}

// LogRecoveryAttempt logs error recovery attempts
func LogRecoveryAttempt(logger *logrus.Logger, originalErr error, recoveryAction string, success bool) {
	severity := ErrorSeverityInfo
	if !success {
		severity = ErrorSeverityMedium
	}

	context := ErrorContext{
		Category:    ErrorCategoryService,
		Severity:    severity,
		Component:   "recovery",
		Operation:   recoveryAction,
		Recoverable: true,
		Metadata: map[string]interface{}{
			"recovery_success": success,
			"original_error":   originalErr.Error(),
		},
	}

	var err error
	if success {
		err = fmt.Errorf("recovery successful for: %v", originalErr)
	} else {
		err = fmt.Errorf("recovery failed for: %v", originalErr)
	}
	
	structuredErr := NewStructuredError(err, context)
	LogStructuredError(logger, structuredErr)
}

// captureStackTrace captures the current stack trace
func captureStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// ClassifyError attempts to classify an error based on its type and message
func ClassifyError(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryUnknown
	}

	errMsg := err.Error()
	
	// Network errors
	networkKeywords := []string{
		"connection refused", "connection reset", "connection timeout",
		"network is unreachable", "no such host", "i/o timeout",
		"dial tcp", "dial udp", "dns", "tls handshake",
	}
	for _, keyword := range networkKeywords {
		if contains(errMsg, keyword) {
			return ErrorCategoryNetwork
		}
	}

	// Hardware errors
	hardwareKeywords := []string{
		"device not found", "hardware", "adapter", "fingerprint",
		"rfid", "scanner", "reader", "port", "serial",
	}
	for _, keyword := range hardwareKeywords {
		if contains(errMsg, keyword) {
			return ErrorCategoryHardware
		}
	}

	// Security errors
	securityKeywords := []string{
		"authentication", "authorization", "hmac", "signature",
		"unauthorized", "forbidden", "invalid token", "expired",
	}
	for _, keyword := range securityKeywords {
		if contains(errMsg, keyword) {
			return ErrorCategorySecurity
		}
	}

	// Storage errors
	storageKeywords := []string{
		"database", "sqlite", "sql", "table", "constraint",
		"disk", "file", "permission", "no space left",
	}
	for _, keyword := range storageKeywords {
		if contains(errMsg, keyword) {
			return ErrorCategoryStorage
		}
	}

	// Resource errors
	resourceKeywords := []string{
		"memory", "cpu", "resource", "limit", "quota",
		"out of memory", "too many", "capacity",
	}
	for _, keyword := range resourceKeywords {
		if contains(errMsg, keyword) {
			return ErrorCategoryResource
		}
	}

	// Configuration errors
	configKeywords := []string{
		"config", "configuration", "invalid", "missing",
		"parse", "yaml", "json", "setting",
	}
	for _, keyword := range configKeywords {
		if contains(errMsg, keyword) {
			return ErrorCategoryConfig
		}
	}

	return ErrorCategoryUnknown
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      containsSubstring(s, substr))))
}

// containsSubstring performs case-insensitive substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}