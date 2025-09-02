package updater

import (
	"fmt"
	"path/filepath"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/sirupsen/logrus"
)

// Factory creates updater instances
type Factory struct {
	logger *logrus.Logger
}

// NewFactory creates a new updater factory
func NewFactory(logger *logrus.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateUpdater creates a new updater instance from configuration
func (f *Factory) CreateUpdater(cfg *config.Config) (*Updater, error) {
	// Check if updates are enabled
	if !cfg.UpdatesEnabled {
		return nil, fmt.Errorf("updates are disabled in configuration")
	}
	
	// Determine manifest URL
	manifestURL := cfg.UpdateManifestURL
	if manifestURL == "" {
		manifestURL = cfg.ServerURL + "/api/v1/updates/manifest.json"
	}
	
	// Determine public key
	publicKey := cfg.UpdatePublicKey
	if publicKey == "" {
		publicKey = getPublicKey() // Fallback to embedded key
	}
	
	// Create updater configuration
	updaterConfig := &UpdaterConfig{
		ManifestURL:    manifestURL,
		PublicKey:      publicKey,
		CheckInterval:  24 * time.Hour, // Check for updates daily
		DeviceID:       cfg.DeviceID,
		CurrentVersion: getCurrentVersion(),
		UpdateDir:      getUpdateDir(),
		BackupDir:      getBackupDir(),
	}
	
	// Validate configuration
	if err := f.validateUpdaterConfig(updaterConfig); err != nil {
		return nil, fmt.Errorf("invalid updater configuration: %w", err)
	}
	
	return NewUpdater(updaterConfig, f.logger), nil
}

// CreateHealthChecker creates a health checker instance
func (f *Factory) CreateHealthChecker(cfg *config.Config) (*HealthChecker, error) {
	updaterConfig := &UpdaterConfig{
		UpdateDir: getUpdateDir(),
		BackupDir: getBackupDir(),
	}
	
	healthURL := "http://localhost:8080/health" // Default health endpoint
	
	return NewHealthChecker(updaterConfig, f.logger, healthURL), nil
}

// CreateQueuePreserver creates a queue preserver instance
func (f *Factory) CreateQueuePreserver(cfg *config.Config) (*QueuePreserver, error) {
	return NewQueuePreserver(f.logger, cfg.DatabasePath, getBackupDir()), nil
}

// validateUpdaterConfig validates the updater configuration
func (f *Factory) validateUpdaterConfig(config *UpdaterConfig) error {
	if config.ManifestURL == "" {
		return fmt.Errorf("manifest URL is required")
	}
	
	if config.PublicKey == "" {
		return fmt.Errorf("public key is required")
	}
	
	if config.CheckInterval <= 0 {
		return fmt.Errorf("check interval must be positive")
	}
	
	if config.UpdateDir == "" {
		return fmt.Errorf("update directory is required")
	}
	
	if config.BackupDir == "" {
		return fmt.Errorf("backup directory is required")
	}
	
	return nil
}

// getPublicKey returns the Ed25519 public key for signature verification
// In production, this should be embedded in the binary or loaded from secure config
func getPublicKey() string {
	// This is a placeholder - in production, use a real public key
	return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}

// getCurrentVersion returns the current version of the bridge
// This should be set during build time using ldflags
var Version = "dev" // Set by build process

func getCurrentVersion() string {
	return Version
}

// getUpdateDir returns the directory for storing updates
func getUpdateDir() string {
	return filepath.Join(".", "updates")
}

// getBackupDir returns the directory for storing backups
func getBackupDir() string {
	return filepath.Join(".", "backups")
}