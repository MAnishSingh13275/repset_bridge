package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthChecker_PerformHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse int
		expectError    bool
	}{
		{
			name:           "healthy service",
			serverResponse: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "unhealthy service",
			serverResponse: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "service unavailable",
			serverResponse: http.StatusServiceUnavailable,
			expectError:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverResponse)
			}))
			defer server.Close()
			
			// Create health checker
			tempDir := t.TempDir()
			config := &UpdaterConfig{
				UpdateDir: filepath.Join(tempDir, "updates"),
				BackupDir: filepath.Join(tempDir, "backups"),
			}
			
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests
			
			hc := NewHealthChecker(config, logger, server.URL)
			
			// Perform health check
			ctx := context.Background()
			err := hc.performHealthCheck(ctx)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHealthChecker_FindLatestBackup(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	config := &UpdaterConfig{
		BackupDir: backupDir,
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	hc := NewHealthChecker(config, logger, "http://localhost")
	
	// Test with no backups
	_, err := hc.findLatestBackup()
	assert.Error(t, err)
	
	// Create backup directory and files
	err = os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)
	
	// Create test backup files with different timestamps
	backupFiles := []struct {
		name    string
		modTime time.Time
	}{
		{"bridge_backup_1000", time.Unix(1000, 0)},
		{"bridge_backup_2000", time.Unix(2000, 0)}, // This should be the latest
		{"bridge_backup_1500", time.Unix(1500, 0)},
	}
	
	for _, backup := range backupFiles {
		filePath := filepath.Join(backupDir, backup.name)
		err = os.WriteFile(filePath, []byte("backup content"), 0644)
		require.NoError(t, err)
		
		// Set modification time
		err = os.Chtimes(filePath, backup.modTime, backup.modTime)
		require.NoError(t, err)
	}
	
	// Find latest backup
	latestBackup, err := hc.findLatestBackup()
	assert.NoError(t, err)
	assert.Contains(t, latestBackup, "bridge_backup_2000")
}

func TestHealthChecker_IsBackupFile(t *testing.T) {
	config := &UpdaterConfig{}
	logger := logrus.New()
	hc := NewHealthChecker(config, logger, "http://localhost")
	
	tests := []struct {
		filename string
		expected bool
	}{
		{"bridge_backup_123", true},
		{"bridge.old", true},
		{"bridge_backup_123.exe", true},
		{"bridge.old.exe", true},
		{"other_file", false},
		{"bridge.txt", false},
		{"backup_bridge", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := hc.isBackupFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthChecker_CopyFile(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create source file
	srcFile := filepath.Join(tempDir, "source.txt")
	srcContent := []byte("test file content")
	err := os.WriteFile(srcFile, srcContent, 0644)
	require.NoError(t, err)
	
	// Create health checker
	config := &UpdaterConfig{}
	logger := logrus.New()
	hc := NewHealthChecker(config, logger, "http://localhost")
	
	// Copy file
	dstFile := filepath.Join(tempDir, "destination.txt")
	err = hc.copyFile(srcFile, dstFile)
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
}

func TestHealthChecker_ScheduleRestart(t *testing.T) {
	tempDir := t.TempDir()
	updateDir := filepath.Join(tempDir, "updates")
	
	config := &UpdaterConfig{
		UpdateDir: updateDir,
	}
	
	logger := logrus.New()
	hc := NewHealthChecker(config, logger, "http://localhost")
	
	// Create update directory first
	err := os.MkdirAll(updateDir, 0755)
	require.NoError(t, err)
	
	// Schedule restart
	err = hc.scheduleRestart()
	assert.NoError(t, err)
	
	// Check that restart signal file was created
	restartFile := filepath.Join(updateDir, "restart_required")
	assert.FileExists(t, restartFile)
	
	// Check file content
	content, err := os.ReadFile(restartFile)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	
	// Should be a valid timestamp
	_, err = time.Parse(time.RFC3339, string(content))
	assert.NoError(t, err)
}

func TestHealthChecker_CleanupOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	
	config := &UpdaterConfig{
		BackupDir: backupDir,
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	hc := NewHealthChecker(config, logger, "http://localhost")
	
	// Create backup directory
	err := os.MkdirAll(backupDir, 0755)
	require.NoError(t, err)
	
	// Create more backup files than the limit (5)
	backupFiles := []string{
		"bridge_backup_1",
		"bridge_backup_2",
		"bridge_backup_3",
		"bridge_backup_4",
		"bridge_backup_5",
		"bridge_backup_6",
		"bridge_backup_7",
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
	err = hc.cleanupOldBackups()
	assert.NoError(t, err)
	
	// Check remaining files
	entries, err := os.ReadDir(backupDir)
	require.NoError(t, err)
	
	backupCount := 0
	otherCount := 0
	
	for _, entry := range entries {
		if hc.isBackupFile(entry.Name()) {
			backupCount++
		} else {
			otherCount++
		}
	}
	
	// Should keep only 3 most recent backups (maxBackups constant in the function)
	assert.Equal(t, 3, backupCount)
	// Non-backup file should still exist
	assert.Equal(t, 1, otherCount)
	
	// Verify that the newest backups were kept
	assert.FileExists(t, filepath.Join(backupDir, "bridge_backup_7"))
	assert.FileExists(t, filepath.Join(backupDir, "bridge_backup_6"))
	assert.FileExists(t, filepath.Join(backupDir, "bridge_backup_5"))
	assert.NoFileExists(t, filepath.Join(backupDir, "bridge_backup_1"))
}