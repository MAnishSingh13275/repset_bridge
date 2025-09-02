package logging

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Initialize sets up structured logging with the specified level
// Returns a basic logrus logger for backward compatibility
func Initialize(logLevel string) *logrus.Logger {
	logger := logrus.New()
	
	// Set log level
	level, err := logrus.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		level = logrus.InfoLevel
		logger.WithError(err).Warn("Invalid log level, defaulting to info")
	}
	logger.SetLevel(level)
	
	// Set JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	
	// Default to stdout
	logger.SetOutput(os.Stdout)
	
	// Add common fields
	logger = logger.WithFields(logrus.Fields{
		"service": "gym-door-bridge",
		"version": "1.0.0", // TODO: Get from build info
	}).Logger
	
	return logger
}

// InitializeEnhanced sets up enhanced structured logging with error handling and recovery capabilities
func InitializeEnhanced(logLevel string) *EnhancedLogger {
	baseLogger := Initialize(logLevel)
	return NewEnhancedLogger(baseLogger)
}

// SetupFileLogging configures logging to write to a file in addition to stdout
func SetupFileLogging(logger *logrus.Logger, logFile string) error {
	if logFile == "" {
		return nil
	}
	
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	
	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	
	// Set output to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, file)
	logger.SetOutput(multiWriter)
	
	logger.WithField("log_file", logFile).Info("File logging enabled")
	
	return nil
}

// NewContextLogger creates a logger with additional context fields
func NewContextLogger(logger *logrus.Logger, fields logrus.Fields) *logrus.Entry {
	return logger.WithFields(fields)
}

// NewAdapterLogger creates a logger specifically for hardware adapters
func NewAdapterLogger(logger *logrus.Logger, adapterName string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"component": "adapter",
		"adapter":   adapterName,
	})
}

// NewServiceLogger creates a logger for internal services
func NewServiceLogger(logger *logrus.Logger, serviceName string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{
		"component": "service",
		"service":   serviceName,
	})
}

// EnhancedLogger provides enhanced logging capabilities with error handling and recovery
type EnhancedLogger struct {
	*logrus.Logger
	recoveryManager    *RecoveryManager
	degradationManager *DegradationManager
	errorStats         *ErrorStatistics
	mutex              sync.RWMutex
}

// ErrorStatistics tracks error statistics for monitoring
type ErrorStatistics struct {
	TotalErrors       int64                        `json:"total_errors"`
	ErrorsByCategory  map[ErrorCategory]int64      `json:"errors_by_category"`
	ErrorsBySeverity  map[ErrorSeverity]int64      `json:"errors_by_severity"`
	LastError         time.Time                    `json:"last_error"`
	LastErrorMessage  string                       `json:"last_error_message"`
	RecoveryAttempts  int64                        `json:"recovery_attempts"`
	RecoverySuccesses int64                        `json:"recovery_successes"`
	mutex             sync.RWMutex
}

// NewEnhancedLogger creates a new enhanced logger with error handling capabilities
func NewEnhancedLogger(baseLogger *logrus.Logger) *EnhancedLogger {
	stats := &ErrorStatistics{
		ErrorsByCategory: make(map[ErrorCategory]int64),
		ErrorsBySeverity: make(map[ErrorSeverity]int64),
	}

	enhanced := &EnhancedLogger{
		Logger:             baseLogger,
		recoveryManager:    NewRecoveryManager(baseLogger),
		degradationManager: NewDegradationManager(baseLogger),
		errorStats:         stats,
	}

	return enhanced
}

// LogErrorWithRecovery logs an error and attempts recovery if possible
func (el *EnhancedLogger) LogErrorWithRecovery(ctx context.Context, err error, context ErrorContext) error {
	if err == nil {
		return nil
	}

	// Create structured error
	structuredErr := NewStructuredError(err, context)
	
	// Update statistics
	el.updateErrorStats(structuredErr)
	
	// Log the error
	LogStructuredError(el.Logger, structuredErr)
	
	// Attempt recovery if error is recoverable
	if context.Recoverable {
		el.errorStats.mutex.Lock()
		el.errorStats.RecoveryAttempts++
		el.errorStats.mutex.Unlock()
		
		recoveryErr := el.recoveryManager.AttemptRecovery(ctx, structuredErr)
		if recoveryErr == nil {
			el.errorStats.mutex.Lock()
			el.errorStats.RecoverySuccesses++
			el.errorStats.mutex.Unlock()
			
			el.Logger.WithFields(logrus.Fields{
				"original_error": err.Error(),
				"category":       context.Category,
				"component":      context.Component,
			}).Info("Error recovery successful")
			
			return nil
		}
		
		// Recovery failed, log it
		el.Logger.WithError(recoveryErr).WithFields(logrus.Fields{
			"original_error": err.Error(),
			"category":       context.Category,
			"component":      context.Component,
		}).Warn("Error recovery failed")
	}
	
	return structuredErr
}

// LogHardwareErrorWithRecovery logs hardware errors and attempts recovery
func (el *EnhancedLogger) LogHardwareErrorWithRecovery(ctx context.Context, err error, adapterName, operation string) error {
	context := ErrorContext{
		Category:    ErrorCategoryHardware,
		Severity:    ErrorSeverityHigh,
		Component:   "adapter",
		Operation:   operation,
		AdapterName: adapterName,
		Recoverable: true,
	}
	
	return el.LogErrorWithRecovery(ctx, err, context)
}

// LogNetworkErrorWithRecovery logs network errors and attempts recovery
func (el *EnhancedLogger) LogNetworkErrorWithRecovery(ctx context.Context, err error, operation string, retryCount int) error {
	severity := ErrorSeverityMedium
	if retryCount > 3 {
		severity = ErrorSeverityHigh
	}

	context := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    severity,
		Component:   "client",
		Operation:   operation,
		Recoverable: true,
		RetryCount:  retryCount,
	}
	
	return el.LogErrorWithRecovery(ctx, err, context)
}

// LogResourceConstraint logs resource constraint issues and triggers degradation
func (el *EnhancedLogger) LogResourceConstraint(ctx context.Context, resourceType string, usage, threshold float64) error {
	// Log resource constraint
	LogResourceError(el.Logger, 
		nil, // No underlying error for resource constraints
		resourceType, 
		usage, 
		threshold)
	
	// Trigger degradation based on resource usage
	return el.degradationManager.HandleResourceConstraint(ctx, resourceType, usage, threshold)
}

// GetRecoveryManager returns the recovery manager
func (el *EnhancedLogger) GetRecoveryManager() *RecoveryManager {
	return el.recoveryManager
}

// GetDegradationManager returns the degradation manager
func (el *EnhancedLogger) GetDegradationManager() *DegradationManager {
	return el.degradationManager
}

// GetErrorStatistics returns current error statistics
func (el *EnhancedLogger) GetErrorStatistics() *ErrorStatistics {
	el.errorStats.mutex.RLock()
	defer el.errorStats.mutex.RUnlock()
	
	// Return a copy to avoid race conditions
	stats := &ErrorStatistics{
		TotalErrors:       el.errorStats.TotalErrors,
		ErrorsByCategory:  make(map[ErrorCategory]int64),
		ErrorsBySeverity:  make(map[ErrorSeverity]int64),
		LastError:         el.errorStats.LastError,
		LastErrorMessage:  el.errorStats.LastErrorMessage,
		RecoveryAttempts:  el.errorStats.RecoveryAttempts,
		RecoverySuccesses: el.errorStats.RecoverySuccesses,
	}
	
	for k, v := range el.errorStats.ErrorsByCategory {
		stats.ErrorsByCategory[k] = v
	}
	for k, v := range el.errorStats.ErrorsBySeverity {
		stats.ErrorsBySeverity[k] = v
	}
	
	return stats
}

// updateErrorStats updates internal error statistics
func (el *EnhancedLogger) updateErrorStats(structuredErr *StructuredError) {
	el.errorStats.mutex.Lock()
	defer el.errorStats.mutex.Unlock()
	
	el.errorStats.TotalErrors++
	el.errorStats.ErrorsByCategory[structuredErr.Context.Category]++
	el.errorStats.ErrorsBySeverity[structuredErr.Context.Severity]++
	el.errorStats.LastError = structuredErr.Timestamp
	el.errorStats.LastErrorMessage = structuredErr.Error()
}

// ResetErrorStatistics resets error statistics (useful for testing)
func (el *EnhancedLogger) ResetErrorStatistics() {
	el.errorStats.mutex.Lock()
	defer el.errorStats.mutex.Unlock()
	
	el.errorStats.TotalErrors = 0
	el.errorStats.ErrorsByCategory = make(map[ErrorCategory]int64)
	el.errorStats.ErrorsBySeverity = make(map[ErrorSeverity]int64)
	el.errorStats.LastError = time.Time{}
	el.errorStats.LastErrorMessage = ""
	el.errorStats.RecoveryAttempts = 0
	el.errorStats.RecoverySuccesses = 0
}