package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo represents information about a file for cleanup operations
type FileInfo struct {
	Path         string
	Hash         string
	Size         int64
	ModTime      time.Time
	IsDirectory  bool
	ShouldRemove bool
	Reason       string
}

// CleanupConfig holds configuration for the cleanup operation
type CleanupConfig struct {
	BackupDir     string
	DryRun        bool
	Verbose       bool
	SkipPatterns  []string
	TargetDirs    []string
}

// CleanupManager handles the file cleanup operations
type CleanupManager struct {
	config    CleanupConfig
	fileMap   map[string][]FileInfo
	backupMap map[string]string
}

func main() {
	config := CleanupConfig{
		BackupDir:    "cleanup-backup-" + time.Now().Format("20060102-150405"),
		DryRun:       false,
		Verbose:      true,
		SkipPatterns: []string{".git", ".kiro", "node_modules", "vendor"},
		TargetDirs:   []string{"."},
	}

	// Parse command line arguments
	for i, arg := range os.Args[1:] {
		switch arg {
		case "--dry-run":
			config.DryRun = true
		case "--verbose":
			config.Verbose = true
		case "--backup-dir":
			if i+1 < len(os.Args[1:]) {
				config.BackupDir = os.Args[i+2]
			}
		case "--help":
			printUsage()
			return
		}
	}

	manager := NewCleanupManager(config)
	
	if err := manager.Run(); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}
}

func printUsage() {
	fmt.Println("File Cleanup Automation Script")
	fmt.Println("Usage: go run cleanup-automation.go [options]")
	fmt.Println("Options:")
	fmt.Println("  --dry-run       Show what would be done without making changes")
	fmt.Println("  --verbose       Enable verbose output")
	fmt.Println("  --backup-dir    Specify backup directory (default: cleanup-backup-TIMESTAMP)")
	fmt.Println("  --help          Show this help message")
}

func NewCleanupManager(config CleanupConfig) *CleanupManager {
	return &CleanupManager{
		config:    config,
		fileMap:   make(map[string][]FileInfo),
		backupMap: make(map[string]string),
	}
}

// Run executes the cleanup process
func (cm *CleanupManager) Run() error {
	fmt.Println("Starting file cleanup automation...")
	
	// Step 1: Scan files and calculate hashes
	if err := cm.scanFiles(); err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	// Step 2: Identify duplicates
	duplicates := cm.findDuplicates()
	
	// Step 3: Identify files to remove based on project cleanup rules
	filesToRemove := cm.identifyFilesToRemove()
	
	// Step 4: Create backup if not dry run
	if !cm.config.DryRun {
		if err := cm.createBackup(append(duplicates, filesToRemove...)); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Step 5: Remove files
	if err := cm.removeFiles(append(duplicates, filesToRemove...)); err != nil {
		return fmt.Errorf("failed to remove files: %w", err)
	}

	fmt.Println("Cleanup completed successfully!")
	return nil
}

// scanFiles walks through directories and collects file information
func (cm *CleanupManager) scanFiles() error {
	fmt.Println("Scanning files and calculating hashes...")
	
	for _, targetDir := range cm.config.TargetDirs {
		err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and files matching skip patterns
			if cm.shouldSkip(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !info.IsDir() {
				hash, err := cm.calculateFileHash(path)
				if err != nil {
					if cm.config.Verbose {
						fmt.Printf("Warning: Could not hash file %s: %v\n", path, err)
					}
					return nil
				}

				fileInfo := FileInfo{
					Path:        path,
					Hash:        hash,
					Size:        info.Size(),
					ModTime:     info.ModTime(),
					IsDirectory: false,
				}

				cm.fileMap[hash] = append(cm.fileMap[hash], fileInfo)
				
				if cm.config.Verbose {
					fmt.Printf("Scanned: %s (hash: %s)\n", path, hash[:8])
				}
			}

			return nil
		})
		
		if err != nil {
			return err
		}
	}

	return nil
}

// shouldSkip determines if a path should be skipped during scanning
func (cm *CleanupManager) shouldSkip(path string) bool {
	for _, pattern := range cm.config.SkipPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// calculateFileHash computes SHA256 hash of a file
func (cm *CleanupManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// findDuplicates identifies duplicate files based on hash comparison
func (cm *CleanupManager) findDuplicates() []FileInfo {
	var duplicates []FileInfo
	
	fmt.Println("Identifying duplicate files...")
	
	for _, files := range cm.fileMap {
		if len(files) > 1 {
			// Keep the first file (usually the one in the most appropriate location)
			// Mark others as duplicates
			for i := 1; i < len(files); i++ {
				duplicate := files[i]
				duplicate.ShouldRemove = true
				duplicate.Reason = fmt.Sprintf("Duplicate of %s", files[0].Path)
				duplicates = append(duplicates, duplicate)
				
				if cm.config.Verbose {
					fmt.Printf("Found duplicate: %s (original: %s)\n", duplicate.Path, files[0].Path)
				}
			}
		}
	}
	
	fmt.Printf("Found %d duplicate files\n", len(duplicates))
	return duplicates
}

// identifyFilesToRemove identifies files that should be removed based on project cleanup rules
func (cm *CleanupManager) identifyFilesToRemove() []FileInfo {
	var filesToRemove []FileInfo
	
	fmt.Println("Identifying files to remove based on cleanup rules...")
	
	// Define patterns for files that should be removed
	removePatterns := []struct {
		pattern string
		reason  string
	}{
		{"*.exe", "Build artifact - executable file"},
		{"*.db", "Runtime database file"},
		{"*.db-shm", "SQLite shared memory file"},
		{"*.db-wal", "SQLite write-ahead log file"},
		{"*.log", "Log file"},
		{"*.tmp", "Temporary file"},
		{"**/build/**", "Build output directory"},
		{"**/dist/**", "Distribution directory"},
		{"**/releases/**/README.md", "Duplicate README in release directory"},
		{"config.yaml", "Runtime configuration file"},
		{"bridge.db*", "Runtime database files"},
	}

	// Check each file against removal patterns
	for _, files := range cm.fileMap {
		for _, file := range files {
			for _, pattern := range removePatterns {
				if cm.matchesPattern(file.Path, pattern.pattern) {
					fileToRemove := file
					fileToRemove.ShouldRemove = true
					fileToRemove.Reason = pattern.reason
					filesToRemove = append(filesToRemove, fileToRemove)
					
					if cm.config.Verbose {
						fmt.Printf("Marked for removal: %s (%s)\n", file.Path, pattern.reason)
					}
					break
				}
			}
		}
	}

	// Additional specific file removals based on project analysis
	specificFiles := []struct {
		path   string
		reason string
	}{
		{"gym-door-bridge.exe", "Root directory executable"},
		{"simple-repset-installer.ps1", "Duplicate installer script"},
		{"repset-bridge-installer-FIXED.ps1", "Duplicate installer script"},
		{"setup-heartbeat-task.ps1", "Duplicate script"},
		{"bridge-heartbeat-service.ps1", "Duplicate script"},
	}

	for _, specific := range specificFiles {
		if _, err := os.Stat(specific.path); err == nil {
			hash, err := cm.calculateFileHash(specific.path)
			if err == nil {
				fileToRemove := FileInfo{
					Path:         specific.path,
					Hash:         hash,
					ShouldRemove: true,
					Reason:       specific.reason,
				}
				filesToRemove = append(filesToRemove, fileToRemove)
				
				if cm.config.Verbose {
					fmt.Printf("Marked specific file for removal: %s (%s)\n", specific.path, specific.reason)
				}
			}
		}
	}
	
	fmt.Printf("Found %d files to remove based on cleanup rules\n", len(filesToRemove))
	return filesToRemove
}

// matchesPattern checks if a file path matches a given pattern
func (cm *CleanupManager) matchesPattern(path, pattern string) bool {
	// Simple pattern matching - can be enhanced with more sophisticated matching
	if strings.Contains(pattern, "**") {
		// Handle recursive directory patterns
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]
			return strings.Contains(path, prefix) && strings.HasSuffix(path, suffix)
		}
	}
	
	if strings.Contains(pattern, "*") {
		// Handle wildcard patterns
		if strings.HasPrefix(pattern, "*.") {
			ext := pattern[1:]
			return strings.HasSuffix(path, ext)
		}
	}
	
	// Exact match
	return path == pattern
}

// createBackup creates a backup of files before removal
func (cm *CleanupManager) createBackup(filesToBackup []FileInfo) error {
	if len(filesToBackup) == 0 {
		return nil
	}

	fmt.Printf("Creating backup in directory: %s\n", cm.config.BackupDir)
	
	// Create backup directory
	if err := os.MkdirAll(cm.config.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup manifest
	manifestPath := filepath.Join(cm.config.BackupDir, "backup-manifest.txt")
	manifest, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create backup manifest: %w", err)
	}
	defer manifest.Close()

	fmt.Fprintf(manifest, "Backup created: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(manifest, "Files backed up:\n\n")

	for _, file := range filesToBackup {
		// Create backup path maintaining directory structure
		backupPath := filepath.Join(cm.config.BackupDir, file.Path)
		backupDir := filepath.Dir(backupPath)
		
		// Create backup directory structure
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory %s: %w", backupDir, err)
		}

		// Copy file to backup location
		if err := cm.copyFile(file.Path, backupPath); err != nil {
			fmt.Printf("Warning: Failed to backup %s: %v\n", file.Path, err)
			continue
		}

		cm.backupMap[file.Path] = backupPath
		fmt.Fprintf(manifest, "%s -> %s (Reason: %s)\n", file.Path, backupPath, file.Reason)
		
		if cm.config.Verbose {
			fmt.Printf("Backed up: %s -> %s\n", file.Path, backupPath)
		}
	}

	fmt.Printf("Backup completed. %d files backed up.\n", len(cm.backupMap))
	return nil
}

// copyFile copies a file from src to dst
func (cm *CleanupManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	return os.Chmod(dst, sourceInfo.Mode())
}

// removeFiles removes the specified files
func (cm *CleanupManager) removeFiles(filesToRemove []FileInfo) error {
	if len(filesToRemove) == 0 {
		fmt.Println("No files to remove.")
		return nil
	}

	fmt.Printf("Removing %d files...\n", len(filesToRemove))
	
	if cm.config.DryRun {
		fmt.Println("DRY RUN - Files that would be removed:")
		for _, file := range filesToRemove {
			fmt.Printf("  - %s (%s)\n", file.Path, file.Reason)
		}
		return nil
	}

	removedCount := 0
	for _, file := range filesToRemove {
		if err := os.Remove(file.Path); err != nil {
			fmt.Printf("Warning: Failed to remove %s: %v\n", file.Path, err)
			continue
		}
		
		removedCount++
		if cm.config.Verbose {
			fmt.Printf("Removed: %s (%s)\n", file.Path, file.Reason)
		}
	}

	fmt.Printf("Successfully removed %d files.\n", removedCount)
	
	// Clean up empty directories
	cm.cleanupEmptyDirectories()
	
	return nil
}

// cleanupEmptyDirectories removes empty directories after file cleanup
func (cm *CleanupManager) cleanupEmptyDirectories() {
	fmt.Println("Cleaning up empty directories...")
	
	// Walk through directories and remove empty ones
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if info.IsDir() && path != "." {
			if cm.shouldSkip(path) {
				return filepath.SkipDir
			}
			
			// Check if directory is empty
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil
			}
			
			if len(entries) == 0 {
				if !cm.config.DryRun {
					if err := os.Remove(path); err == nil {
						if cm.config.Verbose {
							fmt.Printf("Removed empty directory: %s\n", path)
						}
					}
				} else {
					fmt.Printf("Would remove empty directory: %s\n", path)
				}
			}
		}
		
		return nil
	})
}