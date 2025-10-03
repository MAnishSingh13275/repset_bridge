# Build and Release Script for Gym Door Bridge
# This script builds the executable and creates a GitHub release

param(
    [string]$Version = "v1.0.0",
    [string]$ReleaseNotes = "Initial release of Gym Door Bridge",
    [switch]$Draft = $false,
    [switch]$Prerelease = $false
)

Write-Host "🚀 Building and Releasing Gym Door Bridge $Version" -ForegroundColor Cyan
Write-Host "=================================================" -ForegroundColor Cyan

try {
    # Check if GitHub CLI is installed
    $ghVersion = gh --version 2>$null
    if (-not $ghVersion) {
        Write-Host "❌ GitHub CLI (gh) is not installed!" -ForegroundColor Red
        Write-Host "Please install it from: https://cli.github.com/" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "✅ GitHub CLI found: $($ghVersion[0])" -ForegroundColor Green

    # Check if we're in a git repository
    $gitStatus = git status 2>$null
    if (-not $gitStatus) {
        Write-Host "❌ Not in a git repository!" -ForegroundColor Red
        exit 1
    }
    Write-Host "✅ Git repository detected" -ForegroundColor Green

    # Clean previous builds
    Write-Host "🧹 Cleaning previous builds..." -ForegroundColor Yellow
    if (Test-Path "build") {
        Remove-Item -Recurse -Force "build"
    }
    New-Item -ItemType Directory -Path "build" -Force | Out-Null

    # Build for Windows
    Write-Host "🔨 Building Windows executable..." -ForegroundColor Green
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    
    go build -ldflags "-s -w -X main.version=$Version" -o "build/gym-door-bridge.exe" ./cmd
    
    if (-not (Test-Path "build/gym-door-bridge.exe")) {
        throw "Windows build failed - executable not found"
    }
    
    $exeSize = (Get-Item "build/gym-door-bridge.exe").Length / 1MB
    Write-Host "✅ Windows build completed: $([math]::Round($exeSize, 2)) MB" -ForegroundColor Green

    # Create Windows zip package
    Write-Host "📦 Creating Windows package..." -ForegroundColor Green
    Copy-Item "README.md" -Destination "build/"
    Copy-Item "LICENSE" -Destination "build/" -ErrorAction SilentlyContinue
    
    # Create a simple config template
    @"
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
"@ | Out-File -FilePath "build/config.yaml.template" -Encoding UTF8

    # Create installation instructions
    @"
# Gym Door Bridge Installation

## Quick Install (Recommended)
Run PowerShell as Administrator and execute:
```powershell
iex (iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1).Content
```

## Manual Installation
1. Extract all files to a folder (e.g., C:\GymDoorBridge)
2. Run PowerShell as Administrator
3. Navigate to the extracted folder
4. Run: `.\gym-door-bridge.exe install`
5. Pair with your platform: `.\gym-door-bridge.exe pair YOUR_PAIR_CODE`

## Service Management
- Start: `net start GymDoorBridge`
- Stop: `net stop GymDoorBridge`
- Status: `sc query GymDoorBridge`

## API Access
- Local API: http://localhost:8081
- Health Check: http://localhost:8081/api/v1/health

For support, visit: https://github.com/MAnish13275/repset_bridge
"@ | Out-File -FilePath "build/INSTALL.md" -Encoding UTF8

    # Create zip package
    Compress-Archive -Path "build/*" -DestinationPath "build/gym-door-bridge-windows.zip" -Force
    $zipSize = (Get-Item "build/gym-door-bridge-windows.zip").Length / 1MB
    Write-Host "✅ Package created: $([math]::Round($zipSize, 2)) MB" -ForegroundColor Green

    # Build for Linux (optional)
    Write-Host "🔨 Building Linux executable..." -ForegroundColor Green
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -ldflags "-s -w -X main.version=$Version" -o "build/gym-door-bridge-linux" ./cmd
    
    if (Test-Path "build/gym-door-bridge-linux") {
        tar -czf "build/gym-door-bridge-linux.tar.gz" -C "build" "gym-door-bridge-linux" "README.md" "config.yaml.template" "INSTALL.md"
        Write-Host "✅ Linux build completed" -ForegroundColor Green
    }

    # Reset environment
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

    # Create release notes
    $releaseNotesFile = "build/release-notes.md"
    @"
# Gym Door Bridge $Version

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

### Quick Install (Recommended)
```powershell
# Run PowerShell as Administrator
iex (iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1).Content
```

### Install with Pair Code
```powershell
# Replace YOUR_PAIR_CODE with your actual pair code
`$pairCode = "YOUR_PAIR_CODE"
`$script = iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1
Invoke-Expression "& { `$(`$script.Content) } -PairCode '`$pairCode'"
```

## What's Included
- `gym-door-bridge.exe` - Main executable
- `config.yaml.template` - Configuration template
- `README.md` - Documentation
- `INSTALL.md` - Installation instructions

## System Requirements
- Windows 10/11 or Windows Server 2016+
- Administrator privileges for installation
- Network access to biometric devices
- Internet connection for cloud sync

## Support
- Documentation: https://github.com/MAnish13275/repset_bridge
- Issues: https://github.com/MAnish13275/repset_bridge/issues

$ReleaseNotes
"@ | Out-File -FilePath $releaseNotesFile -Encoding UTF8

    # Create the GitHub release
    Write-Host "🚀 Creating GitHub release..." -ForegroundColor Green
    
    $releaseArgs = @(
        "release", "create", $Version,
        "build/gym-door-bridge-windows.zip",
        "--title", "Gym Door Bridge $Version",
        "--notes-file", $releaseNotesFile
    )
    
    if ($Draft) {
        $releaseArgs += "--draft"
    }
    
    if ($Prerelease) {
        $releaseArgs += "--prerelease"
    }
    
    # Add Linux build if it exists
    if (Test-Path "build/gym-door-bridge-linux.tar.gz") {
        $releaseArgs += "build/gym-door-bridge-linux.tar.gz"
    }
    
    & gh @releaseArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ GitHub release created successfully!" -ForegroundColor Green
        Write-Host "🔗 Release URL: https://github.com/MAnish13275/repset_bridge/releases/tag/$Version" -ForegroundColor Cyan
        
        Write-Host "`n📋 Installation Command for Users:" -ForegroundColor Yellow
        Write-Host "iex (iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1).Content" -ForegroundColor White
        
        Write-Host "`n📋 With Pair Code:" -ForegroundColor Yellow
        Write-Host "`$pairCode = `"YOUR_PAIR_CODE`"" -ForegroundColor White
        Write-Host "`$script = iwr -useb https://raw.githubusercontent.com/MAnish13275/repset_bridge/main/scripts/install-bridge.ps1" -ForegroundColor White
        Write-Host "Invoke-Expression `"& { `$(`$script.Content) } -PairCode '`$pairCode'`"" -ForegroundColor White
        
    } else {
        throw "GitHub release creation failed"
    }

} catch {
    Write-Host "❌ Build and release failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} finally {
    # Cleanup
    Write-Host "🧹 Cleaning up..." -ForegroundColor Yellow
}

Write-Host "`n🎉 Build and release completed successfully!" -ForegroundColor Green