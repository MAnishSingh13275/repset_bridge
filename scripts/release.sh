#!/bin/bash
# Build and Release Script for Gym Door Bridge (Linux/macOS)

VERSION=${1:-"v1.0.0"}
RELEASE_NOTES=${2:-"Initial release of Gym Door Bridge"}
DRAFT=${3:-false}
PRERELEASE=${4:-false}

echo "🚀 Building and Releasing Gym Door Bridge $VERSION"
echo "================================================="

# Check if GitHub CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is not installed!"
    echo "Please install it from: https://cli.github.com/"
    exit 1
fi
echo "✅ GitHub CLI found: $(gh --version | head -n1)"

# Check if we're in a git repository
if ! git status &> /dev/null; then
    echo "❌ Not in a git repository!"
    exit 1
fi
echo "✅ Git repository detected"

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf build
mkdir -p build

# Build for Windows
echo "🔨 Building Windows executable..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$VERSION" -o build/gym-door-bridge.exe ./cmd

if [ ! -f "build/gym-door-bridge.exe" ]; then
    echo "❌ Windows build failed - executable not found"
    exit 1
fi

exe_size=$(du -h build/gym-door-bridge.exe | cut -f1)
echo "✅ Windows build completed: $exe_size"

# Build for Linux
echo "🔨 Building Linux executable..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$VERSION" -o build/gym-door-bridge-linux ./cmd

if [ ! -f "build/gym-door-bridge-linux" ]; then
    echo "❌ Linux build failed"
    exit 1
fi

linux_size=$(du -h build/gym-door-bridge-linux | cut -f1)
echo "✅ Linux build completed: $linux_size"

# Copy additional files
echo "📦 Creating packages..."
cp README.md build/ 2>/dev/null || true
cp LICENSE build/ 2>/dev/null || true

# Create config template
cat > build/config.yaml.template << 'EOF'
# Gym Door Bridge Configuration
# This file will be auto-generated during installation

device_id: ""
device_key: ""
server_url: "https://repset.onezy.in"
tier: "normal"
queue_max_size: 10000
heartbeat_interval: 60
unlock_duration: 3000
database_path: "./bridge.db"
log_level: "info"
log_file: ""
enabled_adapters:
  - "simulator"
adapter_configs:
  simulator:
    device_type: "simulator"
    connection: "memory"
    device_config: {}
    sync_interval: 10
updates_enabled: true
api_server:
  enabled: true
  port: 8081
  host: "0.0.0.0"
EOF

# Create installation instructions
cat > build/INSTALL.md << 'EOF'
# Gym Door Bridge Installation

## Windows - Quick Install (Recommended)
Run PowerShell as Administrator and execute:
```powershell
iex (iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1).Content
```

## Windows - Manual Installation
1. Extract all files to a folder (e.g., C:\GymDoorBridge)
2. Run PowerShell as Administrator
3. Navigate to the extracted folder
4. Run: `.\gym-door-bridge.exe install`
5. Pair with your platform: `.\gym-door-bridge.exe pair YOUR_PAIR_CODE`

## Linux Installation
1. Extract the tar.gz file
2. Make executable: `chmod +x gym-door-bridge-linux`
3. Run: `sudo ./gym-door-bridge-linux install`
4. Pair: `./gym-door-bridge-linux pair YOUR_PAIR_CODE`

## Service Management
### Windows
- Start: `net start GymDoorBridge`
- Stop: `net stop GymDoorBridge`
- Status: `sc query GymDoorBridge`

### Linux
- Start: `sudo systemctl start gym-door-bridge`
- Stop: `sudo systemctl stop gym-door-bridge`
- Status: `sudo systemctl status gym-door-bridge`

## API Access
- Local API: http://localhost:8081
- Health Check: http://localhost:8081/api/v1/health

For support, visit: https://github.com/MAnish13275/repset_bridge
EOF

# Create Windows zip package
cd build
zip -r gym-door-bridge-windows.zip gym-door-bridge.exe README.md config.yaml.template INSTALL.md LICENSE 2>/dev/null || \
zip -r gym-door-bridge-windows.zip gym-door-bridge.exe README.md config.yaml.template INSTALL.md
cd ..

zip_size=$(du -h build/gym-door-bridge-windows.zip | cut -f1)
echo "✅ Windows package created: $zip_size"

# Create Linux tar.gz package
tar -czf build/gym-door-bridge-linux.tar.gz -C build gym-door-bridge-linux README.md config.yaml.template INSTALL.md LICENSE 2>/dev/null || \
tar -czf build/gym-door-bridge-linux.tar.gz -C build gym-door-bridge-linux README.md config.yaml.template INSTALL.md

tar_size=$(du -h build/gym-door-bridge-linux.tar.gz | cut -f1)
echo "✅ Linux package created: $tar_size"

# Create release notes
cat > build/release-notes.md << EOF
# Gym Door Bridge $VERSION

## Features
- 🔗 **One-Click Installation**: Install and configure with a single PowerShell command
- 🔄 **Auto-Discovery**: Automatically detects biometric devices on your network
- 🏢 **Multi-Device Support**: Works with ZKTeco, ESSL, Realtime, and other brands
- 🔒 **Secure Pairing**: Connect to your gym management platform with pair codes
- 📡 **Offline Operation**: Queues events when internet is down, syncs when reconnected
- 🖥️ **Windows Service**: Runs automatically on startup, survives restarts
- 🌐 **REST API**: Local API for remote control and monitoring
- 📊 **Health Monitoring**: Real-time system and device health tracking

## Installation

### Windows - Quick Install (Recommended)
\`\`\`powershell
# Run PowerShell as Administrator
iex (iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1).Content
\`\`\`

### Windows - Install with Pair Code
\`\`\`powershell
# Replace YOUR_PAIR_CODE with your actual pair code
\$pairCode = "YOUR_PAIR_CODE"
\$script = iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1
Invoke-Expression "& { \$(\$script.Content) } -PairCode '\$pairCode'"
\`\`\`

### Linux Installation
\`\`\`bash
# Download and extract
wget https://github.com/MAnish13275/repset_bridge/releases/download/$VERSION/gym-door-bridge-linux.tar.gz
tar -xzf gym-door-bridge-linux.tar.gz
chmod +x gym-door-bridge-linux

# Install and pair
sudo ./gym-door-bridge-linux install
./gym-door-bridge-linux pair YOUR_PAIR_CODE
\`\`\`

## What's Included
- Executables for Windows and Linux
- Configuration template
- Installation instructions
- Documentation

## System Requirements
- **Windows**: Windows 10/11 or Windows Server 2016+
- **Linux**: Ubuntu 18.04+, CentOS 7+, or equivalent
- Administrator/root privileges for installation
- Network access to biometric devices
- Internet connection for cloud sync

## Support
- Documentation: https://github.com/MAnish13275/repset_bridge
- Issues: https://github.com/MAnish13275/repset_bridge/issues

$RELEASE_NOTES
EOF

# Create the GitHub release
echo "🚀 Creating GitHub release..."

RELEASE_ARGS=(
    "release" "create" "$VERSION"
    "build/gym-door-bridge-windows.zip"
    "build/gym-door-bridge-linux.tar.gz"
    "--title" "Gym Door Bridge $VERSION"
    "--notes-file" "build/release-notes.md"
)

if [ "$DRAFT" = "true" ]; then
    RELEASE_ARGS+=("--draft")
fi

if [ "$PRERELEASE" = "true" ]; then
    RELEASE_ARGS+=("--prerelease")
fi

if gh "${RELEASE_ARGS[@]}"; then
    echo "✅ GitHub release created successfully!"
    echo "🔗 Release URL: https://github.com/MAnish13275/repset_bridge/releases/tag/$VERSION"
    
    echo ""
    echo "📋 Installation Command for Users:"
    echo "iex (iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1).Content"
    
    echo ""
    echo "📋 With Pair Code:"
    echo "\$pairCode = \"YOUR_PAIR_CODE\""
    echo "\$script = iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1"
    echo "Invoke-Expression \"& { \$(\$script.Content) } -PairCode '\$pairCode'\""
else
    echo "❌ GitHub release creation failed"
    exit 1
fi

echo ""
echo "🎉 Build and release completed successfully!"