package logging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RecoveryStrategy defines different recovery strategies for errors
type RecoveryStrategy string

const (
	// Retry the operation with exponential backoff
	RecoveryStrategyRetry RecoveryStrategy = "retry"
	// Restart the component/service
	RecoveryStrategyRestart RecoveryStrategy = "restart"
	// Degrade functionality gracefully
	RecoveryStrategyDegrade RecoveryStrategy = "degrade"
	// Failover to backup/alternative
	RecoveryStrategyFailover RecoveryStrategy = "failover"
	// Skip/ignore the error and continue
	RecoveryStrategySkip RecoveryStrategy = "skip"
	// No recovery possible
	RecoveryStrategyNone RecoveryStrategy = "none"
)

// RecoveryAction represents a recovery action to be taken
type RecoveryAction struct {
	Strategy    RecoveryStrategy `json:"strategy"`
	MaxAttempts int              `json:"max_attempts"`
	Delay       time.Duration    `json:"delay"`
	Action      func(ctx context.Context) error `json:"-"`
	Description string           `json:"description"`
}

// RecoveryManager manages error recovery operations
type RecoveryManager struct {
	logger        *logrus.Logger
	recoveryMap   map[ErrorCategory]RecoveryAction
	activeRetries map[string]*retryState
	mutex         sync.RWMutex
}

// retryState tracks the state of ongoing retry operations
type retryState struct {
	attempts    int
	lastAttempt time.Time
	maxAttempts int
	delay       time.Duration
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(logger *logrus.Logger) *RecoveryManager {
	rm := &RecoveryManager{
		logger:        logger,
		recoveryMap:   make(map[ErrorCategory]RecoveryAction),
		activeRetries: make(map[string]*retryState),
	}

	// Set up default recovery strategies
	rm.setupDefaultStrategies()
	
	return rm
}

// setupDefaultStrategies configures default recovery strategies for different error categories
func (rm *RecoveryManager) setupDefaultStrategies() {
	// Hardware errors: retry with increasing delays, then degrade
	rm.recoveryMap[ErrorCategoryHardware] = RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 3,
		Delay:       5 * time.Second,
		Description: "Retry hardware operation with exponential backoff",
	}

	// Network errors: retry with exponential backoff
	rm.recoveryMap[ErrorCategoryNetwork] = RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 5,
		Delay:       2 * time.Second,
		Description: "Retry network operation with exponential backoff",
	}

	// Security errors: no automatic recovery (requires manual intervention)
	rm.recoveryMap[ErrorCategorySecurity] = RecoveryAction{
		Strategy:    RecoveryStrategyNone,
		MaxAttempts: 0,
		Delay:       0,
		Description: "Security errors require manual intervention",
	}

	// Storage errors: retry briefly, then degrade
	rm.recoveryMap[ErrorCategoryStorage] = RecoveryAction{
		Strategy:    RecoveryStrategyRetry,
		MaxAttempts: 2,
		Delay:       1 * time.Second,
		Description: "Retry storage operation briefly",
	}

	// Resource errors: degrade performance tier
	rm.recoveryMap[ErrorCategoryResource] = RecoveryAction{
		Strategy:    RecoveryStrategyDegrade,
		MaxAttempts: 1,
		Delay:       0,
		Description: "Degrade performance tier to reduce resource usage",
	}

	// Service errors: restart component
	rm.recoveryMap[ErrorCategoryService] = RecoveryAction{
		Strategy:    RecoveryStrategyRestart,
		MaxAttempts: 2,
		Delay:       10 * time.Second,
		Description: "Restart failed service component",
	}

	// Configuration errors: skip and continue with defaults
	rm.recoveryMap[ErrorCategoryConfig] = RecoveryAction{
		Strategy:    RecoveryStrategySkip,
		MaxAttempts: 1,
		Delay:       0,
		Description: "Skip invalid configuration and use defaults",
	}
}

// RegisterRecoveryAction registers a custom recovery action for an error category
func (rm *RecoveryManager) RegisterRecoveryAction(category ErrorCategory, action RecoveryAction) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	rm.recoveryMap[category] = action
	rm.logger.WithFields(logrus.Fields{
		"category": category,
		"strategy": action.Strategy,
		"max_attempts": action.MaxAttempts,
	}).Info("Registered custom recovery action")
}

// AttemptRecovery attempts to recover from an error using the appropriate strategy
func (rm *RecoveryManager) AttemptRecovery(ctx context.Context, structuredErr *StructuredError) error {
	if structuredErr == nil {
		return fmt.Errorf("structured error is nil")
	}

	// Check if error is recoverable
	if !structuredErr.Context.Recoverable {
		rm.logger.WithField("error", structuredErr.Error()).Info("Error marked as non-recoverable, skipping recovery")
		return structuredErr
	}

	// Get recovery action for this error category
	rm.mutex.RLock()
	recoveryAction, exists := rm.recoveryMap[structuredErr.Context.Category]
	rm.mutex.RUnlock()

	if !exists {
		rm.logger.WithField("category", structuredErr.Context.Category).Warn("No recovery strategy defined for error category")
		return structuredErr
	}

	// Generate unique key for this recovery operation
	recoveryKey := fmt.Sprintf("%s:%s:%s", 
		structuredErr.Context.Category, 
		structuredErr.Context.Component, 
		structuredErr.Context.Operation)

	switch recoveryAction.Strategy {
	case RecoveryStrategyRetry:
		return rm.attemptRetry(ctx, recoveryKey, recoveryAction, structuredErr)
	case RecoveryStrategyRestart:
		return rm.attemptRestart(ctx, recoveryAction, structuredErr)
	case RecoveryStrategyDegrade:
		return rm.attemptDegrade(ctx, recoveryAction, structuredErr)
	case RecoveryStrategyFailover:
		return rm.attemptFailover(ctx, recoveryAction, structuredErr)
	case RecoveryStrategySkip:
		return rm.attemptSkip(ctx, recoveryAction, structuredErr)
	case RecoveryStrategyNone:
		rm.logger.WithField("error", structuredErr.Error()).Info("No recovery strategy available")
		return structuredErr
	default:
		rm.logger.WithField("strategy", recoveryAction.Strategy).Warn("Unknown recovery strategy")
		return structuredErr
	}
}

// attemptRetry implements retry recovery strategy
func (rm *RecoveryManager) attemptRetry(ctx context.Context, recoveryKey string, action RecoveryAction, structuredErr *StructuredError) error {
	rm.mutex.Lock()
	state, exists := rm.activeRetries[recoveryKey]
	if !exists {
		state = &retryState{
			attempts:    0,
			maxAttempts: action.MaxAttempts,
			delay:       action.Delay,
		}
		rm.activeRetries[recoveryKey] = state
	}
	rm.mutex.Unlock()

	// Check if we've exceeded max attempts
	if state.attempts >= state.maxAttempts {
		rm.mutex.Lock()
		delete(rm.activeRetries, recoveryKey)
		rm.mutex.Unlock()
		
		LogRecoveryAttempt(rm.logger, structuredErr, "retry_exhausted", false)
		return fmt.Errorf("retry attempts exhausted for %s: %w", recoveryKey, structuredErr)
	}

	// Calculate delay with exponential backoff
	delay := time.Duration(float64(state.delay) * float64(uint64(1) << uint(state.attempts)))
	if delay > 5*time.Minute {
		delay = 5 * time.Minute // Cap at 5 minutes
	}

	state.attempts++
	state.lastAttempt = time.Now()

	rm.logger.WithFields(logrus.Fields{
		"recovery_key": recoveryKey,
		"attempt":      state.attempts,
		"max_attempts": state.maxAttempts,
		"delay":        delay,
	}).Info("Attempting retry recovery")

	// Wait for delay
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	// If a custom action is provided, execute it
	if action.Action != nil {
		err := action.Action(ctx)
		if err == nil {
			// Success - clean up retry state
			rm.mutex.Lock()
			delete(rm.activeRetries, recoveryKey)
			rm.mutex.Unlock()
			
			LogRecoveryAttempt(rm.logger, structuredErr, "retry_success", true)
			return nil
		}
		
		// Action failed, will retry on next call
		LogRecoveryAttempt(rm.logger, structuredErr, "retry_failed", false)
		return err
	}

	// No custom action - just indicate retry is ready
	LogRecoveryAttempt(rm.logger, structuredErr, "retry_ready", true)
	return nil
}

// attemptRestart implements restart recovery strategy
func (rm *RecoveryManager) attemptRestart(ctx context.Context, action RecoveryAction, structuredErr *StructuredError) error {
	rm.logger.WithFields(logrus.Fields{
		"component": structuredErr.Context.Component,
		"operation": structuredErr.Context.Operation,
	}).Info("Attempting restart recovery")

	if action.Action != nil {
		err := action.Action(ctx)
		LogRecoveryAttempt(rm.logger, structuredErr, "restart", err == nil)
		return err
	}

	// No custom action - log that restart is needed
	LogRecoveryAttempt(rm.logger, structuredErr, "restart_needed", true)
	return nil
}

// attemptDegrade implements graceful degradation recovery strategy
func (rm *RecoveryManager) attemptDegrade(ctx context.Context, action RecoveryAction, structuredErr *StructuredError) error {
	rm.logger.WithFields(logrus.Fields{
		"component": structuredErr.Context.Component,
		"operation": structuredErr.Context.Operation,
	}).Info("Attempting graceful degradation recovery")

	if action.Action != nil {
		err := action.Action(ctx)
		LogRecoveryAttempt(rm.logger, structuredErr, "degrade", err == nil)
		return err
	}

	// No custom action - log that degradation is needed
	LogRecoveryAttempt(rm.logger, structuredErr, "degrade_needed", true)
	return nil
}

// attemptFailover implements failover recovery strategy
func (rm *RecoveryManager) attemptFailover(ctx context.Context, action RecoveryAction, structuredErr *StructuredError) error {
	rm.logger.WithFields(logrus.Fields{
		"component": structuredErr.Context.Component,
		"operation": structuredErr.Context.Operation,
	}).Info("Attempting failover recovery")

	if action.Action != nil {
		err := action.Action(ctx)
		LogRecoveryAttempt(rm.logger, structuredErr, "failover", err == nil)
		return err
	}

	// No custom action - log that failover is needed
	LogRecoveryAttempt(rm.logger, structuredErr, "failover_needed", true)
	return nil
}

// attemptSkip implements skip recovery strategy
func (rm *RecoveryManager) attemptSkip(ctx context.Context, action RecoveryAction, structuredErr *StructuredError) error {
	rm.logger.WithFields(logrus.Fields{
		"component": structuredErr.Context.Component,
		"operation": structuredErr.Context.Operation,
	}).Info("Skipping error and continuing")

	LogRecoveryAttempt(rm.logger, structuredErr, "skip", true)
	return nil // Skip the error
}

// GetRetryState returns the current retry state for a recovery key
func (rm *RecoveryManager) GetRetryState(category ErrorCategory, component, operation string) *retryState {
	recoveryKey := fmt.Sprintf("%s:%s:%s", category, component, operation)
	
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	if state, exists := rm.activeRetries[recoveryKey]; exists {
		// Return a copy to avoid race conditions
		return &retryState{
			attempts:    state.attempts,
			lastAttempt: state.lastAttempt,
			maxAttempts: state.maxAttempts,
			delay:       state.delay,
		}
	}
	
	return nil
}

// ClearRetryState clears the retry state for a specific recovery key
func (rm *RecoveryManager) ClearRetryState(category ErrorCategory, component, operation string) {
	recoveryKey := fmt.Sprintf("%s:%s:%s", category, component, operation)
	
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	delete(rm.activeRetries, recoveryKey)
}

// GetActiveRetries returns the number of active retry operations
func (rm *RecoveryManager) GetActiveRetries() int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	return len(rm.activeRetries)
}