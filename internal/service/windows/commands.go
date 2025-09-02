package windows

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

// AddServiceCommands adds Windows service management commands to the root command
func AddServiceCommands(rootCmd *cobra.Command) {
	if runtime.GOOS != "windows" {
		return // Only add Windows service commands on Windows
	}
	
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Windows service management commands",
		Long:  "Manage the Gym Door Bridge as a Windows service",
	}
	
	// Install command
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install the bridge as a Windows service",
		Long:  "Install the Gym Door Bridge as a Windows service with automatic startup",
		RunE:  runInstallService,
	}
	
	// Uninstall command
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the bridge Windows service",
		Long:  "Remove the Gym Door Bridge Windows service",
		RunE:  runUninstallService,
	}
	
	// Start command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the bridge Windows service",
		Long:  "Start the Gym Door Bridge Windows service",
		RunE:  runStartService,
	}
	
	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the bridge Windows service",
		Long:  "Stop the Gym Door Bridge Windows service",
		RunE:  runStopService,
	}
	
	// Restart command
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the bridge Windows service",
		Long:  "Restart the Gym Door Bridge Windows service",
		RunE:  runRestartService,
	}
	
	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show bridge Windows service status",
		Long:  "Display the current status of the Gym Door Bridge Windows service",
		RunE:  runServiceStatus,
	}
	
	// Add flags for install command
	installCmd.Flags().String("config", "", "Path to configuration file")
	installCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	installCmd.Flags().String("data-dir", "", "Data directory path")
	installCmd.Flags().String("working-dir", "", "Service working directory")
	
	// Add commands to service command
	serviceCmd.AddCommand(installCmd, uninstallCmd, startCmd, stopCmd, restartCmd, statusCmd)
	
	// Add service command to root
	rootCmd.AddCommand(serviceCmd)
}

func runInstallService(cmd *cobra.Command, args []string) error {
	// Check if running as administrator
	if !isRunningAsAdmin() {
		return fmt.Errorf("installing Windows service requires administrator privileges")
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
	defer sm.Close()
	
	// Check if service is already installed
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		return fmt.Errorf("failed to check service installation status: %w", err)
	}
	
	if installed {
		return fmt.Errorf("service is already installed")
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
	
	// Validate configuration
	if err := ValidateServiceConfig(config); err != nil {
		return fmt.Errorf("invalid service configuration: %w", err)
	}
	
	// Create service directories
	if err := CreateServiceDirectories(config); err != nil {
		return fmt.Errorf("failed to create service directories: %w", err)
	}
	
	// Install the service
	if err := sm.InstallService(execPath, config.ConfigPath); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}
	
	// Save service configuration
	if err := SaveServiceConfig(config); err != nil {
		fmt.Printf("Warning: Failed to save service configuration: %v\n", err)
	}
	
	fmt.Printf("Service installed successfully!\n")
	fmt.Printf("Configuration file: %s\n", config.ConfigPath)
	fmt.Printf("Data directory: %s\n", config.DataDirectory)
	fmt.Printf("Working directory: %s\n", config.WorkingDir)
	fmt.Printf("\nUse 'gym-door-bridge service start' to start the service.\n")
	
	return nil
}

func runUninstallService(cmd *cobra.Command, args []string) error {
	// Check if running as administrator
	if !isRunningAsAdmin() {
		return fmt.Errorf("uninstalling Windows service requires administrator privileges")
	}
	
	// Create service manager
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	defer sm.Close()
	
	// Check if service is installed
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		return fmt.Errorf("failed to check service installation status: %w", err)
	}
	
	if !installed {
		return fmt.Errorf("service is not installed")
	}
	
	// Uninstall the service
	if err := sm.UninstallService(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}
	
	// Remove service configuration
	if err := RemoveServiceConfig(); err != nil {
		fmt.Printf("Warning: Failed to remove service configuration: %v\n", err)
	}
	
	return nil
}

func runStartService(cmd *cobra.Command, args []string) error {
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	defer sm.Close()
	
	return sm.StartService()
}

func runStopService(cmd *cobra.Command, args []string) error {
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	defer sm.Close()
	
	return sm.StopService()
}

func runRestartService(cmd *cobra.Command, args []string) error {
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	defer sm.Close()
	
	return sm.RestartService()
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	sm, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	defer sm.Close()
	
	// Check if service is installed
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		return fmt.Errorf("failed to check service installation status: %w", err)
	}
	
	if !installed {
		fmt.Printf("Service Status: Not Installed\n")
		return nil
	}
	
	// Get service status
	status, err := sm.GetServiceStatus()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}
	
	fmt.Printf("Service Name: %s\n", ServiceDisplayName)
	fmt.Printf("Service Status: %s\n", status)
	
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
	}
	
	return nil
}

// isRunningAsAdmin checks if the current process is running with administrator privileges
func isRunningAsAdmin() bool {
	// This is a simplified check - in a production environment, you might want
	// to use more robust privilege checking
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}