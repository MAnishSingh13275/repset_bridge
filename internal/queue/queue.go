package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gym-door-bridge/internal/database"
	"gym-door-bridge/internal/types"
)

// QueueConfig holds configuration for the queue manager
type QueueConfig struct {
	MaxSize         int           `json:"maxSize"`         // Maximum number of events in queue
	BatchSize       int           `json:"batchSize"`       // Number of events to send in each batch
	RetryInterval   time.Duration `json:"retryInterval"`   // Time between retry attempts
	MaxRetries      int           `json:"maxRetries"`      // Maximum number of retry attempts
	EncryptionKey   string        `json:"encryptionKey"`   // Key for encrypting sensitive payloads
	RetentionPolicy string        `json:"retentionPolicy"` // "fifo" or "priority"
}

// QueueStats contains statistics about the queue
type QueueStats struct {
	QueueDepth      int       `json:"queueDepth"`      // Current number of events in queue
	PendingEvents   int       `json:"pendingEvents"`   // Events waiting to be sent
	SentEvents      int64     `json:"sentEvents"`      // Total events successfully sent
	FailedEvents    int64     `json:"failedEvents"`    // Total events that failed to send
	LastSentAt      time.Time `json:"lastSentAt"`      // Timestamp of last successful send
	LastFailureAt   time.Time `json:"lastFailureAt"`   // Timestamp of last failure
	OldestEventTime time.Time `json:"oldestEventTime"` // Timestamp of oldest queued event
}

// QueuedEvent represents an event stored in the queue
type QueuedEvent struct {
	ID          int64               `json:"id"`
	Event       types.StandardEvent `json:"event"`
	CreatedAt   time.Time           `json:"createdAt"`
	SentAt      *time.Time          `json:"sentAt,omitempty"`
	RetryCount  int                 `json:"retryCount"`
	LastError   string              `json:"lastError,omitempty"`
	IsEncrypted bool                `json:"isEncrypted"`
}

// SendResult represents the result of sending events to the cloud
type SendResult struct {
	Success      bool     `json:"success"`
	SentCount    int      `json:"sentCount"`
	FailedCount  int      `json:"failedCount"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	FailedEvents []int64  `json:"failedEvents,omitempty"` // IDs of events that failed
}

// QueueManager defines the interface for managing the offline event queue
type QueueManager interface {
	// Initialize sets up the queue manager with the provided configuration
	Initialize(ctx context.Context, config QueueConfig) error

	// Enqueue adds a new event to the queue
	Enqueue(ctx context.Context, event types.StandardEvent) error

	// GetPendingEvents retrieves events that need to be sent to the cloud
	GetPendingEvents(ctx context.Context, limit int) ([]QueuedEvent, error)

	// MarkEventsSent marks events as successfully sent
	MarkEventsSent(ctx context.Context, eventIDs []int64) error

	// MarkEventsFailed marks events as failed and increments retry count
	MarkEventsFailed(ctx context.Context, eventIDs []int64, errorMessage string) error

	// GetStats returns current queue statistics
	GetStats(ctx context.Context) (QueueStats, error)

	// Cleanup removes old events based on retention policy
	Cleanup(ctx context.Context) error

	// GetQueueDepth returns the current number of events in the queue
	GetQueueDepth(ctx context.Context) (int, error)

	// IsQueueFull checks if the queue has reached its maximum capacity
	IsQueueFull(ctx context.Context) (bool, error)

	// Close gracefully shuts down the queue manager
	Close(ctx context.Context) error
}

// RetentionPolicy constants
const (
	RetentionPolicyFIFO     = "fifo"
	RetentionPolicyPriority = "priority"
)

// sqliteQueueManager implements QueueManager using SQLite for persistence
type sqliteQueueManager struct {
	db     *database.DB
	config QueueConfig
}

// NewSQLiteQueueManager creates a new SQLite-based queue manager
func NewSQLiteQueueManager(db *database.DB) QueueManager {
	return &sqliteQueueManager{
		db: db,
	}
}

// Initialize sets up the queue manager with the provided configuration
func (q *sqliteQueueManager) Initialize(ctx context.Context, config QueueConfig) error {
	q.config = config
	
	// Validate configuration
	if config.MaxSize <= 0 {
		return fmt.Errorf("maxSize must be positive, got %d", config.MaxSize)
	}
	if config.BatchSize <= 0 {
		return fmt.Errorf("batchSize must be positive, got %d", config.BatchSize)
	}
	if config.RetryInterval <= 0 {
		return fmt.Errorf("retryInterval must be positive, got %v", config.RetryInterval)
	}
	if config.MaxRetries < 0 {
		return fmt.Errorf("maxRetries must be non-negative, got %d", config.MaxRetries)
	}
	
	// Set default retention policy if not specified
	if config.RetentionPolicy == "" {
		q.config.RetentionPolicy = RetentionPolicyFIFO
	}
	
	return nil
}

// Enqueue adds a new event to the queue
func (q *sqliteQueueManager) Enqueue(ctx context.Context, event types.StandardEvent) error {
	// Convert StandardEvent to database model first
	dbEvent, err := q.standardEventToDBEvent(event)
	if err != nil {
		return fmt.Errorf("failed to convert event: %w", err)
	}
	
	// Insert into database
	if err := q.db.InsertEvent(dbEvent); err != nil {
		return fmt.Errorf("failed to insert event into database: %w", err)
	}
	
	// Check if queue exceeds capacity and evict if necessary
	depth, err := q.GetQueueDepth(ctx)
	if err != nil {
		return fmt.Errorf("failed to check queue depth: %w", err)
	}
	
	if depth > q.config.MaxSize {
		if err := q.evictOldestEvents(ctx); err != nil {
			return fmt.Errorf("failed to evict oldest events: %w", err)
		}
	}
	
	return nil
}

// GetPendingEvents retrieves events that need to be sent to the cloud
func (q *sqliteQueueManager) GetPendingEvents(ctx context.Context, limit int) ([]QueuedEvent, error) {
	if limit <= 0 {
		limit = q.config.BatchSize
	}
	
	dbEvents, err := q.db.GetUnsentEvents(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get unsent events: %w", err)
	}
	
	queuedEvents := make([]QueuedEvent, len(dbEvents))
	for i, dbEvent := range dbEvents {
		queuedEvent, err := q.dbEventToQueuedEvent(dbEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to convert db event to queued event: %w", err)
		}
		queuedEvents[i] = queuedEvent
	}
	
	return queuedEvents, nil
}

// MarkEventsSent marks events as successfully sent
func (q *sqliteQueueManager) MarkEventsSent(ctx context.Context, eventIDs []int64) error {
	if len(eventIDs) == 0 {
		return nil
	}
	
	// Convert int64 IDs to string IDs for database layer
	stringIDs := make([]string, len(eventIDs))
	for i, id := range eventIDs {
		// We need to get the event_id string from the database using the int64 ID
		// For now, we'll assume the eventIDs are actually the database row IDs
		// and we need to get the event_id strings
		stringIDs[i] = fmt.Sprintf("%d", id)
	}
	
	// Get the actual event_id strings from the database
	actualEventIDs, err := q.getEventIDsByRowIDs(ctx, eventIDs)
	if err != nil {
		return fmt.Errorf("failed to get event IDs: %w", err)
	}
	
	if err := q.db.MarkEventsSent(actualEventIDs); err != nil {
		return fmt.Errorf("failed to mark events as sent: %w", err)
	}
	
	return nil
}

// MarkEventsFailed marks events as failed and increments retry count
func (q *sqliteQueueManager) MarkEventsFailed(ctx context.Context, eventIDs []int64, errorMessage string) error {
	if len(eventIDs) == 0 {
		return nil
	}
	
	// Get the actual event_id strings from the database
	actualEventIDs, err := q.getEventIDsByRowIDs(ctx, eventIDs)
	if err != nil {
		return fmt.Errorf("failed to get event IDs: %w", err)
	}
	
	if err := q.db.IncrementRetryCount(actualEventIDs); err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}
	
	// TODO: Store error message in database (would require schema update)
	// For now, we just increment the retry count
	
	return nil
}

// GetStats returns current queue statistics
func (q *sqliteQueueManager) GetStats(ctx context.Context) (QueueStats, error) {
	stats := QueueStats{}
	
	// Get queue depth
	depth, err := q.db.GetQueueDepth()
	if err != nil {
		return stats, fmt.Errorf("failed to get queue depth: %w", err)
	}
	stats.QueueDepth = depth
	stats.PendingEvents = depth // All unsent events are pending
	
	// TODO: Implement additional statistics
	// This would require additional database queries or schema changes
	// For now, we provide basic stats
	
	return stats, nil
}

// Cleanup removes old events based on retention policy
func (q *sqliteQueueManager) Cleanup(ctx context.Context) error {
	// Clean up old sent events (older than 7 days by default)
	cleanupAge := 7 * 24 * time.Hour
	if err := q.db.CleanupOldEvents(cleanupAge); err != nil {
		return fmt.Errorf("failed to cleanup old events: %w", err)
	}
	
	return nil
}

// GetQueueDepth returns the current number of events in the queue
func (q *sqliteQueueManager) GetQueueDepth(ctx context.Context) (int, error) {
	return q.db.GetQueueDepth()
}

// IsQueueFull checks if the queue has reached its maximum capacity
func (q *sqliteQueueManager) IsQueueFull(ctx context.Context) (bool, error) {
	depth, err := q.GetQueueDepth(ctx)
	if err != nil {
		return false, err
	}
	
	return depth >= q.config.MaxSize, nil
}

// Close gracefully shuts down the queue manager
func (q *sqliteQueueManager) Close(ctx context.Context) error {
	// SQLite connection is managed by the database layer
	// No specific cleanup needed for the queue manager
	return nil
}

// Helper methods

// evictOldestEvents removes the oldest events when queue is full
func (q *sqliteQueueManager) evictOldestEvents(ctx context.Context) error {
	// Get current queue depth
	depth, err := q.GetQueueDepth(ctx)
	if err != nil {
		return err
	}
	
	// Calculate how many events to evict
	evictCount := depth - q.config.MaxSize
	if evictCount <= 0 {
		return nil
	}
	
	// Use a simpler approach: delete the oldest events directly
	return q.db.EvictOldestEventsDirect(evictCount)
}

// standardEventToDBEvent converts a StandardEvent to a database EventQueue model
func (q *sqliteQueueManager) standardEventToDBEvent(event types.StandardEvent) (*database.EventQueue, error) {
	// Serialize raw data to JSON if present
	var rawDataJSON string
	if event.RawData != nil {
		rawDataBytes, err := json.Marshal(event.RawData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw data: %w", err)
		}
		rawDataJSON = string(rawDataBytes)
	}
	
	return &database.EventQueue{
		EventID:        event.EventID,
		ExternalUserID: event.ExternalUserID,
		Timestamp:      event.Timestamp,
		EventType:      event.EventType,
		IsSimulated:    event.IsSimulated,
		RawData:        rawDataJSON,
		CreatedAt:      time.Now(),
		RetryCount:     0,
	}, nil
}

// dbEventToQueuedEvent converts a database EventQueue to a QueuedEvent
func (q *sqliteQueueManager) dbEventToQueuedEvent(dbEvent *database.EventQueue) (QueuedEvent, error) {
	// Convert database event to StandardEvent
	standardEvent := types.StandardEvent{
		EventID:        dbEvent.EventID,
		ExternalUserID: dbEvent.ExternalUserID,
		Timestamp:      dbEvent.Timestamp,
		EventType:      dbEvent.EventType,
		IsSimulated:    dbEvent.IsSimulated,
	}
	
	// Deserialize raw data if present
	if dbEvent.RawData != "" {
		var rawData map[string]interface{}
		if err := json.Unmarshal([]byte(dbEvent.RawData), &rawData); err != nil {
			return QueuedEvent{}, fmt.Errorf("failed to unmarshal raw data: %w", err)
		}
		standardEvent.RawData = rawData
	}
	
	return QueuedEvent{
		ID:          dbEvent.ID,
		Event:       standardEvent,
		CreatedAt:   dbEvent.CreatedAt,
		SentAt:      dbEvent.SentAt,
		RetryCount:  dbEvent.RetryCount,
		IsEncrypted: dbEvent.RawData != "", // Assume encrypted if raw data exists
	}, nil
}

// getEventIDsByRowIDs retrieves event_id strings by database row IDs
func (q *sqliteQueueManager) getEventIDsByRowIDs(ctx context.Context, rowIDs []int64) ([]string, error) {
	if len(rowIDs) == 0 {
		return nil, nil
	}
	
	// This is a simplified implementation - in a real scenario we'd need a proper query
	// For now, we'll get all unsent events and filter by ID
	allEvents, err := q.db.GetUnsentEvents(1000) // Get a large batch
	if err != nil {
		return nil, err
	}
	
	eventIDs := make([]string, 0, len(rowIDs))
	rowIDMap := make(map[int64]bool)
	for _, id := range rowIDs {
		rowIDMap[id] = true
	}
	
	for _, event := range allEvents {
		if rowIDMap[event.ID] {
			eventIDs = append(eventIDs, event.EventID)
		}
	}
	
	return eventIDs, nil
}

// GetTierConfig returns queue configuration for the specified performance tier
func GetTierConfig(tier database.PerformanceTier) QueueConfig {
	switch tier {
	case database.TierLite:
		return QueueConfig{
			MaxSize:         1000,
			BatchSize:       10,
			RetryInterval:   30 * time.Second,
			MaxRetries:      3,
			RetentionPolicy: RetentionPolicyFIFO,
		}
	case database.TierNormal:
		return QueueConfig{
			MaxSize:         10000,
			BatchSize:       50,
			RetryInterval:   15 * time.Second,
			MaxRetries:      5,
			RetentionPolicy: RetentionPolicyFIFO,
		}
	case database.TierFull:
		return QueueConfig{
			MaxSize:         50000,
			BatchSize:       100,
			RetryInterval:   10 * time.Second,
			MaxRetries:      10,
			RetentionPolicy: RetentionPolicyFIFO,
		}
	default:
		return GetTierConfig(database.TierNormal)
	}
}