package rfid

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"gym-door-bridge/internal/types"
)

func TestRFIDAdapter_Initialize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewRFIDAdapter(logger)

	tests := []struct {
		name        string
	config      types.AdapterConfig
		expectError bool
	}{
		{
			name: "valid configuration",
		config: types.AdapterConfig{
				Name:    "rfid",
				Enabled: true,
				Settings: map[string]interface{}{
					"devicePath": "/dev/ttyUSB0",
					"baudRate":   9600.0,
					"frequency":  "13.56MHz",
					"cardTypes":  []interface{}{"mifare", "ntag"},
				},
			},
			expectError: false,
		},
		{
			name: "missing devicePath",
		config: types.AdapterConfig{
				Name:     "rfid",
				Enabled:  true,
				Settings: map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name: "invalid frequency",
		config: types.AdapterConfig{
				Name:    "rfid",
				Enabled: true,
				Settings: map[string]interface{}{
					"devicePath": "/dev/ttyUSB0",
					"frequency":  "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "LF RFID configuration",
		config: types.AdapterConfig{
				Name:    "rfid",
				Enabled: true,
				Settings: map[string]interface{}{
					"devicePath": "COM3",
					"baudRate":   115200.0,
					"frequency":  "125kHz",
					"cardTypes":  []interface{}{"em4100", "hid"},
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

func TestRFIDAdapter_StartStopListening(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewRFIDAdapter(logger)

	config := types.AdapterConfig{
		Name:    "rfid",
		Enabled: true,
		Settings: map[string]interface{}{
			"devicePath": "/dev/ttyUSB0",
			"frequency":  "13.56MHz",
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

func TestRFIDAdapter_UnlockDoor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewRFIDAdapter(logger)

	config := types.AdapterConfig{
		Name:    "rfid",
		Enabled: true,
		Settings: map[string]interface{}{
			"devicePath": "/dev/ttyUSB0",
			"frequency":  "13.56MHz",
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

func TestRFIDAdapter_Status(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewRFIDAdapter(logger)

	// Test initial status
	status := adapter.GetStatus()
	if status.Name != "rfid" {
		t.Errorf("expected name 'rfid', got '%s'", status.Name)
	}
	if status.Status != types.StatusDisabled {
		t.Errorf("expected status 'disabled', got '%s'", status.Status)
	}

	// Test name
	if adapter.Name() != "rfid" {
		t.Errorf("expected name 'rfid', got '%s'", adapter.Name())
	}

	// Test health check
	if adapter.IsHealthy() {
		t.Error("expected adapter to not be healthy initially")
	}
}

func TestRFIDAdapter_ProcessRawCardData(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewRFIDAdapter(logger)

	// Initialize with frequency setting
	config := types.AdapterConfig{
		Name:    "rfid",
		Enabled: true,
		Settings: map[string]interface{}{
			"devicePath": "/dev/ttyUSB0",
			"frequency":  "13.56MHz",
		},
	}
	adapter.Initialize(context.Background(), config)

	// Test processing raw card data
	rawData := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	event, err := adapter.processRawCardData(rawData)
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

	// Check RFID-specific metadata
	if rfid, ok := event.RawData["rfid"].(bool); !ok || !rfid {
		t.Error("expected rfid metadata to be true")
	}

	if frequency, ok := event.RawData["frequency"].(string); !ok || frequency != "13.56MHz" {
		t.Errorf("expected frequency '13.56MHz', got '%v'", event.RawData["frequency"])
	}
}

func TestGetSupportedFrequencies(t *testing.T) {
	frequencies := GetSupportedFrequencies()
	
	expectedFrequencies := []string{"125kHz", "134.2kHz", "13.56MHz", "860-960MHz"}
	if len(frequencies) != len(expectedFrequencies) {
		t.Errorf("expected %d frequencies, got %d", len(expectedFrequencies), len(frequencies))
	}

	frequencyMap := make(map[string]bool)
	for _, frequency := range frequencies {
		frequencyMap[frequency] = true
	}

	for _, expected := range expectedFrequencies {
		if !frequencyMap[expected] {
			t.Errorf("expected frequency '%s' to be supported", expected)
		}
	}
}

func TestValidateFrequency(t *testing.T) {
	tests := []struct {
		frequency string
		valid     bool
	}{
		{"125kHz", true},
		{"134.2kHz", true},
		{"13.56MHz", true},
		{"860-960MHz", true},
		{"invalid", false},
		{"", false},
		{"2.4GHz", false},
	}

	for _, tt := range tests {
		t.Run(tt.frequency, func(t *testing.T) {
			result := ValidateFrequency(tt.frequency)
			if result != tt.valid {
				t.Errorf("expected ValidateFrequency('%s') = %v, got %v", tt.frequency, tt.valid, result)
			}
		})
	}
}

func TestGetSupportedCardTypes(t *testing.T) {
	cardTypes := GetSupportedCardTypes()
	
	expectedTypes := []string{"mifare", "ntag", "iclass", "prox", "em4100", "hid", "indala"}
	if len(cardTypes) != len(expectedTypes) {
		t.Errorf("expected %d card types, got %d", len(expectedTypes), len(cardTypes))
	}

	typeMap := make(map[string]bool)
	for _, cardType := range cardTypes {
		typeMap[cardType] = true
	}

	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("expected card type '%s' to be supported", expected)
		}
	}
}

func TestValidateCardType(t *testing.T) {
	tests := []struct {
		cardType string
		valid    bool
	}{
		{"mifare", true},
		{"ntag", true},
		{"iclass", true},
		{"prox", true},
		{"em4100", true},
		{"hid", true},
		{"indala", true},
		{"invalid", false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.cardType, func(t *testing.T) {
			result := ValidateCardType(tt.cardType)
			if result != tt.valid {
				t.Errorf("expected ValidateCardType('%s') = %v, got %v", tt.cardType, tt.valid, result)
			}
		})
	}
}