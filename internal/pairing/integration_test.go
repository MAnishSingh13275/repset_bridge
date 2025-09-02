package pairing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
)

func TestPairingManager_Integration(t *testing.T) {
	logger := logging.Initialize("debug")

	// Create a test server that simulates the pairing API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/devices/pair" {
			t.Errorf("Expected /api/v1/devices/pair, got %s", r.URL.Path)
		}

		// Parse request
		var req client.PairRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate pair code
		if req.PairCode != "TEST123" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid pair code"}`))
			return
		}

		// Return successful pairing response
		resp := client.PairResponse{
			DeviceID:  "dev_integration_test",
			DeviceKey: "integration_test_key",
			Config: &client.DeviceConfig{
				HeartbeatInterval: 90,
				QueueMaxSize:      15000,
				UnlockDuration:    4000,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create config pointing to test server
	cfg := &config.Config{
		ServerURL:         server.URL,
		Tier:              "normal",
		HeartbeatInterval: 60,
		QueueMaxSize:      10000,
		UnlockDuration:    3000,
	}

	// Create pairing manager with real dependencies
	pm, err := NewPairingManagerWithRealDependencies(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	// Verify device is not paired initially
	if pm.IsPaired() {
		t.Error("Device should not be paired initially")
	}

	// Perform pairing
	ctx := context.Background()
	resp, err := pm.PairDevice(ctx, "TEST123")
	if err != nil {
		t.Fatalf("Pairing failed: %v", err)
	}

	// Verify pairing response
	if resp.DeviceID != "dev_integration_test" {
		t.Errorf("Expected device ID dev_integration_test, got %s", resp.DeviceID)
	}
	if resp.DeviceKey != "integration_test_key" {
		t.Errorf("Expected device key integration_test_key, got %s", resp.DeviceKey)
	}

	// Verify device is now paired
	if !pm.IsPaired() {
		t.Error("Device should be paired after successful pairing")
	}

	// Verify device ID is accessible
	if pm.GetDeviceID() != "dev_integration_test" {
		t.Errorf("Expected device ID dev_integration_test, got %s", pm.GetDeviceID())
	}

	// Verify config was updated
	if cfg.HeartbeatInterval != 90 {
		t.Errorf("Expected heartbeat interval 90, got %d", cfg.HeartbeatInterval)
	}
	if cfg.QueueMaxSize != 15000 {
		t.Errorf("Expected queue max size 15000, got %d", cfg.QueueMaxSize)
	}
	if cfg.UnlockDuration != 4000 {
		t.Errorf("Expected unlock duration 4000, got %d", cfg.UnlockDuration)
	}

	// Test unpairing
	if err := pm.UnpairDevice(); err != nil {
		t.Errorf("Unpair failed: %v", err)
	}

	// Verify device is no longer paired
	if pm.IsPaired() {
		t.Error("Device should not be paired after unpair")
	}
	if pm.GetDeviceID() != "" {
		t.Errorf("Expected empty device ID after unpair, got %s", pm.GetDeviceID())
	}
}

func TestPairingManager_Integration_InvalidPairCode(t *testing.T) {
	logger := logging.Initialize("debug")

	// Create a test server that rejects invalid pair codes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid pair code"}`))
	}))
	defer server.Close()

	// Create config pointing to test server
	cfg := &config.Config{
		ServerURL: server.URL,
		Tier:      "normal",
	}

	// Create pairing manager with real dependencies
	pm, err := NewPairingManagerWithRealDependencies(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	// Attempt pairing with invalid code
	ctx := context.Background()
	_, err = pm.PairDevice(ctx, "INVALID")
	if err == nil {
		t.Error("Expected pairing to fail with invalid pair code")
	}

	// Verify device is still not paired
	if pm.IsPaired() {
		t.Error("Device should not be paired after failed pairing")
	}
}

func TestPairingManager_Integration_AlreadyPaired(t *testing.T) {
	logger := logging.Initialize("debug")

	// Create a test server for initial pairing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := client.PairResponse{
			DeviceID:  "dev_already_paired",
			DeviceKey: "already_paired_key",
			Config: &client.DeviceConfig{
				HeartbeatInterval: 60,
				QueueMaxSize:      10000,
				UnlockDuration:    3000,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create config pointing to test server
	cfg := &config.Config{
		ServerURL: server.URL,
		Tier:      "normal",
	}

	// Create pairing manager with real dependencies
	pm, err := NewPairingManagerWithRealDependencies(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create pairing manager: %v", err)
	}

	// Perform initial pairing
	ctx := context.Background()
	_, err = pm.PairDevice(ctx, "FIRST")
	if err != nil {
		t.Fatalf("Initial pairing failed: %v", err)
	}

	// Verify device is paired
	if !pm.IsPaired() {
		t.Error("Device should be paired after first pairing")
	}

	// Attempt second pairing - should fail
	_, err = pm.PairDevice(ctx, "SECOND")
	if err == nil {
		t.Error("Expected second pairing attempt to fail")
	}

	// Verify device is still paired with original credentials
	if pm.GetDeviceID() != "dev_already_paired" {
		t.Errorf("Expected device ID dev_already_paired, got %s", pm.GetDeviceID())
	}
}