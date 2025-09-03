package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConfigManager implements the ConfigManager interface for testing
type MockConfigManager struct {
	mock.Mock
}

func (m *MockConfigManager) GetCurrentConfig() *config.Config {
	args := m.Called()
	return args.Get(0).(*config.Config)
}

func (m *MockConfigManager) UpdateConfig(updates *ConfigUpdateRequest) (*ConfigUpdateResponse, error) {
	args := m.Called(updates)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ConfigUpdateResponse), nil
}

func (m *MockConfigManager) ReloadConfig(force bool) (*ConfigReloadResponse, error) {
	args := m.Called(force)
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ConfigReloadResponse), nil
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockConfigManager)
		expectedStatus int
		expectedFields []string
	}{
		{
			name: "successful config retrieval",
			setupMocks: func(mockConfigManager *MockConfigManager) {
				testConfig := &config.Config{
					DeviceID:          "test-device-123",
					ServerURL:         "https://api.example.com",
					Tier:              "normal",
					QueueMaxSize:      1000,
					HeartbeatInterval: 60,
					UnlockDuration:    3000,
					DatabasePath:      "./test.db",
					LogLevel:          "info",
					LogFile:           "",
					EnabledAdapters:   []string{"simulator", "gpio"},
					AdapterConfigs:    map[string]map[string]interface{}{
						"simulator": {"enabled": true},
					},
					UpdatesEnabled: true,
					APIServer: config.APIServerConfig{
						Enabled:      true,
						Port:         8081,
						Host:         "0.0.0.0",
						TLSEnabled:   false,
						ReadTimeout:  30,
						WriteTimeout: 30,
						IdleTimeout:  120,
						Auth: config.AuthConfig{
							Enabled:     true,
							HMACSecret:  "secret-key",
							JWTSecret:   "jwt-secret",
							APIKeys:     []string{"key1", "key2"},
							TokenExpiry: 3600,
							AllowedIPs:  []string{"192.168.1.0/24"},
						},
						RateLimit: config.RateLimitConfig{
							Enabled:         true,
							RequestsPerMin:  60,
							BurstSize:       10,
							WindowSize:      60,
							CleanupInterval: 300,
						},
						CORS: config.CORSConfig{
							Enabled:          true,
							AllowedOrigins:   []string{"*"},
							AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
							AllowedHeaders:   []string{"Content-Type", "Authorization"},
							ExposedHeaders:   []string{"X-RateLimit-Limit"},
							AllowCredentials: false,
							MaxAge:           86400,
						},
						Security: config.SecurityConfig{
							HSTSEnabled:           true,
							HSTSMaxAge:            31536000,
							HSTSIncludeSubdomains: true,
							CSPEnabled:            true,
							CSPDirective:          "default-src 'self'",
							FrameOptions:          "DENY",
							ContentTypeOptions:    true,
							XSSProtection:         true,
							ReferrerPolicy:        "strict-origin-when-cross-origin",
						},
					},
				}
				mockConfigManager.On("GetCurrentConfig").Return(testConfig)
			},
			expectedStatus: http.StatusOK,
			expectedFields: []string{"deviceId", "serverUrl", "tier", "apiServer"},
		},
		{
			name: "config manager not available",
			setupMocks: func(mockConfigManager *MockConfigManager) {
				// Don't set up any mocks - this will test the fallback to h.config
			},
			expectedStatus: http.StatusOK,
			expectedFields: []string{"deviceId", "serverUrl", "tier", "apiServer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config
			testConfig := &config.Config{
				DeviceID:          "fallback-device",
				ServerURL:         "https://fallback.example.com",
				Tier:              "lite",
				QueueMaxSize:      500,
				HeartbeatInterval: 30,
				UnlockDuration:    2000,
				DatabasePath:      "./fallback.db",
				LogLevel:          "debug",
				EnabledAdapters:   []string{"simulator"},
				AdapterConfigs:    map[string]map[string]interface{}{},
				UpdatesEnabled:    false,
				APIServer:         config.DefaultConfig().APIServer,
			}

			// Create mocks
			mockConfigManager := &MockConfigManager{}
			if tt.name != "config manager not available" {
				tt.setupMocks(mockConfigManager)
			}

			// Create handlers
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests
			
			var handlers *Handlers
			if tt.name == "config manager not available" {
				handlers = NewHandlers(testConfig, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")
			} else {
				handlers = NewHandlers(testConfig, logger, nil, nil, nil, nil, nil, mockConfigManager, "1.0.0", "test-device")
			}

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/config", nil)
			w := httptest.NewRecorder()

			// Execute request
			handlers.GetConfig(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if w.Code == http.StatusOK {
				var response ConfigResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// Check that expected fields are present
				for _, field := range tt.expectedFields {
					switch field {
					case "deviceId":
						assert.NotEmpty(t, response.DeviceID)
					case "serverUrl":
						assert.NotEmpty(t, response.ServerURL)
					case "tier":
						assert.NotEmpty(t, response.Tier)
					case "apiServer":
						assert.NotNil(t, response.APIServer)
					}
				}

				// Check that sensitive fields are masked properly
				// HasHMACKey should be true when there's a secret, false when empty
				if tt.name == "successful config retrieval" {
					assert.True(t, response.APIServer.Auth.HasHMACKey) // Should be true since we set a secret
				}
				assert.True(t, response.APIServer.Auth.APIKeyCount >= 0)
			}

			// Verify mocks
			if tt.name != "config manager not available" {
				mockConfigManager.AssertExpectations(t)
			}
		})
	}
}

func TestUpdateConfig(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockConfigManager)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful config update",
			requestBody: ConfigUpdateRequest{
				Tier:              stringPtr("full"),
				QueueMaxSize:      intPtr(2000),
				HeartbeatInterval: intPtr(120),
				UnlockDuration:    intPtr(5000),
				LogLevel:          stringPtr("debug"),
				EnabledAdapters:   []string{"gpio", "simulator"},
				UpdatesEnabled:    boolPtr(false),
			},
			setupMocks: func(mockConfigManager *MockConfigManager) {
				mockConfigManager.On("UpdateConfig", mock.AnythingOfType("*api.ConfigUpdateRequest")).Return(
					&ConfigUpdateResponse{
						Success:         true,
						Message:         "Configuration updated successfully",
						UpdatedFields:   []string{"tier", "queueMaxSize", "heartbeatInterval", "unlockDuration", "logLevel", "enabledAdapters", "updatesEnabled"},
						RequiresRestart: true,
						Timestamp:       time.Now().UTC(),
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "config update with API server changes",
			requestBody: ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					Port:        intPtr(8082),
					TLSEnabled:  boolPtr(true),
					TLSCertFile: stringPtr("/path/to/cert.pem"),
					TLSKeyFile:  stringPtr("/path/to/key.pem"),
					Auth: &AuthConfigUpdateRequest{
						Enabled:     boolPtr(true),
						TokenExpiry: intPtr(7200),
						APIKeys:     []string{"new-key-1", "new-key-2"},
					},
				},
			},
			setupMocks: func(mockConfigManager *MockConfigManager) {
				mockConfigManager.On("UpdateConfig", mock.AnythingOfType("*api.ConfigUpdateRequest")).Return(
					&ConfigUpdateResponse{
						Success:         true,
						Message:         "Configuration updated successfully",
						UpdatedFields:   []string{"apiServer.port", "apiServer.tlsEnabled", "apiServer.tlsCertFile", "apiServer.tlsKeyFile", "apiServer.auth.enabled", "apiServer.auth.tokenExpiry", "apiServer.auth.apiKeys"},
						RequiresRestart: true,
						Timestamp:       time.Now().UTC(),
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			setupMocks:     func(mockConfigManager *MockConfigManager) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_JSON",
		},
		{
			name: "validation error - invalid tier",
			requestBody: ConfigUpdateRequest{
				Tier: stringPtr("invalid-tier"),
			},
			setupMocks:     func(mockConfigManager *MockConfigManager) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name: "validation error - negative queue size",
			requestBody: ConfigUpdateRequest{
				QueueMaxSize: intPtr(-100),
			},
			setupMocks:     func(mockConfigManager *MockConfigManager) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name: "validation error - invalid log level",
			requestBody: ConfigUpdateRequest{
				LogLevel: stringPtr("invalid-level"),
			},
			setupMocks:     func(mockConfigManager *MockConfigManager) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name: "validation error - invalid port",
			requestBody: ConfigUpdateRequest{
				APIServer: &APIServerConfigUpdateRequest{
					Port: intPtr(70000), // Invalid port number
				},
			},
			setupMocks:     func(mockConfigManager *MockConfigManager) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name: "config manager error",
			requestBody: ConfigUpdateRequest{
				Tier: stringPtr("full"),
			},
			setupMocks: func(mockConfigManager *MockConfigManager) {
				mockConfigManager.On("UpdateConfig", mock.AnythingOfType("*api.ConfigUpdateRequest")).Return(
					(*ConfigUpdateResponse)(nil), fmt.Errorf("config update failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "CONFIG_UPDATE_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockConfigManager := &MockConfigManager{}
			tt.setupMocks(mockConfigManager)

			// Create handlers
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests
			handlers := NewHandlers(&config.Config{}, logger, nil, nil, nil, nil, nil, mockConfigManager, "1.0.0", "test-device")

			// Create request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			// Create request
			req := httptest.NewRequest("PUT", "/api/v1/config", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute request
			handlers.UpdateConfig(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResponse.Code)
			} else if w.Code == http.StatusOK {
				var response ConfigUpdateResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotEmpty(t, response.Message)
				assert.NotEmpty(t, response.RequestID)
			}

			// Verify mocks
			mockConfigManager.AssertExpectations(t)
		})
	}
}

func TestUpdateConfigWithoutConfigManager(t *testing.T) {
	// Create handlers without config manager
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	handlers := NewHandlers(&config.Config{}, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")

	// Create request
	requestBody := ConfigUpdateRequest{
		Tier: stringPtr("full"),
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/v1/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	handlers.UpdateConfig(w, req)

	// Check response
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var errorResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "CONFIG_MANAGER_UNAVAILABLE", errorResponse.Code)
}

func TestReloadConfig(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockConfigManager)
		expectedStatus int
		expectedError  string
	}{
		{
			name:        "successful config reload without body",
			requestBody: nil,
			setupMocks: func(mockConfigManager *MockConfigManager) {
				mockConfigManager.On("ReloadConfig", false).Return(
					&ConfigReloadResponse{
						Success:       true,
						Message:       "Configuration reloaded successfully",
						ReloadedFrom:  "/etc/bridge/config.yaml",
						ChangedFields: []string{"logLevel", "heartbeatInterval"},
						Timestamp:     time.Now().UTC(),
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful force config reload",
			requestBody: ConfigReloadRequest{
				Force:  true,
				Reason: "Manual reload requested",
			},
			setupMocks: func(mockConfigManager *MockConfigManager) {
				mockConfigManager.On("ReloadConfig", true).Return(
					&ConfigReloadResponse{
						Success:       true,
						Message:       "Configuration force reloaded successfully",
						ReloadedFrom:  "/etc/bridge/config.yaml",
						ChangedFields: []string{},
						Timestamp:     time.Now().UTC(),
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			setupMocks:     func(mockConfigManager *MockConfigManager) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_JSON",
		},
		{
			name: "config manager error",
			requestBody: ConfigReloadRequest{
				Force: false,
			},
			setupMocks: func(mockConfigManager *MockConfigManager) {
				mockConfigManager.On("ReloadConfig", false).Return(
					(*ConfigReloadResponse)(nil), fmt.Errorf("config reload failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "CONFIG_RELOAD_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockConfigManager := &MockConfigManager{}
			tt.setupMocks(mockConfigManager)

			// Create handlers
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests
			handlers := NewHandlers(&config.Config{}, logger, nil, nil, nil, nil, nil, mockConfigManager, "1.0.0", "test-device")

			// Create request
			var req *http.Request
			if tt.requestBody == nil {
				req = httptest.NewRequest("POST", "/api/v1/config/reload", nil)
			} else {
				var body []byte
				var err error
				if str, ok := tt.requestBody.(string); ok {
					body = []byte(str)
				} else {
					body, err = json.Marshal(tt.requestBody)
					assert.NoError(t, err)
				}
				req = httptest.NewRequest("POST", "/api/v1/config/reload", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()

			// Execute request
			handlers.ReloadConfig(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResponse.Code)
			} else if w.Code == http.StatusOK {
				var response ConfigReloadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotEmpty(t, response.Message)
				assert.NotEmpty(t, response.RequestID)
			}

			// Verify mocks
			mockConfigManager.AssertExpectations(t)
		})
	}
}

func TestReloadConfigWithoutConfigManager(t *testing.T) {
	// Create handlers without config manager
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	handlers := NewHandlers(&config.Config{}, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/config/reload", nil)
	w := httptest.NewRecorder()

	// Execute request
	handlers.ReloadConfig(w, req)

	// Check response
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var errorResponse ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "CONFIG_MANAGER_UNAVAILABLE", errorResponse.Code)
}

// Helper functions for creating pointers to primitive types
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}