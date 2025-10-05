package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// installMacOS performs macOS-specific installation
func (i *Installer) installMacOS() error {
	// Check if running as root or with sudo
	if os.Geteuid() != 0 {
		return fmt.Errorf("installation requires administrator privileges. Please run with sudo")
	}

	// Check if already installed
	if installed, err := i.isInstalledMacOS(); err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	} else if installed {
		return fmt.Errorf("service is already installed. Use 'gym-door-bridge uninstall' first")
	}

	// Create base installation
	if err := i.createBaseInstallation(); err != nil {
		return fmt.Errorf("failed to create base installation: %w", err)
	}

	// Install launchd service
	if err := i.installMacOSService(); err != nil {
		return fmt.Errorf("failed to install launchd service: %w", err)
	}

	// Load and start the service
	if err := i.startMacOS(); err != nil {
		i.logger.WithError(err).Warn("Failed to start service automatically")
		i.logger.Info("You can start the service manually using 'gym-door-bridge start'")
	}

	i.ShowInstallationSummary()
	return nil
}

// uninstallMacOS performs macOS-specific uninstallation
func (i *Installer) uninstallMacOS() error {
	// Check if running as root or with sudo
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstallation requires administrator privileges. Please run with sudo")
	}

	// Stop and unload service
	if err := i.stopMacOS(); err != nil {
		i.logger.WithError(err).Warn("Failed to stop service")
	}

	// Remove launchd service
	if err := i.uninstallMacOSService(); err != nil {
		return fmt.Errorf("failed to remove launchd service: %w", err)
	}

	// Remove installation directory
	if err := os.RemoveAll(i.installPath); err != nil {
		i.logger.WithError(err).Warn("Failed to remove installation directory")
		i.logger.Info("You may need to manually remove: " + i.installPath)
	}

	i.logger.Info("✅ Uninstallation completed successfully!")
	return nil
}

// isInstalledMacOS checks if the launchd service is installed
func (i *Installer) isInstalledMacOS() (bool, error) {
	plistPath := "/Library/LaunchDaemons/com.repset.gym-door-bridge.plist"
	_, err := os.Stat(plistPath)
	return !os.IsNotExist(err), nil
}

// installMacOSService installs the launchd service
func (i *Installer) installMacOSService() error {
	i.logger.Info("Installing launchd service...")

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.repset.gym-door-bridge</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>--config</string>
		<string>%s</string>
	</array>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s/logs/bridge.log</string>
	<key>StandardErrorPath</key>
	<string>%s/logs/bridge-error.log</string>
</dict>
</plist>
`, i.GetExecutablePath(), i.configPath, i.installPath, i.installPath, i.installPath)

	plistPath := "/Library/LaunchDaemons/com.repset.gym-door-bridge.plist"
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}

	// Set proper ownership and permissions
	if err := os.Chown(plistPath, 0, 0); err != nil {
		return fmt.Errorf("failed to set plist ownership: %w", err)
	}

	i.logger.Info("✅ Launchd service installed successfully")
	return nil
}

// uninstallMacOSService removes the launchd service
func (i *Installer) uninstallMacOSService() error {
	i.logger.Info("Removing launchd service...")

	plistPath := "/Library/LaunchDaemons/com.repset.gym-door-bridge.plist"
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	i.logger.Info("✅ Launchd service removed successfully")
	return nil
}

// startMacOS starts the launchd service
func (i *Installer) startMacOS() error {
	i.logger.Info("Starting launchd service...")

	if err := exec.Command("launchctl", "load", "/Library/LaunchDaemons/com.repset.gym-door-bridge.plist").Run(); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	// Wait for service to start
	if err := i.WaitForService("Running", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to start: %w", err)
	}

	i.logger.Info("✅ Launchd service started successfully")
	return nil
}

// stopMacOS stops the launchd service
func (i *Installer) stopMacOS() error {
	i.logger.Info("Stopping launchd service...")

	if err := exec.Command("launchctl", "unload", "/Library/LaunchDaemons/com.repset.gym-door-bridge.plist").Run(); err != nil {
		return fmt.Errorf("failed to unload service: %w", err)
	}

	// Wait for service to stop
	if err := i.WaitForService("Stopped", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to stop: %w", err)
	}

	i.logger.Info("✅ Launchd service stopped successfully")
	return nil
}

// statusMacOS returns the launchd service status
func (i *Installer) statusMacOS() (string, error) {
	cmd := exec.Command("launchctl", "list", "com.repset.gym-door-bridge")
	output, err := cmd.Output()
	if err != nil {
		return "Stopped", nil
	}
	
	if strings.Contains(string(output), "com.repset.gym-door-bridge") {
		return "Running", nil
	}
	return "Stopped", nil
}
)

const (
	LaunchdServiceName = "com.gymbridge.door-bridge"
	LaunchdPlistPath   = "/Library/LaunchDaemons/com.gymbridge.door-bridge.plist"
	MacOSAppPath       = "/Applications/GymDoorBridge"
)

// installMacOS performs macOS-specific installation
func (i *Installer) installMacOS() error {
	// Check if running as root/administrator
	if !isSudoOrRoot() {
		return fmt.Errorf("installation requires administrator privileges. Please run with sudo")
	}

	// Check if already installed
	if installed, err := i.isInstalledMacOS(); err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	} else if installed {
		return fmt.Errorf("service is already installed. Use 'gym-door-bridge uninstall' first")
	}

	// Create base installation
	if err := i.createBaseInstallation(); err != nil {
		return fmt.Errorf("failed to create base installation: %w", err)
	}

	// Install launchd daemon
	if err := i.installLaunchdDaemon(); err != nil {
		return fmt.Errorf("failed to install launchd daemon: %w", err)
	}

	// Load and start the daemon
	if err := i.startMacOS(); err != nil {
		i.logger.WithError(err).Warn("Failed to start daemon automatically")
		i.logger.Info("You can start the daemon manually using 'gym-door-bridge start'")
	}

	i.ShowInstallationSummary()
	return nil
}

// uninstallMacOS performs macOS-specific uninstallation
func (i *Installer) uninstallMacOS() error {
	// Check if running as root/administrator
	if !isSudoOrRoot() {
		return fmt.Errorf("uninstallation requires administrator privileges. Please run with sudo")
	}

	// Stop daemon if running
	if err := i.stopMacOS(); err != nil {
		i.logger.WithError(err).Warn("Failed to stop daemon")
	}

	// Unload and remove launchd daemon
	if err := i.uninstallLaunchdDaemon(); err != nil {
		return fmt.Errorf("failed to uninstall launchd daemon: %w", err)
	}

	// Remove installation directory
	if err := os.RemoveAll(i.installPath); err != nil {
		i.logger.WithError(err).Warn("Failed to remove installation directory")
		i.logger.Info("You may need to manually remove: " + i.installPath)
	}

	i.logger.Info("✅ Uninstallation completed successfully!")
	return nil
}

// isInstalledMacOS checks if the launchd daemon is installed
func (i *Installer) isInstalledMacOS() (bool, error) {
	_, err := os.Stat(LaunchdPlistPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check launchd plist: %w", err)
	}
	return true, nil
}

// installLaunchdDaemon creates and installs the launchd daemon plist
func (i *Installer) installLaunchdDaemon() error {
	i.logger.Info("Installing macOS launchd daemon...")

	// Create launchd plist content
	plistContent := i.generateLaunchdPlist()

	// Write plist file
	if err := os.WriteFile(LaunchdPlistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write launchd plist: %w", err)
	}

	// Set proper ownership and permissions
	if err := os.Chown(LaunchdPlistPath, 0, 0); err != nil {
		i.logger.WithError(err).Warn("Failed to set plist ownership")
	}

	i.logger.Info("✅ macOS launchd daemon installed successfully with auto-start and failure recovery")
	return nil
}

// uninstallLaunchdDaemon removes the launchd daemon
func (i *Installer) uninstallLaunchdDaemon() error {
	i.logger.Info("Removing macOS launchd daemon...")

	// Unload the daemon first
	cmd := exec.Command("launchctl", "unload", LaunchdPlistPath)
	if err := cmd.Run(); err != nil {
		i.logger.WithError(err).Warn("Failed to unload daemon (it may not be loaded)")
	}

	// Remove the plist file
	if err := os.Remove(LaunchdPlistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove launchd plist: %w", err)
	}

	i.logger.Info("✅ macOS launchd daemon removed successfully")
	return nil
}

// generateLaunchdPlist generates the launchd plist content
func (i *Installer) generateLaunchdPlist() string {
	execPath := i.GetExecutablePath()
	logFile := filepath.Join(i.installPath, "logs", "bridge.log")
	errorLogFile := filepath.Join(i.installPath, "logs", "bridge-error.log")

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>--config</string>
        <string>%s</string>
    </array>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
        <key>Crashed</key>
        <true/>
    </dict>
    
    <key>ThrottleInterval</key>
    <integer>60</integer>
    
    <key>StandardOutPath</key>
    <string>%s</string>
    
    <key>StandardErrorPath</key>
    <string>%s</string>
    
    <key>WorkingDirectory</key>
    <string>%s</string>
    
    <key>UserName</key>
    <string>root</string>
    
    <key>GroupName</key>
    <string>wheel</string>
    
    <key>ProcessType</key>
    <string>Background</string>
    
    <key>AbandonProcessGroup</key>
    <true/>
    
    <key>ExitTimeOut</key>
    <integer>30</integer>
    
    <key>HardResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>1024</integer>
    </dict>
    
    <key>SoftResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>1024</integer>
    </dict>
    
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>`,
		LaunchdServiceName,
		execPath,
		i.configPath,
		logFile,
		errorLogFile,
		i.installPath)
}

// startMacOS starts the macOS launchd daemon
func (i *Installer) startMacOS() error {
	i.logger.Info("Starting macOS launchd daemon...")

	// Load the daemon
	cmd := exec.Command("launchctl", "load", LaunchdPlistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load daemon: %w", err)
	}

	// Wait a moment for the daemon to start
	time.Sleep(2 * time.Second)

	// Verify it's running
	if err := i.WaitForService("Running", 30*time.Second); err != nil {
		return fmt.Errorf("daemon failed to start: %w", err)
	}

	i.logger.Info("✅ macOS launchd daemon started successfully")
	return nil
}

// stopMacOS stops the macOS launchd daemon
func (i *Installer) stopMacOS() error {
	i.logger.Info("Stopping macOS launchd daemon...")

	// Unload the daemon
	cmd := exec.Command("launchctl", "unload", LaunchdPlistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unload daemon: %w", err)
	}

	// Wait for daemon to stop
	if err := i.WaitForService("Stopped", 30*time.Second); err != nil {
		return fmt.Errorf("daemon failed to stop: %w", err)
	}

	i.logger.Info("✅ macOS launchd daemon stopped successfully")
	return nil
}

// statusMacOS returns the macOS launchd daemon status
func (i *Installer) statusMacOS() (string, error) {
	// Check if plist exists
	if _, err := os.Stat(LaunchdPlistPath); os.IsNotExist(err) {
		return "Not Installed", nil
	}

	// Check if daemon is loaded
	cmd := exec.Command("launchctl", "list", LaunchdServiceName)
	output, err := cmd.Output()
	if err != nil {
		// Not loaded
		return "Stopped", nil
	}

	// Parse launchctl output to determine status
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	
	for _, line := range lines {
		if strings.Contains(line, LaunchdServiceName) {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// First field is PID, second is last exit status, third is label
				pid := fields[0]
				if pid != "-" && pid != "0" {
					return "Running", nil
				}
			}
		}
	}

	return "Stopped", nil
}

// isSudoOrRoot checks if running with sudo or as root
func isSudoOrRoot() bool {
	// Check if running as root
	if os.Geteuid() == 0 {
		return true
	}

	// Check if sudo is available and we can use it
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	return err == nil
}

// RestartService restarts the macOS launchd daemon
func (i *Installer) RestartService() error {
	i.logger.Info("Restarting macOS launchd daemon...")

	if err := i.stopMacOS(); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	// Small delay to ensure daemon is fully stopped
	time.Sleep(2 * time.Second)

	if err := i.startMacOS(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	i.logger.Info("✅ macOS launchd daemon restarted successfully")
	return nil
}

// GetServiceLogs returns recent service logs
func (i *Installer) GetServiceLogs() ([]string, error) {
	logFile := filepath.Join(i.installPath, "logs", "bridge.log")
	
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []string{"No log file found"}, nil
	}

	// Read last 50 lines from log file
	cmd := exec.Command("tail", "-n", "50", logFile)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	
	// Filter out empty lines
	var filteredLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	return filteredLines, nil
}

// SetupLogRotation configures log rotation for macOS
func (i *Installer) SetupLogRotation() error {
	i.logger.Info("Setting up log rotation...")

	// Create a simple log rotation script
	rotateScript := fmt.Sprintf(`#!/bin/bash
# Gym Door Bridge Log Rotation Script

LOG_FILE="%s/logs/bridge.log"
MAX_SIZE=10485760  # 10MB in bytes

if [ -f "$LOG_FILE" ]; then
    SIZE=$(stat -f%%z "$LOG_FILE" 2>/dev/null || echo 0)
    if [ $SIZE -gt $MAX_SIZE ]; then
        mv "$LOG_FILE" "$LOG_FILE.old"
        touch "$LOG_FILE"
        # Restart the service to use the new log file
        launchctl unload "%s"
        launchctl load "%s"
    fi
fi
`, i.installPath, LaunchdPlistPath, LaunchdPlistPath)

	// Write rotation script
	scriptPath := filepath.Join(i.installPath, "rotate-logs.sh")
	if err := os.WriteFile(scriptPath, []byte(rotateScript), 0755); err != nil {
		return fmt.Errorf("failed to create log rotation script: %w", err)
	}

	// Create a simple cron entry suggestion
	i.logger.Info("Log rotation script created at: " + scriptPath)
	i.logger.Info("To enable automatic log rotation, add this to root's crontab:")
	i.logger.Info(fmt.Sprintf("0 * * * * %s", scriptPath))

	return nil
}

// copyFile copies a file from src to dst (macOS-specific)
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file contents
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Ensure all data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}