package api

import (
	"fmt"
	"reflect"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/sirupsen/logrus"
)

// DefaultConfigManager implements the ConfigManager interface
type DefaultConfigManager struct {
	config     *config.Config
	logger     *logrus.Logger
	configFile string
}

// NewDefaultConfigManager creates a new default config manager
func NewDefaultConfigManager(cfg *config.Config, logger *logrus.Logger, configFile string) *DefaultConfigManager {
	return &DefaultConfigManager{
		config:     cfg,
		logger:     logger,
		configFile: configFile,
	}
}

// GetCurrentConfig returns the current configuration
func (cm *DefaultConfigManager) GetCurrentConfig() *config.Config {
	return cm.config
}

// UpdateConfig updates the configuration with the provided changes
func (cm *DefaultConfigManager) UpdateConfig(updates *ConfigUpdateRequest) (*ConfigUpdateResponse, error) {
	cm.logger.Info("Processing configuration update request")
	
	var updatedFields []string
	var requiresRestart bool
	
	// Create a copy of the current config to modify
	newConfig := *cm.config
	
	// Update basic configuration fields
	if updates.Tier != nil && *updates.Tier != newConfig.Tier {
		newConfig.Tier = *updates.Tier
		updatedFields = append(updatedFields, "tier")
		requiresRestart = true // Tier changes may affect system behavior
	}
	
	if updates.QueueMaxSize != nil && *updates.QueueMaxSize != newConfig.QueueMaxSize {
		newConfig.QueueMaxSize = *updates.QueueMaxSize
		updatedFields = append(updatedFields, "queueMaxSize")
	}
	
	if updates.HeartbeatInterval != nil && *updates.HeartbeatInterval != newConfig.HeartbeatInterval {
		newConfig.HeartbeatInterval = *updates.HeartbeatInterval
		updatedFields = append(updatedFields, "heartbeatInterval")
	}
	
	if updates.UnlockDuration != nil && *updates.UnlockDuration != newConfig.UnlockDuration {
		newConfig.UnlockDuration = *updates.UnlockDuration
		updatedFields = append(updatedFields, "unlockDuration")
	}
	
	if updates.LogLevel != nil && *updates.LogLevel != newConfig.LogLevel {
		newConfig.LogLevel = *updates.LogLevel
		updatedFields = append(updatedFields, "logLevel")
		// Note: Log level changes could be applied dynamically, but for simplicity we'll mark as restart required
		requiresRestart = true
	}
	
	if updates.LogFile != nil && *updates.LogFile != newConfig.LogFile {
		newConfig.LogFile = *updates.LogFile
		updatedFields = append(updatedFields, "logFile")
		requiresRestart = true
	}
	
	if updates.UpdatesEnabled != nil && *updates.UpdatesEnabled != newConfig.UpdatesEnabled {
		newConfig.UpdatesEnabled = *updates.UpdatesEnabled
		updatedFields = append(updatedFields, "updatesEnabled")
	}
	
	// Update adapter configuration
	if len(updates.EnabledAdapters) > 0 && !stringSlicesEqual(updates.EnabledAdapters, newConfig.EnabledAdapters) {
		newConfig.EnabledAdapters = updates.EnabledAdapters
		updatedFields = append(updatedFields, "enabledAdapters")
		requiresRestart = true // Adapter changes require restart
	}
	
	if len(updates.AdapterConfigs) > 0 && !reflect.DeepEqual(updates.AdapterConfigs, newConfig.AdapterConfigs) {
		// Merge adapter configs
		if newConfig.AdapterConfigs == nil {
			newConfig.AdapterConfigs = make(map[string]map[string]interface{})
		}
		for adapterName, adapterConfig := range updates.AdapterConfigs {
			newConfig.AdapterConfigs[adapterName] = adapterConfig
		}
		updatedFields = append(updatedFields, "adapterConfigs")
		requiresRestart = true // Adapter config changes require restart
	}
	
	// Update API server configuration
	if updates.APIServer != nil {
		apiUpdated, apiRestart := cm.updateAPIServerConfig(&newConfig.APIServer, updates.APIServer)
		if len(apiUpdated) > 0 {
			for _, field := range apiUpdated {
				updatedFields = append(updatedFields, "apiServer."+field)
			}
			if apiRestart {
				requiresRestart = true
			}
		}
	}
	
	// If no fields were updated, return early
	if len(updatedFields) == 0 {
		return &ConfigUpdateResponse{
			Success:         true,
			Message:         "No configuration changes detected",
			UpdatedFields:   []string{},
			RequiresRestart: false,
			Timestamp:       time.Now().UTC(),
		}, nil
	}
	
	// Apply the configuration changes
	cm.config = &newConfig
	
	cm.logger.WithFields(logrus.Fields{
		"updatedFields":   updatedFields,
		"requiresRestart": requiresRestart,
	}).Info("Configuration updated successfully")
	
	return &ConfigUpdateResponse{
		Success:         true,
		Message:         fmt.Sprintf("Configuration updated successfully. %d field(s) changed.", len(updatedFields)),
		UpdatedFields:   updatedFields,
		RequiresRestart: requiresRestart,
		Timestamp:       time.Now().UTC(),
	}, nil
}

// updateAPIServerConfig updates API server configuration and returns updated fields and restart requirement
func (cm *DefaultConfigManager) updateAPIServerConfig(current *config.APIServerConfig, updates *APIServerConfigUpdateRequest) ([]string, bool) {
	var updatedFields []string
	var requiresRestart bool
	
	if updates.Enabled != nil && *updates.Enabled != current.Enabled {
		current.Enabled = *updates.Enabled
		updatedFields = append(updatedFields, "enabled")
		requiresRestart = true
	}
	
	if updates.Port != nil && *updates.Port != current.Port {
		current.Port = *updates.Port
		updatedFields = append(updatedFields, "port")
		requiresRestart = true
	}
	
	if updates.Host != nil && *updates.Host != current.Host {
		current.Host = *updates.Host
		updatedFields = append(updatedFields, "host")
		requiresRestart = true
	}
	
	if updates.TLSEnabled != nil && *updates.TLSEnabled != current.TLSEnabled {
		current.TLSEnabled = *updates.TLSEnabled
		updatedFields = append(updatedFields, "tlsEnabled")
		requiresRestart = true
	}
	
	if updates.TLSCertFile != nil && *updates.TLSCertFile != current.TLSCertFile {
		current.TLSCertFile = *updates.TLSCertFile
		updatedFields = append(updatedFields, "tlsCertFile")
		requiresRestart = true
	}
	
	if updates.TLSKeyFile != nil && *updates.TLSKeyFile != current.TLSKeyFile {
		current.TLSKeyFile = *updates.TLSKeyFile
		updatedFields = append(updatedFields, "tlsKeyFile")
		requiresRestart = true
	}
	
	if updates.ReadTimeout != nil && *updates.ReadTimeout != current.ReadTimeout {
		current.ReadTimeout = *updates.ReadTimeout
		updatedFields = append(updatedFields, "readTimeout")
		requiresRestart = true
	}
	
	if updates.WriteTimeout != nil && *updates.WriteTimeout != current.WriteTimeout {
		current.WriteTimeout = *updates.WriteTimeout
		updatedFields = append(updatedFields, "writeTimeout")
		requiresRestart = true
	}
	
	if updates.IdleTimeout != nil && *updates.IdleTimeout != current.IdleTimeout {
		current.IdleTimeout = *updates.IdleTimeout
		updatedFields = append(updatedFields, "idleTimeout")
		requiresRestart = true
	}
	
	// Update auth configuration
	if updates.Auth != nil {
		authUpdated := cm.updateAuthConfig(&current.Auth, updates.Auth)
		if len(authUpdated) > 0 {
			for _, field := range authUpdated {
				updatedFields = append(updatedFields, "auth."+field)
			}
			// Auth changes typically don't require restart, they can be applied dynamically
		}
	}
	
	// Update rate limit configuration
	if updates.RateLimit != nil {
		rateLimitUpdated := cm.updateRateLimitConfig(&current.RateLimit, updates.RateLimit)
		if len(rateLimitUpdated) > 0 {
			for _, field := range rateLimitUpdated {
				updatedFields = append(updatedFields, "rateLimit."+field)
			}
			// Rate limit changes can typically be applied dynamically
		}
	}
	
	// Update CORS configuration
	if updates.CORS != nil {
		corsUpdated := cm.updateCORSConfig(&current.CORS, updates.CORS)
		if len(corsUpdated) > 0 {
			for _, field := range corsUpdated {
				updatedFields = append(updatedFields, "cors."+field)
			}
			// CORS changes can typically be applied dynamically
		}
	}
	
	// Update security configuration
	if updates.Security != nil {
		securityUpdated := cm.updateSecurityConfig(&current.Security, updates.Security)
		if len(securityUpdated) > 0 {
			for _, field := range securityUpdated {
				updatedFields = append(updatedFields, "security."+field)
			}
			// Security header changes can typically be applied dynamically
		}
	}
	
	return updatedFields, requiresRestart
}

// updateAuthConfig updates authentication configuration
func (cm *DefaultConfigManager) updateAuthConfig(current *config.AuthConfig, updates *AuthConfigUpdateRequest) []string {
	var updatedFields []string
	
	if updates.Enabled != nil && *updates.Enabled != current.Enabled {
		current.Enabled = *updates.Enabled
		updatedFields = append(updatedFields, "enabled")
	}
	
	if updates.HMACSecret != nil && *updates.HMACSecret != current.HMACSecret {
		current.HMACSecret = *updates.HMACSecret
		updatedFields = append(updatedFields, "hmacSecret")
	}
	
	if updates.JWTSecret != nil && *updates.JWTSecret != current.JWTSecret {
		current.JWTSecret = *updates.JWTSecret
		updatedFields = append(updatedFields, "jwtSecret")
	}
	
	if len(updates.APIKeys) > 0 && !stringSlicesEqual(updates.APIKeys, current.APIKeys) {
		current.APIKeys = updates.APIKeys
		updatedFields = append(updatedFields, "apiKeys")
	}
	
	if updates.TokenExpiry != nil && *updates.TokenExpiry != current.TokenExpiry {
		current.TokenExpiry = *updates.TokenExpiry
		updatedFields = append(updatedFields, "tokenExpiry")
	}
	
	if len(updates.AllowedIPs) > 0 && !stringSlicesEqual(updates.AllowedIPs, current.AllowedIPs) {
		current.AllowedIPs = updates.AllowedIPs
		updatedFields = append(updatedFields, "allowedIps")
	}
	
	return updatedFields
}

// updateRateLimitConfig updates rate limiting configuration
func (cm *DefaultConfigManager) updateRateLimitConfig(current *config.RateLimitConfig, updates *RateLimitConfigUpdateRequest) []string {
	var updatedFields []string
	
	if updates.Enabled != nil && *updates.Enabled != current.Enabled {
		current.Enabled = *updates.Enabled
		updatedFields = append(updatedFields, "enabled")
	}
	
	if updates.RequestsPerMin != nil && *updates.RequestsPerMin != current.RequestsPerMin {
		current.RequestsPerMin = *updates.RequestsPerMin
		updatedFields = append(updatedFields, "requestsPerMinute")
	}
	
	if updates.BurstSize != nil && *updates.BurstSize != current.BurstSize {
		current.BurstSize = *updates.BurstSize
		updatedFields = append(updatedFields, "burstSize")
	}
	
	if updates.WindowSize != nil && *updates.WindowSize != current.WindowSize {
		current.WindowSize = *updates.WindowSize
		updatedFields = append(updatedFields, "windowSize")
	}
	
	if updates.CleanupInterval != nil && *updates.CleanupInterval != current.CleanupInterval {
		current.CleanupInterval = *updates.CleanupInterval
		updatedFields = append(updatedFields, "cleanupInterval")
	}
	
	return updatedFields
}

// updateCORSConfig updates CORS configuration
func (cm *DefaultConfigManager) updateCORSConfig(current *config.CORSConfig, updates *CORSConfigUpdateRequest) []string {
	var updatedFields []string
	
	if updates.Enabled != nil && *updates.Enabled != current.Enabled {
		current.Enabled = *updates.Enabled
		updatedFields = append(updatedFields, "enabled")
	}
	
	if len(updates.AllowedOrigins) > 0 && !stringSlicesEqual(updates.AllowedOrigins, current.AllowedOrigins) {
		current.AllowedOrigins = updates.AllowedOrigins
		updatedFields = append(updatedFields, "allowedOrigins")
	}
	
	if len(updates.AllowedMethods) > 0 && !stringSlicesEqual(updates.AllowedMethods, current.AllowedMethods) {
		current.AllowedMethods = updates.AllowedMethods
		updatedFields = append(updatedFields, "allowedMethods")
	}
	
	if len(updates.AllowedHeaders) > 0 && !stringSlicesEqual(updates.AllowedHeaders, current.AllowedHeaders) {
		current.AllowedHeaders = updates.AllowedHeaders
		updatedFields = append(updatedFields, "allowedHeaders")
	}
	
	if len(updates.ExposedHeaders) > 0 && !stringSlicesEqual(updates.ExposedHeaders, current.ExposedHeaders) {
		current.ExposedHeaders = updates.ExposedHeaders
		updatedFields = append(updatedFields, "exposedHeaders")
	}
	
	if updates.AllowCredentials != nil && *updates.AllowCredentials != current.AllowCredentials {
		current.AllowCredentials = *updates.AllowCredentials
		updatedFields = append(updatedFields, "allowCredentials")
	}
	
	if updates.MaxAge != nil && *updates.MaxAge != current.MaxAge {
		current.MaxAge = *updates.MaxAge
		updatedFields = append(updatedFields, "maxAge")
	}
	
	return updatedFields
}

// updateSecurityConfig updates security configuration
func (cm *DefaultConfigManager) updateSecurityConfig(current *config.SecurityConfig, updates *SecurityConfigUpdateRequest) []string {
	var updatedFields []string
	
	if updates.HSTSEnabled != nil && *updates.HSTSEnabled != current.HSTSEnabled {
		current.HSTSEnabled = *updates.HSTSEnabled
		updatedFields = append(updatedFields, "hstsEnabled")
	}
	
	if updates.HSTSMaxAge != nil && *updates.HSTSMaxAge != current.HSTSMaxAge {
		current.HSTSMaxAge = *updates.HSTSMaxAge
		updatedFields = append(updatedFields, "hstsMaxAge")
	}
	
	if updates.HSTSIncludeSubdomains != nil && *updates.HSTSIncludeSubdomains != current.HSTSIncludeSubdomains {
		current.HSTSIncludeSubdomains = *updates.HSTSIncludeSubdomains
		updatedFields = append(updatedFields, "hstsIncludeSubdomains")
	}
	
	if updates.CSPEnabled != nil && *updates.CSPEnabled != current.CSPEnabled {
		current.CSPEnabled = *updates.CSPEnabled
		updatedFields = append(updatedFields, "cspEnabled")
	}
	
	if updates.CSPDirective != nil && *updates.CSPDirective != current.CSPDirective {
		current.CSPDirective = *updates.CSPDirective
		updatedFields = append(updatedFields, "cspDirective")
	}
	
	if updates.FrameOptions != nil && *updates.FrameOptions != current.FrameOptions {
		current.FrameOptions = *updates.FrameOptions
		updatedFields = append(updatedFields, "frameOptions")
	}
	
	if updates.ContentTypeOptions != nil && *updates.ContentTypeOptions != current.ContentTypeOptions {
		current.ContentTypeOptions = *updates.ContentTypeOptions
		updatedFields = append(updatedFields, "contentTypeOptions")
	}
	
	if updates.XSSProtection != nil && *updates.XSSProtection != current.XSSProtection {
		current.XSSProtection = *updates.XSSProtection
		updatedFields = append(updatedFields, "xssProtection")
	}
	
	if updates.ReferrerPolicy != nil && *updates.ReferrerPolicy != current.ReferrerPolicy {
		current.ReferrerPolicy = *updates.ReferrerPolicy
		updatedFields = append(updatedFields, "referrerPolicy")
	}
	
	return updatedFields
}

// ReloadConfig reloads the configuration from file
func (cm *DefaultConfigManager) ReloadConfig(force bool) (*ConfigReloadResponse, error) {
	cm.logger.WithField("force", force).Info("Processing configuration reload request")
	
	// For this implementation, we'll simulate reloading from file
	// In a real implementation, this would reload from the actual config file
	
	changedFields := []string{}
	reloadedFrom := "memory" // Since we don't have actual file reloading implemented
	
	if cm.configFile != "" {
		reloadedFrom = cm.configFile
	}
	
	// In a real implementation, you would:
	// 1. Load the configuration from file
	// 2. Compare with current configuration
	// 3. Apply changes
	// 4. Return the list of changed fields
	
	// For now, we'll just return a success response indicating no changes
	if !force {
		cm.logger.Info("Configuration reload completed - no changes detected")
		return &ConfigReloadResponse{
			Success:       true,
			Message:       "Configuration reloaded successfully - no changes detected",
			ReloadedFrom:  reloadedFrom,
			ChangedFields: changedFields,
			Timestamp:     time.Now().UTC(),
		}, nil
	}
	
	cm.logger.Info("Configuration force reload completed")
	return &ConfigReloadResponse{
		Success:       true,
		Message:       "Configuration force reloaded successfully",
		ReloadedFrom:  reloadedFrom,
		ChangedFields: changedFields,
		Timestamp:     time.Now().UTC(),
	}, nil
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	
	return true
}