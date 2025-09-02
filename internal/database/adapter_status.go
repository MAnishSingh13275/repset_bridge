package database

import (
	"database/sql"
	"fmt"
	"time"
)

// SetAdapterStatus updates the status of a hardware adapter
func (db *DB) SetAdapterStatus(adapterName, status string, errorMessage string) error {
	var errorMsg sql.NullString
	if errorMessage != "" {
		errorMsg = sql.NullString{String: errorMessage, Valid: true}
	}

	query := `
		INSERT OR REPLACE INTO adapter_status (adapter_name, status, error_message, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := db.conn.Exec(query, adapterName, status, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to set adapter status for %s: %w", adapterName, err)
	}

	return nil
}

// UpdateAdapterLastEvent updates the last event timestamp for an adapter
func (db *DB) UpdateAdapterLastEvent(adapterName string, eventTime time.Time) error {
	query := `
		UPDATE adapter_status 
		SET last_event = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE adapter_name = ?
	`

	_, err := db.conn.Exec(query, eventTime, adapterName)
	if err != nil {
		return fmt.Errorf("failed to update last event for adapter %s: %w", adapterName, err)
	}

	return nil
}

// GetAdapterStatus retrieves the status of a specific adapter
func (db *DB) GetAdapterStatus(adapterName string) (*AdapterStatus, error) {
	query := `
		SELECT adapter_name, status, last_event, error_message, updated_at
		FROM adapter_status 
		WHERE adapter_name = ?
	`

	status := &AdapterStatus{}
	var lastEvent sql.NullTime
	var errorMessage sql.NullString

	err := db.conn.QueryRow(query, adapterName).Scan(
		&status.AdapterName,
		&status.Status,
		&lastEvent,
		&errorMessage,
		&status.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("adapter %s not found", adapterName)
		}
		return nil, fmt.Errorf("failed to get adapter status for %s: %w", adapterName, err)
	}

	if lastEvent.Valid {
		status.LastEvent = &lastEvent.Time
	}
	if errorMessage.Valid {
		status.ErrorMessage = errorMessage.String
	}

	return status, nil
}

// GetAllAdapterStatuses retrieves the status of all adapters
func (db *DB) GetAllAdapterStatuses() ([]*AdapterStatus, error) {
	query := `
		SELECT adapter_name, status, last_event, error_message, updated_at
		FROM adapter_status 
		ORDER BY adapter_name
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query adapter statuses: %w", err)
	}
	defer rows.Close()

	var statuses []*AdapterStatus
	for rows.Next() {
		status := &AdapterStatus{}
		var lastEvent sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&status.AdapterName,
			&status.Status,
			&lastEvent,
			&errorMessage,
			&status.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan adapter status row: %w", err)
		}

		if lastEvent.Valid {
			status.LastEvent = &lastEvent.Time
		}
		if errorMessage.Valid {
			status.ErrorMessage = errorMessage.String
		}

		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating adapter status rows: %w", err)
	}

	return statuses, nil
}

// GetActiveAdapters returns a list of adapter names that are currently active
func (db *DB) GetActiveAdapters() ([]string, error) {
	query := `
		SELECT adapter_name 
		FROM adapter_status 
		WHERE status = ? 
		ORDER BY adapter_name
	`

	rows, err := db.conn.Query(query, AdapterStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to query active adapters: %w", err)
	}
	defer rows.Close()

	var adapters []string
	for rows.Next() {
		var adapterName string
		if err := rows.Scan(&adapterName); err != nil {
			return nil, fmt.Errorf("failed to scan adapter name: %w", err)
		}
		adapters = append(adapters, adapterName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active adapter rows: %w", err)
	}

	return adapters, nil
}

// DeleteAdapterStatus removes an adapter status record
func (db *DB) DeleteAdapterStatus(adapterName string) error {
	query := "DELETE FROM adapter_status WHERE adapter_name = ?"
	_, err := db.conn.Exec(query, adapterName)
	if err != nil {
		return fmt.Errorf("failed to delete adapter status for %s: %w", adapterName, err)
	}
	return nil
}