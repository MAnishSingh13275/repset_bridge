package macos

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ServiceConfig represents macOS daemon configuration
type ServiceConfig struct {
	ConfigPath    string
	LogLevel      string
	DataDirectory string
	WorkingDir    string
	LogPath       string
}

// DefaultServiceConfig returns default service configuration for macOS
func DefaultServiceConfig() *ServiceConfig {
	// Default to /usr/local directory structure
	serviceDir := "/usr/local/lib/gymdoorbridge"
	
	return &ServiceConfig{
		ConfigPath:    "/usr/local/etc/gymdoorbridge/config.yaml",
		LogLevel:      "info",
		DataDirectory: "/usr/local/var/lib/gymdoorbridge",
		WorkingDir:    serviceDir,
		LogPath:       "/usr/local/var/log/gymdoorbridge/bridge.log",
	}
}

// LoadServiceConfig loads service configuration from plist or returns defaults
// On macOS, configuration is typically embedded in the plist file
func LoadServiceConfig() (*ServiceConfig, error) {
	// For macOS, we primarily use default configuration
	// Configuration can be overridden via command line arguments in the plist
	return DefaultServiceConfig(), nil
}

// CreateServiceDirectories creates necessary directories for service operation
func CreateServiceDirectories(config *ServiceConfig) error {
	directories := []string{
		config.WorkingDir,
		config.DataDirectory,
		filepath.Dir(config.ConfigPath),
		filepath.Dir(config.LogPath),
	}
	
	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// ValidateServiceConfig validates service configuration
func ValidateServiceConfig(config *ServiceConfig) error {
	if config.ConfigPath == "" {
		return fmt.Errorf("config path cannot be empty")
	}
	
	if config.DataDirectory == "" {
		return fmt.Errorf("data directory cannot be empty")
	}
	
	if config.WorkingDir == "" {
		return fmt.Errorf("working directory cannot be empty")
	}
	
	if config.LogPath == "" {
		return fmt.Errorf("log path cannot be empty")
	}
	
	validLogLevels := []string{"debug", "info", "warn", "error"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if config.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	
	if !isValidLogLevel {
		return fmt.Errorf("invalid log level: %s (must be one of: %s)", 
			config.LogLevel, strings.Join(validLogLevels, ", "))
	}
	
	return nil
}

// SetDirectoryPermissions sets appropriate permissions for service directories
func SetDirectoryPermissions(config *ServiceConfig) error {
	// Set permissions for data directory (readable/writable by service)
	if err := os.Chmod(config.DataDirectory, 0755); err != nil {
		return fmt.Errorf("failed to set permissions on data directory: %w", err)
	}
	
	// Set permissions for log directory
	logDir := filepath.Dir(config.LogPath)
	if err := os.Chmod(logDir, 0755); err != nil {
		return fmt.Errorf("failed to set permissions on log directory: %w", err)
	}
	
	// Set permissions for config directory
	configDir := filepath.Dir(config.ConfigPath)
	if err := os.Chmod(configDir, 0755); err != nil {
		return fmt.Errorf("failed to set permissions on config directory: %w", err)
	}
	
	return nil
}

// CreateDefaultConfigFile creates a default configuration file if it doesn't exist
func CreateDefaultConfigFile(configPath string) error {
	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // File already exists
	}
	
	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Default configuration content
	defaultConfig := `# Gym Door Bridge Configuration
# This is the default configuration file for macOS daemon

# Logging configuration
log:
  level: info
  format: json

# Hardware adapter configuration
adapters:
  simulator:
    enabled: true
  
# Performance tier settings (auto-detected)
performance:
  tier: auto

# Network configuration
network:
  timeout: 30s
  retry_attempts: 3

# Queue configuration
queue:
  max_size: 10000
  batch_size: 100

# Security configuration
security:
  device_id: ""
  device_key: ""

# Door control configuration
door:
  unlock_duration: 3s
  auto_relock: true
`
	
	// Write default configuration
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write default config file: %w", err)
	}
	
	fmt.Printf("Created default configuration file: %s\n", configPath)
	return nil
}