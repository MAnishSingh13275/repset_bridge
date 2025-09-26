package fingerprint

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"gym-door-bridge/internal/types"
)

func TestFingerprintAdapter_Initialize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewFingerprintAdapter(logger)

	tests := []struct {
		name        string
	config      types.AdapterConfig
		expectError bool
	}{
		{
			name: "valid configuration",
		config: types.AdapterConfig{
				Name:    "fingerprint",
				Enabled: true,
				Settings: map[string]interface{}{
					"devicePath": "/dev/ttyUSB0",
					"baudRate":   9600.0,
					"protocol":   "wiegand",
				},
			},
			expectError: false,
		},
		{
			name: "missing devicePath",
		config: types.AdapterConfig{
				Name:     "fingerprint",
				Enabled:  true,
				Settings: map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name: "custom settings",
		config: types.AdapterConfig{
				Name:    "fingerprint",
				Enabled: true,
				Settings: map[string]interface{}{
					"devicePath": "COM3",
					"baudRate":   115200.0,
					"protocol":   "rs485",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.Initialize(context.Background(), tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				status := adapter.GetStatus()
			if status.Status != types.StatusActive {
					t.Errorf("expected status 'active', got '%s'", status.Status)
				}
			}
		})
	}
}

func TestFingerprintAdapter_StartStopListening(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewFingerprintAdapter(logger)

	config := types.AdapterConfig{
		Name:    "fingerprint",
		Enabled: true,
		Settings: map[string]interface{}{
			"devicePath": "/dev/ttyUSB0",
		},
	}

	// Initialize adapter
	err := adapter.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	// Test starting without callback
	err = adapter.StartListening(context.Background())
	if err == nil {
		t.Error("expected error when starting without callback")
	}

	// Register callback
	adapter.OnEvent(func(event types.RawHardwareEvent) {})

	// Start listening - should fail as this is framework implementation
	err = adapter.StartListening(context.Background())
	if err == nil {
		t.Error("expected error for framework implementation")
	}

	// Stop listening should work even if not started
	err = adapter.StopListening(context.Background())
	if err != nil {
		t.Errorf("unexpected error stopping: %v", err)
	}
}

func TestFingerprintAdapter_UnlockDoor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewFingerprintAdapter(logger)

	config := types.AdapterConfig{
		Name:    "fingerprint",
		Enabled: true,
		Settings: map[string]interface{}{
			"devicePath": "/dev/ttyUSB0",
		},
	}

	// Initialize adapter
	err := adapter.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	// UnlockDoor should fail as this is framework implementation
	err = adapter.UnlockDoor(context.Background(), 3000)
	if err == nil {
		t.Error("expected error for framework implementation")
	}
}

func TestFingerprintAdapter_Status(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewFingerprintAdapter(logger)

	// Test initial status
	status := adapter.GetStatus()
	if status.Name != "fingerprint" {
		t.Errorf("expected name 'fingerprint', got '%s'", status.Name)
	}
	if status.Status != types.StatusDisabled {
		t.Errorf("expected status 'disabled', got '%s'", status.Status)
	}

	// Test name
	if adapter.Name() != "fingerprint" {
		t.Errorf("expected name 'fingerprint', got '%s'", adapter.Name())
	}

	// Test health check
	if adapter.IsHealthy() {
		t.Error("expected adapter to not be healthy initially")
	}
}

func TestFingerprintAdapter_ProcessRawScanData(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewFingerprintAdapter(logger)

	// Test processing raw scan data
	rawData := []byte{0x01, 0x02, 0x03, 0x04}
	event, err := adapter.processRawScanData(rawData)
	if err != nil {
		t.Errorf("unexpected error processing raw data: %v", err)
	}

	if event == nil {
		t.Fatal("expected event but got nil")
	}

	if event.ExternalUserID == "" {
		t.Error("expected external user ID to be set")
	}

	if !types.IsValidEventType(event.EventType) {
		t.Errorf("expected valid event type, got '%s'", event.EventType)
	}

	if event.RawData == nil {
		t.Error("expected raw data to be set")
	}

	// Check fingerprint-specific metadata
	if fingerprint, ok := event.RawData["fingerprint"].(bool); !ok || !fingerprint {
		t.Error("expected fingerprint metadata to be true")
	}
}

func TestGetSupportedProtocols(t *testing.T) {
	protocols := GetSupportedProtocols()
	
	expectedProtocols := []string{"wiegand", "rs485", "tcp", "usb_hid", "serial"}
	if len(protocols) != len(expectedProtocols) {
		t.Errorf("expected %d protocols, got %d", len(expectedProtocols), len(protocols))
	}

	protocolMap := make(map[string]bool)
	for _, protocol := range protocols {
		protocolMap[protocol] = true
	}

	for _, expected := range expectedProtocols {
		if !protocolMap[expected] {
			t.Errorf("expected protocol '%s' to be supported", expected)
		}
	}
}

func TestValidateProtocol(t *testing.T) {
	tests := []struct {
		protocol string
		valid    bool
	}{
		{"wiegand", true},
		{"rs485", true},
		{"tcp", true},
		{"usb_hid", true},
		{"serial", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			result := ValidateProtocol(tt.protocol)
			if result != tt.valid {
				t.Errorf("expected ValidateProtocol('%s') = %v, got %v", tt.protocol, tt.valid, result)
			}
		})
	}
}