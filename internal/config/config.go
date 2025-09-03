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
	QueueMaxSize      int `mapstructure:"queue_max_size"`
	HeartbeatInterval int `mapstructure:"heartbeat_interval"` // seconds

	// Door control configuration
	UnlockDuration int `mapstructure:"unlock_duration"` // milliseconds

	// Database configuration
	DatabasePath string `mapstructure:"database_path"`

	// Logging configuration
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`

	// Adapter configuration
	EnabledAdapters []string                          `mapstructure:"enabled_adapters"`
	AdapterConfigs  map[string]map[string]interface{} `mapstructure:"adapter_configs"`

	// Update configuration
	UpdatesEnabled    bool   `mapstructure:"updates_enabled"`
	UpdateManifestURL string `mapstructure:"update_manifest_url"`
	UpdatePublicKey   string `mapstructure:"update_public_key"`

	// API server configuration
	APIServer APIServerConfig `mapstructure:"api_server"`
}

// APIServerConfig holds API server specific configuration
type APIServerConfig struct {
	Enabled      bool            `mapstructure:"enabled"`
	Port         int             `mapstructure:"port"`
	Host         string          `mapstructure:"host"`
	TLSEnabled   bool            `mapstructure:"tls_enabled"`
	TLSCertFile  string          `mapstructure:"tls_cert_file"`
	TLSKeyFile   string          `mapstructure:"tls_key_file"`
	ReadTimeout  int             `mapstructure:"read_timeout"`
	WriteTimeout int             `mapstructure:"write_timeout"`
	IdleTimeout  int             `mapstructure:"idle_timeout"`
	Auth         AuthConfig      `mapstructure:"auth"`
	RateLimit    RateLimitConfig `mapstructure:"rate_limit"`
	CORS         CORSConfig      `mapstructure:"cors"`
	Security     SecurityConfig  `mapstructure:"security"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	HMACSecret  string   `mapstructure:"hmac_secret"`
	JWTSecret   string   `mapstructure:"jwt_secret"`
	APIKeys     []string `mapstructure:"api_keys"`
	TokenExpiry int      `mapstructure:"token_expiry"` // seconds
	AllowedIPs  []string `mapstructure:"allowed_ips"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	RequestsPerMin  int  `mapstructure:"requests_per_minute"`
	BurstSize       int  `mapstructure:"burst_size"`
	WindowSize      int  `mapstructure:"window_size"`      // seconds
	CleanupInterval int  `mapstructure:"cleanup_interval"` // seconds
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"` // seconds
}

// SecurityConfig holds security headers configuration
type SecurityConfig struct {
	HSTSEnabled           bool   `mapstructure:"hsts_enabled"`
	HSTSMaxAge            int    `mapstructure:"hsts_max_age"` // seconds
	HSTSIncludeSubdomains bool   `mapstructure:"hsts_include_subdomains"`
	CSPEnabled            bool   `mapstructure:"csp_enabled"`
	CSPDirective          string `mapstructure:"csp_directive"`
	FrameOptions          string `mapstructure:"frame_options"` // DENY, SAMEORIGIN, ALLOW-FROM
	ContentTypeOptions    bool   `mapstructure:"content_type_options"`
	XSSProtection         bool   `mapstructure:"xss_protection"`
	ReferrerPolicy        string `mapstructure:"referrer_policy"`
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
		APIServer: APIServerConfig{
			Enabled:      true,
			Port:         8081,
			Host:         "0.0.0.0",
			TLSEnabled:   false,
			TLSCertFile:  "",
			TLSKeyFile:   "",
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
			Auth: AuthConfig{
				Enabled:     false,
				HMACSecret:  "",
				JWTSecret:   "",
				APIKeys:     []string{},
				TokenExpiry: 3600,
				AllowedIPs:  []string{},
			},
			RateLimit: RateLimitConfig{
				Enabled:         true,
				RequestsPerMin:  60,
				BurstSize:       10,
				WindowSize:      60,
				CleanupInterval: 300,
			},
			CORS: CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"Content-Type", "Authorization", "X-API-Key", "X-Requested-With"},
				ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
				AllowCredentials: false,
				MaxAge:           86400,
			},
			Security: SecurityConfig{
				HSTSEnabled:           true,
				HSTSMaxAge:            31536000,
				HSTSIncludeSubdomains: true,
				CSPEnabled:            true,
				CSPDirective:          "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'",
				FrameOptions:          "DENY",
				ContentTypeOptions:    true,
				XSSProtection:         true,
				ReferrerPolicy:        "strict-origin-when-cross-origin",
			},
		},
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
	v.SetDefault("api_server.enabled", cfg.APIServer.Enabled)
	v.SetDefault("api_server.port", cfg.APIServer.Port)
	v.SetDefault("api_server.host", cfg.APIServer.Host)
	v.SetDefault("api_server.tls_enabled", cfg.APIServer.TLSEnabled)
	v.SetDefault("api_server.tls_cert_file", cfg.APIServer.TLSCertFile)
	v.SetDefault("api_server.tls_key_file", cfg.APIServer.TLSKeyFile)
	v.SetDefault("api_server.read_timeout", cfg.APIServer.ReadTimeout)
	v.SetDefault("api_server.write_timeout", cfg.APIServer.WriteTimeout)
	v.SetDefault("api_server.idle_timeout", cfg.APIServer.IdleTimeout)

	// Auth defaults
	v.SetDefault("api_server.auth.enabled", cfg.APIServer.Auth.Enabled)
	v.SetDefault("api_server.auth.hmac_secret", cfg.APIServer.Auth.HMACSecret)
	v.SetDefault("api_server.auth.jwt_secret", cfg.APIServer.Auth.JWTSecret)
	v.SetDefault("api_server.auth.api_keys", cfg.APIServer.Auth.APIKeys)
	v.SetDefault("api_server.auth.token_expiry", cfg.APIServer.Auth.TokenExpiry)
	v.SetDefault("api_server.auth.allowed_ips", cfg.APIServer.Auth.AllowedIPs)

	// Rate limit defaults
	v.SetDefault("api_server.rate_limit.enabled", cfg.APIServer.RateLimit.Enabled)
	v.SetDefault("api_server.rate_limit.requests_per_minute", cfg.APIServer.RateLimit.RequestsPerMin)
	v.SetDefault("api_server.rate_limit.burst_size", cfg.APIServer.RateLimit.BurstSize)
	v.SetDefault("api_server.rate_limit.window_size", cfg.APIServer.RateLimit.WindowSize)
	v.SetDefault("api_server.rate_limit.cleanup_interval", cfg.APIServer.RateLimit.CleanupInterval)

	// CORS defaults
	v.SetDefault("api_server.cors.enabled", cfg.APIServer.CORS.Enabled)
	v.SetDefault("api_server.cors.allowed_origins", cfg.APIServer.CORS.AllowedOrigins)
	v.SetDefault("api_server.cors.allowed_methods", cfg.APIServer.CORS.AllowedMethods)
	v.SetDefault("api_server.cors.allowed_headers", cfg.APIServer.CORS.AllowedHeaders)
	v.SetDefault("api_server.cors.exposed_headers", cfg.APIServer.CORS.ExposedHeaders)
	v.SetDefault("api_server.cors.allow_credentials", cfg.APIServer.CORS.AllowCredentials)
	v.SetDefault("api_server.cors.max_age", cfg.APIServer.CORS.MaxAge)

	// Security defaults
	v.SetDefault("api_server.security.hsts_enabled", cfg.APIServer.Security.HSTSEnabled)
	v.SetDefault("api_server.security.hsts_max_age", cfg.APIServer.Security.HSTSMaxAge)
	v.SetDefault("api_server.security.hsts_include_subdomains", cfg.APIServer.Security.HSTSIncludeSubdomains)
	v.SetDefault("api_server.security.csp_enabled", cfg.APIServer.Security.CSPEnabled)
	v.SetDefault("api_server.security.csp_directive", cfg.APIServer.Security.CSPDirective)
	v.SetDefault("api_server.security.frame_options", cfg.APIServer.Security.FrameOptions)
	v.SetDefault("api_server.security.content_type_options", cfg.APIServer.Security.ContentTypeOptions)
	v.SetDefault("api_server.security.xss_protection", cfg.APIServer.Security.XSSProtection)
	v.SetDefault("api_server.security.referrer_policy", cfg.APIServer.Security.ReferrerPolicy)
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

// Save saves the configuration to a YAML file
func (c *Config) Save(configFile string) error {
	if configFile == "" {
		configFile = "config.yaml"
	}

	// Create viper instance for writing
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	// Set all config values in viper
	v.Set("device_id", c.DeviceID)
	v.Set("device_key", c.DeviceKey)
	v.Set("server_url", c.ServerURL)
	v.Set("tier", c.Tier)
	v.Set("queue_max_size", c.QueueMaxSize)
	v.Set("heartbeat_interval", c.HeartbeatInterval)
	v.Set("unlock_duration", c.UnlockDuration)
	v.Set("database_path", c.DatabasePath)
	v.Set("log_level", c.LogLevel)
	v.Set("log_file", c.LogFile)
	v.Set("enabled_adapters", c.EnabledAdapters)
	v.Set("adapter_configs", c.AdapterConfigs)
	v.Set("updates_enabled", c.UpdatesEnabled)
	v.Set("update_manifest_url", c.UpdateManifestURL)
	v.Set("update_public_key", c.UpdatePublicKey)

	// API Server configuration
	v.Set("api_server.enabled", c.APIServer.Enabled)
	v.Set("api_server.port", c.APIServer.Port)
	v.Set("api_server.host", c.APIServer.Host)
	v.Set("api_server.tls_enabled", c.APIServer.TLSEnabled)
	v.Set("api_server.tls_cert_file", c.APIServer.TLSCertFile)
	v.Set("api_server.tls_key_file", c.APIServer.TLSKeyFile)
	v.Set("api_server.read_timeout", c.APIServer.ReadTimeout)
	v.Set("api_server.write_timeout", c.APIServer.WriteTimeout)
	v.Set("api_server.idle_timeout", c.APIServer.IdleTimeout)

	// Auth configuration
	v.Set("api_server.auth.enabled", c.APIServer.Auth.Enabled)
	v.Set("api_server.auth.hmac_secret", c.APIServer.Auth.HMACSecret)
	v.Set("api_server.auth.jwt_secret", c.APIServer.Auth.JWTSecret)
	v.Set("api_server.auth.api_keys", c.APIServer.Auth.APIKeys)
	v.Set("api_server.auth.token_expiry", c.APIServer.Auth.TokenExpiry)
	v.Set("api_server.auth.allowed_ips", c.APIServer.Auth.AllowedIPs)

	// Rate limit configuration
	v.Set("api_server.rate_limit.enabled", c.APIServer.RateLimit.Enabled)
	v.Set("api_server.rate_limit.requests_per_minute", c.APIServer.RateLimit.RequestsPerMin)
	v.Set("api_server.rate_limit.burst_size", c.APIServer.RateLimit.BurstSize)
	v.Set("api_server.rate_limit.window_size", c.APIServer.RateLimit.WindowSize)
	v.Set("api_server.rate_limit.cleanup_interval", c.APIServer.RateLimit.CleanupInterval)

	// CORS configuration
	v.Set("api_server.cors.enabled", c.APIServer.CORS.Enabled)
	v.Set("api_server.cors.allowed_origins", c.APIServer.CORS.AllowedOrigins)
	v.Set("api_server.cors.allowed_methods", c.APIServer.CORS.AllowedMethods)
	v.Set("api_server.cors.allowed_headers", c.APIServer.CORS.AllowedHeaders)
	v.Set("api_server.cors.exposed_headers", c.APIServer.CORS.ExposedHeaders)
	v.Set("api_server.cors.allow_credentials", c.APIServer.CORS.AllowCredentials)
	v.Set("api_server.cors.max_age", c.APIServer.CORS.MaxAge)

	// Security configuration
	v.Set("api_server.security.hsts_enabled", c.APIServer.Security.HSTSEnabled)
	v.Set("api_server.security.hsts_max_age", c.APIServer.Security.HSTSMaxAge)
	v.Set("api_server.security.hsts_include_subdomains", c.APIServer.Security.HSTSIncludeSubdomains)
	v.Set("api_server.security.csp_enabled", c.APIServer.Security.CSPEnabled)
	v.Set("api_server.security.csp_directive", c.APIServer.Security.CSPDirective)
	v.Set("api_server.security.frame_options", c.APIServer.Security.FrameOptions)
	v.Set("api_server.security.content_type_options", c.APIServer.Security.ContentTypeOptions)
	v.Set("api_server.security.xss_protection", c.APIServer.Security.XSSProtection)
	v.Set("api_server.security.referrer_policy", c.APIServer.Security.ReferrerPolicy)

	// Write the configuration file
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
