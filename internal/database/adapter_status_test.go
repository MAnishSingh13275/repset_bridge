package database

import (
	"testing"
	"time"
)

func TestSetAdapterStatus(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set adapter status without error message
	err := db.SetAdapterStatus("fingerprint", AdapterStatusActive, "")
	if err != nil {
		t.Fatalf("Failed to set adapter status: %v", err)
	}

	// Set adapter status with error message
	err = db.SetAdapterStatus("rfid", AdapterStatusError, "Connection timeout")
	if err != nil {
		t.Fatalf("Failed to set adapter status with error: %v", err)
	}
}

func TestGetAdapterStatus(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set adapter status
	err := db.SetAdapterStatus("fingerprint", AdapterStatusActive, "")
	if err != nil {
		t.Fatalf("Failed to set adapter status: %v", err)
	}

	// Get adapter status
	status, err := db.GetAdapterStatus("fingerprint")
	if err != nil {
		t.Fatalf("Failed to get adapter status: %v", err)
	}

	if status.AdapterName != "fingerprint" {
		t.Errorf("Expected adapter name 'fingerprint', got '%s'", status.AdapterName)
	}

	if status.Status != AdapterStatusActive {
		t.Errorf("Expected status '%s', got '%s'", AdapterStatusActive, status.Status)
	}

	if status.ErrorMessage != "" {
		t.Errorf("Expected empty error message, got '%s'", status.ErrorMessage)
	}

	if status.LastEvent != nil {
		t.Error("Expected LastEvent to be nil for new adapter")
	}
}

func TestGetAdapterStatusWithError(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	errorMessage := "Hardware communication failed"
	err := db.SetAdapterStatus("rfid", AdapterStatusError, errorMessage)
	if err != nil {
		t.Fatalf("Failed to set adapter status: %v", err)
	}

	status, err := db.GetAdapterStatus("rfid")
	if err != nil {
		t.Fatalf("Failed to get adapter status: %v", err)
	}

	if status.Status != AdapterStatusError {
		t.Errorf("Expected status '%s', got '%s'", AdapterStatusError, status.Status)
	}

	if status.ErrorMessage != errorMessage {
		t.Errorf("Expected error message '%s', got '%s'", errorMessage, status.ErrorMessage)
	}
}

func TestGetNonExistentAdapterStatus(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	_, err := db.GetAdapterStatus("non_existent")
	if err == nil {
		t.Error("Expected error when getting non-existent adapter status")
	}
}

func TestUpdateAdapterLastEvent(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set initial adapter status
	err := db.SetAdapterStatus("fingerprint", AdapterStatusActive, "")
	if err != nil {
		t.Fatalf("Failed to set adapter status: %v", err)
	}

	// Update last event time
	eventTime := time.Now()
	err = db.UpdateAdapterLastEvent("fingerprint", eventTime)
	if err != nil {
		t.Fatalf("Failed to update adapter last event: %v", err)
	}

	// Verify last event time was updated
	status, err := db.GetAdapterStatus("fingerprint")
	if err != nil {
		t.Fatalf("Failed to get adapter status: %v", err)
	}

	if status.LastEvent == nil {
		t.Fatal("Expected LastEvent to be set")
	}

	// Allow for small time differences due to database precision
	timeDiff := status.LastEvent.Sub(eventTime)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Expected last event time to be close to %v, got %v", eventTime, *status.LastEvent)
	}
}

func TestGetAllAdapterStatuses(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set multiple adapter statuses
	adapters := map[string]string{
		"fingerprint": AdapterStatusActive,
		"rfid":        AdapterStatusError,
		"simulator":   AdapterStatusDisabled,
	}

	for name, status := range adapters {
		err := db.SetAdapterStatus(name, status, "")
		if err != nil {
			t.Fatalf("Failed to set adapter status for %s: %v", name, err)
		}
	}

	// Get all adapter statuses
	allStatuses, err := db.GetAllAdapterStatuses()
	if err != nil {
		t.Fatalf("Failed to get all adapter statuses: %v", err)
	}

	if len(allStatuses) != len(adapters) {
		t.Errorf("Expected %d adapter statuses, got %d", len(adapters), len(allStatuses))
	}

	// Verify all adapters are present with correct status
	statusMap := make(map[string]string)
	for _, status := range allStatuses {
		statusMap[status.AdapterName] = status.Status
	}

	for name, expectedStatus := range adapters {
		actualStatus, exists := statusMap[name]
		if !exists {
			t.Errorf("Expected adapter %s to be in results", name)
			continue
		}
		if actualStatus != expectedStatus {
			t.Errorf("Expected adapter %s to have status %s, got %s", name, expectedStatus, actualStatus)
		}
	}
}

func TestGetActiveAdapters(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set multiple adapter statuses
	adapters := []struct {
		name   string
		status string
	}{
		{"fingerprint", AdapterStatusActive},
		{"rfid", AdapterStatusActive},
		{"simulator", AdapterStatusDisabled},
		{"webhook", AdapterStatusError},
	}

	for _, adapter := range adapters {
		err := db.SetAdapterStatus(adapter.name, adapter.status, "")
		if err != nil {
			t.Fatalf("Failed to set adapter status for %s: %v", adapter.name, err)
		}
	}

	// Get active adapters
	activeAdapters, err := db.GetActiveAdapters()
	if err != nil {
		t.Fatalf("Failed to get active adapters: %v", err)
	}

	expectedActive := []string{"fingerprint", "rfid"}
	if len(activeAdapters) != len(expectedActive) {
		t.Errorf("Expected %d active adapters, got %d", len(expectedActive), len(activeAdapters))
	}

	// Verify correct adapters are returned (should be sorted)
	for i, expectedName := range expectedActive {
		if i >= len(activeAdapters) {
			t.Errorf("Missing expected active adapter: %s", expectedName)
			continue
		}
		if activeAdapters[i] != expectedName {
			t.Errorf("Expected active adapter %s at position %d, got %s", expectedName, i, activeAdapters[i])
		}
	}
}

func TestDeleteAdapterStatus(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set adapter status
	err := db.SetAdapterStatus("temp_adapter", AdapterStatusActive, "")
	if err != nil {
		t.Fatalf("Failed to set adapter status: %v", err)
	}

	// Verify it exists
	_, err = db.GetAdapterStatus("temp_adapter")
	if err != nil {
		t.Fatalf("Expected adapter to exist before deletion: %v", err)
	}

	// Delete adapter status
	err = db.DeleteAdapterStatus("temp_adapter")
	if err != nil {
		t.Fatalf("Failed to delete adapter status: %v", err)
	}

	// Verify it no longer exists
	_, err = db.GetAdapterStatus("temp_adapter")
	if err == nil {
		t.Error("Expected error when getting deleted adapter status")
	}
}

func TestAdapterStatusReplacement(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Set initial status
	err := db.SetAdapterStatus("test_adapter", AdapterStatusActive, "")
	if err != nil {
		t.Fatalf("Failed to set initial adapter status: %v", err)
	}

	// Update status
	err = db.SetAdapterStatus("test_adapter", AdapterStatusError, "New error occurred")
	if err != nil {
		t.Fatalf("Failed to update adapter status: %v", err)
	}

	// Verify updated status
	status, err := db.GetAdapterStatus("test_adapter")
	if err != nil {
		t.Fatalf("Failed to get updated adapter status: %v", err)
	}

	if status.Status != AdapterStatusError {
		t.Errorf("Expected updated status '%s', got '%s'", AdapterStatusError, status.Status)
	}

	if status.ErrorMessage != "New error occurred" {
		t.Errorf("Expected updated error message 'New error occurred', got '%s'", status.ErrorMessage)
	}
}