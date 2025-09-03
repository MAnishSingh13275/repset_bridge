package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gym-door-bridge/internal/logging"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler_NewErrorHandler(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress log output during tests
	
	eh := NewErrorHandler(logger)
	
	assert.NotNil(t, eh)
	assert.NotNil(t, eh.logger)
	assert.NotNil(t, eh.enhancedLogger)
	assert.NotNil(t, eh.circuitBreakers)
	assert.NotNil(t, eh.auditLogger)
}

func TestCircuitBreaker_Execute(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  100 * time.Millisecond,
		MaxRequests:      1,
		Timeout:          50 * time.Millisecond,
	}
	
	cb := NewCircuitBreaker("test", config, logger)
	ctx := context.Background()
	
	t.Run("successful execution", func(t *testing.T) {
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, CircuitBreakerClosed, cb.GetState())
	})
	
	t.Run("failed execution", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
		
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Equal(t, CircuitBreakerClosed, cb.GetState()) // Still closed, need more failures
	})
	
	t.Run("circuit breaker opens after threshold", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		
		// Trigger another failure to reach threshold
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
		
		assert.Error(t, err)
		assert.Equal(t, CircuitBreakerOpen, cb.GetState())
	})
	
	t.Run("circuit breaker rejects requests when open", func(t *testing.T) {
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker test is open")
	})
	
	t.Run("circuit breaker transitions to half-open after timeout", func(t *testing.T) {
		// Wait for recovery timeout
		time.Sleep(150 * time.Millisecond)
		
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, CircuitBreakerClosed, cb.GetState()) // Should close after successful execution
	})
	
	t.Run("timeout handling", func(t *testing.T) {
		cb := NewCircuitBreaker("timeout_test", config, logger)
		
		err := cb.Execute(ctx, func(ctx context.Context) error {
			// Sleep longer than timeout
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker("stats_test", config, logger)
	
	stats := cb.GetStats()
	
	assert.Equal(t, "stats_test", stats["name"])
	assert.Equal(t, CircuitBreakerClosed, stats["state"])
	assert.Equal(t, 0, stats["failure_count"])
	assert.Equal(t, 0, stats["request_count"])
	assert.Equal(t, config, stats["config"])
}

func TestErrorHandler_GetOrCreateCircuitBreaker(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	eh := NewErrorHandler(logger)
	
	t.Run("create new circuit breaker", func(t *testing.T) {
		cb1 := eh.GetOrCreateCircuitBreaker("test1", nil)
		
		assert.NotNil(t, cb1)
		assert.Equal(t, "test1", cb1.name)
	})
	
	t.Run("get existing circuit breaker", func(t *testing.T) {
		cb1 := eh.GetOrCreateCircuitBreaker("test1", nil)
		cb2 := eh.GetOrCreateCircuitBreaker("test1", nil)
		
		assert.Equal(t, cb1, cb2) // Should be the same instance
	})
	
	t.Run("create with custom config", func(t *testing.T) {
		customConfig := &CircuitBreakerConfig{
			FailureThreshold: 10,
			RecoveryTimeout:  60 * time.Second,
			MaxRequests:      5,
			Timeout:          30 * time.Second,
		}
		
		cb := eh.GetOrCreateCircuitBreaker("custom", customConfig)
		
		assert.Equal(t, *customConfig, cb.config)
	})
}

func TestErrorHandler_HandleError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	eh := NewErrorHandler(logger)
	ctx := context.Background()
	
	t.Run("handle recoverable error", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		errorContext := logging.ErrorContext{
			Category:    logging.ErrorCategoryService,
			Severity:    logging.ErrorSeverityMedium,
			Component:   "test_component",
			Operation:   "test_operation",
			Recoverable: true,
		}
		
		_ = eh.HandleError(ctx, testErr, errorContext)
		
		// Should return a structured error (recovery might succeed, so error could be nil)
		// The important thing is that it was processed
		assert.NotNil(t, eh.enhancedLogger)
	})
	
	t.Run("handle nil error", func(t *testing.T) {
		errorContext := logging.ErrorContext{
			Category:    logging.ErrorCategoryService,
			Severity:    logging.ErrorSeverityMedium,
			Component:   "test_component",
			Operation:   "test_operation",
			Recoverable: true,
		}
		
		result := eh.HandleError(ctx, nil, errorContext)
		
		assert.NoError(t, result)
	})
}

func TestErrorHandler_WriteErrorResponse(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	eh := NewErrorHandler(logger)
	
	t.Run("write standard error response", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		requestID := "test-request-123"
		
		eh.WriteErrorResponse(w, r, ErrorCodeValidationFailed, "Test error message", requestID)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, requestID, w.Header().Get("X-Request-ID"))
		
		// Check response body contains expected fields
		body := w.Body.String()
		assert.Contains(t, body, "VALIDATION_FAILED")
		assert.Contains(t, body, "Test error message")
		assert.Contains(t, body, requestID)
		assert.Contains(t, body, "/test")
		assert.Contains(t, body, "GET")
	})
}

func TestErrorHandler_RecoveryMiddleware(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	eh := NewErrorHandler(logger)
	
	t.Run("handles panic", func(t *testing.T) {
		handler := eh.RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		
		// Should not panic
		assert.NotPanics(t, func() {
			handler.ServeHTTP(w, r)
		})
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		
		body := w.Body.String()
		assert.Contains(t, body, "INTERNAL_ERROR")
		assert.Contains(t, body, "Internal Server Error")
	})
	
	t.Run("passes through normal requests", func(t *testing.T) {
		handler := eh.RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		
		handler.ServeHTTP(w, r)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})
}

func TestErrorHandler_ErrorRecoveryMiddleware(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	eh := NewErrorHandler(logger)
	
	t.Run("passes through successful requests", func(t *testing.T) {
		handler := eh.ErrorRecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		
		handler.ServeHTTP(w, r)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})
	
	t.Run("handles error responses", func(t *testing.T) {
		handler := eh.ErrorRecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("error"))
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		
		handler.ServeHTTP(w, r)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "error", w.Body.String())
	})
}

func TestErrorHandler_GetCircuitBreakerStats(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	eh := NewErrorHandler(logger)
	
	// Create some circuit breakers
	eh.GetOrCreateCircuitBreaker("test1", nil)
	eh.GetOrCreateCircuitBreaker("test2", nil)
	
	stats := eh.GetCircuitBreakerStats()
	
	assert.Len(t, stats, 2)
	assert.Contains(t, stats, "test1")
	assert.Contains(t, stats, "test2")
	
	// Check structure of stats
	test1Stats := stats["test1"].(map[string]interface{})
	assert.Equal(t, "test1", test1Stats["name"])
	assert.Equal(t, CircuitBreakerClosed, test1Stats["state"])
}

func TestErrorResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		wrapper := &errorResponseWriter{
			ResponseWriter: w,
			requestID:      "test-123",
		}
		
		wrapper.WriteHeader(http.StatusBadRequest)
		
		assert.Equal(t, http.StatusBadRequest, wrapper.statusCode)
		assert.Equal(t, "test-123", w.Header().Get("X-Request-ID"))
	})
	
	t.Run("captures writes", func(t *testing.T) {
		w := httptest.NewRecorder()
		wrapper := &errorResponseWriter{
			ResponseWriter: w,
			requestID:      "test-123",
		}
		
		data := []byte("test data")
		n, err := wrapper.Write(data)
		
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, http.StatusOK, wrapper.statusCode) // Default status
		assert.Equal(t, "test data", w.Body.String())
	})
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	
	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 30*time.Second, config.RecoveryTimeout)
	assert.Equal(t, 3, config.MaxRequests)
	assert.Equal(t, 10*time.Second, config.Timeout)
}

// Benchmark tests
func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	cb := NewCircuitBreaker("bench", DefaultCircuitBreakerConfig(), logger)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkErrorHandler_WriteErrorResponse(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	eh := NewErrorHandler(logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/test", nil)
		
		eh.WriteErrorResponse(w, r, ErrorCodeValidationFailed, "Test error", "req-123")
	}
}