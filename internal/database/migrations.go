package database

import (
	"fmt"
)

// migrate runs database migrations to create the required schema
func (db *DB) migrate() error {
	migrations := []string{
		createEventQueueTable,
		createDeviceConfigTable,
		createAdapterStatusTable,
		createExternalUserMappingsTable,
		createIndexes,
	}
	
	// Run main migrations
	for _, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	
	// Handle device_id column migration separately (for existing databases)
	if err := db.migrateDeviceIdColumn(); err != nil {
		return fmt.Errorf("device_id column migration failed: %w", err)
	}
	
	return nil
}

// migrateDeviceIdColumn adds the device_id column if it doesn't exist
func (db *DB) migrateDeviceIdColumn() error {
	// Check if device_id column exists
	var columnExists bool
	query := `SELECT COUNT(*) FROM pragma_table_info('event_queue') WHERE name='device_id'`
	err := db.conn.QueryRow(query).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check device_id column existence: %w", err)
	}
	
	// Add column if it doesn't exist
	if !columnExists {
		_, err := db.conn.Exec(addDeviceIdToEventQueue)
		if err != nil {
			return fmt.Errorf("failed to add device_id column: %w", err)
		}
	}
	
	return nil
}

const createEventQueueTable = `
CREATE TABLE IF NOT EXISTS event_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE NOT NULL,
    external_user_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('entry', 'exit', 'denied')),
    is_simulated BOOLEAN DEFAULT FALSE,
    device_id TEXT NOT NULL DEFAULT '',
    raw_data TEXT, -- Encrypted JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME NULL,
    retry_count INTEGER DEFAULT 0
);`

const createDeviceConfigTable = `
CREATE TABLE IF NOT EXISTS device_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL, -- Encrypted if sensitive
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createAdapterStatusTable = `
CREATE TABLE IF NOT EXISTS adapter_status (
    adapter_name TEXT PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('active', 'error', 'disabled')),
    last_event DATETIME,
    error_message TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createExternalUserMappingsTable = `
CREATE TABLE IF NOT EXISTS external_user_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    external_user_id TEXT UNIQUE NOT NULL,
    internal_user_id TEXT NOT NULL,
    user_name TEXT,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createIndexes = `
CREATE INDEX IF NOT EXISTS idx_event_queue_timestamp ON event_queue(timestamp);
CREATE INDEX IF NOT EXISTS idx_event_queue_sent_at ON event_queue(sent_at);
CREATE INDEX IF NOT EXISTS idx_event_queue_retry_count ON event_queue(retry_count);
CREATE INDEX IF NOT EXISTS idx_event_queue_device_id ON event_queue(device_id);
CREATE INDEX IF NOT EXISTS idx_adapter_status_updated_at ON adapter_status(updated_at);
CREATE INDEX IF NOT EXISTS idx_external_user_mappings_external_id ON external_user_mappings(external_user_id);
CREATE INDEX IF NOT EXISTS idx_external_user_mappings_internal_id ON external_user_mappings(internal_user_id);
`

const addDeviceIdToEventQueue = `
ALTER TABLE event_queue ADD COLUMN device_id TEXT NOT NULL DEFAULT '';`