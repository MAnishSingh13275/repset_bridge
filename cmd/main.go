package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"gym-door-bridge/internal/config"
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
		ctx := context.Background()
		if err := bridgeMain(ctx, cfg); err != nil {
			logger.WithError(err).Fatal("Bridge execution failed")
		}
	}
}

// bridgeMain is the main bridge execution function
func bridgeMain(ctx context.Context, cfg *config.Config) error {
	logger := logging.Initialize(logLevel)
	
	logger.WithField("config", cfg).Info("Bridge main function starting")
	logger.Info("Gym Door Access Bridge initialized successfully")
	
	// TODO: Initialize and start bridge services
	// This is where the actual bridge functionality will be implemented
	// For now, we'll simulate the bridge running
	
	select {
	case <-ctx.Done():
		logger.Info("Bridge shutting down gracefully")
		return ctx.Err()
	}
}