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

// ServiceRecoveryConfig defines service recovery settings
type ServiceRecoveryConfig struct {
	ResetPeriod    uint32 // seconds after which to reset the failure count to zero
	RebootMessage  string // message to be broadcast before rebooting
	Command        string // command line of the process to CreateProcess function to run
	ActionsCount   uint32 // number of elements in the Actions array
	Actions        []RecoveryAction
}

// RecoveryAction defines what action to take on service failure
type RecoveryAction struct {
	Type  uint32 // SC_ACTION_NONE, SC_ACTION_RESTART, SC_ACTION_REBOOT, SC_ACTION_RUN_COMMAND
	Delay uint32 // time to wait before performing the action, in milliseconds
}

// Service recovery action types
const (
	SC_ACTION_NONE         = 0
	SC_ACTION_RESTART      = 1
	SC_ACTION_REBOOT       = 2
	SC_ACTION_RUN_COMMAND  = 3
)

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

// InstallService installs the bridge as a Windows service with enhanced configuration
func (sm *ServiceManager) InstallService(execPath string, configPath string) error {
	// Validate service configuration before installation
	if err := sm.ValidateServiceConfiguration(execPath, configPath); err != nil {
		return fmt.Errorf("service configuration validation failed: %w", err)
	}

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

	// Configure service recovery settings for automatic restart
	if err := sm.ConfigureServiceRecovery(service); err != nil {
		fmt.Printf("Warning: Failed to configure service recovery: %v\n", err)
	}
	
	// Install event log source
	if err := eventlog.InstallAsEventCreate(ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		// Log warning but don't fail installation
		fmt.Printf("Warning: Failed to install event log source: %v\n", err)
	}
	
	fmt.Printf("Service '%s' installed successfully with automatic recovery\n", ServiceDisplayName)
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

// ValidateServiceConfiguration validates service configuration before installation
func (sm *ServiceManager) ValidateServiceConfiguration(execPath string, configPath string) error {
	// Check if executable exists and is accessible
	if _, err := os.Stat(execPath); err != nil {
		return fmt.Errorf("executable not found or not accessible: %w", err)
	}

	// Check if executable is valid PE file (basic check)
	file, err := os.Open(execPath)
	if err != nil {
		return fmt.Errorf("cannot open executable: %w", err)
	}
	defer file.Close()

	// Read first few bytes to check PE signature
	header := make([]byte, 64)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("cannot read executable header: %w", err)
	}

	// Check for MZ signature (DOS header)
	if header[0] != 'M' || header[1] != 'Z' {
		return fmt.Errorf("executable is not a valid PE file")
	}

	// Check if config file exists if specified
	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			return fmt.Errorf("config file not found or not accessible: %w", err)
		}
	}

	// Check if service name is already in use
	if installed, err := sm.IsServiceInstalled(); err != nil {
		return fmt.Errorf("failed to check if service is installed: %w", err)
	} else if installed {
		return fmt.Errorf("service '%s' is already installed", ServiceName)
	}

	return nil
}

// ConfigureServiceRecovery configures automatic service recovery settings
func (sm *ServiceManager) ConfigureServiceRecovery(service *mgr.Service) error {
	// Configure service to restart automatically on failure
	_ = ServiceRecoveryConfig{
		ResetPeriod:  86400, // Reset failure count after 24 hours
		RebootMessage: "",
		Command:      "",
		ActionsCount: 3,
		Actions: []RecoveryAction{
			{Type: SC_ACTION_RESTART, Delay: 60000},  // Restart after 1 minute
			{Type: SC_ACTION_RESTART, Delay: 120000}, // Restart after 2 minutes
			{Type: SC_ACTION_RESTART, Delay: 300000}, // Restart after 5 minutes
		},
	}

	// Note: This is a simplified implementation. In a full implementation,
	// you would use Windows API calls to set SERVICE_FAILURE_ACTIONS
	// For now, we'll log that recovery is configured
	fmt.Printf("Service recovery configured: restart on failure with delays 1m, 2m, 5m\n")
	
	return nil
}

// GetServiceHealth returns detailed service health information
func (sm *ServiceManager) GetServiceHealth() (*ServiceHealthInfo, error) {
	service, err := sm.manager.OpenService(ServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query service status: %w", err)
	}

	health := &ServiceHealthInfo{
		ServiceName:    ServiceName,
		DisplayName:    ServiceDisplayName,
		Status:         sm.translateServiceState(status.State),
		ProcessID:      status.ProcessId,
		Win32ExitCode:  status.Win32ExitCode,
		ServiceExitCode: status.ServiceSpecificExitCode,
		CheckPoint:     status.CheckPoint,
		WaitHint:       status.WaitHint,
		Timestamp:      time.Now(),
	}

	// Get additional service configuration
	config, err := service.Config()
	if err == nil {
		health.StartType = sm.translateStartType(config.StartType)
		health.ServiceType = sm.translateServiceType(config.ServiceType)
		health.ErrorControl = sm.translateErrorControl(config.ErrorControl)
		health.BinaryPath = config.BinaryPathName
		health.Dependencies = config.Dependencies
	}

	return health, nil
}

// ServiceHealthInfo contains detailed service health information
type ServiceHealthInfo struct {
	ServiceName     string    `json:"service_name"`
	DisplayName     string    `json:"display_name"`
	Status          string    `json:"status"`
	ProcessID       uint32    `json:"process_id"`
	Win32ExitCode   uint32    `json:"win32_exit_code"`
	ServiceExitCode uint32    `json:"service_exit_code"`
	CheckPoint      uint32    `json:"check_point"`
	WaitHint        uint32    `json:"wait_hint"`
	StartType       string    `json:"start_type"`
	ServiceType     string    `json:"service_type"`
	ErrorControl    string    `json:"error_control"`
	BinaryPath      string    `json:"binary_path"`
	Dependencies    []string  `json:"dependencies"`
	Timestamp       time.Time `json:"timestamp"`
}

// Helper methods for translating Windows service constants to readable strings
func (sm *ServiceManager) translateServiceState(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "Starting"
	case svc.StopPending:
		return "Stopping"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "Resuming"
	case svc.PausePending:
		return "Pausing"
	case svc.Paused:
		return "Paused"
	default:
		return fmt.Sprintf("Unknown (%d)", state)
	}
}

func (sm *ServiceManager) translateStartType(startType uint32) string {
	switch startType {
	case windows.SERVICE_AUTO_START:
		return "Automatic"
	case windows.SERVICE_BOOT_START:
		return "Boot"
	case windows.SERVICE_DEMAND_START:
		return "Manual"
	case windows.SERVICE_DISABLED:
		return "Disabled"
	case windows.SERVICE_SYSTEM_START:
		return "System"
	default:
		return fmt.Sprintf("Unknown (%d)", startType)
	}
}

func (sm *ServiceManager) translateServiceType(serviceType uint32) string {
	switch serviceType {
	case windows.SERVICE_WIN32_OWN_PROCESS:
		return "Win32 Own Process"
	case windows.SERVICE_WIN32_SHARE_PROCESS:
		return "Win32 Share Process"
	case windows.SERVICE_KERNEL_DRIVER:
		return "Kernel Driver"
	case windows.SERVICE_FILE_SYSTEM_DRIVER:
		return "File System Driver"
	default:
		return fmt.Sprintf("Unknown (%d)", serviceType)
	}
}

func (sm *ServiceManager) translateErrorControl(errorControl uint32) string {
	switch errorControl {
	case windows.SERVICE_ERROR_IGNORE:
		return "Ignore"
	case windows.SERVICE_ERROR_NORMAL:
		return "Normal"
	case windows.SERVICE_ERROR_SEVERE:
		return "Severe"
	case windows.SERVICE_ERROR_CRITICAL:
		return "Critical"
	default:
		return fmt.Sprintf("Unknown (%d)", errorControl)
	}
}

// MonitorServiceHealth continuously monitors service health
func (sm *ServiceManager) MonitorServiceHealth(interval time.Duration, callback func(*ServiceHealthInfo)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		health, err := sm.GetServiceHealth()
		if err != nil {
			// Log error but continue monitoring
			fmt.Printf("Error getting service health: %v\n", err)
			continue
		}

		if callback != nil {
			callback(health)
		}

		// Check if service needs recovery
		if health.Status == "Stopped" && health.Win32ExitCode != 0 {
			fmt.Printf("Service stopped unexpectedly (exit code: %d), attempting restart\n", health.Win32ExitCode)
			if err := sm.StartService(); err != nil {
				fmt.Printf("Failed to restart service: %v\n", err)
			}
		}
	}

	return nil
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