package windows

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

// ServiceManager handles Windows service lifecycle operations
type ServiceManager struct {
	manager *mgr.Mgr
}

// NewServiceManager creates a new service manager instance
func NewServiceManager() (*ServiceManager, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	
	return &ServiceManager{manager: m}, nil
}

// Close closes the service manager connection
func (sm *ServiceManager) Close() error {
	if sm.manager != nil {
		return sm.manager.Disconnect()
	}
	return nil
}

// InstallService installs the bridge as a Windows service
func (sm *ServiceManager) InstallService(execPath string, configPath string) error {
	// Build service configuration
	config := mgr.Config{
		ServiceType:      windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:        windows.SERVICE_AUTO_START,
		ErrorControl:     windows.SERVICE_ERROR_NORMAL,
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		Dependencies:     []string{"Tcpip", "Dnscache"},
		ServiceStartName: "", // Use LocalSystem account
	}
	
	// Build command line arguments
	args := []string{execPath}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	
	// Create the service
	service, err := sm.manager.CreateService(ServiceName, execPath, config, args...)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer service.Close()
	
	// Install event log source
	if err := eventlog.InstallAsEventCreate(ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		// Log warning but don't fail installation
		fmt.Printf("Warning: Failed to install event log source: %v\n", err)
	}
	
	fmt.Printf("Service '%s' installed successfully\n", ServiceDisplayName)
	return nil
}

// UninstallService removes the bridge Windows service
func (sm *ServiceManager) UninstallService() error {
	// Stop the service first if it's running
	if err := sm.StopService(); err != nil {
		fmt.Printf("Warning: Failed to stop service before uninstall: %v\n", err)
	}
	
	// Open the service
	service, err := sm.manager.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()
	
	// Delete the service
	if err := service.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}
	
	// Remove event log source
	if err := eventlog.Remove(ServiceName); err != nil {
		fmt.Printf("Warning: Failed to remove event log source: %v\n", err)
	}
	
	fmt.Printf("Service '%s' uninstalled successfully\n", ServiceDisplayName)
	return nil
}

// StartService starts the bridge Windows service
func (sm *ServiceManager) StartService() error {
	service, err := sm.manager.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()
	
	// Check current status
	status, err := service.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}
	
	if status.State == svc.Running {
		fmt.Printf("Service '%s' is already running\n", ServiceDisplayName)
		return nil
	}
	
	// Start the service
	if err := service.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	
	// Wait for service to start with timeout
	timeout := time.Now().Add(30 * time.Second)
	for time.Now().Before(timeout) {
		status, err := service.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status: %w", err)
		}
		
		if status.State == svc.Running {
			fmt.Printf("Service '%s' started successfully\n", ServiceDisplayName)
			return nil
		}
		
		if status.State == svc.Stopped {
			return fmt.Errorf("service failed to start")
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for service to start")
}

// StopService stops the bridge Windows service
func (sm *ServiceManager) StopService() error {
	service, err := sm.manager.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()
	
	// Check current status
	status, err := service.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}
	
	if status.State == svc.Stopped {
		fmt.Printf("Service '%s' is already stopped\n", ServiceDisplayName)
		return nil
	}
	
	// Stop the service
	status, err = service.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	
	// Wait for service to stop with timeout
	timeout := time.Now().Add(30 * time.Second)
	for time.Now().Before(timeout) {
		status, err := service.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status: %w", err)
		}
		
		if status.State == svc.Stopped {
			fmt.Printf("Service '%s' stopped successfully\n", ServiceDisplayName)
			return nil
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for service to stop")
}

// RestartService restarts the bridge Windows service
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

// GetServiceStatus returns the current status of the bridge service
func (sm *ServiceManager) GetServiceStatus() (string, error) {
	service, err := sm.manager.OpenService(ServiceName)
	if err != nil {
		return "", fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()
	
	status, err := service.Query()
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

// IsServiceInstalled checks if the bridge service is installed
func (sm *ServiceManager) IsServiceInstalled() (bool, error) {
	service, err := sm.manager.OpenService(ServiceName)
	if err != nil {
		// Service doesn't exist
		return false, nil
	}
	defer service.Close()
	
	return true, nil
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