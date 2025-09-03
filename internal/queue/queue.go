package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// EventQueryFilter represents criteria for querying events
type EventQueryFilter struct {
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	EventType    string     `json:"eventType,omitempty"`
	UserID       string     `json:"userId,omitempty"`
	IsSimulated  *bool      `json:"isSimulated,omitempty"`
	SentStatus   string     `json:"sentStatus,omitempty"` // "all", "sent", "pending", "failed"
	Limit        int        `json:"limit"`
	Offset       int        `json:"offset"`
	SortBy       string     `json:"sortBy"`     // "timestamp", "event_type", "external_user_id"
	SortOrder    string     `json:"sortOrder"`  // "asc", "desc"
}

// EventStatistics represents comprehensive event statistics
type EventStatistics struct {
	TotalEvents      int64                    `json:"totalEvents"`
	EventsByType     map[string]int64         `json:"eventsByType"`
	EventsByHour     map[string]int64         `json:"eventsByHour"`
	EventsByDay      map[string]int64         `json:"eventsByDay"`
	PendingEvents    int64                    `json:"pendingEvents"`
	SentEvents       int64                    `json:"sentEvents"`
	FailedEvents     int64                    `json:"failedEvents"`
	UniqueUsers      int64                    `json:"uniqueUsers"`
	SimulatedEvents  int64                    `json:"simulatedEvents"`
	OldestEventTime  *time.Time               `json:"oldestEventTime,omitempty"`
	NewestEventTime  *time.Time               `json:"newestEventTime,omitempty"`
	AveragePerHour   float64                  `json:"averagePerHour"`
	AveragePerDay    float64                  `json:"averagePerDay"`
}

// EventClearCriteria represents criteria for clearing events
type EventClearCriteria struct {
	OlderThan   *time.Time `json:"olderThan,omitempty"`
	EventType   string     `json:"eventType,omitempty"`
	OnlySent    bool       `json:"onlySent,omitempty"`
	OnlyFailed  bool       `json:"onlyFailed,omitempty"`
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

	// QueryEvents retrieves events based on filter criteria
	QueryEvents(ctx context.Context, filter EventQueryFilter) ([]QueuedEvent, int64, error)

	// GetEventStats returns comprehensive event statistics
	GetEventStats(ctx context.Context) (EventStatistics, error)

	// ClearEvents removes events based on criteria
	ClearEvents(ctx context.Context, criteria EventClearCriteria) (int64, error)

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

// QueryEvents retrieves events based on filter criteria
func (q *sqliteQueueManager) QueryEvents(ctx context.Context, filter EventQueryFilter) ([]QueuedEvent, int64, error) {
	// Build the WHERE clause based on filter criteria
	var conditions []string
	var args []interface{}
	
	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.StartTime)
	}
	
	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.EndTime)
	}
	
	if filter.EventType != "" {
		conditions = append(conditions, "event_type = ?")
		args = append(args, filter.EventType)
	}
	
	if filter.UserID != "" {
		conditions = append(conditions, "external_user_id = ?")
		args = append(args, filter.UserID)
	}
	
	if filter.IsSimulated != nil {
		conditions = append(conditions, "is_simulated = ?")
		args = append(args, *filter.IsSimulated)
	}
	
	// Handle sent status filter
	switch filter.SentStatus {
	case "sent":
		conditions = append(conditions, "sent_at IS NOT NULL")
	case "pending":
		conditions = append(conditions, "sent_at IS NULL AND retry_count < ?")
		args = append(args, q.config.MaxRetries)
	case "failed":
		conditions = append(conditions, "sent_at IS NULL AND retry_count >= ?")
		args = append(args, q.config.MaxRetries)
	// "all" or empty means no additional filter
	}
	
	// Build the complete query
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM event_queue %s", whereClause)
	var total int64
	if err := q.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to get event count: %w", err)
	}
	
	// Build ORDER BY clause
	orderBy := fmt.Sprintf("ORDER BY %s %s", filter.SortBy, strings.ToUpper(filter.SortOrder))
	
	// Build main query with pagination
	query := fmt.Sprintf(`
		SELECT id, event_id, external_user_id, timestamp, event_type, is_simulated, 
		       raw_data, created_at, sent_at, retry_count
		FROM event_queue %s %s
		LIMIT ? OFFSET ?
	`, whereClause, orderBy)
	
	args = append(args, filter.Limit, filter.Offset)
	
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()
	
	var events []QueuedEvent
	for rows.Next() {
		var dbEvent database.EventQueue
		var rawData sql.NullString
		var sentAt sql.NullTime
		
		err := rows.Scan(
			&dbEvent.ID,
			&dbEvent.EventID,
			&dbEvent.ExternalUserID,
			&dbEvent.Timestamp,
			&dbEvent.EventType,
			&dbEvent.IsSimulated,
			&rawData,
			&dbEvent.CreatedAt,
			&sentAt,
			&dbEvent.RetryCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event row: %w", err)
		}
		
		if rawData.Valid {
			dbEvent.RawData = rawData.String
		}
		if sentAt.Valid {
			dbEvent.SentAt = &sentAt.Time
		}
		
		queuedEvent, err := q.dbEventToQueuedEvent(&dbEvent)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert db event: %w", err)
		}
		
		events = append(events, queuedEvent)
	}
	
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating event rows: %w", err)
	}
	
	return events, total, nil
}

// GetEventStats returns comprehensive event statistics
func (q *sqliteQueueManager) GetEventStats(ctx context.Context) (EventStatistics, error) {
	stats := EventStatistics{
		EventsByType: make(map[string]int64),
		EventsByHour: make(map[string]int64),
		EventsByDay:  make(map[string]int64),
	}
	
	// Get total events count
	err := q.db.QueryRow("SELECT COUNT(*) FROM event_queue").Scan(&stats.TotalEvents)
	if err != nil {
		return stats, fmt.Errorf("failed to get total events count: %w", err)
	}
	
	// Get events by type
	rows, err := q.db.Query("SELECT event_type, COUNT(*) FROM event_queue GROUP BY event_type")
	if err != nil {
		return stats, fmt.Errorf("failed to get events by type: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return stats, fmt.Errorf("failed to scan event type row: %w", err)
		}
		stats.EventsByType[eventType] = count
	}
	
	// Get sent/pending/failed counts
	err = q.db.QueryRow("SELECT COUNT(*) FROM event_queue WHERE sent_at IS NOT NULL").Scan(&stats.SentEvents)
	if err != nil {
		return stats, fmt.Errorf("failed to get sent events count: %w", err)
	}
	
	err = q.db.QueryRow("SELECT COUNT(*) FROM event_queue WHERE sent_at IS NULL AND retry_count < ?", q.config.MaxRetries).Scan(&stats.PendingEvents)
	if err != nil {
		return stats, fmt.Errorf("failed to get pending events count: %w", err)
	}
	
	err = q.db.QueryRow("SELECT COUNT(*) FROM event_queue WHERE sent_at IS NULL AND retry_count >= ?", q.config.MaxRetries).Scan(&stats.FailedEvents)
	if err != nil {
		return stats, fmt.Errorf("failed to get failed events count: %w", err)
	}
	
	// Get unique users count
	err = q.db.QueryRow("SELECT COUNT(DISTINCT external_user_id) FROM event_queue").Scan(&stats.UniqueUsers)
	if err != nil {
		return stats, fmt.Errorf("failed to get unique users count: %w", err)
	}
	
	// Get simulated events count
	err = q.db.QueryRow("SELECT COUNT(*) FROM event_queue WHERE is_simulated = 1").Scan(&stats.SimulatedEvents)
	if err != nil {
		return stats, fmt.Errorf("failed to get simulated events count: %w", err)
	}
	
	// Get oldest and newest event times
	if stats.TotalEvents > 0 {
		// First try to get timestamps as strings and parse them
		var oldestTimeStr, newestTimeStr sql.NullString
		err = q.db.QueryRow("SELECT MIN(timestamp), MAX(timestamp) FROM event_queue").Scan(&oldestTimeStr, &newestTimeStr)
		if err != nil {
			return stats, fmt.Errorf("failed to get event time range: %w", err)
		}
		
		if oldestTimeStr.Valid && oldestTimeStr.String != "" {
			// Try multiple time formats that SQLite might use
			formats := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999",
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05.999999999Z07:00",
				"2006-01-02T15:04:05.999999999Z",
				"2006-01-02T15:04:05Z",
			}
			for _, format := range formats {
				if parsedTime, err := time.Parse(format, oldestTimeStr.String); err == nil {
					stats.OldestEventTime = &parsedTime
					break
				}
			}
		}
		if newestTimeStr.Valid && newestTimeStr.String != "" {
			// Try multiple time formats that SQLite might use
			formats := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999",
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05.999999999Z07:00",
				"2006-01-02T15:04:05.999999999Z",
				"2006-01-02T15:04:05Z",
			}
			for _, format := range formats {
				if parsedTime, err := time.Parse(format, newestTimeStr.String); err == nil {
					stats.NewestEventTime = &parsedTime
					break
				}
			}
		}
	}
	
	// Calculate averages if we have time range
	if stats.OldestEventTime != nil && stats.NewestEventTime != nil && stats.TotalEvents > 0 {
		duration := stats.NewestEventTime.Sub(*stats.OldestEventTime)
		if duration > 0 {
			hours := duration.Hours()
			days := duration.Hours() / 24
			
			if hours > 0 {
				stats.AveragePerHour = float64(stats.TotalEvents) / hours
			}
			if days > 0 {
				stats.AveragePerDay = float64(stats.TotalEvents) / days
			}
		}
	}
	
	// Get events by hour (last 24 hours)
	hourRows, err := q.db.Query(`
		SELECT strftime('%Y-%m-%d %H:00:00', timestamp) as hour, COUNT(*)
		FROM event_queue 
		WHERE timestamp >= datetime('now', '-24 hours')
		GROUP BY hour
		ORDER BY hour
	`)
	if err != nil {
		return stats, fmt.Errorf("failed to get events by hour: %w", err)
	}
	defer hourRows.Close()
	
	for hourRows.Next() {
		var hour string
		var count int64
		if err := hourRows.Scan(&hour, &count); err != nil {
			return stats, fmt.Errorf("failed to scan hour row: %w", err)
		}
		stats.EventsByHour[hour] = count
	}
	
	// Get events by day (last 30 days)
	dayRows, err := q.db.Query(`
		SELECT strftime('%Y-%m-%d', timestamp) as day, COUNT(*)
		FROM event_queue 
		WHERE timestamp >= datetime('now', '-30 days')
		GROUP BY day
		ORDER BY day
	`)
	if err != nil {
		return stats, fmt.Errorf("failed to get events by day: %w", err)
	}
	defer dayRows.Close()
	
	for dayRows.Next() {
		var day string
		var count int64
		if err := dayRows.Scan(&day, &count); err != nil {
			return stats, fmt.Errorf("failed to scan day row: %w", err)
		}
		stats.EventsByDay[day] = count
	}
	
	return stats, nil
}

// ClearEvents removes events based on criteria
func (q *sqliteQueueManager) ClearEvents(ctx context.Context, criteria EventClearCriteria) (int64, error) {
	var conditions []string
	var args []interface{}
	
	if criteria.OlderThan != nil {
		conditions = append(conditions, "timestamp < ?")
		args = append(args, *criteria.OlderThan)
	}
	
	if criteria.EventType != "" {
		conditions = append(conditions, "event_type = ?")
		args = append(args, criteria.EventType)
	}
	
	if criteria.OnlySent {
		conditions = append(conditions, "sent_at IS NOT NULL")
	}
	
	if criteria.OnlyFailed {
		conditions = append(conditions, "sent_at IS NULL AND retry_count >= ?")
		args = append(args, q.config.MaxRetries)
	}
	
	// Build the DELETE query
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	
	query := fmt.Sprintf("DELETE FROM event_queue %s", whereClause)
	
	result, err := q.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to clear events: %w", err)
	}
	
	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}
	
	return deletedCount, nil
}