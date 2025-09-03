package auth

import (
	"fmt"
	"time"
)

// AuthManager combines HMAC authentication with credential management
type AuthManager struct {
	authenticator *HMACAuthenticator
	credManager   CredentialManager
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() (*AuthManager, error) {
	credManager, err := NewCredentialManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential manager: %w", err)
	}

	return &AuthManager{
		credManager: credManager,
	}, nil
}

// Initialize loads existing credentials or prepares for pairing
func (a *AuthManager) Initialize() error {
	if a.credManager.HasCredentials() {
		deviceID, deviceKey, err := a.credManager.GetCredentials()
		if err != nil {
			return fmt.Errorf("failed to load existing credentials: %w", err)
		}

		a.authenticator = NewHMACAuthenticator(deviceID, deviceKey)
	}

	return nil
}

// SetCredentials stores new device credentials
func (a *AuthManager) SetCredentials(deviceID, deviceKey string) error {
	if err := a.credManager.StoreCredentials(deviceID, deviceKey); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	a.authenticator = NewHMACAuthenticator(deviceID, deviceKey)
	return nil
}

// IsAuthenticated returns true if device has valid credentials
func (a *AuthManager) IsAuthenticated() bool {
	return a.authenticator != nil && a.authenticator.GetDeviceID() != ""
}

// GetDeviceID returns the current device ID
func (a *AuthManager) GetDeviceID() string {
	if a.authenticator == nil {
		return ""
	}
	return a.authenticator.GetDeviceID()
}

// SignRequest signs an API request with current timestamp
func (a *AuthManager) SignRequest(body []byte) (signature string, timestamp int64, err error) {
	if a.authenticator == nil {
		return "", 0, fmt.Errorf("not authenticated")
	}

	timestamp = time.Now().Unix()
	signature, err = a.authenticator.SignRequest(body, timestamp)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign request: %w", err)
	}

	return signature, timestamp, nil
}

// ValidateSignature validates an incoming request signature
func (a *AuthManager) ValidateSignature(body []byte, timestamp int64, signature string) error {
	if a.authenticator == nil {
		return fmt.Errorf("not authenticated")
	}

	return a.authenticator.ValidateSignature(body, timestamp, signature)
}

// RotateCredentials updates device credentials (for key rotation)
func (a *AuthManager) RotateCredentials(newDeviceKey string) error {
	if a.authenticator == nil {
		return fmt.Errorf("not authenticated")
	}

	deviceID := a.authenticator.GetDeviceID()
	if err := a.credManager.StoreCredentials(deviceID, newDeviceKey); err != nil {
		return fmt.Errorf("failed to store rotated credentials: %w", err)
	}

	a.authenticator.UpdateCredentials(deviceID, newDeviceKey)
	return nil
}

// GetCredentials returns the stored device credentials
func (a *AuthManager) GetCredentials() (deviceID, deviceKey string, err error) {
	if !a.IsAuthenticated() {
		return "", "", fmt.Errorf("not authenticated")
	}

	return a.credManager.GetCredentials()
}

// ClearCredentials removes stored credentials
func (a *AuthManager) ClearCredentials() error {
	if err := a.credManager.DeleteCredentials(); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	a.authenticator = nil
	return nil
}
