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

	for i, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", i+1, err)
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
CREATE INDEX IF NOT EXISTS idx_adapter_status_updated_at ON adapter_status(updated_at);
CREATE INDEX IF NOT EXISTS idx_external_user_mappings_external_id ON external_user_mappings(external_user_id);
CREATE INDEX IF NOT EXISTS idx_external_user_mappings_internal_id ON external_user_mappings(internal_user_id);
`