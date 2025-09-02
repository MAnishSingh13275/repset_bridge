package pairing

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
	"github.com/sirupsen/logrus"
)

// mockHTTPClient implements the HTTPClient interface for testing
type mockHTTPClient struct {
	pairDeviceFunc func(ctx context.Context, pairCode string, deviceInfo *client.DeviceInfo) (*client.PairResponse, error)
}

func (m *mockHTTPClient) PairDevice(ctx context.Context, pairCode string, deviceInfo *client.DeviceInfo) (*client.PairResponse, error) {
	if m.pairDeviceFunc != nil {
		return m.pairDeviceFunc(ctx, pairCode, deviceInfo)
	}
	return nil, fmt.Errorf("not implemented")
}

// mockAuthManager implements the AuthManager interface for testing
type mockAuthManager struct {
	deviceID        string
	deviceKey       string
	authenticated   bool
	setCredentialsFunc func(deviceID, deviceKey string) error
	clearCredentialsFunc func() error
}

func newMockAuthManager() *mockAuthManager {
	return &mockAuthManager{}
}

func (m *mockAuthManager) IsAuthenticated() bool {
	return m.authenticated
}

func (m *mockAuthManager) GetDeviceID() string {
	return m.deviceID
}

func (m *mockAuthManager) SetCredentials(deviceID, deviceKey string) error {
	if m.setCredentialsFunc != nil {
		return m.setCredentialsFunc(deviceID, deviceKey)
	}
	m.deviceID = deviceID
	m.deviceKey = deviceKey
	m.authenticated = true
	return nil
}

func (m *mockAuthManager) ClearCredentials() error {
	if m.clearCredentialsFunc != nil {
		return m.clearCredentialsFunc()
	}
	m.deviceID = ""
	m.deviceKey = ""
	m.authenticated = false
	return nil
}

func TestNewPairingManager(t *testing.T) {
	logger := logging.Initialize("debug")
	httpClient := &mockHTTPClient{}
	authManager := newMockAuthManager()
	cfg := &config.Config{
		Tier: "normal",
	}

	tests := []struct {
		name        string
		httpClient  *mockHTTPClient
		authManager *mockAuthManager
		config      *config.Config
		logger      *logrus.Logger
		wantErr     bool
	}{
		{
			name:        "valid parameters",
			httpClient:  httpClient,
			authManager: authManager,
			config:      cfg,
			logger:      logger,
			wantErr:     false,
		},
		{
			name:        "nil http client",
			httpClient:  (*mockHTTPClient)(nil),
			authManager: authManager,
			config:      cfg,
			logger:      logger,
			wantErr:     true,
		},
		{
			name:        "nil auth manager",
			httpClient:  httpClient,
			authManager: (*mockAuthManager)(nil),
			config:      cfg,
			logger:      logger,
			wantErr:     true,
		},
		{
			name:        "nil config",
			httpClient:  httpClient,
			authManager: authManager,
			config:      nil,
			logger:      logger,
			wantErr:     true,
		},
		{
			name:        "nil logger",
			httpClient:  httpClient,
			authManager: authManager,
			config:      cfg,
			logger:      nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := NewPairingManager(tt.httpClient, tt.authManager, tt.config, tt.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPairingManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pm == nil {
				t.Error("NewPairingManager() returned nil manager")
			}
		})
	}
}

func TestPairingManager_PairDevice(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier: "normal",
	}

	tests := []struct {
		name            string
		pairCode        string
		alreadyPaired   bool
		httpClientFunc  func(ctx context.Context, pairCode string, deviceInfo *client.DeviceInfo) (*client.PairResponse, error)
		setCredsFunc    func(deviceID, deviceKey string) error
		wantErr         bool
		wantDeviceID    string
	}{
		{
			name:          "successful pairing",
			pairCode:      "ABC123",
			alreadyPaired: false,
			httpClientFunc: func(ctx context.Context, pairCode string, deviceInfo *client.DeviceInfo) (*client.PairResponse, error) {
				if pairCode != "ABC123" {
					t.Errorf("Expected pair code ABC123, got %s", pairCode)
				}
				if deviceInfo == nil {
					t.Error("Expected device info, got nil")
				}
				if deviceInfo.Platform == "" {
					t.Error("Expected platform to be set")
				}
				if deviceInfo.Hostname == "" {
					t.Error("Expected hostname to be set")
				}
				if deviceInfo.Version == "" {
					t.Error("Expected version to be set")
				}
				if deviceInfo.Tier != "normal" {
					t.Errorf("Expected tier normal, got %s", deviceInfo.Tier)
				}

				return &client.PairResponse{
					DeviceID:  "dev_abc123",
					DeviceKey: "secret_key",
					Config: &client.DeviceConfig{
						HeartbeatInterval: 60,
						QueueMaxSize:      10000,
						UnlockDuration:    3000,
					},
				}, nil
			},
			wantErr:      false,
			wantDeviceID: "dev_abc123",
		},
		{
			name:          "empty pair code",
			pairCode:      "",
			alreadyPaired: false,
			wantErr:       true,
		},
		{
			name:          "already paired",
			pairCode:      "ABC123",
			alreadyPaired: true,
			wantErr:       true,
		},
		{
			name:          "http client error",
			pairCode:      "ABC123",
			alreadyPaired: false,
			httpClientFunc: func(ctx context.Context, pairCode string, deviceInfo *client.DeviceInfo) (*client.PairResponse, error) {
				return nil, fmt.Errorf("network error")
			},
			wantErr: true,
		},
		{
			name:          "credential storage error",
			pairCode:      "ABC123",
			alreadyPaired: false,
			httpClientFunc: func(ctx context.Context, pairCode string, deviceInfo *client.DeviceInfo) (*client.PairResponse, error) {
				return &client.PairResponse{
					DeviceID:  "dev_abc123",
					DeviceKey: "secret_key",
					Config: &client.DeviceConfig{
						HeartbeatInterval: 60,
						QueueMaxSize:      10000,
						UnlockDuration:    3000,
					},
				}, nil
			},
			setCredsFunc: func(deviceID, deviceKey string) error {
				return fmt.Errorf("storage error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &mockHTTPClient{
				pairDeviceFunc: tt.httpClientFunc,
			}
			authManager := newMockAuthManager()
			authManager.authenticated = tt.alreadyPaired
			if tt.setCredsFunc != nil {
				authManager.setCredentialsFunc = tt.setCredsFunc
			}

			pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
			if err != nil {
				t.Fatalf("Failed to create pairing manager: %v", err)
			}

			ctx := context.Background()
			resp, err := pm.PairDevice(ctx, tt.pairCode)

			if (err != nil) != tt.wantErr {
				t.Errorf("PairDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Error("Expected response, got nil")
					return
				}
				if resp.DeviceID != tt.wantDeviceID {
					t.Errorf("Expected device ID %s, got %s", tt.wantDeviceID, resp.DeviceID)
				}
				if authManager.deviceID != tt.wantDeviceID {
					t.Errorf("Expected auth manager device ID %s, got %s", tt.wantDeviceID, authManager.deviceID)
				}
			}
		})
	}
}

func TestPairingManager_IsPaired(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier: "normal",
	}

	tests := []struct {
		name          string
		authenticated bool
		want          bool
	}{
		{
			name:          "paired device",
			authenticated: true,
			want:          true,
		},
		{
			name:          "unpaired device",
			authenticated: false,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &mockHTTPClient{}
			authManager := newMockAuthManager()
			authManager.authenticated = tt.authenticated

			pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
			if err != nil {
				t.Fatalf("Failed to create pairing manager: %v", err)
			}

			result := pm.IsPaired()
			if result != tt.want {
				t.Errorf("IsPaired() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestPairingManager_GetDeviceID(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier: "normal",
	}

	tests := []struct {
		name     string
		deviceID string
		want     string
	}{
		{
			name:     "with device ID",
			deviceID: "dev_123",
			want:     "dev_123",
		},
		{
			name:     "empty device ID",
			deviceID: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &mockHTTPClient{}
			authManager := newMockAuthManager()
			authManager.deviceID = tt.deviceID

			pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
			if err != nil {
				t.Fatalf("Failed to create pairing manager: %v", err)
			}

			result := pm.GetDeviceID()
			if result != tt.want {
				t.Errorf("GetDeviceID() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestPairingManager_UnpairDevice(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier: "normal",
	}

	tests := []struct {
		name            string
		authenticated   bool
		clearCredsFunc  func() error
		wantErr         bool
	}{
		{
			name:          "successful unpair",
			authenticated: true,
			wantErr:       false,
		},
		{
			name:          "not paired",
			authenticated: false,
			wantErr:       true,
		},
		{
			name:          "clear credentials error",
			authenticated: true,
			clearCredsFunc: func() error {
				return fmt.Errorf("clear error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &mockHTTPClient{}
			authManager := newMockAuthManager()
			authManager.authenticated = tt.authenticated
			authManager.deviceID = "dev_123"
			if tt.clearCredsFunc != nil {
				authManager.clearCredentialsFunc = tt.clearCredsFunc
			}

			pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
			if err != nil {
				t.Fatalf("Failed to create pairing manager: %v", err)
			}

			err = pm.UnpairDevice()

			if (err != nil) != tt.wantErr {
				t.Errorf("UnpairDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if authManager.authenticated {
					t.Error("Expected device to be unauthenticated after unpair")
				}
				if authManager.deviceID != "" {
					t.Error("Expected device ID to be cleared after unpair")
				}
			}
		})
	}
}

func TestPairingManager_gatherDeviceInfo(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier: "full",
	}
	httpClient := &mockHTTPClient{}
	authManager := newMockAuthManager()

	pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	deviceInfo, err := pm.gatherDeviceInfo()
	if err != nil {
		t.Errorf("gatherDeviceInfo() error = %v", err)
		return
	}

	if deviceInfo == nil {
		t.Error("Expected device info, got nil")
		return
	}

	// Verify hostname is set (should be actual hostname or "unknown-host")
	if deviceInfo.Hostname == "" {
		t.Error("Expected hostname to be set")
	}

	// Verify platform is set correctly
	expectedPlatform := runtime.GOOS
	if expectedPlatform == "darwin" {
		expectedPlatform = "macos"
	}
	if deviceInfo.Platform != expectedPlatform {
		t.Errorf("Expected platform %s, got %s", expectedPlatform, deviceInfo.Platform)
	}

	// Verify version is set
	if deviceInfo.Version == "" {
		t.Error("Expected version to be set")
	}

	// Verify tier matches config
	if deviceInfo.Tier != cfg.Tier {
		t.Errorf("Expected tier %s, got %s", cfg.Tier, deviceInfo.Tier)
	}
}

func TestPairingManager_updateConfigFromPairResponse(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier:              "normal",
		HeartbeatInterval: 30,
		QueueMaxSize:      5000,
		UnlockDuration:    2000,
	}
	httpClient := &mockHTTPClient{}
	authManager := newMockAuthManager()

	pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	deviceConfig := &client.DeviceConfig{
		HeartbeatInterval: 120,
		QueueMaxSize:      20000,
		UnlockDuration:    5000,
	}

	pm.updateConfigFromPairResponse(deviceConfig)

	if cfg.HeartbeatInterval != 120 {
		t.Errorf("Expected heartbeat interval 120, got %d", cfg.HeartbeatInterval)
	}
	if cfg.QueueMaxSize != 20000 {
		t.Errorf("Expected queue max size 20000, got %d", cfg.QueueMaxSize)
	}
	if cfg.UnlockDuration != 5000 {
		t.Errorf("Expected unlock duration 5000, got %d", cfg.UnlockDuration)
	}
}

func TestPairingManager_updateConfigFromPairResponse_ZeroValues(t *testing.T) {
	logger := logging.Initialize("debug")
	originalHeartbeat := 60
	originalQueueSize := 10000
	originalUnlockDuration := 3000
	
	cfg := &config.Config{
		Tier:              "normal",
		HeartbeatInterval: originalHeartbeat,
		QueueMaxSize:      originalQueueSize,
		UnlockDuration:    originalUnlockDuration,
	}
	httpClient := &mockHTTPClient{}
	authManager := newMockAuthManager()

	pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	// Test with zero values - should not update config
	deviceConfig := &client.DeviceConfig{
		HeartbeatInterval: 0,
		QueueMaxSize:      0,
		UnlockDuration:    0,
	}

	pm.updateConfigFromPairResponse(deviceConfig)

	// Values should remain unchanged
	if cfg.HeartbeatInterval != originalHeartbeat {
		t.Errorf("Expected heartbeat interval %d, got %d", originalHeartbeat, cfg.HeartbeatInterval)
	}
	if cfg.QueueMaxSize != originalQueueSize {
		t.Errorf("Expected queue max size %d, got %d", originalQueueSize, cfg.QueueMaxSize)
	}
	if cfg.UnlockDuration != originalUnlockDuration {
		t.Errorf("Expected unlock duration %d, got %d", originalUnlockDuration, cfg.UnlockDuration)
	}
}

func TestPairingManager_gatherDeviceInfo_HostnameHandling(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier: "normal",
	}
	httpClient := &mockHTTPClient{}
	authManager := newMockAuthManager()

	pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	deviceInfo, err := pm.gatherDeviceInfo()
	if err != nil {
		t.Errorf("gatherDeviceInfo() error = %v", err)
		return
	}

	// Hostname should be set to something (either actual hostname or "unknown-host")
	if deviceInfo.Hostname == "" {
		t.Error("Expected hostname to be set")
	}
}

func TestPairingManager_gatherDeviceInfo_EmptyTier(t *testing.T) {
	logger := logging.Initialize("debug")
	cfg := &config.Config{
		Tier: "", // Empty tier
	}
	httpClient := &mockHTTPClient{}
	authManager := newMockAuthManager()

	pm, err := NewPairingManager(httpClient, authManager, cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	deviceInfo, err := pm.gatherDeviceInfo()
	if err != nil {
		t.Errorf("gatherDeviceInfo() error = %v", err)
		return
	}

	if deviceInfo.Tier != "normal" {
		t.Errorf("Expected default tier 'normal', got %s", deviceInfo.Tier)
	}
}