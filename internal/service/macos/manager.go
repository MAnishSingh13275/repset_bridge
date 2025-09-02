package macos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ServiceManager handles macOS daemon lifecycle operations
type ServiceManager struct {
	plistPath string
}

// NewServiceManager creates a new service manager instance
func NewServiceManager() (*ServiceManager, error) {
	// Use system-wide LaunchDaemons directory
	plistPath := filepath.Join("/Library/LaunchDaemons", ServiceName+".plist")

	return &ServiceManager{
		plistPath: plistPath,
	}, nil
}

// InstallService installs the bridge as a macOS daemon
func (sm *ServiceManager) InstallService(execPath string, configPath string) error {
	// Get absolute paths
	absExecPath, err := filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute executable path: %w", err)
	}

	absConfigPath := ""
	if configPath != "" {
		absConfigPath, err = filepath.Abs(configPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute config path: %w", err)
		}
	}

	// Generate plist content
	plistContent, err := sm.generatePlistContent(absExecPath, absConfigPath)
	if err != nil {
		return fmt.Errorf("failed to generate plist content: %w", err)
	}

	// Write plist file
	if err := os.WriteFile(sm.plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the daemon
	if err := sm.loadDaemon(); err != nil {
		// Clean up plist file if load fails
		os.Remove(sm.plistPath)
		return fmt.Errorf("failed to load daemon: %w", err)
	}

	fmt.Printf("Service '%s' installed successfully\n", ServiceDisplayName)
	fmt.Printf("Plist file: %s\n", sm.plistPath)
	return nil
}

// UninstallService removes the bridge macOS daemon
func (sm *ServiceManager) UninstallService() error {
	// Stop the daemon first if it's running
	if err := sm.StopService(); err != nil {
		fmt.Printf("Warning: Failed to stop service before uninstall: %v\n", err)
	}

	// Unload the daemon
	if err := sm.unloadDaemon(); err != nil {
		fmt.Printf("Warning: Failed to unload daemon: %v\n", err)
	}

	// Remove plist file
	if err := os.Remove(sm.plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	fmt.Printf("Service '%s' uninstalled successfully\n", ServiceDisplayName)
	return nil
}

// StartService starts the bridge macOS daemon
func (sm *ServiceManager) StartService() error {
	// Check if daemon is loaded
	loaded, err := sm.isDaemonLoaded()
	if err != nil {
		return fmt.Errorf("failed to check daemon status: %w", err)
	}

	if !loaded {
		return fmt.Errorf("daemon is not installed")
	}

	// Start the daemon
	cmd := exec.Command("launchctl", "start", ServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start daemon: %w (output: %s)", err, string(output))
	}

	// Wait for daemon to start with timeout
	timeout := time.Now().Add(30 * time.Second)
	for time.Now().Before(timeout) {
		if running, _ := sm.isDaemonRunning(); running {
			fmt.Printf("Service '%s' started successfully\n", ServiceDisplayName)
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for daemon to start")
}

// StopService stops the bridge macOS daemon
func (sm *ServiceManager) StopService() error {
	// Check if daemon is loaded
	loaded, err := sm.isDaemonLoaded()
	if err != nil {
		return fmt.Errorf("failed to check daemon status: %w", err)
	}

	if !loaded {
		fmt.Printf("Service '%s' is not installed\n", ServiceDisplayName)
		return nil
	}

	// Check if daemon is running
	running, err := sm.isDaemonRunning()
	if err != nil {
		return fmt.Errorf("failed to check daemon running status: %w", err)
	}

	if !running {
		fmt.Printf("Service '%s' is already stopped\n", ServiceDisplayName)
		return nil
	}

	// Stop the daemon
	cmd := exec.Command("launchctl", "stop", ServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop daemon: %w (output: %s)", err, string(output))
	}

	// Wait for daemon to stop with timeout
	timeout := time.Now().Add(30 * time.Second)
	for time.Now().Before(timeout) {
		if running, _ := sm.isDaemonRunning(); !running {
			fmt.Printf("Service '%s' stopped successfully\n", ServiceDisplayName)
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for daemon to stop")
}

// RestartService restarts the bridge macOS daemon
func (sm *ServiceManager) RestartService() error {
	if err := sm.StopService(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Small delay to ensure service is fully stopped
	time.Sleep(2 * time.Second)

	if err := sm.StartService(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// GetServiceStatus returns the current status of the bridge daemon
func (sm *ServiceManager) GetServiceStatus() (string, error) {
	loaded, err := sm.isDaemonLoaded()
	if err != nil {
		return "", fmt.Errorf("failed to check daemon loaded status: %w", err)
	}

	if !loaded {
		return "Not Installed", nil
	}

	running, err := sm.isDaemonRunning()
	if err != nil {
		return "", fmt.Errorf("failed to check daemon running status: %w", err)
	}

	if running {
		return "Running", nil
	}

	return "Stopped", nil
}

// IsServiceInstalled checks if the bridge daemon is installed
func (sm *ServiceManager) IsServiceInstalled() (bool, error) {
	_, err := os.Stat(sm.plistPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check plist file: %w", err)
	}

	return true, nil
}

// generatePlistContent generates the launchd plist content
func (sm *ServiceManager) generatePlistContent(execPath, configPath string) (string, error) {
	// Build program arguments
	args := []string{execPath}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	// Generate XML for program arguments
	programArgsXML := ""
	for _, arg := range args {
		programArgsXML += fmt.Sprintf("\t\t<string>%s</string>\n", arg)
	}

	// Get service directories
	serviceConfig := DefaultServiceConfig()
	logDir := filepath.Dir(serviceConfig.LogPath)
	
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
%s	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>UserName</key>
	<string>root</string>
	<key>GroupName</key>
	<string>wheel</string>
	<key>ProcessType</key>
	<string>Background</string>
	<key>ThrottleInterval</key>
	<integer>10</integer>
</dict>
</plist>`,
		ServiceName,
		programArgsXML,
		serviceConfig.LogPath,
		serviceConfig.LogPath,
		serviceConfig.WorkingDir,
	)

	return plistContent, nil
}

// loadDaemon loads the daemon using launchctl
func (sm *ServiceManager) loadDaemon() error {
	cmd := exec.Command("launchctl", "load", sm.plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// unloadDaemon unloads the daemon using launchctl
func (sm *ServiceManager) unloadDaemon() error {
	cmd := exec.Command("launchctl", "unload", sm.plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Don't treat "service not loaded" as an error
		if !strings.Contains(string(output), "not currently loaded") {
			return fmt.Errorf("launchctl unload failed: %w (output: %s)", err, string(output))
		}
	}
	return nil
}

// isDaemonLoaded checks if the daemon is loaded in launchctl
func (sm *ServiceManager) isDaemonLoaded() (bool, error) {
	cmd := exec.Command("launchctl", "list", ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the command fails, the daemon is not loaded
		return false, nil
	}

	// If we get output, the daemon is loaded
	return len(output) > 0, nil
}

// isDaemonRunning checks if the daemon process is currently running
func (sm *ServiceManager) isDaemonRunning() (bool, error) {
	cmd := exec.Command("launchctl", "list", ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}

	// Parse the output to check if PID is present (indicating running)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ServiceName) {
			fields := strings.Fields(line)
			if len(fields) >= 1 && fields[0] != "-" {
				// PID is present, daemon is running
				return true, nil
			}
		}
	}

	return false, nil
}

// GetExecutablePath returns the current executable path
func GetExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve any symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	return execPath, nil
}