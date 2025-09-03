package api

import (
	"testing"

	"gym-door-bridge/internal/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigManager_GetCurrentConfig(t *testing.T) {
	// Create test config
	testConfig := &config.Config{
		DeviceID:          "test-device",
		ServerURL:         "https://api.example.com",
		Tier:              "normal",
		QueueMaxSize:      1000,
		HeartbeatInterval: 60,
		UnlockDuration:    3000,
	}

	// Create config manager
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	cm := NewDefaultConfigManager(testConfig, logger, "test-config.yaml")

	// Test getting current config
	currentConfig := cm.GetCurrentConfig()
	assert.Equal(t, testConfig, currentConfig)
	assert.Equal(t, "test-device", currentConfig.DeviceID)
	assert.Equal(t, "normal", currentConfig.Tier)
}

func TestDefaultConfigManager_UpdateConfig(t *testing.T) {
	tests := []struct {
		name                string
		initialConfig       *config.Config
		updateRequest       *ConfigUpdateRequest
		expectedUpdates     []string
		expectedRestart     bool
		expectError         bool
	}{
		{
			name: "update basic configuration fields",
			initialConfig: &config.Config{
				Tier:              "normal",
				QueueMaxSize:      1000,
				HeartbeatInterval: 60,
				UnlockDuration:    3000,
				LogLevel:          "info",
				UpdatesEnabled:    true,
			},
			updateRequest: &ConfigUpdateRequest{
				Tier:              stringPtr("full"),
				QueueMaxSize:      intPtr(2000),
				HeartbeatInterval: intPtr(120),
				UnlockDuration:    intPtr(5000),
				LogLevel:          stringPtr("debug"),
				UpdatesEnabled:    boolPtr(false),
			},
			expectedUpdates: []string{"tier", "queueMaxSize", "heartbeatInterval", "unlockDuration", "logLevel", "updatesEnabled"},
			expectedRestart: true,
		},
		{
			name: "update adapter configuration",
			initialConfig: &config.Config{
				EnabledAdapters: []string{"simulator"},
				AdapterConfigs:  map[string]map[string]interface{}{
					"simulator": {"enabled": true},
				},
			},
			updateRequest: &ConfigUpdateRequest{
				EnabledAdapters: []string{"gpio", "simulator"},
				AdapterConfigs: map[string]map[string]interface{}{
					"gpio": {"pin": 18, "enabled": true},
					"simulator": {"enabled": false},
				},
			},
			expectedUpdates: []string{"enabledAdapters", "adapterConfigs"},
			expectedRestart: true,
		},
		{
			name: "update API server configuration",
			initialConfig: &config.Config{
				APIServer: config.APIServerConfig{
					Enabled:      true,
					Port:         8081,
					Host:         "0.0.0.0",
					TLSEnabled:   false,
					ReadTimeout:  30,
					WriteTimeout: 30,
					IdleTimeout:  120,
					Auth: config.AuthConfig{
						Enabled:     false,
						TokenExpiry: 3600,
					},
					RateLimit: config.RateLimitConfig{
						Enabled:        true,
						RequestsPerMin: 60,
						BurstSize:      10,
					},
				},
			},
			updateRequest: &ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					Port:       intPtr(8082),
					TLSEnabled: boolPtr(true),
					Auth: &AuthConfigUpdateRequest{
						Enabled:     boolPtr(true),
						TokenExpiry: intPtr(7200),
					},
					RateLimit: &RateLimitConfigUpdateRequest{
						RequestsPerMin: intPtr(120),
						BurstSize:      intPtr(20),
					},
				},
			},
			expectedUpdates: []string{"apiServer.port", "apiServer.tlsEnabled", "apiServer.auth.enabled", "apiServer.auth.tokenExpiry", "apiServer.rateLimit.requestsPerMinute", "apiServer.rateLimit.burstSize"},
			expectedRestart: true,
		},
		{
			name: "no changes - same values",
			initialConfig: &config.Config{
				Tier:         "normal",
				QueueMaxSize: 1000,
				LogLevel:     "info",
			},
			updateRequest: &ConfigUpdateRequest{
				Tier:         stringPtr("normal"),
				QueueMaxSize: intPtr(1000),
				LogLevel:     stringPtr("info"),
			},
			expectedUpdates: []string{},
			expectedRestart: false,
		},
		{
			name: "update CORS configuration",
			initialConfig: &config.Config{
				APIServer: config.APIServerConfig{
					CORS: config.CORSConfig{
						Enabled:        true,
						AllowedOrigins: []string{"*"},
						AllowedMethods: []string{"GET", "POST"},
						MaxAge:         86400,
					},
				},
			},
			updateRequest: &ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					CORS: &CORSConfigUpdateRequest{
						AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
						AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
						MaxAge:         intPtr(3600),
					},
				},
			},
			expectedUpdates: []string{"apiServer.cors.allowedOrigins", "apiServer.cors.allowedMethods", "apiServer.cors.maxAge"},
			expectedRestart: false, // CORS changes don't require restart
		},
		{
			name: "update security configuration",
			initialConfig: &config.Config{
				APIServer: config.APIServerConfig{
					Security: config.SecurityConfig{
						HSTSEnabled:    true,
						HSTSMaxAge:     31536000,
						CSPEnabled:     true,
						CSPDirective:   "default-src 'self'",
						FrameOptions:   "DENY",
						XSSProtection:  true,
						ReferrerPolicy: "strict-origin-when-cross-origin",
					},
				},
			},
			updateRequest: &ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					Security: &SecurityConfigUpdateRequest{
						HSTSMaxAge:     intPtr(63072000), // 2 years
						CSPDirective:   stringPtr("default-src 'self'; script-src 'self' 'unsafe-inline'"),
						FrameOptions:   stringPtr("SAMEORIGIN"),
						ReferrerPolicy: stringPtr("no-referrer"),
					},
				},
			},
			expectedUpdates: []string{"apiServer.security.hstsMaxAge", "apiServer.security.cspDirective", "apiServer.security.frameOptions", "apiServer.security.referrerPolicy"},
			expectedRestart: false, // Security header changes don't require restart
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config manager
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			cm := NewDefaultConfigManager(tt.initialConfig, logger, "test-config.yaml")

			// Perform update
			response, err := cm.UpdateConfig(tt.updateRequest)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			// Check successful response
			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.True(t, response.Success)
			assert.Equal(t, tt.expectedRestart, response.RequiresRestart)

			// Check updated fields
			assert.ElementsMatch(t, tt.expectedUpdates, response.UpdatedFields)

			// Verify configuration was actually updated
			updatedConfig := cm.GetCurrentConfig()
			if tt.updateRequest.Tier != nil {
				assert.Equal(t, *tt.updateRequest.Tier, updatedConfig.Tier)
			}
			if tt.updateRequest.QueueMaxSize != nil {
				assert.Equal(t, *tt.updateRequest.QueueMaxSize, updatedConfig.QueueMaxSize)
			}
			if tt.updateRequest.LogLevel != nil {
				assert.Equal(t, *tt.updateRequest.LogLevel, updatedConfig.LogLevel)
			}
		})
	}
}

func TestDefaultConfigManager_ReloadConfig(t *testing.T) {
	tests := []struct {
		name            string
		force           bool
		expectedSuccess bool
		expectedMessage string
	}{
		{
			name:            "normal reload",
			force:           false,
			expectedSuccess: true,
			expectedMessage: "Configuration reloaded successfully - no changes detected",
		},
		{
			name:            "force reload",
			force:           true,
			expectedSuccess: true,
			expectedMessage: "Configuration force reloaded successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config
			testConfig := &config.Config{
				DeviceID: "test-device",
				Tier:     "normal",
			}

			// Create config manager
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			cm := NewDefaultConfigManager(testConfig, logger, "/etc/bridge/config.yaml")

			// Perform reload
			response, err := cm.ReloadConfig(tt.force)

			// Check response
			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedSuccess, response.Success)
			assert.Contains(t, response.Message, tt.expectedMessage)
			assert.Equal(t, "/etc/bridge/config.yaml", response.ReloadedFrom)
			assert.NotNil(t, response.ChangedFields) // Should be an empty slice, not nil
		})
	}
}

func TestStringSlicesEqual(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []string
		slice2   []string
		expected bool
	}{
		{
			name:     "equal slices",
			slice1:   []string{"a", "b", "c"},
			slice2:   []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "different lengths",
			slice1:   []string{"a", "b"},
			slice2:   []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "different order",
			slice1:   []string{"a", "b", "c"},
			slice2:   []string{"c", "b", "a"},
			expected: false,
		},
		{
			name:     "different values",
			slice1:   []string{"a", "b", "c"},
			slice2:   []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "empty slices",
			slice1:   []string{},
			slice2:   []string{},
			expected: true,
		},
		{
			name:     "nil vs empty",
			slice1:   nil,
			slice2:   []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringSlicesEqual(tt.slice1, tt.slice2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigUpdateValidation(t *testing.T) {
	tests := []struct {
		name        string
		request     *ConfigUpdateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &ConfigUpdateRequest{
				Tier:              stringPtr("full"),
				QueueMaxSize:      intPtr(2000),
				HeartbeatInterval: intPtr(120),
				UnlockDuration:    intPtr(5000),
				LogLevel:          stringPtr("debug"),
			},
			expectError: false,
		},
		{
			name: "invalid tier",
			request: &ConfigUpdateRequest{
				Tier: stringPtr("invalid"),
			},
			expectError: true,
			errorMsg:    "tier must be one of: lite, normal, full",
		},
		{
			name: "negative queue size",
			request: &ConfigUpdateRequest{
				QueueMaxSize: intPtr(-100),
			},
			expectError: true,
			errorMsg:    "queueMaxSize must be positive",
		},
		{
			name: "negative heartbeat interval",
			request: &ConfigUpdateRequest{
				HeartbeatInterval: intPtr(-60),
			},
			expectError: true,
			errorMsg:    "heartbeatInterval must be positive",
		},
		{
			name: "negative unlock duration",
			request: &ConfigUpdateRequest{
				UnlockDuration: intPtr(-1000),
			},
			expectError: true,
			errorMsg:    "unlockDuration must be positive",
		},
		{
			name: "invalid log level",
			request: &ConfigUpdateRequest{
				LogLevel: stringPtr("invalid"),
			},
			expectError: true,
			errorMsg:    "logLevel must be one of: debug, info, warn, error",
		},
		{
			name: "invalid API server port",
			request: &ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					Port: intPtr(70000),
				},
			},
			expectError: true,
			errorMsg:    "apiServer.port must be between 1 and 65535",
		},
		{
			name: "negative API server timeout",
			request: &ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					ReadTimeout: intPtr(-30),
				},
			},
			expectError: true,
			errorMsg:    "apiServer.readTimeout must be positive",
		},
		{
			name: "invalid frame options",
			request: &ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					Security: &SecurityConfigUpdateRequest{
						FrameOptions: stringPtr("INVALID"),
					},
				},
			},
			expectError: true,
			errorMsg:    "apiServer.security.frameOptions must be one of: DENY, SAMEORIGIN, ALLOW-FROM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}