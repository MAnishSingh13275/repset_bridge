package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueuePreserver_PreserveQueue(t *testing.T) {
	tempDir := t.TempDir()
	databasePath := filepath.Join(tempDir, "queue.db")
	backupDir := filepath.Join(tempDir, "backups")
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	qp := NewQueuePreserver(logger, databasePath, backupDir)
	
	// Test with no database file
	ctx := context.Background()
	backupPath, err := qp.PreserveQueue(ctx)
	assert.NoError(t, err)
	assert.Empty(t, backupPath)
	
	// Create database file
	dbContent := []byte("test database content")
	err = os.WriteFile(databasePath, dbContent, 0644)
	require.NoError(t, err)
	
	// Preserve queue
	backupPath, err = qp.PreserveQueue(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, backupPath)
	
	// Verify backup was created
	assert.FileExists(t, backupPath)
	
	// Verify backup content
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, dbContent, backupContent)
	
	// Verify backup filename format
	assert.Contains(t, backupPath, "queue_backup_")
	assert.Contains(t, backupPath, ".db")
}

func TestQueuePreserver_RestoreQueue(t *testing.T) {
	tempDir := t.TempDir()
	databasePath := filepath.Join(tempDir, "queue.db")
	backupDir := filepath.Join(tempDir, "backups")
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	qp := NewQueuePreserver(logger, databasePath, backupDir)
	ctx := context.Background()
	
	// Test with empty backup path
	err := qp.RestoreQueue(ctx, "")
	assert.NoError(t, err)
	
	// Create backup file
	err = os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)
	
	backupPath := filepath.Join(backupDir, "queue_backup_test.db")
	backupContent := []byte("backup database content")
	err = os.WriteFile(backupPath, backupContent, 0644)
	require.NoError(t, err)
	
	// Create current database file
	currentContent := []byte("current database content")
	err = os.WriteFile(databasePath, currentContent, 0644)
	require.NoError(t, err)
	
	// Restore queue
	err = qp.RestoreQueue(ctx, backupPath)
	assert.NoError(t, err)
	
	// Verify database was restored
	restoredContent, err := os.ReadFile(databasePath)
	require.NoError(t, err)
	assert.Equal(t, backupContent, restoredContent)
	
	// Test with non-existent backup
	err = qp.RestoreQueue(ctx, "/non/existent/backup.db")
	assert.Error(t, err)
}

func TestQueuePreserver_ValidateQueue(t *testing.T) {
	tempDir := t.TempDir()
	databasePath := filepath.Join(tempDir, "queue.db")
	backupDir := filepath.Join(tempDir, "backups")
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	qp := NewQueuePreserver(logger, databasePath, backupDir)
	ctx := context.Background()
	
	// Test with no database file (should not error)
	err := qp.ValidateQueue(ctx)
	assert.NoError(t, err)
	
	// Create empty database file
	err = os.WriteFile(databasePath, []byte{}, 0644)
	require.NoError(t, err)
	
	// Validate empty database (should warn but not error)
	err = qp.ValidateQueue(ctx)
	assert.NoError(t, err)
	
	// Create database with content
	dbContent := []byte("database with content")
	err = os.WriteFile(databasePath, dbContent, 0644)
	require.NoError(t, err)
	
	// Validate database with content
	err = qp.ValidateQueue(ctx)
	assert.NoError(t, err)
}

func TestQueuePreserver_CleanupOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	databasePath := filepath.Join(tempDir, "queue.db")
	backupDir := filepath.Join(tempDir, "backups")
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	qp := NewQueuePreserver(logger, databasePath, backupDir)
	ctx := context.Background()
	
	// Test with no backup directory
	err := qp.CleanupOldBackups(ctx)
	assert.NoError(t, err)
	
	// Create backup directory
	err = os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)
	
	// Create more backup files than the limit (5)
	backupFiles := []string{
		"queue_backup_20240101_100000.db",
		"queue_backup_20240101_110000.db",
		"queue_backup_20240101_120000.db",
		"queue_backup_20240101_130000.db",
		"queue_backup_20240101_140000.db",
		"queue_backup_20240101_150000.db",
		"queue_backup_20240101_160000.db",
	}
	
	for i, filename := range backupFiles {
		filePath := filepath.Join(backupDir, filename)
		err = os.WriteFile(filePath, []byte("backup content"), 0644)
		require.NoError(t, err)
		
		// Set different modification times (newer files have higher timestamps)
		modTime := time.Unix(int64(1000+i*100), 0)
		err = os.Chtimes(filePath, modTime, modTime)
		require.NoError(t, err)
	}
	
	// Add a non-backup file that should not be removed
	nonBackupFile := filepath.Join(backupDir, "other_file.txt")
	err = os.WriteFile(nonBackupFile, []byte("other content"), 0644)
	require.NoError(t, err)
	
	// Cleanup old backups
	err = qp.CleanupOldBackups(ctx)
	assert.NoError(t, err)
	
	// Check remaining files
	entries, err := os.ReadDir(backupDir)
	require.NoError(t, err)
	
	backupCount := 0
	otherCount := 0
	
	for _, entry := range entries {
		if qp.isQueueBackupFile(entry.Name()) {
			backupCount++
		} else {
			otherCount++
		}
	}
	
	// Should keep only 5 most recent backups (maxBackups constant in the function)
	assert.Equal(t, 5, backupCount)
	// Non-backup file should still exist
	assert.Equal(t, 1, otherCount)
	
	// Verify that the newest backups were kept
	assert.FileExists(t, filepath.Join(backupDir, "queue_backup_20240101_160000.db"))
	assert.FileExists(t, filepath.Join(backupDir, "queue_backup_20240101_150000.db"))
	assert.NoFileExists(t, filepath.Join(backupDir, "queue_backup_20240101_100000.db"))
	assert.NoFileExists(t, filepath.Join(backupDir, "queue_backup_20240101_110000.db"))
}

func TestQueuePreserver_IsQueueBackupFile(t *testing.T) {
	logger := logrus.New()
	qp := NewQueuePreserver(logger, "", "")
	
	tests := []struct {
		filename string
		expected bool
	}{
		{"queue_backup_20240101_100000.db", true},
		{"queue_backup_test.db", true},
		{"queue_backup_123.db", true},
		{"other_backup.db", false},
		{"queue_backup.txt", false},
		{"backup_queue.db", false},
		{"queue_backup", false},
		{"queue_backup_.db", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := qp.isQueueBackupFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueuePreserver_CopyFile(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create source file
	srcFile := filepath.Join(tempDir, "source.db")
	srcContent := []byte("test database content")
	err := os.WriteFile(srcFile, srcContent, 0644)
	require.NoError(t, err)
	
	// Create queue preserver
	logger := logrus.New()
	qp := NewQueuePreserver(logger, "", "")
	
	// Copy file
	dstFile := filepath.Join(tempDir, "destination.db")
	err = qp.copyFile(srcFile, dstFile)
	assert.NoError(t, err)
	
	// Verify copy
	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcContent, dstContent)
	
	// Verify permissions
	srcInfo, err := os.Stat(srcFile)
	require.NoError(t, err)
	dstInfo, err := os.Stat(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
	
	// Test with non-existent source
	err = qp.copyFile("/non/existent/file", dstFile)
	assert.Error(t, err)
}