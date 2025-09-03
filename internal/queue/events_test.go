package queue

import (
	"context"
	"testing"
	"time"

	"gym-door-bridge/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteQueueManager_QueryEvents(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()
	
	queueManager := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Initialize queue manager
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       10,
		RetryInterval:   30 * time.Second,
		MaxRetries:      3,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	err := queueManager.Initialize(ctx, config)
	require.NoError(t, err)
	
	// Create test events with different characteristics
	now := time.Now()
	testEvents := []types.StandardEvent{
		{
			EventID:        "event1",
			ExternalUserID: "user1",
			Timestamp:      now.Add(-3 * time.Hour),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
		{
			EventID:        "event2",
			ExternalUserID: "user2",
			Timestamp:      now.Add(-2 * time.Hour),
			EventType:      types.EventTypeExit,
			IsSimulated:    true,
			DeviceID:       "device1",
		},
		{
			EventID:        "event3",
			ExternalUserID: "user1",
			Timestamp:      now.Add(-1 * time.Hour),
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
		{
			EventID:        "event4",
			ExternalUserID: "user3",
			Timestamp:      now,
			EventType:      types.EventTypeDenied,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
	}
	
	// Enqueue test events
	for _, event := range testEvents {
		err := queueManager.Enqueue(ctx, event)
		require.NoError(t, err)
	}
	
	// Mark some events as sent
	pendingEvents, err := queueManager.GetPendingEvents(ctx, 2)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 2)
	
	sentEventIDs := []int64{pendingEvents[0].ID}
	err = queueManager.MarkEventsSent(ctx, sentEventIDs)
	require.NoError(t, err)
	
	// Mark one event as failed by incrementing retry count to max
	failedEventIDs := []int64{pendingEvents[1].ID}
	for i := 0; i < 3; i++ { // Increment retry count to reach max retries (3)
		err = queueManager.MarkEventsFailed(ctx, failedEventIDs, "Test failure")
		require.NoError(t, err)
	}
	
	tests := []struct {
		name           string
		filter         EventQueryFilter
		expectedCount  int
		expectedTotal  int64
		validateResult func(t *testing.T, events []QueuedEvent)
	}{
		{
			name: "query all events",
			filter: EventQueryFilter{
				Limit:     10,
				Offset:    0,
				SortBy:    "timestamp",
				SortOrder: "desc",
			},
			expectedCount: 4,
			expectedTotal: 4,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				// Should be sorted by timestamp descending
				assert.Equal(t, "event4", events[0].Event.EventID)
				assert.Equal(t, "event3", events[1].Event.EventID)
				assert.Equal(t, "event2", events[2].Event.EventID)
				assert.Equal(t, "event1", events[3].Event.EventID)
			},
		},
		{
			name: "query with event type filter",
			filter: EventQueryFilter{
				EventType: types.EventTypeEntry,
				Limit:     10,
				Offset:    0,
				SortBy:    "timestamp",
				SortOrder: "asc",
			},
			expectedCount: 2,
			expectedTotal: 2,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				assert.Equal(t, "event1", events[0].Event.EventID)
				assert.Equal(t, "event3", events[1].Event.EventID)
				for _, event := range events {
					assert.Equal(t, types.EventTypeEntry, event.Event.EventType)
				}
			},
		},
		{
			name: "query with user ID filter",
			filter: EventQueryFilter{
				UserID:    "user1",
				Limit:     10,
				Offset:    0,
				SortBy:    "timestamp",
				SortOrder: "desc",
			},
			expectedCount: 2,
			expectedTotal: 2,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				for _, event := range events {
					assert.Equal(t, "user1", event.Event.ExternalUserID)
				}
			},
		},
		{
			name: "query with simulation filter",
			filter: EventQueryFilter{
				IsSimulated: func() *bool { b := true; return &b }(),
				Limit:       10,
				Offset:      0,
				SortBy:      "timestamp",
				SortOrder:   "desc",
			},
			expectedCount: 1,
			expectedTotal: 1,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				assert.Equal(t, "event2", events[0].Event.EventID)
				assert.True(t, events[0].Event.IsSimulated)
			},
		},
		{
			name: "query with time range filter",
			filter: EventQueryFilter{
				StartTime: func() *time.Time { t := now.Add(-2*time.Hour - 30*time.Minute); return &t }(),
				EndTime:   func() *time.Time { t := now.Add(-30 * time.Minute); return &t }(),
				Limit:     10,
				Offset:    0,
				SortBy:    "timestamp",
				SortOrder: "desc",
			},
			expectedCount: 2,
			expectedTotal: 2,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				assert.Equal(t, "event3", events[0].Event.EventID)
				assert.Equal(t, "event2", events[1].Event.EventID)
			},
		},
		{
			name: "query with pagination",
			filter: EventQueryFilter{
				Limit:     2,
				Offset:    1,
				SortBy:    "timestamp",
				SortOrder: "desc",
			},
			expectedCount: 2,
			expectedTotal: 4,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				// Should skip first event and return next 2
				assert.Equal(t, "event3", events[0].Event.EventID)
				assert.Equal(t, "event2", events[1].Event.EventID)
			},
		},
		{
			name: "query sent events only",
			filter: EventQueryFilter{
				SentStatus: "sent",
				Limit:      10,
				Offset:     0,
				SortBy:     "timestamp",
				SortOrder:  "desc",
			},
			expectedCount: 1,
			expectedTotal: 1,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				assert.NotNil(t, events[0].SentAt)
			},
		},
		{
			name: "query pending events only",
			filter: EventQueryFilter{
				SentStatus: "pending",
				Limit:      10,
				Offset:     0,
				SortBy:     "timestamp",
				SortOrder:  "desc",
			},
			expectedCount: 2,
			expectedTotal: 2,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				for _, event := range events {
					assert.Nil(t, event.SentAt)
					assert.Less(t, event.RetryCount, 3) // Less than max retries
				}
			},
		},
		{
			name: "query failed events only",
			filter: EventQueryFilter{
				SentStatus: "failed",
				Limit:      10,
				Offset:     0,
				SortBy:     "timestamp",
				SortOrder:  "desc",
			},
			expectedCount: 1,
			expectedTotal: 1,
			validateResult: func(t *testing.T, events []QueuedEvent) {
				assert.Nil(t, events[0].SentAt)
				assert.GreaterOrEqual(t, events[0].RetryCount, 3)
			},
		},
		{
			name: "query with no results",
			filter: EventQueryFilter{
				EventType: "nonexistent",
				Limit:     10,
				Offset:    0,
				SortBy:    "timestamp",
				SortOrder: "desc",
			},
			expectedCount: 0,
			expectedTotal: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, total, err := queueManager.QueryEvents(ctx, tt.filter)
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(events))
			assert.Equal(t, tt.expectedTotal, total)
			
			if tt.validateResult != nil {
				tt.validateResult(t, events)
			}
		})
	}
}

func TestSQLiteQueueManager_GetEventStats(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()
	
	queueManager := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Initialize queue manager
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       10,
		RetryInterval:   30 * time.Second,
		MaxRetries:      3,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	err := queueManager.Initialize(ctx, config)
	require.NoError(t, err)
	
	// Create test events with different characteristics
	now := time.Now()
	testEvents := []types.StandardEvent{
		{
			EventID:        "event1",
			ExternalUserID: "user1",
			Timestamp:      now.Add(-25 * time.Hour), // More than 24 hours ago
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
		{
			EventID:        "event2",
			ExternalUserID: "user2",
			Timestamp:      now.Add(-2 * time.Hour), // Within 24 hours
			EventType:      types.EventTypeExit,
			IsSimulated:    true,
			DeviceID:       "device1",
		},
		{
			EventID:        "event3",
			ExternalUserID: "user1",
			Timestamp:      now.Add(-1 * time.Hour), // Within 24 hours
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
		{
			EventID:        "event4",
			ExternalUserID: "user3",
			Timestamp:      now,
			EventType:      types.EventTypeDenied,
			IsSimulated:    true,
			DeviceID:       "device1",
		},
	}
	
	// Enqueue test events
	for _, event := range testEvents {
		err := queueManager.Enqueue(ctx, event)
		require.NoError(t, err)
	}
	
	// Mark some events as sent and failed
	pendingEvents, err := queueManager.GetPendingEvents(ctx, 4)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 4)
	
	// Mark first two as sent
	sentEventIDs := []int64{pendingEvents[0].ID, pendingEvents[1].ID}
	err = queueManager.MarkEventsSent(ctx, sentEventIDs)
	require.NoError(t, err)
	
	// Mark one as failed by incrementing retry count to max
	failedEventIDs := []int64{pendingEvents[2].ID}
	for i := 0; i < 3; i++ { // Increment retry count to reach max retries (3)
		err = queueManager.MarkEventsFailed(ctx, failedEventIDs, "Test failure")
		require.NoError(t, err)
	}
	
	// Get statistics
	stats, err := queueManager.GetEventStats(ctx)
	require.NoError(t, err)
	
	// Validate statistics
	assert.Equal(t, int64(4), stats.TotalEvents)
	assert.Equal(t, int64(2), stats.SentEvents)
	assert.Equal(t, int64(1), stats.PendingEvents)
	assert.Equal(t, int64(1), stats.FailedEvents)
	assert.Equal(t, int64(3), stats.UniqueUsers) // user1, user2, user3
	assert.Equal(t, int64(2), stats.SimulatedEvents) // event2, event4
	
	// Check events by type
	assert.Equal(t, int64(2), stats.EventsByType[types.EventTypeEntry])
	assert.Equal(t, int64(1), stats.EventsByType[types.EventTypeExit])
	assert.Equal(t, int64(1), stats.EventsByType[types.EventTypeDenied])
	
	// Check time range
	assert.NotNil(t, stats.OldestEventTime)
	assert.NotNil(t, stats.NewestEventTime)
	assert.True(t, stats.OldestEventTime.Before(*stats.NewestEventTime))
	
	// Check averages are calculated
	assert.Greater(t, stats.AveragePerHour, 0.0)
	assert.Greater(t, stats.AveragePerDay, 0.0)
	
	// Check that we have some hourly/daily data (at least for recent events)
	assert.Greater(t, len(stats.EventsByHour), 0)
	assert.Greater(t, len(stats.EventsByDay), 0)
}

func TestSQLiteQueueManager_ClearEvents(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()
	
	queueManager := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Initialize queue manager
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       10,
		RetryInterval:   30 * time.Second,
		MaxRetries:      3,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	err := queueManager.Initialize(ctx, config)
	require.NoError(t, err)
	
	// Create test events
	now := time.Now()
	testEvents := []types.StandardEvent{
		{
			EventID:        "event1",
			ExternalUserID: "user1",
			Timestamp:      now.Add(-48 * time.Hour), // Old event
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
		{
			EventID:        "event2",
			ExternalUserID: "user2",
			Timestamp:      now.Add(-2 * time.Hour), // Recent event
			EventType:      types.EventTypeExit,
			IsSimulated:    true,
			DeviceID:       "device1",
		},
		{
			EventID:        "event3",
			ExternalUserID: "user1",
			Timestamp:      now.Add(-1 * time.Hour), // Recent event
			EventType:      types.EventTypeEntry,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
		{
			EventID:        "event4",
			ExternalUserID: "user3",
			Timestamp:      now,
			EventType:      types.EventTypeDenied,
			IsSimulated:    false,
			DeviceID:       "device1",
		},
	}
	
	// Enqueue test events
	for _, event := range testEvents {
		err := queueManager.Enqueue(ctx, event)
		require.NoError(t, err)
	}
	
	// Mark some events as sent and failed
	pendingEvents, err := queueManager.GetPendingEvents(ctx, 4)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 4)
	
	// Mark first two as sent
	sentEventIDs := []int64{pendingEvents[0].ID, pendingEvents[1].ID}
	err = queueManager.MarkEventsSent(ctx, sentEventIDs)
	require.NoError(t, err)
	
	// Mark one as failed by incrementing retry count to max
	failedEventIDs := []int64{pendingEvents[2].ID}
	for i := 0; i < 3; i++ { // Increment retry count to reach max retries (3)
		err = queueManager.MarkEventsFailed(ctx, failedEventIDs, "Test failure")
		require.NoError(t, err)
	}
	
	tests := []struct {
		name            string
		criteria        EventClearCriteria
		expectedDeleted int64
		validateResult  func(t *testing.T, qm QueueManager)
	}{
		{
			name: "clear old events",
			criteria: EventClearCriteria{
				OlderThan: func() *time.Time { t := now.Add(-24 * time.Hour); return &t }(),
			},
			expectedDeleted: 1, // Only event1 is older than 24 hours
			validateResult: func(t *testing.T, qm QueueManager) {
				// Verify that only recent events remain
				filter := EventQueryFilter{
					Limit:     10,
					Offset:    0,
					SortBy:    "timestamp",
					SortOrder: "desc",
				}
				events, total, err := qm.QueryEvents(ctx, filter)
				require.NoError(t, err)
				assert.Equal(t, int64(3), total)
				assert.Len(t, events, 3)
				
				// Verify event1 is gone
				for _, event := range events {
					assert.NotEqual(t, "event1", event.Event.EventID)
				}
			},
		},
		{
			name: "clear specific event type",
			criteria: EventClearCriteria{
				EventType: types.EventTypeEntry,
			},
			expectedDeleted: 2, // event1 and event3 are entry events
			validateResult: func(t *testing.T, qm QueueManager) {
				filter := EventQueryFilter{
					EventType: types.EventTypeEntry,
					Limit:     10,
					Offset:    0,
					SortBy:    "timestamp",
					SortOrder: "desc",
				}
				events, total, err := qm.QueryEvents(ctx, filter)
				require.NoError(t, err)
				assert.Equal(t, int64(0), total)
				assert.Len(t, events, 0)
			},
		},
		{
			name: "clear only sent events",
			criteria: EventClearCriteria{
				OnlySent: true,
			},
			expectedDeleted: 2, // Two events were marked as sent
			validateResult: func(t *testing.T, qm QueueManager) {
				filter := EventQueryFilter{
					SentStatus: "sent",
					Limit:      10,
					Offset:     0,
					SortBy:     "timestamp",
					SortOrder:  "desc",
				}
				events, total, err := qm.QueryEvents(ctx, filter)
				require.NoError(t, err)
				assert.Equal(t, int64(0), total)
				assert.Len(t, events, 0)
			},
		},
		{
			name: "clear only failed events",
			criteria: EventClearCriteria{
				OnlyFailed: true,
			},
			expectedDeleted: 1, // One event was marked as failed
			validateResult: func(t *testing.T, qm QueueManager) {
				filter := EventQueryFilter{
					SentStatus: "failed",
					Limit:      10,
					Offset:     0,
					SortBy:     "timestamp",
					SortOrder:  "desc",
				}
				events, total, err := qm.QueryEvents(ctx, filter)
				require.NoError(t, err)
				assert.Equal(t, int64(0), total)
				assert.Len(t, events, 0)
			},
		},
		{
			name:            "clear all events",
			criteria:        EventClearCriteria{},
			expectedDeleted: 4, // All events
			validateResult: func(t *testing.T, qm QueueManager) {
				filter := EventQueryFilter{
					Limit:     10,
					Offset:    0,
					SortBy:    "timestamp",
					SortOrder: "desc",
				}
				events, total, err := qm.QueryEvents(ctx, filter)
				require.NoError(t, err)
				assert.Equal(t, int64(0), total)
				assert.Len(t, events, 0)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-setup for each test to have fresh data
			db := setupTestDB(t)
			defer db.Close()
			
			qm := NewSQLiteQueueManager(db)
			err := qm.Initialize(ctx, config)
			require.NoError(t, err)
			
			// Re-enqueue test events
			for _, event := range testEvents {
				err := qm.Enqueue(ctx, event)
				require.NoError(t, err)
			}
			
			// Re-mark events as sent/failed
			pendingEvents, err := qm.GetPendingEvents(ctx, 4)
			require.NoError(t, err)
			
			sentEventIDs := []int64{pendingEvents[0].ID, pendingEvents[1].ID}
			err = qm.MarkEventsSent(ctx, sentEventIDs)
			require.NoError(t, err)
			
			failedEventIDs := []int64{pendingEvents[2].ID}
			for i := 0; i < 3; i++ { // Increment retry count to reach max retries (3)
				err = qm.MarkEventsFailed(ctx, failedEventIDs, "Test failure")
				require.NoError(t, err)
			}
			
			// Execute clear operation
			deletedCount, err := qm.ClearEvents(ctx, tt.criteria)
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedDeleted, deletedCount)
			
			if tt.validateResult != nil {
				tt.validateResult(t, qm)
			}
		})
	}
}

func TestSQLiteQueueManager_QueryEvents_EmptyDatabase(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()
	
	queueManager := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Initialize queue manager
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       10,
		RetryInterval:   30 * time.Second,
		MaxRetries:      3,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	err := queueManager.Initialize(ctx, config)
	require.NoError(t, err)
	
	// Query empty database
	filter := EventQueryFilter{
		Limit:     10,
		Offset:    0,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
	
	events, total, err := queueManager.QueryEvents(ctx, filter)
	
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, events, 0)
}

func TestSQLiteQueueManager_GetEventStats_EmptyDatabase(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()
	
	queueManager := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Initialize queue manager
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       10,
		RetryInterval:   30 * time.Second,
		MaxRetries:      3,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	err := queueManager.Initialize(ctx, config)
	require.NoError(t, err)
	
	// Get statistics from empty database
	stats, err := queueManager.GetEventStats(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEvents)
	assert.Equal(t, int64(0), stats.SentEvents)
	assert.Equal(t, int64(0), stats.PendingEvents)
	assert.Equal(t, int64(0), stats.FailedEvents)
	assert.Equal(t, int64(0), stats.UniqueUsers)
	assert.Equal(t, int64(0), stats.SimulatedEvents)
	assert.Nil(t, stats.OldestEventTime)
	assert.Nil(t, stats.NewestEventTime)
	assert.Equal(t, 0.0, stats.AveragePerHour)
	assert.Equal(t, 0.0, stats.AveragePerDay)
	assert.NotNil(t, stats.EventsByType)
	assert.NotNil(t, stats.EventsByHour)
	assert.NotNil(t, stats.EventsByDay)
}

func TestSQLiteQueueManager_ClearEvents_EmptyDatabase(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()
	
	queueManager := NewSQLiteQueueManager(db)
	ctx := context.Background()
	
	// Initialize queue manager
	config := QueueConfig{
		MaxSize:         1000,
		BatchSize:       10,
		RetryInterval:   30 * time.Second,
		MaxRetries:      3,
		RetentionPolicy: RetentionPolicyFIFO,
	}
	err := queueManager.Initialize(ctx, config)
	require.NoError(t, err)
	
	// Clear events from empty database
	criteria := EventClearCriteria{}
	deletedCount, err := queueManager.ClearEvents(ctx, criteria)
	
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deletedCount)
}