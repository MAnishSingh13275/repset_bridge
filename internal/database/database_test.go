package database

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

// setupTestDB creates a temporary database for testing
func setupTestDB(t *testing.T, tier PerformanceTier) *DB {
	t.Helper()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gym-door-bridge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Generate random encryption key
	encryptionKey := make([]byte, 32)
	if _, err := rand.Read(encryptionKey); err != nil {
		t.Fatalf("Failed to generate encryption key: %v", err)
	}

	config := Config{
		DatabasePath:    filepath.Join(tempDir, "test.db"),
		EncryptionKey:   encryptionKey,
		PerformanceTier: tier,
	}

	db, err := NewDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Cleanup function
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(tempDir)
	})

	return db
}

func TestNewDB(t *testing.T) {
	tests := []struct {
		name string
		tier PerformanceTier
	}{
		{"Lite tier", TierLite},
		{"Normal tier", TierNormal},
		{"Full tier", TierFull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t, tt.tier)
			if db == nil {
				t.Fatal("Expected database to be created")
			}
			if db.tier != tt.tier {
				t.Errorf("Expected tier %s, got %s", tt.tier, db.tier)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	testData := []byte("sensitive configuration data")

	// Test encryption
	encrypted, err := db.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if encrypted == string(testData) {
		t.Error("Encrypted data should not match original data")
	}

	// Test decryption
	decrypted, err := db.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Expected decrypted data %s, got %s", string(testData), string(decrypted))
	}
}

func TestEncryptDecryptEmptyData(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	testData := []byte("")

	encrypted, err := db.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt empty data: %v", err)
	}

	decrypted, err := db.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt empty data: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Expected empty decrypted data, got %s", string(decrypted))
	}
}