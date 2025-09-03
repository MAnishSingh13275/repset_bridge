package main

import (
	"fmt"

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

	// Check if device is paired (check both config and auth manager)
	isPairedInConfig := cfg.IsPaired()
	isPairedInAuth := false

	// Create pairing manager with real dependencies
	pairingManager, err := pairing.NewPairingManagerWithRealDependencies(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create pairing manager: %w", err)
	}

	isPairedInAuth = pairingManager.IsPaired()

	if !isPairedInConfig && !isPairedInAuth {
		fmt.Println("Device is not paired.")
		return nil
	}

	// Get current device ID for display (prefer config over auth manager)
	deviceID := cfg.DeviceID
	if deviceID == "" {
		deviceID = pairingManager.GetDeviceID()
	}

	// Confirm unpair unless force flag is used
	if !force {
		if deviceID != "" {
			fmt.Printf("This will unpair device '%s' from the cloud platform.\n", deviceID)
		} else {
			fmt.Println("This will clear any stored device credentials.")
		}
		fmt.Printf("Are you sure you want to continue? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Unpair cancelled.")
			return nil
		}
	}

	// Perform unpair operations
	if deviceID != "" {
		fmt.Printf("Unpairing device: %s\n", deviceID)
	} else {
		fmt.Println("Clearing stored credentials...")
	}

	// Try to unpair from auth manager if it thinks it's paired
	if isPairedInAuth {
		if err := pairingManager.UnpairDevice(); err != nil {
			logger.WithError(err).Warn("Failed to unpair from auth manager")
			fmt.Printf("Warning: Failed to clear auth manager credentials: %v\n", err)
		} else {
			fmt.Println("✓ Cleared credentials from auth manager")
		}
	}

	// Clear configuration file credentials if present
	if isPairedInConfig {
		if err := clearConfigCredentials(cfg); err != nil {
			logger.WithError(err).Warn("Failed to clear configuration file credentials")
			fmt.Printf("Warning: Failed to clear configuration file: %v\n", err)
		} else {
			fmt.Println("✓ Cleared credentials from configuration file")
		}
	}

	// Display success information
	fmt.Println("✓ Device unpaired successfully!")
	if deviceID != "" {
		fmt.Printf("Device ID '%s' has been disconnected from the cloud platform.\n", deviceID)
	}
	fmt.Println("All stored credentials have been cleared.")
	fmt.Println()
	fmt.Println("To reconnect this device, use the 'pair' command with a new pair code.")

	return nil
}

// clearConfigCredentials clears device credentials from the configuration
func clearConfigCredentials(cfg *config.Config) error {
	// Clear in-memory configuration
	cfg.DeviceID = ""
	cfg.DeviceKey = ""

	// Save the updated configuration back to file
	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}
