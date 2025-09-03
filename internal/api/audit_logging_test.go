package api

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogger_NewAuditLogger(t *testing.T) {
	logger := logrus.New()
	al := NewAuditLogger(logger)
	
	assert.NotNil(t, al)
	assert.Equal(t, logger, al.logger)
}

func TestAuditLogger_LogEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	al := NewAuditLogger(logger)
	
	t.Run("log basic event", func(t *testing.T) {
		buf.Reset()
		
		event := AuditEvent{
			EventType: AuditEventAuthSuccess,
			Severity:  AuditSeverityMedium,
			ClientIP:  "192.168.1.100",
			Resource:  "authentication",
			Action:    "login",
			Result:    "success",
			Message:   "User logged in successfully",
		}
		
		al.LogEvent(event)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "auth_success")
		assert.Contains(t, logOutput, "medium")
		assert.Contains(t, logOutput, "192.168.1.100")
		assert.Contains(t, logOutput, "authentication")
		assert.Contains(t, logOutput, "login")
		assert.Contains(t, logOutput, "success")
		assert.Contains(t, logOutput, "User logged in successfully")
	})
	
	t.Run("log event with details", func(t *testing.T) {
		buf.Reset()
		
		event := AuditEvent{
			EventType: AuditEventConfigUpdate,
			Severity:  AuditSeverityHigh,
			ClientIP:  "10.0.0.1",
			UserID:    "admin",
			Resource:  "configuration",
			Action:    "update",
			Result:    "success",
			Message:   "Configuration updated",
			Details: map[string]interface{}{
				"field":     "unlock_duration",
				"old_value": 3000,
				"new_value": 5000,
			},
		}
		
		al.LogEvent(event)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "config_update")
		assert.Contains(t, logOutput, "high")
		assert.Contains(t, logOutput, "admin")
		assert.Contains(t, logOutput, "detail_field")
		assert.Contains(t, logOutput, "detail_old_value")
		assert.Contains(t, logOutput, "detail_new_value")
	})
	
	t.Run("log event with state changes", func(t *testing.T) {
		buf.Reset()
		
		beforeState := map[string]interface{}{
			"enabled": false,
			"port":    8080,
		}
		afterState := map[string]interface{}{
			"enabled": true,
			"port":    8081,
		}
		
		event := AuditEvent{
			EventType:   AuditEventConfigUpdate,
			Severity:    AuditSeverityHigh,
			ClientIP:    "172.16.0.1",
			Resource:    "api_server",
			Action:      "configuration_change",
			Result:      "success",
			Message:     "API server configuration changed",
			BeforeState: beforeState,
			AfterState:  afterState,
		}
		
		al.LogEvent(event)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "before_state")
		assert.Contains(t, logOutput, "after_state")
		assert.Contains(t, logOutput, "enabled")
		assert.Contains(t, logOutput, "port")
	})
	
	t.Run("auto-generate ID and timestamp", func(t *testing.T) {
		buf.Reset()
		
		event := AuditEvent{
			EventType: AuditEventDoorUnlock,
			Severity:  AuditSeverityHigh,
			ClientIP:  "192.168.1.50",
			Resource:  "door_control",
			Action:    "unlock",
			Result:    "success",
			Message:   "Door unlocked",
		}
		
		al.LogEvent(event)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "audit_id")
		assert.Contains(t, logOutput, "time")
	})
}

func TestAuditLogger_LogAuthenticationEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	al := NewAuditLogger(logger)
	
	t.Run("log auth success", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("POST", "/api/v1/auth", nil)
		r.Header.Set("User-Agent", "TestClient/1.0")
		r.RemoteAddr = "192.168.1.100:12345"
		
		details := map[string]interface{}{
			"method": "api_key",
			"user":   "test_user",
		}
		
		al.LogAuthenticationEvent(AuditEventAuthSuccess, r, "success", details)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "auth_success")
		assert.Contains(t, logOutput, "medium")
		assert.Contains(t, logOutput, "192.168.1.100")
		assert.Contains(t, logOutput, "TestClient/1.0")
		assert.Contains(t, logOutput, "detail_method")
		assert.Contains(t, logOutput, "detail_user")
	})
	
	t.Run("log auth failure with high severity", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("POST", "/api/v1/auth", nil)
		r.RemoteAddr = "10.0.0.1:54321"
		
		al.LogAuthenticationEvent(AuditEventAuthFailure, r, "failure", nil)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "auth_failure")
		assert.Contains(t, logOutput, "high")
		assert.Contains(t, logOutput, "10.0.0.1")
	})
}

func TestAuditLogger_LogConfigurationEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	al := NewAuditLogger(logger)
	
	r := httptest.NewRequest("PUT", "/api/v1/config", nil)
	r.Header.Set("User-Agent", "ConfigTool/2.0")
	r.RemoteAddr = "172.16.0.10:8080"
	
	// Add request ID to context
	ctx := context.WithValue(r.Context(), "request_id", "req_12345")
	r = r.WithContext(ctx)
	
	beforeState := map[string]interface{}{
		"unlock_duration": 3000,
		"log_level":       "info",
	}
	afterState := map[string]interface{}{
		"unlock_duration": 5000,
		"log_level":       "debug",
	}
	
	al.LogConfigurationEvent(AuditEventConfigUpdate, r, "device_config", beforeState, afterState)
	
	logOutput := buf.String()
	assert.Contains(t, logOutput, "config_update")
	assert.Contains(t, logOutput, "high")
	assert.Contains(t, logOutput, "172.16.0.10")
	assert.Contains(t, logOutput, "ConfigTool/2.0")
	assert.Contains(t, logOutput, "req_12345")
	assert.Contains(t, logOutput, "device_config")
	assert.Contains(t, logOutput, "before_state")
	assert.Contains(t, logOutput, "after_state")
}

func TestAuditLogger_LogDoorControlEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	al := NewAuditLogger(logger)
	
	t.Run("successful door unlock", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("POST", "/api/v1/door/unlock", nil)
		r.RemoteAddr = "192.168.1.200:9999"
		
		details := map[string]interface{}{
			"duration_ms": 5000,
			"adapter":     "fingerprint_reader",
			"reason":      "authorized_access",
		}
		
		al.LogDoorControlEvent(AuditEventDoorUnlock, r, "success", details)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "door_unlock")
		assert.Contains(t, logOutput, "high")
		assert.Contains(t, logOutput, "192.168.1.200")
		assert.Contains(t, logOutput, "door_control")
		assert.Contains(t, logOutput, "success")
		assert.Contains(t, logOutput, "detail_duration_ms")
		assert.Contains(t, logOutput, "detail_adapter")
	})
	
	t.Run("failed door unlock with critical severity", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("POST", "/api/v1/door/unlock", nil)
		r.RemoteAddr = "10.0.0.50:1234"
		
		details := map[string]interface{}{
			"error": "hardware_failure",
		}
		
		al.LogDoorControlEvent(AuditEventDoorUnlock, r, "failure", details)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "door_unlock")
		assert.Contains(t, logOutput, "critical")
		assert.Contains(t, logOutput, "failure")
		assert.Contains(t, logOutput, "detail_error")
	})
}

func TestAuditLogger_LogDataAccessEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	al := NewAuditLogger(logger)
	
	t.Run("data access event", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("GET", "/api/v1/events", nil)
		r.RemoteAddr = "192.168.1.150:5555"
		
		details := map[string]interface{}{
			"query_params": "startTime=2023-01-01&limit=100",
			"result_count": 50,
		}
		
		al.LogDataAccessEvent(AuditEventDataAccess, r, "events", details)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "data_access")
		assert.Contains(t, logOutput, "medium")
		assert.Contains(t, logOutput, "events")
		assert.Contains(t, logOutput, "detail_query_params")
		assert.Contains(t, logOutput, "detail_result_count")
	})
	
	t.Run("data modification event with high severity", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("DELETE", "/api/v1/events", nil)
		r.RemoteAddr = "172.16.0.20:7777"
		
		al.LogDataAccessEvent(AuditEventDataDeletion, r, "events", nil)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "data_deletion")
		assert.Contains(t, logOutput, "high")
	})
}

func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	al := NewAuditLogger(logger)
	
	r := httptest.NewRequest("POST", "/api/v1/door/unlock", nil)
	r.RemoteAddr = "203.0.113.100:12345"
	r.Header.Set("User-Agent", "SuspiciousBot/1.0")
	
	details := map[string]interface{}{
		"reason":        "invalid_signature",
		"attempts":      5,
		"blocked_until": time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	
	al.LogSecurityEvent(AuditEventSecurityViolation, r, "Multiple authentication failures detected", details)
	
	logOutput := buf.String()
	assert.Contains(t, logOutput, "security_violation")
	assert.Contains(t, logOutput, "critical")
	assert.Contains(t, logOutput, "203.0.113.100")
	assert.Contains(t, logOutput, "SuspiciousBot/1.0")
	assert.Contains(t, logOutput, "Multiple authentication failures detected")
	assert.Contains(t, logOutput, "detail_reason")
	assert.Contains(t, logOutput, "detail_attempts")
	assert.Contains(t, logOutput, "detail_blocked_until")
}

func TestAuditLogger_LogPrivilegedAction(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	al := NewAuditLogger(logger)
	
	t.Run("successful privileged action", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("POST", "/api/v1/adapters/fingerprint/enable", nil)
		r.RemoteAddr = "192.168.1.10:8888"
		
		details := map[string]interface{}{
			"adapter": "fingerprint_reader",
			"reason":  "maintenance_complete",
		}
		
		al.LogPrivilegedAction(r, "enable_adapter", "adapter_management", "success", details)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "privileged_action")
		assert.Contains(t, logOutput, "high")
		assert.Contains(t, logOutput, "enable_adapter")
		assert.Contains(t, logOutput, "adapter_management")
		assert.Contains(t, logOutput, "success")
		assert.Contains(t, logOutput, "detail_adapter")
		assert.Contains(t, logOutput, "detail_reason")
	})
	
	t.Run("failed privileged action with critical severity", func(t *testing.T) {
		buf.Reset()
		
		r := httptest.NewRequest("DELETE", "/api/v1/events", nil)
		r.RemoteAddr = "10.0.0.100:9999"
		
		al.LogPrivilegedAction(r, "clear_events", "data_management", "failure", nil)
		
		logOutput := buf.String()
		assert.Contains(t, logOutput, "privileged_action")
		assert.Contains(t, logOutput, "critical")
		assert.Contains(t, logOutput, "clear_events")
		assert.Contains(t, logOutput, "failure")
	})
}

func TestGetRequestIDFromContext(t *testing.T) {
	t.Run("extract request ID from context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
		
		requestID := getRequestIDFromContext(ctx)
		
		assert.Equal(t, "test-request-123", requestID)
	})
	
	t.Run("return empty string when no request ID", func(t *testing.T) {
		ctx := context.Background()
		
		requestID := getRequestIDFromContext(ctx)
		
		assert.Equal(t, "", requestID)
	})
	
	t.Run("return empty string when wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "request_id", 12345)
		
		requestID := getRequestIDFromContext(ctx)
		
		assert.Equal(t, "", requestID)
	})
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()
	
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "req_")
	assert.Contains(t, id2, "req_")
}

func TestRandomString(t *testing.T) {
	str1 := randomString(8)
	str2 := randomString(8)
	
	assert.Len(t, str1, 8)
	assert.Len(t, str2, 8)
	assert.NotEqual(t, str1, str2)
	
	// Test different lengths
	assert.Len(t, randomString(4), 4)
	assert.Len(t, randomString(16), 16)
}

// Benchmark tests
func BenchmarkAuditLogger_LogEvent(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	
	al := NewAuditLogger(logger)
	
	event := AuditEvent{
		EventType: AuditEventAuthSuccess,
		Severity:  AuditSeverityMedium,
		ClientIP:  "192.168.1.100",
		Resource:  "authentication",
		Action:    "login",
		Result:    "success",
		Message:   "User logged in successfully",
		Details: map[string]interface{}{
			"method": "api_key",
			"user":   "test_user",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		al.LogEvent(event)
	}
}

func BenchmarkGenerateRequestID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateRequestID()
	}
}