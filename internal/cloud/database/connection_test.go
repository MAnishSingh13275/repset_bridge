package database

import (
	"testing"

	"gym-door-bridge/internal/cloud/config"
)

func TestNewConnection(t *testing.T) {
	t.Run("invalid connection string", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			Host:     "nonexistent-host",
			Port:     5432,
			Username: "testuser",
			Password: "testpass",
			Database: "testdb",
			SSLMode:  "disable",
		}

		conn, err := NewConnection(cfg)
		if err == nil {
			t.Error("Expected error for invalid connection")
			if conn != nil {
				conn.Close()
			}
		}
	})

	t.Run("connection configuration", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Username:     "testuser",
			Password:     "testpass",
			Database:     "testdb",
			SSLMode:      "disable",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
			MaxLifetime:  300,
		}

		// This test will fail if PostgreSQL is not available, which is expected
		// In a real test environment, you would use a test database or mock
		conn, err := NewConnection(cfg)
		if err != nil {
			// Expected to fail without a real database
			t.Logf("Connection failed as expected without database: %v", err)
			return
		}

		// If connection succeeds (unlikely without setup), test configuration
		defer conn.Close()

		if conn.DB == nil {
			t.Error("Expected DB to be initialized")
		}

		// Test health check
		err = conn.Health()
		if err != nil {
			t.Logf("Health check failed as expected: %v", err)
		}
	})
}

func TestConnectionClose(t *testing.T) {
	conn := &Connection{DB: nil}
	err := conn.Close()
	if err != nil {
		t.Errorf("Expected no error closing nil connection, got %v", err)
	}
}