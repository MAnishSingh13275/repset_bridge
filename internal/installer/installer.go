package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"

	"github.com/sirupsen/logrus"
)

// Installer provides cross-platform installation functionality
type Installer struct {
	logger      *logrus.Logger
	execPath    string
	installPath string
	configPath  string
	platform    string
}

// NewInstaller creates a new cross-platform installer
func NewInstaller() (*Installer, error) {
	logger := logging.Initialize("info")

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get platform-specific install path
	installPath, err := getInstallPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine install path: %w", err)
	}

	configPath := filepath.Join(installPath, "config.yaml")

	return &Installer{
		logger:      logger,
		execPath:    execPath,
		installPath: installPath,
		configPath:  configPath,
		platform:    runtime.GOOS,
	}, nil
}

// Install performs one-click installation and setup
func (i *Installer) Install() error {
	i.logger.Info("Starting Gym Door Bridge installation...")

	switch i.platform {
	case "windows":
		return i.installWindows()
	case "linux":
		return i.installLinux()
	case "darwin":
		return i.installMacOS()
	default:
		return fmt.Errorf("unsupported platform: %s", i.platform)
	}
}

// Uninstall removes the bridge from the system
func (i *Installer) Uninstall() error {
	i.logger.Info("Uninstalling Gym Door Bridge...")

	switch i.platform {
	case "windows":
		return i.uninstallWindows()
	case "linux":
		return i.uninstallLinux()
	case "darwin":
		return i.uninstallMacOS()
	default:
		return fmt.Errorf("unsupported platform: %s", i.platform)
	}
}

// IsInstalled checks if the bridge is already installed
func (i *Installer) IsInstalled() (bool, error) {
	switch i.platform {
	case "windows":
		return i.isInstalledWindows()
	case "linux":
		return i.isInstalledLinux()
	case "darwin":
		return i.isInstalledMacOS()
	default:
		return false, fmt.Errorf("unsupported platform: %s", i.platform)
	}
}

// Start starts the background service
func (i *Installer) Start() error {
	switch i.platform {
	case "windows":
		return i.startWindows()
	case "linux":
		return i.startLinux()
	case "darwin":
		return i.startMacOS()
	default:
		return fmt.Errorf("unsupported platform: %s", i.platform)
	}
}

// Stop stops the background service
func (i *Installer) Stop() error {
	switch i.platform {
	case "windows":
		return i.stopWindows()
	case "linux":
		return i.stopLinux()
	case "darwin":
		return i.stopMacOS()
	default:
		return fmt.Errorf("unsupported platform: %s", i.platform)
	}
}

// Status returns the current status of the background service
func (i *Installer) Status() (string, error) {
	switch i.platform {
	case "windows":
		return i.statusWindows()
	case "linux":
		return i.statusLinux()
	case "darwin":
		return i.statusMacOS()
	default:
		return "", fmt.Errorf("unsupported platform: %s", i.platform)
	}
}

// createBaseInstallation creates the basic installation files and directories
func (i *Installer) createBaseInstallation() error {
	i.logger.Infof("Creating installation directory: %s", i.installPath)

	// Create installation directory
	if err := os.MkdirAll(i.installPath, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	// Create logs directory
	logsDir := filepath.Join(i.installPath, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create data directory
	dataDir := filepath.Join(i.installPath, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Copy executable to installation directory
	execName := "gym-door-bridge"
	if i.platform == "windows" {
		execName += ".exe"
	}
	targetExec := filepath.Join(i.installPath, execName)

	if err := copyFile(i.execPath, targetExec); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}

	// Make executable on Unix systems
	if i.platform != "windows" {
		if err := os.Chmod(targetExec, 0755); err != nil {
			return fmt.Errorf("failed to make executable: %w", err)
		}
	}

	// Generate default configuration
	if err := i.generateDefaultConfig(); err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}

	i.logger.Info("Base installation created successfully")
	return nil
}

// generateDefaultConfig creates a default configuration file
func (i *Installer) generateDefaultConfig() error {
	cfg := &config.Config{
		// Device configuration (will be set during pairing)
		DeviceID:  "",
		DeviceKey: "",

		// Server configuration
		ServerURL: "https://repset.onezy.in",

		// Performance tier
		Tier: "normal",

		// Queue configuration
		QueueMaxSize:     10000,
		HeartbeatInterval: 60,

		// Door control configuration
		UnlockDuration: 3000,

		// Database configuration
		DatabasePath: filepath.Join(i.installPath, "data", "bridge.db"),

		// Logging configuration
		LogLevel: "info",
		LogFile:  filepath.Join(i.installPath, "logs", "bridge.log"),

		// Adapter configuration (default to simulator for initial setup)
		EnabledAdapters: []string{"simulator"},
		AdapterConfigs: map[string]map[string]interface{}{
			"simulator": {
				"device_type":   "simulator",
				"connection":    "memory",
				"device_config": map[string]string{},
				"sync_interval": 10,
			},
		},
	}

	// Set installation metadata
	cfg.SetInstallationMethod("installer", "admin", "", "automatic", "1.0.0")

	// Save configuration
	if err := cfg.Save(i.configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	i.logger.Infof("Configuration file created: %s", i.configPath)
	return nil
}

// getInstallPath returns the appropriate installation path for the platform
func getInstallPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = "C:\\Program Files"
		}
		return filepath.Join(programFiles, "GymDoorBridge"), nil
	case "linux":
		return "/opt/gym-door-bridge", nil
	case "darwin":
		return "/Applications/GymDoorBridge", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// GetExecutablePath returns the target executable path
func (i *Installer) GetExecutablePath() string {
	execName := "gym-door-bridge"
	if i.platform == "windows" {
		execName += ".exe"
	}
	return filepath.Join(i.installPath, execName)
}

// WaitForService waits for service to reach the desired state
func (i *Installer) WaitForService(desiredState string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		status, err := i.Status()
		if err != nil {
			return fmt.Errorf("failed to check service status: %w", err)
		}
		
		if status == desiredState {
			return nil
		}
		
		time.Sleep(1 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for service to reach state: %s", desiredState)
}

// ShowInstallationSummary displays installation summary
func (i *Installer) ShowInstallationSummary() {
	i.logger.Info("🎉 Installation completed successfully!")
	i.logger.Infof("📁 Installation path: %s", i.installPath)
	i.logger.Infof("⚙️  Configuration: %s", i.configPath)
	i.logger.Infof("📊 Status: Run 'gym-door-bridge status' to check service status")
	i.logger.Info("🔗 Next steps:")
	i.logger.Info("   1. Get a pairing code from your gym management platform")
	i.logger.Info("   2. Run: gym-door-bridge pair YOUR_PAIR_CODE")
	i.logger.Info("   3. The service will automatically discover and configure devices")
}