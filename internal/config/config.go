package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gym-door-bridge/internal/types"
	"github.com/spf13/viper"
)

// Config represents the bridge configuration
type Config struct {
	// Device configuration
	DeviceID  string `mapstructure:"device_id"`
	DeviceKey string `mapstructure:"device_key"`
	
	// Server configuration
	ServerURL string `mapstructure:"server_url"`
	
	// Performance tier configuration
	Tier string `mapstructure:"tier"` // lite, normal, full
	
	// Queue configuration
	QueueMaxSize     int `mapstructure:"queue_max_size"`
	HeartbeatInterval int `mapstructure:"heartbeat_interval"` // seconds
	
	// Door control configuration
	UnlockDuration int `mapstructure:"unlock_duration"` // milliseconds
	
	// Database configuration
	DatabasePath string `mapstructure:"database_path"`
	
	// Logging configuration
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`
	
	// Adapter configuration
	EnabledAdapters []string                           `mapstructure:"enabled_adapters"`
	AdapterConfigs  map[string]map[string]interface{} `mapstructure:"adapter_configs"`
	
	// Update configuration
	UpdatesEnabled    bool   `mapstructure:"updates_enabled"`
	UpdateManifestURL string `mapstructure:"update_manifest_url"`
	UpdatePublicKey   string `mapstructure:"update_public_key"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		ServerURL:         "https://api.yourdomain.com",
		Tier:              "normal",
		QueueMaxSize:      10000,
		HeartbeatInterval: 60,
		UnlockDuration:    3000,
		DatabasePath:      "./bridge.db",
		LogLevel:          "info",
		LogFile:           "",
		EnabledAdapters:   []string{"simulator"},
		AdapterConfigs:    make(map[string]map[string]interface{}),
		UpdatesEnabled:    true,
		UpdateManifestURL: "",
		UpdatePublicKey:   "",
	}
}

// Load loads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	cfg := DefaultConfig()
	
	// Set up viper
	v := viper.New()
	
	// Set default values
	setDefaults(v, cfg)
	
	// Configure file locations
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Look for config in current directory and common locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/gym-door-bridge")
		
		// Add user config directory
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".gym-door-bridge"))
		}
	}
	
	// Environment variable configuration
	v.SetEnvPrefix("BRIDGE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}
	
	// Unmarshal into struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return cfg, nil
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper, cfg *Config) {
	v.SetDefault("server_url", cfg.ServerURL)
	v.SetDefault("tier", cfg.Tier)
	v.SetDefault("queue_max_size", cfg.QueueMaxSize)
	v.SetDefault("heartbeat_interval", cfg.HeartbeatInterval)
	v.SetDefault("unlock_duration", cfg.UnlockDuration)
	v.SetDefault("database_path", cfg.DatabasePath)
	v.SetDefault("log_level", cfg.LogLevel)
	v.SetDefault("log_file", cfg.LogFile)
	v.SetDefault("enabled_adapters", cfg.EnabledAdapters)
	v.SetDefault("adapter_configs", cfg.AdapterConfigs)
	v.SetDefault("updates_enabled", cfg.UpdatesEnabled)
	v.SetDefault("update_manifest_url", cfg.UpdateManifestURL)
	v.SetDefault("update_public_key", cfg.UpdatePublicKey)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}
	
	if c.Tier != "lite" && c.Tier != "normal" && c.Tier != "full" {
		return fmt.Errorf("tier must be one of: lite, normal, full")
	}
	
	if c.QueueMaxSize <= 0 {
		return fmt.Errorf("queue_max_size must be positive")
	}
	
	if c.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat_interval must be positive")
	}
	
	if c.UnlockDuration <= 0 {
		return fmt.Errorf("unlock_duration must be positive")
	}
	
	if c.DatabasePath == "" {
		return fmt.Errorf("database_path is required")
	}
	
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error")
	}
	
	return nil
}

// IsPaired returns true if the device has been paired with the cloud
func (c *Config) IsPaired() bool {
	return c.DeviceID != "" && c.DeviceKey != ""
}

// GetAdapterConfigs converts the configuration to adapter configs
func (c *Config) GetAdapterConfigs() []types.AdapterConfig {
	var configs []types.AdapterConfig
	
	for _, adapterName := range c.EnabledAdapters {
		config := types.AdapterConfig{
			Name:    adapterName,
			Enabled: true,
		}
		
		// Add specific settings if configured
		if settings, exists := c.AdapterConfigs[adapterName]; exists {
			config.Settings = settings
		} else {
			config.Settings = make(map[string]interface{})
		}
		
		configs = append(configs, config)
	}
	
	return configs
}