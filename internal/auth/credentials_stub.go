//go:build !windows && !darwin

package auth

import "fmt"

// StubCredentialManager is a placeholder for unsupported platforms
type StubCredentialManager struct{}

// NewStubCredentialManager creates a stub credential manager
func NewStubCredentialManager() (*StubCredentialManager, error) {
	return nil, fmt.Errorf("credential management not supported on this platform")
}

// StoreCredentials is not implemented
func (s *StubCredentialManager) StoreCredentials(deviceID, deviceKey string) error {
	return fmt.Errorf("not implemented")
}

// GetCredentials is not implemented
func (s *StubCredentialManager) GetCredentials() (string, string, error) {
	return "", "", fmt.Errorf("not implemented")
}

// DeleteCredentials is not implemented
func (s *StubCredentialManager) DeleteCredentials() error {
	return fmt.Errorf("not implemented")
}

// HasCredentials is not implemented
func (s *StubCredentialManager) HasCredentials() bool {
	return false
}

// newPlatformCredentialManager returns an error for unsupported platforms
func newPlatformCredentialManager() (CredentialManager, error) {
	return nil, fmt.Errorf("credential management not supported on this platform")
}