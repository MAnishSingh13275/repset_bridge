package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/tier"
	"gym-door-bridge/internal/types"
)

// HealthStatus represents the overall health status of the system
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// String returns the string representation of the health status
func (h HealthStatus) String() string {
	return string(h)
}

// SystemHealth represents the complete health information of the system
type SystemHealth struct {
	Status        HealthStatus           `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	QueueDepth    int                    `json:"queueDepth"`
	AdapterStatus []types.AdapterStatus `json:"adapterStatus"`
	Resources     tier.SystemResources   `json:"resources"`
	Tier          tier.Tier              `json:"tier"`
	LastEventTime *time.Time             `json:"lastEventTime,omitempty"`
	Uptime        time.Duration          `json:"uptime"`
	Version       string                 `json:"version"`
	DeviceID      string                 `json:"deviceId,omitempty"`
}

// HealthCheckConfig holds configuration for the health monitoring system
type HealthCheckConfig struct {
	Port                int           `json:"port"`                // Port for health check endpoint
	Path                string        `json:"path"`                // Path for health check endpoint
	EnableMetrics       bool          `json:"enableMetrics"`       // Enable OpenTelemetry metrics
	MetricsPort         int           `json:"metricsPort"`         // Port for metrics endpoint
	UnhealthyThreshold  time.Duration `json:"unhealthyThreshold"`  // Time before marking as unhealthy
	DegradedThreshold   time.Duration `json:"degradedThreshold"`   // Time before marking as degraded
}

// DefaultHealthCheckConfig returns the default health check configuration
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Port:               8080,
		Path:               "/health",
		EnableMetrics:      false,
		MetricsPort:        9090,
		UnhealthyThreshold: 5 * time.Minute,
		DegradedThreshold:  2 * time.Minute,
	}
}

// HealthMonitor manages the health monitoring system
type HealthMonitor struct {
	mu                sync.RWMutex
	config            HealthCheckConfig
	logger            *logrus.Logger
	startTime         time.Time
	version           string
	deviceID          string
	
	// Dependencies
	queueManager      queue.QueueManager
	tierDetector      TierDetector
	adapterRegistry   AdapterRegistry
	
	// HTTP server for health endpoint
	httpServer        *http.Server
	
	// Metrics (optional)
	metricsExporter   MetricsExporter
	
	// Health state
	lastHealthCheck   time.Time
	currentHealth     SystemHealth
}

// AdapterRegistry interface for getting adapter status
type AdapterRegistry interface {
	GetAllAdapters() []adapters.HardwareAdapter
	GetAdapterStatus(name string) (types.AdapterStatus, error)
}

// MetricsExporter interface for optional OpenTelemetry metrics
type MetricsExporter interface {
	RecordQueueDepth(depth int)
	RecordAdapterStatus(name string, status string)
	RecordSystemResources(resources tier.SystemResources)
	RecordHealthStatus(status HealthStatus)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HealthMonitorOption is a functional option for configuring the HealthMonitor
type HealthMonitorOption func(*HealthMonitor)

// WithLogger sets the logger for the health monitor
func WithLogger(logger *logrus.Logger) HealthMonitorOption {
	return func(h *HealthMonitor) {
		h.logger = logger
	}
}

// WithVersion sets the version for the health monitor
func WithVersion(version string) HealthMonitorOption {
	return func(h *HealthMonitor) {
		h.version = version
	}
}

// WithDeviceID sets the device ID for the health monitor
func WithDeviceID(deviceID string) HealthMonitorOption {
	return func(h *HealthMonitor) {
		h.deviceID = deviceID
	}
}

// WithMetricsExporter sets the metrics exporter
func WithMetricsExporter(exporter MetricsExporter) HealthMonitorOption {
	return func(h *HealthMonitor) {
		h.metricsExporter = exporter
	}
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(
	config HealthCheckConfig,
	queueManager queue.QueueManager,
	tierDetector TierDetector,
	adapterRegistry AdapterRegistry,
	opts ...HealthMonitorOption,
) *HealthMonitor {
	h := &HealthMonitor{
		config:          config,
		logger:          logrus.New(),
		startTime:       time.Now(),
		queueManager:    queueManager,
		tierDetector:    tierDetector,
		adapterRegistry: adapterRegistry,
		version:         "unknown",
	}
	
	// Apply options
	for _, opt := range opts {
		opt(h)
	}
	
	return h
}

// Start begins the health monitoring system
func (h *HealthMonitor) Start(ctx context.Context) error {
	h.logger.Info("Starting health monitor", "port", h.config.Port, "path", h.config.Path)
	
	// Start metrics exporter if enabled
	if h.config.EnableMetrics && h.metricsExporter != nil {
		if err := h.metricsExporter.Start(ctx); err != nil {
			h.logger.WithError(err).Error("Failed to start metrics exporter")
			return fmt.Errorf("failed to start metrics exporter: %w", err)
		}
		h.logger.Info("Metrics exporter started", "port", h.config.MetricsPort)
	}
	
	// Set up HTTP server for health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(h.config.Path, h.handleHealthCheck)
	
	h.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.config.Port),
		Handler: mux,
	}
	
	// Start HTTP server in a goroutine
	go func() {
		if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.WithError(err).Error("Health check HTTP server failed")
		}
	}()
	
	h.logger.Info("Health check endpoint started", "url", fmt.Sprintf("http://localhost:%d%s", h.config.Port, h.config.Path))
	
	// Perform initial health check
	h.updateHealthStatus(ctx)
	
	return nil
}

// Stop gracefully shuts down the health monitoring system
func (h *HealthMonitor) Stop(ctx context.Context) error {
	h.logger.Info("Stopping health monitor")
	
	// Stop HTTP server
	if h.httpServer != nil {
		if err := h.httpServer.Shutdown(ctx); err != nil {
			h.logger.WithError(err).Error("Failed to shutdown health check HTTP server")
		}
	}
	
	// Stop metrics exporter if enabled
	if h.config.EnableMetrics && h.metricsExporter != nil {
		if err := h.metricsExporter.Stop(ctx); err != nil {
			h.logger.WithError(err).Error("Failed to stop metrics exporter")
		}
	}
	
	return nil
}

// GetCurrentHealth returns the current system health
func (h *HealthMonitor) GetCurrentHealth() SystemHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentHealth
}

// UpdateHealth updates the current health status
func (h *HealthMonitor) UpdateHealth(ctx context.Context) error {
	return h.updateHealthStatus(ctx)
}

// handleHealthCheck handles HTTP health check requests
func (h *HealthMonitor) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Update health status
	if err := h.updateHealthStatus(ctx); err != nil {
		h.logger.WithError(err).Error("Failed to update health status")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	health := h.GetCurrentHealth()
	
	// Set appropriate HTTP status code based on health
	var statusCode int
	switch health.Status {
	case HealthStatusHealthy:
		statusCode = http.StatusOK
	case HealthStatusDegraded:
		statusCode = http.StatusOK // Still OK, but degraded
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	default:
		statusCode = http.StatusInternalServerError
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.WithError(err).Error("Failed to encode health response")
	}
}

// updateHealthStatus updates the current health status by collecting information from all components
func (h *HealthMonitor) updateHealthStatus(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	now := time.Now()
	h.lastHealthCheck = now
	
	// Get queue depth
	queueDepth, err := h.queueManager.GetQueueDepth(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get queue depth")
		queueDepth = -1 // Indicate error
	}
	
	// Get adapter statuses
	var adapterStatuses []types.AdapterStatus
	if h.adapterRegistry != nil {
		for _, adapter := range h.adapterRegistry.GetAllAdapters() {
			status := adapter.GetStatus()
			adapterStatuses = append(adapterStatuses, status)
		}
	}
	
	// Get system resources and tier
	var resources tier.SystemResources
	var currentTier tier.Tier
	if h.tierDetector != nil {
		resources = h.tierDetector.GetCurrentResources()
		currentTier = h.tierDetector.GetCurrentTier()
	}
	
	// Determine overall health status
	overallStatus := h.determineOverallHealth(queueDepth, adapterStatuses, resources)
	
	// Get last event time (if available)
	var lastEventTime *time.Time
	if stats, err := h.queueManager.GetStats(ctx); err == nil {
		if !stats.LastSentAt.IsZero() {
			lastEventTime = &stats.LastSentAt
		}
	}
	
	// Update current health
	h.currentHealth = SystemHealth{
		Status:        overallStatus,
		Timestamp:     now,
		QueueDepth:    queueDepth,
		AdapterStatus: adapterStatuses,
		Resources:     resources,
		Tier:          currentTier,
		LastEventTime: lastEventTime,
		Uptime:        now.Sub(h.startTime),
		Version:       h.version,
		DeviceID:      h.deviceID,
	}
	
	// Record metrics if enabled
	if h.config.EnableMetrics && h.metricsExporter != nil {
		h.metricsExporter.RecordQueueDepth(queueDepth)
		h.metricsExporter.RecordSystemResources(resources)
		h.metricsExporter.RecordHealthStatus(overallStatus)
		
		for _, status := range adapterStatuses {
			h.metricsExporter.RecordAdapterStatus(status.Name, status.Status)
		}
	}
	
	return nil
}

// determineOverallHealth determines the overall health status based on component health
func (h *HealthMonitor) determineOverallHealth(queueDepth int, adapterStatuses []types.AdapterStatus, resources tier.SystemResources) HealthStatus {
	// Check for critical issues that make the system unhealthy
	
	// If queue depth is unknown (error), consider unhealthy
	if queueDepth < 0 {
		return HealthStatusUnhealthy
	}
	
	// Check if any critical adapters are in error state
	criticalAdapterErrors := 0
	totalAdapters := len(adapterStatuses)
	
	for _, status := range adapterStatuses {
		if status.Status == types.StatusError {
			criticalAdapterErrors++
		}
	}
	
	// If all adapters are in error, system is unhealthy
	if totalAdapters > 0 && criticalAdapterErrors == totalAdapters {
		return HealthStatusUnhealthy
	}
	
	// Check resource constraints
	if resources.CPUUsage > 95.0 || resources.MemoryUsage > 95.0 || resources.DiskUsage > 95.0 {
		return HealthStatusUnhealthy
	}
	
	// Check for degraded conditions
	
	// High queue depth indicates degraded performance
	tierConfig := tier.GetTierConfig(h.tierDetector.GetCurrentTier())
	if queueDepth > tierConfig.QueueMaxSize/2 {
		return HealthStatusDegraded
	}
	
	// Some adapters in error state indicates degraded performance
	if criticalAdapterErrors > 0 {
		return HealthStatusDegraded
	}
	
	// High resource usage indicates degraded performance
	if resources.CPUUsage > 80.0 || resources.MemoryUsage > 80.0 || resources.DiskUsage > 80.0 {
		return HealthStatusDegraded
	}
	
	// If we get here, system is healthy
	return HealthStatusHealthy
}