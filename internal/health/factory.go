package health

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/tier"
)

// HealthSystemConfig holds configuration for the complete health monitoring system
type HealthSystemConfig struct {
	Health    HealthCheckConfig `json:"health"`
	Heartbeat HeartbeatConfig   `json:"heartbeat"`
	Metrics   MetricsConfig     `json:"metrics"`
	Version   string            `json:"version"`
	DeviceID  string            `json:"deviceId"`
}

// HealthSystem represents the complete health monitoring system
type HealthSystem struct {
	Monitor           *HealthMonitor
	HeartbeatManager  *HeartbeatManager
	AdapterRegistry   *SimpleAdapterRegistry
	MetricsExporter   MetricsExporter
}

// TierDetector interface for getting tier information
type TierDetector interface {
	GetCurrentTier() tier.Tier
	GetCurrentResources() tier.SystemResources
}

// NewHealthSystem creates a complete health monitoring system
func NewHealthSystem(
	config HealthSystemConfig,
	queueManager queue.QueueManager,
	tierDetector TierDetector,
	httpClient HTTPClient,
	logger *logrus.Logger,
) (*HealthSystem, error) {
	// Create adapter registry
	adapterRegistry := NewSimpleAdapterRegistry()
	
	// Create metrics exporter
	var metricsExporter MetricsExporter
	if config.Metrics.Enabled {
		metricsExporter = NewPrometheusMetricsExporter(
			config.Metrics,
			WithMetricsLogger(logger),
		)
	} else {
		metricsExporter = NewNoOpMetricsExporter()
	}
	
	// Create health monitor
	healthMonitor := NewHealthMonitor(
		config.Health,
		queueManager,
		tierDetector,
		adapterRegistry,
		WithLogger(logger),
		WithVersion(config.Version),
		WithDeviceID(config.DeviceID),
		WithMetricsExporter(metricsExporter),
	)
	
	// Create heartbeat manager
	heartbeatManager := NewHeartbeatManager(
		config.Heartbeat,
		httpClient,
		healthMonitor,
		WithHeartbeatLogger(logger),
	)
	
	return &HealthSystem{
		Monitor:          healthMonitor,
		HeartbeatManager: heartbeatManager,
		AdapterRegistry:  adapterRegistry,
		MetricsExporter:  metricsExporter,
	}, nil
}

// Start starts all components of the health system
func (h *HealthSystem) Start(ctx context.Context) error {
	// Start health monitor
	if err := h.Monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}
	
	// Start heartbeat manager
	if err := h.HeartbeatManager.Start(ctx); err != nil {
		// Stop health monitor if heartbeat fails to start
		if stopErr := h.Monitor.Stop(ctx); stopErr != nil {
			return fmt.Errorf("failed to start heartbeat manager: %w, and failed to stop health monitor: %v", err, stopErr)
		}
		return fmt.Errorf("failed to start heartbeat manager: %w", err)
	}
	
	return nil
}

// Stop stops all components of the health system
func (h *HealthSystem) Stop(ctx context.Context) error {
	var errors []error
	
	// Stop heartbeat manager
	if err := h.HeartbeatManager.Stop(ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop heartbeat manager: %w", err))
	}
	
	// Stop health monitor
	if err := h.Monitor.Stop(ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop health monitor: %w", err))
	}
	
	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("errors stopping health system: %v", errors)
	}
	
	return nil
}

// GetDefaultHealthSystemConfig returns default configuration for the health system
func GetDefaultHealthSystemConfig(currentTier tier.Tier) HealthSystemConfig {
	return HealthSystemConfig{
		Health:    DefaultHealthCheckConfig(),
		Heartbeat: GetTierHeartbeatConfig(currentTier),
		Metrics:   DefaultMetricsConfig(),
		Version:   "unknown",
		DeviceID:  "",
	}
}

// UpdateTierConfiguration updates the configuration based on the current tier
func (h *HealthSystem) UpdateTierConfiguration(newTier tier.Tier) {
	// Update heartbeat configuration based on new tier
	newHeartbeatConfig := GetTierHeartbeatConfig(newTier)
	h.HeartbeatManager.UpdateConfig(newHeartbeatConfig)
}