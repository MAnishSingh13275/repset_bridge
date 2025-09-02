package updater

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// QueuePreserver handles queue preservation during updates
type QueuePreserver struct {
	logger       *logrus.Logger
	databasePath string
	backupDir    string
}

// NewQueuePreserver creates a new queue preserver
func NewQueuePreserver(logger *logrus.Logger, databasePath, backupDir string) *QueuePreserver {
	return &QueuePreserver{
		logger:       logger,
		databasePath: databasePath,
		backupDir:    backupDir,
	}
}

// PreserveQueue creates a backup of the queue database before update
func (qp *QueuePreserver) PreserveQueue(ctx context.Context) (string, error) {
	qp.logger.Info("Preserving queue database before update")
	
	// Check if database exists
	if _, err := os.Stat(qp.databasePath); os.IsNotExist(err) {
		qp.logger.Info("No database to preserve")
		return "", nil
	}
	
	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("queue_backup_%s.db", timestamp)
	backupPath := filepath.Join(qp.backupDir, backupFilename)
	
	// Ensure backup directory exists
	if err := os.MkdirAll(qp.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Copy database file
	if err := qp.copyFile(qp.databasePath, backupPath); err != nil {
		return "", fmt.Errorf("failed to backup queue database: %w", err)
	}
	
	qp.logger.WithField("backup_path", backupPath).Info("Queue database backed up successfully")
	return backupPath, nil
}

// RestoreQueue restores the queue database from backup
func (qp *QueuePreserver) RestoreQueue(ctx context.Context, backupPath string) error {
	if backupPath == "" {
		qp.logger.Info("No queue backup to restore")
		return nil
	}
	
	qp.logger.WithField("backup_path", backupPath).Info("Restoring queue database from backup")
	
	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}
	
	// Remove current database if it exists
	if _, err := os.Stat(qp.databasePath); err == nil {
		if err := os.Remove(qp.databasePath); err != nil {
			return fmt.Errorf("failed to remove current database: %w", err)
		}
	}
	
	// Restore from backup
	if err := qp.copyFile(backupPath, qp.databasePath); err != nil {
		return fmt.Errorf("failed to restore queue database: %w", err)
	}
	
	qp.logger.Info("Queue database restored successfully")
	return nil
}

// ValidateQueue performs basic validation on the queue database
func (qp *QueuePreserver) ValidateQueue(ctx context.Context) error {
	qp.logger.Debug("Validating queue database")
	
	// Check if database file exists and is readable
	if _, err := os.Stat(qp.databasePath); err != nil {
		if os.IsNotExist(err) {
			qp.logger.Info("No queue database found (this is normal for new installations)")
			return nil
		}
		return fmt.Errorf("failed to access queue database: %w", err)
	}
	
	// Check file size (basic sanity check)
	info, err := os.Stat(qp.databasePath)
	if err != nil {
		return fmt.Errorf("failed to get database info: %w", err)
	}
	
	if info.Size() == 0 {
		qp.logger.Warn("Queue database is empty")
	} else {
		qp.logger.WithField("size", info.Size()).Debug("Queue database validation passed")
	}
	
	return nil
}

// CleanupOldBackups removes old queue backups to save disk space
func (qp *QueuePreserver) CleanupOldBackups(ctx context.Context) error {
	qp.logger.Debug("Cleaning up old queue backups")
	
	entries, err := os.ReadDir(qp.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No backup directory exists yet
		}
		return fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	// Keep only the 5 most recent queue backups
	const maxBackups = 5
	var backupFiles []os.FileInfo
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		if !qp.isQueueBackupFile(name) {
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
		backupPath := filepath.Join(qp.backupDir, backupFiles[i].Name())
		if err := os.Remove(backupPath); err != nil {
			qp.logger.WithError(err).WithField("file", backupPath).Warn("Failed to remove old queue backup")
		} else {
			qp.logger.WithField("file", backupPath).Debug("Removed old queue backup file")
		}
	}
	
	return nil
}

// isQueueBackupFile checks if a filename is a queue backup file
func (qp *QueuePreserver) isQueueBackupFile(filename string) bool {
	if filepath.Ext(filename) != ".db" {
		return false
	}
	
	base := filename[:len(filename)-3] // Remove .db extension
	if len(base) < 13 {
		return false
	}
	
	if base[:12] != "queue_backup" {
		return false
	}
	
	// Check that there's something after "queue_backup_"
	if len(base) == 12 || (len(base) > 12 && base[12] != '_') {
		return false
	}
	
	return len(base) > 13 // Must have something after "queue_backup_"
}

// copyFile copies a file from src to dst
func (qp *QueuePreserver) copyFile(src, dst string) error {
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