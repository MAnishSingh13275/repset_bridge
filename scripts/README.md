# Build and Deployment Scripts

This directory contains scripts for building, packaging, and deploying the Gym Door Access Bridge.

## Scripts Overview

### Build Scripts

- **`build.sh`** - Cross-platform build script for Linux/macOS
- **`build.ps1`** - Cross-platform build script for Windows PowerShell
- **`Makefile`** - Make-based build system (project root)

### Deployment Scripts

- **`generate-manifest.sh`** - Generates update manifest.json for distribution
- **`release.sh`** - Complete release automation script
- **`install.sh`** - Installation script for Linux/macOS (referenced in docs)
- **`install.ps1`** - Installation script for Windows PowerShell (referenced in docs)

### Configuration Files

- **`manifest.json.template`** - Template for update distribution manifest

## Usage

### Building Binaries

**Linux/macOS:**
```bash
# Make executable (Linux/macOS only)
chmod +x scripts/build.sh scripts/generate-manifest.sh scripts/release.sh

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

**Using Make:**
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build with Docker
make docker-build

# Complete release
make release
```

### Code Signing

#### Windows Authenticode Signing

Set environment variables:
```bash
export WINDOWS_CERT_PATH="/path/to/certificate.p12"
export WINDOWS_CERT_PASSWORD="certificate_password"
```

#### macOS Code Signing and Notarization

Set environment variables:
```bash
export MACOS_CERT_ID="Developer ID Application: Your Name (TEAM_ID)"
export MACOS_NOTARY_USER="your-apple-id@example.com"
export MACOS_NOTARY_PASSWORD="app-specific-password"
```

### Generating Update Manifest

```bash
# Basic manifest generation
./scripts/generate-manifest.sh

# With custom settings
VERSION=1.2.3 \
ROLLOUT_PERCENTAGE=50 \
CDN_BASE_URL=https://cdn.yourdomain.com/gym-door-bridge \
./scripts/generate-manifest.sh
```

### Complete Release Process

```bash
# Stable release
./scripts/release.sh 1.2.3

# Prerelease
./scripts/release.sh --prerelease 1.2.3-rc1

# Gradual rollout
./scripts/release.sh --rollout 50 1.2.3

# Dry run (show what would be done)
./scripts/release.sh --dry-run 1.2.3
```

## Environment Variables

### Build Configuration

- `VERSION` - Release version (default: git describe)
- `BUILD_TIME` - Build timestamp (default: current UTC time)
- `COMMIT_HASH` - Git commit hash (default: git rev-parse HEAD)

### Signing Configuration

- `WINDOWS_CERT_PATH` - Path to Windows code signing certificate (.p12)
- `WINDOWS_CERT_PASSWORD` - Password for Windows certificate
- `MACOS_CERT_ID` - macOS Developer ID for code signing
- `MACOS_NOTARY_USER` - Apple ID for notarization
- `MACOS_NOTARY_PASSWORD` - App-specific password for notarization

### Distribution Configuration

- `CDN_BASE_URL` - Base URL for CDN distribution
- `ROLLOUT_PERCENTAGE` - Percentage rollout for updates (1-100)
- `GITHUB_TOKEN` - GitHub token for creating releases
- `SLACK_WEBHOOK` - Slack webhook URL for notifications

### Docker Configuration

- `DOCKER_USERNAME` - Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub password/token

## CI/CD Integration

### GitHub Actions

The project includes a comprehensive GitHub Actions workflow (`.github/workflows/build-and-release.yml`) that:

1. Runs tests and linting
2. Builds for all platforms
3. Signs binaries (if certificates are configured)
4. Creates Docker images
5. Generates releases
6. Deploys to CDN
7. Sends notifications

### Required Secrets

Configure these secrets in your GitHub repository:

- `WINDOWS_CERT_BASE64` - Base64-encoded Windows certificate
- `WINDOWS_CERT_PASSWORD` - Windows certificate password
- `MACOS_CERT_ID` - macOS signing certificate ID
- `MACOS_NOTARY_USER` - Apple ID for notarization
- `MACOS_NOTARY_PASSWORD` - App-specific password
- `DOCKER_USERNAME` - Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub password
- `CDN_DEPLOY_KEY` - SSH key for CDN deployment
- `CDN_ENDPOINT` - CDN deployment endpoint
- `SLACK_WEBHOOK_URL` - Slack webhook for notifications
- `TEAMS_WEBHOOK_URL` - Microsoft Teams webhook

## Output Artifacts

### Build Artifacts

After running build scripts, you'll find:

```
build/
├── gym-door-bridge-windows-amd64.exe
├── gym-door-bridge-windows-386.exe
├── gym-door-bridge-darwin-amd64
├── gym-door-bridge-darwin-arm64
├── gym-door-bridge-linux-amd64
├── gym-door-bridge-linux-arm64
└── checksums.txt

dist/
├── gym-door-bridge-windows-amd64.exe.zip
├── gym-door-bridge-windows-386.exe.zip
├── gym-door-bridge-darwin-amd64.tar.gz
├── gym-door-bridge-darwin-arm64.tar.gz
├── gym-door-bridge-linux-amd64.tar.gz
├── gym-door-bridge-linux-arm64.tar.gz
└── manifest.json
```

### Docker Images

- `gym-door-bridge:latest` - Latest stable release
- `gym-door-bridge:1.2.3` - Specific version tag
- `gym-door-bridge:main` - Latest main branch build

## Troubleshooting

### Build Issues

**CGO Compilation Errors:**
```bash
# Install build dependencies
# Ubuntu/Debian
sudo apt-get install build-essential libsqlite3-dev

# macOS
xcode-select --install
brew install sqlite
```

**Cross-compilation Issues:**
```bash
# Install cross-compilation tools
sudo apt-get install gcc-mingw-w64  # For Windows targets
sudo apt-get install gcc-aarch64-linux-gnu  # For ARM64 targets
```

### Signing Issues

**Windows Signing:**
- Ensure certificate is valid and not expired
- Check that signtool.exe is available (Windows SDK)
- Verify certificate password is correct

**macOS Signing:**
- Ensure Xcode command line tools are installed
- Check that certificate is installed in keychain
- Verify Apple ID credentials for notarization

### Docker Issues

**Build Failures:**
```bash
# Check Docker daemon is running
docker info

# Clear build cache
docker builder prune

# Build with verbose output
docker build --progress=plain .
```

## Security Considerations

1. **Certificate Storage**: Never commit signing certificates to version control
2. **Environment Variables**: Use secure secret management for sensitive variables
3. **Access Control**: Limit access to signing certificates and deployment credentials
4. **Verification**: Always verify signatures on distributed binaries
5. **Audit Trail**: Maintain logs of all build and deployment activities

## Support

For issues with build and deployment scripts:

1. Check the troubleshooting section above
2. Review the main project documentation
3. Check GitHub Issues for known problems
4. Contact the development team

## Contributing

When modifying build scripts:

1. Test on all target platforms
2. Update documentation
3. Maintain backward compatibility
4. Follow security best practices
5. Add appropriate error handling