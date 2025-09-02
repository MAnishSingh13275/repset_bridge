package windows

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

// TestServiceIntegration tests the complete service lifecycle
func TestServiceIntegration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service integration tests only run on Windows")
	}
	
	// This test requires administrator privileges
	if !isRunningAsAdmin() {
		t.Skip("Service integration tests require administrator privileges")
	}
	
	// Create a temporary directory for test configuration
	tempDir, err := os.MkdirTemp("", "service_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create test configuration
	testConfig := &ServiceConfig{
		ConfigPath:    filepath.Join(tempDir, "config.yaml"),
		LogLevel:      "debug",
		DataDirectory: filepath.Join(tempDir, "data"),
		WorkingDir:    tempDir,
	}
	
	// Validate configuration
	err = ValidateServiceConfig(testConfig)
	require.NoError(t, err)
	
	// Create directories
	err = CreateServiceDirectories(testConfig)
	require.NoError(t, err)
	
	// Verify directories exist
	assert.DirExists(t, testConfig.WorkingDir)
	assert.DirExists(t, testConfig.DataDirectory)
	assert.DirExists(t, filepath.Dir(testConfig.ConfigPath))
	
	t.Log("Service integration test setup completed successfully")
}

// TestServiceManagerIntegration tests service manager operations
func TestServiceManagerIntegration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service manager integration tests only run on Windows")
	}
	
	// Create service manager
	sm, err := NewServiceManager()
	if err != nil {
		t.Skipf("Cannot create service manager (likely permissions): %v", err)
	}
	defer sm.Close()
	
	// Test basic operations that don't require service installation
	
	// Check if service is installed
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		t.Logf("Warning: Could not check service installation: %v", err)
	} else {
		t.Logf("Service installed: %v", installed)
	}
	
	// If service is not installed, test installation would require admin privileges
	// and actual binary, so we'll skip that in automated tests
	
	if installed {
		// If service is installed, test status operations
		status, err := sm.GetServiceStatus()
		if err != nil {
			t.Logf("Warning: Could not get service status: %v", err)
		} else {
			t.Logf("Service status: %s", status)
		}
	}
}

// TestServiceConfigurationIntegration tests configuration management
func TestServiceConfigurationIntegration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service configuration tests only run on Windows")
	}
	
	// Test configuration operations
	originalConfig := &ServiceConfig{
		ConfigPath:    "C:\\test\\integration\\config.yaml",
		LogLevel:      "debug",
		DataDirectory: "C:\\test\\integration\\data",
		WorkingDir:    "C:\\test\\integration",
	}
	
	// Save configuration
	err := SaveServiceConfig(originalConfig)
	if err != nil {
		t.Skipf("Cannot save service config (likely permissions): %v", err)
	}
	
	// Load configuration
	loadedConfig, err := LoadServiceConfig()
	require.NoError(t, err)
	
	// Verify configuration matches
	assert.Equal(t, originalConfig.ConfigPath, loadedConfig.ConfigPath)
	assert.Equal(t, originalConfig.LogLevel, loadedConfig.LogLevel)
	assert.Equal(t, originalConfig.DataDirectory, loadedConfig.DataDirectory)
	assert.Equal(t, originalConfig.WorkingDir, loadedConfig.WorkingDir)
	
	// Clean up
	err = RemoveServiceConfig()
	if err != nil {
		t.Logf("Warning: Could not remove service config: %v", err)
	}
}

// TestServiceExecutionIntegration tests service execution
func TestServiceExecutionIntegration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service execution tests only run on Windows")
	}
	
	// Create a mock bridge function for testing
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		// Simulate bridge work
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
	
	// Create service
	cfg := &config.Config{}
	service := NewService(cfg, bridgeFunc)
	require.NotNil(t, service)
	
	// Test service context
	assert.NotNil(t, service.ctx)
	assert.NotNil(t, service.cancel)
	
	// Test context cancellation
	service.cancel()
	
	select {
	case <-service.ctx.Done():
		// Expected - context should be cancelled
	case <-time.After(1 * time.Second):
		t.Error("Service context was not cancelled within timeout")
	}
	
	t.Log("Service execution integration test completed")
}

// TestEndToEndServiceFlow tests the complete service workflow
func TestEndToEndServiceFlow(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("End-to-end service tests only run on Windows")
	}
	
	// This test simulates the complete service workflow without actually
	// installing the service (which would require admin privileges)
	
	// 1. Create service configuration
	serviceConfig := DefaultServiceConfig()
	require.NotNil(t, serviceConfig)
	
	// 2. Validate configuration
	err := ValidateServiceConfig(serviceConfig)
	require.NoError(t, err)
	
	// 3. Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "end_to_end_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	testConfig := &ServiceConfig{
		ConfigPath:    filepath.Join(tempDir, "config.yaml"),
		LogLevel:      "info",
		DataDirectory: filepath.Join(tempDir, "data"),
		WorkingDir:    tempDir,
	}
	
	err = CreateServiceDirectories(testConfig)
	require.NoError(t, err)
	
	// 4. Test service creation
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		return nil
	}
	
	appConfig := &config.Config{}
	service := NewService(appConfig, bridgeFunc)
	require.NotNil(t, service)
	
	// 5. Test service manager creation
	sm, err := NewServiceManager()
	if err != nil {
		t.Logf("Cannot create service manager (expected in test environment): %v", err)
	} else {
		defer sm.Close()
		
		// Test basic service manager operations
		_, err = sm.IsServiceInstalled()
		if err != nil {
			t.Logf("Service manager operation failed (expected): %v", err)
		}
	}
	
	t.Log("End-to-end service flow test completed successfully")
}

// TestServiceErrorHandling tests error handling in service operations
func TestServiceErrorHandling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Service error handling tests only run on Windows")
	}
	
	// Test service creation with nil config
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		return nil
	}
	
	service := NewService(nil, bridgeFunc)
	assert.NotNil(t, service)
	assert.Nil(t, service.config)
	
	// Test service with error-returning bridge function
	errorBridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		return assert.AnError
	}
	
	errorService := NewService(&config.Config{}, errorBridgeFunc)
	assert.NotNil(t, errorService)
	
	// Test configuration validation errors
	invalidConfig := &ServiceConfig{
		ConfigPath:    "", // Invalid - empty path
		LogLevel:      "info",
		DataDirectory: "C:\\test",
		WorkingDir:    "C:\\test",
	}
	
	err := ValidateServiceConfig(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config path cannot be empty")
}

// TestServicePerformance tests service performance characteristics
func TestServicePerformance(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Service performance tests only run on Windows")
	}
	
	// Test service creation performance
	start := time.Now()
	
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		return nil
	}
	
	for i := 0; i < 100; i++ {
		service := NewService(&config.Config{}, bridgeFunc)
		service.cancel() // Clean up
	}
	
	duration := time.Since(start)
	t.Logf("Created 100 services in %v (avg: %v per service)", duration, duration/100)
	
	// Performance should be reasonable
	assert.Less(t, duration, 1*time.Second, "Service creation should be fast")
}

// Benchmark service operations
func BenchmarkServiceCreation(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Service benchmarks only run on Windows")
	}
	
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		return nil
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := NewService(&config.Config{}, bridgeFunc)
		service.cancel()
	}
}

func BenchmarkServiceManagerCreation(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Service manager benchmarks only run on Windows")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm, err := NewServiceManager()
		if err != nil {
			b.Skipf("Cannot create service manager: %v", err)
		}
		sm.Close()
	}
}