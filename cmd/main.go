package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"gym-door-bridge/internal/bridge"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/installer"
	"gym-door-bridge/internal/logging"
	"gym-door-bridge/internal/service/windows"
	"gym-door-bridge/internal/service/macos"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/svc"
)

var (
	configFile string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:   "gym-door-bridge",
	Short: "Gym Door Access Bridge - Connect door hardware to SaaS platform",
	Long: `A lightweight local agent that connects gym door access hardware 
(fingerprint, RFID, or other devices) with our SaaS platform. The bridge 
normalizes check-in events from various hardware types into a standardized 
format and securely forwards them to our cloud system.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if running on Windows and as a service
		if runtime.GOOS == "windows" {
			isService, err := svc.IsWindowsService()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to determine if running as service: %v\n", err)
				os.Exit(1)
			}
			
			if isService {
				// Running as Windows service
				runAsWindowsService()
				return
			}
		}
		
		// Running as console application
		runAsConsole()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	
	// Add platform-specific service commands
	if runtime.GOOS == "windows" {
		windows.AddServiceCommands(rootCmd)
		addWindowsInstallerCommands()
	} else if runtime.GOOS == "darwin" {
		macos.AddServiceCommands(rootCmd)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runAsWindowsService runs the application as a Windows service
func runAsWindowsService() {
	// Load service configuration from registry
	serviceConfig, err := windows.LoadServiceConfig()
	if err != nil {
		// Log to Windows event log if possible
		fmt.Fprintf(os.Stderr, "Failed to load service configuration: %v\n", err)
		os.Exit(1)
	}
	
	// Override config file if specified in service config
	if serviceConfig.ConfigPath != "" {
		configFile = serviceConfig.ConfigPath
	}
	
	// Override log level if specified in service config
	if serviceConfig.LogLevel != "" {
		logLevel = serviceConfig.LogLevel
	}
	
	// Change working directory to service working directory
	if serviceConfig.WorkingDir != "" {
		if err := os.Chdir(serviceConfig.WorkingDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to change working directory: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Load application configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	
	// Run as Windows service
	err = windows.RunService(cfg, bridgeMain, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Service execution failed: %v\n", err)
		os.Exit(1)
	}
}

// runAsConsole runs the application as a console application
func runAsConsole() {
	// Initialize logging
	logger := logging.Initialize(logLevel)
	
	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}
	
	logger.WithField("config", cfg).Info("Bridge starting up")
	
	// Check platform-specific execution
	if runtime.GOOS == "windows" {
		// Run in debug mode on Windows (allows Ctrl+C handling)
		err = windows.RunService(cfg, bridgeMain, true)
		if err != nil {
			logger.WithError(err).Fatal("Failed to run bridge")
		}
	} else if runtime.GOOS == "darwin" && macos.IsMacOSDaemon() {
		// Running as macOS daemon
		err = macos.RunService(cfg, bridgeMain)
		if err != nil {
			logger.WithError(err).Fatal("Failed to run bridge as daemon")
		}
	} else {
		// Run directly on other platforms or when not running as daemon
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		
		// Handle interrupt signals for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		
		// Start bridge in goroutine
		errChan := make(chan error, 1)
		go func() {
			if err := bridgeMain(ctx, cfg); err != nil {
				errChan <- err
			}
		}()
		
		// Wait for signal or error
		select {
		case sig := <-sigChan:
			logger.WithField("signal", sig).Info("Received shutdown signal")
			cancel()
		case err := <-errChan:
			if err != nil {
				logger.WithError(err).Fatal("Bridge execution failed")
			}
		}
		
		logger.Info("Application shutdown complete")
	}
}

// bridgeMain is the main bridge execution function
func bridgeMain(ctx context.Context, cfg *config.Config) error {
	logger := logging.Initialize(logLevel)
	
	logger.WithField("config", cfg).Info("Bridge main function starting")
	
	// Create bridge manager with version and device ID
	manager, err := bridge.NewManager(cfg,
		bridge.WithVersion("1.0.0"), // TODO: Get from build info
		bridge.WithDeviceID(cfg.DeviceID),
	)
	if err != nil {
		logger.WithError(err).Error("Failed to create bridge manager")
		return fmt.Errorf("failed to create bridge manager: %w", err)
	}
	
	logger.Info("Gym Door Access Bridge initialized successfully")
	
	// Start the bridge manager
	if err := manager.Start(ctx); err != nil {
		logger.WithError(err).Error("Bridge manager stopped with error")
		return fmt.Errorf("bridge manager error: %w", err)
	}
	
	logger.Info("Bridge shutting down gracefully")
	return nil
}

// addWindowsInstallerCommands adds Windows installer commands
func addWindowsInstallerCommands() {
	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install Gym Door Bridge as Windows service with auto-discovery",
		Long: `Install the Gym Door Bridge as a Windows service. This command will:
- Automatically discover biometric devices on the network
- Generate configuration based on discovered devices
- Install the service to run automatically at startup
- Configure logging and database paths

Requires administrator privileges.`,
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := installer.NewWindowsInstaller()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create installer: %v\n", err)
				os.Exit(1)
			}

			if err := installer.Install(); err != nil {
				fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Installation completed successfully!")
			fmt.Println("The Gym Door Bridge service has been installed and started.")
			fmt.Println("Use 'gym-door-bridge pair' to connect to your platform.")
		},
	}

	var uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Gym Door Bridge Windows service",
		Long: `Uninstall the Gym Door Bridge Windows service. This command will:
- Stop the running service
- Remove the service from Windows
- Clean up installation files and registry entries

Requires administrator privileges.`,
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := installer.NewWindowsInstaller()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create installer: %v\n", err)
				os.Exit(1)
			}

			if err := installer.Uninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Uninstallation failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Uninstallation completed successfully!")
		},
	}

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}