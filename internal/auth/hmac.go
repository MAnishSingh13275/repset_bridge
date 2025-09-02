package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// HMACAuthenticator handles HMAC-SHA256 authentication for API requests
type HMACAuthenticator struct {
	deviceID  string
	deviceKey string
}

// NewHMACAuthenticator creates a new HMAC authenticator with device credentials
func NewHMACAuthenticator(deviceID, deviceKey string) *HMACAuthenticator {
	return &HMACAuthenticator{
		deviceID:  deviceID,
		deviceKey: deviceKey,
	}
}

// SignRequest generates HMAC signature for a request
// Signature is calculated as HMAC-SHA256(body + timestamp + deviceId)
func (h *HMACAuthenticator) SignRequest(body []byte, timestamp int64) (string, error) {
	if h.deviceKey == "" {
		return "", fmt.Errorf("device key not set")
	}

	// Create message: body + timestamp + deviceId
	message := string(body) + strconv.FormatInt(timestamp, 10) + h.deviceID

	// Generate HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(h.deviceKey))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	return signature, nil
}

// ValidateSignature validates an HMAC signature with clock skew tolerance
func (h *HMACAuthenticator) ValidateSignature(body []byte, timestamp int64, signature string) error {
	if h.deviceKey == "" {
		return fmt.Errorf("device key not set")
	}

	// Check clock skew tolerance (5 minutes)
	now := time.Now().Unix()
	if abs(now-timestamp) > 300 { // 5 minutes
		return fmt.Errorf("timestamp outside acceptable range")
	}

	// Generate expected signature
	expectedSignature, err := h.SignRequest(body, timestamp)
	if err != nil {
		return fmt.Errorf("failed to generate expected signature: %w", err)
	}

	// Compare signatures using constant time comparison
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("signature validation failed")
	}

	return nil
}

// GetDeviceID returns the device ID
func (h *HMACAuthenticator) GetDeviceID() string {
	return h.deviceID
}

// UpdateCredentials updates the device credentials
func (h *HMACAuthenticator) UpdateCredentials(deviceID, deviceKey string) {
	h.deviceID = deviceID
	h.deviceKey = deviceKey
}

// abs returns the absolute value of an int64
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}