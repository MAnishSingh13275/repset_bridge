package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// installLinux performs Linux-specific installation
func (i *Installer) installLinux() error {
	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("installation requires root privileges. Please run with sudo")
	}

	// Check if already installed
	if installed, err := i.isInstalledLinux(); err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	} else if installed {
		return fmt.Errorf("service is already installed. Use 'gym-door-bridge uninstall' first")
	}

	// Create base installation
	if err := i.createBaseInstallation(); err != nil {
		return fmt.Errorf("failed to create base installation: %w", err)
	}

	// Install systemd service
	if err := i.installLinuxService(); err != nil {
		return fmt.Errorf("failed to install systemd service: %w", err)
	}

	// Enable and start the service
	if err := i.startLinux(); err != nil {
		i.logger.WithError(err).Warn("Failed to start service automatically")
		i.logger.Info("You can start the service manually using 'gym-door-bridge start'")
	}

	i.ShowInstallationSummary()
	return nil
}

// uninstallLinux performs Linux-specific uninstallation
func (i *Installer) uninstallLinux() error {
	// Check if running as root
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstallation requires root privileges. Please run with sudo")
	}

	// Stop and disable service
	if err := i.stopLinux(); err != nil {
		i.logger.WithError(err).Warn("Failed to stop service")
	}

	// Remove systemd service
	if err := i.uninstallLinuxService(); err != nil {
		return fmt.Errorf("failed to remove systemd service: %w", err)
	}

	// Remove installation directory
	if err := os.RemoveAll(i.installPath); err != nil {
		i.logger.WithError(err).Warn("Failed to remove installation directory")
		i.logger.Info("You may need to manually remove: " + i.installPath)
	}

	i.logger.Info("✅ Uninstallation completed successfully!")
	return nil
}

// isInstalledLinux checks if the systemd service is installed
func (i *Installer) isInstalledLinux() (bool, error) {
	cmd := exec.Command("systemctl", "list-unit-files", "gym-door-bridge.service")
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.Contains(string(output), "gym-door-bridge.service"), nil
}

// installLinuxService installs the systemd service
func (i *Installer) installLinuxService() error {
	i.logger.Info("Installing systemd service...")

	serviceContent := fmt.Sprintf(`[Unit]
Description=Gym Door Access Bridge
Documentation=https://github.com/repset/gym-door-bridge
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=5
User=root
WorkingDirectory=%s
ExecStart=%s --config %s
ExecReload=/bin/kill -HUP $MAINPID
KillMode=mixed
TimeoutStopSec=30
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`, i.installPath, i.GetExecutablePath(), i.configPath, i.installPath)

	servicePath := "/etc/systemd/system/gym-door-bridge.service"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}

	// Reload systemd daemon
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable service
	if err := exec.Command("systemctl", "enable", "gym-door-bridge.service").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	i.logger.Info("✅ Systemd service installed successfully")
	return nil
}

// uninstallLinuxService removes the systemd service
func (i *Installer) uninstallLinuxService() error {
	i.logger.Info("Removing systemd service...")

	// Disable service
	exec.Command("systemctl", "disable", "gym-door-bridge.service").Run()

	// Remove service file
	servicePath := "/etc/systemd/system/gym-door-bridge.service"
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd daemon
	exec.Command("systemctl", "daemon-reload").Run()

	i.logger.Info("✅ Systemd service removed successfully")
	return nil
}

// startLinux starts the systemd service
func (i *Installer) startLinux() error {
	i.logger.Info("Starting systemd service...")

	if err := exec.Command("systemctl", "start", "gym-door-bridge.service").Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait for service to start
	if err := i.WaitForService("active", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to start: %w", err)
	}

	i.logger.Info("✅ Systemd service started successfully")
	return nil
}

// stopLinux stops the systemd service
func (i *Installer) stopLinux() error {
	i.logger.Info("Stopping systemd service...")

	if err := exec.Command("systemctl", "stop", "gym-door-bridge.service").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait for service to stop
	if err := i.WaitForService("inactive", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to stop: %w", err)
	}

	i.logger.Info("✅ Systemd service stopped successfully")
	return nil
}

// statusLinux returns the systemd service status
func (i *Installer) statusLinux() (string, error) {
	cmd := exec.Command("systemctl", "is-active", "gym-door-bridge.service")
	output, err := cmd.Output()
	if err != nil {
		return "inactive", nil
	}
	
	status := strings.TrimSpace(string(output))
	switch status {
	case "active":
		return "Running", nil
	case "inactive":
		return "Stopped", nil
	case "failed":
		return "Failed", nil
	default:
		return status, nil
	}
}