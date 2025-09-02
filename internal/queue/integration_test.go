package queue

import (
	"context"
	"testing"
	"time"

	"gym-door-bridge/internal/database"
	"gym-door-bridge/internal/types"
)

// TestQueueManager_BatchProcessing tests the complete workflow of enqueueing,
// batch processing, and marking events as sent
func TestQueueManager_BatchProcessing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	qm := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Initialize with small batch size for testing
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       3, // Small batch for testing
		RetryInterval:   30 * time.Second,
		MaxRetries:      5,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	
	err := qm.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize queue manager: %v", err)
	}
	
	// Enqueue multiple events
	events := []types.StandardEvent{
		{
			EventID:        "batch_event_1",
			ExternalUserID: "user1",
			Timestamp:      time.Now().Add(-3 * time.Minute),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
			RawData:        map[string]interface{}{"location": "front_door"},
		},
		{
			EventID:        "batch_event_2",
			ExternalUserID: "user2",
			Timestamp:      time.Now().Add(-2 * time.Minute),
			EventType:      types.EventTypeExit,
			IsSimulated:    false,
			DeviceID:       "device1",
			RawData:        map[string]interface{}{"location": "back_door"},
		},
		{
			EventID:        "batch_event_3",
			ExternalUserID: "user3",
			Timestamp:      time.Now().Add(-1 * time.Minute),
			EventType:      types.EventTypeEntry,
			IsSimulated:    true,
			DeviceID:       "device1",
		},
		{
			EventID:        "batch_event_4",
			ExternalUserID: "user4",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeDenied,
			IsSimulated:    false,
			DeviceID:       "device1",
			RawData:        map[string]interface{}{"reason": "invalid_card"},
		},
	}
	
	// Enqueue all events
	for _, event := range events {
		err := qm.Enqueue(ctx, event)
		if err != nil {
			t.Fatalf("Failed to enqueue event %s: %v", event.EventID, err)
		}
	}
	
	// Verify all events are pending
	stats, err := qm.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	
	if stats.QueueDepth != 4 {
		t.Errorf("Expected queue depth 4, got %d", stats.QueueDepth)
	}
	
	// Process first batch
	batch1, err := qm.GetPendingEvents(ctx, config.BatchSize)
	if err != nil {
		t.Fatalf("Failed to get first batch: %v", err)
	}
	
	if len(batch1) != 3 {
		t.Errorf("Expected first batch size 3, got %d", len(batch1))
	}
	
	// Verify events are in chronological order (oldest first)
	for i := 1; i < len(batch1); i++ {
		if batch1[i].Event.Timestamp.Before(batch1[i-1].Event.Timestamp) {
			t.Errorf("Events not in chronological order: event %d timestamp %v is before event %d timestamp %v",
				i, batch1[i].Event.Timestamp, i-1, batch1[i-1].Event.Timestamp)
		}
	}
	
	// Mark first batch as sent
	batch1IDs := make([]int64, len(batch1))
	for i, event := range batch1 {
		batch1IDs[i] = event.ID
	}
	
	err = qm.MarkEventsSent(ctx, batch1IDs)
	if err != nil {
		t.Fatalf("Failed to mark first batch as sent: %v", err)
	}
	
	// Verify queue depth decreased
	stats, err = qm.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats after first batch: %v", err)
	}
	
	if stats.QueueDepth != 1 {
		t.Errorf("Expected queue depth 1 after first batch, got %d", stats.QueueDepth)
	}
	
	// Process second batch (should only have 1 event)
	batch2, err := qm.GetPendingEvents(ctx, config.BatchSize)
	if err != nil {
		t.Fatalf("Failed to get second batch: %v", err)
	}
	
	if len(batch2) != 1 {
		t.Errorf("Expected second batch size 1, got %d", len(batch2))
	}
	
	// Verify it's the last event
	if batch2[0].Event.EventID != "batch_event_4" {
		t.Errorf("Expected last event to be batch_event_4, got %s", batch2[0].Event.EventID)
	}
	
	// Mark second batch as sent
	batch2IDs := []int64{batch2[0].ID}
	err = qm.MarkEventsSent(ctx, batch2IDs)
	if err != nil {
		t.Fatalf("Failed to mark second batch as sent: %v", err)
	}
	
	// Verify queue is empty
	stats, err = qm.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get final stats: %v", err)
	}
	
	if stats.QueueDepth != 0 {
		t.Errorf("Expected final queue depth 0, got %d", stats.QueueDepth)
	}
	
	// Verify no more pending events
	finalBatch, err := qm.GetPendingEvents(ctx, config.BatchSize)
	if err != nil {
		t.Fatalf("Failed to get final batch: %v", err)
	}
	
	if len(finalBatch) != 0 {
		t.Errorf("Expected no pending events, got %d", len(finalBatch))
	}
}

// TestQueueManager_EncryptedPayloads tests that raw data is properly encrypted
func TestQueueManager_EncryptedPayloads(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	qm := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       50,
		RetryInterval:   30 * time.Second,
		MaxRetries:      5,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	
	err := qm.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize queue manager: %v", err)
	}
	
	// Create event with sensitive raw data
	sensitiveData := map[string]interface{}{
		"fingerprint_hash": "abc123def456",
		"card_number":      "1234567890",
		"access_code":      "secret123",
	}
	
	event := types.StandardEvent{
		EventID:        "encrypted_event",
		ExternalUserID: "sensitive_user",
		Timestamp:      time.Now(),
		EventType:      types.EventTypeEntry,
		IsSimulated:    false,
		DeviceID:       "device1",
		RawData:        sensitiveData,
	}
	
	// Enqueue event
	err = qm.Enqueue(ctx, event)
	if err != nil {
		t.Fatalf("Failed to enqueue event: %v", err)
	}
	
	// Retrieve event
	pendingEvents, err := qm.GetPendingEvents(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get pending events: %v", err)
	}
	
	if len(pendingEvents) != 1 {
		t.Fatalf("Expected 1 pending event, got %d", len(pendingEvents))
	}
	
	retrievedEvent := pendingEvents[0]
	
	// Verify the event data was properly decrypted and matches original
	if retrievedEvent.Event.EventID != event.EventID {
		t.Errorf("EventID mismatch: expected %s, got %s", event.EventID, retrievedEvent.Event.EventID)
	}
	
	if retrievedEvent.Event.ExternalUserID != event.ExternalUserID {
		t.Errorf("ExternalUserID mismatch: expected %s, got %s", event.ExternalUserID, retrievedEvent.Event.ExternalUserID)
	}
	
	// Verify raw data was properly decrypted
	if retrievedEvent.Event.RawData == nil {
		t.Fatal("Expected raw data to be present")
	}
	
	if retrievedEvent.Event.RawData["fingerprint_hash"] != sensitiveData["fingerprint_hash"] {
		t.Errorf("Fingerprint hash mismatch: expected %v, got %v", 
			sensitiveData["fingerprint_hash"], retrievedEvent.Event.RawData["fingerprint_hash"])
	}
	
	if retrievedEvent.Event.RawData["card_number"] != sensitiveData["card_number"] {
		t.Errorf("Card number mismatch: expected %v, got %v", 
			sensitiveData["card_number"], retrievedEvent.Event.RawData["card_number"])
	}
	
	// Verify the event is marked as encrypted
	if !retrievedEvent.IsEncrypted {
		t.Error("Expected event to be marked as encrypted")
	}
}

// TestQueueManager_TierConfigurations tests different performance tier configurations
func TestQueueManager_TierConfigurations(t *testing.T) {
	tiers := []struct {
		name     string
		tier     database.PerformanceTier
		maxSize  int
		batchSize int
	}{
		{"Lite", database.TierLite, 1000, 10},
		{"Normal", database.TierNormal, 10000, 50},
		{"Full", database.TierFull, 50000, 100},
	}
	
	for _, tt := range tiers {
		t.Run(tt.name, func(t *testing.T) {
			config := GetTierConfig(tt.tier)
			
			if config.MaxSize != tt.maxSize {
				t.Errorf("Expected MaxSize %d for %s tier, got %d", tt.maxSize, tt.name, config.MaxSize)
			}
			
			if config.BatchSize != tt.batchSize {
				t.Errorf("Expected BatchSize %d for %s tier, got %d", tt.batchSize, tt.name, config.BatchSize)
			}
			
			// Verify configuration is valid
			if config.MaxSize <= 0 {
				t.Errorf("Invalid MaxSize for %s tier: %d", tt.name, config.MaxSize)
			}
			
			if config.BatchSize <= 0 {
				t.Errorf("Invalid BatchSize for %s tier: %d", tt.name, config.BatchSize)
			}
			
			if config.RetryInterval <= 0 {
				t.Errorf("Invalid RetryInterval for %s tier: %v", tt.name, config.RetryInterval)
			}
			
			if config.MaxRetries < 0 {
				t.Errorf("Invalid MaxRetries for %s tier: %d", tt.name, config.MaxRetries)
			}
		})
	}
}