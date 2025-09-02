package logging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewRecoveryManager(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	assert.NotNil(t, rm)
	assert.Equal(t, logger, rm.logger)
	assert.NotNil(t, rm.recoveryMap)
	assert.NotNil(t, rm.activeRetries)

	// Check that default strategies are set up
	assert.Contains(t, rm.recoveryMap, ErrorCategoryHardware)
	assert.Contains(t, rm.recoveryMap, ErrorCategoryNetwork)
	assert.Contains(t, rm.recoveryMap, ErrorCategorySecurity)
	assert.Contains(t, rm.recoveryMap, ErrorCategoryStorage)
	assert.Contains(t, rm.recoveryMap, ErrorCategoryResource)
	assert.Contains(t, rm.recoveryMap, ErrorCategoryService)
	assert.Contains(t, rm.recoveryMap, ErrorCategoryConfig)
}

func TestRecoveryManagerRegisterAction(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyFailover,
		MaxAttempts: 3,
		Delay:       5 * time.Second,
		Description: "Custom failover action",
	}

	rm.RegisterRecoveryAction(ErrorCategoryHardware, customAction)

	registeredAction := rm.recoveryMap[ErrorCategoryHardware]
	assert.Equal(t, RecoveryStrategyFailover, registeredAction.Strategy)
	assert.Equal(t, 3, registeredAction.MaxAttempts)
	assert.Equal(t, 5*time.Second, registeredAction.Delay)
}

func TestAttemptRecoveryNonRecoverable(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	err := errors.New("test error")
	errContext := ErrorContext{
		Category:    ErrorCategoryHardware,
		Severity:    ErrorSeverityHigh,
		Component:   "adapter",
		Operation:   "scan",
		Recoverable: false, // Not recoverable
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.Equal(t, structuredErr, result)
}

func TestAttemptRecoveryUnknownCategory(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	err := errors.New("test error")
	errContext := ErrorContext{
		Category:    ErrorCategoryUnknown,
		Severity:    ErrorSeverityMedium,
		Component:   "unknown",
		Operation:   "unknown",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.Equal(t, structuredErr, result)
}

func TestAttemptRecoveryRetryStrategy(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	// Set up a custom retry action with a test function
	actionCalled := false
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 2,
		Delay:       10 * time.Millisecond, // Short delay for testing
		Action: func(ctx context.Context) error {
			actionCalled = true
			return nil // Success
		},
		Description: "Test retry action",
	}

	rm.RegisterRecoveryAction(ErrorCategoryNetwork, customAction)

	err := errors.New("network error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.NoError(t, result)
	assert.True(t, actionCalled)

	// Check that retry state was cleaned up
	state := rm.GetRetryState(ErrorCategoryNetwork, "client", "request")
	assert.Nil(t, state)
}

func TestAttemptRecoveryRetryExhausted(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	// Set up a custom retry action that always fails
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 2,
		Delay:       1 * time.Millisecond, // Very short delay for testing
		Action: func(ctx context.Context) error {
			return errors.New("action failed")
		},
		Description: "Test failing retry action",
	}

	rm.RegisterRecoveryAction(ErrorCategoryNetwork, customAction)

	err := errors.New("network error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	// First attempt
	result1 := rm.AttemptRecovery(ctx, structuredErr)
	assert.Error(t, result1)

	// Second attempt
	result2 := rm.AttemptRecovery(ctx, structuredErr)
	assert.Error(t, result2)

	// Third attempt should be exhausted
	result3 := rm.AttemptRecovery(ctx, structuredErr)
	assert.Error(t, result3)
	assert.Contains(t, result3.Error(), "retry attempts exhausted")
}

func TestAttemptRecoveryRestartStrategy(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	actionCalled := false
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyRestart,
		MaxAttempts: 1,
		Delay:       0,
		Action: func(ctx context.Context) error {
			actionCalled = true
			return nil
		},
		Description: "Test restart action",
	}

	rm.RegisterRecoveryAction(ErrorCategoryService, customAction)

	err := errors.New("service error")
	errContext := ErrorContext{
		Category:    ErrorCategoryService,
		Severity:    ErrorSeverityHigh,
		Component:   "heartbeat",
		Operation:   "start",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.NoError(t, result)
	assert.True(t, actionCalled)
}

func TestAttemptRecoveryDegradeStrategy(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	actionCalled := false
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyDegrade,
		MaxAttempts: 1,
		Delay:       0,
		Action: func(ctx context.Context) error {
			actionCalled = true
			return nil
		},
		Description: "Test degrade action",
	}

	rm.RegisterRecoveryAction(ErrorCategoryResource, customAction)

	err := errors.New("resource error")
	errContext := ErrorContext{
		Category:    ErrorCategoryResource,
		Severity:    ErrorSeverityMedium,
		Component:   "tier",
		Operation:   "monitor",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.NoError(t, result)
	assert.True(t, actionCalled)
}

func TestAttemptRecoverySkipStrategy(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	err := errors.New("config error")
	errContext := ErrorContext{
		Category:    ErrorCategoryConfig,
		Severity:    ErrorSeverityLow,
		Component:   "config",
		Operation:   "parse",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.NoError(t, result) // Skip strategy should return no error
}

func TestAttemptRecoveryNoneStrategy(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	err := errors.New("security error")
	errContext := ErrorContext{
		Category:    ErrorCategorySecurity,
		Severity:    ErrorSeverityCritical,
		Component:   "auth",
		Operation:   "validate",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.Equal(t, structuredErr, result) // None strategy should return original error
}

func TestGetRetryState(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	// Initially no retry state
	state := rm.GetRetryState(ErrorCategoryNetwork, "client", "request")
	assert.Nil(t, state)

	// Create a retry state by attempting recovery
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 3,
		Delay:       1 * time.Millisecond,
		Action: func(ctx context.Context) error {
			return errors.New("still failing")
		},
	}

	rm.RegisterRecoveryAction(ErrorCategoryNetwork, customAction)

	err := errors.New("network error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	// Attempt recovery to create retry state
	rm.AttemptRecovery(ctx, structuredErr)

	// Now we should have retry state
	state = rm.GetRetryState(ErrorCategoryNetwork, "client", "request")
	assert.NotNil(t, state)
	assert.Equal(t, 1, state.attempts)
	assert.Equal(t, 3, state.maxAttempts)
}

func TestClearRetryState(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	// Create retry state first
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 3,
		Delay:       1 * time.Millisecond,
		Action: func(ctx context.Context) error {
			return errors.New("still failing")
		},
	}

	rm.RegisterRecoveryAction(ErrorCategoryNetwork, customAction)

	err := errors.New("network error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	// Attempt recovery to create retry state
	rm.AttemptRecovery(ctx, structuredErr)

	// Verify state exists
	state := rm.GetRetryState(ErrorCategoryNetwork, "client", "request")
	assert.NotNil(t, state)

	// Clear the state
	rm.ClearRetryState(ErrorCategoryNetwork, "client", "request")

	// Verify state is cleared
	state = rm.GetRetryState(ErrorCategoryNetwork, "client", "request")
	assert.Nil(t, state)
}

func TestGetActiveRetries(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	// Initially no active retries
	assert.Equal(t, 0, rm.GetActiveRetries())

	// Create some retry states
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 3,
		Delay:       1 * time.Millisecond,
		Action: func(ctx context.Context) error {
			return errors.New("still failing")
		},
	}

	rm.RegisterRecoveryAction(ErrorCategoryNetwork, customAction)
	rm.RegisterRecoveryAction(ErrorCategoryHardware, customAction)

	ctx := context.Background()

	// Create first retry state
	err1 := errors.New("network error")
	context1 := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}
	structuredErr1 := NewStructuredError(err1, context1)
	rm.AttemptRecovery(ctx, structuredErr1)

	assert.Equal(t, 1, rm.GetActiveRetries())

	// Create second retry state
	err2 := errors.New("hardware error")
	context2 := ErrorContext{
		Category:    ErrorCategoryHardware,
		Severity:    ErrorSeverityHigh,
		Component:   "adapter",
		Operation:   "scan",
		Recoverable: true,
	}
	structuredErr2 := NewStructuredError(err2, context2)
	rm.AttemptRecovery(ctx, structuredErr2)

	assert.Equal(t, 2, rm.GetActiveRetries())
}

func TestRecoveryWithContextCancellation(t *testing.T) {
	logger := logrus.New()
	rm := NewRecoveryManager(logger)

	// Set up a retry action with a longer delay
	customAction := RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 3,
		Delay:       100 * time.Millisecond,
		Action: func(ctx context.Context) error {
			return errors.New("still failing")
		},
	}

	rm.RegisterRecoveryAction(ErrorCategoryNetwork, customAction)

	err := errors.New("network error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel the context immediately
	cancel()

	result := rm.AttemptRecovery(ctx, structuredErr)
	assert.Equal(t, context.Canceled, result)
}

func TestRecoveryStrategyString(t *testing.T) {
	tests := []struct {
		strategy RecoveryStrategy
		expected string
	}{
		{RecoveryStrategyRetry, "retry"},
		{RecoveryStrategyRestart, "restart"},
		{RecoveryStrategyDegrade, "degrade"},
		{RecoveryStrategyFailover, "failover"},
		{RecoveryStrategySkip, "skip"},
		{RecoveryStrategyNone, "none"},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.strategy))
		})
	}
}

func BenchmarkAttemptRecovery(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce logging overhead
	rm := NewRecoveryManager(logger)

	err := errors.New("test error")
	errContext := ErrorContext{
		Category:    ErrorCategoryNetwork,
		Severity:    ErrorSeverityMedium,
		Component:   "client",
		Operation:   "request",
		Recoverable: true,
	}

	structuredErr := NewStructuredError(err, errContext)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.AttemptRecovery(ctx, structuredErr)
		// Clear retry state to reset for next iteration
		rm.ClearRetryState(ErrorCategoryNetwork, "client", "request")
	}
}