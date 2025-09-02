//go:build darwin

package auth

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// MacOSCredentialManager uses macOS Keychain for secure credential storage
type MacOSCredentialManager struct {
	serviceName string
	accountName string
}

// NewMacOSCredentialManager creates a new macOS credential manager
func NewMacOSCredentialManager() (*MacOSCredentialManager, error) {
	return &MacOSCredentialManager{
		serviceName: "GymDoorBridge",
		accountName: "device-credentials",
	}, nil
}

// StoreCredentials stores device credentials in macOS Keychain
func (m *MacOSCredentialManager) StoreCredentials(deviceID, deviceKey string) error {
	creds := DeviceCredentials{
		DeviceID:  deviceID,
		DeviceKey: deviceKey,
	}

	// Marshal credentials to JSON
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Delete existing entry if it exists
	m.DeleteCredentials()

	// Add to keychain
	cmd := exec.Command("security", "add-generic-password",
		"-s", m.serviceName,
		"-a", m.accountName,
		"-w", string(data),
		"-U", // Update if exists
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store credentials in keychain: %w", err)
	}

	return nil
}

// GetCredentials retrieves device credentials from macOS Keychain
func (m *MacOSCredentialManager) GetCredentials() (string, string, error) {
	// Retrieve from keychain
	cmd := exec.Command("security", "find-generic-password",
		"-s", m.serviceName,
		"-a", m.accountName,
		"-w", // Output password only
	)

	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to retrieve credentials from keychain: %w", err)
	}

	// Parse JSON credentials
	data := strings.TrimSpace(string(output))
	var creds DeviceCredentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return creds.DeviceID, creds.DeviceKey, nil
}

// DeleteCredentials removes stored credentials from macOS Keychain
func (m *MacOSCredentialManager) DeleteCredentials() error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", m.serviceName,
		"-a", m.accountName,
	)

	// Ignore error if item doesn't exist
	cmd.Run()
	return nil
}

// HasCredentials checks if credentials are stored in macOS Keychain
func (m *MacOSCredentialManager) HasCredentials() bool {
	cmd := exec.Command("security", "find-generic-password",
		"-s", m.serviceName,
		"-a", m.accountName,
	)

	err := cmd.Run()
	return err == nil
}/
/ newPlatformCredentialManager creates a macOS credential manager
func newPlatformCredentialManager() (CredentialManager, error) {
	return NewMacOSCredentialManager()
}