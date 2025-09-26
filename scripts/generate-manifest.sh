#!/bin/bash

# Script to generate manifest.json from template
# Used for update distribution system

set -e

# Configuration
TEMPLATE_FILE="scripts/manifest.json.template"
OUTPUT_FILE="dist/manifest.json"
BUILD_DIR="build"

# Default values
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
RELEASE_DATE=${RELEASE_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
DESCRIPTION=${DESCRIPTION:-"Gym Door Access Bridge update"}
CHANGELOG=${CHANGELOG:-"See release notes for details"}
MIN_VERSION=${MIN_VERSION:-"1.0.0"}
ROLLOUT_PERCENTAGE=${ROLLOUT_PERCENTAGE:-100}
CRITICAL=${CRITICAL:-false}
CDN_BASE_URL=${CDN_BASE_URL:-"https://cdn.repset.onezy.in/gym-door-bridge"}
BREAKING_CHANGES=${BREAKING_CHANGES:-false}
ROLLBACK_ENABLED=${ROLLBACK_ENABLED:-true}
PREVIOUS_VERSION=${PREVIOUS_VERSION:-""}
BUILD_NUMBER=${BUILD_NUMBER:-$(date +%Y%m%d%H%M%S)}
COMMIT_HASH=${COMMIT_HASH:-$(git rev-parse HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
RELEASE_NOTES_URL=${RELEASE_NOTES_URL:-"https://docs.repset.onezy.in/releases/${VERSION}"}
DOCUMENTATION_URL=${DOCUMENTATION_URL:-"https://docs.repset.onezy.in/gym-door-bridge"}

# Feature arrays (JSON format)
NEW_FEATURES=${NEW_FEATURES:-'[]'}
IMPROVEMENTS=${IMPROVEMENTS:-'[]'}
BUG_FIXES=${BUG_FIXES:-'[]'}
SECURITY_FIXES=${SECURITY_FIXES:-'[]'}
DEPRECATIONS=${DEPRECATIONS:-'[]'}
MIGRATIONS=${MIGRATIONS:-'[]'}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[MANIFEST]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Calculate file hash and size
get_file_info() {
    local file_path=$1
    
    if [ ! -f "$file_path" ]; then
        echo "null,0,null"
        return
    fi
    
    local sha256=$(sha256sum "$file_path" | cut -d' ' -f1)
    local size=$(stat -c%s "$file_path" 2>/dev/null || stat -f%z "$file_path" 2>/dev/null || echo "0")
    local signature="null"
    
    # Try to get signature if available
    if [ -f "${file_path}.sig" ]; then
        signature="\"$(cat "${file_path}.sig" | base64 -w 0)\""
    fi
    
    echo "\"$sha256\",$size,$signature"
}

# Generate manifest
generate_manifest() {
    log "Generating manifest.json for version $VERSION"
    
    # Check if template exists
    if [ ! -f "$TEMPLATE_FILE" ]; then
        error "Template file not found: $TEMPLATE_FILE"
    fi
    
    # Create output directory
    mkdir -p "$(dirname "$OUTPUT_FILE")"
    
    # Get file information for each platform
    log "Calculating file hashes and sizes..."
    
    # Windows amd64
    WINDOWS_AMD64_INFO=$(get_file_info "$BUILD_DIR/gym-door-bridge-windows-amd64.exe")
    WINDOWS_AMD64_SHA256=$(echo "$WINDOWS_AMD64_INFO" | cut -d',' -f1)
    WINDOWS_AMD64_SIZE=$(echo "$WINDOWS_AMD64_INFO" | cut -d',' -f2)
    WINDOWS_AMD64_SIGNATURE=$(echo "$WINDOWS_AMD64_INFO" | cut -d',' -f3)
    
    # Windows 386
    WINDOWS_386_INFO=$(get_file_info "$BUILD_DIR/gym-door-bridge-windows-386.exe")
    WINDOWS_386_SHA256=$(echo "$WINDOWS_386_INFO" | cut -d',' -f1)
    WINDOWS_386_SIZE=$(echo "$WINDOWS_386_INFO" | cut -d',' -f2)
    WINDOWS_386_SIGNATURE=$(echo "$WINDOWS_386_INFO" | cut -d',' -f3)
    
    # Darwin amd64
    DARWIN_AMD64_INFO=$(get_file_info "$BUILD_DIR/gym-door-bridge-darwin-amd64")
    DARWIN_AMD64_SHA256=$(echo "$DARWIN_AMD64_INFO" | cut -d',' -f1)
    DARWIN_AMD64_SIZE=$(echo "$DARWIN_AMD64_INFO" | cut -d',' -f2)
    DARWIN_AMD64_SIGNATURE=$(echo "$DARWIN_AMD64_INFO" | cut -d',' -f3)
    
    # Darwin arm64
    DARWIN_ARM64_INFO=$(get_file_info "$BUILD_DIR/gym-door-bridge-darwin-arm64")
    DARWIN_ARM64_SHA256=$(echo "$DARWIN_ARM64_INFO" | cut -d',' -f1)
    DARWIN_ARM64_SIZE=$(echo "$DARWIN_ARM64_INFO" | cut -d',' -f2)
    DARWIN_ARM64_SIGNATURE=$(echo "$DARWIN_ARM64_INFO" | cut -d',' -f3)
    
    # Linux amd64
    LINUX_AMD64_INFO=$(get_file_info "$BUILD_DIR/gym-door-bridge-linux-amd64")
    LINUX_AMD64_SHA256=$(echo "$LINUX_AMD64_INFO" | cut -d',' -f1)
    LINUX_AMD64_SIZE=$(echo "$LINUX_AMD64_INFO" | cut -d',' -f2)
    LINUX_AMD64_SIGNATURE=$(echo "$LINUX_AMD64_INFO" | cut -d',' -f3)
    
    # Linux arm64
    LINUX_ARM64_INFO=$(get_file_info "$BUILD_DIR/gym-door-bridge-linux-arm64")
    LINUX_ARM64_SHA256=$(echo "$LINUX_ARM64_INFO" | cut -d',' -f1)
    LINUX_ARM64_SIZE=$(echo "$LINUX_ARM64_INFO" | cut -d',' -f2)
    LINUX_ARM64_SIGNATURE=$(echo "$LINUX_ARM64_INFO" | cut -d',' -f3)
    
    # Rollback URL
    ROLLBACK_URL="$CDN_BASE_URL/manifest-$PREVIOUS_VERSION.json"
    if [ -z "$PREVIOUS_VERSION" ]; then
        ROLLBACK_URL="null"
    else
        ROLLBACK_URL="\"$ROLLBACK_URL\""
    fi
    
    # Generate manifest from template
    log "Substituting template variables..."
    
    sed -e "s|{{VERSION}}|$VERSION|g" \
        -e "s|{{RELEASE_DATE}}|$RELEASE_DATE|g" \
        -e "s|{{DESCRIPTION}}|$DESCRIPTION|g" \
        -e "s|{{CHANGELOG}}|$CHANGELOG|g" \
        -e "s|{{MIN_VERSION}}|$MIN_VERSION|g" \
        -e "s|{{ROLLOUT_PERCENTAGE}}|$ROLLOUT_PERCENTAGE|g" \
        -e "s|{{CRITICAL}}|$CRITICAL|g" \
        -e "s|{{CDN_BASE_URL}}|$CDN_BASE_URL|g" \
        -e "s|{{WINDOWS_AMD64_SHA256}}|$WINDOWS_AMD64_SHA256|g" \
        -e "s|{{WINDOWS_AMD64_SIZE}}|$WINDOWS_AMD64_SIZE|g" \
        -e "s|{{WINDOWS_AMD64_SIGNATURE}}|$WINDOWS_AMD64_SIGNATURE|g" \
        -e "s|{{WINDOWS_386_SHA256}}|$WINDOWS_386_SHA256|g" \
        -e "s|{{WINDOWS_386_SIZE}}|$WINDOWS_386_SIZE|g" \
        -e "s|{{WINDOWS_386_SIGNATURE}}|$WINDOWS_386_SIGNATURE|g" \
        -e "s|{{DARWIN_AMD64_SHA256}}|$DARWIN_AMD64_SHA256|g" \
        -e "s|{{DARWIN_AMD64_SIZE}}|$DARWIN_AMD64_SIZE|g" \
        -e "s|{{DARWIN_AMD64_SIGNATURE}}|$DARWIN_AMD64_SIGNATURE|g" \
        -e "s|{{DARWIN_ARM64_SHA256}}|$DARWIN_ARM64_SHA256|g" \
        -e "s|{{DARWIN_ARM64_SIZE}}|$DARWIN_ARM64_SIZE|g" \
        -e "s|{{DARWIN_ARM64_SIGNATURE}}|$DARWIN_ARM64_SIGNATURE|g" \
        -e "s|{{LINUX_AMD64_SHA256}}|$LINUX_AMD64_SHA256|g" \
        -e "s|{{LINUX_AMD64_SIZE}}|$LINUX_AMD64_SIZE|g" \
        -e "s|{{LINUX_AMD64_SIGNATURE}}|$LINUX_AMD64_SIGNATURE|g" \
        -e "s|{{LINUX_ARM64_SHA256}}|$LINUX_ARM64_SHA256|g" \
        -e "s|{{LINUX_ARM64_SIZE}}|$LINUX_ARM64_SIZE|g" \
        -e "s|{{LINUX_ARM64_SIGNATURE}}|$LINUX_ARM64_SIGNATURE|g" \
        -e "s|{{NEW_FEATURES}}|$NEW_FEATURES|g" \
        -e "s|{{IMPROVEMENTS}}|$IMPROVEMENTS|g" \
        -e "s|{{BUG_FIXES}}|$BUG_FIXES|g" \
        -e "s|{{SECURITY_FIXES}}|$SECURITY_FIXES|g" \
        -e "s|{{BREAKING_CHANGES}}|$BREAKING_CHANGES|g" \
        -e "s|{{DEPRECATIONS}}|$DEPRECATIONS|g" \
        -e "s|{{MIGRATIONS}}|$MIGRATIONS|g" \
        -e "s|{{ROLLBACK_ENABLED}}|$ROLLBACK_ENABLED|g" \
        -e "s|{{PREVIOUS_VERSION}}|$PREVIOUS_VERSION|g" \
        -e "s|{{ROLLBACK_URL}}|$ROLLBACK_URL|g" \
        -e "s|{{BUILD_NUMBER}}|$BUILD_NUMBER|g" \
        -e "s|{{COMMIT_HASH}}|$COMMIT_HASH|g" \
        -e "s|{{BUILD_TIME}}|$BUILD_TIME|g" \
        -e "s|{{RELEASE_NOTES_URL}}|$RELEASE_NOTES_URL|g" \
        -e "s|{{DOCUMENTATION_URL}}|$DOCUMENTATION_URL|g" \
        "$TEMPLATE_FILE" > "$OUTPUT_FILE"
    
    # Validate JSON
    if command -v jq &> /dev/null; then
        log "Validating JSON..."
        if jq empty "$OUTPUT_FILE" 2>/dev/null; then
            log "✓ Manifest JSON is valid"
        else
            error "✗ Generated manifest JSON is invalid"
        fi
    else
        warn "jq not found, skipping JSON validation"
    fi
    
    log "✓ Manifest generated: $OUTPUT_FILE"
    
    # Show summary
    log "Manifest Summary:"
    echo "  Version: $VERSION"
    echo "  Release Date: $RELEASE_DATE"
    echo "  Rollout: $ROLLOUT_PERCENTAGE%"
    echo "  Critical: $CRITICAL"
    echo "  Rollback Enabled: $ROLLBACK_ENABLED"
}

# Main execution
main() {
    case "${1:-}" in
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Environment variables:"
            echo "  VERSION              - Release version (default: git describe)"
            echo "  RELEASE_DATE         - Release date (default: current UTC time)"
            echo "  DESCRIPTION          - Release description"
            echo "  CHANGELOG            - Changelog summary"
            echo "  MIN_VERSION          - Minimum compatible version"
            echo "  ROLLOUT_PERCENTAGE   - Rollout percentage (default: 100)"
            echo "  CRITICAL             - Critical update flag (default: false)"
            echo "  CDN_BASE_URL         - CDN base URL for downloads"
            echo "  BREAKING_CHANGES     - Breaking changes flag (default: false)"
            echo "  ROLLBACK_ENABLED     - Enable rollback (default: true)"
            echo "  PREVIOUS_VERSION     - Previous version for rollback"
            echo "  NEW_FEATURES         - JSON array of new features"
            echo "  IMPROVEMENTS         - JSON array of improvements"
            echo "  BUG_FIXES           - JSON array of bug fixes"
            echo "  SECURITY_FIXES      - JSON array of security fixes"
            echo ""
            echo "Example:"
            echo "  VERSION=1.2.3 ROLLOUT_PERCENTAGE=50 $0"
            ;;
        *)
            generate_manifest
            ;;
    esac
}

main "$@"