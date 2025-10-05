package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/bridge"
	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
	"gym-door-bridge/internal/pairing"
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
	} else if runtime.GOOS == "darwin" {
		macos.AddServiceCommands(rootCmd)
	}
	
	// Add cross-platform installer commands
	// TODO: Re-enable after fixing installer compilation issues
	// addCrossPlatformInstallerCommands()
	
	// Add pairing commands
	addPairingCommands()
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
	
	// For Windows service, try to populate device credentials from auth manager if missing from config
	if cfg.DeviceID == "" || cfg.DeviceKey == "" {
		if authManager, err := auth.NewAuthManager(); err == nil {
			if err := authManager.Initialize(); err == nil {
				if authManager.IsAuthenticated() {
					if deviceID, deviceKey, err := authManager.GetCredentials(); err == nil {
						if cfg.DeviceID == "" {
							cfg.DeviceID = deviceID
						}
						if cfg.DeviceKey == "" {
							cfg.DeviceKey = deviceKey
						}
					}
				}
			}
		}
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
	
	// Try to populate device credentials from auth manager if missing from config
	if cfg.DeviceID == "" || cfg.DeviceKey == "" {
		if authManager, err := auth.NewAuthManager(); err == nil {
			if err := authManager.Initialize(); err == nil {
				if authManager.IsAuthenticated() {
					if deviceID, deviceKey, err := authManager.GetCredentials(); err == nil {
						if cfg.DeviceID == "" {
							cfg.DeviceID = deviceID
						}
						if cfg.DeviceKey == "" {
							cfg.DeviceKey = deviceKey
						}
					}
				}
			}
		}
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

// addCrossPlatformInstallerCommands adds cross-platform installer commands
// TODO: Re-enable when installer compilation issues are fixed
/*
func addCrossPlatformInstallerCommands() {
	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install Gym Door Bridge as a background service with auto-discovery",
		Long: `Install the Gym Door Bridge as a background service. This command will:
- Automatically discover biometric devices on the network
- Generate configuration based on discovered devices
- Install the service to run automatically at startup
- Configure automatic restart on failure
- Set up logging and database paths

Requires administrator privileges on Windows or sudo on macOS.`,
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := installer.NewInstaller()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create installer: %v\n", err)
				os.Exit(1)
			}

			if err := installer.Install(); err != nil {
				fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("\n🎉 Installation completed successfully!")
			fmt.Println("📋 Next steps:")
			fmt.Println("   1. Get a pairing code from your gym management platform")
			fmt.Println("   2. Run: gym-door-bridge pair YOUR_PAIR_CODE")
			fmt.Println("   3. The service will run automatically in the background")
		},
	}

	var uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Gym Door Bridge background service",
		Long: `Uninstall the Gym Door Bridge background service. This command will:
- Stop the running service
- Remove the service from the system
- Clean up installation files and system entries

Requires administrator privileges on Windows or sudo on macOS.`,
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := installer.NewInstaller()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create installer: %v\n", err)
				os.Exit(1)
			}

			if err := installer.Uninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Uninstallation failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Uninstallation completed successfully!")
		},
	}

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the Gym Door Bridge background service",
		Long:  `Start the Gym Door Bridge background service.`,
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := installer.NewInstaller()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create installer: %v\n", err)
				os.Exit(1)
			}

			if err := installer.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to start service: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Service started successfully!")
		},
	}

	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the Gym Door Bridge background service",
		Long:  `Stop the Gym Door Bridge background service.`,
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := installer.NewInstaller()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create installer: %v\n", err)
				os.Exit(1)
			}

			if err := installer.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to stop service: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Service stopped successfully!")
		},
	}

	var restartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart the Gym Door Bridge background service",
		Long:  `Restart the Gym Door Bridge background service.`,
		Run: func(cmd *cobra.Command, args []string) {
			installer, err := installer.NewInstaller()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create installer: %v\n", err)
				os.Exit(1)
			}

			// Use the RestartService method if available, otherwise stop and start
			if restarter, ok := installer.(interface{ RestartService() error }); ok {
				if err := restarter.RestartService(); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to restart service: %v\n", err)
					os.Exit(1)
				}
			} else {
				if err := installer.Stop(); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to stop service: %v\n", err)
					os.Exit(1)
				}
				time.Sleep(2 * time.Second)
				if err := installer.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to start service: %v\n", err)
					os.Exit(1)
				}
			}

			fmt.Println("✅ Service restarted successfully!")
		},
	}

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
}
*/

// addPairingCommands adds pairing-related commands
func addPairingCommands() {
	var pairCmd = &cobra.Command{
		Use:   "pair [pair-code]",
		Short: "Pair the bridge with your gym management platform",
		Long: `Pair the bridge with your gym management platform using a pair code.
This establishes a secure connection between the local bridge and your cloud platform.

Example:
  gym-door-bridge pair ABC123DEF456

The pair code is provided by your gym management platform.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pairCode := args[0]
			
			// Load configuration
			cfg, err := config.Load(configFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
				os.Exit(1)
			}
			
			// Initialize components for pairing
			logger := logging.Initialize(logLevel)
			
			// Create auth manager
			authManager, err := auth.NewAuthManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create auth manager: %v\n", err)
				os.Exit(1)
			}
			if err := authManager.Initialize(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize auth manager: %v\n", err)
				os.Exit(1)
			}
			
			// Create HTTP client
			httpClient, err := client.NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create HTTP client: %v\n", err)
				os.Exit(1)
			}
			
			// Create pairing manager
			pairingManager, err := pairing.NewPairingManager(httpClient, authManager, cfg, logger)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create pairing manager: %v\n", err)
				os.Exit(1)
			}
			
			// Perform pairing
			ctx := context.Background()
			pairResp, err := pairingManager.PairDevice(ctx, pairCode)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Pairing failed: %v\n", err)
				os.Exit(1)
			}
			
			// Update configuration with device credentials
			cfg.DeviceID = pairResp.DeviceID
			cfg.DeviceKey = pairResp.DeviceKey
			
			// Update installation metadata
			cfg.SetInstallationMethod("paired", "user", pairCode, "manual", "")
			
			// Save updated configuration
			if err := cfg.Save(configFile); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save configuration: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("✅ Bridge paired successfully!\n")
			fmt.Printf("Device ID: %s\n", pairResp.DeviceID)
			fmt.Printf("Connected to: %s\n", cfg.ServerURL)
			
			// Restart service if running on Windows
			if runtime.GOOS == "windows" {
				fmt.Println("\n🔄 Restarting service to apply new configuration...")
				if err := restartWindowsService(); err != nil {
					fmt.Printf("⚠️  Failed to restart service automatically: %v\n", err)
					fmt.Println("Please restart the 'GymDoorBridge' service manually from Services.msc")
				} else {
					fmt.Println("✅ Service restarted successfully!")
				}
			}
		},
	}

	var unpairCmd = &cobra.Command{
		Use:   "unpair",
		Short: "Unpair the bridge from your gym management platform",
		Long: `Unpair the bridge from your gym management platform.
This removes the secure connection and device credentials.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			cfg, err := config.Load(configFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
				os.Exit(1)
			}
			
			// Initialize components for unpairing
			logger := logging.Initialize(logLevel)
			
			// Create auth manager
			authManager, err := auth.NewAuthManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create auth manager: %v\n", err)
				os.Exit(1)
			}
			if err := authManager.Initialize(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize auth manager: %v\n", err)
				os.Exit(1)
			}
			
			// Create HTTP client
			httpClient, err := client.NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create HTTP client: %v\n", err)
				os.Exit(1)
			}
			
			// Create pairing manager
			pairingManager, err := pairing.NewPairingManager(httpClient, authManager, cfg, logger)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create pairing manager: %v\n", err)
				os.Exit(1)
			}
			
			// Check if paired
			if !pairingManager.IsPaired() {
				fmt.Println("❌ Bridge is not currently paired")
				os.Exit(1)
			}
			
			deviceID := pairingManager.GetDeviceID()
			
			// Perform unpairing
			if err := pairingManager.UnpairDevice(); err != nil {
				fmt.Fprintf(os.Stderr, "Unpairing failed: %v\n", err)
				os.Exit(1)
			}
			
			// Clear configuration credentials
			cfg.DeviceID = ""
			cfg.DeviceKey = ""
			
			// Save updated configuration
			if err := cfg.Save(configFile); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save configuration: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("✅ Bridge unpaired successfully!\n")
			fmt.Printf("Previous Device ID: %s\n", deviceID)
			
			// Restart service if running on Windows
			if runtime.GOOS == "windows" {
				fmt.Println("\n🔄 Restarting service to apply new configuration...")
				if err := restartWindowsService(); err != nil {
					fmt.Printf("⚠️  Failed to restart service automatically: %v\n", err)
					fmt.Println("Please restart the 'GymDoorBridge' service manually from Services.msc")
				} else {
					fmt.Println("✅ Service restarted successfully!")
				}
			}
		},
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show bridge pairing and connection status",
		Long:  `Display the current pairing status and connection information for the bridge.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			cfg, err := config.Load(configFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Println("🔗 Gym Door Bridge Status")
			fmt.Println("========================")
			
			if cfg.IsPaired() {
				fmt.Printf("Status: ✅ PAIRED\n")
				fmt.Printf("Device ID: %s\n", cfg.DeviceID)
				fmt.Printf("Server URL: %s\n", cfg.ServerURL)
				fmt.Printf("Tier: %s\n", cfg.Tier)
				fmt.Printf("Heartbeat Interval: %d seconds\n", cfg.HeartbeatInterval)
				
				// Test connectivity if paired
				fmt.Printf("\nConnectivity Test: ")
				if err := testConnectivity(cfg); err != nil {
					fmt.Printf("❌ FAILED (%v)\n", err)
				} else {
					fmt.Printf("✅ CONNECTED\n")
				}
				
				// Show installation info if available
				if cfg.Installation.InstalledAt != "" {
					fmt.Printf("\nInstallation Info:\n")
					fmt.Printf("  Method: %s\n", cfg.Installation.Method)
					fmt.Printf("  Installed At: %s\n", cfg.Installation.InstalledAt)
					fmt.Printf("  Version: %s\n", cfg.Installation.Version)
				}
				
			// TODO: Re-enable when installer package is fixed
			// Check service status using installer
			// if installer, err := installer.NewInstaller(); err == nil {
			//	fmt.Printf("\nService Status: ")
			//	if status, err := installer.Status(); err == nil {
			//		switch status {
			//		case "Running":
			//			fmt.Printf("✅ RUNNING\n")
			//		case "Stopped":
			//			fmt.Printf("❌ STOPPED\n")
			//		case "Not Installed":
			//			fmt.Printf("⚠️ NOT INSTALLED\n")
			//			fmt.Printf("\nTo install the service, run:\n")
			//			fmt.Printf("  gym-door-bridge install\n")
			//		default:
			//			fmt.Printf("🔄 %s\n", status)
			//		}
			//	} else {
			//		fmt.Printf("❌ ERROR (%v)\n", err)
			//	}
			// }
			} else {
				fmt.Printf("Status: ❌ NOT PAIRED\n")
				fmt.Printf("Server URL: %s\n", cfg.ServerURL)
				fmt.Println("\nTo pair this bridge, run:")
				fmt.Println("  gym-door-bridge pair YOUR_PAIR_CODE")
			}
		},
	}

	var triggerHeartbeatCmd = &cobra.Command{
		Use:   "trigger-heartbeat",
		Short: "Manually trigger a heartbeat to the platform",
		Long:  `Send an immediate heartbeat to the platform to test connectivity and update status.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			cfg, err := config.Load(configFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
				os.Exit(1)
			}
			
			if !cfg.IsPaired() {
				fmt.Println("❌ Bridge is not paired. Use 'gym-door-bridge pair' first.")
				os.Exit(1)
			}
			
			// Initialize components
			logger := logging.Initialize(logLevel)
			
			// Create auth manager
			authManager, err := auth.NewAuthManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create auth manager: %v\n", err)
				os.Exit(1)
			}
			if err := authManager.Initialize(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize auth manager: %v\n", err)
				os.Exit(1)
			}
			
			// Create HTTP client
			httpClient, err := client.NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create HTTP client: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Println("🔄 Triggering heartbeat...")
			
			ctx := context.Background()
			if err := httpClient.TriggerHeartbeat(ctx); err != nil {
				fmt.Printf("❌ Heartbeat failed: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Println("✅ Heartbeat triggered successfully!")
		},
	}

	var deviceStatusCmd = &cobra.Command{
		Use:   "device-status",
		Short: "Check device status with the platform",
		Long:  `Query the platform for current device status and configuration.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration
			cfg, err := config.Load(configFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
				os.Exit(1)
			}
			
			if !cfg.IsPaired() {
				fmt.Println("❌ Bridge is not paired. Use 'gym-door-bridge pair' first.")
				os.Exit(1)
			}
			
			// Initialize components
			logger := logging.Initialize(logLevel)
			
			// Create auth manager
			authManager, err := auth.NewAuthManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create auth manager: %v\n", err)
				os.Exit(1)
			}
			if err := authManager.Initialize(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize auth manager: %v\n", err)
				os.Exit(1)
			}
			
			// Create HTTP client
			httpClient, err := client.NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create HTTP client: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Println("🔍 Checking device status...")
			
			ctx := context.Background()
			statusReq := &client.DeviceStatusRequest{
				RequestID: fmt.Sprintf("status_%d", time.Now().Unix()),
			}
			
			statusResp, err := httpClient.SendDeviceStatus(ctx, statusReq)
			if err != nil {
				fmt.Printf("❌ Status check failed: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Println("✅ Device Status Retrieved:")
			fmt.Printf("  Status: %s\n", statusResp.Status)
			fmt.Printf("  Last Seen: %s\n", statusResp.LastSeen)
			fmt.Printf("  Queue Depth: %d\n", statusResp.QueueDepth)
			
			if statusResp.SystemInfo != nil {
				fmt.Printf("  System Info:\n")
				fmt.Printf("    CPU Usage: %.1f%%\n", statusResp.SystemInfo.CPUUsage)
				fmt.Printf("    Memory Usage: %.1f%%\n", statusResp.SystemInfo.MemoryUsage)
				fmt.Printf("    Disk Space: %.1f%%\n", statusResp.SystemInfo.DiskSpace)
			}
			
			if len(statusResp.ConnectedDevices) > 0 {
				fmt.Printf("  Connected Devices: %d\n", len(statusResp.ConnectedDevices))
				for _, device := range statusResp.ConnectedDevices {
					fmt.Printf("    - %s\n", device)
				}
			}
		},
	}

	rootCmd.AddCommand(pairCmd)
	rootCmd.AddCommand(unpairCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(triggerHeartbeatCmd)
	rootCmd.AddCommand(deviceStatusCmd)
}

// restartWindowsService restarts the Windows service
func restartWindowsService() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("not running on Windows")
	}
	
	// Stop service
	stopCmd := exec.Command("net", "stop", "GymDoorBridge")
	if err := stopCmd.Run(); err != nil {
		// Service might not be running, continue
	}
	
	// Start service
	startCmd := exec.Command("net", "start", "GymDoorBridge")
	return startCmd.Run()
}

// isServiceRunning checks if the Windows service is running
func isServiceRunning() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	
	cmd := exec.Command("sc", "query", "GymDoorBridge")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	// Check if service is running
	return string(output) != "" && 
		   (string(output) != "" && 
		    (len(output) > 0))
}

// testConnectivity tests connectivity to the platform
func testConnectivity(cfg *config.Config) error {
	logger := logging.Initialize("error") // Quiet logging for test
	
	// Create auth manager
	authManager, err := auth.NewAuthManager()
	if err != nil {
		return fmt.Errorf("auth manager creation failed: %w", err)
	}
	if err := authManager.Initialize(); err != nil {
		return fmt.Errorf("auth manager initialization failed: %w", err)
	}
	
	// Create HTTP client
	httpClient, err := client.NewHTTPClient(cfg, authManager, logger)
	if err != nil {
		return fmt.Errorf("HTTP client creation failed: %w", err)
	}
	
	// Test connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return httpClient.CheckConnectivity(ctx)
}