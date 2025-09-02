package windows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	// Registry paths for service configuration
	ServiceRegistryPath = `SYSTEM\CurrentControlSet\Services\` + ServiceName
	ServiceParametersPath = ServiceRegistryPath + `\Parameters`
)

// ServiceConfig represents Windows service configuration
type ServiceConfig struct {
	ConfigPath    string
	LogLevel      string
	DataDirectory string
	WorkingDir    string
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() *ServiceConfig {
	// Default to Program Files directory
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		programFiles = `C:\Program Files`
	}
	
	serviceDir := filepath.Join(programFiles, "GymDoorBridge")
	
	return &ServiceConfig{
		ConfigPath:    filepath.Join(serviceDir, "config.yaml"),
		LogLevel:      "info",
		DataDirectory: filepath.Join(serviceDir, "data"),
		WorkingDir:    serviceDir,
	}
}

// SaveServiceConfig saves service configuration to Windows registry
func SaveServiceConfig(config *ServiceConfig) error {
	// Open or create the Parameters key
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, ServiceParametersPath, registry.ALL_ACCESS)
	if err != nil {
		// Try to create the key if it doesn't exist
		key, _, err = registry.CreateKey(registry.LOCAL_MACHINE, ServiceParametersPath, registry.ALL_ACCESS)
		if err != nil {
			return fmt.Errorf("failed to create service parameters registry key: %w", err)
		}
	}
	defer key.Close()
	
	// Save configuration values
	if err := key.SetStringValue("ConfigPath", config.ConfigPath); err != nil {
		return fmt.Errorf("failed to set ConfigPath: %w", err)
	}
	
	if err := key.SetStringValue("LogLevel", config.LogLevel); err != nil {
		return fmt.Errorf("failed to set LogLevel: %w", err)
	}
	
	if err := key.SetStringValue("DataDirectory", config.DataDirectory); err != nil {
		return fmt.Errorf("failed to set DataDirectory: %w", err)
	}
	
	if err := key.SetStringValue("WorkingDir", config.WorkingDir); err != nil {
		return fmt.Errorf("failed to set WorkingDir: %w", err)
	}
	
	return nil
}

// LoadServiceConfig loads service configuration from Windows registry
func LoadServiceConfig() (*ServiceConfig, error) {
	// Try to open the Parameters key
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, ServiceParametersPath, registry.READ)
	if err != nil {
		// Return default config if registry key doesn't exist
		return DefaultServiceConfig(), nil
	}
	defer key.Close()
	
	config := &ServiceConfig{}
	
	// Load configuration values with defaults
	if configPath, _, err := key.GetStringValue("ConfigPath"); err == nil {
		config.ConfigPath = configPath
	} else {
		config.ConfigPath = DefaultServiceConfig().ConfigPath
	}
	
	if logLevel, _, err := key.GetStringValue("LogLevel"); err == nil {
		config.LogLevel = logLevel
	} else {
		config.LogLevel = "info"
	}
	
	if dataDir, _, err := key.GetStringValue("DataDirectory"); err == nil {
		config.DataDirectory = dataDir
	} else {
		config.DataDirectory = DefaultServiceConfig().DataDirectory
	}
	
	if workingDir, _, err := key.GetStringValue("WorkingDir"); err == nil {
		config.WorkingDir = workingDir
	} else {
		config.WorkingDir = DefaultServiceConfig().WorkingDir
	}
	
	return config, nil
}

// RemoveServiceConfig removes service configuration from Windows registry
func RemoveServiceConfig() error {
	// Try to delete the Parameters key
	err := registry.DeleteKey(registry.LOCAL_MACHINE, ServiceParametersPath)
	if err != nil && !strings.Contains(err.Error(), "cannot find") {
		return fmt.Errorf("failed to remove service configuration: %w", err)
	}
	
	return nil
}

// CreateServiceDirectories creates necessary directories for service operation
func CreateServiceDirectories(config *ServiceConfig) error {
	directories := []string{
		config.WorkingDir,
		config.DataDirectory,
		filepath.Dir(config.ConfigPath),
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