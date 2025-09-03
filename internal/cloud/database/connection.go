package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gym-door-bridge/internal/cloud/config"
	_ "github.com/lib/pq"
)

// Connection wraps the database connection with additional functionality
type Connection struct {
	DB *sql.DB
}

// NewConnection creates a new database connection
func NewConnection(cfg config.DatabaseConfig) (*Connection, error) {
	db, err := sql.Open("postgres", cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{DB: db}, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// Health checks the database connection health
func (c *Connection) Health() error {
	return c.DB.Ping()
}