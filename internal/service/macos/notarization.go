package macos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NotarizationConfig holds configuration for macOS notarization
type NotarizationConfig struct {
	DeveloperID       string // Developer ID for code signing
	TeamID           string // Apple Developer Team ID
	BundleID         string // Bundle identifier
	AppleID          string // Apple ID for notarization
	AppSpecificPassword string // App-specific password for Apple ID
	KeychainProfile  string // Keychain profile name (optional)
}

// DefaultNotarizationConfig returns default notarization configuration
func DefaultNotarizationConfig() *NotarizationConfig {
	return &NotarizationConfig{
		BundleID: ServiceName,
		KeychainProfile: "gym-door-bridge-notarization",
	}
}

// NotarizationManager handles macOS binary notarization
type NotarizationManager struct {
	config *NotarizationConfig
}

// NewNotarizationManager creates a new notarization manager
func NewNotarizationManager(config *NotarizationConfig) *NotarizationManager {
	return &NotarizationManager{
		config: config,
	}
}

// SignBinary signs the binary with Developer ID
func (nm *NotarizationManager) SignBinary(binaryPath string) error {
	if nm.config.DeveloperID == "" {
		return fmt.Errorf("developer ID is required for code signing")
	}
	
	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", binaryPath)
	}
	
	fmt.Printf("Signing binary: %s\n", binaryPath)
	
	// Sign the binary
	cmd := exec.Command("codesign",
		"--sign", nm.config.DeveloperID,
		"--timestamp",
		"--options", "runtime",
		"--verbose",
		binaryPath,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("code signing failed: %w (output: %s)", err, string(output))
	}
	
	fmt.Printf("Binary signed successfully\n")
	return nil
}

// VerifySignature verifies the code signature of a binary
func (nm *NotarizationManager) VerifySignature(binaryPath string) error {
	fmt.Printf("Verifying signature: %s\n", binaryPath)
	
	cmd := exec.Command("codesign", "--verify", "--verbose", binaryPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signature verification failed: %w (output: %s)", err, string(output))
	}
	
	fmt.Printf("Signature verified successfully\n")
	return nil
}

// CreateZipForNotarization creates a zip file for notarization
func (nm *NotarizationManager) CreateZipForNotarization(binaryPath string) (string, error) {
	binaryDir := filepath.Dir(binaryPath)
	binaryName := filepath.Base(binaryPath)
	zipPath := filepath.Join(binaryDir, strings.TrimSuffix(binaryName, filepath.Ext(binaryName))+".zip")
	
	fmt.Printf("Creating zip for notarization: %s\n", zipPath)
	
	// Remove existing zip if it exists
	os.Remove(zipPath)
	
	// Create zip file
	cmd := exec.Command("zip", "-r", zipPath, binaryName)
	cmd.Dir = binaryDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create zip: %w (output: %s)", err, string(output))
	}
	
	fmt.Printf("Zip created successfully: %s\n", zipPath)
	return zipPath, nil
}

// SubmitForNotarization submits the zip file for notarization
func (nm *NotarizationManager) SubmitForNotarization(zipPath string) (string, error) {
	if nm.config.AppleID == "" {
		return "", fmt.Errorf("Apple ID is required for notarization")
	}
	
	if nm.config.AppSpecificPassword == "" && nm.config.KeychainProfile == "" {
		return "", fmt.Errorf("either app-specific password or keychain profile is required")
	}
	
	fmt.Printf("Submitting for notarization: %s\n", zipPath)
	
	// Build xcrun altool command
	args := []string{
		"altool",
		"--notarize-app",
		"--primary-bundle-id", nm.config.BundleID,
		"--username", nm.config.AppleID,
		"--file", zipPath,
	}
	
	// Use keychain profile if available, otherwise use app-specific password
	if nm.config.KeychainProfile != "" {
		args = append(args, "--keychain-profile", nm.config.KeychainProfile)
	} else {
		args = append(args, "--password", nm.config.AppSpecificPassword)
	}
	
	if nm.config.TeamID != "" {
		args = append(args, "--asc-provider", nm.config.TeamID)
	}
	
	cmd := exec.Command("xcrun", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("notarization submission failed: %w (output: %s)", err, string(output))
	}
	
	// Parse request UUID from output
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "RequestUUID") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				requestUUID := strings.TrimSpace(parts[1])
				fmt.Printf("Notarization submitted successfully. Request UUID: %s\n", requestUUID)
				return requestUUID, nil
			}
		}
	}
	
	return "", fmt.Errorf("failed to parse request UUID from notarization output: %s", outputStr)
}

// CheckNotarizationStatus checks the status of a notarization request
func (nm *NotarizationManager) CheckNotarizationStatus(requestUUID string) (string, error) {
	if nm.config.AppleID == "" {
		return "", fmt.Errorf("Apple ID is required for status check")
	}
	
	// Build xcrun altool command
	args := []string{
		"altool",
		"--notarization-info", requestUUID,
		"--username", nm.config.AppleID,
	}
	
	// Use keychain profile if available, otherwise use app-specific password
	if nm.config.KeychainProfile != "" {
		args = append(args, "--keychain-profile", nm.config.KeychainProfile)
	} else {
		args = append(args, "--password", nm.config.AppSpecificPassword)
	}
	
	cmd := exec.Command("xcrun", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("status check failed: %w (output: %s)", err, string(output))
	}
	
	// Parse status from output
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Status:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				status := strings.TrimSpace(parts[1])
				return status, nil
			}
		}
	}
	
	return "", fmt.Errorf("failed to parse status from output: %s", outputStr)
}

// WaitForNotarization waits for notarization to complete
func (nm *NotarizationManager) WaitForNotarization(requestUUID string, timeout time.Duration) error {
	fmt.Printf("Waiting for notarization to complete (timeout: %v)...\n", timeout)
	
	start := time.Now()
	for time.Since(start) < timeout {
		status, err := nm.CheckNotarizationStatus(requestUUID)
		if err != nil {
			return fmt.Errorf("failed to check notarization status: %w", err)
		}
		
		fmt.Printf("Notarization status: %s\n", status)
		
		switch status {
		case "success":
			fmt.Printf("Notarization completed successfully!\n")
			return nil
		case "invalid":
			return fmt.Errorf("notarization failed - binary was rejected")
		case "in progress":
			// Continue waiting
			time.Sleep(30 * time.Second)
		default:
			fmt.Printf("Unknown status: %s, continuing to wait...\n", status)
			time.Sleep(30 * time.Second)
		}
	}
	
	return fmt.Errorf("notarization timeout after %v", timeout)
}

// StapleNotarization staples the notarization ticket to the binary
func (nm *NotarizationManager) StapleNotarization(binaryPath string) error {
	fmt.Printf("Stapling notarization ticket: %s\n", binaryPath)
	
	cmd := exec.Command("xcrun", "stapler", "staple", binaryPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stapling failed: %w (output: %s)", err, string(output))
	}
	
	fmt.Printf("Notarization ticket stapled successfully\n")
	return nil
}

// NotarizeBinary performs the complete notarization process
func (nm *NotarizationManager) NotarizeBinary(binaryPath string) error {
	// Step 1: Sign the binary
	if err := nm.SignBinary(binaryPath); err != nil {
		return fmt.Errorf("signing failed: %w", err)
	}
	
	// Step 2: Verify signature
	if err := nm.VerifySignature(binaryPath); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	
	// Step 3: Create zip for notarization
	zipPath, err := nm.CreateZipForNotarization(binaryPath)
	if err != nil {
		return fmt.Errorf("zip creation failed: %w", err)
	}
	defer os.Remove(zipPath) // Clean up zip file
	
	// Step 4: Submit for notarization
	requestUUID, err := nm.SubmitForNotarization(zipPath)
	if err != nil {
		return fmt.Errorf("notarization submission failed: %w", err)
	}
	
	// Step 5: Wait for notarization to complete
	if err := nm.WaitForNotarization(requestUUID, 30*time.Minute); err != nil {
		return fmt.Errorf("notarization wait failed: %w", err)
	}
	
	// Step 6: Staple the notarization ticket
	if err := nm.StapleNotarization(binaryPath); err != nil {
		return fmt.Errorf("stapling failed: %w", err)
	}
	
	fmt.Printf("Binary notarization completed successfully: %s\n", binaryPath)
	return nil
}

// SetupKeychainProfile sets up a keychain profile for notarization
func (nm *NotarizationManager) SetupKeychainProfile() error {
	if nm.config.AppleID == "" || nm.config.AppSpecificPassword == "" {
		return fmt.Errorf("Apple ID and app-specific password are required for keychain profile setup")
	}
	
	fmt.Printf("Setting up keychain profile: %s\n", nm.config.KeychainProfile)
	
	cmd := exec.Command("xcrun", "altool",
		"--store-password-in-keychain-item", nm.config.KeychainProfile,
		"--username", nm.config.AppleID,
		"--password", nm.config.AppSpecificPassword,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain profile setup failed: %w (output: %s)", err, string(output))
	}
	
	fmt.Printf("Keychain profile setup completed\n")
	return nil
}