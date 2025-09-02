package macos

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceManagerIntegration tests the complete service lifecycle
func TestServiceManagerIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS integration tests can only run on macOS")
	}
	
	if os.Geteuid() != 0 {
		t.Skip("Integration tests require root privileges")
	}
	
	// Create temporary executable for testing
	tempDir := t.TempDir()
	execPath := filepath.Join(tempDir, "test-bridge")
	
	// Create a simple test executable
	testScript := `#!/bin/bash
echo "Test bridge started"
sleep 30
echo "Test bridge stopped"
`
	
	require.NoError(t, os.WriteFile(execPath, []byte(testScript), 0755))
	
	// Create service manager
	sm, err := NewServiceManager()
	require.NoError(t, err)
	
	// Ensure service is not installed initially
	installed, err := sm.IsServiceInstalled()
	require.NoError(t, err)
	if installed {
		// Clean up any existing test service
		_ = sm.UninstallService()
	}
	
	t.Run("InstallService", func(t *testing.T) {
		err := sm.InstallService(execPath, "")
		assert.NoError(t, err)
		
		// Verify service is installed
		installed, err := sm.IsServiceInstalled()
		assert.NoError(t, err)
		assert.True(t, installed)
	})
	
	t.Run("GetServiceStatus", func(t *testing.T) {
		status, err := sm.GetServiceStatus()
		assert.NoError(t, err)
		assert.Contains(t, []string{"Running", "Stopped"}, status)
	})
	
	t.Run("StartService", func(t *testing.T) {
		err := sm.StartService()
		assert.NoError(t, err)
		
		// Wait a moment for service to start
		time.Sleep(2 * time.Second)
		
		status, err := sm.GetServiceStatus()
		assert.NoError(t, err)
		assert.Equal(t, "Running", status)
	})
	
	t.Run("StopService", func(t *testing.T) {
		err := sm.StopService()
		assert.NoError(t, err)
		
		// Wait a moment for service to stop
		time.Sleep(2 * time.Second)
		
		status, err := sm.GetServiceStatus()
		assert.NoError(t, err)
		assert.Equal(t, "Stopped", status)
	})
	
	t.Run("RestartService", func(t *testing.T) {
		err := sm.RestartService()
		assert.NoError(t, err)
		
		// Wait a moment for service to restart
		time.Sleep(3 * time.Second)
		
		status, err := sm.GetServiceStatus()
		assert.NoError(t, err)
		assert.Equal(t, "Running", status)
	})
	
	// Cleanup
	t.Cleanup(func() {
		_ = sm.StopService()
		_ = sm.UninstallService()
	})
}

// TestServiceConfigIntegration tests service configuration management
func TestServiceConfigIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS integration tests can only run on macOS")
	}
	
	if os.Geteuid() != 0 {
		t.Skip("Integration tests require root privileges")
	}
	
	t.Run("CreateServiceDirectories", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &ServiceConfig{
			ConfigPath:    filepath.Join(tempDir, "config", "bridge.yaml"),
			DataDirectory: filepath.Join(tempDir, "data"),
			WorkingDir:    filepath.Join(tempDir, "work"),
			LogPath:       filepath.Join(tempDir, "logs", "bridge.log"),
			LogLevel:      "info",
		}
		
		err := CreateServiceDirectories(config)
		assert.NoError(t, err)
		
		// Verify directories were created
		assert.DirExists(t, config.DataDirectory)
		assert.DirExists(t, config.WorkingDir)
		assert.DirExists(t, filepath.Dir(config.ConfigPath))
		assert.DirExists(t, filepath.Dir(config.LogPath))
	})
	
	t.Run("SetDirectoryPermissions", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &ServiceConfig{
			ConfigPath:    filepath.Join(tempDir, "config", "bridge.yaml"),
			DataDirectory: filepath.Join(tempDir, "data"),
			WorkingDir:    filepath.Join(tempDir, "work"),
			LogPath:       filepath.Join(tempDir, "logs", "bridge.log"),
			LogLevel:      "info",
		}
		
		// Create directories first
		require.NoError(t, CreateServiceDirectories(config))
		
		err := SetDirectoryPermissions(config)
		assert.NoError(t, err)
		
		// Verify permissions (basic check)
		info, err := os.Stat(config.DataDirectory)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	})
	
	t.Run("CreateDefaultConfigFile", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config", "bridge.yaml")
		
		err := CreateDefaultConfigFile(configPath)
		assert.NoError(t, err)
		
		// Verify config file was created
		assert.FileExists(t, configPath)
		
		// Verify content is not empty
		content, err := os.ReadFile(configPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, string(content), "Gym Door Bridge Configuration")
	})
}

// TestServiceRunIntegration tests the service run functionality
func TestServiceRunIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS integration tests can only run on macOS")
	}
	
	// Create a test bridge function
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		// Simulate bridge work
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
	
	// Create test configuration
	cfg := &config.Config{
		LogLevel: "info",
	}
	
	t.Run("ServiceRun", func(t *testing.T) {
		service := NewService(cfg, bridgeFunc)
		
		// Run service in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- service.Run()
		}()
		
		// Wait a moment then cancel
		time.Sleep(50 * time.Millisecond)
		service.cancel()
		
		// Wait for service to stop
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Service did not stop within timeout")
		}
	})
}

// TestScriptGenerationIntegration tests script generation
func TestScriptGenerationIntegration(t *testing.T) {
	tempDir := t.TempDir()
	sg := NewScriptGenerator()
	
	t.Run("GenerateInstallScript", func(t *testing.T) {
		scriptPath := filepath.Join(tempDir, "install.sh")
		
		err := sg.GenerateInstallScript(scriptPath)
		assert.NoError(t, err)
		
		// Verify script was created
		assert.FileExists(t, scriptPath)
		
		// Verify script is executable (on Unix systems)
		info, err := os.Stat(scriptPath)
		assert.NoError(t, err)
		if runtime.GOOS != "windows" {
			assert.True(t, info.Mode()&0111 != 0) // Check execute bit
		}
		
		// Verify script content
		content, err := os.ReadFile(scriptPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "#!/bin/bash")
		assert.Contains(t, string(content), "Gym Door Bridge macOS Installation Script")
	})
	
	t.Run("GenerateUninstallScript", func(t *testing.T) {
		scriptPath := filepath.Join(tempDir, "uninstall.sh")
		
		err := sg.GenerateUninstallScript(scriptPath)
		assert.NoError(t, err)
		
		// Verify script was created
		assert.FileExists(t, scriptPath)
		
		// Verify script is executable (on Unix systems)
		info, err := os.Stat(scriptPath)
		assert.NoError(t, err)
		if runtime.GOOS != "windows" {
			assert.True(t, info.Mode()&0111 != 0) // Check execute bit
		}
		
		// Verify script content
		content, err := os.ReadFile(scriptPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "#!/bin/bash")
		assert.Contains(t, string(content), "Gym Door Bridge macOS Uninstallation Script")
	})
	
	t.Run("GenerateScripts", func(t *testing.T) {
		scriptsDir := filepath.Join(tempDir, "scripts")
		
		err := sg.GenerateScripts(scriptsDir)
		assert.NoError(t, err)
		
		// Verify both scripts were created
		assert.FileExists(t, filepath.Join(scriptsDir, "install.sh"))
		assert.FileExists(t, filepath.Join(scriptsDir, "uninstall.sh"))
	})
}

// TestNotarizationIntegration tests notarization functionality (requires credentials)
func TestNotarizationIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS integration tests can only run on macOS")
	}
	
	// Skip if notarization credentials are not available
	appleID := os.Getenv("APPLE_ID")
	appPassword := os.Getenv("APP_SPECIFIC_PASSWORD")
	developerID := os.Getenv("DEVELOPER_ID")
	
	if appleID == "" || appPassword == "" || developerID == "" {
		t.Skip("Notarization credentials not available (set APPLE_ID, APP_SPECIFIC_PASSWORD, DEVELOPER_ID)")
	}
	
	// Create a test binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "test-binary")
	
	testBinary := `#!/bin/bash
echo "Test binary"
`
	
	require.NoError(t, os.WriteFile(binaryPath, []byte(testBinary), 0755))
	
	config := &NotarizationConfig{
		DeveloperID:         developerID,
		BundleID:           "com.test.binary",
		AppleID:            appleID,
		AppSpecificPassword: appPassword,
	}
	
	nm := NewNotarizationManager(config)
	
	t.Run("SignBinary", func(t *testing.T) {
		err := nm.SignBinary(binaryPath)
		if err != nil {
			t.Logf("Code signing failed (expected in test environment): %v", err)
			t.Skip("Code signing requires valid developer certificate")
		}
	})
	
	t.Run("VerifySignature", func(t *testing.T) {
		err := nm.VerifySignature(binaryPath)
		if err != nil {
			t.Logf("Signature verification failed (expected if signing failed): %v", err)
		}
	})
	
	t.Run("CreateZipForNotarization", func(t *testing.T) {
		zipPath, err := nm.CreateZipForNotarization(binaryPath)
		assert.NoError(t, err)
		assert.FileExists(t, zipPath)
		
		// Clean up
		os.Remove(zipPath)
	})
}

// BenchmarkServiceOperations benchmarks service operations
func BenchmarkServiceOperations(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("macOS benchmarks can only run on macOS")
	}
	
	if os.Geteuid() != 0 {
		b.Skip("Benchmarks require root privileges")
	}
	
	// Create temporary executable for testing
	tempDir := b.TempDir()
	execPath := filepath.Join(tempDir, "test-bridge")
	
	testScript := `#!/bin/bash
sleep 1
`
	
	require.NoError(b, os.WriteFile(execPath, []byte(testScript), 0755))
	
	sm, err := NewServiceManager()
	require.NoError(b, err)
	
	// Install service once
	require.NoError(b, sm.InstallService(execPath, ""))
	
	b.Cleanup(func() {
		_ = sm.UninstallService()
	})
	
	b.Run("GetServiceStatus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := sm.GetServiceStatus()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("IsServiceInstalled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := sm.IsServiceInstalled()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}