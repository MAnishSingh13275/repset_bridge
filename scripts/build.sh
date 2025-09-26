#!/bin/bash

# Build script for cross-platform binary compilation
# Supports Windows, macOS, and Linux builds with proper signing

set -e

# Configuration
PROJECT_NAME="gym-door-bridge"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_DIR="build"
DIST_DIR="dist"

# Build flags
BUILD_FLAGS="-trimpath -ldflags=-s -w -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[BUILD]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Clean previous builds
clean() {
    log "Cleaning previous builds..."
    rm -rf "${BUILD_DIR}" "${DIST_DIR}"
    mkdir -p "${BUILD_DIR}" "${DIST_DIR}"
}

# Build for specific platform
build_platform() {
    local goos=$1
    local goarch=$2
    local ext=$3
    local output_name="${PROJECT_NAME}-${goos}-${goarch}${ext}"
    
    log "Building ${output_name}..."
    
    GOOS=${goos} GOARCH=${goarch} CGO_ENABLED=1 go build \
        ${BUILD_FLAGS} \
        -o "${BUILD_DIR}/${output_name}" \
        ./cmd
    
    if [ $? -eq 0 ]; then
        log "✓ Built ${output_name}"
    else
        error "✗ Failed to build ${output_name}"
    fi
}

# Sign Windows binary
sign_windows() {
    local binary=$1
    
    if [ -z "${WINDOWS_CERT_PATH}" ] || [ -z "${WINDOWS_CERT_PASSWORD}" ]; then
        warn "Windows signing certificate not configured, skipping signing"
        return 0
    fi
    
    if ! command -v osslsigncode &> /dev/null; then
        warn "osslsigncode not found, skipping Windows binary signing"
        return 0
    fi
    
    log "Signing Windows binary: ${binary}"
    osslsigncode sign \
        -pkcs12 "${WINDOWS_CERT_PATH}" \
        -pass "${WINDOWS_CERT_PASSWORD}" \
        -n "${PROJECT_NAME}" \
        -i "https://repset.onezy.in" \
        -t "http://timestamp.digicert.com" \
        -in "${binary}" \
        -out "${binary}.signed"
    
    mv "${binary}.signed" "${binary}"
    log "✓ Windows binary signed"
}

# Sign macOS binary
sign_macos() {
    local binary=$1
    
    if [ -z "${MACOS_CERT_ID}" ]; then
        warn "macOS signing certificate not configured, skipping signing"
        return 0
    fi
    
    if ! command -v codesign &> /dev/null; then
        warn "codesign not found, skipping macOS binary signing"
        return 0
    fi
    
    log "Signing macOS binary: ${binary}"
    codesign --force --sign "${MACOS_CERT_ID}" "${binary}"
    
    # Notarize if credentials are available
    if [ -n "${MACOS_NOTARY_USER}" ] && [ -n "${MACOS_NOTARY_PASSWORD}" ]; then
        log "Notarizing macOS binary..."
        
        # Create a zip for notarization
        local zip_name="${binary}.zip"
        zip -j "${zip_name}" "${binary}"
        
        # Submit for notarization
        xcrun altool --notarize-app \
            --primary-bundle-id "com.repset.onezy.${PROJECT_NAME}" \
            --username "${MACOS_NOTARY_USER}" \
            --password "${MACOS_NOTARY_PASSWORD}" \
            --file "${zip_name}"
        
        rm "${zip_name}"
        log "✓ macOS binary submitted for notarization"
    fi
    
    log "✓ macOS binary signed"
}

# Create checksums
create_checksums() {
    log "Creating checksums..."
    cd "${BUILD_DIR}"
    
    for file in *; do
        if [ -f "$file" ]; then
            sha256sum "$file" >> checksums.txt
        fi
    done
    
    cd ..
    log "✓ Checksums created"
}

# Package binaries
package_binaries() {
    log "Packaging binaries..."
    
    cd "${BUILD_DIR}"
    
    # Create individual archives
    for file in gym-door-bridge-*; do
        if [ -f "$file" ] && [[ "$file" != *.zip ]] && [[ "$file" != *.tar.gz ]]; then
            case "$file" in
                *windows*)
                    zip "../${DIST_DIR}/${file}.zip" "$file" checksums.txt
                    ;;
                *)
                    tar -czf "../${DIST_DIR}/${file}.tar.gz" "$file" checksums.txt
                    ;;
            esac
        fi
    done
    
    cd ..
    log "✓ Binaries packaged"
}

# Main build process
main() {
    log "Starting build process for ${PROJECT_NAME} v${VERSION}"
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        error "Go is not installed or not in PATH"
    fi
    
    # Clean previous builds
    clean
    
    # Build for different platforms
    log "Building binaries..."
    
    # Windows builds
    build_platform "windows" "amd64" ".exe"
    build_platform "windows" "386" ".exe"
    
    # macOS builds
    build_platform "darwin" "amd64" ""
    build_platform "darwin" "arm64" ""
    
    # Linux builds (for Docker)
    build_platform "linux" "amd64" ""
    build_platform "linux" "arm64" ""
    
    # Sign binaries
    log "Signing binaries..."
    
    # Sign Windows binaries
    for binary in ${BUILD_DIR}/gym-door-bridge-windows-*.exe; do
        if [ -f "$binary" ]; then
            sign_windows "$binary"
        fi
    done
    
    # Sign macOS binaries
    for binary in ${BUILD_DIR}/gym-door-bridge-darwin-*; do
        if [ -f "$binary" ]; then
            sign_macos "$binary"
        fi
    done
    
    # Create checksums and package
    create_checksums
    package_binaries
    
    log "Build completed successfully!"
    log "Binaries available in: ${DIST_DIR}/"
    ls -la "${DIST_DIR}/"
}

# Handle command line arguments
case "${1:-}" in
    clean)
        clean
        ;;
    *)
        main
        ;;
esac