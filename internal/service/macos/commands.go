package macos

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// AddServiceCommands adds macOS daemon management commands to the root command
func AddServiceCommands(rootCmd *cobra.Command) {
	if runtime.GOOS != "darwin" {
		return // Only add macOS daemon commands on macOS
	}
	
	serviceCmd := &cobra.Command{
		Use:   "daemon",
		Short: "macOS daemon management commands",
		Long:  "Manage the Gym Door Bridge as a macOS daemon",
	}
	
	// Install command
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install the bridge as a macOS daemon",
		Long:  "Install the Gym Door Bridge as a macOS daemon with automatic startup",
		RunE:  runInstallService,
	}
	
	// Uninstall command
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the bridge macOS daemon",
		Long:  "Remove the Gym Door Bridge macOS daemon",
		RunE:  runUninstallService,
	}
	
	// Start command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the bridge macOS daemon",
		Long:  "Start the Gym Door Bridge macOS daemon",
		RunE:  runStartService,
	}
	
	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the bridge macOS daemon",
		Long:  "Stop the Gym Door Bridge macOS daemon",
		RunE:  runStopService,
	}
	
	// Restart command
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the bridge macOS daemon",
		Long:  "Restart the Gym Door Bridge macOS daemon",
		RunE:  runRestartService,
	}
	
	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show bridge macOS daemon status",
		Long:  "Display the current status of the Gym Door Bridge macOS daemon",
		RunE:  runServiceStatus,
	}
	
	// Add flags for install command
	installCmd.Flags().String("config", "", "Path to configuration file")
	installCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	installCmd.Flags().String("data-dir", "", "Data directory path")
	installCmd.Flags().String("working-dir", "", "Service working directory")
	installCmd.Flags().String("log-path", "", "Log file path")
	
	// Add commands to service command
	serviceCmd.AddCommand(installCmd, uninstallCmd, startCmd, stopCmd, restartCmd, statusCmd)
	
	// Add service command to root
	rootCmd.AddCommand(serviceCmd)
}

func runInstallService(cmd *cobra.Command, args []string) error {
	// Check if running as root
	if !isRunningAsRoot() {
		return fmt.Errorf("installing macOS daemon requires root privileges (use sudo)")
	}
	
	// Get executable path
	execPath, err := GetExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	
	// Create service manager
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Check if service is already installed
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		return fmt.Errorf("failed to check service installation status: %w", err)
	}
	
	if installed {
		return fmt.Errorf("daemon is already installed")
	}
	
	// Build service configuration
	config := DefaultServiceConfig()
	
	// Override with command line flags
	if configPath, _ := cmd.Flags().GetString("config"); configPath != "" {
		config.ConfigPath = configPath
	}
	
	if logLevel, _ := cmd.Flags().GetString("log-level"); logLevel != "" {
		config.LogLevel = logLevel
	}
	
	if dataDir, _ := cmd.Flags().GetString("data-dir"); dataDir != "" {
		config.DataDirectory = dataDir
	}
	
	if workingDir, _ := cmd.Flags().GetString("working-dir"); workingDir != "" {
		config.WorkingDir = workingDir
	}
	
	if logPath, _ := cmd.Flags().GetString("log-path"); logPath != "" {
		config.LogPath = logPath
	}
	
	// Validate configuration
	if err := ValidateServiceConfig(config); err != nil {
		return fmt.Errorf("invalid service configuration: %w", err)
	}
	
	// Create service directories
	if err := CreateServiceDirectories(config); err != nil {
		return fmt.Errorf("failed to create service directories: %w", err)
	}
	
	// Set directory permissions
	if err := SetDirectoryPermissions(config); err != nil {
		return fmt.Errorf("failed to set directory permissions: %w", err)
	}
	
	// Create default configuration file if it doesn't exist
	if err := CreateDefaultConfigFile(config.ConfigPath); err != nil {
		return fmt.Errorf("failed to create default config file: %w", err)
	}
	
	// Install the daemon
	if err := sm.InstallService(execPath, config.ConfigPath); err != nil {
		return fmt.Errorf("failed to install daemon: %w", err)
	}
	
	fmt.Printf("Daemon installed successfully!\n")
	fmt.Printf("Configuration file: %s\n", config.ConfigPath)
	fmt.Printf("Data directory: %s\n", config.DataDirectory)
	fmt.Printf("Working directory: %s\n", config.WorkingDir)
	fmt.Printf("Log file: %s\n", config.LogPath)
	fmt.Printf("\nThe daemon will start automatically on boot.\n")
	fmt.Printf("Use 'gym-door-bridge daemon start' to start the daemon now.\n")
	
	return nil
}

func runUninstallService(cmd *cobra.Command, args []string) error {
	// Check if running as root
	if !isRunningAsRoot() {
		return fmt.Errorf("uninstalling macOS daemon requires root privileges (use sudo)")
	}
	
	// Create service manager
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Check if service is installed
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		return fmt.Errorf("failed to check service installation status: %w", err)
	}
	
	if !installed {
		return fmt.Errorf("daemon is not installed")
	}
	
	// Uninstall the daemon
	if err := sm.UninstallService(); err != nil {
		return fmt.Errorf("failed to uninstall daemon: %w", err)
	}
	
	fmt.Printf("\nNote: Configuration files and data have been preserved.\n")
	fmt.Printf("To remove them manually:\n")
	fmt.Printf("  sudo rm -rf /usr/local/etc/gymdoorbridge\n")
	fmt.Printf("  sudo rm -rf /usr/local/var/lib/gymdoorbridge\n")
	fmt.Printf("  sudo rm -rf /usr/local/var/log/gymdoorbridge\n")
	
	return nil
}

func runStartService(cmd *cobra.Command, args []string) error {
	// Check if running as root
	if !isRunningAsRoot() {
		return fmt.Errorf("managing macOS daemon requires root privileges (use sudo)")
	}
	
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	return sm.StartService()
}

func runStopService(cmd *cobra.Command, args []string) error {
	// Check if running as root
	if !isRunningAsRoot() {
		return fmt.Errorf("managing macOS daemon requires root privileges (use sudo)")
	}
	
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	return sm.StopService()
}

func runRestartService(cmd *cobra.Command, args []string) error {
	// Check if running as root
	if !isRunningAsRoot() {
		return fmt.Errorf("managing macOS daemon requires root privileges (use sudo)")
	}
	
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	return sm.RestartService()
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Check if service is installed
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		return fmt.Errorf("failed to check service installation status: %w", err)
	}
	
	if !installed {
		fmt.Printf("Daemon Status: Not Installed\n")
		return nil
	}
	
	// Get service status
	status, err := sm.GetServiceStatus()
	if err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}
	
	fmt.Printf("Daemon Name: %s\n", ServiceDisplayName)
	fmt.Printf("Daemon Status: %s\n", status)
	
	// Load and display configuration
	config, err := LoadServiceConfig()
	if err != nil {
		fmt.Printf("Warning: Failed to load service configuration: %v\n", err)
	} else {
		fmt.Printf("\nConfiguration:\n")
		fmt.Printf("  Config File: %s\n", config.ConfigPath)
		fmt.Printf("  Log Level: %s\n", config.LogLevel)
		fmt.Printf("  Data Directory: %s\n", config.DataDirectory)
		fmt.Printf("  Working Directory: %s\n", config.WorkingDir)
		fmt.Printf("  Log File: %s\n", config.LogPath)
	}
	
	return nil
}

// isRunningAsRoot checks if the current process is running with root privileges
func isRunningAsRoot() bool {
	return os.Geteuid() == 0
}