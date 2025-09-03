package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the cloud API configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Auth     AuthConfig     `yaml:"auth"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
	IdleTimeout  int `yaml:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Database     string `yaml:"database"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	SSLMode      string `yaml:"ssl_mode"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxLifetime  int    `yaml:"max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
	PoolSize int    `yaml:"pool_size"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string        `yaml:"jwt_secret"`
	JWTExpiration time.Duration `yaml:"jwt_expiration"`
	HMACSecret    string        `yaml:"hmac_secret"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("CLOUD_API_PORT", 8080),
			ReadTimeout:  getEnvInt("CLOUD_API_READ_TIMEOUT", 30),
			WriteTimeout: getEnvInt("CLOUD_API_WRITE_TIMEOUT", 30),
			IdleTimeout:  getEnvInt("CLOUD_API_IDLE_TIMEOUT", 120),
		},
		Database: DatabaseConfig{
			Host:         getEnvString("DB_HOST", "localhost"),
			Port:         getEnvInt("DB_PORT", 5432),
			Database:     getEnvString("DB_NAME", "gym_bridge_cloud"),
			Username:     getEnvString("DB_USER", "postgres"),
			Password:     getEnvString("DB_PASSWORD", ""),
			SSLMode:      getEnvString("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:  getEnvInt("DB_MAX_LIFETIME", 300),
		},
		Redis: RedisConfig{
			Host:     getEnvString("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnvString("REDIS_PASSWORD", ""),
			Database: getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},
		Auth: AuthConfig{
			JWTSecret:     getEnvString("JWT_SECRET", "your-secret-key"),
			JWTExpiration: time.Duration(getEnvInt("JWT_EXPIRATION_HOURS", 24)) * time.Hour,
			HMACSecret:    getEnvString("HMAC_SECRET", "your-hmac-secret"),
		},
		Logging: LoggingConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
	}

	// Validate required configuration
	if config.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD environment variable is required")
	}

	if config.Auth.JWTSecret == "your-secret-key" {
		return nil, fmt.Errorf("JWT_SECRET environment variable must be set")
	}

	if config.Auth.HMACSecret == "your-hmac-secret" {
		return nil, fmt.Errorf("HMAC_SECRET environment variable must be set")
	}

	return config, nil
}

// ConnectionString returns the PostgreSQL connection string
func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.Username, d.Password, d.Database, d.SSLMode)
}

// RedisAddr returns the Redis address
func (r *RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}