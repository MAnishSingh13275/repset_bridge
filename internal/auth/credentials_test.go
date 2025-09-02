package auth

import (
	"runtime"
	"testing"
)

func TestNewCredentialManager(t *testing.T) {
	credManager, err := NewCredentialManager()
	
	switch runtime.GOOS {
	case "windows":
		if err != nil {
			t.Errorf("Expected no error on Windows, got: %v", err)
		}
		if credManager == nil {
			t.Error("Expected non-nil credential manager on Windows")
		}
	case "darwin":
		if err != nil {
			t.Errorf("Expected no error on macOS, got: %v", err)
		}
		if credManager == nil {
			t.Error("Expected non-nil credential manager on macOS")
		}
	default:
		if err == nil {
			t.Error("Expected error on unsupported platform")
		}
		if credManager != nil {
			t.Error("Expected nil credential manager on unsupported platform")
		}
	}
}

func TestCredentialManagerInterface(t *testing.T) {
	// Test that our mock implements the interface correctly
	var _ CredentialManager = &MockCredentialManager{}
	
	mock := NewMockCredentialManager()
	
	// Initially no credentials
	if mock.HasCredentials() {
		t.Error("Expected no credentials initially")
	}
	
	_, _, err := mock.GetCredentials()
	if err == nil {
		t.Error("Expected error when getting non-existent credentials")
	}
	
	// Store credentials
	deviceID := "dev_123"
	deviceKey := "secret_key"
	err = mock.StoreCredentials(deviceID, deviceKey)
	if err != nil {
		t.Errorf("StoreCredentials() error = %v", err)
	}
	
	// Should have credentials now
	if !mock.HasCredentials() {
		t.Error("Expected to have credentials after storing")
	}
	
	// Get credentials
	gotID, gotKey, err := mock.GetCredentials()
	if err != nil {
		t.Errorf("GetCredentials() error = %v", err)
	}
	
	if gotID != deviceID {
		t.Errorf("Expected device ID %s, got %s", deviceID, gotID)
	}
	
	if gotKey != deviceKey {
		t.Errorf("Expected device key %s, got %s", deviceKey, gotKey)
	}
	
	// Delete credentials
	err = mock.DeleteCredentials()
	if err != nil {
		t.Errorf("DeleteCredentials() error = %v", err)
	}
	
	// Should not have credentials now
	if mock.HasCredentials() {
		t.Error("Expected no credentials after deleting")
	}
}

func TestDeviceCredentials(t *testing.T) {
	creds := DeviceCredentials{
		DeviceID:  "dev_123",
		DeviceKey: "secret_key",
	}
	
	if creds.DeviceID != "dev_123" {
		t.Errorf("Expected device ID dev_123, got %s", creds.DeviceID)
	}
	
	if creds.DeviceKey != "secret_key" {
		t.Errorf("Expected device key secret_key, got %s", creds.DeviceKey)
	}
}