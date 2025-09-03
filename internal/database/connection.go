package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// PerformanceTier represents the system performance tier
type PerformanceTier string

const (
	TierLite   PerformanceTier = "lite"
	TierNormal PerformanceTier = "normal"
	TierFull   PerformanceTier = "full"
)

// DB wraps the SQLite database connection with encryption capabilities
type DB struct {
	conn   *sql.DB
	cipher cipher.AEAD
	tier   PerformanceTier
}

// Config holds database configuration options
type Config struct {
	DatabasePath   string
	EncryptionKey  []byte
	PerformanceTier PerformanceTier
}

// NewDB creates a new database connection with the specified configuration
func NewDB(config Config) (*DB, error) {
	// Ensure database directory exists
	if err := os.MkdirAll(filepath.Dir(config.DatabasePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite connection with WAL mode
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_foreign_keys=on", config.DatabasePath)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set up AES-GCM encryption
	block, err := aes.NewCipher(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	db := &DB{
		conn:   conn,
		cipher: gcm,
		tier:   config.PerformanceTier,
	}

	// Configure tier-specific pragmas
	if err := db.configurePragmas(); err != nil {
		return nil, fmt.Errorf("failed to configure pragmas: %w", err)
	}

	// Run migrations
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// configurePragmas sets tier-specific SQLite pragmas
func (db *DB) configurePragmas() error {
	var syncMode string
	switch db.tier {
	case TierLite:
		syncMode = "NORMAL"
	case TierNormal:
		syncMode = "NORMAL"
	case TierFull:
		syncMode = "FULL"
	default:
		syncMode = "NORMAL"
	}

	pragmas := []string{
		fmt.Sprintf("PRAGMA synchronous = %s", syncMode),
		"PRAGMA cache_size = -64000", // 64MB cache
		"PRAGMA temp_store = memory",
		"PRAGMA mmap_size = 268435456", // 256MB mmap
	}

	for _, pragma := range pragmas {
		if _, err := db.conn.Exec(pragma); err != nil {
			return fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Encrypt encrypts data using AES-GCM
func (db *DB) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, db.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := db.cipher.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts data using AES-GCM
func (db *DB) Decrypt(ciphertext string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	nonceSize := db.cipher.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := db.cipher.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// QueryRow executes a query that is expected to return at most one row
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// Query executes a query that returns rows
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// Exec executes a query without returning any rows
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}