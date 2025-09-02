package logging

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewEnhancedLogger(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	assert.NotNil(t, enhanced)
	assert.Equal(t, baseLogger, enhanced.Logger)
	assert.NotNil(t, enhanced.recoveryManager)
	assert.NotNil(t, enhanced.degradationManager)
	assert.NotNil(t, enhanced.errorStats)
	assert.NotNil(t, enhanced.errorStats.ErrorsByCategory)
	assert.NotNil(t, enhanced.errorStats.ErrorsBySeverity)
}

func TestLogErrorWithRecovery(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	ctx := context.Background()
	err := errors.New("test error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: false, // Not recoverable
	}

	result := enhanced.LogErrorWithRecovery(ctx, err, errContext)
	assert.Error(t, result)

	// Check that statistics were updated
	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(1), stats.TotalErrors)
	assert.Equal(t, int64(1), stats.ErrorsByCategory[ErrorCategoryNetwork])
	assert.Equal(t, int64(1), stats.ErrorsBySeverity[ErrorSeverityMedium])
	assert.Equal(t, "test error", stats.LastErrorMessage)
	assert.False(t, stats.LastError.IsZero())
}

func TestLogErrorWithRecoveryNilError(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	ctx := context.Background()
	errContext := ErrorContext{
		Category:  ErrorCategoryNetwork,
		Severity:  ErrorSeverityMedium,
		Component: "client",
	}

	result := enhanced.LogErrorWithRecovery(ctx, nil, errContext)
	assert.NoError(t, result)

	// Statistics should not be updated for nil error
	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(0), stats.TotalErrors)
}

func TestLogErrorWithRecoveryRecoverable(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	// Register a custom recovery action that succeeds
	recoveryAction := RecoveryAction{
		Strategy:    RecoveryStrategySkip,
		MaxAttempts: 1,
		Delay:       0,
		Description: "Test skip recovery",
	}
	enhanced.recoveryManager.RegisterRecoveryAction(ErrorCategoryNetwork, recoveryAction)

	ctx := context.Background()
	err := errors.New("recoverable error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true, // Recoverable
	}

	result := enhanced.LogErrorWithRecovery(ctx, err, errContext)
	assert.NoError(t, result) // Should be recovered

	// Check that recovery statistics were updated
	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(1), stats.RecoveryAttempts)
	assert.Equal(t, int64(1), stats.RecoverySuccesses)
}

func TestLogHardwareErrorWithRecovery(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	ctx := context.Background()
	err := errors.New("hardware failure")

	result := enhanced.LogHardwareErrorWithRecovery(ctx, err, "fingerprint-reader", "scan")
	assert.Error(t, result) // Default hardware recovery should fail without custom action

	// Check statistics
	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(1), stats.TotalErrors)
	assert.Equal(t, int64(1), stats.ErrorsByCategory[ErrorCategoryHardware])
	assert.Equal(t, int64(1), stats.ErrorsBySeverity[ErrorSeverityHigh])
}

func TestLogNetworkErrorWithRecovery(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	ctx := context.Background()
	err := errors.New("connection timeout")

	// Test with low retry count (should be medium severity)
	result := enhanced.LogNetworkErrorWithRecovery(ctx, err, "submit_events", 2)
	assert.Error(t, result)

	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(1), stats.ErrorsBySeverity[ErrorSeverityMedium])

	// Test with high retry count (should be high severity)
	result = enhanced.LogNetworkErrorWithRecovery(ctx, err, "submit_events", 5)
	assert.Error(t, result)

	stats = enhanced.GetErrorStatistics()
	assert.Equal(t, int64(1), stats.ErrorsBySeverity[ErrorSeverityHigh])
}

func TestLogResourceConstraint(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	ctx := context.Background()

	// Test normal resource usage (should not degrade)
	err := enhanced.LogResourceConstraint(ctx, "memory", 50.0, 100.0)
	assert.NoError(t, err)
	assert.Equal(t, DegradationNone, enhanced.degradationManager.GetCurrentLevel())

	// Test high resource usage (should degrade)
	err = enhanced.LogResourceConstraint(ctx, "memory", 85.0, 100.0)
	assert.NoError(t, err)
	assert.Equal(t, DegradationModerate, enhanced.degradationManager.GetCurrentLevel())

	// Test critical resource usage (should degrade further)
	err = enhanced.LogResourceConstraint(ctx, "memory", 98.0, 100.0)
	assert.NoError(t, err)
	assert.Equal(t, DegradationCritical, enhanced.degradationManager.GetCurrentLevel())
}

func TestGetRecoveryManager(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	rm := enhanced.GetRecoveryManager()
	assert.NotNil(t, rm)
	assert.Equal(t, enhanced.recoveryManager, rm)
}

func TestGetDegradationManager(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	dm := enhanced.GetDegradationManager()
	assert.NotNil(t, dm)
	assert.Equal(t, enhanced.degradationManager, dm)
}

func TestGetErrorStatistics(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	// Initially empty statistics
	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(0), stats.TotalErrors)
	assert.Equal(t, 0, len(stats.ErrorsByCategory))
	assert.Equal(t, 0, len(stats.ErrorsBySeverity))
	assert.True(t, stats.LastError.IsZero())
	assert.Empty(t, stats.LastErrorMessage)
	assert.Equal(t, int64(0), stats.RecoveryAttempts)
	assert.Equal(t, int64(0), stats.RecoverySuccesses)

	// Log some errors
	ctx := context.Background()
	err1 := errors.New("first error")
	context1 := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Recoverable: false,
	}
	enhanced.LogErrorWithRecovery(ctx, err1, context1)

	err2 := errors.New("second error")
	context2 := ErrorContext{
		Category:    ErrorCategoryHardware,
		Severity:    ErrorSeverityHigh,
		Component:   "adapter",
		Recoverable: false,
	}
	enhanced.LogErrorWithRecovery(ctx, err2, context2)

	// Check updated statistics
	stats = enhanced.GetErrorStatistics()
	assert.Equal(t, int64(2), stats.TotalErrors)
	assert.Equal(t, int64(1), stats.ErrorsByCategory[ErrorCategoryNetwork])
	assert.Equal(t, int64(1), stats.ErrorsByCategory[ErrorCategoryHardware])
	assert.Equal(t, int64(1), stats.ErrorsBySeverity[ErrorSeverityMedium])
	assert.Equal(t, int64(1), stats.ErrorsBySeverity[ErrorSeverityHigh])
	assert.Equal(t, "second error", stats.LastErrorMessage)
	assert.False(t, stats.LastError.IsZero())
}

func TestResetErrorStatistics(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	// Log some errors first
	ctx := context.Background()
	err := errors.New("test error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Recoverable: false,
	}
	enhanced.LogErrorWithRecovery(ctx, err, errContext)

	// Verify statistics are not empty
	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(1), stats.TotalErrors)

	// Reset statistics
	enhanced.ResetErrorStatistics()

	// Verify statistics are reset
	stats = enhanced.GetErrorStatistics()
	assert.Equal(t, int64(0), stats.TotalErrors)
	assert.Equal(t, 0, len(stats.ErrorsByCategory))
	assert.Equal(t, 0, len(stats.ErrorsBySeverity))
	assert.True(t, stats.LastError.IsZero())
	assert.Empty(t, stats.LastErrorMessage)
	assert.Equal(t, int64(0), stats.RecoveryAttempts)
	assert.Equal(t, int64(0), stats.RecoverySuccesses)
}

func TestUpdateErrorStats(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	err := errors.New("test error")
	errContext := ErrorContext{
		Category:  ErrorCategoryStorage,
		Severity:  ErrorSeverityCritical,
		Component: "database",
	}
	structuredErr := NewStructuredError(err, errContext)

	// Update stats
	enhanced.updateErrorStats(structuredErr)

	// Check that stats were updated
	stats := enhanced.GetErrorStatistics()
	assert.Equal(t, int64(1), stats.TotalErrors)
	assert.Equal(t, int64(1), stats.ErrorsByCategory[ErrorCategoryStorage])
	assert.Equal(t, int64(1), stats.ErrorsBySeverity[ErrorSeverityCritical])
	assert.Equal(t, "test error", stats.LastErrorMessage)
	assert.Equal(t, structuredErr.Timestamp, stats.LastError)
}

func TestConcurrentErrorLogging(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	ctx := context.Background()
	numGoroutines := 10
	errorsPerGoroutine := 100

	done := make(chan bool, numGoroutines)

	// Launch multiple goroutines logging errors concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < errorsPerGoroutine; j++ {
				err := errors.New("concurrent error")
				errContext := ErrorContext{
					Category:    ErrorCategoryNetwork,
					Severity:    ErrorSeverityMedium,
					Component:   "client",
					Recoverable: false,
				}
				enhanced.LogErrorWithRecovery(ctx, err, errContext)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check final statistics
	stats := enhanced.GetErrorStatistics()
	expectedTotal := int64(numGoroutines * errorsPerGoroutine)
	assert.Equal(t, expectedTotal, stats.TotalErrors)
	assert.Equal(t, expectedTotal, stats.ErrorsByCategory[ErrorCategoryNetwork])
	assert.Equal(t, expectedTotal, stats.ErrorsBySeverity[ErrorSeverityMedium])
}

func TestErrorStatisticsCopy(t *testing.T) {
	baseLogger := logrus.New()
	enhanced := NewEnhancedLogger(baseLogger)

	// Log an error
	ctx := context.Background()
	err := errors.New("test error")
	context := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Recoverable: false,
	}
	enhanced.LogErrorWithRecovery(ctx, err, context)

	// Get statistics
	stats1 := enhanced.GetErrorStatistics()
	stats2 := enhanced.GetErrorStatistics()

	// Verify they are separate copies
	assert.NotSame(t, stats1, stats2)
	assert.NotSame(t, stats1.ErrorsByCategory, stats2.ErrorsByCategory)
	assert.NotSame(t, stats1.ErrorsBySeverity, stats2.ErrorsBySeverity)

	// But have the same values
	assert.Equal(t, stats1.TotalErrors, stats2.TotalErrors)
	assert.Equal(t, stats1.ErrorsByCategory[ErrorCategoryNetwork], stats2.ErrorsByCategory[ErrorCategoryNetwork])
	assert.Equal(t, stats1.ErrorsBySeverity[ErrorSeverityMedium], stats2.ErrorsBySeverity[ErrorSeverityMedium])
}

func BenchmarkLogErrorWithRecovery(b *testing.B) {
	baseLogger := logrus.New()
	baseLogger.SetLevel(logrus.ErrorLevel) // Reduce logging overhead
	enhanced := NewEnhancedLogger(baseLogger)

	ctx := context.Background()
	err := errors.New("benchmark error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhanced.LogErrorWithRecovery(ctx, err, errContext)
	}
}

func BenchmarkGetErrorStatistics(b *testing.B) {
	baseLogger := logrus.New()
	baseLogger.SetLevel(logrus.ErrorLevel) // Reduce logging overhead
	enhanced := NewEnhancedLogger(baseLogger)

	// Log some errors first
	ctx := context.Background()
	err := errors.New("benchmark error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Recoverable: false,
	}
	enhanced.LogErrorWithRecovery(ctx, err, errContext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhanced.GetErrorStatistics()
	}
}