package database

import (
	"testing"
)

func TestSetGetConfig(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Test non-sensitive config
	err := db.SetConfig("heartbeat_interval", "60")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	value, err := db.GetConfig("heartbeat_interval")
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if value != "60" {
		t.Errorf("Expected config value '60', got '%s'", value)
	}
}

func TestSetGetSensitiveConfig(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Test sensitive config (should be encrypted)
	sensitiveValue := "super-secret-device-key-12345"
	err := db.SetConfig("device_key", sensitiveValue)
	if err != nil {
		t.Fatalf("Failed to set sensitive config: %v", err)
	}

	value, err := db.GetConfig("device_key")
	if err != nil {
		t.Fatalf("Failed to get sensitive config: %v", err)
	}

	if value != sensitiveValue {
		t.Errorf("Expected sensitive config value '%s', got '%s'", sensitiveValue, value)
	}

	// Verify it's actually encrypted in the database by checking raw value
	var rawValue string
	query := "SELECT value FROM device_config WHERE key = ?"
	err = db.conn.QueryRow(query, "device_key").Scan(&rawValue)
	if err != nil {
		t.Fatalf("Failed to get raw config value: %v", err)
	}

	if rawValue == sensitiveValue {
		t.Error("Sensitive config should be encrypted in database")
	}
}

func TestGetAllConfig(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set multiple config values
	configs := map[string]string{
		"heartbeat_interval": "60",
		"queue_max_size":     "10000",
		"device_key":         "secret-key-123",
		"api_endpoint":       "https://api.example.com",
	}

	for key, value := range configs {
		if err := db.SetConfig(key, value); err != nil {
			t.Fatalf("Failed to set config %s: %v", key, err)
		}
	}

	// Get all configs
	allConfigs, err := db.GetAllConfig()
	if err != nil {
		t.Fatalf("Failed to get all configs: %v", err)
	}

	// Verify all configs are present and correct
	for key, expectedValue := range configs {
		actualValue, exists := allConfigs[key]
		if !exists {
			t.Errorf("Expected config key %s to exist", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected config %s to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

func TestConfigExists(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Test non-existent config
	exists, err := db.ConfigExists("non_existent_key")
	if err != nil {
		t.Fatalf("Failed to check if config exists: %v", err)
	}
	if exists {
		t.Error("Expected non-existent config to return false")
	}

	// Set a config
	err = db.SetConfig("test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Test existing config
	exists, err = db.ConfigExists("test_key")
	if err != nil {
		t.Fatalf("Failed to check if config exists: %v", err)
	}
	if !exists {
		t.Error("Expected existing config to return true")
	}
}

func TestDeleteConfig(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set a config
	err := db.SetConfig("temp_key", "temp_value")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Verify it exists
	exists, err := db.ConfigExists("temp_key")
	if err != nil {
		t.Fatalf("Failed to check if config exists: %v", err)
	}
	if !exists {
		t.Fatal("Expected config to exist before deletion")
	}

	// Delete the config
	err = db.DeleteConfig("temp_key")
	if err != nil {
		t.Fatalf("Failed to delete config: %v", err)
	}

	// Verify it no longer exists
	exists, err = db.ConfigExists("temp_key")
	if err != nil {
		t.Fatalf("Failed to check if config exists after deletion: %v", err)
	}
	if exists {
		t.Error("Expected config to not exist after deletion")
	}
}

func TestConfigReplacement(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set initial value
	err := db.SetConfig("test_key", "initial_value")
	if err != nil {
		t.Fatalf("Failed to set initial config: %v", err)
	}

	// Replace with new value
	err = db.SetConfig("test_key", "updated_value")
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Verify updated value
	value, err := db.GetConfig("test_key")
	if err != nil {
		t.Fatalf("Failed to get updated config: %v", err)
	}

	if value != "updated_value" {
		t.Errorf("Expected updated config value 'updated_value', got '%s'", value)
	}
}