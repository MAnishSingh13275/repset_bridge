package processor

import (
	"context"
	"os"
	"testing"
	"time"

	"gym-door-bridge/internal/database"
	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
)

func TestEventProcessorIntegration(t *testing.T) {
	// Create a temporary database for testing
	tempFile, err := os.CreateTemp("", "test_processor_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Initialize database
	dbConfig := database.Config{
		DatabasePath:    tempFile.Name(),
		EncryptionKey:   []byte("test-key-32-bytes-long-for-aes!!"),
		PerformanceTier: database.TierNormal,
	}
	
	db, err := database.NewDB(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create processor with real database
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	processor := NewEventProcessorWithDB(db, logger)

	// Initialize processor
	processorConfig := ProcessorConfig{
		DeviceID:            "test-device-integration",
		EnableDeduplication: true,
		DeduplicationWindow: 300,
	}

	err = processor.Initialize(context.Background(), processorConfig)
	if err != nil {
		t.Fatalf("Failed to initialize processor: %v", err)
	}

	now := time.Now()
	rawEvent := types.RawHardwareEvent{
		ExternalUserID: "integration-user-123",
		Timestamp:      now,
		EventType:      types.EventTypeEntry,
		RawData: map[string]interface{}{
			"adapter": "IntegrationTestAdapter",
			"test":    true,
		},
	}

	t.Run("process and check deduplication", func(t *testing.T) {
		// Process the event first time
		result1, err := processor.ProcessEvent(context.Background(), rawEvent)
		if err != nil {
			t.Fatalf("Failed to process event: %v", err)
		}

		if !result1.Processed {
			t.Errorf("Expected first event to be processed, got: %s", result1.Reason)
		}

		// Insert the event into database to simulate it being queued
		dbEvent := &database.EventQueue{
			EventID:        result1.Event.EventID,
			ExternalUserID: result1.Event.ExternalUserID,
			Timestamp:      result1.Event.Timestamp,
			EventType:      result1.Event.EventType,
			IsSimulated:    result1.Event.IsSimulated,
		}

		err = db.InsertEvent(dbEvent)
		if err != nil {
			t.Fatalf("Failed to insert event into database: %v", err)
		}

		// Process the same event again - should be detected as duplicate
		result2, err := processor.ProcessEvent(context.Background(), rawEvent)
		if err != nil {
			t.Fatalf("Failed to process duplicate event: %v", err)
		}

		if result2.Processed {
			t.Error("Expected duplicate event to not be processed")
		}

		if result2.Reason != "duplicate event within deduplication window" {
			t.Errorf("Expected duplicate reason, got: %s", result2.Reason)
		}
	})

	t.Run("process different event", func(t *testing.T) {
		// Create a different event
		differentEvent := rawEvent
		differentEvent.ExternalUserID = "different-user-456"

		result, err := processor.ProcessEvent(context.Background(), differentEvent)
		if err != nil {
			t.Fatalf("Failed to process different event: %v", err)
		}

		if !result.Processed {
			t.Errorf("Expected different event to be processed, got: %s", result.Reason)
		}

		// Verify the event has correct metadata
		if result.Event.DeviceID != processorConfig.DeviceID {
			t.Errorf("Expected DeviceID %s, got %s", processorConfig.DeviceID, result.Event.DeviceID)
		}

		if result.Event.EventID == "" {
			t.Error("Expected EventID to be generated")
		}
	})

	t.Run("user mapping resolution integration", func(t *testing.T) {
		// Create a user mapping in the database
		_, err := db.CreateExternalUserMapping("integration-user-456", "internal-user-789", "Integration Test User", "Test mapping")
		if err != nil {
			t.Fatalf("Failed to create user mapping: %v", err)
		}

		// Create an event with the mapped external user ID
		mappedEvent := types.RawHardwareEvent{
			ExternalUserID: "integration-user-456",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			RawData: map[string]interface{}{
				"adapter": "IntegrationTestAdapter",
				"test":    true,
			},
		}

		// Process the event
		result, err := processor.ProcessEvent(context.Background(), mappedEvent)
		if err != nil {
			t.Fatalf("Failed to process mapped event: %v", err)
		}

		if !result.Processed {
			t.Errorf("Expected mapped event to be processed, got: %s", result.Reason)
		}

		// Verify the internal user ID was resolved
		if result.Event.InternalUserID != "internal-user-789" {
			t.Errorf("Expected InternalUserID to be 'internal-user-789', got '%s'", result.Event.InternalUserID)
		}

		// Create an event with an unmapped external user ID
		unmappedEvent := types.RawHardwareEvent{
			ExternalUserID: "unmapped-user-999",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			RawData: map[string]interface{}{
				"adapter": "IntegrationTestAdapter",
				"test":    true,
			},
		}

		// Process the unmapped event
		result2, err := processor.ProcessEvent(context.Background(), unmappedEvent)
		if err != nil {
			t.Fatalf("Failed to process unmapped event: %v", err)
		}

		if !result2.Processed {
			t.Errorf("Expected unmapped event to be processed, got: %s", result2.Reason)
		}

		// Verify the internal user ID is empty for unmapped user
		if result2.Event.InternalUserID != "" {
			t.Errorf("Expected InternalUserID to be empty for unmapped user, got '%s'", result2.Event.InternalUserID)
		}
	})

	t.Run("statistics tracking", func(t *testing.T) {
		stats := processor.GetStats()
		
		// We should have processed at least 4 events now (original 2 + 2 from user mapping test)
		if stats.TotalProcessed < 4 {
			t.Errorf("Expected at least 4 processed events, got %d", stats.TotalProcessed)
		}

		if stats.TotalDuplicates < 1 {
			t.Errorf("Expected at least 1 duplicate event, got %d", stats.TotalDuplicates)
		}

		if stats.LastProcessedAt == 0 {
			t.Error("Expected LastProcessedAt to be set")
		}
	})
}