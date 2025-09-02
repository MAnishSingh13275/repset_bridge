package database

import (
	"fmt"
	"strings"
)

// sensitiveKeys are configuration keys that should be encrypted
var sensitiveKeys = map[string]bool{
	"device_key":     true,
	"encryption_key": true,
	"api_secret":     true,
	"hmac_key":       true,
}

// SetConfig stores a configuration value, encrypting it if it's sensitive
func (db *DB) SetConfig(key, value string) error {
	var storedValue string
	var err error

	// Encrypt sensitive configuration values
	if sensitiveKeys[strings.ToLower(key)] {
		storedValue, err = db.Encrypt([]byte(value))
		if err != nil {
			return fmt.Errorf("failed to encrypt config value for key %s: %w", key, err)
		}
	} else {
		storedValue = value
	}

	query := `
		INSERT OR REPLACE INTO device_config (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	_, err = db.conn.Exec(query, key, storedValue)
	if err != nil {
		return fmt.Errorf("failed to set config %s: %w", key, err)
	}

	return nil
}

// GetConfig retrieves a configuration value, decrypting it if necessary
func (db *DB) GetConfig(key string) (string, error) {
	var value string
	
	query := "SELECT value FROM device_config WHERE key = ?"
	err := db.conn.QueryRow(query, key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("failed to get config %s: %w", key, err)
	}

	// Decrypt sensitive configuration values
	if sensitiveKeys[strings.ToLower(key)] {
		decrypted, err := db.Decrypt(value)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt config value for key %s: %w", key, err)
		}
		return string(decrypted), nil
	}

	return value, nil
}

// GetAllConfig retrieves all configuration key-value pairs
func (db *DB) GetAllConfig() (map[string]string, error) {
	query := "SELECT key, value FROM device_config"
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all config: %w", err)
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan config row: %w", err)
		}

		// Decrypt sensitive values
		if sensitiveKeys[strings.ToLower(key)] {
			decrypted, err := db.Decrypt(value)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt config value for key %s: %w", key, err)
			}
			config[key] = string(decrypted)
		} else {
			config[key] = value
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating config rows: %w", err)
	}

	return config, nil
}

// DeleteConfig removes a configuration key
func (db *DB) DeleteConfig(key string) error {
	query := "DELETE FROM device_config WHERE key = ?"
	_, err := db.conn.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete config %s: %w", key, err)
	}
	return nil
}

// ConfigExists checks if a configuration key exists
func (db *DB) ConfigExists(key string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM device_config WHERE key = ?"
	err := db.conn.QueryRow(query, key).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if config exists %s: %w", key, err)
	}
	return count > 0, nil
}