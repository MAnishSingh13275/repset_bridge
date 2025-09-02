package logging

import (
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewStructuredError(t *testing.T) {
	err := errors.New("test error")
	context := ErrorContext{
		Category:    ErrorCategoryHardware,
		Severity:    ErrorSeverityHigh,
		Component:   "adapter",
		Operation:   "initialize",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, context)

	assert.NotNil(t, structuredErr)
	assert.Equal(t, err, structuredErr.Err)
	assert.Equal(t, context, structuredErr.Context)
	assert.False(t, structuredErr.Timestamp.IsZero())
	assert.NotEmpty(t, structuredErr.Stack) // Should have stack trace for high severity
}

func TestStructuredErrorInterface(t *testing.T) {
	originalErr := errors.New("original error")
	context := ErrorContext{
		Category:  ErrorCategoryNetwork,
		Severity:  ErrorSeverityMedium,
		Component: "client",
	}

	structuredErr := NewStructuredError(originalErr, context)

	// Test Error() method
	assert.Equal(t, "original error", structuredErr.Error())

	// Test Unwrap() method
	assert.Equal(t, originalErr, structuredErr.Unwrap())
}

func TestLogStructuredError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	err := errors.New("test error")
	context := ErrorContext{
		Category:    ErrorCategoryStorage,
		Severity:    ErrorSeverityCritical,
		Component:   "database",
		Operation:   "insert",
		DeviceID:    "test-device",
		Recoverable: false,
		Metadata: map[string]interface{}{
			"table": "events",
			"count": 5,
		},
	}

	structuredErr := NewStructuredError(err, context)

	// This should not panic
	LogStructuredError(logger, structuredErr)

	// Test with nil logger (should not panic)
	LogStructuredError(nil, structuredErr)

	// Test with nil structured error (should not panic)
	LogStructuredError(logger, nil)
}

func TestLogHardwareError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	err := errors.New("hardware communication failed")

	// This should not panic
	LogHardwareError(logger, err, "fingerprint-reader", "scan", true)
}

func TestLogNetworkError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	err := errors.New("connection timeout")

	// Test with low retry count (should be medium severity)
	LogNetworkError(logger, err, "submit_events", 1, true)

	// Test with high retry count (should be high severity)
	LogNetworkError(logger, err, "submit_events", 5, true)
}

func TestLogSecurityError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	err := errors.New("HMAC validation failed")

	// This should not panic
	LogSecurityError(logger, err, "test-device-123", "authenticate")
}

func TestLogStorageError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	err := errors.New("database locked")

	// Test recoverable error (should be high severity)
	LogStorageError(logger, err, "insert_event", true)

	// Test non-recoverable error (should be critical severity)
	LogStorageError(logger, err, "create_table", false)
}

func TestLogResourceError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	err := errors.New("memory usage high")

	// This should not panic
	LogResourceError(logger, err, "memory", 85.5, 100.0)
}

func TestLogServiceError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	err := errors.New("service startup failed")

	// Test recoverable error (should be medium severity)
	LogServiceError(logger, err, "heartbeat", "start", true)

	// Test non-recoverable error (should be high severity)
	LogServiceError(logger, err, "core", "initialize", false)
}

func TestLogRecoveryAttempt(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	originalErr := errors.New("original error")

	// Test successful recovery
	LogRecoveryAttempt(logger, originalErr, "retry", true)

	// Test failed recovery
	LogRecoveryAttempt(logger, originalErr, "restart", false)
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorCategoryUnknown,
		},
		{
			name:     "network error - connection refused",
			err:      errors.New("connection refused"),
			expected: ErrorCategoryNetwork,
		},
		{
			name:     "network error - timeout",
			err:      errors.New("i/o timeout"),
			expected: ErrorCategoryNetwork,
		},
		{
			name:     "hardware error - device not found",
			err:      errors.New("device not found"),
			expected: ErrorCategoryHardware,
		},
		{
			name:     "hardware error - adapter",
			err:      errors.New("adapter initialization failed"),
			expected: ErrorCategoryHardware,
		},
		{
			name:     "security error - authentication",
			err:      errors.New("authentication failed"),
			expected: ErrorCategorySecurity,
		},
		{
			name:     "security error - hmac",
			err:      errors.New("hmac validation error"),
			expected: ErrorCategorySecurity,
		},
		{
			name:     "storage error - database",
			err:      errors.New("database connection failed"),
			expected: ErrorCategoryStorage,
		},
		{
			name:     "storage error - sqlite",
			err:      errors.New("sqlite: table locked"),
			expected: ErrorCategoryStorage,
		},
		{
			name:     "resource error - memory",
			err:      errors.New("out of memory"),
			expected: ErrorCategoryResource,
		},
		{
			name:     "resource error - cpu",
			err:      errors.New("cpu limit exceeded"),
			expected: ErrorCategoryResource,
		},
		{
			name:     "config error - invalid",
			err:      errors.New("invalid configuration"),
			expected: ErrorCategoryConfig,
		},
		{
			name:     "config error - parse",
			err:      errors.New("failed to parse yaml"),
			expected: ErrorCategoryConfig,
		},
		{
			name:     "unknown error",
			err:      errors.New("something went wrong"),
			expected: ErrorCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorSeverityString(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		expected string
	}{
		{ErrorSeverityCritical, "critical"},
		{ErrorSeverityHigh, "high"},
		{ErrorSeverityMedium, "medium"},
		{ErrorSeverityLow, "low"},
		{ErrorSeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.severity))
		})
	}
}

func TestErrorCategoryString(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		expected string
	}{
		{ErrorCategoryHardware, "hardware"},
		{ErrorCategoryNetwork, "network"},
		{ErrorCategorySecurity, "security"},
		{ErrorCategoryStorage, "storage"},
		{ErrorCategoryConfig, "config"},
		{ErrorCategoryResource, "resource"},
		{ErrorCategoryService, "service"},
		{ErrorCategoryUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.category))
		})
	}
}

func TestStructuredErrorWithMetadata(t *testing.T) {
	err := errors.New("test error with metadata")
	metadata := map[string]interface{}{
		"user_id":    "user123",
		"request_id": "req456",
		"timestamp":  time.Now().Unix(),
	}

	context := ErrorContext{
		Category:    ErrorCategoryService,
		Severity:    ErrorSeverityMedium,
		Component:   "api",
		Operation:   "process_request",
		Recoverable: true,
		Metadata:    metadata,
	}

	structuredErr := NewStructuredError(err, context)

	assert.Equal(t, metadata, structuredErr.Context.Metadata)
	assert.Equal(t, "user123", structuredErr.Context.Metadata["user_id"])
	assert.Equal(t, "req456", structuredErr.Context.Metadata["request_id"])
}

func TestCaptureStackTrace(t *testing.T) {
	stack := captureStackTrace()
	
	assert.NotEmpty(t, stack)
	assert.Contains(t, stack, "TestCaptureStackTrace") // Should contain current function name
}

func TestContainsFunction(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "connection refused",
			substr:   "connection refused",
			expected: true,
		},
		{
			name:     "substring at beginning",
			s:        "connection timeout error",
			substr:   "connection",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "network connection",
			substr:   "connection",
			expected: true,
		},
		{
			name:     "substring in middle",
			s:        "tcp connection failed",
			substr:   "connection",
			expected: true,
		},
		{
			name:     "no match",
			s:        "something else",
			substr:   "connection",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "any string",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkClassifyError(b *testing.B) {
	err := errors.New("connection refused by remote host")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClassifyError(err)
	}
}

func BenchmarkNewStructuredError(b *testing.B) {
	err := errors.New("test error")
	context := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewStructuredError(err, context)
	}
}