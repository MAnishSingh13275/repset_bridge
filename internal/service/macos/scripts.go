package macos

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// InstallScript generates the macOS installation script
const InstallScriptTemplate = `#!/bin/bash
set -e

# Gym Door Bridge macOS Installation Script
# This script downloads, installs, and configures the Gym Door Bridge daemon

BRIDGE_VERSION="${BRIDGE_VERSION:-latest}"
BRIDGE_URL="${BRIDGE_URL:-https://releases.gymdoorbridge.com/macos}"
PAIR_CODE="${PAIR_CODE:-}"
CONFIG_PATH="/usr/local/etc/gymdoorbridge/config.yaml"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="com.gymdoorbridge.agent"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check system requirements
check_requirements() {
    log_info "Checking system requirements..."
    
    # Check macOS version
    if [[ $(uname) != "Darwin" ]]; then
        log_error "This script is for macOS only"
        exit 1
    fi
    
    # Check for required tools
    for tool in curl unzip launchctl; do
        if ! command -v $tool &> /dev/null; then
            log_error "Required tool not found: $tool"
            exit 1
        fi
    done
    
    log_info "System requirements satisfied"
}

# Download and verify binary
download_binary() {
    log_info "Downloading Gym Door Bridge binary..."
    
    local temp_dir=$(mktemp -d)
    local binary_url="${BRIDGE_URL}/gym-door-bridge-${BRIDGE_VERSION}-darwin-amd64.tar.gz"
    local checksum_url="${BRIDGE_URL}/gym-door-bridge-${BRIDGE_VERSION}-checksums.txt"
    
    # Download binary archive
    if ! curl -L -o "${temp_dir}/gym-door-bridge.tar.gz" "$binary_url"; then
        log_error "Failed to download binary from $binary_url"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Download checksums
    if ! curl -L -o "${temp_dir}/checksums.txt" "$checksum_url"; then
        log_warn "Failed to download checksums, skipping verification"
    else
        # Verify checksum
        cd "$temp_dir"
        if ! shasum -c checksums.txt --ignore-missing; then
            log_error "Checksum verification failed"
            rm -rf "$temp_dir"
            exit 1
        fi
        log_info "Checksum verification passed"
    fi
    
    # Extract binary
    if ! tar -xzf "${temp_dir}/gym-door-bridge.tar.gz" -C "$temp_dir"; then
        log_error "Failed to extract binary archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Install binary
    if ! cp "${temp_dir}/gym-door-bridge" "$INSTALL_DIR/"; then
        log_error "Failed to install binary to $INSTALL_DIR"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Set permissions
    chmod +x "$INSTALL_DIR/gym-door-bridge"
    
    # Clean up
    rm -rf "$temp_dir"
    
    log_info "Binary installed successfully to $INSTALL_DIR/gym-door-bridge"
}

# Install daemon
install_daemon() {
    log_info "Installing daemon..."
    
    # Stop existing daemon if running
    if launchctl list | grep -q "$SERVICE_NAME"; then
        log_info "Stopping existing daemon..."
        launchctl stop "$SERVICE_NAME" 2>/dev/null || true
        launchctl unload "/Library/LaunchDaemons/${SERVICE_NAME}.plist" 2>/dev/null || true
    fi
    
    # Install daemon using the binary
    if ! "$INSTALL_DIR/gym-door-bridge" daemon install; then
        log_error "Failed to install daemon"
        exit 1
    fi
    
    log_info "Daemon installed successfully"
}

# Configure bridge
configure_bridge() {
    if [[ -n "$PAIR_CODE" ]]; then
        log_info "Configuring bridge with pair code..."
        
        # Create configuration with pair code
        cat > "$CONFIG_PATH" << EOF
# Gym Door Bridge Configuration
# Generated during installation

# Device pairing
pairing:
  pair_code: "$PAIR_CODE"
  auto_pair: true

# Logging configuration
log:
  level: info
  format: json

# Hardware adapter configuration
adapters:
  simulator:
    enabled: true

# Performance tier settings (auto-detected)
performance:
  tier: auto

# Network configuration
network:
  timeout: 30s
  retry_attempts: 3

# Queue configuration
queue:
  max_size: 10000
  batch_size: 100

# Door control configuration
door:
  unlock_duration: 3s
  auto_relock: true
EOF
        
        log_info "Configuration created with pair code"
    else
        log_warn "No pair code provided. Manual configuration required."
        log_info "Edit $CONFIG_PATH to configure the bridge"
    fi
}

# Start daemon
start_daemon() {
    log_info "Starting daemon..."
    
    if ! "$INSTALL_DIR/gym-door-bridge" daemon start; then
        log_error "Failed to start daemon"
        exit 1
    fi
    
    # Wait a moment and check status
    sleep 2
    if "$INSTALL_DIR/gym-door-bridge" daemon status | grep -q "Running"; then
        log_info "Daemon started successfully"
    else
        log_warn "Daemon may not have started properly. Check logs for details."
    fi
}

# Show installation summary
show_summary() {
    log_info "Installation completed successfully!"
    echo
    echo "Gym Door Bridge has been installed and configured:"
    echo "  Binary: $INSTALL_DIR/gym-door-bridge"
    echo "  Configuration: $CONFIG_PATH"
    echo "  Service: $SERVICE_NAME"
    echo
    echo "Useful commands:"
    echo "  Check status: $INSTALL_DIR/gym-door-bridge daemon status"
    echo "  View logs: tail -f /usr/local/var/log/gymdoorbridge/bridge.log"
    echo "  Stop daemon: sudo $INSTALL_DIR/gym-door-bridge daemon stop"
    echo "  Start daemon: sudo $INSTALL_DIR/gym-door-bridge daemon start"
    echo "  Uninstall: sudo $INSTALL_DIR/gym-door-bridge daemon uninstall"
    echo
    if [[ -z "$PAIR_CODE" ]]; then
        echo "Next steps:"
        echo "  1. Edit $CONFIG_PATH with your configuration"
        echo "  2. Add your pair code to the configuration"
        echo "  3. Restart the daemon: sudo $INSTALL_DIR/gym-door-bridge daemon restart"
    fi
}

# Main installation function
main() {
    log_info "Starting Gym Door Bridge installation..."
    
    check_root
    check_requirements
    download_binary
    install_daemon
    configure_bridge
    start_daemon
    show_summary
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --pair-code)
            PAIR_CODE="$2"
            shift 2
            ;;
        --version)
            BRIDGE_VERSION="$2"
            shift 2
            ;;
        --url)
            BRIDGE_URL="$2"
            shift 2
            ;;
        --help)
            echo "Gym Door Bridge macOS Installation Script"
            echo
            echo "Usage: $0 [options]"
            echo
            echo "Options:"
            echo "  --pair-code CODE    Pair code for automatic device registration"
            echo "  --version VERSION   Specific version to install (default: latest)"
            echo "  --url URL          Custom download URL base"
            echo "  --help             Show this help message"
            echo
            echo "Examples:"
            echo "  sudo $0 --pair-code ABC123"
            echo "  sudo $0 --version 1.2.3 --pair-code ABC123"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run main installation
main
`

// UninstallScript generates the macOS uninstallation script
const UninstallScriptTemplate = `#!/bin/bash
set -e

# Gym Door Bridge macOS Uninstallation Script
# This script removes the Gym Door Bridge daemon and optionally cleans up data

INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="com.gymdoorbridge.agent"
REMOVE_DATA=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Uninstall daemon
uninstall_daemon() {
    log_info "Uninstalling daemon..."
    
    if [[ -f "$INSTALL_DIR/gym-door-bridge" ]]; then
        # Use the binary to uninstall itself
        if ! "$INSTALL_DIR/gym-door-bridge" daemon uninstall; then
            log_warn "Failed to uninstall daemon using binary, attempting manual cleanup..."
            
            # Manual cleanup
            launchctl stop "$SERVICE_NAME" 2>/dev/null || true
            launchctl unload "/Library/LaunchDaemons/${SERVICE_NAME}.plist" 2>/dev/null || true
            rm -f "/Library/LaunchDaemons/${SERVICE_NAME}.plist"
        fi
    else
        log_warn "Binary not found, attempting manual daemon cleanup..."
        
        # Manual cleanup
        launchctl stop "$SERVICE_NAME" 2>/dev/null || true
        launchctl unload "/Library/LaunchDaemons/${SERVICE_NAME}.plist" 2>/dev/null || true
        rm -f "/Library/LaunchDaemons/${SERVICE_NAME}.plist"
    fi
    
    log_info "Daemon uninstalled"
}

# Remove binary
remove_binary() {
    log_info "Removing binary..."
    
    if [[ -f "$INSTALL_DIR/gym-door-bridge" ]]; then
        rm -f "$INSTALL_DIR/gym-door-bridge"
        log_info "Binary removed"
    else
        log_warn "Binary not found at $INSTALL_DIR/gym-door-bridge"
    fi
}

# Remove data and configuration
remove_data() {
    if [[ "$REMOVE_DATA" == "true" ]]; then
        log_info "Removing configuration and data..."
        
        # Remove configuration
        if [[ -d "/usr/local/etc/gymdoorbridge" ]]; then
            rm -rf "/usr/local/etc/gymdoorbridge"
            log_info "Configuration removed"
        fi
        
        # Remove data directory
        if [[ -d "/usr/local/var/lib/gymdoorbridge" ]]; then
            rm -rf "/usr/local/var/lib/gymdoorbridge"
            log_info "Data directory removed"
        fi
        
        # Remove logs
        if [[ -d "/usr/local/var/log/gymdoorbridge" ]]; then
            rm -rf "/usr/local/var/log/gymdoorbridge"
            log_info "Log directory removed"
        fi
    else
        log_info "Configuration and data preserved"
        log_info "To remove manually:"
        log_info "  sudo rm -rf /usr/local/etc/gymdoorbridge"
        log_info "  sudo rm -rf /usr/local/var/lib/gymdoorbridge"
        log_info "  sudo rm -rf /usr/local/var/log/gymdoorbridge"
    fi
}

# Show uninstallation summary
show_summary() {
    log_info "Uninstallation completed successfully!"
    echo
    echo "Gym Door Bridge has been removed from your system."
    
    if [[ "$REMOVE_DATA" != "true" ]]; then
        echo
        echo "Configuration and data files have been preserved."
        echo "Remove them manually if no longer needed:"
        echo "  sudo rm -rf /usr/local/etc/gymdoorbridge"
        echo "  sudo rm -rf /usr/local/var/lib/gymdoorbridge"
        echo "  sudo rm -rf /usr/local/var/log/gymdoorbridge"
    fi
}

# Main uninstallation function
main() {
    log_info "Starting Gym Door Bridge uninstallation..."
    
    check_root
    uninstall_daemon
    remove_binary
    remove_data
    show_summary
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --remove-data)
            REMOVE_DATA=true
            shift
            ;;
        --help)
            echo "Gym Door Bridge macOS Uninstallation Script"
            echo
            echo "Usage: $0 [options]"
            echo
            echo "Options:"
            echo "  --remove-data    Also remove configuration and data files"
            echo "  --help          Show this help message"
            echo
            echo "Examples:"
            echo "  sudo $0"
            echo "  sudo $0 --remove-data"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Confirm uninstallation
echo "This will uninstall Gym Door Bridge from your system."
if [[ "$REMOVE_DATA" == "true" ]]; then
    echo "Configuration and data files will also be removed."
fi
echo
read -p "Are you sure you want to continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "Uninstallation cancelled"
    exit 0
fi

# Run main uninstallation
main
`

// ScriptGenerator handles generation of installation and uninstallation scripts
type ScriptGenerator struct{}

// NewScriptGenerator creates a new script generator
func NewScriptGenerator() *ScriptGenerator {
	return &ScriptGenerator{}
}

// GenerateInstallScript generates the macOS installation script
func (sg *ScriptGenerator) GenerateInstallScript(outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Write install script
	if err := os.WriteFile(outputPath, []byte(InstallScriptTemplate), 0755); err != nil {
		return fmt.Errorf("failed to write install script: %w", err)
	}
	
	fmt.Printf("Install script generated: %s\n", outputPath)
	return nil
}

// GenerateUninstallScript generates the macOS uninstallation script
func (sg *ScriptGenerator) GenerateUninstallScript(outputPath string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Write uninstall script
	if err := os.WriteFile(outputPath, []byte(UninstallScriptTemplate), 0755); err != nil {
		return fmt.Errorf("failed to write uninstall script: %w", err)
	}
	
	fmt.Printf("Uninstall script generated: %s\n", outputPath)
	return nil
}

// GenerateScripts generates both installation and uninstallation scripts
func (sg *ScriptGenerator) GenerateScripts(outputDir string) error {
	installPath := filepath.Join(outputDir, "install.sh")
	uninstallPath := filepath.Join(outputDir, "uninstall.sh")
	
	if err := sg.GenerateInstallScript(installPath); err != nil {
		return fmt.Errorf("failed to generate install script: %w", err)
	}
	
	if err := sg.GenerateUninstallScript(uninstallPath); err != nil {
		return fmt.Errorf("failed to generate uninstall script: %w", err)
	}
	
	return nil
}

// CustomInstallScriptConfig holds configuration for custom install script generation
type CustomInstallScriptConfig struct {
	BridgeVersion string
	BridgeURL     string
	PairCode      string
	ConfigPath    string
	InstallDir    string
}

// GenerateCustomInstallScript generates a customized installation script
func (sg *ScriptGenerator) GenerateCustomInstallScript(outputPath string, config *CustomInstallScriptConfig) error {
	tmpl, err := template.New("install").Parse(InstallScriptTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse install script template: %w", err)
	}
	
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Execute template
	if err := tmpl.Execute(file, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	
	// Set executable permissions
	if err := os.Chmod(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}
	
	fmt.Printf("Custom install script generated: %s\n", outputPath)
	return nil
}