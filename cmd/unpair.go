package main

import (
	"fmt"

	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
	"gym-door-bridge/internal/pairing"

	"github.com/spf13/cobra"
)

var unpairCmd = &cobra.Command{
	Use:   "unpair",
	Short: "Unpair device from cloud platform",
	Long: `Remove device pairing and clear stored credentials.
This will disconnect the device from the cloud platform.`,
	RunE: runUnpairCommand,
}

var (
	force bool
)

func init() {
	unpairCmd.Flags().BoolVar(&force, "force", false, "Force unpair without confirmation")
	
	rootCmd.AddCommand(unpairCmd)
}

func runUnpairCommand(cmd *cobra.Command, args []string) error {
	// Initialize logging
	logger := logging.Initialize(logLevel)
	
	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Check if device is paired
	if !cfg.IsPaired() {
		fmt.Println("Device is not paired.")
		return nil
	}
	
	// Create auth manager
	authManager, err := auth.NewAuthManager()
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}
	
	// Initialize auth manager
	if err := authManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize auth manager: %w", err)
	}
	
	// Create pairing manager (we don't need HTTP client for unpair)
	pairingManager, err := pairing.NewPairingManager(nil, authManager, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create pairing manager: %w", err)
	}
	
	// Get current device ID for display
	deviceID := pairingManager.GetDeviceID()
	
	// Confirm unpair unless force flag is used
	if !force {
		fmt.Printf("This will unpair device '%s' from the cloud platform.\n", deviceID)
		fmt.Printf("Are you sure you want to continue? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Unpair cancelled.")
			return nil
		}
	}
	
	// Perform unpair
	fmt.Printf("Unpairing device: %s\n", deviceID)
	
	if err := pairingManager.UnpairDevice(); err != nil {
		return fmt.Errorf("unpair failed: %w", err)
	}
	
	// Clear configuration file credentials
	if err := clearConfigCredentials(cfg); err != nil {
		logger.WithError(err).Warn("Failed to clear configuration file credentials")
		fmt.Printf("Warning: Failed to clear configuration file: %v\n", err)
		fmt.Println("Device credentials have been cleared from secure storage, but you may need to manually update the config file.")
	}
	
	// Display success information
	fmt.Println("✓ Device unpaired successfully!")
	fmt.Printf("Device ID '%s' has been disconnected from the cloud platform.\n", deviceID)
	fmt.Println("Stored credentials have been cleared.")
	fmt.Println()
	fmt.Println("To reconnect this device, use the 'pair' command with a new pair code.")
	
	return nil
}

// clearConfigCredentials clears device credentials from the configuration
func clearConfigCredentials(cfg *config.Config) error {
	// Clear in-memory configuration
	cfg.DeviceID = ""
	cfg.DeviceKey = ""
	
	// TODO: Implement configuration file writing to persist the changes
	// For now, we rely on the credential manager to clear the sensitive data
	// The configuration file update would need to be implemented in the config package
	
	return nil
}