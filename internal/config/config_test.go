package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.ServerURL == "" {
		t.Error("ServerURL should not be empty")
	}
	
	if cfg.Tier != "normal" {
		t.Errorf("Expected tier 'normal', got '%s'", cfg.Tier)
	}
	
	if cfg.QueueMaxSize <= 0 {
		t.Error("QueueMaxSize should be positive")
	}
	
	if cfg.HeartbeatInterval <= 0 {
		t.Error("HeartbeatInterval should be positive")
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	
	// Valid config should pass
	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}
	
	// Invalid tier should fail
	cfg.Tier = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Invalid tier should return error")
	}
	
	// Reset to valid
	cfg.Tier = "normal"
	
	// Empty server URL should fail
	cfg.ServerURL = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Empty server URL should return error")
	}
}

func TestIsPaired(t *testing.T) {
	cfg := DefaultConfig()
	
	// Not paired initially
	if cfg.IsPaired() {
		t.Error("Default config should not be paired")
	}
	
	// Set device ID only
	cfg.DeviceID = "test-device"
	if cfg.IsPaired() {
		t.Error("Config with only device ID should not be paired")
	}
	
	// Set device key only
	cfg.DeviceID = ""
	cfg.DeviceKey = "test-key"
	if cfg.IsPaired() {
		t.Error("Config with only device key should not be paired")
	}
	
	// Set both
	cfg.DeviceID = "test-device"
	cfg.DeviceKey = "test-key"
	if !cfg.IsPaired() {
		t.Error("Config with both device ID and key should be paired")
	}
}