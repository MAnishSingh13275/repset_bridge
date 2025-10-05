package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	ServiceName        = "GymDoorBridge"
	ServiceDisplayName = "Gym Door Access Bridge"
	ServiceDescription = "Connects gym door access hardware to SaaS platform. Runs automatically on startup and restarts on failure."
)

// installWindows performs Windows-specific installation
func (i *Installer) installWindows() error {
	// Check if running as administrator
	if !isAdmin() {
		return fmt.Errorf("installation requires administrator privileges. Please run as administrator")
	}

	// Check if already installed
	if installed, err := i.isInstalledWindows(); err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	} else if installed {
		return fmt.Errorf("service is already installed. Use 'gym-door-bridge uninstall' first")
	}

	// Create base installation
	if err := i.createBaseInstallation(); err != nil {
		return fmt.Errorf("failed to create base installation: %w", err)
	}

	// Install Windows service with auto-start and failure recovery
	targetExec := i.GetExecutablePath()
	if err := i.installWindowsService(targetExec); err != nil {
		return fmt.Errorf("failed to install Windows service: %w", err)
	}

	// Create registry entries
	if err := i.createWindowsRegistryEntries(); err != nil {
		i.logger.WithError(err).Warn("Failed to create registry entries")
	}

	// Start the service
	if err := i.startWindows(); err != nil {
		i.logger.WithError(err).Warn("Failed to start service automatically")
		i.logger.Info("You can start the service manually using 'gym-door-bridge start'")
	}

	i.ShowInstallationSummary()
	return nil
}

// uninstallWindows performs Windows-specific uninstallation
func (i *Installer) uninstallWindows() error {
	// Check if running as administrator
	if !isAdmin() {
		return fmt.Errorf("uninstallation requires administrator privileges. Please run as administrator")
	}

	// Stop service if running
	if err := i.stopWindows(); err != nil {
		i.logger.WithError(err).Warn("Failed to stop service")
	}

	// Remove Windows service
	if err := i.uninstallWindowsService(); err != nil {
		return fmt.Errorf("failed to uninstall Windows service: %w", err)
	}

	// Remove registry entries
	if err := i.removeWindowsRegistryEntries(); err != nil {
		i.logger.WithError(err).Warn("Failed to remove registry entries")
	}

	// Remove installation directory
	if err := os.RemoveAll(i.installPath); err != nil {
		i.logger.WithError(err).Warn("Failed to remove installation directory")
		i.logger.Info("You may need to manually remove: " + i.installPath)
	}

	i.logger.Info("✅ Uninstallation completed successfully!")
	return nil
}

// isInstalledWindows checks if the Windows service is installed
func (i *Installer) isInstalledWindows() (bool, error) {
	m, err := mgr.Connect()
	if err != nil {
		return false, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		// Service doesn't exist
		return false, nil
	}
	s.Close()
	return true, nil
}

// installWindowsService installs the Windows service with auto-start and failure recovery
func (i *Installer) installWindowsService(execPath string) error {
	i.logger.Info("Installing Windows service...")

	// Connect to service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Create service with enhanced configuration
	s, err := m.CreateService(ServiceName, execPath, mgr.Config{
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		StartType:        mgr.StartAutomatic,  // Auto-start on boot
		ServiceStartName: "LocalSystem",       // Run as LocalSystem
		Dependencies:     []string{"Tcpip", "Dnscache"}, // Network dependencies
	}, "--config", i.configPath)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Configure service failure recovery
	if err := i.configureServiceFailureRecovery(s); err != nil {
		i.logger.WithError(err).Warn("Failed to configure failure recovery (service will still work)")
	}

	// Install event log source
	if err := eventlog.InstallAsEventCreate(ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		i.logger.WithError(err).Warn("Failed to install event log source")
	}

	i.logger.Info("✅ Windows service installed successfully with auto-start and failure recovery")
	return nil
}

// uninstallWindowsService removes the Windows service
func (i *Installer) uninstallWindowsService() error {
	i.logger.Info("Removing Windows service...")

	// Connect to service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	defer s.Close()

	// Delete service
	if err := s.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Remove event log source
	if err := eventlog.Remove(ServiceName); err != nil {
		i.logger.WithError(err).Warn("Failed to remove event log source")
	}

	i.logger.Info("✅ Windows service removed successfully")
	return nil
}

// configureServiceFailureRecovery configures automatic restart on failure
func (i *Installer) configureServiceFailureRecovery(s *mgr.Service) error {
	// This is a complex operation that requires Win32 API calls
	// For now, we'll use sc.exe command as a simpler approach
	
	serviceName := ServiceName
	
	// Configure failure actions: restart after 1 min, restart after 2 min, restart after 5 min
	cmd := exec.Command("sc", "failure", serviceName, "reset=86400", "actions=restart/60000/restart/120000/restart/300000")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure failure recovery: %w", err)
	}
	
	// Set failure message
	msgCmd := exec.Command("sc", "failureflag", serviceName, "1")
	msgCmd.Run() // This might fail on older Windows versions, so don't return error
	
	i.logger.Info("✅ Service failure recovery configured: restart on failure with 1m, 2m, 5m delays")
	return nil
}

// startWindows starts the Windows service
func (i *Installer) startWindows() error {
	i.logger.Info("Starting Windows service...")

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Check current status
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}

	if status.State == svc.Running {
		i.logger.Info("✅ Service is already running")
		return nil
	}

	// Start service
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait for service to start
	if err := i.WaitForService("Running", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to start: %w", err)
	}

	i.logger.Info("✅ Windows service started successfully")
	return nil
}

// stopWindows stops the Windows service
func (i *Installer) stopWindows() error {
	i.logger.Info("Stopping Windows service...")

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Check current status
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}

	if status.State == svc.Stopped {
		i.logger.Info("✅ Service is already stopped")
		return nil
	}

	// Stop service
	_, err = s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait for service to stop
	if err := i.WaitForService("Stopped", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to stop: %w", err)
	}

	i.logger.Info("✅ Windows service stopped successfully")
	return nil
}

// statusWindows returns the Windows service status
func (i *Installer) statusWindows() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return "Not Installed", nil
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	switch status.State {
	case svc.Stopped:
		return "Stopped", nil
	case svc.StartPending:
		return "Starting", nil
	case svc.StopPending:
		return "Stopping", nil
	case svc.Running:
		return "Running", nil
	case svc.ContinuePending:
		return "Resuming", nil
	case svc.PausePending:
		return "Pausing", nil
	case svc.Paused:
		return "Paused", nil
	default:
		return fmt.Sprintf("Unknown (%d)", status.State), nil
	}
}

// createWindowsRegistryEntries creates Windows registry entries for service configuration
func (i *Installer) createWindowsRegistryEntries() error {
	i.logger.Info("Creating Windows registry entries...")

	// Create registry key for service configuration
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE,
		`SOFTWARE\GymDoorBridge`, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create registry key: %w", err)
	}
	defer key.Close()

	// Set configuration values
	values := map[string]string{
		"InstallPath":        i.installPath,
		"ConfigPath":         i.configPath,
		"ExecutablePath":     i.GetExecutablePath(),
		"LogLevel":           "info",
		"Version":            "1.0.0",
		"InstallDate":        time.Now().Format("2006-01-02 15:04:05"),
		"AutoStartEnabled":   "true",
		"FailureRecovery":    "true",
	}

	for name, value := range values {
		if err := key.SetStringValue(name, value); err != nil {
			return fmt.Errorf("failed to set registry value %s: %w", name, err)
		}
	}

	i.logger.Info("✅ Windows registry entries created successfully")
	return nil
}

// removeWindowsRegistryEntries removes Windows registry entries
func (i *Installer) removeWindowsRegistryEntries() error {
	i.logger.Info("Removing Windows registry entries...")

	// Delete registry key
	if err := registry.DeleteKey(registry.LOCAL_MACHINE, `SOFTWARE\GymDoorBridge`); err != nil {
		return fmt.Errorf("failed to delete registry key: %w", err)
	}

	i.logger.Info("✅ Windows registry entries removed successfully")
	return nil
}

// isAdmin checks if the current process is running with administrator privileges
func isAdmin() bool {
	// Try to open a handle to the service control manager with all access
	// This requires administrator privileges
	cmd := exec.Command("net", "session")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	return err == nil
}

// copyFile copies a file from src to dst (Windows-specific with proper error handling)
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file contents
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Ensure all data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// RestartService restarts the Windows service
func (i *Installer) RestartService() error {
	i.logger.Info("Restarting Windows service...")

	if err := i.stopWindows(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Small delay to ensure service is fully stopped
	time.Sleep(2 * time.Second)

	if err := i.startWindows(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	i.logger.Info("✅ Windows service restarted successfully")
	return nil
}

// GetServiceLogs returns recent service logs from Windows Event Log
func (i *Installer) GetServiceLogs() ([]string, error) {
	// This is a simplified implementation
	// In a full implementation, you would read from Windows Event Log
	logFile := filepath.Join(i.installPath, "logs", "bridge.log")
	
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []string{"No log file found"}, nil
	}

	// Read last 50 lines from log file
	cmd := exec.Command("powershell", "-Command", 
		fmt.Sprintf("Get-Content '%s' | Select-Object -Last 50", logFile))
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	
	// Filter out empty lines
	var filteredLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	return filteredLines, nil
}