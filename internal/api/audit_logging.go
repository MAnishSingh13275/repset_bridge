package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// AuditEventType represents different types of audit events
type AuditEventType string

const (
	// Authentication Events
	AuditEventAuthSuccess      AuditEventType = "auth_success"
	AuditEventAuthFailure      AuditEventType = "auth_failure"
	AuditEventTokenExpired     AuditEventType = "token_expired"
	AuditEventInvalidToken     AuditEventType = "invalid_token"
	AuditEventIPBlocked        AuditEventType = "ip_blocked"
	AuditEventRateLimitHit     AuditEventType = "rate_limit_hit"

	// Configuration Events
	AuditEventConfigRead       AuditEventType = "config_read"
	AuditEventConfigUpdate     AuditEventType = "config_update"
	AuditEventConfigReload     AuditEventType = "config_reload"
	AuditEventAdapterEnabled   AuditEventType = "adapter_enabled"
	AuditEventAdapterDisabled  AuditEventType = "adapter_disabled"
	AuditEventAdapterConfigUpdate AuditEventType = "adapter_config_update"

	// Door Control Events
	AuditEventDoorUnlock       AuditEventType = "door_unlock"
	AuditEventDoorLock         AuditEventType = "door_lock"
	AuditEventDoorStatusCheck  AuditEventType = "door_status_check"

	// System Events
	AuditEventSystemAccess     AuditEventType = "system_access"
	AuditEventDataAccess       AuditEventType = "data_access"
	AuditEventDataModification AuditEventType = "data_modification"
	AuditEventDataDeletion     AuditEventType = "data_deletion"
	AuditEventPrivilegedAction AuditEventType = "privileged_action"

	// Security Events
	AuditEventSecurityViolation AuditEventType = "security_violation"
	AuditEventSuspiciousActivity AuditEventType = "suspicious_activity"
	AuditEventAccessDenied      AuditEventType = "access_denied"
)

// AuditSeverity represents the severity level of audit events
type AuditSeverity string

const (
	AuditSeverityLow      AuditSeverity = "low"
	AuditSeverityMedium   AuditSeverity = "medium"
	AuditSeverityHigh     AuditSeverity = "high"
	AuditSeverityCritical AuditSeverity = "critical"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   AuditEventType         `json:"event_type"`
	Severity    AuditSeverity          `json:"severity"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	ClientIP    string                 `json:"client_ip"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Result      string                 `json:"result"` // success, failure, denied
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	BeforeState map[string]interface{} `json:"before_state,omitempty"`
	AfterState  map[string]interface{} `json:"after_state,omitempty"`
}

// AuditLogger provides comprehensive audit logging capabilities
type AuditLogger struct {
	logger *logrus.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *logrus.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
	}
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(event AuditEvent) {
	// Ensure required fields are set
	if event.ID == "" {
		event.ID = generateRequestID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Create structured log entry
	entry := al.logger.WithFields(logrus.Fields{
		"audit_id":     event.ID,
		"event_type":   event.EventType,
		"severity":     event.Severity,
		"client_ip":    event.ClientIP,
		"resource":     event.Resource,
		"action":       event.Action,
		"result":       event.Result,
		"request_id":   event.RequestID,
	})

	// Add optional fields
	if event.UserID != "" {
		entry = entry.WithField("user_id", event.UserID)
	}
	if event.SessionID != "" {
		entry = entry.WithField("session_id", event.SessionID)
	}
	if event.UserAgent != "" {
		entry = entry.WithField("user_agent", event.UserAgent)
	}

	// Add details as separate fields
	if event.Details != nil {
		for key, value := range event.Details {
			entry = entry.WithField("detail_"+key, value)
		}
	}

	// Add state changes if present
	if event.BeforeState != nil {
		beforeJSON, _ := json.Marshal(event.BeforeState)
		entry = entry.WithField("before_state", string(beforeJSON))
	}
	if event.AfterState != nil {
		afterJSON, _ := json.Marshal(event.AfterState)
		entry = entry.WithField("after_state", string(afterJSON))
	}

	// Log at appropriate level based on severity
	switch event.Severity {
	case AuditSeverityCritical:
		entry.Error(event.Message)
	case AuditSeverityHigh:
		entry.Warn(event.Message)
	case AuditSeverityMedium:
		entry.Info(event.Message)
	case AuditSeverityLow:
		entry.Debug(event.Message)
	default:
		entry.Info(event.Message)
	}
}

// LogAuthenticationEvent logs authentication-related events
func (al *AuditLogger) LogAuthenticationEvent(eventType AuditEventType, r *http.Request, result string, details map[string]interface{}) {
	severity := AuditSeverityMedium
	if eventType == AuditEventAuthFailure || eventType == AuditEventIPBlocked {
		severity = AuditSeverityHigh
	}

	event := AuditEvent{
		EventType: eventType,
		Severity:  severity,
		ClientIP:  getClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: getRequestIDFromContext(r.Context()),
		Resource:  "authentication",
		Action:    string(eventType),
		Result:    result,
		Message:   fmt.Sprintf("Authentication event: %s", eventType),
		Details:   details,
	}

	al.LogEvent(event)
}

// LogConfigurationEvent logs configuration change events
func (al *AuditLogger) LogConfigurationEvent(eventType AuditEventType, r *http.Request, resource string, beforeState, afterState map[string]interface{}) {
	event := AuditEvent{
		EventType:   eventType,
		Severity:    AuditSeverityHigh,
		ClientIP:    getClientIP(r),
		UserAgent:   r.UserAgent(),
		RequestID:   getRequestIDFromContext(r.Context()),
		Resource:    resource,
		Action:      string(eventType),
		Result:      "success",
		Message:     fmt.Sprintf("Configuration change: %s on %s", eventType, resource),
		BeforeState: beforeState,
		AfterState:  afterState,
	}

	al.LogEvent(event)
}

// LogDoorControlEvent logs door control events
func (al *AuditLogger) LogDoorControlEvent(eventType AuditEventType, r *http.Request, result string, details map[string]interface{}) {
	severity := AuditSeverityHigh
	if result != "success" {
		severity = AuditSeverityCritical
	}

	event := AuditEvent{
		EventType: eventType,
		Severity:  severity,
		ClientIP:  getClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: getRequestIDFromContext(r.Context()),
		Resource:  "door_control",
		Action:    string(eventType),
		Result:    result,
		Message:   fmt.Sprintf("Door control event: %s", eventType),
		Details:   details,
	}

	al.LogEvent(event)
}

// LogDataAccessEvent logs data access events
func (al *AuditLogger) LogDataAccessEvent(eventType AuditEventType, r *http.Request, resource string, details map[string]interface{}) {
	severity := AuditSeverityMedium
	if eventType == AuditEventDataModification || eventType == AuditEventDataDeletion {
		severity = AuditSeverityHigh
	}

	event := AuditEvent{
		EventType: eventType,
		Severity:  severity,
		ClientIP:  getClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: getRequestIDFromContext(r.Context()),
		Resource:  resource,
		Action:    string(eventType),
		Result:    "success",
		Message:   fmt.Sprintf("Data access event: %s on %s", eventType, resource),
		Details:   details,
	}

	al.LogEvent(event)
}

// LogSecurityEvent logs security-related events
func (al *AuditLogger) LogSecurityEvent(eventType AuditEventType, r *http.Request, message string, details map[string]interface{}) {
	event := AuditEvent{
		EventType: eventType,
		Severity:  AuditSeverityCritical,
		ClientIP:  getClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: getRequestIDFromContext(r.Context()),
		Resource:  "security",
		Action:    string(eventType),
		Result:    "violation",
		Message:   message,
		Details:   details,
	}

	al.LogEvent(event)
}

// LogPrivilegedAction logs privileged actions
func (al *AuditLogger) LogPrivilegedAction(r *http.Request, action, resource string, result string, details map[string]interface{}) {
	severity := AuditSeverityHigh
	if result != "success" {
		severity = AuditSeverityCritical
	}

	event := AuditEvent{
		EventType: AuditEventPrivilegedAction,
		Severity:  severity,
		ClientIP:  getClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: getRequestIDFromContext(r.Context()),
		Resource:  resource,
		Action:    action,
		Result:    result,
		Message:   fmt.Sprintf("Privileged action: %s on %s", action, resource),
		Details:   details,
	}

	al.LogEvent(event)
}

// Helper functions

// getRequestIDFromContext extracts request ID from context
func getRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%s", time.Now().UnixNano(), randomString(8))
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}