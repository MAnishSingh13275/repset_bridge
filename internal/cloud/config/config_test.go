package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"CLOUD_API_PORT", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD",
		"REDIS_HOST", "REDIS_PORT", "JWT_SECRET", "HMAC_SECRET",
	}
	
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}
	
	// Clean environment
	for _, key := range envVars {
		os.Unsetenv(key)
	}
	
	defer func() {
		// Restore original environment
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("load with defaults", func(t *testing.T) {
		// Set required environment variables
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("JWT_SECRET", "test-jwt-secret")
		os.Setenv("HMAC_SECRET", "test-hmac-secret")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Test default values
		if cfg.Server.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
		}

		if cfg.Database.Host != "localhost" {
			t.Errorf("Expected default DB host localhost, got %s", cfg.Database.Host)
		}

		if cfg.Database.Port != 5432 {
			t.Errorf("Expected default DB port 5432, got %d", cfg.Database.Port)
		}

		if cfg.Redis.Host != "localhost" {
			t.Errorf("Expected default Redis host localhost, got %s", cfg.Redis.Host)
		}

		if cfg.Auth.JWTExpiration != 24*time.Hour {
			t.Errorf("Expected default JWT expiration 24h, got %v", cfg.Auth.JWTExpiration)
		}
	})

	t.Run("load with custom values", func(t *testing.T) {
		os.Setenv("CLOUD_API_PORT", "9090")
		os.Setenv("DB_HOST", "db.example.com")
		os.Setenv("DB_PORT", "5433")
		os.Setenv("DB_PASSWORD", "custompass")
		os.Setenv("REDIS_HOST", "redis.example.com")
		os.Setenv("JWT_SECRET", "custom-jwt-secret")
		os.Setenv("HMAC_SECRET", "custom-hmac-secret")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if cfg.Server.Port != 9090 {
			t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
		}

		if cfg.Database.Host != "db.example.com" {
			t.Errorf("Expected DB host db.example.com, got %s", cfg.Database.Host)
		}

		if cfg.Database.Port != 5433 {
			t.Errorf("Expected DB port 5433, got %d", cfg.Database.Port)
		}

		if cfg.Redis.Host != "redis.example.com" {
			t.Errorf("Expected Redis host redis.example.com, got %s", cfg.Redis.Host)
		}
	})

	t.Run("missing required DB_PASSWORD", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "test-jwt-secret")
		os.Setenv("HMAC_SECRET", "test-hmac-secret")
		// DB_PASSWORD not set

		_, err := Load()
		if err == nil {
			t.Error("Expected error for missing DB_PASSWORD")
		}
	})

	t.Run("missing required JWT_SECRET", func(t *testing.T) {
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("HMAC_SECRET", "test-hmac-secret")
		// JWT_SECRET not set

		_, err := Load()
		if err == nil {
			t.Error("Expected error for missing JWT_SECRET")
		}
	})

	t.Run("missing required HMAC_SECRET", func(t *testing.T) {
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("JWT_SECRET", "test-jwt-secret")
		// HMAC_SECRET not set

		_, err := Load()
		if err == nil {
			t.Error("Expected error for missing HMAC_SECRET")
		}
	})
}

func TestConnectionString(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	actual := cfg.ConnectionString()

	if actual != expected {
		t.Errorf("Expected connection string %s, got %s", expected, actual)
	}
}

func TestRedisAddr(t *testing.T) {
	cfg := RedisConfig{
		Host: "redis.example.com",
		Port: 6380,
	}

	expected := "redis.example.com:6380"
	actual := cfg.RedisAddr()

	if actual != expected {
		t.Errorf("Expected Redis address %s, got %s", expected, actual)
	}
}