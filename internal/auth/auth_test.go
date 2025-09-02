package auth

import (
	"fmt"
	"testing"
	"time"
)

// MockCredentialManager for testing
type MockCredentialManager struct {
	credentials map[string]string
	hasCredentials bool
}

func NewMockCredentialManager() *MockCredentialManager {
	return &MockCredentialManager{
		credentials: make(map[string]string),
	}
}

func (m *MockCredentialManager) StoreCredentials(deviceID, deviceKey string) error {
	m.credentials["deviceID"] = deviceID
	m.credentials["deviceKey"] = deviceKey
	m.hasCredentials = true
	return nil
}

func (m *MockCredentialManager) GetCredentials() (string, string, error) {
	if !m.hasCredentials {
		return "", "", fmt.Errorf("no credentials stored")
	}
	return m.credentials["deviceID"], m.credentials["deviceKey"], nil
}

func (m *MockCredentialManager) DeleteCredentials() error {
	m.credentials = make(map[string]string)
	m.hasCredentials = false
	return nil
}

func (m *MockCredentialManager) HasCredentials() bool {
	return m.hasCredentials
}

func TestAuthManager_Initialize(t *testing.T) {
	tests := []struct {
		name           string
		hasCredentials bool
		deviceID       string
		deviceKey      string
		wantErr        bool
	}{
		{
			name:           "no existing credentials",
			hasCredentials: false,
			wantErr:        false,
		},
		{
			name:           "existing credentials",
			hasCredentials: true,
			deviceID:       "dev_123",
			deviceKey:      "secret_key",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCredManager := NewMockCredentialManager()
			if tt.hasCredentials {
				mockCredManager.StoreCredentials(tt.deviceID, tt.deviceKey)
			}

			authManager := &AuthManager{
				credManager: mockCredManager,
			}

			err := authManager.Initialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.hasCredentials {
				if !authManager.IsAuthenticated() {
					t.Error("Expected to be authenticated after loading credentials")
				}
				if authManager.GetDeviceID() != tt.deviceID {
					t.Errorf("Expected device ID %s, got %s", tt.deviceID, authManager.GetDeviceID())
				}
			} else {
				if authManager.IsAuthenticated() {
					t.Error("Expected not to be authenticated without credentials")
				}
			}
		})
	}
}

func TestAuthManager_SetCredentials(t *testing.T) {
	mockCredManager := NewMockCredentialManager()
	authManager := &AuthManager{
		credManager: mockCredManager,
	}

	deviceID := "dev_123"
	deviceKey := "secret_key"

	err := authManager.SetCredentials(deviceID, deviceKey)
	if err != nil {
		t.Errorf("SetCredentials() error = %v", err)
		return
	}

	if !authManager.IsAuthenticated() {
		t.Error("Expected to be authenticated after setting credentials")
	}

	if authManager.GetDeviceID() != deviceID {
		t.Errorf("Expected device ID %s, got %s", deviceID, authManager.GetDeviceID())
	}
}

func TestAuthManager_SignRequest(t *testing.T) {
	mockCredManager := NewMockCredentialManager()
	authManager := &AuthManager{
		credManager: mockCredManager,
	}

	// Test without credentials
	body := []byte(`{"test": "data"}`)
	_, _, err := authManager.SignRequest(body)
	if err == nil {
		t.Error("Expected error when signing without credentials")
	}

	// Set credentials and test signing
	deviceID := "dev_123"
	deviceKey := "secret_key"
	authManager.SetCredentials(deviceID, deviceKey)

	signature, timestamp, err := authManager.SignRequest(body)
	if err != nil {
		t.Errorf("SignRequest() error = %v", err)
		return
	}

	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	if timestamp == 0 {
		t.Error("Expected non-zero timestamp")
	}

	// Verify timestamp is recent
	now := time.Now().Unix()
	if abs(now-timestamp) > 5 { // Allow 5 seconds difference
		t.Error("Timestamp should be recent")
	}
}

func TestAuthManager_ValidateSignature(t *testing.T) {
	mockCredManager := NewMockCredentialManager()
	authManager := &AuthManager{
		credManager: mockCredManager,
	}

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()

	// Test without credentials
	err := authManager.ValidateSignature(body, timestamp, "signature")
	if err == nil {
		t.Error("Expected error when validating without credentials")
	}

	// Set credentials
	deviceID := "dev_123"
	deviceKey := "secret_key"
	authManager.SetCredentials(deviceID, deviceKey)

	// Generate valid signature
	signature, signTimestamp, err := authManager.SignRequest(body)
	if err != nil {
		t.Fatalf("Failed to generate signature: %v", err)
	}

	// Validate signature
	err = authManager.ValidateSignature(body, signTimestamp, signature)
	if err != nil {
		t.Errorf("ValidateSignature() error = %v", err)
	}

	// Test invalid signature
	err = authManager.ValidateSignature(body, signTimestamp, "invalid_signature")
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}

func TestAuthManager_RotateCredentials(t *testing.T) {
	mockCredManager := NewMockCredentialManager()
	authManager := &AuthManager{
		credManager: mockCredManager,
	}

	// Test without initial credentials
	err := authManager.RotateCredentials("new_key")
	if err == nil {
		t.Error("Expected error when rotating without initial credentials")
	}

	// Set initial credentials
	deviceID := "dev_123"
	oldKey := "old_key"
	authManager.SetCredentials(deviceID, oldKey)

	// Rotate credentials
	newKey := "new_key"
	err = authManager.RotateCredentials(newKey)
	if err != nil {
		t.Errorf("RotateCredentials() error = %v", err)
		return
	}

	// Test signing with new key
	body := []byte(`{"test": "data"}`)
	signature, timestamp, err := authManager.SignRequest(body)
	if err != nil {
		t.Errorf("Failed to sign with rotated credentials: %v", err)
		return
	}

	// Validate with new key
	err = authManager.ValidateSignature(body, timestamp, signature)
	if err != nil {
		t.Errorf("Failed to validate with rotated credentials: %v", err)
	}

	// Verify stored credentials were updated
	storedID, storedKey, err := mockCredManager.GetCredentials()
	if err != nil {
		t.Errorf("Failed to get stored credentials: %v", err)
		return
	}

	if storedID != deviceID {
		t.Errorf("Expected stored device ID %s, got %s", deviceID, storedID)
	}

	if storedKey != newKey {
		t.Errorf("Expected stored device key %s, got %s", newKey, storedKey)
	}
}

func TestAuthManager_ClearCredentials(t *testing.T) {
	mockCredManager := NewMockCredentialManager()
	authManager := &AuthManager{
		credManager: mockCredManager,
	}

	// Set credentials
	deviceID := "dev_123"
	deviceKey := "secret_key"
	authManager.SetCredentials(deviceID, deviceKey)

	if !authManager.IsAuthenticated() {
		t.Error("Expected to be authenticated before clearing")
	}

	// Clear credentials
	err := authManager.ClearCredentials()
	if err != nil {
		t.Errorf("ClearCredentials() error = %v", err)
		return
	}

	if authManager.IsAuthenticated() {
		t.Error("Expected not to be authenticated after clearing")
	}

	if authManager.GetDeviceID() != "" {
		t.Error("Expected empty device ID after clearing")
	}

	// Verify credentials were deleted from storage
	if mockCredManager.HasCredentials() {
		t.Error("Expected credentials to be deleted from storage")
	}
}

func TestAuthManager_IsAuthenticated(t *testing.T) {
	mockCredManager := NewMockCredentialManager()
	authManager := &AuthManager{
		credManager: mockCredManager,
	}

	// Initially not authenticated
	if authManager.IsAuthenticated() {
		t.Error("Expected not to be authenticated initially")
	}

	// After setting credentials
	authManager.SetCredentials("dev_123", "secret_key")
	if !authManager.IsAuthenticated() {
		t.Error("Expected to be authenticated after setting credentials")
	}

	// After clearing credentials
	authManager.ClearCredentials()
	if authManager.IsAuthenticated() {
		t.Error("Expected not to be authenticated after clearing")
	}
}