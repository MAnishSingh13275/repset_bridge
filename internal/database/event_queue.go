package database

import (
	"database/sql"
	"fmt"
	"time"
)

// InsertEvent adds a new event to the queue
func (db *DB) InsertEvent(event *EventQueue) error {
	// Encrypt raw data if present
	var encryptedRawData sql.NullString
	if event.RawData != "" {
		encrypted, err := db.Encrypt([]byte(event.RawData))
		if err != nil {
			return fmt.Errorf("failed to encrypt raw data: %w", err)
		}
		encryptedRawData = sql.NullString{String: encrypted, Valid: true}
	}

	query := `
		INSERT INTO event_queue (event_id, external_user_id, timestamp, event_type, is_simulated, raw_data)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, 
		event.EventID, 
		event.ExternalUserID, 
		event.Timestamp, 
		event.EventType, 
		event.IsSimulated, 
		encryptedRawData,
	)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	event.ID = id
	return nil
}

// GetUnsentEvents retrieves all events that haven't been sent to the cloud
func (db *DB) GetUnsentEvents(limit int) ([]*EventQueue, error) {
	query := `
		SELECT id, event_id, external_user_id, timestamp, event_type, is_simulated, 
		       raw_data, created_at, sent_at, retry_count
		FROM event_queue 
		WHERE sent_at IS NULL 
		ORDER BY timestamp ASC 
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unsent events: %w", err)
	}
	defer rows.Close()

	var events []*EventQueue
	for rows.Next() {
		event := &EventQueue{}
		var rawData sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.ExternalUserID,
			&event.Timestamp,
			&event.EventType,
			&event.IsSimulated,
			&rawData,
			&event.CreatedAt,
			&event.SentAt,
			&event.RetryCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		// Decrypt raw data if present
		if rawData.Valid {
			decrypted, err := db.Decrypt(rawData.String)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt raw data for event %s: %w", event.EventID, err)
			}
			event.RawData = string(decrypted)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event rows: %w", err)
	}

	return events, nil
}

// MarkEventsSent marks events as successfully sent to the cloud
func (db *DB) MarkEventsSent(eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}

	// Build placeholders for IN clause
	placeholders := make([]interface{}, len(eventIDs))
	for i, id := range eventIDs {
		placeholders[i] = id
	}

	query := fmt.Sprintf(`
		UPDATE event_queue 
		SET sent_at = CURRENT_TIMESTAMP 
		WHERE event_id IN (%s)
	`, generatePlaceholders(len(eventIDs)))

	_, err := db.conn.Exec(query, placeholders...)
	if err != nil {
		return fmt.Errorf("failed to mark events as sent: %w", err)
	}

	return nil
}

// IncrementRetryCount increments the retry count for failed events
func (db *DB) IncrementRetryCount(eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}

	placeholders := make([]interface{}, len(eventIDs))
	for i, id := range eventIDs {
		placeholders[i] = id
	}

	query := fmt.Sprintf(`
		UPDATE event_queue 
		SET retry_count = retry_count + 1 
		WHERE event_id IN (%s)
	`, generatePlaceholders(len(eventIDs)))

	_, err := db.conn.Exec(query, placeholders...)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	return nil
}

// GetQueueDepth returns the number of unsent events in the queue
func (db *DB) GetQueueDepth() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM event_queue WHERE sent_at IS NULL").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue depth: %w", err)
	}
	return count, nil
}

// CleanupOldEvents removes old sent events to prevent database growth
func (db *DB) CleanupOldEvents(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	
	query := `
		DELETE FROM event_queue 
		WHERE sent_at IS NOT NULL AND sent_at < ?
	`

	result, err := db.conn.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old events: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// Log cleanup activity (would use actual logger in production)
		fmt.Printf("Cleaned up %d old events older than %v\n", rowsAffected, olderThan)
	}

	return nil
}

// EvictOldestEvents removes the oldest unsent events when queue is full
func (db *DB) EvictOldestEvents(maxQueueSize int) error {
	query := `
		DELETE FROM event_queue 
		WHERE id IN (
			SELECT id FROM event_queue 
			WHERE sent_at IS NULL 
			ORDER BY timestamp ASC 
			LIMIT (
				SELECT MAX(0, COUNT(*) - ?) 
				FROM event_queue 
				WHERE sent_at IS NULL
			)
		)
	`

	result, err := db.conn.Exec(query, maxQueueSize)
	if err != nil {
		return fmt.Errorf("failed to evict oldest events: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Evicted %d oldest events to maintain queue size limit of %d\n", rowsAffected, maxQueueSize)
	}

	return nil
}

// generatePlaceholders creates a string of SQL placeholders (?, ?, ?)
func generatePlaceholders(count int) string {
	if count == 0 {
		return ""
	}
	
	result := "?"
	for i := 1; i < count; i++ {
		result += ", ?"
	}
	return result
}

// HasSimilarEvent checks if a similar event exists within the specified time window
func (db *DB) HasSimilarEvent(externalUserID, eventType string, windowStart, windowEnd time.Time) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM event_queue 
		WHERE external_user_id = ? 
		  AND event_type = ? 
		  AND timestamp BETWEEN ? AND ?
		LIMIT 1
	`

	var count int
	err := db.conn.QueryRow(query, externalUserID, eventType, windowStart, windowEnd).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check for similar events: %w", err)
	}

	return count > 0, nil
}

// EvictOldestEventsDirect removes the specified number of oldest unsent events
func (db *DB) EvictOldestEventsDirect(count int) error {
	if count <= 0 {
		return nil
	}
	
	query := `
		DELETE FROM event_queue 
		WHERE id IN (
			SELECT id FROM event_queue 
			WHERE sent_at IS NULL 
			ORDER BY timestamp ASC 
			LIMIT ?
		)
	`

	result, err := db.conn.Exec(query, count)
	if err != nil {
		return fmt.Errorf("failed to evict oldest events: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Evicted %d oldest events\n", rowsAffected)
	}

	return nil
}