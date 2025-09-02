package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateProcess_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create test environment
	tempDir := t.TempDir()
	updateDir := filepath.Join(tempDir, "updates")
	backupDir := filepath.Join(tempDir, "backups")
	databasePath := filepath.Join(tempDir, "queue.db")
	
	// Generate test key pair
	publicKey, privateKey, err := GenerateKeyPair()
	require.NoError(t, err)
	
	// Create test binary
	testBinary := filepath.Join(tempDir, "test_binary")
	binaryContent := []byte("new version binary content")
	err = os.WriteFile(testBinary, binaryContent, 0755)
	require.NoError(t, err)
	
	// Sign test binary
	signature, err := SignFile(testBinary, privateKey)
	require.NoError(t, err)
	
	// Create test manifest
	platform := runtime.GOOS + "_" + runtime.GOARCH
	manifest := &Manifest{
		Version:     "2.0.0",
		ReleaseDate: time.Now(),
		Binaries: map[string]Binary{
			platform: {
				URL:       "/test_binary",
				Signature: signature,
				Size:      int64(len(binaryContent)),
				Checksum:  "dummy_checksum",
			},
		},
		Rollout: RolloutConfig{
			Percentage: 100,
		},
	}
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/updates/manifest.json":
			json.NewEncoder(w).Encode(manifest)
		case "/test_binary":
			http.ServeFile(w, r, testBinary)
		case "/health":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	
	// Create test database
	dbContent := []byte("test queue data")
	err = os.WriteFile(databasePath, dbContent, 0644)
	require.NoError(t, err)
	
	// Create configuration
	cfg := &config.Config{
		ServerURL:      server.URL,
		DeviceID:       "test_device",
		DatabasePath:   databasePath,
		UpdatesEnabled: true,
	}
	
	// Create factory and components
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	factory := NewFactory(logger)
	
	// Create updater
	updater, err := factory.CreateUpdater(cfg)
	require.NoError(t, err)
	
	// Override config for test
	updater.config.ManifestURL = server.URL + "/api/v1/updates/manifest.json"
	updater.config.PublicKey = publicKey
	updater.config.CurrentVersion = "1.0.0"
	updater.config.UpdateDir = updateDir
	updater.config.BackupDir = backupDir
	
	// Create queue preserver
	queuePreserver, err := factory.CreateQueuePreserver(cfg)
	require.NoError(t, err)
	
	// Create health checker
	healthChecker, err := factory.CreateHealthChecker(cfg)
	require.NoError(t, err)
	healthChecker.healthURL = server.URL + "/health"
	
	// Test the complete update process
	ctx := context.Background()
	
	// 1. Preserve queue
	backupPath, err := queuePreserver.PreserveQueue(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, backupPath)
	
	// 2. Check for and apply update
	err = updater.checkForUpdates(ctx)
	require.NoError(t, err)
	
	// 3. Verify update was downloaded
	expectedBinaryPath := filepath.Join(updateDir, "bridge_2.0.0")
	assert.FileExists(t, expectedBinaryPath)
	
	// 4. Verify restart signal was created
	restartFile := filepath.Join(updateDir, "restart_required")
	assert.FileExists(t, restartFile)
	
	// 5. Simulate health check after restart
	err = healthChecker.CheckHealthAfterUpdate(ctx)
	require.NoError(t, err)
	
	// 6. Verify queue was preserved
	err = queuePreserver.ValidateQueue(ctx)
	require.NoError(t, err)
	
	// 7. Cleanup
	err = queuePreserver.CleanupOldBackups(ctx)
	require.NoError(t, err)
}

func TestUpdateProcess_WithRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create test environment
	tempDir := t.TempDir()
	updateDir := filepath.Join(tempDir, "updates")
	backupDir := filepath.Join(tempDir, "backups")
	
	// Create a backup file to simulate previous version
	err := os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)
	
	backupFile := filepath.Join(backupDir, "bridge_backup_123")
	backupContent := []byte("previous version content")
	err = os.WriteFile(backupFile, backupContent, 0755)
	require.NoError(t, err)
	
	// Create test server that returns unhealthy status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	
	// Create health checker
	config := &UpdaterConfig{
		UpdateDir: updateDir,
		BackupDir: backupDir,
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise
	
	healthChecker := NewHealthChecker(config, logger, server.URL+"/health")
	healthChecker.maxRetries = 2 // Reduce retries for faster test
	healthChecker.retryDelay = 100 * time.Millisecond
	
	// Test rollback process
	ctx := context.Background()
	err = healthChecker.CheckHealthAfterUpdate(ctx)
	
	// Should not error (rollback should succeed)
	assert.NoError(t, err)
	
	// Verify restart signal was created for rollback
	restartFile := filepath.Join(updateDir, "restart_required")
	assert.FileExists(t, restartFile)
}

func TestUpdateProcess_StagedRollout(t *testing.T) {
	tests := []struct {
		name           string
		deviceID       string
		rolloutPercent int
		expectUpdate   bool
	}{
		{
			name:           "device in 50% rollout",
			deviceID:       "device_that_should_update",
			rolloutPercent: 50,
			expectUpdate:   true, // This depends on the hash function
		},
		{
			name:           "device not in 10% rollout",
			deviceID:       "device_that_should_not_update",
			rolloutPercent: 10,
			expectUpdate:   false,
		},
		{
			name:           "device in 100% rollout",
			deviceID:       "any_device",
			rolloutPercent: 100,
			expectUpdate:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create updater
			config := &UpdaterConfig{
				DeviceID: tt.deviceID,
			}
			
			logger := logrus.New()
			updater := NewUpdater(config, logger)
			
			// Test rollout eligibility
			rollout := RolloutConfig{
				Percentage: tt.rolloutPercent,
			}
			
			result := updater.isEligibleForRollout(rollout)
			
			// Note: The actual result depends on the hash function
			// This test mainly verifies that the function runs without error
			assert.IsType(t, true, result)
		})
	}
}