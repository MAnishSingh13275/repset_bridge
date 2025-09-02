package queue

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gym-door-bridge/internal/database"
	"gym-door-bridge/internal/types"
)

func setupTestDB(t *testing.T) *database.DB {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	// Generate test encryption key
	encryptionKey := make([]byte, 32)
	for i := range encryptionKey {
		encryptionKey[i] = byte(i)
	}
	
	config := database.Config{
		DatabasePath:    dbPath,
		EncryptionKey:   encryptionKey,
		PerformanceTier: database.TierNormal,
	}
	
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	return db
}

func TestNewSQLiteQueueManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	qm := NewSQLiteQueueManager(db)
	if qm == nil {
		t.Fatal("Expected non-nil queue manager")
	}
}

func TestQueueManager_Initialize(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	qm := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	tests := []struct {
		name    string
		config  QueueConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: QueueConfig{
				MaxSize:         1000,
				BatchSize:       50,
				RetryInterval:   30 * time.Second,
				MaxRetries:      5,
				RetentionPolicy: RetentionPolicyFIFO,
			},
			wantErr: false,
		},
		{
			name: "invalid max size",
			config: QueueConfig{
				MaxSize:       0,
				BatchSize:     50,
				RetryInterval: 30 * time.Second,
				MaxRetries:    5,
			},
			wantErr: true,
		},
		{
			name: "invalid batch size",
			config: QueueConfig{
				MaxSize:       1000,
				BatchSize:     0,
				RetryInterval: 30 * time.Second,
				MaxRetries:    5,
			},
			wantErr: true,
		},
		{
			name: "invalid retry interval",
			config: QueueConfig{
				MaxSize:       1000,
				BatchSize:     50,
				RetryInterval: 0,
				MaxRetries:    5,
			},
			wantErr: true,
		},
		{
			name: "invalid max retries",
			config: QueueConfig{
				MaxSize:       1000,
				BatchSize:     50,
				RetryInterval: 30 * time.Second,
				MaxRetries:    -1,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qm.Initialize(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQueueManager_EnqueueAndGetPending(t *testing.T) {
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
	
	// Test enqueuing events
	events := []types.StandardEvent{
		{
			EventID:        "event1",
			ExternalUserID: "user1",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
			RawData:        map[string]interface{}{"test": "data1"},
		},
		{
			EventID:        "event2",
			ExternalUserID: "user2",
			Timestamp:      time.Now(),
			EventType:      types.EventTypeExit,
			IsSimulated:    true,
			DeviceID:       "device1",
			RawData:        map[string]interface{}{"test": "data2"},
		},
	}
	
	for _, event := range events {
		err := qm.Enqueue(ctx, event)
		if err != nil {
			t.Fatalf("Failed to enqueue event %s: %v", event.EventID, err)
		}
	}
	
	// Test getting pending events
	pendingEvents, err := qm.GetPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get pending events: %v", err)
	}
	
	if len(pendingEvents) != 2 {
		t.Errorf("Expected 2 pending events, got %d", len(pendingEvents))
	}
	
	// Verify event data
	for i, pendingEvent := range pendingEvents {
		originalEvent := events[i]
		if pendingEvent.Event.EventID != originalEvent.EventID {
			t.Errorf("Event %d: expected EventID %s, got %s", i, originalEvent.EventID, pendingEvent.Event.EventID)
		}
		if pendingEvent.Event.ExternalUserID != originalEvent.ExternalUserID {
			t.Errorf("Event %d: expected ExternalUserID %s, got %s", i, originalEvent.ExternalUserID, pendingEvent.Event.ExternalUserID)
		}
		if pendingEvent.Event.EventType != originalEvent.EventType {
			t.Errorf("Event %d: expected EventType %s, got %s", i, originalEvent.EventType, pendingEvent.Event.EventType)
		}
		if pendingEvent.Event.IsSimulated != originalEvent.IsSimulated {
			t.Errorf("Event %d: expected IsSimulated %v, got %v", i, originalEvent.IsSimulated, pendingEvent.Event.IsSimulated)
		}
	}
}

func TestQueueManager_MarkEventsSent(t *testing.T) {
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
	
	// Enqueue test events
	event := types.StandardEvent{
		EventID:        "event1",
		ExternalUserID: "user1",
		Timestamp:      time.Now(),
		EventType:      types.EventTypeEntry,
		IsSimulated:    false,
		DeviceID:       "device1",
	}
	
	err = qm.Enqueue(ctx, event)
	if err != nil {
		t.Fatalf("Failed to enqueue event: %v", err)
	}
	
	// Get pending events
	pendingEvents, err := qm.GetPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get pending events: %v", err)
	}
	
	if len(pendingEvents) != 1 {
		t.Fatalf("Expected 1 pending event, got %d", len(pendingEvents))
	}
	
	// Mark event as sent
	eventIDs := []int64{pendingEvents[0].ID}
	err = qm.MarkEventsSent(ctx, eventIDs)
	if err != nil {
		t.Fatalf("Failed to mark events as sent: %v", err)
	}
	
	// Verify no pending events remain
	pendingEvents, err = qm.GetPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get pending events after marking sent: %v", err)
	}
	
	if len(pendingEvents) != 0 {
		t.Errorf("Expected 0 pending events after marking sent, got %d", len(pendingEvents))
	}
}

func TestQueueManager_QueueDepthAndCapacity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	qm := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	config := QueueConfig{
		MaxSize:         3, // Small size for testing
		BatchSize:       50,
		RetryInterval:   30 * time.Second,
		MaxRetries:      5,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	
	err := qm.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize queue manager: %v", err)
	}
	
	// Test initial queue depth
	depth, err := qm.GetQueueDepth(ctx)
	if err != nil {
		t.Fatalf("Failed to get queue depth: %v", err)
	}
	if depth != 0 {
		t.Errorf("Expected initial queue depth 0, got %d", depth)
	}
	
	// Test queue not full initially
	isFull, err := qm.IsQueueFull(ctx)
	if err != nil {
		t.Fatalf("Failed to check if queue is full: %v", err)
	}
	if isFull {
		t.Error("Expected queue not to be full initially")
	}
	
	// Add events up to capacity
	for i := 0; i < 3; i++ {
		event := types.StandardEvent{
			EventID:        fmt.Sprintf("event%d", i+1),
			ExternalUserID: fmt.Sprintf("user%d", i+1),
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		}
		
		err := qm.Enqueue(ctx, event)
		if err != nil {
			t.Fatalf("Failed to enqueue event %d: %v", i+1, err)
		}
	}
	
	// Check queue is now full
	isFull, err = qm.IsQueueFull(ctx)
	if err != nil {
		t.Fatalf("Failed to check if queue is full: %v", err)
	}
	if !isFull {
		t.Error("Expected queue to be full after adding 3 events")
	}
	
	depth, err = qm.GetQueueDepth(ctx)
	if err != nil {
		t.Fatalf("Failed to get queue depth: %v", err)
	}
	if depth != 3 {
		t.Errorf("Expected queue depth 3, got %d", depth)
	}
}

func TestQueueManager_FIFOEviction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	qm := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	config := QueueConfig{
		MaxSize:         2, // Very small size for testing eviction
		BatchSize:       50,
		RetryInterval:   30 * time.Second,
		MaxRetries:      5,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	
	err := qm.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize queue manager: %v", err)
	}
	
	// Add events to fill queue
	events := []types.StandardEvent{
		{
			EventID:        "event1",
			ExternalUserID: "user1",
			Timestamp:      time.Now().Add(-2 * time.Minute), // Oldest
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
		{
			EventID:        "event2",
			ExternalUserID: "user2",
			Timestamp:      time.Now().Add(-1 * time.Minute), // Middle
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
	}
	
	for _, event := range events {
		err := qm.Enqueue(ctx, event)
		if err != nil {
			t.Fatalf("Failed to enqueue event %s: %v", event.EventID, err)
		}
	}
	
	// Add one more event to trigger eviction
	newEvent := types.StandardEvent{
		EventID:        "event3",
		ExternalUserID: "user3",
		Timestamp:      time.Now(), // Newest
		EventType:      types.EventTypeEntry,
		IsSimulated:    false,
		DeviceID:       "device1",
	}
	
	err = qm.Enqueue(ctx, newEvent)
	if err != nil {
		t.Fatalf("Failed to enqueue event that should trigger eviction: %v", err)
	}
	
	// Verify queue still has max size
	depth, err := qm.GetQueueDepth(ctx)
	if err != nil {
		t.Fatalf("Failed to get queue depth: %v", err)
	}
	if depth != 2 {
		t.Errorf("Expected queue depth 2 after eviction, got %d", depth)
	}
	
	// Verify the oldest event was evicted and newest is present
	pendingEvents, err := qm.GetPendingEvents(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get pending events: %v", err)
	}
	
	if len(pendingEvents) != 2 {
		t.Fatalf("Expected 2 pending events, got %d", len(pendingEvents))
	}
	
	// Check that event1 (oldest) was evicted and event3 (newest) is present
	eventIDs := make(map[string]bool)
	for _, event := range pendingEvents {
		eventIDs[event.Event.EventID] = true
	}
	
	if eventIDs["event1"] {
		t.Error("Expected oldest event (event1) to be evicted")
	}
	if !eventIDs["event3"] {
		t.Error("Expected newest event (event3) to be present")
	}
}

func TestQueueManager_GetStats(t *testing.T) {
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
	
	// Test initial stats
	stats, err := qm.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	
	if stats.QueueDepth != 0 {
		t.Errorf("Expected initial queue depth 0, got %d", stats.QueueDepth)
	}
	if stats.PendingEvents != 0 {
		t.Errorf("Expected initial pending events 0, got %d", stats.PendingEvents)
	}
	
	// Add some events
	for i := 0; i < 3; i++ {
		event := types.StandardEvent{
			EventID:        fmt.Sprintf("event%d", i+1),
			ExternalUserID: fmt.Sprintf("user%d", i+1),
			Timestamp:      time.Now(),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		}
		
		err := qm.Enqueue(ctx, event)
		if err != nil {
			t.Fatalf("Failed to enqueue event %d: %v", i+1, err)
		}
	}
	
	// Test stats after adding events
	stats, err = qm.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats after adding events: %v", err)
	}
	
	if stats.QueueDepth != 3 {
		t.Errorf("Expected queue depth 3, got %d", stats.QueueDepth)
	}
	if stats.PendingEvents != 3 {
		t.Errorf("Expected pending events 3, got %d", stats.PendingEvents)
	}
}

func TestQueueManager_Cleanup(t *testing.T) {
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
	
	// Test cleanup doesn't error (detailed testing would require database inspection)
	err = qm.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}
}

func TestGetTierConfig(t *testing.T) {
	tests := []struct {
		tier           database.PerformanceTier
		expectedMaxSize int
	}{
		{database.TierLite, 1000},
		{database.TierNormal, 10000},
		{database.TierFull, 50000},
		{"invalid", 10000}, // Should default to Normal
	}
	
	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			config := GetTierConfig(tt.tier)
			if config.MaxSize != tt.expectedMaxSize {
				t.Errorf("Expected MaxSize %d for tier %s, got %d", tt.expectedMaxSize, tt.tier, config.MaxSize)
			}
			
			// Verify other config values are reasonable
			if config.BatchSize <= 0 {
				t.Errorf("Expected positive BatchSize, got %d", config.BatchSize)
			}
			if config.RetryInterval <= 0 {
				t.Errorf("Expected positive RetryInterval, got %v", config.RetryInterval)
			}
			if config.MaxRetries < 0 {
				t.Errorf("Expected non-negative MaxRetries, got %d", config.MaxRetries)
			}
			if config.RetentionPolicy != RetentionPolicyFIFO {
				t.Errorf("Expected RetentionPolicy %s, got %s", RetentionPolicyFIFO, config.RetentionPolicy)
			}
		})
	}
}

func TestQueueManager_Close(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	qm := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Test close doesn't error
	err := qm.Close(ctx)
	if err != nil {
		t.Fatalf("Failed to close queue manager: %v", err)
	}
}

