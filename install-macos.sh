#!/bin/bash

# Gym Door Bridge - macOS One-Click Installer
# This script downloads and installs the Gym Door Bridge automatically

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Emojis
SUCCESS="✅"
INFO="ℹ️ "
WARNING="⚠️ "
ERROR="❌"
ROCKET="🚀"
TARGET="🎯"
PARTY="🎉"

# Functions for colored output
print_success() { echo -e "${GREEN}${SUCCESS} $1${NC}"; }
print_info() { echo -e "${CYAN}${INFO} $1${NC}"; }
print_warning() { echo -e "${YELLOW}${WARNING} $1${NC}"; }
print_error() { echo -e "${RED}${ERROR} $1${NC}"; }
print_header() { echo -e "\n${MAGENTA}${TARGET} $1${NC}"; }

# Variables
PAIR_CODE=""
FORCE_INSTALL=false
GITHUB_REPO="yourorg/repset-bridge"  # Update with actual repo
SERVICE_NAME="com.gymbridge.door-bridge"
INSTALL_DIR="/Applications/GymDoorBridge"
PLIST_PATH="/Library/LaunchDaemons/$SERVICE_NAME.plist"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--pair-code)
            PAIR_CODE="$2"
            shift 2
            ;;
        -f|--force)
            FORCE_INSTALL=true
            shift
            ;;
        -h|--help)
            cat << 'EOF'
🚀 Gym Door Bridge - macOS One-Click Installer

USAGE:
    ./install-macos.sh [OPTIONS]

OPTIONS:
    -p, --pair-code <code>    Automatically pair with platform using this code
    -f, --force               Force reinstall even if already installed
    -h, --help                Show this help message

EXAMPLES:
    # Install bridge only
    ./install-macos.sh

    # Install and pair in one step
    ./install-macos.sh --pair-code "ABC123DEF456"

    # Force reinstall
    ./install-macos.sh --force

REQUIREMENTS:
    - macOS 10.15 (Catalina) or later
    - Administrator privileges (script will prompt for sudo)
    - Internet connection for download

For support: support@repset.onezy.in
EOF
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            print_info "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Show header
echo -e "${CYAN}
${ROCKET} Gym Door Bridge - macOS Installer
========================================
This will install the Gym Door Bridge as a macOS daemon that:
- Runs automatically on startup
- Restarts automatically on failure
- Updates automatically in the background
- Discovers your door hardware automatically
${NC}"

# Check if running as root (we don't want this)
if [[ $EUID -eq 0 ]]; then
    print_error "Please do not run this script as root!"
    print_info "The script will ask for sudo when needed."
    exit 1
fi

# Check macOS version
print_header "Checking system requirements..."

os_version=$(sw_vers -productVersion)
major_version=$(echo "$os_version" | cut -d. -f1)
minor_version=$(echo "$os_version" | cut -d. -f2)

if [[ $major_version -lt 10 ]] || [[ $major_version -eq 10 && $minor_version -lt 15 ]]; then
    print_error "macOS 10.15 (Catalina) or later is required. Current version: $os_version"
    exit 1
fi
print_success "macOS version: $os_version ✓"

# Check if already installed
if [[ -f "$PLIST_PATH" ]] && [[ "$FORCE_INSTALL" != true ]]; then
    print_warning "Gym Door Bridge is already installed!"
    read -p "Do you want to reinstall? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Installation cancelled."
        exit 0
    fi
    FORCE_INSTALL=true
fi

# Check for required tools
print_info "Checking for required tools..."
if ! command -v curl &> /dev/null; then
    print_error "curl is required but not installed"
    exit 1
fi
print_success "curl available ✓"

# Download latest release
print_header "Downloading latest version..."

TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Get latest release info
print_info "Fetching release information..."
RELEASE_URL="https://api.github.com/repos/$GITHUB_REPO/releases/latest"

if ! RELEASE_JSON=$(curl -s -H "User-Agent: GymDoorBridge-Installer" "$RELEASE_URL"); then
    print_error "Failed to fetch release information"
    exit 1
fi

VERSION=$(echo "$RELEASE_JSON" | grep -o '"tag_name":[[:space:]]*"[^"]*"' | sed 's/"tag_name":[[:space:]]*"\([^"]*\)"/\1/')
if [[ -z "$VERSION" ]]; then
    print_error "Could not parse version from release"
    exit 1
fi
print_success "Latest version: $VERSION"

# Find macOS binary
DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url":[[:space:]]*"[^"]*macos[^"]*"' | sed 's/"browser_download_url":[[:space:]]*"\([^"]*\)"/\1/' | head -1)
if [[ -z "$DOWNLOAD_URL" ]]; then
    DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url":[[:space:]]*"[^"]*darwin[^"]*"' | sed 's/"browser_download_url":[[:space:]]*"\([^"]*\)"/\1/' | head -1)
fi

if [[ -z "$DOWNLOAD_URL" ]]; then
    print_error "No macOS executable found in release"
    exit 1
fi

FILENAME=$(basename "$DOWNLOAD_URL")
TEMP_FILE="$TEMP_DIR/$FILENAME"

print_info "Downloading: $FILENAME"
print_info "From: $DOWNLOAD_URL"

if ! curl -L -o "$TEMP_FILE" --progress-bar "$DOWNLOAD_URL"; then
    print_error "Failed to download executable"
    exit 1
fi

print_success "Downloaded: $TEMP_FILE"

# Make executable
chmod +x "$TEMP_FILE"

# Install
print_header "Installing Gym Door Bridge..."

# Stop existing service if running
if launchctl list | grep -q "$SERVICE_NAME" 2>/dev/null; then
    print_info "Stopping existing service..."
    sudo launchctl unload "$PLIST_PATH" 2>/dev/null || true
    sleep 2
fi

# Run installer
print_info "Running installer with administrator privileges..."
if ! sudo "$TEMP_FILE" install; then
    print_error "Installation failed"
    print_info "Please check the output above for error details"
    exit 1
fi

print_success "Installation completed successfully!"

# Pair with platform if code provided
if [[ -n "$PAIR_CODE" ]]; then
    print_header "Pairing with platform..."
    if "$INSTALL_DIR/gym-door-bridge" pair "$PAIR_CODE"; then
        print_success "Successfully paired with platform!"
    else
        print_warning "Pairing failed. You can pair manually later with:"
        print_info "gym-door-bridge pair YOUR_PAIR_CODE"
    fi
fi

# Verify installation
print_header "Verifying installation..."

# Check if service is installed
if [[ -f "$PLIST_PATH" ]]; then
    print_success "Service plist installed ✓"
else
    print_warning "Service plist not found"
fi

# Check if daemon is loaded
if launchctl list | grep -q "$SERVICE_NAME"; then
    print_success "Service is loaded and running ✓"
else
    print_warning "Service is not running"
fi

# Check if executable exists
EXEC_PATH="$INSTALL_DIR/gym-door-bridge"
if [[ -f "$EXEC_PATH" ]]; then
    print_success "Executable installed: $EXEC_PATH ✓"
else
    print_warning "Executable not found at expected location"
fi

# Test status command
if "$EXEC_PATH" status &>/dev/null; then
    print_success "Bridge status command working ✓"
fi

# Show completion message
echo -e "${GREEN}

${PARTY} Installation Complete!
========================

The Gym Door Bridge has been installed and is running as a macOS daemon.

WHAT'S RUNNING:
✅ LaunchDaemon: $SERVICE_NAME
✅ Auto-start: Enabled (will start automatically on boot)
✅ Auto-restart: Enabled (will restart if it crashes)
✅ Auto-update: Enabled (will update itself automatically)

NEXT STEPS:${NC}"

if [[ -z "$PAIR_CODE" ]]; then
    echo -e "${YELLOW}1. Get a pairing code from your gym management platform
2. Run: gym-door-bridge pair YOUR_PAIR_CODE
3. The bridge will automatically discover your door hardware
${NC}"
else
    echo -e "${YELLOW}1. Check your gym management platform - the bridge should show as \"Connected\"
2. The bridge will automatically discover your door hardware
3. Test your fingerprint scanner or RFID reader
${NC}"
fi

echo -e "${CYAN}
USEFUL COMMANDS:
- gym-door-bridge status    (check status)
- gym-door-bridge restart   (restart service)
- gym-door-bridge logs      (view recent logs)

SUPPORT:
- Email: support@repset.onezy.in  
- Docs: https://docs.repset.onezy.in/bridge
${NC}"

print_success "Ready to go! The bridge will work automatically in the background."

# Add to PATH if not already there
SHELL_RC=""
case $SHELL in
    */zsh) SHELL_RC="$HOME/.zshrc" ;;
    */bash) SHELL_RC="$HOME/.bash_profile" ;;
    *) SHELL_RC="$HOME/.profile" ;;
esac

if [[ -n "$SHELL_RC" ]] && [[ ! $(echo "$PATH" | grep -q "$INSTALL_DIR") ]]; then
    print_info "Adding $INSTALL_DIR to PATH in $SHELL_RC"
    echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$SHELL_RC"
    print_info "Restart your terminal or run: source $SHELL_RC"
fi

echo
print_success "Installation complete! 🚀"