package auth

import (
	"testing"
	"time"
)

func TestHMACAuthenticator_SignRequest(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  string
		deviceKey string
		body      []byte
		timestamp int64
		wantErr   bool
	}{
		{
			name:      "valid signature",
			deviceID:  "dev_123",
			deviceKey: "secret_key",
			body:      []byte(`{"test": "data"}`),
			timestamp: 1640995200,
			wantErr:   false,
		},
		{
			name:      "empty device key",
			deviceID:  "dev_123",
			deviceKey: "",
			body:      []byte(`{"test": "data"}`),
			timestamp: 1640995200,
			wantErr:   true,
		},
		{
			name:      "empty body",
			deviceID:  "dev_123",
			deviceKey: "secret_key",
			body:      []byte{},
			timestamp: 1640995200,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHMACAuthenticator(tt.deviceID, tt.deviceKey)
			signature, err := h.SignRequest(tt.body, tt.timestamp)

			if (err != nil) != tt.wantErr {
				t.Errorf("SignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && signature == "" {
				t.Error("SignRequest() returned empty signature")
			}
		})
	}
}

func TestHMACAuthenticator_ValidateSignature(t *testing.T) {
	deviceID := "dev_123"
	deviceKey := "secret_key"
	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()

	h := NewHMACAuthenticator(deviceID, deviceKey)

	// Generate valid signature
	validSignature, err := h.SignRequest(body, timestamp)
	if err != nil {
		t.Fatalf("Failed to generate signature: %v", err)
	}

	tests := []struct {
		name      string
		body      []byte
		timestamp int64
		signature string
		wantErr   bool
	}{
		{
			name:      "valid signature",
			body:      body,
			timestamp: timestamp,
			signature: validSignature,
			wantErr:   false,
		},
		{
			name:      "invalid signature",
			body:      body,
			timestamp: timestamp,
			signature: "invalid_signature",
			wantErr:   true,
		},
		{
			name:      "timestamp too old",
			body:      body,
			timestamp: timestamp - 400, // 6+ minutes ago
			signature: validSignature,
			wantErr:   true,
		},
		{
			name:      "timestamp too new",
			body:      body,
			timestamp: timestamp + 400, // 6+ minutes in future
			signature: validSignature,
			wantErr:   true,
		},
		{
			name:      "different body",
			body:      []byte(`{"different": "data"}`),
			timestamp: timestamp,
			signature: validSignature,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.ValidateSignature(tt.body, tt.timestamp, tt.signature)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHMACAuthenticator_UpdateCredentials(t *testing.T) {
	h := NewHMACAuthenticator("old_id", "old_key")

	// Update credentials
	newID := "new_id"
	newKey := "new_key"
	h.UpdateCredentials(newID, newKey)

	if h.GetDeviceID() != newID {
		t.Errorf("Expected device ID %s, got %s", newID, h.GetDeviceID())
	}

	// Test signing with new credentials
	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	signature, err := h.SignRequest(body, timestamp)
	if err != nil {
		t.Errorf("Failed to sign with new credentials: %v", err)
	}

	// Validate with new credentials
	if err := h.ValidateSignature(body, timestamp, signature); err != nil {
		t.Errorf("Failed to validate with new credentials: %v", err)
	}
}

func TestHMACAuthenticator_SignatureConsistency(t *testing.T) {
	h := NewHMACAuthenticator("dev_123", "secret_key")
	body := []byte(`{"test": "data"}`)
	timestamp := int64(1640995200)

	// Generate signature multiple times
	sig1, err := h.SignRequest(body, timestamp)
	if err != nil {
		t.Fatalf("Failed to generate first signature: %v", err)
	}

	sig2, err := h.SignRequest(body, timestamp)
	if err != nil {
		t.Fatalf("Failed to generate second signature: %v", err)
	}

	if sig1 != sig2 {
		t.Error("Signatures should be consistent for same input")
	}
}

func TestHMACAuthenticator_EmptyDeviceKey(t *testing.T) {
	h := NewHMACAuthenticator("dev_123", "")
	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()

	// Should fail to sign
	_, err := h.SignRequest(body, timestamp)
	if err == nil {
		t.Error("Expected error when signing with empty device key")
	}

	// Should fail to validate
	err = h.ValidateSignature(body, timestamp, "any_signature")
	if err == nil {
		t.Error("Expected error when validating with empty device key")
	}
}