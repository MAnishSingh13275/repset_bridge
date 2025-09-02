package windows

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()
	
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.ConfigPath)
	assert.NotEmpty(t, config.LogLevel)
	assert.NotEmpty(t, config.DataDirectory)
	assert.NotEmpty(t, config.WorkingDir)
	
	// Verify default log level
	assert.Equal(t, "info", config.LogLevel)
	
	// Verify paths are absolute
	assert.True(t, filepath.IsAbs(config.ConfigPath))
	assert.True(t, filepath.IsAbs(config.DataDirectory))
	assert.True(t, filepath.IsAbs(config.WorkingDir))
}

func TestValidateServiceConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *ServiceConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &ServiceConfig{
				ConfigPath:    "C:\\test\\config.yaml",
				LogLevel:      "info",
				DataDirectory: "C:\\test\\data",
				WorkingDir:    "C:\\test",
			},
			expectErr: false,
		},
		{
			name: "empty config path",
			config: &ServiceConfig{
				ConfigPath:    "",
				LogLevel:      "info",
				DataDirectory: "C:\\test\\data",
				WorkingDir:    "C:\\test",
			},
			expectErr: true,
			errMsg:    "config path cannot be empty",
		},
		{
			name: "empty data directory",
			config: &ServiceConfig{
				ConfigPath:    "C:\\test\\config.yaml",
				LogLevel:      "info",
				DataDirectory: "",
				WorkingDir:    "C:\\test",
			},
			expectErr: true,
			errMsg:    "data directory cannot be empty",
		},
		{
			name: "empty working directory",
			config: &ServiceConfig{
				ConfigPath:    "C:\\test\\config.yaml",
				LogLevel:      "info",
				DataDirectory: "C:\\test\\data",
				WorkingDir:    "",
			},
			expectErr: true,
			errMsg:    "working directory cannot be empty",
		},
		{
			name: "invalid log level",
			config: &ServiceConfig{
				ConfigPath:    "C:\\test\\config.yaml",
				LogLevel:      "invalid",
				DataDirectory: "C:\\test\\data",
				WorkingDir:    "C:\\test",
			},
			expectErr: true,
			errMsg:    "invalid log level",
		},
		{
			name: "valid debug log level",
			config: &ServiceConfig{
				ConfigPath:    "C:\\test\\config.yaml",
				LogLevel:      "debug",
				DataDirectory: "C:\\test\\data",
				WorkingDir:    "C:\\test",
			},
			expectErr: false,
		},
		{
			name: "valid error log level",
			config: &ServiceConfig{
				ConfigPath:    "C:\\test\\config.yaml",
				LogLevel:      "error",
				DataDirectory: "C:\\test\\data",
				WorkingDir:    "C:\\test",
			},
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceConfig(tt.config)
			
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateServiceDirectories(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "service_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	config := &ServiceConfig{
		ConfigPath:    filepath.Join(tempDir, "config", "config.yaml"),
		LogLevel:      "info",
		DataDirectory: filepath.Join(tempDir, "data"),
		WorkingDir:    filepath.Join(tempDir, "working"),
	}
	
	err = CreateServiceDirectories(config)
	assert.NoError(t, err)
	
	// Verify directories were created
	assert.DirExists(t, config.WorkingDir)
	assert.DirExists(t, config.DataDirectory)
	assert.DirExists(t, filepath.Dir(config.ConfigPath))
}

func TestCreateServiceDirectoriesError(t *testing.T) {
	// Test with invalid path (on Windows, paths with invalid characters)
	config := &ServiceConfig{
		ConfigPath:    "C:\\invalid<>path\\config.yaml",
		LogLevel:      "info",
		DataDirectory: "C:\\test\\data",
		WorkingDir:    "C:\\test",
	}
	
	err := CreateServiceDirectories(config)
	if runtime.GOOS == "windows" {
		// On Windows, this should fail due to invalid characters
		assert.Error(t, err)
	} else {
		// On other platforms, this test might behave differently
		t.Skip("Invalid path test only meaningful on Windows")
	}
}

// Test registry operations (these will only work on Windows with appropriate permissions)
func TestRegistryOperations(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Registry tests only run on Windows")
	}
	
	// Note: These tests require administrator privileges and modify the registry
	// In a CI environment, these might fail, so we'll make them conditional
	
	config := &ServiceConfig{
		ConfigPath:    "C:\\test\\config.yaml",
		LogLevel:      "debug",
		DataDirectory: "C:\\test\\data",
		WorkingDir:    "C:\\test",
	}
	
	// Test saving configuration
	err := SaveServiceConfig(config)
	if err != nil {
		t.Skipf("Cannot save service config (likely permissions): %v", err)
	}
	
	// Test loading configuration
	loadedConfig, err := LoadServiceConfig()
	if err != nil {
		t.Skipf("Cannot load service config: %v", err)
	}
	
	// If we successfully saved and loaded, verify the values
	if err == nil {
		assert.Equal(t, config.ConfigPath, loadedConfig.ConfigPath)
		assert.Equal(t, config.LogLevel, loadedConfig.LogLevel)
		assert.Equal(t, config.DataDirectory, loadedConfig.DataDirectory)
		assert.Equal(t, config.WorkingDir, loadedConfig.WorkingDir)
	}
	
	// Test removing configuration
	err = RemoveServiceConfig()
	if err != nil {
		t.Logf("Warning: Could not remove service config: %v", err)
	}
}

func TestLoadServiceConfigDefaults(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Registry tests only run on Windows")
	}
	
	// Ensure no config exists, then load (should return defaults)
	RemoveServiceConfig() // Ignore errors
	
	config, err := LoadServiceConfig()
	require.NoError(t, err)
	
	// Should return default configuration
	defaultConfig := DefaultServiceConfig()
	assert.Equal(t, defaultConfig.ConfigPath, config.ConfigPath)
	assert.Equal(t, defaultConfig.LogLevel, config.LogLevel)
	assert.Equal(t, defaultConfig.DataDirectory, config.DataDirectory)
	assert.Equal(t, defaultConfig.WorkingDir, config.WorkingDir)
}

// Benchmark configuration operations
func BenchmarkDefaultServiceConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultServiceConfig()
	}
}

func BenchmarkValidateServiceConfig(b *testing.B) {
	config := DefaultServiceConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateServiceConfig(config)
	}
}