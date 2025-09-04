# Deployment Guide

This guide covers the deployment of the Gym Door Access Bridge across different environments and platforms.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Build Process](#build-process)
- [Binary Deployment](#binary-deployment)
- [Docker Deployment](#docker-deployment)
- [Update Distribution](#update-distribution)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Development Environment

- Go 1.21 or later
- Git
- Make (optional, for convenience)

### Build Environment

- Cross-compilation support for target platforms
- Code signing certificates (for production)
- Docker (for containerized builds)
- Access to CDN for distribution

### Target Environment

- Windows 10/11 or Windows Server 2019/2022
- macOS 10.15+ (Intel or Apple Silicon)
- Linux (for Docker deployments)
- Minimum 2GB RAM, 1GB disk space
- Network connectivity to SaaS platform

## Build Process

### Local Development Build

```bash
# Clone repository
git clone https://github.com/yourdomain/gym-door-bridge.git
cd gym-door-bridge

# Build for current platform
go build -o gym-door-bridge ./cmd

# Run locally
./gym-door-bridge --config config.yaml.example --log-level debug
```

### Cross-Platform Build

#### Using Build Scripts

**Linux/macOS:**
```bash
# Make script executable
chmod +x scripts/build.sh

# Build all platforms
./scripts/build.sh

# Clean build artifacts
./scripts/build.sh clean
```

**Windows:**
```powershell
# Build all platforms
.\scripts\build.ps1

# Build without signing
.\scripts\build.ps1 -SkipSigning

# Clean build artifacts
.\scripts\build.ps1 -Action clean
```

#### Manual Cross-Compilation

```bash
# Windows 64-bit
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o build/gym-door-bridge-windows-amd64.exe ./cmd

# Windows 32-bit
GOOS=windows GOARCH=386 CGO_ENABLED=1 go build -o build/gym-door-bridge-windows-386.exe ./cmd

# macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o build/gym-door-bridge-darwin-amd64 ./cmd

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o build/gym-door-bridge-darwin-arm64 ./cmd

# Linux 64-bit
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o build/gym-door-bridge-linux-amd64 ./cmd
```

### Code Signing

#### Windows Authenticode Signing

```bash
# Set environment variables
export WINDOWS_CERT_PATH="/path/to/certificate.p12"
export WINDOWS_CERT_PASSWORD="certificate_password"

# Sign binary (done automatically by build script)
osslsigncode sign \
    -pkcs12 "$WINDOWS_CERT_PATH" \
    -pass "$WINDOWS_CERT_PASSWORD" \
    -n "Gym Door Access Bridge" \
    -i "https://repset.onezy.in" \
    -t "http://timestamp.digicert.com" \
    -in "gym-door-bridge-windows-amd64.exe" \
    -out "gym-door-bridge-windows-amd64.exe.signed"
```

#### macOS Code Signing and Notarization

```bash
# Set environment variables
export MACOS_CERT_ID="Developer ID Application: Your Name (TEAM_ID)"
export MACOS_NOTARY_USER="your-apple-id@example.com"
export MACOS_NOTARY_PASSWORD="app-specific-password"

# Sign binary
codesign --force --sign "$MACOS_CERT_ID" gym-door-bridge-darwin-amd64

# Notarize (done automatically by build script)
zip gym-door-bridge-darwin-amd64.zip gym-door-bridge-darwin-amd64
xcrun altool --notarize-app \
    --primary-bundle-id "com.yourdomain.gym-door-bridge" \
    --username "$MACOS_NOTARY_USER" \
    --password "$MACOS_NOTARY_PASSWORD" \
    --file gym-door-bridge-darwin-amd64.zip
```

## Binary Deployment

### Windows Deployment

#### Service Installation

```powershell
# Download and install using PowerShell script
iwr https://cdn.repset.onezy.in/install.ps1 | iex; Install-Bridge -PairCode ABC123

# Manual installation
.\gym-door-bridge-windows-amd64.exe service install --config "C:\Program Files\GymDoorBridge\config.yaml"
.\gym-door-bridge-windows-amd64.exe service start
```

#### Service Management

```powershell
# Check service status
.\gym-door-bridge-windows-amd64.exe service status

# Stop service
.\gym-door-bridge-windows-amd64.exe service stop

# Restart service
.\gym-door-bridge-windows-amd64.exe service restart

# Uninstall service
.\gym-door-bridge-windows-amd64.exe service uninstall
```

### macOS Deployment

#### Daemon Installation

```bash
# Download and install using bash script
curl -sSL https://cdn.repset.onezy.in/install.sh | bash -s -- --pair-code ABC123

# Manual installation
sudo ./gym-door-bridge-darwin-amd64 service install --config /usr/local/etc/gym-door-bridge/config.yaml
sudo ./gym-door-bridge-darwin-amd64 service start
```

#### Daemon Management

```bash
# Check daemon status
./gym-door-bridge-darwin-amd64 service status

# Stop daemon
sudo ./gym-door-bridge-darwin-amd64 service stop

# Restart daemon
sudo ./gym-door-bridge-darwin-amd64 service restart

# Uninstall daemon
sudo ./gym-door-bridge-darwin-amd64 service uninstall
```

## Docker Deployment

### Building Docker Image

```bash
# Build for current platform
docker build -t gym-door-bridge:latest .

# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t gym-door-bridge:latest .

# Build with version tag
docker build --build-arg VERSION=1.2.3 --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) -t gym-door-bridge:1.2.3 .
```

### Running with Docker

#### Using Docker Run

```bash
# Create config file
cp config.yaml.example config.yaml
# Edit config.yaml with your settings

# Run container
docker run -d \
    --name gym-door-bridge \
    --restart unless-stopped \
    -p 8080:8080 \
    -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
    -v bridge_data:/app/data \
    -v bridge_logs:/app/logs \
    gym-door-bridge:latest
```

#### Using Docker Compose

```bash
# Set environment variables
export VERSION=1.2.3
export LOG_LEVEL=info

# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Docker Registry Deployment

```bash
# Tag for registry
docker tag gym-door-bridge:latest your-registry.com/gym-door-bridge:latest

# Push to registry
docker push your-registry.com/gym-door-bridge:latest

# Deploy on target system
docker pull your-registry.com/gym-door-bridge:latest
docker run -d your-registry.com/gym-door-bridge:latest
```

## Update Distribution

### CDN Setup

1. **Upload Binaries**: Upload built binaries to your CDN
2. **Generate Manifest**: Create manifest.json with version information
3. **Update Distribution**: Clients will automatically check for updates

### Manifest Generation

```bash
# Set environment variables for release
export VERSION=1.2.3
export ROLLOUT_PERCENTAGE=100
export CDN_BASE_URL=https://cdn.repset.onezy.in/gym-door-bridge
export NEW_FEATURES='["Feature 1", "Feature 2"]'
export BUG_FIXES='["Fix 1", "Fix 2"]'

# Generate manifest
./scripts/generate-manifest.sh
```

### Staged Rollout

```bash
# Start with 10% rollout
export ROLLOUT_PERCENTAGE=10
./scripts/generate-manifest.sh

# Increase to 50% after monitoring
export ROLLOUT_PERCENTAGE=50
./scripts/generate-manifest.sh

# Full rollout
export ROLLOUT_PERCENTAGE=100
./scripts/generate-manifest.sh
```

### Rollback Process

```bash
# Enable rollback in manifest
export ROLLBACK_ENABLED=true
export PREVIOUS_VERSION=1.2.2
./scripts/generate-manifest.sh

# Emergency rollback - revert to previous manifest
cp dist/manifest-1.2.2.json dist/manifest.json
```

## Monitoring

### Health Checks

```bash
# Check bridge health
curl http://localhost:8080/health

# Check with authentication (if enabled)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/health
```

### Log Monitoring

```bash
# Windows - Event Viewer
# Look for "Gym Door Bridge" source in Application logs

# macOS - Console.app or command line
log show --predicate 'subsystem == "com.yourdomain.gym-door-bridge"' --last 1h

# Linux/Docker
docker logs gym-door-bridge
journalctl -u gym-door-bridge -f
```

### Metrics Collection

```bash
# Prometheus metrics (if enabled)
curl http://localhost:8080/metrics

# Custom metrics endpoint
curl http://localhost:8080/api/v1/metrics
```

## Troubleshooting

### Common Issues

#### Build Issues

**CGO Compilation Errors:**
```bash
# Install build dependencies
# Ubuntu/Debian
sudo apt-get install build-essential libsqlite3-dev

# CentOS/RHEL
sudo yum install gcc sqlite-devel

# macOS
xcode-select --install
brew install sqlite
```

**Cross-compilation Issues:**
```bash
# Install cross-compilation tools
# For Windows targets on Linux
sudo apt-get install gcc-mingw-w64

# Set CGO environment
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc  # For Windows amd64
```

#### Deployment Issues

**Service Installation Fails:**
```bash
# Check permissions
# Windows - Run as Administrator
# macOS/Linux - Use sudo

# Check service dependencies
# Windows - Ensure .NET Framework is installed
# macOS - Check launchd configuration
# Linux - Verify systemd is running
```

**Configuration Issues:**
```bash
# Validate configuration file
./gym-door-bridge --config config.yaml --validate

# Check configuration syntax
# YAML syntax validation
python -c "import yaml; yaml.safe_load(open('config.yaml'))"
```

**Network Connectivity:**
```bash
# Test API connectivity
curl -v https://repset.onezy.in/api/v1/health

# Check DNS resolution
nslookup repset.onezy.in

# Test with proxy (if configured)
curl --proxy http://proxy:8080 https://repset.onezy.in/api/v1/health
```

#### Runtime Issues

**Database Errors:**
```bash
# Check database file permissions
ls -la data/bridge.db

# Verify SQLite installation
sqlite3 --version

# Test database connectivity
sqlite3 data/bridge.db ".tables"
```

**Authentication Failures:**
```bash
# Check device pairing
./gym-door-bridge pair --code ABC123

# Verify HMAC configuration
./gym-door-bridge auth test

# Check system time synchronization
# Windows
w32tm /query /status

# macOS/Linux
timedatectl status
```

**Hardware Adapter Issues:**
```bash
# List available adapters
./gym-door-bridge adapters list

# Test adapter connectivity
./gym-door-bridge adapters test --adapter simulator

# Check adapter logs
./gym-door-bridge logs --adapter fingerprint --level debug
```

### Log Analysis

#### Common Log Patterns

**Successful Operation:**
```
INFO[2024-01-01T10:00:00Z] Bridge starting up config=...
INFO[2024-01-01T10:00:01Z] Hardware adapter initialized adapter=simulator
INFO[2024-01-01T10:00:02Z] Event processor started
INFO[2024-01-01T10:00:03Z] Health monitor active
```

**Network Issues:**
```
WARN[2024-01-01T10:00:00Z] Network connectivity lost, switching to offline mode
INFO[2024-01-01T10:00:01Z] Event queued locally event_id=evt_123
WARN[2024-01-01T10:05:00Z] Retry attempt failed attempt=3 error="connection timeout"
INFO[2024-01-01T10:10:00Z] Network connectivity restored, replaying queued events
```

**Authentication Problems:**
```
ERROR[2024-01-01T10:00:00Z] HMAC validation failed request_id=req_123
WARN[2024-01-01T10:00:01Z] Device authentication rejected, checking key rotation
INFO[2024-01-01T10:00:02Z] Key rotation initiated
```

### Performance Tuning

#### Resource Optimization

**Memory Usage:**
```bash
# Monitor memory usage
# Windows
tasklist /fi "imagename eq gym-door-bridge.exe"

# macOS/Linux
ps aux | grep gym-door-bridge
top -p $(pgrep gym-door-bridge)
```

**Disk Usage:**
```bash
# Check database size
ls -lh data/bridge.db

# Monitor log file growth
du -h logs/

# Clean old logs (if log rotation is not configured)
find logs/ -name "*.log" -mtime +30 -delete
```

**CPU Usage:**
```bash
# Profile CPU usage
./gym-door-bridge --profile-cpu profile.cpu

# Analyze profile
go tool pprof profile.cpu
```

#### Configuration Tuning

**Queue Settings:**
```yaml
# config.yaml
queue:
  maxSize: 10000      # Adjust based on available memory
  batchSize: 100      # Optimize for network conditions
  retryInterval: 30s  # Balance between responsiveness and load
```

**Performance Tier:**
```yaml
# Force specific tier (not recommended for production)
performance:
  forceTier: "normal"  # lite, normal, full
  
# Tier-specific settings
tiers:
  lite:
    heartbeatInterval: 300s
    queueMaxSize: 1000
  normal:
    heartbeatInterval: 60s
    queueMaxSize: 10000
  full:
    heartbeatInterval: 30s
    queueMaxSize: 50000
```

### Support and Maintenance

#### Log Collection

```bash
# Collect diagnostic information
./gym-door-bridge diagnostics collect --output diagnostics.zip

# Manual log collection
# Windows
copy "C:\Program Files\GymDoorBridge\logs\*" diagnostics\
copy "C:\Windows\System32\winevt\Logs\Application.evtx" diagnostics\

# macOS
cp /usr/local/var/log/gym-door-bridge/* diagnostics/
log collect --output diagnostics/system.logarchive --last 24h

# Linux
cp /var/log/gym-door-bridge/* diagnostics/
journalctl -u gym-door-bridge --since "24 hours ago" > diagnostics/service.log
```

#### Remote Support

```bash
# Enable remote diagnostics (temporary)
./gym-door-bridge remote-support enable --duration 1h --token $SUPPORT_TOKEN

# Disable remote diagnostics
./gym-door-bridge remote-support disable
```

For additional support, contact: support@repset.onezy.in