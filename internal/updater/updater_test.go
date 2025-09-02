package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdater_CheckForUpdates(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		manifestVersion string
		rolloutPercent int
		expectUpdate   bool
	}{
		{
			name:           "no update needed - same version",
			currentVersion: "1.0.0",
			manifestVersion: "1.0.0",
			rolloutPercent: 100,
			expectUpdate:   false,
		},
		{
			name:           "update available - different version",
			currentVersion: "1.0.0",
			manifestVersion: "1.1.0",
			rolloutPercent: 100,
			expectUpdate:   true,
		},
		{
			name:           "update available but not in rollout",
			currentVersion: "1.0.0",
			manifestVersion: "1.1.0",
			rolloutPercent: 0,
			expectUpdate:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directories
			tempDir := t.TempDir()
			updateDir := filepath.Join(tempDir, "updates")
			backupDir := filepath.Join(tempDir, "backups")
			
			// Generate test key pair
			publicKey, privateKey, err := GenerateKeyPair()
			require.NoError(t, err)
			
			// Create test binary
			testBinary := filepath.Join(tempDir, "test_binary")
			err = os.WriteFile(testBinary, []byte("test binary content"), 0755)
			require.NoError(t, err)
			
			// Sign test binary
			signature, err := SignFile(testBinary, privateKey)
			require.NoError(t, err)
			
			// Create test manifest
			platform := runtime.GOOS + "_" + runtime.GOARCH
			manifest := &Manifest{
				Version:     tt.manifestVersion,
				ReleaseDate: time.Now(),
				Binaries: map[string]Binary{
					platform: {
						URL:       "/test_binary", // Will be updated after server creation
						Signature: signature,
						Size:      int64(len("test binary content")),
						Checksum:  "dummy_checksum",
					},
				},
				Rollout: RolloutConfig{
					Percentage: tt.rolloutPercent,
				},
			}
			
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/manifest.json":
					json.NewEncoder(w).Encode(manifest)
				case "/test_binary":
					http.ServeFile(w, r, testBinary)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			
			// Update manifest with server URL
			binary := manifest.Binaries[platform]
			binary.URL = server.URL + "/test_binary"
			manifest.Binaries[platform] = binary
			
			// Create updater config
			config := &UpdaterConfig{
				ManifestURL:    server.URL + "/manifest.json",
				PublicKey:      publicKey,
				CheckInterval:  time.Hour,
				DeviceID:       "test_device",
				CurrentVersion: tt.currentVersion,
				UpdateDir:      updateDir,
				BackupDir:      backupDir,
			}
			
			// Create updater
			logger := logrus.New()
			logger.SetLevel(logrus.DebugLevel)
			updater := NewUpdater(config, logger)
			
			// Create directories first
			err = updater.createDirectories()
			require.NoError(t, err)
			
			// Check for updates
			ctx := context.Background()
			err = updater.checkForUpdates(ctx)
			
			if tt.expectUpdate {
				assert.NoError(t, err)
				// Check if binary was downloaded (note: on Windows it gets .exe extension)
				expectedPath := filepath.Join(updateDir, fmt.Sprintf("bridge_%s", tt.manifestVersion))
				if runtime.GOOS == "windows" {
					expectedPath += ".exe"
				}
				// The file might have been moved during the update process, so just check that no error occurred
				// In a real scenario, we'd check the restart signal file instead
				restartFile := filepath.Join(updateDir, "restart_required")
				assert.FileExists(t, restartFile)
			} else {
				// Should not error, but no update should be applied
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdater_VerifySignature(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := GenerateKeyPair()
	require.NoError(t, err)
	
	// Create test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_file")
	testContent := []byte("test file content for signature verification")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)
	
	// Sign the file
	signature, err := SignFile(testFile, privateKey)
	require.NoError(t, err)
	
	// Create updater config
	config := &UpdaterConfig{
		PublicKey: publicKey,
	}
	
	logger := logrus.New()
	updater := NewUpdater(config, logger)
	
	// Test valid signature
	err = updater.verifySignature(testFile, signature)
	assert.NoError(t, err)
	
	// Test invalid signature
	err = updater.verifySignature(testFile, "invalid_signature")
	assert.Error(t, err)
	
	// Test with wrong public key
	wrongPublicKey, _, err := GenerateKeyPair()
	require.NoError(t, err)
	
	config.PublicKey = wrongPublicKey
	updater = NewUpdater(config, logger)
	
	err = updater.verifySignature(testFile, signature)
	assert.Error(t, err)
}

func TestUpdater_RolloutEligibility(t *testing.T) {
	logger := logrus.New()
	config := &UpdaterConfig{
		DeviceID: "test_device_123",
	}
	updater := NewUpdater(config, logger)
	
	tests := []struct {
		name     string
		rollout  RolloutConfig
		expected bool
	}{
		{
			name: "100% rollout",
			rollout: RolloutConfig{
				Percentage: 100,
			},
			expected: true,
		},
		{
			name: "0% rollout",
			rollout: RolloutConfig{
				Percentage: 0,
			},
			expected: false,
		},
		{
			name: "explicit device inclusion",
			rollout: RolloutConfig{
				Percentage: 0,
				DeviceIDs:  []string{"test_device_123", "other_device"},
			},
			expected: true,
		},
		{
			name: "device not in explicit list",
			rollout: RolloutConfig{
				Percentage: 0,
				DeviceIDs:  []string{"other_device"},
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.isEligibleForRollout(tt.rollout)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdater_HashDeviceID(t *testing.T) {
	logger := logrus.New()
	config := &UpdaterConfig{}
	updater := NewUpdater(config, logger)
	
	// Test that same device ID produces same hash
	deviceID := "test_device_123"
	hash1 := updater.hashDeviceID(deviceID)
	hash2 := updater.hashDeviceID(deviceID)
	assert.Equal(t, hash1, hash2)
	
	// Test that different device IDs produce different hashes
	hash3 := updater.hashDeviceID("different_device")
	assert.NotEqual(t, hash1, hash3)
}

func TestUpdater_CreateDirectories(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &UpdaterConfig{
		UpdateDir: filepath.Join(tempDir, "updates"),
		BackupDir: filepath.Join(tempDir, "backups"),
	}
	
	logger := logrus.New()
	updater := NewUpdater(config, logger)
	
	err := updater.createDirectories()
	assert.NoError(t, err)
	
	// Check that directories were created
	assert.DirExists(t, config.UpdateDir)
	assert.DirExists(t, config.BackupDir)
}

func TestGenerateKeyPair(t *testing.T) {
	publicKey, privateKey, err := GenerateKeyPair()
	require.NoError(t, err)
	
	assert.NotEmpty(t, publicKey)
	assert.NotEmpty(t, privateKey)
	assert.NotEqual(t, publicKey, privateKey)
	
	// Keys should be hex encoded
	assert.Len(t, publicKey, 64)   // 32 bytes * 2 hex chars
	assert.Len(t, privateKey, 128) // 64 bytes * 2 hex chars
}

func TestSignFile(t *testing.T) {
	// Generate test key pair
	_, privateKey, err := GenerateKeyPair()
	require.NoError(t, err)
	
	// Create test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_file")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	// Sign file
	signature, err := SignFile(testFile, privateKey)
	require.NoError(t, err)
	
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 128) // 64 bytes * 2 hex chars
}

func TestUpdater_IsUpdateNeeded(t *testing.T) {
	logger := logrus.New()
	config := &UpdaterConfig{
		CurrentVersion: "1.0.0",
	}
	updater := NewUpdater(config, logger)
	
	tests := []struct {
		name            string
		manifestVersion string
		expected        bool
	}{
		{
			name:            "same version",
			manifestVersion: "1.0.0",
			expected:        false,
		},
		{
			name:            "different version",
			manifestVersion: "1.1.0",
			expected:        true,
		},
		{
			name:            "older version",
			manifestVersion: "0.9.0",
			expected:        true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := &Manifest{
				Version: tt.manifestVersion,
			}
			
			result := updater.isUpdateNeeded(manifest)
			assert.Equal(t, tt.expected, result)
		})
	}
}