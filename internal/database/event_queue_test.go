package database

import (
	"fmt"
	"testing"
	"time"
)

func TestInsertEvent(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	event := &EventQueue{
		EventID:        "test-event-1",
		ExternalUserID: "user123",
		Timestamp:      time.Now(),
		EventType:      EventTypeEntry,
		IsSimulated:    false,
		RawData:        `{"fingerprint_id": "fp123", "confidence": 0.95}`,
	}

	err := db.InsertEvent(event)
	if err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	if event.ID == 0 {
		t.Error("Expected event ID to be set after insert")
	}
}

func TestGetUnsentEvents(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Insert test events
	events := []*EventQueue{
		{
			EventID:        "event-1",
			ExternalUserID: "user1",
			Timestamp:      time.Now().Add(-2 * time.Hour),
			EventType:      EventTypeEntry,
			IsSimulated:    false,
			RawData:        `{"test": "data1"}`,
		},
		{
			EventID:        "event-2",
			ExternalUserID: "user2",
			Timestamp:      time.Now().Add(-1 * time.Hour),
			EventType:      EventTypeExit,
			IsSimulated:    true,
		},
	}

	for _, event := range events {
		if err := db.InsertEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	// Retrieve unsent events
	unsentEvents, err := db.GetUnsentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get unsent events: %v", err)
	}

	if len(unsentEvents) != 2 {
		t.Errorf("Expected 2 unsent events, got %d", len(unsentEvents))
	}

	// Check ordering (should be chronological)
	if unsentEvents[0].EventID != "event-1" {
		t.Errorf("Expected first event to be event-1, got %s", unsentEvents[0].EventID)
	}

	// Check raw data decryption
	if unsentEvents[0].RawData != `{"test": "data1"}` {
		t.Errorf("Expected raw data to be decrypted correctly, got %s", unsentEvents[0].RawData)
	}
}

func TestMarkEventsSent(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Insert test event
	event := &EventQueue{
		EventID:        "test-event",
		ExternalUserID: "user123",
		Timestamp:      time.Now(),
		EventType:      EventTypeEntry,
	}

	if err := db.InsertEvent(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Mark as sent
	err := db.MarkEventsSent([]string{"test-event"})
	if err != nil {
		t.Fatalf("Failed to mark event as sent: %v", err)
	}

	// Verify no unsent events
	unsentEvents, err := db.GetUnsentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get unsent events: %v", err)
	}

	if len(unsentEvents) != 0 {
		t.Errorf("Expected 0 unsent events after marking as sent, got %d", len(unsentEvents))
	}
}

func TestIncrementRetryCount(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Insert test event
	event := &EventQueue{
		EventID:        "test-event",
		ExternalUserID: "user123",
		Timestamp:      time.Now(),
		EventType:      EventTypeEntry,
	}

	if err := db.InsertEvent(event); err != nil {
		t.Fatalf("Failed to insert event: %v", err)
	}

	// Increment retry count
	err := db.IncrementRetryCount([]string{"test-event"})
	if err != nil {
		t.Fatalf("Failed to increment retry count: %v", err)
	}

	// Verify retry count was incremented
	unsentEvents, err := db.GetUnsentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get unsent events: %v", err)
	}

	if len(unsentEvents) != 1 {
		t.Fatalf("Expected 1 unsent event, got %d", len(unsentEvents))
	}

	if unsentEvents[0].RetryCount != 1 {
		t.Errorf("Expected retry count to be 1, got %d", unsentEvents[0].RetryCount)
	}
}

func TestGetQueueDepth(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Initially should be 0
	depth, err := db.GetQueueDepth()
	if err != nil {
		t.Fatalf("Failed to get queue depth: %v", err)
	}
	if depth != 0 {
		t.Errorf("Expected initial queue depth to be 0, got %d", depth)
	}

	// Insert events
	for i := 0; i < 3; i++ {
		event := &EventQueue{
			EventID:        fmt.Sprintf("event-%d", i),
			ExternalUserID: "user123",
			Timestamp:      time.Now(),
			EventType:      EventTypeEntry,
		}
		if err := db.InsertEvent(event); err != nil {
			t.Fatalf("Failed to insert event: %v", err)
		}
	}

	// Check depth
	depth, err = db.GetQueueDepth()
	if err != nil {
		t.Fatalf("Failed to get queue depth: %v", err)
	}
	if depth != 3 {
		t.Errorf("Expected queue depth to be 3, got %d", depth)
	}

	// Mark one as sent
	err = db.MarkEventsSent([]string{"event-0"})
	if err != nil {
		t.Fatalf("Failed to mark event as sent: %v", err)
	}

	// Check depth again
	depth, err = db.GetQueueDepth()
	if err != nil {
		t.Fatalf("Failed to get queue depth: %v", err)
	}
	if depth != 2 {
		t.Errorf("Expected queue depth to be 2 after marking one as sent, got %d", depth)
	}
}

func TestCleanupOldEvents(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Insert old sent event
	oldEvent := &EventQueue{
		EventID:        "old-event",
		ExternalUserID: "user123",
		Timestamp:      time.Now().Add(-48 * time.Hour),
		EventType:      EventTypeEntry,
	}
	if err := db.InsertEvent(oldEvent); err != nil {
		t.Fatalf("Failed to insert old event: %v", err)
	}

	// Mark as sent
	if err := db.MarkEventsSent([]string{"old-event"}); err != nil {
		t.Fatalf("Failed to mark old event as sent: %v", err)
	}

	// Insert recent event
	recentEvent := &EventQueue{
		EventID:        "recent-event",
		ExternalUserID: "user123",
		Timestamp:      time.Now(),
		EventType:      EventTypeEntry,
	}
	if err := db.InsertEvent(recentEvent); err != nil {
		t.Fatalf("Failed to insert recent event: %v", err)
	}

	// Cleanup events older than 24 hours
	err := db.CleanupOldEvents(24 * time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup old events: %v", err)
	}

	// Verify recent unsent event still exists
	unsentEvents, err := db.GetUnsentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get unsent events: %v", err)
	}

	if len(unsentEvents) != 1 {
		t.Errorf("Expected 1 unsent event after cleanup, got %d", len(unsentEvents))
	}

	if unsentEvents[0].EventID != "recent-event" {
		t.Errorf("Expected recent event to remain, got %s", unsentEvents[0].EventID)
	}
}

func TestEvictOldestEvents(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Insert multiple events with different timestamps
	for i := 0; i < 5; i++ {
		event := &EventQueue{
			EventID:        fmt.Sprintf("event-%d", i),
			ExternalUserID: "user123",
			Timestamp:      time.Now().Add(time.Duration(-i) * time.Hour),
			EventType:      EventTypeEntry,
		}
		if err := db.InsertEvent(event); err != nil {
			t.Fatalf("Failed to insert event %d: %v", i, err)
		}
	}

	// Evict to keep only 3 events
	err := db.EvictOldestEvents(3)
	if err != nil {
		t.Fatalf("Failed to evict oldest events: %v", err)
	}

	// Check remaining events
	unsentEvents, err := db.GetUnsentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get unsent events: %v", err)
	}

	if len(unsentEvents) != 3 {
		t.Errorf("Expected 3 events after eviction, got %d", len(unsentEvents))
	}

	// Verify the newest events remain (event-0, event-1, event-2)
	expectedEvents := []string{"event-2", "event-1", "event-0"}
	for i, event := range unsentEvents {
		if event.EventID != expectedEvents[i] {
			t.Errorf("Expected event %s at position %d, got %s", expectedEvents[i], i, event.EventID)
		}
	}
}