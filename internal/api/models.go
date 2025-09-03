package api

import (
	"fmt"
	"net/http"
	"time"
)

// DoorUnlockRequest represents a request to unlock the door
type DoorUnlockRequest struct {
	DurationMs  int    `json:"durationMs" validate:"min=1000,max=30000"`
	Reason      string `json:"reason,omitempty"`
	RequestedBy string `json:"requestedBy,omitempty"`
	Adapter     string `json:"adapter,omitempty"`
}

// Validate validates the door unlock request
func (r *DoorUnlockRequest) Validate() error {
	if r.DurationMs < 1000 {
		return fmt.Errorf("durationMs must be at least 1000 milliseconds")
	}
	if r.DurationMs > 30000 {
		return fmt.Errorf("durationMs must not exceed 30000 milliseconds")
	}
	return nil
}

// DoorLockRequest represents a request to lock the door
type DoorLockRequest struct {
	Reason      string `json:"reason,omitempty"`
	RequestedBy string `json:"requestedBy,omitempty"`
	Adapter     string `json:"adapter,omitempty"`
}

// DoorUnlockResponse represents the response to a door unlock request
type DoorUnlockResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Duration  int       `json:"duration"`
	Adapter   string    `json:"adapter"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"requestId,omitempty"`
}

// DoorLockResponse represents the response to a door lock request
type DoorLockResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Adapter   string    `json:"adapter"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"requestId,omitempty"`
}

// DoorStatusResponse represents the current door status
type DoorStatusResponse struct {
	IsLocked        bool      `json:"isLocked"`
	Status          string    `json:"status"` // "locked", "unlocked", "unknown"
	LastUnlockTime  *time.Time `json:"lastUnlockTime,omitempty"`
	LastLockTime    *time.Time `json:"lastLockTime,omitempty"`
	UnlockCount     int64     `json:"unlockCount"`
	ActiveAdapters  []string  `json:"activeAdapters"`
	Timestamp       time.Time `json:"timestamp"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error     string            `json:"error"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	RequestID string            `json:"requestId,omitempty"`
	Path      string            `json:"path,omitempty"`
	Method    string            `json:"method,omitempty"`
	Status    int               `json:"status"`
}

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Authentication and Authorization Errors
	ErrorCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrorCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrorCodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrorCodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	ErrorCodeIPBlocked          ErrorCode = "IP_BLOCKED"
	
	// Validation Errors
	ErrorCodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
	ErrorCodeInvalidJSON        ErrorCode = "INVALID_JSON"
	ErrorCodeMissingField       ErrorCode = "MISSING_FIELD"
	ErrorCodeInvalidFormat      ErrorCode = "INVALID_FORMAT"
	ErrorCodeInvalidRange       ErrorCode = "INVALID_RANGE"
	
	// Resource Errors
	ErrorCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrorCodeConflict           ErrorCode = "CONFLICT"
	ErrorCodeResourceExists     ErrorCode = "RESOURCE_EXISTS"
	ErrorCodeResourceLocked     ErrorCode = "RESOURCE_LOCKED"
	
	// Rate Limiting Errors
	ErrorCodeRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrorCodeTooManyRequests    ErrorCode = "TOO_MANY_REQUESTS"
	
	// Service Errors
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrorCodeTimeout            ErrorCode = "TIMEOUT"
	ErrorCodeCircuitBreakerOpen ErrorCode = "CIRCUIT_BREAKER_OPEN"
	
	// Hardware/Adapter Errors
	ErrorCodeHardwareFailure    ErrorCode = "HARDWARE_FAILURE"
	ErrorCodeAdapterNotFound    ErrorCode = "ADAPTER_NOT_FOUND"
	ErrorCodeAdapterDisabled    ErrorCode = "ADAPTER_DISABLED"
	ErrorCodeDoorOperationFailed ErrorCode = "DOOR_OPERATION_FAILED"
	
	// Configuration Errors
	ErrorCodeConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrorCodeConfigNotFound     ErrorCode = "CONFIG_NOT_FOUND"
	ErrorCodeConfigUpdateFailed ErrorCode = "CONFIG_UPDATE_FAILED"
	
	// Database/Storage Errors
	ErrorCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrorCodeStorageError       ErrorCode = "STORAGE_ERROR"
	ErrorCodeDataCorruption     ErrorCode = "DATA_CORRUPTION"
)

// HTTPStatusMapping maps error codes to HTTP status codes
var HTTPStatusMapping = map[ErrorCode]int{
	// 400 Bad Request
	ErrorCodeValidationFailed:   http.StatusBadRequest,
	ErrorCodeInvalidJSON:        http.StatusBadRequest,
	ErrorCodeMissingField:       http.StatusBadRequest,
	ErrorCodeInvalidFormat:      http.StatusBadRequest,
	ErrorCodeInvalidRange:       http.StatusBadRequest,
	ErrorCodeConfigInvalid:      http.StatusBadRequest,
	
	// 401 Unauthorized
	ErrorCodeUnauthorized:        http.StatusUnauthorized,
	ErrorCodeInvalidCredentials: http.StatusUnauthorized,
	ErrorCodeTokenExpired:       http.StatusUnauthorized,
	ErrorCodeInvalidToken:       http.StatusUnauthorized,
	
	// 403 Forbidden
	ErrorCodeForbidden:          http.StatusForbidden,
	ErrorCodeIPBlocked:          http.StatusForbidden,
	
	// 404 Not Found
	ErrorCodeNotFound:           http.StatusNotFound,
	ErrorCodeAdapterNotFound:    http.StatusNotFound,
	ErrorCodeConfigNotFound:     http.StatusNotFound,
	
	// 409 Conflict
	ErrorCodeConflict:           http.StatusConflict,
	ErrorCodeResourceExists:     http.StatusConflict,
	ErrorCodeResourceLocked:     http.StatusConflict,
	
	// 422 Unprocessable Entity
	ErrorCodeAdapterDisabled:    http.StatusUnprocessableEntity,
	
	// 429 Too Many Requests
	ErrorCodeRateLimitExceeded:  http.StatusTooManyRequests,
	ErrorCodeTooManyRequests:    http.StatusTooManyRequests,
	
	// 500 Internal Server Error
	ErrorCodeInternalError:      http.StatusInternalServerError,
	ErrorCodeHardwareFailure:    http.StatusInternalServerError,
	ErrorCodeDoorOperationFailed: http.StatusInternalServerError,
	ErrorCodeConfigUpdateFailed: http.StatusInternalServerError,
	ErrorCodeDatabaseError:      http.StatusInternalServerError,
	ErrorCodeStorageError:       http.StatusInternalServerError,
	ErrorCodeDataCorruption:     http.StatusInternalServerError,
	
	// 503 Service Unavailable
	ErrorCodeServiceUnavailable: http.StatusServiceUnavailable,
	ErrorCodeCircuitBreakerOpen: http.StatusServiceUnavailable,
	
	// 504 Gateway Timeout
	ErrorCodeTimeout:            http.StatusGatewayTimeout,
}

// GetHTTPStatus returns the appropriate HTTP status code for an error code
func (ec ErrorCode) GetHTTPStatus() int {
	if status, exists := HTTPStatusMapping[ec]; exists {
		return status
	}
	return http.StatusInternalServerError // Default to 500
}

// NewErrorResponse creates a standardized error response
func NewErrorResponse(code ErrorCode, message string, r *http.Request, requestID string) *ErrorResponse {
	response := &ErrorResponse{
		Error:     "true",
		Code:      string(code),
		Message:   message,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
		Status:    code.GetHTTPStatus(),
	}
	
	if r != nil {
		response.Path = r.URL.Path
		response.Method = r.Method
	}
	
	return response
}

// AddDetail adds a detail to the error response
func (er *ErrorResponse) AddDetail(key, value string) *ErrorResponse {
	if er.Details == nil {
		er.Details = make(map[string]string)
	}
	er.Details[key] = value
	return er
}

// DoorStatus constants
const (
	DoorStatusLocked   = "locked"
	DoorStatusUnlocked = "unlocked"
	DoorStatusUnknown  = "unknown"
)

// DeviceStatusResponse represents comprehensive device information
type DeviceStatusResponse struct {
	DeviceID      string                `json:"deviceId"`
	Status        string                `json:"status"`
	Timestamp     time.Time             `json:"timestamp"`
	Uptime        time.Duration         `json:"uptime"`
	Version       string                `json:"version"`
	QueueDepth    int                   `json:"queueDepth"`
	AdapterStatus []AdapterStatusInfo   `json:"adapterStatus"`
	Resources     SystemResourcesInfo   `json:"resources"`
	Tier          string                `json:"tier"`
	LastEventTime *time.Time            `json:"lastEventTime,omitempty"`
}

// AdapterStatusInfo represents adapter status information for API responses
type AdapterStatusInfo struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	LastEvent    time.Time `json:"lastEvent,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// SystemResourcesInfo represents system resource information for API responses
type SystemResourcesInfo struct {
	CPUCores     int     `json:"cpuCores"`
	MemoryGB     float64 `json:"memoryGB"`
	CPUUsage     float64 `json:"cpuUsage"`     // Percentage
	MemoryUsage  float64 `json:"memoryUsage"`  // Percentage
	DiskUsage    float64 `json:"diskUsage"`    // Percentage
	LastUpdated  time.Time `json:"lastUpdated"`
}

// DeviceMetricsResponse represents performance metrics for the device
type DeviceMetricsResponse struct {
	Timestamp        time.Time                  `json:"timestamp"`
	Uptime           time.Duration              `json:"uptime"`
	QueueMetrics     QueueMetricsInfo           `json:"queueMetrics"`
	AdapterMetrics   []AdapterMetricsInfo       `json:"adapterMetrics"`
	SystemMetrics    SystemMetricsInfo          `json:"systemMetrics"`
	PerformanceStats PerformanceStatsInfo       `json:"performanceStats"`
}

// QueueMetricsInfo represents queue-related metrics
type QueueMetricsInfo struct {
	QueueDepth      int       `json:"queueDepth"`
	PendingEvents   int       `json:"pendingEvents"`
	SentEvents      int64     `json:"sentEvents"`
	FailedEvents    int64     `json:"failedEvents"`
	LastSentAt      time.Time `json:"lastSentAt"`
	LastFailureAt   time.Time `json:"lastFailureAt"`
	OldestEventTime time.Time `json:"oldestEventTime"`
}

// AdapterMetricsInfo represents adapter-specific metrics
type AdapterMetricsInfo struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	EventCount     int64     `json:"eventCount"`
	ErrorCount     int64     `json:"errorCount"`
	LastEventTime  time.Time `json:"lastEventTime,omitempty"`
	LastErrorTime  time.Time `json:"lastErrorTime,omitempty"`
	ResponseTimeMs float64   `json:"responseTimeMs"`
}

// SystemMetricsInfo represents system-level metrics
type SystemMetricsInfo struct {
	CPUUsage       float64   `json:"cpuUsage"`
	MemoryUsage    float64   `json:"memoryUsage"`
	DiskUsage      float64   `json:"diskUsage"`
	NetworkRxBytes int64     `json:"networkRxBytes"`
	NetworkTxBytes int64     `json:"networkTxBytes"`
	LastUpdated    time.Time `json:"lastUpdated"`
}

// PerformanceStatsInfo represents performance statistics
type PerformanceStatsInfo struct {
	RequestsPerSecond    float64 `json:"requestsPerSecond"`
	AverageResponseTime  float64 `json:"averageResponseTime"`
	ErrorRate            float64 `json:"errorRate"`
	ThroughputEventsPerS float64 `json:"throughputEventsPerS"`
}

// HealthCheckResponse represents enhanced health check response
type HealthCheckResponse struct {
	Status        string                `json:"status"`
	Timestamp     time.Time             `json:"timestamp"`
	Version       string                `json:"version"`
	DeviceID      string                `json:"deviceId,omitempty"`
	Uptime        time.Duration         `json:"uptime"`
	QueueDepth    int                   `json:"queueDepth"`
	AdapterStatus []AdapterStatusInfo   `json:"adapterStatus"`
	Resources     SystemResourcesInfo   `json:"resources"`
	Tier          string                `json:"tier"`
	LastEventTime *time.Time            `json:"lastEventTime,omitempty"`
}

// ConfigResponse represents the current device configuration
type ConfigResponse struct {
	DeviceID          string                             `json:"deviceId"`
	ServerURL         string                             `json:"serverUrl"`
	Tier              string                             `json:"tier"`
	QueueMaxSize      int                                `json:"queueMaxSize"`
	HeartbeatInterval int                                `json:"heartbeatInterval"`
	UnlockDuration    int                                `json:"unlockDuration"`
	DatabasePath      string                             `json:"databasePath"`
	LogLevel          string                             `json:"logLevel"`
	LogFile           string                             `json:"logFile"`
	EnabledAdapters   []string                           `json:"enabledAdapters"`
	AdapterConfigs    map[string]map[string]interface{} `json:"adapterConfigs"`
	UpdatesEnabled    bool                               `json:"updatesEnabled"`
	APIServer         APIServerConfigResponse            `json:"apiServer"`
	Timestamp         time.Time                          `json:"timestamp"`
}

// APIServerConfigResponse represents API server configuration for responses
type APIServerConfigResponse struct {
	Enabled      bool                    `json:"enabled"`
	Port         int                     `json:"port"`
	Host         string                  `json:"host"`
	TLSEnabled   bool                    `json:"tlsEnabled"`
	ReadTimeout  int                     `json:"readTimeout"`
	WriteTimeout int                     `json:"writeTimeout"`
	IdleTimeout  int                     `json:"idleTimeout"`
	Auth         AuthConfigResponse      `json:"auth"`
	RateLimit    RateLimitConfigResponse `json:"rateLimit"`
	CORS         CORSConfigResponse      `json:"cors"`
	Security     SecurityConfigResponse  `json:"security"`
}

// AuthConfigResponse represents authentication configuration for responses (sensitive fields omitted)
type AuthConfigResponse struct {
	Enabled     bool     `json:"enabled"`
	TokenExpiry int      `json:"tokenExpiry"`
	AllowedIPs  []string `json:"allowedIps"`
	HasHMACKey  bool     `json:"hasHmacKey"`
	HasJWTKey   bool     `json:"hasJwtKey"`
	APIKeyCount int      `json:"apiKeyCount"`
}

// RateLimitConfigResponse represents rate limiting configuration for responses
type RateLimitConfigResponse struct {
	Enabled         bool `json:"enabled"`
	RequestsPerMin  int  `json:"requestsPerMinute"`
	BurstSize       int  `json:"burstSize"`
	WindowSize      int  `json:"windowSize"`
	CleanupInterval int  `json:"cleanupInterval"`
}

// CORSConfigResponse represents CORS configuration for responses
type CORSConfigResponse struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowedOrigins"`
	AllowedMethods   []string `json:"allowedMethods"`
	AllowedHeaders   []string `json:"allowedHeaders"`
	ExposedHeaders   []string `json:"exposedHeaders"`
	AllowCredentials bool     `json:"allowCredentials"`
	MaxAge           int      `json:"maxAge"`
}

// SecurityConfigResponse represents security configuration for responses
type SecurityConfigResponse struct {
	HSTSEnabled           bool   `json:"hstsEnabled"`
	HSTSMaxAge            int    `json:"hstsMaxAge"`
	HSTSIncludeSubdomains bool   `json:"hstsIncludeSubdomains"`
	CSPEnabled            bool   `json:"cspEnabled"`
	CSPDirective          string `json:"cspDirective"`
	FrameOptions          string `json:"frameOptions"`
	ContentTypeOptions    bool   `json:"contentTypeOptions"`
	XSSProtection         bool   `json:"xssProtection"`
	ReferrerPolicy        string `json:"referrerPolicy"`
}

// ConfigUpdateRequest represents a request to update device configuration
type ConfigUpdateRequest struct {
	Tier              *string                            `json:"tier,omitempty"`
	QueueMaxSize      *int                               `json:"queueMaxSize,omitempty"`
	HeartbeatInterval *int                               `json:"heartbeatInterval,omitempty"`
	UnlockDuration    *int                               `json:"unlockDuration,omitempty"`
	LogLevel          *string                            `json:"logLevel,omitempty"`
	LogFile           *string                            `json:"logFile,omitempty"`
	EnabledAdapters   []string                           `json:"enabledAdapters,omitempty"`
	AdapterConfigs    map[string]map[string]interface{} `json:"adapterConfigs,omitempty"`
	UpdatesEnabled    *bool                              `json:"updatesEnabled,omitempty"`
	APIServer         *APIServerConfigUpdateRequest      `json:"apiServer,omitempty"`
}

// APIServerConfigUpdateRequest represents API server configuration update request
type APIServerConfigUpdateRequest struct {
	Enabled      *bool                         `json:"enabled,omitempty"`
	Port         *int                          `json:"port,omitempty"`
	Host         *string                       `json:"host,omitempty"`
	TLSEnabled   *bool                         `json:"tlsEnabled,omitempty"`
	TLSCertFile  *string                       `json:"tlsCertFile,omitempty"`
	TLSKeyFile   *string                       `json:"tlsKeyFile,omitempty"`
	ReadTimeout  *int                          `json:"readTimeout,omitempty"`
	WriteTimeout *int                          `json:"writeTimeout,omitempty"`
	IdleTimeout  *int                          `json:"idleTimeout,omitempty"`
	Auth         *AuthConfigUpdateRequest      `json:"auth,omitempty"`
	RateLimit    *RateLimitConfigUpdateRequest `json:"rateLimit,omitempty"`
	CORS         *CORSConfigUpdateRequest      `json:"cors,omitempty"`
	Security     *SecurityConfigUpdateRequest  `json:"security,omitempty"`
}

// AuthConfigUpdateRequest represents authentication configuration update request
type AuthConfigUpdateRequest struct {
	Enabled     *bool    `json:"enabled,omitempty"`
	HMACSecret  *string  `json:"hmacSecret,omitempty"`
	JWTSecret   *string  `json:"jwtSecret,omitempty"`
	APIKeys     []string `json:"apiKeys,omitempty"`
	TokenExpiry *int     `json:"tokenExpiry,omitempty"`
	AllowedIPs  []string `json:"allowedIps,omitempty"`
}

// RateLimitConfigUpdateRequest represents rate limiting configuration update request
type RateLimitConfigUpdateRequest struct {
	Enabled         *bool `json:"enabled,omitempty"`
	RequestsPerMin  *int  `json:"requestsPerMinute,omitempty"`
	BurstSize       *int  `json:"burstSize,omitempty"`
	WindowSize      *int  `json:"windowSize,omitempty"`
	CleanupInterval *int  `json:"cleanupInterval,omitempty"`
}

// CORSConfigUpdateRequest represents CORS configuration update request
type CORSConfigUpdateRequest struct {
	Enabled          *bool    `json:"enabled,omitempty"`
	AllowedOrigins   []string `json:"allowedOrigins,omitempty"`
	AllowedMethods   []string `json:"allowedMethods,omitempty"`
	AllowedHeaders   []string `json:"allowedHeaders,omitempty"`
	ExposedHeaders   []string `json:"exposedHeaders,omitempty"`
	AllowCredentials *bool    `json:"allowCredentials,omitempty"`
	MaxAge           *int     `json:"maxAge,omitempty"`
}

// SecurityConfigUpdateRequest represents security configuration update request
type SecurityConfigUpdateRequest struct {
	HSTSEnabled           *bool   `json:"hstsEnabled,omitempty"`
	HSTSMaxAge            *int    `json:"hstsMaxAge,omitempty"`
	HSTSIncludeSubdomains *bool   `json:"hstsIncludeSubdomains,omitempty"`
	CSPEnabled            *bool   `json:"cspEnabled,omitempty"`
	CSPDirective          *string `json:"cspDirective,omitempty"`
	FrameOptions          *string `json:"frameOptions,omitempty"`
	ContentTypeOptions    *bool   `json:"contentTypeOptions,omitempty"`
	XSSProtection         *bool   `json:"xssProtection,omitempty"`
	ReferrerPolicy        *string `json:"referrerPolicy,omitempty"`
}

// Validate validates the configuration update request
func (r *ConfigUpdateRequest) Validate() error {
	if r.Tier != nil {
		if *r.Tier != "lite" && *r.Tier != "normal" && *r.Tier != "full" {
			return fmt.Errorf("tier must be one of: lite, normal, full")
		}
	}
	
	if r.QueueMaxSize != nil && *r.QueueMaxSize <= 0 {
		return fmt.Errorf("queueMaxSize must be positive")
	}
	
	if r.HeartbeatInterval != nil && *r.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeatInterval must be positive")
	}
	
	if r.UnlockDuration != nil && *r.UnlockDuration <= 0 {
		return fmt.Errorf("unlockDuration must be positive")
	}
	
	if r.LogLevel != nil {
		validLogLevels := map[string]bool{
			"debug": true, "info": true, "warn": true, "error": true,
		}
		if !validLogLevels[*r.LogLevel] {
			return fmt.Errorf("logLevel must be one of: debug, info, warn, error")
		}
	}
	
	// Validate API server configuration
	if r.APIServer != nil {
		if r.APIServer.Port != nil && (*r.APIServer.Port < 1 || *r.APIServer.Port > 65535) {
			return fmt.Errorf("apiServer.port must be between 1 and 65535")
		}
		
		if r.APIServer.ReadTimeout != nil && *r.APIServer.ReadTimeout <= 0 {
			return fmt.Errorf("apiServer.readTimeout must be positive")
		}
		
		if r.APIServer.WriteTimeout != nil && *r.APIServer.WriteTimeout <= 0 {
			return fmt.Errorf("apiServer.writeTimeout must be positive")
		}
		
		if r.APIServer.IdleTimeout != nil && *r.APIServer.IdleTimeout <= 0 {
			return fmt.Errorf("apiServer.idleTimeout must be positive")
		}
		
		// Validate auth configuration
		if r.APIServer.Auth != nil {
			if r.APIServer.Auth.TokenExpiry != nil && *r.APIServer.Auth.TokenExpiry <= 0 {
				return fmt.Errorf("apiServer.auth.tokenExpiry must be positive")
			}
		}
		
		// Validate rate limit configuration
		if r.APIServer.RateLimit != nil {
			if r.APIServer.RateLimit.RequestsPerMin != nil && *r.APIServer.RateLimit.RequestsPerMin <= 0 {
				return fmt.Errorf("apiServer.rateLimit.requestsPerMinute must be positive")
			}
			
			if r.APIServer.RateLimit.BurstSize != nil && *r.APIServer.RateLimit.BurstSize <= 0 {
				return fmt.Errorf("apiServer.rateLimit.burstSize must be positive")
			}
			
			if r.APIServer.RateLimit.WindowSize != nil && *r.APIServer.RateLimit.WindowSize <= 0 {
				return fmt.Errorf("apiServer.rateLimit.windowSize must be positive")
			}
			
			if r.APIServer.RateLimit.CleanupInterval != nil && *r.APIServer.RateLimit.CleanupInterval <= 0 {
				return fmt.Errorf("apiServer.rateLimit.cleanupInterval must be positive")
			}
		}
		
		// Validate CORS configuration
		if r.APIServer.CORS != nil {
			if r.APIServer.CORS.MaxAge != nil && *r.APIServer.CORS.MaxAge < 0 {
				return fmt.Errorf("apiServer.cors.maxAge must be non-negative")
			}
		}
		
		// Validate security configuration
		if r.APIServer.Security != nil {
			if r.APIServer.Security.HSTSMaxAge != nil && *r.APIServer.Security.HSTSMaxAge <= 0 {
				return fmt.Errorf("apiServer.security.hstsMaxAge must be positive")
			}
			
			if r.APIServer.Security.FrameOptions != nil {
				validFrameOptions := map[string]bool{
					"DENY": true, "SAMEORIGIN": true, "ALLOW-FROM": true,
				}
				if !validFrameOptions[*r.APIServer.Security.FrameOptions] {
					return fmt.Errorf("apiServer.security.frameOptions must be one of: DENY, SAMEORIGIN, ALLOW-FROM")
				}
			}
		}
	}
	
	return nil
}

// ConfigUpdateResponse represents the response to a configuration update
type ConfigUpdateResponse struct {
	Success         bool      `json:"success"`
	Message         string    `json:"message"`
	UpdatedFields   []string  `json:"updatedFields"`
	RequiresRestart bool      `json:"requiresRestart"`
	Timestamp       time.Time `json:"timestamp"`
	RequestID       string    `json:"requestId,omitempty"`
}

// ConfigReloadRequest represents a request to reload configuration
type ConfigReloadRequest struct {
	Force  bool   `json:"force,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// ConfigReloadResponse represents the response to a configuration reload
type ConfigReloadResponse struct {
	Success       bool      `json:"success"`
	Message       string    `json:"message"`
	ReloadedFrom  string    `json:"reloadedFrom"`
	ChangedFields []string  `json:"changedFields"`
	Timestamp     time.Time `json:"timestamp"`
	RequestID     string    `json:"requestId,omitempty"`
}

// EventQueryRequest represents a request to query historical events
type EventQueryRequest struct {
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	EventType    string     `json:"eventType,omitempty"`
	UserID       string     `json:"userId,omitempty"`
	IsSimulated  *bool      `json:"isSimulated,omitempty"`
	Limit        int        `json:"limit,omitempty" validate:"max=1000"`
	Offset       int        `json:"offset,omitempty"`
	SortBy       string     `json:"sortBy,omitempty"`     // "timestamp", "eventType", "userId"
	SortOrder    string     `json:"sortOrder,omitempty"`  // "asc", "desc"
}

// Validate validates the event query request
func (r *EventQueryRequest) Validate() error {
	if r.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}
	if r.Limit > 1000 {
		return fmt.Errorf("limit must not exceed 1000")
	}
	if r.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	
	// Set default limit if not specified
	if r.Limit == 0 {
		r.Limit = 100
	}
	
	// Validate event type if specified
	if r.EventType != "" {
		validEventTypes := map[string]bool{
			"entry": true, "exit": true, "denied": true,
		}
		if !validEventTypes[r.EventType] {
			return fmt.Errorf("eventType must be one of: entry, exit, denied")
		}
	}
	
	// Validate sort parameters
	if r.SortBy != "" {
		validSortFields := map[string]bool{
			"timestamp": true, "eventType": true, "userId": true,
		}
		if !validSortFields[r.SortBy] {
			return fmt.Errorf("sortBy must be one of: timestamp, eventType, userId")
		}
	} else {
		r.SortBy = "timestamp" // Default sort by timestamp
	}
	
	if r.SortOrder != "" {
		if r.SortOrder != "asc" && r.SortOrder != "desc" {
			return fmt.Errorf("sortOrder must be either 'asc' or 'desc'")
		}
	} else {
		r.SortOrder = "desc" // Default to descending (newest first)
	}
	
	// Validate time range
	if r.StartTime != nil && r.EndTime != nil {
		if r.StartTime.After(*r.EndTime) {
			return fmt.Errorf("startTime must be before endTime")
		}
	}
	
	return nil
}

// EventResponse represents an event in API responses
type EventResponse struct {
	ID             int64                  `json:"id"`
	EventID        string                 `json:"eventId"`
	ExternalUserID string                 `json:"externalUserId"`
	InternalUserID string                 `json:"internalUserId,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	EventType      string                 `json:"eventType"`
	IsSimulated    bool                   `json:"isSimulated"`
	DeviceID       string                 `json:"deviceId"`
	RawData        map[string]interface{} `json:"rawData,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	SentAt         *time.Time             `json:"sentAt,omitempty"`
	RetryCount     int                    `json:"retryCount"`
}

// EventsResponse represents the response to an events query
type EventsResponse struct {
	Events     []EventResponse `json:"events"`
	Total      int64           `json:"total"`
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
	HasMore    bool            `json:"hasMore"`
	Timestamp  time.Time       `json:"timestamp"`
	RequestID  string          `json:"requestId,omitempty"`
}

// EventStatsResponse represents event statistics
type EventStatsResponse struct {
	TotalEvents      int64                    `json:"totalEvents"`
	EventsByType     map[string]int64         `json:"eventsByType"`
	EventsByHour     map[string]int64         `json:"eventsByHour"`
	EventsByDay      map[string]int64         `json:"eventsByDay"`
	PendingEvents    int64                    `json:"pendingEvents"`
	SentEvents       int64                    `json:"sentEvents"`
	FailedEvents     int64                    `json:"failedEvents"`
	UniqueUsers      int64                    `json:"uniqueUsers"`
	SimulatedEvents  int64                    `json:"simulatedEvents"`
	OldestEventTime  *time.Time               `json:"oldestEventTime,omitempty"`
	NewestEventTime  *time.Time               `json:"newestEventTime,omitempty"`
	AveragePerHour   float64                  `json:"averagePerHour"`
	AveragePerDay    float64                  `json:"averagePerDay"`
	Timestamp        time.Time                `json:"timestamp"`
	RequestID        string                   `json:"requestId,omitempty"`
}

// EventClearRequest represents a request to clear events
type EventClearRequest struct {
	OlderThan   *time.Time `json:"olderThan,omitempty"`
	EventType   string     `json:"eventType,omitempty"`
	OnlySent    bool       `json:"onlySent,omitempty"`
	OnlyFailed  bool       `json:"onlyFailed,omitempty"`
	Confirm     bool       `json:"confirm"`
	Reason      string     `json:"reason,omitempty"`
}

// Validate validates the event clear request
func (r *EventClearRequest) Validate() error {
	if !r.Confirm {
		return fmt.Errorf("confirm must be true to proceed with event deletion")
	}
	
	// Validate event type if specified
	if r.EventType != "" {
		validEventTypes := map[string]bool{
			"entry": true, "exit": true, "denied": true,
		}
		if !validEventTypes[r.EventType] {
			return fmt.Errorf("eventType must be one of: entry, exit, denied")
		}
	}
	
	// Validate that onlySent and onlyFailed are not both true
	if r.OnlySent && r.OnlyFailed {
		return fmt.Errorf("onlySent and onlyFailed cannot both be true")
	}
	
	return nil
}

// EventClearResponse represents the response to an event clear request
type EventClearResponse struct {
	Success       bool      `json:"success"`
	Message       string    `json:"message"`
	DeletedCount  int64     `json:"deletedCount"`
	Criteria      string    `json:"criteria"`
	Timestamp     time.Time `json:"timestamp"`
	RequestID     string    `json:"requestId,omitempty"`
}

// AdapterInfo represents basic adapter information
type AdapterInfo struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	IsHealthy    bool      `json:"isHealthy"`
	LastEvent    time.Time `json:"lastEvent,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// AdaptersResponse represents the response to GET /api/v1/adapters
type AdaptersResponse struct {
	Adapters        []AdapterInfo `json:"adapters"`
	EnabledAdapters []string      `json:"enabledAdapters"`
	ActiveAdapters  []string      `json:"activeAdapters"`
	TotalCount      int           `json:"totalCount"`
	ActiveCount     int           `json:"activeCount"`
	Timestamp       time.Time     `json:"timestamp"`
}

// AdapterDetailResponse represents the response to GET /api/v1/adapters/{name}
type AdapterDetailResponse struct {
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	IsEnabled    bool                   `json:"isEnabled"`
	IsActive     bool                   `json:"isActive"`
	IsHealthy    bool                   `json:"isHealthy"`
	LastEvent    time.Time              `json:"lastEvent,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	RequestID    string                 `json:"requestId,omitempty"`
}

// AdapterEnableRequest represents a request to enable an adapter
type AdapterEnableRequest struct {
	Reason string `json:"reason,omitempty"`
}

// AdapterEnableResponse represents the response to POST /api/v1/adapters/{name}/enable
type AdapterEnableResponse struct {
	Success         bool      `json:"success"`
	Message         string    `json:"message"`
	Adapter         string    `json:"adapter"`
	Enabled         bool      `json:"enabled"`
	RequiresRestart bool      `json:"requiresRestart"`
	Timestamp       time.Time `json:"timestamp"`
	RequestID       string    `json:"requestId,omitempty"`
}

// AdapterDisableRequest represents a request to disable an adapter
type AdapterDisableRequest struct {
	Reason string `json:"reason,omitempty"`
}

// AdapterDisableResponse represents the response to POST /api/v1/adapters/{name}/disable
type AdapterDisableResponse struct {
	Success         bool      `json:"success"`
	Message         string    `json:"message"`
	Adapter         string    `json:"adapter"`
	Enabled         bool      `json:"enabled"`
	RequiresRestart bool      `json:"requiresRestart"`
	Timestamp       time.Time `json:"timestamp"`
	RequestID       string    `json:"requestId,omitempty"`
}

// AdapterConfigUpdateRequest represents a request to update adapter configuration
type AdapterConfigUpdateRequest struct {
	Config map[string]interface{} `json:"config"`
	Reason string                 `json:"reason,omitempty"`
}

// Validate validates the adapter config update request
func (r *AdapterConfigUpdateRequest) Validate() error {
	if r.Config == nil {
		return fmt.Errorf("config is required")
	}
	
	if len(r.Config) == 0 {
		return fmt.Errorf("config cannot be empty")
	}
	
	return nil
}

// AdapterConfigUpdateResponse represents the response to PUT /api/v1/adapters/{name}/config
type AdapterConfigUpdateResponse struct {
	Success         bool                   `json:"success"`
	Message         string                 `json:"message"`
	Adapter         string                 `json:"adapter"`
	Config          map[string]interface{} `json:"config"`
	RequiresRestart bool                   `json:"requiresRestart"`
	Timestamp       time.Time              `json:"timestamp"`
	RequestID       string                 `json:"requestId,omitempty"`
}

// WebSocket Models

// WebSocketStatusResponse represents WebSocket connection status for API responses
type WebSocketStatusResponse struct {
	Enabled           bool                     `json:"enabled"`
	ConnectionCount   int                      `json:"connectionCount"`
	MaxConnections    int                      `json:"maxConnections"`
	TotalConnections  int64                    `json:"totalConnections"`
	MessagesSent      int64                    `json:"messagesSent"`
	MessagesReceived  int64                    `json:"messagesReceived"`
	Connections       []WebSocketConnectionInfo `json:"connections,omitempty"`
	Timestamp         time.Time                `json:"timestamp"`
}

// WebSocketConnectionInfo represents information about a WebSocket connection for API responses
type WebSocketConnectionInfo struct {
	ID            string                 `json:"id"`
	RemoteAddr    string                 `json:"remoteAddr"`
	UserAgent     string                 `json:"userAgent"`
	ConnectedAt   time.Time              `json:"connectedAt"`
	LastPing      time.Time              `json:"lastPing"`
	MessagesSent  int64                  `json:"messagesSent"`
	Filters       WebSocketFiltersInfo   `json:"filters"`
	Auth          *WebSocketAuthInfo     `json:"auth,omitempty"`
}

// WebSocketFiltersInfo represents WebSocket filters for API responses
type WebSocketFiltersInfo struct {
	EventTypes    []string `json:"eventTypes,omitempty"`
	DeviceID      string   `json:"deviceId,omitempty"`
	UserID        string   `json:"userId,omitempty"`
	MinSeverity   string   `json:"minSeverity,omitempty"`
	IncludeSystem bool     `json:"includeSystem"`
}

// WebSocketAuthInfo represents WebSocket authentication info for API responses
type WebSocketAuthInfo struct {
	UserID    string     `json:"userId"`
	Method    string     `json:"method"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// WebSocketEventRequest represents a request to broadcast an event via WebSocket
type WebSocketEventRequest struct {
	EventType string      `json:"eventType"`
	Data      interface{} `json:"data"`
	TargetIDs []string    `json:"targetIds,omitempty"` // Specific connection IDs to target
}

// Validate validates the WebSocket event request
func (r *WebSocketEventRequest) Validate() error {
	if r.EventType == "" {
		return fmt.Errorf("eventType is required")
	}
	
	if r.Data == nil {
		return fmt.Errorf("data is required")
	}
	
	// Validate event type format
	validEventTypes := map[string]bool{
		"door_unlock":     true,
		"door_lock":       true,
		"door_status":     true,
		"device_status":   true,
		"config_change":   true,
		"adapter_status":  true,
		"system_status":   true,
		"health_check":    true,
		"event_created":   true,
		"event_sent":      true,
		"event_failed":    true,
		"queue_status":    true,
		"error":           true,
		"warning":         true,
		"info":            true,
	}
	
	if !validEventTypes[r.EventType] {
		return fmt.Errorf("invalid eventType: %s", r.EventType)
	}
	
	return nil
}

// WebSocketEventResponse represents the response to a WebSocket event broadcast request
type WebSocketEventResponse struct {
	Success         bool      `json:"success"`
	Message         string    `json:"message"`
	EventType       string    `json:"eventType"`
	EventID         string    `json:"eventId"`
	ConnectionsSent int       `json:"connectionsSent"`
	Timestamp       time.Time `json:"timestamp"`
	RequestID       string    `json:"requestId,omitempty"`
}