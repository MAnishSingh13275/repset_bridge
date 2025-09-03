package main

import (
	"context"
	"fmt"
	"time"

	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
	"gym-door-bridge/internal/pairing"

	"github.com/spf13/cobra"
)

var pairCmd = &cobra.Command{
	Use:   "pair",
	Short: "Pair device with cloud platform",
	Long: `Pair this device with the cloud platform using a pair code.
The pair code should be obtained from the admin portal.`,
	RunE: runPairCommand,
}

var (
	pairCode string
	timeout  int
)

func init() {
	pairCmd.Flags().StringVar(&pairCode, "pair-code", "", "Device pairing code from admin portal (required)")
	pairCmd.Flags().IntVar(&timeout, "timeout", 30, "Pairing timeout in seconds")
	pairCmd.MarkFlagRequired("pair-code")

	rootCmd.AddCommand(pairCmd)
}

func runPairCommand(cmd *cobra.Command, args []string) error {
	// Initialize logging
	logger := logging.Initialize(logLevel)

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger.WithField("server_url", cfg.ServerURL).Info("Starting device pairing")

	// Check if device is already paired
	if cfg.IsPaired() {
		return fmt.Errorf("device is already paired (device_id: %s). Use 'unpair' command to unpair first", cfg.DeviceID)
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

	// Create HTTP client
	httpClient, err := client.NewHTTPClient(cfg, authManager, logger)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create pairing manager
	pairingManager, err := pairing.NewPairingManager(httpClient, authManager, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create pairing manager: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Perform pairing
	fmt.Printf("Pairing device with code: %s\n", pairCode)
	fmt.Printf("Server URL: %s\n", cfg.ServerURL)
	fmt.Printf("Timeout: %d seconds\n", timeout)
	fmt.Println()

	pairResp, err := pairingManager.PairDevice(ctx, pairCode)
	if err != nil {
		return fmt.Errorf("pairing failed: %w", err)
	}

	// Update configuration file with device credentials
	if err := updateConfigWithPairing(cfg, pairResp, authManager); err != nil {
		logger.WithError(err).Warn("Failed to update configuration file")
		fmt.Printf("Warning: Failed to update configuration file: %v\n", err)
		fmt.Println("Device credentials have been stored securely, but you may need to manually update the config file.")
	}

	// Display success information
	fmt.Println("✓ Device paired successfully!")
	fmt.Printf("Device ID: %s\n", pairResp.DeviceID)
	fmt.Printf("Heartbeat Interval: %d seconds\n", pairResp.Config.HeartbeatInterval)
	fmt.Printf("Queue Max Size: %d events\n", pairResp.Config.QueueMaxSize)
	fmt.Printf("Unlock Duration: %d milliseconds\n", pairResp.Config.UnlockDuration)
	fmt.Println()
	fmt.Println("The device is now ready to connect to the cloud platform.")
	fmt.Println("You can start the bridge service to begin processing events.")

	return nil
}

// updateConfigWithPairing updates the configuration file with pairing information
func updateConfigWithPairing(cfg *config.Config, pairResp *client.PairResponse, authManager *auth.AuthManager) error {
	// Update in-memory configuration with device ID from response
	cfg.DeviceID = pairResp.DeviceID

	// Get the device key from secure storage (auth manager stores it)
	if authManager.IsAuthenticated() {
		// We know the device ID, now we need to get the key from secure storage
		// Since the auth manager interface doesn't expose GetDeviceKey directly,
		// we need to use the stored credentials from the credential manager
		deviceID, deviceKey, err := getStoredCredentials(authManager)
		if err != nil {
			return fmt.Errorf("failed to retrieve stored device key: %w", err)
		}
		cfg.DeviceKey = deviceKey

		// Verify device ID matches
		if deviceID != pairResp.DeviceID {
			return fmt.Errorf("device ID mismatch: response=%s, stored=%s", pairResp.DeviceID, deviceID)
		}
	} else {
		// Fallback to response (might be empty)
		cfg.DeviceKey = pairResp.DeviceKey
	}

	if pairResp.Config != nil {
		if pairResp.Config.HeartbeatInterval > 0 {
			cfg.HeartbeatInterval = pairResp.Config.HeartbeatInterval
		}
		if pairResp.Config.QueueMaxSize > 0 {
			cfg.QueueMaxSize = pairResp.Config.QueueMaxSize
		}
		if pairResp.Config.UnlockDuration > 0 {
			cfg.UnlockDuration = pairResp.Config.UnlockDuration
		}
	}

	// Save the updated configuration to file
	if err := cfg.Save(configFile); err != nil {
		return fmt.Errorf("failed to save configuration file: %w", err)
	}

	return nil
}

// getStoredCredentials retrieves credentials from the auth manager's storage
func getStoredCredentials(authManager *auth.AuthManager) (deviceID, deviceKey string, err error) {
	return authManager.GetCredentials()
}
