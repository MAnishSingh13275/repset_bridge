package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/discovery"
	"gym-door-bridge/internal/logging"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"gopkg.in/yaml.v3"
)

const (
	ServiceName        = "GymDoorBridge"
	ServiceDisplayName = "Gym Door Access Bridge"
	ServiceDescription = "Connects gym door access hardware to SaaS platform"
)

// WindowsInstaller handles Windows service installation
type WindowsInstaller struct {
	logger      *logrus.Logger
	execPath    string
	installPath string
	configPath  string
}

// NewWindowsInstaller creates a new Windows installer
func NewWindowsInstaller() (*WindowsInstaller, error) {
	logger := logging.Initialize("info")
	
	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Default installation path
	installPath := filepath.Join(os.Getenv("ProgramFiles"), "GymDoorBridge")
	configPath := filepath.Join(installPath, "config.yaml")

	return &WindowsInstaller{
		logger:      logger,
		execPath:    execPath,
		installPath: installPath,
		configPath:  configPath,
	}, nil
}

// Install installs the service with auto-discovery
func (w *WindowsInstaller) Install() error {
	w.logger.Info("Starting Gym Door Bridge installation")

	// Check if running as administrator
	if !w.isAdmin() {
		return fmt.Errorf("installation requires administrator privileges")
	}

	// Create installation directory
	if err := os.MkdirAll(w.installPath, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Copy executable to installation directory
	targetExec := filepath.Join(w.installPath, "gym-door-bridge.exe")
	if err := w.copyFile(w.execPath, targetExec); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}

	// Run device discovery
	w.logger.Info("Discovering biometric devices...")
	deviceConfig, err := w.runDeviceDiscovery()
	if err != nil {
		w.logger.WithError(err).Warn("Device discovery failed, using default config")
		deviceConfig = w.getDefaultConfig()
	}

	// Generate configuration file
	if err := w.generateConfig(deviceConfig); err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}

	// Install Windows service
	if err := w.installService(targetExec); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	// Create registry entries for service configuration
	if err := w.createRegistryEntries(); err != nil {
		return fmt.Errorf("failed to create registry entries: %w", err)
	}

	// Start the service
	if err := w.startService(); err != nil {
		w.logger.WithError(err).Warn("Failed to start service automatically")
		w.logger.Info("You can start the service manually from Services.msc")
	}

	w.logger.Info("Installation completed successfully!")
	w.logger.Infof("Service installed at: %s", w.installPath)
	w.logger.Infof("Configuration file: %s", w.configPath)
	
	return nil
}

// Uninstall removes the service
func (w *WindowsInstaller) Uninstall() error {
	w.logger.Info("Uninstalling Gym Door Bridge service")

	// Check if running as administrator
	if !w.isAdmin() {
		return fmt.Errorf("uninstallation requires administrator privileges")
	}

	// Stop service if running
	if err := w.stopService(); err != nil {
		w.logger.WithError(err).Warn("Failed to stop service")
	}

	// Remove service
	if err := w.removeService(); err != nil {
		return fmt.Errorf("failed to remove service: %w", err)
	}

	// Remove registry entries
	if err := w.removeRegistryEntries(); err != nil {
		w.logger.WithError(err).Warn("Failed to remove registry entries")
	}

	// Remove installation directory
	if err := os.RemoveAll(w.installPath); err != nil {
		w.logger.WithError(err).Warn("Failed to remove installation directory")
	}

	w.logger.Info("Uninstallation completed successfully!")
	return nil
}

// runDeviceDiscovery runs device discovery and returns configuration
func (w *WindowsInstaller) runDeviceDiscovery() (map[string]interface{}, error) {
	w.logger.Info("Scanning network for biometric devices...")
	
	// Create device discovery instance
	discovery := discovery.NewDeviceDiscovery(w.logger)
	
	// Start discovery
	ctx := context.Background()
	if err := discovery.Start(ctx); err != nil {
		return nil, err
	}
	defer discovery.Stop()

	// Wait for discovery to complete
	time.Sleep(30 * time.Second)

	// Get discovered devices
	devices := discovery.GetDiscoveredDevices()
	if len(devices) == 0 {
		w.logger.Warn("No biometric devices discovered")
		return w.getDefaultConfig(), nil
	}

	w.logger.Infof("Discovered %d biometric device(s)", len(devices))
	for _, device := range devices {
		w.logger.Infof("Found %s device at %s:%d (Model: %s)", 
			device.Type, device.IP, device.Port, device.Model)
	}

	// Generate adapter configuration
	return discovery.GenerateAdapterConfig(), nil
}

// getDefaultConfig returns default configuration
func (w *WindowsInstaller) getDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled_adapters": []string{"simulator"},
		"adapter_configs": map[string]interface{}{
			"simulator": map[string]interface{}{
				"device_type":   "simulator",
				"connection":    "memory",
				"device_config": map[string]string{},
				"sync_interval": 10,
			},
		},
	}
}

// convertToMapStringMap converts map[string]interface{} to map[string]map[string]interface{}
func convertToMapStringMap(input map[string]interface{}) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})
	for key, value := range input {
		if mapValue, ok := value.(map[string]interface{}); ok {
			result[key] = mapValue
		}
	}
	return result
}

// generateConfig generates the configuration file
func (w *WindowsInstaller) generateConfig(deviceConfig map[string]interface{}) error {
	// Load base configuration
	baseConfig := &config.Config{
		ServerURL:    "https://your-platform.com", // Will be updated during pairing
		DeviceID:     "",                          // Will be set during pairing
		DeviceKey:    "",                          // Will be set during pairing
		DatabasePath: filepath.Join(w.installPath, "bridge.db"),
		LogLevel:     "info",
		LogFile:      filepath.Join(w.installPath, "logs", "bridge.log"),
		Tier:         "normal",
		
		// API Server configuration
		APIServer: config.APIServerConfig{
			Enabled: true,
			Host:    "localhost",
			Port:    8081,
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key", "X-Requested-With"},
			},
			RateLimit: config.RateLimitConfig{
				Enabled:        true,
				RequestsPerMin: 60,
				BurstSize:      10,
				WindowSize:     60,
				CleanupInterval: 300,
			},
		},
		
		// Device discovery configuration
		EnabledAdapters: deviceConfig["enabled_adapters"].([]string),
		AdapterConfigs:  convertToMapStringMap(deviceConfig["adapter_configs"].(map[string]interface{})),
		
		// Other settings
		HeartbeatInterval: 60,
		QueueMaxSize:     10000,
		UnlockDuration:   3000,
		UpdatesEnabled:   true,
	}

	// Create logs directory
	logsDir := filepath.Join(w.installPath, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(baseConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write configuration file
	if err := os.WriteFile(w.configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	w.logger.Infof("Configuration file created: %s", w.configPath)
	return nil
}

// installService installs the Windows service
func (w *WindowsInstaller) installService(execPath string) error {
	w.logger.Info("Installing Windows service...")

	// Connect to service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Check if service already exists
	s, err := m.OpenService(ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", ServiceName)
	}

	// Create service
	s, err = m.CreateService(ServiceName, execPath, mgr.Config{
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		StartType:        mgr.StartAutomatic,
		ServiceStartName: "LocalSystem",
		Dependencies:     []string{"Tcpip", "Dnscache"},
	}, "--config", w.configPath)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	w.logger.Info("Windows service installed successfully")
	return nil
}

// removeService removes the Windows service
func (w *WindowsInstaller) removeService() error {
	w.logger.Info("Removing Windows service...")

	// Connect to service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s not found: %w", ServiceName, err)
	}
	defer s.Close()

	// Delete service
	if err := s.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	w.logger.Info("Windows service removed successfully")
	return nil
}

// startService starts the Windows service
func (w *WindowsInstaller) startService() error {
	w.logger.Info("Starting Windows service...")

	// Connect to service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Start service
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	w.logger.Info("Windows service started successfully")
	return nil
}

// stopService stops the Windows service
func (w *WindowsInstaller) stopService() error {
	w.logger.Info("Stopping Windows service...")

	// Connect to service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// Stop service
	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait for service to stop
	timeout := time.Now().Add(30 * time.Second)
	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status: %w", err)
		}
	}

	w.logger.Info("Windows service stopped successfully")
	return nil
}

// createRegistryEntries creates registry entries for service configuration
func (w *WindowsInstaller) createRegistryEntries() error {
	w.logger.Info("Creating registry entries...")

	// Create registry key for service configuration
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, 
		`SOFTWARE\GymDoorBridge`, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create registry key: %w", err)
	}
	defer key.Close()

	// Set configuration values
	if err := key.SetStringValue("InstallPath", w.installPath); err != nil {
		return fmt.Errorf("failed to set InstallPath: %w", err)
	}

	if err := key.SetStringValue("ConfigPath", w.configPath); err != nil {
		return fmt.Errorf("failed to set ConfigPath: %w", err)
	}

	if err := key.SetStringValue("LogLevel", "info"); err != nil {
		return fmt.Errorf("failed to set LogLevel: %w", err)
	}

	w.logger.Info("Registry entries created successfully")
	return nil
}

// removeRegistryEntries removes registry entries
func (w *WindowsInstaller) removeRegistryEntries() error {
	w.logger.Info("Removing registry entries...")

	// Delete registry key
	if err := registry.DeleteKey(registry.LOCAL_MACHINE, `SOFTWARE\GymDoorBridge`); err != nil {
		return fmt.Errorf("failed to delete registry key: %w", err)
	}

	w.logger.Info("Registry entries removed successfully")
	return nil
}

// isAdmin checks if running with administrator privileges
func (w *WindowsInstaller) isAdmin() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

// copyFile copies a file from src to dst
func (w *WindowsInstaller) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}