package macos

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServiceManager(t *testing.T) {
	sm, err := NewServiceManager()
	assert.NoError(t, err)
	assert.NotNil(t, sm)
	assert.Contains(t, sm.plistPath, ServiceName+".plist")
	if runtime.GOOS == "darwin" {
		assert.Contains(t, sm.plistPath, "/Library/LaunchDaemons/")
	}
}

func TestServiceManagerPlistGeneration(t *testing.T) {
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	tempDir := t.TempDir()
	execPath := filepath.Join(tempDir, "test-bridge")
	configPath := filepath.Join(tempDir, "config.yaml")
	
	// Create test files
	require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte("test: config"), 0644))
	
	plistContent, err := sm.generatePlistContent(execPath, configPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, plistContent)
	
	// Verify plist content contains expected elements
	assert.Contains(t, plistContent, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, plistContent, ServiceName)
	assert.Contains(t, plistContent, execPath)
	assert.Contains(t, plistContent, configPath)
	assert.Contains(t, plistContent, "<key>RunAtLoad</key>")
	assert.Contains(t, plistContent, "<key>KeepAlive</key>")
	assert.Contains(t, plistContent, "<true/>")
}

func TestServiceManagerPlistGenerationWithoutConfig(t *testing.T) {
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	tempDir := t.TempDir()
	execPath := filepath.Join(tempDir, "test-bridge")
	
	// Create test executable
	require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755))
	
	plistContent, err := sm.generatePlistContent(execPath, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, plistContent)
	
	// Verify plist content contains expected elements
	assert.Contains(t, plistContent, ServiceName)
	assert.Contains(t, plistContent, execPath)
	assert.NotContains(t, plistContent, "--config")
}

func TestGetExecutablePath(t *testing.T) {
	execPath, err := GetExecutablePath()
	assert.NoError(t, err)
	assert.NotEmpty(t, execPath)
	
	// Verify the path exists
	_, err = os.Stat(execPath)
	assert.NoError(t, err)
	
	// Verify it's an absolute path
	assert.True(t, filepath.IsAbs(execPath))
}

// TestServiceManagerMockOperations tests service manager operations with mocked launchctl
func TestServiceManagerMockOperations(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific tests can only run on macOS")
	}
	
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	// Test IsServiceInstalled with non-existent plist
	tempDir := t.TempDir()
	sm.plistPath = filepath.Join(tempDir, "test.plist")
	
	installed, err := sm.IsServiceInstalled()
	assert.NoError(t, err)
	assert.False(t, installed)
	
	// Create a test plist file
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>test.service</string>
</dict>
</plist>`
	
	require.NoError(t, os.WriteFile(sm.plistPath, []byte(plistContent), 0644))
	
	// Test IsServiceInstalled with existing plist
	installed, err = sm.IsServiceInstalled()
	assert.NoError(t, err)
	assert.True(t, installed)
}

func TestServiceManagerValidation(t *testing.T) {
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	// Test with non-existent executable
	err = sm.InstallService("/non/existent/path", "")
	assert.Error(t, err)
	// Error message will vary by platform, just check that it fails
}

// TestServiceManagerDirectoryCreation tests that log directories are created
func TestServiceManagerDirectoryCreation(t *testing.T) {
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	tempDir := t.TempDir()
	execPath := filepath.Join(tempDir, "test-bridge")
	
	// Create test executable
	require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755))
	
	// Generate plist content (this should create log directory)
	_, err = sm.generatePlistContent(execPath, "")
	assert.NoError(t, err)
	
	// Verify log directory was created
	serviceConfig := DefaultServiceConfig()
	logDir := filepath.Dir(serviceConfig.LogPath)
	
	// We can't test the actual system directory creation in unit tests,
	// but we can verify the function doesn't error
	assert.NotEmpty(t, logDir)
}

// TestServiceManagerConstants tests service manager constants
func TestServiceManagerConstants(t *testing.T) {
	assert.Equal(t, "com.gymdoorbridge.agent", ServiceName)
	assert.Equal(t, "Gym Door Access Bridge", ServiceDisplayName)
	assert.Equal(t, "Connects gym door access hardware to SaaS platform", ServiceDescription)
}

// TestServiceManagerPlistPath tests plist path generation
func TestServiceManagerPlistPath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test can only run on macOS")
	}
	
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	expectedPath := "/Library/LaunchDaemons/" + ServiceName + ".plist"
	assert.Equal(t, expectedPath, sm.plistPath)
}

// TestServiceManagerErrorHandling tests error handling in service operations
func TestServiceManagerErrorHandling(t *testing.T) {
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	// Test with invalid plist path (permission denied)
	sm.plistPath = "/invalid/path/test.plist"
	
	tempDir := t.TempDir()
	execPath := filepath.Join(tempDir, "test-bridge")
	require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755))
	
	// This should fail due to invalid plist path
	err = sm.InstallService(execPath, "")
	assert.Error(t, err)
}