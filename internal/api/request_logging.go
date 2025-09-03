package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RequestLogger provides comprehensive request logging with structured data
type RequestLogger struct {
	logger      *logrus.Logger
	auditLogger *AuditLogger
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger *logrus.Logger) *RequestLogger {
	return &RequestLogger{
		logger:      logger,
		auditLogger: NewAuditLogger(logger),
	}
}

// LoggingResponseWriter wraps http.ResponseWriter to capture response data
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
	body         *bytes.Buffer
	captureBody  bool
}

// NewLoggingResponseWriter creates a new logging response writer
func NewLoggingResponseWriter(w http.ResponseWriter, captureBody bool) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
		captureBody:    captureBody,
	}
}

func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.responseSize += int64(size)
	
	// Capture response body if enabled and not too large
	if lrw.captureBody && lrw.body.Len() < 1024*10 { // Limit to 10KB
		lrw.body.Write(b)
	}
	
	return size, err
}

func (lrw *LoggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")
}

// RequestMetrics holds metrics about a request
type RequestMetrics struct {
	RequestID       string        `json:"request_id"`
	Method          string        `json:"method"`
	Path            string        `json:"path"`
	Query           string        `json:"query,omitempty"`
	StatusCode      int           `json:"status_code"`
	ResponseSize    int64         `json:"response_size"`
	Duration        time.Duration `json:"duration"`
	ClientIP        string        `json:"client_ip"`
	UserAgent       string        `json:"user_agent"`
	Referer         string        `json:"referer,omitempty"`
	RequestSize     int64         `json:"request_size"`
	Protocol        string        `json:"protocol"`
	TLS             bool          `json:"tls"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	AuthMethod      string        `json:"auth_method,omitempty"`
	ErrorMessage    string        `json:"error_message,omitempty"`
	RequestHeaders  map[string]string `json:"request_headers,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
}

// StructuredLoggingMiddleware provides comprehensive request logging
func (rl *RequestLogger) StructuredLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		requestID := generateRequestID()
		
		// Add request ID to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "request_id", requestID)
		r = r.WithContext(ctx)

		// Capture request body size
		var requestSize int64
		if r.ContentLength > 0 {
			requestSize = r.ContentLength
		}

		// Create logging response writer
		captureBody := shouldCaptureResponseBody(r)
		lrw := NewLoggingResponseWriter(w, captureBody)

		// Add request ID header to response
		lrw.Header().Set("X-Request-ID", requestID)

		// Execute request
		next.ServeHTTP(lrw, r)

		// Calculate metrics
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Build request metrics
		metrics := RequestMetrics{
			RequestID:    requestID,
			Method:       r.Method,
			Path:         r.URL.Path,
			Query:        r.URL.RawQuery,
			StatusCode:   lrw.statusCode,
			ResponseSize: lrw.responseSize,
			Duration:     duration,
			ClientIP:     getClientIP(r),
			UserAgent:    r.UserAgent(),
			Referer:      r.Referer(),
			RequestSize:  requestSize,
			Protocol:     r.Proto,
			TLS:          r.TLS != nil,
			StartTime:    startTime,
			EndTime:      endTime,
		}

		// Capture auth method if available
		if authMethod := getAuthMethod(r); authMethod != "" {
			metrics.AuthMethod = authMethod
		}

		// Capture error message for error responses
		if lrw.statusCode >= 400 && lrw.captureBody && lrw.body.Len() > 0 {
			metrics.ErrorMessage = extractErrorMessage(lrw.body.String())
		}

		// Capture headers if configured
		if shouldCaptureHeaders(r) {
			metrics.RequestHeaders = captureRequestHeaders(r)
			metrics.ResponseHeaders = captureResponseHeaders(lrw.Header())
		}

		// Log the request
		rl.logRequest(metrics)

		// Log audit events for sensitive operations
		rl.logAuditEvents(r, metrics)
	})
}

// logRequest logs the request with appropriate level and structured data
func (rl *RequestLogger) logRequest(metrics RequestMetrics) {
	// Determine log level based on status code and duration
	logLevel := logrus.InfoLevel
	if metrics.StatusCode >= 500 {
		logLevel = logrus.ErrorLevel
	} else if metrics.StatusCode >= 400 {
		logLevel = logrus.WarnLevel
	} else if metrics.Duration > 5*time.Second {
		logLevel = logrus.WarnLevel
	}

	// Create log entry with structured fields
	entry := rl.logger.WithFields(logrus.Fields{
		"request_id":     metrics.RequestID,
		"method":         metrics.Method,
		"path":           metrics.Path,
		"status_code":    metrics.StatusCode,
		"duration_ms":    metrics.Duration.Milliseconds(),
		"response_size":  metrics.ResponseSize,
		"client_ip":      metrics.ClientIP,
		"user_agent":     metrics.UserAgent,
		"protocol":       metrics.Protocol,
		"tls":            metrics.TLS,
	})

	// Add optional fields
	if metrics.Query != "" {
		entry = entry.WithField("query", metrics.Query)
	}
	if metrics.Referer != "" {
		entry = entry.WithField("referer", metrics.Referer)
	}
	if metrics.AuthMethod != "" {
		entry = entry.WithField("auth_method", metrics.AuthMethod)
	}
	if metrics.ErrorMessage != "" {
		entry = entry.WithField("error_message", metrics.ErrorMessage)
	}
	if metrics.RequestSize > 0 {
		entry = entry.WithField("request_size", metrics.RequestSize)
	}

	// Add performance metrics
	entry = entry.WithFields(logrus.Fields{
		"start_time": metrics.StartTime.Format(time.RFC3339Nano),
		"end_time":   metrics.EndTime.Format(time.RFC3339Nano),
	})

	// Log message
	message := fmt.Sprintf("%s %s %d %dms", 
		metrics.Method, 
		metrics.Path, 
		metrics.StatusCode, 
		metrics.Duration.Milliseconds())

	entry.Log(logLevel, message)
}

// logAuditEvents logs audit events for sensitive operations
func (rl *RequestLogger) logAuditEvents(r *http.Request, metrics RequestMetrics) {
	// Determine if this is a sensitive operation that requires audit logging
	if !isSensitiveOperation(r) {
		return
	}

	// Log data access events
	if isDataAccessOperation(r) {
		eventType := AuditEventDataAccess
		if r.Method == "PUT" || r.Method == "POST" || r.Method == "PATCH" {
			eventType = AuditEventDataModification
		} else if r.Method == "DELETE" {
			eventType = AuditEventDataDeletion
		}

		details := map[string]interface{}{
			"status_code":   metrics.StatusCode,
			"duration_ms":   metrics.Duration.Milliseconds(),
			"response_size": metrics.ResponseSize,
		}

		rl.auditLogger.LogDataAccessEvent(eventType, r, r.URL.Path, details)
	}

	// Log privileged actions
	if isPrivilegedOperation(r) {
		result := "success"
		if metrics.StatusCode >= 400 {
			result = "failure"
		}

		details := map[string]interface{}{
			"status_code": metrics.StatusCode,
			"duration_ms": metrics.Duration.Milliseconds(),
		}

		rl.auditLogger.LogPrivilegedAction(r, r.Method, r.URL.Path, result, details)
	}
}

// Helper functions

// shouldCaptureResponseBody determines if response body should be captured
func shouldCaptureResponseBody(r *http.Request) bool {
	// Capture body for error responses and sensitive operations
	return strings.Contains(r.URL.Path, "/config") || 
		   strings.Contains(r.URL.Path, "/door") ||
		   strings.Contains(r.URL.Path, "/adapters")
}

// shouldCaptureHeaders determines if headers should be captured
func shouldCaptureHeaders(r *http.Request) bool {
	// Capture headers for sensitive operations
	return isSensitiveOperation(r)
}

// getAuthMethod extracts authentication method from request
func getAuthMethod(r *http.Request) string {
	if r.Header.Get("X-API-Key") != "" {
		return "api_key"
	}
	if r.Header.Get("X-Signature") != "" {
		return "hmac"
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return "jwt"
		}
	}
	return ""
}

// extractErrorMessage extracts error message from response body
func extractErrorMessage(body string) string {
	// Try to extract error message from JSON response
	if strings.Contains(body, "\"message\"") {
		// Simple extraction - in production, you might want to use JSON parsing
		start := strings.Index(body, "\"message\":\"")
		if start != -1 {
			start += 11 // Length of "\"message\":\""
			end := strings.Index(body[start:], "\"")
			if end != -1 {
				return body[start : start+end]
			}
		}
	}
	return ""
}

// captureRequestHeaders captures relevant request headers
func captureRequestHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	
	// Capture specific headers (avoid sensitive ones)
	relevantHeaders := []string{
		"Content-Type", "Accept", "Accept-Encoding", "Accept-Language",
		"Cache-Control", "Connection", "Host", "Origin",
	}
	
	for _, header := range relevantHeaders {
		if value := r.Header.Get(header); value != "" {
			headers[header] = value
		}
	}
	
	return headers
}

// captureResponseHeaders captures relevant response headers
func captureResponseHeaders(headers http.Header) map[string]string {
	captured := make(map[string]string)
	
	// Capture specific headers
	relevantHeaders := []string{
		"Content-Type", "Content-Length", "Cache-Control",
		"X-Request-ID", "X-RateLimit-Remaining",
	}
	
	for _, header := range relevantHeaders {
		if value := headers.Get(header); value != "" {
			captured[header] = value
		}
	}
	
	return captured
}

// isSensitiveOperation determines if an operation is sensitive
func isSensitiveOperation(r *http.Request) bool {
	sensitivePaths := []string{
		"/config", "/door", "/adapters", "/events",
	}
	
	for _, path := range sensitivePaths {
		if strings.Contains(r.URL.Path, path) {
			return true
		}
	}
	
	return false
}

// isDataAccessOperation determines if an operation accesses data
func isDataAccessOperation(r *http.Request) bool {
	dataAccessPaths := []string{
		"/events", "/config", "/status", "/metrics",
	}
	
	for _, path := range dataAccessPaths {
		if strings.Contains(r.URL.Path, path) {
			return true
		}
	}
	
	return false
}

// isPrivilegedOperation determines if an operation is privileged
func isPrivilegedOperation(r *http.Request) bool {
	privilegedPaths := []string{
		"/door/unlock", "/door/lock", "/config", "/adapters",
	}
	
	for _, path := range privilegedPaths {
		if strings.Contains(r.URL.Path, path) {
			return true
		}
	}
	
	return r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE"
}