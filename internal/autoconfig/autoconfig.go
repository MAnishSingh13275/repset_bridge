package autoconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/discovery"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// AutoConfig handles automatic configuration updates
type AutoConfig struct {
	logger        *logrus.Logger
	configPath    string
	discovery     *discovery.DeviceDiscovery
	lastUpdate    time.Time
	updateInterval time.Duration
}

// NewAutoConfig creates a new auto-configuration manager
func NewAutoConfig(configPath string, logger *logrus.Logger) *AutoConfig {
	return &AutoConfig{
		logger:         logger,
		configPath:     configPath,
		discovery:      discovery.NewDeviceDiscovery(logger),
		updateInterval: 5 * time.Minute, // Check for changes every 5 minutes
	}
}

// Start begins auto-configuration monitoring
func (ac *AutoConfig) Start(ctx context.Context) error {
	ac.logger.Info("Starting auto-configuration service")

	// Start device discovery
	if err := ac.discovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start device discovery: %w", err)
	}

	// Start configuration update loop
	go ac.configUpdateLoop(ctx)

	return nil
}

// Stop stops auto-configuration monitoring
func (ac *AutoConfig) Stop() error {
	ac.logger.Info("Stopping auto-configuration service")
	
	if ac.discovery != nil {
		return ac.discovery.Stop()
	}
	
	return nil
}

// configUpdateLoop periodically checks for device changes and updates config
func (ac *AutoConfig) configUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(ac.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ac.updateConfigIfNeeded(); err != nil {
				ac.logger.WithError(err).Error("Failed to update configuration")
			}
		}
	}
}

// updateConfigIfNeeded updates configuration if devices have changed
func (ac *AutoConfig) updateConfigIfNeeded() error {
	// Get current discovered devices
	devices := ac.discovery.GetDiscoveredDevices()
	
	// Load current configuration
	currentConfig, err := ac.loadCurrentConfig()
	if err != nil {
		return fmt.Errorf("failed to load current config: %w", err)
	}

	// Generate new adapter configuration
	newAdapterConfig := ac.discovery.GenerateAdapterConfig()
	
	// Check if configuration has changed
	if ac.hasConfigChanged(currentConfig, newAdapterConfig) {
		ac.logger.Info("Device configuration changed, updating config file")
		
		// Update configuration
		if err := ac.updateConfiguration(currentConfig, newAdapterConfig); err != nil {
			return fmt.Errorf("failed to update configuration: %w", err)
		}
		
		ac.lastUpdate = time.Now()
		ac.logger.Info("Configuration updated successfully")
		
		// Log discovered devices
		for _, device := range devices {
			ac.logger.WithFields(logrus.Fields{
				"type":   device.Type,
				"ip":     device.IP,
				"port":   device.Port,
				"model":  device.Model,
				"status": device.Status,
			}).Info("Device configured")
		}
	}

	return nil
}

// loadCurrentConfig loads the current configuration file
func (ac *AutoConfig) loadCurrentConfig() (*config.Config, error) {
	if _, err := os.Stat(ac.configPath); os.IsNotExist(err) {
		// Config file doesn't exist, return default config
		return ac.getDefaultConfig(), nil
	}

	data, err := os.ReadFile(ac.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// hasConfigChanged checks if the adapter configuration has changed
func (ac *AutoConfig) hasConfigChanged(currentConfig *config.Config, newAdapterConfig map[string]interface{}) bool {
	// Compare enabled adapters
	newEnabledAdapters := newAdapterConfig["enabled_adapters"].([]string)
	if len(currentConfig.EnabledAdapters) != len(newEnabledAdapters) {
		return true
	}

	// Check if enabled adapters are the same
	enabledMap := make(map[string]bool)
	for _, adapter := range currentConfig.EnabledAdapters {
		enabledMap[adapter] = true
	}

	for _, adapter := range newEnabledAdapters {
		if !enabledMap[adapter] {
			return true
		}
	}

	// Compare adapter configs (simplified check)
	newAdapterConfigs := newAdapterConfig["adapter_configs"].(map[string]interface{})
	if len(currentConfig.AdapterConfigs) != len(newAdapterConfigs) {
		return true
	}

	// Check if any adapter config has changed
	for name := range newAdapterConfigs {
		if _, exists := currentConfig.AdapterConfigs[name]; !exists {
			return true
		}
	}

	return false
}

// updateConfiguration updates the configuration file with new adapter settings
func (ac *AutoConfig) updateConfiguration(currentConfig *config.Config, newAdapterConfig map[string]interface{}) error {
	// Update adapter configuration
	currentConfig.EnabledAdapters = newAdapterConfig["enabled_adapters"].([]string)
	currentConfig.AdapterConfigs = newAdapterConfig["adapter_configs"].(map[string]interface{})

	// Create backup of current config
	backupPath := ac.configPath + ".backup." + time.Now().Format("20060102-150405")
	if err := ac.copyFile(ac.configPath, backupPath); err != nil {
		ac.logger.WithError(err).Warn("Failed to create config backup")
	}

	// Marshal updated configuration
	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write updated configuration
	if err := os.WriteFile(ac.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// getDefaultConfig returns a default configuration
func (ac *AutoConfig) getDefaultConfig() *config.Config {
	return &config.Config{
		ServerURL:    "https://your-platform.com",
		DeviceID:     "",
		DeviceKey:    "",
		DatabasePath: "./bridge.db",
		LogLevel:     "info",
		LogFile:      "",
		Tier:         "normal",
		
		APIServer: config.APIServerConfig{
			Enabled: true,
			Host:    "localhost",
			Port:    8081,
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key", "X-Requested-With"},
			},
		},
		
		EnabledAdapters: []string{"simulator"},
		AdapterConfigs: map[string]interface{}{
			"simulator": map[string]interface{}{
				"device_type":   "simulator",
				"connection":    "memory",
				"device_config": map[string]string{},
				"sync_interval": 10,
			},
		},
		
		HeartbeatInterval: 60,
		QueueMaxSize:     10000,
		UnlockDuration:   3000,
		UpdatesEnabled:   true,
	}
}

// copyFile copies a file from src to dst
func (ac *AutoConfig) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// GetDiscoveredDevices returns currently discovered devices
func (ac *AutoConfig) GetDiscoveredDevices() map[string]*discovery.DeviceInfo {
	if ac.discovery == nil {
		return make(map[string]*discovery.DeviceInfo)
	}
	return ac.discovery.GetDiscoveredDevices()
}