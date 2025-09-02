package pairing

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"

	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/config"
	"github.com/sirupsen/logrus"
)

// HTTPClient interface for making pairing requests
type HTTPClient interface {
	PairDevice(ctx context.Context, pairCode string, deviceInfo *client.DeviceInfo) (*client.PairResponse, error)
}

// AuthManager interface for managing device credentials
type AuthManager interface {
	IsAuthenticated() bool
	GetDeviceID() string
	SetCredentials(deviceID, deviceKey string) error
	ClearCredentials() error
}

// PairingManager handles device pairing operations
type PairingManager struct {
	httpClient  HTTPClient
	authManager AuthManager
	config      *config.Config
	logger      *logrus.Logger
}

// NewPairingManager creates a new pairing manager
func NewPairingManager(httpClient HTTPClient, authManager AuthManager, cfg *config.Config, logger *logrus.Logger) (*PairingManager, error) {
	if httpClient == nil || reflect.ValueOf(httpClient).IsNil() {
		return nil, fmt.Errorf("http client is required")
	}
	if authManager == nil || reflect.ValueOf(authManager).IsNil() {
		return nil, fmt.Errorf("auth manager is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &PairingManager{
		httpClient:  httpClient,
		authManager: authManager,
		config:      cfg,
		logger:      logger,
	}, nil
}

// PairDevice pairs the device with the cloud using a pair code
func (p *PairingManager) PairDevice(ctx context.Context, pairCode string) (*client.PairResponse, error) {
	if pairCode == "" {
		return nil, fmt.Errorf("pair code is required")
	}

	// Check if device is already paired
	if p.authManager.IsAuthenticated() {
		return nil, fmt.Errorf("device is already paired")
	}

	p.logger.Info("Starting device pairing process", "pair_code", pairCode)

	// Gather device information
	deviceInfo, err := p.gatherDeviceInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to gather device info: %w", err)
	}

	p.logger.Debug("Device info gathered", 
		"hostname", deviceInfo.Hostname,
		"platform", deviceInfo.Platform,
		"version", deviceInfo.Version,
		"tier", deviceInfo.Tier)

	// Make pairing request to cloud
	pairResp, err := p.httpClient.PairDevice(ctx, pairCode, deviceInfo)
	if err != nil {
		p.logger.Error("Pairing request failed", "error", err)
		return nil, fmt.Errorf("pairing request failed: %w", err)
	}

	// Store received credentials securely
	if err := p.authManager.SetCredentials(pairResp.DeviceID, pairResp.DeviceKey); err != nil {
		p.logger.Error("Failed to store credentials", "error", err)
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}

	// Update configuration with received settings
	if pairResp.Config != nil {
		p.updateConfigFromPairResponse(pairResp.Config)
	}

	p.logger.Info("Device paired successfully", 
		"device_id", pairResp.DeviceID,
		"heartbeat_interval", pairResp.Config.HeartbeatInterval,
		"queue_max_size", pairResp.Config.QueueMaxSize,
		"unlock_duration", pairResp.Config.UnlockDuration)

	return pairResp, nil
}

// IsPaired returns true if the device is already paired
func (p *PairingManager) IsPaired() bool {
	return p.authManager.IsAuthenticated()
}

// GetDeviceID returns the current device ID if paired
func (p *PairingManager) GetDeviceID() string {
	return p.authManager.GetDeviceID()
}

// UnpairDevice removes stored credentials and unpairs the device
func (p *PairingManager) UnpairDevice() error {
	if !p.authManager.IsAuthenticated() {
		return fmt.Errorf("device is not paired")
	}

	deviceID := p.authManager.GetDeviceID()
	
	if err := p.authManager.ClearCredentials(); err != nil {
		return fmt.Errorf("failed to clear credentials: %w", err)
	}

	p.logger.Info("Device unpaired successfully", "device_id", deviceID)
	return nil
}

// gatherDeviceInfo collects information about the current device
func (p *PairingManager) gatherDeviceInfo() (*client.DeviceInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		p.logger.Warn("Failed to get hostname, using default", "error", err)
		hostname = "unknown-host"
	}

	// Determine platform
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}

	// Get version from config or use default
	version := "1.0.0" // This could be injected at build time
	
	// Use tier from config
	tier := p.config.Tier
	if tier == "" {
		tier = "normal" // Default tier
	}

	return &client.DeviceInfo{
		Hostname: hostname,
		Platform: platform,
		Version:  version,
		Tier:     tier,
	}, nil
}

// updateConfigFromPairResponse updates the local configuration with settings from the pairing response
func (p *PairingManager) updateConfigFromPairResponse(deviceConfig *client.DeviceConfig) {
	if deviceConfig.HeartbeatInterval > 0 {
		p.config.HeartbeatInterval = deviceConfig.HeartbeatInterval
	}
	if deviceConfig.QueueMaxSize > 0 {
		p.config.QueueMaxSize = deviceConfig.QueueMaxSize
	}
	if deviceConfig.UnlockDuration > 0 {
		p.config.UnlockDuration = deviceConfig.UnlockDuration
	}

	p.logger.Debug("Configuration updated from pairing response",
		"heartbeat_interval", p.config.HeartbeatInterval,
		"queue_max_size", p.config.QueueMaxSize,
		"unlock_duration", p.config.UnlockDuration)
}