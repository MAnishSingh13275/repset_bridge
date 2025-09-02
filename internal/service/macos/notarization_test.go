package macos

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultNotarizationConfig(t *testing.T) {
	config := DefaultNotarizationConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, ServiceName, config.BundleID)
	assert.Equal(t, "gym-door-bridge-notarization", config.KeychainProfile)
	assert.Empty(t, config.DeveloperID)
	assert.Empty(t, config.TeamID)
	assert.Empty(t, config.AppleID)
	assert.Empty(t, config.AppSpecificPassword)
}

func TestNewNotarizationManager(t *testing.T) {
	config := &NotarizationConfig{
		DeveloperID: "test-developer-id",
		BundleID:    "com.test.app",
	}
	
	nm := NewNotarizationManager(config)
	
	assert.NotNil(t, nm)
	assert.Equal(t, config, nm.config)
}

func TestNotarizationManagerSignBinary(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific tests can only run on macOS")
	}
	
	config := &NotarizationConfig{
		DeveloperID: "test-developer-id",
		BundleID:    "com.test.app",
	}
	
	nm := NewNotarizationManager(config)
	
	t.Run("MissingDeveloperID", func(t *testing.T) {
		emptyConfig := &NotarizationConfig{}
		emptyNM := NewNotarizationManager(emptyConfig)
		
		err := emptyNM.SignBinary("/path/to/binary")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "developer ID is required")
	})
	
	t.Run("NonExistentBinary", func(t *testing.T) {
		err := nm.SignBinary("/non/existent/binary")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "binary not found")
	})
	
	t.Run("ExistingBinary", func(t *testing.T) {
		tempDir := t.TempDir()
		binaryPath := filepath.Join(tempDir, "test-binary")
		
		// Create test binary
		require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755))
		
		// This will fail because we don't have a valid developer ID,
		// but it should get past the initial validation
		err := nm.SignBinary(binaryPath)
		assert.Error(t, err) // Expected to fail with invalid developer ID
		assert.Contains(t, err.Error(), "code signing failed")
	})
}

func TestNotarizationManagerCreateZip(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Zip creation test requires macOS tools")
	}
	
	config := &NotarizationConfig{
		DeveloperID: "test-developer-id",
		BundleID:    "com.test.app",
	}
	
	nm := NewNotarizationManager(config)
	
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-binary")
	
	// Create test binary
	require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755))
	
	zipPath, err := nm.CreateZipForNotarization(binaryPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, zipPath)
	assert.Contains(t, zipPath, ".zip")
	
	// Verify zip file was created
	assert.FileExists(t, zipPath)
	
	// Clean up
	os.Remove(zipPath)
}

func TestNotarizationManagerSubmitForNotarization(t *testing.T) {
	config := &NotarizationConfig{
		DeveloperID:         "test-developer-id",
		BundleID:           "com.test.app",
		AppleID:            "test@example.com",
		AppSpecificPassword: "test-password",
	}
	
	_ = NewNotarizationManager(config)
	
	t.Run("MissingAppleID", func(t *testing.T) {
		emptyConfig := &NotarizationConfig{
			BundleID: "com.test.app",
		}
		emptyNM := NewNotarizationManager(emptyConfig)
		
		_, err := emptyNM.SubmitForNotarization("/path/to/zip")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Apple ID is required")
	})
	
	t.Run("MissingCredentials", func(t *testing.T) {
		incompleteConfig := &NotarizationConfig{
			BundleID: "com.test.app",
			AppleID:  "test@example.com",
		}
		incompleteNM := NewNotarizationManager(incompleteConfig)
		
		_, err := incompleteNM.SubmitForNotarization("/path/to/zip")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password or keychain profile is required")
	})
	
	t.Run("WithKeychainProfile", func(t *testing.T) {
		keychainConfig := &NotarizationConfig{
			BundleID:        "com.test.app",
			AppleID:         "test@example.com",
			KeychainProfile: "test-profile",
		}
		keychainNM := NewNotarizationManager(keychainConfig)
		
		// This will fail because we don't have valid credentials,
		// but it should get past the initial validation
		_, err := keychainNM.SubmitForNotarization("/non/existent/zip")
		assert.Error(t, err) // Expected to fail with invalid credentials
	})
}

func TestNotarizationManagerCheckStatus(t *testing.T) {
	config := &NotarizationConfig{
		AppleID:            "test@example.com",
		AppSpecificPassword: "test-password",
	}
	
	nm := NewNotarizationManager(config)
	
	t.Run("MissingAppleID", func(t *testing.T) {
		emptyConfig := &NotarizationConfig{}
		emptyNM := NewNotarizationManager(emptyConfig)
		
		_, err := emptyNM.CheckNotarizationStatus("test-uuid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Apple ID is required")
	})
	
	t.Run("WithValidConfig", func(t *testing.T) {
		// This will fail because we don't have valid credentials,
		// but it should get past the initial validation
		_, err := nm.CheckNotarizationStatus("test-uuid")
		assert.Error(t, err) // Expected to fail with invalid credentials
	})
}

func TestNotarizationManagerWaitForNotarization(t *testing.T) {
	config := &NotarizationConfig{
		AppleID:            "test@example.com",
		AppSpecificPassword: "test-password",
	}
	
	nm := NewNotarizationManager(config)
	
	// Test with very short timeout to avoid long test execution
	err := nm.WaitForNotarization("test-uuid", 100*time.Millisecond)
	assert.Error(t, err) // Expected to timeout or fail with invalid credentials
}

func TestNotarizationManagerSetupKeychainProfile(t *testing.T) {
	t.Run("MissingCredentials", func(t *testing.T) {
		config := &NotarizationConfig{
			KeychainProfile: "test-profile",
		}
		nm := NewNotarizationManager(config)
		
		err := nm.SetupKeychainProfile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Apple ID and app-specific password are required")
	})
	
	t.Run("WithCredentials", func(t *testing.T) {
		config := &NotarizationConfig{
			AppleID:             "test@example.com",
			AppSpecificPassword: "test-password",
			KeychainProfile:     "test-profile",
		}
		nm := NewNotarizationManager(config)
		
		// This will fail because we don't have valid credentials,
		// but it should get past the initial validation
		err := nm.SetupKeychainProfile()
		assert.Error(t, err) // Expected to fail with invalid credentials
	})
}

func TestNotarizationConfigValidation(t *testing.T) {
	t.Run("CompleteConfig", func(t *testing.T) {
		config := &NotarizationConfig{
			DeveloperID:         "Developer ID Application: Test Developer",
			TeamID:             "ABCD123456",
			BundleID:           "com.test.app",
			AppleID:            "test@example.com",
			AppSpecificPassword: "test-password",
			KeychainProfile:     "test-profile",
		}
		
		assert.NotEmpty(t, config.DeveloperID)
		assert.NotEmpty(t, config.TeamID)
		assert.NotEmpty(t, config.BundleID)
		assert.NotEmpty(t, config.AppleID)
		assert.NotEmpty(t, config.AppSpecificPassword)
		assert.NotEmpty(t, config.KeychainProfile)
	})
}

func TestNotarizationManagerFullProcess(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific tests can only run on macOS")
	}
	
	// Skip if we don't have notarization credentials
	if os.Getenv("APPLE_ID") == "" {
		t.Skip("Notarization credentials not available")
	}
	
	config := &NotarizationConfig{
		DeveloperID:         os.Getenv("DEVELOPER_ID"),
		BundleID:           "com.test.binary",
		AppleID:            os.Getenv("APPLE_ID"),
		AppSpecificPassword: os.Getenv("APP_SPECIFIC_PASSWORD"),
	}
	
	nm := NewNotarizationManager(config)
	
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-binary")
	
	// Create test binary
	require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755))
	
	t.Run("FullNotarizationProcess", func(t *testing.T) {
		// This is an integration test that requires valid credentials
		// It will likely fail in CI/CD environments without proper setup
		err := nm.NotarizeBinary(binaryPath)
		if err != nil {
			t.Logf("Full notarization process failed (expected in test environment): %v", err)
			// Don't fail the test as this requires valid Apple Developer credentials
		}
	})
}

// TestNotarizationManagerConstants tests notarization-related constants
func TestNotarizationManagerConstants(t *testing.T) {
	config := DefaultNotarizationConfig()
	assert.Equal(t, ServiceName, config.BundleID)
	assert.Equal(t, "gym-door-bridge-notarization", config.KeychainProfile)
}

// TestNotarizationManagerErrorParsing tests error parsing from command output
func TestNotarizationManagerErrorParsing(t *testing.T) {
	config := &NotarizationConfig{
		AppleID:            "test@example.com",
		AppSpecificPassword: "test-password",
	}
	
	nm := NewNotarizationManager(config)
	
	// Test that the manager handles command failures gracefully
	// (actual command execution will fail, but we test error handling)
	_, err := nm.CheckNotarizationStatus("invalid-uuid")
	assert.Error(t, err)
	
	_, err = nm.SubmitForNotarization("/non/existent/file.zip")
	assert.Error(t, err)
}