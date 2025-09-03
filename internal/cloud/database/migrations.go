package database

import (
	"database/sql"
	"fmt"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

// migrations contains all database migrations
var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_devices_table",
		Up: `
			CREATE TABLE IF NOT EXISTS devices (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				device_id VARCHAR(255) UNIQUE NOT NULL,
				pair_code VARCHAR(255),
				device_name VARCHAR(255) NOT NULL,
				location VARCHAR(255),
				hostname VARCHAR(255),
				platform VARCHAR(100),
				version VARCHAR(100),
				tier VARCHAR(50),
				status VARCHAR(50) DEFAULT 'active',
				metadata JSONB,
				last_heartbeat TIMESTAMP WITH TIME ZONE,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			CREATE INDEX IF NOT EXISTS idx_devices_device_id ON devices(device_id);
			CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
			CREATE INDEX IF NOT EXISTS idx_devices_last_heartbeat ON devices(last_heartbeat);
		`,
		Down: `DROP TABLE IF EXISTS devices;`,
	},
	{
		Version: 2,
		Name:    "create_users_table",
		Up: `
			CREATE TABLE IF NOT EXISTS users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				external_id VARCHAR(255) UNIQUE NOT NULL,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255),
				phone VARCHAR(50),
				status VARCHAR(50) DEFAULT 'active',
				metadata JSONB,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			CREATE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id);
			CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
			CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
		`,
		Down: `DROP TABLE IF EXISTS users;`,
	},
	{
		Version: 3,
		Name:    "create_permissions_table",
		Up: `
			CREATE TABLE IF NOT EXISTS permissions (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
				access_type VARCHAR(50) NOT NULL,
				time_slots JSONB,
				valid_from TIMESTAMP WITH TIME ZONE,
				valid_until TIMESTAMP WITH TIME ZONE,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			CREATE INDEX IF NOT EXISTS idx_permissions_user_id ON permissions(user_id);
			CREATE INDEX IF NOT EXISTS idx_permissions_device_id ON permissions(device_id);
			CREATE INDEX IF NOT EXISTS idx_permissions_access_type ON permissions(access_type);
			CREATE INDEX IF NOT EXISTS idx_permissions_valid_from ON permissions(valid_from);
			CREATE INDEX IF NOT EXISTS idx_permissions_valid_until ON permissions(valid_until);
		`,
		Down: `DROP TABLE IF EXISTS permissions;`,
	},
	{
		Version: 4,
		Name:    "create_events_table",
		Up: `
			CREATE TABLE IF NOT EXISTS events (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
				user_id UUID REFERENCES users(id) ON DELETE SET NULL,
				event_type VARCHAR(100) NOT NULL,
				event_data JSONB,
				timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
				processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			
			CREATE INDEX IF NOT EXISTS idx_events_device_id ON events(device_id);
			CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
			CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
			CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
			CREATE INDEX IF NOT EXISTS idx_events_processed_at ON events(processed_at);
		`,
		Down: `DROP TABLE IF EXISTS events;`,
	},
	{
		Version: 5,
		Name:    "create_schema_migrations_table",
		Up: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
		`,
		Down: `DROP TABLE IF EXISTS schema_migrations;`,
	},
}

// RunMigrations runs all pending database migrations
func RunMigrations(conn *Connection) error {
	// Ensure schema_migrations table exists
	if err := createMigrationsTable(conn.DB); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current migration version
	currentVersion, err := getCurrentVersion(conn.DB)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		fmt.Printf("Running migration %d: %s\n", migration.Version, migration.Name)
		
		tx, err := conn.DB.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.Up); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
		}

		// Record migration
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			migration.Version, migration.Name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Migration %d completed successfully\n", migration.Version)
	}

	return nil
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	_, err := db.Exec(query)
	return err
}

func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}