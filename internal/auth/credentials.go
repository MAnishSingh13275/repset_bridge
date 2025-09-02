package auth

// CredentialManager handles secure storage and retrieval of device credentials
type CredentialManager interface {
	StoreCredentials(deviceID, deviceKey string) error
	GetCredentials() (deviceID, deviceKey string, err error)
	DeleteCredentials() error
	HasCredentials() bool
}

// NewCredentialManager creates a platform-specific credential manager
func NewCredentialManager() (CredentialManager, error) {
	return newPlatformCredentialManager()
}

// DeviceCredentials represents stored device credentials
type DeviceCredentials struct {
	DeviceID  string `json:"deviceId"`
	DeviceKey string `json:"deviceKey"`
}