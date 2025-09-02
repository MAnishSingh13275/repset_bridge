package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/tier"
)

// HTTPClient interface for sending heartbeats
type HTTPClient interface {
	SendHeartbeat(ctx context.Context, heartbeat *client.HeartbeatRequest) error
}

// HeartbeatConfig holds configuration for the heartbeat manager
type HeartbeatConfig struct {
	Interval          time.Duration `json:"interval"`          // Interval between heartbeats
	Timeout           time.Duration `json:"timeout"`           // Timeout for heartbeat requests
	MaxRetries        int           `json:"maxRetries"`        // Maximum number of retry attempts
	RetryBackoff      time.Duration `json:"retryBackoff"`      // Backoff time between retries
	EnableSystemInfo  bool          `json:"enableSystemInfo"`  // Include system resource info
}

// GetTierHeartbeatConfig returns heartbeat configuration for the specified tier
func GetTierHeartbeatConfig(t tier.Tier) HeartbeatConfig {
	tierConfig := tier.GetTierConfig(t)
	
	return HeartbeatConfig{
		Interval:         tierConfig.HeartbeatInterval,
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		RetryBackoff:     10 * time.Second,
		EnableSystemInfo: t == tier.TierFull, // Only include detailed system info for Full tier
	}
}

// HeartbeatManager manages periodic heartbeat messages to the cloud
type HeartbeatManager struct {
	mu           sync.RWMutex
	config       HeartbeatConfig
	logger       *logrus.Logger
	httpClient   HTTPClient
	healthMonitor *HealthMonitor
	
	// State
	isRunning    bool
	lastSent     time.Time
	lastError    error
	sendCount    int64
	errorCount   int64
	
	// Control
	stopCh       chan struct{}
	stoppedCh    chan struct{}
}

// HeartbeatManagerOption is a functional option for configuring the HeartbeatManager
type HeartbeatManagerOption func(*HeartbeatManager)

// WithHeartbeatLogger sets the logger for the heartbeat manager
func WithHeartbeatLogger(logger *logrus.Logger) HeartbeatManagerOption {
	return func(h *HeartbeatManager) {
		h.logger = logger
	}
}

// NewHeartbeatManager creates a new heartbeat manager
func NewHeartbeatManager(
	config HeartbeatConfig,
	httpClient HTTPClient,
	healthMonitor *HealthMonitor,
	opts ...HeartbeatManagerOption,
) *HeartbeatManager {
	h := &HeartbeatManager{
		config:        config,
		logger:        logrus.New(),
		httpClient:    httpClient,
		healthMonitor: healthMonitor,
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}
	
	// Apply options
	for _, opt := range opts {
		opt(h)
	}
	
	return h
}

// Start begins sending periodic heartbeats
func (h *HeartbeatManager) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.isRunning {
		h.mu.Unlock()
		return fmt.Errorf("heartbeat manager is already running")
	}
	h.isRunning = true
	h.mu.Unlock()
	
	h.logger.Info("Starting heartbeat manager", "interval", h.config.Interval)
	
	// Send initial heartbeat
	if err := h.sendHeartbeat(ctx); err != nil {
		h.logger.WithError(err).Warn("Failed to send initial heartbeat")
	}
	
	// Start heartbeat loop
	go h.heartbeatLoop(ctx)
	
	return nil
}

// Stop gracefully stops the heartbeat manager
func (h *HeartbeatManager) Stop(ctx context.Context) error {
	h.mu.Lock()
	if !h.isRunning {
		h.mu.Unlock()
		return nil
	}
	h.mu.Unlock()
	
	h.logger.Info("Stopping heartbeat manager")
	
	// Signal stop
	close(h.stopCh)
	
	// Wait for heartbeat loop to stop with timeout
	select {
	case <-h.stoppedCh:
		h.logger.Info("Heartbeat manager stopped")
	case <-ctx.Done():
		h.logger.Warn("Heartbeat manager stop timed out")
		return ctx.Err()
	case <-time.After(10 * time.Second):
		h.logger.Warn("Heartbeat manager stop timed out after 10 seconds")
	}
	
	h.mu.Lock()
	h.isRunning = false
	h.mu.Unlock()
	
	return nil
}

// GetStats returns heartbeat statistics
func (h *HeartbeatManager) GetStats() HeartbeatStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	return HeartbeatStats{
		IsRunning:   h.isRunning,
		LastSent:    h.lastSent,
		LastError:   h.lastError,
		SendCount:   h.sendCount,
		ErrorCount:  h.errorCount,
		Interval:    h.config.Interval,
	}
}

// HeartbeatStats contains statistics about heartbeat operations
type HeartbeatStats struct {
	IsRunning   bool          `json:"isRunning"`
	LastSent    time.Time     `json:"lastSent"`
	LastError   error         `json:"lastError,omitempty"`
	SendCount   int64         `json:"sendCount"`
	ErrorCount  int64         `json:"errorCount"`
	Interval    time.Duration `json:"interval"`
}

// heartbeatLoop runs the main heartbeat loop
func (h *HeartbeatManager) heartbeatLoop(ctx context.Context) {
	defer close(h.stoppedCh)
	
	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Heartbeat loop stopped due to context cancellation")
			return
		case <-h.stopCh:
			h.logger.Info("Heartbeat loop stopped")
			return
		case <-ticker.C:
			if err := h.sendHeartbeat(ctx); err != nil {
				h.mu.Lock()
				h.lastError = err
				h.errorCount++
				h.mu.Unlock()
				
				h.logger.WithError(err).Error("Failed to send heartbeat")
			} else {
				h.mu.Lock()
				h.lastSent = time.Now()
				h.sendCount++
				h.lastError = nil
				h.mu.Unlock()
				
				h.logger.Debug("Heartbeat sent successfully")
			}
		}
	}
}

// sendHeartbeat sends a single heartbeat to the cloud
func (h *HeartbeatManager) sendHeartbeat(ctx context.Context) error {
	// Get current health status
	health := h.healthMonitor.GetCurrentHealth()
	
	// Build heartbeat request
	heartbeat := &client.HeartbeatRequest{
		Status:     health.Status.String(),
		Tier:       health.Tier.String(),
		QueueDepth: health.QueueDepth,
	}
	
	// Add last event time if available
	if health.LastEventTime != nil {
		heartbeat.LastEventTime = health.LastEventTime.Format(time.RFC3339)
	}
	
	// Add system info if enabled
	if h.config.EnableSystemInfo {
		heartbeat.SystemInfo = &client.SystemInfo{
			CPUUsage:    health.Resources.CPUUsage,
			MemoryUsage: health.Resources.MemoryUsage,
			DiskSpace:   health.Resources.DiskUsage,
		}
	}
	
	// Send heartbeat with retries
	return h.sendHeartbeatWithRetries(ctx, heartbeat)
}

// sendHeartbeatWithRetries sends a heartbeat with retry logic
func (h *HeartbeatManager) sendHeartbeatWithRetries(ctx context.Context, heartbeat *client.HeartbeatRequest) error {
	var lastErr error
	
	for attempt := 0; attempt <= h.config.MaxRetries; attempt++ {
		// Create timeout context for this attempt
		timeoutCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
		
		// Send heartbeat
		err := h.httpClient.SendHeartbeat(timeoutCtx, heartbeat)
		cancel()
		
		if err == nil {
			return nil // Success
		}
		
		lastErr = err
		
		// Don't retry on the last attempt
		if attempt == h.config.MaxRetries {
			break
		}
		
		// Wait before retrying
		h.logger.WithError(err).Warnf("Heartbeat attempt %d failed, retrying in %v", attempt+1, h.config.RetryBackoff)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.stopCh:
			return fmt.Errorf("heartbeat manager stopped")
		case <-time.After(h.config.RetryBackoff):
			// Continue to next attempt
		}
	}
	
	return fmt.Errorf("heartbeat failed after %d attempts: %w", h.config.MaxRetries+1, lastErr)
}

// UpdateConfig updates the heartbeat configuration
func (h *HeartbeatManager) UpdateConfig(config HeartbeatConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.config = config
	h.logger.Info("Heartbeat configuration updated", "interval", config.Interval)
}