package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewRequestLogger(t *testing.T) {
	logger := logrus.New()
	rl := NewRequestLogger(logger)
	
	assert.NotNil(t, rl)
	assert.Equal(t, logger, rl.logger)
	assert.NotNil(t, rl.auditLogger)
}

func TestNewLoggingResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	
	t.Run("create without body capture", func(t *testing.T) {
		lrw := NewLoggingResponseWriter(w, false)
		
		assert.NotNil(t, lrw)
		assert.Equal(t, w, lrw.ResponseWriter)
		assert.Equal(t, http.StatusOK, lrw.statusCode)
		assert.False(t, lrw.captureBody)
		assert.NotNil(t, lrw.body)
	})
	
	t.Run("create with body capture", func(t *testing.T) {
		lrw := NewLoggingResponseWriter(w, true)
		
		assert.True(t, lrw.captureBody)
	})
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	lrw := NewLoggingResponseWriter(w, false)
	
	lrw.WriteHeader(http.StatusBadRequest)
	
	assert.Equal(t, http.StatusBadRequest, lrw.statusCode)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	
	t.Run("write without body capture", func(t *testing.T) {
		lrw := NewLoggingResponseWriter(w, false)
		
		data := []byte("test response data")
		n, err := lrw.Write(data)
		
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, int64(len(data)), lrw.responseSize)
		assert.Equal(t, "test response data", w.Body.String())
		assert.Equal(t, 0, lrw.body.Len()) // Should not capture
	})
	
	t.Run("write with body capture", func(t *testing.T) {
		w := httptest.NewRecorder()
		lrw := NewLoggingResponseWriter(w, true)
		
		data := []byte("captured response")
		n, err := lrw.Write(data)
		
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, int64(len(data)), lrw.responseSize)
		assert.Equal(t, "captured response", lrw.body.String())
	})
	
	t.Run("write large body with capture limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		lrw := NewLoggingResponseWriter(w, true)
		
		// Write data that exceeds capture limit
		largeData := make([]byte, 1024*15) // 15KB, exceeds 10KB limit
		for i := range largeData {
			largeData[i] = 'A'
		}
		
		n, err := lrw.Write(largeData)
		
		assert.NoError(t, err)
		assert.Equal(t, len(largeData), n)
		assert.Equal(t, int64(len(largeData)), lrw.responseSize)
		assert.True(t, lrw.body.Len() <= 1024*10) // Should not exceed limit
	})
}

func TestRequestLogger_StructuredLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	rl := NewRequestLogger(logger)
	
	t.Run("log successful request", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/status", nil)
		r.Header.Set("User-Agent", "TestClient/1.0")
		r.RemoteAddr = "192.168.1.100:12345"
		
		handler.ServeHTTP(w, r)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "GET")
		assert.Contains(t, logOutput, "/api/v1/status")
		assert.Contains(t, logOutput, "200")
		assert.Contains(t, logOutput, "192.168.1.100")
		assert.Contains(t, logOutput, "TestClient/1.0")
		assert.Contains(t, logOutput, "request_id")
		assert.Contains(t, logOutput, "duration_ms")
		assert.Contains(t, logOutput, "response_size")
	})
	
	t.Run("log error request", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("error"))
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/v1/door/unlock", nil)
		
		handler.ServeHTTP(w, r)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "POST")
		assert.Contains(t, logOutput, "/api/v1/door/unlock")
		assert.Contains(t, logOutput, "400")
		assert.Contains(t, logOutput, "level\":\"warning") // Should be warning level for 4xx
	})
	
	t.Run("log server error request", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/config", nil)
		
		handler.ServeHTTP(w, r)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "500")
		assert.Contains(t, logOutput, "level\":\"error") // Should be error level for 5xx
	})
	
	t.Run("log request with query parameters", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/events?startTime=2023-01-01&limit=100", nil)
		
		handler.ServeHTTP(w, r)
		
		logOutput := buf.String()
		// Query parameters might be URL encoded in JSON
		assert.Contains(t, logOutput, "startTime=2023-01-01")
	})
	
	t.Run("log request with referer", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/status", nil)
		r.Header.Set("Referer", "https://example.com/dashboard")
		
		handler.ServeHTTP(w, r)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "https://example.com/dashboard")
	})
}

func TestRequestLogger_LogAuditEvents(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	rl := NewRequestLogger(logger)
	
	t.Run("log audit event for sensitive operation", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/v1/door/unlock", nil)
		
		handler.ServeHTTP(w, r)
		
		logOutput := buf.String()
		// Should contain both request log and audit log entries
		assert.Contains(t, logOutput, "POST")
		assert.Contains(t, logOutput, "/api/v1/door/unlock")
		// Check for audit event
		assert.Contains(t, logOutput, "privileged_action")
	})
	
	t.Run("log audit event for data access", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/events", nil)
		
		handler.ServeHTTP(w, r)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "data_access")
	})
	
	t.Run("log audit event for configuration change", func(t *testing.T) {
		buf.Reset()
		
		handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		
		w := httptest.NewRecorder()
		r := httptest.NewRequest("PUT", "/api/v1/config", nil)
		
		handler.ServeHTTP(w, r)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "data_modification")
		assert.Contains(t, logOutput, "privileged_action")
	})
}

func TestShouldCaptureResponseBody(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/config", true},
		{"/api/v1/door/unlock", true},
		{"/api/v1/adapters/test", true},
		{"/api/v1/status", false},
		{"/api/v1/health", false},
		{"/api/v1/metrics", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil)
			result := shouldCaptureResponseBody(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldCaptureHeaders(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/config", true},
		{"/api/v1/door/unlock", true},
		{"/api/v1/adapters/test", true},
		{"/api/v1/events", true},
		{"/api/v1/health", false},
		{"/api/v1/metrics", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil)
			result := shouldCaptureHeaders(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAuthMethod(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "API key in X-API-Key header",
			headers:  map[string]string{"X-API-Key": "test-key"},
			expected: "api_key",
		},
		{
			name:     "HMAC signature",
			headers:  map[string]string{"X-Signature": "test-signature"},
			expected: "hmac",
		},
		{
			name:     "JWT token",
			headers:  map[string]string{"Authorization": "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9"},
			expected: "jwt",
		},
		{
			name:     "No auth method",
			headers:  map[string]string{},
			expected: "",
		},
		{
			name:     "Non-bearer authorization",
			headers:  map[string]string{"Authorization": "Basic dGVzdDp0ZXN0"},
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				r.Header.Set(key, value)
			}
			
			result := getAuthMethod(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "extract from JSON response",
			body:     `{"error": true, "message": "Validation failed", "code": "VALIDATION_ERROR"}`,
			expected: "Validation failed",
		},
		{
			name:     "extract from complex JSON",
			body:     `{"status": 400, "error": true, "message": "Invalid request format", "details": {"field": "duration"}}`,
			expected: "Invalid request format",
		},
		{
			name:     "no message field",
			body:     `{"error": true, "code": "ERROR"}`,
			expected: "",
		},
		{
			name:     "non-JSON body",
			body:     "Internal Server Error",
			expected: "",
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractErrorMessage(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCaptureRequestHeaders(t *testing.T) {
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("User-Agent", "TestClient/1.0")
	r.Header.Set("Authorization", "Bearer secret-token") // Should not be captured
	r.Header.Set("X-Custom-Header", "custom-value")      // Should not be captured
	
	headers := captureRequestHeaders(r)
	
	assert.Contains(t, headers, "Content-Type")
	assert.Contains(t, headers, "Accept")
	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "application/json", headers["Accept"])
	
	// Sensitive headers should not be captured
	assert.NotContains(t, headers, "Authorization")
	assert.NotContains(t, headers, "X-Custom-Header")
}

func TestCaptureResponseHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Content-Length", "123")
	headers.Set("X-Request-ID", "req-123")
	headers.Set("X-RateLimit-Remaining", "99")
	headers.Set("Set-Cookie", "session=secret") // Should not be captured
	
	captured := captureResponseHeaders(headers)
	
	assert.Contains(t, captured, "Content-Type")
	assert.Contains(t, captured, "Content-Length")
	assert.Contains(t, captured, "X-Request-ID")
	assert.Contains(t, captured, "X-RateLimit-Remaining")
	assert.Equal(t, "application/json", captured["Content-Type"])
	assert.Equal(t, "123", captured["Content-Length"])
	
	// Sensitive headers should not be captured
	assert.NotContains(t, captured, "Set-Cookie")
}

func TestIsSensitiveOperation(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/config", true},
		{"/api/v1/door/unlock", true},
		{"/api/v1/adapters/test", true},
		{"/api/v1/events", true},
		{"/api/v1/health", false},
		{"/api/v1/status", false},
		{"/api/v1/metrics", false},
		{"/api/v1/ws", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil)
			result := isSensitiveOperation(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDataAccessOperation(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/events", true},
		{"/api/v1/config", true},
		{"/api/v1/status", true},
		{"/api/v1/metrics", true},
		{"/api/v1/door/unlock", false},
		{"/api/v1/adapters/test/enable", false},
		{"/api/v1/health", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.path, nil)
			result := isDataAccessOperation(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPrivilegedOperation(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected bool
	}{
		{"POST", "/api/v1/door/unlock", true},
		{"POST", "/api/v1/door/lock", true},
		{"PUT", "/api/v1/config", true},
		{"POST", "/api/v1/adapters/test/enable", true},
		{"DELETE", "/api/v1/events", true},
		{"GET", "/api/v1/status", false},
		{"GET", "/api/v1/health", false},
		{"GET", "/api/v1/events", false},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.path), func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			result := isPrivilegedOperation(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestMetrics(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(100 * time.Millisecond)
	
	metrics := RequestMetrics{
		RequestID:    "req-123",
		Method:       "POST",
		Path:         "/api/v1/door/unlock",
		StatusCode:   200,
		ResponseSize: 256,
		Duration:     endTime.Sub(startTime),
		ClientIP:     "192.168.1.100",
		UserAgent:    "TestClient/1.0",
		StartTime:    startTime,
		EndTime:      endTime,
	}
	
	assert.Equal(t, "req-123", metrics.RequestID)
	assert.Equal(t, "POST", metrics.Method)
	assert.Equal(t, "/api/v1/door/unlock", metrics.Path)
	assert.Equal(t, 200, metrics.StatusCode)
	assert.Equal(t, int64(256), metrics.ResponseSize)
	assert.Equal(t, 100*time.Millisecond, metrics.Duration)
	assert.Equal(t, "192.168.1.100", metrics.ClientIP)
	assert.Equal(t, "TestClient/1.0", metrics.UserAgent)
}

// Benchmark tests
func BenchmarkStructuredLoggingMiddleware(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	rl := NewRequestLogger(logger)
	
	handler := rl.StructuredLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/status", nil)
		
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkExtractErrorMessage(b *testing.B) {
	body := `{"error": true, "message": "Validation failed", "code": "VALIDATION_ERROR"}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractErrorMessage(body)
	}
}