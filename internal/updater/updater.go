package updater

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// Manifest represents the update manifest structure
type Manifest struct {
	Version     string            `json:"version"`
	ReleaseDate time.Time         `json:"release_date"`
	Binaries    map[string]Binary `json:"binaries"`
	Rollout     RolloutConfig     `json:"rollout"`
	MinVersion  string            `json:"min_version,omitempty"`
}

// Binary represents a platform-specific binary
type Binary struct {
	URL       string `json:"url"`
	Signature string `json:"signature"` // Ed25519 signature in hex
	Size      int64  `json:"size"`
	Checksum  string `json:"checksum"` // SHA256 checksum in hex
}

// RolloutConfig controls staged rollout
type RolloutConfig struct {
	Percentage int      `json:"percentage"` // 0-100, percentage of devices to update
	Regions    []string `json:"regions,omitempty"`
	DeviceIDs  []string `json:"device_ids,omitempty"`
}

// UpdaterConfig holds updater configuration
type UpdaterConfig struct {
	ManifestURL   string        `json:"manifest_url"`
	PublicKey     string        `json:"public_key"`     // Ed25519 public key in hex
	CheckInterval time.Duration `json:"check_interval"` // How often to check for updates
	DeviceID      string        `json:"device_id"`
	CurrentVersion string       `json:"current_version"`
	UpdateDir     string        `json:"update_dir"`     // Directory for downloaded updates
	BackupDir     string        `json:"backup_dir"`     // Directory for backup binaries
}

// Updater manages automatic updates
type Updater struct {
	config *UpdaterConfig
	logger *logrus.Logger
	client *http.Client
}

// NewUpdater creates a new updater instance
func NewUpdater(config *UpdaterConfig, logger *logrus.Logger) *Updater {
	return &Updater{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins the update checking process
func (u *Updater) Start(ctx context.Context) error {
	u.logger.Info("Starting update checker")
	
	// Create necessary directories
	if err := u.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}
	
	// Check for updates immediately on startup
	if err := u.checkForUpdates(ctx); err != nil {
		u.logger.WithError(err).Warn("Initial update check failed")
	}
	
	// Start periodic update checks
	ticker := time.NewTicker(u.config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			u.logger.Info("Update checker stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := u.checkForUpdates(ctx); err != nil {
				u.logger.WithError(err).Warn("Update check failed")
			}
		}
	}
}

// checkForUpdates checks for and applies available updates
func (u *Updater) checkForUpdates(ctx context.Context) error {
	u.logger.Debug("Checking for updates")
	
	// Download manifest
	manifest, err := u.downloadManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	
	// Check if update is needed
	if !u.isUpdateNeeded(manifest) {
		u.logger.Debug("No update needed")
		return nil
	}
	
	// Check rollout eligibility
	if !u.isEligibleForRollout(manifest.Rollout) {
		u.logger.Info("Device not eligible for rollout yet")
		return nil
	}
	
	u.logger.WithField("version", manifest.Version).Info("Update available")
	
	// Download and verify update
	binaryPath, err := u.downloadAndVerifyBinary(ctx, manifest)
	if err != nil {
		return fmt.Errorf("failed to download and verify binary: %w", err)
	}
	
	// Apply update
	if err := u.applyUpdate(binaryPath); err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}
	
	u.logger.WithField("version", manifest.Version).Info("Update applied successfully")
	return nil
}

// downloadManifest downloads and parses the update manifest
func (u *Updater) downloadManifest(ctx context.Context) (*Manifest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u.config.ManifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest download failed with status: %d", resp.StatusCode)
	}
	
	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	
	return &manifest, nil
}

// isUpdateNeeded checks if an update is needed
func (u *Updater) isUpdateNeeded(manifest *Manifest) bool {
	// Simple version comparison - in production, use semantic versioning
	return manifest.Version != u.config.CurrentVersion
}

// isEligibleForRollout checks if this device is eligible for the rollout
func (u *Updater) isEligibleForRollout(rollout RolloutConfig) bool {
	// Check if device is explicitly included
	for _, deviceID := range rollout.DeviceIDs {
		if deviceID == u.config.DeviceID {
			return true
		}
	}
	
	// Check percentage rollout using device ID hash
	if rollout.Percentage > 0 && rollout.Percentage < 100 {
		hash := u.hashDeviceID(u.config.DeviceID)
		threshold := uint32(rollout.Percentage * 0xFFFFFFFF / 100)
		return hash <= threshold
	}
	
	// 100% rollout or no restrictions
	return rollout.Percentage == 100
}

// hashDeviceID creates a consistent hash of the device ID for rollout decisions
func (u *Updater) hashDeviceID(deviceID string) uint32 {
	// Simple hash function for demonstration - use a proper hash in production
	var hash uint32
	for _, b := range []byte(deviceID) {
		hash = hash*31 + uint32(b)
	}
	return hash
}

// downloadAndVerifyBinary downloads and verifies the binary for the current platform
func (u *Updater) downloadAndVerifyBinary(ctx context.Context, manifest *Manifest) (string, error) {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	binary, exists := manifest.Binaries[platform]
	if !exists {
		return "", fmt.Errorf("no binary available for platform: %s", platform)
	}
	
	// Download binary
	binaryPath := filepath.Join(u.config.UpdateDir, fmt.Sprintf("bridge_%s", manifest.Version))
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	
	if err := u.downloadFile(ctx, binary.URL, binaryPath); err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}
	
	// Verify signature
	if err := u.verifySignature(binaryPath, binary.Signature); err != nil {
		os.Remove(binaryPath) // Clean up on verification failure
		return "", fmt.Errorf("signature verification failed: %w", err)
	}
	
	u.logger.Info("Binary downloaded and verified successfully")
	return binaryPath, nil
}

// downloadFile downloads a file from URL to the specified path
func (u *Updater) downloadFile(ctx context.Context, url, filepath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}
	
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// verifySignature verifies the Ed25519 signature of a file
func (u *Updater) verifySignature(filePath, signatureHex string) error {
	// Parse public key
	publicKeyBytes, err := hex.DecodeString(u.config.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}
	
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: %d", len(publicKeyBytes))
	}
	
	publicKey := ed25519.PublicKey(publicKeyBytes)
	
	// Parse signature
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	
	// Read file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	// Verify signature
	if !ed25519.Verify(publicKey, fileContent, signature) {
		return fmt.Errorf("signature verification failed")
	}
	
	return nil
}

// applyUpdate applies the downloaded update
func (u *Updater) applyUpdate(newBinaryPath string) error {
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	
	// Create backup
	backupPath := filepath.Join(u.config.BackupDir, fmt.Sprintf("bridge_backup_%d", time.Now().Unix()))
	if runtime.GOOS == "windows" {
		backupPath += ".exe"
	}
	
	if err := u.copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	
	u.logger.WithField("backup_path", backupPath).Info("Created backup of current binary")
	
	// Replace current binary
	if err := u.replaceExecutable(currentExe, newBinaryPath); err != nil {
		// Try to restore backup on failure
		if restoreErr := u.copyFile(backupPath, currentExe); restoreErr != nil {
			u.logger.WithError(restoreErr).Error("Failed to restore backup after update failure")
		}
		return fmt.Errorf("failed to replace executable: %w", err)
	}
	
	u.logger.Info("Binary replaced successfully")
	
	// Schedule restart
	return u.scheduleRestart()
}

// copyFile copies a file from src to dst
func (u *Updater) copyFile(src, dst string) error {
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

// replaceExecutable replaces the current executable with the new one
func (u *Updater) replaceExecutable(currentPath, newPath string) error {
	// On Windows, we might need to handle file locking differently
	if runtime.GOOS == "windows" {
		return u.replaceExecutableWindows(currentPath, newPath)
	}
	
	// On Unix-like systems, we can usually replace the file directly
	return os.Rename(newPath, currentPath)
}

// replaceExecutableWindows handles executable replacement on Windows
func (u *Updater) replaceExecutableWindows(currentPath, newPath string) error {
	// On Windows, we might need to rename the current executable first
	tempPath := currentPath + ".old"
	
	// Remove any existing .old file
	os.Remove(tempPath)
	
	// Rename current executable
	if err := os.Rename(currentPath, tempPath); err != nil {
		return fmt.Errorf("failed to rename current executable: %w", err)
	}
	
	// Move new executable to current location
	if err := os.Rename(newPath, currentPath); err != nil {
		// Try to restore original
		os.Rename(tempPath, currentPath)
		return fmt.Errorf("failed to move new executable: %w", err)
	}
	
	// Schedule cleanup of old file after restart
	return nil
}

// scheduleRestart schedules a restart of the service
func (u *Updater) scheduleRestart() error {
	u.logger.Info("Scheduling service restart for update")
	
	// Create a restart signal file that the service manager can detect
	restartFile := filepath.Join(u.config.UpdateDir, "restart_required")
	if err := os.WriteFile(restartFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("failed to create restart signal file: %w", err)
	}
	
	return nil
}

// createDirectories creates necessary directories for updates
func (u *Updater) createDirectories() error {
	dirs := []string{u.config.UpdateDir, u.config.BackupDir}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// GenerateKeyPair generates a new Ed25519 key pair for testing
func GenerateKeyPair() (publicKey, privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}
	
	return hex.EncodeToString(pub), hex.EncodeToString(priv), nil
}

// SignFile signs a file with an Ed25519 private key
func SignFile(filePath, privateKeyHex string) (string, error) {
	// Parse private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}
	
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key size: %d", len(privateKeyBytes))
	}
	
	privateKey := ed25519.PrivateKey(privateKeyBytes)
	
	// Read file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	// Sign file
	signature := ed25519.Sign(privateKey, fileContent)
	
	return hex.EncodeToString(signature), nil
}