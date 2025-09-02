package updater

import (
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory_CreateUpdater(t *testing.T) {
	logger := logrus.New()
	factory := NewFactory(logger)
	
	cfg := &config.Config{
		ServerURL:      "https://api.example.com",
		DeviceID:       "test_device_123",
		UpdatesEnabled: true,
	}
	
	updater, err := factory.CreateUpdater(cfg)
	require.NoError(t, err)
	assert.NotNil(t, updater)
	
	// Verify configuration
	assert.Equal(t, "https://api.example.com/api/v1/updates/manifest.json", updater.config.ManifestURL)
	assert.Equal(t, "test_device_123", updater.config.DeviceID)
	assert.Equal(t, 24*time.Hour, updater.config.CheckInterval)
	assert.NotEmpty(t, updater.config.PublicKey)
	assert.NotEmpty(t, updater.config.UpdateDir)
	assert.NotEmpty(t, updater.config.BackupDir)
}

func TestFactory_CreateHealthChecker(t *testing.T) {
	logger := logrus.New()
	factory := NewFactory(logger)
	
	cfg := &config.Config{}
	
	healthChecker, err := factory.CreateHealthChecker(cfg)
	require.NoError(t, err)
	assert.NotNil(t, healthChecker)
}

func TestFactory_CreateQueuePreserver(t *testing.T) {
	logger := logrus.New()
	factory := NewFactory(logger)
	
	cfg := &config.Config{
		DatabasePath: "/path/to/database.db",
	}
	
	queuePreserver, err := factory.CreateQueuePreserver(cfg)
	require.NoError(t, err)
	assert.NotNil(t, queuePreserver)
}

func TestFactory_ValidateUpdaterConfig(t *testing.T) {
	logger := logrus.New()
	factory := NewFactory(logger)
	
	tests := []struct {
		name        string
		config      *UpdaterConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &UpdaterConfig{
				ManifestURL:   "https://example.com/manifest.json",
				PublicKey:     "test_key",
				CheckInterval: time.Hour,
				UpdateDir:     "/tmp/updates",
				BackupDir:     "/tmp/backups",
			},
			expectError: false,
		},
		{
			name: "missing manifest URL",
			config: &UpdaterConfig{
				PublicKey:     "test_key",
				CheckInterval: time.Hour,
				UpdateDir:     "/tmp/updates",
				BackupDir:     "/tmp/backups",
			},
			expectError: true,
		},
		{
			name: "missing public key",
			config: &UpdaterConfig{
				ManifestURL:   "https://example.com/manifest.json",
				CheckInterval: time.Hour,
				UpdateDir:     "/tmp/updates",
				BackupDir:     "/tmp/backups",
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.validateUpdaterConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}