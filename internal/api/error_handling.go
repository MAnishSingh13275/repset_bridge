package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"gym-door-bridge/internal/logging"

	"github.com/sirupsen/logrus"
)

// ErrorHandler provides comprehensive error handling capabilities
type ErrorHandler struct {
	logger          *logrus.Logger
	enhancedLogger  *logging.EnhancedLogger
	circuitBreakers map[string]*CircuitBreaker
	mutex           sync.RWMutex
	auditLogger     *AuditLogger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *logrus.Logger) *ErrorHandler {
	enhancedLogger := logging.NewEnhancedLogger(logger)
	
	return &ErrorHandler{
		logger:          logger,
		enhancedLogger:  enhancedLogger,
		circuitBreakers: make(map[string]*CircuitBreaker),
		auditLogger:     NewAuditLogger(logger),
	}
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	MaxRequests      int           `json:"max_requests"`
	Timeout          time.Duration `json:"timeout"`
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		MaxRequests:      3,
		Timeout:          10 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	config          CircuitBreakerConfig
	state           CircuitBreakerState
	failureCount    int
	requestCount    int
	lastFailureTime time.Time
	lastStateChange time.Time
	mutex           sync.RWMutex
	logger          *logrus.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig, logger *logrus.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		name:            name,
		config:          config,
		state:           CircuitBreakerClosed,
		lastStateChange: time.Now(),
		logger:          logger,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Check if circuit breaker allows the request
	if !cb.allowRequest() {
		cb.logger.WithFields(logrus.Fields{
			"circuit_breaker": cb.name,
			"state":          cb.state,
		}).Warn("Circuit breaker rejected request")
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, cb.config.Timeout)
	defer cancel()

	// Execute function with timeout
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in circuit breaker %s: %v", cb.name, r)
			}
		}()
		errChan <- fn(timeoutCtx)
	}()

	select {
	case err := <-errChan:
		cb.recordResult(err)
		return err
	case <-timeoutCtx.Done():
		err := fmt.Errorf("circuit breaker %s timeout", cb.name)
		cb.recordResult(err)
		return err
	}
}

// allowRequest checks if the circuit breaker allows a request
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if recovery timeout has passed
		if time.Since(cb.lastFailureTime) > cb.config.RecoveryTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.requestCount = 0
			cb.lastStateChange = time.Now()
			cb.logger.WithField("circuit_breaker", cb.name).Info("Circuit breaker transitioning to half-open")
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return cb.requestCount < cb.config.MaxRequests
	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == CircuitBreakerHalfOpen {
		cb.requestCount++
	}

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		// Check if we should open the circuit
		if cb.state == CircuitBreakerClosed && cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitBreakerOpen
			cb.lastStateChange = time.Now()
			cb.logger.WithFields(logrus.Fields{
				"circuit_breaker":  cb.name,
				"failure_count":    cb.failureCount,
				"failure_threshold": cb.config.FailureThreshold,
			}).Warn("Circuit breaker opened due to failures")
		} else if cb.state == CircuitBreakerHalfOpen {
			// Failed in half-open state, go back to open
			cb.state = CircuitBreakerOpen
			cb.lastStateChange = time.Now()
			cb.logger.WithField("circuit_breaker", cb.name).Warn("Circuit breaker reopened after half-open failure")
		}
	} else {
		// Success
		if cb.state == CircuitBreakerHalfOpen && cb.requestCount >= cb.config.MaxRequests {
			// Successful requests in half-open state, close the circuit
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
			cb.requestCount = 0
			cb.lastStateChange = time.Now()
			cb.logger.WithField("circuit_breaker", cb.name).Info("Circuit breaker closed after successful recovery")
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"name":              cb.name,
		"state":             cb.state,
		"failure_count":     cb.failureCount,
		"request_count":     cb.requestCount,
		"last_failure_time": cb.lastFailureTime,
		"last_state_change": cb.lastStateChange,
		"config":            cb.config,
	}
}

// GetOrCreateCircuitBreaker gets or creates a circuit breaker for a service
func (eh *ErrorHandler) GetOrCreateCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	eh.mutex.Lock()
	defer eh.mutex.Unlock()

	if cb, exists := eh.circuitBreakers[name]; exists {
		return cb
	}

	cbConfig := DefaultCircuitBreakerConfig()
	if config != nil {
		cbConfig = *config
	}

	cb := NewCircuitBreaker(name, cbConfig, eh.logger)
	eh.circuitBreakers[name] = cb

	eh.logger.WithFields(logrus.Fields{
		"circuit_breaker":     name,
		"failure_threshold":   cbConfig.FailureThreshold,
		"recovery_timeout":    cbConfig.RecoveryTimeout,
	}).Info("Created new circuit breaker")

	return cb
}

// HandleError handles errors with recovery and circuit breaker logic
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, errorContext logging.ErrorContext) error {
	if err == nil {
		return nil
	}

	// Log error with recovery attempt
	recoveredErr := eh.enhancedLogger.LogErrorWithRecovery(ctx, err, errorContext)

	// If error is related to a service, check circuit breaker
	if errorContext.Component != "" {
		cbName := fmt.Sprintf("%s_%s", errorContext.Component, errorContext.Operation)
		cb := eh.GetOrCreateCircuitBreaker(cbName, nil)
		
		// Record the error in circuit breaker
		cb.recordResult(err)
	}

	return recoveredErr
}

// WriteErrorResponse writes a standardized error response
func (eh *ErrorHandler) WriteErrorResponse(w http.ResponseWriter, r *http.Request, code ErrorCode, message string, requestID string) {
	// Create error response
	errorResponse := NewErrorResponse(code, message, r, requestID)

	// Log the error response
	eh.logger.WithFields(logrus.Fields{
		"error_code":  code,
		"message":     message,
		"status_code": errorResponse.Status,
		"path":        errorResponse.Path,
		"method":      errorResponse.Method,
		"request_id":  requestID,
		"client_ip":   getClientIP(r),
	}).Error("API error response")

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(errorResponse.Status)

	// Write JSON response
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		eh.logger.WithError(err).Error("Failed to encode error response")
		// Fallback to plain text
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// RecoveryMiddleware provides panic recovery with structured logging
func (eh *ErrorHandler) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Generate request ID for tracking
				requestID := generateRequestID()

				// Log panic with stack trace
				eh.logger.WithFields(logrus.Fields{
					"error":      err,
					"stack":      string(debug.Stack()),
					"path":       r.URL.Path,
					"method":     r.Method,
					"request_id": requestID,
					"client_ip":  getClientIP(r),
					"user_agent": r.UserAgent(),
				}).Error("Panic recovered in HTTP handler")

				// Create error context for structured error handling
				errorContext := logging.ErrorContext{
					Category:    logging.ErrorCategoryService,
					Severity:    logging.ErrorSeverityCritical,
					Component:   "api_server",
					Operation:   "request_handling",
					Recoverable: false,
					Metadata: map[string]interface{}{
						"panic_value": fmt.Sprintf("%v", err),
						"path":        r.URL.Path,
						"method":      r.Method,
					},
				}

				// Handle the panic as a structured error
				panicErr := fmt.Errorf("panic in HTTP handler: %v", err)
				eh.HandleError(r.Context(), panicErr, errorContext)

				// Return error response
				eh.WriteErrorResponse(w, r, ErrorCodeInternalError, "Internal Server Error", requestID)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// ErrorRecoveryMiddleware provides error recovery with circuit breaker protection
func (eh *ErrorHandler) ErrorRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		
		// Add request ID to context
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		// Execute request with circuit breaker protection
		cbName := fmt.Sprintf("api_%s", r.URL.Path)
		cb := eh.GetOrCreateCircuitBreaker(cbName, nil)

		err := cb.Execute(ctx, func(ctx context.Context) error {
			// Create a response writer wrapper to capture errors
			wrapper := &errorResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				requestID:      requestID,
			}

			// Execute the next handler
			next.ServeHTTP(wrapper, r)

			// Check if an error status was written
			if wrapper.statusCode >= 400 {
				return fmt.Errorf("HTTP error %d", wrapper.statusCode)
			}

			return nil
		})

		if err != nil {
			// Circuit breaker rejected the request or handler failed
			if cb.GetState() == CircuitBreakerOpen {
				eh.WriteErrorResponse(w, r, ErrorCodeCircuitBreakerOpen, "Service temporarily unavailable", requestID)
			} else {
				// Handler error was already written by the wrapper
				eh.logger.WithFields(logrus.Fields{
					"error":      err,
					"path":       r.URL.Path,
					"method":     r.Method,
					"request_id": requestID,
				}).Error("Request failed in error recovery middleware")
			}
		}
	})
}

// GetCircuitBreakerStats returns statistics for all circuit breakers
func (eh *ErrorHandler) GetCircuitBreakerStats() map[string]interface{} {
	eh.mutex.RLock()
	defer eh.mutex.RUnlock()

	stats := make(map[string]interface{})
	for name, cb := range eh.circuitBreakers {
		stats[name] = cb.GetStats()
	}

	return stats
}

// errorResponseWriter wraps http.ResponseWriter to capture status codes
type errorResponseWriter struct {
	http.ResponseWriter
	statusCode int
	requestID  string
}

func (w *errorResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.Header().Set("X-Request-ID", w.requestID)
	w.ResponseWriter.WriteHeader(code)
}

func (w *errorResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}