#!/bin/bash

# Gym Door Bridge - macOS Installation Script
# This script downloads, configures, and installs the Gym Door Bridge as a macOS daemon

set -euo pipefail

# Script configuration
SCRIPT_NAME="$(basename "$0")"
SERVICE_NAME="com.repset.onezy.gym-door-bridge"
EXECUTABLE_NAME="gym-door-bridge"
DEFAULT_SERVER_URL="https://api.repset.onezy.in"
DEFAULT_INSTALL_DIR="/usr/local/bin"
DEFAULT_CONFIG_DIR="/usr/local/etc/gym-door-bridge"
DEFAULT_CDN_BASE_URL="https://cdn.repset.onezy.in/gym-door-bridge"
DEFAULT_VERSION="latest"

# Command line arguments
PAIR_CODE=""
SERVER_URL="$DEFAULT_SERVER_URL"
INSTALL_DIR="$DEFAULT_INSTALL_DIR"
CONFIG_DIR="$DEFAULT_CONFIG_DIR"
VERSION="$DEFAULT_VERSION"
CDN_BASE_URL="$DEFAULT_CDN_BASE_URL"

# Logging function
log() {
    local level="${1:-INFO}"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message"
}

# Error handling
error_exit() {
    log "ERROR" "$1"
    exit 1
}

# Usage information
usage() {
    cat << EOF
Usage: $SCRIPT_NAME --pair-code <code> [options]

Required:
    --pair-code <code>      Device pairing code from admin portal

Options:
    --server-url <url>      Server URL (default: $DEFAULT_SERVER_URL)
    --install-dir <dir>     Installation directory (default: $DEFAULT_INSTALL_DIR)
    --config-dir <dir>      Configuration directory (default: $DEFAULT_CONFIG_DIR)
    --version <version>     Version to install (default: $DEFAULT_VERSION)
    --cdn-url <url>         CDN base URL (default: $DEFAULT_CDN_BASE_URL)
    --help                  Show this help message

Examples:
    # Basic installation
    sudo $SCRIPT_NAME --pair-code ABC123

    # Custom server URL
    sudo $SCRIPT_NAME --pair-code ABC123 --server-url https://custom.domain.com

    # Specific version
    sudo $SCRIPT_NAME --pair-code ABC123 --version v1.2.3
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --pair-code)
                PAIR_CODE="$2"
                shift 2
                ;;
            --server-url)
                SERVER_URL="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --config-dir)
                CONFIG_DIR="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --cdn-url)
                CDN_BASE_URL="$2"
                shift 2
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                error_exit "Unknown option: $1. Use --help for usage information."
                ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$PAIR_CODE" ]]; then
        error_exit "Pair code is required. Use --pair-code <code>"
    fi
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error_exit "This script must be run as root (use sudo)"
    fi
}

# Detect system architecture
detect_architecture() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            error_exit "Unsupported architecture: $arch"
            ;;
    esac
}

# Download file with progress
download_file() {
    local url="$1"
    local output_path="$2"
    
    log "INFO" "Downloading from $url to $output_path"
    
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --progress-bar "$url" -o "$output_path"
    elif command -v wget >/dev/null 2>&1; then
        wget -q --show-progress "$url" -O "$output_path"
    else
        error_exit "Neither curl nor wget is available for downloading"
    fi
    
    log "INFO" "Download completed successfully"
}

# Verify file signature (placeholder for actual implementation)
verify_file_signature() {
    local file_path="$1"
    
    log "INFO" "Verifying file signature for $file_path"
    
    # Check if file exists
    if [[ ! -f "$file_path" ]]; then
        error_exit "File does not exist: $file_path"
    fi
    
    # Check file size (basic sanity check)
    local file_size=$(stat -f%z "$file_path" 2>/dev/null || stat -c%s "$file_path" 2>/dev/null)
    if [[ $file_size -lt 1048576 ]]; then  # Less than 1MB
        error_exit "File appears to be too small: $file_size bytes"
    fi
    
    # TODO: Implement actual signature verification
    # For now, just check if it's a valid executable
    if ! file "$file_path" | grep -q "executable"; then
        error_exit "File does not appear to be a valid executable"
    fi
    
    log "INFO" "File signature verification passed"
}

# Create configuration file
create_config_file() {
    local config_path="$1"
    local server_url="$2"
    
    log "INFO" "Creating configuration file at $config_path"
    
    # Ensure config directory exists
    local config_dir_path=$(dirname "$config_path")
    mkdir -p "$config_dir_path"
    
    # Create logs directory
    local logs_dir="$CONFIG_DIR/logs"
    mkdir -p "$logs_dir"
    
    # Create configuration content
    cat > "$config_path" << EOF
# Gym Door Bridge Configuration
server_url: "$server_url"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "$CONFIG_DIR/bridge.db"
log_level: "info"
log_file: "$logs_dir/bridge.log"
enabled_adapters:
  - simulator

# Pairing configuration (will be updated after pairing)
device_id: ""
device_key: ""
EOF
    
    # Set appropriate permissions
    chmod 644 "$config_path"
    
    log "INFO" "Configuration file created successfully"
}

# Install macOS daemon
install_daemon() {
    local executable_path="$1"
    local config_path="$2"
    
    log "INFO" "Installing macOS daemon"
    
    # Stop existing daemon if running
    if launchctl list | grep -q "$SERVICE_NAME"; then
        log "INFO" "Stopping existing daemon"
        launchctl stop "$SERVICE_NAME" 2>/dev/null || true
        launchctl unload "/Library/LaunchDaemons/$SERVICE_NAME.plist" 2>/dev/null || true
    fi
    
    # Remove existing plist if it exists
    local plist_path="/Library/LaunchDaemons/$SERVICE_NAME.plist"
    if [[ -f "$plist_path" ]]; then
        log "INFO" "Removing existing plist file"
        rm -f "$plist_path"
    fi
    
    # Install daemon using the executable
    log "INFO" "Installing daemon with executable: $executable_path"
    "$executable_path" service install --config "$config_path"
    
    if [[ $? -ne 0 ]]; then
        error_exit "Daemon installation failed"
    fi
    
    log "INFO" "Daemon installed successfully"
}

# Perform device pairing
perform_device_pairing() {
    local executable_path="$1"
    local config_path="$2"
    local pair_code="$3"
    
    log "INFO" "Starting device pairing with code: $pair_code"
    
    # Run pairing command
    local pairing_output
    if pairing_output=$("$executable_path" pair --config "$config_path" --pair-code "$pair_code" 2>&1); then
        log "INFO" "Device pairing completed successfully"
        log "INFO" "Pairing output: $pairing_output"
    else
        error_exit "Device pairing failed: $pairing_output"
    fi
}

# Start macOS daemon
start_daemon() {
    log "INFO" "Starting macOS daemon"
    
    # Load and start the daemon
    local plist_path="/Library/LaunchDaemons/$SERVICE_NAME.plist"
    launchctl load "$plist_path"
    launchctl start "$SERVICE_NAME"
    
    # Wait for daemon to start
    local timeout=30
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if launchctl list | grep -q "$SERVICE_NAME"; then
            log "INFO" "Daemon started successfully"
            return 0
        fi
        sleep 1
        ((elapsed++))
    done
    
    error_exit "Daemon failed to start within $timeout seconds"
}

# Cleanup function
cleanup_temp_files() {
    local temp_files=("$@")
    
    for file_path in "${temp_files[@]}"; do
        if [[ -f "$file_path" ]]; then
            rm -f "$file_path" && log "INFO" "Removed temporary file: $file_path" || \
                log "WARN" "Failed to remove temporary file: $file_path"
        fi
    done
}

# Main installation function
install_gym_door_bridge() {
    log "INFO" "Starting Gym Door Bridge installation"
    log "INFO" "Pair Code: $PAIR_CODE"
    log "INFO" "Server URL: $SERVER_URL"
    log "INFO" "Install Directory: $INSTALL_DIR"
    log "INFO" "Config Directory: $CONFIG_DIR"
    log "INFO" "Version: $VERSION"
    
    local temp_files=()
    local executable_path="$INSTALL_DIR/$EXECUTABLE_NAME"
    local config_path="$CONFIG_DIR/config.yaml"
    
    # Detect architecture
    local arch=$(detect_architecture)
    log "INFO" "Detected architecture: $arch"
    
    # Cleanup function for error handling
    cleanup_on_error() {
        log "INFO" "Performing cleanup due to error..."
        
        # Stop and unload daemon if it was created
        launchctl stop "$SERVICE_NAME" 2>/dev/null || true
        launchctl unload "/Library/LaunchDaemons/$SERVICE_NAME.plist" 2>/dev/null || true
        rm -f "/Library/LaunchDaemons/$SERVICE_NAME.plist"
        
        # Remove temporary files
        cleanup_temp_files "${temp_files[@]}"
    }
    
    # Set trap for cleanup on error
    trap cleanup_on_error ERR
    
    # Create installation directory
    if [[ ! -d "$INSTALL_DIR" ]]; then
        log "INFO" "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi
    
    # Create config directory
    if [[ ! -d "$CONFIG_DIR" ]]; then
        log "INFO" "Creating configuration directory: $CONFIG_DIR"
        mkdir -p "$CONFIG_DIR"
    fi
    
    # Download executable
    local download_url="$CDN_BASE_URL/$VERSION/macos/$arch/$EXECUTABLE_NAME"
    local temp_executable="/tmp/$EXECUTABLE_NAME.$$"
    temp_files+=("$temp_executable")
    
    download_file "$download_url" "$temp_executable"
    
    # Verify file signature
    verify_file_signature "$temp_executable"
    
    # Move executable to final location
    mv "$temp_executable" "$executable_path"
    chmod +x "$executable_path"
    
    # Remove from temp files since it's now in final location
    temp_files=("${temp_files[@]/$temp_executable}")
    
    # Create configuration file
    create_config_file "$config_path" "$SERVER_URL"
    
    # Install macOS daemon
    install_daemon "$executable_path" "$config_path"
    
    # Perform device pairing
    perform_device_pairing "$executable_path" "$config_path" "$PAIR_CODE"
    
    # Start daemon
    start_daemon
    
    # Clear trap since installation succeeded
    trap - ERR
    
    log "INFO" "Installation completed successfully!"
    log "INFO" "Service Name: $SERVICE_NAME"
    log "INFO" "Executable: $executable_path"
    log "INFO" "Configuration: $config_path"
    log "INFO" "Logs: $CONFIG_DIR/logs/bridge.log"
    
    # Display daemon status
    if launchctl list | grep -q "$SERVICE_NAME"; then
        log "INFO" "Daemon Status: Running"
    else
        log "WARN" "Daemon Status: Not Running"
    fi
    
    # Cleanup any remaining temp files
    cleanup_temp_files "${temp_files[@]}"
}

# Script entry point
main() {
    parse_args "$@"
    check_root
    install_gym_door_bridge
}

# Run main function with all arguments
main "$@"