package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gym-door-bridge/internal/adapters/fingerprint"
	"gym-door-bridge/internal/adapters/rfid"
	"gym-door-bridge/internal/adapters/simulator"
	"gym-door-bridge/internal/adapters/webhook"
	"gym-door-bridge/internal/types"
)

// AdapterManager manages the lifecycle of hardware adapters
type AdapterManager struct {
	adapters      map[string]HardwareAdapter
	configs       map[string]types.AdapterConfig
	eventCallback types.EventCallback
	logger        *slog.Logger
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// AdapterFactory is a function that creates a new adapter instance
type AdapterFactory func(*slog.Logger) HardwareAdapter

// registeredAdapters holds the registry of available adapter types
var registeredAdapters = map[string]AdapterFactory{
	"simulator":   func(logger *slog.Logger) HardwareAdapter { return simulator.NewSimulatorAdapter(logger) },
	"webhook":     func(logger *slog.Logger) HardwareAdapter { return webhook.NewWebhookAdapter(logger) },
	"fingerprint": func(logger *slog.Logger) HardwareAdapter { return fingerprint.NewFingerprintAdapter(logger) },
	"rfid":        func(logger *slog.Logger) HardwareAdapter { return rfid.NewRFIDAdapter(logger) },
}

// NewAdapterManager creates a new adapter manager instance
func NewAdapterManager(logger *slog.Logger) *AdapterManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AdapterManager{
		adapters: make(map[string]HardwareAdapter),
		configs:  make(map[string]types.AdapterConfig),
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterAdapter registers a new adapter type with the manager
func RegisterAdapter(name string, factory AdapterFactory) {
	registeredAdapters[name] = factory
}

// GetRegisteredAdapterTypes returns a list of all registered adapter types
func GetRegisteredAdapterTypes() []string {
	types := make([]string, 0, len(registeredAdapters))
	for name := range registeredAdapters {
		types = append(types, name)
	}
	return types
}

// LoadAdapters loads and initializes adapters based on configuration
func (am *AdapterManager) LoadAdapters(configs []types.AdapterConfig) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.logger.Info("Loading adapters", "count", len(configs))

	for _, config := range configs {
		if err := am.loadAdapter(config); err != nil {
			am.logger.Error("Failed to load adapter",
				"name", config.Name,
				"error", err)
			continue
		}
	}

	am.logger.Info("Adapters loaded successfully",
		"total", len(configs),
		"active", len(am.adapters))

	return nil
}

// loadAdapter loads a single adapter based on configuration
func (am *AdapterManager) loadAdapter(config types.AdapterConfig) error {
	// Check if adapter type is registered
	factory, exists := registeredAdapters[config.Name]
	if !exists {
		return fmt.Errorf("unknown adapter type: %s", config.Name)
	}

	// Skip disabled adapters
	if !config.Enabled {
		am.logger.Info("Skipping disabled adapter", "name", config.Name)
		return nil
	}

	// Create adapter instance
	adapter := factory(am.logger)
	
	// Initialize adapter
	if err := adapter.Initialize(am.ctx, config); err != nil {
		return fmt.Errorf("failed to initialize adapter %s: %w", config.Name, err)
	}

	// Register event callback if set
	if am.eventCallback != nil {
		adapter.OnEvent(am.eventCallback)
	}

	// Store adapter and config
	am.adapters[config.Name] = adapter
	am.configs[config.Name] = config

	am.logger.Info("Adapter loaded successfully", "name", config.Name)
	return nil
}

// StartAll starts all loaded adapters
func (am *AdapterManager) StartAll() error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	am.logger.Info("Starting all adapters", "count", len(am.adapters))

	var errors []error
	for name, adapter := range am.adapters {
		if err := adapter.StartListening(am.ctx); err != nil {
			am.logger.Error("Failed to start adapter",
				"name", name,
				"error", err)
			errors = append(errors, fmt.Errorf("adapter %s: %w", name, err))
			continue
		}
		am.logger.Info("Adapter started successfully", "name", name)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start %d adapters: %v", len(errors), errors)
	}

	am.logger.Info("All adapters started successfully")
	return nil
}

// StopAll stops all running adapters
func (am *AdapterManager) StopAll() error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	am.logger.Info("Stopping all adapters", "count", len(am.adapters))

	var errors []error
	for name, adapter := range am.adapters {
		if err := adapter.StopListening(am.ctx); err != nil {
			am.logger.Error("Failed to stop adapter",
				"name", name,
				"error", err)
			errors = append(errors, fmt.Errorf("adapter %s: %w", name, err))
			continue
		}
		am.logger.Info("Adapter stopped successfully", "name", name)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop %d adapters: %v", len(errors), errors)
	}

	am.logger.Info("All adapters stopped successfully")
	return nil
}

// Shutdown gracefully shuts down the adapter manager
func (am *AdapterManager) Shutdown() error {
	am.logger.Info("Shutting down adapter manager")
	
	// Stop all adapters first
	if err := am.StopAll(); err != nil {
		am.logger.Error("Error stopping adapters during shutdown", "error", err)
	}
	
	// Cancel context
	am.cancel()
	
	// Clear adapters
	am.mutex.Lock()
	am.adapters = make(map[string]HardwareAdapter)
	am.configs = make(map[string]types.AdapterConfig)
	am.mutex.Unlock()
	
	am.logger.Info("Adapter manager shutdown complete")
	return nil
}

// GetAdapter returns a specific adapter by name
func (am *AdapterManager) GetAdapter(name string) (HardwareAdapter, bool) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	adapter, exists := am.adapters[name]
	return adapter, exists
}

// GetAllAdapters returns all loaded adapters
func (am *AdapterManager) GetAllAdapters() map[string]HardwareAdapter {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]HardwareAdapter)
	for name, adapter := range am.adapters {
		result[name] = adapter
	}
	return result
}

// GetAdapterStatus returns the status of all adapters
func (am *AdapterManager) GetAdapterStatus() map[string]types.AdapterStatus {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	status := make(map[string]types.AdapterStatus)
	for name, adapter := range am.adapters {
		status[name] = adapter.GetStatus()
	}
	return status
}

// GetHealthyAdapters returns a list of healthy adapter names
func (am *AdapterManager) GetHealthyAdapters() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	var healthy []string
	for name, adapter := range am.adapters {
		if adapter.IsHealthy() {
			healthy = append(healthy, name)
		}
	}
	return healthy
}

// OnEvent registers a callback for all adapter events
func (am *AdapterManager) OnEvent(callback types.EventCallback) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	am.eventCallback = callback
	
	// Register callback with all existing adapters
	for _, adapter := range am.adapters {
		adapter.OnEvent(callback)
	}
}

// UnlockDoor attempts to unlock the door using the first available adapter that supports it
func (am *AdapterManager) UnlockDoor(durationMs int) error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	var lastError error
	for name, adapter := range am.adapters {
		if !adapter.IsHealthy() {
			continue
		}
		
		if err := adapter.UnlockDoor(am.ctx, durationMs); err != nil {
			am.logger.Debug("Adapter does not support door unlock",
				"adapter", name,
				"error", err)
			lastError = err
			continue
		}
		
		am.logger.Info("Door unlocked successfully", "adapter", name)
		return nil
	}
	
	if lastError != nil {
		return fmt.Errorf("no adapters support door unlock: %w", lastError)
	}
	
	return fmt.Errorf("no healthy adapters available")
}

// ReloadAdapter reloads a specific adapter with new configuration
func (am *AdapterManager) ReloadAdapter(config types.AdapterConfig) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	// Stop existing adapter if it exists
	if existing, exists := am.adapters[config.Name]; exists {
		if err := existing.StopListening(am.ctx); err != nil {
			am.logger.Error("Failed to stop existing adapter",
				"name", config.Name,
				"error", err)
		}
		delete(am.adapters, config.Name)
		delete(am.configs, config.Name)
	}
	
	// Load new adapter
	if err := am.loadAdapter(config); err != nil {
		return fmt.Errorf("failed to reload adapter %s: %w", config.Name, err)
	}
	
	// Start the new adapter if enabled
	if config.Enabled {
		if adapter, exists := am.adapters[config.Name]; exists {
			if err := adapter.StartListening(am.ctx); err != nil {
				am.logger.Error("Failed to start reloaded adapter",
					"name", config.Name,
					"error", err)
				return fmt.Errorf("failed to start reloaded adapter %s: %w", config.Name, err)
			}
		}
	}
	
	am.logger.Info("Adapter reloaded successfully", "name", config.Name)
	return nil
}

// MonitorHealth periodically checks adapter health and logs status
func (am *AdapterManager) MonitorHealth(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			am.checkAdapterHealth()
		case <-am.ctx.Done():
			return
		}
	}
}

// checkAdapterHealth checks the health of all adapters
func (am *AdapterManager) checkAdapterHealth() {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	for name, adapter := range am.adapters {
		status := adapter.GetStatus()
		if !adapter.IsHealthy() {
			am.logger.Warn("Adapter health check failed",
				"name", name,
				"status", status.Status,
				"error", status.ErrorMessage,
				"lastEvent", status.LastEvent)
		} else {
			am.logger.Debug("Adapter health check passed",
				"name", name,
				"status", status.Status,
				"lastEvent", status.LastEvent)
		}
	}
}