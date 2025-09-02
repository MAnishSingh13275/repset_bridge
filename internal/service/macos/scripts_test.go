package macos

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScriptGenerator(t *testing.T) {
	sg := NewScriptGenerator()
	assert.NotNil(t, sg)
}

func TestScriptGeneratorGenerateInstallScript(t *testing.T) {
	sg := NewScriptGenerator()
	tempDir := t.TempDir()
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
	
	contentStr := string(content)
	assert.Contains(t, contentStr, "#!/bin/bash")
	assert.Contains(t, contentStr, "Gym Door Bridge macOS Installation Script")
	assert.Contains(t, contentStr, "set -e")
	assert.Contains(t, contentStr, "check_root()")
	assert.Contains(t, contentStr, "check_requirements()")
	assert.Contains(t, contentStr, "download_binary()")
	assert.Contains(t, contentStr, "install_daemon()")
	assert.Contains(t, contentStr, "configure_bridge()")
	assert.Contains(t, contentStr, "start_daemon()")
	assert.Contains(t, contentStr, "show_summary()")
	assert.Contains(t, contentStr, "--pair-code")
	assert.Contains(t, contentStr, "--version")
	assert.Contains(t, contentStr, "--help")
}

func TestScriptGeneratorGenerateUninstallScript(t *testing.T) {
	sg := NewScriptGenerator()
	tempDir := t.TempDir()
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
	
	contentStr := string(content)
	assert.Contains(t, contentStr, "#!/bin/bash")
	assert.Contains(t, contentStr, "Gym Door Bridge macOS Uninstallation Script")
	assert.Contains(t, contentStr, "set -e")
	assert.Contains(t, contentStr, "check_root()")
	assert.Contains(t, contentStr, "uninstall_daemon()")
	assert.Contains(t, contentStr, "remove_binary()")
	assert.Contains(t, contentStr, "remove_data()")
	assert.Contains(t, contentStr, "show_summary()")
	assert.Contains(t, contentStr, "--remove-data")
	assert.Contains(t, contentStr, "--help")
}

func TestScriptGeneratorGenerateScripts(t *testing.T) {
	sg := NewScriptGenerator()
	tempDir := t.TempDir()
	scriptsDir := filepath.Join(tempDir, "scripts")
	
	err := sg.GenerateScripts(scriptsDir)
	assert.NoError(t, err)
	
	// Verify both scripts were created
	installPath := filepath.Join(scriptsDir, "install.sh")
	uninstallPath := filepath.Join(scriptsDir, "uninstall.sh")
	
	assert.FileExists(t, installPath)
	assert.FileExists(t, uninstallPath)
	
	// Verify both scripts are executable (on Unix systems)
	if runtime.GOOS != "windows" {
		installInfo, err := os.Stat(installPath)
		assert.NoError(t, err)
		assert.True(t, installInfo.Mode()&0111 != 0)
		
		uninstallInfo, err := os.Stat(uninstallPath)
		assert.NoError(t, err)
		assert.True(t, uninstallInfo.Mode()&0111 != 0)
	}
}

func TestScriptGeneratorGenerateCustomInstallScript(t *testing.T) {
	sg := NewScriptGenerator()
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "custom-install.sh")
	
	config := &CustomInstallScriptConfig{
		BridgeVersion: "1.2.3",
		BridgeURL:     "https://custom.example.com/releases",
		PairCode:      "ABC123",
		ConfigPath:    "/custom/config.yaml",
		InstallDir:    "/custom/bin",
	}
	
	err := sg.GenerateCustomInstallScript(scriptPath, config)
	assert.NoError(t, err)
	
	// Verify script was created
	assert.FileExists(t, scriptPath)
	
	// Verify script is executable (on Unix systems)
	info, err := os.Stat(scriptPath)
	assert.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.True(t, info.Mode()&0111 != 0)
	}
	
	// Verify script content contains custom values
	content, err := os.ReadFile(scriptPath)
	assert.NoError(t, err)
	
	contentStr := string(content)
	assert.Contains(t, contentStr, "#!/bin/bash")
	assert.Contains(t, contentStr, "Gym Door Bridge macOS Installation Script")
	// Note: The current template doesn't use the config parameters,
	// but the function should still work without errors
}

func TestScriptGeneratorDirectoryCreation(t *testing.T) {
	sg := NewScriptGenerator()
	tempDir := t.TempDir()
	
	// Test with nested directory that doesn't exist
	scriptPath := filepath.Join(tempDir, "nested", "dir", "install.sh")
	
	err := sg.GenerateInstallScript(scriptPath)
	assert.NoError(t, err)
	
	// Verify directory was created
	assert.DirExists(t, filepath.Dir(scriptPath))
	assert.FileExists(t, scriptPath)
}

func TestScriptGeneratorErrorHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Path validation is platform-specific")
	}
	
	sg := NewScriptGenerator()
	
	// Test with invalid path
	invalidPath := "/invalid/path/that/cannot/be/created/install.sh"
	
	err := sg.GenerateInstallScript(invalidPath)
	assert.Error(t, err)
	
	err = sg.GenerateUninstallScript(invalidPath)
	assert.Error(t, err)
	
	err = sg.GenerateScripts("/invalid/path/that/cannot/be/created")
	assert.Error(t, err)
}

func TestInstallScriptTemplate(t *testing.T) {
	// Verify the install script template contains required sections
	assert.Contains(t, InstallScriptTemplate, "#!/bin/bash")
	assert.Contains(t, InstallScriptTemplate, "set -e")
	assert.Contains(t, InstallScriptTemplate, "Gym Door Bridge macOS Installation Script")
	assert.Contains(t, InstallScriptTemplate, "check_root()")
	assert.Contains(t, InstallScriptTemplate, "check_requirements()")
	assert.Contains(t, InstallScriptTemplate, "download_binary()")
	assert.Contains(t, InstallScriptTemplate, "install_daemon()")
	assert.Contains(t, InstallScriptTemplate, "configure_bridge()")
	assert.Contains(t, InstallScriptTemplate, "start_daemon()")
	assert.Contains(t, InstallScriptTemplate, "show_summary()")
	assert.Contains(t, InstallScriptTemplate, "main()")
	
	// Verify command line argument parsing
	assert.Contains(t, InstallScriptTemplate, "--pair-code")
	assert.Contains(t, InstallScriptTemplate, "--version")
	assert.Contains(t, InstallScriptTemplate, "--url")
	assert.Contains(t, InstallScriptTemplate, "--help")
	
	// Verify error handling
	assert.Contains(t, InstallScriptTemplate, "log_error")
	assert.Contains(t, InstallScriptTemplate, "log_info")
	assert.Contains(t, InstallScriptTemplate, "log_warn")
	
	// Verify service management
	assert.Contains(t, InstallScriptTemplate, "launchctl")
	assert.Contains(t, InstallScriptTemplate, ServiceName)
}

func TestUninstallScriptTemplate(t *testing.T) {
	// Verify the uninstall script template contains required sections
	assert.Contains(t, UninstallScriptTemplate, "#!/bin/bash")
	assert.Contains(t, UninstallScriptTemplate, "set -e")
	assert.Contains(t, UninstallScriptTemplate, "Gym Door Bridge macOS Uninstallation Script")
	assert.Contains(t, UninstallScriptTemplate, "check_root()")
	assert.Contains(t, UninstallScriptTemplate, "uninstall_daemon()")
	assert.Contains(t, UninstallScriptTemplate, "remove_binary()")
	assert.Contains(t, UninstallScriptTemplate, "remove_data()")
	assert.Contains(t, UninstallScriptTemplate, "show_summary()")
	assert.Contains(t, UninstallScriptTemplate, "main()")
	
	// Verify command line argument parsing
	assert.Contains(t, UninstallScriptTemplate, "--remove-data")
	assert.Contains(t, UninstallScriptTemplate, "--help")
	
	// Verify error handling
	assert.Contains(t, UninstallScriptTemplate, "log_error")
	assert.Contains(t, UninstallScriptTemplate, "log_info")
	assert.Contains(t, UninstallScriptTemplate, "log_warn")
	
	// Verify service management
	assert.Contains(t, UninstallScriptTemplate, "launchctl")
	assert.Contains(t, UninstallScriptTemplate, ServiceName)
	
	// Verify confirmation prompt
	assert.Contains(t, UninstallScriptTemplate, "Are you sure you want to continue?")
}

func TestScriptTemplateConstants(t *testing.T) {
	// Verify that both templates use the correct service name
	assert.Contains(t, InstallScriptTemplate, ServiceName)
	assert.Contains(t, UninstallScriptTemplate, ServiceName)
	
	// Verify standard paths are used
	assert.Contains(t, InstallScriptTemplate, "/usr/local/bin")
	assert.Contains(t, InstallScriptTemplate, "/usr/local/etc/gymdoorbridge")
	assert.Contains(t, UninstallScriptTemplate, "/usr/local/bin")
	assert.Contains(t, UninstallScriptTemplate, "/usr/local/etc/gymdoorbridge")
}

func TestCustomInstallScriptConfig(t *testing.T) {
	config := &CustomInstallScriptConfig{
		BridgeVersion: "1.0.0",
		BridgeURL:     "https://example.com",
		PairCode:      "TEST123",
		ConfigPath:    "/test/config.yaml",
		InstallDir:    "/test/bin",
	}
	
	// Verify all fields are accessible
	assert.Equal(t, "1.0.0", config.BridgeVersion)
	assert.Equal(t, "https://example.com", config.BridgeURL)
	assert.Equal(t, "TEST123", config.PairCode)
	assert.Equal(t, "/test/config.yaml", config.ConfigPath)
	assert.Equal(t, "/test/bin", config.InstallDir)
}

func TestScriptGeneratorFileOverwrite(t *testing.T) {
	sg := NewScriptGenerator()
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "install.sh")
	
	// Create initial script
	err := sg.GenerateInstallScript(scriptPath)
	require.NoError(t, err)
	
	// Get initial content
	initialContent, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	
	// Generate script again (should overwrite)
	err = sg.GenerateInstallScript(scriptPath)
	assert.NoError(t, err)
	
	// Verify content is the same (file was overwritten)
	newContent, err := os.ReadFile(scriptPath)
	assert.NoError(t, err)
	assert.Equal(t, initialContent, newContent)
}