package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthChecker performs health checks after updates
type HealthChecker struct {
	config     *UpdaterConfig
	logger     *logrus.Logger
	client     *http.Client
	healthURL  string
	maxRetries int
	retryDelay time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *UpdaterConfig, logger *logrus.Logger, healthURL string) *HealthChecker {
	return &HealthChecker{
		config:     config,
		logger:     logger,
		client:     &http.Client{Timeout: 10 * time.Second},
		healthURL:  healthURL,
		maxRetries: 5,
		retryDelay: 30 * time.Second,
	}
}

// CheckHealthAfterUpdate performs health checks after an update and rolls back if needed
func (hc *HealthChecker) CheckHealthAfterUpdate(ctx context.Context) error {
	hc.logger.Info("Starting post-update health check")
	
	// Wait a bit for the service to fully start
	time.Sleep(10 * time.Second)
	
	// Perform health checks with retries
	for attempt := 1; attempt <= hc.maxRetries; attempt++ {
		hc.logger.WithField("attempt", attempt).Debug("Performing health check")
		
		if err := hc.performHealthCheck(ctx); err != nil {
			hc.logger.WithError(err).WithField("attempt", attempt).Warn("Health check failed")
			
			if attempt == hc.maxRetries {
				hc.logger.Error("All health checks failed, initiating rollback")
				return hc.rollback()
			}
			
			// Wait before next attempt
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(hc.retryDelay):
				continue
			}
		} else {
			hc.logger.Info("Health check passed, update successful")
			return hc.cleanupOldBackups()
		}
	}
	
	return nil
}

// performHealthCheck performs a single health check
func (hc *HealthChecker) performHealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", hc.healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status: %d", resp.StatusCode)
	}
	
	return nil
}

// rollback rolls back to the previous version
func (hc *HealthChecker) rollback() error {
	hc.logger.Warn("Initiating rollback to previous version")
	
	// Find the most recent backup
	backupPath, err := hc.findLatestBackup()
	if err != nil {
		return fmt.Errorf("failed to find backup for rollback: %w", err)
	}
	
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	
	// Replace current executable with backup
	if err := hc.replaceExecutable(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	hc.logger.WithField("backup_path", backupPath).Info("Rollback completed successfully")
	
	// Schedule restart
	return hc.scheduleRestart()
}

// findLatestBackup finds the most recent backup file
func (hc *HealthChecker) findLatestBackup() (string, error) {
	entries, err := os.ReadDir(hc.config.BackupDir)
	if err != nil {
		return "", fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	var latestBackup string
	var latestTime time.Time
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		if !hc.isBackupFile(name) {
			continue
		}
		
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestBackup = filepath.Join(hc.config.BackupDir, name)
		}
	}
	
	if latestBackup == "" {
		return "", fmt.Errorf("no backup files found")
	}
	
	return latestBackup, nil
}

// isBackupFile checks if a filename is a backup file
func (hc *HealthChecker) isBackupFile(filename string) bool {
	base := filepath.Base(filename)
	
	if runtime.GOOS == "windows" {
		// Check for .exe files
		if filepath.Ext(filename) == ".exe" {
			base = base[:len(base)-4] // Remove .exe extension
		}
	}
	
	// Check for backup patterns
	if len(base) >= 13 && base[:13] == "bridge_backup" {
		return true
	}
	
	if base == "bridge.old" {
		return true
	}
	
	return false
}

// replaceExecutable replaces the current executable with the backup
func (hc *HealthChecker) replaceExecutable(currentPath, backupPath string) error {
	// On Windows, we might need to handle file locking differently
	if runtime.GOOS == "windows" {
		return hc.replaceExecutableWindows(currentPath, backupPath)
	}
	
	// On Unix-like systems, we can usually replace the file directly
	return os.Rename(backupPath, currentPath)
}

// replaceExecutableWindows handles executable replacement on Windows
func (hc *HealthChecker) replaceExecutableWindows(currentPath, backupPath string) error {
	// On Windows, we might need to rename the current executable first
	tempPath := currentPath + ".failed"
	
	// Remove any existing .failed file
	os.Remove(tempPath)
	
	// Rename current executable
	if err := os.Rename(currentPath, tempPath); err != nil {
		return fmt.Errorf("failed to rename current executable: %w", err)
	}
	
	// Copy backup to current location
	if err := hc.copyFile(backupPath, currentPath); err != nil {
		// Try to restore original
		os.Rename(tempPath, currentPath)
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	return nil
}

// copyFile copies a file from src to dst
func (hc *HealthChecker) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	
	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}
	
	return os.Chmod(dst, sourceInfo.Mode())
}

// scheduleRestart schedules a restart of the service
func (hc *HealthChecker) scheduleRestart() error {
	hc.logger.Info("Scheduling service restart for rollback")
	
	// Create a restart signal file that the service manager can detect
	restartFile := filepath.Join(hc.config.UpdateDir, "restart_required")
	if err := os.WriteFile(restartFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("failed to create restart signal file: %w", err)
	}
	
	return nil
}

// cleanupOldBackups removes old backup files to save disk space
func (hc *HealthChecker) cleanupOldBackups() error {
	hc.logger.Debug("Cleaning up old backup files")
	
	entries, err := os.ReadDir(hc.config.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	// Keep only the 3 most recent backups
	const maxBackups = 3
	var backupFiles []os.FileInfo
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		if !hc.isBackupFile(entry.Name()) {
			continue
		}
		
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		backupFiles = append(backupFiles, info)
	}
	
	// Sort by modification time (newest first)
	for i := 0; i < len(backupFiles)-1; i++ {
		for j := i + 1; j < len(backupFiles); j++ {
			if backupFiles[i].ModTime().Before(backupFiles[j].ModTime()) {
				backupFiles[i], backupFiles[j] = backupFiles[j], backupFiles[i]
			}
		}
	}
	
	// Remove old backups
	for i := maxBackups; i < len(backupFiles); i++ {
		backupPath := filepath.Join(hc.config.BackupDir, backupFiles[i].Name())
		if err := os.Remove(backupPath); err != nil {
			hc.logger.WithError(err).WithField("file", backupPath).Warn("Failed to remove old backup")
		} else {
			hc.logger.WithField("file", backupPath).Debug("Removed old backup file")
		}
	}
	
	return nil
}