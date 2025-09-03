package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/queue"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// AdapterRegistry interface for getting adapters
type AdapterRegistry interface {
	GetAllAdapters() []adapters.HardwareAdapter
	GetAdapter(name string) (adapters.HardwareAdapter, error)
	GetActiveAdapters() []adapters.HardwareAdapter
}

// DoorController interface for door operations
type DoorController interface {
	UnlockDoor(ctx context.Context, adapterName string, durationMs int) error
	GetStats() map[string]interface{}
}

// HealthMonitor interface for health monitoring
type HealthMonitor interface {
	GetCurrentHealth() SystemHealth
	UpdateHealth(ctx context.Context) error
}

// QueueManager interface for queue operations
type QueueManager interface {
	GetQueueDepth(ctx context.Context) (int, error)
	GetStats(ctx context.Context) (QueueStats, error)
	QueryEvents(ctx context.Context, filter queue.EventQueryFilter) ([]queue.QueuedEvent, int64, error)
	GetEventStats(ctx context.Context) (queue.EventStatistics, error)
	ClearEvents(ctx context.Context, criteria queue.EventClearCriteria) (int64, error)
}

// TierDetector interface for tier detection
type TierDetector interface {
	GetCurrentTier() Tier
	GetCurrentResources() SystemResources
}

// ConfigManager interface for configuration management
type ConfigManager interface {
	GetCurrentConfig() *config.Config
	UpdateConfig(updates *ConfigUpdateRequest) (*ConfigUpdateResponse, error)
	ReloadConfig(force bool) (*ConfigReloadResponse, error)
}

// SystemHealth represents the complete health information
type SystemHealth struct {
	Status        string                `json:"status"`
	Timestamp     time.Time             `json:"timestamp"`
	QueueDepth    int                   `json:"queueDepth"`
	AdapterStatus []AdapterStatus       `json:"adapterStatus"`
	Resources     SystemResources       `json:"resources"`
	Tier          Tier                  `json:"tier"`
	LastEventTime *time.Time            `json:"lastEventTime,omitempty"`
	Uptime        time.Duration         `json:"uptime"`
	Version       string                `json:"version"`
	DeviceID      string                `json:"deviceId,omitempty"`
}

// AdapterStatus represents adapter status
type AdapterStatus struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	LastEvent    time.Time `json:"lastEvent,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// SystemResources represents system resource information
type SystemResources struct {
	CPUCores     int       `json:"cpuCores"`
	MemoryGB     float64   `json:"memoryGB"`
	CPUUsage     float64   `json:"cpuUsage"`
	MemoryUsage  float64   `json:"memoryUsage"`
	DiskUsage    float64   `json:"diskUsage"`
	LastUpdated  time.Time `json:"lastUpdated"`
}

// QueueStats represents queue statistics
type QueueStats struct {
	QueueDepth      int       `json:"queueDepth"`
	PendingEvents   int       `json:"pendingEvents"`
	SentEvents      int64     `json:"sentEvents"`
	FailedEvents    int64     `json:"failedEvents"`
	LastSentAt      time.Time `json:"lastSentAt"`
	LastFailureAt   time.Time `json:"lastFailureAt"`
	OldestEventTime time.Time `json:"oldestEventTime"`
}

// Tier represents performance tier
type Tier string

// Handlers contains all HTTP handlers for the API
type Handlers struct {
	config          *config.Config
	logger          *logrus.Logger
	adapterRegistry AdapterRegistry
	doorController  DoorController
	healthMonitor   HealthMonitor
	queueManager    QueueManager
	tierDetector    TierDetector
	configManager   ConfigManager
	wsManager       *WebSocketManager
	startTime       time.Time
	version         string
	deviceID        string
}

// NewHandlers creates a new handlers instance
func NewHandlers(cfg *config.Config, logger *logrus.Logger, adapterRegistry AdapterRegistry, doorController DoorController, healthMonitor HealthMonitor, queueManager QueueManager, tierDetector TierDetector, configManager ConfigManager, version, deviceID string) *Handlers {
	// Create WebSocket manager
	wsManager := NewWebSocketManager(logger)
	
	return &Handlers{
		config:          cfg,
		logger:          logger,
		adapterRegistry: adapterRegistry,
		doorController:  doorController,
		healthMonitor:   healthMonitor,
		queueManager:    queueManager,
		tierDetector:    tierDetector,
		configManager:   configManager,
		wsManager:       wsManager,
		startTime:       time.Now(),
		version:         version,
		deviceID:        deviceID,
	}
}

// HealthCheck handles GET /api/v1/health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Update health status if health monitor is available
	if h.healthMonitor != nil {
		if err := h.healthMonitor.UpdateHealth(ctx); err != nil {
			h.logger.WithError(err).Error("Failed to update health status")
		}
	}
	
	// Get comprehensive health information
	var response HealthCheckResponse
	
	if h.healthMonitor != nil {
		// Get health from health monitor
		health := h.healthMonitor.GetCurrentHealth()
		
		// Convert adapter statuses
		adapterStatuses := make([]AdapterStatusInfo, len(health.AdapterStatus))
		for i, status := range health.AdapterStatus {
			adapterStatuses[i] = AdapterStatusInfo{
				Name:         status.Name,
				Status:       status.Status,
				LastEvent:    status.LastEvent,
				ErrorMessage: status.ErrorMessage,
				UpdatedAt:    status.UpdatedAt,
			}
		}
		
		response = HealthCheckResponse{
			Status:        health.Status,
			Timestamp:     health.Timestamp,
			Version:       health.Version,
			DeviceID:      health.DeviceID,
			Uptime:        health.Uptime,
			QueueDepth:    health.QueueDepth,
			AdapterStatus: adapterStatuses,
			Resources: SystemResourcesInfo{
				CPUCores:    health.Resources.CPUCores,
				MemoryGB:    health.Resources.MemoryGB,
				CPUUsage:    health.Resources.CPUUsage,
				MemoryUsage: health.Resources.MemoryUsage,
				DiskUsage:   health.Resources.DiskUsage,
				LastUpdated: health.Resources.LastUpdated,
			},
			Tier:          string(health.Tier),
			LastEventTime: health.LastEventTime,
		}
	} else {
		// Fallback to basic health check
		response = HealthCheckResponse{
			Status:    "healthy",
			Timestamp: time.Now().UTC(),
			Version:   h.version,
			DeviceID:  h.deviceID,
			Uptime:    time.Since(h.startTime),
		}
	}
	
	// Determine HTTP status code based on health status
	statusCode := http.StatusOK
	switch response.Status {
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	case "degraded":
		statusCode = http.StatusOK // Still OK, but degraded
	default:
		statusCode = http.StatusOK
	}
	
	h.writeJSONResponse(w, response, statusCode)
}

// UnlockDoor handles POST /api/v1/door/unlock
func (h *Handlers) UnlockDoor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := h.generateRequestID()
	
	// Parse request body
	var req DoorUnlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode unlock request")
		h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
		return
	}
	
	// Use default duration if not specified
	if req.DurationMs == 0 {
		req.DurationMs = h.config.UnlockDuration
		if req.DurationMs == 0 {
			req.DurationMs = 3000 // 3 seconds default
		}
	}
	
	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.WithError(err).Error("Invalid unlock request")
		h.writeErrorResponseLegacy(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR", requestID)
		return
	}
	
	// Log the unlock request
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"durationMs":  req.DurationMs,
		"adapter":     req.Adapter,
		"reason":      req.Reason,
		"requestedBy": req.RequestedBy,
		"clientIP":    getClientIP(r),
	}).Info("Door unlock requested")
	
	// Perform unlock operation
	if err := h.doorController.UnlockDoor(ctx, req.Adapter, req.DurationMs); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to unlock door")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to unlock door: %v", err), http.StatusInternalServerError, "UNLOCK_FAILED", requestID)
		return
	}
	
	// Determine which adapter was used
	adapterName := req.Adapter
	if adapterName == "" {
		if activeAdapters := h.adapterRegistry.GetActiveAdapters(); len(activeAdapters) > 0 {
			adapterName = activeAdapters[0].Name()
		}
	}
	
	// Return success response
	response := DoorUnlockResponse{
		Success:   true,
		Message:   "Door unlocked successfully",
		Duration:  req.DurationMs,
		Adapter:   adapterName,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId": requestID,
		"adapter":   adapterName,
		"duration":  req.DurationMs,
	}).Info("Door unlock completed successfully")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// LockDoor handles POST /api/v1/door/lock
func (h *Handlers) LockDoor(w http.ResponseWriter, r *http.Request) {
	requestID := h.generateRequestID()
	
	// Parse request body (optional for lock)
	var req DoorLockRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.WithError(err).Error("Failed to decode lock request")
			h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
			return
		}
	}
	
	// Log the lock request
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"adapter":     req.Adapter,
		"reason":      req.Reason,
		"requestedBy": req.RequestedBy,
		"clientIP":    getClientIP(r),
	}).Info("Door lock requested")
	
	// Note: Most door systems don't have explicit lock functionality
	// They automatically lock after the unlock duration expires
	// This endpoint serves as a confirmation that the door should be locked
	
	// Determine which adapter would be used
	adapterName := req.Adapter
	if adapterName == "" {
		if activeAdapters := h.adapterRegistry.GetActiveAdapters(); len(activeAdapters) > 0 {
			adapterName = activeAdapters[0].Name()
		}
	}
	
	// Return success response (door locking is typically automatic)
	response := DoorLockResponse{
		Success:   true,
		Message:   "Door lock confirmed - door will lock automatically",
		Adapter:   adapterName,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId": requestID,
		"adapter":   adapterName,
	}).Info("Door lock request processed")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// DoorStatus handles GET /api/v1/door/status
func (h *Handlers) DoorStatus(w http.ResponseWriter, r *http.Request) {
	// Get door controller statistics
	stats := h.doorController.GetStats()
	
	// Get active adapters
	activeAdapters := h.adapterRegistry.GetActiveAdapters()
	adapterNames := make([]string, len(activeAdapters))
	for i, adapter := range activeAdapters {
		adapterNames[i] = adapter.Name()
	}
	
	// Determine door status
	// Since most door systems don't provide real-time lock status,
	// we infer the status based on recent unlock activity
	status := DoorStatusLocked
	isLocked := true
	
	if lastUnlockTime, ok := stats["lastUnlockTime"].(time.Time); ok && !lastUnlockTime.IsZero() {
		// Check if door might still be unlocked based on default unlock duration
		unlockDuration := time.Duration(h.config.UnlockDuration) * time.Millisecond
		if time.Since(lastUnlockTime) < unlockDuration {
			status = DoorStatusUnlocked
			isLocked = false
		}
	}
	
	// If no adapters are active, status is unknown
	if len(activeAdapters) == 0 {
		status = DoorStatusUnknown
		isLocked = false // Can't determine lock status without adapters
	}
	
	// Build response
	response := DoorStatusResponse{
		IsLocked:       isLocked,
		Status:         status,
		UnlockCount:    getInt64FromStats(stats, "unlockCount"),
		ActiveAdapters: adapterNames,
		Timestamp:      time.Now().UTC(),
	}
	
	// Add last unlock time if available
	if lastUnlockTime, ok := stats["lastUnlockTime"].(time.Time); ok && !lastUnlockTime.IsZero() {
		response.LastUnlockTime = &lastUnlockTime
	}
	
	h.logger.WithFields(logrus.Fields{
		"status":         status,
		"isLocked":       isLocked,
		"unlockCount":    response.UnlockCount,
		"activeAdapters": len(activeAdapters),
	}).Debug("Door status requested")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// DeviceStatus handles GET /api/v1/status
func (h *Handlers) DeviceStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	h.logger.Debug("Device status requested")
	
	// Get queue depth
	var queueDepth int
	if h.queueManager != nil {
		if depth, err := h.queueManager.GetQueueDepth(ctx); err != nil {
			h.logger.WithError(err).Error("Failed to get queue depth")
			queueDepth = -1 // Indicate error
		} else {
			queueDepth = depth
		}
	}
	
	// Get adapter statuses
	var adapterStatuses []AdapterStatusInfo
	if h.adapterRegistry != nil {
		adapters := h.adapterRegistry.GetAllAdapters()
		adapterStatuses = make([]AdapterStatusInfo, len(adapters))
		for i, adapter := range adapters {
			status := adapter.GetStatus()
			adapterStatuses[i] = AdapterStatusInfo{
				Name:         status.Name,
				Status:       status.Status,
				LastEvent:    status.LastEvent,
				ErrorMessage: status.ErrorMessage,
				UpdatedAt:    status.UpdatedAt,
			}
		}
	}
	
	// Get system resources and tier
	var resources SystemResourcesInfo
	var currentTier string
	if h.tierDetector != nil {
		sysRes := h.tierDetector.GetCurrentResources()
		resources = SystemResourcesInfo{
			CPUCores:    sysRes.CPUCores,
			MemoryGB:    sysRes.MemoryGB,
			CPUUsage:    sysRes.CPUUsage,
			MemoryUsage: sysRes.MemoryUsage,
			DiskUsage:   sysRes.DiskUsage,
			LastUpdated: sysRes.LastUpdated,
		}
		currentTier = string(h.tierDetector.GetCurrentTier())
	}
	
	// Get last event time from queue stats
	var lastEventTime *time.Time
	if h.queueManager != nil {
		if stats, err := h.queueManager.GetStats(ctx); err == nil {
			if !stats.LastSentAt.IsZero() {
				lastEventTime = &stats.LastSentAt
			}
		}
	}
	
	// Determine overall device status
	deviceStatus := "healthy"
	if queueDepth < 0 {
		deviceStatus = "unhealthy"
	} else if len(adapterStatuses) == 0 {
		deviceStatus = "degraded"
	} else {
		// Check if any adapters are in error state
		errorCount := 0
		for _, status := range adapterStatuses {
			if status.Status == "error" {
				errorCount++
			}
		}
		if errorCount == len(adapterStatuses) {
			deviceStatus = "unhealthy"
		} else if errorCount > 0 {
			deviceStatus = "degraded"
		}
	}
	
	// Build response
	response := DeviceStatusResponse{
		DeviceID:      h.deviceID,
		Status:        deviceStatus,
		Timestamp:     time.Now().UTC(),
		Uptime:        time.Since(h.startTime),
		Version:       h.version,
		QueueDepth:    queueDepth,
		AdapterStatus: adapterStatuses,
		Resources:     resources,
		Tier:          currentTier,
		LastEventTime: lastEventTime,
	}
	
	h.logger.WithFields(logrus.Fields{
		"deviceId":       response.DeviceID,
		"status":         response.Status,
		"queueDepth":     response.QueueDepth,
		"adapterCount":   len(response.AdapterStatus),
		"tier":           response.Tier,
	}).Debug("Device status response prepared")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// DeviceMetrics handles GET /api/v1/metrics
func (h *Handlers) DeviceMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	h.logger.Debug("Device metrics requested")
	
	// Get queue metrics
	var queueMetrics QueueMetricsInfo
	if h.queueManager != nil {
		if stats, err := h.queueManager.GetStats(ctx); err != nil {
			h.logger.WithError(err).Error("Failed to get queue stats")
		} else {
			queueMetrics = QueueMetricsInfo{
				QueueDepth:      stats.QueueDepth,
				PendingEvents:   stats.PendingEvents,
				SentEvents:      stats.SentEvents,
				FailedEvents:    stats.FailedEvents,
				LastSentAt:      stats.LastSentAt,
				LastFailureAt:   stats.LastFailureAt,
				OldestEventTime: stats.OldestEventTime,
			}
		}
	}
	
	// Get adapter metrics
	var adapterMetrics []AdapterMetricsInfo
	if h.adapterRegistry != nil {
		adapters := h.adapterRegistry.GetAllAdapters()
		adapterMetrics = make([]AdapterMetricsInfo, len(adapters))
		for i, adapter := range adapters {
			status := adapter.GetStatus()
			
			// Get adapter-specific metrics from door controller stats
			var eventCount, errorCount int64
			var responseTime float64
			if h.doorController != nil {
				stats := h.doorController.GetStats()
				if adapterStats, ok := stats[adapter.Name()]; ok {
					if statsMap, ok := adapterStats.(map[string]interface{}); ok {
						eventCount = getInt64FromStats(statsMap, "eventCount")
						errorCount = getInt64FromStats(statsMap, "errorCount")
						if rt, ok := statsMap["responseTimeMs"].(float64); ok {
							responseTime = rt
						}
					}
				}
			}
			
			adapterMetrics[i] = AdapterMetricsInfo{
				Name:           status.Name,
				Status:         status.Status,
				EventCount:     eventCount,
				ErrorCount:     errorCount,
				LastEventTime:  status.LastEvent,
				ResponseTimeMs: responseTime,
			}
		}
	}
	
	// Get system metrics
	var systemMetrics SystemMetricsInfo
	if h.tierDetector != nil {
		resources := h.tierDetector.GetCurrentResources()
		systemMetrics = SystemMetricsInfo{
			CPUUsage:    resources.CPUUsage,
			MemoryUsage: resources.MemoryUsage,
			DiskUsage:   resources.DiskUsage,
			LastUpdated: resources.LastUpdated,
			// Network metrics would require additional monitoring
			NetworkRxBytes: 0,
			NetworkTxBytes: 0,
		}
	}
	
	// Calculate performance stats
	var performanceStats PerformanceStatsInfo
	if h.doorController != nil {
		stats := h.doorController.GetStats()
		
		// Calculate requests per second (simplified)
		if uptime := time.Since(h.startTime).Seconds(); uptime > 0 {
			totalRequests := getInt64FromStats(stats, "totalRequests")
			performanceStats.RequestsPerSecond = float64(totalRequests) / uptime
		}
		
		// Get average response time
		if avgTime, ok := stats["averageResponseTime"].(float64); ok {
			performanceStats.AverageResponseTime = avgTime
		}
		
		// Calculate error rate
		totalRequests := getInt64FromStats(stats, "totalRequests")
		totalErrors := getInt64FromStats(stats, "totalErrors")
		if totalRequests > 0 {
			performanceStats.ErrorRate = float64(totalErrors) / float64(totalRequests) * 100
		}
		
		// Calculate throughput (events per second)
		if queueMetrics.SentEvents > 0 && time.Since(h.startTime).Seconds() > 0 {
			performanceStats.ThroughputEventsPerS = float64(queueMetrics.SentEvents) / time.Since(h.startTime).Seconds()
		}
	}
	
	// Build response
	response := DeviceMetricsResponse{
		Timestamp:        time.Now().UTC(),
		Uptime:           time.Since(h.startTime),
		QueueMetrics:     queueMetrics,
		AdapterMetrics:   adapterMetrics,
		SystemMetrics:    systemMetrics,
		PerformanceStats: performanceStats,
	}
	
	h.logger.WithFields(logrus.Fields{
		"queueDepth":        queueMetrics.QueueDepth,
		"adapterCount":      len(adapterMetrics),
		"requestsPerSecond": performanceStats.RequestsPerSecond,
		"errorRate":         performanceStats.ErrorRate,
	}).Debug("Device metrics response prepared")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// GetConfig handles GET /api/v1/config
func (h *Handlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Configuration requested")
	
	// Get current configuration
	var currentConfig *config.Config
	if h.configManager != nil {
		currentConfig = h.configManager.GetCurrentConfig()
	} else {
		currentConfig = h.config
	}
	
	// Build response with sensitive fields omitted/masked
	response := ConfigResponse{
		DeviceID:          currentConfig.DeviceID,
		ServerURL:         currentConfig.ServerURL,
		Tier:              currentConfig.Tier,
		QueueMaxSize:      currentConfig.QueueMaxSize,
		HeartbeatInterval: currentConfig.HeartbeatInterval,
		UnlockDuration:    currentConfig.UnlockDuration,
		DatabasePath:      currentConfig.DatabasePath,
		LogLevel:          currentConfig.LogLevel,
		LogFile:           currentConfig.LogFile,
		EnabledAdapters:   currentConfig.EnabledAdapters,
		AdapterConfigs:    currentConfig.AdapterConfigs,
		UpdatesEnabled:    currentConfig.UpdatesEnabled,
		APIServer: APIServerConfigResponse{
			Enabled:      currentConfig.APIServer.Enabled,
			Port:         currentConfig.APIServer.Port,
			Host:         currentConfig.APIServer.Host,
			TLSEnabled:   currentConfig.APIServer.TLSEnabled,
			ReadTimeout:  currentConfig.APIServer.ReadTimeout,
			WriteTimeout: currentConfig.APIServer.WriteTimeout,
			IdleTimeout:  currentConfig.APIServer.IdleTimeout,
			Auth: AuthConfigResponse{
				Enabled:     currentConfig.APIServer.Auth.Enabled,
				TokenExpiry: currentConfig.APIServer.Auth.TokenExpiry,
				AllowedIPs:  currentConfig.APIServer.Auth.AllowedIPs,
				HasHMACKey:  currentConfig.APIServer.Auth.HMACSecret != "",
				HasJWTKey:   currentConfig.APIServer.Auth.JWTSecret != "",
				APIKeyCount: len(currentConfig.APIServer.Auth.APIKeys),
			},
			RateLimit: RateLimitConfigResponse{
				Enabled:         currentConfig.APIServer.RateLimit.Enabled,
				RequestsPerMin:  currentConfig.APIServer.RateLimit.RequestsPerMin,
				BurstSize:       currentConfig.APIServer.RateLimit.BurstSize,
				WindowSize:      currentConfig.APIServer.RateLimit.WindowSize,
				CleanupInterval: currentConfig.APIServer.RateLimit.CleanupInterval,
			},
			CORS: CORSConfigResponse{
				Enabled:          currentConfig.APIServer.CORS.Enabled,
				AllowedOrigins:   currentConfig.APIServer.CORS.AllowedOrigins,
				AllowedMethods:   currentConfig.APIServer.CORS.AllowedMethods,
				AllowedHeaders:   currentConfig.APIServer.CORS.AllowedHeaders,
				ExposedHeaders:   currentConfig.APIServer.CORS.ExposedHeaders,
				AllowCredentials: currentConfig.APIServer.CORS.AllowCredentials,
				MaxAge:           currentConfig.APIServer.CORS.MaxAge,
			},
			Security: SecurityConfigResponse{
				HSTSEnabled:           currentConfig.APIServer.Security.HSTSEnabled,
				HSTSMaxAge:            currentConfig.APIServer.Security.HSTSMaxAge,
				HSTSIncludeSubdomains: currentConfig.APIServer.Security.HSTSIncludeSubdomains,
				CSPEnabled:            currentConfig.APIServer.Security.CSPEnabled,
				CSPDirective:          currentConfig.APIServer.Security.CSPDirective,
				FrameOptions:          currentConfig.APIServer.Security.FrameOptions,
				ContentTypeOptions:    currentConfig.APIServer.Security.ContentTypeOptions,
				XSSProtection:         currentConfig.APIServer.Security.XSSProtection,
				ReferrerPolicy:        currentConfig.APIServer.Security.ReferrerPolicy,
			},
		},
		Timestamp: time.Now().UTC(),
	}
	
	h.logger.WithFields(logrus.Fields{
		"deviceId":        response.DeviceID,
		"tier":            response.Tier,
		"enabledAdapters": len(response.EnabledAdapters),
		"apiServerPort":   response.APIServer.Port,
	}).Debug("Configuration response prepared")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// UpdateConfig handles PUT /api/v1/config
func (h *Handlers) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	requestID := h.generateRequestID()
	
	h.logger.WithField("requestId", requestID).Info("Configuration update requested")
	
	// Parse request body
	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to decode config update request")
		h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
		return
	}
	
	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Invalid config update request")
		h.writeErrorResponseLegacy(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR", requestID)
		return
	}
	
	// Check if config manager is available
	if h.configManager == nil {
		h.logger.WithField("requestId", requestID).Error("Configuration manager not available")
		h.writeErrorResponseLegacy(w, "Configuration management not available", http.StatusServiceUnavailable, "CONFIG_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Log the update request details
	h.logger.WithFields(logrus.Fields{
		"requestId": requestID,
		"clientIP":  getClientIP(r),
		"hasAPIServerUpdates": req.APIServer != nil,
		"hasAdapterUpdates":   len(req.EnabledAdapters) > 0 || len(req.AdapterConfigs) > 0,
	}).Info("Processing configuration update")
	
	// Perform configuration update
	updateResponse, err := h.configManager.UpdateConfig(&req)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to update configuration")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to update configuration: %v", err), http.StatusInternalServerError, "CONFIG_UPDATE_FAILED", requestID)
		return
	}
	
	// Set request ID in response
	updateResponse.RequestID = requestID
	
	h.logger.WithFields(logrus.Fields{
		"requestId":       requestID,
		"updatedFields":   updateResponse.UpdatedFields,
		"requiresRestart": updateResponse.RequiresRestart,
	}).Info("Configuration updated successfully")
	
	// Return success response
	h.writeJSONResponse(w, updateResponse, http.StatusOK)
}

// ReloadConfig handles POST /api/v1/config/reload
func (h *Handlers) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	requestID := h.generateRequestID()
	
	h.logger.WithField("requestId", requestID).Info("Configuration reload requested")
	
	// Parse request body (optional)
	var req ConfigReloadRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to decode config reload request")
			h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
			return
		}
	}
	
	// Check if config manager is available
	if h.configManager == nil {
		h.logger.WithField("requestId", requestID).Error("Configuration manager not available")
		h.writeErrorResponseLegacy(w, "Configuration management not available", http.StatusServiceUnavailable, "CONFIG_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Log the reload request details
	h.logger.WithFields(logrus.Fields{
		"requestId": requestID,
		"force":     req.Force,
		"reason":    req.Reason,
		"clientIP":  getClientIP(r),
	}).Info("Processing configuration reload")
	
	// Perform configuration reload
	reloadResponse, err := h.configManager.ReloadConfig(req.Force)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to reload configuration")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to reload configuration: %v", err), http.StatusInternalServerError, "CONFIG_RELOAD_FAILED", requestID)
		return
	}
	
	// Set request ID in response
	reloadResponse.RequestID = requestID
	
	h.logger.WithFields(logrus.Fields{
		"requestId":     requestID,
		"reloadedFrom":  reloadResponse.ReloadedFrom,
		"changedFields": reloadResponse.ChangedFields,
	}).Info("Configuration reloaded successfully")
	
	// Return success response
	h.writeJSONResponse(w, reloadResponse, http.StatusOK)
}

// GetEvents handles GET /api/v1/events
func (h *Handlers) GetEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := h.generateRequestID()
	
	h.logger.WithField("requestId", requestID).Debug("Events query requested")
	
	// Check if queue manager is available
	if h.queueManager == nil {
		h.logger.WithField("requestId", requestID).Error("Queue manager not available")
		h.writeErrorResponseLegacy(w, "Event querying not available", http.StatusServiceUnavailable, "QUEUE_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Parse query parameters
	var req EventQueryRequest
	
	// Parse time parameters
	if startTimeStr := r.URL.Query().Get("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err != nil {
			h.writeErrorResponseLegacy(w, "Invalid startTime format, use RFC3339", http.StatusBadRequest, "INVALID_TIME_FORMAT", requestID)
			return
		} else {
			req.StartTime = &startTime
		}
	}
	
	if endTimeStr := r.URL.Query().Get("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err != nil {
			h.writeErrorResponseLegacy(w, "Invalid endTime format, use RFC3339", http.StatusBadRequest, "INVALID_TIME_FORMAT", requestID)
			return
		} else {
			req.EndTime = &endTime
		}
	}
	
	// Parse other parameters
	req.EventType = r.URL.Query().Get("eventType")
	req.UserID = r.URL.Query().Get("userId")
	req.SortBy = r.URL.Query().Get("sortBy")
	req.SortOrder = r.URL.Query().Get("sortOrder")
	
	// Parse boolean parameter
	if isSimulatedStr := r.URL.Query().Get("isSimulated"); isSimulatedStr != "" {
		if isSimulated, err := strconv.ParseBool(isSimulatedStr); err != nil {
			h.writeErrorResponseLegacy(w, "Invalid isSimulated value, use true or false", http.StatusBadRequest, "INVALID_BOOLEAN", requestID)
			return
		} else {
			req.IsSimulated = &isSimulated
		}
	}
	
	// Parse integer parameters
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err != nil {
			h.writeErrorResponseLegacy(w, "Invalid limit value", http.StatusBadRequest, "INVALID_INTEGER", requestID)
			return
		} else {
			req.Limit = limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err != nil {
			h.writeErrorResponseLegacy(w, "Invalid offset value", http.StatusBadRequest, "INVALID_INTEGER", requestID)
			return
		} else {
			req.Offset = offset
		}
	}
	
	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Invalid events query request")
		h.writeErrorResponseLegacy(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR", requestID)
		return
	}
	
	// Convert to queue filter
	filter := queue.EventQueryFilter{
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		EventType:   req.EventType,
		UserID:      req.UserID,
		IsSimulated: req.IsSimulated,
		SentStatus:  "all", // Default to all events
		Limit:       req.Limit,
		Offset:      req.Offset,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
	}
	
	// Map sort field names
	switch filter.SortBy {
	case "userId":
		filter.SortBy = "external_user_id"
	case "eventType":
		filter.SortBy = "event_type"
	case "timestamp":
		filter.SortBy = "timestamp"
	}
	
	// Log the query details
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"startTime":   req.StartTime,
		"endTime":     req.EndTime,
		"eventType":   req.EventType,
		"userId":      req.UserID,
		"isSimulated": req.IsSimulated,
		"limit":       req.Limit,
		"offset":      req.Offset,
		"sortBy":      req.SortBy,
		"sortOrder":   req.SortOrder,
		"clientIP":    getClientIP(r),
	}).Info("Processing events query")
	
	// Query events
	events, total, err := h.queueManager.QueryEvents(ctx, filter)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to query events")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to query events: %v", err), http.StatusInternalServerError, "QUERY_FAILED", requestID)
		return
	}
	
	// Convert to API response format
	eventResponses := make([]EventResponse, len(events))
	for i, event := range events {
		eventResponses[i] = EventResponse{
			ID:             event.ID,
			EventID:        event.Event.EventID,
			ExternalUserID: event.Event.ExternalUserID,
			InternalUserID: event.Event.InternalUserID,
			Timestamp:      event.Event.Timestamp,
			EventType:      event.Event.EventType,
			IsSimulated:    event.Event.IsSimulated,
			DeviceID:       event.Event.DeviceID,
			RawData:        event.Event.RawData,
			CreatedAt:      event.CreatedAt,
			SentAt:         event.SentAt,
			RetryCount:     event.RetryCount,
		}
	}
	
	// Build response
	response := EventsResponse{
		Events:    eventResponses,
		Total:     total,
		Limit:     req.Limit,
		Offset:    req.Offset,
		HasMore:   int64(req.Offset+len(eventResponses)) < total,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":    requestID,
		"eventsCount":  len(eventResponses),
		"total":        total,
		"hasMore":      response.HasMore,
	}).Info("Events query completed successfully")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// GetEventStats handles GET /api/v1/events/stats
func (h *Handlers) GetEventStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := h.generateRequestID()
	
	h.logger.WithField("requestId", requestID).Debug("Event statistics requested")
	
	// Check if queue manager is available
	if h.queueManager == nil {
		h.logger.WithField("requestId", requestID).Error("Queue manager not available")
		h.writeErrorResponseLegacy(w, "Event statistics not available", http.StatusServiceUnavailable, "QUEUE_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Log the stats request
	h.logger.WithFields(logrus.Fields{
		"requestId": requestID,
		"clientIP":  getClientIP(r),
	}).Info("Processing event statistics request")
	
	// Get event statistics
	stats, err := h.queueManager.GetEventStats(ctx)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to get event statistics")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to get event statistics: %v", err), http.StatusInternalServerError, "STATS_FAILED", requestID)
		return
	}
	
	// Build response
	response := EventStatsResponse{
		TotalEvents:      stats.TotalEvents,
		EventsByType:     stats.EventsByType,
		EventsByHour:     stats.EventsByHour,
		EventsByDay:      stats.EventsByDay,
		PendingEvents:    stats.PendingEvents,
		SentEvents:       stats.SentEvents,
		FailedEvents:     stats.FailedEvents,
		UniqueUsers:      stats.UniqueUsers,
		SimulatedEvents:  stats.SimulatedEvents,
		OldestEventTime:  stats.OldestEventTime,
		NewestEventTime:  stats.NewestEventTime,
		AveragePerHour:   stats.AveragePerHour,
		AveragePerDay:    stats.AveragePerDay,
		Timestamp:        time.Now().UTC(),
		RequestID:        requestID,
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":      requestID,
		"totalEvents":    response.TotalEvents,
		"pendingEvents":  response.PendingEvents,
		"sentEvents":     response.SentEvents,
		"failedEvents":   response.FailedEvents,
		"uniqueUsers":    response.UniqueUsers,
	}).Info("Event statistics completed successfully")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// ClearEvents handles DELETE /api/v1/events
func (h *Handlers) ClearEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := h.generateRequestID()
	
	h.logger.WithField("requestId", requestID).Info("Event clearing requested")
	
	// Check if queue manager is available
	if h.queueManager == nil {
		h.logger.WithField("requestId", requestID).Error("Queue manager not available")
		h.writeErrorResponseLegacy(w, "Event clearing not available", http.StatusServiceUnavailable, "QUEUE_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Parse request body
	var req EventClearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to decode event clear request")
		h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
		return
	}
	
	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Invalid event clear request")
		h.writeErrorResponseLegacy(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR", requestID)
		return
	}
	
	// Convert to queue criteria
	criteria := queue.EventClearCriteria{
		OlderThan:  req.OlderThan,
		EventType:  req.EventType,
		OnlySent:   req.OnlySent,
		OnlyFailed: req.OnlyFailed,
	}
	
	// Build criteria description for logging and response
	var criteriaDesc []string
	if req.OlderThan != nil {
		criteriaDesc = append(criteriaDesc, fmt.Sprintf("older than %s", req.OlderThan.Format(time.RFC3339)))
	}
	if req.EventType != "" {
		criteriaDesc = append(criteriaDesc, fmt.Sprintf("event type: %s", req.EventType))
	}
	if req.OnlySent {
		criteriaDesc = append(criteriaDesc, "only sent events")
	}
	if req.OnlyFailed {
		criteriaDesc = append(criteriaDesc, "only failed events")
	}
	
	criteriaString := "all events"
	if len(criteriaDesc) > 0 {
		criteriaString = strings.Join(criteriaDesc, ", ")
	}
	
	// Log the clear request details
	h.logger.WithFields(logrus.Fields{
		"requestId":  requestID,
		"olderThan":  req.OlderThan,
		"eventType":  req.EventType,
		"onlySent":   req.OnlySent,
		"onlyFailed": req.OnlyFailed,
		"reason":     req.Reason,
		"criteria":   criteriaString,
		"clientIP":   getClientIP(r),
	}).Warn("Processing event clearing request")
	
	// Clear events
	deletedCount, err := h.queueManager.ClearEvents(ctx, criteria)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to clear events")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to clear events: %v", err), http.StatusInternalServerError, "CLEAR_FAILED", requestID)
		return
	}
	
	// Build response
	response := EventClearResponse{
		Success:      true,
		Message:      fmt.Sprintf("Successfully deleted %d events", deletedCount),
		DeletedCount: deletedCount,
		Criteria:     criteriaString,
		Timestamp:    time.Now().UTC(),
		RequestID:    requestID,
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":    requestID,
		"deletedCount": deletedCount,
		"criteria":     criteriaString,
		"reason":       req.Reason,
	}).Warn("Event clearing completed successfully")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// GetAdapters handles GET /api/v1/adapters
func (h *Handlers) GetAdapters(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Adapters list requested")
	
	// Get all adapters from registry
	var allAdapters []AdapterInfo
	if h.adapterRegistry != nil {
		adapters := h.adapterRegistry.GetAllAdapters()
		allAdapters = make([]AdapterInfo, len(adapters))
		
		for i, adapter := range adapters {
			status := adapter.GetStatus()
			allAdapters[i] = AdapterInfo{
				Name:         adapter.Name(),
				Status:       status.Status,
				IsHealthy:    adapter.IsHealthy(),
				LastEvent:    status.LastEvent,
				ErrorMessage: status.ErrorMessage,
				UpdatedAt:    status.UpdatedAt,
			}
		}
	}
	
	// Get enabled adapters from configuration
	var enabledAdapters []string
	if h.configManager != nil {
		config := h.configManager.GetCurrentConfig()
		enabledAdapters = config.EnabledAdapters
	}
	
	// Get active adapters
	var activeAdapters []string
	if h.adapterRegistry != nil {
		active := h.adapterRegistry.GetActiveAdapters()
		activeAdapters = make([]string, len(active))
		for i, adapter := range active {
			activeAdapters[i] = adapter.Name()
		}
	}
	
	// Build response
	response := AdaptersResponse{
		Adapters:        allAdapters,
		EnabledAdapters: enabledAdapters,
		ActiveAdapters:  activeAdapters,
		TotalCount:      len(allAdapters),
		ActiveCount:     len(activeAdapters),
		Timestamp:       time.Now().UTC(),
	}
	
	h.logger.WithFields(logrus.Fields{
		"totalAdapters":   len(allAdapters),
		"enabledAdapters": len(enabledAdapters),
		"activeAdapters":  len(activeAdapters),
	}).Debug("Adapters list response prepared")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// GetAdapter handles GET /api/v1/adapters/{name}
func (h *Handlers) GetAdapter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	requestID := h.generateRequestID()
	
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"adapterName": name,
	}).Debug("Specific adapter status requested")
	
	// Check if adapter registry is available
	if h.adapterRegistry == nil {
		h.logger.WithField("requestId", requestID).Error("Adapter registry not available")
		h.writeErrorResponseLegacy(w, "Adapter management not available", http.StatusServiceUnavailable, "ADAPTER_REGISTRY_UNAVAILABLE", requestID)
		return
	}
	
	// Get adapter from registry
	adapter, err := h.adapterRegistry.GetAdapter(name)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"requestId":   requestID,
			"adapterName": name,
		}).Error("Adapter not found")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Adapter '%s' not found", name), http.StatusNotFound, "ADAPTER_NOT_FOUND", requestID)
		return
	}
	
	// Get adapter status
	status := adapter.GetStatus()
	
	// Check if adapter is enabled in configuration
	var isEnabled bool
	var adapterConfig map[string]interface{}
	if h.configManager != nil {
		config := h.configManager.GetCurrentConfig()
		
		// Check if adapter is in enabled list
		for _, enabledName := range config.EnabledAdapters {
			if enabledName == name {
				isEnabled = true
				break
			}
		}
		
		// Get adapter-specific configuration
		if config.AdapterConfigs != nil {
			if cfg, exists := config.AdapterConfigs[name]; exists {
				adapterConfig = cfg
			}
		}
	}
	
	// Check if adapter is currently active
	isActive := false
	if h.adapterRegistry != nil {
		activeAdapters := h.adapterRegistry.GetActiveAdapters()
		for _, activeAdapter := range activeAdapters {
			if activeAdapter.Name() == name {
				isActive = true
				break
			}
		}
	}
	
	// Build response
	response := AdapterDetailResponse{
		Name:         name,
		Status:       status.Status,
		IsEnabled:    isEnabled,
		IsActive:     isActive,
		IsHealthy:    adapter.IsHealthy(),
		LastEvent:    status.LastEvent,
		ErrorMessage: status.ErrorMessage,
		UpdatedAt:    status.UpdatedAt,
		Config:       adapterConfig,
		Timestamp:    time.Now().UTC(),
		RequestID:    requestID,
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"adapterName": name,
		"status":      status.Status,
		"isEnabled":   isEnabled,
		"isActive":    isActive,
		"isHealthy":   adapter.IsHealthy(),
	}).Debug("Adapter status response prepared")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// EnableAdapter handles POST /api/v1/adapters/{name}/enable
func (h *Handlers) EnableAdapter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	requestID := h.generateRequestID()
	
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"adapterName": name,
		"clientIP":    getClientIP(r),
	}).Info("Adapter enable requested")
	
	// Parse optional request body
	var req AdapterEnableRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to decode enable adapter request")
			h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
			return
		}
	}
	
	// Check if config manager is available
	if h.configManager == nil {
		h.logger.WithField("requestId", requestID).Error("Configuration manager not available")
		h.writeErrorResponseLegacy(w, "Configuration management not available", http.StatusServiceUnavailable, "CONFIG_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Check if adapter registry is available to verify adapter exists
	if h.adapterRegistry != nil {
		if _, err := h.adapterRegistry.GetAdapter(name); err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"requestId":   requestID,
				"adapterName": name,
			}).Error("Adapter not found")
			h.writeErrorResponseLegacy(w, fmt.Sprintf("Adapter '%s' not found", name), http.StatusNotFound, "ADAPTER_NOT_FOUND", requestID)
			return
		}
	}
	
	// Get current configuration
	currentConfig := h.configManager.GetCurrentConfig()
	
	// Check if adapter is already enabled
	isAlreadyEnabled := false
	for _, enabledName := range currentConfig.EnabledAdapters {
		if enabledName == name {
			isAlreadyEnabled = true
			break
		}
	}
	
	if isAlreadyEnabled {
		h.logger.WithFields(logrus.Fields{
			"requestId":   requestID,
			"adapterName": name,
		}).Info("Adapter is already enabled")
		
		response := AdapterEnableResponse{
			Success:   true,
			Message:   fmt.Sprintf("Adapter '%s' is already enabled", name),
			Adapter:   name,
			Enabled:   true,
			Timestamp: time.Now().UTC(),
			RequestID: requestID,
		}
		
		h.writeJSONResponse(w, response, http.StatusOK)
		return
	}
	
	// Add adapter to enabled list
	newEnabledAdapters := make([]string, len(currentConfig.EnabledAdapters)+1)
	copy(newEnabledAdapters, currentConfig.EnabledAdapters)
	newEnabledAdapters[len(currentConfig.EnabledAdapters)] = name
	
	// Update configuration
	updateRequest := &ConfigUpdateRequest{
		EnabledAdapters: newEnabledAdapters,
	}
	
	updateResponse, err := h.configManager.UpdateConfig(updateRequest)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to enable adapter")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to enable adapter: %v", err), http.StatusInternalServerError, "ADAPTER_ENABLE_FAILED", requestID)
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":       requestID,
		"adapterName":     name,
		"requiresRestart": updateResponse.RequiresRestart,
		"reason":          req.Reason,
	}).Info("Adapter enabled successfully")
	
	// Build response
	response := AdapterEnableResponse{
		Success:         true,
		Message:         fmt.Sprintf("Adapter '%s' enabled successfully", name),
		Adapter:         name,
		Enabled:         true,
		RequiresRestart: updateResponse.RequiresRestart,
		Timestamp:       time.Now().UTC(),
		RequestID:       requestID,
	}
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// DisableAdapter handles POST /api/v1/adapters/{name}/disable
func (h *Handlers) DisableAdapter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	requestID := h.generateRequestID()
	
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"adapterName": name,
		"clientIP":    getClientIP(r),
	}).Info("Adapter disable requested")
	
	// Parse optional request body
	var req AdapterDisableRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to decode disable adapter request")
			h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
			return
		}
	}
	
	// Check if config manager is available
	if h.configManager == nil {
		h.logger.WithField("requestId", requestID).Error("Configuration manager not available")
		h.writeErrorResponseLegacy(w, "Configuration management not available", http.StatusServiceUnavailable, "CONFIG_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Get current configuration
	currentConfig := h.configManager.GetCurrentConfig()
	
	// Check if adapter is currently enabled and find its position
	adapterIndex := -1
	for i, enabledName := range currentConfig.EnabledAdapters {
		if enabledName == name {
			adapterIndex = i
			break
		}
	}
	
	if adapterIndex == -1 {
		h.logger.WithFields(logrus.Fields{
			"requestId":   requestID,
			"adapterName": name,
		}).Info("Adapter is already disabled")
		
		response := AdapterDisableResponse{
			Success:   true,
			Message:   fmt.Sprintf("Adapter '%s' is already disabled", name),
			Adapter:   name,
			Enabled:   false,
			Timestamp: time.Now().UTC(),
			RequestID: requestID,
		}
		
		h.writeJSONResponse(w, response, http.StatusOK)
		return
	}
	
	// Remove adapter from enabled list
	newEnabledAdapters := make([]string, len(currentConfig.EnabledAdapters)-1)
	copy(newEnabledAdapters[:adapterIndex], currentConfig.EnabledAdapters[:adapterIndex])
	copy(newEnabledAdapters[adapterIndex:], currentConfig.EnabledAdapters[adapterIndex+1:])
	
	// Update configuration
	updateRequest := &ConfigUpdateRequest{
		EnabledAdapters: newEnabledAdapters,
	}
	
	updateResponse, err := h.configManager.UpdateConfig(updateRequest)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to disable adapter")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to disable adapter: %v", err), http.StatusInternalServerError, "ADAPTER_DISABLE_FAILED", requestID)
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":       requestID,
		"adapterName":     name,
		"requiresRestart": updateResponse.RequiresRestart,
		"reason":          req.Reason,
	}).Info("Adapter disabled successfully")
	
	// Build response
	response := AdapterDisableResponse{
		Success:         true,
		Message:         fmt.Sprintf("Adapter '%s' disabled successfully", name),
		Adapter:         name,
		Enabled:         false,
		RequiresRestart: updateResponse.RequiresRestart,
		Timestamp:       time.Now().UTC(),
		RequestID:       requestID,
	}
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// UpdateAdapterConfig handles PUT /api/v1/adapters/{name}/config
func (h *Handlers) UpdateAdapterConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	requestID := h.generateRequestID()
	
	h.logger.WithFields(logrus.Fields{
		"requestId":   requestID,
		"adapterName": name,
		"clientIP":    getClientIP(r),
	}).Info("Adapter configuration update requested")
	
	// Parse request body
	var req AdapterConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to decode adapter config update request")
		h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
		return
	}
	
	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Invalid adapter config update request")
		h.writeErrorResponseLegacy(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR", requestID)
		return
	}
	
	// Check if config manager is available
	if h.configManager == nil {
		h.logger.WithField("requestId", requestID).Error("Configuration manager not available")
		h.writeErrorResponseLegacy(w, "Configuration management not available", http.StatusServiceUnavailable, "CONFIG_MANAGER_UNAVAILABLE", requestID)
		return
	}
	
	// Check if adapter registry is available to verify adapter exists
	if h.adapterRegistry != nil {
		if _, err := h.adapterRegistry.GetAdapter(name); err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"requestId":   requestID,
				"adapterName": name,
			}).Error("Adapter not found")
			h.writeErrorResponseLegacy(w, fmt.Sprintf("Adapter '%s' not found", name), http.StatusNotFound, "ADAPTER_NOT_FOUND", requestID)
			return
		}
	}
	
	// Get current configuration
	currentConfig := h.configManager.GetCurrentConfig()
	
	// Prepare adapter configs update
	newAdapterConfigs := make(map[string]map[string]interface{})
	
	// Copy existing adapter configs
	if currentConfig.AdapterConfigs != nil {
		for adapterName, adapterConfig := range currentConfig.AdapterConfigs {
			newAdapterConfigs[adapterName] = adapterConfig
		}
	}
	
	// Update the specific adapter's configuration
	newAdapterConfigs[name] = req.Config
	
	// Update configuration
	updateRequest := &ConfigUpdateRequest{
		AdapterConfigs: newAdapterConfigs,
	}
	
	updateResponse, err := h.configManager.UpdateConfig(updateRequest)
	if err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to update adapter configuration")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to update adapter configuration: %v", err), http.StatusInternalServerError, "ADAPTER_CONFIG_UPDATE_FAILED", requestID)
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":       requestID,
		"adapterName":     name,
		"requiresRestart": updateResponse.RequiresRestart,
		"reason":          req.Reason,
	}).Info("Adapter configuration updated successfully")
	
	// Build response
	response := AdapterConfigUpdateResponse{
		Success:         true,
		Message:         fmt.Sprintf("Configuration for adapter '%s' updated successfully", name),
		Adapter:         name,
		Config:          req.Config,
		RequiresRestart: updateResponse.RequiresRestart,
		Timestamp:       time.Now().UTC(),
		RequestID:       requestID,
	}
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// WebSocketHandler handles GET /api/v1/ws
func (h *Handlers) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	requestID := h.generateRequestID()
	
	h.logger.WithFields(logrus.Fields{
		"requestId":  requestID,
		"remoteAddr": r.RemoteAddr,
		"userAgent":  r.UserAgent(),
	}).Info("WebSocket connection requested")
	
	// Check if WebSocket manager is available
	if h.wsManager == nil {
		h.logger.WithField("requestId", requestID).Error("WebSocket manager not available")
		h.writeErrorResponseLegacy(w, "WebSocket functionality not available", http.StatusServiceUnavailable, "WEBSOCKET_UNAVAILABLE", requestID)
		return
	}
	
	// Extract authentication information from context
	// This should have been set by the authentication middleware
	var authInfo *AuthenticationInfo
	if authCtx := r.Context().Value("auth"); authCtx != nil {
		if auth, ok := authCtx.(map[string]interface{}); ok {
			authInfo = &AuthenticationInfo{
				Method: getStringFromMap(auth, "method"),
				UserID: getStringFromMap(auth, "userId"),
			}
			
			// Parse expiration if present
			if expiresAtStr := getStringFromMap(auth, "expiresAt"); expiresAtStr != "" {
				if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
					authInfo.ExpiresAt = &expiresAt
				}
			}
		}
	}
	
	// Handle the WebSocket connection
	if err := h.wsManager.HandleWebSocketConnection(w, r, authInfo); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to handle WebSocket connection")
		h.writeErrorResponseLegacy(w, fmt.Sprintf("Failed to establish WebSocket connection: %v", err), http.StatusInternalServerError, "WEBSOCKET_CONNECTION_FAILED", requestID)
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":  requestID,
		"remoteAddr": r.RemoteAddr,
	}).Info("WebSocket connection established successfully")
}

// writeJSONResponse writes a JSON response
func (h *Handlers) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// writeErrorResponse writes a standardized JSON error response using the new error handling system
func (h *Handlers) writeErrorResponse(w http.ResponseWriter, r *http.Request, code ErrorCode, message string, requestID string) {
	// Create error response
	errorResponse := NewErrorResponse(code, message, r, requestID)

	// Log the error response
	h.logger.WithFields(logrus.Fields{
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
		h.logger.WithError(err).Error("Failed to encode error response")
		// Fallback to plain text
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// writeErrorResponseLegacy provides backward compatibility for old error response calls
func (h *Handlers) writeErrorResponseLegacy(w http.ResponseWriter, message string, statusCode int, code string, requestID string) {
	errorResponse := ErrorResponse{
		Error:     "true",
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
		Status:    statusCode,
	}
	
	h.writeJSONResponse(w, errorResponse, statusCode)
}

// generateRequestID generates a unique request ID for tracking
func (h *Handlers) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// getInt64FromStats safely extracts int64 value from stats map
func getInt64FromStats(stats map[string]interface{}, key string) int64 {
	if val, ok := stats[key]; ok {
		if intVal, ok := val.(int64); ok {
			return intVal
		}
		if intVal, ok := val.(int); ok {
			return int64(intVal)
		}
	}
	return 0
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	
	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	
	return r.RemoteAddr
}

// getStringFromMap safely extracts string values from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getTimeFromMap safely extracts time.Time values from map
func getTimeFromMap(m map[string]interface{}, key string) time.Time {
	if val, ok := m[key]; ok {
		if t, ok := val.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

// BroadcastEvent broadcasts an event to all WebSocket connections
func (h *Handlers) BroadcastEvent(eventType string, data interface{}) {
	if h.wsManager != nil {
		h.wsManager.BroadcastEvent(eventType, data)
	}
}

// GetWebSocketConnectionCount returns the number of active WebSocket connections
func (h *Handlers) GetWebSocketConnectionCount() int {
	if h.wsManager != nil {
		return h.wsManager.GetConnectionCount()
	}
	return 0
}

// GetWebSocketConnectionInfo returns information about WebSocket connections
func (h *Handlers) GetWebSocketConnectionInfo() []map[string]interface{} {
	if h.wsManager != nil {
		return h.wsManager.GetConnectionInfo()
	}
	return []map[string]interface{}{}
}

// WebSocketStatus handles GET /api/v1/ws/status
func (h *Handlers) WebSocketStatus(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("WebSocket status requested")
	
	var response WebSocketStatusResponse
	
	if h.wsManager != nil {
		// Get connection info
		connInfo := h.wsManager.GetConnectionInfo()
		connections := make([]WebSocketConnectionInfo, len(connInfo))
		
		for i, info := range connInfo {
			conn := WebSocketConnectionInfo{
				ID:           getStringFromMap(info, "id"),
				RemoteAddr:   getStringFromMap(info, "remoteAddr"),
				UserAgent:    getStringFromMap(info, "userAgent"),
				LastPing:     getTimeFromMap(info, "lastPing"),
				MessagesSent: 0, // TODO: Track message counts
			}
			
			// Parse filters
			if filtersData, ok := info["filters"]; ok {
				if filters, ok := filtersData.(WebSocketFilters); ok {
					conn.Filters = WebSocketFiltersInfo{
						EventTypes:    filters.EventTypes,
						DeviceID:      filters.DeviceID,
						UserID:        filters.UserID,
						MinSeverity:   filters.MinSeverity,
						IncludeSystem: filters.IncludeSystem,
					}
				}
			}
			
			// Parse auth info
			if authData, ok := info["auth"].(map[string]interface{}); ok {
				conn.Auth = &WebSocketAuthInfo{
					UserID: getStringFromMap(authData, "userId"),
					Method: getStringFromMap(authData, "method"),
				}
				
				if expiresAtStr := getStringFromMap(authData, "expiresAt"); expiresAtStr != "" {
					if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
						conn.Auth.ExpiresAt = &expiresAt
					}
				}
			}
			
			connections[i] = conn
		}
		
		response = WebSocketStatusResponse{
			Enabled:         true,
			ConnectionCount: h.wsManager.GetConnectionCount(),
			MaxConnections:  100, // TODO: Get from config
			Connections:     connections,
			Timestamp:       time.Now().UTC(),
		}
	} else {
		response = WebSocketStatusResponse{
			Enabled:         false,
			ConnectionCount: 0,
			MaxConnections:  0,
			Connections:     []WebSocketConnectionInfo{},
			Timestamp:       time.Now().UTC(),
		}
	}
	
	h.logger.WithFields(logrus.Fields{
		"enabled":         response.Enabled,
		"connectionCount": response.ConnectionCount,
		"maxConnections":  response.MaxConnections,
	}).Debug("WebSocket status response prepared")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}

// WebSocketBroadcast handles POST /api/v1/ws/broadcast
func (h *Handlers) WebSocketBroadcast(w http.ResponseWriter, r *http.Request) {
	requestID := h.generateRequestID()
	
	h.logger.WithField("requestId", requestID).Info("WebSocket broadcast requested")
	
	// Check if WebSocket manager is available
	if h.wsManager == nil {
		h.logger.WithField("requestId", requestID).Error("WebSocket manager not available")
		h.writeErrorResponseLegacy(w, "WebSocket functionality not available", http.StatusServiceUnavailable, "WEBSOCKET_UNAVAILABLE", requestID)
		return
	}
	
	// Parse request body
	var req WebSocketEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Failed to decode WebSocket broadcast request")
		h.writeErrorResponseLegacy(w, "Invalid JSON in request body", http.StatusBadRequest, "INVALID_JSON", requestID)
		return
	}
	
	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.WithError(err).WithField("requestId", requestID).Error("Invalid WebSocket broadcast request")
		h.writeErrorResponseLegacy(w, err.Error(), http.StatusBadRequest, "VALIDATION_ERROR", requestID)
		return
	}
	
	// Log the broadcast request
	h.logger.WithFields(logrus.Fields{
		"requestId":  requestID,
		"eventType":  req.EventType,
		"targetIds":  req.TargetIDs,
		"clientIP":   getClientIP(r),
	}).Info("Broadcasting WebSocket event")
	
	// Broadcast the event
	connectionsBefore := h.wsManager.GetConnectionCount()
	h.wsManager.BroadcastEvent(req.EventType, req.Data)
	
	// Generate event ID
	eventID := fmt.Sprintf("evt_%d", time.Now().UnixNano())
	
	// Return success response
	response := WebSocketEventResponse{
		Success:         true,
		Message:         "Event broadcasted successfully",
		EventType:       req.EventType,
		EventID:         eventID,
		ConnectionsSent: connectionsBefore, // Approximate, actual filtering happens in manager
		Timestamp:       time.Now().UTC(),
		RequestID:       requestID,
	}
	
	h.logger.WithFields(logrus.Fields{
		"requestId":       requestID,
		"eventType":       req.EventType,
		"eventId":         eventID,
		"connectionsSent": response.ConnectionsSent,
	}).Info("WebSocket event broadcasted successfully")
	
	h.writeJSONResponse(w, response, http.StatusOK)
}



