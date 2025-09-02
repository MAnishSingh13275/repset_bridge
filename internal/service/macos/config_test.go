package macos

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
	assert.Equal(t, "/usr/local/etc/gymdoorbridge/config.yaml", config.ConfigPath)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, "/usr/local/var/lib/gymdoorbridge", config.DataDirectory)
	assert.Equal(t, "/usr/local/lib/gymdoorbridge", config.WorkingDir)
	assert.Equal(t, "/usr/local/var/log/gymdoorbridge/bridge.log", config.LogPath)
}

func TestLoadServiceConfig(t *testing.T) {
	// LoadServiceConfig should return default config on macOS
	config, err := LoadServiceConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	
	// Should match default config
	defaultConfig := DefaultServiceConfig()
	assert.Equal(t, defaultConfig.ConfigPath, config.ConfigPath)
	assert.Equal(t, defaultConfig.LogLevel, config.LogLevel)
	assert.Equal(t, defaultConfig.DataDirectory, config.DataDirectory)
	assert.Equal(t, defaultConfig.WorkingDir, config.WorkingDir)
	assert.Equal(t, defaultConfig.LogPath, config.LogPath)
}

func TestCreateServiceDirectories(t *testing.T) {
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
}

func TestValidateServiceConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := &ServiceConfig{
			ConfigPath:    "/path/to/config.yaml",
			DataDirectory: "/path/to/data",
			WorkingDir:    "/path/to/work",
			LogPath:       "/path/to/log.log",
			LogLevel:      "info",
		}
		
		err := ValidateServiceConfig(config)
		assert.NoError(t, err)
	})
	
	t.Run("EmptyConfigPath", func(t *testing.T) {
		config := &ServiceConfig{
			ConfigPath:    "",
			DataDirectory: "/path/to/data",
			WorkingDir:    "/path/to/work",
			LogPath:       "/path/to/log.log",
			LogLevel:      "info",
		}
		
		err := ValidateServiceConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config path cannot be empty")
	})
	
	t.Run("EmptyDataDirectory", func(t *testing.T) {
		config := &ServiceConfig{
			ConfigPath:    "/path/to/config.yaml",
			DataDirectory: "",
			WorkingDir:    "/path/to/work",
			LogPath:       "/path/to/log.log",
			LogLevel:      "info",
		}
		
		err := ValidateServiceConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data directory cannot be empty")
	})
	
	t.Run("EmptyWorkingDir", func(t *testing.T) {
		config := &ServiceConfig{
			ConfigPath:    "/path/to/config.yaml",
			DataDirectory: "/path/to/data",
			WorkingDir:    "",
			LogPath:       "/path/to/log.log",
			LogLevel:      "info",
		}
		
		err := ValidateServiceConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "working directory cannot be empty")
	})
	
	t.Run("EmptyLogPath", func(t *testing.T) {
		config := &ServiceConfig{
			ConfigPath:    "/path/to/config.yaml",
			DataDirectory: "/path/to/data",
			WorkingDir:    "/path/to/work",
			LogPath:       "",
			LogLevel:      "info",
		}
		
		err := ValidateServiceConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "log path cannot be empty")
	})
	
	t.Run("InvalidLogLevel", func(t *testing.T) {
		config := &ServiceConfig{
			ConfigPath:    "/path/to/config.yaml",
			DataDirectory: "/path/to/data",
			WorkingDir:    "/path/to/work",
			LogPath:       "/path/to/log.log",
			LogLevel:      "invalid",
		}
		
		err := ValidateServiceConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid log level")
	})
	
	t.Run("ValidLogLevels", func(t *testing.T) {
		validLevels := []string{"debug", "info", "warn", "error"}
		
		for _, level := range validLevels {
			config := &ServiceConfig{
				ConfigPath:    "/path/to/config.yaml",
				DataDirectory: "/path/to/data",
				WorkingDir:    "/path/to/work",
				LogPath:       "/path/to/log.log",
				LogLevel:      level,
			}
			
			err := ValidateServiceConfig(config)
			assert.NoError(t, err, "Log level %s should be valid", level)
		}
	})
}

func TestSetDirectoryPermissions(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Permission tests are platform-specific")
	}
	
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
	
	info, err = os.Stat(filepath.Dir(config.LogPath))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	
	info, err = os.Stat(filepath.Dir(config.ConfigPath))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestCreateDefaultConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config", "bridge.yaml")
	
	t.Run("CreateNewConfigFile", func(t *testing.T) {
		err := CreateDefaultConfigFile(configPath)
		assert.NoError(t, err)
		
		// Verify config file was created
		assert.FileExists(t, configPath)
		
		// Verify content is not empty
		content, err := os.ReadFile(configPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, content)
		
		// Verify content contains expected sections
		contentStr := string(content)
		assert.Contains(t, contentStr, "Gym Door Bridge Configuration")
		assert.Contains(t, contentStr, "log:")
		assert.Contains(t, contentStr, "adapters:")
		assert.Contains(t, contentStr, "performance:")
		assert.Contains(t, contentStr, "network:")
		assert.Contains(t, contentStr, "queue:")
		assert.Contains(t, contentStr, "security:")
		assert.Contains(t, contentStr, "door:")
	})
	
	t.Run("ExistingConfigFile", func(t *testing.T) {
		// File already exists from previous test
		originalContent, err := os.ReadFile(configPath)
		require.NoError(t, err)
		
		// Try to create again
		err = CreateDefaultConfigFile(configPath)
		assert.NoError(t, err)
		
		// Verify content wasn't changed
		newContent, err := os.ReadFile(configPath)
		assert.NoError(t, err)
		assert.Equal(t, originalContent, newContent)
	})
	
	t.Run("InvalidPath", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("Path validation is platform-specific")
		}
		
		invalidPath := "/invalid/path/that/cannot/be/created/config.yaml"
		
		err := CreateDefaultConfigFile(invalidPath)
		assert.Error(t, err)
	})
}

func TestServiceConfigStructure(t *testing.T) {
	config := &ServiceConfig{
		ConfigPath:    "/test/config.yaml",
		LogLevel:      "debug",
		DataDirectory: "/test/data",
		WorkingDir:    "/test/work",
		LogPath:       "/test/log.log",
	}
	
	// Test that all fields are accessible
	assert.Equal(t, "/test/config.yaml", config.ConfigPath)
	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, "/test/data", config.DataDirectory)
	assert.Equal(t, "/test/work", config.WorkingDir)
	assert.Equal(t, "/test/log.log", config.LogPath)
}

func TestCreateServiceDirectoriesErrorHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Path validation is platform-specific")
	}
	
	// Test with invalid path that cannot be created
	config := &ServiceConfig{
		ConfigPath:    "/invalid/path/config.yaml",
		DataDirectory: "/invalid/path/data",
		WorkingDir:    "/invalid/path/work",
		LogPath:       "/invalid/path/log.log",
		LogLevel:      "info",
	}
	
	err := CreateServiceDirectories(config)
	assert.Error(t, err)
}

func TestSetDirectoryPermissionsErrorHandling(t *testing.T) {
	// Test with non-existent directories
	config := &ServiceConfig{
		ConfigPath:    "/non/existent/config.yaml",
		DataDirectory: "/non/existent/data",
		WorkingDir:    "/non/existent/work",
		LogPath:       "/non/existent/log.log",
		LogLevel:      "info",
	}
	
	err := SetDirectoryPermissions(config)
	assert.Error(t, err)
}